package approval

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// RunEscalations sends the reminders and escalations due now, standing in
// for the nightly job; it needs workflows.manage. Each level gets at most one of each, so it is safe to repeat.
func (s *Service) RunEscalations(ctx context.Context, a actor.Actor) (EscalationRun, error) {
	run := EscalationRun{Events: []EscalationEvent{}, NotEscalated: []EscalationEvent{}}
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		grant, err := s.access.Check(ctx, a.ID, PermissionManageWorkflows, nil)
		if err != nil {
			return err
		}
		if !grant.Allowed {
			return errs.Forbidden(grant.Refusal)
		}
		b, err := s.board(ctx)
		if err != nil {
			return err
		}
		rule, err := s.accessRule(ctx)
		if err != nil {
			return err
		}
		stored := slices.Clone(b.subjects)
		slices.SortStableFunc(stored, func(x, y subject) int { return cmp.Compare(x.ID, y.ID) })
		for _, sub := range stored {
			round, found := openRound(b.rounds[sub.ID])
			if !found || !sub.waiting() {
				continue
			}
			e := escalating{runner: a, b: b, rule: rule, sub: sub.as(round.kind()), round: round, run: &run}
			if err := s.escalate(ctx, e); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return EscalationRun{}, err
	}
	return run, nil
}

// escalating is the current level of one open round inside RunEscalations' Write.
type escalating struct {
	runner  actor.Actor
	b       *board
	rule    bool
	sub     subject
	round   Round
	run     *EscalationRun
	entries []Action
	days    int
	where   string
	event   EscalationEvent
	notify  *notifier
}

func (s *Service) escalate(ctx context.Context, e escalating) error {
	e.entries = entriesOf(e.b.entries[e.sub.ID], e.round.ID)
	level := e.round.Levels[e.round.CurrentLevel-1]
	since := waitingSince(e.round, level, e.entries)
	if since == nil || level.Resolution != ResolutionDecide {
		return nil
	}
	e.days = waitedDays(*since, e.b.now)
	e.event = EscalationEvent{Activity: e.sub.ref(), RoundNumber: e.round.Number, Level: level.Number, To: []PersonView{}, WaitedDays: e.days}
	e.notify = s.notifier(actor.System, e.sub, e.b.now)
	e.where = fmt.Sprintf("Round %d · level %d of %d · waited %s", e.round.Number, level.Number, len(e.round.Levels), daysText(e.days))
	var err error
	switch e.due(level) {
	case EventEscalated:
		err = s.escalateLevel(ctx, &e)
	case EventNotEscalated:
		e.event.Action = EventNotEscalated
		e.run.NotEscalated = append(e.run.NotEscalated, e.event)
		return nil
	case EventReminded:
		err = s.remindLevel(ctx, &e)
	default:
		return nil
	}
	if err != nil {
		return err
	}
	e.round.Touch(actor.System, e.b.now)
	if err = s.repo.PutRound(ctx, e.round); err != nil {
		return err
	}
	e.run.Events = append(e.run.Events, e.event)
	return nil
}

// due is what the level needs now: an escalation (or the report that none can be made), a reminder, or "".
func (e escalating) due(level RoundLevel) string {
	escalation := e.round.EscalationDays != nil && e.days >= *e.round.EscalationDays && level.EscalatedAt == nil
	reminder := e.round.ReminderDays != nil && e.days >= *e.round.ReminderDays && level.RemindedAt == nil
	switch {
	case escalation && e.backupUsable(level):
		return EventEscalated
	case escalation:
		return EventNotEscalated
	case reminder:
		return EventReminded
	}
	return ""
}

// backupUsable: active, approves, not an approver of the level, and — for activities while 014's rule is on —
// neither the owner nor the round's submitter.
func (e escalating) backupUsable(level RoundLevel) bool {
	backup := level.Backup
	if backup == nil || !e.b.people.byID[backup.ID].Active || !e.b.canApprove[backup.ID] || includes(level.Approvers, backup.ID) {
		return false
	}
	self := backup.ID == e.sub.OwnerID || backup.ID == e.round.SubmittedBy.ID
	return !e.rule || !e.sub.Kind.barsSelfApproval() || !self
}

func (s *Service) escalateLevel(ctx context.Context, e *escalating) error {
	l := &e.round.Levels[e.round.CurrentLevel-1]
	backup := l.Backup
	l.EscalatedAt = &e.b.now
	if l.RemindedAt == nil {
		l.RemindedAt = &e.b.now
	}
	entry, err := s.newAction(ctx, e.round, Action{Action: ActionEscalated, Level: l.Number, Actor: actor.System, At: e.b.now})
	if err != nil {
		return err
	}
	entry.EscalatedTo, entry.WaitedDays = backup, e.days
	if err = s.repo.PutAction(ctx, entry); err != nil {
		return err
	}
	err = s.logOn(ctx, actor.System, e.sub, statusLog{
		verb: "escalated", details: fmt.Sprintf("%s · escalated to %s · run by %s", e.where, backup.Name, e.runner.Name),
	})
	if err != nil {
		return err
	}
	err = e.notify.send(ctx, []string{backup.ID}, NotifyEscalated, escalatedToYouTitle,
		fmt.Sprintf(escalatedToYouText, subjectText(e.sub), daysText(e.days), l.Number))
	if err == nil {
		err = e.notify.send(ctx, append(idsOf(waitingOn(*l, e.entries)), e.sub.OwnerID), NotifyEscalated, escalatedTitle,
			fmt.Sprintf(escalatedText, subjectText(e.sub), daysText(e.days), l.Number, backup.Name))
	}
	e.event.Action, e.event.To = EventEscalated, []PersonView{e.b.people.copied(*backup)}
	e.run.Escalated++
	return err
}

func (s *Service) remindLevel(ctx context.Context, e *escalating) error {
	l := &e.round.Levels[e.round.CurrentLevel-1]
	l.RemindedAt = &e.b.now
	told, err := s.remind(ctx, e, *l)
	if err != nil {
		return err
	}
	entry, err := s.newAction(ctx, e.round, Action{Action: ActionReminded, Level: l.Number, Actor: actor.System, At: e.b.now})
	if err != nil {
		return err
	}
	entry.WaitedDays = e.days
	if err = s.repo.PutAction(ctx, entry); err != nil {
		return err
	}
	err = s.logOn(ctx, actor.System, e.sub, statusLog{
		verb: "sent a reminder about", details: fmt.Sprintf("%s · to %s · run by %s", e.where, names(e.b.people.actors(told)), e.runner.Name),
	})
	e.event.Action = EventReminded
	for _, id := range told {
		e.event.To = append(e.event.To, e.b.people.person(id))
	}
	e.run.Reminded++
	return err
}

// remind tells the waiting approvers, then their current delegates, and returns who was told.
func (s *Service) remind(ctx context.Context, e *escalating, level RoundLevel) ([]string, error) {
	var told []string
	waiting := waitingOn(level, e.entries)
	text := fmt.Sprintf(reminderText, subjectText(e.sub), daysText(e.days), level.Number)
	title := fmt.Sprintf(reminderTitle, daysText(e.days))
	for _, w := range waiting {
		if err := e.notify.send(ctx, []string{w.ID}, NotifyReminder, title, text); err != nil {
			return nil, err
		}
		told = append(told, w.ID)
	}
	for _, w := range waiting {
		delegates := delegatesOf(e.b.delegations, w.ID)
		if err := e.notify.send(ctx, delegates, NotifyReminder, title, text+fmt.Sprintf(reminderAsDelegate, w.Name)); err != nil {
			return nil, err
		}
		told = append(told, delegates...)
	}
	return unique(told), nil
}
