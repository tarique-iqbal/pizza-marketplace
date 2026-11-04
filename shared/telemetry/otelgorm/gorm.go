package otelgorm

import (
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"
)

func Plugin() gorm.Plugin {
	return tracing.NewPlugin(tracing.WithDBSystem("postgresql"))
}
