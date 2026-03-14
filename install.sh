#!/bin/bash
set -e

INSTALL_DIR="/usr/local/bin"
LIB_DIR="/lib"
CONFIG_DIR="/etc/disco"
SYSTEMD_DIR="/etc/systemd/system"
NSSWITCH_FILE="/etc/nsswitch.conf"
BUILD_BIN="build/bin"
BUILD_LIB="build/lib"

echo "=== Disco Daemon Installation Script ==="
echo "Lightweight name service daemon for offline, airgapped networks"
echo

if [ "$EUID" -ne 0 ]; then
    echo "Error: Please run as root or with sudo"
    exit 1
fi

if [ ! -f "$BUILD_BIN/disco" ] || [ ! -f "$BUILD_BIN/disco-daemon" ]; then
    echo "Error: Binaries not found in $BUILD_BIN/. Run 'make' first."
    exit 1
fi

echo "Creating disco user and group..."
if ! id -u disco &>/dev/null; then
    useradd --system --no-create-home --shell /bin/false disco
    groupadd --system disco
    usermod -a -G disco disco
    echo "Created user and group: disco"
else
    echo "User disco already exists"
fi

echo "Installing binaries..."
install -m 755 "$BUILD_BIN/disco" "$INSTALL_DIR/"
echo "Installed: $INSTALL_DIR/disco (unified CLI)"

install -m 755 "$BUILD_BIN/disco-daemon" "$INSTALL_DIR/"
echo "Installed: $INSTALL_DIR/disco-daemon"

if [ -f "$BUILD_BIN/disco-gps-broadcaster" ]; then
    install -m 755 "$BUILD_BIN/disco-gps-broadcaster" "$INSTALL_DIR/"
    echo "Installed: $INSTALL_DIR/disco-gps-broadcaster"
fi

if [ -f "$BUILD_LIB/libnss_disco.so.2" ]; then
    echo "Installing NSS module..."
    install -m 644 "$BUILD_LIB/libnss_disco.so.2" "$LIB_DIR/"
    ln -sf "$LIB_DIR/libnss_disco.so.2" "$LIB_DIR/libnss_disco.so"
    ldconfig
    echo "Installed: $LIB_DIR/libnss_disco.so.2"
else
    echo "Warning: NSS module not found at $BUILD_LIB/libnss_disco.so.2 (run 'make libnss' on Linux)"
fi

echo "Creating directories..."
mkdir -p "$CONFIG_DIR"
mkdir -p "/var/lib/disco"
mkdir -p "/run"
chown -R disco:disco "$CONFIG_DIR"
chown -R disco:disco "/var/lib/disco"

if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
    echo "Installing configuration file..."
    if [ -f "config.yaml" ]; then
        install -m 644 config.yaml "$CONFIG_DIR/"
    else
        cat >"$CONFIG_DIR/config.yaml" <<EOF
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
  scan_interval: 60s
  service_port_mapping:
    www: [80, 8080]
    smtp: [25, 587]
    mail: [110, 143, 993, 995]
    xmpp: [5222, 5269]
    ftp: [21]

security:
  enabled: false
  key_path: $CONFIG_DIR/keys.json
  require_signed: false

logging:
  level: info
  format: text
  file: /var/log/disco.log
EOF
    fi
    chown root:root "$CONFIG_DIR/config.yaml"
    chmod 644 "$CONFIG_DIR/config.yaml"
    echo "Installed: $CONFIG_DIR/config.yaml"
else
    echo "Configuration file already exists: $CONFIG_DIR/config.yaml"
fi

if [ -f "libnss/disco.service" ]; then
    echo "Installing systemd service..."
    install -m 644 libnss/disco.service "$SYSTEMD_DIR/"
    systemctl daemon-reload
    echo "Installed: $SYSTEMD_DIR/disco.service"
else
    echo "Warning: disco.service not found"
fi

echo
echo "Configuring nsswitch.conf..."
if [ -f "$NSSWITCH_FILE" ]; then
    if grep -q "disco" "$NSSWITCH_FILE"; then
        echo "NSSwitch already configured for 'disco'"
    else
        echo "Updating nsswitch.conf..."
        cp "$NSSWITCH_FILE" "${NSSWITCH_FILE}.backup.$(date +%Y%m%d%H%M%S)"

        sed -i.bak 's/^hosts:.*$/hosts: files disco dns/' "$NSSWITCH_FILE"

        if [ -f "${NSSWITCH_FILE}.bak" ]; then
            rm "${NSSWITCH_FILE}.bak"
        fi

        echo "Updated: $NSSWITCH_FILE"
        echo "Backup created: ${NSSWITCH_FILE}.backup.*"
    fi
else
    echo "Warning: $NSSWITCH_FILE not found"
fi

echo "Installing logrotate configuration..."
cat >/etc/logrotate.d/disco <<EOF
/var/log/disco.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 0640 disco disco
}
EOF
echo "Installed: /etc/logrotate.d/disco"

touch /var/log/disco.log
chown disco:disco /var/log/disco.log
chmod 640 /var/log/disco.log

echo
echo "=== Installation Complete ==="
echo
echo "Next steps:"
echo "  1. Review configuration: $CONFIG_DIR/config.yaml"
echo "  2. Start daemon: systemctl start disco"
echo "  3. Enable autostart: systemctl enable disco"
echo "  4. Check status: systemctl status disco"
echo "  5. View hosts: disco hosts"
echo "  6. View services: disco services"
echo
echo "Quick reference:"
echo "  disco hosts              # List all hosts"
echo "  disco hosts <name>       # Show host details"
echo "  disco services           # List services"
echo "  disco lookup <name>      # Look up hostname"
echo "  disco status             # Show daemon status"
echo
echo "To uninstall, run: sudo bash uninstall.sh"
