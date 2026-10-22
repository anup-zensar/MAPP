package authz

import (
	"context"

	"mapp/internal/platform/errs"
)

// contextKey is private, so no other package can put Grants in a context by accident.
type contextKey struct{}

// With returns a copy of ctx that carries g. Only the access middleware calls it (and tests).
func With(ctx context.Context, g Grants) context.Context {
	return context.WithValue(ctx, contextKey{}, g)
}

// From returns the Grants ctx carries, and false when there are none: an internal call (the seed, a
// background job, a service unit test), which is not restricted (D-228).
func From(ctx context.Context) (Grants, bool) {
	g, ok := ctx.Value(contextKey{}).(Grants)
	return g, ok
}

// Unrestricted returns a copy of ctx without Grants, for a read that must see every record whoever asks —
// such as the "Used by" counts of Settings, which count records the person may not see. Use it only where
// the answer does not show those records.
func Unrestricted(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKey{}, nil)
}

// Has reports whether the person acting holds p in any scope; true for an internal call.
func Has(ctx context.Context, p Permission) bool {
	g, ok := From(ctx)
	return !ok || g.Has(p)
}

// InScope reports whether the person acting holds p for the plan; true for an internal call.
func InScope(ctx context.Context, p Permission, plan PlanFacts) bool {
	g, ok := From(ctx)
	return !ok || g.InScope(p, plan)
}

// CanView reports whether the person acting sees the plan and everything in it (FR-006); true for an
// internal call. Lists keep only the plans it allows (FR-007).
func CanView(ctx context.Context, plan PlanFacts) bool {
	g, ok := From(ctx)
	return !ok || g.CanViewPlan(plan)
}

// CanViewActivity reports whether the person acting sees the activity (FR-006); true for an internal call.
func CanViewActivity(ctx context.Context, a ActivityFacts) bool {
	g, ok := From(ctx)
	return !ok || g.CanViewActivity(a)
}

// Require refuses, with 403 forbidden, an action that needs p when no role of the person acting includes
// it (plan messages 1 and 2). For an action about one plan, use RequireIn.
func Require(ctx context.Context, p Permission) error {
	g, ok := From(ctx)
	if !ok || g.Has(p) {
		return nil
	}
	return errs.Forbidden(g.Refusal(p, nil))
}

// RequireIn refuses, with 403 forbidden, an action that needs p for the plan when the person acting does
// not hold p in its scope (FR-009, FR-010; plan messages 1–3).
func RequireIn(ctx context.Context, p Permission, plan PlanFacts) error {
	g, ok := From(ctx)
	if !ok || g.InScope(p, plan) {
		return nil
	}
	return errs.Forbidden(g.Refusal(p, &plan))
}

// RequireView refuses, with 403 forbidden and message 4, a record of the plan that the person acting may
// not see (FR-008); recordType names the record mid-sentence ("plan", "campaign", …). The message names
// nothing of the record.
func RequireView(ctx context.Context, plan PlanFacts, recordType string) error {
	if CanView(ctx, plan) {
		return nil
	}
	return errs.Forbidden(NoAccessMessage(recordType))
}

// RequireViewActivity refuses, like RequireView, an activity (or a record under it, recordType) that the
// person acting may not see: they see it when they see its plan, own it or are named to approve it.
func RequireViewActivity(ctx context.Context, a ActivityFacts, recordType string) error {
	if CanViewActivity(ctx, a) {
		return nil
	}
	return errs.Forbidden(NoAccessMessage(recordType))
}

// Availability says whether the person may take one action on a record now, and why not (contract
// ActionAvailability): Reason is "" when Available, otherwise the exact message the API would answer.
type Availability struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
}

// Allowed is an available action.
var Allowed = Availability{Available: true, Reason: ""}

// Unavailable is an action that is not possible, with its reason.
func Unavailable(reason string) Availability {
	return Availability{Available: false, Reason: reason}
}

// Available says whether the person acting may take an action that needs p for the plan, with the refusal
// message as the reason when not (FR-011). An internal call may do everything.
func Available(ctx context.Context, p Permission, plan PlanFacts) Availability {
	g, ok := From(ctx)
	if !ok || g.InScope(p, plan) {
		return Allowed
	}
	return Unavailable(g.Refusal(p, &plan))
}

// AvailableAnywhere says whether the person acting holds p at all, with the refusal message when not.
func AvailableAnywhere(ctx context.Context, p Permission) Availability {
	g, ok := From(ctx)
	if !ok || g.Has(p) {
		return Allowed
	}
	return Unavailable(g.Refusal(p, nil))
}

// SelfApproval returns the segregation-of-duties reason (messages 6a, 6b) that stops the person acting
// from approving or declining an activity owned by ownerID whose open round submitterID submitted; "" when
// nothing stops them, also for an internal call.
func SelfApproval(ctx context.Context, ownerID, submitterID string) string {
	g, ok := From(ctx)
	if !ok {
		return ""
	}
	return g.SelfApprovalRefusal(ownerID, submitterID)
}
