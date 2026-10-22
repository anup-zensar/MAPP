# Implementation Plan: Home Dashboard

**Branch**: `009-home-dashboard` | **Date**: 2026-10-19 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/009-home-dashboard/spec.md`

**Design artifacts**: [data-model.md](data-model.md) · [contracts/openapi.yaml](contracts/openapi.yaml) (fragment) ·
[tasks.md](tasks.md) · conventions of [001 plan](../001-app-shell-and-sign-in/plan.md)

## Summary

Home becomes the dashboard of design system v2 (Shell's design language): a greeting header with a **Currency** picker, a **My teams /
Everything** switch, four summary cards, **Plans at a glance**, **Needs your attention**, **Waiting for my
approval**, **My activities** and **Recent changes**. 001's **Getting started** card stays, but only for
new users (no team, or no plan in their teams).

Home stores nothing but one preference. Every figure is calculated **on the server** by a new read-only Go
package `internal/home`, which reads the roll-ups, completeness, statuses and approvals that features
002–008 already calculate, so each figure equals the one on the plan, activity and approvals pages
(FR-017). Each card has **its own endpoint**, so one failing card never hides the others (FR-022). Home
reads its inputs through a small `Sources` struct of functions, wired in `internal/app` from the other
packages' services: the service is tested with fakes, and the names of other packages appear in one
adapter file.

## Technical Context

**Language/Version**: Go 1.27 (API); TypeScript 5.9 strict + React 19.3 (web app) — as 001

**Primary Dependencies**: none new (standard library; `react-router`, `@tanstack/react-query`, `radix-ui`,
`lucide-react`, `openapi-fetch` from 001)

**Storage**: no table. One new field `homeScope` in 001's `preferences` table. Money uses 002's
`internal/platform/money` (exact decimals, domain model §4)

**Testing**: Go `testing` with fake `Sources` and the fixed clock; `apitest` contract tests on the seed;
Vitest + Testing Library; Playwright + axe

**Target Platform / Project Type**: as 001 (web application, React SPA + Go REST API)

**Performance Goals**: SC-004 — all cards within 2 s with the seed, within 3 s for a scope of 50 plans and
5,000 activities (each endpoint one pass over the inputs; cards load in parallel); SC-005 — a currency
change redraws every amount within 1 s

**Constraints**: 127.0.0.1 only; no personal data in server logs; WCAG 2.2 AA, keyboard-only use, every
bar and pill also shows its value or name as text (SC-007)

**Scale/Scope**: 1 page (`/`), 9 cards; 6 new API operations; 1 preference

## Constitution Check

*GATE: checked before design and again after it.*

| Principle | How this plan complies | Result |
|---|---|---|
| I. Spec first | Every element traces to FR-001–FR-022, a story, an edge case or an SC (traceability table). No `[NEEDS CLARIFICATION]` | Pass |
| II. Shell business, plain language | Shell levels and statuses; every text in *User-visible messages* | Pass |
| III. Simple, friendly experience | Design-system Home layout; status chips, completeness chips and reasons always visible; each card has its own empty, loading and failure state; every problem links to where it is fixed | Pass |
| IV. Simple, layered architecture | `internal/home` has handler → service and reads other services through `Sources` (no repository: it stores nothing, Complexity Tracking); every figure, percentage, reason and order is computed in Go; the web app only formats | Pass |
| V. Tested business rules | Service tests per rule with fakes (named with the FR); contract tests per operation; a seed-wide test that Home equals the plan and approvals pages (SC-002); component tests per card; Playwright for every story | Pass |
| VI. Safe by default | Read-only endpoints; the actor is passed to every call; the only write is the person's own display preference | Pass |
| VII. Keep it small | No dependency; one package; one new shared component (`TeamPill`) and a link variant of `SummaryCard`, both with reasons | Pass |

**Post-design re-check**: Pass, with the exceptions in Complexity Tracking.

## Decisions

No `[NEEDS CLARIFICATION]` markers. Decision-log defaults used: D-004 (no roles; the switch shows scope
without enforcing it), D-010 (USD base, totals added in USD and converted once), D-015 (over-allocation
shown, never blocks), D-016 (negative unassigned balance), D-020 (any user may approve, the card lists
the designated approver's queue), D-025 (team pills), D-026 (plan approval progress), Q-006 (switch,
not enforced).

| # | Decision | Reason |
|---|---|---|
| D-170 (009-A) | **Getting started** (001 FR-018) is shown only when the person belongs to no team or none of their teams has a plan, above the summary cards; everyone else sees the dashboard without it. The five steps move unchanged into `GettingStartedCard` | FR-020; people who already plan don't need the steps every day |
| D-171 (009-B) | Home's figures come from a read-only Go package `home` with one endpoint per card; it reads other features through a `Sources` struct of functions built in `internal/app` | FR-017 (same rules as other pages), FR-022 (cards fail alone), testable with fakes |
| Superseded by D-201 (009-C) | ~~Team colour derived from the team ID~~ — **withdrawn at consolidation**: teams store their colour (003, D-065); `TeamPill` (003) shows `TeamRef.colour`; no `lib/teamColor.ts`, no `--team-1…8` tokens (003 adds `--team-<colour>`) | D-025 and 003's design store the colour |
| Superseded by D-201 (009-D) | ~~Fall back to USD when the display currency is inactive~~ — **withdrawn at consolidation**: an inactive display currency stays in use with its current rate on every page, Home included (003, D-063); no "unavailable currency" notice | The owning feature (003) decides; SC-002 holds because every page uses the same rule |
| D-172 (009-E) | The **My teams / Everything** choice is a per-person preference `homeScope` (default `myTeams`) saved on the server, not logged | FR-003 "remembered per user"; display preference like D-038 |
| D-173 (009-F) | "Show all n" in **Needs your attention** and **My activities** expands the list inside the card (the API returns every item) | No other page lists them before feature 010 |
| D-174 (009-G) | **Activities I own** opens All work filtered to the person (`/all-work?owner=<id>`) once All work has left `COMING_SOON_SECTIONS`; until then it scrolls to the **My activities** card | FR-007 (2) fallback, decided at run time from `navigation.ts` |
| D-175 (009-H) | "Oldest waiting" and "time waiting" count from 008's **waiting since** (when the round reached its current level) | Matches what the approver experiences; 008 shows the same |
| D-176 (009-I) | Budget used is a whole percentage, half up (`plannedCost × 100 ÷ budget`), red when planned cost > budget (exact comparison, so 100.4 % shows "100 %" in red) | Same figure as the plan page (004 FR-012) |

## Project Structure

### Documentation (this feature)

```text
specs/009-home-dashboard/
├── spec.md · checklists/requirements.md
├── plan.md · data-model.md · contracts/openapi.yaml · tasks.md
```

### Source code (repository root)

```text
backend/
├── api/openapi.yaml                    # merge the fragment (6 paths, schemas; Preferences += homeScope)
├── internal/home/                      # NEW, read-only
│   ├── model.go                        # Scope, the API views (SummaryView, PlanRow, HomeAttentionItem, …)
│   ├── sources.go                      # Sources (functions) and the fact types Home reads
│   ├── service.go · service_test.go    # Summary, PlansAtAGlance, Waiting, MyActivities, RecentChanges
│   ├── attention.go · attention_test.go  # Needs your attention: reasons and order
│   └── handler.go · handler_test.go
├── internal/app/home_sources.go        # NEW: builds home.Sources from the services of 002–008 and 011
├── internal/app/app.go                 # wires home
├── internal/people/model.go · service.go   # + homeScope preference (+ tests)
└── seed/seed_test.go                   # + TestSeed_HomeStory_009

frontend/src/
├── features/home/
│   ├── HomePage.tsx (+ .module.css, .test.tsx)          # rewritten: layout, header, picker, switch
│   ├── GettingStartedCard.tsx (+ .test.tsx)             # 001's steps, moved
│   ├── CurrencyPicker.tsx (+ .test.tsx)
│   ├── SummaryCards.tsx (+ .module.css, .test.tsx)
│   ├── PlansAtAGlanceCard.tsx (+ .module.css, .test.tsx)
│   ├── AttentionCard.tsx (+ .module.css, .test.tsx)
│   ├── WaitingCard.tsx (+ .module.css, .test.tsx)
│   ├── MyActivitiesCard.tsx (+ .module.css, .test.tsx)
│   ├── RecentChangesCard.tsx (+ .module.css, .test.tsx)
│   ├── SectionError.tsx (+ .test.tsx)                   # "This section couldn't load" + Try again
│   ├── homeApi.ts                                       # queries
│   └── homeText.ts (+ .test.ts)                         # sentences, plurals, status words
├── components/SummaryCard/                              # + icon, detail, to (link variant)
├── components/TeamPill/ (TeamPill.tsx, .module.css, .test.tsx) + export in components/index.ts
├── lib/teamColor.ts (+ .test.ts) · lib/recordLinks.ts (+ .test.ts) · lib/format.ts (+ relative times)
└── styles/tokens.css                                    # + --team-1 … --team-8

frontend/e2e/home.spec.ts (new) · frontend/e2e/smoke.spec.ts (Home assertion updated)
```

## Backend design

### Inputs (`internal/home/sources.go`)

Home never reads another package's storage. `internal/app/home_sources.go` fills this struct from the
services of the features named in each comment (names expected — see Dependencies):

```go
type Sources struct {
    MyTeams      func(ctx context.Context, a actor.Actor) ([]Team, error)                    // 003: the actor's teams
    Plans        func(ctx context.Context, a actor.Actor) ([]Plan, error)                    // 004/005/006: every plan with its roll-ups
    Containers   func(ctx context.Context, a actor.Actor, ownerID string) ([]Container, error) // 005: campaigns and initiatives the person owns
    Activities   func(ctx context.Context, a actor.Actor, ownerID string) ([]Activity, error)  // 006/007/008: activities the person owns
    WaitingForMe func(ctx context.Context, a actor.Actor) ([]Waiting, error)                 // 008: the Waiting for me tab, oldest first
    Currency     func(ctx context.Context, a actor.Actor) (Currency, error)                  // 003 preference + 002 active flag
    FromUSD      func(ctx context.Context, usd money.Money, code string) (money.Money, error) // 002: current rate, 2 decimals half up
    Log          func(ctx context.Context, a actor.Actor, q history.LogQuery) (history.LogPage, error) // 011: QueryLog
}

type Team struct{ ID, Name string }
type Plan struct {
    ID, Name                         string
    Year                             int
    StrategicAOP                     bool
    Status                           string        // "draft" | "published"
    OwnerID                          string
    Teams                            []Team
    Budget                           *money.Money  // USD; nil when not set
    PlannedCost                      money.Money   // USD roll-up (domain model §4.2)
    RemainingBudget                  *money.Money  // USD; plan budget − Σ campaign budgets; nil without budget
    CampaignCount, ActivityCount     int
    ApprovedActivityCount            int           // Approved, In progress, Complete (domain model §2.2)
    Missing                          []MissingField // own needed-to-publish fields
    ItemsBelowNeedingAttention       int
    UpdatedAt                        time.Time
}
type Container struct { // campaign or initiative
    Type                       string        // "campaign" | "initiative"
    ID, Name, OwnerID          string
    Plan                       history.RecordRef
    Path                       []history.RecordRef // ancestors, plan first
    Budget                     *money.Money
    PlannedCost                money.Money
    RemainingBudget            *money.Money  // campaigns only: budget − Σ initiative budgets; nil otherwise
    Missing                    []MissingField
    ItemsBelowNeedingAttention int
    UpdatedAt                  time.Time
}
type Activity struct {
    ID, Name, OwnerID            string
    Plan, Campaign, Initiative   history.RecordRef
    Status                       string        // displayed (domain model §2.3): created, submitted, declined, approved, inProgress, complete
    StartDate, EndDate           string        // "YYYY-MM-DD"; "" when not set
    Ongoing                      bool
    PlannedCost                  *money.Money  // USD; nil when not set
    Unassigned                   money.Money   // USD; negative when line items exceed (domain model §4.3)
    Missing                      []MissingField
    LineItemsNeedingAttention    int
    Decline                      *Decline      // set when Status is declined (008: the decision that declined it)
    UpdatedAt                    time.Time
}
type Decline struct { By actor.Actor; At time.Time; Comment string }
type Waiting struct {
    ActivityID, ActivityName string
    Plan                     history.RecordRef
    Owner                    *actor.Actor
    PlannedCost              *money.Money // USD
    Level, Levels            int
    WaitingSince             time.Time
}
type Currency struct { Code string; Active bool } // the person's display currency
type MissingField struct{ ID, Label string }
```

Deleted records are never returned by these functions (they are no longer stored, FR-019).

### Service operations (`internal/home/service.go`, `attention.go`)

`NewService(src Sources, now clock.Clock) *Service`. Every operation takes `(ctx, a actor.Actor, …)`,
reads only, and converts money through one helper `display(ctx, usd) (Money, error)` that uses the
**display currency** (the person's display currency, active or not — D-201, D-063). Totals are summed in USD and
converted once (FR-017). `scope` is `myTeams` or `everything`; anything else → `400 validation_failed`,
field `scope`, "Choose My teams or Everything.".

**Plans in scope**: `everything` → every plan; `myTeams` → plans with at least one team in `MyTeams`,
each **once** (FR-003, FR-004).

| Operation | Rules |
|---|---|
| `Summary(ctx, a, scope) (SummaryView, error)` | `plans`: count, published, draft of the plans in scope (FR-005 1). `activitiesOwned`: count, declined, submitted of `Activities(a.ID)` — all plans (FR-005 2, edge case). `waitingForMe`: count and the earliest `WaitingSince` of `WaitingForMe` (FR-005 3). `plannedCost`: Σ planned cost and Σ budget (only plans with a budget) of the plans in scope; `budgetUsedPercent` = round half up (Σ planned × 100 ÷ Σ budget) when Σ budget > 0, else null; `overBudget` = Σ planned > Σ budget; `plansWithoutBudget`; `planCount` (FR-005 4, FR-006). `gettingStarted`: `notInTeam` when `MyTeams` is empty; `noTeamPlans` when none of the plans has one of the teams; otherwise null (FR-020). `displayCurrency` = effective code; `unavailableCurrency` = the preference's code when inactive, else null |
| `PlansAtAGlance(ctx, a, scope) (PlanPage, error)` | Rows for the plans in scope ordered by year (newest first), then name ignoring letter case, then ID; at most 10; `total` = all in scope (FR-010). Each row: teams with `mine`, the person's teams first, then by name (FR-008); campaign and activity counts; approved and total activities (FR-009); budget, planned cost; budget used as above (null without budget); `overBudget` |
| `Attention(ctx, a) (HomeAttentionPage, error)` | Records the person owns in **all** plans: `Plans` with `OwnerID = a.ID`, `Containers(a.ID)`, `Activities(a.ID)`. Reasons (FR-011): (a) `declined` — activity status declined, with `Decline`; (b) `negativeBalance` — activity `Unassigned < 0`, amount = −Unassigned; (c) `overAllocated` — plan or campaign with `RemainingBudget < 0`, amount = −Remaining; (d) `overBudget` — plan, campaign or initiative with a budget and planned cost > budget, amount = planned − budget; (e) `missingFields` — `len(Missing) > 0` or items below > 0, with both counts. A record with no reason is left out; a record with several appears once with all, in the order a, b, c, d, e (FR-012). Items ordered by their first reason's group (declined → balance → budget (c, d) → missing fields), then `UpdatedAt` newest first, then ID; all items returned, `total` = count |
| `Waiting(ctx, a) (WaitingPage, error)` | `WaitingForMe` as given (oldest first, the same list as 008's tab); first 5; `total` (FR-013) |
| `MyActivities(ctx, a) (MyActivitiesView, error)` | `Activities(a.ID)` grouped by displayed status in the order declined, created, submitted, approved, inProgress, complete; empty groups left out; within a group by start date (no date last), then name, then ID; every row returned with the group's `count`; `total` (FR-014) |
| `RecentChanges(ctx, a, scope) (RecentPage, error)` | `Log` with `RecordTypes` = plan, campaign, initiative, activity, lineItem, `Limit` 10, and `PlanIDs` = the plans in scope for `myTeams` (no call, empty page, when there are none) or nil for `everything` (FR-015). Items are 011's entries with `recordNow`, so a deleted record shows "(in recycle bin)" |

`Summary`, `PlansAtAGlance` and `RecentChanges` share one helper `plansInScope(ctx, a, scope)`.

### Preference (`internal/people`)

`homeScope` in `Preferences` (default `myTeams`), `PreferencesView`, `PreferencesUpdate.HomeScope *string`;
any other value → `validation_failed`, field `homeScope`, "Choose My teams or Everything."; saved only
when it changes; not logged (D-038).

### History kinds and activity-log entries

Home registers no kind, deletes nothing and writes no log entry: it only reads. Its preference change is
not logged (D-038).

### API operations

Contract: [contracts/openapi.yaml](contracts/openapi.yaml). All need a session; all amounts are in the
effective display currency (`Money`).

| Method and path | Request | Response | Errors |
|---|---|---|---|
| `GET /api/home/summary` | query `scope` (required) | `200 HomeSummary` | 400 `validation_failed` (`scope`), 401 |
| `GET /api/home/plans` | query `scope` (required) | `200 HomePlanPage {items ≤ 10, total}` | 400, 401 |
| `GET /api/home/attention` | — | `200 HomeAttentionPage {items, total}` | 401 |
| `GET /api/home/waiting` | — | `200 WaitingPage {items ≤ 5, total}` | 401 |
| `GET /api/home/my-activities` | — | `200 MyActivitiesPage {groups, total}` | 401 |
| `GET /api/home/recent-changes` | query `scope` (required) | `200 RecentChangesPage {items ≤ 10, total}` | 400, 401 |
| `PATCH /api/me/preferences` (001, extended) | `{homeScope}` and/or 003's `{displayCurrency}` | `200 Preferences` | 400 |

### Validation messages

| Field | Message |
|---|---|
| `scope` (query) | Choose My teams or Everything. |
| `homeScope` (preferences) | Choose My teams or Everything. |

### Demo data

Home adds no records. It relies on the seed of 002–008 and 011 holding these facts, which
`TestSeed_HomeStory_009` checks through the Home endpoints; a fact that is missing is added to
`backend/seed/seed.go` by task T018:

| Fact | Used by |
|---|---|
| Priya Raman, Daniel Okoro and Sara Novak in Lubricants Europe; Mei Lin and Carlos Mendes in Mobility China; Alex Morgan and Carlos Mendes in Global Marketing Ops; Sara Novak also in Mobility China | US1, US2 |
| FY2026 Lubricants Europe Published (owner Priya, budget 1,200,000.00 USD); FY2026 Mobility China Draft (budget 800,000.00 USD); FY2027 Global Brand Draft, AOP, Global Marketing Ops, no plan budget | US1 scenario 2, US2 |
| Priya owns a Declined activity declined by Daniel Okoro with a comment | US3 scenario 1 |
| Priya owns an activity whose line items exceed its planned cost by exactly 1,200.00 USD | US3 scenario 2 |
| Alex Morgan owns the FY2027 Global Brand campaign whose initiative budgets exceed its budget | US3 scenario 3 |
| At least one Submitted activity waits for Daniel Okoro at level 1 of 1 | US3 scenario 5 |

## Frontend design

### Route and navigation

`/` keeps rendering `features/home/HomePage.tsx` (001 kept the route free for this). No navigation change:
Home is already the first sidebar item.

### Layout (`HomePage`)

1. `PageHeader`: eyebrow "Home", title "Welcome back,", highlight "<first name>.", description, actions =
   `CurrencyPicker` (FR-001).
2. `GettingStartedCard` when `summary.gettingStarted` is not null (FR-020), with the no-team sentence for
   `notInTeam` (its "Teams page" is a link to `/settings/teams`).
3. Scope row: the pill switch **My teams / Everything** (004's `PillFilter`, `aria-label` "Scope") and the
   hint (FR-003). The value starts from `preferences.homeScope`; a change shows at once and saves
   `PATCH /api/me/preferences {homeScope}` (001's preference pattern: failed save → "Your choice couldn't
   be saved." with **Try again**).
4. `SummaryCards` — four cards in a row (≥ 1024 px), 2 × 2 (≥ 640 px), one column below.
5. `PlansAtAGlanceCard` (full width).
6. Two columns from 1024 px: `AttentionCard` | `WaitingCard`; then `MyActivitiesCard` (`id="my-activities"`)
   | `RecentChangesCard`. One column below, in that order.

Each card uses its own query; while loading it shows `Skeleton` lines inside its `Card`; on failure
`SectionError` ("This section couldn't load" + **Try again** → `refetch()`), the other cards unaffected
(FR-022). Queries use `staleTime: 0` and `refetchOnMount: 'always'`, so Home shows current figures each
time it is opened or returned to (FR-019).

### Currency picker (`CurrencyPicker`, FR-002)

A labelled native `<select>` "Currency" listing the currencies My profile offers (003's `GET
/api/me/display-currencies`: the active ones, plus the person's own when it was set inactive, "EUR — Euro";
D-217), value = `summary.displayCurrency`. Choosing saves `PATCH /api/me/preferences {displayCurrency}`
(003's field), then invalidates `['session']`, the display currencies and `['home']` so every amount is
redrawn in the new currency (SC-005). An inactive display currency stays in use (D-201), so there is no
fall-back notice. A failed save shows "Your choice couldn't be saved." with **Try again**.

### Summary cards (`SummaryCards`, FR-005–FR-007)

`SummaryCard` gains four optional props (shared change, see Complexity Tracking): `icon` (a 36 px
tinted tile, version 1 style), `detail` (a node under the value) and `to` (the whole card is a link whose
accessible name is "<label>: <value> — <linkLabel>", with `linkLabel` the fourth prop).

| Card | Icon / tint | Value | Detail | Link |
|---|---|---|---|---|
| My teams' plans / All plans | `ClipboardList` on `--blue` | `plans.count` | "<n> published · <n> draft" | `/plans?scope=myTeams` / `/plans?scope=all` |
| Activities I own | `ListChecks` on `--level-campaign` | `activitiesOwned.count` | "<n> declined · <n> submitted" | `/all-work?owner=<id>` or `#my-activities` (D-174) |
| Waiting for my approval | `Hourglass` on `--status-review` | `waitingForMe.count`, tone `review` when > 0 | "oldest waiting <formatWaitingTime>" or "Nothing waiting" | `/approvals?tab=waiting` |
| Planned cost vs plan budget | `Wallet` on `--status-positive` | `formatMoneyCompact(plannedCost)` with `fullValue` | "of <compact budget> plan budget", 004's `ProgressBar` with "<n> % used" (problem tone when `overBudget`), "Includes <n> plans with no budget"; no budget: "—" and "No plan budgets set"; no plans: value "—", "No plans in scope" | as card 1 |

### Plans at a glance (`PlansAtAGlanceCard`, FR-008–FR-010, FR-021)

`Card` title "Plans at a glance" with `InfoTip`, subtitle by scope. A table (scrolls sideways inside the
card on narrow screens): **Plan** (link to `recordHref('plan', id)`, then "<ID> · <year>" and an "AOP"
chip when `strategicAop`), **Status** (`StatusChip`: Draft neutral, Published positive), **Teams**
(`TeamPill` each), **Campaigns**, **Activities**, **Approval** (`ProgressBar` + "<n> of <m> approved", or
"No activities"), **Budget used** (`ProgressBar` + "<n> %", problem tone when `overBudget`; "—" with
"No budget" when null), **Planned cost** (`formatMoney`, right-aligned). Under the table, when `total >
10`, **View all <n> plans** → the Plans list with the scope. Empty: `myTeams` → "None of your teams has a
plan yet." with **Switch to Everything** (sets the scope); `everything` → "No plans yet — create the first
one on the Plans page." with **Go to Plans**.

### Needs your attention (`AttentionCard`, FR-011, FR-012)

Each item: 004's `LevelTag`, name (link to the record), ID, "in <plan name>" (not for a plan), then one
line per reason with its link (`recordHref` + section hash):

| Reason | Text | Link (label → target) |
|---|---|---|
| `declined` | Declined by <name> on <DD MMM YYYY>: '<comment>' | Open approval → `#approval` |
| `negativeBalance` | Line items exceed the planned cost by <amount> | Open line items → `#line-items` |
| `overAllocated` | Over-allocated by <amount> | Open budget → `#budget` |
| `overBudget` | Planned cost is <amount> over budget | Open budget → `#budget` |
| `missingFields` | <n> fields needed (1: 1 field needed) · <n> items below need attention (1: 1 item below needs attention) | Show fields → `#completeness` |

Each link's accessible name adds the record ("Open approval for activity 'Launch film'"). 8 items, then
**Show all <n>** / **Show fewer**. Empty: "Nothing needs your attention right now."

### Waiting for my approval (`WaitingCard`, FR-013)

Rows: name (link to the activity), ID, plan name, owner, planned cost, "Level <n> of <m>", "waiting
<formatWaitingTime>", **Review** (`LinkButton`, `aria-label` "Review <name>") →
`/approvals?tab=waiting&activity=<id>` (008 opens that preview). **View all <n>** → `/approvals?tab=waiting`
when `total > 5`. Empty: "Nothing is waiting for your approval right now."

### My activities (`MyActivitiesCard`, FR-014)

Each group: a toggle button (`aria-expanded`) with the status chip and count; Complete starts collapsed.
Rows: name (link), ID, "<plan> › <campaign>", dates ("1 Mar 2026 – 31 May 2026", "Ongoing from 1 Mar
2026", "No dates yet"), planned cost, `CompletenessChip` (`missing`; "<n> line items need attention" when
`itemsBelow > 0`). Five rows, then **Show all <n>** / **Show fewer**. Status words and tones: Created
neutral, Submitted review, Declined problem, Approved positive, In progress active, Complete done
(design system §4). Empty: "You don't own any activities yet." with **Go to Plans**.

### Recent changes (`RecentChangesCard`, FR-015, FR-016)

Rows: `Avatar`, sentence (`homeText.recentSentence`), relative time (`formatRelativeTime`) whose `Tooltip`
shows `formatDateTime`. Sentence: "<person> changed <fields> of <type label> '<name>' (<id>)" for a
`changed` entry — fields = the change labels with the first letter lower-cased when the second letter is
lower case ("Planned cost" → "planned cost", "GL code" stays), joined "a, b and c", more than three →
"a, b, c and <n> more fields"; any other action: "<person> <verb> <type label> '<name>' (<id>)". The name
links to `recordHref` when `recordNow.state` is `exists`; `inRecycleBin` → plain name + " (in recycle bin)";
`deletedPermanently` → plain name + " (permanently deleted)". Card action **View activity log** →
`/activity-log` (011, or its Coming soon page). Empty: "No changes yet in these plans."

### Shared parts this feature adds

| Part | Why shared |
|---|---|
| `SummaryCard` props `icon`, `detail`, `to` | Home's cards need a sub-line, a bar and a link (FR-005, FR-007); the activity log (011) and later lists reuse the link variant |
| `TeamPill` (`name`, `teamId`) — dot in `teamColor(teamId)`, 12 % tint background, `--ink` text | Team pills appear on Home, the Plans list (004) and All work (010) (D-025); the name is always the text (SC-007) |
| `lib/teamColor.ts` `teamColor(id): string` (a CSS variable `var(--team-n)`) and tokens `--team-1` … `--team-8` = `#2a78d6`, `#8c42bc`, `#1baf7a`, `#e0700b`, `#d6456e`, `#0f9fb5`, `#6b7f12`, `#7a5af8` (same in dark mode) | D-201 |
| `lib/recordLinks.ts` `recordHref(type, id): string \| null` — `plan` → `/plans/<id>`, `campaign` → `/campaigns/<id>`, `initiative` → `/initiatives/<id>`, `activity` → `/activities/<id>`, `lineItem` → `/line-items/<id>` (the routes of 004–007, to align), anything else → null | Home and 011 link to records of every level |
| `lib/format.ts` `formatRelativeTime(ts, now = new Date())`: < 1 min "just now"; < 60 min "<n> minute(s) ago"; < 24 h "<n> hour(s) ago"; the local day before "yesterday"; < 7 days "<n> days ago"; else `formatDate(localDate(ts))`. `formatWaitingTime(since, now)`: < 1 h "less than an hour"; < 24 h "<n> hour(s)"; else "<n> day(s)" (whole days, rounded down) | 008 (notifications, approvals rows), 009 and 011; added here only when 008 has not added them |

### Query keys

`['home', 'summary', scope]`, `['home', 'plans', scope]`, `['home', 'attention']`, `['home', 'waiting']`,
`['home', 'my-activities']`, `['home', 'recent-changes', scope]`, `['currencies']`. A currency change
invalidates `['session']` and `['home']`.

## User-visible messages

| Where | Text |
|---|---|
| Header | Home · Welcome back, · <first name>. · Your plans, your activities and what needs you today. |
| Currency picker | Currency · <CODE> — <name> · <CODE> is no longer available — showing USD · Your choice couldn't be saved. · **Try again** |
| Scope switch | Scope · My teams · Everything · Your own activities, what needs your attention and your approvals always cover all plans. |
| Card 1 | My teams' plans · All plans · tip: Plans that have at least one of your teams. / Every plan in NorthStar. · <n> published · <n> draft |
| Card 2 | Activities I own · tip: Activities you own, in every plan. · <n> declined · <n> submitted |
| Card 3 | Waiting for my approval · tip: Submitted activities waiting at a level where you are an approver. · oldest waiting <time> · Nothing waiting |
| Card 4 | Planned cost vs plan budget · tip: Planned cost of the plans in scope compared with their plan budgets. · of <amount> plan budget · <n> % used · Includes <n> plans with no budget (1: Includes 1 plan with no budget) · No plan budgets set · No plans in scope |
| Card link names | <label>: <value> — open the Plans list · — open your activities · — open Approvals |
| Plans at a glance | Plans at a glance · tip: Every plan your teams work on, the teams on it, and live status rolled up from its activities. · Live roll-up across your teams · Live roll-up across all plans · Plan · Status · Teams · Campaigns · Activities · Approval · Budget used · Planned cost · AOP · <n> of <m> approved · No activities · — · No budget · View all <n> plans · None of your teams has a plan yet. · **Switch to Everything** · No plans yet — create the first one on the Plans page. · **Go to Plans** |
| Needs your attention | Needs your attention · Things you own that have a problem · in <plan> · (reason texts and link labels in the table above) · Open approval for <type> '<name>' · Show all <n> · Show fewer · Nothing needs your attention right now. |
| Waiting for my approval | Waiting for my approval · Level <n> of <m> · waiting <time> · **Review** · Review <name> · View all <n> · Nothing is waiting for your approval right now. |
| My activities | My activities · Activities you own, by status · Created · Submitted · Declined · Approved · In progress · Complete · <date> – <date> · Ongoing from <date> · No dates yet · <n> line items need attention (1: 1 line item needs attention) · Show all <n> · Show fewer · You don't own any activities yet. · **Go to Plans** |
| Recent changes | Recent changes · Latest changes in your teams' plans · Latest changes in all plans · <person> changed <fields> of <type> '<name>' (<id>) · <person> <verb> <type> '<name>' (<id>) · and <n> more fields · (in recycle bin) · (permanently deleted) · just now · <n> minutes ago · … · **View activity log** · No changes yet in these plans. |
| Getting started | Getting started · The Plan module in five steps · (001's five steps unchanged) · You're not in a team yet. Plans belong to teams — add yourself to one on the Teams page. |
| Card failure | This section couldn't load · **Try again** |
| API validation | Choose My teams or Everything. |

## Dependencies

### Uses from feature 001

`actor`, `clock`, `errs`, `web`, `apitest`, `history.RecordRef`; people preferences; web: `PageHeader`,
`SummaryCard` (extended), `Card`, `StatusChip`, `CompletenessChip`, `InfoTip`, `Tooltip`, `Avatar`,
`Button`, `LinkButton`, `EmptyState`, `Skeleton`, `useToast`, `useCurrentUser`/`useSession`, `call`,
`formatDate`, `formatDateTime`, `formatMoney`, `formatMoneyCompact`, `localDate`, `usePageTitle`,
`COMING_SOON_SECTIONS`; 001's Home steps (moved).

### Expects from other features (by name; the consolidation step aligns names)

| From | What | Used for |
|---|---|---|
| 002 | `internal/platform/money`: `Money` (exact decimal + currency), add, compare, negate; percent rounding half up; the schema `Money` (`{amount: "1234.50", currency: "USD"}`); conversion from USD at the current rate (expected `currencies.Service.FromUSD(ctx, usd, code)`) and whether a currency is active; `GET /api/currencies` (`items[]` with `code`, `name`, `active`) | `Sources.FromUSD`, `Sources.Currency`, picker |
| 003 | The actor's teams (expected `people.Service.TeamsOf(ctx, a, personID)` → id, name); preference `displayCurrency` in `Preferences`/`PreferencesUpdate`; route `/settings/teams`; `POST /api/people` (e2e US5) | `Sources.MyTeams`, picker, Getting started link |
| 004 (+ 005, 006 roll-ups) | Every plan with the figures of the Plans list and plan page (expected `plans.Service.Summaries(ctx, a)`): id, name, year, strategic AOP, status, owner, teams, budget, planned cost, remaining budget, campaign and activity counts, approved count, completeness (missing fields, items below), updated at; schemas `PlanStatus` (`draft`, `published`) and `MissingField` (`{id, label}`); the Plans list URL `/plans?scope=myTeams` or `/plans?scope=all`; route `/plans/<id>` with sections `#budget` and `#completeness`; shared components `PillFilter`, `ProgressBar`, `LevelTag` | `Sources.Plans`, cards, rows, links |
| 005 | Campaigns and initiatives owned by a person with budget, planned cost, remaining budget (campaigns), completeness, path, updated at (expected `campaigns.Service.OwnedBy(ctx, a, personID)`); routes `/campaigns/<id>`, `/initiatives/<id>` with `#budget`, `#completeness` | `Sources.Containers` |
| 006 + 007 | Activities owned by a person with the displayed status, dates, ongoing, planned cost, unassigned planned cost (007), completeness incl. line items needing attention, parents, updated at (expected `activities.Service.OwnedBy(ctx, a, personID)`); schema `ActivityStatus` (`created`, `submitted`, `declined`, `approved`, `inProgress`, `complete`); route `/activities/<id>` with `#approval`, `#line-items`, `#completeness`; an API to change a planned cost (e2e US4, expected `PATCH /api/activities/{id}`) | `Sources.Activities`, rows, links |
| 008 | The Waiting for me list (expected `approvals.Service.WaitingFor(ctx, a, personID)`: activity, plan, owner, planned cost, level n of m, waiting since, oldest first) and the decision that declined an activity (expected `approvals.Service.LastDecline(ctx, a, activityID)`: who, when, comment); URLs `/approvals?tab=waiting` and `&activity=<id>` (opens the preview); `formatRelativeTime` / `formatWaitingTime` if 008 adds them | `Sources.WaitingForMe`, `Activity.Decline`, card 3, Review |
| 010 | `/all-work?owner=<personId>` | Card 2 link once All work is built |
| 011 | `history.Service.QueryLog`, `LogQuery{RecordTypes, PlanIDs, Limit}`, `LogPage`, `EntryView` (`ActivityLogEntry` with `recordNow`) | `Sources.Log`, Recent changes |

When 009 is built before 011, 009's backend agent first does 011's tasks T006, T007 and T009 (history
types, refactor and `QueryLog`), which 011 then keeps.

### Provides to later features

`SummaryCard` link variant; `TeamPill`, `teamColor`; `recordHref`; `formatRelativeTime`,
`formatWaitingTime` (when added here); preference `homeScope`; `home` endpoints (only Home uses them).

## Tests

### Go service tests (`internal/home`, fake `Sources`, fixed clock 2026-03-12 09:00 UTC)

| Rule | Test |
|---|---|
| My teams covers plans with one of the person's teams; a plan in two teams counts once; Everything covers all | `service_test.go` `TestPlansInScope_MyTeamsAndEverything_FR003_FR004` |
| Card counts (published/draft; owned declined/submitted in all plans; waiting count and oldest) | `TestSummary_Counts_FR005`, `TestSummary_OwnedActivitiesCoverAllPlans_EdgeCase` |
| Budget used = total planned ÷ total budget; plans without budget; no budget at all; over 100 % | `TestSummary_BudgetUsed_FR006`, `TestSummary_PlansWithoutBudget_FR006`, `TestSummary_NoBudgetAtAll_FR006` |
| Totals added in USD and converted once; inactive display currency falls back to USD | `TestSummary_TotalsConvertedOnce_FR017`, `TestSummary_InactiveCurrencyFallsBackToUSD_EdgeCase` |
| Getting started for no team / no team plans, hidden otherwise | `TestSummary_GettingStarted_FR020` |
| Invalid scope refused | `TestScope_Invalid` |
| Row figures, own teams first, approval progress rule, no budget / no activities | `TestPlansAtAGlance_RowFigures_FR008_FR009`, `TestPlansAtAGlance_OwnTeamsFirst_FR008` |
| Order by year then name, 10 rows, total | `TestPlansAtAGlance_OrderAndLimit_FR010` |
| Each reason (a)–(e), only owned records, all plans | `attention_test.go` `TestAttention_EachReason_FR011`, `TestAttention_OnlyOwnedRecords_FR011` |
| A record with several reasons appears once; group order; newest first within a group | `TestAttention_OneItemAllReasons_FR012`, `TestAttention_Order_FR012` |
| Line-item problems appear through their activity | `TestAttention_LineItemsThroughActivity_Assumption` |
| Waiting = 008's list, oldest first, 5 + total | `TestWaiting_SameListOldestFirst_FR013` |
| My activities groups, order, empty groups hidden, displayed status used as given (ongoing never complete) | `TestMyActivities_Groups_FR014`, `TestMyActivities_UsesDisplayedStatus_EdgeCase` |
| Recent changes: planning types, plans in scope, 10, empty scope without a call | `TestRecentChanges_Scope_FR015` |
| 50 plans and 5,000 activities answer quickly | `TestHome_LargeScope_SC004` (each operation < 300 ms with fakes) |
| `homeScope` preference | `people/service_test.go` `TestUpdatePreferences_HomeScope_FR003` |

### Handler / contract tests (`internal/home/handler_test.go`, `apitest` with the seed)

`TestHomeSummary_Contract`, `TestHomePlans_Contract`, `TestHomeAttention_Contract`,
`TestHomeWaiting_Contract`, `TestHomeMyActivities_Contract`, `TestHomeRecentChanges_Contract`,
`TestHome_InvalidScope_400`, and **`TestHome_EqualsPlanAndApprovalPages_SC002_FR017`**: for every seeded
person and both scopes, each plan row equals 004's plan figures, card 4 equals the sum of the Plans list
and card 3 equals 008's Waiting for me total; **`TestHome_DeletedRecordsNeverCount_FR019`**: delete an
activity through 006's API and its figures leave Home at once. 001's route tests cover the new routes.
`seed/seed_test.go` `TestSeed_HomeStory_009` checks the seed facts.

### Web component tests (Vitest)

| File | Covers |
|---|---|
| `HomePage.test.tsx` | FR-001 header; FR-003 switch saves `homeScope` and refetches with the new scope, hint; FR-019 refetch on return; FR-020 Getting started shown / hidden / no-team sentence; FR-022 one card fails, the others show |
| `CurrencyPicker.test.tsx` | FR-002 active currencies, saving, invalidation; fallback notice |
| `SummaryCards.test.tsx` | FR-005 values, details, amber and red; FR-006 no-budget variants; FR-007 links per scope and the All work fallback |
| `PlansAtAGlanceCard.test.tsx` | FR-008 row texts and pills; FR-009 "n of m approved" / "No activities"; FR-010 View all; FR-021 Switch to Everything |
| `AttentionCard.test.tsx` | FR-011 reason texts and links; FR-012 one item with all reasons, 8 + Show all; empty |
| `WaitingCard.test.tsx` | FR-013 rows, Review link, View all, empty |
| `MyActivitiesCard.test.tsx` | FR-014 groups, collapsed Complete, 5 + Show all, dates, completeness |
| `RecentChangesCard.test.tsx` | FR-015 sentences, links, "(in recycle bin)", relative time + tooltip; FR-016 View activity log |
| `SectionError.test.tsx`, `GettingStartedCard.test.tsx`, `homeText.test.ts` | texts, Try again, plurals, field-label lower-casing |
| `SummaryCard.test.tsx`, `TeamPill.test.tsx`, `teamColor.test.ts`, `recordLinks.test.ts`, `format.test.ts` | new props (link name, detail), pill text and colour, stable colour, routes, relative times |

### End to end (Playwright, `frontend/e2e/home.spec.ts`)

| Journey | Checks |
|---|---|
| US1 (Priya) | header and picker; four cards with seed figures ("1 published · 0 draft", "of $1.2M plan budget"); each card link; EUR → every amount labelled EUR, kept after reload; back to USD |
| US2 (Sara) | both FY2026 plans once each with pills and bars; **Everything** → FY2027 Global Brand appears; kept after reload; back to My teams |
| US3 (Priya, Daniel) | declined item first with "Declined by Daniel Okoro on …" and its link; "Line items exceed the planned cost by $1,200.00"; Daniel: **Review** opens Approvals with the preview (1 step, SC-003) |
| US4 (Priya) | change an activity's planned cost through the API, return Home: Recent changes starts with "Priya Raman changed planned cost of activity '…' (A-…)" and "just now" |
| US5 | a new person without a team (created through 003's API) sees Getting started with the no-team sentence and every card's empty state (SC-006) |
| SC-007 | axe (WCAG 2.2 AA) on Home in light and dark mode; keyboard: switch, card links, Show all, Review |

`frontend/e2e/smoke.spec.ts`: the Home assertion for Priya changes from "Getting started" to the "Plans at
a glance" card (001 plan: kept green by 009).

## Requirement traceability

| FR | Implemented in | Tested by |
|---|---|---|
| FR-001 | `HomePage` header | `HomePage.test.tsx`, e2e US1 |
| FR-002 | `CurrencyPicker`, effective currency in `home.Service` | `CurrencyPicker.test.tsx`, `TestSummary_InactiveCurrency…`, e2e US1 |
| FR-003, FR-004 | scope switch, `homeScope`, `plansInScope` | `TestPlansInScope_*`, people test, `HomePage.test.tsx`, e2e US2 |
| FR-005 – FR-007 | `Summary`, `SummaryCards`, `SummaryCard` link variant | `TestSummary_*`, `SummaryCards.test.tsx`, e2e US1 |
| FR-008 – FR-010 | `PlansAtAGlance`, `PlansAtAGlanceCard`, `TeamPill` | `TestPlansAtAGlance_*`, card test, e2e US2 |
| FR-011, FR-012 | `attention.go`, `AttentionCard` | `TestAttention_*`, card test, e2e US3 |
| FR-013 | `Waiting`, `WaitingCard` | `TestWaiting_*`, card test, e2e US3 |
| FR-014 | `MyActivities`, `MyActivitiesCard` | `TestMyActivities_*`, card test |
| FR-015, FR-016 | `RecentChanges` (011 `QueryLog`), `RecentChangesCard` | `TestRecentChanges_*`, card test, e2e US4 |
| FR-017 | `Sources` = the other features' own calculations; USD totals converted once | `TestHome_EqualsPlanAndApprovalPages_SC002_FR017`, `TestSummary_TotalsConvertedOnce_FR017` |
| FR-018 | `formatMoney`, `formatMoneyCompact` with `fullValue` | card tests |
| FR-019 | refetch on mount; deleted records absent from `Sources` | `HomePage.test.tsx`, `TestHome_DeletedRecordsNeverCount_FR019` |
| FR-020 | `gettingStarted`, `GettingStartedCard` | `TestSummary_GettingStarted_FR020`, page test, e2e US5 |
| FR-021 | every card's empty state | card tests, e2e US5 |
| FR-022 | one query per card, `SectionError` | `HomePage.test.tsx` |

## Complexity Tracking

| Choice | Why needed | Simpler alternative rejected because |
|---|---|---|
| A Go package without repository (`home` reads through `Sources`) | Home stores nothing; it must reuse the exact calculations of 004–008 (FR-017) | A repository would read other features' tables and re-implement their roll-ups |
| `Sources` as a struct of functions plus an adapter file in `internal/app` | Service tests need deterministic fakes for 50 plans / 5,000 activities; other packages' names stay in one file | Importing five services into `home` makes tests need the whole seeded system |
| Six endpoints instead of one | One failing card must not hide the others (FR-022); scope-dependent and scope-free cards refresh separately | One `GET /api/home` fails as a whole |
| `SummaryCard` extended, `TeamPill` added to `@/components` | Used by Home, 011 and the lists of 004 and 010; the 001 README asks for a reason in the plan | Feature-local copies would drift |

## Consolidation (2026-10-19, binding)

The Plan-module consolidation (the cross-feature analyze step) settled the points below. They and the build guide,
[docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md), win where this plan, its data model or its tasks
say otherwise (D-200).

- **Contract (D-192).** The fragment is merged into `backend/api/openapi.yaml`; the merge task (T001) is done. The
  feature's operations are listed in `notYetServed` in `backend/internal/app/app_test.go`; the backend removes each one in
  the change that serves it. Agents do not change the contract; a problem with it is reported.
- **Decisions.** This plan's proposed decisions are numbered D-170–D-176 in `docs/sdd/decision-log.md` (labels kept in the
  table above); the consolidation decisions are D-189–D-207.

- **Withdrawn (D-201).** 009-C (colour from the team ID) and 009-D (USD fall-back) are withdrawn: `TeamPill` is 003's and reads
  `TeamRef.colour` (no `lib/teamColor.ts`, no `--team-1…8` tokens; T019 only checks 003's tokens; T040 documents 003's palette),
  and an inactive display currency stays in use. `HomeSummary.unavailableCurrency` was always `null` and has since been removed from the contract (D-217);
  `TestSummary_InactiveCurrencyFallsBackToUSD` becomes `TestSummary_InactiveDisplayCurrencyStaysInUse`.
- **Contract (D-193).** 009's attention schemas are `HomeAttentionItem`, `HomeAttentionReason`, `HomeAttentionPage` (004 owns
  `AttentionItem`). `Preferences.homeScope` is optional in the merged contract (D-191); 009 always sends it, the web app defaults to
  `myTeams`.
- **Shared parts (D-197).** 009 owns the `SummaryCard` link variant (`icon`, `to`, `linkLabel`; 004 adds `detail`); it reuses 004's
  `recordHref` (D-194), 008's `formatRelativeTime`/`formatWaitingTime`, 003's `TeamPill`.
- **Statuses.** Activity rows show 006's `ActivityDisplayStatus` (inProgress, complete come from it).
- **Sources (D-171).** `home.Sources` adapters in `internal/app` are written against the real exported API of 003-008 when 009 is
  built; the names under *Expects* are the consumer side.
