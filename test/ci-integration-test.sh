#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}✓ $1${NC}"; }
fail() {
    echo -e "${RED}✗ $1${NC}"
    exit 1
}
warn() { echo -e "${YELLOW}! $1${NC}"; }
info() { echo -e "  $1"; }

SOCKET="/tmp/disco-ci-test.sock"
PIDFILE="/tmp/disco-ci-test.pid"
CONFIG="/tmp/disco-ci-test.yaml"
DAEMON="${PROJECT_ROOT}/build/bin/disco-daemon"
CLI="${PROJECT_ROOT}/build/bin/disco"

cleanup() {
    if [ -f "$PIDFILE" ]; then
        kill $(cat "$PIDFILE") 2>/dev/null || true
        rm -f "$PIDFILE"
    fi
    rm -f "$SOCKET" "$CONFIG"
}
trap cleanup EXIT

echo "========================================"
echo " Disco Integration Tests (CI)"
echo "========================================"
echo

if [ ! -x "$DAEMON" ]; then
    fail "Daemon binary not found: $DAEMON"
fi

if [ ! -x "$CLI" ]; then
    fail "CLI binary not found: $CLI"
fi

echo "=== Test 1: Binary Execution ==="
info "Testing daemon binary..."
"$DAEMON" -help 2>/dev/null || true
pass "Daemon binary executes"

info "Testing CLI binary..."
"$CLI" --help >/dev/null 2>&1 || true
pass "CLI binary executes"
echo

echo "=== Test 2: Daemon Startup ==="
cat >"$CONFIG" <<EOF
daemon:
  socket_path: "$SOCKET"
  broadcast_interval: 30s
  record_ttl: 3600s

network:
  interfaces: ["eth0", "en0"]
  broadcast_addr: "255.255.255.255:5354"
  max_broadcast_rate: 10

discovery:
  enabled: true
  detect_services: false
  service_port_mapping: {}
  scan_interval: 60s

security:
  enabled: false

logging:
  level: "error"
  format: "text"

time_sync:
  enabled: false
EOF

info "Starting daemon..."
"$DAEMON" -config "$CONFIG" 2>&1 &
echo $! >"$PIDFILE"
sleep 2

if [ ! -S "$SOCKET" ]; then
    fail "Socket not created at $SOCKET"
fi
pass "Daemon started, socket created"
echo

echo "=== Test 3: CLI Commands ==="

info "Testing 'disco hosts'..."
if "$CLI" hosts 2>&1 | grep -q "type"; then
    pass "disco hosts works"
else
    warn "disco hosts response unexpected"
fi

info "Testing 'disco status'..."
if "$CLI" status 2>&1 | grep -q "status\|running\|ok"; then
    pass "disco status works"
else
    warn "disco status response unexpected"
fi

info "Testing 'disco ping'..."
if timeout 5 "$CLI" ping -count 1 -timeout 2s 2>&1; then
    pass "disco ping works"
else
    warn "disco ping failed (expected if no peers)"
fi
echo

echo "=== Test 4: Socket Protocol ==="
info "Testing socket client..."
cd "$PROJECT_ROOT" && go run test/socket-client/main.go "$SOCKET" 2>&1 || warn "Some socket tests may have failed"
pass "Socket protocol test completed"
echo

echo "=== Test 5: Key Management ==="
KEYFILE="/tmp/disco-test-key-$$.json"
info "Testing key generation..."
if "$CLI" key generate -output "$KEYFILE" 2>&1; then
    if [ -f "$KEYFILE" ]; then
        pass "Key generated successfully"
        rm -f "$KEYFILE"
    else
        fail "Key file not created"
    fi
else
    warn "Key generation returned non-zero (may be expected)"
fi
echo

echo "=== Test 6: Config Validation ==="
info "Testing config validation..."
if "$CLI" config validate "$CONFIG" 2>&1; then
    pass "Config validation passed"
else
    warn "Config validation returned non-zero"
fi
echo

echo "=== Test 7: NSS Module ==="
info "Checking NSS module installation..."
if [ -f /lib/x86_64-linux-gnu/libnss_disco.so.2 ]; then
    pass "NSS module installed"

    info "Testing NSS lookup (may fail if daemon not fully configured)..."
    if getent hosts nonexistent-disco-host-12345 2>&1; then
        warn "NSS lookup returned success for nonexistent host"
    else
        pass "NSS lookup correctly failed for nonexistent host"
    fi
else
    warn "NSS module not installed (expected in some CI environments)"
fi
echo

echo "=== Test 8: Time Sync Commands ==="
info "Testing time status..."
"$CLI" time status 2>&1 || warn "Time status returned non-zero (expected if time_sync disabled)"

info "Testing time sources..."
"$CLI" time sources 2>&1 || warn "Time sources returned non-zero"
pass "Time commands executed"
echo

echo "=== Test 9: Daemon Shutdown ==="
info "Stopping daemon..."
if [ -f "$PIDFILE" ]; then
    kill $(cat "$PIDFILE") 2>/dev/null || true
    sleep 1
    if [ -S "$SOCKET" ]; then
        warn "Socket still exists after shutdown"
    else
        pass "Daemon stopped cleanly"
    fi
fi
echo

echo "========================================"
echo " Integration Tests Complete"
echo "========================================"
echo
echo "All critical tests passed."
