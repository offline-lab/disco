package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/offline-lab/disco/internal/config"
	"github.com/offline-lab/disco/internal/daemon"
)

func printHelp() {
	fmt.Println("NSS Daemon - Lightweight name service for offline networks")
	fmt.Println()
	fmt.Println("WHAT:")
	fmt.Println("  NSS daemon provides automatic host and service discovery via UDP broadcast.")
	fmt.Println("  It integrates with glibc via NSS module for seamless name resolution.")
	fmt.Println("  No DNS server required - perfect for airgapped emergency networks.")
	fmt.Println()
	fmt.Println("WHY:")
	fmt.Println("  • Works without DNS servers or resolv.conf")
	fmt.Println("  • Automatic node discovery - zero configuration")
	fmt.Println("  • Service detection (www, smtp, ssh, etc.)")
	fmt.Println("  • Minimal resource footprint (<20MB RAM)")
	fmt.Println("  • Runs on embedded systems (Raspberry Pi)")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  nss-daemon [options]")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  --config PATH   Path to configuration file")
	fmt.Println("                   (default: /etc/nss-daemon/config.yaml)")
	fmt.Println("  --version       Show version and exit")
	fmt.Println("  --help, -h     Show this help message")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  nss-daemon")
	fmt.Println("  nss-daemon --config /custom/config.yaml")
	fmt.Println()
	fmt.Println("CLIENT TOOLS:")
	fmt.Println("  nss-query          Query daemon for hosts and services")
	fmt.Println("  nss-status          Show daemon status and cached records")
	fmt.Println("  nss-key             Manage security keys")
	fmt.Println("  nss-config-validate Validate configuration files")
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
	configPath := flag.String("config", "/etc/nss-daemon/config.yaml", "Path to configuration file")
	version := flag.Bool("version", false, "Show version and exit")
	help := flag.Bool("help", false, "Show this help message")
	flag.BoolVar(help, "h", false, "Show this help message")
	flag.Parse()

	if *help {
		printHelp()
	}

	if *version {
		fmt.Println("nss-daemon v0.1.0")
		fmt.Println("Build: " + buildInfo())
		os.Exit(0)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		os.Exit(1)
	}

	cfg.SetDefaults()

	if err := cfg.Validate(); err != nil {
		log.Printf("Configuration validation failed: %v", err)
		os.Exit(1)
	}

	d, err := daemon.New(cfg)
	if err != nil {
		log.Printf("Failed to create daemon: %v", err)
		os.Exit(1)
	}

	if err := d.Run(); err != nil {
		log.Printf("Daemon error: %v", err)
		os.Exit(1)
	}
}
