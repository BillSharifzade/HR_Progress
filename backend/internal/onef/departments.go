package onef

import (
	"strings"
)

// explicitDeptAliases maps a normalized 1F department name to the local
// department code it should resolve to. Use this only when normalizeDeptName
// cannot bridge the gap (i.e. real wording differences, not whitespace/hyphens).
//
// Example: 1F sends "Департамент по Закупкам и Логистике" (dative+preposition);
// our seed is "Департамент Закупки и Логистики" (genitive). Same dept, different
// grammar — only an explicit alias handles it.
// The ДФП/БЮД entries are belt-and-braces rather than strictly necessary:
// migration 0016 renamed both departments to the names 1F actually sends, so
// normalizeDeptName already bridges them (it collapses the double space in
// 1F's "Бухгалтерский и Юридический  Департамент"). They are pinned here so a
// future edit to either department's display name cannot quietly resurrect the
// auto-create path that produced `ДФП2` and `БИЮД` in the first place.
var explicitDeptAliases = map[string]string{
	normalizeDeptName("Департамент по Закупкам и Логистике"):      "ДЗЛ",
	normalizeDeptName("Департамент Фармацевтической Промоции"):    "ДФП",
	normalizeDeptName("Бухгалтерский и Юридический  Департамент"): "БЮД",
}

// ignoredDepartments are 1F department names whose users we never sync.
// Per product decision 2026-05-26: these are out of scope; their workers
// should never appear in our DB, even if 1F lists them.
//
// "Департамент Фармацевтической Промоции" (ДФП) is deliberately NOT here.
// It went out of scope on 2026-07-30 and came back in on 2026-08-12; its six
// people must arrive from both 1F and the profile questionnaire. Note the
// binary deployed to production on 2026-07-30 predates the brief re-ignore, so
// production has been syncing ДФП all along — re-adding it here would freeze
// those six records the moment anyone redeploys.
//
// Keys are normalized via normalizeDeptName so minor whitespace/case
// drift in 1F doesn't accidentally bypass the filter.
var ignoredDepartments = map[string]bool{
	normalizeDeptName("Дусти Фарма"):                       true,
	normalizeDeptName("Департамент Инженерной Экспертизы"): true,
}

// isIgnoredDepartment reports whether a 1F Department value should be
// dropped from sync entirely.
func isIgnoredDepartment(name string) bool {
	return ignoredDepartments[normalizeDeptName(name)]
}

// aliasedDeptCode returns the local department code that a 1F department name
// should resolve to, or "" if no explicit alias is configured.
func aliasedDeptCode(name string) string {
	return explicitDeptAliases[normalizeDeptName(name)]
}

// normalizeDeptName folds away whitespace and hyphen variants so that
// "Финансово-Экономический Департамент" (seeded) matches
// "Финансово Экономический Департамент" (1F). The rule:
//   - lowercase
//   - replace '-' with ' '
//   - collapse any run of whitespace to a single space
//   - trim leading/trailing spaces
func normalizeDeptName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, "-", " ")
	// collapse whitespace runs to a single space
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}
	return strings.TrimSpace(b.String())
}
