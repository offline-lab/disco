#!/bin/bash
set -e

INSTALL_DIR="/usr/local/bin"
LIB_DIR="/lib"
CONFIG_DIR="/etc/nss-daemon"
SYSTEMD_DIR="/etc/systemd/system"
NSSWITCH_FILE="/etc/nsswitch.conf"

echo "=== NSS Daemon Installation Script ==="
echo "Lightweight name service daemon for offline, airgapped networks"
echo

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: Please run as root or with sudo"
    exit 1
fi

# Check for required binaries
if [ ! -f "nss-daemon" ] || [ ! -f "nss-status" ] || [ ! -f "nss-key" ]; then
    echo "Error: nss-daemon, nss-status, and nss-key binaries not found. Run 'make' first."
    exit 1
fi

# Check for optional binaries
MISSING=""
for bin in nss-query nss-ping nss-dns nss-config-validate; do
    if [ ! -f "$bin" ]; then
        MISSING="$MISSING $bin"
    fi
done

if [ -n "$MISSING" ]; then
    echo "Warning: Optional binaries not found:$MISSING"
    echo "         Run 'make' to build all binaries."
fi

# Create nss-daemon user and group
echo "Creating nss-daemon user and group..."
if ! id -u nss-daemon &>/dev/null; then
    useradd --system --no-create-home --shell /bin/false nss-daemon
    groupadd --system nss-daemon
    usermod -a -G nss-daemon nss-daemon
    echo "Created user and group: nss-daemon"
else
    echo "User nss-daemon already exists"
fi

# Install binaries
echo "Installing binaries..."
install -m 755 nss-daemon "$INSTALL_DIR/"
install -m 755 nss-status "$INSTALL_DIR/"
install -m 755 nss-query "$INSTALL_DIR/"
install -m 755 nss-key "$INSTALL_DIR/"
install -m 755 nss-ping "$INSTALL_DIR/"
install -m 755 nss-dns "$INSTALL_DIR/"
install -m 755 nss-config-validate "$INSTALL_DIR/"
echo "Installed: $INSTALL_DIR/nss-daemon"
echo "Installed: $INSTALL_DIR/nss-status"
echo "Installed: $INSTALL_DIR/nss-query"
echo "Installed: $INSTALL_DIR/nss-key"
echo "Installed: $INSTALL_DIR/nss-ping"
echo "Installed: $INSTALL_DIR/nss-dns"
echo "Installed: $INSTALL_DIR/nss-config-validate"

# Install NSS module if available
if [ -f "libnss_daemon.so.2" ]; then
    echo "Installing NSS module..."
    install -m 644 libnss_daemon.so.2 "$LIB_DIR/"
    ln -sf "$LIB_DIR/libnss_daemon.so.2" "$LIB_DIR/libnss_daemon.so"
    ldconfig
    echo "Installed: $LIB_DIR/libnss_daemon.so.2"
else
    echo "Warning: NSS module not found (expected on Linux)"
fi

# Create directories
echo "Creating directories..."
mkdir -p "$CONFIG_DIR"
mkdir -p "/var/lib/nss-daemon"
mkdir -p "/run/nss-daemon"
chown -R nss-daemon:nss-daemon "$CONFIG_DIR"
chown -R nss-daemon:nss-daemon "/var/lib/nss-daemon"
chown -R nss-daemon:nss-daemon "/run/nss-daemon"

# Install configuration if not present
if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
    echo "Installing configuration file..."
    if [ -f "config.yaml" ]; then
        install -m 644 config.yaml "$CONFIG_DIR/"
    else
        cat >"$CONFIG_DIR/config.yaml" <<EOF
daemon:
  socket_path: /run/nss-daemon.sock
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
  file: /var/log/nss-daemon.log
EOF
    fi
    chown root:root "$CONFIG_DIR/config.yaml"
    chmod 644 "$CONFIG_DIR/config.yaml"
    echo "Installed: $CONFIG_DIR/config.yaml"
else
    echo "Configuration file already exists: $CONFIG_DIR/config.yaml"
fi

# Install systemd service if available
if [ -f "nss-daemon.service" ]; then
    echo "Installing systemd service..."
    install -m 644 nss-daemon.service "$SYSTEMD_DIR/"
    systemctl daemon-reload
    echo "Installed: $SYSTEMD_DIR/nss-daemon.service"
else
    echo "Warning: nss-daemon.service not found"
fi

# Configure nsswitch.conf
echo
echo "Configuring nsswitch.conf..."
if [ -f "$NSSWITCH_FILE" ]; then
    if grep -q "daemon" "$NSSWITCH_FILE"; then
        echo "NSSwitch already configured for 'daemon'"
    else
        echo "Updating nsswitch.conf..."
        cp "$NSSWITCH_FILE" "${NSSWITCH_FILE}.backup.$(date +%Y%m%d%H%M%S)"

        # Add 'daemon' to hosts line
        sed -i.bak 's/^hosts:.*$/hosts: files daemon dns/' "$NSSWITCH_FILE"

        # Restore from backup if sed created it
        if [ -f "${NSSWITCH_FILE}.bak" ]; then
            rm "${NSSWITCH_FILE}.bak"
        fi

        echo "Updated: $NSSWITCH_FILE"
        echo "Backup created: ${NSSWITCH_FILE}.backup.*"
    fi
else
    echo "Warning: $NSSWITCH_FILE not found"
fi

# Install logrotate config
echo "Installing logrotate configuration..."
cat >/etc/logrotate.d/nss-daemon <<EOF
/var/log/nss-daemon.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 0640 nss-daemon nss-daemon
}
EOF
echo "Installed: /etc/logrotate.d/nss-daemon"

# Set up log file
touch /var/log/nss-daemon.log
chown nss-daemon:nss-daemon /var/log/nss-daemon.log
chmod 640 /var/log/nss-daemon.log

echo
echo "=== Installation Complete ==="
echo
echo "Next steps:"
echo "  1. Review configuration: $CONFIG_DIR/config.yaml"
echo "  2. Start daemon: systemctl start nss-daemon"
echo "  3. Enable autostart: systemctl enable nss-daemon"
echo "  4. Check status: systemctl status nss-daemon"
echo "  5. View health: nss-status health"
echo
echo "To uninstall, run: sudo bash uninstall.sh"
