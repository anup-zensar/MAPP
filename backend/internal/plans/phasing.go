package plans

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/rollup"
)

// SetBudgetPhasing saves the phasing of the plan id's budget (016 FR-020 – FR-023, D-265): Not phased,
// Evenly or Custom months of the plan's year in the plan currency, from the version the person started at.
// It may change whatever the plan's status — a Published plan stays Published — and logs the method and
// every changed month ("Budget by month: Mar 2027 50,000.00 → 60,000.00 GBP"). A Custom phasing that does
// not add up to the budget is saved; it is a needed-to-publish problem (FR-022). A save that changes nothing
// writes no version and no log entry.
func (s *Service) SetBudgetPhasing(ctx context.Context, a actor.Actor, id string, in rollup.PhasingInput) (PlanView, error) {
	var view PlanView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		p, err := s.stored(ctx, id)
		if err != nil {
			return err
		}
		phasing, problems := rollup.ParsePhasing(in, p.Budget, p.Year)
		if in.Version == nil || *in.Version < 1 {
			problems.Add("version", msgReload)
		}
		if err := problems.Err(); err != nil {
			return err
		}
		if err := p.CheckVersion(*in.Version, kindSingular); err != nil {
			return err
		}
		changes := rollup.PhasingChanges(p.BudgetPhasing, phasing, p.Budget, p.Budget, p.Year, p.Currency)
		if len(changes) > 0 {
			p.BudgetPhasing = phasing
			p.Touch(a, s.now())
			if err := s.repo.Put(ctx, p); err != nil {
				return err
			}
			if _, err := s.history.Log(ctx, a, history.LogInput{
				Kind: kindName, RecordID: p.ID, RecordName: p.Name, Action: history.ActionChanged, Changes: changes,
			}); err != nil {
				return err
			}
		}
		view, err = s.view(ctx, a, p)
		return err
	})
	if err != nil {
		return PlanView{}, err
	}
	return view, nil
}
