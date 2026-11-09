package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	grpclib "google.golang.org/grpc"

	"payment-service/internal/container"
	"payment-service/internal/interfaces/grpc/pb"
	"payment-service/internal/interfaces/http/routes"

	"pizza-marketplace/shared/logobs"
	"pizza-marketplace/shared/telemetry"
	"pizza-marketplace/shared/telemetry/otelgin"
	"pizza-marketplace/shared/telemetry/otelgrpc"
)

const (
	serviceName           = "payment-api"
	httpAddr              = ":8080"
	grpcAddr              = ":50051"
	shutdownTimeout       = 10 * time.Second
	telemetryFlushTimeout = 5 * time.Second
)

func main() {
	logger := logobs.NewLogger(serviceName)

	shutdown, err := telemetry.Init(context.Background(), telemetry.Config{ServiceName: serviceName})
	if err != nil {
		logger.Error("failed to initialize telemetry", "error", err)
		return
	}
	defer flushTelemetry(shutdown)

	app, err := container.NewAPIContainer()
	if err != nil {
		logger.Error("application exited with error", "error", err)
		return
	}
	defer app.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	grpcDone := make(chan struct{})

	go func() {
		defer close(grpcDone)
		serveGRPC(ctx, logger, app)
	}()

	router := gin.New()
	router.Use(gin.Recovery(), otelgin.Middleware(serviceName), logobs.Middleware(logger))

	routes.SetupRoutes(router, app.Handlers)

	if err := serve(ctx, logger, router); err != nil {
		logger.Error("server exited with error", "error", err)
	}

	stop()
	<-grpcDone
}

func serve(ctx context.Context, logger *slog.Logger, handler http.Handler) error {
	server := &http.Server{Addr: httpAddr, Handler: handler}

	errCh := make(chan error, 1)

	go func() {
		errCh <- server.ListenAndServe()
	}()

	logger.Info("starting HTTP server", "addr", httpAddr)

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	logger.Info("shutdown initiated")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	logger.Info("shutdown complete")

	return nil
}

func serveGRPC(ctx context.Context, logger *slog.Logger, app *container.APIContainer) {
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		logger.Error("failed to listen for grpc", "error", err)
		return
	}

	server := grpclib.NewServer(otelgrpc.ServerOption())
	pb.RegisterPaymentServiceServer(server, app.GRPCServer)

	stopped := make(chan struct{})

	go func() {
		defer close(stopped)
		<-ctx.Done()
		stopGRPC(server)
	}()

	logger.Info("grpc server listening", "addr", grpcAddr)

	if err := server.Serve(lis); err != nil {
		logger.Error("grpc server exited with error", "error", err)
	}

	<-stopped
}

func stopGRPC(server *grpclib.Server) {
	done := make(chan struct{})

	go func() {
		server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(shutdownTimeout):
		server.Stop()
	}
}

func flushTelemetry(shutdown telemetry.Shutdown) {
	ctx, cancel := context.WithTimeout(context.Background(), telemetryFlushTimeout)
	defer cancel()

	_ = shutdown(ctx)
}
