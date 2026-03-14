package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: socket-client <socket-path>")
		os.Exit(1)
	}
	socket := os.Args[1]

	tests := []struct {
		name  string
		query map[string]string
		check func(map[string]interface{}) bool
	}{
		{
			name:  "LIST_HOSTS",
			query: map[string]string{"type": "LIST_HOSTS", "request_id": "test1"},
			check: func(r map[string]interface{}) bool { return r["type"] != nil },
		},
		{
			name:  "LIST_SERVICES",
			query: map[string]string{"type": "LIST_SERVICES", "request_id": "test2"},
			check: func(r map[string]interface{}) bool { return r["type"] != nil },
		},
		{
			name:  "QUERY_BY_NAME (nonexistent)",
			query: map[string]string{"type": "QUERY_BY_NAME", "name": "nonexistent", "request_id": "test3"},
			check: func(r map[string]interface{}) bool { return r["type"] == "NOTFOUND" },
		},
	}

	for _, tc := range tests {
		conn, err := net.DialTimeout("unix", socket, 2*time.Second)
		if err != nil {
			fmt.Printf("   %s: FAIL (%v)\n", tc.name, err)
			continue
		}

		data, _ := json.Marshal(tc.query)
		if _, err := conn.Write(data); err != nil {
			fmt.Printf("   %s: FAIL (write: %v)\n", tc.name, err)
			_ = conn.Close()
			continue
		}
		if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			fmt.Printf("   %s: FAIL (deadline: %v)\n", tc.name, err)
			_ = conn.Close()
			continue
		}

		buf := make([]byte, 4096)
		n, err := conn.Read(buf)
		_ = conn.Close()

		if err != nil {
			fmt.Printf("   %s: FAIL (%v)\n", tc.name, err)
			continue
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(buf[:n], &resp); err != nil {
			fmt.Printf("   %s: FAIL (parse: %v)\n", tc.name, err)
			continue
		}

		if tc.check(resp) {
			fmt.Printf("   %s: OK\n", tc.name)
		} else {
			fmt.Printf("   %s: FAIL (unexpected response: %v)\n", tc.name, resp)
		}
	}
}
