package onef

import "testing"

func TestIsIgnoredDepartment(t *testing.T) {
	tests := []struct {
		name string
		dept string
		want bool
	}{
		{"дусти фарма is out of scope", "Дусти Фарма", true},
		{"инженерная экспертиза is out of scope", "Департамент Инженерной Экспертизы", true},
		{"casing and spacing drift still matches", "  департамент   инженерной экспертизы ", true},

		// Back IN scope as of 2026-08-12 (third flip). Its six people must
		// sync from 1F; migration 0016 gave the ДФП row the real name so they
		// land in the department that holds the competency matrix.
		{"фармацевтическая промоция is in scope", "Департамент Фармацевтической Промоции", false},
		{"...including with spacing drift", "департамент  фармацевтической промоции", false},

		{"unrelated dept syncs", "Финансово-Экономический Департамент", false},
		{"empty name is not ignored", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isIgnoredDepartment(tt.dept); got != tt.want {
				t.Errorf("isIgnoredDepartment(%q) = %v, want %v", tt.dept, got, tt.want)
			}
		})
	}
}

func TestAliasedDeptCode(t *testing.T) {
	tests := []struct{ in, want string }{
		// Grammar difference normalization cannot bridge.
		{"Департамент по Закупкам и Логистике", "ДЗЛ"},
		// Pinned after migration 0016 so a display-name edit can never send
		// these back down the auto-create path (which is what created `ДФП2`
		// and `БИЮД`). Note 1F sends the bookkeeping dept with a double space.
		{"Бухгалтерский и Юридический  Департамент", "БЮД"},
		{"Бухгалтерский и Юридический Департамент", "БЮД"},
		{"Департамент Фармацевтической Промоции", "ДФП"},
		{"Департамент Информационных Технологий", ""},
	}
	for _, tt := range tests {
		if got := aliasedDeptCode(tt.in); got != tt.want {
			t.Errorf("aliasedDeptCode(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizeDeptName(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Финансово-Экономический Департамент", "финансово экономический департамент"},
		{"Финансово  Экономический\tДепартамент", "финансово экономический департамент"},
		{"  Департамент Закупки и Логистики  ", "департамент закупки и логистики"},
	}
	for _, tt := range tests {
		if got := normalizeDeptName(tt.in); got != tt.want {
			t.Errorf("normalizeDeptName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
