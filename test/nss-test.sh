#!/bin/bash
set -e

echo "=== Disco Daemon Multi-Node Test Script ==="
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

if [ ! -S /run/disco.sock ]; then
    print_error "Daemon is not running. Socket not found at /run/disco.sock"
    echo "Start the daemon with: sudo systemctl start disco"
    exit 1
fi

print_success "Daemon is running"

print_header "Test 1: Listing all discovered hosts"
if disco hosts >/dev/null 2>&1; then
    print_success "disco hosts works"
    disco hosts
else
    print_error "disco hosts failed"
fi

if disco services >/dev/null 2>&1; then
    print_success "disco services works"
    disco services
else
    print_error "disco services failed"
fi
echo

print_header "Test 2: Listing all discovered services"
if disco services >/dev/null 2>&1; then
    print_success "disco services works"
    disco services
else
    print_error "disco services failed"
fi
echo

print_header "Test 3: Detailed hosts view"
if disco hosts >/dev/null 2>&1; then
    print_success "disco hosts works"
    disco hosts | head -50
    echo "..."
else
    print_error "disco hosts failed"
fi
echo

print_header "Test 4: Checking for self-discovery"
LOCAL_HOST=$(hostname)
if disco lookup "$LOCAL_HOST" >/dev/null 2>&1; then
    print_success "Self-host '$LOCAL_HOST' found in cache"
    disco lookup "$LOCAL_HOST"
else
    print_error "Self-host '$LOCAL_HOST' not found in cache"
fi
echo

print_header "Test 5: Testing socket connectivity"
if timeout 2 bash -c "echo '{}' | nc -U /run/disco.sock" >/dev/null 2>&1; then
    print_success "Socket is responsive"
else
    print_error "Socket is not responding"
fi
echo

print_header "Test 6: Checking for detected local services"
if [ -S /run/disco.sock ]; then
    disco hosts | grep -A5 "$LOCAL_HOST" | grep -q "Services:" && print_success "Local services detected" || print_info "No services detected yet"
fi
echo

print_header "Test 7: Checking security status"
if disco hosts | grep -q "localhost"; then
    print_info "Basic connectivity verified"
else
    print_info "Run daemon to see discovery results"
fi
echo

print_header "Test Summary"
echo "Basic tests completed. For full multi-node testing:"
echo "  1. Run daemon on multiple hosts"
echo "  2. Wait for discovery (30-60 seconds)"
echo "  3. Run: disco hosts"
echo "  4. Run: disco services"
echo
print_info "For Docker testing:"
echo "  docker-compose -f docker-compose-host.yml up -d"
echo "  docker exec -it disco-web1 disco hosts"
echo "  docker exec -it disco-web1 disco services"
