package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"runtime/debug"
	"sync"

	"payment-service/internal/container"
)

type runner struct {
	logger *slog.Logger
	app    *container.WorkerContainer
	wg     sync.WaitGroup
}

func newRunner(logger *slog.Logger, app *container.WorkerContainer) *runner {
	return &runner{logger: logger, app: app}
}

func (r *runner) start(ctx context.Context, stop context.CancelFunc) {
	r.logger.Info("starting outbox worker")

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer r.recoverPanic(stop)

		r.app.OutboxWorker.Start(ctx)
	}()

	r.logger.Info("starting health server", "addr", healthAddr)

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer r.recoverPanic(stop)

		if err := runHealthServer(ctx, r.app.HealthHandler); err != nil &&
			!errors.Is(err, context.Canceled) {
			r.logger.Error("health server stopped unexpectedly", "error", err)
			stop()
		}
	}()
}

func (r *runner) done() <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(ch)
	}()
	return ch
}

func (r *runner) recoverPanic(stop context.CancelFunc) {
	if rec := recover(); rec != nil {
		r.logger.Error(
			"worker panic",
			"panic", rec,
			"stack", string(debug.Stack()),
		)
		stop()
	}
}
