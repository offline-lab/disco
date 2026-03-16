# Disco Daemon - Project Overview

> **Status**: v1.0.0 Release Candidate
> **Last Updated**: 2026-03-01
> **Version**: 1.0.0-rc1

## Project Overview

Disco is a lightweight name service daemon for offline, airgapped emergency networks. It provides automatic service discovery and name resolution across nodes without requiring external DNS services or manual configuration.

### Key Features

- **Automatic Discovery**: Nodes discover each other via UDP broadcast
- **Service Detection**: Automatic detection and announcement of local services
- **Native NSS Integration**: Custom NSS module for seamless Linux integration
- **Zero Configuration**: Works out of the box, no manual setup required
- **Resource Efficient**: <10MB memory, designed for embedded systems
- **Security**: Optional message signing and verification (HMAC-SHA256)
- **Time Sync**: Optional GPS-based time synchronization for airgapped networks

## Target Environment

### Hardware
- Raspberry Pi Zero 2W (512MB RAM)
- Battery/solar powered systems
- Limited CPU and memory resources
- 50+ nodes in single broadcast domain

### Network
- Airgapped (no internet access)
- Single broadcast domain or multiple subnets
- Unknown topology at deployment

## Architecture

### Components

**1. disco-daemon** (Go binary, 5.8MB)
- Broadcast listener (UDP 5354)
- Unix socket server for NSS queries
- Service detector
- DNS server (optional, port 53)
- Time sync service (optional)

**2. disco** (Unified CLI, 3.6MB)
- All management functionality in one tool
- Queries daemon via Unix socket
- Commands: hosts, services, lookup, status, ping, announce, key, time, config

**3. disco-gps-broadcaster** (Go binary, 2.5MB)
- GPS time source
- Broadcasts TIME_ANNOUNCE messages
- For Raspberry Pi Zero, Arduino, ESP32

**4. libnss_disco.so.2** (C library)
- Integrates with glibc NSS subsystem
- Queries daemon via Unix socket
- Enables standard name resolution (gethostbyname, etc.)

### Data Flow

```
Application (curl, ssh, etc.)
         ↓
    gethostbyname()
         ↓
    glibc NSS subsystem
         ↓
    libnss_disco.so.2
         ↓
    Unix Socket (/run/disco.sock)
         ↓
    disco-daemon
         ↓
    In-memory cache of discovered hosts
```

### Discovery Protocol

```
Node A                          Node B
  │                               │
  ├──── ANNOUNCE broadcast ──────→│
  │    {hostname, IPs,           │
  │     services, signature}     │
  │                               │
  │←──── ANNOUNCE broadcast ─────┤
  │                               │
  [Both nodes update their caches]
```

## Resource Requirements

### Minimum
- **CPU**: ARMv6 or equivalent
- **RAM**: 512MB total system memory
- **Disk**: <20MB for binaries
- **Network**: UDP broadcast support

### Typical Usage
- **Memory**: <10MB for daemon
- **CPU**: <5% idle, <20% peak
- **Network**: <1 KB/sec per node
- **Disk**: Minimal (in-memory cache)

## Deployment

### Installation

```bash
# Build
make

# Install binaries
sudo install -m 755 build/bin/disco /usr/local/bin/
sudo install -m 755 build/bin/disco-daemon /usr/local/bin/
sudo install -m 755 build/bin/disco-gps-broadcaster /usr/local/bin/

# Install NSS module
sudo install -m 644 build/lib/libnss_disco.so.2 /lib/
sudo ldconfig

# Configure nsswitch.conf
# Add "disco" after "files" in hosts line:
# hosts: files disco dns

# Start daemon
sudo systemctl start disco
```

### Configuration

Minimal configuration (`/etc/disco/config.yaml`):

```yaml
daemon:
  socket_path: /run/disco.sock
  broadcast_interval: 30s
  record_ttl: 3600s

network:
  broadcast_addr: 255.255.255.255:5354
  max_broadcast_rate: 10

discovery:
  enabled: true
  detect_services: true

security:
  enabled: false  # Enable for signed messages

logging:
  level: info
  format: text
```

### Verification

```bash
# Check daemon status
disco status

# List discovered hosts
disco hosts

# Test name resolution
getent hosts web1

# Check services
disco services
```

## Development

### Building

```bash
# All binaries
make

# Individual binaries
go build -o disco-daemon cmd/daemon/main.go
go build -o disco cmd/disco/main.go

# NSS module (Linux only)
make libnss
```

### Testing

```bash
# Unit tests
go test ./...

# With coverage
go test -cover ./...

# CLI tests
go test -v ./cmd/disco/internal/cli/...

# Integration tests (requires daemon)
./test/quick-test.sh
```
