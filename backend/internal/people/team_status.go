package people

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
)

// SetTeamStatus sets the team id inactive (active false) or reactivates it (FR-014). An inactive team is
// not offered for new plans; its members and the plans that use it keep it, and its members still count for
// R1. A team already in that status is answered unchanged: no new version, no log entry.
func (s *Service) SetTeamStatus(ctx context.Context, a actor.Actor, id string, update StatusUpdate) (TeamView, error) {
	if err := checkStatusUpdate(update); err != nil {
		return TeamView{}, err
	}
	var view TeamView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		team, err := s.storedTeam(ctx, id)
		if err != nil {
			return err
		}
		if err = team.CheckVersion(*update.Version, teamSingular); err != nil {
			return err
		}
		if team.Active != *update.Active {
			if team, err = s.changeTeamStatus(ctx, a, team, *update.Active); err != nil {
				return err
			}
		}
		view, err = s.teamDetails(ctx, team)
		return err
	})
	if err != nil {
		return TeamView{}, err
	}
	return view, nil
}

// changeTeamStatus saves team with the status active, a new version and a "deactivated" or "reactivated"
// log entry, inside the caller's Write, and returns the team as it is stored now.
func (s *Service) changeTeamStatus(ctx context.Context, a actor.Actor, team Team, active bool) (Team, error) {
	changed := team
	changed.Active = active
	changes, err := history.Diff(team.logView(nil), changed.logView(nil))
	if err != nil {
		return Team{}, err
	}
	changed.Touch(a, s.now())
	if err = s.repo.PutTeam(ctx, changed); err != nil {
		return Team{}, err
	}
	_, err = s.history.Log(ctx, a, history.LogInput{
		Kind: teamKind, RecordID: changed.ID, RecordName: changed.Name,
		Action: history.ActionStatusChanged, Verb: statusVerb(active), Changes: changes,
	})
	return changed, err
}
