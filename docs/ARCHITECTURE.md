# Disco Architecture

**Version**: 1.0.0-rc1
**Last Updated**: 2026-03-01

---

## Overview

Disco is a lightweight name service daemon for offline, airgapped emergency networks. It provides automatic service discovery and name resolution across nodes without requiring external DNS services or internet connectivity.

### Design Goals

1. **Minimal resources** - Runs on battery/solar-powered embedded devices (RPi Zero 2W, 512MB RAM)
2. **Zero configuration** - Nodes discover each other automatically
3. **Native integration** - Uses NSS module for seamless Linux integration
4. **Offline-first** - Designed for airgapped environments with no internet access
5. **Power efficient** - Minimal CPU usage, no active health checks

---

## Components

### 1. disco-daemon (Go Binary, 5.8MB)

**Purpose**: Main discovery and name resolution service

**Responsibilities**:
- UDP broadcast listener (port 5354)
- Unix socket server (NSS queries)
- Service detection and announcement
- DNS server (optional, port 53)
- Time synchronization service (optional)
- Record cache management

**Resource Usage**:
- Memory: <10MB typical
- CPU: <5% idle, <20% peak
- Network: <1 KB/sec per node

**Configuration**: `/etc/disco/config.yaml`

**Socket**: `/run/disco.sock` (Unix domain socket)

### 2. disco (CLI Tool, 3.6MB)

**Purpose**: Unified command-line interface for administration

**Commands**:
```bash
disco hosts [list|show|forget|mark-lost]  # Host management
disco services [list|show]                # Service discovery
disco lookup <hostname>                   # Name resolution
disco status                               # Daemon status
disco start [flags]                        # Start daemon
disco config validate <file>               # Validate config
disco key [generate|show|add-trusted]      # Key management
disco time                                 # Time sync status
disco timeset                              # Force time update
disco ping <target>                        # Network diagnostics
disco announce [flags]                     # Manual announcements
```

**Design**:
- **No daemon code** - Only queries daemon via Unix socket
- **Lightweight** - Separate binary avoids memory bloat in CLI
- **Secure** - Input validation, command injection prevention

### 3. disco-gps-broadcaster (Go Binary, 2.5MB)

**Purpose**: GPS time source for time synchronization

**Responsibilities**:
- Read GPS time from serial device
- Broadcast TIME_ANNOUNCE messages
- Multi-source time validation

**Protocol**: UDP broadcast on port 5354

**Alternatives**:
- Arduino implementation (see `gps-broadcaster/arduino/`)
- ESPHome implementation (see `gps-broadcaster/esphome/`)

### 4. libnss_disco.so.2 (C Library)

**Purpose**: Native Linux integration via NSS

**Functions**:
- `_nss_disco_gethostbyname_r` - Resolve hostname to IP
- `_nss_disco_gethostbyname2_r` - Resolve hostname (IPv4/IPv6)
- `_nss_disco_gethostbyaddr_r` - Reverse DNS lookup

**Integration**:
```bash
# /etc/nsswitch.conf
hosts: files disco dns
```

**Communication**: Unix domain socket to disco-daemon

---

## Data Flow

### Name Resolution Flow

```
Application (curl, ssh, etc.)
         |
         | getaddrinfo("web1")
         v
    glibc NSS
         |
         | loads libnss_disco.so.2
         v
  NSS Module (C)
         |
         | Unix socket: /run/disco.sock
         v
  disco-daemon (Go)
         |
         | Query in-memory cache
         v
    Return IP: 192.168.1.10
```

### Discovery Protocol

```
Node A                          Node B
  |                               |
  | UDP Broadcast (port 5354)    |
  |----------------------------->|
  | {type: "announce",           |
  |  hostname: "web1",           |
  |  addresses: ["10.0.0.1"],    |
  |  services: {"www": 80}}      |
  |                               |
  |                        Cache update
  |                        TTL: 3600s
  |                               |
  |<-----------------------------|
  | UDP Broadcast (announce)     |
  |        from Node B            |
  |                               |
 Cache update                   |
 TTL: 3600s                      |
```

### Service Detection Flow

```
disco-daemon
     |
     | Scan local ports
     v
  Port 80 open?
     |
     | Yes
     v
  Map to service: "www"
     |
     | Add to announcement
     v
  Broadcast: {"www": 80}
```

---

## Network Protocol

### Broadcast Message Format

**Port**: UDP 5354
**Address**: 255.255.255.255 (configurable)

**Message Types**:

1. **ANNOUNCE** - Node announcement
```json
{
  "type": "ANNOUNCE",
  "message_id": "announce-1234567890",
  "timestamp": 1708123456,
  "hostname": "web1",
  "addresses": ["192.168.1.10", "10.0.0.1"],
  "services": {
    "www": 80,
    "smtp": 25
  },
  "ttl": 3600,
  "signature": "..." // Optional
}
```

2. **TIME_ANNOUNCE** - GPS time broadcast
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
  },
  "signature": "..."
}
```

### NSS Query Protocol

**Transport**: Unix domain socket (`/run/disco.sock`)
**Format**: JSON

**Query**:
```json
{
  "type": "QUERY_BY_NAME",
  "request_id": "query-123456",
  "name": "web1"
}
```

**Response**:
```json
{
  "type": "SUCCESS",
  "request_id": "query-123456",
  "hosts": [{
    "hostname": "web1",
    "addresses": ["192.168.1.10"],
    "status": "healthy",
    "services": {"www": 80},
    "is_static": false,
    "last_seen_ago": "2m",
    "expires_in": "58m"
  }]
}
```

---

## Deployment Patterns

### Pattern 1: Minimal (Headless Nodes)

**Use Case**: Embedded devices, minimal administration

**Components**:
- disco-daemon only
- libnss_disco.so.2

**Installation**:
```bash
sudo install -m 755 disco-daemon /usr/local/bin/
sudo install -m 644 libnss_disco.so.2 /lib/
echo "hosts: files disco dns" >> /etc/nsswitch.conf
systemctl start disco
```

**Resource Usage**: ~10MB RAM, <5% CPU

### Pattern 2: Full (Management Nodes)

**Use Case**: Administration workstations, monitoring

**Components**:
- disco-daemon
- disco (CLI)
- libnss_disco.so.2
- disco-gps-broadcaster (if time sync needed)

**Installation**:
```bash
make && sudo make install
disco start -config /etc/disco/config.yaml
```

**Resource Usage**: ~15MB RAM, <10% CPU

### Pattern 3: GPS Time Source

**Use Case**: Networks requiring precise time synchronization

**Components**:
- disco-gps-broadcaster on one node
- disco-daemon on all nodes (with time_sync enabled)

**Configuration**:
```yaml
time_sync:
  enabled: true
  min_sources: 2
  require_signed: true
```

---

## Configuration

### Minimal Config

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
  enabled: false

logging:
  level: info
  format: text
```

### Full Config

```yaml
daemon:
  socket_path: /run/disco.sock
  broadcast_interval: 30s
  record_ttl: 3600s
  grace_period: 60s

network:
  broadcast_addr: 255.255.255.255:5354
  max_broadcast_rate: 10
  max_connections: 100

discovery:
  enabled: true
  detect_services: true
  scan_interval: 60s
  service_port_mapping:
    www: [80, 443, 8080]
    smtp: [25, 587]
    mail: [110, 143, 993, 995]

security:
  enabled: true
  key_path: /etc/disco/keys.json
  require_signed: true
  max_message_age: 300s

dns:
  enabled: true
  port: 53
  domain: disco
  bind_addresses: ["0.0.0.0"]

time_sync:
  enabled: true
  min_sources: 2
  max_source_spread: 100ms
  max_stale_age: 30s
  step_threshold: 128ms
  slew_threshold: 500us
  poll_interval: 60s
  require_signed: true
  allow_step_backward: false

logging:
  level: info
  format: json
  file: /var/log/disco.log
```

---

## Failure Modes

### Daemon Not Running

**Symptom**: Name resolution fails
**Detection**: Socket file missing
**Fallback**: NSS continues to next source (dns)
**Recovery**: Auto-restart via systemd

### Network Partition

**Symptom**: Nodes not discovered
**Detection**: Records expire
**Behavior**: Mark as "lost", eventually removed
**Recovery**: Auto-discovery on network restore

### Cache Expiration

**Symptom**: All records expire
**Detection**: Empty host list
**Behavior**: Returns NOTFOUND
**Recovery**: Wait for next broadcast (30s default)

### Time Sync Failure

**Symptom**: Clock drift
**Detection**: No TIME_ANNOUNCE messages
**Behavior**: Logs warning, continues operation
**Recovery**: Manual time set or GPS restore

---

## Monitoring

### Health Checks

**Daemon Status**:
```bash
disco status
```

**Host Health**:
```bash
disco hosts --json | jq '.[] | select(.status!="healthy")'
```

**Time Sync**:
```bash
disco time
```

### Logging

**Text Format**:
```
2026-03-01T10:30:00Z INFO Received announcement from web1
2026-03-01T10:30:05Z DEBUG Processing query for host: mail1
2026-03-01T10:30:10Z WARN Host expired: old-node
```

**JSON Format**:
```json
{"level":"info","ts":1708123456,"msg":"Received announcement","hostname":"web1"}
```

---

## Limitations

### Current Limitations

1. **Single broadcast domain** - No routing between subnets
2. **No persistence** - Cache lost on daemon restart
3. **Fixed TTL** - No dynamic adjustment
4. **IPv4 only** - No IPv6 support
5. **No load balancing** - Single IP per hostname

### Known Issues

1. **Large networks** (>100 nodes) may need rate limit tuning
2. **High churn** (frequent node changes) increases CPU usage
3. **Time sync** requires GPS hardware or broadcaster

### Not Supported

- Multi-interface announcements
- DNSSEC
- Dynamic TTL adjustment
- Health checking (passive only)
- Web UI

---

## Development

### Building from Source

```bash
# Clone
git clone https://github.com/offline-lab/disco
cd disco

# Build all
make

# Test
go test ./...

# Install
sudo make install
```

### Contributing

1. Fork repository
2. Create feature branch
3. Add tests (target 70%+ coverage)
4. Submit pull request

### Code Structure

```
internal/
├── config/      - Configuration parsing and validation
├── daemon/      - Core daemon logic, socket server
├── discovery/   - Broadcast protocol, rate limiting
├── dns/         - Optional DNS server
├── logging/     - Structured logging
├── nss/         - NSS protocol definitions
├── security/    - Message signing/verification
├── service/     - Service detection
└── timesync/    - Time synchronization
```

---

## References

- [NSS Module Development](https://www.gnu.org/software/libc/manual/html_node/NSS-Modules.html)
- [mDNS/DNS-SD](https://tools.ietf.org/html/rfc6762)
- [HMAC-SHA256](https://tools.ietf.org/html/rfc2104)

---

**Architecture by**: Flip Hess
**AI Assistance**: GLM-4.7 and GLM-5 from [z.ai](https://z.ai)
