# Implementation Plan: Planning and Currency Extensions

**Branch**: `016-planning-and-currency-extensions` | **Date**: 2026-10-20 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/016-planning-and-currency-extensions/spec.md`

**Design artifacts**: [data-model.md](data-model.md) · [contracts/openapi.yaml](contracts/openapi.yaml) (a
fragment with new operations and schemas plus `x-amendments` to existing schemas, merged into
`backend/api/openapi.yaml`) · [tasks.md](tasks.md) · build guide
[docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md) (its conventions hold). No separate
research.md: the decisions are in *Proposed decisions*; no new dependency is added.

## Summary

Closes five MVP rows of Shell's BRD (BUS.BUD.050, BUS.PLN.020, BUS.PLN.040, BUS.BUD.020, BUS.UX.020) by
extending 002, 004–007 and 010:

1. **Dated exchange rates** with a simulated **GSAP daily rate import** behind a replaceable rate-feed seam;
   the current rate is the latest rate on or before today; every converted amount shows the rate used.
2. **Plan currency**: budgets and planned costs typed in a local currency per plan, shown with the reader's
   display currency (USD by default) beside; cross-plan pages add up in USD as today.
3. **Budget by month** for plans, campaigns and initiatives, with a phased-vs-planned card and a comparison
   in All work's Budget & cost.
4. **Split activity** into two, moving line items and planned cost, with history.
5. **Plan type** (Operational, Strategic, Cross-market) and **multi-year views** (year range on Plans, up to
   five years in All work with Yearly and Quarterly periods).
6. **Shell-attribute filters** and a **Breakdown** view in All work, and a "Planned cost by" card on the plan
   page.
7. **Field help** for every built-in field and calculated value, maintained in Settings and shown as ⓘ tips.

Technical approach: no new runtime part, package or dependency. New code lives in the existing packages
(`platform/money`, `rollup`, `settings`, `plans`, `campaigns`, `activities`, `lineitems`, `allwork`, `app`)
and web folders; stored records gain a few fields read with defaults for older data; every amount is still
calculated on read (D-041) with one new exact function, `money.Convert`.

## Technical Context

**Language/Version**: Go 1.27 (API); TypeScript 5.9 strict + React 19.3 (web app) — as 001

**Primary Dependencies**: none new (`encoding/csv` from the standard library reads the rate file)

**Storage**: local JSON file store; new tables `exchange_rates`, `rate_imports`, `field_help`

**Testing**: `go test` (unit, handler + contract with `apitest`), Vitest (`mockApi`), Playwright + axe

**Target Platform**: developer laptop, `127.0.0.1` only

**Project Type**: web application (`backend/` Go API, `frontend/` React)

**Performance Goals**: All work over a 5-year range (25,000 activities) answers its first page within 010
SC-004's time; a 20,000-row rate file is reviewed in under 10 seconds

**Constraints**: exact decimals (`int64` cents, `math/big` for products); no float; no external calls

**Scale/Scope**: 21 new operations, 9 changed operations, about 20 new and 25 amended schemas

## Constitution Check

| Principle | Check | Result |
|---|---|---|
| I Spec first | Every behaviour is an FR of spec.md, each traced to a BRD ID; amended FRs of 002–010 are named | Pass |
| II Plain language, configuration is data | Shell vocabulary kept (level names fixed, FR-044); rates and help texts are Settings data. **GSAP**: "GSAP integration is out of scope" — the product owner decided on 2026-10-20 to *simulate* GSAP rates with a file import and a feed seam; nothing connects to GSAP | **Exception** — see Complexity Tracking; constitution PATCH proposed: "Out of scope for now: live GSAP integration (simulated file imports are in scope)" |
| III Simple, friendly | One pattern per thing: rate history uses the Settings CRUD; phasing uses the line-item spread's editor ideas; split previews before confirming; all problems at once | Pass |
| IV Layered, server is source of truth | All conversions, phasing, breakdown and split rules in Go; the web app shows amounts and rates it receives; contract first | Pass |
| V Tested rules | Every calculation in data-model §3 has table tests; every FR has a task and a test (traceability below) | Pass |
| VI Safe by default | Money exact with currency; every change logged with the actor; access by 014's permissions (D-301); the uploaded file is size-limited and parsed, never executed | Pass |
| VII Small | No new package or dependency; derived values never stored; the rate feed is one interface with one implementation (justified: the PO asked for a seam for the real feed) | Pass |

## Proposed decisions

Numbered by the consolidation step: D-258–D-275 in docs/sdd/decision-log.md; "reopens" names the decisions they replace or narrow.

| # | Proposed decision | Why | Reopens |
|---|---|---|---|
| D-258 (016-A) | Exchange rates are dated records (`exchange_rates`, one per currency and day, source manual or GSAP import); a currency's current rate is the latest on or before today (UTC); conversions still happen on read with the current rates. | BUS.BUD.050 rate management and daily GSAP rates; keeps D-041 | D-027, D-141 (rate source), 002 Assumptions |
| D-259 (016-B) | GSAP rates are **simulated**: a CSV file import (review, then import) through `settings.RateFeed`, one implementation `GSAPRateFile`; a later automatic feed calls `ApplyRateRows` with the same rules. Format in data-model §2.4. | PO decision 2026-10-20; GSAP frozen until ~Mar 2027 | D-008 (rates only) |
| D-260 (016-C) | Each plan has a **plan currency** (default USD); plan, campaign and initiative budgets, phasing and activity planned costs are kept in it; line items keep their own currency; goals and model deal sizes stay USD. | "Plan in local currency and USD" | D-010, D-079, domain model §4 |
| D-261 (016-D) | Inside a plan, amounts are answered in the plan currency (the existing USD fields become plan-currency fields; Money carries the code) and in the reader's display currency (the existing `display` blocks of D-220, now converted from the plan currency). Cross-plan pages convert to USD before adding. | Reuses D-220's two-block shape; USD "alongside" by default | D-061, D-213, D-220 (wording) |
| D-262 (016-E) | `money.Convert(a, from, to)` = a × to ÷ from, half up, once; `ConvertParts` generalises `ConvertSpread`. Unassigned balance and spread grid are worked out in the plan currency. | One rounding per conversion; ToUSD/FromUSD unchanged | D-024, D-213 |
| D-263 (016-F) | The rate used is answered by `GET /api/exchange-rates/current`; the web app shows it in tips beside converted amounts; exports list the rates on "About this export". No `ratesUsed` field on every answer. | Few contract changes; rates change at most daily | — |
| D-264 (016-G) | Plan currency is locked once any campaign or initiative budget or activity planned cost exists; campaigns are not moved or copied between plans of different currencies. | No silent conversions of approved amounts | D-033 (adds a refusal reason) |
| D-265 (016-H) | Budget phasing: method none · evenly · custom over the 12 months of the plan's year; only custom months stored; saved through `PUT …/budget-phasing` per level, not the record forms; a custom phasing that does not add up is a needed-to-publish problem. | Ongoing updates without reopening the whole form; consistent with spreads | — |
| D-266 (016-I) | Phased vs planned (`GET /api/plans/{planId}/budget-by-month`) is served by package `allwork`, which already gathers every record and spread; it shares the period code with Budget & cost. | One implementation of "planned cost by month" | — |
| D-267 (016-J) | Split activity stays inside the plan; whole line items move and keep their IDs (`LineItem.activityId` changes only by Split); goals stay on the original; the new activity stores `splitFromId`; logged with existing actions (changed "split", created, moved). No undo. | BUS.PLN.040; GSAP keys line items by ID | 007 data model ("never changes") |
| D-268 (016-K) | Plan type is a built-in field (`operational`, `strategic`, `crossMarket`), default operational; the Strategic / AOP flag stays. | Distinct BRD planning modes; reversible | — |
| D-269 (016-L) | Plans list: `yearFrom`/`yearTo` replace `year` (kept as an alias); All work: range of at most 5 years, period Yearly added; saved views read old `year` as a one-year range. | BUS.PLN.020 multi-year | D-017 (views only; plans stay one year) |
| D-270 (016-M) | All work gets Shell-attribute filters (with Not filled in) and a Breakdown view (one or two groupings, rules in data-model §3.5). | BUS.BUD.020 | D-207, Q-009 |
| D-271 (016-N) | Field help: catalogue of keys and defaults in Go (`settings/fieldhelp_catalogue.go`); only edited texts stored (`field_help`, id = key); reset deletes the row; history kind `fieldHelp` logged, never binned; web reads all help once per page load through `useFieldHelp()`. | BUS.UX.020; new fields get defaults without data changes | — |
| D-272 (016-O) | Exchange rates use the recycle bin (kind `exchangeRate`); rate imports are not deletable. | Standard CRUD (002 FR-004) | — |
| D-273 (016-P) | No new error codes: locks and refusals answer `validation_failed` with field problems; conflicts `version_conflict`; restores `cannot_restore`. | Keep the error vocabulary small | — |
| D-274 (016-Q) | Demo data adds P-1004 (GBP, 2027, Cross-market) and 30 days of simulated rates ending with today's seeded rates, appended after existing records. | Existing IDs, totals and tests unchanged | D-209 note on seed IDs |
| D-275 (016-R) | `settings.RatesOn(ctx, d)` is the one way to convert at a past or future date; 015 (financial transactions) would use it for posting-date rates, a later option (D-303), instead of reading `exchange_rates` itself. Rate CRUD and import are guarded by 014's `exchangeRates.manage` once roles are enforced; Field help by 014's Settings administration permission. | One owner per rule across the parallel features 014, 015, 017 | — |

## Project Structure

### Documentation (this feature)

```text
specs/016-planning-and-currency-extensions/
├── spec.md
├── plan.md              # this file
├── data-model.md
├── contracts/openapi.yaml
└── tasks.md
```

### Source code (new and changed files)

```text
backend/
├── api/openapi.yaml                         # merge of contracts/openapi.yaml (T001)
├── internal/platform/money/convert.go       # NEW Convert, ConvertParts, Rates.Convert
├── internal/rollup/phasing.go               # NEW Phasing, PhasingMethod, PhasingMonths, PhasingProblem, Quarter helpers
├── internal/settings/
│   ├── model.go                             # + ExchangeRate, RateSource, RateImport, FieldHelpEntry; Currency legacy fields
│   ├── repository.go, store_json.go         # + tables exchange_rates, rate_imports, field_help
│   ├── rates.go                             # NEW current rate, RatesOn, CurrentRates, rate CRUD, impact, MigrateRates
│   ├── ratefeed.go                          # NEW RateFeed, FeedRow, GSAPRateFile (CSV reader)
│   ├── rates_import.go                      # NEW CheckRateImport, ApplyRateImport, ApplyRateRows, ImportSample
│   ├── fieldhelp_catalogue.go               # NEW the catalogue of data-model §6
│   ├── fieldhelp.go                         # NEW ListFieldHelp, UpdateFieldHelp
│   ├── service_currencies.go, readers.go, usage.go, kinds.go, handler.go, views.go   # changed
├── internal/plans/ (model.go, validate.go, service.go, copy.go, status.go, handler.go, figures.go, list.go)
├── internal/campaigns/ (model.go, validate.go, service.go, moves.go, handler.go, display.go, phasing.go NEW)
├── internal/activities/ (model.go, service.go, form.go, display.go, handler.go, split.go NEW, deps.go)
├── internal/lineitems/ (balance.go, grid.go, display.go, service.go, move.go NEW, handler.go)
├── internal/allwork/ (model.go, query.go, source.go, periods.go, tree.go, rollup.go, columns.go, export.go,
│                      handler.go, views.go, filters_attr.go NEW, breakdown.go NEW, budget_by_month.go NEW)
├── internal/app/ (app.go, allwork_source.go, home_sources.go, approvals_adapter.go, lineitems.go,
│                  activities.go, campaign_adapters.go, references.go, currency.go NEW, app_test.go)
└── seed/ (settings.go, plans.go, campaigns.go, activities.go, lineitems.go + tests)

frontend/src/
├── api/schema.d.ts (generated), api/ratesQueries.ts NEW, api/fieldHelp.ts NEW
├── components/AmountPair/ NEW, RateTip/ NEW, YearRangeField/ NEW (each with test + gallery entry), index.ts
├── features/settings/currencies/ (RateHistorySection, RateDialog, ImportRatesPage, RecentImports NEW; CurrencyPage changed)
├── features/settings/field-help/ NEW (FieldHelpPage, FieldHelpPanel), sections.ts, routes
├── features/budget-phasing/ NEW (BudgetByMonthCard, PhasingEditorPanel, phasingMath.ts)
├── features/plans/ (PlanForm, PlanHeader, PlanSummaryCards, PlanCard, PlanTable, PlansPage, PlanPage, PlannedCostByCard NEW)
├── features/campaigns/ (RecordForm, BudgetCost, RecordDetails, campaignRows, Destinations)
├── features/activities/ (ActivityForm, ActivityHeader, ActivitySections, useActivityActions, SplitActivityDialog NEW, SplitLinks NEW)
├── features/line-items/ (CostFields, LineItemsSection, UnassignedBalanceRow, SpreadGrid, LineItemPage)
└── features/all-work/ (FilterBar, FilterMenu, AttributeFilterMenu NEW, BudgetView, CalendarView, BreakdownView NEW,
                       ColumnChooser, columnHelp.ts NEW, SummaryCards, viewState, allWorkStore, allWorkApi, ExportMenu)
```

**Structure decision**: the existing layout; one new web feature folder (`budget-phasing`, used by plans and
campaigns) and three shared components.

## Backend design

### `platform/money` — `convert.go`

```go
// Convert converts a between two currencies given their rates per 1 USD: a × to ÷ from, half up to the
// cent (016-E). Convert(a, r, One) == ToUSD(a, r); Convert(a, One, r) == FromUSD(a, r); equal rates → a.
func Convert(a Amount, from, to Rate) Amount
// ConvertParts converts parts so they add up to Convert(Sum(parts), from, to): every part but the last with
// Convert, the last gets the rest. ConvertSpread(parts, r) == ConvertParts(parts, r, One).
func ConvertParts(parts []Amount, from, to Rate) []Amount
// Convert converts m into the currency code with rs (ErrUnknownCurrency when a rate is missing).
func (rs Rates) Convert(m Money, code string) (Money, error)
```

`ToUSD`, `FromUSD`, `ConvertSpread` are rewritten on top of these, with their existing tests unchanged.

### `rollup` — `phasing.go`

```go
type PhasingMethod string // "none", "evenly", "custom"
type PhasingMonth struct { Month string `json:"month"`; Amount money.Amount `json:"amount"` } // "2027-03"
type Phasing struct { Method PhasingMethod `json:"method"`; Months []PhasingMonth `json:"months"` }

func (p Phasing) MonthAmounts(budget *money.Amount, year int) []money.Amount // 12 or nil (data-model §3.3)
func CheckPhasing(p Phasing, budget *money.Amount, year int) []errs.FieldError // save rules of §3.3
func PhasingProblem(p Phasing, budget *money.Amount, year int, currency string) *MissingProblem // nil when fine
func ShiftPhasing(p Phasing, years int) Phasing // copies into another year (FR-025)
```

### `settings` — rates, import, field help

- **Repository**: `GetRate`, `RatesOf(code)` (date order), `PutRate`, `DeleteRate`; `PutRateImport`,
  `ListRateImports(limit)`; `GetFieldHelp(key)`, `ListFieldHelp`, `PutFieldHelp`, `DeleteFieldHelp`. IDs
  `XR`, `RI` from `ids.Generator`.
- **Readers** (`readers.go`): `Rates(ctx)` now builds the map from `exchange_rates` with today from the
  service clock (signature unchanged — every caller keeps working); new `RatesOn(ctx, d date.Date)`,
  `CurrentRates(ctx) ([]RateInfo, error)` (`RateInfo{Currency, Rate, Date, Source, ImportFileName}`), and
  `CurrentRate(ctx, code) (RateInfo, bool, error)`.
- **Rate CRUD** (`rates.go`): `ListRates(ctx, a, code)`, `AddRate(ctx, a, code, RateInput{Date, Rate})`,
  `UpdateRate(ctx, a, id, RateUpdate{Rate, Version})`, `DeleteRate(ctx, a, id)` (`history.MoveToBin`),
  `RateImpact(ctx, a, code, RateInput)` / `UpdateRateImpact(ctx, a, id, RateUpdate)`: when the change alters
  the current rate, the line-item warning of 002 plus a second `amountsRecalculated` warning for plans
  planned in the currency (plans' `ReferenceSource` gives `Currency` and the plan's total budget in the plan
  currency); a future-dated rate answers no warning and the note of data-model §3.2. `CreateCurrency` adds the
  first rate; `UpdateCurrency` with `rate` upserts today's rate. Messages: *User-visible messages*.
- **Migration**: `MigrateRates(ctx)` (data-model §2.2), called by `app.New` after the store opens.
- **Rate feed** (`ratefeed.go`):

  ```go
  // RateFeed turns what one source delivers into rows; the file import is the only feed now (FR-008).
  type RateFeed interface {
      Name() string                                          // "gsapDailyRatesFile"
      Read(fileName string, content []byte) ([]FeedRow, error) // error: the file can't be read (header, size)
  }
  type FeedRow struct { Row int; RateType, From, To, ValidFrom, Rate string }
  type GSAPRateFile struct{} // CSV of data-model §2.4
  ```
- **Import** (`rates_import.go`): `CheckRateImport(ctx, a, feed, upload)` → `RateImportCheck` (every row
  classified, data-model §2.4); `ApplyRateImport(ctx, a, feed, upload)` → re-classifies inside one `Write`,
  writes the rows to add and replace (`source gsapImport`, `importId`), the `RateImport` record and one log
  entry; `ApplyRateRows(ctx, a, feedName, rows)` is the shared core a future automatic feed calls;
  `ImportSample(ctx)` → file name and CSV bytes. Upload: `{fileName, content}` (base64) read with
  `web.ReadJSONUpTo` (3 MB request for a 2 MB file).
- **Field help** (`fieldhelp.go`): `ListFieldHelp(ctx, a)` → every catalogue entry with text, `isDefault`,
  version, updated by/at, plus `shellAttributesWithoutHelp` (active attributes with empty help);
  `UpdateFieldHelp(ctx, a, key, FieldHelpUpdate{Text *string; Version int})` — `nil` text resets.
- **Usage** (`usage.go`): currencies count plans (`UsagePart` kind `plan`, "Plan currency of 2 plans") from
  the plans' `ReferenceSource` (`Currency` = plan currency); the delete lock follows.
- **History kinds**: `exchangeRate` (under `currency`; `PrepareRestore` refuses a taken date), `rateImport`
  (log only), `fieldHelp` (log only).

### `plans`

- Model: `Currency`, `PlanType`, `BudgetPhasing` (data-model §4.1); normalisation on read.
- Create/update: `currency` (active or kept, `CanChoose`), `planType` (enum), lock of FR-012 using the level
  registry: `plans.Level.AddFacts` gains `AmountsSet int` per plan (campaigns: budgets set; initiatives:
  budgets set; activities: planned costs set). Budget removal resets phasing.
- `SetBudgetPhasing(ctx, a, planID, PhasingUpdate{Method, Months, Version})` → `rollup.CheckPhasing`, save,
  log "Budget by month".
- Figures: `Plan.figures` stays the display-currency figures (converted from the plan currency); new
  `Plan.planFigures` = the same schema in the plan currency; `Plan.currencyLock` (reason or null).
- Completeness and publish: add `rollup.PhasingProblem` for the plan; campaigns/initiatives add theirs
  through their completeness (005).
- List: `yearFrom`, `yearTo` (inclusive; `year` = both), `type` (repeatable); summary over the range;
  `PlanSummary` adds `planType`, `currency`, `planBudget`, `planPlannedCost` (plan currency); the existing
  amounts stay display currency (converted per plan, data-model §3.1).
- Copy: currency, type and phasing copied; next year `rollup.ShiftPhasing(p, 1)`.
- Plan defaults (`GET /api/plan-defaults`): add `currency: "USD"`, `planType: "operational"`.

### `campaigns`

- Budget currency check (= plan currency) on create/update; `Campaign.budget`, `budgetAndCost`,
  `parentRemainingBudget`, `PlanCampaigns.totals` are in the plan currency; `display` / `displayTotals`
  converted from the plan currency with `Rates.Convert`.
- `SetBudgetPhasing` for campaigns and initiatives (`phasing.go`), completeness adds `PhasingProblem`.
- Destinations: a campaign target plan with another currency → unavailable with the reason of FR-017; copy
  likewise. Moves and copies keep phasing.

### `activities`

- Planned cost currency check; `display` from the plan currency; the activity form's
  `ActivityTypeOption.defaultPlannedCost` is answered converted into the plan currency, with
  `defaultPlannedCostUsd` (the Settings value) beside it for the note.
- `SplitFromID`; `Activity.splitFrom` (`SplitLink` or null) and `Activity.splitInto` (`SplitLink[]`);
  `ActivitySummary.splitFromId`.
- **Split** (`split.go`): `SplitOptions(ctx, a, id)` (destinations as `Destinations(…, purpose=move)` plus the
  line items with their plan-currency values, the suggested name, `needsRestart`); `PreviewSplit` and `Split`
  share `planSplit(ctx, a, id, in) (splitPlan, errs.Problems)`; `PreviewSplit` returns the figures without
  writing; `Split` runs in one `Write`: create the new activity (new ID), `LineItems.MoveToActivity(ctx, a,
  ids, newID)` (new seam method, implemented by 007), update the original (planned cost, `RestartApproval`
  when needed), log (data-model §7). Rules and messages: data-model §3.6.
- `activities.LineItems` adds `MoveToActivity(ctx, a, lineItemIDs []string, toActivityID string) error`.

### `lineitems`

- `ActivityFacts` gains `PlanCurrency string`; conversions with `money.Convert` (data-model §3.1):
  `LineItem.plannedCostPlan`; `Balance` and `Grid` in the plan currency; their display variants converted
  from the plan currency; `roundingHint` in the plan currency; `LineItemCurrency` adds `rateDate`,
  `rateSource`.
- `MoveToActivity` (`move.go`): changes `activityId`, touches each line item, logs `moved`.

### `allwork`

- Query: `yearFrom`, `yearTo` (≤ 5 years; `year` alias), `planType`, `planCurrency`, `attr` (repeatable,
  `<attributeId>:<value>,<value>`), `comparePhased`, `period=year`, `view=breakdown`, `by`, `thenBy`;
  refused values answer `validation_failed` with the parameter's name (D-219).
- `Source.Year` per year of the range; tree and periods across years (`periodsOf(tree, from, to)`; keys
  `2026-03`, `Q1-2026`, `2026`, `other`, `noDates`).
- `filters_attr.go`: attribute filter matching on `Node.Attrs` with D-113's level rules.
- `breakdown.go`: data-model §3.5; `GET /api/all-work/breakdown` (and export of the Breakdown view).
- Compare: `AllWorkRow.phasedBudget` and `overPeriods` for containers when `comparePhased`.
- `budget_by_month.go`: `BudgetByMonth(ctx, a, planID)` → `GET /api/plans/{planId}/budget-by-month`
  (data-model §3.4), using the plan's nodes and plan-currency rates (`Source.PlanRates`).
- Columns: Year, Plan type, Plan currency, Plan budget (plan currency), Planned cost (plan currency).
- Export: "About this export" adds the range, new filters, the breakdown and the rates used
  (`CurrentRates`); file name "MAPP All work – Breakdown – 2026–2027 – 20 Oct 2026".
- Saved views: read old states as data-model §5.2.

### `internal/app`

- `currency.go`: `usdOf(rates, m *money.Money) *money.Amount` and `planRates(ctx, planID)` helpers for
  adapters.
- Adapters converting plan-currency amounts to USD: `allwork_source.go` (Node fields of data-model §5.1),
  `home_sources.go` (plan figures → USD per plan), `approvals_adapter.go` (planned cost and unassigned balance
  from the plan currency to the display currency), `lineitems.go` (`PlanCurrency` in facts),
  `references.go` (plans' `Currency`).
- Wiring in `app.New`: `MigrateRates`, the `GSAPRateFile` feed, the new routes, history kinds.
- `app_test.go`: the new operations start in `notYetServed`; each lane removes its own.

## API operations

| Operation | Purpose | FR |
|---|---|---|
| `GET /api/currencies/{code}/rates` | Rate history and current rate | FR-001, FR-003 |
| `POST /api/currencies/{code}/rates` | Add a dated rate | FR-003 |
| `POST /api/currencies/{code}/rates/impact` | Impact of adding a rate | FR-005 |
| `PATCH /api/exchange-rates/{rateId}` | Edit a rate | FR-003 |
| `POST /api/exchange-rates/{rateId}/impact` | Impact of editing a rate | FR-005 |
| `DELETE /api/exchange-rates/{rateId}` | Delete a rate (recycle bin) | FR-003 |
| `GET /api/exchange-rates/current` | Current rates with date and source | FR-002, FR-009 |
| `POST /api/exchange-rates/import-check` | Review a GSAP rate file | FR-006 |
| `POST /api/exchange-rates/import` | Import it | FR-006, FR-007 |
| `GET /api/exchange-rates/imports` | Recent imports | FR-007 |
| `GET /api/exchange-rates/import-sample` | Sample file of today's rates | FR-006 |
| `GET /api/field-help` | Every help entry | FR-045, FR-047, FR-048 |
| `PATCH /api/field-help/{helpKey}` | Edit or reset a help text | FR-046 |
| `PUT /api/plans/{planId}/budget-phasing` | Plan phasing | FR-020 – FR-023 |
| `PUT /api/campaigns/{campaignId}/budget-phasing` | Campaign phasing | FR-020 – FR-023 |
| `PUT /api/initiatives/{initiativeId}/budget-phasing` | Initiative phasing | FR-020 – FR-023 |
| `GET /api/plans/{planId}/budget-by-month` | Phased vs planned | FR-024 |
| `GET /api/activities/{activityId}/split-options` | Split dialog data | FR-026 |
| `POST /api/activities/{activityId}/split-preview` | Both activities after the split | FR-027 |
| `POST /api/activities/{activityId}/split` | Split | FR-028 – FR-030 |
| `GET /api/all-work/breakdown` | Breakdown view and plan card | FR-040 – FR-042 |
| changed `GET /api/plans` | `yearFrom`, `yearTo`, `type` | FR-033 |
| changed `POST /api/plans`, `PATCH /api/plans/{planId}` | `currency`, `planType` | FR-011, FR-012, FR-032 |
| changed `GET /api/plan-defaults` | defaults of currency and type | FR-011, FR-032 |
| changed `GET /api/all-work`, `GET /api/all-work/export` | range, new filters, `period=year`, `comparePhased`, Breakdown export | FR-034 – FR-039, FR-043 |
| changed `GET /api/all-work/options` | filterable attributes, plan types, currencies | FR-037, FR-039 |
| changed `POST /api/currencies`, `PATCH /api/currencies/{code}` | the rate becomes today's dated rate | FR-003, FR-004 |

## Frontend design

### Shared parts (owner: this feature)

| Part | Where | Contract |
|---|---|---|
| `AmountPair` | `components/AmountPair/` | `{ primary: Money; secondary?: Money \| null; size?: 'sm' \| 'md' }` — primary in full, secondary muted "≈ 6,645.57 USD" only when its currency differs; the secondary carries a `RateTip` |
| `RateTip` | `components/RateTip/` | `{ currencies: string[] }` — reads `useCurrentRates()` and lists "1 USD = 0.79 GBP · rate of 19 Oct 2026 · GSAP import" per non-USD currency; keyboard focusable (InfoTip pattern) |
| `YearRangeField` | `components/YearRangeField/` | `{ years: number[]; from: number \| null; to: number \| null; maxSpan?: number; allowAll?: boolean; onChange }` — two selects "From" "To"; keeps from ≤ to; "All years" when `allowAll` |
| `useCurrentRates()` | `api/ratesQueries.ts` | `GET /api/exchange-rates/current`, staleTime 5 minutes, invalidated by rate saves and imports |
| `useFieldHelp()` | `api/fieldHelp.ts` | `GET /api/field-help` once per page load (staleTime 0 on forms, SC-008); `help(key): string \| undefined` (undefined when empty → no ⓘ) |
| `BudgetByMonthCard`, `PhasingEditorPanel`, `phasingMath.ts` | `features/budget-phasing/` | Card: `{ level: 'plan' \| 'campaign' \| 'initiative'; planId; recordId }`; panel opens from **Edit**; `phasingMath` (BigInt cents, feedback only): even split, quarter split, totals |

Every field of the forms and details of 004–007 passes `tip={help('<key>')}` to the field components
(`FieldShell.tip` already exists); summary cards and fixed tips (`PlanSummaryCards`, `BudgetCost`,
`RecordForm`, `CostFields`, `UnassignedBalanceRow`, `SpreadGrid`, `GoalsFields`, `PlanGoals`,
`ActivityHeader`, All work `SummaryCards`) read their text from `help()` instead of literals.

### Pages and changes

| Page | Change | FR |
|---|---|---|
| Settings (`sections.ts`) | Card **Field help** (`/settings/field-help`, icon `CircleHelp`, count = edited entries); Currencies description adds "rate history and GSAP rate import" | FR-045 |
| Currency details | Current rate with date and source; **Rate history** section (table, Add rate, row menu Edit/Delete); delete via `useUndoableDelete` | FR-001 – FR-005 |
| `/settings/currencies/import` | Upload (`.csv`, 2 MB), review table grouped by outcome with counts and row numbers, **Import n rates**, **Cancel**, **Download a sample file**; *Recent imports* on the Currencies list | FR-006, FR-007 |
| `/settings/field-help` | Search, level groups (PillTabs), rows label + text + chip; **Edit** opens `FormPanel` (TextAreaField 300, counter), **Reset to default** (ConfirmDialog); note with the Shell-attribute link (`/settings/shell-attributes?help=missing`) | FR-045, FR-046, FR-048 |
| Shell attributes list | "No help yet" chip; filter `help=missing` | FR-048 |
| Plan form | Basics: **Plan type** (SegmentedFilter-style radio group, ⓘ); Budget: **Plan currency** (ChoiceField of active currencies, lock note from `currencyLock`), budget `MoneyField` with the plan currency and the USD note | FR-011, FR-012, FR-032 |
| Plans list | **Years** (`YearRangeField`, allowAll), **Plan type** filter chips; cards and rows show the type chip and `AmountPair` | FR-014, FR-033 |
| Plan page | Header chips "Operational" / "Planned in GBP"; summary cards from `planFigures` with `figures` beside (`AmountPair`); **Budget by month** card; **Planned cost by** card (`GET /api/all-work/breakdown?plan=…&amountsIn=plan`) | FR-013, FR-024, FR-042 |
| Campaign / initiative panels and rows | Budget in the plan currency; `AmountPair` for budget, remaining, planned cost; **Budget by month** section; destination reasons from the API | FR-013, FR-017, FR-024 |
| Activity form | Planned cost `MoneyField` in the plan currency with "≈ X USD"; type default note | FR-011, FR-016 |
| Activity page | `AmountPair` in header; **Split activity** in More; split links under the header | FR-013, FR-026, FR-031 |
| Split dialog | `FormDialog`: name, initiative (`DestinationList`), line items (checkbox list with amounts), amount (`MoneyField`), preview of both activities (debounced `split-preview`), re-approval line when needed; **Split activity** / **Cancel** | FR-026 – FR-029 |
| Line items | Columns "Planned cost", "Planned cost (GBP)", display column; balance and grid in the plan currency; conversion line with rate dates | FR-015, FR-009 |
| All work | `YearRangeField` (max 5); Plan type, Plan currency, **More filters** (`AttributeFilterMenu`: attributes by level, option checkboxes, Not filled in); **Breakdown** tab (`BreakdownView`: two pickers, table with expandable groups, notes); Budget & cost **Yearly** and **Compare with phased budget**; Calendar year/quarter bands; column chooser ⓘ via `columnHelp.ts`; view-state migration | FR-034 – FR-043 |

## User-visible messages

| Situation | Text |
|---|---|
| Last current rate delete | EUR needs a rate dated today or earlier. |
| Rate date taken | EUR already has a rate on 19 Oct 2026. |
| Future rate impact | No conversion changes before 01 Jan 2027. |
| New currency's first rate in the future | A new currency needs a rate dated today or earlier. |
| Import header missing | The file needs the column 'valid_from'. |
| Import too large | The file is larger than 2 MB. Split it and import each part. |
| Import nothing to add | Nothing to import: every rate is already present or not used. |
| Import done (toast) | 89 exchange rates imported from 'gsap_rates_2026-10-20.csv'. |
| Plan amount in another currency | Enter the amount in GBP, the plan's currency. |
| Plan currency lock | The plan currency can't change once campaigns, initiatives or activities have amounts in it (4 amounts). |
| Campaign destination, other currency | Planned in EUR — this campaign's amounts are in GBP. |
| Type default conversion | Default 5,000.00 USD = 3,950.00 GBP at today's rate. |
| Phasing without budget | Set a budget before phasing it. |
| Phasing month outside year | Months must be in 2027. |
| Phasing completeness | Budget by month adds up to 590,000.00 GBP, not the budget 600,000.00 GBP. |
| Phasing running total | 100,000.00 of 600,000.00 phased — 500,000.00 left / 10,000.00 too much |
| Budget removed | Budget by month will be removed. |
| Copy from planned cost note | 12,000.00 GBP in 2028 and 3,000.00 GBP without dates are not copied. |
| Over in a period | Over by 14,000.00 GBP |
| Children not phased | 2 campaigns are not phased. |
| Split nothing chosen | Choose line items or an amount to move. |
| Split amount too high | Move at most 40,000.00 GBP, the activity's planned cost. |
| Split goals | Goals stay on 'UK launch event'. |
| Split re-approval | This activity will need approval again. |
| Split toast | 'UK launch event' split — 'Ireland roadshow' created. View |
| Split links | Split from 'UK launch event' (A-1012) · Split into 'Ireland roadshow' (A-1014) · "(deleted)" |
| Breakdown multi | Activities with several values count in each group; the total counts them once. |
| Breakdown line items | Includes 3,000.00 USD on line items of activities without a planned cost. |
| Year range too long | Choose at most 5 years. |
| Field help too long | Use 300 characters or fewer. |
| Field help reset confirm | Reset the help of 'Plan budget' to MAPP's text? |
| Attributes without help | 3 Shell attributes have no help yet — view them |

## Tests

- **Go, pure**: `TestConvert_*` (table: same rate, to USD, from USD, cross, half up, negative, big);
  `TestConvertParts_AddsUp`; `TestPhasing_Evenly_FR020`, `TestPhasing_CustomMismatch_FR022`,
  `TestShiftPhasing_FR025`; `TestCurrentRate_LatestOnOrBeforeToday_FR002`, `TestCurrentRate_FutureRate`;
  `TestGSAPRateFile_*_FR006` (every outcome row of data-model §2.4); `TestBreakdown_*_FR041` (single,
  multiple, split remainder, line-item attribute with unassigned, hidden, then-by, budget groups);
  `TestPlannedByMonth_PlanCurrency_FR024`; `TestSplit_Rules_FR026` … `FR028`.
- **Go, service**: rate CRUD, last-rate refusal, migration idempotence, import check = import counts, plan
  currency lock, budget currency refusal, campaign destination refusal, copy keeps currency/phasing, split
  with restart, field help version 0 conflict, usage counts plans.
- **Handler + contract** (`apitest`): every new and changed operation validated against the merged contract;
  seed facts: P-1004 figures (600,000.00 GBP; 759,493.67 USD), A-1012 unassigned 5,250.00 GBP.
- **Web (Vitest)**: AmountPair, RateTip, YearRangeField, phasingMath, PhasingEditorPanel, BudgetByMonthCard,
  SplitActivityDialog, FieldHelpPage, ImportRatesPage, AttributeFilterMenu, BreakdownView, viewState
  migration, every form shows its tip from `help()`.
- **End-to-end** (`frontend/e2e/planning-currency.spec.ts`, axe on each new screen): import the sample rate
  file; create a GBP plan with budget and phasing; campaign + activity + EUR line item amounts; split an
  activity; All work 2026–2027 Yearly, compare, attribute filter, Breakdown; edit and reset a field help text.

## Build order and lanes

Phase 2 (*Foundation*, one backend and one frontend agent) lands the contract, stored fields, money and
phasing functions, rate tables with the migration, seed additions and shared web parts. Then five lanes work
in parallel, each with a backend and a frontend agent, in different files:

| Lane | Stories | Backend files | Web folders |
|---|---|---|---|
| R Rates | US1 | `settings/rates*.go`, `ratefeed.go`, handler routes for rates | `features/settings/currencies/` |
| C Plan currency | US2 | `plans` (currency, figures, list amounts), `campaigns` (currency, display, destinations), `activities` (currency, display, form), `lineitems` (conversions), `app` adapters | plans/campaigns/activities/line-items amount displays and forms |
| P Phasing and split | US3, US4 | `plans`/`campaigns` phasing endpoints, `allwork/budget_by_month.go`, `activities/split.go`, `lineitems/move.go` | `features/budget-phasing/`, Split dialog |
| A All work and Plans list | US5, US6 | `plans` list range/type, `allwork` (range, filters, breakdown, compare, columns, export, views) | `features/all-work/`, `PlansPage`, `PlannedCostByCard` |
| H Field help | US7 | `settings/fieldhelp*.go` | `features/settings/field-help/`, `help()` in every form |

Shared files (`app.go`, `app_test.go`, `seed/*.go`, `router.tsx`, `components/index.ts`): add lines, never
rewrite others'. **Parallel features**: 014 (access control) also changes `internal/app/allwork_source.go`
(plan visibility filter) — 016 adds fields to the nodes it builds and never touches the filter; 015
(financial transactions) reads rates only through `settings.RatesOn` / `Rates` (016-R); 017 (approval rules)
does not overlap. Whichever feature merges second rebases onto the first and keeps both changes. Lane C and lane P both touch `plans/service.go` and `campaigns/service.go`: lane C owns the
create/update paths, lane P adds `SetBudgetPhasing` in new files `plans/phasing.go` and
`campaigns/phasing.go`.

## Requirement traceability

| FR | Tasks (tests first) |
|---|---|
| FR-001 – FR-005 | T008 – T010, T018, T019, T026, T027 |
| FR-006 – FR-008 | T021 – T025, T028 |
| FR-009 | T009, T010, T014, T015, T035, T043, T064 |
| FR-010 | T020 |
| FR-011 – FR-019 | T029 – T043 |
| FR-020 – FR-025 | T006, T007, T029 (copy), T044 – T053 |
| FR-026 – FR-031 | T054 – T060 |
| FR-032 – FR-038 | T029, T039, T040, T061 – T069 |
| FR-039 – FR-044 | T070 – T077 (FR-044: T072 "then by"; T092 glossary) |
| FR-045 – FR-048 | T078 – T088 |
| FR-049 | T018, T023, T044, T056, T079 (log assertions) |
| SC-001 – SC-009 | T048 (SC-004), T056 (SC-005), T072 (SC-007), T089, T095 |

## Complexity Tracking

| Exception | Why needed | Simpler alternative rejected because |
|---|---|---|
| Simulated GSAP rate import while the constitution keeps GSAP out of scope | Product owner decision 2026-10-20; BRD MVP BUS.BUD.050 asks for daily GSAP rates | Typing every daily rate by hand is not "consuming daily rates"; a live connection is impossible during the GSAP freeze |
| `RateFeed` interface with a single implementation | The PO asked for a seam so the real feed replaces the file without changing rules | Calling the CSV reader directly would tie every rule to the file format |

## Consolidation (2026-10-20, binding)

The consolidation of 014–017 settled the points below. They win where this plan, its data model, its tasks or its
fragment say otherwise (as D-200). Build order across the four features:
[docs/reports/brd-gap-closure-plan.md](../../docs/reports/brd-gap-closure-plan.md).

- **Decisions.** 016-A–016-R are D-258–D-275 in [decision-log.md](../../docs/sdd/decision-log.md); the
  consolidation decisions are D-298–D-309.
- **Contract (D-298).** The fragment and its `x-amendments` are merged into `backend/api/openapi.yaml`; T001–T003
  are done (21 operations in `notYetServed`). Differences from the fragment:
  - every property added to an existing schema is **optional** in the contract (required lists untouched); the
    API always sends it;
  - nothing is removed: `year` stays (required as today) in `AllWorkViewState`, `AllWorkResult` and
    `AllWorkOptions`, and the API answers it equal to `yearFrom`; a saved view sent with only `year` reads as a
    one-year range;
  - parameter `by` is optional in the contract; `GET /api/all-work/breakdown` and the Breakdown export answer 400
    `validation_failed` on `by` when it is missing;
  - `AllWorkBreakdownGroup.children` holds `AllWorkBreakdownSubgroup` (the same fields, no `children`), because the
    contract validator refuses recursive schemas;
  - the `describe` texts were merged with the details they replaced (rules, nulls, display-currency notes).
- **First web task of lane A (typecheck).** Since the merge, `npm run typecheck` fails in
  `frontend/src/features/all-work/viewState.ts` (`FILTER_LABELS` lacks `attributes`, `planTypes`,
  `planCurrencies`). Lane A adds the labels "Shell attributes", "Plan type" and "Plan currency" first.
- **Access (D-301).** FR-049 now follows 014: rate maintenance and import need `exchangeRates.manage`, field help
  `settings.manage`, phasing `plans.edit` (plan) or `campaigns.edit` (campaign, initiative), split
  `activities.edit`; refusals are 403. Route rules: the Table C additions in the 014 plan's Consolidation section.
  Pages that add up several plans (Plans list, All work, Breakdown) include only plans the person can see (014
  FR-007).
- **Money (D-303).** Two additions to this plan:
  - the plan currency is also locked while the plan has a global-budget allocation (014): `Plan.currencyLock` =
    "The plan currency can't change while the plan has an allocation from '<global budget>'." (checked before the
    amounts lock); the allocation is kept in the plan's currency, so `SetBudgetFromAllocation` (014) needs no
    conversion;
  - 015 converts transactions at the current rate; `RatesOn` is not used by 015 in this phase.
- **Placement (D-305).** Plan page: header (type and currency chips) → summary cards (`AmountPair`; 014's
  allocation note on the budget card) → 015's Financials → **Budget by month** → **Planned cost by** → plan tree →
  History. Campaign and initiative panels: … Budget and cost → **Budget by month** → 015's Financials → History.
  Activity page: split links under the header; Line items show 015's Committed and Invoiced after this feature's
  plan-currency columns. Settings cards: the existing ones, then 014's Roles and access, then **Field help**.
- **Shared web parts (as D-197).** This feature owns `AmountPair`, `RateTip`, `YearRangeField`,
  `useCurrentRates()` and `useFieldHelp()`; 014, 015 and 017 reuse them.
- **Log visibility (D-304).** `exchangeRate`, `rateImport` and `fieldHelp` entries need `activityLog.viewAll`;
  phasing and split entries follow their plan. Restoring an exchange rate from the bin needs `exchangeRates.manage`.
- **Demo data (D-306).** The rate history is written inside the Settings step; P-1004 and its records inside the
  planning steps, after 014's roles; P-1004 (2027) gets no global-budget allocation.
- **Build order (D-307).** The backend foundation (T004–T013) may run beside 014's foundation and rebases on it;
  the web foundation (T014–T017) waits for the UI foundation's shared components. Lanes R (rates) and H (field
  help) start once 014's backend foundation is merged; lanes C, P and A start after 014's US1–US2 backend. Lane A changes `internal/allwork` after 014's visibility filter and before 015's financial
  columns; lane C changes `plans` after 014.
- **Constitution (D-308).** The PATCH proposed here is part of the one 1.1.0 amendment written as the first task of
  the build.
