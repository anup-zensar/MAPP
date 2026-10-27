package campaign

import (
	"context"
	"maps"

	"mapp/internal/history"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/store"
	"mapp/internal/settings"
)

// Service holds the rules of campaigns and initiatives (005 plan, "Service"). Every method takes the actor,
// runs in one transaction and reports every problem of a save at once. No method checks roles: every
// signed-in person may do everything (FR-031, D-004); the actor is where those checks will go.
type Service struct {
	tx      store.Transactor
	repo    Store
	ids     *ids.Generator
	history *history.Service
	now     clock.Clock
	deps    Deps
	// displayRates reads the reader's display currency (UseDisplayRates, D-220); nil shows USD.
	displayRates DisplayRates
}

// NewService returns the campaigns service. It keeps the records in repo, gives them readable IDs, logs their
// changes, deletes them, works in transactions and reads the time with tools, and reads what it needs from other
// features from deps.
func NewService(tools history.Tools, repo Store, deps Deps) *Service {
	deps.Plans = inPlanCurrency{deps.Plans}
	return &Service{tx: tools.Tx, repo: repo, ids: tools.IDs, history: tools.Log, now: tools.Now, deps: deps}
}

// checkOwner adds a problem when the owner is newly chosen and is not an active person (FR-002). An owner
// the record already has stays, even when inactive (spec edge case).
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

// checkAttributes checks the Shell attribute values sent for level with Settings' rules (FR-002, FR-009,
// R8, R9), adding their problems to problems, and returns the values to store, never nil.
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
