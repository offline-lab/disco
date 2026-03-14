#!/bin/bash
# Build debian packages for Disco

set -e

echo "=== Building Disco Debian Packages ==="
echo

# Check if we're on Debian/Ubuntu
if [ ! -f /etc/debian_version ] && [ ! -f /etc/lsb-release ]; then
    echo "Error: This script must be run on Debian/Ubuntu"
    echo "For macOS, use: docker run --rm -v $(pwd):/src -w /src ubuntu:22.04 ./build-debian.sh"
    exit 1
fi

# Install build dependencies
echo "Installing build dependencies..."
sudo apt-get update
sudo apt-get install -y build-essential devscripts debhelper golang-go gcc

# Build the package
echo
echo "Building packages..."
dpkg-buildpackage -us -uc -b

echo
echo "=== Build Complete ==="
echo
echo "Packages created:"
ls -lh ../*.deb 2>/dev/null || echo "No packages found"

echo
echo "To install:"
echo "  sudo dpkg -i ../disco_1.0.0-1_*.deb"
echo
echo "Or install individual packages:"
echo "  sudo dpkg -i ../disco-daemon_1.0.0-1_*.deb"
echo "  sudo dpkg -i ../disco-cli_1.0.0-1_*.deb"
echo "  sudo dpkg -i ../disco-common_1.0.0-1_all.deb"
