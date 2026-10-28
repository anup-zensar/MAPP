# Data Model: Planning Rules Aligned with Shell's Process

**Feature**: `034-planning-alignment` · **Plan**: [plan.md](plan.md) · **Contract**:
[contracts/openapi.yaml](contracts/openapi.yaml)

No table, column, constraint or migration changes. This page states the rules that change on existing
records, the seeded Settings data, and the web app's view state.

## 1. Activity status and line items (rule change, D-322)

| Event on a line item | Activity status before | Activity status after | Approval rounds | Activity log |
|---|---|---|---|---|
| created, duplicated (`copied`), changed (any field, cost and currency included), deleted, restored (Undo or recycle bin) | any (Created, Submitted, Declined, Approved; shown as In progress or Complete) | **unchanged** | unchanged (no round ends, no history entry, no notification) | one entry on the line item, as before; **no** status entry on the activity |
| activity's own planned cost, start date, end date or ongoing changed; activity split | Submitted or Approved | Created, after confirmation (`ActivityUpdate.restartApproval`, `ActivitySplitRequest.restartApproval`) | open round ends "Approval restarted" (008) | unchanged (006, 016) |

Unassigned planned cost (domain model §4.3): computed as before; when negative it is shown in red with its
warning and blocks submit and re-submit (008 readiness, D-016); it never changes a status by itself.

Wire fields (research R1):

| Field | Where | Since 034 |
|---|---|---|
| `restartApproval` | `LineItemCreate`, `LineItemUpdate`, `LineItemDuplicate` bodies; `?restartApproval=` of the line-item delete | deprecated; accepted and ignored |
| `restartsApproval` | `LineItemActivity` (in line-item answers) | deprecated; always `false` |

## 2. Shell attribute option lists (seeded Settings data, D-325)

Attributes keep their ID, label, level, type, marking, section, condition and position; only the options,
the parent link of GL code and the help texts change.

| Attribute | Level · type · marking | Parent | Options (order = position) |
|---|---|---|---|
| Zero-based cost hierarchy | activity · multiple choice · optional · default section, every business unit | — | 10: LUBES - Growth drivers · LUBES - Legal or regulatory requirements · LUBES - Long term contractual obligations · LUBES - Other · LUBES - Protecting the core · LUBES - Strategic Priorities · LUBES - Tactical initiatives · RETAIL - Long-Term Contracts · RETAIL - Strategic Priorities · RETAIL - Tactical Investments |
| Spend type | line item · single choice · needed to submit · Finance coding | — | 25 (spec FR-007) |
| GL code | line item · single choice · needed to submit · Finance coding | **none** (was Spend type) | 13 (spec scenario 2.1) |
| Spend category | line item · single choice · optional · Finance coding | — | 19 (spec FR-008) |
| Contracted spend | line item · single choice · optional · Finance coding | — | 5: Not Applicable · No · Yes - Both · Yes - LT · Yes - ST |

Help texts:

| Attribute | Help text |
|---|---|
| Zero-based cost hierarchy | "The zero-based budgeting categories the activity's spend falls into (LUBES or RETAIL)." |
| Spend type | "What kind of spend the line item is, from Shell's list of ATL, BTL and other spend types." |
| GL code | "The general-ledger account the spend posts to: one of Shell's 13 marketing GL codes." |
| Spend category | "Who or what the spend is for, from Shell's list of spend categories." |
| Contracted spend | "Whether the spend is under contract: long term (LT), short term (ST), both, no, or not applicable." |

Seed counts: 35 attributes (SA-1001 … SA-1035, unchanged) and 162 options (SO-1001 … SO-1162; was 110).
Demo values per research R7.

## 3. Plan year (D-326)

| Field | Rule |
|---|---|
| `Plan.year` on create | integer, current year … current year + 10 (clock injected); else problem `year`: "Choose a year from <first> to <last>." |
| `Plan.year` on update | not sent, never re-checked: a stored year of the past stays |
| `PlanDefaults.years` (form defaults) | `[current … current + 10]`, ascending (11 years); `year` = current |
| `NextYearOption` (copy options) | `available` when source year + 1 ≤ current + 10; else `reason` "Plans can only be created up to <current + 10>." |
| Same-year copy | allowed for any source year (unchanged) |
| Year field on screen | 11 options, so the searchable list of design system §5 (research R10) |

## 4. Web app view state (feature folder `line-items`)

| Element | Before | After |
|---|---|---|
| `useLineItemSave` | `pending` attempt + `confirmRestart` / `cancelRestart`; re-asks after a `restartApproval` refusal | sends at once; no pending state |
| `useLineItemActions` | `copying` state and restart dialog; `deleting.restart` | duplicate sends at once; `deleting` holds the line item and activity |
| `DeleteLineItemDialog` props | `restartsApproval` | removed |
| Requests | `restartApproval` in bodies, `?restartApproval=true` on delete | not sent (duplicate body `{}`) |
| `RestartApprovalDialog` | line-item restart dialog | deleted (the activity's own `ReapprovalDialog` stays) |
