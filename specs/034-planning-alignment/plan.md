# Implementation Plan: Planning Rules Aligned with Shell's Process

**Branch**: `feature/034-planning-alignment` (not created: no branches or commits in this delivery, the
product owner's choice) | **Date**: 2026-10-27 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/034-planning-alignment/spec.md` (amends 002, 004, 007, 008)

**Design artifacts**: [research.md](research.md) (R1–R10) · [data-model.md](data-model.md) ·
[contracts/openapi.yaml](contracts/openapi.yaml) (fragment: changed descriptions and deprecated fields, no
new operation) · [contracts/routes.md](contracts/routes.md) · [quickstart.md](quickstart.md) ·
[tasks.md](tasks.md).

## Summary

Three corrections to built features, no new record type, endpoint or migration:

1. **Line-item changes keep the activity's approval (D-322).** Package `internal/lineitem` stops calling
   feature 006's `RestartApproval` from create, duplicate, update (cost or currency), delete and the
   recycle-bin `AfterRestore` hook, and stops refusing those saves without confirmation. The now unused
   `activity.Service.RestartApproval`, its port method in `lineitem.Activities` and its adapter in
   `internal/app` are removed; the activity's own re-approval (planned cost, dates, ongoing, split) and the
   approval hook `ApprovalRestarted` stay as they are. The request flag `restartApproval` of the three
   line-item operations, the delete query parameter and the answer field `LineItemActivity.restartsApproval`
   stay in the contract, marked deprecated and ignored (always false), so the change is backward compatible
   (B6; research R1). The web app drops the line-item restart dialog, the restart line of the delete dialog
   and the retry-after-refusal paths. The approval package changes one sentence (the recall explanation for
   Approved activities) and its three tests.
2. **Shell's configuration lists (D-325).** The seed (`backend/seed/settings.go`) gets FA E3.0.2's values
   for GL code (13), Spend type (25 named), Spend category (19), Contracted spend (5) and Zero-based cost
   hierarchy (10); GL code loses its seeded parent Spend type (research R6); demo line items and the one
   demo activity with a zero-based value get equivalent Shell values (R7). Settings behaviour is unchanged.
3. **Plan year current … +10 (D-326).** `internal/plan` changes its two year constants: a new plan's years
   are the current year to + 10 (no past year); the stored year is never re-checked, so plans of a past
   year keep working; a next-year copy is allowed up to current + 10. The web app shows the years the API
   offers, so it needs no change beyond test fixtures.

Product documents follow: domain model §2.5 and R11, field catalogue §1 #2, §4.2 #21, §5 #8–#11,
demo-users-and-data.md; amendment notes in specs 004, 007 and 008.

## Technical Context

**Language/Version**: Go (pinned in `go.mod`); TypeScript 5.9 strict + React 19

**Primary Dependencies**: none new

**Storage**: PostgreSQL; no schema change, no migration (00019 not needed: the lists are seeded Settings
data, research R8)

**Testing**: Go service and handler tests on fakes and `apitest.DemoDB` (contract-validated); seed tests on
the in-memory demo data; Vitest + Testing Library + MSW + vitest-axe; one Playwright journey file
`frontend/e2e/planning-alignment.spec.ts` for the two P1 stories, on PostgreSQL (`mapp_e2e_034`, ports
18134/14134)

**Target Platform**: developers' machines (127.0.0.1); GitHub Actions

**Project Type**: Web application (Go API + React web app)

**Performance Goals**: unchanged; line-item writes get cheaper (no activity read-modify-write, no approval
hook)

**Constraints**: approval package (`internal/approval`) touched only for the recall sentence and its tests
(feature 031 changes the engine in parallel); every touched file within 300 non-blank lines and its comment
budget (5 %); functions within the quality metrics

**Scale/Scope**: 4 Go packages (`lineitem`, `activity`, `app`, `plan`) + `approval` (1 message) + `seed`;
1 web feature folder (`line-items`) + `approvals/messages.ts`; 5 seeded option lists (52 more options)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle / standard | Check | Result |
|---|---|---|
| I Spec-first | 034 spec FR-001 – FR-017 with acceptance scenarios; amendments of 002, 004, 007, 008 listed in the spec's amendment index and noted in those specs | Pass |
| II Shell business, plain language | Shell's own list values (D-325); vocabulary unchanged; Settings stays the place to change lists | Pass |
| III Documented design | plan, research R1–R10, data-model, contracts fragment and route table, quickstart | Pass |
| IV Server source of truth | The status rule lives in the API; the web app only stops asking; the year list comes from the API | Pass |
| V Simplicity | Code is removed, not added (restart paths, dialog, port method); no new abstraction | Pass |
| VI Meaningful tests | Tests that asserted the old rule are changed to the new one and cite D-322 (research R9 lists each); one test per acceptance scenario; e2e for both P1 stories; no new scoped store read, so no new scope-isolation test | Pass |
| VII Security, scope isolation | No new operation or permission; route rules unchanged (contracts/routes.md); scope checks of the touched operations unchanged | Pass |
| VIII Clean construction | Every touched file brought within its comment budget, except five approval files (Complexity Tracking); requirement IDs only in test names | Pass with one deviation |
| IX Experience | One fewer dialog; the delete dialog no longer warns of a consequence that no longer happens; the activity's own re-approval warning stays | Pass |
| X Palette | No visual change | Pass |
| B5 Data | No schema change; seed data only | Pass |
| B6 Contract | Backward compatible: removed behaviour, fields kept as deprecated and ignored (research R1) | Pass |
| F8 JS budget | No JS budget is recorded in the repository yet; this feature removes one component and two hooks' branches from the line-items chunk (smaller), adds no dependency | Pass |
| Governance → Alignment | `lineitem`, `activity`, `plan`, `approval`, `seed` keep their stack; only lines this feature changes are linted (`golangci-lint run --new`) | Pass |

**Post-design re-check (after Phase 1)**: unchanged — Pass.

### Threat model (STRIDE)

No new trust boundary, data flow or externally reachable interface. The changed behaviour of existing
interfaces:

| Threat | Where | Response |
|---|---|---|
| Tampering / elevation: changing an approved activity's money without re-approval | line-item writes | Accepted by the business (D-322): the approved figure is the activity's planned cost, which still restarts approval; line items are allocations of it. The negative unassigned balance stays visible and blocks the next submission (D-016); every line-item change is logged with the actor (011) |
| Spoofing a confirmation | `restartApproval` flag | Ignored by the server; it can no longer change anything |
| Information disclosure | none | No field added to any answer |
| Denial of service | none | Fewer writes per line-item change |

### Risk model

| Risk | Response | Owner |
|---|---|---|
| Approvers rely on line-item amounts that change after approval | Approval covers the activity's planned cost (009 FR-050, D-322); the line items and the unassigned balance are on the activity page and in the approval preview; product owner confirms D-322 | Product owner |
| A demo database seeded before 034 keeps the invented lists | Reseed (`go run ./cmd/seed`); the user's demo DB is reseeded at integration | Team |
| Spend types beyond the 25 named are missing | Recorded as an assumption; added in Settings when Shell supplies them | Product owner |
| Merge conflict with features 030/031 in `lineitem` and `approval` | Edits kept local: lineitem write paths and one approval message; files touched are listed below | Orchestrator |

## Design elements and touch points

| Element (module) | Change | FR |
|---|---|---|
| `backend/internal/lineitem` (service) | `Create`, `Duplicate`, `Update`, `Delete` no longer check or restart approval; `confirmRestart`, `restart`, the reason texts and `ActivityFacts.restartsApproval` go; `Kinds` drops `AfterRestore`; `Activities` port loses `RestartApproval`; `Input`/`Update`/`DuplicateInput.RestartApproval` kept as ignored wire fields; handler stops reading `?restartApproval` | FR-001, FR-002 |
| `backend/internal/activity` | `RestartApproval` removed (no caller); `logRestart`, `reapprovalFields` and the `ApprovalRestarted` hook stay for the activity's own changes; comments of `hooks.go` and `reapproval.go` updated | FR-001, FR-003 |
| `backend/internal/app/lineitems.go` | `lineItemActivities.RestartApproval` adapter removed | FR-001 |
| `backend/internal/approval/messages.go` | Recall explanation: "Approved activities can't be recalled. Changing its planned cost or dates sends it back for approval." | FR-005 |
| `backend/internal/plan` | `firstYearBeforeCurrent` 0, `lastYearAfterCurrent` 10; comments | FR-014 – FR-016 |
| `backend/seed` | Five option lists, GL code without parent, help texts, demo values | FR-006 – FR-013 |
| `backend/api/openapi.yaml` | Descriptions of the line-item operations; `restartApproval` (3 schemas, 1 parameter) and `restartsApproval` deprecated | FR-001 |
| `frontend/src/features/line-items` | `RestartApprovalDialog.tsx` deleted; `useLineItemSave` saves at once; `useLineItemActions` duplicates and deletes without restart; `DeleteLineItemDialog` loses `restartsApproval`; requests send no flag; texts removed | FR-001, FR-002, FR-005 |
| `frontend/src/features/approvals/messages.ts` | Same recall sentence | FR-005 |
| Docs | domain-model §2.5, R11; field-catalogue §1 #2, §4.2 #21, §5 #8–#11; demo-users-and-data.md; notes in specs 004, 007, 008 | FR-017 |

### Interfaces (changed)

- `lineitem.Activities` (port declared by `lineitem`, implemented in `internal/app`):
  `Facts(ctx, activityID) (ActivityFacts, bool, error)` and `FactsOf(ctx, ids) (map[string]ActivityFacts,
  error)` — `RestartApproval` removed.
- `lineitem.Service.Delete(ctx, a, id) (history.BinEntry, error)` — the `restartApproval bool` parameter
  removed; `lineitem.Service.Duplicate(ctx, a, id) (View, error)` — the `DuplicateInput` parameter removed
  (the handler still decodes the body, so a malformed one is refused, and ignores its deprecated flag).
- `activity.Service.RestartApproval` — removed.
- HTTP: unchanged paths, rules and answers; see [contracts/openapi.yaml](contracts/openapi.yaml).

## Business rules and the tests that prove them

| Rule | Test |
|---|---|
| Adding a line item keeps an Approved/Submitted activity's status, no confirmation needed (FR-001) | `lineitem` `TestCreate_InApprovalKeepsStatus_034_FR001`; handler `TestCreateLineItem_Contract` changed |
| Duplicating keeps the status (FR-001) | `lineitem` `TestDuplicate_InApprovalKeepsStatus_034_FR001` (changed from the old rule) |
| Re-costing or changing the currency keeps the status (FR-001) | `lineitem` `TestUpdate_CostOrCurrencyKeepsStatus_034_FR001` (changed); approval `TestLineItemCurrencyChangeKeepsApproval_034_FR001` (changed), `TestLineItemChangeOnSubmittedKeepsRound_034_FR001` |
| Deleting keeps the status, and Undo/Restore do too (FR-001, FR-002) | `lineitem` `TestDelete_InApprovalKeepsStatus_034_FR001`, `TestRestore_IntoApprovedKeepsStatus_034_FR002` (changed) |
| A negative unassigned balance is saved, shown, and the activity stays Approved (FR-004) | `lineitem` `TestCreate_NegativeBalanceKeepsApproval_034_FR004`; 008's readiness tests keep proving the submit refusal |
| The activity's own planned cost still restarts approval (FR-003) | existing `activity` `TestUpdate_ReapprovalNeedsConfirmation_FR027`, approval `TestPlannedCostChangeRestartsApproval_FR022` (unchanged) |
| Recall explanation (FR-005) | approval `rounds_recall_test`, `view_test`, `handler_changes_test` (changed sentence); web `ApprovalSection` tests via `messages.ts` |
| The web app saves, duplicates and deletes without the dialog (FR-001, FR-005) | `LineItemCreatePage.save.test.tsx`, `LineItemEditPage.test.tsx`, `LineItemPage.test.tsx` (changed, `034 FR-001`) |
| The five lists and GL code without parent (FR-006 – FR-012) | seed `TestShellLists_034_FR006_FR010` (with FR-011, FR-012); `settings_test` dependency test changed |
| Demo records valid, same completeness (FR-013) | seed `lineitems_test`, `activities_test`, `seed_test` (changed values, same completeness) |
| Plan years current … + 10 (FR-014) | `plan` year tests named `034 FR-014` (create check, defaults) |
| A past-year plan can still be edited (FR-015) | `plan` `TestUpdate_PastYearPlan_034_FR015` (clock in the next year) |
| Next-year copy up to current + 10 (FR-016) | `plan` copy-options test changed |
| P1 journeys | `frontend/e2e/planning-alignment.spec.ts` |

## Shared files touched

`backend/api/openapi.yaml` (descriptions, `deprecated`, no `default` on the deprecated request fields),
`backend/internal/app/lineitems.go` and `areas.go` (the removed adapter), `backend/seed/` (settings.go,
shell_lists.go (new), names.go, activities.go, seed.go, settings_attributes.go and their tests),
`frontend/src/api/generated/schema.d.ts` (regenerated), `frontend/e2e/planning.ts` and `settings.spec.ts`
(a seeded value and a comment), `docs/product/{domain-model,field-catalogue,demo-users-and-data}.md`
(sections owned by 034), amendment notes in specs 004, 007 and 008.
Approval package files (kept minimal, feature 031 changes the engine in parallel):
`backend/internal/approval/messages.go`, `view_test.go`, `rounds_recall_test.go`, `handler_changes_test.go`,
and the web copy `frontend/src/features/approvals/messages.ts`; their comments are not brought within the
5 % budget here, to avoid a conflict with 031 (reported to the orchestrator). Every other touched source
file is within the comment budget (Principle VIII), which made the comment trimming of the touched
`lineitem`, `plan`, `activity` and `seed` files part of this feature. Not touched: route table,
migrations, `ids.go`, `internal/stores`.

## Project Structure

### Documentation (this feature)

```text
specs/034-planning-alignment/
├── spec.md
├── checklists/requirements.md
├── plan.md              # this file
├── research.md          # R1–R10
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── openapi.yaml     # fragment: changed operations and fields
│   └── routes.md        # route rules of the touched operations (unchanged)
└── tasks.md
```

### Source Code

```text
backend/internal/lineitem/      create, duplicate, update, delete, service, deps, kinds, view, model, handler + tests
backend/internal/activity/      reapproval.go (+ test), hooks.go
backend/internal/app/           lineitems.go
backend/internal/approval/      messages.go + 3 tests
backend/internal/plan/          validate.go, validate_fields.go, copy.go, model.go + tests
backend/seed/                   settings.go, shell_lists.go (new), names.go, activities.go, seed.go + tests
backend/api/openapi.yaml
frontend/src/features/line-items/   useLineItemSave, useLineItemActions, LineItemForm, DeleteLineItemDialog,
                                    line-item-requests, line-item-text, api; RestartApprovalDialog deleted; tests
frontend/src/features/approvals/messages.ts
frontend/src/testing/fixtures/      line-items.ts (restartsApproval false)
frontend/e2e/planning-alignment.spec.ts (new); planning.ts, settings.spec.ts
```

**Structure Decision**: no new package, folder or file other than the e2e journey; changes stay in the
modules that own each rule.

## Complexity Tracking

| Deviation | Why needed | Simpler alternative rejected because |
|---|---|---|
| Comment budget (VIII) not aligned in `internal/approval/{messages.go, view_test.go, rounds_recall_test.go, handler_changes_test.go}` and `frontend/src/features/approvals/messages.ts` | 034 changes one sentence there; feature 031 rewrites the approval engine in the same files in parallel, and the brief asks for minimal approval edits | Trimming the comments now would rewrite files 031 is changing and force a conflict at integration; the orchestrator aligns them once, on the merged files |
