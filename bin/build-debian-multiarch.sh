#!/bin/bash
# Build debian packages for multiple architectures using Docker
# Includes pre-compiled NSS modules

set -e

VERSION="1.0.0-1"

build_arch() {
    local ARCH=$1
    local GOARCH=$2
    local NSS_ARCH=$3
    local LIBDIR=$4
    
    echo "=== Building for ${ARCH} ==="
    
    docker run --rm -v "$(pwd):/src" -w /src debian:bookworm bash -c "
        set -e
        
        # Install build dependencies
        apt-get update -qq
        apt-get install -y -qq gcc libc6-dev curl
        
        # Install Go
        curl -sL https://go.dev/dl/go1.21.6.linux-amd64.tar.gz | tar -C /usr/local -xzf -
        export PATH=/usr/local/go/bin:\$PATH
        
        # Build binaries
        echo 'Building binaries...'
        GOOS=linux GOARCH=${GOARCH} CGO_ENABLED=0 go build -ldflags='-s -w' -o build/bin/disco-daemon-${ARCH} cmd/daemon/main.go
        GOOS=linux GOARCH=${GOARCH} CGO_ENABLED=0 go build -ldflags='-s -w' -o build/bin/disco-${ARCH} cmd/disco/main.go
        GOOS=linux GOARCH=${GOARCH} CGO_ENABLED=0 go build -ldflags='-s -w' -o build/bin/disco-gps-broadcaster-${ARCH} cmd/gps-broadcaster/main.go
        
        # Build NSS module with correct socket path
        echo 'Building NSS module...'
        mkdir -p build/lib
        sed 's|/run/disco.sock|/run/disco/disco.sock|g' libnss/nss_disco.c > /tmp/nss_disco.c
        ${NSS_ARCH}-linux-gnu-gcc -fPIC -shared -o build/lib/libnss_disco.so.2-${ARCH} /tmp/nss_disco.c
        
        # Create package structure
        PACKAGE_DIR='build/debian-package-${ARCH}'
        rm -rf \$PACKAGE_DIR
        mkdir -p \$PACKAGE_DIR/DEBIAN
        mkdir -p \$PACKAGE_DIR/usr/bin
        mkdir -p \$PACKAGE_DIR/${LIBDIR}
        mkdir -p \$PACKAGE_DIR/etc/disco
        mkdir -p \$PACKAGE_DIR/var/lib/disco
        mkdir -p \$PACKAGE_DIR/lib/systemd/system
        
        # Copy binaries
        cp build/bin/disco-daemon-${ARCH} \$PACKAGE_DIR/usr/bin/disco-daemon
        cp build/bin/disco-${ARCH} \$PACKAGE_DIR/usr/bin/disco
        cp build/bin/disco-gps-broadcaster-${ARCH} \$PACKAGE_DIR/usr/bin/disco-gps-broadcaster
        chmod 755 \$PACKAGE_DIR/usr/bin/*
        
        # Copy NSS module
        cp build/lib/libnss_disco.so.2-${ARCH} \$PACKAGE_DIR/${LIBDIR}/libnss_disco.so.2
        echo '   ✓ NSS module included'
        
        # Copy systemd service with RuntimeDirectory
        cat > \$PACKAGE_DIR/lib/systemd/system/disco.service << 'EOFSERVICE'
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

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/var/lib/disco

[Install]
WantedBy=multi-user.target
EOFSERVICE
        
        # Create config example with correct socket path
        cat > \$PACKAGE_DIR/etc/disco/config.yaml.example << 'EOFCONFIG'
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
EOFCONFIG
        
        # Create control file
        cat > \$PACKAGE_DIR/DEBIAN/control << EOFCONTROL
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
EOFCONTROL
        
        # Create postinst
        cat > \$PACKAGE_DIR/DEBIAN/postinst << 'EOFPOST'
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

# Update ldconfig for NSS module
ldconfig

if ! grep -q 'disco' /etc/nsswitch.conf; then
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
EOFPOST
        chmod 755 \$PACKAGE_DIR/DEBIAN/postinst
        
        # Build package
        dpkg-deb --build \$PACKAGE_DIR disco_${VERSION}_${ARCH}.deb
        
        echo 'Package built:'
        ls -lh disco_${VERSION}_${ARCH}.deb
    "
}

# Build for multiple architectures
if [ "$1" = "amd64" ] || [ "$1" = "" ]; then
    build_arch "amd64" "amd64" "x86_64" "lib/x86_64-linux-gnu"
fi

if [ "$1" = "arm64" ] || [ "$1" = "" ]; then
    build_arch "arm64" "arm64" "aarch64" "lib/aarch64-linux-gnu"
fi

if [ "$1" = "armhf" ] || [ "$1" = "" ]; then
    build_arch "armhf" "arm" "arm-linux-gnueabihf" "lib/arm-linux-gnueabihf"
fi

echo
echo "=== All packages built ==="
ls -lh disco_${VERSION}_*.deb
