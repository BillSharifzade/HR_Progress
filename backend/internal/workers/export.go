package workers

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ExportDataset is the whole "Сотрудники" register plus every related record
// the platform holds for those people, gathered in a fixed number of bulk
// queries rather than per worker: an export of 150 people would otherwise fire
// close to a thousand round trips.
//
// Workers arrive pre-sorted (department → section → grade, senior first →
// name) and every detail slice is keyed back to that order, so the row number
// printed in the first column of every sheet identifies the same person
// throughout the workbook.
type ExportDataset struct {
	GeneratedAt time.Time
	// Filters describes the filter set the export ran under, so a file that
	// holds a subset says so on its cover sheet instead of looking truncated.
	Filters []ExportFilterNote
	Workers []ExportWorker
}

type ExportFilterNote struct {
	Label string
	Value string
}

// ExportWorker is a worker record with everything hanging off it. The embedded
// Worker carries the core fields; the rest is what the detail sheets consume.
type ExportWorker struct {
	Worker

	DepartmentCode *string
	SectionCode    *string
	// ManagerName is resolved through the 1F manager link, which stores the
	// manager's 1F id rather than a local user id.
	ManagerName          *string
	LastAssessmentAt     *time.Time
	LastAssessmentStatus *string

	Profile         *Profile
	Languages       []Language
	Experience      []WorkExperience
	Certifications  []Certification
	Survey          []SurveyAnswer
	History         []History
	Scores          []ExportScore
	RoleAssignments []RoleAssignment
}

// ExportScore flattens an assessment score together with the campaign and
// competency it belongs to — the export never resolves ids, it prints names.
type ExportScore struct {
	PeriodTitle    string
	PeriodStart    time.Time
	PeriodEnd      time.Time
	PeriodStatus   string
	CompetencyCode string
	CompetencyKind string
	CompetencyName string
	AssessorRole   string
	Score          *float64
	Feedback       *string
	AssessedByName *string
	AssessedAt     *time.Time
}

const exportWorkerSelect = `
SELECT
    u.id, u.username, u.employee_no, u.personnel_number,
    u.one_f_user_id, u.one_f_is_manager, u.phone_number, u.last_synced_at,
    u.full_name, u.email, u.birth_date,
    u.department_id, d.name, d.code,
    u.section_id,   s.name, s.code,
    u.grade_id,     g.name, g.level,
    u.position_id,  p.name, u.position,
    u.specialization, u.telegram_id, u.hired_at, u.hobbies, u.is_active,
    m.full_name, u.last_assessment_at, u.last_assessment_status
FROM users u
LEFT JOIN departments d ON d.id = u.department_id AND d.deleted_at IS NULL
LEFT JOIN sections    s ON s.id = u.section_id    AND s.deleted_at IS NULL
LEFT JOIN grades      g ON g.id = u.grade_id      AND g.deleted_at IS NULL
LEFT JOIN positions   p ON p.id = u.position_id   AND p.deleted_at IS NULL
LEFT JOIN users       m ON m.one_f_user_id = u.one_f_manager_user_id AND m.deleted_at IS NULL`

// ExportDataset loads every worker matching f, together with their profile,
// languages, prior employment, certificates, questionnaire answers, history,
// assessment scores and system roles.
func (r *Repository) ExportDataset(ctx context.Context, f ListFilter) (*ExportDataset, error) {
	conds, args := listConditions(f)

	// Unassigned people sort last rather than first: a register opens on the
	// staffed departments, and the "no department" tail is the exception list.
	rows, err := r.pool.Query(ctx, exportWorkerSelect+`
	WHERE `+joinConds(conds)+`
	ORDER BY d.name NULLS LAST, s.name NULLS LAST, g.level DESC NULLS LAST, u.full_name`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ds := &ExportDataset{GeneratedAt: time.Now()}
	byID := map[uuid.UUID]*ExportWorker{}
	for rows.Next() {
		w := ExportWorker{Worker: Worker{Roles: []string{}}}
		if err := rows.Scan(
			&w.ID, &w.Username, &w.EmployeeNo, &w.PersonnelNumber,
			&w.OneFUserID, &w.OneFIsManager, &w.PhoneNumber, &w.LastSyncedAt,
			&w.FullName, &w.Email, &w.BirthDate,
			&w.DepartmentID, &w.DepartmentName, &w.DepartmentCode,
			&w.SectionID, &w.SectionName, &w.SectionCode,
			&w.GradeID, &w.GradeName, &w.GradeLevel,
			&w.PositionID, &w.PositionName, &w.Position,
			&w.Specialization, &w.TelegramID, &w.HiredAt, &w.Hobbies, &w.IsActive,
			&w.ManagerName, &w.LastAssessmentAt, &w.LastAssessmentStatus,
		); err != nil {
			return nil, err
		}
		ds.Workers = append(ds.Workers, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ds.Workers) == 0 {
		return ds, nil
	}

	ids := make([]uuid.UUID, len(ds.Workers))
	for i := range ds.Workers {
		ids[i] = ds.Workers[i].ID
		byID[ds.Workers[i].ID] = &ds.Workers[i]
	}

	for _, load := range []func(context.Context, []uuid.UUID, map[uuid.UUID]*ExportWorker) error{
		r.loadExportRoles,
		r.loadExportProfiles,
		r.loadExportLanguages,
		r.loadExportExperience,
		r.loadExportCertifications,
		r.loadExportSurvey,
		r.loadExportHistory,
		r.loadExportScores,
	} {
		if err := load(ctx, ids, byID); err != nil {
			return nil, err
		}
	}
	return ds, nil
}

func (r *Repository) loadExportRoles(ctx context.Context, ids []uuid.UUID, byID map[uuid.UUID]*ExportWorker) error {
	rows, err := r.pool.Query(ctx,
		roleAssignmentSelect+` WHERE ur.user_id = ANY($1) ORDER BY ur.granted_at`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		a, err := scanRoleAssignment(rows)
		if err != nil {
			return err
		}
		if w := byID[a.UserID]; w != nil {
			w.RoleAssignments = append(w.RoleAssignments, a)
			w.Roles = append(w.Roles, a.Role)
		}
	}
	return rows.Err()
}

func (r *Repository) loadExportProfiles(ctx context.Context, ids []uuid.UUID, byID map[uuid.UUID]*ExportWorker) error {
	rows, err := r.pool.Query(ctx,
		`SELECT `+profileCols+` FROM user_profile WHERE user_id = ANY($1)`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return err
		}
		if w := byID[p.UserID]; w != nil {
			w.Profile = p
		}
	}
	return rows.Err()
}

func (r *Repository) loadExportLanguages(ctx context.Context, ids []uuid.UUID, byID map[uuid.UUID]*ExportWorker) error {
	rows, err := r.pool.Query(ctx, `
		SELECT `+languageCols+` FROM user_languages WHERE user_id = ANY($1)
		ORDER BY array_position(ARRAY['C2','C1','B2','B1','A2','A1'], level), lower(language)`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var l Language
		if err := rows.Scan(&l.ID, &l.UserID, &l.Language, &l.Level,
			&l.Source, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return err
		}
		if w := byID[l.UserID]; w != nil {
			w.Languages = append(w.Languages, l)
		}
	}
	return rows.Err()
}

func (r *Repository) loadExportExperience(ctx context.Context, ids []uuid.UUID, byID map[uuid.UUID]*ExportWorker) error {
	rows, err := r.pool.Query(ctx, `
		SELECT `+experienceCols+` FROM user_work_experience WHERE user_id = ANY($1)
		ORDER BY sort_order, created_at`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		e, err := scanExperience(rows)
		if err != nil {
			return err
		}
		if w := byID[e.UserID]; w != nil {
			w.Experience = append(w.Experience, *e)
		}
	}
	return rows.Err()
}

func (r *Repository) loadExportCertifications(ctx context.Context, ids []uuid.UUID, byID map[uuid.UUID]*ExportWorker) error {
	rows, err := r.pool.Query(ctx, `
		SELECT `+certificationCols+` FROM user_certifications WHERE user_id = ANY($1)
		ORDER BY issued_at DESC NULLS LAST, created_at DESC`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		c, err := scanCertification(rows)
		if err != nil {
			return err
		}
		if w := byID[c.UserID]; w != nil {
			w.Certifications = append(w.Certifications, *c)
		}
	}
	return rows.Err()
}

func (r *Repository) loadExportSurvey(ctx context.Context, ids []uuid.UUID, byID map[uuid.UUID]*ExportWorker) error {
	rows, err := r.pool.Query(ctx, `
		SELECT `+surveyCols+` FROM user_survey_answers WHERE user_id = ANY($1)
		ORDER BY position, question_code`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var a SurveyAnswer
		if err := rows.Scan(&a.ID, &a.UserID, &a.FormKey, &a.QuestionCode, &a.QuestionText,
			&a.AnswerText, &a.Position, &a.SubmittedAt, &a.Source,
			&a.CreatedAt, &a.UpdatedAt); err != nil {
			return err
		}
		if w := byID[a.UserID]; w != nil {
			w.Survey = append(w.Survey, a)
		}
	}
	return rows.Err()
}

func (r *Repository) loadExportHistory(ctx context.Context, ids []uuid.UUID, byID map[uuid.UUID]*ExportWorker) error {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, event_kind::text, event_date, title, description, meta, created_at
		FROM user_history WHERE user_id = ANY($1) ORDER BY event_date DESC`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var h History
		if err := rows.Scan(&h.ID, &h.UserID, &h.EventKind, &h.EventDate,
			&h.Title, &h.Description, &h.Meta, &h.CreatedAt); err != nil {
			return err
		}
		if w := byID[h.UserID]; w != nil {
			w.History = append(w.History, h)
		}
	}
	return rows.Err()
}

func (r *Repository) loadExportScores(ctx context.Context, ids []uuid.UUID, byID map[uuid.UUID]*ExportWorker) error {
	rows, err := r.pool.Query(ctx, `
		SELECT sc.employee_id,
		       ap.title, ap.period_start, ap.period_end, ap.status,
		       c.code, c.kind::text, c.name,
		       sc.assessor_role, sc.score, sc.feedback, ab.full_name, sc.assessed_at
		FROM assessment_scores sc
		JOIN assessment_periods ap ON ap.id = sc.period_id
		JOIN competencies       c  ON c.id  = sc.competency_id
		LEFT JOIN users         ab ON ab.id = sc.assessed_by
		WHERE sc.employee_id = ANY($1)
		ORDER BY ap.period_start DESC, c.sort_order, c.code, sc.assessor_role`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			userID uuid.UUID
			s      ExportScore
		)
		if err := rows.Scan(&userID,
			&s.PeriodTitle, &s.PeriodStart, &s.PeriodEnd, &s.PeriodStatus,
			&s.CompetencyCode, &s.CompetencyKind, &s.CompetencyName,
			&s.AssessorRole, &s.Score, &s.Feedback, &s.AssessedByName, &s.AssessedAt); err != nil {
			return err
		}
		if w := byID[userID]; w != nil {
			w.Scores = append(w.Scores, s)
		}
	}
	return rows.Err()
}

// LookupNames resolves the display names used on the cover sheet for the
// filters an export ran under. An id that no longer resolves comes back empty,
// which the caller renders as the raw id rather than failing the whole export.
func (r *Repository) LookupNames(ctx context.Context, deptID, sectionID, gradeID *uuid.UUID) (dept, section, grade string, err error) {
	get := func(table string, id *uuid.UUID, dst *string) error {
		if id == nil {
			return nil
		}
		err := r.pool.QueryRow(ctx, `SELECT name FROM `+table+` WHERE id = $1`, *id).Scan(dst)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	if err = get("departments", deptID, &dept); err != nil {
		return
	}
	if err = get("sections", sectionID, &section); err != nil {
		return
	}
	err = get("grades", gradeID, &grade)
	return
}
