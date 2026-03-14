package cli

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	MaxHostnameLength = 253
	MaxLabelLength    = 63
	HexKeyLength      = 64
)

func ValidateHostname(hostname string) error {
	if len(hostname) == 0 {
		return fmt.Errorf("hostname cannot be empty")
	}
	if len(hostname) > MaxHostnameLength {
		return fmt.Errorf("hostname too long (max %d characters)", MaxHostnameLength)
	}

	labelRegex := regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
	parts := strings.Split(hostname, ".")
	for _, part := range parts {
		if len(part) == 0 || len(part) > MaxLabelLength {
			return fmt.Errorf("hostname label must be 1-%d characters", MaxLabelLength)
		}

		if !labelRegex.MatchString(strings.ToLower(part)) {
			return fmt.Errorf("hostname contains invalid characters")
		}
	}

	return nil
}

func ValidateHexKey(key string, expectedLen int) error {
	if len(key) != expectedLen {
		return fmt.Errorf("key must be %d hex characters, got %d", expectedLen, len(key))
	}

	matched, _ := regexp.MatchString(`^[0-9a-fA-F]+$`, key)
	if !matched {
		return fmt.Errorf("key must be hexadecimal")
	}

	return nil
}

func ValidateBroadcastAddr(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid address format: %w", err)
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("invalid IP address")
	}

	portNum, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid port number")
	}

	if portNum < 1 || portNum > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	return nil
}

func ValidateConfigPath(path string) error {
	// Resolve to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("cannot resolve path: %w", err)
	}

	// Check file exists and is accessible
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("cannot access config: %w", err)
	}

	// Must be regular file (not directory, device, etc.)
	if !info.Mode().IsRegular() {
		return fmt.Errorf("config must be a regular file")
	}

	// Check if world-writable (informational warning for airgapped env)
	if info.Mode().Perm()&0002 != 0 {
		fmt.Fprintf(os.Stderr, "Warning: Config file is world-writable\n")
	}

	return nil
}

func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

func ValidatePingTarget(target string) error {
	if len(target) == 0 {
		return fmt.Errorf("target cannot be empty")
	}
	if len(target) > MaxHostnameLength {
		return fmt.Errorf("target too long (max %d characters)", MaxHostnameLength)
	}

	// Check if it's a valid IP address
	if ip := net.ParseIP(target); ip != nil {
		return nil
	}

	// Otherwise validate as hostname
	return ValidateHostname(target)
}

func ValidateServiceName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("service name cannot be empty")
	}
	if len(name) > MaxLabelLength {
		return fmt.Errorf("service name too long (max %d characters)", MaxLabelLength)
	}

	// Service names should be simple identifiers (lowercase alphanumeric with hyphens)
	matched, _ := regexp.MatchString(`^[a-z][a-z0-9-]*[a-z0-9]$`, name)
	if !matched {
		return fmt.Errorf("service name must be lowercase alphanumeric with hyphens, starting with a letter")
	}

	return nil
}
