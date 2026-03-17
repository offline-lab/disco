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
cd /path/to/disco-daemon
make
```

This builds:
- `disco-daemon` - Go daemon binary
- `libnss_disco.so.2` - C NSS module

### Option 2: Build Separately

#### Build Daemon
```bash
go build -o disco-daemon cmd/daemon/main.go
```

#### Build NSS Module
```bash
make libnss
# or
gcc -fPIC -shared -o libnss_disco.so.2 -Wl,-soname,libnss_disco.so.2 libnss/nss_disco.c
```

## Installation

### Step 1: Install Daemon

```bash
sudo install -m 755 disco-daemon /usr/local/bin/
```

### Step 2: Install NSS Module

```bash
sudo install -m 644 libnss_disco.so.2 /lib/x86_64-linux-gnu/
sudo ln -sf /lib/x86_64-linux-gnu/libnss_disco.so.2 /lib/x86_64-linux-gnu/libnss_disco.so
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

Edit `/etc/nsswitch.conf` and add `disco` to the `hosts` line:
```
hosts: files disco dns
```

Order matters:
- `files` - Check /etc/hosts first (fastest)
- `disco` - Check our NSS module
- `dns` - Fallback to DNS

### Step 4: Create Configuration Directory

```bash
sudo mkdir -p /etc/disco-daemon
```

### Step 5: Install Configuration

```bash
sudo cp config.yaml /etc/disco-daemon/
# or create custom config:
sudo vim /etc/disco-daemon/config.yaml
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
  socket_path: "/run/disco.sock"
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
  cert_path: "/etc/disco-daemon/cert.pem"
  key_path: "/etc/disco-daemon/key.pem"
  trusted_peers: "/etc/disco-daemon/trusted.pem"
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
| `socket_path must be absolute` | Use full path like `/run/disco.sock` |
| `invalid broadcast_addr` | Format must be `host:port` like `255.255.255.255:5353` |
| `scan_interval must be at least 10 seconds` | Increase scan_interval |
| `log_level invalid` | Use: debug, info, warn, error, fatal |

## Starting the Daemon

### Manual Start

```bash
# Run directly
sudo /usr/local/bin/disco-daemon -config /etc/disco-daemon/config.yaml

# Or if in PATH
sudo disco-daemon -config /etc/disco-daemon/config.yaml
```

### systemd Service

Create `/etc/systemd/system/disco-daemon.service`:
```ini
[Unit]
Description=NSS Daemon for service discovery
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/disco-daemon -config /etc/disco-daemon/config.yaml
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
sudo systemctl enable disco-daemon
sudo systemctl start disco-daemon
sudo systemctl status disco-daemon
```

### Init Scripts (sysvinit)

Create `/etc/init.d/disco-daemon`:
```bash
#!/bin/sh
### BEGIN INIT INFO
# Provides:          disco-daemon
# Required-Start:    $network $remote_fs
# Required-Stop:     $network $remote_fs
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: NSS Daemon
# Description:       NSS Daemon for service discovery
### END INIT INFO

DAEMON=/usr/local/bin/disco-daemon
CONFIG=/etc/disco-daemon/config.yaml
PIDFILE=/var/run/disco-daemon.pid

case "$1" in
    start)
        echo "Starting disco-daemon..."
        start-stop-daemon --start --quiet --pidfile $PIDFILE --exec $DAEMON -- -config $CONFIG
        ;;
    stop)
        echo "Stopping disco-daemon..."
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
sudo chmod +x /etc/init.d/disco-daemon
sudo update-rc.d disco-daemon defaults
```

## Verification

### Test NSS Module Loading

```bash
ldconfig -p | grep libnss_disco
# Should show libnss_disco.so.2 and libnss_disco.so
```

### Test nsswitch.conf

```bash
cat /etc/nsswitch.conf | grep hosts
# Should show: hosts: files daemon dns
```

### Test Socket Server

```bash
# Start daemon (if not running)
sudo disco-daemon -config /etc/disco-daemon/config.yaml &

# Check socket exists
ls -la /run/disco.sock

# Test connectivity
echo '{"type":"QUERY_BY_NAME","name":"test","request_id":"test-001"}' | nc -U /run/disco.sock
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
echo "/lib/x86_64-linux-gnu/" | sudo tee -a /etc/ld.so.conf.d/disco-daemon.conf
sudo ldconfig
```

### Socket Connection Failed

```bash
# Check permissions
ls -la /run/disco.sock

# Remove old socket
sudo rm /run/disco.sock

# Check daemon is running
ps aux | grep disco-daemon
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
disco-daemon -config /etc/disco-daemon/config.yaml -check

# Check logs
journalctl -u disco-daemon -n 50
tail -f /var/log/syslog | grep disco-daemon

# Run in foreground to see errors
sudo disco-daemon -config /etc/disco-daemon/config.yaml
```

## Uninstallation

### Remove Daemon

```bash
# Stop daemon
sudo systemctl stop disco-daemon
sudo systemctl disable disco-daemon

# Remove binary
sudo rm /usr/local/bin/disco-daemon

# Remove configuration
sudo rm -rf /etc/disco-daemon

# Remove service file
sudo rm /etc/systemd/system/disco-daemon.service
sudo systemctl daemon-reload
```

### Remove NSS Module

```bash
# Remove library
sudo rm /lib/x86_64-linux-gnu/libnss_disco.so.2
sudo rm /lib/x86_64-linux-gnu/libnss_disco.so
sudo ldconfig

# Restore nsswitch.conf
sudo cp /etc/nsswitch.conf.bak /etc/nsswitch.conf

# Or edit to remove "daemon" line
sudo vim /etc/nsswitch.conf
```

## Buildroot Integration

For embedded systems using Buildroot:

### Recipe (disco-daemon.mk)

```makefile
DISCO_DAEMON_VERSION = 0.1.0
DISCO_DAEMON_SITE = $(call github,offline-lab,disco,$(DISCO_DAEMON_VERSION))
DISCO_DAEMON_LICENSE = MIT
DISCO_DAEMON_DEPENDENCIES = host-golang host-gcc

define DISCO_DAEMON_BUILD_CMDS
	$(HOST_GO_ENV) $(MAKE) -C $(@D)
endef

define DISCO_DAEMON_INSTALL_TARGET_CMDS
	$(INSTALL) -D -m 755 $(@D)/disco-daemon $(TARGET_DIR)/usr/bin/disco-daemon
	$(INSTALL) -D -m 644 $(@D)/libnss_disco.so.2 $(TARGET_DIR)/lib/libnss_disco.so.2
	ln -sf $(TARGET_DIR)/lib/libnss_disco.so.2 $(TARGET_DIR)/lib/libnss_disco.so
endef

$(eval $(generic-package))
```

### Config Fragment

Create `package/disco-daemon/nsswitch.conf`:
```
hosts: files disco dns
```

This will be installed to `/etc/nsswitch.conf`.

## Yocto Integration

For Yocto/OpenEmbedded:

### Recipe (disco-daemon.bb)

```bitbake
SUMMARY = "NSS Daemon for service discovery"
LICENSE = "MIT"
LIC_FILES_CHKSUM = "file://LICENSE;md5=xxxx"

SRC_URI = "git://github.com/offline-lab/disco.git;branch=main \
           file://nsswitch.conf"

S = "${WORKDIR}/git"

DEPENDS = "golang-native"

inherit go-mod

do_install() {
    install -d ${D}${bindir}
    install -m 0755 ${B}/disco-daemon ${D}${bindir}/disco-daemon

    install -d ${D}${libdir}
    install -m 0644 ${B}/libnss_disco.so.2 ${D}${libdir}/libnss_disco.so.2
    ln -sf libnss_disco.so.2 ${D}${libdir}/libnss_disco.so

    install -d ${D}${sysconfdir}/disco-daemon
    install -m 0644 ${B}/config.yaml ${D}${sysconfdir}/disco-daemon/config.yaml
    install -m 0644 ${WORKDIR}/nsswitch.conf ${D}${sysconfdir}/nsswitch.conf
}

FILES_${PN} += "${bindir}/disco-daemon \
                ${libdir}/libnss_disco.so.2 \
                ${libdir}/libnss_disco.so \
                ${sysconfdir}/disco-daemon/config.yaml \
                ${sysconfdir}/nsswitch.conf"
```

## Docker Installation

### Build Image

```bash
docker build -t disco-daemon .
```

### Run Container

```bash
docker run --name disco-daemon \
  --network host \
  --cap-add=NET_ADMIN,NET_RAW \
  -v /etc/disco-daemon:/etc/disco-daemon \
  disco-daemon
```

## Next Steps

After installation:
1. Configure network interfaces
2. Test name resolution with `getent hosts`
3. Verify discovery is working (check logs, tcpdump)
4. Test actual service discovery (run a service, see if it's announced)
5. Monitor logs for errors
6. Adjust configuration as needed
