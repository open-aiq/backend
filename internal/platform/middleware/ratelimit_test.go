package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func testLimiter(limit int, window time.Duration) (*RateLimiter, *time.Time) {
	now := time.Unix(1_800_000_000, 0)
	limiter := NewRateLimiter(limit, window)
	limiter.now = func() time.Time { return now }
	return limiter, &now
}

func TestRateLimiterAllowsUpToLimitPerKey(t *testing.T) {
	limiter, _ := testLimiter(3, time.Minute)
	for i := range 3 {
		if ok, _ := limiter.Allow("a"); !ok {
			t.Fatalf("call %d should be allowed", i)
		}
	}
	if ok, _ := limiter.Allow("a"); ok {
		t.Fatal("call over the limit should be refused")
	}
	if ok, _ := limiter.Allow("b"); !ok {
		t.Fatal("other keys must not be limited")
	}
}

func TestRateLimiterWindowSlides(t *testing.T) {
	limiter, now := testLimiter(2, time.Minute)
	limiter.Allow("a")
	*now = now.Add(20 * time.Second)
	limiter.Allow("a")

	*now = now.Add(10 * time.Second)
	ok, retryAfter := limiter.Allow("a")
	if ok || retryAfter != 30*time.Second {
		t.Fatalf("allowed = %v, retryAfter = %s; want refused for 30s", ok, retryAfter)
	}

	// The first call has now left the window; the second has not.
	*now = now.Add(30 * time.Second)
	if ok, _ := limiter.Allow("a"); !ok {
		t.Fatal("window should have slid")
	}
	if ok, _ := limiter.Allow("a"); ok {
		t.Fatal("only one slot should have been freed")
	}
}

func TestRateLimiterRefusedCallsDoNotExtendTheWait(t *testing.T) {
	limiter, now := testLimiter(1, time.Minute)
	limiter.Allow("a")
	for range 5 {
		*now = now.Add(10 * time.Second)
		limiter.Allow("a")
	}
	*now = now.Add(10 * time.Second)
	if ok, _ := limiter.Allow("a"); !ok {
		t.Fatal("refused calls must not count against the key")
	}
}

func TestRateLimiterEvictsIdleKeys(t *testing.T) {
	limiter, now := testLimiter(5, time.Minute)
	for i := range 100 {
		limiter.Allow("idle-" + strconv.Itoa(i))
	}
	*now = now.Add(30 * time.Second)
	limiter.Allow("active")

	*now = now.Add(30 * time.Second)
	limiter.Allow("other")

	if len(limiter.hits) != 2 {
		t.Fatalf("keys kept = %d, want 2 (active and other)", len(limiter.hits))
	}
	if _, kept := limiter.hits["active"]; !kept {
		t.Fatal("a key used inside the window must be kept")
	}
}

func TestRateLimitMiddlewareAnswers429Problem(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter, _ := testLimiter(1, time.Minute)
	r := gin.New()
	reached := 0
	r.GET("/x", RateLimit(limiter, func(c *gin.Context) string { return c.GetHeader("X-Key") }), func(c *gin.Context) {
		reached++
		c.Status(http.StatusNoContent)
	})
	serve := func(key string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("X-Key", key)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	if w := serve("a"); w.Code != http.StatusNoContent {
		t.Fatalf("first request status = %d", w.Code)
	}
	w := serve("a")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", w.Code)
	}
	if got := w.Header().Get("Retry-After"); got != "60" {
		t.Fatalf("Retry-After = %q, want 60", got)
	}
	if got := w.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("Content-Type = %q", got)
	}
	var body struct {
		Type   string `json:"type"`
		Status int    `json:"status"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Status != http.StatusTooManyRequests || body.Type != "https://docs.air-iq.net/reference/errors/#too-many-requests" {
		t.Fatalf("problem = %+v", body)
	}
	if reached != 1 {
		t.Fatalf("handler ran %d times, want 1", reached)
	}
	if w := serve("b"); w.Code != http.StatusNoContent {
		t.Fatalf("another key should pass, status = %d", w.Code)
	}
}

func TestClientIPKeyIgnoresForwardedForFromUntrustedPeers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if err := r.SetTrustedProxies(nil); err != nil {
		t.Fatal(err)
	}
	var key string
	r.GET("/x", func(c *gin.Context) { key = ClientIPKey(c) })
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = "203.0.113.9:4000"
	req.Header.Set("X-Forwarded-For", "198.51.100.1")
	r.ServeHTTP(httptest.NewRecorder(), req)
	if key != "203.0.113.9" {
		t.Fatalf("key = %q, want the connecting address", key)
	}
}
