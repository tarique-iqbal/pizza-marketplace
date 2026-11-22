package container

import (
	"identity-service/internal/domain/outbox"
	"identity-service/internal/infrastructure/db"
	"identity-service/internal/infrastructure/persistence"
)

type Shared struct {
	Postgres   *db.Postgres
	OutboxRepo outbox.OutboxRepository
}

func NewShared() (*Shared, error) {
	postgres, err := db.NewDB()
	if err != nil {
		return nil, err
	}

	outboxRepo := persistence.NewOutboxRepository(postgres.DB)

	return &Shared{
		Postgres:   postgres,
		OutboxRepo: outboxRepo,
	}, nil
}

func (c *Shared) Close() {
	if c.Postgres.DB != nil {
		db, err := c.Postgres.DB.DB()
		if err == nil {
			_ = db.Close()
		}
	}
}
