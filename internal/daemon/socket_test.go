package daemon

import (
	"testing"
	"time"

	"github.com/offline-lab/disco/internal/config"
	"github.com/offline-lab/disco/internal/nss"
)

func newTestSocketStore() *RecordStore {
	return NewRecordStore(3600*time.Second, &config.HealthConfig{
		GracePeriod:     60 * time.Second,
		ExpireAfter:     3600 * time.Second,
		CleanupInterval: 30 * time.Second,
	}, nil)
}

func TestSocketServer_handleQueryByName(t *testing.T) {
	store := newTestSocketStore()
	server := NewSocketServer("/tmp/test.sock", store)

	store.AddOrUpdate(&nss.Record{
		Hostname:  "test-host",
		Addresses: []string{"192.168.1.10", "192.168.1.11"},
		Timestamp: time.Now().Unix(),
		TTL:       3600,
	})

	query := &nss.Query{
		Type:      nss.QueryByName,
		Name:      "test-host",
		RequestID: "test-001",
	}

	resp := server.handleQueryByName(query)

	if resp.Type != nss.ResponseOK {
		t.Errorf("Expected OK, got %s", resp.Type)
	}

	if resp.Name != "test-host" {
		t.Errorf("Expected name test-host, got %s", resp.Name)
	}

	if len(resp.Addrs) != 2 {
		t.Errorf("Expected 2 addresses, got %d", len(resp.Addrs))
	}
}

func TestSocketServer_handleQueryByName_NotFound(t *testing.T) {
	store := newTestSocketStore()
	server := NewSocketServer("/tmp/test.sock", store)

	query := &nss.Query{
		Type:      nss.QueryByName,
		Name:      "nonexistent",
		RequestID: "test-002",
	}

	resp := server.handleQueryByName(query)

	if resp.Type != nss.ResponseNotFound {
		t.Errorf("Expected NOTFOUND, got %s", resp.Type)
	}
}

func TestSocketServer_handleQueryByAddr(t *testing.T) {
	store := newTestSocketStore()
	server := NewSocketServer("/tmp/test.sock", store)

	store.AddOrUpdate(&nss.Record{
		Hostname:  "test-host",
		Addresses: []string{"192.168.1.10"},
		Timestamp: time.Now().Unix(),
		TTL:       3600,
	})

	query := &nss.Query{
		Type:      nss.QueryByAddr,
		Addr:      "192.168.1.10",
		RequestID: "test-003",
	}

	resp := server.handleQueryByAddr(query)

	if resp.Type != nss.ResponseOK {
		t.Errorf("Expected OK, got %s", resp.Type)
	}

	if resp.Name != "test-host" {
		t.Errorf("Expected name test-host, got %s", resp.Name)
	}
}

func TestSocketServer_handleQueryByAddr_NotFound(t *testing.T) {
	store := newTestSocketStore()
	server := NewSocketServer("/tmp/test.sock", store)

	query := &nss.Query{
		Type:      nss.QueryByAddr,
		Addr:      "192.168.1.99",
		RequestID: "test-004",
	}

	resp := server.handleQueryByAddr(query)

	if resp.Type != nss.ResponseNotFound {
		t.Errorf("Expected NOTFOUND, got %s", resp.Type)
	}
}

func TestSocketServer_handleQueryList(t *testing.T) {
	store := newTestSocketStore()
	server := NewSocketServer("/tmp/test.sock", store)

	store.AddOrUpdate(&nss.Record{
		Hostname:  "host1",
		Addresses: []string{"192.168.1.10"},
		Timestamp: time.Now().Unix(),
		TTL:       3600,
	})

	store.AddOrUpdate(&nss.Record{
		Hostname:  "host2",
		Addresses: []string{"192.168.1.11"},
		Timestamp: time.Now().Unix(),
		TTL:       3600,
	})

	query := &nss.Query{
		Type:      nss.QueryList,
		RequestID: "test-005",
	}

	resp := server.handleQueryList(query)

	if resp.Type != nss.ResponseOK {
		t.Errorf("Expected OK, got %s", resp.Type)
	}
}

func TestSocketServer_handleQueryListHosts(t *testing.T) {
	store := newTestSocketStore()
	server := NewSocketServer("/tmp/test.sock", store)

	store.AddOrUpdate(&nss.Record{
		Hostname:  "webserver",
		Addresses: []string{"192.168.1.10"},
		Services: map[string]string{
			"www": "192.168.1.10:80",
		},
		Timestamp: time.Now().Unix(),
		TTL:       3600,
	})

	query := &nss.Query{
		Type:      nss.QueryListHosts,
		RequestID: "test-006",
	}

	resp := server.handleQueryListHosts(query)

	if resp.Type != nss.ResponseOK {
		t.Errorf("Expected OK, got %s", resp.Type)
	}
}

func TestSocketServer_handleQueryListServices(t *testing.T) {
	store := newTestSocketStore()
	server := NewSocketServer("/tmp/test.sock", store)

	store.AddOrUpdate(&nss.Record{
		Hostname:  "mailserver",
		Addresses: []string{"192.168.1.20"},
		Services: map[string]string{
			"smtp": "192.168.1.20:25",
			"imap": "192.168.1.20:143",
		},
		Timestamp: time.Now().Unix(),
		TTL:       3600,
	})

	query := &nss.Query{
		Type:      nss.QueryListServices,
		RequestID: "test-007",
	}

	resp := server.handleQueryListServices(query)

	if resp.Type != nss.ResponseOK {
		t.Errorf("Expected OK, got %s", resp.Type)
	}
}

func TestSocketServer_handleQuery_UnknownType(t *testing.T) {
	store := newTestSocketStore()
	server := NewSocketServer("/tmp/test.sock", store)

	query := &nss.Query{
		Type:      "UNKNOWN_TYPE",
		RequestID: "test-008",
	}

	resp := server.handleQuery(query)

	if resp.Type != nss.ResponseError {
		t.Errorf("Expected ERROR, got %s", resp.Type)
	}
}

func TestSocketServer_handleQueryByName_Alias(t *testing.T) {
	store := newTestSocketStore()
	server := NewSocketServer("/tmp/test.sock", store)

	store.AddOrUpdate(&nss.Record{
		Hostname:  "web.local",
		Aliases:   []string{"shop.local", "blog.local"},
		Addresses: []string{"192.168.1.10"},
		Timestamp: time.Now().Unix(),
		TTL:       3600,
	})

	for _, alias := range []string{"shop.local", "blog.local"} {
		resp := server.handleQueryByName(&nss.Query{
			Type:      nss.QueryByName,
			Name:      alias,
			RequestID: "alias-test",
		})
		if resp.Type != nss.ResponseOK {
			t.Errorf("Get(%q): expected OK, got %s", alias, resp.Type)
		}
		if len(resp.Addrs) != 1 || resp.Addrs[0] != "192.168.1.10" {
			t.Errorf("Get(%q): expected [192.168.1.10], got %v", alias, resp.Addrs)
		}
	}
}

func TestSocketServer_handleServiceAnnounce(t *testing.T) {
	store := newTestSocketStore()
	server := NewSocketServer("/tmp/test.sock", store)

	announced := make(map[string]struct{})
	server.SetAnnounceService(func(name string, port int, aliases []string) error {
		announced[name] = struct{}{}
		return nil
	})

	resp := server.handleServiceAnnounce(&nss.Query{
		Type:      nss.ServiceAnnounce,
		Name:      "myapp",
		Port:      8080,
		Aliases:   []string{"myapp.local"},
		RequestID: "svc-001",
	})

	if resp.Type != nss.ResponseOK {
		t.Errorf("Expected OK, got %s: %s", resp.Type, resp.Error)
	}
	if resp.Name != "myapp" {
		t.Errorf("Expected name myapp, got %s", resp.Name)
	}
	if _, ok := announced["myapp"]; !ok {
		t.Error("AnnounceService callback was not called")
	}
}

func TestSocketServer_handleServiceAnnounce_MissingName(t *testing.T) {
	store := newTestSocketStore()
	server := NewSocketServer("/tmp/test.sock", store)
	server.SetAnnounceService(func(name string, port int, aliases []string) error { return nil })

	resp := server.handleServiceAnnounce(&nss.Query{
		Type:      nss.ServiceAnnounce,
		Port:      8080,
		RequestID: "svc-002",
	})

	if resp.Type != nss.ResponseError {
		t.Errorf("Expected ERROR for missing name, got %s", resp.Type)
	}
}

func TestSocketServer_handleServiceAnnounce_InvalidPort(t *testing.T) {
	store := newTestSocketStore()
	server := NewSocketServer("/tmp/test.sock", store)
	server.SetAnnounceService(func(name string, port int, aliases []string) error { return nil })

	resp := server.handleServiceAnnounce(&nss.Query{
		Type:      nss.ServiceAnnounce,
		Name:      "myapp",
		Port:      0,
		RequestID: "svc-003",
	})

	if resp.Type != nss.ResponseError {
		t.Errorf("Expected ERROR for invalid port, got %s", resp.Type)
	}
}

func TestSocketServer_handleServiceAnnounce_NoCallback(t *testing.T) {
	store := newTestSocketStore()
	server := NewSocketServer("/tmp/test.sock", store)
	// no SetAnnounceService called

	resp := server.handleServiceAnnounce(&nss.Query{
		Type:      nss.ServiceAnnounce,
		Name:      "myapp",
		Port:      8080,
		RequestID: "svc-004",
	})

	if resp.Type != nss.ResponseError {
		t.Errorf("Expected ERROR when no callback set, got %s", resp.Type)
	}
}
