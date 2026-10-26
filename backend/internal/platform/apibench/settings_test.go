package apibench_test

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"mapp/internal/platform/apibench"
)

type change struct {
	name, path string
	body       any
}

// settingsPages are the pages of Settings, people and teams that read the records using them ("Used by").
var settingsPages = []page{
	{"settings people", alex, "/api/v1/people/U-1002"},
	{"settings team", alex, "/api/v1/teams/T-1001"},
	{"display currencies", priya, "/api/v1/me/display-currencies"},
	{"access changes", alex, "/api/v1/access/changes"},
	{"access rules", alex, "/api/v1/access-rules"},
	{"activity form", alex, "/api/v1/shell-attribute-forms/activity"},
	{"current rates", alex, "/api/v1/exchange-rates/current"},
	{"euro rates", alex, "/api/v1/currencies/EUR/rates"},
	{"rate imports", alex, "/api/v1/exchange-rates/imports"},
}

var settingsChecks = []change{
	{"model impact", "/api/v1/models/MOD-1001/impact", map[string]any{"inqToMqlRate": "3", "version": 1}},
	{"activity type impact", "/api/v1/activity-types/AT-1002/impact", map[string]any{"modelIds": []string{"MOD-1002"}, "version": 1}},
	{"attribute impact", "/api/v1/shell-attributes/SA-1008/impact", map[string]any{"marking": "neededToPublish", "version": 1}},
	{"currency impact", "/api/v1/currencies/EUR/impact", map[string]any{"rate": "0.95", "version": 1}},
	{"rate impact", "/api/v1/currencies/EUR/rates/impact", map[string]any{"date": "2026-02-01", "rate": "0.95"}},
	{"person impact", "/api/v1/people/U-1002/impact", map[string]any{"teamIds": []string{}, "version": 1}},
	{"team impact", "/api/v1/teams/T-1001/impact", map[string]any{"memberIds": []string{}, "version": 1}},
	{"roles impact", "/api/v1/people/U-1002/roles/impact", map[string]any{"roles": []any{}, "version": 1}},
	{"rate import check", "/api/v1/exchange-rates/import-check", rateFile(1000)},
}

// BenchmarkSettings opens the pages BenchmarkLists leaves out and asks the impact checks, on both databases.
func BenchmarkSettings(b *testing.B) {
	for _, data := range []apibench.Data{apibench.Demo, apibench.Volume} {
		server := apibench.New(b, data)
		clients := signIn(b, server)
		for _, p := range settingsPages {
			b.Run(string(data)+"/"+p.name, func(b *testing.B) { clients[p.who].Measure(b, http.MethodGet, p.path, nil) })
		}
		for _, c := range settingsChecks {
			b.Run(string(data)+"/"+c.name, func(b *testing.B) { clients[alex].Measure(b, http.MethodPost, c.path, c.body) })
		}
	}
}

// BenchmarkSettingsImports imports a file of 1,000 rates and one of 1,000 options into a Shell attribute.
func BenchmarkSettingsImports(b *testing.B) {
	server := apibench.New(b, apibench.Volume)
	client := server.SignIn(b, alex)
	b.Run("rate import", func(b *testing.B) {
		client.Measure(b, http.MethodPost, "/api/v1/exchange-rates/import", rateFile(1000))
	})
	b.Run("option import", func(b *testing.B) {
		client.Measure(b, http.MethodPost, "/api/v1/shell-attributes/SA-1003/options/import", optionFile(1000))
	})
}

func optionFile(rows int) map[string]string {
	lines := []string{"Option"}
	for n := range rows {
		lines = append(lines, fmt.Sprintf("Volume cluster %04d", n))
	}
	content := base64.StdEncoding.EncodeToString([]byte(strings.Join(lines, "\n") + "\n"))
	return map[string]string{"fileName": "options.csv", "content": content}
}

// rateFile is a GSAP daily rate file of rows rows: a rate a day for each of EUR, GBP, CNY and INR.
func rateFile(rows int) map[string]string {
	lines := []string{"rate_type,from_currency,to_currency,valid_from,rate"}
	first := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	codes := []string{"EUR", "GBP", "CNY", "INR"}
	for n := range rows {
		day := first.AddDate(0, 0, n/len(codes)).Format(time.DateOnly)
		lines = append(lines, fmt.Sprintf("M,USD,%s,%s,%d.%03d", codes[n%len(codes)], day, 1+n%7, n%1000))
	}
	content := base64.StdEncoding.EncodeToString([]byte(strings.Join(lines, "\n") + "\n"))
	return map[string]string{"fileName": "rates.csv", "content": content}
}
