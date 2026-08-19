package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"runtime/debug"
	"sync"

	"search-service/internal/container"
	"search-service/internal/infrastructure/messaging"
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
	r.logger.Info("starting consumer")

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer r.recoverPanic(stop)

		if err := messaging.Run(ctx, r.app.Consumer, r.app.Dispatcher); err != nil &&
			!errors.Is(err, context.Canceled) {
			r.logger.Error("consumer stopped unexpectedly", "error", err)
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
