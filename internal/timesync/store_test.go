package timesync

import (
	"testing"
	"time"

	"github.com/offline-lab/disco/internal/discovery"
)

func TestTimeSourceStore_AddAndGet(t *testing.T) {
	store := NewTimeSourceStore(30 * time.Second)

	msg := &discovery.TimeAnnounceMessage{
		Type:      discovery.MessageTimeAnnounce,
		MessageID: "test-1",
		Timestamp: time.Now().UnixNano(),
		SourceID:  "gps-node-1",
		ClockInfo: discovery.ClockInfo{
			Stratum:        1,
			Precision:      -20,
			RootDelay:      0.0,
			RootDispersion: 0.0001,
			ReferenceID:    "GPS",
		},
	}

	store.Add(msg)

	sources := store.GetValidSources()
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}

	if sources[0].SourceID != "gps-node-1" {
		t.Errorf("expected source_id gps-node-1, got %s", sources[0].SourceID)
	}
}

func TestTimeSourceStore_MinimumSources(t *testing.T) {
	store := NewTimeSourceStore(30 * time.Second)

	if store.HasMinimumSources(2) {
		t.Error("expected no minimum sources")
	}

	msg1 := &discovery.TimeAnnounceMessage{
		Type:      discovery.MessageTimeAnnounce,
		Timestamp: time.Now().UnixNano(),
		SourceID:  "gps-node-1",
		ClockInfo: discovery.ClockInfo{Stratum: 1},
	}
	msg2 := &discovery.TimeAnnounceMessage{
		Type:      discovery.MessageTimeAnnounce,
		Timestamp: time.Now().UnixNano(),
		SourceID:  "gps-node-2",
		ClockInfo: discovery.ClockInfo{Stratum: 1},
	}

	store.Add(msg1)
	store.Add(msg2)

	if !store.HasMinimumSources(2) {
		t.Error("expected 2 minimum sources")
	}
}

func TestTimeSourceStore_ExpireStale(t *testing.T) {
	store := NewTimeSourceStore(100 * time.Millisecond)

	msg := &discovery.TimeAnnounceMessage{
		Type:      discovery.MessageTimeAnnounce,
		Timestamp: time.Now().UnixNano(),
		SourceID:  "gps-node-1",
		ClockInfo: discovery.ClockInfo{Stratum: 1},
	}

	store.Add(msg)

	if len(store.GetValidSources()) != 1 {
		t.Fatal("expected 1 source before expiry")
	}

	time.Sleep(150 * time.Millisecond)

	if len(store.GetValidSources()) != 0 {
		t.Error("expected 0 sources after expiry")
	}
}

func TestTimeSourceStore_UpdateExisting(t *testing.T) {
	store := NewTimeSourceStore(30 * time.Second)

	msg1 := &discovery.TimeAnnounceMessage{
		Type:      discovery.MessageTimeAnnounce,
		Timestamp: 1000000000,
		SourceID:  "gps-node-1",
		ClockInfo: discovery.ClockInfo{Stratum: 1},
	}
	store.Add(msg1)

	msg2 := &discovery.TimeAnnounceMessage{
		Type:      discovery.MessageTimeAnnounce,
		Timestamp: 2000000000,
		SourceID:  "gps-node-1",
		ClockInfo: discovery.ClockInfo{Stratum: 1},
	}
	store.Add(msg2)

	sources := store.GetValidSources()
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}

	if sources[0].Timestamp != 2000000000 {
		t.Errorf("expected updated timestamp, got %d", sources[0].Timestamp)
	}
}

func TestTimeSourceStore_Remove(t *testing.T) {
	store := NewTimeSourceStore(30 * time.Second)

	msg := &discovery.TimeAnnounceMessage{
		Type:      discovery.MessageTimeAnnounce,
		Timestamp: time.Now().UnixNano(),
		SourceID:  "gps-node-1",
		ClockInfo: discovery.ClockInfo{Stratum: 1},
	}

	store.Add(msg)
	if store.Count() != 1 {
		t.Fatal("expected 1 source after add")
	}

	store.Remove("gps-node-1")
	if store.Count() != 0 {
		t.Error("expected 0 sources after remove")
	}
}

func TestTimeSourceStore_Clear(t *testing.T) {
	store := NewTimeSourceStore(30 * time.Second)

	msg1 := &discovery.TimeAnnounceMessage{
		Type:      discovery.MessageTimeAnnounce,
		Timestamp: time.Now().UnixNano(),
		SourceID:  "gps-node-1",
		ClockInfo: discovery.ClockInfo{Stratum: 1},
	}
	msg2 := &discovery.TimeAnnounceMessage{
		Type:      discovery.MessageTimeAnnounce,
		Timestamp: time.Now().UnixNano(),
		SourceID:  "gps-node-2",
		ClockInfo: discovery.ClockInfo{Stratum: 1},
	}

	store.Add(msg1)
	store.Add(msg2)
	if store.Count() != 2 {
		t.Fatal("expected 2 sources after add")
	}

	store.Clear()
	if store.Count() != 0 {
		t.Error("expected 0 sources after clear")
	}
}
