package config

import (
	"fmt"
	"net"
	"path/filepath"
	"time"
)

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if err := c.validateDaemon(); err != nil {
		return fmt.Errorf("daemon config invalid: %w", err)
	}

	if err := c.validateNetwork(); err != nil {
		return fmt.Errorf("network config invalid: %w", err)
	}

	if err := c.validateDiscovery(); err != nil {
		return fmt.Errorf("discovery config invalid: %w", err)
	}

	if err := c.validateSecurity(); err != nil {
		return fmt.Errorf("security config invalid: %w", err)
	}

	if err := c.validateLogging(); err != nil {
		return fmt.Errorf("logging config invalid: %w", err)
	}

	return nil
}

func (c *Config) validateDaemon() error {
	if c.Daemon.SocketPath == "" {
		return fmt.Errorf("socket_path is required")
	}

	if !filepath.IsAbs(c.Daemon.SocketPath) {
		return fmt.Errorf("socket_path must be absolute path")
	}

	if c.Daemon.BroadcastInterval < 5*time.Second {
		return fmt.Errorf("broadcast_interval must be at least 5 seconds")
	}

	if c.Daemon.BroadcastInterval > 1*time.Hour {
		return fmt.Errorf("broadcast_interval cannot exceed 1 hour")
	}

	if c.Daemon.RecordTTL < 60*time.Second {
		return fmt.Errorf("record_ttl must be at least 60 seconds")
	}

	if c.Daemon.RecordTTL > 24*time.Hour {
		return fmt.Errorf("record_ttl cannot exceed 24 hours")
	}

	return nil
}

func (c *Config) validateNetwork() error {
	if c.Network.BroadcastAddr == "" {
		return fmt.Errorf("broadcast_addr is required")
	}

	host, port, err := net.SplitHostPort(c.Network.BroadcastAddr)
	if err != nil {
		return fmt.Errorf("invalid broadcast_addr: %w", err)
	}

	if host == "" {
		return fmt.Errorf("broadcast_addr missing host")
	}

	if port == "" {
		return fmt.Errorf("broadcast_addr missing port")
	}

	if c.Network.MaxBroadcastRate < 1 {
		return fmt.Errorf("max_broadcast_rate must be at least 1")
	}

	if c.Network.MaxBroadcastRate > 100 {
		return fmt.Errorf("max_broadcast_rate cannot exceed 100")
	}

	return nil
}

func (c *Config) validateDiscovery() error {
	if !c.Discovery.Enabled {
		return nil
	}

	if c.Discovery.ScanInterval < 10*time.Second {
		return fmt.Errorf("scan_interval must be at least 10 seconds")
	}

	if c.Discovery.ScanInterval > 10*time.Minute {
		return fmt.Errorf("scan_interval cannot exceed 10 minutes")
	}

	if len(c.Discovery.ServicePortMapping) == 0 {
		return fmt.Errorf("service_port_mapping cannot be empty when discovery is enabled")
	}

	for service, ports := range c.Discovery.ServicePortMapping {
		if service == "" {
			return fmt.Errorf("service name cannot be empty")
		}

		if len(ports) == 0 {
			return fmt.Errorf("service %s has no ports configured", service)
		}

		for _, port := range ports {
			if port < 1 || port > 65535 {
				return fmt.Errorf("invalid port %d for service %s", port, service)
			}
		}
	}

	return nil
}

func (c *Config) validateSecurity() error {
	if !c.Security.Enabled {
		return nil
	}

	if c.Security.KeyPath == "" {
		return fmt.Errorf("key_path is required when security is enabled")
	}

	if c.Security.TrustedPeers == "" {
		c.Security.TrustedPeers = "/etc/nss-daemon/trusted_peers.json"
	}

	return nil
}

func (c *Config) validateLogging() error {
	if c.Logging.Level == "" {
		return nil
	}

	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
	}

	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s (valid: debug, info, warn, error, fatal)", c.Logging.Level)
	}

	if c.Logging.File != "" && !filepath.IsAbs(c.Logging.File) {
		return fmt.Errorf("log_file must be absolute path")
	}

	return nil
}
