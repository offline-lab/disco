package commands

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/offline-lab/disco/cmd/disco/internal/cli"
	"github.com/offline-lab/disco/internal/discovery"
	"github.com/spf13/cobra"
)

var (
	announceAddr     string
	announceHostname string
	announceInterval time.Duration
	announceCount    int
	announcePort     int
	announceService  string
)

var announceCmd = &cobra.Command{
	Use:   "announce",
	Short: "Send manual discovery announcements",
	Long: `Send manual discovery broadcast announcements to the network.

This tool sends UDP broadcast messages to announce a host and its services
to other nodes on the network. Useful for testing or manual configuration.`,
	RunE: runAnnounce,
}

func init() {
	rootCmd.AddCommand(announceCmd)

	announceCmd.Flags().StringVarP(&announceHostname, "hostname", "n", "", "Hostname to announce (required)")
	announceCmd.Flags().StringVarP(&announceAddr, "addr", "a", "255.255.255.255:5354", "Broadcast address (host:port)")
	announceCmd.Flags().DurationVarP(&announceInterval, "interval", "i", 5*time.Second, "Announcement interval")
	announceCmd.Flags().IntVarP(&announceCount, "count", "c", 0, "Number of announcements (0 = unlimited)")
	announceCmd.Flags().IntVarP(&announcePort, "port", "p", 0, "Service port (requires --service)")
	announceCmd.Flags().StringVarP(&announceService, "service", "S", "", "Service name to announce")

	if err := announceCmd.MarkFlagRequired("hostname"); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to mark hostname flag as required: %v\n", err)
	}
}

func runAnnounce(cmd *cobra.Command, args []string) error {
	if err := validateAnnounceArgs(); err != nil {
		return err
	}

	ips, err := getLocalIPs()
	if err != nil {
		return fmt.Errorf("error getting local IPs: %w", err)
	}

	services := buildServices()
	msg := createBroadcastMessage(ips, services)
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("error marshaling message: %w", err)
	}

	conn, err := createUDPConnection(announceAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	printAnnounceInfo(ips, services)

	return sendAnnouncements(conn, data, msg)
}

func validateAnnounceArgs() error {
	if announceHostname == "" {
		return fmt.Errorf("hostname is required")
	}

	if err := cli.ValidateHostname(announceHostname); err != nil {
		return fmt.Errorf("invalid hostname: %w", err)
	}

	if announceService != "" && announcePort == 0 {
		return fmt.Errorf("port is required when service is specified")
	}

	if announcePort > 0 && announceService == "" {
		return fmt.Errorf("service is required when port is specified")
	}

	if announceService != "" {
		if err := cli.ValidateServiceName(announceService); err != nil {
			return fmt.Errorf("invalid service name: %w", err)
		}
	}

	if announcePort > 0 {
		if err := cli.ValidatePort(announcePort); err != nil {
			return fmt.Errorf("invalid port: %w", err)
		}
	}

	return nil
}

func buildServices() []discovery.ServiceInfo {
	var services []discovery.ServiceInfo
	if announceService != "" && announcePort > 0 {
		services = append(services, discovery.ServiceInfo{
			Name: announceService,
			Port: announcePort,
		})
	}
	return services
}

func createBroadcastMessage(ips []string, services []discovery.ServiceInfo) *discovery.BroadcastMessage {
	return &discovery.BroadcastMessage{
		Type:      discovery.MessageAnnounce,
		MessageID: fmt.Sprintf("announce-%d", time.Now().UnixNano()),
		Timestamp: time.Now().Unix(),
		Hostname:  announceHostname,
		IPs:       ips,
		Services:  services,
		TTL:       3600,
	}
}

func createUDPConnection(address string) (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, fmt.Errorf("error resolving address: %w", err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("error creating UDP connection: %w", err)
	}

	return conn, nil
}

func printAnnounceInfo(ips []string, services []discovery.ServiceInfo) {
	fmt.Printf("Announcing host: %s\n", announceHostname)
	fmt.Printf("Broadcast address: %s\n", announceAddr)
	fmt.Printf("Interval: %v\n", announceInterval)
	fmt.Printf("Local IPs: %v\n", ips)
	if len(services) > 0 {
		fmt.Printf("Services: %v\n", services)
	}
	fmt.Println()
}

func sendAnnouncements(conn *net.UDPConn, data []byte, msg *discovery.BroadcastMessage) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	announcements := 0
	ticker := time.NewTicker(announceInterval)
	defer ticker.Stop()

	sendAnnouncement(conn, data, announcements)
	announcements++

	for {
		select {
		case <-sigChan:
			fmt.Printf("\nReceived interrupt, stopping after %d announcements.\n", announcements)
			return nil

		case <-ticker.C:
			if announceCount > 0 && announcements >= announceCount {
				fmt.Printf("\nSent %d announcements, stopping.\n", announcements)
				return nil
			}

			msg.Timestamp = time.Now().Unix()
			msg.MessageID = fmt.Sprintf("announce-%d", time.Now().UnixNano())
			data, _ = json.Marshal(msg)

			sendAnnouncement(conn, data, announcements)
			announcements++
		}
	}
}

func sendAnnouncement(conn *net.UDPConn, data []byte, count int) {
	_, err := conn.Write(data)
	if err != nil {
		cli.PrintError("[%d] Error sending: %v\n", count, err)
		return
	}
	cli.PrintSuccess("[%d] Announced: %s (%d bytes)\n", count, announceHostname, len(data))
}

func getLocalIPs() ([]string, error) {
	var ips []string

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip != nil && ip.To4() != nil && !ip.IsLoopback() {
				ips = append(ips, ip.String())
			}
		}
	}

	if len(ips) == 0 {
		return []string{"127.0.0.1"}, nil
	}

	return ips, nil
}
