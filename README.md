# repo-depot

A git clone accelerator and workspace manager for AI agents.

repo-depot maintains a centralized depot of bare git repositories on disk and manages isolated workspaces on top of them. Instead of cloning from GitHub every time, agents register a workspace and clone against the local bare cache -- turning multi-minute clones into seconds. The depot tracks all workspaces, handles save/restore lifecycle, and proxies common GitHub and CI operations through a single gRPC API.

## Architecture

```
                  +-----------+
                  |  repo    |  (Cobra CLI)
                  +-----+-----+
                        | gRPC
                  +-----v-----+
                  |  server    |  (gRPC service)
                  +-----+-----+
                        |
          +-------------+-------------+
          |                           |
   +------v------+          +--------v--------+
   |  bare depot  |          |   workspaces    |
   |  (git cache) |          |  (agent clones) |
   +--------------+          +-----------------+
```

**Depot** -- A directory of bare git repos, one per upstream remote. These are fetched from GitHub once and reused across workspaces.

**Workspaces** -- Isolated project directories containing full git clones sourced from the depot. Each workspace can hold multiple repos as subtrees.

## Features

- **Fast cloning** -- Clone from the local bare cache instead of the network
- **Workspace lifecycle** -- Create, save, delete (with archival), checkout, and rename projects
- **PR management** -- Create, list, merge, comment on, and check PRs via the GitHub CLI
- **Graphite integration** -- Stack management with `gt sync`, `gt create`, `gt submit`, and navigation
- **Google Cloud Build** -- List builds, fetch logs, filter by commit/PR/branch
- **Sync operations** -- Fetch, pull, and bulk-sync repos within a project
- **Approval workflows** -- RabbitMQ-backed approval signals for gated operations

## Quick Start

### Prerequisites

- Go 1.25+
- `protoc` with Go plugins (for proto regeneration)
- Docker (optional, for containerized dev)

### Build

```bash
make build          # Build both binaries to bin/
```

This produces `bin/repo-depot-server` and `bin/repo`.

### Configure

The server reads a YAML config file (default: `config.yaml`, override with `CONFIG_PATH`):

```yaml
version: "1"
git:
  depot_path: /path/to/bare/repos
  workspaces_path: /path/to/workspaces
github:
  default_org: your-org
gcloud:
  default_project: your-project
```

### Run

```bash
# Start the server
bin/repo-depot-server

# Or with live reload
make dev
```

### Use the CLI

```bash
# Create a project workspace
repo project create my-project

# Clone a repo into it (uses bare cache)
repo clone my-project --url https://github.com/org/repo

# Save work back
repo save my-project

# List all projects
repo project list
```

The CLI connects to `localhost:50051` by default. Override with `--server`.

## Development

```bash
make dev             # Server with air live reload
make dev/cli ARGS="ping hello"  # Run CLI commands directly
make test            # Run all tests
make lint            # golangci-lint
make proto           # Regenerate gRPC code from .proto files
make tidy            # go mod tidy across all modules
make install-tools   # Install air, protoc-gen-go, protoc-gen-go-grpc
```

Docker is also available:

```bash
make docker/dev      # Dev server with air in Docker
make docker/up       # Production server in Docker
```

## Project Structure

```
repo-depot/
  cli/           Cobra CLI client (repo)
  server/        gRPC server
    config/      YAML config loader
    internal/    Server implementation
  shared/
    proto/       Protobuf definitions
    gen/         Generated gRPC code
  bin/           Build output
```

## License

TBD
