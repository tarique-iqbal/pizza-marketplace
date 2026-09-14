package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	authapp "identity-service/internal/application/auth"
	"identity-service/internal/domain/auth"
	"identity-service/internal/domain/outbox"
)

type fakeOutboxRepo struct {
	events []outbox.OutboxEvent
	err    error
}

func (f *fakeOutboxRepo) WithTx(tx *gorm.DB) outbox.OutboxRepository {
	return f
}

func (f *fakeOutboxRepo) Create(ctx context.Context, e *outbox.OutboxEvent) error {
	if f.err != nil {
		return f.err
	}
	f.events = append(f.events, *e)
	return nil
}

func (f *fakeOutboxRepo) FetchAndMarkProcessing(
	ctx context.Context,
	limit int,
) ([]outbox.OutboxEvent, error) {
	return nil, nil
}

func (f *fakeOutboxRepo) MarkProcessed(ctx context.Context, id int64) error {
	return nil
}

func (f *fakeOutboxRepo) ReleaseForRetry(
	ctx context.Context,
	id int64,
	errMsg string,
	delay time.Duration,
) error {
	return nil
}

func (f *fakeOutboxRepo) MarkFailed(ctx context.Context, id int64, errMsg string) error {
	return nil
}

func TestDispatchEventsTx_RoutesEmailVerificationCreatedToOutbox(t *testing.T) {
	ev := &auth.EmailVerification{ID: uuid.New(), Email: "ada@example.com", Code: "123456"}
	ev.MarkCreated()

	outboxRepo := &fakeOutboxRepo{}

	err := authapp.DispatchEventsTx(context.Background(), outboxRepo, ev)

	require.NoError(t, err)
	require.Len(t, outboxRepo.events, 1)

	stored := outboxRepo.events[0]
	assert.Equal(t, ev.ID, stored.AggregateID)
	assert.Equal(t, "email.verification_created", stored.EventName)
	assert.Equal(t, outbox.StatusPending, stored.Status)

	var payload authapp.EmailVerificationCreatedPayload
	require.NoError(t, json.Unmarshal(stored.Payload, &payload))
	assert.Equal(t, "ada@example.com", payload.Email)
	assert.Equal(t, "123456", payload.Code)
	assert.Equal(t, "email.verification_created", payload.EventName)
	assert.False(t, payload.OccurredAt.IsZero())
}

func TestDispatchEventsTx_NoOpWhenNoEvents(t *testing.T) {
	ev := &auth.EmailVerification{ID: uuid.New()}
	outboxRepo := &fakeOutboxRepo{}

	err := authapp.DispatchEventsTx(context.Background(), outboxRepo, ev)

	require.NoError(t, err)
	assert.Empty(t, outboxRepo.events)
}

func TestDispatchEventsTx_DrainsAggregateEvents(t *testing.T) {
	ev := &auth.EmailVerification{ID: uuid.New(), Email: "ada@example.com", Code: "123456"}
	ev.MarkCreated()

	outboxRepo := &fakeOutboxRepo{}

	err := authapp.DispatchEventsTx(context.Background(), outboxRepo, ev)

	require.NoError(t, err)
	assert.Empty(t, ev.PullEvents())
}

func TestDispatchEventsTx_OutboxCreateError_ReturnsError(t *testing.T) {
	ev := &auth.EmailVerification{ID: uuid.New(), Email: "ada@example.com", Code: "123456"}
	ev.MarkCreated()

	outboxRepo := &fakeOutboxRepo{err: errors.New("db unavailable")}

	err := authapp.DispatchEventsTx(context.Background(), outboxRepo, ev)

	require.Error(t, err)
}
