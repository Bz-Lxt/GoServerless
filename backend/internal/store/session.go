package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Store) PutSession(ctx context.Context, token, user string, exp time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (token, username, expires_at) VALUES ($1,$2,$3)
		ON CONFLICT (token) DO UPDATE SET username=EXCLUDED.username, expires_at=EXCLUDED.expires_at`,
		token, user, exp,
	)
	if err != nil {
		return fmt.Errorf("put session: %w", err)
	}
	return nil
}

func (s *Store) GetSession(ctx context.Context, token string) (string, time.Time, error) {
	var user string
	var exp time.Time
	err := s.pool.QueryRow(ctx, `SELECT username, expires_at FROM sessions WHERE token=$1`, token).Scan(&user, &exp)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", time.Time{}, err
		}
		return "", time.Time{}, fmt.Errorf("get session: %w", err)
	}
	return user, exp, nil
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token=$1`, token)
	return err
}

func (s *Store) SweepSessions(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at < NOW()`)
	return err
}
