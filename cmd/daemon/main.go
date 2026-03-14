package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/offline-lab/disco/internal/config"
	"github.com/offline-lab/disco/internal/daemon"
	"github.com/offline-lab/disco/internal/logging"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func printHelp() {
	fmt.Println("Disco - Service Discovery Daemon for offline networks")
	fmt.Println()
	fmt.Println("WHAT:")
	fmt.Println("  Disco provides automatic host and service discovery via UDP broadcast.")
	fmt.Println("  It integrates with glibc via NSS module for seamless name resolution.")
	fmt.Println("  No DNS server required - perfect for airgapped emergency networks.")
	fmt.Println()
	fmt.Println("WHY:")
	fmt.Println("  • Works without DNS servers or resolv.conf")
	fmt.Println("  • Automatic node discovery - zero configuration")
	fmt.Println("  • Service detection (www, smtp, ssh, etc.)")
	fmt.Println("  • Time synchronization via GPS broadcasters")
	fmt.Println("  • Minimal resource footprint (<20MB RAM)")
	fmt.Println("  • Runs on embedded systems (Raspberry Pi)")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  disco-daemon [options]")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  --config PATH   Path to configuration file")
	fmt.Println("                   (default: /etc/disco/config.yaml)")
	fmt.Println("  --version       Show version and exit")
	fmt.Println("  --help, -h      Show this help message")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  disco-daemon")
	fmt.Println("  disco-daemon --config /custom/config.yaml")
	fmt.Println()
	fmt.Println("CLIENT TOOLS:")
	fmt.Println("  disco-query         Query daemon for hosts and services")
	fmt.Println("  disco-status        Show daemon status and cached records")
	fmt.Println("  disco-time          Show time synchronization status")
	fmt.Println("  disco-timeset       Force time synchronization")
	fmt.Println("  disco-key           Manage security keys")
	fmt.Println("  disco-config        Validate configuration files")
	fmt.Println()
	fmt.Println("For more information, visit: https://github.com/offline-lab/disco")
	os.Exit(0)
}

func buildInfo() string {
	return fmt.Sprintf("Go %s/%s, %s, %s",
		runtime.Version(),
		runtime.Compiler,
		runtime.GOOS,
		runtime.GOARCH)
}

func main() {
	configPath := flag.String("config", "/etc/disco/config.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version and exit")
	help := flag.Bool("help", false, "Show this help message")
	flag.BoolVar(help, "h", false, "Show this help message")
	flag.Parse()

	if *help {
		printHelp()
	}

	if *showVersion {
		fmt.Printf("disco-daemon %s\n", Version)
		fmt.Printf("  Commit:    %s\n", Commit)
		fmt.Printf("  Built:     %s\n", BuildTime)
		fmt.Printf("  Platform:  %s\n", buildInfo())
		os.Exit(0)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	cfg.SetDefaults()

	warnings, err := cfg.Validate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration validation failed: %v\n", err)
		os.Exit(1)
	}

	if err := setupLogging(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logging: %v\n", err)
		os.Exit(1)
	}

	for _, w := range warnings {
		logging.Warn(w.Message, map[string]interface{}{"field": w.Field})
	}

	d, err := daemon.New(cfg)
	if err != nil {
		logging.Error("Failed to create daemon", err, nil)
		os.Exit(1)
	}

	if err := d.Run(); err != nil {
		logging.Error("Daemon error", err, nil)
		os.Exit(1)
	}
}

func setupLogging(cfg *config.Config) error {
	level := parseLogLevel(cfg.Logging.Level)
	return logging.Setup(logging.Config{
		Level:  level,
		Format: cfg.Logging.Format,
		File:   cfg.Logging.File,
	})
}

func parseLogLevel(level string) logging.LogLevel {
	switch level {
	case "debug":
		return logging.DEBUG
	case "info":
		return logging.INFO
	case "warn":
		return logging.WARN
	case "error":
		return logging.ERROR
	case "fatal":
		return logging.FATAL
	default:
		return logging.INFO
	}
}
