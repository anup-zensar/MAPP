# Implementation Plan: Activities

**Branch**: `006-activities` | **Date**: 2026-10-19 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/006-activities/spec.md`

**Design artifacts**: [data-model.md](data-model.md) · [contracts/openapi.yaml](contracts/openapi.yaml) (fragment,
merged into `backend/api/openapi.yaml`) · [tasks.md](tasks.md). Conventions are feature 001's
([plan](../001-app-shell-and-sign-in/plan.md), [research](../001-app-shell-and-sign-in/research.md),
[data model](../001-app-shell-and-sign-in/data-model.md)); this plan only adds what is new.

## Summary

Feature 006 adds the **activity**: the record under an initiative that carries planned cost, Shell
attributes, goals and the approval status. It delivers:

- **Go API** — package `backend/internal/activities` (handler → service → repository, JSON table
  `activities`): create, read, edit (with the re-approval rule), move, copy and delete to the recycle bin;
  the derived **display status** (*In progress*, *Complete*), **completeness**, **unassigned planned cost**
  and the **ready-to-submit check** (every problem at once, grouped "This activity" / "Line items");
  calculated goals from the plan's model; a `history.Kind` "activity" (Undo exact, later restore as
  *Created*); one **form endpoint** that gives the web app every default and choice the form needs; the
  activity rows of the plan tree; and Go seams that features 004/005 (roll-ups, copies), 007 (line items)
  and 008 (approvals) plug into.
- **Web app** — `frontend/src/features/activities/`: the full-page **new / edit activity form**, the
  **activity page** (header with status and completeness chips, Ready to submit panel, Details, Shell
  attributes, Goals, Line items and Approval placeholders, "From finance (coming later)"), the Move, Copy,
  Delete and "will need approval again" dialogs, and the activity rows and **Add activity** actions in the
  plan tree and the initiative panel of feature 005.

Approach: the same pattern as `internal/samples` and `features/dev` (constitution VII); every rule runs in
the service; the web app repeats the simple checks only for instant feedback (constitution IV); roll-ups
are calculated when read, never stored.

## Technical Context

**Language/Version**: Go 1.27 (API); TypeScript 5.9 strict + React 19 (web app) — unchanged from 001

**Primary Dependencies**: none new. API: standard library; `internal/platform/money` (feature 002).
Web app: the libraries of 001 (React Router 8, TanStack Query 5, Radix, lucide-react, openapi-fetch)

**Storage**: JSON file store, new table `activities` (one file `backend/data/activities.json`); the
activity log and recycle bin of 001; until feature 008, the seed-only table `approval_workflows` is read
(D-103)

**Testing**: Go `testing` (service tests with in-memory fakes of the other packages, fixed clock),
contract tests through `apitest` (clock 12 Mar 2026 09:00 UTC, demo seed); Vitest + Testing Library +
`mockApi`; Playwright `frontend/e2e/activities.spec.ts` + axe

**Target Platform**: developer laptops, Chrome/Edge (as 001)

**Project Type**: web application (React SPA + Go REST API)

**Performance Goals**: a plan with 500 activities answers `GET /api/plans/{planId}/activities` and opens
in under 2 seconds (004 SC-009); every other call well under 100 ms locally

**Constraints**: 127.0.0.1 only; money as exact decimals (never float); calendar dates `YYYY-MM-DD`;
WCAG 2.1 AA; usable at 1280 × 720 and 390 px; deterministic tests

**Scale/Scope**: 5 user stories, 33 FRs; 10 API operations; 3 routes; 11 seed activities

## Constitution Check

*GATE: checked before design and again after it.*

| Principle | How this plan complies | Result |
|---|---|---|
| I. Spec first | Every operation, screen and message traces to an FR, scenario or edge case (traceability table). The one `[NEEDS CLARIFICATION]` (FR-024) is resolved with the decision-log default of Q-008 | Pass |
| II. Shell business, plain language | Activity, initiative, line item, planned cost, "Set later stages myself", "Ongoing (no end date)", "Unassigned planned cost"; no Hive9 terms; Shell attributes, activity types and models are Settings data (002), never hard-coded | Pass |
| III. Simple, friendly experience | Standard CRUD pattern (full-page form, details page with **Edit**, delete dialog + Undo); status chip and completeness chip on the page header and every tree row; one Ready to submit list with links; all form problems at once; every lock and unavailable action says why (SC-008) | Pass |
| IV. Simple, layered architecture | One package `activities` (handler → service → repository); every rule (R3, R5–R7, R9, R10, §2.5, §4.3, §4.5) in the service; roll-ups for 004/005 come from the service; later features plug in through two small interfaces, no third runtime part | Pass |
| V. Tested business rules | One Go test per rule named with its FR; a contract test per operation; component tests per page and dialog; Playwright journeys for US1–US4 (P1) and US5; fixed clock | Pass |
| VI. Safe by default | Actor on every service call; one log entry per change in the same `Write`; delete to the recycle bin; versions on every save (D-030); money exact decimal USD; no personal data in server logs | Pass |
| VII. Keep it small | No new dependency; one extension of a shared component (`CompletenessChip`, reason below); the form endpoint replaces five client calls to other features' endpoints | Pass |

**Post-design re-check**: Pass. The choices that add code are in *Complexity Tracking*.

## Decisions

**Resolved clarification**

- **FR-024 (Q-008)** — the decision-log default applies: a **positive** unassigned planned cost is advice
  only and never blocks submitting (an activity with planned cost and no line items can be submitted); a
  **negative** one blocks submitting (D-016). The Ready to submit panel shows the positive amount under
  "Good to know".

**Proposed decisions** (numbered at consolidation, 2026-10-19)

| ID | Decision | Reason |
|---|---|---|
| D-102 | Activity pages live under their plan's address: `/plans/{planId}/activities/{activityId}`, `/plans/{planId}/activities/{activityId}/edit`, `/plans/{planId}/initiatives/{initiativeId}/activities/new`. A wrong `planId` redirects to the right one. | The **Plans** sidebar item stays highlighted; the address reads like the breadcrumb |
| D-103 | Until feature 008, approval workflows are **seed-only**: the seed writes the two demo workflows into the table `approval_workflows` (internal ids `WF-1001`, `WF-1002` from the `WF` counter, unless 002 fixes another internal-id convention for Settings records), and activities read them through the interface `activities.Workflows`. Feature 008 takes over the table and implements the interface. | FR-012 needs active workflows before 008 exists ("seeded until then"); one table avoids migrating ids later |
| D-104 | One **form endpoint** per form (`GET …/activity-form` for new, `GET /api/activities/{id}/form` for edit) answers the plan context, defaults, the activity types the model allows, the active workflows, the activity-level Shell attributes, the business units of the plan and the names already used in the initiative. | The server decides R2-style defaults, R7 and R9; the web app needs one call instead of five, and can repeat every save check for instant "all at once" feedback |
| D-105 | Planned-cost, goal and approval-progress roll-ups of initiatives, campaigns and plans are **calculated when read** from `activities.(*Service).Rollups`, never stored. | Move, copy, delete, Undo and restore can never leave a stale total (FR-005, SC-006) |
| D-106 | The API refuses a re-approval change on a *Submitted* or *Approved* activity unless the request says `restartApproval: true` (`validation_failed` on `restartApproval`). | Approval is never restarted silently, even by a client that skipped the dialog (FR-027, SC-005) |
| D-107 | An activity edit sends the **whole editable state** (every field, `null` to clear) with `version` in `PATCH /api/activities/{activityId}`. | The edit form always holds the whole record; this avoids "not sent" versus `null` ambiguity in Go |
| D-108 | Values of Shell attributes whose business-unit group is hidden are **kept by the server** even when a save omits them. | R9 and FR-015 ("hidden values are kept") must hold whatever the client sends |
| D-091 | "Today" for *In progress* / *Complete* is the server clock's **UTC calendar date** (same rule as 005's Timing). | One deterministic rule, testable with the fixed clock |
| D-109 | The completeness chip counts the problems of the panel's "This activity" group (empty needed fields, unfinished % splits, dates outside the initiative, a type the model no longer allows, an inactive workflow) and adds "*m* line items need attention"; the negative unassigned balance is listed under "Line items" and marked red on the row. The ready-to-submit check also lists an inactive approval workflow (008 FR-010 e). | The chip, the panel and feature 008's submit check always agree |
| D-110 | Copying writes one `copied` log entry on the new activity ("Copied from 'Launch film' (A-1001), with 3 line items"); line items copied with it, and activities copied inside a plan/campaign/initiative copy, write no entries of their own. A re-approval save writes one `changed` entry that includes the status change and the reason in `details`. | One entry per user action (001 SC-004), readable history |
| D-111 | Activity rows (tree and lists) are ordered by start date, then name ignoring case; activities without dates last. | Same order as 005 FR-020 for campaigns and initiatives |
| D-112 | Until feature 008, **Submit for approval** with nothing missing shows "This activity is ready. Sending activities for approval is coming soon."; until feature 011, **View history** opens `/activity-log?recordType=activity&recordId=<id>`. | FR-025 and FR-032 need visible actions before their features exist |

## Project Structure

### Documentation (this feature)

```text
specs/006-activities/
├── spec.md                  # what and why
├── checklists/requirements.md
├── plan.md                  # this file
├── data-model.md            # the activity record, statuses, derived values, history kind
├── contracts/openapi.yaml   # this feature's paths and schemas (fragment of backend/api/openapi.yaml)
└── tasks.md
```

### Source code (repository root)

```text
backend/
├── api/openapi.yaml                       # + the fragment's paths and schemas
├── internal/
│   ├── app/app.go                         # + wiring of activities (repo, service, kinds, routes, seams)
│   └── activities/                        # NEW
│       ├── model.go                       # Activity (stored), Status, API types (View, Summary, FormView, …)
│       ├── repository.go · store_json.go  # Repository, table "activities"
│       ├── deps.go                        # interfaces of what activities reads from 002/003/004/005,
│       │                                  #   and the seams Workflows (008), LineItems (007), ApprovalHooks (008)
│       ├── workflows_seed.go              # NewSeededWorkflows(): reads table approval_workflows until 008
│       ├── dates.go · dates_test.go       # date parsing, R3/R5 checks, "01 Mar 2026" texts
│       ├── goals.go · goals_test.go       # calculated / typed goals (domain model §4.5)
│       ├── status.go · status_test.go     # display status (In progress / Complete)
│       ├── readiness.go · readiness_test.go   # the ready-to-submit check and completeness
│       ├── names.go · names_test.go       # unique names, " (copy)" and " (restored)" suffixes
│       ├── service.go                     # Create, Get, Update, Delete, ListForPlan, forms
│       ├── view.go · view_test.go         # builds View, Summary and FormView from a record and its context
│       ├── move_copy.go                   # Destinations, Move, Copy, CopyAll
│       ├── provided.go                    # Rollups, TypesNotAllowed, SubmitCheck, SetStatus (for 004/005/008)
│       ├── kinds.go · kinds_test.go       # history kind "activity": Describe, PrepareRestore
│       ├── fakes_test.go                  # in-memory fakes of deps for the service tests
│       ├── create_test.go · update_test.go · move_copy_test.go · provided_test.go
│       └── handler.go · handler_test.go   # package activities_test, apitest
└── seed/seed.go · seed_test.go            # + the demo activities (and the two workflows until 008)

frontend/
├── src/
│   ├── api/schema.d.ts                    # regenerated (npm run api:types)
│   ├── app/router.tsx                     # + the three activity routes
│   ├── components/CompletenessChip/       # + optional props `below` and `onOpen` (reason below)
│   └── features/activities/               # NEW
│       ├── activitiesApi.ts               # queries, mutations, query keys
│       ├── activityStatus.ts (+ .test.ts) # status → word and tone; banner text
│       ├── ActivityStatusChip.tsx (+ .test.tsx)   # StatusChip with the activity's word and tone
│       ├── activityText.ts (+ .test.ts)   # date range, "Ongoing from …", counts in words
│       ├── activityForm.ts (+ .test.ts)   # form state, client checks, re-approval changes, request body
│       ├── ActivityPage.tsx (+ .module.css, .test.tsx)
│       ├── ActivityHeader.tsx (+ .module.css, .test.tsx)
│       ├── ReadyToSubmitPanel.tsx (+ .module.css, .test.tsx)
│       ├── ActivitySections.tsx (+ .module.css, .test.tsx)   # Details, Shell attributes, Goals, From finance
│       ├── ActivityLineItemsSection.tsx   # placeholder card; feature 007 replaces its body
│       ├── ActivityApprovalSection.tsx    # placeholder card; feature 008 replaces its body
│       ├── ActivityFormPage.tsx (+ .module.css, .test.tsx)   # new and edit
│       ├── ActivityForm.tsx (+ .module.css, .test.tsx)       # the form sections (presentational)
│       ├── GoalsFields.tsx (+ .test.tsx)
│       ├── ReapprovalDialog.tsx (+ .test.tsx)
│       ├── DestinationList.tsx (+ .module.css)
│       ├── MoveActivityDialog.tsx (+ .test.tsx) · CopyActivityDialog.tsx (+ .test.tsx)
│       ├── DeleteActivityDialog.tsx (+ .test.tsx)
│       ├── useActivityActions.tsx (+ .test.tsx)              # Copy / Move / Delete / View history, shared by page and row
│       └── activityTreeRows.tsx (+ .test.tsx)                # plan-tree rows and initiative "Add activity"
└── e2e/activities.spec.ts                 # NEW
```

**Structure decision**: the layout of 001 (`docs/architecture/repository-layout.md`); one Go package and one
web feature folder.

## Backend design

### Stored record

`activities.Activity` (table `activities`, all fields in [data-model.md](data-model.md) §1): `ID`,
`InitiativeID` (the only parent link stored — campaign and plan are read through the initiative, so a
campaign moved to another plan by 005 needs no activity change), `Name`, `ExternalName`,
`ActivityTypeID`, `OwnerID`, `StartDate`, `EndDate`, `Ongoing`, `PlannedCost *money.Money`,
`ApprovalWorkflowID`, `Description`, `Attributes settings.AttributeValues`, `INQ *int64`,
`SetLaterStagesMyself bool`, `MQL *int64`, `Revenue *money.Money`, `Status`, `record.Meta`. Empty text
means "not set" (`""`), so `history.Diff` treats empty and missing alike. `Status` is `created`,
`submitted`, `declined` or `approved`.

### What activities reads from other packages (`deps.go`)

Small consumer-side interfaces; the real repositories/services of earlier features satisfy them (names
expected — see *Dependencies*), the service tests use in-memory fakes (`fakes_test.go`).

```go
type PlanReader interface { // plans.Repository (004)
    Get(ctx context.Context, id string) (plans.Plan, bool, error)
}
type HierarchyReader interface { // campaigns.Repository (005)
    GetInitiative(ctx context.Context, id string) (campaigns.Initiative, bool, error)
    GetCampaign(ctx context.Context, id string) (campaigns.Campaign, bool, error)
    CampaignsInPlan(ctx context.Context, planID string) ([]campaigns.Campaign, error)
    InitiativesInCampaign(ctx context.Context, campaignID string) ([]campaigns.Initiative, error)
}
type SettingsReader interface { // settings (002)
    GetModel(ctx context.Context, id string) (settings.Model, bool, error)
    GetActivityType(ctx context.Context, id string) (settings.ActivityType, bool, error)
    ListActivityTypes(ctx context.Context) ([]settings.ActivityType, error)
    AttributesForLevel(ctx context.Context, level settings.Level) ([]settings.Attribute, error)
}
type PeopleReader interface { // people.Repository (001, extended by 003)
    Get(ctx context.Context, id string) (people.Person, bool, error)
}

// Workflows is where activities find approval workflows (008 owns them; NewSeededWorkflows until then).
type Workflows interface {
    GetWorkflow(ctx context.Context, id string) (WorkflowFacts, bool, error)
    ListWorkflows(ctx context.Context) ([]WorkflowFacts, error) // active and inactive, sorted by name
}
type WorkflowFacts struct {
    ID, Name string
    Active   bool
    Levels   []WorkflowLevelFacts // in order
}
type WorkflowLevelFacts struct{ Name string; ApproverIDs []string }

// LineItems is what activities needs from line items (007 implements it; noLineItems until then).
type LineItems interface {
    // Facts returns, per activity id, its line-item totals and problems (map entry missing = no line items).
    Facts(ctx context.Context, activityIDs []string) (map[string]LineItemFacts, error)
    // CopyAll copies every line item of fromActivityID into toActivityID with new IDs (R13), inside the
    // caller's Write, writing no log entries. It returns how many it copied.
    CopyAll(ctx context.Context, a actor.Actor, fromActivityID, toActivityID string) (int, error)
    // DatesChanged re-spreads the E/F/L line items after the activity's dates changed (007 FR-022),
    // inside the caller's Write.
    DatesChanged(ctx context.Context, a actor.Actor, before, after Activity) error
}
type LineItemFacts struct {
    Count             int
    PlannedCostUSD    money.Amount     // sum of the line items' USD planned costs
    AutoSpread        int              // line items spread Evenly / First / Last (re-spread on a date change)
    CustomSpread      int              // line items with a custom spread (kept on a date change)
    Problems          []LineItemProblem
}
type LineItemProblem struct {
    LineItemID, LineItemName string
    Code    string // e.g. "missingField", "customSpreadMismatch", "inactiveFinanceBudgetItem", "splitIncomplete"
    Field   string // the line-item form field, e.g. "glCodeId" or "attributes.<id>"
    Message string // e.g. "GL code — needed to submit"
}

// ApprovalHooks lets feature 008 react inside the same Write (noApprovalHooks until then).
type ApprovalHooks interface {
    ApprovalRestarted(ctx context.Context, a actor.Actor, before, after Activity, changes []history.Change) error
    ActivityDeleting(ctx context.Context, a actor.Actor, act Activity) error
}
```

Shared pure functions used from feature 002 (expected names): `settings.BusinessUnitsOf(planAttributes)
[]settings.OptionRef`; `settings.Shown(attr, businessUnits) (shown bool, unit string)`;
`settings.CheckValues(attrs, before, after settings.AttributeValues, businessUnits) []errs.FieldError`
(save-blocking: unknown attribute or option, newly chosen inactive option, dependent option not linked to
the parent's value, share format; unchanged inactive values pass); `settings.MissingValues(attrs, values,
businessUnits) []settings.Missing` (needed-to-submit: empty shown active attributes — not a dependent one
whose parent value has no options, D-034 — and % splits not totalling 100.00 %, with the total);
`settings.DescribeValue(attr, value) string`; `settings.Waterfall(model, inq int64) settings.Goals{MQL,
Deals int64; Revenue money.Money}`. Money (`internal/platform/money`, 002): `money.Money{Amount
money.Amount; Currency string}` (JSON `{"amount": "1234.50", "currency": "USD"}`), `money.Amount` with
`RoundCents()` (half-up), `Add`, `Sub`, `Sign`, `IsZero`, `String()`, and `money.Money.Text()` →
"120,000.00 USD".

### Constructors and wiring

```go
func NewJSONRepository() Repository
func NewService(tx store.Transactor, repo Repository, generator *ids.Generator,
    hist *history.Service, now clock.Clock, deps Deps) *Service
type Deps struct { Plans PlanReader; Hierarchy HierarchyReader; Settings SettingsReader;
    People PeopleReader; Workflows Workflows }
func (s *Service) UseLineItems(li LineItems)        // called by app.go in feature 007
func (s *Service) UseApprovalHooks(h ApprovalHooks) // called by app.go in feature 008
func (s *Service) Kinds() []history.Kind            // [activity]
func NewHandler(s *Service) *Handler
func (h *Handler) Routes(r *web.Router, signedIn web.Wrap)
func NewSeededWorkflows() Workflows                 // until 008
```

`app.New`: after the campaign and initiative kinds (005) are registered, build `activities.NewService(db,
activities.NewJSONRepository(), generator, historyService, now, activities.Deps{Plans: plansRepo,
Hierarchy: campaignsRepo, Settings: settingsService, People: peopleRepo, Workflows:
activities.NewSeededWorkflows()})`, register its kinds (an error stops start-up), add its routes, and hand
the service to the activity seams of 004 and 005 (Rollups, CopyAll, TypesNotAllowed — see *Provides*).

Repository (`repository.go`): `Get(ctx, id) (Activity, bool, error)`, `ListByInitiative(ctx,
initiativeID) ([]Activity, error)` (ordered by id), `ListByInitiatives(ctx, ids []string)
([]Activity, error)`, `Put(ctx, Activity) error`, `Delete(ctx, id) error`.

### Service operations (every method takes `ctx, a actor.Actor`; one `tx.Write`/`tx.Read` each)

| Operation | Rules (in order) | FR |
|---|---|---|
| `NewForm(ctx, a, initiativeID) (FormView, error)` | Initiative missing → `404` "This initiative no longer exists — someone may have deleted it." Context: plan (id, name, year, status), campaign, initiative (owner, dates), model (rates, deal size, active), business units of the plan, latest end date (31 Dec of plan year + 10). Activity types: active **and** allowed by the model, sorted by name, each with its default planned cost. Workflows: active ones. Attributes: active activity-level attributes of 002 in section and display order. `existingNames`: names of the initiative's activities. Defaults: owner = the initiative's owner (when active), dates = the initiative's dates, approval workflow = the only active one when exactly one is active | FR-001, FR-002, FR-007, FR-008, FR-009, FR-012, FR-014, FR-015, edge "no types" |
| `EditForm(ctx, a, id) (FormView, error)` | As `NewForm` for the activity's initiative, **plus** the current type even when inactive or not allowed (flagged `active`/`allowed`), the current workflow even when inactive, the current attribute values' inactive options (002 definitions carry them); `existingNames` excludes the activity itself; no defaults | FR-013, FR-026 |
| `Create(ctx, a, initiativeID, in Input) (View, error)` | **Shape** (outside the Write, collected): name required (trimmed) ≤ 120; external name ≤ 120; description ≤ 2,000; type required; dates valid `YYYY-MM-DD`; end ≥ start; planned cost currency USD, amount ≥ 0 (rounded half-up to cents); revenue currency USD, ≥ 0; INQ, MQL ≥ 0. **In the Write** (collected with the shape problems, all at once): initiative exists (else 404); name unique in the initiative (case and surrounding spaces ignored); type exists, active and allowed by the plan's model; owner, when set, exists and is active; workflow, when set, exists and is active; dates inside the initiative's dates when it has both (ongoing: start only); end ≤ latest end date; ongoing → end date cleared; `settings.CheckValues` with `before` = empty; goals normalised (not typed → MQL and revenue not stored). Then `ids.Next("A")`, status `created`, `record.NewMeta`, `Put`, log `created` | FR-003 – FR-013, FR-017, FR-018 |
| `Get(ctx, a, id) (View, error)` | 404 "This activity no longer exists — someone may have deleted it."; builds `View` (path refs, type, owner and workflow with `active`, display status, goals, unassigned planned cost, line-item totals, completeness, `readiness`) | FR-019 – FR-024 |
| `ListForPlan(ctx, a, planID) ([]Summary, error)` | 404 "This plan no longer exists — someone may have deleted it."; every activity of every initiative of every campaign of the plan as `Summary`, ordered per D-111 | FR-033 |
| `Update(ctx, a, id, up Update) (View, error)` | Shape as Create, plus `version` missing or < 1 → field `version` "Reload the page and try again.". In the Write: 404; `CheckVersion(version, "activity")`; then, collected: name unique (excluding itself); **type** changed → active and allowed; **owner** changed → active; **workflow** changed → active, and only while status is `created` or `declined` (else field `approvalWorkflowId` "The approval workflow can't change while the activity is submitted or approved."); **ongoing** stored true and sent false → field `ongoing` "Ongoing can't be undone — to plan with an end date, create a new activity."; **dates** checked against the initiative and R5 **only when start, end or ongoing changed** (edge case: other edits save while the initiative's dates moved); `settings.CheckValues(before, after)`; hidden-group values merged back (D-108). Diff (fields + one change per attribute, `attributes.<id>`); no change → answer the stored record (no version, no log). **Re-approval**: status `submitted` or `approved` and planned cost, start date, end date or ongoing changed → without `restartApproval: true` refuse (field `restartApproval` "This activity will need approval again — confirm the change to save it."), with it set status `created` (the status change is part of the `changes`). `Touch`, `Put`, log `changed` (details "Approval restarted because the planned cost changed." — fields joined: "the planned cost and the start date"), `LineItems.DatesChanged` when start/end/ongoing changed, `ApprovalHooks.ApprovalRestarted` when restarted | FR-006 – FR-018, FR-026 – FR-028 |
| `Delete(ctx, a, id) (history.BinEntry, error)` | 404; `ApprovalHooks.ActivityDeleting`; `history.MoveToBin(ctx, a, "activity", id)` (line items of 007 go with it as child kind) | FR-031 |
| `Destinations(ctx, a, id) ([]Destination, error)` | Every initiative of the activity's plan, grouped by campaign (campaigns and initiatives by start date then name), each with `current`, `moveProblem` and `copyProblem` (null when allowed): current initiative → move "It's already in this initiative."; dates not inside the target's dates (target with both dates; ongoing: start only) → both, the dates message; name taken in the target → move only, the name message | FR-029, FR-030 |
| `Move(ctx, a, id, in MoveInput) (View, error)` | `version` required; 404; `CheckVersion`; target must be an initiative of the **same plan** (else field `initiativeId` "Choose another initiative of this plan."); same initiative → same message; dates fit; name free. Set `InitiativeID`, `Touch`, `Put`, log `moved` with change `parent` (old and new `RecordRef`, texts "Helix launch › Digital & social"). Status, values and line items unchanged (D-023, D-033) | FR-029 |
| `Copy(ctx, a, id, in CopyInput) (View, error)` | 404; target initiative of the same plan (same message as Move); dates fit when the target differs (else field `initiativeId`, the dates message); name = first free of "<name> (copy)", "<name> (copy 2)", … in the target, the original part shortened so the whole stays ≤ 120 characters; new ID, all values copied, status `created`, new `record.Meta`; `LineItems.CopyAll`; log `copied` on the copy, details "Copied from '<name>' (<id>)" + ", with <n> line items" | FR-030 |
| `Kinds()` | Kind `activity` (data-model §5): `PrepareRestore` — Undo: name taken in the initiative → `name_taken` "Can't undo — there is now another activity called '<name>'. Restore it from the recycle bin instead."; Restore: name taken → " (restored)", " (restored 2)", … with the note "Renamed to '<new name>' because the name was taken."; status not `created` → `created` with the note "Status reset to Created because it was restored from the recycle bin." (R14); `Touch` when changed | FR-031 |

**Field-to-error mapping**: every problem is an `errs.FieldError` whose `field` is the request's JSON path:
`name`, `externalName`, `activityTypeId`, `ownerId`, `startDate`, `endDate`, `ongoing`, `plannedCost`,
`approvalWorkflowId`, `description`, `attributes.<attributeId>` (from 002), `goals.inq`, `goals.mql`,
`goals.revenue`, `version`, `restartApproval`, `initiativeId`. Messages: *User-visible messages*.

### Derived values (`status.go`, `goals.go`, `readiness.go`)

- **Display status** (FR-021): `approved` and today (D-091) ≥ start → `inProgress`; `approved`, not
  ongoing, end set and today > end → `complete`; otherwise the stored status. No dates → `approved`.
- **Goals** (FR-018, domain model §4.5): `SetLaterStagesMyself` false → MQL, deals and revenue from
  `settings.Waterfall(model, INQ)` at read time (current model values), `calculated: true`; INQ empty →
  all null. True → the typed MQL and revenue, `calculated: false`, `deals` null. Stored MQL/revenue are
  cleared when not typed.
- **Unassigned planned cost** (domain model §4.3): planned cost (0.00 when empty) − `PlannedCostUSD` of the
  line items; `null` when there is neither a planned cost nor a line item.
- **Ready-to-submit check** (`readiness.go`, pure function `check(in checkInput) SubmitCheck`), all at once:
  - *This activity* (in this order): each empty needed-to-submit fixed field — Owner, Start date, End date
    (not when ongoing), Planned cost (0.00 is filled), Approval workflow — code `missingField`; each result
    of `settings.MissingValues` (`missingField` or `splitIncomplete`); dates outside the initiative's dates
    (`datesOutsideInitiative`, field `startDate`); type not allowed by the model (`typeNotAllowed`, field
    `activityTypeId`); workflow inactive (`inactiveWorkflow`, field `approvalWorkflowId`).
  - *Line items*: unassigned planned cost < 0 (`negativeUnassigned`, field `lineItems`), then every
    `LineItemProblem` of 007 in line-item order.
  - *Advice* (never blocks, FR-024): unassigned planned cost > 0 (`unassignedRemaining`).
  - `problemCount` = blocking items; `ready` = `problemCount == 0`.
- **Completeness** (FR-022, D-109): `missingFields` = number of *This activity* items;
  `lineItemsNeedingAttention` = distinct line items with a problem.

### Activity-log entries

| Person's action | Entries (same Write as the change) |
|---|---|
| Create | 1 × `created` |
| Save with changes | 1 × `changed`; one change per field, attributes as `attributes.<id>` with the attribute's label |
| Save of a re-approval change (confirmed) | 1 × `changed` including `status` (Approved → Created); `details` "Approval restarted because the planned cost changed." |
| Save without change / out-of-date save | none |
| Move | 1 × `moved`, change `parent` (label "Initiative") |
| Copy | 1 × `copied` on the new activity (D-110) |
| Delete / Undo / Restore | written by `history` (details "with 3 line items", "Undo, with 3 line items", restore notes) |
| `SetStatus` (used by 008) | 1 × `statusChanged` with the given verb and comment |
| `CopyAll` (inside 004/005 copies) | none |

`Describe` (labels and texts) is in [data-model.md](data-model.md) §5.

### Provides to other features (Go)

| Method on `*activities.Service` | For | Contract |
|---|---|---|
| `Rollups(ctx, initiativeIDs []string) ([]Rollup, error)` | 004, 005, 009, 010 | Joins the caller's transaction. `Rollup{ActivityID, InitiativeID, Name string; PlannedCost money.Amount (0 when empty); HasPlannedCost bool; Status Status; DisplayStatus DisplayStatus; Approved bool (approved, in progress or complete); StartDate, EndDate string; Ongoing bool; INQ, MQL int64; Revenue money.Amount (goals, 0 when empty); NeedsAttention bool (problemCount > 0)}` |
| `CopyAll(ctx, a, fromInitiativeID, toInitiativeID string, yearShift int) (int, error)` | 004 (plan copy), 005 (campaign/initiative copy) | Same names, dates moved `yearShift` years (29 Feb → 28 Feb), status `created`, `LineItems.CopyAll` per activity, no log entries |
| `TypesNotAllowed(ctx, planID string, allowedTypeIDs []string) ([]history.RecordRef, error)` | 004 FR-019 | Activities of the plan whose type is not in the list |
| `SubmitCheck(ctx, a, id) (SubmitCheck, error)` | 008 | The check above |
| `SetStatus(ctx, a, id string, change StatusChange) (Activity, error)` | 008 | `StatusChange{To Status; Verb, Comment string}`; joins the caller's Write; `Touch`, `Put`, log `statusChanged` |
| `Repository.Get` / `ListByInitiative` | 007, 008 | Read activities inside their transactions |
| History kind name `activity` | 007 | Parent of the `lineItem` kind |

### API operations

All need a signed-in person; errors exactly as 001 (`Error`, `fields` always present).

| Method and path | Request | Response | Errors |
|---|---|---|---|
| `GET /api/initiatives/{initiativeId}/activity-form` | — | `200 ActivityForm` (with `defaults`) | 401, 404 |
| `POST /api/initiatives/{initiativeId}/activities` | `ActivityCreate` | `201 Activity` | 400 `validation_failed` / `bad_request`, 401, 404 |
| `GET /api/plans/{planId}/activities` | — | `200 ActivitySummaryPage` `{items, total}` | 401, 404 |
| `GET /api/activities/{activityId}` | — | `200 Activity` | 401, 404 |
| `GET /api/activities/{activityId}/form` | — | `200 ActivityForm` (`defaults` null) | 401, 404 |
| `PATCH /api/activities/{activityId}` | `ActivityUpdate` (every field + `version` + `restartApproval`) | `200 Activity` | 400, 401, 404, 409 `version_conflict` |
| `DELETE /api/activities/{activityId}` | — | `200 DeleteResult` | 401, 404 |
| `GET /api/activities/{activityId}/destinations` | — | `200 ActivityDestinationPage` | 401, 404 |
| `POST /api/activities/{activityId}/move` | `ActivityMove {initiativeId, version}` | `200 Activity` | 400, 401, 404, 409 |
| `POST /api/activities/{activityId}/copy` | `ActivityCopy {initiativeId}` | `201 Activity` (the copy) | 400, 401, 404 |

Undo and restore use 001's `POST /api/recycle-bin/{entryId}/undo|restore`.

## Demo data (`backend/seed/seed.go`)

Added after 005's campaigns and initiatives, in this order (IDs follow from the order; models, types,
attributes from 002, people from 001, plans from 004, initiatives from 005 — names below are those of
[demo-users-and-data.md](../../docs/product/demo-users-and-data.md); initiative names are the ones 005's
plan seeds, adjust at consolidation). Until feature 008: first the two workflows of the demo data
(`WF-1001` "Lubricants EU approval" — level 1 Daniel Okoro; `WF-1002` "China BC approval workflow" —
level 1 Carlos Mendes, level 2 Alex Morgan; both active).

| ID | Name | Plan › campaign › initiative | Status | Type | Owner | Dates | Planned cost | Notes |
|---|---|---|---|---|---|---|---|---|
| A-1001 | Launch film | Lubricants Europe › Helix launch › Digital & social (1 Jan – 30 Jun 2026) | Created | Content production | Priya Raman | 01 Mar – 31 May 2026 | 120,000.00 | Workflow WF-1001; INQ 250,000 (calculated); Country / OU Germany 50.00 %, France 30.00 %, Netherlands 20.00 %; **Company code, Cost centre and PIR empty** ("3 fields needed"); 007 adds line items worth 100,000.00 USD (unassigned 20,000.00) |
| A-1002 | Social always-on | … › Helix launch › Digital & social | Approved | Digital media | Priya Raman | 01 Jan – 30 Jun 2026 | 80,000.00 | Every needed field filled |
| A-1003 | Billboard winter wave | … › Helix launch › Print & outdoor (1 Jan – 31 Dec 2026) | Approved (shows Complete) | Agency fees – Creative | Priya Raman | 05 Jan – 28 Feb 2026 | 45,000.00 | Complete |
| A-1004 | Trade magazine ads | … › Helix launch › Print & outdoor | Submitted | Content production | Priya Raman | 01 Apr – 30 Sep 2026 | 30,000.00 | Waiting for Daniel Okoro (008) |
| A-1005 | Fleet roadshow | … › Rimula fleet › Retail events (1 Jun – 31 Dec 2026) | Declined | Event | Priya Raman | 01 Jul – 30 Sep 2026 | 60,000.00 | Daniel Okoro's comment (008) |
| A-1006 | Christmas trade gifting | … › Rimula fleet › Retail events | Approved | Event | Priya Raman | 01 Nov – 15 Dec 2026 | 25,000.00 | Approved, not started |
| A-1007 | Partner co-marketing fund | … › Rimula fleet › Fleet partnerships (1 Jan – 31 Dec 2026) | Approved (shows In progress) | Sponsorship | Priya Raman | Ongoing from 01 Jan 2026 | 100,000.00 | Ongoing, never Complete |
| A-1008 | Grand opening Shanghai | Mobility China › Retail growth › Store openings (1 Jan – 31 Dec 2026) | Created | Event | Mei Lin | 01 Apr – 30 Jun 2026 | 40,000.00 | Workflow WF-1002; **Country / OU China 60.00 %, Hong Kong 30.00 % (90.00 %)**; **Cost centre empty**; 007 adds a line item without GL code |
| A-1009 | Fleet card launch | Mobility China › Retail growth › Fleet cards (1 Jan – 31 Dec 2026) | Submitted | MOBILITY – Long-term contract | Mei Lin | 01 May – 31 Dec 2026 | 150,000.00 | Complete; waiting at level 2 for Alex Morgan (008) |
| A-1010 | Lunar new year promo | Mobility China › Retail growth › Store openings | Created | Digital media | — | — | — | Deliberately incomplete: owner, dates, planned cost, workflow and attributes empty |
| A-1011 | Press day Hamburg | Lubricants Europe › Helix launch › the initiative 005's seed then deletes | Created | Event | Priya Raman | 01 May – 31 May 2026 | 15,000.00 | Goes to the recycle bin with its initiative (005/011 seed step) |

Every "complete" activity has every needed-to-submit attribute filled with sample options of 002 (the
Lubricants group on the Lubricants plan; Country / OU and Brands at 100.00 %). Records are created by
`actor.System` at the seed time. `seed_test.go` checks the IDs, statuses and that A-1001 reports 3 missing
fields and A-1008 the 90.00 % split.

## Frontend design

### Routes (`frontend/src/app/router.tsx`, inside `RequireSignIn`)

| Path | Page | `handle.title` | Tab title |
|---|---|---|---|
| `/plans/:planId/initiatives/:initiativeId/activities/new` | `ActivityFormPage` (mode `new`) | New activity | New activity · NorthStar |
| `/plans/:planId/activities/:activityId` | `ActivityPage` | Activity | `<activity name>` · NorthStar |
| `/plans/:planId/activities/:activityId/edit` | `ActivityFormPage` (mode `edit`) | Edit activity | Edit `<activity name>` · NorthStar |

No sidebar change: the pages sit under `/plans`, so the **Plans** item stays current (D-102). When
the answer's `plan.id` differs from `:planId`, the page navigates (replace) to the right address.

### Data (`activitiesApi.ts`)

Query keys: `['activity', id]`, `['activity-form', 'new', initiativeId]`, `['activity-form', 'edit', id]`,
`['plan-activities', planId]`, `['activity-destinations', id]`. Every mutation, when settled, invalidates
`['activity', id]`, `['plan-activities', planId]`, the plan and plan-tree queries of 004/005 (expected
keys `['plan', planId]`, `['plan-tree', planId]`) and `['activity-log']`. Hooks: `useActivity`,
`useNewActivityForm`, `useEditActivityForm`, `usePlanActivities`, `useActivityDestinations`,
`useCreateActivity`, `useUpdateActivity`, `useDeleteActivity`, `useMoveActivity`, `useCopyActivity`
(all `call(api.…)` as `sampleApi.ts`).

### Activity form page (`ActivityFormPage` + `ActivityForm`, FR-002, FR-026)

- `PageHeader`: eyebrow "Activity"; title "New activity" / "Edit activity '<name>'"; description
  "In initiative '<initiative>' of campaign '<campaign>' · <plan>". Breadcrumb as the activity page.
- One column (max 720 px), sections in design-system §7 order, each field with its marking as a word:
  1. **Basics** — Activity name (Required) · External name (hint "Leave empty to use the activity name") ·
     Campaign and Initiative (read-only, hint "Change it later with Move") · Activity type (Required; a
     select of `activityTypes`, each "<name> · default <amount>"; current inactive / not allowed one shown
     with "(inactive)" / "— not allowed by the plan's model") · Owner (Needed to submit; 003's
     `PersonPicker`; default labelled "From the initiative") · Approval workflow (Needed to submit; a
     select; default labelled "The only active approval workflow"; locked with its reason while
     submitted or approved).
  2. **Dates & cost** — Start and end dates (Needed to submit; 005's `DateRangeField` with the allowed
     range "Allowed: <initiative dates> (the initiative's dates)", latest end date; default labelled
     "From the initiative") · **Ongoing (no end date)** checkbox (ticking clears the end date and shows the
     warning; saved ongoing → locked with its reason) · Planned cost (Needed to submit; 002's `MoneyInput`,
     USD, "0.00 or more"; type default labelled "From the activity type").
  3. **Shell attributes** — 002's `AttributeFields` for level `activity`, the plan's business units and
     the form's values: always-shown attributes, then each business-unit group with "Shown because the
     plan's business unit includes <unit>"; without a business unit, the sentence "<units> attributes appear
     once the plan's business unit is set" with the link **Open the plan** (`/plans/:planId`); dependent
     fields, split editor and "values that will be cleared" come from 002's component.
  4. **Goals** — `GoalsFields` (below).
  5. **More details** — Description (long text, ≤ 2,000).
  6. New only: a muted card "Line items" with "Save the activity to add its line items".
- Sticky footer: **Save** (primary, never disabled; "Saving…" while waiting) · **Save and add another**
  (new only) · **Cancel**.
- **Type change** (FR-007): picking a type with a default cost when the planned cost is empty fills it
  (label "From the activity type"); when it already has a value, `ConfirmDialog` "Use the default planned
  cost of <amount> for '<type>'?" — **Use default** / **Keep <current amount>** (no danger tone).
- **Client checks** (`activityForm.ts` `validate`, repeating the server's rules for instant feedback, all at
  once): name required, ≤ 120, not in `existingNames` (case/space-insensitive); external name ≤ 120;
  description ≤ 2,000; type chosen; amounts readable and ≥ 0; INQ/MQL whole numbers ≥ 0; end ≥ start; dates
  inside the initiative's dates and end ≤ latest end date (new, or edit when dates changed); 002's
  attribute checks. Problems → `FormErrorSummary` (focus) + messages under fields; nothing is sent. Server
  `validation_failed` → the same presentation from `error.fields`.
- **Save (new)**: `POST` → toast "Activity '<name>' created" and navigate to the activity page;
  **Save and add another** → toast "Activity '<name>' created — View" (action **View**) and a fresh empty
  form for the same initiative (the route remounts with a new key).
- **Save (edit)**: follows 001's edit-form rule (record and `version` copied once). When the stored status
  is submitted or approved and planned cost, start, end or ongoing changed (`reapprovalChanges()`),
  `ReapprovalDialog` first; **Save and restart approval** sends `restartApproval: true`. Success → toast
  "Activity '<name>' saved" and navigate to the activity page. `version_conflict` → the top message
  `conflictMessage(error, 'activity')` with typed values kept and the form's `version` replaced by the
  refetched one.
- **Leaving** with changes: `useUnsavedChanges` (**Keep editing** / **Discard changes**); **Cancel** →
  back to the plan page (new) or the activity page (edit).
- `?field=<field>` on the edit address focuses that field after load (links of the Ready to submit
  panel); input ids are `field-<field>` (`field-attributes.<id>` via `AttributeFields`' `idFor`).

### `GoalsFields` (FR-018)

INQ (whole number ≥ 0); MQL and Revenue read-only while **Set later stages myself** is off, labelled
"Calculated" with an `InfoTip` giving the formula and the model's values, "—" while INQ is empty; live
values from 002's `waterfallFromInq(inq, model)`. On: MQL (whole number) and Revenue (`MoneyInput`, USD)
editable, keeping typed values when INQ changes. Turning it off with typed values shows the note "These
typed values will be replaced when you save: MQL <old> → <new>; Revenue <old> → <new>."

### Activity page (`ActivityPage`, FR-019 – FR-025, FR-032)

- `ActivityHeader`: breadcrumb **Plans › <plan> › <campaign> › <initiative>** (links: `/plans`,
  `/plans/:planId`, 005's `campaignHref` / `initiativeHref`); the level tag "Activity" (004's `LevelTag`,
  magenta); `<h1>` name; `ActivityStatusChip`; `CompletenessChip` with `missing` = the panel's "This
  activity" items, `below` = "<m> line items need attention" and `onOpen` = open and focus the panel; the
  ID; facts: Owner (avatar + name, "(inactive)"), Dates ("01 Mar 2026 – 31 May 2026", "Ongoing from 01 Mar
  2026" or "—"), Planned cost, **Unassigned planned cost** (ⓘ "Planned cost minus the line items' planned
  costs in USD."; red with "Line items exceed the activity's planned cost by <amount>" when negative); "Created
  by <person> on <date>, last changed by <person> on <date>". Amounts use 003's display-currency formatter.
  Actions: **Edit** (primary) · **Submit for approval** (secondary) · **More** (`ActionsMenu`: Copy, Move,
  View history, separator, Delete).
- Banner (submitted, approved, in progress, complete): "<Submitted for approval | Approved | In progress |
  Complete> — changing planned cost or dates will restart approval."
- **Submit for approval** (FR-025): with problems, the alert "Submitting needs <n> things — view list"
  (link to the panel) appears under the header and the panel opens and takes focus; with none, the toast of
  D-112 (feature 008 replaces this handler).
- Sections in order: `ReadyToSubmitPanel` → Details → Shell attributes → Goals → `ActivityLineItemsSection`
  (id `line-items`) → `ActivityApprovalSection` → "From finance (coming later)" (collapsed `<details>`).
- `ReadyToSubmitPanel` (`Card` "Ready to submit", collapsible, open by default when there are problems):
  groups "This activity", "Line items", "Good to know"; each item a link: activity fields →
  `/plans/:planId/activities/:id/edit?field=<field>`; `negativeUnassigned` → `#line-items`; line-item
  items → 007's line-item address (`lineItemFieldHref`, until 007: `#line-items`). No problems → "Ready to
  submit".
- Details: Activity name, External name ("—" with "(the activity name is used)"), Campaign, Initiative,
  Activity type (with "(inactive)" / "Not allowed by the plan's model"), Owner, Start date, End date (or
  "Ongoing (no end date)"), Planned cost, Approval workflow, Description; empty → "—".
- Shell attributes: 002's read-only `AttributeValuesView` with the same group sentences as the form.
- Goals: INQ, MQL, Revenue with "Calculated" + ⓘ or "Typed"; "—" when empty.
- Line items placeholder: `Card` "Line items" — "Line items will appear here — adding them is coming soon."
- Approval placeholder: `Card` "Approval" — "Approval workflow: <name | —>" and "Approval steps and
  decisions will appear here".
- From finance: Total actuals, Total invoiced, Total purchase requests = "—"; "Filled when the Finance module
  and GSAP are connected".

### Actions (`useActivityActions`, used by the page and the tree rows)

- **Copy** → `CopyActivityDialog`: title "Copy activity '<name>'"; `DestinationList` (radio list grouped
  by campaign, current initiative preselected and marked "(this initiative)", unavailable ones visible with
  their `copyProblem`); text "The copy starts as Created, with copies of its <n> line items. Approvals,
  comments and history are not copied."; **Copy activity** / **Cancel** → toast "Activity '<copy name>'
  created", open the copy.
- **Move** → `MoveActivityDialog`: title "Move activity '<name>'"; "Choose another initiative in '<plan>'.
  The activity keeps its status, line items and values."; search box "Search initiatives"; `DestinationList`
  with `moveProblem`; no other initiative → "There's no other initiative in this plan to move it to.";
  **Move activity** / **Cancel**; a refusal shows the API message in the dialog; success → toast "Activity
  '<name>' moved to '<initiative>'".
- **Delete** → `DeleteActivityDialog` (`ConfirmDialog`, tone danger, **Cancel** focused): title "Delete
  activity '<name>'?"; lines: line items ("<n> line items go with it." / "1 line item goes with it.", none
  → no line), planned cost ("The initiative, campaign and plan planned costs go down by <amount>.", none
  without planned cost), submitted → "It leaves the approvers' inbox.", status other than created → "If
  it's restored later from the recycle bin, it comes back as Created.", always "It moves to the recycle
  bin; you can restore it."; **Delete activity**. After delete: navigate to `/plans/:planId`, then
  `useUndoableDelete({ invalidate: [plan activities, plan, plan tree] })` → "Activity '<name>' moved to
  the recycle bin" with **Undo** (10 s).
- **View history** → `/activity-log?recordType=activity&recordId=<id>` (D-112).

### Plan tree rows and Add activity (`activityTreeRows.tsx`, FR-001, FR-033)

- `usePlanActivities(planId)` → `activityRowsByInitiative(summaries)`: `Map<initiativeId, PlanTreeRow[]>`
  given to 004's `PlanTree` through its child-rows extension (expected; see *Dependencies*). Each row:
  level bar and tag "Activity", name as a link to the activity page, ID, `ActivityStatusChip`,
  `CompletenessChip` (`missing` count only, `below`, `onOpen` → activity page with the panel open:
  `…/activities/:id#ready`), owner, dates, planned cost with a red mark (icon + visually hidden text "Line
  items exceed the planned cost") when the unassigned planned cost is negative, and `ActionsMenu` "Actions
  for <name>": View · Edit · Copy · Move · separator · Delete. Selecting the row opens the activity page.
- **Add activity**: added to 005's initiative row actions (hover and ⋯ menu) and to 005's initiative details
  panel (button and an "Activities" list with name links and status chips), linking to the new-activity
  route.

### States of every page

| State | Form page | Activity page |
|---|---|---|
| Loading | `Skeleton` lines in the form layout | `Skeleton` header and two cards |
| Not found (404) | `EmptyState` "This initiative no longer exists — someone may have deleted it." (new) / the activity text (edit) with **Go to Plans** | `EmptyState` "This activity no longer exists — someone may have deleted it." with **Go to Plans** |
| Error / unreachable | `ErrorState` with **Try again** | `ErrorState` with **Try again** |
| Empty | Model without types: notice at the top and on the type field (edge case text, link **Open Settings**) | — (sections show "—") |
| Validation | `FormErrorSummary` + field messages | — |
| Conflict | Top message `conflictMessage(error, 'activity')`, values kept | Move dialog: the same message |
| Locked | Ongoing and workflow locks with reasons; Campaign/Initiative read-only with reason | — |

### Shared component change (reason)

`CompletenessChip` gets two optional props: `below?: string` (a second text, e.g. "2 line items need
attention", shown in the chip and its list) and `onOpen?: () => void` (when given, choosing the chip calls
it instead of opening the popover). Needed because FR-022 asks the chip to open the Ready to submit panel
and to show line items needing attention; 004/005 need the same "items below" text. If 004 adds an
equivalent prop first, 006 uses it instead. The component gallery shows both variants.

## User-visible messages

`<…>` is filled in at run time. Dates `DD MMM YYYY` ("01 Mar 2026"); amounts as 001's `formatMoney` in the
web app and `money.Money.Text()` ("120,000.00 USD") in API messages.

| Where | Text |
|---|---|
| Form header | Activity · New activity · Edit activity '<name>' · In initiative '<initiative>' of campaign '<campaign>' · <plan> |
| Sections | Basics · Dates & cost · Shell attributes · Goals · More details |
| Markings | Required · Needed to submit |
| Defaults | From the initiative · From the activity type · The only active approval workflow |
| Hints | Leave empty to use the activity name · Change it later with Move · Allowed: <start> – <end> (the initiative's dates) · The initiative has no dates yet. |
| Type option / flags | <type> · default <amount> · <type> (inactive) · <type> — not allowed by the plan's model |
| No types (edge) | The model '<model>' has no activity types yet — an administrator adds them in Settings. · **Open Settings** |
| No workflow | No approval workflow is active — an administrator sets them up in Settings. |
| Type default question | Use the default planned cost of <amount> for '<type>'? · **Use default** · **Keep <current amount>** |
| Ongoing | Ongoing can't be undone after saving. · Ongoing can't be undone — to plan with an end date, create a new activity. |
| Workflow lock | It can't change while the activity is submitted or approved. |
| Line items (new form) | Save the activity to add its line items |
| Footer | **Save** · Saving… · **Save and add another** · **Cancel** |
| Toasts | Activity '<name>' created · Activity '<name>' created — **View** · Activity '<name>' saved · Activity '<name>' moved to '<initiative>' · Activity '<copy name>' created |
| Validation | Enter a name. · Use 120 characters or fewer. · '<initiative>' already has an activity called '<name>'. · Use 2,000 characters or fewer. · Choose an activity type. · Choose an activity type that the model '<model>' allows. · Choose an active person. · Choose an active approval workflow. · The approval workflow can't change while the activity is submitted or approved. · Enter a valid date. · End date must be on or after the start date (<start>). · Dates must be inside the initiative's dates (<start> – <end>). · The end date can't be after <31 Dec Y+10> — 10 years after the plan's year. · Planned cost can't be negative. · Enter the planned cost in USD. · Enter an amount, such as 12,500.00. · Enter a whole number of 0 or more. · Revenue can't be negative. · This activity will need approval again — confirm the change to save it. · Reload the page and try again. |
| Not found | This activity no longer exists — someone may have deleted it. · This initiative no longer exists — someone may have deleted it. · This plan no longer exists — someone may have deleted it. · **Go to Plans** |
| Header | Activity (level tag) · Ongoing from <start> · Unassigned planned cost · Planned cost minus the line items' planned costs in USD. · Line items exceed the activity's planned cost by <amount> · Created by <person> on <date>, last changed by <person> on <date> · **Edit** · **Submit for approval** · **More** · Copy · Move · View history · Delete |
| Banner | <Submitted for approval / Approved / In progress / Complete> — changing planned cost or dates will restart approval. |
| Submit | Submitting needs <n> things — view list (1: "Submitting needs 1 thing — view list") · This activity is ready. Sending activities for approval is coming soon. |
| Completeness | All filled in · 1 field needed · <n> fields needed · 1 line item needs attention · <m> line items need attention |
| Ready to submit panel | Ready to submit · This activity · Line items · Good to know · <Label> — needed to submit · <Label> shares add up to <total> % — they must add up to 100.00 % · Dates are outside the initiative's dates (<start> – <end>) · Activity type '<type>' isn't allowed by the plan's model '<model>' — choose another type · Approval workflow '<workflow>' is no longer active — choose an active one · Line items exceed the activity's planned cost by <amount> · <amount> not yet assigned to a line item · Line item '<name>' (<id>): <problem> (texts from 007) |
| Details | — · (the activity name is used) · Ongoing (no end date) · (inactive) · Not allowed by the plan's model |
| Shell attributes | Shown because the plan's business unit includes <unit> · <Lubricants and Mobility> attributes appear once the plan's business unit is set · **Open the plan** |
| Goals | INQ · MQL · Revenue · Set later stages myself · Calculated · Typed · MQL = INQ × <rate> % · deals = MQL × <rate> % · revenue = deals × <deal size> — each rounded to a whole number (model '<model>'). · These typed values will be replaced when you save: MQL <old> → <new>; Revenue <old> → <new>. |
| Line items / Approval placeholders | Line items · Line items will appear here — adding them is coming soon. · Approval · Approval workflow: <name> · Approval steps and decisions will appear here |
| From finance | From finance (coming later) · Total actuals · Total invoiced · Total purchase requests · Filled when the Finance module and GSAP are connected |
| Re-approval dialog | This activity will need approval again · Saving returns '<name>' to Created; it must be submitted for approval again. · <Label>: <old> → <new> · <n> line items will be re-spread over the new dates; <m> custom spreads are kept as typed. · **Save and restart approval** · **Keep editing** |
| Delete dialog | Delete activity '<name>'? · <n> line items go with it. · 1 line item goes with it. · The initiative, campaign and plan planned costs go down by <amount>. · It leaves the approvers' inbox. · If it's restored later from the recycle bin, it comes back as Created. · It moves to the recycle bin; you can restore it. · **Cancel** · **Delete activity** |
| After delete | Activity '<name>' moved to the recycle bin · **Undo** (001) |
| Move dialog | Move activity '<name>' · Choose another initiative in '<plan>'. The activity keeps its status, line items and values. · Search initiatives · (this initiative) · There's no other initiative in this plan to move it to. · **Move activity** · **Cancel** |
| Move / copy refusals | '<activity>' runs <start> – <end>, but '<initiative>' runs <start> – <end>. Change the activity's dates first, or pick another initiative. · '<initiative>' already has an activity called '<name>'. Rename the activity first, or pick another initiative. · It's already in this initiative. · Choose another initiative of this plan. |
| Copy dialog | Copy activity '<name>' · Copy to · The copy starts as Created, with copies of its <n> line items. Approvals, comments and history are not copied. · **Copy activity** · **Cancel** |
| Undo refused / restore notes (API) | Can't undo — there is now another activity called '<name>'. Restore it from the recycle bin instead. · Renamed to '<new name>' because the name was taken. · Status reset to Created because it was restored from the recycle bin. |
| Log details (API) | Approval restarted because the <fields> changed. · Copied from '<name>' (<id>) · , with <n> line items |
| Tree row | Activity · Line items exceed the planned cost · Actions for <name> · View · Edit · Copy · Move · Delete · **Add activity** |

## Business rules and the tests that prove them (constitution V)

Go service tests (`package activities`, fakes of deps, fixed clock 12 Mar 2026), names carry the FR:

| Rule | Test |
|---|---|
| Name and type only are required; other fields may stay empty | `create_test.go` `TestCreate_OnlyNameAndTypeRequired_FR003` |
| Created status, readable A- ID, one `created` entry with the actor | `TestCreate_CreatedWithReadableIDAndLogged_FR004` |
| Roll-ups include the new activity after create, move, copy, delete, Undo | `provided_test.go` `TestRollups_FollowEveryChange_FR005` |
| Name unique in the initiative ignoring case and spaces; external name optional | `TestCreate_NameUniqueInInitiative_FR006`, `TestCreate_ExternalNameOptional_FR006` |
| Type must be active and allowed by the model; no types → empty list in the form | `TestCreate_TypeActiveAndAllowed_FR007`, `TestNewForm_OnlyActiveAllowedTypes_FR007`, `TestNewForm_ModelWithoutTypes_EdgeCase` |
| Form defaults: owner and dates from the initiative, single active workflow | `TestNewForm_Defaults_FR008_FR009_FR012` |
| End ≥ start; inside the initiative's dates; end ≤ 31 Dec Y+10; all problems at once | `dates_test.go` `TestCheckDates_*_FR009`, `TestCreate_AllProblemsAtOnce_FR009` |
| Ongoing clears the end date, checks the start only, cannot be undone | `TestCreate_OngoingClearsEndDate_FR010`, `TestUpdate_OngoingCannotBeUndone_FR010` |
| Planned cost ≥ 0, 0.00 counts as filled, rounded half-up to cents, USD only | `TestCreate_PlannedCostRules_FR011`, `readiness_test.go` `TestCheck_ZeroPlannedCostIsFilled_FR011` |
| Workflow active; changeable only while Created or Declined | `TestCreate_WorkflowMustBeActive_FR012`, `TestUpdate_WorkflowLockedWhenSubmittedOrApproved_FR012` |
| Inactive type, owner, workflow, options kept when unchanged; refused when newly chosen | `TestUpdate_InactiveValuesKeptWhenUnchanged_FR013` |
| Hidden-group values kept on save and not counted; reappear | `TestUpdate_HiddenGroupValuesKept_FR015`, `TestCheck_HiddenGroupsNotCounted_FR015` |
| Attribute values checked on save (unknown option, newly chosen inactive option, dependent option not fitting its parent, invalid share); a total ≠ 100.00 % only blocks submitting | `TestCreate_AttributeValuesChecked_FR016_FR017`, `TestCheck_SplitNot100Listed_FR017` |
| The answer carries the derived values (path, display status, unassigned planned cost, line-item totals, completeness, readiness); goals use the model's current values | `view_test.go` `TestView_DerivedValues_FR019_FR020`, `TestView_GoalsUseCurrentModelValues_FR018` |
| MQL/revenue from the model (250,000 → 5,000 → 2,500,000.00), "—" without INQ, typed values kept, recalculated with new model values | `goals_test.go` `TestGoals_CalculatedFromModel_FR018`, `TestGoals_TypedValuesKept_FR018`, `TestCreate_GoalsMustBeWholeAndNotNegative_FR018` |
| Display status In progress / Complete from today; never Complete when ongoing | `status_test.go` `TestDisplayStatus_FR021` |
| Completeness counts the "This activity" items and line items needing attention | `TestCompleteness_FR022` |
| The check lists everything at once in the specified groups and order; "Ready to submit" when empty | `TestCheck_ListsEveryProblemAtOnce_FR023` (fake line items: missing GL code, custom spread, negative balance), `TestCheck_ReadyWhenNothingMissing_FR023`, `TestCheck_DatesOutsideAndTypeNotAllowed_EdgeCase`, `TestCheck_InactiveWorkflow_FR023` |
| A positive unassigned planned cost is advice only; negative blocks | `TestCheck_PositiveUnassignedIsAdvice_FR024`, `TestCheck_NegativeUnassignedBlocks_FR024` |
| Every saved change logged old → new (attributes per attribute); no change → no version, no log; out-of-date save refused | `update_test.go` `TestUpdate_LogsEveryChange_FR026`, `TestUpdate_NoChangeNoLog_FR026`, `TestUpdate_VersionConflict_FR026` |
| Re-approval change needs confirmation, then Created, reason logged, hooks called; date change re-spreads | `TestUpdate_ReapprovalNeedsConfirmation_FR027`, `TestUpdate_ConfirmedReapprovalReturnsToCreated_FR027` |
| Other edits keep Submitted/Approved | `TestUpdate_OtherEditsKeepStatus_FR028` |
| Dates checked only when changed (initiative dates moved) | `TestUpdate_UnchangedDatesNotChecked_EdgeCase` |
| Move: same plan only, dates must fit, name free; keeps status; logs parent | `move_copy_test.go` `TestMove_Refusals_FR029`, `TestMove_KeepsStatusAndLogsParent_FR029`, `TestDestinations_ReasonsPerTarget_FR029_FR030` |
| Copy: " (copy)", " (copy 2)", shortened to 120; Created; line items copied; one `copied` entry | `TestCopy_Names_FR030`, `TestCopy_CreatedWithLineItems_FR030`, `names_test.go` `TestCopyName_LongNameShortened_EdgeCase` |
| Delete to the bin; Undo exact (IDs, status); Undo refused when name taken; Restore → Created and renamed | `kinds_test.go` `TestDelete_MovesToBin_FR031`, `TestUndo_RestoresExactly_FR031`, `TestUndo_NameTakenRefused_FR031`, `TestRestore_ResetsToCreatedAndRenames_FR031` |
| Describe gives readable labels and texts (owner names, money, dates, options) | `TestDescribe_ReadableChanges_FR026` |
| Plan rows ordered by start date then name, no dates last | `TestListForPlan_Order_FR033` |
| CopyAll shifts years (29 Feb → 28 Feb), no log entries; TypesNotAllowed; SetStatus logs statusChanged | `provided_test.go` `TestCopyAll_ShiftsYears`, `TestTypesNotAllowed`, `TestSetStatus_LogsStatusChanged` |

Handler/contract tests (`handler_test.go`, `package activities_test`, `apitest.New(t)` on the demo seed):
`TestNewActivityForm_Contract_FR001`, `TestCreateActivity_Contract_FR004` (201; 400 with every field
problem; 404 unknown initiative), `TestListPlanActivities_Contract_FR033`, `TestGetActivity_Contract_FR019`
(A-1001: "3 fields needed", unassigned; 404), `TestEditActivityForm_Contract_FR013`,
`TestUpdateActivity_Contract_FR026` (200; 400 `restartApproval`; 409 `version_conflict`),
`TestDeleteActivity_ThenUndo_Contract_FR031`, `TestActivityDestinations_Contract_FR029`,
`TestMoveActivity_Contract_FR029`, `TestCopyActivity_Contract_FR030`; 001's `TestRoutesMatchContract` and
`TestEveryProtectedOperationRequiresSignIn_FR005` cover the new routes automatically.

Component tests (Vitest, `mockApi`): `ActivityFormPage.test.tsx` (US1 1–6, US3, US4 1–5: defaults and
labels, type fills cost and asks, all errors at once, ongoing warning and lock, save and add another,
cancel guard, edit keeps version, conflict, re-approval dialog, `?field=` focus); `GoalsFields.test.tsx`
(US3 6–7); `ActivityPage.test.tsx` (US2 1, 3–4; header, banner, submit message, sections, not found,
error, loading); `ReadyToSubmitPanel.test.tsx` (US2 2, US3 4; groups and links, "Ready to submit");
`ReapprovalDialog.test.tsx`; `MoveActivityDialog.test.tsx`, `CopyActivityDialog.test.tsx`,
`DeleteActivityDialog.test.tsx` (US5 1–4); `useActivityActions.test.tsx` (delete → plan page + Undo);
`activityTreeRows.test.tsx` (US2 5, FR-033); `activityStatus.test.ts`, `activityText.test.ts`,
`activityForm.test.ts`; `CompletenessChip.test.tsx` (new props).

End-to-end (`frontend/e2e/activities.spec.ts`, Playwright, demo seed, real clock; never asserts global
counts):

| Test | Covers |
|---|---|
| US1: add an activity with only a name and a type | Lubricants Europe › Digital & social › **Add activity**; defaults labelled; type fills cost; Save → Created, toast, page; the plan tree shows the row and the initiative's planned cost went up; duplicate name + end before start → both errors at once |
| US2: see status and everything missing | Open A-1008 (Mobility China): chips, panel lists cost centre, 90.00 % split and the missing GL code (when 007 is merged) with links; **Submit for approval** → "Submitting needs … — view list"; a link opens the form at the field |
| US3: attributes, splits and goals | Edit A-1001: Lubricants group sentence; split three countries → 33.33/33.33/33.34; change to 50 → "116.67 % — 16.67 % too much", save works, panel lists it, **Split evenly**; INQ 250,000 → MQL 5,000, revenue $2,500,000.00 |
| US4: edit an approved activity | A-1006 (used by no other test): change description → stays Approved; change planned cost → dialog → Created; both in the activity log API |
| US5: copy, move, delete and Undo | Copy A-1002 → "(copy)" Created; move the copy to another initiative; refused move with the dates message; delete with the dialog texts → plan page toast → **Undo** brings it back with the same ID |
| Accessibility | axe (WCAG 2.1 AA) on the activity page, the form with its error summary, and each dialog |

## Requirement traceability

| FR | Implemented in | Tested by |
|---|---|---|
| FR-001 | `NewForm`, route `…/activities/new`, Add activity in 005's row and panel | `TestNewActivityForm_Contract_FR001`, `activityTreeRows.test.tsx`, e2e US1 |
| FR-002 | `ActivityForm` sections, markings, placeholders | `ActivityFormPage.test.tsx` |
| FR-003, FR-004 | `Create`, form Save / Save and add another | create tests, contract test, form test, e2e US1 |
| FR-005 | `Rollups` (read-time roll-ups) | `TestRollups_FollowEveryChange_FR005`, e2e US1/US5 |
| FR-006 – FR-013 | `Create`/`Update` rules, `NewForm`/`EditForm`, form fields and dialogs | create/update tests, form test |
| FR-014 – FR-017 | 002 attribute functions inside `Create`/`Update`/`check`; `AttributeFields` | update/readiness tests, form test, e2e US3 |
| FR-018 | `goals.go`, `GoalsFields` | goals tests, `GoalsFields.test.tsx`, e2e US3 |
| FR-019, FR-020 | `Get`, `ActivityPage`, sections | contract test, `ActivityPage.test.tsx` |
| FR-021 | `status.go`, `ActivityStatusChip` | `TestDisplayStatus_FR021`, `activityStatus.test.ts` |
| FR-022 – FR-024 | `readiness.go`, `CompletenessChip`, `ReadyToSubmitPanel` | readiness tests, panel test, e2e US2 |
| FR-025 | Submit handler on `ActivityPage` | `ActivityPage.test.tsx`, e2e US2 |
| FR-026 – FR-028 | `Update`, `ReapprovalDialog`, edit-form rule | update tests, form test, e2e US4 |
| FR-029, FR-030 | `Destinations`, `Move`, `Copy`, dialogs | move/copy tests, contract tests, dialog tests, e2e US5 |
| FR-031 | `Delete`, kind `activity`, `DeleteActivityDialog`, `useUndoableDelete` | kinds tests, contract test, dialog test, e2e US5 |
| FR-032 | View history action | `useActivityActions.test.tsx` |
| FR-033 | `ListForPlan`, `activityTreeRows` | `TestListForPlan_Order_FR033`, contract test, rows test |

## Dependencies

### Expects from other features (by name; consolidation aligns names)

| From | Name | Used for |
|---|---|---|
| 001 | `history.Service` (`Log`, `MoveToBin`, `Diff`, `Register`), `history.Kind`, `history.DeleteResult`, `record.Meta`, `errs.*`, `ids.Generator`, `apitest`; web `@/components` (Button, PageHeader, Card, StatusChip, CompletenessChip, InfoTip, ActionsMenu, ConfirmDialog, EmptyState, ErrorState, FormErrorSummary, Skeleton, TextField, Tooltip, useToast), `useUndoableDelete`, `useUnsavedChanges`, `conflictMessage`, `formatDate`, `formatDateTime`, `formatMoney`; schemas `Error`, `RecordRef`, `ActorRef`, `DeleteResult`; responses `BadRequest`, `Unauthorized`, `NotFound`, `Conflict`, `Error` | Everything shared |
| 002 | Go: `internal/platform/money` (`Money`, `Amount`, `RoundCents`, `Add`, `Sub`, `Sign`, `IsZero`, `String`, `Text`); `settings.Model` (`ID`, `Name`, `InqToMqlRate`, `MqlToDealRate`, `AverageDealSize`, `AllowedActivityTypeIDs`, `Active`), `settings.ActivityType` (`ID`, `Name`, `DefaultPlannedCost *money.Money`, `Active`), `settings.Attribute`, `settings.Level` ("activity"), `settings.AttributeValues`, `settings.OptionRef`, `settings.Missing`, `settings.Goals`; functions `BusinessUnitsOf`, `Shown`, `CheckValues`, `MissingValues`, `DescribeValue`, `Waterfall`; reader methods `GetModel`, `GetActivityType`, `ListActivityTypes`, `AttributesForLevel`. OpenAPI: `Money`, `AttributeValues`, `ShellAttribute`. Web: `MoneyInput`, `AttributeFields` (props `level`, `definitions`, `values`, `businessUnits`, `errors`, `onChange`, `idFor`), `AttributeValuesView`, `waterfallFromInq`. Seed: activity types, models, attributes and options by label | Types, attributes, splits, goals, money |
| 003 | `PersonPicker` (single mode), display-currency formatter (`useDisplayMoney`); OpenAPI `PersonRef` (`id`, `name`, `initials`, `jobTitle`, `active`); `people.Person.Active` | Owner, amounts on screen |
| 004 | `plans.Plan` (`ID`, `Name`, `Year`, `ModelID`, `Status`, `Attributes`), `plans.Repository.Get`; web `PlanTree` with a child-rows extension for initiatives, `PlanTreeRow`, `LevelTag`, route `/plans/:planId`, query key `['plan', planId]`; the activity seam of the plans service (to receive `Rollups`, `CopyAll`, `TypesNotAllowed`) | Context, roll-ups, tree |
| 005 | `campaigns.Campaign` (`ID`, `PlanID`, `Name`, `StartDate`, `EndDate`), `campaigns.Initiative` (`ID`, `CampaignID`, `Name`, `OwnerID`, `StartDate`, `EndDate`), repository `GetInitiative`, `GetCampaign`, `CampaignsInPlan`, `InitiativesInCampaign`; history kind `initiative`; web `DateRangeField`, `campaignHref`, `initiativeHref`, the initiative row actions and initiative details panel (to add **Add activity** and the Activities list), query key `['plan-tree', planId]`; the activity seam of the campaigns service; the seed's initiatives (incl. the one it deletes) | Parent, dates, tree, copies |
| 007 | Implements `activities.LineItems` (`Facts`, `CopyAll`, `DatesChanged`) and replaces `ActivityLineItemsSection`; `lineItemFieldHref` | Line items, balance, problems |
| 008 | Implements `activities.Workflows` and `activities.ApprovalHooks`, takes over table `approval_workflows`, replaces `ActivityApprovalSection` and the Submit handler | Workflows, approvals |

### Provides to later features

Go: `activities.Activity`, `Status`, `DisplayStatus`, `Repository`, `Rollup`, `SubmitCheck`,
`SubmitProblem`, `StatusChange`, the methods of *Provides to other features (Go)*, the seams `LineItems`,
`ApprovalHooks`, `Workflows` and history kind `activity`. API: the ten operations and the schemas of the
fragment (`Activity`, `ActivitySummary`, `ActivitySubmitCheck`, …). Web: routes and `activityHref(planId,
id)`, `activityEditHref(planId, id, field?)`, `ActivityStatusChip`, `activityStatus.ts`,
`ReadyToSubmitPanel`, `useActivityActions`, `activitiesApi.ts` hooks and query keys; slots
`ActivityLineItemsSection` (007) and `ActivityApprovalSection` (008).

## Complexity Tracking

| Choice | Why needed | Simpler alternative rejected because |
|---|---|---|
| Form endpoints (`activity-form`, `form`) | The form needs plan, model, allowed types, workflows, attributes and sibling names; R7 and R9 must be decided by the server | Five client calls to other features' endpoints would couple the web app to their shapes and repeat R7/R9 in TypeScript |
| Seams `LineItems`, `ApprovalHooks`, `Workflows` with no-op/seed defaults | 007 and 008 are built after 006 and import `activities`; Go forbids the import cycle the other way | Building 006 with line items and approvals inside would merge three features |
| Seed-only workflows table before 008 | FR-012 needs workflows to exist now | A hard-coded list would contradict "configuration is data" (constitution II) and change IDs later |
| `CompletenessChip` props `below`, `onOpen` | FR-022 needs line items in the chip and the panel on click | A second chip component would split one pattern into two |

## Consolidation (2026-10-19, binding)

The Plan-module consolidation (the cross-feature analyze step) settled the points below. They and the build guide,
[docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md), win where this plan, its data model or its tasks
say otherwise (D-200).

- **Contract (D-192).** The fragment is merged into `backend/api/openapi.yaml`; the merge task (T001) is done. The
  feature's operations are listed in `notYetServed` in `backend/internal/app/app_test.go`; the backend removes each one in
  the change that serves it. Agents do not change the contract; a problem with it is reported.
- **Decisions.** This plan's proposed decisions are numbered D-102–D-112 (Proposed D-H is D-091, recorded with 005) in `docs/sdd/decision-log.md` (labels kept in the
  table above); the consolidation decisions are D-189–D-207.

- **Seams for 007 and 008 (D-196).** Export `(*Service).RestartApproval(ctx, a, activityID string, reasons []string) error` (runs in
  the caller's `Write`; Submitted/Approved → Created; one `statusChanged` entry with the reasons; calls
  `ApprovalHooks.ApprovalRestarted`) and add `ApprovalHooks.ActivityRestored(ctx, a, activityID)`, called after a restore from the
  recycle bin (not after Undo). `Update` uses `RestartApproval` itself.
- **Copies (D-195).** `activities.LineItems.CopyAll(ctx, a, fromActivityID, toActivityID string, yearShift int)`; 006 implements
  `campaigns.Activities.CopyAll` with the same `yearShift` argument; plan copies use 006's own `plans.Level` `activity`.
- **Re-approval convention (D-106, D-189).** `restartApproval` + `400 validation_failed` on `restartApproval` is the convention for 007
  too.
- **Addresses (D-194).** Add the short route `/activities/:activityId` (finds the plan, redirects to
  `/plans/:planId/activities/:activityId`, keeps the hash) and the `recordHref` entries for activities.
- **Completeness chip (D-199).** Use 004's `CompletenessChip` `itemsBelow` (count) instead of new `below`/`onOpen` props.
- **Names (D-193, D-197).** Use 002's names from its mapping table (`Model`, `ActivityTypes`, `Attributes`, `CheckAttributes`,
  `MissingAttributes`, `BusinessUnit`, `Visible`, `DescribeAttribute`, `settings.Waterfall(inq, model)`, `Amount.Format`,
  `Money.Format`), 002's `MoneyField` and `ShellAttributeFields` (input ids `attributes.<attributeId>`), 004's `FormSection`,
  `FormFooter`, `LevelTag`, `TreeTable`, 005's `DateRangeField`, 003's `PersonPicker`. No `RoundCents`: the form rounds typed
  amounts and says so; the API refuses more than two decimals (D-190).
- **Approval workflows (D-103, D-201).** Seed-only workflows `WF-1001`, `WF-1002` are written in 008's table shape
  (`approval_workflows`, 008 data-model §1) so 008 takes the table over unchanged.
- **Statuses.** `ActivityStatus` (stored: created, submitted, declined, approved) and `ActivityDisplayStatus` (+ inProgress, complete)
  are 006's; 009 and 010 show the display status.
