// Package approvaltest is the in-memory fake of the approvals store (workflows, rounds, history entries,
// notifications and delegations), for tests that build services or the whole API without a database.
package approvaltest

import (
	"cmp"
	"context"
	"slices"
	"time"

	"mapp/internal/approval"
	"mapp/internal/platform/fake"
)

type Store struct {
	workflows     *fake.Table[approval.Workflow]
	rounds        *fake.Table[approval.Round]
	actions       *fake.Table[approval.Action]
	notifications *fake.Table[approval.Notification]
	delegations   *fake.Table[approval.Delegation]
}

func NewStore(db *fake.DB) Store {
	return Store{
		workflows:     fake.TableOf(db, "approval_workflows", func(w approval.Workflow) string { return w.ID }),
		rounds:        fake.TableOf(db, "approval_rounds", func(r approval.Round) string { return r.ID }),
		actions:       fake.TableOf(db, "approval_actions", func(a approval.Action) string { return a.ID }),
		notifications: fake.TableOf(db, "notifications", func(n approval.Notification) string { return n.ID }),
		delegations:   fake.TableOf(db, "approval_delegations", func(dl approval.Delegation) string { return dl.ID }),
	}
}

func (s Store) GetWorkflow(_ context.Context, id string) (approval.Workflow, bool, error) {
	return s.workflows.Get(id)
}

func (s Store) ListWorkflows(context.Context) ([]approval.Workflow, error) {
	return s.workflows.List()
}

func (s Store) PutWorkflow(ctx context.Context, w approval.Workflow) error {
	return s.workflows.Put(ctx, normalisedWorkflow(w))
}

func (s Store) DeleteWorkflow(ctx context.Context, id string) error {
	return s.workflows.Delete(ctx, id)
}

func (s Store) GetRound(_ context.Context, id string) (approval.Round, bool, error) {
	return s.rounds.Get(id)
}

func (s Store) ListRounds(context.Context) ([]approval.Round, error) {
	return s.rounds.List()
}

func (s Store) RoundsOf(_ context.Context, subjectID string) ([]approval.Round, error) {
	rounds, err := s.rounds.Where(func(r approval.Round) bool { return r.SubjectID == subjectID })
	slices.SortStableFunc(rounds, func(a, b approval.Round) int { return cmp.Compare(a.Number, b.Number) })
	return rounds, err
}

func (s Store) RoundsOfSubjects(_ context.Context, subjectIDs []string) ([]approval.Round, error) {
	return s.rounds.Where(func(r approval.Round) bool { return slices.Contains(subjectIDs, r.SubjectID) })
}

func (s Store) OpenRounds(context.Context) ([]approval.Round, error) {
	return s.rounds.Where(func(r approval.Round) bool { return r.Outcome == approval.OutcomeOpen })
}

func (s Store) PutRound(ctx context.Context, r approval.Round) error {
	return s.rounds.Put(ctx, normalisedRound(r))
}

func (s Store) PutAction(ctx context.Context, a approval.Action) error {
	a.SubjectKind = cmp.Or(a.SubjectKind, approval.KindActivity)
	return s.actions.Put(ctx, a)
}

func (s Store) ListActions(context.Context) ([]approval.Action, error) {
	return s.actions.List()
}

func (s Store) ActionsOf(_ context.Context, subjectID string) ([]approval.Action, error) {
	return s.actions.Where(func(a approval.Action) bool { return a.SubjectID == subjectID })
}

func (s Store) ActionsOfSubjects(_ context.Context, subjectIDs []string) ([]approval.Action, error) {
	return s.actions.Where(func(a approval.Action) bool { return slices.Contains(subjectIDs, a.SubjectID) })
}

func (s Store) GetNotification(_ context.Context, id string) (approval.Notification, bool, error) {
	return s.notifications.Get(id)
}

// NotificationsOf reverses the order by id before the stable sort by time: the newer id first at the same time.
func (s Store) NotificationsOf(_ context.Context, personID string) ([]approval.Notification, error) {
	mine, err := s.notifications.Where(func(n approval.Notification) bool { return n.RecipientID == personID })
	slices.Reverse(mine)
	slices.SortStableFunc(mine, func(a, b approval.Notification) int { return b.At.Compare(a.At) })
	return mine, err
}

func (s Store) MarkNotificationsRead(ctx context.Context, ids []string, at time.Time) error {
	for _, id := range ids {
		n, found, err := s.notifications.Get(id)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		n.Read, n.ReadAt = true, &at
		if err = s.notifications.Put(ctx, n); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) PutNotification(ctx context.Context, n approval.Notification) error {
	n.SubjectKind = cmp.Or(n.SubjectKind, approval.KindActivity)
	return s.notifications.Put(ctx, n)
}

func (s Store) GetDelegation(_ context.Context, id string) (approval.Delegation, bool, error) {
	return s.delegations.Get(id)
}

func (s Store) ListDelegations(context.Context) ([]approval.Delegation, error) {
	return s.delegations.List()
}

func (s Store) PutDelegation(ctx context.Context, dl approval.Delegation) error {
	return s.delegations.Put(ctx, dl)
}
