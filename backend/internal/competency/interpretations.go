package competency

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrInterpretationNotFound = errors.New("interpretation not configured")

// LookupInterpretation finds the active interpretation text for the
// (department → grade → competency → score) combination (FR-AS7.2.1).
func (r *Repository) LookupInterpretation(ctx context.Context, deptID, gradeID, compID uuid.UUID, score int) (string, bool, error) {
	var text string
	err := r.db.QueryRow(ctx, `
		SELECT text FROM assessment_interpretations
		WHERE department_id = $1 AND grade_id = $2 AND competency_id = $3 AND score = $4 AND is_active
		LIMIT 1`, deptID, gradeID, compID, score).Scan(&text)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return text, true, nil
}

// userDeptGrade returns a user's department and grade (both may be nil).
func (r *Repository) userDeptGrade(ctx context.Context, userID uuid.UUID) (deptID, gradeID *uuid.UUID, err error) {
	err = r.db.QueryRow(ctx, `SELECT department_id, grade_id FROM users WHERE id = $1`, userID).Scan(&deptID, &gradeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	return deptID, gradeID, err
}

// UpsertInterpretation creates or updates the active record for a key, bumping
// the version and writing a history row (FR-AS7.2.3, 7.2.4).
func (r *Repository) UpsertInterpretation(ctx context.Context, req UpsertInterpretationRequest, actorID uuid.UUID) (Interpretation, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Interpretation{}, err
	}
	defer tx.Rollback(ctx)

	var (
		id      uuid.UUID
		version int
		action  = "update"
	)
	err = tx.QueryRow(ctx, `
		SELECT id, version FROM assessment_interpretations
		WHERE department_id = $1 AND grade_id = $2 AND competency_id = $3 AND score = $4 AND is_active`,
		req.DepartmentID, req.GradeID, req.CompetencyID, req.Score).Scan(&id, &version)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		action = "create"
		err = tx.QueryRow(ctx, `
			INSERT INTO assessment_interpretations
				(department_id, grade_id, competency_id, score, text, version, created_by, updated_by)
			VALUES ($1,$2,$3,$4,$5,1,$6,$6)
			RETURNING id, version`,
			req.DepartmentID, req.GradeID, req.CompetencyID, req.Score, req.Text, actorID).Scan(&id, &version)
		if err != nil {
			return Interpretation{}, err
		}
	case err != nil:
		return Interpretation{}, err
	default:
		version++
		if _, err := tx.Exec(ctx, `
			UPDATE assessment_interpretations
			SET text = $2, version = $3, updated_by = $4, updated_at = now()
			WHERE id = $1`, id, req.Text, version, actorID); err != nil {
			return Interpretation{}, err
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO assessment_interpretation_history
			(interpretation_id, department_id, grade_id, competency_id, score, text, version, action, changed_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		id, req.DepartmentID, req.GradeID, req.CompetencyID, req.Score, req.Text, version, action, actorID); err != nil {
		return Interpretation{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Interpretation{}, err
	}
	return r.GetInterpretation(ctx, id)
}

func (r *Repository) GetInterpretation(ctx context.Context, id uuid.UUID) (Interpretation, error) {
	var it Interpretation
	err := r.db.QueryRow(ctx, `
		SELECT i.id, i.department_id, d.name, i.grade_id, g.name, i.competency_id, c.name,
		       i.score, i.text, i.version, i.is_active, i.created_by, i.updated_by, i.created_at, i.updated_at
		FROM assessment_interpretations i
		JOIN departments d ON d.id = i.department_id
		JOIN grades g ON g.id = i.grade_id
		JOIN competencies c ON c.id = i.competency_id
		WHERE i.id = $1`, id).Scan(
		&it.ID, &it.DepartmentID, &it.DepartmentName, &it.GradeID, &it.GradeName,
		&it.CompetencyID, &it.CompetencyName, &it.Score, &it.Text, &it.Version,
		&it.IsActive, &it.CreatedBy, &it.UpdatedBy, &it.CreatedAt, &it.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Interpretation{}, ErrNotFound
	}
	return it, err
}

// ListInterpretations returns active records, optionally filtered.
func (r *Repository) ListInterpretations(ctx context.Context, deptID, gradeID, compID *uuid.UUID) ([]Interpretation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT i.id, i.department_id, d.name, i.grade_id, g.name, i.competency_id, c.name,
		       i.score, i.text, i.version, i.is_active, i.created_by, i.updated_by, i.created_at, i.updated_at
		FROM assessment_interpretations i
		JOIN departments d ON d.id = i.department_id
		JOIN grades g ON g.id = i.grade_id
		JOIN competencies c ON c.id = i.competency_id
		WHERE i.is_active
		  AND ($1::uuid IS NULL OR i.department_id = $1)
		  AND ($2::uuid IS NULL OR i.grade_id = $2)
		  AND ($3::uuid IS NULL OR i.competency_id = $3)
		ORDER BY d.name, g.level, c.sort_order, i.score`, deptID, gradeID, compID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Interpretation, 0)
	for rows.Next() {
		var it Interpretation
		if err := rows.Scan(&it.ID, &it.DepartmentID, &it.DepartmentName, &it.GradeID, &it.GradeName,
			&it.CompetencyID, &it.CompetencyName, &it.Score, &it.Text, &it.Version,
			&it.IsActive, &it.CreatedBy, &it.UpdatedBy, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// DeleteInterpretation soft-deactivates a record and journals it.
func (r *Repository) DeleteInterpretation(ctx context.Context, id uuid.UUID, actorID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var (
		dept, grade, comp uuid.UUID
		score, version    int
		text              string
	)
	err = tx.QueryRow(ctx, `
		SELECT department_id, grade_id, competency_id, score, text, version
		FROM assessment_interpretations WHERE id = $1 AND is_active`, id).Scan(
		&dept, &grade, &comp, &score, &text, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE assessment_interpretations SET is_active = false, updated_by = $2, updated_at = now() WHERE id = $1`,
		id, actorID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO assessment_interpretation_history
			(interpretation_id, department_id, grade_id, competency_id, score, text, version, action, changed_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,'delete',$8)`,
		id, dept, grade, comp, score, text, version, actorID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// CopyInterpretations copies active records from one (dept[,grade]) to another.
func (r *Repository) CopyInterpretations(ctx context.Context, req CopyInterpretationsRequest, actorID uuid.UUID) (int, error) {
	src, err := r.ListInterpretations(ctx, &req.FromDepartmentID, req.FromGradeID, nil)
	if err != nil {
		return 0, err
	}
	copied := 0
	for _, it := range src {
		targetGrade := it.GradeID
		if req.FromGradeID != nil && req.ToGradeID != nil {
			targetGrade = *req.ToGradeID
		}
		if !req.Overwrite {
			if _, found, err := r.LookupInterpretation(ctx, req.ToDepartmentID, targetGrade, it.CompetencyID, it.Score); err != nil {
				return copied, err
			} else if found {
				continue
			}
		}
		if _, err := r.UpsertInterpretation(ctx, UpsertInterpretationRequest{
			DepartmentID: req.ToDepartmentID,
			GradeID:      targetGrade,
			CompetencyID: it.CompetencyID,
			Score:        it.Score,
			Text:         it.Text,
		}, actorID); err != nil {
			return copied, err
		}
		copied++
	}
	return copied, nil
}

func (r *Repository) InterpretationHistory(ctx context.Context, deptID, gradeID, compID *uuid.UUID, score *int) ([]InterpretationHistoryEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, interpretation_id, score, text, version, action, changed_by, changed_at
		FROM assessment_interpretation_history
		WHERE ($1::uuid IS NULL OR department_id = $1)
		  AND ($2::uuid IS NULL OR grade_id = $2)
		  AND ($3::uuid IS NULL OR competency_id = $3)
		  AND ($4::int  IS NULL OR score = $4)
		ORDER BY changed_at DESC
		LIMIT 500`, deptID, gradeID, compID, score)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]InterpretationHistoryEntry, 0)
	for rows.Next() {
		var e InterpretationHistoryEntry
		if err := rows.Scan(&e.ID, &e.InterpretationID, &e.Score, &e.Text, &e.Version,
			&e.Action, &e.ChangedBy, &e.ChangedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ── Service layer ────────────────────────────────────────────────────────────

func (s *Service) LookupInterpretationForScore(ctx context.Context, assesseeID, compID uuid.UUID, score int) (InterpretationLookup, error) {
	dept, grade, err := s.repo.userDeptGrade(ctx, assesseeID)
	if err != nil {
		return InterpretationLookup{}, err
	}
	if dept == nil || grade == nil {
		return InterpretationLookup{Found: false}, nil
	}
	text, found, err := s.repo.LookupInterpretation(ctx, *dept, *grade, compID, score)
	if err != nil {
		return InterpretationLookup{}, err
	}
	return InterpretationLookup{Found: found, Text: text}, nil
}

func (s *Service) UpsertInterpretation(ctx context.Context, req UpsertInterpretationRequest, actorID uuid.UUID) (Interpretation, error) {
	return s.repo.UpsertInterpretation(ctx, req, actorID)
}

func (s *Service) ListInterpretations(ctx context.Context, deptID, gradeID, compID *uuid.UUID) ([]Interpretation, error) {
	return s.repo.ListInterpretations(ctx, deptID, gradeID, compID)
}

func (s *Service) DeleteInterpretation(ctx context.Context, id, actorID uuid.UUID) error {
	return s.repo.DeleteInterpretation(ctx, id, actorID)
}

func (s *Service) CopyInterpretations(ctx context.Context, req CopyInterpretationsRequest, actorID uuid.UUID) (int, error) {
	return s.repo.CopyInterpretations(ctx, req, actorID)
}

func (s *Service) InterpretationHistory(ctx context.Context, deptID, gradeID, compID *uuid.UUID, score *int) ([]InterpretationHistoryEntry, error) {
	return s.repo.InterpretationHistory(ctx, deptID, gradeID, compID, score)
}
