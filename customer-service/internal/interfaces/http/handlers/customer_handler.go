package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	customerapp "customer-service/internal/application/customer"
	"customer-service/internal/application/customer/commands"
	"customer-service/internal/application/customer/queries"
	"customer-service/internal/interfaces/http/middleware"
	"customer-service/internal/interfaces/http/response"
	"customer-service/internal/interfaces/http/validation"
)

type CustomerHandler struct {
	getProfile  *queries.GetProfile
	updatePhone *commands.UpdatePhone
}

func NewCustomerHandler(
	getProfile *queries.GetProfile,
	updatePhone *commands.UpdatePhone,
) *CustomerHandler {
	return &CustomerHandler{
		getProfile:  getProfile,
		updatePhone: updatePhone,
	}
}

func (h *CustomerHandler) GetProfile(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()

	userID := ctx.MustGet(middleware.CtxUserID).(string)

	customerID, err := uuid.Parse(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}

	res, err := h.getProfile.Execute(reqCtx, customerID)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (h *CustomerHandler) UpdatePhone(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()

	var input customerapp.UpdatePhoneRequest
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

	res, err := h.updatePhone.Execute(reqCtx, customerID, input)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, res)
}
