package approval

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/actor"
)

func (f *fixture) resolveSubject(t *testing.T, sub subject, submitter actor.Actor, levels ...Level) resolution {
	t.Helper()
	var res resolution
	f.read(t, func(ctx context.Context) error {
		d, err := f.service.directory(ctx)
		if err != nil {
			return err
		}
		res, err = f.service.resolve(ctx, evaluate(levels, ruleFacts(sub), fixtureNames), resolving{sub: sub, submitter: submitter, d: d})
		return err
	})
	return res
}

func TestResolveOwnerAndRecordField(t *testing.T) {
	t.Parallel()
	owner := Level{ApproverType: SourceOwner, Mode: ModeAny}
	moa := Level{ApproverType: SourceRecordField, RecordField: RecordFieldMoAApprover, Mode: ModeAny}
	pr := func(ownerID, moaID string) subject {
		return subject{Kind: KindPurchaseRequest, ID: "PR-1001", OwnerID: ownerID, MoAApproverID: moaID}
	}
	tests := []struct {
		name      string
		sub       subject
		level     Level
		approvers []actor.Actor
		source    string
		excluded  string
		problem   string
	}{
		{name: "031 FR-010 the record's owner approves", sub: pr(carlos.ID, ""), level: owner,
			approvers: []actor.Actor{carlos}, source: recordOwnerSource},
		{name: "031 FR-011 the MoA approver approves", sub: pr(priya.ID, carlos.ID), level: moa,
			approvers: []actor.Actor{carlos}, source: "MoA approver"},
		{name: "031 FR-015 the owner may approve their own purchase request", sub: pr(priya.ID, priya.ID), level: moa,
			approvers: []actor.Actor{priya}, source: "MoA approver"},
		{name: "031 FR-012 an inactive MoA approver is left out", sub: pr(priya.ID, sara.ID), level: moa,
			approvers: []actor.Actor{}, source: "MoA approver", excluded: ReasonInactive},
		{name: "031 FR-012 a MoA approver who cannot approve is left out", sub: pr(priya.ID, mei.ID), level: moa,
			approvers: []actor.Actor{}, source: "MoA approver", excluded: ReasonCannotApprove},
		{name: "031 FR-012 an empty MoA approver is a problem", sub: pr(priya.ID, ""), level: moa, approvers: []actor.Actor{},
			source: "MoA approver", problem: "Level 1: MoA approver is empty on this purchase request."},
		{name: "031 FR-012 a record without owner is a problem", sub: pr("", ""), level: owner, approvers: []actor.Actor{},
			source: recordOwnerSource, problem: "Level 1: this purchase request has no owner."},
		{name: "031 FR-015 an activity's owner is left out while the rule is on", level: owner, approvers: []actor.Actor{},
			sub: subject{Kind: KindActivity, ID: "A-1001", OwnerID: priya.ID}, source: recordOwnerSource, excluded: ReasonOwner},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			f.deactivatePerson(t, sara)
			f.access.cannotApprove[mei.ID] = true

			res := f.resolveSubject(t, tt.sub, priya, tt.level)

			l := res.levels[0]
			problem := ""
			if len(res.problems) > 0 {
				problem = res.problems[0].Message
			}
			got := []any{l.Approvers, l.Source, excludedReason(l), problem}
			if diff := cmp.Diff([]any{tt.approvers, tt.source, tt.excluded, tt.problem}, got); diff != "" {
				t.Errorf("level (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveRecordFieldBackup(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-012 the backup approver takes a level whose MoA approver is left out", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		f.deactivatePerson(t, sara)
		sub := subject{Kind: KindPurchaseRequest, ID: "PR-1001", OwnerID: priya.ID, MoAApproverID: sara.ID}

		res := f.resolveSubject(t, sub, priya, Level{ApproverType: SourceRecordField, RecordField: RecordFieldMoAApprover,
			Mode: ModeAny, BackupApproverID: alex.ID})

		if l := res.levels[0]; !l.BackupUsed || !reflect.DeepEqual(l.Approvers, []actor.Actor{alex}) {
			t.Errorf("level = %+v, want the backup approver Alex Morgan", l)
		}
	})
}

func excludedReason(l RoundLevel) string {
	if len(l.Excluded) == 0 {
		return ""
	}
	return l.Excluded[0].Reason
}
