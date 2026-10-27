# Research: Backend Layout

## R1. Files over 300 non-blank lines

Counted with `grep -cv '^[[:space:]]*$'` on every Go file outside `internal/platform/database/sqlcdb`.

| | develop (9edb562) | this branch |
|---|---|---|
| Go files | 649 | 954 |
| Files over 300 non-blank lines | 113 | 0 |
| Largest file | 1,201 (`history/service_test.go`) | 297 |

Per package (Go files before → after, and how many were over 300 before):

| Package | Files | Over 300 | How it was split |
|---|---|---|---|
| settings | 60 → 123 | 20 | one file per Settings section and use case (`attributes_*`, `options_*`, `models_*`, `currencies_*`, `rates_*`, `finance_*`, `activity_types_*`, `form_*`, `kinds_*`, `handler_*`, `views_*`, `usage_*`); tests by the behaviour they prove |
| approvals | 51 → 86 | 14 | model by concept; rounds by use case (`rounds_submission`, `rounds_decide`, `rounds_recall`); workflows (`workflows_create`, `_levels`, `_list`, `_view`); delegations; one handler file per section |
| allwork | 42 → 61 | 11 | breakdown, export (notes, table), columns and cells, views and preferences, query and its model; tests and fixtures |
| people | 30 → 47 | 9 | API types per screen (`model_me`, `model_people`, `model_teams`); person and team forms and status; `me.go`; kinds and snapshots |
| campaigns | 41 → 57 | 8 | `create`, `read`, `delete`, `changes`; moves, destinations, move consequences, copies |
| history | 21 → 46 | 8 | log, delete, restore hooks, bin preview/restore/purge; tests by behaviour, three fixture files |
| activities | 47 → 64 | 7 | `create`, `get`, `list`, `update`, `reapproval`, `delete`, `plan_context`; split, its options, preview and links |
| plans | 41 → 58 | 6 | `create`, `get`, `delete`, `references`, `view`; model (figures, list); validation of fields |
| access | 23 → 33 | 5 | the route table by area of the API (`routes_people`, `_settings`, `_planning`, `_approvals`, `_views`), still one table; roles, assignments, grants |
| financials, budgets | 46 → 63 | 8 | services and tests by responsibility |
| home | 9 → 20 | 3 | one file per card (`summary`, `my_activities`, `plans_at_a_glance`, `waiting`, `recent_changes`) |
| lineitems | 34 → 48 | 3 | `create`, `duplicate`, `delete`, `get`, `view`, `section`, `form_options`; tests by use case |
| rollup, samples | 16 → 18 | 2 | the phasing view; notes |
| internal/app | — | 3 | the All work source (`allwork_year`, `_nodes`, `_attributes`, `_choices`, `_locate`), the approvals adapter (`approvals_preview`, `approvals_costs`, `activity_workflows`), line items (`lineitems_for_activities`, `lineitems_level`) |
| internal/platform/store | 2 → 11 | 2 | `transaction`, `tables`, `memo`, `commit`, `ready`; tests by behaviour |
| seed | — | 4 | Settings data, records, attributes and values; the history story by step; tests by story |

Every split moved code verbatim; the check compared the `func` and `type` declarations of each package with
its base and the number of tests (unchanged in every package). The only declarations that differ are the
lint fixes below.

## R2. The Store interfaces (FR-002)

Methods of the area's interface (develop's `Repository` → this branch's `Store` → the extra methods the
wiring and the seed use, in `internal/stores/<area>.go`):

| Area | Repository | Store | stores.<X>Store extra |
|---|---|---|---|
| access | 10 | 9 | 1 |
| activities | 14 | 10 | 4 |
| campaigns | 20 | 16 | 4 |
| lineitems | 10 | 8 | 2 |
| people | 17 | 15 | 2 |
| plans | 8 | 6 | 2 |
| allwork, approvals, auth, budgets, financials, history, samples, settings | 6, 19, 4, 10, 9, 6, 9, 45 | the same | 0 |

Every method of every `Store` is called through the service's store field in the area's service code
(checked by name for each area). Settings' 45 methods are all used: its service covers every Settings
section, and splitting that service is a change of design, not of layout.

Other areas' stores are no longer named inside an area: `auth.People`, `approvals.People` and the activities'
`approvalWorkflows` declare the one or two methods they read.

Constructors: every `NewService` takes `history.Tools` (transaction, IDs, activity log, clock) and at most
three more parameters; auth groups its demo settings and clock in `auth.Options`; `financials.LoadDemo`
takes a `DemoLoad`.

## R3. Lint on moved code

CI lints new and changed lines only, and a moved function counts as new, so the splits surfaced the old
findings of the code they moved (about 500 in all). They were fixed in this feature: shadowed errors,
repeated literals (named constants), unused parameters, dead code, and every function over the hard
complexity limit (cyclop 15+) or with 7+ parameters was given well-named helpers or a parameter struct.
What stays in the "needs justification" band carries a `//nolint` with its count and reason:

| | Code | Tests |
|---|---|---|
| `gocyclo` 10–14 | 9 (mostly "one Write whose every step returns its error") | 56 |
| `revive` argument-limit 5 | 4 (three match an interface's or caller's signature) | 3 |
| `tparallel` | — | 7 (subtests that share one fixture) |
| other (`gochecknoglobals` on the route table, `exhaustive`, `unparam`, `gosec`, `errorlint`) | 3 | 4 |

Fan-out (review only): approvals' `decisionOutcome` calls 12 distinct functions (down from about 25 in the
old `decide`).

## R4. P2: fakes instead of the JSON stand-in, and deleting the JSON store

Not done in this feature. What it takes:

- **Service tests.** 31 test files build their service on the JSON store (`store.Open(t.TempDir())`,
  `<area>.NewJSONStore()`), and many fixtures wire other areas' real services on it too (activities on
  plans and campaigns, line items on activities, ...). Moving them to fakes means a fake transactor, a
  fake activity-log store, and per area a fake of its `Store` (6 to 45 methods) plus fakes of the
  consumer interfaces it reads (`activities.PlanReader`, `plans.Settings`, ...).
- **Handler and contract tests.** 80 test files use `internal/platform/apitest`, which runs the whole API on
  `stores.JSON` with the demo data (`CopyDemoData`), including `internal/app`'s access matrix. Without the
  JSON store each needs fakes behind its own service (GO_STANDARDS §12), or an in-memory store set, which
  would be the JSON store again under another name. Only then can `internal/platform/store`, every
  `store_json*.go` and `stores.JSON` be deleted.

That is a rewrite of the backend's test infrastructure, larger than this feature's layout work, and it
changes how every handler test proves its contract; it should be its own feature (its own spec and
acceptance scenarios for what each test keeps proving).

## R5. P2: singular package names

A trial on a scratch branch renamed plans → plan, campaigns → campaign, activities → activity, lineitems →
lineitem, approvals → approval, budgets → budget and samples → sample (the count nouns; access, auth,
allwork, history, home, people, rollup, settings and financials name areas, not records):

- 61 local variables named like the new packages (35 `plan`, 11 `approval`, 8 `activity`, 7 `campaign`)
  and one test type must be renamed first, or they shadow the package; `gopls rename` does it safely.
- The rename then compiles; it touches 415 files (1,796 lines).
- CI's lint on new lines then reports 181 old findings on the lines the rename touched (50 govet shadow,
  39 gocyclo, 30 goconst, 23 cyclop, 21 revive, 7 tparallel, 4 funlen, ...): every function whose signature
  names a renamed package is re-linted.
