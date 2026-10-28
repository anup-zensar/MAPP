package dashboard

import (
	"time"

	"mapp/internal/allwork"
	"mapp/internal/approval"
	"mapp/internal/budget"
	"mapp/internal/platform/money"
	"mapp/internal/purchaserequest"
)

var (
	displayStatuses  = []string{statusCreated, statusSubmitted, statusDeclined, statusApproved, statusInProgress, statusComplete} //nolint:gochecknoglobals // the order of the contract's ActivityDisplayStatus
	purchaseStatuses = []purchaserequest.Status{                                                                                  //nolint:gochecknoglobals // the order of the contract's PurchaseRequestStatus
		purchaserequest.StatusCreated, purchaserequest.StatusSubmitted, purchaserequest.StatusInReview,
		purchaserequest.StatusDeclined, purchaserequest.StatusApproved, purchaserequest.StatusClosed,
	}
	completenessLevels = []allwork.Level{ //nolint:gochecknoglobals // the levels top-down
		allwork.LevelPlan, allwork.LevelCampaign, allwork.LevelInitiative, allwork.LevelActivity, allwork.LevelLineItem,
	}
)

// approvals counts the rounds of the activities and purchase requests on the board as the monitoring does (017).
func (b *board) approvals(rounds []approval.RoundSummary, purchases []purchaserequest.Summary) Approvals {
	subjects := map[string]bool{}
	for _, a := range b.byLevel[allwork.LevelActivity] {
		subjects[a.ID] = true
	}
	for _, pr := range purchases {
		subjects[pr.ID] = true
	}
	out := Approvals{ActivitiesByStatus: b.activitiesByStatus()}
	var seconds int64
	for _, r := range rounds {
		if !subjects[r.SubjectID] {
			continue
		}
		seconds += out.count(r)
	}
	out.AverageDecisionDays = approval.AverageDecisionDays(int64(out.Approved+out.Declined), seconds)
	return out
}

// count counts one round and returns the seconds it took when it was decided by a person.
func (a *Approvals) count(r approval.RoundSummary) int64 {
	switch {
	case r.Outcome == approval.OutcomeOpen:
		a.Waiting++
		if r.WaitingSince != nil && (a.OldestWaitingSince == nil || r.WaitingSince.Before(*a.OldestWaitingSince)) {
			since := *r.WaitingSince
			a.OldestWaitingSince = &since
		}
		return 0
	case r.AutoApproved:
		a.AutoApproved++
		return 0
	case r.Outcome == approval.OutcomeApproved:
		a.Approved++
	case r.Outcome == approval.OutcomeDeclined:
		a.Declined++
	default:
		return 0
	}
	return int64(r.EndedAt.Sub(r.SubmittedAt) / time.Second)
}

func (b *board) activitiesByStatus() []StatusCount {
	counts := map[string]int{}
	for _, a := range b.byLevel[allwork.LevelActivity] {
		counts[a.Status]++
	}
	out := make([]StatusCount, 0, len(displayStatuses))
	for _, status := range displayStatuses {
		out = append(out, StatusCount{Status: status, Count: counts[status]})
	}
	return out
}

func (b *board) purchaseRequests(s purchaserequest.Summaries, d money.Display) PurchaseRequests {
	amounts, counts := map[purchaserequest.Status]money.Amount{}, map[purchaserequest.Status]int{}
	var total money.Amount
	out := PurchaseRequests{Available: s.SwitchedOn, ByStatus: make([]PurchaseRequestStatus, 0, len(purchaseStatuses))}
	for _, pr := range s.Items {
		if !b.inView(pr.PlanID) {
			continue
		}
		counts[pr.Status]++
		amounts[pr.Status] = amounts[pr.Status].Add(pr.AmountUSD)
		total = total.Add(pr.AmountUSD)
		out.Count++
		if pr.WaitingForGSAP {
			out.WaitingForGSAP++
		}
	}
	for _, status := range purchaseStatuses {
		out.ByStatus = append(out.ByStatus, PurchaseRequestStatus{Status: string(status), Count: counts[status],
			Amount: shown(d, amounts[status])})
	}
	out.Amount = shown(d, total)
	return out
}

// completeness counts, level by level, the records with every needed field filled in (domain model §2.6).
func (b *board) completeness() []Completeness {
	out := make([]Completeness, 0, len(completenessLevels))
	for _, level := range completenessLevels {
		c := Completeness{Level: string(level)}
		for _, n := range b.byLevel[level] {
			c.Total++
			if n.Missing == 0 {
				c.Complete++
			}
		}
		c.Percent = percentOf(c.Complete, c.Total)
		out = append(out, c)
	}
	return out
}

func globalBudgets(page budget.Page, d money.Display) []GlobalBudget {
	out := make([]GlobalBudget, 0, len(page.Items))
	for _, g := range page.Items {
		f := g.Figures
		out = append(out, GlobalBudget{ID: g.ID, Name: g.Name, Amount: d.Money(f.Amount), Allocated: d.Money(f.Allocated),
			Unallocated: d.Money(f.Unallocated), PlannedCost: d.Money(f.PlannedCost), OverAllocated: f.OverAllocated})
	}
	return out
}
