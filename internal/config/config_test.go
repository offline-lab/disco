package config

import (
	"os"
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

func TestLoad(t *testing.T) {
	content := `
daemon:
  socket_path: /tmp/test.sock
  broadcast_interval: 10s
  record_ttl: 1800s
network:
  broadcast_addr: "239.255.255.250:5353"
  max_broadcast_rate: 20
logging:
  level: debug
  format: json
`
	tmpFile := "/tmp/test-config.yaml"
	defer os.Remove(tmpFile) //nolint:errcheck

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Daemon.SocketPath != "/tmp/test.sock" {
		t.Errorf("SocketPath = %s, want /tmp/test.sock", cfg.Daemon.SocketPath)
	}
	if cfg.Daemon.BroadcastInterval != 10*time.Second {
		t.Errorf("BroadcastInterval = %v, want 10s", cfg.Daemon.BroadcastInterval)
	}
	if cfg.Network.BroadcastAddr != "239.255.255.250:5353" {
		t.Errorf("BroadcastAddr = %s, want 239.255.255.250:5353", cfg.Network.BroadcastAddr)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %s, want debug", cfg.Logging.Level)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Load() expected error for nonexistent file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	content := `
daemon:
  socket_path: [invalid yaml
`
	tmpFile := "/tmp/test-invalid.yaml"
	defer os.Remove(tmpFile) //nolint:errcheck

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	_, err := Load(tmpFile)
	if err == nil {
		t.Error("Load() expected error for invalid YAML")
	}
}
