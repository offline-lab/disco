% NSS-PING(1) User Commands

## NAME
nss-ping - Ping discovered hosts to check connectivity

## SYNOPSIS
nss-ping [options] <hostname>

## DESCRIPTION
nss-ping sends ICMP echo requests to discovered hosts to verify network connectivity. It works with hosts discovered by the nss-daemon, either by hostname or IP address.

## OPTIONS
\-\-count N
    Send N echo requests (default: 4)

\-\-interval SECONDS
    Wait SECONDS seconds between pings (default: 1)

\-\-timeout SECONDS
    Wait SECONDS seconds for response (default: 2)

\-\-help, \-h
    Show help message

## OUTPUT
For each ping, nss-ping displays:

    PING hostname (ip) 56(84) bytes of data.
    64 bytes from ip: icmp_seq=1 ttl=64 time=0.123 ms
    64 bytes from ip: icmp_seq=2 ttl=64 time=0.456 ms

Summary statistics:
    --- hostname ping statistics ---
    4 packets transmitted, 4 received, 0.0% packet loss
    round-trip min/avg/max = 0.123/0.289/0.456 ms

## EXAMPLES
Ping a discovered host by hostname:
    nss-ping web1
    nss-ping mail1

Ping with custom count:
    nss-ping --count 10 web1

Ping with custom timeout:
    nss-ping --timeout 5 web1

## NOTES
- Requires root privileges or CAP_NET_RAW capability.
- Only works with IPv4 addresses.
- ICMP may be blocked by firewalls.

## FILES
/run/nss-daemon.sock
    Unix domain socket for daemon communication (used to resolve hostnames to IPs)

## EXIT STATUS
0
    Success (all pings responded or ping completed)

1
    Error (hostname not found, permission denied, etc.)

2
    At least one ping timed out (ping completed but with failures)

## SEE ALSO
nss-daemon(1), nss-query(1), ping(8)

## AUTHOR
NSS Daemon Contributors

## LICENSE
MIT
