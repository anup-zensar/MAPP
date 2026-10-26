package access

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/record"
)

// GetRules returns the access rules (FR-019): the stored ones, or the defaults with version 0 while they were
// never saved.
func (s *Service) GetRules(ctx context.Context, a actor.Actor) (RulesView, error) {
	w, err := s.read(ctx)
	if err != nil {
		return RulesView{}, err
	}
	return w.rulesView(), nil
}

// rulesLog is the access rules as the activity log compares them (data model §3).
type rulesLog struct {
	NoSelfApproval  bool   `json:"noSelfApproval"`
	NewPersonRoleID string `json:"newPersonRoleId"`
}

// UpdateRules changes the access rules with the fields update sends (FR-019): segregation of duties on or
// off, and the role for new people — a role that is not Administrator (message 17), or "" for none. The
// version is 0 for the first save, which creates the record. The change is logged on the access rules and
// applies to everyone's next request (FR-020, FR-037).
func (s *Service) UpdateRules(ctx context.Context, a actor.Actor, update RulesUpdate) (RulesView, error) {
	var view RulesView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		w, err := s.read(ctx)
		if err != nil {
			return err
		}
		var problems errs.Problems
		after := w.rules
		if update.NoSelfApproval != nil {
			after.NoSelfApproval = *update.NoSelfApproval
		}
		if update.NewPersonRoleID != nil {
			after.NewPersonRoleID = *update.NewPersonRoleID
			if r, known := w.roles[after.NewPersonRoleID]; after.NewPersonRoleID != "" && !known {
				problems.Add("newPersonRoleId", msgChooseRole)
			} else if known && r.administrator() {
				problems.Add("newPersonRoleId", msgNewPersonNotAdmin)
			}
		}
		if update.Version == nil || *update.Version < 0 {
			problems.Add("version", msgReload)
		}
		if err := problems.Err(); err != nil {
			return err
		}
		if err := w.rules.CheckVersion(*update.Version, rulesSingular); err != nil {
			return err
		}
		if err := s.saveRules(ctx, a, w.rules, after); err != nil {
			return err
		}
		w2, err := s.read(ctx)
		view = w2.rulesView()
		return err
	})
	if err != nil {
		return RulesView{}, err
	}
	return view, nil
}

// saveRules saves the access rules after a change from before, with a new version and one "changed" log
// entry, inside the caller's Write; a change that changes nothing saves nothing.
func (s *Service) saveRules(ctx context.Context, a actor.Actor, before, after Rules) error {
	changes, err := history.Diff(
		rulesLog{NoSelfApproval: before.NoSelfApproval, NewPersonRoleID: before.NewPersonRoleID},
		rulesLog{NoSelfApproval: after.NoSelfApproval, NewPersonRoleID: after.NewPersonRoleID})
	if err != nil || len(changes) == 0 {
		return err
	}
	if after.Version == 0 { // the defaults were never saved: the first change creates the record
		after.Meta = record.NewMeta(a, s.now())
	} else {
		after.Touch(a, s.now())
	}
	after.ID = rulesID
	if err := s.repo.PutRules(ctx, after); err != nil {
		return err
	}
	_, err = s.history.Log(ctx, a, history.LogInput{
		Kind: RulesKind, RecordID: rulesID, RecordName: rulesRecordName, Action: history.ActionChanged, Changes: changes,
	})
	return err
}

// rulesView returns the access rules as the API returns them.
func (w world) rulesView() RulesView {
	view := RulesView{NoSelfApproval: w.rules.NoSelfApproval, Version: w.rules.Version}
	if r, known := w.roles[w.rules.NewPersonRoleID]; known {
		ref := r.ref()
		view.NewPersonRole = &ref
	}
	if w.rules.Version > 0 {
		view.UpdatedAt, view.UpdatedBy = w.rules.UpdatedAt, w.rules.UpdatedBy
	}
	return view
}
