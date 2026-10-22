package approvals

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

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

// The scopes of the delegation list (GET /api/approval-delegations?scope=).
const (
	ScopeMine = "mine"
	ScopeAll  = "all"
)

// ListDelegations returns the delegations a sees (017 FR-024): with scope mine (the default) those where a
// is the delegating person or the delegate; with scope all everyone's, which needs workflows.manage (403
// otherwise). Current and planned ones come first by first day, then ended and cancelled ones, latest last
// day first. Each says whether a may change it; the page says whether a may manage other people's.
func (s *Service) ListDelegations(ctx context.Context, a actor.Actor, scope string) (DelegationPage, error) {
	if scope == "" {
		scope = ScopeMine
	}
	if scope != ScopeMine && scope != ScopeAll {
		return DelegationPage{}, errs.Validation(errs.FieldError{Field: "scope", Message: chooseScopeMessage})
	}
	var page DelegationPage
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		manage, err := s.access.Check(ctx, a.ID, PermissionManageWorkflows, nil)
		if err != nil {
			return err
		}
		if scope == ScopeAll && !manage.Allowed {
			return errs.Forbidden(manage.Refusal)
		}
		all, err := s.repo.ListDelegations(ctx)
		if err != nil {
			return err
		}
		d, err := s.directory(ctx)
		if err != nil {
			return err
		}
		today := date.Today(s.now())
		list := where(all, func(dl Delegation) bool {
			return scope == ScopeAll || dl.PersonID == a.ID || dl.DelegateID == a.ID
		})
		slices.SortStableFunc(list, func(x, y Delegation) int { return compareDelegations(x, y, today) })
		page = DelegationPage{Items: make([]DelegationView, 0, len(list)), Total: len(list), CanManageOthers: manage.Allowed}
		for _, dl := range list {
			page.Items = append(page.Items, delegationView(dl, d, a, manage.Allowed, today))
		}
		return nil
	})
	if err != nil {
		return DelegationPage{}, err
	}
	return page, nil
}

// compareDelegations orders delegations as the panel lists them: current and planned first, by first day;
// then ended and cancelled, latest last day first; then by id.
func compareDelegations(x, y Delegation, today date.Date) int {
	group := func(dl Delegation) int {
		switch dl.state(today) {
		case DelegationCurrent, DelegationPlanned:
			return 0
		}
		return 1
	}
	if c := cmp.Compare(group(x), group(y)); c != 0 {
		return c
	}
	if group(x) == 0 {
		return cmp.Or(x.StartDate.Compare(y.StartDate), cmp.Compare(x.ID, y.ID))
	}
	return cmp.Or(y.EndDate.Compare(x.EndDate), cmp.Compare(x.ID, y.ID))
}

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

// CancelDelegation cancels the delegation id at once (017 FR-024): the delegate loses the rights it gave;
// decisions already made stay recorded. The rules are UpdateDelegation's. It is logged as cancelled.
func (s *Service) CancelDelegation(ctx context.Context, a actor.Actor, id string, cancel DelegationCancel) (DelegationView, error) {
	var view DelegationView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		stored, manage, err := s.changeableDelegation(ctx, a, id, cancel.Version)
		if err != nil {
			return err
		}
		d, err := s.directory(ctx)
		if err != nil {
			return err
		}
		now := s.now()
		changed := stored
		changed.CancelledAt, changed.CancelledBy = &now, &a
		changes, err := history.Diff(stored.logView(), changed.logView())
		if err != nil {
			return err
		}
		changed.Touch(a, now)
		if err := s.repo.PutDelegation(ctx, changed); err != nil {
			return err
		}
		_, err = s.history.Log(ctx, a, history.LogInput{
			Kind: delegationKind, RecordID: changed.ID, RecordName: delegationName(changed, d), Action: history.ActionStatusChanged,
			Verb: "cancelled", Changes: changes,
		})
		if err != nil {
			return err
		}
		view = delegationView(changed, d, a, manage, date.Today(now))
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
		return Delegation{}, false, errs.Validation(errs.FieldError{Field: "version", Message: reloadMessage})
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

// parseDay reads a date "YYYY-MM-DD"; false when it is empty or not a date.
func parseDay(text string) (date.Date, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return date.Date{}, false
	}
	d, err := date.Parse(text)
	return d, err == nil
}

// delegationName is a delegation's name in the log: "<person> → <delegate>" (data-model §7).
func delegationName(dl Delegation, d directory) string {
	return d.person(dl.PersonID).Name + " → " + d.person(dl.DelegateID).Name
}

// delegationLog is a delegation as the activity log compares it: what people read, without the bookkeeping.
type delegationLog struct {
	DelegateID  string     `json:"delegateId"`
	StartDate   date.Date  `json:"startDate"`
	EndDate     date.Date  `json:"endDate"`
	Note        string     `json:"note"`
	CancelledAt *time.Time `json:"cancelledAt"`
}

// logView returns dl as the activity log compares it.
func (dl Delegation) logView() delegationLog {
	return delegationLog{DelegateID: dl.DelegateID, StartDate: dl.StartDate, EndDate: dl.EndDate, Note: dl.Note, CancelledAt: dl.CancelledAt}
}

// delegationKindOf is the history kind of approval delegations: they are logged (017 FR-035) but never moved
// to the recycle bin, so Remove is never called.
func (s *Service) delegationKindOf() history.Kind {
	return history.Kind{
		Name: delegationKind, Singular: delegationSingular, Plural: "approval delegations",
		Snapshot: func(ctx context.Context, id string) (history.Snapshot, bool, error) {
			dl, found, err := s.repo.GetDelegation(ctx, id)
			if err != nil || !found {
				return history.Snapshot{}, false, err
			}
			d, err := s.directory(ctx)
			if err != nil {
				return history.Snapshot{}, false, err
			}
			data, err := json.Marshal(dl)
			if err != nil {
				return history.Snapshot{}, false, err
			}
			return history.Snapshot{ID: dl.ID, Name: delegationName(dl, d), Data: data}, true, nil
		},
		Remove: func(context.Context, string) error { return errs.NotAllowed(delegationEndedMessage) },
		Put: func(ctx context.Context, snapshot history.Snapshot) error {
			var dl Delegation
			if err := json.Unmarshal(snapshot.Data, &dl); err != nil {
				return err
			}
			return s.repo.PutDelegation(ctx, dl)
		},
		Describe: s.describeDelegation,
	}
}

// describeDelegation gives the labels and texts of a delegation's changed fields in the activity log (data
// model §7): the delegate's name, the dates as "01 Nov 2026", the note and the status.
func (s *Service) describeDelegation(ctx context.Context, field string, value any) (string, string, error) {
	switch field {
	case "delegateId":
		id, _ := value.(string)
		d, err := s.directory(ctx)
		if err != nil {
			return "", "", err
		}
		return "Delegate", d.person(id).Name, nil
	case "startDate", "endDate":
		label := "From"
		if field == "endDate" {
			label = "To"
		}
		text, _ := value.(string)
		if day, ok := parseDay(text); ok {
			text = date.Format(day)
		}
		return label, text, nil
	case "note":
		return "Note", "", nil
	case "cancelledAt":
		if value == nil {
			return "Status", "", nil
		}
		return "Status", cancelledStatusText, nil
	}
	return "", "", nil
}
