package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"search-service/internal/container"
	"search-service/internal/interfaces/http/routes"

	"pizza-marketplace/shared/logobs"
)

const (
	serviceName     = "search-api"
	httpAddr        = ":8080"
	shutdownTimeout = 10 * time.Second
)

func main() {
	logger := logobs.NewLogger(serviceName)

	app, err := container.NewAPIContainer()
	if err != nil {
		logger.Error("application exited with error", "error", err)
		return
	}
	defer app.Close()

	router := gin.New()
	router.Use(gin.Recovery(), logobs.Middleware(logger))

	handlers := &routes.Handlers{
		SearchHandler:        app.SearchHandler,
		GetRestaurantHandler: app.GetRestaurantHandler,
		HealthHandler:        app.HealthHandler,
	}

	routes.SetupRoutes(router, handlers)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := serve(ctx, logger, router); err != nil {
		logger.Error("server exited with error", "error", err)
	}
}

func serve(ctx context.Context, logger *slog.Logger, handler http.Handler) error {
	server := &http.Server{Addr: httpAddr, Handler: handler}

	errCh := make(chan error, 1)

	go func() {
		errCh <- server.ListenAndServe()
	}()

	logger.Info("starting HTTP server", "addr", httpAddr)

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	logger.Info("shutdown initiated")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	logger.Info("shutdown complete")

	return nil
}
