package telemetry

import amqp "github.com/rabbitmq/amqp091-go"

// AMQPHeaderCarrier adapts amqp.Table to propagation.TextMapCarrier, so a
// trace context can be injected into (or extracted from) AMQP message
// headers. This is the generic primitive only — the actual publish/consume
// wrappers that use it need the outbox's own trace_context column (phase 3),
// since every publish in this repo goes through an outbox relay running on
// the worker's own polling loop, not the original request.
type AMQPHeaderCarrier amqp.Table

func (c AMQPHeaderCarrier) Get(key string) string {
	v, ok := c[key]
	if !ok {
		return ""
	}

	s, _ := v.(string)

	return s
}

func (c AMQPHeaderCarrier) Set(key, value string) {
	c[key] = value
}

func (c AMQPHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}

	return keys
}
