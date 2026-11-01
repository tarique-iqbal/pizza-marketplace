package telemetry

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

// GRPCClientOption instruments an outgoing gRPC connection (e.g.
// order-service's call into payment-service) using the global
// TracerProvider/MeterProvider set by Init.
func GRPCClientOption() grpc.DialOption {
	return grpc.WithStatsHandler(otelgrpc.NewClientHandler())
}

// GRPCServerOption instruments an incoming gRPC server (e.g.
// payment-service's own listener) the same way.
func GRPCServerOption() grpc.ServerOption {
	return grpc.StatsHandler(otelgrpc.NewServerHandler())
}
