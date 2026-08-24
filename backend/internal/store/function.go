package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/gogo/goserverless/internal/model"
	"github.com/gogo/goserverless/internal/timeutil"
)

func (s *Store) CreateFunction(ctx context.Context, fn *model.Function, draft *model.FunctionVersion, httpTrigger *model.Trigger) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	env, err := json.Marshal(fn.Env)
	if err != nil {
		return model.Invalid("env is not serializable")
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO functions (
			id, name, runtime, status, description, timeout_sec, memory_mb, cpu_nano,
			max_concurrency, env_json, current_version, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		fn.ID, fn.Name, string(fn.Runtime), string(fn.Status), fn.Description,
		fn.TimeoutSec, fn.MemoryMB, fn.CPUNano, fn.MaxConcurrency, env,
		fn.CurrentVersion, fn.CreatedAt, fn.UpdatedAt,
	)
	if err != nil {
		if isUnique(err) {
			return model.Conflict(fmt.Sprintf("function %q already exists", fn.Name))
		}
		return fmt.Errorf("insert function: %w", err)
	}
	if draft != nil {
		if err := insertVersion(ctx, tx, draft); err != nil {
			return err
		}
	}
	if httpTrigger != nil {
		if err := insertTrigger(ctx, tx, httpTrigger); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) GetFunctionByName(ctx context.Context, name string) (*model.Function, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, name, runtime, status, description, timeout_sec, memory_mb, cpu_nano,
		       max_concurrency, env_json, current_version, created_at, updated_at
		FROM functions WHERE name = $1`, name)
	fn, err := scanFunction(row)
	if err != nil {
		return nil, mapNotFound(err, "function", name)
	}
	return fn, nil
}

func (s *Store) GetFunctionByID(ctx context.Context, id string) (*model.Function, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, name, runtime, status, description, timeout_sec, memory_mb, cpu_nano,
		       max_concurrency, env_json, current_version, created_at, updated_at
		FROM functions WHERE id = $1`, id)
	fn, err := scanFunction(row)
	if err != nil {
		return nil, mapNotFound(err, "function", id)
	}
	return fn, nil
}

func (s *Store) ListFunctions(ctx context.Context) ([]*model.Function, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, runtime, status, description, timeout_sec, memory_mb, cpu_nano,
		       max_concurrency, env_json, current_version, created_at, updated_at
		FROM functions ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list functions: %w", err)
	}
	defer rows.Close()
	out := make([]*model.Function, 0)
	for rows.Next() {
		fn, err := scanFunction(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, fn)
	}
	return out, rows.Err()
}

func (s *Store) UpdateFunction(ctx context.Context, fn *model.Function) error {
	env, err := json.Marshal(fn.Env)
	if err != nil {
		return model.Invalid("env is not serializable")
	}
	fn.UpdatedAt = timeutil.NowUTC()
	tag, err := s.pool.Exec(ctx, `
		UPDATE functions SET
			status=$2, description=$3, timeout_sec=$4, memory_mb=$5, cpu_nano=$6,
			max_concurrency=$7, env_json=$8, current_version=$9, updated_at=$10
		WHERE id=$1`,
		fn.ID, string(fn.Status), fn.Description, fn.TimeoutSec, fn.MemoryMB, fn.CPUNano,
		fn.MaxConcurrency, env, fn.CurrentVersion, fn.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update function: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.NotFound("function", fn.Name)
	}
	return nil
}

func (s *Store) DeleteFunction(ctx context.Context, name string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM functions WHERE name = $1`, name)
	if err != nil {
		return fmt.Errorf("delete function: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.NotFound("function", name)
	}
	return nil
}

func (s *Store) CompareAndSetStatus(ctx context.Context, id string, from, to model.FunctionStatus) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE functions SET status=$3, updated_at=$4 WHERE id=$1 AND status=$2`,
		id, string(from), string(to), timeutil.NowUTC(),
	)
	if err != nil {
		return fmt.Errorf("cas status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.Conflict("function status changed concurrently")
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanFunction(row rowScanner) (*model.Function, error) {
	var (
		fn  model.Function
		rt  string
		st  string
		raw []byte
	)
	err := row.Scan(
		&fn.ID, &fn.Name, &rt, &st, &fn.Description, &fn.TimeoutSec, &fn.MemoryMB,
		&fn.CPUNano, &fn.MaxConcurrency, &raw, &fn.CurrentVersion, &fn.CreatedAt, &fn.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	fn.Runtime = model.RuntimeName(rt)
	fn.Status = model.FunctionStatus(st)
	fn.Env = map[string]string{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &fn.Env); err != nil {
			return nil, fmt.Errorf("decode env: %w", err)
		}
	}
	return &fn, nil
}

func isUnique(err error) bool {
	return err != nil && (containsSQLState(err, "23505"))
}

func containsSQLState(err error, code string) bool {
	type st interface{ SQLState() string }
	for err != nil {
		if e, ok := err.(st); ok && e.SQLState() == code {
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

func insertVersion(ctx context.Context, tx pgx.Tx, v *model.FunctionVersion) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO function_versions (id, function_id, version, status, code, artifact_path, build_log, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		v.ID, v.FunctionID, v.Version, string(v.Status), v.Code, v.ArtifactPath, v.BuildLog, v.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert version: %w", err)
	}
	return nil
}

func insertTrigger(ctx context.Context, tx pgx.Tx, t *model.Trigger) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO triggers (id, function_id, kind, cron_expr, enabled, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		t.ID, t.FunctionID, string(t.Kind), t.CronExpr, t.Enabled, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert trigger: %w", err)
	}
	return nil
}
