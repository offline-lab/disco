package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the daemon configuration
type Config struct {
	Daemon    DaemonConfig    `yaml:"daemon"`
	Network   NetworkConfig   `yaml:"network"`
	Discovery DiscoveryConfig `yaml:"discovery"`
	Security  SecurityConfig  `yaml:"security"`
	Logging   LoggingConfig   `yaml:"logging"`
}

// DaemonConfig contains daemon-specific configuration
type DaemonConfig struct {
	SocketPath        string        `yaml:"socket_path"`
	BroadcastInterval time.Duration `yaml:"broadcast_interval"`
	RecordTTL         time.Duration `yaml:"record_ttl"`
	PIDFile           string        `yaml:"pid_file"`
}

// NetworkConfig contains network configuration
type NetworkConfig struct {
	Interfaces       []string `yaml:"interfaces"`
	BroadcastAddr    string   `yaml:"broadcast_addr"`
	MaxBroadcastRate int      `yaml:"max_broadcast_rate"`
}

// DiscoveryConfig contains service discovery configuration
type DiscoveryConfig struct {
	Enabled            bool             `yaml:"enabled"`
	DetectServices     bool             `yaml:"detect_services"`
	ServicePortMapping map[string][]int `yaml:"service_port_mapping"`
	ScanInterval       time.Duration    `yaml:"scan_interval"`
}

// SecurityConfig contains security configuration
type SecurityConfig struct {
	Enabled       bool   `yaml:"enabled"`
	KeyPath       string `yaml:"key_path"`
	TrustedPeers  string `yaml:"trusted_peers"`
	RequireSigned bool   `yaml:"require_signed"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	File   string `yaml:"file"`
}

// Load loads configuration from a file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// SetDefaults sets default values for configuration
func (c *Config) SetDefaults() {
	if c.Daemon.SocketPath == "" {
		c.Daemon.SocketPath = "/run/nss-daemon.sock"
	}
	if c.Daemon.BroadcastInterval == 0 {
		c.Daemon.BroadcastInterval = 30 * time.Second
	}
	if c.Daemon.RecordTTL == 0 {
		c.Daemon.RecordTTL = 3600 * time.Second
	}
	if c.Network.BroadcastAddr == "" {
		c.Network.BroadcastAddr = "255.255.255.255:5353"
	}
	if c.Network.MaxBroadcastRate == 0 {
		c.Network.MaxBroadcastRate = 10
	}
	if c.Discovery.ScanInterval == 0 {
		c.Discovery.ScanInterval = 60 * time.Second
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "text"
	}
}
