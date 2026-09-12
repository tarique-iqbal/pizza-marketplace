package order_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	orderapp "order-service/internal/application/order"
	"order-service/internal/domain/order"
	apperr "order-service/internal/shared/errors"
	"order-service/tests/testutil"
)

func TestCursor_EncodeDecode_RoundTrips(t *testing.T) {
	c := order.PageCursor{PlacedAt: time.Now().UTC().Truncate(time.Nanosecond), ID: testutil.MustNewID()}

	encoded := orderapp.EncodeCursor(c)
	decoded, err := orderapp.DecodeCursor(encoded)

	require.NoError(t, err)
	require.NotNil(t, decoded)
	assert.True(t, c.PlacedAt.Equal(decoded.PlacedAt))
	assert.Equal(t, c.ID, decoded.ID)
}

func TestDecodeCursor_Empty_ReturnsNilNoError(t *testing.T) {
	decoded, err := orderapp.DecodeCursor("")

	require.NoError(t, err)
	assert.Nil(t, decoded)
}

func TestDecodeCursor_Malformed_ReturnsInvalid(t *testing.T) {
	_, err := orderapp.DecodeCursor("not-valid-base64!!!")

	require.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrInvalid)
}

func TestClampLimit(t *testing.T) {
	assert.Equal(t, orderapp.DefaultOrdersLimit, orderapp.ClampLimit(0))
	assert.Equal(t, orderapp.DefaultOrdersLimit, orderapp.ClampLimit(-5))
	assert.Equal(t, 10, orderapp.ClampLimit(10))
	assert.Equal(t, orderapp.MaxOrdersLimit, orderapp.ClampLimit(1000))
}
