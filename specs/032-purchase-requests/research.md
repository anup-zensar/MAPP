# Research: Purchase Requests on MAPP's Side (032)

Each decision: what was chosen, why, and what else was considered. "Decision for the log" marks the ones
the orchestrator records in `docs/sdd/decision-log.md` (this agent may not edit it, brief §4).

## R1. One new business package, `internal/purchaserequest`

- **Decision**: purchase requests, their lines, routings and the Purchase Request Configuration live in one
  package `internal/purchaserequest` with the standard files (model, store, store_postgres, service, handler)
  split by use case. It reads line items, activities, plans, people, Settings, workflows and GSAP invoices
  through consumer interfaces (`ports.go`) that `internal/app` implements (D-196). It imports `approval` for
  the engine's subject types only (`SubjectKind`, `Outcome`, the section and facts types of brief §8.2):
  approval never imports it — the engine reaches purchase requests through its own `PurchaseRequests` port,
  which `*purchaserequest.Service` implements directly.
- **Why**: constitution B2 (package per business area, consumer interfaces); one place owns every PR rule;
  importing the engine's types avoids re-declaring the approval section (rounds, levels, history entries) a
  second time, and no import cycle is possible because `approval` depends on no business package.
- **Alternatives**: (a) put PR approval routes and rules in `approval` — rejected: the round kind, the status
  mapping, the gates and the modification belong to the PR (ADR-0016 "the rules stay in the services");
  (b) a separate `requisition` package for routing — rejected: routing is part of the PR's life and shares its
  store and transaction.

## R2. Storage: four tables, JSON only for snapshots and attribute values

- **Decision** (migration `00020_purchase_requests.sql`): `purchase_requests`, `purchase_request_lines`,
  `purchase_request_routings`, `purchase_request_configuration`; columns and constraints in data-model §1.
  Attribute values are a `json` object as on line items; the approved version and the routed PR form rows are
  `json` snapshots that are never queried, only restored or downloaded. Routings are write-once (the
  `refuse_change()` trigger of migration 00016). The vendor text is copied into `purchase_requests.vendor` at
  every save, from the attribute with role Vendor, so the list groups, sorts and searches on an indexed column.
- **Why**: relational schema with a constraint for every invariant the database can check (B5); snapshots as
  JSON follow `approval_rounds.levels` and `recycle_bin`; the copied vendor keeps the list query one statement
  per page (B5, SC-007).
- **Alternatives**: a column per approved field (doubles the PR table and the line table for data never
  queried); a child table for routing rows (adds a table whose rows are only ever read together).

## R3. Scope

- **Decision**: a PR stores its plan (`plan_id`, the plan of its first line; every line must be of it,
  FR-013). SQL function `purchase_request_in_scope(...)` = plan in the `plans.view` scope (plan_in_scope),
  or in the `purchaseRequests.list` scope (plan_in_permission_scope), or the person owns the PR, is its MoA
  approver, or is named as an approver in one of its approval rounds (031's subject columns). Every store read
  that serves a person takes the scope (`ViewScope{Plans, List}` from the actor's grants) and filters in SQL;
  the list page filters by the `purchaseRequests.list` scope only. Each scoped read has a scope-isolation test.
  A campaign that holds a line item a PR draws on cannot move to another plan (FR-019), so the stored plan
  stays true.
- **Why**: constitution VII and B5 (scope in the query, 404 outside it); FA F4 (the requisitioner sees PRs
  through the PR grid right, not through plan rights).
- **Alternatives**: deriving the plan from the lines on every read (a five-table join per row and an
  ambiguous plan when a campaign moves).

## R4. Reservation and balance

- **Decision**: no stored balance. Total requested of a line item = Σ of the amounts of the lines on it in PRs
  that exist (the recycle bin holds deleted PRs and removed lines outside the tables), each converted into the
  line item's currency at today's rates (`money.Convert`, D-303); Closed PRs count at their closed amounts.
  Available = planned cost − total requested. Every change that can raise a claim (create, save, add line,
  modification, restore of a PR or a line) checks it inside its `Write`; Writes run one at a time (advisory
  lock, feature 023 R2), so two claims cannot both pass.
- **Why**: one source of truth; no drift between a stored balance and the lines; matches SC-003.
- **Alternatives**: a `reserved_minor` column on line items (a second writer to `line_items`, and a value
  that rate changes make wrong); `SERIALIZABLE` retries (measured as starving in 023).

## R5. Round kinds and the status mapping (brief §8.2)

- **Decision**: the PR decides the kind of the round a submission starts: `modifiedPurchaseRequest` when the
  PR has an approved version (Approved, or In review with a change pending), `purchaseRequest` otherwise. It
  calls `SubmitSubject`/`RecallSubject`/`DecideSubject` inside its own `Write`; the engine calls back
  `ReadyCheck` (032's gates, FR-021) and `RoundChanged`, where 032 sets the status per the Status Model:
  - kind purchaseRequest: open → Submitted when the PR was Created or Declined (or after superseded), In
    review when it was already Submitted or In review (a level approved, more remain); approved → Approved +
    approved version + routing (Original); declined → Declined; recalled or cancelled → Created; superseded →
    Created inside the step (the next `open` makes it Submitted).
  - kind modifiedPurchaseRequest: open → In review with change pending; approved → Approved + new approved
    version + routing (Modification); declined, recalled or cancelled → Approved with the approved version
    restored (lines added by the change removed, values and amounts restored); superseded → no change.
- **Expectation on 031 (Decision for the log)**: the engine calls `RoundChanged` with outcome `open` after
  every decision that leaves the round open (a level approved, further levels remain), not only at the
  round's start, so 032 can show In review. If 031 does not, the PR stays Submitted until the round ends —
  the only effect is the status word.
- **Log entries**: `RoundChanged` writes no entry for the status change or the restore — the engine's decision
  entry covers them (brief §8.2). The routing that approval triggers is a separate event and writes one
  "routed" entry (FR-057; interpretation, Decision for the log).

## R6. Modification in one step; the approved version

- **Decision**: no draft modification. Saving an Approved PR requires `then: submitChanges` (save + start the
  modification round in one `Write`), except a save that only adds lines while the setting is "Allowed
  without approval" (`then: save`, PR stays Approved, routed as Modification). Saving a Submitted or In
  review PR requires `then: resubmit`. The approved version (header values, attribute values and every
  line's values at the last approval) is stored at each approval; a declined or recalled change restores it.
  Lines cannot be removed from an Approved PR (FR-015), so a restore only reverts values and removes lines the
  change added (their reservation ends with them).
- **Why**: 013 Status Model rows "Approved → Submit changes → In review" and FR-033's "Save and re-submit";
  a stored draft would need a third reservation state nobody asked for.
- **Alternatives**: a separate "PR modification" record (013 Key Entities) — rejected as a second record with
  the same fields; the approved-version snapshot gives the same behaviour.

## R7. Routing inside MAPP and the PR form (D-320)

- **Decision**: routing is part of the approval step (`RoundChanged` approved) and of Close, Resend and a
  line added without approval; it writes a write-once routing row (type, time, who, recipients, form rows) and
  sets Sent dates. Recipients: Email group label(s), then PR recipients labels; "Requisitioner queue" when both
  are empty. The PR form is generated from the stored rows by `platform/sheet` (`WriteXLSX`, `WriteCSV`;
  formula-safe text, D-120), file name `MAPP PR form – PR-1001 – <type> – <date>.xlsx`. The requisitioner queue
  is the list filter Waiting for GSAP (FR-032), computed from the PR (approved, routed, a line without PO line or
  with amount/currency ≠ PO value).
- **Why**: e-mail is out of scope (constitution 4.4.0); in-app routing cannot fail, so 013's retries and
  "Routing failed" have nothing to describe; storing the rows proves what was routed (SC-008, 013 edge case
  "SWBS code changed after routing").
- **Alternatives**: in-app notifications to every requisitioner (no requisitioner identity per Email group in
  MAPP; the queue view serves the same purpose); generating the form from current values (loses what was
  routed).

## R8. PR attributes as Settings configuration

- **Decision**: two new Shell attribute levels, `purchaseRequest` and `purchaseRequestLine` (settings `Level`,
  contract `AttributeLevel`), marked optional or needed to submit, with Business unit conditions taken from the
  PR's plan; no % split type. Attribute roles tell the code which attribute it reads: `vendor`, `vendorCode`,
  `vendorEmail`, `shortText`, `prType`, `emailGroup`, `prRecipients` (PR level), and `companyCode`,
  `costCentre` (activity level) and `glCode` (line-item level) for the gates and the PR form (030 may already
  have added `companyCode`; 032's migration keeps every role that exists). An attribute with a role keeps its
  level and type and cannot be deleted (lock reason "MAPP reads this attribute for purchase requests"), like
  the Business unit attribute. Settings' "Used by" counts PRs and PR lines through a reference source.
- **Why**: 013 FR-008 (custom attributes are configuration, changeable without code); the role mechanism
  already exists (D-055) and is the least code; finding attributes by label would break on a rename.
- **Alternatives**: standard columns for every PR field (contradicts FR-008); a new attribute "email" type
  (FR-007 needs one check on one attribute — the role is enough).
- **China-specific attributes**: seeded optional, always shown (Clarifications); a Country/OU condition would
  be a new Settings mechanism nobody else needs yet.

## R9. Permissions, route rules and roles (FA F4) — Decision for the log

- **Decision**: four scoped permissions in a new group "Purchase requests" (014 Table A additions):
  `purchaseRequests.edit` "Raise and edit purchase requests", `purchaseRequests.delete` "Delete purchase
  requests", `purchaseRequests.submit` "Submit purchase requests for approval and recall them",
  `purchaseRequests.list` "See the Purchase requests page". Route rules in
  `access/routes_purchase_requests.go` (contracts/routes.md). Seeded roles: Global planner all four; Planner
  edit, delete, submit; new role **Requisitioner** (RL-1006, not built in, default scope All markets):
  `plans.view`, `purchaseRequests.list`, `finance.import`; Administrator gets them by definition.
  The `roles.permissions` check of migration 00007 is re-created with the four keys.
- **Why**: FA F4.1/F4.2: PR create/edit/delete rights are per role and separate from the PR grid; Marketers
  (Planner) do not see the grid, the TA requisitioner sees the grid and edits nothing. In the simulation the
  requisitioner stands in for the GSAP side, so the role holds `finance.import` (brief: "the requisitioner (or a
  finance.import holder) runs" the simulated GSAP; ADR-0016).
- **Alternatives**: reuse `activities.edit` for PRs (FA F4 shows IT Admin with line items but no PRs); a new
  `purchaseRequests.process` permission for the simulated GSAP (the brief and ADR-0016 name `finance.import`).

## R10. Delete and move protection (FR-019)

- **Decision**: one generic hook in `history`: `(*Service).UseDeleteGuard(func(ctx, snapshots []Snapshot)
  error)`, called by `MoveToBin` after it collects the records and before it removes them. `internal/app`
  installs a guard that refuses (409 `in_use`) when a collected line item is drawn on by a PR, naming up to
  five PRs. Line items: `lineitem.(*Service).UsePurchaseRequests` (port `Requested(ctx, lineItemID, currency)`)
  refuses a planned cost or currency change below Total requested (400 `validation_failed` on
  `plannedCost`). Campaigns: `campaign.(*Service).UsePurchaseRequests` (port `DrawnOnUnder(ctx, campaignID)`)
  refuses a move to another plan. Until set, none of these refuses (as today).
- **Why**: one place covers deletes of all five levels and every future one; the two value rules sit where
  the values are validated; the setter keeps the existing constructors and fakes unchanged.
- **Alternatives**: a check in each of the five delete services (five copies); a foreign key alone (answers
  an internal error instead of a message naming the PRs).

## R11. Line-item and activity totals are served by `purchaserequest`

- **Decision**: `GET /api/v1/line-items/{lineItemId}/purchase-requests` and
  `GET /api/v1/activities/{activityId}/purchase-requests` answer Total requested, Available to request and the
  PRs; the line-item and activity pages show them in a "Purchase requests" section. The `LineItem` and
  `Activity` answers are not changed.
- **Why**: keeps 032 out of the line-item and activity views that 030 and 034 change at the same time; the
  figures come from the server either way (constitution IV).

## R12. Invoiced amounts (FR-041, FR-043)

- **Decision**: port `Invoices.InvoicedOn(ctx, []POLineRef)` implemented in `internal/app` over the
  financials store: invoices (credit notes negative) whose PO number and PO line match, converted to USD at
  today's rates; 032 converts to the line's currency. Lines without a PO (everything before 033) have 0.00.
  Close sets each line to min(amount, invoiced).
- **Why**: the invoice data exists (015) and the PO comes from 033 with the same keys.

## R13. PR approval routes; the inbox preview is a GET

- **Decision**: the PR approval operations of brief §8.2 are served by `purchaserequest`'s handler:
  `GET …/approval`, `GET …/approval/submission`, `POST …/approval/submit|approve|decline|recall`, and the
  inbox preview **`GET /api/v1/approvals/purchase-requests/{purchaseRequestId}`**. The brief writes `POST` for
  the preview path; it reads nothing to change, mirrors `GET /api/v1/approvals/activities/{activityId}`, and is
  safe and cacheable as a GET. Reported to the orchestrator as a deviation.
- **Why**: one handler per area; the approval section JSON reuses the engine's types (R1).

## R14. Notifications

- **Decision**: the engine (031) notifies the owner on approve and decline for every subject kind; 032 adds
  the web link for subjects of kind `purchaseRequest` and `modifiedPurchaseRequest` (to `/purchase-requests/
  {id}`) and a test that an approved PR notifies its owner.

## R15. Demo data (Decision for the log)

- **Decision**: appended after the financial transactions and before the demo history (D-306): the
  Requisitioner role and person **Lea Santos** (U-1007, `lea.santos@shell.com`, "Requisitioner", team Global
  Marketing Ops, role Requisitioner for All markets); the configuration (on; Allowed with modification
  approval); the PR attributes of the field inventory; two workflows "Purchase request approval (MoA)" (type
  purchaseRequest) and "Purchase request change approval (MoA)" (type modifiedPurchaseRequest), one level each
  whose approver is the MoA approver field, backup Alex Morgan; one Created PR **PR-1001** "PR for Media
  booking" on L-1005 (30,000.00 USD, delivery 2026-09-30, owner Priya Raman, MoA approver Daniel Okoro, vendor
  "Brightside Media BV", vendor code V-100104, PR type Standard PR/PO, Service). Existing IDs do not move.
- **Why**: every screen has something to show; the existing demo story keeps its IDs and totals (D-274
  pattern).

## R16. No new dependency

`platform/sheet` already writes XLSX and CSV; `github.com/google/go-cmp`, `libopenapi-validator`, Vitest, MSW
and Playwright are in place.

## R17. Performance (SC-007)

- **Decision**: the list is one statement per page plus one for its lines and one for vendor totals; indexes
  on `purchase_requests (plan_id)`, `(lower(vendor), id)`, `(status)`, `purchase_request_lines
  (purchase_request_id)`, `(line_item_id)`, `(po_number)`. Search uses `lower(...) LIKE` on id, name, vendor,
  GSAP PR number, and an `EXISTS` on lines for HX key and PO number. A benchmark at 10,000 lines is a polish
  task (028's `apibench`).
- **Measured** (T089, `BenchmarkPurchaseRequestsPage` in `internal/platform/apibench`, data
  `purchaseRequests`: the demo plus 2,000 PRs of five lines = 10,000 PR lines, 40 vendors; Apple M5,
  PostgreSQL 18 local, 20 runs each): first page of 50 — 29 ms, 35 statements; search by HX key — 31 ms,
  35 statements; one vendor — 6 ms, 35 statements; Waiting for GSAP (empty) — 2 ms, 25 statements. The
  statement count does not grow with the rows of a page (no N+1); SC-007's 2 seconds holds with a wide margin.

## R18. History kinds

`purchaseRequest` (top level; "purchase request"/"purchase requests"), `purchaseRequestLine` (parent
`purchaseRequest`; "PR line"/"PR lines"), `purchaseRequestConfiguration` (log only, never in the bin). Log and
bin visibility: PR kinds follow the PR's plan (D-238); restore needs `purchaseRequests.delete` in that plan
(PR) or `purchaseRequests.edit` (line); the configuration needs `activityLog.viewAll`. PrepareRestore of a PR
and of a line re-checks Purchase Support, the parent's status and the balance (FR-042, FR-015).

## R19. HX key

Derived (`"HX" + strings.TrimPrefix(lineItemID, "L-")`), never stored, the same function the line item's facts
expose (brief §8.1). The legacy Hive9 numbering question (gap analysis L18) stays open and is not affected.

## R20. Web app structure

- **Decision**: one feature folder `src/features/purchase-requests/` (pages, form, sections, settings card,
  API hooks). It imports `@/features/approvals` (ApprovalChain, ApprovalHistory, decision dialogs) and
  `@/features/activity-log` (HistorySection); `line-items` and `activities` import its sections; `approvals`
  gets its own PR preview panel and calls the PR approval endpoints through its own `api.ts`, so no feature
  imports another in both directions (F2, no cycles).
- **Why**: F2 (feature folders, index-only imports, no cycles).

## R21. Implementation notes (backend, recorded while building)

- **Approval engine port.** 031 delivered `SubmitSubject`/`RecallSubject`/`DecideSubject` and calls the
  subject's `RoundChanged(ctx, kind, id, outcome, round)`; `purchaserequest` declares the small `Engine` and
  `Rounds` ports it calls (`approval.go`) and `subject.go` implements 031's `PurchaseRequests` port, so the
  packages do not import each other in both directions.
- **PO currency.** `purchase_request_lines` has a `po_currency` column beside `po_value_minor` (data-model
  §1.2 named only the value): a PO line's value keeps its own currency, checked together with `po_line`.
- **Top-up numbering.** `RoutedPR.LastPOLine` carries the highest PO line already given, so the simulated
  GSAP continues at the next ten even after a line was removed.
- **Settings levels.** `settings.AllLevels()` lists the four attribute levels (the levels were a closed list
  in several files); `splits(level)` keeps percentage splits to the activity level; every attribute that has a
  role other than business unit is locked in type, level and deletion.
- **Roles on existing attributes.** Company code, Cost centre and GL code carry the roles
  `companyCode`, `costCentre` and `glCode`, so the PR reads them from the line items' activities without
  matching labels.
- **Demo PR on L-1010.** PR-1001 sits on the UK plan's Agency fee (Ireland), not on Media booking (L-1005):
  a PR on L-1005 or L-1009 blocked deletes and moves that the tests of features 004–008 make on the demo data.
- **Delete guard.** `history.UseDeleteGuard` runs after the subtree is collected and before anything is
  removed or logged; `lineitem.UsePurchaseRequests` refuses a planned cost below what is requested (skipped
  while the currency is being cleared); `campaign` refuses a move to another plan while a PR draws on it.
  Moves of activities and initiatives across plans are not guarded yet (gap).
- **Routing entry.** Approval writes one activity-log entry "routed" with the recipients inside the round's
  transaction; a failed routing undoes the approval.
- **Activity workflows.** The seed's activity workflow list (`activity/workflows_seed.go`) keeps only
  workflows of kind *activity*, so the PR workflows never appear on the activity form.

## Decisions for the log (proposed)

| # | Decision |
|---|---|
| 032-A | PRs belong to one plan (their first line's); one company code per PR; lines of other plans refused (R3). |
| 032-B | The MoA approver is a person field, not text attributes; the PR form shows name and e-mail (Clarifications). |
| 032-C | Edits of Submitted/In review PRs are Save and re-submit; edits of Approved PRs are Submit changes, both in one step; no draft modification (R6). |
| 032-D | Routing is in-app (routing records + PR form download + Waiting for GSAP view); no retries or failure state (R7). |
| 032-E | Four scoped permissions in group "Purchase requests"; seeded Requisitioner role (plans.view, purchaseRequests.list, finance.import); Planner without the list page (R9). |
| 032-F | Delete protection through a history delete guard, covering line item to plan; campaign moves across plans refused (R10). |
| 032-G | Close sets each line to min(amount, invoiced); invoiced = GSAP invoices minus credits on the line's PO line (R12). |
| 032-H | PR attributes are Settings levels with attribute roles; China-specific attributes seeded optional (R8). |
| 032-I | 031 calls RoundChanged(open) after each decision that keeps a round open; RoundChanged writes one "routed" entry when it routes (R5). |
| 032-J | Inbox preview is GET /api/v1/approvals/purchase-requests/{id} (R13). |
| 032-K | Demo data additions of R15; PR-1001 on L-1010 (R21). |
| 032-L | `purchase_request_lines.po_currency` stores the PO value's currency (R21). |
| 032-M | Company code, Cost centre and GL code carry attribute roles; `settings.AllLevels()` is the list of levels (R21). |
