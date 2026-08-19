package workers

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// Workbook palette. The header indigo is the app's brand primary (#4F46E5), so
// a printed register and the screen it came from read as the same product.
const (
	clrBrand      = "4F46E5"
	clrBrandDeep  = "3730A3"
	clrHeaderText = "FFFFFF"
	clrInk        = "1F2233"
	clrMuted      = "6B7280"
	clrGrid       = "E3E5F0"
	clrBandA      = "FFFFFF"
	clrBandB      = "F5F6FC"
	clrCover      = "EEF0FD"

	clrPosText = "166534"
	clrPosFill = "DCFCE7"
	clrMidText = "1E40AF"
	clrMidFill = "DBEAFE"
	clrLowText = "92400E"
	clrLowFill = "FEF3C7"
	clrOffText = "6B7280"
)

// Sheet names, in workbook order.
const (
	shOverview   = "Обзор"
	shRegister   = "Сотрудники"
	shProfile    = "Цифровой профиль"
	shLanguages  = "Языки"
	shExperience = "Опыт работы"
	shCerts      = "Сертификаты"
	shSurvey     = "Анкета"
	shScores     = "Оценки"
	shRoles      = "Роли и доступ"
	shHistory    = "История"
)

var userRoleLabels = map[string]string{
	"HR_ADMIN":     "HR-администратор",
	"DEPT_HEAD":    "Руководитель департамента",
	"SECTION_HEAD": "Руководитель отдела",
	"ASSESSOR":     "Ассессор",
	"PRECEPTOR":    "Наставник",
	"ATS":          "Центр обучения (AtS)",
	"BOOK_SPACE":   "Библиотека материалов",
}

var assessorRoleLabels = map[string]string{
	"HEAD":      "Рук. отдела",
	"DEPT_HEAD": "Рук. департамента",
	"HRA":       "Ассессор",
	"DCR_HEAD":  "Рук. ДЧР",
}

var competencyKindLabels = map[string]string{
	"LK": "Личностные",
	"UK": "Управленческие",
	"PK": "Профессиональные",
}

var campaignStatusLabels = map[string]string{
	"draft":        "Черновик",
	"assigned":     "Назначена",
	"in_progress":  "В процессе",
	"admin_review": "На проверке",
	"confirmed":    "Подтверждена",
	"published":    "Опубликована",
}

var historyKindLabels = map[string]string{
	"HIRED":               "Принят",
	"PROMOTED":            "Повышение",
	"TRANSFERRED":         "Перевод",
	"EXTERNAL_EXPERIENCE": "Внешний опыт",
	"COMMENT":             "Комментарий",
	"OTHER":               "Другое",
}

// knownLanguages fixes the column order of the language matrix to the order the
// questionnaire itself listed them in; anything else follows alphabetically.
var knownLanguages = []string{"Таджикский", "Русский", "Английский", "Китайский", "Немецкий", "Турецкий"}

// ─── styling ────────────────────────────────────────────────────────────────

// cellStyle is the full description of one cell's look. It doubles as the cache
// key: the workbook needs a few dozen combinations of banding, alignment and
// number format, and Excel starts to struggle well before a style per cell.
type cellStyle struct {
	fill   string
	align  string // "", "center", "right"
	numFmt string // "", "date", "datetime", "num1", "int", "pct"
	wrap   bool
	bold   bool
	italic bool
	size   float64
	color  string
	border bool
}

var numFormats = map[string]string{
	"date":     "dd.mm.yyyy",
	"datetime": "dd.mm.yyyy hh:mm",
	"num1":     "0.0",
	"int":      "0",
	"pct":      "0.0%",
}

type book struct {
	f      *excelize.File
	styles map[cellStyle]int
}

func newBook() *book {
	return &book{f: excelize.NewFile(), styles: map[cellStyle]int{}}
}

func (b *book) style(s cellStyle) int {
	if id, ok := b.styles[s]; ok {
		return id
	}
	st := &excelize.Style{
		Font: &excelize.Font{
			Bold:   s.bold,
			Italic: s.italic,
			Size:   or(s.size, 10),
			Color:  orStr(s.color, clrInk),
			Family: "Calibri",
		},
		Alignment: &excelize.Alignment{
			Horizontal: s.align,
			Vertical:   "center",
			WrapText:   s.wrap,
		},
	}
	if s.fill != "" {
		st.Fill = excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{s.fill}}
	}
	if s.border {
		st.Border = []excelize.Border{
			{Type: "left", Style: 1, Color: clrGrid},
			{Type: "right", Style: 1, Color: clrGrid},
			{Type: "top", Style: 1, Color: clrGrid},
			{Type: "bottom", Style: 1, Color: clrGrid},
		}
	}
	if f, ok := numFormats[s.numFmt]; ok {
		st.CustomNumFmt = &f
	}
	id, err := b.f.NewStyle(st)
	if err != nil {
		id = 0
	}
	b.styles[s] = id
	return id
}

func or(v, fallback float64) float64 {
	if v == 0 {
		return fallback
	}
	return v
}

func orStr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

// ─── sheet scaffolding ──────────────────────────────────────────────────────

type col struct {
	title string
	width float64
	align string
	fmt   string
	wrap  bool
}

const (
	titleRow    = 1
	subtitleRow = 2
	spacerRow   = 3
	headerRow   = 4
	firstData   = 5
)

type sheet struct {
	b    *book
	name string
	cols []col
	row  int
}

// newSheet lays out the standard page furniture every data sheet shares: a
// title band, a subtitle line, the header strip, frozen panes and an
// autofilter. Gridlines are turned off — the table draws its own.
func (b *book) newSheet(name, title, subtitle string, cols []col, freezeCols int) (*sheet, error) {
	if _, err := b.f.NewSheet(name); err != nil {
		return nil, err
	}
	b.f.SetSheetProps(name, &excelize.SheetPropsOptions{TabColorRGB: ptr(clrBrand)})
	b.f.SetSheetView(name, 0, &excelize.ViewOptions{ShowGridLines: ptr(false)})

	last, _ := excelize.ColumnNumberToName(len(cols))
	for i, c := range cols {
		n, _ := excelize.ColumnNumberToName(i + 1)
		b.f.SetColWidth(name, n, n, c.width)
	}

	b.f.MergeCell(name, "A1", last+"1")
	b.f.SetCellValue(name, "A1", title)
	b.f.SetCellStyle(name, "A1", last+"1", b.style(cellStyle{
		bold: true, size: 16, color: clrBrandDeep, align: "left"}))
	b.f.SetRowHeight(name, titleRow, 26)

	b.f.MergeCell(name, "A2", last+"2")
	b.f.SetCellValue(name, "A2", subtitle)
	b.f.SetCellStyle(name, "A2", last+"2", b.style(cellStyle{size: 9.5, color: clrMuted}))
	b.f.SetRowHeight(name, subtitleRow, 15)
	b.f.SetRowHeight(name, spacerRow, 6)

	hdr := b.style(cellStyle{
		fill: clrBrand, color: clrHeaderText, bold: true,
		align: "center", wrap: true, border: true})
	for i, c := range cols {
		cell, _ := excelize.CoordinatesToCellName(i+1, headerRow)
		b.f.SetCellValue(name, cell, c.title)
		b.f.SetCellStyle(name, cell, cell, hdr)
	}
	b.f.SetRowHeight(name, headerRow, 32)

	top, _ := excelize.CoordinatesToCellName(freezeCols+1, firstData)
	b.f.SetPanes(name, &excelize.Panes{
		Freeze: true, XSplit: freezeCols, YSplit: headerRow, TopLeftCell: top,
		ActivePane: "bottomRight",
		Selection:  []excelize.Selection{{SQRef: top, ActiveCell: top, Pane: "bottomRight"}},
	})
	b.f.AutoFilter(name, fmt.Sprintf("A%d:%s%d", headerRow, last, headerRow), nil)

	// Print-ready by default: landscape with the header row repeated on every
	// page. Deliberately no fit-to-width — squeezing a thirty-column register
	// onto one sheet of paper makes it unreadable; spilling sideways with a
	// repeated header does not.
	b.f.SetPageLayout(name, &excelize.PageLayoutOptions{Orientation: ptr("landscape")})
	b.f.SetDefinedName(&excelize.DefinedName{
		Name:     "_xlnm.Print_Titles",
		RefersTo: fmt.Sprintf("'%s'!$%d:$%d", name, headerRow, headerRow),
		Scope:    name,
	})

	return &sheet{b: b, name: name, cols: cols, row: firstData}, nil
}

// Row height bounds for sheets that wrap text. Heights are set explicitly
// rather than left to the reader's auto-fit, which Excel and LibreOffice
// disagree about; the ceiling stops one 1200-character questionnaire answer
// from turning its row into a page of its own.
const (
	minRowHeight = 18
	maxRowHeight = 108
	lineHeight   = 13
)

// addRow writes one data row. A nil value leaves the cell genuinely empty so
// Excel's filters and pivots treat it as missing rather than as the text "—".
func (s *sheet) addRow(band int, values ...any) int {
	fill := clrBandA
	if band%2 == 1 {
		fill = clrBandB
	}
	r := s.row
	for i, c := range s.cols {
		if i >= len(values) {
			break
		}
		cell, _ := excelize.CoordinatesToCellName(i+1, r)
		if v := values[i]; v != nil {
			s.b.f.SetCellValue(s.name, cell, v)
		}
		s.b.f.SetCellStyle(s.name, cell, cell, s.b.style(cellStyle{
			fill: fill, align: c.align, numFmt: c.fmt, wrap: c.wrap, border: true}))
	}
	s.setRowHeight(r, values)
	s.row++
	return r
}

// setRowHeight sizes a row to its longest wrapped cell, within bounds. Sheets
// with no wrapped column keep the default height.
func (s *sheet) setRowHeight(r int, values []any) {
	lines := 0
	for i, c := range s.cols {
		if !c.wrap || i >= len(values) {
			continue
		}
		txt, ok := values[i].(string)
		if !ok || txt == "" {
			continue
		}
		perLine := int(c.width * 1.05)
		if perLine < 8 {
			perLine = 8
		}
		n := (len([]rune(txt))+perLine-1)/perLine + strings.Count(txt, "\n")
		if n > lines {
			lines = n
		}
	}
	if lines == 0 {
		return
	}
	h := float64(lines)*lineHeight + 5
	if h < minRowHeight {
		h = minRowHeight
	}
	if h > maxRowHeight {
		h = maxRowHeight
	}
	s.b.f.SetRowHeight(s.name, r, h)
}

// paint overrides the look of a single cell in the row addRow just wrote —
// used for the handful of semantic cells (status, CEFR level) that carry a
// colour of their own.
func (s *sheet) paint(row, colIdx int, fg, bg string) {
	cell, _ := excelize.CoordinatesToCellName(colIdx+1, row)
	c := s.cols[colIdx]
	s.b.f.SetCellStyle(s.name, cell, cell, s.b.style(cellStyle{
		fill: bg, color: fg, align: c.align, numFmt: c.fmt,
		wrap: c.wrap, bold: true, border: true}))
}

// colIndex finds a column by its header text. Painting a semantic cell by
// counting offsets breaks silently the moment a column is inserted, so the
// lookup is by title and panics loudly on a typo instead.
func colIndex(cols []col, title string) int {
	for i, c := range cols {
		if c.title == title {
			return i
		}
	}
	panic("export: no column titled " + title)
}

// empty writes the "no records" note that keeps a sheet's structure stable when
// the platform holds nothing for it yet.
func (s *sheet) empty(note string) {
	last, _ := excelize.ColumnNumberToName(len(s.cols))
	s.b.f.MergeCell(s.name, fmt.Sprintf("A%d", s.row), fmt.Sprintf("%s%d", last, s.row))
	s.b.f.SetCellValue(s.name, fmt.Sprintf("A%d", s.row), note)
	s.b.f.SetCellStyle(s.name, fmt.Sprintf("A%d", s.row), fmt.Sprintf("%s%d", last, s.row),
		s.b.style(cellStyle{fill: clrBandA, italic: true, color: clrMuted, align: "center", border: true}))
	s.b.f.SetRowHeight(s.name, s.row, 26)
	s.row++
}

func ptr[T any](v T) *T { return &v }

// ─── entry point ────────────────────────────────────────────────────────────

// buildCtx is the shared state every sheet builder needs: the workbook, the
// data, the department banding, and the subtitle line each sheet repeats.
type buildCtx struct {
	b     *book
	ds    *ExportDataset
	bands []int
	sub   string
}

// WriteExportWorkbook renders the dataset as a formatted XLSX workbook.
func WriteExportWorkbook(w io.Writer, ds *ExportDataset) error {
	b := newBook()
	defer b.f.Close()

	stamp := ds.GeneratedAt.Local().Format("02.01.2006 15:04")
	c := &buildCtx{
		b:  b,
		ds: ds,
		// Departments alternate their background tint, which groups the
		// register visually without breaking the flat, filterable table
		// underneath it.
		bands: departmentBands(ds.Workers),
		sub: fmt.Sprintf("Платформа развития персонала · выгружено %s · сотрудников: %d",
			stamp, len(ds.Workers)),
	}

	counts := map[string]int{}
	for _, build := range []func(*buildCtx) (string, int, error){
		buildRegister, buildProfileSheet, buildLanguageSheet, buildExperienceSheet,
		buildCertSheet, buildSurveySheet, buildScoreSheet, buildRoleSheet, buildHistorySheet,
	} {
		name, n, err := build(c)
		if err != nil {
			return err
		}
		counts[name] = n
	}

	if err := buildOverview(c, counts); err != nil {
		return err
	}

	// The cover comes first and is what opens; Sheet1 is excelize's default and
	// has no place in the finished file.
	if err := b.f.MoveSheet(shOverview, shRegister); err != nil {
		return err
	}
	if err := b.f.DeleteSheet("Sheet1"); err != nil {
		return err
	}
	b.f.SetActiveSheet(0)

	b.f.SetDocProps(&excelize.DocProperties{
		Title:       "Реестр сотрудников",
		Subject:     "Выгрузка данных сотрудников",
		Creator:     "HR Progress",
		Description: c.sub,
		Language:    "ru-RU",
		Created:     ds.GeneratedAt.UTC().Format(time.RFC3339),
		Modified:    ds.GeneratedAt.UTC().Format(time.RFC3339),
	})

	_, err := b.f.WriteTo(w)
	return err
}

func departmentBands(ws []ExportWorker) []int {
	bands := make([]int, len(ws))
	band, prev := 0, ""
	for i, w := range ws {
		cur := deref(w.DepartmentName)
		if i > 0 && cur != prev {
			band++
		}
		bands[i] = band
		prev = cur
	}
	return bands
}
