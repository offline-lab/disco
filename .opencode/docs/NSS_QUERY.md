# NSS Query Tool - User Guide

## Overview

The `nss-query` tool allows you to query the nss-daemon for information about discovered hosts and services. It uses the same Unix domain socket mechanism as the NSS library, making it lightweight and efficient.

## Commands

### `nss-query hosts`

Lists all discovered hosts with their details including:
- Hostname
- IP addresses
- Services advertised
- Last seen timestamp
- Time until record expires

Example:
```bash
$ nss-query hosts
Discovered Hosts (3):

📡 webserver
   Addresses: 192.168.1.10
   Last Seen: 2024-02-11 14:30:45
   Expires In: 35m
   Services: www, ssh

📡 mailserver
   Addresses: 192.168.1.11
   Last Seen: 2024-02-11 14:29:12
   Expires In: 34m
   Services: smtp, imap

📡 filesrv
   Addresses: 192.168.1.12
   Last Seen: 2024-02-11 14:31:00
   Expires In: 36m
```

### `nss-query services`

Lists all discovered services organized by service type. Shows which hosts provide each service.

Example:
```bash
$ nss-query services
Discovered Services (4):

🔧 ftp
   → filesrv (192.168.1.12:21)

🔧 imap
   → mailserver (192.168.1.11:143)

🔧 smtp
   → mailserver (192.168.1.11:25)
   → webserver (192.168.1.10:587)

🔧 ssh
   → webserver (192.168.1.10:22)
```

### `nss-query hosts-services`

Provides a detailed view of all hosts with their associated services. This is the most comprehensive view.

Example:
```bash
$ nss-query hosts-services
Hosts and Services (3):

═══════════════════════════════════════
📡 webserver

Network:
  • 192.168.1.10

Last Seen:   2024-02-11 14:30:45
Expires In: 35m

Services:
  • www
  • ssh

═══════════════════════════════════════
📡 mailserver

Network:
  • 192.168.1.11

Last Seen:   2024-02-11 14:29:12
Expires In: 34m

Services:
  • smtp
  • imap
```

### `nss-query lookup <hostname>`

Looks up a specific host and displays its IP addresses.

Example:
```bash
$ nss-query lookup webserver
Host: webserver
Addresses:
  • 192.168.1.10
  • 10.0.0.10
```

## Use Cases

### Finding a Service

Need to find which host provides SMTP service?
```bash
$ nss-query services | grep -A10 smtp
```

### Checking Network Health

See all hosts and when they were last seen:
```bash
$ nss-query hosts
```

### Finding Host for Service

Looking for a web server?
```bash
$ nss-query hosts-services
```

### Quick Host Lookup

Get IP address for a hostname:
```bash
$ nss-query lookup webserver
```

## Troubleshooting

### "failed to connect to daemon"

The daemon is not running. Start it:
```bash
sudo systemctl start nss-daemon
```

### Empty results

No hosts have been discovered yet. This could be:
- Discovery is disabled in config
- Network has no other hosts
- Firewall blocking UDP broadcasts (port 5353)

### Host showing as "expired"

Host records expire after their TTL (default 1 hour). Check if the host is still on the network:
```bash
$ nss-query hosts
```

## Technical Details

The tool communicates with the daemon via Unix domain socket at `/run/nss-daemon.sock`. All queries use JSON encoding for efficiency and compatibility with the NSS library.

Query types:
- `LIST_HOSTS` - List all hosts with metadata
- `LIST_SERVICES` - List all services by type
- `QUERY_BY_NAME` - Look up specific host
- `LIST` - List all records (legacy)

## Integration

### With Scripts

The tool can be used in scripts:
```bash
#!/bin/bash
HOST=$(nss-query hosts | grep -A2 mailserver | grep "Addresses:" | cut -d: -f2 | tr -d ' ')
echo "Mail server IP: $HOST"
```

### With Monitoring

Regular queries can monitor network health:
```bash
watch -n 30 'nss-query hosts'
```
