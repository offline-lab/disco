% NSS-KEY(1) User Commands

## NAME
nss-key - Generate and manage security keys for nss-daemon

## SYNOPSIS
nss-key <command> [args]

## DESCRIPTION
nss-key manages cryptographic keys used for signing and verifying broadcast messages in nss-daemon. It supports key generation, display, and management.

## COMMANDS
generate [output-file]
    Generate a new key pair and save to the specified file (default: /etc/nss-daemon/keys.json)

    The output file contains:
    - public_key: Your public key for sharing
    - private_key: Your private key (keep secret!)
    - public_keys: List of trusted peer public keys

show [input-file]
    Display key information from the specified file (default: /etc/nss-daemon/keys.json)

    Shows:
    - Public key
    - Private key (truncated for security)
    - List of trusted peers

add-peer <peer-name> <public-key> [output-file]
    Add a trusted peer's public key to the key file.

    peer-name: Human-readable name for the peer (e.g., "web1", "mail1")
    public-key: The peer's public key (hex string)
    output-file: Key file to modify (default: /etc/nss-daemon/keys.json)

remove-peer <peer-name> [output-file]
    Remove a trusted peer from the key file.

help, \-\-help, \-h
    Show help message.

## OPTIONS
\-\-force
    Overwrite existing key file when generating new keys.

## EXAMPLES
Generate a new key pair:
    nss-key generate

Generate and save to custom location:
    nss-key generate /custom/keys.json

Display current keys:
    nss-key show
    nss-key show /etc/nss-daemon/keys.json

Add a trusted peer:
    nss-key add-peer web1 "a1b2c3d4e5f6..."
    nss-key add-peer mail1 "9876543210..." /etc/nss-daemon/keys.json

Remove a trusted peer:
    nss-key remove-peer web1

## KEY FORMAT
Keys are stored in JSON format:

    {
      "public_key": "hex-encoded-public-key",
      "private_key": "hex-encoded-private-key",
      "public_keys": [
        "hex-encoded-peer-key-1",
        "hex-encoded-peer-key-2"
      ]
    }

## SECURITY
- Keep private keys secret! Never share your private_key.
- Share only your public_key with trusted peers.
- Use unique peer names for easy identification.
- Store keys in a secure location with appropriate permissions.

## FILES
/etc/nss-daemon/keys.json
    Default key storage location

## EXIT STATUS
0
    Success

1
    Error (invalid arguments, file not found, permission denied, etc.)

## SEE ALSO
nss-daemon(1), nss-query(1), nss-status(1)

## AUTHOR
NSS Daemon Contributors

## LICENSE
MIT
