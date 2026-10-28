package deck

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func sampleDeck(rows int) Deck {
	table := &Table{Columns: []Column{{Title: "Campaign", Weight: 3}, {Title: "Planned cost", Weight: 2, Align: Right}}}
	for i := range rows {
		table.Rows = append(table.Rows, Row{Cells: []string{fmt.Sprintf("Campaign %d", i+1), "1,000.00 USD"}})
	}
	return Deck{
		Title: "FY2026 R&D (Q1) – 中国", Author: "SP&A One", Footer: "SP&A One — demo data",
		Created: time.Date(2026, 10, 28, 9, 30, 0, 0, time.UTC),
		Pages: []Page{
			{Title: "FY2026 R&D (Q1) – 中国", Subtitle: "Plan · 2026", Cards: []Card{{Label: "Budget", Value: "1,200,000.00 USD"},
				{Label: "Planned cost", Value: "1,300,000.00 USD", Note: "Over budget", Problem: true}}},
			{Title: "Campaigns", Cards: []Card{{Label: "Campaigns", Value: "40"}}, Table: table, Notes: []string{"GSAP data as of 27 Oct 2026"},
				Chart: &Chart{Categories: []string{"Helix", "Rimula"}, MaxLabel: "1,500.00 USD",
					Series: []Series{{Name: "Budget", Colour: ChartBudget, Values: []int64{100000, 150000}},
						{Name: "Actual", Colour: ChartActual, Values: []int64{-5, 50000}}}}},
		},
	}
}

func TestLayout_TableContinuesWithItsHeader_036_FR023(t *testing.T) {
	t.Parallel()
	sheets := sampleDeck(40).sheets()

	if len(sheets) != 4 {
		t.Fatalf("sheets = %d, want 4: the title page, then 14 + 19 + 7 rows", len(sheets))
	}
	if body := tableRows(t, sheets[1:]); body != 40 {
		t.Errorf("rows drawn = %d, want 40", body)
	}
	if got := sheets[2].texts[0].Value; got != "Campaigns (continued)" {
		t.Errorf("continuation title = %q", got)
	}
	last := sheets[3].texts[len(sheets[3].texts)-1]
	if last.Value != "Page 4 of 4" || last.Align != Right {
		t.Errorf("footer = %+v, want Page 4 of 4 on the right", last)
	}
}

func tableRows(t *testing.T, sheets []sheet) int {
	t.Helper()
	body := 0
	for i, s := range sheets {
		if len(s.grids) != 1 {
			t.Fatalf("sheet %d has %d tables, want 1", i+2, len(s.grids))
		}
		g := s.grids[0]
		if g.Rows[0][0].Text != "Campaign" || !g.Rows[0][0].Bold {
			t.Errorf("sheet %d header = %+v, want the bold column titles", i+2, g.Rows[0][0])
		}
		body += len(g.Rows) - 1
	}
	return body
}

func TestLayout_ChartBesideTableAndBarsToScale(t *testing.T) {
	t.Parallel()
	page := sampleDeck(3).sheets()[1]

	table := page.grids[0]
	if table.X <= margin {
		t.Errorf("table at x %d, want beside the chart", table.X)
	}
	var bars []box
	for _, b := range page.boxes {
		if b.Fill == ChartBudget || b.Fill == ChartActual {
			bars = append(bars, b)
		}
	}
	// The legend's two swatches, Helix's budget, Rimula's budget and actual; Helix's negative actual has no bar.
	if len(bars) != 5 {
		t.Fatalf("coloured boxes = %d, want 5", len(bars))
	}
	if tallest := bars[3]; bars[2].H*3 != tallest.H*2 {
		t.Errorf("bar heights %d and %d, want 2:3", bars[2].H, tallest.H)
	}
}

func TestFit_ShortensWithEllipsis(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, in, want string
		width          int
	}{
		{name: "fits", in: "Helix launch", width: 200, want: "Helix launch"},
		{name: "too long", in: "Helix launch campaign for Europe", width: 60, want: "Helix launch…"},
		{name: "control characters", in: "Line\tone\nlater", width: 200, want: "Line one later"},
		{name: "nothing fits", in: "Helix", width: 2, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := fit(tt.in, tt.width, 9, false)
			if got != tt.want || textWidth(got, 9, false) > tt.width {
				t.Errorf("fit(%q, %d) = %q, want %q", tt.in, tt.width, got, tt.want)
			}
		})
	}
}

func TestScaled_LargeAmountsDoNotOverflow(t *testing.T) {
	t.Parallel()
	if got := scaled(9_000_000_000_000_000, 9_000_000_000_000_000, 400); got != 400 {
		t.Errorf("scaled = %d, want 400", got)
	}
	if got := scaled(1, 3, 300); got != 100 {
		t.Errorf("scaled(1, 3, 300) = %d, want 100", got)
	}
}

func TestWinANSI_MapsWindows1252AndReplacesTheRest(t *testing.T) {
	t.Parallel()
	got := winANSI("€ – é 中")
	want := []byte{0x80, ' ', 0x96, ' ', 0xE9, ' ', '?'}
	if string(got) != string(want) {
		t.Errorf("winANSI = % X, want % X", got, want)
	}
	if escaped := string(escapePDF([]byte(`a(b)\`))); escaped != `a\(b\)\\` {
		t.Errorf("escapePDF = %s", escaped)
	}
	if !strings.HasPrefix(unicodeString("é"), "<FEFF00E9") {
		t.Errorf("unicodeString = %s", unicodeString("é"))
	}
}
