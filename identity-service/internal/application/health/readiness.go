package health

import "context"

type Readiness struct {
	Postgres Pinger
	Redis    Pinger
	RabbitMQ Pinger
}

func NewReadiness(
	postgres Pinger,
	redis Pinger,
	rabbitMQ Pinger,
) *Readiness {
	return &Readiness{
		Postgres: postgres,
		Redis:    redis,
		RabbitMQ: rabbitMQ,
	}
}

func (r *Readiness) Check(ctx context.Context) ([]Result, bool) {
	checks := []struct {
		name   string
		pinger Pinger
	}{
		{
			name:   "postgres",
			pinger: r.Postgres,
		},
		{
			name:   "redis",
			pinger: r.Redis,
		},
		{
			name:   "rabbitmq",
			pinger: r.RabbitMQ,
		},
	}

	results := make([]Result, 0, len(checks))
	healthy := true

	for _, check := range checks {
		err := check.pinger.Ping(ctx)

		if err != nil {
			healthy = false

			results = append(results, Result{
				Name:   check.name,
				Status: "down",
				Error:  err.Error(),
			})

			continue
		}

		results = append(results, Result{
			Name:   check.name,
			Status: "up",
		})
	}

	return results, healthy
}
