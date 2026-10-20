package financials

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"northstar/internal/platform/date"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
)

// CSVFeed is the simulated GSAP extract (FR-004, D-247): a UTF-8 .csv file with a header row naming the
// columns of csvColumns, in any order, uploaded through the API. It is read in memory only and never
// logged: it holds vendor names and business codes.
type CSVFeed struct{}

// The limits of an extract: at most 700 KB decoded, so that its base64 text fits in the API's 1 MB
// request body (001 R10, as 002's option import, D-051), and at most 5,000 data rows.
const (
	maxFileBytes = 700 * 1024
	maxFileRows  = 5000
)

// The columns of the simulated GSAP extract, in the order of the sample file (plan, "Simulated GSAP
// extract").
const (
	colType           = "type"
	colDocumentNumber = "document_number"
	colLine           = "line"
	colPONumber       = "po_number"
	colPOLine         = "po_line"
	colPostingDate    = "posting_date"
	colAccrualMonth   = "accrual_month"
	colAmount         = "amount"
	colCurrency       = "currency"
	colVendorCode     = "vendor_code"
	colVendorName     = "vendor_name"
	colDescription    = "description"
	colSwbsCode       = "swbs_code"
	colTrackingKey    = "tracking_key"
	colLineItemID     = "line_item_id"
)

// csvColumns are every column the feed reads, in the sample file's order.
var csvColumns = []string{
	colType, colDocumentNumber, colLine, colPONumber, colPOLine, colPostingDate, colAccrualMonth, colAmount,
	colCurrency, colVendorCode, colVendorName, colDescription, colSwbsCode, colTrackingKey, colLineItemID,
}

// requiredColumns must be named in the header row.
var requiredColumns = []string{colType, colDocumentNumber, colPostingDate, colAmount, colCurrency}

// The forms of the extract's values.
var (
	documentForm     = regexp.MustCompile(`^[A-Za-z0-9/-]{1,20}$`)
	lineForm         = regexp.MustCompile(`^[0-9]{1,6}$`)
	monthForm        = regexp.MustCompile(`^[0-9]{4}-(0[1-9]|1[0-2])$`)
	amountForm       = regexp.MustCompile(`^-?[0-9]{1,15}(\.[0-9]{1,2})?$`)
	trackingKeyForm  = regexp.MustCompile(`^HX[0-9]{1,9}$`)
	lineItemIDForm   = regexp.MustCompile(`^L-[0-9]{1,9}$`)
	byteOrderMarkUTF = []byte("\xEF\xBB\xBF") // the byte-order mark Excel writes in "CSV UTF-8" files
)

// typeNames are the accepted spellings of the type column, upper case.
var typeNames = map[string]Type{
	"PO": TypePurchaseOrder, "PURCHASE ORDER": TypePurchaseOrder, "INVOICE": TypeInvoice, "PAYMENT": TypePayment,
	"ACCRUAL": TypeAccrual,
}

// Name is the feed's name on the screens.
func (CSVFeed) Name() string { return csvFeedName }

// Source says that the extract is a file.
func (CSVFeed) Source() Source { return SourceFile }

// Read parses an extract (FR-004, FR-005). A file it cannot use at all is refused as a whole with one
// message: a name not ending in .csv (fileName), or on content a file over 700 KB, not UTF-8, unreadable as
// CSV, without the required columns, without data rows or with more than 5,000. Otherwise every row is
// checked as a whole and every problem reported with its row and column; rows with a problem are left out.
// Empty lines are skipped; data rows keep the file's line numbers, so the first is row 2.
func (CSVFeed) Read(_ context.Context, d Delivery) ([]Incoming, []RowProblem, []string, error) {
	if !strings.HasSuffix(lower(strings.TrimSpace(d.FileName)), ".csv") {
		return nil, nil, nil, fieldProblem("fileName", msgFileName)
	}
	if len(d.Content) > maxFileBytes {
		return nil, nil, nil, contentProblem(msgFileTooLarge)
	}
	content := bytes.TrimPrefix(d.Content, byteOrderMarkUTF)
	if !utf8.Valid(content) {
		return nil, nil, nil, contentProblem(msgFileNotUTF8)
	}
	records, err := readRecords(content)
	if err != nil {
		return nil, nil, nil, contentProblem(msgFileUnreadable)
	}
	if len(records) == 0 {
		return nil, nil, nil, contentProblem(msgFileNoRows)
	}
	header, unused := headerOf(records[0].cells)
	var missing []string
	for _, name := range requiredColumns {
		if _, found := header[name]; !found {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return nil, nil, nil, contentProblem(fmt.Sprintf(msgFileColumns, strings.Join(missing, ", ")))
	}
	data := records[1:]
	switch {
	case len(data) == 0:
		return nil, nil, nil, contentProblem(msgFileNoRows)
	case len(data) > maxFileRows:
		return nil, nil, nil, contentProblem(msgFileTooMany)
	}
	rows, problems := []Incoming{}, []RowProblem{}
	for _, r := range data {
		row, rowProblems := parseRow(r, header)
		if len(rowProblems) > 0 {
			problems = append(problems, rowProblems...)
			continue
		}
		rows = append(rows, row)
	}
	return rows, problems, unused, nil
}

// csvRecord is one non-empty record of the file with the line it starts on.
type csvRecord struct {
	line  int
	cells []string
}

// readRecords reads every record of the file that has at least one non-blank cell.
func readRecords(content []byte) ([]csvRecord, error) {
	reader := csv.NewReader(bytes.NewReader(content))
	reader.FieldsPerRecord = -1 // rows may be shorter or longer than the header
	reader.LazyQuotes = true    // a stray quote inside a vendor name is part of it
	var records []csvRecord
	for {
		cells, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return records, nil
		}
		if err != nil {
			return nil, err
		}
		if slices.IndexFunc(cells, func(c string) bool { return strings.TrimSpace(c) != "" }) < 0 {
			continue // a line of empty cells is an empty line
		}
		line, _ := reader.FieldPos(0)
		records = append(records, csvRecord{line: line, cells: cells})
	}
}

// headerOf returns the column index of every column the feed reads, matched ignoring letter case and
// surrounding spaces, and the names of the other columns as the file writes them. A column named twice is
// read from its first place.
func headerOf(cells []string) (map[string]int, []string) {
	header := map[string]int{}
	unused := []string{}
	for i, cell := range cells {
		name := lower(strings.TrimSpace(cell))
		if _, seen := header[name]; slices.Contains(csvColumns, name) && !seen {
			header[name] = i
			continue
		}
		if trimmed := strings.TrimSpace(cell); trimmed != "" {
			unused = append(unused, trimmed)
		}
	}
	return header, unused
}

// rowReader reads the cells of one record by column name and collects the row's problems.
type rowReader struct {
	r        csvRecord
	header   map[string]int
	problems []RowProblem
}

// cell returns the trimmed value of the column name, "" when the file has no such column or the record is
// shorter.
func (rr *rowReader) cell(name string) string {
	i, found := rr.header[name]
	if !found || i >= len(rr.r.cells) {
		return ""
	}
	return strings.TrimSpace(rr.r.cells[i])
}

// problem records a problem of the column name.
func (rr *rowReader) problem(name, message string) {
	rr.problems = append(rr.problems, RowProblem{Row: rr.r.line, Column: name, Message: message})
}

// longerThan reports whether text has more than n characters.
func longerThan(text string, n int) bool {
	return utf8.RuneCountInString(text) > n
}

// parseRow checks one data row against the column rules of the extract and returns it, or every problem
// it has.
func parseRow(r csvRecord, header map[string]int) (Incoming, []RowProblem) {
	rr := &rowReader{r: r, header: header}
	in := Incoming{Row: r.line}

	typeText := strings.ToUpper(rr.cell(colType))
	typ, knownType := typeNames[typeText]
	if !knownType {
		rr.problem(colType, msgType)
	}
	in.Type = typ

	in.DocumentNumber = rr.cell(colDocumentNumber)
	switch {
	case in.DocumentNumber == "":
		rr.problem(colDocumentNumber, msgDocumentEmpty)
	case !documentForm.MatchString(in.DocumentNumber):
		rr.problem(colDocumentNumber, msgDocumentForm)
	}
	if in.LineNumber = rr.cell(colLine); in.LineNumber != "" && !lineForm.MatchString(in.LineNumber) {
		rr.problem(colLine, msgLine)
	}
	if typ == TypePurchaseOrder {
		in.PONumber = in.DocumentNumber // a PO line refers to its own PO; po_number and po_line are ignored
	} else if knownType {
		if in.PONumber = rr.cell(colPONumber); in.PONumber != "" && !documentForm.MatchString(in.PONumber) {
			rr.problem(colPONumber, msgPONumber)
		}
		if in.PONumber != "" {
			if in.POLineNumber = rr.cell(colPOLine); in.POLineNumber != "" && !lineForm.MatchString(in.POLineNumber) {
				rr.problem(colPOLine, msgPOLine)
			}
		}
	}
	posting, err := date.Parse(rr.cell(colPostingDate))
	if err != nil {
		rr.problem(colPostingDate, msgPostingDate)
	}
	in.PostingDate = posting
	if typ == TypeAccrual {
		if in.AccrualMonth = rr.cell(colAccrualMonth); !monthForm.MatchString(in.AccrualMonth) {
			rr.problem(colAccrualMonth, msgAccrualMonth)
		}
	}
	amountText := rr.cell(colAmount)
	amount, err := money.ParseAmount(amountText)
	if !amountForm.MatchString(amountText) || err != nil {
		rr.problem(colAmount, msgAmount)
	}
	in.Amount = amount
	if in.Currency = strings.ToUpper(rr.cell(colCurrency)); in.Currency == "" {
		rr.problem(colCurrency, msgCurrencyEmpty)
	}

	if in.VendorCode = rr.cell(colVendorCode); longerThan(in.VendorCode, 20) {
		rr.problem(colVendorCode, msgVendorCode)
	}
	if in.VendorName = rr.cell(colVendorName); longerThan(in.VendorName, 120) {
		rr.problem(colVendorName, msgVendorName)
	}
	if in.Description = rr.cell(colDescription); longerThan(in.Description, 200) {
		rr.problem(colDescription, msgDescription)
	}
	in.SwbsCode = rr.cell(colSwbsCode)
	if longerThan(in.SwbsCode, 40) || strings.IndexFunc(in.SwbsCode, unicode.IsSpace) >= 0 {
		rr.problem(colSwbsCode, msgSwbsCode)
	}
	if in.TrackingKey = strings.ToUpper(rr.cell(colTrackingKey)); in.TrackingKey != "" && !trackingKeyForm.MatchString(in.TrackingKey) {
		rr.problem(colTrackingKey, msgTrackingKey)
	}
	if in.SentLineItemID = strings.ToUpper(rr.cell(colLineItemID)); in.SentLineItemID != "" && !lineItemIDForm.MatchString(in.SentLineItemID) {
		rr.problem(colLineItemID, msgLineItemID)
	}
	return in, rr.problems
}

// contentProblem refuses a whole file with one message on content (FR-004).
func contentProblem(message string) error {
	return fieldProblem("content", message)
}

// fieldProblem is the validation error of one field.
func fieldProblem(field, message string) error {
	return errs.Validation(errs.FieldError{Field: field, Message: message})
}
