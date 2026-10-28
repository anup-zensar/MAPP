package approval

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/errs"
)

func TestCreateWorkflowObjectType(t *testing.T) {
	t.Parallel()
	moa := LevelInput{ApproverType: SourceRecordField, RecordField: RecordFieldMoAApprover}
	tests := []struct {
		name  string
		kind  SubjectKind
		level LevelInput
		want  SubjectKind
	}{
		{"031 FR-004 without an object type it approves activities", "", LevelInput{ApproverIDs: []string{daniel.ID}}, KindActivity},
		{"031 FR-004 a purchase-request workflow", KindPurchaseRequest, moa, KindPurchaseRequest},
		{"031 FR-004 a modification workflow", KindModifiedPurchaseRequest, LevelInput{ApproverType: SourceOwner}, KindModifiedPurchaseRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			view, err := f.service.CreateWorkflow(t.Context(), alex, WorkflowInput{Name: "PR approval", ObjectType: tt.kind,
				Levels: []LevelInput{tt.level}})
			if err != nil || view.ObjectType != tt.want || view.ObjectTypeFixed {
				t.Fatalf("CreateWorkflow = %+v, %v; want an unused %s workflow", view, err, tt.want)
			}
		})
	}
	t.Run("031 FR-004 an unknown object type is a problem", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		_, err := f.service.CreateWorkflow(t.Context(), alex, WorkflowInput{Name: "Odd", ObjectType: "invoice",
			Levels: levelInputs([]string{daniel.ID})})
		assertError(t, err, errs.Validation(errs.FieldError{Field: objectTypeField, Message: chooseObjectTypeMessage}))
	})
}

func TestUpdateWorkflowObjectType(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-005 an unused workflow changes what it approves, logged as Approves", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		created := f.createWorkflow(t, alex, "Spare", []string{daniel.ID})
		kind := KindPurchaseRequest

		view, err := f.service.UpdateWorkflow(t.Context(), alex, created.ID, WorkflowUpdate{ObjectType: &kind, Version: version(1)})

		if err != nil || view.ObjectType != KindPurchaseRequest {
			t.Fatalf("UpdateWorkflow = %+v, %v; want a purchase-request workflow", view, err)
		}
		assertLog(t, f, created.ID, []string{
			"Alex Morgan changed approval workflow 'Spare' [activity → purchaseRequest]",
			"Alex Morgan created approval workflow 'Spare'",
		})
		change := f.log(t, created.ID)[0].Changes[0]
		if change.Label != "Approves" || change.OldText != "Activity" || change.NewText != "Purchase request" {
			t.Errorf("change = %+v, want Approves: Activity → Purchase request", change)
		}
	})
	t.Run("031 FR-005 a used workflow keeps what it approves", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		f.addActivity(t, "A-1001", "Launch film", priya.ID, lubricantsFlow)
		kind := KindPurchaseRequest

		_, err := f.service.UpdateWorkflow(t.Context(), alex, lubricantsFlow, WorkflowUpdate{ObjectType: &kind, Version: version(1)})

		assertError(t, err, errs.Validation(errs.FieldError{Field: objectTypeField,
			Message: "It approves activities and is in use, so this can't change. Create a new workflow instead."}))
		if views := f.workflowViews(t, alex); !views[1].ObjectTypeFixed || views[1].ID != lubricantsFlow {
			t.Errorf("views[1] = %+v, want Lubricants EU approval fixed", views[1])
		}
	})
}

func TestRecordFieldOnActivityWorkflow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		kind  SubjectKind
		level LevelInput
		want  string
	}{
		{"031 FR-011 activities have no field that names an approver", KindActivity,
			LevelInput{ApproverType: SourceRecordField, RecordField: RecordFieldMoAApprover},
			"Level 1: activities have no field that names an approver. Choose another option."},
		{"031 FR-011 a purchase-request level names a field", KindPurchaseRequest,
			LevelInput{ApproverType: SourceRecordField, RecordField: "vendor"}, "Level 1: choose the field that names the approver."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			_, err := f.service.CreateWorkflow(t.Context(), alex, WorkflowInput{Name: "Field", ObjectType: tt.kind, Levels: []LevelInput{tt.level}})
			assertError(t, err, errs.Validation(errs.FieldError{Field: "levels.1.recordField", Message: tt.want}))
		})
	}
}

func TestConditionsPerObjectType(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-014 purchase-request workflows use amount conditions only, worded as the amount", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		below := "10000.00"
		_, err := f.service.CreateWorkflow(t.Context(), alex, WorkflowInput{Name: "PR", ObjectType: KindPurchaseRequest,
			Levels: []LevelInput{{ApproverType: SourceOwner, AppliesWhen: &ConditionInput{ActivityTypeIDs: []string{eventType}}}}})
		assertError(t, err, errs.Validation(errs.FieldError{Field: "levels.1.appliesWhen",
			Message: "Level 1: only activity workflows can use business unit and activity type conditions."}))

		view, err := f.service.CreateWorkflow(t.Context(), alex, WorkflowInput{Name: "PR", ObjectType: KindPurchaseRequest,
			Levels: []LevelInput{{ApproverType: SourceOwner, AutoApproveWhen: &ConditionInput{PlannedCostBelow: &below}}}})
		if err != nil || view.LevelsText != "Level 1 · Record owner · automatic when amount is below $10,000.00" {
			t.Errorf("view = %q, %v; want the amount wording", view.LevelsText, err)
		}
	})
}

func TestWorkflowUsageOfPurchaseRequests(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-009 a purchase-request workflow lists the purchase requests that used it", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		prs := f.usePurchases(t)
		f.putKindWorkflow(t, prFlow, KindPurchaseRequest, named(daniel))
		f.addPurchaseRequest(t, prs, purchaseRequest{id: purchaseRequestA, owner: priya.ID, workflow: prFlow})
		f.submitPR(t, KindPurchaseRequest)

		details, err := f.service.GetWorkflow(t.Context(), alex, prFlow)

		if err != nil || details.Workflow.UsedByCount != 1 || details.Workflow.WaitingCount != 1 || !details.Workflow.ObjectTypeFixed ||
			len(details.Activities) != 1 || details.Activities[0].Activity.Type != string(KindPurchaseRequest) ||
			details.Activities[0].DisplayStatus != StatusSubmitted {
			t.Errorf("details = %+v, %v; want PR-1001 submitted", details, err)
		}
		_, err = f.service.DeleteWorkflow(t.Context(), alex, prFlow)
		assertError(t, err, errs.InUse("Used by 1 purchase request — set it inactive instead."))
	})
}

func TestSinglePersonLevels(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-013 owner and record-field levels have one approver and name their source", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		view, err := f.service.CreateWorkflow(t.Context(), alex, WorkflowInput{Name: "PR", ObjectType: KindPurchaseRequest,
			Levels: []LevelInput{
				{ApproverType: SourceRecordField, RecordField: RecordFieldMoAApprover, Mode: ModeAll, ApproverIDs: []string{daniel.ID}},
				{ApproverType: SourceOwner, Mode: ModeAll},
			}})
		if err != nil {
			t.Fatalf("CreateWorkflow: %v", err)
		}
		moa, owner := view.Levels[0], view.Levels[1]
		got := []any{moa.Mode, len(moa.Approvers), moa.RecordField, owner.Mode, view.LevelsText}
		want := []any{ModeAny, 0, &RecordFieldRef{ID: RecordFieldMoAApprover, Label: "MoA approver"}, ModeAny,
			"Level 1 · MoA approver → Level 2 · Record owner"}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("levels (-want +got):\n%s", diff)
		}
	})
}
