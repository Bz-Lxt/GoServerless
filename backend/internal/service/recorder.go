package service

import (
	"context"
	"strings"

	"github.com/gogo/goserverless/internal/cache"
	"github.com/gogo/goserverless/internal/idgen"
	"github.com/gogo/goserverless/internal/invoker"
	"github.com/gogo/goserverless/internal/logger"
	"github.com/gogo/goserverless/internal/logstream"
	"github.com/gogo/goserverless/internal/model"
	"github.com/gogo/goserverless/internal/store"
	"github.com/gogo/goserverless/internal/timeutil"
)

type Recorder struct {
	st  *store.Store
	hub *logstream.Hub
	rd  *cache.Redis
}

func NewRecorder(st *store.Store, hub *logstream.Hub, rd *cache.Redis) *Recorder {
	return &Recorder{st: st, hub: hub, rd: rd}
}

func (r *Recorder) Record(fn *model.Function, kind model.TriggerKind, res *invoker.Result) {
	if res == nil {
		return
	}
	ctx := context.Background()
	name := fn.Name
	full, err := r.st.GetFunctionByName(ctx, name)
	if err != nil {
		logger.Warn(ctx, "record skip: function missing", "fn", name, "err", err)
		return
	}
	inv := &model.Invocation{
		ID:          idgen.UUID(),
		FunctionID:  full.ID,
		Name:        name,
		Version:     res.Version,
		TriggerKind: kind,
		StatusCode:  res.StatusCode,
		Success:     res.StatusCode >= 200 && res.StatusCode < 400 && res.Error == "",
		ColdStart:   res.ColdStart,
		WakeupMS:    timeutil.DurationMS(res.Wakeup),
		ExecMS:      timeutil.DurationMS(res.Exec),
		E2EMS:       timeutil.DurationMS(res.E2E),
		Error:       res.Error,
		Logs:        trimLogs(res.Logs),
		CreatedAt:   timeutil.NowUTC(),
	}
	if err := r.st.InsertInvocation(ctx, inv); err != nil {
		logger.Warn(ctx, "insert invocation failed", "err", err)
	}
	if r.rd != nil {
		r.rd.Incr(ctx, "metrics:"+name+":invocations")
		if inv.ColdStart {
			r.rd.Incr(ctx, "metrics:"+name+":cold")
		}
	}
	if r.hub != nil && inv.Logs != "" {
		r.hub.Publish(name, inv.Version, inv.Logs, inv.ColdStart)
	}
}

func trimLogs(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 16<<10 {
		return s[:16<<10] + "\n...truncated"
	}
	return s
}
