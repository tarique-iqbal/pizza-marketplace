package bootstrap

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"order-service/internal/interfaces/http/handlers"
	"order-service/internal/interfaces/http/routes"
)

const healthAddr = ":8081"

const healthShutdownTimeout = 5 * time.Second

func runHealthServer(ctx context.Context, handler *handlers.HealthHandler) error {
	router := gin.New()
	routes.SetupHealthRoutes(router, handler)

	server := &http.Server{Addr: healthAddr, Handler: router}

	errCh := make(chan error, 1)

	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), healthShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}

		return ctx.Err()
	}
}
