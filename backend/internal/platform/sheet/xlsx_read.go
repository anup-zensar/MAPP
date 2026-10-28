package sheet

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strings"
)

type workbookPart struct {
	Pr struct {
		Date1904 string `xml:"date1904,attr"`
	} `xml:"workbookPr"`
	Sheets []struct {
		Name string `xml:"name,attr"`
		RID  string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
	} `xml:"sheets>sheet"`
}

type relationshipsPart struct {
	List []struct {
		ID     string `xml:"Id,attr"`
		Target string `xml:"Target,attr"`
	} `xml:"Relationship"`
}

type richText struct {
	T    string `xml:"t"`
	Runs []struct {
		T string `xml:"t"`
	} `xml:"r"`
}

// xlsxFile is an opened workbook: its parts by name and what every sheet needs to read its cells.
type xlsxFile struct {
	parts    map[string]*zip.File
	strings  []string
	dates    dateStyles
	date1904 bool
}

// ReadXLSX reads every sheet of an .xlsx file, in the workbook's order, as text: numbers as written,
// date cells as YYYY-MM-DD, booleans as TRUE or FALSE, formulas by their saved result. A sheet with more
// than maxRows rows holding text is refused with ErrTooManyRows; anything that is not a workbook with
// ErrUnreadable.
func ReadXLSX(data []byte, maxRows int) ([]Table, error) {
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, ErrUnreadable
	}
	f := &xlsxFile{parts: map[string]*zip.File{}}
	for _, part := range z.File {
		f.parts[strings.TrimPrefix(part.Name, "/")] = part
	}
	var wb workbookPart
	if err = f.decode("xl/workbook.xml", &wb, true); err != nil {
		return nil, err
	}
	f.date1904 = wb.Pr.Date1904 == "1" || strings.EqualFold(wb.Pr.Date1904, "true")
	if err = f.readShared(); err != nil {
		return nil, err
	}
	targets, err := f.sheetTargets()
	if err != nil {
		return nil, err
	}
	tables := make([]Table, 0, len(wb.Sheets))
	for _, s := range wb.Sheets {
		rows, err := f.readSheet(targets[s.RID], maxRows)
		if err != nil {
			return nil, err
		}
		tables = append(tables, Table{Name: s.Name, Rows: rows})
	}
	return tables, nil
}

func (f *xlsxFile) readShared() error {
	var styles stylesPart
	if err := f.decode("xl/styles.xml", &styles, false); err != nil {
		return err
	}
	f.dates = styles.dateStyles()
	var table struct {
		Items []richText `xml:"si"`
	}
	if err := f.decode("xl/sharedStrings.xml", &table, false); err != nil {
		return err
	}
	f.strings = make([]string, len(table.Items))
	for i, item := range table.Items {
		f.strings[i] = item.String()
	}
	return nil
}

// sheetTargets returns the part name of every sheet relationship, by its id.
func (f *xlsxFile) sheetTargets() (map[string]string, error) {
	var rels relationshipsPart
	if err := f.decode("xl/_rels/workbook.xml.rels", &rels, true); err != nil {
		return nil, err
	}
	targets := map[string]string{}
	for _, r := range rels.List {
		if strings.HasPrefix(r.Target, "/") {
			targets[r.ID] = strings.TrimPrefix(r.Target, "/")
		} else {
			targets[r.ID] = path.Join("xl", r.Target)
		}
	}
	return targets, nil
}

// decode reads the XML part name into v, at most maxPartBytes of it; a missing part is ErrUnreadable only
// when it is required.
func (f *xlsxFile) decode(name string, v any, required bool) error {
	part, found := f.parts[name]
	switch {
	case !found && required:
		return ErrUnreadable
	case !found:
		return nil
	case part.UncompressedSize64 > maxPartBytes:
		return ErrUnreadable
	}
	rc, err := part.Open()
	if err != nil {
		return ErrUnreadable
	}
	defer func() { _ = rc.Close() }()
	if err := xml.NewDecoder(io.LimitReader(rc, maxPartBytes)).Decode(v); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrUnreadable, name, err)
	}
	return nil
}

func (r richText) String() string {
	var b strings.Builder
	b.WriteString(r.T)
	for _, run := range r.Runs {
		b.WriteString(run.T)
	}
	return b.String()
}
