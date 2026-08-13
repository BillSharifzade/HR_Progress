package workers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ErrInvalidInput marks a caller mistake (a malformed date, an impossible
// range) as opposed to a server fault, so handlers can answer 422 rather than
// reporting a 500 for something the client can fix.
var ErrInvalidInput = errors.New("invalid input")

// --- models ---------------------------------------------------------------

// Profile is the structured half of the digital-profile questionnaire: the
// answers that come from a fixed option set and are therefore worth querying.
// Open-ended answers live in SurveyAnswer instead.
type Profile struct {
	UserID uuid.UUID `json:"user_id"`

	EducationLevels []string `json:"education_levels"`
	Institution     *string  `json:"institution,omitempty"`
	Specialty       *string  `json:"specialty,omitempty"`

	PriorExperienceBand   *string  `json:"prior_experience_band,omitempty"`
	CareerGoal            *string  `json:"career_goal,omitempty"`
	DevelopmentDirections []string `json:"development_directions"`

	MobilityReadiness         *string `json:"mobility_readiness,omitempty"`
	RelocationReadiness       *string `json:"relocation_readiness,omitempty"`
	InternalProjectsReadiness *string `json:"internal_projects_readiness,omitempty"`
	TeachingReadiness         *string `json:"teaching_readiness,omitempty"`

	ProfessionalInterests []string `json:"professional_interests"`
	LearningFormats       []string `json:"learning_formats"`
	LearningHoursBand     *string  `json:"learning_hours_band,omitempty"`

	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
	Source      string     `json:"source"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type UpsertProfileRequest struct {
	EducationLevels []string `json:"education_levels"`
	Institution     *string  `json:"institution"`
	Specialty       *string  `json:"specialty"`

	PriorExperienceBand   *string  `json:"prior_experience_band"`
	CareerGoal            *string  `json:"career_goal"`
	DevelopmentDirections []string `json:"development_directions"`

	MobilityReadiness         *string `json:"mobility_readiness"`
	RelocationReadiness       *string `json:"relocation_readiness"`
	InternalProjectsReadiness *string `json:"internal_projects_readiness"`
	TeachingReadiness         *string `json:"teaching_readiness"`

	ProfessionalInterests []string `json:"professional_interests"`
	LearningFormats       []string `json:"learning_formats"`
	LearningHoursBand     *string  `json:"learning_hours_band"`
}

type Language struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Language  string    `json:"language"`
	Level     string    `json:"level"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UpsertLanguageRequest struct {
	Language string `json:"language" validate:"required,min=1,max=100"`
	Level    string `json:"level"    validate:"required,oneof=A1 A2 B1 B2 C1 C2"`
}

type WorkExperience struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	Company     string     `json:"company"`
	Position    *string    `json:"position,omitempty"`
	StartedOn   *time.Time `json:"started_on,omitempty"`
	EndedOn     *time.Time `json:"ended_on,omitempty"`
	Description *string    `json:"description,omitempty"`
	SortOrder   int        `json:"sort_order"`
	RawText     *string    `json:"raw_text,omitempty"`
	Source      string     `json:"source"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type UpsertWorkExperienceRequest struct {
	Company     string  `json:"company"  validate:"required,min=1,max=300"`
	Position    *string `json:"position"`
	StartedOn   *string `json:"started_on"`
	EndedOn     *string `json:"ended_on"`
	Description *string `json:"description"`
	SortOrder   *int    `json:"sort_order"`
}

type SurveyAnswer struct {
	ID           uuid.UUID  `json:"id"`
	UserID       uuid.UUID  `json:"user_id"`
	FormKey      string     `json:"form_key"`
	QuestionCode string     `json:"question_code"`
	QuestionText string     `json:"question_text"`
	AnswerText   string     `json:"answer_text"`
	Position     int        `json:"position"`
	SubmittedAt  *time.Time `json:"submitted_at,omitempty"`
	Source       string     `json:"source"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type UpsertSurveyAnswerRequest struct {
	FormKey      string `json:"form_key"`
	QuestionCode string `json:"question_code" validate:"required,min=1,max=100"`
	QuestionText string `json:"question_text" validate:"required,min=1,max=1000"`
	AnswerText   string `json:"answer_text"   validate:"required,min=1"`
	Position     *int   `json:"position"`
}

// --- profile --------------------------------------------------------------

const profileCols = `user_id, education_levels, institution, specialty,
	prior_experience_band, career_goal, development_directions,
	mobility_readiness, relocation_readiness, internal_projects_readiness, teaching_readiness,
	professional_interests, learning_formats, learning_hours_band,
	submitted_at, source, created_at, updated_at`

func scanProfile(row pgx.Row) (*Profile, error) {
	p := &Profile{}
	err := row.Scan(&p.UserID, &p.EducationLevels, &p.Institution, &p.Specialty,
		&p.PriorExperienceBand, &p.CareerGoal, &p.DevelopmentDirections,
		&p.MobilityReadiness, &p.RelocationReadiness, &p.InternalProjectsReadiness, &p.TeachingReadiness,
		&p.ProfessionalInterests, &p.LearningFormats, &p.LearningHoursBand,
		&p.SubmittedAt, &p.Source, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// GetProfile returns ErrNotFound when the worker has no profile row yet —
// which is the normal state for anyone who never filled the questionnaire in.
func (r *Repository) GetProfile(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	p, err := scanProfile(r.pool.QueryRow(ctx,
		`SELECT `+profileCols+` FROM user_profile WHERE user_id = $1`, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

// UpsertProfile creates or replaces the whole profile row. Text arrays are
// normalized (trimmed, blanks dropped) so the UI can send whatever the tag
// widget produced without worrying about empty chips.
func (r *Repository) UpsertProfile(ctx context.Context, userID uuid.UUID, req UpsertProfileRequest) (*Profile, error) {
	return scanProfile(r.pool.QueryRow(ctx, `
		INSERT INTO user_profile (user_id, education_levels, institution, specialty,
			prior_experience_band, career_goal, development_directions,
			mobility_readiness, relocation_readiness, internal_projects_readiness, teaching_readiness,
			professional_interests, learning_formats, learning_hours_band, source)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,'manual')
		ON CONFLICT (user_id) DO UPDATE SET
			education_levels            = EXCLUDED.education_levels,
			institution                 = EXCLUDED.institution,
			specialty                   = EXCLUDED.specialty,
			prior_experience_band       = EXCLUDED.prior_experience_band,
			career_goal                 = EXCLUDED.career_goal,
			development_directions      = EXCLUDED.development_directions,
			mobility_readiness          = EXCLUDED.mobility_readiness,
			relocation_readiness        = EXCLUDED.relocation_readiness,
			internal_projects_readiness = EXCLUDED.internal_projects_readiness,
			teaching_readiness          = EXCLUDED.teaching_readiness,
			professional_interests      = EXCLUDED.professional_interests,
			learning_formats            = EXCLUDED.learning_formats,
			learning_hours_band         = EXCLUDED.learning_hours_band
		RETURNING `+profileCols,
		userID,
		cleanStrings(req.EducationLevels), blankToNil(req.Institution), blankToNil(req.Specialty),
		blankToNil(req.PriorExperienceBand), blankToNil(req.CareerGoal), cleanStrings(req.DevelopmentDirections),
		blankToNil(req.MobilityReadiness), blankToNil(req.RelocationReadiness),
		blankToNil(req.InternalProjectsReadiness), blankToNil(req.TeachingReadiness),
		cleanStrings(req.ProfessionalInterests), cleanStrings(req.LearningFormats),
		blankToNil(req.LearningHoursBand),
	))
}

func (r *Repository) DeleteProfile(ctx context.Context, userID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM user_profile WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- languages ------------------------------------------------------------

const languageCols = `id, user_id, language, level, source, created_at, updated_at`

func (r *Repository) ListLanguages(ctx context.Context, userID uuid.UUID) ([]Language, error) {
	// Order by CEFR strength so the strongest language reads first; the UI
	// relies on this rather than sorting client-side.
	rows, err := r.pool.Query(ctx, `
		SELECT `+languageCols+` FROM user_languages WHERE user_id = $1
		ORDER BY array_position(ARRAY['C2','C1','B2','B1','A2','A1'], level), lower(language)`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Language{}
	for rows.Next() {
		var l Language
		if err := rows.Scan(&l.ID, &l.UserID, &l.Language, &l.Level, &l.Source, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// UpsertLanguage is keyed on (user_id, lower(language)) so adding a language
// someone already has updates the level instead of failing.
func (r *Repository) UpsertLanguage(ctx context.Context, userID uuid.UUID, req UpsertLanguageRequest) (*Language, error) {
	l := &Language{}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO user_languages (user_id, language, level, source)
		VALUES ($1, $2, $3, 'manual')
		ON CONFLICT (user_id, lower(language)) DO UPDATE SET level = EXCLUDED.level
		RETURNING `+languageCols,
		userID, strings.TrimSpace(req.Language), req.Level,
	).Scan(&l.ID, &l.UserID, &l.Language, &l.Level, &l.Source, &l.CreatedAt, &l.UpdatedAt)
	return l, err
}

func (r *Repository) UpdateLanguage(ctx context.Context, id, userID uuid.UUID, req UpsertLanguageRequest) (*Language, error) {
	l := &Language{}
	err := r.pool.QueryRow(ctx, `
		UPDATE user_languages SET language = $3, level = $4
		WHERE id = $1 AND user_id = $2
		RETURNING `+languageCols,
		id, userID, strings.TrimSpace(req.Language), req.Level,
	).Scan(&l.ID, &l.UserID, &l.Language, &l.Level, &l.Source, &l.CreatedAt, &l.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return l, err
}

func (r *Repository) DeleteLanguage(ctx context.Context, id, userID uuid.UUID) error {
	return r.deleteScoped(ctx, `user_languages`, id, userID)
}

// --- work experience ------------------------------------------------------

const experienceCols = `id, user_id, company, position, started_on, ended_on,
	description, sort_order, raw_text, source, created_at, updated_at`

func scanExperience(row pgx.Row) (*WorkExperience, error) {
	e := &WorkExperience{}
	err := row.Scan(&e.ID, &e.UserID, &e.Company, &e.Position, &e.StartedOn, &e.EndedOn,
		&e.Description, &e.SortOrder, &e.RawText, &e.Source, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (r *Repository) ListWorkExperience(ctx context.Context, userID uuid.UUID) ([]WorkExperience, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+experienceCols+` FROM user_work_experience WHERE user_id = $1
		ORDER BY sort_order, started_on DESC NULLS LAST, created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []WorkExperience{}
	for rows.Next() {
		var e WorkExperience
		if err := rows.Scan(&e.ID, &e.UserID, &e.Company, &e.Position, &e.StartedOn, &e.EndedOn,
			&e.Description, &e.SortOrder, &e.RawText, &e.Source, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) CreateWorkExperience(ctx context.Context, userID uuid.UUID, req UpsertWorkExperienceRequest) (*WorkExperience, error) {
	started, ended, err := parseExperienceDates(req)
	if err != nil {
		return nil, err
	}
	return scanExperience(r.pool.QueryRow(ctx, `
		INSERT INTO user_work_experience (user_id, company, position, started_on, ended_on, description, sort_order, source)
		VALUES ($1,$2,$3,$4,$5,$6,COALESCE($7, 0),'manual')
		RETURNING `+experienceCols,
		userID, strings.TrimSpace(req.Company), blankToNil(req.Position),
		started, ended, blankToNil(req.Description), req.SortOrder,
	))
}

func (r *Repository) UpdateWorkExperience(ctx context.Context, id, userID uuid.UUID, req UpsertWorkExperienceRequest) (*WorkExperience, error) {
	started, ended, err := parseExperienceDates(req)
	if err != nil {
		return nil, err
	}
	e, err := scanExperience(r.pool.QueryRow(ctx, `
		UPDATE user_work_experience
		SET company = $3, position = $4, started_on = $5, ended_on = $6,
		    description = $7, sort_order = COALESCE($8, sort_order)
		WHERE id = $1 AND user_id = $2
		RETURNING `+experienceCols,
		id, userID, strings.TrimSpace(req.Company), blankToNil(req.Position),
		started, ended, blankToNil(req.Description), req.SortOrder,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return e, err
}

func (r *Repository) DeleteWorkExperience(ctx context.Context, id, userID uuid.UUID) error {
	return r.deleteScoped(ctx, `user_work_experience`, id, userID)
}

func parseExperienceDates(req UpsertWorkExperienceRequest) (*time.Time, *time.Time, error) {
	started, err := parseOptionalDate(req.StartedOn)
	if err != nil {
		return nil, nil, err
	}
	ended, err := parseOptionalDate(req.EndedOn)
	if err != nil {
		return nil, nil, err
	}
	if started != nil && ended != nil && ended.Before(*started) {
		return nil, nil, fmt.Errorf("%w: ended_on is before started_on", ErrInvalidInput)
	}
	return started, ended, nil
}

// --- survey answers -------------------------------------------------------

const surveyCols = `id, user_id, form_key, question_code, question_text, answer_text,
	position, submitted_at, source, created_at, updated_at`

func (r *Repository) ListSurveyAnswers(ctx context.Context, userID uuid.UUID) ([]SurveyAnswer, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+surveyCols+` FROM user_survey_answers WHERE user_id = $1
		ORDER BY position, question_code`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SurveyAnswer{}
	for rows.Next() {
		var a SurveyAnswer
		if err := rows.Scan(&a.ID, &a.UserID, &a.FormKey, &a.QuestionCode, &a.QuestionText,
			&a.AnswerText, &a.Position, &a.SubmittedAt, &a.Source, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repository) UpsertSurveyAnswer(ctx context.Context, userID uuid.UUID, req UpsertSurveyAnswerRequest) (*SurveyAnswer, error) {
	formKey := strings.TrimSpace(req.FormKey)
	if formKey == "" {
		formKey = "digital_profile"
	}
	a := &SurveyAnswer{}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO user_survey_answers (user_id, form_key, question_code, question_text, answer_text, position, source)
		VALUES ($1,$2,$3,$4,$5,COALESCE($6, 0),'manual')
		ON CONFLICT (user_id, form_key, question_code) DO UPDATE SET
			question_text = EXCLUDED.question_text,
			answer_text   = EXCLUDED.answer_text,
			position      = EXCLUDED.position
		RETURNING `+surveyCols,
		userID, formKey, strings.TrimSpace(req.QuestionCode), strings.TrimSpace(req.QuestionText),
		strings.TrimSpace(req.AnswerText), req.Position,
	).Scan(&a.ID, &a.UserID, &a.FormKey, &a.QuestionCode, &a.QuestionText,
		&a.AnswerText, &a.Position, &a.SubmittedAt, &a.Source, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

func (r *Repository) UpdateSurveyAnswer(ctx context.Context, id, userID uuid.UUID, req UpsertSurveyAnswerRequest) (*SurveyAnswer, error) {
	a := &SurveyAnswer{}
	err := r.pool.QueryRow(ctx, `
		UPDATE user_survey_answers
		SET question_text = $3, answer_text = $4, position = COALESCE($5, position)
		WHERE id = $1 AND user_id = $2
		RETURNING `+surveyCols,
		id, userID, strings.TrimSpace(req.QuestionText), strings.TrimSpace(req.AnswerText), req.Position,
	).Scan(&a.ID, &a.UserID, &a.FormKey, &a.QuestionCode, &a.QuestionText,
		&a.AnswerText, &a.Position, &a.SubmittedAt, &a.Source, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

func (r *Repository) DeleteSurveyAnswer(ctx context.Context, id, userID uuid.UUID) error {
	return r.deleteScoped(ctx, `user_survey_answers`, id, userID)
}

// --- helpers --------------------------------------------------------------

// deleteScoped removes a row by id, but only if it belongs to the given user,
// so a mistyped worker_id in the URL can never delete another person's row.
// The table name is never caller-supplied — all call sites pass a literal.
func (r *Repository) deleteScoped(ctx context.Context, table string, id, userID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM `+table+` WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// cleanStrings trims each element and drops the blanks, so a tag input that
// emitted a stray empty chip doesn't persist one.
func cleanStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// blankToNil turns a whitespace-only optional string into a NULL, so "cleared
// in the UI" and "never answered" are the same thing in the database.
func blankToNil(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func parseOptionalDate(s *string) (*time.Time, error) {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", strings.TrimSpace(*s))
	if err != nil {
		return nil, fmt.Errorf("%w: expected date as YYYY-MM-DD, got %q", ErrInvalidInput, *s)
	}
	return &t, nil
}
