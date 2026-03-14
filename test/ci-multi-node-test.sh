#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

pass() { echo -e "${GREEN}✓ $1${NC}"; }
fail() {
    echo -e "${RED}✗ $1${NC}"
    exit 1
}
warn() { echo -e "${YELLOW}! $1${NC}"; }
info() { echo -e "${BLUE}ℹ $1${NC}"; }

if ! command -v docker &>/dev/null; then
    echo "========================================"
    echo " Docker not available - SKIPPING"
    echo "========================================"
    echo ""
    echo "Multi-node tests require Docker."
    echo "Install Docker or run on a machine with Docker available."
    exit 0
fi

if ! docker info >/dev/null 2>&1; then
    echo "========================================"
    echo " Docker daemon not running - SKIPPING"
    echo "========================================"
    echo ""
    echo "Multi-node tests require a running Docker daemon."
    exit 0
fi

NETWORK_NAME="disco-test-net"
NODE1="disco-test-node1"
NODE2="disco-test-node2"
NODE3="disco-test-node3"

cleanup() {
    info "Cleaning up containers..."
    docker rm -f $NODE1 $NODE2 $NODE3 2>/dev/null || true
    docker network rm $NETWORK_NAME 2>/dev/null || true
}
trap cleanup EXIT

echo "========================================"
echo " Disco Multi-Node Integration Tests"
echo "========================================"
echo

DAEMON="${PROJECT_ROOT}/build/bin/disco-daemon-amd64"
CLI="${PROJECT_ROOT}/build/bin/disco-amd64"

if [ ! -x "$DAEMON" ]; then
    info "Building Linux binaries..."
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "${PROJECT_ROOT}/build/bin/disco-daemon-amd64" "${PROJECT_ROOT}/cmd/daemon"
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "${PROJECT_ROOT}/build/bin/disco-amd64" "${PROJECT_ROOT}/cmd/disco"
else
    info "Using pre-built Linux binaries"
fi
pass "Binaries ready"
echo

info "Creating Docker network..."
docker network create --driver bridge $NETWORK_NAME 2>/dev/null || true
pass "Network created"
echo

info "Creating test containers..."
for node in $NODE1 $NODE2 $NODE3; do
    docker run -d --name $node \
        --platform linux/amd64 \
        --network $NETWORK_NAME \
        --hostname $node \
        debian:bookworm \
        sleep infinity 2>/dev/null || true
done
pass "Containers created"
echo

info "Installing dependencies in containers..."
for node in $NODE1 $NODE2 $NODE3; do
    docker exec $node apt-get update -qq 2>/dev/null
    docker exec $node apt-get install -y -qq gcc libc6-dev netcat-openbsd iproute2 procps 2>/dev/null
done
pass "Dependencies installed"
echo

info "Copying binaries to containers..."
for node in $NODE1 $NODE2 $NODE3; do
    docker cp "$DAEMON" "$node:/usr/local/bin/disco-daemon"
    docker cp "$CLI" "$node:/usr/local/bin/disco"
    docker exec $node chmod +x /usr/local/bin/disco-daemon /usr/local/bin/disco

    docker cp "${PROJECT_ROOT}/libnss/nss_disco.c" "$node:/tmp/nss_disco.c"
    docker exec $node bash -c "gcc -fPIC -shared -o /lib/libnss_disco.so.2 /tmp/nss_disco.c && ldconfig"
done
pass "Binaries deployed"
echo

info "Configuring NSS in containers..."
for node in $NODE1 $NODE2 $NODE3; do
    docker exec $node bash -c "echo 'hosts: files disco dns' > /etc/nsswitch.conf"
done
pass "NSS configured"
echo

info "Getting container IPs..."
NODE1_IP=$(docker exec $NODE1 hostname -i | cut -d' ' -f1)
NODE2_IP=$(docker exec $NODE2 hostname -i | cut -d' ' -f1)
NODE3_IP=$(docker exec $NODE3 hostname -i | cut -d' ' -f1)
BROADCAST=$(echo $NODE1_IP | sed 's/\.[0-9]*$/.255/')
info "Node1 IP: $NODE1_IP"
info "Node2 IP: $NODE2_IP"
info "Node3 IP: $NODE3_IP"
info "Broadcast: $BROADCAST"
echo

info "Creating daemon configs..."
for node in $NODE1 $NODE2 $NODE3; do
    docker exec $node bash -c "mkdir -p /etc/disco /run && cat > /etc/disco/config.yaml << EOF
daemon:
  socket_path: /run/disco.sock
  broadcast_interval: 5s
  record_ttl: 60s

network:
  interfaces: [\"eth0\"]
  broadcast_addr: \"${BROADCAST}:5354\"
  max_broadcast_rate: 50

discovery:
  enabled: true
  detect_services: true
  service_port_mapping:
    ssh: [22]
  scan_interval: 10s

security:
  enabled: false

logging:
  level: \"info\"
  format: \"text\"

time_sync:
  enabled: false
EOF"
done
pass "Configs created"
echo

info "Starting daemons..."
for node in $NODE1 $NODE2 $NODE3; do
    docker exec $node bash -c "nohup /usr/local/bin/disco-daemon -config /etc/disco/config.yaml > /var/log/disco.log 2>&1 &"
done
sleep 3
pass "Daemons started"
echo

info "Verifying daemons are running..."
FAILURES=0
for node in $NODE1 $NODE2 $NODE3; do
    if docker exec $node pgrep -f disco-daemon >/dev/null 2>&1; then
        pass "$node daemon running"
    else
        warn "$node daemon NOT running"
        docker exec $node cat /var/log/disco.log 2>/dev/null || warn "No log file"
        FAILURES=$((FAILURES + 1))
    fi
done

if [ $FAILURES -gt 0 ]; then
    fail "$FAILURES daemon(s) failed to start"
else
    pass "All daemons running"
fi
echo

info "Waiting for discovery (10s)..."
sleep 10
echo

echo "=== Testing Cross-Node Discovery ==="
echo

info "Querying hosts on $NODE1..."
docker exec $NODE1 /usr/local/bin/disco hosts 2>&1 || warn "Query failed"
echo

info "Testing NSS lookup: $NODE1 -> $NODE2..."
if docker exec $NODE1 getent hosts $NODE2 2>&1 | grep -qE "[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+"; then
    pass "Node1 discovered Node2 via NSS!"
    docker exec $NODE1 getent hosts $NODE2
else
    fail "Node1 did NOT discover Node2"
fi
echo

info "Testing NSS lookup: $NODE1 -> $NODE3..."
if docker exec $NODE1 getent hosts $NODE3 2>&1 | grep -qE "[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+"; then
    pass "Node1 discovered Node3 via NSS!"
    docker exec $NODE1 getent hosts $NODE3
else
    fail "Node1 did NOT discover Node3"
fi
echo

info "Testing NSS lookup: $NODE2 -> $NODE1..."
if docker exec $NODE2 getent hosts $NODE1 2>&1 | grep -qE "[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+"; then
    pass "Node2 discovered Node1 via NSS!"
else
    fail "Node2 did NOT discover Node1"
fi
echo

info "Testing reverse lookup on $NODE1 ($NODE2_IP)..."
result=$(docker exec $NODE1 getent hosts $NODE2_IP 2>&1 || true)
if echo "$result" | grep -qi "node"; then
    pass "Reverse lookup works"
else
    warn "Reverse lookup may not work (expected in some cases)"
fi
echo

info "Testing service discovery..."
docker exec $NODE1 /usr/local/bin/disco services 2>&1 || warn "Service query failed"
echo

echo "========================================"
echo " Multi-Node Integration Tests Complete"
echo "========================================"
