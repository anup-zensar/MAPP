package home_test

import (
	"net/http"
	"testing"
	"time"

	"mapp/internal/platform/apitest"
	"mapp/internal/platform/money"
)

// The demo people's e-mail addresses (demo-users-and-data.md).
const (
	alexsEmail   = "alex.morgan@shell.com"
	priyasEmail  = "priya.raman@shell.com"
	danielsEmail = "daniel.okoro@shell.com"
	meisEmail    = "mei.lin@shell.com"
	carlosEmail  = "carlos.mendes@shell.com"
	sarasEmail   = "sara.novak@shell.com"
)

// everyone is every seeded person.
var everyone = []string{alexsEmail, priyasEmail, danielsEmail, meisEmail, carlosEmail, sarasEmail}

// now is the time of the test server's clock: 12 Mar 2026 09:00 UTC. The seeded approvals are dated before
// it.
var now = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// listScopes are Home's scopes with the matching scope of the Plans list (feature 004).
var listScopes = map[string]string{"myTeams": "myTeams", "everything": "all"}

// get sends GET path as client, checks that it answers 200 and decodes the answer into a T.
func get[T any](t *testing.T, client *apitest.Client, path string, opts ...apitest.RequestOption) T {
	t.Helper()
	response := client.Do(t, http.MethodGet, path, nil, opts...)
	if response.Status != http.StatusOK {
		t.Fatalf("GET %s = %d %s, want 200", path, response.Status, response.Body)
	}
	var v T
	response.Decode(t, &v)
	return v
}

// usd returns text as US dollars, such as "460000.00".
func usd(t *testing.T, text string) money.Money {
	t.Helper()
	amount, err := money.ParseAmount(text)
	if err != nil {
		t.Fatalf("ParseAmount(%q): %v", text, err)
	}
	return money.USD(amount)
}

// orZero is m, or 0.00 USD when there is none.
func orZero(m *money.Money) money.Money {
	if m == nil {
		return money.USD(money.Amount{})
	}
	return *m
}

// same reports whether two optional values, such as amounts or percentages, are equal.
func same[T comparable](x, y *T) bool {
	return (x == nil && y == nil) || (x != nil && y != nil && *x == *y)
}
