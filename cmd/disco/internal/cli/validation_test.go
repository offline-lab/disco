package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateHostname(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		wantErr  bool
	}{
		{"empty", "", true},
		{"too long", string(make([]byte, 254)), true},
		{"valid simple", "web1", false},
		{"valid with dots", "web1.example.com", false},
		{"valid numbers", "node123", false},
		{"valid hyphens", "web-server-1", false},
		{"invalid chars", "web_1", true},
		{"starts with hyphen", "-web1", true},
		{"ends with hyphen", "web1-", true},
		{"label too long", string(make([]byte, 64)) + ".com", true},
		{"max length with dots", "a" + strings.Repeat(".a", 62), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHostname(tt.hostname)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHostname(%q) error = %v, wantErr %v", tt.hostname, err, tt.wantErr)
			}
		})
	}
}

func TestValidateHexKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"valid 64 chars", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", false},
		{"too short", "0123456789abcdef", true},
		{"too long", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdeff", true},
		{"invalid chars", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdeg", true},
		{"empty", "", true},
		{"uppercase valid", "0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHexKey(tt.key, HexKeyLength)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHexKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateBroadcastAddr(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{"valid broadcast", "255.255.255.255:5354", false},
		{"valid multicast", "224.0.0.1:5354", false},
		{"valid unicast", "192.168.1.10:5354", false},
		{"invalid format", "not-an-address", true},
		{"missing port", "192.168.1.10", true},
		{"invalid port", "192.168.1.10:abc", true},
		{"port too low", "192.168.1.10:0", true},
		{"port too high", "192.168.1.10:65536", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBroadcastAddr(tt.addr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBroadcastAddr(%q) error = %v, wantErr %v", tt.addr, err, tt.wantErr)
			}
		})
	}
}

func TestValidateConfigPath(t *testing.T) {
	tmpDir := t.TempDir()

	validConfig := filepath.Join(tmpDir, "valid.yaml")
	if err := os.WriteFile(validConfig, []byte("test: data"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		setup   func() string
		wantErr bool
	}{
		{
			name: "valid file",
			setup: func() string {
				return validConfig
			},
			wantErr: false,
		},
		{
			name: "non-existent file",
			setup: func() string {
				return filepath.Join(tmpDir, "nonexistent.yaml")
			},
			wantErr: true,
		},
		{
			name: "directory",
			setup: func() string {
				dir := filepath.Join(tmpDir, "testdir")
				if err := os.Mkdir(dir, 0755); err != nil && !os.IsExist(err) {
					return ""
				}
				return dir
			},
			wantErr: true,
		},
		{
			name: "relative path",
			setup: func() string {
				relPath := filepath.Join("..", "testdata", "config.yaml")
				return relPath
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup()
			err := ValidateConfigPath(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfigPath(%q) error = %v, wantErr %v", path, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"valid port", 8080, false},
		{"min port", 1, false},
		{"max port", 65535, false},
		{"too low", 0, true},
		{"too high", 65536, true},
		{"negative", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePort(tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePort(%d) error = %v, wantErr %v", tt.port, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePingTarget(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		wantErr bool
	}{
		{"valid hostname", "web1", false},
		{"valid IP", "192.168.1.10", false},
		{"valid IPv6", "::1", false},
		{"empty", "", true},
		{"too long", string(make([]byte, 254)), true},
		{"valid FQDN", "web1.example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePingTarget(tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePingTarget(%q) error = %v, wantErr %v", tt.target, err, tt.wantErr)
			}
		})
	}
}

func TestValidateServiceName(t *testing.T) {
	tests := []struct {
		name    string
		service string
		wantErr bool
	}{
		{"valid simple", "www", false},
		{"valid with numbers", "smtp25", false},
		{"valid with hyphens", "web-server", false},
		{"empty", "", true},
		{"too long", string(make([]byte, 64)), true},
		{"starts with number", "1web", true},
		{"uppercase", "WWW", true},
		{"underscore", "web_server", true},
		{"starts with hyphen", "-web", true},
		{"ends with hyphen", "web-", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServiceName(tt.service)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateServiceName(%q) error = %v, wantErr %v", tt.service, err, tt.wantErr)
			}
		})
	}
}
