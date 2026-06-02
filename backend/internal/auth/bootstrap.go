package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EnsureBootstrapAdmin creates an HR_ADMIN user if no users exist yet.
// Idempotent: if any user is present, this is a no-op.
func EnsureBootstrapAdmin(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger, username, password string) error {
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE deleted_at IS NULL`).Scan(&count); err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil
	}
	if username == "" || password == "" {
		return errors.New("bootstrap admin: username and password must be set")
	}

	hash, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash bootstrap password: %w", err)
	}

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var userID string
	err = tx.QueryRow(ctx, `
		INSERT INTO users (username, password_hash, full_name, must_change_password, is_active)
		VALUES ($1, $2, $3, true, true)
		RETURNING id`,
		username, hash, "Администратор системы").Scan(&userID)
	if err != nil {
		return fmt.Errorf("insert admin: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO user_roles (user_id, role) VALUES ($1, 'HR_ADMIN')`, userID)
	if err != nil {
		return fmt.Errorf("grant HR_ADMIN: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	log.Warn("bootstrap admin created — must be changed on first login",
		slog.String("username", username))
	return nil
}
