package authz_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
)

// A context without Grants is an internal call — the seed, a job, a service unit test — and is not
// restricted (D-228).
func TestNoGrants_Unrestricted_FR010(t *testing.T) {
	ctx := context.Background()

	if err := authz.Require(ctx, authz.SettingsManage); err != nil {
		t.Errorf("Require = %v", err)
	}
	if err := authz.RequireIn(ctx, authz.PlansDelete, plan(mobility)); err != nil {
		t.Errorf("RequireIn = %v", err)
	}
	if err := authz.RequireView(ctx, plan(mobility), "plan"); err != nil {
		t.Errorf("RequireView = %v", err)
	}
	if a := authz.Available(ctx, authz.PlansEdit, plan(mobility)); !a.Available || a.Reason != "" {
		t.Errorf("Available = %+v", a)
	}
	if reason := authz.SelfApproval(ctx, "U-1003", "U-1003"); reason != "" {
		t.Errorf("SelfApproval = %q", reason)
	}
}

// A refused action is 403 forbidden with the plain message (FR-010).
func TestRequireIn_Refuses403_FR010(t *testing.T) {
	ctx := authz.With(context.Background(), grants("U-1006", []string{lubricants, mobility}, viewer))

	err := authz.RequireIn(ctx, authz.ActivitiesEdit, plan(lubricants))

	var apiErr *errs.Error
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusForbidden || apiErr.Code != errs.CodeForbidden {
		t.Fatalf("err = %v, want 403 forbidden", err)
	}
	want := "Changing activities and line items isn't part of your role (Viewer). Ask an administrator if you need it."
	if apiErr.Message != want {
		t.Errorf("message = %q\nwant %q", apiErr.Message, want)
	}
}

// Opening a record outside the person's access answers message 4, which names nothing of the record (FR-008).
func TestRequireView_NoAccessMessage_FR008(t *testing.T) {
	ctx := authz.With(context.Background(), grants("U-1004", []string{mobility}, planner))

	err := authz.RequireView(ctx, plan(lubricants), "plan")
	errActivity := authz.RequireViewActivity(ctx, authz.ActivityFacts{Plan: plan(lubricants)}, "activity")

	if err == nil || err.Error() != "forbidden: You don't have access to this plan. Ask an administrator if you need it." {
		t.Errorf("RequireView = %v", err)
	}
	if errActivity == nil || errActivity.Error() != "forbidden: You don't have access to this activity. Ask an administrator if you need it." {
		t.Errorf("RequireViewActivity = %v", errActivity)
	}
	if err := authz.RequireView(ctx, plan(mobility), "plan"); err != nil {
		t.Errorf("own market: %v", err)
	}
}

// Availability carries the same message as the refusal, so screens explain a disabled action exactly as the
// API would refuse it (FR-011).
func TestAvailable_ReasonIsTheRefusal_FR011(t *testing.T) {
	ctx := authz.With(context.Background(), grants("U-1002", []string{lubricants}, planner))

	deletePlan := authz.Available(ctx, authz.PlansDelete, plan(lubricants))
	editPlan := authz.Available(ctx, authz.PlansEdit, plan(lubricants))
	settings := authz.AvailableAnywhere(ctx, authz.SettingsManage)

	if deletePlan.Available || deletePlan.Reason != "Deleting plans isn't part of your role (Planner). Ask an administrator if you need it." {
		t.Errorf("delete = %+v", deletePlan)
	}
	if !editPlan.Available || editPlan.Reason != "" {
		t.Errorf("edit = %+v", editPlan)
	}
	if settings.Available || settings.Reason != "Changing Settings isn't part of your role (Planner). Ask an administrator if you need it." {
		t.Errorf("settings = %+v", settings)
	}
}

// Has, InScope and CanView read the Grants of the context.
func TestContextChecks_FR006(t *testing.T) {
	ctx := authz.With(context.Background(), grants("U-1004", []string{mobility}, planner))

	if authz.Has(ctx, authz.PlansDelete) || !authz.Has(ctx, authz.PlansEdit) {
		t.Error("Has does not follow the Planner role")
	}
	if authz.InScope(ctx, authz.PlansEdit, plan(lubricants)) || !authz.InScope(ctx, authz.PlansEdit, plan(mobility)) {
		t.Error("InScope does not follow Mei's team")
	}
	if authz.CanView(ctx, plan(lubricants)) || !authz.CanView(ctx, plan(mobility)) {
		t.Error("CanView does not follow Mei's team")
	}
	g, ok := authz.From(ctx)
	if !ok || g.PersonID != "U-1004" {
		t.Errorf("From = %+v, %v", g, ok)
	}
}
