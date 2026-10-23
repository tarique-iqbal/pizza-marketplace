package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"identity-service/internal/container"
	logobs "identity-service/internal/infrastructure/observability/logger"
	"identity-service/internal/interfaces/http/routes"
)

const healthAddr = ":8081"

func main() {
	logger := logobs.NewLogger("identity-worker")

	c, err := container.NewWorkerContainer(logger)
	if err != nil {
		logger.Error("failed to initialize worker container", "error", err)
		os.Exit(1)
	}
	defer c.Close()

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	ctx = logobs.WithContext(ctx, logger)

	router := gin.New()
	routes.SetupHealthRoutes(router, c.HealthHandler)
	healthServer := &http.Server{Addr: healthAddr, Handler: router}

	var wg sync.WaitGroup

	logger.Info("starting outbox worker...")

	wg.Add(1)
	go func() {
		defer wg.Done()
		c.Worker.Start(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		logger.Info("starting health server", "addr", healthAddr)

		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("health server exited with error", "error", err)
		}
	}()

	<-ctx.Done()

	logger.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("health server shutdown error", "error", err)
	}

	wg.Wait()

	logger.Info("worker stopped gracefully")
}
