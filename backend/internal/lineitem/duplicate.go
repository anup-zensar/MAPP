package lineitem

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

// Duplicate copies the line item id into its activity with a new ID, the name "<name> (copy)", " (copy 2)", …
// and no SWBS code (FR-029, 030 FR-005); the activity's status never changes (D-322).
func (s *Service) Duplicate(ctx context.Context, a actor.Actor, id string) (View, error) {
	var view View
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		li, found, err := s.repo.Get(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return s.gone(ctx, a, id)
		}
		f, err := s.facts(ctx, li.ActivityID)
		if err != nil {
			return err
		}
		siblings, err := s.repo.ListByActivity(ctx, li.ActivityID)
		if err != nil {
			return err
		}
		dup := copyOf(li)
		dup.Name = uniqueName(li.Name, copyNameSuffix, siblings, maxNameLength)
		if dup.ID, err = s.ids.Next(ctx, ids.PrefixLineItem); err != nil {
			return err
		}
		dup.Meta = record.NewMeta(a, s.now())
		if err = s.repo.Put(ctx, dup); err != nil {
			return err
		}
		if _, err = s.history.Log(ctx, a, history.LogInput{
			Kind: lineItemKind, RecordID: dup.ID, RecordName: dup.Name, Action: history.ActionCopied,
			Details: fmt.Sprintf(copiedDetails, li.Name, li.ID),
		}); err != nil {
			return err
		}
		view, err = s.shownView(ctx, a, f, dup)
		return err
	})
	if err != nil {
		return View{}, err
	}
	return view, nil
}

const copiedDetails = "Copied from '%s' (%s)"

func copyNameSuffix(n int) string {
	if n == 1 {
		return " (copy)"
	}
	return fmt.Sprintf(" (copy %d)", n)
}

// copyOf shares no list, map or pointer with li and leaves out the SWBS code: every copy path uses it.
func copyOf(li LineItem) LineItem {
	dup := li
	dup.Swbs = Swbs{}
	dup.Spread.Months = slices.Clone(li.Spread.Months)
	dup.Attributes = maps.Clone(li.Attributes)
	if li.ContractedAmount != nil {
		contracted := *li.ContractedAmount
		dup.ContractedAmount = &contracted
	}
	if li.FinanceBudgetYear != nil {
		year := *li.FinanceBudgetYear
		dup.FinanceBudgetYear = &year
	}
	return dup
}
