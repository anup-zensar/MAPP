# Implementation Plan: All Work Views

**Branch**: `010-all-work-views` | **Date**: 2026-10-19 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/010-all-work-views/spec.md`

**Design artifacts**: [data-model.md](data-model.md) · [contracts/openapi.yaml](contracts/openapi.yaml) ·
[tasks.md](tasks.md). Conventions are feature 001's ([plan](../001-app-shell-and-sign-in/plan.md),
[research](../001-app-shell-and-sign-in/research.md) R3–R11, R15, R18–R19); this plan states only what is
new.

## Summary

**All work** is a read-only, cross-plan view of one year: every plan, campaign, initiative, activity and
line item as a **Grid**, a **Calendar** and a **Budget & cost** view, under one filter bar, with six
summary cards, saved views and export to a spreadsheet or CSV file.

- **Go API** — a new package `internal/allwork`. One operation, `GET /api/all-work`, answers the whole
  view for a filter set: the shown rows in tree order with match/context roles, the requested cells,
  filtered roll-ups with their unfiltered values, summary cards, the "Showing N of M" count and the totals
  row; for Budget & cost also the period amounts. All matching, roll-ups, sorting, spreads and currency
  conversion run on the server (constitution IV); the web app never adds money. `GET /api/all-work/export`
  writes the same result as `.xlsx` or `.csv` with a standard-library spreadsheet writer
  (`internal/platform/sheet`, new). Saved views and the remembered columns are personal settings in two
  small tables, not business records.
- **Web app** — the `/all-work` page (`src/features/all-work/`): page header, summary cards, filter bar
  with chips, pill tabs, the tree grid with a pager, the column chooser, the Budget & cost and Calendar
  views, the Views menu and Export. Filters live in a per-person session store, so they survive moving
  around NorthStar; a new visit starts from the default saved view.

All work reads the records of features 002–008 through one adapter (`allwork.Source`), so the
calculations those features own (statuses, completeness, balances, spreads, goals) are reused, never
re-implemented.

## Technical Context

**Language/Version**: Go 1.27 (API); TypeScript 5.9 strict + React 19.3 (web app)

**Primary Dependencies**: no new dependency. API: standard library only (`archive/zip`, `encoding/xml`,
`encoding/csv` for files). Web app: the libraries of feature 001 (react-router, TanStack Query, radix-ui,
lucide-react, openapi-fetch). No grid or virtualisation library: rows are paged (Decision 3).

**Storage**: JSON file store; new tables `saved_views` and `all_work_preferences`; everything else is read
from the tables of features 002–008 inside one `store.Read`.

**Testing**: Go `testing` + `apitest` contract validation; Vitest + Testing Library; Playwright + axe.

**Target Platform**: as feature 001 (Windows laptops first; Chrome and Edge).

**Project Type**: web application (React SPA + Go REST API).

**Performance Goals**: SC-004 — with 5,000 activities and 20,000 line items in one year, `GET
/api/all-work` answers in under 1 second on a developer laptop and the page shows the result in under
2 seconds (50 rows per page rendered).

**Constraints**: `127.0.0.1` only; no personal data in server logs (the request log writes the path, never
the query string, so search text and IDs of the query are not logged); WCAG 2.1 AA (SC-008); usable at
1280 × 720 (the grid scrolls sideways inside its card).

**Scale/Scope**: 1 page route; 7 API paths (10 operations); 2 stored record types; ~25 web components and
helpers.

## Constitution Check

*GATE: checked before design and again after it.*

| Principle | How this plan complies | Result |
|---|---|---|
| I. Spec first | Every element traces to an FR, scenario or edge case (traceability table). The only open marker (FR-010) is resolved with the default of Q-009. Paging, the session store and the existence check on click serve FR-009, FR-016, SC-004 and an edge case; they are recorded as proposed decisions | Pass |
| II. Plain language | Shell's levels and words only; Sys_Gen_Balance appears only inside the row label the spec prescribes; every text is in *User-visible messages* | Pass |
| III. Simple, friendly | ZenReach page header, summary cards, pill tabs, chips with **Clear all**, result count; status always a word + colour (the chip moves into the name cell when the Status column is hidden); context rows say why they are shown; empty states with the next step; saved-view delete has Undo | Pass |
| IV. Layered architecture | One package `allwork` (handler → service → repository); all rules on the server; reading other packages only through `allwork.Source`; API in OpenAPI; generated web types | Pass |
| V. Tested rules | A Go test per rule named with its FR (matching, roll-ups, spreads, sorting, export, saved views), a contract test per operation, component tests per view, end-to-end tests for the P1 stories, a stress test for SC-004; fixed clock (2026-03-12 in `apitest`, 2026-10-18 in pure tests) | Pass |
| VI. Safe by default | Read-only on business records; the actor is passed to every service call; saved views are per person and never visible to others (FR-030); money exact (`money` package of 002); no personal data in logs | Pass |
| VII. Small and readable | No new dependency; one adapter interface (`Source`) justified in *Complexity Tracking*; the spreadsheet writer is ~200 lines of standard library | Pass |

**Post-design re-check**: Pass — no third runtime part, no new library, two small tables.

## Decisions

The spec's only `[NEEDS CLARIFICATION]` (FR-010) is resolved with the decision-log default of **Q-009**:
**only the filters of FR-004** — no Shell-attribute filters and no "Not filled in" choice in this phase.
FR-010 is therefore satisfied by *not* offering attribute filters; a test proves the filter bar offers
exactly the FR-004 filters.

Proposed decisions (numbered at consolidation, 2026-10-19):

| # | Proposed decision | Reason |
|---|---|---|
| D-113 | **Matching rules** (data-model §3): a filter tested on a record's own level must hold for it; a filter tested only on higher levels holds when the nearest ancestor at a tested level meets it; a filter tested only on lower levels holds when any descendant is a full match. Line items are not tested by filters; they follow their activity. Search is tested on plans, campaigns, initiatives and activities: a record meets it when its own name, ID, external name or description contains the text, when one of its line items does (activities), or when an ancestor does. Campaign / initiative status: values of one group (Approval progress, Timing) combine with OR, the two groups with AND | FR-005/FR-006 need one exact rule that tests and the three views share |
| D-114 | **The server computes the whole view** for each filter set (`GET /api/all-work`): rows in final order, cells, filtered and unfiltered roll-ups, totals, cards, and for Budget & cost the period amounts already converted to the display currency. The web app only expands, collapses, pages and hides unassigned rows | Constitution IV; totals must equal cards to the cent (SC-002, SC-003) |
| D-115 | **Rows are paged on screen**: the visible rows (after expansion) are shown 50 per page, with 100 and 200 as choices, in the Grid, Budget & cost and Calendar. The totals row closes every page; a page that starts inside a branch shows "Continued: <path>". Feature 012's "Select all N matching" builds on this pager | SC-004 with 5,000 activities; 012 FR-002 expects pages of 50 |
| D-116 | Container and total **Unassigned planned cost** = the sum of the matching activities' **positive** balances; negative balances are counted separately and flagged ("n activities have line items above their planned cost") | Keeps the column, the totals row and the summary card equal (FR-003, FR-014) |
| D-117 | **Plan budget** column = each container's own budget (plan, campaign or initiative budget); **Remaining budget** = own budget − the budgets of its children (plans and campaigns); **Budget used** = planned cost ÷ own budget, whole per cent rounded half up; totals sum plan rows only | Glossary "Plan budget", "Remaining budget"; FR-014 |
| D-118 | The **status chip is always visible**: it is in the Status column (a default column) and moves into the name cell when the user hides that column. A line item's Status is its completeness (FR-011); its SWBS request status is a Line item column | Product direction "status always visible"; FR-019 locks only Name |
| D-119 | Standard group adds an **Activities** column (number of matching activities below a container), because FR-013 lists the activity count as a roll-up | FR-013 / FR-018 consistency |
| D-120 | **Spreadsheet files are `.xlsx`** written by a small standard-library package `internal/platform/sheet` (`archive/zip` + `encoding/xml`); CSV is UTF-8 with a byte-order mark, comma-separated, notes first. Feature 012 adds the reader to the same package | No third-party dependency (constitution VII); Excel opens both |
| D-121 | **Saved views** (table `saved_views`, IDs `SV-1001`…) and **All work preferences** (remembered columns, order and sort; table `all_work_preferences`) are personal settings like D-038: no version, not in the activity log, not in the recycle bin. Deleting a view answers the deleted view, and **Undo** creates it again | Spec assumptions; 011 "saved views do not appear in the log" |
| D-122 | **Session state**: filters, search, view, expansion, page, page size and scroll position are kept in memory per person while the browser tab lives; a reload or a new sign-in is a new visit (FR-009). Opening `/all-work?<filters>` (feature 009's cards) applies those filters once and removes them from the address | FR-009, FR-016; 009 FR-007 |
| D-123 | Clicking a row first asks `GET /api/all-work/records/{recordId}`; a deleted record answers 404 with "This <level> was deleted — it is in the recycle bin" and the view refreshes | Spec edge case |
| D-124 | The Calendar shows plans as rows without a bar (a plan is its whole year); FR-028's Month layout is built last (P3) | FR-025, FR-028 |
| D-125 | A **new readable-ID prefix `SV`** for saved views | Module convention: every stored type gets a prefix |

## Project Structure

### Documentation (this feature)

```text
specs/010-all-work-views/
├── spec.md · checklists/requirements.md
├── plan.md              # this file
├── data-model.md        # stored types, matching rules, roll-ups, periods, columns
├── contracts/openapi.yaml   # fragment merged into backend/api/openapi.yaml
└── tasks.md
```

### Source code

```text
backend/
├── api/openapi.yaml                 # + the paths and schemas of contracts/openapi.yaml
├── internal/app/app.go              # + allwork repository, source, service, handler routes
├── seed/seed.go · seed_test.go      # + two saved views for Priya Raman
└── internal/
    ├── platform/sheet/              # NEW: spreadsheet files (010 writes, 012 adds reading)
    │   ├── sheet.go                 # Workbook, Sheet, Cell (text, number, money, date, bold)
    │   ├── xlsx_write.go · xlsx_write_test.go
    │   └── csv_write.go · csv_write_test.go
    └── allwork/                     # NEW
        ├── model.go                 # API types + stored SavedView, Preferences
        ├── query.go · query_test.go # Query: parse and validate url.Values → Query; FilterSet
        ├── source.go                # Source interface, YearData, Node (the input of everything below)
        ├── source_planning.go       # the Source built on packages 002–008 (the only file importing them)
        ├── tree.go · tree_test.go   # tree, match/context, search, Show levels, count
        ├── rollup.go · rollup_test.go   # roll-ups, unfiltered values, summary cards, totals
        ├── columns.go · columns_test.go # column catalogue and cell values
        ├── sorting.go · sorting_test.go
        ├── periods.go · periods_test.go # Budget & cost periods
        ├── export.go · export_test.go   # export file (uses platform/sheet)
        ├── views.go · views_test.go     # saved views and preferences rules
        ├── repository.go · store_json.go
        ├── service.go · service_test.go # Query, Options, Export, Locate, preferences, views
        ├── stress_test.go               # SC-004
        └── handler.go · handler_test.go

frontend/
├── e2e/all-work.spec.ts             # NEW
└── src/
    ├── app/router.tsx · navigation.ts   # /all-work real page; ALL_WORK leaves COMING_SOON_SECTIONS
    ├── lib/download.ts · download.test.ts   # NEW: save a file answered by the API (also used by 012)
    └── features/all-work/           # NEW
        ├── AllWorkPage.tsx (+ .module.css, .test.tsx)
        ├── allWorkApi.ts            # queries and mutations
        ├── allWorkStore.ts · allWorkStore.test.ts   # per-person session state (useSyncExternalStore)
        ├── viewState.ts · viewState.test.ts         # standard state, query params, equality, expansion
        ├── FilterBar.tsx · FilterMenu.tsx (+ css, tests)
        ├── SummaryCards.tsx (+ test)
        ├── AllWorkGrid.tsx · GridRow.tsx · CellValue.tsx · Pager.tsx (+ css, tests)
        ├── ColumnChooser.tsx (+ css, test)
        ├── BudgetView.tsx (+ css, test)
        ├── CalendarView.tsx · MonthCalendar.tsx (+ css, tests)
        ├── ViewsMenu.tsx · SaveViewDialog.tsx · ManageViewsPanel.tsx (+ tests)
        ├── ExportMenu.tsx (+ test)
        ├── recordHref.ts · recordHref.test.ts       # the address of a row's record
        └── Highlight.tsx (+ test)   # marks the searched text
```

**Structure decision**: the layout of feature 001. `allwork` is a read model over features 002–008 plus
two small personal tables; it registers no history kind.

## Backend design

### Request flow

`GET /api/all-work?…` → handler parses `url.Values` into `allwork.Query` (`ParseQuery`, all problems at
once) → `Service.Query(ctx, a, q)` → one `tx.Read`: `Source.DisplayCurrency(ctx, a.ID)`,
`Source.Year(ctx, year)` → pure functions `buildTree` → `applyFilters` → `rollUp` → `sortTree` →
`flatten` (+ `periods` for Budget & cost) → `Result`.

### `allwork.Source` (the only coupling to features 002–008)

```go
// Source reads everything All work shows, with the values the owning features calculate. It runs inside
// the caller's transaction. source_planning.go implements it on packages settings, people, plans,
// campaigns, activities and lineitems; tests use an in-memory Source.
type Source interface {
    Years(ctx context.Context) ([]int, error)                       // years that have plans
    Year(ctx context.Context, year int, today string) (YearData, error)
    Columns(ctx context.Context) ([]Column, error)                  // the catalogue, attributes included
    Currency(ctx context.Context, personID string) (Currency, error) // display currency + rate
    Options(ctx context.Context, year int, personID string) (Options, error)
    Locate(ctx context.Context, recordID string) (Location, bool, error)
    InRecycleBin(ctx context.Context, recordID string) (bool, error)
}
```

`YearData{Today string; Nodes []Node; Total map[Level]int}` — one `Node` per plan, campaign, initiative,
activity and line item of the year's plans (data-model §2), with: level, ID, parent ID, plan ID, name,
external name, description, owner (ID, name, inactive), dates and ongoing, the status chip, timing,
completeness (missing count), SWBS status, own budget, planned cost in USD, local planned cost, line-item
month amounts in USD (and the amount that cannot be spread yet), INQ/MQL/revenue, created/updated, plan
facts used by filters (status, strategic, team IDs, business-unit option IDs), activity type ID, activity
display status, the sibling order of the plan tree, and `Fields map[string]Cell` with the value of every
level-specific column of data-model §6. `Source` calls the functions listed in *Dependencies*; it never
computes a business rule itself.

### Service operations

All take `(ctx, a actor.Actor, …)`; reads run in one `tx.Read`, writes in one `tx.Write`.

| Operation | Rules (FR) |
|---|---|
| `Query(ctx, a, q Query) (Result, error)` | Year: `q.Year` or the current year (FR-004). Tree in plan-tree order, plans by name (FR-011). Match/context per data-model §3 (FR-005, FR-006); Show levels (FR-007); count "N of M" with the noun of data-model §3.5 (FR-008). Roll-ups from shown matches, `unfiltered` when records below are hidden (FR-013), unassigned rows for every activity with a non-zero balance (FR-012), totals = cards (FR-003, FR-014). Sort within each parent, empty last, statuses in lifecycle order; unassigned rows stay last (FR-015). Money in the display currency: sum in USD, convert once per shown value, half up (FR-017, edge case). Cells only for `q.Columns`; unknown columns and unknown filter IDs go to `leftOut` and are ignored (FR-031). View `calendar` drops line-item and unassigned rows (FR-025); view `budget` adds periods (FR-022 – FR-024). `generatedAt` = now (edge case "Updated at") |
| `Options(ctx, a, year *int) (Options, error)` | Years with plans + the current year, newest first (FR-004); plans of the year by name with status; owners of plans, campaigns, initiatives and activities of the year by name, "(inactive)" kept (FR-004, edge case); teams with the person's teams first (`mine: true`); business-unit options; activity types; the column catalogue (FR-018) |
| `Export(ctx, a, q Query, format) (File, error)` | The `Query` result with every row (collapsed branches included), Level and Path columns, the visible columns (or the view's fixed columns), the totals row, money as numbers in the display currency with the currency in the heading, local amounts with a Currency column, dates as dates, statuses as words, notes before the data; file name per FR-034 (FR-032 – FR-034) |
| `Locate(ctx, a, recordID) (Location, error)` | Known record → its level, name, plan ID and activity ID. Unknown → `not_found` with "This <level> was deleted — it is in the recycle bin" when an unresolved bin entry holds it, otherwise "This <level> no longer exists — someone may have deleted it." (edge case) |
| `Preferences(ctx, a) (Preferences, error)` | Stored record or the defaults: the default columns of FR-019, no sort (FR-019, FR-021) |
| `UpdatePreferences(ctx, a, p Preferences) (Preferences, error)` | Columns: at most 100, no duplicates, `name` never stored (always shown); sort column must be a catalogue key; saves only when something changed; not logged (FR-021, D-121) |
| `ListViews(ctx, a) ([]SavedView, error)` | Only `a`'s views, sorted by name ignoring letter case (FR-030) |
| `CreateView(ctx, a, in SavedViewCreate) (SavedView, error)` | Name required, trimmed, ≤ 60 characters, unique among `a`'s views ignoring case; state validated (data-model §1.2); `isDefault: true` clears `a`'s other default in the same Write (FR-029, FR-031) |
| `UpdateView(ctx, a, id, in SavedViewUpdate) (SavedView, error)` | Only `a`'s own view (another person's → `not_found`, never revealing it); name rules as create; `state` replaces the state (Update view); `isDefault` true/false (FR-030, FR-031) |
| `DeleteView(ctx, a, id) (SavedView, error)` | Only `a`'s own; answers the deleted view so the web app's **Undo** can create it again (FR-031) |

### Validation and error messages (API)

| Field / case | Message |
|---|---|
| `year` not 2000–2100 | Choose a year from 2000 to 2100. |
| `view` | Choose grid, calendar or budget. |
| `planStatus`, `containerStatus`, `activityStatus`, `strategic`, `showLevels`, `period`, `format` not in their lists | Choose one of the listed values. |
| `sort` not `<column>:asc` or `<column>:desc` with a catalogue column | Choose a column of the grid to sort by. |
| `search` over 200 characters | Use 200 characters or fewer. |
| `column` more than 100 | Choose at most 100 columns. |
| Saved view `name` empty / too long / taken | Enter a name for the view. / Use 60 characters or fewer. / You already have a view called '<name>'. |
| Saved view `state` invalid (enums, `columns` over 100) | The view can't be saved — reload the page and try again. |
| Saved view of another person or gone | This view no longer exists — it may have been deleted in another tab. |
| Preferences `columns` over 100 / duplicate | Choose at most 100 columns. / Each column can be chosen once. |
| Record gone (Locate) | This <level> was deleted — it is in the recycle bin · This <level> no longer exists — someone may have deleted it. |

All are `validation_failed` (field named) or `not_found`, as feature 001.

### History kinds and activity log

None. All work never changes business records (FR-002). Saved views and preferences are personal
settings and are not logged (spec assumptions, D-121).

### API operations

| Method and path | Request | Response | Errors |
|---|---|---|---|
| `GET /api/all-work` | Query: `year`, `view`, `plan`*, `planStatus`*, `containerStatus`*, `activityStatus`*, `owner`*, `team`*, `businessUnit`*, `activityType`*, `strategic`, `showLevels`, `search`, `column`* (ordered), `sort`, `period` (* = repeatable) | `200 AllWorkResult` | 400 `validation_failed`, 401 |
| `GET /api/all-work/options` | `year` (optional) | `200 AllWorkOptions` | 400, 401 |
| `GET /api/all-work/export` | the query of `GET /api/all-work` + `format` (`xlsx`\|`csv`, required) + `unassigned` (default `true`) | `200` file, `Content-Disposition: attachment` | 400, 401 |
| `GET /api/all-work/records/{recordId}` | — | `200 AllWorkRecordLocation` | 401, 404 |
| `GET /api/all-work/preferences` | — | `200 AllWorkPreferences` | 401 |
| `PUT /api/all-work/preferences` | `AllWorkPreferences` | `200 AllWorkPreferences` | 400, 401 |
| `GET /api/saved-views` | — | `200 SavedViewPage` | 401 |
| `POST /api/saved-views` | `SavedViewCreate` | `201 SavedView` | 400, 401 |
| `PATCH /api/saved-views/{viewId}` | `SavedViewUpdate` | `200 SavedView` | 400, 401, 404 |
| `DELETE /api/saved-views/{viewId}` | — | `200 SavedView` (the deleted view) | 401, 404 |

### Export file (FR-032 – FR-034)

- **Spreadsheet**: sheet 1 "About this export" (rows: View, Year, Filters — one line per chip or "None",
  Search, Show levels, Display currency, Exported by, Exported at "18 Oct 2026, 14:05 UTC"); sheet 2 named
  after the view ("Grid", "Calendar", "Budget & cost") with a bold heading row, then the rows, then the
  totals row. Money cells are numbers with format `#,##0.00`; dates are date cells with format
  `dd mmm yyyy`; percent cells are numbers.
- **CSV**: the same notes as `label,value` lines, an empty line, then the heading and rows; numbers
  without thousands separators ("12500.00"), dates `YYYY-MM-DD`.
- **Columns**: Level, Path ("FY2026 Lubricants Europe › Helix launch"), Name, then — Grid: the visible
  columns in order; Calendar: ID, Status, Start date, End date, Owner, Planned cost; Budget & cost: ID,
  Plan budget, the periods, Other years, No dates, Total. Money headings carry the currency ("Planned
  cost (USD)"); a local amount column is followed by "Currency". A "Shown for context" column ("Yes") is
  added when the export holds context rows. Unassigned rows follow the screen's switch (`unassigned`).
- **File name**: `NorthStar All work – <Grid|Calendar|Budget & cost> – <year> – <DD MMM YYYY>.<xlsx|csv>`
  in `filename*` (UTF-8) with an ASCII fallback using "-".

### Wiring (`internal/app/app.go`)

`allwork.NewJSONRepository()`; `allwork.NewPlanningSource(…the repositories/services listed in
Dependencies…, history)`; `allwork.NewService(tx, repo, source, generator, now)`;
`allwork.NewHandler(service).Routes(router, signedIn)`. No kinds to register.

### Demo data (`backend/seed/seed.go`)

Two saved views for Priya Raman (U-1002), neither default (so every visit starts from the standard view
in demonstrations and tests): **"Lubricants by month"** — Budget & cost, 2026, plan "FY2026 Lubricants
Europe", Monthly; **"Declined activities"** — Grid, 2026, Activity status Declined. `seed_test.go` checks
both and that Daniel Okoro has none.

## Frontend design

### Route and navigation

| Path | Page | Tab title |
|---|---|---|
| `/all-work` | `AllWorkPage` (replaces the Coming-soon page; `handle: { title: 'All work' }`) | All work · NorthStar |

`navigation.ts`: `ALL_WORK` becomes a `NavItem` and leaves `COMING_SOON_SECTIONS` (its sentence is
removed). The page reads `?year=…&owner=…&activityStatus=…` (the API's parameter names) once on arrival,
applies them over the standard view, and replaces the address with `/all-work` (D-122).

### State

- `allWorkStore.ts`: a module-level store keyed by person ID — `{ initialized, state: ViewState,
  appliedViewId: string | null, page, pageSize: 50 | 100 | 200, scrollY }` — read with
  `useSyncExternalStore` through `useAllWorkState()` (hook in a `.ts` file, per the lint rules). Signing
  out or another person clears nothing visible to them, because it is keyed by person.
- `viewState.ts` (pure, fully unit-tested): `standardState(prefs)`, `toQueryParams(state)`,
  `fromAddress(searchParams)`, `sameState(a, b)` (for "Modified"), `isExpanded(row, state, hits)`,
  `visibleRows(rows, state)`, `chipsOf(state, options)`, `leftOutOf(state, options)`.
- Query keys: `['all-work', params]` (`staleTime: Infinity`, `refetchOnWindowFocus: false`,
  `placeholderData: keepPreviousData` — old rows stay while new ones load), `['all-work-options', year]`,
  `['all-work-preferences']`, `['saved-views']`. **Refresh** refetches `['all-work', params]`.
- First visit of a session: wait for `['saved-views']` and `['all-work-preferences']`; apply the default
  view (dropping left-out values with the message) or the standard state; then mark the store
  initialized.
- Columns: while no saved view is applied, changing columns or sort saves `PUT /api/all-work/preferences`
  (debounced 500 ms). While a view is applied, changes only mark it **Modified**.

### Page layout (`AllWorkPage`)

1. `PageHeader`: eyebrow "All work", title "See all planned work", highlight "across plans", description
   "Filter, compare and export every campaign, initiative, activity and line item of a year."; actions:
   **Views** (secondary), **Export** (secondary). No primary button (nothing is created here).
2. `SummaryCards`: six `SummaryCard`s (Campaigns, Initiatives, Activities, Planned cost, Plan budget,
   Unassigned planned cost) with tips, compact money and full value (FR-003); the unassigned card adds the
   red negative-balance line.
3. `FilterBar`: search box (`TextField`, label "Search", placeholder "Search name, ID, external name or
   description", 300 ms debounce); a wrapping row of `FilterMenu` buttons — Year (single), Plan, Plan
   status, Campaign / initiative status, Activity status, Owner, Team, Business unit, Activity type,
   Strategic / AOP plans (single), Show levels (single); each button shows its count ("Owner · 2"). Below:
   chips (one per value, "<Filter>: <value>", remove button "Remove filter <Filter>: <value>"), **Clear
   all**, and the count "Showing N of M activities" (`aria-live="polite"`).
4. Pill tabs **Grid · Calendar · Budget & cost** (`PillTabs`, 004) and the view toolbar: Grid — **Expand
   all**, **Collapse all**, the switch "Unassigned planned cost" (`role="switch"`), **Columns**,
   **Refresh** and "Updated at 14:05"; Budget & cost — Expand/Collapse all, the unassigned switch,
   **Monthly · Quarterly**, Refresh; Calendar — **Timeline · Month**, Refresh.
5. The view in a `Card`, then the `Pager` ("Rows 1–50 of 236", **Previous**, **Next**, "Rows per page"
   50 · 100 · 200).

`FilterMenu` (local to the feature): a button opening a Radix Popover with a checklist (radio list for
single choices), a search box when more than 10 choices, inactive values labelled "(inactive)", Team with
a **My teams** choice first that ticks the person's teams. Filters apply as soon as a box changes (FR-009).

### Grid (`AllWorkGrid`, `GridRow`, `CellValue`)

- A `<table>` inside the card (horizontal scroll inside the card), `<caption class="visually-hidden">`
  "All work grid". Header: Name (always first, sticky left) then `result.columns`; each sortable heading is
  a button with `aria-sort`; clicks cycle ascending → descending → default (FR-015).
- Name cell: indentation by depth, `LevelTag` (004) bar and tag, an expand button for rows with children
  (accessible name "Expand <name>" / "Collapse <name>", `aria-expanded`), the name as a link (to
  `recordHref`), "AOP" tag on strategic plans, the status chip when the Status column is hidden (D-118),
  search highlight (`Highlight`) and, for a match in external name or description, a one-line excerpt.
  Context rows use `--ink-2` text and a small "Shown for context" label with an `InfoTip`.
- Rows: clicking anywhere except a control, or Enter on the link, calls `Locate` and then navigates;
  on 404 the API message is shown as a toast and `['all-work']` is invalidated (D-123). The unassigned row
  opens its activity (FR-016).
- `CellValue` renders an `AllWorkCell`: `empty` → "—"; money right-aligned with `formatMoney`, red with a
  minus sign when negative or `tone: problem`; `percent` "80 %", red above 100; `status` → `StatusChip`;
  `date` → `formatDate`; `dateTime` → `formatDateTime`; text cut to one line with the full text in a
  `Tooltip` on hover and focus (edge case); `inactive` adds " (inactive)"; `unfiltered` adds a "Filtered"
  mark (button with tooltip "Filters hide some records below. Without filters: <value>.").
- Unassigned rows: name "Unassigned planned cost (Sys_Gen_Balance)", amount in the Planned cost and
  Unassigned columns, red with a minus sign when negative; hidden when the switch is off (FR-012).
- Totals row (`<tfoot>`): "Total · N activities" and the totals cells (FR-014).
- Expansion (data-model §4): default plans, campaigns and initiatives open, activities closed; with a
  search, the paths to hits open and matching containers closed; **Expand all** / **Collapse all** set the
  mode; single toggles are kept in `state.expansion.toggled`.
- Pager: rows are the visible rows after expansion; page and scroll are restored on return (FR-016); a
  page starting inside a branch shows the line "Continued: <path>".

### Column chooser (`ColumnChooser`, FR-018 – FR-021)

`SidePanel` "Columns" (description "Choose and order the columns of the grid."): search box "Search
columns"; groups Standard, Plan, Campaign, Initiative, Activity, Line item from `options.columnGroups`,
each a list of checkboxes (inactive attributes labelled); **Name** ticked and disabled with the `InfoTip`
"The name column is always shown so you can tell rows apart"; "Shown columns, in order": the chosen
columns with a drag handle (HTML drag and drop) and **Move up** / **Move down** buttons (accessible names
"Move <label> up/down"); **Reset to default columns**. Changes apply at once.

### Budget & cost (`BudgetView`, FR-022 – FR-024)

Same rows and expansion as the Grid (line items and unassigned rows included); columns Name, Plan budget,
`result.periods` (months or quarters, then Other years and No dates when present), Total. Activity rows
with a negative balance show, under the name, "Line items exceed the planned cost by <X>" in red. Footer:
period totals, grand total, Plan budget of plan rows. Empty cells "—". Note above the table: "To change how
a cost is spread, open its line item". Rows open records as in the Grid.

### Calendar (`CalendarView`, `MonthCalendar`, FR-025 – FR-028)

- Timeline: a name column (level tag, name link, status chip) and a 12-month track (month headings
  Jan–Dec <year>). Bars for campaigns, initiatives and activities from `startDate` to `endDate` clipped to
  the year: left edge "Continues from <year − 1>", right edge "Continues into <year + 1>", ongoing → open
  right end and "Ongoing". Bar colour = status tone (activities: status; campaigns and initiatives:
  Approval progress); the bar shows name and status word when wide enough. Each bar is a link (focusable)
  with a `Tooltip`: name · ID, level · status, dates, owner, planned cost, "Change dates on the record's
  page" (FR-027, no dragging). Legend of tones with words. **Today** line when `result.today` is in the
  year. Below: "No dates yet (N)" with links.
- Month layout (P3, FR-028): `MonthCalendar` — Monday-to-Sunday grid of one month, ‹ › and **Today**,
  activities listed on the days they run (at most 3 per day, then "+ n more" opening the day's list).

### Views (`ViewsMenu`, `SaveViewDialog`, `ManageViewsPanel`, FR-029 – FR-031)

- **Views** button label: "Views" or "View: <name>", plus "Modified" when the state differs.
- Menu (Radix DropdownMenu, `modal={false}`): the views (active one ticked, "Default" tag); separator;
  **Save current view…**; when modified, **Update view** and **Save as new view…**; **Manage views…**;
  **Standard view** (back to the standard state).
- `SaveViewDialog`: Name (Required, max 60), "Use as my default view" checkbox, **Save view** / **Cancel**;
  a name the person already uses asks "Replace your view '<name>'?" → **Replace view** (PATCH that view's
  state) / **Cancel**.
- `ManageViewsPanel` (`SidePanel` "Your saved views"): per view **Set as default** / **Remove default**,
  **Rename** (inline dialog), **Delete** (`ConfirmDialog`), toast with **Undo** that re-creates it.
- Applying a view: its state replaces the current one; values unknown to `options` (or reported in
  `leftOut`) are dropped and named in a toast.

### Export (`ExportMenu`, FR-032 – FR-034)

DropdownMenu **Export** → **Spreadsheet (.xlsx)**, **CSV (.csv)**. With `count.shown = 0` and no rows it
shows the toast "Nothing to export — clear a filter first." and calls nothing. Otherwise
`downloadFromApi(api.GET('/api/all-work/export', { params, parseAs: 'blob' }))` (`lib/download.ts`) saves
the file under the name of `Content-Disposition`; while waiting the item reads "Preparing your file…";
then the toast "Exported '<file name>'".

### States

| State | Shown as |
|---|---|
| Loading (first load) | `Skeleton` for the cards and 8 grid lines |
| Refreshing / changing filters | the previous rows stay, with "Updating…" next to the count (`aria-busy` on the card) |
| Error | `ErrorState` with **Try again** (the API message, or the unreachable sentence) |
| No plans in the year | `EmptyState` "No plans in <year> yet" · "Plans of <year> appear here once they're created." · **Go to Plans** |
| Nothing matches | `EmptyState` "Nothing matches these filters" · "Try removing a filter or changing the search." · **Clear filters** |
| Validation (query) | toast with the API message; the offending filter is reset |
| Saved view form errors | under the field and `FormErrorSummary` |
| Saved view gone (404) | toast "This view no longer exists — it may have been deleted in another tab." and the list refetches |
| Record gone on click (404) | toast with the API message; the view refetches |
| Left-out values | toast "Some parts of this view no longer exist and were left out: <list>." |
| Conflict | none — saved views and preferences have no version (D-121) |

### User-visible messages

| Where | Text |
|---|---|
| Page header | All work · See all planned work · across plans · Filter, compare and export every campaign, initiative, activity and line item of a year. |
| Tabs | Grid · Calendar · Budget & cost |
| Summary cards | Campaigns · Initiatives · Activities · Planned cost · Plan budget · Unassigned planned cost |
| Card tips | Campaigns that match the filters. Campaigns shown only for context are not counted. · Initiatives that match the filters. Initiatives shown only for context are not counted. · Activities that match the filters and search. · The sum of the matching activities' planned cost. An activity without a planned cost counts as zero. · The sum of the budgets of the matching plans. · Planned cost not yet split into line items: the sum of the matching activities' positive balances. |
| Negative balances | 1 activity has line items above its planned cost · <n> activities have line items above their planned cost |
| Search | Search · Search name, ID, external name or description |
| Filters | Year · Plan · Plan status · Campaign / initiative status · Activity status · Owner · Team · Business unit · Activity type · Strategic / AOP plans · Show levels · My teams · Search filters |
| Filter choices | Draft · Published · No activities · In progress · All approved · Not started · Running · Finished · Created · Submitted · Declined · Approved · In progress · Complete · Include · Only · Exclude · Everything · Complete branches only · Campaigns without initiatives · Initiatives without activities |
| Chips | <Filter>: <value> · Remove filter <Filter>: <value> · Clear all |
| Count | Showing <n> of <m> activities (campaigns · initiatives; "activity" when m is 1) · Updating… |
| Toolbar | Expand all · Collapse all · Unassigned planned cost · Columns · Refresh · Updated at <HH:mm> · Monthly · Quarterly · Timeline · Month |
| Grid | Name · Expand <name> · Collapse <name> · AOP · Shown for context · This row doesn't match the filters; it's shown so you can see where the matching rows are. · Filtered · Filters hide some records below. Without filters: <value>. · Unassigned planned cost (Sys_Gen_Balance) · Total · <n> activities · Continued: <path> · All work grid |
| Level tags | PLAN · CAMPAIGN · INITIATIVE · ACTIVITY · LINE ITEM |
| Pager | Rows <a>–<b> of <n> · Previous · Next · Rows per page |
| Column chooser | Columns · Choose and order the columns of the grid. · Search columns · Standard · Plan · Campaign · Initiative · Activity · Line item · The name column is always shown so you can tell rows apart · Shown columns, in order · Move <label> up · Move <label> down · Reset to default columns |
| Budget & cost | Plan budget · Other years · No dates · Total · Q1 · Q2 · Q3 · Q4 · To change how a cost is spread, open its line item · Line items exceed the planned cost by <X> |
| Calendar | Today · Ongoing · Continues from <year> · Continues into <year> · No dates yet (<n>) · Status · Change dates on the record's page · From <date>, ongoing · Owner: <name> · Planned cost: <money> · Previous month · Next month · + <n> more · Mon · Tue · Wed · Thu · Fri · Sat · Sun |
| Views | Views · View: <name> · Modified · Default · Save current view… · Update view · Save as new view… · Manage views… · Standard view · No saved views yet — set up filters and columns, then choose Save current view. |
| Save dialog | Save current view · Name · Use as my default view · Save view · Cancel · Replace your view '<name>'? · It will show what you see now. · Replace view |
| Manage | Your saved views · Set as default · Remove default · Rename · Rename view · Delete · Delete view '<name>'? · Only your saved view is deleted; no plans or records change. · Delete view |
| View toasts | View '<name>' saved · View '<name>' updated · View '<name>' renamed · '<name>' is now your default view · '<name>' is no longer your default view · View '<name>' deleted · **Undo** (alt text: Save the view again) · View '<name>' restored · Some parts of this view no longer exist and were left out: <list>. |
| Export | Export · Spreadsheet (.xlsx) · CSV (.csv) · Preparing your file… · Exported '<file name>' · Nothing to export — clear a filter first. |
| Export file | About this export · View · Year · Filters · None · Search · Show levels · Display currency · Exported by · Exported at · Level · Path · Shown for context · Yes · Currency |
| Empty states | No plans in <year> yet · Plans of <year> appear here once they're created. · Go to Plans · Nothing matches these filters · Try removing a filter or changing the search. · Clear filters |
| Record gone | This <level> was deleted — it is in the recycle bin · This <level> no longer exists — someone may have deleted it. |
| Saved view gone | This view no longer exists — it may have been deleted in another tab. |
| Validation | the API table above |

## Tests

### Go — rules (pure functions use a hand-built `YearData` dated 2026-10-18)

| Test | Proves |
|---|---|
| `tree_test.go` `TestTree_PlansByNameChildrenInPlanTreeOrder_FR011`, `TestFilter_EachFilterOnItsLevel_FR004` (table: one case per filter), `TestFilter_OnlyFR004Filters_FR010` (the query parser refuses any other filter name) | FR-004, FR-010, FR-011 |
| `TestFilter_ActivityStatusShowsOnlyBranchesToMatches_FR005`, `TestFilter_OwnerParentsAreContext_FR005`, `TestFilter_HigherLevelFilterInheritedFromNearestAncestor_FR005`, `TestFilter_ContainerStatusOrWithinGroupAndAcross_FR005` | FR-005 |
| `TestSearch_MatchesNameIdExternalNameDescription_IgnoringCase_FR006`, `TestSearch_LineItemHitMakesActivityMatch_FR006`, `TestSearch_MatchingContainerShowsEverythingInside_FR006` | FR-006 |
| `TestShowLevels_CompleteBranchesOnly_FR007`, `TestShowLevels_CampaignsWithoutInitiatives_FR007`, `TestShowLevels_InitiativesWithoutActivities_FR007` | FR-007 |
| `TestCount_ShowingNOfMWithNoun_FR008` | FR-008 |
| `rollup_test.go` `TestRollup_ContainersSumMatchesShown_FR013`, `TestRollup_UnfilteredValueWhenRecordsHidden_FR013`, `TestRollup_TotalsEqualSummaryCards_FR014_SC003` (property over every filter combination of the seed-like data), `TestSummary_SixCards_FR003`, `TestSummary_UnassignedPositiveOnlyAndNegativeCount_FR003`, `TestRollup_EmptyPlannedCostCountsZeroAllEmptyIsDash_EdgeCase` | FR-003, FR-013, FR-014, SC-002 |
| `TestUnassignedRows_OnlyNonZeroBalances_FR012` | FR-012 |
| `sorting_test.go` `TestSort_ChildrenOfEveryParentAscDescDefault_FR015`, `TestSort_EmptyLast_StatusLifecycle_UnassignedStays_FR015` | FR-015 |
| `TestMoney_DisplayCurrency_SumInUSDThenConvert_FR017` (EUR and GBP line items, display EUR: totals do not drift) | FR-017, edge case |
| `columns_test.go` `TestColumns_StandardGroupAndLevelGroups_FR018` (no "later" fields; new active attribute appears), `TestColumns_DefaultColumns_FR019`, `TestCells_BlankOnOtherLevels_DashOnOwnLevel_SplitShares_FR020`, `TestCells_InactiveValuesLabelled_EdgeCase` | FR-018 – FR-020 |
| `periods_test.go` `TestPeriods_EvenlyFirstLastCustom_FR023` (10,000.00 over Mar–May → 3,333.33 / 3,333.33 / 3,333.34), `TestPeriods_PositiveUnassignedSpreadEvenly_OngoingFirstMonth_FR023`, `TestPeriods_NegativeBalanceNotSpread_FR023`, `TestPeriods_OtherYearsAndNoDatesOnlyWhenUsed_FR022`, `TestPeriods_QuarterIsSumOfMonths_FR022`, `TestPeriods_ContainersAndFooter_GrandTotalEqualsPlannedCostCard_FR024` | FR-022 – FR-024, SC-002 |
| `service_test.go` `TestQuery_CalendarViewHasNoLineItemRows_FR025`, `TestQuery_TodayAndDatesForCalendar_FR026`, `TestQuery_DefaultYearIsCurrentYear_FR004`, `TestQuery_UnknownIdsAndColumnsLeftOut_FR031`, `TestQuery_NeverWrites_FR002` (store unchanged, no log entry) | FR-002, FR-004, FR-025, FR-026, FR-031 |
| `TestOptions_YearsOwnersTeamsMineFirst_FR004` | FR-004 |
| `TestLocate_DeletedRecordInRecycleBin_EdgeCase`, `TestLocate_UnknownRecord_EdgeCase` | edge case, FR-016 |
| `views_test.go` `TestCreateView_NameRequiredMax60UniquePerPersonIgnoringCase_FR029`, `TestViews_PersonalOnly_FR030` (Daniel lists none of Priya's; PATCH/DELETE of Priya's view by Daniel → not_found), `TestViews_OneDefaultPerPerson_FR031`, `TestViews_RenameUpdateStateDelete_FR031`, `TestViews_NotLogged_D_i` | FR-029 – FR-031 |
| `TestPreferences_DefaultsAndRemembered_FR019_FR021`, `TestPreferences_RefusesDuplicatesAndTooMany_FR021` | FR-019, FR-021 |
| `export_test.go` `TestExport_GridHasEveryRowLevelPathColumnsTotals_FR032` (collapsed rows included), `TestExport_MoneyNumbersCurrencyHeadingsDatesStatusWords_FR033`, `TestExport_NotesBeforeData_AboutSheet_FileName_FR034`, `TestExport_CalendarAndBudgetColumns_FR032`, `TestExport_UnassignedFollowsSwitch_FR012` | FR-032 – FR-034 |
| `platform/sheet` `TestXLSX_OpensAsZipWithWorkbookSheetsStrings`, `TestXLSX_NumbersDatesBoldHeadings`, `TestXLSX_EscapesTextAndSheetNames`, `TestCSV_BOMNotesThenTable` | D-120 |
| `stress_test.go` `TestQuery_StressYear_SC004` (5,000 activities, 20,000 line items, 20 plans; every view; < 1 s; skipped with `-short`) | SC-004 |

### Go — handler and contract (`handler_test.go`, `package allwork_test`, `apitest` with the seed)

`TestGetAllWork_Contract` (default, every view, every filter parameter, `column` order kept, `sort`),
`TestGetAllWork_InvalidParameters_FR004` (all problems at once), `TestGetOptions_Contract`,
`TestExport_XlsxAndCsvContentTypeAndDisposition_FR034`, `TestLocate_Contract` (200, 404 message),
`TestPreferences_GetPut_Contract`, `TestSavedViews_CRUD_Contract` (201, 200, 404 for another person),
`TestEveryAllWorkOperationRequiresSignIn`. `TestRoutesMatchContract` (feature 001) covers route coverage.

### Web — component and unit tests (Vitest, `mockApi`)

| File | Proves |
|---|---|
| `viewState.test.ts` | standard state (FR-009), query params incl. repeated and ordered `column` (FR-018), address filters (D-122), equality for "Modified" (FR-030), expansion defaults/search/expand all (FR-006, FR-011), visible rows + paging, chips and Clear all keep the year (FR-008), left-out values (FR-031) |
| `allWorkStore.test.ts` | state kept between page visits and per person; page and scroll kept (FR-009, FR-016) |
| `AllWorkPage.test.tsx` | header, six cards with tips and full values (FR-001, FR-003); default view applied on first visit and standard view otherwise (FR-009); address filters applied once (D-122); tab switch keeps filters, search and expansion (FR-001); loading, error + Try again, no plans, nothing matches + Clear filters; Refresh and "Updated at" |
| `FilterBar.test.tsx` | every FR-004 filter and nothing else (FR-010), immediate apply (FR-009), My teams first, chips + remove + Clear all, count text (FR-008) |
| `AllWorkGrid.test.tsx` | level tags, links, AOP, status chip moves into the name when Status is hidden (D-118), context rows labelled (FR-005), highlight and excerpt (FR-006), expand/collapse by mouse and keyboard and Expand/Collapse all (FR-011), unassigned rows and switch (FR-012), Filtered mark tip (FR-013), totals row (FR-014), sort cycle and `aria-sort` (FR-015), row click → Locate → navigate; 404 → toast + refetch (FR-016, edge case), money in display currency with local column (FR-017), pager and "Continued:" |
| `ColumnChooser.test.tsx` | groups and search, Name locked with reason, tick adds a column, reorder by Move up/down and by drag, Reset (FR-018, FR-019, FR-021); preferences saved only without an applied view |
| `BudgetView.test.tsx` | Plan budget column, period columns incl. Other years/No dates, Monthly/Quarterly refetch, negative-balance text, footer, read-only note, row opens record (FR-022 – FR-024) |
| `CalendarView.test.tsx` | bars positioned by dates, status colour + word + legend, Today line (fixed date 2026-10-18), Ongoing open end, Continues from/into, No dates yet list, tooltip content, not draggable (FR-025 – FR-027) |
| `MonthCalendar.test.tsx` | Monday-first grid, ‹ › and Today, activities on their days, "+ n more" (FR-028) |
| `ViewsMenu.test.tsx` / `SaveViewDialog.test.tsx` / `ManageViewsPanel.test.tsx` | save (name rules, replace question, toast), active view and Modified with Update / Save as new, default set and unset, rename, delete + Undo re-creates, left-out toast (FR-029 – FR-031) |
| `ExportMenu.test.tsx` | both formats call the export with the current query, file saved with the server's name, nothing-to-export message (FR-032 – FR-034) |
| `recordHref.test.ts`, `Highlight.test.tsx`, `lib/download.test.ts` | addresses per level; marked text; blob saved, API error turned into `ApiError` |

### End-to-end (`frontend/e2e/all-work.spec.ts`, real API with the seed, as `smoke.spec.ts`)

1. **US1** (Priya): open All work → both 2026 plans, activities collapsed; the totals row's planned cost
   equals the Planned cost card; expand an activity with a non-zero balance → its unassigned row; open a
   campaign row, go Back → filters, expansion and page restored (FR-001, FR-011 – FR-016).
2. **US2**: type an activity ID → one row, highlighted, "Showing 1 of N activities"; Activity status
   Declined → chip, only branches to declined activities; Clear all (FR-004 – FR-008).
3. **US4**: Budget & cost → Quarterly → footer total equals the Planned cost card (FR-022 – FR-024).
4. **US6**: save "E2E view <unique>" as default, sign in again → applied; sign in as Daniel → not listed;
   delete it with Undo (FR-029 – FR-031).
5. **US7**: Export → Spreadsheet → a download named "NorthStar All work – Grid – …xlsx" (FR-034).
6. axe (WCAG 2.1 AA) on the Grid, Calendar, Budget & cost, the column chooser and the Views menu; the Grid
   used with the keyboard alone (SC-008).

## Dependencies

### Uses (expected names — the consolidation step maps them if an owning plan names them differently)

| From | What |
|---|---|
| 001 | `store.Transactor`, `ids.Generator` (prefix `SV`), `clock.Clock`, `actor`, `errs` (`Validation`, `NotFound`, `Problems`), `web` (`ReadJSON`, `WriteJSON`, `WriteError`, `Router`, `Wrap`), `apitest`; `history.Service.ListBin` (Locate); web: `PageHeader`, `SummaryCard`, `Card`, `StatusChip`, `InfoTip`, `Tooltip`, `SidePanel`, `ConfirmDialog`, `useToast`, `EmptyState`, `ErrorState`, `Skeleton`, `FormErrorSummary`, `TextField`, `Button`, `ActionsMenu`, `lib/format`, `usePageTitle`, `api`, `call`, `ApiError` |
| 002 | Go `money`: `money.Amount` (exact, 2 decimals, `String()` "1234.50", `Add`, `Sub`, `Sign`, `IsZero`), `money.Money{Amount, Currency}` with JSON `{"amount","currency"}`, `money.Rate`, `money.FromUSD(usd Amount, r Rate) Amount` (half up), `money.SplitEvenly(total Amount, n int) []Amount` (rounded down, remainder last). `settings`: currencies with rates, activity types, models (names), workflows (names), Shell attributes with options, levels, types, business-unit conditions and active flags, finance budget items; the plan-level Business unit attribute. OpenAPI schema `Money` |
| 003 | `people` person names and active flag, the person's teams (`My teams`), teams with names and active flag, the person's display currency (`Preferences.DisplayCurrency`, default USD); web: date format applied by `lib/format` |
| 004 | `plans` records (name, year, status, strategic, owner, teams, business unit, budget, model, goal fields, description, defaults, BCPIR, SMP, created/updated) and `plans` completeness (own missing count); web: `LevelTag` (level bar + tag) and `PillTabs` shared components, plan page address `/plans/{planId}` |
| 005 | `campaigns` campaign and initiative records (name, owner, dates, budget, description, attribute values, parent, sibling order), their Approval progress and Timing functions, completeness; addresses `/plans/{planId}/campaigns/{campaignId}` and `/plans/{planId}/initiatives/{initiativeId}` |
| 006 | `activities` records (name, external name, type, owner, dates, ongoing, planned cost, workflow, description, attributes with splits, INQ / set later stages / MQL / revenue), `activities.DisplayStatus(activity, today)` (six statuses), completeness, sibling order; address `/activities/{activityId}` |
| 007 | `lineitems` records (name, owner, description, currency, local planned cost, spread, attributes, contracted amount, SWBS request, finance budget year/item), `lineitems.USDPlannedCost`, `lineitems.MonthAmountsUSD(item, activity, rate)` (month amounts in USD with the rounding rule of §4.4, plus the amount that cannot be spread while the activity has no dates), the balance `activity planned cost − Σ line items`, completeness, SWBS status; address `/line-items/{lineItemId}` |
| 008 | nothing beyond the activity status (stored by 006) |

### Provides to later features

- `GET /api/all-work` and `AllWorkResult` rows with `role`, `recordId`, `level` and `planId`, which feature
  012 uses for selection and "Select all N matching"; `AllWorkGrid` accepts a `selection` prop (added by
  012) and exposes the visible page rows.
- `internal/platform/sheet` (writer) — 012 adds the reader and uses both.
- `frontend/src/lib/download.ts` (`downloadFromApi`, `saveBlob`) — used by 012.
- `/all-work?<API filter names>` opens All work filtered — used by 009's "Activities I own" card
  (`/all-work?owner=<person ID>`).

## Requirement traceability

| FR | Implemented in | Tested by |
|---|---|---|
| FR-001 | `AllWorkPage`, router, navigation | `AllWorkPage.test.tsx`, e2e 1 |
| FR-002 | read-only service (no write operation on business records) | `TestQuery_NeverWrites_FR002` |
| FR-003 | `rollup.go` summary, `SummaryCards` | `TestSummary_*_FR003`, `AllWorkPage.test.tsx` |
| FR-004 | `query.go`, `tree.go`, `Options`, `FilterBar` | `TestFilter_EachFilterOnItsLevel_FR004`, `TestOptions_*`, `FilterBar.test.tsx` |
| FR-005 – FR-007 | `tree.go` | tree tests, `AllWorkGrid.test.tsx`, e2e 2 |
| FR-008 | `tree.go` count, `FilterBar` chips | `TestCount_*`, `FilterBar.test.tsx` |
| FR-009 | `allWorkStore`, `viewState`, first-visit logic | `allWorkStore.test.ts`, `AllWorkPage.test.tsx` |
| FR-010 | resolved by Q-009: no attribute filters | `TestFilter_OnlyFR004Filters_FR010`, `FilterBar.test.tsx` |
| FR-011 – FR-017 | `tree.go`, `rollup.go`, `sorting.go`, `AllWorkGrid` | Go tests above, `AllWorkGrid.test.tsx`, e2e 1 |
| FR-018 – FR-021 | `columns.go`, preferences, `ColumnChooser` | columns and preferences tests, `ColumnChooser.test.tsx` |
| FR-022 – FR-024 | `periods.go`, `BudgetView` | periods tests, `BudgetView.test.tsx`, e2e 3 |
| FR-025 – FR-027 | `CalendarView`, calendar view of `Query` | `TestQuery_Calendar*`, `CalendarView.test.tsx` |
| FR-028 | `MonthCalendar` | `MonthCalendar.test.tsx` |
| FR-029 – FR-031 | `views.go`, Views components | views tests, Views component tests, e2e 4 |
| FR-032 – FR-034 | `export.go`, `platform/sheet`, `ExportMenu` | export and sheet tests, `ExportMenu.test.tsx`, e2e 5 |

## Complexity Tracking

| Choice | Why needed | Simpler alternative rejected because |
|---|---|---|
| `allwork.Source` interface with one implementation (`source_planning.go`) | All work reads six other packages; one adapter keeps that coupling in one file, lets rule tests run on hand-built years, and lets the stress test build 5,000 activities in memory | Calling six packages from every function spreads the coupling and makes SC-004 untestable without a huge store |
| `internal/platform/sheet` (own `.xlsx` writer) | FR-032 asks for a spreadsheet with an "About this export" sheet; 012 needs to read and write templates | A third-party library is a new dependency for ~200 lines of standard library (constitution VII) |
| Paging rows on screen | SC-004 with 5,000 activities; 012 builds selection on it | Rendering every row is slow; a virtualisation library is a new dependency |

## Consolidation (2026-10-19, binding)

The Plan-module consolidation (the cross-feature analyze step) settled the points below. They and the build guide,
[docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md), win where this plan, its data model or its tasks
say otherwise (D-200).

- **Contract (D-192).** The fragment is merged into `backend/api/openapi.yaml`; the merge task (T001) is done. The
  feature's operations are listed in `notYetServed` in `backend/internal/app/app_test.go`; the backend removes each one in
  the change that serves it. Agents do not change the contract; a problem with it is reported.
- **Decisions.** This plan's proposed decisions are numbered D-113–D-125 in `docs/sdd/decision-log.md` (labels kept in the
  table above); the consolidation decisions are D-189–D-207.

- **Spec (D-207).** FR-010's marker is resolved in spec.md: only the FR-004 filters.
- **Shared parts (D-197).** Reuse 004's `TreeTable`, `PillTabs`, `FilterChips`, `ProgressBar`, `LevelTag`, 002's `SegmentedFilter`,
  `SearchField`, 003's `PersonPicker`, `TeamPill`, and 004's `recordHref` (D-194). 010 owns `lib/download.ts` and
  `platform/sheet`.
- **Money.** Use 002's names (`(Amount).SplitEvenly`, `Rates.FromUSD`, `ParseAmount`); 007's month amounts come through an
  `internal/app` adapter over 007's spread functions.
- **Statuses.** Activity status filters and chips use 006's `ActivityDisplayStatus`.
