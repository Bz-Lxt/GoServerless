package store

import (
	"context"
	"fmt"

	"github.com/gogo/goserverless/internal/model"
	"github.com/gogo/goserverless/internal/timeutil"
)

func (s *Store) ListTriggers(ctx context.Context, functionID string) ([]*model.Trigger, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, function_id, kind, cron_expr, enabled, created_at, updated_at
		FROM triggers WHERE function_id=$1 ORDER BY kind, created_at`, functionID)
	if err != nil {
		return nil, fmt.Errorf("list triggers: %w", err)
	}
	defer rows.Close()
	out := make([]*model.Trigger, 0)
	for rows.Next() {
		t, err := scanTrigger(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) ListEnabledCron(ctx context.Context) ([]*model.Trigger, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, function_id, kind, cron_expr, enabled, created_at, updated_at
		FROM triggers WHERE kind='cron' AND enabled=TRUE`)
	if err != nil {
		return nil, fmt.Errorf("list cron: %w", err)
	}
	defer rows.Close()
	out := make([]*model.Trigger, 0)
	for rows.Next() {
		t, err := scanTrigger(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) ReplaceTriggers(ctx context.Context, functionID string, items []*model.Trigger) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM triggers WHERE function_id=$1`, functionID); err != nil {
		return fmt.Errorf("clear triggers: %w", err)
	}
	now := timeutil.NowUTC()
	for _, t := range items {
		t.FunctionID = functionID
		if t.CreatedAt.IsZero() {
			t.CreatedAt = now
		}
		t.UpdatedAt = now
		if err := insertTrigger(ctx, tx, t); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func scanTrigger(row rowScanner) (*model.Trigger, error) {
	var t model.Trigger
	var kind string
	if err := row.Scan(&t.ID, &t.FunctionID, &kind, &t.CronExpr, &t.Enabled, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, err
	}
	t.Kind = model.TriggerKind(kind)
	return &t, nil
}
