package daemon

import (
	"sync"
	"time"

	"github.com/offline-lab/disco/internal/config"
	"github.com/offline-lab/disco/internal/nss"
)

type HealthTracker struct {
	config      *config.HealthConfig
	staticHosts map[string]*nss.Record
	mu          sync.RWMutex
}

func NewHealthTracker(cfg *config.HealthConfig, staticHosts map[string]config.StaticHost) *HealthTracker {
	ht := &HealthTracker{
		config:      cfg,
		staticHosts: make(map[string]*nss.Record),
	}

	for hostname, sh := range staticHosts {
		record := &nss.Record{
			Hostname:  hostname,
			Addresses: sh.Addresses,
			Services:  make(map[string]string),
			IsStatic:  true,
			Status:    nss.StatusStatic,
		}
		for _, svc := range sh.Services {
			proto := svc.Protocol
			if proto == "" {
				proto = "tcp"
			}
			record.Services[svc.Name] = proto
		}
		ht.staticHosts[hostname] = record
	}

	return ht
}

func (ht *HealthTracker) ComputeStatus(record *nss.Record) nss.HostStatus {
	if record.IsStatic {
		return nss.StatusStatic
	}

	age := time.Since(time.Unix(record.Timestamp, 0))

	switch {
	case age < ht.config.GracePeriod:
		return nss.StatusHealthy
	case age < ht.config.ExpireAfter:
		return nss.StatusStale
	default:
		return nss.StatusLost
	}
}

func (ht *HealthTracker) UpdateRecordStatus(record *nss.Record) {
	record.Status = ht.ComputeStatus(record)
}

func (ht *HealthTracker) GetStaticHosts() map[string]*nss.Record {
	ht.mu.RLock()
	defer ht.mu.RUnlock()

	result := make(map[string]*nss.Record)
	for k, v := range ht.staticHosts {
		result[k] = v
	}
	return result
}

func (ht *HealthTracker) IsStatic(hostname string) bool {
	ht.mu.RLock()
	defer ht.mu.RUnlock()
	_, exists := ht.staticHosts[hostname]
	return exists
}

func (ht *HealthTracker) ShouldExpire(record *nss.Record) bool {
	if record.IsStatic {
		return false
	}
	age := time.Since(time.Unix(record.Timestamp, 0))
	return age >= ht.config.ExpireAfter
}
