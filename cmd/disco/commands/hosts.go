package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/offline-lab/disco/cmd/disco/internal/cli"
	"github.com/offline-lab/disco/internal/nss"
	"github.com/spf13/cobra"
)

var hostsCmd = &cobra.Command{
	Use:   "hosts",
	Short: "List and manage discovered hosts",
	Long: `List all discovered hosts with their health status, or manage individual hosts.

Health status can be:
  healthy  - Seen recently (within grace period)
  stale    - Not seen recently but not expired  
  lost     - Expired, will be removed
  static   - Defined in config, never expires`,
	Args: cobra.NoArgs,
	Run:  listHosts,
}

var hostsShowCmd = &cobra.Command{
	Use:   "show <hostname>",
	Short: "Show detailed information about a host",
	Args:  cobra.ExactArgs(1),
	Run:   showHost,
}

var hostsForgetCmd = &cobra.Command{
	Use:   "forget <hostname>",
	Short: "Remove host from cache",
	Long:  "Remove a host from the daemon's cache. It will be rediscovered if it broadcasts again.",
	Args:  cobra.ExactArgs(1),
	Run:   forgetHost,
}

var hostsMarkLostCmd = &cobra.Command{
	Use:   "mark-lost <hostname>",
	Short: "Mark host as lost",
	Long:  "Manually mark a host as lost. It will be removed during the next cleanup cycle.",
	Args:  cobra.ExactArgs(1),
	Run:   markLostHost,
}

func init() {
	rootCmd.AddCommand(hostsCmd)
	hostsCmd.AddCommand(hostsShowCmd)
	hostsCmd.AddCommand(hostsForgetCmd)
	hostsCmd.AddCommand(hostsMarkLostCmd)
}

func listHosts(cmd *cobra.Command, args []string) {
	client := cli.NewDaemonClient(getSocketPath())
	response, err := client.Query(&nss.Query{
		Type:      nss.HostsList,
		RequestID: cli.GenerateRequestID("hosts"),
	})
	checkError(err)

	response, err = cli.HandleResponse(response, err)
	checkError(err)

	if outputJSON {
		checkError(cli.OutputJSON(response.Hosts))
		return
	}

	if len(response.Hosts) == 0 {
		fmt.Println("No hosts discovered")
		return
	}

	table := cli.NewTable("ID", "HOSTNAME", "ADDRESSES", "STATUS", "SERVICES", "LAST SEEN")
	for _, h := range response.Hosts {
		id := h.MachineID
		if id == "" {
			id = "-"
		} else if len(id) > 8 {
			id = id[:8]
		}

		addrs := cli.Truncate(cli.JoinStrings(h.Addresses, ", "), 34)

		services := make([]string, 0, len(h.Services))
		for svc := range h.Services {
			services = append(services, svc)
		}
		svcStr := cli.Truncate(cli.JoinStrings(services, ", "), 18)
		if svcStr == "" {
			svcStr = "-"
		}

		status := cli.ColorizeStatus(h.Status)

		table.AddRow(id, h.Hostname, addrs, status, svcStr, h.LastSeenAgo)
	}
	table.Print()
}

func showHost(cmd *cobra.Command, args []string) {
	name := args[0]

	// Validate: must be a hostname, a full 32-char machine ID, or a hex prefix.
	isHex := cli.IsHexPrefix(name)
	isFull := cli.ValidateHexKey(name, 32) == nil
	if !isHex && !isFull {
		if err := cli.ValidateHostname(name); err != nil {
			checkError(fmt.Errorf("invalid hostname or machine ID: %w", err))
		}
	}

	client := cli.NewDaemonClient(getSocketPath())

	// First: exact lookup by hostname or full machine ID.
	response, err := client.Query(&nss.Query{
		Type:      nss.HostsShow,
		Name:      name,
		RequestID: cli.GenerateRequestID("host"),
	})
	checkError(err)

	response, err = cli.HandleResponse(response, err)
	checkError(err)

	if response.Type != nss.ResponseNotFound && len(response.Hosts) > 0 {
		printHostDetail(response.Hosts[0])
		return
	}

	// Fallback: prefix match against all machine IDs when a short hex string was given.
	if !isHex {
		cli.Fatal(fmt.Sprintf("Host not found: %s", name), nil, cli.ExitError)
	}

	listResp, err := client.Query(&nss.Query{
		Type:      nss.HostsList,
		RequestID: cli.GenerateRequestID("host-prefix"),
	})
	checkError(err)
	listResp, err = cli.HandleResponse(listResp, err)
	checkError(err)

	var matches []nss.HostHealth
	for _, h := range listResp.Hosts {
		if strings.HasPrefix(h.MachineID, name) {
			matches = append(matches, h)
		}
	}

	switch len(matches) {
	case 0:
		cli.Fatal(fmt.Sprintf("Host not found: %s", name), nil, cli.ExitError)
	case 1:
		printHostDetail(matches[0])
	default:
		fmt.Fprintf(os.Stderr, "Ambiguous prefix %q matches %d hosts — use more characters:\n", name, len(matches))
		for _, h := range matches {
			short := h.MachineID
			if len(short) > 8 {
				short = short[:8]
			}
			fmt.Fprintf(os.Stderr, "  %s  %s\n", short, h.Hostname)
		}
		os.Exit(int(cli.ExitError))
	}
}

func printHostDetail(h nss.HostHealth) {
	fmt.Printf("Hostname:    %s\n", h.Hostname)
	if h.MachineID != "" {
		fmt.Printf("Machine ID:  %s\n", h.MachineID)
	}
	fmt.Printf("Addresses:   %s\n", cli.JoinStrings(h.Addresses, ", "))
	fmt.Printf("Status:      %s\n", cli.ColorizeStatus(h.Status))
	fmt.Printf("Last Seen:   %s\n", h.LastSeenAgo)
	fmt.Printf("Static:      %v\n", h.IsStatic)

	if len(h.Services) > 0 {
		fmt.Println("\nServices:")
		for svc, proto := range h.Services {
			fmt.Printf("  - %s (%s)\n", svc, proto)
		}
	}
}

func forgetHost(cmd *cobra.Command, args []string) {
	hostname := args[0]

	if err := cli.ValidateHostname(hostname); err != nil {
		checkError(fmt.Errorf("invalid hostname: %w", err))
	}

	client := cli.NewDaemonClient(getSocketPath())
	response, err := client.Query(&nss.Query{
		Type:      nss.HostsForget,
		Name:      hostname,
		RequestID: cli.GenerateRequestID("forget"),
	})
	checkError(err)

	_, err = cli.HandleResponse(response, err)
	checkError(err)

	fmt.Printf("Host forgotten: %s\n", hostname)
}

func markLostHost(cmd *cobra.Command, args []string) {
	hostname := args[0]

	if err := cli.ValidateHostname(hostname); err != nil {
		checkError(fmt.Errorf("invalid hostname: %w", err))
	}

	client := cli.NewDaemonClient(getSocketPath())
	response, err := client.Query(&nss.Query{
		Type:      nss.HostsMarkLost,
		Name:      hostname,
		RequestID: cli.GenerateRequestID("lost"),
	})
	checkError(err)

	_, err = cli.HandleResponse(response, err)
	checkError(err)

	fmt.Printf("Host marked as lost: %s\n", hostname)
}

