package plans

import (
	"context"
	"encoding/json"
	"maps"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
)

// References lists what every plan uses, the stored ones and those in the recycle bin, inside the
// caller's transaction: for the "Used by" counts of Settings (002) and of people and teams (003).
func (s *Service) References(ctx context.Context, a actor.Actor) ([]References, error) {
	var refs []References
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		stored, err := s.repo.List(ctx)
		if err != nil {
			return err
		}
		refs = make([]References, 0, len(stored))
		for _, p := range stored {
			refs = append(refs, referencesOf(p, false))
		}
		bin, err := s.history.ListBin(authz.Unrestricted(ctx), a) // every plan in the bin, whoever asks (014)
		if err != nil {
			return err
		}
		for _, entry := range bin {
			for _, snapshot := range entry.Snapshots {
				if snapshot.Kind != kindName {
					continue
				}
				r, err := ReferencesFromSnapshot(snapshot.Data)
				if err != nil {
					return err
				}
				refs = append(refs, r)
			}
		}
		return nil
	})
	return refs, err
}

// ReferencesFromSnapshot returns what the plan of a recycle-bin snapshot uses (the snapshot's Data).
func ReferencesFromSnapshot(data json.RawMessage) (References, error) {
	var p Plan
	if err := json.Unmarshal(data, &p); err != nil {
		return References{}, err
	}
	return referencesOf(p, true), nil
}

// referencesOf returns what the plan p uses; inBin says whether it is in the recycle bin.
func referencesOf(p Plan, inBin bool) References {
	p = p.Normalised() // a snapshot of before 016 has no currency
	refs := References{
		PlanID: p.ID, Name: p.Name, ModelID: p.ModelID, OwnerID: p.OwnerID, DefaultFinanceBudgetID: p.DefaultFinanceBudgetID,
		TeamIDs: slices.Clone(p.TeamIDs), Attributes: maps.Clone(p.Attributes), InRecycleBin: inBin, Currency: p.Currency,
	}
	if p.Budget != nil {
		budget := *p.Budget
		refs.Budget = &budget
	}
	return refs
}
