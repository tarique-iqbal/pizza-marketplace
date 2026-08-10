package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/application/restaurant/commands"
	"restaurant-service/internal/domain/restaurant"
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

	handler := handlers.NewPizzaHandler(
		commands.NewCreatePizza(restaurantRepo, pizzaRepo, toppingRepo),
		commands.NewUpdatePizza(restaurantRepo, pizzaRepo, pizzaPriceRepo, pizzaSizeRepo, toppingRepo),
	)

	return pizzaHandlerSetup{DB: db.DB, Handler: handler}
}

func pizzaRouter(h *handlers.PizzaHandler, ownerID, role string) *gin.Engine {
	router := gin.Default()
	router.Use(MockAuthMiddleware(ownerID, role), middleware.RequireRole("owner"))

	router.POST("/restaurants/:id/pizzas", h.CreatePizza)
	router.PUT("/restaurants/:id/pizzas/:pizzaId", h.UpdatePizza)

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

	var response resapp.PizzaResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, "Diavola", response.Name)
	assert.Equal(t, restaurant.PizzaAvailable, response.Status)
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

func TestPizzaHandler_UpdatePizza_Success(t *testing.T) {
	h := setupPizzaHandler(t)

	var pizza restaurant.Pizza
	require.NoError(t, h.DB.Order("sort_order").First(&pizza).Error)

	var res restaurant.Restaurant
	require.NoError(t, h.DB.Take(&res, "id = ?", pizza.RestaurantID).Error)

	router := pizzaRouter(h.Handler, res.OwnerID.String(), "owner")

	body, _ := json.Marshal(map[string]any{"name": "Renamed Pizza"})
	req, _ := http.NewRequest(
		http.MethodPut,
		"/restaurants/"+res.ID.String()+"/pizzas/"+pizza.ID.String(),
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response resapp.PizzaResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, "Renamed Pizza", response.Name)
}

func TestPizzaHandler_UpdatePizza_SetsToppings_NoPriceRequired(t *testing.T) {
	h := setupPizzaHandler(t)

	var pizza restaurant.Pizza
	require.NoError(t, h.DB.Order("sort_order").First(&pizza).Error)

	var res restaurant.Restaurant
	require.NoError(t, h.DB.Take(&res, "id = ?", pizza.RestaurantID).Error)

	var topping restaurant.Topping
	require.NoError(t, h.DB.Order("name").Take(&topping).Error)

	router := pizzaRouter(h.Handler, res.OwnerID.String(), "owner")

	body, _ := json.Marshal(map[string]any{
		"name":       pizza.Name,
		"toppingIds": []string{topping.ID.String()},
	})
	req, _ := http.NewRequest(
		http.MethodPut,
		"/restaurants/"+res.ID.String()+"/pizzas/"+pizza.ID.String(),
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response resapp.PizzaResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response.Toppings, 1)
	assert.Equal(t, topping.ID, response.Toppings[0].ToppingID)
}

func TestPizzaHandler_Unauthorized(t *testing.T) {
	h := setupPizzaHandler(t)

	var res restaurant.Restaurant
	require.NoError(t, h.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	router := gin.Default()
	router.Use(middleware.AuthMiddleware())
	router.Use(middleware.RequireRole("owner"))
	router.POST("/restaurants/:id/pizzas", h.Handler.CreatePizza)

	body, _ := json.Marshal(map[string]any{"name": "Diavola"})
	req, _ := http.NewRequest(http.MethodPost, "/restaurants/"+res.ID.String()+"/pizzas", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestPizzaHandler_Forbidden_WrongRole(t *testing.T) {
	h := setupPizzaHandler(t)

	var res restaurant.Restaurant
	require.NoError(t, h.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	router := pizzaRouter(h.Handler, res.OwnerID.String(), "customer")

	body, _ := json.Marshal(map[string]any{"name": "Diavola"})
	req, _ := http.NewRequest(http.MethodPost, "/restaurants/"+res.ID.String()+"/pizzas", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
}
