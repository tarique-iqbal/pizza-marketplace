package otelamqp

import amqp "github.com/rabbitmq/amqp091-go"

type HeaderCarrier amqp.Table

func (c HeaderCarrier) Get(key string) string {
	v, ok := c[key]
	if !ok {
		return ""
	}

	s, _ := v.(string)

	return s
}

func (c HeaderCarrier) Set(key, value string) {
	c[key] = value
}

func (c HeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}

	return keys
}
