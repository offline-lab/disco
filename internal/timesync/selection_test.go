package timesync

import (
	"testing"
	"time"

	"github.com/offline-lab/disco/internal/discovery"
)

func TestSelectBestTime_SingleSource(t *testing.T) {
	sources := []*TimeSource{
		{
			SourceID:  "gps-1",
			Timestamp: 1708123456000000000,
			ClockInfo: discovery.ClockInfo{Stratum: 1, RootDispersion: 0.0001},
		},
	}

	result, err := SelectBestTime(sources, 1, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Offset == 0 {
		t.Error("expected non-zero offset calculation")
	}
}

func TestSelectBestTime_MultipleSourcesAgreeing(t *testing.T) {
	now := time.Now().UnixNano() - int64(10*time.Millisecond)

	sources := []*TimeSource{
		{
			SourceID:  "gps-1",
			Timestamp: now,
			ClockInfo: discovery.ClockInfo{Stratum: 1, RootDispersion: 0.0001},
		},
		{
			SourceID:  "gps-2",
			Timestamp: now + int64(5*time.Millisecond),
			ClockInfo: discovery.ClockInfo{Stratum: 1, RootDispersion: 0.0001},
		},
	}

	result, err := SelectBestTime(sources, 2, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.SourceCount != 2 {
		t.Errorf("expected 2 sources, got %d", result.SourceCount)
	}
}

func TestSelectBestTime_InsufficientSources(t *testing.T) {
	sources := []*TimeSource{
		{SourceID: "gps-1", Timestamp: time.Now().UnixNano()},
	}

	_, err := SelectBestTime(sources, 2, 100*time.Millisecond)
	if err == nil {
		t.Error("expected error for insufficient sources")
	}
}

func TestSelectBestTime_SourcesDisagree(t *testing.T) {
	now := time.Now().UnixNano()

	sources := []*TimeSource{
		{
			SourceID:  "gps-1",
			Timestamp: now,
			ClockInfo: discovery.ClockInfo{Stratum: 1, RootDispersion: 0.0001},
		},
		{
			SourceID:  "gps-2",
			Timestamp: now + int64(500*time.Millisecond),
			ClockInfo: discovery.ClockInfo{Stratum: 1, RootDispersion: 0.0001},
		},
	}

	_, err := SelectBestTime(sources, 2, 100*time.Millisecond)
	if err == nil {
		t.Error("expected error for disagreeing sources")
	}
}

func TestSelectBestTime_WeightedByDispersion(t *testing.T) {
	now := time.Now().UnixNano()

	sources := []*TimeSource{
		{
			SourceID:  "gps-1",
			Timestamp: now,
			ClockInfo: discovery.ClockInfo{Stratum: 1, RootDispersion: 0.0001},
		},
		{
			SourceID:  "gps-2",
			Timestamp: now + int64(10*time.Millisecond),
			ClockInfo: discovery.ClockInfo{Stratum: 1, RootDispersion: 0.01},
		},
	}

	result, err := SelectBestTime(sources, 2, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.SourceCount != 2 {
		t.Errorf("expected 2 sources, got %d", result.SourceCount)
	}
}

func TestSelectBestTime_HighStratumFiltered(t *testing.T) {
	now := time.Now().UnixNano()

	sources := []*TimeSource{
		{
			SourceID:  "gps-1",
			Timestamp: now,
			ClockInfo: discovery.ClockInfo{Stratum: 16},
		},
		{
			SourceID:  "gps-2",
			Timestamp: now + int64(5*time.Millisecond),
			ClockInfo: discovery.ClockInfo{Stratum: 1, RootDispersion: 0.0001},
		},
	}

	_, err := SelectBestTime(sources, 2, 100*time.Millisecond)
	if err == nil {
		t.Error("expected error when high stratum source is filtered")
	}
}

func TestSelectBestTime_HighDispersionFiltered(t *testing.T) {
	now := time.Now().UnixNano()

	sources := []*TimeSource{
		{
			SourceID:  "gps-1",
			Timestamp: now,
			ClockInfo: discovery.ClockInfo{Stratum: 1, RootDispersion: 2.0},
		},
		{
			SourceID:  "gps-2",
			Timestamp: now + int64(5*time.Millisecond),
			ClockInfo: discovery.ClockInfo{Stratum: 1, RootDispersion: 0.0001},
		},
	}

	_, err := SelectBestTime(sources, 2, 100*time.Millisecond)
	if err == nil {
		t.Error("expected error when high dispersion source is filtered")
	}
}
