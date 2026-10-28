# Implementation Plan: Purchase Requests on MAPP's Side

**Branch**: none (no Git branches or commits, product owner 2026-10-27) | **Date**: 2026-10-27 |
**Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/032-purchase-requests/spec.md`; binding interfaces of the shared
brief §8.1 (SWBS facts, 030), §8.2 (approval engine port and subject API, 031) and §8.3 (what 032 provides to
033).

## Summary

A new business package `internal/purchaserequest` owns purchase requests (PR-…), their lines (PRL-…), their
routings to the requisitioner (PRR-…) and the Purchase Request Configuration. A PR is raised from line items
of one plan and one company code, reserves its lines' amounts from the first save (balance computed on read,
checked inside the one-at-a-time `Write`), passes FR-021's gates, and is approved — and its changes
re-approved — by the approval engine of 031 through the subject API (`SubmitSubject`, `RecallSubject`,
`DecideSubject`, `SectionOf`); the engine calls back `ReadyCheck` and `RoundChanged`, where 032 sets the
status, keeps the approved version and routes the PR. Routing is in-app: a write-once routing record and a
PR form downloadable as XLSX or CSV; the Waiting for GSAP view of the Purchase requests page is the
requisitioner queue. 032 declares the `GSAP` port that 033 implements and stores GSAP's answer. Shell
attributes gain two levels and roles; access gains four permissions and a Requisitioner role; history gains a
delete guard so nothing a PR draws on can be deleted. The web app gets a `purchase-requests` feature, sections
on the line-item and activity pages, a PR preview on the Approvals page and a Settings card.

## Technical Context

**Language/Version**: Go (pinned in `go.mod`); TypeScript 5.9, React 19.
**Primary Dependencies**: Gin (handler layer), pgx/v5, sqlc, goose; TanStack Query, React Router, React Hook
Form + Zod, Radix, lucide-react, CSS Modules. No new dependency (research R16).
**Storage**: PostgreSQL 18; migration `00020_purchase_requests.sql`; queries in
`backend/internal/platform/database/queries/purchase_requests.sql`.
**Testing**: Go `testing` + `go-cmp` with fakes (`purchaserequesttest`, `store_fake_test.go`); contract tests
through `internal/platform/apitest`; scope-isolation tests on PostgreSQL (`TEST_DATABASE_URL`); Vitest +
Testing Library + `user-event` + MSW (`reply()`) + `vitest-axe` + `fast-check`; Playwright + axe on the real
API and PostgreSQL (`E2E_API_PORT=18132 E2E_WEB_PORT=14132` backend lane, `18142/14142` frontend lane).
**Target Platform**: local laptop, servers on 127.0.0.1.
**Project Type**: web application (Go API + React web app).
**Performance Goals**: SC-007 — the Purchase requests page's first page ≤ 2 s at p95 with 10,000 PR lines;
one statement per page plus two (lines, vendor totals); no N+1 (B5, 028).
**Constraints**: constitution 4.4.0 — files ≤ 300 non-blank lines, functions within the metrics, comments ≤
5%, no TODO; scope in SQL; money in minor units; injected clock; fail-closed route table.
**Scale/Scope**: 22 new operations, 1 migration, 4 new tables, 4 permissions, 2 attribute levels, ~3 web
pages + 3 sections + 1 settings page.

## Constitution Check

| Principle | How 032 meets it |
|---|---|
| I Spec first | Every operation and screen traces to spec FRs; requirement IDs in test names (`TestSave/032 FR-012 …`, `describe("032 FR-012: …")`). |
| II Shell business | Shell's statuses, fields and words (Purchase request, PR line, HX key, Requisitioner, MoA approver); PR attributes are Settings data; GSAP fields read-only and labelled; no e-mail. |
| III Documented design | research.md (R1–R20), data-model.md (tables, types, ports, rules), contracts/ (OpenAPI fragment, routes and web interfaces), threat and risk models below; no new ADR (ADR-0016 covers the GSAP port; the delete guard is a small hook, not cross-cutting architecture). |
| IV Two parts, server truth | Every figure (requested, available, totals, invoiced, closed amounts, waiting for GSAP) is computed by the API; the web app shows them. Other areas are reached through consumer ports implemented in `internal/app`. |
| V Simplicity | One package, use case per file; balance computed, not stored; approval section types reused from `approval`; one generic delete guard instead of five checks. |
| VI Meaningful tests | Unit tests for every rule (table below), a test per acceptance scenario, a contract test per operation, scope-isolation tests for `GetInScope`, `ListInScope`, `Page`, `OnLineItems`; each web view's five states; e2e for US1–US3. |
| VII Security | Route rules fail closed; scope from the actor only; 404 for out-of-scope PRs; inputs validated with limits (200 lines, 120/2,000 characters); PR form cells formula-safe; no personal data logged. |
| VIII Clean construction | Comment budget per file; no TODO; one concept per file. |
| IX Experience | Standard CRUD; status and completeness chips; all errors at once; confirmations stating amounts released; one yellow action per page (Edit on the PR page; Save/Submit changes in the form); unavailable actions say why. |
| X Palette | Status chips use the existing status tokens; no new colours. |
| B1–B9 | Gin only in handlers; sqlc + goose; store interface declared by the service; `Write` per change with its log entry; `record.Meta` + version checks; `history.MoveToBin` + registered kinds. |
| F1–F10 | Feature folder; hooks in `api.ts`; Zod for forms and URL state; MSW typed from the contract; no `any`/`as`/`!`; axe in every component test. |

Governance → Alignment: `lineitem`, `campaign`, `history`, `settings`, `access` predate 2.0.0 and are touched
only where 032 needs it (one setter each, the levels and roles); they keep their structure.

### Threat model (STRIDE) for the new operations

| Threat | Where | Response |
|---|---|---|
| Spoofing: acting as another person | every operation | session cookie → actor (001); the actor comes only from the session; scope only from the actor's grants |
| Tampering: claiming more than a line item's balance | create, save, restore | balance rule inside the `Write` (one Write at a time), never trusted from the client; amounts parsed as decimals with ≤ 2 places |
| Tampering: editing an approved PR without approval | save | `then` must fit the status; Approved needs `submitChanges` (gates + modification round) except added lines under "withoutApproval" |
| Tampering: forging GSAP fields | save | the input has no GSAP fields (`additionalProperties: false`); only the `GSAP` port's answer writes them (033) |
| Tampering: formula injection in the PR form | form download | `platform/sheet` writes text that starts with = + - @ tab or CR with a leading apostrophe (D-120) |
| Repudiation | every change | one activity-log entry per change in the same `Write`; routings write-once (trigger); approval actions write-once (008) |
| Information disclosure: another market's PRs | every read | `purchase_request_in_scope` in SQL; list filtered by the `purchaseRequests.list` scope; 404 for out-of-scope; isolation tests |
| Information disclosure: names of hidden PRs in refusals | delete guards | the guard names PRs only by ID and name to people who can see the deleted record's plan (the PR's plan is the same plan) |
| Denial of service: huge bodies or lists | create, save, list, options | request body limit (web), ≤ 200 lines per PR, page sizes 50/100/200, options ≤ 50 |
| Elevation of privilege: approving one's own PR outside a workflow | approve | the engine's decide rules (031): only named approvers, delegates, backups or on behalf; self-approval only when the workflow names the owner or MoA approver (FR-027) |
| Elevation: Requisitioner editing PRs | save, delete, submit | Requisitioner role lacks the three PR change permissions (403) |

### Risk model

| Risk | Response | Owner |
|---|---|---|
| 031's subject API differs from brief §8.2 (names, `Decision`, `Section`) | The `Approvals` port names 032's needs; the implementer adapts the port's signatures to 031's delivered types; contract schema `PurchaseRequestApprovalSection` reuses the existing approval schemas | Team |
| 031 calls `RoundChanged` only at round start and end | Status stays Submitted until the end (cosmetic); listed as an expectation (research R5) for the orchestrator | Orchestrator |
| Migrations 00017–00019 change the same checks (`shell_attributes.role`, `roles.permissions`) | 00020 re-creates each check from the latest definition plus 032's values (data-model §1.5); implementer reads the merged migrations first | Team |
| Rates move PR lines in other currencies above planned cost | Shown as negative Available ("Requested above planned cost by …"); new claims refused; accepted (D-303) | Product owner |
| Hive9 legacy HX keys (gap analysis L18) | Unchanged by 032; HX derived from MAPP's number | Product owner |
| Purchase requests page performance at 10,000 lines | Indexed columns, one statement per page, benchmark task T089 | Team |

## Project Structure

### Documentation (this feature)

```text
specs/032-purchase-requests/
├── spec.md · plan.md · research.md · data-model.md · quickstart.md · tasks.md
├── contracts/openapi.yaml (22 operations) · contracts/routes.md
└── checklists/requirements.md   (the /speckit-analyze report is returned to the orchestrator)
```

### Source code

```text
backend/
├── api/openapi.yaml                                   # fragment merged (merge rules 1–3)
├── internal/
│   ├── purchaserequest/                               # NEW business area
│   │   ├── model.go  model_routing.go  views.go  views_list.go  messages.go
│   │   ├── ports.go  store.go  store_postgres.go  store_postgres_list.go  store_postgres_routing.go
│   │   ├── service.go          # Service, NewService(store, tools, deps), UseApprovals, UseGSAP
│   │   ├── balance.go          # available to request (data-model §3.1)
│   │   ├── validate.go  validate_lines.go   # save rules (§3.2)
│   │   ├── create.go  save.go  defaults.go  options.go
│   │   ├── readiness.go        # gates (§3.4) — ReadyCheck, submission, completeness
│   │   ├── subject.go          # approval.PurchaseRequests: PurchaseRequest, ListInScope, ReadyCheck, RoundChanged (§3.5)
│   │   ├── approval.go         # Approval section, Submit, Recall, Decide, preview (wraps the engine)
│   │   ├── modification.go     # approved version: take, compare, restore
│   │   ├── routing.go  form_file.go          # route, resend, PR form rows and file (§3.6)
│   │   ├── close.go  delete.go  kinds.go     # close (§3.8), delete + history kinds + PrepareRestore (§3.9)
│   │   ├── list.go  sections.go              # page (§3.7), line-item and activity sections
│   │   ├── configuration.go  access.go       # configuration; ActionAvailability per action
│   │   ├── handler.go  handler_approval.go  handler_config.go
│   │   ├── purchaserequesttest/store.go      # fake of stores.PurchaseRequestStore
│   │   └── *_test.go, store_fake_test.go, store_postgres_test.go (isolation)
│   ├── app/
│   │   ├── purchaserequests.go               # wiring, UseApprovals/UsePurchaseRequests, guards
│   │   ├── purchaserequest_lineitems.go      # LineItems port over lineitem, activity, settings (030 facts)
│   │   ├── purchaserequest_deps.go           # Plans, People, Currencies, Attributes, Workflows, Invoices
│   │   └── purchaserequest_guards.go         # history delete guard, lineitem and campaign ports
│   ├── stores/purchaserequests.go  stores/storestest (+ purchase requests)
│   ├── access/routes_purchase_requests.go  routes.go (joinRules)
│   ├── platform/authz/catalogue.go (+4 permissions, group)  platform/ids/ids.go (+PR, PRL, PRR)
│   ├── history/delete_guard.go (UseDeleteGuard; MoveToBin calls it)
│   ├── lineitem/purchase_requests.go (UsePurchaseRequests; planned cost/currency rule in update)
│   ├── campaign/purchase_requests.go (UsePurchaseRequests; move-to-other-plan rule)
│   ├── settings/ (levels, roles, locks, nouns, reference source)
│   └── platform/database/migrations/00020_purchase_requests.sql, queries/purchase_requests.sql, sqlcdb (generated)
└── seed/purchase_requests.go, seed/access.go (+Requisitioner), seed/people (+Lea Santos), seed/settings_attributes.go (+PR levels)
frontend/
├── src/features/purchase-requests/                    # NEW feature folder
│   ├── index.ts  api.ts  schemas.ts  purchase-request-text.ts  purchase-request-links.ts
│   ├── PurchaseRequestsPage.tsx (+ .module.css)  PurchaseRequestFilters.tsx  VendorGroupTable.tsx
│   ├── PurchaseRequestPage.tsx  PurchaseRequestHeader.tsx  PurchaseRequestLinesTable.tsx  ApprovedVersionCard.tsx
│   ├── PurchaseRequestApprovalSection.tsx  RoutingHistory.tsx  GsapFields.tsx
│   ├── PurchaseRequestCreatePage.tsx  PurchaseRequestEditPage.tsx  PurchaseRequestForm.tsx  LineRowsEditor.tsx
│   ├── LineItemPicker.tsx  ClosePurchaseRequestDialog.tsx  DeletePurchaseRequestDialog.tsx  RemoveLineDialog.tsx
│   ├── SubmitPurchaseRequestDialog.tsx  LineItemPurchaseRequests.tsx  ActivityPurchaseRequests.tsx
│   ├── PurchaseRequestSettingsPage.tsx  settings-sections.ts  HxKey.tsx  PurchaseRequestStatusChip.tsx
│   └── *.test.tsx
├── src/features/approvals/ (PurchaseRequestPreviewPanel.tsx, api.ts additions, index.ts exports ApprovalHistory, RecallDialog)
├── src/features/notifications/ (PR links)  src/features/line-items/LineItemPage.tsx  src/features/activities/ActivitySections.tsx
├── src/features/settings/shell-attributes/ (two level tabs and labels)
├── src/app/{navigation.ts, routes.tsx, pages.tsx, settings-sections.ts}  src/ui/record-links.ts
├── src/testing/ (MSW handlers and fixtures for the new operations)
└── e2e/purchase-requests.spec.ts
```

**Structure Decision**: the standard two-part layout; one Go package and one feature folder per business
area (constitution B2, F2).

## Backend design

### Service API (`purchaserequest.Service`; every method takes `ctx` and the actor)

| Method | Operation | Notes |
|---|---|---|
| `Page(ctx, a, q)` | 1 | list scope; vendor groups; display currency |
| `Create(ctx, a, in)` | 2 | §3.2; status Created; log "created" |
| `Defaults(ctx, a, lineItemID)` | 3 | FR-003 |
| `Options(ctx, a, q)` | 4 | FR-010; ≤ 50 |
| `Get(ctx, a, id)` | 5 | view scope |
| `Save(ctx, a, id, in)` | 6 | §3.2; removed lines → `MoveToBin(purchaseRequestLine)`; `then` resubmit/submitChanges → `Approvals.SubmitSubject` in the same `Write` |
| `Delete(ctx, a, id)` | 7 | §3.9 |
| `ClosePreview`, `Close` | 8, 9 | §3.8 |
| `Resend(ctx, a, id, version)` | 10 | §3.6 |
| `Form(ctx, a, id, routingID, format)` | 11 | file bytes + name |
| `Approval`, `Submission`, `Submit`, `Decide`, `Recall`, `Preview` | 12–18 | via `Approvals`; kind per research R5 |
| `LineItemSection`, `ActivitySection` | 19, 20 | FR-018 |
| `Configuration`, `SaveConfiguration` | 21, 22 | log kind `purchaseRequestConfiguration` |
| `PurchaseRequest`, `ListInScope`, `ReadyCheck`, `RoundChanged` | engine | `approval.PurchaseRequests` (brief §8.2) |
| `DrawnOn(ctx, lineItemIDs)`, `Requested(ctx, lineItemID, currency)` | guards | for `internal/app`'s guard adapters |
| `References(ctx)` | Settings usage | attribute values of every PR and line |

### Wiring (`internal/app/purchaserequests.go`)

`prs := purchaserequest.NewService(stores.PurchaseRequests, tools, deps)`; `prs.UseApprovals(approvals)`;
`approvals.UsePurchaseRequests(prs)`; `history.UseDeleteGuard(guardOf(prs))`;
`lineItems.UsePurchaseRequests(prs)`; `campaigns.UsePurchaseRequests(campaignGuard(prs, stores))`;
`settings.AddReferenceSource(prReferences(prs))`; register kinds `purchaseRequest`, `purchaseRequestLine`;
`purchaserequest.NewHandler(prs).RegisterRoutes(r)`.

### Activity-log entries (FR-057)

| Action | Kind · action | Details |
|---|---|---|
| Create | purchaseRequest · created | "1 line, 9,200.00 EUR" |
| Save | purchaseRequest · changed | one change per field (history.Change), lines added and removed named; removed lines also get their bin entry |
| Save and re-submit / Submit changes | purchaseRequest · changed | + the engine's submitted/resubmitted entry |
| Submit, re-submit, recall, approve, decline | (engine) | approval entries (031) |
| Routing | purchaseRequest · routed | "Original · routed to DE_LUBES" |
| Resend | purchaseRequest · routed | "Resend · routed to …" |
| Close | purchaseRequest · closed | "Released 1,700.00 EUR" |
| Delete, restore | history | standard |
| Configuration | purchaseRequestConfiguration · changed | old → new |

### User-visible messages (exact texts; `messages.go`)

| Key | Text |
|---|---|
| switchedOff | Purchase requests are switched off for your organisation. |
| onlyAvailable | Only %s %s of line item %s is not yet requested. |
| otherCurrency | %s %s is %s %s at today's rate. |
| twoCompanyCodes | One purchase request cannot charge two company codes (%s and %s). |
| otherPlan | Line item %s is in another plan than this purchase request. |
| alreadyOnPR | Already on this purchase request |
| nothingLeft | Nothing left to request |
| activityNotApproved | Activity '%s' must be approved before this purchase request can be submitted. |
| noSwbs | Line item %s '%s' has no SWBS code yet (%s). |
| noCompanyCode | Activity '%s' has no company code. |
| approvedNoDelete | Approved purchase requests cannot be deleted — close them instead. |
| awaitingNoDelete | Purchase requests awaiting approval cannot be deleted — recall them first. |
| closedNoChange | Closed purchase requests cannot be changed. |
| approvedNoRemove | Lines of approved purchase requests cannot be removed — close the purchase request instead. |
| noLinesOnApproved | Your organisation does not allow lines to be added to approved purchase requests. |
| lastLine | A purchase request needs at least one line — delete the purchase request instead. |
| awaitingNotice | Awaiting approval — saving changes will restart approval. |
| editToChange | Edit the purchase request and choose Submit changes. |
| belowInvoiced | The amount cannot be below the %s %s already invoiced on this line. |
| noFormYet | The PR form is created when the purchase request is approved. |
| drawnOn | Can't delete — purchase request %s draws on line item %s. Delete or close the purchase request first. |
| plannedBelowRequested | Purchase requests %s already request %s %s of this line item; the planned cost cannot be lower. |
| campaignMove | Can't move — purchase request %s draws on a line item of this campaign. |
| filledByGsap | Filled by GSAP |
| hxLabel | Generated from the line-item ID |
| swbsLabel | Filled by GSAP through the line item |

### Rules and their tests (Principle VI)

| Rule (data-model) | FR | Test (package `purchaserequest`, fixed clock 2026-10-27 09:00 UTC, fakes) |
|---|---|---|
| Available = planned − claims, other currencies converted, Closed at closed amount | FR-012, FR-017 | `TestAvailable` (table: none; one claim; boundary 3,200.00 vs 3,200.01; EUR line on USD item 3,700.00 → 4,021.74; closed PR; deleted PR not counted) |
| Pre-fill | FR-003 | `TestDefaults` |
| Save problems all at once | FR-005, FR-007, FR-011, FR-053 | `TestCreate_ValidationAllAtOnce` (name, workflows, delivery date, vendor e-mail, amount format) |
| One plan, one company code, once per PR | FR-013, FR-014 | `TestCreate_LinesRules` |
| Remove lines only Created/Declined; ≥ 1 line | FR-015 | `TestSave_RemoveLine` |
| Version conflict | FR-016 | `TestSave_OutOfDateVersion` |
| `then` fits status; Approved needs submitChanges; withoutApproval adds lines | FR-023, FR-036, FR-039 | `TestSave_ThenByStatus` (table over 6 statuses × 3 thens) |
| Modified amount ≥ invoiced | FR-038 | `TestSave_BelowInvoiced` |
| Gates | FR-021 | `TestReadyCheck` (table: no line, activity not approved, SWBS missing, two company codes, none, inactive workflow, wrong object type, MoA empty, overall limit empty, needed attribute, all at once) |
| Round kind | R5 | `TestSubmit_KindByStatus` |
| Status mapping | FR-022, Status Model | `TestRoundChanged` (every row of data-model §3.5, plus a refused pair, plus switched off) |
| Approved version kept and restored | FR-036, FR-037 | `TestModification_ApproveAndRestore` |
| Routing rows, recipients, Sent date once | FR-030, FR-033 | `TestRoute` (Email group; recipients; none → "Requisitioner queue"; resend keeps Sent date; changed lines) |
| PR form file | FR-031 | `TestForm_ColumnsAndFormulaSafety` |
| Waiting for GSAP | FR-032 | `TestWaitingForGSAP` (table: not routed, no PO, amount changed, currency changed, all matched) |
| Close amounts | FR-041 | `TestClose` (no PO → 0.00; invoiced 7,500.00 of 9,200.00; invoiced above amount keeps amount; change pending refused) |
| Delete and restore | FR-040, FR-042 | `TestDelete_ByStatus`, `TestRestore_BalanceTaken` |
| Switched off | FR-004 | `TestSwitchedOff_RefusesEveryChange` (table over create, save, submit, decide via RoundChanged, close, delete, restore) |
| Access per action | FR-049, FR-050, FR-056 | `TestAccess_Availability` (Planner, Requisitioner, Viewer, owner, other team) |
| Total requested on line item and activity | FR-018 | `TestSections` |
| Guards | FR-019 | `app`: `TestDeleteGuard_RefusesEveryLevel_FR019`; `lineitem`: `TestUpdate_PlannedCostBelowRequested_032FR019`; `campaign`: `TestMove_OtherPlanWithPR_032FR019` |
| Attribute levels and roles | FR-006 | `settings`: `TestAttributeLevels_PurchaseRequest_032FR006`, `TestRoleLocks_032FR006` |
| Permissions catalogue | FR-049 | `authz`: catalogue test updated (group, scoped) |
| Scope isolation | FR-050 | `store_postgres_test.go`: `GetInScope`, `ListInScope`, `Page`, `OnLineItems` (plan scope, list scope, owner, MoA approver, approver in a round, other team) |
| Contract | all | `handler_test.go` (external package) through `apitest`: one test per operation, status codes and envelopes |

### JavaScript budget (F8)

The feature is lazy-loaded per route (`pages.tsx`): initial route unchanged (budget recorded in the
repository stays). Route chunk `purchase-requests` ≤ 60 kB gzip (no new library; tables and forms reuse
`src/ui`); the line-item and activity sections are small components inside their pages' chunks (≤ 6 kB
gzip added each).

### Shared files touched (merged at integration)

`backend/api/openapi.yaml`; `backend/internal/access/routes.go` (+ new `routes_purchase_requests.go`);
`backend/internal/app/*` (new files + `app.go`/`areas.go` registration, `history_access.go`, `app_test.go`
`notYetServed`); `backend/internal/platform/ids/ids.go`; `backend/internal/platform/authz/catalogue.go`;
`backend/internal/stores/*`; `backend/seed/*`; `backend/internal/history` (delete guard);
`backend/internal/lineitem`, `backend/internal/campaign`, `backend/internal/settings` (small additions);
`docs/product/{domain-model,field-catalogue,glossary,design-system}.md` (sections on purchase requests only);
`frontend/src/app/{navigation.ts,routes.tsx,pages.tsx,settings-sections.ts}`; `frontend/src/ui/record-links.ts`;
`frontend/src/api/generated/*` (regenerated); `frontend/src/testing/*`.

## Frontend design

- **Purchase requests page**: page header "Purchase requests" (no primary button: PRs are raised from line
  items; a secondary "New purchase request" opens the line-item step), search, filter chips (Status, PR type,
  Vendor, Owner, Created, Waiting for GSAP), a grouped table (vendor rows with count and totals, expandable PR
  rows with status chip, expandable line rows with HX key and line item link), pager. URL state through
  `schemas.ts` (Zod; property test with fast-check). States: loading skeleton, empty (two texts), error with
  request ID, no permission, switched off.
- **PR page**: header with title, status chip, completeness chip linking to the problems, notice, actions
  (Edit — the one yellow action; Submit/Re-submit; Recall; Close; Resend PR form; Download PR form; Delete).
  Sections: Details (standard fields, attributes, read-only derived fields, GSAP fields with "Filled by
  GSAP"), Lines (table; HX key with copy; PO fields), Approved version (while a change is In review),
  Approval (sections from `GET …/approval` using `ApprovalChain`/`ApprovalHistory`), Routing history
  (download per routing), History.
- **Create and edit pages**: one form (React Hook Form + Zod) with header, attributes (the existing Shell
  attribute fields component at levels purchaseRequest and purchaseRequestLine), a lines editor (Add line →
  `LineItemPicker` dialog; Remove line → `RemoveLineDialog`), the footer's primary button labelled by the
  status: Save · Save and re-submit · Submit changes; server errors mapped onto fields (`lines[1].amount`);
  unsaved-changes prompt (shared blocker).
- **Line item page** section "Purchase requests": Total requested, Available to request, rows, "New purchase
  request" (secondary). **Activity page** section "Purchase requests": total in the plan currency and a row per
  line item.
- **Approvals page**: rows of PR kinds open `PurchaseRequestPreviewPanel` with Approve/Decline.
- **Settings › Purchase requests**: a form with Purchase Support (switch) and Lines on approved purchase
  requests (radio group); read-only without permission. **Shell attributes**: two more level tabs.

## Complexity Tracking

| Item | Why | Simpler alternative rejected |
|---|---|---|
| `purchaserequest` imports `approval` (types) | Reuses the engine's section and subject types (brief §8.2) | Re-declaring rounds and history views in 032 (duplication, drift) |
| JSON snapshots (approved version, routed form rows) | Never queried; restored or downloaded whole | A column per field doubles two tables (research R2) |
| SQL scope function with 7 parameters | SQL, not a Go function; mirrors `activity_in_scope` | Two queries and a merge in Go (scope after loading, forbidden by B5) |
| `Store.Claims` and `Store.DrawnOn` read every purchase request whoever asks | The balance rule (FR-012) and the delete guards (FR-019) must count every claim, including PRs the person cannot see; they answer an amount or a refusal, never a record. Same sanctioned pattern as Settings' "Used by" counts (`authz.Unrestricted`, D-228); the scoped reads keep their isolation tests | Scoped claims would let a person over-claim a line item through PRs they cannot see (breaks SC-003) |
| Sections and approval answers are not paginated | Bounded by one line item's PRs, one activity's line items, one PR's rounds (B5 applies to list endpoints; the Purchase requests list and the options are paginated / capped) | Paging a section of a handful of rows |
