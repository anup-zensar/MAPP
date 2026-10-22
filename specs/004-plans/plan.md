# Implementation Plan: Plans

**Branch**: `004-plans` (built on `plan-module`) | **Date**: 2026-10-19 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/004-plans/spec.md`

**Design artifacts**: [data-model.md](data-model.md) · [contracts/openapi.yaml](contracts/openapi.yaml) (OpenAPI
3.1 fragment, merged into `backend/api/openapi.yaml`) · [tasks.md](tasks.md). Conventions not repeated here
come from feature 001: [plan](../001-app-shell-and-sign-in/plan.md),
[research](../001-app-shell-and-sign-in/research.md) (R3 store, R5 versions, R8 history, R9 errors, R10 API),
[data model](../001-app-shell-and-sign-in/data-model.md).

## Summary

Feature 004 adds the **plan**, the top of the planning hierarchy, as MAPP's first full standard CRUD
record (design system §6):

- **Go API** — a new business package `internal/plans` (handler → service → repository) with the stored
  `Plan`, eleven operations (list with filters and summary, defaults for a new plan, create, view, edit,
  consequences preview, publish, unpublish, copy options, copy, delete), the plan's calculated **figures**
  (remaining budget, planned cost, budget used, approval progress), its **goal waterfall** (forwards and
  backwards from the model), its **completeness** and the **publish check**. Plans read Settings (002) and
  People and teams (003) through two small interfaces that `internal/app` adapts, and lower levels (005–007)
  plug in through a `plans.Level` registry, like `history.Kind`, so no package imports a package above it.
  Every change is logged with `history.Log`; delete goes through `history.MoveToBin` with a registered
  `plan` kind (Undo exact; a later Restore returns the plan as Draft).
- **React web app** — the **Plans** page (`/plans`: summary cards, status pills, year filter, My teams / All
  plans, search, chips, count, cards or table), the full-page **plan form** (`/plans/new`,
  `/plans/:planId/edit`) with sections Basics · Budget · Shell attributes · Goals and live goal
  calculation, and the **plan page** (`/plans/:planId`: header, five summary cards, the **plan tree**,
  Details, Goals) whose layout has fixed extension points for 005–007, 011 and 013. Dialogs: publish
  blocked, unpublish with reason, copy, delete with Undo, and model / business-unit consequences.

Approach: reuse 001's history core, components and helpers; one new pure calculation file per concern
(figures, goals, completeness); five new shared components that later features need anyway (TreeTable,
ProgressBar, LevelTag, FormDialog, FilterChips) and two small extensions (SummaryCard `detail`,
CompletenessChip `itemsBelow`).

## Technical Context

**Language/Version**: Go 1.27 (API); TypeScript 5.9 strict + React 19 (web app) — fixed by 001

**Primary Dependencies**: no new dependency. API: standard library (`math/big` for exact goal and percentage
arithmetic) + `platform/money` (feature 002). Web app: the libraries of 001 (React Router 8, TanStack Query
5, Radix, lucide-react, openapi-fetch)

**Storage**: JSON file store, new table `plans` (`backend/data/plans.json`); readable IDs `P-1001…` from the
`ids` counter `P`; activity log and recycle bin from 001

**Testing**: Go `testing` (service tests with fakes of the two interfaces and of `plans.Level`; handler
tests with `apitest` against the merged contract); Vitest + Testing Library (`mockApi`); Playwright
`frontend/e2e/plans.spec.ts` with axe

**Target Platform**: developer laptops (Windows 11 first); current Chrome and Edge

**Project Type**: web application — React SPA + Go REST API

**Performance Goals**: SC-009 — with 200 plans the list answers and filters in under 1 s: `List` asks each
`Level` **once** for all plan IDs (`AddFacts(planIDs)`), never once per plan; a plan with 500 activities
opens in under 2 s (one `AddFacts` call per plan view)

**Constraints**: 127.0.0.1 only; no personal data in logs; exact decimals (never float) for money, rates,
goals and percentages; deterministic tests (fixed clock 2026-03-12 09:00 UTC → current year 2026)

**Scale/Scope**: 3 seed plans; demo-sized data (hundreds of plans). 4 web routes, 11 API operations, 1 table

## Constitution Check

*GATE: checked before design and again after it.*

| Principle | How this plan complies | Result |
|---|---|---|
| I. Spec first | Every element traces to an FR, acceptance scenario, edge case or SC (traceability table). No `[NEEDS CLARIFICATION]` in the spec; the defaults of D-035 and Q-001/Q-004/Q-005/Q-006 are applied (Decisions) | Pass |
| II. Shell business, plain language | Shell's words only (Plan, Campaign, Initiative, Activity, Line item, AOP, BCPIR, SMP, INQ, MQL, Goal stage, Default line-item spread); Hive9's "Waterfall Goal", "Default Line-Item Allocation", "Set plan as Strategic/AOP" never shown; models, attributes, teams, finance budgets come from Settings data, never hard-coded | Pass |
| III. Simple, friendly experience | Standard CRUD pattern (list → plan page → Edit → delete to the bin with Undo); status and completeness chips on every list card, table row, tree row and the header; all problems at once; every disabled control and lock says why (owner, year, next-year copy); unsaved changes protected; consequences confirmed before saving; WCAG 2.2 AA with axe in the e2e test | Pass |
| IV. Simple, layered architecture | One package `plans` (handler → service → repository); all rules and calculations in the service (figures, goals, completeness, publish check, copy names, R1); the web app repeats only the live goal preview and the owner check for instant feedback; contract first; storage only through `plans.Repository` and `store.Transactor` | Pass |
| V. Tested business rules | Service test per rule named with its FR; handler/contract test per operation; component test per page and dialog; `plans.spec.ts` covers every P1 story (US1–US5) and US6; deterministic fixed clock | Pass |
| VI. Safe by default | Actor passed to every service method (reads included) and recorded on every change; no role checks (D-004, FR-032); input validated by the API; amounts exact decimals in USD with display conversion on the server; delete → recycle bin, restore parents first | Pass |
| VII. Keep it small | No new library; new shared components only where 005–010 need them too (each with its reason); the `Level` registry is the smallest way to avoid import cycles (Complexity Tracking) | Pass |

**Post-design re-check**: Pass. The choices that add code beyond the visible screens are in Complexity Tracking.

## Decisions

### Open questions and earlier decisions applied (defaults, not re-decided)

| Topic | Applied |
|---|---|
| D-035 | **All plans** is the default of the My teams switch; a plan with no campaigns can be published; a copy keeps the source's owner and teams |
| Q-001 | BCPIR plan is a yes/no field on the plan, nothing more |
| Q-004 / D-011 | The plan year is a calendar year; the plan row shows "1 Jan – 31 Dec <year>" |
| Q-005 | Markings as in field catalogue §1: Required = name, year, model; Needed to publish = teams, owner, goal stage, business unit (plus any plan-level attribute Settings marks Needed to publish) |
| Q-006 | My teams only narrows the list; everyone opens every plan |
| Q-007 | Not affected: publishing a plan never depends on activity approval and vice versa |
| D-009, D-015, D-026, D-030, D-031, D-037 | Two-level markings; negative remaining budget never blocks; plans show approval progress; out-of-date saves refused; Undo exact, later restore resets status; Undo window 10 s on screen / 2 min server |

### Proposed decisions (numbered at consolidation, 2026-10-19)

| ID | Decision | Reason |
|---|---|---|
| D-072 | **Lower levels plug into plans through a registry** `plans.Level` (005 campaigns and initiatives, 006 activities, 007 line items), registered in `internal/app` top-down: each adds its figures for many plans at once (`AddFacts`), copies its records for a plan copy (`Copy`), and may list activities whose type a model disallows. | `campaigns` imports `plans` (parent checks), so `plans` cannot import `campaigns`; same pattern as `history.Kind`; batching keeps the list fast (SC-009) |
| D-073 | **Plans read Settings (002) and People/teams (003) through two consumer interfaces** `plans.Settings` and `plans.Directory`, implemented by adapters in `internal/app`. | 002/003 count "Used by n plans" and will need to read plans, so a direct import would risk a cycle; fakes make every plan rule testable without 002/003 |
| D-074 | **Derived values are never stored**: MQL, deals, revenue, planned cost, remaining budget, budget used, approval progress and completeness are calculated on every read from the current model, settings and records below. | A rate change (002 FR-011), a model change (FR-019) or a change below updates every number with no recalculation job |
| D-075 | **The goal waterfall is one exact function shared by 002, 004 and 006** (owned by 002), forwards and backwards, rounding half up at each stage to a whole number, revenue to the cent (algorithm in *Calculations*). A stage that would divide by a 0 % rate or a 0 deal size is "—" with its reason. | One rule, three users (002 worked example, plan goal, activity goals); exact integers avoid float drift |
| D-076 | **Publish refusal is an error with a new code** `not_ready` (409) and a new optional `missing` list on the shared `Error` (groups of `{record, fields}`); 006/008 reuse it for "Submit needs n fields". | The publish answer must list every missing field grouped by record (FR-024) while keeping 001's single error format |
| D-077 | **Consequence previews are a read-only operation**, `POST /api/plans/{planId}/change-effects`; the update itself never needs a "confirmed" flag. | Confirmations are a screen behaviour; the server stays the source of truth for what a change affects (activities of disallowed types, attribute groups hidden/shown) without a two-step save |
| D-078 | **`PATCH /api/plans/{planId}` sends the whole form** (every editable field, `null` to clear), not only changed fields (deviation from 001 R10). The server compares, saves and logs only what changed. | Several plan fields can be cleared; telling "not sent" from `null` in Go needs extra code; the form always has every value anyway |
| D-079 | **Amounts in plan views are converted to the viewer's display currency by the API** (`figures`, `goals`), rounded half up to the cent, each labelled with its currency; editable stored values (plan budget, revenue goal) stay in USD in the same answer. | Constitution IV (conversion on the server) and FR-015, while the edit form keeps USD inputs (003 FR-023) |
| D-080 | **Plans list filtering happens in the API** (`scope`, `status`, `year`, `q`); the summary cards cover scope + year (ignoring status and search); the count reads "<matching> of <all plans in the team scope>". | FR-001/FR-003 and US2 scenario 2 ("1 of 3"); one tested implementation; money totals stay on the server |
| D-081 | **List filters and layout are remembered per person in the browser** (`localStorage` key `mapp.plans.list.<personId>`; status, year, scope, layout — not the search text); a filter in the address (`/plans?status=draft&year=2026&scope=myTeams&q=…`) wins over the remembered one. | FR-003 "remembered per user" without a server change to 001's preferences; works with Home's links (009 FR-007). Other browsers start from the defaults |
| D-048 | **Shell attribute values are stored on their record** as a map `attributes: {attributeId: value}` (all levels), validated and interpreted by Settings (002); hidden values stay in the map (R9). | One row per record; PostgreSQL maps it to a JSONB column or a values table behind the same repository interface |
| D-082 | **A plan copy writes one `copied` log entry, on the new plan** (details: source and counts); copied campaigns, initiatives, activities and line items get no entries of their own. | R13: copies carry no history; one readable entry; 011's "Include items inside" still finds it |
| D-083 | **Publish and unpublish are idempotent and carry no version**: publishing a Published plan or unpublishing a Draft plan changes nothing and logs nothing; the server re-checks completeness at the moment of publishing. | A status action does not overwrite fields, so an out-of-date page cannot lose data |
| D-084 | **Copy into "Same year" is always offered; "Next year" only when source year + 1 ≤ current year + 5.** | FR-028 limits only the next year; an old plan can still be duplicated in its own year |
| D-085 | **An owner who is not in any of the plan's teams counts as a field needing attention** ("Owner" in completeness) and blocks publishing; saving such an owner is refused (R1), but a plan whose owner later leaves its teams (003 FR-013) keeps the owner, flagged. | R1 stays true for published plans without blocking 003's membership changes |
| D-086 | **The plan page is one scrolling page of sections** (header, summary cards, plan tree, Details beside Goals, then 011's History); campaign and initiative details (005) open as side panels on **child routes** of `/plans/:planId`. | FR-011–FR-016 list sections, not tabs; 005 FR-017 needs addresses that open the plan page with a panel |
| D-087 | **Restoring a plan from the recycle bin (not Undo) returns it as Draft** and renames it " (restored)", " (restored 2)"… when another plan has its name; Undo of a taken name is refused (`name_taken`). Implemented by the plan kind's `PrepareRestore`. | Implements 011 FR-019/FR-022 and D-037 for plans where the kind lives |
| D-088 | **Budget used is "—" when the plan budget is 0** ("Plan budget is 0"), and the bar is red as soon as the planned cost is above 0. | Avoids a division by zero while still flagging spending without budget |
| D-089 | **The plan seed writes no activity-log entries**; 011 adds the back-dated demo history. | Same as 001's people seed; keeps the log seed in one place |

## Project Structure

### Documentation (this feature)

```text
specs/004-plans/
├── spec.md                    # what and why
├── checklists/requirements.md # spec quality
├── plan.md                    # this file
├── data-model.md              # the stored plan, derived values, statuses, history kind
├── contracts/openapi.yaml     # this feature's paths and schemas (fragment)
└── tasks.md                   # tasks T001…
```

### Source code (repository root)

```text
backend/
├── api/openapi.yaml                 # + the fragment (T001) and the Error change
├── internal/
│   ├── plans/                       # NEW business package
│   │   ├── model.go                 # Plan (stored), enums, API types (views and inputs)
│   │   ├── deps.go                  # Settings and Directory interfaces + their small types
│   │   ├── levels.go                # Level, Facts, AttentionItem, CopyJob; RegisterLevel
│   │   ├── repository.go · store_json.go   # table "plans"
│   │   ├── validate.go              # field checks → errs.Problems (FR-010, FR-007, R1)
│   │   ├── figures.go               # figures, approval progress, completeness (pure)
│   │   ├── goals.go                 # plan goals and planned share (pure, uses the shared waterfall)
│   │   ├── service.go               # Defaults, List, Get, Create, Update, ChangeEffects, Lookup, References
│   │   ├── status.go                # Publish, Unpublish
│   │   ├── copy.go                  # CopyOptions, Copy, copyName, ShiftDate
│   │   ├── kinds.go                 # history kind "plan": snapshot, put, remove, describe, prepareRestore
│   │   ├── handler.go               # routes of the 11 operations
│   │   └── *_test.go                # service_test, figures_test, goals_test, status_test, copy_test,
│   │                                # kinds_test, fakes_test (package plans); handler_test (plans_test)
│   └── app/
│       ├── app.go                   # + plans repository, service, kinds, levels, routes
│       └── plans_deps.go            # NEW adapters: settingsForPlans, directoryForPlans (+ test)
└── seed/seed.go · seed_test.go      # + addPlans (three demo plans)

frontend/
├── e2e/plans.spec.ts                # NEW end-to-end journeys US1–US6 + axe
└── src/
    ├── app/navigation.ts · router.tsx (+ tests)   # Plans leaves Coming soon; 4 routes
    ├── api/errors.ts (+ test)       # + kind notReady and ApiError.missing
    ├── components/                  # NEW TreeTable, ProgressBar, LevelTag, FormDialog, FilterChips;
    │                                # CHANGED SummaryCard (detail), CompletenessChip (itemsBelow); index.ts
    └── features/plans/              # NEW
        ├── plansApi.ts              # queries, mutations, query keys
        ├── planText.ts (+ test)     # words: status, spread, goal stage, approval progress, counts
        ├── recordLinks.ts (+ test)  # record and field links per record type (005–007 add theirs)
        ├── listPreferences.ts (+ test)  # remembered filters per person
        ├── PlansPage.tsx · PlanCard.tsx · PlanTable.tsx (+ css, tests)
        ├── PlanCreatePage.tsx · PlanEditPage.tsx · PlanForm.tsx · GoalFields.tsx (+ css, tests)
        ├── PlanPage.tsx · PlanHeader.tsx · PlanSummaryCards.tsx · PlanDetails.tsx · PlanGoals.tsx (+ css, tests)
        ├── tree/PlanTree.tsx · tree/usePlanTreeNodes.ts (+ css, tests)
        └── dialogs/PublishBlockedDialog.tsx · UnpublishDialog.tsx · CopyPlanDialog.tsx ·
            DeletePlanDialog.tsx · ChangeEffectsDialog.tsx · usePlanActions.ts (+ tests)
```

**Structure decision**: the web-application layout of 001; one Go package per business area; one feature
folder per area (`features/plans`), as `backend/README.md` and `frontend/README.md` describe.

## Backend design

### Package `plans` — constructors and wiring

| Item | Signature |
|---|---|
| Repository | `type Repository interface { Get(ctx, id string) (Plan, bool, error); List(ctx) ([]Plan, error); Put(ctx, Plan) error; Delete(ctx, id string) error }`; `NewJSONRepository() Repository` (table `plans`, `store.Get/List/Put/Delete`) |
| Service | `NewService(tx store.Transactor, repo Repository, generator *ids.Generator, hist *history.Service, settings Settings, directory Directory, now clock.Clock) *Service`; `(*Service).RegisterLevel(l Level) error`; `(*Service).Kinds() []history.Kind` |
| Handler | `NewHandler(s *Service) *Handler`; `(*Handler).Routes(r *web.Router, signedIn web.Wrap)` |
| Wiring in `app.New` | after people, history, settings (002) and teams (003): `plansRepo := plans.NewJSONRepository()`; `planService := plans.NewService(db, plansRepo, generator, historyService, settingsForPlans{…}, directoryForPlans{…}, now)`; register `planService.Kinds()` with history **before** 005's kinds; 005–007 later call `planService.RegisterLevel(…)` in order campaign, initiative, activity, lineItem; `plans.NewHandler(planService).Routes(router, signedIn)` |

Every service method takes `(ctx, a actor.Actor, …)`, runs inside one `s.tx.Read` or `s.tx.Write`, and
returns API types only. `Lookup` and `References` join the caller's transaction (for 005–007 and 002/003).

### Dependencies as interfaces (`deps.go`)

Plans owns these interfaces and small types; `internal/app/plans_deps.go` implements them over the real
002 and 003 services (names on that side are whatever 002/003 ship). All methods run inside the caller's
transaction.

```go
type AttributeValues = map[string]json.RawMessage // attribute ID → value as Settings defines it (opaque here)

// Settings is what plans needs from feature 002.
type Settings interface {
    Model(ctx context.Context, id string) (Model, bool, error)              // active or inactive
    ActiveModels(ctx context.Context) ([]Model, error)                      // sorted by name
    FinanceBudget(ctx context.Context, id string) (FinanceBudget, bool, error)
    // Field problems of the plan-level values (fields "attributes.<id>"); previous lets inactive
    // attributes and options that were already on the plan stay (002 FR-027).
    CheckPlanAttributes(ctx context.Context, values, previous AttributeValues) ([]errs.FieldError, error)
    // Needed-to-publish attributes that are shown (business-unit condition met), active and empty, in
    // display order; a dependent attribute without options for its parent value is not missing (D-034).
    MissingPlanAttributes(ctx context.Context, values AttributeValues) ([]MissingField, error)
    BusinessUnit(ctx context.Context, values AttributeValues) ([]Option, error) // chosen BU options, in order
    // Attribute groups (any level) whose business-unit condition changes from met to not met (hidden) or
    // back (shown) between before and after (FR-020).
    AttributeGroupChanges(ctx context.Context, before, after AttributeValues) (hidden, shown []AttributeGroupChange, err error)
    DescribeAttribute(ctx context.Context, attributeID string, value json.RawMessage) (label, text string, err error)
}

type Model struct {
    ID, Name               string
    Active                 bool
    InqToMqlRate           string        // percent with up to 2 decimals, "2.00"
    MqlToDealRate          string        // percent, "5.00"
    AverageDealSize        money.Amount  // USD
    AllowedActivityTypeIDs []string
}
type FinanceBudget struct { ID string; Year int; Name string; Active bool }
type Option struct { ID, Label string; Active bool }
type AttributeGroupChange struct {
    Unit       string   // the business-unit option's label, "Lubricants"
    Levels     []string // where its attributes are: "plan", "campaign", "initiative", "activity", "lineItem"
    Attributes []string // labels, in display order
}

// Directory is what plans needs from feature 003 (and 002's rates for the display currency).
type Directory interface {
    Team(ctx context.Context, id string) (Team, bool, error)
    TeamsOf(ctx context.Context, personID string) ([]Team, error) // every team the person is in, any status
    Person(ctx context.Context, id string) (Person, bool, error)
    DisplayCurrency(ctx context.Context, a actor.Actor) (Converter, error) // USD until 003 adds the preference
}
type Team struct { ID, Name string; Active bool; MemberIDs []string }
type Person struct { ID, Name string; Active bool }
type Converter struct {
    Currency string                            // "EUR"
    FromUSD  func(usd money.Amount) money.Amount // usd × rate (units per 1 USD), half up to the cent
}
```

### Levels below the plan (`levels.go`) — what 005–007 implement

```go
type Level struct {
    Name string // "campaign", "initiative", "activity", "lineItem"; registered top-down
    // Required. Adds this level's figures for every plan in planIDs to facts[planID] (entries exist).
    AddFacts func(ctx context.Context, planIDs []string, facts map[string]*Facts) error
    // Required. Copies this level's records of job.SourcePlanID whose parent's old ID is in job.NewIDs,
    // per R13, adding old → new IDs and job.Counts[Name]. Writes no log entries.
    Copy func(ctx context.Context, job *CopyJob) error
    // Optional (006). Activities of the plan whose type is not in allowedTypeIDs, ordered by path.
    ActivitiesWithTypeNotAllowed func(ctx context.Context, planID string, allowedTypeIDs []string) ([]errs.RecordRef, error)
}

type Facts struct {
    Campaigns, Initiatives, Activities, ApprovedActivities, LineItems int // approved = Approved, In progress, Complete
    CampaignBudgets money.Amount  // sum of campaign budgets (USD), 0 when none
    PlannedCost     *money.Amount // sum of activity planned costs (USD); nil when nothing below has one
    ActivityGoals   GoalTotals    // sums of all activity goals: INQ, MQL (int64), Revenue (USD)
    NeedsAttention  []AttentionItem // every record below that is not complete, top-down, then by name
}
type AttentionItem struct {
    Record        errs.RecordRef // {type: "campaign", id: "C-1001", name: "Retail growth"}
    Missing       []MissingField
    BlocksPublish bool           // campaigns and initiatives (005); false for activities and line items
}
type CopyJob struct {
    Actor        actor.Actor
    Now          time.Time
    SourcePlanID string
    TargetPlanID string
    Years        int               // 0 same year, 1 next year: every copied date + Years (ShiftDate)
    NewIDs       map[string]string // old ID → new ID of every record copied so far (the plan first)
    Counts       map[string]int    // records copied per level name, for the log details
}
```

`RegisterLevel` refuses an empty or duplicate name and a missing `AddFacts` or `Copy` (start-up error).
Until 005 registers a level, every plan has zero `Facts`: counts 0, campaign budgets 0.00, planned cost
"—", no items below. `ShiftDate(date string, years int) (string, error)` is exported for 005–007: adds
`years` to a `YYYY-MM-DD` date, 29 February → 28 February when the target year is not a leap year, `""`
stays `""` (ongoing activities keep no end date).

### Stored type

`Plan` (table `plans`) — every field, rule and message is in [data-model.md](data-model.md) §1. In short:
`id, name, year, modelId, teamIds, ownerId, aop, description, budget, defaultFinanceBudgetId, defaultSpread,
bcpir, goalStage, goalAmount, attributes, status` + `record.Meta`.

```go
type Status string    // StatusDraft "draft", StatusPublished "published"
type GoalStage string // StageINQ "inq", StageMQL "mql", StageRevenue "revenue"
type Spread string    // SpreadEvenly "evenly", SpreadFirstMonth "firstMonth", SpreadLastMonth "lastMonth", SpreadCustom "custom"

type Plan struct {
    ID                     string          `json:"id"`
    Name                   string          `json:"name"`
    Year                   int             `json:"year"`
    ModelID                string          `json:"modelId"`
    TeamIDs                []string        `json:"teamIds"`                // never nil
    OwnerID                string          `json:"ownerId"`                // "" = none
    AOP                    bool            `json:"aop"`
    Description            string          `json:"description"`
    Budget                 *money.Amount   `json:"budget"`                 // USD; nil = none
    DefaultFinanceBudgetID string          `json:"defaultFinanceBudgetId"` // "" = none
    DefaultSpread          Spread          `json:"defaultSpread"`
    BCPIR                  bool            `json:"bcpir"`
    GoalStage              GoalStage       `json:"goalStage"`
    GoalAmount             string          `json:"goalAmount"`             // "" = none; canonical decimal text
    AttributeValues        AttributeValues `json:"attributes"`        // never nil
    Status                 Status          `json:"status"`
    record.Meta
}
```

API types in `model.go` (JSON exactly as the contract schema in brackets; slices never nil):

| Go type | Contract schema | Notes |
|---|---|---|
| `PlanView` | `Plan` | stored fields as references (`Model ModelSummary`, `Teams []TeamRef`, `Owner *PersonRef`, `DefaultFinanceBudget *FinanceBudgetRef`, `Budget *money.Money`, `GoalAmount *string`) + `OwnerProblem`, `BusinessUnit []OptionRef`, `Figures`, `Goals`, `Completeness`, `Contents`, `record.Meta` |
| `PlanSummary`, `PlanList`, `ListSummary` | `PlanSummary`, `PlanList`, `PlanListSummary` | |
| `Figures`, `Goals`, `GoalFigures`, `GoalTotals`, `Completeness`, `Contents` | `PlanFigures`, `PlanGoals`, `GoalFigures`, `GoalTotals`, `Completeness`, `PlanContents` | `GoalFigures.Reasons` is a struct of four `string` fields with `omitempty` |
| `AttentionItem` (levels.go) | `AttentionItem` | `BlocksPublish` has `json:"-"` |
| `MissingField` = `errs.MissingField` | `MissingField` | |
| `Defaults` | `PlanDefaults` | |
| `PlanInput`, `PlanUpdate` | `PlanCreate`, `PlanUpdate` | `Year *int`, `ModelID *string`, `OwnerID *string`, `Budget *MoneyInput{Amount, Currency string}` (parsed by the service so problems are field errors), `DefaultFinanceBudgetID *string`, `GoalAmount *string`, `DefaultSpread string`, `GoalStage string`; `PlanUpdate.Version *int` |
| `ListFilter` | query of `GET /api/plans` | `Scope, Status, Q string; Year *int` (the handler parses `year`; not a number → problem `year`) |
| `UnpublishRequest` | `UnpublishRequest` | |
| `CopyOptions`, `NextYearOption`, `CopyRequest`, `CopyResult` | `PlanCopyOptions`, `NextYearOption`, `PlanCopyRequest`, `PlanCopyResult` | |
| `ChangeEffectsRequest`, `ChangeEffects`, `AttributeGroupChange` (deps.go, with JSON tags) | `PlanChangeEffectsRequest`, `PlanChangeEffects`, `AttributeGroupChange` | |
| `References` | — (Go only) | `PlanID, ModelID, OwnerID, DefaultFinanceBudgetID string; TeamIDs []string; AttributeValues AttributeValues` |

The reference types (`ModelSummary`, `TeamRef`, `PersonRef`, `FinanceBudgetRef`, `OptionRef`) are built by
`plans` from `Model`, `Team`, `Person`, `FinanceBudget` and `Option` of `deps.go` with JSON tags matching
the 002/003 schemas; an ID that no longer resolves is shown with its ID as name and `active: false`.

### Service operations and their rules

| Operation | Rules (→ FR) |
|---|---|
| `Defaults(ctx, a) (Defaults, error)` | Current year = year of `now()` (UTC); `years` = current − 1 … current + 5; `teamIds` = the actor's **active** teams (`TeamsOf`) sorted by name; `ownerId` = actor when that list is not empty (R1), otherwise null; `modelId` = the only active model, or null when there are 0 or ≥ 2; `aop`/`bcpir` false, `defaultSpread` evenly, `goalStage` inq (FR-006) |
| `List(ctx, a, ListFilter) (PlanList, error)` | `ListFilter{Scope, Status string; Year *int; Q string}`; invalid scope/status → field problems. Scope `myTeams` keeps plans with at least one of `TeamsOf(actor)` (any status; FR-004, Q-006). `total` = plans in scope. `summary` over scope + year (status and search ignored): counts and USD sums of budgets (only set ones; null when none set) and planned costs (only known ones; null when none), converted (FR-001). `items` = scope + year + status + search (`q` trimmed, contained in name or ID ignoring case), sorted year desc, then name ignoring case, then ID (FR-003). `years` = distinct years in scope, newest first. One `AddFacts` per level for all plans in scope (SC-009) |
| `Get(ctx, a, id) (PlanView, error)` | Unknown → `404` "This plan no longer exists — someone may have deleted it."; the view of *View assembly* (FR-011–FR-017) |
| `Create(ctx, a, PlanInput) (PlanView, error)` | Trim name and description; check every field at once (*Validation*); in one `Write`: name free (ignoring case, among stored plans), `ids.Next("P")`, status Draft, `record.NewMeta`, `Put`, log `created` (FR-010) |
| `Update(ctx, a, id, PlanUpdate) (PlanView, error)` | `version` missing or < 1 → problem `version` "Reload the page and try again."; field checks (year is not in the body; it never changes, FR-018); in one `Write`: get (404), `CheckVersion(version, "plan")` → `409 version_conflict` naming who and when (FR-023), name free (excluding itself), R1 (FR-021), unchanged inactive model/teams/owner/finance budget/options allowed; status unchanged — a Published plan stays Published (FR-022); changes = `history.Diff` on the record without `attributes` + one change per attribute whose value differs (`attributes.<id>`); no change → nothing saved, no version, no log; else `Touch`, `Put`, log `changed` with the changes (FR-022, FR-031) |
| `ChangeEffects(ctx, a, id, ChangeEffectsRequest) (ChangeEffects, error)` | Read-only. `modelChanged` = request model ≠ stored; when changed and the model exists: every level's `ActivitiesWithTypeNotAllowed(planID, newModel.AllowedActivityTypeIDs)` (FR-019); `hiddenGroups`/`shownGroups` = `AttributeGroupChanges(stored values, request values)` (FR-020). Unknown model → problem `modelId` "Choose a model from the list." |
| `Publish(ctx, a, id) (PlanView, error)` | One `Write`: get (404); Published → unchanged, no log; readiness = own missing fields (completeness) + every level item with `BlocksPublish`; any → `409 not_ready` "Publish needs <n> field(s)" with `missing` groups (plan first, then items in order) and nothing changes; else status Published, `Touch`, `Put`, log `statusChanged` verb `published` with the status change (FR-024). Remaining budget, campaign dates and activities never block (FR-024, US5#4) |
| `Unpublish(ctx, a, id, UnpublishRequest) (PlanView, error)` | Reason trimmed, 1–500 characters (problems on `reason`); Draft → unchanged, no log; else status Draft, `Touch`, `Put`, log `statusChanged` verb `unpublished` with the status change and `Comment` = reason (FR-025) |
| `CopyOptions(ctx, a, id) (CopyOptions, error)` | `name` = `copyName(source name)`; `sameYear` = source year; `nextYear{year, available, reason}` with available = source year + 1 ≤ current + 5, else reason "Plans can only be created up to <current + 5>."; `contents` = the plan's counts (FR-026, FR-028) |
| `Copy(ctx, a, id, CopyRequest) (CopyResult, error)` | Name checked as on create (incl. unique); `year` must be source year or source year + 1, the latter only when available (problems on `year`); one `Write` (all or nothing, FR-027): new ID, every source field copied (owner, teams, budget, goal, attribute values, AOP, BCPIR, spread — D-035), status Draft, creator = actor; next year: `year + 1` and `defaultFinanceBudgetId` cleared when it was set (note "Default finance budget cleared — it belonged to <year>."); `includeContents` → every level's `Copy` in registration order with `NewIDs{source → new}`; log one `copied` entry on the new plan with details (*Activity log*); answer the new plan's view + notes (FR-026–FR-028) |
| `Delete(ctx, a, id) (history.BinEntry, error)` | One `Write`: get (404); `history.MoveToBin(ctx, a, "plan", id)` — the plan and every record below (kinds of 005–007) in one entry; history logs `deleted` with the contents (FR-029). Undo/Restore by 001's endpoints (FR-030) |
| `Lookup(ctx, a, id) (Plan, bool, error)` | For 005–007: the stored plan inside the caller's transaction |
| `References(ctx, a) ([]References, error)` | For 002/003 "Used by" counts: per stored plan `{PlanID, ModelID, TeamIDs, OwnerID, DefaultFinanceBudgetID, AttributeValues}`; `ReferencesFromSnapshot(data json.RawMessage) (References, error)` decodes a plan snapshot of the recycle bin |

**Validation** (`validate.go`, every problem collected before any return; texts in *User-visible messages*):

| Field | Check |
|---|---|
| `name` | required; ≤ 120 characters after trimming; unique among plans ignoring case and surrounding spaces (checked inside the `Write`) |
| `year` (create, copy) | required; create: current − 1 … current + 5; copy: source year or source year + 1 (≤ current + 5) |
| `modelId` | required; exists; active unless unchanged on update |
| `teamIds` | each exists; active unless already on the plan; duplicates removed; stored sorted by team name |
| `ownerId` | optional; exists; active unless unchanged; with no team → "Choose teams first — …"; otherwise a member of at least one chosen team (R1) |
| `description` | ≤ 2,000 characters after trimming |
| `budget` | optional; `currency` must be USD; amount parses (`money.Parse`), ≥ 0, ≤ 2 decimals |
| `defaultFinanceBudgetId` | optional; exists; its year = plan year; active unless unchanged |
| `defaultSpread` | one of evenly, firstMonth, lastMonth, custom |
| `goalStage` | one of inq, mql, revenue |
| `goalAmount` | optional; ≥ 0; inq/mql: whole number; revenue: ≤ 2 decimals |
| `attributes` | `Settings.CheckPlanAttributes` problems appended as they come |

### View assembly and calculations (`figures.go`, `goals.go`)

All arithmetic is exact (`money.Amount`, `math/big.Rat`), in USD, converted at the end with the actor's
`Converter`. Rounding is **half up** (amounts ≥ 0; negative amounts round half away from zero).

| Value | Rule (→ FR) |
|---|---|
| `figures.budget` | plan budget, or null |
| `figures.campaignBudgets` | `Facts.CampaignBudgets` (0.00 without campaigns) — "Allocated to campaigns" |
| `figures.remainingBudget` | budget − campaign budgets; null without budget (FR-012, domain §4.1) |
| `figures.overAllocatedBy` | −remaining when remaining < 0, else null (D-015) |
| `figures.plannedCost` | `Facts.PlannedCost` or null ("—") (domain §4.2) |
| `figures.budgetUsedPercent` | null when no budget or budget = 0; else round(planned cost (0 when null) × 100 ÷ budget) — a whole number |
| `figures.overBudget` | planned cost > budget (budget 0 counts; no budget → false) — red bar above 100 % |
| `figures.activities` / `activitiesApproved` | `Facts.Activities` / `Facts.ApprovedActivities` |
| `figures.approvalProgress` | `noActivities` (0 activities), `allApproved` (approved = activities > 0), else `inProgress` (D-026) |
| `completeness.missing` | own needed-to-publish fields in form order: Teams (`teamIds` empty), Owner (`ownerId` empty **or** not in any team), then `MissingPlanAttributes` (business unit and others), Goal stage (never empty in practice) (FR-017) |
| `completeness.itemsBelow` | `Facts.NeedsAttention` (every level) as `{record, missing}` |
| `contents` | `{campaigns, initiatives, activities, approvedActivities, lineItems}` from `Facts` |
| `ownerProblem` | the R1 message when the owner is set and in none of the teams; "" otherwise (FR-021) |
| `businessUnit` | `Settings.BusinessUnit(values)` |

**Goal waterfall** (shared function, D-075). Rates `r1` (INQ→MQL) and `r2` (MQL→deal) are
percentages with two decimals, so `R1 = r1 × 100` and `R2 = r2 × 100` are integers; the deal size `D` is in
cents. `round()` = half up to an integer.

| Goal stage | INQ | MQL | Deals | Revenue (cents) |
|---|---|---|---|---|
| inq, amount `I` | `I` | `round(I × R1 ÷ 10000)` | `round(MQL × R2 ÷ 10000)` | `Deals × D` |
| mql, amount `M` | `round(M × 10000 ÷ R1)` | `M` | `round(M × R2 ÷ 10000)` | `Deals × D` |
| revenue, amount `V` (cents) | `round(MQL × 10000 ÷ R1)` | `round(Deals × 10000 ÷ R2)` | `round(V ÷ D)` | `V` |

A division by `R1 = 0`, `R2 = 0` or `D = 0` gives null for that stage (and the stages computed from it) with
the reason "The model's <INQ→MQL rate \| MQL→deal rate \| average deal size> is 0, so this can't be worked
out." Without a goal amount every calculated stage is null with no reason (the page says "No goal set
yet"). Checks: INQ 250,000 with 2 %, 5 %, 10,000 → MQL 5,000, deals 250, revenue 2,500,000.00; Revenue
2,500,000.00 → deals 250, MQL 5,000, INQ 250,000 (FR-009, SC-006).

`goals.activities` = `Facts.ActivityGoals` (INQ, MQL, revenue). `goals.plannedPercent` = round(activity total
of the goal stage × 100 ÷ plan goal of that stage) — null when the plan goal is null or 0. Example: INQ goal
250,000 and activity INQ 150,000 → 60 (FR-016, US3#5).

### History kind `plan` (`kinds.go`)

| Part | Value |
|---|---|
| `Name` / `Singular` / `Plural` / `Parent` | `plan` / `plan` / `plans` / — |
| `Snapshot` | the stored plan, `Name` = plan name, `ParentID` "" |
| `Remove` / `Put` | `repo.Delete` / decode and `repo.Put` exactly |
| `Describe` | labels and texts of *Activity log* below; `attributes.<id>` → `Settings.DescribeAttribute`; unknown ids of models, teams, people or budgets → the id itself |
| `PrepareRestore` | another stored plan has the name (ignoring case): Undo → `errs.NameTaken("Can't undo — there is now another plan called '<other name>'. Restore it from the recycle bin instead.")`; Restore → name " (restored)", " (restored 2)", … + note "Renamed to '<new name>' because the name was taken.". Restore (not Undo) of a Published plan → status Draft + note "Returned as Draft — publish it again when it's ready." Any change → `Meta.Touch(rc.Actor, rc.Now)` |

### Activity log

| Person's action | Entries |
|---|---|
| Create | 1 × `created` (record `plan`, path empty) |
| Edit with changes | 1 × `changed`; `changes` = each changed field, labels: name "Plan name", modelId "Model", teamIds "Teams", ownerId "Owner", aop "Strategic / AOP plan", description "Description", budget "Plan budget", defaultFinanceBudgetId "Default finance budget", defaultSpread "Default line-item spread", bcpir "BCPIR plan", goalStage "Goal stage", goalAmount "Goal amount", `attributes.<id>` the attribute's label. Texts: model / team / person / budget names; money "1,200,000.00 USD"; Yes / No; Evenly · All in first month · All in last month · Custom; INQ · MQL · Revenue; goal "250,000" or "2,500,000.00 USD" |
| Edit without change, or refused | none |
| Publish | 1 × `statusChanged`, verb `published`, change `status` Draft → Published |
| Unpublish | 1 × `statusChanged`, verb `unpublished`, change `status` Published → Draft, `comment` = reason |
| Copy | 1 × `copied` on the **new** plan, details "from '<source name>' (<source ID>)" + ", with 2 campaigns, 4 initiatives, 7 activities and 15 line items" when contents were copied (counts > 0 only, history's count wording) |
| Delete | 1 × `deleted` (by `MoveToBin`), details "with …" |
| Undo / Restore | 1 × `restored` (by history) with the kind's notes |

### Seed (`backend/seed/seed.go`)

`addPlans(ctx, generator, createdAt) (map[string]string, error)` after the 002 and 003 seed functions, through
`plans.NewJSONRepository()`, created by `actor.System`, no log entries; returns name → ID for 005's seed.
Model, team, person, finance budget and attribute option IDs are looked up by name from the 002/003 seeds.

| ID | Name | Year | Model | Team | Owner | Budget (USD) | Status | Business unit | Other |
|---|---|---|---|---|---|---|---|---|---|
| P-1001 | FY2026 Lubricants Europe | 2026 | Lubricants 1.0 | Lubricants Europe | Priya Raman | 1,200,000.00 | Published | Lubricants | goal INQ 250,000; default finance budget "2026 Lubricants Budget"; spread Evenly; description "Lubricants marketing in Europe for 2026." |
| P-1002 | FY2026 Mobility China | 2026 | Mobility 1.0 | Mobility China | Mei Lin | 800,000.00 | Draft | — (deliberately missing) | goal INQ 150,000; default finance budget "2026 Mobility Budget" |
| P-1003 | FY2027 Global Brand | 2027 | Lubricants 1.0 | Global Marketing Ops | Alex Morgan | 500,000.00 | Draft | Lubricants, Mobility | Strategic / AOP Yes; SMP Yes; goal Revenue 2,000,000.00; no default finance budget |

Seed test: three plans with these values; P counter at 1003; `List` order FY2027 Global Brand, FY2026
Lubricants Europe, FY2026 Mobility China; summary 3 plans, 2 Draft, 1 Published (US2#1).

## API operations

Tag **Plans**. All need a session (`401` otherwise). Errors use 001's `Error`; texts in *User-visible
messages*. Schemas in [contracts/openapi.yaml](contracts/openapi.yaml).

| Method and path | Request | Response | Errors |
|---|---|---|---|
| `GET /api/plan-defaults` | — | `200 PlanDefaults` | 401 |
| `GET /api/plans` | query `scope` (all \| myTeams, default all), `status` (draft \| published), `year` (integer), `q` | `200 PlanList` `{items: PlanSummary[], total, summary, years}` | 400 `validation_failed` (scope, status, year), 401 |
| `POST /api/plans` | `PlanCreate` | `201 Plan` | 400 `validation_failed` (every field at once) / `bad_request`, 401 |
| `GET /api/plans/{planId}` | — | `200 Plan` | 401, 404 |
| `PATCH /api/plans/{planId}` | `PlanUpdate` (whole form + `version`) | `200 Plan` | 400, 401, 404, 409 `version_conflict` |
| `DELETE /api/plans/{planId}` | — | `200 DeleteResult` | 401, 404 |
| `POST /api/plans/{planId}/publish` | — | `200 Plan` | 401, 404, 409 `not_ready` (with `missing`) |
| `POST /api/plans/{planId}/unpublish` | `UnpublishRequest {reason}` | `200 Plan` | 400 (reason), 401, 404 |
| `GET /api/plans/{planId}/copy-options` | — | `200 PlanCopyOptions` | 401, 404 |
| `POST /api/plans/{planId}/copy` | `PlanCopyRequest {name, year, includeContents}` | `201 PlanCopyResult {plan, notes}` | 400 (name, year), 401, 404 |
| `POST /api/plans/{planId}/change-effects` | `PlanChangeEffectsRequest {modelId, attributes}` | `200 PlanChangeEffects` | 400 (modelId, attributes), 401, 404 |

**Contract changes to shared schemas** (applied when merging, T001): `Error.code` gains `not_ready`;
`Error` gains optional `missing: ReadinessGroup[]` (only with `not_ready`); the `Conflict` response
description lists `not_ready`. Go: `errs.CodeNotReady`, `errs.NotReady(message string, missing
[]ReadinessGroup) *Error` (status 409), `Error.Missing []ReadinessGroup \`json:"missing,omitempty"\``,
`errs.ReadinessGroup{Record RecordRef; Fields []MissingField}` and `errs.MissingField{Field, Label string}`
(`plans.MissingField` is an alias of it).

## Frontend design

### Routes and navigation

| Path | Page | Tab title (`handle.title`, then the page's own) |
|---|---|---|
| `/plans` | `PlansPage` | Plans · MAPP |
| `/plans/new` | `PlanCreatePage` | New plan · MAPP |
| `/plans/:planId` | `PlanPage` (layout route with `<Outlet/>`; 005 adds child routes for side panels) | "<plan name> · MAPP" once loaded ("Plan" before) |
| `/plans/:planId/edit` | `PlanEditPage` (`?focus=<field>` focuses that field after loading) | "Edit <plan name> · MAPP" ("Edit plan" before) |

- `navigation.ts`: `PLANS` becomes a `NavItem` (no `comingSoon`) and leaves `COMING_SOON_SECTIONS`;
  `router.tsx` adds the four routes inside `RequireSignIn`; Plans stays highlighted on every `/plans/…`
  address (NavLink without `end`). `navigation.test.ts`/`router.test.tsx` updated.
- **View history** goes to `/activity-log?recordType=plan&recordId=<id>` (Coming soon until 011).
- The smoke test's check (heading "Plans", level 1, on `/plans`) stays true.

### Data (`plansApi.ts`)

Query keys: `['plans', filter]`, `['plan', planId]`, `['plan-defaults']`, `['plan-copy-options', planId]`.
Hooks: `usePlans(filter)` (keeps previous data while refetching — `placeholderData: keepPreviousData`),
`usePlan(planId)`, `usePlanDefaults()`, `useCopyOptions(planId, enabled)`, mutations `useCreatePlan`,
`useUpdatePlan` (version from the form, never the cache), `useChangeEffects`, `usePublishPlan`,
`useUnpublishPlan`, `useCopyPlan`, `useDeletePlan`; `fetchLatestPlan(queryClient, id)` for conflicts. Every
mutation invalidates `['plans']` and `['plan', id]` (and `['recycle-bin']`, `['activity-log']`) when settled.
Settings and teams data come from 002/003 hooks (*Expects*).

### Shared components

| Component | Status | Use / reason |
|---|---|---|
| Button, LinkButton, PageHeader, Card, SummaryCard, StatusChip, CompletenessChip, InfoTip, Tooltip, ActionsMenu, Avatar, ConfirmDialog, Toast/useToast, EmptyState, Skeleton, FormErrorSummary, TextField, ErrorState | reuse (001) | everywhere below |
| `useUndoableDelete`, `useUnsavedChanges`, `usePageTitle`, `formatDate`, `formatDateTime`, `formatMoney`, `formatMoneyCompact`, `conflictMessage` | reuse (001) | delete with Undo, form guard, titles, formats, conflicts |
| **TreeTable** (`components/TreeTable/` + `filterTree.ts`) | **new** | design system §5 tree table; plan tree (004), campaign/initiative/activity rows (005, 006), All work grid (010) |
| **ProgressBar** | **new** | design system §4 progress bars; budget used (004, 005), approval progress (005), Home card (009), All work (010) |
| **LevelTag** | **new** | design system §4 level bar and tag (PLAN, CAMPAIGN, …); tree rows and headers of 004–007 |
| **FormDialog** | **new** | a centred dialog with a small form (Radix Dialog; first field focused): copy (004–006), unpublish reason (004), decline comment (008), move to… (005, 006). `ConfirmDialog` is an alert dialog whose description must stay text |
| **FilterChips** | **new** | design system §6.1 removable filter chips: Plans list (004), All work (010), Activity log (011) |
| SummaryCard | **changed**: optional `detail?: ReactNode` under the value | "Over-allocated by …", "No plan budget set", a budget-used bar (004); Home cards (009) |
| CompletenessChip | **changed**: optional `itemsBelow?: {id, label, href}[]` → a second chip "1 item below needs attention" / "<n> items below need attention" whose popover lists links | FR-017; parents in 005 |
| Breadcrumb, SearchField, PillFilter, SelectField, MultiChoiceField, TextAreaField, YesNoField, MoneyField, NumberField, AttributeFields, AttributeValueList | **expected from 002** (built first; 002's own forms need them) | form and list controls; Shell attributes on the form and in Details |
| PersonPicker (team-restricted mode), TeamPills | **expected from 003** | Owner; teams on cards, header and Details |

New and changed components follow 001's rules: own folder, CSS Module on tokens, a test, export in
`components/index.ts`. Props:

- `TreeTable`: `label: string` (accessible name), `columns: {key, header, align?: 'start' | 'end'}[]`,
  `nodes: TreeNode[]`, `expandedIds: Set<string>`, `onToggle(id)`, `emptyText?: string`,
  `footer?: ReactNode`. `TreeNode = {id, level: 'plan' | 'campaign' | 'initiative' | 'activity' | 'lineItem',
  name, href?, onOpen?, cells: Record<string, ReactNode>, children: TreeNode[], actions?: ActionsMenuItem[],
  highlighted?: boolean}`. ARIA treegrid: rows `aria-level`, `aria-expanded` (rows with children), roving
  `tabIndex`; ↑/↓ move, → expands or goes to the first child, ← collapses or goes to the parent, Home/End,
  Enter opens; each expandable row has a chevron button "Expand <name>" / "Collapse <name>". Level bar 4 px
  in the level colour + `LevelTag`. Money columns right-aligned.
- `filterTree(nodes, query) → {nodes, openIds}`: keeps nodes whose name or id contains the trimmed query
  (ignoring case) and all their ancestors; `openIds` = every ancestor of a match.
- `ProgressBar`: `value: number | null` (percent), `label: string` (accessible name), `tone?: 'problem'`;
  `role="progressbar"`, `aria-valuenow` capped display at 100 % width; brand-gradient fill, red when
  `tone='problem'`; "—" text handled by the caller.
- `LevelTag`: `level` → upper-case "PLAN", "CAMPAIGN", "INITIATIVE", "ACTIVITY", "LINE ITEM" on a 12 % tint of
  `--level-<level>`, text in `--ink`.
- `FormDialog`: `open`, `onOpenChange`, `title`, `description?`, `children`, `submitLabel`, `onSubmit`,
  `busy?`, `cancelLabel?` ('Cancel'), `hideCancel?`; Radix Dialog (not AlertDialog), a `<form noValidate>`;
  the first field (or the submit button when there is none) gets the focus; Enter submits; a second submit
  while `busy` is ignored; focus returns to the opener.
- `FilterChips`: `chips: {key, label}[]`, `onRemove(key)`, `onClearAll?`; each chip is a button "Remove
  filter: <label>".

### Plans list (`/plans`, US2, FR-001–FR-004)

- `PageHeader`: eyebrow "Plans", title "Plans", description, action **New plan** (`LinkButton` primary →
  `/plans/new`).
- Five `SummaryCard`s (compact money with full value): Plans · Draft · Published · Total planned cost ·
  Total plan budget, with tips; they follow scope + year.
- Toolbar (wraps on narrow screens): `PillFilter` "Status" (All · Draft · Published); `SelectField` "Year"
  (All years + `years`); `PillFilter` "Show" (All plans · My teams) with the My-teams hint under the toolbar
  while on; `SearchField` "Search name or ID" (sent as `q` 250 ms after typing stops); `PillFilter` "Layout"
  (Cards · Table); then `FilterChips` (status, year, My teams, search) and the count "<n> of <m>".
- **Cards** (`PlanCard`, grid 1–3 columns): name (link to the plan page), "P-1001 · 2026", chips (status,
  completeness — choosing a field opens `/plans/<id>/edit?focus=<field>`, AOP), owner with avatar or "No owner
  yet", `TeamPills`, "Business unit: <labels>" or "No business unit yet", a budget block (`ProgressBar` of
  budget used, "Planned cost <a> · Budget <b> · <p> % used" or "Planned cost <a> · No budget set", red above
  100 %), "<n> of <m> activities approved", and `ActionsMenu` (View, Edit, Copy, Publish or Unpublish,
  Delete last after a separator). Selecting the card's name opens the plan page — never the form.
- **Table** (`PlanTable`): columns Name (link + ID), Year, Status, Completeness, Owner, Teams, Business unit,
  Budget used (bar + %), Activities approved, actions. Horizontal scroll inside its card on narrow screens.
- Filters come from the address, else from `listPreferences` (per person), else defaults (All, All years,
  All plans, Cards); every change updates the address (`replace`) and the remembered values.
- Actions run through `usePlanActions` (shared with the plan page): publish (blocked dialog), unpublish
  dialog, copy dialog (→ new plan page), delete dialog (stays on the list; toast with Undo).

### Plan form (`/plans/new`, `/plans/:planId/edit`, US1, US4, FR-005–FR-010, FR-018–FR-023)

`PlanForm` (one column, max 720 px) is shared by both pages; it receives initial values, the mode and
`onSubmit`, and keeps the values once (001's edit-form rule). Sections and fields in dependency order:

| Section | Fields (id = API field name) | Marking | Behaviour |
|---|---|---|---|
| Basics | Plan name (`name`, max 120) · Year (`year`) · Model (`modelId`) · Teams (`teamIds`) · Owner (`ownerId`) · Strategic / AOP plan (`aop`) · Description (`description`, max 2,000) | Required · Required · Required · Needed to publish · Needed to publish · — · — | Year: create `SelectField` of `years`; edit read-only with a lock icon and the reason (FR-018). Model: active models only, the chosen one's rates and deal size as hint; an inactive current model shows "<name> (inactive)". Owner: `PersonPicker` team-restricted with `teamIds`; disabled with the reason while no team; the R1 message under it when the owner is in none of the teams (FR-021) |
| Budget | Plan budget (`budget`, USD `MoneyField`) · Default finance budget (`defaultFinanceBudgetId`, finance budgets of the plan's year, "No default" first) · Default line-item spread (`defaultSpread`) · BCPIR plan (`bcpir`) | — | Changing the year clears a budget of another year with the note (FR-007). On edit of an existing plan, a changed default finance budget or spread shows "Applies to line items created from now on." (FR-022) |
| Shell attributes | `AttributeFields` level `plan` (business unit, SMP and every other active plan-level attribute, display order, markings, condition texts) — ids `attributes.<attributeId>` | per Settings | FR-008, R9 |
| Goals | Goal stage (`goalStage`, INQ · MQL · Revenue) · Goal amount (`goalAmount`; `NumberField` whole number for INQ/MQL, `MoneyField` USD for Revenue) · `GoalFields` calculated rows | Needed to publish · — | Live waterfall with the shared TS function; each calculated stage labelled "Calculated" with its formula; "Choose a model to calculate" without a model (FR-009) |

- **Defaults** (create): from `GET /api/plan-defaults`; each defaulted field shows its default hint until
  the person changes it (FR-006).
- **Sticky footer**: **Save** (primary, never disabled) · **Cancel** (→ the list or the plan page; the
  unsaved-changes guard asks first).
- **Save on create**: `POST /api/plans`; `validation_failed` → `FormErrorSummary` + messages under fields,
  focus to the summary (FR-010); success → navigate to `/plans/<id>` (guard off) and toast "Plan '<name>'
  created — add campaigns next." (US1#2).
- **Save on edit**: if the model or the business unit changed → `POST …/change-effects`; when it lists
  disallowed activities or attribute groups → `ChangeEffectsDialog` (title, parts, **Change model** when only
  the model is involved, otherwise **Save changes**; **Keep editing**) before `PATCH` (FR-019, FR-020). Then
  `PATCH`; `version_conflict` → alert with `conflictMessage(error, 'plan')`, typed values kept, version taken
  from `fetchLatestPlan` (FR-023); success → `/plans/<id>`, toast "Plan '<name>' saved".
- A Published plan's edit page shows the published banner (FR-022). `?focus=<field>` focuses and scrolls
  to that field once the form is drawn (links from completeness and the publish dialog).

### Plan page (`/plans/:planId`, US3, FR-011–FR-017) — layout and extension points

```
PlanPage  (layout route; usePageTitle(plan.name))
├── PlanHeader        Breadcrumb "Plans › <name>" · LevelTag PLAN · <h1> name · StatusChip · CompletenessChip
│                     (+ itemsBelow) · AOP chip · meta row: ID · Owner · Year · TeamPills · Business unit ·
│                     "Created by … on …, last changed by … on …" · actions: Edit (primary) ·
│                     Publish | Unpublish (secondary) · ⋯ More = ActionsMenu (Copy, View history, Delete)
├── PlanSummaryCards  Plan budget · Planned cost · Remaining budget · Budget used · Activities approved
├── PlanTree          Card "Plan tree": toolbar [SearchField, Expand all, Collapse all, {toolbarActions}]
│                     TreeTable(nodes = usePlanTreeNodes(plan), footer = {treeFooter})
├── div.columns       (two columns from 1024 px, stacked below)
│   ├── PlanDetails   Card "Details"
│   └── PlanGoals     Card "Goals"
├── {011: PlanHistory card}
└── <Outlet/>         {005: campaign / initiative side panels on child routes}
```

| Extension point | Who | How |
|---|---|---|
| Tree rows | 005 (campaigns, initiatives), 006 (activities) | edit `tree/usePlanTreeNodes.ts`: 004 returns `[planNode]` with `children: []`; later features fill `children` from their own queries |
| Tree toolbar action **Add campaign** and the totals row | 005 | `PlanTree` props `toolbarActions?: ReactNode` and `footer?: ReactNode`, passed by `PlanPage` |
| Side panels with their own address | 005 | child routes of `/plans/:planId` (e.g. `campaigns/:campaignId`); `PlanPage` renders `<Outlet/>` |
| Record and field links (completeness, publish dialog, model-change list) | 005–007 | add entries to `RECORD_LINKS` in `recordLinks.ts` (`label`, `record(ref, planId)`, `field(ref, field, planId)`) |
| History section | 011 | a card after the columns |
| SP&A / BCPIR views | 013 | its own section or route, decided in 013 |

- **Header** facts: Owner "Owner: <name>" (+ " (inactive)", + a warning icon followed by the
  `ownerProblem` text in `--status-problem-text` when set — never in a tooltip only), or "No owner yet";
  "Year 2026"; teams (inactive "(inactive)"); "Business unit: Lubricants" or "No business unit yet".
  **Publish** on Draft, **Unpublish** on Published. **⋯ More** is `ActionsMenu` with `label` = the plan name
  (accessible name "Actions for <plan name>") and the items Copy, View history, separator, Delete.
- **Summary cards** (values in the display currency, compact with full value; tips with formulas):
  Plan budget (no budget → "—", detail "No plan budget set"); Planned cost ("—" detail "Nothing planned yet"
  when null); Remaining budget (red with detail "Over-allocated by <amount>" when negative; "—" detail "No
  plan budget set"); Budget used ("<p> %" + `ProgressBar`, red when over budget; "—" + reason); Activities
  approved ("<n> of <m>", detail "No activities yet" / "In progress" / "All approved").
- **Plan tree** (FR-013, FR-014): columns Name, Owner, Dates, Status, Budget, Planned cost, Budget used.
  Plan row: plan level bar + PLAN tag, name (no link), owner, "1 Jan – 31 Dec <year>", status +
  completeness chips, budget with "Remaining <amount>" / red "Over-allocated by <amount>" / "No budget set"
  beneath, planned cost, budget used bar. Plan row expanded by default. Search filters with `filterTree`
  and opens the path; no match → "Nothing in this plan matches '<text>'"; plan without campaigns → the
  empty text under the plan row. **Expand all** / **Collapse all** set `expandedIds`.
- **Details** (FR-015): definition lists in groups Basics (Plan name, Year, Model with "(inactive)" and its
  rates, Teams, Owner, Strategic / AOP plan, Description), Budget (Plan budget, Default finance budget, Default
  line-item spread, BCPIR plan), Shell attributes (`AttributeValueList` level plan: only shown attributes,
  "—" when empty, "(inactive)" values); subtitle "Amounts in <currency>". Empty values "—".
- **Goals** (FR-016): stage table — rows INQ, MQL, Revenue; columns "Plan goal" (typed stage plain, other
  stages with "Calculated" and an `InfoTip` formula; null with reason → "—" + reason) and "Planned in
  activities"; below "<p> % of the <stage> goal planned" or "No goal set yet — set a goal amount in Edit."
- **States**: loading → `Skeleton` blocks in the page frame; error → `ErrorState` with **Try again**;
  `404` → "Plan not found" card with **Go to Plans**.

### Dialogs (`dialogs/`)

| Dialog | Component | Content and behaviour (→ FR) |
|---|---|---|
| Publish blocked | `PublishBlockedDialog` (`FormDialog` with `hideCancel` and `submitLabel` "Close"; submitting closes it) | title "Publish needs <n> field(s)", intro, groups ("Plan", "Campaign '<name>'", …), each field a link from `recordLinks` (following a link closes the dialog); the plan stays Draft (FR-024) |
| Unpublish | `UnpublishDialog` (`FormDialog`) | title, `TextAreaField` reason (Required), hint; **Unpublish plan**; problems from the API under the field (FR-025) |
| Copy | `CopyPlanDialog` (`FormDialog`) | loads `copy-options`; name (`TextField`, pre-filled, editable); year radios "Same year (<y>)" / "Next year (<y+1>)" (the latter disabled with its reason when unavailable); checkbox "Include campaigns, initiatives, activities and line items" (on) with the counts or "This plan has nothing inside yet."; next-year note; **Copy plan**; success → `/plans/<new id>`, toast, and one toast per note (FR-026–FR-028) |
| Delete | `DeletePlanDialog` (`ConfirmDialog`, tone danger) | title, contents sentence, published sentence, approved-activities sentence, recycle-bin sentence; **Cancel** focused; **Delete plan**; then `useUndoableDelete({invalidate: [['plans']]})` and, from the plan page, navigate to `/plans` (FR-029, FR-030) |
| Consequences | `ChangeEffectsDialog` (`ConfirmDialog`) | model part: "<n> activities have a type the new model doesn't allow:" + links + the effect sentence; business-unit part: one sentence per hidden / shown group; buttons per *Plan form* (FR-019, FR-020) |

`usePlanActions(plan)` returns the action handlers and the dialogs' state for the list and the plan page.

### States

| Page | Loading | Empty | Error | Validation | Conflict | Not found |
|---|---|---|---|---|---|---|
| Plans list | 5 skeleton cards + skeleton list; old data kept while refetching | no plans: `EmptyState` + **Create your first plan**; filters hide all: "Nothing matches these filters" + **Clear filters** | `ErrorState` + Try again (cards and list area) | invalid address filters are ignored (defaults used) | — | — |
| Create | skeleton form while defaults load | — | defaults failed: `ErrorState` | summary + field messages, focus summary | — | — |
| Edit | skeleton form | — | `ErrorState` | as create | alert `conflictMessage(error, 'plan')`, values kept | "Plan not found" + Go to Plans |
| Plan page | skeleton header, cards, tree | tree empty text; Goals "No goal set yet…" | `ErrorState` | — | — | "Plan not found" + Go to Plans |
| Dialogs | copy: skeleton lines while options load | — | API message under the form or in a toast | messages under fields | — | toast with the API message, then the list refreshes |

## User-visible messages

Every text appears here once; code and tests quote it. `<…>` is filled in at run time. Texts of 001
(conflict, unsaved changes, Undo, errors) are reused unchanged.

| Where | Text |
|---|---|
| Plans header | eyebrow "Plans" · title "Plans" · "Every yearly marketing plan, its status, its budget and what's still missing." · **New plan** |
| Summary cards (list) | Plans — tip "Plans in the chosen year and teams." · Draft — "Plans still being prepared." · Published — "Plans agreed as the plan of record." · Total planned cost — "The planned cost of these plans added up: the sum of their activities' planned cost." · Total plan budget — "The plan budgets of these plans added up. Plans without a budget add nothing." |
| List toolbar | Status: All · Draft · Published · Year: All years · Show: All plans · My teams · Search name or ID · Layout: Cards · Table · "<n> of <m>" · **Clear filters** · chip "Remove filter: <label>" |
| My teams hint | Shows your teams' plans. Everyone can still open every plan in this phase. |
| List empty / filtered | No plans yet — create your first plan · A plan holds one year of marketing for one business. · **Create your first plan** / Nothing matches these filters · **Clear filters** |
| Card and row | "<ID> · <year>" · Owner: <name> · No owner yet · Business unit: <labels> · No business unit yet · Planned cost <a> · Budget <b> · <p> % used · No budget set · <n> of <m> activities approved · AOP |
| Row actions | View · Edit · Copy · Publish · Unpublish · Delete |
| Create page | eyebrow "Plans" · title "New plan" · "A name, a model and a business unit are enough to start — everything else can follow." |
| Edit page | eyebrow "Plans" · title "Edit plan" · description "<plan name>" · breadcrumb "Plans › <plan name> › Edit" |
| Form sections | Basics · Budget · Shell attributes · Goals |
| Field labels | Plan name · Year · Model · Teams · Owner · Strategic / AOP plan · Description · Plan budget · Default finance budget · Default line-item spread · BCPIR plan · Goal stage · Goal amount |
| Markings | Required · Needed to publish |
| Defaults | Default: the current year · Default: your teams · Default: you · Default: the only active model · Default: No · Default: Evenly · Default: INQ |
| Model hint | INQ→MQL <r1> % · MQL→deal <r2> % · average deal size <amount> |
| Owner disabled | Choose teams first — the owner must belong to one of the plan's teams |
| Owner not in teams (R1) | <person> isn't in any of the chosen teams — choose another owner or clear the field |
| Year locked | Year can't change once the plan is saved — copy the plan into the next year or create a new plan. |
| Finance budget | No default · Default finance budget cleared — '<budget>' belongs to <year>. · No finance budgets for <year> yet — an administrator adds them in Settings. |
| Changed defaults (edit) | Applies to line items created from now on. |
| Spread options | Evenly · All in first month · All in last month · Custom |
| Goal stage options | INQ (inquiries) · MQL (marketing-qualified leads) · Revenue |
| Goal amount label | Goal amount (INQ) · Goal amount (MQL) · Goal amount (Revenue, USD) |
| Calculated stages | Calculated · INQ = MQL ÷ <r1> % · MQL = INQ × <r1> % · Deals = MQL × <r2> % · Deals = revenue ÷ <deal size> · MQL = deals ÷ <r2> % · Revenue = deals × <deal size> · Choose a model to calculate · The model's INQ→MQL rate is 0, so this can't be worked out. · The model's MQL→deal rate is 0, so this can't be worked out. · The model's average deal size is 0, so this can't be worked out. |
| Published banner (edit) | This plan is published. Changes are saved straight away and recorded in the activity log. |
| Form buttons | **Save** · **Cancel** |
| Toasts | Plan '<name>' created — add campaigns next. · Plan '<name>' saved · Plan '<name>' published · Plan '<name>' unpublished — it's a Draft again · Plan '<new name>' created as a copy of '<source name>' · Plan '<name>' moved to the recycle bin (Undo; from 001) |
| Consequences dialog | title "Change the model?" (model only) / "Save these changes?" · "<n> activity has a type the new model doesn't allow:" / "<n> activities have a type the new model doesn't allow:" · "They keep their type but need an allowed type before they can be submitted. Goals will be recalculated with the new model." · "<unit> attributes will be hidden on the <levels>. Their values are kept and come back if you add <unit> again." · "<unit> attributes will be shown on the <levels>." (levels: "plan", "plan and its activities", "plan, its campaigns and its activities", …) · **Change model** / **Save changes** · **Keep editing** |
| Plan header | Plans › <name> · PLAN · Owner: <name> · No owner yet · Year <year> · Business unit: <labels> · No business unit yet · Created by <name> on <date>, last changed by <name> on <date> · **Edit** · **Publish** · **Unpublish** · More: Copy · View history · Delete · (inactive) |
| Summary cards (plan) | Plan budget — tip "The money this plan may spend in its year." · Planned cost — "The planned cost of every activity in the plan, added up." · Remaining budget — "Plan budget minus the budgets given to campaigns. It can go below zero." · Budget used — "Planned cost divided by the plan budget." · Activities approved — "Activities that are Approved, In progress or Complete, out of all activities in the plan." · details: No plan budget set · Nothing planned yet · Over-allocated by <amount> · Plan budget is 0 · No activities yet · In progress · All approved |
| Plan tree | Plan tree · The plan and everything in it · Search name or ID in this plan · **Expand all** · **Collapse all** · columns Name · Owner · Dates · Status · Budget · Planned cost · Budget used · "1 Jan – 31 Dec <year>" · Remaining <amount> · Over-allocated by <amount> · No budget set · Expand <name> · Collapse <name> · No campaigns yet — a campaign is a theme that runs for most of the year. · Nothing in this plan matches '<text>' |
| Completeness | All filled in · 1 field needed · <n> fields needed (001) · 1 item below needs attention · <n> items below need attention · "<Type> '<name>' — <n> field(s) needed" |
| Details | Details · Amounts in <currency> · Basics · Budget · Shell attributes · — · Yes · No · (inactive) |
| Goals | Goals · The plan's goal beside what its activities plan · Stage · Plan goal · Planned in activities · INQ · MQL · Revenue · <p> % of the <stage> goal planned · No goal set yet — set a goal amount in Edit. |
| Publish blocked | Publish needs 1 field · Publish needs <n> fields · Fill these in, then publish again: · Plan · <Type> '<name>' · **Close** |
| Unpublish dialog | Unpublish plan '<name>'? · Why are you unpublishing this plan? · The plan returns to Draft. Your reason is recorded in the activity log. · **Unpublish plan** · **Cancel** |
| Copy dialog | Copy plan '<name>' · Name of the copy · Year · Same year (<year>) · Next year (<year>) · Plans can only be created up to <year>. · Include campaigns, initiatives, activities and line items · <counts> · This plan has nothing inside yet. · Every date moves one year later. · **Copy plan** · **Cancel** |
| Copy notes (API) | Default finance budget cleared — it belonged to <year>. |
| Delete dialog | Delete plan '<name>'? · It holds <counts>. · It holds nothing yet. · This plan is published. · 1 of its activities is approved / <n> of its activities are approved; if the plan is restored later from the recycle bin, they return as Created. · It moves to the recycle bin; you can restore it. · **Cancel** · **Delete plan** |
| Counts | 1 campaign · <n> campaigns · 1 initiative · <n> initiatives · 1 activity · <n> activities · 1 line item · <n> line items (joined "a, b, c and d") |
| Not found | Plan not found · This plan no longer exists — someone may have deleted it. · **Go to Plans** |
| API validation | Enter a plan name. · Use 120 characters or fewer. · There is already a plan called '<name>'. Choose another name. · Choose a year from <first> to <last>. · Choose a model. · Choose a model from the list. · '<model>' is inactive — choose an active model. · Choose teams from the list. · '<team>' is inactive — choose an active team. · Choose an owner from the list. · <name> is inactive — choose someone else. · Choose teams first — the owner must belong to one of the plan's teams. · <person> isn't in any of the chosen teams — choose another owner or clear the field. · Use 2,000 characters or fewer. · Enter the plan budget in USD. · Enter an amount, for example 1200000 or 1,200,000.00. · Use at most 2 decimals. · Plan budget can't be negative. · Choose a finance budget from the list. · Choose a finance budget of <year>. · '<budget>' is inactive — choose an active finance budget. · Choose how line items are spread. · Choose INQ, MQL or Revenue. · Enter a number, for example 250000. · Enter a whole number. · The goal can't be negative. · Reload the page and try again. · Enter a reason. · Use 500 characters or fewer. · Copy into <year> or <year + 1>. · Plans can only be created up to <year>. · Choose all or myTeams. · Choose draft or published. · Choose a year such as 2026. |
| API other | This plan no longer exists — someone may have deleted it. · Not saved — <person> changed this plan (001) · Publish needs 1 field · Publish needs <n> fields |
| Restore notes (kind) | Can't undo — there is now another plan called '<name>'. Restore it from the recycle bin instead. · Renamed to '<new name>' because the name was taken. · Returned as Draft — publish it again when it's ready. |
| Log details | from '<source name>' (<source ID>) · from '<source name>' (<source ID>), with <counts> |

## Tests

### Go service tests (`backend/internal/plans/*_test.go`, package `plans`, fakes of Settings, Directory, Level; fixed clock 2026-03-12 09:00 UTC)

| Rule | Test |
|---|---|
| Defaults: year, years, active teams, owner only when eligible, single active model | `TestDefaults_FR006` |
| Only name, year, model block saving; every problem reported at once | `TestCreate_OnlyNameYearModelRequired_FR010`, `TestCreate_AllProblemsAtOnce_FR010` |
| Name ≤ 120, unique ignoring case and spaces; a deleted plan's name is free | `TestCreate_NameRules_FR010`, `TestCreate_DeletedNameReusable_EdgeCase` |
| Description ≤ 2,000; budget USD, ≥ 0, 2 decimals; goal ≥ 0, whole for INQ/MQL | `TestCreate_DescriptionBudgetGoalChecks_FR010` |
| Year range current − 1 … + 5 | `TestCreate_YearRange_FR006` |
| Only active models; finance budget of the plan year | `TestCreate_ActiveModelAndYearFinanceBudget_FR007` |
| Owner disabled without teams; owner must be in a chosen team (R1) | `TestCreate_OwnerNeedsTeams_FR007`, `TestCreate_OwnerMustBeInChosenTeam_FR021` |
| Attribute problems from Settings appear with the others | `TestCreate_AttributeProblemsCollected_FR008` |
| New plan is Draft with P-1001…, logged `created` | `TestCreate_DraftWithIDAndLog_FR010_FR031` |
| Two creates with one name: the second refused | `TestCreate_SameNameTwice_EdgeCase` |
| Waterfall forwards, backwards from MQL and Revenue, half-up rounding, zero rate and zero deal size | `goals_test.go`: `TestWaterfall_FromINQ_FR009`, `TestWaterfall_FromMQL_FR009`, `TestWaterfall_FromRevenue_FR009`, `TestWaterfall_RoundsHalfUp_FR009`, `TestWaterfall_ZeroRateGivesReason_EdgeCase` |
| Goal share planned (60 %), null without goal | `TestPlannedShare_FR016` |
| Remaining budget negative (1,200,000 − 1,300,000 = −100,000), over-allocated amount | `figures_test.go`: `TestFigures_RemainingBudgetNegative_FR012` |
| Budget used 80 %, over 100 % flagged, no budget / zero budget → null | `TestFigures_BudgetUsed_FR012` |
| Activities approved and approval progress | `TestFigures_ApprovalProgress_FR012` |
| Display currency conversion of every amount (EUR 0.92), half up | `TestFigures_DisplayCurrency_FR015` |
| Completeness: own fields in order, ineligible owner counts, items below listed | `TestCompleteness_OwnFieldsAndItemsBelow_FR017` |
| Inactive model/team/owner/option unchanged are kept on update | `TestUpdate_InactiveValuesKept_EdgeCase` |
| Missing version; out-of-date version names who and when; nothing saved | `TestUpdate_MissingVersion_FR023`, `TestUpdate_VersionConflict_FR023` |
| No change → no version, no log | `TestUpdate_NoChangeNoLog_FR031` |
| Published stays Published; each changed field logged with old and new readable values, attributes one change each | `TestUpdate_PublishedStaysPublishedAndLogsFields_FR022` |
| Owner left the teams: refused on save until fixed or cleared | `TestUpdate_OwnerNotInTeams_FR021` |
| Hidden attribute values kept | `TestUpdate_HiddenAttributeValuesKept_FR020` |
| Model change lists disallowed activities from levels; BU change lists hidden/shown groups | `TestChangeEffects_DisallowedActivities_FR019`, `TestChangeEffects_AttributeGroups_FR020` |
| List: scope, status, year, search (name or ID, any case), order, total, summary over scope + year, years | `TestList_FiltersSortAndSummary_FR001_FR003`, `TestList_MyTeams_FR004`, `TestList_SearchNameOrID_FR003` |
| List asks each level once for 200 plans | `TestList_200PlansOneFactsCallPerLevel_SC009` |
| Publish: complete Draft → Published + log; missing fields grouped (plan, then campaign items); warnings and activities never block; Published → no change | `status_test.go`: `TestPublish_CompleteDraft_FR024`, `TestPublish_ListsMissingGrouped_FR024`, `TestPublish_WarningsDoNotBlock_FR024`, `TestPublish_AlreadyPublished_NoChange` |
| Unpublish: reason 1–500, back to Draft, reason in the log | `TestUnpublish_ReasonRules_FR025`, `TestUnpublish_ReturnsToDraftAndLogsReason_FR025` |
| Copy name "(copy)", "(copy 2)", shortened to 120 | `copy_test.go`: `TestCopyName_Suffixes_FR026`, `TestCopyName_ShortenedTo120_EdgeCase` |
| Next year only up to current + 5 | `TestCopyOptions_NextYearAvailability_FR028` |
| Copy without contents: Draft, new ID, all fields, creator = copier, one `copied` entry | `TestCopy_PlanOnly_FR027` |
| Copy with contents: levels called in order with ID map and years; all or nothing when a level fails | `TestCopy_WithContents_FR027`, `TestCopy_AllOrNothing_FR027` |
| Next-year copy: year + 1, finance budget cleared with note | `TestCopy_NextYear_FR028` |
| ShiftDate: +1 year, 29 Feb → 28 Feb, "" stays "" | `TestShiftDate_FR028` |
| Delete: plan moves to the bin with its contents and is logged; Undo exact | `kinds_test.go`: `TestDelete_MovesToBin_FR029`, `TestUndo_RestoresExactly_FR030` |
| Undo with a taken name refused; Restore renames and returns as Draft | `TestUndo_NameTaken_EdgeCase`, `TestRestore_RenamesAndReturnsDraft_R14` |
| Describe gives readable labels and texts | `TestDescribe_ReadableLog_FR031` |
| Every action logged with the actor | `TestEveryChangeLogged_FR031` |
| Another person may edit, publish, unpublish, copy and delete a plan they did not create | `TestNoRoleChecks_FR032` |

### Handler and contract tests (`handler_test.go`, package `plans_test`, `apitest` with the seed)

One test per operation, every request and response checked against the merged contract:
`TestPlanDefaults_FR006`, `TestListPlans_SeedOrderSummaryAndFilters_FR001_FR003` (3 plans, 2 Draft, 1
Published; Draft + 2026 → FY2026 Mobility China, total 3; `q=mobility`, `q=p-1002`; invalid status → 400),
`TestCreatePlan_201AndValidation_FR010`, `TestGetPlan_ViewAnd404_FR011`, `TestUpdatePlan_200_409_404_FR018_FR023`,
`TestChangeEffects_FR019_FR020`, `TestPublish_NotReadyThenPublished_FR024` (FY2026 Mobility China → 409
`not_ready` with group "Plan" › Business unit), `TestUnpublish_FR025`, `TestCopyOptions_FR026`,
`TestCopy_201_FR027_FR028`, `TestDeletePlan_AndUndo_FR029_FR030` (via `/api/recycle-bin/{id}/undo`),
`TestPlansOperations_WithoutSession_FR005` (401 everywhere). `app_test.go` `TestRoutesMatchContract`
covers registration. `seed_test.go` `TestSeed_Plans`. `app/plans_deps_test.go` checks the adapters with the
seed (models, teams, display currency USD).

### Web component tests (Vitest, `mockApi`)

| File | Proves |
|---|---|
| `PlansPage.test.tsx` | cards with every fact (FR-002), summary cards (FR-001), pills, year, search, chips, count "1 of 3" (US2#2, FR-003), My teams hint (FR-004), both empty states (US2#5), error and loading, table layout, filters from the address and remembered per person, actions menu items |
| `PlanCard.test.tsx` / `PlanTable.test.tsx` | budget bar red above 100 % and "No budget set", approvals "n of m", completeness link to the edit field |
| `PlanForm.test.tsx` | section and field order, markings as words, default hints (US1#1, FR-005, FR-006), owner disabled with reason and R1 message (US1#4, FR-007, FR-021), live goals 250,000 → 5,000 and 2,500,000.00 "Calculated" (US1#5, FR-009), SMP shown with the reason (FR-008), finance budget cleared on year change (FR-007), year lock on edit (FR-018), changed-defaults hint (FR-022), all API problems at once + focus summary (US1#3, FR-010) |
| `PlanCreatePage.test.tsx` | defaults loaded, save → plan page + toast (US1#2), unsaved-changes guard |
| `PlanEditPage.test.tsx` | conflict message keeps values (FR-023), model-change dialog with links (US4#2, FR-019), BU dialog (US4#3, FR-020), published banner (US4#5), `?focus=` focuses the field, not found |
| `PlanPage.test.tsx` | header facts and actions (US3#1, FR-011), negative remaining budget in red (US3#2), budget used 80 % and red above 100 %, no budget "—" (US3#3), completeness "1 field needed" linking to the field (US3#4, FR-017), goals and "60 % of the INQ goal planned" (US3#5, FR-016), details "—" / Calculated / (inactive) (FR-015), View history link (FR-031), loading, error, not found |
| `PlanTree.test.tsx` | plan row cells, empty text, search keeps ancestors and opens the path, no-match text, Expand all / Collapse all (FR-013, FR-014) |
| `dialogs/*.test.tsx` | publish success toast and blocked dialog grouped with links (US5#1–2), unpublish reason rules (US5#3), copy options, next-year unavailable reason, counts, toasts (US6#1–2), delete dialog sentences, Cancel focused, Undo toast (US6#3–4), consequences texts |
| `planText.test.ts`, `recordLinks.test.ts`, `listPreferences.test.ts` | words and counts joined, links for plan fields, storage per person with failures ignored |
| `components/TreeTable/*.test.tsx`, `filterTree.test.ts`, `ProgressBar`, `LevelTag`, `FormDialog`, `FilterChips`, `SummaryCard` (detail), `CompletenessChip` (itemsBelow) | keyboard expand/collapse (→, ←, ↑, ↓, Home, End, Enter), ARIA, search filter, rendering |
| `api/errors.test.ts` | `not_ready` → kind `notReady` with `missing` |
| `app/navigation.test.ts`, `app/router.test.tsx` | Plans no longer Coming soon; the four routes and titles |

### End-to-end (`frontend/e2e/plans.spec.ts`, Playwright + axe, as `smoke.spec.ts`)

| Test | Journey |
|---|---|
| US1 create (SC-001) | As Priya: Plans → **New plan** → name (unique) → model Lubricants 1.0 → business unit Lubricants → **Save** (≤ 6 steps) → plan page shows Draft and "All filled in"; toast |
| US2 find | Draft + 2026 → FY2026 Mobility China visible, chips and "of" count; search "mobility" and "P-1001"; My teams as Mei Lin shows the hint; Clear filters |
| US3 glance | Open FY2026 Lubricants Europe: header facts, five cards, plan tree row, Details, Goals match the seed |
| US4 edit | Edit an own plan: year locked text, change budget and teams (owner R1 message), save → values; published plan stays Published |
| US5 publish | FY2026 Mobility China → **Publish** → "Publish needs 1 field" (Plan › Business unit) without changing it; own plan without business unit → link → fill → publish → toast → unpublish with a reason → Draft |
| US6 copy and delete | Copy FY2026 Lubricants Europe into 2027 (unique name) → Draft 2027 page; delete it from its page → dialog sentences → list + toast → **Undo** → back with the same ID |
| Accessibility | axe (WCAG 2.2 AA) on the list (cards and table), the create form with its error summary, the plan page, the copy, unpublish and delete dialogs |

Tests use their own plans (unique names) and never assert global counts, except reading the seed in US2/US3
without changing it (001's isolation rules).

## Requirement traceability

| FR | Implemented in | Tested by |
|---|---|---|
| FR-001 | `Service.List` summary; `PlansPage` header and cards | `TestList_FiltersSortAndSummary…`, `TestListPlans_…`, `PlansPage.test.tsx`, e2e US2 |
| FR-002 | `PlanSummary`; `PlanCard`, `PlanTable` | `PlanCard.test.tsx`, `PlanTable.test.tsx`, `PlansPage.test.tsx` |
| FR-003 | `List` filters/sort/years; toolbar, chips, `listPreferences` | `TestList_…_FR003`, `PlansPage.test.tsx`, `listPreferences.test.ts`, e2e US2 |
| FR-004 | `List` scope myTeams; hint | `TestList_MyTeams_FR004`, `PlansPage.test.tsx`, e2e US2 |
| FR-005, FR-006 | `Defaults`; `PlanForm` | `TestDefaults_FR006`, `TestCreate_YearRange_FR006`, `PlanForm.test.tsx`, e2e US1 |
| FR-007 | validation; `PlanForm` owner, model, finance budget | `TestCreate_ActiveModel…_FR007`, `TestCreate_OwnerNeedsTeams_FR007`, `PlanForm.test.tsx` |
| FR-008 | `Settings.CheckPlanAttributes`; `AttributeFields` | `TestCreate_AttributeProblemsCollected_FR008`, `PlanForm.test.tsx` |
| FR-009 | waterfall; `GoalFields` | `goals_test.go`, `PlanForm.test.tsx` |
| FR-010 | `Create`, validation | `TestCreate_*_FR010`, `TestCreatePlan_201AndValidation_FR010`, `PlanForm.test.tsx`, e2e US1 |
| FR-011 | `Get`; `PlanHeader` | `TestGetPlan_ViewAnd404_FR011`, `PlanPage.test.tsx`, e2e US3 |
| FR-012 | `figures.go`; `PlanSummaryCards` | `figures_test.go`, `PlanPage.test.tsx` |
| FR-013, FR-014 | `PlanTree`, `TreeTable`, `filterTree` | `PlanTree.test.tsx`, TreeTable tests, e2e US3 |
| FR-015 | conversion; `PlanDetails` | `TestFigures_DisplayCurrency_FR015`, `PlanPage.test.tsx` |
| FR-016 | `goals.go`; `PlanGoals` | `TestPlannedShare_FR016`, `PlanPage.test.tsx` |
| FR-017 | completeness; CompletenessChip `itemsBelow` | `TestCompleteness_…_FR017`, `PlanPage.test.tsx` |
| FR-018 | `Update` (no year); edit page lock | `TestUpdatePlan_…_FR018…`, `PlanForm.test.tsx`, e2e US4 |
| FR-019 | `ChangeEffects`, `Level.ActivitiesWithTypeNotAllowed`; dialog | `TestChangeEffects_DisallowedActivities_FR019`, `PlanEditPage.test.tsx` |
| FR-020 | `ChangeEffects`, `Settings.AttributeGroupChanges` | `TestChangeEffects_AttributeGroups_FR020`, `TestUpdate_HiddenAttributeValuesKept_FR020`, `PlanEditPage.test.tsx` |
| FR-021 | R1 check, `ownerProblem` | `TestCreate_OwnerMustBeInChosenTeam_FR021`, `TestUpdate_OwnerNotInTeams_FR021`, `PlanForm.test.tsx` |
| FR-022 | `Update` keeps status; banner and hint | `TestUpdate_PublishedStaysPublished…_FR022`, `PlanEditPage.test.tsx`, e2e US4 |
| FR-023 | `CheckVersion`; conflict alert | `TestUpdate_VersionConflict_FR023`, `PlanEditPage.test.tsx` |
| FR-024 | `Publish`, `not_ready` | `status_test.go`, `TestPublish_NotReadyThenPublished_FR024`, dialog tests, e2e US5 |
| FR-025 | `Unpublish` | `TestUnpublish_*_FR025`, dialog tests, e2e US5 |
| FR-026 – FR-028 | `CopyOptions`, `Copy`, `ShiftDate`, `Level.Copy` | `copy_test.go`, `TestCopy_201…`, `CopyPlanDialog.test.tsx`, e2e US6 |
| FR-029, FR-030 | `Delete`, kind, `useUndoableDelete` | `kinds_test.go`, `TestDeletePlan_AndUndo…`, `DeletePlanDialog.test.tsx`, e2e US6 |
| FR-031 | `history.Log` everywhere; View history link | `TestEveryChangeLogged_FR031`, `TestDescribe_ReadableLog_FR031`, `PlanPage.test.tsx` |
| FR-032 | no permission checks | `TestNoRoleChecks_FR032` |

## Dependencies

### Uses from 001 (exists)

Go: `record.Meta` (`NewMeta`, `Touch`, `CheckVersion`), `errs` (`Problems`, `Validation`, `NotFound`,
`NameTaken`, `RecordRef`), `history.Service` (`Log`, `Diff`, `MoveToBin`, `Register`), `history.Kind`,
`history.DeleteResult`, `history.BinEntry.View`, `ids.Generator.Next`, `store.Transactor`, `store.Get/List/Put/Delete`,
`web.ReadJSON/WriteJSON/WriteError/Router/Wrap`, `actor`, `clock`, `apitest`. Endpoints: `POST
/api/recycle-bin/{entryId}/undo`. Web: the components and hooks in *Shared components*, `api`/`call`,
`ApiError`, `conflictMessage`, `renderWithProviders`/`renderRoutes`/`mockApi`/`findMessages`, e2e
`signInAs`, `uniqueName`, `expectNoAxeViolations`.

### Expects from other features (by name; adapters in `internal/app` absorb Go naming differences)

| From | What |
|---|---|
| 002 (Go) | `platform/money`: `money.Amount` (exact, JSON decimal string "1200000.00"), `money.Parse(string) (Amount, error)`, `Amount.String()`, `Add`, `Sub`, `Neg`, `Cmp`, `Sign`, `IsZero`, `money.Money{Amount, Currency}` (JSON `{amount, currency}`), a thousands-grouped text for the log (`Amount.Grouped()` → "1,200,000.00"); the settings service functions behind `plans.Settings` (models with rates, deal size, allowed types and active flag; plan-level attributes with markings, conditions, dependencies; checking and completeness of attribute values; identification of the **Business unit** attribute; attribute groups by business-unit option at every level; finance budgets with a stable ID, year, name and active flag; attribute and option labels); the **waterfall function** forwards and backwards (algorithm above); a "Used by" registration that calls `plans.Service.References` / `ReferencesFromSnapshot` |
| 002 (API schemas) | `Money`, `AttributeValue`, `OptionRef` `{id, label, active}`, `ModelSummary` `{id, name, active, inqToMqlRate, mqlToDealRate, averageDealSize: Money}`, `FinanceBudgetRef` `{id, year, name, active}`; endpoints `GET /api/models`, `GET /api/attributes?level=plan`, `GET /api/finance-budgets?year=` (for the form) |
| 002 (web) | Breadcrumb, SearchField, PillFilter, SelectField, MultiChoiceField, TextAreaField, YesNoField, MoneyField, NumberField, AttributeFields (`level`, `values`, `onChange`, `businessUnit`, `errors`; input ids `attributes.<id>`), AttributeValueList; hooks `useModels()`, `usePlanAttributes()`, `useFinanceBudgets(year)`; TS `calculateWaterfall(model, stage, amount)` (`frontend/src/lib/waterfall.ts`) with the same vectors as Go |
| 003 (Go) | the teams/people service functions behind `plans.Directory` (team by ID with active flag and member IDs; the teams of a person; person by ID with active flag; the display currency preference, with 002's rate); a "Used by" registration (teams, owners) calling `References` |
| 003 (API schemas / web) | `PersonRef` `{id, name, active}`, `TeamRef` `{id, name, active}`; `GET /api/teams` (with members); `PersonPicker` (single, `teamIds` restriction mode, `value`, `onChange`, `disabledReason`, `error`, `marking`), `TeamPills`, `useTeams()`; `CurrentUser` with the display currency |
| 005–007 | implement and register `plans.Level` (campaign, initiative, activity, lineItem) and their `history.Kind`s with parents down from `plan`; fill `usePlanTreeNodes`, `toolbarActions`, `footer`, child routes and `RECORD_LINKS` entries |

### Provides to later features

| To | What |
|---|---|
| 005–007 | `plans.Level`, `Facts`, `AttentionItem`, `CopyJob`, `RegisterLevel`, `ShiftDate`, `Service.Lookup`, kind name `"plan"`; API schemas `MissingField`, `AttentionItem`, `Completeness`, `ReadinessGroup`, `SpreadMethod`, `GoalStage`; the `not_ready` error; web: `TreeTable`, `filterTree`, `ProgressBar`, `LevelTag`, `FormDialog`, `FilterChips`, CompletenessChip `itemsBelow`, SummaryCard `detail`, `usePlan`, the `['plan', id]` key, `PlanTree` extension props, `RECORD_LINKS`, `planText` count wording |
| 002, 003 | `Service.References`, `ReferencesFromSnapshot` for "Used by n plans" (models, teams, owners, finance budgets, attribute options) |
| 006 | `Plan.modelId`, `attributes` and the business unit (through `Lookup` + 002), `defaultSpread`, `defaultFinanceBudgetId`, `year` for 007 |
| 009, 010 | `GET /api/plans` (scope, year) and `Plan`/`PlanSummary` figures; `/plans?status=&year=&scope=` addresses |
| 011 | the `plan` kind (restore rules), log entries with readable changes; the History card slot |
| 013 | `bcpir` and the plan page section slot |

## Complexity Tracking

| Choice | Why needed | Simpler alternative rejected because |
|---|---|---|
| `plans.Level` registry | Plan figures, completeness, publish check, copy and delete counts need data owned by 005–007, which import `plans` | Importing them from `plans` creates import cycles; asking each level per plan makes the list slow (SC-009) |
| `plans.Settings` / `plans.Directory` interfaces + adapters in `internal/app` | 002/003 must count plan usage (cycle risk) and are designed in parallel; fakes make rules testable alone | Importing `settings`/`teams` directly couples 004's code to names not yet fixed and blocks unit tests on 002/003 internals |
| `change-effects` read-only operation | FR-019/FR-020 need server data (activities below, attribute groups) before saving | A confirmation flag on PATCH turns one save into a two-step protocol with partial-failure cases |
| Shared `Error` gains `not_ready` + `missing` | FR-024 lists every missing field grouped by record | A 200 answer with `published: false` would break 001's "every failure is an Error" rule |
| Five new shared components and two extensions | Each is needed by at least two features (table *Shared components*) | Feature-local copies would drift apart in 005–010 |
| Whole-form PATCH | Clearable fields without tri-state decoding | Partial PATCH needs a generic "absent vs null" type for every nullable field |

## Consolidation (2026-10-19, binding)

The Plan-module consolidation (the cross-feature analyze step) settled the points below. They and the build guide,
[docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md), win where this plan, its data model or its tasks
say otherwise (D-200).

- **Contract (D-192).** The fragment is merged into `backend/api/openapi.yaml`; the merge task (T001) is done. The
  feature's operations are listed in `notYetServed` in `backend/internal/app/app_test.go`; the backend removes each one in
  the change that serves it. Agents do not change the contract; a problem with it is reported.
- **Decisions.** This plan's proposed decisions are numbered D-072–D-089 (its row on Shell attribute values is D-048, recorded with 002) in `docs/sdd/decision-log.md` (labels kept in the
  table above); the consolidation decisions are D-189–D-207.

- **Shell attribute values (D-048, D-193).** The field is `attributes` (not `attributeValues`) in the contract, Go JSON and field ids
  (`attributes.<attributeId>`); this plan, data model, tasks and fragment were updated.
- **Contract (D-193).** 004 owns `ModelSummary`, `FinanceBudgetRef {id, year, name}` (finance budgets have no status) and `OptionRef`;
  `MissingField` is 002's (`reason` optional). `Error.code` `not_ready` and `Error.missing` are in the merged contract; Go adds
  `errs.CodeNotReady`, `errs.NotReady(message, missing)` and `Error.Missing` once (006 and 008 reuse them).
- **Settings reads.** Use 002's endpoints `GET /api/shell-attribute-forms/plan`, `GET /api/models`, `GET /api/finance-budgets?year=`
  (no `GET /api/attributes`) and 002's mapping table (*How later features read Settings*).
- **Dates (D-198).** Use `platform/date` (`date.Date`, `AddYears`); no `ShiftDate` on strings.
- **Shared web parts to build (D-197).** `TreeTable` (+ `filterTree`), `ProgressBar`, `LevelTag`, `FormDialog`, `FilterChips` as
  planned, **plus** `PillTabs` (tabs for 008 and 010), `FormSection` and `FormFooter` (full-page form layout and sticky footer, used
  by 006 and 007), the `SummaryCard` `detail` prop and the `CompletenessChip` `itemsBelow` prop, which takes a list **or** a count
  (D-199). Use 002's `SegmentedFilter`, `ChoiceField`, `CheckboxField`, `DecimalField`, `MoneyField`, `TextAreaField`,
  `ShellAttributeFields`/`ShellAttributeValues` and 003's `PersonPicker`, `TeamPills` in place of the draft names in *Shared
  components*.
- **Record links (D-194).** `RECORD_LINKS` lives in `frontend/src/lib/recordLinks.ts` as `recordHref(type, id, planId?)`; 005-007
  add their entries; 009-011 use it.
- **Levels and copies (D-195).** `plans.Level.Copy` gets a `CopyJob` with the year shift and the old-to-new IDs of the level above
  (005's *Alignment* section); levels register top-down: campaign, initiative (005), activity (006), lineItem (007).
- **Usage.** `Service.References` / `ReferencesFromSnapshot` give each plan's name and recycle-bin state as well (003's `PlanUse`
  and 002's `settings.Reference` need them).
