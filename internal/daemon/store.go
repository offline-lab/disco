package daemon

import (
	"log"
	"sync"
	"time"

	"github.com/offline-lab/disco/internal/nss"
)

// RecordStore manages the in-memory cache of host records
type RecordStore struct {
	mu       sync.RWMutex
	records  map[string]*nss.Record
	ttl      time.Duration
	stopChan chan struct{}
}

// NewRecordStore creates a new record store
func NewRecordStore(ttl time.Duration) *RecordStore {
	rs := &RecordStore{
		records:  make(map[string]*nss.Record),
		ttl:      ttl,
		stopChan: make(chan struct{}),
	}
	go rs.cleanupExpiredRecords()
	return rs
}

// Stop halts the background cleanup goroutine
func (rs *RecordStore) Stop() {
	close(rs.stopChan)
}

// Get retrieves a record by hostname
func (rs *RecordStore) Get(hostname string) (*nss.Record, bool) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	record, exists := rs.records[hostname]
	if !exists {
		return nil, false
	}

	if time.Now().Unix() >= (record.Timestamp + record.TTL) {
		return nil, false
	}

	return record, true
}

// GetByAddr retrieves a record by IP address
func (rs *RecordStore) GetByAddr(addr string) (*nss.Record, bool) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	for _, record := range rs.records {
		if time.Now().Unix() >= (record.Timestamp + record.TTL) {
			continue
		}

		for _, a := range record.Addresses {
			if a == addr {
				return record, true
			}
		}
	}

	return nil, false
}

// AddOrUpdate adds or updates a record
func (rs *RecordStore) AddOrUpdate(record *nss.Record) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if record.TTL == 0 {
		record.TTL = int64(rs.ttl.Seconds())
	}
	record.Timestamp = time.Now().Unix()

	rs.records[record.Hostname] = record
}

// Delete removes a record
func (rs *RecordStore) Delete(hostname string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	delete(rs.records, hostname)
}

// List returns all records
func (rs *RecordStore) List() []*nss.Record {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	var records []*nss.Record
	now := time.Now().Unix()

	log.Printf("List: total records in map: %d", len(rs.records))
	for _, record := range rs.records {
		log.Printf("List: record %s, timestamp=%d, ttl=%d, now=%d, expired=%v",
			record.Hostname, record.Timestamp, record.TTL, now, now > (record.Timestamp+record.TTL))
		if now > (record.Timestamp + record.TTL) {
			continue
		}
		records = append(records, record)
	}
	log.Printf("List: returning %d valid records", len(records))

	return records
}

// cleanupExpiredRecords periodically removes expired records
func (rs *RecordStore) cleanupExpiredRecords() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-rs.stopChan:
			return
		case <-ticker.C:
			rs.mu.Lock()
			now := time.Now().Unix()

			for hostname, record := range rs.records {
				if now > (record.Timestamp + record.TTL) {
					delete(rs.records, hostname)
				}
			}
			rs.mu.Unlock()
		}
	}
}
