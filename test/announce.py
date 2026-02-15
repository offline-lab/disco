#!/usr/bin/env python3
"""
NSS Announce - Send broadcast announcements for testing
"""

import socket
import json
import time
import argparse
import sys


def get_local_ips():
    """Get local IP addresses"""
    import subprocess

    try:
        result = subprocess.run(["hostname", "-I"], capture_output=True, text=True)
        ips = result.stdout.strip().split()
        return ips if ips else ["127.0.0.1"]
    except:
        return ["127.0.0.1"]


def announce(hostname, broadcast_addr, service=None, port=None, interval=5, count=0):
    """Send broadcast announcements"""

    services = []
    if service and port:
        services.append({"name": service, "port": int(port), "addr": ""})

    ips = get_local_ips()

    host, port_num = broadcast_addr.rsplit(":", 1)
    port_int = int(port_num)

    print(f"Announcing host: {hostname}")
    print(f"Broadcast address: {broadcast_addr}")
    print(f"Interval: {interval}s")
    print(f"Local IPs: {ips}")
    if services:
        print(f"Services: {services}")
    print()

    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_BROADCAST, 1)

    sent = 0
    try:
        while True:
            if count > 0 and sent >= count:
                print(f"\nSent {sent} announcements, stopping.")
                break

            msg = {
                "type": "ANNOUNCE",
                "message_id": f"announce-{int(time.time() * 1000000)}",
                "timestamp": int(time.time()),
                "hostname": hostname,
                "ips": ips,
                "services": services,
                "ttl": 3600,
            }

            data = json.dumps(msg).encode()
            sock.sendto(data, (host, port_int))
            print(f"[{sent}] Announced: {hostname} ({len(data)} bytes)")
            sent += 1

            if count == 0 or sent < count:
                time.sleep(interval)

    except KeyboardInterrupt:
        print(f"\nStopped after {sent} announcements")
    finally:
        sock.close()


def main():
    parser = argparse.ArgumentParser(description="Send broadcast announcements")
    parser.add_argument("-hostname", required=True, help="Hostname to announce")
    parser.add_argument(
        "-addr", default="255.255.255.255:5354", help="Broadcast address"
    )
    parser.add_argument(
        "-interval", type=float, default=5, help="Announcement interval in seconds"
    )
    parser.add_argument(
        "-count", type=int, default=0, help="Number of announcements (0=unlimited)"
    )
    parser.add_argument("-service", help="Service name to announce")
    parser.add_argument("-port", type=int, help="Service port")

    args = parser.parse_args()

    announce(
        hostname=args.hostname,
        broadcast_addr=args.addr,
        service=args.service,
        port=args.port,
        interval=args.interval,
        count=args.count,
    )


if __name__ == "__main__":
    main()
