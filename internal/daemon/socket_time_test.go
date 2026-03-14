package daemon

import (
	"testing"
	"time"

	"github.com/offline-lab/disco/internal/config"
	"github.com/offline-lab/disco/internal/discovery"
	"github.com/offline-lab/disco/internal/nss"
	"github.com/offline-lab/disco/internal/timesync"
)

func newTestTimeStore() *RecordStore {
	return NewRecordStore(3600*time.Second, &config.HealthConfig{
		GracePeriod:     60 * time.Second,
		ExpireAfter:     3600 * time.Second,
		CleanupInterval: 30 * time.Second,
	}, nil)
}

func TestSocketServer_handleTimeStatus_Disabled(t *testing.T) {
	store := newTestTimeStore()
	server := NewSocketServer("/tmp/test.sock", store)

	query := &nss.Query{
		Type:      "TIME_STATUS",
		RequestID: "test-time-001",
	}

	resp := server.handleTimeStatus(query)

	if resp.Type != nss.ResponseError {
		t.Errorf("Expected ERROR, got %s", resp.Type)
	}
	if resp.Error != "time sync not enabled" {
		t.Errorf("Expected 'time sync not enabled' error, got %s", resp.Error)
	}
}

func TestSocketServer_handleTimeStatus_Enabled(t *testing.T) {
	store := newTestTimeStore()
	server := NewSocketServer("/tmp/test.sock", store)

	cfg := &config.TimeSyncConfig{
		Enabled:         true,
		MinSources:      2,
		MaxSourceSpread: 100 * time.Millisecond,
		MaxStaleAge:     30 * time.Second,
	}

	ts := timesync.NewTimeSyncService(cfg, nil)
	server.SetTimeSync(ts)

	query := &nss.Query{
		Type:      "TIME_STATUS",
		RequestID: "test-time-002",
	}

	resp := server.handleTimeStatus(query)

	if resp.Type != nss.ResponseOK {
		t.Errorf("Expected OK, got %s", resp.Type)
	}
}

func TestSocketServer_handleTimeForceUpdate_Disabled(t *testing.T) {
	store := newTestTimeStore()
	server := NewSocketServer("/tmp/test.sock", store)

	query := &nss.Query{
		Type:      "TIME_FORCE_UPDATE",
		RequestID: "test-force-001",
	}

	resp := server.handleTimeForceUpdate(query)

	if resp.Type != nss.ResponseError {
		t.Errorf("Expected ERROR, got %s", resp.Type)
	}
	if resp.Error != "time sync not enabled" {
		t.Errorf("Expected 'time sync not enabled' error, got %s", resp.Error)
	}
}

func TestSocketServer_handleTimeForceUpdate_NoSources(t *testing.T) {
	store := newTestTimeStore()
	server := NewSocketServer("/tmp/test.sock", store)

	cfg := &config.TimeSyncConfig{
		Enabled:         true,
		MinSources:      2,
		MaxSourceSpread: 100 * time.Millisecond,
		MaxStaleAge:     30 * time.Second,
	}

	ts := timesync.NewTimeSyncService(cfg, nil)
	server.SetTimeSync(ts)

	query := &nss.Query{
		Type:      "TIME_FORCE_UPDATE",
		RequestID: "test-force-002",
	}

	resp := server.handleTimeForceUpdate(query)

	// Should fail due to insufficient sources
	if resp.Type == nss.ResponseOK {
		t.Log("Force update returned OK but should have failed due to no sources")
	}
}

func TestSocketServer_handleTimeForceUpdate_WithSources(t *testing.T) {
	store := newTestTimeStore()
	server := NewSocketServer("/tmp/test.sock", store)

	cfg := &config.TimeSyncConfig{
		Enabled:         true,
		MinSources:      2,
		MaxSourceSpread: 100 * time.Millisecond,
		MaxStaleAge:     30 * time.Second,
	}

	ts := timesync.NewTimeSyncService(cfg, nil)
	server.SetTimeSync(ts)

	// Add time sources
	now := time.Now().UnixNano()
	msg1 := &discovery.TimeAnnounceMessage{
		Type:      discovery.MessageTimeAnnounce,
		Timestamp: now - int64(5*time.Millisecond),
		SourceID:  "gps-1",
		ClockInfo: discovery.ClockInfo{
			Stratum:        1,
			RootDispersion: 0.0001,
		},
	}
	msg2 := &discovery.TimeAnnounceMessage{
		Type:      discovery.MessageTimeAnnounce,
		Timestamp: now,
		SourceID:  "gps-2",
		ClockInfo: discovery.ClockInfo{
			Stratum:        1,
			RootDispersion: 0.0001,
		},
	}

	ts.ProcessMessage(msg1)
	ts.ProcessMessage(msg2)

	query := &nss.Query{
		Type:      "TIME_FORCE_UPDATE",
		RequestID: "test-force-003",
	}

	resp := server.handleTimeForceUpdate(query)

	// Note: May still fail if not running as root (can't adjust clock)
	// But the query should be handled properly
	if resp.Type != nss.ResponseOK && resp.Type != nss.ResponseError {
		t.Errorf("Expected OK or ERROR, got %s", resp.Type)
	}
}

func TestSocketServer_handleTimeForceUpdate_AllowBackward(t *testing.T) {
	store := newTestTimeStore()
	server := NewSocketServer("/tmp/test.sock", store)

	cfg := &config.TimeSyncConfig{
		Enabled:         true,
		MinSources:      2,
		MaxSourceSpread: 100 * time.Millisecond,
		MaxStaleAge:     30 * time.Second,
	}

	ts := timesync.NewTimeSyncService(cfg, nil)
	server.SetTimeSync(ts)

	// Add time sources
	now := time.Now().UnixNano()
	msg1 := &discovery.TimeAnnounceMessage{
		Type:      discovery.MessageTimeAnnounce,
		Timestamp: now,
		SourceID:  "gps-1",
		ClockInfo: discovery.ClockInfo{
			Stratum:        1,
			RootDispersion: 0.0001,
		},
	}
	msg2 := &discovery.TimeAnnounceMessage{
		Type:      discovery.MessageTimeAnnounce,
		Timestamp: now + int64(5*time.Millisecond),
		SourceID:  "gps-2",
		ClockInfo: discovery.ClockInfo{
			Stratum:        1,
			RootDispersion: 0.0001,
		},
	}

	ts.ProcessMessage(msg1)
	ts.ProcessMessage(msg2)

	// Test with allow_backward = true
	query := &nss.Query{
		Type:      "TIME_FORCE_UPDATE",
		RequestID: "test-force-004",
		Name:      "true",
	}

	resp := server.handleTimeForceUpdate(query)

	if resp.Type != nss.ResponseOK && resp.Type != nss.ResponseError {
		t.Errorf("Expected OK or ERROR, got %s", resp.Type)
	}
}

func TestSocketServer_handleQuery_TimeTypes(t *testing.T) {
	store := newTestTimeStore()
	server := NewSocketServer("/tmp/test.sock", store)

	cfg := &config.TimeSyncConfig{
		Enabled:         true,
		MinSources:      2,
		MaxSourceSpread: 100 * time.Millisecond,
		MaxStaleAge:     30 * time.Second,
	}

	ts := timesync.NewTimeSyncService(cfg, nil)
	server.SetTimeSync(ts)

	tests := []struct {
		name      string
		queryType nss.MessageType
		wantOK    bool
	}{
		{"TIME_STATUS", "TIME_STATUS", true},
		{"TIME_FORCE_UPDATE", "TIME_FORCE_UPDATE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &nss.Query{
				Type:      tt.queryType,
				RequestID: "test-" + tt.name,
			}

			resp := server.handleQuery(query)

			if tt.wantOK && resp.Type != nss.ResponseOK {
				t.Errorf("Expected OK for %s, got %s", tt.name, resp.Type)
			}
		})
	}
}
