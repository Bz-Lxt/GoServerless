package service

import (
	"context"
	"os"
	"path/filepath"

	"github.com/gogo/goserverless/internal/builder"
	"github.com/gogo/goserverless/internal/cache"
	"github.com/gogo/goserverless/internal/config"
	"github.com/gogo/goserverless/internal/idgen"
	"github.com/gogo/goserverless/internal/model"
	"github.com/gogo/goserverless/internal/pool"
	rt "github.com/gogo/goserverless/internal/runtime"
	"github.com/gogo/goserverless/internal/store"
	"github.com/gogo/goserverless/internal/timeutil"
	"github.com/gogo/goserverless/internal/validate"
)

type Functions struct {
	cfg *config.Config
	st  *store.Store
	reg *rt.Registry
	pipe *builder.Pipeline
	rd  *cache.Redis
	pool *pool.Pool
}

func NewFunctions(cfg *config.Config, st *store.Store, reg *rt.Registry, pipe *builder.Pipeline, rd *cache.Redis, p *pool.Pool) *Functions {
	return &Functions{cfg: cfg, st: st, reg: reg, pipe: pipe, rd: rd, pool: p}
}

func (s *Functions) endpoint(name string) string {
	return s.cfg.PublicBaseURL + "/api/v1/run/" + name
}

func (s *Functions) decorate(fn *model.Function) *model.Function {
	if fn == nil {
		return nil
	}
	fn.Endpoint = s.endpoint(fn.Name)
	if fn.Env != nil {
		masked := make(map[string]string, len(fn.Env))
		for k, v := range fn.Env {
			if looksSecret(k) && v != "" {
				masked[k] = "••••••"
			} else {
				masked[k] = v
			}
		}
		fn.Env = masked
	}
	return fn
}

func looksSecret(k string) bool {
	u := k
	for _, p := range []string{"KEY", "SECRET", "TOKEN", "PASSWORD", "PASS"} {
		if containsFold(u, p) {
			return true
		}
	}
	return false
}

func containsFold(s, sub string) bool {
	return len(s) >= len(sub) && (indexFold(s, sub) >= 0)
}

func indexFold(s, sub string) int {
	ls, lsub := len(s), len(sub)
	for i := 0; i+lsub <= ls; i++ {
		ok := true
		for j := 0; j < lsub; j++ {
			a, b := s[i+j], sub[j]
			if a >= 'a' && a <= 'z' {
				a -= 32
			}
			if a != b {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

func (s *Functions) Create(ctx context.Context, in model.CreateFunctionInput) (*model.Function, error) {
	if in.TimeoutSec == 0 {
		in.TimeoutSec = 30
	}
	if in.MemoryMB == 0 {
		in.MemoryMB = 128
	}
	if in.MaxConcurrency == 0 {
		in.MaxConcurrency = 10
	}
	if in.Env == nil {
		in.Env = map[string]string{}
	}
	if in.Code == "" {
		rtImpl, err := s.reg.Get(in.Runtime)
		if err != nil {
			return nil, err
		}
		in.Code = rtImpl.DefaultTemplate()
	}
	if err := validate.CreateInput(in); err != nil {
		return nil, err
	}
	now := timeutil.NowUTC()
	fn := &model.Function{
		ID:             idgen.UUID(),
		Name:           in.Name,
		Runtime:        in.Runtime,
		Status:         model.StatusDraft,
		Description:    in.Description,
		TimeoutSec:     in.TimeoutSec,
		MemoryMB:       in.MemoryMB,
		CPUNano:        s.cfg.DefaultCPUNano,
		MaxConcurrency: in.MaxConcurrency,
		Env:            in.Env,
		CurrentVersion: 0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	draft := &model.FunctionVersion{
		ID:         idgen.UUID(),
		FunctionID: fn.ID,
		Version:    0,
		Status:     model.StatusDraft,
		Code:       in.Code,
		CreatedAt:  now,
	}
	trig := &model.Trigger{
		ID:         idgen.UUID(),
		FunctionID: fn.ID,
		Kind:       model.TriggerHTTP,
		Enabled:    true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.st.CreateFunction(ctx, fn, draft, trig); err != nil {
		return nil, err
	}
	if s.rd != nil {
		s.rd.SetRoute(ctx, fn.Name, string(fn.Status))
	}
	return s.decorate(fn), nil
}

func (s *Functions) Get(ctx context.Context, name string) (*model.Function, string, error) {
	if err := validate.FunctionName(name); err != nil {
		return nil, "", err
	}
	fn, err := s.st.GetFunctionByName(ctx, name)
	if err != nil {
		return nil, "", err
	}
	code, err := s.st.LatestCode(ctx, fn.ID)
	if err != nil {
		return nil, "", err
	}
	return s.decorate(fn), code, nil
}

func (s *Functions) List(ctx context.Context) ([]*model.Function, error) {
	items, err := s.st.ListFunctions(ctx)
	if err != nil {
		return nil, err
	}
	for _, fn := range items {
		s.decorate(fn)
	}
	return items, nil
}

func (s *Functions) Update(ctx context.Context, name string, in model.UpdateFunctionInput) (*model.Function, error) {
	fn, err := s.st.GetFunctionByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if in.Description != nil {
		fn.Description = *in.Description
	}
	if in.TimeoutSec != nil {
		if err := validate.Timeout(*in.TimeoutSec); err != nil {
			return nil, err
		}
		fn.TimeoutSec = *in.TimeoutSec
	}
	if in.MemoryMB != nil {
		if err := validate.MemoryMB(*in.MemoryMB); err != nil {
			return nil, err
		}
		fn.MemoryMB = *in.MemoryMB
	}
	if in.MaxConcurrency != nil {
		if err := validate.Concurrency(*in.MaxConcurrency); err != nil {
			return nil, err
		}
		fn.MaxConcurrency = *in.MaxConcurrency
	}
	if in.Env != nil {
		if err := validate.Env(in.Env); err != nil {
			return nil, err
		}
		fn.Env = in.Env
	}
	if err := s.st.UpdateFunction(ctx, fn); err != nil {
		return nil, err
	}
	return s.decorate(fn), nil
}

func (s *Functions) Delete(ctx context.Context, name string) error {
	fn, err := s.st.GetFunctionByName(ctx, name)
	if err != nil {
		return err
	}
	if err := s.st.DeleteFunction(ctx, name); err != nil {
		return err
	}
	_ = os.RemoveAll(filepath.Join(s.cfg.ArtifactRoot, fn.Name))
	if s.rd != nil {
		s.rd.DelRoute(ctx, name)
	}
	return nil
}

func (s *Functions) Deploy(ctx context.Context, name string, code string) (*model.Function, error) {
	fn, err := s.st.GetFunctionByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if code == "" {
		code, err = s.st.LatestCode(ctx, fn.ID)
		if err != nil {
			return nil, err
		}
	}
	if err := validate.Code(code); err != nil {
		return nil, err
	}
	if _, err := s.pipe.NewVersion(ctx, fn, code); err != nil {
		return nil, err
	}
	s.pipe.Enqueue(fn.Name)
	if s.rd != nil {
		s.rd.SetRoute(ctx, fn.Name, string(model.StatusBuilding))
	}
	fn, err = s.st.GetFunctionByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return s.decorate(fn), nil
}

func (s *Functions) Rollback(ctx context.Context, name string, version int) (*model.Function, error) {
	fn, err := s.st.GetFunctionByName(ctx, name)
	if err != nil {
		return nil, err
	}
	v, err := s.st.GetVersion(ctx, fn.ID, version)
	if err != nil {
		return nil, err
	}
	if v.Status != model.StatusReady && v.Code == "" {
		return nil, model.Invalid("target version has no usable code")
	}
	return s.Deploy(ctx, name, v.Code)
}

func (s *Functions) Versions(ctx context.Context, name string) ([]*model.FunctionVersion, error) {
	fn, err := s.st.GetFunctionByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return s.st.ListVersions(ctx, fn.ID, 10)
}

func (s *Functions) Triggers(ctx context.Context, name string) ([]*model.Trigger, error) {
	fn, err := s.st.GetFunctionByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return s.st.ListTriggers(ctx, fn.ID)
}

func (s *Functions) ReplaceTriggers(ctx context.Context, name string, items []model.Trigger) ([]*model.Trigger, error) {
	if err := validate.Triggers(items); err != nil {
		return nil, err
	}
	fn, err := s.st.GetFunctionByName(ctx, name)
	if err != nil {
		return nil, err
	}
	now := timeutil.NowUTC()
	out := make([]*model.Trigger, 0, len(items))
	for _, t := range items {
		cp := t
		if cp.ID == "" {
			cp.ID = idgen.UUID()
		}
		cp.FunctionID = fn.ID
		cp.CreatedAt = now
		cp.UpdatedAt = now
		out = append(out, &cp)
	}
	if err := s.st.ReplaceTriggers(ctx, fn.ID, out); err != nil {
		return nil, err
	}
	return s.st.ListTriggers(ctx, fn.ID)
}

func (s *Functions) ReadyFunction(ctx context.Context, name string) (*model.Function, *model.FunctionVersion, error) {
	fn, err := s.st.GetFunctionByName(ctx, name)
	if err != nil {
		return nil, nil, err
	}
	if fn.Status != model.StatusReady {
		return nil, nil, model.NotReady(name)
	}
	ver, err := s.st.GetVersion(ctx, fn.ID, fn.CurrentVersion)
	if err != nil {
		return nil, nil, err
	}
	if ver.Status != model.StatusReady {
		return nil, nil, model.NotReady(name)
	}
	return fn, ver, nil
}

func (s *Functions) Metrics(ctx context.Context, name string) (*model.FunctionMetrics, error) {
	fn, err := s.st.GetFunctionByName(ctx, name)
	if err != nil {
		return nil, err
	}
	m, err := s.st.Metrics(ctx, fn.ID)
	if err != nil {
		return nil, err
	}
	m.FunctionName = fn.Name
	if s.pool != nil {
		a, i := s.pool.Stats()
		m.ActiveSlots = a
		m.IdleSlots = i
	}
	m.PoolMisses = m.ColdStarts
	if m.Invocations > m.ColdStarts {
		m.PoolHits = m.Invocations - m.ColdStarts
	}
	return m, nil
}

func (s *Functions) Invocations(ctx context.Context, name string, limit int) ([]*model.Invocation, error) {
	fn, err := s.st.GetFunctionByName(ctx, name)
	if err != nil {
		return nil, err
	}
	items, err := s.st.ListInvocations(ctx, fn.ID, limit)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		it.Name = fn.Name
		it.CreatedAt = it.CreatedAt.In(timeutil.Beijing())
	}
	return items, nil
}

func (s *Functions) Template(name model.RuntimeName) string {
	rtImpl, err := s.reg.Get(name)
	if err != nil {
		return ""
	}
	return rtImpl.DefaultTemplate()
}

func (s *Functions) Overview(ctx context.Context) (map[string]any, error) {
	fns, err := s.st.ListFunctions(ctx)
	if err != nil {
		return nil, err
	}
	var ready, building, failed int
	for _, f := range fns {
		switch f.Status {
		case model.StatusReady:
			ready++
		case model.StatusBuilding:
			building++
		case model.StatusFailed:
			failed++
		}
	}
	active, idle := 0, 0
	if s.pool != nil {
		active, idle = s.pool.Stats()
	}
	return map[string]any{
		"functions": len(fns),
		"ready":     ready,
		"building":  building,
		"failed":    failed,
		"pool_active": active,
		"pool_idle":   idle,
	}, nil
}
