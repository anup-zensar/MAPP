package approval

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// RunEscalations sends the reminders and escalations that are due now (017 FR-026 – FR-028, D-285, D-286):
// it stands in for the nightly job that will call it once MAPP is hosted, and needs workflows.manage
// (403 otherwise). In one Write, for the open round of each stored activity, by activity id, at its current
// level:
//
//   - escalation due — the round escalates and the level has waited at least its escalation days and is not
//     escalated yet: when the level's backup approver can take it (active, holding approvals.decide, not the
//     owner or the submitter while 014's access rule is on, not an approver of the level) the level is
//     escalated to them — they may then decide it — and they, the approvers still waiting and the owner are
//     told; a reminder due at the same time is not sent separately. Otherwise nothing changes and the level
//     is reported in notEscalated;
//   - otherwise a reminder due — the level has waited at least the reminder days and had no reminder — the
//     approvers still waiting and their current delegates are reminded.
//
// Each level gets at most one reminder and one escalation, so the run is safe to repeat (SC-005). Its events
// are recorded as made by MAPP, with "run by <person>" in the activity log.
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
		stored := slices.Clone(b.activities)
		slices.SortStableFunc(stored, func(x, y ActivityFacts) int { return cmp.Compare(x.ID, y.ID) })
		for _, facts := range stored {
			round, found := openRound(b.rounds[facts.ID])
			if !found || facts.Status != StatusSubmitted {
				continue
			}
			if err := s.escalate(ctx, a, b, rule, facts, round, &run); err != nil {
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

// escalate sends what is due for the current level of round, the open round of facts, inside RunEscalations'
// Write, and adds it to run.
func (s *Service) escalate(ctx context.Context, a actor.Actor, b *board, rule bool, facts ActivityFacts, round Round, run *EscalationRun) error {
	entries := entriesOf(b.entries[facts.ID], round.ID)
	level := round.Levels[round.CurrentLevel-1]
	since := waitingSince(round, level, entries)
	if since == nil || level.Resolution != ResolutionDecide {
		return nil
	}
	days := waitedDays(*since, b.now)
	event := EscalationEvent{
		Activity:    errs.RecordRef{Type: "activity", ID: facts.ID, Name: facts.Name},
		RoundNumber: round.Number, Level: level.Number, To: []PersonView{}, WaitedDays: days,
	}
	waiting := waitingOn(level, entries)
	notify := s.notifier(actor.System, facts, b.now)
	where := fmt.Sprintf("Round %d · level %d of %d · waited %s", round.Number, level.Number, len(round.Levels), daysText(days))
	l := &round.Levels[round.CurrentLevel-1]

	switch {
	case round.EscalationDays != nil && days >= *round.EscalationDays && level.EscalatedAt == nil:
		backup := level.Backup
		usable := backup != nil && b.people.byID[backup.ID].Active && b.canApprove[backup.ID] && !includes(level.Approvers, backup.ID) &&
			!(rule && (backup.ID == facts.OwnerID || backup.ID == round.SubmittedBy.ID))
		if !usable {
			event.Action = EventNotEscalated
			run.NotEscalated = append(run.NotEscalated, event)
			return nil
		}
		l.EscalatedAt = &b.now
		if l.RemindedAt == nil {
			l.RemindedAt = &b.now
		}
		entry, err := s.newAction(ctx, round, Action{Action: ActionEscalated, Level: level.Number, Actor: actor.System, At: b.now})
		if err != nil {
			return err
		}
		entry.EscalatedTo, entry.WaitedDays = backup, days
		if err := s.repo.PutAction(ctx, entry); err != nil {
			return err
		}
		err = s.logOnActivity(ctx, actor.System, facts, statusLog{
			verb: "escalated", details: fmt.Sprintf("%s · escalated to %s · run by %s", where, backup.Name, a.Name),
		})
		if err != nil {
			return err
		}
		err = notify.send(ctx, []string{backup.ID}, NotifyEscalated, escalatedToYouTitle,
			fmt.Sprintf(escalatedToYouText, activityText(facts), daysText(days), level.Number))
		if err == nil {
			err = notify.send(ctx, append(idsOf(waiting), facts.OwnerID), NotifyEscalated, escalatedTitle,
				fmt.Sprintf(escalatedText, activityText(facts), daysText(days), level.Number, backup.Name))
		}
		if err != nil {
			return err
		}
		event.Action, event.To = EventEscalated, []PersonView{b.people.copied(*backup)}
		run.Escalated++
	case round.ReminderDays != nil && days >= *round.ReminderDays && level.RemindedAt == nil:
		l.RemindedAt = &b.now
		var told []string
		text := fmt.Sprintf(reminderText, activityText(facts), daysText(days), level.Number)
		title := fmt.Sprintf(reminderTitle, daysText(days))
		for _, w := range waiting {
			if err := notify.send(ctx, []string{w.ID}, NotifyReminder, title, text); err != nil {
				return err
			}
			told = append(told, w.ID)
		}
		for _, w := range waiting {
			delegates := delegatesOf(b.delegations, w.ID)
			if err := notify.send(ctx, delegates, NotifyReminder, title, text+fmt.Sprintf(reminderAsDelegate, w.Name)); err != nil {
				return err
			}
			told = append(told, delegates...)
		}
		told = unique(told)
		entry, err := s.newAction(ctx, round, Action{Action: ActionReminded, Level: level.Number, Actor: actor.System, At: b.now})
		if err != nil {
			return err
		}
		entry.WaitedDays = days
		if err := s.repo.PutAction(ctx, entry); err != nil {
			return err
		}
		to := b.people.actors(told)
		err = s.logOnActivity(ctx, actor.System, facts, statusLog{
			verb: "sent a reminder about", details: fmt.Sprintf("%s · to %s · run by %s", where, names(to), a.Name),
		})
		if err != nil {
			return err
		}
		event.Action = EventReminded
		for _, id := range told {
			event.To = append(event.To, b.people.person(id))
		}
		run.Reminded++
	default:
		return nil
	}
	round.Touch(actor.System, b.now)
	if err := s.repo.PutRound(ctx, round); err != nil {
		return err
	}
	run.Events = append(run.Events, event)
	return nil
}
