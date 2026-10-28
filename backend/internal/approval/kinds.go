package approval

import (
	"context"
	"encoding/json"
	"fmt"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
)

const workflowKind = "approvalWorkflow"

const workflowSingular = "approval workflow"

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

func (s *Service) putWorkflow(ctx context.Context, snapshot history.Snapshot) error {
	var w Workflow
	if err := json.Unmarshal(snapshot.Data, &w); err != nil {
		return err
	}
	return s.repo.PutWorkflow(ctx, w)
}

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
	case objectTypeField:
		kind, _ := value.(string)
		return "Approves", SubjectKind(kind).Label(), nil
	case "levels":
		return s.describeLevels(ctx, value)
	case "reminderDays", "escalationDays":
		return describeDays(field, value)
	}
	return "", "", nil
}

// describeLevels writes changed levels as their levels text; the log has no record type, so it uses the
// activity wording of conditions (031 research R7).
func (s *Service) describeLevels(ctx context.Context, value any) (string, string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", "", err
	}
	var levels []Level
	if err = json.Unmarshal(data, &levels); err != nil {
		return "", "", err
	}
	rc, err := s.ruleContext(ctx)
	if err != nil {
		return "", "", err
	}
	return "Levels", levelsText(levelViews(normalisedLevels(levels), rc, KindActivity)), nil
}

func describeDays(field string, value any) (string, string, error) {
	label := remindAfterLabel
	if field == "escalationDays" {
		label = escalateAfterLabel
	}
	days, isNumber := value.(json.Number)
	if !isNumber {
		return label, offText, nil
	}
	n, err := days.Int64()
	if err != nil {
		return "", "", err
	}
	return label, daysText(int(n)), nil
}

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
	w.Touch(rc.Actor, rc.Now)
	data, err := json.Marshal(w)
	if err != nil {
		return nil, err
	}
	snapshot.Name, snapshot.Data = w.Name, data
	return []string{fmt.Sprintf(renamedOnRestoreMessage, w.Name)}, nil
}
