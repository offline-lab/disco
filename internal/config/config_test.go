package config

import (
	"testing"
	"time"
)

func TestConfig_Validate_Daemon(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "Valid daemon config",
			config: &Config{
				Daemon: DaemonConfig{
					SocketPath:        "/run/disco.sock",
					BroadcastInterval: 30 * time.Second,
					RecordTTL:         3600 * time.Second,
				},
				Network: NetworkConfig{
					BroadcastAddr:    "255.255.255.255:5353",
					MaxBroadcastRate: 10,
				},
				Discovery: DiscoveryConfig{
					Enabled:      true,
					ScanInterval: 60 * time.Second,
					ServicePortMapping: map[string][]int{
						"www":  {80, 443},
						"smtp": {25},
					},
				},
				Security: SecurityConfig{
					Enabled: false,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "text",
				},
			},
			wantErr: false,
		},
		{
			name: "Missing socket path",
			config: &Config{
				Daemon: DaemonConfig{
					SocketPath:        "",
					BroadcastInterval: 30 * time.Second,
					RecordTTL:         3600 * time.Second,
				},
			},
			wantErr: true,
		},
		{
			name: "Non-absolute socket path",
			config: &Config{
				Daemon: DaemonConfig{
					SocketPath:        "relative/path.sock",
					BroadcastInterval: 30 * time.Second,
					RecordTTL:         3600 * time.Second,
				},
			},
			wantErr: true,
		},
		{
			name: "Broadcast interval too short",
			config: &Config{
				Daemon: DaemonConfig{
					SocketPath:        "/run/disco.sock",
					BroadcastInterval: 1 * time.Second,
					RecordTTL:         3600 * time.Second,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate_Network(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "Valid network config",
			config: &Config{
				Daemon: DaemonConfig{
					SocketPath:        "/run/disco.sock",
					BroadcastInterval: 30 * time.Second,
					RecordTTL:         3600 * time.Second,
				},
				Network: NetworkConfig{
					BroadcastAddr:    "255.255.255.255:5353",
					MaxBroadcastRate: 10,
				},
			},
			wantErr: false,
		},
		{
			name: "Missing broadcast address",
			config: &Config{
				Daemon: DaemonConfig{
					SocketPath:        "/run/disco.sock",
					BroadcastInterval: 30 * time.Second,
					RecordTTL:         3600 * time.Second,
				},
				Network: NetworkConfig{
					BroadcastAddr: "",
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid broadcast address format",
			config: &Config{
				Daemon: DaemonConfig{
					SocketPath:        "/run/disco.sock",
					BroadcastInterval: 30 * time.Second,
					RecordTTL:         3600 * time.Second,
				},
				Network: NetworkConfig{
					BroadcastAddr: "invalid-address",
				},
			},
			wantErr: true,
		},
		{
			name: "Max broadcast rate too low",
			config: &Config{
				Daemon: DaemonConfig{
					SocketPath:        "/run/disco.sock",
					BroadcastInterval: 30 * time.Second,
					RecordTTL:         3600 * time.Second,
				},
				Network: NetworkConfig{
					BroadcastAddr:    "255.255.255.255:5353",
					MaxBroadcastRate: 0,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate_Discovery(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "Valid discovery config",
			config: &Config{
				Daemon: DaemonConfig{
					SocketPath:        "/run/disco.sock",
					BroadcastInterval: 30 * time.Second,
					RecordTTL:         3600 * time.Second,
				},
				Discovery: DiscoveryConfig{
					Enabled:      true,
					ScanInterval: 60 * time.Second,
					ServicePortMapping: map[string][]int{
						"www":  {80, 443},
						"smtp": {25},
					},
				},
				Network: NetworkConfig{
					BroadcastAddr:    "255.255.255.255:5353",
					MaxBroadcastRate: 10,
				},
			},
			wantErr: false,
		},
		{
			name: "Scan interval too short",
			config: &Config{
				Daemon: DaemonConfig{
					SocketPath:        "/run/disco.sock",
					BroadcastInterval: 30 * time.Second,
					RecordTTL:         3600 * time.Second,
				},
				Discovery: DiscoveryConfig{
					Enabled:      true,
					ScanInterval: 1 * time.Second,
				},
			},
			wantErr: true,
		},
		{
			name: "Empty service port mapping",
			config: &Config{
				Daemon: DaemonConfig{
					SocketPath:        "/run/disco.sock",
					BroadcastInterval: 30 * time.Second,
					RecordTTL:         3600 * time.Second,
				},
				Discovery: DiscoveryConfig{
					Enabled:            true,
					ServicePortMapping: map[string][]int{},
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid port number",
			config: &Config{
				Daemon: DaemonConfig{
					SocketPath:        "/run/disco.sock",
					BroadcastInterval: 30 * time.Second,
					RecordTTL:         3600 * time.Second,
				},
				Discovery: DiscoveryConfig{
					Enabled:      true,
					ScanInterval: 60 * time.Second,
					ServicePortMapping: map[string][]int{
						"www": {99999},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_SetDefaults(t *testing.T) {
	cfg := &Config{}

	cfg.SetDefaults()

	if cfg.Daemon.SocketPath != "/run/disco.sock" {
		t.Errorf("Expected default socket path /run/disco.sock, got %s", cfg.Daemon.SocketPath)
	}

	if cfg.Daemon.BroadcastInterval != 30*time.Second {
		t.Errorf("Expected default broadcast interval 30s, got %v", cfg.Daemon.BroadcastInterval)
	}

	if cfg.Daemon.RecordTTL != 3600*time.Second {
		t.Errorf("Expected default record TTL 3600s, got %v", cfg.Daemon.RecordTTL)
	}

	if cfg.Network.BroadcastAddr != "255.255.255.255:5353" {
		t.Errorf("Expected default broadcast addr 255.255.255.255:5353, got %s", cfg.Network.BroadcastAddr)
	}

	if cfg.Logging.Level != "info" {
		t.Errorf("Expected default log level info, got %s", cfg.Logging.Level)
	}
}
