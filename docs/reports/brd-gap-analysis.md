# BRD gap analysis: NorthStar against Shell's Business Requirements

| | |
|---|---|
| **Date** | 2026-10-20 |
| **Readers** | Product owner and the Shell client team |
| **Requirements** | Shell's latest BRD (`Business Requirements.pdf`, 2 pages), transcribed row by row in [../reference/brd-requirements.md](../reference/brd-requirements.md): 36 rows, of which 11 are **MVP Requirement** |
| **Application compared** | NorthStar on branch `plan-module` (commit `e4966aa`): features 001–010 built, integrated and tested. Feature 011 (activity log and recycle bin pages) is built on branch `lane-011` but not merged (D-208, D-209); a worktree branch `lane-011b` exists. Feature 012 (bulk actions and import) is deferred, and 013 (BCPIR and SP&A) waits on Q-001 |
| **Method** | For every BRD row: read the specs (`specs/0NN-*/spec.md`, `plan.md`), the decision log (`docs/sdd/decision-log.md`), the product docs (`docs/product/`) and the code (`backend/internal/*`, `frontend/src/features/*`). The app was not run for this report; behaviour is taken from the code, the specs and the recorded end-to-end tests (D-221) |

**Status key.** **Met**: the requirement works today. **Partly met**: some of it works, with named gaps.
**Not met**: nothing usable exists yet. **Not applicable**: not a product requirement for NorthStar (the
client moved it to the technical track, or it is a vendor question).

**Size key.** Rough effort for one *lane*, meaning one backend and one frontend developer working from a
spec, with tests and documentation included. **S**: up to 3 working days. **M**: 1–2 weeks. **L**: 3–5
weeks. These are order-of-magnitude figures. Shell's answers to the questions in §6 will narrow them.

---

## 1. Summary

- **Of the 11 MVP requirements, 2 are met, 6 are partly met and 3 are not met.**
  - **Met:** stopping a submission when mandatory data is missing (BUS.WFL.030), and re-using last year's
    plan (BUS.PLN.050, with small gaps).
  - **Partly met:**
    - budget categories (BUS.BUD.020);
    - currency (BUS.BUD.050);
    - multi-year, quarterly and cross-market planning (BUS.PLN.020);
    - monthly phasing and splitting (BUS.PLN.040);
    - field help (BUS.UX.020);
    - approvals (BUS.WFL.010).
  - **Not met:**
    - role- and team-based access (BUS.ADM.020);
    - global/local budget allocation (BUS.BUD.040);
    - financial transactions in the main views (BUS.FIN.080).
- **The three "not met" gaps come from decisions to leave things out of this phase, not from defects.**
  - Roles were postponed: every signed-in user can do everything (D-004, ADR-0005).
  - Finance and GSAP were left out of scope (D-008).
  - A global/local budget concept was never in the specs.
  - Each needs a Shell decision and a new feature.
- **Most "partly met" gaps come from simple defaults chosen while Shell's answers were pending:**
  - one current exchange rate (D-027);
  - budgets and activity costs in USD only (D-079, D-220);
  - simple ordered approval levels (D-205 / Q-002);
  - no filtering by Shell attributes (D-207 / Q-009);
  - single-year plans (D-017);
  - single copies only within the same year (D-096).
  - The BRD now asks for more than each of these defaults. Section 5 lists the decisions to reopen.
- **The BRD supports the current deferrals with one exception:**
  - Bulk actions and Excel import (012), and business case / PIR (013), are **Future-State/Later** in the
    BRD.
  - However, the BRD's MVP approvals row asks for an **audit trail**. That needs the activity log pages
    of **011 to be merged**.
- **Access control is the first thing to build.** Global/local visibility (BUD.040), role-based
  approvals and segregation of duties (WFL.010), and access to financial data (FIN.080) all depend on it.
- **Effort to close the MVP gaps:**
  - about 5 large, 5 medium and 2 small items;
  - roughly **20–35 lane-weeks**;
  - with three lanes in parallel, about **8–12 calendar weeks** after Shell answers the questions in §6.
- **Eight questions for Shell (Q-010 to Q-017 in §6) block detailed specs.** The most urgent are the
  role catalogue, the meaning of "global/local", the exchange-rate policy and the auto-approval
  criteria. We also need the "note on financial transactions" that BUS.FIN.080 refers to: it is not in
  the PDF.

### 1.1 Summary table

MVP rows first, then the other dispositions in BRD order. Sections 3 and 4 give the evidence.

| Requirement | Disposition | Status | Gap | Proposed change | Size |
|---|---|---|---|---|---|
| **BUS.ADM.020** Role- and team-based access control; edit vs view-only | MVP | **Not met** | No roles. Every signed-in user can see and change everything, including Settings (D-004, D-020) | New feature **014 Roles and access**: roles and team/market access rules set up in Settings, view-only vs edit, enforced by the API | L |
| **BUS.BUD.020** Budget categories (region, brand, spend type, business unit, channel); client-updatable attributes; flexible framework | MVP | **Partly met** | The categories exist as admin-configurable Shell attributes. Budgets cannot be set or analysed by category: All work has no attribute filters or grouping (D-207), and the five planning levels are fixed | **010**: attribute filters and "group by attribute" with totals. Plan page: spend breakdown by category. Ask whether budgets must be set per category (Q-017) | M |
| **BUS.BUD.040** Global/local budget allocation with visibility for each owner | MVP | **Not met** | No global vs local budget, no allocation across plans, no visibility rules | New feature **015 Global and local budgets**, after 014 | L |
| **BUS.BUD.050** Currency conversion, rate management, GSAP daily rates; plan in local currency and USD | MVP | **Partly met** | Only line items accept a local currency; budgets and activity planned cost are USD only. One current rate per currency with no history; no GSAP rate feed | **002 + 004–007**: dated rates with rate types, import of the GSAP rate file, local-currency budgets and costs shown beside USD | L |
| **BUS.FIN.080** POs, invoices, payments, accruals prominent in main views | MVP | **Not met** | No transaction data. Activities and line items show only a collapsed "From finance (coming later)" section of dashes | New feature **016 Financial transactions** (read-only, loaded from GSAP extract files), shown on the activity, line items, plan tree and All work | L |
| **BUS.PLN.020** Multi-year, annual, quarterly, monthly; strategic, operational, cross-market planning | MVP | **Partly met** | Annual plans, monthly/quarterly cost phasing and calendars exist. No multi-year plans or views; "Strategic/AOP" is only a tag; no views by market | **010**: year range and multi-year totals. Strategic plan linking annual plans (after Q-016). Filters by market (with BUD.020) | L (in M pieces) |
| **BUS.PLN.040** Split budget by month; split an activity from its original budget; ongoing re-phasing | MVP | **Partly met** | Costs are phased by month. Budgets are one annual amount. There is no Split action, and Move stays within the same plan | **004/005/010**: monthly budget phasing with distribute tools. **006/007**: Split activity. Widen Move (D-033) | M + M |
| **BUS.PLN.050** Re-use prior-year content | MVP | **Met** (small gaps) | Copying a whole plan into next year works. A single campaign or activity cannot be copied into another year, and there is no "structure only" copy | **005/006**: copy single records across years. **004**: "structure only" option | S |
| **BUS.UX.020** Contextual help (ⓘ) maintainable by administrators | MVP | **Partly met** | Shell attributes have admin-maintained help. Built-in fields (plan budget, planned cost, dates, workflow, …) have none, and the few fixed tips cannot be edited | **002**: Settings › Field help for every built-in field, read by every form | M |
| **BUS.WFL.010** Multi-level, parallel, conditional approvals, delegation, notifications, escalation, roles, SoD, compliance, audit, monitoring, dashboards (client: simplify to auto vs manager approval) | MVP | **Partly met** | Multi-level approvals, monitoring, in-app notifications and approval history exist. Missing: auto/manager/conditional approval, delegation, escalation, segregation of duties, roles. The log page (011) is not merged | **008 extension "Approval rules"**: auto vs manager approval, delegates, reminders and escalation, segregation of duties. Merge **011** | L |
| **BUS.WFL.030** Highlight missing mandatory data and halt submission | MVP | **Met** | Nothing material. The markings of built-in fields are fixed and should be confirmed (Q-005) | Confirm the markings with Shell | S |
| BUS.AI.010 AI capabilities; future-ready | Future | Not met (by design) | No AI (out of scope). The open REST API and next-year copy are the base | Keep for later. "Structure only" copy (PLN.050) answers the client's example | — |
| BUS.BUD.030 Overspend hard stop or warning; include underspend | Future | Partly met | Overspending warns in red and never blocks. No underspend warning, and no setting | Underspend warning; optional per-plan setting | S–M |
| BUS.FIN.020 Full PR/PO automation | Future | Not met | Finance and GSAP out of scope (D-008). The SWBS request is only recorded | Technical track, after the GSAP freeze | L (later) |
| BUS.FIN.040 Monthly accruals per PO | Future | Not met | No POs or accruals | Later, on top of 016 | L (later) |
| BUS.FIN.050 Review and approve supplier invoices | Future | Not met | No invoices | Later, on 016 and 008 | M–L (later) |
| BUS.FIN.060 PO/invoice documents linked to the PR | Future | Not met | No file attachments anywhere | Later, with 016 | M (later) |
| BUS.PRF.020 PIR linked to business case / SP&A activity | Future | Not met | PIR is only a Yes/No attribute; 013 waits on Q-001 | 013 after Q-001 | L (later) |
| BUS.PRF.030 Central view of all content, scoped by access | Future | Partly met | Home and All work show plans, budgets and costs centrally. No KPIs/PIRs, no access scoping | Scoping with 014; PIRs with 013 | — |
| BUS.RPT.010 Standard dashboards | Future | Partly met | Home, plan and All work summary cards only. No executive, regional or KPI dashboards; no forecast or actuals | Later (Measure replacement) | L (later) |
| BUS.RPT.050 Calendar views by campaign, owner, period, country | Future | Partly met | Calendar (timeline and month) filters by plan, owner, team, business unit and type; not by country | Comes with attribute filters (BUD.020) | S after BUD.020 |
| BUS.UX.010 Web/mobile, accessibility, languages, Excel, bulk, dashboards, notifications, search | Future | Partly met | Responsive web, WCAG 2.1 AA checks, Excel/CSV export, Home, in-app notifications and search exist. No offline use or mobile app; English only; no bulk (012) | Later: 012, global search, more languages | L (later) |
| BUS.WFL.040 Notify owners of changes; customisable | Future | Not met | Notifications only about approvals | **008**: "changes to what I own" plus preferences; e-mail later | M |
| BUS.WS.01 Export plans to ppt/pdf/word/excel; share to Outlook/Teams | Future | Partly met | All work exports .xlsx/.csv; no PDF/PPT/Word, no sharing | Plan export to PDF/.xlsx; "share link" | M |
| BUS.WS.02 Import a plan skeleton / upload an updated plan from Excel | Future | Not met | Activity import is specified in 012 (deferred). There is no update round-trip | Build 012 import; add update by ID | L |
| BUS.ADM.010 Shell can configure without the vendor (DIY) | RFP | Partly met | Settings are fully self-service and live without a release. No role/permission admin; audit page not merged | Covered by 014, 011 and UX.020 | — |
| BUS.ADM.050 Migrate current and historical Hive9 data | RFP | Not met | No migration. New plans are limited to current year −1 … +5 | New migration feature | L |
| BUS.FIN.010 GSAP integration (PR, PO, invoices, …) | RFP | Not met (technical) | Out of scope; GSAP frozen until about March 2027 | Technical track | — |
| BUS.INT.010 Standard integrations (Power BI, Teams, …) | RFP | Not applicable (technical) | No connectors. Open REST API with OpenAPI contract; Excel/CSV export | Technical track | — |
| BUS.WS.03 Multiple time zones; offices | RFP | Not applicable (vendor) | Times shown in each person's own time zone; stored in UTC | Vendor/support answer | — |
| BUS.WS.04 Keep 5 years of data to refer back to | RFP | Partly met | Nothing is purged and the log is append-only. No Hive9 history, and earlier years cannot be created | With ADM.050: read-only archive years | M (with ADM.050) |
| BUS.BUD.010 Ownership, approvals, transfers, locking, freeze, multi-currency, carry-forward | Clarification (duplicate) | Partly met | Ownership and approvals met; multi-currency and carry-forward partly; no transfers or locking | Covered by BUD.050 and PLN.050; optional plan lock | S–M |
| BUS.PLN.010 Global template, standard hierarchy and attributes | Clarification (in MVP) | Partly met | Standard hierarchy, attributes and models are configurable. No "global template" record | Covered by BUD.020 and BUD.040 | — |
| BUS.PLN.030 Rolling forecasts, phasing, allocation, scenarios | Clarification (duplicate) | Partly met | Phasing and allocation met. No forecasts or scenarios | Later | — |
| BUS.PLN.060 In-cell formulas | Clarification (not required) | Not applicable | The client says it is not required. Calculated values are automatic | None | — |
| BUS.RPT.060 Filters and saved filter sets | Clarification (duplicate) | Met (mostly) | Filters, search, personal saved views and a default view exist. No attribute filters; views are not shared | Attribute filters (BUD.020) | — |

---

## 2. What NorthStar already gives Shell against this BRD

Before the gaps, here is what the BRD asks for that already works on the integrated branch. Each item is
tested by Go tests, component tests and the end-to-end journeys in `frontend/e2e/core-flow.spec.ts` and
`frontend/e2e/module-rest.spec.ts` (D-208, D-221).

- **Mandatory data stops the process.** Mandatory data stops a submission or a publish, and every
  missing field is listed at once with links (BUS.WFL.030).
- **Last year's plan can be re-used.** A whole plan copies into next year with every campaign,
  initiative, activity and line item, its dates moved one year and its statuses reset (BUS.PLN.050).
- **Shell attributes are self-service.** Administrators can add attributes, options, dependent lists
  and business-unit conditions at every level, without a release (BUS.BUD.020, BUS.ADM.010).
- **Costs are phased by month.** Each line item's cost is spread over months (evenly, first month, last
  month or custom), with a quarterly view and a Budget & cost view across plans (BUS.PLN.040,
  BUS.PLN.020).
- **Approvals have levels and an inbox.** Activities are approved level by level, with an approvals
  page, summary cards, a waiting-time indicator, in-app notifications and a full approval history on
  each activity (BUS.WFL.010).
- **Everyone can choose a display currency.** Every amount is converted and labelled, and line items can
  be typed in a local currency (BUS.BUD.050).

---

## 3. MVP requirements in detail

### 3.1 BUS.ADM.020: role- and team-based access control (Not met · L)

**Shell asks:** Team-based and role-based access control (RBAC) for plans, POs, invoices and related
financial data, configurable through access rules rather than code. The client also wants to control who
can edit and who can only view.

**Today in NorthStar**
- **No roles, by decision.** The product owner postponed roles and authentication. Every signed-in user
  may do everything (D-004, D-020; `docs/architecture/adr/0005-demo-sign-in-rbac-later.md`;
  feature-map cross-cutting rule 4).
- **The API checks only that someone is signed in.** `RequireSignIn` turns the session into an "actor"
  and nothing else (`backend/internal/auth/middleware.go:10-33`). Services say so explicitly: "No method
  checks roles … the actor is where those checks will go" (`backend/internal/campaigns/service.go:22-24`;
  also `backend/internal/history/service.go:545`).
- **A person has no role field** (`backend/internal/people/model.go:19-26`).
- **Every spec states Shell's rule and defers it:** Settings (002 FR-008), people and teams (003
  FR-027), plans (004 FR-032), approvals (008 FR-017, FR-021, Assumptions); domain model §8.
- **What can be built on:**
  - teams and memberships (003);
  - plan teams (`backend/internal/plans/model.go:72`);
  - the **My teams** switch on Plans, Home and All work, which narrows the view but does not enforce
    anything (D-035, D-080, Q-006);
  - the actor recorded on every change (D-004);
  - the "on behalf of" record on approvals (008 FR-017).
- **There is no financial data to protect yet.** No POs or invoices exist (see BUS.FIN.080).

**What is missing**
1. **Roles.** A role catalogue with permissions per action and level: view, create, edit, delete,
   publish/unpublish, submit, approve, administer Settings, and view or edit financial data.
2. **Access rules.** Data scoping by **team** and by **market** (Business unit, Cluster/Region,
   Country/OU values), set up in Settings rather than in code. This is also what BUS.PRF.030's comment
   describes: "Global can view all data for all markets vs local market can view all data for their
   market".
3. **View-only vs edit.** A view-only role. On screen, actions a person may not use are hidden or
   disabled with the reason, following the existing "unavailable actions say why" rule (004 SC-003).
4. **Enforcement everywhere, by the API.** The API must enforce the rules on every operation, and also
   in Home, All work, exports and counts, so that totals include only visible data.
5. **Administration and audit.** Administration of role assignments, with every change in the activity
   log.

**Proposed change**
- **New feature 014 "Roles and access"** (spec, plan, tasks).
- **A few roles instead of Hive9's twenty.** Map Hive9's 20 roles
  (`docs/reference/hive9-functional-analysis.md` §F4) to about six NorthStar roles: Administrator,
  Global budget owner, Planner (marketer), Approver, Finance viewer, Viewer.
- **Each assignment is a role plus a scope:** everything, my teams, or chosen markets.
- **Checks go in the services**, at the actor checkpoint every service already has (ADR-0005
  "Consequences").
- **Works with demo sign-in.** It can be built while demo sign-in remains, by assigning roles to demo
  people. Entra ID app roles replace the assignment later.
- **Decisions:** reopen D-004, D-020 and Q-006.

**Size:** L. Every service and screen of 002–011 gains a check, and Home, All work and exports must
filter by visibility.

**Question for Shell:** Q-010 (§6).

---

### 3.2 BUS.BUD.020: budget categories (Partly met · M)

**Shell asks:** Support budget categories such as region, brand, spend type, business unit and channel.
The client adds: (1) the client must be able to update attributes; (2) the tool must fit the business's
framework or hierarchy.

**Today in NorthStar**
- **Administrators maintain Shell attributes at every level, without a release.** They set label, type,
  marking, help, section, order, dependent options, business-unit condition and option import (002
  FR-016–FR-027, FR-031; D-014). Comment 1 is met.
- **The BRD's examples are all seeded** (`docs/product/field-catalogue.md`):

  | Category | Where | Catalogue reference |
  |---|---|---|
  | Business unit | Plan | §1 #12 |
  | Cluster/Region | Campaign | §2 #8–9 |
  | Channel/audience | Activity | §4.2 #12 |
  | Country/OU, with a % split | Activity | §4.2 #13 |
  | Brands, with a % split | Activity | §4.3 #22 |
  | Spend type → GL code | Line item | §5 #8–9 |

- **Any attribute can be a column in All work and in its export** (010 FR-018, FR-032).

**What is missing**
1. **Budgets by category.** Budgets exist only on the hierarchy: plan, campaign and initiative (domain
   model §4.1; `backend/internal/plans/model.go:76`). A budget cannot be set per brand or channel.
2. **Totals by category.** Planned cost cannot be seen added up by category.
3. **Filters and grouping.** All work offers no attribute filters and no grouping. This was resolved as
   "standard filters only" while Q-009 was open (D-207, 010 FR-010), so "planned cost by brand" needs an
   export to Excel.
4. **A fixed hierarchy.** The five planning levels and their names are fixed (domain model §1, D-006),
   while Hive9 let the tenant name its hierarchy. Comment 2 is only partly met.

**Proposed change**
- **(a) Reopen Q-009 / D-207 in 010.** Add Shell-attribute filters (with "Not filled in") and a
  **Group by** choice (any choice attribute) with totals. M.
- **(b) Category breakdown on the plan page.** Planned cost by spend type, brand and channel. This
  overlaps 013's SP&A view story and can be taken from it. S.
- **(c) Ask about category budgets (Q-017).** Ask Shell whether budgets must be *set* per category,
  which would need budget envelopes, or only *reported* by category, which (a) and (b) cover.
- **(d) Optional.** Configurable level labels. S.

**Size:** M for (a) and (b).

---

### 3.3 BUS.BUD.040: global/local budget allocation and visibility (Not met · L)

**Shell asks:** Global and local budget allocation, with appropriate visibility for global and local
budget owners respectively.

**Today in NorthStar**
- **Budget flows down one plan's hierarchy.** It goes from plan budget to campaign budget to initiative
  budget, with "Remaining plan/campaign budget". Over-allocation shows in red and never blocks (domain
  model §4.1; D-015; feature 005).
- **Ownership.** Plans have teams and an owner who must belong to one of them (R1). A plan can be tagged
  Strategic/AOP (`backend/internal/plans/model.go:74`).
- **No restriction on visibility.** Everyone sees every plan (D-004). "My teams" only narrows the list
  (010 Assumptions; D-035).

**What is missing**
1. **Two kinds of budget.** NorthStar has no **global** budget versus **local** budget, and no global or
   local budget owner.
2. **Allocation across plans.** A global owner cannot split a global amount into local (market) plans or
   see "allocated vs planned" per market.
3. **Visibility rules.** A global owner should see all markets; a local owner should see only their own
   market and allocation.

**Proposed change**
- **New feature 015 "Global and local budgets"**, after 014.
- **A global budget per year and business unit**, owned by a global budget owner, allocated to local
  market plans. Each local plan's budget becomes its allocation: read-only for the local owner, or shown
  as "requested vs allocated".
- **Two views:**
  - the global view shows allocated, planned and remaining per market, with the same over-allocation
    rule as D-015;
  - the local owner sees their own allocation only (014 scopes).
- **Every allocation change is logged.**
- **An alternative design:** a "global plan" level above market plans. Shell's answer to Q-011 decides
  which design.

**Size:** L.

**Depends on:** BUS.ADM.020.

**Question for Shell:** Q-011.

---

### 3.4 BUS.BUD.050: currency conversion, exchange rates, GSAP daily rates (Partly met · L)

**Shell asks:**
- currency conversion in planning, analysis and reporting;
- exchange-rate management;
- whether NorthStar can consume daily rates from GSAP.

The client adds: "Ability to plan in local currency and USD".

**Today in NorthStar**
- **Currencies and rates in Settings.** Settings keeps currencies with a rate "units per 1 USD", and
  previews the impact before a rate change (002 FR-028–FR-030;
  `backend/internal/settings/model.go:145-154`).
- **Line items can be typed in a local currency** and are converted to USD whenever they are read
  (007; D-024, D-141; `backend/internal/lineitems/model.go:38-39`).
- **Display currency.** Each person chooses a display currency, and the API converts every amount on
  screen and in exports, labelled with its currency (003 FR-023; D-061, D-213, D-220; 010 FR-017,
  FR-033).

**What is missing**
1. **Planning in local currency.** Plan, campaign and initiative budgets and the activity's planned cost
   are typed in **USD only**:
   - `backend/internal/plans/model.go:76`;
   - `backend/internal/activities/model.go:57`;
   - D-079, and D-220: "the campaign, initiative and activity forms keep USD".
   A marketer in the UK or China cannot type a budget or activity cost in GBP or CNY.
2. **Exchange-rate management.** There is **one current rate** per currency: no history, no effective
   dates, no planning rate vs daily rate (D-027; 002 Assumptions).
   - Every conversion uses today's rate (D-041, D-141). A rate change therefore silently changes the USD
     value of approved activities (domain model §2.5, last paragraph).
   - Hive9 itself separates "Actual Cost" from "Actual @ Monthly FX Cost"
     (`docs/reference/hive9-functional-analysis.md` line 1335).
3. **GSAP rates.** No GSAP feed: GSAP is out of scope (D-008), and GSAP integrations are frozen until
   about March 2027 (functional analysis §D1).

**Proposed change** (extends 002, with changes in 004–007 and 010)
1. **Dated rates.** Rates with an effective date and a **rate type**: a planning (budget) rate per year,
   and a daily rate.
2. **GSAP rate import now, feed later.** Import of the **GSAP daily rate extract** as a file now,
   following the workshop's freeze-period option of file exchange (functional analysis §D1). An
   automated feed comes after the freeze.
3. **Local-currency budgets and costs.** Local currency for budgets and the activity's planned cost,
   with USD shown beside the local amount everywhere it appears.
4. **A rate policy.** For example, plans use the year's planning rate so that approved values do not
   drift, while reports may use the daily rate.
5. **Decisions:** reopen D-010, D-027, D-041, D-079, D-141 and D-220.

**Size:** L.

**Question for Shell:** Q-012.

---

### 3.5 BUS.FIN.080: financial transactions in the main working views (Not met · L)

**Shell asks:** Make POs, invoices, payments and accruals prominent in the user's main working views,
not buried several levels deep. The client adds that it must be simple to reach and intuitive, and
refers to "the note on financial transactions", which is not in the PDF.

**Today in NorthStar**
- **Finance is out of scope for this phase** (D-008; `docs/product/vision-and-scope.md` "Out of scope
  now"; feature-map "Later").
- **No transaction exists in the data model.** NorthStar has no PO, invoice, payment or accrual records,
  and the sidebar has no Finance entry (`frontend/src/app/navigation.ts:45-89`).
- **The only trace is collapsed and empty.** The activity page and the line-item page each have a
  section **"From finance (coming later)"**. It is *closed by default* and every value shows "—":
  - `frontend/src/features/activities/ActivitySections.tsx:160-175`;
  - `frontend/src/features/line-items/FromFinanceSection.tsx:10-29`;
  - field catalogue §4.6 and §5 #17; 007 FR-025.

  This is the opposite of "prominent".
- **All work offers no finance columns on purpose** (010 FR-018).

**What is missing:** everything.
- Transaction records (PO, invoice, payment, accrual) linked to line items.
- A source for the data during the GSAP freeze.
- Figures shown prominently: committed, invoiced, paid, accrued and remaining against planned cost.
  These belong on the activity header, line-item rows, the plan tree and the default All work columns.
- A transactions list one click from each activity.

**Proposed change:** new feature **016 "Financial transactions"**, read-only in the MVP.
1. **Data model.** Transactions with type, number, date, amount, currency, vendor, SWBS code and line
   item. Load them by **file import of GSAP extracts** (workshop option, functional analysis §D1).
2. **Activity page.** A "Financials" card, **open by default**, with committed (PO), invoiced, paid,
   accrued and remaining against planned cost. It replaces the collapsed placeholder.
3. **Line items, plan tree and All work.** Line-item rows and the plan tree show committed and invoiced.
   All work gets "Committed / Invoiced / Paid / Accrued" columns and an "Actual" row in Budget & cost.
4. **Transactions page.** A list of transactions with filters, linked both ways to activities and line
   items.
5. **Visibility.** Financial data visibility comes from 014 (BRD: "RBAC for … POs, invoices").
6. **Decision:** reopen D-008 for read-only transactions. Live GSAP integration stays later (BUS.FIN.010,
   BUS.FIN.020).

**Size:** L.

**Depends on:** BUS.ADM.020 (who may see financial data) and BUS.BUD.050 (rates for transactions in
local currencies).

**Question for Shell:** Q-015.

---

### 3.6 BUS.PLN.020: multi-year, annual, quarterly, monthly; strategic, operational, cross-market (Partly met · L)

**Shell asks:**
- multi-year, annual, quarterly and monthly planning;
- strategic, operational and cross-market planning;
- future or custom development options.

**Today in NorthStar**
- **Annual planning.** A plan covers one fiscal year, which is the calendar year (domain model §1;
  D-011; `backend/internal/plans/model.go:70`). Plans can be made for the current year −1 to +5
  (`backend/internal/plans/validate.go:24`).
- **Monthly and quarterly planning (of costs).**
  - Line items are spread per month, with a quarterly display (007 FR-003, FR-017–FR-022).
  - All work Budget & cost shows months or quarters (010 FR-022).
  - The calendar has a timeline and a month view (010 FR-025, FR-028;
    `frontend/src/features/all-work/MonthCalendar.tsx`).
- **Beyond one year.** Activities may run up to 10 years past the plan's year (R5) or be ongoing (R6).
  Custom spreads may fall in other years (the "Other years" column, 010 FR-022).
- **Strategic.** A **Strategic/AOP** yes/no on the plan (field catalogue §1 #6;
  `backend/internal/plans/model.go:74`) and the SMP attribute (§1 #13). All work can filter
  Strategic/AOP plans (010 FR-004).
- **Cross-market.** Country/OU split on activities, Cluster/Region on campaigns, and All work across
  every plan of a year.
- **Custom development.** NorthStar is Shell's own code, with an OpenAPI-documented API (D-002) and
  configurable Settings.

**What is missing**
1. **Multi-year planning.**
   - A plan holds one year, and linking multi-year activities was left out (D-017).
   - All work shows **one year at a time**, with no multi-year totals (010 FR-004, Assumptions).
2. **Quarterly and monthly budgets.** Only costs are phased (see BUS.PLN.040).
3. **Strategic vs operational.** "AOP" is only a tag. Nothing links a strategic (multi-year) plan to the
   operational annual plans that deliver it.
4. **Cross-market.** No filters or grouping by market (D-207), and no roll-up across market plans.

**Proposed change**
- **(a) All work year range.** A "from–to" year range in All work, with multi-year totals and calendar
  (010). M.
- **(b) Strategic plan record.** A strategic plan spanning several years that links the annual plans
  delivering it and adds up their figures (004). M. Only if Shell confirms the need (Q-016); otherwise
  the AOP tag plus (a) may be enough.
- **(c) Market filters and grouping.** Shared with BUS.BUD.020 (a).
- **(d) Budget phasing.** Shared with BUS.PLN.040.
- **Decision:** reopen D-017.

**Size:** L in total, built as M pieces.

**Question for Shell:** Q-016.

---

### 3.7 BUS.PLN.040: split the budget by month; split an activity from its original budget (Partly met · M + M)

**Shell asks:** Users can split their budget by month, and split an activity from its original budget.
The client adds: "Phasing and ongoing updating of budget".

**Today in NorthStar**
- **Line-item costs are split by month**, with four methods: Evenly, All in first month, All in last
  month, Custom (domain model §4.4; 007 FR-017–FR-022; `backend/internal/lineitems/spread.go`).
  - Custom months may lie outside the activity's dates.
  - Spreads re-phase automatically when the activity's dates change (D-140).
  - Quarterly display exists, and All work Budget & cost adds months up across plans (010 FR-022).
- **Re-phasing is easy to keep up to date.** Changing a spread does **not** restart approval. Changing
  the planned cost does, after a confirmation (domain model §2.5; D-013).
- **Moving an activity.** An activity can be **moved** to another initiative *of the same plan* and
  copied (D-033; feature 006).

**What is missing**
1. **Budget phasing.** Plan, campaign and initiative **budgets** are one annual amount
   (`backend/internal/plans/model.go:76`; domain model §4.1). Only planned cost is phased. Hive9 offered
   "Budget Distribution" (evenly or by percentage, per month or quarter) and "Copy Planned Cost to
   Budget" (functional analysis §C2).
2. **Splitting an activity.** There is no **Split** action to divide one activity, and its line items,
   into two. Move is limited to the same plan, so an activity cannot be taken out of the plan or budget
   it was first planned under.
3. **No phasing history.** The phasing at approval time is not kept, so "approved phasing vs current
   phasing" cannot be shown.

**Proposed change**
- **(a) Budget phasing** (004/005, shown in 010).
  - Monthly or quarterly budget amounts on plan, campaign and initiative.
  - Distribute tools: evenly, by percentage, or copy from planned cost.
  - Budget vs planned per month in Budget & cost.
  - M.
- **(b) Split activity** (006/007).
  - Pick an amount or some line items to move into a new activity, in the same or another initiative.
  - Both activities record "Split from A-1001".
  - The re-approval rules of domain model §2.5 apply.
  - M.
- **(c) Widen Move.** Allow moving an activity to another plan of the same year (reopen D-033). S.

**Size:** M + M, plus S for (c).

**Question for Shell:** Q-013, what "split an activity from its original budget" means exactly.

---

### 3.8 BUS.PLN.050: re-use prior-year planning content (Met, small gaps · S)

**Shell asks:** Re-use prior-year planning content when starting a new planning cycle. The client adds:
"reuse the historical or prior year plans/inputs to save time from recreating from scratch".

**Today in NorthStar**
- **Copy plan into next year** (004 US6, FR-026–FR-028; `backend/internal/plans/copy.go:68`;
  `frontend/src/features/plans/dialogs/CopyPlanDialog.tsx`).
  - The copy is "Same year" or "Next year", with or without everything below it.
  - Dates move one year (29 February becomes 28 February).
  - Statuses reset to Draft and Created; no approvals or history are copied (R13, D-082).
  - Finance budget links move to the new year (D-148).
  - It is tested end to end ("copy a plan into next year", D-221).
- **Single copies** of campaigns, initiatives and activities, with their line items (005 FR-026,
  feature 006).

**Small gaps**
1. **No cross-year single copy.** One campaign or activity from last year's plan cannot be copied into
   this year's plan, because single copies stay within the same year (D-096). Bulk copy to any plan
   belongs to deferred feature 012.
2. **No "structure only" option.** A copy cannot keep only the skeleton, without planned costs, line
   items and amounts. The client's BUS.AI.010 comment gives exactly this example ("a skeleton for next
   year").
3. **No Hive9 history.** Historical Hive9 plans are not available to copy (BUS.ADM.050).

**Proposed change**
- **(1) Cross-year single copies** (005/006, revisit D-096). S.
- **(2) An "Include costs and line items" choice** in the copy dialog (004). S.
- **(3) Hive9 history** is handled by BUS.ADM.050.

**Size:** S.

---

### 3.9 BUS.UX.020: contextual help maintained by Shell administrators (Partly met · M)

**Shell asks:** Info icons (or equivalent) that explain what data an attribute or field expects,
maintainable by Shell administrators.

**Today in NorthStar**
- **Shell attributes have help that administrators maintain.** Each attribute has help text of up to
  300 characters, maintained in Settings and shown as an ⓘ tip on every form (002 FR-016, FR-024):
  - `backend/internal/settings/model.go:108`;
  - `frontend/src/features/settings/shell-attributes/ShellAttributeEditorPage.tsx:484-490`;
  - `frontend/src/components/ShellAttributeFields/ShellAttributeFields.tsx:202`.
- **Some calculated values have fixed ⓘ explanations written in the code:**
  - `frontend/src/features/plans/PlanSummaryCards.tsx:24-72`;
  - `frontend/src/features/campaigns/BudgetCost.tsx`;
  - `frontend/src/features/campaigns/RecordForm.tsx:201`;
  - `frontend/src/features/line-items/CostFields.tsx:84`.

**What is missing**
1. **Help for built-in fields.** None of the following has any help:
   - plan: plan name, year, model, teams, owner, plan budget, default finance budget, default line-item
     spread, BCPIR, goal stage;
   - campaign and initiative: dates and budget;
   - activity: activity type, dates, ongoing, planned cost, approval workflow;
   - line item: currency, planned cost, spread, finance budget year and item, "Request SWBS code".

   The plan, activity and line-item forms have no tip at all (`frontend/src/features/plans/PlanForm.tsx`,
   `frontend/src/features/activities/ActivityForm.tsx`, `frontend/src/features/line-items/LineItemForm.tsx`).
2. **Editable fixed tips.** The fixed tips cannot be changed by administrators.

**Proposed change** (extends 002)
- **A "Field help" Settings section.** One list of every built-in field per level, from the field
  catalogue, each with editable help text of up to 300 characters.
- **Good defaults.** The help is seeded with defaults, and changes are logged.
- **One way to read it.** Forms, details and summary cards read help from the API, using the same
  `InfoTip` component as today.
- **The import template too.** The 012 template's Instructions sheet can reuse the same text.

**Size:** M.

---

### 3.10 BUS.WFL.010: approvals (Partly met · L)

**Shell asks:**
- multi-level, parallel and conditional approvals;
- delegation of authority;
- automated notifications and escalations;
- role-based approvals and segregation of duties;
- compliance controls and an audit trail;
- workflow monitoring and process dashboards.

**The client simplifies this:**
- (1) "multi layered approval approach that is auto approval/manager approval — so approval
  flexibility";
- (2) "simplified between approval workflow + approval delegations/control + approval
  tracking/notification".

**Today in NorthStar, capability by capability**

| Capability | Status | Evidence | Gap |
|---|---|---|---|
| Multi-level approvals | Met | Ordered levels, each with one or more approvers; the last level approves the activity (008 FR-002, FR-004, FR-015; `backend/internal/approvals/model.go:20-35`) | — |
| Parallel approvals | Not met | Within a level, *any one* approver decides (D-205) | No "all must approve", no parallel levels |
| Conditional approvals | Not met | Every level applies to every submission; no thresholds (D-205, Q-002) | No amount or type conditions |
| Delegation of authority | Partly met | Anyone may approve "on behalf of" the designated approvers, and it is recorded (008 FR-017; `backend/internal/approvals/rounds.go:495`) | No formal delegate or out-of-office with dates |
| Automated notifications | Partly met | In-app notifications and bell, fetched every 30 seconds (008 FR-033–FR-036; D-160) | No e-mail or Teams |
| Escalations | Not met | Only a red "Start date passed" flag and the time waiting (`backend/internal/approvals/inbox.go:261`; 008 FR-030) | No reminders or escalation |
| Role-based approvals | Not met | Approvers are named people; no roles (D-020) | Needs 014 |
| Segregation of duties | Not met | The owner may approve their own activity (008 Edge cases); anyone may approve (FR-017) | No rule |
| Compliance controls | Partly met | Material changes (cost, dates, line-item amounts) restart approval (D-013, 008 FR-022); a round keeps a snapshot of its levels (FR-006); only the first decision counts (FR-018); a decline needs a comment (FR-016) | Controls depend on roles and SoD |
| Audit trail | Partly met | Approval history per round on the activity (008 FR-025), and every action in the activity log (FR-026, D-039) | The **Activity log page and record History (011) are not merged**, so the general log cannot be read in the app (`frontend/src/app/navigation.ts:89`, "Coming soon") |
| Workflow monitoring | Met | Approvals page with Waiting for me / All open / Decided by me, filters, time waiting, sidebar badge (008 FR-027–FR-032) | — |
| Process dashboards | Partly met | Four approval summary cards (008 FR-028) and Home's "Waiting for my approval" (009 FR-013) | No cycle-time or throughput figures |
| Auto approval vs manager approval (client) | Not met | No auto-approval; a person has no manager (`backend/internal/people/model.go:19-26`) | Both missing |

**Proposed change** (extends 008 with new user stories, "Approval rules"), following the client's three
simplified parts:
1. **Approval workflow (auto vs manager).**
   - A workflow can **auto-approve** when its conditions are met, for example planned cost at or below a
     threshold, or an activity type on a list. This is logged as "Auto-approved by rule …".
   - Otherwise it uses **manager approval**: a new approver type, "the activity owner's line manager".
     People gain a manager field (003), later taken from Entra ID.
   - Optional further levels apply only above an amount. This answers Q-002.
2. **Delegations and control.**
   - A person can name a **delegate** with from–to dates. Notifications and "Waiting for me" go to the
     delegate, and the history shows "X as delegate of Y".
   - A **segregation-of-duties** setting stops the owner or submitter from approving their own activity
     (enforced with 014).
   - **Role-based** approver levels come with 014.
3. **Tracking and notification.**
   - The approvals page exists.
   - Add **reminders** after N days waiting and **escalation** to the level's escalation contact or
     manager after M days.
   - Add cycle-time figures to the approvals cards.
   - Add e-mail (Outlook) as a second channel later, with BUS.WFL.040.
4. **Merge feature 011** so that the audit trail can be read (Activity log page, History on each record).
5. **Parallel approvals ("all approvers of a level must approve")** only if Shell still wants them after
   simplification (Q-014).

**Decisions:** reopen D-205 and D-160, and answer Q-002.

**Size:** L.

**Depends on:** BUS.ADM.020, for role-based approvals and for enforcing segregation of duties.

**Question for Shell:** Q-014.

---

### 3.11 BUS.WFL.030: highlight missing mandatory data and halt submission (Met · S)

**Shell asks:** Highlight missing mandatory data, and halt a submission until the required data is
completed.

**Today in NorthStar**
- **Two levels of "mandatory" (D-009):**
  - **Required** fields block saving, and there are very few of them.
  - **Needed to publish** and **Needed to submit** fields block publishing a plan or submitting an
    activity.
- **Submitting an activity runs a ready-to-submit check.** The check lists every problem at once, each
  linked to its field (008 FR-010; D-109; `frontend/src/features/activities/ReadyToSubmitPanel.tsx`).
  - Its scope: every empty needed field of the activity *and of all its line items*, every % split not
    at 100 %, every custom spread that does not add up, a negative unassigned balance, and an inactive
    workflow.
  - The API refuses the submission with the full list:
    - `backend/internal/activities/provided.go:166`;
    - `backend/internal/approvals/rounds.go:82-117`.
- **Publishing a plan is refused while needed fields are missing.** The refusal lists every missing
  field, grouped by record: plan, campaigns, initiatives (004 FR-024; D-076;
  `backend/internal/plans/status.go:33-50`;
  `frontend/src/features/plans/dialogs/PublishBlockedDialog.tsx`).
- **Every record shows a completeness chip:** "All filled in" or "n fields needed" (domain model §2.6).
- **Administrators choose which Shell attributes are needed** (002 FR-017).
- **Tested end to end** in the core-flow journeys (D-208).

**Small points to confirm (not gaps):**
1. The markings of the *built-in* fields are fixed by the field catalogue (Q-005 is still open).
2. A plan does not have to be published before its activities are submitted (D-206).

**Size:** S (confirmation with Shell only).

---

## 4. Other rows (Future-State/Later, RFP/Selection, Needs Clarification)

These are not MVP. Each gets a short assessment so that nothing is lost and the specs can mention them as
future integration points.

### 4.1 Future-State/Later (14)

| Requirement | Status | Today (evidence) | Gap and proposal | Size |
|---|---|---|---|---|
| BUS.AI.010 AI capabilities; future ready | Not met (by design) | AI is out of scope (vision and scope; PRISM: "foundations before AI"). The open REST API with OpenAPI contract (D-002) lets Shell-owned AI services read and write later. Next-year copy already creates next year's plan from this year's (004 FR-028) | The client's example, a skeleton for next year, is answered by the "structure only" copy of §3.8 | — |
| BUS.BUD.030 Overspend hard stop or warning; include underspend | Partly met | Over-allocation and planned cost above budget show in red but never block (D-015, D-088, D-176). Line items above the activity's planned cost block submitting (D-016) | Add an **underspend** warning (for example, planned cost below a set % of budget) and, if Shell wants it, a per-plan "warn or block" setting. The client says "not a blocker but a warning", which matches today's default | S–M |
| BUS.FIN.020 Full PR/PO automation | Not met | Finance and GSAP out of scope (D-008). Line items only record "Request SWBS code" and its status (domain model §2.4) | The client moved it to the technical track. Later Finance module, after the GSAP freeze | L (later) |
| BUS.FIN.040 Monthly accruals per PO | Not met | No POs or accruals | Later, on top of 016 | L (later) |
| BUS.FIN.050 Review and approve supplier invoices | Not met | No invoices | Later: invoices from 016 plus an approval rule from 008 | M–L (later) |
| BUS.FIN.060 PO and invoice documents linked to the PR | Not met | NorthStar stores no file attachments anywhere | Later: document links or attachments on transactions in 016 | M (later) |
| BUS.PRF.020 PIR linked to the business case / SP&A activity | Not met | PIR is a Yes/No attribute on activities (field catalogue §4.2 #18). 013 has stories for BCPIR and SP&A views but waits on Q-001 | Later: 013 after Q-001 | L (later) |
| BUS.PRF.030 Central view of all content, scoped by access | Partly met | Home (009) and All work (010) show plans, budgets, costs and approvals of every plan in one place | No KPIs or PIRs; no access scoping. The scoping in the client's comment comes with 014 | — |
| BUS.RPT.010 Standard dashboards | Partly met | Home summary cards and "Plans at a glance" (009), plan summary cards (004 FR-012), All work cards and Budget & cost (010) | No executive, regional or KPI dashboards; no business case vs forecast (no forecasts or actuals yet). Belongs to a later reporting (Measure) phase | L (later) |
| BUS.RPT.050 Calendar views by campaign, owner, period, country | Partly met | All work Calendar timeline and Month view, filtered by plan, owner, team, business unit and activity type (010 FR-004, FR-025, FR-028) | Not by country, and no grouping. Comes with the attribute filters of §3.2 (a) | S after BUD.020 |
| BUS.UX.010 Web/mobile, accessibility, languages, Excel, bulk, dashboards, notifications, search | Partly met | Responsive web with a mobile navigation drawer (`frontend/src/app/NavDrawer.tsx`; design system §9). Automated WCAG 2.1 AA checks in the end-to-end tests (D-221). Excel/CSV export (010 FR-032; D-120), personal dashboard (Home), in-app notifications (008), search in lists and All work | No offline use or native mobile app; English only; no bulk upload or editing (012 deferred); no global search | L (later) |
| BUS.WFL.040 Notify owners of changes; customisable | Not met | Notifications exist only for approval events (`backend/internal/approvals/model.go:116-124`) | 008: notify owners of changes to plans, budgets and activities they own, with per-person preferences (which events, in-app or e-mail) | M |
| BUS.WS.01 Export plans to ppt/pdf/word/excel; share to Outlook/Teams | Partly met | All work exports .xlsx and .csv with an "About this export" sheet (010 FR-032–FR-034). Printing and PDF are explicitly out of scope (010 Assumptions) | Export of one plan to PDF and .xlsx from the plan page, and a "copy link / share" action. PowerPoint/Word and Teams sharing later | M |
| BUS.WS.02 Import a plan skeleton / upload an updated plan from Excel | Not met | 012 (deferred) specifies importing *new* activities into one plan, optionally creating campaigns and initiatives (012 FR-024–FR-034). Settings can import attribute options (002 FR-031) | Build 012's import, and add a **round-trip**: export a plan, edit it in Excel, re-import to update by ID. 012 creates only, and imports no line items | L |

### 4.2 RFP/Selection Requirement (6)

| Requirement | Status | Today (evidence) | Gap and proposal | Size |
|---|---|---|---|---|
| BUS.ADM.010 Configuration without the vendor (DIY) | Partly met | Administrators maintain, live and without a release, all of the following (002 FR-007, SC-008): models, activity types, Shell attributes and options (with import), currencies and rates, finance budget items, approval workflows, people and teams. Every change is logged | Missing: role and permission administration (014), a readable audit page (011 not merged), field help (§3.9). Release management is not a business setting | Covered by 014, 011 and §3.9 |
| BUS.ADM.050 Migrate current and historical Hive9 data | Not met | No migration exists. New plans are limited to the current year −1 … +5 (`backend/internal/plans/validate.go:24`; `backend/internal/plans/model.go:249`). Hive9's data model and API are documented in `docs/reference/hive9-functional-analysis.md` | New feature: a Hive9 extract mapped onto plans, campaigns, initiatives, activities and line items. Earlier years imported read-only. Shell attribute values mapped to options | L |
| BUS.FIN.010 GSAP integration | Not met (technical) | Out of scope (D-008). GSAP frozen until about March 2027 | The client moved it to the technical track. The file-exchange option (§D1 of the functional analysis) is reused by §3.4 and §3.5 | — |
| BUS.INT.010 Standard integrations | Not applicable (technical) | No connectors. The API is REST with an OpenAPI contract (D-002); All work exports to Excel/CSV | The client moved it to the technical track | — |
| BUS.WS.03 Multiple time zones; offices | Not applicable (vendor question) | Times are stored in UTC and shown in each person's own time zone (D-166, D-179) | Vendor and support-model decision, still pending at Shell | — |
| BUS.WS.04 Keep 5 years of data to refer back to | Partly met | Nothing is purged. The activity log is append-only (D-039); deleted records stay in the recycle bin until purged | No Hive9 history, and older years cannot be created. Solve with BUS.ADM.050 (read-only archive years) | M (with ADM.050) |

### 4.3 Needs Clarification (5)

| Requirement | Status | Today (evidence) | Comment | Size |
|---|---|---|---|---|
| BUS.BUD.010 Ownership, approvals, transfers, locking, freeze, multi-currency, carry-forward (client: duplicate) | Partly met | Owners at every level (field catalogue) and activity approvals (008) exist. Multi-currency only on line items (§3.4). A next-year copy keeps budgets (004 Assumptions) | No budget transfer action (budgets are edited by hand). No locking or freeze: a Published plan stays editable (domain model §2.1). A "lock plan" option could follow the roles work | S–M |
| BUS.PLN.010 Global template, standard hierarchy and attributes (client: captured in MVP) | Partly met | Standard five-level hierarchy, configurable attributes and models (002) | No "global template" record. Copying a template plan is the workaround. Covered by §3.2 and §3.3 | — |
| BUS.PLN.030 Rolling forecasts, phasing, allocation, scenarios (client: duplicate) | Partly met | Cost phasing (007), budget allocation down the hierarchy (005) | No rolling forecasts or scenarios. Later | — |
| BUS.PLN.060 In-cell formulas (client: not required) | Not applicable | The client says it is Hive9-specific and not required. NorthStar calculates roll-ups, goals and conversions automatically (domain model §4) | None | — |
| BUS.RPT.060 Filters and saved filter sets (client: duplicate) | Met (mostly) | All work filters, search, personal saved views and a default view (010 FR-004–FR-010, FR-029–FR-031); Plans list filters remembered (D-081); approvals filters (008 FR-031) | No Shell-attribute filters (§3.2), and saved views are personal only | — |

---

## 5. Decisions to reopen

The BRD asks for more than these recorded decisions allow. Each one should be reopened and replaced by a
new decision once Shell answers the matching question in §6.

| Decision | What it says today | BRD rows | Proposal |
|---|---|---|---|
| D-004, D-020, Q-006 | No roles: everyone can do and approve everything; plans visible to all | ADM.020, WFL.010, BUD.040, FIN.080, PRF.030 | Replace with feature 014 Roles and access |
| D-008 | Finance, GSAP and Measure out of scope | FIN.080, BUD.050 | Bring **read-only transactions** and **rate import** (both by file) into the MVP; live GSAP stays later |
| D-010, D-079, D-220 | Budgets and activity planned cost in USD; forms keep USD | BUD.050 | Allow a local currency, with USD shown beside it |
| D-027, D-041, D-141 | One current rate; every conversion uses today's rate | BUD.050 | Dated rates with a planning rate and a daily rate; set a rate policy |
| D-017 | Multi-year activities out of this phase | PLN.020 | Revisit after Q-016 |
| D-033, D-096 | Move within the same plan; single copies within the same year | PLN.040, PLN.050 | Allow cross-year single copies and moving activities to another plan |
| D-205 (Q-002), D-160 | Ordered levels, any one approver, no thresholds; in-app notifications only | WFL.010, WFL.040 | Approval rules (auto vs manager, conditions, delegates, escalation, SoD); e-mail later |
| D-207 (Q-009) | No Shell-attribute filters in All work | BUD.020, PLN.020, RPT.050, RPT.060 | Attribute filters and grouping with totals |
| D-208, D-209 | Feature 011 set aside, not merged | WFL.010 (audit trail), ADM.010 | Merge 011 |

---

## 6. Questions for Shell

Proposed new open questions, numbered after the decision log's Q-009. Existing questions that the BRD
makes urgent are listed at the end.

| ID | Question | Blocks |
|---|---|---|
| Q-010 | **Roles and scopes.** Which roles does Shell want? What may each view and edit, at which level, including financial data? Should scoping be by team, by market (Business unit, Cluster, Country/OU) or both? Can we start from about six roles mapped from Hive9's 20 (functional analysis §F4)? | ADM.020 |
| Q-011 | **Global vs local.** What is a global budget and a local budget in Shell's process? Does a global owner allocate money top-down to market plans, or do markets request and global approve? Which figures does each owner see and edit? | BUD.040 |
| Q-012 | **Exchange rates.** Which rate converts a plan: a fixed annual planning (budget) rate, or the daily rate? Must approved activities keep their USD value when rates change? What is the format of the GSAP daily rate extract? Which levels must accept a local currency (plan budget, campaign budget, activity planned cost)? | BUD.050 |
| Q-013 | **Splitting.** Does "split an activity from its original budget" mean dividing one activity into two, or moving an activity (or part of it) to another budget, plan or finance budget item? | PLN.040 |
| Q-014 | **Approvals, simplified.** What are the criteria for auto-approval (an amount threshold, activity types)? Where does "manager" come from (Entra ID manager attribute, team leader)? Are parallel approvals still needed? What reminder and escalation timings apply? What segregation-of-duties rule applies (for example, owner never approves own activity)? | WFL.010 |
| Q-015 | **Financial transactions.** Which transaction types and fields should users see, and where does the data come from during the GSAP freeze (file extracts, how often)? Please share "the note on financial transactions" referred to in BUS.FIN.080. | FIN.080 |
| Q-016 | **Multi-year and strategic plans.** What are a strategic plan and a multi-year plan in Shell's process? Is the Strategic/AOP tag plus a multi-year view enough, or must strategic plans link the annual plans that deliver them? | PLN.020 |
| Q-017 | **Budget categories.** Must budgets be *set* per category (for example a budget per brand or channel), or only *reported* by category? | BUD.020 |
| Q-002 *(existing)* | Real approval workflows: levels, approvers, thresholds | WFL.010 |
| Q-005 *(existing)* | Which built-in fields are "needed to publish/submit" | WFL.030 |
| Q-006 *(existing)* | Hide plans from people outside the plan's teams | ADM.020, BUD.040 |
| Q-009 *(existing)* | Filters by Shell attribute in All work | BUD.020, PLN.020, RPT.050 |

---

## 7. Recommended order of work for closing the MVP gaps

The order follows the dependencies, delivers visible improvements early and keeps three lanes busy in
parallel. Each item follows the usual Spec Kit steps:
1. update the spec, or write a new one;
2. clarify, plan and write the tasks;
3. build it, in the API and the web app;
4. test it: Go tests, unit and component tests, and an end-to-end journey with the accessibility check;
5. record the decision in the decision log.

**Step 0: now, no code (about 1 week)**
1. **Workshop with Shell.** Answer Q-010 to Q-017, plus Q-002, Q-005, Q-006 and Q-009. Get the "note on
   financial transactions". Record the answers as decisions, replacing the ones in §5.
2. **Merge feature 011.** It gives the audit trail that BUS.WFL.010 and BUS.ADM.010 ask for, and
   unblocks Home's "Recent changes" card.

**Step 1: quick wins in parallel lanes (about 2 weeks; start straight away, as they need few answers)**

| Lane | Work | Requirement | Size |
|---|---|---|---|
| A | Field help in Settings for every built-in field (002) | UX.020 | M |
| B | Shell-attribute filters and "group by attribute" with totals in All work, and a spend breakdown on the plan page (010, 004) | BUD.020; helps PLN.020, RPT.050, RPT.060 | M |
| C | Cross-year single copies and a "structure only" plan copy (004–006); confirm field markings; optional underspend warning | PLN.050, WFL.030, BUD.030 | S + S (+ S) |

**Step 2: the foundation (about 4–5 weeks; starts once Q-010 is answered, alongside step 1)**
- **Lane A, after field help:** feature **014 Roles and access**. These items depend on it: global/local
  visibility, role-based approvals and segregation of duties, and who may see financial data. (BUS.ADM.020, L)

**Step 3: built on roles (about 4–5 weeks, two lanes)**
- **Lane B:** **008 Approval rules**: auto vs manager approval, conditional levels, delegates, reminders
  and escalation, segregation of duties. (BUS.WFL.010, L)
- **Lane C:** **015 Global and local budgets**. (BUS.BUD.040, L)

**Step 4: planning depth (about 3 weeks; can overlap step 3 if a lane is free)**
- Budget phasing by month or quarter with distribute tools, and the **Split activity** action; widen
  Move. (BUS.PLN.040, M + M + S)
- All work year range and multi-year totals; a strategic plan link if Q-016 asks for it. (BUS.PLN.020, M)

**Step 5: money (about 6–8 weeks, two lanes)**
- **Lane A:** dated exchange rates, GSAP rate file import, local-currency budgets and costs.
  (BUS.BUD.050, L)
- **Lane B:** feature **016 Financial transactions**, read-only, from GSAP extract files.
  (BUS.FIN.080, L)
  - Its data model and file import can start in step 3.
  - Its screens are finished once rates (step 5, lane A) and roles (step 2) are in place, because
    transactions come in local currencies and their visibility is a role rule.

**Why this order**
- **Roles block four other items.** Building them first avoids reworking global/local visibility,
  role-based approvals, segregation of duties and financial-data access.
- **The quick wins come early and are visible.** Field help, filters and copy improvements need almost no
  answers from Shell, and the client team can see them in the next demo.
- **Rates come before transactions.** Exchange rates must exist before financial transactions in local
  currencies can be shown and added up correctly.
- **The biggest items wait for answers.** Global/local, approvals and finance are the biggest and least
  defined items, so they start after Shell's answers instead of on guesses.

**Not needed for the MVP, according to this BRD:**
- bulk actions and import (012), Future-State as BUS.WS.02 and BUS.UX.010;
- BCPIR/PIR views (013), Future-State as BUS.PRF.020;
- the live GSAP integration and PR/PO automation, which are on the technical track.

---

## 8. Sources

- **Client requirements:** `C:\Users\AK121426\Downloads\Business Requirements.pdf`, transcribed in
  [../reference/brd-requirements.md](../reference/brd-requirements.md).
- **Decisions and scope:**
  - [../sdd/decision-log.md](../sdd/decision-log.md) (D-001–D-221, Q-001–Q-009);
  - [../sdd/feature-map.md](../sdd/feature-map.md);
  - [../architecture/adr/0005-demo-sign-in-rbac-later.md](../architecture/adr/0005-demo-sign-in-rbac-later.md).
- **Product:**
  - [../product/vision-and-scope.md](../product/vision-and-scope.md);
  - [../product/domain-model.md](../product/domain-model.md);
  - [../product/field-catalogue.md](../product/field-catalogue.md).
- **Specs:**
  - `specs/002-settings-and-shell-attributes/spec.md` (FR-016, FR-024, FR-028–FR-030);
  - `specs/004-plans/spec.md` (FR-024–FR-032);
  - `specs/007-line-items/spec.md` (FR-003, FR-017–FR-025);
  - `specs/008-activity-approvals/spec.md` (FR-001–FR-037, Edge cases, Assumptions);
  - `specs/010-all-work-views/spec.md` (FR-004–FR-034, Assumptions);
  - `specs/012-bulk-actions-and-import/spec.md` (FR-024–FR-034);
  - `specs/013-bcpir-and-spa/spec.md`.
- **Hive9 reference:** `docs/reference/hive9-functional-analysis.md` §B1, §C2–§C6, §D1, §F4.
- **Code:** the files cited in each section, under `backend/internal/` and `frontend/src/`.
- **Tests:** `frontend/e2e/core-flow.spec.ts`, `frontend/e2e/module-rest.spec.ts`,
  `frontend/e2e/smoke.spec.ts`.
- **Repository state:** branch `plan-module` at commit `e4966aa`, read with read-only git commands; no
  files other than this report and the BRD transcription were changed.
