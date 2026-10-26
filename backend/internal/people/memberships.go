package people

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

// pair is one person in one team: a membership without its id and bookkeeping.
type pair struct {
	teamID, personID string
}

// applyMemberships changes the stored memberships inside the caller's Write (FR-012): it creates a
// membership for each pair of add that does not exist yet — never a second one for the same pair — and
// deletes the membership of each pair of remove. A membership change is a change of both records (D-057):
// every other team and person whose memberships changed gets a new version and one "changed" log entry
// ("Members" on a team, "Teams" on a person), so that an edit form opened before the change is refused with
// version_conflict instead of silently undoing it. The record being saved, saving, is not touched here: its
// operation saves and logs it once, together with its other changes, before calling applyMemberships.
func (s *Service) applyMemberships(ctx context.Context, a actor.Actor, saving string, add, remove []pair) error {
	before, err := s.repo.ListMemberships(ctx)
	if err != nil {
		return err
	}
	stored := map[pair]Membership{}
	for _, m := range before {
		stored[pair{m.TeamID, m.PersonID}] = m
	}

	var changed []pair
	for _, p := range remove {
		m, found := stored[p]
		if !found {
			continue
		}
		if err := s.repo.DeleteMembership(ctx, m.ID); err != nil {
			return err
		}
		delete(stored, p)
		changed = append(changed, p)
	}
	for _, p := range add {
		if _, found := stored[p]; found {
			continue
		}
		id, err := s.ids.Next(ctx, ids.PrefixTeamMembership)
		if err != nil {
			return err
		}
		m := Membership{ID: id, TeamID: p.teamID, PersonID: p.personID, Meta: record.NewMeta(a, s.now())}
		if err := s.repo.PutMembership(ctx, m); err != nil {
			return err
		}
		stored[p] = m
		changed = append(changed, p)
	}

	// The other records whose memberships changed, each once, in id order: teams, then people.
	was, is := directory{memberships: before}, directory{memberships: slices.Collect(maps.Values(stored))}
	var teamIDs, personIDs []string
	for _, p := range changed {
		if p.teamID != saving && !slices.Contains(teamIDs, p.teamID) {
			teamIDs = append(teamIDs, p.teamID)
		}
		if p.personID != saving && !slices.Contains(personIDs, p.personID) {
			personIDs = append(personIDs, p.personID)
		}
	}
	slices.Sort(teamIDs)
	slices.Sort(personIDs)
	for _, id := range teamIDs {
		if err := s.logMembersChange(ctx, a, id, was.memberIDsOf(id), is.memberIDsOf(id)); err != nil {
			return err
		}
	}
	for _, id := range personIDs {
		if err := s.logTeamsChange(ctx, a, id, was.teamIDsOf(id), is.teamIDsOf(id)); err != nil {
			return err
		}
	}
	return nil
}

// logMembersChange gives the team teamID a new version and logs the change of its members from before to
// after, inside the caller's Write.
func (s *Service) logMembersChange(ctx context.Context, a actor.Actor, teamID string, before, after []string) error {
	team, found, err := s.repo.GetTeam(ctx, teamID)
	if err != nil || !found {
		return err
	}
	changes, err := history.Diff(team.logView(before), team.logView(after))
	if err != nil {
		return err
	}
	team.Touch(a, s.now())
	if err := s.repo.PutTeam(ctx, team); err != nil {
		return err
	}
	_, err = s.history.Log(ctx, a, history.LogInput{
		Kind: teamKind, RecordID: team.ID, RecordName: team.Name, Action: history.ActionChanged, Changes: changes,
	})
	return err
}

// logTeamsChange gives the person personID a new version and logs the change of their teams from before to
// after, inside the caller's Write.
func (s *Service) logTeamsChange(ctx context.Context, a actor.Actor, personID string, before, after []string) error {
	p, found, err := s.repo.Get(ctx, personID)
	if err != nil || !found {
		return err
	}
	changes, err := history.Diff(p.logView(before), p.logView(after))
	if err != nil {
		return err
	}
	p.Touch(a, s.now())
	if err := s.repo.Put(ctx, p); err != nil {
		return err
	}
	_, err = s.history.Log(ctx, a, history.LogInput{
		Kind: personKind, RecordID: p.ID, RecordName: p.Name, Action: history.ActionChanged, Changes: changes,
	})
	return err
}

// membershipDelta returns the pairs to add and remove when the record on one side of the memberships goes
// from the ids before to the ids after: pairOf turns one id of the other side into a pair.
func membershipDelta(before, after []string, pairOf func(id string) pair) (add, remove []pair) {
	for _, id := range after {
		if !slices.Contains(before, id) {
			add = append(add, pairOf(id))
		}
	}
	for _, id := range before {
		if !slices.Contains(after, id) {
			remove = append(remove, pairOf(id))
		}
	}
	return add, remove
}

// ownerImpact returns the owner warnings of a membership change, inside the caller's transaction (FR-013,
// rule R1, D-058). removed holds, for each person who leaves teams, the teams they leave; after holds every
// team they belong to after the change (inactive teams count, FR-014). A stored plan — not one in the
// recycle bin — needs attention when its owner leaves one of its teams and then belongs to none of them.
// There is one warning per affected owner, owners by name, their plans by name. usesOne and usesMany say
// which teams the plans use ("uses this team" / "use this team"). Without plans there are no warnings.
func (s *Service) ownerImpact(ctx context.Context, d directory, removed, after map[string][]string, usesOne, usesMany string) ([]ImpactWarning, error) {
	plans, err := s.allPlans(ctx)
	if err != nil {
		return nil, err
	}
	affected := map[string][]PlanUse{} // owner id → plans that will need attention
	for _, p := range plans {
		left, leaving := removed[p.OwnerID]
		if p.InRecycleBin || !leaving || !sharesAny(p.TeamIDs, left) || sharesAny(p.TeamIDs, after[p.OwnerID]) {
			continue
		}
		affected[p.OwnerID] = append(affected[p.OwnerID], p)
	}

	owners := make([]Person, 0, len(affected))
	for id := range affected {
		owner, found := d.people[id]
		if !found {
			owner = Person{ID: id, Name: id}
		}
		owners = append(owners, owner)
	}
	slices.SortFunc(owners, comparePeople)

	warnings := []ImpactWarning{}
	for _, owner := range owners {
		plans := affected[owner.ID]
		names := make([]string, 0, len(plans))
		for _, p := range plans {
			names = append(names, p.Name)
		}
		slices.SortFunc(names, compareNames)
		list := strings.Join(names, ", ")

		message := fmt.Sprintf(ownerWarningOneMessage, owner.Name, usesOne, list, owner.FirstName())
		if len(plans) > 1 {
			message = fmt.Sprintf(ownerWarningManyMessage, owner.Name, len(plans), usesMany, list, owner.FirstName())
		}
		warnings = append(warnings, ImpactWarning{Kind: ImpactOwnerNeedsAttention, Message: message, Count: len(plans)})
	}
	return warnings, nil
}

// sharesAny reports whether the two lists of ids have an id in common.
func sharesAny(a, b []string) bool {
	return slices.ContainsFunc(a, func(id string) bool { return slices.Contains(b, id) })
}

// without returns ids without the id left, as a new list.
func without(ids []string, left string) []string {
	return slices.DeleteFunc(slices.Clone(ids), func(id string) bool { return id == left })
}
