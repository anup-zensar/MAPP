package settings

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

// addOptions stores n options "<prefix><number>" (numbered from 1, four digits) for the attribute
// attributeID directly, as an import would.
func (f *fixture) addOptions(t *testing.T, attributeID, prefix string, n int) {
	t.Helper()
	f.write(t, func(ctx context.Context, repo Store) error {
		for i := 1; i <= n; i++ {
			o := Option{ID: f.next(ctx, ids.PrefixShellOption), AttributeID: attributeID, Label: fmt.Sprintf("%s%04d", prefix, i), Position: i,
				ParentOptionIDs: []string{}, Active: true, Meta: record.NewMeta(actor.System, start)}
			if err := repo.PutOption(ctx, o); err != nil {
				return err
			}
		}
		return nil
	})
}

// An option is added last with a label no other option of the attribute has, ignoring letter case; only a
// choice attribute has options (FR-018).
func TestAddOption_UniqueLabel_FR018(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	tf := shellForm(t, f)
	tf = f.addAttributes(t, testAttribute{level: LevelActivity, label: "Notes", typ: TypeText})
	channel := tf.attribute(LevelActivity, "Channel")

	_, taken := f.service.CreateOption(t.Context(), alex, channel.ID, OptionInput{Label: " tv "})
	_, empty := f.service.CreateOption(t.Context(), alex, channel.ID, OptionInput{Label: " "})
	_, text := f.service.CreateOption(t.Context(), alex, tf.attribute(LevelActivity, "Notes").ID, OptionInput{Label: "Online"})
	_, gone := f.service.CreateOption(t.Context(), alex, "SA-9999", OptionInput{Label: "Online"})
	added, err := f.service.CreateOption(t.Context(), alex, channel.ID, OptionInput{Label: " Online "})

	assertError(t, taken, problems("label", "'TV' is already an option of Channel."))
	assertError(t, empty, problems("label", "Enter a label."))
	assertError(t, text, problems("label", "Notes has no options — its type is Text."))
	assertError(t, gone, errs.NotFound("This Shell attribute no longer exists — someone may have deleted it."))
	if err != nil || added.Label != "Online" || added.Position != 3 || !added.Active || added.AttributeID != channel.ID {
		t.Errorf("added = %+v, %v", added, err)
	}
	entry := f.lastEntry(t)
	if entry.Action != history.ActionCreated || entry.RecordType != "shellAttributeOption" || entry.RecordName != "Online" ||
		len(entry.Path) != 1 || entry.Path[0].Name != "Channel" {
		t.Errorf("entry = %+v", entry)
	}
}

// An option of a dependent attribute is linked to at least one option of the parent, newly linked ones
// active (FR-022).
func TestAddOption_DependentNeedsActiveParent_FR022(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	tf := shellForm(t, f)
	region := tf.attribute(LevelActivity, "Region")
	india, closed := tf.option(LevelActivity, "Cluster", "Retail – India"), tf.option(LevelActivity, "Cluster", "Closed")

	_, unlinked := f.service.CreateOption(t.Context(), alex, region.ID, OptionInput{Label: "RETAIL – INDIA"})
	_, inactive := f.service.CreateOption(t.Context(), alex, region.ID, OptionInput{Label: "RETAIL – INDIA", ParentOptionIDs: []string{closed}})
	added, err := f.service.CreateOption(t.Context(), alex, region.ID, OptionInput{Label: "RETAIL – INDIA", ParentOptionIDs: []string{india}})

	assertError(t, unlinked, problems("parentOptionIds", "Link 'RETAIL – INDIA' to at least one cluster option."))
	assertError(t, inactive, problems("parentOptionIds", "'Closed' is inactive — link active cluster options only."))
	if err != nil || !reflect.DeepEqual(added.ParentOptions, []SettingRef{{ID: india, Name: "Retail – India", Active: true}}) {
		t.Errorf("added = %+v, %v", added, err)
	}
}

// A rename keeps the option's place and is logged with the old and new label; another option's label is
// refused; links may stay on a parent option that became inactive (US2-4, FR-018).
func TestRenameOption_FR018(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	tf := shellForm(t, f)
	tv := tf.option(LevelActivity, "Channel", "TV")

	renamed, err := f.service.UpdateOption(t.Context(), alex, tv, OptionUpdate{Label: text("Television"), Version: version(1)})
	_, taken := f.service.UpdateOption(t.Context(), alex, tv, OptionUpdate{Label: text("radio"), Version: version(2)})
	_, stale := f.service.UpdateOption(t.Context(), alex, tv, OptionUpdate{Label: text("TV"), Version: version(1)})
	_, gone := f.service.UpdateOption(t.Context(), alex, "SO-9999", OptionUpdate{Label: text("TV"), Version: version(1)})

	if err != nil || renamed.Label != "Television" || renamed.Position != 1 || renamed.Version != 2 {
		t.Errorf("renamed = %+v, %v", renamed, err)
	}
	if entry := f.lastEntry(t); entry.RecordName != "Television" || entry.Changes[0].Field != "label" || entry.Changes[0].OldText != "TV" {
		t.Errorf("entry = %+v", entry)
	}
	assertError(t, taken, problems("label", "'Radio' is already an option of Channel."))
	assertError(t, stale, errs.VersionConflict("option", "Alex Morgan", start))
	assertError(t, gone, errs.NotFound("This option no longer exists — someone may have deleted it."))
}

// An option swaps places with its neighbour; both are logged; the ends go no further (FR-018).
func TestMoveOption_FR018(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	tf := shellForm(t, f)
	tv, radio := tf.option(LevelActivity, "Channel", "TV"), tf.option(LevelActivity, "Channel", "Radio")

	_, first := f.service.MoveOption(t.Context(), alex, tv, OptionMoveInput{Direction: MoveUp, Version: version(1)})
	moved, err := f.service.MoveOption(t.Context(), alex, radio, OptionMoveInput{Direction: MoveUp, Version: version(1)})
	_, last := f.service.MoveOption(t.Context(), alex, tv, OptionMoveInput{Direction: MoveDown, Version: version(2)})

	assertError(t, first, problems("direction", "'TV' is already first."))
	if err != nil || moved.Position != 1 || moved.Version != 2 {
		t.Errorf("moved = %+v, %v", moved, err)
	}
	if got := f.optionLabels(t, tf.attribute(LevelActivity, "Channel").ID); !slices.Equal(got, []string{"Radio", "TV"}) {
		t.Errorf("options = %q", got)
	}
	assertError(t, last, problems("direction", "'TV' is already last."))
	if entries := f.logEntries(t); len(entries) != 2 || entries[0].Changes[0].Field != "position" {
		t.Errorf("log = %+v", entries)
	}
}

// An option is set inactive and reactivated, each logged; the same status again changes nothing (FR-018).
func TestSetOptionActive_FR018(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	tv := shellForm(t, f).option(LevelActivity, "Channel", "TV")

	inactive, err := f.service.SetOptionActive(t.Context(), alex, tv, StatusUpdate{Active: yes(false), Version: version(1)})
	again, err2 := f.service.SetOptionActive(t.Context(), alex, tv, StatusUpdate{Active: yes(false), Version: version(2)})
	active, err3 := f.service.SetOptionActive(t.Context(), alex, tv, StatusUpdate{Active: yes(true), Version: version(2)})

	if err != nil || inactive.Active || err2 != nil || again.Version != 2 || err3 != nil || !active.Active || active.Version != 3 {
		t.Errorf("inactive %+v %v; again %+v %v; active %+v %v", inactive, err, again, err2, active, err3)
	}
	entries := f.logEntries(t)
	if len(entries) != 2 || entries[0].Verb != "deactivated" || entries[1].Verb != "reactivated" {
		t.Errorf("log = %+v", entries)
	}
}

// An option in use — a value on a record, a linked option of a dependent attribute, a condition — cannot be
// deleted: the answer says where it is used (FR-004, FR-018).
func TestDeleteOption_InUseRefused_FR004(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	tf := shellForm(t, f)
	tv := tf.option(LevelActivity, "Channel", "TV")
	f.refs = []Reference{activityRef("A-1001", "P-1001", "", true, values(tf.attribute(LevelActivity, "Channel").ID, `["`+tv+`"]`))}

	_, value := f.service.DeleteOption(t.Context(), alex, tv)
	_, linked := f.service.DeleteOption(t.Context(), alex, tf.option(LevelActivity, "Cluster", "Retail – China"))
	_, condition := f.service.DeleteOption(t.Context(), alex, tf.option(LevelPlan, "Business unit", "Lubricants"))

	assertError(t, value, errs.InUse("'TV' is in use (used by 1 activity (1 in the recycle bin)) — set it inactive instead."))
	assertError(t, linked, errs.InUse("'Retail – China' is in use (linked to 1 Region option) — set it inactive instead."))
	assertError(t, condition, errs.InUse("'Lubricants' is in use (condition of 1 attribute) — set it inactive instead."))
}

// The last option of a choice attribute cannot be deleted (FR-018).
func TestDeleteOption_LastOptionRefused_FR018(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	tf := f.addAttributes(t, testAttribute{level: LevelActivity, label: "Solo", typ: TypeSingleChoice, options: []string{"Only"}})

	_, err := f.service.DeleteOption(t.Context(), alex, tf.option(LevelActivity, "Solo", "Only"))

	assertError(t, err, errs.InUse("'Solo' needs at least one option, so 'Only' can't be deleted."))
}

// The last option carries the delete lock with the same reason, so the web app can say why without
// sending the request; a second option lifts it from both (D-211, design system 6.6). GetOption answers
// the option as stored now, or not found.
func TestOptionLocks_LastOptionCantBeDeleted_D211(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	tf := f.addAttributes(t, testAttribute{level: LevelActivity, label: "Solo", typ: TypeSingleChoice, options: []string{"Only"}})
	only := tf.option(LevelActivity, "Solo", "Only")
	lock := []Lock{{Field: "delete", Reason: "'Solo' needs at least one option, so 'Only' can't be deleted."}}

	alone, err := f.service.GetOption(t.Context(), alex, only)
	if err != nil || !reflect.DeepEqual(alone.Locks, lock) {
		t.Fatalf("GetOption = %+v, %v; want the delete lock", alone.Locks, err)
	}
	second, err := f.service.CreateOption(t.Context(), alex, tf.attribute(LevelActivity, "Solo").ID, OptionInput{Label: "Another"})
	if err != nil || second.Locks == nil || len(second.Locks) != 0 {
		t.Fatalf("CreateOption = %+v, %v; want no locks", second.Locks, err)
	}
	if now, err := f.service.GetOption(t.Context(), alex, only); err != nil || len(now.Locks) != 0 {
		t.Errorf("GetOption after a second option = %+v, %v; want no locks", now.Locks, err)
	}
	_, gone := f.service.GetOption(t.Context(), alex, "SO-99999")
	assertError(t, gone, errs.NotFound("This option no longer exists — someone may have deleted it."))
}

// An unused option moves to the recycle bin, under its attribute, and Undo brings it back (FR-018, FR-004).
func TestDeleteOption_Unused_Undo_FR018(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	tf := shellForm(t, f)
	channel := tf.attribute(LevelActivity, "Channel")
	radio := tf.options["activity/Channel/Radio"]

	entry, err := f.service.DeleteOption(t.Context(), alex, radio.ID)
	if err != nil || entry.RecordType != "shellAttributeOption" || entry.Parent == nil || entry.Parent.Name != "Channel" {
		t.Fatalf("DeleteOption = %+v, %v", entry, err)
	}
	if got := f.optionLabels(t, channel.ID); !slices.Equal(got, []string{"TV"}) {
		t.Errorf("after the delete = %q", got)
	}
	if _, err := f.history.Undo(t.Context(), alex, entry.ID); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if back, found, _ := f.service.Option(t.Context(), radio.ID); !found || !reflect.DeepEqual(back, radio) {
		t.Errorf("after Undo = %+v", back)
	}
}

// Search finds an option by any part of its label among 1,217, ignoring letter case and surrounding
// spaces (SC-007, FR-018).
func TestListOptions_SearchAnyPart_SC007(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	tf := f.addAttributes(t, testAttribute{level: LevelActivity, label: "Cost centre", typ: TypeSingleChoice})
	costCentre := tf.attribute(LevelActivity, "Cost centre")
	f.addOptions(t, costCentre.ID, "CN0400", 1217)

	found, err := f.service.ListOptions(t.Context(), alex, costCentre.ID, OptionFilter{Q: " 0220 ", Limit: 100})
	lower, err2 := f.service.ListOptions(t.Context(), alex, costCentre.ID, OptionFilter{Q: "cn04001", Limit: 100})

	if err != nil || found.Total != 1 || len(found.Items) != 1 || found.Items[0].Label != "CN04000220" {
		t.Errorf("search 0220 = %+v, %v", found, err)
	}
	if err2 != nil || lower.Total != 218 || len(lower.Items) != 100 {
		t.Errorf("search cn04001: total %d, %d items, %v", lower.Total, len(lower.Items), err2)
	}
}

// An attribute's options come in pages of at most 500 (FR-018).
func TestListOptions_Paged_FR018(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	tf := f.addAttributes(t, testAttribute{level: LevelActivity, label: "Cost centre", typ: TypeSingleChoice})
	costCentre := tf.attribute(LevelActivity, "Cost centre")
	f.addOptions(t, costCentre.ID, "CC", 60)

	page, err := f.service.ListOptions(t.Context(), alex, costCentre.ID, OptionFilter{Limit: 25, Offset: 50})
	_, wrong := f.service.ListOptions(t.Context(), alex, costCentre.ID, OptionFilter{Limit: 501, Offset: -1, Status: "some"})

	if err != nil || page.Total != 60 || len(page.Items) != 10 || page.Items[0].Label != "CC0051" {
		t.Errorf("last page: total %d, %d items, %v", page.Total, len(page.Items), err)
	}
	assertError(t, wrong, problems("limit", "Choose a number from 1 to 500.", "offset", "Choose 0 or more.", "status", "Choose active, inactive or all."))
}

// Each option has its own usage: the records whose value has it, those in the recycle bin too (US2-3,
// FR-020).
func TestOptionUsage_PerOption_FR020(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	tf := shellForm(t, f)
	channel := tf.attribute(LevelActivity, "Channel")
	tv := tf.option(LevelActivity, "Channel", "TV")
	f.refs = []Reference{
		activityRef("A-1001", "P-1001", "", false, values(channel.ID, `["`+tv+`"]`)),
		activityRef("A-1002", "P-1001", "", true, values(channel.ID, `["`+tv+`"]`)),
	}

	page, err := f.service.ListOptions(t.Context(), alex, channel.ID, OptionFilter{Limit: 100})

	want := Usage{InUse: true, Summary: "Used by 2 activities (1 in the recycle bin)", Records: []history.RecordRef{},
		Parts: []UsagePart{{Kind: "activity", Text: "Used by 2 activities (1 in the recycle bin)", Count: 2, InRecycleBin: 1}}}
	if err != nil || !reflect.DeepEqual(page.Items[0].Usage, want) || page.Items[1].Usage.InUse {
		t.Errorf("usage = %+v, %v", page.Items, err)
	}
}
