# Research: Planning Rules Aligned with Shell's Process

Feature 034 · 2026-10-27 · Spec: [spec.md](spec.md) · Plan: [plan.md](plan.md)

## R1 — What happens to the `restartApproval` flag and `restartsApproval` field

**Decision**: Keep them in the contract, marked `deprecated: true` and documented as ignored: the request
property `restartApproval` of `LineItemCreate`, `LineItemUpdate` and `LineItemDuplicate`, the query
parameter `restartApproval` of `DELETE /api/v1/line-items/{lineItemId}` (its reusable parameter
`RestartApproval` is used by that operation only), and the answer property `LineItemActivity.restartsApproval`,
which is now always `false` ("a line-item change never sends this activity back for approval"). The server
accepts the old flag and ignores it; the delete handler no longer reads the query parameter, so an unreadable
value is no longer refused. The deprecated request properties lose their `default: false`, so the generated
web types make them optional and the web app can leave them out (a default never changes what a client may
send). The web app stops sending and reading them. The activity's own
`ActivityUpdate.restartApproval` and the split's `restartApproval` keep their meaning.

**Rationale**: B6 — breaking changes need a new API version, additive changes must stay backward compatible.
The API decodes bodies with unknown fields refused, so removing the property would make an older web page
(still open in a browser when the new API starts) fail every line-item save with 400. Keeping the field
costs three struct fields and a few lines of contract; an older page that still reads `restartsApproval`
gets `false` and therefore never shows the old dialog — exactly the new behaviour. The fields can go in a
future `/api/v2`.

**Alternatives considered**: (a) remove them — simplest code, but a breaking change of `/api/v1` (B6) and
needs an ADR; (b) keep the old answer `restartsApproval` = "activity is Submitted or Approved" — misleading,
an older page would keep asking for a confirmation that changes nothing.

## R2 — Remove `activity.Service.RestartApproval` and its port

**Decision**: Remove `activity.Service.RestartApproval`, `lineitem.Activities.RestartApproval`, the adapter
`app.lineItemActivities.RestartApproval`, `lineitem.ActivityFacts.restartsApproval`, `confirmRestart`,
`restart`, `restartMessage` and the reason texts (`addedReason`, `deletedReason`, `changedReason`,
`restoredReason`, `restartedNote`). Keep `logRestart`, `reapprovalFields`, `inApproval`, the
`ApprovalRestarted` hook and its `internal/app` adapter: the activity's own update and split still use them.

**Rationale**: constitution VIII — every declaration is used. After D-322 nothing calls them. The approval
engine (feature 008/031) keeps receiving restarts through `ApprovalRestarted` from the activity's own
changes, so its hook usage is unchanged. The engine's handling of a change given as a ready-made sentence
(a label without old and new values, `approval/service.go`, whose comment names 007's line-item changes as
the example) is left as it is: feature 031 owns the engine, and the comment is reported to the
orchestrator rather than edited here.

**Alternatives considered**: keep `RestartApproval` "in case" — dead code (YAGNI).

## R3 — Restoring a line item

**Decision**: Drop the line-item kind's `AfterRestore` hook (it only restarted approval, D-147). Undo and
Restore keep their name rules (`PrepareRestore`), unchanged.

**Rationale**: FR-002; `history.Kind.AfterRestore` is optional (nil is skipped).

## R4 — Negative unassigned balance (D-016)

**Decision**: No code change. The balance and its red warning come from `lineitem.balance` and the
`UnassignedBalance` answer; the submit refusal is 008's readiness check ("Ready to submit"), which reads the
balance at submission and re-submission. A line-item change on an Approved activity that makes the balance
negative is saved and leaves the status alone. One new service test proves the "saved, stays Approved,
balance negative" combination; the refusal is already proven by 008's readiness tests.

## R5 — Edits in the approval package (feature 031 works there in parallel)

**Decision**: Only the recall explanation in `internal/approval/messages.go` changes ("… Changing its
planned cost or dates sends it back for approval."), with the three tests that quote it
(`view_test.go`, `rounds_recall_test.go`, `handler_changes_test.go`); in `handler_changes_test.go` the test
`TestLineItemCurrencyChangeRestartsApproval_FR022` becomes `TestLineItemCurrencyChangeKeepsApproval_034_FR001`
(D-322). Nothing in the engine, its rounds, hooks or store changes. The web app's copy of the sentence
(`frontend/src/features/approvals/messages.ts`) changes with it.

## R6 — GL code without a seeded parent (spend type → GL code)

**Decision**: Seed GL code as an independent single choice (no parent attribute, no option links). Keep the
dependency mechanism of feature 002 (any attribute may depend on another) and the line-item form's
"will be cleared — it doesn't fit" behaviour for attributes that do depend (Cluster → Region shows it in the
demo). Domain model R11 becomes: "A line item's GL code options depend on its spend type only when an
administrator links them in Settings; Shell's seed has no such links yet (D-325)."

**Rationale**: FA §E3.0.2 lists the 13 GL codes and the spend types as independent filter groups and shows
no mapping; FA §B7 saw neither on the live form; the GSAP workshop says only that "one spend type can map
to several GL codes", without the mapping. The previous seed links (e.g. "BTL – Event → 7250040") were
invented (and 7250040 is not one of Shell's codes). An invented mapping would wrongly hide valid codes.

**Alternatives considered**: (a) invent a plausible mapping from the names (ATL - Digital → 7250080, …) —
contradicts D-325's "no invented options"; (b) link every GL code to every spend type — same offer as no
parent, but heavier data and a misleading dependency in Settings.

## R7 — Shell's lists and the demo records

**Decision**: The option lists, verbatim from FA §E3.0.2 (hyphen "-" as observed live):

| Attribute (level) | Options | Count |
|---|---|---|
| GL code (line item) | 6140900 - PMTDR Credit · 7250050 - Advertising - Media Planning/Auditing · 7250060 - Advertising - Corp/Agency/Production · 7250070 - Advertising - Media Buying · 7250080 - Advertising - Digital · 7250100 - Public Relations · 7250150 - Sponsorship programmes · 7250190 - Loyalty Programmes - Materials · 7250200 - Loyalty Programmes - Services · 7250250 - Market Research · 7250300 - Promotions & POP - Incentives/Reimbursement · 7250310 - Promotions & POP - Agency/Print/Logistics · 7250350 - Brand Development & Coordination | 13 |
| Spend type (line item) | the 25 named in E3.0.2, in its order (spec FR-007) | 25 of 57 |
| Spend category (line item) | Not Applicable … Tradeshow (spec FR-008) | 19 |
| Contracted spend (line item) | Not Applicable · No · Yes - Both · Yes - LT · Yes - ST | 5 |
| Zero-based cost hierarchy (activity) | LUBES - Growth drivers … RETAIL - Tactical Investments (spec FR-010) | 10 |

The FA's group header says "GL Code (15)" while it lists 13 codes plus "(Not Completed)"; the workshop and
§B7 both say 13 — so 13. The field catalogue's "13 marketing GL codes" now matches the seed (it named 13
while the seed had 5).

Demo values (same completeness, same records, only the labels change):

| Old seeded value | New Shell value |
|---|---|
| Spend type "Digital – Platform development" | "ATL - Digital" |
| Spend type "BTL – Event" | "BTL - Event" |
| Spend type "Agency – Creative" | "BTL - Agency Fees" |
| GL code "7250080 – Advertising digital" | "7250080 - Advertising - Digital" |
| GL code "7250040 – Events and sponsorship" | "7250150 - Sponsorship programmes" |
| GL code "7250060 – Agency fees" | "7250060 - Advertising - Corp/Agency/Production" |
| Zero-based cost hierarchy "Production" (Launch film) | "LUBES - Growth drivers" |

"Station signage" (L-1007) keeps no GL code, so its completeness story is unchanged. Labels, markings,
sections, conditions and positions of the five attributes are unchanged; their help texts describe Shell's
lists (GL code no longer says "choose a spend type first").

**Consequence for IDs**: options are numbered in seed order, so every option after the zero-based list
(Brands … Contracted spend) gets a new SO- number: 162 options instead of 110, the next option ID is
SO-1163. No test or document depends on a seeded option ID beyond the seed's own counter test (searched:
the other SO- IDs in tests belong to fakes or fixtures).

## R8 — No migration

**Decision**: No migration (number 00019 stays unused). The lists are Settings data written by the demo
seed (`go run ./cmd/seed -reset`), not schema; constitution II keeps business configuration as data.
A database seeded before 034 keeps its old options until it is reseeded; its records stay valid because
nothing references the new options.

## R9 — Tests that asserted the old rule

Each is changed to the new rule and cites D-322 / 034 in its name or comment, or removed with the code it
proved:

| Test | Was | Now |
|---|---|---|
| `lineitem` create test "on a Submitted or Approved activity only with confirmation" | refused without the flag, Created after | saved without confirmation, status unchanged (034 FR-001) |
| `lineitem` duplicate test (same) | as above | as above |
| `lineitem` update test (cost/currency needs confirmation) | as above | cost and currency change keep the status (034 FR-001) |
| `lineitem` delete test (needs `restartApproval`) | as above | deleted, status unchanged (034 FR-001) |
| `lineitem` kinds test "restore restarts approval" | restore sent the activity to Created with a note | restore keeps the status, no note (034 FR-002) |
| `lineitem` handler tests (create, duplicate, delete with the flag, `?restartApproval=maybe` refused) | 400 without the flag | 201/200 without it; the old flag still accepted |
| `activity` `TestRestartApproval_SendsBackAndLogs_FR027`, `TestLineItemReapprovalChange_CallsApprovalHook_FR022` | proved `RestartApproval` | removed with the method (R2); the hook path stays proven by `TestReapprovalChange_CallsApprovalHook_FR022` |
| `approval` `TestLineItemCurrencyChangeRestartsApproval_FR022` | Approved → Created | `TestLineItemCurrencyChangeKeepsApproval_034_FR001`: stays Approved, no new history entry |
| web `LineItemCreatePage.save`, `LineItemEditPage`, `LineItemPage` restart-dialog tests | dialog, then `restartApproval: true` | no dialog, request without the flag (034 FR-001) |
| web `line-item-requests` / `line-item-text` tests of the removed helpers | helpers | removed with the helpers |

## R10 — Plan years

**Decision**: `firstYearBeforeCurrent = 0`, `lastYearAfterCurrent = 10` in `internal/plan/validate.go`; the
same `years()` drives the form defaults (`years` list), the create check and the next-year copy limit.
Update never re-checks the stored year (already true: `check` keeps `stored.Year`), so a plan of a past
year can be edited; this is now proven by a test with the clock in the following year. A copy into the
source's own year stays allowed for any year (spec clarification 4). Global budgets (014, `internal/budget`)
and finance budget items (002) keep current − 1 … + 5: they are separate decisions not covered by D-326.

**Consequence on screen**: the web app shows the years the API offers, so it needs no code change; but 11
years is above the design system's limit for a plain select (§5: up to 10 options are a select, more are
the searchable `Combobox`), so the new-plan Year field becomes the searchable list, with the chosen year as
a chip. This follows the design system rather than making an exception for one field; the component tests
of the plan form choose the year through the list. To be confirmed with the product owner (a design-system
exception for year fields would bring the select back).

**Decision for the log (proposed)**: none new — D-322, D-325 and D-326 cover this feature. Proposed
wording for the open points, for the orchestrator to confirm with the product owner: the spec's five
clarifications (25 spend types; GL code independent; Shell's exact text; same-year copies of past plans
allowed; zero-based cost hierarchy on every activity) and R1 (deprecated-but-accepted flag).
