#!/bin/bash
# Build debian package for arm64 (Raspberry Pi)

set -e

ARCH="arm64"
VERSION="1.0.0-1"
PACKAGE_NAME="disco_${VERSION}_${ARCH}"
PACKAGE_DIR="build/debian-package-${ARCH}"

echo "=== Building Debian Package for ARM64 ==="
echo

# Build binaries for arm64
echo "1. Building binaries for linux/arm64..."
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o build/bin/disco-daemon-linux-arm64 cmd/daemon/main.go
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o build/bin/disco-linux-arm64 cmd/disco/main.go
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o build/bin/disco-gps-broadcaster-linux-arm64 cmd/gps-broadcaster/main.go
echo "   ✓ Binaries built"

# Build NSS module for arm64 (requires cross-compiler)
echo "2. Building NSS module for arm64..."
if command -v aarch64-linux-gnu-gcc &> /dev/null; then
    aarch64-linux-gnu-gcc -fPIC -shared -o build/lib/libnss_disco.so.2-arm64 libnss/nss_disco.c
    echo "   ✓ NSS module built"
    NSS_AVAILABLE=true
else
    echo "   ⚠ aarch64-linux-gnu-gcc not found, NSS module not included"
    NSS_AVAILABLE=false
fi

# Create package directories
echo "3. Creating package structure..."
rm -rf "$PACKAGE_DIR"
mkdir -p "$PACKAGE_DIR/DEBIAN"
mkdir -p "$PACKAGE_DIR/usr/bin"
mkdir -p "$PACKAGE_DIR/lib/aarch64-linux-gnu"
mkdir -p "$PACKAGE_DIR/etc/disco"
mkdir -p "$PACKAGE_DIR/var/lib/disco"
mkdir -p "$PACKAGE_DIR/lib/systemd/system"

# Copy binaries
echo "4. Copying binaries..."
cp build/bin/disco-daemon-linux-arm64 "$PACKAGE_DIR/usr/bin/disco-daemon"
cp build/bin/disco-linux-arm64 "$PACKAGE_DIR/usr/bin/disco"
cp build/bin/disco-gps-broadcaster-linux-arm64 "$PACKAGE_DIR/usr/bin/disco-gps-broadcaster"
chmod 755 "$PACKAGE_DIR/usr/bin"/*

# Copy NSS module (if built)
if [ "$NSS_AVAILABLE" = true ]; then
    cp build/lib/libnss_disco.so.2-arm64 "$PACKAGE_DIR/lib/aarch64-linux-gnu/libnss_disco.so.2"
    echo "   ✓ NSS module included"
fi

# Copy systemd service
cp debian/disco-daemon.service "$PACKAGE_DIR/lib/systemd/system/disco.service"

# Copy config example (with discovery enabled)
cat > "$PACKAGE_DIR/etc/disco/config.yaml.example" << 'CONFEOF'
# Disco Daemon Configuration
# See: https://github.com/offline-lab/disco

daemon:
  socket_path: "/run/disco.sock"
  broadcast_interval: 5s
  record_ttl: 300s

network:
  interfaces: ["eth0"]
  broadcast_addr: "255.255.255.255:5354"
  max_broadcast_rate: 20

discovery:
  enabled: true
  detect_services: true
  service_port_mapping:
    ssh: [22]
    www: [80, 443]
    smtp: [25]
  scan_interval: 60s

health:
  passive_mode: true
  stale_threshold: 180s
  lost_threshold: 300s

security:
  enabled: false
  key_path: "/etc/disco/keys.json"
  require_signed: false

logging:
  level: "info"
  format: "text"
CONFEOF

# Create control file
cat > "$PACKAGE_DIR/DEBIAN/control" << EOF
Package: disco
Version: ${VERSION}
Section: net
Priority: optional
Architecture: ${ARCH}
Maintainer: Flip Hess <flip@offline-lab.org>
Description: Lightweight name service daemon for airgapped networks
 Disco provides automatic service discovery and name resolution across
 nodes in an offline network without requiring external DNS services.
 .
 This package contains all components: daemon, CLI, and NSS module.
 .
 Optimized for embedded systems (Raspberry Pi, etc.)
