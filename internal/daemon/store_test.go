package daemon

import (
	"testing"
	"time"

	"github.com/offline-lab/disco/internal/config"
	"github.com/offline-lab/disco/internal/nss"
)

func newTestStore(ttl time.Duration) *RecordStore {
	return NewRecordStore(ttl, &config.HealthConfig{
		GracePeriod:     60 * time.Second,
		ExpireAfter:     3600 * time.Second,
		CleanupInterval: 30 * time.Second,
	}, nil)
}

func TestRecordStore_New(t *testing.T) {
	ttl := 3600 * time.Second
	store := newTestStore(ttl)

	if store == nil {
		t.Fatal("newTestStore() returned nil")
	}

	if store.ttl != ttl {
		t.Errorf("Expected TTL %v, got %v", ttl, store.ttl)
	}
}

func TestRecordStore_AddOrUpdate(t *testing.T) {
	store := newTestStore(3600 * time.Second)

	record := &nss.Record{
		Hostname:  "test-host",
		Addresses: []string{"192.168.1.10"},
		Timestamp: time.Now().Unix(),
		TTL:       3600,
	}

	store.AddOrUpdate(record)

	retrieved, exists := store.Get("test-host")
	if !exists {
		t.Fatal("Record not found after AddOrUpdate()")
	}

	if retrieved.Hostname != "test-host" {
		t.Errorf("Expected hostname test-host, got %s", retrieved.Hostname)
	}

	if len(retrieved.Addresses) != 1 {
		t.Errorf("Expected 1 address, got %d", len(retrieved.Addresses))
	}

	if retrieved.Addresses[0] != "192.168.1.10" {
		t.Errorf("Expected address 192.168.1.10, got %s", retrieved.Addresses[0])
	}
}

func TestRecordStore_GetNonExistent(t *testing.T) {
	store := newTestStore(3600 * time.Second)

	_, exists := store.Get("nonexistent")
	if exists {
		t.Error("Get() returned true for non-existent record")
	}
}

func TestRecordStore_Delete(t *testing.T) {
	store := newTestStore(3600 * time.Second)

	record := &nss.Record{
		Hostname:  "test-host",
		Addresses: []string{"192.168.1.10"},
		Timestamp: time.Now().Unix(),
		TTL:       3600,
	}

	store.AddOrUpdate(record)
	store.Delete("test-host")

	_, exists := store.Get("test-host")
	if exists {
		t.Error("Record still exists after Delete()")
	}
}

func TestRecordStore_List(t *testing.T) {
	store := newTestStore(3600 * time.Second)

	record1 := &nss.Record{
		Hostname:  "host1",
		Addresses: []string{"192.168.1.10"},
		Timestamp: time.Now().Unix(),
		TTL:       3600,
	}

	record2 := &nss.Record{
		Hostname:  "host2",
		Addresses: []string{"192.168.1.11"},
		Timestamp: time.Now().Unix(),
		TTL:       3600,
	}

	store.AddOrUpdate(record1)
	store.AddOrUpdate(record2)

	all := store.List()
	if len(all) != 2 {
		t.Errorf("Expected 2 records, got %d", len(all))
	}

	hostnames := make(map[string]bool)
	for _, r := range all {
		hostnames[r.Hostname] = true
	}

	if !hostnames["host1"] || !hostnames["host2"] {
		t.Error("Missing expected hostnames in List()")
	}
}

func TestRecordStore_ListEmpty(t *testing.T) {
	store := newTestStore(3600 * time.Second)

	all := store.List()
	if len(all) != 0 {
		t.Errorf("Expected 0 records, got %d", len(all))
	}
}

func TestRecordStore_GetByAddr(t *testing.T) {
	store := newTestStore(3600 * time.Second)

	record := &nss.Record{
		Hostname:  "test-host",
		Addresses: []string{"192.168.1.10", "192.168.1.11"},
		Timestamp: time.Now().Unix(),
		TTL:       3600,
	}

	store.AddOrUpdate(record)

	// Test with first address
	retrieved, exists := store.GetByAddr("192.168.1.10")
	if !exists {
		t.Fatal("Record not found by address 192.168.1.10")
	}

	if retrieved.Hostname != "test-host" {
		t.Errorf("Expected hostname test-host, got %s", retrieved.Hostname)
	}

	// Test with second address
	retrieved, exists = store.GetByAddr("192.168.1.11")
	if !exists {
		t.Fatal("Record not found by address 192.168.1.11")
	}

	// Test with non-existent address
	_, exists = store.GetByAddr("192.168.1.99")
	if exists {
		t.Error("GetByAddr() returned true for non-existent address")
	}
}

func TestRecordStore_UpdateRecord(t *testing.T) {
	store := newTestStore(3600 * time.Second)

	record1 := &nss.Record{
		Hostname:  "test-host",
		Addresses: []string{"192.168.1.10"},
		Timestamp: time.Now().Unix(),
		TTL:       3600,
	}

	store.AddOrUpdate(record1)

	record2 := &nss.Record{
		Hostname:  "test-host",
		Addresses: []string{"192.168.1.10", "192.168.1.11"},
		Timestamp: time.Now().Unix(),
		TTL:       3600,
	}

	store.AddOrUpdate(record2)

	retrieved, _ := store.Get("test-host")
	if len(retrieved.Addresses) != 2 {
		t.Errorf("Expected 2 addresses after update, got %d", len(retrieved.Addresses))
	}
}

func TestRecordStore_Expiration(t *testing.T) {
	// With health-based tracking, records transition to "lost" status
	// when they exceed ExpireAfter. The Get() method excludes lost records.
	store := newTestStore(3600 * time.Second)

	record := &nss.Record{
		Hostname:  "test-host",
		Addresses: []string{"192.168.1.10"},
		Timestamp: 0, // Very old timestamp - will be marked as lost
		TTL:       3600,
		IsStatic:  false,
	}

	// Add directly to store without AddOrUpdate (which would reset timestamp)
	store.mu.Lock()
	store.records[record.Hostname] = record
	store.healthTracker.UpdateRecordStatus(record)
	store.mu.Unlock()

	// Record should be marked as lost
	if record.Status != nss.StatusLost {
		t.Errorf("Expected status %s, got %s", nss.StatusLost, record.Status)
	}

	// Get() should not return lost records
	_, exists := store.Get("test-host")
	if exists {
		t.Error("Lost record should not be returned by Get()")
	}
}

func TestRecordStore_ConcurrentAccess(t *testing.T) {
	store := newTestStore(3600 * time.Second)

	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(n int) {
			hostname := string(rune('a' + n))
			record := &nss.Record{
				Hostname:  hostname,
				Addresses: []string{"192.168.1.10"},
				Timestamp: time.Now().Unix(),
				TTL:       3600,
			}
			store.AddOrUpdate(record)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(n int) {
			hostname := string(rune('a' + n))
			store.Get(hostname)
			done <- true
		}(i)
	}

	// Wait for all reads
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestRecordStore_RecordWithServices(t *testing.T) {
	store := newTestStore(3600 * time.Second)

	record := &nss.Record{
		Hostname:  "test-host",
		Addresses: []string{"192.168.1.10"},
		Timestamp: time.Now().Unix(),
		TTL:       3600,
		Services: map[string]string{
			"www":  "192.168.1.10:80",
			"smtp": "192.168.1.10:25",
		},
	}

	store.AddOrUpdate(record)

	retrieved, exists := store.Get("test-host")
	if !exists {
		t.Fatal("Record not found")
	}

	if len(retrieved.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(retrieved.Services))
	}

	if retrieved.Services["www"] != "192.168.1.10:80" {
		t.Errorf("Expected www service, got %s", retrieved.Services["www"])
	}
}

func TestRecordStore_RecordWithAliases(t *testing.T) {
	store := newTestStore(3600 * time.Second)

	record := &nss.Record{
		Hostname:  "test-host",
		Aliases:   []string{"alias1", "alias2"},
		Addresses: []string{"192.168.1.10"},
		Timestamp: time.Now().Unix(),
		TTL:       3600,
	}

	store.AddOrUpdate(record)

	retrieved, exists := store.Get("test-host")
	if !exists {
		t.Fatal("Record not found")
	}

	if len(retrieved.Aliases) != 2 {
		t.Errorf("Expected 2 aliases, got %d", len(retrieved.Aliases))
	}

	if retrieved.Aliases[0] != "alias1" {
		t.Errorf("Expected alias1, got %s", retrieved.Aliases[0])
	}
}

func TestRecordStore_DefaultTTL(t *testing.T) {
	store := newTestStore(7200 * time.Second)

	record := &nss.Record{
		Hostname:  "test-host",
		Addresses: []string{"192.168.1.10"},
		Timestamp: 0, // Will be set by AddOrUpdate
		TTL:       0, // Should use store's default
	}

	store.AddOrUpdate(record)

	retrieved, _ := store.Get("test-host")
	if retrieved.TTL != 7200 {
		t.Errorf("Expected TTL 7200, got %d", retrieved.TTL)
	}
}

func TestRecordStore_MultipleRecords(t *testing.T) {
	store := newTestStore(3600 * time.Second)

	for i := 0; i < 100; i++ {
		record := &nss.Record{
			Hostname:  string(rune('a'+i%26)) + string(rune('a'+i/26)),
			Addresses: []string{"192.168.1.10"},
			Timestamp: time.Now().Unix(),
			TTL:       3600,
		}
		store.AddOrUpdate(record)
	}

	all := store.List()
	if len(all) != 100 {
		t.Errorf("Expected 100 records, got %d", len(all))
	}
}

func TestRecordStore_GetByAddrExpired(t *testing.T) {
	store := newTestStore(3600 * time.Second)

	record := &nss.Record{
		Hostname:  "test-host",
		Addresses: []string{"192.168.1.10"},
		Timestamp: 0, // Very old - will be lost
		TTL:       3600,
		IsStatic:  false,
	}

	// Add directly without AddOrUpdate which would reset timestamp
	store.mu.Lock()
	store.records[record.Hostname] = record
	store.healthTracker.UpdateRecordStatus(record)
	store.mu.Unlock()

	// Should not be found because it's lost
	_, exists := store.GetByAddr("192.168.1.10")
	if exists {
		t.Error("Lost record should not be found via GetByAddr()")
	}
}

func TestRecordStore_ListExcludesExpired(t *testing.T) {
	store := newTestStore(3600 * time.Second)

	record := &nss.Record{
		Hostname:  "test-host",
		Addresses: []string{"192.168.1.10"},
		Timestamp: 0, // Very old - will be lost
		TTL:       3600,
	}

	store.mu.Lock()
	store.records[record.Hostname] = record
	store.healthTracker.UpdateRecordStatus(record)
	store.mu.Unlock()

	records := store.List()
	if len(records) != 0 {
		t.Errorf("Expected 0 records for lost record, got %d", len(records))
	}
}
