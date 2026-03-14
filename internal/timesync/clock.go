package timesync

import (
	"time"
)

type AdjustmentMethod int

const (
	AdjustmentNone AdjustmentMethod = iota
	AdjustmentSlew
	AdjustmentStep
)

type ClockConfig struct {
	StepThreshold     time.Duration
	SlewThreshold     time.Duration
	AllowStepBackward bool
}

type ClockAdjustment struct {
	Method   AdjustmentMethod
	Offset   time.Duration
	Absolute time.Time
}

func CalculateAdjustment(offsetSeconds float64, cfg *ClockConfig) *ClockAdjustment {
	offset := time.Duration(offsetSeconds * float64(time.Second))
	absOffset := offset
	if absOffset < 0 {
		absOffset = -absOffset
	}

	if absOffset < cfg.SlewThreshold {
		return &ClockAdjustment{Method: AdjustmentNone, Offset: offset}
	}

	if offset < 0 && !cfg.AllowStepBackward {
		return &ClockAdjustment{Method: AdjustmentNone, Offset: offset}
	}

	if absOffset >= cfg.StepThreshold {
		return &ClockAdjustment{
			Method:   AdjustmentStep,
			Offset:   offset,
			Absolute: time.Now().Add(offset),
		}
	}

	return &ClockAdjustment{Method: AdjustmentSlew, Offset: offset}
}
