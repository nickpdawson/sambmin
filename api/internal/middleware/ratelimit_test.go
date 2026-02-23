package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterIPLimit(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute, 5, 15*time.Minute)

	// First 3 should be allowed
	for i := 0; i < 3; i++ {
		blocked, _ := rl.Check("1.2.3.4", "user1")
		if blocked {
			t.Fatalf("attempt %d should not be blocked", i+1)
		}
		rl.RecordFailure("1.2.3.4", "user1")
	}

	// 4th should be blocked
	blocked, retryAfter := rl.Check("1.2.3.4", "user1")
	if !blocked {
		t.Fatal("4th attempt should be blocked")
	}
	if retryAfter <= 0 {
		t.Fatal("retryAfter should be positive")
	}

	// Different IP should not be blocked
	blocked, _ = rl.Check("5.6.7.8", "user1")
	if blocked {
		t.Fatal("different IP should not be blocked by first IP's limit")
	}
}

func TestRateLimiterUserLimit(t *testing.T) {
	rl := NewRateLimiter(100, time.Minute, 2, 15*time.Minute)

	// First 2 should be allowed
	for i := 0; i < 2; i++ {
		blocked, _ := rl.Check("1.2.3.4", "targetuser")
		if blocked {
			t.Fatalf("attempt %d should not be blocked", i+1)
		}
		rl.RecordFailure("1.2.3.4", "targetuser")
	}

	// 3rd attempt for same username (different IP) should be blocked
	blocked, _ := rl.Check("5.6.7.8", "targetuser")
	if !blocked {
		t.Fatal("3rd attempt for same user should be blocked")
	}

	// Different username should not be blocked
	blocked, _ = rl.Check("1.2.3.4", "otheruser")
	if blocked {
		t.Fatal("different username should not be blocked")
	}
}

func TestRateLimiterEmptyUsername(t *testing.T) {
	rl := NewRateLimiter(100, time.Minute, 2, 15*time.Minute)

	// Empty username should not cause per-user rate limiting
	for i := 0; i < 10; i++ {
		blocked, _ := rl.Check("1.2.3.4", "")
		if blocked {
			t.Fatalf("attempt %d with empty username should not be blocked by user limit", i+1)
		}
		rl.RecordFailure("1.2.3.4", "")
	}
}

func TestPruneOld(t *testing.T) {
	now := time.Now()
	entries := []time.Time{
		now.Add(-2 * time.Minute),
		now.Add(-30 * time.Second),
		now,
	}

	result := pruneOld(entries, now, time.Minute)
	if len(result) != 2 {
		t.Errorf("expected 2 entries after pruning, got %d", len(result))
	}
}
