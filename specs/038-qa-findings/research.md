# Research: QA Findings (038)

Every finding was traced in the code of `develop@85a3d495`. Paths are relative to `backend/internal/` for Go and
`frontend/src/` for the web app unless they start with `backend/`, `frontend/` or `specs/`. Line numbers are those
of `85a3d495`. "Tests" names the file where the failing test of each fix goes (constitution VI).

Statement counts of H-2 come from QA's API log (`querycount`), taken on the same laptop class the plan targets;
they were not re-measured here because the workbook has no benchmark yet — T030 adds one and records the
before/after numbers in this file (028 FR-008 practice).

## R1 — C-1: reservation while a purchase-request change is in review

**Root cause.** A change overwrites the lines: `Service.saved` → `Service.store` (`purchaserequest/save.go:31,81`)
puts the new line amounts, then `submitAfterSave` opens the modification round. The approved amounts live only in
`purchase_requests.approved_version` (`PurchaseRequest.Approved *Version`, `purchaserequest/model.go:110`;
`Version.approvedLine`, `:169`). The claim query `ListPurchaseRequestClaims`
(`backend/internal/platform/database/queries/purchase_requests.sql:23`) → `PostgresStore.Claims`
(`purchaserequest/store_postgres.go:114`) sums **current lines only**, so a lowered pending amount stops being
reserved at once. Decline or recall runs `modificationEvent` → `Service.restore` (`purchaserequest/rounds.go:152,
173`), which writes the approved amounts back with no room check — hence requested 19,600 / available −3,800.
032 FR-017 ("released only when … an approved modification lowers it") and FR-037 ("reserve the modified
amounts") disagree for a lowering change; FR-017 is the intent.

**Decision.** A claim carries the approved amount of its line when the request has `change_pending`; the balance
counts `max(current, approved)` per line, both converted with the current rate (D-303). A line removed by the change
keeps a claim of its approved amount (read from the approved version). Every reader already goes through
`balance.requested` / `available` / `claimProblem` (`purchaserequest/balance.go:15,32,40`, loaded by `balanceOf`,
`defaults.go:80`): line-item and activity sections (`sections.go:38,114`), options (`options.go:80`), guards
(`guards.go:15`), the save check (`validate.go:64` → `validate_lines.go:146`) and restore (`delete.go:77,102`). So
the fix is in one query, one store mapping and one function, mirrored in the fake
(`purchaserequest/purchaserequesttest/store.go:177`). `restore` then can never overrun.

**Alternatives.** Keep pending lines in a separate table — larger change, two sources of truth. Refuse lowering
changes — removes a 032 capability.

**Tests.** `purchaserequest/balance_test.go` (rule, table-driven: lowered, raised, added, removed line, rate
change); `purchaserequest/handler_change_test.go` (QA sequence through `apitest`, contract-validated). The
changed `Claims` read is not a scoped read — it takes line-item IDs the service has already resolved in the actor's
scope and serves a rule — so it gets no database test (constitution VI forbids query tests); the end-to-end journey
runs it on PostgreSQL.

## R2 — H-4 and H-5: financial transactions, scope and the link permission

**Root cause (H-5).** `Service.Link` / `Unlink` (`financials/service.go:62,128`) load the whole book unscoped
(`s.load`, `financials/book.go:27`) and find the transaction in Go, then:
1. `Link` never calls `mayOpen` (`financials/transaction.go:44`): it checks only `access.Link(ctx, a, "")` and the
   target plan — Mei links FT-1004, which she cannot open.
2. `Unlink` runs `CheckVersion` and the "manual links only" rule before `access.Link(ctx, a, planID)` → 409 instead
   of 404.
3. `record.Meta.CheckVersion` (`platform/record/record.go:48`) puts `UpdatedBy.Name` in the message before any
   scope check; `Link` also reveals `msgLinkedByPerson`.
4. `financialAccess.Link` (`app/financials.go:114`) uses `authz.RequireIn` without `RequireView`, so an out-of-scope
   line item answers 403 while a missing one answers 400 (`SeeFinanceIn`, `:95`, does call `RequireView`).

**Root cause (H-4).** Not-linked rows are filtered in Go: `scopeOf` (`financials/overview.go:18`) sets `unlinked =
access.Import(...) == nil`; `placed.inFilters` (`financials/list.go:124`) drops them otherwise; `mayOpen` checks
`Import` for them (plain `Require`, so 403 not 404). Pinned by `TestAccess_NoImportHidesNotLinked_FR028`
(`financials/service_test.go:121`) and written in `specs/015-financial-transactions/plan.md:236`. Spec Q-015c and
014 Table B give `finance.link` to Global planner and Planner (`platform/authz/grants_test.go:21-28`, migration
`00007_access.sql:11`), who can never see a row to link. In the web app `TransactionPanel.tsx:73-83` shows Link and
Unlink from the link state only and ignores `canLink` / `canUnlink` (`financials/transaction.go:109-110`).

**Decision (D-359).** Keep visibility as built (product owner). Order in `Link` and `Unlink`: find → `mayOpen` with
not-linked rows answering **404** (hidden, `NotFound(msgTransactionGone)`) for people without import → permission
→ `CheckVersion` → state rules. `financialAccess.Link` adds `RequireView` and maps "hidden" to the same
`missingLineItem` field problem as a missing ID (`orGone`). Link needs `finance.import` (to see) and `finance.link`
in the target plan; Unlink needs `finance.link` in the current line item's plan. Built-in Global planner and Planner
lose `finance.link` (migration 00027, seed, `grants_test.go`); the role service refuses a role with `finance.link`
without `finance.import` (field `permissions`), and the access check treats `finance.link` without import as not
held, so an existing custom role grants nothing extra. The panel gates on `canLink`/`canUnlink`; the tabs drop Not
linked when `counts.notLinked` is absent.

**Alternatives.** Let `finance.link` holders see not-linked rows (rejected by the product owner); keep planners'
`finance.link` for unlinking only (a permission whose use is invisible after one click — the transaction vanishes
from their view).

**Tests.** `financials/service_test.go` (order, 404 parity, no name in messages), `financials/handler_test.go`
(contract: identical bodies for missing and out-of-scope), `access/roles_test.go` for the role rule,
`platform/authz/grants_test.go`; frontend `TransactionPanel.test.tsx`, `FinancialsPage.test.tsx`,
`home/FinancialsCard.test.tsx`. Scope isolation: the book read is not scoped in SQL today (filtered in Go, 015
design); `financials/store_postgres_test.go:92` stays; no scoped store read changes.

## R3 — H-2: workbook check at 5,000 rows

**Root cause.** `workbook/service.go:71 Check` and `:88 Import` → `apply` (`:109`) → `apply.go:68 run` → `:155
save` → `:185 write`, which calls `Planning.Create`/`Update` per ready row. For an activity row
(`app/workbook_create.go:70 createActivity`): `permit` (`app/planning_edits.go:35`), then `activities.NewForm`
(`activity/form.go:14`: plan context, business units, activity types, rates, workflows and each approver, attributes,
siblings, and `overspendLimits` → `overspend/service.go:106 Limits`, which reads **every budget** and rates), then
`activities.Create` (`activity/create.go:16`) repeats context and checks — `checkName` re-reads siblings
(`activity/fields.go:64`, quadratic) — then ID, put, log, view. The transaction memo (`platform/memo`, 028 FR-006)
is cleared after each put, so reference reads repeat on every row: ≈18 statements per row (QA: 250 rows 4,534;
1,000 rows 18,034; 5,000 rows 78,325 and 500 "select budgets with their planned cost: context canceled"). `Check`
runs inside `s.tx.Write`, which takes `pg_advisory_xact_lock(hashtext('mapp.write'))`
(`platform/database/database.go:63`); file parsing (`readUpload`) also runs under it. Problems keep the full value
(`workbook/apply.go:99 problem()`, `workbook/model.go:39`). Side finding: `frontend/nginx.conf` limits bodies to
4 MB while the API accepts 7 MB uploads (`workbook/read.go:17-19`, `app/transport.go:30-31`).

**Decision.** (1) Parse the file before `Write`. (2) A workbook-level preparation reads once per check what
`Plan()` and `Choices` do not already hold — activity types, people, workflows with approvers, attributes,
business-unit options, siblings by parent — and passes them to new service entry points
`activity.Service.CreateFromImport` / `UpdateFromImport` (and the campaign equivalents) that take this prepared
context instead of rebuilding it; they keep calling the same rule functions (`checkFields`), so every message stays
the same. (3) No `NewForm` and no `overspendLimits` on the import path (037 FR-012: imports are never refused for
overspend). (4) Sibling names held in memory and updated as rows are created. (5) `problem()` truncates to 200
runes + "…". (6) nginx `client_max_body_size 8m`. Target: ≤ 4 statements per row (ID, put, log, plus at most one)
and ≤ 150 fixed (SC-002).

**Alternatives.** Bulk inserts of all rows (larger rewrite of history logging); a background job (new
infrastructure, not needed at this size).

**Tests.** `platform/apibench`: `BenchmarkWorkbookCheck` (250, 1,000, 5,000 rows) and a test that the statements of
a 100-row check minus those of a 10-row check are ≤ 4 × 90 (028's method; PostgreSQL, skipped under `-short`);
`workbook/apply_test.go` and `handler_import_test.go` keep proving identical problems; `workbook/apply_test.go`
for the 200-character cap.

**Measured** (`BenchmarkWorkbookCheck`, demo plan P-1001, generated rows: one campaign, an
initiative per 100 rows, half activities, half line items; developer laptop shared with other builds, so the
times are high):

| | 10 rows | 100 rows | 100 − 10 | 1,000 rows | 1,000 rows time |
|---|---|---|---|---|---|
| Before (develop) | 169 | 1,429 | 1,260 (14/row) | 14,006 | 6 min 14 s |
| After (T032) | — | — | ≤ 360 (test passes) | 2,152 | 23 s (machine under load) |
| After (T034 run) | — | — | ≤ 360 (test passes) | 2,152 | 8.7 s (machine under load) |

| | 5,000 rows | 5,000 rows time |
|---|---|---|
| Before (QA) | 78,325 | canceled |
| After (`BenchmarkWorkbookCheck/5000_rows`, T034) | 10,552 | 34.2 s (machine under load) |

After T032 activities and line items are checked by `activity.Importer` / `lineitem.Importer`, which read each
parent once and store each sheet with `AddMany`, one `NextIDs` and one read-back; each row costs its log entry
(2 statements). No store read was added (only existing `ListByInitiative`, `ListByIDs`, `ListByActivity(ies)`,
`AddMany`), so T033 has no new scoped read to isolate. 5,000 rows send ≈ 2.1 statements per row, linear with
1,000 rows. The 5 s (1,000 rows) and 20 s (5,000 rows, T034) budgets are still to be confirmed on an idle machine.

## R4 — H-1: unknown dashboard currency or market

**Root cause.** The web parser (`features/dashboards/schemas.ts:16 readFilters`) checks only syntax; `XXX` and
`T-9999` are well formed and reach the API, which refuses them in `dashboard/input.go:121 chooseCurrency`
(`:134-135`) and `:143 chooseTeam` (`:146-147`) — written in 036's data model (`data-model.md:13-14`) and contract
(`contracts/api.md:10-11`) and pinned by `dashboard/handler_test.go:137-142,159`, contradicting 036 FR-007.
`DashboardsPage.tsx:83` then replaces the whole page, filter bar included.

**Decision.** Fall back in the API (only it knows the active currencies and the person's markets): an unknown code
takes the `default:` path (`c.Default`, else USD); an unknown or out-of-scope team takes the "mine, else first" path.
Syntax errors stay 400 in `ParseQuery` (the web parser already strips them). The response's `filters` echo the
values used; the page replaces the address with them (`setParams(…, {replace: true})`). Exports share `read()`.

**Tests.** `dashboard/handler_test.go` (cases change to 200 with the default), `DashboardsPage.test.tsx`.

## R5 — H-3 and L-3: delete from a record page

**Root cause.** `features/purchase-requests/DeletePurchaseRequestDialog.tsx:31-45`: the hook's `onSettled` awaits
`invalidateAllButSession`, the page's own query refetches and gets 404, `PurchaseRequestPage.tsx:38,96` swaps in the
not-found state and unmounts the dialog, and TanStack drops the per-call `onSuccess` (navigate + Undo). L-3: plan
and line-item deletes (`features/plans/api.ts:151-172`, `features/line-items/api.ts:109-127`) invalidate prefixes
that include the active record's queries before navigating.

**Decision.** The pattern of `usePlanActions.tsx:149-160`: `await mutateAsync()`, `removeQueries` for the deleted
record, navigate, `afterDelete` (`ui/useUndoableDelete.ts`), then invalidate lists. Landing page: Purchase requests
when `purchaseRequests.list` is held, else the line item.

**Tests.** `PurchaseRequestPage.test.tsx` (delete from the full page; MSW fails on an unexpected GET of the deleted
record), `usePlanActions.delete.test.tsx`, line-item page test.

## R6 — Sign-in rate limit (D-361)

**Where.** `auth/handler.go:55 signIn` → `auth/service.go:51 SignIn` → `:85 checkCredentials`; one
`IncorrectCredentials` (`auth/errs.go:138`, 401) for every failure; routes exist only with demo sign-in
(`handler.go:27`); the service has an injected clock (`Options.Now`). The engine sets `SetTrustedProxies(nil)`, so
`ClientIP` is the proxy's address.

**Decision.** A small `auth.Attempts` type (map keyed by normalised e-mail and by client, each holding failure times
within the window and a lock end; mutex; pruned on access) checked in `SignIn` before credentials and updated after.
Limits: e-mail 5, client 20, window 15 min, lock 15 min from the failure that reaches the limit. Refusal: 429
`too_many_attempts`, `Retry-After` in seconds, message "Too many sign-in attempts. Try again in 15 minutes." (minutes
rounded up from the remaining wait). Success clears both keys. The lock check runs before the credential check, so a
locked e-mail is refused even with the right password, and before `tx.Write`, so refused attempts do not take the
write lock. Client address: the engine trusts only loopback proxies (`SetTrustedProxies(["127.0.0.1","::1"])` plus
`TRUSTED_PROXIES` for compose); the Vite proxy sets `xfwd: true`; nginx sets `X-Forwarded-For $remote_addr`.
Logging: one warning per lock with the key kind, never the e-mail (constitution VII personal data).

**Alternatives.** Persist attempts in PostgreSQL (survives restarts, but a write per failure under the global lock);
exponential back-off (harder to explain on screen); CAPTCHA (no).

**Tests.** `auth/attempts_test.go` (window, limit, lock, reset, normalisation, injected clock),
`auth/handler_test.go` (contract: 429 with envelope and `Retry-After`), frontend `SignInPage.test.tsx` (message),
e2e journey.

## R7 — Security headers and CSP (D-361)

**Where.** API: `platform/web/middleware.go:63 SecurityHeaders` sets only `Cache-Control: no-store` and
`X-Content-Type-Options: nosniff`. Web: `frontend/vite.config.ts` sets no headers on `server` or `preview`;
`frontend/nginx.conf` has only `add_header Cache-Control …` in its two locations (nginx does not inherit server-level
`add_header` into a location that has its own). `frontend/index.html` has one static inline theme script (feature
001 R16) and the module `/src/main.tsx`. Fonts are self-hosted (`@fontsource/jost`), images only `favicon.svg`;
React's `style={}` uses the CSSOM (not blocked by `style-src`); Radix Dialog's scroll lock (`react-remove-scroll`)
may inject a `<style>` element — to be verified in T072 against the built app.

**Decision.**
- API answers: `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'`, `X-Frame-Options: DENY`,
  `Referrer-Policy: no-referrer`, `X-Content-Type-Options: nosniff`.
- Web pages: `default-src 'self'; script-src 'self' 'sha256-<theme script>'; style-src 'self'; img-src 'self'
  data:; font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'self'; form-action 'self';
  frame-ancestors 'none'`, `X-Frame-Options: DENY`, `Referrer-Policy: same-origin`, `X-Content-Type-Options:
  nosniff`. If T072 shows Radix injects a `<style>` element, `style-src` adds that element's hash or, failing that,
  `'unsafe-inline'` for styles only, recorded in an ADR (F7 forbids it for scripts, not styles).
- One source of the policy: `frontend/security-headers.ts` exports the header map and computes the theme script's
  hash from `index.html` at build time; `vite.config.ts` uses it for `preview.headers` (enforced) and
  `server.headers` (as `Content-Security-Policy-Report-Only`, because Vite's dev runtime injects its own inline
  preamble); a test compares `nginx.conf`'s `add_header` lines with the same map, so the two cannot drift. The
  theme script's comment lines move out of the script (the hash covers its exact bytes; `.gitattributes` keeps
  `index.html` LF).

**Alternatives.** Nonces (need a server that templates `index.html`; nginx static serving cannot); moving the theme
script to a file (a flash of the wrong theme before the module loads — the reason for 001 R16).

**Tests.** `platform/web/web_test.go` (headers on every answer), `frontend/security-headers.test.ts` (hash matches
`index.html`; nginx lines equal the map), e2e: listen for `securitypolicyviolation` and fail on any (SC-005).

## R8 — L-14: no Origin with a foreign Referer

**Root cause.** `platform/web/middleware.go:72 SameOrigin` uses `http.NewCrossOriginProtection()`, which decides
from `Sec-Fetch-Site`, then `Origin`, and lets a request with neither through without reading `Referer`.

**Decision.** For unsafe methods, when neither `Origin` nor `Sec-Fetch-Site` is present and `Referer` is present,
refuse unless its host equals the request's `Host` and is local (same `otherSiteMessage`). Mitigated today by
`SameSite=Strict`, so P1 only as part of the hardening set.

**Tests.** `platform/web/web_test.go:33 TestEngine_RefusesCrossOriginRequests` (new cases).

## R9 — M-3 and L-11: control characters at the trust boundary

**Root cause.** `web.BindJSON` (`platform/web/respond.go:14`) → `readBody`/`decode` (`platform/web/web.go:33,41`)
decodes strictly but never inspects strings; PostgreSQL then rejects 0x00 (`SQLSTATE 22021`) → 500. Path IDs are
read with `c.Param` (169 places) and reach SQL. Each package has its own `checkName` (`campaign/validate.go:100`,
`activity/fields.go:64`, `plan/validate.go`, `lineitem/validate.go`, `budget/validate.go`,
`purchaserequest/messages.go`) and none refuses control or bidi characters. Go's decoder turns a lone surrogate
escape into U+FFFD, so it can only be seen in the raw bytes.

**Decision.** In `decode`, before unmarshalling, scan the raw body with `json.Decoder.Token()` keeping the path
(`name`, `plannedCost.amount`, `recordIds.0` — the dotted convention of `bulk/request.go:18`): a string token with
U+0000–U+001F other than tab/LF/CR, U+007F, or a raw `\uD800–\uDFFF` escape not forming a pair → `errs.Validation`
with "Remove the control characters from this text." on that path. A middleware after routing (in `NewEngine`,
`platform/web/engine.go:24`) answers 404 `not_found` when any path parameter holds a control character. Single-line
names: new helper `record.CheckSingleLine(name) string` (tab, line breaks, U+202A–U+202E, U+2066–U+2069, U+200E/F)
called by each `checkName` and by the workbook parser. Tab/LF/CR stay allowed in multi-line texts by the binder.

**Alternatives.** A database-level check (500 remains); stripping silently (changes what people typed without
telling them).

**Tests.** `platform/web/web_test.go` (fuzz test of the scan, F7/B9 parsers; table of fields), contract test of
sign-in with NUL, `platform/record` unit test of the helper, one name test per package.

## R10 — M-18: approving over planned cost (D-360)

**Where.** Activity: `approval.Service.Approve` → `decideActivity` → `DecideSubject` (`approval/rounds_decide.go:13`,
`approval/subject_api.go:70`). Purchase request: `purchaserequest.Service.Decide` (`approval_actions.go:103`) →
`engine.DecideSubject`. `DecideSubject` runs `roundToDecide` → `decidingOn` (authority) → `recordDecision`, with no
cost rule. `activitySubject` (`approval/subject.go:33`) drops `ActivityFacts.UnassignedPlannedCost`
(`approval/ports.go:33`); purchase requests have `overspentActivityGates` (`purchaserequest/readiness.go:74`, from
`LineItemFacts.ActivityOverBy`, 037). The organisation setting is the singleton `overspend_configuration`
(`backend/internal/platform/database/migrations/00025_overspend.sql`; `overspend/model.go` `Configuration`,
`store.go:22`, `store_postgres.go:60,74`, `service.go:39,50,99`, `handler.go:20`; routes
`access/routes_overspend.go`; OpenAPI `OverspendConfiguration(Input)`, `backend/api/openapi.yaml:8594,8611`).

**Decision.** The subject gains `Overages []Overage` (activity ID, name, amount in plan currency) filled by each
subject kind (activity: its own facts; purchase request: the activities of its lines, from `ActivityOverBy`).
`DecideSubject`, after `decidingOn` and before `recordDecision`, asks an `ApprovalOverspend` port (overspend
service → configuration) for the mode: Refuse + overages + Approve → 409 `not_allowed` with the message of FR-020;
Warn → the decision log entry gains the overage clause. The approval views (`ActivityApproval`,
`PurchaseRequestApproval`) gain `overPlannedCost` and the approve action's `ActionAvailability` carries the refusal
reason. Column `approval_over_planned_cost` (migration 00026, default `warn` — today's behaviour, so nothing
changes until an administrator opts in, as D-347 did for budgets).

**Alternatives.** Gate only the last level (an earlier approver would approve unknowingly); always refuse (product
owner asked for both).

**Tests.** `approval/rounds_decide_test.go` (warn/refuse × activity/PR/change × decline allowed),
`overspend/service_test.go`, `overspend/handler_test.go` (contract, 403, version, log), handler tests of both
approve operations; frontend `OverspendSettingsPage.test.tsx`, approval section and Approve dialog tests; e2e.

## R11 — M-1: spread grid conversion

**Root cause.** `lineitem/display.go:75 grid()` converts each cell, row total and grand total separately with
`d.money` (its comment at `:73-74` admits a cent difference), against domain model §4.4. `money.ConvertParts(parts,
from, to)` (`platform/money/convert.go:24`) converts all parts but the last and gives the last `Convert(Sum) −
Σothers`; `money.Display` keeps its rates unexported (`platform/money/display.go:46`).

**Decision.** Add `Display.Parts(ms []Money) []Money` (same rate lookup as `Money()`, distributing with
`ConvertParts`). Per row: convert non-empty cells with `Parts` (row total = their sum = `Convert(row total)`);
column totals = sums of converted cells; grand total = sum of converted row totals. Accepted consequence: the grand
total may differ by a cent from the separately converted "Assigned" figure (D-061 allows a cent between a list and a
figure; the grid itself adds up).

**Tests.** New `lineitem/display_test.go` (USD, GBP, EUR display; property: rows and columns add up),
`platform/money/display_test.go`.

## R12 — M-2 and L-2: line items above planned cost in period figures

**Root cause.** `spa/scope.go:83 placePlanned` places every line-item month and spreads only a positive balance
(`:95`), then `notScheduled = planned − Σmonths` (`:101`) — negative, and netted across activities. Pinned by
`spa/periods_test.go:70`; shown red in `SpaPeriods.tsx:100`. All work never spreads a negative balance
(`allwork/periods.go:107`, 010 FR-023) and explains it only on the Unassigned card (`all-work-text.ts:124`,
`SummaryCards.tsx:70`, from `allwork/model.go:157 NegativeBalanceCount`).

**Decision.** Per activity: unscheduled = max(0, planned − Σ its placed months); excess = max(0, Σ months − planned).
`SpaPeriods` gains `linesAbovePlannedCost` (Money or null) and its trace `linesAbovePlannedCostTrace`. The note of FR-024
uses a count: `DashboardMonths.negativeBalanceCount`, Budget & cost from the existing summary count, Budget by month
from the record's count.

**Tests.** `spa/periods_test.go` (changed expectation), `SpaPeriods.test.tsx`, `dashboard/panels_test.go`,
`BudgetView.test.tsx`, `BudgetByMonthCard.test.tsx`.

## R13 — Purchase-request screens (M-7 – M-10, L-24)

- **M-7**: no `ViewOnlyNote` in `PurchaseRequestHeader.tsx:71-98` / `PurchaseRequestPage.tsx`; follow
  `LineItemPage.tsx:59,117` (`changesNothingIn`). The Requisitioner's text comes from
  `purchaserequest/access.go:59-60` (and `approval_actions.go:93`), which returns `msgEditToChange` for an approved
  request before checking `PurchaseRequestsEdit`; check the permission first.
- **M-8**: `features/purchase-requests/form-progress.ts:7-27` counts standard fields only; the server's list
  (`purchaserequest/readiness.go:147-165`) includes needed attributes. Add attribute sections as
  `activity-form-progress.ts:16-39` with `missingAttributes` (`ui/ShellAttributeFields/rules.ts:158`).
- **M-9**: `purchaserequest.Service.Submission` (`approval_actions.go:24`) builds levels from `SectionOf(…)
  .UpcomingLevels` → `levelView` (`approval/workflows_view.go:106`), which lists fixed `ApproverIDs` only. The
  activity check resolves approvers (`submitCheck` → `resolvedLevels` → `candidates`, `approval/resolve.go:131`,
  record-field via `subject.personIn`). Add `engine.SubmissionOf(ctx, a, kind, id)` reusing `submitCheck` and
  `submissionOutcomes`; `PurchaseRequestSubmission` gains `outcomes` (`ApprovalSubmissionLevel[]`).
- **M-10**: `PurchaseRequestForm.tsx:98-102` saves at once when `mode.then === 'submitChanges'`; open a confirm
  step (`PreviewDialog.tsx`, server preview) first.
- **L-24**: `PurchaseRequestPage.tsx:46,93-94` hard-code Plans.

## R14 — Forms, session and access gating (M-4, M-6, M-12, L-28)

- **M-4**: `ui/useUnsavedChanges.tsx:39-45` stops blocking once the person changes; `session/SessionProvider.tsx:58-74
  endSession` navigates with `flushSync`. Add a `lostChanges` title to the session-lost state, set by
  `useUnsavedChanges` when dirty; the sign-in page shows FR-031's sentence. Drafts are not stored (F7).
- **L-28**: `SessionProvider.tsx:246-249 sessionLostAddress` drops `page.hash`.
- **M-6**: `features/plans/phasing/BudgetByMonthCard.tsx:193-197` renders the button whenever a budget exists; pass
  `access` and gate with `allows(record.access?.edit)` (`session/gate-action.ts`).
- **M-12**: `features/global-budgets/schemas.ts:27-34` checks only amount; `features/settings/shell-attributes/
  schemas.ts:52-75` misses help-text length and duplicate label and returns early on options.

## R15 — Home, All work, exports (M-5, M-11, L-10)

- **M-5**: `home-links.ts:16 myActivitiesHref` sends no year; `home/summary.go:31,89 activityCounts` counts all
  plans. `HomeActivityCounts` gains `yearFrom`/`yearTo` (the earliest owned plan year and at most 4 more); the count
  covers that window; the link carries both.
- **M-11**: `allwork/status.go:21` labels `inProgress` "In progress"; the web app renames it (`status-words.ts:17`).
  Change the backend label to "Partly approved" and make `statusWord` a pass-through.
- **L-10**: `workbook/build.go:52-53 recordRow` writes `sheet.Text(format(...))`; use `sheet.Money` for budget and
  planned cost, and compare parsed amounts in `readCells` (`workbook/apply.go:128`) so a numeric cell read back is
  not "changed".

## R16 — Imports (M-13, L-21, L-13)

`settings/ratefeed.go:70 GSAPRateFile.Read` uses its own `csv.Reader` without `LazyQuotes` (`:77`) → `ErrBareQuote` →
"can't be read" (`:85,111`); empty file → `io.EOF` reported as missing `rate_type` (`:81-83`); missing columns
return at the first (`:94-97`); `rates_import.go:218,234` ignores `money.ErrOutOfRange` (use `rateTooLargeMessage`,
`service_currencies.go:33`); `parseRateDate` (`rates_import.go:286`) and `checkRateInput` (`rates.go:58`) have no
minimum date. GSAP extract: `financials/feed_csv.go:144 readRecords` never sets `Comma`; export
`sheet.delimiterOf` (`platform/sheet/csv_read.go:47`), already used in spirit by `settings/importfile.go:64-68`.

## R17 — Precise answers (L-5, L-15, L-16)

- **L-5 (a)**: `platform/date/date.go:163 UnmarshalJSON` refuses a wrong date as unreadable; campaign and initiative
  inputs use `*date.Date` (`campaign/model.go:28,43,73,125,152`). Make them strings parsed in
  `campaign/validate.go` with "Enter a valid date." as `activity/dates.go:26`. **(b)** not a bug: `bulk.FieldChange`
  is `{key, value:{date|optionIds|yesNo}}` (`bulk/model.go:44-54`). **(c)** `readBody` turns
  `*http.MaxBytesError` into `bad_request`; return 413 `too_large` ("This request is larger than <limit>.").
- **L-15**: `DecideSubject` runs `roundToDecide` → `movedOn` (`approval/rounds_decide.go:73,219`) before `decidingOn`
  → `authority` (`:85`, `approval/authority.go:141`) — the order of 014 plan.md:215 and 017 data-model §6. New first
  step: refuse 403 a person holding neither `approvals.decide` nor `approvals.decideOnBehalf` in the plan.
- **L-16**: `overspend/model.go:101 DefaultConfiguration` has a zero `UpdatedAt`; same in
  `purchaserequest/model_routing.go:60`. Views send `updatedAt`/`updatedBy` as null when never saved; pages show
  FR-057's text (`OverspendSettingsPage.tsx:118`, `PurchaseRequestSettingsPage.tsx:87`).

## R18 — L-18: date format in headings and month labels (D-362)

`format/intl.ts:161 formatWeekdayDate` takes no format (used by `features/activity-log/log-text.ts:185 dayLabel`,
`LogEntryList.tsx:47`). Month labels ignore the preference in six places: `financials/financials-text.ts:63-75`,
`home/financials-text.ts:27-39`, `plans/RecordFinancials.tsx:127,156`, `line-items/line-item-math.ts:69
formatMonth`, `plans/phasing/phasing-math.ts:30`, `dashboards/dashboard-text.ts:43` (Intl en-GB → "Sept 2026").
Add `formatWeekdayDate(date, format)` and `formatMonth('YYYY-MM', format)` to `format/intl.ts`, expose `weekdayDate`
and `month` in `useFormatters`, and replace the six helpers (deleting four duplicate month arrays). Timeline column
headings and chart axis ticks keep month names.

## R19 — Remaining low findings (locations)

| Finding | Location | Fix |
|---|---|---|
| L-1 | `features/activities/useActivitySave.ts:132-138` | add `nextStep`; one `CONFLICT_NEXT_STEP` in `api/errors.ts` next to `conflictMessage` (`:107`) |
| L-4 | `line-items/line-item-requests.ts:51-59`; `plans/phasing/PhasingEditor.tsx:143-144` | prefix the month label |
| L-6 | `activities/useActivityActions.tsx:38`, `activity-tree-rows.tsx:118` | `MOVE_TO` (`plans/campaigns/texts.ts:16`) |
| L-7 | `line-items/SpreadEditor.tsx:186-205`, `plans/phasing/CustomMonths.tsx:125-140`, `format/decimal.ts:12,37` | paste handler; strip currency symbol/code in `parseMoneyInput` |
| L-8 | `backend/seed/seed.go:117`, `seed/history_planning.go:17` | created meta = first history entry |
| L-9 | `features/bulk/SubmitDialog.tsx:18-23` | `plural()` (`format/text.ts:28`) |
| L-12 | `format/intl.ts:228 formatMoneyCompact`; `global-budgets/global-budgets-text.ts:63`; `GlobalBudgetPage.tsx:130-139`, `ui/ConfirmDialog/ConfirmDialog.tsx:66` | full amount at ≥1000 of the largest unit; whole percent; ConfirmDialog `acknowledgeOnly` |
| L-17 | `features/people/access/access-words.ts:23-89`; duplicate `session/access.ts` | add the 4 purchase-request labels from `platform/authz/catalogue.go:78-81`; remove the duplicate table |
| L-19 | `ui/FieldShell/FieldShell.tsx:85-89`, `ui/MoneyField/MoneyField.tsx:49` | reserve the note row |
| L-20 | `ui/FormPanel/FormPanel.tsx:80` | Enter on a `select` → `requestSubmit()` |
| L-22 | `settings/currencies/OutcomeGroup.tsx:25-26`, `rates.module.css:206` | chevron marker |
| L-23 | `backend/internal/history/bin.go:189,197`, `activity/kinds.go:25,117,123`; `recycle-bin/RestoreDialog.tsx:116-126,173-178`; `ui/Toast/Toast.tsx:62`; `BinRow.tsx:88-92` | notes not duplicated in preview; "— View"; one tag style |
| L-25 | `features/home/GettingStartedCard.tsx:15` | steps by permission (`useCanChange`) |
| L-27 | `features/dev/RecentlyDeletedCard.tsx:44-52` | gate Restore |
| L-29 | `features/dashboards/PurchaseRequestsPanel.tsx:20-55` | empty sentence when count 0 |
| L-30 | `backend/seed/settings.go:49-57`, `seed/lineitems.go:65-70,132`, `seed/purchase_requests.go:70` | FY2027 budget items; key by year |
| M-14 | `features/plans/PlanCard.module.css:38-48` | `overflow-wrap: anywhere` |
| M-15 | `features/spa/Spa.module.css:30-49` | `min-width: 0; max-width: 100%` |
| M-16 | `features/workbook/CheckResult.tsx:14,84`, `workbook.module.css:57` | `role="region"`, label, `tabIndex={0}` |
| M-17 | `activities/SplitLinks.module.css:2-10`; `dashboards/DashboardsPage.tsx:111` with `ui/EmptyState`; `ui/FormRail/FormRail.module.css:96-99` | redefine `--ink-2`/`--link` to `var(--ink)` on grey containers (design system §3.1) |
| L-26 | `features/home/SummaryCards.tsx:~58`, `HomePage.tsx:52`; design system §2 | intended — no change |
