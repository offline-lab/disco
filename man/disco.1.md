% DISCO(1) User Commands

## NAME
disco - Unified CLI for disco daemon management and queries

## SYNOPSIS
disco <command> [subcommand] [options] [args]

## DESCRIPTION
disco is the unified command-line interface for managing and querying the disco-daemon. It provides commands for host discovery, service management, configuration validation, key management, and time synchronization.

## GLOBAL OPTIONS
\-\-socket PATH
    Path to Unix domain socket (default: /run/disco.sock)

\-\-json
    Output in JSON format

\-\-no-color
    Disable colored output

\-\-help, \-h
    Show help message

## COMMANDS

### Host Management
hosts list
    List all discovered hosts with health status

hosts show <hostname>
    Show detailed information about a specific host

hosts forget <hostname>
    Remove a host from the discovery cache

hosts mark-lost <hostname>
    Mark a host as lost (manually trigger lost state)

### Service Discovery
services
    List all discovered services

services show <service>
    Show detailed information about a specific service

### Name Resolution
lookup <hostname>
    Resolve a hostname to IP address(es)

### Daemon Management
status
    Show daemon status and statistics

start [options]
    Start the disco-daemon with specified options
    Options:
        \-\-config PATH    Configuration file path
        \-\-foreground     Run in foreground (don't daemonize)

### Configuration
config validate <file>
    Validate a configuration file

### Key Management
key generate
    Generate a new HMAC key for message signing

key show
    Display the current HMAC key

key add-trusted <public-key>
    Add a trusted peer's public key for signature verification

### Time Synchronization
time
    Show time synchronization status

timeset
    Force immediate time synchronization update

### Network Diagnostics
ping <hostname>
    Ping a discovered host to check connectivity
    Options:
        \-\-count N         Number of pings (1-10, default: 4)
        \-\-interval DUR    Time between pings (min 100ms, default: 1s)
        \-\-port PORT       Target port (default: 5353)
        \-\-verbose         Show detailed output

### Manual Announcements
announce
    Send manual discovery broadcast announcements
    Options:
        \-\-hostname NAME   Hostname to announce (required)
        \-\-addr ADDR       Broadcast address (default: 255.255.255.255:5354)
        \-\-interval DUR    Announcement interval (default: 5s)
        \-\-count N         Number of announcements (0 = unlimited)
        \-\-service NAME    Service name to announce
        \-\-port PORT       Service port (requires \-\-service)

## OUTPUT FORMATS
By default, disco outputs human-readable text with colors. Use \-\-json for machine-readable output.

Example text output:
    HOSTNAME    ADDRESS          LAST SEEN    STATUS
    web1        192.168.1.10     2m ago       healthy
    mail1       192.168.1.11     5m ago       healthy

Example JSON output:
    {"hostname":"web1","address":"192.168.1.10","last_seen":"2024-01-15T10:30:00Z","status":"healthy"}

## EXAMPLES
List all discovered hosts:
    disco hosts

Show details for a specific host:
    disco hosts show web1

List all services:
    disco services

Look up a hostname:
    disco lookup web1

Check daemon status:
    disco status

Validate configuration:
    disco config validate /etc/disco/config.yaml

Generate a new signing key:
    disco key generate

Check time sync status:
    disco time

Force time update:
    disco timeset

Ping a discovered host:
    disco ping web1

Ping with custom count and interval:
    disco ping --count 10 --interval 500ms web1

Send manual announcement:
    disco announce --hostname test1

Announce with service:
    disco announce --hostname mail1 --service smtp --port 25

Start daemon in foreground:
    disco start --config /etc/disco/config.yaml --foreground

## FILES
/run/disco.sock
    Unix domain socket for daemon communication

/etc/disco/config.yaml
    Default configuration file location

## EXIT STATUS
0
    Success

1
    Error (invalid arguments, connection failed, etc.)

2
    Partial failure (e.g., some hosts not found)

## SEE ALSO
disco-daemon(1), disco-gps-broadcaster(1)

## AUTHOR
Disco Daemon Contributors

## LICENSE
MIT
