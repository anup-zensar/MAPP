package app

import (
	"context"
	"encoding/json"

	"mapp/internal/history"
	"mapp/internal/purchaserequest"
	"mapp/internal/settings"
	"mapp/internal/stores"
)

// guardPurchaseRequests keeps what purchase requests draw on: no delete of a line item or a record
// above it, no planned cost below what is requested, no campaign moved to another plan.
func (w *wiring) guardPurchaseRequests(prs *purchaserequest.Service) {
	w.history.UseDeleteGuard(func(ctx context.Context, snapshots []history.Snapshot) error {
		var lineItemIDs []string
		for _, s := range snapshots {
			if s.Kind == lineItemKind {
				lineItemIDs = append(lineItemIDs, s.ID)
			}
		}
		return prs.RefuseDrawnOn(ctx, lineItemIDs)
	})
	w.lineItems.UsePurchaseRequests(prs)
	w.campaigns.UsePurchaseRequests(campaignPurchaseRequests{set: w.set, prs: prs})
}

type campaignPurchaseRequests struct {
	set stores.Set
	prs *purchaserequest.Service
}

func (x campaignPurchaseRequests) RefuseCampaignMove(ctx context.Context, campaignID string) error {
	initiatives, err := x.set.Campaigns.InitiativesOfCampaign(ctx, campaignID)
	if err != nil {
		return err
	}
	initiativeIDs := make([]string, 0, len(initiatives))
	for _, i := range initiatives {
		initiativeIDs = append(initiativeIDs, i.ID)
	}
	acts, err := x.set.Activities.ListByInitiatives(ctx, initiativeIDs)
	if err != nil {
		return err
	}
	activityIDs := make([]string, 0, len(acts))
	for _, a := range acts {
		activityIDs = append(activityIDs, a.ID)
	}
	items, err := x.set.LineItems.ListByActivities(ctx, activityIDs)
	if err != nil {
		return err
	}
	lineItemIDs := make([]string, 0, len(items))
	for _, li := range items {
		lineItemIDs = append(lineItemIDs, li.ID)
	}
	return x.prs.RefuseMove(ctx, lineItemIDs)
}

// purchaseRequestReferences is Settings' ReferenceSource of purchase requests and PR lines, stored and in the
// recycle bin: the Shell attribute values they use.
func purchaseRequestReferences(repo stores.PurchaseRequestStore, h *history.Service) settings.ReferenceSource {
	return func(ctx context.Context) ([]settings.Reference, error) {
		all, err := repo.All(ctx)
		if err != nil {
			return nil, err
		}
		var refs []settings.Reference
		for _, pr := range all {
			refs = append(refs, prReferences(pr, false)...)
		}
		inBin, err := h.SnapshotsInBin(ctx, kindPurchaseRequestLine)
		if err != nil {
			return nil, err
		}
		for _, s := range inBin {
			var l purchaserequest.Line
			if json.Unmarshal(s.Data, &l) == nil {
				refs = append(refs, lineReference(l, true))
			}
		}
		return refs, nil
	}
}

func prReferences(pr purchaserequest.PurchaseRequest, inBin bool) []settings.Reference {
	refs := []settings.Reference{{Record: history.RecordRef{Type: kindPurchaseRequest, ID: pr.ID, Name: pr.Name},
		InBin: inBin, Level: settings.LevelPurchaseRequest, PlanID: pr.PlanID, Attributes: pr.Attributes}}
	for _, l := range pr.Lines {
		refs = append(refs, lineReference(l, inBin))
	}
	return refs
}

func lineReference(l purchaserequest.Line, inBin bool) settings.Reference {
	return settings.Reference{Record: history.RecordRef{Type: kindPurchaseRequestLine, ID: l.ID, Name: l.ID},
		InBin: inBin, Level: settings.LevelPurchaseRequestLine, Attributes: l.Attributes}
}
