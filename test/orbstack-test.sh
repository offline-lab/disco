#!/bin/bash
set -e

echo "========================================"
echo "  NSS Daemon Multi-Node Linux Test"
echo "  Using OrbStack"
echo "========================================"
echo

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}✓ $1${NC}"; }
fail() { echo -e "${RED}✗ $1${NC}"; }
warn() { echo -e "${YELLOW}! $1${NC}"; }

cleanup() {
    echo
    echo "Cleaning up machines..."
    for name in nss-node1 nss-node2 nss-node3; do
        orb delete $name 2>/dev/null || true
    done
}

if [ "$1" = "clean" ]; then
    cleanup
    exit 0
fi

echo "1. Checking prerequisites..."
if ! command -v orb &>/dev/null; then
    fail "orb CLI not found"
    echo "  Install OrbStack from https://orbstack.dev"
    exit 1
fi
pass "OrbStack CLI available"

echo
echo "2. Building binaries for Linux..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/nss-daemon-linux cmd/daemon/main.go
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/nss-query-linux cmd/query/main.go
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/nss-status-linux cmd/status/main.go
pass "Go binaries built"

echo
echo "3. Compiling NSS module..."
# Cross-compile NSS module (needs linux/amd64 gcc)
if command -v x86_64-linux-gnu-gcc &>/dev/null; then
    x86_64-linux-gnu-gcc -fPIC -shared -o build/libnss_daemon.so.2 libnss/nss_daemon.c
elif docker run --rm -v "$(pwd):/src" -w /src debian:bookworm-slim bash -c "apt-get update -qq && apt-get install -y -qq gcc && gcc -fPIC -shared -o build/libnss_daemon.so.2 libnss/nss_daemon.c" 2>/dev/null; then
    pass "NSS module compiled via docker"
else
    warn "Could not cross-compile NSS module, will compile in VM"
fi

echo
echo "4. Creating Linux machines..."

create_machine() {
    local name=$1
    local ip=$2
    echo "   Creating $name..."

    # Create machine
    orb create debian:bookworm $name 2>/dev/null || true

    # Wait for it to be ready
    sleep 2

    # Install dependencies
    orb -m $name sh -c "apt-get update -qq && apt-get install -y -qq gcc libc6-dev" 2>/dev/null

    # Copy binaries
    orb -m $name mkdir -p /opt/nss-daemon
    cat build/nss-daemon-linux | orb -m $name sh -c "cat > /opt/nss-daemon/nss-daemon && chmod +x /opt/nss-daemon/nss-daemon"
    cat build/nss-query-linux | orb -m $name sh -c "cat > /opt/nss-daemon/nss-query && chmod +x /opt/nss-daemon/nss-query"
    cat build/nss-status-linux | orb -m $name sh -c "cat > /opt/nss-daemon/nss-status && chmod +x /opt/nss-daemon/nss-status"

    # Copy and compile NSS module if not cross-compiled
    if [ -f build/libnss_daemon.so.2 ]; then
        cat build/libnss_daemon.so.2 | orb -m $name sh -c "cat > /tmp/libnss_daemon.so.2"
    else
        cat libnss/nss_daemon.c | orb -m $name sh -c "cat > /tmp/nss_daemon.c && gcc -fPIC -shared -o /tmp/libnss_daemon.so.2 /tmp/nss_daemon.c"
    fi

    # Install NSS module
    orb -m $name sh -c "cp /tmp/libnss_daemon.so.2 /lib/x86_64-linux-gnu/ && ln -sf /lib/x86_64-linux-gnu/libnss_daemon.so.2 /lib/x86_64-linux-gnu/libnss_daemon.so && ldconfig"

    # Install binaries to system
    orb -m $name sh -c "cp /opt/nss-daemon/nss-daemon /usr/local/bin/ && cp /opt/nss-daemon/nss-query /usr/local/bin/ && cp /opt/nss-daemon/nss-status /usr/local/bin/"

    # Configure nsswitch
    orb -m $name sh -c "echo 'hosts: files daemon dns' > /etc/nsswitch.conf"

    # Create config
    orb -m $name sh -c "mkdir -p /etc/nss-daemon && cat > /etc/nss-daemon/config.yaml << 'EOF'
daemon:
  socket_path: \"/run/nss-daemon.sock\"
  broadcast_interval: 3s
  record_ttl: 300s

network:
  interfaces: [\"eth0\"]
  broadcast_addr: \"255.255.255.255:5354\"
  max_broadcast_rate: 10

discovery:
  enabled: true
  detect_services: false
  service_port_mapping: {}
  scan_interval: 30s

security:
  enabled: false

logging:
  level: \"debug\"
  format: \"text\"
EOF"

    # Set hostname
    orb -m $name hostnamectl set-hostname $name 2>/dev/null || true

    pass "$name ready"
}

create_machine "nss-node1" "192.168.215.10"
create_machine "nss-node2" "192.168.215.11"
create_machine "nss-node3" "192.168.215.12"

echo
echo "5. Starting daemons..."
orb -m nss-node1 sh -c "/usr/local/bin/nss-daemon -config /etc/nss-daemon/config.yaml &" || true
orb -m nss-node2 sh -c "/usr/local/bin/nss-daemon -config /etc/nss-daemon/config.yaml &" || true
orb -m nss-node3 sh -c "/usr/local/bin/nss-daemon -config /etc/nss-daemon/config.yaml &" || true

echo "   Waiting for daemons..."
sleep 3

echo
echo "6. Verifying daemons..."
for node in nss-node1 nss-node2 nss-node3; do
    if orb -m $node pgrep -f nss-daemon &>/dev/null; then
        pass "$node daemon running"
    else
        fail "$node daemon NOT running"
    fi
done

echo
echo "7. Verifying sockets..."
for node in nss-node1 nss-node2 nss-node3; do
    if orb -m $node test -S /run/nss-daemon.sock 2>/dev/null; then
        pass "$node socket exists"
    else
        fail "$node socket missing"
    fi
done

echo
echo "8. Waiting for discovery (10s)..."
sleep 10

echo
echo "9. Testing nss-query on node1..."
orb -m nss-node1 /usr/local/bin/nss-query hosts 2>/dev/null || warn "nss-query returned empty"

echo
echo "10. Testing NSS integration (getent)..."
echo "   Self lookup (node1)..."
if orb -m nss-node1 getent hosts nss-node1 2>/dev/null | grep -q "."; then
    pass "getent hosts nss-node1 works"
    orb -m nss-node1 getent hosts nss-node1 2>/dev/null
else
    fail "getent hosts nss-node1 failed"
fi

echo "   Peer lookup (node2)..."
if orb -m nss-node1 getent hosts nss-node2 2>/dev/null | grep -q "."; then
    pass "getent hosts nss-node2 works (peer discovered!)"
    orb -m nss-node1 getent hosts nss-node2 2>/dev/null
else
    warn "getent hosts nss-node2 not found yet"
fi

echo "   Peer lookup (node3)..."
if orb -m nss-node1 getent hosts nss-node3 2>/dev/null | grep -q "."; then
    pass "getent hosts nss-node3 works (peer discovered!)"
    orb -m nss-node1 getent hosts nss-node3 2>/dev/null
else
    warn "getent hosts nss-node3 not found yet"
fi

echo
echo "11. Testing from node2..."
orb -m nss-node2 /usr/local/bin/nss-query hosts 2>/dev/null || true

echo
echo "========================================"
echo "  Test Complete"
echo "========================================"
echo
echo "Machines are still running. To interact:"
echo "  orb -m nss-node1 bash"
echo "  orb -m nss-node1 getent hosts nss-node2"
echo
echo "To clean up machines:"
echo "  $0 clean"
