# NSS Daemon - Implementation Tasks

## Recently Fixed

### Priority 0: NSS Module - PARTIALLY FIXED
- [x] Fixed missing includes (<arpa/inet.h> for inet_pton/inet_ntop)
- [x] Rewrote JSON parsing to be safer and simpler
- [x] Fixed memory leaks (free'ing strdup'ed addresses)
- [x] Added proper error handling and bounds checking
- [ ] Test compilation on Linux system - STILL NEEDED

### Priority 1: Service Announcement - FIXED
- [x] Fixed updateServiceAnnouncements() - goroutine now stops properly
- [x] Fixed service announcement to broadcast on schedule
- [x] Services are now properly added to announcer and broadcast
- [x] Added proper graceful shutdown

### Priority 2: Rate Limiting - FIXED
- [x] Implemented token bucket rate limiting
- [x] Applied rate limiting to announcer
- [x] Prevents broadcast storms

### Priority 3: Duplicate Suppression - FIXED
- [x] Implemented message deduplication
- [x] Added duplicate filter to listener
- [x] Prevents processing same message multiple times

### Priority 4: Socket Connection Limits - FIXED
- [x] Added semaphore for connection limiting (max 100 concurrent)
- [x] Prevents DOS attacks
- [x] Properly releases semaphore on connection close

### Priority 5: Configuration Validation - ADDED
- [x] Implemented comprehensive configuration validation
- [x] Added validation for daemon settings (socket path, intervals, TTL)
- [x] Added validation for network settings (broadcast addr, rate limits)
- [x] Added validation for discovery settings (service mappings, port ranges)
- [x] Added validation for security settings (cert/key paths)
- [x] Added validation for logging settings (log levels, file paths)

### Priority 6: Structured Logging - ADDED
- [x] Implemented structured logging framework
- [x] Support for text and JSON formats
- [x] Configurable log levels (DEBUG, INFO, WARN, ERROR, FATAL)
- [x] Support for log files or stdout
- [x] Contextual fields for all log messages

### Priority 7: Installation Documentation - ADDED
- [x] Created comprehensive INSTALL.md guide
- [x] Documented manual and systemd installation
- [x] Added troubleshooting section
- [x] Included Buildroot/Yocto integration examples
- [x] Docker installation instructions
- [x] Verification steps

## Current Blocking Issues

### Priority 0: Docker Multi-Node - STILL BROKEN
- [ ] Fix Docker networking for broadcast support (macvlan needs specific interface)
- [ ] Create working Docker test setup
- [ ] Test multi-node discovery works

## Pending Tasks (After Docker Fixes)

### Phase 3 (Discovery - Complete)
- [ ] Detect network interfaces and bind to them
- [ ] Handle interface changes dynamically
- [ ] Parse announcements with validation
- [ ] Update local cache from broadcasts safely
- [ ] Handle record conflicts
- [ ] Implement conflict resolution (newer timestamp wins)

### Phase 4 (Service Detection - Complete)
- [ ] Detect new services and announce immediately
- [ ] Handle service lifecycle (up/down/changed)
- [ ] Implement service cleanup
- [ ] Add service health checks
- [ ] Remove dead services after N failures
- [ ] Implement graceful degradation
- [ ] Add retry logic with backoff

### Phase 5: Security - NOT STARTED
- [ ] Signature generation/verification (ed25519)
- [ ] Certificate/key management
- [ ] Replay attack prevention (nonce + timestamp)
- [ ] Broadcast message validation
- [ ] Implement peer authentication
- [ ] Add trust management (TOFU)
- [ ] Handle key rotation
- [ ] Validate message sources

### Phase 6: Integration - NOT STARTED
- [ ] Installation scripts (with validation)
- [ ] systemd service file (with dependency management)
- [ ] NSS module installation to /lib/ (with ldconfig)
- [ ] nsswitch.conf management (backup/restore)
- [ ] Configuration examples and validation
- [ ] Add structured logging framework
- [ ] Implement metrics (Prometheus)
- [ ] Create health endpoint (/healthz)
- [ ] Add status command (CLI tool)

### Phase 7: Testing - PARTIAL
- [x] Unit tests (Go daemon) - some tests pass
- [ ] Unit tests (NSS module) - needs Linux testing
- [ ] Integration tests (multi-node) - blocked by Docker setup
- [ ] Performance benchmarks
- [ ] Security testing

### Phase 8: Observability - NOT STARTED
- [ ] Add request/response logging
- [ ] Add discovery metrics (nodes discovered, messages sent/received)
- [ ] Add cache metrics (hit rate, size, expiration)
- [ ] Add service metrics (detected, failed, announced)
- [ ] Add performance metrics (latency, throughput)

## Next Immediate Steps

1. **Test NSS module on Linux** - Verify it compiles and works
2. **Fix Docker setup** - Create working multi-node test environment
3. **Implement multi-interface support** - Bind to all interfaces
4. **Add message validation** - Validate JSON, IPs, timestamps
5. **Implement security features** - Signatures, authentication
6. **Add error logging** - Structured logging with levels

## Work Completed

- Basic Go daemon structure
- Unix socket server with connection limiting (max 100)
- In-memory record store with TTL
- UDP broadcast announcement with rate limiting
- Broadcast message listener with duplicate suppression
- Basic service port detection
- C NSS module with safer JSON parsing
- Fixed service announcement to actually broadcast
- Fixed graceful shutdown (no goroutine leaks)
- Makefile for building both components
- Basic unit tests (passing)
- Docker setup (needs network fixes)

## Known Limitations

- Docker networking doesn't support broadcast properly (needs workarounds)
- NSS module needs testing on Linux
- No multi-interface support yet
- No message validation yet
- No security features at all
- Tests don't work cross-platform (Linux-only for NSS)
- Limited logging and observability
