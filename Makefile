.PHONY: help proto gencerts build build-dev build-debug build-release clean test test-race test-short test-coverage fmt vet lint
.PHONY: run-client run-server run-proxy install tools
.PHONY: docker-build docker-client docker-server docker-proxy
.PHONY: all

BINARY_DIR := bin
GO := go
GOFLAGS := -v

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

LDFLAGS_BASE := \
	-X 'network-tunneler/internal/version.Version=$(VERSION)' \
	-X 'network-tunneler/internal/version.Commit=$(COMMIT)' \
	-X 'network-tunneler/internal/version.BuildTime=$(BUILD_TIME)'

LDFLAGS_DEBUG := $(LDFLAGS_BASE) \
	-X 'network-tunneler/internal/version.Debug=true'

LDFLAGS_RELEASE := -s -w $(LDFLAGS_BASE) \
	-X 'network-tunneler/internal/version.Debug=false'

LDFLAGS := $(LDFLAGS_DEBUG)

CLIENT_BIN := $(BINARY_DIR)/client
SERVER_BIN := $(BINARY_DIR)/server
PROXY_BIN := $(BINARY_DIR)/proxy
GENCERTS_BIN := $(BINARY_DIR)/gencerts

all: proto build

help:
	@echo "Network Tunneler - Available targets:"
	@echo ""
	@echo "Build & Development:"
	@echo "  all           - Generate proto + build all binaries"
	@echo "  build         - Build all binaries with Nix (optimized)"
	@echo "  build-dev     - Quick development build (debug mode, default)"
	@echo "  build-debug   - Debug build with symbols and debug flag"
	@echo "  build-release - Release build (stripped, optimized)"
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
	@echo "  run-client     - Run the client"
	@echo "  run-server    - Run the server"
	@echo "  run-proxy   - Run the proxy"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build  - Build all Docker images with Nix"
	@echo "  docker-client  - Build client Docker image"
	@echo "  docker-server - Build server Docker image"
	@echo "  docker-proxy- Build proxy Docker image"

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

build-dev: build-debug

build-debug: $(BINARY_DIR)
	@echo "==> Building binaries (debug mode)..."
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS_DEBUG)" -o $(CLIENT_BIN) ./cmd/client
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS_DEBUG)" -o $(SERVER_BIN) ./cmd/server
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS_DEBUG)" -o $(PROXY_BIN) ./cmd/proxy
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS_DEBUG)" -o $(GENCERTS_BIN) ./cmd/gencerts
	@echo "==> Debug build complete"

build-release: $(BINARY_DIR)
	@echo "==> Building binaries (release mode)..."
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS_RELEASE)" -o $(CLIENT_BIN) ./cmd/client
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS_RELEASE)" -o $(SERVER_BIN) ./cmd/server
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS_RELEASE)" -o $(PROXY_BIN) ./cmd/proxy
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS_RELEASE)" -o $(GENCERTS_BIN) ./cmd/gencerts
	@echo "==> Release build complete"

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
	$(GO) install ./cmd/client
	$(GO) install ./cmd/server
	$(GO) install ./cmd/proxy
	$(GO) install ./cmd/gencerts
	@echo "==> Install complete"

run-client:
	@echo "==> Running client..."
	$(GO) run ./cmd/client

run-server:
	@echo "==> Running server..."
	$(GO) run ./cmd/server

run-proxy:
	@echo "==> Running proxy..."
	$(GO) run ./cmd/proxy

docker-build: docker-client docker-server docker-proxy

docker-client:
	@echo "==> Building client Docker image..."
	nix build .#docker-client
	@echo "==> Client image: result"

docker-server:
	@echo "==> Building server Docker image..."
	nix build .#docker-server
	@echo "==> Server image: result"

docker-proxy:
	@echo "==> Building proxy Docker image..."
	nix build .#docker-proxy
	@echo "==> Proxy image: result"

tools:
	@echo "==> Installing development tools..."
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "==> Tools installed"
