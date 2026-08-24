package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/gogo/goserverless/internal/idgen"
	"github.com/gogo/goserverless/internal/invoker"
	"github.com/gogo/goserverless/internal/logger"
	"github.com/gogo/goserverless/internal/model"
	"github.com/gogo/goserverless/internal/store"
)

type Recorder interface {
	Record(fn *model.Function, kind model.TriggerKind, res *invoker.Result)
}

type Scheduler struct {
	st   *store.Store
	inv  *invoker.Invoker
	rec  Recorder
	mu   sync.Mutex
	last map[string]time.Time
}

func New(st *store.Store, inv *invoker.Invoker, rec Recorder) *Scheduler {
	return &Scheduler{st: st, inv: inv, rec: rec, last: map[string]time.Time{}}
}

func (s *Scheduler) Start(ctx context.Context) {
	t := time.NewTicker(1 * time.Second)
	go func() {
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-t.C:
				s.tick(ctx, now)
			}
		}
	}()
}

func (s *Scheduler) tick(ctx context.Context, now time.Time) {
	items, err := s.st.ListEnabledCron(ctx)
	if err != nil {
		logger.Warn(ctx, "list cron failed", "err", err)
		return
	}
	parser := cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	for _, tr := range items {
		sched, err := parser.Parse(tr.CronExpr)
		if err != nil {
			continue
		}
		s.mu.Lock()
		prev := s.last[tr.ID]
		s.mu.Unlock()
		if prev.IsZero() {
			s.mu.Lock()
			s.last[tr.ID] = now
			s.mu.Unlock()
			continue
		}
		next := sched.Next(prev)
		if now.Before(next) {
			continue
		}
		s.mu.Lock()
		s.last[tr.ID] = now
		s.mu.Unlock()
		go s.fire(ctx, tr)
	}
}

func (s *Scheduler) fire(ctx context.Context, tr *model.Trigger) {
	fn, err := s.st.GetFunctionByID(ctx, tr.FunctionID)
	if err != nil {
		logger.Warn(ctx, "cron function missing", "trigger", tr.ID, "err", err)
		return
	}
	if fn.Status != model.StatusReady {
		return
	}
	ev := invoker.HTTPEvent{
		Method:  "POST",
		Path:    "/cron",
		Headers: map[string]string{"X-GSCF-Trigger": "cron"},
		Query:   map[string]string{},
		Body:    `{"trigger":"cron","id":"` + idgen.RequestID() + `"}`,
	}
	res, err := s.inv.Invoke(ctx, fn.Name, model.TriggerCron, ev)
	if err != nil {
		logger.Warn(ctx, "cron invoke failed", "fn", fn.Name, "err", err)
		return
	}
	if s.rec != nil {
		s.rec.Record(fn, model.TriggerCron, res)
	}
}
