#!/bin/bash
# Build debian packages using Docker (works on macOS)

set -e

echo "=== Building Debian Packages in Docker ==="
echo

# Check for docker
if ! command -v docker &>/dev/null; then
    echo "Error: docker not found"
    echo "Install: brew install docker"
    exit 1
fi

# Build in Docker container
echo "Building in Ubuntu 22.04 container..."
docker run --rm \
    -v "$(pwd):/src" \
    -w /src \
    ubuntu:22.04 \
    bash -c "
        set -e
        
        echo 'Installing build dependencies...'
        apt-get update -qq
        apt-get install -y -qq build-essential devscripts debhelper golang-go gcc > /dev/null
        
        echo
        echo 'Building debian packages...'
        dpkg-buildpackage -us -uc -b
        
        echo
        echo 'Build complete!'
        ls -lh ../*.deb
    "

echo
echo "=== Build Complete ==="
echo
echo "Packages created:"
ls -lh ../*.deb 2>/dev/null || echo "No packages found in parent directory"
echo
echo "To install on Debian/Ubuntu:"
echo "  sudo dpkg -i ../disco_1.0.0-1_amd64.deb"
echo
