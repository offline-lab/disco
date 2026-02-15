package discovery

import (
	"testing"
	"time"
)

func TestRateLimiter_New(t *testing.T) {
	rl := NewRateLimiter(10, 10)
	if rl == nil {
		t.Fatal("NewRateLimiter() returned nil")
	}

	if rl.rate != 10 {
		t.Errorf("Expected rate 10, got %d", rl.rate)
	}

	if rl.maxBurst != 10 {
		t.Errorf("Expected maxBurst 10, got %d", rl.maxBurst)
	}

	if rl.tokens != 10 {
		t.Errorf("Expected initial tokens 10, got %d", rl.tokens)
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(10, 10)

	// Should allow first 10 requests
	for i := 0; i < 10; i++ {
		if !rl.Allow() {
			t.Errorf("Allow() returned false for request %d (should be within burst)", i)
		}
	}

	// 11th request should be denied (burst exhausted)
	if rl.Allow() {
		t.Error("Allow() returned true for request beyond burst limit")
	}

	// Wait for token replenishment
	time.Sleep(1*time.Second + 200*time.Millisecond)

	// Should allow again after waiting
	if !rl.Allow() {
		t.Error("Allow() returned false after waiting for replenishment")
	}
}

func TestRateLimiter_TokenReplenishment(t *testing.T) {
	rl := NewRateLimiter(1, 10)

	// Exhaust all tokens
	for i := 0; i < 10; i++ {
		rl.Allow()
	}

	// Verify tokens are exhausted
	if rl.Allow() {
		t.Error("Should not allow immediately after burst exhaustion")
	}

	// Wait for enough time to replenish 1 token (1 second for rate=1)
	time.Sleep(1*time.Second + 200*time.Millisecond)

	// Should allow 1 more request
	if !rl.Allow() {
		t.Error("Should allow after token replenishment")
	}

	// Should still be exhausted
	if rl.Allow() {
		t.Error("Should not allow second request without waiting longer")
	}
}

func TestDuplicateFilter_New(t *testing.T) {
	df := NewDuplicateFilter(5 * time.Minute)
	if df == nil {
		t.Fatal("NewDuplicateFilter() returned nil")
	}

	if df.ttl != 5*time.Minute {
		t.Errorf("Expected TTL 5m, got %v", df.ttl)
	}
}

func TestDuplicateFilter_Seen(t *testing.T) {
	df := NewDuplicateFilter(5 * time.Minute)

	messageID := "test-message-1"

	// First time should not be seen
	if df.Seen(messageID) {
		t.Error("Seen() returned true for new message")
	}

	// Second time should be seen
	if !df.Seen(messageID) {
		t.Error("Seen() returned false for duplicate message")
	}
}

func TestDuplicateFilter_MultipleMessages(t *testing.T) {
	df := NewDuplicateFilter(5 * time.Minute)

	messageIDs := []string{"msg-1", "msg-2", "msg-3"}

	// First pass - none seen
	for _, id := range messageIDs {
		if df.Seen(id) {
			t.Errorf("Seen() returned true for new message %s", id)
		}
	}

	// Second pass - all seen
	for _, id := range messageIDs {
		if !df.Seen(id) {
			t.Errorf("Seen() returned false for duplicate message %s", id)
		}
	}
}

func TestDuplicateFilter_Expiration(t *testing.T) {
	// Create filter with short TTL
	df := NewDuplicateFilter(100 * time.Millisecond)

	messageID := "test-expire-1"

	// First time not seen
	if df.Seen(messageID) {
		t.Error("Seen() returned true for new message")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not be seen after expiration
	if df.Seen(messageID) {
		t.Error("Seen() returned true for expired message")
	}
}

func TestDuplicateFilter_Concurrent(t *testing.T) {
	df := NewDuplicateFilter(5 * time.Minute)

	messageIDs := []string{"msg-1", "msg-2", "msg-3", "msg-4", "msg-5"}

	// Concurrent access
	done := make(chan bool)
	for _, id := range messageIDs {
		go func(msgID string) {
			df.Seen(msgID)
			done <- true
		}(id)
	}

	// Wait for all goroutines
	for range messageIDs {
		<-done
	}

	// Verify all messages are now marked as seen
	for _, id := range messageIDs {
		if !df.Seen(id) {
			t.Errorf("Message %s should be seen after concurrent access", id)
		}
	}
}
