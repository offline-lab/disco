package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/offline-lab/disco/internal/logging"
	"github.com/offline-lab/disco/internal/nss"
	"github.com/offline-lab/disco/internal/timesync"
)

const maxConnections = 100

type SocketServer struct {
	socketPath    string
	store         *RecordStore
	timeSync      *timesync.TimeSyncService
	listener      net.Listener
	connSemaphore chan struct{}
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

func NewSocketServer(socketPath string, store *RecordStore) *SocketServer {
	return &SocketServer{
		socketPath:    socketPath,
		store:         store,
		connSemaphore: make(chan struct{}, maxConnections),
		stopChan:      make(chan struct{}),
	}
}

func (s *SocketServer) SetTimeSync(ts *timesync.TimeSyncService) {
	s.timeSync = ts
}

func (s *SocketServer) Start() error {
	_ = os.Remove(s.socketPath)

	ln, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on unix socket: %w", err)
	}
	s.listener = ln

	socketDir := filepath.Dir(s.socketPath)
	if err := os.Chmod(socketDir, 0755); err != nil {
		logging.Warn("Failed to set socket directory permissions", map[string]interface{}{"error": err.Error()})
	}

	if err := os.Chmod(s.socketPath, 0666); err != nil {
		logging.Warn("Failed to set socket permissions", map[string]interface{}{"error": err.Error()})
	}

	for {
		select {
		case <-s.stopChan:
			return nil
		default:
			if err := ln.(*net.UnixListener).SetDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
				continue
			}
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
				_ = conn.Close()
			}
		}
	}
}

func (s *SocketServer) Stop() {
	close(s.stopChan)
	if s.listener != nil {
		_ = s.listener.Close()
	}
	s.wg.Wait()
}

func (s *SocketServer) handleConnection(conn net.Conn) {
	defer func() {
		_ = conn.Close()
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
		logging.Debug("Failed to encode response", map[string]interface{}{"error": err.Error()})
	}
}

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
	case "TIME_STATUS":
		return s.handleTimeStatus(query)
	case "TIME_FORCE_UPDATE":
		return s.handleTimeForceUpdate(query)
	case nss.HostsList:
		return s.handleHostsList(query)
	case nss.HostsShow:
		return s.handleHostsShow(query)
	case nss.HostsForget:
		return s.handleHostsForget(query)
	case nss.HostsMarkLost:
		return s.handleHostsMarkLost(query)
	case nss.ServicesList:
		return s.handleServicesList(query)
	case nss.ServicesShow:
		return s.handleServicesShow(query)
	case nss.ServicesForget:
		return s.handleServicesForget(query)
	default:
		return nss.NewErrorResponse(query.RequestID, "unknown query type")
	}
}

func (s *SocketServer) handleQueryByName(query *nss.Query) *nss.Response {
	record, exists := s.store.Get(query.Name)
	if !exists {
		return nss.NewNotFoundResponse(query.RequestID)
	}

	addrs := make([]string, len(record.Addresses))
	copy(addrs, record.Addresses)

	return nss.NewOKResponse(query.RequestID, record.Hostname, addrs)
}

func (s *SocketServer) handleQueryByAddr(query *nss.Query) *nss.Response {
	record, exists := s.store.GetByAddr(query.Addr)
	if !exists {
		return nss.NewNotFoundResponse(query.RequestID)
	}

	addrs := make([]string, len(record.Addresses))
	copy(addrs, record.Addresses)

	return nss.NewOKResponse(query.RequestID, record.Hostname, addrs)
}

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

	data, err := json.Marshal(recordsList)
	if err != nil {
		logging.Error("Failed to marshal records", err, nil)
		return nss.NewErrorResponse(query.RequestID, "internal error")
	}

	return &nss.Response{
		Type:      nss.ResponseOK,
		RequestID: query.RequestID,
		Records:   data,
	}
}

func (s *SocketServer) handleHostsList(query *nss.Query) *nss.Response {
	records := s.store.ListAll()
	hosts := make([]nss.HostHealth, 0, len(records))
	now := time.Now()

	for _, r := range records {
		var lastSeenAgo string
		if r.IsStatic {
			lastSeenAgo = "(static)"
		} else {
			ago := now.Sub(time.Unix(r.Timestamp, 0))
			lastSeenAgo = formatDuration(ago)
		}

		hosts = append(hosts, nss.HostHealth{
			Hostname:    r.Hostname,
			Addresses:   r.Addresses,
			Status:      string(r.Status),
			Services:    r.Services,
			LastSeen:    r.Timestamp,
			LastSeenAgo: lastSeenAgo,
			IsStatic:    r.IsStatic,
		})
	}

	return &nss.Response{
		Type:      nss.ResponseOK,
		RequestID: query.RequestID,
		Hosts:     hosts,
		Count:     len(hosts),
	}
}

func (s *SocketServer) handleHostsShow(query *nss.Query) *nss.Response {
	hostname := query.Name
	if hostname == "" {
		return nss.NewErrorResponse(query.RequestID, "hostname required")
	}

	records := s.store.ListAll()
	for _, r := range records {
		if r.Hostname == hostname {
			var lastSeenAgo string
			if r.IsStatic {
				lastSeenAgo = "(static)"
			} else {
				ago := time.Since(time.Unix(r.Timestamp, 0))
				lastSeenAgo = formatDuration(ago)
			}

			return &nss.Response{
				Type:      nss.ResponseOK,
				RequestID: query.RequestID,
				Hosts: []nss.HostHealth{{
					Hostname:    r.Hostname,
					Addresses:   r.Addresses,
					Status:      string(r.Status),
					Services:    r.Services,
					LastSeen:    r.Timestamp,
					LastSeenAgo: lastSeenAgo,
					IsStatic:    r.IsStatic,
				}},
				Count: 1,
			}
		}
	}

	return nss.NewNotFoundResponse(query.RequestID)
}

func (s *SocketServer) handleHostsForget(query *nss.Query) *nss.Response {
	hostname := query.Name
	if hostname == "" {
		return nss.NewErrorResponse(query.RequestID, "hostname required")
	}

	s.store.Forget(hostname)
	return &nss.Response{
		Type:      nss.ResponseOK,
		RequestID: query.RequestID,
		Name:      hostname,
	}
}

func (s *SocketServer) handleHostsMarkLost(query *nss.Query) *nss.Response {
	hostname := query.Name
	if hostname == "" {
		return nss.NewErrorResponse(query.RequestID, "hostname required")
	}

	s.store.MarkLost(hostname)
	return &nss.Response{
		Type:      nss.ResponseOK,
		RequestID: query.RequestID,
		Name:      hostname,
	}
}

func (s *SocketServer) handleServicesList(query *nss.Query) *nss.Response {
	records := s.store.ListAll()
	servicesMap := make(map[string]*nss.ServiceHealth)

	for _, r := range records {
		for svcName, svcProto := range r.Services {
			if _, exists := servicesMap[svcName]; !exists {
				servicesMap[svcName] = &nss.ServiceHealth{
					Name:     svcName,
					Protocol: svcProto,
					Hosts:    []string{},
					Status:   string(r.Status),
				}
			}
			servicesMap[svcName].Hosts = append(servicesMap[svcName].Hosts, r.Hostname)
		}
	}

	services := make([]nss.ServiceHealth, 0, len(servicesMap))
	for _, svc := range servicesMap {
		services = append(services, *svc)
	}

	return &nss.Response{
		Type:      nss.ResponseOK,
		RequestID: query.RequestID,
		Services:  services,
		Count:     len(services),
	}
}

func (s *SocketServer) handleServicesShow(query *nss.Query) *nss.Response {
	serviceName := query.Name
	if serviceName == "" {
		return nss.NewErrorResponse(query.RequestID, "service name required")
	}

	records := s.store.ListAll()
	var hosts []string
	var protocol string
	var status string

	for _, r := range records {
		if proto, exists := r.Services[serviceName]; exists {
			hosts = append(hosts, r.Hostname)
			protocol = proto
			if status == "" || status == string(nss.StatusHealthy) {
				status = string(r.Status)
			}
		}
	}

	if len(hosts) == 0 {
		return nss.NewNotFoundResponse(query.RequestID)
	}

	return &nss.Response{
		Type:      nss.ResponseOK,
		RequestID: query.RequestID,
		Services: []nss.ServiceHealth{{
			Name:     serviceName,
			Protocol: protocol,
			Hosts:    hosts,
			Status:   status,
		}},
		Count: 1,
	}
}

func (s *SocketServer) handleServicesForget(query *nss.Query) *nss.Response {
	return nss.NewErrorResponse(query.RequestID, "not implemented: services are tied to hosts")
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

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

		firstSeen := r.FirstSeen
		if firstSeen == 0 {
			firstSeen = r.Timestamp
		}

		hosts = append(hosts, hostInfo{
			Hostname:  r.Hostname,
			Addresses: r.Addresses,
			Services:  services,
			FirstSeen: firstSeen,
			LastSeen:  r.Timestamp,
			TTL:       r.TTL,
			ExpiresIn: expiresIn,
		})
	}

	data, err := json.Marshal(struct {
		Type  string     `json:"type"`
		Hosts []hostInfo `json:"hosts"`
		Count int        `json:"count"`
	}{Type: string(nss.ResponseOK), Hosts: hosts, Count: len(hosts)})
	if err != nil {
		logging.Error("Failed to marshal hosts response", err, nil)
		return nss.NewErrorResponse(query.RequestID, "internal error")
	}

	return &nss.Response{
		Type:      nss.ResponseOK,
		RequestID: query.RequestID,
		Records:   data,
		Count:     len(hosts),
	}
}

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
			if _, err := fmt.Sscanf(svcAddr, "%s:%d", &addr, &port); err != nil {
				continue
			}

			services[svcName] = append(services[svcName], serviceHost{
				Hostname: r.Hostname,
				Address:  addr,
				Port:     port,
			})
		}
	}

	data, err := json.Marshal(struct {
		Type     string                   `json:"type"`
		Services map[string][]serviceHost `json:"services"`
		Count    int                      `json:"count"`
	}{Type: string(nss.ResponseOK), Services: services, Count: len(services)})
	if err != nil {
		logging.Error("Failed to marshal services response", err, nil)
		return nss.NewErrorResponse(query.RequestID, "internal error")
	}

	return &nss.Response{
		Type:      nss.ResponseOK,
		RequestID: query.RequestID,
		Records:   data,
		Count:     len(services),
	}
}

func (s *SocketServer) handleTimeStatus(query *nss.Query) *nss.Response {
	if s.timeSync == nil {
		return &nss.Response{
			Type:      nss.ResponseError,
			RequestID: query.RequestID,
			Error:     "time sync not enabled",
		}
	}

	status := s.timeSync.GetStatus()

	data, err := json.Marshal(struct {
		Type         string  `json:"type"`
		RequestID    string  `json:"request_id"`
		Synced       bool    `json:"synced"`
		SourceCount  int     `json:"source_count"`
		LastOffset   float64 `json:"last_offset"`
		LastSyncTime string  `json:"last_sync_time"`
		LastError    string  `json:"last_error,omitempty"`
	}{
		Type:         string(nss.ResponseOK),
		RequestID:    query.RequestID,
		Synced:       status.Synced,
		SourceCount:  status.SourceCount,
		LastOffset:   status.LastOffset,
		LastSyncTime: status.LastSyncTime.Format(time.RFC3339),
		LastError:    status.LastError,
	})
	if err != nil {
		logging.Error("Failed to marshal time status response", err, nil)
		return nss.NewErrorResponse(query.RequestID, "internal error")
	}

	return &nss.Response{
		Type:      nss.ResponseOK,
		RequestID: query.RequestID,
		Records:   data,
	}
}

func (s *SocketServer) handleTimeForceUpdate(query *nss.Query) *nss.Response {
	if s.timeSync == nil {
		return &nss.Response{
			Type:      nss.ResponseError,
			RequestID: query.RequestID,
			Error:     "time sync not enabled",
		}
	}

	allowBackward := query.Name == "true" || query.Name == "1"
	result := s.timeSync.ForceUpdate(allowBackward)

	data, err := json.Marshal(struct {
		Type        string  `json:"type"`
		RequestID   string  `json:"request_id"`
		Success     bool    `json:"success"`
		Offset      float64 `json:"offset"`
		Method      string  `json:"method"`
		SourceCount int     `json:"source_count"`
		Error       string  `json:"error,omitempty"`
	}{
		Type:        string(nss.ResponseOK),
		RequestID:   query.RequestID,
		Success:     result.Success,
		Offset:      result.Offset,
		Method:      result.Method,
		SourceCount: result.SourceCount,
		Error:       result.Error,
	})
	if err != nil {
		logging.Error("Failed to marshal time force update response", err, nil)
		return nss.NewErrorResponse(query.RequestID, "internal error")
	}

	return &nss.Response{
		Type:      nss.ResponseOK,
		RequestID: query.RequestID,
		Records:   data,
	}
}
