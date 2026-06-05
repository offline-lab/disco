package commands

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/offline-lab/disco/cmd/disco/internal/cli"
	"github.com/offline-lab/disco/internal/nss"
	"github.com/spf13/cobra"
)

var (
	pingPort     int
	pingCount    int
	pingInterval time.Duration
	pingVerbose  bool
)

var pingCmd = &cobra.Command{
	Use:   "ping [target]",
	Short: "Ping a discovered host",
	Long: `Send UDP ping requests to check if a daemon is responding.

This is a simple network ping tool for testing daemon connectivity.
It sends UDP ping requests to check if the daemon is running on other nodes
and can test network latency and packet loss.`,
	Args: cobra.ExactArgs(1),
	RunE: runPing,
}

func init() {
	rootCmd.AddCommand(pingCmd)

	pingCmd.Flags().IntVarP(&pingPort, "port", "p", 5354, "Target port")
	pingCmd.Flags().IntVarP(&pingCount, "count", "n", 4, "Number of pings (1-10)")
	pingCmd.Flags().DurationVarP(&pingInterval, "interval", "i", 1*time.Second, "Time between pings (min 100ms)")
	pingCmd.Flags().BoolVarP(&pingVerbose, "verbose", "v", false, "Verbose output")
}

func runPing(cmd *cobra.Command, args []string) error {
	raw := args[0]

	if err := cli.ValidatePingTarget(raw); err != nil {
		return fmt.Errorf("invalid target: %w", err)
	}

	if err := validatePingArgs(); err != nil {
		return err
	}

	target := resolveTarget(raw)
	if target != raw && pingVerbose {
		fmt.Printf("Resolved %q to %s\n", raw, target)
	}

	if pingVerbose {
		fmt.Printf("Pinging %s on port %d (%d times, %s interval)...\n\n",
			target, pingPort, pingCount, pingInterval)
	}

	successful, latencies := executePings(target)
	printPingResults(target, successful, latencies)

	if successful > 0 {
		return nil
	}
	return fmt.Errorf("host is down or unreachable")
}

// resolveTarget looks up the target in the daemon as a hostname, machine ID,
// or service name and returns the first matching IP address. Falls back to the
// original target if the daemon is unreachable or no match is found.
func resolveTarget(target string) string {
	if net.ParseIP(target) != nil {
		return target
	}

	client := cli.NewDaemonClient(getSocketPath())
	response, err := client.Query(&nss.Query{
		Type:      nss.HostsList,
		RequestID: cli.GenerateRequestID("ping"),
	})
	if err != nil {
		return target
	}
	response, err = cli.HandleResponse(response, err)
	if err != nil {
		return target
	}

	return matchTarget(target, response.Hosts)
}

// matchTarget resolves a target string against a list of hosts.
// Priority: exact hostname > exact machine ID > service name.
// Machine ID matching is exact (not prefix) to prevent collisions with hostnames.
func matchTarget(target string, hosts []nss.HostHealth) string {
	for _, h := range hosts {
		if strings.EqualFold(h.Hostname, target) && len(h.Addresses) > 0 {
			return h.Addresses[0]
		}
	}
	for _, h := range hosts {
		if h.MachineID != "" && h.MachineID == strings.ToLower(target) && len(h.Addresses) > 0 {
			return h.Addresses[0]
		}
	}
	for _, h := range hosts {
		if _, ok := h.Services[target]; ok && len(h.Addresses) > 0 {
			return h.Addresses[0]
		}
	}
	return target
}

func validatePingArgs() error {
	if pingCount < 1 || pingCount > 10 {
		return fmt.Errorf("count must be between 1 and 10")
	}

	if pingInterval < 100*time.Millisecond {
		return fmt.Errorf("interval must be at least 100ms")
	}

	return nil
}

func executePings(target string) (successful int, latencies []time.Duration) {
	latencies = make([]time.Duration, 0, pingCount)
	const timeout = 2 * time.Second

	for i := 0; i < pingCount; i++ {
		start := time.Now()
		address := net.JoinHostPort(target, fmt.Sprintf("%d", pingPort))
		deadline := time.Now().Add(timeout)

		conn, err := net.DialTimeout("udp", address, timeout)
		if err != nil {
			if pingVerbose {
				cli.PrintError("[%d] ERROR: %v\n", i+1, err)
			}
			time.Sleep(pingInterval)
			continue
		}

		_ = conn.SetDeadline(deadline)

		_, err = conn.Write([]byte("PING"))
		if err != nil {
			if pingVerbose {
				cli.PrintError("[%d] ERROR: %v\n", i+1, err)
			}
			_ = conn.Close()
			time.Sleep(pingInterval)
			continue
		}

		if err := conn.SetDeadline(deadline); err != nil {
			if pingVerbose {
				cli.PrintError("[%d] ERROR: %v\n", i+1, err)
			}
			_ = conn.Close()
			time.Sleep(pingInterval)
			continue
		}
		buf := make([]byte, 1024)
		_, err = conn.Read(buf)
		_ = conn.Close()

		if err != nil {
			if pingVerbose {
				cli.PrintError("[%d] ERROR: %v\n", i+1, err)
			}
		} else if strings.HasPrefix(string(buf), "PONG") {
			latency := time.Since(start)
			latencies = append(latencies, latency)
			successful++
			if pingVerbose {
				cli.PrintSuccess("[%d] PONG from %s (latency: %v)\n", i+1, target, latency)
			}
		} else {
			if pingVerbose {
				cli.PrintWarning("[%d] Unexpected response: %s\n", i+1, string(buf))
			}
		}

		time.Sleep(pingInterval)
	}

	return successful, latencies
}

func printPingResults(target string, successful int, latencies []time.Duration) {
	fmt.Printf("\n=== Results ===\n")
	fmt.Printf("Target:    %s:%d\n", target, pingPort)
	fmt.Printf("Success:   %d/%d\n", successful, pingCount)

	if successful > 0 {
		var total time.Duration
		for _, l := range latencies {
			total += l
		}
		avgLatency := total / time.Duration(successful)
		fmt.Printf("Latency:   %v (avg over %d successes)\n", avgLatency, successful)

		if successful == pingCount {
			cli.PrintSuccess("Status:     UP\n")
		} else {
			lossRate := float64(pingCount-successful) / float64(pingCount) * 100
			cli.PrintWarning("Status:     PARTIAL (%.0f%% loss)\n", lossRate)
		}
	} else {
		cli.PrintError("Status:     DOWN\n")
	}
}
