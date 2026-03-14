package timesync

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/offline-lab/disco/internal/config"
	"github.com/offline-lab/disco/internal/discovery"
)

func TestTimeSyncService_ForceUpdate_NoSources(t *testing.T) {
	cfg := &config.TimeSyncConfig{
		Enabled:         true,
		MinSources:      2,
		MaxSourceSpread: 100 * time.Millisecond,
		MaxStaleAge:     30 * time.Second,
	}

	svc := NewTimeSyncService(cfg, nil)
	defer svc.Stop()

	result := svc.ForceUpdate(false)

	if result.Success {
		t.Error("Expected failure with no sources")
	}
	if result.Error == "" {
		t.Error("Expected error message")
	}
}

func TestTimeSyncService_ForceUpdate_InsufficientSources(t *testing.T) {
	cfg := &config.TimeSyncConfig{
		Enabled:         true,
		MinSources:      2,
		MaxSourceSpread: 100 * time.Millisecond,
		MaxStaleAge:     30 * time.Second,
	}

	svc := NewTimeSyncService(cfg, nil)
	defer svc.Stop()

	now := time.Now().UnixNano()
	msg := &discovery.TimeAnnounceMessage{
		Type:      discovery.MessageTimeAnnounce,
		Timestamp: now,
		SourceID:  "gps-1",
		ClockInfo: discovery.ClockInfo{
			Stratum:        1,
			RootDispersion: 0.0001,
		},
	}

	svc.ProcessMessage(msg)

	result := svc.ForceUpdate(false)

	if result.Success {
		t.Error("Expected failure with only 1 source (need 2)")
	}
}

func TestTimeSyncService_ForceUpdate_Success(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Clock adjustment only supported on Linux")
	}

	cfg := &config.TimeSyncConfig{
		Enabled:         true,
		MinSources:      2,
		MaxSourceSpread: 100 * time.Millisecond,
		MaxStaleAge:     30 * time.Second,
	}

	svc := NewTimeSyncService(cfg, nil)
	defer svc.Stop()

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

	svc.ProcessMessage(msg1)
	svc.ProcessMessage(msg2)

	result := svc.ForceUpdate(false)

	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}
	if result.SourceCount != 2 {
		t.Errorf("Expected 2 sources, got %d", result.SourceCount)
	}
	if result.Method == "" {
		t.Error("Expected method to be set")
	}
}

func TestTimeSyncService_ForceUpdate_AllowBackward(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Clock adjustment only supported on Linux")
	}

	cfg := &config.TimeSyncConfig{
		Enabled:         true,
		MinSources:      2,
		MaxSourceSpread: 100 * time.Millisecond,
		MaxStaleAge:     30 * time.Second,
	}

	svc := NewTimeSyncService(cfg, nil)
	defer svc.Stop()

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

	svc.ProcessMessage(msg1)
	svc.ProcessMessage(msg2)

	result := svc.ForceUpdate(true)

	if !result.Success {
		t.Errorf("Expected success with allowBackward=true, got: %s", result.Error)
	}
}

func TestTimeSyncService_ForceUpdate_UpdatesStatus(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Clock adjustment only supported on Linux")
	}

	cfg := &config.TimeSyncConfig{
		Enabled:         true,
		MinSources:      2,
		MaxSourceSpread: 100 * time.Millisecond,
		MaxStaleAge:     30 * time.Second,
	}

	svc := NewTimeSyncService(cfg, nil)
	defer svc.Stop()

	status := svc.GetStatus()
	if status.Synced {
		t.Error("Expected not synced initially")
	}

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

	svc.ProcessMessage(msg1)
	svc.ProcessMessage(msg2)

	result := svc.ForceUpdate(false)
	if !result.Success {
		t.Fatalf("ForceUpdate failed: %s", result.Error)
	}

	status = svc.GetStatus()
	if !status.Synced {
		t.Error("Expected synced after force update")
	}
	if status.SourceCount != 2 {
		t.Errorf("Expected 2 sources in status, got %d", status.SourceCount)
	}
}

func TestTimeSyncService_ForceUpdate_Concurrent(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("Skipping - requires CAP_SYS_TIME capability (not available in CI)")
	}
}

func TestTimeSyncService_ForceUpdate_MinSources1(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Clock adjustment only supported on Linux")
	}

	cfg := &config.TimeSyncConfig{
		Enabled:         true,
		MinSources:      1,
		MaxSourceSpread: 100 * time.Millisecond,
		MaxStaleAge:     30 * time.Second,
	}

	svc := NewTimeSyncService(cfg, nil)
	defer svc.Stop()

	now := time.Now().UnixNano()
	msg := &discovery.TimeAnnounceMessage{
		Type:      discovery.MessageTimeAnnounce,
		Timestamp: now,
		SourceID:  "gps-1",
		ClockInfo: discovery.ClockInfo{
			Stratum:        1,
			RootDispersion: 0.0001,
		},
	}

	svc.ProcessMessage(msg)

	result := svc.ForceUpdate(false)

	if !result.Success {
		t.Errorf("Expected success with min_sources=1, got: %s", result.Error)
	}
	if result.SourceCount != 1 {
		t.Errorf("Expected 1 source, got %d", result.SourceCount)
	}
}

func TestTimeSyncService_ForceUpdate_NonLinux_PlatformError(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("This test is for non-Linux platforms")
	}

	cfg := &config.TimeSyncConfig{
		Enabled:         true,
		MinSources:      2,
		MaxSourceSpread: 100 * time.Millisecond,
		MaxStaleAge:     30 * time.Second,
	}

	svc := NewTimeSyncService(cfg, nil)
	defer svc.Stop()

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

	svc.ProcessMessage(msg1)
	svc.ProcessMessage(msg2)

	result := svc.ForceUpdate(false)

	// On non-Linux, we expect the platform error
	if result.Success {
		t.Error("Expected failure on non-Linux platform")
	}
	if result.Error != "clock adjustment not supported on this platform" {
		t.Errorf("Expected platform error, got: %s", result.Error)
	}
}
