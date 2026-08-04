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
	Middleware      *middleware.Middleware
	Publisher       *messaging.RabbitMQPublisher
	AddressHandler  *handlers.AddressHandler
	DeliveryHandler *handlers.DeliveryHandler
	PayoutHandler   *handlers.PayoutHandler
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

	updateDelivery := commands.NewUpdateDelivery(restaurantRepo)
	deliveryHandler := handlers.NewDeliveryHandler(updateDelivery)

	payoutDetailsRepo := persistence.NewPayoutDetailsRepository(base.DB)
	createPayout := commands.NewCreatePayout(restaurantRepo, payoutDetailsRepo)
	updatePayout := commands.NewUpdatePayout(restaurantRepo, payoutDetailsRepo)
	payoutHandler := handlers.NewPayoutHandler(createPayout, updatePayout)

	return &APIContainer{
		Shared:          base,
		Middleware:      middleware,
		Publisher:       publisher,
		AddressHandler:  addressHandler,
		DeliveryHandler: deliveryHandler,
		PayoutHandler:   payoutHandler,
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
