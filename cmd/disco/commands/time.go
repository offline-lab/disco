package commands

import (
	"fmt"
	"time"

	"github.com/offline-lab/disco/internal/client"
	"github.com/spf13/cobra"
)

var timeCmd = &cobra.Command{
	Use:   "time",
	Short: "Show time synchronization status",
	Long:  "Display the current time synchronization status from the Disco daemon.",
	Args:  cobra.NoArgs,
	Run:   showTimeStatus,
}

var timesetCmd = &cobra.Command{
	Use:   "timeset",
	Short: "Force time synchronization",
	Long:  "Force the daemon to immediately synchronize time from GPS sources.",
	Args:  cobra.NoArgs,
	Run:   forceTimeUpdate,
}

var (
	forceUpdate   bool
	allowBackward bool
	verbose       bool
	timeoutDur    time.Duration
)

func init() {
	rootCmd.AddCommand(timeCmd)
	rootCmd.AddCommand(timesetCmd)

	timesetCmd.Flags().BoolVarP(&forceUpdate, "force", "f", false, "Force immediate time update")
	timesetCmd.Flags().BoolVarP(&allowBackward, "backward", "b", false, "Allow stepping clock backward")
	timesetCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	timesetCmd.Flags().DurationVarP(&timeoutDur, "timeout", "t", 30*time.Second, "Timeout for operations")
}

func showTimeStatus(cmd *cobra.Command, args []string) {
	c := client.NewDaemonClient(getSocketPath())
	status, err := c.GetTimeStatus()
	if err != nil {
		checkError(fmt.Errorf("failed to get time status: %w", err))
	}

	syncStr := "NO"
	if status.Synced {
		syncStr = "YES"
	}
	fmt.Printf("Synced: %s\n", syncStr)
	fmt.Printf("Sources: %d\n", status.SourceCount)
	fmt.Printf("Offset: %.6f seconds\n", status.LastOffset)
	if status.LastSyncTime != "" {
		fmt.Printf("Last sync: %s\n", status.LastSyncTime)
	}
	if status.LastError != "" {
		fmt.Printf("Error: %s\n", status.LastError)
	}
}

func forceTimeUpdate(cmd *cobra.Command, args []string) {
	c := client.NewDaemonClient(getSocketPath()).WithTimeout(timeoutDur)

	if !forceUpdate {
		status, err := c.GetTimeStatus()
		if err != nil {
			checkError(fmt.Errorf("failed to get time status: %w", err))
		}
		printTimeStatusResult(status)
		return
	}

	if verbose {
		fmt.Printf("Sending force update request...\n")
		fmt.Printf("  Allow backward: %v\n", allowBackward)
	}

	result, err := c.ForceTimeUpdate(allowBackward)
	if err != nil {
		checkError(fmt.Errorf("failed to force time update: %w", err))
	}

	if !result.Success {
		errMsg := result.Error
		if errMsg == "" {
			errMsg = "unknown error"
		}
		checkError(fmt.Errorf("force update failed: %s", errMsg))
	}

	fmt.Printf("Time adjusted successfully\n")
	fmt.Printf("  Method: %s\n", result.Method)
	fmt.Printf("  Offset: %.6f seconds\n", result.Offset)
	fmt.Printf("  Sources: %d\n", result.SourceCount)
}

func printTimeStatusResult(s *client.TimeStatus) {
	syncStr := "NO"
	if s.Synced {
		syncStr = "YES"
	}
	fmt.Printf("Synced: %s\n", syncStr)
	fmt.Printf("Sources: %d\n", s.SourceCount)
	fmt.Printf("Offset: %.6f seconds\n", s.LastOffset)
	if s.LastSyncTime != "" {
		fmt.Printf("Last sync: %s\n", s.LastSyncTime)
	}
	if s.LastError != "" {
		fmt.Printf("Error: %s\n", s.LastError)
	}
}
