package container

import (
	"log/slog"

	"identity-service/internal/application/outbox"
	"identity-service/internal/interfaces/http"

	"pizza-marketplace/shared/health"
)

type WorkerContainer struct {
	*Shared
	Worker        *outbox.Worker
	HealthHandler *http.HealthHandler
}

func NewWorkerContainer(logger *slog.Logger) (*WorkerContainer, error) {
	base, err := NewShared()
	if err != nil {
		return nil, err
	}

	relayer := outbox.NewRelay(base.Publisher)
	worker := outbox.NewWorker(base.OutboxRepo, relayer, outbox.DefaultConfig(), logger)

	readiness := health.NewReadiness(
		health.Check{Name: "postgres", Pinger: base.Postgres},
		health.Check{Name: "rabbitmq", Pinger: base.Publisher},
	)
	healthHandler := http.NewHealthHandler(readiness)

	return &WorkerContainer{
		Shared:        base,
		Worker:        worker,
		HealthHandler: healthHandler,
	}, nil
}
