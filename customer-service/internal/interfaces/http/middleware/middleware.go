package middleware

import (
	"github.com/gin-gonic/gin"
)

type Middleware struct {
	Auth gin.HandlerFunc
}

func NewMiddleware() *Middleware {
	return &Middleware{
		Auth: AuthMiddleware(),
	}
}
