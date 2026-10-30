package apibench_test

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"mapp/internal/platform/apibench"
)

const (
	demoPlan     = "P-1001"
	bulkRecords  = 500
	importRows   = 1000
	spaRecords   = 2000
	campaignCopy = "/api/v1/campaigns/%s/copy"
)

// BenchmarkBulkAndImport changes the owner of 500 activities of the demo plan, grown to hold them, and checks a
// file of 1,000 new activities for it; "-bench BenchmarkBulkAndImport/demo" leaves out the volume, slow to build.
func BenchmarkBulkAndImport(b *testing.B) {
	for _, data := range []apibench.Data{apibench.Demo, apibench.Volume} {
		b.Run(string(data), func(b *testing.B) { bulkAndImport(b, data) })
	}
}

func bulkAndImport(b *testing.B, data apibench.Data) {
	b.Helper()
	server := apibench.New(b, data)
	alexClient := server.SignIn(b, alex)
	selected := growPlan(b, server, alexClient, bulkRecords)[:bulkRecords]
	c := &changes{server: server, alex: alexClient}
	b.Run("bulk change owner preview", func(b *testing.B) { c.changeOwner(b, "/api/v1/bulk-actions/preview", selected) })
	b.Run("bulk change owner", func(b *testing.B) { c.changeOwner(b, "/api/v1/bulk-actions", selected) })
	file := activitiesFile(b, server)
	path := "/api/v1/plans/" + demoPlan + "/imports/check"
	var check struct {
		Ready int `json:"ready"`
	}
	alexClient.Do(b, http.MethodPost, path, file).Decode(b, &check)
	if check.Ready != importRows {
		b.Fatalf("apibench: %d of %d rows are ready to import", check.Ready, importRows)
	}
	b.Run("import check", func(b *testing.B) { alexClient.Measure(b, http.MethodPost, path, file) })
}

// growPlan copies the demo plan's campaigns into it until it holds at least want activities, and returns them.
func growPlan(b *testing.B, server *apibench.Server, client *apibench.Client, want int) []string {
	b.Helper()
	originals := server.CampaignIDs(b, demoPlan)
	activities := server.ActivityIDs(b, demoPlan)
	for round := 1; len(activities) < want; round++ {
		for _, campaignID := range originals {
			body := map[string]any{"planId": demoPlan, "name": fmt.Sprintf("Bench %s %d", campaignID, round)}
			client.Do(b, http.MethodPost, fmt.Sprintf(campaignCopy, campaignID), body)
		}
		activities = server.ActivityIDs(b, demoPlan)
	}
	return activities
}

func (c *changes) changeOwner(b *testing.B, path string, recordIDs []string) {
	b.Helper()
	owners := [2]string{"U-1001", "U-1002"}
	c.measure(b, func() int64 {
		owners[0], owners[1] = owners[1], owners[0]
		body := map[string]any{"action": "changeOwner", "recordIds": recordIDs, "ownerId": owners[0]}
		return c.alex.Do(b, http.MethodPost, path, body).Queries
	})
}

func activitiesFile(b *testing.B, server *apibench.Server) map[string]any {
	b.Helper()
	campaignID := server.CampaignIDs(b, demoPlan)[0]
	initiatives, err := server.Set().Campaigns.InitiativesOfCampaign(b.Context(), campaignID)
	if err != nil {
		b.Fatal(err)
	}
	lines := []string{"Campaign;Initiative;Activity name;Activity type;Planned cost"}
	for n := 1; n <= importRows; n++ {
		lines = append(lines, fmt.Sprintf("%s;%s;Bench import %04d;Event;1000.00", campaignID, initiatives[0].ID, n))
	}
	content := base64.StdEncoding.EncodeToString([]byte(strings.Join(lines, "\n") + "\n"))
	return map[string]any{"fileName": "activities.csv", "content": content}
}
