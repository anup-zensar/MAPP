package settings

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
)

// A rename shows at once and the log keeps the old and the new label; saving without a change writes
// nothing; an out-of-date version is refused (US2-4, FR-006, FR-007).
func TestRenameAttribute_LogKeepsOldAndNew_FR006(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	channel := shellForm(t, f).attribute(LevelActivity, "Channel")

	renamed, err := f.service.UpdateAttribute(t.Context(), alex, channel.ID, ShellAttributeUpdate{Label: text(" Channel / audience "), Version: version(1)})
	if err != nil || renamed.Label != "Channel / audience" || renamed.Version != 2 {
		t.Fatalf("renamed = %+v, %v", renamed, err)
	}
	entry := f.lastEntry(t)
	want := []history.Change{{Field: "label", Label: "Label", Old: json.RawMessage(`"Channel"`), New: json.RawMessage(`"Channel / audience"`),
		OldText: "Channel", NewText: "Channel / audience"}}
	if entry.Action != history.ActionChanged || entry.RecordName != "Channel / audience" || !reflect.DeepEqual(entry.Changes, want) {
		t.Errorf("entry = %+v", entry)
	}
	same, err := f.service.UpdateAttribute(t.Context(), alex, channel.ID, ShellAttributeUpdate{Label: text("Channel / audience"), Version: version(2)})
	if err != nil || same.Version != 2 || len(f.logEntries(t)) != 1 {
		t.Errorf("no change: %+v, %v", same, err)
	}
	_, stale := f.service.UpdateAttribute(t.Context(), alex, channel.ID, ShellAttributeUpdate{Label: text("Media"), Version: version(1)})
	assertError(t, stale, errs.VersionConflict("Shell attribute", "Alex Morgan", start))
	_, gone := f.service.UpdateAttribute(t.Context(), alex, "SA-9999", ShellAttributeUpdate{Version: version(1)})
	assertError(t, gone, errs.NotFound("This Shell attribute no longer exists — someone may have deleted it."))
}

// Type and level are locked once a record has a value — also a record in the recycle bin only (FR-021).
func TestUpdateAttribute_TypeAndLevelLockedWhenValues_FR021(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	tf := shellForm(t, f)
	channel := tf.attribute(LevelActivity, "Channel")
	f.refs = []Reference{activityRef("A-1001", "P-1001", "", true, values(channel.ID, `["`+tf.option(LevelActivity, "Channel", "TV")+`"]`))}

	_, locked := f.service.UpdateAttribute(t.Context(), alex, channel.ID, ShellAttributeUpdate{Type: text("singleChoice"), Level: text("lineItem"),
		Version: version(1)})
	f.refs = nil
	changed, err := f.service.UpdateAttribute(t.Context(), alex, channel.ID, ShellAttributeUpdate{Type: text("singleChoice"), Version: version(1)})

	assertError(t, locked, problems(
		"level", "Level can't change once records use this attribute — create a new attribute instead.",
		"type", "Type can't change once records use this attribute — create a new attribute instead."))
	if err != nil || changed.Type != TypeSingleChoice || changed.OptionCount != 2 {
		t.Errorf("without values: %+v, %v", changed, err)
	}
}

// A choice attribute that gets a type without options loses its options for good, after a warning, and the
// log says which (FR-021, D-049).
func TestUpdateAttribute_TypeChangeRemovesOptions_FR021(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	channel := shellForm(t, f).attribute(LevelActivity, "Channel")
	update := ShellAttributeUpdate{Type: text("text"), Version: version(1)}

	impact, err := f.service.AttributeImpact(t.Context(), alex, channel.ID, update)
	if err != nil || !reflect.DeepEqual(impact.Warnings, []ImpactWarning{{Kind: ImpactOptionsRemoved, Message: "Its 2 options will be removed: TV, Radio.", Count: 2}}) {
		t.Errorf("impact = %+v, %v", impact, err)
	}
	changed, err := f.service.UpdateAttribute(t.Context(), alex, channel.ID, update)

	if err != nil || changed.Type != TypeText || changed.OptionCount != 0 {
		t.Fatalf("changed = %+v, %v", changed, err)
	}
	if got := f.optionLabels(t, channel.ID); len(got) != 0 {
		t.Errorf("options left = %q", got)
	}
	if entry := f.lastEntry(t); entry.Details != "Removed its 2 options: TV, Radio." || entry.Changes[0].Field != "type" {
		t.Errorf("entry = %+v", entry)
	}
}

// An attribute that comes to depend on another links every active option at once — each unlinked one is
// listed — and each option whose links changed is logged; depending on nothing again removes the links
// (FR-022).
func TestUpdateAttribute_BecomeDependentLinksAllOptions_FR022(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	tf := shellForm(t, f)
	channel, cluster := tf.attribute(LevelActivity, "Channel"), tf.attribute(LevelActivity, "Cluster")
	tv, radio := tf.option(LevelActivity, "Channel", "TV"), tf.option(LevelActivity, "Channel", "Radio")
	china, india := tf.option(LevelActivity, "Cluster", "Retail – China"), tf.option(LevelActivity, "Cluster", "Retail – India")

	_, unlinked := f.service.UpdateAttribute(t.Context(), alex, channel.ID, ShellAttributeUpdate{ParentAttributeID: &cluster.ID, Version: version(1)})
	dependent, err := f.service.UpdateAttribute(t.Context(), alex, channel.ID, ShellAttributeUpdate{ParentAttributeID: &cluster.ID,
		OptionLinks: &[]OptionLinks{{OptionID: tv, ParentOptionIDs: []string{china}}, {OptionID: radio, ParentOptionIDs: []string{china, india}}},
		Version:     version(1)})

	assertError(t, unlinked, problems("optionLinks."+tv, "Link 'TV' to at least one cluster option.",
		"optionLinks."+radio, "Link 'Radio' to at least one cluster option."))
	if err != nil || dependent.ParentAttribute == nil || *dependent.ParentAttribute != (SettingRef{ID: cluster.ID, Name: "Cluster", Active: true}) {
		t.Fatalf("dependent = %+v, %v", dependent, err)
	}
	options, _ := f.service.Options(t.Context(), channel.ID)
	if !slices.Equal(options[0].ParentOptionIDs, []string{china}) || !slices.Equal(options[1].ParentOptionIDs, []string{china, india}) || options[0].Version != 2 {
		t.Errorf("options = %+v", options)
	}
	entries := f.logEntries(t)
	if len(entries) != 3 || entries[1].RecordType != "shellAttributeOption" || entries[1].Changes[0].Field != "parentOptionIds" ||
		entries[1].Changes[0].NewText != "Retail – China" {
		t.Errorf("log = %+v", entries)
	}

	independent, err := f.service.UpdateAttribute(t.Context(), alex, channel.ID, ShellAttributeUpdate{ParentAttributeID: text(""), Version: version(2)})
	options, _ = f.service.Options(t.Context(), channel.ID)
	if err != nil || independent.ParentAttribute != nil || len(options[0].ParentOptionIDs) != 0 || len(options[1].ParentOptionIDs) != 0 ||
		len(f.logEntries(t)) != 6 {
		t.Errorf("independent = %+v, %v; options %+v", independent, err, options)
	}
}

// Before a marking becomes needed, the warning counts the live records of the level that have no value
// and would show as incomplete — not those in the recycle bin, those where the attribute is hidden by its
// condition, nor a dependent without options for the chosen parent (US2-5, FR-021, D-034).
func TestAttributeImpact_MarkingMakesIncomplete_FR021(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	tf := shellForm(t, f)
	region, brands := tf.attribute(LevelActivity, "Region"), tf.attribute(LevelActivity, "Brands")
	cluster := tf.attribute(LevelActivity, "Cluster").ID
	china, india := `"`+tf.option(LevelActivity, "Cluster", "Retail – China")+`"`, `"`+tf.option(LevelActivity, "Cluster", "Retail – India")+`"`
	unit := tf.attribute(LevelPlan, "Business unit").ID
	f.refs = []Reference{
		planRef("P-1001", "Lubricants FY2026", "", false, values(unit, `["`+tf.option(LevelPlan, "Business unit", "Lubricants")+`"]`)),
		planRef("P-1002", "Mobility FY2026", "", false, values(unit, `["`+tf.option(LevelPlan, "Business unit", "Mobility")+`"]`)),
	}
	for i := range 7 { // no Region yet: counted
		v := values(cluster, china)
		if i >= 5 {
			v = nil
		}
		f.refs = append(f.refs, activityRef(fmt.Sprintf("A-10%02d", i), "P-1001", "", false, v))
	}
	f.refs = append(f.refs,
		activityRef("A-1100", "P-1001", "", false, values(cluster, india)), // no region for this cluster (D-034)
		activityRef("A-1101", "P-1001", "", false, values(cluster, china, region.ID, `"`+tf.option(LevelActivity, "Region", "RETAIL – CHINA")+`"`)),
		activityRef("A-1102", "P-1001", "", true, nil), // in the recycle bin
		// Mobility: Brands is hidden; it has a Region, so it counts for neither attribute.
		activityRef("A-1103", "P-1002", "", false, values(cluster, china, region.ID, `"`+tf.option(LevelActivity, "Region", "RETAIL – CHINA")+`"`)),
	)
	needed := ShellAttributeUpdate{Marking: text("neededToSubmit"), Version: version(1)}

	regionImpact, err := f.service.AttributeImpact(t.Context(), alex, region.ID, needed)
	brandsImpact, err2 := f.service.AttributeImpact(t.Context(), alex, brands.ID, needed)
	unchanged, err3 := f.service.AttributeImpact(t.Context(), alex, region.ID, ShellAttributeUpdate{Label: text("Area"), Version: version(1)})

	want := []ImpactWarning{{Kind: ImpactBecomesIncomplete, Count: 7,
		Message: "7 activities have no value yet and will show as incomplete. Submitted and approved activities keep their status."}}
	if err != nil || !reflect.DeepEqual(regionImpact.Warnings, want) {
		t.Errorf("Region impact = %+v, %v", regionImpact, err)
	}
	if err2 != nil || len(brandsImpact.Warnings) != 1 || brandsImpact.Warnings[0].Count != 9 {
		t.Errorf("Brands impact = %+v, %v (want the 9 live Lubricants activities)", brandsImpact, err2)
	}
	if err3 != nil || len(unchanged.Warnings) != 0 {
		t.Errorf("a rename = %+v, %v", unchanged, err3)
	}
	if stored, _, _ := f.storedAttribute(t, region.ID); stored.Marking != MarkingOptional {
		t.Errorf("the impact check saved the marking")
	}
}

// The Business unit attribute cannot get another type or section, be set inactive, deleted or moved,
// because conditions depend on it; it can be renamed (FR-023, D-055).
func TestBusinessUnit_Locked_FR023(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	unit := shellForm(t, f).attribute(LevelPlan, "Business unit")
	const reason = "Business unit can't be deleted, set inactive, moved or given another type, because other attributes are shown or hidden by it."

	_, typed := f.service.UpdateAttribute(t.Context(), alex, unit.ID, ShellAttributeUpdate{Type: text("singleChoice"), Section: text("Other"),
		Version: version(1)})
	_, inactive := f.service.SetAttributeActive(t.Context(), alex, unit.ID, StatusUpdate{Active: yes(false), Version: version(1)})
	_, deleted := f.service.DeleteAttribute(t.Context(), alex, unit.ID)
	_, moved := f.service.MoveAttribute(t.Context(), alex, unit.ID, MoveInput{What: MoveAttribute, Direction: MoveDown, Version: version(1)})
	renamed, err := f.service.UpdateAttribute(t.Context(), alex, unit.ID, ShellAttributeUpdate{Label: text("Business units"), Version: version(1)})

	assertError(t, typed, problems("type", reason, "section", reason))
	assertError(t, inactive, problems("active", reason))
	assertError(t, deleted, errs.InUse(reason))
	assertError(t, moved, problems("direction", reason))
	if err != nil || renamed.Label != "Business units" || !renamed.BusinessUnit {
		t.Errorf("renamed = %+v, %v", renamed, err)
	}
}

// An attribute that an active attribute depends on cannot be set inactive, and a dependent cannot be
// reactivated while its parent is inactive (FR-004, FR-023, US2-7).
func TestSetAttributeInactive_ParentWithActiveDependentRefused_FR004(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	tf := shellForm(t, f)
	cluster, region := tf.attribute(LevelActivity, "Cluster"), tf.attribute(LevelActivity, "Region")

	_, parentFirst := f.service.SetAttributeActive(t.Context(), alex, cluster.ID, StatusUpdate{Active: yes(false), Version: version(1)})
	regionOff, err1 := f.service.SetAttributeActive(t.Context(), alex, region.ID, StatusUpdate{Active: yes(false), Version: version(1)})
	clusterOff, err2 := f.service.SetAttributeActive(t.Context(), alex, cluster.ID, StatusUpdate{Active: yes(false), Version: version(1)})
	_, dependentFirst := f.service.SetAttributeActive(t.Context(), alex, region.ID, StatusUpdate{Active: yes(true), Version: version(2)})

	assertError(t, parentFirst, problems("active", "Set 'Region' inactive first — it depends on 'Cluster'."))
	if err1 != nil || err2 != nil || regionOff.Active || clusterOff.Active || clusterOff.Version != 2 {
		t.Errorf("Region %+v, %v; Cluster %+v, %v", regionOff, err1, clusterOff, err2)
	}
	assertError(t, dependentFirst, problems("active", "Reactivate 'Cluster' first — 'Region' depends on it."))
	if entry := f.lastEntry(t); entry.Verb != "deactivated" || entry.RecordName != "Cluster" {
		t.Errorf("last entry = %+v", entry)
	}
}

// An attribute counts the records with a value, those in the recycle bin too, and the attributes depending
// on it; each option counts its own records (US2-3, FR-003, FR-020).
func TestAttributeUsage_RecordsAndBin_FR003_FR020(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	tf := shellForm(t, f)
	channel, cluster := tf.attribute(LevelActivity, "Channel"), tf.attribute(LevelActivity, "Cluster")
	tv, radio := tf.option(LevelActivity, "Channel", "TV"), tf.option(LevelActivity, "Channel", "Radio")
	for i := range 12 {
		value := `["` + tv + `"]`
		if i < 2 {
			value = `["` + tv + `","` + radio + `"]`
		}
		f.refs = append(f.refs, activityRef(fmt.Sprintf("A-10%02d", i), "P-1001", "", i == 11, values(channel.ID, value)))
	}

	view, err := f.service.GetAttribute(t.Context(), alex, channel.ID)
	parent, err2 := f.service.GetAttribute(t.Context(), alex, cluster.ID)
	options, err3 := f.service.ListOptions(t.Context(), alex, channel.ID, OptionFilter{Limit: 100})

	want := Usage{InUse: true, Summary: "Used by 12 activities (1 in the recycle bin)", Records: []history.RecordRef{},
		Parts: []UsagePart{{Kind: "activity", Text: "Used by 12 activities (1 in the recycle bin)", Count: 12, InRecycleBin: 1}}}
	if err != nil || !reflect.DeepEqual(view.Usage, want) {
		t.Errorf("Channel usage = %+v, %v", view.Usage, err)
	}
	if err2 != nil || parent.Usage.Summary != "Region depends on it" || parent.Usage.Parts[0].Kind != "dependentAttribute" {
		t.Errorf("Cluster usage = %+v, %v", parent.Usage, err2)
	}
	if err3 != nil || options.Items[0].Usage.Summary != "Used by 12 activities (1 in the recycle bin)" ||
		options.Items[1].Usage.Summary != "Used by 2 activities" {
		t.Errorf("option usage = %+v, %v", options, err3)
	}
}

// An attribute in use cannot be deleted: the answer says where it is used and to set it inactive instead
// (US2-7, FR-004).
func TestDeleteAttribute_InUseRefused_FR004(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	tf := shellForm(t, f)
	channel, cluster := tf.attribute(LevelActivity, "Channel"), tf.attribute(LevelActivity, "Cluster")
	f.refs = []Reference{activityRef("A-1001", "P-1001", "", false, values(channel.ID, `["`+tf.option(LevelActivity, "Channel", "TV")+`"]`))}

	_, used := f.service.DeleteAttribute(t.Context(), alex, channel.ID)
	_, parent := f.service.DeleteAttribute(t.Context(), alex, cluster.ID)
	_, gone := f.service.DeleteAttribute(t.Context(), alex, "SA-9999")

	assertError(t, used, errs.InUse("'Channel' is in use (used by 1 activity) — set it inactive instead."))
	assertError(t, parent, errs.InUse("'Cluster' is in use (region depends on it) — set it inactive instead."))
	assertError(t, gone, errs.NotFound("This Shell attribute no longer exists — someone may have deleted it."))
}

// An unused attribute moves to the recycle bin with its options, and Undo brings them all back exactly
// (US2-7, FR-004).
func TestDeleteAttribute_WithOptions_Undo_FR004(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	channel := shellForm(t, f).attribute(LevelActivity, "Channel")
	options, _ := f.service.Options(t.Context(), channel.ID)

	entry, err := f.service.DeleteAttribute(t.Context(), alex, channel.ID)
	if err != nil {
		t.Fatalf("DeleteAttribute: %v", err)
	}
	if !reflect.DeepEqual(entry.Contents, []history.ContentCount{{RecordType: "shellAttributeOption", Label: "options", Count: 2}}) {
		t.Errorf("contents = %+v", entry.Contents)
	}
	if _, gone := f.service.GetAttribute(t.Context(), alex, channel.ID); gone == nil || len(f.optionLabels(t, channel.ID)) != 0 {
		t.Errorf("after the delete: %v, options %q", gone, f.optionLabels(t, channel.ID))
	}
	if _, err := f.history.Undo(t.Context(), alex, entry.ID); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	back, found, _ := f.storedAttribute(t, channel.ID)
	backOptions, _ := f.service.Options(t.Context(), channel.ID)
	if !found || !reflect.DeepEqual(back, channel) || !reflect.DeepEqual(backOptions, options) {
		t.Errorf("after Undo = %+v with %+v", back, backOptions)
	}
}
