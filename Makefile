.PHONY: help build clean test fmt lint run-agent run-server run-implant docker-build

# Default target
help:
	@echo "Network Tunneler - Available targets:"
	@echo "  build         - Build all binaries"
	@echo "  clean         - Clean build artifacts"
	@echo "  test          - Run tests"
	@echo "  fmt           - Format Go code"
	@echo "  lint          - Run linters"
	@echo "  run-agent     - Run the agent"
	@echo "  run-server    - Run the server"
	@echo "  run-implant   - Run the implant"
	@echo "  docker-build  - Build all Docker images with Nix"

# Build all binaries
build:
	@echo "Building all components..."
	nix build .#agent
	nix build .#server
	nix build .#implant

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf result result-*
	go clean

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run linters
lint:
	@echo "Running linters..."
	go vet ./...

# Run agent
run-agent:
	go run ./cmd/agent

# Run server
run-server:
	go run ./cmd/server

# Run implant
run-implant:
	go run ./cmd/implant

# Build Docker images
docker-build:
	@echo "Building Docker images..."
	nix build .#docker-agent
	nix build .#docker-server
	nix build .#docker-implant
