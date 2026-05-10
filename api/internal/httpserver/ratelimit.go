package httpserver

import (
	"sync"
	"time"
)

type bucket struct {
	count int
	reset time.Time
}

type RateLimiter struct {
	limit   int
	window  time.Duration
	mu      sync.Mutex
	buckets map[string]bucket
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window, buckets: map[string]bucket{}}
}

func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	b := r.buckets[key]
	if b.reset.IsZero() || now.After(b.reset) {
		b = bucket{count: 0, reset: now.Add(r.window)}
	}
	if b.count >= r.limit {
		r.buckets[key] = b
		return false
	}
	b.count++
	r.buckets[key] = b
	return true
}
