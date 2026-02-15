package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
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
	Family    int    `json:"family,omitempty"`
	RequestID string `json:"request_id"`
}

type Response struct {
	Type  string   `json:"type"`
	Name  string   `json:"name,omitempty"`
	Addrs []string `json:"addrs,omitempty"`
}

var (
	verbose   bool
	queryType string
	value     string
	timeout   time.Duration
)

func init() {
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.StringVar(&queryType, "query", "", "Query type: name or addr")
	flag.StringVar(&value, "value", "", "Hostname or IP address to query")
	flag.DurationVar(&timeout, "timeout", 2*time.Second, "Query timeout")
}

func main() {
	flag.Parse()

	if value == "" {
		printUsage()
		os.Exit(1)
	}

	switch strings.ToLower(queryType) {
	case "", "name", "hostname", "a":
		lookupByName(value)
	case "addr", "address", "ptr":
		lookupByAddr(value)
	default:
		fmt.Fprintf(os.Stderr, "Unknown query type: %s\n", queryType)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("NSS DNS Client - Query NSS daemon")
	fmt.Println()
	fmt.Println("Usage: nss-dns [options] <hostname|ip-address>")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -query <type>    Query type (name or addr, default: name)")
	fmt.Println("  -v                Verbose output")
	fmt.Println("  -timeout <dur>    Query timeout (default: 2s)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  nss-dns webserver")
	fmt.Println("  nss-dns -query name webserver")
	fmt.Println("  nss-dns -query addr 192.168.1.10")
	fmt.Println("  nss-dns 192.168.1.10")
	fmt.Println()
	fmt.Println("This is a client for the nss-daemon service.")
	fmt.Println("For more information, see: man nss-daemon")
}

func sendQuery(query *Query) (*Response, error) {
	query.RequestID = fmt.Sprintf("dns-%d", time.Now().UnixNano())

	conn, err := net.DialTimeout("unix", socketPath, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	data, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	if _, err := conn.Write(data); err != nil {
		return nil, fmt.Errorf("failed to send query: %w", err)
	}

	buf := make([]byte, 8192)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("no response from daemon")
	}

	var resp Response
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

func lookupByName(hostname string) {
	if verbose {
		fmt.Printf("Looking up %s...\n", hostname)
	}

	query := &Query{
		Type:   "QUERY_BY_NAME",
		Name:   hostname,
		Family: 2,
	}

	resp, err := sendQuery(query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if resp.Type == "NOTFOUND" || resp.Type == "ERROR" {
		fmt.Printf("%s: not found\n", hostname)
		os.Exit(1)
	}

	fmt.Printf("Server:\t%s\n", resp.Name)
	fmt.Printf("Addresses:")
	for _, addr := range resp.Addrs {
		fmt.Printf("\t%s\n", addr)
	}

	if verbose {
		fmt.Printf("\nResponse Type: %s\n", resp.Type)
		fmt.Printf("Addresses found: %d\n", len(resp.Addrs))
	}
}

func lookupByAddr(ip string) {
	if verbose {
		fmt.Printf("Looking up %s...\n", ip)
	}

	query := &Query{
		Type:   "QUERY_BY_ADDR",
		Addr:   ip,
		Family: 2,
	}

	resp, err := sendQuery(query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if resp.Type == "NOTFOUND" || resp.Type == "ERROR" {
		fmt.Printf("%s: not found\n", ip)
		os.Exit(1)
	}

	fmt.Printf("Hostname:\t%s\n", resp.Name)
	fmt.Printf("Addresses:")
	for _, addr := range resp.Addrs {
		fmt.Printf("\t%s\n", addr)
	}

	if verbose {
		fmt.Printf("\nResponse Type: %s\n", resp.Type)
		fmt.Printf("Addresses found: %d\n", len(resp.Addrs))
	}
}
