package people

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/record"
)

// The longest team name and description, in characters (003 data model §2).
const (
	maxTeamNameLength    = 80
	maxDescriptionLength = 300
)

// ListTeams returns every team, active or not, sorted by name ignoring letter case, each with its members
// (active or not, by name) and the plans that use it, including plans in the recycle bin (FR-010, FR-014).
// It is also the data of the team picker. Every signed-in person may see it (FR-027), so a is not used yet.
func (s *Service) ListTeams(ctx context.Context, a actor.Actor) ([]TeamView, error) {
	var views []TeamView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		d, err := s.readDirectory(ctx)
		if err != nil {
			return err
		}
		plans, err := s.allPlans(ctx)
		if err != nil {
			return err
		}
		views = make([]TeamView, 0, len(d.teams))
		for _, t := range d.sortedTeams() {
			views = append(views, d.teamView(t, plans))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return views, nil
}

// GetTeam returns the team id with its members and the plans that use it (US2 scenario 3).
func (s *Service) GetTeam(ctx context.Context, a actor.Actor, id string) (TeamView, error) {
	var view TeamView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		t, err := s.storedTeam(ctx, id)
		if err != nil {
			return err
		}
		view, err = s.teamDetails(ctx, t)
		return err
	})
	if err != nil {
		return TeamView{}, err
	}
	return view, nil
}

// CreateTeam creates an active team with its members (FR-010, FR-011). Every problem of the form is
// reported at once. Without a colour, the team gets the palette colour the fewest teams have (D-065). The
// log gets the team's "created" entry and a "changed" entry for every member (FR-012, FR-028).
func (s *Service) CreateTeam(ctx context.Context, a actor.Actor, input TeamInput) (TeamView, error) {
	var view TeamView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		d, err := s.readDirectory(ctx)
		if err != nil {
			return err
		}
		var problems errs.Problems
		form := checkTeamForm(&problems, d, "", teamForm{input.Name, input.Description, input.Colour, input.MemberIDs}, false)
		if err := problems.Err(); err != nil {
			return err
		}
		if form.Colour == "" {
			form.Colour = leastUsedColour(d.sortedTeams())
		}

		id, err := s.ids.Next(ctx, ids.PrefixTeam)
		if err != nil {
			return err
		}
		team := Team{ID: id, Name: form.Name, Description: form.Description, Colour: form.Colour, Active: true,
			Meta: record.NewMeta(a, s.now())}
		if err := s.repo.PutTeam(ctx, team); err != nil {
			return err
		}
		_, err = s.history.Log(ctx, a, history.LogInput{
			Kind: teamKind, RecordID: team.ID, RecordName: team.Name, Action: history.ActionCreated,
		})
		if err != nil {
			return err
		}
		add, _ := membershipDelta(nil, form.MemberIDs, func(personID string) pair { return pair{team.ID, personID} })
		if err := s.applyMemberships(ctx, a, team.ID, add, nil); err != nil {
			return err
		}
		view, err = s.teamDetails(ctx, team)
		return err
	})
	if err != nil {
		return TeamView{}, err
	}
	return view, nil
}

// UpdateTeam changes the team id with the fields update sends — name, description, colour, every member
// after the change — and keeps the others (FR-011, FR-012, D-210). Every problem of the form the change
// leads to, a missing version included, is reported at once; an out-of-date version is refused with
// version_conflict (001 FR-029). Removing a member who owns a plan is allowed: the impact check warns
// first (FR-013). A form that changes nothing is not a change: no new version, no log entry. Otherwise the
// team gets a new version and one "changed" entry, and every person who joined or left gets one too
// (D-057). A rename shows on every plan at once, because plans store the team's id (US2 scenario 4).
func (s *Service) UpdateTeam(ctx context.Context, a actor.Actor, id string, update TeamUpdate) (TeamView, error) {
	var view TeamView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		team, err := s.storedTeam(ctx, id)
		if err != nil {
			return err
		}
		d, err := s.readDirectory(ctx)
		if err != nil {
			return err
		}
		var problems errs.Problems
		form := checkTeamForm(&problems, d, team.ID, update.applyTo(storedTeamForm(team, d)), true)
		checkVersionSent(&problems, update.Version)
		if err := problems.Err(); err != nil {
			return err
		}
		if err := team.CheckVersion(*update.Version, teamSingular); err != nil {
			return err
		}

		before := d.memberIDsOf(team.ID)
		changed := team
		changed.Name, changed.Description, changed.Colour = form.Name, form.Description, form.Colour
		changes, err := history.Diff(team.logView(before), changed.logView(form.MemberIDs))
		if err != nil {
			return err
		}
		if len(changes) > 0 {
			changed.Touch(a, s.now())
			if err := s.repo.PutTeam(ctx, changed); err != nil {
				return err
			}
			_, err = s.history.Log(ctx, a, history.LogInput{
				Kind: teamKind, RecordID: changed.ID, RecordName: changed.Name, Action: history.ActionChanged, Changes: changes,
			})
			if err != nil {
				return err
			}
			add, remove := membershipDelta(before, form.MemberIDs, func(personID string) pair { return pair{team.ID, personID} })
			if err := s.applyMemberships(ctx, a, team.ID, add, remove); err != nil {
				return err
			}
		}
		view, err = s.teamDetails(ctx, changed)
		return err
	})
	if err != nil {
		return TeamView{}, err
	}
	return view, nil
}

// TeamImpact says what saving update would affect, before it is saved, and saves nothing (FR-013, D-058):
// one ownerNeedsAttention warning for each removed member who owns plans that use this team and would then
// belong to none of their teams; members that are not sent are not removed. The form is checked exactly
// like UpdateTeam's, except for the version.
func (s *Service) TeamImpact(ctx context.Context, a actor.Actor, id string, update TeamUpdate) (ChangeImpact, error) {
	impact := ChangeImpact{Warnings: []ImpactWarning{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		team, err := s.storedTeam(ctx, id)
		if err != nil {
			return err
		}
		d, err := s.readDirectory(ctx)
		if err != nil {
			return err
		}
		var problems errs.Problems
		form := checkTeamForm(&problems, d, team.ID, update.applyTo(storedTeamForm(team, d)), true)
		if err := problems.Err(); err != nil {
			return err
		}

		removed, after := map[string][]string{}, map[string][]string{}
		for _, personID := range d.memberIDsOf(team.ID) {
			if !slices.Contains(form.MemberIDs, personID) {
				removed[personID] = []string{team.ID}
				after[personID] = without(d.teamIDsOf(personID), team.ID)
			}
		}
		impact.Warnings, err = s.ownerImpact(ctx, d, removed, after, usesThisTeam, useThisTeam)
		return err
	})
	if err != nil {
		return ChangeImpact{}, err
	}
	return impact, nil
}

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
		if err := team.CheckVersion(*update.Version, teamSingular); err != nil {
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
	if err := s.repo.PutTeam(ctx, changed); err != nil {
		return Team{}, err
	}
	_, err = s.history.Log(ctx, a, history.LogInput{
		Kind: teamKind, RecordID: changed.ID, RecordName: changed.Name,
		Action: history.ActionStatusChanged, Verb: statusVerb(active), Changes: changes,
	})
	return changed, err
}

// DeleteTeam moves the team id, together with its memberships, to the recycle bin (FR-015), and returns
// the recycle-bin entry so that the person can undo the delete. A team that any plan uses, including plans
// in the recycle bin, is refused with in_use: it can be set inactive instead (FR-014).
func (s *Service) DeleteTeam(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		team, err := s.storedTeam(ctx, id)
		if err != nil {
			return err
		}
		plans, err := s.allPlans(ctx)
		if err != nil {
			return err
		}
		switch uses := len(plansUsing(plans, team.ID)); {
		case uses == 1:
			return errs.InUse(fmt.Sprintf(teamInUseOneMessage, team.Name))
		case uses > 1:
			return errs.InUse(fmt.Sprintf(teamInUseManyMessage, team.Name, uses))
		}
		entry, err = s.history.MoveToBin(ctx, a, teamKind, team.ID)
		return err
	})
	if err != nil {
		return history.BinEntry{}, err
	}
	return entry, nil
}

// storedTeam returns the stored team id, or not_found with the message the web app shows.
func (s *Service) storedTeam(ctx context.Context, id string) (Team, error) {
	t, found, err := s.repo.GetTeam(ctx, id)
	if err != nil {
		return Team{}, err
	}
	if !found {
		return Team{}, errs.NotFound(teamGoneMessage)
	}
	return t, nil
}

// teamDetails returns t as the team operations answer it, inside the caller's transaction: with its
// members as stored now and the plans that use it.
func (s *Service) teamDetails(ctx context.Context, t Team) (TeamView, error) {
	d, err := s.readDirectory(ctx)
	if err != nil {
		return TeamView{}, err
	}
	plans, err := s.allPlans(ctx)
	if err != nil {
		return TeamView{}, err
	}
	return d.teamView(t, plans), nil
}

// teamForm is what the team form sends: the fields of TeamInput and TeamUpdate without the version.
type teamForm struct {
	Name, Description, Colour string
	MemberIDs                 []string
}

// storedTeamForm is the team t as its form starts: the stored name, description, colour and members.
// TeamUpdate.applyTo puts the sent fields in place.
func storedTeamForm(t Team, d directory) teamForm {
	return teamForm{Name: t.Name, Description: t.Description, Colour: t.Colour, MemberIDs: d.memberIDsOf(t.ID)}
}

// checkTeamForm returns form with its texts trimmed and its member ids without repeats, and adds every
// problem to problems (FR-011, 003 data model §2): the name is required, has at most 80 characters and no
// other team has it, ignoring letter case; the description has at most 300 characters; the colour is one of
// the palette — or empty, when colourRequired is false (create picks one); every member is a person, and a
// person who is not a member yet must be active (D-070). teamID is the team being edited, "" on create: it
// may keep its name and its inactive members.
func checkTeamForm(problems *errs.Problems, d directory, teamID string, form teamForm, colourRequired bool) teamForm {
	form.Name = strings.TrimSpace(form.Name)
	form.Description = strings.TrimSpace(form.Description)
	form.MemberIDs = uniqueIDs(form.MemberIDs)

	other, taken := teamNamed(d.sortedTeams(), form.Name, teamID)
	switch {
	case form.Name == "":
		problems.Add("name", enterNameMessage)
	case utf8.RuneCountInString(form.Name) > maxTeamNameLength:
		problems.Add("name", use80Message)
	case taken:
		problems.Add("name", fmt.Sprintf(teamNameTakenMessage, other.Name))
	}
	checkLength(problems, "description", form.Description, maxDescriptionLength, use300Message)
	if form.Colour != "" || colourRequired {
		if !slices.ContainsFunc(Palette, func(c Colour) bool { return c.Key == form.Colour }) {
			problems.Add("colour", chooseColourMessage)
		}
	}

	members := d.memberIDsOf(teamID)
	if slices.ContainsFunc(form.MemberIDs, func(id string) bool { _, known := d.people[id]; return !known }) {
		problems.Add("memberIds", unknownPersonMessage)
	}
	for _, id := range form.MemberIDs {
		if p, known := d.people[id]; known && !p.Active && !slices.Contains(members, id) {
			problems.Add("memberIds", fmt.Sprintf(inactivePersonMessage, p.Name))
		}
	}
	return form
}

// leastUsedColour returns the palette colour the fewest of teams have, the first in palette order when
// several have as few (D-025, D-065).
func leastUsedColour(teams []Team) string {
	best, fewest := Palette[0].Key, len(teams)+1
	for _, c := range Palette {
		used := 0
		for _, t := range teams {
			if t.Colour == c.Key {
				used++
			}
		}
		if used < fewest {
			best, fewest = c.Key, used
		}
	}
	return best
}
