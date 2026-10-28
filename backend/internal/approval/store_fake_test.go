package approval

import (
	"cmp"
	"context"
	"slices"
	"time"

	"mapp/internal/platform/fake"
)

// fakeStore is approvaltest.Store for this package's own tests, which cannot import approvaltest.
type fakeStore struct {
	workflows     *fake.Table[Workflow]
	rounds        *fake.Table[Round]
	actions       *fake.Table[Action]
	notifications *fake.Table[Notification]
	delegations   *fake.Table[Delegation]
}

func newFakeStore(db *fake.DB) fakeStore {
	return fakeStore{
		workflows:     fake.TableOf(db, "approval_workflows", func(w Workflow) string { return w.ID }),
		rounds:        fake.TableOf(db, "approval_rounds", func(r Round) string { return r.ID }),
		actions:       fake.TableOf(db, "approval_actions", func(a Action) string { return a.ID }),
		notifications: fake.TableOf(db, "notifications", func(n Notification) string { return n.ID }),
		delegations:   fake.TableOf(db, "approval_delegations", func(dl Delegation) string { return dl.ID }),
	}
}

func (s fakeStore) GetWorkflow(_ context.Context, id string) (Workflow, bool, error) {
	return s.workflows.Get(id)
}

func (s fakeStore) ListWorkflows(context.Context) ([]Workflow, error) {
	return s.workflows.List()
}

func (s fakeStore) PutWorkflow(ctx context.Context, w Workflow) error {
	w.Levels = normalisedLevels(slices.Clone(w.Levels))
	if w.Levels == nil {
		w.Levels = []Level{}
	}
	w.ObjectType = w.Kind()
	return s.workflows.Put(ctx, w)
}

func (s fakeStore) DeleteWorkflow(ctx context.Context, id string) error {
	return s.workflows.Delete(ctx, id)
}

func (s fakeStore) GetRound(_ context.Context, id string) (Round, bool, error) {
	return s.rounds.Get(id)
}

func (s fakeStore) ListRounds(context.Context) ([]Round, error) {
	return s.rounds.List()
}

func (s fakeStore) RoundsOf(_ context.Context, subjectID string) ([]Round, error) {
	rounds, err := s.rounds.Where(func(r Round) bool { return r.SubjectID == subjectID })
	slices.SortStableFunc(rounds, func(a, b Round) int { return cmp.Compare(a.Number, b.Number) })
	return rounds, err
}

func (s fakeStore) OpenRounds(context.Context) ([]Round, error) {
	return s.rounds.Where(func(r Round) bool { return r.Outcome == OutcomeOpen })
}

func (s fakeStore) PutRound(ctx context.Context, r Round) error {
	r.Levels, r.SubjectKind = Round{Levels: slices.Clone(r.Levels)}.normalised().Levels, r.kind()
	return s.rounds.Put(ctx, r)
}

func (s fakeStore) PutAction(ctx context.Context, a Action) error {
	a.SubjectKind = a.subjectKind()
	return s.actions.Put(ctx, a)
}

func (s fakeStore) ListActions(context.Context) ([]Action, error) {
	return s.actions.List()
}

func (s fakeStore) ActionsOf(_ context.Context, subjectID string) ([]Action, error) {
	return s.actions.Where(func(a Action) bool { return a.SubjectID == subjectID })
}

func (s fakeStore) GetNotification(_ context.Context, id string) (Notification, bool, error) {
	return s.notifications.Get(id)
}

func (s fakeStore) NotificationsOf(_ context.Context, personID string) ([]Notification, error) {
	mine, err := s.notifications.Where(func(n Notification) bool { return n.RecipientID == personID })
	slices.Reverse(mine)
	slices.SortStableFunc(mine, func(a, b Notification) int { return b.At.Compare(a.At) })
	return mine, err
}

func (s fakeStore) MarkNotificationsRead(ctx context.Context, ids []string, at time.Time) error {
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

func (s fakeStore) PutNotification(ctx context.Context, n Notification) error {
	n.SubjectKind = n.subjectKind()
	return s.notifications.Put(ctx, n)
}

func (s fakeStore) GetDelegation(_ context.Context, id string) (Delegation, bool, error) {
	return s.delegations.Get(id)
}

func (s fakeStore) ListDelegations(context.Context) ([]Delegation, error) {
	return s.delegations.List()
}

func (s fakeStore) PutDelegation(ctx context.Context, dl Delegation) error {
	return s.delegations.Put(ctx, dl)
}
