# Data Model: All Work Views

**Feature**: `010-all-work-views` · **Plan**: [plan.md](plan.md) · **Contract**: [contracts/openapi.yaml](contracts/openapi.yaml)

All work stores only two kinds of **personal settings**. Everything else it shows is read from the records
of features 002–008 and calculated per request. Conventions of feature 001's data model apply (UTC
timestamps, text lengths in Unicode characters after trimming, arrays never null).

---

## 1. Stored records

### 1.1 Saved view — table `saved_views`

A person's named All work state (domain model §7). Not a business record (decision D-121): no `version`, no
activity-log entry, no recycle bin; nobody but its owner can list, read, change or delete it (FR-030).

| Field | Type | Rules |
|---|---|---|
| `id` | string | `SV-1001`, … (readable-ID counter `SV`, proposed prefix) |
| `personId` | string | The owner; set from the actor on create; never changes |
| `name` | string | **Required** ("Enter a name for the view."); trimmed; at most 60 characters ("Use 60 characters or fewer."); unique among the owner's views ignoring letter case ("You already have a view called '<name>'.") (FR-029) |
| `isDefault` | boolean | At most one `true` per person; setting it clears the person's other default in the same Write (FR-031) |
| `state` | ViewState (§1.2) | Validated on create and update ("The view can't be saved — reload the page and try again.") |
| `createdAt` | timestamp | Set once |
| `updatedAt` | timestamp | Every saved change |

Life cycle: created (Save current view / Save as new view) → changed (Update view, rename, set or remove
default) → deleted (answer = the deleted view; **Undo** creates a new view with the same name, state and
default flag, so its ID changes).

### 1.2 ViewState (inside a saved view; also the web app's session state)

| Field | Type | Rules |
|---|---|---|
| `view` | `grid` \| `calendar` \| `budget` | |
| `year` | integer | 2000–2100 |
| `filters` | Filters (§1.3) | |
| `search` | string | at most 200 characters |
| `columns` | list of column keys (§6) | ordered; at most 100; no duplicates; `name` never listed (always shown) |
| `sort` | `{column, direction}` or null | `direction` = `asc` \| `desc`; null = the plan-tree order |
| `expansion` | `{mode, toggled}` | `mode` = `default` \| `allExpanded` \| `allCollapsed`; `toggled` = row keys switched away from the mode (§4) |
| `showUnassigned` | boolean | default `true` (FR-012) |
| `period` | `month` \| `quarter` | default `month` |
| `calendarLayout` | `timeline` \| `month` | default `timeline` (FR-028) |

### 1.3 Filters

Each filter that holds IDs stores `{id, label}` pairs: the label is the name at the time of saving, so a
value that no longer exists can be named in the "left out" message (FR-031).

| Field | Values | Tested on (FR-004) |
|---|---|---|
| `plans` | `[{id, label}]` (P-…) | plans |
| `planStatuses` | `draft`, `published` | plans |
| `containerStatuses` | `noActivities`, `inProgress`, `allApproved` (Approval progress) · `notStarted`, `running`, `finished` (Timing) | campaigns and initiatives |
| `activityStatuses` | `created`, `submitted`, `declined`, `approved`, `inProgress`, `complete` | activities |
| `owners` | `[{id, label}]` (U-…) | plans, campaigns, initiatives, activities |
| `teams` | `[{id, label}]` | plans (a plan's teams include any chosen team) |
| `businessUnits` | `[{id, label}]` (options of the plan-level Business unit attribute) | plans (the plan's business unit includes any chosen option) |
| `activityTypes` | `[{id, label}]` | activities |
| `strategic` | `include` (default) · `only` · `exclude` | plans |
| `showLevels` | `everything` (default) · `completeBranches` · `campaignsWithoutInitiatives` · `initiativesWithoutActivities` | branches (§3.4) |

An empty list means "no filter". The year is not a filter chip (FR-008).

### 1.4 All work preferences — table `all_work_preferences`

One record per person who changed the grid's columns while no saved view was applied; key = person ID.
Not a business record (like D-038): no version, not logged.

| Field | Type | Rules |
|---|---|---|
| `personId` | string | key |
| `columns` | list of column keys | as ViewState `columns`; default (no record): `id`, `status`, `owner`, `startDate`, `endDate`, `planBudget`, `plannedCost`, `lineItems`, `unassigned` (FR-019) |
| `sort` | `{column, direction}` or null | default null |
| `updatedAt` | timestamp | |

Errors: "Choose at most 100 columns." · "Each column can be chosen once." Unknown keys are kept (an
attribute may come back) and reported in `leftOut` when queried.

---

## 2. Input of every calculation: the year's nodes

`allwork.Source.Year` returns one **Node** per plan of the year and every campaign, initiative, activity
and line item under them (never records in the recycle bin). Nodes carry values computed by the owning
features (plan Dependencies); All work never recomputes them.

| Node field | Levels | Source |
|---|---|---|
| `level`, `id`, `parentId`, `planId`, `order` (position among siblings in the plan tree) | all | 004–007 |
| `name`, `externalName`, `description` | all (`externalName`: activities) | 004–007 |
| `ownerId`, `ownerName`, `ownerInactive` | all | 003 |
| `startDate`, `endDate`, `ongoing` | campaigns, initiatives, activities (plans: 1 Jan – 31 Dec of the year) | 005, 006 |
| `status` (chip: value, label, tone, detail) | plan status · Approval progress · activity display status · line-item completeness | 004, 005, 006, 007 |
| `timing` | campaigns, initiatives | 005 |
| `missing` (own needed fields) | all | 004–007 |
| `swbs` | line items | 007 |
| `budget` (USD) | plans, campaigns, initiatives | 004, 005 |
| `plannedCostUSD` | activities (typed), line items (converted) | 006, 007 |
| `localPlannedCost` (Money) | line items | 007 |
| `monthsUSD` + `unspreadUSD` | line items | 007 (§4.4 incl. rounding) |
| `inq`, `mql`, `revenueUSD` | activities | 006 (§4.5) |
| `createdAt`, `updatedAt` | all | 004–007 |
| `planStatus`, `strategic`, `teamIds`, `businessUnitIds` | plans | 004 |
| `activityTypeId`, `activityStatus` | activities | 006 |
| `fields` (column key → Cell) | all | 002–007 (level-specific columns, §6) |

Derived in All work: an activity's **balance** = planned cost (empty = 0.00) − Σ its line items' USD
planned cost (domain model §4.3).

---

## 3. Matching (FR-005 – FR-008)

### 3.1 Filter tests on one record

For record *r* and active filter *f* with tested levels *T(f)* (§1.3):

1. *r*'s level ∈ *T(f)*: *f* holds when *r* meets it.
2. Every level of *T(f)* is above *r*'s level: *f* holds when the **nearest ancestor** of *r* whose level is
   in *T(f)* meets it (a campaign/initiative status filter on an activity is judged on its initiative).
3. Every level of *T(f)* is below *r*'s level: *f* holds when **any descendant** of *r* is a match (3.2).

Line items are never tested; they take the role of their activity. Campaign / initiative status: values of
the Approval-progress group combine with OR, values of the Timing group with OR, and the two groups with
AND; a record without dates has no Timing value.

### 3.2 Match and context

- *r* is a **match** when every active filter holds for it and it meets the search (3.3).
- *r* is shown when it is a match or has a shown descendant; a shown record that is not a match is a
  **context** row ("Shown for context"), never counted in cards, count or totals.
- Children of a match that are not matches themselves are not shown — except under a search (3.3).

### 3.3 Search (FR-006)

The text is trimmed and compared ignoring letter case. A plan, campaign, initiative or activity meets the
search when the text is contained in its **name, ID, external name or description**, or (activities) in
those of one of its line items, or in those of any of its ancestors. A row records which of its own fields
matched (`searchHit.fields`) and, for external name or description, a 60-character excerpt around the
first hit. Line items whose own text matches are marked too.

### 3.4 Show levels (FR-007)

| Choice | Effect after 3.2 |
|---|---|
| Everything | none |
| Complete branches only | remove every plan, campaign and initiative without a shown activity below |
| Campaigns without initiatives | matches are only campaigns that have **no initiative at all** (stored) and meet the filters on their level; their plans are context rows |
| Initiatives without activities | matches are only initiatives with **no activity at all**; their plans and campaigns are context rows |

### 3.5 Count (FR-008)

"Showing *N* of *M* <noun>": noun = activities (campaigns, initiatives for the last two Show-levels
choices); *N* = matches of that level shown; *M* = all records of that level in the year's plans.

---

## 4. Rows and expansion (web app)

- **Row keys**: the record ID (`P-1001`, `C-1001`, `I-1001`, `A-1001`, `L-1001`); unassigned rows
  `A-1001:unassigned`.
- **Default expansion**: plans, campaigns and initiatives open; activities closed (FR-011). With a search:
  rows with a hit below them open, everything else closed (matching containers show their contents,
  collapsed). `allExpanded` / `allCollapsed` open or close everything. A row is open when the mode says so
  XOR its key is in `toggled`.
- **Visible rows** = shown rows whose ancestors are all open, minus unassigned rows when `showUnassigned`
  is off; paged 50 / 100 / 200 per page.

---

## 5. Money: roll-ups, cards, totals (FR-003, FR-013, FR-014, FR-017)

All sums are made in **USD** with exact decimals; each value is converted once to the display currency
(`money.FromUSD`, rate = units per 1 USD, half up) just before it is returned.

| Value | Activity | Container (plan, campaign, initiative) |
|---|---|---|
| Planned cost | typed (empty → "—", counts as 0.00) | Σ planned cost of the **matching activities shown below** |
| Line items | Σ its line items (USD) | Σ over the matching activities below |
| Unassigned planned cost | its balance (red when negative) | Σ of the **positive** balances below; `negativeCount` of those below (decision D-116) |
| Activities | — | number of matching activities below |
| INQ, MQL, Revenue | its values | Σ over the matching activities below ("—" when all empty) |
| Plan budget | — | its own budget (D-117) |
| Remaining budget | — | own budget − Σ budgets of its children (plans, campaigns); red when negative |
| Budget used | — | planned cost ÷ own budget, whole per cent, half up; red above 100 |

- **Unfiltered value** (FR-013): when the set of activities below a container differs from the matching
  activities shown below it, every roll-up cell of the row carries `unfiltered` = the same calculation over
  all activities below.
- A sum of only empty values is "—" (edge case).
- **Summary cards** (FR-003): Campaigns / Initiatives / Activities = number of matches of that level;
  Planned cost = Σ matching activities; Plan budget = Σ budgets of matching plans; Unassigned = Σ positive
  balances of matching activities + `negativeBalanceCount`.
- **Totals row** (FR-014): "Total · N activities" (N = matching activities); money columns sum the
  matching activities; Plan budget sums matching plan rows only → totals equal the cards.

---

## 6. Columns (FR-018 – FR-020)

Name is always shown and is not a column key. A level-specific column is **blank** (null cell) on other
levels and "—" (`empty` cell) for an empty value on its own level. Choice values show their labels;
multiple choices are joined with ", "; split values show shares ("Germany 60 %, France 40 %"); inactive
values are labelled.

### 6.1 Standard group

| Key | Label | Levels | Kind |
|---|---|---|---|
| `id` | ID | all | text |
| `level` | Level | all | text |
| `owner` | Owner | all | text |
| `startDate` / `endDate` | Start date / End date | plans (1 Jan / 31 Dec), campaigns, initiatives, activities ("Ongoing" text) | date |
| `status` | Status | all (D-118) | status |
| `timing` | Timing | campaigns, initiatives | status |
| `completeness` | Completeness | all | status ("All filled in" / "n fields needed") |
| `planBudget` | Plan budget | plans, campaigns, initiatives | money |
| `remainingBudget` | Remaining budget | plans, campaigns | money |
| `budgetUsed` | Budget used | plans, campaigns, initiatives | percent |
| `plannedCost` | Planned cost | all (unassigned rows: their amount) | money |
| `lineItems` | Line items | activities, containers | money |
| `unassigned` | Unassigned planned cost | activities, containers, unassigned rows | money |
| `activities` | Activities | containers (D-119) | number |
| `inq` / `mql` / `revenue` | INQ / MQL / Revenue | activities, containers | number / number / money |
| `createdAt` / `updatedAt` | Created / Last changed | all records | dateTime |

Default columns: `id`, `status`, `owner`, `startDate`, `endDate`, `planBudget`, `plannedCost`,
`lineItems`, `unassigned` (FR-019).

### 6.2 Level groups (field catalogue §1–§5)

| Group | Keys (label) |
|---|---|
| Plan | `plan.year` (Year), `plan.model` (Model), `plan.teams` (Teams), `plan.strategic` (Strategic / AOP plan), `plan.description` (Description), `plan.defaultFinanceBudget` (Default finance budget), `plan.defaultSpread` (Default line-item spread), `plan.bcpir` (BCPIR plan), `plan.goalStage` (Goal stage), `plan.goalAmount` (Goal amount), `plan.goalMql` (MQL goal), `plan.goalRevenue` (Revenue goal), and `attr.<id>` for every plan-level Shell attribute |
| Campaign | `campaign.description` (Description) and `attr.<id>` for every campaign-level attribute (Cluster, Region, …) |
| Initiative | `initiative.description` (Description) and `attr.<id>` for every initiative-level attribute |
| Activity | `activity.externalName` (External name), `activity.type` (Activity type), `activity.ongoing` (Ongoing), `activity.approvalWorkflow` (Approval workflow), `activity.description` (Description), `activity.setLaterStages` (Set later stages myself), and `attr.<id>` for every activity-level attribute |
| Line item | `lineItem.description` (Description), `lineItem.localPlannedCost` (Planned cost in local currency, followed by its currency), `lineItem.currency` (Currency), `lineItem.spread` (Spread over time), `lineItem.contractedAmount` (Contracted amount), `lineItem.requestSwbs` (Request SWBS code from GSAP), `lineItem.swbsStatus` (SWBS request), `lineItem.financeBudgetYear` (Finance budget year), `lineItem.financeBudgetItem` (Finance budget item), and `attr.<id>` for every line-item attribute |

Attributes: every **active** attribute of the level, plus inactive ones that a stored view or preference
still names (labelled "(inactive)"). Fields marked "later" (actuals, invoiced, purchase requests, SWBS
code) are never offered.

---

## 7. Sorting (FR-015)

Default order: plans by name (ignoring case); children in the plan tree's sibling order (`order`); line
items in creation order; the unassigned row last. With a sort column, the children of **every** parent
are ordered by that column (ascending or descending); records without a value come last in both
directions; ties keep the default order; unassigned rows stay last. Status order (lifecycle):

| Status family | Order |
|---|---|
| Plan | Draft, Published |
| Approval progress | No activities, In progress, All approved |
| Timing | Not started, Running, Finished (no dates last) |
| Activity | Created, Submitted, Declined, Approved, In progress, Complete |
| Completeness | by number of fields needed (All filled in = 0 first) |
| SWBS request | Not requested, Waiting for activity approval, Ready to send to GSAP |

---

## 8. Budget & cost periods (FR-022 – FR-024)

Period keys: `2026-01` … `2026-12` (labels Jan … Dec) or `Q1` … `Q4`; `other` (Other years) and `noDates`
(No dates), each present only when some row has an amount in it.

| Row | Amount per period (USD, then converted) |
|---|---|
| Line item | its USD month amounts (007's spread, §4.4 rounding with the difference in the last month); months outside the year → `other`; `unspreadUSD` (activity without dates) → `noDates` |
| Unassigned (balance > 0) | the balance spread **evenly** over the activity's months (rounded down to the cent, remainder in the last month); ongoing → all in the start month; activity without dates → `noDates` |
| Unassigned (balance < 0) | nothing; the activity row shows "Line items exceed the planned cost by X" |
| Activity | Σ of its line items' and its unassigned row's periods |
| Container | Σ of the matching activities below, per period |
| Quarter | Σ of its three months (in USD, before conversion) |
| Total | Σ of every period of the row, `other` and `noDates` included |

Footer: per period Σ over matching activities; grand total; Plan budget = Σ matching plan budgets. When no
matching activity has a negative balance, grand total = Planned cost card.

---

## 9. Status chips and tones

| Level | Values (label → tone) |
|---|---|
| Plan | Draft → neutral · Published → positive |
| Campaign, initiative — Approval progress | No activities → neutral · In progress (detail "n of m activities approved") → review · All approved → positive |
| Campaign, initiative — Timing | Not started → neutral · Running → active · Finished → done · "—" (detail "No dates yet") |
| Activity | Created → neutral · Submitted → review · Declined → problem · Approved → positive · In progress → active · Complete → done |
| Line item — completeness | All filled in → positive · n fields needed → problem |
| Line item — SWBS request | Not requested → neutral · Waiting for activity approval → review · Ready to send to GSAP → positive |

Calendar bars use the tone of the activity status, or of the campaign's or initiative's Approval progress.

---

## 10. Calculated, never stored

| Item | Defined in |
|---|---|
| Filter set (year, filters, search, Show levels) | §1.2–§1.3, §3 |
| Row (key, level, role, search hit, cells, roll-ups, periods) | §3–§8; API schema `AllWorkRow` |
| Period amount | §8 |
| Summary, count, totals | §3.5, §5 |
