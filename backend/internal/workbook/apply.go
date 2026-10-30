package workbook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/platform/sheet"
)

const maxProblemValue = 200

// applier runs the rows of an upload against a plan inside one Write: every row is created, updated, left
// unchanged or left out with its problems (FR-029 – FR-033). A check and an import run the same applier.
type applier struct {
	a       actor.Actor
	port    Writer
	plan    Plan
	byID    map[string]Stored
	names   map[nameKey]string
	bad     map[string]rowRef
	twice   map[rowRef]string
	result  Check
	counts  map[Level]*LevelCounts
	records []ImportRecord
}

type nameKey struct {
	level    Level
	parentID string
	name     string
}

type rowRef struct {
	sheet string
	row   int
}

// rowWork is one row on its way: its parent, its record when it names one, its values and its problems.
type rowWork struct {
	s        sheetRows
	row      sheet.Row
	parentID string
	stored   *Stored
	values   Values
	changed  map[string]bool
	problems []Problem
	leftOut  bool
}

func newApplier(a actor.Actor, port Writer, plan Plan) *applier {
	ap := &applier{a: a, port: port, plan: plan, byID: map[string]Stored{}, names: map[nameKey]string{},
		bad: map[string]rowRef{}, twice: map[rowRef]string{}, counts: map[Level]*LevelCounts{},
		result: Check{Problems: []Problem{}, IgnoredColumns: []string{}, Levels: []LevelCounts{}}}
	for _, r := range plan.Records {
		ap.byID[r.ID] = r
		ap.names[nameKey{r.Level, r.ParentID, strings.ToLower(r.Values.Name)}] = r.ID
	}
	for _, l := range levels {
		ap.counts[l] = &LevelCounts{Level: l}
	}
	return ap
}

func (ap *applier) run(ctx context.Context, sheets []sheetRows) error {
	for _, s := range sheets {
		ap.findTwice(s)
		for _, row := range s.rows {
			if err := ap.apply(ctx, s, row); err != nil {
				return err
			}
		}
		if err := ap.flush(ctx); err != nil {
			return err
		}
	}
	ap.finish()
	return nil
}

// flush stores what the sheet created, before the next sheet's rows name it, and takes the readable ids.
func (ap *applier) flush(ctx context.Context) error {
	readable, err := ap.port.Flush(ctx)
	if err != nil || len(readable) == 0 {
		return err
	}
	for key, id := range ap.names {
		if fresh, found := readable[id]; found {
			ap.names[key] = fresh
		}
	}
	for i, r := range ap.records {
		if fresh, found := readable[r.RecordID]; found {
			ap.records[i].RecordID = fresh
		}
	}
	return nil
}

func (ap *applier) apply(ctx context.Context, s sheetRows, row sheet.Row) error {
	w := &rowWork{s: s, row: row, changed: map[string]bool{}}
	ap.result.Rows++
	ap.checkRequired(w)
	if reason, twice := ap.twice[rowRef{s.name, row.Number}]; twice {
		w.problem("", s.text(row, fieldName), reason)
	}
	ap.resolve(w)
	if !w.leftOut {
		ap.readCells(w)
	}
	if len(w.problems) > 0 || w.leftOut {
		ap.leaveOut(w)
		return nil
	}
	return ap.save(ctx, w)
}

func (w *rowWork) problem(column, value, reason string) {
	w.problems = append(w.problems, Problem{Sheet: w.s.name, Row: w.row.Number, Column: column, Value: shortened(value), Reason: reason})
}

// shortened is a value found as a problem shows it: at most maxProblemValue characters, then "…".
func shortened(value string) string {
	if utf8.RuneCountInString(value) <= maxProblemValue {
		return value
	}
	return string([]rune(value)[:maxProblemValue]) + "…"
}

func (ap *applier) checkRequired(w *rowWork) {
	for _, c := range w.s.columns {
		if c.required() && w.s.text(w.row, c.field) == "" {
			w.problem(c.heading, "", msgRequired)
		}
	}
}

// readCells reads every cell into the row's values; a cell that still holds its record's value is kept as it
// is, so an inactive option already there may stay.
func (ap *applier) readCells(w *rowWork) {
	w.values = Values{Attributes: map[string]json.RawMessage{}}
	if w.stored != nil {
		w.values = w.stored.Values
		w.values.Attributes = maps.Clone(w.stored.Values.Attributes)
		if w.values.Attributes == nil {
			w.values.Attributes = map[string]json.RawMessage{}
		}
	}
	if w.s.level != LevelLineItem {
		w.values.Currency = ap.plan.Currency
	}
	for _, c := range w.s.columns {
		if isStructural(c.field) {
			continue
		}
		text := w.s.text(w.row, c.field)
		if w.stored != nil && ap.unchanged(c, text, w.stored.Values) {
			continue
		}
		if problem := checkCell(c, text, &w.values, ap.plan.Choices); problem != "" {
			w.problem(c.heading, text, problem)
			continue
		}
		w.changed[c.field] = true
	}
}

// unchanged reports whether a cell still holds the record's value: the same text, or the same amount written
// as a number.
func (ap *applier) unchanged(c column, text string, stored Values) bool {
	if text == format(c, stored, ap.plan.Choices) {
		return true
	}
	was, isAmount := amountField(c.field, &stored)
	var read *money.Amount
	return isAmount && *was != nil && parseAmount(text, &read) == "" && read != nil && *read == **was
}

// checkCell reads a cell into values; a name must be a single line.
func checkCell(c column, text string, values *Values, ch Choices) string {
	if c.field == fieldName {
		if problem := record.CheckSingleLine(text); problem != "" {
			return problem
		}
	}
	return parse(c, text, values, ch)
}

func isStructural(field string) bool {
	return field == fieldID || field == fieldCampaign || field == fieldInitiative || field == fieldActivity
}

func (ap *applier) leaveOut(w *rowWork) {
	if len(w.problems) == 0 {
		w.problem("", "", msgGone)
	}
	ap.result.Problems = append(ap.result.Problems, w.problems...)
	ap.result.WithProblems++
	ap.counts[w.s.level].LeftOut++
	ap.bad[pathOf(w.s, w.row)] = rowRef{w.s.name, w.row.Number}
}

func (ap *applier) save(ctx context.Context, w *rowWork) error {
	counts := ap.counts[w.s.level]
	if w.stored != nil && len(w.changed) == 0 {
		counts.Unchanged++
		ap.result.Ready++
		return nil
	}
	if w.stored != nil && needsApproval(*w.stored, w.changed) {
		w.problem("", w.stored.Values.Name, fmt.Sprintf(msgReapproval, statusWord(w.stored.Status)))
		ap.leaveOut(w)
		return nil
	}
	id, action, err := ap.write(ctx, w)
	if done, refusedErr := ap.refused(w, err); done || refusedErr != nil {
		return refusedErr
	}
	ap.names[nameKey{w.s.level, w.parentID, strings.ToLower(w.values.Name)}] = id
	ap.records = append(ap.records, ImportRecord{Level: w.s.level, RecordID: id, Name: w.values.Name, ParentID: w.parentID, Action: action})
	if action == ActionCreated {
		counts.Create++
	} else {
		counts.Update++
	}
	ap.result.Ready++
	if ap.leavesNeededEmpty(w) {
		ap.result.NeedMoreFields++
	}
	return nil
}

func (ap *applier) write(ctx context.Context, w *rowWork) (string, string, error) {
	if w.stored != nil {
		return w.stored.ID, ActionUpdated, ap.port.Update(ctx, ap.a, Change{ID: w.stored.ID, Values: w.values, Present: w.changed})
	}
	id, err := ap.port.Create(ctx, ap.a, NewRecord{Level: w.s.level, ParentID: w.parentID, Values: w.values})
	return id, ActionCreated, err
}

// refused leaves the row out with the owning operation's reasons, each under its column; any other error
// stops the run.
func (ap *applier) refused(w *rowWork, err error) (bool, error) {
	if err == nil {
		return false, nil
	}
	apiErr, ok := errors.AsType[*errs.Error](err)
	if !ok {
		return true, err
	}
	switch {
	case apiErr.Code == errs.CodeNotFound:
		w.problem("", w.s.text(w.row, fieldID), msgGone)
	case len(apiErr.Fields) == 0:
		w.problem("", "", apiErr.Message)
	}
	for _, f := range apiErr.Fields {
		if c, found := columnOfField(w.s, f.Field); found {
			w.problem(c.heading, w.s.text(w.row, c.field), f.Message)
		} else {
			w.problem("", "", f.Message)
		}
	}
	ap.leaveOut(w)
	return true, nil
}

func (ap *applier) leavesNeededEmpty(w *rowWork) bool {
	for _, c := range columnsOf(w.s.level, ap.plan.Choices) {
		needed := c.marking == markPublish || c.marking == markSubmit
		if needed && format(c, w.values, ap.plan.Choices) == "" {
			return true
		}
	}
	return false
}

func (ap *applier) finish() {
	for _, l := range levels {
		ap.result.Levels = append(ap.result.Levels, *ap.counts[l])
	}
	changes := 0
	for _, c := range ap.counts {
		changes += c.Create + c.Update
	}
	ap.result.NothingToChange = changes == 0 && ap.result.WithProblems == 0
}
