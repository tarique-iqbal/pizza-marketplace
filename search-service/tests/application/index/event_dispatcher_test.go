package index_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	idxapp "search-service/internal/application/index"
	"search-service/internal/domain/index"
)

type recordingHandler struct {
	received []index.EventPayload
	err      error
}

func (h *recordingHandler) Handle(ctx context.Context, event index.EventPayload) error {
	h.received = append(h.received, event)
	return h.err
}

func TestEventDispatcher_DispatchesToRegisteredHandler(t *testing.T) {
	dispatcher := idxapp.NewEventDispatcher()
	handler := &recordingHandler{}
	dispatcher.Register("restaurant.launched", handler)

	payload := index.EventPayload{Name: "restaurant.launched", Data: []byte(`{}`)}

	require.NoError(t, dispatcher.Dispatch(context.Background(), payload))
	require.Len(t, handler.received, 1)
	assert.Equal(t, payload, handler.received[0])
}

func TestEventDispatcher_UnknownEvent_ReturnsError(t *testing.T) {
	dispatcher := idxapp.NewEventDispatcher()

	err := dispatcher.Dispatch(context.Background(), index.EventPayload{Name: "restaurant.unknown"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "restaurant.unknown")
}

func TestEventDispatcher_PropagatesHandlerError(t *testing.T) {
	dispatcher := idxapp.NewEventDispatcher()
	handler := &recordingHandler{err: errors.New("boom")}
	dispatcher.Register("restaurant.launched", handler)

	err := dispatcher.Dispatch(context.Background(), index.EventPayload{Name: "restaurant.launched"})

	require.Error(t, err)
	assert.Equal(t, "boom", err.Error())
}
