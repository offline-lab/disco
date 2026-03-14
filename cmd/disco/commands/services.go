package commands

import (
	"fmt"

	"github.com/offline-lab/disco/cmd/disco/internal/cli"
	"github.com/offline-lab/disco/internal/nss"
	"github.com/spf13/cobra"
)

var servicesCmd = &cobra.Command{
	Use:   "services",
	Short: "List discovered services",
	Long:  "List all services discovered across all hosts, or show details for a specific service.",
	Args:  cobra.MaximumNArgs(1),
	Run:   listServices,
}

func init() {
	rootCmd.AddCommand(servicesCmd)
}

func listServices(cmd *cobra.Command, args []string) {
	client := cli.NewDaemonClient(getSocketPath())

	if len(args) > 0 {
		showService(cmd, args, client)
		return
	}

	response, err := client.Query(&nss.Query{
		Type:      nss.ServicesList,
		RequestID: cli.GenerateRequestID("services"),
	})
	checkError(err)

	response, err = cli.HandleResponse(response, err)
	checkError(err)

	if outputJSON {
		checkError(cli.OutputJSON(response.Services))
		return
	}

	if len(response.Services) == 0 {
		fmt.Println("No services discovered")
		return
	}

	table := cli.NewTable("SERVICE", "PROTOCOL", "HOSTS")
	for _, s := range response.Services {
		hosts := cli.Truncate(cli.JoinStrings(s.Hosts, ", "), 38)
		table.AddRow(s.Name, s.Protocol, hosts)
	}
	table.Print()
}

func showService(cmd *cobra.Command, args []string, client *cli.DaemonClient) {
	name := args[0]

	response, err := client.Query(&nss.Query{
		Type:      nss.ServicesShow,
		Name:      name,
		RequestID: cli.GenerateRequestID("service"),
	})
	checkError(err)

	response, err = cli.HandleResponse(response, err)
	checkError(err)

	if response.Type == nss.ResponseNotFound || len(response.Services) == 0 {
		cli.Fatal(fmt.Sprintf("Service not found: %s", name), nil, cli.ExitError)
	}

	s := response.Services[0]

	fmt.Printf("Service:   %s\n", s.Name)
	fmt.Printf("Protocol:  %s\n", s.Protocol)
	fmt.Printf("Status:    %s\n", cli.ColorizeStatus(s.Status))
	fmt.Printf("\nHosts:\n")
	for _, h := range s.Hosts {
		fmt.Printf("  - %s\n", h)
	}
}
