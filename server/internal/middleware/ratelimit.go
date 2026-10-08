package middleware

import (
	"net/http"
	"sync"
	"time"

	"domain-connect-backend/internal/errorz"
)

// rateLimiter is a per-client token bucket.
type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64
	burst   float64
}

type bucket struct {
	tokens float64
	last   time.Time
}

// RateLimit returns middleware allowing `perMinute` requests per client, with
// short bursts up to `burst` absorbed.
func RateLimit(perMinute, burst int) func(http.Handler) http.Handler {
	rl := &rateLimiter{
		buckets: make(map[string]*bucket),
		rate:    float64(perMinute) / 60.0,
		burst:   float64(burst),
	}
	go rl.cleanup()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.allow(ClientIP(r)) {
				w.Header().Set("Retry-After", "60")
				errorz.ErrTooManyRequests.SendError(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (rl *rateLimiter) allow(key string) bool {
	if key == "" {
		key = "unknown"
	}
	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[key]
	if !ok {
		rl.buckets[key] = &bucket{tokens: rl.burst - 1, last: now}
		return true
	}

	b.tokens += now.Sub(b.last).Seconds() * rl.rate
	if b.tokens > rl.burst {
		b.tokens = rl.burst
	}
	b.last = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// cleanup drops buckets for clients that have gone quiet, so the map does not
// grow without bound on a public endpoint.
func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cutoff := time.Now().Add(-15 * time.Minute)
		rl.mu.Lock()
		for key, b := range rl.buckets {
			if b.last.Before(cutoff) {
				delete(rl.buckets, key)
			}
		}
		rl.mu.Unlock()
	}
}
