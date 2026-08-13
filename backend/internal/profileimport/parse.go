// Package profileimport turns the digital-profile questionnaire export into
// profile rows. The parsing rules here are all responses to real defects in
// the source data, not hypotheticals — see the comment on each.
package profileimport

import (
	"regexp"
	"strings"
	"unicode"
)

// --- CEFR levels ----------------------------------------------------------

// cefrOrder ranks the levels so a multi-tick can be collapsed to the best one.
var cefrOrder = map[string]int{"A1": 1, "A2": 2, "B1": 3, "B2": 4, "C1": 5, "C2": 6}

// normalizeCEFR fixes the alphabet mix in the source: the form's A-levels were
// typed with a CYRILLIC А ("А1", "А2") while B and C are Latin. Without this
// the database CHECK rejects exactly the beginner rows.
func normalizeCEFR(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	r := strings.NewReplacer(
		"А", "A", // Cyrillic А (U+0410) → Latin A
		"В", "B", // Cyrillic В (U+0412) → Latin B
		"С", "C", // Cyrillic С (U+0421) → Latin C
	)
	return r.Replace(s)
}

// HighestCEFR picks the strongest level out of an answer.
//
// The language grid rendered as checkboxes rather than radio buttons, so 27
// people ticked several levels for one language and 5 ticked all six. Taking
// the highest is the agreed rule.
//
// Returns "" when the answer holds no recognisable level.
func HighestCEFR(answer string) string {
	best, bestRank := "", 0
	for _, part := range splitList(answer) {
		lvl := normalizeCEFR(part)
		if rank, ok := cefrOrder[lvl]; ok && rank > bestRank {
			best, bestRank = lvl, rank
		}
	}
	return best
}

// --- multi-select answers -------------------------------------------------

// splitList splits a Google Forms multi-select, which joins the chosen options
// with ", ". Splitting is depth-aware: several answers contain commas inside
// parentheses (e.g. "ЦУПЭС (Центр управления проектами в секторе)"), and a
// naive split would shred them.
func splitList(s string) []string {
	var out []string
	var buf strings.Builder
	depth := 0
	for _, r := range s {
		switch r {
		case '(', '[', '«', '"':
			depth++
			buf.WriteRune(r)
		case ')', ']', '»':
			if depth > 0 {
				depth--
			}
			buf.WriteRune(r)
		case ',', ';':
			if depth == 0 {
				out = append(out, buf.String())
				buf.Reset()
				continue
			}
			buf.WriteRune(r)
		default:
			buf.WriteRune(r)
		}
	}
	out = append(out, buf.String())

	cleaned := make([]string, 0, len(out))
	for _, v := range out {
		if v = strings.TrimSpace(v); v != "" {
			cleaned = append(cleaned, v)
		}
	}
	return cleaned
}

// SplitOptions parses a multi-select answer into its chosen options.
func SplitOptions(s string) []string { return splitList(s) }

// --- certificates ---------------------------------------------------------

var urlRe = regexp.MustCompile(`https?://\S+`)

// CertificateURLs pulls the links out of the certificate answer. Every
// response in this round is one or more Google Drive URLs joined by ", ".
func CertificateURLs(s string) []string {
	found := urlRe.FindAllString(s, -1)
	out := make([]string, 0, len(found))
	for _, u := range found {
		out = append(out, strings.TrimRight(u, ".,;"))
	}
	return out
}

// --- work experience ------------------------------------------------------

// noExperienceRe recognises the ways people said "this is my first job".
// Matched against the whole trimmed answer, and only for short answers, so a
// long narrative that happens to contain "не работал" is not thrown away.
var noExperienceRe = regexp.MustCompile(
	`(?i)^(нигде|нет|нету|не\s*работал[аи]?|нигде\s*не\s*работал[аи]?|` +
		`это\s*)?(мо[её]|первое)?\s*(первое\s*)?(место\s*работы)?[\s.\-—]*$|` +
		`(?i)^(до,?\s*)?нигде\s*не\s*работал[аи]?[\s.]*$|` +
		`(?i)^я\s*вообще\s*не\s*работал[аи]?[\s.]*$|` +
		`(?i)^не\s*работал[аи]?\s*в\s*других\s*компаниях[\s.]*$|` +
		`(?i)^отсутству?е?т[\s.]*$|^0$|^-+$|^—+$`)

// dashSplitRe finds the separator between a company and a position. Only a
// spaced dash counts: hyphens inside a name ("Тревел-Групп") must survive.
var dashSplitRe = regexp.MustCompile(`\s+[-–—]\s+`)

// parenPositionRe matches the other shape people used: "Барки Точик (Электромонтер)".
var parenPositionRe = regexp.MustCompile(`^(.+?)\s*\(([^()]*)\)\s*$`)

// Employment is one parsed row of previous employment.
type Employment struct {
	Company  string
	Position string
	// Description carries the full answer when it could not be parsed into a
	// company/position pair, so nothing is lost to truncation.
	Description string
	// Raw is the fragment this row was split out of, kept so a wrong split is
	// visible in the UI and correctable rather than silently becoming truth.
	Raw string
}

// maxListFragments caps how many companies one answer may split into. Beyond
// this the answer is almost certainly prose, not a list.
const maxListFragments = 8

// looksLikeProse reports whether a comma-separated answer is a sentence rather
// than a list of employers.
//
// This matters a great deal: one response reads "До работы в ЗАО «КОИНОТИ НАВ»
// я работала проектным менеджером в компании YALLA.TJ, где занималась
// операционным управлением, аналитикой и оптимизацией процессов, …" — splitting
// that on commas produced seven rows of sentence fragments like "где занималась
// операционным управлением", which are not employers and are worse than no
// data at all.
//
// The tell is simple and reliable in Russian: list items are proper nouns and
// start with a capital, while a continuing clause starts lowercase.
func looksLikeProse(fragments []string) bool {
	if len(fragments) < 2 {
		return false
	}
	if len(fragments) > maxListFragments {
		return true
	}
	for _, f := range fragments[1:] {
		for _, r := range f {
			if unicode.IsLetter(r) {
				if unicode.IsLower(r) {
					return true
				}
				break
			}
		}
	}
	return false
}

// maxCompanyLen matches the API validation ceiling on the company field.
const maxCompanyLen = 300

// ParseEmployment splits the "which companies did you work for" answer.
//
// This answer is free text and genuinely messy: 15 people said they had no
// prior job, 65 wrote a single unpunctuated phrase, and others wrote lists
// with commas that also appear inside parentheses. The parser is deliberately
// conservative — when a fragment has no recognisable company/position
// separator the whole fragment becomes the company and HR fixes it in the UI.
// Nothing is dropped except genuine "no experience" answers.
func ParseEmployment(answer string) []Employment {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return nil
	}
	// Only treat short answers as "no experience"; a long text that merely
	// contains such a phrase is real content.
	if len([]rune(answer)) <= 40 && noExperienceRe.MatchString(answer) {
		return nil
	}

	fragments := splitList(answer)
	if looksLikeProse(fragments) {
		// Keep it whole. The company field gets a readable prefix and the full
		// text is preserved in Description and Raw, so HR can split it by hand
		// in the UI rather than inheriting nonsense rows.
		return []Employment{{
			Company:     truncate(answer, 120),
			Description: answer,
			Raw:         answer,
		}}
	}

	var out []Employment
	for _, fragment := range fragments {
		fragment = strings.TrimSpace(fragment)
		if fragment == "" {
			continue
		}
		company, position := fragment, ""

		if parts := dashSplitRe.Split(fragment, 2); len(parts) == 2 {
			// "Company - Position", possibly with further dashes in the
			// position ("… - айти специалист - специалист по разработке").
			company, position = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		} else if m := parenPositionRe.FindStringSubmatch(fragment); m != nil {
			company, position = strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
		}

		if company == "" {
			continue
		}
		out = append(out, Employment{
			Company:  truncate(company, maxCompanyLen),
			Position: truncate(position, maxCompanyLen),
			Raw:      fragment,
		})
	}
	return out
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return strings.TrimSpace(string(r[:max-1])) + "…"
}

// --- misc -----------------------------------------------------------------

// blankAnswerRe catches the placeholders people typed instead of leaving a
// free-text field empty. These become no row at all rather than a stored "нет".
var blankAnswerRe = regexp.MustCompile(`(?i)^(нет|нету|н/?д|no|none|-+|—+|0|нет информации|` +
	`не\s*знаю|пока\s*нет|отсутствует|нет\s*дополнительной\s*информации)[\s.]*$`)

// IsBlankAnswer reports whether a free-text answer carries no information.
func IsBlankAnswer(s string) bool {
	s = strings.TrimSpace(s)
	return s == "" || blankAnswerRe.MatchString(s)
}
