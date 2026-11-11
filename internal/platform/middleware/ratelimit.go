package middleware

import (
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"go-aiq-backend/internal/platform/problem"
)

// RateLimiter is a small in-memory sliding-window limiter: each key may be
// allowed at most limit times within any window. State lives in this process
// only, so limits are per server instance and reset on restart.
//
// Keys that have been idle for a full window are evicted, so memory stays
// bounded by the number of keys active within one window.
type RateLimiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu        sync.Mutex
	hits      map[string][]time.Time
	lastSweep time.Time
}

// NewRateLimiter allows limit calls per key within any window.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window, now: time.Now, hits: map[string][]time.Time{}}
}

// Allow records a call for key and reports whether it is within the limit.
// When it is not, retryAfter is how long until the key is allowed again.
func (l *RateLimiter) Allow(key string) (allowed bool, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	recent := l.hits[key][:0]
	for _, at := range l.hits[key] {
		if now.Sub(at) < l.window {
			recent = append(recent, at)
		}
	}
	if len(recent) >= l.limit {
		l.hits[key] = recent
		return false, recent[0].Add(l.window).Sub(now)
	}
	l.hits[key] = append(recent, now)
	return true, 0
}

// sweep drops keys with no call inside the window. It runs at most once per
// window so the cost is spread across calls instead of needing a goroutine.
func (l *RateLimiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < l.window {
		return
	}
	l.lastSweep = now
	for key, hits := range l.hits {
		if len(hits) == 0 || now.Sub(hits[len(hits)-1]) >= l.window {
			delete(l.hits, key)
		}
	}
}

// RateLimit rejects requests over the limiter's budget with 429 and a
// Retry-After header. key picks what is limited (see ClientIPKey, DeviceKey).
func RateLimit(limiter *RateLimiter, key func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, retryAfter := limiter.Allow(key(c))
		if !allowed {
			c.Header("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
			problem.Abort(c, problem.TooManyRequests, "Too many requests; retry later.")
			return
		}
		c.Next()
	}
}

// ClientIPKey limits by client IP. It is only as accurate as the router's
// trusted proxies (TRUSTED_PROXIES): if the proxy in front of the server is
// not trusted, every visitor shares the proxy's address and one budget.
func ClientIPKey(c *gin.Context) string { return c.ClientIP() }

// DeviceKey limits by authenticated device. It must run after DeviceAuth.
func DeviceKey(c *gin.Context) string { return DeviceID(c).String() }
