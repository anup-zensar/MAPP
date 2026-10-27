package plan

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
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
	CampaignBudgets    money.Amount    // sum of the campaign budgets (in the plan currency, 016)
	PlannedCost        *money.Amount   // sum of the activities' planned costs (plan currency); nil when nothing below has one
	AmountsSet         int             // 016 FR-012: campaign and initiative budgets and activity planned costs that are set
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

// RegisterLevel adds a level below the plan. internal/app calls it while the API starts, top-down: campaign,
// initiative (005), activity (006), line item (007). It refuses a level without a name or with a name
// already registered, and a level without AddFacts or Copy; a refusal stops the API from starting.
func (s *Service) RegisterLevel(l Level) error {
	switch {
	case l.Name == "":
		return errors.New("plans: a level needs a name")
	case slices.ContainsFunc(s.levels, func(registered Level) bool { return registered.Name == l.Name }):
		return fmt.Errorf("plans: the level %q is already registered", l.Name)
	case l.AddFacts == nil || l.Copy == nil:
		return fmt.Errorf("plans: the level %q needs the functions AddFacts and Copy", l.Name)
	}
	s.levels = append(s.levels, l)
	return nil
}

// facts asks every level once for the facts of all the plans planIDs (SC-009: never once per plan). A plan
// below which no level has anything keeps zero Facts.
func (s *Service) facts(ctx context.Context, planIDs []string) (map[string]*Facts, error) {
	facts := make(map[string]*Facts, len(planIDs))
	for _, id := range planIDs {
		facts[id] = &Facts{}
	}
	for _, l := range s.levels {
		if err := l.AddFacts(ctx, planIDs, facts); err != nil {
			return nil, err
		}
	}
	return facts, nil
}
