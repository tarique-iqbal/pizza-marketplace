package testutil

import (
	"github.com/google/uuid"
)

func MustNewID() uuid.UUID {
	return uuid.Must(uuid.NewV7())
}
