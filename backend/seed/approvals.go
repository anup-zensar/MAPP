package seed

import (
	"context"
	"fmt"
	"time"

	"mapp/internal/activity"
	"mapp/internal/approval"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
	"mapp/internal/stores"
)

// day is 24 hours: the approval history is dated in days and hours before the seed time.
const day = 24 * time.Hour

const (
	splitComment  = "Please split the agency fee by country before re-submitting."
	budgetComment = "The planned cost is above this quarter's budget — please reduce it or move part of it to Q3."

	activityApprovedTitle = "Activity approved"
	activityDeclinedTitle = "Activity declined"
	approvalNeededTitle   = "Approval needed"
	levelOneApprovedTitle = "Level 1 approved"
)

type demoDecision struct {
	by       string
	ago      time.Duration
	declined bool
	comment  string
}

// demoRound is one approval round of 008 data model §8; a round whose decisions do not end it is open.
type demoRound struct {
	plan, activity string
	resubmitted    bool
	submittedBy    string
	submittedAgo   time.Duration
	decisions      []demoDecision
}

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

func approvedNotice(activity string) demoNotification {
	return demoNotification{recipient: priyaRaman, plan: lubricantsPlan, activity: activity, kind: approval.NotifyApproved,
		title: activityApprovedTitle, text: "Daniel Okoro approved %s", by: danielOkoro, ago: 38 * day, read: true}
}

// declinedText quotes the first 80 characters of the comment, then "…".
func declinedText(by, comment string) string {
	runes := []rune(comment)
	if len(runes) > 80 {
		comment = string(runes[:80]) + "…"
	}
	return by + " declined %s: \"" + comment + "\""
}

var demoNotifications = []demoNotification{
	approvedNotice("Social always-on"),
	approvedNotice("Billboard winter wave"),
	{recipient: priyaRaman, plan: lubricantsPlan, activity: "Trade magazine ads", kind: approval.NotifyDeclined,
		title: activityDeclinedTitle, text: declinedText(danielOkoro, splitComment), by: danielOkoro, ago: 8 * day, read: true},
	{recipient: danielOkoro, plan: lubricantsPlan, activity: "Trade magazine ads", kind: approval.NotifyApprovalNeeded,
		title: approvalNeededTitle, text: "Priya Raman re-submitted %s", by: priyaRaman, ago: 2 * day},
	{recipient: priyaRaman, plan: lubricantsPlan, activity: "Fleet roadshow", kind: approval.NotifyDeclined,
		title: activityDeclinedTitle, text: declinedText(danielOkoro, budgetComment), by: danielOkoro, ago: 5 * day},
	approvedNotice("Christmas trade gifting"),
	approvedNotice("Partner co-marketing fund"),
	{recipient: alexMorgan, plan: mobilityPlan, activity: "Fleet card launch", kind: approval.NotifyApprovalNeeded,
		title: approvalNeededTitle, text: "Carlos Mendes approved level 1 of %s — level 2 waits for you", by: carlosMendes, ago: 2 * time.Hour},
	{recipient: meiLin, plan: mobilityPlan, activity: "Fleet card launch", kind: approval.NotifyLevelApproved,
		title: levelOneApprovedTitle, text: "Carlos Mendes approved level 1 of 2 of %s", by: carlosMendes, ago: 2 * time.Hour},
}

// addApprovals creates the rounds with their history entries, then the notifications, as the statuses say.
func (d *demo) addApprovals(ctx context.Context) error {
	if err := d.addApprovalRules(ctx); err != nil {
		return err
	}
	if err := d.putRounds(ctx, demoRounds); err != nil {
		return err
	}
	if err := d.addNotifications(ctx, d.stores.Approvals, d.personActor(ctx)); err != nil {
		return err
	}
	return d.addDelegation(ctx)
}

func (d *demo) personActor(ctx context.Context) func(name string) (actor.Actor, error) {
	return func(name string) (actor.Actor, error) {
		id, err := find(d.people, "person", name)
		if err != nil {
			return actor.Actor{}, err
		}
		p, _, err := d.stores.People.Get(ctx, id)
		return p.Actor(), err
	}
}

func (d *demo) putRounds(ctx context.Context, list []demoRound) error {
	numbers := map[string]int{}
	for _, dr := range list {
		if err := d.putRound(ctx, dr, numbers); err != nil {
			return err
		}
	}
	return nil
}

func (d *demo) putRound(ctx context.Context, dr demoRound, numbers map[string]int) error {
	act, workflow, err := d.roundSubject(ctx, dr)
	if err != nil {
		return err
	}
	submitter, err := d.personActor(ctx)(dr.submittedBy)
	if err != nil {
		return err
	}
	levels, err := d.roundLevels(ctx, workflow)
	if err != nil {
		return err
	}
	numbers[act.ID]++
	submittedAt := d.at.Add(-dr.submittedAgo)
	round := approval.Round{
		SubjectKind: approval.KindActivity, SubjectID: act.ID, Number: numbers[act.ID], WorkflowID: workflow.ID,
		WorkflowName: workflow.Name,
		Levels:       levels, CurrentLevel: 1, Outcome: approval.OutcomeOpen, SubmittedAt: submittedAt, SubmittedBy: submitter,
		Meta: record.NewMeta(submitter, submittedAt),
	}
	round.Levels[0].StartedAt = &submittedAt
	if err = d.roundRules(ctx, &round, workflow, ruleSubject{act: act, planID: d.plans[dr.plan]}); err != nil {
		return err
	}
	if round.ID, err = d.next(ctx, ids.PrefixApprovalRound); err != nil {
		return err
	}
	submitted := approval.ActionSubmitted
	if dr.resubmitted {
		submitted = approval.ActionResubmitted
	}
	if err := d.addAction(ctx, d.stores.Approvals, round, roundAction{kind: submitted, by: submitter, at: submittedAt}); err != nil {
		return err
	}
	if err := d.decide(ctx, &round, dr.decisions); err != nil {
		return err
	}
	return d.stores.Approvals.PutRound(ctx, round)
}

func (d *demo) roundSubject(ctx context.Context, dr demoRound) (activity.Activity, approval.Workflow, error) {
	activityID, err := find(d.activities, "activity", dr.plan+"/"+dr.activity)
	if err != nil {
		return activity.Activity{}, approval.Workflow{}, err
	}
	act, found, err := d.stores.Activities.Get(ctx, activityID)
	if err != nil || !found {
		return activity.Activity{}, approval.Workflow{}, cmpErr(err, fmt.Errorf("seed: activity %s is not stored", activityID))
	}
	workflow, found, err := d.stores.Approvals.GetWorkflow(ctx, act.ApprovalWorkflowID)
	if err != nil || !found {
		return activity.Activity{}, approval.Workflow{}, cmpErr(err, fmt.Errorf("seed: activity %s has no approval workflow", activityID))
	}
	return act, workflow, nil
}

func (d *demo) roundLevels(ctx context.Context, workflow approval.Workflow) ([]approval.RoundLevel, error) {
	levels := make([]approval.RoundLevel, 0, len(workflow.Levels))
	for i, l := range workflow.Levels {
		level := approval.RoundLevel{Number: i + 1, Name: l.Name, Approvers: []actor.Actor{}}
		for _, approverID := range l.ApproverIDs {
			p, _, err := d.stores.People.Get(ctx, approverID)
			if err != nil {
				return nil, err
			}
			level.Approvers = append(level.Approvers, p.Actor())
		}
		levels = append(levels, level)
	}
	return levels, nil
}

func (d *demo) decide(ctx context.Context, round *approval.Round, decisions []demoDecision) error {
	for _, decision := range decisions {
		by, err := d.personActor(ctx)(decision.by)
		if err != nil {
			return err
		}
		at := d.at.Add(-decision.ago)
		action := approval.ActionApproved
		if decision.declined {
			action = approval.ActionDeclined
		}
		if err := d.addAction(ctx, d.stores.Approvals, *round, roundAction{kind: action, by: by, at: at, comment: decision.comment}); err != nil {
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
	return nil
}

type roundAction struct {
	kind    approval.ActionKind
	by      actor.Actor
	at      time.Time
	comment string
}

func (d *demo) addAction(ctx context.Context, repo stores.ApprovalStore, round approval.Round, action roundAction) error {
	id, err := d.next(ctx, ids.PrefixApprovalAction)
	if err != nil {
		return err
	}
	level := round.Levels[round.CurrentLevel-1]
	return repo.PutAction(ctx, approval.Action{
		ID: id, RoundID: round.ID, RoundNumber: round.Number, SubjectKind: round.SubjectKind, SubjectID: round.SubjectID,
		Action: action.kind,
		Level:  level.Number, LevelName: level.Name, Approvers: level.Approvers, Actor: action.by, OnBehalfOf: []actor.Actor{},
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
			Text:        fmt.Sprintf(dn.text, fmt.Sprintf("'%s' (%s)", dn.activity, activityID)),
			SubjectKind: approval.KindActivity, SubjectID: activityID, SubjectName: dn.activity, Actor: by, At: at, Read: dn.read,
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
