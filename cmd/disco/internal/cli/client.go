package cli

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/offline-lab/disco/internal/nss"
)

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

func (c *DaemonClient) Query(query *nss.Query) (*nss.Response, error) {
	conn, err := net.DialTimeout("unix", c.socketPath, c.timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w (is disco-daemon running?)", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(c.timeout))

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	if err := encoder.Encode(query); err != nil {
		return nil, fmt.Errorf("failed to send query: %w", err)
	}

	var response nss.Response
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

func GenerateRequestID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func HandleResponse(response *nss.Response, err error) (*nss.Response, error) {
	if err != nil {
		return nil, err
	}
	if response.Type == nss.ResponseError {
		return nil, fmt.Errorf("%s", response.Error)
	}
	return response, nil
}
