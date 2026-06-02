package onef

import "strings"

// InferGradeLevel maps a 1F Dolzhnost (job title in Russian, free-text)
// to the local grade level (1..7), or 0 if no confident match.
//
// 1F data is dirty (e.g. observed "Главный cпециалист" — Latin 'c' in 'cпециалист'),
// so matching is case-insensitive on the lowercased Cyrillic stem only.
// Order matters: longest/most-specific phrases must be checked before the
// generic "специалист" fallback.
func InferGradeLevel(dolzhnost string) int {
	s := strings.ToLower(strings.TrimSpace(dolzhnost))
	if s == "" {
		return 0
	}

	// Convert Latin 'c' (U+0063) to Cyrillic 'с' (U+0441) so "cпециалист" matches "специалист".
	// Both glyphs are visually identical and routinely confused in copy-paste.
	s = strings.ReplaceAll(s, "c", "с")

	switch {
	case strings.Contains(s, "руководитель департамента"),
		strings.Contains(s, "директор департамента"):
		return 7
	case strings.Contains(s, "заместитель руководителя департамента"),
		strings.Contains(s, "заместитель директора департамента"):
		return 6
	case strings.Contains(s, "руководитель отдела"),
		strings.Contains(s, "начальник отдела"):
		return 5
	case strings.Contains(s, "главный"):
		return 4
	case strings.Contains(s, "ведущий"):
		return 3
	case strings.Contains(s, "стаж"): // "стажёр", "стажер"
		return 1
	case strings.Contains(s, "специалист"):
		return 2
	}
	return 0
}
