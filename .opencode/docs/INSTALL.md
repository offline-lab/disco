# Installation Guide

## Prerequisites

### Linux System
- Linux distribution with glibc (e.g., Debian, Ubuntu, Alpine, etc.)
- Go 1.21+ for building the daemon
- GCC for building the NSS module
- Root or sudo access for installation

### Tools Required
```bash
# Check prerequisites
go version
gcc --version
make --version
```

## Building

### Option 1: Build Everything
```bash
cd /path/to/nss-daemon
make
```

This builds:
- `nss-daemon` - Go daemon binary
- `libnss_daemon.so.2` - C NSS module

### Option 2: Build Separately

#### Build Daemon
```bash
go build -o nss-daemon cmd/daemon/main.go
```

#### Build NSS Module
```bash
make libnss
# or
gcc -fPIC -shared -o libnss_daemon.so.2 -Wl,-soname,libnss_daemon.so.2 libnss/nss_daemon.c
```

## Installation

### Step 1: Install Daemon

```bash
sudo install -m 755 nss-daemon /usr/local/bin/
```

### Step 2: Install NSS Module

```bash
sudo install -m 644 libnss_daemon.so.2 /lib/x86_64-linux-gnu/
sudo ln -sf /lib/x86_64-linux-gnu/libnss_daemon.so.2 /lib/x86_64-linux-gnu/libnss_daemon.so
sudo ldconfig
```

**Note:** The library path may vary by system:
- Debian/Ubuntu: `/lib/x86_64-linux-gnu/`
- Alpine: `/lib/`
- Other systems: Check `ldconfig -p | grep libnss`

### Step 3: Configure NSSwitch

Backup current config:
```bash
sudo cp /etc/nsswitch.conf /etc/nsswitch.conf.bak
```

Edit `/etc/nsswitch.conf` and add `daemon` to the `hosts` line:
```
hosts: files daemon dns
```

Order matters:
- `files` - Check /etc/hosts first (fastest)
- `daemon` - Check our NSS module
- `dns` - Fallback to DNS

### Step 4: Create Configuration Directory

```bash
sudo mkdir -p /etc/nss-daemon
```

### Step 5: Install Configuration

```bash
sudo cp config.yaml /etc/nss-daemon/
# or create custom config:
sudo nano /etc/nss-daemon/config.yaml
```

### Step 6: Create Runtime Directory

```bash
sudo mkdir -p /run
sudo chmod 1777 /run
```

## Configuration

### Example Configuration

```yaml
daemon:
  socket_path: "/run/nss-daemon.sock"
  broadcast_interval: 30s
  record_ttl: 3600s

network:
  interfaces: ["eth0", "wlan0"]
  broadcast_addr: "255.255.255.255:5353"
  max_broadcast_rate: 10

discovery:
  enabled: true
  detect_services: true
  service_port_mapping:
    www: [80, 443]
    smtp: [25, 587]
    mail: [143, 993]
    xmpp: [5222, 5269]
  scan_interval: 60s

security:
  enabled: false
  cert_path: "/etc/nss-daemon/cert.pem"
  key_path: "/etc/nss-daemon/key.pem"
  trusted_peers: "/etc/nss-daemon/trusted.pem"
  require_signed: false

logging:
  level: "info"
  format: "text"
  file: ""
```

### Configuration Validation

The daemon validates configuration on startup. Common issues:

| Error | Solution |
|-------|----------|
| `socket_path must be absolute` | Use full path like `/run/nss-daemon.sock` |
| `invalid broadcast_addr` | Format must be `host:port` like `255.255.255.255:5353` |
| `scan_interval must be at least 10 seconds` | Increase scan_interval |
| `log_level invalid` | Use: debug, info, warn, error, fatal |

## Starting the Daemon

### Manual Start

```bash
# Run directly
sudo /usr/local/bin/nss-daemon -config /etc/nss-daemon/config.yaml

# Or if in PATH
sudo nss-daemon -config /etc/nss-daemon/config.yaml
```

### systemd Service

Create `/etc/systemd/system/nss-daemon.service`:
```ini
[Unit]
Description=NSS Daemon for service discovery
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/nss-daemon -config /etc/nss-daemon/config.yaml
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable nss-daemon
sudo systemctl start nss-daemon
sudo systemctl status nss-daemon
```

### Init Scripts (sysvinit)

Create `/etc/init.d/nss-daemon`:
```bash
#!/bin/sh
### BEGIN INIT INFO
# Provides:          nss-daemon
# Required-Start:    $network $remote_fs
# Required-Stop:     $network $remote_fs
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: NSS Daemon
# Description:       NSS Daemon for service discovery
### END INIT INFO

DAEMON=/usr/local/bin/nss-daemon
CONFIG=/etc/nss-daemon/config.yaml
PIDFILE=/var/run/nss-daemon.pid

case "$1" in
    start)
        echo "Starting nss-daemon..."
        start-stop-daemon --start --quiet --pidfile $PIDFILE --exec $DAEMON -- -config $CONFIG
        ;;
    stop)
        echo "Stopping nss-daemon..."
        start-stop-daemon --stop --quiet --pidfile $PIDFILE --exec $DAEMON
        ;;
    restart)
        $0 stop
        $0 start
        ;;
    status)
        status_of_proc -p $PIDFILE $DAEMON && exit 0 || exit $?
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status}"
        exit 1
        ;;
esac
```

Enable:
```bash
sudo chmod +x /etc/init.d/nss-daemon
sudo update-rc.d nss-daemon defaults
```

## Verification

### Test NSS Module Loading

```bash
ldconfig -p | grep libnss_daemon
# Should show libnss_daemon.so.2 and libnss_daemon.so
```

### Test nsswitch.conf

```bash
cat /etc/nsswitch.conf | grep hosts
# Should show: hosts: files daemon dns
```

### Test Socket Server

```bash
# Start daemon (if not running)
sudo nss-daemon -config /etc/nss-daemon/config.yaml &

# Check socket exists
ls -la /run/nss-daemon.sock

# Test connectivity
echo '{"type":"QUERY_BY_NAME","name":"test","request_id":"test-001"}' | nc -U /run/nss-daemon.sock
```

### Test Name Resolution

```bash
# Get all hosts
getent hosts

# Query specific host
getent hosts web1

# Or use standard tools
ping web1
curl http://web1
```

### Test Discovery

```bash
# Listen for broadcasts
sudo tcpdump -i any -n udp port 5353
```

## Troubleshooting

### NSS Module Not Found

```bash
# Check if installed
ldconfig -p | grep libnss

# Check library paths
echo "/lib/x86_64-linux-gnu/" | sudo tee -a /etc/ld.so.conf.d/nss-daemon.conf
sudo ldconfig
```

### Socket Connection Failed

```bash
# Check permissions
ls -la /run/nss-daemon.sock

# Remove old socket
sudo rm /run/nss-daemon.sock

# Check daemon is running
ps aux | grep nss-daemon
```

### Name Resolution Not Working

```bash
# Check NSS configuration
getent hosts <hostname>

# Enable NSS debugging
export NSS_DEBUG=yes
getent hosts <hostname>

# Check nsswitch.conf order
cat /etc/nsswitch.conf

# Try with NSS bypass
nslookup <hostname>
```

### Discovery Not Working

```bash
# Check firewall
sudo iptables -L -n | grep 5353

# Allow broadcast
sudo iptables -I INPUT -p udp --dport 5353 -j ACCEPT
sudo iptables -I OUTPUT -p udp --sport 5353 -j ACCEPT

# Check MULTICAST on interface
ip addr show eth0
# Look for MULTICAST flag
```

### Daemon Won't Start

```bash
# Check configuration validity
nss-daemon -config /etc/nss-daemon/config.yaml -check

# Check logs
journalctl -u nss-daemon -n 50
tail -f /var/log/syslog | grep nss-daemon

# Run in foreground to see errors
sudo nss-daemon -config /etc/nss-daemon/config.yaml
```

## Uninstallation

### Remove Daemon

```bash
# Stop daemon
sudo systemctl stop nss-daemon
sudo systemctl disable nss-daemon

# Remove binary
sudo rm /usr/local/bin/nss-daemon

# Remove configuration
sudo rm -rf /etc/nss-daemon

# Remove service file
sudo rm /etc/systemd/system/nss-daemon.service
sudo systemctl daemon-reload
```

### Remove NSS Module

```bash
# Remove library
sudo rm /lib/x86_64-linux-gnu/libnss_daemon.so.2
sudo rm /lib/x86_64-linux-gnu/libnss_daemon.so
sudo ldconfig

# Restore nsswitch.conf
sudo cp /etc/nsswitch.conf.bak /etc/nsswitch.conf

# Or edit to remove "daemon" line
sudo nano /etc/nsswitch.conf
```

## Buildroot Integration

For embedded systems using Buildroot:

### Recipe (nss-daemon.mk)

```makefile
NSS_DAEMON_VERSION = 0.1.0
NSS_DAEMON_SITE = $(call github,flip,nss-daemon,$(NSS_DAEMON_VERSION))
NSS_DAEMON_LICENSE = MIT
NSS_DAEMON_DEPENDENCIES = host-golang host-gcc

define NSS_DAEMON_BUILD_CMDS
	$(HOST_GO_ENV) $(MAKE) -C $(@D)
endef

define NSS_DAEMON_INSTALL_TARGET_CMDS
	$(INSTALL) -D -m 755 $(@D)/nss-daemon $(TARGET_DIR)/usr/bin/nss-daemon
	$(INSTALL) -D -m 644 $(@D)/libnss_daemon.so.2 $(TARGET_DIR)/lib/libnss_daemon.so.2
	ln -sf libnss_daemon.so.2 $(TARGET_DIR)/lib/libnss_daemon.so
endef

$(eval $(generic-package))
```

### Config Fragment

Create `package/nss-daemon/nsswitch.conf`:
```
hosts: files daemon dns
```

This will be installed to `/etc/nsswitch.conf`.

## Yocto Integration

For Yocto/OpenEmbedded:

### Recipe (nss-daemon.bb)

```bitbake
SUMMARY = "NSS Daemon for service discovery"
LICENSE = "MIT"
LIC_FILES_CHKSUM = "file://LICENSE;md5=xxxx"

SRC_URI = "git://github.com/flip/nss-daemon.git;branch=main \
           file://nsswitch.conf"

S = "${WORKDIR}/git"

DEPENDS = "golang-native"

inherit go-mod

do_install() {
    install -d ${D}${bindir}
    install -m 0755 ${B}/nss-daemon ${D}${bindir}/nss-daemon

    install -d ${D}${libdir}
    install -m 0644 ${B}/libnss_daemon.so.2 ${D}${libdir}/libnss_daemon.so.2
    ln -sf libnss_daemon.so.2 ${D}${libdir}/libnss_daemon.so

    install -d ${D}${sysconfdir}/nss-daemon
    install -m 0644 ${B}/config.yaml ${D}${sysconfdir}/nss-daemon/config.yaml
    install -m 0644 ${WORKDIR}/nsswitch.conf ${D}${sysconfdir}/nsswitch.conf
}

FILES_${PN} += "${bindir}/nss-daemon \
                ${libdir}/libnss_daemon.so.2 \
                ${libdir}/libnss_daemon.so \
                ${sysconfdir}/nss-daemon/config.yaml \
                ${sysconfdir}/nsswitch.conf"
```

## Docker Installation

### Build Image

```bash
docker build -t nss-daemon .
```

### Run Container

```bash
docker run --name nss-daemon \
  --network host \
  --cap-add=NET_ADMIN,NET_RAW \
  -v /etc/nss-daemon:/etc/nss-daemon \
  nss-daemon
```

## Next Steps

After installation:
1. Configure network interfaces
2. Test name resolution with `getent hosts`
3. Verify discovery is working (check logs, tcpdump)
4. Test actual service discovery (run a service, see if it's announced)
5. Monitor logs for errors
6. Adjust configuration as needed
