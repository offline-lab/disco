package commands

import (
	"fmt"

	"github.com/offline-lab/disco/cmd/disco/internal/cli"
	"github.com/offline-lab/disco/internal/nss"
	"github.com/spf13/cobra"
)

var locationCmd = &cobra.Command{
	Use:   "location",
	Short: "List GPS location broadcasts received by the daemon",
	Long: `Show GPS location data received from LOCATION_ANNOUNCE broadcasts on the network.

Each entry represents the latest fix reported by a GPS source (e.g. a Heltec
Wireless Tracker or a Raspberry Pi running disco-gps-broadcaster). The daemon
stores only the most recent fix per source — use --json to feed this data into
map or tracking applications.`,
	Args: cobra.NoArgs,
	Run:  listLocations,
}

var locationShowCmd = &cobra.Command{
	Use:   "show <source-id>",
	Short: "Show location detail for one GPS source",
	Args:  cobra.ExactArgs(1),
	Run:   showLocation,
}

func init() {
	rootCmd.AddCommand(locationCmd)
	locationCmd.AddCommand(locationShowCmd)
}

func listLocations(cmd *cobra.Command, args []string) {
	client := cli.NewDaemonClient(getSocketPath())
	response, err := client.Query(&nss.Query{
		Type:      nss.LocationList,
		RequestID: cli.GenerateRequestID("location"),
	})
	checkError(err)

	response, err = cli.HandleResponse(response, err)
	checkError(err)

	if outputJSON {
		checkError(cli.OutputJSON(response.Locations))
		return
	}

	if len(response.Locations) == 0 {
		fmt.Println("No location broadcasts received")
		return
	}

	table := cli.NewTable("SOURCE", "LATITUDE", "LONGITUDE", "ALTITUDE", "SATS", "LAST SEEN")
	for _, loc := range response.Locations {
		table.AddRow(
			loc.SourceID,
			fmt.Sprintf("%.6f", loc.Latitude),
			fmt.Sprintf("%.6f", loc.Longitude),
			fmt.Sprintf("%.1fm", loc.Altitude),
			fmt.Sprintf("%d", loc.Satellites),
			loc.LastSeenAgo,
		)
	}
	table.Print()
}

func showLocation(cmd *cobra.Command, args []string) {
	client := cli.NewDaemonClient(getSocketPath())
	response, err := client.Query(&nss.Query{
		Type:      nss.LocationShow,
		Name:      args[0],
		RequestID: cli.GenerateRequestID("location"),
	})
	checkError(err)

	response, err = cli.HandleResponse(response, err)
	checkError(err)

	if len(response.Locations) == 0 {
		cli.Fatal(fmt.Sprintf("Source not found: %s", args[0]), nil, cli.ExitError)
	}

	if outputJSON {
		checkError(cli.OutputJSON(response.Locations[0]))
		return
	}

	loc := response.Locations[0]
	fmt.Printf("Source:     %s\n", loc.SourceID)
	fmt.Printf("Latitude:   %.6f\n", loc.Latitude)
	fmt.Printf("Longitude:  %.6f\n", loc.Longitude)
	fmt.Printf("Altitude:   %.1f m\n", loc.Altitude)
	fmt.Printf("Satellites: %d\n", loc.Satellites)
	fmt.Printf("Fix:        %v\n", loc.Fix)
	fmt.Printf("Last Seen:  %s\n", loc.LastSeenAgo)
}
