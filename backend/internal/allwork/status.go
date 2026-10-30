package allwork

import "fmt"

// The status chips of All work (data model §9) as words, tones and lifecycle sort order (data model §7); the
// values come from the owning features (Node.Status, Timing, Swbs).

const (
	statusDraft        = "draft"
	statusPublished    = "published"
	statusNoActivities = "noActivities"
	statusInProgress   = "inProgress"
	statusAllApproved  = "allApproved"
	statusNotStarted   = "notStarted"
	statusRunning      = "running"
	statusFinished     = "finished"
	statusCreated      = "created"
	statusSubmitted    = "submitted"
	statusDeclined     = "declined"
	statusApproved     = "approved"
	statusComplete     = "complete"
)

type chip struct {
	label string
	tone  Tone
	order int
}

var planStatuses = map[string]chip{
	statusDraft:     {"Draft", ToneNeutral, 0},
	statusPublished: {"Published", TonePositive, 1},
}

var progressStatuses = map[string]chip{
	statusNoActivities: {"No activities", ToneNeutral, 0},
	statusInProgress:   {"Partly approved", ToneReview, 1},
	statusAllApproved:  {"All approved", TonePositive, 2},
}

var timingStatuses = map[string]chip{
	statusNotStarted: {"Not started", ToneNeutral, 0},
	statusRunning:    {"Running", ToneActive, 1},
	statusFinished:   {"Finished", ToneDone, 2},
}

var activityStatuses = map[string]chip{
	statusCreated:    {"Created", ToneNeutral, 0},
	statusSubmitted:  {"Submitted", ToneReview, 1},
	statusDeclined:   {"Declined", ToneProblem, 2},
	statusApproved:   {"Approved", TonePositive, 3},
	statusInProgress: {"In progress", ToneActive, 4},
	statusComplete:   {"Complete", ToneDone, 5},
}

var swbsStatuses = map[string]chip{
	"notRequested":       {"Not requested", ToneNeutral, 0},
	"waitingForApproval": {"Waiting for activity approval", ToneReview, 1},
	"readyToSend":        {"Ready to send to GSAP", TonePositive, 2},
	"received":           {"SWBS code received", ToneDone, 3},
}

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

// statusOf is nil for a node without a known status; a line item's status is its completeness (FR-011, D-118).
func statusOf(n *Node) *Status {
	if n.Level == LevelLineItem {
		return completenessOf(n.Missing)
	}
	s := chipOf(statusFamily(n.Level), n.Status)
	if s != nil && n.Status == statusInProgress && (n.Level == LevelCampaign || n.Level == LevelInitiative) {
		s.Detail = fmt.Sprintf("%d of %d activities approved", n.Approved, n.Total)
	}
	return s
}

func timingOf(n *Node) *Status {
	return chipOf(timingStatuses, n.Timing)
}

func swbsOf(n *Node) *Status {
	return chipOf(swbsStatuses, n.Swbs)
}

func completenessOf(missing int) *Status {
	switch missing {
	case 0:
		return &Status{Value: statusComplete, Label: "All filled in", Tone: TonePositive}
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
