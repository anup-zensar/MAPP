package seed

import (
	"context"
	"fmt"
	"time"

	"mapp/internal/approval"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
	"mapp/internal/stores"
)

// day is 24 hours: the approval history is dated in days and hours before the seed time.
const day = 24 * time.Hour

// The decline comments of the demo story (008 data model §8).
const (
	splitComment  = "Please split the agency fee by country before re-submitting."
	budgetComment = "The planned cost is above this quarter's budget — please reduce it or move part of it to Q3."
)

// demoDecision is one decision on the current level of a round, ago before the seed time.
type demoDecision struct {
	by       string
	ago      time.Duration
	declined bool
	comment  string
}

// demoRound is one approval round of 008 data model §8: who submitted it and when, and the decisions
// taken since. A round whose decisions do not end it is open.
type demoRound struct {
	plan, activity string
	resubmitted    bool
	submittedBy    string
	submittedAgo   time.Duration
	decisions      []demoDecision
}

// approvedBy is the one round of an approved Lubricants activity: submitted 40 days before, approved by
// Daniel Okoro 38 days before.
func approvedBy(activity string) demoRound {
	return demoRound{plan: lubricantsPlan, activity: activity, submittedBy: "Priya Raman", submittedAgo: 40 * day,
		decisions: []demoDecision{{by: "Daniel Okoro", ago: 38 * day}}}
}

// demoRounds are AR-1001 … AR-1008, the rounds of the seeded statuses: Created activities have none.
var demoRounds = []demoRound{
	approvedBy("Social always-on"),
	approvedBy("Billboard winter wave"),
	{plan: lubricantsPlan, activity: "Trade magazine ads", submittedBy: "Priya Raman", submittedAgo: 9 * day,
		decisions: []demoDecision{{by: "Daniel Okoro", ago: 8 * day, declined: true, comment: splitComment}}},
	{plan: lubricantsPlan, activity: "Trade magazine ads", resubmitted: true, submittedBy: "Priya Raman", submittedAgo: 2 * day},
	{plan: lubricantsPlan, activity: "Fleet roadshow", submittedBy: "Priya Raman", submittedAgo: 6 * day,
		decisions: []demoDecision{{by: "Daniel Okoro", ago: 5 * day, declined: true, comment: budgetComment}}},
	approvedBy("Christmas trade gifting"),
	approvedBy("Partner co-marketing fund"),
	{plan: mobilityPlan, activity: "Fleet card launch", submittedBy: "Mei Lin", submittedAgo: 4 * day,
		decisions: []demoDecision{{by: "Carlos Mendes", ago: 2 * time.Hour}}},
}

// demoNotification is one notification of 008 data model §8. Text has one %s for "'<activity>' (<ID>)".
type demoNotification struct {
	recipient, plan, activity string
	kind                      approval.NotificationKind
	title, text               string
	by                        string
	ago                       time.Duration
	read                      bool
}

// approvedNotice tells Priya Raman, read, that Daniel Okoro approved one of her activities.
func approvedNotice(activity string) demoNotification {
	return demoNotification{recipient: "Priya Raman", plan: lubricantsPlan, activity: activity, kind: approval.NotifyApproved,
		title: "Activity approved", text: "Daniel Okoro approved %s", by: "Daniel Okoro", ago: 38 * day, read: true}
}

// declinedText is the text of a declined notification: the first 80 characters of the comment, then "…".
func declinedText(by, comment string) string {
	runes := []rune(comment)
	if len(runes) > 80 {
		comment = string(runes[:80]) + "…"
	}
	return by + " declined %s: \"" + comment + "\""
}

// demoNotifications are NT-1001 … NT-1009.
var demoNotifications = []demoNotification{
	approvedNotice("Social always-on"),
	approvedNotice("Billboard winter wave"),
	{recipient: "Priya Raman", plan: lubricantsPlan, activity: "Trade magazine ads", kind: approval.NotifyDeclined,
		title: "Activity declined", text: declinedText("Daniel Okoro", splitComment), by: "Daniel Okoro", ago: 8 * day, read: true},
	{recipient: "Daniel Okoro", plan: lubricantsPlan, activity: "Trade magazine ads", kind: approval.NotifyApprovalNeeded,
		title: "Approval needed", text: "Priya Raman re-submitted %s", by: "Priya Raman", ago: 2 * day},
	{recipient: "Priya Raman", plan: lubricantsPlan, activity: "Fleet roadshow", kind: approval.NotifyDeclined,
		title: "Activity declined", text: declinedText("Daniel Okoro", budgetComment), by: "Daniel Okoro", ago: 5 * day},
	approvedNotice("Christmas trade gifting"),
	approvedNotice("Partner co-marketing fund"),
	{recipient: "Alex Morgan", plan: mobilityPlan, activity: "Fleet card launch", kind: approval.NotifyApprovalNeeded,
		title: "Approval needed", text: "Carlos Mendes approved level 1 of %s — level 2 waits for you", by: "Carlos Mendes", ago: 2 * time.Hour},
	{recipient: "Mei Lin", plan: mobilityPlan, activity: "Fleet card launch", kind: approval.NotifyLevelApproved,
		title: "Level 1 approved", text: "Carlos Mendes approved level 1 of 2 of %s", by: "Carlos Mendes", ago: 2 * time.Hour},
}

// addApprovals creates the approval rounds with their history entries, then the notifications, through
// the approvals repository. The rounds match the activities' seeded statuses.
func (d *demo) addApprovals(ctx context.Context) error {
	if err := d.addApprovalRules(ctx); err != nil { // 017: timings, backups and WF-1003 first
		return err
	}
	repo := d.stores.Approvals
	activityRepo := d.stores.Activities
	peopleRepo := d.stores.People
	person := func(name string) (actor.Actor, error) {
		id, err := find(d.people, "person", name)
		if err != nil {
			return actor.Actor{}, err
		}
		p, _, err := peopleRepo.Get(ctx, id)
		return p.Actor(), err
	}

	numbers := map[string]int{} // activity ID → its last round number
	for _, dr := range demoRounds {
		activityID, err := find(d.activities, "activity", dr.plan+"/"+dr.activity)
		if err != nil {
			return err
		}
		activity, found, err := activityRepo.Get(ctx, activityID)
		if err != nil || !found {
			return cmpErr(err, fmt.Errorf("seed: activity %s is not stored", activityID))
		}
		workflow, found, err := repo.GetWorkflow(ctx, activity.ApprovalWorkflowID)
		if err != nil || !found {
			return cmpErr(err, fmt.Errorf("seed: activity %s has no approval workflow", activityID))
		}
		submitter, err := person(dr.submittedBy)
		if err != nil {
			return err
		}
		levels := make([]approval.RoundLevel, 0, len(workflow.Levels))
		for i, l := range workflow.Levels {
			level := approval.RoundLevel{Number: i + 1, Name: l.Name, Approvers: []actor.Actor{}}
			for _, approverID := range l.ApproverIDs {
				p, _, err := peopleRepo.Get(ctx, approverID)
				if err != nil {
					return err
				}
				level.Approvers = append(level.Approvers, p.Actor())
			}
			levels = append(levels, level)
		}

		numbers[activityID]++
		submittedAt := d.at.Add(-dr.submittedAgo)
		round := approval.Round{
			ActivityID: activityID, Number: numbers[activityID], WorkflowID: workflow.ID, WorkflowName: workflow.Name,
			Levels: levels, CurrentLevel: 1, Outcome: approval.OutcomeOpen, SubmittedAt: submittedAt, SubmittedBy: submitter,
			Meta: record.NewMeta(submitter, submittedAt),
		}
		round.Levels[0].StartedAt = &submittedAt
		if err := d.roundRules(ctx, &round, workflow, activity, d.plans[dr.plan]); err != nil {
			return err
		}
		if round.ID, err = d.next(ctx, ids.PrefixApprovalRound); err != nil {
			return err
		}
		submitted := approval.ActionSubmitted
		if dr.resubmitted {
			submitted = approval.ActionResubmitted
		}
		if err := d.addAction(ctx, repo, round, roundAction{kind: submitted, by: submitter, at: submittedAt}); err != nil {
			return err
		}
		for _, decision := range dr.decisions {
			by, err := person(decision.by)
			if err != nil {
				return err
			}
			at := d.at.Add(-decision.ago)
			action := approval.ActionApproved
			if decision.declined {
				action = approval.ActionDeclined
			}
			if err := d.addAction(ctx, repo, round, roundAction{kind: action, by: by, at: at, comment: decision.comment}); err != nil {
				return err
			}
			switch {
			case decision.declined:
				round.Outcome, round.EndedAt = approval.OutcomeDeclined, &at
			case round.CurrentLevel == len(round.Levels):
				round.Outcome, round.EndedAt = approval.OutcomeApproved, &at
			default:
				round.CurrentLevel++
				round.Levels[round.CurrentLevel-1].StartedAt = &at
			}
			round.Touch(by, at)
		}
		if err := repo.PutRound(ctx, round); err != nil {
			return err
		}
	}
	if err := d.addNotifications(ctx, repo, person); err != nil {
		return err
	}
	return d.addDelegation(ctx)
}

// roundAction is what one history entry of a round records: what was done, by whom, when and why.
type roundAction struct {
	kind    approval.ActionKind
	by      actor.Actor
	at      time.Time
	comment string
}

// addAction writes one history entry of round at its current level.
func (d *demo) addAction(ctx context.Context, repo stores.ApprovalStore, round approval.Round, action roundAction) error {
	id, err := d.next(ctx, ids.PrefixApprovalAction)
	if err != nil {
		return err
	}
	level := round.Levels[round.CurrentLevel-1]
	return repo.PutAction(ctx, approval.Action{
		ID: id, RoundID: round.ID, RoundNumber: round.Number, ActivityID: round.ActivityID, Action: action.kind,
		Level: level.Number, LevelName: level.Name, Approvers: level.Approvers, Actor: action.by, OnBehalfOf: []actor.Actor{},
		At: action.at, Comment: action.comment, Changes: []approval.ChangeText{},
	})
}

// addNotifications writes the demo notifications; read ones were read an hour after they came.
func (d *demo) addNotifications(ctx context.Context, repo stores.ApprovalStore, person func(string) (actor.Actor, error)) error {
	for _, dn := range demoNotifications {
		activityID, err := find(d.activities, "activity", dn.plan+"/"+dn.activity)
		if err != nil {
			return err
		}
		recipient, err := find(d.people, "person", dn.recipient)
		if err != nil {
			return err
		}
		by, err := person(dn.by)
		if err != nil {
			return err
		}
		at := d.at.Add(-dn.ago)
		notification := approval.Notification{
			RecipientID: recipient, Kind: dn.kind, Title: dn.title,
			Text:       fmt.Sprintf(dn.text, fmt.Sprintf("'%s' (%s)", dn.activity, activityID)),
			ActivityID: activityID, ActivityName: dn.activity, Actor: by, At: at, Read: dn.read,
		}
		if dn.read {
			readAt := at.Add(time.Hour)
			notification.ReadAt = &readAt
		}
		if notification.ID, err = d.next(ctx, ids.PrefixNotification); err != nil {
			return err
		}
		if err := repo.PutNotification(ctx, notification); err != nil {
			return err
		}
	}
	return nil
}
