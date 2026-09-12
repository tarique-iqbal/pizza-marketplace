package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	orderapp "order-service/internal/application/order"
	"order-service/internal/application/order/commands"
	"order-service/internal/application/order/queries"
	"order-service/internal/interfaces/http/middleware"
	"order-service/internal/interfaces/http/response"
	"order-service/internal/interfaces/http/validation"
)

type OrderHandler struct {
	checkout             *commands.Checkout
	getOrder             *queries.GetOrder
	listMyOrders         *queries.ListMyOrders
	listRestaurantOrders *queries.ListRestaurantOrders
	markReady            *commands.MarkReady
	complete             *commands.Complete
}

func NewOrderHandler(
	checkout *commands.Checkout,
	getOrder *queries.GetOrder,
	listMyOrders *queries.ListMyOrders,
	listRestaurantOrders *queries.ListRestaurantOrders,
	markReady *commands.MarkReady,
	complete *commands.Complete,
) *OrderHandler {
	return &OrderHandler{
		checkout:             checkout,
		getOrder:             getOrder,
		listMyOrders:         listMyOrders,
		listRestaurantOrders: listRestaurantOrders,
		markReady:            markReady,
		complete:             complete,
	}
}

func (h *OrderHandler) Checkout(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()

	var input orderapp.CheckoutRequest
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

	res, err := h.checkout.Execute(reqCtx, customerID, input)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, res)
}

func (h *OrderHandler) GetOrder(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()

	orderID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	userID, err := uuid.Parse(ctx.MustGet(middleware.CtxUserID).(string))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	role := ctx.MustGet(middleware.CtxUserRole).(string)

	res, err := h.getOrder.Execute(reqCtx, orderID, userID, role)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (h *OrderHandler) ListMyOrders(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()

	customerID, err := uuid.Parse(ctx.MustGet(middleware.CtxUserID).(string))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}

	limit, _ := strconv.Atoi(ctx.Query("limit"))

	res, err := h.listMyOrders.Execute(reqCtx, customerID, ctx.Query("cursor"), limit)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (h *OrderHandler) ListRestaurantOrders(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()

	restaurantID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid restaurant id"})
		return
	}

	ownerID, err := uuid.Parse(ctx.MustGet(middleware.CtxUserID).(string))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid owner id"})
		return
	}

	limit, _ := strconv.Atoi(ctx.Query("limit"))

	res, err := h.listRestaurantOrders.Execute(reqCtx, restaurantID, ownerID, ctx.Query("cursor"), limit)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (h *OrderHandler) MarkReady(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()

	orderID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	ownerID, err := uuid.Parse(ctx.MustGet(middleware.CtxUserID).(string))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid owner id"})
		return
	}

	res, err := h.markReady.Execute(reqCtx, orderID, ownerID)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (h *OrderHandler) Complete(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()

	orderID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	ownerID, err := uuid.Parse(ctx.MustGet(middleware.CtxUserID).(string))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid owner id"})
		return
	}

	res, err := h.complete.Execute(reqCtx, orderID, ownerID)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, res)
}
