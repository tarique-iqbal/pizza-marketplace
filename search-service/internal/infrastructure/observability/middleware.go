package observability

import (
	"fmt"
	"log/slog"
	"time"

	"search-service/internal/infrastructure/observability/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func Middleware(base *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		start := time.Now()

		requestID := newRequestID()

		requestLogger := base.With(
			"request_id", requestID,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
		)

		ctx := logger.WithContext(
			c.Request.Context(),
			requestLogger,
		)

		c.Request = c.Request.WithContext(ctx)

		c.Next()

		requestLogger.Info(
			"http request completed",
			"status", c.Writer.Status(),
			"duration", time.Since(start),
		)
	}
}

func newRequestID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}

	return id.String()
}
