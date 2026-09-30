// Package ratelimit is a small in-memory fixed-window rate limiter.
package ratelimit

import (
	"sync"
	"time"
)

type Limiter struct {
	limit     int
	window    time.Duration
	now       func() time.Time
	mu        sync.Mutex
	hits      map[string]*bucket
	lastSweep time.Time
}

type bucket struct {
	count int
	start time.Time
}

// New allows limit hits per key per window.
func New(limit int, window time.Duration) *Limiter {
	return &Limiter{limit: limit, window: window, now: time.Now, hits: map[string]*bucket{}}
}

// Allow records a hit for key and reports whether it is within the limit.
func (l *Limiter) Allow(key string) bool {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	if now.Sub(l.lastSweep) >= l.window {
		for k, b := range l.hits {
			if now.Sub(b.start) >= l.window {
				delete(l.hits, k)
			}
		}
		l.lastSweep = now
	}

	b := l.hits[key]
	if b == nil || now.Sub(b.start) >= l.window {
		l.hits[key] = &bucket{count: 1, start: now}
		return true
	}
	if b.count >= l.limit {
		return false
	}
	b.count++
	return true
}
