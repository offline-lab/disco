# Disco Daemon

A lightweight name service daemon for offline, airgapped emergency networks using custom NSS module for native Linux integration.

**Author:** Flip Hess  
**Repository:** https://github.com/offline-lab/disco

---

## AI Disclosure

This software was written using **GLM-4.7** and **GLM-5** from [z.ai](https://z.ai).

---

## Overview

Disco provides automatic service discovery and name resolution across nodes in an offline network without requiring external DNS services. It uses a custom NSS module (libnss_disco.so.2) for seamless integration with glibc, avoiding the need for DNS servers or resolv.conf modifications.

## Architecture

- **Go Daemon**: Handles discovery, broadcast, service detection, and name resolution
- **C NSS Module**: libnss_disco.so.2 integrates with glibc via Unix domain socket
- **Zero Configuration**: Nodes discover each other automatically via UDP broadcast

## Features

### Core Functionality
- Automatic node discovery via UDP broadcast
- Service detection with human-readable names (www, smtp, mail, xmpp, etc.)
- Native NSS integration (no DNS server needed)
- Minimal resource footprint (<10MB memory)
- Works in embedded/buildroot systems (Raspberry Pi Zero 2W)

### Production Features
- **Rate Limiting**: Token bucket algorithm prevents broadcast storms (10 msg/sec)
- **Duplicate Suppression**: Message deduplication with 5-minute TTL
- **Connection Limits**: DOS protection with 100 concurrent connection limit
- **Graceful Shutdown**: Clean shutdown without resource leaks
- **Configuration Validation**: Comprehensive validation catches misconfiguration
- **Structured Logging**: Multiple formats (text/JSON) with configurable levels

### Security Features (Optional)
- **Message Signing**: HMAC-SHA256 signatures for broadcast messages
- **Signature Verification**: Rejects unsigned or invalid messages
- **Replay Protection**: 5-minute TTL prevents replay attacks
- **Trusted Peers**: Only accepts messages from known sources
- **Lightweight**: Security adds minimal overhead

### Time Synchronization (Optional)
- **GPS Time Sources**: Receives time from GPS broadcaster devices (ESP32/Pi Zero)
- **Multi-Source Validation**: Requires 2+ agreeing sources before adjusting clock
- **Clock Discipline**: Step for large offsets (>128ms), slew for small offsets
- **Security**: Signed time messages with HMAC-SHA256
- **Status Query**: `disco-time` tool shows sync status

#### Time Sync Configuration

Enable time synchronization in `config.yaml`:

```yaml
time_sync:
  enabled: true
  min_sources: 2              # Require 2+ agreeing GPS sources
  max_source_spread: 100ms   # Max disagreement between sources
  max_stale_age: 30s         # Max age of time message
  step_threshold: 128ms      # Step clock if offset > this
  slew_threshold: 500us      # Slew clock if offset > this
  poll_interval: 60s         # How often to check/adjust
  require_signed: true       # Require signed time messages
  allow_step_backward: false # Prevent stepping clock backward
```

#### GPS Broadcaster Protocol

GPS broadcasters send `TIME_ANNOUNCE` messages via UDP broadcast on port 5354:

```json
{
  "type": "TIME_ANNOUNCE",
  "timestamp": 1708123456789000000,
  "source_id": "gps-node-1",
  "clock_info": {
    "stratum": 1,
    "precision": -20,
    "root_dispersion": 0.0001,
    "reference_id": "GPS"
  }
}
```

#### Time Status Monitoring

Use `disco-time` to check synchronization status:

    $ disco-time
    Synced: YES
    Sources: 2
    Offset: +0.000023 seconds

Watch mode for continuous monitoring:

    $ disco-time -w

### Management Tools

**Unified CLI**: `disco` - Query and manage daemon

```
disco hosts                    # List all hosts with health status
disco hosts <name>             # Show host details
disco hosts forget <name>     # Remove host from cache
disco hosts mark-lost <name>   # Mark host as lost
disco services                 # List all services
disco services <name>          # Show service details
disco lookup <name>            # Look up hostname
disco status                   # Show daemon status
```

### DNS Server (Optional)

Disco can act as a DNS server for the `.disco` domain, allowing standard DNS tools to query discovered hosts.

**Enable in config.yaml:**
```yaml
dns:
  enabled: true
  port: 53
  domain: "disco"
  bind_addresses: ["0.0.0.0"]
```

**Query via DNS:**
```bash
# Standard DNS query
dig @localhost web1.disco

# Or configure /etc/resolv.conf
nameserver 127.0.0.1
search disco

# Then use normally
ping web1
```

**Note**: Requires running disco-daemon as root or with capabilities:
```bash
sudo setcap 'cap_net_bind_service=+ep' /usr/local/bin/disco-daemon
```

**Additional Commands**:
- `disco ping <hostname>` - Ping discovered hosts
- `disco announce` - Send manual announcement
- `disco time` - Time sync status
- `disco timeset` - Force time update

**Key Management**: `disco-key` - Generate and manage security keys

## Quick Start

### Building
```bash
# Build everything
make

# Build daemon only
go build -o disco-daemon cmd/daemon/main.go

# Build NSS module only
make libnss
```

### Installation
```bash
# Install daemon
sudo install -m 755 disco-daemon /usr/local/bin/

# Install NSS module
sudo install -m 644 libnss_disco.so.2 /lib/x86_64-linux-gnu/
sudo ln -sf /lib/x86_64-linux-gnu/libnss_disco.so.2 /lib/x86_64-linux-gnu/libnss_disco.so
sudo ldconfig

# Configure nsswitch.conf
# Add "disco" after "files" in hosts line:
# hosts: files disco dns

# Start daemon
sudo disco-daemon -config /etc/disco/config.yaml
```

See [docs/INSTALL.md](docs/INSTALL.md) for comprehensive installation instructions.

## Usage

### Naming Convention

- Node names: Hostname-based (e.g., `web1`, `mail1`)
- Service names: Simple descriptors (e.g., `www`, `smtp`, `mail`, `xmpp`)

### Verification
```bash
# Test name resolution
getent hosts web1

# List all discovered hosts
disco hosts

# List all discovered services
disco services

# Look up a specific host
disco lookup web1
```

## Requirements

- Go 1.21+ (for daemon)
- GCC (for NSS module)
- Linux with glibc (NSS module only works on Linux)
- Root or sudo access for installation

## Testing

### Quick Tests

```bash
# Validate configuration
./disco config validate config.yaml

# Run all Go tests
go test ./...

# Run quick integration test
./test/quick-test.sh
```

### Multi-Node Testing (Docker)

```bash
# Build and start 3 nodes
docker-compose up -d

# Query from any node
docker exec -it disco-node1 disco hosts
docker exec -it disco-node1 disco services

# Stop
docker-compose down
```

See [docs/TESTING_GUIDE.md](docs/TESTING_GUIDE.md) for comprehensive testing instructions.

## Documentation

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) - System architecture and design
- [docs/POWER_EFFICIENCY.md](docs/POWER_EFFICIENCY.md) - Power and resource optimization
- [.opencode/plans/PROJECT.md](.opencode/plans/PROJECT.md) - Project overview and roadmap

## License

MIT

Copyright (c) 2024-2025 Flip Hess
