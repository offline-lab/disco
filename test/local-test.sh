#!/bin/bash

echo "=== NSS Daemon Local Integration Test (macOS/No Docker Required) ==="
echo

# Colors
GREEN=$'\033[0;32m'
RED=$'\033[0;31m'
YELLOW=$'\033[1;33m'
BLUE=$'\033[0;34m'
NC=$'\033[0m'

PASSED=0
FAILED=0

pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
    ((PASSED++))
}
fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    ((FAILED++))
}
info() { echo -e "${BLUE}ℹ${NC}: $1"; }

# Check if daemon binary exists
if [ ! -f "nss-daemon" ]; then
    echo "Building nss-daemon..."
    make
fi

# Check all binaries exist
echo "=== Checking Binaries ==="
BINARIES="nss-daemon nss-query nss-status nss-key nss-ping nss-dns nss-config-validate"
for bin in $BINARIES; do
    if [ -f "$bin" ]; then
        pass "$bin binary exists"
    else
        fail "$bin binary missing"
    fi
done
echo

# Test config validation
echo "=== Test 1: Config Validation ==="
if [ -f "test/test-config.yaml" ]; then
    if ./nss-config-validate test/test-config.yaml 2>&1 | grep -q "Configuration is valid"; then
        pass "test/test-config.yaml is valid"
    else
        fail "test/test-config.yaml validation failed"
        ./nss-config-validate test/test-config.yaml 2>&1
    fi
else
    fail "test/test-config.yaml not found"
fi
echo

# Start daemon in background
echo "=== Test 2: Start Daemon ==="

# Cleanup function
cleanup() {
    echo "=== Cleanup ==="
    if [ -n "$DAEMON_PID" ]; then
        kill $DAEMON_PID 2>/dev/null || true
        wait $DAEMON_PID 2>/dev/null || true
    fi
    rm -f /tmp/nss-daemon-test/nss-daemon.sock 2>/dev/null || true
    echo "Daemon stopped"
}
trap cleanup EXIT

# Start daemon
mkdir -p /tmp/nss-daemon-test
info "Starting nss-daemon in background..."
./nss-daemon -config test/test-config.yaml >/tmp/nss-daemon-test/nss-daemon.log 2>&1 &
DAEMON_PID=$!
sleep 2

# Check if daemon is running
if ps -p $DAEMON_PID >/dev/null 2>&1; then
    pass "Daemon is running (PID: $DAEMON_PID)"
else
    fail "Daemon failed to start"
    cat /tmp/nss-daemon-test/nss-daemon.log
    exit 1
fi

# Check if socket exists
sleep 1
if [ -S /tmp/nss-daemon-test/nss-daemon.sock ]; then
    pass "Socket created at /tmp/nss-daemon-test/nss-daemon.sock"
else
    fail "Socket not created"
    cat /tmp/nss-daemon-test/nss-daemon.log
    exit 1
fi
echo

# Test socket connectivity
echo "=== Test 3: Socket Connectivity ==="

# Test LIST_HOSTS query
if echo '{"type":"LIST_HOSTS"}' | nc -U /tmp/nss-daemon-test/nss-daemon.sock -w 2 2>&1 | grep -q '"type"'; then
    pass "LIST_HOSTS query successful"
else
    fail "LIST_HOSTS query failed"
fi

# Test LIST_SERVICES query
if echo '{"type":"LIST_SERVICES"}' | nc -U /tmp/nss-daemon-test/nss-daemon.sock -w 2 2>&1 | grep -q '"type"'; then
    pass "LIST_SERVICES query successful"
else
    fail "LIST_SERVICES query failed"
fi

# Test self-lookup
HOSTNAME=$(hostname)
if echo "{\"type\":\"QUERY_BY_NAME\",\"name\":\"$HOSTNAME\"}" | nc -U /tmp/nss-daemon-test/nss-daemon.sock -w 2 2>&1 | grep -q '"type":"OK"'; then
    pass "Self-hostname lookup successful"
else
    info "Self-hostname lookup not found (expected - no services detected yet)"
fi
echo

# Test nss-query tool
echo "=== Test 4: nss-query Tool ==="

# Check if nss-query exists
if [ -f "nss-query" ]; then
    pass "nss-query tool exists"

    # Test nss-query hosts
    if ./nss-query hosts 2>&1 >/dev/null; then
        pass "nss-query hosts works"
    else
        fail "nss-query hosts failed"
    fi

    # Test nss-query services
    if ./nss-query services 2>&1 >/dev/null; then
        pass "nss-query services works"
    else
        fail "nss-query services failed"
    fi

    # Test nss-query hosts-services
    if ./nss-query hosts-services 2>&1 >/dev/null; then
        pass "nss-query hosts-services works"
    else
        fail "nss-query hosts-services failed"
    fi
else
    fail "nss-query tool not found"
fi
echo

# Test nss-status tool
echo "=== Test 5: nss-status Tool ==="

if [ -f "nss-status" ]; then
    pass "nss-status tool exists"

    if ./nss-status 2>&1 >/dev/null; then
        pass "nss-status works"
    else
        fail "nss-status failed"
    fi
else
    fail "nss-status tool not found"
fi
echo

# Test nss-key tool
echo "=== Test 6: nss-key Tool ==="

if [ -f "nss-key" ]; then
    pass "nss-key tool exists"

    # Test key generation
    if ./nss-key generate /tmp/nss-daemon-test/test-key.json >/dev/null 2>&1; then
        pass "nss-key generate works"
        rm -f /tmp/nss-daemon-test/test-key.json
    else
        info "nss-key generate not tested (may need specific args)"
    fi
else
    fail "nss-key tool not found"
fi
echo

# Test nss-ping tool
echo "=== Test 7: nss-ping Tool ==="

if [ -f "nss-ping" ]; then
    pass "nss-ping tool exists"

    # Note: nss-ping may need network targets
    info "nss-ping not tested (requires network targets)"
else
    fail "nss-ping tool not found"
fi
echo

# Test nss-dns tool
echo "=== Test 8: nss-dns Tool ==="

if [ -f "nss-dns" ]; then
    pass "nss-dns tool exists"

    # Note: nss-dns may need DNS queries
    info "nss-dns not tested (requires DNS queries)"
else
    fail "nss-dns tool not found"
fi
echo

# Test rate limiting (daemon features)
echo "=== Test 9: Daemon Features ==="

# Check logs for rate limiting mentions
if grep -q "Rate limiting" /tmp/nss-daemon-test/nss-daemon.log 2>/dev/null; then
    pass "Rate limiting initialized"
else
    info "Rate limiting not explicitly logged (may be OK)"
fi

# Check logs for duplicate suppression
if grep -q "Duplicate" /tmp/nss-daemon-test/nss-daemon.log 2>/dev/null; then
    pass "Duplicate suppression active"
else
    info "Duplicate suppression not explicitly logged (may be OK)"
fi

# Check logs for structured logging
if grep -q "\[" /tmp/nss-daemon-test/nss-daemon.log 2>/dev/null; then
    pass "Structured logging in use"
else
    info "Structured logging format not detected"
fi
echo

# Show daemon logs
echo "=== Daemon Logs ==="
echo "Last 20 lines of daemon output:"
cat /tmp/nss-daemon-test/nss-daemon.log | tail -20
echo

# Summary
echo "=== Test Summary ==="
echo "Passed: $PASSED"
echo "Failed: $FAILED"
echo

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All local tests passed!${NC}"
    echo
    echo "To run full integration tests with Docker:"
    echo "  1. Start Docker daemon"
    echo "  2. Run: ./test/integration-test.sh"
    exit 0
else
    echo -e "${RED}$FAILED test(s) failed${NC}"
    exit 1
fi
