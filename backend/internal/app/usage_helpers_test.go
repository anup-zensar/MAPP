package app_test

import (
	"net/http"
	"testing"

	"mapp/internal/platform/apitest"
)

type usage struct {
	InUse   bool
	Summary string
	Records []recordRef
}

type recordRef struct {
	Type, ID, Name string
}

type usageCount struct {
	Role, RecordType, Label string
	Count                   int
}

func usageAt(t *testing.T, c *apitest.Client, path string) usage {
	t.Helper()
	response := c.Do(t, http.MethodGet, path, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET %s = %d %s, want 200", path, response.Status, response.Body)
	}
	var record struct{ Usage usage }
	response.Decode(t, &record)
	return record.Usage
}

func summaries(t *testing.T, c *apitest.Client, paths []string) map[string]string {
	t.Helper()
	got := map[string]string{}
	for _, path := range paths {
		got[path] = usageAt(t, c, path).Summary
	}
	return got
}

func personUsage(t *testing.T, c *apitest.Client, personID string) []usageCount {
	t.Helper()
	var person struct{ Usage []usageCount }
	c.Do(t, http.MethodGet, "/api/v1/people/"+personID, nil).Decode(t, &person)
	return person.Usage
}

func deletePlan(t *testing.T, c *apitest.Client, id string) string {
	t.Helper()
	response := c.Do(t, http.MethodDelete, "/api/v1/plans/"+id, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("DELETE plan %s = %d %s, want 200", id, response.Status, response.Body)
	}
	var deleted struct{ RecycleBinEntry struct{ ID string } }
	response.Decode(t, &deleted)
	return deleted.RecycleBinEntry.ID
}

// settingsPaths: one Settings record of every kind that features 003–007 use.
var settingsPaths = []string{
	"/api/v1/models/MOD-1002",
	"/api/v1/activity-types/AT-1006",
	"/api/v1/shell-attributes/SA-1001",
	"/api/v1/shell-attributes/SA-1003",
	"/api/v1/shell-attributes/SA-1011",
	"/api/v1/shell-attributes/SA-1012",
	"/api/v1/shell-attributes/SA-1033",
	"/api/v1/currencies/USD",
	"/api/v1/currencies/CNY",
	"/api/v1/finance-budget-items/FBI-1001",
}
