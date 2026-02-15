% NSS-CONFIG-VALIDATE(1) User Commands

## NAME
nss-config-validate - Validate nss-daemon configuration files

## SYNOPSIS
nss-config-validate [options] <config-file>

## DESCRIPTION
nss-config-validate checks the syntax and validity of nss-daemon configuration files before starting the daemon. This helps catch configuration errors early.

## OPTIONS
\-\-verbose, \-v
    Show detailed validation results

\-\-help, \-h
    Show help message

## VALIDATION CHECKS
The validator checks:

**Daemon Configuration**
  - socket_path exists and is absolute path
  - broadcast_interval is at least 5 seconds
  - broadcast_interval does not exceed 1 hour
  - record_ttl is at least 60 seconds
  - record_ttl does not exceed 24 hours

**Network Configuration**
  - broadcast_addr is valid (format: host:port)
  - max_broadcast_rate is between 1 and 100

**Discovery Configuration**
  - scan_interval is at least 10 seconds
  - scan_interval does not exceed 10 minutes
  - service_port_mapping is not empty
  - Service names are not empty
  - Ports are between 1 and 65535

**Security Configuration**
  - key_path is provided when security is enabled
  - Trusted peers file path is valid

**Logging Configuration**
  - log_level is one of: debug, info, warn, error, fatal
  - log_file path is absolute (if specified)

## OUTPUT
Success:
    Validating configuration: /etc/nss-daemon/config.yaml

    ✅ Configuration file loaded successfully

    Validating configuration...

    ✅ All checks passed. Configuration is ready to use.

Error:
    Validating configuration: /etc/nss-daemon/config.yaml

    ❌ Configuration validation failed: daemon config invalid: socket_path is required

With \-\-verbose:
    Shows detailed breakdown of each check:
    ✅ socket_path: /run/nss-daemon.sock
    ✅ broadcast_interval: 30s
    ❌ max_broadcast_rate: 150 (must be <= 100)

## EXAMPLES
Validate configuration:
    nss-config-validate /etc/nss-daemon/config.yaml

Validate with verbose output:
    nss-config-validate --verbose config.yaml

## CONFIGURATION FILE FORMAT
The configuration file is in YAML format. See nss-daemon(1) for full configuration reference.

## FILES
/etc/nss-daemon/config.yaml
    Default configuration file

## EXIT STATUS
0
    Configuration is valid

1
    Configuration file has errors

2
    Configuration file not found or not readable

## SEE ALSO
nss-daemon(1)

## AUTHOR
NSS Daemon Contributors

## LICENSE
MIT
