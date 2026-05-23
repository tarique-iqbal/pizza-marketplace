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
