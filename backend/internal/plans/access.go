package plans

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
)

// Access control (feature 014). The access middleware checks every operation that names a plan in its path
// — the person must see the plan (FR-008) and hold the operation's permission in its scope (FR-009) — with
// AccessFacts. The plans service checks what the path does not say: the plans a list shows (FR-007), the
// teams a person may give a plan (FR-013) and the teams of a copy.

// The messages of the Teams field for a person whose plan permission is not for All markets (014 plan
// "User-visible messages" 20, 21 and 22).
const (
	msgOwnTeamsOnly   = "You can only choose your own teams: %s."
	msgKeepOneTeam    = "Keep at least one of your teams (%s) on the plan, or you won't be able to see it."
	msgNotInATeamYet  = "Plans belong to teams, and you are not in a team yet. Ask an administrator to add you to one."
	teamNameSeparator = ", "
)

// AccessFacts returns what an access decision needs to know of the plan p (014 data model §2.2): its teams,
// its owner and who created it.
func AccessFacts(p Plan) authz.PlanFacts {
	return authz.PlanFacts{ID: p.ID, TeamIDs: slices.Clone(p.TeamIDs), OwnerID: p.OwnerID, CreatedByID: p.CreatedBy.ID}
}

// checkTeams adds the problems of the Teams field when a person whose permission p (plans.create or
// plans.edit) is not for All markets saves the plan with the teams after (014 FR-013, Edge cases): a team
// added that is not one of theirs; or no team of theirs left on the plan, which they could then no longer
// see — a plan without teams stays theirs only while they created it. before are the plan's teams so far
// (none for a new plan): teams outside their scope that the plan already has may stay.
func checkTeams(ctx context.Context, p authz.Permission, before []string, after authz.PlanFacts, problems *errs.Problems) {
	g, restricted := authz.From(ctx)
	if !restricted || !g.Has(p) {
		return // an internal call; or a person without p, whom the access middleware refuses
	}
	all, scope := g.Teams(p)
	if all {
		return
	}
	names := g.ScopeTeamNames(p)
	mine := func(id string) bool { return slices.Contains(scope, id) }
	added := slices.ContainsFunc(after.TeamIDs, func(id string) bool { return !slices.Contains(before, id) && !mine(id) })
	keepsOne := slices.ContainsFunc(after.TeamIDs, mine) || (len(after.TeamIDs) == 0 && after.CreatedByID == g.PersonID)
	switch {
	case (added || !keepsOne) && len(names) == 0:
		problems.Add("teamIds", msgNotInATeamYet)
	case added:
		problems.Add("teamIds", fmt.Sprintf(msgOwnTeamsOnly, strings.Join(names, teamNameSeparator)))
	case !keepsOne:
		problems.Add("teamIds", fmt.Sprintf(msgKeepOneTeam, strings.Join(names, teamNameSeparator)))
	}
}
