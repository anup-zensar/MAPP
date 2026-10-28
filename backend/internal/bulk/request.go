package bulk

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/errs"
)

const (
	maxRecords     = 1000
	maxComment     = 2000
	fieldRecordIDs = "recordIds"
	fieldOwnerID   = "ownerId"
	fieldTargetID  = "targetId"
	fieldKey       = "field.key"
	fieldValue     = "field.value"
	msgChooseOne   = "Choose one of the listed options."
	kindDate       = "date"
	kindSingle     = "singleChoice"
	kindYesNo      = "yesNo"
)

func levelOf(id string) (Level, bool) {
	prefix, _, _ := strings.Cut(id, "-")
	switch prefix {
	case "C":
		return LevelCampaign, true
	case "I":
		return LevelInitiative, true
	case "A":
		return LevelActivity, true
	}
	return "", false
}

func uniqueIDs(ids []string) []string {
	unique := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" && !slices.Contains(unique, id) {
			unique = append(unique, id)
		}
	}
	return unique
}

func countByLevel(ids []string) LevelCounts {
	var c LevelCounts
	for _, id := range ids {
		switch level, _ := levelOf(id); level {
		case LevelCampaign:
			c.Campaigns++
		case LevelInitiative:
			c.Initiatives++
		case LevelActivity:
			c.Activities++
		}
	}
	return c
}

func isOneLevel(c LevelCounts) bool {
	levels := 0
	for _, n := range []int{c.Campaigns, c.Initiatives, c.Activities} {
		if n > 0 {
			levels++
		}
	}
	return levels <= 1
}

// checkRequest finds every problem of a request that needs no record read, all at once.
func checkRequest(req Request, ids []string) error {
	var p errs.Problems
	switch {
	case len(ids) == 0:
		p.Add(fieldRecordIDs, msgSelectOne)
	case len(ids) > maxRecords:
		p.Add(fieldRecordIDs, msgTooMany)
	case slices.ContainsFunc(ids, func(id string) bool { _, ok := levelOf(id); return !ok }):
		p.Add(fieldRecordIDs, msgNotSelectable)
	}
	if _, known := verbs[req.Action]; !known {
		p.Add("action", msgChooseAction)
	}
	actionProblems(&p, req, countByLevel(ids))
	return p.Err()
}

func actionProblems(p *errs.Problems, req Request, selected LevelCounts) {
	if field, message := missingChoice(req); message != "" {
		p.Add(field, message)
	}
	if req.Action == ActionSubmit && utf8.RuneCountInString(req.Comment) > maxComment {
		p.Add("comment", msgCommentTooLong)
	}
	needsOneLevel := req.Action == ActionChangeField || req.Action == ActionCopy || req.Action == ActionMove
	if needsOneLevel && !isOneLevel(selected) {
		p.Add(fieldRecordIDs, fmt.Sprintf(msgOneLevel, actionWord(req.Action)))
	}
}

// missingChoice names what the action needs chosen and the request left empty.
func missingChoice(req Request) (field, message string) {
	switch req.Action {
	case ActionChangeOwner:
		if strings.TrimSpace(req.OwnerID) == "" {
			return fieldOwnerID, msgChoosePerson
		}
	case ActionChangeField:
		if req.Field == nil || strings.TrimSpace(req.Field.Key) == "" {
			return fieldKey, msgChooseField
		}
	case ActionCopy, ActionMove:
		if strings.TrimSpace(req.TargetID) == "" {
			return fieldTargetID, chooseTargetMessage(req.Action)
		}
	case ActionSubmit, ActionDelete:
	}
	return "", ""
}

func chooseTargetMessage(action Kind) string {
	if action == ActionCopy {
		return msgChooseCopy
	}
	return msgChooseMove
}

// fieldOf finds the field Change field sets among those offered for the level, and checks its value.
func fieldOf(offered []Field, change FieldChange, level Level) (Field, error) {
	i := slices.IndexFunc(offered, func(f Field) bool { return f.Key == change.Key })
	if i < 0 {
		return Field{}, errs.Validation(errs.FieldError{Field: fieldKey,
			Message: fmt.Sprintf(msgFieldNotOffered, change.Key, levelPlural(level))})
	}
	field := offered[i]
	if change.Value == nil {
		if field.Required {
			return Field{}, errs.Validation(errs.FieldError{Field: fieldValue, Message: fmt.Sprintf(msgRequiredField, field.Label)})
		}
		return field, nil
	}
	if message := valueProblem(field, *change.Value); message != "" {
		return Field{}, errs.Validation(errs.FieldError{Field: fieldValue, Message: message})
	}
	return field, nil
}

func valueProblem(field Field, v FieldValue) string {
	switch field.Kind {
	case kindDate:
		if v.Date == nil {
			return msgChooseValue
		}
	case kindYesNo:
		if v.YesNo == nil {
			return msgChooseValue
		}
	default:
		return choiceProblem(field, v.OptionIDs)
	}
	return ""
}

func choiceProblem(field Field, optionIDs []string) string {
	if len(optionIDs) == 0 || (field.Kind == kindSingle && len(optionIDs) > 1) {
		return msgChooseValue
	}
	for _, id := range optionIDs {
		if !slices.ContainsFunc(field.Options, func(o FieldOption) bool { return o.ID == id }) {
			return msgChooseOne
		}
	}
	return ""
}
