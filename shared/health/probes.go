package health

import "context"

type Result struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type Checker interface {
	Check(ctx context.Context) ([]Result, bool)
}

type Pinger interface {
	Ping(ctx context.Context) error
}

type Check struct {
	Name   string
	Pinger Pinger
}

type Readiness struct {
	checks []Check
}

func NewReadiness(checks ...Check) *Readiness {
	return &Readiness{checks: checks}
}

func (r *Readiness) Check(ctx context.Context) ([]Result, bool) {
	results := make([]Result, 0, len(r.checks))
	healthy := true

	for _, check := range r.checks {
		err := check.Pinger.Ping(ctx)

		if err != nil {
			healthy = false

			results = append(results, Result{
				Name:   check.Name,
				Status: "down",
				Error:  err.Error(),
			})

			continue
		}

		results = append(results, Result{
			Name:   check.Name,
			Status: "up",
		})
	}

	return results, healthy
}
