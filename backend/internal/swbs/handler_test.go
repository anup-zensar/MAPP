package swbs_test

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/lineitem"
	"mapp/internal/platform/apitest"
	"mapp/internal/swbs"
)

const (
	alexsEmail  = "alex.morgan@shell.com"
	priyasEmail = "priya.raman@shell.com"
	runsPath    = "/api/v1/simulated-gsap/swbs-runs"
)

func decoded[T any](t *testing.T, response *apitest.Response, status int) T {
	t.Helper()
	if response.Status != status {
		t.Fatalf("status %d %s, want %d", response.Status, response.Body, status)
	}
	var v T
	response.Decode(t, &v)
	return v
}

type apiError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func TestSwbsRuns_Contract_030_FR011(t *testing.T) {
	t.Parallel()
	alex := apitest.New(t).SignIn(t, alexsEmail)

	page := decoded[swbs.RunPage](t, alex.Do(t, http.MethodGet, runsPath, nil), http.StatusOK)

	want := []swbs.Code{{LineItemID: "L-1003", LineItemName: "Venue hire", CompanyCode: "DE03", SwbsCode: "SIM-DE03-0001"}}
	if page.Total != 1 || page.ReadyToSend != 1 || len(page.Items) != 1 || page.Items[0].ID != "SWR-1001" ||
		page.Items[0].Source != swbs.SourceDemo || page.Items[0].RanBy.Name != "MAPP" {
		t.Errorf("page = %+v", page)
	}
	if diff := cmp.Diff(want, page.Items[0].Codes); diff != "" {
		t.Errorf("demo run codes (-want +got):\n%s", diff)
	}
	refused := alex.Do(t, http.MethodGet, runsPath+"?limit=0", nil, apitest.SkipRequestValidation())
	if body := decoded[apiError](t, refused, http.StatusBadRequest); body.Error.Code != "validation_failed" {
		t.Errorf("limit 0 = %+v", body)
	}
}

func TestSwbsRun_Contract_030_FR006(t *testing.T) {
	t.Parallel()
	alex := apitest.New(t).SignIn(t, alexsEmail)

	run := decoded[swbs.RunView](t, alex.Do(t, http.MethodPost, runsPath, map[string]any{}), http.StatusCreated)

	want := swbs.RunView{ID: "SWR-1002", Source: swbs.SourceSimulated, RanAt: run.RanAt, RanBy: run.RanBy, Sent: 1,
		Codes: []swbs.Code{{LineItemID: "L-1006", LineItemName: "Stand build", CompanyCode: "GB01", SwbsCode: "SIM-GB01-0001"}}}
	if diff := cmp.Diff(want, run); diff != "" || run.RanBy.Name != "Alex Morgan" {
		t.Errorf("run by %s (-want +got):\n%s", run.RanBy.Name, diff)
	}
	stand := decoded[lineitem.Details](t, alex.Do(t, http.MethodGet, "/api/v1/line-items/L-1006", nil), http.StatusOK).LineItem
	got := lineitem.Swbs{SwbsCode: stand.SwbsCode, SwbsRequestedAt: stand.SwbsRequestedAt, SwbsReceivedAt: stand.SwbsReceivedAt}
	ranAt := run.RanAt
	if diff := cmp.Diff(lineitem.Swbs{SwbsCode: "SIM-GB01-0001", SwbsRequestedAt: &ranAt, SwbsReceivedAt: &ranAt}, got); diff != "" {
		t.Errorf("Stand build SWBS (-want +got):\n%s", diff)
	}
	if stand.SwbsStatus != lineitem.SwbsReceived || stand.HXKey != "HX1006" {
		t.Errorf("Stand build status %s, HX key %s", stand.SwbsStatus, stand.HXKey)
	}
	again := decoded[swbs.RunView](t, alex.Do(t, http.MethodPost, runsPath, map[string]any{}), http.StatusCreated)
	if again.Sent != 0 || len(again.Codes) != 0 {
		t.Errorf("a second run = %+v, want nothing sent", again)
	}
}

func TestSwbsRun_Named_Contract_030_FR008(t *testing.T) {
	t.Parallel()
	alex := apitest.New(t).SignIn(t, alexsEmail)
	tests := []struct {
		name   string
		body   map[string]any
		status int
		code   string
		opts   []apitest.RequestOption
	}{
		{name: "unknown line item", body: map[string]any{"lineItemIds": []string{"L-9999"}}, status: http.StatusNotFound,
			code: "not_found"},
		{name: "not ready", body: map[string]any{"lineItemIds": []string{"L-1001"}}, status: http.StatusConflict,
			code: "not_allowed"},
		{name: "an empty list", body: map[string]any{"lineItemIds": []string{}}, status: http.StatusBadRequest,
			code: "validation_failed", opts: []apitest.RequestOption{apitest.SkipRequestValidation()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			body := decoded[apiError](t, alex.Do(t, http.MethodPost, runsPath, tt.body, tt.opts...), tt.status)
			if body.Error.Code != tt.code {
				t.Errorf("code = %s, want %s", body.Error.Code, tt.code)
			}
		})
	}
}

func TestSwbsRun_OneNamed_Contract_030_FR015(t *testing.T) {
	t.Parallel()
	alex := apitest.New(t).SignIn(t, alexsEmail)

	only := decoded[swbs.RunView](t, alex.Do(t, http.MethodPost, runsPath, map[string]any{"lineItemIds": []string{"L-1006"}}),
		http.StatusCreated)
	if only.Sent != 1 || only.Codes[0].LineItemID != "L-1006" {
		t.Errorf("named run = %+v", only)
	}
}

func TestSwbsRuns_Forbidden_030_FR012(t *testing.T) {
	t.Parallel()
	priya := apitest.New(t).SignIn(t, priyasEmail)

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		var body any
		if method == http.MethodPost {
			body = map[string]any{}
		}
		if refused := decoded[apiError](t, priya.Do(t, method, runsPath, body), http.StatusForbidden); refused.Error.Code != "forbidden" {
			t.Errorf("%s = %+v", method, refused)
		}
	}
}

func TestSwbsCode_LineItemSeed_030_FR021(t *testing.T) {
	t.Parallel()
	priya := apitest.New(t).SignIn(t, priyasEmail)

	venue := decoded[lineitem.Details](t, priya.Do(t, http.MethodGet, "/api/v1/line-items/L-1003", nil), http.StatusOK).LineItem
	stand := decoded[lineitem.Details](t, priya.Do(t, http.MethodGet, "/api/v1/line-items/L-1006", nil), http.StatusOK).LineItem
	if venue.SwbsStatus != lineitem.SwbsReceived || venue.SwbsCode != "SIM-DE03-0001" || venue.SwbsRequestedAt == nil {
		t.Errorf("Venue hire = %s %q", venue.SwbsStatus, venue.SwbsCode)
	}
	if stand.SwbsStatus != lineitem.SwbsReadyToSend || stand.SwbsCode != "" || stand.SwbsReceivedAt != nil {
		t.Errorf("Stand build = %s %q", stand.SwbsStatus, stand.SwbsCode)
	}
}
