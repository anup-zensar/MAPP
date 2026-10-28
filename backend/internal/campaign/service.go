package campaign

import (
	"context"
	"maps"

	"mapp/internal/history"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/settings"
)

type Service struct {
	tx           Transactor
	repo         Store
	ids          *ids.Generator
	history      *history.Service
	now          clock.Clock
	deps         Deps
	displayRates DisplayRates // nil until UseDisplayRates: amounts show in USD
	purchases    PurchaseRequests
}

func NewService(tools history.Tools, repo Store, deps Deps) *Service {
	deps.Plans = inPlanCurrency{deps.Plans}
	return &Service{tx: tools.Tx, repo: repo, ids: tools.IDs, history: tools.Log, now: tools.Now, deps: deps}
}

// checkOwner refuses a newly chosen owner who is not an active person; an owner already stored stays.
func (s *Service) checkOwner(ctx context.Context, problems *errs.Problems, ownerID, storedOwnerID *string) error {
	if ownerID == nil || (storedOwnerID != nil && *storedOwnerID == *ownerID) {
		return nil
	}
	person, found, err := s.deps.People.Get(ctx, *ownerID)
	if err != nil {
		return err
	}
	if !found || !person.Active {
		problems.Add("ownerId", chooseActivePersonMessage)
	}
	return nil
}

// attributesSent holds no stored values for a new record.
type attributesSent struct {
	level          settings.Level
	plan           PlanFacts
	values, stored settings.AttributeValues
}

// checkAttributes adds Settings' problems to problems and returns the values to store, never nil.
func (s *Service) checkAttributes(ctx context.Context, problems *errs.Problems, sent attributesSent) (settings.AttributeValues, error) {
	businessUnit, err := s.deps.Settings.BusinessUnit(ctx, sent.plan.Attributes)
	if err != nil {
		return nil, err
	}
	result, err := s.deps.Settings.CheckAttributes(ctx, sent.level, settings.AttributeInput{
		Values: orEmpty(sent.values), Previous: orEmpty(sent.stored), BusinessUnit: businessUnit,
	})
	if err != nil {
		return nil, err
	}
	*problems = append(*problems, result.Problems...)
	if result.Values != nil {
		return result.Values, nil
	}
	return maps.Clone(orEmpty(sent.values)), nil
}
