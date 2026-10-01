package outbox_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	outboxapp "identity-service/internal/application/outbox"
	"identity-service/internal/domain/outbox"
	"identity-service/tests/testutil"
)

// A worker shutdown must not silently strand outbox rows that were claimed as part of
// a batch but never actually got dispatched: processBatch's releaseUnprocessed path
// requeues them (status back to pending, reason "worker shutdown") instead of leaving
// them stuck at 'processing' until their lease expires. This test forces that path
// deterministically rather than relying on real shutdown timing: with Concurrency=1,
// only one event can be in flight at once, so once event 0 is dispatched and held open
// by the test, the dispatch loop is necessarily parked trying to claim the concurrency
// slot for event 1, a stable state that ctx cancellation alone can't unstick. Only
// releasing event 0 afterward lets the loop continue, by which point ctx is already
// cancelled, so event 2's own ctx check is guaranteed to observe that and go through
// releaseUnprocessed instead of ever reaching handleEvent.
func TestWorker_Shutdown_ReleasesUndispatchedBatchEvents(t *testing.T) {
	tdb := newEmptyTestDB(t)

	for range 3 {
		ev := outbox.NewOutboxEvent(testutil.MustNewID(), outbox.EventUserRegistered, []byte(`{}`))
		require.NoError(t, tdb.DB.Create(&ev).Error)
	}

	var calls int32
	started := make(chan struct{})
	release := make(chan struct{})

	relayer := &relayStub{
		fn: func(ctx context.Context, e outbox.OutboxEvent) error {
			if atomic.AddInt32(&calls, 1) == 1 {
				close(started)
				<-release
				return nil
			}
			// Event 1 is dispatched the instant event 0 frees the slot, with ctx
			// already cancelled by then: it goes through handleEvent's own
			// cancelled-context path, not releaseUnprocessed. Only event 2
			// onward demonstrates the claim this test makes.
			<-ctx.Done()
			return ctx.Err()
		},
	}

	config := testConfig()
	config.Concurrency = 1
	config.BatchSize = 10
	config.PollInterval = 20 * time.Millisecond
	config.EventTimeout = 5 * time.Second

	worker := outboxapp.NewWorker(tdb.Repo, relayer, config, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		worker.Start(ctx)
	}()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("first event never started processing")
	}

	// Give the dispatch loop time to reach (and block claiming the
	// concurrency slot for) event 1 before cancelling.
	time.Sleep(100 * time.Millisecond)

	cancel()
	time.Sleep(100 * time.Millisecond)
	close(release)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not shut down")
	}

	require.Eventually(t, func() bool {
		rows := listEvents(t, tdb.DB)
		return len(rows) == 3 &&
			rows[0].Status == outbox.StatusProcessed &&
			rows[2].Status == outbox.StatusPending &&
			rows[2].LastError != nil
	}, 5*time.Second, 100*time.Millisecond, "event 0 processed, event 2 released by releaseUnprocessed")

	rows := listEvents(t, tdb.DB)
	require.Len(t, rows, 3)
	assert.Equal(t, outbox.StatusProcessed, rows[0].Status)
	assert.Equal(t, outbox.StatusPending, rows[2].Status)
	require.NotNil(t, rows[2].LastError)
	assert.Equal(t, "worker shutdown", *rows[2].LastError)
}
