# Implementation Plan: Campaigns and Initiatives

**Branch**: `005-campaigns-and-initiatives` | **Date**: 2026-10-19 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/005-campaigns-and-initiatives/spec.md`

**Design artifacts**: [data-model.md](data-model.md) · [contracts/openapi.yaml](contracts/openapi.yaml) (a
fragment merged into `backend/api/openapi.yaml`) · [tasks.md](tasks.md). Conventions are feature 001's
([plan](../001-app-shell-and-sign-in/plan.md), [research](../001-app-shell-and-sign-in/research.md),
[data model](../001-app-shell-and-sign-in/data-model.md)); this plan repeats them only where 005 adds to them.

## Summary

Feature 005 adds the two middle levels of the planning hierarchy and the **roll-up rules** the rest of the
module relies on:

- **Go API**: a new package `backend/internal/campaigns` (campaigns and initiatives: create, view, edit,
  delete with Undo, move, copy, destinations for the move and copy dialogs) following the samples pattern
  (handler → service → repository, `record.Meta`, `history.Log`, `history.MoveToBin`); a pure package
  `backend/internal/rollup` that defines every container calculation once (remaining budget, planned
  cost, budget used, left to plan, approval progress, timing, goal sums); a small shared calendar-date
  package `backend/internal/platform/date`. Everything this package needs from features 002, 003, 004 and
  006 comes through four small consumer-side interfaces wired in `internal/app`, so there is no import
  cycle (campaigns ↔ activities) and features can be built in order.
- **Web app**: campaign and initiative rows, the **Add campaign** / **Add initiative** entry points and the
  totals row inside feature 004's plan tree; side panels for create, details and edit (each with its own
  address); move, copy and delete dialogs; derived status chips; one new shared component,
  `DateRangeField`.

## Technical Context

**Language/Version**: Go 1.27 (API); TypeScript 5.9 strict + React 19.3 (web app) — as feature 001

**Primary Dependencies**: no new ones. API: standard library (tests: libopenapi-validator, as 001). Web
app: react-router 8, TanStack Query 5, radix-ui, lucide-react, openapi-fetch (001, research R12)

**Storage**: JSON file store, two new tables `campaigns` and `initiatives` (relational shape: `planId`,
`campaignId`), readable-ID counters `C` and `I`; derived values are never stored

**Testing**: Go `testing` (service tests with fakes and a fixed clock, `apitest` contract tests); Vitest +
Testing Library with `mockApi`; Playwright + axe (`frontend/e2e/campaigns.spec.ts`)

**Target Platform**: developer laptops (Windows 11 first); Chrome and Edge

**Project Type**: web application (React SPA + Go REST API)

**Performance Goals**: SC-008 — a plan with 20 campaigns and 100 initiatives lists, expands fully and shows
updated totals after a save in under 1 s: one `GET /api/plans/{planId}/campaigns` reads the plan's
campaigns, initiatives and activity summaries once and computes every roll-up in one pass (linear in the
number of records)

**Constraints**: servers on `127.0.0.1` only; no personal data in server logs; exact decimals for money
(no floats anywhere, including the web app's live remaining budget); deterministic tests (fixed clock
2026-03-12 09:00 UTC in Go, UTC and fake timers in Vitest)

**Scale/Scope**: 2 stored record types, 15 API operations, 6 web routes (panels inside the plan page),
3 dialogs, 1 new shared component, 12 seed records

## Constitution Check

*GATE: checked before design and again after it.*

| Principle | How this plan complies | Result |
|---|---|---|
| I. Spec first | Every element traces to an FR, scenario or edge case (Requirement traceability). The spec has no `[NEEDS CLARIFICATION]`; clarify skipped as allowed. Choices the spec leaves open are listed as proposed decisions | Pass |
| II. Shell business, plain language | Campaign, Initiative, Approval progress, Timing, Remaining budget, Planned cost, Budget used, Left to plan (glossary); no Hive9 words; every text in *User-visible messages*. Shell attributes come from Settings (002), never hard-coded | Pass |
| III. Simple, friendly experience | Standard CRUD pattern with side panels (design system §6.2); details before edit; all problems at once; unavailable move destinations listed with reason and fix; status and completeness on every row and panel; delete to the recycle bin with Undo; unsaved changes protected | Pass |
| IV. Simple, layered architecture | One business package `campaigns` (handler → service → repository); calculations in the API (`rollup`), the web app only repeats the remaining-budget subtraction for live feedback while typing (allowed: "faster feedback"); contract-first, generated web types; storage behind `campaigns.Repository` | Pass |
| V. Tested business rules | A Go test per rule named with its FR; hand-calculated money cases (SC-004); a contract test per operation; component tests per panel and dialog; Playwright journeys for US1–US4 (P1) plus US5–US6 | Pass |
| VI. Safe by default | Actor passed to every service method and logged with every change; no role checks (FR-031, D-004); optimistic concurrency on edit and move (D-030); deletes go to the recycle bin; budgets exact USD decimals | Pass |
| VII. Keep it small | No new dependency; one new shared web component with a reason; derived values computed on read instead of a cache; the interfaces in `deps.go` exist only to break the campaigns ↔ activities cycle and decouple build order (Complexity Tracking) | Pass |

**Post-design re-check**: Pass — no new runtime part, no new library, no stored totals.

## Decisions

The spec has no `[NEEDS CLARIFICATION]`. Open questions use the decision-log defaults: **Q-003**
(fictitious attribute options), **Q-005** (markings as in the field catalogue: campaign owner, dates,
cluster and region needed to publish; initiative owner and dates needed to publish), **Q-004** (fiscal year
= calendar year, D-011). New decisions, numbered at consolidation (2026-10-19):

| # | Proposed decision | Reason |
|---|---|---|
| D-090 | **Roll-ups are computed by the Go API on every read, never stored.** One pure package, `backend/internal/rollup`, defines them; the campaigns service calls it for every campaign, initiative and plan total it returns, and features 004, 009 and 010 call the same functions. Sums of USD amounts are exact (no rounding); **budget used** = planned cost × 100 ÷ budget rounded half up to a whole percent (null without a budget, with a budget of 0.00 or without a planned cost); "over budget" (red) is the exact comparison planned cost > budget; **left to plan** = budget − planned cost. | Stored totals go stale after every change below (006, 007, rate changes in 002); demo-sized data makes recomputation cheap; one definition prevents the views from disagreeing (SC-004, 010 SC-002) |
| D-091 | **"Today" for derived date statuses** (Timing here; In progress / Complete in 006) **is the API clock's calendar date in UTC.** | One deterministic rule, testable with the fixed clock; the web app never re-derives statuses |
| D-092 | **Dates count as one needed-to-publish field, "Dates"** (start and end are saved both or neither). | Matches the spec's "2 fields needed" for a campaign missing cluster and region, and the single date-range control |
| D-093 | **The API applies no defaults on create**: owner from the parent, dates from the plan or campaign and yes/no = No are filled in and labelled by the form (FR-003) and sent explicitly; the API stores what it receives. | The default is visible to the user before saving; no hidden server behaviour; import (012) chooses its own defaults |
| D-094 | **Region is never filled in by the API.** The form fills a cluster's only region and removes a removed cluster's regions (R8, through 002's attribute fields); the API only validates that every region belongs to a chosen cluster. | No silent server-side changes to what the user saw |
| D-095 | **The date-containment checks run only when the dates being saved change**: R3 on an initiative save, FR-007 on a campaign or initiative save. A save that keeps the dates is never refused because of them. | A record that came back from the recycle bin outside its parent's dates can still have other fields edited (same idea as 006's edge case) |
| D-096 | **Single campaign and initiative copies use the move destinations (FR-023)** — a campaign into its own plan or another plan of the same year; an initiative into a campaign of the same plan whose dates contain it — without the name rule (the copy's name is chosen in the dialog). This narrows D-033's "copies may target any plan" for single copies, as FR-026 says; bulk copy (012) keeps D-033. | FR-026 refers to FR-023; copies never change dates (spec Assumptions) |
| D-097 | **A copy writes one `copied` log entry, on the new top record**, with the source and what came with it ("Copied from 'Helix launch' (C-1001), with 2 initiatives, 7 activities and 15 line items"); the copied records below get no entries of their own. | One entry per business change (001 SC-004), mirroring delete's single entry |
| D-098 | **Each campaign and initiative panel has its own address, including create and edit**: `/plans/:planId/campaigns/new`, `…/campaigns/:campaignId`, `…/campaigns/:campaignId/edit`, `…/campaigns/:campaignId/initiatives/new`, `/plans/:planId/initiatives/:initiativeId`, `…/initiatives/:initiativeId/edit`. A panel opened at an address whose plan is not the record's plan (after a move) replaces the address with the right one. | FR-017; Back and the unsaved-changes guard (`useBlocker`) work without extra code |
| D-099 | **Messages written by the API show dates as `DD MMM YYYY`** ("01 Mar 2026"; ranges in one year as "01 Feb – 30 Nov 2026"), whatever date format the person chose in feature 003. | The API does not know the person's preference when it builds a validation message; the web app formats every other date |
| D-100 | **Move and copy dialogs read their destinations from the API** (`GET …/destinations?purpose=move\|copy`), which marks unavailable destinations with the reason and fix, the budget effect and, for campaigns, the model and business-unit consequences; the API checks the same rules again on the move or copy itself. | The server is the source of truth for every rule (constitution IV); the dialog never guesses |
| D-101 | **New shared calendar-date package `backend/internal/platform/date`** (type `date.Date`, JSON `YYYY-MM-DD`), created by this feature — the first in the build order with dates — and used by 006, 007, 010 and 012. | Dates are compared, shifted (plan copy, 29 Feb → 28 Feb) and formatted in several packages; one tested type avoids time-zone mistakes |

## Project Structure

### Documentation (this feature)

```text
specs/005-campaigns-and-initiatives/
├── spec.md · checklists/requirements.md
├── plan.md                  # this file
├── data-model.md            # stored records, derived values, kinds, seed
├── contracts/openapi.yaml   # fragment merged into backend/api/openapi.yaml
└── tasks.md
```

### Source code

```text
backend/
├── api/openapi.yaml                       # + the fragment's paths and schemas (shared file)
├── seed/seed.go · seed_test.go            # + addCampaigns (shared file)
└── internal/
    ├── app/app.go                         # + wiring and kind registration (shared file)
    ├── app/campaign_adapters.go           # NEW: plans/people/settings/activities → campaigns.Deps
    ├── platform/date/date.go · date_test.go   # NEW shared package (D-101)
    ├── rollup/rollup.go · rollup_test.go      # NEW pure roll-up rules (D-090)
    └── campaigns/                         # NEW business package
        ├── model.go          # stored Campaign, Initiative; API views and inputs
        ├── deps.go           # PlanSource, PersonSource, Settings, Activities + NoActivities
        ├── repository.go · store_json.go
        ├── views.go          # builds CampaignView / InitiativeView / PlanCampaigns with rollup
        ├── validate.go       # field rules, date rules (R3, FR-007), name rules, messages
        ├── service.go        # create, read, update, delete
        ├── moves.go          # destinations, move, copy, CopyPlanContents, name suggestions
        ├── provide.go        # PlanRollups, InitiativeFacts, InitiativesOfPlan (for 004, 006, 009, 010, 012)
        ├── kinds.go          # history kinds campaign and initiative
        ├── handler.go
        ├── fakes_test.go     # fake PlanSource, PersonSource, Settings, Activities for service tests
        ├── service_test.go · moves_test.go · views_test.go · kinds_test.go
        └── handler_test.go   # package campaigns_test (apitest)

frontend/
├── e2e/campaigns.spec.ts                  # NEW
└── src/
    ├── app/router.tsx                     # + six child routes under /plans/:planId (shared file)
    ├── components/DateRangeField/         # NEW shared component (+ export in index.ts)
    ├── lib/format.ts · format.test.ts     # + formatDateRange (shared file)
    └── features/campaigns/                # NEW
        ├── texts.ts                       # every user-visible text of this feature
        ├── campaignApi.ts                 # queries, mutations, query keys, refreshPlanning
        ├── useCampaignTreeContent.tsx     # rows, totals row, Add campaign, dialogs for 004's PlanTree
        ├── campaignRows.ts                # pure: API data → 004's TreeNode[] (+ test)
        ├── extensions.ts                  # the slots feature 006 fills (initiative actions, activity rows)
        ├── StatusChips.tsx                # ApprovalProgressChip, TimingChip, OutsideYearNote (+ test)
        ├── BudgetCost.tsx                 # RemainingBudgetLine, BudgetCostSection, GoalsSection (+ test)
        ├── CampaignForm.tsx · InitiativeForm.tsx       # shared by create and edit
        ├── NewCampaignPanel.tsx · CampaignPanel.tsx     # (+ tests) details and edit modes
        ├── NewInitiativePanel.tsx · InitiativePanel.tsx # (+ tests)
        ├── MoveDialog.tsx · CopyDialog.tsx · DeleteRecordDialog.tsx   # (+ tests)
        ├── useRecordActions.ts            # opens the three dialogs from rows and panels
        └── *.module.css
```

**Structure decision**: the web-application layout of `docs/architecture/repository-layout.md` (package
`campaigns`, folder `features/campaigns`). Shared files this feature extends are listed under
*Shared files*.

## Backend design

### Dependencies through small interfaces (`campaigns/deps.go`)

The service needs facts from four other features. Each is a **consumer-side interface** defined in
`campaigns`; `internal/app/campaign_adapters.go` implements them on the real packages. Service tests use
fakes (`fakes_test.go`).

```go
// Deps holds what campaigns needs from other features. internal/app fills it.
type Deps struct {
    Plans      PlanSource   // feature 004
    People     PersonSource // features 001/003
    Settings   Settings     // feature 002
    Activities Activities   // feature 006; NoActivities{} until then
}

type PlanFacts struct {
    ID, Name   string
    Year       int
    Status     string                   // "draft" | "published"
    OwnerID    *string
    Budget     *money.Money             // USD; nil when none
    ModelID    string
    ModelName  string
    Attributes settings.AttributeValues // the plan's Shell attribute values (business unit, R9)
}
type PlanSource interface {
    Plan(ctx context.Context, id string) (PlanFacts, bool, error)
    PlansOfYear(ctx context.Context, year int) ([]PlanFacts, error) // every plan of that year, any status
}

type PersonFacts struct{ ID, Name, Initials string; Active bool }
type PersonSource interface {
    Person(ctx context.Context, id string) (PersonFacts, bool, error)
}

type Settings interface {
    // CheckAttributes returns every problem with values for level ("campaign" | "initiative"), field
    // "attributes.<id>", given the stored values (inactive options already held stay allowed) and the
    // plan's values (business-unit conditions). Includes "a region must belong to a chosen cluster".
    CheckAttributes(ctx context.Context, level string, values, stored, plan settings.AttributeValues) ([]errs.FieldError, error)
    // MissingAttributes lists the shown, active, needed-to-publish attributes that are empty, in form
    // order; a dependent attribute whose chosen parent options have no options set up is not missing (D-034).
    MissingAttributes(ctx context.Context, level string, values, plan settings.AttributeValues) ([]MissingField, error)
    // AttributeChanges returns one history.Change per changed attribute, field "attributes.<id>".
    AttributeChanges(ctx context.Context, before, after settings.AttributeValues) ([]history.Change, error)
    // DescribeAttribute gives the label and readable text of an "attributes.<id>" field for the log.
    DescribeAttribute(ctx context.Context, field string, value any) (label, text string, err error)
    // GroupChanges names the attribute groups (all levels below a plan) hidden or shown when records move
    // from a plan with values from to a plan with values to (R9), e.g. "Lubricants attributes (activities)".
    GroupChanges(ctx context.Context, from, to settings.AttributeValues) (hidden, shown []string, err error)
    // AllowedActivityTypes returns the ids of the activity types the model allows (R7).
    AllowedActivityTypes(ctx context.Context, modelID string) (map[string]bool, error)
    ActivityTypeName(ctx context.Context, typeID string) (string, error)
}

type ActivitySummary struct {
    ID, InitiativeID, Name, TypeID string
    PlannedCost          *money.Money   // USD; nil when empty
    Approved             bool           // stored status Approved (shown Approved, In progress or Complete)
    StartDate, EndDate   *date.Date     // EndDate nil for ongoing activities
    Ongoing              bool
    INQ, MQL             *int64         // as feature 006 shows them (calculated or typed)
    Revenue              *money.Money
    Incomplete           bool           // its own completeness is not "All filled in"
    LineItems            int
    IncompleteLineItems  []errs.RecordRef
}
type ActivityCopyCounts struct{ Activities, LineItems int }
type Activities interface {
    // Summaries returns one summary per activity under the given initiatives (any order).
    Summaries(ctx context.Context, initiativeIDs []string) ([]ActivitySummary, error)
    // CopyAll copies every activity of fromInitiativeID, with its line items, into toInitiativeID per R13
    // (new IDs, Created, no approvals, comments, history or GSAP values), moving every date by shiftYears
    // (0 or 1), inside the caller's Write; it writes no log entries.
    CopyAll(ctx context.Context, a actor.Actor, fromInitiativeID, toInitiativeID string, shiftYears int) (ActivityCopyCounts, error)
}
// NoActivities is the Activities of a build without feature 006: no activities, nothing to copy.
type NoActivities struct{}
```

All calls run inside the caller's transaction (the context carries it), so adapters simply call the other
packages' repositories or services.

### Shared date package (`platform/date`, D-101)

| Name | Behaviour |
|---|---|
| `type Date struct{ … }` | A calendar date without time zone. JSON `"2026-03-01"`; `*date.Date` encodes `null`. Decoding anything else fails (the request answers `400 bad_request`) |
| `Parse(s string) (Date, error)`, `MustParse(s) Date`, `New(y, m, d) Date`, `(Date) String()` | `YYYY-MM-DD` |
| `(Date) Before(o)`, `After(o)`, `Compare(o) int`, `Year() int` | |
| `Today(now time.Time) Date` | The date of `now` in UTC (D-091) |
| `YearStart(y)`, `YearEnd(y)` | 1 Jan, 31 Dec |
| `(Date) AddYears(n int) Date` | 29 Feb → 28 Feb when the target year has none |
| `Format(d) string` | `"01 Mar 2026"` (two-digit day, fixed English month list, never "Sept") |
| `FormatRange(start, end) string` | Same year: `"01 Feb – 30 Nov 2026"`; otherwise `"01 Dec 2025 – 31 Jan 2026"` |
| `FormatRangeFull(start, end) string` | Always both years: `"01 Mar 2026 – 31 Dec 2026"` (the R3 message) |

### Roll-ups (`internal/rollup`, D-090)

**Who** computes: the API, in the pure functions below (no storage, no clock of their own), called by the
campaigns service when it builds a view (`views.go`) and by later features (004 plan cards, 009 home, 010
grid). **When**: on every read — nothing derived is stored, so every change below (an activity's planned
cost, a line item, a move, a delete, an exchange-rate change upstream) shows on the next read. **Rounding**:
money sums are exact (USD amounts with 2 decimals); the only rounding is the whole-percent budget used,
half up. The web app never recomputes them; it refreshes after each change (FR-012).

```go
package rollup

// Sum adds the amounts that are present; nil when none is (nothing below has a planned cost).
func Sum(amounts []*money.Money) *money.Money
// Total adds the amounts that are present; 0.00 USD when none is (allocated budgets).
func Total(amounts []*money.Money) money.Money
// Remaining = budget − allocated; nil when budget is nil. May be negative.
func Remaining(budget *money.Money, allocated money.Money) *money.Money
// OverBy returns −remaining when remaining < 0.00, otherwise nil.
func OverBy(remaining *money.Money) *money.Money
// BudgetUsedPercent = planned × 100 ÷ budget, rounded half up to a whole number (computed on cents with
// integers: (2 × planned¢ × 100 + budget¢) ÷ (2 × budget¢)); nil when budget is nil or 0.00, or planned is nil.
func BudgetUsedPercent(planned, budget *money.Money) *int
// OverBudget reports planned > budget, compared exactly; false when either is nil.
func OverBudget(planned, budget *money.Money) bool
// LeftToPlan = budget − (planned, or 0.00 when nil); nil when budget is nil.
func LeftToPlan(budget, planned *money.Money) *money.Money

type Progress struct{ State string; Approved, Total int } // "noActivities" | "inProgress" | "allApproved"
func ApprovalProgress(approved, total int) Progress

// Timing: "" when start or end is nil; "notStarted" (today < start); "running" (start ≤ today ≤ end);
// "finished" (today > end).
func Timing(start, end *date.Date, today date.Date) string

type Goals struct{ INQ, MQL *int64; Revenue *money.Money } // each nil when nothing below has it
func AddGoals(goals []Goals) Goals

type BudgetAndCost struct {
    AllocatedToChildren  money.Money
    RemainingForChildren *money.Money
    OverAllocatedBy      *money.Money
    PlannedCost          *money.Money
    BudgetUsedPercent    *int
    OverBudget           bool
    LeftToPlan           *money.Money
}
// Container combines the above for one plan, campaign or initiative.
func Container(budget *money.Money, childBudgets []*money.Money, plannedCost *money.Money) BudgetAndCost
```

Money operations used (from feature 002's `platform/money`): zero USD value, add, subtract, compare, sign,
and the amount in cents as an integer (for the percent formula). Budgets and planned costs are always USD
(validated on input; line items are converted to USD by 007 before they reach an activity's total), so
`rollup` only ever adds USD amounts.

### Stored records and API types (`model.go`)

Stored (every field in [data-model.md](data-model.md) §1–§2):

```go
type Campaign struct {
    ID          string                   `json:"id"`
    PlanID      string                   `json:"planId"`
    Name        string                   `json:"name"`
    Description string                   `json:"description"`
    OwnerID     *string                  `json:"ownerId"`
    StartDate   *date.Date               `json:"startDate"`
    EndDate     *date.Date               `json:"endDate"`
    Budget      *money.Money             `json:"budget"`
    Attributes  settings.AttributeValues `json:"attributes"` // never nil: {} when empty
    record.Meta
}
type Initiative struct { ID, CampaignID, Name, Description string; OwnerID *string; StartDate, EndDate *date.Date;
    Budget *money.Money; Attributes settings.AttributeValues; record.Meta } // same JSON names, "campaignId"
```

API types (contract schemas in brackets): `CampaignView` (`Campaign`, embeds the stored campaign and adds
`owner`, `plan`, `outsidePlanYear`, `parentRemainingBudget`, `budgetAndCost`, `approvalProgress`, `timing`,
`completeness`, `goals`, `counts`, `initiatives []InitiativeView`), `InitiativeView` (`Initiative`, adds
`planId`, `owner`, `plan`, `campaign`, `parentRemainingBudget`, `budgetAndCost`, `approvalProgress`,
`timing`, `completeness`, `goals`, `counts`), `PlanCampaigns`, `PlanCampaignTotals`, `BudgetAndCost`,
`ApprovalSummary` (its `state` is 004's `ApprovalProgress` enum), `RecordCompleteness`, `MissingField{Field,
Label}` (004's schema, not redefined by the fragment), `GoalSums`, `CampaignCounts`,
`InitiativeCounts`, `PlanRef`, `CampaignRef`, `Destination`, `DestinationList`, `MoveConsequences`,
`TypeNotAllowed`; inputs `CampaignInput` (`CampaignCreate`), `CampaignUpdate` (with `Version *int`),
`InitiativeInput`, `InitiativeUpdate`, `CampaignMove{PlanID; Version *int}`, `InitiativeMove{CampaignID;
Version *int}`, `CampaignCopy{PlanID, Name}`, `InitiativeCopy{CampaignID, Name}`. `timing` is `*string`
(nil → `null`). Every slice is created non-nil.

### Repository (`repository.go`, JSON in `store_json.go`, tables `campaigns`, `initiatives`)

| Method | Meaning |
|---|---|
| `GetCampaign(ctx, id) (Campaign, bool, error)` · `PutCampaign(ctx, c)` · `DeleteCampaign(ctx, id)` | one campaign |
| `CampaignsOfPlan(ctx, planID) ([]Campaign, error)` | the plan's campaigns, ordered by id |
| `GetInitiative(ctx, id)` · `PutInitiative(ctx, i)` · `DeleteInitiative(ctx, id)` | one initiative |
| `InitiativesOfCampaigns(ctx, campaignIDs []string) ([]Initiative, error)` | ordered by id |

(JSON implementation: `store.List` and filter — fine for demo sizes; PostgreSQL later uses the indexes of
data-model §2.)

### Service (`campaigns.Service`)

`NewService(tx store.Transactor, repo Repository, generator *ids.Generator, hist *history.Service, now
clock.Clock, deps Deps) *Service`. Every method takes `(ctx, a actor.Actor, …)`, runs in one `tx.Read` or
`tx.Write`, and reports every problem at once: inside the Write it checks, in this order, **not found**
(404, nothing else reported) → **version** (missing: problem on `version`; out of date: `409
version_conflict` "Not saved — <person> changed this campaign") → **every field rule** collected in one
`errs.Problems` (static rules, name uniqueness, owner, attributes, date rules) → `problems.Err()` before
anything is written. No method checks roles (FR-031).

| Operation | Rules (FR) | Result and log |
|---|---|---|
| `ListPlanCampaigns(ctx, a, planID)` | Plan must exist (404 plan message). Reads the plan's campaigns, their initiatives and `Activities.Summaries` once; builds views; orders campaigns and each campaign's initiatives by start date, then name ignoring case, undated last, then id (FR-020); totals = counts, `Total(campaign budgets)`, plan budget, `Remaining`, `OverBy`, `Sum(campaign planned costs)` (FR-022) | `PlanCampaigns` |
| `GetCampaign(ctx, a, id)` / `GetInitiative` | 404 when gone | `CampaignView` / `InitiativeView` (FR-016) |
| `CreateCampaign(ctx, a, planID, in)` | Plan exists (404). Trim `name`, `description`. Name 1–120 and unique in the plan ignoring case (FR-005); description ≤ 2,000; `ownerId`, when set, an existing **active** person (FR-002); dates both or neither, start ≤ end (FR-005); budget USD, ≤ 2 decimals, ≥ 0 (FR-005, D-015: never compared with the remaining budget); `Settings.CheckAttributes("campaign", in, {}, plan)` (FR-002, FR-009). `attributes` nil → `{}`. Dates outside the plan year are allowed (FR-008). ID `ids.Next("C")`, `record.NewMeta` | 201 view; log `created` (FR-004, FR-030) |
| `UpdateCampaign(ctx, a, id, in)` | As create, plus: a stored inactive owner may stay (only a changed owner must be active); `CheckAttributes` with the stored values; **FR-007** when start or end changes and both new dates are set: every initiative with dates must have `start ≥ new start` and `end ≤ new end`, else one problem on `startDate` naming each (D-095). Allowed whatever the plan's status (FR-018). `history.Diff(stored without attributes, new without attributes)` + `Settings.AttributeChanges`; no change → answer the stored view, no version, no log | view; log `changed` with every change (FR-030) |
| `DeleteCampaign(ctx, a, id)` / `DeleteInitiative` | 404 when gone; `history.MoveToBin("campaign"\|"initiative", id)` takes everything below (FR-028) | `history.BinEntry` (handler answers `DeleteResult`); log `deleted` by history |
| `CreateInitiative(ctx, a, campaignID, in)` | Campaign exists (404 campaign message). Field rules as campaign; name unique in the campaign; **R3** (FR-006): when the campaign has both dates and the initiative has dates, `campaign start ≤ start` and `end ≤ campaign end`, else problem on `startDate` with the range | 201 view; log `created` |
| `UpdateInitiative(ctx, a, id, in)` | As create; R3 only when the dates change (D-095); **FR-007** when the dates change and are set: every activity with a start date — non-ongoing: `start ≥ new start` and `end ≤ new end`; ongoing: `new start ≤ start ≤ new end` — else one problem on `startDate` naming each | view; log `changed` |
| `CampaignDestinations(ctx, a, id, purpose)` | `purpose` must be `move` or `copy` (else problem on `purpose`). Plans of the campaign's plan year (`PlansOfYear`). **move**: every other plan; unavailable when it has a campaign with this name (ignoring case): reason "Already has a campaign called '<name>' — rename it first". **copy**: own plan first (`current: true`), then the others, all available, `suggestedName` per plan (§ Copy names). For each: `remainingAfter` = `Remaining(plan budget, allocated there + this campaign's budget)` (for the own plan on copy: allocated + budget again; move never lists the own plan), `overAllocatedBy`; `consequences` when the model or the business-unit values differ: `modelName`, `modelChanges`, `businessUnitChanges`, `typesNotAllowed` = activities below whose `TypeID` the target model does not allow (with `ActivityTypeName`), `GroupChanges(from plan attrs, to plan attrs)` (FR-024). Order: current first, then name ignoring case. `rule` texts below | `DestinationList` (FR-023, FR-026) |
| `InitiativeDestinations(ctx, a, id, purpose)` | Campaigns of the initiative's plan. **move**: the others; unavailable when (a) one has an initiative with this name: "Already has an initiative called '<name>' — rename it first", or (b) it has both dates, the initiative has dates and they do not fit: "Dates don't fit (<FormatRange>) — change the initiative's dates first" (both reasons joined with " · " when both apply). **copy**: own campaign first, then the others; only (b) makes one unavailable; `suggestedName`. `startDate`/`endDate` of each campaign; `remainingAfter`/`overAllocatedBy` against each campaign's budget; `consequences` null. Order: current first, then start date, name | `DestinationList` |
| `MoveCampaign(ctx, a, id, in)` | 404; version (409); target plan: gone → problem `planId` "That plan no longer exists — choose another one."; other year → "Campaigns move only between plans of the same year (<year>)."; current plan → "'<campaign>' is already in '<plan>'."; name taken there → "'<plan>' already has a campaign called '<name>' — rename it first." Then `PlanID` = target, `Touch`, `Put` (FR-023, FR-025). Nothing below changes | view; log `moved`, change `parent` old → new plan id (texts: plan names) |
| `MoveInitiative(ctx, a, id, in)` | Target campaign: gone → `campaignId` "That campaign no longer exists — choose another one."; other plan → "Initiatives move only between campaigns of the same plan."; current → "'<initiative>' is already in '<campaign>'."; name taken → "'<campaign>' already has an initiative called '<name>' — rename it first."; dates → "Dates don't fit (<FormatRange>) — change the initiative's dates first." (all applicable problems at once). Then `CampaignID` = target, `Touch`, `Put` | view; log `moved`, change `parent` (texts: campaign names) |
| `CopyCampaign(ctx, a, id, in)` | 404; `name` rules as create in the **target** plan; `planId`: gone / other year ("Campaigns can be copied only into plans of the same year (<year>).") — problems together. In one Write: new campaign (all stored fields of the source except id, `planId` = target, name = `in.Name` trimmed, new meta); for each initiative by id: new initiative (all fields, same name, new id, new meta) and `Activities.CopyAll(old, new, 0)`. Any error → nothing is stored (FR-027) | 201 view; log `copied` on the new campaign (D-097) |
| `CopyInitiative(ctx, a, id, in)` | `name` unique in the target campaign; `campaignId`: gone / other plan ("Initiatives can be copied only into campaigns of the same plan.") / dates don't fit (message as move); then copy + `Activities.CopyAll` | 201 view; log `copied` |

**Copy names** (`suggestCopyName(name string, taken func(string) bool) string`, R13 and edge case):
`"<name> (copy)"`, then `"<name> (copy 2)"`, `"<name> (copy 3)"`, … — the first one not taken (ignoring
case) among the destination's children. When `<name> + suffix` is longer than 120 characters, `<name>` is
cut to `120 − len(suffix)` characters and trailing spaces are trimmed.

**Views** (`views.go`) — one builder for both levels, from the stored records, the plan facts, people and
the activity summaries, using `rollup`:

| View field | Source |
|---|---|
| `owner` | `PersonSource.Person(ownerId)` → `PersonRef{id, name, initials, active}`; null when no owner |
| `plan` / `campaign` | `PlanRef{id, name, year}` / `CampaignRef{id, name, startDate, endDate}` |
| `budgetAndCost` | `rollup.Container(budget, child budgets, planned cost)`; planned cost of an initiative = `rollup.Sum(activity planned costs)`, of a campaign = `rollup.Sum(initiative planned costs)` |
| `parentRemainingBudget` | campaign: `Remaining(plan budget, Total(all campaign budgets of the plan))`; initiative: `Remaining(campaign budget, Total(all initiative budgets of the campaign))` |
| `approvalProgress` | `rollup.ApprovalProgress(count Approved, count all)` over the activities below |
| `timing` | `rollup.Timing(start, end, date.Today(now()))`; `""` → null |
| `outsidePlanYear` | campaign only: start < `YearStart(plan year)` or end > `YearEnd(plan year)` |
| `completeness.missing` | `ownerId` "Owner" when null; `startDate` "Dates" when no dates (D-092); then `Settings.MissingAttributes(level, values, plan attrs)` |
| `completeness.itemsBelowNeedingAttention` | campaign: initiatives with missing fields + activities `Incomplete` + `len(IncompleteLineItems)` below; initiative: its activities and line items likewise |
| `goals` | `rollup.AddGoals` over the activities below |
| `counts` | initiatives, activities, approved activities, `Σ LineItems` |

### Services this package provides to other features (`provide.go`, `moves.go`)

| Function | For | Behaviour |
|---|---|---|
| `(*Service).PlanRollups(ctx, planIDs []string) (map[string]PlanRollup, error)` | 004 plans list, plan page cards, publish check, delete dialog; 009 | Joins the caller's transaction (read only). `PlanRollup{Campaigns, Initiatives, Activities, ApprovedActivities, LineItems int; AllocatedToCampaigns money.Money; PlannedCost *money.Money; Goals rollup.Goals; ItemsNeedingAttention int; IncompleteBelow []errs.RecordRef; PublishProblems []RecordProblems}`, with `RecordProblems{Record errs.RecordRef; Missing []MissingField}` for every campaign and initiative with missing needed-to-publish fields, in tree order (004 FR-024). Every plan id gets an entry (zeros for a plan without campaigns) |
| `(*Service).CopyPlanContents(ctx, a, fromPlanID, toPlanID string, shiftYears int) (CopyCounts, map[string]string, error)` | 004 plan copy (FR-027, FR-028) | Inside the caller's Write: copies every campaign and initiative with the **same names**, every date `AddYears(shiftYears)`; no log entries; returns `{Campaigns, Initiatives}` and the old → new ID of every campaign and initiative (for 004's `CopyJob.NewIDs`). It does **not** call `Activities.CopyAll`: in 004's `plans.Level` registry, 006 and 007 copy their own records from `NewIDs` (single campaign and initiative copies still use `CopyAll`) |
| `(*Service).InitiativeFacts(ctx, id) (InitiativeFacts, bool, error)` and `InitiativesOfPlan(ctx, planID) ([]InitiativeFacts, error)` | 006 (R3, defaults from the initiative, move targets grouped by campaign), 012 | `InitiativeFacts{ID, Name string; OwnerID *string; StartDate, EndDate *date.Date; CampaignID, CampaignName, PlanID string}`; joins the caller's transaction |
| `CreateCampaign`, `CreateInitiative`, `MoveCampaign`, `MoveInitiative`, `CopyCampaign`, `CopyInitiative` | 012 bulk actions and import | Join the caller's transaction, so 012 can run many in one Write |
| package `rollup`, package `platform/date` | 004, 006, 007, 009, 010, 012 | As above |
| `(*Service).Kinds() []history.Kind` | `internal/app` | `campaign` (parent `plan`), `initiative` (parent `campaign`) — data-model §7 |

### History kinds and log (`kinds.go`)

As [data-model.md](data-model.md) §7–§8. `PrepareRestore` texts: Undo refused —
`errs.NameTaken("Can't undo — there is now another campaign called '<name>'. Restore it from the recycle
bin instead.")` (initiative: "another initiative"); Restore note — "Renamed to '<new name>' because the
name was taken." `Describe` uses `PersonSource` for `ownerId`, `date.Format`, money as "400,000.00 USD",
`parent` → "Plan"/"Campaign" with the parent's name (`PlanSource` / repository), and
`Settings.DescribeAttribute` for `attributes.<id>`.

### API validation and error messages

Every message appears here once; Go constants in `validate.go`, tests quote this table. `<…>` is filled in
at run time; dates per D-099.

| Field / case | Message |
|---|---|
| `name` empty | Enter a name. |
| `name` too long | Use 120 characters or fewer. |
| `name` taken, campaign | '<plan name>' already has a campaign called '<other campaign's name as stored>'. |
| `name` taken, initiative | '<campaign name>' already has an initiative called '<other name as stored>'. |
| `description` too long | Use 2,000 characters or fewer. |
| `ownerId` unknown or inactive (when changed) | Choose an active person. |
| `endDate` missing while start set | Enter an end date too, or clear the start date. |
| `startDate` missing while end set | Enter a start date too, or clear the end date. |
| `endDate` before start | End date must be on or after the start date (<DD MMM YYYY>). |
| `startDate` outside the campaign (R3) | Dates must be inside the campaign's dates (<FormatRangeFull>). |
| `startDate` FR-007, one record | <name> (<FormatRange>) would be outside these dates. Change its dates first, or choose dates that include it. |
| `startDate` FR-007, several | These initiatives would be outside these dates: <name> (<range>), <name> (<range>). Change their dates first, or choose dates that include them. (initiative save: "These activities …"; an ongoing activity shows "(ongoing from <DD MMM YYYY>)") |
| `budget` negative | Budget can't be negative. |
| `budget` not USD, not a number or more than 2 decimals | Enter an amount in USD with at most 2 decimals, such as 12500 or 12500.50. |
| `attributes.<id>` | Feature 002's attribute messages |
| `version` missing or < 1 | Reload the page and try again. |
| `purpose` | Choose move or copy. |
| Move/copy targets | the texts in the service table above |
| 404 campaign / initiative / plan | This campaign no longer exists — someone may have deleted it. / This initiative no longer exists — someone may have deleted it. / This plan no longer exists — someone may have deleted it. |
| Conflict (API message, 001) | Not saved — <person> changed this campaign / … this initiative |
| Undo refused / restore note | as *History kinds* |
| Destination `rule` | Campaigns move only between plans of the same year · Initiatives move only between campaigns of the same plan · A copy can go into this plan or another plan of the same year · A copy can go into this campaign or another campaign of the same plan |
| Unavailable reasons | Already has a campaign called '<name>' — rename it first · Already has an initiative called '<name>' — rename it first · Dates don't fit (<FormatRange>) — change the initiative's dates first |
| Copy log details | Copied from '<source name>' (<source id>)[, with <contents>] — contents as history's "with 2 initiatives, 7 activities and 15 line items" |

### API operations

All need a signed-in person (401 `sign_in_required` / `session_expired` otherwise). Error bodies are
001's `Error`.

| Method and path | Request | Response | Errors |
|---|---|---|---|
| `GET /api/plans/{planId}/campaigns` | — | 200 `PlanCampaigns` | 404 plan |
| `POST /api/plans/{planId}/campaigns` | `CampaignCreate` | 201 `Campaign` | 400 `validation_failed` (all fields) / `bad_request`; 404 plan |
| `GET /api/campaigns/{campaignId}` | — | 200 `Campaign` | 404 |
| `PATCH /api/campaigns/{campaignId}` | `CampaignUpdate` | 200 `Campaign` | 400; 404; 409 `version_conflict` |
| `DELETE /api/campaigns/{campaignId}` | — | 200 `DeleteResult` | 404 |
| `GET /api/campaigns/{campaignId}/destinations?purpose=move\|copy` | — | 200 `DestinationList` | 400 `purpose`; 404 |
| `POST /api/campaigns/{campaignId}/move` | `CampaignMove` | 200 `Campaign` | 400 (`planId`, `version`); 404; 409 |
| `POST /api/campaigns/{campaignId}/copy` | `CampaignCopy` | 201 `Campaign` | 400 (`name`, `planId`); 404 |
| `POST /api/campaigns/{campaignId}/initiatives` | `InitiativeCreate` | 201 `Initiative` | 400; 404 campaign |
| `GET /api/initiatives/{initiativeId}` | — | 200 `Initiative` | 404 |
| `PATCH /api/initiatives/{initiativeId}` | `InitiativeUpdate` | 200 `Initiative` | 400; 404; 409 |
| `DELETE /api/initiatives/{initiativeId}` | — | 200 `DeleteResult` | 404 |
| `GET /api/initiatives/{initiativeId}/destinations?purpose=move\|copy` | — | 200 `DestinationList` | 400; 404 |
| `POST /api/initiatives/{initiativeId}/move` | `InitiativeMove` | 200 `Initiative` | 400 (`campaignId`, `version`); 404; 409 |
| `POST /api/initiatives/{initiativeId}/copy` | `InitiativeCopy` | 201 `Initiative` | 400 (`name`, `campaignId`); 404 |

Handlers follow `samples/handler.go`: `web.ReadJSON` → `actor.From` → service → `web.WriteJSON` /
`web.WriteError`; lists answer `{items, total, …}`; deletes answer `history.DeleteResult{RecycleBinEntry:
entry.View()}`; Undo uses 001's `POST /api/recycle-bin/{entryId}/undo`.

### Wiring (`internal/app/app.go`, `campaign_adapters.go`)

After 004's plans service and `plan` kind: build `campaigns.NewService(db, campaigns.NewJSONRepository(),
generator, historyService, now, campaigns.Deps{Plans: …, People: …, Settings: …, Activities:
campaigns.NoActivities{}})`; register its `Kinds()` (campaign, then initiative) before 006's `activity`
kind; `campaigns.NewHandler(svc).Routes(router, signedIn)`. Feature 006 replaces `NoActivities{}` with its
adapter; feature 004 receives `svc.PlanRollups` / `svc.CopyPlanContents` through two `plans.Level` registrations (`campaign`, `initiative`) made in `campaign_adapters.go` right after the kinds (*Alignment with feature 004 as planned*).
`TestRoutesMatchContract` fails until the 15 routes and the contract agree.

### Demo data (`seed/seed.go`)

`addCampaigns(ctx, generator, createdAt)` after 004's plans and 002's attributes: the 4 campaigns and 8
initiatives of [data-model.md](data-model.md) §9 through `campaigns.NewJSONRepository()`, ids from the
generator in table order (C-1001 … C-1004, I-1001 … I-1008), `record.NewMeta(actor.System, createdAt)`,
attribute values looked up by label in 002's seed. The seed command's summary line gains ", 4 campaigns, 8
initiatives".

## Frontend design

### Routes (children of feature 004's `/plans/:planId` route, `router.tsx`)

Each element is a side panel drawn over the plan page through 004's `<Outlet />`; `handle: { title:
'Plans' }`. Closing a panel navigates to `/plans/:planId` (guarded by `useUnsavedChanges`).

| Path | Element | Opens |
|---|---|---|
| `campaigns/new` | `NewCampaignPanel` | **Add campaign** (toolbar, plan row ⋯, empty tree) |
| `campaigns/:campaignId` | `CampaignPanel` (details) | selecting a campaign row, **View**, toast **View**, FR-017 address |
| `campaigns/:campaignId/edit` | `CampaignPanel` (edit) | **Edit**; `?focus=<field id>` focuses that field (completeness links; same parameter as 004's plan form) |
| `campaigns/:campaignId/initiatives/new` | `NewInitiativePanel` | **Add initiative** (row quick action, row ⋯, details ⋯, empty initiatives) |
| `initiatives/:initiativeId` | `InitiativePanel` (details) | selecting an initiative row |
| `initiatives/:initiativeId/edit` | `InitiativePanel` (edit) | **Edit** |

When the loaded record's `planId` differs from `:planId` (it was moved), the panel navigates to the same
panel under the right plan with `replace`. Panel tab title: `usePageTitle(<record name>)`; create panels
`usePageTitle('New campaign')` / `('New initiative')`.

### Plan tree integration (feature 004 extension points)

004's plan page renders `PlanTree`; 005 fills it through one hook, `useCampaignTreeContent(planId)`,
which returns:

| Part | Content |
|---|---|
| `nodes` (004's `TreeNode[]`, set as the plan node's `children` in `usePlanTreeNodes`; *Alignment with feature 004 as planned*) | from `campaignRows(data, handlers)` — one node per campaign with its initiatives as `children`, in the API's order (FR-020); plus `extensions.useRowsBelowInitiatives(planId)` (empty until 006) |
| `toolbarActions` | **Add campaign** (secondary button with a plus icon) → `campaigns/new` (FR-001) |
| `planRowActions: ActionsMenuItem[]` | **Add campaign** (FR-001) |
| `emptyAction` | **Add campaign** under 004's empty sentence (FR-014 of 004) |
| `totalsRow` | "Total · 2 campaigns · 4 initiatives · 7 activities"; budget cell: `allocatedToCampaigns` with the label "Allocated to campaigns"; planned-cost cell: `totals.plannedCost` (FR-022) |
| `dialogs` | the Move, Copy and Delete dialogs opened from row menus (one instance) |
| `isLoading`, `error` | 004 shows its skeleton rows / ErrorState with **Try again** (`refetch`) |

Row fields (FR-019, FR-021):

| Cell | Campaign row | Initiative row |
|---|---|---|
| Level bar + tag | `LevelTag level="campaign"` (violet) | `level="initiative"` (aqua) |
| Name | link to its panel address; ID beside it; child count "2 initiatives" / "1 initiative" / "0 initiatives" | "3 activities" / "1 activity" / "0 activities" |
| Owner | avatar + name; "(inactive)" | same |
| Dates | `formatDateRange` or "—"; R4 warning icon with tooltip (FR-008) | same, no R4 |
| Status | `ApprovalProgressChip`, `TimingChip`, `CompletenessChip` (+ "n items below need attention") | same |
| Budget | budget; beneath: "<amount> left" or red "Over-allocated by <amount>"; nothing without a budget | budget only (activities have no budget) |
| Planned cost | amount or "—" | same |
| Budget used | `ProgressBar` (red when `overBudget`) + "125 %"; "—" | same |
| Actions | hover/focus quick action **Add initiative** (icon button, tooltip "Add initiative to <name>") and ⋯ `ActionsMenu`: Add initiative · View · Edit · Copy · Move to… · Delete (separated, last) | ⋯: View · Edit · Copy · Move to… · Delete; plus `extensions.initiativeExtraActions(initiative)` (006: Add activity) |
| Start expanded | yes (FR-020) | no (006's activity rows start collapsed) |

The ⋯ button is always in the DOM and focusable (keyboard), visually faded until the row is hovered or has
focus (`:focus-within`) — never hidden on touch screens. After a create, move or copy the panel or dialog
navigates to `/plans/:planId` with `state: { highlight: '<id>' }`; 004's `PlanTree` expands the ancestors,
scrolls to and highlights that row (FR-004).

### Panels

**Create** (`NewCampaignPanel`, `NewInitiativePanel`; `SidePanel`, design system §6.2): title and
description from the messages table; form below; footer **Save** (primary), **Save and add another**,
**Cancel**. Defaults pre-filled **and labelled** (FR-003): owner = `planOwner` / campaign owner ("From the
plan" / "From the campaign", shown while unchanged); dates = `YYYY-01-01`–`YYYY-12-31` of the plan year /
the campaign's dates, empty when it has none ("The plan's year" / "The campaign's dates"); yes/no
attributes = No (002's `defaultAttributeValues(level)`). Save → `POST`; success → `refreshPlanning()`,
toast "<Level> '<name>' created" with **View** (opens the details address), navigate to `/plans/:planId`
with the highlight; **Save and add another** → same, then navigate to the same `new` address with a new
location key (fresh form, highlight state kept) (FR-001, FR-004).

**Details** (`CampaignPanel` / `InitiativePanel`, design system §6.3, FR-016): header — breadcrumb
"Plans › <plan> › <campaign>[ › <initiative>]" (links), `LevelTag`, name as the panel title, chips
(approval progress, timing, completeness + "n items below need attention" link that closes the panel and
highlights the record in the tree), R4 note (campaign), facts (ID, Owner, Dates, "Created by … on …, last
changed by … on …"). Sections: **Budget and cost** (`BudgetCostSection`: budget; campaign: "Allocated to
initiatives", "Remaining for initiatives"; initiative: "Remaining campaign budget"; "Planned cost" with
"Calculated" and ⓘ; "Budget used" bar; "Left to plan") · **Goals** (`GoalsSection`: INQ, MQL, Revenue,
"Calculated", ⓘ; "—" each when null) · **Description** ("—") · Shell attribute sections (002's
`AttributeValuesView`, "—" when empty, "(inactive)") · **Initiatives (n)** (campaign: each initiative
with link, dates, approval chip, budget; empty text + **Add initiative**) / **Activities** (initiative:
"Activities: n (k approved) · Line items: m", or "No activities yet."). Footer: **Edit** (primary), ⋯
**More** (`ActionsMenu` labelled with the name): campaign — Add initiative, Copy, Move to…, View history,
Delete; initiative — Copy, Move to…, View history, Delete, plus `extensions.initiativeExtraActions`.
**View history** → `/activity-log?recordType=<campaign|initiative>&recordId=<id>` (Coming soon until 011,
FR-030). `CompletenessChip.onSelectField(id)` → the edit address with `?focus=<id>`.

**Edit** (same panel, `/edit` address, FR-018): the form with the record's values; follows 001's edit-form
rule (copies the record and `version` once). Footer **Save**, **Cancel** (back to details). Save is never
disabled (a second click while saving is ignored); on 400 every field message appears under its field and
in `FormErrorSummary` (which takes focus); on 409 the panel shows `conflictMessage(error, 'campaign')`,
keeps the typed values and takes the new `version` from a refetch; success → `refreshPlanning()`, toast
"<Level> '<name>' saved", back to the details address. Leaving with changes → 001's "Discard changes?"
dialog (`useUnsavedChanges`).

**Forms** (`CampaignForm`, `InitiativeForm`; one column; field ids `campaign-<field>` /
`initiative-<field>` with dots replaced by dashes, used by `FormErrorSummary.fieldId`):

| Section | Campaign | Initiative |
|---|---|---|
| Basics | **Campaign name** (`TextField`, Required, max 120) · **Owner** (`PersonPicker`, Needed to publish, default hint) · **Start date / End date** (`DateRangeField`, Needed to publish, default hint; R4 warning under it when outside the plan year) · **Description** (`TextAreaField`, max 2,000) | **Initiative name** · **Campaign** (read-only text, hint "Change it with Move") · **Owner** · **Dates** (`DateRangeField` with `allowed` = the campaign's dates and the hint "Must be inside the campaign's dates (…)") · **Description** |
| Budget | **Campaign budget** (`MoneyField`, USD) · `RemainingBudgetLine` "Remaining plan budget" | **Initiative budget** · "Remaining campaign budget" |
| Shell attributes (and other sections from Settings) | 002's `AttributeFields level="campaign"` with the plan's attribute values: Region disabled with "Choose a cluster first" until a cluster is chosen, only regions of the chosen clusters, a single region filled in with its message, removed clusters' regions removed with a note naming them (R8, FR-009), sections and markings from Settings (FR-002) | `level="initiative"` |

`RemainingBudgetLine` (FR-010): `shown = parentRemainingBudget + savedOwnBudget − typedBudget` with exact
cent arithmetic (`@/lib/money` from 002); "—" with "No budget to allocate from" when the parent has no
budget; negative → red `formatMoney` value and "Over-allocated by <amount>"; never blocks Save (D-015).

### Dialogs

Move and Copy use 004's `FormDialog` (a radio list and a name field are form content; first field
focused, Enter submits); Delete uses 001's `ConfirmDialog` (Cancel focused). All are opened from rows
(through `useCampaignTreeContent`) or panels (`useRecordActions`).

- **Move** (`MoveDialog`, FR-023 – FR-025): loads `…/destinations?purpose=move` (`staleTime: 0`); shows the
  rule, then a radio list (name; plan status chip or campaign dates); unavailable items have a disabled
  radio with their `unavailableReason` beneath; empty list → the "nothing to move to" text instead of the
  list. Choosing a destination shows its budget warning (`overAllocatedBy`) and, for campaigns, the
  consequences block (types not allowed with IDs, goals recalculated, groups hidden/shown). Confirm
  without a choice → "Choose where to move it." under the list. Confirm → `POST …/move` with the version
  the panel or row holds; success → `refreshPlanning()`, toast "<Level> '<name>' moved to '<destination>'"
  with **View**, highlight; 400/409 → the API message in an alert inside the dialog (409 via
  `conflictMessage`), list refetched.
- **Copy** (`CopyDialog`, FR-026, FR-027): loads `purpose=copy`; **Name of the copy** (`TextField`,
  Required) pre-filled with the current destination's `suggestedName` and following the chosen
  destination's suggestion until the user edits it; **Copy into** radio list (current first, "Current"
  tag); counts sentence from the record's `counts`; R13 sentence; budget warning; consequences (campaign).
  Confirm → `POST …/copy`; 400 → messages under **Name** / the list; success → `refreshPlanning()`, toast
  "<Level> '<new name>' created" with **View**, navigate to `/plans/:planId` with the new record
  highlighted.
- **Delete** (`DeleteRecordDialog`, FR-028, FR-029): `tone="danger"`; title, contents sentence from
  `counts`, "Plan totals will update.", the R14 sentence only when `approvedActivities > 0`, the recycle-bin
  sentence; red **Delete campaign** / **Delete initiative**. Confirm → `DELETE`; success → close the panel
  (navigate to `/plans/:planId`), `afterDelete(result)` from `useUndoableDelete({ invalidate: [['plans'],
  ['campaigns'], ['initiatives']] })` (10 s Undo, 001), `refreshPlanning()`; 404 → toast with the API
  message and refresh.

### Data and refresh (`campaignApi.ts`)

| Query key | Call |
|---|---|
| `['plans', planId, 'campaigns']` | `GET /api/plans/{planId}/campaigns` |
| `['campaigns', id]` | `GET /api/campaigns/{campaignId}` |
| `['initiatives', id]` | `GET /api/initiatives/{initiativeId}` |
| `['campaigns', id, 'destinations', purpose]`, `['initiatives', id, 'destinations', purpose]` | destinations, `staleTime: 0` |

`refreshPlanning(queryClient)` invalidates `['plans']` (prefix: 004's plan list and the campaigns list),
`['plan']` (004's plan page and summary cards, key `['plan', planId]`), `['campaigns']`, `['initiatives']`, `['recycle-bin']`, `['activity-log']` and is awaited
by every mutation's `onSettled`, so rows, panels, the plan's summary cards and the totals row show the new
values together, without a reload (FR-012, FR-015).

### States

| Where | Loading | Empty | Error | Validation | Conflict | Not found |
|---|---|---|---|---|---|---|
| Tree rows (`useCampaignTreeContent`) | 004's skeleton rows | 004's sentence + **Add campaign** | `ErrorState` + **Try again** | — | — | plan gone: 004's page |
| Create panels | `Skeleton` in the body while the plan/campaign loads | — | `ErrorState` in the body | `FormErrorSummary` + field messages | — | parent gone: the API's 404 message in an alert + **Back to the plan** |
| Details panels | `Skeleton` | sections show "—"; no initiatives / activities texts | `ErrorState` + **Try again** | — | — | "This campaign no longer exists — someone may have deleted it." + **Back to the plan** |
| Edit panels | `Skeleton` until the record is loaded once | — | save failure: `ErrorState` message as a toast, values kept | summary + fields | `conflictMessage`, values kept, version refreshed | as details |
| Move / Copy dialogs | `Skeleton lines={3}` | the nothing-to-move text | `ErrorState` inside the dialog | messages under name / list | 409 alert in the dialog | record gone: dialog shows the API message; Cancel closes |
| Delete dialog | busy on the button | — | toast with the API message | — | — | toast "This campaign no longer exists …" + refresh |

### New shared component and lib additions

- **`DateRangeField`** (`src/components/DateRangeField/`, exported from `index.ts`) — reason: the first
  date range in the module; design system §5 "Date range picker" is used by campaigns, initiatives (005),
  activities (006, with ongoing) and bulk changes (012). Props: `label`, `idPrefix`, `start`, `end`
  (`'YYYY-MM-DD' | ''`), `onChange({ start, end })`, `allowed?: { start: string; end: string } | null`,
  `allowedHint?`, `hint?`, `marking?`, `errors?: { start?: string; end?: string }`, `warning?`. A
  `<fieldset>` with a `<legend>` and two native `<input type="date">` labelled "Start date" and "End
  date", `min`/`max` from `allowed`, errors under each input (`aria-invalid`, `aria-describedby`), the
  allowed-range hint and warning under both. Test: labels, min/max, errors, onChange.
- **`formatDateRange(start, end)`** in `src/lib/format.ts`: same year "01 Feb – 30 Nov 2026", otherwise
  "01 Dec 2025 – 31 Jan 2026", "—" when either is missing (the server's `FormatRange`).

### User-visible messages (web)

The API messages are in *API validation and error messages*; the web app shows them as received. `<…>` is
filled in at run time; amounts use the shared money format in the person's display currency.

| Where | Text |
|---|---|
| Tree toolbar / plan row ⋯ / empty tree | Add campaign |
| Campaign row quick action (tooltip) | Add initiative to <campaign> |
| Row ⋯ campaign / initiative | Add initiative · View · Edit · Copy · Move to… · Delete / View · Edit · Copy · Move to… · Delete |
| Child counts | 1 initiative · <n> initiatives · 0 initiatives / 1 activity · <n> activities · 0 activities |
| Remaining beneath a campaign's budget | <amount> left · Over-allocated by <amount> |
| R4 warning (row tooltip, form, details) | This campaign runs outside <year>, the plan's year |
| Totals row | Total · <n> campaigns · <n> initiatives · <n> activities (singular: 1 campaign, 1 initiative, 1 activity) · Allocated to campaigns |
| Approval progress chip | No activities · In progress · <n> of <m> activities approved · All approved |
| Timing chip | Not started · Running · Finished · — (tip: Timing appears once dates are set) |
| Items below | 1 item below needs attention · <n> items below need attention |
| Create panel titles | New campaign · Inside plan '<plan>' / New initiative · Inside campaign '<campaign>' |
| Edit panel titles | Edit campaign · Edit initiative |
| Form sections | Basics · Budget · (attribute sections from Settings, default "Shell attributes") |
| Field labels | Campaign name · Initiative name · Campaign · Owner · Start date · End date · Description · Campaign budget · Initiative budget |
| Default hints | From the plan · The plan's year · From the campaign · The campaign's dates |
| Read-only campaign | Change it with Move |
| Allowed range | Must be inside the campaign's dates (<DD MMM YYYY> – <DD MMM YYYY>) |
| Remaining budget | Remaining plan budget · Remaining campaign budget · Over-allocated by <amount> · — · No budget to allocate from · ⓘ The plan's budget minus the budgets of all its campaigns, including this one. / The campaign's budget minus the budgets of all its initiatives, including this one. |
| Form footer | Save · Save and add another · Cancel |
| Toasts | <Campaign\|Initiative> '<name>' created — **View** (alt text: Open <name>) · <Campaign\|Initiative> '<name>' saved · <Campaign\|Initiative> '<name>' moved to '<destination>' — **View** |
| Details facts | ID · Owner · Dates · Created by <person> on <DD MMM YYYY>, last changed by <person> on <DD MMM YYYY> · (inactive) |
| Budget and cost section | Budget and cost · Budget · Allocated to initiatives · Remaining for initiatives · Remaining campaign budget · Planned cost · Calculated · Budget used · Left to plan · No planned cost below yet · No budget set |
| ⓘ tips | Planned cost (campaign): Sum of the planned costs of its initiatives. · (initiative): Sum of the planned costs of its activities. · Budget used: Planned cost ÷ budget. · Left to plan: Budget − planned cost. · Goals: Sum of the goals of the activities below. |
| Goals section | Goals · INQ · MQL · Revenue |
| Children | Initiatives (<n>) · No initiatives yet — an initiative groups similar work inside a campaign. · Activities: <n> (<k> approved) · Line items: <m> · No activities yet. |
| Details actions | Edit · More: Add initiative · Copy · Move to… · View history · Delete |
| Panel not found | This campaign no longer exists — someone may have deleted it. / This initiative no longer exists — someone may have deleted it. · **Back to the plan** |
| Move dialog | Move campaign '<name>' / Move initiative '<name>' · Choose the plan to move it to. Its initiatives, activities and line items move with it and keep their IDs, dates and statuses. / Choose the campaign to move it to. Its activities and line items move with it and keep their IDs, dates and statuses. · <rule from the API> · There's no other plan in <year> to move this campaign to. / There's no other campaign in this plan to move this initiative to. · Choose where to move it. · **Move campaign** / **Move initiative** · Cancel |
| Budget warning (move, copy) | This will over-allocate '<destination>' by <amount>. |
| Consequences (campaign) | '<plan>' uses another model or business unit: · These activities have a type the model '<model>' doesn't allow. They keep their type but need an allowed type before they can be submitted: <name> (<id>), … · Goals will be recalculated with the model '<model>'. · Hidden there, values kept: <groups> · Shown there: <groups> |
| Copy dialog | Copy campaign '<name>' / Copy initiative '<name>' · Name of the copy · Copy into · Current · It copies <contents>. / It copies the campaign only — there's nothing below it. (initiative: "the initiative only") · Copies get new IDs. Activities start as Created, without approvals, comments or history. · **Copy campaign** / **Copy initiative** · Cancel |
| Delete dialog | Delete campaign '<name>'? / Delete initiative '<name>'? · With it go <contents>. (omitted when nothing is below) · Plan totals will update. · <n> of these activities are approved. If they are restored later from the recycle bin, they return as Created and need approval again. (1: "1 of these activities is approved. If it is restored later …, it returns as Created and needs approval again.") · It moves to the recycle bin; you can restore it. · **Delete campaign** / **Delete initiative** · Cancel |
| After delete / Undo | 001's: <Campaign\|Initiative> '<name>' moved to the recycle bin — **Undo** · <Campaign\|Initiative> '<name>' restored |

Contents sentences ("2 initiatives, 7 activities and 15 line items") list only the non-zero counts, with
singular forms for 1, joined with ", " and " and ".

## Tests

Test names end with the requirement they prove (constitution V).

### Go unit tests

| File | Tests |
|---|---|
| `platform/date/date_test.go` | `TestParse_AndString`, `TestJSON_DateAndNull`, `TestAddYears_LeapDayBecomes28Feb`, `TestFormat_TwoDigitDayEnglishMonths`, `TestFormatRange_SameYearShortOtherwiseFull`, `TestToday_UsesUTC_DecisionB` |
| `rollup/rollup_test.go` | `TestSum_SkipsEmpty_NilWhenNothing_FR011`, `TestTotal_ZeroWhenNothing_FR022`, `TestRemaining_NegativeAllowed_FR010` (800,000 − (600,000 + 300,000) = −100,000.00, over by 100,000.00), `TestRemaining_NilWithoutBudget_FR010`, `TestBudgetUsedPercent_RoundsHalfUp_FR011` (150,000/120,000 → 125; 124.5 → 125; 99.4 → 99; nil for no budget, budget 0.00, no planned cost), `TestOverBudget_ExactComparison_FR011` (100.4 % → 100 and over), `TestLeftToPlan_FR011`, `TestApprovalProgress_States_FR013` (0/0, 0/5, 3/5, 5/5), `TestTiming_BeforeOnStartWithinOnEndAfter_FR013` (+ no dates), `TestAddGoals_NilWhenNothing_FR016`, `TestContainer_HandCalculated_SC004` (table of 10 cases to the cent) |
| `campaigns/service_test.go` | `TestCreateCampaign_OnlyNameRequired_FR005`, `TestCreateCampaign_EveryProblemAtOnce_FR005` (name taken + end before start + negative budget + 2,001-character description + one date only, one call), `TestCreateCampaign_NameUniqueInPlanIgnoringCaseAndSpaces_FR005` (allowed in another plan), `TestCreateCampaign_ReadableIDAndCreatedEntry_FR004_FR030`, `TestCreateCampaign_StoresWhatItReceives_DecisionD`, `TestCreateCampaign_NewOwnerMustBeActive_FR002`, `TestCreateCampaign_AttributeProblemsReportedWithOthers_FR009`, `TestCreateCampaign_OutsidePlanYearWarnsOnly_FR008`, `TestCreateCampaign_OverAllocationAllowed_FR010`, `TestCreateCampaign_PlanGone_NotFound`, `TestCreateCampaign_SameNameTwice_SecondRefused_EdgeCase`, `TestCreateInitiative_DatesInsideCampaign_FR006` (message with the full range), `TestCreateInitiative_CampaignWithoutDates_NotChecked_EdgeCase`, `TestCreateInitiative_NameUniqueInCampaign_FR005`, `TestUpdateCampaign_MissingVersion_FR018`, `TestUpdateCampaign_VersionConflict_FR018`, `TestUpdateCampaign_NoChangeNoVersionNoLog_FR030`, `TestUpdateCampaign_LogsEachChangedFieldWithTexts_FR030` (owner name, "400,000.00 USD", dates, one change per attribute), `TestUpdateCampaign_DatesLeavingInitiativesOutside_Refused_FR007` ("Digital & social (01 Feb – 30 Nov 2026)"), `TestUpdateCampaign_UnchangedDatesNeverRefused_DecisionF`, `TestUpdateCampaign_AllowedInPublishedPlan_FR018`, `TestUpdateCampaign_InactiveOwnerKept_EdgeCase`, `TestUpdateInitiative_DatesLeavingActivitiesOutside_Refused_FR007` (ongoing: start only; undated ignored), `TestUpdateInitiative_OutsideCampaign_Refused_FR006`, `TestNoRoleChecks_AnyoneChangesAnyRecord_FR031` |
| `campaigns/views_test.go` | `TestCampaignView_PlannedCostRollsUp_FR011` (100,000 + 50,000 under 120,000 → 150,000.00, 125 %, over budget), `TestViews_NothingBelow_DashAndNoActivities_FR011_FR013`, `TestCampaignView_RemainingForInitiatives_FR010`, `TestViews_ParentRemainingIncludesOwnBudget_FR010`, `TestViews_ParentWithoutBudget_Nil_EdgeCase`, `TestViews_ApprovalProgressCountsApproved_FR013` ("3 of 5", "0 of 5", all), `TestViews_TimingAgainstToday_FR013`, `TestCampaignView_CompletenessOwnFields_FR014` (no cluster and region → 2; dates count once), `TestCampaignView_RegionWithoutOptionsNotMissing_EdgeCase`, `TestViews_ItemsBelowNeedingAttention_FR014`, `TestViews_GoalSums_FR016`, `TestListPlanCampaigns_OrderByStartThenNameUndatedLast_FR020`, `TestListPlanCampaigns_Totals_FR022`, `TestViews_FollowChangesBelow_FR012_FR015` (fake activity cost changes → next read changes) |
| `campaigns/moves_test.go` | `TestInitiativeDestinations_Move_UnavailableWithReasonAndFix_FR023`, `TestInitiativeDestinations_Copy_CurrentFirstSuggestedNames_FR026`, `TestCampaignDestinations_Move_SameYearOnly_FR023`, `TestCampaignDestinations_Consequences_FR024` (types not allowed, model, groups), `TestDestinations_OverAllocationWarning_FR026`, `TestDestinations_NoneAvailable_EmptyList_EdgeCase`, `TestDestinations_BadPurpose_Refused`, `TestMoveInitiative_KeepsIDDatesValuesAndChildren_FR025`, `TestMoveInitiative_RefusedWithEveryReason_FR023`, `TestMoveCampaign_SameYearOnly_FR023`, `TestMoveCampaign_KeepsOutsideYearWarning_FR023`, `TestMove_LogsParentChange_FR030`, `TestMove_VersionConflict_FR025`, `TestCopyCampaign_CopiesEverythingBelow_FR027` (new IDs, same fields, `CopyAll` per initiative), `TestCopyCampaign_AllOrNothing_FR027` (fake `CopyAll` fails → no campaign, no initiative stored), `TestCopyCampaign_NameCheckedInDestination_FR026`, `TestCopyInitiative_DatesMustFit_FR026`, `TestCopy_OneCopiedEntryWithDetails_DecisionH`, `TestSuggestCopyName_CopyThenNumbers_R13`, `TestSuggestCopyName_ShortensLongNames_EdgeCase`, `TestCopyPlanContents_ShiftsDatesKeepsNames_For004`, `TestPlanRollups_CountsMoneyAndPublishProblems_For004`, `TestInitiativeFacts_For006` |
| `campaigns/kinds_test.go` | `TestDeleteCampaign_EverythingInOneBinEntry_FR028` (contents "with 2 initiatives"), `TestDeleteInitiative_FR028`, `TestUndoDelete_RestoresExactly_FR029` (same IDs, versions, roll-ups), `TestUndo_NameTaken_Refused_FR029`, `TestRestore_NameTaken_GetsRestoredSuffix_FR029`, `TestDescribe_ReadableValues_FR030`, `TestDelete_Gone_NotFound` |
| `app/campaign_adapters_test.go` | `TestCampaignLevels_AddFacts_For004` (004's `plans.Facts` for a plan count its campaigns, initiatives and campaign budgets, and list incomplete campaigns and initiatives with `BlocksPublish`), FR-014 |
| `seed/seed_test.go` | `TestSeed_CampaignsAndInitiatives` (4 campaigns, 8 initiatives, IDs, counters C = 1004 and I = 1008, Retail growth "2 fields needed", Sponsorships over-allocation −50,000.00) |

Service tests use `fakes_test.go` (fixed clock 2026-03-12, in-memory fakes of the four interfaces, a real
JSON store in `t.TempDir()`, a real `history.Service` with a minimal fake `plan` kind registered first and,
for delete tests, a fake `activity` kind under `initiative`).

### Contract tests (`campaigns/handler_test.go`, package `campaigns_test`, `apitest` on the seed)

One test per operation, each request and answer validated against the contract:
`TestListPlanCampaigns_Contract` (P-1001: 2 campaigns, 4 initiatives, totals),
`TestListPlanCampaigns_UnknownPlan404`, `TestCreateCampaign_Contract201`, `TestCreateCampaign_AllProblems400`,
`TestGetCampaign_Contract`, `TestGetCampaign_404`, `TestUpdateCampaign_Contract`,
`TestUpdateCampaign_MissingVersion400`, `TestUpdateCampaign_Conflict409`, `TestDeleteCampaign_ContractAndUndo`
(DeleteResult, then `POST /api/recycle-bin/{id}/undo` brings C-1001 back), `TestCampaignDestinations_Contract`
(move, copy, `purpose=other` 400 with `SkipRequestValidation`), `TestMoveCampaign_Contract`,
`TestMoveCampaign_Refused400`, `TestCopyCampaign_Contract201`, `TestCreateInitiative_Contract201`,
`TestCreateInitiative_OutsideCampaign400`, `TestGetInitiative_Contract`, `TestGetInitiative_404`,
`TestUpdateInitiative_Contract`, `TestUpdateInitiative_Conflict409`, `TestDeleteInitiative_Contract`,
`TestInitiativeDestinations_Contract`, `TestMoveInitiative_Contract` (I-1001 → C-1002),
`TestCopyInitiative_Contract201`, `TestAnyoneCanChangeAnyCampaign_FR031` (Daniel edits a campaign Priya
created). `app_test.go`'s `TestRoutesMatchContract` and
`TestEveryProtectedOperationRequiresSignIn_FR005` cover the 15 routes.

### Web component tests (Vitest, `mockApi` with contract-shaped answers)

| File | Proves |
|---|---|
| `campaignRows.test.ts` | FR-019 – FR-022: rows and parent ids, order kept, child-count texts, remaining text and red flag, R4 flag, ⋯ items per level, extra initiative actions, totals text |
| `StatusChips.test.tsx` | FR-013, FR-015: words and tones; "0 of 5"; Timing "—" with its tip |
| `BudgetCost.test.tsx` | FR-010, FR-011: live remaining incl. typed amount and negative "Over-allocated by", "—" without parent budget; planned cost "Calculated" and tip; "—"; budget-used bar red when over budget |
| `NewCampaignPanel.test.tsx` | US1 1–5, FR-001, FR-003 (labelled defaults), FR-004 (toast **View**, highlight state), FR-005 (all problems from a 400 at once, summary focused), FR-008 (warning, save allowed), FR-009 (with 002's `AttributeFields`: region disabled, single region filled with its message), **Save and add another**, Cancel guard |
| `NewInitiativePanel.test.tsx` | US2 1–3, FR-006 (allowed range in the date fields and hint; the API's R3 message under the dates), campaign read-only "Change it with Move" |
| `CampaignPanel.test.tsx` | US3 1–4, FR-016 – FR-018: details content and ⋯ More items, `/edit` form, save refreshes, FR-007 message under the dates, cluster removal note and Discard guard, 409 conflict keeps values, not found, loading, error + Try again, View history link, wrong-plan address replaced |
| `InitiativePanel.test.tsx` | details, edit, R3/FR-007 messages, More items + extension slot |
| `useCampaignTreeContent.test.tsx` | FR-001 (toolbar, plan row, empty tree), FR-020 start expanded, row ⋯ opens dialogs, FR-012 refresh after a change |
| `MoveDialog.test.tsx` | US6 1–3, FR-023 – FR-025: rule, unavailable reasons, empty text, consequences, budget warning, "Choose where to move it.", success toast, 400/409 shown |
| `CopyDialog.test.tsx` | US6 4–5, FR-026, FR-027: suggested names follow destination until edited, counts, R13 sentence, warning, name error, success highlight |
| `DeleteRecordDialog.test.tsx` | US5 1–2, FR-028, FR-029: texts with counts, approved sentence only when approved > 0, Cancel focused, danger label, Undo toast (fake timers) |
| `components/DateRangeField/DateRangeField.test.tsx`, `lib/format.test.ts` | the shared component; `formatDateRange` |

### End-to-end (`frontend/e2e/campaigns.spec.ts`, Playwright on the seeded API, as `smoke.spec.ts`)

Each test signs in with `signInAs`, uses names unique to the test and asserts only its own records or
before/after differences:

1. **US1 Add a campaign** (Mei Lin, FY2026 Mobility China): **Add campaign** → defaults labelled → name,
   cluster Retail – China → region filled with its message → budget 300000 → remaining "Over-allocated by"
   in red → **Save** → toast, highlighted row, totals row "Allocated to campaigns" up by 300,000 (SC-001: ≤ 5 steps).
2. **US2 Add an initiative** (Priya, Helix launch): row **Add initiative** → name → **Save** (3 steps); a
   start date before the campaign's start shows the R3 message.
3. **US3/US4 View and edit**: open `/plans/P-1001/campaigns/C-1001` directly → details facts and chips →
   **Edit** budget and owner → row, panel and plan summary card update; narrowing dates past "Digital &
   social" shows the FR-007 problem; rows show Approval progress, Timing, completeness and budget used.
4. **US5 Delete and Undo**: create a campaign with an initiative, **Delete** → dialog counts →
   **Delete campaign** → **Undo** within 10 s → same IDs back, totals restored.
5. **US6 Move and copy**: create two campaigns and an initiative; move it (unavailable destination shows
   its reason), check both rows; copy a campaign → "(copy)" row highlighted with new IDs.
6. **Accessibility**: axe (WCAG 2.2 AA) on the create panel with errors, a details panel, the move dialog
   and the delete dialog; keyboard: reach a row's ⋯ menu and **Add initiative** without a mouse.

## Dependencies

### Expects from other features (by name; adapters in `internal/app` absorb Go naming differences)

| From | What | Used for |
|---|---|---|
| 001 | `history.Service` (`Register`, `Log`, `Diff`, `MoveToBin`, `Kind`, `RestoreContext`, `DeleteResult`), `record.Meta`, `ids.Generator`, `errs`, `web`, `actor`, `apitest`; web: `SidePanel`, `ConfirmDialog`, `ActionsMenu`, `StatusChip`, `CompletenessChip`, `InfoTip`, `Tooltip`, `TextField`, `FormErrorSummary`, `ErrorState`, `Skeleton`, `EmptyState`, `Button`, `Avatar`, `useToast`, `useUndoableDelete`, `useUnsavedChanges`, `usePageTitle`, `conflictMessage`, `formatMoney`, `formatDate` | everything |
| 002 | Go `backend/internal/platform/money` (`money.Money` USD amount: zero, parse "12500.50", add, subtract, compare, sign, cents); `settings.AttributeValues` (Go) and schema **`Money`**, **`AttributeValues`**; attribute rules for `deps.Settings` (check values with dependent options and business-unit conditions, missing needed-to-publish attributes with D-034, per-attribute log changes and describe, group changes between business units, model's allowed activity types and type names); web: `AttributeFields` (`level`, `values`, `onChange`, `planValues`, `errors`, `fieldId`) with R8 behaviour, `AttributeValuesView`, `defaultAttributeValues(level)`, `MoneyField`, `@/lib/money` (exact cents: parse, add, subtract, compare) | budgets, attributes, completeness, move consequences |
| 003 | schema **`PersonRef`** `{id, name, initials, active}`; person lookup for `deps.People`; web `PersonPicker` (single), display-money formatting in the person's display currency (`useDisplayMoney()` or equivalent) | owners, amounts |
| 004 | plan facts for `deps.Plans` (id, name, year, status, owner, budget, model id and name, plan attribute values; plans of a year); history kind `plan` registered before `campaign`; web: the `/plans/:planId` route rendering `<Outlet />`, `PlanTree` accepting `rows: PlanTreeRow[]` (with `parentId`, `level`, name/href/id, cells, `actions`, `startExpanded`), `toolbarActions`, `planRowActions`, `emptyAction`, `totalsRow`, `highlightId` from `location.state.highlight`; shared `LevelTag`, `ProgressBar`, `TextAreaField`; plan queries keyed under `['plans', …]`; 004's own interface for plan roll-ups satisfied by `PlanRollups` and `CopyPlanContents` — 004's planned names for all of these are in *Alignment with feature 004 as planned* below | rows, plan page, summary cards |
| 006 | an implementation of `campaigns.Activities` (`Summaries`, `CopyAll`); the `activity` kind with parent `initiative`; `extensions.ts` filled with **Add activity** and activity rows | roll-ups, copies, delete contents |

### Alignment with feature 004 as planned

Feature 004's plan (written at the same time) names its extension points differently from the rows above.
Where they differ, **004's names win**; the table says how 005 plugs in. Behaviour and texts of this plan do
not change.

| 005 above | 004's extension point | How 005 uses it |
|---|---|---|
| "004's own interface for plan roll-ups" (`PlanRollups`, `CopyPlanContents`) | `plans.Level` registry: `planService.RegisterLevel(plans.Level{Name, AddFacts, Copy})`, called top-down after the `plan` kind; `plans.Facts`, `plans.AttentionItem{Record, Missing, BlocksPublish}`, `plans.CopyJob{…, Years, NewIDs, Counts}` | `internal/app/campaign_adapters.go` registers two levels. `campaign`: `AddFacts` fills `Campaigns`, `CampaignBudgets` (= `AllocatedToCampaigns`) and, from `PublishProblems`, one `AttentionItem` per incomplete campaign with `BlocksPublish: true`, all from one `svc.PlanRollups(planIDs)` call; `Copy` runs `CopyPlanContents(job.Actor, job.SourcePlanID, job.TargetPlanID, job.Years)` and adds its old → new IDs to `job.NewIDs` and its counts to `job.Counts["campaign"]` and `["initiative"]`; 006's and 007's levels then copy activities and line items. `initiative`: `AddFacts` adds `Initiatives` and the initiatives' attention items; `Copy` is a no-op (the campaign level copied them). Activity and line-item counts, `PlannedCost`, `ActivityGoals` and their attention items are added by 006's and 007's own levels, so 005 adds only its own records to `Facts` (no double counting). `campaigns` keeps not importing `plans` |
| `PlanSource.Plan` / `PlansOfYear` | `plans.Service.Lookup(ctx, a, id)` (joins the caller's transaction) and the plans repository | adapter in `campaign_adapters.go` |
| `date.AddYears` for plan copies | `plans.ShiftDate(string, years)` | both follow the same rule (29 Feb → 28 Feb); 005 uses its own `platform/date` and tests the same cases |
| `PlanTree` props `rows`, `planRowActions`, `emptyAction`, `totalsRow`, `highlightId` | `TreeTable` `TreeNode{id, level, name, href?, onOpen?, cells, children, actions?, highlighted?}`; `tree/usePlanTreeNodes.ts` (004 returns `[planNode]` with `children: []`); `PlanTree` props `toolbarActions?` and `footer?` | `campaignRows` returns `TreeNode[]` (campaign nodes with their initiative nodes as `children`); 005 edits `usePlanTreeNodes.ts` to set `planNode.children` from `useCampaignTreeContent(planId).nodes`; **Add campaign** goes in `toolbarActions`; the totals row is `footer`. The plan row's ⋯ **Add campaign**, the empty-tree **Add campaign** and highlighting are added to `PlanTree` as optional props (`planActions?`, `emptyAction?`, `highlightId?`) only if 004 did not build them — a shared-file change reported in the task |
| completeness field links | `RECORD_LINKS` in `features/plans/recordLinks.ts` (`label`, `record(ref, planId)`, `field(ref, field, planId)`) | 005 adds `campaign` → `/plans/<planId>/campaigns/<id>` (field: `…/edit?focus=<field>`) and `initiative` → `/plans/<planId>/initiatives/<id>` (field: `…/edit?focus=<field>`), so 004's publish dialog and plan chip link to them |
| `CompletenessChip` with a count of items below | `CompletenessChip` `itemsBelow?: {id, label, href}[]` (004) | rows and panels pass the count through a small `itemsBelowCount` prop added next to `itemsBelow` (or 004's list, if a later decision chooses lists for every level) |
| `Completeness`, `ApprovalProgress`, `MissingField` schemas | 004 owns `MissingField {field, label}`, `ApprovalProgress` (string enum) and `Completeness {missing, itemsBelow: AttentionItem[]}` | the fragment reuses `MissingField` and `ApprovalProgress` and names its own shapes `RecordCompleteness` and `ApprovalSummary`, so the merge has no duplicate names. If 002's `MissingField` (with `reason`) is the merged one, 005 sends `reason: ""` |
| query keys under `['plans', …]` | `['plans', filter]`, `['plan', planId]` | `refreshPlanning` invalidates both prefixes |
| attribute names (`AttributeFields`, `AttributeValuesView`, `CheckAttributes(values, stored, plan)`) | 002 plans `ShellAttributeFields`, `ShellAttributeValues`, `(*settings.Service).CheckAttributes(ctx, level, AttributeInput{Values, Previous, BusinessUnit})` returning `{Problems, Cleared, Missing, Values}`; Go `money.Amount` (cents) and `money.Money{Amount, Currency}` | the `Settings` adapter maps 005's four methods onto 002's service; the web app imports 002's names |

### Provides to later features

| To | What |
|---|---|
| 004 | `(*campaigns.Service).PlanRollups`, `CopyPlanContents`; package `rollup`; web `useCampaignTreeContent(planId)` for the plan page |
| 006 | `InitiativeFacts`, `InitiativesOfPlan`, the `initiative` kind (parent of `activity`), the `campaigns.Activities` interface to implement, `platform/date`, `DateRangeField`, `formatDateRange`, `features/campaigns/extensions.ts` (initiative actions, activity rows), `InitiativePanel` details |
| 007 | `platform/date`, `rollup` (line-item totals feed activity summaries) |
| 009, 010 | `rollup` functions (container money, approval progress, timing, goals), `PlanRollups`, schemas `BudgetAndCost`, `ApprovalSummary`, `RecordCompleteness`, `GoalSums`, `StatusChips` (approval progress and timing chips) |
| 011 | kinds `campaign` and `initiative` with `PrepareRestore` (" (restored)") and `Describe`; log entries with paths |
| 012 | service methods joining the caller's transaction (create, move, copy), destination rules |

## Shared files

| File | Change |
|---|---|
| `backend/api/openapi.yaml` | + the fragment's tag, 10 paths (15 operations), 3 parameters and 24 schemas (`MissingField` and `ApprovalProgress` are 004's, reused) |
| `backend/internal/app/app.go` | + campaigns service, kinds, `plans.Level` registrations, routes; `campaign_adapters.go` and `campaign_adapters_test.go` (new) |
| `backend/seed/seed.go`, `seed_test.go` | + `addCampaigns`, summary line |
| `frontend/src/app/router.tsx` | + 6 child routes under `/plans/:planId` |
| `frontend/src/features/plans/PlanPage.tsx`, `tree/usePlanTreeNodes.ts`, `recordLinks.ts` (004) | `usePlanTreeNodes` fills the plan node's children from `useCampaignTreeContent(planId)`; `PlanPage` passes its `toolbarActions` and `footer` to `PlanTree`; `RECORD_LINKS` gains `campaign` and `initiative`; `PlanTree` gains optional `planActions`, `emptyAction`, `highlightId` only if 004 lacks them |
| `frontend/src/components/index.ts`, `DateRangeField/` | new shared component |
| `frontend/src/lib/format.ts`, `format.test.ts` | + `formatDateRange` |
| `frontend/src/api/schema.d.ts` | regenerated (`npm run api:types`) |
| Navigation | no change (Plans leaves "Coming soon" with feature 004) |

## Requirement traceability

| FR | Implemented in | Tested by |
|---|---|---|
| FR-001 | `useCampaignTreeContent`, `campaignRows`, `NewCampaignPanel`, `NewInitiativePanel` | `useCampaignTreeContent.test`, panel tests, e2e 1–2 |
| FR-002 | forms + `AttributeFields`; `CheckAttributes`; owner rule | `TestCreateCampaign_NewOwnerMustBeActive_FR002`, panel tests |
| FR-003 | create panels (defaults and labels); Decision D | `NewCampaignPanel.test`, `TestCreateCampaign_StoresWhatItReceives_DecisionD` |
| FR-004 | `CreateCampaign`/`CreateInitiative` (`ids`), toast + highlight | `TestCreateCampaign_ReadableIDAndCreatedEntry_FR004_FR030`, panel tests, e2e 1 |
| FR-005 | `validate.go` | `TestCreateCampaign_EveryProblemAtOnce_FR005` and name tests, `_AllProblems400` |
| FR-006 | `validate.go` R3, `DateRangeField allowed` | `TestCreateInitiative_DatesInsideCampaign_FR006`, `NewInitiativePanel.test`, e2e 2 |
| FR-007 | `UpdateCampaign`, `UpdateInitiative` | `…_DatesLeaving…_Refused_FR007`, `CampaignPanel.test`, e2e 3 |
| FR-008 | `outsidePlanYear`, row/form/details note | `TestCreateCampaign_OutsidePlanYearWarnsOnly_FR008`, `campaignRows.test` |
| FR-009 | `AttributeFields` (002) + `CheckAttributes` | `TestCreateCampaign_AttributeProblemsReportedWithOthers_FR009`, `TestCampaignView_RegionWithoutOptionsNotMissing_EdgeCase`, `NewCampaignPanel.test`, e2e 1 |
| FR-010 | `rollup.Remaining`, `parentRemainingBudget`, `RemainingBudgetLine` | rollup and views tests, `BudgetCost.test`, e2e 1 |
| FR-011 | `rollup.Sum`, `BudgetUsedPercent`, `OverBudget`, `BudgetCostSection` | rollup tests, `TestCampaignView_PlannedCostRollsUp_FR011`, `BudgetCost.test` |
| FR-012 | derived on read + `refreshPlanning` | `TestViews_FollowChangesBelow_FR012_FR015`, `useCampaignTreeContent.test`, e2e 3 |
| FR-013 | `rollup.ApprovalProgress`, `Timing`, `StatusChips` | rollup tests, views tests, `StatusChips.test` |
| FR-014 | `completeness` in `views.go`, `MissingAttributes` | `TestCampaignView_CompletenessOwnFields_FR014`, `TestViews_ItemsBelowNeedingAttention_FR014`, `TestPlanRollups_CountsMoneyAndPublishProblems_For004` |
| FR-015 | chips on rows and panels, refresh | `StatusChips.test`, `TestViews_FollowChangesBelow_FR012_FR015` |
| FR-016 | `CampaignPanel`, `InitiativePanel` details, `GoalsSection` | panel tests, `TestViews_GoalSums_FR016` |
| FR-017 | routes (Decision I) | `CampaignPanel.test` (address, wrong plan), e2e 3 |
| FR-018 | edit mode, `useUnsavedChanges`, version check | `TestUpdateCampaign_MissingVersion_FR018`, `TestUpdateCampaign_VersionConflict_FR018`, `TestUpdateCampaign_AllowedInPublishedPlan_FR018`, `CampaignPanel.test` |
| FR-019 | `campaignRows` | `campaignRows.test`, e2e 3 |
| FR-020 | API order; `startExpanded` | `TestListPlanCampaigns_OrderByStartThenNameUndatedLast_FR020`, `campaignRows.test` |
| FR-021 | row actions, quick action | `campaignRows.test`, e2e 6 (keyboard) |
| FR-022 | totals | `TestListPlanCampaigns_Totals_FR022`, `campaignRows.test`, e2e 1 |
| FR-023 | destinations, move rules | `moves_test.go` FR-023 tests, `MoveDialog.test`, e2e 5 |
| FR-024 | consequences | `TestCampaignDestinations_Consequences_FR024`, `MoveDialog.test` |
| FR-025 | `MoveCampaign`, `MoveInitiative` | `TestMoveInitiative_KeepsIDDatesValuesAndChildren_FR025`, `TestMove_VersionConflict_FR025`, e2e 5 |
| FR-026 | copy destinations, `suggestCopyName`, `CopyDialog` | moves tests FR-026, `CopyDialog.test` |
| FR-027 | `CopyCampaign`, `CopyInitiative`, `Activities.CopyAll` | `TestCopyCampaign_CopiesEverythingBelow_FR027`, `_AllOrNothing_FR027`, e2e 5 |
| FR-028 | kinds + `MoveToBin`, `DeleteRecordDialog` | `kinds_test.go`, `DeleteRecordDialog.test`, e2e 4 |
| FR-029 | 001 Undo, `useUndoableDelete`, `PrepareRestore` | `TestUndoDelete_RestoresExactly_FR029`, `TestUndo_NameTaken_Refused_FR029`, e2e 4 |
| FR-030 | `history.Log` calls, `Describe`, View history link | `…_FR030` tests, `TestDescribe_ReadableValues_FR030`, `CampaignPanel.test` |
| FR-031 | no role checks | `TestNoRoleChecks_AnyoneChangesAnyRecord_FR031`, `TestAnyoneCanChangeAnyCampaign_FR031` |

## Complexity Tracking

| Choice | Why needed | Simpler alternative rejected because |
|---|---|---|
| Consumer-side interfaces in `deps.go` with adapters in `internal/app` | Campaigns needs activity facts while activities needs initiative facts — importing each other is a cycle; the interfaces also let 005 be built and tested before 006 and independently of the exact Go names chosen by 002–004 | Importing `activities` from `campaigns` creates the cycle; a shared "everything" package would mix business areas |
| `rollup` as its own pure package | 004, 009 and 010 must compute the same numbers; they cannot import `campaigns` without coupling to its storage | Copying the formulas into each feature lets them drift (SC-004, 010 SC-002) |
| Destinations endpoints | The move and copy dialogs must say why a destination is unavailable (FR-023, SC-003) using rules only the server knows (names, dates, models, attribute conditions) | Computing them in the web app duplicates server rules (constitution IV) |
| `extensions.ts` slot file | Feature 006 adds **Add activity** and activity rows to this feature's rows and panel without rewriting them | A plug-in registry is more machinery than one small file with two functions |

## Consolidation (2026-10-19, binding)

The Plan-module consolidation (the cross-feature analyze step) settled the points below. They and the build guide,
[docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md), win where this plan, its data model or its tasks
say otherwise (D-200).

- **Contract (D-192).** The fragment is merged into `backend/api/openapi.yaml`; the merge task (T001) is done. The
  feature's operations are listed in `notYetServed` in `backend/internal/app/app_test.go`; the backend removes each one in
  the change that serves it. Agents do not change the contract; a problem with it is reported.
- **Decisions.** This plan's proposed decisions are numbered D-090–D-101 in `docs/sdd/decision-log.md` (labels kept in the
  table above); the consolidation decisions are D-189–D-207.

- **Contract (D-193).** `ApprovalSummary` and `RecordCompleteness` stay 005's (008's page summary is now `ApprovalCounts`);
  `MissingField` is 002's (`reason` optional).
- **Completeness (D-199).** Campaigns and initiatives answer counts; the chip uses 004's `CompletenessChip` `itemsBelow` as a count.
- **Dates (D-101, D-198).** `platform/date` follows this plan's design and is built with the shared foundation; 004 uses it too.
- **Copies (D-195).** Single copies call `campaigns.Activities.CopyAll(ctx, a, fromInitiativeID, toInitiativeID, yearShift)`
  (implemented by 006); plan copies go through the `plans.Level` chain as in *Alignment with feature 004*.
- **Shared web parts (D-197).** 005 owns `DateRangeField` and `formatDateRange`; it uses 004's `TreeTable`, `FormDialog`,
  `LevelTag`, `ProgressBar`, 002's `MoneyField`, `ShellAttributeFields`, 003's `PersonPicker`, and adds its entries to 004's
  `recordHref` (D-194).
- **Typed amounts (D-190).** Budgets with more than two decimals are rounded by the form, never by the API.
