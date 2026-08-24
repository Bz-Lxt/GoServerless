package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/gogo/goserverless/internal/model"
)

func (s *Store) NextVersion(ctx context.Context, functionID string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(version), 0) FROM function_versions WHERE function_id = $1`, functionID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("next version: %w", err)
	}
	return n + 1, nil
}

func (s *Store) InsertVersion(ctx context.Context, v *model.FunctionVersion) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO function_versions (id, function_id, version, status, code, artifact_path, build_log, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		v.ID, v.FunctionID, v.Version, string(v.Status), v.Code, v.ArtifactPath, v.BuildLog, v.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert version: %w", err)
	}
	return nil
}

func (s *Store) UpdateVersion(ctx context.Context, v *model.FunctionVersion) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE function_versions SET status=$2, artifact_path=$3, build_log=$4 WHERE id=$1`,
		v.ID, string(v.Status), v.ArtifactPath, v.BuildLog,
	)
	if err != nil {
		return fmt.Errorf("update version: %w", err)
	}
	return nil
}

func (s *Store) GetVersion(ctx context.Context, functionID string, version int) (*model.FunctionVersion, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, function_id, version, status, code, artifact_path, build_log, created_at
		FROM function_versions WHERE function_id=$1 AND version=$2`, functionID, version)
	v, err := scanVersion(row)
	if err != nil {
		return nil, mapNotFound(err, "version", fmt.Sprintf("%s@%d", functionID, version))
	}
	return v, nil
}

func (s *Store) LatestCode(ctx context.Context, functionID string) (string, error) {
	var code string
	err := s.pool.QueryRow(ctx, `
		SELECT code FROM function_versions WHERE function_id=$1 ORDER BY version DESC LIMIT 1`, functionID).Scan(&code)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("latest code: %w", err)
	}
	return code, nil
}

func (s *Store) ListVersions(ctx context.Context, functionID string, limit int) ([]*model.FunctionVersion, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, function_id, version, status, code, artifact_path, build_log, created_at
		FROM function_versions WHERE function_id=$1 ORDER BY version DESC LIMIT $2`, functionID, limit)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	defer rows.Close()
	out := make([]*model.FunctionVersion, 0)
	for rows.Next() {
		v, err := scanVersion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) TrimVersions(ctx context.Context, functionID string, keep int) error {
	if keep < 1 {
		keep = 10
	}
	_, err := s.pool.Exec(ctx, `
		DELETE FROM function_versions
		WHERE function_id=$1 AND version NOT IN (
			SELECT version FROM function_versions WHERE function_id=$1 ORDER BY version DESC LIMIT $2
		)`, functionID, keep)
	if err != nil {
		return fmt.Errorf("trim versions: %w", err)
	}
	return nil
}

func scanVersion(row rowScanner) (*model.FunctionVersion, error) {
	var v model.FunctionVersion
	var st string
	if err := row.Scan(&v.ID, &v.FunctionID, &v.Version, &st, &v.Code, &v.ArtifactPath, &v.BuildLog, &v.CreatedAt); err != nil {
		return nil, err
	}
	v.Status = model.FunctionStatus(st)
	return &v, nil
}
