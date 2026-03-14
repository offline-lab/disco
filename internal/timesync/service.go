package timesync

import (
	"sync"
	"time"

	"github.com/offline-lab/disco/internal/config"
	"github.com/offline-lab/disco/internal/discovery"
	"github.com/offline-lab/disco/internal/logging"
	"github.com/offline-lab/disco/internal/security"
)

type SyncStatus struct {
	Synced       bool
	SourceCount  int
	LastOffset   float64
	LastSyncTime time.Time
	LastError    string
}

type ForceUpdateResult struct {
	Success     bool
	Offset      float64
	Method      string
	SourceCount int
	Error       string
}

type TimeSyncService struct {
	config     *config.TimeSyncConfig
	keyManager *security.KeyManager
	store      *TimeSourceStore
	status     SyncStatus
	mu         sync.RWMutex
	stopChan   chan struct{}
}

func NewTimeSyncService(cfg *config.TimeSyncConfig, km *security.KeyManager) *TimeSyncService {
	return &TimeSyncService{
		config:     cfg,
		keyManager: km,
		store:      NewTimeSourceStore(cfg.MaxStaleAge),
		stopChan:   make(chan struct{}),
	}
}

func (s *TimeSyncService) ProcessMessage(msg *discovery.TimeAnnounceMessage) {
	if !s.config.Enabled {
		return
	}

	if s.config.RequireSigned && msg.Signature == nil && s.keyManager != nil {
		logging.Debug("rejecting unsigned time message", nil)
		return
	}

	s.store.Add(msg)
	logging.Debug("time message received", map[string]interface{}{
		"source_id": msg.SourceID,
		"stratum":   msg.ClockInfo.Stratum,
	})
}

func (s *TimeSyncService) Start() {
	ticker := time.NewTicker(s.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.evaluateAndAdjust()
		}
	}
}

func (s *TimeSyncService) evaluateAndAdjust() {
	sources := s.store.GetValidSources()

	update, err := SelectBestTime(sources, s.config.MinSources, s.config.MaxSourceSpread)
	if err != nil {
		s.mu.Lock()
		s.status.Synced = false
		s.status.LastError = err.Error()
		s.mu.Unlock()
		logging.Debug("time selection failed", map[string]interface{}{"error": err.Error()})
		return
	}

	clockCfg := &ClockConfig{
		StepThreshold:     s.config.StepThreshold,
		SlewThreshold:     s.config.SlewThreshold,
		AllowStepBackward: s.config.AllowStepBackward,
	}

	adj := CalculateAdjustment(update.Offset, clockCfg)

	s.mu.Lock()
	s.status.Synced = true
	s.status.SourceCount = update.SourceCount
	s.status.LastOffset = update.Offset
	s.status.LastSyncTime = time.Now()
	s.status.LastError = ""
	s.mu.Unlock()

	if adj.Method == AdjustmentNone {
		logging.Debug("clock within tolerance", map[string]interface{}{
			"offset": update.Offset,
		})
		return
	}

	if err := ApplyAdjustment(adj); err != nil {
		logging.Error("failed to adjust clock", err, nil)
		s.mu.Lock()
		s.status.LastError = err.Error()
		s.mu.Unlock()
		return
	}

	methodStr := "slewed"
	if adj.Method == AdjustmentStep {
		methodStr = "stepped"
	}
	logging.Info("clock adjusted", map[string]interface{}{
		"method":  methodStr,
		"offset":  update.Offset,
		"sources": update.SourceCount,
	})
}

func (s *TimeSyncService) GetStatus() SyncStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func (s *TimeSyncService) Stop() {
	close(s.stopChan)
}

func (s *TimeSyncService) ForceUpdate(allowBackward bool) *ForceUpdateResult {
	sources := s.store.GetValidSources()

	update, err := SelectBestTime(sources, s.config.MinSources, s.config.MaxSourceSpread)
	if err != nil {
		return &ForceUpdateResult{
			Success: false,
			Error:   err.Error(),
		}
	}

	clockCfg := &ClockConfig{
		StepThreshold:     time.Millisecond,
		SlewThreshold:     time.Microsecond,
		AllowStepBackward: allowBackward,
	}

	adj := CalculateAdjustment(update.Offset, clockCfg)

	s.mu.Lock()
	s.status.Synced = true
	s.status.SourceCount = update.SourceCount
	s.status.LastOffset = update.Offset
	s.status.LastSyncTime = time.Now()
	s.status.LastError = ""
	s.mu.Unlock()

	if adj.Method == AdjustmentNone {
		return &ForceUpdateResult{
			Success:     true,
			Offset:      update.Offset,
			Method:      "none",
			SourceCount: update.SourceCount,
		}
	}

	if err := ApplyAdjustment(adj); err != nil {
		s.mu.Lock()
		s.status.LastError = err.Error()
		s.mu.Unlock()
		return &ForceUpdateResult{
			Success: false,
			Error:   err.Error(),
		}
	}

	methodStr := "slewed"
	if adj.Method == AdjustmentStep {
		methodStr = "stepped"
	}

	logging.Info("clock force-adjusted", map[string]interface{}{
		"method":  methodStr,
		"offset":  update.Offset,
		"sources": update.SourceCount,
	})

	return &ForceUpdateResult{
		Success:     true,
		Offset:      update.Offset,
		Method:      methodStr,
		SourceCount: update.SourceCount,
	}
}
