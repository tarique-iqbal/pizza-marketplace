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

type fakePublisher struct {
	events []event.Event
	err    error
}

func (f *fakePublisher) PublishEvent(ctx context.Context, e event.Event) error {
	f.events = append(f.events, e)
	return f.err
}

func (f *fakePublisher) PublishRaw(ctx context.Context, topic string, jsonData []byte) error {
	return f.err
}

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

func TestDispatchEvents_PublishesPendingEvent(t *testing.T) {
	res := restaurantReadyForReview()
	publisher := &fakePublisher{}

	resapp.DispatchEvents(context.Background(), publisher, res)

	require.Len(t, publisher.events, 1)

	payload, ok := publisher.events[0].(resapp.RestaurantReadyForReviewPayload)
	require.True(t, ok)

	assert.Equal(t, res.ID, payload.RestaurantID)
	assert.Equal(t, "Pizza Paradise", payload.RestaurantName)
	assert.Equal(t, "restaurant.ready_for_review", payload.EventName)
	assert.False(t, payload.OccurredAt.IsZero())
}

func TestDispatchEvents_DrainsAggregateEvents(t *testing.T) {
	res := restaurantReadyForReview()
	publisher := &fakePublisher{}

	resapp.DispatchEvents(context.Background(), publisher, res)
	assert.Empty(t, res.PullEvents())
}

func TestDispatchEvents_PublishesApprovedEvent(t *testing.T) {
	email := "kontakt@pizzaparadise.de"
	res := &restaurant.Restaurant{ID: uuid.New(), Name: "Pizza Paradise", Email: &email, Status: restaurant.StatusReview}
	require.NoError(t, res.Approve())

	publisher := &fakePublisher{}

	resapp.DispatchEvents(context.Background(), publisher, res)

	require.Len(t, publisher.events, 1)

	payload, ok := publisher.events[0].(resapp.RestaurantApprovedPayload)
	require.True(t, ok)

	assert.Equal(t, res.ID, payload.RestaurantID)
	assert.Equal(t, "Pizza Paradise", payload.RestaurantName)
	assert.Equal(t, email, payload.Email)
	assert.Equal(t, "restaurant.approved", payload.EventName)
	assert.False(t, payload.OccurredAt.IsZero())
}

func TestDispatchEvents_DropsLaunchedEvent(t *testing.T) {
	// RestaurantLaunched needs the current menu attached, which only the caller
	// (LaunchRestaurant) can fetch — it's published directly via
	// resapp.NewRestaurantLaunchedPayload, not through this generic dispatch path.
	res := &restaurant.Restaurant{ID: uuid.New(), Name: "Pizza Paradise", Status: restaurant.StatusApproved}
	require.NoError(t, res.Launch())

	publisher := &fakePublisher{}

	resapp.DispatchEvents(context.Background(), publisher, res)

	assert.Empty(t, publisher.events)
}

func TestDispatchEvents_NoOpWhenNoEvents(t *testing.T) {
	res := &restaurant.Restaurant{
		ID:        uuid.New(),
		Status:    restaurant.StatusDraft,
		Checklist: restaurant.NewChecklist(),
	}
	publisher := &fakePublisher{}

	resapp.DispatchEvents(context.Background(), publisher, res)

	assert.Empty(t, publisher.events)
}

func TestDispatchEvents_BestEffort_SwallowsPublishError(t *testing.T) {
	res := restaurantReadyForReview()
	publisher := &fakePublisher{err: errors.New("broker unavailable")}

	assert.NotPanics(t, func() {
		resapp.DispatchEvents(context.Background(), publisher, res)
	})

	assert.Len(t, publisher.events, 1)
}

func TestDispatchEventsTx_RoutesOutboxWorthyEventToOutbox(t *testing.T) {
	res := &restaurant.Restaurant{ID: uuid.New(), Name: "Pizza Paradise", Status: restaurant.StatusActive}
	res.NotifyUpdated()

	outboxRepo := &fakeOutboxRepo{}

	bestEffort, err := resapp.DispatchEventsTx(context.Background(), outboxRepo, res, enrichAsUpdated)

	require.NoError(t, err)
	assert.Empty(t, bestEffort)
	require.Len(t, outboxRepo.events, 1)

	stored := outboxRepo.events[0]
	assert.Equal(t, res.ID, stored.AggregateID)
	assert.Equal(t, "restaurant.updated", stored.EventName)
	assert.Equal(t, outbox.StatusPending, stored.Status)
	assert.JSONEq(t, `{"name":"restaurant.updated"}`, string(stored.Payload))
}

func TestDispatchEventsTx_ReturnsNonOutboxEventForBestEffort(t *testing.T) {
	res := restaurantReadyForReview()
	outboxRepo := &fakeOutboxRepo{}

	bestEffort, err := resapp.DispatchEventsTx(context.Background(), outboxRepo, res)

	require.NoError(t, err)
	assert.Empty(t, outboxRepo.events)
	require.Len(t, bestEffort, 1)

	payload, ok := bestEffort[0].(resapp.RestaurantReadyForReviewPayload)
	require.True(t, ok)
	assert.Equal(t, "restaurant.ready_for_review", payload.EventName)
}

func TestDispatchEventsTx_DrainsAggregateEvents(t *testing.T) {
	res := restaurantReadyForReview()
	outboxRepo := &fakeOutboxRepo{}

	_, err := resapp.DispatchEventsTx(context.Background(), outboxRepo, res)

	require.NoError(t, err)
	assert.Empty(t, res.PullEvents())
}

func TestDispatchEventsTx_OutboxCreateError_ReturnsError(t *testing.T) {
	res := &restaurant.Restaurant{ID: uuid.New(), Name: "Pizza Paradise", Status: restaurant.StatusActive}
	res.NotifyUpdated()

	outboxRepo := &fakeOutboxRepo{err: errors.New("db unavailable")}

	_, err := resapp.DispatchEventsTx(context.Background(), outboxRepo, res, enrichAsUpdated)

	require.Error(t, err)
}

func TestPublishBestEffort_PublishesEvents(t *testing.T) {
	publisher := &fakePublisher{}
	events := []event.Event{stubPayload{Name: "restaurant.ready_for_review"}}

	resapp.PublishBestEffort(context.Background(), publisher, events)

	require.Len(t, publisher.events, 1)
	assert.Equal(t, "restaurant.ready_for_review", publisher.events[0].GetEventName())
}

func TestPublishBestEffort_SwallowsPublishError(t *testing.T) {
	publisher := &fakePublisher{err: errors.New("broker unavailable")}
	events := []event.Event{stubPayload{Name: "restaurant.ready_for_review"}}

	assert.NotPanics(t, func() {
		resapp.PublishBestEffort(context.Background(), publisher, events)
	})
}
