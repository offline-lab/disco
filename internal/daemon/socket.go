package daemon

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/offline-lab/disco/internal/nss"
)

const (
	maxConnections = 100
)

// SocketServer handles NSS queries via Unix domain socket
type SocketServer struct {
	socketPath    string
	store         *RecordStore
	listener      net.Listener
	activeConns   int
	connSemaphore chan struct{}
	connMutex     sync.Mutex
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

// NewSocketServer creates a new socket server
func NewSocketServer(socketPath string, store *RecordStore) *SocketServer {
	return &SocketServer{
		socketPath:    socketPath,
		store:         store,
		connSemaphore: make(chan struct{}, maxConnections),
		stopChan:      make(chan struct{}),
	}
}

// Start begins listening for NSS queries
func (s *SocketServer) Start() error {
	os.Remove(s.socketPath)

	ln, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on unix socket: %w", err)
	}
	s.listener = ln

	socketDir := filepath.Dir(s.socketPath)
	if err := os.Chmod(socketDir, 0755); err != nil {
		log.Printf("Warning: failed to set socket directory permissions: %v", err)
	}

	// Socket must be world-readable/writable for NSS module to access
	if err := os.Chmod(s.socketPath, 0666); err != nil {
		log.Printf("Warning: failed to set socket permissions: %v", err)
	}

	for {
		select {
		case <-s.stopChan:
			return nil
		default:
			ln.(*net.UnixListener).SetDeadline(time.Now().Add(100 * time.Millisecond))
			conn, err := ln.Accept()
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					continue
				}
				continue
			}

			select {
			case s.connSemaphore <- struct{}{}:
				s.wg.Add(1)
				go s.handleConnection(conn)
			default:
				conn.Close()
			}
		}
	}
}

// Stop gracefully shuts down the socket server
func (s *SocketServer) Stop() {
	close(s.stopChan)
	if s.listener != nil {
		s.listener.Close()
	}
	s.wg.Wait()
}

// handleConnection processes a single client connection
func (s *SocketServer) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		<-s.connSemaphore
		s.wg.Done()
	}()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	var query nss.Query
	if err := decoder.Decode(&query); err != nil {
		return
	}

	response := s.handleQuery(&query)

	if err := encoder.Encode(response); err != nil {
		return
	}
}

// handleQuery processes an NSS query and returns a response
func (s *SocketServer) handleQuery(query *nss.Query) *nss.Response {
	switch query.Type {
	case nss.QueryByName:
		return s.handleQueryByName(query)
	case nss.QueryByAddr:
		return s.handleQueryByAddr(query)
	case nss.QueryList:
		return s.handleQueryList(query)
	case nss.QueryListHosts:
		return s.handleQueryListHosts(query)
	case nss.QueryListServices:
		return s.handleQueryListServices(query)
	default:
		return nss.NewErrorResponse(query.RequestID, "unknown query type")
	}
}

// handleQueryByName handles hostname lookups
func (s *SocketServer) handleQueryByName(query *nss.Query) *nss.Response {
	record, exists := s.store.Get(query.Name)
	if !exists {
		return nss.NewNotFoundResponse(query.RequestID)
	}

	addrs := make([]string, len(record.Addresses))
	copy(addrs, record.Addresses)

	return nss.NewOKResponse(query.RequestID, record.Hostname, addrs)
}

// handleQueryByAddr handles reverse lookups
func (s *SocketServer) handleQueryByAddr(query *nss.Query) *nss.Response {
	record, exists := s.store.GetByAddr(query.Addr)
	if !exists {
		return nss.NewNotFoundResponse(query.RequestID)
	}

	addrs := make([]string, len(record.Addresses))
	copy(addrs, record.Addresses)

	return nss.NewOKResponse(query.RequestID, record.Hostname, addrs)
}

// handleQueryList handles listing all records
func (s *SocketServer) handleQueryList(query *nss.Query) *nss.Response {
	records := s.store.List()

	type recordInfo struct {
		Hostname  string            `json:"hostname"`
		Addresses []string          `json:"addresses"`
		Services  map[string]string `json:"services,omitempty"`
		Timestamp int64             `json:"timestamp"`
		TTL       int64             `json:"ttl"`
	}

	recordsList := make([]recordInfo, len(records))
	for i, r := range records {
		recordsList[i] = recordInfo{
			Hostname:  r.Hostname,
			Addresses: r.Addresses,
			Services:  r.Services,
			Timestamp: r.Timestamp,
			TTL:       r.TTL,
		}
	}

	return &nss.Response{
		Type:      nss.ResponseOK,
		RequestID: query.RequestID,
		Records:   mustMarshal(recordsList),
		Count:     len(records),
	}
}

func mustMarshal(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}

// handleQueryListHosts handles listing all hosts
func (s *SocketServer) handleQueryListHosts(query *nss.Query) *nss.Response {
	records := s.store.List()

	type hostInfo struct {
		Hostname  string   `json:"hostname"`
		Addresses []string `json:"addresses"`
		Services  []string `json:"services"`
		FirstSeen int64    `json:"first_seen"`
		LastSeen  int64    `json:"last_seen"`
		TTL       int64    `json:"ttl"`
		ExpiresIn int64    `json:"expires_in"`
	}

	hosts := make([]hostInfo, 0, len(records))
	now := time.Now().Unix()
	for _, r := range records {
		services := make([]string, 0, len(r.Services))
		for svc := range r.Services {
			services = append(services, svc)
		}

		expiresIn := (r.Timestamp + r.TTL) - now
		if expiresIn < 0 {
			expiresIn = 0
		}

		hosts = append(hosts, hostInfo{
			Hostname:  r.Hostname,
			Addresses: r.Addresses,
			Services:  services,
			FirstSeen: r.Timestamp,
			LastSeen:  r.Timestamp,
			TTL:       r.TTL,
			ExpiresIn: expiresIn,
		})
	}

	data, _ := json.Marshal(map[string]interface{}{
		"type":  nss.ResponseOK,
		"hosts": hosts,
		"count": len(hosts),
	})
	var response nss.Response
	json.Unmarshal(data, &response)
	return &response
}

// handleQueryListServices handles listing all services
func (s *SocketServer) handleQueryListServices(query *nss.Query) *nss.Response {
	records := s.store.List()

	type serviceHost struct {
		Hostname string `json:"hostname"`
		Address  string `json:"address"`
		Port     int    `json:"port"`
	}

	services := make(map[string][]serviceHost)
	for _, r := range records {
		for svcName, svcAddr := range r.Services {
			var addr string
			var port int
			fmt.Sscanf(svcAddr, "%s:%d", &addr, &port)

			services[svcName] = append(services[svcName], serviceHost{
				Hostname: r.Hostname,
				Address:  addr,
				Port:     port,
			})
		}
	}

	data, _ := json.Marshal(map[string]interface{}{
		"type":     nss.ResponseOK,
		"services": services,
		"count":    len(services),
	})
	var response nss.Response
	json.Unmarshal(data, &response)
	return &response
}
