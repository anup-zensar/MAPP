package apibench

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"mapp/internal/history"
	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/ids"
)

const (
	alexID       = "U-1001"
	globalBudget = "GB-1001"
	gsapHeader   = "type,document_number,line,po_number,po_line,posting_date,accrual_month,amount,currency," +
		"vendor_code,vendor_name,description,swbs_code,tracking_key,line_item_id"
)

func (v *volume) people() {
	existing, err := v.set.People.List(v.ctx)
	v.must(err)
	teams, err := v.set.People.ListTeams(v.ctx)
	v.must(err)
	v.write(func(ctx context.Context, next func(string) string) error {
		return v.addPeople(ctx, next, len(existing)+1, teams)
	})
	v.progress("%d people", peopleWanted)
}

func (v *volume) addPeople(ctx context.Context, next func(string) string, from int, teams []people.Team) error {
	for n := from; n <= peopleWanted; n++ {
		p := people.Person{ID: next(ids.PrefixPerson), Name: fmt.Sprintf("Volume Person %04d", n),
			Email: fmt.Sprintf("volume.person%04d@shell.com", n), JobTitle: "Marketer", Active: true, Meta: systemMeta()}
		m := people.Membership{ID: next(ids.PrefixTeamMembership), TeamID: teams[n%len(teams)].ID, PersonID: p.ID,
			Meta: systemMeta()}
		if err := v.set.People.Put(ctx, p); err != nil {
			return err
		}
		if err := v.set.People.PutMembership(ctx, m); err != nil {
			return err
		}
	}
	return nil
}

// waitingForApproval has Priya submit activities of her volume plans until 500 wait for Daniel.
func (v *volume) waitingForApproval() {
	submitted := 0
	for n := 1; n <= lubricantsPlans && submitted < waitingActivities; n++ {
		for _, a := range v.activitiesOf(v.planNamed(fmt.Sprintf("Volume Lubricants %03d", n))) {
			body := map[string]any{"version": a.Version, "comment": ""}
			if v.priya.Send(v.tb, http.MethodPost, "/api/v1/activities/"+a.ID+"/approval/submit", body).Status == http.StatusOK {
				submitted++
			}
			if submitted == waitingActivities {
				break
			}
		}
	}
	if submitted < waitingActivities {
		v.tb.Fatalf("apibench: only %d activities could be submitted", submitted)
	}
	v.progress("%d activities waiting for approval", submitted)
}

// gsapRows imports 5,000 rows: a purchase order and an invoice for 2,500 line items of Priya's plans.
func (v *volume) gsapRows() {
	v.alex.Do(v.tb, http.MethodPost, "/api/v1/financials/imports", v.gsapFile(6))
	v.progress("%d GSAP rows", gsapRows)
}

// gsapFile is a 5,000-row GSAP extract; each series gives other document numbers, so its rows are new.
func (v *volume) gsapFile(series int) map[string]any {
	lines := []string{gsapHeader}
	for n := 1; len(lines) <= gsapRows; n++ {
		for _, li := range v.lineItemsOf(v.planNamed(fmt.Sprintf("Volume Lubricants %03d", n))) {
			number := strings.TrimPrefix(li.ID, ids.PrefixLineItem+"-")
			po := fmt.Sprintf("4%d%s", series, number)
			lines = append(lines,
				fmt.Sprintf("PO,%s,10,,,2026-02-01,,1000.00,EUR,V-1,Vendor %s,Volume,,HX%s,", po, number, number),
				fmt.Sprintf("INVOICE,5%d%s,,%s,10,2026-02-02,,400.00,EUR,V-1,Vendor %s,Volume,,,", series-4, number, po, number))
			if len(lines) > gsapRows {
				break
			}
		}
	}
	content := base64.StdEncoding.EncodeToString([]byte(strings.Join(lines, "\n") + "\n"))
	return map[string]any{"fileName": "volume.csv", "content": content}
}

// GSAPFile is a 5,000-row GSAP extract of new transactions for the volume's line items: series 7 to 9.
func (s *Server) GSAPFile(tb testing.TB, series int) map[string]any {
	tb.Helper()
	v := &volume{tb: tb, ctx: tb.Context(), set: s.set}
	return v.gsapFile(series)
}

// allocations allocates the global budget to every Lubricants volume plan.
func (v *volume) allocations() {
	for n := 1; n <= lubricantsPlans; n++ {
		planID := v.planNamed(fmt.Sprintf("Volume Lubricants %03d", n))
		p, _, err := v.set.Plans.Get(v.ctx, planID)
		v.must(err)
		amount := map[string]string{fieldAmount: "10000.00", fieldCurrency: usd}
		body := map[string]any{fieldPlanID: planID, fieldAmount: amount, "planVersion": p.Version}
		v.alex.Do(v.tb, http.MethodPost, "/api/v1/global-budgets/"+globalBudget+"/allocations", body)
	}
	v.progress("%d allocations", lubricantsPlans)
}

// recycleBin deletes activities of the last Global plans, so the recycle bin holds 100 entries.
func (v *volume) recycleBin() {
	deleted := 0
	for n := globalPlansPerYear; deleted < binnedActivities; n-- {
		planID := v.planNamed(fmt.Sprintf("Volume Global %d %03d", lastGlobalYear, n))
		for _, a := range v.activitiesOf(planID)[:activitiesPerTarget] {
			v.alex.Do(v.tb, http.MethodDelete, "/api/v1/activities/"+a.ID, nil)
			deleted++
		}
	}
	v.progress("%d activities in the recycle bin", deleted)
}

// logEntries adds entries about the template's activities until the log holds 50,000.
func (v *volume) logEntries() {
	existing, err := v.set.History.Entries(v.ctx)
	v.must(err)
	paths := v.activityPaths(v.planNamed(TemplatePlan))
	v.write(func(ctx context.Context, _ func(string) string) error {
		return v.appendEntries(ctx, len(existing), paths)
	})
	v.progress("%d log entries", logEntriesWanted)
}

// appendEntries appends entries from number from on, up to logEntriesWanted, about each of paths in turn.
func (v *volume) appendEntries(ctx context.Context, from int, paths [][]history.RecordRef) error {
	generator := ids.NewGenerator(v.set.Counters)
	for n := from; n < logEntriesWanted; n++ {
		seq, err := generator.NextNumber(ctx, ids.PrefixLogEntry)
		if err != nil {
			return err
		}
		if err := v.set.History.AppendEntry(ctx, logEntry(seq, paths[n%len(paths)])); err != nil {
			return err
		}
	}
	return nil
}

func logEntry(seq int, path []history.RecordRef) history.Entry {
	last := path[len(path)-1]
	return history.Entry{
		ID: ids.PrefixLogEntry + "-" + strconv.Itoa(seq), Seq: seq, At: startTime(),
		Actor:  actor.Actor{ID: "U-1002", Name: "Priya Raman"},
		Action: history.ActionChanged, Verb: string(history.ActionChanged), RecordType: last.Type,
		RecordTypeLabel: last.Type, RecordID: last.ID, RecordName: last.Name, Path: path[:len(path)-1],
		Changes: []history.Change{},
	}
}

// activityPaths returns, for every activity of the plan, its path from the plan down to the activity itself.
func (v *volume) activityPaths(planID string) [][]history.RecordRef {
	p, _, err := v.set.Plans.Get(v.ctx, planID)
	v.must(err)
	plan := history.RecordRef{Type: "plan", ID: p.ID, Name: p.Name}
	campaignOf := map[string]history.RecordRef{}
	for _, c := range v.campaignsOf(planID) {
		campaignOf[c.ID] = history.RecordRef{Type: "campaign", ID: c.ID, Name: c.Name}
	}
	initiativeOf := map[string][]history.RecordRef{}
	for _, i := range v.initiativesOf(planID) {
		initiativeOf[i.ID] = []history.RecordRef{plan, campaignOf[i.CampaignID], {Type: "initiative", ID: i.ID, Name: i.Name}}
	}
	var paths [][]history.RecordRef
	for _, a := range v.activitiesOf(planID) {
		path := append(append([]history.RecordRef{}, initiativeOf[a.InitiativeID]...),
			history.RecordRef{Type: "activity", ID: a.ID, Name: a.Name})
		paths = append(paths, path)
	}
	return paths
}
