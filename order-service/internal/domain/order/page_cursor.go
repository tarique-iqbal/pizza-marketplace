package order

import (
	"time"

	"github.com/google/uuid"
)

// PageCursor is a keyset pagination seek position: the (placed_at, id) of the
// last row a caller has already seen. nil means "first page".
type PageCursor struct {
	PlacedAt time.Time
	ID       uuid.UUID
}
