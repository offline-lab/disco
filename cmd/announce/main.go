package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/offline-lab/disco/internal/discovery"
)

func main() {
	broadcastAddr := flag.String("addr", "255.255.255.255:5354", "Broadcast address (host:port)")
	hostname := flag.String("hostname", "", "Hostname to announce (required)")
	interval := flag.Duration("interval", 5*time.Second, "Announcement interval")
	count := flag.Int("count", 0, "Number of announcements (0 = unlimited)")
	port := flag.Int("port", 0, "Port for service")
	service := flag.String("service", "", "Service name to announce")
	flag.Parse()

	if *hostname == "" {
		fmt.Println("Usage: nss-announce -hostname <name> [options]")
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  nss-announce -hostname web1")
		fmt.Println("  nss-announce -hostname mail1 -service smtp -port 25")
		fmt.Println("  nss-announce -hostname test1 -addr 255.255.255.255:5354 -count 5")
		os.Exit(1)
	}

	// Create message
	services := []discovery.ServiceInfo{}
	if *service != "" && *port > 0 {
		services = append(services, discovery.ServiceInfo{
			Name: *service,
			Port: *port,
		})
	}

	// Get local IPs
	ips, err := getLocalIPs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting local IPs: %v\n", err)
		os.Exit(1)
	}

	msg := discovery.BroadcastMessage{
		Type:      discovery.MessageAnnounce,
		MessageID: fmt.Sprintf("announce-%d", time.Now().UnixNano()),
		Timestamp: time.Now().Unix(),
		Hostname:  *hostname,
		IPs:       ips,
		Services:  services,
		TTL:       3600,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling message: %v\n", err)
		os.Exit(1)
	}

	// Resolve broadcast address
	addr, err := net.ResolveUDPAddr("udp", *broadcastAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving address: %v\n", err)
		os.Exit(1)
	}

	// Create UDP connection
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating UDP connection: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("Announcing host: %s\n", *hostname)
	fmt.Printf("Broadcast address: %s\n", *broadcastAddr)
	fmt.Printf("Interval: %v\n", *interval)
	fmt.Printf("Local IPs: %v\n", ips)
	if len(services) > 0 {
		fmt.Printf("Services: %v\n", services)
	}
	fmt.Println()

	announcements := 0
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	// Send first announcement immediately
	sendAnnouncement(conn, data, *hostname, announcements)
	announcements++

	for range ticker.C {
		if *count > 0 && announcements >= *count {
			fmt.Printf("\nSent %d announcements, stopping.\n", announcements)
			break
		}

		// Update timestamp and message ID
		msg.Timestamp = time.Now().Unix()
		msg.MessageID = fmt.Sprintf("announce-%d", time.Now().UnixNano())
		data, _ = json.Marshal(msg)

		sendAnnouncement(conn, data, *hostname, announcements)
		announcements++
	}
}

func sendAnnouncement(conn *net.UDPConn, data []byte, hostname string, count int) {
	_, err := conn.Write(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%d] Error sending: %v\n", count, err)
		return
	}
	fmt.Printf("[%d] Announced: %s (%d bytes)\n", count, hostname, len(data))
}

func getLocalIPs() ([]string, error) {
	var ips []string

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		// Skip loopback and down interfaces
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

			// Only include IPv4 addresses
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
