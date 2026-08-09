package container

import (
	"os"

	"restaurant-service/internal/application/restaurant/commands"
	"restaurant-service/internal/infrastructure/geocoder"
	"restaurant-service/internal/infrastructure/messaging"
	"restaurant-service/internal/infrastructure/persistence"
	"restaurant-service/internal/interfaces/http/handlers"
	"restaurant-service/internal/interfaces/http/middleware"
)

type APIContainer struct {
	*Shared
	Middleware          *middleware.Middleware
	Publisher           *messaging.RabbitMQPublisher
	AddressHandler      *handlers.AddressHandler
	ContactHandler      *handlers.ContactHandler
	DeliveryHandler     *handlers.DeliveryHandler
	PayoutHandler       *handlers.PayoutHandler
	OpeningHoursHandler *handlers.OpeningHoursHandler
	ToppingPriceHandler *handlers.ToppingPriceHandler
	PizzaHandler        *handlers.PizzaHandler
}

func NewAPIContainer() (*APIContainer, error) {
	base, err := NewShared()
	if err != nil {
		return nil, err
	}

	opencageApiKey := os.Getenv("OPENCAGE_API_KEY")

	middleware := middleware.NewMiddleware()
	publisher, err := messaging.NewRabbitMQPublisher(base.AMQPURL)
	if err != nil {
		return nil, err
	}

	restaurantRepo := persistence.NewRestaurantRepository(base.DB)

	geocoder := geocoder.NewOpenCageGeocoder(opencageApiKey)
	updateAddress := commands.NewUpdateAddress(geocoder, restaurantRepo)
	addressHandler := handlers.NewAddressHandler(updateAddress)

	updateContact := commands.NewUpdateContact(restaurantRepo)
	contactHandler := handlers.NewContactHandler(updateContact)

	updateDelivery := commands.NewUpdateDelivery(restaurantRepo)
	deliveryHandler := handlers.NewDeliveryHandler(updateDelivery)

	payoutDetailsRepo := persistence.NewPayoutDetailsRepository(base.DB)
	createPayout := commands.NewCreatePayout(restaurantRepo, payoutDetailsRepo)
	updatePayout := commands.NewUpdatePayout(restaurantRepo, payoutDetailsRepo)
	payoutHandler := handlers.NewPayoutHandler(createPayout, updatePayout)

	updateOpeningHours := commands.NewUpdateOpeningHours(restaurantRepo)
	openingHoursHandler := handlers.NewOpeningHoursHandler(updateOpeningHours)

	toppingRepo := persistence.NewToppingRepository(base.DB)
	toppingPriceRepo := persistence.NewToppingPriceRepository(base.DB)

	setToppingPrices := commands.NewSetToppingPrices(restaurantRepo, toppingRepo, toppingPriceRepo)
	toppingPriceHandler := handlers.NewToppingPriceHandler(setToppingPrices)

	pizzaRepo := persistence.NewPizzaRepository(base.DB)

	createPizza := commands.NewCreatePizza(restaurantRepo, pizzaRepo, toppingRepo)
	pizzaHandler := handlers.NewPizzaHandler(createPizza)

	return &APIContainer{
		Shared:              base,
		Middleware:          middleware,
		Publisher:           publisher,
		AddressHandler:      addressHandler,
		ContactHandler:      contactHandler,
		DeliveryHandler:     deliveryHandler,
		PayoutHandler:       payoutHandler,
		OpeningHoursHandler: openingHoursHandler,
		ToppingPriceHandler: toppingPriceHandler,
		PizzaHandler:        pizzaHandler,
	}, nil
}

func (c *APIContainer) Close() {
	if c.DB != nil {
		db, err := c.DB.DB()
		if err == nil {
			_ = db.Close()
		}
	}

	if c.Publisher != nil {
		c.Publisher.Close()
	}
}
