#!/bin/bash
set -e

echo "========================================"
echo "  NSS Daemon Multi-Node Test"
echo "  3 Linux VMs via OrbStack"
echo "========================================"
echo

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'
pass() { echo -e "${GREEN}✓ $1${NC}"; }
fail() { echo -e "${RED}✗ $1${NC}"; }
warn() { echo -e "${YELLOW}! $1${NC}"; }

# Build binaries
echo "1. Building Linux binaries (arm64)..."
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/nss-daemon-linux cmd/daemon/main.go
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/nss-query-linux cmd/query/main.go
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/nss-status-linux cmd/status/main.go
pass "Binaries built"

# Cleanup existing VMs
echo "2. Cleaning up old VMs..."
for name in nss-node1 nss-node2 nss-node3; do
    orb delete $name 2>/dev/null || true
done
sleep 2
pass "Cleaned"

# Create VMs
echo "3. Creating 3 Linux VMs..."
orb create debian:bookworm nss-node1 2>/dev/null &
orb create debian:bookworm nss-node2 2>/dev/null &
orb create debian:bookworm nss-node3 2>/dev/null &
wait
sleep 3
pass "VMs created"

# Get VM IPs
echo "4. Getting VM IPs..."
NODE1_IP=$(orb -m nss-node1 ip addr show eth0 | grep "inet " | awk '{print $2}' | cut -d/ -f1)
NODE2_IP=$(orb -m nss-node2 ip addr show eth0 | grep "inet " | awk '{print $2}' | cut -d/ -f1)
NODE3_IP=$(orb -m nss-node3 ip addr show eth0 | grep "inet " | awk '{print $2}' | cut -d/ -f1)
echo "   nss-node1: $NODE1_IP"
echo "   nss-node2: $NODE2_IP"
echo "   nss-node3: $NODE3_IP"

# Get broadcast address (last octet .255)
BROADCAST=$(echo $NODE1_IP | sed 's/\.[0-9]*$/.255/')
echo "   Broadcast: $BROADCAST"

# Setup function for each node
setup_node() {
    local name=$1
    local hostname=$2
    local broadcast=$3

    echo "   Setting up $hostname..."

    # Install dependencies
    orb -m $name sh -c "apt-get update -qq && apt-get install -y -qq gcc libc6-dev netcat-openbsd" 2>/dev/null

    # Copy binaries
    cat build/nss-daemon-linux | orb -m $name sh -c "cat > /tmp/nss-daemon && chmod +x /tmp/nss-daemon"
    cat build/nss-query-linux | orb -m $name sh -c "cat > /tmp/nss-query && chmod +x /tmp/nss-query"
    cat build/nss-status-linux | orb -m $name sh -c "cat > /tmp/nss-status && chmod +x /tmp/nss-status"

    # Compile NSS module
    cat libnss/nss_daemon.c | orb -m $name sh -c "cat > /tmp/nss_daemon.c && gcc -fPIC -shared -o /tmp/libnss_daemon.so.2 /tmp/nss_daemon.c"

    # Install everything
    orb -m $name sh -c "
        mv /tmp/nss-daemon /usr/local/bin/ && chmod +x /usr/local/bin/nss-daemon
        mv /tmp/nss-query /usr/local/bin/ && chmod +x /usr/local/bin/nss-query
        mv /tmp/nss-status /usr/local/bin/ && chmod +x /usr/local/bin/nss-status
        mv /tmp/libnss_daemon.so.2 /lib/aarch64-linux-gnu/
        ln -sf /lib/aarch64-linux-gnu/libnss_daemon.so.2 /lib/aarch64-linux-gnu/libnss_daemon.so
        ldconfig
        echo 'hosts: files daemon dns' > /etc/nsswitch.conf
        hostname $hostname
        mkdir -p /etc/nss-daemon /run
    "

    # Create config
    orb -m $name sh -c "cat > /etc/nss-daemon/config.yaml << ENDCONFIG
daemon:
  socket_path: \"/run/nss-daemon.sock\"
  broadcast_interval: 3s
  record_ttl: 300s
network:
  interfaces: [\"eth0\"]
  broadcast_addr: \"${broadcast}:5354\"
  max_broadcast_rate: 20
discovery:
  enabled: true
  detect_services: true
  service_port_mapping:
    ssh: [22]
    www: [80, 443, 8080]
  scan_interval: 30s
security:
  enabled: false
logging:
  level: \"info\"
  format: \"text\"
ENDCONFIG"
}

echo "5. Setting up nodes (in parallel)..."
setup_node nss-node1 node1 $BROADCAST &
setup_node nss-node2 node2 $BROADCAST &
setup_node nss-node3 node3 $BROADCAST &
wait
pass "All nodes configured"

# Start an SSH server on node2 for service discovery testing
echo "6. Starting services for discovery testing..."
orb -m nss-node2 sh -c "apt-get install -y -qq openssh-server && service ssh start" 2>/dev/null || true
orb -m nss-node3 sh -c "apt-get install -y -qq python3 && python3 -m http.server 8080 &" 2>/dev/null || true
pass "Services started"

# Start daemons
echo "7. Starting daemons on all nodes..."
orb -m nss-node1 sh -c "/usr/local/bin/nss-daemon -config /etc/nss-daemon/config.yaml 2>&1 &" || true
orb -m nss-node2 sh -c "/usr/local/bin/nss-daemon -config /etc/nss-daemon/config.yaml 2>&1 &" || true
orb -m nss-node3 sh -c "/usr/local/bin/nss-daemon -config /etc/nss-daemon/config.yaml 2>&1 &" || true

echo "   Waiting for daemons to start..."
sleep 3

# Verify daemons
echo "8. Verifying daemons..."
for node in nss-node1 nss-node2 nss-node3; do
    if orb -m $node pgrep -f nss-daemon &>/dev/null; then
        pass "$node daemon running"
    else
        fail "$node daemon NOT running"
    fi
done

# Verify sockets
echo "9. Verifying sockets..."
for node in nss-node1 nss-node2 nss-node3; do
    if orb -m $node test -S /run/nss-daemon.sock 2>/dev/null; then
        pass "$node socket ready"
    else
        fail "$node socket missing"
    fi
done

# Wait for discovery
echo "10. Waiting for discovery (15s)..."
sleep 15

# Test cross-node discovery
echo
echo "========================================"
echo "  TESTING CROSS-NODE DISCOVERY"
echo "========================================"

echo
echo "11. Testing from node1..."
echo "   Querying discovered hosts..."
orb -m nss-node1 /usr/local/bin/nss-query hosts 2>&1 | head -20 || echo "   (query failed)"

echo
echo "   Testing getent hosts node2 (should find it via NSS)..."
if orb -m nss-node1 getent hosts node2 2>&1 | grep -qE "[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+"; then
    pass "node1 discovered node2!"
    orb -m nss-node1 getent hosts node2
else
    fail "node1 did NOT discover node2"
fi

echo
echo "   Testing getent hosts node3..."
if orb -m nss-node1 getent hosts node3 2>&1 | grep -qE "[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+"; then
    pass "node1 discovered node3!"
    orb -m nss-node1 getent hosts node3
else
    fail "node1 did NOT discover node3"
fi

echo
echo "12. Testing from node2..."
echo "   Testing getent hosts node1..."
if orb -m nss-node2 getent hosts node1 2>&1 | grep -qE "[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+"; then
    pass "node2 discovered node1!"
else
    fail "node2 did NOT discover node1"
fi

echo
echo "13. Testing from node3..."
echo "   Testing getent hosts node1..."
if orb -m nss-node3 getent hosts node1 2>&1 | grep -qE "[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+"; then
    pass "node3 discovered node1!"
else
    fail "node3 did NOT discover node1"
fi

echo
echo "14. Testing reverse lookup (IP -> hostname)..."
echo "   Testing getent hosts $NODE2_IP..."
result=$(orb -m nss-node1 getent hosts $NODE2_IP 2>&1)
if echo "$result" | grep -qi "node"; then
    pass "Reverse lookup works!"
    echo "   $result"
else
    warn "Reverse lookup not working (may need peer in cache)"
fi

echo
echo "========================================"
echo "  TESTING SERVICE DISCOVERY"
echo "========================================"

echo
echo "15. Listing discovered services from node1..."
orb -m nss-node1 /usr/local/bin/nss-query services 2>&1 | head -20 || echo "   (no services or query failed)"

echo
echo "16. Listing hosts with services..."
orb -m nss-node1 /usr/local/bin/nss-query hosts-services 2>&1 | head -30 || echo "   (query failed)"

echo
echo "========================================"
echo "  TEST COMPLETE"
echo "========================================"
echo
echo "Summary:"
echo "  - 3 Linux VMs running on OrbStack"
echo "  - Each VM has NSS module installed"
echo "  - Daemons discovering via UDP broadcast"
echo
echo "To interact:"
echo "  orb -m nss-node1 bash"
echo "  orb -m nss-node1 getent hosts node2"
echo "  orb -m nss-node1 /usr/local/bin/nss-query hosts"
echo
echo "To cleanup:"
echo "  orb delete nss-node1 nss-node2 nss-node3"
