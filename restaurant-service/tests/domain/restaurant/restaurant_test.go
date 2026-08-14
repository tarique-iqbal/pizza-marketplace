package restaurant

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"restaurant-service/internal/domain/restaurant"
)

func TestRestaurant_CompleteChecklistItem_TransitionsToReview(t *testing.T) {
	res := restaurant.Restaurant{
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

	assert.Equal(t, restaurant.StatusReview, res.Status)

	events := res.PullEvents()
	require.Len(t, events, 1)

	event, ok := events[0].(restaurant.RestaurantReadyForReview)
	require.True(t, ok)

	assert.Equal(t, res.ID, event.RestaurantID)
	assert.Equal(t, "Pizza Paradise", event.RestaurantName)
	assert.Equal(t, "restaurant.ready_for_review", event.GetEventName())
	assert.False(t, event.ReadyAt.IsZero())
}

func TestRestaurant_CompleteChecklistItem_StaysInDraftIfIncomplete(t *testing.T) {
	res := restaurant.Restaurant{
		ID:        uuid.New(),
		Status:    restaurant.StatusDraft,
		Checklist: restaurant.NewChecklist(),
	}

	res.CompleteChecklistItem(restaurant.ChecklistBasic)

	assert.Equal(t, restaurant.StatusDraft, res.Status)
	assert.Empty(t, res.PullEvents())
}

func TestRestaurant_CompleteChecklistItem_NoOpIfAlreadyPastDraft(t *testing.T) {
	res := restaurant.Restaurant{
		ID:     uuid.New(),
		Status: restaurant.StatusReview,
		Checklist: restaurant.Checklist{
			restaurant.ChecklistBasic:        true,
			restaurant.ChecklistContact:      true,
			restaurant.ChecklistAddress:      true,
			restaurant.ChecklistDelivery:     true,
			restaurant.ChecklistPayment:      true,
			restaurant.ChecklistOpeningHours: true,
		},
	}

	res.CompleteChecklistItem(restaurant.ChecklistContact)

	assert.Equal(t, restaurant.StatusReview, res.Status)
	assert.Empty(t, res.PullEvents())
}

func TestRestaurant_Approve_TransitionsToApproved(t *testing.T) {
	res := restaurant.Restaurant{
		ID:     uuid.New(),
		Name:   "Pizza Paradise",
		Status: restaurant.StatusReview,
	}

	err := res.Approve()
	require.NoError(t, err)

	assert.Equal(t, restaurant.StatusApproved, res.Status)

	events := res.PullEvents()
	require.Len(t, events, 1)

	event, ok := events[0].(restaurant.RestaurantApproved)
	require.True(t, ok)

	assert.Equal(t, res.ID, event.RestaurantID)
	assert.Equal(t, "Pizza Paradise", event.RestaurantName)
	assert.Equal(t, "restaurant.approved", event.GetEventName())
	assert.False(t, event.ApprovedAt.IsZero())
}

func TestRestaurant_Approve_FailsIfNotPendingReview(t *testing.T) {
	for _, status := range []restaurant.RestaurantStatus{
		restaurant.StatusDraft,
		restaurant.StatusApproved,
		restaurant.StatusActive,
	} {
		res := restaurant.Restaurant{ID: uuid.New(), Status: status}

		err := res.Approve()

		require.ErrorIs(t, err, restaurant.ErrNotPendingReview)
		assert.Equal(t, status, res.Status)
		assert.Empty(t, res.PullEvents())
	}
}

func TestRestaurant_PullEvents_DrainsQueue(t *testing.T) {
	res := restaurant.Restaurant{
		ID:     uuid.New(),
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

	require.Len(t, res.PullEvents(), 1)
	assert.Empty(t, res.PullEvents())
}
