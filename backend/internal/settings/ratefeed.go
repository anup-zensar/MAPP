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

// RateFeed is the seam for GSAP rates (016 FR-008): GSAPRateFile reads uploads; MAPP calls no external system.
type RateFeed interface {
	Name() string
	Read(fileName string, content []byte) ([]FeedRow, error)
}

type FeedRow struct {
	Row       int
	RateType  string
	From      string
	To        string
	ValidFrom string
	Rate      string
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
	rateFileColumnsMessage    = "The file needs the columns %s and '%s'."
	rateFileTooLargeMessage   = "The file is larger than 2 MB. Split it and import each part."
	rateFileTooManyMessage    = "The file has %s rows — import at most 20,000 rows at a time."
	rateFileUnreadableMessage = "This file can't be read. Save it as a .csv file in UTF-8 and try again."
	rateFileEmptyMessage      = "The file has no rows below the header."
	rateFileNothingMessage    = "This file is empty."
)

// GSAPRateFile reads the GSAP daily rate extract (data-model §2.4): UTF-8 CSV, header in any order and case.
type GSAPRateFile struct{}

func (GSAPRateFile) Name() string { return gsapFeedName }

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

// headerIndex names every missing column in one message, since a field shows one message (038 FR-034).
func headerIndex(header []string) (map[string]int, error) {
	index := map[string]int{}
	for i, cell := range header {
		name := strings.ToLower(strings.TrimSpace(cell))
		if _, seen := index[name]; !seen {
			index[name] = i
		}
	}
	var missing []string
	for _, column := range gsapColumns {
		if _, found := index[column]; !found {
			missing = append(missing, column)
		}
	}
	switch len(missing) {
	case 0:
		return index, nil
	case 1:
		return nil, contentProblem(fmt.Sprintf(rateFileColumnMessage, missing[0]))
	}
	last := len(missing) - 1
	return nil, contentProblem(fmt.Sprintf(rateFileColumnsMessage, "'"+strings.Join(missing[:last], "', '")+"'", missing[last]))
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

// UseRateFeed sets the feed rate files are read with (FR-008).
func (s *Service) UseRateFeed(feed RateFeed) {
	s.feed = feed
}
