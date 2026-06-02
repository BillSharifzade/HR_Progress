package competency

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListDepartments(ctx context.Context) ([]Department, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, code, name, COALESCE(description,''), is_active
		FROM departments
		WHERE deleted_at IS NULL AND is_active = true
		ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Department
	for rows.Next() {
		var d Department
		if err := rows.Scan(&d.ID, &d.Code, &d.Name, &d.Description, &d.IsActive); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repository) ListCompetencies(ctx context.Context) ([]Competency, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, code, kind, name, description, why_important, sort_order, is_active, created_at, updated_at
		FROM competencies
		WHERE deleted_at IS NULL AND is_active = true
		ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Competency
	for rows.Next() {
		var c Competency
		if err := rows.Scan(&c.ID, &c.Code, &c.Kind, &c.Name, &c.Description, &c.WhyImportant,
			&c.SortOrder, &c.IsActive, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) ListRequirements(ctx context.Context, deptID uuid.UUID) ([]Requirement, error) {
	rows, err := r.db.Query(ctx, `
		SELECT dcr.id, dcr.department_id, dcr.competency_id,
		       c.code, c.name, c.kind,
		       dcr.grade_id, g.level, g.name,
		       dcr.required_min, dcr.is_key, dcr.description
		FROM dept_competency_requirements dcr
		JOIN competencies c ON c.id = dcr.competency_id
		JOIN grades g ON g.id = dcr.grade_id
		WHERE dcr.department_id = $1
		ORDER BY c.sort_order, g.level`, deptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Requirement
	for rows.Next() {
		var req Requirement
		if err := rows.Scan(
			&req.ID, &req.DepartmentID, &req.CompetencyID,
			&req.CompetencyCode, &req.CompetencyName, &req.CompetencyKind,
			&req.GradeID, &req.GradeLevel, &req.GradeName,
			&req.RequiredMin, &req.IsKey, &req.Description,
		); err != nil {
			return nil, err
		}
		out = append(out, req)
	}
	return out, rows.Err()
}

func (r *Repository) UpsertRequirements(ctx context.Context, deptID uuid.UUID, reqs []UpsertRequirementRequest) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM dept_competency_requirements WHERE department_id = $1`, deptID); err != nil {
		return err
	}
	for _, req := range reqs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO dept_competency_requirements
				(department_id, competency_id, grade_id, required_min, is_key, description)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			deptID, req.CompetencyID, req.GradeID, req.RequiredMin, req.IsKey, req.Description); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) ListEmployees(ctx context.Context, deptID uuid.UUID) ([]Employee, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id, u.full_name, u.grade_id, g.name, g.level, u.section_id
		FROM users u
		LEFT JOIN grades g ON g.id = u.grade_id
		WHERE u.department_id = $1 AND u.is_active = true AND u.deleted_at IS NULL
		ORDER BY g.level NULLS LAST, u.full_name`, deptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Employee
	for rows.Next() {
		var e Employee
		if err := rows.Scan(&e.ID, &e.FullName, &e.GradeID, &e.GradeName, &e.GradeLevel, &e.SectionID); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) BulkUpsertScores(ctx context.Context, periodID uuid.UUID, reqs []UpsertScoreRequest, assessedBy uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	now := time.Now()
	for _, req := range reqs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO assessment_scores
				(period_id, employee_id, competency_id, assessor_role, score, feedback, assessed_by, assessed_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (period_id, employee_id, competency_id, assessor_role) DO UPDATE SET
				score       = EXCLUDED.score,
				feedback    = EXCLUDED.feedback,
				assessed_by = EXCLUDED.assessed_by,
				assessed_at = EXCLUDED.assessed_at,
				updated_at  = now()`,
			periodID, req.EmployeeID, req.CompetencyID, req.AssessorRole,
			req.Score, req.Feedback, assessedBy, now); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) ListAllDepartments(ctx context.Context) ([]Department, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, code, name, COALESCE(description,''), is_active
		FROM departments
		WHERE deleted_at IS NULL
		ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Department
	for rows.Next() {
		var d Department
		if err := rows.Scan(&d.ID, &d.Code, &d.Name, &d.Description, &d.IsActive); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repository) ListGrades(ctx context.Context) ([]Grade, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, level, description, is_active
		FROM grades
		WHERE deleted_at IS NULL AND is_active = true
		ORDER BY level`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Grade
	for rows.Next() {
		var g Grade
		if err := rows.Scan(&g.ID, &g.Name, &g.Level, &g.Description, &g.IsActive); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *Repository) CreateDepartment(ctx context.Context, code, name string, description *string) (Department, error) {
	var d Department
	err := r.db.QueryRow(ctx, `
		INSERT INTO departments (code, name, description)
		VALUES ($1, $2, $3)
		RETURNING id, code, name, COALESCE(description,''), is_active`,
		code, name, description,
	).Scan(&d.ID, &d.Code, &d.Name, &d.Description, &d.IsActive)
	return d, err
}

func (r *Repository) UpdateDepartment(ctx context.Context, id uuid.UUID, req UpdateDepartmentRequest) (Department, error) {
	var d Department
	err := r.db.QueryRow(ctx, `
		UPDATE departments SET name = $2, description = $3, is_active = $4, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, code, name, COALESCE(description,''), is_active`,
		id, req.Name, req.Description, req.IsActive,
	).Scan(&d.ID, &d.Code, &d.Name, &d.Description, &d.IsActive)
	if err != nil && err.Error() == "no rows in result set" {
		return Department{}, ErrNotFound
	}
	return d, err
}

func (r *Repository) DepartmentCodeExists(ctx context.Context, code string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM departments WHERE code = $1 AND deleted_at IS NULL)`, code,
	).Scan(&exists)
	return exists, err
}

func (r *Repository) DeleteDepartment(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE departments SET deleted_at = now(), is_active = false
		WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) ListPeriods(ctx context.Context, deptID *uuid.UUID) ([]Period, error) {
	var rows interface{ Next() bool; Scan(...any) error; Err() error; Close() }
	var err error

	if deptID != nil {
		rows, err = r.db.Query(ctx, `
			SELECT id, title, department_id, period_start, period_end, is_active, created_by, created_at, updated_at
			FROM assessment_periods
			WHERE department_id = $1
			ORDER BY period_start DESC`, *deptID)
	} else {
		rows, err = r.db.Query(ctx, `
			SELECT id, title, department_id, period_start, period_end, is_active, created_by, created_at, updated_at
			FROM assessment_periods
			ORDER BY period_start DESC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Period
	for rows.Next() {
		var p Period
		if err := rows.Scan(&p.ID, &p.Title, &p.DepartmentID, &p.PeriodStart, &p.PeriodEnd,
			&p.IsActive, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) CreatePeriod(ctx context.Context, p Period) (Period, error) {
	err := r.db.QueryRow(ctx, `
		INSERT INTO assessment_periods (title, department_id, period_start, period_end, is_active, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, title, department_id, period_start, period_end, is_active, created_by, created_at, updated_at`,
		p.Title, p.DepartmentID, p.PeriodStart, p.PeriodEnd, p.IsActive, p.CreatedBy,
	).Scan(&p.ID, &p.Title, &p.DepartmentID, &p.PeriodStart, &p.PeriodEnd, &p.IsActive, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (r *Repository) GetPeriod(ctx context.Context, id uuid.UUID) (Period, error) {
	var p Period
	err := r.db.QueryRow(ctx, `
		SELECT id, title, department_id, period_start, period_end, is_active, created_by, created_at, updated_at
		FROM assessment_periods WHERE id = $1`, id,
	).Scan(&p.ID, &p.Title, &p.DepartmentID, &p.PeriodStart, &p.PeriodEnd, &p.IsActive, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (r *Repository) ListScores(ctx context.Context, periodID uuid.UUID) ([]Score, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, period_id, employee_id, competency_id, assessor_role,
		       score, feedback, assessed_by, assessed_at, updated_at
		FROM assessment_scores
		WHERE period_id = $1
		ORDER BY employee_id, competency_id, assessor_role`, periodID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Score, 0)
	for rows.Next() {
		var s Score
		if err := rows.Scan(&s.ID, &s.PeriodID, &s.EmployeeID, &s.CompetencyID, &s.AssessorRole,
			&s.Score, &s.Feedback, &s.AssessedBy, &s.AssessedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) CreateCompetency(ctx context.Context, code string, req CreateCompetencyRequest, sortOrder int) (Competency, error) {
	var c Competency
	err := r.db.QueryRow(ctx, `
		INSERT INTO competencies (code, kind, name, description, why_important, sort_order, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, true)
		RETURNING id, code, kind, name, description, why_important, sort_order, is_active, created_at, updated_at`,
		code, req.Kind, req.Name, req.Description, req.WhyImportant, sortOrder,
	).Scan(&c.ID, &c.Code, &c.Kind, &c.Name, &c.Description, &c.WhyImportant, &c.SortOrder, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (r *Repository) UpdateCompetency(ctx context.Context, id uuid.UUID, req UpdateCompetencyRequest) (Competency, error) {
	var c Competency
	err := r.db.QueryRow(ctx, `
		UPDATE competencies SET
			kind = $2, name = $3, description = $4,
			why_important = $5, is_active = $6, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, code, kind, name, description, why_important, sort_order, is_active, created_at, updated_at`,
		id, req.Kind, req.Name, req.Description, req.WhyImportant, req.IsActive,
	).Scan(&c.ID, &c.Code, &c.Kind, &c.Name, &c.Description, &c.WhyImportant, &c.SortOrder, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	if err != nil && err.Error() == "no rows in result set" {
		return Competency{}, ErrNotFound
	}
	return c, err
}

func (r *Repository) CompetencyCodeExists(ctx context.Context, code string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM competencies WHERE code = $1 AND deleted_at IS NULL)`, code,
	).Scan(&exists)
	return exists, err
}

func (r *Repository) MaxCompetencySortOrder(ctx context.Context) (int, error) {
	var n *int
	err := r.db.QueryRow(ctx,
		`SELECT MAX(sort_order) FROM competencies WHERE deleted_at IS NULL`,
	).Scan(&n)
	if err != nil {
		return 0, err
	}
	if n == nil {
		return 0, nil
	}
	return *n, nil
}

func (r *Repository) ReorderCompetencies(ctx context.Context, ids []uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for i, id := range ids {
		if _, err := tx.Exec(ctx,
			`UPDATE competencies SET sort_order = $2, updated_at = now()
			 WHERE id = $1 AND deleted_at IS NULL`, id, (i+1)*10,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) DeleteCompetency(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE competencies SET deleted_at = now(), is_active = false
		WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) UpsertScore(ctx context.Context, periodID uuid.UUID, req UpsertScoreRequest, assessedBy uuid.UUID) (Score, error) {
	now := time.Now()
	var s Score
	err := r.db.QueryRow(ctx, `
		INSERT INTO assessment_scores
			(period_id, employee_id, competency_id, assessor_role, score, feedback, assessed_by, assessed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (period_id, employee_id, competency_id, assessor_role) DO UPDATE SET
			score       = EXCLUDED.score,
			feedback    = EXCLUDED.feedback,
			assessed_by = EXCLUDED.assessed_by,
			assessed_at = EXCLUDED.assessed_at,
			updated_at  = now()
		RETURNING id, period_id, employee_id, competency_id, assessor_role,
		          score, feedback, assessed_by, assessed_at, updated_at`,
		periodID, req.EmployeeID, req.CompetencyID, req.AssessorRole,
		req.Score, req.Feedback, assessedBy, now,
	).Scan(&s.ID, &s.PeriodID, &s.EmployeeID, &s.CompetencyID, &s.AssessorRole,
		&s.Score, &s.Feedback, &s.AssessedBy, &s.AssessedAt, &s.UpdatedAt)
	return s, err
}
