#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COMPOSE_FILE="${SCRIPT_DIR}/docker-compose.test.yml"

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

if ! command -v docker &>/dev/null || ! command -v docker-compose &>/dev/null && ! docker compose version &>/dev/null; then
    echo "Docker/Docker Compose not available - SKIPPING"
    exit 0
fi

if ! docker info >/dev/null 2>&1; then
    echo "Docker daemon not running - SKIPPING"
    exit 0
fi

DAEMON="${PROJECT_ROOT}/build/bin/disco-daemon-amd64"
CLI="${PROJECT_ROOT}/build/bin/disco-amd64"

if [ ! -x "$DAEMON" ]; then
    info "Building Linux binaries..."
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "${PROJECT_ROOT}/build/bin/disco-daemon-amd64" "${PROJECT_ROOT}/cmd/daemon"
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "${PROJECT_ROOT}/build/bin/disco-amd64" "${PROJECT_ROOT}/cmd/disco"
fi
pass "Binaries ready"

cleanup() {
    info "Cleaning up..."
    docker compose -f "$COMPOSE_FILE" down --remove-orphans 2>/dev/null || true
}
trap cleanup EXIT

echo "========================================"
echo " Disco Multi-Node Integration Tests"
echo "========================================"
echo

info "Starting containers with docker-compose..."
docker compose -f "$COMPOSE_FILE" up -d --wait
pass "Containers started"
echo

NODES="disco-node1 disco-node2 disco-node3"

info "Installing dependencies..."
for node in $NODES; do
    docker exec $node apt-get update -qq 2>/dev/null
    docker exec $node apt-get install -y -qq gcc libc6-dev netcat-openbsd iproute2 procps 2>/dev/null
done
pass "Dependencies installed"
echo

info "Copying binaries and building NSS module..."
for node in $NODES; do
    docker cp "$DAEMON" "$node:/usr/local/bin/disco-daemon"
    docker cp "$CLI" "$node:/usr/local/bin/disco"
    docker exec $node chmod +x /usr/local/bin/disco-daemon /usr/local/bin/disco
    docker cp "${PROJECT_ROOT}/libnss/nss_disco.c" "$node:/tmp/nss_disco.c"
    docker exec $node bash -c "gcc -fPIC -shared -o /lib/libnss_disco.so.2 /tmp/nss_disco.c && ldconfig"
done
pass "Binaries deployed"
echo

info "Configuring NSS..."
for node in $NODES; do
    docker exec $node bash -c "echo 'hosts: files disco dns' > /etc/nsswitch.conf"
done
pass "NSS configured"
echo

NODE1_IP=$(docker exec disco-node1 hostname -i | cut -d' ' -f1)
BROADCAST=$(echo $NODE1_IP | sed 's/\.[0-9]*$/.255/')
info "Broadcast: $BROADCAST"

info "Creating daemon configs..."
for node in $NODES; do
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
    test: [8080]
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
for node in $NODES; do
    docker exec $node bash -c "nohup /usr/local/bin/disco-daemon -config /etc/disco/config.yaml > /var/log/disco.log 2>&1 &"
done
sleep 3

FAILURES=0
for node in $NODES; do
    if docker exec $node pgrep -f disco-daemon >/dev/null 2>&1; then
        pass "$node daemon running"
    else
        warn "$node daemon NOT running"
        docker exec $node cat /var/log/disco.log 2>/dev/null || true
        FAILURES=$((FAILURES + 1))
    fi
done
[ $FAILURES -gt 0 ] && fail "$FAILURES daemon(s) failed to start"
pass "All daemons running"
echo

info "Waiting for discovery (10s)..."
sleep 10
echo

echo "=== Testing Cross-Node Discovery ==="
echo

info "Querying hosts on disco-node1..."
docker exec disco-node1 /usr/local/bin/disco hosts 2>&1
echo

info "Testing NSS lookup: node1 -> node2..."
if docker exec disco-node1 getent hosts disco-node2 2>&1 | grep -qE "[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+"; then
    pass "Node1 discovered Node2 via NSS!"
    docker exec disco-node1 getent hosts disco-node2
else
    fail "Node1 did NOT discover Node2"
fi
echo

info "Testing NSS lookup: node1 -> node3..."
if docker exec disco-node1 getent hosts disco-node3 2>&1 | grep -qE "[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+"; then
    pass "Node1 discovered Node3 via NSS!"
else
    fail "Node1 did NOT discover Node3"
fi
echo

info "Testing NSS lookup: node2 -> node1..."
if docker exec disco-node2 getent hosts disco-node1 2>&1 | grep -qE "[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+"; then
    pass "Node2 discovered Node1 via NSS!"
else
    fail "Node2 did NOT discover Node1"
fi
echo

NODE2_IP=$(docker exec disco-node2 hostname -i | cut -d' ' -f1)
info "Testing reverse lookup on node1 ($NODE2_IP)..."
result=$(docker exec disco-node1 getent hosts $NODE2_IP 2>&1 || true)
if echo "$result" | grep -qi "node"; then
    pass "Reverse lookup works"
else
    warn "Reverse lookup may not work (expected in some cases)"
fi
echo

info "Starting test service listener on disco-node2 (port 8080)..."
docker exec -d disco-node2 bash -c "while true; do nc -l -p 8080; done"
sleep 3
if docker exec disco-node2 pgrep -f "nc -l" >/dev/null 2>&1; then
    pass "Test listener started"
else
    warn "Test listener may not have started"
fi
echo

info "Waiting for service discovery (scan + announcement + broadcast ~45s)..."
sleep 45
echo

info "Testing service discovery on disco-node1..."
SERVICES=$(docker exec disco-node1 /usr/local/bin/disco services 2>&1)
echo "$SERVICES"
if echo "$SERVICES" | grep -q "test"; then
    pass "Service discovery working - test service found!"
else
    warn "Service not found - checking hosts..."
    docker exec disco-node1 /usr/local/bin/disco hosts 2>&1 || true
fi
echo

echo "========================================"
echo " Multi-Node Integration Tests Complete"
echo "========================================"
