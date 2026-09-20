package container

import (
	addresscmd "customer-service/internal/application/address/commands"
	addressqry "customer-service/internal/application/address/queries"
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
	AddressHandler  *handlers.AddressHandler
}

func NewAPIContainer() (*APIContainer, error) {
	base, err := NewShared()
	if err != nil {
		return nil, err
	}

	mw := middleware.NewMiddleware()

	customerRepo := persistence.NewCustomerRepository(base.DB)
	addressRepo := persistence.NewAddressRepository(base.DB)

	getProfile := queries.NewGetProfile(customerRepo)
	updatePhone := commands.NewUpdatePhone(base.DB, customerRepo, base.OutboxRepo)

	listAddresses := addressqry.NewList(addressRepo)
	deleteAddress := addresscmd.NewDelete(addressRepo)
	setDefaultAddress := addresscmd.NewSetDefault(base.DB, addressRepo)

	customerHandler := handlers.NewCustomerHandler(getProfile, updatePhone)
	addressHandler := handlers.NewAddressHandler(listAddresses, deleteAddress, setDefaultAddress)

	return &APIContainer{
		Shared:          base,
		Middleware:      mw,
		CustomerHandler: customerHandler,
		AddressHandler:  addressHandler,
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
