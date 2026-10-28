package bulk

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// undoWindow is the recycle bin's Undo window (D-037), which a bulk delete shares.
const undoWindow = 2 * time.Minute

// runner is one run of a bulk action inside its Write.
type runner struct {
	s       *Service
	a       actor.Actor
	req     Request
	field   Field
	result  Result
	binIDs  []string
	effects consequenceSums
	newID   string
}

func (r *runner) process(ctx context.Context, recordIDs []string) error {
	records := make([]Record, 0, len(recordIDs))
	for _, id := range recordIDs {
		rec, findErr := r.s.planning.Find(ctx, r.a, id)
		if findErr != nil {
			level, _ := levelOf(id)
			if err := r.record(Record{ID: id, Level: level}, findErr); err != nil {
				return err
			}
			continue
		}
		records = append(records, rec)
	}
	if r.req.Action == ActionDelete {
		records = topLevel(records)
		if err := r.summarise(ctx, records); err != nil {
			return err
		}
	}
	for _, rec := range records {
		if err := r.record(rec, r.one(ctx, rec)); err != nil {
			return err
		}
	}
	r.finish()
	return nil
}

// one runs the action on one record; a skip and the operations' *errs.Error say why it was not changed.
func (r *runner) one(ctx context.Context, rec Record) error {
	switch r.req.Action {
	case ActionChangeOwner:
		return changedOrSkipped(r.s.planning.SetOwner(ctx, r.a, rec, r.req.OwnerID))
	case ActionChangeField:
		return r.changeField(ctx, rec)
	case ActionSubmit:
		return r.submit(ctx, rec)
	case ActionCopy:
		return r.copyOne(ctx, rec)
	case ActionMove:
		if err := r.consequences(ctx, rec); err != nil {
			return err
		}
		return r.s.planning.Move(ctx, r.a, rec, r.req.TargetID)
	case ActionDelete:
		return r.deleteOne(ctx, rec)
	}
	return nil
}

func (r *runner) deleteOne(ctx context.Context, rec Record) error {
	entry, err := r.s.planning.Delete(ctx, r.a, rec)
	if err == nil {
		r.result.RecycleBinEntries = append(r.result.RecycleBinEntries, entry.View())
		r.binIDs = append(r.binIDs, entry.ID)
	}
	return err
}

// skip is a refusal of the bulk action itself rather than of the owning operation.
type skip struct{ reason string }

func (s skip) Error() string { return s.reason }

func changedOrSkipped(changed bool, err error) error {
	if err == nil && !changed {
		return skip{reason: reasonAlreadySet}
	}
	return err
}

func (r *runner) changeField(ctx context.Context, rec Record) error {
	needsApproval := rec.Level == LevelActivity && (r.field.Key == "startDate" || r.field.Key == "endDate") &&
		(rec.Status == StatusSubmitted || rec.Status == StatusApproved)
	if needsApproval {
		r.result.Reapproval = append(r.result.Reapproval, refOf(rec))
		if !r.req.RestartApproval {
			return skip{reason: keepReason(rec)}
		}
	}
	return changedOrSkipped(r.s.planning.SetField(ctx, r.a, rec,
		SetField{Field: r.field, Value: r.req.Field.Value, RestartApproval: r.req.RestartApproval}))
}

func keepReason(rec Record) string {
	if rec.Status == StatusSubmitted {
		return reasonKeepApprovers
	}
	return fmt.Sprintf(reasonKeepApproval, rec.Shown)
}

func (r *runner) submit(ctx context.Context, rec Record) error {
	switch {
	case rec.Level != LevelActivity:
		return skip{reason: reasonOnlyActivities}
	case rec.Status == StatusSubmitted:
		return skip{reason: reasonSubmitted}
	case rec.Status == StatusApproved:
		return skip{reason: reasonApproved}
	}
	return r.s.planning.Submit(ctx, r.a, rec, r.req.Comment)
}

func (r *runner) copyOne(ctx context.Context, rec Record) error {
	if err := r.consequences(ctx, rec); err != nil {
		return err
	}
	newID, err := r.s.planning.Copy(ctx, r.a, rec, r.req.TargetID)
	if err == nil {
		r.newID = newID
	}
	return err
}

// record turns the outcome of one record into its line of the result; an error that is neither a skip nor
// an *errs.Error stops the run, so nothing is saved.
func (r *runner) record(rec Record, err error) error {
	o := Outcome{RecordID: rec.ID, Level: rec.Level, Name: rec.Name, PlanID: rec.PlanID, Outcome: OutcomeDone,
		Reasons: []string{}, NewRecordID: r.newID}
	r.newID = ""
	var refused skip
	apiErr, isAPI := errors.AsType[*errs.Error](err)
	switch {
	case err == nil:
		r.result.Counts.Done++
	case errors.As(err, &refused):
		o.Outcome, o.Reasons = OutcomeSkipped, []string{refused.reason}
		r.result.Counts.Skipped++
	case isAPI && apiErr.Code == errs.CodeNotFound:
		o.Outcome, o.Reasons = OutcomeFailed, []string{reasonGone}
		r.result.Counts.Failed++
	case isAPI:
		o.Outcome, o.Reasons = OutcomeSkipped, reasonsOf(rec, apiErr)
		r.result.Counts.Skipped++
	default:
		return err
	}
	r.result.Outcomes = append(r.result.Outcomes, o)
	return nil
}

func (r *runner) finish() {
	r.result.Effects = r.effects.effects(r.req.Action)
	if r.req.Action == ActionDelete && len(r.binIDs) > 0 {
		until := r.s.now().Add(undoWindow)
		r.result.UndoAvailableUntil = &until
	}
}

func (r *runner) stored() Action {
	return Action{ID: *r.result.ID, Kind: r.req.Action, Note: r.result.Note, Actor: r.a, RanAt: r.s.now(),
		BinEntryIDs: slices.Clone(r.binIDs)}
}

// topLevel leaves out every record that has a selected ancestor: it goes with that one (FR-019).
func topLevel(records []Record) []Record {
	selected := map[string]bool{}
	for _, rec := range records {
		selected[rec.ID] = true
	}
	return slices.DeleteFunc(slices.Clone(records), func(rec Record) bool {
		return slices.ContainsFunc(rec.Ancestors, func(id string) bool { return selected[id] })
	})
}

func refOf(rec Record) RecordRef {
	return RecordRef{RecordID: rec.ID, Level: rec.Level, Name: rec.Name, PlanID: rec.PlanID, Status: rec.Shown}
}
