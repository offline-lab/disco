package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	socketPath = "/run/nss-daemon.sock"
)

type Query struct {
	Type      string `json:"type"`
	Name      string `json:"name,omitempty"`
	Addr      string `json:"addr,omitempty"`
	RequestID string `json:"request_id"`
}

type Response struct {
	Type string `json:"type"`
}

func printHelp() {
	fmt.Println("nss-query - Query daemon for host and service information")
	fmt.Println()
	fmt.Println("WHAT:")
	fmt.Println("  Query the NSS daemon for discovered hosts, services, and status.")
	fmt.Println("  Uses the same Unix domain socket as the NSS library for efficiency.")
	fmt.Println()
	fmt.Println("WHY:")
	fmt.Println("  • See all hosts on the network with timing information")
	fmt.Println("  • Find which hosts provide specific services (www, smtp, etc.)")
	fmt.Println("  • Debug network discovery issues")
	fmt.Println("  • Get detailed view of hosts and their advertised services")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  nss-query <command> [args]")
	fmt.Println()
	fmt.Println("COMMANDS:")
	fmt.Println("  hosts            List all discovered hosts")
	fmt.Println("  services         List all discovered services")
	fmt.Println("  hosts-services   Detailed view of hosts with their services")
	fmt.Println("  lookup <name>   Look up a specific host by hostname")
	fmt.Println("  help             Show this help message")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  nss-query hosts")
	fmt.Println("  nss-query services")
	fmt.Println("  nss-query lookup webserver")
	fmt.Println()
	fmt.Println("RELATED:")
	fmt.Println("  nss-daemon          The main daemon")
	fmt.Println("  nss-status          Status and records viewer")
	fmt.Println("  nss-key             Key management tool")
	fmt.Println()
	fmt.Println("For more information, visit: https://github.com/offline-lab/disco")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "hosts":
		listHosts()
	case "services":
		listServices()
	case "hosts-services":
		listHostsWithServices()
	case "lookup":
		if len(os.Args) < 3 {
			fmt.Println("Usage: nss-query lookup <hostname>")
			os.Exit(1)
		}
		lookupHost(os.Args[2])
	case "help", "--help", "-h":
		printHelp()
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("NSS Query Tool - Query daemon for host and service information")
	fmt.Println()
	fmt.Println("Usage: nss-query <command> [args]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  hosts            List all discovered hosts")
	fmt.Println("  services         List all discovered services")
	fmt.Println("  hosts-services   List hosts with their services")
	fmt.Println("  lookup <name>   Look up a specific host")
	fmt.Println("  help             Show this help message")
}

func sendQuery(query *Query) (map[string]interface{}, error) {
	query.RequestID = fmt.Sprintf("cli-%d", time.Now().UnixNano())

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer conn.Close()

	data, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	if _, err := conn.Write(data); err != nil {
		return nil, fmt.Errorf("failed to send query: %w", err)
	}

	buf := make([]byte, 32768)
	n, err := conn.Read(buf)
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("daemon closed connection")
		}
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(buf[:n], &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response, nil
}

func formatTimestamp(ts int64) string {
	if ts == 0 {
		return "unknown"
	}
	t := time.Unix(ts, 0)
	return t.Format("2006-01-02 15:04:05")
}

func formatDuration(seconds int64) string {
	if seconds <= 0 {
		return "expired"
	}

	minutes := seconds / 60
	hours := minutes / 60

	if hours > 24 {
		days := hours / 24
		return fmt.Sprintf("%dd %dh", days, hours%24)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes%60)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%ds", seconds)
}

func listHosts() {
	response, err := sendQuery(&Query{Type: "LIST_HOSTS"})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	hosts, ok := response["hosts"].([]interface{})
	if !ok {
		fmt.Printf("Error: invalid response format\n")
		os.Exit(1)
	}

	if len(hosts) == 0 {
		fmt.Println("No hosts discovered")
		return
	}

	fmt.Printf("Discovered Hosts (%d):\n\n", len(hosts))

	for _, h := range hosts {
		host, ok := h.(map[string]interface{})
		if !ok {
			continue
		}

		hostname, _ := host["hostname"].(string)
		addresses, _ := host["addresses"].([]interface{})
		services, _ := host["services"].([]interface{})
		lastSeen, _ := host["last_seen"].(float64)
		expiresIn, _ := host["expires_in"].(float64)

		fmt.Printf("📡 %s\n", hostname)
		fmt.Printf("   Addresses: ")
		for i, a := range addresses {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("%v", a)
		}
		fmt.Println()
		fmt.Printf("   Last Seen: %s\n", formatTimestamp(int64(lastSeen)))
		fmt.Printf("   Expires In: %s\n", formatDuration(int64(expiresIn)))
		if len(services) > 0 {
			fmt.Printf("   Services: ")
			for i, s := range services {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Printf("%v", s)
			}
			fmt.Println()
		}
		fmt.Println()
	}
}

func listServices() {
	response, err := sendQuery(&Query{Type: "LIST_SERVICES"})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	services, ok := response["services"].(map[string]interface{})
	if !ok {
		fmt.Printf("Error: invalid response format\n")
		os.Exit(1)
	}

	if len(services) == 0 {
		fmt.Println("No services discovered")
		return
	}

	svcNames := make([]string, 0, len(services))
	for name := range services {
		svcNames = append(svcNames, name)
	}
	sort.Strings(svcNames)

	fmt.Printf("Discovered Services (%d):\n\n", len(svcNames))

	for _, svcName := range svcNames {
		hosts, ok := services[svcName].([]interface{})
		if !ok {
			continue
		}

		fmt.Printf("🔧 %s\n", svcName)
		for _, h := range hosts {
			host, ok := h.(map[string]interface{})
			if !ok {
				continue
			}

			hostname, _ := host["hostname"].(string)
			address, _ := host["address"].(string)
			port, _ := host["port"].(float64)

			fmt.Printf("   → %s (%s:%d)\n", hostname, address, int(port))
		}
		fmt.Println()
	}
}

func listHostsWithServices() {
	response, err := sendQuery(&Query{Type: "LIST_HOSTS"})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	hosts, ok := response["hosts"].([]interface{})
	if !ok {
		fmt.Printf("Error: invalid response format\n")
		os.Exit(1)
	}

	if len(hosts) == 0 {
		fmt.Println("No hosts discovered")
		return
	}

	fmt.Printf("Hosts and Services (%d):\n\n", len(hosts))

	for _, h := range hosts {
		host, ok := h.(map[string]interface{})
		if !ok {
			continue
		}

		hostname, _ := host["hostname"].(string)
		addresses, _ := host["addresses"].([]interface{})
		services, _ := host["services"].([]interface{})
		lastSeen, _ := host["last_seen"].(float64)
		expiresIn, _ := host["expires_in"].(float64)

		fmt.Printf("═══════════════════════════════════════\n")
		fmt.Printf("📡 %s\n\n", hostname)

		fmt.Printf("Network:\n")
		for _, a := range addresses {
			fmt.Printf("  • %v\n", a)
		}
		fmt.Printf("\nLast Seen:   %s\n", formatTimestamp(int64(lastSeen)))
		fmt.Printf("Expires In: %s\n", formatDuration(int64(expiresIn)))

		if len(services) > 0 {
			fmt.Printf("\nServices:\n")
			for _, s := range services {
				svcName := fmt.Sprintf("%v", s)
				fmt.Printf("  • %s\n", svcName)
			}
		} else {
			fmt.Printf("\nNo services advertised\n")
		}

		fmt.Println()
	}
}

func lookupHost(hostname string) {
	response, err := sendQuery(&Query{
		Type: "QUERY_BY_NAME",
		Name: hostname,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if response["type"] == "NOTFOUND" {
		fmt.Printf("Host '%s' not found\n", hostname)
		os.Exit(1)
	}

	name, _ := response["name"].(string)
	addresses, _ := response["addrs"].([]interface{})

	fmt.Printf("Host: %s\n", name)
	fmt.Printf("Addresses:\n")
	for _, a := range addresses {
		fmt.Printf("  • %v\n", a)
	}
}

func isLocalHost(hostname string) bool {
	localHostname, _ := os.Hostname()
	return hostname == localHostname
}

func getHostname() string {
	hn, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	if idx := strings.IndexByte(hn, '.'); idx > 0 {
		return hn[:idx]
	}
	return hn
}
