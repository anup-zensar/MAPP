package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"northstar/internal/history"
	"northstar/internal/platform/errs"
)

// shellForm stores the attributes most attribute tests start from: the plan's Business unit (Lubricants,
// Mobility) and, at activity level, Cluster (Retail – China, Retail – India and the inactive Closed),
// Region depending on it (RETAIL – CHINA, linked to Retail – China; none for Retail – India), Channel (TV,
// Radio), and Brands in the section "Lubricants attributes", shown when the business unit includes
// Lubricants. Activity positions: Cluster 1, Region 2, Channel 3, Brands 4.
func shellForm(t *testing.T, f *fixture) testForm {
	t.Helper()
	return f.addAttributes(t,
		testAttribute{level: LevelPlan, label: "Business unit", typ: TypeMultipleChoice, options: []string{"Lubricants", "Mobility"}},
		testAttribute{level: LevelActivity, label: "Cluster", typ: TypeSingleChoice, options: []string{"Retail – China", "Retail – India", "Closed"},
			inactiveOptions: []string{"Closed"}},
		testAttribute{level: LevelActivity, label: "Region", typ: TypeSingleChoice, parent: "Cluster", options: []string{"RETAIL – CHINA"},
			links: map[string][]string{"RETAIL – CHINA": {"Retail – China"}}},
		testAttribute{level: LevelActivity, label: "Channel", typ: TypeMultipleChoice, options: []string{"TV", "Radio"}},
		testAttribute{level: LevelActivity, label: "Brands", typ: TypeMultipleChoice, section: "Lubricants attributes", condition: []string{"Lubricants"},
			options: []string{"Castrol", "Shell Helix"}},
	)
}

// mediaChannel is the attribute of US2-1: an activity single choice with four options, needed to submit.
func mediaChannel() ShellAttributeInput {
	return ShellAttributeInput{
		Label: "Media channel", Level: "activity", Type: "singleChoice", Marking: "neededToSubmit", Section: "Shell attributes",
		Options: []NewOption{{Label: "TV"}, {Label: "Radio"}, {Label: "Print"}, {Label: "Online"}},
	}
}

// createAttribute creates an attribute as Alex and stops the test when that fails.
func (f *fixture) createAttribute(t *testing.T, input ShellAttributeInput) ShellAttributeView {
	t.Helper()
	view, err := f.service.CreateAttribute(t.Context(), alex, input)
	if err != nil {
		t.Fatalf("CreateAttribute(%q): %v", input.Label, err)
	}
	return view
}

// storedAttribute reads the attribute id as it is stored.
func (f *fixture) storedAttribute(t *testing.T, id string) (Attribute, bool, error) {
	t.Helper()
	var a Attribute
	var found bool
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		a, found, err = f.service.repo.GetAttribute(ctx, id)
		return err
	})
	return a, found, err
}

// labelsOf returns the labels of the attributes of level in display order.
func (f *fixture) labelsOf(t *testing.T, level Level) []string {
	t.Helper()
	listed, err := f.service.ListAttributes(t.Context(), alex, &level)
	if err != nil {
		t.Fatalf("ListAttributes: %v", err)
	}
	labels := []string{}
	for _, a := range listed {
		labels = append(labels, a.Label)
	}
	return labels
}

// optionLabels returns the labels of the options of the attribute id in display order.
func (f *fixture) optionLabels(t *testing.T, id string) []string {
	t.Helper()
	options, err := f.service.Options(t.Context(), id)
	if err != nil {
		t.Fatalf("Options: %v", err)
	}
	labels := []string{}
	for _, o := range options {
		labels = append(labels, o.Label)
	}
	return labels
}

// A new attribute is listed last in its section, with its options in the order given, shows on its level's
// form with its marking, and is logged once with its options counted (US2-1, FR-019).
func TestCreateAttribute_ListedLastInSection_FR019(t *testing.T) {
	f := newTestService(t)
	shellForm(t, f)

	created := f.createAttribute(t, mediaChannel())

	if created.ID != "SA-1006" || created.Position != 4 || created.Marking != MarkingNeededToSubmit || created.OptionCount != 4 ||
		created.ActiveOptionCount != 4 || !created.Active || created.Section != "Shell attributes" {
		t.Errorf("created = %+v", created)
	}
	if got := f.labelsOf(t, LevelActivity); !slices.Equal(got, []string{"Cluster", "Region", "Channel", "Media channel", "Brands"}) {
		t.Errorf("activity attributes = %q", got)
	}
	if got := f.optionLabels(t, created.ID); !slices.Equal(got, []string{"TV", "Radio", "Print", "Online"}) {
		t.Errorf("options = %q", got)
	}
	form, err := f.service.Form(t.Context(), alex, LevelActivity)
	if err != nil {
		t.Fatalf("Form: %v", err)
	}
	shell := form.Sections[0].Attributes
	if last := shell[len(shell)-1]; last.Label != "Media channel" || last.Marking != MarkingNeededToSubmit || len(last.Options) != 4 {
		t.Errorf("last on the form = %+v", last)
	}
	entries := f.logEntries(t)
	if len(entries) != 1 || entries[0].Action != history.ActionCreated || entries[0].RecordType != "shellAttribute" ||
		entries[0].RecordName != "Media channel" || entries[0].Details != "with 4 options" {
		t.Errorf("log = %+v", entries)
	}
}

// The problems of the options come together with the other problems: none for a choice type, the same
// label twice ignoring case (once per label), an empty or too long label, and options on a type without
// any (US2-2, FR-018).
func TestCreateAttribute_OptionProblemsTogether_FR018(t *testing.T) {
	f := newTestService(t)
	shellForm(t, f)
	noOptions := mediaChannel()
	noOptions.Options = nil
	twice := mediaChannel()
	twice.Label = " "
	twice.Options = []NewOption{{Label: "TV"}, {Label: "tv"}, {Label: " "}, {Label: " TV "}, {Label: strings.Repeat("x", 121)}}
	notChoice := mediaChannel()
	notChoice.Type, notChoice.Label = "text", "Notes"

	_, none := f.service.CreateAttribute(t.Context(), alex, noOptions)
	_, together := f.service.CreateAttribute(t.Context(), alex, twice)
	_, text := f.service.CreateAttribute(t.Context(), alex, notChoice)

	assertError(t, none, problems("options", "Add at least one option."))
	assertError(t, together, problems("label", "Enter a label.", "options[1].label", "'TV' appears twice.",
		"options[2].label", "Enter a label for every option.", "options[4].label", "Use 120 characters or fewer."))
	assertError(t, text, problems("options", "Only choice attributes have options."))
	if got := f.labelsOf(t, LevelActivity); len(got) != 4 || len(f.logEntries(t)) != 0 {
		t.Errorf("after refused creates: %q and %d log entries", got, len(f.logEntries(t)))
	}
}

// A label is unique within its level, ignoring letter case and surrounding spaces, and at most 80
// characters; another level may use it (FR-016).
func TestCreateAttribute_LabelUniqueWithinLevel_FR016(t *testing.T) {
	f := newTestService(t)
	shellForm(t, f)
	sameLevel := mediaChannel()
	sameLevel.Label = " channel "
	tooLong := mediaChannel()
	tooLong.Label = strings.Repeat("é", 81)
	otherLevel := mediaChannel()
	otherLevel.Label, otherLevel.Level, otherLevel.Marking = "Channel", "campaign", "neededToPublish"

	_, taken := f.service.CreateAttribute(t.Context(), alex, sameLevel)
	_, long := f.service.CreateAttribute(t.Context(), alex, tooLong)
	created, err := f.service.CreateAttribute(t.Context(), alex, otherLevel)

	assertError(t, taken, problems("label", "The Activity form already has a Shell attribute called 'Channel'."))
	assertError(t, long, problems("label", "Use 80 characters or fewer."))
	if err != nil || created.Level != LevelCampaign || created.Label != "Channel" || created.Position != 1 {
		t.Errorf("campaign Channel = %+v, %v", created, err)
	}
}

// Help text has at most 300 characters and a section at most 60; an empty section is "Shell attributes"
// (FR-016, FR-019).
func TestCreateAttribute_HelpTextLimit_FR016(t *testing.T) {
	f := newTestService(t)
	shellForm(t, f)
	tooLong := mediaChannel()
	tooLong.HelpText, tooLong.Section = strings.Repeat("é", 301), strings.Repeat("s", 61)
	atLimit := mediaChannel()
	atLimit.HelpText, atLimit.Section = " "+strings.Repeat("é", 300)+" ", "  "

	_, long := f.service.CreateAttribute(t.Context(), alex, tooLong)
	created, err := f.service.CreateAttribute(t.Context(), alex, atLimit)

	assertError(t, long, problems("helpText", "Use 300 characters or fewer.", "section", "Use 60 characters or fewer."))
	if err != nil || created.HelpText != strings.Repeat("é", 300) || created.Section != DefaultSection {
		t.Errorf("created = %+v, %v", created, err)
	}
}

// Needed to publish is for plans, campaigns and initiatives, needed to submit for activities and line items;
// a level, type and marking must be chosen (FR-017).
func TestCreateAttribute_MarkingByLevel_FR017(t *testing.T) {
	f := newTestService(t)
	shellForm(t, f)
	plan := mediaChannel()
	plan.Level = "plan"
	lineItem := mediaChannel()
	lineItem.Level, lineItem.Marking = "lineItem", "neededToPublish"
	initiative := mediaChannel()
	initiative.Level, initiative.Marking = "initiative", "neededToPublish"

	_, planErr := f.service.CreateAttribute(t.Context(), alex, plan)
	_, lineItemErr := f.service.CreateAttribute(t.Context(), alex, lineItem)
	_, blank := f.service.CreateAttribute(t.Context(), alex, ShellAttributeInput{Label: "Blank"})
	created, err := f.service.CreateAttribute(t.Context(), alex, initiative)

	assertError(t, planErr, problems("marking", "At plan level the marking is Optional or Needed to publish."))
	assertError(t, lineItemErr, problems("marking", "At line item level the marking is Optional or Needed to submit."))
	assertError(t, blank, problems("level", "Choose a level.", "type", "Choose a type.", "marking", "Choose a marking."))
	if err != nil || created.Marking != MarkingNeededToPublish {
		t.Errorf("initiative = %+v, %v", created, err)
	}
}

// A yes/no attribute is always optional (FR-017).
func TestCreateAttribute_YesNoAlwaysOptional_FR017(t *testing.T) {
	f := newTestService(t)
	needed := ShellAttributeInput{Label: "Sponsored", Level: "activity", Type: "yesNo", Marking: "neededToSubmit"}
	optional := needed
	optional.Marking = "optional"

	_, refused := f.service.CreateAttribute(t.Context(), alex, needed)
	created, err := f.service.CreateAttribute(t.Context(), alex, optional)

	assertError(t, refused, problems("marking", "Yes/no attributes are always optional — for a needed yes-or-no answer, use a single "+
		"choice with the options Yes and No."))
	if err != nil || created.Type != TypeYesNo || created.OptionCount != 0 {
		t.Errorf("created = %+v, %v", created, err)
	}
}

// Multiple choice with % split is only for activities and line items, which carry cost (FR-017, D-029).
func TestCreateAttribute_SplitOnlyActivityAndLineItem_FR017(t *testing.T) {
	f := newTestService(t)
	campaign := ShellAttributeInput{Label: "Country split", Level: "campaign", Type: "multipleChoiceSplit", Marking: "optional",
		Options: []NewOption{{Label: "China"}, {Label: "India"}}}
	lineItem := campaign
	lineItem.Level = "lineItem"

	_, refused := f.service.CreateAttribute(t.Context(), alex, campaign)
	created, err := f.service.CreateAttribute(t.Context(), alex, lineItem)

	assertError(t, refused, problems("type", "Multiple choice with % split is only for activities and line items, which carry cost."))
	if err != nil || created.Type != TypeMultipleChoiceSplit || created.Level != LevelLineItem {
		t.Errorf("created = %+v, %v", created, err)
	}
}

// An attribute depends only on a single or multiple choice attribute of its level that comes before it,
// and only a choice attribute depends on anything (FR-022).
func TestCreateAttribute_DependentNeedsEarlierChoiceParent_FR022(t *testing.T) {
	f := newTestService(t)
	tf := shellForm(t, f)
	tf = f.addAttributes(t, testAttribute{level: LevelActivity, label: "Notes", typ: TypeText})
	china := tf.option(LevelActivity, "Cluster", "Retail – China")
	area := func(parent string) ShellAttributeInput {
		return ShellAttributeInput{Label: "Area", Level: "activity", Type: "singleChoice", Marking: "optional", ParentAttributeID: parent,
			Options: []NewOption{{Label: "North", ParentOptionIDs: []string{china}}}}
	}
	notChoice := area(tf.attribute(LevelActivity, "Cluster").ID)
	notChoice.Type, notChoice.Options = "text", nil
	chooseParent := problems("parentAttributeId", "Choose a single or multiple choice attribute that comes before this one on the Activity form.")

	_, otherLevel := f.service.CreateAttribute(t.Context(), alex, area(tf.attribute(LevelPlan, "Business unit").ID))
	_, later := f.service.CreateAttribute(t.Context(), alex, area(tf.attribute(LevelActivity, "Brands").ID))
	_, textParent := f.service.CreateAttribute(t.Context(), alex, area(tf.attribute(LevelActivity, "Notes").ID))
	_, unknown := f.service.CreateAttribute(t.Context(), alex, area("SA-9999"))
	_, text := f.service.CreateAttribute(t.Context(), alex, notChoice)
	created, err := f.service.CreateAttribute(t.Context(), alex, area(tf.attribute(LevelActivity, "Cluster").ID))

	assertError(t, otherLevel, chooseParent)
	assertError(t, later, chooseParent)
	assertError(t, textParent, chooseParent)
	assertError(t, unknown, chooseParent)
	assertError(t, text, problems("parentAttributeId", "Only choice attributes can depend on another attribute."))
	if err != nil || created.ParentAttribute == nil || created.ParentAttribute.Name != "Cluster" {
		t.Fatalf("created = %+v, %v", created, err)
	}
	if options, _ := f.service.Options(t.Context(), created.ID); len(options) != 1 || !slices.Equal(options[0].ParentOptionIDs, []string{china}) {
		t.Errorf("options = %+v", options)
	}
}

// Every option of a dependent attribute that is not linked to an option of the parent is listed at once;
// links go to the parent's options only, and to active ones (US3-2, FR-022).
func TestCreateAttribute_UnlinkedOptionsAllListed_FR022(t *testing.T) {
	f := newTestService(t)
	tf := shellForm(t, f)
	china, closed := tf.option(LevelActivity, "Cluster", "Retail – China"), tf.option(LevelActivity, "Cluster", "Closed")
	input := ShellAttributeInput{Label: "Area", Level: "activity", Type: "singleChoice", Marking: "optional",
		ParentAttributeID: tf.attribute(LevelActivity, "Cluster").ID, Options: []NewOption{
			{Label: "A"}, {Label: "B", ParentOptionIDs: []string{}}, {Label: "C", ParentOptionIDs: []string{china}},
			{Label: "D", ParentOptionIDs: []string{tf.option(LevelActivity, "Channel", "TV")}}, {Label: "E", ParentOptionIDs: []string{closed}},
		}}

	_, err := f.service.CreateAttribute(t.Context(), alex, input)

	assertError(t, err, problems(
		"options[0].parentOptionIds", "Link 'A' to at least one cluster option.",
		"options[1].parentOptionIds", "Link 'B' to at least one cluster option.",
		"options[3].parentOptionIds", "'D' can only be linked to options of cluster.",
		"options[4].parentOptionIds", "'Closed' is inactive — link active cluster options only."))
}

// A condition is made of Business unit options; without a Business unit attribute there are no conditions
// (FR-023).
func TestCreateAttribute_ConditionUsesBusinessUnitOptions_FR023(t *testing.T) {
	f := newTestService(t)
	tf := shellForm(t, f)
	lubricants := tf.option(LevelPlan, "Business unit", "Lubricants")
	input := func(condition ...string) ShellAttributeInput {
		return ShellAttributeInput{Label: "Viscosity", Level: "lineItem", Type: "text", Marking: "optional", ConditionOptionIDs: condition}
	}

	_, wrong := f.service.CreateAttribute(t.Context(), alex, input(tf.option(LevelActivity, "Cluster", "Retail – China")))
	created, err := f.service.CreateAttribute(t.Context(), alex, input(lubricants, lubricants))
	withoutUnit := newTestService(t)
	_, none := withoutUnit.service.CreateAttribute(t.Context(), alex, input("SO-1001"))

	assertError(t, wrong, problems("conditionOptionIds", "Choose business units for the condition from the Business unit options."))
	if err != nil || !reflect.DeepEqual(created.Condition, []SettingRef{{ID: lubricants, Name: "Lubricants", Active: true}}) {
		t.Errorf("created = %+v, %v", created, err)
	}
	assertError(t, none, problems("conditionOptionIds", "There is no Business unit attribute yet, so conditions can't be set."))
}

// A rename shows at once and the log keeps the old and the new label; saving without a change writes
// nothing; an out-of-date version is refused (US2-4, FR-006, FR-007).
func TestRenameAttribute_LogKeepsOldAndNew_FR006(t *testing.T) {
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
