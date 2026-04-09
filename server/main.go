package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/amp-labs/amp-common/envutil"
	"github.com/amp-labs/amp-common/logger"
	"github.com/amp-labs/amp-common/startup"
	repodepotv1 "github.com/eberle1080/repo-depot/shared/gen/repodepot/v1"
	"github.com/eberle1080/repo-depot/server/config"
	"github.com/eberle1080/repo-depot/server/internal/approval"
	"github.com/eberle1080/repo-depot/server/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	ctx := context.Background()

	if err := startup.ConfigureEnvironment(); err != nil {
		fmt.Fprintf(os.Stderr, "startup error: %v\n", err)
		os.Exit(1)
	}

	logger.ConfigureLogging(ctx, "repo-depot-server")
	log := logger.Get(ctx)

	cfg, err := config.Load(ctx)
	if err != nil {
		logger.Fatal(ctx, "failed to load config", "error", err)
	}

	log.Info("config loaded", "depot_path", cfg.Git.DepotPath)

	approvals, err := approval.Connect(ctx, cfg.RabbitMQ.URL())
	if err != nil {
		logger.Fatal(ctx, "failed to connect approval manager", "error", err)
	}

	if approvals != nil {
		defer approvals.Close()
		log.Info("approval manager ready")
	} else {
		log.Info("approval manager disabled (no rabbitmq configured)")
	}

	port, err := envutil.Port(ctx, "GRPC_PORT", envutil.Default[uint16](50051)).Value()
	if err != nil {
		logger.Fatal(ctx, "invalid GRPC_PORT", "error", err)
	}

	lis, lisErr := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if lisErr != nil {
		logger.Fatal(ctx, "failed to listen", "error", lisErr)
	}
	svc := service.New(cfg, approvals)
	srv := grpc.NewServer()
	repodepotv1.RegisterRepodepotServiceServer(srv, svc)
	reflection.Register(srv)

	log.Info("starting gRPC server", "port", port)

	go func() {
		if err := srv.Serve(lis); err != nil {
			logger.Fatal(ctx, "gRPC server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down gRPC server")
	srv.GracefulStop()
}
