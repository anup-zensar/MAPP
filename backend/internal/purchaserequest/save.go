package purchaserequest

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/settings"
)

// Save replaces the purchase request's header and lines: Save on Created and Declined, Save and
// re-submit on Submitted and In review, Submit changes on Approved, each in one step.
func (s *Service) Save(ctx context.Context, a actor.Actor, id string, in Input) (View, error) {
	var view View
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		pr, err := s.saved(ctx, a, id, in)
		if err != nil {
			return err
		}
		view, err = s.viewOf(ctx, a, pr)
		return err
	})
	return view, err
}

func (s *Service) saved(ctx context.Context, a actor.Actor, id string, in Input) (PurchaseRequest, error) {
	before, config, err := s.changeable(ctx, id, in.Version)
	if err != nil {
		return PurchaseRequest{}, err
	}
	ck, addsOnly, err := s.checkSave(ctx, in, before, config)
	if err != nil {
		return PurchaseRequest{}, err
	}
	pr, err := s.store(ctx, a, before, ck.pr)
	if err != nil {
		return PurchaseRequest{}, err
	}
	switch {
	case in.Then != ThenSave:
		return s.submitAfterSave(ctx, a, toSubmit{pr: pr, facts: ck.facts, comment: in.Comment})
	case addsOnly:
		return s.approveAddedLines(ctx, a, pr, before.Approved)
	}
	return pr, nil
}

// changeable returns the stored purchase request when the person may change it now from the version they saw.
func (s *Service) changeable(ctx context.Context, id string, version *int) (PurchaseRequest, Configuration, error) {
	pr, err := s.visible(ctx, id)
	if err != nil {
		return PurchaseRequest{}, Configuration{}, err
	}
	facts, err := s.factsOf(ctx, pr)
	if err != nil {
		return PurchaseRequest{}, Configuration{}, err
	}
	if err = authz.RequireIn(ctx, authz.PurchaseRequestsEdit, pr.planFacts(facts)); err != nil {
		return PurchaseRequest{}, Configuration{}, err
	}
	config, err := s.switchedOn(ctx)
	if err != nil {
		return PurchaseRequest{}, Configuration{}, err
	}
	if pr.Status == StatusClosed {
		return PurchaseRequest{}, Configuration{}, errs.NotAllowed(msgClosedNoChange)
	}
	if version == nil {
		return PurchaseRequest{}, Configuration{}, errs.Validation(errs.FieldError{Field: "version", Message: msgReload})
	}
	return pr, config, pr.CheckVersion(*version, recordNoun)
}

// store saves pr over before: new lines get their IDs, lines left out go to the recycle bin, and one "changed"
// entry names every changed field and the lines added, changed and removed.
func (s *Service) store(ctx context.Context, a actor.Actor, before, pr PurchaseRequest) (PurchaseRequest, error) {
	pr.Touch(a, s.now())
	var added, changed []string
	for i, l := range pr.Lines {
		stored := before.line(l.ID)
		switch {
		case stored == nil:
			var err error
			if pr.Lines[i], err = s.newLine(ctx, a, pr.ID, l); err != nil {
				return PurchaseRequest{}, err
			}
			added = append(added, l.LineItemID)
		case !l.sameAs(*stored):
			pr.Lines[i].Touch(a, s.now())
			changed = append(changed, l.LineItemID)
		}
	}
	removed, err := s.removeLines(ctx, a, before, pr)
	if err != nil {
		return PurchaseRequest{}, err
	}
	if err = s.repo.Put(ctx, pr); err != nil {
		return PurchaseRequest{}, err
	}
	changes, err := settings.Diff(headerOf(before), headerOf(pr))
	if err != nil {
		return PurchaseRequest{}, err
	}
	_, err = s.history.Log(ctx, a, history.LogInput{Kind: recordKind, RecordID: pr.ID, RecordName: pr.Name,
		Action: history.ActionChanged, Changes: changes, Details: lineChangesText(added, changed, removed)})
	return pr, err
}

func (s *Service) removeLines(ctx context.Context, a actor.Actor, before, pr PurchaseRequest) ([]string, error) {
	var removed []string
	for _, l := range before.Lines {
		if pr.line(l.ID) != nil {
			continue
		}
		if _, err := s.history.MoveToBin(ctx, a, lineKind, l.ID); err != nil {
			return nil, err
		}
		removed = append(removed, l.LineItemID)
	}
	return removed, nil
}

// toSubmit is a saved purchase request about to be submitted, with the facts of its line items.
type toSubmit struct {
	pr      PurchaseRequest
	facts   map[string]LineItemFacts
	comment string
}

// submitAfterSave starts the round of Save and re-submit or Submit changes once the gates pass.
func (s *Service) submitAfterSave(ctx context.Context, a actor.Actor, t toSubmit) (PurchaseRequest, error) {
	kind := submissionKind(t.pr)
	problems, err := s.gates(ctx, t.pr, t.facts, kind)
	if err != nil {
		return PurchaseRequest{}, err
	}
	if len(problems) > 0 {
		return PurchaseRequest{}, notReady(t.pr, problems, t.facts)
	}
	if s.rounds == nil {
		return PurchaseRequest{}, errs.NotFound(msgNotFound)
	}
	err = s.rounds.Submit(ctx, a, Submission{Kind: kind, ID: t.pr.ID, Comment: strings.TrimSpace(t.comment)})
	if err != nil {
		return PurchaseRequest{}, err
	}
	return s.stored(ctx, t.pr.ID)
}

// approveAddedLines keeps an Approved purchase request Approved when a save only added lines and the setting
// allows it without approval, and routes it as a modification.
func (s *Service) approveAddedLines(ctx context.Context, a actor.Actor, pr PurchaseRequest, previous *Version) (PurchaseRequest, error) {
	pr.Approved = pr.snapshot(s.now())
	if err := s.route(ctx, a, &pr, routing{typ: RoutingModification, previous: previous}); err != nil {
		return PurchaseRequest{}, err
	}
	return pr, s.repo.Put(ctx, pr)
}

func (s *Service) stored(ctx context.Context, id string) (PurchaseRequest, error) {
	pr, found, err := s.repo.Get(ctx, id)
	if err != nil {
		return PurchaseRequest{}, err
	}
	if !found {
		return PurchaseRequest{}, errs.NotFound(msgNotFound)
	}
	return pr, nil
}

func (pr PurchaseRequest) line(id string) *Line {
	if id == "" {
		return nil
	}
	i := slices.IndexFunc(pr.Lines, func(l Line) bool { return l.ID == id })
	if i < 0 {
		return nil
	}
	return &pr.Lines[i]
}

// header is what the log compares of a purchase request; lines are summed up in the entry's details.
type header struct {
	Name                   string                   `json:"name"`
	Description            string                   `json:"description"`
	OwnerID                string                   `json:"ownerId"`
	MoAApproverID          string                   `json:"moaApproverId"`
	ApprovalWorkflowID     string                   `json:"approvalWorkflowId"`
	ModificationWorkflowID string                   `json:"modificationWorkflowId"`
	Attributes             settings.AttributeValues `json:"attributes"`
}

func headerOf(pr PurchaseRequest) header {
	return header{Name: pr.Name, Description: pr.Description, OwnerID: pr.OwnerID, MoAApproverID: pr.MoAApproverID,
		ApprovalWorkflowID: pr.WorkflowID, ModificationWorkflowID: pr.ModificationWorkflowID, Attributes: pr.Attributes}
}

func lineChangesText(added, changed, removed []string) string {
	var parts []string
	for _, p := range []struct {
		word string
		ids  []string
	}{{"Lines added", added}, {"Lines changed", changed}, {"Lines removed", removed}} {
		if len(p.ids) > 0 {
			parts = append(parts, fmt.Sprintf("%s: %s", p.word, strings.Join(p.ids, ", ")))
		}
	}
	return strings.Join(parts, " · ")
}
