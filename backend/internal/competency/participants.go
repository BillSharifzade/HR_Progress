package competency

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrNotParticipant       = errors.New("user is not a participant in this period")
	ErrRoleNotInParticipant = errors.New("user is not a participant in this role")
	ErrParticipantsLocked   = errors.New("participants are locked for this period")
)

// ListParticipants returns all participants assigned to a period with the
// user's display info (username + full_name) so the UI can render them.
func (r *Repository) ListParticipants(ctx context.Context, periodID uuid.UUID) ([]Participant, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ap.id, ap.period_id, ap.user_id, ap.role, ap.added_at,
		       u.username, u.full_name
		  FROM assessment_participants ap
		  JOIN users u ON u.id = ap.user_id
		 WHERE ap.period_id = $1
		 ORDER BY ap.role, u.full_name`, periodID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Participant, 0)
	for rows.Next() {
		var p Participant
		if err := rows.Scan(&p.ID, &p.PeriodID, &p.UserID, &p.Role, &p.AddedAt,
			&p.UserName, &p.FullName); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// AddParticipants inserts the given (user, role) pairs in one tx.
// Returns ErrParticipantsLocked if the period already has participants —
// the list is fixed at first-fill (locked-in policy).
func (r *Repository) AddParticipants(ctx context.Context, periodID uuid.UUID, parts []ParticipantInput) error {
	if len(parts) == 0 {
		return errors.New("no participants supplied")
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var count int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM assessment_participants WHERE period_id = $1`,
		periodID,
	).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return ErrParticipantsLocked
	}

	for _, p := range parts {
		_, err := tx.Exec(ctx, `
			INSERT INTO assessment_participants (period_id, user_id, role)
			VALUES ($1, $2, $3)
			ON CONFLICT (period_id, user_id, role) DO NOTHING`,
			periodID, p.UserID, p.Role)
		if err != nil {
			return fmt.Errorf("insert participant (%s, %s): %w", p.UserID, p.Role, err)
		}
	}
	return tx.Commit(ctx)
}

// MyRolesIn returns the roles the user holds in the given period — both
// explicitly assigned (assessment_participants, e.g. ASSESSOR) and derived
// from org structure (SECTION_HEAD → HEAD, DEPT_HEAD of period's dept →
// DEPT_HEAD, DEPT_HEAD of ДЧР → DCR_HEAD). Returns (nil, nil) if user has
// no role in this period.
func (r *Repository) MyRolesIn(ctx context.Context, periodID, userID uuid.UUID) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT role FROM (
		  SELECT role::text FROM assessment_participants
		   WHERE period_id = $1 AND user_id = $2
		  UNION
		  SELECT 'HEAD' FROM user_roles ur
		    JOIN sections s ON s.id = ur.scope_section_id AND s.deleted_at IS NULL
		    JOIN assessment_periods p ON p.id = $1
		   WHERE ur.user_id = $2 AND ur.role = 'SECTION_HEAD'
		     AND s.department_id = p.department_id
		  UNION
		  SELECT 'DEPT_HEAD' FROM user_roles ur
		    JOIN assessment_periods p ON p.id = $1
		   WHERE ur.user_id = $2 AND ur.role = 'DEPT_HEAD'
		     AND ur.scope_department_id = p.department_id
		  UNION
		  SELECT 'DCR_HEAD' FROM user_roles ur
		    JOIN departments d ON d.id = ur.scope_department_id
		     AND d.deleted_at IS NULL AND d.code = 'ДЧР'
		   WHERE ur.user_id = $2 AND ur.role = 'DEPT_HEAD'
		     AND EXISTS (SELECT 1 FROM assessment_periods WHERE id = $1)
		) src`, periodID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

// ListMyPeriods returns periods this user is participating in.
func (r *Repository) ListMyPeriods(ctx context.Context, userID uuid.UUID) ([]MyPeriod, error) {
	// Union explicit participation (ASSESSOR rows) with org-derived roles
	// (HEAD / DEPT_HEAD / DCR_HEAD) so heads see periods even though only
	// ASSESSORs are stored in assessment_participants.
	rows, err := r.db.Query(ctx, `
		WITH my_participation AS (
		  SELECT period_id, role::text AS role
		    FROM assessment_participants WHERE user_id = $1
		  UNION
		  SELECT p.id AS period_id, 'HEAD' AS role
		    FROM user_roles ur
		    JOIN sections s ON s.id = ur.scope_section_id AND s.deleted_at IS NULL
		    JOIN assessment_periods p ON p.department_id = s.department_id
		   WHERE ur.user_id = $1 AND ur.role = 'SECTION_HEAD'
		  UNION
		  SELECT p.id AS period_id, 'DEPT_HEAD' AS role
		    FROM user_roles ur
		    JOIN assessment_periods p ON p.department_id = ur.scope_department_id
		   WHERE ur.user_id = $1 AND ur.role = 'DEPT_HEAD'
		  UNION
		  SELECT p.id AS period_id, 'DCR_HEAD' AS role
		    FROM user_roles ur
		    JOIN departments d ON d.id = ur.scope_department_id
		     AND d.deleted_at IS NULL AND d.code = 'ДЧР'
		    CROSS JOIN assessment_periods p
		   WHERE ur.user_id = $1 AND ur.role = 'DEPT_HEAD'
		)
		SELECT p.id, p.title, p.period_start, p.period_end, p.is_active,
		       d.name, p.department_id,
		       array_agg(DISTINCT mp.role ORDER BY mp.role)
		  FROM my_participation mp
		  JOIN assessment_periods p ON p.id = mp.period_id
		  LEFT JOIN departments d ON d.id = p.department_id
		 GROUP BY p.id, p.title, p.period_start, p.period_end, p.is_active, d.name, p.department_id
		 ORDER BY p.period_start DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]MyPeriod, 0)
	for rows.Next() {
		var m MyPeriod
		if err := rows.Scan(&m.PeriodID, &m.Title, &m.PeriodStart, &m.PeriodEnd, &m.IsActive,
			&m.Department, &m.DepartmentID, &m.Roles); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// AssessorParticipants returns the user_ids of every ASSESSOR-group
// participant in a period.
func (r *Repository) AssessorParticipants(ctx context.Context, periodID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `
		SELECT user_id FROM assessment_participants
		 WHERE period_id = $1 AND role = 'ASSESSOR'`, periodID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// UpsertScoreFor a specific user. Sets user_id and assessed_by. autoInterp is
// the system-suggested interpretation snapshot at save time (FR-AS7.2 rule 24).
func (r *Repository) UpsertScoreFor(ctx context.Context, periodID, employeeID, competencyID, userID uuid.UUID, role string, score *float64, feedback, autoInterp *string) (Score, error) {
	var s Score
	err := r.db.QueryRow(ctx, `
		INSERT INTO assessment_scores
		    (period_id, employee_id, competency_id, assessor_role, user_id, score, feedback, auto_interpretation, assessed_by, assessed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $5, now())
		ON CONFLICT (period_id, employee_id, competency_id, user_id)
		  WHERE user_id IS NOT NULL
		  DO UPDATE SET assessor_role       = EXCLUDED.assessor_role,
		                score               = EXCLUDED.score,
		                feedback            = EXCLUDED.feedback,
		                auto_interpretation = EXCLUDED.auto_interpretation,
		                assessed_by         = EXCLUDED.assessed_by,
		                assessed_at         = EXCLUDED.assessed_at
		RETURNING id, period_id, employee_id, competency_id, assessor_role,
		          score, feedback, auto_interpretation, assessed_by, assessed_at, updated_at`,
		periodID, employeeID, competencyID, role, userID, score, feedback, autoInterp,
	).Scan(&s.ID, &s.PeriodID, &s.EmployeeID, &s.CompetencyID, &s.AssessorRole,
		&s.Score, &s.Feedback, &s.AutoInterpretation, &s.AssessedBy, &s.AssessedAt, &s.UpdatedAt)
	return s, err
}

// MaybeFinalize checks whether every ASSESSOR-group participant has a non-null
// score for (period, employee, competency); if so, it writes the consolidated
// matrix mark and upserts the consolidated row. Idempotent — safe to call after
// each save.
//
// The matrix mark averages "voices" that each weigh equally (Way A): the whole
// ASSESSOR group counts as a SINGLE voice (the average of their marks), and the
// subdivision head (HEAD), department head (DEPT_HEAD) and DHR head (DCR_HEAD)
// each count as one voice too. So e.g. assessors {6,8}, HEAD 9, DEPT 8, DHR 7 →
// avg(7, 9, 8, 7) = 7.75 — adding more assessors does not outweigh the heads.
// HRA is excluded: it is the derived assessor average, not a real evaluator.
// The head slots fold in whenever those marks exist; until their scoring UI
// ships the mark equals the assessor average and auto-upgrades later. NB: the
// learning-group engine (groups.go) deliberately uses ASSESSOR-only scores.
func (r *Repository) MaybeFinalize(ctx context.Context, periodID, employeeID, competencyID uuid.UUID) error {
	assessors, err := r.AssessorParticipants(ctx, periodID)
	if err != nil {
		return err
	}
	if len(assessors) == 0 {
		return nil
	}
	// `have` gates finalization: how many assigned assessors have scored this
	// cell. `avg` is the across-voices average — the assessor group as one
	// voice plus each present head — which is the value written to the matrix.
	var have int
	var avg float64
	err = r.db.QueryRow(ctx, `
		WITH assessor AS (
		  SELECT count(*) AS n, AVG(score) AS group_avg
		    FROM assessment_scores
		   WHERE period_id = $1 AND employee_id = $2 AND competency_id = $3
		     AND assessor_role = 'ASSESSOR' AND score IS NOT NULL
		     AND user_id = ANY($4::uuid[])
		),
		voices AS (
		  -- the assessor group counts as a single voice
		  SELECT group_avg AS v FROM assessor WHERE group_avg IS NOT NULL
		  UNION ALL
		  -- each head counts as its own voice
		  SELECT score FROM assessment_scores
		   WHERE period_id = $1 AND employee_id = $2 AND competency_id = $3
		     AND assessor_role IN ('HEAD', 'DEPT_HEAD', 'DCR_HEAD')
		     AND score IS NOT NULL
		)
		SELECT (SELECT n FROM assessor),
		       COALESCE((SELECT AVG(v) FROM voices), 0)`,
		periodID, employeeID, competencyID, assessors,
	).Scan(&have, &avg)
	if err != nil {
		return err
	}
	if have < len(assessors) {
		// Not yet complete — remove any stale consolidated row.
		_, err := r.db.Exec(ctx, `
			DELETE FROM assessment_consolidated
			 WHERE period_id = $1 AND employee_id = $2 AND competency_id = $3`,
			periodID, employeeID, competencyID)
		return err
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO assessment_consolidated (period_id, employee_id, competency_id, avg_score)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (period_id, employee_id, competency_id)
		  DO UPDATE SET avg_score = EXCLUDED.avg_score, finalized_at = now()`,
		periodID, employeeID, competencyID, avg)
	return err
}

// ListConsolidated returns all finalized consolidated rows for a period.
func (r *Repository) ListConsolidated(ctx context.Context, periodID uuid.UUID) ([]ConsolidatedScore, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, period_id, employee_id, competency_id, avg_score, finalized_at
		  FROM assessment_consolidated
		 WHERE period_id = $1`, periodID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ConsolidatedScore, 0)
	for rows.Next() {
		var c ConsolidatedScore
		if err := rows.Scan(&c.ID, &c.PeriodID, &c.EmployeeID, &c.CompetencyID,
			&c.AvgScore, &c.FinalizedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// MyScoresIn returns the scores this user has saved in a period (across all
// workers / competencies they're scoring).
func (r *Repository) MyScoresIn(ctx context.Context, periodID, userID uuid.UUID) ([]Score, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, period_id, employee_id, competency_id, assessor_role,
		       score, feedback, auto_interpretation, assessed_by, assessed_at, updated_at
		  FROM assessment_scores
		 WHERE period_id = $1 AND user_id = $2
		 ORDER BY employee_id, competency_id`, periodID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Score, 0)
	for rows.Next() {
		var s Score
		if err := rows.Scan(&s.ID, &s.PeriodID, &s.EmployeeID, &s.CompetencyID, &s.AssessorRole,
			&s.Score, &s.Feedback, &s.AutoInterpretation, &s.AssessedBy, &s.AssessedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListUsersWithRole returns active users who hold a given system role
// (HR_ADMIN, ASSESSOR, etc.). Used to populate participant pickers.
func (r *Repository) ListUsersWithRole(ctx context.Context, role string) ([]Employee, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT u.id, u.full_name, u.grade_id, g.name, g.level
		  FROM users u
		  JOIN user_roles ur ON ur.user_id = u.id
		  LEFT JOIN grades g ON g.id = u.grade_id
		 WHERE ur.role::text = $1
		   AND u.deleted_at IS NULL
		   AND u.is_active = true
		 ORDER BY u.full_name`, strings.ToUpper(role))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Employee, 0)
	for rows.Next() {
		var e Employee
		if err := rows.Scan(&e.ID, &e.FullName, &e.GradeID, &e.GradeName, &e.GradeLevel); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
