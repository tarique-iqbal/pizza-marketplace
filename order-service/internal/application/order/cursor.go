package order

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"order-service/internal/domain/order"
	apperr "order-service/internal/shared/errors"
)

const (
	DefaultOrdersLimit = 20
	MaxOrdersLimit     = 100
)

func EncodeCursor(c order.PageCursor) string {
	raw := c.PlacedAt.Format(time.RFC3339Nano) + "|" + c.ID.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor returns (nil, nil) for an empty string — the first-page case.
func DecodeCursor(s string) (*order.PageCursor, error) {
	if s == "" {
		return nil, nil
	}

	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("malformed cursor: %w", apperr.ErrInvalid)
	}

	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("malformed cursor: %w", apperr.ErrInvalid)
	}

	placedAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, fmt.Errorf("malformed cursor: %w", apperr.ErrInvalid)
	}

	id, err := uuid.Parse(parts[1])
	if err != nil {
		return nil, fmt.Errorf("malformed cursor: %w", apperr.ErrInvalid)
	}

	return &order.PageCursor{PlacedAt: placedAt, ID: id}, nil
}

func ClampLimit(limit int) int {
	if limit <= 0 {
		return DefaultOrdersLimit
	}
	if limit > MaxOrdersLimit {
		return MaxOrdersLimit
	}

	return limit
}
