package settings

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
)

// importFile is an import request for a file named name holding text.
func importFile(name, text string) ImportFile {
	return ImportFile{FileName: name, Content: base64.StdEncoding.EncodeToString([]byte(text))}
}

// costCentres is the file of US6: a header and 1,217 rows — 1,214 cost centres CN04000001 … CN04001214 and
// three faulty rows: an empty option (row 11), one longer than 120 characters (row 501) and CN04000220
// again (row 1,218). It returns the file and the row of CN04000220.
func costCentres() (string, int) {
	var b strings.Builder
	b.WriteString("Option\n")
	code, row220 := 0, 0
	for k := 1; k <= 1217; k++ {
		switch k {
		case 10:
			b.WriteString(" ,see the note\n")
		case 500:
			b.WriteString(strings.Repeat("9", 121) + "\n")
		case 1217:
			b.WriteString("cn04000220\n")
		default:
			code++
			fmt.Fprintf(&b, "CN04%06d\n", code)
			if code == 220 {
				row220 = k + 1 // the header is row 1
			}
		}
	}
	return b.String(), row220
}

// The check reads every row and says what importing would do, each row with its number in the file:
// 1,214 options to add, and three problems — empty, too long, twice in the file. It adds nothing (US6-1,
// FR-031).
func TestImportCheck_RowOutcomes_FR031(t *testing.T) {
	f := newTestService(t)
	tf := f.addAttributes(t, testAttribute{level: LevelLineItem, label: "Cost centre", typ: TypeSingleChoice})
	text, row220 := costCentres()

	check, err := f.service.CheckImport(t.Context(), alex, tf.attribute(LevelLineItem, "Cost centre").ID, importFile("cost-centres.csv", text))
	if err != nil {
		t.Fatalf("CheckImport: %v", err)
	}

	if check.TotalRows != 1217 || len(check.ToAdd) != 1214 || len(check.AlreadyPresent) != 0 {
		t.Errorf("totals: %d rows, %d to add, %d present", check.TotalRows, len(check.ToAdd), len(check.AlreadyPresent))
	}
	if first := check.ToAdd[0]; first != (ImportRow{Row: 2, Label: "CN04000001"}) {
		t.Errorf("first row to add = %+v", first)
	}
	wantProblems := []ImportRow{
		{Row: 11, Label: "", Message: "Empty — enter an option."},
		{Row: 501, Label: strings.Repeat("9", 121), Message: "Longer than 120 characters."},
		{Row: 1218, Label: "cn04000220", Message: fmt.Sprintf("Appears twice in the file (also row %d).", row220)},
	}
	if !reflect.DeepEqual(check.Problems, wantProblems) {
		t.Errorf("problems = %+v\nwant %+v", check.Problems, wantProblems)
	}
	if options, _ := f.service.Options(t.Context(), tf.attribute(LevelLineItem, "Cost centre").ID); len(options) != 0 || len(f.logEntries(t)) != 0 {
		t.Errorf("the check added %d options or logged", len(options))
	}
}

// The import adds only the new valid rows, in file order after the existing options, and writes one log
// entry on the attribute for all of them (US6-2, FR-031).
func TestImport_AddsOnlyNewValid_OneLogEntry_FR031(t *testing.T) {
	f := newTestService(t)
	tf := f.addAttributes(t, testAttribute{level: LevelLineItem, label: "Cost centre", typ: TypeSingleChoice, options: []string{"CN04000001", "CN04000002"}})
	attribute := tf.attribute(LevelLineItem, "Cost centre")
	file := importFile("cost-centres.csv", "Option\nCN04000004\n cn04000002 \nCN04000003\n"+strings.Repeat("x", 121)+"\n")

	result, err := f.service.Import(t.Context(), daniel, attribute.ID, file)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	if result != (ImportResult{Added: 2, AlreadyPresent: 1, Problems: 1}) {
		t.Errorf("result = %+v", result)
	}
	options, _ := f.service.Options(t.Context(), attribute.ID)
	var labels []string
	for _, o := range options {
		labels = append(labels, fmt.Sprintf("%d %s", o.Position, o.Label))
	}
	if want := []string{"1 CN04000001", "2 CN04000002", "3 CN04000004", "4 CN04000003"}; !reflect.DeepEqual(labels, want) {
		t.Errorf("options = %q, want %q", labels, want)
	}
	if added := options[2]; !added.Active || added.CreatedBy != daniel || len(added.ParentOptionIDs) != 0 || added.ParentOptionIDs == nil {
		t.Errorf("added option = %+v", added)
	}
	entries := f.logEntries(t)
	if len(entries) != 1 {
		t.Fatalf("log = %+v, want one entry", entries)
	}
	entry := entries[0]
	if entry.Action != history.ActionChanged || entry.Verb != "imported options into" || entry.RecordType != "shellAttribute" ||
		entry.RecordID != attribute.ID || entry.RecordName != "Cost centre" || entry.Actor != daniel || len(entry.Changes) != 0 ||
		entry.Details != "Added 2 options from 'cost-centres.csv'; skipped 2 rows." {
		t.Errorf("entry = %+v", entry)
	}
	if stored, err := f.service.GetAttribute(t.Context(), alex, attribute.ID); err != nil || stored.Version != attribute.Version {
		t.Errorf("the attribute's version changed: %d, %v", stored.Version, err)
	}
}

// An import never removes, renames, relinks or reactivates options: a row matching an option ignoring
// letter case — active or not — is skipped as already present (FR-031).
func TestImport_NeverRenamesOrReactivates_FR031(t *testing.T) {
	f := newTestService(t)
	tf := f.addAttributes(t, testAttribute{level: LevelActivity, label: "Media channel", typ: TypeSingleChoice,
		options: []string{"TV", "Radio", "Print"}, inactiveOptions: []string{"TV"}})
	attribute := tf.attribute(LevelActivity, "Media channel")
	file := importFile("channels.csv", "tv\nRADIO\nOnline\n")

	check, err := f.service.CheckImport(t.Context(), alex, attribute.ID, file)
	if err != nil {
		t.Fatalf("CheckImport: %v", err)
	}
	result, err := f.service.Import(t.Context(), alex, attribute.ID, file)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	wantPresent := []ImportRow{{Row: 1, Label: "tv", Message: "Already an option — skipped."}, {Row: 2, Label: "RADIO", Message: "Already an option — skipped."}}
	if !reflect.DeepEqual(check.AlreadyPresent, wantPresent) || len(check.ToAdd) != 1 || check.ToAdd[0].Label != "Online" || check.TotalRows != 3 {
		t.Errorf("check = %+v", check)
	}
	if result != (ImportResult{Added: 1, AlreadyPresent: 2, Problems: 0}) {
		t.Errorf("result = %+v", result)
	}
	options, _ := f.service.Options(t.Context(), attribute.ID)
	if len(options) != 4 || options[0].Label != "TV" || options[0].Active || options[0].Version != 1 || options[1].Label != "Radio" ||
		options[2].Label != "Print" || options[3].Label != "Online" {
		t.Errorf("options = %+v", options)
	}
}

// For a dependent attribute the second column names the parent option, found ignoring letter case among
// the active options of the parent attribute; rows with the same option and different parents become one
// option linked to each parent (FR-031, FR-022).
func TestImport_DependentParentColumn_FR031(t *testing.T) {
	f := newTestService(t)
	tf := f.addAttributes(t,
		testAttribute{level: LevelCampaign, label: "Cluster", typ: TypeMultipleChoice,
			options:         []string{"Lubricants – Europe", "Lubricants – Asia Pacific", "Retail – China", "Fleet – China"},
			inactiveOptions: []string{"Fleet – China"}},
		testAttribute{level: LevelCampaign, label: "Region", typ: TypeMultipleChoice, parent: "Cluster",
			options: []string{"RETAIL – CHINA"}, links: map[string][]string{"RETAIL – CHINA": {"Retail – China"}}},
	)
	region := tf.attribute(LevelCampaign, "Region")
	file := importFile("regions.csv", "Option;Parent option\n"+
		"LUBES – NORDICS;Lubricants – Europe\n"+
		"LUBES – NORDICS; lubricants – asia pacific\n"+
		"LUBES – IBERIA;\n"+
		"LUBES – BALTICS;Lubricants – Africa\n"+
		"lubes – nordics;LUBRICANTS – EUROPE\n"+
		"retail – china;Retail – China\n"+
		"FLEET – WEST CHINA;Fleet – China\n")

	check, err := f.service.CheckImport(t.Context(), alex, region.ID, file)
	if err != nil {
		t.Fatalf("CheckImport: %v", err)
	}
	result, err := f.service.Import(t.Context(), alex, region.ID, file)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	wantAdd := []ImportRow{{Row: 2, Label: "LUBES – NORDICS", ParentLabel: "Lubricants – Europe"}, {Row: 3, Label: "LUBES – NORDICS", ParentLabel: "lubricants – asia pacific"}}
	wantProblems := []ImportRow{
		{Row: 4, Label: "LUBES – IBERIA", Message: "Enter the parent option in the second column."},
		{Row: 5, Label: "LUBES – BALTICS", ParentLabel: "Lubricants – Africa", Message: "Parent option 'Lubricants – Africa' not found in Cluster."},
		{Row: 6, Label: "lubes – nordics", ParentLabel: "LUBRICANTS – EUROPE", Message: "Appears twice in the file (also row 2)."},
		{Row: 8, Label: "FLEET – WEST CHINA", ParentLabel: "Fleet – China", Message: "'Fleet – China' is inactive — link active cluster options only."},
	}
	wantPresent := []ImportRow{{Row: 7, Label: "retail – china", ParentLabel: "Retail – China", Message: "Already an option — skipped."}}
	if !reflect.DeepEqual(check.ToAdd, wantAdd) || !reflect.DeepEqual(check.Problems, wantProblems) || !reflect.DeepEqual(check.AlreadyPresent, wantPresent) {
		t.Errorf("check = %+v\nwant to add %+v\nproblems %+v\npresent %+v", check, wantAdd, wantProblems, wantPresent)
	}
	if result != (ImportResult{Added: 1, AlreadyPresent: 1, Problems: 4}) || f.lastEntry(t).Details != "Added 1 option from 'regions.csv'; skipped 5 rows." {
		t.Errorf("result = %+v, entry %+v", result, f.lastEntry(t))
	}
	options, _ := f.service.Options(t.Context(), region.ID)
	europe, asia := tf.option(LevelCampaign, "Cluster", "Lubricants – Europe"), tf.option(LevelCampaign, "Cluster", "Lubricants – Asia Pacific")
	if len(options) != 2 || options[1].Label != "LUBES – NORDICS" || !reflect.DeepEqual(options[1].ParentOptionIDs, []string{europe, asia}) {
		t.Errorf("options = %+v", options)
	}
}

// The file is refused as a whole — on the field content — when it cannot be decoded or read, is larger
// than 700 KB, is an .xls file, has more than 5,000 rows or none below the header; rows beyond the 5,000
// options of an attribute are problems; an attribute without options takes no import (FR-031, D-053).
func TestImport_RowAndSizeLimits_FR031(t *testing.T) {
	f := newTestService(t)
	existing := make([]string, 4999)
	for i := range existing {
		existing[i] = fmt.Sprintf("CC-%04d", i+1)
	}
	tf := f.addAttributes(t,
		testAttribute{level: LevelLineItem, label: "Cost centre", typ: TypeSingleChoice, options: existing},
		testAttribute{level: LevelLineItem, label: "Notes", typ: TypeText})
	costCentre := tf.attribute(LevelLineItem, "Cost centre").ID
	check := func(file ImportFile) error {
		_, err := f.service.CheckImport(t.Context(), alex, costCentre, file)
		return err
	}

	var tooMany strings.Builder
	tooMany.WriteString("Option\n")
	for i := range 5001 {
		fmt.Fprintf(&tooMany, "NEW-%d\n", i)
	}
	assertError(t, check(importFile("big.csv", tooMany.String())), problems("content", "The file has 5,001 rows — import at most 5,000 rows at a time."))
	_, importErr := f.service.Import(t.Context(), alex, costCentre, importFile("big.csv", tooMany.String()))
	assertError(t, importErr, problems("content", "The file has 5,001 rows — import at most 5,000 rows at a time."))
	assertError(t, check(importFile("empty.csv", "Option\n\n")), problems("content", "The file has no rows below the header."))
	assertError(t, check(importFile("large.csv", strings.Repeat("a", 700*1024+1))),
		problems("content", "The file is larger than 700 KB — split it into smaller files."))
	assertError(t, check(ImportFile{FileName: "x.csv", Content: "not base64!"}),
		problems("content", "This file can't be read. Save it again as .csv or .xlsx and try again."))
	assertError(t, check(importFile("old.xls", "\xD0\xCF\x11\xE0\xA1\xB1\x1A\xE1")),
		problems("content", "Choose a .csv or .xlsx file. An older .xls file can be saved as .xlsx in Excel."))

	overLimit, err := f.service.CheckImport(t.Context(), alex, costCentre, importFile("new.csv", "NEW-1\nNEW-2\nNEW-3\n"))
	wantProblems := []ImportRow{{Row: 2, Label: "NEW-2", Message: "Over the 5,000-option limit of this attribute."},
		{Row: 3, Label: "NEW-3", Message: "Over the 5,000-option limit of this attribute."}}
	if err != nil || len(overLimit.ToAdd) != 1 || !reflect.DeepEqual(overLimit.Problems, wantProblems) {
		t.Errorf("over the limit = %+v, %v", overLimit, err)
	}

	_, text := f.service.CheckImport(t.Context(), alex, tf.attribute(LevelLineItem, "Notes").ID, importFile("notes.csv", "A\n"))
	_, gone := f.service.Import(t.Context(), alex, "SA-9999", importFile("x.csv", "A\n"))
	assertError(t, text, problems("content", "Notes has no options — its type is Text."))
	assertError(t, gone, errs.NotFound("This Shell attribute no longer exists — someone may have deleted it."))
}

// A file with nothing new to add is refused by the import, which adds and logs nothing; the check shows
// why (FR-031).
func TestImport_NothingToAdd_FR031(t *testing.T) {
	f := newTestService(t)
	tf := f.addAttributes(t, testAttribute{level: LevelActivity, label: "Media channel", typ: TypeSingleChoice, options: []string{"TV", "Radio"}})
	attribute := tf.attribute(LevelActivity, "Media channel")
	file := importFile("channels.csv", "Option\nTV\n   ,a note\n")

	check, err := f.service.CheckImport(t.Context(), alex, attribute.ID, file)
	_, nothing := f.service.Import(t.Context(), alex, attribute.ID, file)

	if err != nil || check.TotalRows != 2 || len(check.ToAdd) != 0 || len(check.AlreadyPresent) != 1 || len(check.Problems) != 1 {
		t.Errorf("check = %+v, %v", check, err)
	}
	assertError(t, nothing, problems("content", "Nothing to add — every row is already an option or has a problem."))
	if options, _ := f.service.Options(t.Context(), attribute.ID); len(options) != 2 || len(f.logEntries(t)) != 0 {
		t.Errorf("options = %+v, log %+v", options, f.logEntries(t))
	}
}
