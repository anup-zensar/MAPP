package settings

import (
	"encoding/json"
	"slices"
	"testing"

	"northstar/internal/history"
)

// move moves the attribute id as Alex.
func (f *fixture) move(t *testing.T, id, what, direction string, v int) ([]ShellAttributeView, error) {
	t.Helper()
	return f.service.MoveAttribute(t.Context(), alex, id, MoveInput{What: what, Direction: direction, Version: version(v)})
}

// positionsOf returns "<label>:<position>" of each attribute, in the order given.
func positionsOf(list []ShellAttributeView) []string {
	out := []string{}
	for _, a := range list {
		out = append(out, a.Label+":"+itoaTest(a.Position))
	}
	return out
}

// itoaTest writes a small whole number.
func itoaTest(n int) string {
	encoded, _ := json.Marshal(n)
	return string(encoded)
}

// An attribute swaps places with its neighbour in its section, and the level's list answers in the new
// order (US2-6, FR-019).
func TestMoveAttribute_WithinSection_FR019(t *testing.T) {
	f := newTestService(t)
	channel := shellForm(t, f).attribute(LevelActivity, "Channel")

	list, err := f.move(t, channel.ID, MoveAttribute, MoveUp, 1)

	if got := positionsOf(list); err != nil || !slices.Equal(got, []string{"Cluster:1", "Channel:2", "Region:3", "Brands:4"}) {
		t.Errorf("after the move = %q, %v", got, err)
	}
	if got := f.labelsOf(t, LevelActivity); !slices.Equal(got, []string{"Cluster", "Channel", "Region", "Brands"}) {
		t.Errorf("the form's order = %q", got)
	}
}

// Nothing moves past the edge of its section (FR-019, data model §4.2).
func TestMoveAttribute_EdgeOfSectionRefused_FR019(t *testing.T) {
	f := newTestService(t)
	tf := shellForm(t, f)

	_, first := f.move(t, tf.attribute(LevelActivity, "Cluster").ID, MoveAttribute, MoveUp, 1)
	_, last := f.move(t, tf.attribute(LevelActivity, "Channel").ID, MoveAttribute, MoveDown, 1)
	_, alone := f.move(t, tf.attribute(LevelActivity, "Brands").ID, MoveAttribute, MoveUp, 1)

	assertError(t, first, problems("direction", "'Cluster' is already first in its section."))
	assertError(t, last, problems("direction", "'Channel' is already last in its section."))
	assertError(t, alone, problems("direction", "'Brands' is already first in its section."))
	if len(f.logEntries(t)) != 0 {
		t.Errorf("refused moves were logged")
	}
}

// A section moves as a whole before the previous section, and the level is numbered 1…n again; the first
// and the last section go no further (FR-019).
func TestMoveSection_RenumbersLevel_FR019(t *testing.T) {
	f := newTestService(t)
	tf := shellForm(t, f)
	tf = f.addAttributes(t, testAttribute{level: LevelActivity, label: "Notes", typ: TypeText, section: "Extra"})

	list, err := f.move(t, tf.attribute(LevelActivity, "Brands").ID, MoveSection, MoveUp, 1)
	_, first := f.move(t, tf.attribute(LevelActivity, "Brands").ID, MoveSection, MoveUp, 2)
	_, last := f.move(t, tf.attribute(LevelActivity, "Notes").ID, MoveSection, MoveDown, 1)

	want := []string{"Brands:1", "Cluster:2", "Region:3", "Channel:4", "Notes:5"}
	if got := positionsOf(list); err != nil || !slices.Equal(got, want) {
		t.Errorf("after the section move = %q, %v", got, err)
	}
	assertError(t, first, problems("direction", "This section is already first."))
	assertError(t, last, problems("direction", "This section is already last."))
}

// A dependent always stays below its parent: a move or a section change that would put it above is
// refused and nothing changes (edge case, FR-022, data model §4.2).
func TestMove_DependentStaysBelowParent_EdgeCase(t *testing.T) {
	f := newTestService(t)
	tf := shellForm(t, f)
	cluster, region := tf.attribute(LevelActivity, "Cluster"), tf.attribute(LevelActivity, "Region")

	_, up := f.move(t, region.ID, MoveAttribute, MoveUp, 1)
	_, down := f.move(t, cluster.ID, MoveAttribute, MoveDown, 1)
	_, section := f.service.UpdateAttribute(t.Context(), alex, cluster.ID, ShellAttributeUpdate{Section: text("Later"), Version: version(1)})

	assertError(t, up, problems("direction", "Region must stay below Cluster."))
	assertError(t, down, problems("direction", "Region must stay below Cluster."))
	assertError(t, section, problems("section", "Region must stay below Cluster."))
	if got := f.labelsOf(t, LevelActivity); !slices.Equal(got, []string{"Cluster", "Region", "Channel", "Brands"}) {
		t.Errorf("order = %q", got)
	}
}

// The Business unit attribute stays first on the plan form: nothing moves above it (FR-023).
func TestMove_BusinessUnitStaysFirst_FR023(t *testing.T) {
	f := newTestService(t)
	shellForm(t, f)
	tf := f.addAttributes(t,
		testAttribute{level: LevelPlan, label: "Budget owner", typ: TypeText},
		testAttribute{level: LevelPlan, label: "Finance code", typ: TypeText, section: "Finance"},
	)

	_, owner := f.move(t, tf.attribute(LevelPlan, "Budget owner").ID, MoveAttribute, MoveUp, 1)
	_, finance := f.move(t, tf.attribute(LevelPlan, "Finance code").ID, MoveSection, MoveUp, 1)

	const message = "Business unit stays first on the plan form, because other attributes are shown or hidden by it."
	assertError(t, owner, problems("direction", message))
	assertError(t, finance, problems("direction", message))
}

// A new section puts the attribute last in it; nothing else moves (FR-019, data model §4.2).
func TestChangeSection_GoesLastInNewSection_FR019(t *testing.T) {
	f := newTestService(t)
	tf := shellForm(t, f)
	channel := tf.attribute(LevelActivity, "Channel")

	existing, err := f.service.UpdateAttribute(t.Context(), alex, channel.ID, ShellAttributeUpdate{Section: text("Lubricants attributes"), Version: version(1)})
	if err != nil || existing.Position != 5 {
		t.Fatalf("to Lubricants attributes = %+v, %v", existing, err)
	}
	if got := f.labelsOf(t, LevelActivity); !slices.Equal(got, []string{"Cluster", "Region", "Brands", "Channel"}) {
		t.Errorf("order = %q", got)
	}
	added, err := f.service.UpdateAttribute(t.Context(), alex, tf.attribute(LevelActivity, "Cluster").ID, ShellAttributeUpdate{HelpText: text("Where"), Version: version(1)})
	if err != nil || added.Position != 1 {
		t.Errorf("an edit without a new section moved it: %+v, %v", added, err)
	}
	media, err := f.service.UpdateAttribute(t.Context(), alex, channel.ID, ShellAttributeUpdate{Section: text("Media"), Version: version(2)})
	if err != nil || media.Position != 5 || f.labelsOf(t, LevelActivity)[3] != "Channel" {
		t.Errorf("to a new section = %+v, %v", media, err)
	}
	list, _ := f.service.ListAttributes(t.Context(), alex, nil)
	for _, a := range list {
		if a.Label != "Channel" && a.Label != "Cluster" && a.Version != 1 {
			t.Errorf("%s changed to version %d", a.Label, a.Version)
		}
	}
}

// Each attribute whose place changed gets one "changed" entry with the field position (FR-006).
func TestMove_LogsEachChangedPlace_FR006(t *testing.T) {
	f := newTestService(t)
	tf := shellForm(t, f)

	if _, err := f.move(t, tf.attribute(LevelActivity, "Channel").ID, MoveAttribute, MoveUp, 1); err != nil {
		t.Fatalf("move: %v", err)
	}

	entries := f.logEntries(t)
	if len(entries) != 2 {
		t.Fatalf("log = %+v", entries)
	}
	got := map[string]history.Change{}
	for _, e := range entries {
		if e.Action != history.ActionChanged || e.RecordType != "shellAttribute" || len(e.Changes) != 1 {
			t.Errorf("entry = %+v", e)
		}
		got[e.RecordName] = e.Changes[0]
	}
	if c := got["Channel"]; c.Field != "position" || c.Label != "Place on the form" || string(c.Old) != "3" || string(c.New) != "2" {
		t.Errorf("Channel's change = %+v", c)
	}
	if c := got["Region"]; string(c.Old) != "2" || string(c.New) != "3" {
		t.Errorf("Region's change = %+v", c)
	}
}
