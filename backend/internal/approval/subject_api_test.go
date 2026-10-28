package approval

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

const (
	prFlow           = "WF-1101"
	prChangesFlow    = "WF-1102"
	prTwoLevelFlow   = "WF-1103"
	purchaseRequestA = "PR-1001"
)

// newPurchaseFixture is a fixture with feature 032 connected and PR-1001 of Priya Raman on workflow.
func newPurchaseFixture(t *testing.T, pr purchaseRequest, workflows ...Workflow) (*fixture, *fakePurchases) {
	t.Helper()
	f := newFixture(t)
	prs := f.usePurchases(t)
	for _, w := range workflows {
		f.putKindWorkflow(t, w.ID, w.ObjectType, w.Levels...)
	}
	pr.id = purchaseRequestA
	if pr.owner == "" {
		pr.owner = priya.ID
	}
	f.addPurchaseRequest(t, prs, pr)
	return f, prs
}

func kindFlow(id string, kind SubjectKind, levels ...Level) Workflow {
	return Workflow{ID: id, ObjectType: kind, Levels: levels}
}

func TestPurchaseRequestsNotWired(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-025 every purchase-request operation answers not found", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		_, sectionErr := f.service.SectionOf(ctx, priya, KindPurchaseRequest, purchaseRequestA)
		_, submitErr := f.service.SubmitSubject(ctx, priya, KindPurchaseRequest, purchaseRequestA, "")
		_, recallErr := f.service.RecallSubject(ctx, priya, KindModifiedPurchaseRequest, purchaseRequestA)
		for _, err := range []error{sectionErr, submitErr, recallErr} {
			assertError(t, err, errs.NotFound(purchaseRequestGoneMessage))
		}
	})
}

func TestSubmitSubjectGates(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-021 the PR's gates and the workflow's problems come together, nothing starts", func(t *testing.T) {
		t.Parallel()
		f, prs := newPurchaseFixture(t, purchaseRequest{workflow: "WF-1999"})
		gate := SubmitProblem{Field: "lines", Group: GroupPurchaseRequest, Message: "Activity 'Launch film' must be approved first."}
		prs.problems[purchaseRequestA] = []SubmitProblem{gate}

		_, err := f.service.SubmitSubject(t.Context(), priya, KindPurchaseRequest, purchaseRequestA, "")

		assertError(t, err, errs.Validation(errs.FieldError{Field: "lines", Message: gate.Message},
			errs.FieldError{Field: workflowField, Message: workflowNeededMessage}))
		if len(f.rounds(t, purchaseRequestA)) != 0 || len(prs.events[purchaseRequestA]) != 0 {
			t.Errorf("rounds or events were written: %v", prs.events)
		}
	})
	t.Run("031 FR-008 a workflow of another object type is a problem", func(t *testing.T) {
		t.Parallel()
		f, _ := newPurchaseFixture(t, purchaseRequest{workflow: lubricantsFlow})

		_, err := f.service.SubmitSubject(t.Context(), priya, KindPurchaseRequest, purchaseRequestA, "")

		want := "Approval workflow 'Lubricants EU approval' approves activities — choose one that approves purchase requests."
		assertError(t, err, errs.Validation(errs.FieldError{Field: workflowField, Message: want}))
	})
}

func TestRoundChangedLevelByLevel(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-023 FR-030 start, a level approved with more to come, approved, each logged", func(t *testing.T) {
		t.Parallel()
		f, prs := newPurchaseFixture(t, purchaseRequest{workflow: prTwoLevelFlow},
			kindFlow(prTwoLevelFlow, KindPurchaseRequest, named(daniel), named(carlos)))
		var statuses []string
		submitted := f.submitPR(t, KindPurchaseRequest)
		afterLevel1 := f.mustDecide(t, daniel, ActionApproved, "")
		approved := f.mustDecide(t, carlos, ActionApproved, "")
		statuses = append(statuses, submitted.Subject.Status, afterLevel1.Subject.Status, approved.Subject.Status)

		want := []purchaseEvent{{KindPurchaseRequest, OutcomeOpen}, {KindPurchaseRequest, OutcomeOpen}, {KindPurchaseRequest, OutcomeApproved}}
		if diff := cmp.Diff(want, prs.events[purchaseRequestA], cmp.AllowUnexported(purchaseEvent{})); diff != "" {
			t.Errorf("events (-want +got):\n%s", diff)
		}
		if !slices.Equal(statuses, []string{StatusSubmitted, StatusInReview, StatusApproved}) {
			t.Errorf("statuses = %v", statuses)
		}
		assertLogVerbs(t, f, "submitted", "approved level 1 of", "approved")
	})
}

func TestRoundChangedEnds(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-023 declined and recalled end the round", func(t *testing.T) {
		t.Parallel()
		f, prs := newPurchaseFixture(t, purchaseRequest{workflow: prFlow}, kindFlow(prFlow, KindPurchaseRequest, named(daniel)))
		f.submitPR(t, KindPurchaseRequest)
		f.mustDecide(t, daniel, ActionDeclined, "Wrong vendor")
		f.submitPR(t, KindPurchaseRequest)

		recalled, err := f.service.RecallSubject(t.Context(), priya, KindPurchaseRequest, purchaseRequestA)

		if err != nil || recalled.Subject.Status != StatusCreated || recalled.Rounds[0].Outcome != OutcomeRecalled {
			t.Fatalf("RecallSubject = %+v, %v; want Created with the round recalled", recalled.Subject, err)
		}
		want := []purchaseEvent{{KindPurchaseRequest, OutcomeOpen}, {KindPurchaseRequest, OutcomeDeclined},
			{KindPurchaseRequest, OutcomeOpen}, {KindPurchaseRequest, OutcomeRecalled}}
		if diff := cmp.Diff(want, prs.events[purchaseRequestA], cmp.AllowUnexported(purchaseEvent{})); diff != "" {
			t.Errorf("events (-want +got):\n%s", diff)
		}
	})
	t.Run("031 FR-017 re-submitting a submitted PR supersedes its round and opens a new one", func(t *testing.T) {
		t.Parallel()
		f, prs := newPurchaseFixture(t, purchaseRequest{workflow: prFlow}, kindFlow(prFlow, KindPurchaseRequest, named(daniel)))
		f.submitPR(t, KindPurchaseRequest)

		sec := f.submitPR(t, KindPurchaseRequest)

		got := []Outcome{sec.Rounds[0].Outcome, sec.Rounds[1].Outcome}
		if !slices.Equal(got, []Outcome{OutcomeOpen, OutcomeSuperseded}) || sec.Subject.Status != StatusSubmitted {
			t.Fatalf("outcomes %v, status %s; want round 1 superseded and round 2 open", got, sec.Subject.Status)
		}
		want := []purchaseEvent{{KindPurchaseRequest, OutcomeOpen}, {KindPurchaseRequest, OutcomeSuperseded}, {KindPurchaseRequest, OutcomeOpen}}
		if diff := cmp.Diff(want, prs.events[purchaseRequestA], cmp.AllowUnexported(purchaseEvent{})); diff != "" {
			t.Errorf("events (-want +got):\n%s", diff)
		}
	})
}

func TestModificationRounds(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-002 FR-022 a modification uses its own workflow and numbers after the approval", func(t *testing.T) {
		t.Parallel()
		f, _ := newPurchaseFixture(t, purchaseRequest{workflow: prFlow, modification: prChangesFlow},
			kindFlow(prFlow, KindPurchaseRequest, named(daniel)), kindFlow(prChangesFlow, KindModifiedPurchaseRequest, Level{ApproverType: SourceOwner}))
		f.submitPR(t, KindPurchaseRequest)
		approved := f.mustDecide(t, daniel, ActionApproved, "")

		sec := f.submitPR(t, KindModifiedPurchaseRequest)

		round := sec.Rounds[0]
		got := []any{approved.Actions.Submit.Available, round.Number, round.Kind, round.Workflow.ID, sec.Subject.Status, round.Levels[0].Source}
		want := []any{false, 2, KindModifiedPurchaseRequest, prChangesFlow, StatusInReview, recordOwnerSource}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("modification round (-want +got):\n%s", diff)
		}
		if declined := f.mustDecide(t, priya, ActionDeclined, "No"); declined.Subject.Status != StatusApproved {
			t.Errorf("after the modification is declined: %s, want Approved", declined.Subject.Status)
		}
	})
}

func TestPurchaseRequestSelfApproval(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-015 FR-016 the owner approves their own PR while the access rule is on, recorded as self-approval", func(t *testing.T) {
		t.Parallel()
		moa := Level{ApproverType: SourceRecordField, RecordField: RecordFieldMoAApprover}
		f, _ := newPurchaseFixture(t, purchaseRequest{moa: priya.ID, workflow: prFlow}, kindFlow(prFlow, KindPurchaseRequest, moa))
		f.submitPR(t, KindPurchaseRequest)

		sec := f.mustDecide(t, priya, ActionApproved, "")

		decision := sec.Rounds[0].Levels[0].Decision
		if sec.Subject.Status != StatusApproved || decision == nil || !decision.SelfApproval || decision.Capacity != CapacityApprover {
			t.Errorf("status %s, decision %+v; want Approved by a self-approval", sec.Subject.Status, decision)
		}
		if entries := f.log(t, purchaseRequestA); !strings.Contains(entries[0].Details, "self-approval") {
			t.Errorf("log details = %q, want self-approval", entries[0].Details)
		}
	})
}

func TestActivitySelfApprovalStillBarred(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-015 an activity's owner is still left out and refused", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		f.putWorkflow(t, "WF-1003", "Owner approves", nil, nil, Level{ApproverType: SourceOwner})
		f.addActivity(t, "A-1001", "Launch film", priya.ID, "WF-1003")
		approval := f.submit(t, priya, "A-1001", "")

		level := approval.Rounds[0].Levels[0]
		if len(level.Approvers) != 0 || !level.NoApprover || excludedOf(level) != ReasonOwner {
			t.Fatalf("level = %+v, want the owner left out and no approver", level)
		}
		f.access.onBehalf = []string{alex.ID}
		_, err := f.service.Approve(t.Context(), priya, "A-1001", DecisionInput{RoundID: approval.Rounds[0].ID, Level: version(1)})
		assertError(t, err, errs.Forbidden("Only the approvers of level 1 (nobody) can approve or decline this activity."))
	})
}

func TestSectionActions(t *testing.T) {
	t.Parallel()
	open := &Round{SubjectKind: KindPurchaseRequest, Outcome: OutcomeOpen}
	openChanges := &Round{SubjectKind: KindModifiedPurchaseRequest, Outcome: OutcomeOpen}
	tests := []struct {
		name   string
		kind   SubjectKind
		status string
		open   *Round
		want   string
	}{
		{"031 FR-024 created PR submits", KindPurchaseRequest, StatusCreated, nil, kindSubmit},
		{"031 FR-024 declined PR re-submits", KindPurchaseRequest, StatusDeclined, nil, kindResubmit},
		{"031 FR-024 PR in review re-submits", KindPurchaseRequest, StatusInReview, open, kindResubmit},
		{"031 FR-024 approved PR has no first submission", KindPurchaseRequest, StatusApproved, nil, ""},
		{"031 FR-024 approved PR submits changes", KindModifiedPurchaseRequest, StatusApproved, nil, kindSubmit},
		{"031 FR-024 changes in review re-submit", KindModifiedPurchaseRequest, StatusInReview, openChanges, kindResubmit},
		{"031 FR-024 no changes while the first round is open", KindModifiedPurchaseRequest, StatusSubmitted, open, ""},
		{"031 FR-024 closed PR submits nothing", KindModifiedPurchaseRequest, StatusClosed, nil, ""},
		{"031 FR-017 submitted activity re-submits", KindActivity, StatusSubmitted, nil, kindResubmit},
		{"031 FR-017 approved activity submits nothing", KindActivity, StatusApproved, nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := submissionKind(subject{Kind: tt.kind, Status: tt.status}, tt.open); got != tt.want {
				t.Errorf("submissionKind = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRecallSubjectRights(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-024 nothing to recall without an open round", func(t *testing.T) {
		t.Parallel()
		f, _ := newPurchaseFixture(t, purchaseRequest{})

		_, err := f.service.RecallSubject(t.Context(), priya, KindPurchaseRequest, purchaseRequestA)

		assertError(t, err, errs.BadRequest("This purchase request is not waiting for approval, so there is nothing to recall."))
	})
	t.Run("031 FR-029 refusals name the purchase request", func(t *testing.T) {
		t.Parallel()
		f, _ := newPurchaseFixture(t, purchaseRequest{workflow: prFlow}, kindFlow(prFlow, KindPurchaseRequest, named(daniel)))
		f.submitPR(t, KindPurchaseRequest)
		f.access.onBehalf = []string{alex.ID}

		_, refused := f.decideSubject(t, mei, ActionApproved, "")
		_, noComment := f.decideSubject(t, daniel, ActionDeclined, "")

		assertError(t, refused, errs.Forbidden("Only the approvers of level 1 (Daniel Okoro) can approve or decline this purchase request."))
		assertError(t, noComment, errs.Validation(errs.FieldError{Field: "comment",
			Message: "Add a comment explaining why the purchase request is declined."}))
	})
}

func (f *fixture) submitPR(t *testing.T, kind SubjectKind) Section {
	t.Helper()
	sec, err := f.service.SubmitSubject(t.Context(), priya, kind, purchaseRequestA, "")
	if err != nil {
		t.Fatalf("SubmitSubject(%s): %v", kind, err)
	}
	return sec
}

func (f *fixture) mustDecide(t *testing.T, a actor.Actor, action ActionKind, comment string) Section {
	t.Helper()
	sec, err := f.decideSubject(t, a, action, comment)
	if err != nil {
		t.Fatalf("DecideSubject(%s, %s): %v", a.Name, action, err)
	}
	return sec
}

func assertLogVerbs(t *testing.T, f *fixture, verbs ...string) {
	t.Helper()
	entries := f.log(t, purchaseRequestA)
	got := make([]string, 0, len(entries))
	for i := len(entries) - 1; i >= 0; i-- {
		got = append(got, entries[i].Verb)
	}
	if !slices.Equal(got, verbs) {
		t.Errorf("log verbs = %q, want %q", got, verbs)
	}
}

func excludedOf(level ChainLevel) string {
	if len(level.Excluded) == 0 {
		return ""
	}
	return level.Excluded[0].Reason
}
