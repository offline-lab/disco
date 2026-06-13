package daemon

import (
	"sync"
	"time"

	"github.com/offline-lab/disco/internal/discovery"
	"github.com/offline-lab/disco/internal/nss"
)

type LocationStore struct {
	mu      sync.RWMutex
	records map[string]*nss.LocationHealth
}

func NewLocationStore() *LocationStore {
	return &LocationStore{
		records: make(map[string]*nss.LocationHealth),
	}
}

func (ls *LocationStore) AddOrUpdate(msg *discovery.LocationAnnounceMessage) {
	ts := msg.Timestamp
	if ts == 0 {
		ts = time.Now().Unix()
	}

	ls.mu.Lock()
	defer ls.mu.Unlock()

	ls.records[msg.SourceID] = &nss.LocationHealth{
		SourceID:   msg.SourceID,
		Latitude:   msg.Location.Latitude,
		Longitude:  msg.Location.Longitude,
		Altitude:   msg.Location.Altitude,
		Satellites: msg.Location.Satellites,
		Fix:        msg.Location.Fix,
		LastSeen:   ts,
	}
}

func (ls *LocationStore) Get(sourceID string) (*nss.LocationHealth, bool) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	record, ok := ls.records[sourceID]
	if !ok {
		return nil, false
	}
	copy := *record
	copy.LastSeenAgo = formatDuration(time.Since(time.Unix(record.LastSeen, 0)))
	return &copy, true
}

func (ls *LocationStore) List() []*nss.LocationHealth {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	records := make([]*nss.LocationHealth, 0, len(ls.records))
	for _, r := range ls.records {
		entry := *r
		entry.LastSeenAgo = formatDuration(time.Since(time.Unix(r.LastSeen, 0)))
		records = append(records, &entry)
	}
	return records
}
