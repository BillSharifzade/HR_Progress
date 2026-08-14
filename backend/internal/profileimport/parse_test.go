package profileimport

import (
	"testing"

	"github.com/google/uuid"
)

func TestHighestCEFR(t *testing.T) {
	tests := []struct{ in, want string }{
		{"B1", "B1"},
		// Cyrillic А in the source — the whole reason this normalisation exists.
		{"А1", "A1"},
		{"А2", "A2"},
		// Checkbox grid: several levels ticked for one language, highest wins.
		{"А1, А2, B1", "B1"},
		{"А1, А2, B1, B2, C1, C2", "C2"},
		{"C1, B2", "C1"},
		{"", ""},
		{"не владею", ""},
	}
	for _, tt := range tests {
		if got := HighestCEFR(tt.in); got != tt.want {
			t.Errorf("HighestCEFR(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSplitOptionsRespectsParentheses(t *testing.T) {
	// Real answer: the commas inside the parenthetical must not split it.
	in := "Барки Точик (Электромонтер), ЦУПЭС (Центр управления проектами, в секторе) ( вед.специалист)"
	got := SplitOptions(in)
	if len(got) != 2 {
		t.Fatalf("expected 2 fragments, got %d: %#v", len(got), got)
	}
	if got[0] != "Барки Точик (Электромонтер)" {
		t.Errorf("first fragment = %q", got[0])
	}
}

func TestSplitOptionsMultiSelect(t *testing.T) {
	got := SplitOptions("Высшее, Магистратура")
	if len(got) != 2 || got[0] != "Высшее" || got[1] != "Магистратура" {
		t.Errorf("got %#v", got)
	}
}

func TestParseEmployment(t *testing.T) {
	t.Run("no prior experience yields no rows", func(t *testing.T) {
		for _, s := range []string{
			"Нигде", "Не работал", "не работала", "нигде не работал",
			"Первое место работы", "Это мое первое место работы",
			"Я вообще не работал", "Не работал в других компаниях", "-", "0",
		} {
			if got := ParseEmployment(s); len(got) != 0 {
				t.Errorf("ParseEmployment(%q) = %#v, want none", s, got)
			}
		}
	})

	t.Run("company - position list", func(t *testing.T) {
		got := ParseEmployment(
			"Thefiftyfive Group - Ведущий специалист разработки, Technohub Dushanbe - главный ментор")
		if len(got) != 2 {
			t.Fatalf("want 2 rows, got %d: %#v", len(got), got)
		}
		if got[0].Company != "Thefiftyfive Group" || got[0].Position != "Ведущий специалист разработки" {
			t.Errorf("row 0 = %+v", got[0])
		}
		if got[1].Company != "Technohub Dushanbe" {
			t.Errorf("row 1 = %+v", got[1])
		}
	})

	t.Run("extra dashes stay in the position", func(t *testing.T) {
		got := ParseEmployment("Opulence Group - айти специалист - специалист по разработке")
		if len(got) != 1 {
			t.Fatalf("want 1 row, got %#v", got)
		}
		if got[0].Company != "Opulence Group" {
			t.Errorf("company = %q", got[0].Company)
		}
		if got[0].Position != "айти специалист - специалист по разработке" {
			t.Errorf("position = %q", got[0].Position)
		}
	})

	t.Run("hyphenated company name is not split", func(t *testing.T) {
		got := ParseEmployment("Тревел-Групп")
		if len(got) != 1 || got[0].Company != "Тревел-Групп" {
			t.Errorf("got %#v", got)
		}
	})

	t.Run("parenthesised position", func(t *testing.T) {
		got := ParseEmployment("Барки Точик (Электромонтер)")
		if len(got) != 1 || got[0].Company != "Барки Точик" || got[0].Position != "Электромонтер" {
			t.Errorf("got %#v", got)
		}
	})

	t.Run("unparseable narrative becomes one row and keeps its text", func(t *testing.T) {
		in := "во всех указанных компаниях (Озон курс, Яндекс, Скриникс стартап и Коиноти Нав ДИТ) в качестве Go разработчик"
		got := ParseEmployment(in)
		if len(got) != 1 {
			t.Fatalf("want 1 row (commas are inside parens), got %d: %#v", len(got), got)
		}
		if got[0].Raw != in {
			t.Errorf("raw not preserved: %q", got[0].Raw)
		}
	})

	t.Run("long text mentioning не работал is kept", func(t *testing.T) {
		in := "ООО «HLB Таджикистан Аутсорсинг» - ведущий бухгалтер, ранее не работал по специальности"
		if got := ParseEmployment(in); len(got) == 0 {
			t.Error("a long answer must not be discarded as 'no experience'")
		}
	})
}

func TestCertificateURLs(t *testing.T) {
	got := CertificateURLs(
		"https://drive.google.com/open?id=1dbmBOM, https://drive.google.com/open?id=14maMzY")
	if len(got) != 2 {
		t.Fatalf("want 2 urls, got %#v", got)
	}
	if got[0] != "https://drive.google.com/open?id=1dbmBOM" {
		t.Errorf("trailing comma not trimmed: %q", got[0])
	}
}

func TestIsBlankAnswer(t *testing.T) {
	for _, s := range []string{"", "нет", "Нет", "-", "—", "Нет информации", "0"} {
		if !IsBlankAnswer(s) {
			t.Errorf("IsBlankAnswer(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"Разработка ПО", "нет времени на обучение"} {
		if IsBlankAnswer(s) {
			t.Errorf("IsBlankAnswer(%q) = true, want false", s)
		}
	}
}

// --- matching -------------------------------------------------------------

func emp(name string) Employee { return Employee{ID: uuid.New(), FullName: name} }

func TestMatchName(t *testing.T) {
	staff := []Employee{
		emp("Максадов Рахматджон Камолиддинович"),
		emp("Акопян Ованес Гагикович"),
		emp("Кенджаева Фарзуна Хушвахтовна"),
		emp("Ёраков Шахбоз Одинаевич"),
	}

	t.Run("exact", func(t *testing.T) {
		m := MatchName("Максадов Рахматджон Камолиддинович", staff)
		if m.Status != MatchExact {
			t.Fatalf("status = %s (score %.3f)", m.Status, m.Score)
		}
		if m.UserID != staff[0].ID {
			t.Error("matched the wrong person")
		}
	})

	t.Run("missing patronymic is PARTIAL, not lost", func(t *testing.T) {
		m := MatchName("Акопян Ованес", staff)
		if m.Status != MatchPartial {
			t.Fatalf("status = %s (score %.3f)", m.Status, m.Score)
		}
		if m.UserID != staff[1].ID {
			t.Error("matched the wrong person")
		}
	})

	t.Run("ё is folded to е", func(t *testing.T) {
		if m := MatchName("Ераков Шахбоз Одинаевич", staff); m.Status == MatchUnresolved {
			t.Errorf("ё/е drift should still match, got %s (%.3f)", m.Status, m.Score)
		}
	})

	t.Run("transliteration drift still matches", func(t *testing.T) {
		m := MatchName("Кенджаева Фарзуна Хушвактовна", staff) // note: вак vs вах
		if m.Status == MatchUnresolved {
			t.Errorf("expected a match, got %s (%.3f)", m.Status, m.Score)
		}
	})

	t.Run("unknown person is unresolved", func(t *testing.T) {
		if m := MatchName("Петров Пётр Петрович", staff); m.Status != MatchUnresolved {
			t.Errorf("status = %s (%.3f) — must not guess", m.Status, m.Score)
		}
	})

	t.Run("same-name trap refuses rather than guessing", func(t *testing.T) {
		twins := []Employee{
			emp("Хакимов Илхом Бекбобоевич"),
			emp("Хакимов Илхом Сафарович"),
		}
		m := MatchName("Хакимов Илхом", twins)
		if m.Status != MatchUnresolved {
			t.Fatalf("two equally-good candidates must be UNRESOLVED, got %s -> %s",
				m.Status, m.UserID)
		}
		if len(m.Candidates) < 2 {
			t.Error("unresolved rows should report their candidates for review")
		}
	})

	t.Run("word order does not matter", func(t *testing.T) {
		if m := MatchName("Рахматджон Максадов Камолиддинович", staff); m.Status == MatchUnresolved {
			t.Errorf("token alignment should be order-independent, got %s", m.Status)
		}
	})
}

func TestMatchNameHomoglyphsAndTajik(t *testing.T) {
	staff := []Employee{
		emp("Отахонов Фахриддин Улугбекович"),
		emp("Каландаров Собир Миралиевич"),
		emp("Сатыева Заррина Рахмонкуловна"),
		emp("Абдухоликов Манучехр Амирхучаевич"),
		emp("Мирзоева Сабрина Фарходовна"),
	}
	tests := []struct {
		name, in, wantMatch string
	}{
		// Latin O typed instead of Cyrillic О — real row 14 of the export.
		{"latin O homoglyph", "Oтахонов Фахриддин Улугбекович", "Отахонов Фахриддин Улугбекович"},
		// Tajik Қ plus a vowel difference in the given name.
		{"tajik Қ", "Қаландаров Сабир Миралиевич", "Каландаров Собир Миралиевич"},
		// Tajik Ҷ against a Russian-spelled surname.
		{"tajik Ҷ", "Сатҷева Заррина Рахмонкуловна", "Сатыева Заррина Рахмонкуловна"},
		// Patronymic transliteration drift with an otherwise obvious match.
		{"patronymic drift", "Абдухоликов Манучехр Амихуджаевич", "Абдухоликов Манучехр Амирхучаевич"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := MatchName(tt.in, staff)
			if m.Status == MatchUnresolved {
				t.Fatalf("unresolved (top %.3f); expected %q", m.Score, tt.wantMatch)
			}
			var got string
			for _, e := range staff {
				if e.ID == m.UserID {
					got = e.FullName
				}
			}
			if got != tt.wantMatch {
				t.Errorf("matched %q, want %q", got, tt.wantMatch)
			}
		})
	}

	t.Run("folding must not merge genuinely different people", func(t *testing.T) {
		if m := MatchName("Мирзоева Гульшан Фарходовна", staff); m.Status != MatchUnresolved {
			var got string
			for _, e := range staff {
				if e.ID == m.UserID {
					got = e.FullName
				}
			}
			// Гульшан vs Сабрина is a different given name — must not match.
			if got == "Мирзоева Сабрина Фарходовна" {
				t.Errorf("matched a different person: %q (%.3f)", got, m.Score)
			}
		}
	})
}

func TestParseEmploymentProse(t *testing.T) {
	// Real answer from the export. Splitting this on commas produced seven rows
	// of sentence fragments ("где занималась операционным управлением"), which
	// are not employers.
	in := "До работы в ЗАО «КОИНОТИ НАВ» я работала проектным менеджером в компании YALLA.TJ, " +
		"где занималась операционным управлением, аналитикой и оптимизацией процессов, " +
		"представляла проект в акселерационных программах, а также участвовала в найме и обучении сотрудников."
	got := ParseEmployment(in)
	if len(got) != 1 {
		t.Fatalf("prose must stay one row, got %d:\n%#v", len(got), got)
	}
	if got[0].Description != in {
		t.Error("full text must be preserved in Description")
	}
	if got[0].Raw != in {
		t.Error("full text must be preserved in Raw")
	}

	// A genuine list must still split.
	list := "Группа компаний Вавилон, Глобальный ИТ Дистрибьютер МУК Таджикистан, ИТ Интегратор Сомон ИТ"
	if got := ParseEmployment(list); len(got) != 3 {
		t.Errorf("a capitalised list must split into 3, got %d: %#v", len(got), got)
	}
}

// The override map is keyed on NormalizedKey(form name). A typo there would
// silently do nothing, so assert every key is reachable from the exact string
// the questionnaire actually contains.
func TestReviewedNameOverridesAreReachable(t *testing.T) {
	formNames := map[string]string{
		"Ниёззода Мухайё Исмоилчон":     "Ниёззода Мухайёи Исмоилджон",
		"Мачидов Исматулло":             "Маджидов Исматулло Рахматуллоевич",
		"Латипов Хасан Гафорович":       "Латипов Хасанчон Гафорович",
		"Абдуллозода Абдулло Абдулазиз": "Абдуллозода Абдулоджон Абдулазиз",
		"Хамидова Марьям Сухробовна":    "Хамидова Бибимарям Сухробовна",
	}
	if len(reviewedNameOverrides) != len(formNames) {
		t.Fatalf("override map has %d entries, test covers %d",
			len(reviewedNameOverrides), len(formNames))
	}
	for formName, wantEmployee := range formNames {
		got, ok := reviewedNameOverrides[NormalizedKey(formName)]
		if !ok {
			t.Errorf("no override reachable for %q (key %q)", formName, NormalizedKey(formName))
			continue
		}
		if got != wantEmployee {
			t.Errorf("%q -> %q, want %q", formName, got, wantEmployee)
		}
	}

	// The override must resolve against a real employee record.
	staff := []Employee{emp("Ниёззода Мухайёи Исмоилджон"), emp("Хукматзода Абдулло Файзулло")}
	if _, ok := FindByFullName(staff, "Ниёззода Мухайёи Исмоилджон"); !ok {
		t.Error("FindByFullName failed on an exact employee name")
	}

	// The collision the threshold exists to catch must stay unmatched.
	if _, ok := reviewedNameOverrides[NormalizedKey("Хукматов Файзулло")]; ok {
		t.Error("Хукматов Файзулло must NOT be overridden — different person from Хукматзода")
	}
}

// The department alias keys are normalizeDept() output; a typo would silently
// fall through to the name lookup and leave a created employee unplaced.
func TestFormDeptAliasKeys(t *testing.T) {
	forForm := map[string]string{
		"Департамент Закупа и Логистики":       "ДЗЛ",
		"Бухгалтерско-Юридический Департамент": "БЮД",
	}
	if len(formDeptAliases) != len(forForm) {
		t.Fatalf("alias map has %d entries, test covers %d", len(formDeptAliases), len(forForm))
	}
	for answer, wantCode := range forForm {
		got, ok := formDeptAliases[normalizeDept(answer)]
		if !ok {
			t.Errorf("no alias reachable for %q (key %q)", answer, normalizeDept(answer))
			continue
		}
		if got != wantCode {
			t.Errorf("%q -> %q, want %q", answer, got, wantCode)
		}
	}
	// The options that match a department name directly must NOT be aliased —
	// an alias there would mask a rename.
	for _, direct := range []string{
		"Департамент Информационных Технологий", "Финансово-Экономический Департамент",
		"Департамент Человеческих Ресурсов", "Департамент Фармацевтической Промоции",
		"Административно-Хозяйственный Департамент",
	} {
		if _, ok := formDeptAliases[normalizeDept(direct)]; ok {
			t.Errorf("%q should resolve by name, not by alias", direct)
		}
	}
	// Hyphen/space folding must agree with the 1F integration's normaliser.
	if normalizeDept("Финансово-Экономический  Департамент") != "финансово экономический департамент" {
		t.Errorf("normalizeDept folding is wrong: %q", normalizeDept("Финансово-Экономический  Департамент"))
	}
}

// The roster gate must be more suspicious than the matcher: a false "1F has
// them" costs one review, a false "1F lacks them" costs a permanent duplicate.
func TestOneFRosterIsDeliberatelySuspicious(t *testing.T) {
	roster := &OneFRoster{names: []string{
		"Исмоилов Мустафо Шамсулоевич",
		"Хукматзода Абдулло Файзулло",
		"Мирзоев Зиё Толибджонович",
		"Масрур Шарифи",
	}}

	// Near-misses the matcher refuses must still count as "present in 1F",
	// so they are held for review rather than created as duplicates.
	for _, name := range []string{"Исмоилзод Мустафо Шамсуло", "Хукматов Файзулло"} {
		if got, ok := roster.Contains(name); !ok {
			t.Errorf("%q must be treated as possibly present in 1F, got absent", name)
		} else if got == "" {
			t.Errorf("%q matched but reported no 1F spelling", name)
		}
	}

	// Genuinely absent people must remain creatable.
	for _, name := range []string{
		"Фаттоев Масрур Саймуродович", "Тимур Мирзоев Фирдавсович",
		"Бекмурадова Фируза Тулкинжоновна", "Хабиби Мухаммад",
	} {
		if got, ok := roster.Contains(name); ok {
			t.Errorf("%q should be absent from this roster, but matched %q", name, got)
		}
	}

	if rosterSuspicionThreshold >= FuzzyThreshold {
		t.Errorf("roster gate (%.2f) must be looser than the matcher (%.2f)",
			rosterSuspicionThreshold, FuzzyThreshold)
	}
}
