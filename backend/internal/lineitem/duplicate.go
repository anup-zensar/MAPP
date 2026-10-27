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

// Duplicate copies the line item id into its activity with every value, including the spread, a new ID and
// the name "<name> (copy)", " (copy 2)", … (FR-029, R13: there are no GSAP or finance values to leave out in
// this phase). It counts as adding a line item: on a Submitted or Approved activity only with
// in.RestartApproval (FR-031). One "copied" log entry names the original.
func (s *Service) Duplicate(ctx context.Context, a actor.Actor, id string, in DuplicateInput) (View, error) { //nolint:gocyclo // 11: one Write whose every step returns its error.
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
		if err = confirmRestart(nil, f, in.RestartApproval); err != nil {
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
		if f, err = s.restart(ctx, a, f, fmt.Sprintf(addedReason, dup.Name, dup.ID, dup.PlannedCost.Format(), dup.Currency)); err != nil {
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

// copiedDetails are the details of the "copied" log entry of a duplicate.
const copiedDetails = "Copied from '%s' (%s)"

// copyNameSuffix is the suffix of the n-th copy's name: " (copy)", " (copy 2)" and so on.
func copyNameSuffix(n int) string {
	if n == 1 {
		return " (copy)"
	}
	return fmt.Sprintf(" (copy %d)", n)
}

// copyOf returns a copy of li that shares no list, map or pointer with it.
func copyOf(li LineItem) LineItem {
	dup := li
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
