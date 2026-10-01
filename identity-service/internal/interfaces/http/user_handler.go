package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"identity-service/internal/application/user"
	"identity-service/internal/application/user/commands"
	"identity-service/internal/application/user/queries"
	"identity-service/internal/interfaces/http/response"
	"identity-service/internal/interfaces/http/validation"
)

type UserHandler struct {
	regCustomer *commands.RegisterCustomer
	regOwner    *commands.RegisterOwner
	findByID    *queries.FindByID
}

func NewUserHandler(
	regCustomer *commands.RegisterCustomer,
	regOwner *commands.RegisterOwner,
	findByID *queries.FindByID,
) *UserHandler {
	return &UserHandler{
		regCustomer: regCustomer,
		regOwner:    regOwner,
		findByID:    findByID,
	}
}

func (h *UserHandler) RegisterOwner(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()

	var input user.RegisterOwnerRequest
	if err := ctx.ShouldBindJSON(&input); err != nil {
		errors := validation.ExtractValidationErrors(err)
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"errors": errors})
		return
	}

	res, err := h.regOwner.Execute(reqCtx, input)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, res)
}

func (h *UserHandler) RegisterCustomer(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()

	var input user.RegisterCustomerRequest
	if err := ctx.ShouldBindJSON(&input); err != nil {
		errors := validation.ExtractValidationErrors(err)
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"errors": errors})
		return
	}

	res, err := h.regCustomer.Execute(reqCtx, input)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, res)
}

func (h *UserHandler) FindByID(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()
	idParam := ctx.Param("id")

	userID, err := uuid.Parse(idParam)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	res, err := h.findByID.Execute(reqCtx, userID)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, res)
}
