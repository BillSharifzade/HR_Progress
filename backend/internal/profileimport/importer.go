package profileimport

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FormKey identifies this questionnaire in user_survey_answers.
const FormKey = "digital_profile"

// SourceForm marks rows the importer created, so a re-import can replace its
// own output without touching anything HR entered by hand.
const SourceForm = "form"

// outOfScopeDepartments are the self-reported departments whose people are not
// part of the platform. The filter applies ONLY to rows that failed to match an
// employee: someone who is in the system but mis-selected their department is
// still a real employee and still gets their profile.
var outOfScopeDepartments = map[string]bool{
	"ats":         true,
	"хирад":       true,
	"дусти фарма": true,
	"департамент инженерной экспертизы": true,
}

// Row is one questionnaire response, as produced by scripts/form_to_json.py.
type Row struct {
	SourceRow   int    `json:"source_row"`
	SubmittedAt string `json:"submitted_at"`
	FullName    string `json:"full_name"`
	Department  string `json:"department"`

	EducationLevels string `json:"education_levels"`
	Institution     string `json:"institution"`
	Specialty       string `json:"specialty"`
	Certificates    string `json:"certificates"`

	LangTajik   string `json:"lang_tajik"`
	LangRussian string `json:"lang_russian"`
	LangEnglish string `json:"lang_english"`
	LangChinese string `json:"lang_chinese"`
	LangGerman  string `json:"lang_german"`
	LangTurkish string `json:"lang_turkish"`

	PriorExperienceBand string `json:"prior_experience_band"`
	NotableProjects     string `json:"notable_projects"`
	PreviousEmployers   string `json:"previous_employers"`
	CareerGoal          string `json:"career_goal"`
	DevelopmentDirs     string `json:"development_directions"`

	MobilityReadiness  string `json:"mobility_readiness"`
	RelocationReady    string `json:"relocation_readiness"`
	InternalProjects   string `json:"internal_projects_readiness"`
	ExpertiseAreas     string `json:"expertise_areas"`
	ColleaguesAskAbout string `json:"colleagues_ask_about"`
	TeachingReadiness  string `json:"teaching_readiness"`
	TeachingTopics     string `json:"teaching_topics"`

	Hobbies               string `json:"hobbies"`
	ProfessionalInterests string `json:"professional_interests"`
	LearningFormats       string `json:"learning_formats"`
	LearningHoursBand     string `json:"learning_hours_band"`
	ExtraNote             string `json:"extra_note"`
}

// File is the parsed intermediate produced by the converter script.
type File struct {
	QuestionTexts map[string]string `json:"question_texts"`
	Rows          []Row             `json:"rows"`
}

// languageColumns maps a display name to the field holding its CEFR answer.
func (r Row) languageColumns() map[string]string {
	return map[string]string{
		"Таджикский": r.LangTajik,
		"Русский":    r.LangRussian,
		"Английский": r.LangEnglish,
		"Китайский":  r.LangChinese,
		"Немецкий":   r.LangGerman,
		"Турецкий":   r.LangTurkish,
	}
}

// openEndedAnswers are the questions with no structured profile field; they
// become «Результаты опросов» entries.
func (r Row) openEndedAnswers() []struct{ Code, Value string } {
	return []struct{ Code, Value string }{
		{"notable_projects", r.NotableProjects},
		{"expertise_areas", r.ExpertiseAreas},
		{"colleagues_ask_about", r.ColleaguesAskAbout},
		{"teaching_topics", r.TeachingTopics},
		{"extra_note", r.ExtraNote},
	}
}

// hasAnyAnswer reports whether the row carries anything worth importing.
// Rows that are entirely empty are skipped outright, per the agreed rule.
func (r Row) hasAnyAnswer() bool {
	fields := []string{
		r.EducationLevels, r.Institution, r.Specialty, r.Certificates,
		r.PriorExperienceBand, r.NotableProjects, r.PreviousEmployers, r.CareerGoal,
		r.DevelopmentDirs, r.MobilityReadiness, r.RelocationReady, r.InternalProjects,
		r.ExpertiseAreas, r.ColleaguesAskAbout, r.TeachingReadiness, r.TeachingTopics,
		r.Hobbies, r.ProfessionalInterests, r.LearningFormats, r.LearningHoursBand, r.ExtraNote,
	}
	for _, f := range fields {
		if !IsBlankAnswer(f) {
			return true
		}
	}
	for _, v := range r.languageColumns() {
		if HighestCEFR(v) != "" {
			return true
		}
	}
	return false
}

// RowOutcome records what happened to one form row.
type RowOutcome struct {
	SourceRow  int
	FormName   string
	Department string
	Status     MatchStatus
	Skipped    string // non-empty when deliberately not imported
	UserID     uuid.UUID
	MatchName  string
	Score      float64
	Candidates []Candidate

	Languages   int
	Employments int
	Answers     int
	Certs       int
	HobbySet    bool
}

// Report summarises an import run.
type Report struct {
	Outcomes []RowOutcome
	DryRun   bool
}

func (rep Report) countBy(pred func(RowOutcome) bool) int {
	n := 0
	for _, o := range rep.Outcomes {
		if pred(o) {
			n++
		}
	}
	return n
}

// String renders a human-readable summary, which is what actually gets
// reviewed before a commit run.
func (rep Report) String() string {
	var b strings.Builder
	mode := "COMMIT"
	if rep.DryRun {
		mode = "DRY RUN — nothing written"
	}
	fmt.Fprintf(&b, "\n=== Profile import (%s) ===\n", mode)

	imported := rep.countBy(func(o RowOutcome) bool { return o.Skipped == "" && o.Status != MatchUnresolved })
	fmt.Fprintf(&b, "rows in file      : %d\n", len(rep.Outcomes))
	fmt.Fprintf(&b, "imported          : %d\n", imported)
	for _, st := range []MatchStatus{MatchExact, MatchPartial, MatchFuzzy} {
		if n := rep.countBy(func(o RowOutcome) bool { return o.Skipped == "" && o.Status == st }); n > 0 {
			fmt.Fprintf(&b, "  %-14s: %d\n", st, n)
		}
	}
	fmt.Fprintf(&b, "skipped           : %d\n", rep.countBy(func(o RowOutcome) bool { return o.Skipped != "" }))
	fmt.Fprintf(&b, "UNRESOLVED        : %d\n",
		rep.countBy(func(o RowOutcome) bool { return o.Skipped == "" && o.Status == MatchUnresolved }))

	var langs, emps, answers, certs, hobbies int
	for _, o := range rep.Outcomes {
		langs += o.Languages
		emps += o.Employments
		answers += o.Answers
		certs += o.Certs
		if o.HobbySet {
			hobbies++
		}
	}
	fmt.Fprintf(&b, "\nrows written: languages=%d  employment=%d  survey answers=%d  certificate links=%d  hobbies=%d\n",
		langs, emps, answers, certs, hobbies)

	// Skipped rows, grouped by reason.
	byReason := map[string][]RowOutcome{}
	for _, o := range rep.Outcomes {
		if o.Skipped != "" {
			byReason[o.Skipped] = append(byReason[o.Skipped], o)
		}
	}
	reasons := make([]string, 0, len(byReason))
	for k := range byReason {
		reasons = append(reasons, k)
	}
	sort.Strings(reasons)
	for _, reason := range reasons {
		fmt.Fprintf(&b, "\n--- skipped: %s (%d) ---\n", reason, len(byReason[reason]))
		for _, o := range byReason[reason] {
			fmt.Fprintf(&b, "  row %-4d %-40s %s\n", o.SourceRow, trimTo(o.FormName, 40), trimTo(o.Department, 45))
		}
	}

	// Unresolved rows with their best candidates — this is the list a human
	// has to act on.
	unresolved := rep.countBy(func(o RowOutcome) bool { return o.Skipped == "" && o.Status == MatchUnresolved })
	if unresolved > 0 {
		fmt.Fprintf(&b, "\n--- UNRESOLVED, needs a human (%d) ---\n", unresolved)
		for _, o := range rep.Outcomes {
			if o.Skipped != "" || o.Status != MatchUnresolved {
				continue
			}
			fmt.Fprintf(&b, "  row %-4d %-40s  %s\n", o.SourceRow, trimTo(o.FormName, 40), trimTo(o.Department, 40))
			for _, c := range o.Candidates {
				fmt.Fprintf(&b, "           candidate %.3f  %s\n", c.Score, c.FullName)
			}
		}
	}

	// Fuzzy matches are imported but worth eyeballing.
	if n := rep.countBy(func(o RowOutcome) bool { return o.Skipped == "" && o.Status == MatchFuzzy }); n > 0 {
		fmt.Fprintf(&b, "\n--- FUZZY matches, imported but worth a look (%d) ---\n", n)
		for _, o := range rep.Outcomes {
			if o.Skipped == "" && o.Status == MatchFuzzy {
				fmt.Fprintf(&b, "  row %-4d %.3f  %-38s → %s\n",
					o.SourceRow, o.Score, trimTo(o.FormName, 38), o.MatchName)
			}
		}
	}
	return b.String()
}

func trimTo(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// Importer writes questionnaire answers into profile tables.
type Importer struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Importer { return &Importer{pool: pool} }

// LoadFile reads the JSON intermediate.
func LoadFile(path string) (*File, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f File
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &f, nil
}

func (im *Importer) loadEmployees(ctx context.Context) ([]Employee, error) {
	rows, err := im.pool.Query(ctx,
		`SELECT id, full_name FROM users WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Employee
	for rows.Next() {
		var e Employee
		if err := rows.Scan(&e.ID, &e.FullName); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Run matches every row and, unless dryRun, writes the results in a single
// transaction. A dry run performs the identical matching so its counts are the
// real ones, then rolls back.
func (im *Importer) Run(ctx context.Context, f *File, dryRun bool) (*Report, error) {
	employees, err := im.loadEmployees(ctx)
	if err != nil {
		return nil, fmt.Errorf("load employees: %w", err)
	}

	tx, err := im.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback is the dry-run path

	rep := &Report{DryRun: dryRun}
	byName := map[string]Employee{}
	for _, e := range employees {
		byName[e.ID.String()] = e
	}

	for _, row := range f.Rows {
		out := RowOutcome{SourceRow: row.SourceRow, FormName: row.FullName, Department: row.Department}

		if !row.hasAnyAnswer() {
			out.Skipped = "no answers at all"
			rep.Outcomes = append(rep.Outcomes, out)
			continue
		}

		m := MatchName(row.FullName, employees)
		out.Status, out.Score, out.Candidates = m.Status, m.Score, m.Candidates

		if m.Status == MatchUnresolved {
			// The department filter only applies to people we could not find:
			// an out-of-scope department plus no match means they are simply
			// not our employee.
			if outOfScopeDepartments[strings.ToLower(strings.TrimSpace(row.Department))] {
				out.Skipped = "out-of-scope department, no matching employee"
			}
			rep.Outcomes = append(rep.Outcomes, out)
			continue
		}

		out.UserID = m.UserID
		out.MatchName = byName[m.UserID.String()].FullName

		if err := im.writeRow(ctx, tx, row, m.UserID, &out); err != nil {
			return nil, fmt.Errorf("row %d (%s): %w", row.SourceRow, row.FullName, err)
		}
		rep.Outcomes = append(rep.Outcomes, out)
	}

	if dryRun {
		return rep, nil // deferred rollback discards everything
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return rep, nil
}

func (im *Importer) writeRow(ctx context.Context, tx pgx.Tx, row Row, userID uuid.UUID, out *RowOutcome) error {
	submitted := parseSubmitted(row.SubmittedAt)

	// --- hobbies: fill only when empty, so an HR edit is never clobbered ---
	if !IsBlankAnswer(row.Hobbies) {
		tag, err := tx.Exec(ctx, `
			UPDATE users SET hobbies = $2
			WHERE id = $1 AND (hobbies IS NULL OR btrim(hobbies) = '')`,
			userID, row.Hobbies)
		if err != nil {
			return fmt.Errorf("hobbies: %w", err)
		}
		out.HobbySet = tag.RowsAffected() > 0
	}

	// --- structured profile ---
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_profile (user_id, education_levels, institution, specialty,
			prior_experience_band, career_goal, development_directions,
			mobility_readiness, relocation_readiness, internal_projects_readiness, teaching_readiness,
			professional_interests, learning_formats, learning_hours_band,
			submitted_at, source)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
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
			learning_hours_band         = EXCLUDED.learning_hours_band,
			submitted_at                = EXCLUDED.submitted_at,
			source                      = EXCLUDED.source`,
		userID,
		SplitOptions(row.EducationLevels), nilIfBlank(row.Institution), nilIfBlank(row.Specialty),
		nilIfBlank(row.PriorExperienceBand), nilIfBlank(row.CareerGoal), SplitOptions(row.DevelopmentDirs),
		nilIfBlank(row.MobilityReadiness), nilIfBlank(row.RelocationReady),
		nilIfBlank(row.InternalProjects), nilIfBlank(row.TeachingReadiness),
		SplitOptions(row.ProfessionalInterests), SplitOptions(row.LearningFormats),
		nilIfBlank(row.LearningHoursBand), submitted, SourceForm,
	); err != nil {
		return fmt.Errorf("profile: %w", err)
	}

	// --- languages ---
	for name, answer := range row.languageColumns() {
		level := HighestCEFR(answer)
		if level == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_languages (user_id, language, level, source)
			VALUES ($1,$2,$3,$4)
			ON CONFLICT (user_id, lower(language)) DO UPDATE SET level = EXCLUDED.level`,
			userID, name, level, SourceForm); err != nil {
			return fmt.Errorf("language %s: %w", name, err)
		}
		out.Languages++
	}

	// --- work experience: replace this importer's own rows only ---
	if _, err := tx.Exec(ctx,
		`DELETE FROM user_work_experience WHERE user_id = $1 AND source = $2`,
		userID, SourceForm); err != nil {
		return fmt.Errorf("clear employment: %w", err)
	}
	for i, e := range ParseEmployment(row.PreviousEmployers) {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_work_experience
				(user_id, company, position, description, sort_order, raw_text, source)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			userID, e.Company, nilIfBlank(e.Position), nilIfBlank(e.Description),
			i, e.Raw, SourceForm); err != nil {
			return fmt.Errorf("employment %q: %w", e.Company, err)
		}
		out.Employments++
	}

	// --- open-ended answers ---
	for i, qa := range row.openEndedAnswers() {
		if IsBlankAnswer(qa.Value) {
			continue
		}
		questionText := questionTextFor(qa.Code)
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_survey_answers
				(user_id, form_key, question_code, question_text, answer_text, position, submitted_at, source)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (user_id, form_key, question_code) DO UPDATE SET
				question_text = EXCLUDED.question_text,
				answer_text   = EXCLUDED.answer_text,
				position      = EXCLUDED.position,
				submitted_at  = EXCLUDED.submitted_at,
				source        = EXCLUDED.source`,
			userID, FormKey, qa.Code, questionText, qa.Value, i, submitted, SourceForm); err != nil {
			return fmt.Errorf("answer %s: %w", qa.Code, err)
		}
		out.Answers++
	}

	// --- certificate links (idempotent on the URL) ---
	for _, u := range CertificateURLs(row.Certificates) {
		tag, err := tx.Exec(ctx, `
			INSERT INTO user_certifications (user_id, title, source_url, source)
			SELECT $1, $2, $3, $4
			WHERE NOT EXISTS (
				SELECT 1 FROM user_certifications
				WHERE user_id = $1 AND source_url = $3
			)`,
			userID, "Сертификат из анкеты", u, SourceForm)
		if err != nil {
			return fmt.Errorf("certificate: %w", err)
		}
		if tag.RowsAffected() > 0 {
			out.Certs++
		}
	}
	return nil
}

// questionTexts are the wordings shown in «Результаты опросов». Kept short and
// readable rather than reusing the raw spreadsheet headers, which carry stray
// whitespace and trailing spaces from the export.
var questionTexts = map[string]string{
	"notable_projects":     "В каких наиболее значимых проектах Вы принимали участие?",
	"expertise_areas":      "В каких профессиональных областях Вы считаете себя наиболее компетентным?",
	"colleagues_ask_about": "По каким вопросам к Вам чаще всего обращаются коллеги?",
	"teaching_topics":      "По каким темам Вы могли бы провести внутреннее обучение?",
	"extra_note":           "Дополнительная информация для профиля",
}

func questionTextFor(code string) string {
	if t, ok := questionTexts[code]; ok {
		return t
	}
	return code
}

func nilIfBlank(s string) *string {
	if IsBlankAnswer(s) {
		return nil
	}
	t := strings.TrimSpace(s)
	return &t
}

func parseSubmitted(s string) *time.Time {
	if s == "" {
		return nil
	}
	for _, layout := range []string{
		"2006-01-02T15:04:05.999999", "2006-01-02T15:04:05", "2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}
