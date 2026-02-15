#!/bin/bash
set -e

echo "=== NSS Daemon Multi-Node Test Script ==="
echo

COLOR_RESET='\033[0m'
COLOR_GREEN='\033[0;32m'
COLOR_RED='\033[0;31m'
COLOR_YELLOW='\033[1;33m'
COLOR_BLUE='\033[0;34m'

print_success() {
    echo -e "${COLOR_GREEN}✓${COLOR_RESET} $1"
}

print_error() {
    echo -e "${COLOR_RED}✗${COLOR_RESET} $1"
}

print_info() {
    echo -e "${COLOR_BLUE}ℹ${COLOR_RESET} $1"
}

print_header() {
    echo -e "${COLOR_YELLOW}▶${COLOR_RESET} $1"
}

# Check if daemon is running
if [ ! -S /run/nss-daemon.sock ]; then
    print_error "Daemon is not running. Socket not found at /run/nss-daemon.sock"
    echo "Start the daemon with: sudo systemctl start nss-daemon"
    exit 1
fi

print_success "Daemon is running"

# Test 1: Query all hosts
print_header "Test 1: Listing all discovered hosts"
if nss-query hosts >/dev/null 2>&1; then
    print_success "nss-query hosts works"
    nss-query hosts
else
    print_error "nss-query hosts failed"
fi
echo

# Test 2: Query all services
print_header "Test 2: Listing all discovered services"
if nss-query services >/dev/null 2>&1; then
    print_success "nss-query services works"
    nss-query services
else
    print_error "nss-query services failed"
fi
echo

# Test 3: Detailed hosts view
print_header "Test 3: Detailed hosts view"
if nss-query hosts-services >/dev/null 2>&1; then
    print_success "nss-query hosts-services works"
    nss-query hosts-services | head -50
    echo "..."
else
    print_error "nss-query hosts-services failed"
fi
echo

# Test 4: Check for self-discovery
print_header "Test 4: Checking for self-discovery"
LOCAL_HOST=$(hostname)
if nss-query lookup "$LOCAL_HOST" >/dev/null 2>&1; then
    print_success "Self-host '$LOCAL_HOST' found in cache"
    nss-query lookup "$LOCAL_HOST"
else
    print_error "Self-host '$LOCAL_HOST' not found in cache"
fi
echo

# Test 5: Check socket connectivity
print_header "Test 5: Testing socket connectivity"
if timeout 2 bash -c "echo '{}' | nc -U /run/nss-daemon.sock" >/dev/null 2>&1; then
    print_success "Socket is responsive"
else
    print_error "Socket is not responding"
fi
echo

# Test 6: Check for local services
print_header "Test 6: Checking for detected local services"
if [ -S /run/nss-daemon.sock ]; then
    nss-query hosts | grep -A5 "$LOCAL_HOST" | grep -q "Services:" && print_success "Local services detected" || print_info "No services detected yet"
fi
echo

# Test 7: Security check (if enabled)
print_header "Test 7: Checking security status"
if nss-query hosts | grep -q "localhost"; then
    print_info "Basic connectivity verified"
else
    print_info "Run daemon to see discovery results"
fi
echo

print_header "Test Summary"
echo "Basic tests completed. For full multi-node testing:"
echo "  1. Run daemon on multiple hosts"
echo "  2. Wait for discovery (30-60 seconds)"
echo "  3. Run: nss-query hosts"
echo "  4. Run: nss-query hosts-services"
echo
print_info "For Docker testing:"
echo "  docker-compose -f docker-compose-host.yml up -d"
echo "  docker exec -it nss-daemon-web1 nss-query hosts"
echo "  docker exec -it nss-daemon-web1 nss-query hosts-services"
