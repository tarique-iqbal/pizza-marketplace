package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"restaurant-service/internal/domain/outbox"
)

type WorkerConfig struct {
	PollInterval time.Duration
	BatchSize    int
	Concurrency  int
	MaxRetries   int
	StopTimeout  time.Duration
}

type Worker struct {
	repo    outbox.OutboxRepository
	relayer Relayer
	config  WorkerConfig
	logger  *slog.Logger

	activeWg  sync.WaitGroup
	mu        sync.RWMutex
	isRunning bool
}

func DefaultConfig() WorkerConfig {
	return WorkerConfig{
		PollInterval: 2 * time.Second,
		BatchSize:    50,
		Concurrency:  5,
		MaxRetries:   3,
		StopTimeout:  30 * time.Second,
	}
}

func NewWorker(
	repo outbox.OutboxRepository,
	relayer Relayer,
	config WorkerConfig,
	logger *slog.Logger,
) *Worker {
	return &Worker{
		repo:    repo,
		relayer: relayer,
		config:  config,
		logger:  logger,
	}
}

func (w *Worker) Start(ctx context.Context) {
	w.mu.Lock()
	if w.isRunning {
		w.mu.Unlock()
		w.logger.Warn("worker already running")
		return
	}
	w.isRunning = true
	w.mu.Unlock()

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	w.logger.Info("worker started",
		"poll_interval", w.config.PollInterval,
		"batch_size", w.config.BatchSize,
		"concurrency", w.config.Concurrency,
		"max_retries", w.config.MaxRetries,
	)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("stopping worker...")
			w.shutdown()
			w.logger.Info("worker stopped")
			return

		case <-ticker.C:
			w.activeWg.Add(1)
			go func() {
				defer w.activeWg.Done()
				w.processBatch(ctx)
			}()
		}
	}
}

func (w *Worker) shutdown() {
	done := make(chan struct{})
	go func() {
		w.activeWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		w.logger.Info("all in-flight events completed")
	case <-time.After(w.config.StopTimeout):
		w.logger.Warn("shutdown timeout, some events may be lost",
			"timeout", w.config.StopTimeout,
		)
	}

	w.mu.Lock()
	w.isRunning = false
	w.mu.Unlock()
}

func (w *Worker) processBatch(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	events, err := w.repo.FetchAndMarkProcessing(ctx, w.config.BatchSize)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		w.logger.ErrorContext(ctx, "failed to fetch events", "error", err)
		return
	}

	if len(events) == 0 {
		return
	}

	w.logger.Debug("processing batch", "count", len(events))

	sem := make(chan struct{}, w.config.Concurrency)
	var wg sync.WaitGroup

	for i := range events {
		ev := events[i]

		if ctx.Err() != nil {
			w.releaseUnprocessed(ctx, events[i:])
			break
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(e outbox.OutboxEvent) {
			defer wg.Done()
			defer func() { <-sem }()

			if ctx.Err() != nil {
				return
			}

			w.handleEvent(ctx, e)
		}(ev)
	}

	wg.Wait()
}

func (w *Worker) releaseUnprocessed(ctx context.Context, events []outbox.OutboxEvent) {
	releaseCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	for _, ev := range events {
		if err := w.repo.ReleaseForRetry(releaseCtx, ev.ID, "worker shutdown", 0); err != nil {
			w.logger.ErrorContext(releaseCtx, "failed to release event",
				"event_id", ev.ID,
				"error", err,
			)
		}
	}
}

func (w *Worker) handleEvent(ctx context.Context, ev outbox.OutboxEvent) {
	eventCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	logger := w.logger.With(
		"event_id", ev.ID,
		"event_name", ev.EventName,
		"attempts", ev.Attempts,
		"max_retries", w.config.MaxRetries,
	)

	err := w.relayer.Process(eventCtx, ev)
	if err != nil {
		logger.Error("event processing failed", "error", err)

		// attempts already incremented
		if ev.Attempts >= w.config.MaxRetries {
			logger.Warn("max retries reached, marking as failed")

			failErr := fmt.Errorf("max retries (%d) reached: %w", w.config.MaxRetries, err)
			if markErr := w.repo.MarkFailed(eventCtx, ev.ID, failErr.Error()); markErr != nil {
				logger.Error("CRITICAL: failed to mark event as failed", "error", markErr)
			}
			return
		}

		backoff := computeBackoff(ev.Attempts)
		logger.Info("releasing event for retry", "backoff", backoff)

		if releaseErr := w.repo.ReleaseForRetry(
			eventCtx, ev.ID, err.Error(), backoff,
		); releaseErr != nil {
			logger.Error("CRITICAL: failed to release event for retry", "error", releaseErr)
		}
		return
	}

	if err := w.repo.MarkProcessed(eventCtx, ev.ID); err != nil {
		logger.Error("failed to mark event as processed", "error", err)
		return
	}

	logger.Debug("event processed successfully")
}

func computeBackoff(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}

	// exponential backoff: 1s, 2s, 4s, 8s
	backoff := time.Duration(1<<uint(attempts-1)) * time.Second

	if backoff < time.Second {
		backoff = time.Second
	}
	if backoff > 5*time.Minute {
		backoff = 5 * time.Minute
	}

	return backoff
}
