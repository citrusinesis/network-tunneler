# Network Tunneler

Go Network Tunneler - L3 Network Tunneling PoC

## Overview

A proof-of-concept implementation of a network tunneling system built with Go, designed for L3 network tunneling capabilities.

## Components

- **Agent**: Client-side tunneling agent
- **Server**: Central tunneling server
- **Implant**: Network implant component

## Development

This project uses Nix flakes for reproducible development environments.

### Prerequisites

- Nix with flakes enabled
- (Optional) direnv for automatic environment loading

### Setup

```bash
# Enter the development shell
nix develop

# Or use direnv (if you have .envrc set up)
direnv allow
```

### Building

```bash
# Build all binaries
nix build

# Build specific component
nix build .#agent
nix build .#server
nix build .#implant
```

### Docker Images

```bash
# Build Docker images
nix build .#docker-agent
nix build .#docker-server
nix build .#docker-implant
```

## Project Structure

```
.
├── cmd/
│   ├── agent/      # Agent binary entry point
│   ├── server/     # Server binary entry point
│   └── implant/    # Implant binary entry point
├── flake.nix       # Nix flake configuration
├── flake.lock      # Nix flake lock file
└── go.mod          # Go module definition
```

## License

TODO: Add license information
