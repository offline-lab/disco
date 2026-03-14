package commands

import (
	"fmt"

	"github.com/offline-lab/disco/cmd/disco/internal/cli"
	"github.com/offline-lab/disco/internal/nss"
	"github.com/spf13/cobra"
)

var lookupCmd = &cobra.Command{
	Use:   "lookup <hostname>",
	Short: "Look up a hostname",
	Long:  "Look up IP addresses for a discovered hostname.",
	Args:  cobra.ExactArgs(1),
	Run:   doLookup,
}

func init() {
	rootCmd.AddCommand(lookupCmd)
}

func doLookup(cmd *cobra.Command, args []string) {
	hostname := args[0]

	if err := cli.ValidateHostname(hostname); err != nil {
		checkError(fmt.Errorf("invalid hostname: %w", err))
	}

	client := cli.NewDaemonClient(getSocketPath())
	response, err := client.Query(&nss.Query{
		Type:      nss.QueryByName,
		Name:      hostname,
		RequestID: cli.GenerateRequestID("lookup"),
	})
	checkError(err)

	response, err = cli.HandleResponse(response, err)
	checkError(err)

	if response.Type == nss.ResponseNotFound {
		cli.Fatal(fmt.Sprintf("Host not found: %s", hostname), nil, cli.ExitError)
	}

	for _, addr := range response.Addrs {
		fmt.Println(addr)
	}
}
