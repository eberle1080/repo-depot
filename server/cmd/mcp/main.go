package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/amp-labs/amp-common/logger"
	"github.com/amp-labs/amp-common/startup"
	"github.com/eberle1080/mcp-protocol/schema"
	"github.com/eberle1080/repo-depot/shared/build"
	serverproto "github.com/eberle1080/mcp-protocol/server"
	mcpserver "github.com/eberle1080/mcp/server"
	"github.com/eberle1080/repo-depot/server/config"
	"github.com/eberle1080/repo-depot/server/internal/approval"
	"github.com/eberle1080/repo-depot/server/internal/mcptools"
	"github.com/eberle1080/repo-depot/server/internal/service"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := startup.ConfigureEnvironment(); err != nil {
		fmt.Fprintf(os.Stderr, "startup error: %v\n", err)
		os.Exit(1)
	}

	logger.ConfigureLogging(ctx, "repo-depot-mcp")
	log := logger.Get(ctx)

	cfg, err := config.Load(ctx)
	if err != nil {
		logger.Fatal(ctx, "failed to load config", "error", err)
	}

	approvals, err := approval.Connect(ctx, cfg.RabbitMQ.URL())
	if err != nil {
		logger.Fatal(ctx, "failed to connect approval manager", "error", err)
	}

	if approvals != nil {
		defer approvals.Close()
	}

	svc := service.New(cfg, approvals)

	version := "dev"
	if info, ok := build.ReadInfo(); ok {
		version = info.GitCommit[:8]
	}

	newHandler := serverproto.WithDefaultHandler(ctx, func(h *serverproto.DefaultHandler) error {
		return mcptools.RegisterAll(h, svc)
	})

	srv, err := mcpserver.New(
		mcpserver.WithNewHandler(newHandler),
		mcpserver.WithImplementation(schema.Implementation{
			Name:    "repo-depot",
			Version: version,
		}),
	)
	if err != nil {
		logger.Fatal(ctx, "failed to create MCP server", "error", err)
	}

	transport := os.Getenv("MCP_TRANSPORT")

	switch transport {
	case "http":
		port := os.Getenv("MCP_HTTP_PORT")
		if port == "" {
			port = "8080"
		}

		log.Info("starting MCP HTTP server", "port", port)

		if err := srv.HTTP(ctx, ":"+port).ListenAndServe(); err != nil {
			logger.Fatal(ctx, "MCP HTTP server error", "error", err)
		}
	default:
		log.Info("starting MCP stdio server")

		if err := srv.Stdio(ctx).ListenAndServe(); err != nil {
			logger.Fatal(ctx, "MCP stdio server error", "error", err)
		}
	}
}
