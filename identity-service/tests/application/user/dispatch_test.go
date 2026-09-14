package user_test

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

	userapp "identity-service/internal/application/user"
	"identity-service/internal/domain/outbox"
	"identity-service/internal/domain/user"
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

func (f *fakeOutboxRepo) FetchAndMarkProcessing(ctx context.Context, limit int) ([]outbox.OutboxEvent, error) {
	return nil, nil
}

func (f *fakeOutboxRepo) MarkProcessed(ctx context.Context, id int64) error {
	return nil
}

func (f *fakeOutboxRepo) ReleaseForRetry(ctx context.Context, id int64, errMsg string, delay time.Duration) error {
	return nil
}

func (f *fakeOutboxRepo) MarkFailed(ctx context.Context, id int64, errMsg string) error {
	return nil
}

func TestDispatchEventsTx_RoutesUserRegisteredToOutbox(t *testing.T) {
	u := &user.User{ID: uuid.New(), Email: "ada@example.com", FirstName: "Ada", Role: user.RoleCustomer}
	u.MarkRegistered()

	outboxRepo := &fakeOutboxRepo{}

	err := userapp.DispatchEventsTx(context.Background(), outboxRepo, u)

	require.NoError(t, err)
	require.Len(t, outboxRepo.events, 1)

	stored := outboxRepo.events[0]
	assert.Equal(t, u.ID, stored.AggregateID)
	assert.Equal(t, "user.registered", stored.EventName)
	assert.Equal(t, outbox.StatusPending, stored.Status)
}

func TestDispatchEventsTx_RoutesRestaurantInitiatedUnderRestaurantID(t *testing.T) {
	u := &user.User{ID: uuid.New(), Email: "owner@example.com", FirstName: "Owner", Role: user.RoleOwner}
	restaurantID := uuid.New()
	u.MarkRestaurantInitiated(restaurantID, "Pizza Paradise", "DE123456789")

	outboxRepo := &fakeOutboxRepo{}

	err := userapp.DispatchEventsTx(context.Background(), outboxRepo, u)

	require.NoError(t, err)
	require.Len(t, outboxRepo.events, 1)

	stored := outboxRepo.events[0]
	assert.Equal(t, restaurantID, stored.AggregateID, "restaurant.initiated must use restaurantID, not the owner's userID")
	assert.NotEqual(t, u.ID, stored.AggregateID)
	assert.Equal(t, "restaurant.initiated", stored.EventName)

	var payload userapp.RestaurantInitiatedPayload
	require.NoError(t, json.Unmarshal(stored.Payload, &payload))
	assert.Equal(t, restaurantID, payload.RestaurantID)
	assert.Equal(t, u.ID, payload.OwnerID)
	assert.Equal(t, "Pizza Paradise", payload.BusinessName)
	assert.Equal(t, "DE123456789", payload.VATNumber)
	assert.Equal(t, "restaurant.initiated", payload.EventName)
	assert.False(t, payload.OccurredAt.IsZero())
}

func TestDispatchEventsTx_DispatchesBothEventsInOrder(t *testing.T) {
	u := &user.User{ID: uuid.New(), Email: "owner@example.com", FirstName: "Owner", Role: user.RoleOwner}
	restaurantID := uuid.New()
	u.MarkRegistered()
	u.MarkRestaurantInitiated(restaurantID, "Pizza Paradise", "DE123456789")

	outboxRepo := &fakeOutboxRepo{}

	err := userapp.DispatchEventsTx(context.Background(), outboxRepo, u)

	require.NoError(t, err)
	require.Len(t, outboxRepo.events, 2)
	assert.Equal(t, "user.registered", outboxRepo.events[0].EventName)
	assert.Equal(t, "restaurant.initiated", outboxRepo.events[1].EventName)
}

func TestDispatchEventsTx_NoOpWhenNoEvents(t *testing.T) {
	u := &user.User{ID: uuid.New()}
	outboxRepo := &fakeOutboxRepo{}

	err := userapp.DispatchEventsTx(context.Background(), outboxRepo, u)

	require.NoError(t, err)
	assert.Empty(t, outboxRepo.events)
}

func TestDispatchEventsTx_DrainsAggregateEvents(t *testing.T) {
	u := &user.User{ID: uuid.New()}
	u.MarkRegistered()

	outboxRepo := &fakeOutboxRepo{}

	err := userapp.DispatchEventsTx(context.Background(), outboxRepo, u)

	require.NoError(t, err)
	assert.Empty(t, u.PullEvents())
}

func TestDispatchEventsTx_OutboxCreateError_ReturnsError(t *testing.T) {
	u := &user.User{ID: uuid.New()}
	u.MarkRegistered()

	outboxRepo := &fakeOutboxRepo{err: errors.New("db unavailable")}

	err := userapp.DispatchEventsTx(context.Background(), outboxRepo, u)

	require.Error(t, err)
}
