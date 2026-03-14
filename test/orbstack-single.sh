#!/bin/bash
set -e

echo "=== Disco Daemon Single-Node Test (OrbStack) ==="
echo

echo "1. Building Linux binaries (arm64 for OrbStack)..."
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/disco-daemon-linux cmd/daemon/main.go
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/disco-linux cmd/disco/main.go
echo "   ✓ Done"
echo
echo "2. Creating/checking VM..."
orb create debian:bookworm disco-test 2>/dev/null || echo "   VM already exists, reusing"
sleep 2
echo
echo "3. Installing dependencies in VM..."
orb -m disco-test sh -c "apt-get update -qq && apt-get install -y -qq gcc libc6-dev netcat-openbsd" 2>/dev/null
echo "   ✓ Done"
echo
echo "4. Copying Go binaries..."
cat build/disco-daemon-linux | orb -m disco-test sh -c "cat > /usr/local/bin/disco-daemon && chmod +x /usr/local/bin/disco-daemon"
cat build/disco-linux | orb -m disco-test sh -c "cat > /usr/local/bin/disco && chmod +x /usr/local/bin/disco"
echo "   ✓ Done"
echo
echo "5. Compiling and installing NSS module (in VM)..."
cat libnss/nss_disco.c | orb -m disco-test sh -c "cat > /tmp/nss_disco.c && gcc -fPIC -shared -o /tmp/libnss_disco.so.2 /tmp/nss_disco.c && ln -sf /tmp/libnss_disco.so.2 /lib/aarch64-linux-gnu/libnss_disco.so.2 /lib/aarch64-linux-gnu/libnss_disco.so && ldconfig
echo "   ✓ Done"
echo
echo "6. Configuring nsswitch..."
orb -m disco-test sh -c "echo 'hosts: files disco dns' > /etc/nsswitch.conf"
echo "   ✓ Done"
echo
echo "7. Creating daemon config..."
orb -m disco-test sh -c 'mkdir -p /etc/disco && cat > /etc/disco/config.yaml << "EOF"
daemon:
  socket_path: "/run/disco.sock"
  broadcast_interval: 5s
  record_ttl: 300s
network:
  interfaces: ["eth0"]
  broadcast_addr: "255.255.255.255:5354"
  max_broadcast_rate: 20
discovery:
  enabled: false
security:
  enabled: false
logging:
  level: "debug"
  format: "text"
EOF'
echo "   ✓ Done"
echo
echo "8. Starting daemon..."
orb -m disco-test sh -c "pkill disco-daemon 2>/dev/null || true"
orb -m disco-test sh -c "/usr/local/bin/disco-daemon -config /etc/disco/config.yaml 2>&1 &"
sleep 2
echo
echo "9. Checking daemon..."
if orb -m disco-test pgrep -f disco-daemon &>/dev/null; then
    echo "   ✓ Daemon running"
else
    echo "   ✗ Daemon NOT running"
    exit 1
fi
echo
echo "10. Checking socket..."
if orb -m disco-test test -S /run/disco.sock 2>/dev/null; then
    echo "   ✓ Socket exists"
else
    echo "   ✗ Socket missing"
    exit 1
fi
echo
echo "11. Testing disco CLI..."
orb -m disco-test /usr/local/bin/disco hosts 2>/dev/null || echo "   (empty result expected - no peers yet)"
echo
echo
echo "=========================================="
echo "  12. TESTING NSS MODULE"
echo "=========================================="
echo
echo "   Testing: getent hosts localhost (sanity check)..."
orb -m disco-test getent hosts localhost 2>/dev/null && echo "   ✓ localhost resolved" || echo "   ✗ localhost failed"
echo
echo "   Testing: getent hosts disco-test (via disco daemon)..."
result=$(orb -m disco-test getent hosts disco-test 2>&1)
if echo "$result" | grep -qE "[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+"; then
    echo "   ✓ NSS MODULE WORKS!"
    echo "   Result: $result"
else
    echo "   ✗ NSS module failed"
    echo "   Output: $result"
    echo "   Debugging info..."
    orb -m disco-test sh -c "ldconfig -p | grep nss_disco" || echo "   Library not in ldconfig"
    orb -m disco-test cat /etc/nsswitch.conf
fi
echo
echo
echo "=========================================="
echo "  Test Complete"
echo "=========================================="
echo
echo "To interact with VM:"
echo "  orb -m disco-test bash"
echo "  orb -m disco-test getent hosts somehost"
echo
echo
echo "To remove VM:"
echo "  orb delete disco-test"
