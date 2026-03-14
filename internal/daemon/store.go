package daemon

import (
	"sync"
	"time"

	"github.com/offline-lab/disco/internal/config"
	dnsserver "github.com/offline-lab/disco/internal/dns"
	"github.com/offline-lab/disco/internal/nss"
)

type RecordStore struct {
	mu            sync.RWMutex
	records       map[string]*nss.Record
	ttl           time.Duration
	healthConfig  *config.HealthConfig
	healthTracker *HealthTracker
	stopChan      chan struct{}
}

func NewRecordStore(ttl time.Duration, healthCfg *config.HealthConfig, staticHosts map[string]config.StaticHost) *RecordStore {
	rs := &RecordStore{
		records:       make(map[string]*nss.Record),
		ttl:           ttl,
		healthConfig:  healthCfg,
		healthTracker: NewHealthTracker(healthCfg, staticHosts),
		stopChan:      make(chan struct{}),
	}

	for hostname, record := range rs.healthTracker.GetStaticHosts() {
		rs.records[hostname] = record
	}

	go rs.cleanupExpiredRecords()
	return rs
}

// Stop halts the background cleanup goroutine
func (rs *RecordStore) Stop() {
	close(rs.stopChan)
}

func (rs *RecordStore) Get(hostname string) (*nss.Record, bool) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	record, exists := rs.records[hostname]
	if !exists {
		return nil, false
	}

	if record.IsStatic {
		return record, true
	}

	rs.healthTracker.UpdateRecordStatus(record)
	if record.Status == nss.StatusLost {
		return nil, false
	}

	return record, true
}

// GetByAddr retrieves a record by IP address
func (rs *RecordStore) GetByAddr(addr string) (*nss.Record, bool) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	for _, record := range rs.records {
		if record.IsStatic {
			for _, a := range record.Addresses {
				if a == addr {
					return record, true
				}
			}
			continue
		}

		rs.healthTracker.UpdateRecordStatus(record)
		if record.Status == nss.StatusLost {
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

func (rs *RecordStore) AddOrUpdate(record *nss.Record) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.healthTracker.IsStatic(record.Hostname) {
		return
	}

	if record.TTL == 0 {
		record.TTL = int64(rs.ttl.Seconds())
	}

	now := time.Now().Unix()
	if existing, exists := rs.records[record.Hostname]; exists {
		record.FirstSeen = existing.FirstSeen
	} else {
		record.FirstSeen = now
	}
	record.Timestamp = now
	record.IsStatic = false
	rs.healthTracker.UpdateRecordStatus(record)

	rs.records[record.Hostname] = record
}

// Delete removes a record
func (rs *RecordStore) Delete(hostname string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	delete(rs.records, hostname)
}

func (rs *RecordStore) List() []*nss.Record {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	var records []*nss.Record

	for _, record := range rs.records {
		if record.IsStatic {
			records = append(records, record)
			continue
		}
		rs.healthTracker.UpdateRecordStatus(record)
		if record.Status != nss.StatusLost {
			records = append(records, record)
		}
	}

	return records
}

func (rs *RecordStore) ListAll() []*nss.Record {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	var records []*nss.Record
	for _, record := range rs.records {
		rs.healthTracker.UpdateRecordStatus(record)
		records = append(records, record)
	}
	return records
}

func (rs *RecordStore) MarkLost(hostname string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if record, exists := rs.records[hostname]; exists && !record.IsStatic {
		record.Status = nss.StatusLost
	}
}

func (rs *RecordStore) Forget(hostname string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if record, exists := rs.records[hostname]; exists && !record.IsStatic {
		delete(rs.records, hostname)
	}
}

func (rs *RecordStore) GetAllRecords() []dnsserver.DNSRecord {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	var records []dnsserver.DNSRecord
	for _, record := range rs.records {
		rs.healthTracker.UpdateRecordStatus(record)
		if record.Status == nss.StatusLost {
			continue
		}

		services := make(map[string]dnsserver.ServiceInfo)
		for name, proto := range record.Services {
			services[name] = dnsserver.ServiceInfo{
				Protocol: proto,
			}
		}

		records = append(records, dnsserver.DNSRecord{
			Hostname:  record.Hostname,
			Addresses: record.Addresses,
			Services:  services,
			Status:    string(record.Status),
			LastSeen:  time.Unix(record.Timestamp, 0),
			IsStatic:  record.IsStatic,
		})
	}
	return records
}

func (rs *RecordStore) cleanupExpiredRecords() {
	interval := rs.healthConfig.CleanupInterval
	if interval == 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-rs.stopChan:
			return
		case <-ticker.C:
			rs.mu.Lock()
			for hostname, record := range rs.records {
				if rs.healthTracker.ShouldExpire(record) {
					delete(rs.records, hostname)
				} else {
					rs.healthTracker.UpdateRecordStatus(record)
				}
			}
			rs.mu.Unlock()
		}
	}
}
