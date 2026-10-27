package settings

import (
	"cmp"
	"context"
	"slices"
	"testing"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

// testAttribute describes a Shell attribute for addAttributes. parent is the label of an earlier attribute
// of the same level; condition lists Business unit option labels; links gives each option's parent option
// labels.
type testAttribute struct {
	level           Level
	label           string
	typ             AttributeType
	marking         Marking
	section         string
	parent          string
	condition       []string
	role            string
	inactive        bool
	options         []string
	links           map[string][]string
	inactiveOptions []string
	help            string // its help text (016 FR-048)
}

// testForm is what addAttributes stored: attributes by "<level>/<label>", options by
// "<level>/<attribute>/<option>".
type testForm struct {
	attributes map[string]Attribute
	options    map[string]Option
}

// attribute returns the attribute label of level.
func (tf testForm) attribute(level Level, label string) Attribute {
	return tf.attributes[string(level)+"/"+label]
}

// option returns the id of the option of the attribute label of level.
func (tf testForm) option(level Level, label, option string) string {
	return tf.options[string(level)+"/"+label+"/"+option].ID
}

// addAttributes stores Shell attributes with their options directly, positions 1… per level in the order
// given, and marks the plan-level "Business unit" with the business-unit role.
func (f *fixture) addAttributes(t *testing.T, list ...testAttribute) testForm {
	t.Helper()
	tf := testForm{attributes: map[string]Attribute{}, options: map[string]Option{}}
	f.write(t, func(ctx context.Context, repo Store) error {
		existing, err := repo.ListAttributes(ctx)
		if err != nil {
			return err
		}
		positions := map[Level]int{}
		for _, a := range existing {
			positions[a.Level] = max(positions[a.Level], a.Position)
			tf.attributes[string(a.Level)+"/"+a.Label] = a
		}
		options, err := repo.ListOptions(ctx)
		if err != nil {
			return err
		}
		for _, o := range options {
			for _, a := range existing {
				if a.ID == o.AttributeID {
					tf.options[string(a.Level)+"/"+a.Label+"/"+o.Label] = o
				}
			}
		}
		for _, ta := range list {
			positions[ta.level]++
			a := Attribute{
				ID: f.next(ctx, ids.PrefixShellAttribute), Label: ta.label, HelpText: ta.help, Level: ta.level, Type: ta.typ,
				Marking: cmp.Or(ta.marking, MarkingOptional), Section: cmp.Or(ta.section, DefaultSection),
				Position: positions[ta.level], Condition: []string{}, Role: ta.role, Active: !ta.inactive,
				Meta: record.NewMeta(actor.System, start),
			}
			if ta.level == LevelPlan && ta.label == "Business unit" {
				a.Role = RoleBusinessUnit
			}
			if ta.parent != "" {
				a.ParentAttributeID = tf.attribute(ta.level, ta.parent).ID
			}
			for _, unit := range ta.condition {
				a.Condition = append(a.Condition, tf.option(LevelPlan, "Business unit", unit))
			}
			if err := repo.PutAttribute(ctx, a); err != nil {
				return err
			}
			tf.attributes[string(a.Level)+"/"+a.Label] = a
			for i, label := range ta.options {
				o := Option{
					ID: f.next(ctx, ids.PrefixShellOption), AttributeID: a.ID, Label: label, Position: i + 1,
					ParentOptionIDs: []string{}, Active: !slices.Contains(ta.inactiveOptions, label), Meta: record.NewMeta(actor.System, start),
				}
				for _, parent := range ta.links[label] {
					o.ParentOptionIDs = append(o.ParentOptionIDs, tf.option(ta.level, ta.parent, parent))
				}
				if err := repo.PutOption(ctx, o); err != nil {
					return err
				}
				tf.options[string(a.Level)+"/"+a.Label+"/"+label] = o
			}
		}
		return nil
	})
	return tf
}

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

// activityLabels returns the labels of the activity attributes in display order.
func (f *fixture) activityLabels(t *testing.T) []string {
	t.Helper()
	level := LevelActivity
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
