package access

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/history/historytest"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/fake"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

var start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// The teams and people of the tests: the demo people and teams of the seed.
var (
	lubricants = authz.TeamRef{ID: "T-1001", Name: "Lubricants Europe", Colour: "blue", Active: true}
	mobility   = authz.TeamRef{ID: "T-1002", Name: "Mobility China", Colour: "violet", Active: true}
	globalOps  = authz.TeamRef{ID: "T-1003", Name: "Global Marketing Ops", Colour: "teal", Active: true}

	alex   = actor.Actor{ID: "U-1001", Name: "Alex Morgan"}
	priya  = actor.Actor{ID: "U-1002", Name: "Priya Raman"}
	daniel = actor.Actor{ID: "U-1003", Name: "Daniel Okoro"}
	mei    = actor.Actor{ID: "U-1004", Name: "Mei Lin"}
	carlos = actor.Actor{ID: "U-1005", Name: "Carlos Mendes"}
	sara   = actor.Actor{ID: "U-1006", Name: "Sara Novak"}

	janeActor = actor.Actor{ID: "U-1007", Name: "Jane Doe"} // people added in Settings › People
	johnActor = actor.Actor{ID: "U-1008", Name: "John Smith"}
)

type fakeDirectory struct {
	people []Person
	teams  []authz.TeamRef
}

func (d *fakeDirectory) People(context.Context) ([]Person, error) {
	return slices.Clone(d.people), nil
}

func (d *fakeDirectory) Teams(context.Context) ([]authz.TeamRef, error) {
	return slices.Clone(d.teams), nil
}

func (d *fakeDirectory) CheckPersonVersion(_ context.Context, personID string, version int) error {
	p := d.person(personID)
	if p == nil {
		return errs.NotFound("This person doesn't exist.")
	}
	if p.Version != version {
		return errs.VersionConflict("person", "Someone", start)
	}
	return nil
}

func (d *fakeDirectory) TouchPerson(_ context.Context, _ actor.Actor, personID string) (int, error) {
	p := d.person(personID)
	p.Version++
	return p.Version, nil
}

func (d *fakeDirectory) person(id string) *Person {
	for i := range d.people {
		if d.people[i].Ref.ID == id {
			return &d.people[i]
		}
	}
	return nil
}

// fixture holds the roles of the seed and the six demo people in their teams, with their history kinds.
type fixture struct {
	service   *Service
	history   *history.Service
	db        *fake.DB
	directory *fakeDirectory
	now       time.Time
}

// The ids of the seeded roles (seed/access.go).
const (
	administratorID = "RL-1001"
	globalPlannerID = "RL-1002"
	plannerID       = "RL-1003"
	approverID      = "RL-1004"
	viewerID        = "RL-1005"
)

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := fake.NewDB()
	f := &fixture{db: db, now: start, directory: &fakeDirectory{
		teams: []authz.TeamRef{lubricants, mobility, globalOps},
		people: []Person{
			person(alex, "Marketing administrator", globalOps.ID),
			person(priya, "Marketer", lubricants.ID),
			person(daniel, "Marketing team leader", lubricants.ID),
			person(mei, "Marketer", mobility.ID),
			person(carlos, "Business champion", mobility.ID, globalOps.ID),
			person(sara, "Local finance advisor", lubricants.ID, mobility.ID),
		},
	}}
	generator := ids.NewGenerator(fake.NewCounters(db))
	clock := func() time.Time { return f.now }
	f.history = history.NewService(db, historytest.NewStore(db), generator, clock)
	f.service = NewService(history.Tools{Tx: db, IDs: generator, Log: f.history, Now: clock}, newFakeStore(db), f.directory)
	for _, kind := range append(f.service.Kinds(), f.personKind()) {
		if err := f.history.Register(kind); err != nil {
			t.Fatalf("Register(%s): %v", kind.Name, err)
		}
	}
	f.write(t, func(ctx context.Context) error {
		repo := newFakeStore(db)
		roles := []Role{
			{Name: "Administrator", BuiltIn: authz.BuiltInAdministrator, Permissions: authz.All(), DefaultScope: authz.ScopeAll},
			{Name: "Global planner", BuiltIn: authz.BuiltInGlobalPlanner, DefaultScope: authz.ScopeAll, Permissions: []authz.Permission{
				authz.PlansView, authz.PlansCreate, authz.PlansEdit, authz.PlansPublish, authz.PlansDelete, authz.CampaignsEdit,
				authz.CampaignsDelete, authz.ActivitiesEdit, authz.ActivitiesDelete, authz.ActivitiesSubmit, authz.DataExport,
				authz.FinanceView, authz.FinanceLink, authz.ApprovalsDecide, authz.GlobalBudgetsView, authz.GlobalBudgetsManage}},
			{Name: "Planner", BuiltIn: authz.BuiltInPlanner, DefaultScope: authz.ScopeMyTeams, Permissions: []authz.Permission{
				authz.PlansView, authz.PlansCreate, authz.PlansEdit, authz.PlansPublish, authz.CampaignsEdit, authz.CampaignsDelete,
				authz.ActivitiesEdit, authz.ActivitiesDelete, authz.ActivitiesSubmit, authz.DataExport, authz.FinanceView,
				authz.FinanceLink}},
			{Name: "Approver", BuiltIn: authz.BuiltInApprover, DefaultScope: authz.ScopeMyTeams,
				Permissions: []authz.Permission{authz.PlansView, authz.DataExport, authz.FinanceView, authz.ApprovalsDecide}},
			{Name: "Viewer", BuiltIn: authz.BuiltInViewer, DefaultScope: authz.ScopeMyTeams,
				Permissions: []authz.Permission{authz.PlansView, authz.DataExport, authz.FinanceView}},
		}
		for _, r := range roles {
			var err error
			if r.ID, err = generator.Next(ctx, ids.PrefixRole); err != nil {
				return err
			}
			r.Meta = record.NewMeta(actor.System, start)
			if err := repo.PutRole(ctx, r); err != nil {
				return err
			}
		}
		for _, a := range []struct {
			person actor.Actor
			role   string
			scope  authz.ScopeType
		}{
			{alex, administratorID, authz.ScopeAll}, {carlos, globalPlannerID, authz.ScopeAll},
			{daniel, plannerID, authz.ScopeMyTeams}, {daniel, approverID, authz.ScopeMyTeams},
			{priya, plannerID, authz.ScopeMyTeams}, {mei, plannerID, authz.ScopeMyTeams}, {sara, viewerID, authz.ScopeMyTeams},
		} {
			id, err := generator.Next(ctx, ids.PrefixRoleAssignment)
			if err != nil {
				return err
			}
			if err := repo.PutAssignment(ctx, Assignment{ID: id, PersonID: a.person.ID, RoleID: a.role, ScopeType: a.scope,
				TeamIDs: []string{}, Meta: record.NewMeta(actor.System, start)}); err != nil {
				return err
			}
		}
		return repo.PutRules(ctx, Rules{NoSelfApproval: true, NewPersonRoleID: viewerID, Meta: record.NewMeta(actor.System, start)})
	})
	return f
}

func person(a actor.Actor, jobTitle string, teamIDs ...string) Person {
	return Person{Ref: authz.PersonRef{ID: a.ID, Name: a.Name, Initials: initials(a.Name), JobTitle: jobTitle, Active: true},
		Email: a.ID + "@shell.com", TeamIDs: teamIDs, Version: 1}
}

func initials(name string) string {
	first, last := name[:1], ""
	for i := len(name) - 1; i > 0; i-- {
		if name[i-1] == ' ' {
			last = name[i : i+1]
			break
		}
	}
	return first + last
}

// personKind is the history kind of people, enough for the log entries of a person's roles.
func (f *fixture) personKind() history.Kind {
	return history.Kind{
		Name: personKind, Singular: "person", Plural: "people",
		Snapshot: func(_ context.Context, id string) (history.Snapshot, bool, error) {
			p := f.directory.person(id)
			if p == nil {
				return history.Snapshot{}, false, nil
			}
			data, err := json.Marshal(p.Ref)
			return history.Snapshot{ID: id, Name: p.Ref.Name, Data: data}, true, err
		},
		Remove: func(context.Context, string) error { return nil },
		Put:    func(context.Context, history.Snapshot) error { return nil },
		Describe: func(_ context.Context, field string, _ any) (string, string, error) {
			if field == rolesField {
				return "Roles", "", nil
			}
			return "", "", nil
		},
	}
}

func (f *fixture) write(t *testing.T, fn func(ctx context.Context) error) {
	t.Helper()
	if err := f.db.Write(t.Context(), fn); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func (f *fixture) grants(t *testing.T, personID string) authz.Grants {
	t.Helper()
	g, err := f.service.GrantsFor(t.Context(), personID)
	if err != nil {
		t.Fatalf("GrantsFor(%s): %v", personID, err)
	}
	return g
}

func (f *fixture) log(t *testing.T) []history.Entry {
	t.Helper()
	entries, err := f.history.Entries(t.Context())
	if err != nil {
		t.Fatalf("Entries: %v", err)
	}
	return entries
}

// planOf is a plan of the teams teamIDs, owned and created by nobody of the tests.
func planOf(teamIDs ...string) authz.PlanFacts {
	return authz.PlanFacts{ID: "P-1001", TeamIDs: teamIDs, OwnerID: "U-9999", CreatedByID: "U-9999"}
}

func version(v int) *int {
	return &v
}

func apiError(t *testing.T, err error) *errs.Error {
	t.Helper()
	apiErr, ok := err.(*errs.Error)
	if !ok {
		t.Fatalf("err = %v (%T), want an API error", err, err)
	}
	return apiErr
}
