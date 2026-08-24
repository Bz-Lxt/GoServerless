package store

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/gogo/goserverless/internal/model"
)

func (s *Store) InsertInvocation(ctx context.Context, inv *model.Invocation) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO invocations (
			id, function_id, version, trigger_kind, status_code, success, cold_start,
			wakeup_ms, exec_ms, e2e_ms, error, logs, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		inv.ID, inv.FunctionID, inv.Version, string(inv.TriggerKind), inv.StatusCode,
		inv.Success, inv.ColdStart, inv.WakeupMS, inv.ExecMS, inv.E2EMS, inv.Error, inv.Logs, inv.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert invocation: %w", err)
	}
	return nil
}

func (s *Store) ListInvocations(ctx context.Context, functionID string, limit int) ([]*model.Invocation, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, function_id, version, trigger_kind, status_code, success, cold_start,
		       wakeup_ms, exec_ms, e2e_ms, error, logs, created_at
		FROM invocations WHERE function_id=$1 ORDER BY created_at DESC LIMIT $2`, functionID, limit)
	if err != nil {
		return nil, fmt.Errorf("list invocations: %w", err)
	}
	defer rows.Close()
	out := make([]*model.Invocation, 0)
	for rows.Next() {
		inv, err := scanInvocation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

func (s *Store) Metrics(ctx context.Context, functionID string) (*model.FunctionMetrics, error) {
	m := &model.FunctionMetrics{}
	var last any
	err := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE success),
			COUNT(*) FILTER (WHERE NOT success),
			COUNT(*) FILTER (WHERE cold_start),
			COALESCE(AVG(exec_ms), 0),
			COALESCE(AVG(wakeup_ms), 0),
			MAX(created_at)
		FROM invocations WHERE function_id=$1`, functionID).Scan(
		&m.Invocations, &m.Successes, &m.Failures, &m.ColdStarts,
		&m.AvgExecMS, &m.AvgWakeupMS, &last,
	)
	if err != nil {
		return nil, fmt.Errorf("metrics: %w", err)
	}
	if t, ok := last.(interface{ UTC() interface{} }); ok {
		_ = t
	}

	rows, err := s.pool.Query(ctx, `
		SELECT exec_ms FROM invocations WHERE function_id=$1 ORDER BY created_at DESC LIMIT 500`, functionID)
	if err != nil {
		return nil, fmt.Errorf("percentile sample: %w", err)
	}
	defer rows.Close()
	samples := make([]float64, 0, 128)
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		samples = append(samples, float64(v))
	}
	m.P95ExecMS = percentile(samples, 0.95)
	m.P99ExecMS = percentile(samples, 0.99)
	return m, rows.Err()
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	cp := append([]float64(nil), values...)
	sort.Float64s(cp)
	if p <= 0 {
		return cp[0]
	}
	if p >= 1 {
		return cp[len(cp)-1]
	}
	idx := p * float64(len(cp)-1)
	lo := int(math.Floor(idx))
	hi := int(math.Ceil(idx))
	if lo == hi {
		return cp[lo]
	}
	frac := idx - float64(lo)
	return cp[lo]*(1-frac) + cp[hi]*frac
}

func scanInvocation(row rowScanner) (*model.Invocation, error) {
	var inv model.Invocation
	var kind string
	if err := row.Scan(
		&inv.ID, &inv.FunctionID, &inv.Version, &kind, &inv.StatusCode, &inv.Success, &inv.ColdStart,
		&inv.WakeupMS, &inv.ExecMS, &inv.E2EMS, &inv.Error, &inv.Logs, &inv.CreatedAt,
	); err != nil {
		return nil, err
	}
	inv.TriggerKind = model.TriggerKind(kind)
	return &inv, nil
}
