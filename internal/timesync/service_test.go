package timesync

import (
	"testing"
	"time"

	"github.com/offline-lab/disco/internal/config"
	"github.com/offline-lab/disco/internal/discovery"
	"github.com/offline-lab/disco/internal/security"
)

func TestTimeSyncService_ProcessMessage(t *testing.T) {
	cfg := &config.TimeSyncConfig{
		Enabled:         true,
		MinSources:      2,
		MaxSourceSpread: 100 * time.Millisecond,
		MaxStaleAge:     30 * time.Second,
	}

	svc := NewTimeSyncService(cfg, nil)
	defer svc.Stop()

	msg := &discovery.TimeAnnounceMessage{
		Type:      discovery.MessageTimeAnnounce,
		Timestamp: time.Now().UnixNano(),
		SourceID:  "gps-1",
		ClockInfo: discovery.ClockInfo{
			Stratum:        1,
			RootDispersion: 0.0001,
		},
	}

	svc.ProcessMessage(msg)

	if svc.store.Count() != 1 {
		t.Errorf("expected 1 source in store, got %d", svc.store.Count())
	}
}

func TestTimeSyncService_Status(t *testing.T) {
	cfg := &config.TimeSyncConfig{
		Enabled:         true,
		MinSources:      2,
		MaxSourceSpread: 100 * time.Millisecond,
	}

	svc := NewTimeSyncService(cfg, nil)

	status := svc.GetStatus()
	if status.Synced {
		t.Error("expected not synced initially")
	}
}

func TestTimeSyncService_RejectUnsigned(t *testing.T) {
	cfg := &config.TimeSyncConfig{
		Enabled:       true,
		MinSources:    2,
		RequireSigned: true,
	}

	km := &security.KeyManager{}
	svc := NewTimeSyncService(cfg, km)
	defer svc.Stop()

	msg := &discovery.TimeAnnounceMessage{
		Type:      discovery.MessageTimeAnnounce,
		Timestamp: time.Now().UnixNano(),
		SourceID:  "gps-1",
		ClockInfo: discovery.ClockInfo{Stratum: 1},
	}

	svc.ProcessMessage(msg)

	if svc.store.Count() != 0 {
		t.Error("expected unsigned message to be rejected when keyManager is set")
	}
}

func TestTimeSyncService_AcceptUnsignedWhenNoSecurity(t *testing.T) {
	cfg := &config.TimeSyncConfig{
		Enabled:       true,
		MinSources:    2,
		RequireSigned: true,
	}

	svc := NewTimeSyncService(cfg, nil)
	defer svc.Stop()

	msg := &discovery.TimeAnnounceMessage{
		Type:      discovery.MessageTimeAnnounce,
		Timestamp: time.Now().UnixNano(),
		SourceID:  "gps-1",
		ClockInfo: discovery.ClockInfo{Stratum: 1},
	}

	svc.ProcessMessage(msg)

	if svc.store.Count() != 1 {
		t.Error("expected unsigned message to be accepted when no security configured")
	}
}
