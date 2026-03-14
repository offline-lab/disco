package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the daemon configuration
type Config struct {
	Daemon      DaemonConfig          `yaml:"daemon"`
	Network     NetworkConfig         `yaml:"network"`
	Discovery   DiscoveryConfig       `yaml:"discovery"`
	Security    SecurityConfig        `yaml:"security"`
	Logging     LoggingConfig         `yaml:"logging"`
	TimeSync    TimeSyncConfig        `yaml:"time_sync"`
	Health      HealthConfig          `yaml:"health"`
	DNS         DNSConfig             `yaml:"dns"`
	StaticHosts map[string]StaticHost `yaml:"static_hosts"`
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

// TimeSyncConfig contains time synchronization configuration
type TimeSyncConfig struct {
	Enabled           bool          `yaml:"enabled"`
	MinSources        int           `yaml:"min_sources"`
	MaxSourceSpread   time.Duration `yaml:"max_source_spread"`
	MaxStaleAge       time.Duration `yaml:"max_stale_age"`
	MaxDispersion     float64       `yaml:"max_dispersion"`
	StepThreshold     time.Duration `yaml:"step_threshold"`
	SlewThreshold     time.Duration `yaml:"slew_threshold"`
	PollInterval      time.Duration `yaml:"poll_interval"`
	RequireSigned     bool          `yaml:"require_signed"`
	AllowStepBackward bool          `yaml:"allow_step_backward"`
}

type HealthConfig struct {
	GracePeriod     time.Duration `yaml:"grace_period"`
	ExpireAfter     time.Duration `yaml:"expire_after"`
	CleanupInterval time.Duration `yaml:"cleanup_interval"`
}

type DNSConfig struct {
	Enabled       bool     `yaml:"enabled"`
	Port          int      `yaml:"port"`
	Domain        string   `yaml:"domain"`
	BindAddresses []string `yaml:"bind_addresses"`
	TTLHealthy    int      `yaml:"ttl_healthy"`
	TTLStale      int      `yaml:"ttl_stale"`
}

type StaticHost struct {
	Addresses []string            `yaml:"addresses"`
	Services  []StaticHostService `yaml:"services"`
}

type StaticHostService struct {
	Name     string `yaml:"name"`
	Port     int    `yaml:"port"`
	Protocol string `yaml:"protocol"`
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
		c.Daemon.SocketPath = "/run/disco.sock"
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
	if c.TimeSync.MinSources == 0 {
		c.TimeSync.MinSources = 2
	}
	if c.TimeSync.MaxSourceSpread == 0 {
		c.TimeSync.MaxSourceSpread = 100 * time.Millisecond
	}
	if c.TimeSync.MaxStaleAge == 0 {
		c.TimeSync.MaxStaleAge = 30 * time.Second
	}
	if c.TimeSync.MaxDispersion == 0 {
		c.TimeSync.MaxDispersion = 1.0
	}
	if c.TimeSync.StepThreshold == 0 {
		c.TimeSync.StepThreshold = 128 * time.Millisecond
	}
	if c.TimeSync.SlewThreshold == 0 {
		c.TimeSync.SlewThreshold = 500 * time.Microsecond
	}
	if c.TimeSync.PollInterval == 0 {
		c.TimeSync.PollInterval = 60 * time.Second
	}
	if c.Health.GracePeriod == 0 {
		c.Health.GracePeriod = 60 * time.Second
	}
	if c.Health.ExpireAfter == 0 {
		c.Health.ExpireAfter = 3600 * time.Second
	}
	if c.Health.CleanupInterval == 0 {
		c.Health.CleanupInterval = 30 * time.Second
	}
	if c.DNS.Port == 0 {
		c.DNS.Port = 53
	}
	if c.DNS.Domain == "" {
		c.DNS.Domain = "disco"
	}
	if len(c.DNS.BindAddresses) == 0 {
		c.DNS.BindAddresses = []string{"0.0.0.0"}
	}
	if c.DNS.TTLHealthy == 0 {
		c.DNS.TTLHealthy = 30
	}
	if c.DNS.TTLStale == 0 {
		c.DNS.TTLStale = 10
	}
}
