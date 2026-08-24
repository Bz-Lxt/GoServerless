package limiter

import (
	"sync"
	"time"
)

type window struct {
	start time.Time
	n     int
}

type Limiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	byKey    map[string]*window
}

func New(limit int, d time.Duration) *Limiter {
	if limit <= 0 {
		limit = 30
	}
	if d <= 0 {
		d = time.Second
	}
	return &Limiter{limit: limit, window: d, byKey: map[string]*window{}}
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	w := l.byKey[key]
	if w == nil || now.Sub(w.start) >= l.window {
		l.byKey[key] = &window{start: now, n: 1}
		return true
	}
	if w.n >= l.limit {
		return false
	}
	w.n++
	return true
}
