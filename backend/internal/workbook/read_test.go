package workbook

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/errs"
)

func TestRead_HeadingsAnyOrderAndCase_IgnoredColumns_MarkingRow_012FR028(t *testing.T) {
	t.Parallel()
	up := uploadOf(t, map[string][][]string{"Activities": {
		{"activity NAME *", "Notes", " Initiative ", "Campaign", "Activity Type"},
		{"Required", "", "Required", "Required", "Required"},
		{"Podcast", "ask Mei", "Digital", "Helix launch", "Event"},
	}})

	sheets, ignored, err := readUpload(up, testChoices)
	if err != nil {
		t.Fatalf("readUpload: %v", err)
	}

	if diff := cmp.Diff([]string{"Activities: Notes"}, ignored); diff != "" {
		t.Errorf("ignored (-want +got):\n%s", diff)
	}
	s := sheets[0]
	got := []string{s.text(s.rows[0], fieldName), s.text(s.rows[0], fieldInitiative), s.text(s.rows[0], fieldType)}
	if diff := cmp.Diff([]string{"Podcast", "Digital", "Event"}, got); diff != "" || len(s.rows) != 1 {
		t.Errorf("row (-want +got):\n%s rows %d", diff, len(s.rows))
	}
}

func TestRead_FileRefusals_EveryMissingColumnAtOnce_012FR028(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		up   Upload
		want []string
	}{
		{name: "two required columns missing", up: uploadOf(t, map[string][][]string{"Initiatives": {{"Owner"}, {"x"}}}),
			want: []string{"The Initiatives sheet has no 'Campaign' column — download a fresh template",
				"The Initiatives sheet has no 'Initiative name' column — download a fresh template"}},
		{name: "no level sheet", up: Upload{FileName: "a.csv", Content: ""}, want: []string{msgEmptyFile}},
		{name: "no file name", up: Upload{Content: "YQ=="}, want: []string{msgFileName}},
		{name: "a text file", up: Upload{FileName: "notes.txt", Content: base64.StdEncoding.EncodeToString([]byte("a"))},
			want: []string{msgChooseFile}},
		{name: "too many rows", up: Upload{FileName: "a.csv", Content: base64.StdEncoding.EncodeToString(
			[]byte("Campaign,Initiative,Activity name,Activity type\n" + strings.Repeat("a,b,c,d\n", maxFileRows+1)))},
			want: []string{msgTooManyRows}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := readUpload(tt.up, testChoices)
			apiErr, ok := errors.AsType[*errs.Error](err)
			if !ok {
				t.Fatalf("error = %v, want validation_failed", err)
			}
			got := []string{}
			for _, f := range apiErr.Fields {
				got = append(got, f.Message)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("messages (-want +got):\n%s", diff)
			}
		})
	}
}

// FuzzCheck runs any CSV through the check: it never panics, and every row read is ready or left out.
func FuzzCheck(f *testing.F) {
	f.Add("Campaign,Initiative,Activity name,Activity type\nHelix launch,Digital,Podcast,Event\n")
	f.Add("ID;Campaign;Initiative;Activity name;Activity type;Planned cost\nA-1;Helix launch;Digital;Launch film;Event;-1\n")
	f.Add("Campaign,Initiative,Activity name,Activity type\nNowhere,,x,y\n\"a\nb\",c")
	f.Fuzz(func(t *testing.T, text string) {
		fp := &fakePlanning{plan: testPlan()}
		up := Upload{FileName: "f.csv", Content: base64.StdEncoding.EncodeToString([]byte(text))}
		sheets, _, err := readUpload(up, fp.plan.Choices)
		if err != nil {
			return
		}
		ap := newApplier(priya, fp, fp.plan)
		if err := ap.run(t.Context(), sheets); err != nil {
			t.Fatalf("run: %v", err)
		}
		if ap.result.Ready+ap.result.WithProblems != ap.result.Rows {
			t.Fatalf("%d rows, %d ready and %d with problems", ap.result.Rows, ap.result.Ready, ap.result.WithProblems)
		}
	})
}
