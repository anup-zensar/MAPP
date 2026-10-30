package approval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/memo"
	"mapp/internal/platform/record"
)

func (s *PostgresStore) GetRound(ctx context.Context, id string) (Round, bool, error) {
	rounds, err := s.rounds(ctx, sqlcdb.ListRoundsParams{ID: &id})
	if err != nil || len(rounds) == 0 {
		return Round{}, false, err
	}
	return rounds[0], true, nil
}

func (s *PostgresStore) ListRounds(ctx context.Context) ([]Round, error) {
	return s.rounds(ctx, sqlcdb.ListRoundsParams{})
}

func (s *PostgresStore) RoundsOf(ctx context.Context, subjectID string) ([]Round, error) {
	return s.rounds(ctx, sqlcdb.ListRoundsParams{SubjectID: &subjectID})
}

func (s *PostgresStore) RoundsOfSubjects(ctx context.Context, subjectIDs []string) ([]Round, error) {
	rows, err := s.db.Queries(ctx).ListRoundsOfSubjects(ctx, subjectIDs)
	if err != nil {
		return nil, fmt.Errorf("select approval rounds of records: %w", err)
	}
	return roundsFromRows(rows)
}

// OpenRounds reads the open rounds once per transaction (memo): a restore asks for those of every activity
// it puts back.
func (s *PostgresStore) OpenRounds(ctx context.Context) ([]Round, error) {
	return memo.Get(ctx, keptArea+":openRounds", func(ctx context.Context) ([]Round, error) {
		rows, err := s.db.Queries(ctx).ListOpenRounds(ctx)
		if err != nil {
			return nil, fmt.Errorf("select open approval rounds: %w", err)
		}
		return roundsFromRows(rows)
	})
}

// RoundsInScope keeps the rounds of the activities the person acting may see; a round of an activity in the
// recycle bin is in no scope.
func (s *PostgresStore) RoundsInScope(ctx context.Context, scope authz.Scope) ([]Round, error) {
	rows, err := s.db.Queries(ctx).ListRoundsInScope(ctx, sqlcdb.ListRoundsInScopeParams{
		SeesEverything: scope.Everything, ViewerID: scope.PersonID, ViewsPlans: scope.Holds,
		ViewerTeamIds: scope.TeamIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("select approval rounds in scope: %w", err)
	}
	return roundsFromRows(rows)
}

// PutRound saves r with feature 017's defaults in its levels.
func (s *PostgresStore) PutRound(ctx context.Context, r Round) error {
	normalised := Round{Levels: slices.Clone(r.Levels)}.normalised().Levels
	levels, levelsErr := json.Marshal(normalised)
	facts, factsErr := json.Marshal(r.Facts)
	if err := errors.Join(levelsErr, factsErr); err != nil {
		return fmt.Errorf("encode approval round %s: %w", r.ID, err)
	}
	err := s.change(ctx).PutRound(ctx, sqlcdb.PutRoundParams{
		ID: r.ID, SubjectKind: string(r.kind()), SubjectID: r.SubjectID, Number: r.Number, WorkflowID: r.WorkflowID,
		WorkflowName: r.WorkflowName,
		Levels:       levels, CurrentLevel: r.CurrentLevel, Outcome: string(r.Outcome), SubmittedAt: r.SubmittedAt,
		SubmittedByID: r.SubmittedBy.ID, SubmittedByName: r.SubmittedBy.Name, Comment: r.Comment, EndedAt: r.EndedAt,
		Facts: facts, ReminderDays: r.ReminderDays, EscalationDays: r.EscalationDays, AutoApproved: r.AutoApproved,
		Version: r.Version, CreatedAt: r.CreatedAt, CreatedByID: r.CreatedBy.ID, CreatedByName: r.CreatedBy.Name,
		UpdatedAt: r.UpdatedAt, UpdatedByID: r.UpdatedBy.ID, UpdatedByName: r.UpdatedBy.Name,
	})
	if err != nil {
		return fmt.Errorf("save approval round %s: %w", r.ID, err)
	}
	return nil
}

func (s *PostgresStore) PutAction(ctx context.Context, a Action) error {
	approvers, approversErr := json.Marshal(a.Approvers)
	onBehalfOf, onBehalfErr := json.Marshal(a.OnBehalfOf)
	changes, changesErr := json.Marshal(a.Changes)
	if err := errors.Join(approversErr, onBehalfErr, changesErr); err != nil {
		return fmt.Errorf("encode approval history entry %s: %w", a.ID, err)
	}
	params := sqlcdb.InsertActionParams{
		ID: a.ID, RoundID: a.RoundID, RoundNumber: a.RoundNumber, SubjectKind: string(a.subjectKind()), SubjectID: a.SubjectID,
		Level: a.Level, LevelName: a.LevelName, Approvers: approvers, ActorID: a.Actor.ID, ActorName: a.Actor.Name,
		OnBehalfOf: onBehalfOf, At: a.At, Comment: a.Comment, Changes: changes, Capacity: string(a.Capacity),
		Action: string(a.Action), Rule: a.Rule, WaitedDays: a.WaitedDays, SelfApproval: a.SelfApproval,
	}
	params.AsDelegateOfID, params.AsDelegateOfName = actorColumns(a.AsDelegateOf)
	params.EscalatedToID, params.EscalatedToName = actorColumns(a.EscalatedTo)
	if err := s.change(ctx).InsertAction(ctx, params); err != nil {
		return fmt.Errorf("save approval history entry %s: %w", a.ID, err)
	}
	return nil
}

func (s *PostgresStore) ListActions(ctx context.Context) ([]Action, error) {
	rows, err := s.db.Queries(ctx).ListActions(ctx, nil)
	return actionsFromRows(rows, err)
}

func (s *PostgresStore) ActionsOf(ctx context.Context, subjectID string) ([]Action, error) {
	rows, err := s.db.Queries(ctx).ListActions(ctx, &subjectID)
	return actionsFromRows(rows, err)
}

func (s *PostgresStore) ActionsOfSubjects(ctx context.Context, subjectIDs []string) ([]Action, error) {
	rows, err := s.db.Queries(ctx).ListActionsOfSubjects(ctx, subjectIDs)
	return actionsFromRows(rows, err)
}

func (s *PostgresStore) ActionsInScope(ctx context.Context, scope authz.Scope) ([]Action, error) {
	rows, err := s.db.Queries(ctx).ListActionsInScope(ctx, sqlcdb.ListActionsInScopeParams{
		SeesEverything: scope.Everything, ViewerID: scope.PersonID, ViewsPlans: scope.Holds,
		ViewerTeamIds: scope.TeamIDs,
	})
	return actionsFromRows(rows, err)
}

func (s *PostgresStore) rounds(ctx context.Context, filter sqlcdb.ListRoundsParams) ([]Round, error) {
	rows, err := s.db.Queries(ctx).ListRounds(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("select approval rounds: %w", err)
	}
	return roundsFromRows(rows)
}

func roundsFromRows(rows []sqlcdb.ApprovalRound) ([]Round, error) {
	rounds := make([]Round, 0, len(rows))
	for _, row := range rows {
		r := Round{
			ID: row.ID, SubjectKind: SubjectKind(row.SubjectKind), SubjectID: row.SubjectID, Number: row.Number,
			WorkflowID: row.WorkflowID, WorkflowName: row.WorkflowName, CurrentLevel: row.CurrentLevel, Outcome: Outcome(row.Outcome),
			SubmittedAt: row.SubmittedAt, SubmittedBy: actor.Actor{ID: row.SubmittedByID, Name: row.SubmittedByName},
			Comment: row.Comment, EndedAt: row.EndedAt, ReminderDays: row.ReminderDays,
			EscalationDays: row.EscalationDays, AutoApproved: row.AutoApproved,
			Meta: record.Meta{
				Version:   row.Version,
				CreatedAt: row.CreatedAt,
				CreatedBy: actor.Actor{ID: row.CreatedByID, Name: row.CreatedByName},
				UpdatedAt: row.UpdatedAt,
				UpdatedBy: actor.Actor{ID: row.UpdatedByID, Name: row.UpdatedByName},
			},
		}
		if err := errors.Join(json.Unmarshal(row.Levels, &r.Levels), json.Unmarshal(row.Facts, &r.Facts)); err != nil {
			return nil, fmt.Errorf("decode approval round %s: %w", row.ID, err)
		}
		rounds = append(rounds, r.normalised())
	}
	return rounds, nil
}

func actionsFromRows(rows []sqlcdb.ApprovalAction, err error) ([]Action, error) {
	if err != nil {
		return nil, fmt.Errorf("select approval history entries: %w", err)
	}
	actions := make([]Action, 0, len(rows))
	for _, row := range rows {
		a := Action{
			ID: row.ID, RoundID: row.RoundID, RoundNumber: row.RoundNumber, SubjectKind: SubjectKind(row.SubjectKind),
			SubjectID: row.SubjectID, Action: ActionKind(row.Action), Level: row.Level, LevelName: row.LevelName,
			Actor: actor.Actor{ID: row.ActorID, Name: row.ActorName}, At: row.At, Comment: row.Comment,
			Capacity: Capacity(row.Capacity), AsDelegateOf: actorOf(row.AsDelegateOfID, row.AsDelegateOfName),
			Rule: row.Rule, EscalatedTo: actorOf(row.EscalatedToID, row.EscalatedToName), WaitedDays: row.WaitedDays,
			SelfApproval: row.SelfApproval,
		}
		err = errors.Join(json.Unmarshal(row.Approvers, &a.Approvers), json.Unmarshal(row.OnBehalfOf, &a.OnBehalfOf),
			json.Unmarshal(row.Changes, &a.Changes))
		if err != nil {
			return nil, fmt.Errorf("decode approval history entry %s: %w", row.ID, err)
		}
		actions = append(actions, a)
	}
	return actions, nil
}

func actorColumns(a *actor.Actor) (id, name *string) {
	if a == nil {
		return nil, nil
	}
	return &a.ID, &a.Name
}

func actorOf(id, name *string) *actor.Actor {
	if id == nil || name == nil {
		return nil
	}
	return &actor.Actor{ID: *id, Name: *name}
}
