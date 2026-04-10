# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is repo-depot?

A Go monorepo providing git clone acceleration and workspace management for AI agents. It maintains bare repo caches locally and creates fast clones from them, with workspace tracking, PR management, Graphite stack operations, and Google Cloud Build integration.

## Build & Development Commands

```bash
make build              # Build all binaries (server, CLI, MCP) to bin/
make build/server       # Build gRPC server only
make build/cli          # Build CLI only
make build/mcp          # Build MCP server only
make proto              # Regenerate gRPC code from shared/proto/ → shared/gen/
make dev                # Run server with air live reload
make dev/cli ARGS="…"   # Run CLI commands directly
make test               # Run all tests
make lint               # golangci-lint
make fmt                # gofmt
make tidy               # go mod tidy all modules + go work sync
make install-tools      # Install air, protoc plugins
make docker/up          # Production server in Docker
make docker/dev         # Dev server with live reload
```

## Module Layout (Go Workspaces)

Three modules under `go.work`:

- **`shared/`** — Proto definitions (`shared/proto/repodepot.proto`) and generated gRPC code (`shared/gen/repodepotv1`). This is the shared contract between server and CLI.
- **`server/`** — gRPC server (port 50051) and MCP server (stdio or HTTP). Entry points: `server/main.go` (gRPC), `server/cmd/mcp/main.go` (MCP).
- **`cli/`** — Cobra CLI client (`rdcli`). Commands in `cli/cmd/`.

## Architecture

**Depot pattern (core domain):** Two-layer git storage:
1. Bare repo cache (`depot_path`) — one mirror clone per upstream repo
2. Workspaces (`workspaces_path`) — full clones from bare cache for agent use

**Wrapper-executor pattern:** External tools are wrapped in `server/internal/` packages with `Runner` types:
- `git.Runner`, `gh.Runner` (GitHub CLI), `gcloud.Runner` (Cloud Build), `gt.Runner` (Graphite)

**Service layer:** `server/internal/service/` implements the gRPC `RepodepotService`. `server/internal/depot/` manages on-disk cache layout and cloning. `server/internal/mcptools/` registers MCP tools that delegate to the same service layer.

**Approval workflow:** Optional RabbitMQ-based human-in-the-loop gating (`server/internal/approval/`).

**Configuration:** YAML-based (`config.yaml` or `CONFIG_PATH` env var), loaded by `server/config/`.

## Key Conventions

- **Go version:** 1.25.1
- **amp-common utilities:** `envutil.Port(ctx, "KEY", envutil.Default[uint16](val)).Value()` for env vars; `logger.Get(ctx)` for logging; `startup.ConfigureEnvironment()` for init
- **MCP transport:** stdio by default, HTTP via `MCP_TRANSPORT=http` + `MCP_HTTP_PORT`
- **Linting:** golangci-lint with most linters enabled; revive allows "ID" and "API" naming; US locale for misspell
