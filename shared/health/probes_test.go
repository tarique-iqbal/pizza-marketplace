package health_test

import (
	"context"
	"errors"
	"testing"

	"pizza-marketplace/shared/health"
)

type mockPinger struct {
	err error
}

func (m *mockPinger) Ping(ctx context.Context) error {
	return m.err
}

func TestReadiness_Check_AllHealthy(t *testing.T) {
	t.Parallel()

	readiness := health.NewReadiness(
		health.Check{Name: "postgres", Pinger: &mockPinger{}},
		health.Check{Name: "redis", Pinger: &mockPinger{}},
		health.Check{Name: "rabbitmq", Pinger: &mockPinger{}},
	)

	results, healthy := readiness.Check(context.Background())

	if !healthy {
		t.Fatalf("expected healthy=true")
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	for _, r := range results {
		if r.Status != "up" {
			t.Errorf("expected status=up for %s", r.Name)
		}

		if r.Error != "" {
			t.Errorf("expected empty error for %s", r.Name)
		}
	}
}

func TestReadiness_Check_OneServiceDown(t *testing.T) {
	t.Parallel()

	readiness := health.NewReadiness(
		health.Check{Name: "postgres", Pinger: &mockPinger{}},
		health.Check{Name: "redis", Pinger: &mockPinger{err: errors.New("redis unavailable")}},
		health.Check{Name: "rabbitmq", Pinger: &mockPinger{}},
	)

	results, healthy := readiness.Check(context.Background())

	if healthy {
		t.Fatalf("expected healthy=false")
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	var redisFound bool

	for _, r := range results {
		if r.Name != "redis" {
			continue
		}

		redisFound = true

		if r.Status != "down" {
			t.Errorf("expected redis status=down")
		}

		if r.Error != "redis unavailable" {
			t.Errorf("unexpected redis error: %s", r.Error)
		}
	}

	if !redisFound {
		t.Fatalf("redis result not found")
	}
}

func TestReadiness_Check_AllDown(t *testing.T) {
	t.Parallel()

	readiness := health.NewReadiness(
		health.Check{Name: "postgres", Pinger: &mockPinger{err: errors.New("postgres down")}},
		health.Check{Name: "redis", Pinger: &mockPinger{err: errors.New("redis down")}},
		health.Check{Name: "rabbitmq", Pinger: &mockPinger{err: errors.New("rabbitmq down")}},
	)

	results, healthy := readiness.Check(context.Background())

	if healthy {
		t.Fatalf("expected healthy=false")
	}

	expected := map[string]string{
		"postgres": "postgres down",
		"redis":    "redis down",
		"rabbitmq": "rabbitmq down",
	}

	for _, r := range results {
		if r.Status != "down" {
			t.Errorf("expected %s status=down", r.Name)
		}

		expectedErr := expected[r.Name]

		if r.Error != expectedErr {
			t.Errorf(
				"expected error=%q for %s, got %q",
				expectedErr,
				r.Name,
				r.Error,
			)
		}
	}
}

func TestReadiness_Check_PreservesCheckOrder(t *testing.T) {
	t.Parallel()

	readiness := health.NewReadiness(
		health.Check{Name: "first", Pinger: &mockPinger{}},
		health.Check{Name: "second", Pinger: &mockPinger{}},
		health.Check{Name: "third", Pinger: &mockPinger{}},
	)

	results, _ := readiness.Check(context.Background())

	names := make([]string, len(results))
	for i, r := range results {
		names[i] = r.Name
	}

	expected := []string{"first", "second", "third"}

	for i, name := range expected {
		if names[i] != name {
			t.Fatalf("expected order %v, got %v", expected, names)
		}
	}
}

func TestReadiness_Check_NoChecks(t *testing.T) {
	t.Parallel()

	readiness := health.NewReadiness()

	results, healthy := readiness.Check(context.Background())

	if !healthy {
		t.Fatalf("expected healthy=true with no checks configured")
	}

	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}
