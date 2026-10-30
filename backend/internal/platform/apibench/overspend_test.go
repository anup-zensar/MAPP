package apibench_test

import (
	"fmt"
	"net/http"
	"testing"

	"mapp/internal/campaign"
	"mapp/internal/platform/apibench"
	"mapp/internal/platform/money"
)

// BenchmarkOverspend creates an activity under a campaign set to Refuse, on both databases: one that fits and one
// the guard refuses, so the guard's reads of every budget before and after the save show in time and statements.
func BenchmarkOverspend(b *testing.B) {
	for _, data := range []apibench.Data{apibench.Demo, apibench.Volume} {
		b.Run(string(data), func(b *testing.B) {
			server := apibench.New(b, data)
			client := server.SignIn(b, alex)
			refuseCampaign(b, client, "C-1001", "1000000.00")
			saves := &overspendSaves{client: client, path: "/api/v1/initiatives/I-1001/activities"}
			b.Run("allowed save", func(b *testing.B) { saves.measure(b, "1.00", http.StatusCreated) })
			b.Run("refused save", func(b *testing.B) { saves.measure(b, "2000000.00", http.StatusConflict) })
		})
	}
}

type overspendSaves struct {
	client *apibench.Client
	path   string
	count  int
}

func (s *overspendSaves) measure(b *testing.B, cost string, want int) {
	b.Helper()
	var queries int64
	for b.Loop() {
		s.count++
		body := map[string]any{"name": fmt.Sprintf("Bench overspend %d", s.count), "activityTypeId": "AT-1002",
			"plannedCost": map[string]string{"amount": cost, "currency": "USD"}}
		answer := s.client.Send(b, http.MethodPost, s.path, body)
		if answer.Status != want {
			b.Fatalf("apibench: POST %s: %d %s, want %d", s.path, answer.Status, answer.Body, want)
		}
		queries += answer.Queries
	}
	b.ReportMetric(float64(queries)/float64(b.N), "queries/op")
}

// refuseCampaign sets the campaign to Refuse with room left to plan above its planned cost.
func refuseCampaign(b *testing.B, client *apibench.Client, campaignID, room string) {
	b.Helper()
	var view campaign.CampaignView
	client.Do(b, http.MethodGet, "/api/v1/campaigns/"+campaignID, nil).Decode(b, &view)
	extra, err := money.ParseAmount(room)
	if err != nil {
		b.Fatal(err)
	}
	planned := view.BudgetAndCost.PlannedCost
	client.Do(b, http.MethodPatch, "/api/v1/campaigns/"+campaignID, map[string]any{
		"name": view.Name, "description": view.Description, "ownerId": view.OwnerID, "startDate": view.StartDate,
		"endDate": view.EndDate, "attributes": view.Attributes, "overspend": "refuse", "version": view.Version,
		"budget": map[string]string{"amount": planned.Amount.Add(extra).String(), "currency": planned.Currency},
	})
}
