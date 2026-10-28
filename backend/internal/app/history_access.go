package app

import (
	"context"
	"encoding/json"

	"mapp/internal/history"
	"mapp/internal/plan"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/memo"
	"mapp/internal/stores"
)

// historyAccess implements history.Access (014, D-238, D-304): entries about planning records follow the plan's
// visibility; administration kinds need the whole activity log, GSAP imports and SWBS runs also finance.import,
// global budgets globalBudgets.view for All markets; any other kind the whole log (fail closed).
type historyAccess struct {
	plans            stores.PlanStore
	history          *history.Service
	purchaseRequests stores.PurchaseRequestStore
}

var managedBy = map[string]authz.Permission{
	"model": authz.SettingsManage, "activityType": authz.SettingsManage, "shellAttribute": authz.SettingsManage,
	"shellAttributeOption": authz.SettingsManage, "financeBudget": authz.SettingsManage,
	"financeBudgetItem": authz.SettingsManage, "currency": authz.ExchangeRatesManage,
	"person": authz.PeopleManage, "team": authz.PeopleManage, "teamMembership": authz.PeopleManage,
	"approvalWorkflow": authz.WorkflowsManage, "role": authz.AccessManage, "accessRules": authz.AccessManage,
	"globalBudget": authz.GlobalBudgetsManage, "approvalDelegation": authz.WorkflowsManage,
	"exchangeRate": authz.ExchangeRatesManage, "rateImport": authz.ExchangeRatesManage, "fieldHelp": authz.SettingsManage,
	kindFinancialImport: authz.FinanceImport, kindFinancialTransaction: authz.FinanceImport, kindSwbsRun: authz.FinanceImport,
}

const (
	kindFinancialImport      = "financialImport"
	kindFinancialTransaction = "financialTransaction"
	kindSwbsRun              = "swbsRun"
)

func isImportedFinance(kind string) bool {
	switch kind {
	case kindFinancialImport, kindFinancialTransaction, kindSwbsRun:
		return true
	}
	return false
}

var deletedWith = map[string]authz.Permission{
	"plan": authz.PlansDelete, "campaign": authz.CampaignsDelete, "initiative": authz.CampaignsDelete,
	"activity": authz.ActivitiesDelete, "lineItem": authz.ActivitiesDelete, "budgetAllocation": authz.GlobalBudgetsManage,
	kindPurchaseRequest: authz.PurchaseRequestsDelete, kindPurchaseRequestLine: authz.PurchaseRequestsEdit,
}

var openToAll = map[string]bool{"sampleFolder": true, "sampleNote": true}

var _ history.Access = historyAccess{}

func (h historyAccess) CanSeeEntry(ctx context.Context, e history.Entry) bool {
	return h.canSee(ctx, e.RecordType, e.RecordID, e.Path, nil)
}

func (h historyAccess) CanSeeBinEntry(ctx context.Context, e history.BinEntry) bool {
	return h.canSee(ctx, e.RecordType, e.RecordID, e.Path, e.Snapshots)
}

func (h historyAccess) CanRestore(ctx context.Context, e history.BinEntry) error {
	g, restricted := authz.From(ctx)
	if !restricted || openToAll[e.RecordType] {
		return nil
	}
	if !h.canSee(ctx, e.RecordType, e.RecordID, e.Path, e.Snapshots) {
		return errs.NotFound(authz.NotFoundMessage(e.RecordTypeLabel))
	}
	if p, planning := deletedWith[e.RecordType]; planning {
		pl, _ := h.planOf(ctx, e.RecordType, e.RecordID, e.Path, e.Snapshots)
		return authz.RequireIn(ctx, p, pl)
	}
	if p, managed := managedBy[e.RecordType]; managed {
		return authz.Require(ctx, p)
	}
	if !g.IsAdministrator() { // a kind this adapter does not know: administrators only
		return errs.Forbidden(g.Refusal(authz.RecycleBinPurge, nil))
	}
	return nil
}

func (h historyAccess) canSee(ctx context.Context, kind, id string, path []history.RecordRef, snapshots []history.Snapshot) bool {
	g, restricted := authz.From(ctx)
	switch {
	case !restricted || openToAll[kind]:
		return true
	case kind == "globalBudget":
		all, _ := g.Teams(authz.GlobalBudgetsView)
		return all || g.Has(authz.ActivityLogViewAll)
	case isImportedFinance(kind):
		return g.Has(authz.FinanceImport) || g.Has(authz.ActivityLogViewAll)
	}
	if _, planning := deletedWith[kind]; !planning {
		return g.Has(authz.ActivityLogViewAll)
	}
	p, found := h.planOf(ctx, kind, id, path, snapshots)
	if !found { // deleted permanently: All markets only
		all, _ := g.Teams(authz.PlansView)
		return all
	}
	return g.CanViewPlan(p)
}

func (h historyAccess) planOf(ctx context.Context, kind, id string, path []history.RecordRef, snapshots []history.Snapshot) (authz.PlanFacts, bool) {
	planID, found := h.planIDOf(ctx, entryRecord{kind: kind, id: id, path: path, snapshots: snapshots})
	if !found {
		return authz.PlanFacts{}, false
	}
	known, err := h.knownPlans(ctx)
	if err != nil {
		return authz.PlanFacts{}, false
	}
	if facts, stored := known.stored[planID]; stored {
		return facts, true
	}
	for _, s := range snapshots {
		if s.Kind == "plan" && s.ID == planID {
			return planSnapshotFacts(s)
		}
	}
	facts, inBin := known.inBin[planID]
	return facts, inBin
}

type knownPlans struct {
	stored, inBin map[string]authz.PlanFacts
}

func (h historyAccess) knownPlans(ctx context.Context) (knownPlans, error) {
	return memo.Get(ctx, memo.Derived+":historyPlans", func(ctx context.Context) (knownPlans, error) {
		known := knownPlans{stored: map[string]authz.PlanFacts{}, inBin: map[string]authz.PlanFacts{}}
		all, err := h.plans.List(ctx)
		if err != nil {
			return knownPlans{}, err
		}
		for _, p := range all {
			known.stored[p.ID] = plan.AccessFacts(p)
		}
		inBin, err := h.history.SnapshotsInBin(ctx, "plan")
		if err != nil {
			return knownPlans{}, err
		}
		for _, s := range inBin {
			if facts, ok := planSnapshotFacts(s); ok {
				known.inBin[s.ID] = facts
			}
		}
		return known, nil
	})
}

func planSnapshotFacts(s history.Snapshot) (authz.PlanFacts, bool) {
	var p plan.Plan
	if err := json.Unmarshal(s.Data, &p); err != nil {
		return authz.PlanFacts{}, false
	}
	return plan.AccessFacts(p), true
}
