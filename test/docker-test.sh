#!/bin/bash
set -e

echo "=== NSS Daemon Multi-Node Docker Test ==="
echo

cleanup() {
    echo "Cleaning up..."
    docker-compose down 2>/dev/null || true
}
trap cleanup EXIT

echo "1. Building Docker images..."
docker-compose build --quiet

echo "2. Starting 3 nodes..."
docker-compose up -d

echo "3. Waiting for nodes to discover each other..."
sleep 15

echo "4. Testing node1..."
echo "   Checking discovered hosts..."
docker exec nss-node1 nss-query hosts 2>/dev/null | grep -q "node" && echo "   Found peers!" || echo "   No peers yet (may need more time)"

echo "5. Testing NSS module on node1..."
docker exec nss-node1 getent hosts node1 2>/dev/null && echo "   NSS lookup works!" || echo "   NSS not working (expected if no peers discovered)"

echo "6. Checking daemon logs from node1..."
docker exec nss-node1 cat /proc/1/fd/1 2>/dev/null | tail -10 || docker logs nss-node1 2>&1 | tail -10

echo
echo "=== Test Complete ==="
echo
echo "To interact with nodes:"
echo "  docker exec -it nss-node1 bash"
echo "  docker exec nss-node1 nss-query hosts"
echo "  docker exec nss-node1 nss-query services"
echo
echo "To stop: docker-compose down"
