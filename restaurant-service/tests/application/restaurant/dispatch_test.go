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
			restaurant.ChecklistPayment:      true,
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
	assert.False(t, payload.ReadyAt.IsZero())
}

func TestDispatchEvents_DrainsAggregateEvents(t *testing.T) {
	res := restaurantReadyForReview()
	publisher := &fakePublisher{}

	resapp.DispatchEvents(context.Background(), publisher, res)
	assert.Empty(t, res.PullEvents())
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
