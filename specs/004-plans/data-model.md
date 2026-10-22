# Data Model: Plans

**Feature**: `004-plans` · **Plan**: [plan.md](plan.md) · **Contract**: [contracts/openapi.yaml](contracts/openapi.yaml)

The record this feature stores, its fields, rules, statuses, calculated values and history behaviour.
Conventions (timestamps, text lengths after trimming, `ActorRef`, `RecordRef`, bookkeeping `record.Meta`,
one table per record type, arrays never `null`) are those of
[001's data model](../001-app-shell-and-sign-in/data-model.md). Field meanings come from
[field catalogue §1](../../docs/product/field-catalogue.md) and rules from
[domain model §2.1, §2.6, §3, §4.1–4.2, §4.5, R1, R7, R9, R13, R14](../../docs/product/domain-model.md).

**Markings**: **R** Required (blocks saving) · **P** Needed to publish (blocks publishing) · — optional ·
**calc** calculated, never stored.

---

## 1. Plan — table `plans` (`backend/data/plans.json`)

One fiscal year of marketing for one business. Readable ID prefix **`P`** (`P-1001`, …; 001's `ids` package,
counter `P`, never reused). No new ID prefix is introduced by this feature.

| # | Field (JSON) | Label on screen | Type (stored) | Mark | Default | Validation and message |
|---|---|---|---|---|---|---|
| — | `id` | ID | string `P-<n>` | — | next of counter `P` | given by the server; never changes |
| 1 | `name` | Plan name | string ≤ 120 | R | — | trimmed; empty → "Enter a plan name."; > 120 → "Use 120 characters or fewer."; another stored plan with the same name ignoring letter case → "There is already a plan called '<name>'. Choose another name." (the other plan's name as stored) |
| 2 | `year` | Year | integer | R | current year (UTC clock) | create: current − 1 … current + 5, else "Choose a year from <first> to <last>."; **never changes after the first save** (not in the update body) |
| 3 | `modelId` | Model | string (002 model ID) | R | the only active model, else none | empty → "Choose a model."; unknown → "Choose a model from the list."; inactive and newly chosen → "'<model>' is inactive — choose an active model."; an inactive model already on the plan is kept |
| 4 | `teamIds` | Teams | array of 003 team IDs, sorted by team name, no duplicates | P (≥ 1) | the creator's active teams | unknown → "Choose teams from the list."; inactive and newly added → "'<team>' is inactive — choose an active team." |
| 5 | `ownerId` | Owner | string (003 person ID); `""` = none | P | the creator, when they have an active team | unknown → "Choose an owner from the list."; inactive and newly chosen → "<name> is inactive — choose someone else."; set while `teamIds` is empty → "Choose teams first — the owner must belong to one of the plan's teams."; not a member of any chosen team (R1) → "<person> isn't in any of the chosen teams — choose another owner or clear the field." |
| 6 | `aop` | Strategic / AOP plan | boolean | — | false | — ("AOP" chip when true) |
| 7 | `description` | Description | string ≤ 2,000 | — | "" | trimmed; > 2,000 → "Use 2,000 characters or fewer." |
| 8 | `budget` | Plan budget | decimal string in USD (`"1200000.00"`) or `null` | — | null | request `{amount, currency}`: currency ≠ USD → "Enter the plan budget in USD."; not a number → "Enter an amount, for example 1200000 or 1,200,000.00."; > 2 decimals → "Use at most 2 decimals."; negative → "Plan budget can't be negative." |
| 9 | `defaultFinanceBudgetId` | Default finance budget | string (002 finance budget ID); `""` = none | — | none | unknown → "Choose a finance budget from the list."; year ≠ plan year → "Choose a finance budget of <year>."; inactive and newly chosen → "'<budget>' is inactive — choose an active finance budget." |
| 10 | `defaultSpread` | Default line-item spread | `evenly` \| `firstMonth` \| `lastMonth` \| `custom` | — | `evenly` | other → "Choose how line items are spread." (labels Evenly · All in first month · All in last month · Custom) |
| 11 | `bcpir` | BCPIR plan | boolean | — | false | — (Q-001) |
| 12, 13 | `attributes` | Business unit, Strategic Marketing Plan (SMP), and every other plan-level Shell attribute | object: attribute ID → value (opaque to plans; shapes defined by 002: string, boolean, integer, array of option IDs) | per Settings (Business unit: P) | `{}`; yes/no attributes absent = No | checked by Settings (002): types, known and active options, dependent options, lengths; field names `attributes.<attributeId>`. Values of hidden attributes (business-unit condition not met) are **kept** (R9) |
| 14 | `goalStage` | Goal stage | `inq` \| `mql` \| `revenue` | P | `inq` | other → "Choose INQ, MQL or Revenue." |
| 15 | `goalAmount` | Goal amount | decimal string or `""` = none | — | none | not a number → "Enter a number, for example 250000."; negative → "The goal can't be negative."; `inq`/`mql` with decimals → "Enter a whole number."; `revenue` (USD) with > 2 decimals → "Use at most 2 decimals." |
| 18 | `status` | Status | `draft` \| `published` | calc (system) | `draft` | changed only by Publish / Unpublish |
| — | `version`, `createdAt`, `createdBy`, `updatedAt`, `updatedBy` | Created by … on …, last changed by … on … | `record.Meta` | — | — | every update sends `version`; missing → "Reload the page and try again."; out of date → `409 version_conflict` "Not saved — <person> changed this plan" |

Dropped Shell fields (field catalogue §1): ShellAllPlans (D-018), Related plans, Custom goals.

**Relations** (foreign keys by ID, relational in shape):

| From | To | Cardinality | Owner of the target |
|---|---|---|---|
| `modelId` | Model | many plans → 1 model | 002 |
| `teamIds[]` | Team | many ↔ many (a join table in PostgreSQL) | 003 |
| `ownerId` | Person | many plans → 0..1 person | 001/003 |
| `defaultFinanceBudgetId` | Finance budget (year + name group of items) | many plans → 0..1 budget | 002 |
| `attributes` keys | Shell attribute (level `plan`) and its options | values per plan | 002 |
| Campaign `planId` | Plan | 1 plan → many campaigns | 005 |

A plan never stores lists of its children; campaigns point to their plan.

---

## 2. Status and transitions (domain model §2.1)

```
           create / copy
               │
               ▼
           ┌───────┐   Publish (all needed-to-publish fields of the plan,     ┌───────────┐
           │ Draft │ ──── its campaigns and initiatives filled) ───────────► │ Published │
           └───────┘ ◄──── Unpublish (reason 1–500 characters, logged) ────── └───────────┘
               ▲                                                                   │
               └──────────── Restore from the recycle bin (not Undo) ──────────────┘
```

| Transition | Allowed when | Effect |
|---|---|---|
| create, copy | always | status Draft |
| Publish | status Draft **and** readiness has no missing field | status Published; log `statusChanged` "published" |
| Publish refused | readiness lists fields | `409 not_ready` with every missing field grouped by record; nothing changes |
| Publish of a Published plan | — | nothing changes, nothing logged |
| Unpublish | status Published; reason given | status Draft; log `statusChanged` "unpublished" with the reason as comment |
| Unpublish of a Draft plan | — | nothing changes, nothing logged |
| Edit | any status | status unchanged (FR-022) |
| Undo of a delete | within 2 minutes; name not taken | exactly as before, status included |
| Restore from the recycle bin | parent check n/a (top level) | Published → Draft; name clash → " (restored)", " (restored 2)", … |

**Readiness** (what Publish checks) = the plan's own completeness `missing` list + every item below that
blocks publishing (campaigns and initiatives, from 005). Activities, line items, a negative remaining budget
and campaign dates outside the year never block (FR-024).

---

## 3. Calculated values (never stored; recalculated on every read)

All sums in USD with exact decimals; converted to the reader's **display currency** at the end (rate =
units per 1 USD, half up to the cent); percentages are whole numbers rounded half up.

| Value | Formula | Empty / special |
|---|---|---|
| Planned cost | sum of the plan's activities' planned cost (via campaigns, domain §4.2) — from the `activity` level | null ("—") when nothing below has a planned cost |
| Allocated to campaigns | sum of campaign budgets — from the `campaign` level | 0.00 without campaigns |
| Remaining budget | plan budget − allocated to campaigns (domain §4.1) | null without a plan budget; negative allowed, shown red with "Over-allocated by <−remaining>" (D-015) |
| Budget used | planned cost (0 when null) × 100 ÷ plan budget | null without a budget or with budget 0 ("Plan budget is 0"); "over budget" when planned cost > budget → red bar |
| Activities approved | activities in Approved (stored), shown as Approved, In progress or Complete, "n of m" | 0 of 0 until 006 |
| Approval progress | No activities (m = 0) · All approved (n = m > 0) · In progress (otherwise) | D-026 |
| Plan goal | waterfall from `goalStage`, `goalAmount` and the model's current rates and deal size (domain §4.5; algorithm in plan.md) | each stage null with a reason when a rate or the deal size is 0; all null without a goal amount |
| Activity goals | sums of every activity goal in the plan: INQ, MQL, revenue (USD) — from the `activity` level | 0 until 006 |
| Share planned | activity total of the goal stage × 100 ÷ plan goal of that stage | null without a goal or with a goal of 0 |
| Completeness — own | needed-to-publish fields that are empty, in form order: Teams, Owner (empty or not in a team, R1), shown active needed attributes (incl. Business unit) from Settings, Goal stage | "All filled in" when none |
| Completeness — items below | every campaign, initiative, activity and line item below that is not complete, with its missing fields | from 005–007 |
| Contents | counts of campaigns, initiatives, activities (and approved ones) and line items | for the copy and delete dialogs |
| Business unit | the chosen options of the Business unit attribute, with labels and active flags | for the header, cards and table |

List summary (Plans page): over plans in the team scope and the chosen year — number of plans, Draft,
Published, total planned cost (known ones; null when none), total plan budget (set ones; null when none).

---

## 4. History

**Kind** `plan` (registered by `plans.Service.Kinds()`, before 005's kinds):

| Part | Value |
|---|---|
| Name / Singular / Plural | `plan` / `plan` / `plans` |
| Parent | none (top level); 005 registers `campaign` with parent `plan` |
| Snapshot | the stored plan; name = plan name; parent id "" |
| Remove / Put | delete from / write back to `plans` exactly |
| Describe | labels: Plan name, Year, Model, Teams, Owner, Strategic / AOP plan, Description, Plan budget, Default finance budget, Default line-item spread, BCPIR plan, Goal stage, Goal amount, Status, and each attribute's label for `attributes.<id>`; texts: names of models, teams, people, finance budgets; "1,200,000.00 USD"; Yes / No; spread and stage words; option labels from Settings |
| PrepareRestore | name taken by another plan: Undo refused (`name_taken`), Restore renames " (restored)" … with a note; Restore of a Published plan → Draft with the note "Returned as Draft — publish it again when it's ready." |

**Activity-log entries** (one per change, written in the same `Write` as the change):

| Action | `action` / `verb` | Changes / comment / details |
|---|---|---|
| Create | `created` / created | — |
| Edit (something changed) | `changed` / changed | every changed field, `attributes.<id>` one change per attribute |
| Edit (nothing changed or refused) | none | — |
| Publish | `statusChanged` / published | status Draft → Published |
| Unpublish | `statusChanged` / unpublished | status Published → Draft; comment = reason |
| Copy | `copied` / copied (on the new plan) | details "from '<source>' (<source ID>)[, with <counts>]" |
| Delete | `deleted` (by history) | details "with <counts>" |
| Undo / Restore | `restored` (by history) | details "Undo…" / "Restored from the recycle bin…" + the kind's notes |

**Recycle bin**: deleting a plan moves it and every campaign, initiative, activity and line item below it
into **one** bin entry (`contents` counts per kind); Undo within 2 minutes restores them exactly with the
same IDs and versions.

---

## 5. Copy (R13, FR-026–FR-028)

| Element | Copy |
|---|---|
| New plan | new `P-` ID; every field of §1 copied (owner, teams, budget, goal, attribute values, AOP, BCPIR, spread, model, description); `status` Draft; bookkeeping new (creator = the person copying); no log history of the source |
| Name | given in the dialog; suggested "<name> (copy)", then "<name> (copy 2)", "(copy 3)", … (first not used by a stored plan, ignoring case); the original part is shortened (trailing spaces removed) so the whole name is ≤ 120 characters |
| Same year | `year` unchanged; dates of copied records unchanged |
| Next year | `year` + 1 (only when ≤ current year + 5); every copied date + 1 year with `ShiftDate` (29 Feb → 28 Feb, no end date stays none); `defaultFinanceBudgetId` cleared with the note "Default finance budget cleared — it belonged to <year>."; 007 defines how line items' finance budget links move |
| Contents (option on by default) | each level copies its records in registration order (campaign → initiative → activity → line item), new IDs, activities Created, no approvals, comments, history or GSAP values; off → only the plan |
| Atomicity | one `Write`: all or nothing |

---

## 6. Web-app state (not stored on the server)

| Key | Where | Value |
|---|---|---|
| `mapp.plans.list.<personId>` | `localStorage` (try/catch; missing or invalid → defaults) | `{status: 'all' \| 'draft' \| 'published', year: number \| null, scope: 'all' \| 'myTeams', layout: 'cards' \| 'table'}` |
| address `/plans?status=&year=&scope=&q=` | URL | wins over the remembered values; invalid values ignored |
| plan form values | component state | copied once from the plan (with `version`); background refetches never change them (001's edit-form rule) |
