package allwork

import "fmt"

// The status chips of All work (data model §9): every status a row shows, as a word and a tone, and the
// lifecycle order in which statuses sort (data model §7). The values come from the owning features
// (Node.Status, Timing, Swbs); All work only words them.

// chip is one status value as a chip, with its place in the lifecycle order.
type chip struct {
	label string
	tone  Tone
	order int
}

// planStatuses are the plan statuses (domain model §2.1).
var planStatuses = map[string]chip{
	"draft":     {"Draft", ToneNeutral, 0},
	"published": {"Published", TonePositive, 1},
}

// progressStatuses are the Approval progress values of campaigns and initiatives (domain model §2.2).
var progressStatuses = map[string]chip{
	"noActivities": {"No activities", ToneNeutral, 0},
	"inProgress":   {"In progress", ToneReview, 1},
	"allApproved":  {"All approved", TonePositive, 2},
}

// timingStatuses are the Timing values of campaigns and initiatives (D-091).
var timingStatuses = map[string]chip{
	"notStarted": {"Not started", ToneNeutral, 0},
	"running":    {"Running", ToneActive, 1},
	"finished":   {"Finished", ToneDone, 2},
}

// activityStatuses are the activity display statuses (domain model §2.3).
var activityStatuses = map[string]chip{
	"created":    {"Created", ToneNeutral, 0},
	"submitted":  {"Submitted", ToneReview, 1},
	"declined":   {"Declined", ToneProblem, 2},
	"approved":   {"Approved", TonePositive, 3},
	"inProgress": {"In progress", ToneActive, 4},
	"complete":   {"Complete", ToneDone, 5},
}

// swbsStatuses are the SWBS request statuses of line items (007 data model §3.5).
var swbsStatuses = map[string]chip{
	"notRequested":       {"Not requested", ToneNeutral, 0},
	"waitingForApproval": {"Waiting for activity approval", ToneReview, 1},
	"readyToSend":        {"Ready to send to GSAP", TonePositive, 2},
}

// statusFamily returns the status values of a level's Status.
func statusFamily(level Level) map[string]chip {
	switch level {
	case LevelPlan:
		return planStatuses
	case LevelCampaign, LevelInitiative:
		return progressStatuses
	case LevelActivity:
		return activityStatuses
	}
	return nil
}

// statusOf returns the status chip of a node (FR-011, D-118): plan status, Approval progress with "n of m
// activities approved", activity display status, or a line item's completeness. Nil when the node has no
// known status.
func statusOf(n *Node) *Status {
	if n.Level == LevelLineItem {
		return completenessOf(n.Missing)
	}
	s := chipOf(statusFamily(n.Level), n.Status)
	if s != nil && n.Status == "inProgress" && (n.Level == LevelCampaign || n.Level == LevelInitiative) {
		s.Detail = fmt.Sprintf("%d of %d activities approved", n.Approved, n.Total)
	}
	return s
}

// timingOf returns the Timing chip of a campaign or initiative, nil without dates.
func timingOf(n *Node) *Status {
	return chipOf(timingStatuses, n.Timing)
}

// swbsOf returns the SWBS request chip of a line item.
func swbsOf(n *Node) *Status {
	return chipOf(swbsStatuses, n.Swbs)
}

// completenessOf returns the completeness chip for missing own needed fields: "All filled in" or "n fields
// needed".
func completenessOf(missing int) *Status {
	switch missing {
	case 0:
		return &Status{Value: "complete", Label: "All filled in", Tone: TonePositive}
	case 1:
		return &Status{Value: "incomplete", Label: "1 field needed", Tone: ToneProblem}
	}
	return &Status{Value: "incomplete", Label: fmt.Sprintf("%d fields needed", missing), Tone: ToneProblem}
}

// chipOf returns the chip of value in family, nil when the value is not one of them.
func chipOf(family map[string]chip, value string) *Status {
	c, known := family[value]
	if !known {
		return nil
	}
	return &Status{Value: value, Label: c.label, Tone: c.tone}
}

// levelWords are the levels as people read them: the Level column, messages and exports.
var levelWords = map[Level]string{
	LevelPlan:       "Plan",
	LevelCampaign:   "Campaign",
	LevelInitiative: "Initiative",
	LevelActivity:   "Activity",
	LevelLineItem:   "Line item",
	LevelUnassigned: "Unassigned planned cost",
}
