# NSS Daemon

A lightweight name service daemon for offline, airgapped emergency networks using custom NSS module for native Linux integration.

**Author:** Flip Hess  
**Repository:** https://github.com/offline-lab/disco

---

## AI Disclosure

This software was written using **GLM-4.7** and **GLM-5** from [z.ai](https://z.ai).

---

## Overview

This daemon provides automatic service discovery and name resolution across nodes in an offline network without requiring external DNS services. It uses a custom NSS module (libnss_daemon.so.2) for seamless integration with glibc, avoiding the need for DNS servers or resolv.conf modifications.

## Architecture

- **Go Daemon**: Handles discovery, broadcast, service detection, and name resolution
- **C NSS Module**: libnss_daemon.so.2 integrates with glibc via Unix domain socket
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

### Management Tools
- **CLI Query Tool**: `nss-query` - Query daemon for hosts, services, and status
  - `nss-query hosts` - List all discovered hosts with last seen time
  - `nss-query services` - List all discovered services by type
  - `nss-query hosts-services` - Detailed view of hosts and their services
  - `nss-query lookup <name>` - Look up a specific host
- **Config Validator**: `nss-config-validate` - Validate configuration files before starting
- **Key Management**: `nss-key` - Generate and manage security keys

## Quick Start

### Building
```bash
# Build everything
make

# Build daemon only
go build -o nss-daemon cmd/daemon/main.go

# Build NSS module only
make libnss
```

### Installation
```bash
# Install daemon
sudo install -m 755 nss-daemon /usr/local/bin/

# Install NSS module
sudo install -m 644 libnss_daemon.so.2 /lib/x86_64-linux-gnu/
sudo ln -sf /lib/x86_64-linux-gnu/libnss_daemon.so.2 /lib/x86_64-linux-gnu/libnss_daemon.so
sudo ldconfig

# Configure nsswitch.conf
# Add "daemon" after "files" in hosts line:
# hosts: files daemon dns

# Start daemon
sudo nss-daemon -config /etc/nss-daemon/config.yaml
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
nss-query hosts

# List all discovered services
nss-query services

# List hosts with their services
nss-query hosts-services

# Look up a specific host
nss-query lookup web1
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
./nss-config-validate config.yaml

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
docker exec -it nss-daemon-web1 nss-query hosts
docker exec -it nss-daemon-web1 nss-query hosts-services

# Stop
docker-compose down
```

See [docs/TESTING_GUIDE.md](docs/TESTING_GUIDE.md) for comprehensive testing instructions.

## Documentation

- [docs/INSTALL.md](docs/INSTALL.md) - Comprehensive installation guide
- [docs/TESTING_GUIDE.md](docs/TESTING_GUIDE.md) - Testing guide
- [docs/NSS_QUERY.md](docs/NSS_QUERY.md) - Query tool documentation

## License

MIT

Copyright (c) 2024 Flip Hess
