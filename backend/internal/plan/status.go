package plan

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// The messages of publishing and unpublishing (plan "User-visible messages").
const (
	msgPublishNeedsOne  = "Publish needs 1 field"
	msgPublishNeedsMany = "Publish needs %d fields"
	msgEnterReason      = "Enter a reason."
	msgReasonTooLong    = "Use 500 characters or fewer."
)

// maxReasonLength is the longest unpublish reason, in characters after trimming (FR-025).
const maxReasonLength = 500

// The verbs of the plan's status changes in the activity log (plan "Activity log").
const (
	verbPublished   = "published"
	verbUnpublished = "unpublished"
)

// Publish makes the Draft plan id the plan of record (FR-024, D-083), in one Write. It succeeds only when
// the plan and every record below that blocks publishing (campaigns and initiatives, 005) have every
// needed-to-publish field; otherwise it answers not_ready listing every missing field, the plan first, then
// the records below in order, and changes nothing. Remaining budget, campaign dates, activities and line
// items never block. Publishing a Published plan changes nothing and logs nothing.
func (s *Service) Publish(ctx context.Context, a actor.Actor, id string) (PlanView, error) {
	var view PlanView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		p, err := s.stored(ctx, id)
		if err != nil {
			return err
		}
		if view, err = s.view(ctx, a, p); err != nil {
			return err
		}
		if p.Status == StatusPublished {
			return nil
		}
		if missing, count := readiness(p, view.Completeness); count > 0 {
			return errs.NotReady(publishNeeds(count), missing)
		}
		view, err = s.changeStatus(ctx, a, p, StatusPublished, verbPublished, "")
		return err
	})
	if err != nil {
		return PlanView{}, err
	}
	return view, nil
}

// Unpublish returns the Published plan id to Draft (FR-025, D-083), in one Write, and records the reason
// in the activity log. The reason, without its surrounding spaces, has 1 to 500 characters. Unpublishing a
// Draft plan changes nothing and logs nothing.
func (s *Service) Unpublish(ctx context.Context, a actor.Actor, id string, request UnpublishRequest) (PlanView, error) {
	reason := strings.TrimSpace(request.Reason)
	var problems errs.Problems
	switch {
	case reason == "":
		problems.Add("reason", msgEnterReason)
	case utf8.RuneCountInString(reason) > maxReasonLength:
		problems.Add("reason", msgReasonTooLong)
	}
	if err := problems.Err(); err != nil {
		return PlanView{}, err
	}
	var view PlanView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		p, err := s.stored(ctx, id)
		if err != nil {
			return err
		}
		if p.Status == StatusDraft {
			view, err = s.view(ctx, a, p)
			return err
		}
		view, err = s.changeStatus(ctx, a, p, StatusDraft, verbUnpublished, reason)
		return err
	})
	if err != nil {
		return PlanView{}, err
	}
	return view, nil
}

// changeStatus gives the stored plan p the status to inside the caller's Write: a new version, and one
// "statusChanged" entry with the verb, the status change and the person's comment. It returns the plan as
// the API returns it after the change.
func (s *Service) changeStatus(ctx context.Context, a actor.Actor, p Plan, to Status, verb, comment string) (PlanView, error) {
	before := p
	p.Status = to
	p.Touch(a, s.now())
	changes, err := planChanges(before, p) // the status alone
	if err != nil {
		return PlanView{}, err
	}
	if err := s.repo.Put(ctx, p); err != nil {
		return PlanView{}, err
	}
	if _, err := s.history.Log(ctx, a, history.LogInput{
		Kind: kindName, RecordID: p.ID, RecordName: p.Name, Action: history.ActionStatusChanged, Verb: verb,
		Changes: changes, Comment: comment,
	}); err != nil {
		return PlanView{}, err
	}
	return s.view(ctx, a, p)
}

// readiness lists what stops the plan p from being published, grouped by record: the plan's own missing
// fields first, then each record below that blocks publishing, in the levels' order (FR-024). It also
// returns how many fields are missing in all; zero means p can be published.
func readiness(p Plan, c Completeness) ([]errs.ReadinessGroup, int) {
	groups := []errs.ReadinessGroup{}
	count := 0
	if len(c.Missing) > 0 {
		groups = append(groups, errs.ReadinessGroup{
			Record: errs.RecordRef{Type: kindName, ID: p.ID, Name: p.Name}, Fields: c.Missing,
		})
		count += len(c.Missing)
	}
	for _, item := range c.ItemsBelow {
		if !item.BlocksPublish || len(item.Missing) == 0 {
			continue
		}
		groups = append(groups, errs.ReadinessGroup{Record: item.Record, Fields: item.Missing})
		count += len(item.Missing)
	}
	return groups, count
}

// publishNeeds is the message of a refused publish: "Publish needs 1 field", "Publish needs 3 fields".
func publishNeeds(count int) string {
	if count == 1 {
		return msgPublishNeedsOne
	}
	return fmt.Sprintf(msgPublishNeedsMany, count)
}
