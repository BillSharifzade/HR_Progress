package onef

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// --- sync run lifecycle ---

func (r *Repository) StartRun(ctx context.Context, trigger string, triggeredBy *uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO onef_sync_runs (trigger_kind, triggered_by, status)
		VALUES ($1, $2, $3)
		RETURNING id`,
		trigger, triggeredBy, StatusRunning,
	).Scan(&id)
	return id, err
}

func (r *Repository) FinishRun(ctx context.Context, id uuid.UUID, res SyncResult, errMsg string) error {
	status := StatusSuccess
	var errPtr *string
	if errMsg != "" {
		status = StatusFailed
		errPtr = &errMsg
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE onef_sync_runs SET
			finished_at = now(),
			status = $2,
			fetched_count = $3,
			created_count = $4,
			updated_count = $5,
			skipped_count = $6,
			error_message = $7,
			duration_ms = $8
		WHERE id = $1`,
		id, status, res.FetchedCount, res.CreatedCount, res.UpdatedCount,
		res.SkippedCount, errPtr, res.DurationMS,
	)
	return err
}

func (r *Repository) ListRuns(ctx context.Context, limit int) ([]SyncRun, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, started_at, finished_at, triggered_by, trigger_kind,
		       status, fetched_count, created_count, updated_count, skipped_count,
		       error_message, duration_ms
		FROM onef_sync_runs
		ORDER BY started_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SyncRun
	for rows.Next() {
		var s SyncRun
		if err := rows.Scan(&s.ID, &s.StartedAt, &s.FinishedAt, &s.TriggeredBy,
			&s.TriggerKind, &s.Status, &s.FetchedCount, &s.CreatedCount,
			&s.UpdatedCount, &s.SkippedCount, &s.ErrorMessage, &s.DurationMS); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// --- dept / section auto-create (lookup by name, create if missing) ---

func (r *Repository) FindOrCreateDepartment(ctx context.Context, name string) (uuid.UUID, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return uuid.Nil, errors.New("empty department name")
	}
	var id uuid.UUID

	// First: explicit alias map (handles real wording differences that
	// normalization can't bridge, e.g. "по Закупкам" vs "Закупки").
	if code := aliasedDeptCode(name); code != "" {
		err := r.pool.QueryRow(ctx, `
			SELECT id FROM departments
			WHERE deleted_at IS NULL AND code = $1`, code,
		).Scan(&id)
		if err == nil {
			return id, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, err
		}
		// Alias targets a code that doesn't exist — fall through to
		// the normalized lookup / auto-create rather than erroring.
	}

	// Second: normalized-name lookup against the seeded depts so that
	// "Финансово Экономический Департамент" (1F) matches "Финансово-Экономический
	// Департамент" (seed). regexp_replace + replace fold hyphens and whitespace.
	normalized := normalizeDeptName(name)
	err := r.pool.QueryRow(ctx, `
		SELECT id FROM departments
		WHERE deleted_at IS NULL
		  AND regexp_replace(lower(replace(name, '-', ' ')), '\s+', ' ', 'g') = $1`,
		normalized,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}

	code, err := r.uniqueDeptCode(ctx, initials(name))
	if err != nil {
		return uuid.Nil, err
	}
	err = r.pool.QueryRow(ctx, `
		INSERT INTO departments (code, name)
		VALUES ($1, $2)
		RETURNING id`,
		code, name,
	).Scan(&id)
	return id, err
}

func (r *Repository) uniqueDeptCode(ctx context.Context, base string) (string, error) {
	if base == "" {
		base = "DEPT"
	}
	for i := 0; i < 50; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s%d", base, i+1)
		}
		var exists bool
		if err := r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM departments WHERE code = $1 AND deleted_at IS NULL)`,
			candidate,
		).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", errors.New("could not allocate unique dept code")
}

func (r *Repository) FindOrCreateSection(ctx context.Context, deptID uuid.UUID, name string) (uuid.UUID, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return uuid.Nil, errors.New("empty section name")
	}
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT id FROM sections
		WHERE deleted_at IS NULL AND department_id = $1 AND lower(name) = lower($2)`,
		deptID, name,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}

	code, err := r.uniqueSectionCode(ctx, deptID, initials(name))
	if err != nil {
		return uuid.Nil, err
	}
	err = r.pool.QueryRow(ctx, `
		INSERT INTO sections (department_id, code, name)
		VALUES ($1, $2, $3)
		RETURNING id`,
		deptID, code, name,
	).Scan(&id)
	return id, err
}

func (r *Repository) uniqueSectionCode(ctx context.Context, deptID uuid.UUID, base string) (string, error) {
	if base == "" {
		base = "SEC"
	}
	for i := 0; i < 50; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s%d", base, i+1)
		}
		var exists bool
		if err := r.pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM sections
				WHERE department_id = $1 AND lower(code) = lower($2) AND deleted_at IS NULL
			)`, deptID, candidate,
		).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", errors.New("could not allocate unique section code")
}

// --- grade lookup ---

func (r *Repository) GradeIDByLevel(ctx context.Context, level int) (*uuid.UUID, error) {
	if level <= 0 {
		return nil, nil
	}
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT id FROM grades WHERE level = $1 AND deleted_at IS NULL`,
		level,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// --- worker upsert (matched by one_f_user_id) ---

// UserMatch is the local user matched against a 1F UserID.
type UserMatch struct {
	ID       uuid.UUID
	Username string
}

func (r *Repository) FindByOneFUserID(ctx context.Context, oneFUserID int64) (*UserMatch, error) {
	var m UserMatch
	err := r.pool.QueryRow(ctx, `
		SELECT id, username FROM users
		WHERE deleted_at IS NULL AND one_f_user_id = $1`,
		oneFUserID,
	).Scan(&m.ID, &m.Username)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// UsernameExists is used by callers that build candidate usernames.
func (r *Repository) UsernameExists(ctx context.Context, username string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 AND deleted_at IS NULL)`,
		username,
	).Scan(&exists)
	return exists, err
}

// CreateWorkerInput is what InsertWorker needs to materialize a row.
type CreateWorkerInput struct {
	Username           string
	PasswordHash       string
	FullName           string
	Email              *string
	PhoneNumber        *string
	BirthDate          *time.Time
	HiredAt            *time.Time
	DepartmentID       *uuid.UUID
	SectionID          *uuid.UUID
	GradeID            *uuid.UUID
	Position           *string
	OneFUserID         int64
	OneFManagerUserID  *int64
	OneFIsManager      bool
}

func (r *Repository) InsertWorker(ctx context.Context, in CreateWorkerInput) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (
			username, password_hash, full_name, email, phone_number,
			birth_date, hired_at,
			department_id, section_id, grade_id, position,
			one_f_user_id, one_f_manager_user_id, one_f_is_manager,
			last_synced_at, must_change_password, is_active
		) VALUES (
			$1,$2,$3,$4,$5,
			$6,$7,
			$8,$9,$10,$11,
			$12,$13,$14,
			now(), true, true
		)
		RETURNING id`,
		in.Username, in.PasswordHash, in.FullName, in.Email, in.PhoneNumber,
		in.BirthDate, in.HiredAt,
		in.DepartmentID, in.SectionID, in.GradeID, in.Position,
		in.OneFUserID, in.OneFManagerUserID, in.OneFIsManager,
	).Scan(&id)
	return id, err
}

// UpdateWorkerInput is the subset of fields 1F is allowed to overwrite.
type UpdateWorkerInput struct {
	FullName          string
	Email             *string
	PhoneNumber       *string
	BirthDate         *time.Time
	HiredAt           *time.Time
	DepartmentID      *uuid.UUID
	SectionID         *uuid.UUID
	Position          *string
	OneFManagerUserID *int64
	OneFIsManager     bool
	GradeID           *uuid.UUID // overwritten only when current grade is NULL (preserve HR-set grades)
}

func (r *Repository) UpdateWorker(ctx context.Context, id uuid.UUID, in UpdateWorkerInput) error {
	// Grade is the only field 1F owns "conditionally": only fill if local is NULL,
	// so an HR admin's manual grade assignment survives subsequent polls.
	_, err := r.pool.Exec(ctx, `
		UPDATE users SET
			full_name             = $2,
			email                 = $3,
			phone_number          = $4,
			birth_date            = $5,
			hired_at              = $6,
			department_id         = $7,
			section_id            = $8,
			position              = $9,
			one_f_manager_user_id = $10,
			one_f_is_manager      = $11,
			grade_id              = COALESCE(grade_id, $12),
			last_synced_at        = now()
		WHERE id = $1 AND deleted_at IS NULL`,
		id,
		in.FullName, in.Email, in.PhoneNumber, in.BirthDate, in.HiredAt,
		in.DepartmentID, in.SectionID, in.Position,
		in.OneFManagerUserID, in.OneFIsManager, in.GradeID,
	)
	return err
}

// --- role grants (idempotent) ---

func (r *Repository) GrantSectionHead(ctx context.Context, userID uuid.UUID, sectionID, deptID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_roles (user_id, role, scope_section_id, scope_department_id)
		VALUES ($1, 'SECTION_HEAD', $2, $3)
		ON CONFLICT (user_id, role,
		             COALESCE(scope_department_id, '00000000-0000-0000-0000-000000000000'::uuid),
		             COALESCE(scope_section_id,    '00000000-0000-0000-0000-000000000000'::uuid))
		DO NOTHING`,
		userID, sectionID, deptID,
	)
	return err
}

func (r *Repository) GrantDeptHead(ctx context.Context, userID uuid.UUID, deptID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_roles (user_id, role, scope_department_id)
		VALUES ($1, 'DEPT_HEAD', $2)
		ON CONFLICT (user_id, role,
		             COALESCE(scope_department_id, '00000000-0000-0000-0000-000000000000'::uuid),
		             COALESCE(scope_section_id,    '00000000-0000-0000-0000-000000000000'::uuid))
		DO NOTHING`,
		userID, deptID,
	)
	return err
}

// --- helpers ---

// initials returns the first letter of each whitespace- or hyphen-separated
// word, uppercased. Mirrors competency.initials / workers.sectionInitials
// (duplicated locally to avoid cross-package coupling).
func initials(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return unicode.IsSpace(r) || r == '-'
	})
	var b strings.Builder
	for _, p := range parts {
		for _, r := range p {
			b.WriteRune(unicode.ToUpper(r))
			break
		}
	}
	return b.String()
}
