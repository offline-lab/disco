package commands

import (
	"fmt"
	"strconv"

	"github.com/offline-lab/disco/cmd/disco/internal/cli"
	"github.com/offline-lab/disco/internal/nss"
	"github.com/spf13/cobra"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage advertised services",
}

var serviceAddAliases []string

var serviceAddCmd = &cobra.Command{
	Use:   "add <name> <port>",
	Short: "Advertise a new service via the running daemon",
	Long: `Tell the running disco daemon to advertise a new service on this host.
The daemon will update its store, broadcast the new service immediately,
and include it in all future announcements.

Useful in systemd unit ExecStartPost= to register portable services:

  ExecStartPost=disco service add myapp 8080 --alias myapp.local`,
	Args: cobra.ExactArgs(2),
	RunE: runServiceAdd,
}

func init() {
	rootCmd.AddCommand(serviceCmd)
	serviceCmd.AddCommand(serviceAddCmd)
	serviceAddCmd.Flags().StringArrayVarP(&serviceAddAliases, "alias", "a", nil, "Alias name(s) to advertise for this service")
}

func runServiceAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	port, err := strconv.Atoi(args[1])
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid port %q: must be 1-65535", args[1])
	}

	client := cli.NewDaemonClient(getSocketPath())
	response, err := client.Query(&nss.Query{
		Type:      nss.ServiceAnnounce,
		Name:      name,
		Port:      port,
		Aliases:   serviceAddAliases,
		RequestID: cli.GenerateRequestID("service-add"),
	})
	if err != nil {
		return err
	}

	if _, err = cli.HandleResponse(response, err); err != nil {
		return err
	}

	fmt.Printf("Announced service %q on port %d", name, port)
	if len(serviceAddAliases) > 0 {
		fmt.Printf(" with aliases: %v", serviceAddAliases)
	}
	fmt.Println()
	return nil
}
