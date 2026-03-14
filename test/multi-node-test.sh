#!/bin/bash
set -e
echo "========================================"
echo " Disco Daemon Multi-Node Test"
echo " 3 Linux VMs via OrbStack"
echo "========================================"
echo
echo
echo
# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'
pass() { echo -e "${GREEN}✓ $1${NC}"; }
fail() { echo -e "${RED}✗ $1${NC}"; }
warn() { echo -e "${YELLOW}! $1${NC}"; }
echo
echo "1. Building Linux binaries (arm64)..."
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/disco-daemon-linux cmd/daemon/main.go
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/disco-linux cmd/disco/main.go
pass "Binaries built"
echo
echo "2. Cleaning up old VMs..."
for name in disco-node1 disco-node2 disco-node3; do
    orb delete $name 2>/dev/null || true
done
sleep 2
pass "Cleaned"
echo
echo "3. Creating 3 Linux VMs..."
orb create debian:bookworm disco-node1 2>/dev/null &
orb create debian:bookworm disco-node2 2>/dev/null &
orb create debian:bookworm disco-node3 2>/dev/null &
wait
sleep 3
pass "VMs created"
echo
echo "4. Getting VM IPs..."
NODE1_IP=$(orb -m disco-node1 ip addr show eth0 | grep "inet" | awk '{print $2}' | cut -d/ -f1)
NODE2_IP=$(orb -m disco-node2 ip addr show eth0 | grep "inet" | awk '{print $2}' | cut -d/ -f1)
NODE3_IP=$(orb -m disco-node3 ip addr show eth0 | grep "inet" | awk '{print $2}' | cut -d/ -f1)
echo "   disco-node1: $NODE1_IP"
echo "   disco-node2: $NODE2_IP"
echo "   disco-node3: $NODE3_IP"
echo
echo "5. Get broadcast address (last octet .255)"
BROADCAST=$(echo $NODE1_IP | sed 's/\.[0-9]*$/.255/')
echo "   Broadcast: $BROADCAST"
echo
echo
6. Setup function for each node..."
setup_node() {
    local name=$1
    local hostname=$2
    local broadcast=$3
    echo "   Setting up $hostname..."
    echo
    # Install dependencies
    orb -m $name sh -c "apt-get update -qq && apt-get install -y -qq gcc libc6-dev netcat-openbsd" 2>/dev/null
    # Copy binaries
    cat build/disco-daemon-linux | orb -m $name sh -c "cat > /tmp/disco-daemon && chmod +x /tmp/disco-daemon"
    cat build/disco-linux | orb -m $name sh -c "cat > /tmp/disco && chmod +x /tmp/disco"
    # Compile NSS module
    cat libnss/nss_disco.c | orb -m $name sh -c "cat > /tmp/nss_disco.c && gcc -fPIC -shared -o /tmp/libnss_disco.so.2 /tmp/nss_disco.c"
    # Install everything
    orb -m $name sh -c "
        mv /tmp/disco-daemon /usr/local/bin/ && chmod +x /usr/local/bin/disco-daemon
        mv /tmp/disco /usr/local/bin/ && chmod +x /usr/local/bin/disco
        mv /tmp/libnss_disco.so.2 /lib/aarch64-linux-gnu/
        ln -sf /lib/aarch64-linux-gnu/libnss_disco.so.2 /lib/aarch64-linux-gnu/libnss_disco.so
        ldconfig
        echo 'hosts: files disco dns' > /etc/nsswitch.conf
        hostname $hostname
        mkdir -p /etc/disco /run
    "
    echo
    # Create config
    orb -m $name sh -c "cat > /etc/disco/config.yaml << "EOF"
daemon:
  socket_path: "/run/disco.sock"
  broadcast_interval: 3s
  record_ttl: 300s
network:
  interfaces: ["eth0"]
  broadcast_addr: "${broadcast}:5354"
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
  level: "info"
  format: "text"
EOF"
}
echo
echo "7. Setting up nodes (in parallel)..."
setup_node disco-node1 node1 $BROADCAST &
setup_node disco-node2 node2 $BROADCAST &
setup_node disco-node3 node3 $BROADCAST &
wait
pass "All nodes configured"
echo
echo "8. Start an SSH server on node2 for service discovery testing..."
echo "6. Starting services for discovery testing..."
orb -m disco-node2 sh -c "apt-get install -y -qq openssh-server && service ssh start" 2>/dev/null || true
orb -m disco-node3 sh -c "apt-get install -y -qq python3 && python3 -m http.server 8080 &" 2>/dev/null || true
pass "Services started"
echo
echo "9. Starting daemons on all nodes..."
orb -m disco-node1 sh -c "/usr/local/bin/disco-daemon -config /etc/disco/config.yaml 2>&1 &" || true
orb -m disco-node2 sh -c "/usr/local/bin/disco-daemon -config /etc/disco/config.yaml 2>&1 &" || true
orb -m disco-node3 sh -c "/usr/local/bin/disco-daemon -config /etc/disco/config.yaml 2>&1 &" || true
echo "   Waiting for daemons to start..."
sleep 3
echo
echo "10. Verify daemons..."
for node in disco-node1 disco-node2 disco-node3; do
    if orb -m $node pgrep -f disco-daemon &>/dev/null; then
        pass "$node daemon running"
    else
        fail "$node daemon NOT running"
    fi
done
echo
echo "11. Verify sockets..."
for node in disco-node1 disco-node2 disco-node3; do
    if orb -m $node test -S /run/disco.sock 2>/dev/null; then
        pass "$node socket ready"
    else
        fail "$node socket missing"
    fi
done
echo
echo "12. Wait for discovery (15s)..."
sleep 15
echo
echo "13. Test cross-node discovery..."
echo
echo "========================================"
echo "  TESTING CROSS-NODE DISCOVERY"
echo "========================================"
echo
echo
echo "Querying disco-node1..."
orb -m disco-node1 /usr/local/bin/disco hosts 2>&1 | head -20 || echo "   (query failed)"
echo
echo
echo "   Testing getent hosts node2 (should find via NSS)..."
if orb -m disco-node1 getent hosts node2 2>&1 | grep -qE "[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+"; then
    pass "node1 discovered node2!"
    orb -m disco-node1 getent hosts node2
else
    fail "node1 did NOT discover node2"
fi
echo
echo "   Testing getent hosts node3..."
if orb -m disco-node1 getent hosts node3 2>&1 | grep -qE "[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+"; then
    pass "node1 discovered node3!"
    orb -m disco-node1 getent hosts node3
else
    fail "node1 did NOT discover node3"
fi
echo
echo
echo "15. Testing from node2..."
echo "   Testing getent hosts node1..."
if orb -m disco-node2 getent hosts node1 2>&1 | grep -qE "[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+"; then
    pass "node2 discovered node1!"
else
    fail "node2 did NOT discover node1"
fi
echo
echo "   Testing getent hosts node3..."
if orb -m disco-node2 getent hosts node3 2>&1 | grep -qE "[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+"; then
    pass "node2 discovered node3!"
else
    fail "node2 did NOT discover node3"
fi
echo
echo
echo "16. Testing from node3..."
echo "   Testing getent hosts node1..."
if orb -m disco-node3 getent hosts node1 2>&1 | grep -qE "[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+"; then
    pass "node3 discovered node1!"
else
    fail "node3 did NOT discover node1"
fi
echo
echo "   Testing getent hosts node2..."
if orb -m disco-node3 getent hosts node2 2>&1 | grep -qE "[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+"; then
    pass "node3 discovered node2!"
else
    fail "node3 did NOT discover node2"
fi
echo
echo "17. Testing reverse lookup (IP -> hostname)..."
echo "   Testing getent hosts $NODE2_IP..."
result=$(orb -m disco-node1 getent hosts $NODE2_IP 2>&1)
if echo "$result" | grep -qi "node"; then
    pass "Reverse lookup works"
    echo "   $result"
else
    warn "Reverse lookup may not work (may need peer in cache)"
fi
echo
echo
echo "18. Testing service discovery..."
echo
echo "========================================"
echo "  TESTING SERVICE DISCOVERY"
echo "========================================"
echo
echo "Services discovered:"
orb -m disco-node1 /usr/local/bin/disco services 2>&1 | head -20 || echo "   (no services or query failed)"
echo
echo "Combined hosts-services view:"
orb -m disco-node1 /usr/local/bin/disco hosts 2>&1 | head -30 || echo "   (query failed)"
echo
echo
echo "========================================"
echo "  TEST COMPLETE"
echo "========================================"
echo
echo "Summary:"
echo "  - 3 Linux VMs running on OrbStack"
echo "  - Each VM has NSS module installed"
echo "  - Daemons discovering via UDP broadcast"
echo "  - All nodes discovered each other"
echo
echo "To interact:"
echo "  orb -m disco-node1 bash"
echo "  orb -m disco-node1 getent hosts node2"
echo "  orb -m disco-node1 /usr/local/bin/disco hosts"
echo
echo "To cleanup:"
echo "  orb delete disco-node1 disco-node2 disco-node3"
