package user

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"identity-service/internal/domain/user"
)

func TestUser_MarkRegistered_AppendsEvent(t *testing.T) {
	u := user.User{
		ID:        uuid.New(),
		FirstName: "Ada",
		Email:     "ada@example.com",
		Role:      user.RoleCustomer,
	}

	u.MarkRegistered()

	events := u.PullEvents()
	require.Len(t, events, 1)

	event, ok := events[0].(user.UserRegistered)
	require.True(t, ok)

	assert.Equal(t, u.ID, event.UserID)
	assert.Equal(t, u.Email, event.Email)
	assert.Equal(t, u.FirstName, event.FirstName)
	assert.Equal(t, u.Role, event.Role)
	assert.Equal(t, "user.registered", event.GetEventName())
	assert.False(t, event.OccurredAt.IsZero())
}

func TestUser_MarkRestaurantInitiated_AppendsEvent(t *testing.T) {
	u := user.User{ID: uuid.New()}
	restaurantID := uuid.New()

	u.MarkRestaurantInitiated(restaurantID, "Pizza Paradise", "DE123456789")

	events := u.PullEvents()
	require.Len(t, events, 1)

	event, ok := events[0].(user.RestaurantInitiated)
	require.True(t, ok)

	assert.Equal(t, restaurantID, event.RestaurantID)
	assert.Equal(t, u.ID, event.OwnerID)
	assert.Equal(t, "Pizza Paradise", event.BusinessName)
	assert.Equal(t, "DE123456789", event.VATNumber)
	assert.Equal(t, "restaurant.initiated", event.GetEventName())
	assert.False(t, event.OccurredAt.IsZero())
}

func TestUser_PullEvents_DrainsQueue(t *testing.T) {
	u := user.User{ID: uuid.New()}

	u.MarkRegistered()

	require.Len(t, u.PullEvents(), 1)
	assert.Empty(t, u.PullEvents())
}

func TestUser_PullEvents_PreservesOrder(t *testing.T) {
	u := user.User{ID: uuid.New()}
	restaurantID := uuid.New()

	u.MarkRestaurantInitiated(restaurantID, "Pizza Paradise", "DE123456789")
	u.MarkRegistered()

	events := u.PullEvents()
	require.Len(t, events, 2)

	_, isRestaurantInitiated := events[0].(user.RestaurantInitiated)
	assert.True(t, isRestaurantInitiated)

	_, isUserRegistered := events[1].(user.UserRegistered)
	assert.True(t, isUserRegistered)
}
