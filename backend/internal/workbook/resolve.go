package workbook

import (
	"fmt"
	"strings"

	"mapp/internal/platform/sheet"
)

const (
	pathSeparator   = " › "
	statusSubmitted = "submitted"
	statusApproved  = "approved"
)

func parentFields(l Level) []string {
	switch l {
	case LevelCampaign:
		return nil
	case LevelInitiative:
		return []string{fieldCampaign}
	case LevelActivity:
		return []string{fieldCampaign, fieldInitiative}
	case LevelLineItem:
	}
	return []string{fieldCampaign, fieldInitiative, fieldActivity}
}

// resolve finds the record a row's ID names and the parent its parent columns name (FR-029, FR-030, FR-033).
func (ap *applier) resolve(w *rowWork) {
	if id := w.s.text(w.row, fieldID); id != "" {
		stored, found := ap.byID[id]
		if !found || stored.Level != w.s.level {
			w.problem("ID", id, fmt.Sprintf(msgNotInPlan, id, withArticle(w.s.level)))
			return
		}
		w.stored = &stored
	}
	parentID, found := ap.parentOf(w)
	if !found {
		return
	}
	w.parentID = parentID
	if w.stored != nil && w.stored.ParentID != parentID {
		current := ap.byID[w.stored.ParentID].Values.Name
		w.problem("", w.stored.Values.Name, fmt.Sprintf(msgMoveOnScreen, levelWord(w.s.level), current))
	}
}

func (ap *applier) parentOf(w *rowWork) (string, bool) {
	parent, path := ap.plan.ID, ""
	for i, field := range parentFields(w.s.level) {
		text := w.s.text(w.row, field)
		if text == "" {
			return "", false
		}
		level := levels[i]
		path = joinPath(path, text)
		id, found := ap.find(level, parent, text)
		if found {
			parent = id
			continue
		}
		if ref, bad := ap.bad[path]; bad {
			w.leftOut = true
			w.problem("", text, fmt.Sprintf(msgParentProblem, levelWord(level), ref.sheet, ref.row))
			return "", false
		}
		heading, _ := columnOfField(w.s, field)
		w.problem(heading.heading, text, fmt.Sprintf(msgNoParent, levelWord(level), text, sheetName(level)))
		return "", false
	}
	return parent, true
}

// find finds a record of level under parent by ID, or by name ignoring case, created by an earlier row too.
func (ap *applier) find(level Level, parent, text string) (string, bool) {
	if stored, found := ap.byID[text]; found && stored.Level == level && stored.ParentID == parent {
		return stored.ID, true
	}
	id, found := ap.names[nameKey{level, parent, strings.ToLower(text)}]
	return id, found
}

func joinPath(path, text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	if path == "" {
		return text
	}
	return path + pathSeparator + text
}

// pathOf is the row's place in the file: its parents' texts and its name, which children name it by.
func pathOf(s sheetRows, row sheet.Row) string {
	path := ""
	for _, field := range parentFields(s.level) {
		path = joinPath(path, s.text(row, field))
	}
	return joinPath(path, s.text(row, fieldName))
}

func (ap *applier) findTwice(s sheetRows) {
	first := map[string]sheet.Row{}
	for _, row := range s.rows {
		if s.text(row, fieldName) == "" {
			continue
		}
		path := pathOf(s, row)
		earlier, seen := first[path]
		if !seen {
			first[path] = row
			continue
		}
		parent := ap.plan.Name
		if fields := parentFields(s.level); len(fields) > 0 {
			parent = s.text(row, fields[len(fields)-1])
		}
		reason := fmt.Sprintf(msgTwice, parent, earlier.Number, row.Number)
		ap.twice[rowRef{s.name, earlier.Number}] = reason
		ap.twice[rowRef{s.name, row.Number}] = reason
	}
}

func needsApproval(stored Stored, changed map[string]bool) bool {
	inApproval := stored.Status == statusSubmitted || stored.Status == statusApproved
	return stored.Level == LevelActivity && inApproval &&
		(changed[fieldPlanned] || changed[fieldStart] || changed[fieldEnd] || changed[fieldOngoing])
}

func statusWord(status string) string {
	if status == statusSubmitted {
		return "Submitted"
	}
	return "Approved"
}

// columnOfField finds the column of a field the owning operation names: "plannedCost.amount" is the
// Planned cost column, "attributes.SA-1001" the attribute's.
func columnOfField(s sheetRows, field string) (column, bool) {
	if !strings.HasPrefix(field, attrPrefix) {
		field, _, _ = strings.Cut(field, ".")
	}
	for _, c := range s.columns {
		if c.field == field {
			return c, true
		}
	}
	return column{}, false
}
