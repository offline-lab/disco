# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2026-05-24

### Fixed

- **Daemon socket server blocked startup**: `Start()` blocked forever in accept loop,
  preventing DNS server startup and graceful shutdown. Split into setup + goroutine.
- **DNS server never resolved queries**: Domain suffix stripping was broken in all
  handlers (A, TXT, CNAME). Rewrote with correct `HasSuffix`/`TrimSuffix` logic.
- **DNS SRV query failed for single-label domains**: Minimum parts check was too
  strict (`< 4` → `< 3`), rejecting valid queries like `_www._tcp.disco`.
- **DNS server leaked on shutdown**: Only tracked one server instance. Now tracks all
  bind-address servers and shuts them all down.
- **Record store race condition**: `Get`, `GetByAddr`, `List` called status-mutating
  `UpdateRecordStatus` under `RLock`. Changed to read-only `ComputeStatus`.
- **Record store service port always zero**: `GetAllRecords` didn't parse "addr:port"
  service values. Now uses `net.SplitHostPort` to populate `Port` and `Protocol`.
- **Broadcast listener goroutine leak**: `WaitGroup` was local to `Start()`, so
  `Stop()` couldn't wait for goroutines. Moved to struct field.
- **Broadcast listener panic on stop**: `Stop()` closed channels before waiting for
  goroutines, causing writes to closed channels. Reordered: close conns → wait → close chans.
- **Unsigned time messages accepted when signing required**: Extra `keyManager != nil`
  condition allowed unsigned messages through when no key manager was configured.
- **Service reachability checker misidentified protocols**: `parseProtocol` treated IP
  addresses as protocol names. Replaced with `net.SplitHostPort`-based parsing.
- **Socket server missing connection deadline**: Added 5-second deadline to prevent
  slow-client DoS on the Unix socket.
- **Socket `handleQueryListServices` broken Sscanf**: Replaced `fmt.Sscanf` (which
  can't parse "host:port") with `net.SplitHostPort` + `strconv.Atoi`.
- **Default broadcast port inconsistency**: Config default was 5353 while all other
  code used 5354. Fixed config default to 5354.
- **Ping command default port**: Was 5353, fixed to 5354.

### Changed

- Removed dead duplicate `JoinStrings` function from hosts command
- Removed redundant `--force` flag from `disco timeset` (command always forces)
- Updated daemon help text to reference current `disco` subcommands

## [1.0.0] - 2025-03-01

### Added

- **Unified CLI with Cobra framework**
  - All CLI tools merged into single `disco` command
  - Consistent command structure with subcommands
  - Colorized output with status indicators
  - JSON output support (`--json` flag)
  - Shell completion support for bash/zsh/fish
  - Input validation on all commands
  - Security hardening for `disco start` command
- Time synchronization feature for airgapped networks
  - GPS time source support via UDP broadcast
  - Multi-source validation (requires 2+ agreeing sources)
  - Clock discipline (step for large offsets, slew for small)
  - `disco time` command for monitoring sync status
  - `disco timeset` command for forced updates
  - GPS broadcasters for Raspberry Pi Zero (Go), Arduino, and ESPHome
- CI/CD pipeline with GitHub Actions
  - Automated testing with race detection
  - Cross-platform builds (linux/amd64, linux/arm64)
  - Docker image builds
- `internal/client` package for shared daemon communication
- Configuration validation for time_sync settings
- Unified man page (`disco.1.md`) documenting all commands

### Changed

- **Project renamed from nss-daemon to disco**
  - All binaries renamed: `nss-*` → `disco-*`
  - NSS module renamed: `libnss_daemon` → `libnss_disco`
  - Socket path: `/run/nss-daemon.sock` → `/run/disco.sock`
  - Config directory: `/etc/nss-daemon` → `/etc/disco`
  - Service user: `nss-daemon` → `disco`
  - systemd service: `nss-daemon.service` → `disco.service`
- **CLI consolidation (major refactor)**
  - Deleted standalone binaries: `disco-ping`, `disco-announce`, `disco-query`, `disco-status`, `disco-key`, `disco-dns`, `disco-config-validate`
  - Migrated all functionality to unified `disco` command
  - Command structure: `disco <command> [subcommand]`
  - Old 1,266-line monolithic CLI refactored to ~1,747 lines across 16 modular files
  - Binary size reduced: Multiple tools → Single 3.6MB binary
- Makefile now uses pattern rules for cleaner builds
- Makefile supports version injection via ldflags
- Dockerfile includes all disco-* tools
- Install script includes all disco-* tools
- Removed committed binaries from repository
- Documentation restructured to reflect unified CLI

### Fixed

- Command injection vulnerability in daemon launcher (now uses whitelist)
- Weak random number generation (switched to `io.ReadFull`)
- Missing input validation on user inputs
- Race condition in TimeSourceStore.GetValidSources()
- Security check inconsistency between listener and time sync service
- Duplicate test function declarations in service_force_test.go

## [0.1.0] - 2024-02-15

### Added

- Initial release (formerly nss-daemon)
- Core NSS daemon with UDP broadcast discovery
- Custom NSS module (libnss_disco.so.2) for glibc integration
- Service detection with configurable port mapping
- Rate limiting and duplicate suppression
- Optional message signing with HMAC-SHA256
- CLI tools: disco-query, disco-status, disco-key, disco-ping, disco-dns, disco-announce
- Configuration validation tool
- Docker support for multi-node testing
- Installation and uninstallation scripts
- systemd service file
