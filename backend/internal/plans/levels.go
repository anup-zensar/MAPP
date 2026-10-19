package plans

import (
	"context"
	"time"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
	"northstar/internal/settings"
)

// The level registry (004 plan, "Levels below the plan"; D-195): the features below the plan — campaign and
// initiative (005), activity (006), line item (007) — each register a Level with the plans service, top-down,
// through an adapter in internal/app. Plans then gets its figures, completeness, copies and delete counts
// without importing them. The service method RegisterLevel is feature 004's.

// Level is one level below the plan, as it registers itself.
type Level struct {
	// Name is the level: "campaign", "initiative", "activity" or "lineItem".
	Name string
	// AddFacts adds this level's figures for every plan in planIDs to facts[planID] (the entries exist).
	// Required. It runs inside the caller's transaction.
	AddFacts func(ctx context.Context, planIDs []string, facts map[string]*Facts) error
	// Copy copies this level's records of job.SourcePlanID whose parent's old ID is in job.NewIDs (R13),
	// adding their old → new IDs to job.NewIDs and their number to job.Counts[Name]. It writes no log
	// entries. Required; a level copied by the level above it does nothing.
	Copy func(ctx context.Context, job *CopyJob) error
	// ActivitiesWithTypeNotAllowed lists the activities of the plan whose type is not in allowedTypeIDs,
	// ordered by path. Optional (006).
	ActivitiesWithTypeNotAllowed func(ctx context.Context, planID string, allowedTypeIDs []string) ([]errs.RecordRef, error)
}

// Facts are what the levels below report about one plan. Until a level registers, every plan has zero
// Facts: counts 0, campaign budgets 0.00, planned cost nil ("—"), nothing needing attention.
type Facts struct {
	Campaigns          int
	Initiatives        int
	Activities         int
	ApprovedActivities int // Approved, In progress or Complete
	LineItems          int
	CampaignBudgets    money.Amount    // sum of the campaign budgets (USD)
	PlannedCost        *money.Amount   // sum of the activities' planned costs (USD); nil when nothing below has one
	ActivityGoals      GoalTotals      // sums of every activity goal: INQ, MQL, revenue in USD
	NeedsAttention     []AttentionItem // every record below that is not complete, top-down, then by name
}

// AttentionItem is a record below the plan that is not complete, with its missing fields (contract
// AttentionItem). BlocksPublish is true for campaigns and initiatives (005), whose missing fields stop
// the plan from being published; it is not part of the API answer.
type AttentionItem struct {
	Record        errs.RecordRef          `json:"record"` // {type: "campaign", id: "C-1001", name: "Retail growth"}
	Missing       []settings.MissingField `json:"missing"`
	BlocksPublish bool                    `json:"-"`
}

// CopyJob is one plan copy as it goes down the levels (D-195).
type CopyJob struct {
	Actor        actor.Actor
	Now          time.Time
	SourcePlanID string
	TargetPlanID string
	Years        int               // 0 same year, 1 next year: every copied date + Years (date.AddYears)
	NewIDs       map[string]string // old ID → new ID of every record copied so far, the plan first
	Counts       map[string]int    // records copied per level name, for the log details
}
