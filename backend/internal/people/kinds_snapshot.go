package people

import (
	"context"
	"encoding/json"

	"mapp/internal/history"
)

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
