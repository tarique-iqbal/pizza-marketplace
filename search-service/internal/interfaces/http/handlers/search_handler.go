package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"search-service/internal/application/query"
	"search-service/internal/domain/index"
)

type searchRequest struct {
	House       string `form:"house" binding:"required"`
	Street      string `form:"street" binding:"required"`
	City        string `form:"city" binding:"required"`
	PostalCode  string `form:"postalCode" binding:"required"`
	Q           string `form:"q"`
	Fulfillment string `form:"fulfillment"`
	Tags        string `form:"tags"`
	OpenNow     bool   `form:"openNow"`
	Sort        string `form:"sort"`
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

	var tags []string
	if req.Tags != "" {
		tags = strings.Split(req.Tags, ",")
	}

	results, err := h.searchRestaurants.Execute(ctx.Request.Context(), query.SearchRestaurantsRequest{
		Address:     address,
		Text:        req.Q,
		Fulfillment: req.Fulfillment,
		Tags:        tags,
		OpenNow:     req.OpenNow,
		Sort:        req.Sort,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"results": results})
}
