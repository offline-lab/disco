#!/bin/bash
# Build all debian packages with pre-compiled NSS modules

set -e

VERSION="1.0.0-1"

docker run --rm -v "$(pwd):/src" -w /src debian:bookworm bash << 'DOCKEREOF'
set -e

# Install all build dependencies
apt-get update -qq
apt-get install -y -qq \
    gcc libc6-dev \
    gcc-x86-64-linux-gnu \
    gcc-aarch64-linux-gnu \
    gcc-arm-linux-gnueabihf \
    curl

# Install Go
curl -sL https://go.dev/dl/go1.21.6.linux-amd64.tar.gz | tar -C /usr/local -xzf -
export PATH=/usr/local/go/bin:$PATH

# Prepare NSS source with correct socket path
sed 's|/run/disco.sock|/run/disco/disco.sock|g' libnss/nss_disco.c > /tmp/nss_disco.c

build_package() {
    local ARCH=$1
    local GOARCH=$2
    local CC=$3
    local LIBDIR=$4
    
    echo "=== Building ${ARCH} package ==="
    
    # Build Go binaries
    echo "Building Go binaries..."
    GOOS=linux GOARCH=${GOARCH} CGO_ENABLED=0 go build -ldflags='-s -w' -o build/bin/disco-daemon-${ARCH} cmd/daemon/main.go
    GOOS=linux GOARCH=${GOARCH} CGO_ENABLED=0 go build -ldflags='-s -w' -o build/bin/disco-${ARCH} cmd/disco/main.go
    GOOS=linux GOARCH=${GOARCH} CGO_ENABLED=0 go build -ldflags='-s -w' -o build/bin/disco-gps-broadcaster-${ARCH} cmd/gps-broadcaster/main.go
    
    # Build NSS module
    echo "Building NSS module..."
    mkdir -p build/lib
    ${CC} -fPIC -shared -o build/lib/libnss_disco.so.2-${ARCH} /tmp/nss_disco.c
    
    # Create package structure
    PACKAGE_DIR="build/debian-package-${ARCH}"
    rm -rf $PACKAGE_DIR
    mkdir -p $PACKAGE_DIR/DEBIAN
    mkdir -p $PACKAGE_DIR/usr/bin
    mkdir -p $PACKAGE_DIR/${LIBDIR}
    mkdir -p $PACKAGE_DIR/etc/disco
    mkdir -p $PACKAGE_DIR/var/lib/disco
    mkdir -p $PACKAGE_DIR/lib/systemd/system
    
    # Copy binaries
    cp build/bin/disco-daemon-${ARCH} $PACKAGE_DIR/usr/bin/disco-daemon
    cp build/bin/disco-${ARCH} $PACKAGE_DIR/usr/bin/disco
    cp build/bin/disco-gps-broadcaster-${ARCH} $PACKAGE_DIR/usr/bin/disco-gps-broadcaster
    chmod 755 $PACKAGE_DIR/usr/bin/*
    
    # Copy NSS module
    cp build/lib/libnss_disco.so.2-${ARCH} $PACKAGE_DIR/${LIBDIR}/libnss_disco.so.2
    echo "   ✓ NSS module included"
    
    # Systemd service
    cat > $PACKAGE_DIR/lib/systemd/system/disco.service << 'SERVICEEOF'
[Unit]
Description=Disco Name Service Daemon
Documentation=https://github.com/offline-lab/disco
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=disco
Group=disco
RuntimeDirectory=disco
ExecStart=/usr/bin/disco-daemon -config /etc/disco/config.yaml
Restart=on-failure
RestartSec=5
LimitNOFILE=65536
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/var/lib/disco

[Install]
WantedBy=multi-user.target
SERVICEEOF
    
    # Config example
    cat > $PACKAGE_DIR/etc/disco/config.yaml.example << 'CONFIGEOF'
daemon:
  socket_path: '/run/disco/disco.sock'
  broadcast_interval: 5s
  record_ttl: 300s

network:
  interfaces: ['eth0']
  broadcast_addr: '255.255.255.255:5354'
  max_broadcast_rate: 20

discovery:
  enabled: true
  detect_services: true
  service_port_mapping:
    ssh: [22]
    www: [80, 443]
  scan_interval: 60s

health:
  passive_mode: true
  stale_threshold: 180s
  lost_threshold: 300s

security:
  enabled: false

logging:
  level: 'info'
  format: 'text'
CONFIGEOF
    
    # Control file
    cat > $PACKAGE_DIR/DEBIAN/control << CONTROLEOF
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
 This package contains: daemon, CLI, and NSS module (pre-compiled).
 .
 Optimized for embedded systems (Raspberry Pi, etc.)
CONTROLEOF
    
    # Postinst
    cat > $PACKAGE_DIR/DEBIAN/postinst << 'POSTINSTEOF'
#!/bin/bash
set -e

if ! getent passwd disco > /dev/null; then
    adduser --system --group --home /var/lib/disco --gecos 'Disco daemon' --shell /bin/false disco
fi

mkdir -p /var/lib/disco /run/disco
chown disco:disco /var/lib/disco /run/disco

if [ ! -f /etc/disco/config.yaml ]; then
    cp /etc/disco/config.yaml.example /etc/disco/config.yaml
    chmod 640 /etc/disco/config.yaml
    chown root:disco /etc/disco/config.yaml
fi

ldconfig

if ! grep -q 'disco' /etc/nsswitch.conf 2>/dev/null; then
    echo
    echo '==========================================='
    echo 'Disco has been installed!'
    echo '==========================================='
    echo
    echo 'To enable name resolution, add to /etc/nsswitch.conf:'
    echo '  hosts: files disco dns'
    echo
    echo 'Then start the daemon:'
    echo '  systemctl enable --now disco'
    echo
fi
POSTINSTEOF
    chmod 755 $PACKAGE_DIR/DEBIAN/postinst
    
    # Build package
    dpkg-deb --build $PACKAGE_DIR disco_${VERSION}_${ARCH}.deb
    
    echo "✓ Package built: disco_${VERSION}_${ARCH}.deb"
    ls -lh disco_${VERSION}_${ARCH}.deb
    echo
}

# Build all architectures
build_package "amd64" "amd64" "x86_64-linux-gnu-gcc" "lib/x86_64-linux-gnu"
build_package "arm64" "arm64" "aarch64-linux-gnu-gcc" "lib/aarch64-linux-gnu"
build_package "armhf" "arm" "arm-linux-gnueabihf-gcc" "lib/arm-linux-gnueabihf"

echo "=== All packages built successfully ==="
ls -lh disco_${VERSION}_*.deb
DOCKEREOF
