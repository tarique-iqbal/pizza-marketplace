package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	addresscmd "customer-service/internal/application/address/commands"
	addressqry "customer-service/internal/application/address/queries"
	"customer-service/internal/application/customer/commands"
	"customer-service/internal/application/customer/queries"
	"customer-service/internal/infrastructure/persistence"
	"customer-service/internal/interfaces/http/handlers"
	"customer-service/internal/interfaces/http/middleware"
	"customer-service/internal/interfaces/http/routes"
	"customer-service/tests/testutil"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	exitCode := m.Run()
	os.Exit(exitCode)
}

func newRouter(t *testing.T) *gin.Engine {
	db := testutil.DB(t)

	customerRepo := persistence.NewCustomerRepository(db.DB)
	addressRepo := persistence.NewAddressRepository(db.DB)
	outboxRepo := persistence.NewOutboxRepository(db.DB)

	customerHandler := handlers.NewCustomerHandler(
		queries.NewGetProfile(customerRepo),
		commands.NewUpdatePhone(db.DB, customerRepo, outboxRepo),
	)
	addressHandler := handlers.NewAddressHandler(
		addressqry.NewList(addressRepo),
		addresscmd.NewDelete(addressRepo),
		addresscmd.NewSetDefault(db.DB, addressRepo),
	)

	router := gin.New()
	routes.SetupRoutes(
		router,
		&routes.Handlers{CustomerHandler: customerHandler, AddressHandler: addressHandler},
		middleware.NewMiddleware(),
	)

	return router
}

func doRequest(
	router *gin.Engine,
	method, path string,
	customerID *uuid.UUID,
	body []byte,
) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	if customerID != nil {
		req.Header.Set("X-User-ID", customerID.String())
		req.Header.Set("X-User-Role", "customer")
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	return recorder
}
