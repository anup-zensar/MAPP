package seed

import (
	"context"
	"encoding/json"
	"fmt"

	"mapp/internal/activities"
	"mapp/internal/approvals"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// The demo data of feature 017, approval rules (017 data-model §13, FR-037): line managers, the reminder,
// escalation and backup settings of the two seeded workflows, a third workflow "Standard marketing
// approval" showing every rule type, and one planned delegation. Seeded statuses, rounds and notifications
// stay as feature 008 has them.

// demoManagers are the line managers of the demo people, by name (017 data-model §13). Alex Morgan has none.
var demoManagers = map[string]string{
	"Priya Raman":   "Daniel Okoro",
	"Sara Novak":    "Daniel Okoro",
	"Mei Lin":       "Carlos Mendes",
	"Daniel Okoro":  "Alex Morgan",
	"Carlos Mendes": "Alex Morgan",
}

// addLineManagers sets the line managers once every demo person exists (the end of the people step, D-306).
func (d *demo) addLineManagers(ctx context.Context) error {
	repo := d.stores.People
	for _, p := range demoPeople {
		manager, has := demoManagers[p.name]
		if !has {
			continue
		}
		var r resolver
		id, managerID := r.id(d.people, "person", p.name), r.id(d.people, "person", manager)
		if r.err != nil {
			return r.err
		}
		person, _, err := repo.Get(ctx, id)
		if err != nil {
			return err
		}
		person.ManagerID = managerID
		if err := repo.Put(ctx, person); err != nil {
			return err
		}
	}
	return nil
}

// demoTimings are the reminder and escalation days and the backup approver of each level (by name) of the
// two workflows feature 006 seeds: Lubricants is short, so a demo shows a reminder and an escalation on the
// first run.
var demoTimings = map[string]struct {
	reminder, escalation int
	backups              []string
}{
	lubricantsFlow: {1, 2, []string{"Alex Morgan"}},
	mobilityFlow:   {3, 5, []string{"Alex Morgan", "Carlos Mendes"}},
}

// standardFlow is the third demo workflow, which no seeded activity uses.
const standardFlow = "Standard marketing approval"

// addApprovalRules gives WF-1001 and WF-1002 their timings and backups, and creates WF-1003 "Standard
// marketing approval": level 1 the owner's line manager, approved automatically below 10,000.00 USD; level 2
// Alex Morgan from 100,000.00 USD; level 3 Daniel Okoro and Alex Morgan, all must approve, for Mobility.
func (d *demo) addApprovalRules(ctx context.Context) error {
	repo := d.stores.Approvals
	for name, timing := range demoTimings {
		var r resolver
		id := r.id(d.workflows, "approval workflow", name)
		if r.err != nil {
			return r.err
		}
		w, _, err := repo.GetWorkflow(ctx, id)
		if err != nil {
			return err
		}
		for i, backup := range timing.backups {
			w.Levels[i].BackupApproverID = r.id(d.people, "person", backup)
		}
		if r.err != nil {
			return r.err
		}
		w.ReminderDays, w.EscalationDays = days(timing.reminder), days(timing.escalation)
		if err := repo.PutWorkflow(ctx, w); err != nil {
			return err
		}
	}

	var r resolver
	alex, carlos, daniel := r.id(d.people, "person", "Alex Morgan"), r.id(d.people, "person", "Carlos Mendes"), r.id(d.people, "person", "Daniel Okoro")
	mobility := r.id(d.settings.Options, "Business unit option", optionKey(settings.LevelPlan, "Business unit", "Mobility"))
	if r.err != nil {
		return r.err
	}
	below, err := money.ParseAmount("10000.00")
	if err != nil {
		return err
	}
	atLeast, err := money.ParseAmount("100000.00")
	if err != nil {
		return err
	}
	id, err := d.next(ctx, ids.PrefixApprovalWorkflow)
	if err != nil {
		return err
	}
	none := []string{}
	workflow := approvals.Workflow{
		ID: id, Name: standardFlow, Active: true, ReminderDays: days(3), EscalationDays: days(5), Meta: d.meta(),
		Description: "Approved automatically below 10,000 USD, otherwise by the owner's line manager. The marketing administrator " +
			"joins from 100,000 USD; Mobility activities also need a second check by two people.",
		Levels: []approvals.Level{
			{Name: "Line manager", ApproverType: approvals.SourceManager, ApproverIDs: none, Mode: approvals.ModeAny,
				AutoApproveWhen:  &approvals.Condition{PlannedCostBelow: &below, BusinessUnitIDs: none, ActivityTypeIDs: none},
				BackupApproverID: alex},
			{Name: "Marketing administrator", ApproverType: approvals.SourcePeople, ApproverIDs: []string{alex}, Mode: approvals.ModeAny,
				AppliesWhen:      &approvals.Condition{PlannedCostAtLeast: &atLeast, BusinessUnitIDs: none, ActivityTypeIDs: none},
				BackupApproverID: carlos},
			{Name: "Mobility second check", ApproverType: approvals.SourcePeople, ApproverIDs: []string{daniel, alex}, Mode: approvals.ModeAll,
				AppliesWhen:      &approvals.Condition{BusinessUnitIDs: []string{mobility}, ActivityTypeIDs: none},
				BackupApproverID: carlos},
		},
	}
	if err := repo.PutWorkflow(ctx, workflow); err != nil {
		return err
	}
	d.workflows[standardFlow] = id
	return nil
}

// days returns a pointer to n days.
func days(n int) *int {
	return &n
}

// roundRules fills the fields feature 017 adds to a seeded round (017 data-model §13): the workflow's
// timings, each level decided by its named approvers where any one approves with the workflow's backup
// approver, and the values the rules use of the activity.
func (d *demo) roundRules(ctx context.Context, round *approvals.Round, workflow approvals.Workflow, act activities.Activity, planID string) error {
	round.ReminderDays, round.EscalationDays = workflow.ReminderDays, workflow.EscalationDays
	peopleRepo := d.stores.People
	for i := range round.Levels {
		l := &round.Levels[i]
		l.ApproverType, l.Mode, l.Resolution, l.Excluded = approvals.SourcePeople, approvals.ModeAny, approvals.ResolutionDecide, []approvals.Excluded{}
		backupID := workflow.Levels[i].BackupApproverID
		if backupID == "" || named(l.Approvers, backupID) {
			continue
		}
		p, _, err := peopleRepo.Get(ctx, backupID)
		if err != nil {
			return err
		}
		backup := p.Actor()
		l.Backup = &backup
	}
	facts, err := d.ruleFacts(ctx, act, planID)
	if err != nil {
		return err
	}
	round.Facts = &facts
	return nil
}

// named reports whether personID is one of actors.
func named(actors []actor.Actor, personID string) bool {
	for _, a := range actors {
		if a.ID == personID {
			return true
		}
	}
	return false
}

// ruleFacts are the values of act the approval rules look at: its planned cost, its activity type and the
// business units of its plan, planID.
func (d *demo) ruleFacts(ctx context.Context, act activities.Activity, planID string) (approvals.RuleFacts, error) {
	facts := approvals.RuleFacts{PlannedCost: act.PlannedCost, BusinessUnits: []approvals.FactOption{}}
	settingsRepo := d.stores.Settings
	if act.ActivityTypeID != "" {
		t, found, err := settingsRepo.GetActivityType(ctx, act.ActivityTypeID)
		if err != nil {
			return approvals.RuleFacts{}, err
		}
		if found {
			facts.ActivityType = &approvals.FactRef{ID: t.ID, Name: t.Name}
		}
	}
	plan, found, err := d.stores.Plans.Get(ctx, planID)
	if err != nil || !found {
		return approvals.RuleFacts{}, cmpErr(err, fmt.Errorf("seed: plan %s is not stored", planID))
	}
	var units []string
	if raw, found := plan.Attributes[d.settings.Attributes["plan/Business unit"]]; found {
		if err := json.Unmarshal(raw, &units); err != nil {
			return approvals.RuleFacts{}, err
		}
	}
	for _, id := range units {
		o, found, err := settingsRepo.GetOption(ctx, id)
		if err != nil {
			return approvals.RuleFacts{}, err
		}
		if found {
			facts.BusinessUnits = append(facts.BusinessUnits, approvals.FactOption{ID: o.ID, Label: o.Label})
		}
	}
	return facts, nil
}

// addDelegation writes DLG-1001: Daniel Okoro delegates his approvals to Carlos Mendes from a week after
// the seed time for five days, "Annual leave" — planned, so it changes no list today. Daniel created it the
// day before.
func (d *demo) addDelegation(ctx context.Context) error {
	var r resolver
	daniel, carlos := r.id(d.people, "person", "Daniel Okoro"), r.id(d.people, "person", "Carlos Mendes")
	if r.err != nil {
		return r.err
	}
	p, _, err := d.stores.People.Get(ctx, daniel)
	if err != nil {
		return err
	}
	id, err := d.next(ctx, ids.PrefixApprovalDelegation)
	if err != nil {
		return err
	}
	today := date.Today(d.at)
	return d.stores.Approvals.PutDelegation(ctx, approvals.Delegation{
		ID: id, PersonID: daniel, DelegateID: carlos, StartDate: today.AddDays(7), EndDate: today.AddDays(11), Note: "Annual leave",
		Meta: record.NewMeta(p.Actor(), d.at.Add(-day)),
	})
}
