#!/bin/bash
# Simple debian package build for testing (without docker)

set -e

echo "=== Building Debian Package Structure ==="
echo

# Create package directories
PACKAGE_DIR="build/debian-package"
rm -rf "$PACKAGE_DIR"
mkdir -p "$PACKAGE_DIR/DEBIAN"
mkdir -p "$PACKAGE_DIR/usr/bin"
mkdir -p "$PACKAGE_DIR/lib"
mkdir -p "$PACKAGE_DIR/etc/disco"
mkdir -p "$PACKAGE_DIR/var/lib/disco"
mkdir -p "$PACKAGE_DIR/lib/systemd/system"

# Copy binaries
echo "Copying binaries..."
cp build/bin/disco-daemon "$PACKAGE_DIR/usr/bin/"
cp build/bin/disco "$PACKAGE_DIR/usr/bin/"
cp build/bin/disco-gps-broadcaster "$PACKAGE_DIR/usr/bin/"
chmod 755 "$PACKAGE_DIR/usr/bin"/*

# Copy NSS module (if built)
if [ -f build/lib/libnss_disco.so.2 ]; then
    cp build/lib/libnss_disco.so.2 "$PACKAGE_DIR/lib/"
fi

# Copy systemd service
cp debian/disco-daemon.service "$PACKAGE_DIR/lib/systemd/system/disco.service"

# Copy config example
cp config.yaml "$PACKAGE_DIR/etc/disco/config.yaml.example"

# Create control file
cat >"$PACKAGE_DIR/DEBIAN/control" <<'EOF'
Package: disco
Version: 1.0.0-1
Section: net
Priority: optional
Architecture: amd64
Maintainer: Flip Hess <flip@offline-lab.org>
Description: Lightweight name service daemon for airgapped networks
 Disco provides automatic service discovery and name resolution across
 nodes in an offline network without requiring external DNS services.
 .
 This package contains all components: daemon, CLI, and NSS module.

EOF

# Create postinst script
cat >"$PACKAGE_DIR/DEBIAN/postinst" <<'EOF'
#!/bin/bash
set -e

# Create disco user
if ! getent passwd disco > /dev/null; then
    adduser --system --group --home /var/lib/disco \
            --gecos "Disco daemon" --shell /bin/false disco
fi

# Create directories
mkdir -p /var/lib/disco
mkdir -p /run/disco
chown disco:disco /var/lib/disco
chown disco:disco /run/disco

# Install default config
if [ ! -f /etc/disco/config.yaml ]; then
    cp /etc/disco/config.yaml.example /etc/disco/config.yaml
    chmod 640 /etc/disco/config.yaml
    chown root:disco /etc/disco/config.yaml
fi

# Update ldconfig
ldconfig

# Remind about nsswitch.conf
if ! grep -q "disco" /etc/nsswitch.conf; then
    echo ""
    echo "Disco has been installed."
    echo ""
    echo "To enable name resolution, add 'disco' to /etc/nsswitch.conf:"
    echo "  hosts: files disco dns"
    echo ""
    echo "Then start the daemon:"
    echo "  systemctl enable disco"
    echo "  systemctl start disco"
    echo ""
fi

EOF
chmod 755 "$PACKAGE_DIR/DEBIAN/postinst"

# Create postrm script
cat >"$PACKAGE_DIR/DEBIAN/postrm" <<'EOF'
#!/bin/bash
set -e

if [ "$1" = "purge" ]; then
    # Remove user
    if getent passwd disco > /dev/null; then
        deluser --system disco || true
    fi
    
    # Remove directories
    rm -rf /var/lib/disco
    rm -rf /run/disco
    
    # Remove from nsswitch.conf
    if [ -f /etc/nsswitch.conf ]; then
        sed -i '/\bdisco\b/d' /etc/nsswitch.conf
    fi
fi

EOF
chmod 755 "$PACKAGE_DIR/DEBIAN/postrm"

# Build the package
echo
echo "Building package..."
dpkg-deb --build "$PACKAGE_DIR" "disco_1.0.0-1_amd64.deb"

echo
echo "=== Package Created ==="
ls -lh disco_1.0.0-1_amd64.deb

echo
echo "To install:"
echo "  sudo dpkg -i disco_1.0.0-1_amd64.deb"
echo
echo "To test in VM:"
echo "  multipass transfer disco_1.0.0-1_amd64.deb vm-name:/tmp/"
echo "  multipass exec vm-name -- sudo dpkg -i /tmp/disco_1.0.0-1_amd64.deb"
echo
