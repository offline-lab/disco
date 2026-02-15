package discovery

import (
	"sync"
	"time"
)

const (
	defaultMaxBurst = 10
	defaultRate     = 10
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	maxBurst int
	rate     int
	tokens   int
	lastTime time.Time
	mu       sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate, maxBurst int) *RateLimiter {
	if rate <= 0 {
		rate = defaultRate
	}
	if maxBurst <= 0 {
		maxBurst = defaultMaxBurst
	}

	return &RateLimiter{
		rate:     rate,
		maxBurst: maxBurst,
		tokens:   maxBurst,
		lastTime: time.Now(),
	}
}

// Allow checks if an action is allowed
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastTime)
	rl.lastTime = now

	rl.tokens += int(elapsed.Seconds()) * rl.rate
	if rl.tokens > rl.maxBurst {
		rl.tokens = rl.maxBurst
	}

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// DuplicateFilter filters duplicate messages based on message ID
type DuplicateFilter struct {
	seen     map[string]time.Time
	mu       sync.Mutex
	ttl      time.Duration
	stopChan chan struct{}
}

// NewDuplicateFilter creates a new duplicate filter
func NewDuplicateFilter(ttl time.Duration) *DuplicateFilter {
	df := &DuplicateFilter{
		seen:     make(map[string]time.Time),
		ttl:      ttl,
		stopChan: make(chan struct{}),
	}

	go df.cleanup()

	return df
}

// Stop halts the cleanup goroutine
func (df *DuplicateFilter) Stop() {
	close(df.stopChan)
}

// Seen checks if a message ID has been seen
func (df *DuplicateFilter) Seen(messageID string) bool {
	df.mu.Lock()
	defer df.mu.Unlock()

	_, exists := df.seen[messageID]
	if !exists {
		df.seen[messageID] = time.Now()
		return false
	}

	return true
}

// cleanup removes old entries from the filter
func (df *DuplicateFilter) cleanup() {
	ticker := time.NewTicker(df.ttl)
	defer ticker.Stop()

	for {
		select {
		case <-df.stopChan:
			return
		case <-ticker.C:
			df.mu.Lock()
			now := time.Now()
			for id, timestamp := range df.seen {
				if now.Sub(timestamp) > df.ttl {
					delete(df.seen, id)
				}
			}
			df.mu.Unlock()
		}
	}
}
