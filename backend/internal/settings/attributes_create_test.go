package settings

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"mapp/internal/history"
)

// A new attribute is listed last in its section, with its options in the order given, shows on its level's
// form with its marking, and is logged once with its options counted (US2-1, FR-019).
func TestCreateAttribute_ListedLastInSection_FR019(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	shellForm(t, f)

	created := f.createAttribute(t, mediaChannel())

	if created.ID != "SA-1006" || created.Position != 4 || created.Marking != MarkingNeededToSubmit || created.OptionCount != 4 ||
		created.ActiveOptionCount != 4 || !created.Active || created.Section != "Shell attributes" {
		t.Errorf("created = %+v", created)
	}
	if got := f.activityLabels(t); !slices.Equal(got, []string{"Cluster", "Region", "Channel", "Media channel", "Brands"}) {
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
	t.Parallel()
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
	if got := f.activityLabels(t); len(got) != 4 || len(f.logEntries(t)) != 0 {
		t.Errorf("after refused creates: %q and %d log entries", got, len(f.logEntries(t)))
	}
}

// A label is unique within its level, ignoring letter case and surrounding spaces, and at most 80
// characters; another level may use it (FR-016).
func TestCreateAttribute_LabelUniqueWithinLevel_FR016(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
