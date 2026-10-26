package settings

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

// The limits of an option import (FR-031, D-051, D-053): the file — decoded, at most 700 KB, so that its
// base64 text fits in the API's 1 MB request body — and its rows.
const (
	maxImportBytes = 700 * 1024
	maxImportRows  = 5000
)

// headerLabel is the first cell of a header row, ignoring letter case: the template's "Option".
const headerLabel = "option"

// Messages of the plan's "User-visible messages" table, row "Import".
const (
	fileTooLargeMessage   = "The file is larger than 700 KB — split it into smaller files."
	fileUnreadableMessage = "This file can't be read. Save it again as .csv or .xlsx and try again."
	oldExcelMessage       = "Choose a .csv or .xlsx file. An older .xls file can be saved as .xlsx in Excel."
	tooManyRowsMessage    = "The file has %s rows — import at most 5,000 rows at a time."
	noRowsMessage         = "The file has no rows below the header."
	emptyRowMessage       = "Empty — enter an option."
	longRowMessage        = "Longer than 120 characters."
	twiceInFileMessage    = "Appears twice in the file (also row %d)."
	enterParentMessage    = "Enter the parent option in the second column."
	parentNotFoundMessage = "Parent option '%s' not found in %s."
	overLimitMessage      = "Over the 5,000-option limit of this attribute."
	alreadyOptionMessage  = "Already an option — skipped."
	nothingToAddMessage   = "Nothing to add — every row is already an option or has a problem."
	importedVerb          = "imported options into"
	importedDetails       = "Added %s from '%s'; skipped %s."
)

// CheckImport says what importing the file into the options of the attribute attributeID would do, row by
// row (FR-031, US6-1): the rows to add, the rows already present (skipped) and the rows with problems, each
// with its number in the file. It adds nothing. A file that cannot be used at all is refused on content.
// Every signed-in person may use it (FR-008).
func (s *Service) CheckImport(ctx context.Context, a actor.Actor, attributeID string, file ImportFile) (ImportCheck, error) {
	rows, fileErr := importRows(file)
	var check ImportCheck
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		f, attribute, err := s.importTarget(ctx, attributeID)
		if err != nil {
			return err
		}
		if fileErr != nil {
			return fileErr
		}
		check = f.planImport(attribute, rows).check
		return nil
	})
	return check, err
}

// Import adds the new valid options of the file to the attribute attributeID (FR-031, US6-2): it checks the
// file again, as CheckImport does, inside one transaction, and adds the rows to add in file order after the
// existing options. It never removes, renames, relinks or reactivates an option. One "changed" entry on the
// attribute records the whole import; with nothing to add, the import is refused and nothing is logged.
func (s *Service) Import(ctx context.Context, a actor.Actor, attributeID string, file ImportFile) (ImportResult, error) {
	rows, fileErr := importRows(file)
	var result ImportResult
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		f, attribute, err := s.importTarget(ctx, attributeID)
		if err != nil {
			return err
		}
		if fileErr != nil {
			return fileErr
		}
		plan := f.planImport(attribute, rows)
		if len(plan.add) == 0 {
			return errs.Validation(errs.FieldError{Field: "content", Message: nothingToAddMessage})
		}
		position := 0
		for _, o := range f.options[attribute.ID] {
			position = max(position, o.Position)
		}
		if err = s.addOptions(ctx, a, plan.add, position); err != nil {
			return err
		}
		result = ImportResult{Added: len(plan.add), AlreadyPresent: len(plan.check.AlreadyPresent), Problems: len(plan.check.Problems)}
		skipped := result.AlreadyPresent + result.Problems
		_, err = s.history.Log(ctx, a, history.LogInput{
			Kind: attributeKind, RecordID: attribute.ID, RecordName: attribute.Label, Action: history.ActionChanged, Verb: importedVerb,
			Details: fmt.Sprintf(importedDetails, plural(result.Added, "option", "options"), strings.TrimSpace(file.FileName),
				plural(skipped, "row", "rows")),
		})
		return err
	})
	return result, err
}

// addOptions stores new options all at once, numbered from after+1 in their order, with the next IDs.
func (s *Service) addOptions(ctx context.Context, a actor.Actor, options []Option, after int) error {
	newIDs, err := s.ids.NextIDs(ctx, ids.PrefixShellOption, len(options))
	if err != nil {
		return err
	}
	added := make([]Option, 0, len(options))
	for i, o := range options {
		o.ID, o.Position, o.Meta = newIDs[i], after+i+1, record.NewMeta(a, s.now())
		added = append(added, o)
	}
	return s.repo.AddOptions(ctx, added)
}

// importTarget reads the options of every attribute and returns the attribute attributeID, which must be
// a choice attribute — or answers not found, or refuses the import on content.
func (s *Service) importTarget(ctx context.Context, attributeID string) (formData, Attribute, error) {
	f, err := s.loadForm(ctx, "")
	if err != nil {
		return formData{}, Attribute{}, err
	}
	attribute, found := f.byID[attributeID]
	if !found {
		return formData{}, Attribute{}, errs.NotFound(attributeGoneMessage)
	}
	if !attribute.IsChoice() {
		return formData{}, Attribute{}, contentProblem(fmt.Sprintf(noOptionsMessage, attribute.Label, typeWords[attribute.Type]))
	}
	return f, attribute, nil
}

// importRows decodes the file and returns its data rows, without the header row (its first cell reads
// "Option"), or the reason the whole file is refused: not base64, larger than 700 KB, an .xls file,
// unreadable, more than 5,000 rows or none. The file is read in memory only.
func importRows(file ImportFile) ([]importRow, error) {
	content, err := base64.StdEncoding.DecodeString(file.Content)
	if err != nil {
		return nil, contentProblem(fileUnreadableMessage)
	}
	if len(content) > maxImportBytes {
		return nil, contentProblem(fileTooLargeMessage)
	}
	rows, err := readRows(content)
	switch {
	case errors.Is(err, errOldExcel):
		return nil, contentProblem(oldExcelMessage)
	case err != nil:
		return nil, contentProblem(fileUnreadableMessage)
	}
	if len(rows) > 0 && strings.EqualFold(strings.TrimSpace(rows[0].label), headerLabel) {
		rows = rows[1:]
	}
	switch {
	case len(rows) == 0:
		return nil, contentProblem(noRowsMessage)
	case len(rows) > maxImportRows:
		return nil, contentProblem(fmt.Sprintf(tooManyRowsMessage, thousands(int64(len(rows)))))
	}
	return rows, nil
}

// contentProblem is the validation error of an import file that cannot be used: one problem on content.
func contentProblem(message string) error {
	return errs.Validation(errs.FieldError{Field: "content", Message: message})
}

// importPlan is what importing rows into an attribute does: every row's outcome, and the new options to
// add in file order (no id, place or bookkeeping yet).
type importPlan struct {
	check ImportCheck
	add   []Option
}

// planImport works out the outcome of each row for the attribute a, in order (002 plan, Options): an empty
// option, one longer than 120 characters, a dependent's row without parent option or with a parent option
// that is not an active option of the parent attribute (ignoring letter case), and a row that appeared
// before in the file — the same option, for a dependent the same option and parent — are problems; an
// option the attribute has, active or not, ignoring letter case, is already present; an option beyond the
// 5,000 of an attribute is a problem; any other row is added. Rows of a dependent with the same option and
// different parents become one option linked to each parent.
func (f formData) planImport(a Attribute, rows []importRow) importPlan {
	plan := importPlan{check: ImportCheck{TotalRows: len(rows), ToAdd: []ImportRow{}, AlreadyPresent: []ImportRow{}, Problems: []ImportRow{}}}
	parent, dependent := f.byID[a.ParentAttributeID]
	existing := map[string]bool{}
	for _, o := range f.options[a.ID] {
		existing[labelKey(o.Label)] = true
	}
	parents := map[string]Option{}
	if dependent {
		for _, o := range f.options[parent.ID] {
			parents[labelKey(o.Label)] = o
		}
	}
	seen := map[string]int{}  // option (and parent option) → the row it first appeared on
	added := map[string]int{} // option → its index in plan.add
	count := len(f.options[a.ID])

	for _, r := range rows {
		row := ImportRow{Row: r.number, Label: strings.TrimSpace(r.label)}
		if dependent {
			row.ParentLabel = strings.TrimSpace(r.parent)
		}
		problem := func(message string) {
			row.Message = message
			plan.check.Problems = append(plan.check.Problems, row)
		}
		var parentOption Option
		if row.Label == "" {
			problem(emptyRowMessage)
			continue
		}
		if utf8.RuneCountInString(row.Label) > maxOptionLabel {
			problem(longRowMessage)
			continue
		}
		key := labelKey(row.Label)
		if dependent {
			found := false
			parentOption, found = parents[labelKey(row.ParentLabel)]
			switch {
			case row.ParentLabel == "":
				problem(enterParentMessage)
				continue
			case !found:
				problem(fmt.Sprintf(parentNotFoundMessage, row.ParentLabel, parent.Label))
				continue
			case !parentOption.Active:
				problem(fmt.Sprintf(inactiveLinkMessage, parentOption.Label, midSentence(parent.Label)))
				continue
			}
		}
		rowKey := key + "\x00" + parentOption.ID
		if earlier, twice := seen[rowKey]; twice {
			problem(fmt.Sprintf(twiceInFileMessage, earlier))
			continue
		}
		seen[rowKey] = r.number
		if existing[key] {
			row.Message = alreadyOptionMessage
			plan.check.AlreadyPresent = append(plan.check.AlreadyPresent, row)
			continue
		}
		if i, joined := added[key]; joined { // a dependent's option seen with another parent: one more link
			plan.add[i].ParentOptionIDs = append(plan.add[i].ParentOptionIDs, parentOption.ID)
			plan.check.ToAdd = append(plan.check.ToAdd, row)
			continue
		}
		if count >= maxOptions {
			problem(overLimitMessage)
			continue
		}
		o := Option{AttributeID: a.ID, Label: row.Label, ParentOptionIDs: []string{}, Active: true}
		if dependent {
			o.ParentOptionIDs = append(o.ParentOptionIDs, parentOption.ID)
		}
		added[key] = len(plan.add)
		plan.add = append(plan.add, o)
		count++
		plan.check.ToAdd = append(plan.check.ToAdd, row)
	}
	return plan
}

// labelKey is how an import compares labels: ignoring letter case and surrounding spaces.
func labelKey(label string) string {
	return strings.ToLower(strings.TrimSpace(label))
}
