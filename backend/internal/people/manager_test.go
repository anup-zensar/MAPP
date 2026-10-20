package people

import (
	"reflect"
	"testing"

	"northstar/internal/platform/errs"
)

// A person's line manager (017 FR-016, data-model §9): optional; a known person; not the person; active
// when added or changed (one who became inactive may stay); no loop through the chain of managers. Every
// problem comes with the form's others, and a change is logged with the managers' names.
func TestManager_Rules_FR016(t *testing.T) {
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	daniel := f.addPerson(t, "Daniel Okoro", "", true)
	sara := f.addPerson(t, "Sara Novak", "", false)

	// Priya's line manager becomes Daniel Okoro, then Daniel's becomes Alex Morgan.
	got, err := f.service.UpdatePerson(t.Context(), alex.Actor(), priya.ID, PersonUpdate{ManagerID: text(daniel.ID), Version: version(1)})
	if err != nil {
		t.Fatalf("UpdatePerson(Priya): %v", err)
	}
	if got.Version != 2 {
		t.Errorf("Priya's version = %d, want 2", got.Version)
	}
	if _, err := f.service.UpdatePerson(t.Context(), alex.Actor(), daniel.ID, PersonUpdate{ManagerID: text(alex.ID), Version: version(1)}); err != nil {
		t.Fatalf("UpdatePerson(Daniel): %v", err)
	}
	assertLog(t, f, []string{
		"Alex Morgan changed person 'Priya Raman' (U-1002): Line manager Not filled in → Daniel Okoro",
		"Alex Morgan changed person 'Daniel Okoro' (U-1003): Line manager Not filled in → Alex Morgan",
	})

	tests := []struct {
		name     string
		personID string
		version  int
		manager  string
		want     string
	}{
		{"unknown", priya.ID, 2, "U-9999", "Choose the line manager from the list."},
		{"themselves", priya.ID, 2, priya.ID, "A person can't be their own line manager."},
		{"inactive", priya.ID, 2, sara.ID, "Sara Novak is inactive — choose an active person."},
		{"direct loop", daniel.ID, 2, priya.ID, "Priya Raman reports to Daniel Okoro, so Daniel Okoro can't report to Priya Raman."},
		{"loop through the chain", alex.ID, 1, priya.ID, "Priya Raman reports to Alex Morgan, so Alex Morgan can't report to Priya Raman."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.service.UpdatePerson(t.Context(), alex.Actor(), tt.personID,
				PersonUpdate{ManagerID: text(tt.manager), Version: version(tt.version)})
			assertError(t, err, errs.Validation(errs.FieldError{Field: "managerId", Message: tt.want}))
		})
	}

	// A new person may get a manager at once, with every other problem of the form reported beside it.
	_, err = f.service.CreatePerson(t.Context(), alex.Actor(), PersonInput{Name: " ", Email: "jane.doe@northstar.demo", ManagerID: sara.ID})
	assertError(t, err, errs.Validation(
		errs.FieldError{Field: "name", Message: "Enter a name."},
		errs.FieldError{Field: "managerId", Message: "Sara Novak is inactive — choose an active person."},
	))
	jane, err := f.service.CreatePerson(t.Context(), alex.Actor(), PersonInput{Name: "Jane Doe", Email: "jane.doe@northstar.demo", ManagerID: daniel.ID})
	if err != nil {
		t.Fatalf("CreatePerson: %v", err)
	}

	// A manager who became inactive stays; clearing the manager is a change too.
	inactiveDaniel := f.storedPerson(t, daniel.ID)
	inactiveDaniel.Active = false
	f.putPerson(t, inactiveDaniel)
	if _, err := f.service.UpdatePerson(t.Context(), alex.Actor(), jane.ID, PersonUpdate{JobTitle: text("Marketer"), Version: version(1)}); err != nil {
		t.Errorf("UpdatePerson keeping an inactive manager: %v", err)
	}
	cleared, err := f.service.UpdatePerson(t.Context(), alex.Actor(), priya.ID, PersonUpdate{ManagerID: text(""), Version: version(2)})
	if err != nil || cleared.Version != 3 {
		t.Errorf("clearing the manager = version %d, %v; want version 3", cleared.Version, err)
	}

	list, err := f.service.ListPeople(t.Context(), alex.Actor())
	if err != nil {
		t.Fatalf("ListPeople: %v", err)
	}
	byName := map[string]PersonListItem{}
	for _, p := range list {
		byName[p.Name] = p
	}
	if m := byName["Jane Doe"].Manager; m == nil || m.Name != "Daniel Okoro" || m.Active {
		t.Errorf("Jane's manager = %+v, want Daniel Okoro (inactive)", m)
	}
	if byName["Priya Raman"].Manager != nil || byName["Daniel Okoro"].ReportCount != 1 || byName["Alex Morgan"].ReportCount != 1 {
		t.Errorf("Priya's manager %+v, Daniel's reports %d, Alex's reports %d; want none, 1, 1",
			byName["Priya Raman"].Manager, byName["Daniel Okoro"].ReportCount, byName["Alex Morgan"].ReportCount)
	}
}

// The people a person is the line manager of, active or not, are the last usage count of their details:
// "Line manager of 2 people" in the details and the Deactivate dialog (017 FR-016).
func TestUsage_Manager_FR016(t *testing.T) {
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	daniel := f.addPerson(t, "Daniel Okoro", "", true)
	priya := f.addPerson(t, "Priya Raman", "", true)
	sara := f.addPerson(t, "Sara Novak", "", false)
	for _, p := range []Person{priya, sara} {
		p.ManagerID = daniel.ID
		f.putPerson(t, p)
	}
	alexAsReport := alex
	alexAsReport.ManagerID = priya.ID
	f.putPerson(t, alexAsReport)

	tests := []struct {
		person Person
		want   []UsageCount
	}{
		{daniel, []UsageCount{{Role: "manager", RecordType: "person", Label: "people", Count: 2}}},
		{priya, []UsageCount{{Role: "manager", RecordType: "person", Label: "person", Count: 1}}},
		{sara, []UsageCount{}},
	}
	for _, tt := range tests {
		got, err := f.service.GetPerson(t.Context(), alex.Actor(), tt.person.ID)
		if err != nil {
			t.Fatalf("GetPerson(%s): %v", tt.person.Name, err)
		}
		if !reflect.DeepEqual(got.Usage, tt.want) {
			t.Errorf("usage of %s = %+v, want %+v", tt.person.Name, got.Usage, tt.want)
		}
	}
}
