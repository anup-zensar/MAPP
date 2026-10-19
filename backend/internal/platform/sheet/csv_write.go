package sheet

import (
	"encoding/csv"
	"io"
	"strings"
)

// byteOrderMark starts a CSV file so that Excel reads it as UTF-8 (D-120).
const byteOrderMark = "\xEF\xBB\xBF"

// WriteCSV writes a CSV file to w (D-120): the UTF-8 byte-order mark, the notes as "label,value" lines, an
// empty line, then the rows. Numbers and money are written without thousands separators ("12500.00"), dates
// as YYYY-MM-DD and moments as YYYY-MM-DD hh:mm (UTC). A text that a spreadsheet would read as a formula —
// starting with =, +, -, @, a tab or a carriage return — is written with a leading apostrophe, so opening the
// file never runs anything.
func WriteCSV(w io.Writer, notes [][2]string, rows [][]Cell) error {
	if _, err := io.WriteString(w, byteOrderMark); err != nil {
		return err
	}
	c := csv.NewWriter(w)
	for _, note := range notes {
		if err := c.Write([]string{safeText(note[0]), safeText(note[1])}); err != nil {
			return err
		}
	}
	if len(notes) > 0 {
		if err := c.Write([]string{""}); err != nil {
			return err
		}
	}
	for _, row := range rows {
		record := make([]string, len(row))
		for i, cell := range row {
			record[i] = csvText(cell)
		}
		if err := c.Write(record); err != nil {
			return err
		}
	}
	c.Flush()
	return c.Error()
}

// csvText returns a cell as CSV text.
func csvText(c Cell) string {
	switch c.kind {
	case kindText:
		return safeText(c.text)
	case kindNumber, kindMoney:
		return c.text
	case kindDate:
		return c.date.String()
	case kindDateTime:
		return c.moment.Format("2006-01-02 15:04")
	}
	return ""
}

// safeText returns a text that a spreadsheet would read as a formula with a leading apostrophe.
func safeText(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}
