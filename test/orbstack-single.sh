#!/bin/bash
set -e

echo "=== NSS Daemon Single-Node Test (OrbStack) ==="
echo

echo "1. Building Linux binaries (arm64 for OrbStack)..."
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/nss-daemon-linux cmd/daemon/main.go
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/nss-query-linux cmd/query/main.go
echo "   ✓ Done"

echo "2. Creating/checking VM..."
orb create debian:bookworm nss-test 2>/dev/null || echo "   VM already exists, reusing"
sleep 2

echo "3. Installing dependencies in VM..."
orb -m nss-test sh -c "apt-get update -qq && apt-get install -y -qq gcc libc6-dev" 2>/dev/null
echo "   ✓ Done"

echo "4. Copying Go binaries..."
cat build/nss-daemon-linux | orb -m nss-test sh -c "cat > /usr/local/bin/nss-daemon && chmod +x /usr/local/bin/nss-daemon"
cat build/nss-query-linux | orb -m nss-test sh -c "cat > /usr/local/bin/nss-query && chmod +x /usr/local/bin/nss-query"
echo "   ✓ Done"

echo "5. Compiling and installing NSS module (in VM)..."
cat libnss/nss_daemon.c | orb -m nss-test sh -c "cat > /tmp/nss_daemon.c && gcc -fPIC -shared -o /lib/aarch64-linux-gnu/libnss_daemon.so.2 /tmp/nss_daemon.c && ln -sf /lib/aarch64-linux-gnu/libnss_daemon.so.2 /lib/aarch64-linux-gnu/libnss_daemon.so && ldconfig"
echo "   ✓ Done"

echo "6. Configuring nsswitch..."
orb -m nss-test sh -c "echo 'hosts: files daemon dns' > /etc/nsswitch.conf"
echo "   ✓ Done"

echo "7. Creating daemon config..."
orb -m nss-test sh -c 'mkdir -p /etc/nss-daemon && cat > /etc/nss-daemon/config.yaml << "EOF"
daemon:
  socket_path: "/run/nss-daemon.sock"
  broadcast_interval: 5s
  record_ttl: 300s
network:
  interfaces: ["eth0"]
  broadcast_addr: "255.255.255.255:5354"
  max_broadcast_rate: 10
discovery:
  enabled: false
security:
  enabled: false
logging:
  level: "debug"
  format: "text"
EOF'
echo "   ✓ Done"

echo "8. Starting daemon..."
orb -m nss-test sh -c "pkill nss-daemon 2>/dev/null || true"
orb -m nss-test sh -c "/usr/local/bin/nss-daemon -config /etc/nss-daemon/config.yaml 2>&1 &"
sleep 2

echo "9. Checking daemon..."
if orb -m nss-test pgrep -f nss-daemon &>/dev/null; then
    echo "   ✓ Daemon running"
else
    echo "   ✗ Daemon NOT running"
    exit 1
fi

echo "10. Checking socket..."
if orb -m nss-test test -S /run/nss-daemon.sock 2>/dev/null; then
    echo "   ✓ Socket exists"
else
    echo "   ✗ Socket missing"
    exit 1
fi

echo "11. Testing nss-query..."
orb -m nss-test /usr/local/bin/nss-query hosts 2>/dev/null || echo "   (empty result expected - no peers yet)"

echo
echo "=========================================="
echo "  12. TESTING NSS MODULE"
echo "=========================================="
echo
echo "   Testing: getent hosts localhost (sanity check)..."
orb -m nss-test getent hosts localhost 2>/dev/null && echo "   ✓ localhost resolved" || echo "   ✗ localhost failed"

echo
echo "   Testing: getent hosts nss-test (via NSS daemon)..."
result=$(orb -m nss-test getent hosts nss-test 2>&1)
if echo "$result" | grep -qE "[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+"; then
    echo "   ✓✓✓ NSS MODULE WORKS! ✓✓✓"
    echo "   Result: $result"
else
    echo "   ✗ NSS module failed"
    echo "   Output: $result"
    echo
    echo "   Debugging info:"
    orb -m nss-test sh -c "ldconfig -p | grep nss_daemon" || echo "   Library not in ldconfig"
    orb -m nss-test cat /etc/nsswitch.conf
fi

echo
echo "=========================================="
echo "  Test Complete"
echo "=========================================="
echo
echo "To interact with VM:"
echo "  orb -m nss-test bash"
echo "  orb -m nss-test getent hosts somehost"
echo
echo "To remove VM:"
echo "  orb delete nss-test"
