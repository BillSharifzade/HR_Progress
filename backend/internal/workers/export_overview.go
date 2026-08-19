package workers

import (
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"
)

const companyName = "ЗАО «КОИНОТИ НАВ»"

// cover writes the opening sheet. It deliberately does not reuse the data-sheet
// scaffolding: there is no header strip to freeze and nothing to filter, only a
// report cover with a handful of stacked tables.
type cover struct {
	b    *book
	name string
	row  int
}

func (o *cover) cell(colIdx int, v any, st cellStyle) {
	ref, _ := excelize.CoordinatesToCellName(colIdx+1, o.row)
	if v != nil {
		o.b.f.SetCellValue(o.name, ref, v)
	}
	o.b.f.SetCellStyle(o.name, ref, ref, o.b.style(st))
}

// span styles a run of columns as one cell so a long value can cross the
// narrow numeric columns without widening them.
func (o *cover) span(fromCol, toCol int, v any, st cellStyle) {
	from, _ := excelize.CoordinatesToCellName(fromCol+1, o.row)
	to, _ := excelize.CoordinatesToCellName(toCol+1, o.row)
	o.b.f.MergeCell(o.name, from, to)
	if v != nil {
		o.b.f.SetCellValue(o.name, from, v)
	}
	o.b.f.SetCellStyle(o.name, from, to, o.b.style(st))
}

const coverCols = 7

// section starts a titled block: a full-width brand band with the block name.
func (o *cover) section(title string) {
	o.row++
	o.b.f.SetRowHeight(o.name, o.row, 8)
	o.row++
	o.span(0, coverCols-1, title, cellStyle{
		fill: clrBrand, color: clrHeaderText, bold: true, size: 10.5, align: "left"})
	o.b.f.SetRowHeight(o.name, o.row, 22)
	o.row++
}

func headStyle(align string) cellStyle {
	return cellStyle{fill: clrCover, color: clrBrandDeep, bold: true, size: 9.5,
		align: align, border: true}
}

// head writes a table header row; widths come from the sheet's column setup.
func (o *cover) head(titles ...string) {
	for i, t := range titles {
		align := "center"
		if i == 0 {
			align = "left"
		}
		o.cell(i, t, headStyle(align))
	}
	o.b.f.SetRowHeight(o.name, o.row, 20)
	o.row++
}

// line writes one table row. Column 0 is the label; the rest are figures.
func (o *cover) line(band int, fmts []string, values ...any) {
	fill := clrBandA
	if band%2 == 1 {
		fill = clrBandB
	}
	for i, v := range values {
		align, numFmt := "center", ""
		if i < len(fmts) {
			numFmt = fmts[i]
		}
		if i == 0 {
			align = "left"
		}
		o.cell(i, v, cellStyle{fill: fill, align: align, numFmt: numFmt, border: true})
	}
	o.row++
}

// kv writes a label/value pair, the value spanning the numeric columns.
func (o *cover) kv(label string, value any) {
	o.cell(0, label, cellStyle{color: clrMuted, size: 10})
	o.span(1, 4, value, cellStyle{bold: true, align: "left"})
	o.row++
}

func buildOverview(c *buildCtx, counts map[string]int) error {
	name := shOverview
	if _, err := c.b.f.NewSheet(name); err != nil {
		return err
	}
	// FitToPage is what actually activates FitToWidth below; without it Excel
	// and LibreOffice both ignore the fit and print the cover clipped.
	c.b.f.SetSheetProps(name, &excelize.SheetPropsOptions{
		TabColorRGB: ptr(clrBrandDeep), FitToPage: ptr(true)})
	c.b.f.SetSheetView(name, 0, &excelize.ViewOptions{ShowGridLines: ptr(false)})
	c.b.f.SetColWidth(name, "A", "A", 38)
	c.b.f.SetColWidth(name, "B", "G", 15)
	c.b.f.SetPageLayout(name, &excelize.PageLayoutOptions{
		Orientation: ptr("portrait"), FitToWidth: ptr(1), FitToHeight: ptr(0)})

	o := &cover{b: c.b, name: name, row: 1}
	total := len(c.ds.Workers)

	o.span(0, coverCols-1, "Реестр сотрудников", cellStyle{
		bold: true, size: 20, color: clrBrandDeep})
	c.b.f.SetRowHeight(name, o.row, 32)
	o.row++
	o.span(0, coverCols-1, companyName+" · Платформа развития персонала",
		cellStyle{size: 11, color: clrInk})
	o.row++
	o.span(0, coverCols-1,
		"Сформировано: "+c.ds.GeneratedAt.Local().Format("02.01.2006 в 15:04"),
		cellStyle{size: 9.5, color: clrMuted})
	c.b.f.SetRowHeight(name, o.row, 16)
	o.row++

	// ── Параметры выгрузки ──
	o.section("ПАРАМЕТРЫ ВЫГРУЗКИ")
	o.kv("Сотрудников в файле", total)
	if len(c.ds.Filters) == 0 {
		o.kv("Фильтры", "не применялись — выгружены все доступные сотрудники")
	}
	for _, f := range c.ds.Filters {
		o.kv(f.Label, f.Value)
	}
	o.kv("Сортировка", "департамент → отдел → грейд (от старшего) → ФИО")
	o.kv("Ключ сотрудника", "колонка «№» одинакова на всех листах файла")

	// ── Состав файла ──
	o.section("СОСТАВ ФАЙЛА")
	// The description column runs to the sheet edge, so its header spans the
	// same width instead of stopping short of the cells beneath it.
	o.cell(0, "Лист", headStyle("left"))
	o.cell(1, "Строк", headStyle("center"))
	o.span(2, coverCols-1, "Что содержит", headStyle("left"))
	o.b.f.SetRowHeight(name, o.row, 20)
	o.row++
	contents := []struct {
		sheet, about string
	}{
		{shRegister, "Полный реестр: место в оргструктуре, контакты, стаж, роли, сводка по данным"},
		{shProfile, "Анкета цифрового профиля: образование, карьерная цель, мобильность, обучение"},
		{shLanguages, "Матрица владения языками: уровень CEFR по каждому языку"},
		{shExperience, "Опыт работы до прихода в компанию"},
		{shCerts, "Сертификаты: кем и когда выдан, ссылка или приложенный файл"},
		{shSurvey, "Открытые ответы анкеты: экспертиза, проекты, темы для обучения"},
		{shScores, "Оценки компетенций по кампаниям ассессмента с интерпретациями"},
		{shRoles, "Назначенные роли в системе и область их действия"},
		{shHistory, "События внутри компании: приём, повышения, переводы, комментарии"},
	}
	for i, row := range contents {
		fill := clrBandA
		if i%2 == 1 {
			fill = clrBandB
		}
		o.cell(0, row.sheet, cellStyle{fill: fill, bold: true, border: true})
		o.cell(1, counts[row.sheet], cellStyle{
			fill: fill, align: "center", numFmt: "int", border: true})
		o.span(2, coverCols-1, row.about, cellStyle{fill: fill, size: 9.5, color: clrMuted, border: true})
		o.row++
	}

	// ── По департаментам ──
	o.section("СОТРУДНИКИ ПО ДЕПАРТАМЕНТАМ")
	o.head("Департамент", "Всего", "Активных", "Отделов", "Доля")
	numFmts := []string{"", "int", "int", "int", "pct"}
	for i, d := range departmentStats(c.ds) {
		o.line(i, numFmts, d.name, d.total, d.active, countVal(d.sections), share(d.total, total))
	}
	// Written as an ordinary row, then re-styled as the totals line.
	o.line(0, numFmts, "Итого", total, activeCount(c.ds), nil, share(total, total))
	o.styleTotalRow(numFmts)

	// ── По грейдам ──
	o.section("РАСПРЕДЕЛЕНИЕ ПО ГРЕЙДАМ")
	o.head("Грейд", "Уровень", "Сотрудников", "Доля")
	for i, g := range gradeStats(c.ds) {
		o.line(i, []string{"", "int", "int", "pct"}, g.name, intVal(g.level), g.total, share(g.total, total))
	}

	// ── Заполненность ──
	o.section("ЗАПОЛНЕННОСТЬ ДАННЫХ")
	o.head("Показатель", "Заполнено", "Из", "Доля")
	for i, m := range completeness(c.ds) {
		o.line(i, []string{"", "int", "int", "pct"}, m.label, m.filled, total, share(m.filled, total))
	}

	o.row++
	o.span(0, coverCols-1,
		"Файл содержит персональные данные сотрудников. Распространение — только внутри компании.",
		cellStyle{italic: true, size: 9, color: clrMuted})
	return nil
}

// styleTotalRow re-paints the row just written as a totals line. It takes the
// same number formats the table body uses so the highlight stops exactly where
// the table does rather than trailing off across the empty columns.
func (o *cover) styleTotalRow(fmts []string) {
	r := o.row - 1
	for i, f := range fmts {
		ref, _ := excelize.CoordinatesToCellName(i+1, r)
		align := "center"
		if i == 0 {
			align = "left"
		}
		o.b.f.SetCellStyle(o.name, ref, ref, o.b.style(cellStyle{
			fill: clrCover, bold: true, color: clrBrandDeep,
			align: align, numFmt: f, border: true}))
	}
}

func share(part, whole int) any {
	if whole == 0 {
		return nil
	}
	return float64(part) / float64(whole)
}

func activeCount(ds *ExportDataset) int {
	n := 0
	for i := range ds.Workers {
		if ds.Workers[i].IsActive {
			n++
		}
	}
	return n
}

type deptStat struct {
	name          string
	total, active int
	sections      int
	sortKey       string
}

func departmentStats(ds *ExportDataset) []deptStat {
	idx := map[string]*deptStat{}
	sections := map[string]map[string]bool{}
	order := []string{}
	for i := range ds.Workers {
		w := &ds.Workers[i]
		key := deref(w.DepartmentName)
		label := key
		if label == "" {
			label = "Без департамента"
		}
		d, ok := idx[key]
		if !ok {
			d = &deptStat{name: label, sortKey: strings.ToLower(key)}
			idx[key] = d
			sections[key] = map[string]bool{}
			order = append(order, key)
		}
		d.total++
		if w.IsActive {
			d.active++
		}
		if s := deref(w.SectionName); s != "" {
			sections[key][s] = true
		}
	}
	out := make([]deptStat, 0, len(order))
	for _, key := range order {
		d := idx[key]
		d.sections = len(sections[key])
		out = append(out, *d)
	}
	// Unassigned people sort last, matching the register's own ordering.
	sort.SliceStable(out, func(i, j int) bool {
		if (out[i].sortKey == "") != (out[j].sortKey == "") {
			return out[j].sortKey == ""
		}
		return out[i].sortKey < out[j].sortKey
	})
	return out
}

type gradeStat struct {
	name  string
	level *int
	total int
}

func gradeStats(ds *ExportDataset) []gradeStat {
	idx := map[string]*gradeStat{}
	for i := range ds.Workers {
		w := &ds.Workers[i]
		key := deref(w.GradeName)
		label := key
		if label == "" {
			label = "Грейд не присвоен"
		}
		g, ok := idx[key]
		if !ok {
			g = &gradeStat{name: label, level: w.GradeLevel}
			idx[key] = g
		}
		g.total++
	}
	out := make([]gradeStat, 0, len(idx))
	for _, g := range idx {
		out = append(out, *g)
	}
	// Senior first, ungraded last — the same reading order as the register.
	sort.Slice(out, func(i, j int) bool {
		li, lj := -1, -1
		if out[i].level != nil {
			li = *out[i].level
		}
		if out[j].level != nil {
			lj = *out[j].level
		}
		if li != lj {
			return li > lj
		}
		return out[i].name < out[j].name
	})
	return out
}

type completenessStat struct {
	label  string
	filled int
}

func completeness(ds *ExportDataset) []completenessStat {
	var profile, langs, exp, certs, scores, hired, birth, email, grade, section int
	for i := range ds.Workers {
		w := &ds.Workers[i]
		if w.Profile != nil {
			profile++
		}
		if len(w.Languages) > 0 {
			langs++
		}
		if len(w.Experience) > 0 {
			exp++
		}
		if len(w.Certifications) > 0 {
			certs++
		}
		if len(w.Scores) > 0 {
			scores++
		}
		if w.HiredAt != nil {
			hired++
		}
		if w.BirthDate != nil {
			birth++
		}
		if str(w.Email) != nil {
			email++
		}
		if str(w.GradeName) != nil {
			grade++
		}
		if str(w.SectionName) != nil {
			section++
		}
	}
	return []completenessStat{
		{"Анкета цифрового профиля", profile},
		{"Владение языками", langs},
		{"Опыт до компании", exp},
		{"Сертификаты", certs},
		{"Оценки компетенций", scores},
		{"Дата приёма", hired},
		{"Дата рождения", birth},
		{"Email", email},
		{"Грейд присвоен", grade},
		{"Отдел указан", section},
	}
}
