package client

import (
	"encoding/json"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

func TestNewDaemonClient(t *testing.T) {
	client := NewDaemonClient("/tmp/test.sock")
	if client.socketPath != "/tmp/test.sock" {
		t.Errorf("socketPath = %s, want /tmp/test.sock", client.socketPath)
	}
	if client.timeout != 5*time.Second {
		t.Errorf("timeout = %v, want 5s", client.timeout)
	}
}

func TestDaemonClient_WithTimeout(t *testing.T) {
	client := NewDaemonClient("/tmp/test.sock")
	client.WithTimeout(10 * time.Second)
	if client.timeout != 10*time.Second {
		t.Errorf("timeout = %v, want 10s", client.timeout)
	}
}

func TestDaemonClient_Connect_MissingSocket(t *testing.T) {
	client := NewDaemonClient("/tmp/nonexistent-socket-12345.sock")
	_, err := client.connect()
	if err == nil {
		t.Error("expected error for missing socket")
	}
}

func TestDaemonClient_GetTimeStatus(t *testing.T) {
	socketPath := "/tmp/disco-client-test-" + time.Now().Format("20060102150405") + ".sock"
	os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to create socket: %v", err)
	}
	defer listener.Close()
	defer os.Remove(socketPath)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		var req map[string]interface{}
		if err := json.NewDecoder(conn).Decode(&req); err != nil {
			return
		}

		resp := map[string]interface{}{
			"type":           "TIME_STATUS_RESPONSE",
			"synced":         true,
			"source_count":   3,
			"last_offset":    0.001,
			"last_sync_time": "2024-01-01T00:00:00Z",
		}
		json.NewEncoder(conn).Encode(resp)
	}()

	client := NewDaemonClient(socketPath)
	status, err := client.GetTimeStatus()
	if err != nil {
		t.Fatalf("GetTimeStatus() error = %v", err)
	}

	if !status.Synced {
		t.Error("Synced = false, want true")
	}
	if status.SourceCount != 3 {
		t.Errorf("SourceCount = %d, want 3", status.SourceCount)
	}

	wg.Wait()
}

func TestDaemonClient_ForceTimeUpdate(t *testing.T) {
	socketPath := "/tmp/disco-client-test-force-" + time.Now().Format("20060102150405") + ".sock"
	os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to create socket: %v", err)
	}
	defer listener.Close()
	defer os.Remove(socketPath)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		var req map[string]interface{}
		if err := json.NewDecoder(conn).Decode(&req); err != nil {
			return
		}

		resp := map[string]interface{}{
			"type":         "TIME_FORCE_UPDATE_RESPONSE",
			"success":      true,
			"offset":       0.5,
			"method":       "step",
			"source_count": 2,
		}
		json.NewEncoder(conn).Encode(resp)
	}()

	client := NewDaemonClient(socketPath)
	result, err := client.ForceTimeUpdate(false)
	if err != nil {
		t.Fatalf("ForceTimeUpdate() error = %v", err)
	}

	if !result.Success {
		t.Error("Success = false, want true")
	}
	if result.Method != "step" {
		t.Errorf("Method = %s, want step", result.Method)
	}
	if result.Offset != 0.5 {
		t.Errorf("Offset = %f, want 0.5", result.Offset)
	}

	wg.Wait()
}

func TestTimeStatus_Fields(t *testing.T) {
	status := &TimeStatus{
		Synced:       true,
		SourceCount:  5,
		LastOffset:   0.123,
		LastSyncTime: "2024-01-01T00:00:00Z",
		LastError:    "",
	}

	if !status.Synced {
		t.Error("Synced should be true")
	}
	if status.SourceCount != 5 {
		t.Errorf("SourceCount = %d, want 5", status.SourceCount)
	}
}

func TestForceUpdateResult_Fields(t *testing.T) {
	result := &ForceUpdateResult{
		Success:     true,
		Offset:      0.5,
		Method:      "step",
		SourceCount: 2,
		Error:       "",
	}

	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Method != "step" {
		t.Errorf("Method = %s, want step", result.Method)
	}
}
