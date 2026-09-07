package container

import (
	"order-service/internal/application/cart/commands"
	"order-service/internal/application/cart/queries"
	"order-service/internal/infrastructure/persistence"
	"order-service/internal/interfaces/http/handlers"
	"order-service/internal/interfaces/http/middleware"
)

type APIContainer struct {
	*Shared
	Middleware  *middleware.Middleware
	CartHandler *handlers.CartHandler
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

	addItem := commands.NewAddItem(cartRepo, pizzaRepo, pizzaPriceRepo, toppingPriceRepo)
	updateItemQuantity := commands.NewUpdateItemQuantity(cartRepo)
	removeItem := commands.NewRemoveItem(cartRepo)
	getCart := queries.NewGetCart(cartRepo, pizzaRepo, pizzaPriceRepo, toppingPriceRepo)

	cartHandler := handlers.NewCartHandler(addItem, updateItemQuantity, removeItem, getCart)

	return &APIContainer{
		Shared:      base,
		Middleware:  mw,
		CartHandler: cartHandler,
	}, nil
}

func (c *APIContainer) Close() {
	if c.DB != nil {
		db, err := c.DB.DB()
		if err == nil {
			_ = db.Close()
		}
	}
}
