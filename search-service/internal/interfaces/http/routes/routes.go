package routes

import (
	"github.com/gin-gonic/gin"

	"search-service/internal/interfaces/http/handlers"
)

type Handlers struct {
	SearchHandler        *handlers.SearchHandler
	GetRestaurantHandler *handlers.GetRestaurantHandler
}

func SetupRoutes(router *gin.Engine, h *Handlers) {
	router.GET("/search", h.SearchHandler.Search)
	router.GET("/search/restaurant/:id", h.GetRestaurantHandler.Get)
}
