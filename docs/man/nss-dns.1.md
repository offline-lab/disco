% NSS-DNS(1) User Commands

## NAME
nss-dns - Query DNS servers and compare with nss-daemon results

## SYNOPSIS
nss-dns [options] <hostname>

## DESCRIPTION
nss-dns performs DNS lookups for hostnames and can compare the results with those from nss-daemon. This is useful for:
- Verifying DNS resolution
- Comparing traditional DNS with daemon discovery
- Debugging name resolution issues

## OPTIONS
\-\-type TYPE
    Query type: A (default), AAAA, MX, TXT, ANY

\-\-server SERVER
    Use specific DNS server (default: system resolver)

\-\-compare
    Compare DNS results with nss-daemon results

\-\-help, \-h
    Show help message

## OUTPUT
Without \-\-compare:
    DNS lookup for hostname:
    IP address 1
    IP address 2
    ...

With \-\-compare:
    nss-daemon: IP address 1
    DNS:         IP address 2
    Match: No

## EXAMPLES
Basic DNS lookup:
    nss-dns web1

Lookup IPv6 addresses:
    nss-dns --type AAAA web1

Use specific DNS server:
    nss-dns --server 8.8.8.8 web1

Compare DNS with nss-daemon:
    nss-dns --compare web1

Lookup MX records:
    nss-dns --type MX example.com

## NOTES
- Requires network connectivity to DNS servers.
- DNS results may differ from nss-daemon results (expected).
- Useful for debugging hybrid DNS + daemon setups.

## FILES
/run/nss-daemon.sock
    Unix domain socket for daemon communication

## EXIT STATUS
0
    Success

1
    Error (hostname not found, DNS error, etc.)

2
    Comparison mismatch (DNS and daemon results differ)

## SEE ALSO
nss-daemon(1), nss-query(1), dig(1), nslookup(1)

## AUTHOR
NSS Daemon Contributors

## LICENSE
MIT
