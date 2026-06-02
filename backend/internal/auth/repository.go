package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/google/uuid"
)

var ErrNotFound = errors.New("not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

type userRow struct {
	ID                 uuid.UUID
	PersonnelNumber    *string
	Username           string
	Email              *string
	PasswordHash       string
	FullName           string
	BirthDate          *time.Time
	DepartmentID       *uuid.UUID
	SectionID          *uuid.UUID
	GradeID            *uuid.UUID
	PositionID         *uuid.UUID
	Specialization     *string
	TelegramID         *int64
	HiredAt            *time.Time
	Hobbies            *string
	MustChangePassword bool
	IsActive           bool
}

const userSelect = `
SELECT id, personnel_number, username, email, password_hash, full_name,
       birth_date, department_id, section_id, grade_id, position_id,
       specialization, telegram_id, hired_at, hobbies,
       must_change_password, is_active
FROM users WHERE deleted_at IS NULL`

func scanUser(row pgx.Row) (*userRow, error) {
	u := &userRow{}
	err := row.Scan(
		&u.ID, &u.PersonnelNumber, &u.Username, &u.Email, &u.PasswordHash, &u.FullName,
		&u.BirthDate, &u.DepartmentID, &u.SectionID, &u.GradeID, &u.PositionID,
		&u.Specialization, &u.TelegramID, &u.HiredAt, &u.Hobbies,
		&u.MustChangePassword, &u.IsActive,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (r *Repository) GetUserByUsername(ctx context.Context, username string) (*userRow, error) {
	return scanUser(r.pool.QueryRow(ctx, userSelect+" AND lower(username) = lower($1)", username))
}

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*userRow, error) {
	return scanUser(r.pool.QueryRow(ctx, userSelect+" AND id = $1", id))
}

func (r *Repository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, []uuid.UUID, []uuid.UUID, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT role::text, scope_department_id, scope_section_id FROM user_roles WHERE user_id = $1`, userID)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()
	var roles []string
	var deptIDs []uuid.UUID
	var secIDs []uuid.UUID
	for rows.Next() {
		var role string
		var dept *uuid.UUID
		var sec *uuid.UUID
		if err := rows.Scan(&role, &dept, &sec); err != nil {
			return nil, nil, nil, err
		}
		roles = append(roles, role)
		if dept != nil {
			deptIDs = append(deptIDs, *dept)
		}
		if sec != nil {
			secIDs = append(secIDs, *sec)
		}
	}
	return roles, deptIDs, secIDs, rows.Err()
}

func (r *Repository) UpdatePasswordHash(ctx context.Context, userID uuid.UUID, hash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET password_hash = $1, must_change_password = false WHERE id = $2`,
		hash, userID)
	return err
}

func (r *Repository) InsertRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time, userAgent, ip string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, user_agent, ip)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, '')::inet)`,
		userID, tokenHash, expiresAt, userAgent, ip)
	return err
}

type refreshRow struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
	RevokedAt *time.Time
}

func (r *Repository) GetActiveRefreshToken(ctx context.Context, tokenHash string) (*refreshRow, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, expires_at, revoked_at
		 FROM refresh_tokens WHERE token_hash = $1`, tokenHash)
	rr := &refreshRow{}
	err := row.Scan(&rr.ID, &rr.UserID, &rr.ExpiresAt, &rr.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return rr, err
}

func (r *Repository) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`, id)
	return err
}

func (r *Repository) RevokeAllRefreshTokensForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`,
		userID)
	return err
}

