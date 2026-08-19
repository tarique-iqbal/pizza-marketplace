package routes

import (
	"github.com/gin-gonic/gin"

	"search-service/internal/interfaces/http/handlers"
)

type Handlers struct {
	SearchHandler *handlers.SearchHandler
}

func SetupRoutes(router *gin.Engine, h *Handlers) {
	router.GET("/search", h.SearchHandler.Search)
}
