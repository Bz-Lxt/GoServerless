package observe

import (
	"context"
	"sync"
	"time"
)

type CheckFunc func(ctx context.Context) error

type Report struct {
	OK      bool              `json:"ok"`
	Time    string            `json:"time"`
	TZ      string            `json:"tz"`
	Latency map[string]int64  `json:"latency_ms"`
	Errors  map[string]string `json:"errors,omitempty"`
}

type Registry struct {
	mu     sync.Mutex
	checks map[string]CheckFunc
}

func NewRegistry() *Registry {
	return &Registry{checks: map[string]CheckFunc{}}
}

func (r *Registry) Add(name string, fn CheckFunc) {
	r.mu.Lock()
	r.checks[name] = fn
	r.mu.Unlock()
}

func (r *Registry) Run(ctx context.Context, now string) Report {
	r.mu.Lock()
	snapshot := make(map[string]CheckFunc, len(r.checks))
	for k, v := range r.checks {
		snapshot[k] = v
	}
	r.mu.Unlock()

	rep := Report{OK: true, Time: now, TZ: "Asia/Shanghai", Latency: map[string]int64{}, Errors: map[string]string{}}
	for name, fn := range snapshot {
		start := time.Now()
		err := fn(ctx)
		rep.Latency[name] = time.Since(start).Milliseconds()
		if err != nil {
			rep.OK = false
			rep.Errors[name] = err.Error()
		}
	}
	if len(rep.Errors) == 0 {
		rep.Errors = nil
	}
	return rep
}
