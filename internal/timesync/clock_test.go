package timesync

import (
	"testing"
	"time"
)

func TestCalculateAdjustment_StepThreshold(t *testing.T) {
	cfg := &ClockConfig{
		StepThreshold:     128 * time.Millisecond,
		SlewThreshold:     500 * time.Microsecond,
		AllowStepBackward: false,
	}

	adj := CalculateAdjustment(0.200, cfg)
	if adj.Method != AdjustmentStep {
		t.Errorf("expected step adjustment, got %v", adj.Method)
	}
}

func TestCalculateAdjustment_SlewThreshold(t *testing.T) {
	cfg := &ClockConfig{
		StepThreshold:     128 * time.Millisecond,
		SlewThreshold:     500 * time.Microsecond,
		AllowStepBackward: false,
	}

	adj := CalculateAdjustment(0.050, cfg)
	if adj.Method != AdjustmentSlew {
		t.Errorf("expected slew adjustment, got %v", adj.Method)
	}
}

func TestCalculateAdjustment_NoAdjustment(t *testing.T) {
	cfg := &ClockConfig{
		StepThreshold:     128 * time.Millisecond,
		SlewThreshold:     500 * time.Microsecond,
		AllowStepBackward: false,
	}

	adj := CalculateAdjustment(0.0001, cfg)
	if adj.Method != AdjustmentNone {
		t.Errorf("expected no adjustment, got %v", adj.Method)
	}
}

func TestCalculateAdjustment_BackwardNotAllowed(t *testing.T) {
	cfg := &ClockConfig{
		StepThreshold:     128 * time.Millisecond,
		SlewThreshold:     500 * time.Microsecond,
		AllowStepBackward: false,
	}

	adj := CalculateAdjustment(-0.200, cfg)
	if adj.Method != AdjustmentNone {
		t.Errorf("expected no adjustment for backward step, got %v", adj.Method)
	}
}

func TestCalculateAdjustment_BackwardAllowed(t *testing.T) {
	cfg := &ClockConfig{
		StepThreshold:     128 * time.Millisecond,
		SlewThreshold:     500 * time.Microsecond,
		AllowStepBackward: true,
	}

	adj := CalculateAdjustment(-0.200, cfg)
	if adj.Method != AdjustmentStep {
		t.Errorf("expected step adjustment for backward when allowed, got %v", adj.Method)
	}
}

func TestCalculateAdjustment_BackwardSlewNotAllowed(t *testing.T) {
	cfg := &ClockConfig{
		StepThreshold:     128 * time.Millisecond,
		SlewThreshold:     500 * time.Microsecond,
		AllowStepBackward: false,
	}

	adj := CalculateAdjustment(-0.050, cfg)
	if adj.Method != AdjustmentNone {
		t.Errorf("expected no adjustment for backward slew when not allowed, got %v", adj.Method)
	}
}
