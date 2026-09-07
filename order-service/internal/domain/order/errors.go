package order

import "errors"

var (
	ErrInvalidStatusTransition = errors.New("invalid order status transition")
	ErrOutsideDeliveryRadius   = errors.New("delivery address is outside restaurant's delivery radius")
	ErrGeocodingUnavailable    = errors.New("geocoding service unavailable")
	ErrFulfillmentNotSupported = errors.New("restaurant does not support the requested fulfillment method")
	ErrBelowMinimumOrder       = errors.New("order subtotal is below the restaurant's minimum order")
)
