package access

import (
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"

	"northstar/internal/platform/authz"
	"northstar/internal/platform/errs"
)

// The role page groups the fixed permissions as Table A does (FR-001).
func TestPermissions_GroupedAsTableA_FR001(t *testing.T) {
	got := newFixture(t).service.Permissions()

	var groups []string
	count := 0
	for _, g := range got.Groups {
		groups = append(groups, g.Name)
		count += len(g.Permissions)
	}
	if !slices.Equal(groups, authz.Groups) || count != 25 {
		t.Errorf("groups = %q with %d permissions, want Table A's five groups and 25 permissions", groups, count)
	}
	first := got.Groups[0].Permissions[0]
	if first != (PermissionInfo{Key: authz.PlansView, Label: "View plans and everything in them", Scoped: true}) {
		t.Errorf("first permission = %+v", first)
	}
}

// The roles list starts with Administrator, then by name, each with the people who hold it; the
// Administrator role is locked and lists every permission (FR-018, message 11).
func TestListRoles_AdministratorFirstWithLocks_FR018(t *testing.T) {
	f := newFixture(t)

	roles, err := f.service.ListRoles(t.Context(), alex)

	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	var names []string
	for _, r := range roles {
		names = append(names, r.Name)
	}
	if want := []string{"Administrator", "Approver", "Global planner", "Planner", "Viewer"}; !slices.Equal(names, want) {
		t.Errorf("roles = %q, want %q", names, want)
	}
	admin := roles[0]
	if len(admin.Permissions) != 25 || len(admin.Locks) != 5 || admin.Locks[0].Reason != msgAdministratorLock ||
		len(admin.People) != 1 || admin.People[0].ID != alex.ID {
		t.Errorf("Administrator = %+v", admin)
	}
	planner := roles[3]
	var holders []string
	for _, p := range planner.People {
		holders = append(holders, p.Name)
	}
	if !slices.Equal(holders, []string{"Daniel Okoro", "Mei Lin", "Priya Raman"}) ||
		!reflect.DeepEqual(planner.Locks, []Lock{{Field: "delete", Reason: "'Planner' is given to 3 people. Take it away from them first."}}) {
		t.Errorf("Planner = holders %q, locks %+v", holders, planner.Locks)
	}
}

// A new role needs a unique name, at most 60 characters, a description of at most 300, at least one
// known permission and a default scope; every problem is reported at once (FR-018, messages 14, 15).
func TestCreateRole_Rules_FR018(t *testing.T) {
	f := newFixture(t)

	_, err := f.service.CreateRole(t.Context(), alex, RoleInput{Name: " planner ", Description: string(make([]byte, 301)),
		Permissions: []authz.Permission{}, DefaultScope: authz.ScopeTeams})
	got := apiError(t, err)
	want := []errs.FieldError{
		{Field: "name", Message: "A role called 'planner' already exists."},
		{Field: "description", Message: "Use 300 characters or fewer."},
		{Field: "permissions", Message: "Choose at least one permission."},
		{Field: "defaultScope", Message: "Choose All markets or My teams."},
	}
	if got.Status != http.StatusBadRequest || !reflect.DeepEqual(got.Fields, want) {
		t.Errorf("CreateRole = %d %+v\nwant 400 %+v", got.Status, got.Fields, want)
	}
	_, err = f.service.CreateRole(t.Context(), alex, RoleInput{Name: "Analyst", Permissions: []authz.Permission{"plans.fly"},
		DefaultScope: authz.ScopeAll})
	if got := apiError(t, err); !reflect.DeepEqual(got.Fields, []errs.FieldError{{Field: "permissions", Message: "'plans.fly' is not a permission."}}) {
		t.Errorf("unknown permission = %+v", got.Fields)
	}

	role, err := f.service.CreateRole(t.Context(), alex, RoleInput{Name: " Analyst ", Description: "Reads and exports.",
		Permissions: []authz.Permission{authz.DataExport, authz.PlansView, authz.DataExport}, DefaultScope: authz.ScopeMyTeams})

	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if role.ID != "RL-1006" || role.Name != "Analyst" || role.BuiltIn != "" ||
		!slices.Equal(role.Permissions, []authz.Permission{authz.PlansView, authz.DataExport}) || role.Version != 1 ||
		len(role.People) != 0 || len(role.Locks) != 0 {
		t.Errorf("created role = %+v", role)
	}
	entries := f.log(t)
	if len(entries) != 1 || entries[0].RecordType != RoleKind || entries[0].Verb != "created" ||
		entries[0].RecordName != "Analyst" || entries[0].Details != "with 2 permissions" {
		t.Errorf("log = %+v, want one created entry with 2 permissions", entries)
	}
}

// Changing a role keeps the fields not sent, refuses an out-of-date version, logs every changed field with
// permissions as labels and "Added: …; removed: …", and writes nothing when nothing changes (FR-018,
// FR-037, data model §3).
func TestUpdateRole_ChangedFieldsLogged_FR018_FR037(t *testing.T) {
	f := newFixture(t)
	f.now = start.Add(time.Hour)
	permissions := []authz.Permission{authz.PlansView, authz.FinanceView, authz.ActivitiesSubmit}
	scope := authz.ScopeAll

	role, err := f.service.UpdateRole(t.Context(), alex, viewerID, RoleUpdate{Version: version(1), Permissions: &permissions,
		DefaultScope: &scope})

	if err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	if role.Name != "Viewer" || role.Version != 2 || role.DefaultScope != authz.ScopeAll ||
		!slices.Equal(role.Permissions, []authz.Permission{authz.PlansView, authz.ActivitiesSubmit, authz.FinanceView}) {
		t.Errorf("updated role = %+v", role)
	}
	entries := f.log(t)
	if len(entries) != 1 {
		t.Fatalf("log = %+v, want one entry", entries)
	}
	e := entries[0]
	if e.Verb != "changed" || e.RecordID != viewerID || len(e.Changes) != 2 ||
		e.Details != "Added: Submit activities for approval and recall them; removed: Export from All work" {
		t.Errorf("entry = %+v", e)
	}
	var texts []string
	for _, c := range e.Changes {
		texts = append(texts, c.Label+": "+c.OldText+" → "+c.NewText)
	}
	want := []string{
		"Default scope: My teams → All markets",
		"Permissions: View plans and everything in them, Export from All work, See financial data → " +
			"View plans and everything in them, Submit activities for approval and recall them, See financial data",
	}
	if !slices.Equal(texts, want) {
		t.Errorf("changes =\n%q\nwant\n%q", texts, want)
	}

	_, err = f.service.UpdateRole(t.Context(), alex, viewerID, RoleUpdate{Version: version(1), Permissions: &permissions})
	if got := apiError(t, err); got.Code != errs.CodeVersionConflict || got.Message != "Not saved — Alex Morgan changed this role" {
		t.Errorf("out-of-date update = %+v", got)
	}
	if _, err := f.service.UpdateRole(t.Context(), alex, viewerID, RoleUpdate{Version: version(2), Permissions: &permissions}); err != nil {
		t.Fatalf("an update that changes nothing: %v", err)
	}
	if n := len(f.log(t)); n != 1 {
		t.Errorf("an update that changes nothing wrote a log entry (%d entries)", n)
	}
}

// The Administrator role can't be changed or deleted (FR-018, message 11).
func TestAdministratorRole_Locked_FR018(t *testing.T) {
	f := newFixture(t)
	name := "Boss"

	_, updateErr := f.service.UpdateRole(t.Context(), alex, administratorID, RoleUpdate{Version: version(1), Name: &name})
	_, deleteErr := f.service.DeleteRole(t.Context(), alex, administratorID)

	for _, err := range []error{updateErr, deleteErr} {
		if got := apiError(t, err); got.Status != http.StatusConflict || got.Code != errs.CodeNotAllowed ||
			got.Message != "The Administrator role always has every permission and can't be changed." {
			t.Errorf("err = %+v, want 409 not_allowed with message 11", got)
		}
	}
}

// A role someone holds can't be deleted (409 in_use, message 13); an unused role goes to the recycle bin and
// Undo brings it back; when it was the role for new people, the rule becomes "No role", logged (FR-018,
// data model §1.3).
func TestDeleteRole_InUseThenBinAndUndo_FR018(t *testing.T) {
	f := newFixture(t)

	_, err := f.service.DeleteRole(t.Context(), alex, approverID)
	if got := apiError(t, err); got.Code != errs.CodeInUse || got.Message != "'Approver' is given to 1 person. Take it away from them first." {
		t.Errorf("deleting a role Daniel holds = %+v", got)
	}

	if _, err := f.service.SetPersonRoles(t.Context(), alex, sara.ID, PersonRolesUpdate{Version: version(1), Roles: []AssignmentInput{
		{RoleID: plannerID, Scope: ScopeInput{Type: authz.ScopeMyTeams}}}}); err != nil {
		t.Fatalf("SetPersonRoles: %v", err)
	}
	entry, err := f.service.DeleteRole(t.Context(), alex, viewerID)
	if err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}
	rules, _ := f.service.GetRules(t.Context(), alex)
	if entry.RecordType != RoleKind || rules.NewPersonRole != nil || rules.Version != 2 {
		t.Errorf("bin entry %+v, rules %+v; want the role in the bin and no role for new people", entry, rules)
	}
	entries := f.log(t)
	last := entries[len(entries)-2] // the rule change, then the delete
	if last.RecordType != RulesKind || len(last.Changes) != 1 || last.Changes[0].OldText != "Viewer" || last.Changes[0].NewText != "No role" {
		t.Errorf("last log entry = %+v, want the rule change Viewer → No role", last)
	}

	if _, err := f.history.Undo(t.Context(), alex, entry.ID); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if role, err := f.service.GetRole(t.Context(), alex, viewerID); err != nil || role.Name != "Viewer" {
		t.Errorf("after Undo GetRole = %+v, %v", role, err)
	}
}

// A role restored from the bin while another role took its name is renamed "<name> (restored)"; an Undo is
// refused instead (001's name rule, FR-018).
func TestRestoreRole_NameTaken_FR018(t *testing.T) {
	f := newFixture(t)
	analyst, err := f.service.CreateRole(t.Context(), alex, RoleInput{Name: "Analyst", Permissions: []authz.Permission{authz.PlansView},
		DefaultScope: authz.ScopeAll})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	entry, err := f.service.DeleteRole(t.Context(), alex, analyst.ID)
	if err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}
	if _, err := f.service.CreateRole(t.Context(), alex, RoleInput{Name: "analyst", Permissions: []authz.Permission{authz.PlansView},
		DefaultScope: authz.ScopeAll}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	_, undoErr := f.history.Undo(t.Context(), alex, entry.ID)
	restored, restoreErr := f.history.Restore(t.Context(), alex, entry.ID)

	if got := apiError(t, undoErr); got.Code != errs.CodeNameTaken {
		t.Errorf("Undo = %+v, want name_taken", got)
	}
	if restoreErr != nil {
		t.Fatalf("Restore: %v", restoreErr)
	}
	role, _ := f.service.GetRole(t.Context(), alex, analyst.ID)
	if role.Name != "Analyst (restored)" || restored.Details != "Renamed to 'Analyst (restored)' because the name was taken." {
		t.Errorf("restored role %q with details %q", role.Name, restored.Details)
	}
}
