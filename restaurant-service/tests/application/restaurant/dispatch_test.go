package restaurant_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	resapp "restaurant-service/internal/application/restaurant"
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
