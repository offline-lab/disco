# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
