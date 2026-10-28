# Implementation Plan: The Approval Engine Approves Any Record Type

**Branch**: `feature/031-approval-any-record` | **Date**: 2026-10-27 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/031-approval-any-record/spec.md` (amends 008 and 017)

**Design artifacts**: [research.md](research.md) · [data-model.md](data-model.md) ·
[contracts/openapi.yaml](contracts/openapi.yaml) (fragment) · [contracts/routes.md](contracts/routes.md) ·
[contracts/go-api.md](contracts/go-api.md) (the Go interface feature 032 consumes) · [quickstart.md](quickstart.md)

## Summary

The approval engine (`backend/internal/approval`, features 008 and 017) approves only activities: a round
has an `ActivityID`, the service reads every record through the `Activities` port, and the access rule
"Nobody approves an activity they own or submitted" (D-302) is applied everywhere. This feature makes the
round's record a **subject** — a record type (`SubjectKind`: `activity`, `purchaseRequest`,
`modifiedPurchaseRequest`) plus the record's id — and runs every engine step (submit, decide, recall,
escalate, inbox, monitoring, notifications) on an internal `subject` value built either from an activity
(existing `Activities` port) or from a purchase request (new `PurchaseRequests` port, implemented by feature
032 in `internal/app`). On top:

- **Workflow object types**: `Workflow.ObjectType`, chosen at create, changeable only while unused; the
  activity form and the activity submission see activity workflows only; Settings shows and filters it.
- **Two approver sources**: `owner` (the record's owner) and `recordField` (the person a record field names;
  `moaApprover` for purchase requests), resolved at submission with the 017 exclusions, backup approver and
  on-behalf behaviour.
- **Self-approval by record type**: D-302 applies to activities only; a purchase-request decision by its
  owner or submitter is allowed and stored as self-approval.
- **Re-submit from Submitted** (D-323): the open round ends with the new outcome `superseded` and a new
  round starts in the same transaction; the activity page offers it with the consequence stated.
- **Subject API** for 032: `SubmitSubject`, `RecallSubject`, `DecideSubject`, `SectionOf`, wired with
  `UsePurchaseRequests`; until then PR operations answer 404.
- **Inbox, Monitoring, notifications** carry the subject kind and label; the Approvals page gets a Record
  type column and filter.

Storage: migration `00018_approval_subjects.sql` renames `activity_id` to `subject_id` (rounds, actions,
notifications), adds `subject_kind` (every existing row `activity`), the workflow `object_type`, the level
`record_field`, the action `self_approval`, and widens the outcome, action and approver-type checks.

## Technical Context

**Language/Version**: Go (pinned in `backend/go.mod`); TypeScript 5.9 strict + React 19

**Primary Dependencies**: none new. Backend: Gin (handlers), pgx/v5 + sqlc + goose. Frontend: TanStack
Query, React Hook Form + Zod, Radix, CSS Modules.

**Storage**: PostgreSQL 18; one forward-only migration `00018_approval_subjects.sql`; sqlc queries
`approval_rounds.sql`, `approval_actions.sql`, `approval_workflows.sql`, `notifications.sql` updated.

**Testing**: Go `testing` + `go-cmp`; service tests on `fake.DB` with `fakeActivities` and a new
`fakePurchaseRequests`; handler and contract tests through `apitest`; the existing scope-isolation test of
`RoundsInScope`/`ActionsInScope` keeps proving the scope filter on the renamed column. Vitest + Testing
Library + MSW + vitest-axe; Playwright e2e `frontend/e2e/approval-any-record.spec.ts` (US1, US2).

**Target Platform / Project Type**: local web application (Go API + React SPA), 127.0.0.1 only.

**Performance Goals**: no new N+1: the board reads purchase requests in one `ListInScope` call, as it reads
activities (feature 028).

**Constraints**: every existing activity route, contract and test keeps working (SC-001); comment budget
5% per touched file; files ≤ 300 non-blank lines; function metrics of constitution V.

**Scale/Scope**: no new HTTP route. Changed responses: approval workflows (object type, sources, record
field), approval section rounds (kind, superseded), history entries (superseded, self-approval), inbox rows
(subject), notifications (subject), monitoring (kind), rule options (object types, record fields),
submission (replaced round).

## Constitution Check

*GATE: passed before Phase 0; re-checked after Phase 1 design — still passing.*

| Principle | How this plan complies | Result |
|---|---|---|
| I. Spec first | Every change traces to 031 FR-001 – FR-030 (table *Business rules and their tests*); 008 and 017 amendments named in the spec | Pass |
| II. Shell business, plain language | Record types "Activity", "Purchase request", "Modified purchase request"; "MoA approver"; "Replaced by a new submission"; no Hive9 terms | Pass |
| III. Documented design | plan, research, data-model, contracts (OpenAPI fragment, route table, Go API for 032), quickstart; STRIDE and risk model below | Pass |
| IV. Two parts, server as source of truth | All rules in `approval` service; the PR port is a consumer-defined interface implemented in `internal/app` by 032; the web app shows the server's `objectTypeFixed`, kinds and labels and recomputes nothing | Pass |
| V. Simplicity | No new package, no new route; one internal `subject` value replaces `ActivityFacts` inside the engine; files split by responsibility (`subject.go`, `subject_api.go`, `ports_purchase.go`, `resubmit.go`) | Pass |
| VI. Meaningful tests | Unit tests per rule (sources, self-approval by kind, object type lock, superseded, PR events), contract tests for every changed response, existing scope-isolation test kept on the renamed column, component tests per state, e2e for US1 and US2 | Pass |
| VII. Security | Scope only from the actor (`authz.ScopeOf`), PR records read through the port's `ListInScope`; out-of-scope PR = 404; decisions keep 014/017 authority; self-approval only where Shell's process allows it (D-324) | Pass |
| VIII. Clean construction | Comment budget per touched file; requirement IDs only in test names | Pass |
| IX, X. Experience and palette | Existing components and tokens; Re-submit states the consequence before confirming; one primary action per dialog | Pass |
| B5 relational schema | New columns with `CHECK` constraints; open-round exclusion and numbering per `subject_id` | Pass |

## Design elements (one-to-one onto modules)

| Element | Module | Interface (detail in data-model.md and contracts/go-api.md) |
|---|---|---|
| Subject kinds | `internal/approval/kind.go` | `SubjectKind`, `KindActivity`, `KindPurchaseRequest`, `KindModifiedPurchaseRequest`; `Label()`, `IsValid()` |
| Subject (engine view of a record) | `internal/approval/subject.go` | unexported `subject`; `activitySubject`, `purchaseSubject`; `s.subjectOf`, `s.moved` (status hand-off) |
| PR port | `internal/approval/ports_purchase.go` | `PurchaseRequests`, `PurchaseRequestFacts`, `UsePurchaseRequests` |
| Subject API | `internal/approval/subject_api.go` | `SubmitSubject`, `RecallSubject`, `DecideSubject`, `SectionOf`, `Section`, `SubjectDecision` |
| Re-submit from Submitted | `internal/approval/resubmit.go` | `supersede` inside the submission's Write |
| Approver sources | `internal/approval/resolve.go`, `sources.go` | `SourceOwner`, `SourceRecordField`, `RecordFieldMoAApprover` |
| Workflow object type | `internal/approval/workflows*.go` | `Workflow.ObjectType`, `WorkflowInput.ObjectType`, `WorkflowUpdate.ObjectType` |
| Storage | migration 00018, queries, `store_postgres*.go`, both fakes | `Store` unchanged in shape; columns renamed |
| Activity workflow field | `internal/app/activity_workflows.go` | lists and finds activity workflows only |
| Home waiting list | `internal/app/home_sources.go` | keeps activity rows only (Home links to activities) |
| Seed | `backend/seed/approvals.go`, `history_approvals.go` | stored rounds, entries and notifications name their subject |
| Settings › Approval workflows | `frontend/src/features/approval-workflows` | object type field, Approves column and filter, new sources, record field, kind-aware conditions |
| Approvals page, section, notifications | `frontend/src/features/approvals`, `features/notifications` | Record type column and filter, re-submit consequence, superseded and self-approval texts, subject links |

## Business rules and their tests

| Rule (FR) | Test |
|---|---|
| Existing rows read as activity rounds (FR-001) | `store_postgres_test.go` scope test on renamed columns; `apitest` contract tests of every approvals route on the demo data |
| One numbering and one open round per record (FR-002) | `subject_api_test.go` `TestModificationRoundNumbering/031 FR-002` |
| Object type default, lock once used (FR-004, FR-005) | `workflows_kind_test.go` `TestCreateWorkflowObjectType/031 FR-004`, `TestUpdateWorkflowObjectType/031 FR-005` |
| Activity field sees activity workflows only (FR-007) | `internal/app/activity_workflows_test.go` `TestActivityWorkflowsOnlyActivity/031 FR-007` |
| Wrong-kind workflow is a submit problem (FR-008) | `rounds_kind_test.go` `TestSubmitWorkflowOfAnotherKind/031 FR-008` |
| Usage lists records of every type (FR-009) | `workflows_kind_test.go` `TestWorkflowUsageOfPurchaseRequests/031 FR-009` |
| Owner and record-field sources, empty field problem, activity refuses record field (FR-010 – FR-013) | `sources_test.go` table `TestResolveOwnerAndRecordField/031 FR-010…FR-013`; `workflows_kind_test.go` `TestRecordFieldOnActivityWorkflow/031 FR-011` |
| Conditions per object type (FR-014) | `workflows_kind_test.go` `TestConditionsPerObjectType/031 FR-014`; `rules_test.go` amount wording |
| Self-approval by kind (FR-015, FR-016) | `subject_api_test.go` `TestPurchaseRequestSelfApproval/031 FR-015`, `TestActivitySelfApprovalStillBarred/031 FR-015` |
| Re-submit from Submitted (FR-017 – FR-020) | `resubmit_test.go` `TestResubmitSubmitted/031 FR-017`, `…Problems`, `…StaleDecision/031 FR-019`, `…AutoApproved/031 FR-020`; handler contract test |
| PR gates with workflow problems (FR-021) | `subject_api_test.go` `TestSubmitSubjectGates/031 FR-021` |
| PR workflows per kind (FR-022) | `subject_api_test.go` `TestModificationUsesModificationWorkflow/031 FR-022` |
| RoundChanged events (FR-023) | `subject_api_test.go` `TestRoundChangedEvents/031 FR-023` (table: start, level approved, approved, declined, recalled, superseded) |
| PR actions availability and recall rights (FR-024) | `subject_api_test.go` `TestSectionActions/031 FR-024`, `TestRecallSubjectRights/031 FR-024` |
| Not wired → not found (FR-025) | `subject_api_test.go` `TestPurchaseRequestsNotWired/031 FR-025` |
| Inbox, monitoring, notifications carry kind (FR-026 – FR-028) | `inbox_kind_test.go` `TestInboxCarriesKind/031 FR-026`, `TestMonitoringKind/031 FR-027`, `TestNotificationSubject/031 FR-028`; frontend `ApprovalsTable.test.tsx`, `approval-filters.test.ts`, `NotificationsMenu` test |
| Messages use the record's word (FR-029) | `subject_api_test.go` `TestRefusalNamesPurchaseRequest/031 FR-029` |
| Every event logged (FR-030) | assertions on `f.log` in the subject API tests |

## Threat model (STRIDE) — new interfaces

| Interface | Threat | Response |
|---|---|---|
| `PurchaseRequests` port (in-process, 032 → engine) | **Spoofing/Elevation**: a caller passes an actor or kind it should not | The engine never trusts a status or approver list from the caller: approvers are resolved from facts read inside the same transaction; authority (014/017 check order) runs in the engine for every decision; the actor comes from the request context in 032's handler |
| | **Information disclosure**: inbox lists PRs outside the viewer's scope | The board reads PRs only through `ListInScope(authz.ScopeOf(ctx), ids)`; 032's implementation filters in SQL (scope-isolation test is 032's); a PR not returned is not listed |
| | **Tampering**: a decision on a stale or replaced round | Round id and level are checked (`roundToDecide`); superseded rounds refuse decisions with 409 |
| | **Repudiation**: self-approval not visible | Every decision is stored with capacity and `selfApproval`, and logged in the activity log |
| Workflow `objectType` input (HTTP) | **Tampering**: invalid kind or changing a used workflow | Allowlist of three kinds; change refused once used; validated with all other problems at once |
| Level `recordField` input (HTTP) | **Tampering**: unknown field name | Allowlist per object type (`moaApprover` for PR kinds only) |
| Notifications and inbox rows | **Information disclosure**: a notification about a PR the person can no longer see | Only the recipient reads their notifications (unchanged); `exists` reports only whether the record is stored and visible |

No cryptography, browser storage or new origin is involved.

## Risk model

| Risk | Response | Owner |
|---|---|---|
| Renaming `activity_id` breaks an existing query or the scope function | sqlc vet and generate; the scope function is recreated in the same migration; the scope-isolation test and the e2e journeys run the migration | Implementer (031) |
| 032 needs a status signal for In-Review | `RoundChanged` reports `OutcomeOpen` again after a decision that approves a level while later levels remain (research R3, agreed with the 032 planner); documented in contracts/go-api.md | Coordinator, 2026-10-27 |
| Two features seed PR workflows | 031 seeds none; 032 seeds "Purchase request approval" and "Purchase request modification approval" (research R12) | Orchestrator |
| Home's Waiting for me differs from the Approvals badge once PRs exist | Home keeps activity rows (it links to activities); the badge counts every kind. Recorded as a follow-up for 032 | Product owner to confirm |
| Activity texts change by accident | Activity messages and notification texts kept byte for byte; existing tests unchanged except where the spec changes behaviour (re-submit availability) | Implementer |

## JavaScript budget

No new dependency and no new route chunk; the approvals and approval-workflows chunks grow by a few
hundred bytes of labels and one filter. The initial-route budget in the repository is unchanged.

## Shared files touched

`backend/api/openapi.yaml`, `backend/internal/app/{activity_workflows.go,home_sources.go,areas.go}`,
`backend/internal/stores/*` (no change expected), `backend/seed/{approvals.go,history_approvals.go}`,
`docs/product/domain-model.md` (§2.3, §6 approval workflow row, §7 notification), frontend generated types.
No route-table change (`internal/access/routes*.go`): no new operation.

## Project Structure

### Documentation (this feature)

```text
specs/031-approval-any-record/
├── spec.md · plan.md · research.md · data-model.md · quickstart.md · tasks.md
├── checklists/requirements.md
└── contracts/openapi.yaml · routes.md · go-api.md
```

### Source Code (repository root)

```text
backend/internal/platform/database/migrations/00018_approval_subjects.sql
backend/internal/platform/database/queries/{approval_rounds,approval_actions,approval_workflows,notifications}.sql
backend/internal/approval/
├── kind.go · subject.go · subject_api.go · ports_purchase.go · resubmit.go · sources.go   (new)
├── model*.go · rounds*.go · resolve.go · authority.go · access_control.go · view*.go         (changed)
├── inbox*.go · monitoring.go · escalations.go · notifications.go · hooks.go · messages.go     (changed)
├── workflows*.go · rules.go · options.go · store*.go · approvaltest/store.go                  (changed)
└── *_test.go (new: subject_api_test, resubmit_test, sources_test, workflows_kind_test, inbox_kind_test)
backend/internal/app/activity_workflows.go · home_sources.go
backend/seed/approvals.go · history_approvals.go
frontend/src/features/approval-workflows/  (form, list, levels editor, conditions, details)
frontend/src/features/approvals/           (table, filters, section, submit dialog, chain, history)
frontend/src/features/notifications/       (subject link and label)
frontend/e2e/approval-any-record.spec.ts
```

**Structure Decision**: the existing web application layout; no new package or feature folder — the engine
stays one business area (`approval`), and the screens stay in their feature folders.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| `SubmitSubject` and `DecideSubject` take 5 parameters (needs-justification band) | The signatures are the interface agreed with feature 032 (integration brief §8.2) | A request struct would break the agreed interface |
