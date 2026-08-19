package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"search-service/internal/application/query"
	"search-service/internal/domain/index"
)

type searchRequest struct {
	House      string `form:"house" binding:"required"`
	Street     string `form:"street" binding:"required"`
	City       string `form:"city" binding:"required"`
	PostalCode string `form:"postalCode" binding:"required"`
	Q          string `form:"q"`
}

type SearchHandler struct {
	searchRestaurants *query.SearchRestaurants
}

func NewSearchHandler(searchRestaurants *query.SearchRestaurants) *SearchHandler {
	return &SearchHandler{searchRestaurants: searchRestaurants}
}

func (h *SearchHandler) Search(ctx *gin.Context) {
	var req searchRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	address := index.Address{
		House:      req.House,
		Street:     req.Street,
		City:       req.City,
		PostalCode: req.PostalCode,
	}

	results, err := h.searchRestaurants.Execute(ctx.Request.Context(), address, req.Q)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"results": results})
}
