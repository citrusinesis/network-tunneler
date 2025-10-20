.PHONY: help proto gencerts build build-dev clean test test-race test-short test-coverage fmt vet lint
.PHONY: run-agent run-server run-implant install tools
.PHONY: docker-build docker-agent docker-server docker-implant
.PHONY: all

BINARY_DIR := bin
GO := go
GOFLAGS := -v

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

LDFLAGS := -s -w \
	-X 'network-tunneler/internal/version.Version=$(VERSION)' \
	-X 'network-tunneler/internal/version.Commit=$(COMMIT)' \
	-X 'network-tunneler/internal/version.BuildTime=$(BUILD_TIME)'

AGENT_BIN := $(BINARY_DIR)/agent
SERVER_BIN := $(BINARY_DIR)/server
IMPLANT_BIN := $(BINARY_DIR)/implant
GENCERTS_BIN := $(BINARY_DIR)/gencerts

all: proto build

help:
	@echo "Network Tunneler - Available targets:"
	@echo ""
	@echo "Build & Development:"
	@echo "  all           - Generate proto + build all binaries"
	@echo "  build         - Build all binaries with Nix (optimized)"
	@echo "  build-dev     - Quick development build without Nix"
	@echo "  proto         - Generate protobuf Go code"
	@echo "  gencerts      - Generate TLS certificates"
	@echo "  clean         - Clean build artifacts and caches"
	@echo "  install       - Install binaries to GOBIN"
	@echo ""
	@echo "Testing:"
	@echo "  test          - Run all tests"
	@echo "  test-short    - Run tests with -short flag"
	@echo "  test-race     - Run tests with race detector"
	@echo "  test-coverage - Run tests with coverage report"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt           - Format Go code with gofmt"
	@echo "  vet           - Run go vet"
	@echo "  lint          - Run go vet (use golangci-lint if available)"
	@echo ""
	@echo "Run:"
	@echo "  run-agent     - Run the agent"
	@echo "  run-server    - Run the server"
	@echo "  run-implant   - Run the implant"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build  - Build all Docker images with Nix"
	@echo "  docker-agent  - Build agent Docker image"
	@echo "  docker-server - Build server Docker image"
	@echo "  docker-implant- Build implant Docker image"

proto:
	@echo "==> Generating protobuf code..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/packet.proto
	@echo "==> Proto generation complete"

gencerts:
	@echo "==> Generating TLS certificates..."
	$(GO) run ./cmd/gencerts
	@echo "==> Certificates generated"

build:
	@echo "==> Building with Nix (optimized)..."
	nix build .#default
	@echo "==> Build complete"

build-dev: $(BINARY_DIR)
	@echo "==> Building binaries (development)..."
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(AGENT_BIN) ./cmd/agent
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(SERVER_BIN) ./cmd/server
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(IMPLANT_BIN) ./cmd/implant
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(GENCERTS_BIN) ./cmd/gencerts
	@echo "==> Development build complete"

$(BINARY_DIR):
	@mkdir -p $(BINARY_DIR)

clean:
	@echo "==> Cleaning build artifacts..."
	rm -rf $(BINARY_DIR)
	rm -rf result result-*
	rm -rf *.tar
	$(GO) clean -cache -testcache -modcache
	@echo "==> Clean complete"

test:
	@echo "==> Running tests..."
	$(GO) test -v ./...

test-short:
	@echo "==> Running tests (short)..."
	$(GO) test -short -timeout 30s ./...

test-race:
	@echo "==> Running tests with race detector..."
	$(GO) test -race ./...

test-coverage:
	@echo "==> Running tests with coverage..."
	$(GO) test -coverprofile=coverage.txt -covermode=atomic ./...
	$(GO) tool cover -html=coverage.txt -o coverage.html
	@echo "==> Coverage report: coverage.html"

fmt:
	@echo "==> Formatting code..."
	$(GO) fmt ./...
	@echo "==> Format complete"

vet:
	@echo "==> Running go vet..."
	$(GO) vet ./...
	@echo "==> Vet complete"

lint: vet
	@echo "==> Running linters..."
	@golangci-lint run ./...

install:
	@echo "==> Installing binaries..."
	$(GO) install ./cmd/agent
	$(GO) install ./cmd/server
	$(GO) install ./cmd/implant
	$(GO) install ./cmd/gencerts
	@echo "==> Install complete"

run-agent:
	@echo "==> Running agent..."
	$(GO) run ./cmd/agent

run-server:
	@echo "==> Running server..."
	$(GO) run ./cmd/server

run-implant:
	@echo "==> Running implant..."
	$(GO) run ./cmd/implant

docker-build: docker-agent docker-server docker-implant

docker-agent:
	@echo "==> Building agent Docker image..."
	nix build .#docker-agent
	@echo "==> Agent image: result"

docker-server:
	@echo "==> Building server Docker image..."
	nix build .#docker-server
	@echo "==> Server image: result"

docker-implant:
	@echo "==> Building implant Docker image..."
	nix build .#docker-implant
	@echo "==> Implant image: result"

tools:
	@echo "==> Installing development tools..."
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "==> Tools installed"
