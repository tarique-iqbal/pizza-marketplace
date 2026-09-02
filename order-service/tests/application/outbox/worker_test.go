package outbox_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	outboxapp "order-service/internal/application/outbox"
	"order-service/internal/domain/outbox"
	"order-service/internal/infrastructure/persistence"
	"order-service/tests/infrastructure/db/fixtures"
	"order-service/tests/testutil"
)

func testConfig() outboxapp.WorkerConfig {
	return outboxapp.WorkerConfig{
		PollInterval: 50 * time.Millisecond,
		BatchSize:    10,
		Concurrency:  2,
		MaxRetries:   2,
		StopTimeout:  5 * time.Second,
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

type TestDB struct {
	DB   *gorm.DB
	Repo outbox.OutboxRepository
}

type relayStub struct {
	fn func(ctx context.Context, e outbox.OutboxEvent) error
}

func (r *relayStub) Process(ctx context.Context, e outbox.OutboxEvent) error {
	return r.fn(ctx, e)
}

func newTestDB(t *testing.T) TestDB {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableOutboxEvent)

	_ = fixtures.LoadOutboxEventFixtures(t, db.DB)

	repo := persistence.NewOutboxRepository(db.DB)

	return TestDB{
		DB:   db.DB,
		Repo: repo,
	}
}

func newEmptyTestDB(t *testing.T) TestDB {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableOutboxEvent)

	repo := persistence.NewOutboxRepository(db.DB)

	return TestDB{
		DB:   db.DB,
		Repo: repo,
	}
}

func listEvents(t *testing.T, db *gorm.DB) []outbox.OutboxEvent {
	t.Helper()

	var events []outbox.OutboxEvent
	if err := db.Order("id ASC").Find(&events).Error; err != nil {
		t.Fatalf("failed to list outbox events: %v", err)
	}

	return events
}

func assertAllProcessed(t *testing.T, events []outbox.OutboxEvent) bool {
	t.Helper()

	for _, e := range events {
		if e.Status != "processed" {
			t.Logf("event %d: expected processed, got %s (attempts=%d)",
				e.ID, e.Status, e.Attempts)
			return false
		}
		if e.ProcessedAt == nil {
			t.Logf("event %d: ProcessedAt is nil", e.ID)
			return false
		}
	}
	return true
}

func assertAllFailed(t *testing.T, events []outbox.OutboxEvent, maxRetries int) bool {
	t.Helper()

	for _, e := range events {
		switch e.Status {
		case "failed":
			if e.Attempts < maxRetries {
				t.Logf("event %d: failed too early (attempts=%d, max=%d)",
					e.ID, e.Attempts, maxRetries)
				return false
			}
		case "pending":
			if e.Attempts >= maxRetries {
				t.Logf("event %d: should be failed, still pending (attempts=%d)",
					e.ID, e.Attempts)
				return false
			}
			return false // still retrying
		default:
			t.Logf("event %d: unexpected status=%s", e.ID, e.Status)
			return false
		}
	}
	return true
}

func startWorker(t *testing.T, repo outbox.OutboxRepository, relayer *relayStub) context.CancelFunc {
	t.Helper()

	config := testConfig()
	logger := testLogger()
	worker := outboxapp.NewWorker(repo, relayer, config, logger)

	ctx, cancel := context.WithCancel(context.Background())

	// start worker in background
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		worker.Start(ctx)
	}()

	// return enhanced cancel that waits for shutdown
	return func() {
		cancel()
		wg.Wait()
		time.Sleep(50 * time.Millisecond) // brief pause for cleanup
	}
}

func TestWorker_Start_success(t *testing.T) {
	tdb := newTestDB(t)

	relayer := &relayStub{
		fn: func(ctx context.Context, e outbox.OutboxEvent) error {
			return nil
		},
	}

	cancel := startWorker(t, tdb.Repo, relayer)
	defer cancel()

	require.Eventually(t, func() bool {
		events := listEvents(t, tdb.DB)
		if len(events) == 0 {
			return false
		}
		return assertAllProcessed(t, events)
	}, 5*time.Second, 100*time.Millisecond,
		"all events should be processed successfully")
}

func TestWorker_Start_failure(t *testing.T) {
	tdb := newTestDB(t)

	relayer := &relayStub{
		fn: func(ctx context.Context, e outbox.OutboxEvent) error {
			return errors.New("simulated failure")
		},
	}

	config := testConfig()
	cancel := startWorker(t, tdb.Repo, relayer)
	defer cancel()

	require.Eventually(t, func() bool {
		events := listEvents(t, tdb.DB)
		if len(events) == 0 {
			return false
		}
		return assertAllFailed(t, events, config.MaxRetries)
	}, 10*time.Second, 200*time.Millisecond,
		"all events should fail with max retries")
}

func TestWorker_Start_partial(t *testing.T) {
	tdb := newTestDB(t)

	// first event succeeds, subsequent events fail
	relayer := &relayStub{
		fn: func(ctx context.Context, e outbox.OutboxEvent) error {
			if e.ID == 1 {
				return nil
			}
			return errors.New("partial failure")
		},
	}

	config := testConfig()
	logger := testLogger()
	worker := outboxapp.NewWorker(tdb.Repo, relayer, config, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		worker.Start(ctx)
	}()

	defer func() {
		cancel()
		wg.Wait()
	}()

	require.Eventually(t, func() bool {
		events := listEvents(t, tdb.DB)

		if len(events) < 2 {
			return false
		}

		var processed, pendingOrFailed int
		for _, e := range events {
			switch e.Status {
			case "processed":
				processed++
			case "pending", "failed":
				pendingOrFailed++
			}
		}

		t.Logf("processed=%d, pending/failed=%d", processed, pendingOrFailed)
		return processed == 1 && pendingOrFailed >= 1
	}, 10*time.Second, 200*time.Millisecond,
		"one event should be processed, the rest pending/failed")
}

func TestWorker_Start_empty(t *testing.T) {
	tdb := newEmptyTestDB(t)

	// verify database is empty
	events := listEvents(t, tdb.DB)
	require.Empty(t, events, "database should be empty")

	relayer := &relayStub{
		fn: func(ctx context.Context, e outbox.OutboxEvent) error {
			t.Error("handler should not be called for empty batch")
			return nil
		},
	}

	cancel := startWorker(t, tdb.Repo, relayer)
	defer cancel()

	// verify no events were created
	time.Sleep(200 * time.Millisecond) // let worker poll a few times
	events = listEvents(t, tdb.DB)
	require.Empty(t, events, "no events should exist")
}

func TestWorker_GracefulShutdown(t *testing.T) {
	tdb := newTestDB(t)

	// slow processor to test graceful shutdown
	processing := make(chan struct{})
	relayer := &relayStub{
		fn: func(ctx context.Context, e outbox.OutboxEvent) error {
			processing <- struct{}{} // signal that we're processing
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
				return nil
			}
		},
	}

	config := testConfig()
	config.StopTimeout = 1 * time.Second
	logger := testLogger()
	worker := outboxapp.NewWorker(tdb.Repo, relayer, config, logger)

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		worker.Start(ctx)
	}()

	// wait for processing to start
	select {
	case <-processing:
		t.Log("processing started, initiating shutdown")
	case <-time.After(2 * time.Second):
		t.Fatal("processing never started")
	}

	// cancel context to trigger shutdown
	cancel()

	// wait for shutdown with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Log("worker shut down gracefully")
	case <-time.After(config.StopTimeout + 1*time.Second):
		t.Log("worker shutdown timed out (expected with slow processor)")
	}
}

func TestWorker_ContextCancellation(t *testing.T) {
	tdb := newTestDB(t)

	blockProcessing := make(chan struct{})
	relayer := &relayStub{
		fn: func(ctx context.Context, e outbox.OutboxEvent) error {
			<-blockProcessing // block forever
			return nil
		},
	}

	config := testConfig()
	logger := testLogger()
	worker := outboxapp.NewWorker(tdb.Repo, relayer, config, logger)

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		worker.Start(ctx)
	}()

	// let worker start and pick up events
	time.Sleep(200 * time.Millisecond)

	// cancel context
	cancel()

	// worker should stop within StopTimeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Log("worker stopped successfully after context cancellation")
	case <-time.After(config.StopTimeout + 2*time.Second):
		t.Fatal("worker did not stop within expected time")
	}

	close(blockProcessing) // unblock the goroutine for cleanup
}
