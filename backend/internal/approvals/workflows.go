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
		d, err := s.directory(ctx)
		if err != nil {
			return err
		}
		slices.SortStableFunc(workflows, func(x, y Workflow) int {
			return cmp.Or(cmp.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name)), cmp.Compare(x.ID, y.ID))
		})
		views = make([]WorkflowView, 0, len(workflows))
		for _, w := range workflows {
			views = append(views, workflowView(w, u, d))
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
		d, err := s.directory(ctx)
		if err != nil {
			return err
		}
		details = WorkflowDetails{Workflow: workflowView(w, u, d), Activities: []WorkflowUsage{}}
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
// checking every rule at once (FR-002, FR-003): the name, the description and each level with its
// approvers. It is logged as created (FR-026).
func (s *Service) CreateWorkflow(ctx context.Context, a actor.Actor, in WorkflowInput) (WorkflowView, error) {
	name, description, levels := strings.TrimSpace(in.Name), strings.TrimSpace(in.Description), levelsOf(in.Levels)
	var view WorkflowView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		d, err := s.directory(ctx)
		if err != nil {
			return err
		}
		var problems errs.Problems
		if err := s.checkWorkflow(ctx, &problems, d, name, description, levels, Workflow{}); err != nil {
			return err
		}
		if err := problems.Err(); err != nil {
			return err
		}
		id, err := s.ids.Next(ctx, ids.PrefixApprovalWorkflow)
		if err != nil {
			return err
		}
		w := Workflow{ID: id, Name: name, Description: description, Active: true, Levels: levels, Meta: record.NewMeta(a, s.now())}
		if err := s.repo.PutWorkflow(ctx, w); err != nil {
			return err
		}
		_, err = s.history.Log(ctx, a, history.LogInput{Kind: workflowKind, RecordID: w.ID, RecordName: w.Name, Action: history.ActionCreated})
		if err != nil {
			return err
		}
		view, err = s.viewOf(ctx, w, d)
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
		if update.Levels != nil {
			changed.Levels = levelsOf(*update.Levels)
		}
		d, err := s.directory(ctx)
		if err != nil {
			return err
		}
		if err := s.checkWorkflow(ctx, &problems, d, changed.Name, changed.Description, changed.Levels, stored); err != nil {
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
			view, err = s.viewOf(ctx, stored, d)
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
		view, err = s.viewOf(ctx, changed, d)
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
		d, err := s.directory(ctx)
		if err != nil {
			return err
		}
		if stored.Active == active {
			view, err = s.viewOf(ctx, stored, d)
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
		view, err = s.viewOf(ctx, changed, d)
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

// levelsOf returns the levels of a request, names without surrounding spaces; never nil.
func levelsOf(inputs []LevelInput) []Level {
	levels := make([]Level, 0, len(inputs))
	for _, in := range inputs {
		approvers := in.ApproverIDs
		if approvers == nil {
			approvers = []string{}
		}
		levels = append(levels, Level{Name: strings.TrimSpace(in.Name), ApproverIDs: approvers})
	}
	return levels
}

// checkWorkflow adds every problem of a workflow with name, description and levels to problems (FR-002,
// FR-003, data model §1); stored is the workflow before an update (its id "" on a create), whose inactive
// approvers may stay in their level.
func (s *Service) checkWorkflow(ctx context.Context, problems *errs.Problems, d directory, name, description string,
	levels []Level, stored Workflow) error {
	switch {
	case name == "":
		problems.Add("name", enterNameMessage)
	case tooLong(name, maxWorkflowNameLength):
		problems.Add("name", nameTooLongMessage)
	default:
		workflows, err := s.repo.ListWorkflows(ctx)
		if err != nil {
			return err
		}
		if other, taken := workflowNamed(workflows, name, stored.ID); taken {
			problems.Add("name", fmt.Sprintf(nameTakenMessage, other.Name))
		}
	}
	if tooLong(description, maxCommentLength) {
		problems.Add("description", tooLong2000Message)
	}
	if len(levels) == 0 {
		problems.Add("levels", addLevelMessage)
	}
	for i, l := range levels {
		n := i + 1
		if tooLong(l.Name, maxLevelNameLength) {
			problems.Add(fmt.Sprintf("levels.%d.name", n), fmt.Sprintf(levelNameTooLongMessage, n))
		}
		field := fmt.Sprintf("levels.%d.approverIds", n)
		if len(l.ApproverIDs) == 0 {
			problems.Add(field, fmt.Sprintf(chooseApproverMessage, n))
		}
		var kept []string // approvers of this level before the update: they may stay when inactive
		if i < len(stored.Levels) {
			kept = stored.Levels[i].ApproverIDs
		}
		seen, unknown := map[string]bool{}, false
		for _, id := range l.ApproverIDs {
			p, found := d[id]
			switch {
			case !found:
				unknown = true
			case seen[id]:
				problems.Add(field, fmt.Sprintf(listedTwiceMessage, n, p.Name))
			case !p.Active && !slices.Contains(kept, id):
				problems.Add(field, fmt.Sprintf(inactiveApproverMessage, n, p.Name))
			}
			seen[id] = true
		}
		if unknown {
			problems.Add(field, fmt.Sprintf(unknownApproverMessage, n))
		}
	}
	return nil
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
func (s *Service) viewOf(ctx context.Context, w Workflow, d directory) (WorkflowView, error) {
	u, err := s.usage(ctx)
	if err != nil {
		return WorkflowView{}, err
	}
	return workflowView(w, u, d), nil
}

// workflowView returns w as the API shows it: its levels with their approvers now, the levels text, its
// usage and a warning for each level without an active approver (FR-005, edge case).
func workflowView(w Workflow, u usages, d directory) WorkflowView {
	levels := levelViews(w.Levels, d)
	warnings := []string{}
	for _, l := range levels {
		if !slices.ContainsFunc(l.Approvers, func(p PersonView) bool { return p.Active }) {
			warnings = append(warnings, fmt.Sprintf(noActiveApproverWarning, l.Number))
		}
	}
	return WorkflowView{
		ID: w.ID, Name: w.Name, Description: w.Description, Active: w.Active, Levels: levels, LevelsText: levelsText(levels),
		UsedByCount: len(u.usedBy[w.ID]), WaitingCount: len(u.waiting[w.ID]), Warnings: warnings, Meta: w.Meta,
	}
}

// planName is the name of the plan of a path, "" when the path has none.
func planName(path []errs.RecordRef) string {
	if len(path) == 0 {
		return ""
	}
	return path[0].Name
}
