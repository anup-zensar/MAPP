package bulk

import (
	"fmt"
	"strings"
)

const (
	msgSelectOne       = "Select at least one record."
	msgTooMany         = "Bulk actions handle up to 1,000 records at a time — narrow the filters"
	msgNotSelectable   = "Only campaigns, initiatives and activities can be selected."
	msgChooseAction    = "Choose one of the listed actions."
	msgChoosePerson    = "Choose a person."
	msgInactivePerson  = "%s is inactive — choose an active person."
	msgChooseField     = "Choose a field."
	msgChooseValue     = "Choose a value, or choose Clear the value."
	msgRequiredField   = "%s is required — it can't be cleared."
	msgFieldNotOffered = "%s can't be changed for %s here."
	msgChooseCopy      = "Choose where to copy them."
	msgChooseMove      = "Choose where to move them."
	msgOneLevel        = "%s works on one level at a time — keep only one level."
	msgCommentTooLong  = "Use 2,000 characters or fewer."
	msgUndoOnlyDelete  = "Only a bulk delete can be undone."
	msgActionGone      = "This bulk action no longer exists."
	msgUndoTooLate     = "Undo is no longer available — restore them from the recycle bin"
	msgNoTargetPlan    = "%s works within one plan — select records of one plan"
	msgNoTargetYear    = "Move works within one year — select campaigns of plans of one year"
	msgNoTarget        = "No place to %s them to"
	notOfferedText     = "Names, IDs, money, calculated fields, Ongoing, text and % split fields are edited one record at a time."

	reasonGone           = "No longer exists"
	reasonAlreadySet     = "Already set — nothing to change"
	reasonOnlyActivities = "Only activities go through approval"
	reasonSubmitted      = "Already submitted"
	reasonApproved       = "Already approved"
	reasonKeepApproval   = "%s — left unchanged so it keeps its approval"
	reasonKeepApprovers  = "Submitted — left unchanged so it stays with its approvers"
	reasonFieldsNeeded   = "%d fields needed: %s"
	reasonOneFieldNeeded = "1 field needed: %s"

	effectTypesKept = "%d activities keep a type the model '%s' doesn't allow — change it before submitting"
	effectHidden    = "%s attributes hidden on %d campaigns — their values are kept"
	effectShown     = "%s attributes added on %d campaigns — fill them in before submitting"
	effectTotals    = "Totals and budgets of '%s' and '%s' update"

	verbChanged       = "changed"
	nounChange        = "change"
	pluralCampaigns   = "campaigns"
	pluralInitiatives = "initiatives"
	pluralActivities  = "activities"
)

var verbs = map[Kind]string{ //nolint:gochecknoglobals // the verb of each action: built once, only read
	ActionChangeOwner: verbChanged, ActionChangeField: verbChanged, ActionSubmit: "submitted", ActionCopy: "copied",
	ActionMove: "moved", ActionDelete: "deleted",
}

var nouns = map[Kind]string{ //nolint:gochecknoglobals // the word of each action in notes and messages
	ActionChangeOwner: nounChange, ActionChangeField: nounChange, ActionSubmit: "submit", ActionCopy: "copy",
	ActionMove: "move", ActionDelete: "delete",
}

// countsText names the counts by level: "2 campaigns and 10 activities", "1 activity"; "records" for none.
func countsText(c LevelCounts) string {
	var parts []string
	for _, p := range []struct {
		n              int
		singular, many string
	}{{c.Campaigns, "campaign", pluralCampaigns}, {c.Initiatives, "initiative", pluralInitiatives}, {c.Activities, "activity", pluralActivities}} {
		switch {
		case p.n == 1:
			parts = append(parts, "1 "+p.singular)
		case p.n > 1:
			parts = append(parts, fmt.Sprintf("%d %s", p.n, p.many))
		}
	}
	if len(parts) == 0 {
		return "0 records"
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
}

// noteOf is the note every activity-log entry of the run carries: "Bulk change of 20 activities (BLK-1001)".
func noteOf(action Kind, selected LevelCounts, id string) string {
	note := fmt.Sprintf("Bulk %s of %s", nouns[action], countsText(selected))
	if id == "" {
		return note
	}
	return note + " (" + id + ")"
}

func actionWord(action Kind) string {
	if action == ActionChangeField {
		return "Change field"
	}
	if action == ActionCopy {
		return "Copy"
	}
	return "Move"
}

func levelPlural(l Level) string {
	if l == LevelCampaign {
		return pluralCampaigns
	}
	if l == LevelInitiative {
		return pluralInitiatives
	}
	return pluralActivities
}
