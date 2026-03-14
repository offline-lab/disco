package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/offline-lab/disco/cmd/disco/internal/cli"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start [flags]",
	Short: "Start the disco daemon",
	Long: `Start the disco-daemon binary with specified flags.

Only specific flags are allowed for security:
  -config <path>   Path to configuration file
  -v, -verbose     Enable verbose logging
  -help            Show daemon help

The daemon binary is searched in:
  1. Same directory as the disco CLI
  2. System PATH`,
	Args: cobra.ArbitraryArgs,
	Run:  startDaemon,
}

var allowedDaemonFlags = map[string]bool{
	"-config":  true,
	"-v":       true,
	"-verbose": true,
	"-help":    true,
	"--help":   true,
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func startDaemon(cmd *cobra.Command, args []string) {
	var validatedArgs []string

	for i := 0; i < len(args); i++ {
		flag := args[i]

		if !allowedDaemonFlags[flag] {
			cli.Error(fmt.Sprintf("disallowed flag: %s", flag), nil)
			fmt.Fprintln(os.Stderr, "\nAllowed flags:")
			fmt.Fprintln(os.Stderr, "  -config <path>   Path to configuration file")
			fmt.Fprintln(os.Stderr, "  -v, -verbose     Enable verbose logging")
			fmt.Fprintln(os.Stderr, "  -help            Show daemon help")
			os.Exit(1)
		}

		validatedArgs = append(validatedArgs, flag)

		if flag == "-config" && i+1 < len(args) {
			configPath := args[i+1]
			if err := cli.ValidateConfigPath(configPath); err != nil {
				checkError(fmt.Errorf("invalid config path: %w", err))
			}
			validatedArgs = append(validatedArgs, configPath)
			i++
		}
	}

	var daemonPath string

	execPath, err := os.Executable()
	if err == nil {
		daemonPath = filepath.Join(filepath.Dir(execPath), "disco-daemon")
		if _, err := os.Stat(daemonPath); os.IsNotExist(err) {
			daemonPath = ""
		}
	}

	if daemonPath == "" {
		daemonPath = "disco-daemon"
	}

	daemonCmd := exec.Command(daemonPath, validatedArgs...)
	daemonCmd.Stdin = os.Stdin
	daemonCmd.Stdout = os.Stdout
	daemonCmd.Stderr = os.Stderr

	if err := daemonCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
		checkError(err)
	}
}
