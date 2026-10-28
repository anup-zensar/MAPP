package deck

import "strconv"

// Every position is in points from the top left of a 16:9 page, PowerPoint's widescreen size.
const (
	pageWidth     = 960
	pageHeight    = 540
	margin        = 40
	contentWidth  = pageWidth - 2*margin
	titleTop      = 26
	titleHeight   = 34
	titleSize     = 24
	subtitleTop   = 62
	textHeight    = 18
	subtitleSize  = 12
	contentTop    = 96
	cardHeight    = 64
	cardGap       = 12
	cardPad       = 12
	valueSize     = 18
	maxCards      = 6
	maxCardsInRow = 4
	rowHeight     = 20
	tableSize     = 9
	cellPad       = 6
	footerTop     = 512
	footerSize    = 8
	noteHeight    = 14
	noteSize      = 9
	sectionGap    = 16
	chartShare    = 55
	stackedShare  = 45
	maxBeside     = 4
	continued     = " (continued)"
)

type box struct {
	X, Y, W, H int
	Fill, Line Colour
}

type text struct {
	X, Y, W, H, Size int
	Bold             bool
	Colour           Colour
	Align            Align
	Value            string
}

type cell struct {
	Text  string
	Align Align
	Bold  bool
	Fill  Colour
}

type grid struct {
	X, Y   int
	Widths []int
	Rows   [][]cell
}

type sheet struct {
	boxes []box
	grids []grid
	texts []text
}

type area struct {
	x, y, w, h int
}

func (d Deck) sheets() []sheet {
	var all []sheet
	for _, p := range d.Pages {
		all = append(all, layoutPage(p)...)
	}
	for i := range all {
		all[i].footer(d.Footer, i+1, len(all))
	}
	return all
}

func layoutPage(p Page) []sheet {
	first := header(p.Title, p.Subtitle)
	top := contentTop
	if len(p.Cards) > 0 {
		top += first.cards(p.Cards, top) + sectionGap
	}
	bottom := footerTop - sectionGap - len(p.Notes)*noteHeight
	first.notes(p.Notes, bottom+sectionGap/2)
	free := area{x: margin, y: top, w: contentWidth, h: bottom - top}
	if p.Chart != nil {
		var chartArea area
		chartArea, free = splitForChart(free, p.Table)
		first.chart(*p.Chart, chartArea)
	}
	if p.Table == nil {
		return []sheet{first}
	}
	return tablePages(first, p, free)
}

// splitForChart gives a chart the whole free area without a table, the left of it beside a narrow table, and the
// top of it above a wide one; it returns the chart's area and what is left for the table.
func splitForChart(free area, table *Table) (chart, rest area) {
	switch {
	case table == nil:
		return free, area{}
	case len(table.Columns) <= maxBeside:
		chart, rest = free, free
		chart.w = contentWidth * chartShare / 100
		rest.x, rest.w = margin+chart.w+sectionGap, contentWidth-chart.w-sectionGap
		return chart, rest
	}
	chart, rest = free, free
	chart.h = free.h * stackedShare / 100
	rest.y, rest.h = free.y+chart.h+sectionGap, free.h-chart.h-sectionGap
	return chart, rest
}

// tablePages draws the table's rows on first, then on as many continuation pages as they need.
func tablePages(first sheet, p Page, free area) []sheet {
	rows := p.Table.Rows
	pages := []sheet{}
	current := first
	for {
		n := min(max(free.h/rowHeight-1, 1), len(rows))
		current.grid(p.Table.Columns, rows[:n], free)
		pages = append(pages, current)
		rows = rows[n:]
		if len(rows) == 0 {
			return pages
		}
		current = header(p.Title+continued, p.Subtitle)
		free = area{x: margin, y: contentTop, w: contentWidth, h: footerTop - sectionGap - contentTop}
	}
}

func header(title, subtitle string) sheet {
	return sheet{texts: []text{
		{X: margin, Y: titleTop, W: contentWidth, H: titleHeight, Size: titleSize, Bold: true, Colour: Ink,
			Value: fit(title, contentWidth, titleSize, true)},
		{X: margin, Y: subtitleTop, W: contentWidth, H: textHeight, Size: subtitleSize, Colour: Subtle,
			Value: fit(subtitle, contentWidth, subtitleSize, false)},
	}}
}

func (s *sheet) footer(label string, page, pages int) {
	number := "Page " + strconv.Itoa(page) + " of " + strconv.Itoa(pages)
	s.texts = append(s.texts,
		text{X: margin, Y: footerTop, W: contentWidth / 2, H: noteHeight, Size: footerSize, Colour: Subtle, Value: label},
		text{X: margin + contentWidth/2, Y: footerTop, W: contentWidth / 2, H: noteHeight, Size: footerSize,
			Colour: Subtle, Align: Right, Value: number})
}

func (s *sheet) notes(lines []string, top int) {
	for i, line := range lines {
		s.texts = append(s.texts, text{X: margin, Y: top + i*noteHeight, W: contentWidth, H: noteHeight, Size: noteSize,
			Colour: Subtle, Value: fit(line, contentWidth, noteSize, false)})
	}
}

// cards draws up to six cards, in two rows when there are more than four, and returns the height they take.
func (s *sheet) cards(cards []Card, top int) int {
	n := min(len(cards), maxCards)
	perRow := n
	if n > maxCardsInRow {
		perRow = (n + 1) / 2
	}
	width := (contentWidth - (perRow-1)*cardGap) / perRow
	inner := width - 2*cardPad
	for i, c := range cards[:n] {
		x, y := margin+(i%perRow)*(width+cardGap), top+(i/perRow)*(cardHeight+cardGap)
		valueColour := Ink
		if c.Problem {
			valueColour = Danger
		}
		s.boxes = append(s.boxes, box{X: x, Y: y, W: width, H: cardHeight, Fill: White, Line: Hairline})
		s.texts = append(s.texts,
			text{X: x + cardPad, Y: y + 6, W: inner, H: 14, Size: noteSize, Colour: Subtle, Value: fit(c.Label, inner, noteSize, false)},
			text{X: x + cardPad, Y: y + 21, W: inner, H: 24, Size: valueSize, Bold: true, Colour: valueColour,
				Value: fit(c.Value, inner, valueSize, true)},
			text{X: x + cardPad, Y: y + 46, W: inner, H: 12, Size: footerSize, Colour: Subtle, Value: fit(c.Note, inner, footerSize, false)})
	}
	rows := (n + perRow - 1) / perRow
	return rows*cardHeight + (rows-1)*cardGap
}

func (s *sheet) grid(columns []Column, rows []Row, at area) {
	widths := columnWidths(columns, at.w)
	g := grid{X: at.x, Y: at.y, Widths: widths, Rows: [][]cell{headerCells(columns, widths)}}
	for _, r := range rows {
		g.Rows = append(g.Rows, rowCells(columns, widths, r))
	}
	s.grids = append(s.grids, g)
}

func headerCells(columns []Column, widths []int) []cell {
	cells := make([]cell, len(columns))
	for i, c := range columns {
		cells[i] = cell{Text: fit(c.Title, widths[i]-2*cellPad, tableSize, true), Align: c.Align, Bold: true, Fill: Band}
	}
	return cells
}

func rowCells(columns []Column, widths []int, r Row) []cell {
	cells := make([]cell, len(columns))
	for i, c := range columns {
		value := ""
		if i < len(r.Cells) {
			value = r.Cells[i]
		}
		cells[i] = cell{Text: fit(value, widths[i]-2*cellPad, tableSize, r.Bold), Align: c.Align, Bold: r.Bold, Fill: White}
	}
	return cells
}

// columnWidths shares width out by the columns' weights (0 counts as 1); the last column takes what rounding leaves.
func columnWidths(columns []Column, width int) []int {
	total := 0
	for _, c := range columns {
		total += max(c.Weight, 1)
	}
	widths := make([]int, len(columns))
	used := 0
	for i, c := range columns {
		widths[i] = width * max(c.Weight, 1) / total
		used += widths[i]
	}
	if len(widths) > 0 {
		widths[len(widths)-1] += width - used
	}
	return widths
}

func (g grid) height() int {
	return len(g.Rows) * rowHeight
}
