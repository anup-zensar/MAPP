package apibench_test

import (
	"net/http"
	"slices"
	"testing"
	"time"

	"mapp/internal/platform/apibench"
)

// BenchmarkSpaView opens the SP&A view of the demo plan, then of the same plan grown to 2,000 activities: the
// statements stay the same whatever the plan's size, and p95-ms is the time SC-006 limits.
func BenchmarkSpaView(b *testing.B) {
	for _, data := range []apibench.Data{apibench.Demo, apibench.Volume} {
		b.Run(string(data), func(b *testing.B) { spaViews(b, data) })
	}
}

func spaViews(b *testing.B, data apibench.Data) {
	b.Helper()
	server := apibench.New(b, data)
	alexClient := server.SignIn(b, alex)
	path := "/api/v1/plans/" + demoPlan + "/spa"
	b.Run("demo plan", func(b *testing.B) { measureP95(b, alexClient, path) })
	activities := growPlan(b, server, alexClient, spaRecords)
	b.Logf("the grown plan holds %d activities", len(activities))
	b.Run("2000 activities", func(b *testing.B) { measureP95(b, alexClient, path) })
	campaignID := server.CampaignIDs(b, demoPlan)[0]
	b.Run("2000 activities campaign", func(b *testing.B) { measureP95(b, alexClient, path+"?campaignId="+campaignID) })
}

func measureP95(b *testing.B, client *apibench.Client, path string) {
	b.Helper()
	var took []time.Duration
	var queries int64
	for b.Loop() {
		started := time.Now()
		queries += client.Do(b, http.MethodGet, path, nil).Queries
		took = append(took, time.Since(started))
	}
	slices.Sort(took)
	b.ReportMetric(float64(took[len(took)*95/100].Milliseconds()), "p95-ms")
	b.ReportMetric(float64(queries)/float64(b.N), "queries/op")
}
