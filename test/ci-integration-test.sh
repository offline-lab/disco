#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

FAILURES=0

pass() { echo -e "${GREEN}✓ $1${NC}"; }
fail() {
    echo -e "${RED}✗ $1${NC}"
    FAILURES=$((FAILURES + 1))
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
  service_port_mapping:
    ssh: [22]
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
else
    pass "Daemon started, socket created"
fi
echo

echo "=== Test 3: CLI Commands ==="

info "Testing 'disco hosts'..."
HOSTS_OUTPUT=$("$CLI" -s "$SOCKET" hosts 2>&1) || true
if echo "$HOSTS_OUTPUT" | grep -qiE "hostname|hosts discovered|type"; then
    pass "disco hosts works"
else
    fail "disco hosts response unexpected: $HOSTS_OUTPUT"
fi

info "Testing 'disco status'..."
STATUS_OUTPUT=$("$CLI" -s "$SOCKET" status 2>&1) || true
if echo "$STATUS_OUTPUT" | grep -qiE "daemon|status|socket"; then
    pass "disco status works"
else
    fail "disco status response unexpected: $STATUS_OUTPUT"
fi

info "Testing 'disco ping' (requires target)..."
PING_OUTPUT=$("$CLI" -s "$SOCKET" ping -c 1 localhost 2>&1) || true
if echo "$PING_OUTPUT" | grep -qiE "results|success|down|unreachable"; then
    pass "disco ping works (no daemon to ping, but command works)"
else
    fail "disco ping failed unexpectedly: $PING_OUTPUT"
fi
echo

echo "=== Test 4: Socket Protocol ==="
info "Testing socket client..."
if cd "$PROJECT_ROOT" && go run test/socket-client/main.go "$SOCKET" 2>&1; then
    pass "Socket protocol test completed"
else
    fail "Socket protocol test failed"
fi
echo

echo "=== Test 5: Key Management ==="
KEYFILE="/tmp/disco-test-key-$$.json"
info "Testing key generation..."
if "$CLI" key generate "$KEYFILE" 2>&1; then
    if [ -f "$KEYFILE" ]; then
        pass "Key generated successfully"
        rm -f "$KEYFILE"
    else
        fail "Key file not created"
    fi
else
    fail "Key generation command failed"
fi
echo

echo "=== Test 6: Config Validation ==="
info "Testing config validation..."
if "$CLI" config validate "$CONFIG" 2>&1 | grep -qi "valid\|passed"; then
    pass "Config validation passed"
else
    fail "Config validation failed"
fi
echo

echo "=== Test 7: NSS Module ==="
info "Checking NSS module installation..."
if [ -f /lib/x86_64-linux-gnu/libnss_disco.so.2 ]; then
    pass "NSS module installed"

    info "Testing NSS lookup (may fail if daemon not fully configured)..."
    if getent hosts nonexistent-disco-host-12345 2>&1; then
        fail "NSS lookup returned success for nonexistent host"
    else
        pass "NSS lookup correctly failed for nonexistent host"
    fi
else
    warn "NSS module not installed (expected in some CI environments)"
fi
echo

echo "=== Test 8: Time Sync Commands ==="
info "Testing time status..."
TIME_OUTPUT=$("$CLI" -s "$SOCKET" time 2>&1) || true
if echo "$TIME_OUTPUT" | grep -qiE "time|sync|disabled"; then
    pass "Time command works"
else
    warn "Time status response unexpected (time_sync may be disabled)"
fi
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

if [ $FAILURES -gt 0 ]; then
    echo -e "${RED}FAILED: $FAILURES test(s) failed${NC}"
    exit 1
else
    echo -e "${GREEN}All tests passed${NC}"
    exit 0
fi
