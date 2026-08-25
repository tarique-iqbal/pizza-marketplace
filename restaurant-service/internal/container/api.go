package container

import (
	"os"

	payoutcmd "restaurant-service/internal/application/payout/commands"
	pizzacmd "restaurant-service/internal/application/pizza/commands"
	pizzaqry "restaurant-service/internal/application/pizza/queries"
	"restaurant-service/internal/application/restaurant/commands"
	resqry "restaurant-service/internal/application/restaurant/queries"
	toppingcmd "restaurant-service/internal/application/topping/commands"
	"restaurant-service/internal/infrastructure/geocoder"
	"restaurant-service/internal/infrastructure/messaging"
	"restaurant-service/internal/infrastructure/persistence"
	"restaurant-service/internal/interfaces/http/handlers"
	"restaurant-service/internal/interfaces/http/middleware"
)

type APIContainer struct {
	*Shared
	Middleware           *middleware.Middleware
	Publisher            *messaging.RabbitMQPublisher
	GetRestaurantHandler *handlers.GetRestaurantHandler
	AddressHandler       *handlers.AddressHandler
	ContactHandler       *handlers.ContactHandler
	DeliveryHandler      *handlers.DeliveryHandler
	PayoutHandler        *handlers.PayoutHandler
	OpeningHoursHandler  *handlers.OpeningHoursHandler
	ToppingPriceHandler  *handlers.ToppingPriceHandler
	PizzaHandler         *handlers.PizzaHandler
	ApproveHandler       *handlers.ApproveHandler
	LaunchHandler        *handlers.LaunchHandler
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
	payoutDetailsRepo := persistence.NewPayoutDetailsRepository(base.DB)

	geocoder := geocoder.NewOpenCageGeocoder(opencageApiKey)
	updateAddress := commands.NewUpdateAddress(geocoder, restaurantRepo, payoutDetailsRepo, publisher)
	addressHandler := handlers.NewAddressHandler(updateAddress)

	updateContact := commands.NewUpdateContact(restaurantRepo, payoutDetailsRepo, publisher)
	contactHandler := handlers.NewContactHandler(updateContact)

	updateDelivery := commands.NewUpdateDelivery(restaurantRepo, payoutDetailsRepo, publisher)
	deliveryHandler := handlers.NewDeliveryHandler(updateDelivery)

	createPayout := payoutcmd.NewCreatePayout(restaurantRepo, payoutDetailsRepo, publisher)
	updatePayout := payoutcmd.NewUpdatePayout(restaurantRepo, payoutDetailsRepo)
	payoutHandler := handlers.NewPayoutHandler(createPayout, updatePayout)

	updateOpeningHours := commands.NewUpdateOpeningHours(restaurantRepo, payoutDetailsRepo, publisher)
	openingHoursHandler := handlers.NewOpeningHoursHandler(updateOpeningHours)

	toppingRepo := persistence.NewToppingRepository(base.DB)
	toppingPriceRepo := persistence.NewToppingPriceRepository(base.DB)

	setToppingPrices := toppingcmd.NewSetToppingPrices(restaurantRepo, toppingRepo, toppingPriceRepo)
	toppingPriceHandler := handlers.NewToppingPriceHandler(setToppingPrices)

	pizzaRepo := persistence.NewPizzaRepository(base.DB)
	pizzaPriceRepo := persistence.NewPizzaPriceRepository(base.DB)
	pizzaSizeRepo := persistence.NewPizzaSizeRepository(base.DB)

	createPizza := pizzacmd.NewCreatePizza(restaurantRepo, pizzaRepo, toppingRepo)
	updatePizza := pizzacmd.NewUpdatePizza(
		restaurantRepo, pizzaRepo, pizzaPriceRepo, pizzaSizeRepo, toppingRepo, publisher,
	)
	setPizzaPrices := pizzacmd.NewSetPizzaPrices(
		restaurantRepo, pizzaRepo, pizzaPriceRepo, pizzaSizeRepo, toppingRepo, publisher,
	)
	pizzaCatalog := pizzaqry.NewPizzaCatalog(
		pizzaRepo, pizzaPriceRepo, pizzaSizeRepo, toppingRepo, toppingPriceRepo,
	)
	listPizzas := pizzaqry.NewListPizzas(restaurantRepo, pizzaCatalog)
	pizzaHandler := handlers.NewPizzaHandler(createPizza, updatePizza, setPizzaPrices, listPizzas)

	getRestaurant := resqry.NewGetRestaurant(restaurantRepo, payoutDetailsRepo, pizzaCatalog)
	getRestaurantHandler := handlers.NewGetRestaurantHandler(getRestaurant)

	approveRestaurant := commands.NewApproveRestaurant(restaurantRepo, payoutDetailsRepo, publisher)
	approveHandler := handlers.NewApproveHandler(approveRestaurant)

	launchRestaurant := commands.NewLaunchRestaurant(restaurantRepo, payoutDetailsRepo, pizzaCatalog, publisher)
	getLaunchReadiness := resqry.NewGetLaunchReadiness(restaurantRepo, pizzaCatalog)
	launchHandler := handlers.NewLaunchHandler(launchRestaurant, getLaunchReadiness)

	return &APIContainer{
		Shared:               base,
		Middleware:           middleware,
		Publisher:            publisher,
		GetRestaurantHandler: getRestaurantHandler,
		AddressHandler:       addressHandler,
		ContactHandler:       contactHandler,
		DeliveryHandler:      deliveryHandler,
		PayoutHandler:        payoutHandler,
		OpeningHoursHandler:  openingHoursHandler,
		ToppingPriceHandler:  toppingPriceHandler,
		PizzaHandler:         pizzaHandler,
		ApproveHandler:       approveHandler,
		LaunchHandler:        launchHandler,
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
