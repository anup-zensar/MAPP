---

description: "Task list for feature 032 Purchase Requests on MAPP's Side"
---

# Tasks: Purchase Requests on MAPP's Side

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md),
[contracts/openapi.yaml](contracts/openapi.yaml), [contracts/routes.md](contracts/routes.md),
[quickstart.md](quickstart.md)

**Prerequisites**: features 030 (SWBS facts, brief §8.1) and 031 (approval engine subject API, brief §8.2)
merged. Read migrations 00017–00019 before writing 00020 (data-model §1.5).

**Tests**: required (constitution VI). Write each test first and see it fail. Go test names carry the FR,
e.g. `TestAvailable/032 FR-012 3,200.01 EUR is refused`; web tests `describe("032 FR-012: …")`.

**Tracks**: two implementers work in parallel from the merged contract (T001):
- **[BE]** backend (`backend/`), ports for the e2e API 18132, database `mapp_f032` / `mapp_e2e_032`;
- **[FE]** frontend (`frontend/`), MSW mocks typed from the contract until the backend lands; e2e lane ports
  18142/14142, database `mapp_e2e_032w`.
No Git branches or commits (product owner, 2026-10-27).

## Format: `[ID] [P?] [Story?] [Track] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: US1–US7 of spec.md (Setup, Foundation and Polish have none)
- **[Track]**: [BE] backend or [FE] frontend

---

## Phase 1: Setup (shared)

- [x] T001 [BE] Merge `specs/032-purchase-requests/contracts/openapi.yaml` into `backend/api/openapi.yaml` following its merge rules 1–3 (tag, 18 paths / 22 operations, parameters, schemas; `AttributeLevel` + `purchaseRequest`, `purchaseRequestLine`; `PermissionKey` + four keys; `PermissionGroup` + "Purchase requests"). Add the 22 operations to `notYetServed` in `backend/internal/app/app_test.go`. Run `go test ./internal/app/...`. **Both tracks start after this task.**
- [X] T002 [P] [FE] Regenerate the API types with `pnpm run api:types` (`frontend/src/api/generated/`); fix exhaustive switches over `AttributeLevel` and `PermissionKey` that now fail `pnpm run typecheck` (labels "Purchase request", "PR line", and the four permission labels of contracts/routes.md).
- [x] T003 [P] [BE] Add `PrefixPurchaseRequest = "PR"`, `PrefixPurchaseRequestLine = "PRL"`, `PrefixPurchaseRequestRouting = "PRR"` to `backend/internal/platform/ids/ids.go` and `Prefixes`; extend `ids_test.go`.
- [x] T004 [P] [BE] Test first in `backend/internal/platform/authz/catalogue_test.go` (`TestCatalogue/032 FR-049 purchase-request permissions are scoped, in group Purchase requests`), then add `PurchaseRequestsEdit`, `PurchaseRequestsDelete`, `PurchaseRequestsSubmit`, `PurchaseRequestsList` and `GroupPurchaseRequests` (after GroupFinance in `Groups`) to `catalogue.go` with the labels and nouns of contracts/routes.md.
- [x] T005 [BE] Write `backend/internal/platform/database/migrations/00020_purchase_requests.sql`: the four tables of data-model §1.1–§1.4 with every CHECK, UNIQUE, FK and index; the write-once trigger on `purchase_request_routings`; re-created checks of §1.5 (`shell_attributes.level`, `shell_attributes.role`, `roles.permissions`, each from its latest definition plus 032's values); the function `purchase_request_in_scope` of §1.6 using 031's subject columns of `approval_rounds`.
- [x] T006 [BE] Write `backend/internal/platform/database/queries/purchase_requests.sql` (get, get in scope, list in scope by ids, page + page lines + vendor totals, on line items in scope, claims, drawn on, put/delete header and line, routings, configuration, all for references); run `sqlc generate && sqlc vet`.

---

## Phase 2: Foundation (blocks every story)

### Backend

- [x] T007 [P] [BE] Test first `backend/internal/purchaserequest/model_test.go` (`TestHXKey/032 FR-020 L-1006 gives HX1006`, no padding or separator), then `model.go` and `model_routing.go`: the types of data-model §2.1 (statuses, `LinesOnApproved`, `RoutingType`, `PurchaseRequest`, `Line`, `GSAPHeader`, `PurchaseOrderLine`, `Version`, `VersionLine`, `Routing`, `FormRow`, `Configuration`, `ViewScope`) and `HXKey`. Package doc comment of one or two lines.
- [x] T008 [P] [BE] Write `backend/internal/purchaserequest/store.go` (Transactor, Store, Claim, ListQuery, Page — data-model §2.2) and `ports.go` (Deps, LineItems, LineItemFacts, Plans, People, Currencies, Attributes, Workflows, Invoices, POLineRef, Approvals over 031's subject types, GSAP, RoutedPR, RoutedLine, GSAPOrder, OrderLine — data-model §2.3).
- [x] T009 [BE] Write the fakes: `backend/internal/purchaserequest/purchaserequesttest/store.go` (fake of `stores.PurchaseRequestStore` on `platform/fake`, scope applied like the SQL function) and `store_fake_test.go` for the package's own tests; add `PurchaseRequests` to `backend/internal/stores/stores.go`, `stores/purchaserequests.go` and `stores/storestest` (feature 029 pattern).
- [x] T010 [BE] Write `backend/internal/purchaserequest/store_postgres.go`, `store_postgres_routing.go` (everything but `Page`) and the scope-isolation tests in `store_postgres_test.go`: `TestGetInScope/032 FR-050 …`, `TestListInScope/032 FR-050 …`, `TestOnLineItems/032 FR-050 …` — plan in scope, list permission scope, owner, MoA approver, approver named in a round (031's columns), another team's PR not returned.
- [x] T011 [P] [BE] Settings (`backend/internal/settings`): tests first `TestAttributeLevels/032 FR-006 purchaseRequest and purchaseRequestLine accept optional and neededToSubmit, refuse neededToPublish and multipleChoiceSplit` and `TestRoleLocks/032 FR-006 an attribute with a role keeps level and type and cannot be deleted`; then add `LevelPurchaseRequest`, `LevelPurchaseRequestLine` to `Levels`, the ten roles of data-model §4, `levelNouns`, the level labels in `kinds.go`, the marking and type rules, the role locks ("MAPP reads this attribute for purchase requests") and `Reference` support for the two levels in `usage.go`.
- [x] T012 [P] [BE] Write `backend/internal/access/routes_purchase_requests.go` with the 22 rules of contracts/routes.md and add `purchaseRequestRoutes()` to `joinRules` in `routes.go`; `go test ./internal/app/ -run Access` proves every contract operation has a rule.
- [x] T013 [BE] Write `backend/internal/purchaserequest/service.go` (`Service`, `NewService(store Store, tools history.Tools, deps Deps)`, `UseApprovals`, `UseGSAP`, the scope helpers `viewScope(ctx)` and `listScope(ctx)`), `messages.go` (every text of plan "User-visible messages"), `configuration.go` read path (`Configuration(ctx)` with the defaults when no row) and the switched-off guard used by every change.
- [x] T014 [BE] Test first `backend/internal/app/history_access_test.go` (`TestHistoryAccess/032 FR-057 PR entries follow the PR's plan; configuration needs activityLog.viewAll; restore needs purchaseRequests.delete or .edit`), then `backend/internal/purchaserequest/kinds.go` (kinds `purchaseRequest` and `purchaseRequestLine` with Snapshot, ChildIDs, Remove, Put, Describe; PrepareRestore filled in T062 and T071), register them in `internal/app`, and extend `app/history_access.go`.
- [x] T015 [BE] Test first `backend/internal/app/purchaserequest_lineitems_test.go` (`TestLineItemFacts/032 FR-021 facts carry activity status, SWBS code and status (030), HX key, company code, cost centre and GL code by role`), then the adapters `app/purchaserequest_lineitems.go` (LineItems over lineitem, activity, settings and 030's facts) and `app/purchaserequest_deps.go` (Plans, People incl. e-mail, Currencies, Attributes with `Roles`/`AttributeID`, Workflows with 031's object type, Invoices — `InvoicedOn` in T082), and the wiring `app/purchaserequests.go` (`UseApprovals`, `approvals.UsePurchaseRequests(prs)`, handler registration, Settings reference source).
- [x] T016 [BE] Seed (`backend/seed/`), tests first in `seed/purchase_requests_test.go` (`TestSeed/032 FR-051 Requisitioner role and Lea Santos, configuration, PR attributes, PR workflows and PR-1001 exist; existing IDs unchanged`): Requisitioner role (RL-1006) and the new permissions of Global planner and Planner in `access.go`; Lea Santos (U-1007, Global Marketing Ops, Requisitioner for All markets); the PR and PR line attributes of the spec's Field Inventory in `settings_attributes.go`; `purchase_requests.go` with the configuration, the two workflows (type purchaseRequest / modifiedPurchaseRequest, level 1 record field `moaApprover`, backup Alex Morgan) and PR-1001 (research R15), after the financial transactions and before the demo history (D-306).

### Frontend

- [X] T017 [P] [FE] Create `frontend/src/features/purchase-requests/` with `index.ts` (one-line purpose), `api.ts` (key factory `purchaseRequestKeys` of data-model §5 and a TanStack Query hook per operation, invalidations as listed there), `schemas.ts` (`purchaseRequestFormSchema`, `purchaseRequestListSearchSchema`) and `schemas.test.ts` (`describe("032 FR-045: list URL state")` with a fast-check property test: any query string parses, invalid values fall back to defaults).
- [X] T018 [P] [FE] Tests first, then `purchase-request-text.ts` (every user text of plan "User-visible messages" and the labels), `PurchaseRequestStatusChip.tsx` (word + status token per status; `describe("032 FR-052: status chip")`) and `HxKey.tsx` (read-only, copy button, "Generated from the line-item ID"; `describe("032 FR-020: HX key")`), with vitest-axe.
- [X] T019 [P] [FE] MSW: `frontend/src/testing/purchase-requests.ts` — handlers for the 22 operations with `reply()` typed from the contract and fixtures for PR-1001 in each status, a routed PR, a closed PR, the list page and the sections.
- [X] T020 [FE] Routing and navigation, tests first (`app/routes.test.tsx`, `app/Sidebar.test.tsx`: `describe("032 FR-044/FR-049: Purchase requests in the sidebar only with purchaseRequests.list")`): add the five routes of contracts/routes.md to `app/routes.tsx` and lazy pages to `app/pages.tsx`; the sidebar item "Purchase requests" after Financials in `app/navigation.ts` (`permission: 'purchaseRequests.list'`); `purchaseRequestHref` in `src/ui/record-links.ts`.

**Checkpoint**: contract merged, schema migrated, packages compile, fakes and MSW in place.

---

## Phase 3: User Story 1 — Raise a purchase request from a line item (P1) 🎯 MVP

**Goal**: create a PR from a line item, reserved at once, with HX key; totals on line item and activity.
**Independent Test**: spec US1.

### Tests first

- [x] T021 [P] [US1] [BE] `backend/internal/purchaserequest/balance_test.go` `TestAvailable` (032 FR-012, FR-017): no claims; one 6,000.00 EUR claim leaves 3,200.00; 3,200.00 passes and 3,200.01 fails; a 3,700.00 EUR line on a USD line item is 4,021.74 USD and the message shows both; a Closed PR counts at its closed amount; the line being saved is not counted twice.
- [x] T022 [P] [US1] [BE] `defaults_test.go` `TestDefaults` (032 FR-003, FR-002): "PR for Stand build", owner the actor, 9,200.00 EUR, overall limit 9,200.00, HX1006; a line item with nothing left answers 400 with "Nothing left to request"; no `purchaseRequests.edit` in the plan 403; a line item outside the scope 404; switched off 409.
- [x] T023 [P] [US1] [BE] `create_test.go`: `TestCreate_ValidationAllAtOnce` (032 FR-005, FR-007, FR-011, FR-053: name, both workflows, delivery date, vendor e-mail "abc", amount "1.234" — one answer, fields by input path); `TestCreate_LinesRules` (032 FR-002, FR-013, FR-014: two company codes, another plan, a line item twice, a deleted line item); `TestCreate_ReservesAtOnce` (032 FR-017, US1 sc.4: activity Submitted, status Created, claims stored); `TestCreate_SwitchedOff` (032 FR-004); `TestCreate_LogEntry` (032 FR-057).
- [x] T024 [P] [US1] [BE] `sections_test.go` `TestSections` (032 FR-018): line item requested and available in its currency, rows limited to PRs the person sees, `create` availability per permission and Purchase Support; activity total in the plan currency; a negative available labelled.
- [x] T025 [P] [US1] [BE] `views_test.go` `TestGet_View` (032 FR-008, FR-009, FR-020, FR-052): derived header fields (company code, cost centres, HX keys, amount and invoiced in the display currency), GSAP fields empty with source "", missing fields, notice per status; `access_test.go` `TestAccess_Availability` (032 FR-049, FR-050, FR-056: Planner, Requisitioner, Viewer, owner, another team — each action's availability and reason).
- [x] T026 [P] [US1] [BE] `handler_test.go` (package `purchaserequest_test`, `apitest`): contract tests for `createPurchaseRequest`, `getPurchaseRequestDefaults`, `getPurchaseRequest`, `getLineItemPurchaseRequests`, `getActivityPurchaseRequests` — 201/200, 400 envelope with several fields, 403, 404 for another market's PR, 409 switched off.

### Implementation

- [x] T027 [US1] [BE] `balance.go` (data-model §3.1) — makes T021 pass.
- [x] T028 [US1] [BE] `validate.go`, `validate_lines.go` (data-model §3.2 for create), `create.go`, `defaults.go` (vendor copied from the role attribute; log entry) — T022, T023 pass.
- [x] T029 [US1] [BE] `views.go`, `access.go` and `Get` (view scope, 404 outside) — T025 passes.
- [x] T030 [US1] [BE] `sections.go` (`LineItemSection`, `ActivitySection`) — T024 passes.
- [x] T031 [US1] [BE] `handler.go` routes for the five operations of T026; remove them from `notYetServed`.

### Frontend (tests first)

- [X] T032 [P] [US1] [FE] `LineItemPurchaseRequests.test.tsx` (`describe("032 FR-001/FR-018: line item Purchase requests section")`): loading, empty, error, forbidden, success; requested and available as the server sends them; New purchase request shown or explained per `create`; switched-off notice; axe.
- [X] T033 [P] [US1] [FE] `ActivityPurchaseRequests.test.tsx` (`describe("032 FR-018: activity Purchase requests section")`): five states, total in the plan currency, a row per line item.
- [X] T034 [P] [US1] [FE] `PurchaseRequestCreatePage.test.tsx` (`describe("032 FR-003/FR-053/FR-054: new purchase request")`): pre-filled from defaults; a 400 maps every field error at once (header and `lines[0].amount`) with a summary; unsaved-changes prompt; success goes to the PR page; 403 and 404 states; axe.
- [X] T035 [P] [US1] [FE] `PurchaseRequestPage.test.tsx` (`describe("032 FR-008/FR-009/FR-020/FR-052: purchase request page")`): header fields, derived fields, "Filled by GSAP" on GSAP fields, HX keys, completeness chip linking to problems, notice, actions hidden or explained per `access`; five states; axe.
- [X] T036 [US1] [FE] `LineItemPurchaseRequests.tsx`, added to `features/line-items/LineItemPage.tsx` after the Financials section.
- [X] T037 [US1] [FE] `ActivityPurchaseRequests.tsx`, added to `features/activities/ActivitySections.tsx` after Line items.
- [X] T038 [US1] [FE] `PurchaseRequestForm.tsx` (header fields, person pickers for Owner and MoA approver, workflow pickers filtered by object type (031), Shell attribute fields at level `purchaseRequest`, one line row with its attributes at level `purchaseRequestLine`) and `PurchaseRequestCreatePage.tsx`.
- [X] T039 [US1] [FE] `PurchaseRequestPage.tsx`, `PurchaseRequestHeader.tsx`, `PurchaseRequestLinesTable.tsx`, `GsapFields.tsx`.

**Checkpoint**: US1 works on its own (API and screens).

---

## Phase 4: User Story 2 — Submit and get it approved (P1)

**Goal**: gates, approval through the engine, statuses, self-approval, recall, re-submit, inbox preview.
**Independent Test**: spec US2.

### Tests first

- [x] T040 [P] [US2] [BE] `readiness_test.go` `TestReadyCheck` (032 FR-021, FR-028): no line; activity not approved (exact message); SWBS missing with its status label; two company codes; none; workflow inactive; wrong object type; MoA approver empty; overall limit empty; needed attribute at each level; all at once; a very large amount is not refused (no MoA or PO threshold).
- [x] T041 [P] [US2] [BE] `subject_test.go` `TestRoundChanged` (032 FR-022: every row of data-model §3.5, a refused pair, switched off → 409), `TestPurchaseRequestFacts` (brief §8.2 facts: USD amount, path, both workflows, plan facts), `TestListInScope` (032 FR-050: plans scope from the engine, list scope from the context).
- [x] T042 [P] [US2] [BE] `approval_test.go` with a fake engine: `TestSubmit_KindByStatus` (research R5), `TestSubmit_ApprovedAnswersEditToChange`, `TestSave_ResubmitInOneStep` (032 FR-023), `TestRecall_OwnerOrSubmitter` (032 FR-024), `TestApproval_SectionsOrder` (032 FR-026: focus section first, then the other kind).
- [x] T043 [P] [US2] [BE] `backend/internal/app/purchaserequests_approval_test.go` through `apitest` with the real engine (031): Submitted → In review → Approved on a two-level workflow (032 FR-022); decline keeps the reservation and notifies the owner (032 FR-025, FR-058); self-approval accepted (032 FR-027); a level from 100,000.00 USD not needed (US2 sc.8); the approver sees the PR in `GET /api/v1/approvals` (032 FR-029).
- [x] T044 [P] [US2] [BE] `handler_approval_test.go` contract tests for the seven approval operations (12–18), including 409 not_ready with the gate problems in `error.missing`, grouped by record.

### Implementation

- [x] T045 [US2] [BE] `readiness.go` (gates, submission view, completeness) — T040 passes.
- [x] T046 [US2] [BE] `subject.go` (`PurchaseRequest`, `ListInScope`, `ReadyCheck`, `RoundChanged` per data-model §3.5) — T041 passes.
- [x] T047 [US2] [BE] `approval.go` and `handler_approval.go` (Approval, Submission, Submit, Decide, Recall, Preview; `Save` with `then: resubmit`) — T042–T044 pass; remove operations 12–18 from `notYetServed`.

### Frontend (tests first)

- [X] T048 [P] [US2] [FE] `PurchaseRequestApprovalSection.test.tsx` (`describe("032 FR-023–FR-026: PR approval section")`): sections in order, chain and history, Approve/Decline/Recall per `actions`, decline needs a comment, "Awaiting approval — saving changes will restart approval." notice; loading, error, forbidden and success states of `GET …/approval`; axe.
- [X] T049 [P] [US2] [FE] `SubmitPurchaseRequestDialog.test.tsx` (`describe("032 FR-021: submit")`): every problem listed at once with a link to its field or record; levels shown; Submit and Re-submit labels; loading and error states of the check; success refreshes the PR; axe.
- [X] T050 [P] [US2] [FE] `features/approvals/PurchaseRequestPreviewPanel.test.tsx` (`describe("032 FR-029: PR in the approvals inbox")`): an inbox row of kind purchaseRequest or modifiedPurchaseRequest opens the PR preview (summary, lines, total, chain), Approve and Decline work, five states, axe; `features/notifications` test (`describe("032 FR-058: PR notifications open the PR")`).
- [X] T051 [US2] [FE] `PurchaseRequestApprovalSection.tsx`, `SubmitPurchaseRequestDialog.tsx`; export `ApprovalHistory` and `RecallDialog` from `features/approvals/index.ts`.
- [X] T052 [US2] [FE] `features/approvals/PurchaseRequestPreviewPanel.tsx`, its hooks in `features/approvals/api.ts`, the subject-kind switch in `ApprovalsPage`; PR links in `features/notifications` and in Home's Waiting for my approval card (items of the PR subject kinds) through `purchaseRequestHref`.

---

## Phase 5: User Story 3 — Route to the requisitioner inside MAPP (P1)

**Goal**: routing at approval, PR form download, requisitioner queue, resend.
**Independent Test**: spec US3.

### Tests first

- [x] T053 [P] [US3] [BE] `routing_test.go`: `TestRoute` (032 FR-030: recipients from Email group then PR recipients, "Requisitioner queue" when empty, Sent dates set once, a "routed" log entry, changed lines marked on Modification), `TestResend` (032 FR-033: Approved or Closed and routed only; Sent date unchanged), `TestWaitingForGSAP` (032 FR-032: not routed, no PO, line without PO line, amount changed, currency changed, all matched).
- [x] T054 [P] [US3] [BE] `form_file_test.go` `TestForm` (032 FR-031, FR-028): XLSX and CSV columns in the spec's order, the notes, MoA approver name and e-mail, a short text starting with "=" written safely, the file name.
- [x] T055 [P] [US3] [BE] Contract tests for `resendPurchaseRequestForm` and `downloadPurchaseRequestForm` (content types, Content-Disposition, 404 for a routing of another PR).

### Implementation

- [x] T056 [US3] [BE] `routing.go` (called by `subject.go` on approved; `Resend`), `form_file.go` (`platform/sheet`), the handler routes; remove operations 10–11 from `notYetServed`.

### Frontend (tests first)

- [X] T057 [P] [US3] [FE] `RoutingHistory.test.tsx` (`describe("032 FR-031/FR-033/FR-035: routing history")`): one row per routing (time, type, recipients, who), XLSX and CSV download links, Resend PR form with its confirmation, "The PR form is created when the purchase request is approved." before approval; axe.
- [X] T058 [US3] [FE] `RoutingHistory.tsx`; Download PR form and Resend PR form actions on `PurchaseRequestPage.tsx`.

**Checkpoint**: the P1 flow (create → submit → approve → routed → form) works end to end.

---

## Phase 6: User Story 4 — Manage the lines (P2)

### Tests first

- [x] T059 [P] [US4] [BE] `options_test.go` `TestOptions` (032 FR-010, FR-002): search by name, ID and HX key; available amounts; "Already on this purchase request", "Nothing left to request", the two-company-codes reason; other plans not listed; at most 50; without `purchaseRequestId` only plans where the person holds `purchaseRequests.edit`.
- [x] T060 [P] [US4] [BE] `save_test.go`: `TestSave_RemoveLine` (032 FR-015: bin entry with the confirmation's amounts, reservation released, last line refused, Approved refused), `TestSave_OutOfDateVersion` (032 FR-016), `TestRestoreLine` (PrepareRestore: PR still Created/Declined and the balance fits, otherwise cannot_restore naming the line item).
- [x] T061 [P] [US4] [BE] Contract tests for `listPurchaseRequestLineItemOptions` and `savePurchaseRequest` (save of a Created PR, 409 version_conflict).
- [x] T062 [US4] [BE] `options.go`; `save.go` for Created and Declined (add, change, remove lines; removed lines through `history.MoveToBin(purchaseRequestLine)`); PrepareRestore of the line kind in `kinds.go`; remove operations 4 and 6 from `notYetServed`.

### Frontend (tests first)

- [X] T063 [P] [US4] [FE] `LineItemPicker.test.tsx` (`describe("032 FR-010/FR-001: line item picker")`: search, available amounts, unavailable rows with their reasons, loading / no match / error states, keyboard and Escape, axe) and `RemoveLineDialog.test.tsx` (`describe("032 FR-015/FR-055: remove line")`: the exact confirmation text with the amount, unavailable reasons).
- [X] T064 [US4] [FE] `LineItemPicker.tsx` (dialog; also the line-item step of `PurchaseRequestCreatePage` when opened without `lineItemId`, FR-001), `LineRowsEditor.tsx`, `RemoveLineDialog.tsx`, `PurchaseRequestEditPage.tsx` (Save for Created and Declined).

---

## Phase 7: User Story 5 — Find, view, edit and delete (P2)

### Tests first

- [x] T065 [P] [US5] [BE] `list_test.go` `TestPage` (032 FR-044–FR-047): vendor groups with totals over all matching PRs, "Not completed" vendor last, search over PR ID, name, vendor, HX key, GSAP PR number and PO number, every filter (incl. PR type and vendor Not completed, created range, waiting for GSAP), sort allowlist inside groups, paging, amounts in the display currency, 409 when switched off.
- [x] T066 [P] [US5] [BE] `store_postgres_test.go` `TestPage_ScopeIsolation` (032 FR-050): only plans where the person holds `purchaseRequests.list`.
- [x] T067 [P] [US5] [BE] `delete_test.go`: `TestDelete_ByStatus` (032 FR-040: each refusal text), `TestDelete_ReleasesToBin` (032 FR-042: lines in the entry, claims gone, Undo), `TestRestore_BalanceTaken` (032 FR-042: refused naming each line item and its available amount; restored Created otherwise).
- [x] T068 [P] [US5] [BE] Guards (032 FR-019), tests first: `backend/internal/history/delete_guard_test.go` (`TestMoveToBin_GuardRefuses`: nothing removed, no log entry); `backend/internal/app/purchaserequest_guards_test.go` through `apitest` (`TestDeleteGuard_RefusesEveryLevel`: line item, activity, initiative, campaign, plan → 409 in_use naming the PR); `backend/internal/lineitem` (`TestUpdate_PlannedCostBelowRequested`, currency change too); `backend/internal/campaign` (`TestMove_OtherPlanWithPR`).
- [x] T069 [P] [US5] [BE] Contract tests for `listPurchaseRequests` and `deletePurchaseRequest`.

### Implementation

- [x] T070 [US5] [BE] `list.go` and `store_postgres_list.go` (one statement per page, then the page's lines and its vendors' totals; indexes of data-model §1.1–§1.2).
- [x] T071 [US5] [BE] `delete.go` and PrepareRestore of the PR kind in `kinds.go`; remove operations 1 and 7 from `notYetServed`.
- [x] T072 [US5] [BE] `history/delete_guard.go` (`UseDeleteGuard`; `MoveToBin` calls it after collecting), `lineitem/purchase_requests.go` (`UsePurchaseRequests`; the rule in the update path), `campaign/purchase_requests.go` (`UsePurchaseRequests`; the rule in move), `app/purchaserequest_guards.go` (adapters with the plan's messages), wired in `app/purchaserequests.go`.

### Frontend (tests first)

- [X] T073 [P] [US5] [FE] `PurchaseRequestsPage.test.tsx` (`describe("032 FR-044–FR-048: Purchase requests page")`): grouped table with vendor totals from the server, expandable PR and line rows, search, filter chips removable one by one, URL state kept on Back, sort, pager, Waiting for GSAP view, states loading / empty / no match with Clear filters / error with request ID / no permission / switched off; axe.
- [X] T074 [P] [US5] [FE] `DeletePurchaseRequestDialog.test.tsx` (`describe("032 FR-040/FR-042/FR-055/FR-056: delete")`: confirmation text with lines and line-item IDs, Undo toast, unavailable reasons) and `PurchaseRequestEditPage.test.tsx` (`describe("032 FR-016/FR-053/FR-054: edit")`: two errors at once, 409 version conflict offers a reload and keeps the entries, unsaved-changes prompt).
- [X] T075 [P] [US5] [FE] In `features/line-items` and `features/activities` delete-dialog tests: the server's 409 in_use message naming the PR is shown (`describe("032 FR-019: delete refused while a PR draws on it")`).
- [X] T076 [US5] [FE] `PurchaseRequestsPage.tsx`, `PurchaseRequestFilters.tsx`, `VendorGroupTable.tsx` (+ module CSS with tokens only).
- [X] T077 [US5] [FE] `DeletePurchaseRequestDialog.tsx` and delete with Undo on `PurchaseRequestPage.tsx`.

---

## Phase 8: User Story 6 — Modify or close an approved PR (P2)

### Tests first

- [x] T078 [P] [US6] [BE] `modification_test.go`: `TestModification_ApproveAndRestore` (032 FR-036, FR-037: approved version kept; approved change becomes the values and routes Modification; declined or recalled restores values and removes added lines), `TestSave_ThenByStatus` (032 FR-023, FR-036, FR-039: table of 6 statuses × 3 `then` values), `TestSave_BelowInvoiced` (032 FR-038), `TestSave_LinesOnApproved` (032 FR-039: notAllowed reason; withoutApproval keeps Approved and routes Modification).
- [x] T079 [P] [US6] [BE] `close_test.go` `TestClose` (032 FR-041, FR-034: no PO → 0.00 and all released; 7,500.00 of 9,200.00 invoiced → 7,500.00 and 1,700.00 released; invoiced above amount keeps the amount; change pending refused; Closed routes Close; Closed refuses every change) and `backend/internal/app/purchaserequest_invoices_test.go` (`TestInvoicedOn/032 FR-043 invoices minus credit notes on the PO line, in USD`).
- [x] T080 [P] [US6] [BE] Contract tests for `getPurchaseRequestClosePreview` and `closePurchaseRequest`.
- [x] T081 [US6] [BE] `modification.go` (take, compare, restore the approved version) and the `submitChanges` and `withoutApproval` paths of `save.go`.
- [x] T082 [US6] [BE] `close.go`, the `Invoices` adapter over the financials store in `app/purchaserequest_deps.go`; remove operations 8–9 from `notYetServed`.

### Frontend (tests first)

- [X] T083 [P] [US6] [FE] `ApprovedVersionCard.test.tsx` (`describe("032 FR-036/FR-037: approved version while a change is in review")`), `ClosePurchaseRequestDialog.test.tsx` (`describe("032 FR-041/FR-055: close")`: per-line current, invoiced, released from the preview; loading, error and unavailable states; Closed notice afterwards), and edit-page footer tests (`describe("032 FR-023/FR-036/FR-039: save label by status")`: Save · Save and re-submit · Submit changes; Add line unavailable with its reason).
- [X] T084 [US6] [FE] `ApprovedVersionCard.tsx`, `ClosePurchaseRequestDialog.tsx`, the edit page's footer label and Add line availability.

---

## Phase 9: User Story 7 — Configuration and access (P2)

- [x] T085 [P] [US7] [BE] `configuration_test.go`: `TestSaveConfiguration` (032 FR-004: settings.manage else 403, version conflict, log entry of kind purchaseRequestConfiguration, defaults without a row) and `TestSwitchedOff_RefusesEveryChange` (032 FR-004: create, save, submit, decide through RoundChanged, close, delete, restore); contract tests for `getPurchaseRequestConfiguration` and `savePurchaseRequestConfiguration`.
- [x] T086 [US7] [BE] `configuration.go` save path and `handler_config.go`; remove operations 21–22 from `notYetServed` (the list must now be free of 032 operations).
- [X] T087 [P] [US7] [FE] `PurchaseRequestSettingsPage.test.tsx` (`describe("032 FR-004: Settings › Purchase requests")`: read-only without `canChange`, save, conflict, five states, axe), `app/settings-sections.test.ts` (the card is listed after Approval workflows), and the Shell attributes page test (`describe("032 FR-006: Purchase request and PR line levels")`).
- [X] T088 [US7] [FE] `PurchaseRequestSettingsPage.tsx`, `settings-sections.ts` (`PURCHASE_REQUEST_SETTINGS_SECTIONS`) wired in `app/settings-sections.ts`; the two level tabs and labels in `features/settings/shell-attributes/`.

---

## Phase 10: Polish and cross-cutting

- [x] T089 [BE] Benchmark `BenchmarkPurchaseRequestsPage` in `backend/internal/app` with `apibench` at 10,000 PR lines (SC-007); record the result in research.md R17.
- [x] T090 [FE] End-to-end `frontend/e2e/purchase-requests.spec.ts` (US1–US3, P1, axe on every page): Priya creates an activity in P-1001 with a line item (Request SWBS code on), submits it; Daniel approves it; Priya raises a PR from the line item (pre-fill, HX key), submits — refused for the SWBS code; Alex runs the simulated SWBS batch (030); Priya submits; Daniel approves from the Approvals page; Lea opens Purchase requests › Waiting for GSAP and downloads the PR form; deleting the line item is refused. Needs the [BE] track merged; run with the ports of quickstart.md.
- [x] T091 [BE] Product documents (sections owned by 032 only): `docs/product/domain-model.md` (purchase request, PR line, reservation, statuses), `field-catalogue.md` (PR fields; line item #16–#17 and activity #37 now live), `glossary.md` (Purchase request, PR line, HX key, Requisitioner, Requisitioner queue, PR form, MoA approver, Available to request), `demo-users-and-data.md` (Lea Santos, PR-1001, PR workflows).
- [X] T092 [FE] `docs/product/design-system.md`: the Purchase requests page, the PR page and form, the line-item and activity sections, the Settings card (screens and wording).
- [x] T093 [BE] Gates: `gofmt -l`, `golangci-lint run --new ./...`, `sqlc generate && sqlc vet`, `go vet ./...`, `go test -race ./...` with `TEST_DATABASE_URL`, `govulncheck ./...`; files ≤ 300 non-blank lines and comments ≤ 5% in every touched file; statement coverage ≥ 80% on new code.
- [X] T094 [FE] Gates: `pnpm run api:check`, `typecheck`, `lint`, `format:check`, `test:coverage` (≥ 80% branch on new code), `build` within the JS budget of plan.md.
- [x] T095 [BE] Run quickstart.md's API walkthrough on database `mapp_f032`, port 18132, and record what each step answered.

---

## Dependencies and execution order

- T001 first; then the two tracks run in parallel. [BE]: T003–T016 → US1 (T021–T031) → US2 (T040–T047) → US3 (T053–T056) → US4 → US5 → US6 → US7 → T089, T091, T093, T095. [FE]: T002, T017–T020 → US1 (T032–T039) → US2 (T048–T052) → US3 (T057–T058) → US4 → US5 → US6 → US7 → T092, T094; T090 last, after the [BE] track is merged.
- Inside a story: tests before code; models → services → handlers.
- US2 depends on US1 (a saved PR); US3 on US2 (approval routes); US4–US7 depend on US1 only, except US6's routing (US3).

## Parallel examples

- After T001: T003, T004, T007, T008, T011, T012 [BE] together with T002, T017, T018, T019 [FE].
- US1 tests: T021–T026 [BE] together with T032–T035 [FE].

## Implementation strategy

MVP = Phases 1–5 (US1–US3: raise, approve, route) — with 033 this is the whole PR→PO demo. Then US4–US7 in
order; every story is demonstrable on the seed data.

**Backend note (implementation).** The [BE] tests are grouped by behaviour rather than one file per source
file: the service rules run through `apitest` on fakes in `backend/internal/purchaserequest/handler_create_test.go`
(T022, T023, T026), `handler_approval_test.go` (T040–T044, T053–T055), `handler_change_test.go`
(T060–T061, T067–T069, T078–T080), `handler_lines_test.go` (FR-039, FR-042 restore), `handler_list_test.go`
(T059, T065, T085), `readiness_test.go`, `balance_test.go`, `model_test.go`, and the scope-isolation tests in
`store_postgres_test.go` (T010, T066); the guard in `history/delete_guard_test.go` (T068); the seed's
expectations in the existing `seed/*_test.go` (T016); access in `access/*_test.go` (T012). T089 measured in
research R17; T095 recorded in the backend report.

## Converge (frontend track)

- [ ] T096 [FE] FR-004: hide the sidebar's Purchase requests link while Purchase Support is off (today only the permission gates it; the page itself says "Purchase requests are switched off for your organisation."). Needs a frame-level read of the configuration — every page test's frame answers then grow by one request.
- [ ] T097 [FE] FR-029, R14: tell purchase-request rows and notifications apart by 031's subject kind everywhere; Approvals' preview switch and Home's Waiting for my approval card still recognise a PR by its "PR-" ID (`isPurchaseRequestId`), because the Home waiting items carry no subject kind.
- [x] T098 [FE] FR-053: a server 400 on a header Shell attribute is listed in the summary; confirm it also shows at the attribute's own field (reported by the form track as not showing) and fix if needed. — Confirmed, no fix needed: the API names the field `attributes.<id>` and the form keys its fields the same way; `PurchaseRequestCreatePage.test.tsx` ("032 FR-053: a refused Shell attribute …") proves the summary and the field's own error.
- [ ] T099 [FE] FR-054: the leave prompt offers Keep editing and Discard changes (design system §2.5); the spec's "save, discard or stay" adds Save — confirm with the product owner.
- [ ] T100 [FE] T090 and 033 T020: run both end-to-end journeys once the [BE] track is merged and fix the selectors the real API answers differently.
