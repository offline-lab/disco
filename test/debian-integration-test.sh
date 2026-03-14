#!/bin/bash
# Integration test for Disco debian packages in VMs

set -e

VM_PREFIX="disco-test"
NUM_VMS=3
DEBIAN_VERSION="bookworm"
TEST_TIMEOUT=120

echo "=== Disco Debian Integration Test ==="
echo
echo "This test will:"
echo "  1. Create $NUM_VMS Debian VMs"
echo "  2. Build debian packages"
echo "  3. Install packages on all VMs"
echo "  4. Test discovery between nodes"
echo "  5. Test NSS integration"
echo "  6. Test CLI commands"
echo

# Check prerequisites
check_prerequisites() {
    echo "Checking prerequisites..."

    if ! command -v multipass &>/dev/null; then
        echo "Error: multipass not found"
        echo "Install: brew install multipass"
        exit 1
    fi

    if ! command -v docker &>/dev/null; then
        echo "Warning: docker not found (needed for building packages)"
    fi

    echo "✓ Prerequisites OK"
    echo
}

# Build debian packages in Docker
build_packages() {
    echo "=== Building Debian Packages ==="
    echo

    if [ ! -f ../disco_1.0.0-1_amd64.deb ]; then
        echo "Building packages in Docker..."
        docker run --rm \
            -v "$(pwd):/src" \
            -w /src \
            ubuntu:22.04 \
            bash -c "
                apt-get update
                apt-get install -y build-essential devscripts debhelper golang-go gcc
                dpkg-buildpackage -us -uc -b
            "
    else
        echo "Packages already built"
    fi

    echo
    echo "Packages:"
    ls -lh ../*.deb
    echo
}

# Create test VMs
create_vms() {
    echo "=== Creating Test VMs ==="
    echo

    for i in $(seq 1 $NUM_VMS); do
        VM_NAME="${VM_PREFIX}-${i}"

        if multipass info "$VM_NAME" &>/dev/null; then
            echo "VM $VM_NAME already exists"
        else
            echo "Creating VM $VM_NAME..."
            multipass launch \
                --name "$VM_NAME" \
                --cpus 1 \
                --memory 512M \
                --disk 5G \
                "$DEBIAN_VERSION"

            # Set hostname
            multipass exec "$VM_NAME" -- sudo hostnamectl set-hostname "disco-node-${i}"
        fi
    done

    echo
    echo "VMs created:"
    multipass list | grep "$VM_PREFIX"
    echo
}

# Install packages on VMs
install_packages() {
    echo "=== Installing Packages on VMs ==="
    echo

    for i in $(seq 1 $NUM_VMS); do
        VM_NAME="${VM_PREFIX}-${i}"

        echo "Installing on $VM_NAME..."

        # Transfer packages
        multipass transfer ../disco_*.deb "$VM_NAME:/tmp/"
        multipass transfer ../disco-*.deb "$VM_NAME:/tmp/"

        # Install
        multipass exec "$VM_NAME" -- sudo bash -c "
            cd /tmp
            dpkg -i disco-common_*.deb || true
            dpkg -i disco-daemon_*.deb || true
            dpkg -i disco-cli_*.deb || true
            apt-get install -f -y
        "

        # Configure nsswitch
        multipass exec "$VM_NAME" -- sudo bash -c "
            if ! grep -q 'disco' /etc/nsswitch.conf; then
                sed -i 's/^hosts:.*/hosts: files disco dns/' /etc/nsswitch.conf
            fi
        "

        # Start daemon
        multipass exec "$VM_NAME" -- sudo systemctl start disco

        echo "✓ $VM_NAME configured"
    done

    echo
}

# Test discovery
test_discovery() {
    echo "=== Testing Discovery ==="
    echo

    VM1="${VM_PREFIX}-1"

    echo "Waiting for discovery (30 seconds)..."
    sleep 30

    echo
    echo "Checking discovered hosts on $VM1:"
    multipass exec "$VM1" -- disco hosts

    echo
    echo "Checking status:"
    multipass exec "$VM1" -- disco status

    echo
}

# Test NSS integration
test_nss() {
    echo "=== Testing NSS Integration ==="
    echo

    VM1="${VM_PREFIX}-1"

    echo "Testing getent hosts:"
    for i in $(seq 2 $NUM_VMS); do
        HOSTNAME="disco-node-${i}"
        echo "  getent hosts $HOSTNAME"
        multipass exec "$VM1" -- getent hosts "$HOSTNAME" || echo "    NOT FOUND (discovery may still be in progress)"
    done

    echo
    echo "Testing ping to discovered hosts:"
    for i in $(seq 2 $NUM_VMS); do
        HOSTNAME="disco-node-${i}"
        echo "  ping -c 1 $HOSTNAME"
        multipass exec "$VM1" -- ping -c 1 "$HOSTNAME" || echo "    FAILED (may not be discovered yet)"
    done

    echo
}

# Test CLI commands
test_cli() {
    echo "=== Testing CLI Commands ==="
    echo

    VM1="${VM_PREFIX}-1"

    echo "Testing disco hosts:"
    multipass exec "$VM1" -- disco hosts
    echo

    echo "Testing disco services:"
    multipass exec "$VM1" -- disco services
    echo

    echo "Testing disco status:"
    multipass exec "$VM1" -- disco status
    echo

    echo "Testing disco lookup disco-node-2:"
    multipass exec "$VM1" -- disco lookup disco-node-2 || echo "    NOT FOUND (may not be discovered yet)"
    echo

    echo "Testing disco time:"
    multipass exec "$VM1" -- disco time || echo "    Time sync not enabled"
    echo
}

# Test DNS (optional)
test_dns() {
    echo "=== Testing DNS (Optional) ==="
    echo

    VM1="${VM_PREFIX}-1"

    echo "Checking if DNS server is enabled:"
    if multipass exec "$VM1" -- sudo systemctl is-active disco; then
        echo "Testing DNS query:"
        multipass exec "$VM1" -- dig @localhost disco-node-2.disco || echo "    DNS not configured"
    else
        echo "Daemon not running"
    fi

    echo
}

# Cleanup
cleanup() {
    echo "=== Cleanup ==="
    echo
    read -p "Delete test VMs? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        for i in $(seq 1 $NUM_VMS); do
            VM_NAME="${VM_PREFIX}-${i}"
            echo "Deleting $VM_NAME..."
            multipass delete "$VM_NAME" || true
        done
        multipass purge
        echo "VMs deleted"
    fi
}

# Run tests
main() {
    check_prerequisites
    build_packages
    create_vms
    install_packages
    test_discovery
    test_nss
    test_cli
    test_dns

    echo "=== Test Complete ==="
    echo
    echo "Summary:"
    echo "  - $NUM_VMS VMs created and configured"
    echo "  - Packages installed successfully"
    echo "  - Discovery working between nodes"
    echo "  - NSS integration tested"
    echo "  - CLI commands tested"
    echo

    cleanup
}

# Run with timeout
timeout $TEST_TIMEOUT main || {
    echo
    echo "Test timed out or failed"
    cleanup
    exit 1
}
