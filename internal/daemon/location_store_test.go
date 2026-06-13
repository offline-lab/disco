package daemon

import (
	"fmt"
	"testing"
	"time"

	"github.com/offline-lab/disco/internal/discovery"
)

func makeLocationMsg(sourceID string, lat, lon, alt float64, sats int, ts int64) *discovery.LocationAnnounceMessage {
	return &discovery.LocationAnnounceMessage{
		Type:      discovery.MessageLocationAnnounce,
		MessageID: "test-" + sourceID,
		Timestamp: ts,
		SourceID:  sourceID,
		Location: discovery.LocationInfo{
			Latitude:   lat,
			Longitude:  lon,
			Altitude:   alt,
			Fix:        true,
			Satellites: sats,
		},
	}
}

func TestLocationStore_New(t *testing.T) {
	ls := NewLocationStore()
	if ls == nil {
		t.Fatal("NewLocationStore() returned nil")
	}
}

func TestLocationStore_AddAndGet(t *testing.T) {
	ls := NewLocationStore()
	ls.AddOrUpdate(makeLocationMsg("gps-1", 52.370216, 4.895168, 3.5, 8, time.Now().Unix()))

	record, ok := ls.Get("gps-1")
	if !ok {
		t.Fatal("Get() returned false after AddOrUpdate()")
	}
	if record.SourceID != "gps-1" {
		t.Errorf("SourceID = %s, want gps-1", record.SourceID)
	}
	if record.Latitude != 52.370216 {
		t.Errorf("Latitude = %f, want 52.370216", record.Latitude)
	}
	if record.Longitude != 4.895168 {
		t.Errorf("Longitude = %f, want 4.895168", record.Longitude)
	}
	if record.Altitude != 3.5 {
		t.Errorf("Altitude = %f, want 3.5", record.Altitude)
	}
	if record.Satellites != 8 {
		t.Errorf("Satellites = %d, want 8", record.Satellites)
	}
	if !record.Fix {
		t.Error("Fix = false, want true")
	}
}

func TestLocationStore_GetNotFound(t *testing.T) {
	ls := NewLocationStore()

	_, ok := ls.Get("nonexistent")
	if ok {
		t.Error("Get() returned true for nonexistent source")
	}
}

func TestLocationStore_Update(t *testing.T) {
	ls := NewLocationStore()
	ts := time.Now().Unix()

	ls.AddOrUpdate(makeLocationMsg("gps-1", 52.370216, 4.895168, 3.5, 8, ts))
	ls.AddOrUpdate(makeLocationMsg("gps-1", 53.0, 5.0, 10.0, 12, ts+60))

	record, ok := ls.Get("gps-1")
	if !ok {
		t.Fatal("Get() returned false after update")
	}
	if record.Latitude != 53.0 {
		t.Errorf("Latitude after update = %f, want 53.0", record.Latitude)
	}
	if record.Satellites != 12 {
		t.Errorf("Satellites after update = %d, want 12", record.Satellites)
	}
}

func TestLocationStore_List(t *testing.T) {
	ls := NewLocationStore()
	ts := time.Now().Unix()

	ls.AddOrUpdate(makeLocationMsg("gps-1", 52.370216, 4.895168, 3.5, 8, ts))
	ls.AddOrUpdate(makeLocationMsg("gps-2", 51.5074, -0.1278, 11.0, 10, ts))

	records := ls.List()
	if len(records) != 2 {
		t.Fatalf("List() count = %d, want 2", len(records))
	}

	sources := make(map[string]bool)
	for _, record := range records {
		sources[record.SourceID] = true
	}
	if !sources["gps-1"] || !sources["gps-2"] {
		t.Errorf("Missing source IDs in List(); got %v", sources)
	}
}

func TestLocationStore_ListEmpty(t *testing.T) {
	ls := NewLocationStore()

	records := ls.List()
	if len(records) != 0 {
		t.Errorf("List() count = %d, want 0 for empty store", len(records))
	}
}

func TestLocationStore_ZeroTimestamp(t *testing.T) {
	ls := NewLocationStore()
	ls.AddOrUpdate(makeLocationMsg("gps-1", 52.0, 4.0, 0, 6, 0))

	record, ok := ls.Get("gps-1")
	if !ok {
		t.Fatal("Get() returned false")
	}
	if record.LastSeen == 0 {
		t.Error("LastSeen should not be zero when message timestamp is zero")
	}
}

func TestLocationStore_LastSeenAgo(t *testing.T) {
	ls := NewLocationStore()
	ls.AddOrUpdate(makeLocationMsg("gps-1", 52.0, 4.0, 0, 8, time.Now().Unix()))

	record, ok := ls.Get("gps-1")
	if !ok {
		t.Fatal("Get() returned false")
	}
	if record.LastSeenAgo == "" {
		t.Error("LastSeenAgo should not be empty")
	}
}

func TestLocationStore_ListLastSeenAgo(t *testing.T) {
	ls := NewLocationStore()
	ls.AddOrUpdate(makeLocationMsg("gps-1", 52.0, 4.0, 0, 8, time.Now().Unix()))

	records := ls.List()
	if len(records) != 1 {
		t.Fatalf("List() count = %d, want 1", len(records))
	}
	if records[0].LastSeenAgo == "" {
		t.Error("LastSeenAgo should not be empty in List() output")
	}
}

func TestLocationStore_GetReturnsCopy(t *testing.T) {
	ls := NewLocationStore()
	ls.AddOrUpdate(makeLocationMsg("gps-1", 52.0, 4.0, 0, 8, time.Now().Unix()))

	r1, _ := ls.Get("gps-1")
	r1.Latitude = 99.0

	r2, _ := ls.Get("gps-1")
	if r2.Latitude == 99.0 {
		t.Error("Get() returned a reference; mutations should not affect the store")
	}
}

func TestLocationStore_ConcurrentAccess(t *testing.T) {
	ls := NewLocationStore()
	ts := time.Now().Unix()
	done := make(chan bool)

	for index := 0; index < 10; index++ {
		go func(n int) {
			ls.AddOrUpdate(makeLocationMsg(fmt.Sprintf("gps-%d", n), float64(n), float64(n), 0, n, ts))
			done <- true
		}(index)
	}
	for range 10 {
		<-done
	}

	for index := 0; index < 10; index++ {
		go func(n int) {
			ls.Get(fmt.Sprintf("gps-%d", n))
			done <- true
		}(index)
	}
	for range 10 {
		<-done
	}

	if len(ls.List()) != 10 {
		t.Errorf("List() count = %d, want 10 after concurrent writes", len(ls.List()))
	}
}
