package commands

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/offline-lab/disco/cmd/disco/internal/cli"
	"github.com/offline-lab/disco/internal/nss"
	"github.com/spf13/cobra"
)

var (
	checkTimeout time.Duration
	checkVerbose bool
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check service reachability",
	Long: `Check which discovered services are actually reachable.

This command queries the daemon for all discovered hosts and their services,
then attempts to connect to each service to verify it's running.

Examples:
  disco check                    Check all services
  disco check --timeout 2s       Use 2 second timeout per service
  disco check -v                 Show detailed output`,
	Args: cobra.NoArgs,
	Run:  checkServices,
}

func init() {
	rootCmd.AddCommand(checkCmd)
	checkCmd.Flags().DurationVarP(&checkTimeout, "timeout", "t", 2*time.Second, "connection timeout per service")
	checkCmd.Flags().BoolVarP(&checkVerbose, "verbose", "v", false, "show detailed output")
}

type ServiceCheck struct {
	Host      string
	Address   string
	Service   string
	Protocol  string
	Port      int
	Reachable bool
	Error     error
}

func checkServices(cmd *cobra.Command, args []string) {
	client := cli.NewDaemonClient(getSocketPath())

	response, err := client.Query(&nss.Query{
		Type:      nss.HostsList,
		RequestID: cli.GenerateRequestID("check"),
	})
	checkError(err)

	response, err = cli.HandleResponse(response, err)
	checkError(err)

	if len(response.Hosts) == 0 {
		fmt.Println("No hosts discovered")
		return
	}

	var checks []ServiceCheck
	for _, h := range response.Hosts {
		for svcName, svcValue := range h.Services {
			port, _ := parseServiceAddr(svcValue)
			if port == 0 {
				continue
			}

			for _, addr := range h.Addresses {
				checks = append(checks, ServiceCheck{
					Host:     h.Hostname,
					Address:  addr,
					Service:  svcName,
					Protocol: "tcp",
					Port:     port,
				})
			}
		}
	}

	if len(checks) == 0 {
		fmt.Println("No TCP services found to check")
		return
	}

	if checkVerbose {
		fmt.Printf("Checking %d service(s) with %s timeout...\n\n", len(checks), checkTimeout)
	}

	for i := range checks {
		checks[i] = checkService(&checks[i])
	}

	if outputJSON {
		checkError(cli.OutputJSON(checks))
		return
	}

	printCheckResults(checks)
}

func parseServiceAddr(svcValue string) (int, string) {
	host, portStr, err := net.SplitHostPort(svcValue)
	if err != nil {
		return 0, ""
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, ""
	}
	return port, host
}

func checkService(check *ServiceCheck) ServiceCheck {
	address := fmt.Sprintf("%s:%d", check.Address, check.Port)

	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", address)

	if err != nil {
		check.Reachable = false
		check.Error = err
	} else {
		_ = conn.Close()
		check.Reachable = true
	}

	return *check
}

func printCheckResults(checks []ServiceCheck) {
	reachable := 0
	unreachable := 0

	table := cli.NewTable("HOST", "SERVICE", "ADDRESS", "PORT", "STATUS")

	for _, c := range checks {
		var status string
		if c.Reachable {
			status = cli.Colorize("UP", cli.ColorGreen)
			reachable++
		} else {
			status = cli.Colorize("DOWN", cli.ColorRed)
			unreachable++
		}
		table.AddRow(c.Host, c.Service, c.Address, fmt.Sprintf("%d", c.Port), status)

		if checkVerbose && c.Error != nil {
			fmt.Printf("  Error: %v\n", c.Error)
		}
	}

	table.Print()

	fmt.Printf("\nSummary: %s up, %s down, %d total\n",
		cli.Colorize(fmt.Sprintf("%d", reachable), cli.ColorGreen),
		cli.Colorize(fmt.Sprintf("%d", unreachable), cli.ColorRed),
		len(checks))
}
