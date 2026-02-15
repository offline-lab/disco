% NSS-QUERY(1) User Commands

## NAME
nss-query - Query daemon for host and service information

## SYNOPSIS
nss-query <command> [args]

## DESCRIPTION
nss-query allows you to query the nss-daemon for information about discovered hosts and services. It uses the same Unix domain socket mechanism as the NSS library, making it lightweight and efficient.

## COMMANDS
hosts
    List all discovered hosts with details including hostname, IP addresses, services, last seen time, and expiration.

services
    List all discovered services grouped by service type, showing which hosts provide each service.

hosts-services
    Display detailed view of hosts and their services, including network information and timing details.

lookup <name>
    Look up a specific host by hostname and display its IP addresses.

help, \-\-help, \-h
    Show help message.

## OUTPUT
The output is formatted for human readability with color-coded information (on supported terminals).

hosts command shows:
    📡 Hostname
       Addresses: IP addresses
       Last Seen: Timestamp
       Expires In: Time remaining
       Services: List of services

services command shows:
    🔧 Service name
       → hostname (address:port)

hosts-services command shows:
    ══════════════════════════════════════
    📡 Hostname

    Network:
      • IP address

    Last Seen:   Timestamp
    Expires In: Time remaining

    Services:
      • Service name

## EXAMPLES
List all discovered hosts:
    nss-query hosts

List all discovered services:
    nss-query services

Show detailed host and service information:
    nss-query hosts-services

Look up a specific host:
    nss-query lookup web1
    nss-query lookup mail1

## FILES
/run/nss-daemon.sock
    Unix domain socket for daemon communication

## EXIT STATUS
0
    Success

1
    Error (see error message)

## SEE ALSO
nss-daemon(1), nss-status(1), nss-key(1)

## AUTHOR
NSS Daemon Contributors

## LICENSE
MIT
