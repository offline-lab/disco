#!/bin/bash
# Quick test for debian packages in a single VM

set -e

VM_NAME="disco-test-quick"

echo "=== Quick Debian Package Test ==="
echo

# Check prerequisites
if ! command -v multipass &>/dev/null; then
    echo "Error: multipass not found"
    exit 1
fi

# Create VM
echo "Creating test VM..."
multipass launch --name "$VM_NAME" --cpus 1 --memory 512M bookworm || true

echo
echo "Installing dependencies..."
multipass exec "$VM_NAME" -- sudo apt-get update
multipass exec "$VM_NAME" -- sudo apt-get install -y build-essential golang-go

echo
echo "Building packages in VM..."
multipass exec "$VM_NAME" -- bash -c "
    cd /tmp
    git clone https://github.com/offline-lab/disco || true
    cd disco
    make
    sudo make install
"

echo
echo "Configuring NSS..."
multipass exec "$VM_NAME" -- sudo bash -c "
    echo 'hosts: files disco dns' > /etc/nsswitch.conf
    systemctl start disco
"

echo
echo "Testing disco commands..."
multipass exec "$VM_NAME" -- disco hosts
multipass exec "$VM_NAME" -- disco status

echo
echo "✓ Quick test complete"
echo
read -p "Delete VM? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    multipass delete "$VM_NAME"
    multipass purge
fi
