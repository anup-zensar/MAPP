package financials

import (
	"errors"
	"strings"
	"testing"

	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
)

func readCSV(t *testing.T, name string, lines ...string) ([]Incoming, []RowProblem, []string, error) {
	t.Helper()
	return CSVFeed{}.Read(t.Context(), Delivery{FileName: name, Content: []byte(strings.Join(lines, "\n") + "\n")})
}

// validRow is a valid PO row, in the header's column order.
const validRow = "PO,4500001001,10,,,2026-01-12,,15800.00,GBP,V-100101,Northlight Events GmbH,Venue hire,DE03-M26-0003,HX1003,"

func withCell(column, value string) string {
	cells := strings.Split(validRow, ",")
	for i, name := range csvColumns {
		if name == column {
			cells[i] = value
		}
	}
	return strings.Join(cells, ",")
}

func TestCSVFeed_ReadsAValidRow_FR004(t *testing.T) {
	t.Parallel()
	rows, problems, unused, err := readCSV(t, "gsap.csv", header, validRow,
		"invoice,5100002001,,4500001001,10,2026-02-20,,-150.50,usd,,,,,hx1003,l-1003",
		"Accrual,9100004001,,,,2026-09-30,2026-09,2500,USD,,,,,,")
	if err != nil || len(problems) != 0 || len(unused) != 0 {
		t.Fatalf("Read = %v, problems %+v, unused %v", err, problems, unused)
	}
	want := Fields{
		Type: TypePurchaseOrder, DocumentNumber: "4500001001", LineNumber: "10", PONumber: "4500001001",
		PostingDate: date.MustParse("2026-01-12"), Amount: amountOf(t, "15800.00"), Currency: "GBP", VendorCode: "V-100101",
		VendorName: "Northlight Events GmbH", Description: "Venue hire", SwbsCode: "DE03-M26-0003", TrackingKey: "HX1003",
	}
	if len(rows) != 3 || rows[0].Fields != want || rows[0].Row != 2 {
		t.Fatalf("rows = %+v", rows)
	}
	invoice := rows[1]
	if invoice.Type != TypeInvoice || invoice.PONumber != "4500001001" || invoice.POLineNumber != "10" ||
		invoice.Amount != amountOf(t, "-150.50") || invoice.Currency != "USD" || invoice.TrackingKey != "HX1003" ||
		invoice.SentLineItemID != "L-1003" || invoice.AccrualMonth != "" {
		t.Errorf("invoice = %+v", invoice)
	}
	if accrual := rows[2]; accrual.Type != TypeAccrual || accrual.AccrualMonth != "2026-09" || accrual.Row != 4 {
		t.Errorf("accrual = %+v", accrual)
	}
}

func TestCSVFeed_ColumnRules_FR005(t *testing.T) {
	t.Parallel()
	cases := []struct {
		column, value, message string
	}{
		{colType, "REFUND", "Type must be PO, INVOICE, PAYMENT or ACCRUAL."},
		{colType, "", "Type must be PO, INVOICE, PAYMENT or ACCRUAL."},
		{colDocumentNumber, "", "Document number is missing."},
		{colDocumentNumber, "45 00", `Use 20 letters, digits, "-" or "/" at most.`},
		{colDocumentNumber, strings.Repeat("9", 21), `Use 20 letters, digits, "-" or "/" at most.`},
		{colLine, "1234567", "Line must be a number of up to 6 digits."},
		{colLine, "1a", "Line must be a number of up to 6 digits."},
		{colPostingDate, "", "Posting date must be a date like 2026-03-31."},
		{colPostingDate, "2026-02-30", "Posting date must be a date like 2026-03-31."},
		{colPostingDate, "31/03/2026", "Posting date must be a date like 2026-03-31."},
		{colAmount, "", "Amount must be a number like 1234.50 (a minus sign for credits)."},
		{colAmount, `"1,234.50"`, "Amount must be a number like 1234.50 (a minus sign for credits)."},
		{colAmount, "12.345", "Amount must be a number like 1234.50 (a minus sign for credits)."},
		{colAmount, "1234567890123456", "Amount must be a number like 1234.50 (a minus sign for credits)."},
		{colCurrency, "", "Currency is missing."},
		{colVendorCode, strings.Repeat("v", 21), "Use 20 characters or fewer."},
		{colVendorName, strings.Repeat("n", 121), "Use 120 characters or fewer."},
		{colDescription, strings.Repeat("d", 201), "Use 200 characters or fewer."},
		{colSwbsCode, "GB01 M26", "SWBS code: use 40 characters or fewer, without spaces."},
		{colSwbsCode, strings.Repeat("S", 41), "SWBS code: use 40 characters or fewer, without spaces."},
		{colTrackingKey, "HX-1003", "Tracking key must look like HX1003."},
		{colTrackingKey, "1003", "Tracking key must look like HX1003."},
		{colLineItemID, "1003", "Line item ID must look like L-1003."},
	}
	for _, c := range cases {
		t.Run(c.column+"="+c.value, func(t *testing.T) {
			rows, problems, _, err := readCSV(t, "gsap.csv", header, validRow, withCell(c.column, c.value))
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			want := RowProblem{Row: 3, Column: c.column, Message: c.message}
			if len(rows) != 1 || len(problems) != 1 || problems[0] != want {
				t.Errorf("rows %d, problems %+v, want %+v", len(rows), problems, want)
			}
		})
	}
}

// The accrual month is needed only on accruals; PO number and line are read only on other types.
func TestCSVFeed_TypeDependentColumns_FR005(t *testing.T) {
	t.Parallel()
	invoice := "INVOICE,5100,,45 00,10,2026-02-20,,10.00,USD,,,,,,"
	_, problems, _, _ := readCSV(t, "gsap.csv", header, invoice)
	if len(problems) != 1 || problems[0].Column != colPONumber || problems[0].Message != `PO number: use 20 letters, digits, "-" or "/" at most.` {
		t.Errorf("invoice problems = %+v", problems)
	}
	poLine := "INVOICE,5100,,4500,1234567,2026-02-20,,10.00,USD,,,,,,"
	_, problems, _, _ = readCSV(t, "gsap.csv", header, poLine)
	if len(problems) != 1 || problems[0].Message != "PO line must be a number of up to 6 digits." {
		t.Errorf("PO line problems = %+v", problems)
	}
	poIgnores := "PO,4500,10,45 00,1234567,2026-02-20,bad,10.00,USD,,,,,,"
	rows, problems, _, _ := readCSV(t, "gsap.csv", header, poIgnores)
	if len(problems) != 0 || rows[0].PONumber != "4500" || rows[0].POLineNumber != "" || rows[0].AccrualMonth != "" {
		t.Errorf("PO row = %+v, problems %+v", rows, problems)
	}
	noMonth := "ACCRUAL,9100,,,,2026-09-30,,10.00,USD,,,,,,"
	_, problems, _, _ = readCSV(t, "gsap.csv", header, noMonth)
	if len(problems) != 1 || problems[0].Message != "Accrual month must be a month like 2026-09." {
		t.Errorf("accrual problems = %+v", problems)
	}
	noPOLineWithoutPO := "PAYMENT,1500,,,abc,2026-02-20,,10.00,USD,,,,,,"
	rows, problems, _, _ = readCSV(t, "gsap.csv", header, noPOLineWithoutPO)
	if len(problems) != 0 || rows[0].POLineNumber != "" {
		t.Errorf("a PO line without a PO number is ignored: %+v %+v", rows, problems)
	}
}

// A row with several problems lists all of them; rows keep their file line numbers, empty lines skipped.
func TestCSVFeed_EveryProblemWithItsRow_FR005(t *testing.T) {
	t.Parallel()
	_, problems, _, err := readCSV(t, "gsap.csv", header, validRow, "", ",,,,,,,,,,,,,,",
		"REFUND,,10,,,2026-13-01,,x,USD,,,,,,")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	columns := []string{}
	for _, p := range problems {
		if p.Row != 5 {
			t.Errorf("problem on row %d, want 5: %+v", p.Row, p)
		}
		columns = append(columns, p.Column)
	}
	if strings.Join(columns, ",") != "type,document_number,posting_date,amount" {
		t.Errorf("columns = %v", columns)
	}
}

func TestCSVFeed_HeaderBOMAndQuoting_FR004(t *testing.T) {
	t.Parallel()
	content := "\xEF\xBB\xBF Amount ,CURRENCY,Type,document_number,posting_date,cost_centre\n" +
		"\"1234.50\",EUR,po,\"45/0001\",2026-03-01,CC-1\n"
	rows, problems, unused, err := CSVFeed{}.Read(t.Context(), Delivery{FileName: "GSAP.CSV", Content: []byte(content)})
	if err != nil || len(problems) != 0 {
		t.Fatalf("Read = %v, %+v", err, problems)
	}
	if len(rows) != 1 || rows[0].DocumentNumber != "45/0001" || rows[0].Amount != amountOf(t, "1234.50") || rows[0].Currency != "EUR" {
		t.Errorf("rows = %+v", rows)
	}
	if len(unused) != 1 || unused[0] != "cost_centre" {
		t.Errorf("unused = %v", unused)
	}
	quotedVendor := "PO,4500,10,,,2026-01-12,,1.00,USD,V-1,\"Smith, Jones & Co\",,,,"
	rows, _, _, _ = readCSV(t, "gsap.csv", header, quotedVendor)
	if rows[0].VendorName != "Smith, Jones & Co" {
		t.Errorf("vendor = %q", rows[0].VendorName)
	}
}

func TestCSVFeed_Semicolons_038_FR035_L13(t *testing.T) {
	t.Parallel()
	semicolons := func(line string) string { return strings.ReplaceAll(line, ",", ";") }
	row := strings.Replace(semicolons(validRow), "Northlight Events GmbH", "Smith, Jones & Co", 1)

	rows, problems, unused, err := readCSV(t, "gsap.csv", semicolons(header), row)

	if err != nil || len(problems) != 0 || len(unused) != 0 {
		t.Fatalf("Read = %v, problems %+v, unused %v", err, problems, unused)
	}
	if len(rows) != 1 || rows[0].DocumentNumber != "4500001001" || rows[0].Amount != amountOf(t, "15800.00") ||
		rows[0].VendorName != "Smith, Jones & Co" {
		t.Errorf("rows = %+v", rows)
	}
}

// Any content, comma- or semicolon-separated, is read or refused without a panic (B9, 038 FR-035).
func FuzzCSVFeed(f *testing.F) {
	f.Add([]byte(header + "\n" + validRow + "\n"))
	f.Add([]byte(strings.ReplaceAll(header+"\n"+validRow+"\n", ",", ";")))
	f.Add([]byte("type;amount\n\"PO;1\";\"x\n"))
	f.Add([]byte(";;;\n;\n"))
	f.Fuzz(func(t *testing.T, content []byte) {
		_, _, _, _ = CSVFeed{}.Read(t.Context(), Delivery{FileName: "gsap.csv", Content: content})
	})
}

func TestCSVFeed_WholeFileRefusals_FR004(t *testing.T) {
	t.Parallel()
	big := header + "\n" + strings.Repeat(validRow+"\n", 700*1024/len(validRow)+1)
	many := header + "\n" + strings.Repeat("PO,1,,,,2026-01-01,,1,USD,,,,,,\n", 5001)
	cases := []struct {
		name, fileName, content, field, message string
	}{
		{"not .csv", "gsap.xlsx", header, "fileName", "Choose a .csv file — GSAP extracts are imported as CSV."},
		{"too large", "gsap.csv", big, "content", "The file is larger than 700 KB — split it into smaller files."},
		{"not UTF-8", "gsap.csv", header + "\nPO,1,,,,2026-01-01,,1,USD,,\xff\xfe,,,,", "content",
			`The file isn't UTF-8 text — save it as "CSV UTF-8" and try again.`},
		{"empty", "gsap.csv", "", "content", "The file has no rows."},
		{"header only", "gsap.csv", header + "\n\n", "content", "The file has no rows."},
		{"too many rows", "gsap.csv", many, "content", "The file has more than 5,000 rows — split it into smaller files."},
		{"columns missing", "gsap.csv", "document_number,posting_date,currency\n1,2026-01-01,USD", "content",
			"The first row must name the columns. Missing: type, amount."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, _, err := CSVFeed{}.Read(t.Context(), Delivery{FileName: c.fileName, Content: []byte(c.content)})
			var apiErr *errs.Error
			if !errors.As(err, &apiErr) || apiErr.Code != errs.CodeValidationFailed || len(apiErr.Fields) != 1 ||
				apiErr.Fields[0] != (errs.FieldError{Field: c.field, Message: c.message}) {
				t.Errorf("Read = %v, want %s: %q", err, c.field, c.message)
			}
		})
	}
}
