package commands

import (
	"fmt"
	"os"

	"github.com/offline-lab/disco/cmd/disco/internal/cli"
	"github.com/offline-lab/disco/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  "Configuration management commands for Disco daemon.",
}

var configValidateCmd = &cobra.Command{
	Use:   "validate <config-file>",
	Short: "Validate configuration file",
	Long:  "Validate a Disco daemon configuration file and display a summary.",
	Args:  cobra.ExactArgs(1),
	Run:   validateConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configValidateCmd)
}

func validateConfig(cmd *cobra.Command, args []string) {
	configPath := args[0]

	if err := cli.ValidateConfigPath(configPath); err != nil {
		checkError(fmt.Errorf("invalid config path: %w", err))
	}

	fmt.Printf("Validating configuration: %s\n", configPath)
	fmt.Println()

	cfg, err := config.Load(configPath)
	if err != nil {
		cli.Error("Failed to load config", err)
		os.Exit(1)
	}

	fmt.Println("✅ Configuration file loaded successfully")
	fmt.Println()

	cfg.SetDefaults()

	fmt.Println("Validating configuration...")
	fmt.Println()

	warnings, err := cfg.Validate()
	if err != nil {
		cli.Error("Configuration validation failed", err)
		os.Exit(1)
	}

	fmt.Println("✅ Configuration is valid")
	fmt.Println()

	if len(warnings) > 0 {
		fmt.Println("Warnings:")
		for _, w := range warnings {
			fmt.Printf("  ⚠️  %s: %s\n", w.Field, w.Message)
		}
		fmt.Println()
	}

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
		fmt.Printf("Log File:           stdout\n")
	}
	fmt.Println()
	fmt.Println("Service Port Mappings:")
	fmt.Println("─────────────────────────────────────────────")
	for service, ports := range cfg.Discovery.ServicePortMapping {
		fmt.Printf("  %-10s → %v\n", service+":", ports)
	}
}
