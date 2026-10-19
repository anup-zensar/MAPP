# Data Model: Home Dashboard

**Feature**: `009-home-dashboard` · **Plan**: [plan.md](plan.md)

Home stores almost nothing: **one preference**. Everything else is **derived** on each request from the
records of features 002–008 and the activity log (001/011), with their rules (domain model §2, §4), so
every figure equals the one on the record's own page (FR-017). No new table, no new ID prefix, no history
kind. JSON names are those of the API. Money is `{"amount": "1234.50", "currency": "EUR"}` (002), always in
the **effective display currency** (§2).

---

## 1. Stored: preference `homeScope` (table `preferences`, extended)

| Field | Type | Default | Rule |
|---|---|---|---|
| `homeScope` | enum `myTeams` · `everything` | `myTeams` | The **My teams / Everything** switch, per person (FR-003). Any other value → field error "Choose My teams or Everything.". Display preference: no version, not logged (D-038) |

(001 has `theme`, `sidebarCollapsed`; 003 adds `displayCurrency`, `dateFormat`; 011 adds
`historyIncludeInside`.)

## 2. Derived: scope and currency

| Concept | Rule |
|---|---|
| **Plans in scope** | `myTeams`: every plan with at least one of the person's teams (003), each plan once even when several of its teams are the person's (FR-004). `everything`: every plan. Deleted plans are not stored and never count (FR-019) |
| **Personal cards** | Activities I own, Waiting for my approval, Needs your attention and My activities always cover **all plans** (FR-003 hint, edge case) |
| **Effective display currency** | The person's display currency (003) when it is active (002), otherwise USD; `unavailableCurrency` then names the inactive one (edge case, plan D-201) |
| **Conversion** | Sums are made in USD and converted once at the current rate (002), 2 decimals half up (FR-017, D-010) |
| **Budget used** | `plannedCost × 100 ÷ budget`, rounded half up to a whole number; null when there is no budget (or the budget is 0); **over budget** when planned cost > budget, compared exactly (plan D-176) |

## 3. Derived: summary (`HomeSummary`, FR-005, FR-006, FR-020)

| Field | Rule |
|---|---|
| `scope` | The scope asked for |
| `displayCurrency` · `unavailableCurrency` | §2 |
| `gettingStarted` | `notInTeam` when the person is in no team; `noTeamPlans` when no plan has one of their teams; otherwise null |
| `plans` `{count, published, draft}` | Plans in scope by status (domain model §2.1) |
| `activitiesOwned` `{count, declined, submitted}` | Activities whose owner is the person, all plans; declined and submitted by stored status |
| `waitingForMe` `{count, oldestWaitingSince}` | 008's **Waiting for me** list; `oldestWaitingSince` = the earliest "waiting since" (when the round reached its current level), null when empty |
| `plannedCost` `{planCount, plannedCost, planBudget, budgetUsedPercent, overBudget, plansWithoutBudget}` | Over the plans in scope: Σ planned cost (domain model §4.2); Σ plan budget of the plans that have one (null when none has); budget used and over budget of the totals (§2); the number of plans without a budget |

## 4. Derived: plan row (`HomePlanRow`, FR-008–FR-010)

| Field | Rule |
|---|---|
| `id`, `name`, `year`, `strategicAop`, `status` | From the plan (field catalogue §1 #1, #2, #6, #18) |
| `teams[]` `{id, name, mine}` | The plan's teams; `mine` when the person is a member; the person's teams first, then by name |
| `campaignCount`, `activityCount` | Records in the plan |
| `approvedActivityCount` | Activities Approved, In progress or Complete (domain model §2.2, D-026); with `activityCount` 0 the row shows "No activities" |
| `budget`, `plannedCost`, `budgetUsedPercent`, `overBudget` | §2 per plan; `budget` null shows "—" ("No budget") |

Order: year newest first, then name ignoring letter case, then ID. At most 10 rows; `total` = plans in
scope.

## 5. Derived: attention item (`HomeAttentionItem`, FR-011, FR-012)

A plan, campaign, initiative or activity **whose owner is the person**, in any plan, with at least one
reason:

| `kind` | Applies to | Condition | Data |
|---|---|---|---|
| `declined` | activity | Stored status Declined | `declinedBy`, `declinedAt`, `comment` (008's declining decision) |
| `negativeBalance` | activity | Unassigned planned cost < 0 (domain model §4.3) | `amount` = how much the line items exceed |
| `overAllocated` | plan, campaign | Remaining budget < 0 (domain model §4.1, D-015) | `amount` = −remaining budget |
| `overBudget` | plan, campaign, initiative | A budget is set and planned cost > budget (domain model §4.2) | `amount` = planned cost − budget |
| `missingFields` | all four | Own needed-to-publish/submit fields missing, or items below needing attention (domain model §2.6) | `missing[]` `{id, label}`, `itemsBelow` |

- A record appears **once** with all its reasons, in the order of the table.
- Items are ordered by their first reason's group — declined, balance, budget (`overAllocated`,
  `overBudget`), missing fields — then by the record's last change, newest first, then by ID.
- Line items never appear on their own: their problems count in their activity's `itemsBelow`
  (Assumptions).
- Fields: `recordType`, `recordTypeLabel`, `id`, `name`, `plan` (a `RecordRef`; the plan itself for a
  plan), `path` (ancestors, plan first; `[]` for a plan), `changedAt`, `reasons[]`.

## 6. Derived: waiting item (`WaitingItem`, FR-013)

`{id, name, plan, owner, plannedCost, level, levels, waitingSince}` for each entry of 008's **Waiting for
me** list (Submitted activities whose current level lists the person), oldest first; first 5 plus `total`.

## 7. Derived: my activities (`MyActivitiesPage`, FR-014)

Groups by **displayed** status (domain model §2.3; In progress and Complete derived from dates, an
ongoing activity is never Complete) in the order Declined, Created, Submitted, Approved, In progress,
Complete; empty groups left out. Each row: `id`, `name`, `plan`, `campaign`, `initiative` (`RecordRef`s),
`status`, `startDate`, `endDate` (null when not set), `ongoing`, `plannedCost` (null when not set),
`missing[]`, `itemsBelow` (line items needing attention). Order within a group: start date (none last),
name, ID.

| Status | Word | Tone |
|---|---|---|
| `created` | Created | neutral |
| `submitted` | Submitted | review |
| `declined` | Declined | problem |
| `approved` | Approved | positive |
| `inProgress` | In progress | active |
| `complete` | Complete | done |

## 8. Derived: recent change (FR-015)

The 10 newest activity-log entries (011's `ActivityLogEntry`, with `recordNow`) whose record type is
`plan`, `campaign`, `initiative`, `activity` or `lineItem` and that belong to a plan in scope (011's
scope matching: the plan itself, anything with the plan in its stored path, or moved into it). A record
whose `recordNow.state` is `inRecycleBin` shows "(in recycle bin)" without a link.

## 9. ~~Derived: team colour~~ — withdrawn (D-201): the colour is `TeamRef.colour`, stored by feature 003

`teamColor(teamId)` = `var(--team-<n>)`, `n = 1 + (sum of the UTF-16 code units of teamId) mod 8`. The
same team has the same colour on every screen and in both themes; the pill always shows the team's name
as text (SC-007).

## 10. Statuses and transitions

Home changes no status. It shows the plan statuses of domain model §2.1 (Draft, Published), the activity
statuses of §2.3 and the approval progress of §2.2, exactly as their features compute them.
