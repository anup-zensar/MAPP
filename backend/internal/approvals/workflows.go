package approvals

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
)

// The longest workflow name and level name, in characters (data model §1).
const (
	maxWorkflowNameLength = 120
	maxLevelNameLength    = 60
)

// ListWorkflows returns every approval workflow, active and inactive, sorted by name ignoring letter case,
// each with its levels and their approvers now, its levels text, how many activities use it and wait for
// approval with it, and its warnings (FR-001, FR-005).
func (s *Service) ListWorkflows(ctx context.Context, a actor.Actor) ([]WorkflowView, error) {
	var views []WorkflowView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		workflows, err := s.repo.ListWorkflows(ctx)
		if err != nil {
			return err
		}
		u, err := s.usage(ctx)
		if err != nil {
			return err
		}
		rc, err := s.ruleContext(ctx)
		if err != nil {
			return err
		}
		slices.SortStableFunc(workflows, func(x, y Workflow) int {
			return cmp.Or(cmp.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name)), cmp.Compare(x.ID, y.ID))
		})
		views = make([]WorkflowView, 0, len(workflows))
		for _, w := range workflows {
			views = append(views, workflowView(w, u, rc))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return views, nil
}

// GetWorkflow returns the approval workflow id as ListWorkflows does, with the activities that use it, by
// plan name and then activity name (FR-005).
func (s *Service) GetWorkflow(ctx context.Context, a actor.Actor, id string) (WorkflowDetails, error) {
	var details WorkflowDetails
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		w, err := s.storedWorkflow(ctx, id)
		if err != nil {
			return err
		}
		u, err := s.usage(ctx)
		if err != nil {
			return err
		}
		rc, err := s.ruleContext(ctx)
		if err != nil {
			return err
		}
		details = WorkflowDetails{Workflow: workflowView(w, u, rc), Activities: []WorkflowUsage{}}
		for _, facts := range u.usedBy[w.ID] {
			details.Activities = append(details.Activities, WorkflowUsage{
				Activity: errs.RecordRef{Type: "activity", ID: facts.ID, Name: facts.Name}, Path: nonNilPath(facts.Path),
				DisplayStatus: facts.DisplayStatus, Waiting: u.waiting[w.ID][facts.ID],
			})
		}
		slices.SortStableFunc(details.Activities, func(x, y WorkflowUsage) int {
			return cmp.Or(cmp.Compare(strings.ToLower(planName(x.Path)), strings.ToLower(planName(y.Path))),
				cmp.Compare(strings.ToLower(x.Activity.Name), strings.ToLower(y.Activity.Name)))
		})
		return nil
	})
	if err != nil {
		return WorkflowDetails{}, err
	}
	return details, nil
}

// CreateWorkflow creates an active approval workflow from in, its texts without surrounding spaces, after
// checking every rule at once (FR-002, FR-003; 017 FR-001 – FR-008, FR-025): the name, the description, the
// reminder and escalation days and each level with its approver source, conditions and backup approver. It
// is logged as created (FR-026).
func (s *Service) CreateWorkflow(ctx context.Context, a actor.Actor, in WorkflowInput) (WorkflowView, error) {
	name, description := strings.TrimSpace(in.Name), strings.TrimSpace(in.Description)
	levels, amountProblems := levelsOf(in.Levels)
	var view WorkflowView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		rc, err := s.ruleContext(ctx)
		if err != nil {
			return err
		}
		w := Workflow{Name: name, Description: description, Active: true, Levels: levels,
			ReminderDays: in.ReminderDays, EscalationDays: in.EscalationDays}
		var problems errs.Problems
		if err := s.checkWorkflow(ctx, &problems, rc, w, amountProblems, Workflow{}); err != nil {
			return err
		}
		if err := problems.Err(); err != nil {
			return err
		}
		if w.ID, err = s.ids.Next(ctx, ids.PrefixApprovalWorkflow); err != nil {
			return err
		}
		w.Meta = record.NewMeta(a, s.now())
		if err := s.repo.PutWorkflow(ctx, w); err != nil {
			return err
		}
		_, err = s.history.Log(ctx, a, history.LogInput{Kind: workflowKind, RecordID: w.ID, RecordName: w.Name, Action: history.ActionCreated})
		if err != nil {
			return err
		}
		view, err = s.viewOf(ctx, w, rc)
		return err
	})
	if err != nil {
		return WorkflowView{}, err
	}
	return view, nil
}

// UpdateWorkflow changes the fields of the approval workflow id that update sends (D-203), levels as a
// whole new list, after checking every rule on the result at once (FR-003). update.Version must be the
// version the person started from (D-030). An approver already in a level may stay there after becoming
// inactive, but may not be added (008-P8). A save that changes nothing makes no new version and no log
// entry; otherwise one "changed" entry lists every changed field (FR-026). Open rounds keep the levels they
// were submitted with (FR-006).
func (s *Service) UpdateWorkflow(ctx context.Context, a actor.Actor, id string, update WorkflowUpdate) (WorkflowView, error) {
	var view WorkflowView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		stored, err := s.storedWorkflow(ctx, id)
		if err != nil {
			return err
		}
		var problems errs.Problems
		if update.Version == nil || *update.Version < 1 {
			problems.Add("version", reloadMessage)
		} else if err := stored.CheckVersion(*update.Version, workflowSingular); err != nil {
			return err
		}
		changed := stored
		if update.Name != nil {
			changed.Name = strings.TrimSpace(*update.Name)
		}
		if update.Description != nil {
			changed.Description = strings.TrimSpace(*update.Description)
		}
		amountProblems := map[string]string{}
		if update.Levels != nil {
			changed.Levels, amountProblems = levelsOf(*update.Levels)
		}
		if update.ReminderDays.Sent {
			changed.ReminderDays = update.ReminderDays.Days
		}
		if update.EscalationDays.Sent {
			changed.EscalationDays = update.EscalationDays.Days
		}
		rc, err := s.ruleContext(ctx)
		if err != nil {
			return err
		}
		if err := s.checkWorkflow(ctx, &problems, rc, changed, amountProblems, stored); err != nil {
			return err
		}
		if err := problems.Err(); err != nil {
			return err
		}

		changes, err := history.Diff(stored, changed)
		if err != nil {
			return err
		}
		if len(changes) == 0 {
			view, err = s.viewOf(ctx, stored, rc)
			return err
		}
		changed.Touch(a, s.now())
		if err := s.repo.PutWorkflow(ctx, changed); err != nil {
			return err
		}
		_, err = s.history.Log(ctx, a, history.LogInput{
			Kind: workflowKind, RecordID: changed.ID, RecordName: changed.Name, Action: history.ActionChanged, Changes: changes,
		})
		if err != nil {
			return err
		}
		view, err = s.viewOf(ctx, changed, rc)
		return err
	})
	if err != nil {
		return WorkflowView{}, err
	}
	return view, nil
}

// SetWorkflowActive sets the approval workflow id inactive (active false) or active again (FR-007), from the
// version the person saw. An inactive workflow is not offered on activities; those that use it keep it, and
// open rounds finish with the levels they were submitted with. Setting the value it already has changes
// nothing; otherwise the change is logged as "deactivated" or "reactivated" (FR-026).
func (s *Service) SetWorkflowActive(ctx context.Context, a actor.Actor, id string, active bool, version *int) (WorkflowView, error) {
	if version == nil || *version < 1 {
		return WorkflowView{}, errs.Validation(errs.FieldError{Field: "version", Message: reloadMessage})
	}
	var view WorkflowView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		stored, err := s.storedWorkflow(ctx, id)
		if err != nil {
			return err
		}
		if err := stored.CheckVersion(*version, workflowSingular); err != nil {
			return err
		}
		rc, err := s.ruleContext(ctx)
		if err != nil {
			return err
		}
		if stored.Active == active {
			view, err = s.viewOf(ctx, stored, rc)
			return err
		}
		changed := stored
		changed.Active = active
		changes, err := history.Diff(stored, changed)
		if err != nil {
			return err
		}
		changed.Touch(a, s.now())
		if err := s.repo.PutWorkflow(ctx, changed); err != nil {
			return err
		}
		verb := "reactivated"
		if !active {
			verb = "deactivated"
		}
		_, err = s.history.Log(ctx, a, history.LogInput{
			Kind: workflowKind, RecordID: changed.ID, RecordName: changed.Name, Action: history.ActionStatusChanged,
			Verb: verb, Changes: changes,
		})
		if err != nil {
			return err
		}
		view, err = s.viewOf(ctx, changed, rc)
		return err
	})
	if err != nil {
		return WorkflowView{}, err
	}
	return view, nil
}

// DeleteWorkflow moves the approval workflow id to the recycle bin and returns the bin entry for Undo
// (FR-008). A workflow that any activity uses is refused with in_use: it can be set inactive instead.
func (s *Service) DeleteWorkflow(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		if _, err := s.storedWorkflow(ctx, id); err != nil {
			return err
		}
		u, err := s.usage(ctx)
		if err != nil {
			return err
		}
		switch n := len(u.usedBy[id]); n {
		case 0:
		case 1:
			return errs.InUse(inUseOneMessage)
		default:
			return errs.InUse(fmt.Sprintf(inUseManyMessage, n))
		}
		entry, err = s.history.MoveToBin(ctx, a, workflowKind, id)
		return err
	})
	if err != nil {
		return history.BinEntry{}, err
	}
	return entry, nil
}

// storedWorkflow returns the stored workflow id, or not_found with the plan's message.
func (s *Service) storedWorkflow(ctx context.Context, id string) (Workflow, error) {
	w, found, err := s.repo.GetWorkflow(ctx, id)
	if err != nil {
		return Workflow{}, err
	}
	if !found {
		return Workflow{}, errs.NotFound(workflowGoneMessage)
	}
	return w, nil
}

// levelsOf returns the levels of a request, normalised (017 plan, Service operations): names and ids without
// surrounding spaces; a level without an approver source has named people, one without a mode "any one
// approves"; approvers are kept only for named people, the role only for role levels, and line-manager
// levels are always "any". Amounts that cannot be read are returned as problems by condition field
// ("levels.<n>.appliesWhen"), so that checkWorkflow reports them with the others. Levels are never nil.
func levelsOf(inputs []LevelInput) ([]Level, map[string]string) {
	levels := make([]Level, 0, len(inputs))
	problems := map[string]string{}
	for i, in := range inputs {
		n := i + 1
		l := Level{
			Name: strings.TrimSpace(in.Name), ApproverType: strings.TrimSpace(in.ApproverType), ApproverIDs: []string{},
			RoleID: strings.TrimSpace(in.RoleID), Mode: strings.TrimSpace(in.Mode), BackupApproverID: strings.TrimSpace(in.BackupApproverID),
		}
		if l.ApproverType == "" {
			l.ApproverType = SourcePeople
		}
		if l.Mode == "" || l.ApproverType == SourceManager {
			l.Mode = ModeAny
		}
		if l.ApproverType == SourcePeople && in.ApproverIDs != nil {
			l.ApproverIDs = in.ApproverIDs
		}
		if l.ApproverType != SourceRole {
			l.RoleID = ""
		}
		var bad bool
		if l.AppliesWhen, bad = conditionOf(in.AppliesWhen); bad {
			problems[fmt.Sprintf("levels.%d.appliesWhen", n)] = fmt.Sprintf(amountsMessage, n)
		}
		if l.AutoApproveWhen, bad = conditionOf(in.AutoApproveWhen); bad {
			problems[fmt.Sprintf("levels.%d.autoApproveWhen", n)] = fmt.Sprintf(amountsMessage, n)
		}
		levels = append(levels, l)
	}
	return levels, problems
}

// conditionOf reads a condition as sent: nil stays nil; empty amount texts are no part. bad is true when an
// amount is not a decimal number with at most two decimals.
func conditionOf(in *ConditionInput) (*Condition, bool) {
	if in == nil {
		return nil, false
	}
	c := Condition{BusinessUnitIDs: nonNilIDs(in.BusinessUnitIDs), ActivityTypeIDs: nonNilIDs(in.ActivityTypeIDs)}
	bad := false
	for _, part := range []struct {
		text *string
		into **money.Amount
	}{{in.PlannedCostAtLeast, &c.PlannedCostAtLeast}, {in.PlannedCostBelow, &c.PlannedCostBelow}} {
		if part.text == nil || strings.TrimSpace(*part.text) == "" {
			continue
		}
		amount, err := money.ParseAmount(strings.TrimSpace(*part.text))
		if err != nil {
			bad = true
			continue
		}
		*part.into = &amount
	}
	return &c, bad
}

// nonNilIDs returns ids, or an empty list for nil.
func nonNilIDs(ids []string) []string {
	if ids == nil {
		return []string{}
	}
	return ids
}

// The day counts of reminders and escalation (017 FR-007).
const (
	maxReminderDays   = 60
	maxEscalationDays = 90
)

// checkWorkflow adds every problem of the workflow w to problems (FR-002, FR-003; 017 FR-008, FR-025, data
// model §1–§2); amountProblems are the amounts levelsOf could not read; stored is the workflow before an
// update (its id "" on a create), whose inactive approvers and backup approvers may stay in their level.
func (s *Service) checkWorkflow(ctx context.Context, problems *errs.Problems, rc ruleContext, w Workflow,
	amountProblems map[string]string, stored Workflow) error {
	switch {
	case w.Name == "":
		problems.Add("name", enterNameMessage)
	case tooLong(w.Name, maxWorkflowNameLength):
		problems.Add("name", nameTooLongMessage)
	default:
		workflows, err := s.repo.ListWorkflows(ctx)
		if err != nil {
			return err
		}
		if other, taken := workflowNamed(workflows, w.Name, stored.ID); taken {
			problems.Add("name", fmt.Sprintf(nameTakenMessage, other.Name))
		}
	}
	if tooLong(w.Description, maxCommentLength) {
		problems.Add("description", tooLong2000Message)
	}
	remindOK := w.ReminderDays == nil || (*w.ReminderDays >= 1 && *w.ReminderDays <= maxReminderDays)
	if !remindOK {
		problems.Add("reminderDays", reminderDaysMessage)
	}
	switch {
	case w.EscalationDays == nil:
	case *w.EscalationDays < 1 || *w.EscalationDays > maxEscalationDays:
		problems.Add("escalationDays", escalationDaysMessage)
	case remindOK && w.ReminderDays != nil && *w.EscalationDays <= *w.ReminderDays:
		problems.Add("escalationDays", escalationOrderMessage)
	}
	if len(w.Levels) == 0 {
		problems.Add("levels", addLevelMessage)
	}
	for i, l := range w.Levels {
		var before Level // the level at this position before the update: its people may stay when inactive
		if i < len(stored.Levels) {
			before = stored.Levels[i]
		}
		checkLevel(problems, rc, i+1, l, before, w.EscalationDays != nil, amountProblems)
	}
	return nil
}

// checkLevel adds every problem of the level l at position n to problems (data-model §2), in the order of
// its fields. before is the level at that position before an update.
func checkLevel(problems *errs.Problems, rc ruleContext, n int, l, before Level, escalates bool, amountProblems map[string]string) {
	d := rc.people
	field := func(name string) string { return fmt.Sprintf("levels.%d.%s", n, name) }
	if tooLong(l.Name, maxLevelNameLength) {
		problems.Add(field("name"), fmt.Sprintf(levelNameTooLongMessage, n))
	}
	switch l.ApproverType {
	case SourcePeople, SourceManager, SourceRole:
	default:
		problems.Add(field("approverType"), fmt.Sprintf(chooseSourceMessage, n))
	}

	if l.ApproverType == SourcePeople {
		approvers := field("approverIds")
		if len(l.ApproverIDs) == 0 {
			problems.Add(approvers, fmt.Sprintf(chooseApproverMessage, n))
		}
		seen, unknown := map[string]bool{}, false
		for _, id := range l.ApproverIDs {
			p, found := d.byID[id]
			added := !slices.Contains(before.ApproverIDs, id)
			switch {
			case !found:
				unknown = true
			case seen[id]:
				problems.Add(approvers, fmt.Sprintf(listedTwiceMessage, n, p.Name))
			case !p.Active && added:
				problems.Add(approvers, fmt.Sprintf(inactiveApproverMessage, n, p.Name))
			case !rc.canApprove[id] && added:
				problems.Add(approvers, fmt.Sprintf(approverCannotApproveMessage, p.Name))
			}
			seen[id] = true
		}
		if unknown {
			problems.Add(approvers, fmt.Sprintf(unknownApproverMessage, n))
		}
	}

	if l.ApproverType == SourceRole {
		role, known := rc.roles[l.RoleID]
		switch {
		case l.RoleID == "":
			problems.Add(field("roleId"), fmt.Sprintf(chooseRoleMessage, n))
		case !known:
			problems.Add(field("roleId"), fmt.Sprintf(roleFromListMessage, n))
		case !role.Approves:
			problems.Add(field("roleId"), fmt.Sprintf(roleCannotApproveMessage, n, role.Name))
		}
	}
	if l.Mode != ModeAny && l.Mode != ModeAll {
		problems.Add(field("mode"), fmt.Sprintf(chooseModeMessage, n))
	}

	for _, c := range []struct {
		name, emptyMessage string
		condition          *Condition
	}{{"appliesWhen", emptyAppliesWhenMessage, l.AppliesWhen}, {"autoApproveWhen", emptyAutoApproveMessage, l.AutoApproveWhen}} {
		if message, bad := amountProblems[field(c.name)]; bad {
			problems.Add(field(c.name), message)
			continue
		}
		if c.condition == nil {
			continue
		}
		if message := conditionProblem(rc, n, *c.condition, c.emptyMessage); message != "" {
			problems.Add(field(c.name), message)
		}
	}

	backup := field("backupApproverId")
	if l.BackupApproverID == "" {
		if escalates {
			problems.Add(backup, fmt.Sprintf(backupRequiredMessage, n))
		}
		return
	}
	p, found := d.byID[l.BackupApproverID]
	changed := l.BackupApproverID != before.BackupApproverID
	switch {
	case !found:
		problems.Add(backup, fmt.Sprintf(backupFromListMessage, n))
	case !p.Active && changed:
		problems.Add(backup, fmt.Sprintf(backupInactiveMessage, n, p.Name))
	case !rc.canApprove[p.ID] && changed:
		problems.Add(backup, fmt.Sprintf(backupCannotApproveMessage, n, p.Name))
	case l.ApproverType == SourcePeople && slices.Contains(l.ApproverIDs, p.ID):
		problems.Add(backup, fmt.Sprintf(backupIsApproverMessage, n, p.Name))
	}
}

// conditionProblem returns the first problem of the condition c of the level n (data-model §2: one message
// per condition field), or "".
func conditionProblem(rc ruleContext, n int, c Condition, emptyMessage string) string {
	switch {
	case c.empty():
		return fmt.Sprintf(emptyMessage, n)
	case (c.PlannedCostAtLeast != nil && c.PlannedCostAtLeast.Sign() < 0) || (c.PlannedCostBelow != nil && c.PlannedCostBelow.Sign() < 0):
		return fmt.Sprintf(amountsMessage, n)
	case c.PlannedCostAtLeast != nil && c.PlannedCostBelow != nil && c.PlannedCostAtLeast.Cmp(*c.PlannedCostBelow) >= 0:
		return fmt.Sprintf(amountOrderMessage, n)
	case !knownOnce(c.BusinessUnitIDs, rc.names.businessUnits):
		return fmt.Sprintf(businessUnitsMessage, n)
	case !knownOnce(c.ActivityTypeIDs, rc.names.activityTypes):
		return fmt.Sprintf(activityTypesMessage, n)
	}
	return ""
}

// knownOnce reports whether every id is one of choices, each listed once. Inactive choices may be listed.
func knownOnce(ids []string, choices map[string]Choice) bool {
	seen := map[string]bool{}
	for _, id := range ids {
		if _, known := choices[id]; !known || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}

// workflowNamed returns the workflow among workflows, other than exceptID, whose name is name ignoring
// letter case, and false when there is none. Names are stored without surrounding spaces.
func workflowNamed(workflows []Workflow, name, exceptID string) (Workflow, bool) {
	for _, w := range workflows {
		if w.ID != exceptID && strings.EqualFold(w.Name, name) {
			return w, true
		}
	}
	return Workflow{}, false
}

// usages is who uses each workflow (data model §1): by workflow id, the stored activities that have it in
// their Approval workflow field or have a round with it, in the order of the activities, and those of them
// with an open round of it.
type usages struct {
	usedBy  map[string][]ActivityFacts
	waiting map[string]map[string]bool // workflow id → activity id → waiting
}

// usage reads who uses each workflow, inside the caller's transaction.
func (s *Service) usage(ctx context.Context) (usages, error) {
	list, err := s.activities.List(ctx)
	if err != nil {
		return usages{}, err
	}
	rounds, err := s.repo.ListRounds(ctx)
	if err != nil {
		return usages{}, err
	}
	byActivity := map[string][]Round{}
	for _, r := range rounds {
		byActivity[r.ActivityID] = append(byActivity[r.ActivityID], r)
	}
	u := usages{usedBy: map[string][]ActivityFacts{}, waiting: map[string]map[string]bool{}}
	for _, facts := range list {
		uses := map[string]bool{}
		if facts.WorkflowID != "" {
			uses[facts.WorkflowID] = true
		}
		for _, r := range byActivity[facts.ID] {
			uses[r.WorkflowID] = true
			if r.Outcome == OutcomeOpen {
				if u.waiting[r.WorkflowID] == nil {
					u.waiting[r.WorkflowID] = map[string]bool{}
				}
				u.waiting[r.WorkflowID][facts.ID] = true
			}
		}
		for id := range uses {
			u.usedBy[id] = append(u.usedBy[id], facts)
		}
	}
	return u, nil
}

// viewOf returns w as the API shows it, reading its usage inside the caller's transaction.
func (s *Service) viewOf(ctx context.Context, w Workflow, rc ruleContext) (WorkflowView, error) {
	u, err := s.usage(ctx)
	if err != nil {
		return WorkflowView{}, err
	}
	return workflowView(w, u, rc), nil
}

// workflowView returns w as the API shows it: its levels with their rules and approvers now, the levels
// text, its timings, its usage and its warnings — a named-people level without an active approver, a
// backup approver who is inactive or cannot approve, a role that no longer exists (FR-005; 017 FR-009).
func workflowView(w Workflow, u usages, rc ruleContext) WorkflowView {
	levels := levelViews(w.Levels, rc)
	warnings := []string{}
	for i, l := range w.Levels {
		n := i + 1
		view := levels[i]
		if l.ApproverType == SourcePeople && !slices.ContainsFunc(view.Approvers, func(p PersonView) bool { return p.Active }) {
			warnings = append(warnings, fmt.Sprintf(noActiveApproverWarning, n))
		}
		if b := view.BackupApprover; b != nil && !b.Active {
			warnings = append(warnings, fmt.Sprintf(backupInactiveWarning, n, b.Name))
		} else if b != nil && !rc.canApprove[b.ID] {
			warnings = append(warnings, fmt.Sprintf(backupCannotApproveWarning, n, b.Name))
		}
		if _, known := rc.roles[l.RoleID]; l.ApproverType == SourceRole && !known {
			warnings = append(warnings, fmt.Sprintf(roleGoneWarning, n, l.RoleID))
		}
	}
	return WorkflowView{
		ID: w.ID, Name: w.Name, Description: w.Description, Active: w.Active, Levels: levels, LevelsText: levelsText(levels),
		UsedByCount: len(u.usedBy[w.ID]), WaitingCount: len(u.waiting[w.ID]), Warnings: warnings,
		ReminderDays: w.ReminderDays, EscalationDays: w.EscalationDays, Meta: w.Meta,
	}
}

// ruleContext is what the views and checks of workflows read once per operation: the people, whether each
// can approve (014), the roles, and the labels of the values conditions name.
type ruleContext struct {
	people        directory
	canApprove    map[string]bool
	roles         map[string]RoleInfo
	names         choiceNames
	activityTypes []Choice
	businessUnits []Choice
}

// ruleContext reads the rule context inside the caller's transaction.
func (s *Service) ruleContext(ctx context.Context) (ruleContext, error) {
	d, err := s.directory(ctx)
	if err != nil {
		return ruleContext{}, err
	}
	everyone := make([]string, 0, len(d.byID))
	for id := range d.byID {
		everyone = append(everyone, id)
	}
	slices.Sort(everyone)
	can, err := s.access.CanApprove(ctx, everyone)
	if err != nil {
		return ruleContext{}, err
	}
	roles, err := s.roles(ctx)
	if err != nil {
		return ruleContext{}, err
	}
	types, err := s.choices.ActivityTypes(ctx)
	if err != nil {
		return ruleContext{}, err
	}
	units, err := s.choices.BusinessUnits(ctx)
	if err != nil {
		return ruleContext{}, err
	}
	return ruleContext{people: d, canApprove: can, roles: roles, names: namesOf(units, types), activityTypes: types, businessUnits: units}, nil
}

// levelViews returns a workflow's levels as the API shows them, with their approvers' names now and their
// rules in words; never nil.
func levelViews(levels []Level, rc ruleContext) []LevelView {
	d := rc.people
	views := make([]LevelView, 0, len(levels))
	for i, l := range levels {
		n := i + 1
		view := LevelView{
			Number: n, Name: l.Name, Label: levelLabel(l.Name, n), Approvers: make([]PersonView, 0, len(l.ApproverIDs)),
			ApproverType: l.ApproverType, Mode: l.Mode,
			AppliesWhen: conditionView(l.AppliesWhen, rc), AutoApproveWhen: conditionView(l.AutoApproveWhen, rc),
		}
		for _, id := range l.ApproverIDs {
			view.Approvers = append(view.Approvers, d.person(id))
		}
		if l.BackupApproverID != "" {
			backup := d.person(l.BackupApproverID)
			view.BackupApprover = &backup
		}
		var source string
		switch l.ApproverType {
		case SourceManager:
			source = "Line manager"
		case SourceRole:
			role, known := rc.roles[l.RoleID]
			if !known {
				role.RoleRef = RoleRef{ID: l.RoleID, Name: l.RoleID}
			}
			ref := role.RoleRef
			view.Role = &ref
			source = "Role: " + role.Name
		default:
			names := make([]string, 0, len(view.Approvers))
			for _, p := range view.Approvers {
				if p.Active {
					names = append(names, p.Name)
				} else {
					names = append(names, p.Name+" (inactive)")
				}
			}
			source = strings.Join(names, ", ")
		}
		view.Summary = levelSummary(l, n, source, rc.names)
		views = append(views, view)
	}
	return views
}

// levelsText writes the levels in one line, their summaries joined with " → " (data-model §2, FR-005):
// "Level 1 · Line manager · automatic when planned cost is below $10,000.00 → Level 2 · Alex Morgan".
func levelsText(levels []LevelView) string {
	parts := make([]string, 0, len(levels))
	for _, l := range levels {
		parts = append(parts, l.Summary)
	}
	return strings.Join(parts, " → ")
}

// conditionView returns a condition as the API answers it, with its values' labels and its text; nil for
// none.
func conditionView(c *Condition, rc ruleContext) *ConditionView {
	if c == nil {
		return nil
	}
	view := &ConditionView{BusinessUnits: []OptionRef{}, ActivityTypes: []SettingRef{}, Text: conditionText(*c, rc.names)}
	if c.PlannedCostAtLeast != nil {
		usd := money.USD(*c.PlannedCostAtLeast)
		view.PlannedCostAtLeast = &usd
	}
	if c.PlannedCostBelow != nil {
		usd := money.USD(*c.PlannedCostBelow)
		view.PlannedCostBelow = &usd
	}
	for _, id := range c.BusinessUnitIDs {
		choice, known := rc.names.businessUnits[id]
		if !known {
			choice = Choice{ID: id, Label: id}
		}
		view.BusinessUnits = append(view.BusinessUnits, OptionRef{ID: id, Label: choice.Label, Active: choice.Active})
	}
	for _, id := range c.ActivityTypeIDs {
		choice, known := rc.names.activityTypes[id]
		if !known {
			choice = Choice{ID: id, Label: id}
		}
		view.ActivityTypes = append(view.ActivityTypes, SettingRef{ID: id, Name: choice.Label, Active: choice.Active})
	}
	return view
}

// planName is the name of the plan of a path, "" when the path has none.
func planName(path []errs.RecordRef) string {
	if len(path) == 0 {
		return ""
	}
	return path[0].Name
}
