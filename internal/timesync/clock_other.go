//go:build !linux

package timesync

import (
	"fmt"
	"time"
)

func ApplyAdjustment(adj *ClockAdjustment) error {
	switch adj.Method {
	case AdjustmentNone:
		return nil
	case AdjustmentStep, AdjustmentSlew:
		return fmt.Errorf("clock adjustment not supported on this platform")
	}
	return nil
}

func GetClockOffset() (time.Duration, error) {
	return 0, fmt.Errorf("clock offset not available on this platform")
}
