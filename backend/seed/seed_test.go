package seed_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"mapp/internal/budget"
	"mapp/internal/history"
	"mapp/internal/people"
	"mapp/internal/platform/fake"
	"mapp/internal/platform/ids"
	"mapp/internal/sample"
	"mapp/internal/settings"
	"mapp/internal/stores/storestest"
	"mapp/seed"
)

var loadedAt = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// The seven demo people (032 FR-051 adds Lea Santos) of docs/product/demo-users-and-data.md, in its table's order, active, created by MAPP.
func TestLoad_SixDemoPeople_FR028(t *testing.T) {
	t.Parallel()
	db := loadedStore(t)

	want := []people.Person{
		demoPerson("U-1001", "Alex Morgan", "alex.morgan@shell.com", "Marketing administrator"),
		demoPerson("U-1002", "Priya Raman", "priya.raman@shell.com", "Marketer"),
		demoPerson("U-1003", "Daniel Okoro", "daniel.okoro@shell.com", "Marketing team leader"),
		demoPerson("U-1004", "Mei Lin", "mei.lin@shell.com", "Marketer"),
		demoPerson("U-1005", "Carlos Mendes", "carlos.mendes@shell.com", "Business champion"),
		demoPerson("U-1006", "Sara Novak", "sara.novak@shell.com", "Local finance advisor"),
		demoPerson("U-1007", "Lea Santos", "lea.santos@shell.com", "Requisitioner"),
	}
	for i, manager := range []string{"", "U-1003", "U-1001", "U-1005", "U-1001", "U-1003", ""} {
		want[i].ManagerID = manager
	}
	got := storedPeople(t, db)
	if len(got) != len(want) {
		t.Fatalf("loaded %d people, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("person %d =\n%+v\nwant\n%+v", i+1, got[i], want[i])
		}
	}
}

// The next record of each type continues after the seed; the deleted initiative's IDs are never given out again.
func TestLoad_IDsComeFromTheCounters_FR027_FR040(t *testing.T) {
	t.Parallel()
	db := loadedStore(t)

	want := map[string]string{
		ids.PrefixPerson: "U-1008", ids.PrefixActivityType: "AT-1007", ids.PrefixModel: "MOD-1003",
		ids.PrefixShellAttribute: "SA-1057", ids.PrefixShellOption: "SO-1186", ids.PrefixFinanceBudget: "FB-1004",
		ids.PrefixFinanceBudgetItem: "FBI-1009", ids.PrefixTeam: "T-1004", ids.PrefixTeamMembership: "TM-1010",
		ids.PrefixPlan: "P-1005", ids.PrefixCampaign: "C-1006", ids.PrefixInitiative: "I-1012", ids.PrefixActivity: "A-1014",
		ids.PrefixLineItem: "L-1012", ids.PrefixLogEntry: "LOG-1055", ids.PrefixBinEntry: "BIN-1002",
		ids.PrefixApprovalWorkflow: "WF-1006", ids.PrefixApprovalRound: "AR-1009",
		ids.PrefixExchangeRate: "XR-1125", ids.PrefixRateImport: "RI-1002",
		ids.PrefixApprovalAction: "AH-1016", ids.PrefixNotification: "NT-1010",
		ids.PrefixFinancialTransaction: "FT-1012", ids.PrefixFinancialImport: "FIM-1002",
		ids.PrefixApprovalDelegation: "DLG-1002", budget.PrefixGlobalBudget: "GB-1002", budget.PrefixAllocation: "BA-1003",
		ids.PrefixRole: "RL-1007", ids.PrefixRoleAssignment: "RA-1009",
		ids.PrefixPurchaseRequest: "PR-1002", ids.PrefixPurchaseRequestLine: "PRL-1002",
	}
	generator := ids.NewGenerator(fake.NewCounters(db))
	err := db.Write(t.Context(), func(ctx context.Context) error {
		for prefix, wantID := range want {
			next, err := generator.Next(ctx, prefix)
			if err != nil {
				return err
			}
			if next != wantID {
				t.Errorf("next %s ID = %s, want %s", prefix, next, wantID)
			}
		}
		return errors.New("roll back") // leave the counters as the seed left them
	})
	if err == nil || err.Error() != "roll back" {
		t.Fatalf("Write: %v", err)
	}
}

// A store that has given out an ID holds data: it is refused and left as it was.
func TestLoad_RefusesAStoreWithData_FR028(t *testing.T) {
	t.Parallel()
	db := fake.NewDB()
	set := storestest.New(db)
	err := db.Write(t.Context(), func(ctx context.Context) error {
		id, err := ids.NewGenerator(set.Counters).Next(ctx, ids.PrefixSampleFolder)
		if err != nil {
			return err
		}
		return set.Samples.PutFolder(ctx, sample.Folder{ID: id})
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	err = seed.Load(t.Context(), set, fixedClock)

	if !errors.Is(err, seed.ErrNotEmpty) {
		t.Errorf("Load = %v, want ErrNotEmpty", err)
	}
	if got := storedPeople(t, db); len(got) != 0 {
		t.Errorf("people = %+v, want none", got)
	}
}

func TestLoad_Twice_FR028(t *testing.T) {
	t.Parallel()
	db := loadedStore(t)

	err := seed.Load(t.Context(), storestest.New(db), fixedClock)

	if !errors.Is(err, seed.ErrNotEmpty) {
		t.Errorf("second Load = %v, want ErrNotEmpty", err)
	}
	if got := storedPeople(t, db); len(got) != 7 {
		t.Errorf("%d people after loading twice, want 7", len(got))
	}
}

// Every log entry is one of the demo history's, about a planning record: none about Settings, people or teams.
func TestLoad_OnlyTheHistoryStepWritesHistory_D202(t *testing.T) {
	t.Parallel()
	entries, _ := demoLog(t, loadedStore(t))
	planning := []string{"plan", "campaign", "initiative", "activity", "lineItem"}
	actions := []history.Action{history.ActionCreated, history.ActionStatusChanged, history.ActionChanged, history.ActionDeleted}
	for _, e := range entries {
		if !slices.Contains(planning, e.RecordType) || !slices.Contains(actions, e.Action) {
			t.Errorf("%s: a %s entry about a %s, want only the demo history's entries about planning records",
				entryText(e), e.Action, e.RecordType)
		}
	}
}

// seed.Loaded, which the seed command prints, says exactly what is stored.
func TestLoaded_SaysWhatIsLoaded_FR028(t *testing.T) {
	t.Parallel()
	d := loadDemo(t)
	activeOptions := 0
	for _, o := range d.options {
		if o.Active {
			activeOptions++
		}
	}
	entries, bin := demoLog(t, loadedStore(t))
	got := fmt.Sprintf("%d people, %d teams, %d models, %d activity types, %d Shell attributes with %d options, "+
		"%d currencies, %d finance budget items, %d plans, %d campaigns, %d initiatives, %d activities, %d line items, "+
		"%d approval workflows, %d approval rounds, %d approval delegation, %d roles, %d role assignments, "+
		"%d purchase request, %d activity-log entries, %d recycle-bin entry",
		len(d.people), len(d.teams), len(d.models), len(d.activityTypes), len(d.attributes), activeOptions,
		len(d.currencies), len(d.items), len(d.plans), len(d.campaigns), len(d.initiatives), len(d.activities),
		len(d.lineItems), len(d.workflows), len(d.rounds), len(d.delegations), len(d.roles), len(d.assignments),
		len(d.purchaseRequests), len(entries), len(bin))
	if got != seed.Loaded {
		t.Errorf("stored: %s\nseed.Loaded: %s", got, seed.Loaded)
	}
}

// A dependent attribute value (Region, GL code) must be linked to a chosen option of its parent attribute.
func TestLoad_AttributeValuesFitTheirAttributes_D048(t *testing.T) {
	t.Parallel()
	d := loadDemo(t)
	check := func(record string, level settings.Level, values settings.AttributeValues) {
		if values == nil {
			t.Errorf("%s: attributes are nil, want {}", record)
		}
		for attributeID, raw := range values {
			attribute, found := d.attributeByID[attributeID]
			if !found || attribute.Level != level {
				t.Errorf("%s: %s is not a %s-level attribute", record, attributeID, level)
				continue
			}
			for _, optionID := range optionIDs(t, attribute, raw) {
				option, found := d.optionByID[optionID]
				if !found || option.AttributeID != attributeID {
					t.Errorf("%s: %s is not an option of %s", record, optionID, attribute.Label)
					continue
				}
				if attribute.ParentAttributeID == "" {
					continue
				}
				chosen := optionIDs(t, d.attributeByID[attribute.ParentAttributeID], values[attribute.ParentAttributeID])
				if !slices.ContainsFunc(option.ParentOptionIDs, func(p string) bool { return slices.Contains(chosen, p) }) {
					t.Errorf("%s: %s '%s' is not linked to a chosen parent option %v", record, attribute.Label, option.Label, chosen)
				}
			}
		}
	}
	for _, p := range d.plans {
		check(p.ID, settings.LevelPlan, p.Attributes)
	}
	for _, c := range d.campaigns {
		check(c.ID, settings.LevelCampaign, c.Attributes)
	}
	for _, i := range d.initiatives {
		check(i.ID, settings.LevelInitiative, i.Attributes)
	}
	for _, a := range d.activities {
		check(a.ID, settings.LevelActivity, a.Attributes)
	}
	for _, l := range d.lineItems {
		check(l.ID, settings.LevelLineItem, l.Attributes)
	}
}

// optionIDs returns the option ids of a choice value, those of the shares for a % split; none for other types.
func optionIDs(t *testing.T, attribute settings.Attribute, raw json.RawMessage) []string {
	t.Helper()
	if raw == nil {
		return nil
	}
	var list []string
	var err error
	switch attribute.Type {
	case settings.TypeSingleChoice:
		var one string
		err = json.Unmarshal(raw, &one)
		list = []string{one}
	case settings.TypeMultipleChoice:
		err = json.Unmarshal(raw, &list)
	case settings.TypeMultipleChoiceSplit:
		var shares []settings.SplitShare
		err = json.Unmarshal(raw, &shares)
		for _, s := range shares {
			list = append(list, s.OptionID)
		}
	}
	if err != nil {
		t.Errorf("value of %s: %s: %v", attribute.Label, raw, err)
	}
	return list
}
