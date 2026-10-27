package app

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"mapp/internal/activity"
	"mapp/internal/approval"
	"mapp/internal/campaign"
	"mapp/internal/history"
	"mapp/internal/lineitem"
	"mapp/internal/people"
	"mapp/internal/plan"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/settings"
	"mapp/internal/stores"
)

// The adapters below tell Settings (002) and people (003) where their records are used (D-196): Settings'
// reference sources ("Used by", in-use deletes, impact statements) and people's plan source, usage counts
// and currencies. They read the other features inside the caller's transaction.

// planningRecords reads the records below the plans — campaigns, initiatives, activities, line items —
// stored and in the recycle bin.
type planningRecords struct {
	history    *history.Service
	campaigns  stores.CampaignStore
	activities stores.ActivityStore
	lineItems  stores.LineItemStore
}

// references lists a Settings reference for every campaign, initiative, activity and line item, stored and
// in the recycle bin (Settings' ReferenceSource of features 005, 006 and 007).
func (p planningRecords) references(ctx context.Context) ([]settings.Reference, error) {
	refs, err := p.stored(ctx)
	if err != nil {
		return nil, err
	}
	binned, err := p.inBin(ctx)
	if err != nil {
		return nil, err
	}
	return append(refs, binned...), nil
}

// stored lists the references of the stored records, each with its plan.
func (p planningRecords) stored(ctx context.Context) ([]settings.Reference, error) {
	campaignList, err := p.campaigns.ListCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	initiatives, err := p.campaigns.ListInitiatives(ctx)
	if err != nil {
		return nil, err
	}
	activityList, err := p.activities.List(ctx)
	if err != nil {
		return nil, err
	}
	items, err := p.lineItems.List(ctx)
	if err != nil {
		return nil, err
	}

	planOf := map[string]string{} // record id → plan id
	var refs []settings.Reference
	for _, c := range campaignList {
		planOf[c.ID] = c.PlanID
		refs = append(refs, campaignReference(c, false))
	}
	for _, i := range initiatives {
		planOf[i.ID] = planOf[i.CampaignID]
		refs = append(refs, initiativeReference(i, planOf[i.ID], false))
	}
	for _, a := range activityList {
		planOf[a.ID] = planOf[a.InitiativeID]
		refs = append(refs, activityReference(a, planOf[a.ID], false))
	}
	for _, li := range items {
		refs = append(refs, lineItemReference(li, planOf[li.ActivityID], false))
	}
	return refs, nil
}

// inBin lists the references of the records in the recycle bin. Everything in one bin entry belongs to the
// plan of its deleted record: the first record of the entry's path, or the deleted plan itself.
func (p planningRecords) inBin(ctx context.Context) ([]settings.Reference, error) {
	// Every record in the bin counts, whoever asks (014): entries undone or restored are left out.
	entries, err := p.history.ListBin(authz.Unrestricted(ctx), actor.System)
	if err != nil {
		return nil, err
	}
	var refs []settings.Reference
	for _, entry := range entries {
		planID := entry.RecordID
		if len(entry.Path) > 0 && entry.Path[0].Type == "plan" {
			planID = entry.Path[0].ID
		}
		for _, snapshot := range entry.Snapshots {
			ref, known, err := snapshotReference(snapshot, planID)
			if err != nil {
				return nil, err
			}
			if known {
				refs = append(refs, ref)
			}
		}
	}
	return refs, nil
}

// snapshotReference returns the reference of a record of the recycle bin, and false for kinds that are not
// planning records below the plan (plans report themselves).
func snapshotReference(s history.Snapshot, planID string) (settings.Reference, bool, error) {
	decode := func(v any) error {
		if err := json.Unmarshal(s.Data, v); err != nil {
			return fmt.Errorf("app: read %s %s in the recycle bin: %w", s.Kind, s.ID, err)
		}
		return nil
	}
	switch s.Kind {
	case "campaign":
		var c campaign.Campaign
		err := decode(&c)
		return campaignReference(c, true), err == nil, err
	case "initiative":
		var i campaign.Initiative
		err := decode(&i)
		return initiativeReference(i, planID, true), err == nil, err
	case "activity":
		var a activity.Activity
		err := decode(&a)
		return activityReference(a, planID, true), err == nil, err
	case "lineItem":
		var li lineitem.LineItem
		err := decode(&li)
		return lineItemReference(li, planID, true), err == nil, err
	}
	return settings.Reference{}, false, nil
}

func campaignReference(c campaign.Campaign, inBin bool) settings.Reference {
	return settings.Reference{
		Record: history.RecordRef{Type: "campaign", ID: c.ID, Name: c.Name}, InBin: inBin,
		Level: settings.LevelCampaign, PlanID: c.PlanID, Attributes: c.Attributes,
	}
}

func initiativeReference(i campaign.Initiative, planID string, inBin bool) settings.Reference {
	return settings.Reference{
		Record: history.RecordRef{Type: "initiative", ID: i.ID, Name: i.Name}, InBin: inBin,
		Level: settings.LevelInitiative, PlanID: planID, Attributes: i.Attributes,
	}
}

func activityReference(a activity.Activity, planID string, inBin bool) settings.Reference {
	return settings.Reference{
		Record: history.RecordRef{Type: "activity", ID: a.ID, Name: a.Name}, InBin: inBin,
		Level: settings.LevelActivity, PlanID: planID, ActivityID: a.ID, Attributes: a.Attributes,
		ActivityTypeID: a.ActivityTypeID,
	}
}

func lineItemReference(li lineitem.LineItem, planID string, inBin bool) settings.Reference {
	planned := li.PlannedCost
	return settings.Reference{
		Record: history.RecordRef{Type: "lineItem", ID: li.ID, Name: li.Name}, InBin: inBin,
		Level: settings.LevelLineItem, PlanID: planID, ActivityID: li.ActivityID, Attributes: li.Attributes,
		Currency: li.Currency, PlannedCost: &planned, FinanceBudgetItemID: li.FinanceBudgetItemID,
	}
}

// planReferences is Settings' ReferenceSource of plans (004): every plan, stored and in the recycle bin,
// with its model, default finance budget, Shell attribute values, and its plan currency with its budget in it
// (016: "Plan currency of n plans", FR-010; the rate impact, FR-005).
func planReferences(svc *plan.Service) settings.ReferenceSource {
	return func(ctx context.Context) ([]settings.Reference, error) {
		list, err := svc.References(ctx, actor.System)
		if err != nil {
			return nil, err
		}
		refs := make([]settings.Reference, 0, len(list))
		for _, p := range list {
			refs = append(refs, settings.Reference{
				Record: history.RecordRef{Type: "plan", ID: p.PlanID, Name: p.Name}, InBin: p.InRecycleBin,
				Level: settings.LevelPlan, PlanID: p.PlanID, Attributes: p.Attributes, ModelID: p.ModelID,
				FinanceBudgetID: p.DefaultFinanceBudgetID, Currency: p.Currency, Budget: p.Budget, // 016 FR-005, FR-010
			})
		}
		return refs, nil
	}
}

// personReferences is Settings' ReferenceSource of people (003): the display currency of each person.
func personReferences(svc *people.Service) settings.ReferenceSource {
	return func(ctx context.Context) ([]settings.Reference, error) {
		byCurrency, err := svc.PeopleByDisplayCurrency(ctx)
		if err != nil {
			return nil, err
		}
		var refs []settings.Reference
		for _, code := range slices.Sorted(maps.Keys(byCurrency)) { // the same order every time
			for _, id := range byCurrency[code] {
				refs = append(refs, settings.Reference{Record: history.RecordRef{Type: "person", ID: id}, Currency: code})
			}
		}
		return refs, nil
	}
}

// planUses is people's PlanSource (004): every plan, stored and in the recycle bin, with its owner and
// teams.
func planUses(svc *plan.Service) people.PlanSource {
	return func(ctx context.Context) ([]people.PlanUse, error) {
		list, err := svc.References(ctx, actor.System)
		if err != nil {
			return nil, err
		}
		uses := make([]people.PlanUse, 0, len(list))
		for _, p := range list {
			uses = append(uses, people.PlanUse{
				ID: p.PlanID, Name: p.Name, OwnerID: p.OwnerID, TeamIDs: p.TeamIDs, InRecycleBin: p.InRecycleBin,
			})
		}
		return uses, nil
	}
}

const ownerRole = "owner"

// personUsages are the counts of what a person owns or approves below the plans (005, 006, 007, 008), over
// the stored records: records in the recycle bin are not stored, so they are not counted (FR-004).
func personUsages(campaignsRepo stores.CampaignStore, activitiesRepo stores.ActivityStore,
	lineItemsRepo stores.LineItemStore, approvalsRepo stores.ApprovalStore) []people.PersonUsage {
	return []people.PersonUsage{
		{Role: ownerRole, RecordType: "campaign", Singular: "campaign", Plural: "campaigns", Count: campaignsRepo.CountCampaignsOwnedBy},
		{Role: ownerRole, RecordType: "initiative", Singular: "initiative", Plural: "initiatives",
			Count: campaignsRepo.CountInitiativesOwnedBy},
		{Role: ownerRole, RecordType: "activity", Singular: "activity", Plural: "activities", Count: activitiesRepo.CountOwnedBy},
		{Role: ownerRole, RecordType: "lineItem", Singular: "line item", Plural: "line items", Count: lineItemsRepo.CountOwnedBy},
		{Role: "approver", RecordType: "approvalWorkflow", Singular: "approval workflow", Plural: "approval workflows",
			Count: func(ctx context.Context, personID string) (int, error) {
				list, err := approvalsRepo.ListWorkflows(ctx)
				return countWhere(list, err, func(w approval.Workflow) bool {
					return slices.ContainsFunc(w.Levels, func(l approval.Level) bool { return slices.Contains(l.ApproverIDs, personID) })
				})
			}},
	}
}

// countWhere counts the records of list that keep says yes to; err is the error of reading the list.
func countWhere[T any](list []T, err error, keep func(T) bool) (int, error) {
	if err != nil {
		return 0, err
	}
	n := 0
	for _, record := range list {
		if keep(record) {
			n++
		}
	}
	return n, nil
}

// peopleCurrencies is people's CurrencySource over Settings' currencies (002): every currency, active or
// not, with its current rate.
func peopleCurrencies(svc *settings.Service) people.CurrencySource {
	return func(ctx context.Context) ([]people.Currency, error) {
		list, err := svc.Currencies(ctx)
		if err != nil {
			return nil, err
		}
		currencies := make([]people.Currency, 0, len(list))
		for _, c := range list {
			currencies = append(currencies, people.Currency{Code: c.Code, Name: c.Name, Rate: c.Rate.String(), Active: c.Active})
		}
		return currencies, nil
	}
}
