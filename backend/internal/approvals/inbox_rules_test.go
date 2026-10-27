package approvals

import (
	"slices"
	"testing"
	"time"
)

// Waiting for me holds what the person may decide as approver, delegate or backup approver after
// escalation — never on behalf, and never what they own or submitted (017 FR-031); the summary counts it
// (SC-007).
func TestInbox_DelegateAndBackup_FR031(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.putWorkflow(t, "WF-1003", "Backed up", days(1), days(2), Level{ApproverIDs: []string{daniel.ID}, BackupApproverID: carlos.ID})
	f.addActivity(t, "A-1001", "Escalated", priya.ID, "WF-1003")
	f.addActivity(t, "A-1002", "Mei's", mei.ID, lubricantsFlow)
	f.submit(t, priya, "A-1001", "")
	f.submit(t, priya, "A-1002", "")
	f.delegate(t, "DLG-1001", daniel, mei, f.today(), f.today().AddDays(5))
	f.clock.Advance(49 * time.Hour)
	f.mustRun(t)

	if ids := f.inboxIDs(t, carlos, TabWaiting); !slices.Equal(ids, []string{"A-1001"}) {
		t.Errorf("Carlos (backup) = %v, want A-1001", ids)
	}
	if ids := f.inboxIDs(t, mei, TabWaiting); !slices.Equal(ids, []string{"A-1001"}) {
		t.Errorf("Mei (delegate, owner of A-1002) = %v, want A-1001 only", ids)
	}
	if ids := f.inboxIDs(t, alex, TabWaiting); len(ids) != 0 {
		t.Errorf("Alex (on behalf) = %v, want nothing", ids)
	}
	counts, err := f.service.Summary(t.Context(), mei, nil, nil)
	if err != nil || counts.WaitingForMe != 1 || counts.AllOpen != 2 {
		t.Errorf("Mei's counts = %+v, %v; want 1 waiting of 2 open", counts, err)
	}
}

// The rows flag what needs attention (017 FR-031): the level's mode and "<a> of <b> approved", the whole
// days waited, overdue, the backup approver of an escalated level, and "no approver" when nobody waiting can
// act.
func TestInbox_RowFlags_FR031(t *testing.T) { //nolint:gocyclo // 14: every flag of both rows, checked in one condition per row
	t.Parallel()
	f := newFixture(t)
	f.putWorkflow(t, "WF-1003", "All", days(1), days(3), Level{ApproverIDs: []string{daniel.ID, carlos.ID}, Mode: ModeAll, BackupApproverID: alex.ID})
	f.putWorkflow(t, "WF-1004", "Own", nil, nil, Level{ApproverIDs: []string{daniel.ID}})
	f.addActivity(t, "A-1001", "All of them", priya.ID, "WF-1003")
	f.addActivity(t, "A-1002", "Daniel's own", daniel.ID, "WF-1004")
	f.submit(t, priya, "A-1001", "")
	f.submit(t, daniel, "A-1002", "")
	f.approve(t, daniel, "A-1001", "")
	f.clock.Advance(3*24*time.Hour + time.Hour)
	f.mustRun(t)

	rows := f.inbox(t, alex, TabOpen)
	all, own := rows[0], rows[1]
	if all.Round.Mode != ModeAll || all.Round.ApprovedCount != 1 || all.Round.RequiredCount != 2 || *all.WaitingDays != 3 ||
		!all.Overdue || all.EscalatedTo == nil || all.EscalatedTo.Name != "Alex Morgan" || all.NoApprover {
		t.Errorf("row = %+v, want 1 of 2 approved, 3 days, overdue, escalated to Alex Morgan", all)
	}
	if !own.NoApprover || own.Overdue || own.EscalatedTo != nil || *own.WaitingDays != 3 {
		t.Errorf("row = %+v, want no approver and never overdue", own)
	}
	f.access.cannotApprove[carlos.ID] = true
	if rows := f.inbox(t, alex, TabOpen); !rows[0].NoApprover {
		t.Error("a level whose only waiting approver can no longer approve is not flagged")
	}
}
