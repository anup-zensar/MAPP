package settings

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/errs"
)

// The rate feed seam (016 FR-008, D-259): GSAPRateFile reads an uploaded GSAP daily rate extract; a later
// automatic feed implements RateFeed (or calls ApplyRateRows) and the import's rules stay the same.

type RateFeed interface {
	Name() string
	// Read returns the rows in order; an error (an *errs.Error) means the delivery can't be read at all.
	Read(fileName string, content []byte) ([]FeedRow, error)
}

type FeedRow struct {
	Row       int
	RateType  string
	From      string
	To        string
	ValidFrom string
	Rate      string // units of To per 1 From: "0.9214"
}

const (
	maxRateFileBytes   = 2 << 20
	maxRateFileRows    = 20_000
	MaxRateUploadBytes = 3 << 20
)

const gsapFeedName = "gsapDailyRatesFile"

var gsapColumns = []string{"rate_type", "from_currency", "to_currency", "valid_from", "rate"}

const (
	rateFileColumnMessage     = "The file needs the column '%s'."
	rateFileTooLargeMessage   = "The file is larger than 2 MB. Split it and import each part."
	rateFileTooManyMessage    = "The file has %s rows — import at most 20,000 rows at a time."
	rateFileUnreadableMessage = "This file can't be read. Save it as a .csv file in UTF-8 and try again."
	rateFileEmptyMessage      = "The file has no rows below the header."
	rateFileNothingMessage    = "This file is empty."
)

type GSAPRateFile struct{}

func (GSAPRateFile) Name() string { return gsapFeedName }

// Read never logs the file, which holds business data.
func (GSAPRateFile) Read(_ string, content []byte) ([]FeedRow, error) {
	if len(content) > maxRateFileBytes {
		return nil, contentProblem(rateFileTooLargeMessage)
	}
	content = bytes.TrimPrefix(content, byteOrderUTF)
	if !utf8.Valid(content) {
		return nil, contentProblem(rateFileUnreadableMessage)
	}
	if len(bytes.TrimSpace(content)) == 0 {
		return nil, contentProblem(rateFileNothingMessage)
	}
	reader := csv.NewReader(bytes.NewReader(content))
	reader.FieldsPerRecord = -1 // short rows are read and checked cell by cell
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true // a stray quote makes a row's cell wrong, not the file unreadable (038 FR-034)
	header, err := reader.Read()
	if err != nil {
		return nil, contentProblem(rateFileUnreadableMessage)
	}
	index, err := headerIndex(header)
	if err != nil {
		return nil, err
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

func headerIndex(header []string) (map[string]int, error) {
	index := map[string]int{}
	for i, cell := range header {
		name := strings.ToLower(strings.TrimSpace(cell))
		if _, seen := index[name]; !seen {
			index[name] = i
		}
	}
	var problems errs.Problems
	for _, column := range gsapColumns {
		if _, found := index[column]; !found {
			problems.Add("content", fmt.Sprintf(rateFileColumnMessage, column))
		}
	}
	return index, problems.Err()
}

func blank(record []string) bool {
	for _, c := range record {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func (s *Service) rateFeed() RateFeed {
	if s.feed == nil {
		return GSAPRateFile{}
	}
	return s.feed
}

func (s *Service) UseRateFeed(feed RateFeed) {
	s.feed = feed
}
