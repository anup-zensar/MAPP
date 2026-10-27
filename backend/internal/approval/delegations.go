package approval

import (
	"context"
	"fmt"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

// Approval delegations (017 FR-022 – FR-024, data-model §7): a person delegates their approvals to one other
// person who can approve, for an inclusive range of UTC calendar dates. Delegations are cancelled, never
// deleted, and kept for the audit trail (017-P9).

// delegationKind is the history kind of approval delegations (017-P14).
const delegationKind = "approvalDelegation"

// delegationSingular is the mid-sentence name of a delegation, as the log and the messages show it.
const delegationSingular = "approval delegation"

// The limits of a delegation (data-model §7).
const (
	maxDelegationDays = 90
	maxNoteLength     = 500
)

// delegationView returns dl as the API answers it to a; manage says whether a holds workflows.manage.
func delegationView(dl Delegation, d directory, a actor.Actor, manage bool, today date.Date) DelegationView {
	state := dl.state(today)
	return DelegationView{
		ID: dl.ID, Person: d.person(dl.PersonID), Delegate: d.person(dl.DelegateID), StartDate: dl.StartDate, EndDate: dl.EndDate,
		Note: dl.Note, State: state, CanChange: (dl.PersonID == a.ID || manage) && (state == DelegationPlanned || state == DelegationCurrent),
		CancelledAt: dl.CancelledAt, CancelledBy: dl.CancelledBy, Meta: dl.Meta,
	}
}

// delegationForm is what a create or update sends, as texts, before it is checked.
type delegationForm struct {
	personID, delegateID, startDate, endDate, note string
}

// CreateDelegation delegates the approvals of in.PersonID (a when empty) to in.DelegateID for a date range
// (017 FR-022): someone else's needs workflows.manage (403). Every problem of data-model §7 is reported at
// once; an overlap with another delegation of the same person is refused. It is logged as created.
func (s *Service) CreateDelegation(ctx context.Context, a actor.Actor, in DelegationInput) (DelegationView, error) {
	form := delegationForm{personID: strings.TrimSpace(in.PersonID), delegateID: strings.TrimSpace(in.DelegateID),
		startDate: in.StartDate, endDate: in.EndDate, note: in.Note}
	if form.personID == "" {
		form.personID = a.ID
	}
	var view DelegationView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		manage, err := s.access.Check(ctx, a.ID, PermissionManageWorkflows, nil)
		if err != nil {
			return err
		}
		if form.personID != a.ID && !manage.Allowed {
			return errs.Forbidden(manage.Refusal)
		}
		d, err := s.directory(ctx)
		if err != nil {
			return err
		}
		dl, err := s.checkDelegation(ctx, a, d, form, nil)
		if err != nil {
			return err
		}
		if dl.ID, err = s.ids.Next(ctx, ids.PrefixApprovalDelegation); err != nil {
			return err
		}
		dl.Meta = record.NewMeta(a, s.now())
		if err := s.repo.PutDelegation(ctx, dl); err != nil {
			return err
		}
		_, err = s.history.Log(ctx, a, history.LogInput{
			Kind: delegationKind, RecordID: dl.ID, RecordName: delegationName(dl, d), Action: history.ActionCreated,
		})
		if err != nil {
			return err
		}
		view = delegationView(dl, d, a, manage.Allowed, date.Today(s.now()))
		return nil
	})
	if err != nil {
		return DelegationView{}, err
	}
	return view, nil
}

// UpdateDelegation changes the fields of the delegation id that update sends (017 FR-024, D-203): only the
// delegating person or someone holding workflows.manage (403), only while planned or current (409
// not_allowed), from the version the person saw. A current delegation keeps its first day. Every problem is
// reported at once; a change is logged with the old and new values.
func (s *Service) UpdateDelegation(ctx context.Context, a actor.Actor, id string, update DelegationUpdate) (DelegationView, error) {
	var view DelegationView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		stored, manage, err := s.changeableDelegation(ctx, a, id, update.Version)
		if err != nil {
			return err
		}
		form := delegationForm{personID: stored.PersonID, delegateID: stored.DelegateID, startDate: stored.StartDate.String(),
			endDate: stored.EndDate.String(), note: stored.Note}
		if update.DelegateID != nil {
			form.delegateID = strings.TrimSpace(*update.DelegateID)
		}
		if update.StartDate != nil {
			form.startDate = *update.StartDate
		}
		if update.EndDate != nil {
			form.endDate = *update.EndDate
		}
		if update.Note != nil {
			form.note = *update.Note
		}
		d, err := s.directory(ctx)
		if err != nil {
			return err
		}
		changed, err := s.checkDelegation(ctx, a, d, form, &stored)
		if err != nil {
			return err
		}
		changes, err := history.Diff(stored.logView(), changed.logView())
		if err != nil {
			return err
		}
		if len(changes) > 0 {
			changed.Touch(a, s.now())
			if err := s.repo.PutDelegation(ctx, changed); err != nil {
				return err
			}
			_, err = s.history.Log(ctx, a, history.LogInput{
				Kind: delegationKind, RecordID: changed.ID, RecordName: delegationName(changed, d), Action: history.ActionChanged, Changes: changes,
			})
			if err != nil {
				return err
			}
		}
		view = delegationView(changed, d, a, manage, date.Today(s.now()))
		return nil
	})
	if err != nil {
		return DelegationView{}, err
	}
	return view, nil
}

// changeableDelegation returns the stored delegation id if a may change it now, and whether a holds
// workflows.manage: gone → 404; not the delegating person nor a manager → 403; ended or cancelled → 409
// not_allowed; no version → 400; another version → version_conflict.
func (s *Service) changeableDelegation(ctx context.Context, a actor.Actor, id string, version *int) (Delegation, bool, error) {
	stored, found, err := s.repo.GetDelegation(ctx, id)
	if err != nil {
		return Delegation{}, false, err
	}
	if !found {
		return Delegation{}, false, errs.NotFound(delegationGoneMessage)
	}
	manage, err := s.access.Check(ctx, a.ID, PermissionManageWorkflows, nil)
	if err != nil {
		return Delegation{}, false, err
	}
	if stored.PersonID != a.ID && !manage.Allowed {
		d, err := s.directory(ctx)
		if err != nil {
			return Delegation{}, false, err
		}
		return Delegation{}, false, errs.Forbidden(fmt.Sprintf(changeDelegationMessage, d.person(stored.PersonID).Name))
	}
	if state := stored.state(date.Today(s.now())); state == DelegationEnded || state == DelegationCancelled {
		return Delegation{}, false, errs.NotAllowed(delegationEndedMessage)
	}
	if version == nil || *version < 1 {
		return Delegation{}, false, errs.Validation(errs.FieldError{Field: versionField, Message: reloadMessage})
	}
	if err := stored.CheckVersion(*version, delegationSingular); err != nil {
		return Delegation{}, false, err
	}
	return stored, manage.Allowed, nil
}

// checkDelegation returns the delegation the form describes, or every problem of data-model §7 at once:
// the delegating person known and active; the delegate chosen, known, not the delegating person, active and
// holding approvals.decide; the first day today or later (and unchanged once the delegation is current);
// the last day on or after the first, today or later, at most 90 days in all; a note of at most 500
// characters; and no day shared with another delegation of the same person that is not cancelled. stored is
// the delegation before an update, nil on a create.
func (s *Service) checkDelegation(ctx context.Context, a actor.Actor, d directory, form delegationForm, stored *Delegation) (Delegation, error) {
	today := date.Today(s.now())
	var problems errs.Problems
	dl := Delegation{PersonID: form.personID, DelegateID: form.delegateID, Note: strings.TrimSpace(form.note)}
	if stored != nil {
		dl = *stored
		dl.DelegateID, dl.Note = form.delegateID, strings.TrimSpace(form.note)
	}

	person, personKnown := d.byID[dl.PersonID]
	switch {
	case !personKnown:
		problems.Add("personId", choosePersonMessage)
	case !person.Active && stored == nil:
		problems.Add("personId", fmt.Sprintf(personInactiveMessage, person.Name))
	}

	delegate, delegateKnown := d.byID[dl.DelegateID]
	delegateChanged := stored == nil || stored.DelegateID != dl.DelegateID
	switch {
	case dl.DelegateID == "":
		problems.Add("delegateId", chooseDelegateMessage)
	case !delegateKnown:
		problems.Add("delegateId", choosePersonMessage)
	case dl.DelegateID == dl.PersonID && dl.PersonID == a.ID:
		problems.Add("delegateId", delegateSelfMessage)
	case dl.DelegateID == dl.PersonID:
		problems.Add("delegateId", fmt.Sprintf(delegateThemselvesMessage, person.Name))
	case !delegate.Active && delegateChanged:
		problems.Add("delegateId", fmt.Sprintf(personInactiveMessage, delegate.Name))
	case delegateChanged:
		can, err := s.access.CanApprove(ctx, []string{dl.DelegateID})
		if err != nil {
			return Delegation{}, err
		}
		if !can[dl.DelegateID] {
			problems.Add("delegateId", fmt.Sprintf(delegateCannotApproveMessage, delegate.Name))
		}
	}

	start, startOK := parseDay(form.startDate)
	end, endOK := parseDay(form.endDate)
	current := stored != nil && stored.state(today) == DelegationCurrent
	datesOK := false // both dates are right: the overlap check needs them
	switch {
	case !startOK:
		problems.Add("startDate", enterFirstDayMessage)
	case current && start != stored.StartDate:
		problems.Add("startDate", startedMessage)
	case !current && start.Before(today):
		problems.Add("startDate", firstDayTodayMessage)
	default:
		datesOK = true
	}
	startFine := datesOK
	datesOK = false
	switch {
	case !endOK:
		problems.Add("endDate", enterLastDayMessage)
	case startOK && end.Before(start):
		problems.Add("endDate", lastAfterFirstMessage)
	case end.Before(today):
		problems.Add("endDate", lastDayTodayMessage)
	case startOK && end.After(start.AddDays(maxDelegationDays-1)):
		problems.Add("endDate", maxDaysMessage)
	default:
		datesOK = startFine
	}
	if tooLong(dl.Note, maxNoteLength) {
		problems.Add("note", noteTooLongMessage)
	}
	dl.StartDate, dl.EndDate = start, end

	if datesOK && personKnown { // the days of other delegations, once the dates themselves are right
		all, err := s.repo.ListDelegations(ctx)
		if err != nil {
			return Delegation{}, err
		}
		for _, other := range all {
			if other.ID == dl.ID || other.PersonID != dl.PersonID || other.CancelledAt != nil ||
				other.EndDate.Before(start) || end.Before(other.StartDate) {
				continue
			}
			message := fmt.Sprintf(overlapMessage, d.person(other.DelegateID).Name)
			if dl.PersonID != a.ID {
				message = fmt.Sprintf(overlapOtherMessage, person.Name, d.person(other.DelegateID).Name)
			}
			problems.Add("startDate", message)
			break
		}
	}
	if err := problems.Err(); err != nil {
		return Delegation{}, err
	}
	return dl, nil
}
