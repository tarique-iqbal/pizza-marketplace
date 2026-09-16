package bootstrap

import (
	"context"
	"log/slog"
	"runtime/debug"
	"sync"

	"customer-service/internal/container"
	"customer-service/internal/infrastructure/messaging"
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
	r.logger.Info("starting event consumer")

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer r.recoverPanic(stop)

		_ = messaging.Run(ctx, r.app.Consumer, r.app.Dispatcher)
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
