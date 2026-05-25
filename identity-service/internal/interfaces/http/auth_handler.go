package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"identity-service/internal/application/auth"
	"identity-service/internal/interfaces/http/validation"
)

type AuthHandler struct {
	login        *auth.Login
	emailOTP     *auth.RequestEmailOTP
	refreshToken *auth.RefreshToken
	logout       *auth.Logout
}

func NewAuthHandler(
	login *auth.Login,
	emailOTP *auth.RequestEmailOTP,
	refreshToken *auth.RefreshToken,
	logout *auth.Logout,
) *AuthHandler {
	return &AuthHandler{
		login:        login,
		emailOTP:     emailOTP,
		refreshToken: refreshToken,
		logout:       logout,
	}
}

func (h *AuthHandler) Login(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()
	var input auth.LoginRequest

	if err := ctx.ShouldBindJSON(&input); err != nil {
		errors := validation.ExtractValidationErrors(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"errors": errors})
		return
	}

	response, err := h.login.Execute(reqCtx, input)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (h *AuthHandler) CreateEmailVerification(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()
	var input auth.EmailVerificationRequest

	if err := ctx.ShouldBindJSON(&input); err != nil {
		errors := validation.ExtractValidationErrors(err)
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"errors": errors})
		return
	}

	err := h.emailOTP.Execute(reqCtx, input)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "could not create verification",
		})
		return
	}

	ctx.Status(http.StatusNoContent)
}

func (h *AuthHandler) Refresh(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()
	var input auth.RefreshRequest

	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	response, err := h.refreshToken.Execute(reqCtx, input)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (h *AuthHandler) Logout(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()
	var input auth.LogoutRequest

	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if _, exists := ctx.Get("userID"); !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.logout.Execute(reqCtx, input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "logged out successfully",
	})
}

func (h *AuthHandler) Verify(ctx *gin.Context) {
	ctx.Status(http.StatusOK)
}
