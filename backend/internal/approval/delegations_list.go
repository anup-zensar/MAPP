package approval

import (
	"cmp"
	"context"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
)

// The scopes of the delegation list (GET /api/v1/approval-delegations?scope=).
const (
	ScopeMine = "mine"
	ScopeAll  = "all"
)

// ListDelegations returns the delegations a sees (017 FR-024): with scope mine (the default) those where a
// is the delegating person or the delegate; with scope all everyone's, which needs workflows.manage (403
// otherwise). Current and planned ones come first by first day, then ended and cancelled ones, latest last
// day first. Each says whether a may change it; the page says whether a may manage other people's.
func (s *Service) ListDelegations(ctx context.Context, a actor.Actor, asked string) (DelegationPage, error) {
	scope, known := listScope(asked)
	if !known {
		return DelegationPage{}, errs.Validation(errs.FieldError{Field: "scope", Message: chooseScopeMessage})
	}
	var page DelegationPage
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		manage, err := s.manageOthers(ctx, a, scope)
		if err != nil {
			return err
		}
		all, err := s.repo.ListDelegations(ctx)
		if err != nil {
			return err
		}
		d, err := s.directory(ctx)
		if err != nil {
			return err
		}
		today := date.Today(s.now())
		list := visibleTo(all, a, scope)
		slices.SortStableFunc(list, func(x, y Delegation) int { return compareDelegations(x, y, today) })
		page = DelegationPage{Items: make([]DelegationView, 0, len(list)), Total: len(list), CanManageOthers: manage}
		for _, dl := range list {
			page.Items = append(page.Items, delegationView(dl, d, a, manage, today))
		}
		return nil
	})
	if err != nil {
		return DelegationPage{}, err
	}
	return page, nil
}

func listScope(asked string) (string, bool) {
	switch asked {
	case "":
		return ScopeMine, true
	case ScopeMine, ScopeAll:
		return asked, true
	}
	return "", false
}

// manageOthers says whether a holds workflows.manage, and refuses everyone's delegations without it (403).
func (s *Service) manageOthers(ctx context.Context, a actor.Actor, scope string) (bool, error) {
	manage, err := s.access.Check(ctx, a.ID, PermissionManageWorkflows, nil)
	if err != nil {
		return false, err
	}
	if scope == ScopeAll && !manage.Allowed {
		return false, errs.Forbidden(manage.Refusal)
	}
	return manage.Allowed, nil
}

// visibleTo keeps everyone's delegations for scope all, otherwise those where a delegates or is the delegate.
func visibleTo(all []Delegation, a actor.Actor, scope string) []Delegation {
	return where(all, func(dl Delegation) bool {
		return scope == ScopeAll || dl.PersonID == a.ID || dl.DelegateID == a.ID
	})
}

// compareDelegations orders delegations as the panel lists them: current and planned first, by first day;
// then ended and cancelled, latest last day first; then by id.
func compareDelegations(x, y Delegation, today date.Date) int {
	group := func(dl Delegation) int {
		switch dl.state(today) {
		case DelegationCurrent, DelegationPlanned:
			return 0
		}
		return 1
	}
	if c := cmp.Compare(group(x), group(y)); c != 0 {
		return c
	}
	if group(x) == 0 {
		return cmp.Or(x.StartDate.Compare(y.StartDate), cmp.Compare(x.ID, y.ID))
	}
	return cmp.Or(y.EndDate.Compare(x.EndDate), cmp.Compare(x.ID, y.ID))
}
