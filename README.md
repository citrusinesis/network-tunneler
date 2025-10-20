# Network Tunneler

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A distributed L3 network tunneling system implementing a Client-Server-Proxy architecture for transparent traffic forwarding across network boundaries.

## 📋 Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Features](#features)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Usage](#usage)
- [Technical Deep Dive](#technical-deep-dive)
- [Roadmap](#roadmap)

## Overview

Network Tunneler is a distributed network tunneling system that enables transparent traffic forwarding across network boundaries. It implements a three-component architecture where the Client captures packets using Netfilter, the Server acts as a central relay, and the Proxy forwards traffic to remote networks.

### Use Cases

- **Network Extension**: Extend network connectivity across NAT boundaries and firewalls
- **Remote Access**: Access remote network resources from local networks
- **Network Research**: Explore packet manipulation and tunneling protocols
- **Distributed Systems**: Study connection state management across network boundaries
- **Development**: Build and test distributed network applications

## Architecture

```
┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
│     Client      │         │     Server      │         │     Proxy       │
│   (Local Net)   │◄───────►│   (Central)     │◄───────►│  (Remote Net)   │
│                 │  mTLS   │     Relay       │  mTLS   │                 │
└─────────────────┘         └─────────────────┘         └─────────────────┘
        │                           │                            │
   Netfilter                  Connection                   Remote
   Intercept                   Tracking                    Network
   (iptables)                  (4-tuple)                  (net.Dial)
```

### Component Roles

#### Client (Packet Capture & Delivery)
- Captures packets destined for specific CIDR ranges using **Netfilter/iptables**
- Extracts original destination via `SO_ORIGINAL_DST` socket option
- Generates connection IDs from TCP 4-tuple (src IP, src port, dst IP, dst port)
- Serializes packets with metadata using **Protocol Buffers**
- Maintains bidirectional connection mapping for response delivery
- Communicates with Server over **mTLS** with gRPC

#### Server (Central Relay & Router)
- Accepts connections from multiple Clients and Proxies
- Implements **connection tracking** to map Client connections → Proxy connections
- Routes packets based on connection ID hashing
- Multiplexes multiple tunnels through single infrastructure
- Provides metrics and monitoring capabilities
- Handles Client/Proxy registration and heartbeat

#### Proxy (Network Gateway)
- Establishes **reverse connection** to Server (outbound-only connectivity)
- Receives packets from Server for specific network ranges
- Rewrites source IP to appear as Proxy's address
- Forwards to remote network targets using `net.Dial`
- Collects responses and relays back through Server
- Supports multiple managed CIDR blocks

### Packet Flow Example

```
User on Client machine: curl http://100.64.1.5:80

1. Netfilter captures packet → redirects to local handler (port 9999)
2. Client extracts original dest via SO_ORIGINAL_DST (100.64.1.5:80)
3. Client creates connection ID: hash(src_ip, src_port, dst_ip, dst_port)
4. Client → Server: PACKET(conn_id, data, metadata) [gRPC stream]
5. Server maps connection to appropriate Proxy
6. Server → Proxy: PACKET(conn_id, data, dst) [gRPC stream]
7. Proxy rewrites: src_ip = proxy_ip
8. Proxy → Remote(192.168.1.5:80): TCP connection
9. Remote → Proxy: HTTP Response
10. Proxy → Server: RESPONSE(conn_id, data)
11. Server → Client: RESPONSE(conn_id, data)
12. Client delivers to original connection
```

## Features

### Core Functionality

- ✅ **Netfilter Integration**: Kernel-level packet interception on Linux
- ✅ **SO_ORIGINAL_DST**: Retrieves pre-NAT destination addresses
- ✅ **Connection Tracking**: Bidirectional 4-tuple mapping with SHA-256 hashing
- ✅ **gRPC Streaming**: Efficient bidirectional packet transport
- ✅ **mTLS Authentication**: Mutual TLS for all connections
- ✅ **Protocol Buffers**: High-performance serialization

### Advanced Features

- ✅ **Concurrent Goroutine Management**: Efficient goroutine lifecycle management
  - Per-connection goroutines for packet handling
  - Graceful shutdown with context cancellation
  - WaitGroup synchronization for cleanup
  - Read/write loop goroutines with proper error handling
- ✅ **Multiple Configuration Sources**: YAML, JSON, .env files
- ✅ **Structured Logging**: JSON logs with contextual information
- ✅ **Graceful Shutdown**: Signal handling (SIGINT, SIGTERM)
- ✅ **Connection Cleanup**: Idle timeout and resource management
- ✅ **Concurrent Safety**: Thread-safe connection tracking with `sync.RWMutex`
- ✅ **Error Recovery**: Automatic reconnection with exponential backoff
- ✅ **Certificate Generation**: Built-in mTLS certificate generator

### Phase 1+ Features (In Progress)

- 🚧 **TCP Sequence Tracking**: Simplified out-of-order detection
- 🚧 **Connection State Machine**: Basic TCP state tracking (SYN, FIN, RST)
- 🚧 **Metrics Export**: Prometheus-compatible metrics endpoint
- 🚧 **Multi-Proxy Support**: Route to different proxies based on CIDR

## Prerequisites

### Required

- **Go 1.21+** or **Nix with flakes enabled**
- **Linux** (for Netfilter/iptables support on Client)
- **Root privileges** (for iptables rules on Client)
- **protoc** (Protocol Buffer compiler)

### Optional

- **direnv**: Automatic environment loading
- **Docker**: For containerized deployment
- **golangci-lint**: For code quality checks

## Installation

### Option 1: Using Nix (Recommended)

```bash
# Clone the repository
git clone https://github.com/yourusername/network-tunneler.git
cd network-tunneler

# Enter development shell (installs all dependencies)
nix develop

# Build all binaries
nix build

# Binaries available in ./result/bin/
./result/bin/client --version
./result/bin/server --version
./result/bin/proxy --version
```

### Option 2: Using Go

```bash
# Clone the repository
git clone https://github.com/yourusername/network-tunneler.git
cd network-tunneler

# Install dependencies
go mod download

# Generate protobuf code
make proto

# Build binaries
make build-dev

# Binaries available in ./bin/
./bin/client --version
./bin/server --version
./bin/proxy --version
```

## Quick Start

### 1. Generate TLS Certificates

```bash
# Generate CA, server, client, and proxy certificates
make gencerts

# Or manually
./bin/gencerts
```

This creates:
- `certs/ca/` - Certificate Authority
- `certs/server/` - Server certificates
- `certs/client/` - Client certificates
- `certs/proxy/` - Proxy certificates

### 2. Start the Server

```bash
# Terminal 1
./bin/server --listen :8081 \
  --tls-cert certs/server/cert.pem \
  --tls-key certs/server/key.pem \
  --tls-ca certs/ca/cert.pem

# Or with config file
./bin/server --config configs/server.yaml
```

### 3. Start the Proxy

```bash
# Terminal 2
./bin/proxy \
  --server localhost:8081 \
  --proxy-id proxy-1 \
  --managed-cidr 192.168.1.0/24 \
  --tls-cert certs/proxy/cert.pem \
  --tls-key certs/proxy/key.pem \
  --tls-ca certs/ca/cert.pem

# Or with config file
./bin/proxy --config configs/proxy.yaml
```

### 4. Start the Client (Requires Root)

```bash
# Terminal 3
sudo ./bin/client \
  --server localhost:8081 \
  --cidr 100.64.0.0/10 \
  --listen-port 9999 \
  --tls-cert certs/client/cert.pem \
  --tls-key certs/client/key.pem \
  --tls-ca certs/ca/cert.pem

# Or with config file
sudo ./bin/client --config configs/client.yaml
```

### 5. Test the Tunnel

```bash
# Terminal 4 - Traffic to 100.64.0.0/10 will be tunneled
curl http://100.64.1.5:80

# The request will be:
# 1. Captured by Client (iptables)
# 2. Sent to Server
# 3. Forwarded to Proxy
# 4. Delivered to actual remote IP (192.168.1.5:80)
# 5. Response returned through same path
```

## Configuration

### Configuration File Formats

All components support multiple configuration formats: **YAML**, **JSON**, and **.env**

#### Example: Client Configuration (YAML)

```yaml
# configs/client.yaml
server_addr: "localhost:8081"
listen_port: 9999
target_cidr: "100.64.0.0/10"
client_id: ""  # Auto-generated if empty

tls:
  cert_file: "certs/client/cert.pem"
  key_file: "certs/client/key.pem"
  ca_file: "certs/ca/cert.pem"
  insecure: false

log:
  level: "info"
  format: "json"
  output: "stdout"
```

#### Example: Server Configuration (YAML)

```yaml
# configs/server.yaml
listen_addr: ":8081"

tls:
  cert_file: "certs/server/cert.pem"
  key_file: "certs/server/key.pem"
  ca_file: "certs/ca/cert.pem"
  insecure: false

log:
  level: "info"
  format: "json"
  output: "stdout"
```

#### Example: Proxy Configuration (YAML)

```yaml
# configs/proxy.yaml
server_addr: "localhost:8081"
proxy_id: "proxy-1"
managed_cidr: "192.168.1.0/24"

tls:
  cert_file: "certs/proxy/cert.pem"
  key_file: "certs/proxy/key.pem"
  ca_file: "certs/ca/cert.pem"
  insecure: false

log:
  level: "info"
  format: "json"
  output: "stdout"
```

## Usage

### Basic Workflow

1. **Deploy Server** in a publicly accessible location
2. **Deploy Proxy** in the remote network (behind NAT/firewall)
3. **Run Client** on the local machine
4. **Send Traffic** to the CIDR range configured on Client

### Advanced Scenarios

#### Multiple Proxies for Different Networks

```bash
# Proxy 1 - Manages Network A (192.168.1.0/24)
./bin/proxy --proxy-id proxy-1 --managed-cidr 192.168.1.0/24 --server server:8081

# Proxy 2 - Manages Network B (10.0.0.0/8)
./bin/proxy --proxy-id proxy-2 --managed-cidr 10.0.0.0/8 --server server:8081

# Client routes automatically based on destination
curl http://100.64.1.5:80   # → proxy-1 → 192.168.1.5:80
curl http://100.64.10.5:80  # → proxy-2 → 10.0.10.5:80
```

## Technical Deep Dive

### Netfilter Integration

The Client uses Linux Netfilter/iptables to intercept packets:

```go
// Setup iptables rule to redirect traffic
iptables -t nat -A OUTPUT \
  -d 100.64.0.0/10 \
  -j REDIRECT --to-ports 9999

// Extract original destination (before NAT)
const SO_ORIGINAL_DST = 80
addr := &syscall.RawSockaddrInet4{}
syscall.Getsockopt(fd, syscall.SOL_IP, SO_ORIGINAL_DST, addr)
```

### Connection ID Generation

Connections are tracked using a cryptographic hash of the 4-tuple:

```go
// Generate connection ID from 4-tuple
func GenerateConnectionID(srcIP, srcPort, dstIP, dstPort string) string {
    data := fmt.Sprintf("%s:%s->%s:%s", srcIP, srcPort, dstIP, dstPort)
    hash := sha256.Sum256([]byte(data))
    return hex.EncodeToString(hash[:16]) // 32 hex chars
}
```

### Goroutine Management

The system uses structured concurrency patterns for efficient goroutine lifecycle management:

```go
// Per-connection handler goroutine
func (h *ConnectionHandler) handleConnection(conn net.Conn) {
    defer conn.Close()
    defer h.tracker.Remove(connID)

    // Each connection gets its own goroutine
    // Cleanup is guaranteed by defer statements
}

// Server connection with read/write goroutines
type ServerConnection struct {
    ctx        context.Context
    cancel     context.CancelFunc
    wg         sync.WaitGroup
}

func (sc *ServerConnection) Connect(ctx context.Context) error {
    sc.ctx, sc.cancel = context.WithCancel(ctx)

    // Start read loop goroutine
    sc.wg.Add(1)
    go sc.readLoop()

    // Start write loop goroutine
    sc.wg.Add(1)
    go sc.writeLoop()

    return nil
}

func (sc *ServerConnection) Close() error {
    sc.cancel()      // Signal goroutines to stop
    sc.wg.Wait()     // Wait for all goroutines to finish
    return nil
}

func (sc *ServerConnection) readLoop() {
    defer sc.wg.Done()

    for {
        select {
        case <-sc.ctx.Done():
            return  // Context cancelled, exit gracefully
        default:
            // Read and process packets
        }
    }
}
```

**Key Patterns:**
- **Context Cancellation**: Propagate shutdown signals to all goroutines
- **WaitGroup Synchronization**: Ensure all goroutines complete before exit
- **Defer Cleanup**: Guarantee resource cleanup even on panic
- **Channel Coordination**: Safe communication between goroutines
- **Per-Connection Isolation**: Each connection handled independently

### gRPC Streaming

Bidirectional streaming for efficient packet transport:

```protobuf
service Tunnel {
  rpc ClientStream(stream ClientMessage) returns (stream ServerMessage);
  rpc ProxyStream(stream ProxyMessage) returns (stream ServerMessage);
}
```

### Concurrent Connection Tracking

Thread-safe connection state management:

```go
type ConnectionTracker struct {
    connections map[string]*ConnectionState
    mu          sync.RWMutex
}

// Lock only when modifying
func (t *ConnectionTracker) Track(conn net.Conn) {
    t.mu.Lock()
    defer t.mu.Unlock()
    t.connections[id] = state
}

// RLock for reading
func (t *ConnectionTracker) Get(id string) *ConnectionState {
    t.mu.RLock()
    defer t.mu.RUnlock()
    return t.connections[id]
}
```

## Roadmap

### Phase 0: Foundation ✅ (Complete)

- [x] Basic Client, Server, Proxy implementation
- [x] Netfilter integration with SO_ORIGINAL_DST
- [x] Connection tracking (4-tuple mapping)
- [x] gRPC bidirectional streaming
- [x] mTLS authentication
- [x] Protocol Buffer serialization
- [x] Goroutine lifecycle management

### Phase 1: Stability & TCP Tracking 🚧 (In Progress)

- [ ] Simplified TCP sequence tracking
- [ ] Out-of-order packet detection (log only)
- [ ] Basic TCP state machine (SYN, ESTABLISHED, FIN)
- [ ] Connection timeout and cleanup
- [ ] Enhanced error handling and recovery
- [ ] Structured JSON logging

### Phase 2: Production Quality 📅 (Planned)

- [ ] Multiple Proxy support with routing
- [ ] Performance metrics (Prometheus format)
- [ ] Throughput and latency tracking
- [ ] Connection statistics dashboard
- [ ] Rate limiting and flow control
- [ ] Enhanced error recovery

---

**Built with ❤️ in Go**
