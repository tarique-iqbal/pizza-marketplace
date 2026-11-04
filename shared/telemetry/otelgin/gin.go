package otelgin

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func Middleware(serviceName string) gin.HandlerFunc {
	return otelgin.Middleware(serviceName)
}
