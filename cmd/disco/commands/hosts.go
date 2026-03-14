package commands

import (
	"fmt"

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

	table := cli.NewTable("HOSTNAME", "ADDRESSES", "STATUS", "SERVICES", "LAST SEEN")
	for _, h := range response.Hosts {
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

		table.AddRow(h.Hostname, addrs, status, svcStr, h.LastSeenAgo)
	}
	table.Print()
}

func showHost(cmd *cobra.Command, args []string) {
	hostname := args[0]

	if err := cli.ValidateHostname(hostname); err != nil {
		checkError(fmt.Errorf("invalid hostname: %w", err))
	}

	client := cli.NewDaemonClient(getSocketPath())
	response, err := client.Query(&nss.Query{
		Type:      nss.HostsShow,
		Name:      hostname,
		RequestID: cli.GenerateRequestID("host"),
	})
	checkError(err)

	response, err = cli.HandleResponse(response, err)
	checkError(err)

	if response.Type == nss.ResponseNotFound || len(response.Hosts) == 0 {
		cli.Fatal(fmt.Sprintf("Host not found: %s", hostname), nil, cli.ExitError)
	}

	h := response.Hosts[0]

	fmt.Printf("Hostname:    %s\n", h.Hostname)
	fmt.Printf("Addresses:   %s\n", cli.JoinStrings(h.Addresses, ", "))
	fmt.Printf("Status:      %s\n", cli.ColorizeStatus(h.Status))
	fmt.Printf("Last Seen:   %s\n", h.LastSeenAgo)
	fmt.Printf("Static:      %v\n", h.IsStatic)

	if len(h.Services) > 0 {
		fmt.Println("\nServices:")
		for name, proto := range h.Services {
			fmt.Printf("  - %s (%s)\n", name, proto)
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

	response, err = cli.HandleResponse(response, err)
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

	response, err = cli.HandleResponse(response, err)
	checkError(err)

	fmt.Printf("Host marked as lost: %s\n", hostname)
}

func JoinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
