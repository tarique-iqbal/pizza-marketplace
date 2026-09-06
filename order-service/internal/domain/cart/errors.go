package cart

import "errors"

var (
	ErrCartRestaurantMismatch = errors.New("cart already contains items from a different restaurant")
	ErrCartEmpty              = errors.New("cart is empty")
	ErrCartItemUnavailable    = errors.New("one or more cart items are no longer available")
)
