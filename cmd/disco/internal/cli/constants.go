package cli

import "time"

const (
	// Socket and paths
	DefaultSocketPath = "/run/disco.sock"
	DefaultConfigPath = "/etc/disco/config.yaml"
	DefaultKeysPath   = "/etc/disco/keys.json"

	// Network defaults
	DefaultBroadcastAddr = "255.255.255.255:5354"
	DefaultBroadcastPort = 5354
	MinPort              = 1
	MaxPort              = 65535

	// Ping command
	DefaultPingCount    = 4
	MinPingCount        = 1
	MaxPingCount        = 10
	DefaultPingTimeout  = 2 * time.Second
	DefaultPingInterval = 1 * time.Second
	MinPingInterval     = 100 * time.Millisecond

	// Announce command
	DefaultAnnounceInterval = 5 * time.Second
	MinAnnounceInterval     = 1 * time.Second

	// Output formatting
	MaxAddressDisplayWidth  = 34
	MaxServiceDisplayWidth  = 18
	MaxHostnameDisplayWidth = 20
)
