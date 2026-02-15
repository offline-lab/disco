package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
)

const (
	socketPath = "/run/nss-daemon.sock"
)

type Query struct {
	Type string `json:"type"`
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "list":
		listRecords()
	case "help":
		printUsage()
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: nss-status <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  list  - List cached records")
	fmt.Println("  help  - Show this help message")
}

func listRecords() {
	resp, err := sendQuery(map[string]interface{}{"type": "LIST"})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(resp, &data); err == nil {
		if records, ok := data["records"].([]interface{}); ok {
			fmt.Printf("Cached Records (%d):\n", len(records))
			for i, r := range records {
				if rec, ok := r.(map[string]interface{}); ok {
					hostname, _ := rec["hostname"].(string)
					addresses, _ := rec["addresses"].([]interface{})

					fmt.Printf("  %d. %s -> ", i+1, hostname)
					for j, addr := range addresses {
						if j > 0 {
							fmt.Print(", ")
						}
						fmt.Printf("%v", addr)
					}
					fmt.Println()
				}
			}
		}
	}
}

func sendQuery(query map[string]interface{}) ([]byte, error) {
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

	buf := make([]byte, 8192)
	n, err := conn.Read(buf)
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("daemon closed connection")
		}
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return buf[:n], nil
}
