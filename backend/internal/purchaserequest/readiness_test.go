package purchaserequest

import (
	"slices"
	"testing"
)

func TestGates_ActivitiesSwbsAndCompanyCodes(t *testing.T) {
	t.Parallel()
	facts := map[string]LineItemFacts{
		"L-1006": {ID: "L-1006", Name: "Stand build", ActivityID: "A-1003", ActivityName: "Billboard winter wave",
			ActivityStatus: "approved", SwbsCode: "SIM-GB01-0001", SwbsStatus: "received", CompanyCode: "GB01"},
		"L-1005": {ID: "L-1005", Name: "Media booking", ActivityID: "A-1004", ActivityName: "Trade magazine ads",
			ActivityStatus: "submitted", SwbsStatus: "waitingForApproval", CompanyCode: "NL02"},
		"L-1001": {ID: "L-1001", Name: "Agency fee", ActivityID: "A-1001", ActivityName: "Launch film",
			ActivityStatus: "approved", SwbsCode: "SIM-X", SwbsStatus: "readyToSend"},
	}
	tests := []struct {
		name  string
		lines []string
		want  []string
	}{
		{name: "032 FR-021 an approved activity with its SWBS code passes", lines: []string{"L-1006"}},
		{name: "032 FR-021 activity not approved and SWBS code missing", lines: []string{"L-1005"}, want: []string{
			"Activity 'Trade magazine ads' must be approved before this purchase request can be submitted.",
			"Line item L-1005 'Media booking' has no SWBS code yet (Waiting for approval).",
		}},
		{name: "032 FR-021 two company codes", lines: []string{"L-1006", "L-1005"}, want: []string{
			"Activity 'Trade magazine ads' must be approved before this purchase request can be submitted.",
			"Line item L-1005 'Media booking' has no SWBS code yet (Waiting for approval).",
			"One purchase request cannot charge two company codes (GB01 and NL02).",
		}},
		{name: "032 FR-021 no company code", lines: []string{"L-1001"}, want: []string{"Activity 'Launch film' has no company code."}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pr := PurchaseRequest{ID: "PR-1"}
			for _, id := range tt.lines {
				pr.Lines = append(pr.Lines, Line{LineItemID: id})
			}
			var got []string
			for _, p := range append(activityGates(pr, facts), companyCodeGates(pr, facts)...) {
				got = append(got, p.Message)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("gates = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSubmissionKind(t *testing.T) {
	t.Parallel()
	v := &Version{}
	tests := []struct {
		name string
		pr   PurchaseRequest
		want RoundKind
	}{
		{name: "R5 a created purchase request starts its first round", pr: PurchaseRequest{Status: StatusCreated}, want: KindPurchaseRequest},
		{name: "R5 a declined one too", pr: PurchaseRequest{Status: StatusDeclined}, want: KindPurchaseRequest},
		{name: "R5 an approved one starts a change", pr: PurchaseRequest{Status: StatusApproved, Approved: v}, want: KindModification},
		{name: "R5 a change in review re-submits the change", pr: PurchaseRequest{Status: StatusInReview, ChangePending: true, Approved: v},
			want: KindModification},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := submissionKind(tt.pr); got != tt.want {
				t.Errorf("submissionKind = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestFirstRoundTransitions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		status Status
		event  RoundEvent
		allows bool
	}{
		{StatusCreated, EventOpen, true}, {StatusDeclined, EventOpen, true}, {StatusSubmitted, EventOpen, true},
		{StatusSubmitted, EventApproved, true}, {StatusInReview, EventDeclined, true}, {StatusSubmitted, EventSuperseded, true},
		{StatusCreated, EventApproved, false}, {StatusApproved, EventOpen, false}, {StatusClosed, EventRecalled, false},
	}
	for _, tt := range tests {
		if got := firstRoundAllows(tt.status, tt.event); got != tt.allows {
			t.Errorf("032 FR-022 %s on %s allowed = %v, want %v", tt.event, tt.status, got, tt.allows)
		}
	}
	if opened(StatusCreated) != StatusSubmitted || opened(StatusSubmitted) != StatusInReview {
		t.Errorf("032 FR-022 opened: created → %s, submitted → %s", opened(StatusCreated), opened(StatusSubmitted))
	}
}
