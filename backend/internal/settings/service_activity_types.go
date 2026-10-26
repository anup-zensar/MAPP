package settings

import (
	"context"
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

// The limit of an activity type's name (data model §3).
const maxActivityTypeName = 120

// Messages of the plan's "User-visible messages" table about activity types.
const (
	activityTypeGoneMessage    = "This activity type no longer exists — someone may have deleted it."
	activityTypeNameTaken      = "An activity type called '%s' already exists."
	defaultCostMessage         = "Enter an amount of 0 or more, with at most two decimals."
	defaultCostInUSDMessage    = "Enter the default planned cost in USD."
	tickedModelGoneMessage     = "A model you ticked no longer exists — reload the page and try again."
	lastTypeOfModelMessage     = "'%s' needs at least one activity type, so '%s' can't be removed from it."
	removedFromModelMessage    = "In plans using '%s', %s use '%s'. They keep it; new activities can't choose it."
	removedFromModelOneMessage = "In plans using '%s', 1 activity uses '%s'. It keeps it; new activities can't choose it."
)

// ListActivityTypes returns every activity type, active and inactive, by name ignoring letter case, each
// with the models that allow it and its usage (FR-002, FR-003). Every signed-in person may see them
// (FR-008).
func (s *Service) ListActivityTypes(ctx context.Context, a actor.Actor) ([]ActivityTypeView, error) {
	var views []ActivityTypeView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		types, err := s.repo.ListActivityTypes(ctx)
		if err != nil {
			return err
		}
		slices.SortStableFunc(types, func(x, y ActivityType) int { return compareNames(x.Name, y.Name) })
		views, err = s.activityTypeViews(ctx, types)
		return err
	})
	return views, err
}

// GetActivityType returns the activity type id with its models and usage (US5-2).
func (s *Service) GetActivityType(ctx context.Context, a actor.Actor, id string) (ActivityTypeView, error) {
	var view ActivityTypeView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		t, err := s.storedActivityType(ctx, id)
		if err != nil {
			return err
		}
		view, err = s.activityTypeView(ctx, t)
		return err
	})
	return view, err
}

// CreateActivityType creates an active activity type (FR-013, US5-1): a name of at most 120 characters that
// no other type has, an optional default planned cost of 0 or more USD, and the models that allow it. Each
// of those models gains the type in the same transaction, with one "changed" entry on the model (data
// model §12). Every problem is reported at once.
func (s *Service) CreateActivityType(ctx context.Context, a actor.Actor, input ActivityTypeInput) (ActivityTypeView, error) {
	modelIDs := input.ModelIDs
	if modelIDs == nil {
		modelIDs = []string{}
	}
	all := ActivityTypeUpdate{
		Name: &input.Name, DefaultPlannedCost: OptionalMoney{Set: true, Value: input.DefaultPlannedCost}, ModelIDs: &modelIDs,
	}
	var view ActivityTypeView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		var problems errs.Problems
		t, allowedIn, err := s.applyActivityType(ctx, ActivityType{}, all, &problems)
		if err != nil {
			return err
		}
		if err := problems.Err(); err != nil {
			return err
		}
		if t.ID, err = s.ids.Next(ctx, ids.PrefixActivityType); err != nil {
			return err
		}
		t.Active, t.Meta = true, record.NewMeta(a, s.now())
		if err := s.repo.PutActivityType(ctx, t); err != nil {
			return err
		}
		if err := s.logCreated(ctx, a, activityTypeKind, t.ID, t.Name, ""); err != nil {
			return err
		}
		if err := s.changeAllowedIn(ctx, a, t.ID, allowedIn); err != nil {
			return err
		}
		view, err = s.activityTypeView(ctx, t)
		return err
	})
	return view, err
}

// UpdateActivityType changes the fields of the activity type id that update sends (FR-012, FR-013, D-203),
// from the version the person started at (FR-007). A default planned cost sent as null is removed; it
// applies only when someone picks the type afterwards (R7). modelIds, when sent, is the complete new list
// of models allowing the type: models added gain it and models left out lose it — refused when a model
// would keep no type. The allowed types live on the models (D-046), so each changed model gets its own
// "changed" entry, and the type itself changes (new version, one entry) only when its own fields do.
func (s *Service) UpdateActivityType(ctx context.Context, a actor.Actor, id string, update ActivityTypeUpdate) (ActivityTypeView, error) {
	var view ActivityTypeView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		stored, err := s.storedActivityType(ctx, id)
		if err != nil {
			return err
		}
		var problems errs.Problems
		checkVersion(&problems, update.Version)
		changed, allowedIn, err := s.applyActivityType(ctx, stored, update, &problems)
		if err != nil {
			return err
		}
		if err := problems.Err(); err != nil {
			return err
		}
		if err := stored.CheckVersion(*update.Version, activityTypeNoun); err != nil {
			return err
		}
		changes, err := history.Diff(stored, changed)
		if err != nil {
			return err
		}
		if len(changes) > 0 {
			changed.Touch(a, s.now())
			if err := s.repo.PutActivityType(ctx, changed); err != nil {
				return err
			}
			if err := s.logChanged(ctx, a, activityTypeKind, changed.ID, changed.Name, changes, ""); err != nil {
				return err
			}
			stored = changed
		}
		if err := s.changeAllowedIn(ctx, a, stored.ID, allowedIn); err != nil {
			return err
		}
		view, err = s.activityTypeView(ctx, stored)
		return err
	})
	return view, err
}

// ActivityTypeImpact says, before a change of the activity type id is saved, what it would affect (FR-012,
// D-044): for each model that would no longer allow it, the live activities of the type in that model's
// live plans, which keep their type. The body is checked as UpdateActivityType checks it, except the
// version; nothing is saved.
func (s *Service) ActivityTypeImpact(ctx context.Context, a actor.Actor, id string, update ActivityTypeUpdate) (ChangeImpact, error) {
	impact := ChangeImpact{Warnings: []ImpactWarning{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		stored, err := s.storedActivityType(ctx, id)
		if err != nil {
			return err
		}
		var problems errs.Problems
		_, allowedIn, err := s.applyActivityType(ctx, stored, update, &problems)
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
		for _, change := range allowedIn {
			if change.allow {
				continue
			}
			plans, _ := modelInUse(refs, change.model.ID)
			n := liveActivitiesOfType(refs, stored.ID, plans)
			if n == 0 {
				continue
			}
			message := fmt.Sprintf(removedFromModelMessage, change.model.Name, plural(n, "activity", "activities"), stored.Name)
			if n == 1 {
				message = fmt.Sprintf(removedFromModelOneMessage, change.model.Name, stored.Name)
			}
			impact.Warnings = append(impact.Warnings, ImpactWarning{Kind: ImpactActivityTypeRemoved, Message: message, Count: n})
		}
		return nil
	})
	return impact, err
}

// SetActivityTypeActive sets the activity type id inactive or reactivates it (FR-004): new activities cannot
// choose an inactive type; activities using it keep it. The same status again changes nothing.
func (s *Service) SetActivityTypeActive(ctx context.Context, a actor.Actor, id string, update StatusUpdate) (ActivityTypeView, error) {
	active, version, err := checkStatus(update)
	if err != nil {
		return ActivityTypeView{}, err
	}
	var view ActivityTypeView
	err = s.tx.Write(ctx, func(ctx context.Context) error {
		t, err := s.storedActivityType(ctx, id)
		if err != nil {
			return err
		}
		if err := t.CheckVersion(version, activityTypeNoun); err != nil {
			return err
		}
		if t.Active != active {
			t.Active = active
			t.Touch(a, s.now())
			if err := s.repo.PutActivityType(ctx, t); err != nil {
				return err
			}
			if err := s.logStatus(ctx, a, activityTypeKind, t.ID, t.Name, active); err != nil {
				return err
			}
		}
		view, err = s.activityTypeView(ctx, t)
		return err
	})
	return view, err
}

// DeleteActivityType moves the activity type id to the recycle bin, so that it can be undone (FR-004,
// US5-5). A type in use — by activities, also those in the recycle bin, or allowed in a model, also one in
// the recycle bin — is refused with in_use: set it inactive instead.
func (s *Service) DeleteActivityType(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		t, err := s.storedActivityType(ctx, id)
		if err != nil {
			return err
		}
		u, err := s.usage(ctx)
		if err != nil {
			return err
		}
		if usage := u.activityType(t.ID); usage.InUse {
			return inUseError(t.Name, usage)
		}
		entry, err = s.history.MoveToBin(ctx, a, activityTypeKind, t.ID)
		return err
	})
	return entry, err
}

// storedActivityType reads the activity type id, or answers not found.
func (s *Service) storedActivityType(ctx context.Context, id string) (ActivityType, error) {
	t, found, err := s.repo.GetActivityType(ctx, id)
	if err != nil {
		return ActivityType{}, err
	}
	if !found {
		return ActivityType{}, errs.NotFound(activityTypeGoneMessage)
	}
	return t, nil
}

// allowedChange is a model whose allowed activity types change because the models allowing one type were
// edited from that type: it gains the type (allow) or loses it.
type allowedChange struct {
	model Model
	allow bool
}

// applyActivityType returns t with the fields update sends, and adds every problem with them to problems
// (data model §3): the name, the default planned cost and, when modelIds is sent, the models that are to
// allow it — returned as the models whose list changes. Fields not sent stay as they are.
func (s *Service) applyActivityType(ctx context.Context, t ActivityType, update ActivityTypeUpdate, problems *errs.Problems) (ActivityType, []allowedChange, error) {
	if update.Name != nil {
		name := strings.TrimSpace(*update.Name)
		checkText(problems, "name", name, enterNameMessage, maxActivityTypeName)
		if name != "" && utf8.RuneCountInString(name) <= maxActivityTypeName {
			types, err := s.repo.ListActivityTypes(ctx)
			if err != nil {
				return ActivityType{}, nil, err
			}
			if other, taken := findNamed(types, name, t.ID, func(x ActivityType) (string, string) { return x.ID, x.Name }); taken {
				problems.Add("name", fmt.Sprintf(activityTypeNameTaken, other.Name))
			}
		}
		t.Name = name
	}
	if update.DefaultPlannedCost.Set {
		t.DefaultPlannedCost = parseDefaultCost(problems, update.DefaultPlannedCost.Value)
	}
	if update.ModelIDs == nil {
		return t, nil, nil
	}
	allowedIn, err := s.checkAllowedIn(ctx, t, *update.ModelIDs, problems)
	return t, allowedIn, err
}

// parseDefaultCost reads a default planned cost: none (nil), or USD 0.00 or more with at most two decimals
// (FR-013).
func parseDefaultCost(problems *errs.Problems, in *money.Input) *money.Money {
	if in == nil {
		return nil
	}
	amount, err := money.ParseAmount(strings.TrimSpace(in.Amount))
	switch {
	case err != nil || amount.Sign() < 0:
		problems.Add("defaultPlannedCost", defaultCostMessage)
	case in.Currency != money.BaseCurrency:
		problems.Add("defaultPlannedCost", defaultCostInUSDMessage)
	default:
		cost := money.USD(amount)
		return &cost
	}
	return nil
}

// checkAllowedIn checks modelIDs, the complete new list of models that are to allow the activity type t
// (FR-013, data model §3): each must be a stored model, and a model that would lose t must keep another
// type. It returns the models whose list changes, by name, each gaining or losing t. A new type (no id yet)
// is allowed by no model, so models only gain it.
func (s *Service) checkAllowedIn(ctx context.Context, t ActivityType, modelIDs []string, problems *errs.Problems) ([]allowedChange, error) {
	models, err := s.repo.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	slices.SortStableFunc(models, func(x, y Model) int { return compareNames(x.Name, y.Name) })
	wanted := unique(modelIDs)
	changes := []allowedChange{}
	for _, m := range models {
		allowed := t.ID != "" && m.Allows(t.ID)
		switch wants := slices.Contains(wanted, m.ID); {
		case wants && !allowed:
			changes = append(changes, allowedChange{model: m, allow: true})
		case !wants && allowed && len(withoutID(m.ActivityTypeIDs, t.ID)) == 0:
			problems.Add("modelIds", fmt.Sprintf(lastTypeOfModelMessage, m.Name, t.Name))
		case !wants && allowed:
			changes = append(changes, allowedChange{model: m, allow: false})
		}
	}
	for _, id := range wanted {
		if !slices.ContainsFunc(models, func(m Model) bool { return m.ID == id }) {
			problems.Add("modelIds", tickedModelGoneMessage)
			break
		}
	}
	return changes, nil
}

// changeAllowedIn saves the models of changes, each gaining or losing the activity type typeID, with their
// allowed types in the order of the activity-type list (by name), and one "changed" entry each (data
// model §12). The type must be stored already, so the log names it.
func (s *Service) changeAllowedIn(ctx context.Context, a actor.Actor, typeID string, changes []allowedChange) error {
	if len(changes) == 0 {
		return nil
	}
	types, err := s.repo.ListActivityTypes(ctx)
	if err != nil {
		return err
	}
	for _, change := range changes {
		typeIDs := withoutID(change.model.ActivityTypeIDs, typeID)
		if change.allow {
			typeIDs = append(typeIDs, typeID)
		}
		changed := change.model
		changed.ActivityTypeIDs = inTypeOrder(types, typeIDs)
		diff, err := history.Diff(change.model, changed)
		if err != nil {
			return err
		}
		changed.Touch(a, s.now())
		if err := s.repo.PutModel(ctx, changed); err != nil {
			return err
		}
		if err := s.logChanged(ctx, a, modelKind, changed.ID, changed.Name, diff, ""); err != nil {
			return err
		}
	}
	return nil
}

// withoutID returns list without id; never nil.
func withoutID(list []string, id string) []string {
	return filter(list, func(other string) bool { return other != id })
}

// inTypeOrder returns the activity type ids in the order of the activity-type list: by name ignoring
// letter case. An id that is not a stored type stays, after the others.
func inTypeOrder(types []ActivityType, typeIDs []string) []string {
	sorted := slices.Clone(types)
	slices.SortStableFunc(sorted, func(x, y ActivityType) int { return compareNames(x.Name, y.Name) })
	ordered := []string{}
	for _, t := range sorted {
		if slices.Contains(typeIDs, t.ID) {
			ordered = append(ordered, t.ID)
		}
	}
	for _, id := range typeIDs {
		if !slices.Contains(ordered, id) {
			ordered = append(ordered, id)
		}
	}
	return ordered
}

// activityTypeView returns the activity type t as the API answers it.
func (s *Service) activityTypeView(ctx context.Context, t ActivityType) (ActivityTypeView, error) {
	views, err := s.activityTypeViews(ctx, []ActivityType{t})
	if err != nil {
		return ActivityTypeView{}, err
	}
	return views[0], nil
}

// activityTypeViews returns activity types as the API answers them: each with the models allowing it, by
// name, and its usage.
func (s *Service) activityTypeViews(ctx context.Context, types []ActivityType) ([]ActivityTypeView, error) {
	models, err := s.repo.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	slices.SortStableFunc(models, func(x, y Model) int { return compareNames(x.Name, y.Name) })
	u, err := s.usage(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]ActivityTypeView, 0, len(types))
	for _, t := range types {
		allowedIn := []SettingRef{}
		for _, m := range models {
			if m.Allows(t.ID) {
				allowedIn = append(allowedIn, SettingRef{ID: m.ID, Name: m.Name, Active: m.Active})
			}
		}
		views = append(views, ActivityTypeView{
			ID: t.ID, Name: t.Name, DefaultPlannedCost: t.DefaultPlannedCost, Models: allowedIn, Active: t.Active,
			Usage: u.activityType(t.ID), Meta: t.Meta,
		})
	}
	return views, nil
}
