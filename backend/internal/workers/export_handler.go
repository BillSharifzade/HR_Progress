package workers

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"hrprogress/internal/httpx"
)

const xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// export streams the whole worker register as a formatted XLSX workbook.
//
// The workbook is rendered into memory before the first byte goes out: writing
// straight to the ResponseWriter would commit a 200 that cannot be taken back
// if the render fails halfway, leaving the browser with a truncated file that
// still looks like a successful download.
func (h *Handler) export(w http.ResponseWriter, r *http.Request) {
	f, err := parseListFilter(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_PARAM", err.Error())
		return
	}

	ds, err := h.repo.ExportDataset(r.Context(), f)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	deptName, sectionName, gradeName, err := h.repo.LookupNames(
		r.Context(), f.DepartmentID, f.SectionID, f.GradeID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	ds.Filters = describeFilters(f, deptName, sectionName, gradeName)

	var buf bytes.Buffer
	if err := WriteExportWorkbook(&buf, ds); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "EXPORT_FAILED", err.Error())
		return
	}

	name := fmt.Sprintf("Сотрудники_%s.xlsx", ds.GeneratedAt.Local().Format("2006-01-02_1504"))
	ascii := fmt.Sprintf("workers_%s.xlsx", ds.GeneratedAt.Local().Format("2006-01-02_1504"))

	w.Header().Set("Content-Type", xlsxContentType)
	// Both forms are sent: the quoted ASCII name for older clients, the RFC 5987
	// form for everything that can read the Cyrillic original.
	w.Header().Set("Content-Disposition", fmt.Sprintf(
		`attachment; filename="%s"; filename*=UTF-8''%s`, ascii, url.PathEscape(name)))
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

// describeFilters turns the applied filter set into the lines printed on the
// cover sheet, so a partial export is self-evidently partial.
func describeFilters(f ListFilter, dept, section, grade string) []ExportFilterNote {
	notes := []ExportFilterNote{}
	add := func(label, name string, set bool) {
		if !set {
			return
		}
		if name == "" {
			name = "(запись не найдена)"
		}
		notes = append(notes, ExportFilterNote{Label: label, Value: name})
	}
	add("Фильтр: департамент", dept, f.DepartmentID != nil)
	add("Фильтр: отдел", section, f.SectionID != nil)
	add("Фильтр: грейд", grade, f.GradeID != nil)
	if f.Search != "" {
		notes = append(notes, ExportFilterNote{Label: "Фильтр: поиск", Value: f.Search})
	}
	scope := "только активные сотрудники"
	if f.IncludeInactive {
		scope = "включая неактивных сотрудников"
	}
	notes = append(notes, ExportFilterNote{Label: "Охват", Value: scope})
	return notes
}
