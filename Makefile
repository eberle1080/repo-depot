PROTO_DIR := shared/proto
GEN_DIR    := shared/gen

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z/_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-28s\033[0m %s\n", $$1, $$2}'

##@ Proto

.PHONY: proto
proto: ## Regenerate protobuf/gRPC code from .proto files
	@mkdir -p $(GEN_DIR)
	protoc \
		--go_out=$(GEN_DIR) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(GEN_DIR) \
		--go-grpc_opt=paths=source_relative \
		-I $(PROTO_DIR) \
		$(PROTO_DIR)/repodepot/v1/repodepot.proto

##@ Build

.PHONY: build
build: build/server build/cli build/mcp ## Build all binaries

.PHONY: build/server
build/server: ## Build the server binary
	@mkdir -p bin
	go build -o bin/repo-depot-server ./server

.PHONY: build/cli
build/cli: ## Build the CLI binary
	@mkdir -p bin
	go build -o bin/rdcli ./cli

.PHONY: build/mcp
build/mcp: ## Build the MCP server binary
	@mkdir -p bin
	go build -o bin/repo-depot-mcp ./server/cmd/mcp

##@ Development

.PHONY: dev
dev: ## Run server locally with air (live reload)
	air -c .air.toml

.PHONY: dev/cli
dev/cli: ## Run CLI directly (pass args via ARGS="ping hello")
	go run ./cli $(ARGS)

##@ Docker

.PHONY: docker/up
docker/up: ## Start production server in Docker
	docker compose up server

.PHONY: docker/dev
docker/dev: ## Start development server in Docker with air
	docker compose up server-dev

.PHONY: docker/build
docker/build: ## Build Docker images
	docker compose build

.PHONY: docker/down
docker/down: ## Stop all Docker services
	docker compose down

##@ Testing

.PHONY: test
test: ## Run all tests
	go test ./...

.PHONY: test/coverage
test/coverage: ## Run tests with HTML coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

##@ Code Quality

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run ./...

.PHONY: fmt
fmt: ## Format all Go code
	gofmt -w .

.PHONY: tidy
tidy: ## Run go mod tidy on all modules and sync workspace
	cd shared && go mod tidy
	cd server && go mod tidy
	cd cli && go mod tidy
	go work sync

##@ Tools

.PHONY: install-tools
install-tools: ## Install dev tools (air, protoc plugins)
	go install github.com/air-verse/air@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf bin/ tmp/ coverage.out
