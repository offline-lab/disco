//go:build linux

package timesync

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

type timeval struct {
	Sec  int64
	Usec int64
}

func ApplyAdjustment(adj *ClockAdjustment) error {
	switch adj.Method {
	case AdjustmentNone:
		return nil
	case AdjustmentStep:
		return stepClock(adj.Absolute)
	case AdjustmentSlew:
		return slewClock(adj.Offset)
	}
	return nil
}

func stepClock(target time.Time) error {
	tv := &timeval{
		Sec:  target.Unix(),
		Usec: int64(target.Nanosecond() / 1000),
	}

	_, _, errno := syscall.Syscall(syscall.SYS_SETTIMEOFDAY,
		uintptr(unsafe.Pointer(tv)), 0, 0)
	if errno != 0 {
		return fmt.Errorf("settimeofday failed: %v", errno)
	}
	return nil
}

func slewClock(offset time.Duration) error {
	var offsetNanos int64
	if offset >= 0 {
		offsetNanos = int64(offset)
	} else {
		offsetNanos = -int64(-offset)
	}

	const (
		ADJ_OFFSET = 0x0001
		ADJ_NANO   = 0x2000
	)

	tx := &timex{
		Modes:  ADJ_OFFSET | ADJ_NANO,
		Offset: offsetNanos,
	}

	_, _, errno := syscall.Syscall6(syscall.SYS_ADJTIMEX,
		uintptr(unsafe.Pointer(tx)), 0, 0, 0, 0, 0)
	if errno != 0 {
		return fmt.Errorf("adjtimex failed: %v", errno)
	}
	return nil
}

type timex struct {
	Modes     uint32
	Offset    int64
	Freq      int64
	Maxerror  int64
	Esterror  int64
	Status    int32
	Constant  int64
	Precision int64
	Tolerance int64
	Time      timeval
	Tick      int64
	Ppsfreq   int64
	Jitter    int64
	Shift     int32
	Stabil    int64
	Jitcnt    int64
	Calcnt    int64
	Errcnt    int64
	Stbcnt    int64
	Tai       int32
	_         [44]byte
}

func GetClockOffset() (time.Duration, error) {
	tx := &timex{}

	_, _, errno := syscall.Syscall6(syscall.SYS_ADJTIMEX,
		uintptr(unsafe.Pointer(tx)), 0, 0, 0, 0, 0)
	if errno != 0 {
		return 0, fmt.Errorf("adjtimex failed: %v", errno)
	}

	return time.Duration(tx.Offset), nil
}
