package apibench_test

import (
	"net/http"
	"testing"

	"mapp/internal/platform/apibench"
)

// BenchmarkDashboards measures feature 036's SC-003 on the volume: the Executive dashboard of 2026 within
// 3 seconds, and a plan's export within 5 seconds (the plan of 500 activities is the largest).
func BenchmarkDashboards(b *testing.B) {
	server := apibench.New(b, apibench.Volume)
	clients := signIn(b, server)
	large := server.Records(b).Large.Plan
	for _, p := range []page{
		{"executive", alex, "/api/v1/dashboards/executive?year=2026"},
		{"executive one market", priya, "/api/v1/dashboards/executive?year=2026"},
		{"regional", priya, "/api/v1/dashboards/regional?year=2026"},
		{"executive export pptx", alex, "/api/v1/dashboards/executive/export?year=2026&format=pptx"},
		{"plan export pptx", alex, "/api/v1/plans/" + large + "/export?format=pptx"},
		{"plan export pdf", alex, "/api/v1/plans/" + large + "/export?format=pdf"},
	} {
		b.Run(p.name, func(b *testing.B) { clients[p.who].Measure(b, http.MethodGet, p.path, nil) })
	}
}
