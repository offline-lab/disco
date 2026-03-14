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

## Current Implementation Status

### ✅ Completed Features

**Core Functionality**
- [x] Automatic node discovery via UDP broadcast
- [x] Service detection and announcement
- [x] Custom NSS module for Linux integration
- [x] Unix socket-based query interface
- [x] In-memory host cache with TTL
- [x] Rate limiting and duplicate suppression

**Security**
- [x] Message signing with HMAC-SHA256
- [x] Signature verification
- [x] Replay attack prevention (5-minute TTL)
- [x] Trusted peers list
- [x] Secure random number generation

**Time Synchronization**
- [x] GPS time source support
- [x] Multi-source validation (requires 2+ sources)
- [x] Clock discipline (step/slew)
- [x] Signed time messages
- [x] Time status monitoring

**DNS Server** (Optional)
- [x] DNS server for .disco domain
- [x] Query discovered hosts via DNS
- [x] Configurable bind addresses

**CLI Tools**
- [x] Unified `disco` command
- [x] Host management (list, show, forget, mark-lost)
- [x] Service discovery
- [x] Hostname lookup
- [x] Network diagnostics (ping, announce)
- [x] Key management
- [x] Time sync status
- [x] Config validation

**Code Quality**
- [x] Input validation on all commands
- [x] Comprehensive test suite (57.4% coverage)
- [x] Security hardening (command injection prevention)
- [x] Modular code structure
- [x] Constants centralized

### 🚧 In Progress

**Testing**
- [ ] Increase test coverage to 70%+
- [ ] Integration tests with real daemon
- [ ] Performance benchmarks
- [ ] Security testing

**Documentation**
- [ ] Complete ARCHITECTURE.md
- [ ] Add DNS documentation to README
- [ ] Update all examples
- [ ] Create troubleshooting guide

### 📋 Planned (Post v1.0.0)

**Enhancements**
- [ ] IPv6 support
- [ ] Web UI for monitoring
- [ ] Metrics export (Prometheus)
- [ ] Multi-interface broadcast
- [ ] Advanced health checks

**Optimizations**
- [ ] Reduce memory footprint
- [ ] Optimize CPU usage
- [ ] Minimize disk I/O
- [ ] Power efficiency improvements

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

## Security Model

### Airgapped Environment Considerations

**Threat Model:**
- Physical access to network
- Malicious nodes joining network
- Message injection/replay
- DNS poisoning

**Mitigations:**
- Message signing (optional)
- Replay protection (5-minute TTL)
- Trusted peers list
- Rate limiting (10 msg/sec)
- No active health checks (passive only)

**Not Protected Against:**
- Compromised trusted nodes
- Physical layer attacks
- DoS from within network

### Security Best Practices

1. **Enable signing** in production
   ```yaml
   security:
     enabled: true
     require_signed: true
   ```

2. **Pre-provision keys** on all nodes
   ```bash
   disco key generate
   # Copy keys to all nodes securely
   ```

3. **Use trusted peers** list
   ```bash
   disco key add-trusted <peer-public-key>
   ```

4. **Monitor for anomalies**
   ```bash
   disco hosts --json | jq '.[] | select(.status=="lost")'
   ```

## Performance Characteristics

### Scalability
- **Nodes tested**: Up to 50 nodes
- **Discovery time**: 30-60 seconds
- **Query latency**: <1ms (local socket)
- **Memory per node**: ~100 bytes

### Limitations
- Single broadcast domain (no routing)
- No persistence across restarts
- Fixed TTL (no dynamic adjustment)
- No load balancing

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

### Code Quality

- **Test Coverage**: 57.4% (target: 70%+)
- **Static Analysis**: go vet, staticcheck
- **Formatting**: gofmt
- **Imports**: goimports

## Project Structure

```
disco/
├── cmd/
│   ├── daemon/              # Main daemon
│   ├── disco/               # Unified CLI
│   │   ├── commands/        # CLI commands
│   │   └── internal/        # CLI utilities
│   └── gps-broadcaster/     # GPS time source
├── internal/
│   ├── config/              # Configuration parsing
│   ├── daemon/              # Daemon core logic
│   ├── discovery/           # Broadcast protocol
│   ├── dns/                 # DNS server
│   ├── logging/             # Structured logging
│   ├── nss/                 # NSS protocol
│   ├── security/            # Signing/crypto
│   ├── service/             # Service detection
│   └── timesync/            # Time synchronization
├── libnss/                  # C NSS module
├── docs/                    # Documentation
└── test/                    # Test scripts
```

## Known Issues

1. **No IPv6 support** - Only IPv4 addresses supported
2. **No persistence** - Cache lost on daemon restart
3. **Single broadcast domain** - No routing between subnets
4. **No health checks** - Passive detection only
5. **Memory could be lower** - Room for optimization

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes with tests
4. Ensure all tests pass
5. Submit pull request

## License

MIT License

## References

- [NSS Module Development](https://www.gnu.org/software/libc/manual/html_node/NSS-Modules.html)
- [mDNS/DNS-SD](https://www.rfc-editor.org/rfc/rfc6762)
- [HMAC-SHA256](https://tools.ietf.org/html/rfc2104)

## Contact

- **Author**: Flip Hess
- **Repository**: https://github.com/offline-lab/disco
- **Issues**: https://github.com/offline-lab/disco/issues

---

**Built with Go 1.21+ and GLM-4.7/5 from [z.ai](https://z.ai)**
