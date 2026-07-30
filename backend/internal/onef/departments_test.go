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

		// ДФП was un-ignored on 2026-07-29 — it must sync like any other dept.
		{"фармацевтическая промоция syncs", "Департамент Фармацевтической Промоции", false},

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
