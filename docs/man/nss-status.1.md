% NSS-STATUS(1) User Commands

## NAME
nss-status - Display daemon status and cached records

## SYNOPSIS
nss-status [options]

## DESCRIPTION
nss-status displays the current status of the nss-daemon, including:
- Daemon process information
- Cache statistics
- Number of discovered hosts
- Number of discovered services
- Recent activity

## OPTIONS
\-\-format FORMAT
    Output format: text or json (default: text)

\-\-help, \-h
    Show help message

## OUTPUT
The default text format shows:

Daemon Status:
  Status: Running
  PID: 12345
  Uptime: 2h 30m
  Socket: /run/nss-daemon.sock

Cache Statistics:
  Total Records: 15
  Expired Records: 2
  Fresh Records: 13

Discovery:
  Hosts Discovered: 13
  Services Detected: 24
  Last Broadcast: 10 seconds ago

JSON format provides machine-readable output with the same information.

## EXAMPLES
Show daemon status (default text format):
    nss-status

Show status in JSON format:
    nss-status --format json

## FILES
/run/nss-daemon.sock
    Unix domain socket for daemon communication

## EXIT STATUS
0
    Success

1
    Error (daemon not running, socket not found, etc.)

## SEE ALSO
nss-daemon(1), nss-query(1), nss-key(1)

## AUTHOR
NSS Daemon Contributors

## LICENSE
MIT
