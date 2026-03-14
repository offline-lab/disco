package commands

import (
	"fmt"
	"os"

	"github.com/offline-lab/disco/cmd/disco/internal/cli"
	"github.com/spf13/cobra"
)

var (
	socketPath string
	outputJSON bool
)

var rootCmd = &cobra.Command{
	Use:   "disco",
	Short: "Query and manage Disco daemon",
	Long: `Disco is a unified CLI tool for managing and querying the Disco daemon.

It provides commands for host discovery, service management, time synchronization,
configuration validation, and cryptographic key management.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&socketPath, "socket", "s", cli.DefaultSocketPath, "Path to daemon socket")
	rootCmd.PersistentFlags().BoolVarP(&outputJSON, "json", "j", false, "Output in JSON format")
}

func getSocketPath() string {
	if socketPath != "" {
		return socketPath
	}
	if env := os.Getenv("DISCO_SOCKET"); env != "" {
		return env
	}
	return cli.DefaultSocketPath
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
