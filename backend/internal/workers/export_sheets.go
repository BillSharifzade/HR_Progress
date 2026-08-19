package workers

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ─── value helpers ──────────────────────────────────────────────────────────

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// str returns nil for an absent value so the cell stays genuinely empty:
// Excel's filters count blanks, but would treat a placeholder dash as data.
func str(p *string) any {
	if p == nil || strings.TrimSpace(*p) == "" {
		return nil
	}
	return strings.TrimSpace(*p)
}

func text(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func firstStr(ps ...*string) any {
	for _, p := range ps {
		if v := str(p); v != nil {
			return v
		}
	}
	return nil
}

func int64Val(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

func intVal(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

// countVal blanks a zero. Used only where a zero would be noise (a department
// with no sections); per-person counts stay numeric so a column of them can be
// averaged without blanks skewing the result.
func countVal(n int) any {
	if n == 0 {
		return nil
	}
	return n
}

// dateVal renders a stored DATE. Postgres hands these over as UTC midnight, so
// the wall-clock fields are reused as-is — converting to local time would move
// the day for anyone west of Greenwich.
func dateVal(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// stampVal renders a timestamptz in local time. The wall clock is carried in a
// UTC-located value because that is what excelize serialises verbatim.
func stampVal(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	l := t.Local()
	return time.Date(l.Year(), l.Month(), l.Day(), l.Hour(), l.Minute(), 0, 0, time.UTC)
}

func listVal(items []string) any {
	clean := make([]string, 0, len(items))
	for _, s := range items {
		if s = strings.TrimSpace(s); s != "" {
			clean = append(clean, s)
		}
	}
	if len(clean) == 0 {
		return nil
	}
	return strings.Join(clean, ", ")
}

// yearsSince is the elapsed time in whole-and-tenths of years, so tenure and
// age stay numeric and therefore sortable inside Excel.
func yearsSince(t *time.Time, until time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	days := until.Sub(*t).Hours() / 24
	if days < 0 {
		return nil
	}
	return float64(int(days/365.2425*10+0.5)) / 10
}

func ageVal(birth *time.Time, until time.Time) any {
	if birth == nil || birth.IsZero() {
		return nil
	}
	age := until.Year() - birth.Year()
	if until.YearDay() < birth.YearDay() {
		age--
	}
	if age < 0 || age > 120 {
		return nil
	}
	return age
}

func roleLabels(roles []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, r := range roles {
		label := userRoleLabels[r]
		if label == "" {
			label = r
		}
		if !seen[label] {
			seen[label] = true
			out = append(out, label)
		}
	}
	return out
}

func languageSummary(langs []Language) any {
	parts := make([]string, 0, len(langs))
	for _, l := range langs {
		parts = append(parts, l.Language+" "+l.Level)
	}
	return listVal(parts)
}

// cefrColors maps a CEFR level onto the fill/text pair used in the language
// matrix: green for fluent, blue for independent, amber for basic.
func cefrColors(level string) (fg, bg string) {
	switch level {
	case "C1", "C2":
		return clrPosText, clrPosFill
	case "B1", "B2":
		return clrMidText, clrMidFill
	case "A1", "A2":
		return clrLowText, clrLowFill
	}
	return clrInk, clrBandA
}

func positionOf(w *ExportWorker) any { return firstStr(w.PositionName, w.Position) }
func deptOf(w *ExportWorker) any     { return str(w.DepartmentName) }
func sectionOf(w *ExportWorker) any  { return str(w.SectionName) }
func statusText(active bool) string {
	if active {
		return "Активен"
	}
	return "Неактивен"
}

// idCols are the four columns every detail sheet opens with, so a row can
// always be traced back to a person and their place in the org.
var idCols = []col{
	{"№", 6, "center", "int", false},
	{"ФИО", 30, "", "", false},
	{"Департамент", 30, "", "", false},
	{"Отдел", 26, "", "", false},
}

func idVals(n int, w *ExportWorker) []any {
	return []any{n, w.FullName, deptOf(w), sectionOf(w)}
}

// ─── Сотрудники ─────────────────────────────────────────────────────────────

func buildRegister(c *buildCtx) (string, int, error) {
	cols := []col{
		{"№", 6, "center", "int", false},
		{"ФИО", 32, "", "", false},
		{"Таб. номер", 12, "center", "", false},
		{"ID", 8, "center", "int", false},
		{"Департамент", 32, "", "", false},
		{"Отдел", 28, "", "", false},
		{"Должность", 32, "", "", false},
		{"Грейд", 26, "", "", false},
		{"Ур.", 6, "center", "int", false},
		{"Статус", 12, "center", "", false},
		{"Дата приёма", 13, "center", "date", false},
		{"Стаж, лет", 10, "center", "num1", false},
		{"Дата рождения", 14, "center", "date", false},
		{"Возраст", 9, "center", "int", false},
		{"Email", 28, "", "", false},
		{"Телефон", 16, "", "", false},
		{"Telegram ID", 13, "center", "int", false},
		{"Логин", 18, "", "", false},
		{"Специализация", 22, "", "", false},
		{"Роли в системе", 30, "", "", false},
		{"Руководитель", 26, "", "", false},
		{"Языки", 32, "", "", false},
		{"Анкета", 13, "center", "date", false},
		{"Серти-\nфикатов", 10, "center", "int", false},
		{"Мест\nработы", 10, "center", "int", false},
		{"Оце-\nнок", 8, "center", "int", false},
		{"Последняя\nоценка", 14, "center", "date", false},
		{"1F ID", 9, "center", "int", false},
		{"Синхронизация 1F", 17, "center", "datetime", false},
		{"Хобби и увлечения", 34, "", "", false},
	}
	statusCol := colIndex(cols, "Статус")
	s, err := c.b.newSheet(shRegister, "Реестр сотрудников", c.sub, cols, 2)
	if err != nil {
		return shRegister, 0, err
	}
	if len(c.ds.Workers) == 0 {
		s.empty("По заданным фильтрам сотрудники не найдены")
		return shRegister, 0, nil
	}

	now := c.ds.GeneratedAt
	for i := range c.ds.Workers {
		w := &c.ds.Workers[i]
		var submitted *time.Time
		if w.Profile != nil {
			submitted = w.Profile.SubmittedAt
		}
		row := s.addRow(c.bands[i],
			i+1,
			w.FullName,
			str(w.PersonnelNumber),
			w.EmployeeNo,
			deptOf(w),
			sectionOf(w),
			positionOf(w),
			str(w.GradeName),
			intVal(w.GradeLevel),
			statusText(w.IsActive),
			dateVal(w.HiredAt),
			yearsSince(w.HiredAt, now),
			dateVal(w.BirthDate),
			ageVal(w.BirthDate, now),
			str(w.Email),
			str(w.PhoneNumber),
			int64Val(w.TelegramID),
			text(w.Username),
			str(w.Specialization),
			listVal(roleLabels(w.Roles)),
			str(w.ManagerName),
			languageSummary(w.Languages),
			dateVal(submitted),
			len(w.Certifications),
			len(w.Experience),
			len(w.Scores),
			dateVal(w.LastAssessmentAt),
			int64Val(w.OneFUserID),
			stampVal(w.LastSyncedAt),
			str(w.Hobbies),
		)
		if w.IsActive {
			s.paint(row, statusCol, clrPosText, clrPosFill)
		} else {
			s.paint(row, statusCol, clrOffText, clrBandB)
		}
	}
	return shRegister, len(c.ds.Workers), nil
}

// ─── Цифровой профиль ───────────────────────────────────────────────────────

func buildProfileSheet(c *buildCtx) (string, int, error) {
	cols := append(append([]col{}, idCols...), []col{
		{"Уровень образования", 26, "", "", true},
		{"Учебное заведение", 34, "", "", true},
		{"Специальность", 30, "", "", true},
		{"Стаж до компании", 16, "", "", false},
		{"Карьерная цель", 34, "", "", true},
		{"Направления развития", 34, "", "", true},
		{"Переход в другой департамент", 22, "", "", true},
		{"Готовность к релокации", 18, "", "", true},
		{"Внутренние проекты", 18, "", "", true},
		{"Готов обучать коллег", 18, "", "", true},
		{"Профессиональные интересы", 38, "", "", true},
		{"Форматы обучения", 34, "", "", true},
		{"Часов в месяц", 14, "", "", false},
		{"Дата анкеты", 13, "center", "date", false},
		{"Источник", 12, "center", "", false},
	}...)
	s, err := c.b.newSheet(shProfile, "Цифровой профиль сотрудника",
		c.sub+" · заполнено анкет: "+fmt.Sprint(countProfiles(c.ds)), cols, 2)
	if err != nil {
		return shProfile, 0, err
	}

	n := 0
	for i := range c.ds.Workers {
		w := &c.ds.Workers[i]
		if w.Profile == nil {
			continue
		}
		p := w.Profile
		vals := append(idVals(i+1, w),
			listVal(p.EducationLevels),
			str(p.Institution),
			str(p.Specialty),
			str(p.PriorExperienceBand),
			str(p.CareerGoal),
			listVal(p.DevelopmentDirections),
			str(p.MobilityReadiness),
			str(p.RelocationReadiness),
			str(p.InternalProjectsReadiness),
			str(p.TeachingReadiness),
			listVal(p.ProfessionalInterests),
			listVal(p.LearningFormats),
			str(p.LearningHoursBand),
			dateVal(p.SubmittedAt),
			sourceLabel(p.Source),
		)
		s.addRow(c.bands[i], vals...)
		n++
	}
	if n == 0 {
		s.empty("Ни один сотрудник из выгрузки не заполнил анкету цифрового профиля")
	}
	return shProfile, n, nil
}

func countProfiles(ds *ExportDataset) int {
	n := 0
	for i := range ds.Workers {
		if ds.Workers[i].Profile != nil {
			n++
		}
	}
	return n
}

func sourceLabel(src string) any {
	switch src {
	case "form":
		return "Анкета"
	case "manual":
		return "Вручную"
	case "1f", "onef":
		return "1Ф"
	}
	return text(src)
}

// ─── Языки ──────────────────────────────────────────────────────────────────

// buildLanguageSheet renders languages as a matrix rather than a list: one row
// per person, one column per language, the CEFR level in the cell. That is the
// shape HR actually reads it in — "who speaks English at B2 or better" is a
// column filter instead of a pivot.
func buildLanguageSheet(c *buildCtx) (string, int, error) {
	langs := languageColumns(c.ds)

	cols := append([]col{}, idCols...)
	for _, l := range langs {
		cols = append(cols, col{l, 14, "center", "", false})
	}
	cols = append(cols, col{"Всего\nязыков", 10, "center", "int", false})

	s, err := c.b.newSheet(shLanguages, "Владение языками",
		c.sub+" · языков в справочнике: "+fmt.Sprint(len(langs)), cols, 2)
	if err != nil {
		return shLanguages, 0, err
	}
	if len(langs) == 0 {
		s.empty("Данные о владении языками отсутствуют")
		return shLanguages, 0, nil
	}

	idx := map[string]int{}
	for i, l := range langs {
		idx[strings.ToLower(l)] = i
	}

	n := 0
	for i := range c.ds.Workers {
		w := &c.ds.Workers[i]
		if len(w.Languages) == 0 {
			continue
		}
		cells := make([]any, len(langs))
		levels := make([]string, len(langs))
		for _, l := range w.Languages {
			if j, ok := idx[strings.ToLower(l.Language)]; ok {
				cells[j] = l.Level
				levels[j] = l.Level
			}
		}
		vals := append(idVals(i+1, w), cells...)
		vals = append(vals, len(w.Languages))
		row := s.addRow(c.bands[i], vals...)
		for j, lvl := range levels {
			if lvl == "" {
				continue
			}
			fg, bg := cefrColors(lvl)
			s.paint(row, len(idCols)+j, fg, bg)
		}
		n++
	}
	if n == 0 {
		s.empty("Данные о владении языками отсутствуют")
	}
	return shLanguages, n, nil
}

// languageColumns lists every language present in the dataset, keeping the
// order the questionnaire used and appending anything else alphabetically.
func languageColumns(ds *ExportDataset) []string {
	seen := map[string]string{}
	for i := range ds.Workers {
		for _, l := range ds.Workers[i].Languages {
			key := strings.ToLower(l.Language)
			if _, ok := seen[key]; !ok {
				seen[key] = l.Language
			}
		}
	}
	out := []string{}
	for _, known := range knownLanguages {
		if name, ok := seen[strings.ToLower(known)]; ok {
			out = append(out, name)
			delete(seen, strings.ToLower(known))
		}
	}
	rest := []string{}
	for _, name := range seen {
		rest = append(rest, name)
	}
	sort.Slice(rest, func(i, j int) bool { return strings.ToLower(rest[i]) < strings.ToLower(rest[j]) })
	return append(out, rest...)
}

// ─── Опыт работы ────────────────────────────────────────────────────────────

func buildExperienceSheet(c *buildCtx) (string, int, error) {
	cols := append(append([]col{}, idCols...), []col{
		{"Компания", 38, "", "", true},
		{"Должность", 30, "", "", true},
		{"Начало", 13, "center", "date", false},
		{"Окончание", 13, "center", "date", false},
		{"Описание", 52, "", "", true},
		{"Источник", 12, "center", "", false},
	}...)
	s, err := c.b.newSheet(shExperience, "Опыт работы до компании", c.sub, cols, 2)
	if err != nil {
		return shExperience, 0, err
	}

	n := 0
	for i := range c.ds.Workers {
		w := &c.ds.Workers[i]
		for _, e := range w.Experience {
			s.addRow(c.bands[i], append(idVals(i+1, w),
				text(e.Company), str(e.Position),
				dateVal(e.StartedOn), dateVal(e.EndedOn),
				str(e.Description), sourceLabel(e.Source),
			)...)
			n++
		}
	}
	if n == 0 {
		s.empty("Записи о предыдущем опыте работы отсутствуют")
	}
	return shExperience, n, nil
}

// ─── Сертификаты ────────────────────────────────────────────────────────────

func buildCertSheet(c *buildCtx) (string, int, error) {
	cols := append(append([]col{}, idCols...), []col{
		{"Название", 44, "", "", true},
		{"Кем выдан", 28, "", "", true},
		{"Дата выдачи", 13, "center", "date", false},
		{"Действителен до", 15, "center", "date", false},
		{"Ссылка", 46, "", "", false},
		{"Файл", 28, "", "", false},
		{"Источник", 12, "center", "", false},
	}...)
	s, err := c.b.newSheet(shCerts, "Сертификаты и подтверждённая квалификация", c.sub, cols, 2)
	if err != nil {
		return shCerts, 0, err
	}

	n := 0
	for i := range c.ds.Workers {
		w := &c.ds.Workers[i]
		for _, cert := range w.Certifications {
			s.addRow(c.bands[i], append(idVals(i+1, w),
				text(cert.Title), str(cert.IssuedBy),
				dateVal(cert.IssuedAt), dateVal(cert.ExpiresAt),
				str(cert.SourceURL), str(cert.FileName), sourceLabel(cert.Source),
			)...)
			n++
		}
	}
	if n == 0 {
		s.empty("Сертификаты не загружены")
	}
	return shCerts, n, nil
}

// ─── Анкета ─────────────────────────────────────────────────────────────────

// buildSurveySheet pivots the open-ended questionnaire answers into one column
// per question, so the sheet reads as a comparison across people rather than as
// a log of answers.
func buildSurveySheet(c *buildCtx) (string, int, error) {
	questions := surveyQuestions(c.ds)

	cols := append([]col{}, idCols...)
	for range questions {
		cols = append(cols, col{"", 52, "", "", true})
	}
	for i, q := range questions {
		cols[len(idCols)+i].title = q.text
	}

	s, err := c.b.newSheet(shSurvey, "Открытые ответы анкеты", c.sub, cols, 2)
	if err != nil {
		return shSurvey, 0, err
	}
	if len(questions) == 0 {
		s.empty("Открытые ответы анкеты отсутствуют")
		return shSurvey, 0, nil
	}

	idx := map[string]int{}
	for i, q := range questions {
		idx[q.code] = i
	}

	n := 0
	for i := range c.ds.Workers {
		w := &c.ds.Workers[i]
		if len(w.Survey) == 0 {
			continue
		}
		cells := make([]any, len(questions))
		for _, a := range w.Survey {
			if j, ok := idx[a.QuestionCode]; ok {
				cells[j] = a.AnswerText
			}
		}
		s.addRow(c.bands[i], append(idVals(i+1, w), cells...)...)
		n++
	}
	if n == 0 {
		s.empty("Открытые ответы анкеты отсутствуют")
	}
	return shSurvey, n, nil
}

type surveyQuestion struct {
	code, text string
	position   int
}

func surveyQuestions(ds *ExportDataset) []surveyQuestion {
	seen := map[string]surveyQuestion{}
	for i := range ds.Workers {
		for _, a := range ds.Workers[i].Survey {
			if _, ok := seen[a.QuestionCode]; !ok {
				seen[a.QuestionCode] = surveyQuestion{a.QuestionCode, a.QuestionText, a.Position}
			}
		}
	}
	out := make([]surveyQuestion, 0, len(seen))
	for _, q := range seen {
		out = append(out, q)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].position != out[j].position {
			return out[i].position < out[j].position
		}
		return out[i].code < out[j].code
	})
	return out
}

// ─── Оценки ─────────────────────────────────────────────────────────────────

func buildScoreSheet(c *buildCtx) (string, int, error) {
	cols := append(append([]col{}, idCols...), []col{
		{"Кампания", 30, "", "", true},
		{"Период", 22, "center", "", false},
		{"Статус", 14, "center", "", false},
		{"Компетенция", 34, "", "", true},
		{"Тип", 18, "", "", false},
		{"Код", 10, "center", "", false},
		{"Оценщик (роль)", 18, "", "", false},
		{"Балл", 9, "center", "num1", false},
		{"Оценил", 26, "", "", false},
		{"Дата оценки", 13, "center", "date", false},
		{"Интерпретация", 60, "", "", true},
	}...)
	scoreCol := colIndex(cols, "Балл")
	s, err := c.b.newSheet(shScores, "Оценки компетенций", c.sub, cols, 2)
	if err != nil {
		return shScores, 0, err
	}

	n := 0
	for i := range c.ds.Workers {
		w := &c.ds.Workers[i]
		for _, sc := range w.Scores {
			period := sc.PeriodStart.Format("02.01.2006") + " – " + sc.PeriodEnd.Format("02.01.2006")
			var score any
			if sc.Score != nil {
				score = *sc.Score
			}
			row := s.addRow(c.bands[i], append(idVals(i+1, w),
				text(sc.PeriodTitle), period, labelOr(campaignStatusLabels, sc.PeriodStatus),
				text(sc.CompetencyName), labelOr(competencyKindLabels, sc.CompetencyKind),
				text(sc.CompetencyCode), labelOr(assessorRoleLabels, sc.AssessorRole),
				score, str(sc.AssessedByName), dateVal(sc.AssessedAt), str(sc.Feedback),
			)...)
			if sc.Score != nil {
				fg, bg := scoreColors(*sc.Score)
				s.paint(row, scoreCol, fg, bg)
			}
			n++
		}
	}
	if n == 0 {
		s.empty("Оценки компетенций по этим сотрудникам ещё не выставлены")
	}
	return shScores, n, nil
}

// scoreColors bands the 0–10 scale the way the matrix screen does: 7 and above
// reads as strong, 4–7 as developing, below 4 as a gap.
func scoreColors(score float64) (fg, bg string) {
	switch {
	case score >= 7:
		return clrPosText, clrPosFill
	case score >= 4:
		return clrMidText, clrMidFill
	default:
		return clrLowText, clrLowFill
	}
}

func labelOr(m map[string]string, key string) any {
	if v, ok := m[key]; ok {
		return v
	}
	return text(key)
}

// ─── Роли и доступ ──────────────────────────────────────────────────────────

func buildRoleSheet(c *buildCtx) (string, int, error) {
	cols := append(append([]col{}, idCols...), []col{
		{"Роль", 30, "", "", false},
		{"Область: департамент", 30, "", "", false},
		{"Область: отдел", 28, "", "", false},
		{"Кем выдана", 26, "", "", false},
		{"Дата выдачи", 17, "center", "datetime", false},
	}...)
	s, err := c.b.newSheet(shRoles, "Роли и права доступа", c.sub, cols, 2)
	if err != nil {
		return shRoles, 0, err
	}

	n := 0
	for i := range c.ds.Workers {
		w := &c.ds.Workers[i]
		for _, a := range w.RoleAssignments {
			granted := a.GrantedAt
			s.addRow(c.bands[i], append(idVals(i+1, w),
				labelOr(userRoleLabels, a.Role),
				str(a.ScopeDepartment), str(a.ScopeSection),
				str(a.GrantedByName), stampVal(&granted),
			)...)
			n++
		}
	}
	if n == 0 {
		s.empty("Роли в системе никому из выгрузки не назначены")
	}
	return shRoles, n, nil
}

// ─── История ────────────────────────────────────────────────────────────────

func buildHistorySheet(c *buildCtx) (string, int, error) {
	cols := append(append([]col{}, idCols...), []col{
		{"Событие", 20, "", "", false},
		{"Дата", 13, "center", "date", false},
		{"Заголовок", 44, "", "", true},
		{"Описание", 60, "", "", true},
	}...)
	s, err := c.b.newSheet(shHistory, "История в компании", c.sub, cols, 2)
	if err != nil {
		return shHistory, 0, err
	}

	n := 0
	for i := range c.ds.Workers {
		w := &c.ds.Workers[i]
		for _, h := range w.History {
			eventDate := h.EventDate
			s.addRow(c.bands[i], append(idVals(i+1, w),
				labelOr(historyKindLabels, h.EventKind),
				dateVal(&eventDate), text(h.Title), str(h.Description),
			)...)
			n++
		}
	}
	if n == 0 {
		s.empty("Записи о движении внутри компании отсутствуют")
	}
	return shHistory, n, nil
}
