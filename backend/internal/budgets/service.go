package budgets

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
	"mapp/internal/platform/store"
)

// Service holds the rules of global budgets and their allocations (feature 014, FR-029 – FR-036).
type Service struct {
	tx        store.Transactor
	repo      Store
	ids       *ids.Generator
	history   *history.Service
	plans     Plans
	directory Directory
	rates     Rates
	access    Access
	now       clock.Clock
}

// NewService returns the global budgets service. It keeps global budgets and allocations in repo, gives them
// readable IDs, logs their changes and deletes them, works in transactions and reads the time with tools,
// and reads plans, people and rates and asks the access rules through deps.
// Without deps.Access every action is allowed (internal calls, the seed, tests).
func NewService(tools history.Tools, repo Store, deps Deps) *Service {
	access := deps.Access
	if access == nil {
		access = unrestricted{}
	}
	return &Service{
		tx: tools.Tx, repo: repo, ids: tools.IDs, history: tools.Log, plans: deps.Plans, directory: deps.Directory,
		rates: deps.Rates, access: access, now: tools.Now,
	}
}

// List returns the global budgets of year (every year when year is nil), by name, with their four totals,
// the years that have one, newest first, and whether the person may create one (FR-036). The totals count
// every allocation, also those to plans the person cannot see (FR-032).
func (s *Service) List(ctx context.Context, a actor.Actor, year *int) (Page, error) {
	page := Page{Items: []Summary{}, Years: []int{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		all, err := s.repo.ListBudgets(ctx)
		if err != nil {
			return err
		}
		chosen := []GlobalBudget{}
		for _, b := range all {
			if !slices.Contains(page.Years, b.Year) {
				page.Years = append(page.Years, b.Year)
			}
			if year == nil || b.Year == *year {
				chosen = append(chosen, b)
			}
		}
		slices.SortFunc(page.Years, func(x, y int) int { return cmp.Compare(y, x) })
		slices.SortFunc(chosen, byName)
		f, err := s.newFigurer(ctx)
		if err != nil {
			return err
		}
		for _, b := range chosen {
			allocations, err := s.repo.AllocationsOf(ctx, b.ID)
			if err != nil {
				return err
			}
			list, err := f.allocated(ctx, allocations)
			if err != nil {
				return err
			}
			owner, err := s.personRef(ctx, b.OwnerID)
			if err != nil {
				return err
			}
			page.Items = append(page.Items, Summary{
				ID: b.ID, Name: b.Name, Year: b.Year, Owner: owner, Figures: figuresOf(b.Amount, list),
				AllocationCount: len(allocations),
			})
		}
		page.Total = len(page.Items)
		page.Create, err = availabilityOf(s.access.ManageAll(ctx))
		return err
	})
	if err != nil {
		return Page{}, err
	}
	return page, nil
}

// byName orders global budgets by name ignoring letter case, then by id.
func byName(x, y GlobalBudget) int {
	return cmp.Or(strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name)), strings.Compare(x.ID, y.ID))
}

// Get returns the global budget id with its figures, the allocations to plans the person can see, the
// others added up as Other markets, and the visible plans of its year without an allocation (FR-032, FR-033).
func (s *Service) Get(ctx context.Context, a actor.Actor, id string) (GlobalBudgetView, error) {
	var view GlobalBudgetView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		b, err := s.stored(ctx, id)
		if err != nil {
			return err
		}
		view, err = s.view(ctx, b)
		return err
	})
	if err != nil {
		return GlobalBudgetView{}, err
	}
	return view, nil
}

// Create creates a global budget from the whole form in one Write (FR-029): it needs globalBudgets.manage
// for All markets; every problem is reported at once and nothing is saved while there is one; otherwise it
// gets a new permanent ID and a "created" log entry.
func (s *Service) Create(ctx context.Context, a actor.Actor, in GlobalBudgetInput) (GlobalBudgetView, error) {
	if err := s.access.ManageAll(ctx); err != nil {
		return GlobalBudgetView{}, err
	}
	var view GlobalBudgetView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		var problems errs.Problems
		b := GlobalBudget{
			Name: strings.TrimSpace(in.Name), OwnerID: strings.TrimSpace(in.OwnerID),
			Description: strings.TrimSpace(in.Description),
		}
		if err := s.checkName(ctx, b.Name, "", &problems); err != nil {
			return err
		}
		b.Year = s.checkYear(in.Year, &problems)
		b.Amount, _ = checkAmount(in.Amount, "USD", msgAmountInUSD, "amount", &problems)
		if err := s.checkOwner(ctx, b.OwnerID, &problems); err != nil {
			return err
		}
		checkDescription(b.Description, &problems)
		if err := problems.Err(); err != nil {
			return err
		}
		var err error
		if b.ID, err = s.ids.Next(ctx, PrefixGlobalBudget); err != nil {
			return err
		}
		b.Meta = record.NewMeta(a, s.now())
		if err := s.repo.PutBudget(ctx, b); err != nil {
			return err
		}
		if _, err := s.history.Log(ctx, a, history.LogInput{
			Kind: kindBudget, RecordID: b.ID, RecordName: b.Name, Action: history.ActionCreated,
		}); err != nil {
			return err
		}
		view, err = s.view(ctx, b)
		return err
	})
	if err != nil {
		return GlobalBudgetView{}, err
	}
	return view, nil
}

// Update saves the changed fields of the global budget id from the version the person started from (FR-029,
// D-203): name, amount, owner and description; the year never changes. It needs globalBudgets.manage for All
// markets. Every problem is reported at once, a missing version among them; a save from an out-of-date
// version is refused naming who changed it. An owner the budget already has may stay although they lost the
// role. A save that changes nothing creates no version and no log entry; otherwise one "changed" entry lists
// every changed field.
func (s *Service) Update(ctx context.Context, a actor.Actor, id string, update GlobalBudgetUpdate) (GlobalBudgetView, error) {
	var view GlobalBudgetView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		stored, err := s.stored(ctx, id)
		if err != nil {
			return err
		}
		if err := s.access.ManageAll(ctx); err != nil {
			return err
		}
		var problems errs.Problems
		changed := stored
		if update.Name != nil {
			changed.Name = strings.TrimSpace(*update.Name)
			if err := s.checkName(ctx, changed.Name, stored.ID, &problems); err != nil {
				return err
			}
		}
		if update.Amount != nil {
			changed.Amount, _ = checkAmount(update.Amount, "USD", msgAmountInUSD, "amount", &problems)
		}
		if update.OwnerID != nil {
			changed.OwnerID = strings.TrimSpace(*update.OwnerID)
			if changed.OwnerID != stored.OwnerID {
				if err := s.checkOwner(ctx, changed.OwnerID, &problems); err != nil {
					return err
				}
			}
		}
		if update.Description != nil {
			changed.Description = strings.TrimSpace(*update.Description)
			checkDescription(changed.Description, &problems)
		}
		checkVersion(update.Version, "version", &problems)
		if err := problems.Err(); err != nil {
			return err
		}
		if err := stored.CheckVersion(*update.Version, kindBudgetSingular); err != nil {
			return err
		}
		changes, err := history.Diff(stored, changed)
		if err != nil {
			return err
		}
		if len(changes) == 0 {
			view, err = s.view(ctx, stored)
			return err
		}
		changed.Touch(a, s.now())
		if err := s.repo.PutBudget(ctx, changed); err != nil {
			return err
		}
		if _, err := s.history.Log(ctx, a, history.LogInput{
			Kind: kindBudget, RecordID: changed.ID, RecordName: changed.Name, Action: history.ActionChanged, Changes: changes,
		}); err != nil {
			return err
		}
		view, err = s.view(ctx, changed)
		return err
	})
	if err != nil {
		return GlobalBudgetView{}, err
	}
	return view, nil
}

// Delete moves the global budget id to the recycle bin, in one Write, so it can be undone (FR-035). It needs
// globalBudgets.manage for All markets, and is refused with 409 in_use while the global budget has
// allocations (message 30).
func (s *Service) Delete(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		b, err := s.stored(ctx, id)
		if err != nil {
			return err
		}
		if err := s.access.ManageAll(ctx); err != nil {
			return err
		}
		allocations, err := s.repo.AllocationsOf(ctx, id)
		if err != nil {
			return err
		}
		if len(allocations) > 0 {
			return errs.InUse(inUseMessage(b.Name, len(allocations)))
		}
		entry, err = s.history.MoveToBin(ctx, a, kindBudget, id)
		return err
	})
	if err != nil {
		return history.BinEntry{}, err
	}
	return entry, nil
}

// stored reads the global budget id inside the caller's transaction; one that does not exist is not_found.
func (s *Service) stored(ctx context.Context, id string) (GlobalBudget, error) {
	b, found, err := s.repo.GetBudget(ctx, id)
	if err != nil {
		return GlobalBudget{}, err
	}
	if !found {
		return GlobalBudget{}, errs.NotFound(msgBudgetGone)
	}
	return b, nil
}

// availabilityOf turns the answer of an access check into an Availability: available when it allowed the
// action, otherwise not available with the refusal's message as the reason. An error that is not a refusal
// is a failure, returned as it is.
func availabilityOf(err error) (Availability, error) {
	if err == nil {
		return available, nil
	}
	var refusal *errs.Error
	if errors.As(err, &refusal) {
		return Availability{Reason: refusal.Message}, nil
	}
	return Availability{}, err
}
