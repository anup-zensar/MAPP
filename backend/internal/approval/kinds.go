package approval

import (
	"context"
	"encoding/json"
	"fmt"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
)

// workflowKind is the history kind of approval workflows (data model §1).
const workflowKind = "approvalWorkflow"

// workflowSingular is the mid-sentence name of an approval workflow, as the log, the recycle bin and the
// messages show it.
const workflowSingular = "approval workflow"

// Kinds returns the history kinds of this package for internal/app to register: approval workflows, which
// have no parent and no children, and approval delegations (017), which are logged but never deleted. Rounds, history entries and notifications are not deleted on their own,
// so they have no kind. With it, history moves a workflow to the recycle bin, brings it back, and logs its
// changes with readable labels and texts (FR-008, FR-026).
func (s *Service) Kinds() []history.Kind {
	return []history.Kind{{
		Name:           workflowKind,
		Singular:       workflowSingular,
		Plural:         "approval workflows",
		Snapshot:       s.workflowSnapshot,
		Remove:         s.repo.DeleteWorkflow,
		Put:            s.putWorkflow,
		Describe:       s.describeWorkflow,
		PrepareRestore: s.prepareWorkflowRestore,
	}, s.delegationKindOf()}
}

// workflowSnapshot reads the workflow id as history keeps it: the stored workflow and its name.
func (s *Service) workflowSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	w, found, err := s.repo.GetWorkflow(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	data, err := json.Marshal(w)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	return history.Snapshot{ID: w.ID, Name: w.Name, Data: data}, true, nil
}

// putWorkflow stores the workflow of a snapshot again, exactly as the snapshot holds it.
func (s *Service) putWorkflow(ctx context.Context, snapshot history.Snapshot) error {
	var w Workflow
	if err := json.Unmarshal(snapshot.Data, &w); err != nil {
		return err
	}
	return s.repo.PutWorkflow(ctx, w)
}

// describeWorkflow gives the labels and texts of a workflow's changed fields in the activity log (data
// model §1): the levels as their levels text with the approvers' names now, the active flag as its status.
func (s *Service) describeWorkflow(ctx context.Context, field string, value any) (string, string, error) {
	switch field {
	case "name":
		return "Name", "", nil
	case "description":
		return "Description", "", nil
	case "active":
		if value == true {
			return "Status", activeText, nil
		}
		return "Status", inactiveText, nil
	case "levels":
		data, err := json.Marshal(value)
		if err != nil {
			return "", "", err
		}
		var levels []Level
		if err := json.Unmarshal(data, &levels); err != nil {
			return "", "", err
		}
		rc, err := s.ruleContext(ctx)
		if err != nil {
			return "", "", err
		}
		return "Levels", levelsText(levelViews(normalisedLevels(levels), rc)), nil
	case "reminderDays", "escalationDays":
		label := remindAfterLabel
		if field == "escalationDays" {
			label = escalateAfterLabel
		}
		days, isNumber := value.(json.Number)
		n, err := days.Int64()
		if !isNumber || err != nil {
			return label, offText, nil
		}
		return label, daysText(int(n)), nil
	}
	return "", "", nil
}

// prepareWorkflowRestore applies the name rule of workflows just before one comes back (data model §1): no
// two workflows have the same name. When another workflow has meanwhile taken the name, an Undo, which
// must put the workflow back exactly, is refused with name_taken; a Restore renames it to "<name>
// (restored)", "<name> (restored 2)", … and says so in a note.
func (s *Service) prepareWorkflowRestore(ctx context.Context, rc history.RestoreContext, snapshot *history.Snapshot) ([]string, error) {
	var w Workflow
	if err := json.Unmarshal(snapshot.Data, &w); err != nil {
		return nil, err
	}
	workflows, err := s.repo.ListWorkflows(ctx)
	if err != nil {
		return nil, err
	}
	other, taken := workflowNamed(workflows, w.Name, w.ID)
	if !taken {
		return nil, nil
	}
	if rc.Undo {
		return nil, errs.NameTaken(fmt.Sprintf(undoNameTakenMessage, other.Name))
	}

	candidate := w.Name + " (restored)"
	for n := 2; ; n++ {
		if _, taken := workflowNamed(workflows, candidate, w.ID); !taken {
			break
		}
		candidate = fmt.Sprintf("%s (restored %d)", w.Name, n)
	}
	w.Name = candidate
	w.Touch(rc.Actor, rc.Now) // a changed record gets a new version
	data, err := json.Marshal(w)
	if err != nil {
		return nil, err
	}
	snapshot.Name, snapshot.Data = w.Name, data
	return []string{fmt.Sprintf(renamedOnRestoreMessage, w.Name)}, nil
}
