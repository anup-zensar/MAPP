package activity_test

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

	"mapp/internal/activity"
	"mapp/internal/campaign/campaigntest"
	"mapp/internal/history"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
)

func changeTexts(e history.Entry) []string {
	var texts []string
	for _, c := range e.Changes {
		texts = append(texts, c.Label+": "+c.OldText+" → "+c.NewText)
	}
	return texts
}

// A save writes one "changed" entry with readable old and new values, one change per Shell attribute.
func TestUpdate_LogsEveryChange_FR026(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.clock.Advance(time.Hour)

	view := f.mustEdit(t, "A-1001", func(u *activity.Update) {
		u.Description = "Hero film for the Helix launch."
		u.OwnerID = text(daniel.ID)
		u.ActivityTypeID = eventType
		u.Attributes[companyCode] = json.RawMessage(`"SO-1040"`)
		u.Attributes[countrySplit] = split("SO-1034", "100")
	})

	if view.Version != 2 || view.UpdatedBy != priya || !view.UpdatedAt.Equal(start.Add(time.Hour)) {
		t.Errorf("version %d, updated by %v at %v", view.Version, view.UpdatedBy, view.UpdatedAt)
	}
	entries := f.entries(t)
	if len(entries) != 1 || entries[0].Action != history.ActionChanged {
		t.Fatalf("entries = %+v, want one changed entry", entries)
	}
	want := []string{
		"Activity type: Content production → Event",
		"Country / OU: Germany 50.00 %, United Kingdom 30.00 %, Netherlands 20.00 % → Germany 100.00 %",
		"Company code: Not filled in → DE03",
		"Description: Not filled in → Hero film for the Helix launch.",
		"Owner: Priya Raman → Daniel Okoro",
	}
	if got := changeTexts(entries[0]); !slices.Equal(got, want) {
		t.Errorf("changes =\n%q\nwant\n%q", got, want)
	}
}

// Dates, money, numbers, yes/no, workflow and status read well in the log too.
func TestDescribe_ReadableChanges_FR026(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	mql := int64(4_000)

	f.mustEdit(t, "A-1001", func(u *activity.Update) {
		u.StartDate, u.EndDate = text("2026-03-02"), text("2026-06-30")
		u.PlannedCost = usdInput("150000")
		u.ApprovalWorkflowID = text(chinaFlow)
		u.Goals.SetLaterStagesMyself, u.Goals.MQL, u.Goals.Revenue = true, &mql, usdInput("1000000")
	})

	want := []string{
		"Approval workflow: Lubricants EU approval → China BC approval workflow",
		"End date: 31 May 2026 → 30 Jun 2026",
		"MQL: Not filled in → 4,000",
		"Planned cost: 120,000.00 USD → 150,000.00 USD",
		"Revenue: Not filled in → 1,000,000.00 USD",
		"Set later stages myself: No → Yes",
		"Start date: 01 Mar 2026 → 02 Mar 2026",
	}
	if got := changeTexts(f.entries(t)[0]); !slices.Equal(got, want) {
		t.Errorf("changes =\n%q\nwant\n%q", got, want)
	}
}

func TestUpdate_NoChangeNoLog_FR026(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	view := f.mustEdit(t, "A-1001", func(u *activity.Update) { u.Name = " Launch film " })

	if view.Version != 1 || len(f.entries(t)) != 0 {
		t.Errorf("version %d, %d entries; want no change", view.Version, len(f.entries(t)))
	}
}

// A save from an out-of-date version is refused, naming who changed the activity and when.
func TestUpdate_VersionConflict_FR026(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	form := formOf(f.get(t, "A-1001"))
	f.clock.Advance(90 * time.Minute)
	f.mustEdit(t, "A-1001", func(u *activity.Update) { u.Description = "Daniel's change" })
	form.Description = "Priya's change"

	_, err := f.service.Update(t.Context(), daniel, "A-1001", form)

	assertError(t, err, errs.VersionConflict("activity", "Priya Raman", start.Add(90*time.Minute)))
	if stored, _ := f.stored(t, "A-1001"); stored.Description != "Daniel's change" {
		t.Errorf("description = %q, want the first save kept", stored.Description)
	}
}

// Without a version, or with one below 1, the save is refused under "version" with every other problem.
func TestUpdate_MissingVersion_FR026(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	form := formOf(f.get(t, "A-1001"))
	form.Version, form.Name = nil, ""

	_, err := f.service.Update(t.Context(), priya, "A-1001", form)

	assertError(t, err, problems("name", "Enter a name.", "version", "Reload the page and try again."))
}

// On a Submitted activity, a change of cost, dates or ongoing is refused until the person confirms re-approval.
func TestUpdate_ReapprovalNeedsConfirmation_FR027(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	changes := map[string]func(u *activity.Update){
		"planned cost": func(u *activity.Update) { u.PlannedCost = usdInput("35000.00") },
		"start date":   func(u *activity.Update) { u.StartDate = text("2026-04-02") },
		"end date":     func(u *activity.Update) { u.EndDate = text("2026-09-29") },
		"ongoing":      func(u *activity.Update) { u.Ongoing = true },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			_, err := f.edit(t, priya, "A-1004", change)

			assertError(t, err, problems("restartApproval", "This activity will need approval again — confirm the change to save it."))
		})
	}
	if stored, _ := f.stored(t, "A-1004"); stored.Version != 1 || stored.Status != activity.StatusSubmitted || len(f.entries(t)) != 0 {
		t.Errorf("stored = version %d, %s; want nothing saved", stored.Version, stored.Status)
	}
}

// Confirmed, the change saves with a "changed" and a status entry, and line items and approvals are told.
func TestUpdate_ConfirmedReapprovalReturnsToCreated_FR027(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	view := f.mustEdit(t, "A-1006", func(u *activity.Update) {
		u.PlannedCost = usdInput("30000.00")
		u.EndDate = text("2026-12-20")
		u.RestartApproval = true
	})

	if view.Status != activity.StatusCreated || view.DisplayStatus != activity.DisplayCreated || view.Version != 2 {
		t.Errorf("status %s (%s), version %d; want Created at version 2", view.Status, view.DisplayStatus, view.Version)
	}
	entries := f.entries(t)
	if len(entries) != 2 || entries[1].Action != history.ActionChanged || entries[0].Action != history.ActionStatusChanged {
		t.Fatalf("entries = %+v, want changed, then statusChanged", entries)
	}
	if got := changeTexts(entries[1]); !slices.Equal(got, []string{"End date: 15 Dec 2026 → 20 Dec 2026",
		"Planned cost: 25,000.00 USD → 30,000.00 USD"}) {
		t.Errorf("changed entry = %q", got)
	}
	status := entries[0]
	if status.Verb != "restarted approval of" || status.Details != "Approval restarted because the planned cost and the end date changed." ||
		!slices.Equal(changeTexts(status), []string{"Status: Approved → Created"}) {
		t.Errorf("status entry = %+v", status)
	}
	if !slices.Equal(f.hooks.restarted, []string{"A-1006: approved → created"}) || !slices.Equal(f.lineItems.datesChanged, []string{"A-1006"}) {
		t.Errorf("approval hooks %q, line items told %q", f.hooks.restarted, f.lineItems.datesChanged)
	}
}

// Other edits keep a Submitted or Approved activity's status, and a Declined one stays Declined.
func TestUpdate_OtherEditsKeepStatus_FR028(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	inq := int64(10_000)

	approved := f.mustEdit(t, "A-1002", func(u *activity.Update) {
		u.Name, u.ExternalName, u.Description = "Social always-on EU", "Social EU", "Paid social."
		u.OwnerID = text(daniel.ID)
		u.Attributes["SA-1021"] = json.RawMessage(`["SO-1062"]`)
		u.Goals.INQ = &inq
	})
	declined := f.mustEdit(t, "A-1005", func(u *activity.Update) { u.PlannedCost = usdInput("55000.00") })

	if approved.Status != activity.StatusApproved || approved.Version != 2 || declined.Status != activity.StatusDeclined {
		t.Errorf("statuses %s, %s; want Approved and Declined kept", approved.Status, declined.Status)
	}
	if len(f.hooks.restarted) != 0 {
		t.Errorf("approval restarted for %q", f.hooks.restarted)
	}
}

func TestUpdate_OngoingCannotBeUndone_FR010(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	_, err := f.edit(t, priya, "A-1007", func(u *activity.Update) {
		u.Ongoing, u.EndDate, u.RestartApproval = false, text("2026-09-30"), true
	})

	assertError(t, err, problems("ongoing", "Ongoing can't be undone — to plan with an end date, create a new activity."))
}

// The approval workflow can change only while the activity is Created or Declined.
func TestUpdate_WorkflowLockedWhenSubmittedOrApproved_FR012(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	_, err := f.edit(t, priya, "A-1002", func(u *activity.Update) { u.ApprovalWorkflowID = text(chinaFlow) })

	assertError(t, err, problems("approvalWorkflowId", "The approval workflow can't change while the activity is submitted or approved."))
	if view := f.mustEdit(t, "A-1005", func(u *activity.Update) { u.ApprovalWorkflowID = nil }); view.ApprovalWorkflow != nil {
		t.Errorf("declined activity: workflow %+v, want removed", view.ApprovalWorkflow)
	}
}

// While the activity is Submitted, the workflow field says to recall it first; nothing is saved.
func TestWorkflowFieldLockReason_FR013(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	_, chosen := f.edit(t, priya, "A-1004", func(u *activity.Update) { u.ApprovalWorkflowID = text(chinaFlow) })
	_, removed := f.edit(t, priya, "A-1004", func(u *activity.Update) { u.ApprovalWorkflowID = nil })

	locked := problems("approvalWorkflowId", "Recall the activity to choose another workflow.")
	assertError(t, chosen, locked)
	assertError(t, removed, locked)
	if stored, _ := f.stored(t, "A-1004"); stored.Version != 1 || stored.ApprovalWorkflowID != lubricantsFlow {
		t.Errorf("stored = version %d, workflow %s; want nothing saved", stored.Version, stored.ApprovalWorkflowID)
	}
}

// Inactive values already chosen are kept by other edits; choosing an inactive one anew is refused.
func TestUpdate_InactiveValuesKeptWhenUnchanged_FR013(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.deactivateType(t, contentType)
	f.deactivatePerson(t, priya.ID)
	f.deactivateWorkflow(t, lubricantsFlow)
	f.deactivateOption(t, "SO-1034") // Germany, in A-1001's Country / OU split

	view := f.mustEdit(t, "A-1001", func(u *activity.Update) { u.Description = "Still fine." })

	if !view.ActivityType.Allowed || view.ActivityType.Active || view.Owner.Active || view.ApprovalWorkflow.Active {
		t.Errorf("kept values: type %+v, owner %+v, workflow %+v", view.ActivityType, view.Owner, view.ApprovalWorkflow)
	}
	_, err := f.edit(t, priya, "A-1008", func(u *activity.Update) {
		u.ActivityTypeID, u.OwnerID = contentType, text(priya.ID)
		u.Attributes[countrySplit] = split("SO-1034", "100.00")
	})
	assertError(t, err, problems(
		"activityTypeId", "Choose an activity type that the model 'Mobility 1.0' allows.",
		"ownerId", "Choose an active person.",
		"attributes.SA-1013", "'Germany' is inactive — choose another option."))
}

// When the initiative's dates move, other edits still save; a date change is checked against the new dates.
func TestUpdate_UnchangedDatesNotChecked_EdgeCase(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	repo := campaigntest.NewStore(f.db)
	f.change(t, func(ctx context.Context) error {
		initiative, _, err := repo.GetInitiative(ctx, digitalAndSocial)
		if err != nil {
			return err
		}
		april := date.MustParse("2026-04-01")
		initiative.StartDate = &april
		return repo.PutInitiative(ctx, initiative)
	})

	view := f.mustEdit(t, "A-1001", func(u *activity.Update) { u.Description = "Dates moved." })

	if !slices.ContainsFunc(view.Readiness.Activity, func(p activity.SubmitProblem) bool {
		return p.Code == activity.ProblemDatesOutsideInitiative && p.Message == "Dates are outside the initiative's dates (01 Apr 2026 – 30 Jun 2026)"
	}) {
		t.Errorf("readiness = %+v, want the dates listed", view.Readiness.Activity)
	}
	_, err := f.edit(t, priya, "A-1001", func(u *activity.Update) { u.EndDate = text("2026-05-30") })
	assertError(t, err, problems("startDate", "Dates must be inside the initiative's dates (01 Apr 2026 – 30 Jun 2026)."))
}

// The edit form offers the activity's own inactive type and workflow, flagged, and leaves its own name out.
func TestEditForm_IncludesCurrentInactiveValues_FR013(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.deactivateType(t, contentType)
	f.deactivateWorkflow(t, lubricantsFlow)

	form, err := f.service.EditForm(t.Context(), priya, "A-1001")

	if err != nil {
		t.Fatalf("EditForm: %v", err)
	}
	if !slices.ContainsFunc(form.ActivityTypes, func(o activity.TypeOption) bool { return o.ID == contentType && !o.Active && o.Allowed }) ||
		!slices.ContainsFunc(form.ApprovalWorkflows, func(o activity.WorkflowOption) bool { return o.ID == lubricantsFlow && !o.Active }) {
		t.Errorf("types %+v, workflows %+v; want the current inactive ones", form.ActivityTypes, form.ApprovalWorkflows)
	}
	if form.Defaults != nil || !slices.Equal(form.ExistingNames, []string{"Social always-on"}) {
		t.Errorf("defaults %+v, names %q", form.Defaults, form.ExistingNames)
	}
	other, err := f.service.EditForm(t.Context(), priya, "A-1002")
	if err != nil || slices.ContainsFunc(other.ActivityTypes, func(o activity.TypeOption) bool { return o.ID == contentType }) {
		t.Errorf("another activity's form offers the inactive type: %+v, %v", other.ActivityTypes, err)
	}
	_, err = f.service.EditForm(t.Context(), priya, "A-9999")
	assertError(t, err, errs.NotFound("This activity no longer exists — someone may have deleted it."))
}
