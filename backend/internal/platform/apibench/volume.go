package apibench

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/database"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
	"mapp/internal/stores"
	"mapp/seed"
)

// volumeVersion is part of the databases' names: a change to what they hold needs a new one.
const volumeVersion = "1"

// The volume on top of the demo data, after the specs' scale targets: Priya's team holds 50 plans of 2026
// with 5,000 activities and 20,000 line items, one plan with 500 activities; Global Marketing Ops holds 50
// plans in each of 2027 to 2030; 1,000 people, 5,000 GSAP rows and 50,000 log entries.
const (
	lubricantsPlans     = 48
	activitiesPerTarget = 10
	initiativesPerGroup = 5
	lineItemsPerTarget  = 4
	bigPlanCampaigns    = 5
	globalPlansPerYear  = 50
	firstGlobalYear     = 2027
	lastGlobalYear      = 2030
	peopleWanted        = 1000
	waitingActivities   = 5
	gsapRows            = 5000
	logEntriesWanted    = 50000
	binnedActivities    = 100
)

// Names of plans the benchmarks open: the template of the Lubricants plans and the plan of 500 activities.
const (
	TemplatePlan = "Volume Lubricants 001"
	BigPlan      = "Volume Lubricants big"
)

const (
	fieldName     = "name"
	fieldPlanID   = "planId"
	fieldAmount   = "amount"
	fieldCurrency = "currency"
	usd           = "USD"
)

type volume struct {
	tb      testing.TB
	ctx     context.Context
	set     stores.Set
	alex    *Client
	priya   *Client
	started time.Time
}

func loadDemo(tb testing.TB, db *database.DB) {
	tb.Helper()
	if err := seed.Load(tb.Context(), stores.Postgres(db), startTime); err != nil {
		tb.Fatalf("apibench: loading the demo data: %v", err)
	}
}

func loadVolume(tb testing.TB, db *database.DB) {
	tb.Helper()
	loadDemo(tb, db)
	s := serve(tb, db)
	v := &volume{tb: tb, ctx: tb.Context(), set: s.set, alex: s.SignIn(tb, alex), priya: s.SignIn(tb, priya), started: time.Now()}
	for _, step := range []func(){
		v.lubricantsPlans, v.globalPlans, v.people, v.waitingForApproval, v.gsapRows, v.allocations,
		v.recycleBin, v.logEntries,
	} {
		step()
	}
}

// progress tells the person waiting for the first build how far it is: it takes minutes.
func (v *volume) progress(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "apibench %6.0fs: %s\n", time.Since(v.started).Seconds(), fmt.Sprintf(format, args...))
}

func (v *volume) copyPlan(id, name string, year int) string {
	var copied struct {
		Plan struct {
			ID string `json:"id"`
		} `json:"plan"`
	}
	body := map[string]any{fieldName: name, "year": year, "includeContents": true}
	v.alex.Do(v.tb, http.MethodPost, "/api/v1/plans/"+id+"/copy", body).Decode(v.tb, &copied)
	return copied.Plan.ID
}

func (v *volume) write(fn func(ctx context.Context, next func(prefix string) string) error) {
	generator := ids.NewGenerator(v.set.Counters)
	err := v.set.Tx.Write(v.ctx, func(ctx context.Context) error {
		next := func(prefix string) string {
			id, err := generator.Next(ctx, prefix)
			if err != nil {
				v.tb.Fatalf("apibench: next %s: %v", prefix, err)
			}
			return id
		}
		return fn(ctx, next)
	})
	if err != nil {
		v.tb.Fatalf("apibench: writing the volume: %v", err)
	}
}

func (v *volume) must(err error) {
	if err != nil {
		v.tb.Fatalf("apibench: reading the volume: %v", err)
	}
}

func systemMeta() record.Meta {
	return record.NewMeta(actor.System, startTime())
}
