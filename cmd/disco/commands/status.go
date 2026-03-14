package commands

import (
	"fmt"
	"strings"

	"github.com/offline-lab/disco/cmd/disco/internal/cli"
	"github.com/offline-lab/disco/internal/nss"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status",
	Long:  "Display daemon status including host counts by health state.",
	Args:  cobra.NoArgs,
	Run:   doStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func doStatus(cmd *cobra.Command, args []string) {
	client := cli.NewDaemonClient(getSocketPath())
	response, err := client.Query(&nss.Query{
		Type:      nss.HostsList,
		RequestID: cli.GenerateRequestID("status"),
	})
	checkError(err)

	response, err = cli.HandleResponse(response, err)
	checkError(err)

	healthy := 0
	stale := 0
	lost := 0
	static := 0

	for _, h := range response.Hosts {
		switch h.Status {
		case "healthy":
			healthy++
		case "stale":
			stale++
		case "lost":
			lost++
		case "static":
			static++
		}
	}

	fmt.Println("Disco Daemon Status")
	fmt.Println(strings.Repeat("-", 30))
	fmt.Printf("Socket:      %s\n", getSocketPath())
	fmt.Printf("Total hosts: %d\n", len(response.Hosts))
	fmt.Printf("  Healthy:   %d\n", healthy)
	fmt.Printf("  Stale:     %d\n", stale)
	fmt.Printf("  Lost:      %d\n", lost)
	fmt.Printf("  Static:    %d\n", static)
}
