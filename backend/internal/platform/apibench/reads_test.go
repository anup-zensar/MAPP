package apibench_test

import (
	"net/http"
	"testing"

	"mapp/internal/platform/apibench"
)

const (
	alex   = "alex.morgan@shell.com"
	priya  = "priya.raman@shell.com"
	daniel = "daniel.okoro@shell.com"
)

type page struct {
	name, who, path string
}

// listPages are the list and overview pages: the same requests on the demo data and on the volume.
var listPages = []page{
	{"plans", alex, "/api/v1/plans"},
	{"plans search", alex, "/api/v1/plans?q=lubricants"},
	{"plans draft", alex, "/api/v1/plans?status=draft"},
	{"plans my teams", priya, "/api/v1/plans?scope=myTeams"},
	{"home summary", priya, "/api/v1/home/summary?scope=myTeams"},
	{"home plans", priya, "/api/v1/home/plans?scope=myTeams"},
	{"home attention", priya, "/api/v1/home/attention"},
	{"home waiting", daniel, "/api/v1/home/waiting"},
	{"home my activities", priya, "/api/v1/home/my-activities"},
	{"home recent changes", priya, "/api/v1/home/recent-changes?scope=myTeams"},
	{"home financials", priya, "/api/v1/financials/overview?scope=myTeams"},
	{"all work grid", priya, "/api/v1/all-work?year=2026"},
	{"all work search", priya, "/api/v1/all-work?year=2026&search=launch"},
	{"all work calendar", priya, "/api/v1/all-work?year=2026&view=calendar"},
	{"all work budget", priya, "/api/v1/all-work?year=2026&view=budget"},
	{"all work options", priya, "/api/v1/all-work/options?year=2026"},
	{"all work five years", alex, "/api/v1/all-work?yearFrom=2026&yearTo=2030"},
	{"approvals waiting", daniel, "/api/v1/approvals"},
	{"approvals open", daniel, "/api/v1/approvals?tab=open"},
	{"approvals decided", daniel, "/api/v1/approvals?tab=decided"},
	{"approvals summary", daniel, "/api/v1/approvals/summary"},
	{"notifications", daniel, "/api/v1/notifications"},
	{"activity log", alex, "/api/v1/activity-log"},
	{"activity log filtered", alex, "/api/v1/activity-log?recordType=activity&q=launch"},
	{"activity log summary", alex, "/api/v1/activity-log/summary?todayStart=2026-03-12T00:00:00Z&weekStart=2026-03-09T00:00:00Z"},
	{"recycle bin", alex, "/api/v1/recycle-bin"},
	{"financials", alex, "/api/v1/financials/transactions"},
	{"financials overview", alex, "/api/v1/financials/overview?scope=everything"},
	{"global budgets", alex, "/api/v1/global-budgets"},
	{"global budget", alex, "/api/v1/global-budgets/GB-1001"},
	{"people", alex, "/api/v1/people"},
	{"teams", alex, "/api/v1/teams"},
	{"me", priya, "/api/v1/me"},
	{"settings overview", alex, "/api/v1/settings/overview"},
	{"models", alex, "/api/v1/models"},
	{"activity types", alex, "/api/v1/activity-types"},
	{"shell attributes", alex, "/api/v1/shell-attributes"},
	{"currencies", alex, "/api/v1/currencies"},
	{"finance budgets", alex, "/api/v1/finance-budgets"},
	{"approval workflows", alex, "/api/v1/approval-workflows"},
	{"roles", alex, "/api/v1/roles"},
	{"access people", alex, "/api/v1/access/people"},
}

// BenchmarkLists shows whether a page's queries grow with the data: each runs on the demo data and on the
// volume of the scale targets (feature 028).
func BenchmarkLists(b *testing.B) {
	for _, data := range []apibench.Data{apibench.Demo, apibench.Volume} {
		server := apibench.New(b, data)
		clients := signIn(b, server)
		for _, p := range listPages {
			b.Run(string(data)+"/"+p.name, func(b *testing.B) { clients[p.who].Measure(b, http.MethodGet, p.path, nil) })
		}
	}
}

// BenchmarkRecords opens the pages of one record of the demo plan and of the plan of 500 activities.
func BenchmarkRecords(b *testing.B) {
	server := apibench.New(b, apibench.Volume)
	alexClient := server.SignIn(b, alex)
	records := server.Records(b)
	for _, size := range []struct {
		name string
		tree apibench.Tree
	}{{"small", records.Small}, {"large", records.Large}} {
		for _, p := range recordPages(size.tree) {
			b.Run(size.name+"/"+p.name, func(b *testing.B) { alexClient.Measure(b, http.MethodGet, p.path, nil) })
		}
	}
}

func recordPages(t apibench.Tree) []page {
	return []page{
		{"plan", alex, "/api/v1/plans/" + t.Plan},
		{"plan tree", alex, "/api/v1/plans/" + t.Plan + "/campaigns"},
		{"plan activities", alex, "/api/v1/plans/" + t.Plan + "/activities"},
		{"plan financials", alex, "/api/v1/financials/records/" + t.Plan},
		{"plan budget by month", alex, "/api/v1/plans/" + t.Plan + "/budget-by-month"},
		{"plan copy options", alex, "/api/v1/plans/" + t.Plan + "/copy-options"},
		{"campaign", alex, "/api/v1/campaigns/" + t.Campaign},
		{"initiative", alex, "/api/v1/initiatives/" + t.Initiative},
		{"activity", alex, "/api/v1/activities/" + t.Activity},
		{"activity form", alex, "/api/v1/activities/" + t.Activity + "/form"},
		{"activity line items", alex, "/api/v1/activities/" + t.Activity + "/line-items"},
		{"activity approval", alex, "/api/v1/activities/" + t.Activity + "/approval"},
		{"activity destinations", alex, "/api/v1/activities/" + t.Activity + "/destinations"},
		{"activity history", alex, "/api/v1/activity-log?recordId=" + t.Activity},
		{"line item", alex, "/api/v1/line-items/" + t.LineItem},
		{"line item form options", alex, "/api/v1/activities/" + t.Activity + "/line-items/form-options"},
		{"all work record", alex, "/api/v1/all-work/records/" + t.Activity},
	}
}

func signIn(b *testing.B, server *apibench.Server) map[string]*apibench.Client {
	b.Helper()
	return map[string]*apibench.Client{
		alex: server.SignIn(b, alex), priya: server.SignIn(b, priya), daniel: server.SignIn(b, daniel),
	}
}
