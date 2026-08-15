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

	pizzaapp "restaurant-service/internal/application/pizza"
	"restaurant-service/internal/application/pizza/commands"
	"restaurant-service/internal/application/pizza/queries"
	"restaurant-service/internal/domain/pizza"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/domain/topping"
	"restaurant-service/internal/infrastructure/persistence"
	"restaurant-service/internal/interfaces/http/handlers"
	"restaurant-service/internal/interfaces/http/middleware"
	"restaurant-service/tests/infrastructure/db/fixtures"
	"restaurant-service/tests/testutil"
)

type pizzaHandlerSetup struct {
	DB      *gorm.DB
	Handler *handlers.PizzaHandler
}

func setupPizzaHandler(t *testing.T) pizzaHandlerSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	require.NoError(t, fixtures.LoadRestaurantFixtures(t, db.DB))
	require.NoError(t, fixtures.LoadPizzaFixtures(t, db.DB))

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)
	pizzaRepo := persistence.NewPizzaRepository(db.DB)
	pizzaPriceRepo := persistence.NewPizzaPriceRepository(db.DB)
	pizzaSizeRepo := persistence.NewPizzaSizeRepository(db.DB)
	toppingRepo := persistence.NewToppingRepository(db.DB)
	toppingPriceRepo := persistence.NewToppingPriceRepository(db.DB)

	pizzaCatalog := queries.NewPizzaCatalog(pizzaRepo, pizzaPriceRepo, pizzaSizeRepo, toppingRepo, toppingPriceRepo)

	handler := handlers.NewPizzaHandler(
		commands.NewCreatePizza(restaurantRepo, pizzaRepo, toppingRepo),
		commands.NewUpdatePizza(restaurantRepo, pizzaRepo, pizzaPriceRepo, pizzaSizeRepo, toppingRepo),
		commands.NewSetPizzaPrices(restaurantRepo, pizzaRepo, pizzaPriceRepo, pizzaSizeRepo, toppingRepo),
		queries.NewListPizzas(restaurantRepo, pizzaCatalog),
	)

	return pizzaHandlerSetup{DB: db.DB, Handler: handler}
}

func pizzaRouter(h *handlers.PizzaHandler, ownerID, role string) *gin.Engine {
	router := gin.Default()
	router.Use(MockAuthMiddleware(ownerID, role), middleware.RequireRole("owner"))

	router.GET("/restaurants/:id/pizzas", h.ListPizzas)
	router.POST("/restaurants/:id/pizzas", h.CreatePizza)
	router.PUT("/restaurants/:id/pizzas/:pizzaId", h.UpdatePizza)
	router.PUT("/restaurants/:id/pizzas/:pizzaId/prices", h.SetPizzaPrices)

	return router
}

func TestPizzaHandler_CreatePizza_Success(t *testing.T) {
	h := setupPizzaHandler(t)

	var res restaurant.Restaurant
	require.NoError(t, h.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	router := pizzaRouter(h.Handler, res.OwnerID.String(), "owner")

	body, _ := json.Marshal(map[string]any{"name": "Diavola"})
	req, _ := http.NewRequest(http.MethodPost, "/restaurants/"+res.ID.String()+"/pizzas", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusCreated, recorder.Code)

	var response pizzaapp.PizzaResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, "Diavola", response.Name)
	assert.Equal(t, pizza.PizzaAvailable, response.Status)
}

func TestPizzaHandler_CreatePizza_ChecklistIncomplete(t *testing.T) {
	h := setupPizzaHandler(t)

	var res restaurant.Restaurant
	require.NoError(t, h.DB.Where("name = ?", "Pizza Paradise").Take(&res).Error)

	router := pizzaRouter(h.Handler, res.OwnerID.String(), "owner")

	body, _ := json.Marshal(map[string]any{"name": "Diavola"})
	req, _ := http.NewRequest(http.MethodPost, "/restaurants/"+res.ID.String()+"/pizzas", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "onboarding")
}

func TestPizzaHandler_CreatePizza_ValidationError_MissingName(t *testing.T) {
	h := setupPizzaHandler(t)

	var res restaurant.Restaurant
	require.NoError(t, h.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	router := pizzaRouter(h.Handler, res.OwnerID.String(), "owner")

	req, _ := http.NewRequest(http.MethodPost, "/restaurants/"+res.ID.String()+"/pizzas", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "errors")
}

func TestPizzaHandler_ListPizzas_Success(t *testing.T) {
	h := setupPizzaHandler(t)

	var res restaurant.Restaurant
	require.NoError(t, h.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	router := pizzaRouter(h.Handler, res.OwnerID.String(), "owner")

	req, _ := http.NewRequest(http.MethodGet, "/restaurants/"+res.ID.String()+"/pizzas", nil)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response []pizzaapp.PizzaResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response, 2)
	assert.Equal(t, "Margherita", response[0].Name)
	assert.True(t, response[0].IsVegetarian)
	assert.Equal(t, "Salami", response[1].Name)
	assert.False(t, response[1].IsVegetarian)
}

func TestPizzaHandler_UpdatePizza_Success(t *testing.T) {
	h := setupPizzaHandler(t)

	var p pizza.Pizza
	require.NoError(t, h.DB.Order("sort_order").First(&p).Error)

	var res restaurant.Restaurant
	require.NoError(t, h.DB.Take(&res, "id = ?", p.RestaurantID).Error)

	router := pizzaRouter(h.Handler, res.OwnerID.String(), "owner")

	body, _ := json.Marshal(map[string]any{"name": "Renamed Pizza"})
	req, _ := http.NewRequest(
		http.MethodPut,
		"/restaurants/"+res.ID.String()+"/pizzas/"+p.ID.String(),
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response pizzaapp.PizzaResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, "Renamed Pizza", response.Name)
}

func TestPizzaHandler_SetPizzaPrices_Success(t *testing.T) {
	h := setupPizzaHandler(t)

	var p pizza.Pizza
	require.NoError(t, h.DB.Order("sort_order").First(&p).Error)

	var res restaurant.Restaurant
	require.NoError(t, h.DB.Take(&res, "id = ?", p.RestaurantID).Error)

	var size pizza.PizzaSize
	require.NoError(t, h.DB.Order("diameter_cm").Take(&size).Error)

	router := pizzaRouter(h.Handler, res.OwnerID.String(), "owner")

	body, _ := json.Marshal(map[string]any{
		"prices": []map[string]any{{"sizeId": size.ID.String(), "price": "9.99"}},
	})
	req, _ := http.NewRequest(
		http.MethodPut,
		"/restaurants/"+res.ID.String()+"/pizzas/"+p.ID.String()+"/prices",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response pizzaapp.PizzaResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response.Prices, 1)
	assert.True(t, response.Prices[0].IsActive)
	assert.Equal(t, size.ID, response.Prices[0].SizeID)
	assert.True(t, decimal.RequireFromString("9.99").Equal(decimal.Decimal(response.Prices[0].Price)))
}

func TestPizzaHandler_UpdatePizza_SetsToppings_NoPriceRequired(t *testing.T) {
	h := setupPizzaHandler(t)

	var p pizza.Pizza
	require.NoError(t, h.DB.Order("sort_order").First(&p).Error)

	var res restaurant.Restaurant
	require.NoError(t, h.DB.Take(&res, "id = ?", p.RestaurantID).Error)

	var t1 topping.Topping
	require.NoError(t, h.DB.Order("name").Take(&t1).Error)

	router := pizzaRouter(h.Handler, res.OwnerID.String(), "owner")

	body, _ := json.Marshal(map[string]any{
		"name":       p.Name,
		"toppingIds": []string{t1.ID.String()},
	})
	req, _ := http.NewRequest(
		http.MethodPut,
		"/restaurants/"+res.ID.String()+"/pizzas/"+p.ID.String(),
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response pizzaapp.PizzaResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response.Toppings, 1)
	assert.Equal(t, t1.ID, response.Toppings[0].ToppingID)
}

func TestPizzaHandler_Unauthorized(t *testing.T) {
	h := setupPizzaHandler(t)

	var res restaurant.Restaurant
	require.NoError(t, h.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	router := gin.Default()
	router.Use(middleware.AuthMiddleware())
	router.Use(middleware.RequireRole("owner"))
	router.GET("/restaurants/:id/pizzas", h.Handler.ListPizzas)

	req, _ := http.NewRequest(http.MethodGet, "/restaurants/"+res.ID.String()+"/pizzas", nil)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestPizzaHandler_Forbidden_WrongRole(t *testing.T) {
	h := setupPizzaHandler(t)

	var res restaurant.Restaurant
	require.NoError(t, h.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	router := pizzaRouter(h.Handler, res.OwnerID.String(), "customer")

	req, _ := http.NewRequest(http.MethodGet, "/restaurants/"+res.ID.String()+"/pizzas", nil)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
}
