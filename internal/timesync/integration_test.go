//go:build integration

package timesync

import (
	"testing"
	"time"

	"github.com/offline-lab/disco/internal/config"
	"github.com/offline-lab/disco/internal/discovery"
)

func TestIntegration_FullSyncFlow(t *testing.T) {
	cfg := &config.TimeSyncConfig{
		Enabled:           true,
		MinSources:        2,
		MaxSourceSpread:   100 * time.Millisecond,
		MaxStaleAge:       30 * time.Second,
		MaxDispersion:     1.0,
		StepThreshold:     128 * time.Millisecond,
		SlewThreshold:     500 * time.Microsecond,
		PollInterval:      1 * time.Second,
		RequireSigned:     false,
		AllowStepBackward: false,
	}

	svc := NewTimeSyncService(cfg, nil)
	defer svc.Stop()

	now := time.Now().UnixNano()

	msg1 := &discovery.TimeAnnounceMessage{
		Type:      discovery.MessageTimeAnnounce,
		Timestamp: now - int64(10*time.Millisecond),
		SourceID:  "gps-1",
		ClockInfo: discovery.ClockInfo{
			Stratum:        1,
			Precision:      -20,
			RootDelay:      0.0,
			RootDispersion: 0.0001,
			ReferenceID:    "GPS",
		},
	}

	msg2 := &discovery.TimeAnnounceMessage{
		Type:      discovery.MessageTimeAnnounce,
		Timestamp: now,
		SourceID:  "gps-2",
		ClockInfo: discovery.ClockInfo{
			Stratum:        1,
			Precision:      -20,
			RootDelay:      0.0,
			RootDispersion: 0.0001,
			ReferenceID:    "GPS",
		},
	}

	svc.ProcessMessage(msg1)
	svc.ProcessMessage(msg2)

	svc.evaluateAndAdjust()

	status := svc.GetStatus()
	if !status.Synced {
		t.Errorf("expected synced, got: %s", status.LastError)
	}
	if status.SourceCount != 2 {
		t.Errorf("expected 2 sources, got %d", status.SourceCount)
	}
}

func TestIntegration_SourceDisagreement(t *testing.T) {
	cfg := &config.TimeSyncConfig{
		Enabled:         true,
		MinSources:      2,
		MaxSourceSpread: 50 * time.Millisecond,
		MaxStaleAge:     30 * time.Second,
	}

	svc := NewTimeSyncService(cfg, nil)
	defer svc.Stop()

	now := time.Now().UnixNano()

	msg1 := &discovery.TimeAnnounceMessage{
		Type:      discovery.MessageTimeAnnounce,
		Timestamp: now,
		SourceID:  "gps-1",
		ClockInfo: discovery.ClockInfo{Stratum: 1, RootDispersion: 0.0001},
	}

	msg2 := &discovery.TimeAnnounceMessage{
		Type:      discovery.MessageTimeAnnounce,
		Timestamp: now + int64(500*time.Millisecond),
		SourceID:  "gps-2",
		ClockInfo: discovery.ClockInfo{Stratum: 1, RootDispersion: 0.0001},
	}

	svc.ProcessMessage(msg1)
	svc.ProcessMessage(msg2)

	svc.evaluateAndAdjust()

	status := svc.GetStatus()
	if status.Synced {
		t.Error("expected not synced due to disagreement")
	}
}
