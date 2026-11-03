package logobs

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestLevelHandler_DropsRecordsBelowLevel(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(&levelHandler{Handler: inner, level: slog.LevelInfo})

	logger.Debug("dropped")

	if buf.Len() != 0 {
		t.Fatalf("expected debug record to be dropped, got %q", buf.String())
	}

	logger.With("k", "v").Info("kept")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("failed to parse log line: %v", err)
	}

	if record["k"] != "v" {
		t.Fatal("expected info record with attrs to pass through after With")
	}
}
