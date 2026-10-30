package approval

import (
	"context"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// subject is a record being approved as every step of the engine sees it, whatever its record type (R4).
type subject struct {
	Kind                   SubjectKind
	ID, Name               string
	Status, DisplayStatus  string
	Path                   []errs.RecordRef
	OwnerID, MoAApproverID string
	Amount                 *money.Money
	WorkflowID             string
	ModificationWorkflowID string
	StartDate              *date.Date
	record.Meta
	Access         authz.PlanFacts
	Plan           PlanFacts
	ActivityTypeID string
	TypeName       string
	BusinessUnits  []Choice
	Overages       []Overage
}

func activitySubject(f ActivityFacts) subject {
	return subject{
		Kind: KindActivity, ID: f.ID, Name: f.Name, Status: f.Status, DisplayStatus: f.DisplayStatus, Path: f.Path,
		OwnerID: f.OwnerID, Amount: f.PlannedCost, WorkflowID: f.WorkflowID, StartDate: f.StartDate, Meta: f.Meta,
		Access: f.Access, Plan: f.Plan, ActivityTypeID: f.ActivityTypeID, TypeName: f.TypeName, BusinessUnits: f.BusinessUnits,
		Overages: activityOverages(f),
	}
}

func activityOverages(f ActivityFacts) []Overage {
	if f.OverPlannedCost == nil {
		return nil
	}
	return []Overage{{Activity: errs.RecordRef{Type: activityKind, ID: f.ID, Name: f.Name}, OverBy: *f.OverPlannedCost}}
}

func purchaseSubject(kind SubjectKind, f PurchaseRequestFacts) subject {
	return subject{
		Kind: kind, ID: f.ID, Name: f.Name, Status: f.Status, DisplayStatus: f.Status, Path: f.Path, OwnerID: f.OwnerID,
		MoAApproverID: f.MoAApproverID, Amount: f.Amount, WorkflowID: f.WorkflowID,
		ModificationWorkflowID: f.ModificationWorkflowID, Meta: f.Meta, Access: f.Access, Plan: f.Plan, Overages: f.Overages,
	}
}

// as is the same record seen as another record type: a purchase request's approval or its modification.
func (sub subject) as(kind SubjectKind) subject {
	if sub.Kind.isPurchaseRequest() && kind.isPurchaseRequest() {
		sub.Kind = kind
	}
	return sub
}

func (sub subject) workflowID() string {
	if sub.Kind == KindModifiedPurchaseRequest {
		return sub.ModificationWorkflowID
	}
	return sub.WorkflowID
}

func (sub subject) personIn(field string) string {
	if field == RecordFieldMoAApprover {
		return sub.MoAApproverID
	}
	return ""
}

func (sub subject) ref() errs.RecordRef {
	return errs.RecordRef{Type: sub.Kind.recordType(), ID: sub.ID, Name: sub.Name}
}

func (sub subject) view() SubjectView {
	return SubjectView{Kind: sub.Kind, Label: sub.Kind.Label(), ID: sub.ID, Name: sub.Name, Status: sub.Status,
		DisplayStatus: sub.DisplayStatus, Version: sub.Version}
}

// waiting reports whether the record is in approval: an activity is Submitted exactly while a round is open.
func (sub subject) waiting() bool {
	return sub.Kind != KindActivity || sub.Status == StatusSubmitted
}

func (s *Service) subjectOf(ctx context.Context, kind SubjectKind, id string) (subject, error) {
	if !kind.isPurchaseRequest() {
		facts, err := s.activity(ctx, id)
		return activitySubject(facts), err
	}
	if s.purchases == nil {
		return subject{}, errs.NotFound(purchaseRequestGoneMessage)
	}
	facts, found, err := s.purchases.PurchaseRequest(ctx, id)
	if err != nil {
		return subject{}, err
	}
	if !found {
		return subject{}, errs.NotFound(purchaseRequestGoneMessage)
	}
	return purchaseSubject(kind, facts), nil
}

func (s *Service) readyCheck(ctx context.Context, a actor.Actor, sub subject) (ReadyCheck, error) {
	if sub.Kind == KindActivity {
		return s.activities.ReadyCheck(ctx, a, sub.ID)
	}
	return s.purchases.ReadyCheck(ctx, a, sub.ID, sub.Kind)
}

// moved hands the record a round event and returns it as stored now: the engine sets an activity's status
// itself; a purchase request's status is 032's mapping (data-model §8).
func (s *Service) moved(ctx context.Context, a actor.Actor, sub subject, event Outcome) (subject, error) {
	if sub.Kind.isPurchaseRequest() {
		facts, err := s.purchases.RoundChanged(ctx, a, sub.ID, sub.Kind, event)
		return purchaseSubject(sub.Kind, facts), err
	}
	status := activityStatusAfter(event)
	if status == "" {
		return sub, nil
	}
	facts, err := s.activities.SetStatus(ctx, a, sub.ID, status)
	return activitySubject(facts), err
}

// continued is what a record hears when a level is approved and the round waits at a later one: a purchase
// request hears the round is still open (032 shows it In review); an activity hears nothing.
func continued(kind SubjectKind) Outcome {
	if kind.isPurchaseRequest() {
		return OutcomeOpen
	}
	return ""
}

func activityStatusAfter(event Outcome) string {
	switch event { //nolint:exhaustive // the other events leave an activity's status as it is
	case OutcomeOpen:
		return StatusSubmitted
	case OutcomeApproved:
		return StatusApproved
	case OutcomeDeclined:
		return StatusDeclined
	case OutcomeRecalled:
		return StatusCreated
	}
	return ""
}

// subjectsWithRounds reads the records of rounds that the actor's scope sees, each once, activities first.
func (s *Service) subjectsWithRounds(ctx context.Context, rounds []Round) ([]subject, error) {
	var activityIDs, purchaseIDs []string
	for _, id := range recordsOf(rounds) {
		if id.kind.isPurchaseRequest() {
			purchaseIDs = append(purchaseIDs, id.id)
		} else {
			activityIDs = append(activityIDs, id.id)
		}
	}
	scope := authz.ScopeOf(ctx)
	acts, err := s.activities.ListInScope(ctx, scope, nonNilIDs(activityIDs))
	if err != nil {
		return nil, err
	}
	list := make([]subject, 0, len(acts))
	for _, f := range acts {
		list = append(list, activitySubject(f))
	}
	if s.purchases == nil || len(purchaseIDs) == 0 {
		return list, nil
	}
	prs, err := s.purchases.ListInScope(ctx, scope, purchaseIDs)
	for _, f := range prs {
		list = append(list, purchaseSubject(KindPurchaseRequest, f))
	}
	return list, err
}

type recordID struct {
	kind SubjectKind
	id   string
}

// recordsOf returns the records of rounds, each once, in the order of their ids.
func recordsOf(rounds []Round) []recordID {
	kinds := map[string]SubjectKind{}
	ids := []string{}
	for _, r := range rounds {
		if _, seen := kinds[r.SubjectID]; !seen {
			kinds[r.SubjectID] = r.kind()
			ids = append(ids, r.SubjectID)
		}
	}
	list := make([]recordID, 0, len(ids))
	for _, id := range inIDOrder(ids) {
		list = append(list, recordID{kind: kinds[id], id: id})
	}
	return list
}

// subjectsStored reports which records of the notifications are still stored and visible, read all at once.
func (s *Service) subjectsStored(ctx context.Context, notifications []Notification) (map[string]bool, error) {
	var activityIDs, purchaseIDs []string
	for _, n := range notifications {
		if n.SubjectKind.isPurchaseRequest() {
			purchaseIDs = append(purchaseIDs, n.SubjectID)
		} else {
			activityIDs = append(activityIDs, n.SubjectID)
		}
	}
	stored := map[string]bool{}
	acts, err := s.activities.ListOf(ctx, nonNilIDs(activityIDs))
	for _, f := range acts {
		stored[f.ID] = true
	}
	if err != nil || s.purchases == nil || len(purchaseIDs) == 0 {
		return stored, err
	}
	prs, err := s.purchases.ListInScope(ctx, authz.ScopeOf(ctx), purchaseIDs)
	for _, f := range prs {
		stored[f.ID] = true
	}
	return stored, err
}
