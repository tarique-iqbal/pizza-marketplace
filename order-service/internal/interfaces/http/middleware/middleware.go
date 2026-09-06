package middleware

import (
	"github.com/gin-gonic/gin"
)

type Middleware struct {
	Auth           gin.HandlerFunc
	EnsureCustomer gin.HandlerFunc
	EnsureOwner    gin.HandlerFunc
}

func NewMiddleware() *Middleware {
	return &Middleware{
		Auth:           AuthMiddleware(),
		EnsureCustomer: RequireRole("customer"),
		EnsureOwner:    RequireRole("owner"),
	}
}
