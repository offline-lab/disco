# DNS Daemon - Offline Emergency Network

## Project Overview

Building a lightweight DNS daemon for offline, airgapped networks used in emergencies. The daemon enables automatic service discovery and name resolution across nodes with minimal resource usage.

### Target Hardware
- Raspberry Pi Zero 2W or similar low-powered devices
- Battery-powered systems
- Limited CPU and memory

### Network Scale
- 50+ nodes
- Multiple interfaces per node
- Unknown topology (single broadcast domain or multiple subnets)

## Requirements

### Functional Requirements
1. **Automatic Node Discovery**
   - Nodes discover each other on network join
   - Broadcast/multicast-based announcements
   - Support multiple network interfaces

2. **Service Discovery**
   - Daemon detects local services automatically
   - Broadcast service announcements
   - Human-readable service names (smtp, mail, www, etc)
   - Hostname-based node identification

3. **Name Resolution**
   - Services accessible via human-readable names
   - No /etc/hosts rewriting (avoid root filesystem writes)
   - Integrate with Linux name resolution
   - Standard DNS or DNS-like protocol

4. **Security**
   - No master node or manual key exchange
   - "Just works" when nodes added
   - Security-minded design to prevent DNS poisoning/takeover
   - Can pre-provision certificates if needed
   - Simple trust model

5. **Resource Constraints**
   - Minimal CPU usage
   - Minimal memory usage
   - Single binary daemon
   - Go language preferred

### Non-Functional Requirements
- No external dependencies on network services
- Airgapped operation
- Broadcast storm prevention
- Graceful handling of partial connectivity

## Name Resolution Integration

### CHOSEN APPROACH: Custom NSS Module + Go Daemon

**Architecture:**
- Go daemon handles discovery, broadcast, and service detection
- C libnss module (libnss_daemon.so.2) queries daemon via Unix domain socket
- Add "hosts: files daemon" to /etc/nsswitch.conf

**Pros:**
- Native integration with glibc
- No DNS server needed
- No resolv.conf changes
- No systemd requirements
- No capabilities/setcap needed
- Works in embedded/buildroot systems
- Very low resource footprint

**Cons:**
- Requires C compilation for NSS module
- Need to install libnss_daemon.so.2 to /lib/

**Implementation:**
1. Go daemon listens on Unix socket (e.g., /run/nss-daemon.sock)
2. NSS module connects and queries for hostnames
3. Simple protocol: JSON or text-based queries
4. Fast, efficient IPC over Unix domain socket

## Architecture Design

### Discovery Protocol
```
┌─────────────┐         Broadcast          ┌─────────────┐
│   Node A    │ ◄─────────────────────────► │   Node B    │
│  Daemon     │                           │  Daemon     │
│             │   Announce:              │             │
│ - hostname  │   {type: "announce",      │ - hostname  │
│ - services  │    hostname: "web1",      │ - services  │
│ - IP:port   │    services: [           │ - IP:port   │
│ - signature │      {name: "www",        │ - signature │
│             │       port: 80},          │             │
│             │       {name: "smtp",      │             │
│             │       port: 25}           │             │
│             │      ]}                    │             │
└─────────────┘                           └─────────────┘
```

### Name Resolution Flow
```
┌──────────────┐     getaddrinfo()     ┌──────────────┐
│ Application  │ ──────────────────►    │   glibc NSS  │
│ (curl, etc)  │                        │   (libnss_   │
│              │ ◄──────────────────── │    daemon)   │
│              │   hostname resolution  │              │
└──────────────┘                        └──────┬───────┘
                                               │
                                               ▼
                                     ┌─────────────────┐
                                     │   Unix Socket   │
                                     │  /run/daemon.sock│
                                     └────────┬────────┘
                                              │
                                              ▼
                                     ┌─────────────────┐
                                     │   Go Daemon     │
                                     │  (discovery +   │
                                     │   service cache)│
                                     └─────────────────┘
```

### Service Detection
- Scan common ports or use system APIs
- Map detected services to names:
  - Port 80/443 → "www"
  - Port 25 → "smtp"
  - Port 143/993 → "mail"
  - Port 5222 → "xmpp"
  - Configurable mapping

### Record Storage
- In-memory cache
- Optional disk persistence
- TTL-based expiration
- Conflict resolution based on:
  - Timestamp (newer wins)
  - Signature verification

## Security Model

### Node Authentication
- Pre-provisioned certificates (optional)
- OR trust on first use (TOFU) with fingerprint storage
- OR simple shared secret for initial deployment

### Broadcast Protection
- Rate limiting announcements
- Duplicate suppression
- Only forward valid signed messages

### DNS Poisoning Prevention
- All broadcast messages signed
- Reject unsigned messages if auth enabled
- Validate signatures before adding records

### Threat Mitigation
- Replay attack prevention (timestamps/nonce)
- Spoofing prevention (signatures)
- Broadcast storms (rate limiting)

## Implementation Plan

### Phase 1: Core Daemon (Go)
- [ ] Project setup (Go modules, basic structure)
- [ ] Configuration file format (YAML)
- [ ] Unix socket server for NSS queries
- [ ] In-memory record storage
- [ ] Query protocol implementation

### Phase 2: NSS Module (C)
- [ ] libnss_daemon.so.2 structure
- [ ] _nss_daemon_gethostbyname_r implementation
- [ ] _nss_daemon_gethostbyname2_r implementation
- [ ] _nss_daemon_gethostbyaddr_r implementation
- [ ] Unix socket client code

### Phase 3: Discovery Protocol
- [ ] Broadcast message format definition
- [ ] Announcement mechanism (UDP broadcast)
- [ ] Broadcast listener
- [ ] Rate limiting and duplicate suppression
- [ ] Multiple interface support

### Phase 4: Service Detection
- [ ] Port scanning/monitoring
- [ ] Service name mapping
- [ ] Local service announcement
- [ ] Service lifecycle tracking

### Phase 5: Security
- [ ] Signature generation/verification
- [ ] Certificate/key management
- [ ] Replay attack prevention
- [ ] Broadcast message validation

### Phase 6: Integration
- [ ] Installation scripts
- [ ] systemd service file (optional)
- [ ] NSS module installation to /lib/
- [ ] nsswitch.conf management
- [ ] Configuration examples

### Phase 7: Testing
- [ ] Unit tests (Go daemon)
- [ ] Unit tests (NSS module)
- [ ] Integration tests (multi-node)
- [ ] Performance benchmarks
- [ ] Security testing

## Dependencies

### Go Daemon
- `gopkg.in/yaml.v3` - Configuration parsing
- `github.com/google/uuid` - Unique message IDs
- `golang.org/x/crypto` - Cryptographic functions (signatures)

### C NSS Module
- Standard C library (glibc headers)
- No external dependencies

### Build Tools
- Go 1.21+ for daemon
- GCC for NSS module
- Make for building both

## Configuration

### Example config.yaml
```yaml
daemon:
  socket_path: "/run/nss-daemon.sock"
  broadcast_interval: 30s
  record_ttl: 3600s

network:
  interfaces: ["eth0", "wlan0"]
  broadcast_addr: "255.255.255.255:5353"
  max_broadcast_rate: 10/second

discovery:
  enabled: true
  detect_services: true
  service_port_mapping:
    www: [80, 443]
    smtp: [25, 587]
    mail: [143, 993]
    xmpp: [5222, 5269]

security:
  enabled: true
  cert_path: "/etc/nss-daemon/cert.pem"
  key_path: "/etc/nss-daemon/key.pem"
  trusted_peers: "/etc/nss-daemon/trusted.pem"
  require_signed: true

logging:
  level: "info"
  format: "text"
```

## Naming Convention

### Node Names
- Hostname-based (from `/etc/hostname` or system)
- Example: `web1`, `mail1`, `gateway`

### Service Names
- Simple, human-readable
- Format: `<hostname>-<service>` or just `<service>` for global services
- Examples:
  - `www` → web server (can be load balanced)
  - `web1-www` → web server on node web1
  - `smtp` → SMTP server
  - `mail` → IMAP server
  - `xmpp` → XMPP server

### Name Resolution
- Applications resolve via getaddrinfo()
- glibc loads libnss_daemon.so.2
- NSS module queries Go daemon via Unix socket
- Returns standard struct hostent to application

## Deployment

### Installation
```bash
# Build Go daemon
go build -o nss-daemon cmd/daemon/main.go

# Build NSS module
make libnss_daemon.so

# Install daemon
sudo cp nss-daemon /usr/local/bin/

# Install NSS module
sudo cp libnss_daemon.so.2 /lib/
sudo ldconfig

# Enable systemd service (optional)
sudo systemctl enable nss-daemon
sudo systemctl start nss-daemon
```

### Configuration
```bash
# Add to /etc/nsswitch.conf
# hosts: files daemon dns

# Create config directory
sudo mkdir -p /etc/nss-daemon
sudo cp config.yaml /etc/nss-daemon/
```

## Testing Strategy

### Unit Tests
- DNS query/response handling
- Broadcast message parsing
- Signature validation
- Service detection

### Integration Tests
- Multi-node setup with containers
- Broadcast propagation
- Record consistency
- Failover scenarios

### Performance Tests
- Query latency
- Broadcast overhead
- Memory usage
- CPU usage

## Resource Budget (RPi Zero 2W)

### Target Metrics
- Memory: < 20MB
- CPU: < 5% idle, < 20% peak
- Binary size: < 10MB
- Broadcast traffic: < 1 KB/sec per node

## Future Enhancements

- IPv6 support
- DNSSEC support
- Web UI for monitoring
- Metrics export (Prometheus)
- Integration with other services (consul, etcd)
- Advanced service health checks
