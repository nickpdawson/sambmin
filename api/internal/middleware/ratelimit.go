package middleware

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements per-IP and per-username sliding window rate limiting.
type RateLimiter struct {
	mu sync.Mutex

	// ipAttempts tracks failed attempts per IP.
	ipAttempts map[string][]time.Time
	// userAttempts tracks failed attempts per username.
	userAttempts map[string][]time.Time

	ipLimit      int
	ipWindow     time.Duration
	userLimit    int
	userWindow   time.Duration
}

// NewRateLimiter creates a rate limiter with per-IP and per-username windows.
func NewRateLimiter(ipLimit int, ipWindow time.Duration, userLimit int, userWindow time.Duration) *RateLimiter {
	rl := &RateLimiter{
		ipAttempts:   make(map[string][]time.Time),
		userAttempts: make(map[string][]time.Time),
		ipLimit:      ipLimit,
		ipWindow:     ipWindow,
		userLimit:    userLimit,
		userWindow:   userWindow,
	}

	// Background cleanup every 5 minutes
	go rl.cleanupLoop()

	return rl
}

// Check returns true if the request should be rate-limited (blocked).
// Call this before processing the login attempt.
func (rl *RateLimiter) Check(ip, username string) (blocked bool, retryAfter time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Check per-IP limit
	rl.ipAttempts[ip] = pruneOld(rl.ipAttempts[ip], now, rl.ipWindow)
	if len(rl.ipAttempts[ip]) >= rl.ipLimit {
		oldest := rl.ipAttempts[ip][0]
		return true, rl.ipWindow - now.Sub(oldest)
	}

	// Check per-username limit
	if username != "" {
		rl.userAttempts[username] = pruneOld(rl.userAttempts[username], now, rl.userWindow)
		if len(rl.userAttempts[username]) >= rl.userLimit {
			oldest := rl.userAttempts[username][0]
			return true, rl.userWindow - now.Sub(oldest)
		}
	}

	return false, 0
}

// RecordFailure records a failed login attempt for rate limiting.
func (rl *RateLimiter) RecordFailure(ip, username string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rl.ipAttempts[ip] = append(rl.ipAttempts[ip], now)
	if username != "" {
		rl.userAttempts[username] = append(rl.userAttempts[username], now)
	}
}

// pruneOld removes entries older than the window.
func pruneOld(entries []time.Time, now time.Time, window time.Duration) []time.Time {
	cutoff := now.Add(-window)
	i := 0
	for i < len(entries) && entries[i].Before(cutoff) {
		i++
	}
	return entries[i:]
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for k, v := range rl.ipAttempts {
			v = pruneOld(v, now, rl.ipWindow)
			if len(v) == 0 {
				delete(rl.ipAttempts, k)
			} else {
				rl.ipAttempts[k] = v
			}
		}
		for k, v := range rl.userAttempts {
			v = pruneOld(v, now, rl.userWindow)
			if len(v) == 0 {
				delete(rl.userAttempts, k)
			} else {
				rl.userAttempts[k] = v
			}
		}
		rl.mu.Unlock()
	}
}

// ClientIP extracts the client IP from the request, preferring X-Real-IP
// (set by nginx), then X-Forwarded-For, then RemoteAddr.
func ClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// Take the first IP (client)
		if i := len(ip); i > 0 {
			for j := 0; j < len(ip); j++ {
				if ip[j] == ',' {
					return ip[:j]
				}
			}
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// RateLimitResponse sends a 429 response with Retry-After header.
func RateLimitResponse(w http.ResponseWriter, retryAfter time.Duration) {
	seconds := int(retryAfter.Seconds()) + 1
	w.Header().Set("Retry-After", fmt.Sprintf("%d", seconds))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	fmt.Fprintf(w, `{"error":"too many login attempts, retry after %d seconds"}`, seconds)
}
