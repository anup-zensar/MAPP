package settings

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// The limits of a model (data model §2, D-053).
const maxModelName = 80

// maxDealSize is the largest average deal size: 100,000,000.00 USD.
var maxDealSize = money.FromCents(10_000_000_000)

// Messages of the plan's "User-visible messages" table about models.
const (
	modelGoneMessage         = "This model no longer exists — someone may have deleted it."
	modelNameTakenMessage    = "A model called '%s' already exists."
	modelRateMessage         = "Enter a rate above 0 % and at most 100 %, with at most two decimals."
	dealSizeMessage          = "Enter an average deal size above 0 USD, with at most two decimals."
	dealSizeTooLargeMessage  = "Use 100,000,000.00 USD or less."
	dealSizeInUSDMessage     = "Enter the average deal size in USD."
	tickTypeMessage          = "Tick at least one activity type."
	tickedTypeGoneMessage    = "An activity type you ticked no longer exists — reload the page and try again."
	tickedTypeInactive       = "'%s' is inactive — tick active activity types only."
	goalsRecalculatedMessage = "Used by %s (%s). Their calculated MQL and revenue goals will be recalculated; " +
		"typed goals ('Set later stages myself') are kept."
	typeRemovedMessage    = "%s use '%s'. They keep it; new activities can't choose it."
	typeRemovedOneMessage = "1 activity uses '%s'. It keeps it; new activities can't choose it."
	goalsDetails          = "Used by %s (%s): calculated goals recalculated."
)

// ListModels returns every model, active and inactive, by name ignoring letter case, each with its
// allowed activity types and its usage (FR-002, FR-003). Every signed-in person may see them (FR-008).
func (s *Service) ListModels(ctx context.Context, a actor.Actor) ([]ModelView, error) {
	var views []ModelView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		models, err := s.repo.ListModels(ctx)
		if err != nil {
			return err
		}
		slices.SortStableFunc(models, func(x, y Model) int { return compareNames(x.Name, y.Name) })
		views, err = s.modelViews(ctx, models)
		return err
	})
	return views, err
}

// GetModel returns the model id with its usage and the live plans using it (US1-3).
func (s *Service) GetModel(ctx context.Context, a actor.Actor, id string) (ModelView, error) {
	var view ModelView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		m, err := s.storedModel(ctx, id)
		if err != nil {
			return err
		}
		view, err = s.modelView(ctx, m)
		return err
	})
	return view, err
}

// CreateModel creates an active model (FR-009, US1-1): a name of at most 80 characters that no other model
// has, two rates above 0 % and at most 100 %, an average deal size in USD above 0 and at most
// 100,000,000.00, and at least one active activity type. Every problem is reported at once (US1-5).
func (s *Service) CreateModel(ctx context.Context, a actor.Actor, input ModelInput) (ModelView, error) {
	typeIDs := input.ActivityTypeIDs
	all := ModelUpdate{
		Name: &input.Name, InqToMqlRate: &input.InqToMqlRate, MqlToDealRate: &input.MqlToDealRate,
		AverageDealSize: OptionalMoney{Set: true, Value: input.AverageDealSize}, ActivityTypeIDs: &typeIDs,
	}
	var view ModelView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		var problems errs.Problems
		m, err := s.applyModel(ctx, Model{}, all, &problems)
		if err != nil {
			return err
		}
		if err := problems.Err(); err != nil {
			return err
		}
		if m.ID, err = s.ids.Next(ctx, ids.PrefixModel); err != nil {
			return err
		}
		m.Active, m.Meta = true, record.NewMeta(a, s.now())
		if err := s.repo.PutModel(ctx, m); err != nil {
			return err
		}
		if err := s.logCreated(ctx, a, modelKind, m.ID, m.Name, ""); err != nil {
			return err
		}
		view, err = s.modelView(ctx, m)
		return err
	})
	return view, err
}

// UpdateModel changes the fields of the model id that update sends (FR-009, FR-011, FR-012, D-203), from
// the version the person started at (FR-007). Activity types newly allowed must be active; an inactive type
// already allowed may stay (FR-004). A save that changes nothing writes no version and no log entry;
// otherwise one "changed" entry lists every old and new value and, when the rates or the deal size of a
// model in use changed, says the goals were recalculated (US1-4).
func (s *Service) UpdateModel(ctx context.Context, a actor.Actor, id string, update ModelUpdate) (ModelView, error) {
	var view ModelView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		m, err := s.storedModel(ctx, id)
		if err != nil {
			return err
		}
		var problems errs.Problems
		checkVersion(&problems, update.Version)
		changed, err := s.applyModel(ctx, m, update, &problems)
		if err != nil {
			return err
		}
		if err := problems.Err(); err != nil {
			return err
		}
		if err := m.CheckVersion(*update.Version, modelNoun); err != nil {
			return err
		}
		changes, err := history.Diff(m, changed)
		if err != nil {
			return err
		}
		if len(changes) > 0 {
			details := ""
			if goalsChange(m, changed) {
				refs, err := s.references(ctx)
				if err != nil {
					return err
				}
				if plans, activities := modelInUse(refs, m.ID); len(plans) > 0 {
					details = fmt.Sprintf(goalsDetails, plural(len(plans), "plan", "plans"), plural(activities, "activity", "activities"))
				}
			}
			changed.Touch(a, s.now())
			if err := s.repo.PutModel(ctx, changed); err != nil {
				return err
			}
			if err := s.logChanged(ctx, a, modelKind, changed.ID, changed.Name, changes, details); err != nil {
				return err
			}
			m = changed
		}
		view, err = s.modelView(ctx, m)
		return err
	})
	return view, err
}

// ModelImpact says, before a change of the model id is saved, what it would affect (FR-011, FR-012, D-044):
// changed rates or deal size recalculate the goals of the live plans using it; each activity type no longer
// allowed is kept by the live activities of those plans that use it. The body is checked as UpdateModel
// checks it, except the version; nothing is saved.
func (s *Service) ModelImpact(ctx context.Context, a actor.Actor, id string, update ModelUpdate) (ChangeImpact, error) {
	impact := ChangeImpact{Warnings: []ImpactWarning{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		m, err := s.storedModel(ctx, id)
		if err != nil {
			return err
		}
		var problems errs.Problems
		changed, err := s.applyModel(ctx, m, update, &problems)
		if err != nil {
			return err
		}
		if err := problems.Err(); err != nil {
			return err
		}
		refs, err := s.references(ctx)
		if err != nil {
			return err
		}
		plans, activities := modelInUse(refs, m.ID)
		if goalsChange(m, changed) && len(plans) > 0 {
			impact.Warnings = append(impact.Warnings, ImpactWarning{
				Kind:    ImpactGoalsRecalculated,
				Message: fmt.Sprintf(goalsRecalculatedMessage, plural(len(plans), "plan", "plans"), plural(activities, "activity", "activities")),
				Count:   len(plans),
			})
		}
		for _, typeID := range m.ActivityTypeIDs {
			if slices.Contains(changed.ActivityTypeIDs, typeID) {
				continue
			}
			n := liveActivitiesOfType(refs, typeID, plans)
			if n == 0 {
				continue
			}
			t, _, err := s.repo.GetActivityType(ctx, typeID)
			if err != nil {
				return err
			}
			message := fmt.Sprintf(typeRemovedMessage, plural(n, "activity", "activities"), t.Name)
			if n == 1 {
				message = fmt.Sprintf(typeRemovedOneMessage, t.Name)
			}
			impact.Warnings = append(impact.Warnings, ImpactWarning{Kind: ImpactActivityTypeRemoved, Message: message, Count: n})
		}
		return nil
	})
	return impact, err
}

// SetModelActive sets the model id inactive or reactivates it (FR-004): plans using an inactive model keep
// it and still calculate goals; new plans cannot choose it. The same status again changes nothing.
func (s *Service) SetModelActive(ctx context.Context, a actor.Actor, id string, update StatusUpdate) (ModelView, error) {
	active, version, err := checkStatus(update)
	if err != nil {
		return ModelView{}, err
	}
	var view ModelView
	err = s.tx.Write(ctx, func(ctx context.Context) error {
		m, err := s.storedModel(ctx, id)
		if err != nil {
			return err
		}
		if err := m.CheckVersion(version, modelNoun); err != nil {
			return err
		}
		if m.Active != active {
			m.Active = active
			m.Touch(a, s.now())
			if err := s.repo.PutModel(ctx, m); err != nil {
				return err
			}
			if err := s.logStatus(ctx, a, modelKind, m.ID, m.Name, active); err != nil {
				return err
			}
		}
		view, err = s.modelView(ctx, m)
		return err
	})
	return view, err
}

// DeleteModel moves the model id to the recycle bin, so that it can be undone (FR-004, US1-7). A model in
// use — by any plan, also one in the recycle bin — is refused with in_use: set it inactive instead.
func (s *Service) DeleteModel(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		m, err := s.storedModel(ctx, id)
		if err != nil {
			return err
		}
		u, err := s.usage(ctx)
		if err != nil {
			return err
		}
		if usage := u.model(m.ID); usage.InUse {
			return inUseError(m.Name, usage)
		}
		entry, err = s.history.MoveToBin(ctx, a, modelKind, m.ID)
		return err
	})
	return entry, err
}

// storedModel reads the model id, or answers not found.
func (s *Service) storedModel(ctx context.Context, id string) (Model, error) {
	m, found, err := s.repo.GetModel(ctx, id)
	if err != nil {
		return Model{}, err
	}
	if !found {
		return Model{}, errs.NotFound(modelGoneMessage)
	}
	return m, nil
}

// applyModel returns m with the fields update sends, and adds every problem with them to problems (data
// model §2): the name, the rates, the deal size and the allowed activity types, which it checks against the
// stored models and activity types. Fields not sent stay as they are.
func (s *Service) applyModel(ctx context.Context, m Model, update ModelUpdate, problems *errs.Problems) (Model, error) {
	if update.Name != nil {
		name := strings.TrimSpace(*update.Name)
		checkText(problems, "name", name, enterNameMessage, maxModelName)
		if name != "" && utf8.RuneCountInString(name) <= maxModelName {
			models, err := s.repo.ListModels(ctx)
			if err != nil {
				return Model{}, err
			}
			if other, taken := findNamed(models, name, m.ID, func(m Model) (string, string) { return m.ID, m.Name }); taken {
				problems.Add("name", fmt.Sprintf(modelNameTakenMessage, other.Name))
			}
		}
		m.Name = name
	}
	if update.InqToMqlRate != nil {
		m.InqToMqlRate = parseModelRate(problems, "inqToMqlRate", *update.InqToMqlRate)
	}
	if update.MqlToDealRate != nil {
		m.MqlToDealRate = parseModelRate(problems, "mqlToDealRate", *update.MqlToDealRate)
	}
	if update.AverageDealSize.Set {
		m.AverageDealSize = parseDealSize(problems, update.AverageDealSize.Value)
	}
	if update.ActivityTypeIDs != nil {
		typeIDs, err := s.allowedTypes(ctx, *update.ActivityTypeIDs, m.ActivityTypeIDs, problems)
		if err != nil {
			return Model{}, err
		}
		m.ActivityTypeIDs = typeIDs
	}
	return m, nil
}

// parseModelRate reads a conversion rate: a percentage above 0 and at most 100.00 with at most two
// decimals (FR-009). Above 0, because plan goals are worked backwards from it.
func parseModelRate(problems *errs.Problems, field, text string) money.Percent {
	rate, err := money.ParsePercent(strings.TrimSpace(text))
	if err != nil || rate.IsZero() || rate.Cmp(money.Hundred) > 0 {
		problems.Add(field, modelRateMessage)
		return money.Percent{}
	}
	return rate
}

// parseDealSize reads the average deal size: USD, above 0 and at most 100,000,000.00 (FR-009, D-053).
func parseDealSize(problems *errs.Problems, in *money.Input) money.Money {
	if in == nil {
		problems.Add("averageDealSize", dealSizeMessage)
		return money.Money{}
	}
	amount, err := money.ParseAmount(strings.TrimSpace(in.Amount))
	switch {
	case errors.Is(err, money.ErrOutOfRange):
		problems.Add("averageDealSize", dealSizeTooLargeMessage)
	case err != nil || amount.Sign() <= 0:
		problems.Add("averageDealSize", dealSizeMessage)
	case amount.Cmp(maxDealSize) > 0:
		problems.Add("averageDealSize", dealSizeTooLargeMessage)
	case in.Currency != money.BaseCurrency:
		problems.Add("averageDealSize", dealSizeInUSDMessage)
	default:
		return money.USD(amount)
	}
	return money.Money{}
}

// allowedTypes checks the activity types a model is to allow (FR-009, FR-004): at least one, each still
// stored, and each active unless the model already allowed it (previous). It returns them once each, in the
// order of the activity-type list (by name).
func (s *Service) allowedTypes(ctx context.Context, typeIDs, previous []string, problems *errs.Problems) ([]string, error) {
	typeIDs = unique(typeIDs)
	if len(typeIDs) == 0 {
		problems.Add("activityTypeIds", tickTypeMessage)
		return typeIDs, nil
	}
	types, err := s.repo.ListActivityTypes(ctx)
	if err != nil {
		return nil, err
	}
	slices.SortStableFunc(types, func(x, y ActivityType) int { return compareNames(x.Name, y.Name) })
	allowed := []string{}
	for _, t := range types {
		if !slices.Contains(typeIDs, t.ID) {
			continue
		}
		if !t.Active && !slices.Contains(previous, t.ID) {
			problems.Add("activityTypeIds", fmt.Sprintf(tickedTypeInactive, t.Name))
		}
		allowed = append(allowed, t.ID)
	}
	if len(allowed) < len(typeIDs) {
		problems.Add("activityTypeIds", tickedTypeGoneMessage)
	}
	return allowed, nil
}

// goalsChange reports whether a change of a model changes the goals it calculates: its rates or its deal
// size (FR-011).
func goalsChange(before, after Model) bool {
	return before.InqToMqlRate != after.InqToMqlRate || before.MqlToDealRate != after.MqlToDealRate ||
		before.AverageDealSize != after.AverageDealSize
}

// modelInUse returns the live plans using the model modelID and the number of live activities in them.
// Impact statements count live records only: records in the recycle bin are recalculated when they come
// back (data model §10).
func modelInUse(refs []Reference, modelID string) (plans map[string]bool, activities int) {
	plans = map[string]bool{}
	for _, ref := range refs {
		if !ref.InBin && ref.Level == LevelPlan && ref.ModelID == modelID {
			plans[ref.PlanID] = true
		}
	}
	for _, ref := range refs {
		if !ref.InBin && ref.Level == LevelActivity && plans[ref.PlanID] {
			activities++
		}
	}
	return plans, activities
}

// liveActivitiesOfType counts the live activities of the activity type typeID in the plans.
func liveActivitiesOfType(refs []Reference, typeID string, plans map[string]bool) int {
	n := 0
	for _, ref := range refs {
		if !ref.InBin && ref.Level == LevelActivity && ref.ActivityTypeID == typeID && plans[ref.PlanID] {
			n++
		}
	}
	return n
}

// modelView returns the model m as the API answers it.
func (s *Service) modelView(ctx context.Context, m Model) (ModelView, error) {
	views, err := s.modelViews(ctx, []Model{m})
	if err != nil {
		return ModelView{}, err
	}
	return views[0], nil
}

// modelViews returns models as the API answers them, with their allowed activity types by name and their
// usage, read once for all of them.
func (s *Service) modelViews(ctx context.Context, models []Model) ([]ModelView, error) {
	types, err := s.repo.ListActivityTypes(ctx)
	if err != nil {
		return nil, err
	}
	u, err := s.usage(ctx)
	if err != nil {
		return nil, err
	}
	typesByID := map[string]ActivityType{}
	for _, t := range types {
		typesByID[t.ID] = t
	}
	views := make([]ModelView, 0, len(models))
	for _, m := range models {
		refs := []SettingRef{}
		for _, id := range m.ActivityTypeIDs {
			if t, found := typesByID[id]; found {
				refs = append(refs, SettingRef{ID: t.ID, Name: t.Name, Active: t.Active})
			}
		}
		slices.SortStableFunc(refs, func(x, y SettingRef) int { return compareNames(x.Name, y.Name) })
		views = append(views, ModelView{
			ID: m.ID, Name: m.Name, InqToMqlRate: m.InqToMqlRate, MqlToDealRate: m.MqlToDealRate,
			AverageDealSize: m.AverageDealSize, ActivityTypes: refs, Active: m.Active, Usage: u.model(m.ID), Meta: m.Meta,
		})
	}
	return views, nil
}
