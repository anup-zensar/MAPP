---

description: "Tasks for feature 038 — fixes from exploratory QA of develop@85a3d495"
---

# Tasks: QA Findings — Fixes from Exploratory QA of develop@85a3d495

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md),
[contracts/](contracts/), [quickstart.md](quickstart.md)

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Required (constitution VI). Every finding gets a failing test **before** its fix; the fix task names
the test task it depends on and starts only after that test has been run and seen to fail. Test names cite the
requirement and the finding, e.g. `TestReservation_038_FR001_C1`, `describe('038 FR-013 (H-3): delete a purchase
request', …)`.

**Organization**: one phase per user story (spec order: P1, then P2, then P3). Each task is labelled with its
**lane** — `B1`–`B5` backend, `F1`–`F4` frontend, `D` docs — as defined in [plan.md](plan.md) §Lanes; a lane owns
its files, so tasks of different lanes never touch the same file. `[P]` = can run in parallel with the other `[P]`
tasks of its phase (different files, no unfinished dependency).

## Format: `[ID] [P?] [Story] Lane — Description`

Paths: Go under `backend/internal/` unless they start with `backend/`; web under `frontend/src/` unless they start
with `frontend/`.

---

## Phase 1: Setup (shared, one person, before any lane starts)

- [x] T001 Backend — Contract first: apply [contracts/openapi.yaml](contracts/openapi.yaml) to
  `backend/api/openapi.yaml` (new schemas `OverspendApprovalMode`, `ActivityOverage`; properties on
  `OverspendConfiguration(Input)`, `PurchaseRequestConfiguration`, `ActivityApproval`, `PurchaseRequestApproval`,
  `PurchaseRequestSubmission`, `HomeActivityCounts`, `SpaPeriods`, `DashboardMonths`, `BudgetByMonthRow`; responses
  `TooManyAttempts` (with `Retry-After`) on `signIn` and `TooLarge`; codes `too_many_attempts`, `too_large`; the
  behaviour notes, including dropping the 400 for unknown dashboard currency/team); run `pnpm run api:types` to
  regenerate `frontend/src/api/generated/schema.d.ts`
- [x] T002 Backend — `platform/errs`: constructors `TooManyAttempts(message, retryAfter)` (429) and
  `TooLarge(message)` (413) with codes `too_many_attempts` and `too_large`, and their status mapping in
  `platform/web/respond.go`; unit test `TestErrorCodes_038_FR014_FR054` in `platform/errs` (write it first)
- [x] T003 Frontend — `api/errors.ts`: export `CONFLICT_NEXT_STEP = "Your changes are still here. Save again to
  replace their changes, or Cancel to see theirs."` next to `conflictMessage` (used by T150; the ten files that copy
  the string switch to it when next touched)
- [x] T004 Docs — Decision log D-353 – D-357, feature map row 038 "Specified", amendment notes in 003, 009, 011,
  013, 014, 015, 032, 036, 037 (done with the spec) — FR-062

**Checkpoint**: contract, generated types and error codes in place — all lanes start in parallel.

---

## Phase 2: Foundational

None beyond Phase 1: every story below depends only on T001 – T003.

---

## Phase 3: User Story 1 — A purchase-request change never frees money before it is approved (P1) 🎯 MVP

**Goal**: FR-001 – FR-003 (C-1). **Independent Test**: spec US1.

- [x] T010 [P] [US1] B1 — Failing unit tests `TestReservation_038_FR001_C1` (table: lowered 15,800→12,000 keeps
  15,800; raised 10,000→12,000 reserves 12,000; added line; removed line keeps approved; other currency at the
  current rate; no change in review = today) in `purchaserequest/balance_test.go`
- [x] T011 [P] [US1] B1 — Failing contract test `TestChangeInReview_KeepsApprovedReserved_038_FR001_FR003_C1` in
  `purchaserequest/handler_change_test.go`: QA's sequence (approve PR at 15,800 GBP, lower to 12,000 and submit
  changes, second PR for 3,800 refused with the available problem, decline → available 0.00; variant approve → 3,800
  available; variant recall; line-item and activity Total requested, PR line options and restore room check use the
  same reservation — FR-002)
- [x] T012 [US1] B1 — Fix (after T010, T011 fail): `backend/internal/platform/database/queries/purchase_requests.sql`
  `ListPurchaseRequestClaims` selects `p.change_pending, p.approved_version`; `sqlc generate && sqlc vet`;
  `purchaserequest/store_postgres.go` `Claims` fills `Claim.Approved` and adds `RemovedByChange` claims from the
  approved version; `purchaserequest/balance.go` `requested` counts `max(convert(Amount), convert(Approved))`;
  mirror in `purchaserequest/purchaserequesttest/store.go` and `purchaserequest/store_fake_test.go`
- [x] T013 [US1] Frontend/E2E (F1) — `frontend/e2e/qa-findings-pr-reservation.spec.ts`: Priya lowers an approved PR
  and submits changes; a second PR is refused on the form; Daniel declines; line item shows Available 0.00; axe
  (after T012)

**Checkpoint**: C-1 closed; QA script `area3/repro-overres.mjs` no longer reproduces.

---

## Phase 4: User Story 2 — Financial transactions follow the person's scope (P1)

**Goal**: FR-004 – FR-007 (H-4, H-5). **Independent Test**: spec US2.

- [x] T020 [P] [US2] B2 — Failing tests in `financials/service_test.go`: `TestLink_InvisibleTransactionIsMissing_038_FR007_H5`
  (Mei links FT-1004 → same error as a missing ID), `TestUnlink_OutOfScopeIsMissingBeforeVersion_038_FR007_H5`
  (stale version → 404, no person named), `TestLink_HiddenTargetEqualsMissing_038_FR007_H5`,
  `TestLink_NeedsImportAndLink_038_FR005_H4`, `TestUnlink_NeedsLinkInLinkedPlan_038_FR005`; change
  `TestAccess_NoImportHidesNotLinked_FR028` to expect 404 on opening a not-linked transaction (FR-004)
- [x] T021 [P] [US2] B2 — Failing contract test `TestLinkScopeParity_038_FR007_H5` in `financials/handler_test.go`:
  response bodies for out-of-scope and missing transactions and target line items are byte-identical; GET of a
  not-linked transaction as Mei → 404; list `counts.notLinked` absent for Mei (FR-004)
- [x] T022 [P] [US2] B2 — Failing tests: `access/roles_test.go` `TestRole_LinkNeedsImport_038_FR006_H4` (field
  `permissions`, message "Linking financial transactions needs Import GSAP extracts too — only people who import see
  transactions that are not linked."); `platform/authz/grants_test.go` Table B rows: Global planner and Planner
  without `finance.link`; access check: `finance.link` without `finance.import` is not held; contract test in
  `access/handler_test.go` (`createRole`/`updateRole` → 400 on `permissions`)
- [x] T023 [US2] B2 — Fix (after T020, T021): `financials/service.go` `Link`/`Unlink` order visible → permission →
  `CheckVersion` → state (move to `financials/link.go` if `service.go` passes 300 lines); `financials/transaction.go`
  `mayOpen` answers `NotFound(msgTransactionGone)` for hidden not-linked rows; `financials/list.go`/`overview.go`
  omit the not-linked count when hidden; `app/financials.go` `financialAccess.Link` adds `RequireView` and maps
  hidden to `missingLineItem` (`orGone`); no person's name before visibility
- [x] T024 [US2] B2 — Fix (after T022): `access/roles.go` + `access/messages.go` role rule;
  `platform/authz/grants.go` built-in grants; link check requires import;
  `backend/internal/platform/database/migrations/00027_finance_link_roles.sql` exactly as data-model §1.2
  ("`UPDATE roles SET permissions = array_remove(permissions, 'finance.link') … WHERE built_in IN ('globalPlanner',
  'planner')`")
- [x] T025 [P] [US2] F2 — Failing tests: `features/financials/TransactionPanel.test.tsx` (Link/Unlink only when
  `canLink`/`canUnlink`), `features/financials/FinancialsPage.test.tsx` (no Not linked tab or count when
  `counts.notLinked` is absent), `features/home/FinancialsCard.test.tsx` (no "transactions not linked" line) —
  `038 FR-004/FR-005 (H-4)`
- [x] T026 [US2] F2 — Fix (after T025): `TransactionPanel.tsx`, `FinancialsPage.tsx`, `home/FinancialsCard.tsx` (the
  Not linked tab is filtered out in `FinancialsPage.tsx`; `financials-text.ts` belongs to F3 and is not changed here)
- [x] T027 [P] [US2] F4 — Failing test then fix: `features/people/access/RolePages.test.tsx` shows the server's
  `permissions` problem on the role form; `RoleForm.tsx` places it on the permission list (FR-006)
- [x] T028 [US2] Frontend/E2E (F2) — `frontend/e2e/qa-findings-financials.spec.ts`: Mei sees no Not linked tab and
  gets "no longer exists" for FT-1004; request probes (link FT-1004, unlink out-of-scope, out-of-scope vs missing
  target) give identical answers; Alex links FT-1004 to L-1007 and unlinks it; axe (after T023, T024, T026)

---

## Phase 5: User Story 3 — The plan workbook checks a full-size file in seconds (P1)

**Goal**: FR-008 – FR-011 (H-2). **Independent Test**: spec US3.

- [x] T030 [P] [US3] B4 — Failing measurement: `platform/apibench/workbook_test.go` with `BenchmarkWorkbookCheck`
  (250, 1,000, 5,000 rows generated with `platform/sheet`) and `TestWorkbookCheckStatements_038_FR008_H2`
  (statements of a 100-row check − a 10-row check ≤ 4 × 90; a 1,000-row check ≤ 4,150 statements and ≤ 5 s); record
  the "before" numbers in research.md R3; the generator also writes `frontend/e2e/fixtures/workbook-5000.xlsx`
- [x] T031 [P] [US3] B4 — Failing tests in `workbook/apply_test.go`: `TestProblemValue_Capped_038_FR010_H2` (4,000
  characters → first 200 + "…", FR-010); `TestCheck_ReadsFileOutsideWrite_038_FR009_H2` (fake transactor records that the
  upload was parsed before `Write`, FR-009)
- [x] T032 [US3] B4 — Fix (after T030, T031): `workbook/service.go` parses before `s.tx.Write`; new
  `workbook/prepare.go` reads reference data once per check; new `activity/import.go`
  (`CreateFromImport`/`UpdateFromImport`) and `campaign/import.go` take the prepared context and call the existing
  rule functions (same messages); `app/workbook_create.go`, `app/workbook_update.go` use them (no `NewForm`, no
  overspend limits — 037 FR-012); sibling names kept in memory; `workbook/apply.go` `problem()` cap. All existing
  workbook tests stay green unchanged (FR-008 "same problems")
- [x] T033 [US3] B4 — Scope isolation: for each store read added by T032, a test in that package's
  `store_postgres_test.go` proving records of a plan outside the actor's scope are not returned (e.g.
  `TestListImportContext_ScopeIsolation_038_FR008`); record the "after" numbers in research.md R3
- [ ] T034 [US3] Frontend/E2E (F4) — `frontend/e2e/qa-findings-workbook.spec.ts`: Check file on
  `fixtures/workbook-5000.xlsx` shows the result within 20 s, Import, Undo import; axe (after T032, T074)

---

## Phase 6: User Story 4 — A dashboard link always opens a dashboard (P1)

**Goal**: FR-012 (H-1). **Independent Test**: spec US4.

- [x] T040 [P] [US4] B5 — Failing tests: change `dashboard/handler_test.go` cases `?currency=XYZ` (:137-142) and
  `?teamId=T-9999` (:159) to expect 200 with the default currency and team and `filters` showing them; add export
  cases — `038 FR-012 (H-1)`
- [x] T041 [US4] B5 — Fix (after T040): `dashboard/input.go` `chooseCurrency` and `chooseTeam` fall back; correct
  `specs/036-planning-dashboards-and-export/data-model.md:13-14` and `contracts/api.md:10-11`
- [x] T042 [P] [US4] F2 — Failing test `features/dashboards/DashboardsPage.test.tsx`: when the answer's filters
  differ from the address, the filter bar shows them and the address is replaced (no error state)
- [x] T043 [US4] F2 — Fix (after T042): `DashboardsPage.tsx` replaces the search params with the filters used
- [x] T044 [US4] Frontend/E2E (F2) — `frontend/e2e/qa-findings-dashboards.spec.ts`: `?currency=XXX`, `?currency=JPY`,
  `/regional?team=T-9999` show a dashboard with filters; axe (after T041, T043)

---

## Phase 7: User Story 5 — Deleting a purchase request returns with Undo (P1)

**Goal**: FR-013 (H-3, L-3). **Independent Test**: spec US5.

- [x] T050 [P] [US5] F1 — Failing test in `features/purchase-requests/PurchaseRequestPage.test.tsx`: delete from the
  full page → Purchase requests page, toast "Purchase request '<name>' deleted" with Undo; MSW fails on any GET of
  the deleted request — `038 FR-013 (H-3)`
- [x] T051 [US5] F1 — Fix (after T050): `DeletePurchaseRequestDialog.tsx` and the purchase-request delete mutation:
  `await mutateAsync()`, `removeQueries` for the record, navigate (Purchase requests, else the line item), `afterDelete`
  (`ui/useUndoableDelete.ts`), then invalidate lists
- [x] T052 [P] [US5] F4 — Failing tests: `features/plans/usePlanActions.delete.test.tsx` and the line-item page delete
  test: no request for the deleted record after delete — `038 FR-013 (L-3)`
- [x] T053 [US5] F4 — Fix (after T052): `features/plans/api.ts`, `features/line-items/api.ts` remove the record's
  queries, navigate, then invalidate lists
- [x] T054 [US5] Frontend/E2E (F1) — `frontend/e2e/qa-findings-pr-delete.spec.ts`: delete PR → toast → Undo restores;
  no 404 for it in the console; axe (after T051)

---

## Phase 8: User Story 6 — Sign-in and every page are hardened (P1)

**Goal**: FR-014 – FR-017, FR-011 (security checks, L-14, M-3). **Independent Test**: spec US6.

- [x] T060 [P] [US6] B3 — Failing tests `auth/attempts_test.go` `TestAttempts_038_FR014` with an injected clock:
  5 per e-mail and 20 per client in 15 min; lock 15 min from the limiting failure; refused attempts do not extend it;
  success clears both keys; e-mail case and spaces ignored; unknown e-mails count alike
- [x] T061 [P] [US6] B3 — Failing contract tests in `auth/handler_test.go`: 6th attempt → 429 `too_many_attempts`
  with `Retry-After`, also with the right password; message "Too many sign-in attempts. Try again in 15 minutes.";
  a NUL in the e-mail → 400 `validation_failed` (M-3)
- [x] T062 [US6] B3 — Fix (after T060, T061): `auth/attempts.go`; `auth/service.go` `SignIn` checks the lock before
  credentials and before `tx.Write`; `app/transport.go` trusts only `127.0.0.1`, `::1` and `TRUSTED_PROXIES`
  (`app/config.go`, validated at startup); one warning log per lock without the e-mail
- [x] T063 [P] [US6] B3 — Failing tests in `platform/web/web_test.go`: security headers on every answer incl. errors
  and 404 (FR-015); `TestEngine_RefusesCrossOriginRequests` new cases: foreign `Referer` without `Origin` refused,
  same-host `Referer` and no headers allowed (FR-016, L-14); NUL in a nested body field → 400 naming
  `plannedCost.amount`; unpaired `\ud800` → 400; tab in a description allowed; NUL in a path parameter → 404
  (FR-017, M-3); body over the limit → 413 `too_large` (FR-054); fuzz test `FuzzControlScan`
- [x] T064 [US6] B3 — Fix (after T063): `platform/web/middleware.go` (`SecurityHeaders`: `Content-Security-Policy:
  default-src 'none'; frame-ancestors 'none'`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`; `SameOrigin`
  Referer rule), `platform/web/web.go` (`decode` token scan, `readBody` `*http.MaxBytesError` → `TooLarge`),
  `platform/web/engine.go` (path-parameter middleware after routing)
- [x] T065 [P] [US6] F3 — Failing test `frontend/security-headers.test.ts`: the sha256 of `index.html`'s inline
  script equals the hash in the policy; `nginx.conf`'s `add_header` lines in each location equal the header map;
  preview headers enforce, dev headers are `Content-Security-Policy-Report-Only` — `038 FR-015`
- [x] T066 [US6] F3 — Fix (after T065): `frontend/security-headers.ts` (policy of research R7, hash computed from
  `index.html`); `frontend/vite.config.ts` (`server.headers`, `preview.headers`, proxy `xfwd: true`);
  `frontend/nginx.conf` (headers with `always` in every location, `proxy_set_header X-Forwarded-For $remote_addr`,
  `client_max_body_size 8m` — FR-011); `frontend/index.html` (comment moved out of the script);
  `.gitattributes` (`frontend/index.html text eol=lf`)
- [x] T067 [US6] F3 — Verify on the built app (`pnpm run build && vite preview`) whether Radix Dialog injects a
  `<style>` element; if so add its hash or a style-only `'unsafe-inline'` recorded in a new ADR in
  `docs/architecture/adr/` (F7) (after T066)
- [x] T068 [P] [US6] F3 — Failing test then fix: `features/sign-in/SignInPage.test.tsx` shows the 429 message;
  `features/sign-in/SignInForm.tsx` shows the server message for `too_many_attempts`
- [x] T069 [US6] F3 — `frontend/e2e/support/no-csp-violations.ts`: a fixture that fails any test on a
  `securitypolicyviolation`; used by every `qa-findings-*.spec.ts` (after T066)
- [x] T070 [US6] Frontend/E2E (F3) — `frontend/e2e/security.spec.ts`: 6 wrong passwords → message; right password
  still refused; headers present on a page and on an API answer; framing refused; axe (after T062, T064, T066, T069)

---

## Phase 9: User Story 7 — Approving with line items above planned cost (P2)

**Goal**: FR-018 – FR-021 (M-18), FR-057 for Overspend control (L-16). **Independent Test**: spec US7.

- [x] T080 [P] [US7] B1 — Failing tests `overspend/service_test.go` and `overspend/handler_test.go`
  (contract-validated): `approvalOverPlannedCost` defaults to `warn`; saved by `settings.manage` only (403 otherwise);
  version conflict; one log entry "Approval over planned cost: Warn → Refuse"; never-saved → `updatedAt`/`updatedBy`
  null (`038 FR-018, FR-057 (M-18, L-16)`)
- [x] T081 [P] [US7] B1 — Failing tests `approval/rounds_decide_test.go` and `approval/view_test.go` and the
  contract test of `approveActivityLevel` in `approval/handler_test.go`: Warn → approval
  recorded with "while line items were <overage> above planned cost"; Refuse → 409 `not_allowed` with the FR-020
  message and nothing changed; Decline and Recall allowed; views list `overPlannedCost`; and
  `purchaserequest/handler_overspent_test.go` for a PR round and a change round — `038 FR-019 – FR-021 (M-18)`
- [x] T082 [US7] B1 — Fix (after T080): `backend/internal/platform/database/migrations/00026_overspend_approval.sql`
  (column `approval_over_planned_cost text NOT NULL DEFAULT 'warn' CHECK (approval_over_planned_cost IN ('warn',
  'refuse'))`), `queries/overspend.sql`, `sqlc generate`; `overspend/model.go`, `store.go`, `store_postgres.go`,
  `service.go`, `handler.go`, `kinds.go`, `overspendtest/`; `DefaultConfiguration` gives null last change
- [x] T083 [US7] B1 — Fix (after T081, T082): `approval/subject.go` (`Overages`), `approval/ports.go`
  (`ApprovalOverspend`), `approval/rounds_decide.go` gate after `decidingOn` (split to `approval/rounds_overage.go` if
  needed), `approval/view.go` + `section.go` (`overPlannedCost`, approve availability reason),
  `purchaserequest/approval_actions.go` (overages from `ActivityOverBy`), wiring in `app/approval*.go`
- [x] T084 [P] [US7] F1 — Failing tests: `features/overspend/OverspendSettingsPage.test.tsx` (second choice,
  read-only state, save, "Not changed since SP&A One was set up"), `features/approvals/ApprovalSection.test.tsx`
  (warning per activity; Approve disabled with the reason under Refuse), approve dialog in
  `features/approvals/ApprovalStatusAction.test.tsx`
- [x] T085 [US7] F1 — Fix (after T084): `features/overspend/OverspendSettingsPage.tsx`, `overspend-text.ts`,
  `api.ts`; `features/approvals/ApprovalSection.tsx`, `DecisionActions.tsx`, `ApprovalAlert.tsx`; the purchase-request
  approval section in `features/purchase-requests/`
- [x] T086 [US7] Frontend/E2E (F1) — `frontend/e2e/qa-findings-approval-overage.spec.ts`: Warn shows the warning
  and approves; Alex sets Refuse; Daniel's Approve disabled with reason; Decline works; axe (after T083, T085)

---

## Phase 10: User Story 8 — Money figures add up (P2)

**Goal**: FR-022 – FR-024 (M-1, M-2, L-2). **Independent Test**: spec US8.

- [x] T090 [P] [US8] B5 — Failing tests: new `lineitem/display_test.go` `TestGrid_AddsUp_038_FR022_M1` (A-1011
  figures in USD and EUR; rows, columns and grand total add up; row total = converted planned cost) and
  `platform/money/display_test.go` `TestDisplayParts`
- [x] T091 [US8] B5 — Fix (after T090): `platform/money/display.go` `Display.Parts`; `lineitem/display.go` `grid()`
- [x] T092 [P] [US8] B5 — Failing tests `spa/periods_test.go`: change `TestPeriods_FR017_LineItemsAboveThePlannedCost`
  (:70) to `notScheduled` null and `linesAbovePlannedCost` 10,000.00; add the netting case (one activity 5,000
  unscheduled, another 10,000 above) — `038 FR-023 (M-2)`; contract test of the SP&A answer in `spa/handler_test.go`
- [x] T093 [US8] B5 — Fix (after T092): `spa/scope.go` `placePlanned` per activity, `spa/periods.go`, `spa/model.go`
- [x] T094 [P] [US8] B5 — Failing tests `dashboard/panels_test.go` and the `allwork` budget-by-month test:
  `negativeBalanceCount` — `038 FR-024 (L-2)`; contract tests of the dashboard and budget-by-month answers in
  `dashboard/handler_test.go` and the `allwork` handler test
- [x] T095 [US8] B5 — Fix (after T094): `dashboard/months.go`, `dashboard/model.go`, `allwork/budget_by_month.go`
- [x] T096 [P] [US8] F2 — Failing tests: `features/spa/SpaPeriods.test.tsx` (row "Line items above planned cost", no
  negative Not yet scheduled), `features/all-work/BudgetView.test.tsx` and `features/dashboards/MonthlyChart` test
  (note of FR-024)
- [x] T097 [US8] F2 — Fix (after T096): `features/spa/SpaPeriods.tsx`, `spa-text.ts`, `features/all-work/BudgetView.tsx`,
  `features/dashboards/MonthlyChart.tsx`
- [x] T098 [P] [US8] F4 — Failing test then fix: `features/plans/phasing/BudgetByMonthCard.test.tsx` note from
  `negativeBalanceCount`; `BudgetByMonthCard.tsx`

---

## Phase 11: User Story 9 — Purchase-request screens say what will happen (P2)

**Goal**: FR-025 – FR-028 (M-7 – M-10). **Independent Test**: spec US9.

- [x] T100 [P] [US9] B1 — Failing test `purchaserequest/access_test.go` `TestRequisitionerReason_038_FR025_M7`: a
  person without `purchaseRequests.edit` on an approved PR gets the permission reason, not "Edit the purchase
  request and choose Submit changes"
- [x] T101 [US9] B1 — Fix (after T100): `purchaserequest/access.go:59-60`, `approval_actions.go:93` check the
  permission first
- [x] T102 [P] [US9] B1 — Failing contract test `purchaserequest/handler_approval_test.go`
  `TestSubmission_NamesRecordFieldApprover_038_FR027_M9`: `outcomes` names the MoA approver
- [x] T103 [US9] B1 — Fix (after T102): `approval/subject_api.go` `SubmissionOf` reusing `submitCheck` and
  `submissionOutcomes`; `purchaserequest/approval_actions.go` `Submission` fills `Outcomes`
- [x] T104 [P] [US9] F1 — Failing tests: `PurchaseRequestPage.test.tsx` (Viewer: one View only note, no disabled change
  actions — M-7), `PurchaseRequestEditPage.test.tsx` (counter includes needed attributes — M-8; Submit changes opens a
  confirmation before saving — M-10), `SubmitPurchaseRequestDialog.test.tsx` (names the approver from `outcomes` — M-9); test names cite 038 FR-025 – FR-028
- [x] T105 [US9] F1 — Fix (after T104): `PurchaseRequestHeader.tsx`, `PurchaseRequestPage.tsx` (`ViewOnlyNote`,
  `changesNothingIn`), `form-progress.ts` (attribute sections), `PurchaseRequestForm.tsx` + `PreviewDialog.tsx`
  (confirm step), `SubmitPurchaseRequestDialog.tsx` — FR-025, FR-026, FR-027, FR-028

---

## Phase 12: User Story 10 — Screens follow access and keep work visible (P2)

**Goal**: FR-029 – FR-031 (M-6, M-12, M-4, L-28). **Independent Test**: spec US10.

- [x] T110 [P] [US10] F4 — Failing test `features/plans/phasing/BudgetByMonthCard.test.tsx` and
  `PlanPage.access.test.tsx`: no "Edit budget by month" for a Viewer — `038 FR-029 (M-6)` (after T098, same file)
- [x] T111 [US10] F4 — Fix (after T110): `BudgetByMonthCard.tsx` takes `access` and gates with `allows()`;
  `PlanPage.tsx`, `features/plans/campaigns/RecordDetails.tsx` pass it
- [x] T112 [P] [US10] F4 — Failing tests `features/global-budgets/schemas.test.ts` and
  `features/settings/shell-attributes/schemas.test.ts`: every problem at the first Save — `038 FR-030 (M-12)`
- [x] T113 [US10] F4 — Fix (after T112): both `schemas.ts` (name and owner checks; help-text length, duplicate label,
  no early return)
- [x] T114 [P] [US10] F3 — Failing tests `app/router.session-lost.test.tsx`: a dirty form when a 401 arrives → sign-in
  page says "Your session ended, so your changes in '<title>' weren't saved."; return address keeps `#fragment` —
  `038 FR-031 (M-4, L-28)`
- [x] T115 [US10] F3 — Fix (after T114): `ui/useUnsavedChanges.tsx`, `session/SessionProvider.tsx`
  (`lostChanges`, `page.hash` in `sessionLostAddress`), `features/sign-in/SignInPage.tsx`

---

## Phase 13: User Story 11 — Home, All work and exports agree (P2)

**Goal**: FR-032, FR-033 (M-5, M-11). **Independent Test**: spec US11.

- [x] T120 [P] [US11] B5 — Failing tests `home/summary_test.go`: `yearFrom`/`yearTo` = earliest owned plan year and
  at most 4 more; count within them — `038 FR-032 (M-5)`; contract test of `HomeActivityCounts` in
  `home/handler_test.go`; if the home store read changes, a scope-isolation test in
  `home/store_postgres_test.go`
- [x] T121 [US11] B5 — Fix (after T120): `home/model.go`, `home/summary.go`, `home/sources.go` (and its store query)
- [x] T122 [P] [US11] F2 — Failing test then fix: `features/home/SummaryCards.test.tsx` link carries
  `yearFrom`/`yearTo` and the card names the years ("in 2026–2027"); `features/home/home-links.ts`, `SummaryCards.tsx`
- [x] T123 [P] [US11] B4 — Failing test `allwork/export_test.go` status "Partly approved" — `038 FR-033 (M-11)`, then
  fix `allwork/status.go`
- [x] T124 [P] [US11] F2 — Failing test `features/all-work/status-words.test.ts` (containers keep the API's word),
  then `status-words.ts` becomes a pass-through for containers (after T123)

---

## Phase 14: User Story 12 — Imports say exactly what is wrong (P2)

**Goal**: FR-034, FR-035 (M-13, L-21, L-13). **Independent Test**: spec US12.

- [x] T130 [P] [US12] B4 — Failing tests `settings/ratefeed_test.go`, `settings/rates_import_test.go`: stray-quote
  row listed as a problem; empty file "This file is empty."; all missing columns named; 1000001 → too-large
  message; date 1999-12-31 refused, 2000-01-01 accepted — `038 FR-034 (M-13, L-21)`; extend the fuzz test of the
  rate file reader with stray quotes and empty input (B9)
- [x] T131 [US12] B4 — Fix (after T130): `settings/ratefeed.go` (`LazyQuotes`, empty file, all columns),
  `settings/rates_import.go` (`ErrOutOfRange`, minimum date), `settings/rates.go`
- [x] T132 [P] [US12] B2 — Failing test `financials/feed_csv_test.go` semicolon extract — `038 FR-035 (L-13)`; the
  GSAP extract fuzz test gains semicolon-separated seeds (B9)
- [x] T133 [US12] B2 — Fix (after T132): export `sheet.DelimiterOf` in `platform/sheet/csv_read.go`; use it in
  `financials/feed_csv.go` `readRecords`

---

## Phase 15: User Story 13 — Layout and accessibility hold everywhere (P2)

**Goal**: FR-036 – FR-038 (M-14 – M-17). **Independent Test**: spec US13.

- [x] T140 [US13] Frontend/E2E (F4) — Failing e2e `frontend/e2e/qa-findings-layout.spec.ts`: 120-character names;
  Plans list at 390, 768, 1440 px and SP&A view at 390 px have `scrollWidth ≤ innerWidth`; axe (light, dark) on the
  import problems table, split links, Regional dashboard empty state and PR form at 390 px — `038 FR-036 – FR-038`
- [x] T141 [P] [US13] F4 — Failing component test `features/workbook/ImportPage.test.tsx` (problems table is a focusable
  labelled region); fixes `features/workbook/CheckResult.tsx` (M-16), `features/plans/PlanCard.module.css` (M-14),
  `features/activities/SplitLinks.module.css` (M-17) — FR-037 (after T140)
- [x] T142 [P] [US13] F2 — Fixes `features/spa/Spa.module.css` (M-15) and the dashboards' empty-state hint on grey
  (`features/dashboards/DashboardsPage.module.css`, `--ink-2: var(--ink)`) (M-17) (after T140)
- [x] T143 [P] [US13] F3 — Fix `ui/FormRail/FormRail.module.css` `.progress` on grey (M-17) (after T140)

---

## Phase 16: User Story 14 — Wording and small interactions are consistent (P3)

**Goal**: FR-039 – FR-053 (L-1, L-4, L-6, L-7, L-9, L-12, L-17, L-19, L-20, L-22 – L-25, L-27, L-29).

- [x] T150 [P] [US14] F4 — Failing test `features/activities/ActivityEditPage.test.tsx` conflict next step, then fix
  `useActivitySave.ts` with `CONFLICT_NEXT_STEP` — FR-039 (L-1)
- [x] T151 [P] [US14] F4 — Failing tests `line-items/SpreadEditor.test.tsx`, `plans/phasing/BudgetByMonthCard.test.tsx`
  (summary names the month), then fix `line-items/line-item-requests.ts`, `plans/phasing/PhasingEditor.tsx` — FR-040
  (L-4) (after T111)
- [x] T152 [P] [US14] F4 — Failing test `activities/useActivityActions.test.tsx` "Move to…", then fix
  `useActivityActions.tsx`, `activity-tree-rows.tsx` — FR-041 (L-6)
- [x] T153 [P] [US14] F3 — Failing property test (`fast-check`) `format/decimal.test.ts`: leading currency symbol or
  code and thousands separators accepted; then fix `format/decimal.ts` `parseMoneyInput` — FR-042 (L-7)
- [x] T154 [US14] F4 — Failing test `line-items/SpreadEditor.test.tsx` paste "1000⇥2000⇥3000" fills three months,
  extra values reported, with a `fast-check` property for the clipboard split (F10: untrusted input); fix
  `SpreadEditor.tsx`, `plans/phasing/CustomMonths.tsx` — FR-042 (L-7) (after T151, T153)
- [x] T155 [P] [US14] F4 — New failing test `features/bulk/SubmitDialog.test.tsx` "Submit 1 activity", then fix
  `SubmitDialog.tsx` with `plural()` — FR-043 (L-9)
- [x] T156 [P] [US14] F3 — Failing tests `format/intl.test.ts` (999,999,999,999,999.99 never "1000T") and
  `ui/ConfirmDialog/ConfirmDialog.test.tsx` (`acknowledgeOnly` shows one Close), then fix `format/intl.ts`
  `formatMoneyCompact`, `ui/ConfirmDialog/ConfirmDialog.tsx` — FR-044 (L-12)
- [x] T157 [US14] F4 — Failing tests `global-budgets/GlobalBudgetPage.allocations.test.tsx` ("26%"),
  `GlobalBudgetPage.test.tsx` (one Close), then fix `global-budgets-text.ts`, `GlobalBudgetPage.tsx`; same dialog in
  `approval-workflows/ApprovalWorkflowsPage.tsx` — FR-044 (L-12) (after T156)
- [x] T158 [P] [US14] F4 — Failing test `people/access/access-words.test.ts` (four purchase-request labels of
  `platform/authz/catalogue.go:78-81`), then fix `access-words.ts` — FR-045 (L-17)
- [x] T159 [P] [US14] F3 — Remove the stale permission table from `session/access.ts` in favour of
  `features/people/access/access-words.ts` (or align it), with its test — FR-045 (L-17)
- [x] T160 [P] [US14] F3 — Failing tests `ui/MoneyField/MoneyField.test.tsx` (note row reserved) and
  `ui/FormPanel/FormPanel.test.tsx` (Enter in a select submits), then fix `ui/FieldShell/FieldShell.tsx`,
  `ui/FormPanel/FormPanel.tsx` — FR-046, FR-047 (L-19, L-20)
- [x] T161 [P] [US14] F4 — Failing test `settings/currencies/ImportRatesPage.test.tsx` (groups expose expanded state
  and a marker), then fix `OutcomeGroup.tsx`, `rates.module.css` — FR-048 (L-22)
- [x] T162 [P] [US14] B5 — Failing test in `history` bin preview tests: a rename and a status reset appear once;
  then fix `history/bin.go`, `activity/kinds.go` — FR-049 (L-23)
- [x] T163 [P] [US14] F4 — Failing tests `recycle-bin/RestoreDialog.test.tsx` (toast "— View"),
  `recycle-bin/BinRow` test (one tag style), then fix `RestoreDialog.tsx`, `restore-text.ts`, `BinRow.tsx` — FR-049
- [x] T164 [P] [US14] F3 — Failing test `ui/Toast/Toast.test.tsx` (a toast with an action reads "<message> — <action>"),
  then fix `ui/Toast/Toast.tsx` — FR-049 (L-23)
- [x] T165 [P] [US14] F1 — Failing test `PurchaseRequestPage.test.tsx` breadcrumb on a missing request, then fix
  `PurchaseRequestPage.tsx:46,93-94` — FR-050 (L-24) (after T105, same file)
- [x] T166 [P] [US14] F2 — Failing test `home/GettingStartedCard.test.tsx` (Viewer without team: no Settings step,
  "ask an administrator"), then fix `GettingStartedCard.tsx` — FR-051 (L-25)
- [x] T167 [P] [US14] F4 — Failing test `dev/SampleRecordsPage.delete.test.tsx` (no Restore for a Viewer), then fix
  `dev/RecentlyDeletedCard.tsx` — FR-052 (L-27)
- [x] T168 [P] [US14] F2 — Failing test `dashboards/DashboardPanels.test.tsx` (count 0 → "No purchase requests on these
  plans yet."), then fix `PurchaseRequestsPanel.tsx` — FR-053 (L-29)

---

## Phase 17: User Story 15 — The server answers precisely (P3)

**Goal**: FR-054 – FR-057 (L-5, L-11, L-15, L-16).

- [x] T170 [P] [US15] B3 — Failing handler test in `campaign` (create and update with `startDate` "2026-02-30" →
  `validation_failed` "Enter a valid date." on `startDate`), then fix `campaign/model.go` (string dates) and
  `campaign/validate.go` — FR-054 (L-5 a)
- [x] T171 [P] [US15] B3 — Failing unit test `platform/record` `TestCheckSingleLine_038_FR055_L11` (tab, LF, CR,
  U+202A–U+202E, U+2066–U+2069, U+200E/F refused; ordinary text and emoji accepted), then add
  `record.CheckSingleLine`
- [x] T172 [US15] B3 — Failing name tests per package, then call `CheckSingleLine` from `plan/validate.go`,
  `campaign/validate.go`, `activity/fields.go`, `lineitem/validate.go`, `budget/validate.go`, Settings list name
  checks (`settings/attributes_validate.go` and the other `settings` name checks) — FR-055 (after T171)
- [x] T173 [P] [US15] B1 — Failing test then call `CheckSingleLine` in `purchaserequest` name check
  (`purchaserequest/messages.go` / validation) — FR-055 (after T171)
- [x] T174 [P] [US15] B2 — Failing test then call `CheckSingleLine` for team names in `people/teams.go` — FR-055
  (after T171)
- [x] T175 [P] [US15] B4 — Failing test in `workbook/apply_test.go` (a name cell with a tab is a problem of its row),
  then the workbook parser applies `CheckSingleLine` — FR-055 (after T171, T032)
- [x] T176 [P] [US15] B1 — Failing test `approval/handler_roles_test.go` (Viewer approving a decided round → 403
  `forbidden`, no name), then fix `approval/rounds_decide.go` (permission before `roundToDecide`) — FR-056 (L-15)
- [x] T177 [P] [US15] B1 — Failing test then fix: `PurchaseRequestConfiguration` never saved → null last change
  (`purchaserequest/model_routing.go:60`, its handler test) — FR-057 (L-16)
- [x] T178 [P] [US15] F1 — Failing test then fix `PurchaseRequestSettingsPage` (in `features/purchase-requests/`)
  "Not changed since SP&A One was set up" — FR-057 (L-16)

---

## Phase 18: User Story 16 — The person's date format is used everywhere (P3)

**Goal**: FR-058 (L-18).

- [x] T180 [P] [US16] F3 — Failing tests `format/intl.test.ts` (`formatWeekdayDate(date, format)` and
  `formatMonth('YYYY-MM', format)` for the four formats of FR-058's table), `features/activity-log/log-text.test.ts`,
  `LogEntryList.test.tsx`, `HistorySection.test.tsx`
- [x] T181 [US16] F3 — Fix (after T180): `format/intl.ts`, `format/useFormatters.ts` (`weekdayDate`, `month`),
  `features/activity-log/log-text.ts` `dayLabel(date, today, format)`, `LogEntryList.tsx`, `HistorySection.tsx`
- [x] T182 [US16] F3 — Failing tests then replace the six month-label helpers of research R18 with `month`:
  `features/financials/financials-text.ts`, `features/home/financials-text.ts`, `features/plans/RecordFinancials.tsx`,
  `features/line-items/line-item-math.ts` `formatMonth` (callers unchanged), `features/plans/phasing/phasing-math.ts`,
  `features/dashboards/dashboard-text.ts` (also fixes "Sept"); tests `financials-text.test.ts`,
  `line-item-math.test.ts`, `phasing-math.test.ts`, `FinancialsCard.test.tsx`, `RecordFinancials.test.tsx` (after T181)

---

## Phase 19: User Story 17 — Exports and demo data are coherent (P3)

**Goal**: FR-059 – FR-061 (L-10, L-8, L-30).

- [x] T190 [P] [US17] B4 — Failing tests: workbook export writes budget and planned cost as numbers; unchanged
  export re-imported → `nothingToChange` — FR-059 (L-10); then fix `workbook/build.go` (`sheet.Money`) and
  `workbook/apply.go` `readCells` (compare parsed amounts) (after T032)
- [x] T191 [P] [US17] B5 — Failing tests `backend/seed/history_test.go` (created meta = first history entry) and
  `backend/seed/settings_test.go`, `lineitems_test.go` (FY2027 finance budget items; FY2027 line items carry an item;
  key built from the line item's year), then fix `backend/seed/seed.go`, `history_planning.go`, `settings.go`,
  `settings_records.go`, `lineitems.go:132`; update counts in `seed/seed.go:26` and `docs/product/demo-users-and-data.md`
  — FR-060, FR-061 (L-8, L-30)

---

## Phase 21: User Story 18 — Mandatory fields show a red asterisk (P3)

- [x] T205 [US18] F3 — Failing test `ui/FieldShell/FieldShell.test.tsx`: a field marked Required shows a `*` after its
  label in `--danger`, bold, at least 19 px, `aria-hidden`; the control is `aria-required` and described "required" — FR-063
- [x] T206 [US18] F3 — "Needed to publish" and "Needed to submit" keep their word tags, no asterisk — FR-063
- [x] T207 [US18] F3 — Failing test `ui/RequiredLegend/RequiredLegend.test.tsx`; "* Required" legend once per form
  that has a required field (FormErrorSummary, FormDialog, ConfirmDialog) — FR-064
- [x] T208 [US18] F3 — Design system FieldShell row and token `--text-required-mark`; e2e/axe colour-contrast pass — SC-013

## Phase 20: Polish & cross-cutting

- [x] T200 Docs (D) — Domain model §4.4 note on the grid's conversion (FR-022) and §4.3 approval rule (FR-019 –
  FR-021); design system: approve warning pattern, `acknowledgeOnly` dialog, rounding note; glossary "Approval over
  planned cost"
- [x] T201 Docs (D) — `docs/sdd/feature-map.md` row 038 → "Built" when every task is done; release note in
  quickstart (built-in planners lose "Link financial transactions", D-353)
- [ ] T202 Gates — backend `golangci-lint fmt ./...`, `golangci-lint run ./...`, `sqlc generate && sqlc vet` (no diff),
  `go vet ./...`, `go test -race ./...` with `TEST_DATABASE_URL`, `govulncheck ./...`; frontend `pnpm run check`,
  `pnpm run build`, full `pnpm run test:e2e`; comment budget (≤ 5 %) and file length (≤ 300 non-blank lines) of every
  touched file; 80 % decision coverage of changed code
- [ ] T203 QA re-run — reset the demo data (`go run ./cmd/seed -reset`), start API and preview, and re-run QA's
  reproduction scripts (`…/scratchpad/qa/area1` – `area4`, `qa/verify`, `qa/smoke`) per [quickstart.md](quickstart.md);
  record per finding ID "no longer reproduces" or the remaining difference in a table appended to this file; any
  remaining item becomes a new task (SC-012)
- [ ] T204 `/speckit-converge` — compare code with spec, plan and tasks; append remaining work or "clean"

---

## Dependencies & Execution Order

- **Phase 1** (T001 – T003) blocks everything; T004 is done.
- **Lanes** run in parallel after Phase 1. Inside a lane, follow task order; each fix depends on its test.
- **Cross-lane dependencies** (the only ones):
  - T034 (workbook e2e) needs T032 (B4) and T066 (F3, nginx body limit).
  - T124 (F2 status words) after T123 (B4).
  - T172 – T175 (name checks in B3, B1, B2, B4) after T171 (B3 helper).
  - T157 (F4) after T156 (F3, `ConfirmDialog`); T154 (F4) after T153 (F3, `parseMoneyInput`); T150 after T003.
  - E2E tasks after the fixes they cover: T013, T028, T034, T044, T054, T070, T086, T140 (written first, fails until
    T141 – T143).
  - T069 (no-CSP-violation fixture) before the other e2e specs are finalised.
- **Stories** are independent of each other except US14's T151/T165 sharing files with US10/US9 (same lane, ordered).

### Lane work queues (for parallel implementers)

| Lane | Tasks |
|---|---|
| B1 | T010 – T012, T080 – T083, T100 – T103, T173, T176, T177 |
| B2 | T020 – T024, T132, T133, T174 |
| B3 | T060 – T064, T170 – T172 |
| B4 | T030 – T033, T123, T130, T131, T175, T190 |
| B5 | T040, T041, T090 – T095, T120, T121, T162, T191 |
| F1 | T013, T050, T051, T054, T084 – T086, T104, T105, T165, T178 |
| F2 | T025, T026, T028, T042 – T044, T096, T097, T122, T124, T142, T166, T168 |
| F3 | T003, T065 – T070, T114, T115, T143, T153, T156, T159, T160, T164, T180 – T182 |
| F4 | T027, T034, T052, T053, T098, T110 – T113, T140, T141, T150 – T152, T154, T155, T157, T158, T161, T163, T167 |
| D | T004, T200, T201 |
| Any | T001, T002 (Phase 1), T202 – T204 (end) |

## Parallel Example: Phase 3 + Phase 4 + Phase 8 at once

```text
B1: T010, T011 → T012          B2: T020, T021, T022 → T023, T024      B3: T060, T061, T063 → T062, T064
F2: T025 → T026                F3: T065 → T066 → T067, T069           F1: T050 → T051
```

## Implementation Strategy

1. **MVP**: Phase 1, then US1 (C-1) — the only critical finding; demo-able alone.
2. **P1 set**: US2 – US6 in parallel lanes; each closes with its e2e journey.
3. **P2** (US7 – US13), then **P3** (US14 – US17); each story is independently testable.
4. Finish with gates (T202), the QA re-run (T203) and converge (T204).

## Notes

- `backend/api/openapi.yaml` and `schema.d.ts` change only in T001; a lane needing another contract change stops and
  raises it rather than editing the contract.
- Migrations are numbered in advance (00026 B1, 00027 B2) so lanes cannot collide.
- A test that pinned the old behaviour (`TestAccess_NoImportHidesNotLinked_FR028`, `dashboard/handler_test.go`
  400 cases, `TestPeriods_FR017_LineItemsAboveThePlannedCost`, `web_test.go` "larger than 1 MB" 400, export "In
  progress" assertions) is changed in the failing-test task of its fix, with the 038 requirement in its name.
