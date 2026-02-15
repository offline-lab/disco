package daemon

import (
	"testing"
	"time"

	"github.com/offline-lab/disco/internal/nss"
)

func TestSocketServer_handleQueryByName(t *testing.T) {
	store := NewRecordStore(3600 * time.Second)
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
	store := NewRecordStore(3600 * time.Second)
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
	store := NewRecordStore(3600 * time.Second)
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
	store := NewRecordStore(3600 * time.Second)
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
	store := NewRecordStore(3600 * time.Second)
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
	store := NewRecordStore(3600 * time.Second)
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
	store := NewRecordStore(3600 * time.Second)
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
	store := NewRecordStore(3600 * time.Second)
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
