package app

import (
	"context"
	"encoding/json"

	"mapp/internal/history"
	"mapp/internal/plans"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
)

// historyAccess implements history.Access (feature 014, D-238, D-304): who sees which activity-log and
// recycle-bin entries, and who may bring a deleted record back.
//
//   - Entries about planning records (plans and everything in them) follow the plan's visibility: the plan
//     from the entry (the plan itself, or the first record of its path), stored or in the recycle bin; the
//     entries of a plan deleted permanently are seen only with "View plans" for All markets.
//   - Entries about Settings, people, teams, approval workflows, roles and the access rules need "See the
//     whole activity log"; global budgets also "See global budgets" for All markets; GSAP imports and
//     transactions (015) also "Import GSAP extracts" (D-304). Entries of any other kind need "See the whole
//     activity log" too (fail closed): the delegations (017), exchange rates, rate imports and field help
//     (016) among them.
//   - Restoring needs to see the entry and the level's delete permission in the plan, or the permission that
//     manages an administration record. The developer sample records stay open to everyone.
type historyAccess struct {
	plans   plans.Repository
	history *history.Service
}

// The managing permission of each administration kind: what restoring one of its records needs.
var managedBy = map[string]authz.Permission{
	"model": authz.SettingsManage, "activityType": authz.SettingsManage, "shellAttribute": authz.SettingsManage,
	"shellAttributeOption": authz.SettingsManage, "financeBudget": authz.SettingsManage,
	"financeBudgetItem": authz.SettingsManage, "currency": authz.ExchangeRatesManage,
	"person": authz.PeopleManage, "team": authz.PeopleManage, "teamMembership": authz.PeopleManage,
	"approvalWorkflow": authz.WorkflowsManage, "role": authz.AccessManage, "accessRules": authz.AccessManage,
	"globalBudget": authz.GlobalBudgetsManage, "approvalDelegation": authz.WorkflowsManage,
	"exchangeRate": authz.ExchangeRatesManage, "rateImport": authz.ExchangeRatesManage, "fieldHelp": authz.SettingsManage,
	"financialImport": authz.FinanceImport, "financialTransaction": authz.FinanceImport,
}

// importedFinance are the kinds of feature 015's GSAP imports, which "Import GSAP extracts" also opens in the
// log (D-304).
var importedFinance = map[string]bool{"financialImport": true, "financialTransaction": true}

// deletedWith is the delete permission of each level of a plan: what restoring one of its records needs.
var deletedWith = map[string]authz.Permission{
	"plan": authz.PlansDelete, "campaign": authz.CampaignsDelete, "initiative": authz.CampaignsDelete,
	"activity": authz.ActivitiesDelete, "lineItem": authz.ActivitiesDelete, "budgetAllocation": authz.GlobalBudgetsManage,
}

// openToAll are the kinds of the developer sample pages, which every signed-in person uses (D-036).
var openToAll = map[string]bool{"sampleFolder": true, "sampleNote": true}

// The compiler checks here that historyAccess implements history.Access.
var _ history.Access = historyAccess{}

// CanSeeEntry reports whether the person acting may see the log entry e.
func (h historyAccess) CanSeeEntry(ctx context.Context, e history.Entry) bool {
	return h.canSee(ctx, e.RecordType, e.RecordID, e.Path, nil)
}

// CanSeeBinEntry reports whether the person acting may see the recycle-bin entry e.
func (h historyAccess) CanSeeBinEntry(ctx context.Context, e history.BinEntry) bool {
	return h.canSee(ctx, e.RecordType, e.RecordID, e.Path, e.Snapshots)
}

// CanRestore refuses an Undo or Restore of e that the person acting may not make: 404 when they cannot see
// it, as for a missing record, otherwise 403 forbidden.
func (h historyAccess) CanRestore(ctx context.Context, e history.BinEntry) error {
	g, restricted := authz.From(ctx)
	if !restricted || openToAll[e.RecordType] {
		return nil
	}
	if !h.canSee(ctx, e.RecordType, e.RecordID, e.Path, e.Snapshots) {
		return errs.NotFound(authz.NotFoundMessage(e.RecordTypeLabel))
	}
	if p, planning := deletedWith[e.RecordType]; planning {
		plan, _ := h.planOf(ctx, e.RecordType, e.RecordID, e.Path, e.Snapshots)
		return authz.RequireIn(ctx, p, plan)
	}
	if p, managed := managedBy[e.RecordType]; managed {
		return authz.Require(ctx, p)
	}
	if !g.IsAdministrator() { // a kind this adapter does not know: administrators only
		return errs.Forbidden(g.Refusal(authz.RecycleBinPurge, nil))
	}
	return nil
}

// canSee reports whether the person acting may see an entry about the record id of kind, whose ancestors
// are path; snapshots are the records of a recycle-bin entry (nil for a log entry).
func (h historyAccess) canSee(ctx context.Context, kind, id string, path []history.RecordRef, snapshots []history.Snapshot) bool {
	g, restricted := authz.From(ctx)
	switch {
	case !restricted || openToAll[kind]:
		return true
	case kind == "globalBudget":
		all, _ := g.Teams(authz.GlobalBudgetsView)
		return all || g.Has(authz.ActivityLogViewAll)
	case importedFinance[kind]:
		return g.Has(authz.FinanceImport) || g.Has(authz.ActivityLogViewAll)
	}
	if _, planning := deletedWith[kind]; !planning {
		return g.Has(authz.ActivityLogViewAll)
	}
	plan, found := h.planOf(ctx, kind, id, path, snapshots)
	if !found { // deleted permanently: All markets only
		all, _ := g.Teams(authz.PlansView)
		return all
	}
	return g.CanViewPlan(plan)
}

// planOf returns the access facts of the plan an entry is about: the plan itself or the first record of the
// path, as stored, as the entry's own snapshot, or as a snapshot in the recycle bin; false when the plan is
// nowhere any more.
func (h historyAccess) planOf(ctx context.Context, kind, id string, path []history.RecordRef, snapshots []history.Snapshot) (authz.PlanFacts, bool) {
	planID := id
	if kind != "plan" {
		if len(path) == 0 || path[0].Type != "plan" {
			return authz.PlanFacts{}, false
		}
		planID = path[0].ID
	}
	if p, found, err := h.plans.Get(ctx, planID); err == nil && found {
		return plans.AccessFacts(p), true
	}
	for _, s := range snapshots {
		if s.Kind == "plan" && s.ID == planID {
			return planSnapshotFacts(s)
		}
	}
	inBin, err := h.history.SnapshotsInBin(ctx, "plan")
	if err != nil {
		return authz.PlanFacts{}, false
	}
	for _, s := range inBin {
		if s.ID == planID {
			return planSnapshotFacts(s)
		}
	}
	return authz.PlanFacts{}, false
}

// planSnapshotFacts returns the access facts of a plan as the recycle bin keeps it.
func planSnapshotFacts(s history.Snapshot) (authz.PlanFacts, bool) {
	var p plans.Plan
	if err := json.Unmarshal(s.Data, &p); err != nil {
		return authz.PlanFacts{}, false
	}
	return plans.AccessFacts(p), true
}
