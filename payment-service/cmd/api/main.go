package main

import (
	"log/slog"
	"net"

	"github.com/gin-gonic/gin"
	grpclib "google.golang.org/grpc"

	"payment-service/internal/container"
	"payment-service/internal/infrastructure/observability"
	logobs "payment-service/internal/infrastructure/observability/logger"
	"payment-service/internal/interfaces/grpc/pb"
	"payment-service/internal/interfaces/http/routes"
)

const grpcAddr = ":50051"

func main() {
	logger := logobs.NewLogger("payment-api")

	app, err := container.NewAPIContainer()
	if err != nil {
		logger.Error("application exited with error", "error", err)
		return
	}
	defer app.Close()

	go startGRPCServer(logger, app)

	router := gin.New()
	router.Use(gin.Recovery(), observability.Middleware(logger))

	routes.SetupRoutes(router, app.Handlers)

	if err := router.Run(":8080"); err != nil {
		logger.Error("failed to start server", "error", err)
	}
}

func startGRPCServer(logger *slog.Logger, app *container.APIContainer) {
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		logger.Error("failed to listen for grpc", "error", err)
		return
	}

	grpcServer := grpclib.NewServer()
	pb.RegisterPaymentServiceServer(grpcServer, app.GRPCServer)

	logger.Info("grpc server listening", "addr", grpcAddr)

	if err := grpcServer.Serve(lis); err != nil {
		logger.Error("grpc server exited with error", "error", err)
	}
}
