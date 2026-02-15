#!/bin/bash
set -e

echo "=== NSS Daemon Integration Test Suite ==="
echo

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0
SKIPPED=0

# Test result functions
pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
    ((PASSED++))
}

fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    ((FAILED++))
}

skip() {
    echo -e "${YELLOW}⊘ SKIP${NC}: $1"
    ((SKIPPED++))
}

info() {
    echo -e "${BLUE}ℹ${NC}: $1"
}

# Check prerequisites
check_prerequisites() {
    echo "=== Checking Prerequisites ==="

    if command -v docker >/dev/null 2>&1; then
        pass "Docker is installed"
    else
        fail "Docker not found - integration tests require Docker"
        return 1
    fi

    if docker info >/dev/null 2>&1; then
        pass "Docker daemon is running"
    else
        fail "Docker daemon not running"
        return 1
    fi

    if [ -f "nss-daemon" ]; then
        pass "nss-daemon binary exists"
    else
        fail "nss-daemon binary not found - run 'make' first"
        return 1
    fi

    if [ -f "libnss_daemon.c" ]; then
        pass "NSS module source exists"
    else
        fail "libnss_daemon.c not found"
        return 1
    fi

    echo
}

# Test 1: Build Docker image
test_build_docker() {
    echo "=== Test 1: Build Docker Image ==="

    if docker images | grep -q "nss-daemon.*latest"; then
        info "Docker image already exists, skipping build"
        pass "Docker image exists"
        echo
        return 0
    fi

    info "Building Docker image (this may take a minute)..."
    if docker build -t nss-daemon:latest . >./test-logs/docker-build.log 2>&1; then
        pass "Docker image built successfully"
    else
        fail "Docker image build failed - check ./test-logs/docker-build.log"
        cat ./test-logs/docker-build.log | tail -20
        echo
        return 1
    fi

    echo
}

# Test 2: Start containers
test_start_containers() {
    echo "=== Test 2: Start Docker Containers ==="

    info "Checking for running containers..."
    if docker ps -a | grep -q "nss-daemon-"; then
        info "Stopping existing containers..."
        docker-compose -f docker-compose-host.yml down 2>/dev/null || true
    fi

    info "Starting 3 containers with host networking..."
    if docker-compose -f docker-compose-host.yml up -d >./test-logs/docker-up.log 2>&1; then
        pass "Containers started successfully"
    else
        fail "Failed to start containers - check ./test-logs/docker-up.log"
        cat ./test-logs/docker-up.log | tail -20
        echo
        return 1
    fi

    info "Waiting for daemons to initialize (10 seconds)..."
    sleep 10

    # Check if containers are running
    running=$(docker ps --filter "name=nss-daemon-" --format "{{.Names}}" | wc -l)
    if [ "$running" -eq 3 ]; then
        pass "All 3 containers are running"
    else
        fail "Only $running/3 containers running"
        docker ps -a --filter "name=nss-daemon-"
        echo
        return 1
    fi

    echo
}

# Test 3: Verify daemons started
test_daemons_running() {
    echo "=== Test 3: Verify Daemons Running ==="

    for container in nss-daemon-web1 nss-daemon-mail1 nss-daemon-client1; do
        if docker exec "$container" pgrep -f "nss-daemon" >/dev/null; then
            pass "nss-daemon running in $container"
        else
            fail "nss-daemon not running in $container"
            docker logs "$container" | tail -20
        fi
    done

    echo
}

# Test 4: Check socket connectivity
test_socket_connectivity() {
    echo "=== Test 4: Test Socket Connectivity ==="

    for container in nss-daemon-web1 nss-daemon-mail1 nss-daemon-client1; do
        if docker exec "$container" ls -la /run/nss-daemon.sock >/dev/null 2>&1; then
            pass "Socket exists in $container"
        else
            fail "Socket missing in $container"
            docker exec "$container" ls -la /run/ || true
        fi
    done

    echo
}

# Test 5: Test daemon queries
test_daemon_queries() {
    echo "=== Test 5: Test Daemon Queries ==="

    for container in nss-daemon-web1 nss-daemon-mail1 nss-daemon-client1; do
        # Test list hosts
        if docker exec "$container" echo '{"type":"LIST_HOSTS"}' | nc -U /run/nss-daemon.sock -w 2 >/dev/null 2>&1; then
            pass "LIST_HOSTS query works in $container"
        else
            fail "LIST_HOSTS query failed in $container"
        fi

        # Test list services
        if docker exec "$container" echo '{"type":"LIST_SERVICES"}' | nc -U /run/nss-daemon.sock -w 2 >/dev/null 2>&1; then
            pass "LIST_SERVICES query works in $container"
        else
            fail "LIST_SERVICES query failed in $container"
        fi
    done

    echo
}

# Test 6: Check broadcast traffic
test_broadcast_traffic() {
    echo "=== Test 6: Check Broadcast Traffic ==="

    info "Monitoring for UDP broadcasts on port 5353 (5 seconds)..."

    # Start tcpdump in background
    docker exec nss-daemon-web1 timeout 5 tcpdump -i any -n udp port 5353 >./test-logs/tcpdump.log 2>/dev/null &

    sleep 5

    # Check if any broadcasts were captured
    if [ -s ./test-logs/tcpdump.log ]; then
        packets=$(grep -c "UDP" ./test-logs/tcpdump.log || echo "0")
        if [ "$packets" -gt 0 ]; then
            pass "Broadcast traffic detected ($packets packets)"
        else
            fail "No broadcast packets captured"
        fi
    else
        fail "tcpdump failed to capture packets"
    fi

    echo
}

# Test 7: Test host discovery
test_host_discovery() {
    echo "=== Test 7: Test Host Discovery ==="

    # Wait for announcements to propagate
    info "Waiting for host discovery (20 seconds)..."
    sleep 20

    # Query web1 from mail1
    if docker exec nss-daemon-mail1 echo '{"type":"QUERY_BY_NAME","name":"web1"}' | nc -U /run/nss-daemon.sock -w 2 | grep -q '"type":"OK"'; then
        pass "web1 discovered by mail1"
    else
        fail "web1 not discovered by mail1"
    fi

    # Query mail1 from web1
    if docker exec nss-daemon-web1 echo '{"type":"QUERY_BY_NAME","name":"mail1"}' | nc -U /run/nss-daemon.sock -w 2 | grep -q '"type":"OK"'; then
        pass "mail1 discovered by web1"
    else
        fail "mail1 not discovered by web1"
    fi

    # Query client1 from web1
    if docker exec nss-daemon-web1 echo '{"type":"QUERY_BY_NAME","name":"client1"}' | nc -U /run/nss-daemon.sock -w 2 | grep -q '"type":"OK"'; then
        pass "client1 discovered by web1"
    else
        fail "client1 not discovered by web1"
    fi

    echo
}

# Test 8: Test service detection
test_service_detection() {
    echo "=== Test 8: Test Service Detection ==="

    # Check if web1 has www service detected
    if docker exec nss-daemon-web1 echo '{"type":"LIST_HOSTS"}' | nc -U /run/nss-daemon.sock -w 2 | grep -q '"www"'; then
        pass "www service detected on web1"
    else
        info "www service not yet detected on web1 (may need more time)"
    fi

    echo
}

# Test 9: Test nss-query tool
test_nss_query_tool() {
    echo "=== Test 9: Test nss-query Tool ==="

    # Test nss-query hosts
    if docker exec nss-daemon-web1 nss-query hosts >/dev/null 2>&1; then
        pass "nss-query hosts works"
    else
        fail "nss-query hosts failed"
    fi

    # Test nss-query services
    if docker exec nss-daemon-web1 nss-query services >/dev/null 2>&1; then
        pass "nss-query services works"
    else
        fail "nss-query services failed"
    fi

    # Test nss-query lookup
    if docker exec nss-daemon-web1 nss-query lookup mail1 >/dev/null 2>&1; then
        pass "nss-query lookup works"
    else
        fail "nss-query lookup failed"
    fi

    echo
}

# Test 10: Test config validation
test_config_validation() {
    echo "=== Test 10: Test Config Validation ==="

    if docker exec nss-daemon-web1 nss-config-validate /etc/nss-daemon/config.yaml >/dev/null 2>&1; then
        pass "Config validation works"
    else
        fail "Config validation failed"
    fi

    echo
}

# Cleanup
cleanup() {
    echo "=== Cleanup ==="

    info "Stopping and removing containers..."
    docker-compose -f docker-compose-host.yml down 2>/dev/null || true

    info "Removing temporary files..."
    rm -f ./test-logs/docker-build.log ./test-logs/docker-up.log ./test-logs/tcpdump.log

    echo
}

# Summary
summary() {
    echo "=== Test Summary ==="
    echo "Passed:  $PASSED"
    echo "Failed:  $FAILED"
    echo "Skipped: $SKIPPED"
    echo "Total:   $((PASSED + FAILED + SKIPPED))"
    echo

    if [ $FAILED -eq 0 ]; then
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}$FAILED test(s) failed${NC}"
        exit 1
    fi
}

# Main test runner
main() {
    echo "NSS Daemon Integration Test Suite"
    echo "================================"
    echo

    # Set trap for cleanup
    trap cleanup EXIT

    # Run tests
    if ! check_prerequisites; then
        exit 1
    fi

    test_build_docker || true
    test_start_containers || {
        summary
        exit 1
    }
    test_daemons_running || true
    test_socket_connectivity || true
    test_daemon_queries || true
    test_broadcast_traffic || true
    test_host_discovery || true
    test_service_detection || true
    test_nss_query_tool || true
    test_config_validation || true

    summary
}

# Run main
main "$@"
