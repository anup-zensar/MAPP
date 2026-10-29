package purchaserequest

import (
	"context"

	"mapp/internal/approval"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
)

// Engine is the approval engine of feature 031 as purchase requests use it; *approval.Service implements it.
// Its calls join the caller's Write.
type Engine interface {
	SubmitSubject(ctx context.Context, a actor.Actor, kind approval.SubjectKind, id, comment string) (approval.Section, error)
	RecallSubject(ctx context.Context, a actor.Actor, kind approval.SubjectKind, id string) (approval.Section, error)
	DecideSubject(ctx context.Context, a actor.Actor, kind approval.SubjectKind, id string, d approval.SubjectDecision) (approval.Section, error)
	SectionOf(ctx context.Context, a actor.Actor, kind approval.SubjectKind, id string) (approval.Section, error)
}

func (s *Service) UseApprovals(e Engine) {
	s.engine = e
	s.rounds = engineRounds{engine: e}
}

type engineRounds struct {
	engine Engine
}

func (r engineRounds) Submit(ctx context.Context, a actor.Actor, s Submission) error {
	_, err := r.engine.SubmitSubject(ctx, a, approval.SubjectKind(s.Kind), s.ID, s.Comment)
	return err
}

type ApprovalSection struct {
	Kind           RoundKind             `json:"kind"`
	Workflow       *approval.WorkflowRef `json:"workflow"`
	UpcomingLevels []approval.LevelView  `json:"upcomingLevels"`
	Rounds         []approval.RoundView  `json:"rounds"`
	LatestDecline  *approval.DeclineNote `json:"latestDecline"`
	Viewer         approval.Viewer       `json:"viewer"`
	Actions        approval.ActionSet    `json:"actions"`
}

// ApprovalView is a purchase request's Approval section (contract PurchaseRequestApproval): the kind in focus
// first, then the other kind when it has rounds.
type ApprovalView struct {
	PurchaseRequestID      string               `json:"purchaseRequestId"`
	PurchaseRequestName    string               `json:"purchaseRequestName"`
	PurchaseRequestVersion int                  `json:"purchaseRequestVersion"`
	Status                 Status               `json:"status"`
	ChangePending          bool                 `json:"changePending"`
	Owner                  *approval.PersonView `json:"owner"`
	MoAApprover            *approval.PersonView `json:"moaApprover"`
	Sections               []ApprovalSection    `json:"sections"`
	OverPlannedCost        []approval.Overage   `json:"overPlannedCost"`
}

type SubmissionView struct {
	PurchaseRequestID string                     `json:"purchaseRequestId"`
	Version           int                        `json:"version"`
	Kind              string                     `json:"kind"`
	RoundKind         RoundKind                  `json:"roundKind"`
	Available         bool                       `json:"available"`
	Reason            string                     `json:"reason"`
	Ready             bool                       `json:"ready"`
	Problems          []Problem                  `json:"problems"`
	Workflow          *approval.WorkflowRef      `json:"workflow"`
	Levels            []approval.LevelView       `json:"levels"`
	Outcomes          []approval.SubmissionLevel `json:"outcomes"`
}

type PreviewView struct {
	PurchaseRequest Row          `json:"purchaseRequest"`
	Approval        ApprovalView `json:"approval"`
}

// openKind is the kind of the round open now; for a purchase request in no round, the kind a submission starts.
func openKind(pr PurchaseRequest) RoundKind {
	if pr.ChangePending {
		return KindModification
	}
	if pr.Status == StatusSubmitted || pr.Status == StatusInReview {
		return KindPurchaseRequest
	}
	return submissionKind(pr)
}

func (s *Service) approvalOf(ctx context.Context, a actor.Actor, pr PurchaseRequest) (ApprovalView, error) {
	focus := openKind(pr)
	other := KindModification
	if focus == KindModification {
		other = KindPurchaseRequest
	}
	first, err := s.engine.SectionOf(ctx, a, approval.SubjectKind(focus), pr.ID)
	if err != nil {
		return ApprovalView{}, err
	}
	view := ApprovalView{PurchaseRequestID: pr.ID, PurchaseRequestName: pr.Name, PurchaseRequestVersion: pr.Version,
		Status: pr.Status, ChangePending: pr.ChangePending, Owner: first.Owner,
		Sections: []ApprovalSection{sectionOf(focus, first)}, OverPlannedCost: []approval.Overage{}}
	found, err := s.deps.People.People(ctx, []string{pr.MoAApproverID})
	if err != nil {
		return ApprovalView{}, err
	}
	if p, ok := found[pr.MoAApproverID]; ok && pr.MoAApproverID != "" {
		view.MoAApprover = &approval.PersonView{ID: p.ID, Name: p.Name, Initials: p.Initials, Active: p.Active}
	}
	second, err := s.engine.SectionOf(ctx, a, approval.SubjectKind(other), pr.ID)
	if err != nil {
		return ApprovalView{}, err
	}
	if rest := sectionOf(other, second); len(rest.Rounds) > 0 {
		view.Sections = append(view.Sections, withoutSubmit(rest))
	}
	return s.withSubmitRules(ctx, pr, view)
}

// withSubmitRules replaces the engine's submit and re-submit offers of the current section with the purchase
// request's own: its status and the person's purchaseRequests.submit in the plan decide them.
func (s *Service) withSubmitRules(ctx context.Context, pr PurchaseRequest, view ApprovalView) (ApprovalView, error) {
	facts, err := s.factsOf(ctx, pr)
	if err != nil {
		return ApprovalView{}, err
	}
	f := accessFacts{pr: pr, plan: pr.planFacts(facts)}
	view.Sections[0].Actions.Submit = actionOf(f.submit(ctx))
	view.Sections[0].Actions.Resubmit = actionOf(f.resubmit(ctx))
	return view, nil
}

func withoutSubmit(sec ApprovalSection) ApprovalSection {
	sec.Actions.Submit, sec.Actions.Resubmit = approval.ActionAvailability{}, approval.ActionAvailability{}
	return sec
}

func actionOf(a authz.Availability) approval.ActionAvailability {
	return approval.ActionAvailability{Available: a.Available, Reason: a.Reason}
}

func sectionOf(kind RoundKind, sec approval.Section) ApprovalSection {
	rounds := make([]approval.RoundView, 0, len(sec.Rounds))
	for _, r := range sec.Rounds {
		if RoundKind(r.Kind) == kind {
			rounds = append(rounds, r)
		}
	}
	return ApprovalSection{Kind: kind, Workflow: sec.Workflow, UpcomingLevels: sec.UpcomingLevels, Rounds: rounds,
		LatestDecline: sec.LatestDecline, Viewer: sec.Viewer, Actions: sec.Actions}
}
