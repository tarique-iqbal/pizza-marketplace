package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	toppingapp "restaurant-service/internal/application/topping"
	"restaurant-service/internal/application/topping/commands"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/domain/topping"
	"restaurant-service/internal/infrastructure/persistence"
	"restaurant-service/internal/interfaces/http/handlers"
	"restaurant-service/internal/interfaces/http/middleware"
	"restaurant-service/tests/infrastructure/db/fixtures"
	"restaurant-service/tests/testutil"
)

type toppingPriceHandlerSetup struct {
	DB      *gorm.DB
	Handler *handlers.ToppingPriceHandler
}

func setupToppingPriceHandler(t *testing.T) toppingPriceHandlerSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	require.NoError(t, fixtures.LoadRestaurantFixtures(t, db.DB))

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)
	toppingRepo := persistence.NewToppingRepository(db.DB)
	toppingPriceRepo := persistence.NewToppingPriceRepository(db.DB)

	handler := handlers.NewToppingPriceHandler(
		commands.NewSetToppingPrices(restaurantRepo, toppingRepo, toppingPriceRepo, testutil.NoopPublisher{}),
	)

	return toppingPriceHandlerSetup{DB: db.DB, Handler: handler}
}

func toppingPriceRouter(h *handlers.ToppingPriceHandler, ownerID, role string) *gin.Engine {
	router := gin.Default()
	router.Use(MockAuthMiddleware(ownerID, role), middleware.RequireRole("owner"))

	router.PUT("/restaurants/:id/topping-prices", h.SetToppingPrices)

	return router
}

func TestToppingPriceHandler_SetToppingPrices_Success(t *testing.T) {
	h := setupToppingPriceHandler(t)

	var res restaurant.Restaurant
	require.NoError(t, h.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	var t1 topping.Topping
	require.NoError(t, h.DB.Order("name").Take(&t1).Error)

	router := toppingPriceRouter(h.Handler, res.OwnerID.String(), "owner")

	body, _ := json.Marshal(map[string]any{
		"prices": []map[string]any{{"toppingId": t1.ID.String(), "extraPrice": "1.50"}},
	})
	req, _ := http.NewRequest(
		http.MethodPut,
		"/restaurants/"+res.ID.String()+"/topping-prices",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response []toppingapp.ToppingPriceResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response, 1)
	assert.Equal(t, t1.ID, response[0].ToppingID)
	assert.True(t, decimal.RequireFromString("1.50").Equal(decimal.Decimal(response[0].ExtraPrice)))
}

func TestToppingPriceHandler_SetToppingPrices_ValidationError_EmptyPrices(t *testing.T) {
	h := setupToppingPriceHandler(t)

	var res restaurant.Restaurant
	require.NoError(t, h.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	router := toppingPriceRouter(h.Handler, res.OwnerID.String(), "owner")

	req, _ := http.NewRequest(
		http.MethodPut,
		"/restaurants/"+res.ID.String()+"/topping-prices",
		bytes.NewBufferString(`{"prices": []}`),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "errors")
}

func TestToppingPriceHandler_Unauthorized(t *testing.T) {
	h := setupToppingPriceHandler(t)

	var res restaurant.Restaurant
	require.NoError(t, h.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	router := gin.Default()
	router.Use(middleware.AuthMiddleware())
	router.Use(middleware.RequireRole("owner"))
	router.PUT("/restaurants/:id/topping-prices", h.Handler.SetToppingPrices)

	req, _ := http.NewRequest(
		http.MethodPut,
		"/restaurants/"+res.ID.String()+"/topping-prices",
		bytes.NewBufferString(`{"prices": []}`),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}
