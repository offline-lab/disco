package daemon

import (
	"testing"
	"time"

	"github.com/offline-lab/disco/internal/config"
	"github.com/offline-lab/disco/internal/nss"
)

func TestRecordStore_Stop(t *testing.T) {
	healthCfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	store := NewRecordStore(time.Hour, healthCfg, nil)

	store.Stop()

	store.mu.Lock()
	store.mu.Unlock()
}

func TestRecordStore_ListAll(t *testing.T) {
	healthCfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	store := NewRecordStore(time.Hour, healthCfg, nil)
	defer store.Stop()

	store.AddOrUpdate(&nss.Record{Hostname: "host1", Addresses: []string{"1.1.1.1"}})
	store.AddOrUpdate(&nss.Record{Hostname: "host2", Addresses: []string{"2.2.2.2"}})

	records := store.ListAll()
	if len(records) != 2 {
		t.Errorf("ListAll() count = %d, want 2", len(records))
	}
}

func TestRecordStore_MarkLost(t *testing.T) {
	healthCfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	store := NewRecordStore(time.Hour, healthCfg, nil)
	defer store.Stop()

	store.AddOrUpdate(&nss.Record{Hostname: "host1", Addresses: []string{"1.1.1.1"}})

	store.MarkLost("host1")

	store.mu.Lock()
	record, exists := store.records["host1"]
	store.mu.Unlock()

	if !exists {
		t.Fatal("Record should still exist after MarkLost")
	}
	if record.Status != nss.StatusLost {
		t.Errorf("Status = %s, want lost", record.Status)
	}
}

func TestRecordStore_Forget(t *testing.T) {
	healthCfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	store := NewRecordStore(time.Hour, healthCfg, nil)
	defer store.Stop()

	store.AddOrUpdate(&nss.Record{Hostname: "host1", Addresses: []string{"1.1.1.1"}})
	store.Forget("host1")

	_, exists := store.Get("host1")
	if exists {
		t.Error("Forget() did not remove record")
	}
}

func TestRecordStore_GetAllRecords(t *testing.T) {
	healthCfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	store := NewRecordStore(time.Hour, healthCfg, nil)
	defer store.Stop()

	store.AddOrUpdate(&nss.Record{
		Hostname:  "host1",
		Addresses: []string{"1.1.1.1"},
		Services:  map[string]string{"ssh": "tcp:22"},
	})

	records := store.GetAllRecords()
	if len(records) != 1 {
		t.Fatalf("GetAllRecords() count = %d, want 1", len(records))
	}

	if records[0].Hostname != "host1" {
		t.Errorf("Hostname = %s, want host1", records[0].Hostname)
	}
}

func TestHealthTracker_ShouldExpire(t *testing.T) {
	cfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	ht := NewHealthTracker(cfg, nil)

	now := time.Now().Unix()

	tests := []struct {
		name     string
		record   *nss.Record
		expected bool
	}{
		{
			name: "static record never expires",
			record: &nss.Record{
				IsStatic:  true,
				Timestamp: 0,
			},
			expected: false,
		},
		{
			name: "fresh record",
			record: &nss.Record{
				Timestamp: now,
				IsStatic:  false,
			},
			expected: false,
		},
		{
			name: "expired record",
			record: &nss.Record{
				Timestamp: now - 400,
				IsStatic:  false,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ht.ShouldExpire(tt.record)
			if result != tt.expected {
				t.Errorf("ShouldExpire() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSocketServer_FormatDuration(t *testing.T) {
	tests := []struct {
		d        time.Duration
		expected string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m"},
		{2 * time.Minute, "2m"},
		{90 * time.Minute, "1h30m"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.d)
		if result != tt.expected {
			t.Errorf("formatDuration(%v) = %s, want %s", tt.d, result, tt.expected)
		}
	}
}

func TestSocketServer_HandleHostsList(t *testing.T) {
	healthCfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	store := NewRecordStore(time.Hour, healthCfg, nil)
	defer store.Stop()

	store.AddOrUpdate(&nss.Record{
		Hostname:  "host1",
		Addresses: []string{"1.1.1.1"},
		Services:  map[string]string{"ssh": "tcp:22"},
	})

	server := NewSocketServer("/tmp/test.sock", store)

	query := &nss.Query{
		Type:      nss.HostsList,
		RequestID: "req-1",
	}

	resp := server.handleHostsList(query)

	if resp.Type != nss.ResponseOK {
		t.Errorf("Response type = %s, want OK", resp.Type)
	}
	if len(resp.Hosts) != 1 {
		t.Errorf("Hosts count = %d, want 1", len(resp.Hosts))
	}
}

func TestSocketServer_HandleServicesList(t *testing.T) {
	healthCfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	store := NewRecordStore(time.Hour, healthCfg, nil)
	defer store.Stop()

	store.AddOrUpdate(&nss.Record{
		Hostname:  "host1",
		Addresses: []string{"1.1.1.1"},
		Services:  map[string]string{"ssh": "tcp:22", "http": "tcp:80"},
	})

	server := NewSocketServer("/tmp/test.sock", store)

	query := &nss.Query{
		Type:      nss.ServicesList,
		RequestID: "req-1",
	}

	resp := server.handleServicesList(query)

	if resp.Type != nss.ResponseOK {
		t.Errorf("Response type = %s, want OK", resp.Type)
	}
	if resp.Count != 2 {
		t.Errorf("Count = %d, want 2", resp.Count)
	}
}

func TestSocketServer_HandleHostsShow(t *testing.T) {
	healthCfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	store := NewRecordStore(time.Hour, healthCfg, nil)
	defer store.Stop()

	store.AddOrUpdate(&nss.Record{
		Hostname:  "host1",
		Addresses: []string{"1.1.1.1"},
		Services:  map[string]string{"ssh": "tcp:22"},
	})

	server := NewSocketServer("/tmp/test.sock", store)

	query := &nss.Query{
		Type:      nss.HostsShow,
		RequestID: "req-1",
		Name:      "host1",
	}

	resp := server.handleHostsShow(query)

	if resp.Type != nss.ResponseOK {
		t.Errorf("Response type = %s, want OK", resp.Type)
	}
	if len(resp.Hosts) != 1 {
		t.Errorf("Hosts count = %d, want 1", len(resp.Hosts))
	}
	if resp.Hosts[0].Hostname != "host1" {
		t.Errorf("Hostname = %s, want host1", resp.Hosts[0].Hostname)
	}
}

func TestSocketServer_HandleHostsShowNotFound(t *testing.T) {
	healthCfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	store := NewRecordStore(time.Hour, healthCfg, nil)
	defer store.Stop()

	server := NewSocketServer("/tmp/test.sock", store)

	query := &nss.Query{
		Type:      nss.HostsShow,
		RequestID: "req-1",
		Name:      "nonexistent",
	}

	resp := server.handleHostsShow(query)

	if resp.Type != nss.ResponseNotFound {
		t.Errorf("Response type = %s, want NotFound", resp.Type)
	}
}

func TestSocketServer_HandleHostsForget(t *testing.T) {
	healthCfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	store := NewRecordStore(time.Hour, healthCfg, nil)
	defer store.Stop()

	store.AddOrUpdate(&nss.Record{
		Hostname:  "host1",
		Addresses: []string{"1.1.1.1"},
	})

	server := NewSocketServer("/tmp/test.sock", store)

	query := &nss.Query{
		Type:      nss.HostsForget,
		RequestID: "req-1",
		Name:      "host1",
	}

	resp := server.handleHostsForget(query)

	if resp.Type != nss.ResponseOK {
		t.Errorf("Response type = %s, want OK", resp.Type)
	}

	_, exists := store.Get("host1")
	if exists {
		t.Error("host1 should be forgotten")
	}
}

func TestSocketServer_HandleHostsMarkLost(t *testing.T) {
	healthCfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	store := NewRecordStore(time.Hour, healthCfg, nil)
	defer store.Stop()

	store.AddOrUpdate(&nss.Record{
		Hostname:  "host1",
		Addresses: []string{"1.1.1.1"},
	})

	server := NewSocketServer("/tmp/test.sock", store)

	query := &nss.Query{
		Type:      nss.HostsMarkLost,
		RequestID: "req-1",
		Name:      "host1",
	}

	resp := server.handleHostsMarkLost(query)

	if resp.Type != nss.ResponseOK {
		t.Errorf("Response type = %s, want OK", resp.Type)
	}

	store.mu.Lock()
	record, exists := store.records["host1"]
	store.mu.Unlock()

	if !exists {
		t.Fatal("host1 should still exist")
	}
	if record.Status != nss.StatusLost {
		t.Errorf("Status = %s, want lost", record.Status)
	}
}

func TestSocketServer_HandleHostsMarkLostNotFound(t *testing.T) {
	healthCfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	store := NewRecordStore(time.Hour, healthCfg, nil)
	defer store.Stop()

	server := NewSocketServer("/tmp/test.sock", store)

	query := &nss.Query{
		Type:      nss.HostsMarkLost,
		RequestID: "req-1",
		Name:      "nonexistent",
	}

	resp := server.handleHostsMarkLost(query)

	if resp.Type != nss.ResponseOK {
		t.Errorf("Response type = %s, want OK (MarkLost always succeeds)", resp.Type)
	}
}

func TestSocketServer_HandleServicesShow(t *testing.T) {
	healthCfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	store := NewRecordStore(time.Hour, healthCfg, nil)
	defer store.Stop()

	store.AddOrUpdate(&nss.Record{
		Hostname:  "host1",
		Addresses: []string{"1.1.1.1"},
		Services:  map[string]string{"ssh": "tcp:22", "http": "tcp:80"},
	})

	server := NewSocketServer("/tmp/test.sock", store)

	query := &nss.Query{
		Type:      nss.ServicesShow,
		RequestID: "req-1",
		Name:      "ssh",
	}

	resp := server.handleServicesShow(query)

	if resp.Type != nss.ResponseOK {
		t.Errorf("Response type = %s, want OK", resp.Type)
	}
	if resp.Count != 1 {
		t.Errorf("Count = %d, want 1", resp.Count)
	}
}

func TestSocketServer_HandleServicesShowNotFound(t *testing.T) {
	healthCfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	store := NewRecordStore(time.Hour, healthCfg, nil)
	defer store.Stop()

	server := NewSocketServer("/tmp/test.sock", store)

	query := &nss.Query{
		Type:      nss.ServicesShow,
		RequestID: "req-1",
		Name:      "nonexistent",
	}

	resp := server.handleServicesShow(query)

	if resp.Type != nss.ResponseNotFound {
		t.Errorf("Response type = %s, want NotFound", resp.Type)
	}
}

func TestSocketServer_HandleServicesForget(t *testing.T) {
	healthCfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	store := NewRecordStore(time.Hour, healthCfg, nil)
	defer store.Stop()

	server := NewSocketServer("/tmp/test.sock", store)

	query := &nss.Query{
		Type:      nss.ServicesForget,
		RequestID: "req-1",
		Name:      "ssh",
	}

	resp := server.handleServicesForget(query)

	if resp.Type != nss.ResponseError {
		t.Errorf("Response type = %s, want Error", resp.Type)
	}
}

func TestSocketServer_HandleQuery(t *testing.T) {
	healthCfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	store := NewRecordStore(time.Hour, healthCfg, nil)
	defer store.Stop()

	store.AddOrUpdate(&nss.Record{
		Hostname:  "host1",
		Addresses: []string{"1.1.1.1"},
		Services:  map[string]string{"ssh": "tcp:22"},
	})

	server := NewSocketServer("/tmp/test.sock", store)

	tests := []struct {
		name      string
		queryType nss.MessageType
	}{
		{"QueryByName", nss.QueryByName},
		{"QueryByAddr", nss.QueryByAddr},
		{"QueryList", nss.QueryList},
		{"QueryListHosts", nss.QueryListHosts},
		{"QueryListServices", nss.QueryListServices},
		{"HostsList", nss.HostsList},
		{"ServicesList", nss.ServicesList},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &nss.Query{
				Type:      tt.queryType,
				RequestID: "req-1",
			}
			if tt.queryType == nss.QueryByName {
				query.Name = "host1"
			}
			if tt.queryType == nss.QueryByAddr {
				query.Addr = "1.1.1.1"
			}

			resp := server.handleQuery(query)
			if resp.Type == nss.ResponseError && resp.Error == "unknown query type" {
				t.Errorf("Query type %s not handled", tt.queryType)
			}
		})
	}
}

func TestSocketServer_HandleQueryUnknown(t *testing.T) {
	healthCfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	store := NewRecordStore(time.Hour, healthCfg, nil)
	defer store.Stop()

	server := NewSocketServer("/tmp/test.sock", store)

	query := &nss.Query{
		Type:      "UNKNOWN_TYPE",
		RequestID: "req-1",
	}

	resp := server.handleQuery(query)

	if resp.Type != nss.ResponseError {
		t.Errorf("Response type = %s, want Error", resp.Type)
	}
	if resp.Error != "unknown query type" {
		t.Errorf("Error = %s, want 'unknown query type'", resp.Error)
	}
}

func TestSocketServer_HandleQueryList(t *testing.T) {
	healthCfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	store := NewRecordStore(time.Hour, healthCfg, nil)
	defer store.Stop()

	store.AddOrUpdate(&nss.Record{
		Hostname:  "host1",
		Addresses: []string{"1.1.1.1"},
		Services:  map[string]string{"ssh": "tcp:22"},
	})

	server := NewSocketServer("/tmp/test.sock", store)

	query := &nss.Query{
		Type:      nss.QueryList,
		RequestID: "req-1",
	}

	resp := server.handleQueryList(query)

	if resp.Type != nss.ResponseOK {
		t.Errorf("Response type = %s, want OK", resp.Type)
	}
}

func TestSocketServer_HandleQueryListHosts(t *testing.T) {
	healthCfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	store := NewRecordStore(time.Hour, healthCfg, nil)
	defer store.Stop()

	store.AddOrUpdate(&nss.Record{
		Hostname:  "host1",
		Addresses: []string{"1.1.1.1"},
	})

	server := NewSocketServer("/tmp/test.sock", store)

	query := &nss.Query{
		Type:      nss.QueryListHosts,
		RequestID: "req-1",
	}

	resp := server.handleQueryListHosts(query)

	if resp.Type != nss.ResponseOK {
		t.Errorf("Response type = %s, want OK", resp.Type)
	}
}

func TestSocketServer_HandleQueryListServices(t *testing.T) {
	healthCfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	store := NewRecordStore(time.Hour, healthCfg, nil)
	defer store.Stop()

	store.AddOrUpdate(&nss.Record{
		Hostname:  "host1",
		Addresses: []string{"1.1.1.1"},
		Services:  map[string]string{"ssh": "tcp:22"},
	})

	server := NewSocketServer("/tmp/test.sock", store)

	query := &nss.Query{
		Type:      nss.QueryListServices,
		RequestID: "req-1",
	}

	resp := server.handleQueryListServices(query)

	if resp.Type != nss.ResponseOK {
		t.Errorf("Response type = %s, want OK", resp.Type)
	}
}

func TestSocketServer_HandleTimeStatus(t *testing.T) {
	healthCfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	store := NewRecordStore(time.Hour, healthCfg, nil)
	defer store.Stop()

	server := NewSocketServer("/tmp/test.sock", store)

	query := &nss.Query{
		Type:      "TIME_STATUS",
		RequestID: "req-1",
	}

	resp := server.handleTimeStatus(query)

	if resp.Type != nss.ResponseError {
		t.Errorf("Response type = %s, want Error (time sync not enabled)", resp.Type)
	}
}

func TestSocketServer_HandleTimeForceUpdate(t *testing.T) {
	healthCfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	store := NewRecordStore(time.Hour, healthCfg, nil)
	defer store.Stop()

	server := NewSocketServer("/tmp/test.sock", store)

	query := &nss.Query{
		Type:      "TIME_FORCE_UPDATE",
		RequestID: "req-1",
	}

	resp := server.handleTimeForceUpdate(query)

	if resp.Type != nss.ResponseError {
		t.Errorf("Response type = %s, want Error (time sync not enabled)", resp.Type)
	}
}

func TestHealthTracker_ComputeStatus(t *testing.T) {
	cfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	ht := NewHealthTracker(cfg, nil)

	now := time.Now().Unix()

	tests := []struct {
		name     string
		record   *nss.Record
		expected nss.HostStatus
	}{
		{
			name: "static record",
			record: &nss.Record{
				IsStatic: true,
			},
			expected: nss.StatusStatic,
		},
		{
			name: "fresh record",
			record: &nss.Record{
				Timestamp: now,
				IsStatic:  false,
			},
			expected: nss.StatusHealthy,
		},
		{
			name: "stale record",
			record: &nss.Record{
				Timestamp: now - 60,
				IsStatic:  false,
			},
			expected: nss.StatusStale,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ht.ComputeStatus(tt.record)
			if result != tt.expected {
				t.Errorf("ComputeStatus() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestHealthTracker_GetStaticHosts(t *testing.T) {
	staticHosts := map[string]config.StaticHost{
		"static1": {Addresses: []string{"1.1.1.1"}},
	}
	cfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	ht := NewHealthTracker(cfg, staticHosts)

	hosts := ht.GetStaticHosts()
	if len(hosts) != 1 {
		t.Errorf("GetStaticHosts() count = %d, want 1", len(hosts))
	}
}

func TestHealthTracker_IsStatic(t *testing.T) {
	staticHosts := map[string]config.StaticHost{
		"static1": {Addresses: []string{"1.1.1.1"}},
	}
	cfg := &config.HealthConfig{
		GracePeriod: 30 * time.Second,
		ExpireAfter: 5 * time.Minute,
	}
	ht := NewHealthTracker(cfg, staticHosts)

	if !ht.IsStatic("static1") {
		t.Error("IsStatic(static1) = false, want true")
	}
	if ht.IsStatic("nonexistent") {
		t.Error("IsStatic(nonexistent) = true, want false")
	}
}
