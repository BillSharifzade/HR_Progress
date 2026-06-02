package workers

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"strings"
	"unicode"
)

// randomSuffix returns a 4-digit zero-padded random number using crypto/rand.
func randomSuffix() (string, error) {
	var n uint32
	if err := binary.Read(rand.Reader, binary.BigEndian, &n); err != nil {
		return "", err
	}
	return fmt.Sprintf("%04d", n%10000), nil
}

var cyrillicToLatin = map[rune]string{
	'А': "A", 'а': "a", 'Б': "B", 'б': "b", 'В': "V", 'в': "v",
	'Г': "G", 'г': "g", 'Д': "D", 'д': "d", 'Е': "E", 'е': "e",
	'Ё': "Yo", 'ё': "yo", 'Ж': "Zh", 'ж': "zh", 'З': "Z", 'з': "z",
	'И': "I", 'и': "i", 'Й': "Y", 'й': "y", 'К': "K", 'к': "k",
	'Л': "L", 'л': "l", 'М': "M", 'м': "m", 'Н': "N", 'н': "n",
	'О': "O", 'о': "o", 'П': "P", 'п': "p", 'Р': "R", 'р': "r",
	'С': "S", 'с': "s", 'Т': "T", 'т': "t", 'У': "U", 'у': "u",
	'Ф': "F", 'ф': "f", 'Х': "H", 'х': "h", 'Ц': "Ts", 'ц': "ts",
	'Ч': "Ch", 'ч': "ch", 'Ш': "Sh", 'ш': "sh", 'Щ': "Sch", 'щ': "sch",
	'Ъ': "", 'ъ': "", 'Ы': "Y", 'ы': "y", 'Ь': "", 'ь': "",
	'Э': "E", 'э': "e", 'Ю': "Yu", 'ю': "yu", 'Я': "Ya", 'я': "ya",
}

// sectionInitials returns the first letter of each whitespace- or hyphen-separated word, uppercased.
func sectionInitials(name string) string {
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

func transliterate(s string) string {
	var b strings.Builder
	for _, r := range s {
		if v, ok := cyrillicToLatin[r]; ok {
			b.WriteString(v)
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// UsernameBase is the public form of usernameBase; callable from other packages
// (e.g. internal/onef) that need to generate usernames the same way.
func UsernameBase(fullName string) string { return usernameBase(fullName) }

// usernameBase returns "f.lastname" style (first letter of first part + dot + last part), lowercased.
// Falls back to a flattened version of the whole input.
func usernameBase(fullName string) string {
	parts := strings.FieldsFunc(fullName, func(r rune) bool {
		return unicode.IsSpace(r) || r == '-'
	})
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return strings.ToLower(transliterate(parts[0]))
	}
	first := transliterate(parts[0])
	last := transliterate(parts[len(parts)-1])
	if first == "" || last == "" {
		flat := ""
		for _, p := range parts {
			flat += transliterate(p)
		}
		return strings.ToLower(flat)
	}
	return strings.ToLower(string(first[0])) + "." + strings.ToLower(last)
}
