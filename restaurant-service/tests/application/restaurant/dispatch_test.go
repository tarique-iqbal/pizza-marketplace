package restaurant_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/domain/outbox"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/shared/event"
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

type stubPayload struct {
	Name string `json:"name"`
}

func (p stubPayload) GetEventName() string {
	return p.Name
}

func enrichAsUpdated(e restaurant.DomainEvent) (event.Event, bool) {
	if _, ok := e.(restaurant.RestaurantUpdated); ok {
		return stubPayload{Name: "restaurant.updated"}, true
	}
	return nil, false
}

func restaurantReadyForReview() *restaurant.Restaurant {
	res := &restaurant.Restaurant{
		ID:     uuid.New(),
		Name:   "Pizza Paradise",
		Status: restaurant.StatusDraft,
		Checklist: restaurant.Checklist{
			restaurant.ChecklistBasic:        true,
			restaurant.ChecklistContact:      true,
			restaurant.ChecklistAddress:      true,
			restaurant.ChecklistDelivery:     true,
			restaurant.ChecklistPayout:       true,
			restaurant.ChecklistOpeningHours: false,
		},
	}
	res.CompleteChecklistItem(restaurant.ChecklistOpeningHours)
	return res
}

func TestDispatchEventsTx_RoutesEnrichedEventToOutbox(t *testing.T) {
	res := &restaurant.Restaurant{ID: uuid.New(), Name: "Pizza Paradise", Status: restaurant.StatusActive}
	res.NotifyUpdated()

	outboxRepo := &fakeOutboxRepo{}

	err := resapp.DispatchEventsTx(context.Background(), outboxRepo, res, enrichAsUpdated)

	require.NoError(t, err)
	require.Len(t, outboxRepo.events, 1)

	stored := outboxRepo.events[0]
	assert.Equal(t, res.ID, stored.AggregateID)
	assert.Equal(t, "restaurant.updated", stored.EventName)
	assert.Equal(t, outbox.StatusPending, stored.Status)
	assert.JSONEq(t, `{"name":"restaurant.updated"}`, string(stored.Payload))
}

func TestDispatchEventsTx_RoutesReadyForReviewEventToOutbox(t *testing.T) {
	res := restaurantReadyForReview()
	outboxRepo := &fakeOutboxRepo{}

	err := resapp.DispatchEventsTx(context.Background(), outboxRepo, res)

	require.NoError(t, err)
	require.Len(t, outboxRepo.events, 1)
	assert.Equal(t, "restaurant.ready_for_review", outboxRepo.events[0].EventName)
}

func TestDispatchEventsTx_RoutesApprovedEventToOutbox(t *testing.T) {
	email := "kontakt@pizzaparadise.de"
	res := &restaurant.Restaurant{ID: uuid.New(), Name: "Pizza Paradise", Email: &email, Status: restaurant.StatusReview}
	require.NoError(t, res.Approve())

	outboxRepo := &fakeOutboxRepo{}

	err := resapp.DispatchEventsTx(context.Background(), outboxRepo, res)

	require.NoError(t, err)
	require.Len(t, outboxRepo.events, 1)
	assert.Equal(t, "restaurant.approved", outboxRepo.events[0].EventName)
}

func TestDispatchEventsTx_DropsUnmappedLaunchedEvent(t *testing.T) {
	// RestaurantLaunched needs the restaurant's priced pizzas, which only
	// LaunchRestaurant's own enricher can fetch - not toEventPayload's fallback.
	res := &restaurant.Restaurant{ID: uuid.New(), Name: "Pizza Paradise", Status: restaurant.StatusApproved}
	require.NoError(t, res.Launch())

	outboxRepo := &fakeOutboxRepo{}

	err := resapp.DispatchEventsTx(context.Background(), outboxRepo, res)

	require.NoError(t, err)
	assert.Empty(t, outboxRepo.events)
}

func TestDispatchEventsTx_NoOpWhenNoEvents(t *testing.T) {
	res := &restaurant.Restaurant{
		ID:        uuid.New(),
		Status:    restaurant.StatusDraft,
		Checklist: restaurant.NewChecklist(),
	}
	outboxRepo := &fakeOutboxRepo{}

	err := resapp.DispatchEventsTx(context.Background(), outboxRepo, res)

	require.NoError(t, err)
	assert.Empty(t, outboxRepo.events)
}

func TestDispatchEventsTx_DrainsAggregateEvents(t *testing.T) {
	res := restaurantReadyForReview()
	outboxRepo := &fakeOutboxRepo{}

	err := resapp.DispatchEventsTx(context.Background(), outboxRepo, res)

	require.NoError(t, err)
	assert.Empty(t, res.PullEvents())
}

func TestDispatchEventsTx_OutboxCreateError_ReturnsError(t *testing.T) {
	res := &restaurant.Restaurant{ID: uuid.New(), Name: "Pizza Paradise", Status: restaurant.StatusActive}
	res.NotifyUpdated()

	outboxRepo := &fakeOutboxRepo{err: errors.New("db unavailable")}

	err := resapp.DispatchEventsTx(context.Background(), outboxRepo, res, enrichAsUpdated)

	require.Error(t, err)
}
