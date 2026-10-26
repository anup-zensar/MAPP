package apibench_test

import (
	"fmt"
	"net/http"
	"testing"

	"mapp/internal/platform/apibench"
)

// BenchmarkChanges runs each operation on the demo plan (small) and on the plan of 500 activities or the
// template of 100 (large), on the volume, so that a count that grows with the records shows.
func BenchmarkChanges(b *testing.B) {
	server := apibench.New(b, apibench.Volume)
	c := &changes{server: server, alex: server.SignIn(b, alex), daniel: server.SignIn(b, daniel)}
	records := server.Records(b)
	for _, size := range []struct {
		name       string
		tree, copy apibench.Tree
	}{{"small", records.Small, records.Small}, {"large", records.Large, records.Template}} {
		b.Run(size.name+"/copy plan", func(b *testing.B) { c.copyPlan(b, size.copy.Plan) })
		b.Run(size.name+"/copy campaign", func(b *testing.B) { c.copyCampaign(b, size.tree) })
		b.Run(size.name+"/copy initiative", func(b *testing.B) { c.copyInitiative(b, size.tree) })
		b.Run(size.name+"/copy activity", func(b *testing.B) { c.copyActivity(b, size.tree) })
		b.Run(size.name+"/move activity", func(b *testing.B) { c.moveActivity(b, size.tree) })
		b.Run(size.name+"/move initiative", func(b *testing.B) { c.moveInitiative(b, size.tree) })
		b.Run(size.name+"/delete plan", func(b *testing.B) { c.deletePlan(b, size.copy.Plan, false) })
		b.Run(size.name+"/undo delete", func(b *testing.B) { c.deletePlan(b, size.copy.Plan, true) })
		b.Run(size.name+"/restore", func(b *testing.B) { c.restore(b, size.copy.Plan) })
		b.Run(size.name+"/delete permanently", func(b *testing.B) { c.purge(b, size.copy.Plan) })
	}
	b.Run("submit and recall", func(b *testing.B) { c.submitAndRecall(b, readyActivity(b, server, c.alex)) })
	b.Run("approve", func(b *testing.B) { c.approve(b) })
	b.Run("small/empty recycle bin", func(b *testing.B) { c.emptyBin(b, records.Small.Plan, false) })
	b.Run("large/empty recycle bin", func(b *testing.B) { c.emptyBin(b, records.Template.Plan, true) })
}

type changes struct {
	server       *apibench.Server
	alex, daniel *apibench.Client
	copies       int
}

func (c *changes) name(what string) string {
	c.copies++
	return fmt.Sprintf("Bench %s %d", what, c.copies)
}

func (c *changes) measure(b *testing.B, step func() int64) {
	b.Helper()
	var queries int64
	for b.Loop() {
		queries += step()
	}
	b.ReportMetric(float64(queries)/float64(b.N), "queries/op")
}

func (c *changes) copyPlan(b *testing.B, planID string) {
	b.Helper()
	c.measure(b, func() int64 {
		body := map[string]any{"name": c.name("plan"), "year": 2026, "includeContents": true}
		return c.alex.Do(b, http.MethodPost, "/api/v1/plans/"+planID+"/copy", body).Queries
	})
}

func (c *changes) copyCampaign(b *testing.B, t apibench.Tree) {
	b.Helper()
	c.measure(b, func() int64 {
		body := map[string]any{"planId": t.Plan, "name": c.name("campaign")}
		return c.alex.Do(b, http.MethodPost, "/api/v1/campaigns/"+t.Campaign+"/copy", body).Queries
	})
}

func (c *changes) copyInitiative(b *testing.B, t apibench.Tree) {
	b.Helper()
	c.measure(b, func() int64 {
		body := map[string]any{"campaignId": t.Campaign, "name": c.name("initiative")}
		return c.alex.Do(b, http.MethodPost, "/api/v1/initiatives/"+t.Initiative+"/copy", body).Queries
	})
}

func (c *changes) copyActivity(b *testing.B, t apibench.Tree) {
	b.Helper()
	c.measure(b, func() int64 {
		body := map[string]any{"initiativeId": t.Initiative}
		return c.alex.Do(b, http.MethodPost, "/api/v1/activities/"+t.Activity+"/copy", body).Queries
	})
}

func (c *changes) moveActivity(b *testing.B, t apibench.Tree) {
	b.Helper()
	targets := [2]string{t.ActivityTarget, ""}
	c.measure(b, func() int64 {
		a, _, err := c.server.Set().Activities.Get(b.Context(), t.ActivityMove)
		if err != nil {
			b.Fatal(err)
		}
		if targets[1] == "" {
			targets[1] = a.InitiativeID
		}
		targets[0], targets[1] = targets[1], targets[0]
		body := map[string]any{"initiativeId": targets[1], "version": a.Version}
		return c.alex.Do(b, http.MethodPost, "/api/v1/activities/"+t.ActivityMove+"/move", body).Queries
	})
}

func (c *changes) moveInitiative(b *testing.B, t apibench.Tree) {
	b.Helper()
	targets := [2]string{t.InitiativeTarget, ""}
	c.measure(b, func() int64 {
		i, _, err := c.server.Set().Campaigns.GetInitiative(b.Context(), t.InitiativeMove)
		if err != nil {
			b.Fatal(err)
		}
		if targets[1] == "" {
			targets[1] = i.CampaignID
		}
		targets[0], targets[1] = targets[1], targets[0]
		body := map[string]any{"campaignId": targets[1], "version": i.Version}
		return c.alex.Do(b, http.MethodPost, "/api/v1/initiatives/"+t.InitiativeMove+"/move", body).Queries
	})
}

// deletePlan deletes the plan and undoes the delete each time; measured is the delete or, with undo, the undo.
func (c *changes) deletePlan(b *testing.B, planID string, undo bool) {
	b.Helper()
	c.measure(b, func() int64 {
		deleted := c.alex.Do(b, http.MethodDelete, "/api/v1/plans/"+planID, nil)
		undone := c.alex.Do(b, http.MethodPost, "/api/v1/recycle-bin/"+entryID(b, deleted)+"/undo", nil)
		if undo {
			return undone.Queries
		}
		return deleted.Queries
	})
}

func (c *changes) restore(b *testing.B, planID string) {
	b.Helper()
	c.measure(b, func() int64 {
		deleted := c.alex.Do(b, http.MethodDelete, "/api/v1/plans/"+planID, nil)
		return c.alex.Do(b, http.MethodPost, "/api/v1/recycle-bin/"+entryID(b, deleted)+"/restore", nil).Queries
	})
}

// purge copies the plan, deletes the copy and measures its permanent delete from the recycle bin.
func (c *changes) purge(b *testing.B, planID string) {
	b.Helper()
	c.measure(b, func() int64 {
		copied := c.copyOf(b, planID)
		deleted := c.alex.Do(b, http.MethodDelete, "/api/v1/plans/"+copied, nil)
		return c.alex.Do(b, http.MethodDelete, "/api/v1/recycle-bin/"+entryID(b, deleted), nil).Queries
	})
}

// emptyBin deletes a copy of the plan and empties the recycle bin of that entry alone or, with all, of every
// entry in it.
func (c *changes) emptyBin(b *testing.B, planID string, all bool) {
	b.Helper()
	c.measure(b, func() int64 {
		deleted := c.alex.Do(b, http.MethodDelete, "/api/v1/plans/"+c.copyOf(b, planID), nil)
		entries := []string{entryID(b, deleted)}
		if all {
			var bin struct {
				Items []struct {
					ID string `json:"id"`
				} `json:"items"`
			}
			c.alex.Do(b, http.MethodGet, "/api/v1/recycle-bin", nil).Decode(b, &bin)
			entries = entries[:0]
			for _, item := range bin.Items {
				entries = append(entries, item.ID)
			}
		}
		return c.alex.Do(b, http.MethodPost, "/api/v1/recycle-bin/empty", map[string]any{"entryIds": entries}).Queries
	})
}

func (c *changes) copyOf(b *testing.B, planID string) string {
	b.Helper()
	var copied struct {
		Plan struct {
			ID string `json:"id"`
		} `json:"plan"`
	}
	body := map[string]any{"name": c.name("plan"), "year": 2026, "includeContents": true}
	c.alex.Do(b, http.MethodPost, "/api/v1/plans/"+planID+"/copy", body).Decode(b, &copied)
	return copied.Plan.ID
}

func (c *changes) submitAndRecall(b *testing.B, activityID string) {
	b.Helper()
	c.measure(b, func() int64 {
		a, _, err := c.server.Set().Activities.Get(b.Context(), activityID)
		if err != nil {
			b.Fatal(err)
		}
		submitted := c.alex.Do(b, http.MethodPost, "/api/v1/activities/"+activityID+"/approval/submit",
			map[string]any{"version": a.Version, "comment": ""})
		var round struct {
			Rounds []struct {
				ID string `json:"id"`
			} `json:"rounds"`
		}
		c.alex.Do(b, http.MethodGet, "/api/v1/activities/"+activityID+"/approval", nil).Decode(b, &round)
		recalled := c.alex.Do(b, http.MethodPost, "/api/v1/activities/"+activityID+"/approval/recall",
			map[string]any{"roundId": round.Rounds[0].ID, "reason": "Bench"})
		return submitted.Queries + recalled.Queries
	})
}

func (c *changes) approve(b *testing.B) {
	b.Helper()
	var waiting struct {
		Items []struct {
			Activity struct {
				ID string `json:"id"`
			} `json:"activity"`
			Round struct {
				ID    string `json:"id"`
				Level int    `json:"currentLevel"`
			} `json:"round"`
		} `json:"items"`
	}
	c.daniel.Do(b, http.MethodGet, "/api/v1/approvals", nil).Decode(b, &waiting)
	next := 0
	c.measure(b, func() int64 {
		row := waiting.Items[next]
		next++
		body := map[string]any{"roundId": row.Round.ID, "level": row.Round.Level, "comment": ""}
		return c.daniel.Do(b, http.MethodPost, "/api/v1/activities/"+row.Activity.ID+"/approval/approve", body).Queries
	})
}

func entryID(b *testing.B, deleted apibench.Response) string {
	b.Helper()
	var entry struct {
		RecycleBinEntry struct {
			ID string `json:"id"`
		} `json:"recycleBinEntry"`
	}
	deleted.Decode(b, &entry)
	return entry.RecycleBinEntry.ID
}
