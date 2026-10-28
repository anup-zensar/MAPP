package swbs_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/authz"
	"mapp/internal/swbs"
)

func TestRun_SendsReady_030_FR006(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.threeReady()

	run := f.run(t, swbs.RunInput{})

	want := swbs.RunView{ID: "SWR-1001", Source: swbs.SourceSimulated, RanAt: start, RanBy: alex, Sent: 3, Codes: []swbs.Code{
		{LineItemID: "L-999", LineItemName: "Old stand", CompanyCode: "GB01", SwbsCode: "SIM-GB01-0001"},
		{LineItemID: "L-1005", LineItemName: "Media booking", CompanyCode: "NL02", SwbsCode: "SIM-NL02-0001"},
		{LineItemID: "L-1006", LineItemName: "Stand build", CompanyCode: "GB01", SwbsCode: "SIM-GB01-0002"},
	}}
	if diff := cmp.Diff(want, run); diff != "" {
		t.Errorf("run (-want +got):\n%s", diff)
	}
	stored := []string{f.items.codeOf("L-1006"), f.items.codeOf("L-1001")}
	if diff := cmp.Diff([]string{"SIM-GB01-0002", ""}, stored); diff != "" {
		t.Errorf("codes of Stand build and Agency fee (-want +got):\n%s", diff)
	}
}

func TestRun_LogsOnce_030_FR010(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.threeReady()

	f.run(t, swbs.RunInput{})

	entries := f.log(t)
	if len(entries) != 1 {
		t.Fatalf("%d log entries, want 1", len(entries))
	}
	got := []string{entries[0].RecordType, entries[0].RecordID, entries[0].Verb, entries[0].RecordName, entries[0].Details}
	want := []string{"swbsRun", "SWR-1001", "ran", "Simulated GSAP", "Simulated GSAP returned 3 SWBS codes: " +
		"Old stand (L-999) SIM-GB01-0001; Media booking (L-1005) SIM-NL02-0001; Stand build (L-1006) SIM-GB01-0002."}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("log entry (-want +got):\n%s", diff)
	}
}

func TestRun_NumberingGoesOnAcrossRuns_030_FR007(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.threeReady()
	f.run(t, swbs.RunInput{})

	f.items.add("L-1010", "Venue", "GB01", true)
	if next := f.run(t, swbs.RunInput{}); len(next.Codes) != 1 || next.Codes[0].SwbsCode != "SIM-GB01-0003" {
		t.Errorf("the next GB01 code = %+v, want SIM-GB01-0003", next.Codes)
	}
}

func TestRun_NothingReady_030_FR009(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.items.add("L-1001", "Agency fee", "DE03", false)

	run := f.run(t, swbs.RunInput{})

	if run.Sent != 0 || len(run.Codes) != 0 || run.ID != "SWR-1001" {
		t.Errorf("empty run = %+v", run)
	}
	if entries := f.log(t); len(entries) != 1 || entries[0].Details != "Nothing was ready to send." {
		t.Errorf("log = %+v", entries)
	}
}

func TestRun_Named_030_FR008(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		named     []string
		status    int
		code      string
		message   string
		wantCodes int
	}{
		{name: "unknown line item", named: []string{"L-1006", "L-9999"}, status: http.StatusNotFound, code: "not_found",
			message: "This line item doesn't exist or you can't see it."},
		{name: "not ready", named: []string{"L-1006", "L-1001", "L-1001"}, status: http.StatusConflict, code: "not_allowed",
			message: "Not sent — these line items aren't ready to send to the simulated GSAP: Agency fee (L-1001). " +
				"Reload to see their SWBS status."},
		{name: "only the named ready line item", named: []string{"L-1006"}, wantCodes: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			f.items.add("L-1006", "Stand build", "GB01", true)
			f.items.add("L-1005", "Media booking", "NL02", true)
			f.items.add("L-1001", "Agency fee", "DE03", false)

			run, err := f.service.Run(t.Context(), alex, swbs.RunInput{LineItemIDs: tt.named})

			if tt.code != "" {
				if got := assertCode(t, err, tt.status, tt.code); got.Message != tt.message {
					t.Errorf("message = %q, want %q", got.Message, tt.message)
				}
				if f.items.codeOf("L-1006") != "" || len(f.log(t)) != 0 {
					t.Errorf("a refused run saved something")
				}
				return
			}
			if err != nil || len(run.Codes) != tt.wantCodes || f.items.codeOf("L-1005") != "" {
				t.Errorf("run = %+v, %v; Media booking %q", run, err, f.items.codeOf("L-1005"))
			}
		})
	}
}

func TestRun_Validation_030_FR008(t *testing.T) {
	t.Parallel()
	many := make([]string, 201)
	for i := range many {
		many[i] = "L-1006"
	}
	tests := map[string]struct {
		named   []string
		message string
	}{
		"an empty list":         {named: []string{}, message: "Name at least one line item, or leave the list out to send every ready line item."},
		"too many line items":   {named: many, message: "Send at most 200 line items at a time."},
		"an empty ID":           {named: []string{""}, message: "Use a line-item ID such as L-1006."},
		"an ID that's too long": {named: []string{strings.Repeat("L", 41)}, message: "Use a line-item ID such as L-1006."},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			_, err := f.service.Run(t.Context(), alex, swbs.RunInput{LineItemIDs: tt.named})
			got := assertCode(t, err, http.StatusBadRequest, "validation_failed")
			if len(got.Fields) != 1 || got.Fields[0].Field != "lineItemIds" || got.Fields[0].Message != tt.message {
				t.Errorf("fields = %+v, want lineItemIds %q", got.Fields, tt.message)
			}
		})
	}
}

func TestRun_Forbidden_030_FR012(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.items.add("L-1006", "Stand build", "GB01", true)
	planner := authz.With(context.Background(), authz.Grants{PersonID: "U-1002"})

	_, runErr := f.service.Run(planner, alex, swbs.RunInput{})
	_, listErr := f.service.Runs(planner, alex, 0)

	assertCode(t, runErr, http.StatusForbidden, "forbidden")
	assertCode(t, listErr, http.StatusForbidden, "forbidden")
	if f.items.codeOf("L-1006") != "" {
		t.Errorf("a refused run sent Stand build")
	}
}

func TestRun_ReceiveFails_030_FR006(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.items.add("L-1006", "Stand build", "GB01", true)
	f.items.failed = true

	if _, err := f.service.Run(t.Context(), alex, swbs.RunInput{}); err == nil {
		t.Fatal("Run succeeded although the codes could not be stored")
	}
	page, err := f.service.Runs(t.Context(), alex, 0)
	if err != nil || page.Total != 0 || len(f.log(t)) != 0 {
		t.Errorf("after a failed run: %+v, %v, %d log entries", page, err, len(f.log(t)))
	}
}

func TestRuns_030_FR011(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.items.add("L-1006", "Stand build", "GB01", true)
	f.run(t, swbs.RunInput{})
	f.items.add("L-1005", "Media booking", "NL02", true)
	f.items.add("L-1007", "Signage", "NL02", true)
	f.run(t, swbs.RunInput{LineItemIDs: []string{"L-1005"}})

	page, err := f.service.Runs(t.Context(), alex, 1)
	if err != nil {
		t.Fatalf("Runs: %v", err)
	}
	got := []any{page.Total, page.ReadyToSend, len(page.Items), page.Items[0].ID}
	if diff := cmp.Diff([]any{2, 1, 1, "SWR-1002"}, got); diff != "" {
		t.Errorf("total, ready, items, first (-want +got):\n%s", diff)
	}
	all, err := f.service.Runs(t.Context(), alex, 0)
	if err != nil || len(all.Items) != 2 || all.Items[1].Codes[0].SwbsCode != "SIM-GB01-0001" {
		t.Errorf("default page = %+v, %v", all, err)
	}
}

func TestRuns_Limit_030_FR011(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	for _, limit := range []int{-1, 101} {
		_, err := f.service.Runs(t.Context(), alex, limit)
		if got := assertCode(t, err, http.StatusBadRequest, "validation_failed"); got.Fields[0].Message != "Ask for 1 to 100 runs." {
			t.Errorf("limit %d: %+v", limit, got.Fields)
		}
	}
}
