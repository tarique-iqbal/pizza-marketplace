package container

import (
	"os"

	"google.golang.org/grpc"

	"order-service/internal/application/cart/commands"
	"order-service/internal/application/cart/queries"
	ordercmd "order-service/internal/application/order/commands"
	"order-service/internal/infrastructure/geocoder"
	"order-service/internal/infrastructure/payment"
	"order-service/internal/infrastructure/persistence"
	"order-service/internal/interfaces/http/handlers"
	"order-service/internal/interfaces/http/middleware"
)

type APIContainer struct {
	*Shared
	Middleware   *middleware.Middleware
	CartHandler  *handlers.CartHandler
	OrderHandler *handlers.OrderHandler
	paymentConn  *grpc.ClientConn
}

func NewAPIContainer() (*APIContainer, error) {
	base, err := NewShared()
	if err != nil {
		return nil, err
	}

	mw := middleware.NewMiddleware()

	cartRepo := persistence.NewCartRepository(base.DB)
	pizzaRepo := persistence.NewPizzaRepository(base.DB)
	pizzaPriceRepo := persistence.NewPizzaPriceRepository(base.DB)
	toppingPriceRepo := persistence.NewToppingPriceRepository(base.DB)
	orderRepo := persistence.NewOrderRepository(base.DB)
	geocodeRepo := persistence.NewGeocodeRepository(base.DB)
	customerRepo := persistence.NewCustomerRepository(base.DB)
	restaurantRepo := persistence.NewRestaurantRepository(base.DB)

	addItem := commands.NewAddItem(cartRepo, pizzaRepo, pizzaPriceRepo, toppingPriceRepo)
	updateItemQuantity := commands.NewUpdateItemQuantity(cartRepo)
	removeItem := commands.NewRemoveItem(cartRepo)
	getCart := queries.NewGetCart(cartRepo, pizzaRepo, pizzaPriceRepo, toppingPriceRepo)

	cartHandler := handlers.NewCartHandler(addItem, updateItemQuantity, removeItem, getCart)

	cachingGeocoder := geocoder.NewCachingGeocoder(
		geocodeRepo,
		geocoder.NewOpenCageGeocoder(os.Getenv("OPENCAGE_API_KEY")),
	)

	paymentConn, err := payment.Dial(os.Getenv("PAYMENT_SERVICE_ADDR"))
	if err != nil {
		return nil, err
	}
	rawPaymentClient := payment.NewClient(paymentConn)
	paymentProvider := payment.NewCircuitBreakerProvider(rawPaymentClient)

	checkout := ordercmd.NewCheckout(
		base.DB,
		cartRepo,
		orderRepo,
		customerRepo,
		restaurantRepo,
		pizzaRepo,
		pizzaPriceRepo,
		toppingPriceRepo,
		cachingGeocoder,
		paymentProvider,
		os.Getenv("FRONTEND_BASE_URL"),
	)
	orderHandler := handlers.NewOrderHandler(checkout)

	return &APIContainer{
		Shared:       base,
		Middleware:   mw,
		CartHandler:  cartHandler,
		OrderHandler: orderHandler,
		paymentConn:  paymentConn,
	}, nil
}

func (c *APIContainer) Close() {
	if c.DB != nil {
		db, err := c.DB.DB()
		if err == nil {
			_ = db.Close()
		}
	}

	if c.paymentConn != nil {
		_ = c.paymentConn.Close()
	}
}
