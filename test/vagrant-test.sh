#!/bin/bash
set -e

echo "========================================"
echo "  NSS Daemon Multi-Node Linux Test"
echo "========================================"
echo

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}✓ $1${NC}"; }
fail() { echo -e "${RED}✗ $1${NC}"; }
warn() { echo -e "${YELLOW}! $1${NC}"; }

cleanup() {
    echo
    echo "Cleaning up..."
    vagrant halt 2>/dev/null || true
}
trap cleanup EXIT

echo "1. Checking prerequisites..."
if ! command -v vagrant &>/dev/null; then
    fail "vagrant not installed"
    echo "  Install: brew install vagrant"
    exit 1
fi
pass "vagrant installed"

if ! command -v virtualbox &>/dev/null && ! VBoxManage --version &>/dev/null 2>&1; then
    warn "VirtualBox not found - may need to install"
fi

echo
echo "2. Bringing up VMs (this takes a few minutes)..."
vagrant up --provision

echo
echo "3. Starting daemons on all nodes..."
vagrant ssh node1 -c "sudo nss-daemon -config /etc/nss-daemon/config.yaml &" 2>/dev/null || true
vagrant ssh node2 -c "sudo nss-daemon -config /etc/nss-daemon/config.yaml &" 2>/dev/null || true
vagrant ssh node3 -c "sudo nss-daemon -config /etc/nss-daemon/config.yaml &" 2>/dev/null || true

echo "   Waiting for daemons to start..."
sleep 3

echo
echo "4. Verifying daemons are running..."
for node in node1 node2 node3; do
    if vagrant ssh $node -c "pgrep -f nss-daemon" &>/dev/null; then
        pass "$node daemon running"
    else
        fail "$node daemon NOT running"
    fi
done

echo
echo "5. Verifying sockets exist..."
for node in node1 node2 node3; do
    if vagrant ssh $node -c "test -S /run/nss-daemon.sock" 2>/dev/null; then
        pass "$node socket exists"
    else
        fail "$node socket missing"
    fi
done

echo
echo "6. Waiting for node discovery (15s)..."
sleep 15

echo
echo "7. Testing nss-query on node1..."
echo "   Querying hosts..."
vagrant ssh node1 -c "sudo nss-query hosts" 2>/dev/null || warn "nss-query failed"

echo
echo "8. Testing NSS integration on node1..."
echo "   Testing: getent hosts node1 (self)..."
if vagrant ssh node1 -c "getent hosts node1" 2>/dev/null | grep -q "192.168.56"; then
    pass "getent hosts node1 works"
else
    fail "getent hosts node1 failed"
fi

echo "   Testing: getent hosts node2 (peer)..."
if vagrant ssh node1 -c "getent hosts node2" 2>/dev/null | grep -q "192.168.56"; then
    pass "getent hosts node2 works (peer discovered!)"
else
    warn "getent hosts node2 not found (may need more discovery time)"
fi

echo
echo "9. Testing reverse lookup on node1..."
if vagrant ssh node1 -c "getent hosts 192.168.56.10" 2>/dev/null | grep -q "node"; then
    pass "reverse lookup works"
else
    warn "reverse lookup failed"
fi

echo
echo "10. Checking daemon logs from node1..."
vagrant ssh node1 -c "ps aux | grep nss-daemon | head -5" 2>/dev/null || true

echo
echo "========================================"
echo "  Test Complete"
echo "========================================"
echo
echo "To keep VMs running and test manually:"
echo "  vagrant ssh node1"
echo "  sudo nss-query hosts"
echo "  getent hosts node2"
echo
echo "To destroy VMs:"
echo "  vagrant destroy -f"
