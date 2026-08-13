package profileimport

import (
	"sort"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

// MatchStatus describes how confidently a form row was tied to an employee.
type MatchStatus string

const (
	MatchExact      MatchStatus = "EXACT"      // identical token sets
	MatchPartial    MatchStatus = "PARTIAL"    // form omitted a name part (usually the patronymic)
	MatchFuzzy      MatchStatus = "FUZZY"      // close enough on every token
	MatchOverride   MatchStatus = "OVERRIDE"   // tied by hand after review
	MatchUnresolved MatchStatus = "UNRESOLVED" // too weak, or two candidates too close together
)

// NormalizedKey renders a name in the form used to key manual overrides.
func NormalizedKey(s string) string { return strings.Join(normalizeName(s), " ") }

// FindByFullName looks an employee up by their exact recorded name, compared
// after the same normalisation the matcher uses.
func FindByFullName(employees []Employee, fullName string) (Employee, bool) {
	want := NormalizedKey(fullName)
	for _, e := range employees {
		if NormalizedKey(e.FullName) == want {
			return e, true
		}
	}
	return Employee{}, false
}

// Candidate is one possible employee for a form row.
type Candidate struct {
	UserID   uuid.UUID
	FullName string
	Score    float64
}

// Employee is the minimal view of a user the matcher needs.
type Employee struct {
	ID       uuid.UUID
	FullName string
}

// Match is the outcome for one form row.
type Match struct {
	Status     MatchStatus
	UserID     uuid.UUID
	Score      float64
	Candidates []Candidate // top few, for reporting unresolved rows
}

// homoglyphs folds Latin characters that are visually identical to Cyrillic
// ones onto the Cyrillic letter. People type these by accident constantly —
// row 14 of this export reads "Oтахонов" with a LATIN O — and without folding
// they look like a spelling difference to any edit-distance measure. The same
// class of bug already bit the 1F grade inference, which carries a Latin "c"
// in "Главный cпециалист".
//
// Applied before lowercasing, because the uppercase pairs (H/Н, M/М, T/Т) have
// no lowercase equivalent to fold afterwards.
var homoglyphs = strings.NewReplacer(
	"A", "А", "a", "а",
	"B", "В",
	"C", "С", "c", "с",
	"E", "Е", "e", "е",
	"H", "Н",
	"K", "К", "k", "к",
	"M", "М",
	"O", "О", "o", "о",
	"P", "Р", "p", "р",
	"T", "Т",
	"X", "Х", "x", "х",
	"Y", "У", "y", "у",
)

// tajikLetters folds the Tajik Cyrillic extensions onto their nearest Russian
// letter. The questionnaire and 1F disagree constantly on these — the form has
// "Қаландаров"/"Сатҷева" where the employee record reads "Каландаров"/"Сатыева"
// — and treating them as different characters costs a whole token's score.
var tajikLetters = strings.NewReplacer(
	"Қ", "К", "қ", "к",
	"Ҷ", "Ч", "ҷ", "ч",
	"Ҳ", "Х", "ҳ", "х",
	"Ғ", "Г", "ғ", "г",
	"Ӯ", "У", "ӯ", "у",
	"Ӣ", "И", "ӣ", "и",
	"Ё", "Е", "ё", "е",
)

// normalizeName folds homoglyphs and Tajik letters, lowercases, strips
// punctuation and splits into tokens. Matching runs on tokens rather than the
// whole string: the form frequently omits the patronymic, and whole-string
// similarity punishes that so hard ("Акопян Ованес" vs "Акопян Ованес
// Гагикович" scores ~0.72) that real people would be discarded.
func normalizeName(s string) []string {
	s = homoglyphs.Replace(strings.TrimSpace(s))
	s = tajikLetters.Replace(s)
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return ' '
	}, s)
	return strings.Fields(s)
}

// similarity is normalized Levenshtein over two tokens: 1 for identical, 0 for
// nothing in common. Tolerates the single-character transliteration drift the
// data is full of (Джаъфар/Джафар, Хушвахтовна/Хушвактовна).
func similarity(a, b string) float64 {
	if a == b {
		return 1
	}
	ra, rb := []rune(a), []rune(b)
	if len(ra) == 0 || len(rb) == 0 {
		return 0
	}
	prev := make([]int, len(rb)+1)
	curr := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		curr[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = min3(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	dist := prev[len(rb)]
	longest := len(ra)
	if len(rb) > longest {
		longest = len(rb)
	}
	return 1 - float64(dist)/float64(longest)
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}

// scoreNames aligns the shorter token list onto the longer one, greedily
// pairing each token with its best unused counterpart, and averages the pair
// scores. Order-independent, so "Иванов Иван" and "Иван Иванов" score the same.
//
// The second return value counts how many tokens found a strong partner.
// Requiring EVERY token to be strong is too harsh: patronymics drift in
// transliteration far more than surnames do ("Амихуджаевич" vs "Амирхучаевич"),
// and rejecting on that alone discards obviously-correct matches whose nearest
// rival is 0.35 behind. Demanding two strong tokens — in practice the surname
// and the given name — plus the ambiguity gap is the safer trade.
func scoreNames(a, b []string) (score float64, strongTokens int) {
	if len(a) == 0 || len(b) == 0 {
		return 0, 0
	}
	short, long := a, b
	if len(short) > len(long) {
		short, long = long, short
	}
	used := make([]bool, len(long))
	total := 0.0
	for _, tok := range short {
		bestIdx, bestScore := -1, 0.0
		for i, cand := range long {
			if used[i] {
				continue
			}
			if s := similarity(tok, cand); s > bestScore {
				bestIdx, bestScore = i, s
			}
		}
		if bestIdx >= 0 {
			used[bestIdx] = true
		}
		if bestScore >= strongTokenThreshold {
			strongTokens++
		}
		total += bestScore
	}
	return total / float64(len(short)), strongTokens
}

// Thresholds. AmbiguityGap exists because the data contains genuine same-name
// traps — 1F holds two "Хакимов Илхом" in different departments — and writing
// a profile against the wrong user_id is far harder to undo than to prevent.
const (
	ExactThreshold = 0.995
	FuzzyThreshold = 0.90
	AmbiguityGap   = 0.02
	maxCandidates  = 3
	partialPenalty = 0.0 // partials are not penalised, only labelled
	// strongTokenThreshold is what counts as "this name part agrees".
	strongTokenThreshold = 0.85
	// minStrongTokens: surname and given name must both agree, so a lone
	// surname collision can never carry a match on its own.
	minStrongTokens = 2
)

// MatchName ties one questionnaire name to an employee, or reports that it
// cannot be done safely.
func MatchName(formName string, employees []Employee) Match {
	formTokens := normalizeName(formName)
	if len(formTokens) == 0 {
		return Match{Status: MatchUnresolved}
	}

	scored := make([]Candidate, 0, len(employees))
	strong := make(map[uuid.UUID]int, len(employees))
	sameLen := make(map[uuid.UUID]bool, len(employees))

	for _, e := range employees {
		empTokens := normalizeName(e.FullName)
		if len(empTokens) == 0 {
			continue
		}
		s, strongTokens := scoreNames(formTokens, empTokens)
		scored = append(scored, Candidate{UserID: e.ID, FullName: e.FullName, Score: s})
		strong[e.ID] = strongTokens
		sameLen[e.ID] = len(empTokens) == len(formTokens)
	}
	if len(scored) == 0 {
		return Match{Status: MatchUnresolved}
	}

	sort.Slice(scored, func(i, j int) bool { return scored[i].Score > scored[j].Score })
	top := scored[0]
	shortlist := scored[:min(maxCandidates, len(scored))]

	// Two candidates within AmbiguityGap of each other: refuse rather than
	// coin-flip between two real people.
	if len(scored) > 1 && top.Score-scored[1].Score < AmbiguityGap {
		return Match{Status: MatchUnresolved, Score: top.Score, Candidates: shortlist}
	}
	// A single token can never carry a match: too many surnames repeat.
	if len(formTokens) < minStrongTokens {
		return Match{Status: MatchUnresolved, Score: top.Score, Candidates: shortlist}
	}
	if top.Score < FuzzyThreshold || strong[top.UserID] < minStrongTokens {
		return Match{Status: MatchUnresolved, Score: top.Score, Candidates: shortlist}
	}

	status := MatchFuzzy
	switch {
	case top.Score >= ExactThreshold && sameLen[top.UserID]:
		status = MatchExact
	case top.Score >= ExactThreshold:
		status = MatchPartial
	}
	return Match{Status: status, UserID: top.UserID, Score: top.Score - partialPenalty, Candidates: shortlist}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
