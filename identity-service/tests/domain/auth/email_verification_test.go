package auth

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"identity-service/internal/domain/auth"
)

func TestEmailVerification_MarkCreated_AppendsEvent(t *testing.T) {
	ev := auth.EmailVerification{
		ID:    uuid.New(),
		Email: "ada@example.com",
		Code:  "123456",
	}

	ev.MarkCreated()

	events := ev.PullEvents()
	require.Len(t, events, 1)

	event, ok := events[0].(auth.EmailVerificationCreated)
	require.True(t, ok)

	assert.Equal(t, ev.Email, event.Email)
	assert.Equal(t, ev.Code, event.Code)
	assert.Equal(t, "email.verification_created", event.GetEventName())
	assert.False(t, event.OccurredAt.IsZero())
}

func TestEmailVerification_PullEvents_DrainsQueue(t *testing.T) {
	ev := auth.EmailVerification{ID: uuid.New(), Email: "ada@example.com", Code: "123456"}

	ev.MarkCreated()

	require.Len(t, ev.PullEvents(), 1)
	assert.Empty(t, ev.PullEvents())
}
