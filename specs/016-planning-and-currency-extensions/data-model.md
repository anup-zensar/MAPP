# Data Model: Planning and Currency Extensions (016)

What this feature stores, what it changes in stored records of 002–007 and 010, and every calculation, with
worked examples. Conventions of 001–010 hold: JSON tables behind repositories, `record.Meta` (version,
created/updated by and at) on every stored record, readable or internal IDs from `ids.Generator`, money as
`money.Amount` (two decimals) with a currency code, rates as `money.Rate` (units per 1 USD, six decimals),
dates as `date.Date` (`YYYY-MM-DD`), derived values computed on read and never stored (D-041).

## 1. Summary of changes

| Table | Change | Owner package |
|---|---|---|
| `exchange_rates` | **New** (prefix `XR`) | `settings` |
| `rate_imports` | **New** (prefix `RI`) | `settings` |
| `field_help` | **New** (id = help key; only edited texts) | `settings` |
| `currencies` | `rate`, `rateChangedAt`, `rateChangedBy` become legacy (read once by the migration, §2.2) | `settings` |
| `plans` | + `currency`, `planType`, `budgetPhasing` | `plans` |
| `campaigns`, `initiatives` | + `budgetPhasing`; `budget.currency` = the plan currency | `campaigns` |
| `activities` | + `splitFromId`; `plannedCost.currency` = the plan currency | `activities` |
| `line_items` | `activityId` may change, by Split only | `lineitems` |
| `saved_views` | view state: `year` → `yearFrom` + `yearTo`; new filters, breakdown, period `year`, `comparePhased` (§5.2) | `allwork` |

New readable-ID prefixes: `XR` (exchange rate, internal), `RI` (rate import, internal).

## 2. Settings

### 2.1 Exchange rate (`exchange_rates`)

| Field | Type | Rules |
|---|---|---|
| `id` | string | `XR-1001`… |
| `currency` | ISO code | An existing currency other than USD; never changes |
| `date` | date | The day the rate applies from; never changes (to move a rate, add one on the new date and delete this one) |
| `rate` | `money.Rate` | Above 0, at most six decimals, at most 1,000,000 (D-053) |
| `source` | `manual` · `gsapImport` | `manual` for Add rate and every Edit (an edited imported rate becomes `manual`) |
| `importId` | string | The `RI-…` that wrote it; `""` for `manual` |
| `record.Meta` | | |

- Unique per (`currency`, `date`): "EUR already has a rate on 19 Oct 2026."
- A currency other than USD always has at least one rate with `date` ≤ today. Delete refuses the last one:
  "EUR needs a rate dated today or earlier." (`validation_failed` on `id`).
- Deleted rates go to the recycle bin (history kind `exchangeRate`, under the currency's kind). Restore is
  refused (`cannot_restore`) when the currency already has a rate on that date or the currency is gone.

### 2.2 Currency (`currencies`) — changed

The stored fields `rate`, `rateChangedAt` and `rateChangedBy` are no longer the source of a conversion. The
**migration** `settings.MigrateRates(ctx)` runs at API start inside one `Write`: for every currency other
than USD that has no `exchange_rates` row, it creates one with `date` = the UTC date of `rateChangedAt`,
`rate` = `rate`, `source` = `manual`, `createdBy` = `rateChangedBy`; then it writes the currency back without
the three fields (`omitempty`). It writes no activity-log entry and does nothing on a second start.

API `Currency` (contract) is computed on read: `rate` and `inverseRate` = the current rate (§3.2);
`rateChangedAt` / `rateChangedBy` = that rate row's `updatedAt` / `updatedBy`; new `rateDate`, `rateSource`
and `rateImportFileName` (null for `manual`); `usage` adds a `plans` part (plans whose currency it is).

`POST /api/currencies` (create) keeps its body: `rate` becomes the first exchange rate dated today.
`PATCH /api/currencies/{code}` with `rate`: adds today's rate or replaces it (`source` `manual`).

### 2.3 Rate import (`rate_imports`)

| Field | Type | Rules |
|---|---|---|
| `id` | string | `RI-1001`… |
| `fileName` | string | 1–200 characters, as uploaded |
| `feed` | string | `gsapDailyRatesFile` (the only feed now, FR-008) |
| `firstDate`, `lastDate` | date or null | Earliest and latest date among the rates added or replaced; null when none |
| `added`, `replaced`, `unchanged`, `notUsed`, `problems` | int ≥ 0 | Row counts of the review (§2.4) |
| `record.Meta` | | `createdBy` = the importer; never updated or deleted |

### 2.4 The simulated GSAP daily rate extract

Until Shell confirms GSAP's real layout (Q-012) MAPP reads this file. It mirrors SAP's exchange-rate
table (rate type, from and to currency, valid-from date, rate).

- `.csv`, UTF-8 (a byte-order mark is allowed), comma-separated, at most 2 MB and 20,000 data rows.
- Row 1 is the header, in any order, ignoring letter case and surrounding spaces:
  `rate_type, from_currency, to_currency, valid_from, rate`. A missing column: "The file needs the column
  'valid_from'." (`validation_failed` on `content`; nothing is reviewed).
- Row numbers in messages are the file's own (the header is row 1).

| Row | Outcome | Message |
|---|---|---|
| `rate_type` ≠ `M` (ignoring case) | not used | "Rate type B is not used — only M rates are imported." |
| `from_currency` ≠ `USD` | not used | "Only rates quoted from USD are used." |
| `to_currency` = `USD` | not used | "USD is always 1." |
| `to_currency` not a MAPP currency | not used | "CHF is not set up in Settings › Currencies." |
| `valid_from` not `YYYY-MM-DD` or `YYYYMMDD`, or not a date | problem | "Row 12: 'valid_from' must be a date such as 2026-10-19." |
| `rate` not a number above 0 with ≤ 6 decimals, or > 1,000,000 | problem | "Row 41: the rate must be above 0 with at most 6 decimals." |
| Same currency and date twice in the file | problem (the later row) | "Row 57: EUR on 19 Oct 2026 appears twice (also row 33)." |
| Stored rate on that date equal | already present | — |
| Stored rate on that date different | replaces | "0.9214 → 0.9230" |
| Otherwise | to add | — |

Inactive currencies are imported (they keep converting, 002 FR-004). The review answers every row; the import
re-reads the file and saves the *to add* and *replaces* rows in one `Write`, so a review and an import of the
same file give the same counts.

The **sample file** (`GET /api/exchange-rates/import-sample`) has one `M` row per active currency other than
USD, `valid_from` today, `rate` = the current rate × 1.005 rounded half up to six decimals, named
`gsap_rates_<today>.csv`.

### 2.5 Field help (`field_help`)

| Field | Type | Rules |
|---|---|---|
| `key` | string (the record id) | One of the catalogue's keys (§6); others are refused with 404 |
| `text` | string | 0–300 characters after trimming; plain text, line breaks removed |
| `record.Meta` | | |

Only edited texts are stored. A help entry's API answer combines the catalogue (label, level, group, default)
with the stored row. **Reset to default** deletes the row (no recycle bin) and logs `changed` (text old →
default). History kind `fieldHelp`: logged, never in the recycle bin. The API version of an entry without a
stored row is 0; an update with version 0 when a row exists answers `version_conflict`.

## 3. Calculations

### 3.1 Converting amounts

`money.Convert(a, from, to Rate) Amount` = `a × to ÷ from`, rounded half up to the cent; the same rate on
both sides returns `a` unchanged. `ToUSD(a, r)` = `Convert(a, r, One)` and `FromUSD(a, r)` =
`Convert(a, One, r)` (existing functions keep their results). `ConvertParts(parts, from, to)` converts each
part except the last with `Convert` and gives the last `Convert(sum) − others` (D-024, generalises
`ConvertSpread`).

| Amount | Calculation | Example (EUR 0.92, GBP 0.79 per USD) |
|---|---|---|
| Plan, campaign, initiative budget; activity planned cost | Typed in the plan currency; stored as typed | 600,000.00 GBP |
| Line item in the plan currency | `Convert(cost, rLine, rPlan)` | 23,000.00 EUR → 19,750.00 GBP |
| Line item in USD | `Convert(cost, rLine, 1)` (unchanged, D-141) | 23,000.00 EUR → 25,000.00 USD |
| Unassigned planned cost (plan currency) | activity planned cost − Σ line items in the plan currency (each rounded) | 40,000.00 − (15,000.00 + 19,750.00) = 5,250.00 GBP |
| Roll-ups inside a plan | Sums in the plan currency, exact | — |
| Amount beside, inside a plan | `Convert(amount, rPlan, rDisplay)`, each shown amount converted once (a total from its plan-currency total) | 5,250.00 GBP → 6,645.57 USD |
| Plans list and Home totals | Each plan's total converted once with `Convert(total, rPlan, 1)`, the USD values added, the sum converted once to the display currency | — |
| All work | Every record's amount converted to USD once (budget per record, planned cost per activity, line items per line item, as 010 does), summed in USD, converted once for display | — |
| Activity type default planned cost | `Convert(default, 1, rPlan)` | 5,000.00 USD → 3,950.00 GBP |
| Rounding hint (D-145) | Measured in the plan currency: balance ≠ 0.00 and \|balance\| ≤ 0.05 and a line item uses another currency than the plan's | — |

Pages that add per record (All work) and pages that convert per plan (Plans, Home) may differ by a cent for a
non-USD plan, as D-213 already accepts for display conversions.

### 3.2 Current rate

`current(c, today)` = the rate of currency `c` with the greatest `date` ≤ `today` (today = the API clock's UTC
date, D-091); USD = 1. `settings.(*Service).Rates(ctx)` keeps its signature and answers the current rates;
`RatesOn(ctx, d)` answers the rates current on `d` (used by tests and by the impact check of a future-dated
rate, which reports "No conversion changes before 01 Jan 2027").

**Rate used** (`RateUsed` in the contract): `{currency, rate, inverseRate, rateDate, source,
importFileName}` — the current rate of a currency and where it came from. `GET /api/exchange-rates/current`
answers every currency's; the web app shows them in tips (FR-009) and never converts.

### 3.3 Budget phasing

```
Phasing { method: none | evenly | custom; months: [ { month: "2027-03", amount: "50000.00" } ] }
```

- `months` is stored only for `custom`: sorted, at most one entry per month, months of the plan's year only,
  amounts ≥ 0.00 with at most two decimals; a month without an entry is 0.00. Otherwise `months` is `[]`.
- **Monthly amounts** (12, January to December of the plan's year): `none` → no amounts; `evenly` →
  `budget.SplitEvenly(12)` (600,000.00 → 50,000.00 each; 100,000.00 → 8,333.33 × 11 and 8,333.37 in
  December); `custom` → the stored months.
- **Quarter entry** (web only): a quarter amount `q` fills its months with `q.SplitEvenly(3)` (100,000.00 →
  33,333.33 / 33,333.33 / 33,333.34). Quarterly display adds three months.
- **Problem**: `custom` whose months do not add up to the budget → a needed-to-publish problem of the record:
  `MissingField {field: "budgetPhasing", label: "Budget by month", reason: "Adds up to 590,000.00 GBP, not
  the budget 600,000.00 GBP"}`. It is part of the record's completeness and of 004's publish check.
- **Budget removed** (`budget` null): the saved phasing becomes `none` in the same save; the form says so at
  the budget field before saving ("Budget by month will be removed").
- **Validation** of a phasing save: method other than `none` without a budget → "Set a budget before phasing
  it."; a month outside the plan's year → "Months must be in 2027."; negative or more than two decimals → the
  money messages of 002.

### 3.4 Planned cost by month (plan currency)

For a plan, campaign or initiative, in the plan currency, by month of the plan's year plus `otherYears` and
`noDates`, from every activity below:

1. Each line item's months by its spread (domain model §4.4, D-140), converted with
   `ConvertParts(months, rLine, rPlan)`; an amount that cannot be spread (activity without dates) goes to
   `noDates` converted with `Convert`.
2. A **positive** unassigned balance (§3.1) spread over the activity's months with `SplitEvenly` (an ongoing
   activity: its first month); without dates → `noDates`. A negative balance is never spread (010 FR-023).
3. Months outside the plan's year → `otherYears`.

Rows of the Budget by month card (per month and per quarter): **Budget** (§3.3), **Planned cost** (this
section), **Difference** = Budget − Planned cost (only when phased), **Allocated to campaigns / initiatives**
= Σ of the children's monthly budgets (children `none` add nothing; the card says "2 campaigns are not
phased"). A period is **over** when Planned cost > Budget, or when Allocated > Budget, and shows "Over by X".

### 3.5 Breakdown (All work and the plan page)

Input: the activities that match the filters (010 FR-005) and their line items. Amounts in USD per record
(All work; then converted once per shown amount to the display currency) or in the plan currency (plan page,
one plan).

| Grouping | Value of an activity |
|---|---|
| Shell attribute at level plan, campaign or initiative | The value on the activity's ancestor at that level |
| Shell attribute at level activity | The activity's own value |
| Shell attribute at level line item | Each line item's own value (see below) |
| Plan · Plan type · Plan currency · Year | Of the activity's plan |
| Team · Business unit | The plan's teams · the plan's Business unit options (several → each) |
| Activity type · Owner | The activity's (no owner → "No owner") |

- Empty value, or hidden by its business-unit condition → group **Not filled in**.
- Single choice, yes/no (groups "Yes", "No") → one group, the full amount.
- Multiple choice without split → the full amount in each chosen group; the answer sets `countsInEach` and
  the totals row counts every activity once.
- Multiple choice with % split → `amount × share ÷ 100` half up per option, in the value's option order,
  the last option getting `amount − Σ others`. Germany 60 % / France 40 % of 10,000.00 → 6,000.00 / 4,000.00.
- Line-item attribute → each line item's planned cost goes to its own value's group(s); each activity's
  unassigned planned cost (positive or negative) goes to **Not on a line item**. Line items of activities
  without a planned cost are included and the answer's `note` says "Includes <X> on line items of activities
  without a planned cost" (the totals row then differs from the Planned cost card by exactly X).
- Per group: `activities` (distinct activities with an amount or a value in the group), `lineItems`
  (line-item groupings: distinct line items; otherwise the line items of the group's activities),
  `plannedCost`, `share` = plannedCost ÷ total planned cost × 100, half up to two decimals.
- **Budget** (groupings whose value comes from a plan, campaign or initiative: Shell attributes of those
  levels, Plan, Plan type, Plan currency, Year, Team, Business unit): Σ budgets of the records of that level
  whose value is in the group (several values → each), counting only records shown by the filters; **Budget
  used** as D-090. Other groupings: null.
- **Then by**: the same rules inside each group.
- Groups are ordered by Settings' option order (attributes), by name otherwise, then **Not filled in** and
  **Not on a line item** last.

### 3.6 Split activity

Input: `name`, `initiativeId`, `lineItemIds`, `plannedCost` (decimal string in the plan currency or null),
`restartApproval`, `version`.

| Rule | Problem (field) |
|---|---|
| `name` 1–120, unique in the target initiative ignoring case and spaces | "'Events' already has an activity called 'Ireland roadshow'." (`name`) |
| Target initiative in the same plan, dates containing the activity's (ongoing: start only) | Move's messages of 006 (`initiativeId`) |
| Each line item belongs to the activity | "L-1009 is not a line item of this activity." (`lineItemIds`) |
| `plannedCost` 0.00 – original planned cost; null or 0.00 when the original has none | "Move at most 40,000.00 GBP, the activity's planned cost." (`plannedCost`) |
| At least one line item, or plannedCost > 0.00 | "Choose line items or an amount to move." (`lineItemIds`) |
| Original Submitted or Approved and `restartApproval` not true | D-189's `validation_failed` on `restartApproval` |
| `version` = the original's | `version_conflict` |

Result: original `plannedCost` = old − moved (null stays null); new activity: copied fields (FR-028),
`plannedCost` = moved (null when the original had none), goals empty, `status` created, `splitFromId` =
original's id; ticked line items get `activityId` = new id. Unassigned balances follow §3.1. "Split into" is
derived: the live and binned activities whose `splitFromId` is this id.

## 4. Planning records

### 4.1 Plan (`plans`) — added fields

| Field | Type | Rules |
|---|---|---|
| `currency` | ISO code | Required; an active currency, or the one the plan already has; `""` in older data reads as `USD`; locked by FR-012 (count = campaign and initiative budgets set + activity planned costs set; `validation_failed` on `currency` with the lock text) |
| `planType` | `operational` · `strategic` · `crossMarket` | Required; default `operational`; `""` in older data reads as `operational` |
| `budgetPhasing` | Phasing (§3.3) | Default `{method: none, months: []}`; changed only by `PUT /api/plans/{planId}/budget-phasing` (and by a budget removal) |

`budget` stays `*money.Amount`, now in the plan currency. Create and update bodies add `currency` and
`planType`; the API answers `budget` as Money in the plan currency.

### 4.2 Campaign and initiative — added field

`budgetPhasing` (§3.3), changed only by `PUT /api/campaigns/{id}/budget-phasing`,
`PUT /api/initiatives/{id}/budget-phasing` and by a budget removal. `budget.currency` must be the plan's
currency ("Enter the amount in GBP, the plan's currency.", `validation_failed` on `budget`).

### 4.3 Activity — added field and changed rule

| Field | Type | Rules |
|---|---|---|
| `splitFromId` | string | The activity it was split from; `""` otherwise; set only by Split; never copied |
| `plannedCost` | Money | Currency = the plan currency (`validation_failed` on `plannedCost`: "Enter the amount in GBP, the plan's currency.") |

### 4.4 Line item — changed rule

`activityId` changes only through Split (§3.6); everything else stays. API `LineItem` adds
`plannedCostPlan` (Money, plan currency); `UnassignedBalance` and `SpreadGrid` are in the plan currency
(display variants stay in the display currency, D-213).

### 4.5 Normalising older data

Repositories normalise on read: plan `currency` `""` → `USD`; `planType` `""` → `operational`; a missing
`budgetPhasing` → none. Stored USD budgets and planned costs of existing plans are already in their plan
currency (USD). No rewrite is needed.

## 5. All work

### 5.1 `allwork.Node` — added fields (filled by `internal/app/allwork_source.go`)

| Field | For | Meaning |
|---|---|---|
| `PlanYear int` | every node | The plan's year (the range spans several) |
| `PlanType string`, `PlanCurrency string` | plans | FR-032, FR-011 |
| `BudgetPlan *money.Money` | plans, campaigns, initiatives | Budget in the plan currency (column "Plan budget (plan currency)") |
| `PlannedCostPlan *money.Money` | activities, line items | In the plan currency (column "Planned cost (plan currency)") |
| `PhasedBudgetUSD []MonthAmount` | plans, campaigns, initiatives | The 12 monthly budgets converted to USD with `ConvertParts`; nil when `none` |
| `Attrs map[string]AttrValue` | every level | Shell attribute values by attribute id: `{OptionIDs []string; Shares map[string]money.Percent; Yes *bool; Shown bool}` for filters and Breakdown |

`Budget`, `PlannedCost`, `MonthsUSD`, `UnspreadUSD`, `Revenue` stay in USD: the adapter converts the
plan-currency values with §3.1. `Source.Year` is called once per year of the range.

### 5.2 View state and saved views

`AllWorkViewState`: `year` is replaced by `yearFrom` and `yearTo` (`yearTo − yearFrom` ≤ 4); `view` adds
`breakdown`; `period` adds `year`; new `comparePhased` (bool), `breakdown` `{by, thenBy}` (grouping keys:
`attr:<attributeId>`, `plan`, `planType`, `planCurrency`, `year`, `team`, `businessUnit`, `activityType`,
`owner`); `filters` adds `planTypes`, `planCurrencies`, `attributes` (`[{attributeId, values}]`, values =
option ids, `yes`, `no` or `none` for Not filled in). Saved views stored before 016 are read with
`yearFrom = yearTo = year`, the new filters empty, `comparePhased` false, `breakdown` null.

## 6. Field help catalogue

Keys are stable; labels are the screen labels; texts are the defaults (≤ 300 characters). The catalogue lives
in `backend/internal/settings/fieldhelp_catalogue.go`. Groups order the Settings page.

| Key | Label | Default text |
|---|---|---|
| `plan.name` | Plan name | A name everyone recognises, such as "FY2027 Lubricants UK & Ireland". It must be unique in MAPP. |
| `plan.year` | Year | The fiscal year the plan covers, January to December. It can't change after the plan is saved. |
| `plan.planType` | Plan type | Operational: one business's or market's plan for the year. Strategic: sets direction for several years. Cross-market: campaigns that run across several markets. |
| `plan.model` | Model | The plan's rulebook: which activity types it allows and how inquiries turn into MQLs, deals and revenue. |
| `plan.teams` | Teams | The teams that work on this plan. The owner must belong to one of them. |
| `plan.owner` | Owner | The person responsible for the plan. Choose a member of the plan's teams. |
| `plan.aop` | Strategic / AOP plan | Marks the plan as part of the annual operating plan. AOP plans carry an "AOP" tag in lists. |
| `plan.description` | Description | What the plan is for, in a few sentences. |
| `plan.currency` | Plan currency | The currency the plan's budgets and planned costs are typed in. Each amount also shows in USD or your display currency. It can't change once amounts are planned in it. |
| `plan.budget` | Plan budget | The money this plan may spend in its year, in the plan currency. Campaign budgets are allocated from it. |
| `plan.budgetPhasing` | Budget by month | How the budget is spread over the months of the year: not phased, evenly or month by month. It is compared with each month's planned cost. |
| `plan.defaultFinanceBudget` | Default finance budget | The finance budget the plan's line items link to by default. Its items are listed first on each line item. |
| `plan.defaultSpread` | Default line-item spread | How new line items spread their cost over months, unless someone changes it on the line item. |
| `plan.bcpir` | BCPIR plan | Marks the plan as a BCPIR plan. |
| `plan.goalStage` | Goal stage | The funnel stage the plan's goal is set at: INQ, MQL or Revenue. The other stages are calculated with the model. |
| `plan.goalAmount` | Goal amount | The plan's goal at the chosen stage: a number of inquiries or MQLs, or revenue in USD. |
| `plan.status` | Status | Draft while the plan is prepared; Published once it is the agreed plan. A published plan can still be edited. |
| `plan.plannedCost` | Planned cost | The sum of the planned costs of every activity in the plan. |
| `plan.remainingBudget` | Remaining budget | Plan budget minus the budgets given to campaigns. Below zero means campaigns have more budget than the plan. |
| `plan.budgetUsed` | Budget used | Planned cost divided by the plan budget. Red above 100 %. |
| `plan.activitiesApproved` | Activities approved | Activities that are Approved, In progress or Complete, out of all activities in the plan. |
| `plan.goals` | Goals | The plan's goal next to the sum of its activities' goals. |
| `plan.plannedCostBy` | Planned cost by | The plan's planned cost added up by a category, such as spend type, brand or country. |
| `campaign.name` | Campaign name | A theme that runs for most of the year. Unique in its plan. |
| `campaign.owner` | Owner | The person responsible for the campaign. It starts as the plan's owner. |
| `campaign.dates` | Dates | When the campaign runs. Expected inside the plan's year; initiatives must fit inside these dates. |
| `campaign.description` | Description | What the campaign is about. |
| `campaign.budget` | Campaign budget | The part of the plan budget given to this campaign, in the plan currency. Initiative budgets are allocated from it. |
| `campaign.budgetPhasing` | Budget by month | How the campaign budget is spread over the months of the plan's year. |
| `campaign.remainingPlanBudget` | Remaining plan budget | Plan budget minus all campaign budgets, including this one. Red when campaigns have more than the plan. |
| `campaign.plannedCost` | Planned cost | The sum of the planned costs of the activities in this campaign. |
| `campaign.budgetUsed` | Budget used | Planned cost divided by the campaign budget. Red above 100 %. |
| `campaign.leftToPlan` | Left to plan | Campaign budget minus planned cost. Below zero means the campaign is over budget. |
| `campaign.approvalProgress` | Approval progress | How many activities below are approved out of all of them. |
| `campaign.timing` | Timing | Not started, Running or Finished, compared with today's date. |
| `campaign.goals` | Goals | The sum of the goals of the activities below. |
| `initiative.name` | Initiative name | A group of similar work inside a campaign. Unique in its campaign. |
| `initiative.campaign` | Campaign | The campaign the initiative belongs to. Change it with Move. |
| `initiative.owner` | Owner | The person responsible for the initiative. It starts as the campaign's owner. |
| `initiative.dates` | Dates | When the initiative runs. Must be inside the campaign's dates; activities must fit inside these. |
| `initiative.description` | Description | What the initiative covers. |
| `initiative.budget` | Initiative budget | The part of the campaign budget given to this initiative, in the plan currency. |
| `initiative.budgetPhasing` | Budget by month | How the initiative budget is spread over the months of the plan's year. |
| `initiative.remainingCampaignBudget` | Remaining campaign budget | Campaign budget minus all initiative budgets, including this one. Red when initiatives have more than the campaign. |
| `initiative.plannedCost` | Planned cost | The sum of the planned costs of the activities in this initiative. |
| `initiative.budgetUsed` | Budget used | Planned cost divided by the initiative budget. Red above 100 %. |
| `initiative.leftToPlan` | Left to plan | Initiative budget minus planned cost. Below zero means the initiative is over budget. |
| `initiative.approvalProgress` | Approval progress | How many activities below are approved out of all of them. |
| `initiative.timing` | Timing | Not started, Running or Finished, compared with today's date. |
| `initiative.goals` | Goals | The sum of the goals of the activities below. |
| `activity.name` | Activity name | One real piece of work, such as "Launch film". Unique in its initiative. |
| `activity.externalName` | External name | The name connected systems use. Leave it empty to use the activity name. |
| `activity.initiative` | Campaign, Initiative | Where the activity sits in the plan. Change it with Move. |
| `activity.activityType` | Activity type | The kind of work. Only the types the plan's model allows are offered. |
| `activity.owner` | Owner | The person responsible for the activity. Needed to submit. |
| `activity.dates` | Dates | When the work happens. Must be inside the initiative's dates. Changing them on an approved activity restarts approval. |
| `activity.ongoing` | Ongoing (no end date) | For work without an end date. It can't be undone once saved. |
| `activity.plannedCost` | Planned cost | What the activity is expected to cost, in the plan currency. Line items divide it into costs. Changing it on an approved activity restarts approval. |
| `activity.approvalWorkflow` | Approval workflow | The approval levels the activity goes through when submitted. |
| `activity.description` | Description | What the activity is and why it is done. |
| `activity.status` | Status | Created, Submitted, Declined, Approved, In progress or Complete. |
| `activity.unassignedPlannedCost` | Unassigned planned cost | Planned cost not yet given to a line item: the activity's planned cost minus its line items. Below zero blocks submitting. |
| `activity.inq` | INQ (inquiries) | The number of inquiries the activity is expected to bring. MQL and revenue are calculated with the plan's model. |
| `activity.setLaterStagesMyself` | Set later stages myself | Type MQL and revenue yourself instead of calculating them with the model. |
| `activity.mql` | MQL | Marketing-qualified leads: INQ × the model's INQ→MQL rate, unless typed. |
| `activity.revenue` | Revenue | Expected revenue in USD: deals × the model's average deal size, unless typed. |
| `activity.splitFrom` | Split from / Split into | The activity this one was split from, or the activities split off it. |
| `activity.fromFinance` | From finance (coming later) | Values the Finance module and GSAP will fill. |
| `lineItem.name` | Line item name | One cost of the activity, such as "Agency fee". Unique in its activity. |
| `lineItem.owner` | Owner | The person responsible for this cost. It starts as the activity's owner. |
| `lineItem.description` | Description | What the cost is for. |
| `lineItem.currency` | Currency | The currency the cost is typed in. It is converted to the plan currency and USD at today's rate. |
| `lineItem.plannedCost` | Planned cost | The cost in the line item's currency. It starts as the activity's unassigned planned cost. |
| `lineItem.plannedCostPlan` | Planned cost (plan currency) | The planned cost converted into the plan currency at today's rate. |
| `lineItem.plannedCostUsd` | Planned cost in USD | The planned cost converted into USD at today's rate. |
| `lineItem.spread` | Spread over time | How the cost is spread over months: evenly, all in the first or last month, or month by month. |
| `lineItem.contractedAmount` | Contracted amount | The amount already agreed in a contract, in the line item's currency. |
| `lineItem.requestSwbs` | Request SWBS code from GSAP | Ask GSAP for an SWBS code once the activity is approved. |
| `lineItem.swbsCode` | SWBS code | Filled by GSAP later. |
| `lineItem.financeBudgetYear` | Finance budget year | The year of the finance budget this cost links to. |
| `lineItem.financeBudgetItem` | Finance budget item | The finance budget item this cost is charged to. Needed to submit. |
| `lineItem.fromFinance` | From finance (coming later) | Actuals, invoices and purchase requests, filled by Finance and GSAP later. |
| `phasing.budget` | Budget | The phased budget of each month or quarter. |
| `phasing.plannedCost` | Planned cost | The planned cost of each month, from the line items' spreads and the unassigned planned cost spread evenly. |
| `phasing.difference` | Difference | Budget minus planned cost. Red when the planned cost is higher. |
| `phasing.allocated` | Allocated to campaigns / initiatives | The phased budgets of the records below, month by month. |
| `common.id` | ID | The record's permanent ID, such as A-1012. |
| `common.completeness` | Completeness | "All filled in", or how many fields are still needed before publishing or submitting. |
| `common.created` | Created | Who created the record and when. |
| `common.lastChanged` | Last changed | Who changed the record last and when. |
| `allWork.campaigns` | Campaigns | The campaigns that match the filters. |
| `allWork.initiatives` | Initiatives | The initiatives that match the filters. |
| `allWork.activities` | Activities | The activities that match the filters. |
| `allWork.plannedCost` | Planned cost | The sum of the matching activities' planned cost, in your display currency. |
| `allWork.planBudget` | Plan budget | The sum of the budgets of the plans shown, in your display currency. |
| `allWork.unassignedPlannedCost` | Unassigned planned cost | The sum of the matching activities' planned cost not yet given to line items. |
| `allWork.breakdownShare` | Share | The group's planned cost as a share of the total. |

All work's column chooser shows, for each column, the help of the field with the same meaning (Plan budget →
`plan.budget`, Planned cost → `activity.plannedCost`, Status → `activity.status`, ID → `common.id`, …; the
mapping is a table in `frontend/src/features/all-work/columnHelp.ts`); Shell-attribute columns show the
attribute's own help.

## 7. Activity-log entries

No new `action` values: new verbs use the existing ones (D-167).

| Change | `recordType` | `action` | Verb / details |
|---|---|---|---|
| Add, edit, delete, restore a rate | `exchangeRate` (name "EUR on 19 Oct 2026") | created / changed / deleted / restored | `changes`: rate old → new; source |
| Currency form rate change | `exchangeRate` | created or changed | as above |
| Rate import | `rateImport` (name = file name) | created | verb "imported"; details "89 exchange rates (87 added, 2 replaced) for 19 Sep – 19 Oct 2026" |
| Field help edit / reset | `fieldHelp` (name "Plan › Plan budget") | changed | `changes`: help text old → new |
| Plan currency, plan type | `plan` | changed | in the plan's ordinary change entry |
| Phasing | `plan`, `campaign`, `initiative` | changed | `changes`: "Budget by month" method and months old → new ("Mar 2027 50,000.00 → 60,000.00 GBP") |
| Split, original | `activity` | changed | verb "split"; details "Into 'Ireland roadshow' (A-1014): 1 line item and 19,750.00 GBP"; `changes`: planned cost |
| Split, new activity | `activity` | created | details "Split from 'UK launch event' (A-1012)" |
| Split, each moved line item | `lineItem` | moved | `changes`: Activity old → new |

## 8. Demo data

Added after the existing seed steps' records, so no existing ID, 2026 total or test fact changes:

- **Rates** (`seed/settings.go`): one `RI-1001` import "gsap_rates_seed.csv" by MAPP; for EUR, GBP, CNY
  and INR one `gsapImport` rate per day for the 30 days before the seed day, each `seeded rate × (1 + d)`
  with `d` cycling −0.6 %, −0.3 %, 0, +0.3 %, +0.6 % (rounded half up to six decimals), and a rate on the seed
  day equal to today's seeded rate (EUR 0.92, GBP 0.79, CNY 7.10, INR 83.50), so current rates are unchanged.
- **Plans** (`seed/plans.go`): P-1001 and P-1002 `operational`, P-1003 `strategic`, all `USD`, no phasing;
  new **P-1004 "FY2027 Lubricants UK & Ireland"** — Priya Raman, team Lubricants Europe, model Lubricants 1.0,
  plan type `crossMarket`, currency `GBP`, budget 600,000.00 GBP phased `evenly`, Draft.
- **Campaigns and initiatives**: campaign "Helix UK & Ireland" (budget 400,000.00 GBP, `custom` phasing
  100,000.00 in each of Mar, Jun, Sep and Nov 2027) with initiatives "UK" (250,000.00 GBP) and "Ireland"
  (150,000.00 GBP), dates 1 Jan – 31 Dec 2027.
- **Activities**: A-1012 "UK launch event" in "UK" (Event, 01 Mar – 31 May 2027, 40,000.00 GBP, Created) and
  A-1013 "Dealer conference" in "Ireland" (Event, 01 Sep – 30 Sep 2027, 25,000.00 GBP, Created).
- **Line items** of A-1012: L-1009 "Venue" 15,000.00 GBP evenly, L-1010 "Agency fee (Ireland)" 23,000.00 EUR
  evenly (= 19,750.00 GBP; unassigned 5,250.00 GBP).
- **Shell attributes**: every seeded attribute gets a help text (FR-048), written in the seed table next to its
  label.
- **Field help**: no stored rows (defaults apply).
