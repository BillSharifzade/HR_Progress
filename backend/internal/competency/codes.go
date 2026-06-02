package competency

import (
	"strings"
	"unicode"
)

// initials returns the first letter of each whitespace- or hyphen-separated word, uppercased.
// Example: "Финансово-Экономический Департамент" -> "ФЭД".
func initials(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return unicode.IsSpace(r) || r == '-'
	})
	var b strings.Builder
	for _, p := range parts {
		for _, r := range p {
			b.WriteRune(unicode.ToUpper(r))
			break
		}
	}
	return b.String()
}

func deriveDeptCode(name string) string {
	return initials(name)
}

func deriveCompetencyCode(name string, kind Kind) string {
	tail := initials(name)
	if tail == "" {
		return string(kind)
	}
	return string(kind) + "_" + tail
}
