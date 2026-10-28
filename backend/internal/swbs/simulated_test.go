package swbs_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/fake"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
	"mapp/internal/swbs"
	"mapp/internal/swbs/swbstest"
)

type issued map[string]int

func (i issued) CountCodes(_ context.Context, prefix string) (int, error) {
	return i[prefix], nil
}

func TestSimulated_030_FR007(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		before   issued
		requests []swbs.Request
		want     []string
	}{
		{name: "the first code of a company code", requests: []swbs.Request{{LineItemID: "L-1006", CompanyCode: "GB01"}},
			want: []string{"SIM-GB01-0001"}},
		{name: "numbering goes on after the codes already given", before: issued{"SIM-GB01-": 41},
			requests: []swbs.Request{{LineItemID: "L-1006", CompanyCode: "GB01"}, {LineItemID: "L-1007", CompanyCode: "GB01"}},
			want:     []string{"SIM-GB01-0042", "SIM-GB01-0043"}},
		{name: "each company code has its own numbers", before: issued{"SIM-NL02-": 2},
			requests: []swbs.Request{{LineItemID: "L-1", CompanyCode: "GB01"}, {LineItemID: "L-2", CompanyCode: "NL02"},
				{LineItemID: "L-3", CompanyCode: "GB01"}},
			want: []string{"SIM-GB01-0001", "SIM-NL02-0003", "SIM-GB01-0002"}},
		{name: "no company code uses 0000", before: issued{"SIM-0000-": 7}, requests: []swbs.Request{{LineItemID: "L-1"}},
			want: []string{"SIM-0000-0008"}},
		{name: "a company code is kept to its letters and digits", requests: []swbs.Request{{LineItemID: "L-1", CompanyCode: " gb 01 "}},
			want: []string{"SIM-GB01-0001"}},
		{name: "a long company code is cut to ten characters",
			requests: []swbs.Request{{LineItemID: "L-1", CompanyCode: strings.Repeat("A", 12)}}, want: []string{"SIM-AAAAAAAAAA-0001"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			answers, err := swbs.NewSimulated(tt.before).RequestCodes(t.Context(), tt.requests)
			if err != nil {
				t.Fatalf("RequestCodes: %v", err)
			}
			got := []string{}
			for i, a := range answers {
				if a.LineItemID != tt.requests[i].LineItemID {
					t.Errorf("answer %d is for %s, want %s", i, a.LineItemID, tt.requests[i].LineItemID)
				}
				got = append(got, a.SwbsCode)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("codes (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLoadDemo_030_FR021(t *testing.T) {
	t.Parallel()
	db := fake.NewDB()
	store := swbstest.NewStore(db)
	items := &fakeLineItems{}
	items.add("L-1003", "Venue hire", "DE03", true)
	var run swbs.Run
	err := db.Write(t.Context(), func(ctx context.Context) error {
		var err error
		run, err = swbs.LoadDemo(ctx, store, ids.NewGenerator(fake.NewCounters(db)), swbs.DemoLoad{
			Requests: []swbs.Request{{LineItemID: "L-1003", LineItemName: "Venue hire", CompanyCode: "DE03"}}, Items: items, At: start,
		})
		return err
	})
	if err != nil {
		t.Fatalf("LoadDemo: %v", err)
	}
	want := swbs.Run{ID: "SWR-1001", Source: swbs.SourceDemo, Meta: record.NewMeta(actor.System, start), Codes: []swbs.Code{
		{LineItemID: "L-1003", LineItemName: "Venue hire", CompanyCode: "DE03", SwbsCode: "SIM-DE03-0001"},
	}}
	stored, _, err := store.GetRun(t.Context(), run.ID)
	if diff := cmp.Diff(want, stored); err != nil || diff != "" {
		t.Errorf("stored demo run (-want +got), %v:\n%s", err, diff)
	}
	if items.codeOf("L-1003") != "SIM-DE03-0001" {
		t.Errorf("Venue hire's code = %q", items.codeOf("L-1003"))
	}
}
