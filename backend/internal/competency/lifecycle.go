package competency

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Campaign lifecycle errors.
var (
	ErrBadTransition = errors.New("invalid status transition")
	ErrNoCriteria    = errors.New("campaign has no criteria")
	ErrNoAssessees   = errors.New("campaign has no assessees")
	ErrNotPublished  = errors.New("results are not published yet")
)

// Status constants for assessment campaigns (Section 5).
const (
	StatusDraft       = "draft"
	StatusAssigned    = "assigned"
	StatusInProgress  = "in_progress"
	StatusAdminReview = "admin_review"
	StatusConfirmed   = "confirmed"
	StatusPublished   = "published"
)

// ── Targeting (FR-AS1) ───────────────────────────────────────────────────────

// SetPeriodTargets replaces the department/section targeting of a campaign.
func (r *Repository) SetPeriodTargets(ctx context.Context, periodID uuid.UUID, depts, sections []uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM assessment_period_departments WHERE period_id = $1`, periodID); err != nil {
		return err
	}
	for _, d := range depts {
		if _, err := tx.Exec(ctx,
			`INSERT INTO assessment_period_departments (period_id, department_id) VALUES ($1,$2)
			 ON CONFLICT DO NOTHING`, periodID, d); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM assessment_period_sections WHERE period_id = $1`, periodID); err != nil {
		return err
	}
	for _, s := range sections {
		if _, err := tx.Exec(ctx,
			`INSERT INTO assessment_period_sections (period_id, section_id) VALUES ($1,$2)
			 ON CONFLICT DO NOTHING`, periodID, s); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) PeriodTargets(ctx context.Context, periodID uuid.UUID) (depts, sections []uuid.UUID, err error) {
	depts = []uuid.UUID{}
	sections = []uuid.UUID{}
	rows, err := r.db.Query(ctx, `SELECT department_id FROM assessment_period_departments WHERE period_id = $1`, periodID)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, nil, err
		}
		depts = append(depts, id)
	}
	rows.Close()
	rows, err = r.db.Query(ctx, `SELECT section_id FROM assessment_period_sections WHERE period_id = $1`, periodID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, nil, err
		}
		sections = append(sections, id)
	}
	return depts, sections, rows.Err()
}

// ── Criteria (FR-AS3) ────────────────────────────────────────────────────────

// SetCriteria replaces the criteria set of a campaign.
func (r *Repository) SetCriteria(ctx context.Context, periodID uuid.UUID, inputs []CriterionInput) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM assessment_criteria WHERE period_id = $1`, periodID); err != nil {
		return err
	}
	for i, in := range inputs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO assessment_criteria (period_id, competency_id, name, description, min_score, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (period_id, competency_id) DO UPDATE SET
				name = EXCLUDED.name, description = EXCLUDED.description,
				min_score = EXCLUDED.min_score, sort_order = EXCLUDED.sort_order`,
			periodID, in.CompetencyID, in.Name, in.Description, in.MinScore, (i+1)*10); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) ListCriteria(ctx context.Context, periodID uuid.UUID) ([]Criterion, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ac.id, ac.period_id, ac.competency_id, c.code,
		       COALESCE(ac.name, c.name), ac.description, ac.min_score, ac.sort_order
		FROM assessment_criteria ac
		JOIN competencies c ON c.id = ac.competency_id
		WHERE ac.period_id = $1
		ORDER BY ac.sort_order, c.name`, periodID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Criterion, 0)
	for rows.Next() {
		var c Criterion
		if err := rows.Scan(&c.ID, &c.PeriodID, &c.CompetencyID, &c.CompetencyCode,
			&c.Name, &c.Description, &c.MinScore, &c.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ── Assessees (FR-AS2) ───────────────────────────────────────────────────────

// AddAssessees resolves user ids directly plus everyone in the given departments
// and sections, then inserts them as assessees (status "participant").
func (r *Repository) AddAssessees(ctx context.Context, periodID uuid.UUID, userIDs, deptIDs, sectionIDs []uuid.UUID) (int, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	resolved := map[uuid.UUID]struct{}{}
	for _, id := range userIDs {
		resolved[id] = struct{}{}
	}
	if len(deptIDs) > 0 {
		rows, err := tx.Query(ctx,
			`SELECT id FROM users WHERE department_id = ANY($1::uuid[]) AND is_active = true AND deleted_at IS NULL`, deptIDs)
		if err != nil {
			return 0, err
		}
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return 0, err
			}
			resolved[id] = struct{}{}
		}
		rows.Close()
	}
	if len(sectionIDs) > 0 {
		rows, err := tx.Query(ctx,
			`SELECT id FROM users WHERE section_id = ANY($1::uuid[]) AND is_active = true AND deleted_at IS NULL`, sectionIDs)
		if err != nil {
			return 0, err
		}
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return 0, err
			}
			resolved[id] = struct{}{}
		}
		rows.Close()
	}

	added := 0
	for id := range resolved {
		tag, err := tx.Exec(ctx, `
			INSERT INTO assessment_assessees (period_id, user_id, status)
			VALUES ($1, $2, 'participant')
			ON CONFLICT (period_id, user_id) DO NOTHING`, periodID, id)
		if err != nil {
			return 0, err
		}
		added += int(tag.RowsAffected())
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return added, nil
}

func (r *Repository) ListAssessees(ctx context.Context, periodID uuid.UUID) ([]Assessee, error) {
	rows, err := r.db.Query(ctx, `
		SELECT aa.id, aa.period_id, aa.user_id, u.full_name, aa.status,
		       u.grade_id, g.name, u.department_id, aa.added_at
		FROM assessment_assessees aa
		JOIN users u ON u.id = aa.user_id
		LEFT JOIN grades g ON g.id = u.grade_id
		WHERE aa.period_id = $1
		ORDER BY u.full_name`, periodID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Assessee, 0)
	for rows.Next() {
		var a Assessee
		if err := rows.Scan(&a.ID, &a.PeriodID, &a.UserID, &a.FullName, &a.Status,
			&a.GradeID, &a.GradeName, &a.DepartmentID, &a.AddedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repository) RemoveAssessee(ctx context.Context, periodID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM assessment_assessees WHERE period_id = $1 AND user_id = $2`, periodID, userID)
	return err
}

// ── Per-assessee assessors (FR-AS4) ──────────────────────────────────────────

// SetAssesseeAssessors replaces the assessor set for one assessee and keeps the
// period-wide ASSESSOR participant rows in sync (so they appear in "my periods").
func (r *Repository) SetAssesseeAssessors(ctx context.Context, periodID, assesseeID uuid.UUID, assessorIDs []uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`DELETE FROM assessment_assessee_assessors WHERE period_id = $1 AND assessee_user_id = $2`,
		periodID, assesseeID); err != nil {
		return err
	}
	for _, aid := range assessorIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO assessment_assessee_assessors (period_id, assessee_user_id, assessor_user_id)
			VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, periodID, assesseeID, aid); err != nil {
			return err
		}
		// Mirror into participants so the assessor sees the campaign.
		if _, err := tx.Exec(ctx, `
			INSERT INTO assessment_participants (period_id, user_id, role)
			VALUES ($1, $2, 'ASSESSOR') ON CONFLICT (period_id, user_id, role) DO NOTHING`,
			periodID, aid); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) ListAssesseeAssessors(ctx context.Context, periodID uuid.UUID) ([]AssesseeAssessor, error) {
	rows, err := r.db.Query(ctx, `
		SELECT aaa.id, aaa.period_id, aaa.assessee_user_id, aaa.assessor_user_id, u.full_name
		FROM assessment_assessee_assessors aaa
		JOIN users u ON u.id = aaa.assessor_user_id
		WHERE aaa.period_id = $1
		ORDER BY aaa.assessee_user_id, u.full_name`, periodID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]AssesseeAssessor, 0)
	for rows.Next() {
		var a AssesseeAssessor
		if err := rows.Scan(&a.ID, &a.PeriodID, &a.AssesseeUserID, &a.AssessorUserID, &a.AssessorName); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// AssessorsOf returns the assessor user ids assigned to a specific assessee. If
// none are assigned per-assessee, falls back to the period-wide ASSESSOR group.
func (r *Repository) AssessorsOf(ctx context.Context, periodID, assesseeID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx,
		`SELECT assessor_user_id FROM assessment_assessee_assessors WHERE period_id = $1 AND assessee_user_id = $2`,
		periodID, assesseeID)
	if err != nil {
		return nil, err
	}
	out := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, id)
	}
	rows.Close()
	if len(out) > 0 {
		return out, nil
	}
	return r.AssessorParticipants(ctx, periodID)
}

// ── Status transitions (Section 5, FR-AS9, FR-AS10) ──────────────────────────

func (r *Repository) SetStatus(ctx context.Context, periodID uuid.UUID, status string) error {
	var tsCol string
	switch status {
	case StatusConfirmed:
		tsCol = ", confirmed_at = now()"
	case StatusPublished:
		tsCol = ", published_at = now()"
	}
	tag, err := r.db.Exec(ctx,
		`UPDATE assessment_periods SET status = $2, updated_at = now()`+tsCol+` WHERE id = $1`, periodID, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) CountCriteria(ctx context.Context, periodID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT count(*) FROM assessment_criteria WHERE period_id = $1`, periodID).Scan(&n)
	return n, err
}

func (r *Repository) CountAssessees(ctx context.Context, periodID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT count(*) FROM assessment_assessees WHERE period_id = $1`, periodID).Scan(&n)
	return n, err
}

// SyncTalentProfiles stamps every assessee with the publish date/status (FR-AS11).
func (r *Repository) SyncTalentProfiles(ctx context.Context, periodID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users u SET last_assessment_at = now(), last_assessment_status = 'passed'
		FROM assessment_assessees aa
		WHERE aa.period_id = $1 AND aa.user_id = u.id`, periodID)
	return err
}

// ── Service layer ────────────────────────────────────────────────────────────

func (s *Service) SetCriteria(ctx context.Context, periodID uuid.UUID, inputs []CriterionInput) error {
	if _, err := s.repo.GetPeriod(ctx, periodID); err != nil {
		return ErrNotFound
	}
	return s.repo.SetCriteria(ctx, periodID, inputs)
}

func (s *Service) ListCriteria(ctx context.Context, periodID uuid.UUID) ([]Criterion, error) {
	return s.repo.ListCriteria(ctx, periodID)
}

func (s *Service) AddAssessees(ctx context.Context, periodID uuid.UUID, req AddAssesseesRequest) (int, error) {
	if _, err := s.repo.GetPeriod(ctx, periodID); err != nil {
		return 0, ErrNotFound
	}
	users, err := parseUUIDs(req.UserIDs)
	if err != nil {
		return 0, errors.New("invalid user_ids")
	}
	depts, err := parseUUIDs(req.DepartmentIDs)
	if err != nil {
		return 0, errors.New("invalid department_ids")
	}
	sections, err := parseUUIDs(req.SectionIDs)
	if err != nil {
		return 0, errors.New("invalid section_ids")
	}
	return s.repo.AddAssessees(ctx, periodID, users, depts, sections)
}

func (s *Service) ListAssessees(ctx context.Context, periodID uuid.UUID) ([]Assessee, error) {
	return s.repo.ListAssessees(ctx, periodID)
}

func (s *Service) RemoveAssessee(ctx context.Context, periodID, userID uuid.UUID) error {
	return s.repo.RemoveAssessee(ctx, periodID, userID)
}

func (s *Service) SetAssesseeAssessors(ctx context.Context, periodID, assesseeID uuid.UUID, assessorIDs []string) error {
	ids, err := parseUUIDs(assessorIDs)
	if err != nil {
		return errors.New("invalid assessor_user_ids")
	}
	return s.repo.SetAssesseeAssessors(ctx, periodID, assesseeID, ids)
}

func (s *Service) ListAssesseeAssessors(ctx context.Context, periodID uuid.UUID) ([]AssesseeAssessor, error) {
	return s.repo.ListAssesseeAssessors(ctx, periodID)
}

// Transition validates and applies a campaign status change.
func (s *Service) Transition(ctx context.Context, periodID uuid.UUID, to string, actorID uuid.UUID) (Period, error) {
	p, err := s.repo.GetPeriod(ctx, periodID)
	if err != nil {
		return Period{}, ErrNotFound
	}
	if !allowedTransition(p.Status, to) {
		return Period{}, fmt.Errorf("%w: %s → %s", ErrBadTransition, p.Status, to)
	}
	switch to {
	case StatusAssigned:
		if n, _ := s.repo.CountCriteria(ctx, periodID); n == 0 {
			return Period{}, ErrNoCriteria
		}
		if n, _ := s.repo.CountAssessees(ctx, periodID); n == 0 {
			return Period{}, ErrNoAssessees
		}
	case StatusConfirmed:
		// Form learning groups on confirmation (FR-AS13: only assessor scores).
		if err := s.repo.FormGroups(ctx, periodID, p.GroupSize, actorID); err != nil {
			return Period{}, err
		}
	case StatusPublished:
		if err := s.repo.SyncTalentProfiles(ctx, periodID); err != nil {
			return Period{}, err
		}
	}
	if err := s.repo.SetStatus(ctx, periodID, to); err != nil {
		return Period{}, err
	}
	return s.repo.GetPeriod(ctx, periodID)
}

func allowedTransition(from, to string) bool {
	switch from {
	case StatusDraft:
		return to == StatusAssigned
	case StatusAssigned:
		return to == StatusInProgress || to == StatusDraft
	case StatusInProgress:
		return to == StatusAdminReview
	case StatusAdminReview:
		return to == StatusConfirmed || to == StatusInProgress // confirm or return
	case StatusConfirmed:
		return to == StatusPublished || to == StatusAdminReview
	}
	return false
}

// EmployeeResult is one published consolidated competency result for a worker.
type EmployeeResult struct {
	PeriodID       uuid.UUID `json:"period_id"`
	PeriodTitle    string    `json:"period_title"`
	CompetencyID   uuid.UUID `json:"competency_id"`
	CompetencyName string    `json:"competency_name"`
	AvgScore       float64   `json:"avg_score"`
	PublishedAt    *string   `json:"published_at,omitempty"`
	// Divergent is true when any two role marks (HEAD/DEPT_HEAD/HRA/DCR_HEAD)
	// for this cell differ by more than 4 (assessment disagreement flag).
	Divergent bool `json:"divergent"`
}

// MyPublishedResults returns the employee's consolidated results from published
// campaigns only (FR-AS10: not visible before publication).
func (r *Repository) MyPublishedResults(ctx context.Context, userID uuid.UUID) ([]EmployeeResult, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ac.period_id, p.title, ac.competency_id, c.name, ac.avg_score,
		       to_char(p.published_at, 'YYYY-MM-DD'),
		       COALESCE((
		         SELECT MAX(s.score) - MIN(s.score) > 4
		           FROM assessment_scores s
		          WHERE s.period_id = ac.period_id
		            AND s.employee_id = ac.employee_id
		            AND s.competency_id = ac.competency_id
		            AND s.assessor_role IN ('HEAD','DEPT_HEAD','HRA','DCR_HEAD')
		            AND s.score IS NOT NULL
		       ), false)
		FROM assessment_consolidated ac
		JOIN assessment_periods p ON p.id = ac.period_id
		JOIN competencies c ON c.id = ac.competency_id
		WHERE ac.employee_id = $1 AND p.status = 'published'
		ORDER BY p.published_at DESC NULLS LAST, c.sort_order`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]EmployeeResult, 0)
	for rows.Next() {
		var e EmployeeResult
		if err := rows.Scan(&e.PeriodID, &e.PeriodTitle, &e.CompetencyID, &e.CompetencyName,
			&e.AvgScore, &e.PublishedAt, &e.Divergent); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Service) MyPublishedResults(ctx context.Context, userID uuid.UUID) ([]EmployeeResult, error) {
	return s.repo.MyPublishedResults(ctx, userID)
}

// GetPeriodFull returns a period plus its targeting and criteria.
func (s *Service) GetPeriodFull(ctx context.Context, id uuid.UUID) (Period, error) {
	p, err := s.repo.GetPeriod(ctx, id)
	if err != nil {
		return Period{}, ErrNotFound
	}
	depts, sections, err := s.repo.PeriodTargets(ctx, id)
	if err != nil {
		return Period{}, err
	}
	p.DepartmentIDs = depts
	p.SectionIDs = sections
	return p, nil
}
