package client

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type TimeStatus struct {
	Synced       bool    `json:"synced"`
	SourceCount  int     `json:"source_count"`
	LastOffset   float64 `json:"last_offset_seconds"`
	LastSyncTime string  `json:"last_sync_time"`
	LastError    string  `json:"last_error,omitempty"`
}

type ForceUpdateResult struct {
	Success     bool    `json:"success"`
	Offset      float64 `json:"offset_seconds"`
	Method      string  `json:"method"`
	SourceCount int     `json:"source_count"`
	Error       string  `json:"error,omitempty"`
}

type DaemonClient struct {
	socketPath string
	timeout    time.Duration
}

func NewDaemonClient(socketPath string) *DaemonClient {
	return &DaemonClient{
		socketPath: socketPath,
		timeout:    5 * time.Second,
	}
}

func (c *DaemonClient) WithTimeout(timeout time.Duration) *DaemonClient {
	c.timeout = timeout
	return c
}

func (c *DaemonClient) connect() (net.Conn, error) {
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	if err := conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to set deadline: %w", err)
	}
	return conn, nil
}

func (c *DaemonClient) GetTimeStatus() (*TimeStatus, error) {
	conn, err := c.connect()
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	req := map[string]interface{}{
		"type":       "TIME_STATUS",
		"request_id": fmt.Sprintf("time-%d", time.Now().UnixNano()),
	}

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	status := &TimeStatus{}
	if v, ok := resp["synced"].(bool); ok {
		status.Synced = v
	}
	if v, ok := resp["source_count"].(float64); ok {
		status.SourceCount = int(v)
	}
	if v, ok := resp["last_offset"].(float64); ok {
		status.LastOffset = v
	}
	if v, ok := resp["last_sync_time"].(string); ok {
		status.LastSyncTime = v
	}
	if v, ok := resp["last_error"].(string); ok {
		status.LastError = v
	}

	return status, nil
}

func (c *DaemonClient) ForceTimeUpdate(allowBackward bool) (*ForceUpdateResult, error) {
	conn, err := c.connect()
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	req := map[string]interface{}{
		"type":           "TIME_FORCE_UPDATE",
		"request_id":     fmt.Sprintf("force-%d", time.Now().UnixNano()),
		"allow_backward": allowBackward,
	}

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	result := &ForceUpdateResult{}
	if v, ok := resp["success"].(bool); ok {
		result.Success = v
	}
	if v, ok := resp["offset"].(float64); ok {
		result.Offset = v
	}
	if v, ok := resp["method"].(string); ok {
		result.Method = v
	}
	if v, ok := resp["source_count"].(float64); ok {
		result.SourceCount = int(v)
	}
	if v, ok := resp["error"].(string); ok {
		result.Error = v
	}

	return result, nil
}
