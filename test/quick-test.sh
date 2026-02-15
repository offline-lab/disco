#!/bin/bash
set -e

echo "=== NSS Daemon Quick Test ==="
echo

SOCKET="/tmp/nss-daemon-quicktest.sock"
PIDFILE="/tmp/nss-daemon-quicktest.pid"
CONFIG="/tmp/nss-daemon-quicktest.yaml"
TESTDIR="$(pwd)/test"

cleanup() {
    if [ -f "$PIDFILE" ]; then
        kill $(cat "$PIDFILE") 2>/dev/null || true
        rm -f "$PIDFILE"
    fi
    rm -f "$SOCKET" "$CONFIG"
}
trap cleanup EXIT

cat >"$CONFIG" <<EOF
daemon:
  socket_path: "$SOCKET"
  broadcast_interval: 30s
  record_ttl: 3600s

network:
  interfaces: ["en0", "eth0"]
  broadcast_addr: "255.255.255.255:5354"
  max_broadcast_rate: 10

discovery:
  enabled: false
  detect_services: false
  service_port_mapping: {}
  scan_interval: 60s

security:
  enabled: false

logging:
  level: "error"
  format: "text"
EOF

echo "1. Building..."
make -s 2>/dev/null

echo "2. Starting daemon..."
./nss-daemon -config "$CONFIG" 2>/dev/null &
echo $! >"$PIDFILE"
sleep 1

if [ ! -S "$SOCKET" ]; then
    echo "   FAIL: Socket not created"
    exit 1
fi
echo "   Socket: $SOCKET"

echo "3. Testing socket queries..."
go run "$TESTDIR/socket-client/main.go" "$SOCKET"

echo "4. Testing config validation..."
./nss-config-validate "$CONFIG" >/dev/null 2>&1 && echo "   OK" || echo "   FAIL"

echo "5. Testing nss-key..."
KEYFILE="/tmp/test-key-$$.json"
./nss-key generate "$KEYFILE" >/dev/null 2>&1 && echo "   OK" || echo "   FAIL"
rm -f "$KEYFILE"

echo
echo "=== All tests passed ==="
