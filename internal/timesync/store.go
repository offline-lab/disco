package timesync

import (
	"sync"
	"time"

	"github.com/offline-lab/disco/internal/discovery"
)

type TimeSource struct {
	SourceID      string
	Timestamp     int64
	ReceivedAt    time.Time
	ClockInfo     discovery.ClockInfo
	LeapIndicator int
}

type TimeSourceStore struct {
	sources  map[string]*TimeSource
	maxStale time.Duration
	mu       sync.RWMutex
}

func NewTimeSourceStore(maxStale time.Duration) *TimeSourceStore {
	return &TimeSourceStore{
		sources:  make(map[string]*TimeSource),
		maxStale: maxStale,
	}
}

func (s *TimeSourceStore) Add(msg *discovery.TimeAnnounceMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	src, exists := s.sources[msg.SourceID]
	if !exists {
		src = &TimeSource{}
		s.sources[msg.SourceID] = src
	}

	src.SourceID = msg.SourceID
	src.Timestamp = msg.Timestamp
	src.ReceivedAt = time.Now()
	src.ClockInfo = msg.ClockInfo
	src.LeapIndicator = msg.LeapIndicator
}

func (s *TimeSourceStore) GetValidSources() []*TimeSource {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	valid := make([]*TimeSource, 0, len(s.sources))

	for _, src := range s.sources {
		if now.Sub(src.ReceivedAt) <= s.maxStale {
			valid = append(valid, src)
		}
	}

	return valid
}

func (s *TimeSourceStore) HasMinimumSources(min int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	count := 0

	for _, src := range s.sources {
		if now.Sub(src.ReceivedAt) <= s.maxStale {
			count++
			if count >= min {
				return true
			}
		}
	}

	return false
}

func (s *TimeSourceStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sources)
}

func (s *TimeSourceStore) Remove(sourceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sources, sourceID)
}

func (s *TimeSourceStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sources = make(map[string]*TimeSource)
}
