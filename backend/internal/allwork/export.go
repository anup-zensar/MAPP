package allwork

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/platform/sheet"
)

// Export of the current view to a spreadsheet or CSV file (FR-032 – FR-034, D-120): every matching and
// context row, collapsed branches included, with Level, Path and Name, the view's columns and the totals
// row; money as numbers in the display currency, dates as dates, statuses as words; the notes — view, year,
// filters, search, Show levels, display currency, exporter, time — before the data.

// The content types of the files (contract exportAllWork).
const (
	contentTypeXLSX = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	contentTypeCSV  = "text/csv"
)

// localColumns are the columns of amounts in a line item's own currency: each is followed by a Currency
// column in the file (FR-033).
var localColumns = []string{"lineItem.localPlannedCost", "lineItem.contractedAmount"}

// viewNames are the views as people read them: the sheet name and the file name (FR-034).
var viewNames = map[string]string{ViewGrid: "Grid", ViewCalendar: "Calendar", ViewBudget: "Budget & cost", ViewBreakdown: "Breakdown"}

// showLevelsWords are the Show levels choices as people read them.
var showLevelsWords = map[string]string{
	ShowEverything: "Everything", ShowCompleteBranches: "Complete branches only",
	ShowCampaignsWithoutInitiatives: "Campaigns without initiatives", ShowInitiativesWithoutActivities: "Initiatives without activities",
}

// File is an exported file.
type File struct {
	Name        string // "MAPP All work – Grid – 2026 – 18 Oct 2026.xlsx"
	ContentType string
	Body        []byte
}

// Export returns the view of q as a spreadsheet (.xlsx) or CSV file for a (FR-032 – FR-034); unassigned says
// whether the unassigned planned cost rows are included, as the screen's switch does (FR-012). It reads in
// one transaction and changes nothing.
func (s *Service) Export(ctx context.Context, a actor.Actor, q Query, format string, unassigned bool) (File, error) {
	var in input
	var b built
	var breakdown *Breakdown
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		if in, err = s.input(ctx, a, q); err != nil {
			return err
		}
		if in.rateNotes, err = s.src.RatesUsed(ctx); err != nil { // 016 FR-009
			return err
		}
		if q.View == ViewBreakdown { // 016 FR-043: the group table
			result, err := breakdownOf(in)
			if err != nil {
				return err
			}
			breakdown = &result
		}
		b = build(in)
		return nil
	})
	if err != nil {
		return File{}, err
	}
	if breakdown != nil {
		return exportBreakdown(in, b, *breakdown, a, format)
	}
	return exportFile(in, b, a, format, unassigned)
}

// exportBreakdown writes the Breakdown view as a file of format (016 FR-043): the notes, then the group table.
func exportBreakdown(in input, b built, br Breakdown, a actor.Actor, format string) (File, error) {
	e := exporter{in: in, b: b}
	b.result.View = ViewBreakdown
	e.b = b
	notes := e.notes(a)
	notes = append(notes, [2]string{"Break down by", br.By.Label})
	if br.ThenBy != nil {
		notes = append(notes, [2]string{"Then by", br.ThenBy.Label})
	}
	for _, note := range br.Notes {
		notes = append(notes, [2]string{"Note", note})
	}
	return writeFile(b.result, in, notes, breakdownTable(br), format)
}

// breakdownTable is the group table of a Breakdown: each group, then its "then by" groups, and the totals.
func breakdownTable(br Breakdown) [][]sheet.Cell {
	withCurrency := func(label string) sheet.Cell { return sheet.Bold(label + " (" + br.Currency + ")") }
	heading := []sheet.Cell{sheet.Bold(br.By.Label)}
	if br.ThenBy != nil {
		heading = append(heading, sheet.Bold(br.ThenBy.Label))
	}
	heading = append(heading, sheet.Bold("Activities"), sheet.Bold("Line items"), withCurrency("Planned cost"),
		sheet.Bold("Share (%)"), withCurrency("Budget"), sheet.Bold("Budget used (%)"))
	table := [][]sheet.Cell{heading}
	line := func(first, second string, activities, lineItems int, planned money.Money, share string, budget *money.Money, used *int) []sheet.Cell {
		cells := []sheet.Cell{sheet.Text(first)}
		if br.ThenBy != nil {
			cells = append(cells, sheet.Text(second))
		}
		cells = append(cells, sheet.Number(int64(activities)), sheet.Number(int64(lineItems)), sheet.Money(planned.Amount), sheet.Text(share))
		if budget != nil {
			cells = append(cells, sheet.Money(budget.Amount))
		} else {
			cells = append(cells, sheet.Cell{})
		}
		if used != nil {
			cells = append(cells, sheet.Percent(int64(*used)))
		} else {
			cells = append(cells, sheet.Cell{})
		}
		return cells
	}
	for _, g := range br.Groups {
		table = append(table, line(g.Label, "", g.Activities, g.LineItems, g.PlannedCost, g.Share, g.Budget, g.BudgetUsedPercent))
		for _, c := range g.Children {
			table = append(table, line("", c.Label, c.Activities, c.LineItems, c.PlannedCost, c.Share, c.Budget, c.BudgetUsedPercent))
		}
	}
	t := br.Total
	return append(table, line("Total", "", t.Activities, t.LineItems, t.PlannedCost, t.Share, t.Budget, t.BudgetUsedPercent))
}

// exportFile writes the worked-out view b as a file of format.
func exportFile(in input, b built, a actor.Actor, format string, unassigned bool) (File, error) {
	e := exporter{in: in, b: b, unassigned: unassigned}
	return writeFile(b.result, in, e.notes(a), e.table(), format)
}

// writeFile writes the notes and the table of the view r as a file of format: a CSV, or a spreadsheet with the
// notes on its first sheet "About this export".
func writeFile(r Result, in input, notes [][2]string, table [][]sheet.Cell, format string) (File, error) {
	file := File{Name: fileName(r, in, format)}
	var body bytes.Buffer
	switch format {
	case FormatCSV:
		file.ContentType = contentTypeCSV
		if err := sheet.WriteCSV(&body, notes, table); err != nil {
			return File{}, err
		}
	default:
		file.ContentType = contentTypeXLSX
		about := make([][]sheet.Cell, 0, len(notes))
		for _, n := range notes {
			about = append(about, []sheet.Cell{sheet.Bold(n[0]), sheet.Text(n[1])})
		}
		wb := sheet.Workbook{Sheets: []sheet.Sheet{
			{Name: "About this export", Rows: about},
			{Name: viewNames[r.View], Rows: table},
		}}
		if err := sheet.WriteXLSX(&body, wb); err != nil {
			return File{}, err
		}
	}
	file.Body = body.Bytes()
	return file, nil
}

// fileName is "MAPP All work – <view> – <year> – <DD MMM YYYY>.<format>" (FR-034); over several years
// "2026–2027" (016).
func fileName(r Result, in input, format string) string {
	return fmt.Sprintf("MAPP All work – %s – %s – %s.%s", viewNames[r.View], yearsText(r, "–"), date.Format(in.today), format)
}

// yearsText is the year of a view, or its range "2026<sep>2027" (016 FR-034).
func yearsText(r Result, sep string) string {
	if r.YearTo > r.YearFrom && r.YearFrom != 0 {
		return strconv.Itoa(r.YearFrom) + sep + strconv.Itoa(r.YearTo)
	}
	return strconv.Itoa(r.Year)
}

// ContentDisposition returns the Content-Disposition header of a download of name: an ASCII file name, with
// "-" for every other character, and the UTF-8 name in filename* (RFC 6266, RFC 5987).
func ContentDisposition(name string) string {
	ascii := strings.Map(func(r rune) rune {
		switch {
		case r == '"' || r == '\\':
			return '-'
		case r < 0x20 || r > 0x7e:
			return '-'
		}
		return r
	}, name)
	var encoded strings.Builder
	for _, c := range []byte(name) {
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.IndexByte("-._~", c) >= 0 {
			encoded.WriteByte(c)
		} else {
			fmt.Fprintf(&encoded, "%%%02X", c)
		}
	}
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, ascii, encoded.String())
}

// exporter writes one view.
type exporter struct {
	in         input
	b          built
	unassigned bool
}
