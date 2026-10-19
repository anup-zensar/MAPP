# Implementation Plan: Settings and Shell Attributes

**Branch**: `002-settings-and-shell-attributes` (built on `plan-module`) | **Date**: 2026-10-19 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/002-settings-and-shell-attributes/spec.md`

**Design artifacts**: [data-model.md](data-model.md) (records, fields, rules, seed) ·
[contracts/openapi.yaml](contracts/openapi.yaml) (fragment: 47 operations) · [tasks.md](tasks.md).
Conventions are feature 001's ([plan](../001-app-shell-and-sign-in/plan.md),
[research](../001-app-shell-and-sign-in/research.md), [data-model](../001-app-shell-and-sign-in/data-model.md));
this plan names only what is new.

## Summary

Feature 002 builds **Settings**: the page of section cards and five sections that follow the standard
CRUD pattern — **Models** (with the live goal waterfall), **Activity types**, **Shell attributes** with
their options, dependencies, business-unit conditions, order, preview and spreadsheet import,
**Currencies** with exchange rates, and **Finance budget items** grouped by budget. It also delivers what
features 003–008 build on:

- **`internal/platform/money`** — exact decimal money, exchange rates and percentages: parsing,
  formatting, conversion with the one current rate per currency, half-up rounding, even splits and
  spreads (D-010, D-024, D-027).
- **`internal/settings`** — the Settings records, their history kinds, and Go functions later features
  call: model and activity-type lookups, `Waterfall`, `Rates`, finance budget lookups, and the single
  implementation of Shell attribute values on forms (`CheckAttributes`, `MissingAttributes`, `Diff`,
  `DescribeAttribute`, FR-024 – FR-027). "Used by" counts come from **reference sources** that 003–007
  register, so Settings never imports them.
- **Web**: the Settings pages; new shared form controls (money, decimals, searchable choices, % split,
  check box, long text, breadcrumb, search, segmented filter, lock note) and **`ShellAttributeFields`**,
  the component that renders a level's Shell attributes on the preview and on the forms of 004–007.

The approach repeats 001's pattern (handler → service → repository; page → shared components → API
client) and keeps every rule in Go; the web app repeats a rule only for instant feedback (the worked
example, the attribute form's visibility and dependency rules), each proven equal by shared test cases.

## Technical Context

**Language/Version**: Go 1.27 (API); TypeScript 5.9 strict + React 19.3 (web app) — as 001

**Primary Dependencies**: none new. Go standard library only (`math/big` for exact products,
`encoding/csv`, `archive/zip` + `encoding/xml` to read `.xlsx`, `encoding/base64`). Web app: the libraries
of 001 (`radix-ui` Popover for the searchable choice, `@tanstack/react-query`, `react-router`,
`lucide-react`).

**Storage**: 001's JSON file store; new tables `models`, `activity_types`, `shell_attributes`,
`shell_attribute_options`, `currencies`, `finance_budgets`, `finance_budget_items`; new counter prefixes
`MOD`, `AT`, `SA`, `SO`, `FB`, `FBI`

**Testing**: Go `testing` with fixed clock and fake reference sources; `apitest` contract tests for every
operation; Vitest + Testing Library for every component and page; Playwright + axe for the main journeys
(`frontend/e2e/settings.spec.ts`)

**Target Platform**: as 001 (Windows 11 laptops, current Chrome and Edge)

**Project Type**: web application — React single-page app + Go REST API

**Performance Goals**: every Settings request answers in under 200 ms locally with the seed plus one
attribute of 5,000 options; option search answers in under 100 ms; the activity form definition with
5,000 options is about 0.5 MB and is fetched when a form opens (SC-008)

**Constraints**: servers on `127.0.0.1`; no personal data in logs; WCAG 2.1 AA including keyboard
reordering (SC-009); request bodies ≤ 1 MB (001), so import files are limited to 700 KB (base64 ≈ 950 KB)

**Scale/Scope**: 5 Settings sections + the Settings page; 9 web routes; 47 API operations; 7 record kinds;
11 new shared web components; seed of 5 currencies, 6 activity types, 2 models, 2 finance budgets with 6
items, 35 Shell attributes with 110 options

## Constitution Check

*GATE: checked before design and again after it.*

| Principle | How this plan complies | Result |
|---|---|---|
| I. Spec first | Every element traces to an FR, scenario, edge case or SC (traceability table). Nothing beyond the spec, except the shared pieces later features need, each tied to their FRs (FR-024 – FR-027 are this spec's own) | Pass |
| II. Shell business, plain language | Shell words only (Shell attribute, model, INQ, MQL, finance budget item); every text in *User-visible messages*; configuration is data managed on Settings screens (constitution II "business configuration is data") — the seed is only a starting point | Pass |
| III. Simple, friendly experience | Standard CRUD pattern on every section; details in side panels (attributes on a full page, FR-020); all problems at once; impact stated before saving; nothing greyed out without a reason (`Lock` reasons from the API); delete with Undo; in-use records offer **Set inactive**; keyboard reordering | Pass |
| IV. Simple, layered architecture | One business package `settings` (handler → service → repository) plus one platform package `money`; every rule in Go; storage through the `Repository` interface; usage reaches other packages only through registered functions (no import cycles, no new runtime part); OpenAPI fragment, generated web types | Pass |
| V. Tested business rules | One Go test per rule, named with its FR (*Tests*); money and waterfall tested with exact tables; a contract test per operation; a component test per page and component; Playwright journeys for US1–US4 (P1) and US5–US6 | Pass |
| VI. Safe by default | Actor on every service call; every change logged in the same transaction; money as exact decimals (never floats, in Go and TS); deletes go to the recycle bin; inputs validated by the API; import files parsed in memory, never written to disk or logged | Pass (1 exception, *Complexity Tracking*) |
| VII. Keep it small and readable | No new dependency; `.xlsx` read with the standard library (one small file); one generic reference-source hook instead of per-feature callbacks | Pass |

**Post-design re-check**: Pass. The one exception (options removed by a type change are not sent to the
recycle bin) is in *Complexity Tracking*.

## Decisions

The spec has no `[NEEDS CLARIFICATION]` marker. The defaults of the decision log apply: Q-003 (fictitious
option lists — seed in data-model §13), Q-005 (markings as in the field catalogue), D-004 (no roles:
FR-008), D-010 / D-027 (USD base, one current rate), D-014, D-029, D-034. New decisions, numbered at
consolidation (2026-10-19):

| Proposed | Decision | Why |
|---|---|---|
| **D-041** — Settings-derived values are computed on read | Features 003–010 never store values derived from Settings: line-item USD amounts, display conversions and model-calculated goals are calculated when read, with `settings.Rates` and `settings.Waterfall`. Typed goals ("Set later stages myself") are stored | FR-007, FR-011, FR-029, SC-008: a change applies everywhere at once with no recalculation job and no stale copies |
| **D-042** — "Used by" through reference sources | Features 003–007 register a `settings.ReferenceSource` listing their records (live and in the recycle bin) and the Settings records each uses; Settings computes every usage count, in-use check and impact statement from them | Settings cannot import plans or activities (they import Settings); one generic hook is simpler than one callback per question |
| **D-043** — In-use deletes answer `409 in_use` | New error code `in_use` ("'<name>' is in use (…) — set it inactive instead.") and `cannot_restore` (a restore that cannot happen, with the reason) | FR-004; the web app tells the cases apart by code |
| **D-044** — Impact is checked before saving | Each change that alters calculated values or completeness has a read-only `POST …/impact` (same body as the update) that validates and returns the warnings; the web app shows them in a confirmation and then saves. Statements count live records only | FR-011, FR-012, FR-021, FR-030, SC-005; keeps 001's update and error shapes unchanged |
| **D-045** — Settings record ids | Internal ids with prefixes `MOD`, `AT`, `SA`, `SO`, `FB`, `FBI` from 001's generator; a currency's id is its ISO code. Not shown as readable IDs | Spec Assumptions (no readable IDs); log entries need a stable `recordId` |
| **D-046** — Allowed activity types live on the model | The model stores `activityTypeIds`; editing allowed models from the type rewrites those models (logged on each). A type allowed by any model (live or in the bin) counts as in use | Domain model §6 puts the list on the model; keeps Undo exact and a model never without types |
| **D-047** — Finance budgets are records | A finance budget (year + name) is its own record, created with its first item, renamed once for all items, never deleted on its own in this phase; plans refer to it by id | FR-015: renaming one record makes items and plans follow without touching them |
| **D-048** — Attribute values live on the record | Plans, campaigns, initiatives, activities and line items keep Shell attribute values in an `attributes` field (attribute id → value, data-model §9) and log them attribute by attribute (`settings.Diff`) | One read gives a record with its values; snapshots in the recycle bin carry them; readable log entries |
| **D-049** — Options removed by a type change are deleted permanently | Before any record has a value, changing an attribute from a choice type to a non-choice type removes its options without the recycle bin; the log entry lists their labels | No record uses them (FR-021's lock); a restored option would not fit the new type |
| **D-050** — Attribute order | Each attribute has a section and a position; Move up/down works inside a section, Move section up/down moves a whole section; a dependent stays below its parent and Business unit stays first | FR-019, edge case "Region must stay below Cluster", FR-023; keyboard friendly (SC-009) |
| **D-051** — Option import on the server | The web app sends the file (≤ 700 KB, `.csv` or `.xlsx`) as base64 JSON twice — check, then add — and the API reads it with the standard library | FR-031; constitution IV (rules in Go), VII (no spreadsheet library) |
| **D-052** — Money in two decimals for every currency | All amounts, JPY included, have two decimals; rates have six; conversions round half-up per line item and totals add the rounded values | Domain model §4.3–4.4 ("to the cent"); what users see adds up |
| **D-053** — Limits | Average deal size ≤ 100,000,000.00 USD; rate ≤ 1,000,000; worked-example INQ ≤ 1,000,000,000; ≤ 5,000 options per attribute; number attributes ±999,999,999,999; text 120 / long text 2,000 characters | Keeps exact arithmetic in `int64` and forms fast; far above Shell's real values |
| **D-054** — Currency code taken on restore | A currency whose code was taken meanwhile is not restored: Undo `name_taken`, Restore `cannot_restore`; a finance budget item whose code was taken is restored without its code (Restore) | A code cannot take a " (restored)" suffix (feature 011 FR-022) |
| **D-055** — Business unit attribute | The plan-level Business unit attribute is marked `role = businessUnit` by the seed; it cannot be deleted, set inactive, moved, retyped or given a condition | FR-023 |

## Project Structure

### Documentation (this feature)

```text
specs/002-settings-and-shell-attributes/
├── spec.md · checklists/requirements.md
├── plan.md              # this file
├── data-model.md        # records, values, usage, kinds, log, seed
├── contracts/openapi.yaml   # fragment merged into backend/api/openapi.yaml
└── tasks.md
```

### Source code (new and changed files)

```text
backend/
├── api/openapi.yaml                       # CHANGED: + the fragment; Error.code + in_use, cannot_restore
├── seed/seed.go · seed_test.go            # CHANGED: addSettings (data-model §13), SettingsSeed
└── internal/
    ├── app/app.go · app_test.go           # CHANGED: wire settings (every environment), register kinds
    ├── platform/errs/errs.go · errs_test.go   # CHANGED: CodeInUse, CodeCannotRestore, InUse, CannotRestore
    ├── platform/money/                    # NEW
    │   ├── money.go        # Amount, Money, Input, parsing, formatting, arithmetic, SplitEvenly
    │   ├── rate.go         # Rate, Rates, ToUSD, FromUSD, Inverse, ConvertSpread
    │   ├── percent.go      # Percent, OfWhole, PercentSplitEvenly
    │   └── money_test.go · rate_test.go · percent_test.go
    ├── history/service.go · service_test.go   # CHANGED: SnapshotsInBin
    └── settings/                          # NEW
        ├── model.go            # stored types and enums (data-model §2–§8)
        ├── views.go            # API types (views, inputs, updates, pages, impact, usage)
        ├── values.go           # AttributeValues, value shapes, Diff
        ├── repository.go · store_json.go
        ├── service.go          # Service, NewService, AddReferenceSource, Overview, shared helpers
        ├── service_models.go · waterfall.go
        ├── service_activity_types.go
        ├── service_attributes.go · order.go
        ├── service_options.go
        ├── service_import.go · importfile.go
        ├── service_currencies.go
        ├── service_finance.go
        ├── usage.go            # Reference, ReferenceSource, usage index, impact statements
        ├── form.go             # Form, CheckAttributes, MissingAttributes, BusinessUnit, DescribeAttribute
        ├── readers.go          # Model, ActivityType, Currency, Rates, FinanceBudget(Item) lookups, CanChoose
        ├── kinds.go            # the 7 history kinds, Describe, PrepareRestore
        ├── handler.go
        └── *_test.go           # one per file above; handler_test.go is package settings_test

frontend/
├── e2e/settings.spec.ts                   # NEW
└── src/
    ├── app/navigation.ts · router.tsx (+ tests)   # CHANGED: Settings routes, leaves COMING_SOON_SECTIONS
    ├── api/errors.ts (+ test)             # CHANGED: kinds inUse, cannotRestore
    ├── api/settingsQueries.ts (+ test)    # NEW: read hooks and query keys other features reuse
    ├── lib/decimal.ts · waterfall.ts · text.ts (+ tests)   # NEW
    ├── lib/useUndoableDelete.ts (+ test)  # CHANGED: shows cannot_restore's message
    ├── components/                        # NEW: Breadcrumb, SearchField, SegmentedFilter, MoneyField,
    │                                      # DecimalField, CheckboxField, TextAreaField, ChoiceField,
    │                                      # SplitEditor, LockedNote, ShellAttributeFields (+ rules.ts,
    │                                      # ShellAttributeValues, ClearValuesDialog); CHANGED: TextField
    │                                      # (suggestions), index.ts
    ├── features/dev/ComponentGalleryPage.tsx (+ test)   # CHANGED: the new components
    └── features/settings/                 # NEW
        ├── SettingsPage.tsx · sections.ts
        ├── shared/  SettingDeleteDialog.tsx · ImpactDialog.tsx · UsageLine.tsx · MetaLine.tsx ·
        │            useListFilters.ts · useSettingStatus.ts · settingsNouns.ts · SettingsTable.module.css
        ├── models/  ModelsPage.tsx · ModelDetails.tsx · ModelForm.tsx · WaterfallExample.tsx · modelsApi.ts
        ├── activity-types/  ActivityTypesPage.tsx · ActivityTypeDetails.tsx · ActivityTypeForm.tsx · activityTypesApi.ts
        ├── shell-attributes/ ShellAttributesPage.tsx · ShellAttributePage.tsx · ShellAttributeEditorPage.tsx ·
        │            OptionsEditor.tsx · OptionsCard.tsx · OptionPanel.tsx · ImportOptionsDialog.tsx ·
        │            AttributePreview.tsx · importTemplate.ts · shellAttributesApi.ts
        ├── currencies/  CurrenciesPage.tsx · CurrencyDetails.tsx · CurrencyForm.tsx · currenciesApi.ts
        └── finance-budget-items/ FinanceBudgetItemsPage.tsx · FinanceBudgetItemDetails.tsx ·
                     FinanceBudgetItemForm.tsx · RenameBudgetDialog.tsx · financeBudgetsApi.ts
        (each .tsx with .module.css where it has styles, and .test.tsx)
```

**Structure decision**: 001's layout. The backend package is `internal/settings` (one business area,
constitution IV) split into files per record type; the web pages live in `features/settings/<section>/`.

## Backend design

### Package `money` (owner: 002; used by every feature)

Public API (names fixed; data-model §1 has the rules and examples):

```go
type Amount struct{ /* cents int64 */ }
func ParseAmount(s string) (Amount, error)          // ErrInvalidAmount, ErrOutOfRange
func FromCents(c int64) Amount
func (a Amount) Cents() int64
func (a Amount) String() string                     // "1234.50"
func (a Amount) Format() string                     // "1,234.50"
func (a Amount) Add(b Amount) Amount; Sub; Neg; Cmp(b) int; Sign() int; IsZero() bool
func (a Amount) MulInt(n int64) (Amount, error)     // ErrOutOfRange
func (a Amount) SplitEvenly(n int) []Amount         // n ≥ 1, a ≥ 0; rounded down, remainder last
func Sum(as ...Amount) Amount
// MarshalJSON / UnmarshalJSON: a JSON string ("1234.50"); UnmarshalJSON accepts 0–2 decimals

type Money struct { Amount Amount `json:"amount"`; Currency string `json:"currency"` }
func (m Money) Format() string                      // "1,234.50 EUR"
type Input struct { Amount string `json:"amount"`; Currency string `json:"currency"` }   // request bodies
func ParseInput(in Input) (Money, error)            // ErrInvalidAmount, ErrOutOfRange, ErrInvalidCurrency

type Rate struct{ /* micros int64 */ }
var One Rate
func ParseRate(s string) (Rate, error)              // ErrInvalidRate (format, ≤ 0), ErrOutOfRange (> 1,000,000)
func (r Rate) String() string; Format() string; Inverse() Rate; IsOne() bool
func ToUSD(local Amount, r Rate) Amount
func FromUSD(usd Amount, r Rate) Amount
func ConvertSpread(parts []Amount, r Rate) []Amount // D-024
type Rates map[string]Rate
func (rs Rates) ToUSD(m Money) (Amount, error)      // ErrUnknownCurrency
func (rs Rates) FromUSD(usd Amount, code string) (Money, error)

type Percent struct{ /* hundredths int64 */ }
var Hundred Percent
func ParsePercent(s string) (Percent, error)        // ErrInvalidPercent (format, > 3 integer digits, > 2 decimals)
func (p Percent) String() string                    // "2.00"
func (p Percent) Format() string                    // "2.00 %"
func (p Percent) OfWhole(n int64) int64             // half-up, math/big
func PercentSplitEvenly(n int) []Percent
func SumPercent(ps ...Percent) Percent
```

The package has no storage and no messages for users: callers turn its errors into their own field
messages. Rounding half-up (half away from zero for negatives) is implemented once, in an unexported
`roundDiv(num, den *big.Int) int64`.

### Package `settings`

**Constructors and wiring** (`internal/app/app.go`, every environment, before the demo-only samples):

| Item | Signature |
|---|---|
| Repository | `NewJSONRepository() Repository` — methods `GetModel/ListModels/PutModel/DeleteModel`, the same four for `ActivityType`, `Attribute`, `Option` (plus `OptionsOf(ctx, attributeID)`), `Currency` (key = code), `FinanceBudget`, `FinanceBudgetItem` (plus `ItemsOf(ctx, budgetID)`); `List…` sorted as 001's `store.List` and never nil |
| Service | `NewService(tx store.Transactor, repo Repository, ids *ids.Generator, hist *history.Service, now clock.Clock) *Service` |
| Kinds | `(*Service).Kinds() []history.Kind` in the order model, activityType, shellAttribute, shellAttributeOption, currency, financeBudget, financeBudgetItem (data-model §11) |
| References | `(*Service).AddReferenceSource(src ReferenceSource)` — called by `app.New` for each later feature's source (003–007) |
| Handler | `NewHandler(s *Service) *Handler`; `(*Handler).Routes(r *web.Router, signedIn web.Wrap)` — every route behind `signedIn` |

Every service method takes `(ctx, a actor.Actor, …)`, validates with `errs.Problems` (all problems at
once), works in one `s.tx.Write` (or `Read`), checks the version with `CheckVersion(expected, noun)` and
calls `Touch`, writes its log entries with `history.Log` in the same `Write`, deletes only with
`history.MoveToBin`, and returns API types from `views.go`. Text inputs are trimmed; lengths count Unicode
characters. Nouns for conflicts: "model", "activity type", "Shell attribute", "option", "currency",
"finance budget", "finance budget item".

### Shared rules (every record type)

| Operation | Rules | FR |
|---|---|---|
| `Create…` | validate all fields; unique name (`sameName` = `EqualFold` after trim); new id; `record.NewMeta`; `active = true`; one `created` entry | FR-002, FR-005, FR-006 |
| `Update…` | missing `version` or < 1 → `validation_failed` on `version` ("Reload the page and try again."); not found → `404` with the type's "no longer exists" text; validate the sent fields; `CheckVersion`; nothing changed → return the stored view, no version, no log; else `Touch`, one `changed` entry with `settings.Diff` changes | FR-006, FR-007 |
| `Set…Active` | `StatusUpdate{active, version}`; locks first (USD, Business unit, parent with active dependents, reactivating a dependent of an inactive parent); same value → no change; else `Touch` and one `statusChanged` entry, verb `deactivated` / `reactivated` | FR-004, FR-006 |
| `Delete…` | in one `Write`: usage index (below); in use → `409 in_use` with "'<name>' is in use (<summary, first letter lower case>) — set it inactive instead."; locked (USD, Business unit, last option of a choice attribute) → `409 in_use` with the lock reason; else `history.MoveToBin` (which logs `deleted`) | FR-003, FR-004 |
| Lists and gets | views carry `usage` (all) and `locks` (attributes, currencies, items) computed with one usage index per request | FR-003 |

### Models (`service_models.go`, `waterfall.go`)

| Method | Rules beyond the shared ones | FR / scenario |
|---|---|---|
| `ListModels(ctx, a) ([]ModelView, error)` | sorted by name ignoring case; each with `activityTypes` (SettingRef, by name) and `usage` | FR-002, FR-003 |
| `GetModel(ctx, a, id)` | `usage.records`: live plans using it (≤ 50, by name) | US1-3 |
| `CreateModel(ctx, a, ModelInput)` | data-model §2 validation; `activityTypeIds` de-duplicated; each must exist (else "An activity type you ticked no longer exists — reload the page and try again.") and be active | FR-009, US1-1, US1-5 |
| `UpdateModel(ctx, a, id, ModelUpdate)` | newly added types must be active, kept inactive ones allowed; at least one type; when rates or deal size changed and the model is used by live plans, the `changed` entry's details say "Used by 2 plans (31 activities): calculated goals recalculated." | FR-009, FR-011, US1-4 |
| `ModelImpact(ctx, a, id, ModelUpdate) (ChangeImpact, error)` | same validation; warnings (live records): `goalsRecalculated` when a rate or the deal size changes and live plans use it; one `activityTypeRemoved` per removed type used by live activities of the model's live plans | FR-011, FR-012, US1-4, US1-6 |
| `SetModelActive`, `DeleteModel` | shared rules; usage = plans | FR-004, US1-7 |
| `Waterfall(inq int64, m Model) (Goals, error)` | data-model §2.1; `ErrOutOfRange` passes through | FR-010, SC-006 |

### Activity types (`service_activity_types.go`)

| Method | Rules beyond the shared ones | FR / scenario |
|---|---|---|
| `ListActivityTypes`, `GetActivityType` | `models`: models whose `activityTypeIds` contain it; `usage`: activities + "Allowed in n models" | FR-003, US5-2 |
| `CreateActivityType(ctx, a, ActivityTypeInput{Name, DefaultPlannedCost *money.Input, ModelIDs})` | name ≤ 120, unique; default ≥ 0.00 USD; each model must exist; each listed model gains the type (model `Touch`, one `changed` entry on the model, field `activityTypeIds`) in the same `Write` | FR-013, US5-1 |
| `UpdateActivityType(ctx, a, id, ActivityTypeUpdate)` | `DefaultPlannedCost` is `OptionalMoney` (sent `null` removes it); `ModelIDs` (when sent) is the complete list: added models gain it, removed models lose it — refused when a model would keep no type ("'<model>' needs at least one activity type, so '<type>' can't be removed from it."); each changed model logged | FR-012, FR-013, US5-2 |
| `ActivityTypeImpact` | one `activityTypeRemoved` warning per removed model whose live plans have live activities of this type | FR-012 |
| `SetActivityTypeActive`, `DeleteActivityType` | usage = activities and models (live or in the bin) | FR-004, US5-5 |

### Shell attributes (`service_attributes.go`, `order.go`)

| Method | Rules beyond the shared ones | FR / scenario |
|---|---|---|
| `ListAttributes(ctx, a, level *Level)` | display order (data-model §4.2); each with `optionCount`, `activeOptionCount`, `parentAttribute`, `condition`, `usage`, `locks` | FR-002, FR-019 |
| `GetAttribute` | as above | FR-020 |
| `CreateAttribute(ctx, a, AttributeInput)` | label ≤ 80, unique in level; help ≤ 300; level, type, marking valid and combined as data-model §4 (yes/no optional; split only activity and line item; marking by level); section default "Shell attributes", ≤ 60; position last in section; choice types need ≥ 1 option, ≤ 5,000, labels ≤ 120 and unique ignoring case ("'TV' appears twice." once per clashing label); non-choice types take no options; `parentAttributeId`: a choice attribute of the same level, before it, and only for choice types; then every option needs ≥ 1 parent option of that attribute, each active ("Link '<option>' to at least one <parent> option." for **each** unlinked option); `conditionOptionIds`: options of the Business unit attribute (not on Business unit itself); all problems together; one `created` entry with details "with 4 options" | FR-016 – FR-019, FR-022, FR-023, US2-1, US2-2, US3-2 |
| `UpdateAttribute(ctx, a, id, AttributeUpdate)` | label, help, marking, section, condition as on create; type and level refused when locked (data-model §4.1) with the lock reason on the field; a new section → last in it; type from a choice type to a non-choice type removes its options permanently (details "Removed its 4 options: …"); `parentAttributeId` changed to an attribute → `optionLinks` must link every active option (each missing one listed); each option whose links changed gets `Touch` and a `changed` entry; changed to `""` → links removed the same way | FR-016 – FR-023, US2-4 |
| `AttributeImpact` | `becomesIncomplete`: marking from optional to needed and live records of the level, where the attribute is shown, have no value (count with the same rule as `MissingAttributes`); `optionsRemoved`: type change that removes options | FR-021, US2-5 |
| `SetAttributeActive` | Business unit locked; an attribute with **active** dependents cannot be set inactive; a dependent of an inactive parent cannot be reactivated ("Reactivate '<parent>' first — '<label>' depends on it.") | FR-004, FR-023, US2-7 |
| `MoveAttribute(ctx, a, id, MoveInput{What, Direction, Version})` | data-model §4.2; version of the clicked attribute checked; refusals as `validation_failed` on `direction` with the messages table text; each attribute whose position changed: `Touch` + one `changed` entry (field `position`); answers the level's list | FR-019, US2-6, edge case |
| `DeleteAttribute` | Business unit locked; usage = values (live or bin), dependents, conditions; else `MoveToBin` with its options | FR-004, US2-7 |

### Options (`service_options.go`, `service_import.go`, `importfile.go`)

| Method | Rules | FR / scenario |
|---|---|---|
| `ListOptions(ctx, a, attributeID, OptionFilter{Q, Status, Limit, Offset})` | display order; `Q` matches any part of the label ignoring case and surrounding spaces; `limit` 1–500 (default 100), `offset` ≥ 0 (else `validation_failed` "Choose a number from 1 to 500." / "Choose 0 or more."); total = all matches; each with `parentOptions` and `usage` | FR-018, FR-020, SC-007, US2-3 |
| `CreateOption(ctx, a, attributeID, OptionInput)` | the attribute must be a choice type ("<attribute> has no options — its type is <Type>."); label rules; ≤ 5,000; dependent → ≥ 1 active parent option; position last; one `created` entry | FR-018, FR-022 |
| `UpdateOption(ctx, a, id, OptionUpdate)` | rename (unique), parent links (≥ 1 for dependents, newly added links active) | FR-018, FR-022, US2-4 |
| `SetOptionActive` | shared rules | FR-018 |
| `MoveOption(ctx, a, id, OptionMoveInput)` | swap with the neighbour; at the ends "'<label>' is already first." / "… last."; both options `Touch` + `changed` (field `position`) | FR-018 |
| `DeleteOption` | usage = values, linked dependent options, conditions; the last option of a choice attribute cannot be deleted ("'<attribute>' needs at least one option, so '<label>' can't be deleted."); else `MoveToBin` | FR-004, FR-018 |
| `CheckImport(ctx, a, attributeID, ImportFile{FileName, Content})` | decode base64 (> 700 KB decoded → problem on `content`); `readRows`: `.xlsx` (zip signature) — first worksheet, columns A and B, shared and inline strings, row numbers from the file; otherwise CSV (UTF-8, BOM removed, `;` when the first line has more `;` than `,`); `.xls` or anything unreadable → problem on `content`; the first row is a header when its first cell is "Option" (ignoring case); > 5,000 data rows → problem; per row, in order: empty, > 120 characters, dependent without parent label, parent label not found among the parent's options (ignoring case), duplicate in the file (same label — for dependents same label **and** parent; rows with the same label and different parents become one option linked to each parent), already an option (ignoring case, active or not) → `alreadyPresent`, beyond the 5,000-option limit → problem; otherwise `toAdd` | FR-031, US6-1 |
| `Import(ctx, a, attributeID, ImportFile)` | runs the same check again inside one `Write` and adds the `toAdd` options in file order (positions after the last), never removing, renaming, relinking or reactivating; one `changed` entry on the attribute, verb `imported options into`, details "Added <n> options from '<file>'; skipped <m> rows."; nothing to add → `validation_failed` "Nothing to add — every row is already an option or has a problem." | FR-031, US6-2 |

### Forms and values (`form.go`, `values.go`) — the reader API of FR-024 – FR-027

| Function | Contract | FR |
|---|---|---|
| `(*Service).Form(ctx, a, level) (FormView, error)` | the level's sections (display order) with every attribute (active and inactive) and every option; `conditional` per section; `businessUnitAttributeId` and `businessUnitOptions`. API `GET /api/shell-attribute-forms/{level}` | FR-019, FR-024 |
| `(*Service).CheckAttributes(ctx, level, AttributeInput{Values, Previous AttributeValues; BusinessUnit []string}) (AttributeResult, error)` | data-model §9: `Problems` (block saving), `Cleared`, `Missing`, normalised `Values`. Joins the caller's transaction; no actor (it only reads Settings) | FR-024 – FR-027, US3-1, US3-2, US3-4 |
| `(*Service).CheckValues(ctx, a, level, CheckRequest)` | the API wrapper (`POST …/check`) for the preview's **Try saving** | FR-020 |
| `(*Service).MissingAttributes(ctx, level, values, businessUnit) ([]MissingField, error)` | only the `Missing` part, for completeness chips and publish/submit checks of 004–008 | FR-026, FR-027 |
| `(*Service).BusinessUnit(ctx, planValues AttributeValues) ([]string, error)` | the plan's Business unit option ids (empty when none chosen) | FR-023, FR-026 |
| `Diff(before, after any) ([]history.Change, error)` | 001's `history.Diff` for every field except `attributes`, which is compared attribute by attribute into fields `attributes.<attributeId>` (sorted by attribute id) | FR-006 |
| `(*Service).DescribeAttribute(ctx, field string, value any) (label, text string, err error)` | for fields `attributes.<id>`: the attribute's label and the value's text (data-model §12); `""`, `""` for other fields — level kinds of 004–007 call it from their `Describe` | FR-006 |

Helpers the rules use: `Visible(attr, bu)` (exported, *Readers*), `allowedOptions(attr, values)` (active options linked to any
selected parent option), `noOptionsForParent(attr, values)` (D-034), `midSentence(label)` (lower-cases
the first letter only when the second is lower case: "Region" → "region", "GL code" stays).

### Readers for later features (`readers.go`)

All are methods of `*settings.Service` (except the two pure functions at the end), join the caller's
transaction (they run in `s.tx.Read`, which joins a `Read` or `Write` already in `ctx` — 001's store), take
no actor, return the stored Go types of *How later features read Settings* below (records in the recycle
bin are never returned) and answer lists sorted as the Settings pages show them, never `nil`:

| Function | Use |
|---|---|
| `Model(ctx, id) (Model, bool, error)` · `Models(ctx) ([]Model, error)` · `(Model) Allows(activityTypeID string) bool` | 004 plan model, 006 type check (R7), 010 names |
| `ActivityType(ctx, id) (ActivityType, bool, error)` · `ActivityTypes(ctx) ([]ActivityType, error)` | 006 default planned cost and type list, 010, 012 |
| `Attributes(ctx, level Level) ([]Attribute, error)` (display order, active and inactive) · `Options(ctx, attributeID) ([]Option, error)` (display order) · `Option(ctx, id) (Option, bool, error)` | 004–007 forms and views, 010 columns, 012 import |
| `Currency(ctx, code) (Currency, bool, error)` · `Currencies(ctx) ([]Currency, error)` (USD first, then by code) · `Rates(ctx) (money.Rates, error)` | 003 display currencies, 007 line-item currency, 009, 010 conversions |
| `FinanceBudget(ctx, id) (FinanceBudget, bool, error)` · `FinanceBudgetsOfYear(ctx, year int) ([]FinanceBudget, error)` (budgets with ≥ 1 item, by name) · `FinanceBudgetItem(ctx, id) (FinanceBudgetItem, bool, error)` · `FinanceBudgetItems(ctx, budgetID) ([]FinanceBudgetItem, error)` | 004 default budget, 007 link (R12) |
| `CanChoose(active bool, id, previousID string) bool` (pure) | the inactive rule of FR-004: a record may keep an inactive setting it already has, never choose a new one |
| `Visible(attr Attribute, businessUnit []string) bool` (pure) | the condition rule of FR-023/FR-026 (exported form of `visible`); 005 moves between plans, 010 columns |

### Usage, references and impact (`usage.go`)

```go
// Reference is one record outside Settings and the Settings records it uses (data-model §10).
type Reference struct {
	Record              history.RecordRef // {type: "activity", id: "A-1003", name: "Launch film"}
	InBin               bool              // true while the record is in the recycle bin
	Level               Level             // its Shell-attribute level; "" for people
	PlanID              string            // its plan (a plan: its own id); "" for people
	ActivityID          string            // an activity: its own id; a line item: its activity; "" otherwise
	Attributes          AttributeValues   // its Shell attribute values; nil when none
	ModelID             string            // plans
	ActivityTypeID      string            // activities
	Currency            string            // line items: local currency; people: display currency
	PlannedCost         *money.Amount     // line items: planned cost in Currency
	FinanceBudgetID     string            // plans: default finance budget
	FinanceBudgetItemID string            // line items: linked finance budget item
}

// ReferenceSource lists every record of one kind, live and in the recycle bin, inside the caller's
// transaction. Features 003–007 implement one each (reading the bin with history.SnapshotsInBin).
type ReferenceSource func(ctx context.Context) ([]Reference, error)
```

- `buildUsage(ctx)` (inside the request's transaction) reads every source once plus the Settings tables
  and the Settings snapshots in the bin, and answers `Usage` for any record (data-model §10, texts in
  *User-visible messages*). Plan references give the business unit of their campaigns, initiatives,
  activities and line items (`PlanID`).
- Impact statements use live references only and the same counting functions (`becomesIncomplete` uses
  `missingFor`, the function behind `MissingAttributes`).
- Currency impact: for every live line item in the currency, `ToUSD(cost, old)` and `ToUSD(cost, new)`;
  the sums of those rounded values are "before → after"; activities = distinct `ActivityID`.

### Currencies (`service_currencies.go`)

| Method | Rules beyond the shared ones | FR / scenario |
|---|---|---|
| `ListCurrencies` | USD first, then by code; `inverseRate`, `usage`, `locks` | FR-028, US4-2 |
| `CreateCurrency(ctx, a, CurrencyInput{Code, Name, Symbol, Rate})` | code: trimmed, upper-cased, `^[A-Z]{3}$` ("Enter a three-letter code such as JPY."), unique ("A currency with the code '<CODE>' already exists."); name ≤ 80, unique; symbol ≤ 5; rate parsed (`ParseRate`); all problems at once; `rateChangedAt/By` = now/actor | FR-028, US4-1, US4-5 |
| `UpdateCurrency(ctx, a, code, CurrencyUpdate)` | no code field (the lock is shown); USD rate other than 1 → lock reason on `rate`; a changed rate sets `rateChangedAt/By` and the `changed` entry's details (live line items and activities, USD before → after; empty when none) | FR-028 – FR-030, US4-3, US4-4 |
| `CurrencyImpact` | `amountsRecalculated` when the rate changes and live line items use it | FR-030, US4-3 |
| `SetCurrencyActive`, `DeleteCurrency` | USD locked ("USD is NorthStar's base currency; its rate is always 1."); usage = line items and people | FR-004, FR-028, US4-2 |
| `Rates(ctx)` | every currency's current rate, active or not | FR-029 |

### Finance budgets and items (`service_finance.go`)

| Method | Rules | FR / scenario |
|---|---|---|
| `ListFinanceBudgets(ctx, a, year *int) (FinanceBudgetPage, error)` | default year = the clock's UTC year; budgets of that year with ≥ 1 item, by name, items by name; `activeItemCount`, `usage` (plans' default); `years` = current − 1 … + 5 plus every year with items | FR-014, FR-015, US5-3 |
| `CreateFinanceBudgetItem(ctx, a, ItemInput{Year, BudgetName, Name, Code})` | year in current − 1 … + 5; budget name 1–120; name 1–120, unique in the budget ("The budget '<budget>' already has an item called '<name>'."); code ≤ 40, unique in the year ("The code '<code>' is already used in <year> by '<item>'."); the budget is found by year + name ignoring case or created (one `created` entry for it); one `created` entry for the item | FR-014, US5-3 |
| `GetFinanceBudgetItem` | `year`, `budgetName`, `usage`, `locks` (year) | US5-4 |
| `UpdateFinanceBudgetItem(ctx, a, id, ItemUpdate{BudgetName, Name, Code *string; Version})` | a new budget name moves it to that budget of the same year (found or created); same rules as create | FR-014, US5-4 |
| `RenameFinanceBudget(ctx, a, id, RenameInput{Name, Version})` | 1–120, unique in its year ("<year> already has a budget called '<name>'."); one `changed` entry on the budget | FR-015, US5-4 |
| `SetFinanceBudgetItemActive`, `DeleteFinanceBudgetItem` | usage = line items | FR-004, US5-5 |

### History kinds, restore and log

Data-model §11 (kinds and `PrepareRestore`) and §12 (entries, `Describe` texts). The name rule reuses
001's samples approach (`restoredName`); each kind's uniqueness scope is its list (models, types), its
level (attributes), its attribute (options) or its budget (items).

### Changes to shared parts (owned by 001)

| File | Change |
|---|---|
| `backend/internal/platform/errs/errs.go` | `CodeInUse = "in_use"`, `CodeCannotRestore = "cannot_restore"` (both 409); `InUse(message string) *Error`, `CannotRestore(message string) *Error`. Feature 003 (built in parallel) adds the same `CodeInUse` / `InUse`: whichever branch merges second keeps the one definition (409, `Fields` empty) |
| `backend/internal/history/service.go` | `(*Service).SnapshotsInBin(ctx, kindName string) ([]Snapshot, error)`: the snapshots of that kind in unresolved recycle-bin entries (deleted records and their contents), oldest entry first; joins the caller's transaction |
| `backend/api/openapi.yaml` | merge the fragment; `Error.code` enum gains `in_use`, `cannot_restore`; the `Conflict` response description lists them |
| `backend/internal/app/app.go` | build `settings` (all environments), register its kinds, `AddReferenceSource` calls for later features, routes |
| `backend/seed/seed.go` | `addSettings` (data-model §13) inside `Load`'s `Write`, after the people |
| `frontend/src/api/errors.ts` | `ApiError.kind` gains `inUse` and `cannotRestore` (the API's message is shown) |
| `frontend/src/lib/useUndoableDelete.ts` | `cannotRestore` shows the API's message |
| `frontend/src/components/TextField` | optional `suggestions: string[]` (a `<datalist>`) for budget names |
| `frontend/src/app/navigation.ts`, `router.tsx` | Settings routes; Settings leaves `COMING_SOON_SECTIONS` |

## API operations

All under the session (`401 sign_in_required` without it). Bodies and answers: the fragment's schemas.
Errors: `400 validation_failed` (all field problems) or `bad_request`; `404 not_found`; `409
version_conflict` on every update, status change and move; `409 in_use` on deletes; `409 cannot_restore`
only from the recycle bin (001's endpoints).

| Method and path | Request → response | Errors beyond 401 | FR |
|---|---|---|---|
| `GET /api/settings/overview` | → `SettingsOverview` | — | FR-001 |
| `GET /api/models` | → `ModelPage` | — | FR-002, FR-003 |
| `POST /api/models` | `ModelCreate` → 201 `Model` | 400 | FR-009 |
| `GET /api/models/{modelId}` | → `Model` | 404 | FR-003 |
| `PATCH /api/models/{modelId}` | `ModelUpdate` → `Model` | 400, 404, 409 | FR-009, FR-011, FR-012 |
| `POST /api/models/{modelId}/impact` | `ModelUpdate` → `ChangeImpact` | 400, 404 | FR-011, FR-012 |
| `POST /api/models/{modelId}/status` | `StatusUpdate` → `Model` | 400, 404, 409 | FR-004 |
| `DELETE /api/models/{modelId}` | → `DeleteResult` | 404, 409 `in_use` | FR-004 |
| `GET /api/activity-types` | → `ActivityTypePage` | — | FR-002 |
| `POST /api/activity-types` | `ActivityTypeCreate` → 201 `ActivityType` | 400 | FR-013 |
| `GET /api/activity-types/{activityTypeId}` | → `ActivityType` | 404 | FR-003 |
| `PATCH /api/activity-types/{activityTypeId}` | `ActivityTypeUpdate` → `ActivityType` | 400, 404, 409 | FR-012, FR-013 |
| `POST /api/activity-types/{activityTypeId}/impact` | `ActivityTypeUpdate` → `ChangeImpact` | 400, 404 | FR-012 |
| `POST /api/activity-types/{activityTypeId}/status` | `StatusUpdate` → `ActivityType` | 400, 404, 409 | FR-004 |
| `DELETE /api/activity-types/{activityTypeId}` | → `DeleteResult` | 404, 409 | FR-004 |
| `GET /api/shell-attributes?level=` | → `ShellAttributePage` (display order) | 400 (unknown level) | FR-002, FR-019 |
| `POST /api/shell-attributes` | `ShellAttributeCreate` → 201 `ShellAttribute` | 400 | FR-016 – FR-019, FR-022, FR-023 |
| `GET /api/shell-attributes/{attributeId}` | → `ShellAttribute` | 404 | FR-020 |
| `PATCH /api/shell-attributes/{attributeId}` | `ShellAttributeUpdate` → `ShellAttribute` | 400, 404, 409 | FR-016 – FR-023 |
| `POST /api/shell-attributes/{attributeId}/impact` | `ShellAttributeUpdate` → `ChangeImpact` | 400, 404 | FR-021 |
| `POST /api/shell-attributes/{attributeId}/status` | `StatusUpdate` → `ShellAttribute` | 400, 404, 409 | FR-004, FR-023 |
| `POST /api/shell-attributes/{attributeId}/move` | `AttributeMove` → `ShellAttributePage` | 400, 404, 409 | FR-019 |
| `DELETE /api/shell-attributes/{attributeId}` | → `DeleteResult` | 404, 409 | FR-004 |
| `GET /api/shell-attributes/{attributeId}/options?q=&status=&limit=&offset=` | → `ShellAttributeOptionPage` | 400, 404 | FR-018, FR-020 |
| `POST /api/shell-attributes/{attributeId}/options` | `OptionCreate` → 201 `ShellAttributeOption` | 400, 404 | FR-018, FR-022 |
| `POST /api/shell-attributes/{attributeId}/options/import-check` | `OptionImportRequest` → `OptionImportCheck` | 400, 404 | FR-031 |
| `POST /api/shell-attributes/{attributeId}/options/import` | `OptionImportRequest` → `OptionImportResult` | 400, 404 | FR-031 |
| `PATCH /api/shell-attribute-options/{optionId}` | `OptionUpdate` → `ShellAttributeOption` | 400, 404, 409 | FR-018, FR-022 |
| `POST /api/shell-attribute-options/{optionId}/status` | `StatusUpdate` → `ShellAttributeOption` | 400, 404, 409 | FR-018 |
| `POST /api/shell-attribute-options/{optionId}/move` | `OptionMove` → `ShellAttributeOption` | 400, 404, 409 | FR-018 |
| `DELETE /api/shell-attribute-options/{optionId}` | → `DeleteResult` | 404, 409 | FR-004, FR-018 |
| `GET /api/shell-attribute-forms/{level}` | → `AttributeForm` | 400 | FR-024 |
| `POST /api/shell-attribute-forms/{level}/check` | `AttributeCheckRequest` → `AttributeCheckResult` | 400 | FR-020, FR-025 – FR-027 |
| `GET /api/currencies` | → `CurrencyPage` | — | FR-028 |
| `POST /api/currencies` | `CurrencyCreate` → 201 `Currency` | 400 | FR-028 |
| `GET /api/currencies/{code}` | → `Currency` | 404 | FR-003 |
| `PATCH /api/currencies/{code}` | `CurrencyUpdate` → `Currency` | 400, 404, 409 | FR-028 – FR-030 |
| `POST /api/currencies/{code}/impact` | `CurrencyUpdate` → `ChangeImpact` | 400, 404 | FR-030 |
| `POST /api/currencies/{code}/status` | `StatusUpdate` → `Currency` | 400, 404, 409 | FR-004, FR-028 |
| `DELETE /api/currencies/{code}` | → `DeleteResult` | 404, 409 | FR-004, FR-028 |
| `GET /api/finance-budgets?year=` | → `FinanceBudgetPage` | 400 (year not a number) | FR-014, FR-015 |
| `PATCH /api/finance-budgets/{budgetId}` | `FinanceBudgetRename` → `FinanceBudget` | 400, 404, 409 | FR-015 |
| `POST /api/finance-budget-items` | `FinanceBudgetItemCreate` → 201 `FinanceBudgetItem` | 400 | FR-014 |
| `GET /api/finance-budget-items/{itemId}` | → `FinanceBudgetItem` | 404 | FR-003 |
| `PATCH /api/finance-budget-items/{itemId}` | `FinanceBudgetItemUpdate` → `FinanceBudgetItem` | 400, 404, 409 | FR-014 |
| `POST /api/finance-budget-items/{itemId}/status` | `StatusUpdate` → `FinanceBudgetItem` | 400, 404, 409 | FR-004 |
| `DELETE /api/finance-budget-items/{itemId}` | → `DeleteResult` | 404, 409 | FR-004 |

Field names in `validation_failed` are the request's JSON names; nested ones use dots and indexes:
`averageDealSize`, `options[2].label`, `options[2].parentOptionIds`, `optionLinks.SO-1012`,
`attributes.SA-1004`, `content`, `direction`.

## Demo data

Data-model §13: `seed.Load` calls `addSettings(ctx, generator, createdAt)` after the people, in the same
`Write`, through `settings.NewJSONRepository()` and the ID generator, created by `actor.System`, without
log entries (as 001's people). It returns `SettingsSeed{Models, ActivityTypes, Attributes, Options,
Budgets, Items map[string]string}` keyed by name (attributes `"<level>/<label>"`, options
`"<level>/<attribute>/<option>"`, budgets `"<year>/<name>"`, items `"<year>/<budget>/<item>"`), which
the planning seed of 004–007 receives. The message of `cmd/seed` does not change.

## Frontend design

### Routes and navigation

| Path | Page | Tab title |
|---|---|---|
| `/settings` | `SettingsPage` | Settings |
| `/settings/models` (`?id=MOD-1001` opens its details) | `ModelsPage` | Models |
| `/settings/activity-types` (`?id=`) | `ActivityTypesPage` | Activity types |
| `/settings/shell-attributes` (`?level=plan`, default `plan`) | `ShellAttributesPage` | Shell attributes |
| `/settings/shell-attributes/new` (`?level=`) | `ShellAttributeEditorPage` (create) | New Shell attribute |
| `/settings/shell-attributes/:attributeId` | `ShellAttributePage` | <label> · Shell attributes |
| `/settings/shell-attributes/:attributeId/edit` | `ShellAttributeEditorPage` (edit) | Edit <label> |
| `/settings/currencies` (`?id=EUR`) | `CurrenciesPage` | Currencies |
| `/settings/finance-budget-items` (`?year=2026&id=`) | `FinanceBudgetItemsPage` | Finance budget items |

- All inside `RequireSignIn` with `handle: { title }`; `navigation.ts`: Settings becomes a plain
  `NavItem` and leaves `COMING_SOON_SECTIONS`; the sidebar highlights Settings on every `/settings/*`
  path (NavLink's default matching). Home's step 1 already links to `/settings`.
- List filters live in the URL (`q`, `status` — `active` default, `inactive`, `all`) through
  `useListFilters`, so Back and links keep them.
- Every section page starts with `Breadcrumb` (Settings › <Section>), then `PageHeader` (eyebrow
  "Settings").

### Read hooks other features reuse (`src/api/settingsQueries.ts`)

`SETTINGS_KEYS` (`['models']`, `['activity-types']`, `['shell-attributes', level]`,
`['shell-attribute', id]`, `['shell-attribute-options', id, params]`, `['shell-attribute-form', level]`,
`['currencies']`, `['finance-budgets', year]`, `['settings-overview']`) and `useModels()`,
`useActivityTypes()`, `useCurrencies()`, `useFinanceBudgets(year)`, `useAttributeForm(level)`
(`staleTime: 0`, refetched on mount, so a changed setting shows on the next form — SC-008). Mutations
live in each section's `…Api.ts` and invalidate the keys they affect plus `['activity-log']`,
`['recycle-bin']` and `['settings-overview']`.

### New shared components (`src/components/`, each with a test and a gallery entry)

| Component | Props (essentials) | Behaviour | Why shared |
|---|---|---|---|
| `Breadcrumb` | `items: {label, to?}[]` | `<nav aria-label="Breadcrumb">`, links separated by "›", last item `aria-current="page"` | Every section and details page (002, 004–007) |
| `SearchField` | `label`, `value`, `onChange`, `placeholder?` | `type="search"`, search icon, **Clear search** button (Tooltip) | Every list toolbar |
| `SegmentedFilter<T>` | `label`, `options: {value, label, count?}[]`, `value`, `onChange` | Pill toggle buttons in a `role="group"` with `aria-pressed`, "Active 5" | Status and level filters (002, 004, 010) |
| `MoneyField` | `label`, `name`, `value` (text), `onChange`, `currency`, `error?`, `hint?`, `marking?` | Currency inside the field; accepts `12500`, `12,500.00`, `12.5k` (`parseMoneyInput`); formats on blur | Deal size, default cost (002), budgets and costs (004–007) |
| `DecimalField` | `label`, `name`, `value` (text), `onChange`, `decimals: 0 \| 2 \| 6`, `suffix?`, `error?`, `hint?`, `marking?` | `inputMode="decimal"`; suffix "%" or "per USD"; whole numbers with `decimals: 0` | Rates (002), INQ goals (004, 006) |
| `CheckboxField` | `label`, `name`, `checked`, `onChange`, `hint?`, `error?` | Labelled tick box | Yes/no attributes, AOP and BCPIR flags (004) |
| `TextAreaField` | `label`, `name`, `value`, `onChange`, `maxLength?`, `hint?`, `error?`, `marking?` | Label above, error under, "n characters left" when ≤ 50 remain | Help text (002), descriptions (004–007) |
| `ChoiceField` | `label`, `name`, `options: {id, label, active}[]`, `value: string \| string[]`, `multiple?`, `onChange`, `error?`, `hint?`, `marking?`, `disabledReason?` | ≤ 10 options: `<select>` (single, first entry "Choose…") or a check-box group (`fieldset`/`legend`, multiple); > 10: ARIA combobox (input + `listbox` in a Radix Popover, `aria-activedescendant`, arrows/Enter/Escape) filtering by any part of the label, at most 50 shown with "Showing 50 of 1,217 — keep typing to narrow down."; chosen values as chips with **Remove <label>**. Inactive options are offered only when already chosen and read "<label> (inactive)". `disabledReason` disables the control and shows the reason under it | Every choice on Settings and planning forms (FR-024, SC-007) |
| `SplitEditor` | `label`, `name`, `options`, `value: {optionId, share}[]`, `onChange`, `error?`, `marking?`, `hint?` | Picks values with `ChoiceField` (multiple); a table Value · Share (%) with `DecimalField`s; running total line; **Split evenly**; adding or removing a value splits evenly again (note) (R10, design system §5) | % split attributes (FR-027) |
| `LockedNote` | `children` | Lock icon + reason in `--ink-2` (design system §6.6) | Locks in 002, the year lock of 004 |
| `ShellAttributeFields` | `form: AttributeForm`, `values`, `onChange(values)`, `businessUnit: string[] \| null`, `previousValues?`, `errors?: Record<string, string>`, `highlightAttributeId?`, `planLink?` | Renders a level's attributes by section (FR-024 – FR-027): marking word, ⓘ help (`InfoTip`), the control per type (`ChoiceField`, `SplitEditor`, `CheckboxField`, `TextField`, `TextAreaField`, `DecimalField` 0 decimals), dependency (disabled with "Choose a <parent> first."; options filtered by parent; auto-fill with its message when exactly one fits; D-034 note), conditions (hidden attributes keep their values; section note; no-BU note with `planLink`), inactive values read-only "(inactive)". Also exports `ShellAttributeValues` (read-only, "—" for empty), `ClearValuesDialog` (lists values to be cleared before saving) and `rules.ts` (`visibleAttributes`, `allowedOptions`, `valuesToClear`, `missingAttributes`, `autoFill`) mirroring Go | The preview (FR-020) and the forms and details of 004–007 |

`TextField` gains `suggestions?: string[]` (renders a `<datalist>`).

### Shared Settings parts (`features/settings/shared/`)

- `SettingDeleteDialog` — given the record (`name`, `usage`, `active`, record type): unused → 001's
  delete dialog texts, confirm "Delete <record type>"; in use → title "'<name>' is in use", body
  "<summary> — set it inactive instead. <inactive effect>", buttons **Cancel** (focused) and **Set
  inactive** (already inactive: "<summary>, so it can't be deleted. It's already inactive." and **Close**);
  a `409 in_use` from the API switches the dialog to the in-use state with the API's message. After a
  delete: `useUndoableDelete`.
- `ImpactDialog` — `ConfirmDialog` titled "Save changes to <record type> '<name>'?" listing the warnings'
  messages, **Save changes** / **Keep editing** (focused).
- `UsageLine` — `usage.summary` plus links of `usage.records` (plans → `/plans/<id>`).
- `MetaLine` — "Created by <name> on <date>, last changed by <name> on <date>".
- `useSettingStatus(kind)` — the Set inactive / Reactivate mutation, toasts, conflict message.
- `useListFilters()` — `q` and `status` in the URL; reordering and "Clear filters".
- The **save flow** of every edit form: client-side required checks for instant feedback → (when the
  record has an impact endpoint and a relevant field changed) `POST …/impact` → `validation_failed` shows
  every problem (`FormErrorSummary` + fields) → warnings → `ImpactDialog` → `PATCH` → toast; `409
  version_conflict` → `conflictMessage(error, noun)` at the top, typed values kept, latest version taken
  (001's edit-form rule).

### Pages

**Settings** (`SettingsPage`, FR-001): `PageHeader` eyebrow "Settings", title "Settings", description
"Set up the lists and rules every plan uses. Changes here apply to everyone straight away."; a grid of
`Card`s from `sections.ts` (002's five, then People, Teams — 003 — and Approval workflows — 008), each a
link with icon, title, explanation and "<n> active" from `GET /api/settings/overview`; sections not yet
built show a neutral "Coming soon" chip and are not links. States: loading (counts as `Skeleton`), error
(cards without counts and "Counts couldn't be loaded." with **Try again**).

**Models** (`ModelsPage`, US1): header "Models", description, **New model**. Toolbar: `SearchField`
("Search name"), `SegmentedFilter` (Active · Inactive · All with counts), "<shown> of <total>". Table:
Name (button opening details), Rates ("2.00 % → 5.00 %"), Average deal size, Allowed activity types
(count), Used by, Status chip, `ActionsMenu` (View · Edit · Duplicate · Set inactive / Reactivate ·
Delete). `SidePanel` details (`ModelDetails`): every field, `WaterfallExample` with its editable INQ,
allowed types as chips ("(inactive)" when so), `UsageLine`, `MetaLine`; footer **Edit** (primary) and ⋯
More (Duplicate, Set inactive / Reactivate, Delete). `ModelForm` in the panel ("New model" / "Edit
model"): Name, INQ→MQL rate, MQL→deal rate (`DecimalField` 2, "%"), Average deal size (`MoneyField`
USD), Allowed activity types (`ChoiceField` multiple: active types plus already-allowed inactive ones),
then the **Worked example** (live, `lib/waterfall.ts`); **Save** / **Cancel**; unsaved-changes guard;
Duplicate opens it unsaved with "<name> (copy)".

**Activity types** (`ActivityTypesPage`, US5): as Models; columns Name, Default planned cost, Allowed in
(model names), Used by, Status. Form: Name, Default planned cost (`MoneyField` USD, optional; after a
change on edit the note "Applies when someone picks this type from now on."), Allowed in models
(`ChoiceField` multiple). Impact check when models are removed.

**Shell attributes** (`ShellAttributesPage`, US2): header "Shell attributes", **New Shell attribute**
(→ `/settings/shell-attributes/new?level=<level>`). Level `SegmentedFilter` (Plan · Campaign · Initiative
· Activity · Line item, with counts), `SearchField` ("Search label or section"), status filter. Table
grouped by section (a header row per section with **Move section up** / **Move section down** icon
buttons, accessible names "Move section '<name>' up/down"); rows: order buttons ("Move '<label>' up/down";
at an edge the button stays focusable with `aria-disabled` and a tooltip giving the reason), Label (link),
Type, Marking, Options (count), Depends on / Shown when, Used by, Status, `ActionsMenu` (View · Edit · Set
inactive / Reactivate · Delete). While searching or filtering, the order buttons explain "Clear the search
and filters to change the order." After a move, focus stays on the moved row's button and a polite live
region says "Moved '<label>' up." (SC-009).

**Shell attribute** (`ShellAttributePage`, US2, US3, FR-020): `Breadcrumb` Settings › Shell attributes ›
<Level> form › <label>; header with the label, level tag, status chip, **Edit** (primary), ⋯ (Set
inactive / Reactivate, Delete); locks shown with `LockedNote`. Cards:
1. **Settings** — level, type, marking, section, help text, depends on, shown when, `UsageLine`, `MetaLine`.
2. **Options** (`OptionsCard`, choice types; otherwise "<Type> attributes have no options.") — `SearchField`
   ("Search options", server `q`), status filter, table Label · Linked to · Used by · Status · ⋯ (Edit ·
   Move up · Move down · Set inactive / Reactivate · Delete), "Showing <n> of <total> — search to find
   the others." with **Show more**; buttons **Add option** (`OptionPanel`: Label; Linked to for
   dependents), **Import from a spreadsheet** (`ImportOptionsDialog`), **Download template**
   (`importTemplate.ts`, a CSV built in the browser).
3. **Preview — <Level> form** (`AttributePreview`): note "Behaves like the real form. Nothing you type
   here is saved."; below plan level the check boxes "Preview as a plan whose business unit includes:"
   (all ticked at first); `ShellAttributeFields` with local values and the attribute highlighted; a live
   `CompletenessChip`; **Try saving** → `POST …/check` → "Saving would work." or the problems
   (`FormErrorSummary`), "These values would be cleared: …", and the fields still needed.

**Shell attribute editor** (`ShellAttributeEditorPage`, full page, US2, US3): sections *Basics* (Label,
Help text, Level, Type, Marking — markings offered by level; yes/no fixes Optional with its hint), *Place
on the form* (Section with suggestions of the level's sections), *Depends on and condition* (Depends on —
choice attributes of the level placed before it, "None"; Shown when the plan's business unit includes —
check boxes), *Options* (create only — `OptionsEditor`: rows Label · Linked to (dependents) · Move up ·
Move down · Remove, **Add option**; on edit, when Depends on changes, the same editor shows every active
option with its links). Locked fields show `LockedNote`. Sticky footer **Save** / **Cancel**; impact check
on edit; unsaved-changes guard. After create → the attribute page with toast "Shell attribute '<label>'
created".

**Import** (`ImportOptionsDialog`, US6): file input (`.csv,.xlsx`), size check (700 KB) in the browser,
then `import-check` → review: the counts line, three lists with row numbers (to add, already present,
problems — each collapsible, the first 100 rows shown with "and <n> more"), **Add <n> options** (primary)
and **Cancel**; nothing to add → the message and **Cancel** only. Success → toast "<n> options added to
<label>".

**Currencies** (`CurrenciesPage`, US4): columns Code, Name, Symbol, Rate ("1 USD = 0.92 EUR" over "1 EUR =
1.086957 USD"), Used by, Rate last changed, Status. Form: Code (create only; afterwards `LockedNote`),
Name, Symbol, Rate (`DecimalField` 6, "per USD") with the live line "1 USD = … · 1 … = … USD"; USD shows
`LockedNote` for rate and status. Impact check when the rate changes.

**Finance budget items** (`FinanceBudgetItemsPage`, US5): header "Finance budget items", description with
"A simple list until the Finance module arrives"; **New finance budget item**; toolbar: Budget year
(`SegmentedFilter` of `years`, current year first selected), search, status filter. One card per budget:
title = budget name, subtitle "<n> items · <usage summary>", action **Rename budget**
(`RenameBudgetDialog`); table Item · Code · Linked to · Status · ⋯. Form (side panel): Budget year
(select; `LockedNote` on edit), Budget name (`TextField` with `suggestions` = that year's budget names,
hint "Items with the same year and budget name form one finance budget."), Item name, Code.

### States on every page

| State | Shown as |
|---|---|
| Loading | `Skeleton` rows in the table card; old data stays while refreshing |
| Empty | `EmptyState` with the section's sentence and create button |
| Filtered to nothing | "Nothing matches these filters" with **Clear filters** |
| Error loading | `ErrorState` with **Try again** (unreachable → 001's sentence) |
| Validation | `FormErrorSummary` (focus) + message under each field, all at once |
| Conflict | `conflictMessage(error, noun)` at the top of the form, typed values kept |
| Not found (`?id=` or `:attributeId` gone) | details: the type's "no longer exists" message with **Back to <section>**; attribute page: `ErrorState`-style card with that message |
| In use (delete) | `SettingDeleteDialog` in-use state |
| Impact | `ImpactDialog` |
| Locked | `LockedNote` with the API's reason |

### Front-end rules mirrored from Go (feedback only)

`lib/waterfall.ts` (`waterfall`, BigInt; same table as `waterfall_test.go`), `lib/decimal.ts`
(`parseMoneyInput`, `parseDecimalInput`, `formatPercent`, `formatRate`, `inverseRate`, `formatWhole`,
`parseWholeInput`, `toCents`, `formatCents`),
`ShellAttributeFields/rules.ts` (same cases as `form_test.go`), `lib/text.ts` (`midSentence`, `plural`).
The server stays the judge: a save always goes through the API.

## User-visible messages

Every text appears here once; code and tests quote it. `<…>` is filled in at run time. Plurals: "1
model" / "<n> models" — the singular for 1 everywhere.

| Where | Text |
|---|---|
| Settings page | Settings · Set up the lists and rules every plan uses. Changes here apply to everyone straight away. |
| Settings cards | Models — The rulebooks plans use: conversion rates, average deal size and allowed activity types. · Activity types — The kinds of work an activity can be, each with an optional default planned cost. · Shell attributes — Shell's own fields on each level's form, with their options and markings. · Currencies — The currencies line items and people can use, with their rate per 1 USD. · Finance budget items — The finance budget items line items link to — a simple list until the Finance module arrives. · People — Everyone who can sign in to NorthStar. · Teams — The teams plans belong to. · Approval workflows — Who approves activities, level by level. · <n> active · Coming soon · Counts couldn't be loaded. |
| Section descriptions | Models: The rulebooks plans use. Changes apply to every plan using the model. · Activity types: The kinds of work activities can be. · Shell attributes: Shell's own fields on each level's form. Changes apply to every form straight away. · Currencies: Currencies for line items and display, with their rate per 1 USD. · Finance budget items: The finance budget items line items link to. A simple list until the Finance module arrives. |
| Create buttons / empty states | New model · No models yet — create your first model · New activity type · No activity types yet — create your first activity type · New Shell attribute · No <level> attributes yet — create the first one · New currency · New finance budget item · No finance budget items for <year> yet — create the first one |
| Lists | Search name · Search label or section · Search code or name · Search item, budget or code · Search options · Active · Inactive · All · <shown> of <total> · Nothing matches these filters · Clear filters · (inactive) · Not used yet · Created by <name> on <date>, last changed by <name> on <date> |
| Actions | View · Edit · Duplicate · Set inactive · Reactivate · Delete · Save · Cancel · Save changes · Keep editing · Close · Back to <section> |
| Toasts | <Record type> '<name>' created (models also **View**) · <Record type> '<name>' saved · <Record type> '<name>' set inactive · <Record type> '<name>' reactivated · Option '<label>' added · <n> options added to <label> · Budget renamed to '<name>' |
| Not found | This model no longer exists — someone may have deleted it. · This activity type no longer exists — someone may have deleted it. · This Shell attribute no longer exists — someone may have deleted it. · This option no longer exists — someone may have deleted it. · This currency no longer exists — someone may have deleted it. · This finance budget no longer exists — someone may have deleted it. · This finance budget item no longer exists — someone may have deleted it. |
| Delete, unused | Delete <record type> '<name>'? · It moves to the recycle bin; you can restore it. · It moves to the recycle bin together with its <n> options; you can restore it. · Delete model · Delete activity type · Delete Shell attribute · Delete option · Delete currency · Delete finance budget item |
| Delete, in use | '<name>' is in use · <summary> — set it inactive instead. · <summary>, so it can't be deleted. It's already inactive. · API: '<name>' is in use (<summary, lower-case first letter>) — set it inactive instead. |
| Inactive effects | Model: New plans can't choose an inactive model; plans using it keep it and still calculate goals. · Activity type: New activities can't choose an inactive type; activities using it keep it. · Shell attribute: Forms stop offering it; values already on records stay visible as "(inactive)" and never block publishing or submitting. · Option: Forms stop offering it; records that have it keep it as "(inactive)". · Currency: Line items and people can't choose an inactive currency; those using it keep it and amounts still convert. · Finance budget item: Line items can't link to an inactive item; those linked keep it. |
| Usage parts | Used by <n> plans · Used by <n> campaigns · Used by <n> initiatives · Used by <n> activities · Used by <n> line items · Display currency of <n> people · Allowed in <n> models · <Label> depends on it · <n> attributes depend on it · Condition of <n> attributes · Linked to <n> <attribute> options · Linked to <n> line items · Default finance budget of <n> plans · suffix " (<n> in the recycle bin)" · joined with " · " |
| Impact dialog | Save changes to <record type> '<name>'? · Used by <n> plans (<m> activities). Their calculated MQL and revenue goals will be recalculated; typed goals ('Set later stages myself') are kept. · <n> activities use '<type>'. They keep it; new activities can't choose it. (1: 1 activity uses '<type>'. It keeps it; new activities can't choose it.) · In plans using '<model>', <n> activities use '<type>'. They keep it; new activities can't choose it. · <n> <records> have no value yet and will show as incomplete. Submitted and approved activities keep their status. (plan, campaign, initiative: … Published plans keep their status.) · Its <n> options will be removed: <first five labels>< and <k> more>. (1: Its option '<label>' will be removed.) · <n> line items in <m> activities use <CODE>. Their <CODE> amounts stay the same; their USD amounts will be recalculated (<before> → <after> USD). Unassigned planned cost of these activities will change. No activity goes back for approval. |
| Log details | with <n> options · Used by <n> plans (<m> activities): calculated goals recalculated. · <n> line items in <m> activities recalculated: <before> → <after> USD. · Removed its <n> options: <labels>. · Added <n> options from '<file>'; skipped <m> rows. · Renamed to '<new name>' because the name was taken. (001) · Code '<code>' removed because another item now uses it. |
| Model form | New model · Edit model · Name · INQ→MQL rate · MQL→deal rate · Average deal size · Allowed activity types · Activities in plans using this model can be of these types. · Worked example · Calculated — not saved · Example number of inquiries · <INQ> INQ → <MQL> MQL → <deals> deals → <revenue> USD revenue · Fill in both rates and the average deal size to see the example. · Too large to calculate — try fewer inquiries. · ⓘ MQL = inquiries × INQ→MQL rate and deals = MQL × MQL→deal rate, each rounded to a whole number; revenue = deals × average deal size. · <name> (copy) |
| Model validation | Enter a name. · Use 80 characters or fewer. · A model called '<name>' already exists. · Enter a rate above 0 % and at most 100 %, with at most two decimals. · Enter an average deal size above 0 USD, with at most two decimals. · Use 100,000,000.00 USD or less. · Enter the average deal size in USD. · Tick at least one activity type. · An activity type you ticked no longer exists — reload the page and try again. · '<type>' is inactive — tick active activity types only. · Enter a whole number from 0 to 1,000,000,000. (example) |
| Activity type form | New activity type · Edit activity type · Name · Default planned cost · Offered when someone picks this type for an activity. · Applies when someone picks this type from now on. · Allowed in models · Plans using these models can choose this type. |
| Activity type validation | Enter a name. · Use 120 characters or fewer. · An activity type called '<name>' already exists. · Enter an amount of 0 or more, with at most two decimals. · Enter the default planned cost in USD. · A model you ticked no longer exists — reload the page and try again. · '<model>' needs at least one activity type, so '<type>' can't be removed from it. |
| Attribute list | Plan · Campaign · Initiative · Activity · Line item · Order · Label · Type · Marking · Options · Depends on / shown when · Used by · Status · Move '<label>' up · Move '<label>' down · Move section '<name>' up · Move section '<name>' down · Clear the search and filters to change the order. · Moved '<label>' up. / Moved '<label>' down. · Always shown |
| Attribute words | Single choice · Multiple choice · Multiple choice with % split · Yes/no · Text · Long text · Number · Optional · Needed to publish · Needed to submit · Required |
| Attribute editor | New Shell attribute · Edit Shell attribute '<label>' · Basics · Label · Help text · Shown as an ⓘ tip next to the field on forms. · Level · Type · Marking · Always optional; shown as a tick box that defaults to No. · People split the record's cost over the chosen values; the shares must add up to 100 %. · Place on the form · Section · Attributes of the same section are shown together; sections appear in the order of their first attribute. · Depends on and condition · Depends on · None · Its options are filtered by the value chosen for that attribute. · Shown when the plan's business unit includes · Leave all unticked to show it always. · Options · Add option · Remove '<label>' · Linked to |
| Attribute validation | Enter a label. · Use 80 characters or fewer. · The <Level> form already has a Shell attribute called '<label>'. · Use 300 characters or fewer. · Choose a level. · Choose a type. · Choose a marking. · At <level> level the marking is Optional or Needed to publish. · At <level> level the marking is Optional or Needed to submit. · Yes/no attributes are always optional — for a needed yes-or-no answer, use a single choice with the options Yes and No. · Multiple choice with % split is only for activities and line items, which carry cost. · Use 60 characters or fewer. · Add at least one option. · '<label>' appears twice. · Enter a label for every option. · Use 120 characters or fewer. · An attribute can have at most 5,000 options. · Only choice attributes have options. · Only choice attributes can depend on another attribute. · Choose a single or multiple choice attribute that comes before this one on the <Level> form. · Link '<option>' to at least one <parent> option. · '<option>' can only be linked to options of <parent>. · '<parent option>' is inactive — link active <parent> options only. · Choose business units for the condition from the Business unit options. · There is no Business unit attribute yet, so conditions can't be set. |
| Locks | Type can't change once records use this attribute — create a new attribute instead. · Level can't change once records use this attribute — create a new attribute instead. · <Label> depends on it, so its type can't change. · Level can't change while it depends on, or is depended on by, another attribute. · Business unit can't be deleted, set inactive, moved or given another type, because other attributes are shown or hidden by it. · Set '<dependent>' inactive first — it depends on '<label>'. · Reactivate '<parent>' first — '<label>' depends on it. · The code can't change once the currency is saved. · USD is NorthStar's base currency; its rate is always 1. · The year can't change once the item is saved — create a new item for another year. · '<attribute>' needs at least one option, so '<label>' can't be deleted. |
| Moves | '<label>' is already first in its section. · '<label>' is already last in its section. · This section is already first. · This section is already last. · <dependent> must stay below <parent>. · Business unit stays first on the plan form, because other attributes are shown or hidden by it. · '<label>' is already first. · '<label>' is already last. |
| Attribute page | Settings · Options · Preview — <Level> form · Behaves like the real form. Nothing you type here is saved. · Preview as a plan whose business unit includes: · Try saving · Shows what saving a real record would say. Nothing is saved. · Saving would work. · These values would be cleared: · <Type> attributes have no options. · Showing <n> of <total> — search to find the others. · Show more · Import from a spreadsheet · Download template |
| Option panel / validation | Add option · Edit option '<label>' · Label · Linked to · Shown when one of these <parent> options is chosen. · Enter a label. · Use 120 characters or fewer. · '<label>' is already an option of <attribute>. · <attribute> has no options — its type is <Type>. |
| Import | Import options into <label> · Download the template, fill one row per option<, with its <parent> option in the second column>, then upload it. Nothing is added until you confirm. · Choose a file · <n> options to add · <n> already present (skipped) · <n> rows with problems · Row <n> · and <n> more · Add <n> options (1: Add 1 option) · Nothing to add — every row is already an option or has a problem. · Choose a .csv or .xlsx file. An older .xls file can be saved as .xlsx in Excel. · The file is larger than 700 KB — split it into smaller files. · This file can't be read. Save it again as .csv or .xlsx and try again. · The file has <n> rows — import at most 5,000 rows at a time. · The file has no rows below the header. · Empty — enter an option. · Longer than 120 characters. · Appears twice in the file (also row <n>). · Enter the parent option in the second column. · Parent option '<label>' not found in <parent>. · Over the 5,000-option limit of this attribute. · Already an option — skipped. · Template: file "<label> – options template.csv", header "Option" or "Option,Parent option" |
| Attribute fields on forms | Needed to publish · Needed to submit · Choose… · Type to search · Showing 50 of <n> — keep typing to narrow down. · No option matches '<text>'. · Remove <label> · Choose a <parent> first. · <Label> set to '<option>', the only <label> for this <parent>. · No options set up for this <parent> — ask an administrator. · Shown because the plan's business unit includes <labels joined with " and ">. · More Shell attributes appear once the plan's business unit is chosen. · <Label> is inactive — its value is kept and can't be changed. · Inactive options stay on this record but can't be chosen again. · — |
| Split editor | Value · Share · <total> % — complete · <total> % — <left> % left · <total> % — <over> % too much · Split evenly · Shares split evenly again. |
| Clear dialog (004–007 forms) | Clear values that no longer fit? · <Label>: <options> · Clear and save · Keep editing |
| Check results (API) | Shares add up to <total> % — they must add up to 100.00 %. · Choose one of the options of <label>. · '<option>' is inactive — choose another option. · '<option>' doesn't belong to the chosen <parent> — choose another <label>. · Enter text. · Enter a whole number between -999,999,999,999 and 999,999,999,999. · Choose Yes or No. · Enter a share above 0 % with at most two decimals. · Each option can appear only once. · <Label> is inactive and can't be changed. · This Shell attribute no longer exists — reload the page. · <Label> belongs to the <Level> form. · Use 2,000 characters or fewer. |
| Currency form | New currency · Edit currency · Code · Three letters, as in ISO 4217 (EUR, JPY). · Name · Symbol · Rate (units per 1 USD) · 1 USD = <rate> <CODE> · 1 <CODE> = <inverse> USD · Rate last changed by <name> on <date> |
| Currency validation | Enter a three-letter code such as JPY. · A currency with the code '<CODE>' already exists. · Enter a name. · Use 80 characters or fewer. · A currency called '<name>' already exists. · Use 5 characters or fewer. · Enter a rate above 0, with at most six decimals. · Use a rate of 1,000,000 or less. |
| Finance page and form | Budget year · Rename budget · <n> items · Item · Code · Linked to · New finance budget item · Edit finance budget item · Budget name · Items with the same year and budget name form one finance budget. · Item name · Rename budget '<name>' · New name · All <n> items move with it.< <n> plans use it as their default finance budget and will show the new name.> |
| Finance validation | Choose a year from <first> to <last>. · Enter a budget name. · Use 120 characters or fewer. · Enter a name. · The budget '<budget>' already has an item called '<name>'. · Use 40 characters or fewer. · The code '<code>' is already used in <year> by '<item>'. · <year> already has a budget called '<name>'. |
| Restore (API) | Can't undo — there is now another <record type> called '<name>'. Restore it from the recycle bin instead. (001) · Can't undo — there is now another currency with the code '<CODE>'. Restore it from the recycle bin instead. · Can't restore — there is now another currency with the code '<CODE>'. · Can't restore — '<attribute>' no longer has options. · Can't undo — the code '<code>' is now used by '<item>'. Restore it from the recycle bin instead. |
| Options list (API) | Choose a number from 1 to 500. · Choose 0 or more. · Choose active, inactive or all. |

## Dependencies

### Uses (feature 001)

- Go: `actor`, `clock`, `errs` (`Validation`, `NotFound`, `VersionConflict`, `NameTaken`, `Problems`),
  `web` (`ReadJSON`, `WriteJSON`, `WriteError`, `Router`, `Wrap`), `store` (`Transactor`, `Get`, `List`,
  `Put`, `Delete`), `ids.Generator`, `record.Meta`, `history` (`Kind`, `Register`, `Log`, `Diff`,
  `MoveToBin`, `Snapshot`, `RestoreContext`, `DeleteResult`, `BinEntry.View`), `apitest`, `seed.Load`.
- Contract components: `Error` responses (`BadRequest`, `Unauthorized`, `NotFound`, `Conflict`,
  `Error`), `DeleteResult`, `ActorRef`, `RecordRef`, `FieldError`.
- Web: `PageHeader`, `Card`, `Button`/`LinkButton`, `StatusChip`, `CompletenessChip` (`MissingField`),
  `InfoTip`, `Tooltip`, `ActionsMenu`, `SidePanel`, `ConfirmDialog`, `useToast`, `EmptyState`, `Skeleton`,
  `ErrorState`, `FormErrorSummary`, `TextField`; `api`/`call`, `ApiError`, `conflictMessage`;
  `useUndoableDelete`, `useUnsavedChanges`, `usePageTitle`, `formatDate`, `formatDateTime`,
  `formatMoney`; `mockApi`, `renderWithProviders`, `renderRoutes`; e2e `signInAs`, `uniqueName`,
  `expectNoAxeViolations`.

### Expects from other features (optional at build time — 002 works without them)

| From | What | Effect in 002 |
|---|---|---|
| 003 | a `settings.ReferenceSource` for people (`Record` person, `Currency` = display currency); People and Teams cards in `features/settings/sections.ts` | "Display currency of n people"; their Settings cards |
| 004 | a `ReferenceSource` for plans (`ModelID`, `FinanceBudgetID`, `Attributes`, `Level` plan); plan route `/plans/:planId` | model and budget usage, plan links, BU of lower levels |
| 005 | sources for campaigns and initiatives (`PlanID`, `Attributes`) | attribute usage and marking impact |
| 006 | a source for activities (`PlanID`, `ActivityID`, `ActivityTypeID`, `Attributes`) | type usage, model impact |
| 007 | a source for line items (`PlanID`, `ActivityID`, `Currency`, `PlannedCost`, `FinanceBudgetItemID`, `Attributes`) | currency and item usage, rate impact |
| 008 | the Approval workflows card in `sections.ts` | its Settings card |

Each source is wired in `internal/app/app.go` with `settingsService.AddReferenceSource(...)` and reads
records in the bin with `history.SnapshotsInBin`.

### Provides to later features

| What | Name |
|---|---|
| Money (Go) | package `money`: `Amount`, `Money`, `Input`, `ParseAmount`, `ParseInput`, `Rate`, `Rates`, `ToUSD`, `FromUSD`, `ConvertSpread`, `SplitEvenly`, `Percent`, `PercentSplitEvenly`, `OfWhole`, `MulInt`, `Sum` |
| Money (contract) | schemas `Money` (responses), `MoneyInput` (requests), `Percent`, `ExchangeRate` |
| Settings readers (Go) | `settings.(*Service)`: `Model`, `Models`, `ActivityType`, `ActivityTypes`, `Attributes`, `Options`, `Option`, `Currency`, `Currencies`, `Rates`, `FinanceBudget`, `FinanceBudgetsOfYear`, `FinanceBudgetItem`, `FinanceBudgetItems`; `settings.CanChoose`, `settings.Visible`, `(Model).Allows`, `settings.Waterfall`, `settings.Goals` |
| Attribute values (Go) | `settings.AttributeValues`, `settings.Level` (+ constants), `(*Service).CheckAttributes` (`AttributeInput`, `AttributeResult`), `MissingAttributes`, `BusinessUnit`, `Form`, `DescribeAttribute`, `settings.Diff`, `settings.MissingField`, `settings.ClearedValue` |
| Usage hook (Go) | `settings.Reference`, `settings.ReferenceSource`, `(*Service).AddReferenceSource`; `history.(*Service).SnapshotsInBin` |
| Errors | codes `in_use`, `cannot_restore`; `errs.InUse`, `errs.CannotRestore` |
| Contract | `AttributeValues`, `AttributeValue`, `SplitShare`, `AttributeForm`, `FormSection`, `FormAttribute`, `FormOption`, `MissingField`, `ClearedValue`, `AttributeCheckResult`, `SettingRef`, `Usage`, `Lock`, `ChangeImpact`, `StatusUpdate`, `Model`, `ActivityType`, `Currency`, `FinanceBudget`, `FinanceBudgetItem`, `AttributeLevel`, `AttributeType`, `AttributeMarking` |
| Endpoints | `GET /api/models`, `/api/activity-types`, `/api/currencies`, `/api/finance-budgets?year=`, `/api/shell-attribute-forms/{level}`, `POST /api/shell-attribute-forms/{level}/check` |
| Web | components `ShellAttributeFields`, `ShellAttributeValues`, `ClearValuesDialog`, `rules.ts`, `ChoiceField`, `SplitEditor`, `MoneyField`, `DecimalField`, `CheckboxField`, `TextAreaField`, `Breadcrumb`, `SearchField`, `SegmentedFilter`, `LockedNote`; hooks `useModels`, `useActivityTypes`, `useCurrencies`, `useFinanceBudgets`, `useAttributeForm`, `SETTINGS_KEYS`; `lib/decimal.ts`, `lib/waterfall.ts`, `lib/text.ts`; `features/settings/sections.ts` |
| Seed | `seed.SettingsSeed` (ids by name) passed to the planning seed |

### How later features read Settings (binding for 003–012)

**Rules.**

1. **Go, server side**: a later feature reads Settings records only through the readers above and the
   attribute functions of *Forms and values* — never with `store.Get` on a Settings table and never through
   the HTTP API. It may declare its own small interface (for example `plans.Settings`, `activities.SettingsReader`)
   that `*settings.Service` satisfies directly or through a few-line adapter in `internal/app/app.go`; 002
   adds no aliases for the names other drafts used (mapping below).
2. **Same transaction**: readers are called inside the feature's own `s.tx.Read` / `s.tx.Write` with the
   context it received, so a check and the save see the same Settings (001's store joins the transaction).
3. **Never copy derived values** (D-041): store only ids (`modelId`, `activityTypeId`, option ids in
   `attributes`, currency codes, `financeBudgetId`, `financeBudgetItemId`) and typed amounts in their own
   currency; compute USD amounts with `Rates(ctx)` + `money.Rates.ToUSD`, display amounts with
   `money.Rates.FromUSD`, calculated goals with `settings.Waterfall(inq, model)` — at read time.
4. **Inactive settings**: offer only `Active` records for new choices; on save accept an inactive id only
   when `CanChoose(active, id, previousID)` is true (it is already on the record); label kept inactive ones
   "(inactive)".
5. **Shell attribute values**: keep them in the record's `attributes` field (`settings.AttributeValues`),
   check them on every save with `CheckAttributes` (its `Problems` → `validation_failed`, its `Values` are
   what to store, its `Cleared` were shown by `ClearValuesDialog` first), list what blocks publishing or
   submitting with `MissingAttributes`, log with `settings.Diff` and describe with `DescribeAttribute`;
   the plan's business unit comes from `BusinessUnit(ctx, plan.Attributes)`.
6. **Web**: read with the hooks of `src/api/settingsQueries.ts` (`staleTime: 0` for forms, so SC-008
   holds); render attribute fields with `ShellAttributeFields` and values with `ShellAttributeValues`;
   never compute money on the client except the feedback-only helpers (`lib/decimal.ts`, `lib/waterfall.ts`).
7. **Usage**: register one `settings.ReferenceSource` per record kind in `internal/app/app.go` (it lists
   live records and those in the bin through `history.SnapshotsInBin`); nothing else is needed for "Used
   by", in-use deletes and impact statements.

**Exported Go types** (`internal/settings/model.go`; JSON names as data-model.md; `record.Meta` embedded):

```go
type Level string          // LevelPlan "plan", LevelCampaign "campaign", LevelInitiative "initiative", LevelActivity "activity", LevelLineItem "lineItem"
type AttributeType string  // TypeSingleChoice, TypeMultipleChoice, TypeMultipleChoiceSplit, TypeYesNo, TypeText, TypeLongText, TypeNumber
type Marking string        // MarkingOptional, MarkingNeededToPublish, MarkingNeededToSubmit

type Model struct {
	ID, Name                    string
	InqToMqlRate, MqlToDealRate money.Percent
	AverageDealSize             money.Money // USD
	ActivityTypeIDs             []string
	Active                      bool
	record.Meta
}
type Goals struct{ INQ, MQL, Deals int64; Revenue money.Amount } // Revenue in USD
type ActivityType struct { ID, Name string; DefaultPlannedCost *money.Money; Active bool; record.Meta }
type Attribute struct {
	ID, Label, HelpText string
	Level               Level
	Type                AttributeType
	Marking             Marking
	Section             string
	Position            int
	ParentAttributeID   string   // "" when it depends on nothing
	Condition           []string // Business unit option ids; empty = always shown
	Role                string   // "businessUnit" or ""
	Active              bool
	record.Meta
}
func (a Attribute) IsChoice() bool
type Option struct { ID, AttributeID, Label string; Position int; ParentOptionIDs []string; Active bool; record.Meta }
type Currency struct { Code, Name, Symbol string; Rate money.Rate; RateChangedAt time.Time; RateChangedBy actor.Actor; Active bool; record.Meta }
type FinanceBudget struct { ID string; Year int; Name string; record.Meta }
type FinanceBudgetItem struct { ID, BudgetID, Name, Code string; Active bool; record.Meta }

type AttributeValues map[string]json.RawMessage // attribute id → value (data-model §9)
type AttributeInput struct { Values, Previous AttributeValues; BusinessUnit []string }
type AttributeResult struct {
	Problems []errs.FieldError // field "attributes.<id>"; block saving
	Cleared  []ClearedValue    // {AttributeID, Label string; OptionIDs, OptionLabels []string}
	Missing  []MissingField    // {Field, Label, Reason string}; block publishing / submitting
	Values   AttributeValues   // what to store
}
```

**Names in the drafts of 003–012 → what 002 provides.** The later feature's adapter or code uses the right
column; where the semantics differ, 002's rule is the one that holds.

| Draft name (feature) | 002 provides |
|---|---|
| `money.Parse` (004, 012) | `money.ParseAmount` |
| `Amount.Grouped()` (004), `money.Format(Amount)` (007), `Money.Text()` (006) | `Amount.Format()` "1,200,000.00", `Money.Format()` "120,000.00 USD" |
| `Amount.RoundCents()`, a rounding `ParseAmount` (006, 007) | none: typed amounts have at most two decimals (`MoneyInput`); `ParseAmount` answers `ErrInvalidAmount` and the caller shows "… with at most two decimals."; computed values are already rounded half-up by `ToUSD`, `FromUSD`, `OfWhole` |
| arithmetic on `money.Money` (005, 009) | arithmetic on `Money.Amount` (`Add`, `Sub`, `Neg`, `Cmp`, `Sign`, `IsZero`, `Sum`); the currency is the caller's |
| `money.SplitEvenly(total, n)` (010) · `settings.EvenSplit(n)` (012) | `(Amount).SplitEvenly(n)` · `money.PercentSplitEvenly(n)` |
| `currencies.Service.FromUSD(ctx, usd, code)` (009), conversions with `DisplayRateOf` (003) | `Rates(ctx)` then `rates.FromUSD(usd, code)`; or `Currency(ctx, code)` then `money.FromUSD(usd, c.Rate)` |
| currency list for `people.UseCurrencies` (003), active currencies (007, 009) | `Currencies(ctx)` (filter `Active` for new choices) |
| `GetModel`, `GetActivityType`, `ListActivityTypes` (006) | `Model`, `ActivityType`, `ActivityTypes` |
| `Model.AllowedActivityTypeIDs` (006), `AllowedActivityTypes(ctx, modelID)` (005) | `Model.ActivityTypeIDs`, `(Model).Allows(id)` |
| `AttributesForLevel(ctx, level)` (006), "attributes with options" (010, 012) | `Attributes(ctx, level)` + `Options(ctx, attributeID)`; or the whole form with `Form(ctx, a, level)` |
| `settings.BusinessUnitsOf(planAttributes)` (006) | `BusinessUnit(ctx, planValues)` → option ids (labels from `Option(ctx, id)`) |
| `settings.Shown(attr, businessUnits)` (006), `GroupChanges` (005) | `settings.Visible(attr, businessUnit)`; groups = `FormView` sections with `conditional` |
| `settings.CheckValues(attrs, before, after, bu)` (006), `CheckAttributes(ctx, level, values, stored, plan)` (005), line-item value check (007) | `CheckAttributes(ctx, level, AttributeInput{Values, Previous, BusinessUnit})` → `AttributeResult` |
| `settings.MissingValues` / `settings.Missing` (006), `MissingAttributes(ctx, level, values, plan)` (005) | `MissingAttributes(ctx, level, values, businessUnit)` → `[]MissingField` |
| `AttributeChanges(ctx, before, after)` (005) | `settings.Diff(beforeRecord, afterRecord)` (whole record; `attributes.<id>` per attribute) |
| `settings.DescribeValue(attr, value)` (006) | `DescribeAttribute(ctx, "attributes.<id>", value)` |
| `settings.Waterfall(model, inq)` (004, 006) | `settings.Waterfall(inq, model)` → `Goals`; working backwards from MQL or revenue stays 004's rule |
| `settings.OptionRef` (006) · API `OptionRef`, `ModelSummary`, `FinanceBudgetRef` (004) | API `SettingRef {id, name, active}` and `FormOption`; summary shapes of a plan are 004's own schemas |
| `GET /api/attributes?level=` (004) | `GET /api/shell-attribute-forms/{level}` (forms) or `GET /api/shell-attributes?level=` (lists) |
| line-item usage counters and `RateChangeImpact` (007), `plans.Service.References` (004) | `ReferenceSource` (007 fills `Currency`, `PlannedCost`, `FinanceBudgetItemID`, `Attributes`); 002 computes counts and impact |
| web `AttributeFields`, `AttributeValuesView`, `defaultAttributeValues(level)` (004–006) | `ShellAttributeFields` (props in *New shared components*), `ShellAttributeValues`, `{}` (a missing yes/no reads as No) |
| web `MoneyInput` (004, 006, 007), `@/lib/money` cents arithmetic (005) | `MoneyField`; `lib/decimal.ts` `toCents(amount: string): bigint` and `formatCents(cents: bigint): string` (feedback only) |
| web `waterfallFromInq` (006) | `lib/waterfall.ts` `waterfall(inq: bigint, model: {inqToMqlRate, mqlToDealRate, averageDealSize})` → `{mql, deals: bigint; revenue: string}` or `'tooLarge'` |

**Overlap with feature 003 (built in parallel).** 003's plan also creates `SearchField` and
`TextAreaField` (same purpose), `FilterPills` (= `SegmentedFilter`), `PageHeader` `breadcrumbs`
(= `Breadcrumb`), `Combobox` (= `ChoiceField`'s searchable mode) and the web error kind `inUse`. The branch
that merges second reuses what the first created and adds only missing props (`count` on the pills,
`marking`/`maxLength` on the text area); the consolidation step keeps one name per component (proposed:
003's `FilterPills` and `PageHeader breadcrumbs`, 002's `ChoiceField` built on 003's `Combobox`) and
updates the references in 004–012.

## Tests

### Go service tests (one per rule; names carry the FR)

| File | Tests |
|---|---|
| `platform/money/*_test.go` | `TestParseAmount_FormatsAndLimits_D010`, `TestAmountJSON_StringWithTwoDecimals_D010`, `TestToUSD_RoundsHalfUp_FR029` (1,000.00 EUR at 0.92 → 1,086.96; negative half away from zero), `TestFromUSD_RoundsHalfUp_D010`, `TestRateInverse_SixDecimals_FR028` (0.92 → 1.086957, 150 → 0.006667), `TestParseRate_Limits_FR028`, `TestConvertSpread_LastMonthTakesDifference_D024`, `TestSplitEvenly_RemainderLast_D024` (10,000.00/3), `TestPercentSplitEvenly_R10` (3 → 33.33, 33.33, 33.34), `TestOfWhole_HalfUp_FR010`, `TestMulInt_OutOfRange_D010`, `TestRates_UnknownCurrency_FR029` |
| `settings/waterfall_test.go` | `TestWaterfall_SeededModels_FR010_SC006` (data-model §2.1 table), `TestWaterfall_RoundsHalfUp_FR010`, `TestWaterfall_TooLarge_FR010` |
| `settings/service_models_test.go` | `TestCreateModel_ListedActive_FR009` (US1-1), `TestCreateModel_AllProblemsAtOnce_FR009` (US1-5), `TestCreateModel_RateBounds_FR009`, `TestCreateModel_DealSizeAboveZeroUSD_FR009`, `TestCreateModel_NameUniqueIgnoringCaseAndSpaces_FR005`, `TestCreateModel_InactiveTypeRefused_FR004`, `TestUpdateModel_LogsEachOldAndNewValue_FR006_FR011` (US1-4), `TestUpdateModel_KeepsInactiveTypeAlreadyAllowed_FR004`, `TestUpdateModel_VersionConflict_FR007`, `TestUpdateModel_NoChangeNoLog_FR006`, `TestModelImpact_RatesInUse_FR011`, `TestModelImpact_Unused_NoWarning_FR011`, `TestModelImpact_RemovedTypeInUse_FR012` (US1-6), `TestModelUsage_CountsRecycleBin_FR003`, `TestDeleteModel_InUseRefused_FR004` (US1-7), `TestDeleteModel_UnusedToBin_Undo_FR004`, `TestSetModelActive_LoggedAndReversible_FR004_FR006`, `TestModelRestore_NameTaken_FR004` |
| `settings/service_activity_types_test.go` | `TestCreateActivityType_AllowedInModel_FR013` (US5-1, model logged), `TestActivityType_Validation_FR013_FR005`, `TestUpdateActivityType_DefaultCostChangedAndRemoved_FR013`, `TestUpdateActivityType_ModelsEditedFromType_FR013`, `TestUpdateActivityType_LastTypeOfModelRefused_FR009`, `TestActivityTypeImpact_RemovedModel_FR012`, `TestActivityTypeUsage_ActivitiesAndModels_FR003` (US5-2), `TestDeleteActivityType_InUseRefused_FR004`, `TestDeleteActivityType_Unused_Undo_FR004` (US5-5) |
| `settings/service_attributes_test.go` | `TestCreateAttribute_ListedLastInSection_FR019` (US2-1), `TestCreateAttribute_OptionProblemsTogether_FR018` (US2-2), `TestCreateAttribute_LabelUniqueWithinLevel_FR016`, `TestCreateAttribute_HelpTextLimit_FR016`, `TestCreateAttribute_MarkingByLevel_FR017`, `TestCreateAttribute_YesNoAlwaysOptional_FR017`, `TestCreateAttribute_SplitOnlyActivityAndLineItem_FR017`, `TestCreateAttribute_DependentNeedsEarlierChoiceParent_FR022`, `TestCreateAttribute_UnlinkedOptionsAllListed_FR022` (US3-2), `TestCreateAttribute_ConditionUsesBusinessUnitOptions_FR023`, `TestRenameAttribute_LogKeepsOldAndNew_FR006` (US2-4), `TestUpdateAttribute_TypeAndLevelLockedWhenValues_FR021` (also a value only in the bin), `TestUpdateAttribute_TypeChangeRemovesOptions_FR021`, `TestUpdateAttribute_BecomeDependentLinksAllOptions_FR022`, `TestAttributeImpact_MarkingMakesIncomplete_FR021` (US2-5: 7 activities, hidden and D-034 ones not counted), `TestBusinessUnit_Locked_FR023`, `TestSetAttributeInactive_ParentWithActiveDependentRefused_FR004`, `TestAttributeUsage_RecordsAndBin_FR003_FR020` (US2-3), `TestDeleteAttribute_InUseRefused_FR004` (US2-7), `TestDeleteAttribute_WithOptions_Undo_FR004` |
| `settings/order_test.go` | `TestMoveAttribute_WithinSection_FR019` (US2-6), `TestMoveAttribute_EdgeOfSectionRefused_FR019`, `TestMoveSection_RenumbersLevel_FR019`, `TestMove_DependentStaysBelowParent_EdgeCase`, `TestMove_BusinessUnitStaysFirst_FR023`, `TestChangeSection_GoesLastInNewSection_FR019`, `TestMove_LogsEachChangedPlace_FR006` |
| `settings/service_options_test.go` | `TestAddOption_UniqueLabel_FR018`, `TestAddOption_DependentNeedsActiveParent_FR022`, `TestRenameOption_FR018` (US2-4), `TestMoveOption_FR018`, `TestSetOptionActive_FR018`, `TestDeleteOption_InUseRefused_FR004`, `TestDeleteOption_LastOptionRefused_FR018`, `TestDeleteOption_Unused_Undo_FR018`, `TestListOptions_SearchAnyPart_SC007` ("0220"), `TestListOptions_5000Paged_FR018`, `TestOptionUsage_PerOption_FR020` (US2-3) |
| `settings/service_import_test.go`, `importfile_test.go` | `TestImportCheck_RowOutcomes_FR031` (US6-1: 1,217 rows, 3 faulty), `TestImport_AddsOnlyNewValid_OneLogEntry_FR031` (US6-2), `TestImport_NeverRenamesOrReactivates_FR031`, `TestImport_DependentParentColumn_FR031`, `TestImport_RowAndSizeLimits_FR031`, `TestImport_NothingToAdd_FR031`, `TestReadRows_CSVCommaSemicolonBOM`, `TestReadRows_XLSXSharedAndInlineStrings`, `TestReadRows_RefusesXLS` |
| `settings/form_test.go` | `TestForm_SectionsInOrderOfFirstAttribute_FR019_FR024`, `TestCheckAttributes_ShapesAndLimits_FR016`, `TestCheckAttributes_DependentOptionsFilteredByParent_FR025` (US3-1), `TestCheckAttributes_ParentChangeClearsDependent_FR025` (US3-4), `TestCheckAttributes_HiddenKeptNotMissing_FR026` (US3-3), `TestCheckAttributes_InactiveKeptNeverBlocks_FR027`, `TestCheckAttributes_NewInactiveOptionRefused_FR004`, `TestCheckAttributes_SplitMustTotal100ToCount_FR027`, `TestCheckAttributes_NoOptionsForParentNotMissing_D034`, `TestMissingAttributes_MarkingPerLevel_FR017`, `TestBusinessUnit_FromPlanValues_FR023`, `TestDiff_AttributeByAttribute_FR006`, `TestDescribeAttribute_Texts_FR006` |
| `settings/service_currencies_test.go` | `TestCreateCurrency_JPY_FR028` (US4-1), `TestCreateCurrency_AllProblemsAtOnce_FR028` (US4-5), `TestCurrency_CodeLocked_FR028`, `TestUSD_RateStatusDeleteLocked_FR028` (US4-4), `TestCurrencyImpact_RateChange_FR030` (US4-3: 14 line items, 6 activities, 52,173.91 → 50,526.32), `TestUpdateCurrency_RateChangeLoggedWithDetails_FR006_FR030`, `TestRates_CurrentRateOfEveryCurrency_FR029`, `TestCurrencyUsage_LineItemsAndPeople_FR003` (US4-2), `TestDeleteCurrency_InUseRefused_FR004`, `TestDeleteCurrency_Unused_Undo_FR004`, `TestRestoreCurrency_CodeTaken_FR028` |
| `settings/service_finance_test.go` | `TestCreateItem_CreatesBudget_FR014` (US5-3), `TestCreateItem_Validation_FR014`, `TestCreateItem_JoinsBudgetIgnoringCase_FR014`, `TestUpdateItem_YearLocked_MovesBudget_FR014`, `TestRenameBudget_AllItemsFollow_FR015` (US5-4), `TestRenameBudget_NameTakenInYear_FR015`, `TestListBudgets_YearFilterAndGrouping_FR015`, `TestItemUsage_LineItemsAndPlans_FR003` (US5-4), `TestDeleteItem_InUseRefused_FR004`, `TestDeleteItem_Unused_Undo_FR004` (US5-5), `TestRestoreItem_CodeTaken_FR014` |
| `settings/service_test.go`, `kinds_test.go`, `usage_test.go` | `TestOverview_ActiveCounts_FR001`, `TestEveryChangeLogged_FR006` (create, change, inactivate, reactivate, delete, restore for each kind), `TestKinds_DescribeTexts_FR006`, `TestUsage_SummaryAndRecycleBin_FR003`, `TestUsage_SettingsInBinCount_EdgeCase` (restore never meets a missing setting) |
| `settings/readers_test.go` | `TestReaders_SortedWithoutBinRecords_FR007` (every reader; inside a caller's `Write`), `TestCanChoose_KeepsInactiveNeverChoosesNew_FR004`, `TestVisible_ConditionAnyBusinessUnit_FR023_FR026` |
| `history/service_test.go` | `TestSnapshotsInBin_KindAndContents_FR003` |
| `platform/errs/errs_test.go` | `TestInUseAndCannotRestore_409` |
| `seed/seed_test.go` | `TestSeed_Settings_FR032` (counts; Region → Cluster and GL code → Spend type links; Business unit role; markings and conditions as data-model §13; ids MOD-1001, SA-1001 …) |

### Contract and handler tests (`settings/handler_test.go`, package `settings_test`, `apitest`)

One test per operation of the API table, each checked against the contract: `TestOverview_API_FR001`,
`TestModelsAPI_ListCreateGet_FR009`, `TestModelsAPI_UpdateConflict_FR007`, `TestModelsAPI_Impact_FR011`,
`TestModelsAPI_Status_FR004`, `TestModelsAPI_DeleteAndUndo_FR004`, `TestActivityTypesAPI_*` (same six),
`TestShellAttributesAPI_ListByLevel_FR019`, `…_Create_FR016`, `…_Get_FR020`, `…_Update_FR021`,
`…_Impact_FR021`, `…_Status_FR023`, `…_Move_FR019`, `…_Delete_FR004`, `TestOptionsAPI_ListSearch_SC007`,
`…_Create_FR018`, `…_Update_FR018`, `…_Status_FR018`, `…_Move_FR018`, `…_Delete_FR004`,
`TestImportAPI_CheckAndImport_FR031`, `TestFormsAPI_Get_FR024`, `TestFormsAPI_Check_FR025`,
`TestCurrenciesAPI_*` (six), `TestFinanceAPI_ListYear_FR015`, `…_Rename_FR015`, `…_CreateGetUpdate_FR014`,
`…_Status_FR004`, `…_Delete_FR004`, `TestSettings_AnySignedInPersonMayChange_FR008` (Daniel edits Alex's
model). `app_test.go`: `TestRoutesMatchContract` and `TestEveryProtectedOperationRequiresSignIn_FR005`
cover the 47 new operations; `TestSettingsKindsRegistered_FR006`.

### Web component and page tests (Vitest)

| File | Covers |
|---|---|
| `lib/decimal.test.ts`, `lib/waterfall.test.ts`, `lib/text.test.ts` | parsing (`12.5k`, `12,500.00`, bad input), inverse rate, the waterfall table (same cases as Go), `midSentence` |
| `components/<Name>/<Name>.test.tsx` × 11 | each component's behaviour and accessible names; `ChoiceField`: select / check boxes / combobox switch at 10, search "0220" among 1,217, 50-row cap message, inactive chosen value; `SplitEditor`: totals, Split evenly, re-split note; `ShellAttributeFields`: sections and markings, help tip, dependent disabled → filtered → auto-fill message, D-034 note, hidden section keeps values, condition note, no-BU note, inactive read-only; `rules.test.ts` mirrors `form_test.go` |
| `features/settings/SettingsPage.test.tsx` | cards, counts, Coming soon cards, error state (FR-001) |
| `models/ModelsPage.test.tsx`, `ModelForm.test.tsx`, `WaterfallExample.test.tsx` | list, filters, details panel, create → toast with View, all problems at once, impact dialog, conflict message, Duplicate, delete unused vs in use, Set inactive (US1) |
| `activity-types/ActivityTypesPage.test.tsx` | create with models, default-cost note, impact, delete in use (US5) |
| `shell-attributes/ShellAttributesPage.test.tsx`, `ShellAttributeEditorPage.test.tsx`, `ShellAttributePage.test.tsx`, `OptionsCard.test.tsx`, `ImportOptionsDialog.test.tsx`, `AttributePreview.test.tsx` | level filter, grouped order and move buttons (keyboard, live region), editor problems together, locks, options search and paging, import review and add, preview Try saving (US2, US3, US6) |
| `currencies/CurrenciesPage.test.tsx` | rates both ways live, USD locks, impact dialog, problems at once (US4) |
| `finance-budget-items/FinanceBudgetItemsPage.test.tsx` | year filter, grouping, suggestions, rename budget, delete in use (US5) |
| `features/settings/shared/*.test.tsx` | delete dialog states, impact dialog, usage line |
| `app/router.test.tsx`, `app/navigation` | Settings routes and titles; Settings no longer "Coming soon" |

### End-to-end (`frontend/e2e/settings.spec.ts`, Playwright + axe; as `smoke.spec.ts`)

1. **US1** (Alex): Settings → cards → Models → New model "<unique>" (2 %, 5 %, 10,000, two types) → the
   example reads "250,000 INQ → 5,000 MQL → 250 deals → 2,500,000.00 USD revenue" and changes with the
   rate → Save → toast → View → details; empty save → three problems at once; Duplicate → "(copy)".
2. **US2** (Alex): Shell attributes → Activity → New "Media channel <unique>" (single choice, four options,
   Needed to submit) → last in "Shell attributes" → attribute page preview shows "Needed to submit" → set
   an option inactive → move the attribute up with the keyboard → delete it → Undo brings it back.
3. **US3** (Alex): Region's preview (campaign) → choose "Retail – China" → "Region set to 'RETAIL –
   CHINA', the only region for this cluster." → Brands' preview: untick Lubricants → Brands hidden; tick
   it → "Shown because the plan's business unit includes Lubricants."
4. **US4** (Alex): Currencies → New JPY "Japanese yen" "¥" 150 → listed Active; EUR → rate 0.95 → "1 EUR =
   1.052632 USD" → saved; USD shows the lock; "EURO", "EUR" and 0 → three problems at once.
5. **US5–US6** (Sara): Finance budget items → New "Europe – Retail <unique>" in a new budget → Rename
   budget; a new attribute "Cost centre <unique>" → import a generated CSV of 1,217 rows with 3 faulty →
   review counts → Add 1,214 options → search "0220" finds it.
6. **Accessibility**: axe (light and dark) on Settings, Models with the panel open, the attribute page with
   the preview, the import review, Currencies, Finance budget items; keyboard-only reorder (SC-009).

Journeys that need plans, activities or line items in use (US1-4, US1-6, US1-7, US2-3/5/7, US4-2/3, US5-2/4
in-use parts) are proven by Go tests with fake reference sources and by page tests with mocked answers;
features 004–007 add them to their end-to-end files once their records exist.

## Requirement traceability

| FR | Implemented in | Tested by |
|---|---|---|
| FR-001 | `SettingsPage`, `sections.ts`, `Overview` | `SettingsPage.test.tsx`, `TestOverview_*`, e2e 1 |
| FR-002 | every section page, shared parts, `Breadcrumb`, `SegmentedFilter`, `SearchField` | page tests, e2e 1–5 |
| FR-003 | `usage.go`, `UsageLine` | `TestUsage_*`, `*Usage_*` tests, page tests |
| FR-004 | `Set…Active`, `Delete…`, `CanChoose`, `SettingDeleteDialog` | `TestDelete*_InUseRefused`, `*_Unused_Undo`, `TestSet*Active*`, dialog tests, e2e 2 |
| FR-005 | `sameName` in every create/update | `*NameUnique*`, validation tests |
| FR-006 | `history.Log` in every method, kinds' `Describe`, `Diff` | `TestEveryChangeLogged_FR006`, `TestKinds_DescribeTexts_FR006`, `*Logs*` tests |
| FR-007 | `CheckVersion`, no caching, `staleTime: 0` forms | `*VersionConflict_FR007`, page conflict tests |
| FR-008 | no permission checks | `TestSettings_AnySignedInPersonMayChange_FR008` |
| FR-009 – FR-012 | models, `Waterfall`, `ModelForm`, `WaterfallExample`, impact | model tests, `waterfall_test.go`, `waterfall.test.ts`, e2e 1 |
| FR-013 | activity types | activity-type tests, page test |
| FR-014, FR-015 | finance budgets and items | finance tests, page test, e2e 5 |
| FR-016 – FR-023 | attributes, options, order, kinds, editor, attribute page | attribute, option, order tests, e2e 2–3 |
| FR-024 – FR-027 | `form.go`, `ShellAttributeFields`, `rules.ts`, `ChoiceField`, `SplitEditor` | `form_test.go`, component tests, e2e 3 |
| FR-028 – FR-030 | currencies, `money`, impact | currency and money tests, page test, e2e 4 |
| FR-031 | import | import tests, dialog test, e2e 5 |
| FR-032 | `seed.go` `addSettings` | `TestSeed_Settings_FR032` |
| SC-006, SC-007, SC-009 | waterfall; option search and `ChoiceField`; keyboard reorder and axe | waterfall tests; `TestListOptions_SearchAnyPart_SC007`, `ChoiceField.test.tsx`; e2e 6 |

## Complexity Tracking

| Choice or exception | Why needed | Simpler alternative rejected because |
|---|---|---|
| **Exception to constitution VI**: options removed by a type change are deleted permanently, not moved to the recycle bin (D-049) | FR-021 allows the change only while no record uses the attribute; the log entry keeps the labels | A restored option could not fit the new type; restoring them all would undo the type change silently |
| Reference sources (`settings.ReferenceSource`) | Usage, in-use checks and impact need records of 003–007 without import cycles | Each feature answering each question with its own callback would multiply hooks; storing counters would drift |
| `.xlsx` reader in the standard library (`importfile.go`) | FR-031 names Excel files; no new dependency (constitution VII) | A spreadsheet library adds a dependency for one sheet and two columns |
| 11 shared web components | 002 is the first feature with money, decimals, searchable choices, splits and Shell attributes on forms; 004–007 reuse each | Building them inside Settings would make 004–007 copy them |
| Impact endpoints (`…/impact`) | Spec requires the statement before saving (SC-005) and only the server knows the counts | Making the update fail with a "confirmation required" error would change 001's error shape |

## Consolidation (2026-10-19, binding)

The Plan-module consolidation (the cross-feature analyze step) settled the points below. They and the build guide,
[docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md), win where this plan, its data model or its tasks
say otherwise (D-200).

- **Contract (D-192).** The fragment is merged into `backend/api/openapi.yaml`; the merge task (T001) is done. The
  feature's operations are listed in `notYetServed` in `backend/internal/app/app_test.go`; the backend removes each one in
  the change that serves it. Agents do not change the contract; a problem with it is reported.
- **Decisions.** This plan's proposed decisions are numbered D-041–D-055 in `docs/sdd/decision-log.md` (labels kept in the
  table above); the consolidation decisions are D-189–D-207.

- **Owned shared parts (D-197).** 002 builds `Breadcrumb`, `SearchField`, `SegmentedFilter`, `MoneyField`, `DecimalField`,
  `CheckboxField`, `TextAreaField` (with 003's `rows?` prop), `ChoiceField`, `SplitEditor`, `LockedNote` and `ShellAttributeFields`
  (+ `ShellAttributeValues`, `ClearValuesDialog`, `rules.ts`). `ChoiceField`'s searchable mode (> 10 options) uses 003's
  `Combobox` (one component, `src/components/Combobox/`, designed in 003's plan). 003 builds no `FilterPills`, `SelectField` or
  `PageHeader` breadcrumbs. The *Overlap with feature 003* paragraph above is settled by this.
- **Contract (D-193).** `MissingField` is 002's, the one definition of the module, with `reason` **optional** (Go:
  `Reason string \`json:"reason,omitempty"\``). `ModelSummary`, `FinanceBudgetRef` and `OptionRef` are 004's schemas, built by
  004 from `Model`, `FinanceBudget` and `Option` (as the mapping table says).
- **Error codes.** `in_use` and `cannot_restore` are in the merged contract with 003's `not_allowed`, 004's `not_ready` (+ `missing`)
  and 011's `parent_deleted_permanently`; the web app's `ApiError` kinds `inUse`, `cannotRestore`, `notAllowed`, `notReady`,
  `parentDeletedPermanently` already exist (`frontend/src/api/errors.ts`). Go: add `errs.CodeInUse`/`InUse` and
  `CodeCannotRestore`/`CannotRestore` once (003 reuses them).
- **Typed amounts (D-190).** `MoneyInput` and `ParseAmount` refuse more than two decimals; the web forms of 006/007 round and say so
  before sending.
- **Usage sources.** 003-007 register `settings.ReferenceSource`s as in *Expects*; 004's source carries `Name` in `Record`; 007 fills
  `PlannedCost` in the line item's own currency.
- **Settings cards.** `features/settings/sections.ts` lists 002's five cards; 003 adds People and Teams, 008 Approval workflows.
- **Seed (D-202).** `addSettings` runs right after the people and returns `seed.SettingsSeed`; it writes no log entries.
