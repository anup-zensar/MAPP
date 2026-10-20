package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"northstar/internal/campaigns"
	"northstar/internal/history"
	"northstar/internal/plans"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/clock"
	"northstar/internal/platform/date"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
	"northstar/internal/platform/store"
	"northstar/internal/settings"
)

// noun names an activity in the version-conflict message: "Not saved — Priya Raman changed this activity".
const noun = "activity"

// The longest texts, in characters (data model §1).
const (
	maxNameLength        = 120
	maxDescriptionLength = 2000
)

// Messages of the plan's "User-visible messages" table.
const (
	activityGoneMessage     = "This activity no longer exists — someone may have deleted it."
	initiativeGoneMessage   = "This initiative no longer exists — someone may have deleted it."
	planGoneMessage         = "This plan no longer exists — someone may have deleted it."
	enterNameMessage        = "Enter a name."
	nameTooLongMessage      = "Use 120 characters or fewer."
	nameTakenFormat         = "'%s' already has an activity called '%s'."
	descriptionTooLong      = "Use 2,000 characters or fewer."
	chooseTypeMessage       = "Choose an activity type."
	typeNotAllowedMessage   = "Choose an activity type that the model '%s' allows."
	choosePersonMessage     = "Choose an active person."
	chooseWorkflowMessage   = "Choose an active approval workflow."
	workflowLockedMessage   = "The approval workflow can't change while the activity is submitted or approved."
	workflowRecallMessage   = "Recall the activity to choose another workflow." // while Submitted (008 FR-013)
	ongoingLockedMessage    = "Ongoing can't be undone — to plan with an end date, create a new activity."
	costNegativeMessage     = "Planned cost can't be negative."
	costCurrencyMessage     = "Enter the amount in %s, the plan's currency." // 016 FR-011
	revenueCurrencyMessage  = "Enter the revenue in USD."
	enterAmountMessage      = "Enter an amount, such as 12,500.00."
	restartApprovalMessage  = "This activity will need approval again — confirm the change to save it."
	reloadMessage           = "Reload the page and try again."
	approvalRestartedFormat = "Approval restarted because %s changed."
	restartVerb             = "restarted approval of"
)

// Service holds the rules of activities (feature 006).
type Service struct {
	tx        store.Transactor
	repo      Repository
	ids       *ids.Generator
	history   *history.Service
	now       clock.Clock
	plans     PlanReader
	hierarchy HierarchyReader
	settings  SettingsReader
	people    PeopleReader
	workflows Workflows
	lineItems LineItems
	hooks     ApprovalHooks
	// displayRates reads the reader's display currency (UseDisplayRates, D-220); nil shows USD.
	displayRates DisplayRates
}

// NewService returns the activities service. It keeps activities in repo, gives them readable IDs with
// generator, logs their changes and deletes them through hist, works in transactions of tx, reads the time
// from now and reads the plans, campaigns, initiatives, Settings, people and approval workflows of deps.
// Until UseLineItems and UseApprovalHooks are called, activities have no line items and nothing reacts to
// approval changes.
func NewService(tx store.Transactor, repo Repository, generator *ids.Generator, hist *history.Service, now clock.Clock, deps Deps) *Service {
	return &Service{
		tx: tx, repo: repo, ids: generator, history: hist, now: now,
		plans: deps.Plans, hierarchy: deps.Hierarchy, settings: deps.Settings, people: deps.People, workflows: deps.Workflows,
		lineItems: noLineItems{}, hooks: noApprovalHooks{},
	}
}

// UseLineItems tells the service where line items are (feature 007, through internal/app).
func (s *Service) UseLineItems(li LineItems) {
	s.lineItems = li
}

// UseApprovalHooks tells the service who reacts to approval changes (feature 008, through internal/app).
func (s *Service) UseApprovalHooks(h ApprovalHooks) {
	s.hooks = h
}

// planContext is where an activity sits: its initiative, campaign and plan, the plan's model and business
// unit.
type planContext struct {
	plan         plans.Plan
	campaign     campaigns.Campaign
	initiative   campaigns.Initiative
	model        settings.Model
	businessUnit []string // the plan's Business unit option ids
}

// contextOf reads the plan context of the initiative initiativeID, and false when the initiative, its
// campaign or its plan is gone.
func (s *Service) contextOf(ctx context.Context, initiativeID string) (planContext, bool, error) {
	initiative, found, err := s.hierarchy.GetInitiative(ctx, initiativeID)
	if err != nil || !found {
		return planContext{}, false, err
	}
	campaign, found, err := s.hierarchy.GetCampaign(ctx, initiative.CampaignID)
	if err != nil || !found {
		return planContext{}, false, err
	}
	plan, found, err := s.plans.Get(ctx, campaign.PlanID)
	if err != nil || !found {
		return planContext{}, false, err
	}
	pc, err := s.planFacts(ctx, plan)
	pc.campaign, pc.initiative = campaign, initiative
	return pc, true, err
}

// planFacts reads what every activity of plan shares: its model and its business unit.
func (s *Service) planFacts(ctx context.Context, plan plans.Plan) (planContext, error) {
	model, _, err := s.settings.Model(ctx, plan.ModelID)
	if err != nil {
		return planContext{}, err
	}
	businessUnit, err := s.settings.BusinessUnit(ctx, plan.Attributes)
	if err != nil {
		return planContext{}, err
	}
	return planContext{plan: plan, model: model, businessUnit: businessUnit}, nil
}

// Create creates an activity under the initiative initiativeID with the status Created (FR-003, FR-004):
// only the name and the type are required; every problem is reported at once (FR-006 – FR-018). It writes
// a "created" log entry in the same Write.
func (s *Service) Create(ctx context.Context, a actor.Actor, initiativeID string, in Input) (View, error) {
	var view View
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		pc, found, err := s.contextOf(ctx, initiativeID)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(initiativeGoneMessage)
		}
		var problems errs.Problems
		activity, err := s.checkFields(ctx, &problems, pc, in, nil)
		if err != nil {
			return err
		}
		if err := problems.Err(); err != nil {
			return err
		}
		if activity.ID, err = s.ids.Next(ctx, ids.PrefixActivity); err != nil {
			return err
		}
		activity.Status, activity.Meta = StatusCreated, record.NewMeta(a, s.now())
		if err := s.repo.Put(ctx, activity); err != nil {
			return err
		}
		_, err = s.history.Log(ctx, a, history.LogInput{
			Kind: kindName, RecordID: activity.ID, RecordName: activity.Name, Action: history.ActionCreated,
		})
		if err != nil {
			return err
		}
		view, err = s.viewOf(ctx, activity, pc)
		return err
	})
	if err != nil {
		return View{}, err
	}
	return s.shown(ctx, a, view)
}

// Get returns the activity id with everything derived from it (FR-019 – FR-024).
func (s *Service) Get(ctx context.Context, a actor.Actor, id string) (View, error) {
	var view View
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		activity, pc, err := s.load(ctx, id)
		if err != nil {
			return err
		}
		view, err = s.viewOf(ctx, activity, pc)
		return err
	})
	if err != nil {
		return View{}, err
	}
	return s.shown(ctx, a, view)
}

// load reads the activity id and its plan context, or answers not found.
func (s *Service) load(ctx context.Context, id string) (Activity, planContext, error) {
	activity, found, err := s.repo.Get(ctx, id)
	if err != nil {
		return Activity{}, planContext{}, err
	}
	if !found {
		return Activity{}, planContext{}, errs.NotFound(activityGoneMessage)
	}
	pc, found, err := s.contextOf(ctx, activity.InitiativeID)
	if err != nil {
		return Activity{}, planContext{}, err
	}
	if !found {
		return Activity{}, planContext{}, errs.NotFound(activityGoneMessage)
	}
	return activity, pc, nil
}

// ListForPlan returns every activity of every initiative of the plan planID as plan-tree rows, ordered by
// start date, then name ignoring letter case, activities without a start date last (FR-033, D-111).
func (s *Service) ListForPlan(ctx context.Context, a actor.Actor, planID string) ([]Summary, error) {
	var summaries []Summary
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		plan, found, err := s.plans.Get(ctx, planID)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(planGoneMessage)
		}
		all, err := s.activitiesOf(ctx, plan)
		if err != nil {
			return err
		}
		facts, err := s.lineItemFacts(ctx, all.activities)
		if err != nil {
			return err
		}
		summaries = make([]Summary, 0, len(all.activities))
		for _, activity := range all.activities {
			d, err := s.derive(ctx, activity, all.contextOf(activity), facts[activity.ID])
			if err != nil {
				return err
			}
			summaries = append(summaries, displaySummary(money.DisplayUSD, Summary{
				ID: activity.ID, Name: activity.Name, InitiativeID: activity.InitiativeID, ActivityType: d.activityType,
				Owner: d.owner, StartDate: activity.StartDate, EndDate: activity.EndDate, Ongoing: activity.Ongoing,
				PlannedCost: activity.PlannedCost, UnassignedPlannedCost: d.unassigned, Status: activity.Status,
				DisplayStatus: d.displayStatus, Completeness: d.completeness, Version: activity.Version,
				SplitFromID: splitFromID(activity),
			}))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.SortStableFunc(summaries, func(x, y Summary) int {
		if c := compareStart(x.StartDate, y.StartDate); c != 0 {
			return c
		}
		return strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name))
	})
	return s.shownSummaries(ctx, a, summaries)
}

// planActivities are every activity of a plan, with the plan's facts and each activity's initiative and
// campaign.
type planActivities struct {
	base        planContext // the plan, its model and business unit
	campaigns   map[string]campaigns.Campaign
	initiatives map[string]campaigns.Initiative
	activities  []Activity
}

// activitiesOf reads every activity of every initiative of every campaign of plan.
func (s *Service) activitiesOf(ctx context.Context, plan plans.Plan) (planActivities, error) {
	base, err := s.planFacts(ctx, plan)
	if err != nil {
		return planActivities{}, err
	}
	all := planActivities{base: base, campaigns: map[string]campaigns.Campaign{}, initiatives: map[string]campaigns.Initiative{}}
	campaignList, err := s.hierarchy.CampaignsOfPlan(ctx, plan.ID)
	if err != nil {
		return planActivities{}, err
	}
	for _, c := range campaignList {
		all.campaigns[c.ID] = c
	}
	initiatives, err := s.hierarchy.InitiativesOfCampaigns(ctx, slices.Collect(maps.Keys(all.campaigns)))
	if err != nil {
		return planActivities{}, err
	}
	for _, i := range initiatives {
		all.initiatives[i.ID] = i
	}
	all.activities, err = s.repo.ListByInitiatives(ctx, slices.Collect(maps.Keys(all.initiatives)))
	return all, err
}

// contextOf returns the plan context of a, an activity of the plan.
func (p planActivities) contextOf(a Activity) planContext {
	pc := p.base
	pc.initiative = p.initiatives[a.InitiativeID]
	pc.campaign = p.campaigns[pc.initiative.CampaignID]
	return pc
}

// compareStart orders start dates, earliest first and no date last.
func compareStart(x, y *date.Date) int {
	switch {
	case x == nil && y == nil:
		return 0
	case x == nil:
		return 1
	case y == nil:
		return -1
	}
	return x.Compare(*y)
}

// Update saves an edited activity (FR-026 – FR-028). update holds every editable field and the version the
// person started from: a missing version is reported with the other problems; an out-of-date one is
// refused with version_conflict. A save that changes nothing answers the activity unchanged. On a
// Submitted or Approved activity, a change of planned cost, start date, end date or ongoing needs
// RestartApproval (D-106) and returns it to Created; other edits keep the status.
func (s *Service) Update(ctx context.Context, a actor.Actor, id string, update Update) (View, error) {
	var view View
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		before, pc, err := s.load(ctx, id)
		if err != nil {
			return err
		}
		if update.Version != nil && *update.Version >= 1 {
			if err := before.CheckVersion(*update.Version, noun); err != nil {
				return err
			}
		}
		var problems errs.Problems
		after, err := s.checkFields(ctx, &problems, pc, update.fields(), &before)
		if err != nil {
			return err
		}
		if update.Version == nil || *update.Version < 1 {
			problems.Add("version", reloadMessage)
		}
		if err := problems.Err(); err != nil {
			return err
		}

		changes, err := diffActivity(before, after)
		if err != nil {
			return err
		}
		if len(changes) == 0 {
			view, err = s.viewOf(ctx, before, pc)
			return err
		}
		restartFields := reapprovalFields(before, changes)
		if len(restartFields) > 0 && !update.RestartApproval {
			return errs.Validation(errs.FieldError{Field: "restartApproval", Message: restartApprovalMessage})
		}
		if len(restartFields) > 0 {
			after.Status = StatusCreated
		}
		after.Touch(a, s.now())
		if err := s.repo.Put(ctx, after); err != nil {
			return err
		}
		entry, err := s.history.Log(ctx, a, history.LogInput{
			Kind: kindName, RecordID: after.ID, RecordName: after.Name, Action: history.ActionChanged, Changes: changes,
		})
		if err != nil {
			return err
		}
		if datesChanged(before, after) {
			if err := s.lineItems.DatesChanged(ctx, a, before, after); err != nil {
				return err
			}
		}
		if len(restartFields) > 0 {
			details := fmt.Sprintf(approvalRestartedFormat, joinWords(restartFields))
			if err := s.logRestart(ctx, a, before, after, entry.Changes, details); err != nil {
				return err
			}
		}
		view, err = s.viewOf(ctx, after, pc)
		return err
	})
	if err != nil {
		return View{}, err
	}
	return s.shown(ctx, a, view)
}

// RestartApproval sends the Submitted or Approved activity activityID back to Created (FR-027, D-196), for
// changes made elsewhere, such as feature 007's line-item changes: it runs inside the caller's Write,
// writes one status entry with the reasons and tells feature 008, which keeps each reason, a ready-made
// sentence, as one change of its approval history (008 plan, Reconciliation). An activity with another
// status is left as it is.
func (s *Service) RestartApproval(ctx context.Context, a actor.Actor, activityID string, reasons []string) error {
	return s.tx.Write(ctx, func(ctx context.Context) error {
		activity, found, err := s.repo.Get(ctx, activityID)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(activityGoneMessage)
		}
		if !inApproval(activity) {
			return nil
		}
		before := activity
		activity.Status = StatusCreated
		activity.Touch(a, s.now())
		if err := s.repo.Put(ctx, activity); err != nil {
			return err
		}
		changes := make([]history.Change, 0, len(reasons))
		for _, reason := range reasons {
			changes = append(changes, history.Change{Label: reason}) // no old or new text: the label says it all
		}
		return s.logRestart(ctx, a, before, activity, changes, strings.Join(reasons, " "))
	})
}

// logRestart writes the status entry of a restarted approval (Submitted or Approved → Created) with its
// details, and tells feature 008 through the approval hooks, in the caller's Write. changes are the
// changes that caused it, with their labels and texts.
func (s *Service) logRestart(ctx context.Context, a actor.Actor, before, after Activity, changes []history.Change, details string) error {
	_, err := s.history.Log(ctx, a, history.LogInput{
		Kind: kindName, RecordID: after.ID, RecordName: after.Name, Action: history.ActionStatusChanged, Verb: restartVerb,
		Changes: []history.Change{{Field: "status", Old: mustJSON(before.Status), New: mustJSON(after.Status)}},
		Details: details,
	})
	if err != nil {
		return err
	}
	return s.hooks.ApprovalRestarted(ctx, a, before, after, changes)
}

// inApproval reports whether a re-approval change of a would restart its approval: it is Submitted or
// Approved (shown also as In progress or Complete).
func inApproval(a Activity) bool {
	return a.Status == StatusSubmitted || a.Status == StatusApproved
}

// reapprovalWords are the fields whose change restarts approval (domain model §2.5), in the order the
// reason names them.
var reapprovalWords = []struct{ field, words string }{
	{"plannedCost", "the planned cost"},
	{"startDate", "the start date"},
	{"endDate", "the end date"},
	{"ongoing", "ongoing (no end date)"},
}

// reapprovalFields returns, in words, the changed fields that restart the approval of before; none when
// before is not Submitted or Approved.
func reapprovalFields(before Activity, changes []history.Change) []string {
	if !inApproval(before) {
		return nil
	}
	var words []string
	for _, w := range reapprovalWords {
		if slices.ContainsFunc(changes, func(c history.Change) bool { return c.Field == w.field }) {
			words = append(words, w.words)
		}
	}
	return words
}

// joinWords joins words as a sentence does: "a", "a and b", "a, b and c".
func joinWords(words []string) string {
	if len(words) <= 1 {
		return strings.Join(words, "")
	}
	return strings.Join(words[:len(words)-1], ", ") + " and " + words[len(words)-1]
}

// datesChanged reports whether the start date, end date or ongoing differ.
func datesChanged(before, after Activity) bool {
	return !sameDate(before.StartDate, after.StartDate) || !sameDate(before.EndDate, after.EndDate) ||
		before.Ongoing != after.Ongoing
}

// sameDate reports whether two optional dates are the same.
func sameDate(x, y *date.Date) bool {
	if x == nil || y == nil {
		return x == nil && y == nil
	}
	return x.Compare(*y) == 0
}

// diffActivity returns the changed fields of an activity, with one change per Shell attribute
// ("attributes.<id>") instead of one for all of them, sorted by field.
func diffActivity(before, after Activity) ([]history.Change, error) {
	b, x := before, after
	b.Attributes, x.Attributes = nil, nil
	changes, err := history.Diff(b, x)
	if err != nil {
		return nil, err
	}
	attributeIDs := map[string]bool{}
	for id := range before.Attributes {
		attributeIDs[id] = true
	}
	for id := range after.Attributes {
		attributeIDs[id] = true
	}
	for _, id := range slices.Sorted(maps.Keys(attributeIDs)) {
		oldValue, newValue := before.Attributes[id], after.Attributes[id]
		if sameJSON(oldValue, newValue) || (isEmptyValue(oldValue) && isEmptyValue(newValue)) {
			continue
		}
		changes = append(changes, history.Change{Field: "attributes." + id, Old: orNull(oldValue), New: orNull(newValue)})
	}
	slices.SortStableFunc(changes, func(x, y history.Change) int { return strings.Compare(x.Field, y.Field) })
	return changes, nil
}

// orNull returns value, or JSON null for no value.
func orNull(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage("null")
	}
	return value
}

// Delete moves the activity id, with its line items, to the recycle bin (FR-031) and returns the
// recycle-bin entry, so that the person can undo it. Feature 008 is told first.
func (s *Service) Delete(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		activity, found, err := s.repo.Get(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(activityGoneMessage)
		}
		if err := s.hooks.ActivityDeleting(ctx, a, activity); err != nil {
			return err
		}
		entry, err = s.history.MoveToBin(ctx, a, kindName, id)
		return err
	})
	if err != nil {
		return history.BinEntry{}, err
	}
	return entry, nil
}

// fields returns the editable fields of an update, as a create sends them.
func (u Update) fields() Input {
	return Input{
		Name: u.Name, ExternalName: u.ExternalName, ActivityTypeID: u.ActivityTypeID, OwnerID: u.OwnerID,
		StartDate: u.StartDate, EndDate: u.EndDate, Ongoing: u.Ongoing, PlannedCost: u.PlannedCost,
		ApprovalWorkflowID: u.ApprovalWorkflowID, Description: u.Description, Attributes: u.Attributes, Goals: u.Goals,
	}
}

// checkFields checks the fields of a create (before nil) or an update of before, adds every problem to
// problems in the order of the form, and returns the activity as it would be saved (FR-006 – FR-018).
// Rules that hold only for new choices — an active type the model allows, an active owner, an active
// workflow, dates inside the initiative's — are checked when the value is new or changed (FR-013, edge
// case "initiative dates moved").
func (s *Service) checkFields(ctx context.Context, problems *errs.Problems, pc planContext, in Input, before *Activity) (Activity, error) {
	next := Activity{Status: StatusCreated}
	if before != nil {
		next = *before
	}
	next.InitiativeID = pc.initiative.ID
	exceptID := next.ID

	next.Name = strings.TrimSpace(in.Name)
	switch {
	case next.Name == "":
		problems.Add("name", enterNameMessage)
	case utf8.RuneCountInString(next.Name) > maxNameLength:
		problems.Add("name", nameTooLongMessage)
	default:
		siblings, err := s.repo.ListByInitiative(ctx, pc.initiative.ID)
		if err != nil {
			return Activity{}, err
		}
		if other, taken := namedLike(siblings, next.Name, exceptID); taken {
			problems.Add("name", fmt.Sprintf(nameTakenFormat, pc.initiative.Name, other.Name))
		}
	}
	next.ExternalName = strings.TrimSpace(in.ExternalName)
	if utf8.RuneCountInString(next.ExternalName) > maxNameLength {
		problems.Add("externalName", nameTooLongMessage)
	}

	next.ActivityTypeID = strings.TrimSpace(in.ActivityTypeID)
	if next.ActivityTypeID == "" {
		problems.Add("activityTypeId", chooseTypeMessage)
	} else if before == nil || before.ActivityTypeID != next.ActivityTypeID {
		t, found, err := s.settings.ActivityType(ctx, next.ActivityTypeID)
		if err != nil {
			return Activity{}, err
		}
		if !found || !t.Active || !pc.model.Allows(t.ID) {
			problems.Add("activityTypeId", fmt.Sprintf(typeNotAllowedMessage, pc.model.Name))
		}
	}

	next.OwnerID = strings.TrimSpace(deref(in.OwnerID))
	if next.OwnerID != "" && (before == nil || before.OwnerID != next.OwnerID) {
		person, found, err := s.people.Get(ctx, next.OwnerID)
		if err != nil {
			return Activity{}, err
		}
		if !found || !person.Active {
			problems.Add("ownerId", choosePersonMessage)
		}
	}

	next.Ongoing = in.Ongoing
	next.StartDate = parseDate(problems, "startDate", in.StartDate)
	next.EndDate = nil
	if !in.Ongoing {
		next.EndDate = parseDate(problems, "endDate", in.EndDate)
	}
	if before == nil || datesChanged(*before, next) {
		checkDates(problems, next.StartDate, next.EndDate, next.Ongoing, pc.initiative, latestEndDate(pc.plan.Year))
	}
	if before != nil && before.Ongoing && !in.Ongoing {
		problems.Add("ongoing", ongoingLockedMessage)
	}

	next.PlannedCost = nil
	if in.PlannedCost != nil {
		next.PlannedCost = parseAmountField(problems, "plannedCost", *in.PlannedCost, pc.plan.Currency, costNegativeMessage,
			fmt.Sprintf(costCurrencyMessage, pc.plan.Currency))
	}

	next.ApprovalWorkflowID = strings.TrimSpace(deref(in.ApprovalWorkflowID))
	if before == nil || before.ApprovalWorkflowID != next.ApprovalWorkflowID {
		switch {
		case before != nil && before.Status == StatusSubmitted: // an open approval round (008 FR-013)
			problems.Add("approvalWorkflowId", workflowRecallMessage)
		case before != nil && inApproval(*before):
			problems.Add("approvalWorkflowId", workflowLockedMessage)
		case next.ApprovalWorkflowID != "":
			workflow, found, err := s.workflows.GetWorkflow(ctx, next.ApprovalWorkflowID)
			if err != nil {
				return Activity{}, err
			}
			if !found || !workflow.Active {
				problems.Add("approvalWorkflowId", chooseWorkflowMessage)
			}
		}
	}

	next.Description = strings.TrimSpace(in.Description)
	if utf8.RuneCountInString(next.Description) > maxDescriptionLength {
		problems.Add("description", descriptionTooLong)
	}

	values, err := s.checkAttributes(ctx, problems, pc, in.Attributes, before)
	if err != nil {
		return Activity{}, err
	}
	next.Attributes = values

	goals := parseGoals(problems, in.Goals)
	next.INQ, next.SetLaterStagesMyself, next.MQL, next.Revenue = goals.INQ, goals.SetLaterStagesMyself, goals.MQL, goals.Revenue
	return next, nil
}

// checkAttributes checks the Shell attribute values of a save with feature 002's rules and returns the
// values to store. Values of attributes hidden by the plan's business unit are kept even when the save
// leaves them out (FR-015, D-108).
func (s *Service) checkAttributes(ctx context.Context, problems *errs.Problems, pc planContext, values settings.AttributeValues, before *Activity) (settings.AttributeValues, error) {
	previous := settings.AttributeValues{}
	if before != nil && before.Attributes != nil {
		previous = before.Attributes
	}
	if values == nil {
		values = settings.AttributeValues{}
	}
	result, err := s.settings.CheckAttributes(ctx, settings.LevelActivity,
		settings.AttributeInput{Values: values, Previous: previous, BusinessUnit: pc.businessUnit})
	if err != nil {
		return nil, err
	}
	*problems = append(*problems, result.Problems...)
	stored := result.Values
	if stored == nil {
		stored = settings.AttributeValues{}
	}
	attrs, err := s.settings.Attributes(ctx, settings.LevelActivity)
	if err != nil {
		return nil, err
	}
	for _, attr := range attrs {
		_, sent := values[attr.ID]
		if kept, had := previous[attr.ID]; had && !sent && !settings.Visible(attr, pc.businessUnit) {
			stored[attr.ID] = kept
		}
	}
	return stored, nil
}

// parseAmountField reads an amount of a request in currency: at most two decimals (D-190), in currency, not
// negative. A problem is added under field with the first message that applies, and the amount is then nil.
func parseAmountField(problems *errs.Problems, field string, in money.Input, currency, negativeMessage, currencyMessage string) *money.Money {
	amount, err := money.ParseAmount(strings.TrimSpace(in.Amount))
	switch {
	case err != nil:
		problems.Add(field, enterAmountMessage)
	case in.Currency != currency:
		problems.Add(field, currencyMessage)
	case amount.Sign() < 0:
		problems.Add(field, negativeMessage)
	default:
		return &money.Money{Amount: amount, Currency: currency}
	}
	return nil
}

// parseUSD reads a USD amount that may not be negative, with the generic currency message: goals stay in USD
// (016 FR-019).
func parseUSD(problems *errs.Problems, field string, in money.Input, negativeMessage string) *money.Money {
	return parseAmountField(problems, field, in, money.BaseCurrency, negativeMessage, revenueCurrencyMessage)
}

// deref returns the text p points at, or "" for nil.
func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
