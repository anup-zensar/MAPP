package apibench_test

import (
	"net/http"
	"testing"

	"mapp/internal/platform/apibench"
)

// waitingRow is a row of Daniel's Waiting for me tab, as the approval benchmarks decide on it.
type waitingRow struct {
	Activity struct {
		ID string `json:"id"`
	} `json:"activity"`
	Round struct {
		ID    string `json:"id"`
		Level int    `json:"currentLevel"`
	} `json:"round"`
}

// BenchmarkApprovals opens the approval pages of one activity and runs the decisions that BenchmarkChanges
// leaves out, on the volume.
func BenchmarkApprovals(b *testing.B) {
	server := apibench.New(b, apibench.Volume)
	alexClient, danielClient := server.SignIn(b, alex), server.SignIn(b, daniel)
	waiting := waitingRows(b, danielClient)
	activity := waiting[0].Activity.ID
	for _, p := range []page{
		{"approval monitoring", alex, "/api/v1/approvals/monitoring"},
		{"approval workflow", alex, "/api/v1/approval-workflows/WF-1001"},
		{"approval delegations", alex, "/api/v1/approval-delegations"},
		{"activity approval", daniel, "/api/v1/activities/" + activity + "/approval"},
		{"approval preview", daniel, "/api/v1/approvals/activities/" + activity},
	} {
		client := map[string]*apibench.Client{alex: alexClient, daniel: danielClient}[p.who]
		b.Run(p.name, func(b *testing.B) { client.Measure(b, http.MethodGet, p.path, nil) })
	}
	c := &changes{server: server, alex: alexClient, daniel: danielClient}
	b.Run("decline", func(b *testing.B) { c.decide(b, "decline", waiting[1:]) })
	b.Run("mark notifications read", func(b *testing.B) {
		c.measure(b, func() int64 {
			return danielClient.Do(b, http.MethodPost, "/api/v1/notifications/read-all", nil).Queries
		})
	})
}

func waitingRows(b *testing.B, daniel *apibench.Client) []waitingRow {
	b.Helper()
	var tab struct {
		Items []waitingRow `json:"items"`
	}
	daniel.Do(b, http.MethodGet, "/api/v1/approvals", nil).Decode(b, &tab)
	if len(tab.Items) < 2 {
		b.Fatalf("apibench: %d activities wait for Daniel, want at least 2", len(tab.Items))
	}
	return tab.Items
}

// decide takes the decision on the next waiting row for every iteration.
func (c *changes) decide(b *testing.B, decision string, rows []waitingRow) {
	b.Helper()
	next := 0
	c.measure(b, func() int64 {
		row := rows[next%len(rows)]
		next++
		body := map[string]any{"roundId": row.Round.ID, "level": row.Round.Level, "comment": "Bench reason"}
		return c.daniel.Do(b, http.MethodPost, "/api/v1/activities/"+row.Activity.ID+"/approval/"+decision, body).Queries
	})
}

// readyActivity returns an activity of the volume's Lubricants plans that is ready to submit.
func readyActivity(b *testing.B, server *apibench.Server, client *apibench.Client) string {
	b.Helper()
	all, err := server.Set().Activities.List(b.Context())
	if err != nil {
		b.Fatal(err)
	}
	for _, a := range all {
		if a.Status != "created" || a.ApprovalWorkflowID == "" {
			continue
		}
		var check struct {
			Available bool `json:"available"`
			Ready     bool `json:"ready"`
		}
		client.Do(b, http.MethodGet, "/api/v1/activities/"+a.ID+"/approval/submission", nil).Decode(b, &check)
		if check.Available && check.Ready {
			return a.ID
		}
	}
	b.Fatal("apibench: no activity is ready to submit")
	return ""
}
