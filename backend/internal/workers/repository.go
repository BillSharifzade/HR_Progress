package workers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// --- Workers ---

const workerDetailSelect = `
SELECT
    u.id, u.username, u.employee_no, u.personnel_number,
    u.one_f_user_id, u.one_f_is_manager, u.phone_number, u.last_synced_at,
    u.full_name, u.email, u.birth_date,
    u.department_id, d.name,
    u.section_id,   s.name,
    u.grade_id,     g.name, g.level,
    u.position_id,  p.name, u.position,
    u.specialization, u.telegram_id, u.hired_at, u.hobbies, u.is_active
FROM users u
LEFT JOIN departments d ON d.id = u.department_id AND d.deleted_at IS NULL
LEFT JOIN sections    s ON s.id = u.section_id    AND s.deleted_at IS NULL
LEFT JOIN grades      g ON g.id = u.grade_id      AND g.deleted_at IS NULL
LEFT JOIN positions   p ON p.id = u.position_id   AND p.deleted_at IS NULL
WHERE u.deleted_at IS NULL`

func scanWorker(row pgx.Row) (*Worker, error) {
	w := &Worker{Roles: []string{}}
	err := row.Scan(
		&w.ID, &w.Username, &w.EmployeeNo, &w.PersonnelNumber,
		&w.OneFUserID, &w.OneFIsManager, &w.PhoneNumber, &w.LastSyncedAt,
		&w.FullName, &w.Email, &w.BirthDate,
		&w.DepartmentID, &w.DepartmentName,
		&w.SectionID, &w.SectionName,
		&w.GradeID, &w.GradeName, &w.GradeLevel,
		&w.PositionID, &w.PositionName, &w.Position,
		&w.Specialization, &w.TelegramID, &w.HiredAt, &w.Hobbies, &w.IsActive,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return w, err
}

func (r *Repository) List(ctx context.Context, f ListFilter) ([]WorkerSummary, error) {
	conds := []string{"u.deleted_at IS NULL"}
	args := []any{}
	idx := 1

	if !f.IncludeInactive {
		conds = append(conds, fmt.Sprintf("u.is_active = $%d", idx))
		args = append(args, true)
		idx++
	}
	if f.DepartmentID != nil {
		conds = append(conds, fmt.Sprintf("u.department_id = $%d", idx))
		args = append(args, *f.DepartmentID)
		idx++
	}
	if f.SectionID != nil {
		conds = append(conds, fmt.Sprintf("u.section_id = $%d", idx))
		args = append(args, *f.SectionID)
		idx++
	}
	if f.GradeID != nil {
		conds = append(conds, fmt.Sprintf("u.grade_id = $%d", idx))
		args = append(args, *f.GradeID)
		idx++
	}
	if f.Search != "" {
		conds = append(conds, fmt.Sprintf(`(
			lower(u.full_name) ILIKE $%d OR
			lower(COALESCE(u.personnel_number,'')) ILIKE $%d OR
			lower(COALESCE(u.email,'')) ILIKE $%d OR
			COALESCE(u.telegram_id::text,'') ILIKE $%d
		)`, idx, idx, idx, idx))
		args = append(args, "%"+f.Search+"%")
		idx++
	}
	_ = idx

	q := `
	SELECT u.id, u.employee_no, u.personnel_number, u.one_f_user_id, u.full_name,
	       d.name, s.name, g.name, g.level, u.position, u.hired_at, u.is_active
	FROM users u
	LEFT JOIN departments d ON d.id = u.department_id AND d.deleted_at IS NULL
	LEFT JOIN sections    s ON s.id = u.section_id    AND s.deleted_at IS NULL
	LEFT JOIN grades      g ON g.id = u.grade_id      AND g.deleted_at IS NULL
	WHERE ` + strings.Join(conds, " AND ") + `
	ORDER BY u.full_name`

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []WorkerSummary
	for rows.Next() {
		var w WorkerSummary
		if err := rows.Scan(
			&w.ID, &w.EmployeeNo, &w.PersonnelNumber, &w.OneFUserID, &w.FullName,
			&w.DepartmentName, &w.SectionName, &w.GradeName, &w.GradeLevel,
			&w.PositionName, &w.HiredAt, &w.IsActive,
		); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*Worker, error) {
	w, err := scanWorker(r.pool.QueryRow(ctx, workerDetailSelect+" AND u.id = $1", id))
	if err != nil {
		return nil, err
	}
	roleRows, err := r.pool.Query(ctx,
		`SELECT role::text FROM user_roles WHERE user_id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer roleRows.Close()
	for roleRows.Next() {
		var role string
		if err := roleRows.Scan(&role); err != nil {
			return nil, err
		}
		w.Roles = append(w.Roles, role)
	}
	return w, roleRows.Err()
}

// --- History ---

func (r *Repository) ListHistory(ctx context.Context, userID uuid.UUID) ([]History, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, event_kind::text, event_date, title, description, meta, created_at
		FROM user_history WHERE user_id = $1 ORDER BY event_date DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []History
	for rows.Next() {
		var h History
		if err := rows.Scan(&h.ID, &h.UserID, &h.EventKind, &h.EventDate,
			&h.Title, &h.Description, &h.Meta, &h.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *Repository) CreateHistory(ctx context.Context, userID uuid.UUID, req CreateHistoryRequest, createdBy uuid.UUID) (*History, error) {
	eventDate, err := time.Parse("2006-01-02", req.EventDate)
	if err != nil {
		return nil, err
	}
	h := &History{}
	err = r.pool.QueryRow(ctx, `
		INSERT INTO user_history (user_id, event_kind, event_date, title, description, meta, created_by)
		VALUES ($1, $2::history_event_kind, $3, $4, $5, $6, $7)
		RETURNING id, user_id, event_kind::text, event_date, title, description, meta, created_at`,
		userID, req.EventKind, eventDate, req.Title, req.Description, req.Meta, createdBy,
	).Scan(&h.ID, &h.UserID, &h.EventKind, &h.EventDate,
		&h.Title, &h.Description, &h.Meta, &h.CreatedAt)
	return h, err
}

// --- Certifications ---

func (r *Repository) ListCertifications(ctx context.Context, userID uuid.UUID) ([]Certification, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, title, issued_by, issued_at, expires_at, created_at, updated_at
		FROM user_certifications WHERE user_id = $1 ORDER BY issued_at DESC NULLS LAST`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Certification
	for rows.Next() {
		var c Certification
		if err := rows.Scan(&c.ID, &c.UserID, &c.Title, &c.IssuedBy,
			&c.IssuedAt, &c.ExpiresAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) CreateCertification(ctx context.Context, userID uuid.UUID, req UpsertCertificationRequest) (*Certification, error) {
	var issuedAt, expiresAt *time.Time
	if req.IssuedAt != nil {
		t, err := time.Parse("2006-01-02", *req.IssuedAt)
		if err != nil {
			return nil, err
		}
		issuedAt = &t
	}
	if req.ExpiresAt != nil {
		t, err := time.Parse("2006-01-02", *req.ExpiresAt)
		if err != nil {
			return nil, err
		}
		expiresAt = &t
	}
	c := &Certification{}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO user_certifications (user_id, title, issued_by, issued_at, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, title, issued_by, issued_at, expires_at, created_at, updated_at`,
		userID, req.Title, req.IssuedBy, issuedAt, expiresAt,
	).Scan(&c.ID, &c.UserID, &c.Title, &c.IssuedBy,
		&c.IssuedAt, &c.ExpiresAt, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (r *Repository) DeleteCertification(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM user_certifications WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Positions ---

func (r *Repository) ListPositions(ctx context.Context) ([]Position, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, is_active FROM positions WHERE deleted_at IS NULL ORDER BY lower(name)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Position
	for rows.Next() {
		var p Position
		if err := rows.Scan(&p.ID, &p.Name, &p.IsActive); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) CreatePosition(ctx context.Context, name string) (*Position, error) {
	p := &Position{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO positions (name) VALUES ($1) RETURNING id, name, is_active`, name,
	).Scan(&p.ID, &p.Name, &p.IsActive)
	return p, err
}

// --- Sections ---

func (r *Repository) ListSections(ctx context.Context, deptID *uuid.UUID) ([]Section, error) {
	q := `SELECT id, department_id, code, name, description, is_active FROM sections WHERE deleted_at IS NULL`
	args := []any{}
	if deptID != nil {
		q += ` AND department_id = $1`
		args = append(args, *deptID)
	}
	q += ` ORDER BY name`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Section
	for rows.Next() {
		var s Section
		if err := rows.Scan(&s.ID, &s.DepartmentID, &s.Code, &s.Name, &s.Description, &s.IsActive); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) CreateSection(ctx context.Context, req CreateSectionRequest) (*Section, error) {
	code := req.Code
	if code == nil || *code == "" {
		derived := sectionInitials(req.Name)
		if derived != "" {
			code = &derived
		} else {
			code = nil
		}
	}
	s := &Section{}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO sections (department_id, code, name, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id, department_id, code, name, description, is_active`,
		req.DepartmentID, code, req.Name, req.Description,
	).Scan(&s.ID, &s.DepartmentID, &s.Code, &s.Name, &s.Description, &s.IsActive)
	return s, err
}

func (r *Repository) UpdateSection(ctx context.Context, id uuid.UUID, req UpdateSectionRequest) (*Section, error) {
	s := &Section{}
	err := r.pool.QueryRow(ctx, `
		UPDATE sections SET name=$2, description=$3, is_active=$4, updated_at=now()
		WHERE id=$1 AND deleted_at IS NULL
		RETURNING id, department_id, code, name, description, is_active`,
		id, req.Name, req.Description, req.IsActive,
	).Scan(&s.ID, &s.DepartmentID, &s.Code, &s.Name, &s.Description, &s.IsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return s, err
}

func (r *Repository) DeleteSection(ctx context.Context, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE sections SET deleted_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Worker CRUD ---

func parseDate(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) UsernameExists(ctx context.Context, username string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 AND deleted_at IS NULL)`,
		username,
	).Scan(&exists)
	return exists, err
}

// UniqueUsername returns base + "-NNNN" where NNNN is a crypto-random 4-digit
// suffix that is not already taken. The suffix is always present, so each user
// gets a non-guessable username even when two people share a name. Retries on
// the rare collision; bails after 64 attempts.
func (r *Repository) UniqueUsername(ctx context.Context, base string) (string, error) {
	if base == "" {
		base = "user"
	}
	for attempt := 0; attempt < 64; attempt++ {
		suffix, err := randomSuffix()
		if err != nil {
			return "", err
		}
		candidate := base + "-" + suffix
		exists, err := r.UsernameExists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", errors.New("could not generate unique username after 64 attempts")
}

func (r *Repository) Create(ctx context.Context, req CreateWorkerRequest, passwordHash string) (*Worker, error) {
	birthDate, err := parseDate(req.BirthDate)
	if err != nil {
		return nil, err
	}
	hiredAt, err := parseDate(req.HiredAt)
	if err != nil {
		return nil, err
	}
	username := req.Username
	if username == "" {
		username, err = r.UniqueUsername(ctx, usernameBase(req.FullName))
		if err != nil {
			return nil, err
		}
	}
	var id uuid.UUID
	err = r.pool.QueryRow(ctx, `
		INSERT INTO users (
			username, password_hash, full_name, email, personnel_number,
			birth_date, department_id, section_id, grade_id, position,
			specialization, telegram_id, hired_at, hobbies,
			must_change_password, is_active
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,true,true)
		RETURNING id`,
		username, passwordHash, req.FullName, req.Email, req.PersonnelNumber,
		birthDate, req.DepartmentID, req.SectionID, req.GradeID, req.Position,
		req.Specialization, req.TelegramID, hiredAt, req.Hobbies,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	if err := r.AssignDerivedRoles(ctx, id); err != nil {
		return nil, fmt.Errorf("derive roles: %w", err)
	}
	return r.Get(ctx, id)
}

// AssignDerivedRoles grants org-structure roles to a user based on their
// current grade + scope:
//   - grade level 5 (Руководитель Отдела) + section_id → SECTION_HEAD scoped to section
//   - grade level 7 (Руководитель Департамента) + department_id → DEPT_HEAD scoped to dept
//
// HR head (DCR_HEAD) is just the DEPT_HEAD of the ДЧР department — same role,
// no special handling required. Idempotent (ON CONFLICT DO NOTHING).
func (r *Repository) AssignDerivedRoles(ctx context.Context, userID uuid.UUID) error {
	var (
		gradeLevel *int
		sectionID  *uuid.UUID
		deptID     *uuid.UUID
	)
	err := r.pool.QueryRow(ctx, `
		SELECT g.level, u.section_id, u.department_id
		  FROM users u
		  LEFT JOIN grades g ON g.id = u.grade_id
		 WHERE u.id = $1 AND u.deleted_at IS NULL`, userID,
	).Scan(&gradeLevel, &sectionID, &deptID)
	if err != nil {
		return err
	}
	if gradeLevel == nil {
		return nil
	}
	switch *gradeLevel {
	case 5:
		if sectionID == nil {
			return nil
		}
		_, err = r.pool.Exec(ctx, `
			INSERT INTO user_roles (user_id, role, scope_section_id, scope_department_id)
			VALUES ($1, 'SECTION_HEAD', $2, $3)
			ON CONFLICT (user_id, role,
			             COALESCE(scope_department_id, '00000000-0000-0000-0000-000000000000'::uuid),
			             COALESCE(scope_section_id,    '00000000-0000-0000-0000-000000000000'::uuid))
			DO NOTHING`, userID, *sectionID, deptID)
		return err
	case 7:
		if deptID == nil {
			return nil
		}
		_, err = r.pool.Exec(ctx, `
			INSERT INTO user_roles (user_id, role, scope_department_id)
			VALUES ($1, 'DEPT_HEAD', $2)
			ON CONFLICT (user_id, role,
			             COALESCE(scope_department_id, '00000000-0000-0000-0000-000000000000'::uuid),
			             COALESCE(scope_section_id,    '00000000-0000-0000-0000-000000000000'::uuid))
			DO NOTHING`, userID, *deptID)
		return err
	}
	return nil
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, req UpdateWorkerRequest) (*Worker, error) {
	birthDate, err := parseDate(req.BirthDate)
	if err != nil {
		return nil, err
	}
	hiredAt, err := parseDate(req.HiredAt)
	if err != nil {
		return nil, err
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE users SET
			full_name=$1, email=$2, personnel_number=$3, birth_date=$4,
			department_id=$5, section_id=$6, grade_id=$7, position=$8,
			specialization=$9, telegram_id=$10, hired_at=$11, hobbies=$12
		WHERE id=$13 AND deleted_at IS NULL`,
		req.FullName, req.Email, req.PersonnelNumber, birthDate,
		req.DepartmentID, req.SectionID, req.GradeID, req.Position,
		req.Specialization, req.TelegramID, hiredAt, req.Hobbies,
		id,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	if err := r.AssignDerivedRoles(ctx, id); err != nil {
		return nil, fmt.Errorf("derive roles: %w", err)
	}
	return r.Get(ctx, id)
}

// --- Role assignments ---

var ErrInvalidScope = errors.New("invalid scope for role")
var ErrRoleExists = errors.New("role already granted for that scope")

const roleAssignmentSelect = `
SELECT ur.id, ur.user_id, ur.role::text,
       ur.scope_department_id, d.name,
       ur.scope_section_id,    s.name,
       ur.granted_by, gb.full_name,
       ur.granted_at
FROM user_roles ur
LEFT JOIN departments d ON d.id = ur.scope_department_id
LEFT JOIN sections    s ON s.id = ur.scope_section_id
LEFT JOIN users      gb ON gb.id = ur.granted_by`

func scanRoleAssignment(row pgx.Row) (RoleAssignment, error) {
	var a RoleAssignment
	err := row.Scan(
		&a.ID, &a.UserID, &a.Role,
		&a.ScopeDepartmentID, &a.ScopeDepartment,
		&a.ScopeSectionID, &a.ScopeSection,
		&a.GrantedByID, &a.GrantedByName,
		&a.GrantedAt,
	)
	return a, err
}

func (r *Repository) ListRoleAssignments(ctx context.Context, userID uuid.UUID) ([]RoleAssignment, error) {
	rows, err := r.pool.Query(ctx,
		roleAssignmentSelect+" WHERE ur.user_id = $1 ORDER BY ur.granted_at DESC", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]RoleAssignment, 0)
	for rows.Next() {
		a, err := scanRoleAssignment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repository) GrantRole(ctx context.Context, userID uuid.UUID, req GrantRoleRequest, grantedBy uuid.UUID) (RoleAssignment, error) {
	if err := validateRoleScope(req); err != nil {
		return RoleAssignment{}, err
	}
	// If a section is given, derive the department from it to keep them consistent.
	deptID := req.ScopeDepartmentID
	if req.ScopeSectionID != nil {
		var sectDept uuid.UUID
		if err := r.pool.QueryRow(ctx,
			`SELECT department_id FROM sections WHERE id = $1 AND deleted_at IS NULL`,
			*req.ScopeSectionID,
		).Scan(&sectDept); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return RoleAssignment{}, ErrInvalidScope
			}
			return RoleAssignment{}, err
		}
		deptID = &sectDept
	}
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO user_roles (user_id, role, scope_department_id, scope_section_id, granted_by)
		VALUES ($1, $2::role_kind, $3, $4, $5)
		RETURNING id`,
		userID, req.Role, deptID, req.ScopeSectionID, grantedBy,
	).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "user_roles_uniq") {
			return RoleAssignment{}, ErrRoleExists
		}
		return RoleAssignment{}, err
	}
	return scanRoleAssignment(r.pool.QueryRow(ctx, roleAssignmentSelect+" WHERE ur.id = $1", id))
}

func (r *Repository) RevokeRole(ctx context.Context, userID, assignmentID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM user_roles WHERE id = $1 AND user_id = $2`, assignmentID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func validateRoleScope(req GrantRoleRequest) error {
	switch req.Role {
	case "DEPT_HEAD":
		if req.ScopeDepartmentID == nil || req.ScopeSectionID != nil {
			return ErrInvalidScope
		}
	case "SECTION_HEAD":
		if req.ScopeSectionID == nil {
			return ErrInvalidScope
		}
	case "HR_ADMIN", "ASSESSOR", "PRECEPTOR", "ATS", "BOOK_SPACE":
		if req.ScopeDepartmentID != nil || req.ScopeSectionID != nil {
			return ErrInvalidScope
		}
	}
	return nil
}

// ResetPassword writes a new password hash for the user. Only admins can
// invoke this path; end users cannot change their own password, so we don't
// set must_change_password.
// Returns the user's username so the admin can pass it back to the worker.
func (r *Repository) ResetPassword(ctx context.Context, userID uuid.UUID, hash string) (string, error) {
	var username string
	err := r.pool.QueryRow(ctx, `
		UPDATE users
		   SET password_hash = $1, must_change_password = false
		 WHERE id = $2 AND deleted_at IS NULL
		RETURNING username`, hash, userID).Scan(&username)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return username, err
}

func (r *Repository) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE users SET is_active=$1 WHERE id=$2 AND deleted_at IS NULL`, active, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
