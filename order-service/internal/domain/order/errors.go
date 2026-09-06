package order

import "errors"

var (
	ErrInvalidStatusTransition = errors.New("invalid order status transition")
	ErrOutsideDeliveryRadius   = errors.New("delivery address is outside restaurant's delivery radius")
	ErrGeocodingUnavailable    = errors.New("geocoding service unavailable")
)
