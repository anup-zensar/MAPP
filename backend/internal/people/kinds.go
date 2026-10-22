package people

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
)

// The history kinds of this package (003 data model §6).
const (
	personKind     = "person"
	teamKind       = "team"
	membershipKind = "teamMembership"
)

// The mid-sentence names of people and teams, as the log, the recycle bin and the version-conflict message
// ("Not saved — Daniel Okoro changed this team") show them.
const (
	personSingular = "person"
	teamSingular   = "team"
)

// errNeverDeleted is what the person kind's Remove answers: people are never deleted (FR-008). The kind
// needs a Remove because history.Register requires one; no operation ever moves a person to the bin.
var errNeverDeleted = errors.New("people: people are never deleted (FR-008)")

// Kinds returns the history kinds of people, teams and team memberships, parents first, for internal/app to
// register before the API serves requests (FR-028). With them, history logs the changes of people and teams
// in words, deletes a team together with its memberships and brings both back (FR-015).
func (s *Service) Kinds() []history.Kind {
	return []history.Kind{
		{
			Name:     personKind,
			Singular: personSingular,
			Plural:   "people",
			Snapshot: s.personSnapshot,
			Remove:   func(context.Context, string) error { return errNeverDeleted },
			Put:      s.putPerson,
			Describe: s.describePerson,
		},
		{
			Name:           teamKind,
			Singular:       teamSingular,
			Plural:         "teams",
			Snapshot:       s.teamSnapshot,
			Remove:         s.repo.DeleteTeam,
			Put:            s.putTeam,
			Describe:       s.describeTeam,
			PrepareRestore: s.prepareTeamRestore,
		},
		{
			Name:     membershipKind,
			Singular: "team membership",
			Plural:   "team memberships",
			Parent:   teamKind,
			Snapshot: s.membershipSnapshot,
			ChildIDs: s.membershipIDs,
			Remove:   s.repo.DeleteMembership,
			Put:      s.putMembership,
		},
	}
}

// personLog is a person as the activity log compares them (003 data model §6): what people read, without
// the bookkeeping. TeamIDs is ordered by id.
type personLog struct {
	Name      string   `json:"name"`
	Email     string   `json:"email"`
	JobTitle  string   `json:"jobTitle"`
	Active    bool     `json:"active"`
	TeamIDs   []string `json:"teamIds"`
	ManagerID string   `json:"managerId"` // 017 FR-016, FR-035: line-manager changes are logged
}

// logView returns p with the teams teamIDs as the activity log compares them.
func (p Person) logView(teamIDs []string) personLog {
	sorted := slices.Sorted(slices.Values(teamIDs))
	return personLog{Name: p.Name, Email: p.Email, JobTitle: p.JobTitle, Active: p.Active, TeamIDs: sorted, ManagerID: p.ManagerID}
}

// teamLog is a team as the activity log compares it (003 data model §6). MemberIDs is ordered by id.
type teamLog struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Colour      string   `json:"colour"`
	Active      bool     `json:"active"`
	MemberIDs   []string `json:"memberIds"`
}

// logView returns t with the members memberIDs as the activity log compares it.
func (t Team) logView(memberIDs []string) teamLog {
	sorted := slices.Sorted(slices.Values(memberIDs))
	return teamLog{Name: t.Name, Description: t.Description, Colour: t.Colour, Active: t.Active, MemberIDs: sorted}
}

// describePerson gives the activity log's label and text of a person's field (003 data model §6): the
// teams by name, the status as Active or Inactive. Other fields keep history's defaults.
func (s *Service) describePerson(ctx context.Context, field string, value any) (string, string, error) {
	switch field {
	case "email":
		return emailLabel, "", nil
	case "jobTitle":
		return jobTitleLabel, "", nil
	case "active":
		return statusLabel, statusText(value), nil
	case "teamIds":
		names, err := s.namesOf(ctx, value, s.teamName)
		if names == "" {
			names = noTeamText
		}
		return teamsLabel, names, err
	case "managerId":
		id, _ := value.(string)
		if id == "" {
			return managerLabel, "", nil // no manager: history's "Not filled in"
		}
		name, err := s.personName(ctx, id)
		return managerLabel, name, err
	case "roles": // logged by access control (014 data model §3): the texts are written when the change is made
		return rolesLabel, "", nil
	}
	return "", "", nil
}

// describeTeam gives the activity log's label and text of a team's field: the members by name, the colour
// by its label, the status as Active or Inactive.
func (s *Service) describeTeam(ctx context.Context, field string, value any) (string, string, error) {
	switch field {
	case "colour":
		key, _ := value.(string)
		return colourLabel, colourLabelOf(key), nil
	case "active":
		return statusLabel, statusText(value), nil
	case "memberIds":
		names, err := s.namesOf(ctx, value, s.personName)
		if names == "" {
			names = noMembersText
		}
		return membersLabel, names, err
	}
	return "", "", nil
}

// statusText writes a stored "active" value as people read it; "" (history's default) for anything else.
func statusText(value any) string {
	active, isBool := value.(bool)
	switch {
	case !isBool:
		return ""
	case active:
		return activeText
	default:
		return inactiveText
	}
}

// colourLabelOf returns the label of a palette colour ("Blue" for "blue"), or the key when it is not one.
func colourLabelOf(key string) string {
	for _, c := range Palette {
		if c.Key == key {
			return c.Label
		}
	}
	return key
}

// namesOf turns a stored list of ids (value, decoded: []any of strings) into the names nameOf gives, sorted
// ignoring letter case and joined with ", "; "" for an empty list. The names are those at the time of the
// change, because the log fixes its texts when the entry is written.
func (s *Service) namesOf(ctx context.Context, value any, nameOf func(context.Context, string) (string, error)) (string, error) {
	list, _ := value.([]any)
	names := make([]string, 0, len(list))
	for _, item := range list {
		id, _ := item.(string)
		name, err := nameOf(ctx, id)
		if err != nil {
			return "", err
		}
		names = append(names, name)
	}
	slices.SortFunc(names, compareNames)
	return strings.Join(names, ", "), nil
}

// teamName returns the name of the team id, or the id itself when the team is no longer stored.
func (s *Service) teamName(ctx context.Context, id string) (string, error) {
	t, found, err := s.repo.GetTeam(ctx, id)
	if err != nil || !found {
		return id, err
	}
	return t.Name, nil
}

// personName returns the name of the person id, or the id itself when there is no such person.
func (s *Service) personName(ctx context.Context, id string) (string, error) {
	p, found, err := s.repo.Get(ctx, id)
	if err != nil || !found {
		return id, err
	}
	return p.Name, nil
}

// personSnapshot reads the person id as history keeps it.
func (s *Service) personSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	p, found, err := s.repo.Get(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	data, err := json.Marshal(p)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	return history.Snapshot{ID: p.ID, Name: p.Name, Data: data}, true, nil
}

// teamSnapshot reads the team id as history keeps it.
func (s *Service) teamSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	t, found, err := s.repo.GetTeam(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	data, err := json.Marshal(t)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	return history.Snapshot{ID: t.ID, Name: t.Name, Data: data}, true, nil
}

// membershipSnapshot reads the membership id as history keeps it: named after its person, under its team.
func (s *Service) membershipSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	m, found, err := s.repo.GetMembership(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	name, err := s.personName(ctx, m.PersonID)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	data, err := json.Marshal(m)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	return history.Snapshot{ID: m.ID, Name: name, ParentID: m.TeamID, Data: data}, true, nil
}

// membershipIDs lists the ids of the memberships of the team teamID.
func (s *Service) membershipIDs(ctx context.Context, teamID string) ([]string, error) {
	memberships, err := s.repo.MembershipsOfTeam(ctx, teamID)
	if err != nil {
		return nil, err
	}
	list := make([]string, 0, len(memberships))
	for _, m := range memberships {
		list = append(list, m.ID)
	}
	return list, nil
}

// putPerson stores the person of a snapshot again, exactly as the snapshot holds it.
func (s *Service) putPerson(ctx context.Context, snapshot history.Snapshot) error {
	var p Person
	if err := json.Unmarshal(snapshot.Data, &p); err != nil {
		return err
	}
	return s.repo.Put(ctx, p)
}

// putTeam stores the team of a snapshot again, exactly as the snapshot holds it.
func (s *Service) putTeam(ctx context.Context, snapshot history.Snapshot) error {
	var t Team
	if err := json.Unmarshal(snapshot.Data, &t); err != nil {
		return err
	}
	return s.repo.PutTeam(ctx, t)
}

// putMembership stores the membership of a snapshot again, exactly as the snapshot holds it.
func (s *Service) putMembership(ctx context.Context, snapshot history.Snapshot) error {
	var m Membership
	if err := json.Unmarshal(snapshot.Data, &m); err != nil {
		return err
	}
	return s.repo.PutMembership(ctx, m)
}

// prepareTeamRestore applies the name rule of teams just before a team comes back (003 data model §2): no
// two teams may have the same name. When another team has meanwhile taken the name, an Undo, which must put
// the team back exactly, is refused with name_taken and the team stays in the recycle bin. A Restore
// renames the team instead, to "<name> (restored)", "<name> (restored 2)" and so on, and says so in a note
// (feature 011 FR-022). The same rule as the sample folders of feature 001.
func (s *Service) prepareTeamRestore(ctx context.Context, rc history.RestoreContext, snapshot *history.Snapshot) ([]string, error) {
	var team Team
	if err := json.Unmarshal(snapshot.Data, &team); err != nil {
		return nil, err
	}
	teams, err := s.repo.ListTeams(ctx)
	if err != nil {
		return nil, err
	}
	other, taken := teamNamed(teams, team.Name, team.ID)
	if !taken {
		return nil, nil
	}
	if rc.Undo {
		return nil, errs.NameTaken(fmt.Sprintf(undoNameTakenMessage, other.Name))
	}

	team.Name = restoredTeamName(teams, team.Name, team.ID)
	team.Touch(rc.Actor, rc.Now) // a changed record gets a new version
	data, err := json.Marshal(team)
	if err != nil {
		return nil, err
	}
	snapshot.Name, snapshot.Data = team.Name, data
	return []string{fmt.Sprintf(renamedNote, team.Name)}, nil
}

// teamNamed returns the team among teams, other than exceptID, whose name is name ignoring letter case,
// and false when there is none. Names are stored without surrounding spaces, so these are ignored too.
func teamNamed(teams []Team, name, exceptID string) (Team, bool) {
	for _, t := range teams {
		if t.ID != exceptID && strings.EqualFold(t.Name, name) {
			return t, true
		}
	}
	return Team{}, false
}

// restoredTeamName returns the first of "<name> (restored)", "<name> (restored 2)", … that no team among
// teams other than id has. The suffix may make the name longer than 80 characters: nothing but a missing
// parent may block a restore (feature 011 FR-025).
func restoredTeamName(teams []Team, name, id string) string {
	candidate := name + " (restored)"
	for n := 2; ; n++ {
		if _, taken := teamNamed(teams, candidate, id); !taken {
			return candidate
		}
		candidate = fmt.Sprintf("%s (restored %d)", name, n)
	}
}
