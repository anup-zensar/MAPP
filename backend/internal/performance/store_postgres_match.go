package performance

import (
	"context"
	"fmt"
	"strings"

	"mapp/internal/platform/authz"
	"mapp/internal/platform/database/sqlcdb"
)

type scopeArgs struct {
	seesEverything bool
	viewerID       string
	viewsPlans     bool
	teamIDs        []string
}

// argsOf spreads a scope across the five arguments activity_in_scope() takes, so every matcher filters the
// same way and a key outside the scope simply does not resolve.
func argsOf(scope authz.Scope) scopeArgs {
	return scopeArgs{
		seesEverything: scope.Everything, viewerID: scope.PersonID,
		viewsPlans: scope.Holds, teamIDs: scope.TeamIDs,
	}
}

func lowered(keys []string) []string {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, strings.ToLower(strings.TrimSpace(k)))
	}
	return out
}

// grouped keeps every activity a key reached: more than one is ambiguous, and FR-019 forbids guessing.
func grouped[T any](rows []T, pair func(T) (string, string)) map[string][]string {
	found := make(map[string][]string, len(rows))
	for _, r := range rows {
		key, activityID := pair(r)
		found[key] = append(found[key], activityID)
	}
	return found
}

func (s *PostgresStore) MatchByActivityID(ctx context.Context, scope authz.Scope, keys []string) (map[string][]string, error) {
	a := argsOf(scope)
	rows, err := s.db.Queries(ctx).MatchActivitiesByID(ctx, sqlcdb.MatchActivitiesByIDParams{
		Keys: keys, SeesEverything: a.seesEverything, ViewerID: a.viewerID,
		ViewsPlans: a.viewsPlans, ViewerTeamIds: a.teamIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("match %d activity ids: %w", len(keys), err)
	}
	return grouped(rows, func(r sqlcdb.MatchActivitiesByIDRow) (string, string) {
		return r.Key, r.ActivityID
	}), nil
}

func (s *PostgresStore) MatchByMediaCode(ctx context.Context, scope authz.Scope, codes []int64) (map[string][]string, error) {
	a := argsOf(scope)
	rows, err := s.db.Queries(ctx).MatchActivitiesByMediaCode(ctx, sqlcdb.MatchActivitiesByMediaCodeParams{
		Codes: codes, SeesEverything: a.seesEverything, ViewerID: a.viewerID,
		ViewsPlans: a.viewsPlans, ViewerTeamIds: a.teamIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("match %d media codes: %w", len(codes), err)
	}
	return grouped(rows, func(r sqlcdb.MatchActivitiesByMediaCodeRow) (string, string) {
		return r.Key, r.ActivityID
	}), nil
}

func (s *PostgresStore) MatchByIntegrationID(ctx context.Context, scope authz.Scope, sourceID string, keys []string) (map[string][]string, error) {
	a := argsOf(scope)
	rows, err := s.db.Queries(ctx).MatchActivitiesByIntegrationID(ctx, sqlcdb.MatchActivitiesByIntegrationIDParams{
		SourceID: sourceID, Keys: lowered(keys), SeesEverything: a.seesEverything, ViewerID: a.viewerID,
		ViewsPlans: a.viewsPlans, ViewerTeamIds: a.teamIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("match %d integration ids of source %s: %w", len(keys), sourceID, err)
	}
	return grouped(rows, func(r sqlcdb.MatchActivitiesByIntegrationIDRow) (string, string) {
		return strings.ToLower(r.Key), r.ActivityID
	}), nil
}

func (s *PostgresStore) MatchByExternalName(ctx context.Context, scope authz.Scope, keys []string) (map[string][]string, error) {
	a := argsOf(scope)
	rows, err := s.db.Queries(ctx).MatchActivitiesByExternalName(ctx, sqlcdb.MatchActivitiesByExternalNameParams{
		Keys: lowered(keys), SeesEverything: a.seesEverything, ViewerID: a.viewerID,
		ViewsPlans: a.viewsPlans, ViewerTeamIds: a.teamIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("match %d external names: %w", len(keys), err)
	}
	return grouped(rows, func(r sqlcdb.MatchActivitiesByExternalNameRow) (string, string) {
		return r.Key, r.ActivityID
	}), nil
}
