package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	cartapp "order-service/internal/application/cart"
	"order-service/internal/application/cart/commands"
	"order-service/internal/interfaces/http/middleware"
	"order-service/internal/interfaces/http/response"
	"order-service/internal/interfaces/http/validation"
)

type CartHandler struct {
	addItem            *commands.AddItem
	updateItemQuantity *commands.UpdateItemQuantity
	removeItem         *commands.RemoveItem
}

func NewCartHandler(
	addItem *commands.AddItem,
	updateItemQuantity *commands.UpdateItemQuantity,
	removeItem *commands.RemoveItem,
) *CartHandler {
	return &CartHandler{addItem: addItem, updateItemQuantity: updateItemQuantity, removeItem: removeItem}
}

func (h *CartHandler) AddItem(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()

	var input cartapp.AddItemRequest
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{
			"errors": validation.ExtractValidationErrors(err),
		})
		return
	}

	userID := ctx.MustGet(middleware.CtxUserID).(string)

	customerID, err := uuid.Parse(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}

	res, err := h.addItem.Execute(reqCtx, customerID, input)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, res)
}

func (h *CartHandler) UpdateItemQuantity(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()

	itemID, err := uuid.Parse(ctx.Param("itemId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	var input cartapp.UpdateItemQuantityRequest
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{
			"errors": validation.ExtractValidationErrors(err),
		})
		return
	}

	userID := ctx.MustGet(middleware.CtxUserID).(string)

	customerID, err := uuid.Parse(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}

	res, err := h.updateItemQuantity.Execute(reqCtx, customerID, itemID, input)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (h *CartHandler) RemoveItem(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()

	itemID, err := uuid.Parse(ctx.Param("itemId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	userID := ctx.MustGet(middleware.CtxUserID).(string)

	customerID, err := uuid.Parse(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}

	if err := h.removeItem.Execute(reqCtx, customerID, itemID); err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}
