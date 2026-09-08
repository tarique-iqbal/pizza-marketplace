package commands_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	orderapp "order-service/internal/application/order"
	"order-service/internal/application/order/commands"
	"order-service/internal/domain/cart"
	"order-service/internal/domain/order"
	"order-service/internal/domain/readmodel"
	"order-service/internal/infrastructure/persistence"
	apperr "order-service/internal/shared/errors"
	"order-service/tests/testutil"
)

type fakeGeocoder struct {
	lat, lon float64
	err      error
	calls    []order.Address
}

func (f *fakeGeocoder) Geocode(_ context.Context, addr order.Address) (float64, float64, error) {
	f.calls = append(f.calls, addr)
	return f.lat, f.lon, f.err
}

type fakePaymentProvider struct {
	result order.CreatePaymentResult
	err    error
	calls  []order.CreatePaymentRequest
}

func (f *fakePaymentProvider) CreatePayment(
	_ context.Context,
	req order.CreatePaymentRequest,
) (order.CreatePaymentResult, error) {
	f.calls = append(f.calls, req)
	return f.result, f.err
}

func (f *fakePaymentProvider) CancelPayment(_ context.Context, _ string) error {
	return nil
}

type checkoutSeed struct {
	db         *gorm.DB
	restaurant readmodel.Restaurant
	pizza      readmodel.Pizza
	price      readmodel.PizzaPrice
	customer   readmodel.Customer
	cart       *cart.Cart
	geocoder   *fakeGeocoder
	payment    *fakePaymentProvider
	checkout   *commands.Checkout
}

func seedCheckout(t *testing.T, configureRestaurant func(*readmodel.Restaurant)) checkoutSeed {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant, testutil.TableCustomer, testutil.TableCart)

	deliveryKm := int16(10)
	restaurant := readmodel.Restaurant{
		ID:           testutil.MustNewID(),
		OwnerID:      testutil.MustNewID(),
		Name:         "Pizza Paradise",
		OwnerEmail:   "owner@pizzaparadise.de",
		Lat:          53.5511,
		Lon:          9.9937,
		DeliveryKm:   &deliveryKm,
		DeliveryFee:  decimal.NewFromFloat(2.50),
		MinimumOrder: decimal.NewFromFloat(5.00),
		Pickup:       true,
		DeliveryType: readmodel.DeliveryNone,
		Currency:     "EUR",
		UpdatedAt:    time.Now().UTC(),
	}
	if configureRestaurant != nil {
		configureRestaurant(&restaurant)
	}
	require.NoError(t, db.DB.Create(&restaurant).Error)

	pizza := readmodel.Pizza{
		ID: testutil.MustNewID(), RestaurantID: restaurant.ID,
		Name: "Margherita", Status: readmodel.PizzaAvailable, UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, db.DB.Create(&pizza).Error)

	price := readmodel.PizzaPrice{
		PizzaID: pizza.ID, SizeID: testutil.MustNewID(), DiameterCm: 26,
		Price: decimal.NewFromFloat(7.50), IsActive: true, UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, db.DB.Create(&price).Error)

	customer := readmodel.Customer{
		ID: testutil.MustNewID(), Email: "customer@example.com", FirstName: "Nina",
	}
	require.NoError(t, db.DB.Create(&customer).Error)

	cartRepo := persistence.NewCartRepository(db.DB)
	newCart := cart.NewCart(testutil.MustNewID(), customer.ID, restaurant.ID)
	require.NoError(t, cartRepo.Create(context.Background(), newCart))

	item := cart.NewCartItem(testutil.MustNewID(), pizza.ID, price.SizeID, 2, nil)
	require.NoError(t, cartRepo.AddOrMergeItem(context.Background(), newCart.ID, item))

	loadedCart, err := cartRepo.FindByCustomer(context.Background(), customer.ID)
	require.NoError(t, err)

	geocoder := &fakeGeocoder{}
	payment := &fakePaymentProvider{
		result: order.CreatePaymentResult{PaymentID: "pay_1", CheckoutURL: "https://pay.example.com/1"},
	}

	checkout := commands.NewCheckout(
		db.DB,
		cartRepo,
		persistence.NewOrderRepository(db.DB),
		persistence.NewCustomerRepository(db.DB),
		persistence.NewRestaurantRepository(db.DB),
		persistence.NewPizzaRepository(db.DB),
		persistence.NewPizzaPriceRepository(db.DB),
		persistence.NewToppingPriceRepository(db.DB),
		geocoder,
		payment,
		"https://frontend.example.com",
	)

	return checkoutSeed{
		db: db.DB, restaurant: restaurant, pizza: pizza, price: price, customer: customer,
		cart: loadedCart, geocoder: geocoder, payment: payment, checkout: checkout,
	}
}

func TestCheckout_NoCart(t *testing.T) {
	seed := seedCheckout(t, nil)

	_, err := seed.checkout.Execute(context.Background(), testutil.MustNewID(), orderapp.CheckoutRequest{
		Fulfillment: "pickup",
	})

	assert.ErrorIs(t, err, cart.ErrCartEmpty)
}

func TestCheckout_PickupNotSupported(t *testing.T) {
	seed := seedCheckout(t, func(r *readmodel.Restaurant) { r.Pickup = false })

	_, err := seed.checkout.Execute(context.Background(), seed.customer.ID, orderapp.CheckoutRequest{
		Fulfillment: "pickup",
	})

	assert.ErrorIs(t, err, order.ErrFulfillmentNotSupported)
}

func TestCheckout_DeliveryNotSupported(t *testing.T) {
	seed := seedCheckout(t, nil)

	_, err := seed.checkout.Execute(context.Background(), seed.customer.ID, orderapp.CheckoutRequest{
		Fulfillment: "delivery",
		DeliveryAddress: &orderapp.AddressInput{
			House: "1", Street: "Main St", City: "Hamburg", PostalCode: "12345",
		},
	})

	assert.ErrorIs(t, err, order.ErrFulfillmentNotSupported)
}

func TestCheckout_CartItemUnavailable_PizzaArchived(t *testing.T) {
	seed := seedCheckout(t, nil)
	require.NoError(t, seed.db.Delete(&seed.pizza).Error)

	_, err := seed.checkout.Execute(context.Background(), seed.customer.ID, orderapp.CheckoutRequest{
		Fulfillment: "pickup",
	})

	assert.ErrorIs(t, err, cart.ErrCartItemUnavailable)
}

func TestCheckout_CartItemUnavailable_SizeInactive(t *testing.T) {
	seed := seedCheckout(t, nil)
	require.NoError(t, seed.db.Model(&seed.price).Update("is_active", false).Error)

	_, err := seed.checkout.Execute(context.Background(), seed.customer.ID, orderapp.CheckoutRequest{
		Fulfillment: "pickup",
	})

	assert.ErrorIs(t, err, cart.ErrCartItemUnavailable)
}

func TestCheckout_BelowMinimumOrder(t *testing.T) {
	seed := seedCheckout(t, func(r *readmodel.Restaurant) { r.MinimumOrder = decimal.NewFromFloat(100.00) })

	_, err := seed.checkout.Execute(context.Background(), seed.customer.ID, orderapp.CheckoutRequest{
		Fulfillment: "pickup",
	})

	assert.ErrorIs(t, err, order.ErrBelowMinimumOrder)
}

func TestCheckout_PickupHappyPath(t *testing.T) {
	seed := seedCheckout(t, nil)

	res, err := seed.checkout.Execute(context.Background(), seed.customer.ID, orderapp.CheckoutRequest{
		Fulfillment: "pickup",
	})

	require.NoError(t, err)
	assert.Equal(t, "https://pay.example.com/1", res.CheckoutURL)

	orderRepo := persistence.NewOrderRepository(seed.db)
	found, err := orderRepo.FindByID(context.Background(), res.OrderID)
	require.NoError(t, err)
	assert.Equal(t, order.StatusPending, found.Status)
	assert.True(t, found.Subtotal.Equal(decimal.NewFromFloat(15.00)))
	assert.True(t, found.Total.Equal(decimal.NewFromFloat(15.00)))
	require.NotNil(t, found.PaymentID)
	assert.Equal(t, "pay_1", *found.PaymentID)
	require.Len(t, found.Items, 1)
	assert.Equal(t, "Margherita", found.Items[0].PizzaName)

	cartRepo := persistence.NewCartRepository(seed.db)
	remaining, err := cartRepo.FindByCustomer(context.Background(), seed.customer.ID)
	require.NoError(t, err)
	assert.Nil(t, remaining, "checkout must clear the cart")

	require.Len(t, seed.payment.calls, 1)
	assert.True(t, seed.payment.calls[0].Amount.Equal(decimal.NewFromFloat(15.00)))
}

func TestCheckout_DeliveryHappyPath(t *testing.T) {
	seed := seedCheckout(t, func(r *readmodel.Restaurant) { r.DeliveryType = readmodel.DeliveryOwn })
	seed.geocoder.lat = 53.5600
	seed.geocoder.lon = 9.9900

	res, err := seed.checkout.Execute(context.Background(), seed.customer.ID, orderapp.CheckoutRequest{
		Fulfillment: "delivery",
		DeliveryAddress: &orderapp.AddressInput{
			House: "1", Street: "Main St", City: "Hamburg", PostalCode: "12345",
		},
	})

	require.NoError(t, err)

	orderRepo := persistence.NewOrderRepository(seed.db)
	found, err := orderRepo.FindByID(context.Background(), res.OrderID)
	require.NoError(t, err)
	assert.True(t, found.DeliveryFee.Equal(decimal.NewFromFloat(2.50)))
	assert.True(t, found.Total.Equal(decimal.NewFromFloat(17.50)))
	require.NotNil(t, found.DeliveryAddress)
	assert.Equal(t, "Main St", found.DeliveryAddress.Street)
}

func TestCheckout_OutsideDeliveryRadius(t *testing.T) {
	seed := seedCheckout(t, func(r *readmodel.Restaurant) {
		r.DeliveryType = readmodel.DeliveryOwn
		km := int16(1)
		r.DeliveryKm = &km
	})
	seed.geocoder.lat = 52.5200
	seed.geocoder.lon = 13.4050

	_, err := seed.checkout.Execute(context.Background(), seed.customer.ID, orderapp.CheckoutRequest{
		Fulfillment: "delivery",
		DeliveryAddress: &orderapp.AddressInput{
			House: "1", Street: "Main St", City: "Berlin", PostalCode: "10117",
		},
	})

	assert.ErrorIs(t, err, order.ErrOutsideDeliveryRadius)
}

func TestCheckout_GeocodingUnavailable(t *testing.T) {
	seed := seedCheckout(t, func(r *readmodel.Restaurant) { r.DeliveryType = readmodel.DeliveryOwn })
	seed.geocoder.err = apperr.ErrInvalid

	_, err := seed.checkout.Execute(context.Background(), seed.customer.ID, orderapp.CheckoutRequest{
		Fulfillment: "delivery",
		DeliveryAddress: &orderapp.AddressInput{
			House: "1", Street: "Main St", City: "Hamburg", PostalCode: "12345",
		},
	})

	assert.ErrorIs(t, err, order.ErrGeocodingUnavailable)
}
