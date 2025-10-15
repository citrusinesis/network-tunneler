.PHONY: help build clean test fmt lint run-agent run-server run-implant docker-build proto

help:
	@echo "Network Tunneler - Available targets:"
	@echo "  proto         - Generate protobuf Go code"
	@echo "  build         - Build all binaries"
	@echo "  clean         - Clean build artifacts"
	@echo "  test          - Run tests"
	@echo "  fmt           - Format Go code"
	@echo "  lint          - Run linters"
	@echo "  run-agent     - Run the agent"
	@echo "  run-server    - Run the server"
	@echo "  run-implant   - Run the implant"
	@echo "  docker-build  - Build all Docker images with Nix"

proto:
	@echo "Generating protobuf code..."
	protoc --go_out=. --go_opt=paths=source_relative \
		proto/packet.proto

clean:
	@echo "Cleaning build artifacts..."
	rm -rf result result-*
	go clean

test:
	@echo "Running tests..."
	go test -v ./...

fmt:
	@echo "Formatting code..."
	go fmt ./...

lint:
	@echo "Running linters..."
	go vet ./...

run-agent:
	go run ./cmd/agent

run-server:
	go run ./cmd/server

run-implant:
	go run ./cmd/implant

build:
	@echo "Building all components..."
	nix build .#default

docker-build:
	@echo "Building Docker images..."
	nix build .#docker-agent
	nix build .#docker-server
	nix build .#docker-implant
