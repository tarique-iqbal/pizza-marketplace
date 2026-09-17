package container

import (
	"customer-service/internal/application/customer/commands"
	"customer-service/internal/application/customer/queries"
	"customer-service/internal/infrastructure/persistence"
	"customer-service/internal/interfaces/http/handlers"
	"customer-service/internal/interfaces/http/middleware"
)

type APIContainer struct {
	*Shared
	Middleware      *middleware.Middleware
	CustomerHandler *handlers.CustomerHandler
}

func NewAPIContainer() (*APIContainer, error) {
	base, err := NewShared()
	if err != nil {
		return nil, err
	}

	mw := middleware.NewMiddleware()

	customerRepo := persistence.NewCustomerRepository(base.DB)

	getProfile := queries.NewGetProfile(customerRepo)
	updatePhone := commands.NewUpdatePhone(base.DB, customerRepo, base.OutboxRepo)

	customerHandler := handlers.NewCustomerHandler(getProfile, updatePhone)

	return &APIContainer{
		Shared:          base,
		Middleware:      mw,
		CustomerHandler: customerHandler,
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
