package allwork

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/date"
	"northstar/internal/platform/money"
	"northstar/internal/platform/sheet"
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
var viewNames = map[string]string{ViewGrid: "Grid", ViewCalendar: "Calendar", ViewBudget: "Budget & cost"}

// showLevelsWords are the Show levels choices as people read them.
var showLevelsWords = map[string]string{
	ShowEverything: "Everything", ShowCompleteBranches: "Complete branches only",
	ShowCampaignsWithoutInitiatives: "Campaigns without initiatives", ShowInitiativesWithoutActivities: "Initiatives without activities",
}

// File is an exported file.
type File struct {
	Name        string // "NorthStar All work – Grid – 2026 – 18 Oct 2026.xlsx"
	ContentType string
	Body        []byte
}

// Export returns the view of q as a spreadsheet (.xlsx) or CSV file for a (FR-032 – FR-034); unassigned says
// whether the unassigned planned cost rows are included, as the screen's switch does (FR-012). It reads in
// one transaction and changes nothing.
func (s *Service) Export(ctx context.Context, a actor.Actor, q Query, format string, unassigned bool) (File, error) {
	var in input
	var b built
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		if in, err = s.input(ctx, a, q); err != nil {
			return err
		}
		b = build(in)
		return nil
	})
	if err != nil {
		return File{}, err
	}
	return exportFile(in, b, a, format, unassigned)
}

// exportFile writes the worked-out view b as a file of format.
func exportFile(in input, b built, a actor.Actor, format string, unassigned bool) (File, error) {
	e := exporter{in: in, b: b, unassigned: unassigned}
	notes := e.notes(a)
	table := e.table()
	file := File{Name: fileName(b.result, in, format)}
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
			{Name: viewNames[b.result.View], Rows: table},
		}}
		if err := sheet.WriteXLSX(&body, wb); err != nil {
			return File{}, err
		}
	}
	file.Body = body.Bytes()
	return file, nil
}

// fileName is "NorthStar All work – <view> – <year> – <DD MMM YYYY>.<format>" (FR-034).
func fileName(r Result, in input, format string) string {
	return fmt.Sprintf("NorthStar All work – %s – %d – %s.%s", viewNames[r.View], r.Year, date.Format(in.today), format)
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

// notes returns the notes before the data (FR-034): the view, the year, one line per active filter (or
// None), the search, Show levels, the display currency, who exported it and when.
func (e exporter) notes(a actor.Actor) [][2]string {
	r := e.b.result
	notes := [][2]string{{"View", viewNames[r.View]}, {"Year", strconv.Itoa(r.Year)}}
	chips := e.chips()
	if len(chips) == 0 {
		chips = []string{"None"}
	}
	for i, chip := range chips {
		label := ""
		if i == 0 {
			label = "Filters"
		}
		notes = append(notes, [2]string{label, chip})
	}
	search := e.b.query.Search
	if search == "" {
		search = "None"
	}
	now := e.in.now.UTC()
	return append(notes,
		[2]string{"Search", search},
		[2]string{"Show levels", showLevelsWords[e.b.query.ShowLevels]},
		[2]string{"Display currency", r.DisplayCurrency},
		[2]string{"Exported by", a.Name},
		[2]string{"Exported at", date.Format(date.Today(now)) + ", " + now.Format("15:04") + " UTC"},
	)
}

// chips returns the active filters as their chips read, "<Filter>: <value>", in the order of the filter bar
// (FR-008).
func (e exporter) chips() []string {
	q := e.b.query
	var chips []string
	add := func(filter string, values []string, label func(string) string) {
		for _, v := range values {
			chips = append(chips, filter+": "+label(v))
		}
	}
	add("Plan", q.Plans, func(id string) string {
		if n, found := e.b.tree.byID[id]; found {
			return n.n.Name
		}
		return id
	})
	add("Plan status", q.PlanStatuses, chipLabel(planStatuses))
	add("Campaign / initiative status", q.ContainerStatuses, func(v string) string {
		if c, found := progressStatuses[v]; found {
			return c.label
		}
		return chipLabel(timingStatuses)(v)
	})
	add("Activity status", q.ActivityStatuses, chipLabel(activityStatuses))
	add("Owner", q.Owners, optionLabel(e.in.choices.People))
	add("Team", q.Teams, optionLabel(e.in.choices.Teams))
	add("Business unit", q.BusinessUnits, optionLabel(e.in.choices.BusinessUnits))
	add("Activity type", q.ActivityTypes, optionLabel(e.in.choices.ActivityTypes))
	switch q.Strategic {
	case StrategicOnly:
		chips = append(chips, "Strategic / AOP plans: Only")
	case StrategicExclude:
		chips = append(chips, "Strategic / AOP plans: Exclude")
	}
	return chips
}

// chipLabel returns the word of a status value of family.
func chipLabel(family map[string]chip) func(string) string {
	return func(v string) string {
		if c, found := family[v]; found {
			return c.label
		}
		return v
	}
}

// optionLabel returns the name of an option ID among options.
func optionLabel(options []Option) func(string) string {
	return func(id string) string {
		for _, o := range options {
			if o.ID == id {
				return o.Label
			}
		}
		return id
	}
}

// table returns the heading row, the rows of the view and the totals row (FR-032, FR-033).
func (e exporter) table() [][]sheet.Cell {
	r := e.b.result
	withContext := false
	rows := make([]Row, 0, len(r.Rows))
	for _, row := range r.Rows {
		if row.Level == LevelUnassigned && !e.unassigned {
			continue
		}
		rows = append(rows, row)
		withContext = withContext || row.Role == RoleContext
	}
	heading := []sheet.Cell{sheet.Bold("Level"), sheet.Bold("Path"), sheet.Bold("Name")}
	if withContext {
		heading = append(heading, sheet.Bold("Shown for context"))
	}
	heading = append(heading, e.headings()...)
	table := [][]sheet.Cell{heading}
	for _, row := range rows {
		line := []sheet.Cell{sheet.Text(levelWords[row.Level]), sheet.Text(e.path(row)), sheet.Text(row.Name)}
		if withContext {
			yes := ""
			if row.Role == RoleContext {
				yes = "Yes"
			}
			line = append(line, sheet.Text(yes))
		}
		table = append(table, append(line, e.values(row)...))
	}
	noun := "activities"
	if r.Totals.Activities == 1 {
		noun = "activity"
	}
	totals := []sheet.Cell{{}, {}, sheet.Bold(fmt.Sprintf("Total · %d %s", r.Totals.Activities, noun))}
	if withContext {
		totals = append(totals, sheet.Cell{})
	}
	return append(table, append(totals, e.totals()...))
}

// headings returns the headings of the view's columns (FR-033): money with the currency, "Planned cost
// (USD)"; a local amount followed by "Currency".
func (e exporter) headings() []sheet.Cell {
	cur := e.b.result.DisplayCurrency
	withCurrency := func(label string) sheet.Cell { return sheet.Bold(label + " (" + cur + ")") }
	var headings []sheet.Cell
	switch e.b.result.View {
	case ViewCalendar:
		headings = []sheet.Cell{sheet.Bold("ID"), sheet.Bold("Status"), sheet.Bold("Start date"), sheet.Bold("End date"),
			sheet.Bold("Owner"), withCurrency("Planned cost")}
	case ViewBudget:
		headings = []sheet.Cell{sheet.Bold("ID"), withCurrency("Plan budget")}
		for _, p := range e.b.result.Periods {
			headings = append(headings, withCurrency(p.Label))
		}
		headings = append(headings, withCurrency("Total"))
	default:
		for _, c := range e.b.result.Columns {
			switch {
			case isLocal(c.Key):
				headings = append(headings, sheet.Bold(c.Label), sheet.Bold("Currency"))
			case c.Kind == KindMoney:
				headings = append(headings, withCurrency(c.Label))
			default:
				headings = append(headings, sheet.Bold(c.Label))
			}
		}
	}
	return headings
}

// values returns the values of a row in the view's columns.
func (e exporter) values(row Row) []sheet.Cell {
	id := sheet.Text(row.RecordID)
	if row.Level == LevelUnassigned {
		id = sheet.Cell{}
	}
	switch e.b.result.View {
	case ViewCalendar:
		end := sheet.Date(row.EndDate)
		if row.Ongoing {
			end = sheet.Text(ongoingWord)
		}
		owner := sheet.Cell{}
		if row.Owner != nil {
			owner = sheet.Text(inactiveLabel(row.Owner.Name, row.Owner.Inactive))
		}
		status := sheet.Cell{}
		if row.Status != nil {
			status = sheet.Text(row.Status.Label)
		}
		return []sheet.Cell{id, status, sheet.Date(row.StartDate), end, owner, moneyValue(row.PlannedCost)}
	case ViewBudget:
		values := []sheet.Cell{id, moneyValue(row.Budget)}
		for _, a := range row.Periods {
			values = append(values, moneyValue(a))
		}
		return append(values, moneyValue(row.Total))
	}
	var values []sheet.Cell
	for i, c := range e.b.result.Columns {
		values = append(values, sheetCell(row.Cells[i]))
		if isLocal(c.Key) {
			currency := sheet.Cell{}
			if cell := row.Cells[i]; cell != nil && cell.Money != nil {
				currency = sheet.Text(cell.Money.Currency)
			}
			values = append(values, currency)
		}
	}
	return values
}

// totals returns the totals row's values in the view's columns (FR-014, FR-024).
func (e exporter) totals() []sheet.Cell {
	t := e.b.result.Totals
	switch e.b.result.View {
	case ViewCalendar:
		return []sheet.Cell{{}, {}, {}, {}, {}, moneyValue(t.PlannedCost)}
	case ViewBudget:
		values := []sheet.Cell{{}, moneyValue(t.PlanBudget)}
		for _, a := range t.Periods {
			values = append(values, moneyValue(a))
		}
		return append(values, moneyValue(t.Total))
	}
	var values []sheet.Cell
	for i, c := range e.b.result.Columns {
		values = append(values, sheetCell(t.Cells[i]))
		if isLocal(c.Key) {
			values = append(values, sheet.Cell{})
		}
	}
	return values
}

// path returns the names of a row's ancestors, top-down: "FY2026 Lubricants Europe › Helix launch" (FR-032).
func (e exporter) path(row Row) string {
	var names []string
	n := e.b.tree.byID[row.RecordID]
	if n != nil && row.Level != LevelUnassigned {
		n = n.parent
	}
	for ; n != nil; n = n.parent {
		names = append([]string{n.n.Name}, names...)
	}
	return strings.Join(names, " › ")
}

// sheetCell returns a grid cell as a cell of the file (FR-033): money and numbers as numbers, dates as dates,
// statuses as words, inactive values labelled; nil and "—" as empty cells.
func sheetCell(c *Cell) sheet.Cell {
	if c == nil {
		return sheet.Cell{}
	}
	switch c.Kind {
	case KindMoney:
		return sheet.Money(c.Money.Amount)
	case KindNumber:
		return sheet.Number(*c.Number)
	case KindPercent:
		n, _ := strconv.ParseInt(c.Percent, 10, 64)
		return sheet.Percent(n)
	case KindDate:
		return sheet.Date(c.Date)
	case KindDateTime:
		return sheet.DateTime(*c.DateTime)
	case KindText, KindStatus:
		return sheet.Text(inactiveLabel(c.Text, c.Inactive))
	}
	return sheet.Cell{}
}

// moneyValue returns an optional amount as a cell.
func moneyValue(m *money.Money) sheet.Cell {
	if m == nil {
		return sheet.Cell{}
	}
	return sheet.Money(m.Amount)
}

// inactiveLabel adds "(inactive)" to the text of an inactive value.
func inactiveLabel(text string, inactive bool) string {
	if inactive {
		return text + " (inactive)"
	}
	return text
}

// isLocal reports whether the column holds amounts in a line item's own currency.
func isLocal(key string) bool {
	return slices.Contains(localColumns, key)
}
