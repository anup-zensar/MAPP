package access

import (
	"context"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
)

// The access rules: segregation of duties and the role for new people, changed with the version and
// logged with their labels and texts; the Administrator role is never the role for new people (FR-019,
// FR-023, FR-037, message 17).
func TestUpdateRules_ChangedAndLogged_FR019_FR037(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.now = start.Add(time.Hour)
	off, planner, admin := false, plannerID, administratorID

	_, adminErr := f.service.UpdateRules(t.Context(), alex, RulesUpdate{Version: version(1), NewPersonRoleID: &admin})
	_, missingErr := f.service.UpdateRules(t.Context(), alex, RulesUpdate{})
	got, err := f.service.UpdateRules(t.Context(), alex, RulesUpdate{Version: version(1), NoSelfApproval: &off, NewPersonRoleID: &planner})
	_, staleErr := f.service.UpdateRules(t.Context(), alex, RulesUpdate{Version: version(1), NoSelfApproval: &off})

	if a := apiError(t, adminErr); !reflect.DeepEqual(a.Fields, []errs.FieldError{{Field: "newPersonRoleId",
		Message: "New people can't be made administrators automatically."}}) {
		t.Errorf("Administrator for new people = %+v", a.Fields)
	}
	if a := apiError(t, missingErr); a.Status != http.StatusBadRequest || a.Fields[0].Field != "version" {
		t.Errorf("no version = %+v", a)
	}
	if err != nil {
		t.Fatalf("UpdateRules: %v", err)
	}
	if got.NoSelfApproval || got.NewPersonRole == nil || got.NewPersonRole.Name != "Planner" || got.Version != 2 ||
		got.UpdatedAt != f.now || got.UpdatedBy != alex {
		t.Errorf("rules = %+v", got)
	}
	if a := apiError(t, staleErr); a.Code != errs.CodeVersionConflict || a.Message != "Not saved — Alex Morgan changed this access rules" {
		t.Errorf("an out-of-date change = %+v", a)
	}
	entries := f.log(t)
	if len(entries) != 1 {
		t.Fatalf("log = %+v, want one entry", entries)
	}
	var texts []string
	for _, c := range entries[0].Changes {
		texts = append(texts, c.Label+": "+c.OldText+" → "+c.NewText)
	}
	want := []string{"Role for new people: Viewer → Planner", "Nobody approves an activity they own or submitted: On → Off"}
	if entries[0].RecordType != RulesKind || entries[0].RecordName != "Access rules" || !slices.Equal(texts, want) {
		t.Errorf("entry %s %q: %q\nwant %q", entries[0].RecordType, entries[0].RecordName, texts, want)
	}
}

// While the rules were never saved, the defaults apply with version 0, and the first change saves them
// (data model §1.3).
func TestUpdateRules_FirstSaveFromDefaults_FR019(t *testing.T) {
	t.Parallel()
	service := newEmptyStoreService(t)
	on := true
	defaults, err := service.GetRules(t.Context(), alex)
	if err != nil || defaults.Version != 0 || !defaults.NoSelfApproval || defaults.NewPersonRole != nil || defaults.UpdatedAt != nil {
		t.Errorf("rules of an empty store = %+v, %v; want the defaults with version 0", defaults, err)
	}

	got, err := service.UpdateRules(t.Context(), alex, RulesUpdate{Version: version(0), NoSelfApproval: &on})
	if err != nil || got.Version != 0 {
		t.Errorf("a change that keeps the defaults = %+v, %v; want nothing saved", got, err)
	}
	off := false
	got, err = service.UpdateRules(t.Context(), alex, RulesUpdate{Version: version(0), NoSelfApproval: &off})
	if err != nil || got.Version != 1 || got.NoSelfApproval || got.UpdatedBy != alex {
		t.Errorf("the first change = %+v, %v; want version 1", got, err)
	}
}

// Recent access changes are the latest entries about roles, the access rules and people's roles, newest
// first, at most 20; other changes of people are left out (FR-038, US7).
func TestRecentChanges_AccessEntriesNewestFirst_FR038(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	analyst, _ := f.service.CreateRole(t.Context(), alex, RoleInput{Name: "Analyst", Permissions: []authz.Permission{authz.PlansView},
		DefaultScope: authz.ScopeAll})
	off := false
	if _, err := f.service.UpdateRules(t.Context(), alex, RulesUpdate{Version: version(1), NoSelfApproval: &off}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.SetPersonRoles(t.Context(), alex, sara.ID, PersonRolesUpdate{Version: version(1), Roles: []AssignmentInput{
		{RoleID: analyst.ID, Scope: ScopeInput{Type: authz.ScopeAll}}}}); err != nil {
		t.Fatal(err)
	}
	f.write(t, func(ctx context.Context) error { // a change of a person that is not about access
		_, err := f.history.Log(ctx, alex, history.LogInput{Kind: personKind, RecordID: sara.ID, RecordName: sara.Name,
			Action: history.ActionChanged, Changes: []history.Change{{Field: "jobTitle", Old: encode(""), New: encode("Advisor")}}})
		return err
	})
	for i := range 20 {
		name := "Analyst " + string(rune('A'+i))
		if _, err := f.service.UpdateRole(t.Context(), alex, analyst.ID, RoleUpdate{Version: version(i + 1), Name: &name}); err != nil {
			t.Fatal(err)
		}
	}

	entries, total, err := f.service.RecentChanges(t.Context(), alex)

	if err != nil || total != 23 || len(entries) != 20 {
		t.Fatalf("RecentChanges = %d entries of %d, %v; want 20 of 23", len(entries), total, err)
	}
	if entries[0].RecordName != "Analyst T" || entries[19].RecordName != "Analyst A" || entries[0].Seq < entries[1].Seq {
		t.Errorf("first %q, last %q: want newest first", entries[0].RecordName, entries[19].RecordName)
	}
}
