package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"customer-service/internal/application/address/commands"
	"customer-service/internal/application/address/queries"
	"customer-service/internal/interfaces/http/middleware"
	"customer-service/internal/interfaces/http/response"
)

type AddressHandler struct {
	list       *queries.List
	deleteAddr *commands.Delete
	setDefault *commands.SetDefault
}

func NewAddressHandler(
	list *queries.List,
	deleteAddr *commands.Delete,
	setDefault *commands.SetDefault,
) *AddressHandler {
	return &AddressHandler{
		list:       list,
		deleteAddr: deleteAddr,
		setDefault: setDefault,
	}
}

func (h *AddressHandler) List(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()

	userID := ctx.MustGet(middleware.CtxUserID).(string)

	customerID, err := uuid.Parse(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}

	res, err := h.list.Execute(reqCtx, customerID)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (h *AddressHandler) Delete(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()

	addressID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid address id"})
		return
	}

	userID := ctx.MustGet(middleware.CtxUserID).(string)

	customerID, err := uuid.Parse(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}

	if err := h.deleteAddr.Execute(reqCtx, customerID, addressID); err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

func (h *AddressHandler) SetDefault(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()

	addressID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid address id"})
		return
	}

	userID := ctx.MustGet(middleware.CtxUserID).(string)

	customerID, err := uuid.Parse(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}

	res, err := h.setDefault.Execute(reqCtx, customerID, addressID)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, res)
}
