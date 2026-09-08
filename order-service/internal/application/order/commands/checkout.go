package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	orderapp "order-service/internal/application/order"
	"order-service/internal/domain/cart"
	"order-service/internal/domain/order"
	"order-service/internal/domain/readmodel"
	apperr "order-service/internal/shared/errors"
	"order-service/internal/shared/geo"
)

type Checkout struct {
	db               *gorm.DB
	cartRepo         cart.CartRepository
	orderRepo        order.OrderRepository
	customerRepo     readmodel.CustomerRepository
	restaurantRepo   readmodel.RestaurantRepository
	pizzaRepo        readmodel.PizzaRepository
	pizzaPriceRepo   readmodel.PizzaPriceRepository
	toppingPriceRepo readmodel.ToppingPriceRepository
	geocoder         order.Geocoder
	paymentProvider  order.PaymentProvider
	frontendBaseURL  string
}

func NewCheckout(
	db *gorm.DB,
	cartRepo cart.CartRepository,
	orderRepo order.OrderRepository,
	customerRepo readmodel.CustomerRepository,
	restaurantRepo readmodel.RestaurantRepository,
	pizzaRepo readmodel.PizzaRepository,
	pizzaPriceRepo readmodel.PizzaPriceRepository,
	toppingPriceRepo readmodel.ToppingPriceRepository,
	geocoder order.Geocoder,
	paymentProvider order.PaymentProvider,
	frontendBaseURL string,
) *Checkout {
	return &Checkout{
		db:               db,
		cartRepo:         cartRepo,
		orderRepo:        orderRepo,
		customerRepo:     customerRepo,
		restaurantRepo:   restaurantRepo,
		pizzaRepo:        pizzaRepo,
		pizzaPriceRepo:   pizzaPriceRepo,
		toppingPriceRepo: toppingPriceRepo,
		geocoder:         geocoder,
		paymentProvider:  paymentProvider,
		frontendBaseURL:  frontendBaseURL,
	}
}

func (uc *Checkout) Execute(
	ctx context.Context,
	customerID uuid.UUID,
	input orderapp.CheckoutRequest,
) (orderapp.CheckoutResponse, error) {
	c, err := uc.cartRepo.FindByCustomer(ctx, customerID)
	if err != nil {
		return orderapp.CheckoutResponse{}, fmt.Errorf("failed to look up cart: %w", err)
	}
	if c == nil || len(c.Items) == 0 {
		return orderapp.CheckoutResponse{}, cart.ErrCartEmpty
	}

	restaurant, err := uc.restaurantRepo.FindByID(ctx, c.RestaurantID)
	if err != nil {
		return orderapp.CheckoutResponse{}, fmt.Errorf("failed to look up restaurant: %w", err)
	}

	fulfillment := order.Fulfillment(input.Fulfillment)

	if fulfillment == order.FulfillmentPickup && !restaurant.Pickup {
		return orderapp.CheckoutResponse{}, order.ErrFulfillmentNotSupported
	}
	if fulfillment == order.FulfillmentDelivery {
		if restaurant.DeliveryType == readmodel.DeliveryNone {
			return orderapp.CheckoutResponse{}, order.ErrFulfillmentNotSupported
		}
		if input.DeliveryAddress == nil {
			return orderapp.CheckoutResponse{}, fmt.Errorf("delivery address required: %w", apperr.ErrInvalid)
		}
	}

	customer, err := uc.customerRepo.FindByID(ctx, customerID)
	if err != nil {
		return orderapp.CheckoutResponse{}, fmt.Errorf("failed to look up customer: %w", err)
	}

	toppingPrices, err := uc.toppingPriceRepo.ListByRestaurant(ctx, c.RestaurantID)
	if err != nil {
		return orderapp.CheckoutResponse{}, fmt.Errorf("failed to look up topping prices: %w", err)
	}

	toppingByID := make(map[uuid.UUID]readmodel.ToppingPrice, len(toppingPrices))
	for _, tp := range toppingPrices {
		toppingByID[tp.ToppingID] = tp
	}

	items := make([]order.OrderItem, 0, len(c.Items))
	subtotal := decimal.Zero

	for _, ci := range c.Items {
		item, lineTotal, err := uc.resolveOrderItem(ctx, ci, toppingByID)
		if err != nil {
			return orderapp.CheckoutResponse{}, err
		}

		items = append(items, item)
		subtotal = subtotal.Add(lineTotal)
	}

	if subtotal.LessThan(restaurant.MinimumOrder) {
		return orderapp.CheckoutResponse{}, order.ErrBelowMinimumOrder
	}

	var deliveryAddress *order.Address
	var deliveryLat, deliveryLon *float64
	deliveryFee := decimal.Zero

	if fulfillment == order.FulfillmentDelivery {
		deliveryAddress = &order.Address{
			House:      input.DeliveryAddress.House,
			Street:     input.DeliveryAddress.Street,
			PostalCode: input.DeliveryAddress.PostalCode,
			City:       input.DeliveryAddress.City,
		}

		lat, lon, err := uc.geocoder.Geocode(ctx, *deliveryAddress)
		if err != nil {
			return orderapp.CheckoutResponse{}, fmt.Errorf("%w: %s", order.ErrGeocodingUnavailable, err)
		}

		if restaurant.DeliveryKm == nil {
			return orderapp.CheckoutResponse{}, order.ErrOutsideDeliveryRadius
		}

		distanceKm := geo.HaversineKm(lat, lon, restaurant.Lat, restaurant.Lon)
		if distanceKm > float64(*restaurant.DeliveryKm) {
			return orderapp.CheckoutResponse{}, order.ErrOutsideDeliveryRadius
		}

		deliveryLat = &lat
		deliveryLon = &lon
		deliveryFee = restaurant.DeliveryFee
	}

	total := subtotal.Add(deliveryFee)

	orderID, err := uuid.NewV7()
	if err != nil {
		return orderapp.CheckoutResponse{}, fmt.Errorf("failed to generate order id: %w", err)
	}

	newOrder := order.NewOrder(
		orderID, customerID, c.RestaurantID,
		fulfillment, customer.Email, input.ContactPhone,
		deliveryAddress, deliveryLat, deliveryLon,
		items, subtotal, deliveryFee, total,
		restaurant.Currency,
	)

	err = uc.db.Transaction(func(tx *gorm.DB) error {
		if err := uc.orderRepo.WithTx(tx).Create(ctx, newOrder); err != nil {
			return fmt.Errorf("failed to create order: %w", err)
		}

		if err := uc.cartRepo.WithTx(tx).Clear(ctx, c.ID); err != nil {
			return fmt.Errorf("failed to clear cart: %w", err)
		}

		return nil
	})
	if err != nil {
		return orderapp.CheckoutResponse{}, err
	}

	redirectURL := uc.frontendBaseURL + "/orders/" + orderID.String()

	paymentResult, err := uc.paymentProvider.CreatePayment(ctx, order.CreatePaymentRequest{
		OrderID:      orderID,
		RestaurantID: c.RestaurantID,
		CustomerID:   customerID,
		Amount:       total,
		Currency:     restaurant.Currency,
		RedirectURL:  redirectURL,
	})
	if err != nil {
		return orderapp.CheckoutResponse{}, fmt.Errorf("failed to create payment: %w", err)
	}

	newOrder.PaymentID = &paymentResult.PaymentID
	if err := uc.orderRepo.Update(ctx, newOrder); err != nil {
		return orderapp.CheckoutResponse{}, fmt.Errorf("failed to record payment id: %w", err)
	}

	return orderapp.CheckoutResponse{
		OrderID:     orderID,
		CheckoutURL: paymentResult.CheckoutURL,
	}, nil
}

func (uc *Checkout) resolveOrderItem(
	ctx context.Context,
	ci cart.CartItem,
	toppingByID map[uuid.UUID]readmodel.ToppingPrice,
) (order.OrderItem, decimal.Decimal, error) {
	pizza, err := uc.pizzaRepo.FindByID(ctx, ci.PizzaID)
	if err != nil && !errors.Is(err, apperr.ErrNotFound) {
		return order.OrderItem{}, decimal.Zero, fmt.Errorf("failed to look up pizza: %w", err)
	}
	if err != nil {
		return order.OrderItem{}, decimal.Zero, cart.ErrCartItemUnavailable
	}

	prices, err := uc.pizzaPriceRepo.ListByPizza(ctx, ci.PizzaID)
	if err != nil {
		return order.OrderItem{}, decimal.Zero, fmt.Errorf("failed to look up pizza prices: %w", err)
	}

	var unitPrice decimal.Decimal
	var diameter int16
	priceFound := false
	for _, p := range prices {
		if p.SizeID == ci.SizeID && p.IsActive {
			unitPrice = p.Price
			diameter = p.DiameterCm
			priceFound = true
			break
		}
	}
	if !priceFound {
		return order.OrderItem{}, decimal.Zero, cart.ErrCartItemUnavailable
	}

	for _, toppingID := range ci.ExtraToppingIDs {
		tp, ok := toppingByID[toppingID]
		if !ok {
			return order.OrderItem{}, decimal.Zero, cart.ErrCartItemUnavailable
		}
		unitPrice = unitPrice.Add(tp.ExtraPrice)
	}

	itemID, err := uuid.NewV7()
	if err != nil {
		return order.OrderItem{}, decimal.Zero, fmt.Errorf("failed to generate order item id: %w", err)
	}

	lineTotal := unitPrice.Mul(decimal.NewFromInt(int64(ci.Quantity)))

	return order.OrderItem{
		ID:              itemID,
		PizzaID:         ci.PizzaID,
		SizeID:          ci.SizeID,
		PizzaName:       pizza.Name,
		SizeDiameter:    diameter,
		ExtraToppingIDs: ci.ExtraToppingIDs,
		Quantity:        ci.Quantity,
		UnitPrice:       unitPrice,
		TotalPrice:      lineTotal,
	}, lineTotal, nil
}
