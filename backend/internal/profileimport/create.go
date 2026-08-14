package profileimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"hrprogress/internal/auth"
	"hrprogress/internal/workers"
)

// --- 1F roster ------------------------------------------------------------

// OneFRoster is the set of people 1F currently knows about. It exists so the
// importer can tell two very different situations apart:
//
//   - the respondent is in 1F but has not reached our database yet — a sync
//     problem, and creating a local user would produce a duplicate the moment
//     the sync catches up, because the sync matches on one_f_user_id;
//   - the respondent is in no HR system at all — the only case where creating
//     a user from questionnaire data is the right answer.
type OneFRoster struct {
	names []string
}

// LoadOneFRoster reads a raw 1F payload (the bare array the endpoint returns,
// or an object wrapping one).
func LoadOneFRoster(path string) (*OneFRoster, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var direct []map[string]any
	if err := json.Unmarshal(raw, &direct); err != nil {
		var wrapper map[string]json.RawMessage
		if err2 := json.Unmarshal(raw, &wrapper); err2 != nil {
			return nil, fmt.Errorf("parse 1F payload: %w", err)
		}
		for _, v := range wrapper {
			if err := json.Unmarshal(v, &direct); err == nil && len(direct) > 0 {
				break
			}
		}
		if len(direct) == 0 {
			return nil, errors.New("no user array found in 1F payload")
		}
	}
	r := &OneFRoster{}
	for _, u := range direct {
		if n, ok := u["DisplayName"].(string); ok && strings.TrimSpace(n) != "" {
			r.names = append(r.names, strings.TrimSpace(n))
		}
	}
	if len(r.names) == 0 {
		return nil, errors.New("1F payload contained no DisplayName values")
	}
	return r, nil
}

func (r *OneFRoster) Size() int { return len(r.names) }

// rosterSuspicionThreshold is deliberately LOWER than the matcher's own
// FuzzyThreshold, because the two mistakes cost very different amounts.
//
// Saying "1F might have this person" when it does not costs one row held back
// for a human to look at. Saying "1F does not have them" when it does creates a
// permanent duplicate: the sync keys on one_f_user_id, so it will never
// recognise the record we made, and the person ends up in the system twice.
//
// The gap between 0.75 and 0.90 is where the real near-misses live —
// "Исмоилзод Мустафо Шамсуло" scores 0.805 against 1F's "Исмоилов Мустафо
// Шамсулоевич" (the -ов/-зод surname variant is routine here), and "Хукматов
// Файзулло" scores 0.850 against "Хукматзода Абдулло Файзулло" in the same
// department. Neither may be auto-created.
const rosterSuspicionThreshold = 0.75

// Contains reports whether 1F plausibly holds this person, and if so under
// which spelling. Homoglyph and Tajik-letter folding applies, so a respondent
// whose 1F record merely spells their name differently is not treated as
// absent.
func (r *OneFRoster) Contains(name string) (string, bool) {
	if r == nil {
		return "", false
	}
	formTokens := normalizeName(name)
	if len(formTokens) == 0 {
		return "", false
	}
	bestName, bestScore := "", 0.0
	for _, candidate := range r.names {
		score, _ := scoreNames(formTokens, normalizeName(candidate))
		if score > bestScore {
			bestName, bestScore = candidate, score
		}
	}
	if bestScore < rosterSuspicionThreshold {
		return "", false
	}
	return bestName, true
}

// --- department resolution ------------------------------------------------

// formDeptAliases bridge the wording the questionnaire offered and the names
// the departments actually carry. Both differences are real: the form writes
// "Закупа" where the department is "Закупки", and "Бухгалтерско-Юридический"
// where it is "Бухгалтерский и Юридический". Normalisation cannot close
// either gap, and every other option matches directly.
var formDeptAliases = map[string]string{
	"департамент закупа и логистики":       "ДЗЛ",
	"бухгалтерско юридический департамент": "БЮД",
}

var wsRe = regexp.MustCompile(`\s+`)

// normalizeDept mirrors the folding the 1F integration uses, so the two agree.
func normalizeDept(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", " ")
	return wsRe.ReplaceAllString(s, " ")
}

// resolveDepartment maps a self-reported department onto a real one. It never
// creates a department: an answer that matches nothing means the person is not
// placeable, and the row stays unresolved for a human.
func resolveDepartment(ctx context.Context, tx pgx.Tx, answer string) (uuid.UUID, bool, error) {
	normalized := normalizeDept(answer)
	if normalized == "" {
		return uuid.Nil, false, nil
	}

	if code, ok := formDeptAliases[normalized]; ok {
		var id uuid.UUID
		err := tx.QueryRow(ctx,
			`SELECT id FROM departments WHERE deleted_at IS NULL AND code = $1`, code).Scan(&id)
		if err == nil {
			return id, true, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, false, err
		}
	}

	var id uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT id FROM departments
		WHERE deleted_at IS NULL
		  AND regexp_replace(lower(replace(name, '-', ' ')), '\s+', ' ', 'g') = $1`,
		normalized).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, err
	}
	return id, true, nil
}

// --- user creation --------------------------------------------------------

// uniqueUsername allocates a free username inside the import transaction.
// Repository.UniqueUsername cannot be used here because it reads through the
// pool and would not see users created earlier in this same transaction.
func uniqueUsername(ctx context.Context, tx pgx.Tx, fullName string) (string, error) {
	base := workers.UsernameBase(fullName)
	if base == "" {
		base = "user"
	}
	for i := 0; i < 64; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s%d", base, i+1)
		}
		var exists bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM users WHERE lower(username) = lower($1))`,
			candidate).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no free username for %q", fullName)
}

// createEmployeeFromForm inserts a user built purely from questionnaire data.
//
// one_f_user_id is left NULL by design: these people are in no 1F record, so
// there is nothing to key on. That also means a later sync will not recognise
// them — if they are added to 1F afterwards, they must be reconciled by hand.
//
// The account gets a random password and must_change_password, matching what
// the "create worker" screen does; HR issues real credentials through the
// existing credentials-reset action.
func createEmployeeFromForm(ctx context.Context, tx pgx.Tx, fullName, department string) (uuid.UUID, error) {
	deptID, ok, err := resolveDepartment(ctx, tx, department)
	if err != nil {
		return uuid.Nil, err
	}
	var deptArg any
	if ok {
		deptArg = deptID
	}

	username, err := uniqueUsername(ctx, tx, fullName)
	if err != nil {
		return uuid.Nil, err
	}
	tempPassword, err := auth.GenerateTempPassword()
	if err != nil {
		return uuid.Nil, fmt.Errorf("generate password: %w", err)
	}
	hash, err := auth.HashPassword(tempPassword)
	if err != nil {
		return uuid.Nil, fmt.Errorf("hash password: %w", err)
	}

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO users (username, password_hash, full_name, department_id,
			must_change_password, is_active)
		VALUES ($1, $2, $3, $4, true, true)
		RETURNING id`,
		username, hash, strings.TrimSpace(fullName), deptArg,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert user %q: %w", fullName, err)
	}
	return id, nil
}
