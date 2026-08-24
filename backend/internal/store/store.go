package store

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gogo/goserverless/internal/logger"
	"github.com/gogo/goserverless/internal/model"
)

//go:embed schema.sql
var SchemaSQL string

type Store struct {
	pool *pgxpool.Pool
}

func Connect(ctx context.Context, url string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	cfg.MaxConns = 16
	cfg.MinConns = 2
	cfg.MaxConnIdleTime = 5 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

func (s *Store) Pool() *pgxpool.Pool { return s.pool }

func (s *Store) Migrate(ctx context.Context, sqlText string) error {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire: %w", err)
	}
	defer conn.Release()

	// Serialize DDL across replicas / double start (knowledge-base: pg_advisory_lock).
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock(88421023)"); err != nil {
		return fmt.Errorf("advisory lock: %w", err)
	}
	defer func() {
		if _, uerr := conn.Exec(ctx, "SELECT pg_advisory_unlock(88421023)"); uerr != nil {
			logger.Warn(ctx, "advisory unlock failed", "err", uerr)
		}
	}()
	if _, err := conn.Exec(ctx, sqlText); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	logger.Info(ctx, "schema migrated")
	return nil
}

func mapNotFound(err error, resource, name string) error {
	if err == nil {
		return nil
	}
	if err == pgx.ErrNoRows {
		return model.NotFound(resource, name)
	}
	return err
}
