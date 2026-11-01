package telemetry

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// GinMiddleware starts a server span per request using the global
// TracerProvider set by Init.
func GinMiddleware(serviceName string) gin.HandlerFunc {
	return otelgin.Middleware(serviceName)
}
