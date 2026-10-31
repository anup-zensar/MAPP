package performance

import (
	"strings"
	"time"

	"mapp/internal/platform/money"
)

// Env is resolved once for the whole file, never per row (research R2).
type Env struct {
	Today      time.Time
	Currencies map[string]bool
	Resolved   Resolved
	Approved   func(activityID string) bool
}

type RowResult struct {
	Row        Row
	Outcome    Outcome
	ActivityID string
	MatchedOn  MatchedOn
	Reason     string
	Faults     []Fault
	Candidates []string
	FactDate   time.Time
	Measured   []Measured
}

// Classify decides every row of one file. A reference repeated inside the file is rejected rather than
// overwriting its neighbour; a reference repeated in a later load replaces the fact instead (046-C4).
func Classify(rows []Row, env Env) []RowResult {
	pass := &filePass{env: env, seen: map[string]bool{}}
	results := make([]RowResult, 0, len(rows))
	for _, row := range rows {
		results = append(results, pass.row(row))
	}
	return results
}

type filePass struct {
	env  Env
	seen map[string]bool
}

func (p *filePass) row(row Row) RowResult {
	faults := Validate(row, p.env.Today, p.env.Currencies)
	reference := strings.TrimSpace(row.Reference.Value)
	if reference != "" {
		if p.seen[reference] {
			faults = append(faults, Fault{Column: row.Reference.Column,
				Code: faultRepeatedReference, Message: msgReferenceRepeated})
		}
		p.seen[reference] = true
	}
	if len(faults) > 0 {
		return RowResult{Row: row, Outcome: OutcomeRejected, Reason: reasonRejected + faults[0].Code, Faults: faults}
	}
	decided := Decide(row.Keys, p.env.Resolved, p.env.Approved)
	return RowResult{Row: row, Outcome: decided.Outcome, ActivityID: decided.ActivityID,
		MatchedOn: decided.MatchedOn, Reason: decided.Reason, Candidates: decided.Candidates,
		FactDate: mustDate(row.Date.Value), Measured: measuredOf(row)}
}

// mustDate and measuredOf read only what Validate already accepted, so neither can meet a bad value.
func mustDate(raw string) time.Time {
	on, err := time.Parse(time.DateOnly, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}
	}
	return on
}

func measuredOf(row Row) []Measured {
	var measured []Measured
	for _, reading := range row.filled() {
		raw := strings.TrimSpace(reading.Value)
		item := Measured{Measure: reading.Measure}
		if reading.money() {
			amount, err := money.ParseInput(money.Input{Amount: raw, Currency: currencyOf(row)})
			if err != nil {
				continue
			}
			item.Amount = &amount
		} else {
			count, ok := wholeCount(raw)
			if !ok {
				continue
			}
			item.Count = &count
		}
		measured = append(measured, item)
	}
	return measured
}

func currencyOf(row Row) string {
	return strings.ToUpper(strings.TrimSpace(row.Currency.Value))
}
