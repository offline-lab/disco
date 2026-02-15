package main

import (
	"fmt"
	"os"
	"time"

	"github.com/offline-lab/disco/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: nss-config-validate <config-file>")
		fmt.Println()
		fmt.Println("Validates a nss-daemon configuration file.")
		os.Exit(1)
	}

	configPath := os.Args[1]

	fmt.Printf("Validating configuration: %s\n", configPath)
	fmt.Println()

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("❌ Failed to load config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Configuration file loaded successfully")
	fmt.Println()

	cfg.SetDefaults()

	fmt.Println("Validating configuration...")
	fmt.Println()

	if err := cfg.Validate(); err != nil {
		fmt.Printf("❌ Configuration validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Configuration is valid")
	fmt.Println()

	printConfigSummary(cfg)

	fmt.Println("✅ All checks passed. Configuration is ready to use.")
}

func printConfigSummary(cfg *config.Config) {
	fmt.Println("Configuration Summary:")
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("Socket Path:     %s\n", cfg.Daemon.SocketPath)
	fmt.Printf("Broadcast Interval: %v\n", cfg.Daemon.BroadcastInterval)
	fmt.Printf("Record TTL:        %v\n", cfg.Daemon.RecordTTL)
	fmt.Println()
	fmt.Printf("Broadcast Address: %s\n", cfg.Network.BroadcastAddr)
	fmt.Printf("Max Broadcast Rate: %d msg/sec\n", cfg.Network.MaxBroadcastRate)
	if len(cfg.Network.Interfaces) > 0 {
		fmt.Printf("Interfaces:        %v\n", cfg.Network.Interfaces)
	}
	fmt.Println()
	fmt.Printf("Discovery Enabled:  %v\n", cfg.Discovery.Enabled)
	fmt.Printf("Service Detection: %v\n", cfg.Discovery.DetectServices)
	fmt.Printf("Scan Interval:     %v\n", cfg.Discovery.ScanInterval)
	if cfg.Discovery.DetectServices {
		fmt.Printf("Service Mappings:  %d services\n", len(cfg.Discovery.ServicePortMapping))
	}
	fmt.Println()
	fmt.Printf("Security Enabled:   %v\n", cfg.Security.Enabled)
	fmt.Printf("Require Signed:     %v\n", cfg.Security.RequireSigned)
	fmt.Println()
	fmt.Printf("Log Level:          %s\n", cfg.Logging.Level)
	fmt.Printf("Log Format:         %s\n", cfg.Logging.Format)
	if cfg.Logging.File != "" {
		fmt.Printf("Log File:           %s\n", cfg.Logging.File)
	} else {
		fmt.Printf("Log File:           stdout")
	}
	fmt.Println()
	fmt.Println("Service Port Mappings:")
	fmt.Println("─────────────────────────────────────────────")
	for service, ports := range cfg.Discovery.ServicePortMapping {
		fmt.Printf("  %-10s → %v\n", service+":", ports)
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return d.String()
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", d/time.Minute)
	}
	return fmt.Sprintf("%dh", d/time.Hour)
}
