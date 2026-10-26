package settings

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// The rate feed seam (016 FR-008, D-259). GSAP's daily exchange rates are simulated: an administrator
// uploads the GSAP daily rate extract, and the only RateFeed, GSAPRateFile, turns it into rows. A later
// automatic feed implements RateFeed (or calls ApplyRateRows) and every rule of the import stays the same.
// MAPP connects to no external system.

// RateFeed turns what one source delivers into rows of rates.
type RateFeed interface {
	// Name is the feed's name, stored on every import: "gsapDailyRatesFile".
	Name() string
	// Read returns the rows of a delivery in order. An error is a delivery that can't be read at all (a
	// missing column, too large); it is an *errs.Error with a message for the person.
	Read(fileName string, content []byte) ([]FeedRow, error)
}

// FeedRow is one row of a rate delivery, as delivered: the texts are checked by the import (data-model §2.4).
type FeedRow struct {
	Row       int    // the row number in the file; the header is row 1
	RateType  string // "M": the only rate type used
	From      string // the currency the rate is quoted from: "USD"
	To        string // the currency of the rate: "EUR"
	ValidFrom string // "2026-10-19" or "20261019"
	Rate      string // units of To per 1 From: "0.9214"
}

// The limits of a GSAP rate file (FR-006).
const (
	maxRateFileBytes = 2 << 20 // 2 MB
	maxRateFileRows  = 20_000
	// MaxRateUploadBytes is the request body of the import operations: the file in base64 and its name.
	MaxRateUploadBytes = 3 << 20
)

// gsapFeedName is the name of the simulated GSAP daily rate feed.
const gsapFeedName = "gsapDailyRatesFile"

// The columns of the GSAP daily rate extract, in the order of the sample file.
var gsapColumns = []string{"rate_type", "from_currency", "to_currency", "valid_from", "rate"}

// Messages of an unreadable rate file (016 plan "User-visible messages"), on the field "content".
const (
	rateFileColumnMessage     = "The file needs the column '%s'."
	rateFileTooLargeMessage   = "The file is larger than 2 MB. Split it and import each part."
	rateFileTooManyMessage    = "The file has %s rows — import at most 20,000 rows at a time."
	rateFileUnreadableMessage = "This file can't be read. Save it as a .csv file in UTF-8 and try again."
	rateFileEmptyMessage      = "The file has no rows below the header."
)

// GSAPRateFile is the simulated GSAP daily rate extract (data-model §2.4): a CSV file in UTF-8 (a byte-order
// mark is allowed), comma-separated, with the header rate_type, from_currency, to_currency, valid_from, rate
// in any order and letter case; at most 2 MB and 20,000 rows.
type GSAPRateFile struct{}

// Name is "gsapDailyRatesFile".
func (GSAPRateFile) Name() string { return gsapFeedName }

// Read reads the rows of the file. It never logs the file, which holds business data.
func (GSAPRateFile) Read(_ string, content []byte) ([]FeedRow, error) {
	if len(content) > maxRateFileBytes {
		return nil, contentProblem(rateFileTooLargeMessage)
	}
	content = bytes.TrimPrefix(content, byteOrderUTF)
	if !utf8.Valid(content) {
		return nil, contentProblem(rateFileUnreadableMessage)
	}
	reader := csv.NewReader(bytes.NewReader(content))
	reader.FieldsPerRecord = -1 // short rows are read and checked cell by cell
	reader.TrimLeadingSpace = true
	header, err := reader.Read()
	if errors.Is(err, io.EOF) {
		return nil, contentProblem(fmt.Sprintf(rateFileColumnMessage, gsapColumns[0]))
	}
	if err != nil {
		return nil, contentProblem(rateFileUnreadableMessage)
	}
	index := map[string]int{}
	for i, cell := range header {
		name := strings.ToLower(strings.TrimSpace(cell))
		if _, seen := index[name]; !seen {
			index[name] = i
		}
	}
	for _, column := range gsapColumns {
		if _, found := index[column]; !found {
			return nil, contentProblem(fmt.Sprintf(rateFileColumnMessage, column))
		}
	}
	cell := func(record []string, column string) string {
		if i := index[column]; i < len(record) {
			return strings.TrimSpace(record[i])
		}
		return ""
	}
	var rows []FeedRow
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, contentProblem(rateFileUnreadableMessage)
		}
		line, _ := reader.FieldPos(0)
		if blank(record) {
			continue
		}
		if len(rows) == maxRateFileRows {
			return nil, contentProblem(fmt.Sprintf(rateFileTooManyMessage, "more than "+thousands(maxRateFileRows)))
		}
		rows = append(rows, FeedRow{
			Row: line, RateType: cell(record, "rate_type"), From: cell(record, "from_currency"), To: cell(record, "to_currency"),
			ValidFrom: cell(record, "valid_from"), Rate: cell(record, "rate"),
		})
	}
	if len(rows) == 0 {
		return nil, contentProblem(rateFileEmptyMessage)
	}
	return rows, nil
}

// blank reports whether every cell of record is empty.
func blank(record []string) bool {
	for _, c := range record {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}
