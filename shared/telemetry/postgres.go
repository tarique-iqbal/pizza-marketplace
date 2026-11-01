package telemetry

import (
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"
)

// GormPlugin wraps the official GORM OTel plugin, so services register it
// via this package (db.Use(telemetry.GormPlugin())) rather than importing
// gorm.io/plugin/opentelemetry/tracing directly.
func GormPlugin() gorm.Plugin {
	return tracing.NewPlugin(tracing.WithDBSystem("postgresql"))
}
