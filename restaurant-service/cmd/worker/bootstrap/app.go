package bootstrap

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	"restaurant-service/internal/container"
	logobs "restaurant-service/internal/infrastructure/observability/logger"
)

const shutdownTimeout = 10 * time.Second

type App struct {
	logger *slog.Logger
}

func NewApp(logger *slog.Logger) *App {
	return &App{logger: logger}
}

func (a *App) Run() error {
	app, err := container.NewWorkerContainer(a.logger)
	if err != nil {
		return err
	}
	defer app.Close()

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	ctx = logobs.WithContext(ctx, a.logger)

	runner := newRunner(a.logger, app)
	runner.start(ctx, stop)

	<-ctx.Done()
	a.logger.Info("shutdown initiated")

	return a.awaitShutdown(runner)
}

func (a *App) awaitShutdown(r *runner) error {
	select {
	case <-r.done():
		a.logger.Info("shutdown complete")
	case <-time.After(shutdownTimeout):
		a.logger.Warn("forced shutdown after timeout")
	}
	return nil
}
