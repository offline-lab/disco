% NSS-DAEMON(1) User Commands

## NAME
nss-daemon - Lightweight name service daemon for offline, airgapped networks

## SYNOPSIS
nss-daemon [options]

## DESCRIPTION
nss-daemon provides automatic service discovery and name resolution across nodes in an offline network without requiring external DNS services. It uses a custom NSS module (libnss_daemon.so.2) for seamless integration with glibc, avoiding the need for DNS servers or resolv.conf modifications.

The daemon automatically discovers nodes via UDP broadcast, detects local services, and responds to name resolution queries from the NSS module.

## OPTIONS
\-\-config PATH
    Path to configuration file (default: /etc/nss-daemon/config.yaml)

\-\-version
    Show version and exit

\-\-help, \-h
    Show help message

## CONFIGURATION
The daemon reads configuration from a YAML file. Key settings include:

**daemon**
  socket_path: Path to Unix domain socket (default: /run/nss-daemon.sock)
  broadcast_interval: Time between broadcasts (default: 30s)
  record_ttl: Time-to-live for cached records (default: 3600s)

**network**
  interfaces: List of network interfaces to use (default: all)
  broadcast_addr: UDP broadcast address (default: 255.255.255.255:5353)
  max_broadcast_rate: Maximum broadcasts per second (default: 10)

**discovery**
  enabled: Enable/disable service discovery (default: true)
  detect_services: Auto-detect local services (default: true)
  service_port_mapping: Map service names to ports (e.g., www: [80, 443])
  scan_interval: Time between service scans (default: 60s)

**security**
  enabled: Enable message signing/verification (default: false)
  key_path: Path to key file (required if security enabled)
  require_signed: Reject unsigned messages (default: false)

**logging**
  level: Log level: debug, info, warn, error, fatal (default: info)
  format: Log format: text or json (default: text)
  file: Path to log file (default: stdout)

## OUTPUT
The daemon logs to stdout or the configured log file. Log messages include timestamps and contextual information.

## EXAMPLES
Start with default configuration:
    nss-daemon

Start with custom configuration:
    nss-daemon --config /custom/config.yaml

Run in foreground to see logs:
    nss-daemon --config /etc/nss-daemon/config.yaml

## FILES
/etc/nss-daemon/config.yaml
    Default configuration file

/run/nss-daemon.sock
    Unix domain socket for NSS module queries

/lib/x86_64-linux-gnu/libnss_daemon.so.2
    NSS module library

## SIGNALS
SIGINT, SIGTERM
    Gracefully shut down the daemon

## SEE ALSO
nss-query(1), nss-status(1), nss-key(1), nss-config-validate(1)

## AUTHOR
NSS Daemon Contributors

## LICENSE
MIT
