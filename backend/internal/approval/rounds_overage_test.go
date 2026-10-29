package approval

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

type fixedOverspend bool

func (r fixedOverspend) RefusesOverPlannedCost(context.Context) (bool, error) {
	return bool(r), nil
}

const overageRefusal = "Line items of activity 'Fleet card launch' are 25,000.00 USD above its planned cost — " +
	"ask the owner to lower them, or decline."

// overspentActivity is A-1009 on Daniel's workflow, submitted by Mei, its line items 25,000.00 USD above its cost.
func overspentActivity(t *testing.T, refuse bool) *fixture {
	t.Helper()
	f := newFixture(t)
	f.service.UseApprovalOverspend(fixedOverspend(refuse))
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, lubricantsFlow)
	f.submit(t, mei, "A-1009", "")
	facts := f.activities.facts["A-1009"]
	over := money.USD(mustAmount(t, "25000.00"))
	facts.OverPlannedCost = &over
	f.activities.facts["A-1009"] = facts
	return f
}

func fleetCardOverage(t *testing.T) []Overage {
	t.Helper()
	return []Overage{{Activity: errs.RecordRef{Type: "activity", ID: "A-1009", Name: "Fleet card launch"},
		OverBy: money.USD(mustAmount(t, "25000.00"))}}
}

func TestApprove_WarnsOverPlannedCost_038_FR019_FR021_M18(t *testing.T) {
	t.Parallel()
	f := overspentActivity(t, false)
	before, err := f.service.Approval(t.Context(), daniel, "A-1009")
	if err != nil {
		t.Fatal(err)
	}

	f.approve(t, daniel, "A-1009", "")

	if diff := cmp.Diff(fleetCardOverage(t), before.OverPlannedCost); diff != "" || !before.Actions.Approve.Available {
		t.Errorf("038 FR-019 section before approving: approve %+v, overages (-want +got):\n%s", before.Actions.Approve, diff)
	}
	want := "Daniel Okoro approved activity 'Fleet card launch' [submitted → approved] " +
		"(Round 1 · level 1 of 1 · while line items were 25,000.00 USD above planned cost)"
	if got := logLine(f.log(t, "A-1009")[0]); got != want {
		t.Errorf("038 FR-019 log = %s, want %s", got, want)
	}
}

func TestApprove_RefusedOverPlannedCost_038_FR020_M18(t *testing.T) {
	t.Parallel()
	f := overspentActivity(t, true)
	round := f.openRound(t, "A-1009")
	view, err := f.service.Approval(t.Context(), daniel, "A-1009")
	if err != nil {
		t.Fatal(err)
	}

	_, refused := f.service.Approve(t.Context(), daniel, "A-1009", DecisionInput{RoundID: round.ID, Level: version(1)})

	assertError(t, refused, errs.NotAllowed(overageRefusal))
	if after := f.openRound(t, "A-1009"); after.Version != round.Version || f.activities.facts["A-1009"].Status != StatusSubmitted {
		t.Errorf("038 FR-020 something changed: round %+v", after)
	}
	if view.Actions.Approve != (ActionAvailability{Reason: overageRefusal}) || !view.Actions.Decline.Available {
		t.Errorf("038 FR-020 actions = %+v, want Approve refused with the reason, Decline offered", view.Actions)
	}
	if declined := f.decline(t, daniel, "A-1009", "Lower the line items"); declined.Status != StatusDeclined {
		t.Errorf("038 FR-020 decline = %s, want declined", declined.Status)
	}
}

func TestDecideSubject_RefusedOverPlannedCost_038_FR020_M18(t *testing.T) {
	t.Parallel()
	for _, kind := range []SubjectKind{KindPurchaseRequest, KindModifiedPurchaseRequest} {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			f, prs := newPurchaseFixture(t, purchaseRequest{workflow: prFlow, modification: prChangesFlow},
				kindFlow(prFlow, KindPurchaseRequest, named(daniel)), kindFlow(prChangesFlow, KindModifiedPurchaseRequest, named(daniel)))
			f.service.UseApprovalOverspend(fixedOverspend(true))
			if kind == KindModifiedPurchaseRequest {
				f.submitPR(t, KindPurchaseRequest)
				f.mustDecide(t, daniel, ActionApproved, "")
			}
			f.submitPR(t, kind)
			facts := prs.facts[purchaseRequestA]
			facts.Overages = fleetCardOverage(t)
			prs.facts[purchaseRequestA] = facts

			_, refused := f.decideSubject(t, daniel, ActionApproved, "")

			assertError(t, refused, errs.NotAllowed(overageRefusal))
			f.mustDecide(t, daniel, ActionDeclined, "Lower the line items")
		})
	}
}
