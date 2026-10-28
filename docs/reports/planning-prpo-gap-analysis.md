# MAPP Planning Module and PR/PO: Gap Analysis

<p class="cover-meta">Client target (PRISM SP&A deck, SP&A Campaign workflow, context specs, Hive9 Functional Analysis) compared with MAPP as built · 27 October 2026 · Scope: Planning module and the PR/PO processes attached to it</p>

> **Bottom line.** MAPP covers the planning hierarchy well: plans, campaigns, initiatives, activities, line items, activity approvals, All work views, the activity log and the recycle bin. The client's end-to-end SP&A workflow stops in MAPP at "activity approved". The steps after it are missing or only partly there:
> - SWBS request and return
> - the Purchase Request (PR) form and its approval
> - GSAP PR and PO numbers coming back
> - invoice approval and the Service Entry
> - PR modification and top-up
>
> Some of these are not built because our constitution and decision log keep purchase requests, e-mail and live GSAP out of this phase. Building them is a **scope decision first and an engineering task second**.
>
> **The Hive9 Functional Analysis (FA) sharpens the picture:**
> - **Hive9 never sends a PR to GSAP.** The approved PR form is e-mailed to a requisitioner and typed into GSAP by hand. The only file that goes out is the daily SWBS request. Only data *coming back* (SWBS code, PR and PO numbers, invoices) needs GSAP.
> - **GSAP integrations are frozen until about March 2027** (Shell's S/4HANA "Nucleus" cut-over, January 2027), and **we have no GSAP access**. Everything that needs GSAP is **on hold** (§6.1). The existing simulated GSAP feeds stay as they are.
> - **A lot can go ahead without GSAP:** the PR form and its approval in MAPP (once in scope), SWBS codes entered by Finance, exchange rates (Hive9 keeps them itself), the HX key, Shell's real master data, and the planning-rule fixes.
>
> Inside the planning scope, about fifteen business rules work differently from Hive9 and the client specs. The biggest are:
> - approvals tied to activities only, with no record-owner or record-field approvers;
> - line items added to an approved activity restart its approval, which breaks Shell's as-is order;
> - campaign and initiative share one permission, so four Hive9 roles cannot be built;
> - exchange rates, budget hard stop, Latest Estimate;
> - **seed values that are made up**: GL codes, the Zero-Based Budgeting cost hierarchy and most lists.

---

## 1. Executive summary

### Headline numbers

We assessed **442 items**: requirements, or tight groups of requirements, from the 16 client specs in scope, plus the 9 steps of the SP&A workflow.

| Status | Items | Share | Meaning |
|---|---|---|---|
| DONE | 88 | 20% | Built and matches the client spec |
| PARTIAL | 110 | 25% | Built in part; the gap is named |
| MISSING | 120 | 27% | Not built; 67 of these are purchase-request requirements |
| DIFFERENT | 65 | 15% | Built, but the business rule or flow differs from the client spec |
| DEFERRED | 58 | 13% | Left out on purpose by a recorded decision or the scope rules |

Without the purchase-request spec (client 013), the planning side is about **24% done, 28% partial, 19% missing, 17% different and 12% deferred**. The core is solid. What remains is finishing work and aligning the rules.

The **Hive9 Functional Analysis** (FA, 24 Sep 2026) was then reviewed section by section: hierarchy and forms, statuses and approvals, finance and PR/GSAP, and administration and roles. It added **more than 40 new gaps** and **about 25 corrections** to this report. They are marked "FA" in the tables below, with the FA section (for example FA B6). The counts above come from the client specs and are unchanged.

> **How this fits with the internal build-status note ("core build done, 1–2 weeks before UAT").** The two documents measure different things, and both hold.
> - **The status note** measures MAPP against **our own specs**. It is accurate: the build is clean, all 36 test packages pass, and the open items it lists are confirmed in §8. The 1–2 weeks covers that current-scope finishing work.
> - **This report** measures MAPP against **the client's target**: deck, workflow, context specs and the FA. The PR/PO process, the planning-rule differences (§5–6) and the scope decisions (§9) are **not in that 1–2 week estimate**. They need client answers (§11) and, for PR/PO, a change of scope before they can be planned.

> **GSAP constraint.** We have no access to GSAP: no test system, no file drop and no file layouts (the FA rates its own knowledge of the layouts "Low"). GSAP integrations are also frozen until about March 2027, and may be redesigned after S/4HANA goes live (FA D1, D9).
> - Work that depends on GSAP is **on hold** until access is provided, whatever the client's priority.
> - Today MAPP reads GSAP purchase orders, invoices, payments, accruals and exchange rates from **simulated** files behind adapters (ADR-0009). That stays as it is and keeps the demo working. Nothing new is built against guessed GSAP layouts.
> - The FA's own freeze-period remedies need no GSAP connection: Finance loads SWBS codes it created by hand, and GSAP extracts are matched on the HX key once their layout is known (FA D8).
> - §6.1 lists what is on hold and what can go ahead.

### The things that matter most

| # | Finding | Type | Priority |
|---|---|---|---|
| 1 | **Purchase requests do not exist in MAPP.** There is no PR form, approval, routing, close-out or modification. In Hive9 the PR is a form raised from a line item, approved in Hive9 (self-approval allowed), e-mailed to a requisitioner and retyped into GSAP (FA C4, D3). None of that needs GSAP; only the PR and PO numbers coming back do (**on hold**). | MISSING (scope) | High |
| 2 | **The approval engine handles Activities only** (`Round.ActivityID`). PR, modified PR, accrual, campaign and initiative group approvals, BC/PIR and budget increases cannot use it. It also lacks two approver types Hive9 uses: **Record Owner** and **a person taken from a field on the record**. The PR's MOA approver needs the second (FA C6). Hive9 has **no plan approval**: Draft → Published only. | DIFFERENT | High |
| 3 | **(FA) Adding a line item to an approved activity restarts its approval.** In Shell's as-is flow, line items are added *after* the activity is approved, and then the SWBS code and PR follow (FA A4, D3). In MAPP every added, duplicated, deleted or re-costed line item sends the activity back to Created, even when its planned cost does not change. That stalls SWBS readiness and any future PR. | DIFFERENT | High |
| 4 | **SWBS codes.** Line items show a derived status only. Nothing is sent and no code is stored (D-255). The automatic round trip (request file and code back) is **on hold (GSAP)**. Storing a code that **Finance enters or uploads** needs no GSAP: Finance can create codes by hand in GSAP (FA D7, D8). | PARTIAL | High (manual entry) · On hold (round trip) |
| 5 | **(FA) HX key: never shown, and will collide.** "HX" + line-item ID is the *only* link from GSAP back to a line item; the requisitioner types it by hand (FA D5, G8). MAPP never shows it on a line item. MAPP also numbers line items from 1001 (`platform/ids/ids.go:67`), while Hive9's IDs reach 1,248,388. Open Hive9-era POs carrying legacy HX keys would match the wrong MAPP line items. | MISSING | High |
| 6 | **(FA) Campaign and initiative share one permission** (`campaigns.edit`, `campaigns.delete` in `authz/catalogue.go`). That makes four of Hive9's roles impossible to build: Marketer, Mkg. Admin support, IT Admin and VP/GM, about 50 users (FA F4). | DIFFERENT | High |
| 7 | **(FA) Seed master data is made up.** Three of the five seeded GL codes (7250010, 7250040, 7250090) are not among Shell's 13. Spend Type has 5 values (Shell: 57), Spend Category 3 (19), Contracted Spend 3 (5), and Cluster, Region, Function and Focus Area are invented. These values feed the SWBS request. The FA lists the real values (E3.0.2). | DIFFERENT | High (quick fix) |
| 8 | **Exchange rates differ.** Hive9 keeps an annual planned rate per year plus 12 monthly rates per currency **in its own Currency screen**, with per-currency switches for monthly rates on actuals and on PRs (FA F4, F6). MAPP converts every amount with one current dated rate. **This needs no GSAP.** | DIFFERENT | High |
| 9 | **There is no budget hard stop or budget release.** In Hive9 the hard limit is a setting on the **Finance Budget row** the line item links to, not on the Plan Budget (FA C2, B7). So it belongs to the Finance Budgets module, which is out of scope today. | MISSING (scope) | High |
| 10 | **Latest Estimate is not modelled.** Hive9 has LE1–LE4 (typed) and Locked LE1–LE4 (read-only) as single amounts on the Activity (FA B6). The deck relies on "Planned cost (LE) vs PO vs actuals". | DEFERRED | High |
| 11 | **Bulk actions and import (our 012) are not built.** All 11 operations are designed and in the contract but not served. Hive9's bulk menu also has Change Status, Link To, Package/Unpackage and Create Purchase Request, and no Delete (FA B8). | MISSING | High |
| 12 | **The approval rules differ.** Re-approval runs on more triggers. Self-approval is barred org-wide by a switch that is on by default (Hive9 allows it for PRs). An admin can decide "on behalf". Re-Submit from Submitted is not allowed (confirmed live in Hive9, FA Figure 7). | DIFFERENT | High |
| 13 | **The Zero-Based Budgeting "cost hierarchy" has the wrong values.** The seed has Media, Agency, Events, Production. Live Hive9 has **10 values, 7 "LUBES –" and 3 "RETAIL –"** (FA E3.0.2). The deck lists 7 differently worded categories. Shell must choose. | DIFFERENT | High (quick fix) |
| 14 | **BC and PIR records.** The deck shows Business Case and PIR with approval, attachments and PPT export. **Hive9 itself has only the "BCPIR Plan?" flag and a "PIR" Yes/No on the activity, and MAPP has both.** BC/PIR records are a *new* deck requirement, not a parity gap. Our SP&A/BCPIR views (013) are Draft. | MISSING (new) | Med |

---

## 2. Scope and method

**What we read:**
- **The PRISM SP&A deck** (19 slides): text, tables, notes and embedded images. The slide 11 PR flow diagram was rebuilt from its vector drawing.
- **The SP&A Campaign workflow image** (HD).
- **The Hive9 Functional Analysis** (`context/Hive9_Functional_Analysis 2.docx`; Zensar, 24 Sep 2026).
  - It covers parts A–D and F–H and Appendices 1–3, with 8 screenshots. Part E (Measure) was not reviewed, except its attribute value lists (E3.0.2).
  - **Sources:** the live Hive9 10.9.0 sandbox, the Hive9 User & Admin Guide, and the GSAP–Hive9 integration workshop with Shell (10 Sep 2026).
  - **Evidence labels:** Live · Guide · Workshop · Inferred.
  - **Confidence (FA Appendix 1):** Plan forms and fields, and the 20-role matrix, are High. Organizational Setup and Models are Low–Medium (guide only). Behaviour under each role is Low (one account was used). GSAP file layouts are Low.
- **All 26 client specs**, with these 16 in scope:

| In scope (assessed) | Out of scope (not assessed) |
|---|---|
| 001 App shell · 002 Configuration · 003 People and teams · 004 Plan hierarchy · 005 Line items and cost allocation · 006 BCPIR/SP&A plans · 007 Workspace views · 008 Bulk, import, export · 009 Approval workflows · 010 Audit log and recycle bin · 011 Finance budgets (only where they feed plans) · 012 Transactions and actuals · 013 Purchase requests · 014 Accruals (only the PO-linked parts) · 015 GSAP integration · 021 Notifications (only planning and approval events) · 022 Identity and access · 026 Activity work board | 016–020 Measure (reports, dashboards, attribution) · 023 AI assistant · 024 Budget optimizer · 025 Self-service dashboard builder · FA part E (Measure) |

**What we compared against:**
- **Our specs:** all 29 features in `specs/`, including their open tasks.
- **Our project docs:** `docs/sdd/feature-map.md` and `decision-log.md`, the product docs (domain model, field catalogue), and the ADRs.
- **The code**, checked directly: `backend/internal/*`, `backend/api/openapi.yaml` (176 paths, 234 operations), migrations, seed, and `frontend/src/features/*`.

Five reviewers covered the client specs and four more covered the FA. Each cited file evidence for every status. Key claims were spot-checked in the code (see the Appendix).

**Status definitions:**
- **DONE**: built and matches the client spec.
- **PARTIAL**: built in part.
- **MISSING**: not built.
- **DIFFERENT**: built, but the rule or flow differs.
- **DEFERRED**: excluded by a recorded decision (D-number) or by the scope rules in the constitution and CLAUDE.md.

**On hold (GSAP)** is a planning flag, not a status. The status still describes the code. The flag marks work that cannot go ahead because we have no GSAP access. The counts in §1 and §4 are unchanged.

> **Watch out:** the client specs use their own decision numbers (D-008, D-010, D-042 and so on), which **do not match our decision log**. For example, their D-010 is the HX key, while ours is the base currency. Agree which log is authoritative before writing new specs.

---

## 3. The client's target process

### 3.1 SP&A workflow (Slide 7, seven swimlanes)

**The swimlanes:** VP/GM · Marketing Team Lead · Marketer or Vendor · Hive9/GSAP Scheduler · PR Requisitioner (TA/Manila) · PR/GR Approver · Account Payment Team.

**Colour key:** green = Hive9, yellow = GSAP, dark = outside.

![SP&A Campaign workflow, client slide 7](../../context/Slide%207%20-%20SP%26A%20Campaign%20workflow%20(HD).png)

Steps marked **on hold** need GSAP and wait for access (§6.1). Steps marked **never MAPP** happen in GSAP and stay there.

| # | Step (client workflow) | Lane / system | MAPP support | Evidence or gap |
|---|---|---|---|---|
| 0 | Create/allocate finance budget → Create plan → Create campaign/initiative → Create/modify activity and line items → activity/line item approval workflow | VP/GM, Team Lead, Marketer · Hive9 | PARTIAL | Global budgets set plan budgets (`internal/budget`); plan hierarchy (004–007); activity approval with levels, conditions, recall and delegation (008, 017). **(FA)** Two problems. First, line items added to an approved activity restart its approval, against the as-is order (FA A4, D3). Second, the campaign "Statuses & Approvals" and initiative "SMP Initiative – Footer" group approvals are absent (FA B4, B9). |
| 1 | Approved → **SWBS request to GSAP**; GSAP returns the SWBS code (daily file FSSI 1523, 24–48 h) | Hive9/GSAP scheduler | PARTIAL · round trip **on hold** | A "Request SWBS code" flag (ticked by default, as in Hive9) plus a derived status (`lineitem/completeness.go` `SwbsStatusOf`). No outbound file and no stored code (D-255). **Storing a code that Finance enters needs no GSAP** (FA D8). |
| 2 | Marketer **raises the PR form** from the line item: Standard PR, Manual PR or FI Payment | Marketer · Hive9 | MISSING | No PR record (D-243; constitution scope). Doesn't need GSAP. In Hive9 a PR can be created in any activity status but submitted only once the activity is approved (FA C4). |
| 3 | **PR approval workflow in Hive9** by the MOA approver; self-approval allowed; a decline goes back to the marketer | Team Lead / MOA · Hive9 | MISSING | The approval engine is tied to activities and cannot take the approver from a field on the PR (FA C6). |
| 4 | Approved → **e-mail with the PR form (Excel)** to the requisitioner's mailbox. FI Payment and Manual PR branches. | Requisitioner · e-mail | MISSING | E-mail is out of scope. In-app routing plus a downloadable PR form needs no GSAP. The Manual PR and FI Payment branches happen in GSAP and outside MAPP. |
| 5 | Requisitioner **retypes the PR into GSAP** with the HX key → GSAP **PR number** returned → **PR approval in GSAP** by the MOA | Requisitioner, MOA · GSAP | MISSING · PR number back **on hold** · approval **never MAPP** | The feed has no GSAP PR number. **The GSAP approval and PO release are GSAP's job and never MAPP work** (FA D7). MAPP doesn't show the HX key the requisitioner must type (FA G8). |
| 6 | **PO created** (automatically with an outline agreement, otherwise by the CP buyer) and **released** (≥ $50,000 needs the CP lead; set in GSAP) → **PO number and PO copy** returned (PGSI 941) → PO sent to the vendor | GSAP, outside | PARTIAL · new fields **on hold** | PO lines arrive through the simulated CSV and are matched by HX key, showing as *Committed* (015). That stays as it is. PO status, PO copy link and the link to a PR need the real feed. Release thresholds live in GSAP; MAPP needs nothing for them (FA D7). |
| 7 | Vendor invoices → AP processes → **invoice and link back** (PGSI 941) → **marketer approves or rejects the invoice** → requisitioner releases the **Service Entry** → payment | Vendor, AP, Marketer, Requisitioner | PARTIAL · **on hold** | Invoices and credits arrive as *Invoiced* from the simulated feed. The invoice link, the approve/reject (BUS.FIN.050 Later) and GR/SE need GSAP. **(FA)** The Service Entry comes **after** the invoice approval, not before (FA D3 stage 4). A manual "work delivered" confirmation could stand in during the freeze (FA D8). |
| 8 | Payment → **actuals** reach the line item and roll up (daily) | AP · GSAP | PARTIAL · **on hold** | The Payment type shows as *Paid* (= actuals, D-243), loaded by hand from a simulated file (ADR-0009). A daily feed needs GSAP. |
| 9 | **Modify/top up a PR**: a modification approval in Hive9, then the GSAP checks, the top-up in GSAP and a modified PO | Marketer, Requisitioner, CP | MISSING · GSAP part **on hold** | No PR, so no modification round. The modification approval in MAPP needs no GSAP. **(FA)** "Released" is not a Hive9 PR status; it almost certainly means the GSAP PR was released (FA D3 2.2). A modification in MAPP starts from **Approved**. |

### 3.2 The PR as Hive9 really does it (FA C4, D2–D7)

**Actors:**
- **Marketer:** raises the PR form, shares the PO, approves the invoice.
- **MOA approver:** approves twice, in Hive9 and again in GSAP, where it counts.
- **Requisitioner:** the TA role, 6 users, view-only in Hive9. Retypes the form into GSAP and releases the Service Entry.
- **SC analyst / CP buyer:** builds the PO when there is no outline agreement.
- **CP lead:** releases POs of $50,000 or more ($100,000 in other ERPs; set in GSAP).
- **Accounts Payable:** processes invoices and payment.

**Interfaces:**
- **FSSI 1523** (Hive9 → GSAP, daily CSV): **SWBS requests only.**
- **PGSI 941** (GSAP → Hive9, daily CSV): the GSAP PR number, PO number, date, amount and copy link, the invoice amount and link, and the service entry date. It is matched **only on "HX" + line-item ID**, typed by hand.
- There is no REST API either way.

**Keys:**
- line-item ID, which matches the SWBS response;
- the SWBS code: one per line item, tied to one company code; GSAP rejects cross-company charging;
- the HX key;
- the GSAP PR and PO numbers;
- the company code, an activity attribute.

**PR rules (FA C4, D7):**
- **Statuses:** Created · Submitted · In-Review · Declined · Approved · Closed.
- One PR line per line item, defaulting to the unreserved balance. Lines cannot exceed the line item's amount less other PR lines.
- **Budget is reserved at creation, before approval.** Only cancelling or deleting the PR releases it.
- Approved PRs cannot be deleted, only **closed out down to the invoiced amount**.
- Whether lines can be added to an approved PR is a setting with three options; the tenant value is "1", meaning unknown.
- **Visibility:** the PR grid is hidden from Marketers (who create PRs) and visible to the TA requisitioners (who edit nothing).

**Where it is fragile (FA D3):** stage 2 is an e-mail and a person retyping a form. The only thread back is the HX number typed by hand.

### 3.3 The hierarchy, money rules and relationships (FA A3, B11)

- **Budget flows down.** Plan, Campaign and Initiative budgets are typed, and each child shows "Remaining <parent> Budget". Whether an initiative budget may exceed its campaign's is contradictory in the guide and untested live.
- **Cost flows up.** Planned cost is typed only on Activity and Line Item, and is read-only and greyed out above them.
- **Actuals come only from finance.**
- **The reconciliation points** are Sys_Gen_Balance and the Finance Budget link.

**Relationships:**
- Line item → Finance Budget row: many to one; **required to save** in Shell's tenant.
- Line item → SWBS code: one to one, created by GSAP.
- Line item → PR lines: one to many.
- Hive9 PR → GSAP PR → PO: one to one to one, linked only by the HX key.
- Plan ↔ Plan and Activity ↔ Activity: "related", many to many.

**Statuses per level (FA B9):**

| Level | Statuses |
|---|---|
| Plan | Draft · Published. Publish has no approver; no Unpublish was seen. |
| Campaign | Created (full list not captured), plus the "Statuses & Approvals" group workflow |
| Initiative | Created · Submitted · **GroupSubmitted · GroupApproved** · In-Progress · Complete |
| Activity | Created · Submitted · Declined · Approved · In-Progress · Complete. Accepted and Committed are configured but unused. |
| PR | Created · Submitted · In-Review · Declined · Approved · Closed |

### 3.4 Purchase requisition flow stages (Slide 11) and the legacy process

- **Standard/Automatic PR.** Vendor and Outline Agreement (OA) are on the form. GSAP creates the PO automatically.
- **Manual PR.** The vendor and/or OA are not known. The CP team or buyer adds them in GSAP.
- **FI Payment PR.** This goes through an FI payment approval, with an e-mail to CP and FM. In Hive9, PR Type is a header attribute that may be left blank. Nothing shows it changes Hive9's behaviour (FA, inferred).
- **Modify/Update PR.** There is a new approval in Hive9, then the GSAP checks and the top-up in GSAP.
- **The legacy "3 ways to buy"** that Hive9 replaced:
  1. Check whether a Global Corporate Card (GCC) can be used.
  2. If not, contact the CP lead.
  3. A budget (WBS/GL/cost centre) is required.
  4. The PR is raised by CP SSU.
  5. MoA approval follows.
  6. The CP buyer creates the PO.
  7. GR/SE and then payment.

  **No client spec or FA section covers the GCC check.**

Every step that happens *in GSAP* stays outside MAPP. Only its data coming back is **on hold** while we have no GSAP access.

### 3.5 Planning calendar and features (Slides 6, 10, 13, 14)

**The yearly cycle (slide 10):**
1. SP&A plan kick-off and SMP planning (Jan–Mar).
2. Plan set-up and budget distribution.
3. BC development.
4. PR-PO initiation and campaign execution.
5. Actual financials and accrual financials (monthly).
6. Campaign evaluation and PIR completion (Oct–Dec).

**Features the deck promises:**
- **Planning:** SWBS within 24–48 h, daily GSAP actuals, approval workflows, standardised data.
- **PR/PO:** PR/PO raised in Hive9, PR/PO number back in 24–48 h, attachments. The PO release thresholds the deck mentions are set in GSAP (FA D7).
- **Budget control:** budget hard stop, Latest Estimate vs PO (commitment) vs actuals.
- **BC/PIR:** BC linked to activities with Marketing and Finance approval, PIR with PPT export.

**ZBB cost hierarchy:**
- **Deck (slide 14):** Legally or regulatory mandated costs · Long term contracts · Core business defend & grow · Strategic priorities · Growth drivers · Tactical investments · Non-core spend.
- **Live Hive9 (FA E3.0.2), 10 values:**
  - LUBES –: Growth drivers, Legal or regulatory requirements, Long term contractual obligations, Other, Protecting the core, Strategic Priorities, Tactical initiatives.
  - RETAIL –: Long-Term Contracts, Strategic Priorities, Tactical Investments.

---

## 4. Coverage by area

| Area | Client specs | Unit | DONE | PARTIAL | MISSING | DIFFERENT | DEFERRED |
|---|---|---|---|---|---|---|---|
| A. App shell, configuration, people, access | 001, 002, 003, 022 | requirement groups | 22 | 25 | 12 | 16 | 4 |
| B. Plan hierarchy, line items, BC/PIR & SP&A | 004, 005, 006 | requirement groups | 20 | 13 | 10 | 16 | 7 |
| C. Views, bulk/import, audit log, work board | 007, 008, 010, 026 | requirement groups | 17 | 14 | 20 | 12 | 10 |
| D. Approvals and notifications | 009, 021 (approval part) | requirement groups | 20 | 16 | 11 | 7 | 4 |
| E. PR/PO, actuals, GSAP, budgets | 013, 012, 015, 011/014 parts | single FRs | 9 | 42 | 67 | 14 | 33 |
| **Total** | 16 specs | | **88** | **110** | **120** | **65** | **58** |

The FA rows added in §5 (marked "FA") are not in these counts.

---

## 5. Findings by area

### A. App shell, configuration, people and access (client 001, 002, 003, 022; FA F)

**Works well:**
- one layout, breadcrumbs, bookmarkable addresses, all errors at once, the unsaved-changes guard, version conflicts, delete with Undo, help tips;
- teams, the person picker and approval delegation;
- server-side access that fails closed, with scopes and 404 outside them, and a full audit of access changes.

**Organisation switches (FA F2).** Most are hard-wired in MAPP to Shell's live value: budgets on, AOP on, shrink dates off, user category off, fixed hierarchy names, default search by name. For those the gap is "no switch", not wrong behaviour. The rows below cover the differences.

| Req | Requirement (short) | Status | Gap / evidence | Priority |
|---|---|---|---|---|
| 001 FR-015–017; 002 FR-059–063, FR-078; FA F4, F6 | Annual planned rate per fiscal year plus 12 monthly rates per currency, kept in the app; switches "use monthly rates for actuals" and "for PRs"; copy from the previous year | DIFFERENT | One dated "current" rate converts everything (016 FR-002, `settings/rates_book.go`); units per USD (D-010). **(FA)** Hive9 keeps the rates in its own Currency screen, edited by the Exchange Manager and Global FINOPS roles, so **this needs no GSAP**. Only an automatic GSAP rate feed, if one exists, is on hold; the simulated rate import stays. | High |
| FA F4; 022 FR-017 | **Campaign and initiative rights are separate** (Marketer: Campaign CE, Initiative CED; Mkg. Admin support: no campaign rights; VP/GM: nothing below campaign) | DIFFERENT | `campaigns.edit` and `campaigns.delete` cover both levels (`authz/catalogue.go`). Four Hive9 roles (~50 users) can't be built. Activity and line item are identical in every Hive9 role, so keeping them merged is safe. | High |
| 022 FR-010–016; FA F4 | One role per person; Hive9 has 20 roles (production has 19, per client 022 FR-014) | DIFFERENT | Five editable roles, several per person, with team scopes (D-223, Q-010). The difference is not just the count: see the row above. | High |
| 003 FR-016/017/022/023; FA F5 | "Can Edit Team Content?" and "Can Approve?" on each team membership | DIFFERENT | No flags on `people.Membership`; roles and scopes do this job (D-223). | High |
| 022 FR-030–033 | Only the assigned approver or their delegate decides; self-approval set per workflow | DIFFERENT | `approvals.decideOnBehalf` lets the Administrator decide. Self-approval is barred by one org-wide switch that is **on by default** (`access/model.go:64`, D-302). | High |
| 022 FR-019–021 | Screens hide or disable forbidden actions with the reason; NoAccess on 403 | PARTIAL | The server is done. Screens are not: 014 tasks T043–T051 are open. | High |
| 002 FR-008–011; FA E3.0.2 | Shared master-data lists with code, label and description; **Shell's real values** | DIFFERENT | Options carry a label only and belong to one attribute. **(FA)** Seed values are made up: 3 of 5 GL codes are not Shell's; Spend Type 5 vs 57; Spend Category 3 vs 19; Contracted Spend 3 vs 5; Cluster 51, Region 14, Company Code 61 and Country/OU 125 are invented samples. The FA gives most real lists; Cost Centre (1,217) still has to be exported. Static lists need no GSAP. | High |
| 003 FR-021; FA B3, B10 | Owners restricted to the plan's teams | PARTIAL | Checked for the plan owner only (`plan/validate.go`). **(FA)** Hive9 itself restricts only the **plan** owner; lower levels use a plain list. Confirm with Shell. | Med |
| 002 FR-065–070, FR-080; 004 FR-013; FA F2 | Organisation settings page | PARTIAL | Only the access rules exist. Behavioural differences: finance-budget link checked at submit, not save (row below); session time-out; unallocated line-item amounts not allowed; related plans; PR configuration. | Med |
| FA F2 | Session time-out 10–120 min, default 60 | DIFFERENT | Fixed 12-hour idle limit (`auth/model.go:19`). Revisit with SSO. | Med |
| FA F2, G5; FA B10 #7 | Finance budget link **required to save** a line item (tenant switch on) | DIFFERENT | Needed to submit, not to save (`lineitem/completeness.go`). MAPP lets drafts save and lists every problem at once. Confirm submit-time is acceptable. | Med |
| 002 FR-012–014 | List import: check everything, import nothing on any error, update by code, 20k rows | DIFFERENT | Adds new valid rows only, skips existing ones, up to 5,000 rows. | Med |
| 002 FR-016/017, FR-019 | Attribute types date, currency, decimal, attachment; min/max; "required to save" | PARTIAL | No such types; markings are optional, needed to publish, or needed to submit. **(FA)** The guide means the red-asterisk fields are required to *submit*, while Hive9's live form blocks *saving*, one error at a time. MAPP's approach follows the guide's intent and is deliberate. | Med |
| 002 FR-020–024 | Calculated, roll-up and lookup attributes; approvable attribute groups | MISSING | — | Med |
| 002 FR-029–035 | Dependent options from an ancestor record; visibility conditions beyond Business Unit | PARTIAL | Same-level parent only; Business-Unit-only conditions. **(FA)** Line-item finance attributes and initiative attributes were hidden on Mobility plans, probably through a BU dependency. | Med |
| 002 FR-077 | Per-model "Re-approve when planned cost changes" switch | DIFFERENT | A fixed rule for every model (`activity/reapproval.go`). See §5D. | Med |
| 002 FR-005/006; 003 FR-008 | In-use configuration and people deleted to the bin with a "(deleted)" marker | DIFFERENT | Set inactive instead; people are never deleted. | Med |
| 003 FR-011–013 | Import users from a spreadsheet | MISSING | — | Med |
| 001 FR-012–014 | Display-currency switch in the header | PARTIAL | Only in My profile. | Med |
| FA F4.3 | Teams-only administration (Marketing Team Leader, SOM Role) | DIFFERENT | `people.manage` covers people and teams together. | Low |
| 022 FR-024; FA F4.1 | Financials-only visibility shows money as "Restricted" | DIFFERENT | `finance.view` hides only GSAP figures. **(FA)** "Own-team financials only" is off for all 20 Hive9 roles, and `finance.view` can already be scoped more narrowly than `plans.view`. | Low |
| 003 FR-042; 022; FA F5 | Admin acts as a user with their consent (24 h / always) | DEFERRED | Nothing in MAPP; after SSO. | Low |
| 022 FR-002–009 | SSO with MFA, time-out warning | DEFERRED | D-004, ADR-0005: demo sign-in until SSO. | — |

### B. Plan hierarchy, line items, BC/PIR and SP&A (client 004, 005, 006; FA A3, B3–B7)

**Works well:**
- the hierarchy and its validation; budget flowing down and cost rolling up; remaining budget that may go negative;
- model goals and the waterfall;
- line-item spreads with the unassigned balance (Sys_Gen_Balance);
- copy rules (no approvals or GSAP values); activity statuses; audit.
- **(FA) Already matching Hive9:**
  - plan name unique system-wide;
  - model, goals, AOP and BCPIR flags;
  - the conditional SMP flag;
  - "Create SWBS code" ticked by default;
  - the line-item name default;
  - the 10-year end-date limit and irreversible "Ongoing";
  - the required activity attributes on Mobility: Channel, Country/OU, Company Code, Cost Centre, Global Campaign, Growth Ambition, PIR.

| Req | Requirement (short) | Status | Gap / evidence | Priority |
|---|---|---|---|---|
| 005 FR-002; FA A4, D3 | Line-item changes never change the activity's status; **line items are added after approval in the as-is flow** | DIFFERENT | Line-item add, duplicate, delete, cost or currency change restarts approval (D-013; `lineitem/create.go`, `duplicate.go`, `update.go`, `delete.go`). **(FA)** This breaks Shell's order: approve the activity → add line items → SWBS → PR. Restart only when the activity's planned cost changes (L3). | High |
| 005 FR-014/015 | SWBS code read-only from GSAP; GSAP status Draft/Approved/Sent/Confirmed; lock "Create SWBS" once sent | PARTIAL | Derived status only; no stored code (D-255); nothing sent. **(FA)** One code per line item, sent when the activity is approved and the box is ticked. A code entered by Finance can be stored now (L4a); the round trip is on hold (L4b). | High (manual) · On hold (round trip) |
| FA D5, G8 | **HX key** ("HX" + line-item ID) shown on the line item and PR form, with copy | MISSING | MAPP derives HX only when matching transactions (`financials/match.go:8`); no line-item screen shows it. Needs no GSAP. | High |
| FA E3.0.2, B6, B7 | Shell's real attribute values (GL, Spend Type and Category, Contracted Spend, Cluster, Region, Company Code, Country/OU, Brand, GKA, Function, Focus Area…) | DIFFERENT | Seed values are invented (`seed/settings.go:105–192`). `docs/product/field-catalogue.md` says "13 marketing GL codes" but the seed has 5. | High (quick) |
| 004 FR-036; FA E3.0.2 | (Zero-Based) Cost Hierarchy | DIFFERENT | Seeded as "Media, Agency, Events, Production" for all BUs (`seed/settings.go:153`). **(FA)** Live Hive9 has 10 BU-prefixed values (7 LUBES, 3 RETAIL); the deck lists 7 other labels. Shell to choose. | High (quick fix) |
| 005 FR-041, FR-048 | "Purchase Requests & POs" section and Create PR; planned cost not below the PR amount; **Total PR Amount** on the line item, rolled up to the activity | MISSING | D-243: no purchase requests. POs are read-only (015). The PR part and the Total PR Amount need no GSAP; the PO data does (on hold). | High |
| 004 FR-056; FA B6.3 | Latest Estimate LE1–LE4 and Locked LE1–LE4 | DEFERRED | Feature-map "Later"; no fields or API. **(FA)** These are single amounts on the Activity, typed and locked, not phased. | High |
| 005 FR-039; FA C2, B7.2 | Hard limit / remaining-budget check (hard stop) | MISSING | Finance budget items have no amounts (`settings/model.go`). **(FA)** The hard limit is a **Finance Budget row** setting (Is Hard Limit, Budget Lock, Forecast Lock). That is the Finance Budgets module, out of scope (CLAUDE.md #13). | High (scope) |
| 006 FR-004–021 | SP&A view: planned, committed, actual, remaining, utilisation, variance by quarter and channel | MISSING | Our 013 is Draft (gated on Q-001); no code. | Med |
| Deck; FA B3, B6 | Business Case record linked to activities, with approval and attachments; PIR with approval and PPT export | MISSING (new) | **(FA)** Hive9 has only the "BCPIR Plan?" flag and a required "PIR" Yes/No on the activity. MAPP has both (`plan.Plan.BCPIR`, `seed/settings.go:148`), so parity is DONE. The records are a **new** deck requirement. | Med |
| 004 FR-024/025; FA B4, B9 | Campaign "Statuses & Approvals" and **initiative** "SMP Initiative – Footer" group approvals; initiative statuses Submitted, GroupSubmitted, GroupApproved; campaign submit cascade | DEFERRED | D-012: derived approval progress only (`campaign/model.go`). **(FA)** Configured in Shell's tenant. The guide says the cascade **approves** never-submitted children, which would bypass each activity's own workflow. | Med |
| 004 FR-034, FR-052–054; FA B5, B6 | Activities may cross fiscal years (multi-year linking); dates against the parent | DIFFERENT | R3 keeps an activity inside its initiative and an initiative inside its campaign (`activity/dates.go`, `campaign/validate.go` `checkInsideCampaign`). **(FA)** Crossing fiscal years is confirmed (multi-year needs a parent per year; D-017). Sitting outside the parent is unconfirmed: the guide says initiatives stay inside their campaign, but a live initiative ends after its campaign. | Med |
| FA B6, E3.0.2 | Attribute definitions: FIT4 (Yes/No/To Be Confirmed; not shown on Mobility); FIT4 ID and Innovation not in the tenant; "Actual Financials Rating" (5 values); Country/OU and Brands single choice, not % split | DIFFERENT | MAPP asks FIT4 and FIT4 ID **for every BU** (`seed/settings.go:150–152`) and has Innovation; no Actual Financials Rating. | Med |
| 005 FR-016–018 | Convert at the annual planned rate of the plan's fiscal year; refuse when it is missing | DIFFERENT | Latest dated rate (D-258); plan currency (D-260). See §5A; needs no GSAP. | Med |
| 004 FR-066; 005 FR-044 | Refuse delete while transactions or PRs are linked | DIFFERENT | Allowed with a warning (015 FR-029). | Med |
| 004 FR-017 | Refuse a model change when activities use a disallowed type | DIFFERENT | Allowed with a warning; those activities are flagged. | Med |
| 004 FR-012 | Warn and confirm when the creator is in none of the plan's teams | DIFFERENT | Refused instead; visibility follows role scope (D-226). | Med |
| 004 FR-037 | Separate cost and performance splits (each 100%) | PARTIAL | One share per option. | Med |
| 004 FR-038; 005 FR-019; FA G4 | Total PR amount, % of actual cost, Actuals @ monthly FX | PARTIAL | Committed, Invoiced, Paid and Accrued from the simulated feed. **(FA)** The PR amount comes from MAPP. % of actual cost (paid ÷ planned) and Actuals @ monthly FX can be computed now from MAPP's own rates (`settings.RatesOn` exists but `financials` doesn't use it). The real actuals feed is on hold. | Med |
| 005 FR-036/037 | Finance budget link per budget year, with the remaining amount in the picker | PARTIAL | Items have no amounts, so there is no "remaining". | Med |
| 006 FR-001–003 | BCPIR marker and filter wherever plans are listed | PARTIAL | The flag is stored and logged; no marker or filter. | Med |
| 004 FR-009; FA B3 | Plan year from now to +10, **no past year** | DIFFERENT | We allow −1 to +5. | Med |
| FA B6.1 | Activity **Date Sets**: Due, Build and Completion dates; Start and End time | MISSING | Not in MAPP or the feature map. | Low |
| FA B3, B6; 004 US10 | Custom Goals, Related Plans (tenant switch on), Related Primary Activities, Media Codes | DEFERRED | Feature-map "Later" (`feature-map.md:87`). Media Code also feeds Measure matching (§5 note on Measure). | Low |
| FA B7.1 | Allocation by **quarter** (E/F/L on quarters); repeated "Add past/future year" | PARTIAL | Stored monthly; quarterly is display only (`lineitem/grid.go`). | Low |
| FA B7.2 | Sys_Gen_Balance spread with the plan's default allocation | DIFFERENT | Always spread evenly (`allwork/periods.go`). | Low |
| FA B1.2 | Plan dates derived from its children, shown in the grid | MISSING | Plans have no dates (fixed 1 Jan – 31 Dec in the row). | Low |
| FA B3–B7 | Rich-text descriptions | DIFFERENT | Plain text ≤ 2,000 characters. | Low |
| FA B6 | Type change silently resets planned cost to the model default | DIFFERENT (on purpose) | MAPP asks "Use default" or "Keep current", which avoids surprises (constitution IX). | Low |

### C. Views, bulk and import, audit log, work board (client 007, 008, 010, 026; FA B1, B8)

**Works well:**
- the three All work views (grid, budget & cost, calendar); roll-ups; column chooser; personal saved views; AOP and Show levels; export to .xlsx and .csv;
- the activity log on every record; write-once audit;
- the recycle bin with restore-with-parents and purge. Note that log and bin visibility is broader than in Hive9 (row below).

| Req | Requirement (short) | Status | Gap / evidence | Priority |
|---|---|---|---|---|
| 008 FR-001–010, 013–016, 019, 022–025 | Bulk select, change owner, change field, copy, move, delete with per-item results | MISSING | Designed in our 012 (D-126–D-139); 11 contract operations in `notYetServed`; no screens. | High |
| FA B8 | Hive9 bulk ("Honeycomb") menu: **Change Status**, Link To, Package/Unpackage, Create Template, **Create Purchase Req.**, Export Selected Columns; **no Delete** | DIFFERENT | Our 012 has change owner and field, submit, copy, move and **delete**; no Change Status, Link To or Package. | Med |
| 008 FR-044–051 | Import activities from a template, validated, with undo | MISSING | Designed in 012; row limit 1,000 (ours) vs 5,000 (client) is still open. | High |
| 007 FR-004, FR-020, FR-022–028 | Row actions and inline editing in the workspace | DEFERRED | All work is read-only on purpose (010 FR-002) so approval restarts are never silent. **(FA)** Hive9 edits in an "Inspection Window" panel, not in the grid cells. | High |
| 010 FR-002/004 | Audit fields: source (User/Import/GSAP/System/Restore), on-behalf, year, operation reference | PARTIAL | `activity_log` has none of these. | High |
| FA F4.3 | **Plan audit log and recycle bin visible to admin roles only** (Client Admin, System Admin, SOM Admin, GM View) | DIFFERENT | Every signed-in user sees entries for plans they can see (`access/routes_views.go:36–38`, D-238), and anyone allowed to delete a kind may restore it (`app/history_access.go`). Shell to confirm. | Med |
| 007 FR-043/044 | Drag on the calendar to reschedule | DIFFERENT | Not draggable on purpose (010 FR-027). | Med |
| 007 FR-071, FR-085 | Filters, view and options remembered per user across sessions and devices | DIFFERENT | Only columns and sort are stored; filters last for the session (D-122). | Med |
| 007 FR-079–082 | Share saved views with teams or users | MISSING | Views are personal only (010 FR-030). | Med |
| 007 FR-034–038 | 8 summary cards with Annual/Total, Allocated, Actual @ FX; hide/reorder | PARTIAL | 6 fixed cards. | Med |
| 007 FR-047/048 | Budget & Cost per period including Actuals | PARTIAL | No actual cost per period. | Med |
| 007 FR-052–062 | Attribute filter panel per level, line-item filters, date sets | PARTIAL | The backend exists; the UI (016 T074–T077) is open. | Med |
| 010 FR-011 | Separate Plan, Finance and Configuration logs | DIFFERENT | One log with a type filter. **(FA)** Hive9's navigation shows only a "Finance Log"; the Plan log is admin-only. | Med |
| 010 FR-018 | Export the audit log | MISSING | — | Med |
| 010 FR-043–045 | Retention (bin 90 days, audit 7 years) with a daily job | DIFFERENT | Nothing is purged automatically; the write-once trigger would block audit retention. | Med |
| 008 FR-030/031 | Copy Plan 1–10 copies, named `_Copy_MMDDYYYY_HHMMSS_n` | DIFFERENT | One copy with a "(copy)" suggestion. | Med |
| 007 FR-010–012; FA B1 | Grid search with 7 criteria (incl. Media Code, SWBS Code, Integration ID) | DIFFERENT | 4 criteria: name, ID, external name, description. SWBS Code search is on hold (GSAP) until codes are stored. | Low |
| FA F6 | Calendar coloured by activity type ("Theme") | DIFFERENT | Coloured by status (`all-work/CalendarView.tsx`). | Low |
| 008 FR-017 | Create Purchase Request from a selection | DEFERRED | Purchase requests are out of scope. | — |
| 026 all | Activity Work Board (kanban, columns, WIP, swimlanes) | MISSING | Nothing exists. The client marks it "Proposed, needs Shell approval"; it is not in the FA. | Low |
| 008 FR-018, FR-037–043, FR-057–067 | Templates, snapshots, calendar feed | DEFERRED | Feature-map "Later". | Low |

### D. Approvals and notifications (client 009, 021; FA B9, C6, G9)

**Works well:**
- serial levels with Any or All; amount and attribute conditions; frozen rounds; auto-approval; recall; the stale-action conflict (409);
- delegation, reminders, escalation, Monitoring; a complete approval history (Level · Approver · Date · Status · Comment, as in Hive9); the inbox;
- re-approval warnings; in-app notifications for approval events; required decline comment;
- the workflow chosen per activity, as a required field (as in Hive9).

| Req | Requirement (short) | Status | Gap / evidence | Priority |
|---|---|---|---|---|
| 009 FR-004, FR-016, FR-034/035; FA C6 | Workflow object types: Activity, **PR, Modified PR, Accrual**, attribute groups | PARTIAL | Activity only (`approval/model_rounds.go` `ActivityID`). | High |
| 009 FR-007; FA C6 | Approver types **Record Owner** and **Custom Attribute** (a person taken from a field on the record, e.g. the PR's MOA approver) | MISSING | Sources are named people, line manager and role holders only (`approval/model.go`). Checking the MOA against GSAP's authorisations is on hold (GSAP). | High (with PRs) |
| 009 FR-050; FA A4 | Re-approval on a **planned-cost** change, per model setting | DIFFERENT | Always on; also triggers on dates and line-item add, delete, amount or currency (D-013). **(FA)** This collides with the as-is order (§5B first row). | High |
| 009 FR-003 | Per-workflow settings: threshold currency, comment on decline/approve, allow self-approval (default **yes**) | DIFFERENT | None on `Workflow`. Self-approval is barred by one org-wide switch, **on by default** (D-280, D-302). **(FA)** The as-is PR flow relies on self-approval (FA D3 1.4). | High |
| 009 FR-008 | Conditions on Country/OU and custom attributes | PARTIAL | Planned cost (USD), business unit and activity type only. **(FA)** Shell writes the amount limits into the workflow names (e.g. `2LEVEL-ALL-SIGN-OFF-600KUSDLIMIT`), with 100+ workflows. | High |
| 009 FR-014 | Bulk-load 100+ workflow definitions from a file | MISSING | — | High |
| 009 FR-017; FA Figure 7 | Re-Submit from Submitted, cancelling the open round | DIFFERENT | Created or Declined only (`rounds_submission.go:80`); Submitted needs a Recall first. **(FA)** Confirmed as live Hive9 behaviour. | Med |
| 009 FR-013; FA C6 | Basic (team "Can Approve?") vs Advanced regimes; org-wide switch with a **cut-over guard**: nothing left in Submitted | MISSING | Advanced-style only. No plan for in-flight Hive9 approvals at go-live (see §9). | Med |
| 009 FR-027 | Self-exclusion empties a level → refuse the submission | DIFFERENT | Falls back to the backup approver or an on-behalf decision. | Med |
| 009 FR-030; FA B8 | In-Progress and Complete set by a status change (Hive9 "Change Status") | DIFFERENT | Derived from dates. | Med |
| 009 FR-041–044 | Approvals queue: item type, dates, status columns, per-row actions, sort, paging, saved views | PARTIAL | Activities only; actions are in the preview panel. | Med |
| 009 FR-029 | Bulk submit | MISSING | Depends on 012. | Med |
| 009 FR-048/049; FA G5 | Gates: SWBS, PR submission, performance, External Name generated on approval | PARTIAL | SWBS is a **derived readiness label only**; the request is on hold (GSAP). No PR gate. External Name is free text (`activity/model.go:52`); **(FA)** Hive9 generates it on approval from a Model rule. Once SWBS sending exists, the "sent" state must be stored so that leaving approval doesn't undo it (FR-049). | Med |
| 021 FR-001–005; FA F4 | Per-user notification rules (event × channel), org defaults | MISSING | Fixed in-app approval notifications only. **(FA)** The per-role notification defaults are identical for all 20 Hive9 roles, so org-level defaults are enough. | Med |
| 021 FR-020, FR-023–029 | Notification List page; custom and due-date notifications | MISSING | — | Med |
| 021 N01–N14; FA G9 | Planning events: edited, deleted, restored, assigned, shared, comments, Sys_Gen non-zero, line item linked or unlinked, budget overage, new unlinked transactions | MISSING | Notifications live inside `approval`. Home shows some of these as attention items (`home/attention.go`). SWBS and PR-pull events are on hold (GSAP). | Med |
| FA B6, G9; 004 FR-058 | Comments on records and a **Commenter** role (14 users) | DEFERRED | D-159: decline comments only. | Med |
| 009 FR-005/006 | Default level mode **All**; levels labelled A/B/C | DIFFERENT | Default Any; levels numbered (`approval/workflows_levels.go`). | Low |
| 009 FR-033; FA B9 | Plan Draft → Published, **no approver, no Unpublish** | DIFFERENT | Unpublish exists (`plan/status.go:61`). **(FA)** MAPP's Publish also requires every campaign and initiative to be complete, a stricter gate than anything seen in Hive9. | Low |
| FA F5 | "Activate Delegated Approver" on/off toggle | DIFFERENT | Dated delegations of at most 90 days, on the Approvals page. | Low |
| 009 FR-025; 021 N04/05 | Notify the owner **and the submitter** of a decision | PARTIAL | Owner only (`rounds_decide.go:207`). | Low |
| 021 e-mail | E-mail channel | DEFERRED | E-mail is out of scope (constitution). | — |

### E. Purchase requests, transactions and actuals, GSAP, budgets (client 013, 012, 015, 011/014 parts; FA C, D)

**Works well:**
- a read-only, simulated GSAP feed of PO lines, invoices (credits negative), payments and accruals;
- check-then-import, idempotent re-runs, matching by line-item ID → HX key → PO → SWBS, manual linking of unmatched rows;
- Planned, Committed, Not yet committed, Invoiced and Paid at every level; a Financials page; finance permissions.

Most of client 015 (GSAP integration) and 012 (transactions and actuals), and the GSAP-facing parts of 013, need GSAP. Those rows are marked **On hold (GSAP)** in the priority column. They keep their status, because the status describes the code.

| Req | Requirement (short) | Status | Gap / evidence | Priority |
|---|---|---|---|---|
| 013 FR-001–036, 048–071; FA C4 | **The whole PR lifecycle**: create from a line item, header and lines, one company code, balance check, reservation, submit gates, approval and modification workflows, close-out, delete or restore, list page, audit | MISSING | No PR concept (D-243; constitution scope). The reusable parts already exist: `record.Meta`, `history`, the approval engine (once it handles any record type), Shell attribute levels. **Needs no GSAP.** | High |
| 013 FR-023–025; FA C4, D7 | PR reservation **from creation, before approval**; only cancel or delete releases it; roll-up to the activity | MISSING | Nothing reserves money; Committed comes only from GSAP PO lines. | High |
| FA D5, G8; Appendix 2 | **Legacy HX keys.** Open Hive9-era PRs and POs carry HX + Hive9 line-item IDs (up to 7 digits). MAPP numbers from 1001. | MISSING | `financials/match.go` maps HXnnnn → MAPP L-nnnn, so legacy keys would match the wrong line items or none. Decide now: start MAPP numbering above Hive9's highest ID, keep Hive9 IDs, or add a legacy mapping. Testing on real data is on hold (GSAP). | High (decision) |
| FA C6, D7 | **MOA matrix** (legal entity × amount → authorised approvers) to check the PR's MOA approver | MISSING | None. The MOA picked in Hive9 is sometimes not authorised in GSAP. Needs Shell data, not GSAP. | Med |
| FA C4, E3.7 | PR master lists: vendors, Outline Agreements (287), e-mail groups (145), PR recipients (153), contract holders (58), China OA vendor and sub-category | MISSING | None. A GSAP vendor-master feed would be on hold; Shell-maintained lists need no GSAP. | Med |
| 013 FR-026 | Refuse deleting a line item or activity that a PR uses; refuse lowering planned cost below the PR total | DIFFERENT | Delete is allowed with a warning (015 FR-029). | High |
| 013 FR-030; FA C4, G5 | Submission gates: activity approved, SWBS code present, required fields filled | MISSING | Line items store no SWBS code (D-255). **(FA)** Hive9's system gate is the approved activity. The SWBS code is a practical need of the GSAP PR, not a confirmed Hive9 gate. See §6.1. | High |
| 013 FR-039–045 | Route the approved PR form to the requisitioner (Excel), with retries and routing history | DEFERRED | E-mail is out of scope. Recommended: automatic in-app routing on approval plus a downloadable PR form with the FA's columns (HX key, short text, cost centre, WBS, GL, amount, MOA approver, documents). Needs no GSAP. | High |
| 015 FR-006–016 | Daily FSSI 1523 SWBS request file and SWBS code return | DEFERRED | "Sending to GSAP (future)"; ADR-0009 covers inbound feeds only. Needs GSAP. | On hold (GSAP) |
| 015 FR-017/019 | Inbound fields: GSAP PR number, PO status and copy link, invoice status and link, GR/SE (GRSP amount), PO line status, PR-level transactions | PARTIAL | The CSV has type, document, PO, date, amount, currency, vendor, SWBS and HX only. New fields need the real GSAP layout. | On hold (GSAP) |
| 015 FR-018 | Matching M1–M8: attach to an approved PR line; company-code check | DIFFERENT | D-245 matches without PRs and without a company-code check. Matching against real GSAP data needs GSAP. | On hold (GSAP) |
| 015 FR-035–037; FA D9 | Finance exception list (Reprocess, Assign, Dismiss, last seen); handling of rejected SWBS requests and unmatched HX keys | PARTIAL | A "Not linked" tab plus manual link. Exceptions come from real GSAP files. | On hold (GSAP) |
| 015 FR-038/040 | GSAP status Draft/Approved/Sent/Confirmed on the line item, with PR and PO data | PARTIAL | Three derived statuses only. Sent and Confirmed need GSAP. | On hold (GSAP) |
| 015 FR-041–043 | Company-code rule | MISSING | Company code exists only as an activity attribute. The rule checks GSAP rows. | On hold (GSAP) |
| 013 FR-017; 012 FR-016/017 | Balance check at the monthly PR rate or the annual planned rate; actuals at the annual rate plus @ monthly FX | DIFFERENT | One current rate (D-303). **(FA)** Both rate sets live in Hive9's own Currency screen, so this **needs no GSAP** (L7). | Med |
| 013 FR-046/047 | Supporting documents (10 MB per file, 20 MB per PR) | DEFERRED | BUS.FIN.060 Later; no upload capability in MAPP. Needs no GSAP. | Med |
| 013 FR-058–060; FA D3 4.1 | Marketer approves or rejects vendor invoices; a reissued invoice cancels the old one | DEFERRED | BUS.FIN.050 Later. The invoice and its link come from GSAP. A manual "work delivered" confirmation could stand in during the freeze (FA D8). | On hold (GSAP) |
| 012 FR-009–014 | Split one transaction across line items; Attributed and Remaining | DIFFERENT | One whole link per transaction (D-253). This acts on GSAP actual-cost postings whose real layout is unknown. | On hold (GSAP) |
| 015 FR-020/021 | Latest business date wins; cancelled invoices leave the totals | PARTIAL | The last import wins; no invoice status. Both need the real GSAP data. | On hold (GSAP) |
| 015 FR-034 | Archive every file with its outcome | MISSING | The file is read in memory only. This only matters for real GSAP files. | On hold (GSAP) |
| 015 FR-001/002, FR-046–053; 012 FR-001–006 | Automated route and schedule, expected-arrival alerts, versioned column mappings, the 23-column actual-cost extract | PARTIAL / DEFERRED | Manual simulated CSV with fixed columns (D-247), which stays as it is. | On hold (GSAP) |
| 011 FR-024/025; FA C2 | Hard limit on the finance budget row; refuse a change that would push the remaining budget below zero | DIFFERENT | Warn only (D-015, D-235; BRD BUS.BUD.030 "warning, not a blocker"). **(FA)** It is a Finance Budget row setting, so it needs the Finance Budgets scope decision. | Med (scope) |
| FA C5, B9 | Accrual Requests **raised in Hive9** (header and lines approved one by one; Approved & Sent → Processed) | DIFFERENT | MAPP only imports accruals from the simulated GSAP feed (D-243, D-244). Accrual calculation is out of scope; sending requests out is on hold (GSAP). | Low |
| FA F4.4 | Monitoring lists like Hive9's "SWBS Not Received", "Pending PR", "PR Track Invoices" | MISSING | A "waiting for SWBS" list can be built on the derived status now. "Approved PR with no GSAP PR number" needs the feed (on hold). | Low |
| 013 FR-061–064 | Finance › Purchase requests page grouped Vendor › PR › line › line item | MISSING | The Financials page lists transactions only. Needs no GSAP. | Med |
| Deck | "3 ways to buy": GCC check before a PR | MISSING | Not in any client spec or the FA; needs clarifying. | Low |

**Measure lineage to protect now (FA G7, G8).** Measure is out of scope, but it joins on the **Activity**. So the Plan data model should not block it later:
- a stable activity ID;
- **Media Code** and an **integration ID** on the activity (neither exists);
- **External Name** generated by a Model rule;
- **Tactic Category** on activity types (missing);
- a history of model rates, because Hive9 restates projections with no version stamp (FA G6).

All of these are Low for now.

---

## 6. Business logic and flow changes needed (consolidated)

Ordered by dependency. Each item names the packages it touches. Sizes are relative: S, M, L, XL. ⏸ = on hold (GSAP).

| # | Change | Why (client source) | Touches | Size |
|---|---|---|---|---|
| L1 | **Approve any record type.** Make the round's subject generic: `Round.ActivityID` becomes `SubjectKind` + `SubjectID`, and `Workflow.ObjectType` is added. Add the approver sources **Record Owner** and **person from a field on the record**. Add one facts adapter per type, generic decide routes and an object-type filter on the inbox. Targets: PR, modified PR, campaign and initiative group approvals, BC/PIR, budget increase, accrual. (No plan approval: Hive9 has none.) | 009 FR-004/007/016/034; FA C6, B4, B9; deck "multi-layered approval" | `approval/*`, `app/approvals_adapter.go`, `access/routes_*.go`, openapi, `features/approvals` | L |
| L2 | **Per-workflow settings:** comment on decline/approve, allow self-approval, threshold currency, default mode All. Agree how they relate to the global segregation-of-duties switch (D-280, D-302) and to the admin "decide on behalf" right. PRs need self-approval (FA D3 1.4). | 009 FR-003, FR-005, FR-027; 022 FR-030–033 | `approval/model.go`, `rounds_decide.go`, `resolve.go`, `access/rules.go` | M |
| L3 | **Re-approval rule.** Restart only when the activity's planned cost changes (client and FA), switched per model. Line items that only split the unassigned balance must not restart approval, because Shell adds them after approval. Ours today: cost, dates and every line-item change (D-013). Allow Re-Submit from Submitted. | 009 FR-017, FR-050; 002 FR-077; 005 FR-002; FA A4, D3, Fig. 7 | `activity/reapproval.go`, `lineitem/create,duplicate,update,delete.go`, `approval/rounds_submission.go`, `settings` models | M |
| L4a | **Store the SWBS code on the line item; Finance enters or uploads it.** A read-only field for everyone else, the date it was received, and an import through the existing check-then-import pattern. The FA's freeze-period remedy. Partly reverses D-255; ADR-0009 needs a note (MAPP records codes, never creates them). | FA D7, D8; 005 FR-014 | `lineitem/*`, migration, openapi, `features/line-items` | M |
| L4b | ⏸ **SWBS round trip.** Outbound FSSI 1523 request file, automatic code return, Sent and Confirmed states with a 24–48 h SLA. "Sent" must be stored so that leaving approval doesn't undo it (009 FR-049). Needs a new ADR for outbound feeds. | Deck step 1; 015 FR-005–016 | `lineitem`, `financials`, `approval/hooks.go` | L |
| L5 | **Purchase Request record, MAPP side** (new `purchaserequest` package; see §7). Hangs off line items; reserves money at creation; approval and modification approval; close-out; automatic in-app routing plus a PR form. Needs no GSAP but does need the scope decision (§9). The GSAP fields (PR number, PO data) exist but stay empty until GSAP access arrives. | Deck steps 2–4, 9 (MAPP part); 013; FA C4, D3 | new package, `lineitem`, `activity`, `approval`, `access`, `settings` (PR attribute level), new screens | XL |
| L6 | ⏸ **GSAP matching to PR lines and richer inbound fields.** PR number, PO status and copy link, GR/SE, invoice status and link, company-code check (M5–M7), supersede by business date, exception records with Dismiss, rejected-SWBS handling. All of this depends on the real GSAP file layouts; building against guesses risks rework. | 015 FR-017–021, 035–043; FA D4–D6, D9 | `financials/*`, CSV layout (D-247), migrations | L |
| L7 | **Exchange rates by fiscal year, kept in MAPP.** Annual planned rate per currency and year, 12 monthly rates, and per-currency switches for monthly rates on actuals and on PRs; copy year to year. Actuals @ monthly FX and % of actual cost on the existing simulated feed. **Needs no GSAP** (Hive9 keeps rates itself). Only an automatic GSAP rate feed, if any, is on hold. Reopens D-010, D-027, D-258, D-303. | 001 FR-015–017; 002 FR-059–063; 005 FR-016–018; 012 FR-016/017; FA F4, F6 | `settings/rates*`, `money`, `lineitem`, `rollup`, `allwork`, `budget`, `financials` | L |
| L8 | **Budget hard stop and release.** In Hive9 the hard limit sits on the **Finance Budget row** (Is Hard Limit, Budget Lock, Forecast Lock). So this needs the Finance Budgets scope decision first (§9). Then: amounts and settings on finance budget items; remaining-budget check on line-item save, link and PR reservation; specify the empty `019-budget-increase-approval`. | Deck "Budget release & hard stop"; 005 FR-039; 011 FR-024/025; FA C2, B7.2 | `settings` finance items, `budget`, `lineitem/validate.go`, L1 | M–L |
| L9 | **Latest Estimate.** LE1–LE4 (typed) and Locked LE1–LE4 (read-only) as single amounts on the activity; an LE-lock action per round and year; LE column in All work; LE vs PO vs actuals. Needs no GSAP; the PO and actuals side uses the simulated feed until then. | Deck; 004 FR-056; FA B6.3 | `activity`, `allwork`, `financials`, migration, UI | M |
| L10 | **BC and PIR records** (a *new* deck requirement; Hive9 has flags only, and MAPP has them too). BC linked to one or more activities, with approval (L1) and attachments; PIR as a dated review with approval and PPT export. Answer Q-001 first, then build our 013 SP&A and BCPIR views. | Deck slides 6, 13, 16; 006 | new package(s), `plan`, `approval`, file storage (ADR) | L |
| L11 | **Owners and team flags.** Plan owner from the plan's teams is done. Lower levels: confirm with Shell (Hive9 restricts only the plan). Membership flags "Can Approve?" and "Can Edit Team Content?" if Shell keeps them. | 003 FR-016–023; FA B3, F5 | `activity/fields.go`, `campaign/*`, `people/*`, `approval/resolve.go` | M |
| L12 | **Delete protection.** Refuse deleting a line item, activity, campaign or plan with linked transactions or PRs, instead of warning (015 FR-029). Approved PRs can't be deleted, only closed out. | 004 FR-066; 005 FR-044; 013 FR-026; FA C4 | `lineitem`, `activity`, `campaign`, `plan` delete paths | S |
| L13 | **Activity dates.** Support crossing fiscal years through multi-year linking (confirmed by the FA; D-017). Relax "inside the parent" for initiatives and activities only if Shell confirms. A live Hive9 initiative ends after its campaign. | 004 FR-034, FR-052–054; FA B5, B6 | `activity/dates.go`, `campaign/validate.go`, `rollup` | M |
| L14 | **Master data with codes.** Option code and description, or a shared List entity; atomic import that updates by code. Then load **Shell's real lists** (FA E3.0.2) and the PR lists (vendors, OA, e-mail groups, recipients, contract holders). Shell asked for a Shell-side admin to maintain them (FA F3). | 002 FR-008–014; FA E3.0.2, C4, F3 | `settings/*`, seed | M |
| L15 | **Audit model.** Add source, on-behalf, year, plan and operation ID; export; retention with a controlled exception to the write-once trigger (ADR). Decide log and bin visibility (Hive9: admins only). | 010 FR-002/004, 018, 043–045; FA F4.3 | `history`, migration, `activity-log` UI, `access/routes_views.go` | M |
| L16 | **Notifications as a module.** Move them out of `approval`; add an event catalogue (FA G9: planning events and PR events; SWBS and PR-pull events ⏸), org-level defaults, a list page, notify the submitter as well as the owner. | 021; FA G9, F4 | new `notification` package, UI | M |
| L17 | **(FA) Split campaign and initiative permissions** (`initiatives.edit` and `initiatives.delete`), and teams-only administration. This makes Marketer, Mkg. Admin support, IT Admin, VP/GM, Marketing Team Leader and SOM Role buildable. Add PR permissions with L5. | FA F4; 022 FR-017 | `platform/authz/catalogue.go`, `access/routes_*.go`, `seed/access.go`, UI | M |
| L18 | **(FA) HX key.** Show "HX" + line-item number on the line item and the PR form with copy; decide the numbering against Hive9's legacy IDs (start above Hive9's highest, keep Hive9's, or map legacy keys). Matching on real data is ⏸. | FA D5, G8, Appendix 2 | `platform/ids`, `lineitem` view, `financials/match.go`, migration plan | S–M |
| L19 | **(FA) Cut-over design.** Plan what happens at go-live to Hive9 records in flight: Submitted activities, open PRs, POs carrying Hive9 HX keys, and the status mapping (GroupSubmitted, GroupApproved, In-Progress, Complete, Accepted, Committed). | FA C6, Appendix 2 | migration tooling, `approval`, `financials` | M |
| Q1 | **Quick fixes:** Shell's real GL codes and finance lists (`seed/settings.go:177–192`) and `field-catalogue.md`; ZBB values once Shell chooses (`seed/settings.go:153`); FIT4 only where Shell uses it; Re-Submit from Submitted; plan year range current to +10; BCPIR marker and filter; Unpublish (keep or remove); display-currency switch in the header. | 004 FR-036, FR-009; 009 FR-017, FR-033; 006 FR-001–003; FA E3.0.2 | seed, docs, `approval`, `plan`, `TopBar.tsx` | S each |

### 6.1 What waits for GSAP access, and what can go ahead

**Rule:** a piece of work is on hold if it sends data to GSAP, needs new data back from GSAP, or depends on the real layout of GSAP files. Building such work against guessed layouts risks rework, and none of it can be tested end to end. The simulated GSAP feeds that already exist (015 transactions, 016 rates; ADR-0009) **stay as they are**, so the demo and the Financials figures keep working.

**Never MAPP work:** steps that happen inside GSAP are not "on hold". They stay outside MAPP permanently:
- GSAP PR approval by the MOA;
- PO creation and release, including the $50,000 threshold;
- vendor and OA set-up;
- Service Entry release;
- payment runs.

| On hold until GSAP access | Can go ahead without GSAP |
|---|---|
| SWBS request file and the automatic code return (L4b; workflow step 1) | Everything in phases 0–2 of the roadmap (§10): open tasks, quick fixes, planning-rule alignment, bulk and import |
| GSAP PR number coming back (step 5) | **SWBS code entered or uploaded by Finance** (L4a) |
| PO status, PO copy link, PO line status, link from PO to PR (step 6, L6) | **HX key** shown and copyable; HX numbering decision (L18); cut-over design (L19) |
| Invoice link, the marketer's invoice approve/reject, a reissued invoice cancelling the old one, GR/SE (GRSP amount) (step 7) | Approve any record type with record-owner and record-field approvers (L1), per-workflow settings (L2), re-approval rule (L3) |
| Daily actuals and payment feed; the actual-cost extract; splitting postings across line items; PR-level transactions (step 8; 012) | **Exchange rates in MAPP:** annual and monthly, Actuals @ monthly FX, % of actual cost (L7) |
| PR top-up and modified PO in GSAP (step 9, GSAP part) | Latest Estimate (L9); BC and PIR records (L10); notifications module (L16) |
| Matching to PR lines, company-code check, supersede by business date, cancelled invoices, file archive, exception list with Dismiss, rejected-SWBS handling (L6; 015) | Owners and team flags (L11), delete protection (L12), master data with codes and Shell's real lists (L14), audit fields (L15), permission split (L17) |
| Automated or scheduled GSAP route, expected-arrival alerts, column mappings (015); any automatic GSAP rate feed; any GSAP vendor-master feed | **Once in scope (§9):** the PR record, its lines and reservation, approval and modification in MAPP, close-out, automatic in-app routing, the PR form with the FA's columns, the MOA matrix and PR master lists, Total PR Amount (L5) |
| Checking the MOA approver against GSAP's authorisations | A manual "work delivered" confirmation as a freeze-period stand-in for invoice approval (FA D8), if Shell wants it |

**Needs a scope decision, not GSAP:** the budget hard stop (L8) sits on Finance Budget rows, which is the Finance Budgets module (§9).

**What unlocks the on-hold items:**
- **(a) Shell's integration technical document and sample CSVs** for FSSI 1523, PGSI 941 and the actual-cost extract, which Shell promised (FA D4, H3). These unlock the file-import layouts through manual upload, as ADR-0009 already allows, without any system access.
- **(b) A live connection.** Not before about March 2027, and possibly redesigned after S/4HANA (FA D1, D9). Don't build automated transport against today's layouts.

**The SWBS gate on PRs.** The client spec 013 FR-030 requires an SWBS code before a PR is submitted, and SWBS codes come from GSAP. The FA shows Hive9's system gate is the **approved activity**; the SWBS code is what the requisitioner needs to build the GSAP PR. Decide one of these:
- **(a)** keep the gate, so PRs can be drafted but not submitted until a code exists;
- **(b)** make the SWBS gate a setting that is off until GSAP access;
- **(c)** hold the PR feature as a whole until GSAP access;
- **(d)** allow submission without the code, flag "SWBS code missing" on the PR form, and fill it when Finance enters the code (L4a).

We recommend **(d) with L4a**, which matches Hive9's own gate and the FA's freeze-period remedy, if Shell confirms. Otherwise **(a) with L4a**.

---

## 7. Proposed Purchase Request model (for discussion)

Updated with the FA (C4, D3–D7). Everything here needs **no GSAP** except the feed-only fields, which **stay empty while GSAP is on hold**.

**PurchaseRequest** (`PR-1001`, embeds `record.Meta`):

| Field | Notes |
|---|---|
| Basics | name, description, owner, creator |
| `planId` | for scope; all lines in one plan |
| `companyCode` | from the first line's activity; one per PR (GSAP refuses cross-company charging) |
| `type` | Standard PR/PO · Manual PR/PO · FI Payment. A required choice, not a workflow switch: nothing in Hive9 shows it changes behaviour (FA, inferred) |
| `status` | **Created → Submitted → In-Review → Approved / Declined → Closed** (Hive9's six). "Released" is the GSAP release, not a MAPP status; a modification round starts from Approved |
| `approvalWorkflowId`, `modificationWorkflowId` | approval through L1 |
| `moaApprover` | a person, used as the approver (record-field source, L1); checked against the MOA matrix (company code × amount band) |
| Attributes | Material/Service, short text, vendor, vendor code, Outline Agreement, e-mail group, recipients, **contract holder**; China-only OA vendor and sub-category. Values from the PR master lists (L14) |
| `sentAt` | routing date (Hive9's "Sent Date") |
| `deliveryDate` | rolled up from the lines |
| `gsapPrNumber`, `gsapPrDate` | read-only; feed only; **empty while GSAP is on hold** |
| `approvedSnapshot` | used during a modification round |

**PurchaseRequestLine:**

| Field | Notes |
|---|---|
| `lineItemId` | unique per PR; default amount = the line item's unreserved balance |
| `hxKey` | **derived**, not stored: "HX" + line-item number, shown with copy on the line item and the PR form (L18) |
| Money | currency, amount (minor units), overall limit, delivery date, closed amount |
| PO and invoice fields | PO number, line, line status, date, value, copy link; invoice amount; GRSP amount. Feed only; **empty while GSAP is on hold** |

**Rules:**
- **Reserved** = the sum of PR lines in Created, Submitted, In-Review, **Declined** and Approved, in full, plus Closed PRs at their closed amount. Only delete (or a cancel, if Shell defines one) releases the money.
  - It is checked in the line item's currency, inside the same `Write` transaction.
  - Conversion honours the per-currency "monthly rates for PRs" switch (L7).
  - A restore from the bin re-checks the balance.
- Planned cost cannot drop below the reserved amount.
- **Total PR Amount** shows on the line item and rolls up to the activity. It stays separate from Committed, which stays PO-only.
- **Creation** is allowed in any activity status; **submission** needs the approved activity. For the SWBS code, see the four options in §6.1.
- **Only PRs that are not approved go to the bin.** Approved PRs can only be **closed out**. Closing "to the invoiced amount" needs GSAP invoices; until then the user types the amount.
- **Organisation setting:** "lines on an approved PR" (three options; Shell's value "1" to be explained).
- **Routing:** automatic on approval, as an in-app record plus a downloadable PR form (Excel or CSV) with the FA's columns: HX key, short text, cost centre, WBS, GL, amount, MOA approver, documents. No e-mail (constitution).

**Access** (FA F4: rights to edit PRs are separate from seeing the PR grid):
- **Permissions:** `pr.edit` (create and edit), `pr.delete`, `pr.submit`, and `pr.list` (the Finance › Purchase requests page).
- **No routing permission:** routing is automatic.
- **Requisitioner (TA)** = Viewer + `pr.list`.
- **Marketers:** `pr.edit` and `pr.submit`, without `pr.list` if Shell keeps Hive9's rule.

---

## 8. Work still open on our side (inside the current scope)

These are open in our own feature map and task lists, whatever the client specs say. Each was confirmed as an unticked task in `specs/*/tasks.md`. The last column shows whether the internal build-status note lists it.

| Feature | Open items | In status note? |
|---|---|---|
| 012 Bulk actions and import | Spec **Draft**. The contract and access rules exist, but the 11 operations are listed as not served in `contract_routes_test.go` and answer 404; there are no screens. **Decision needed: build it, or take it out of the contract.** | Yes |
| 016 Planning and currency | Attribute filters, Breakdown tab and "Planned cost by" card (T074–T077); field-help tips on the record screens (T084–T088) | Yes |
| 016 Planning and currency | Import GSAP rates route (T028), which works on the **simulated** rate file and doesn't need GSAP access; e2e (T089) | No |
| 014 Access control | Role-aware screens (T043–T051): a Viewer still sees buttons they can't use. The API refuses the action, so this is a **UX gap, not a security hole**. | Yes |
| 014 Access control | e2e for the access journeys (T061) | No |
| 017 Approval rules | Approvals read roles from a stand-in rather than from 014 (T001, T008); line-manager field on People (T036) | Yes |
| 017 Approval rules | e2e (T047–T049) | No |
| 015 Financial transactions | Linked-transactions sentence in some delete dialogs (T045) | Yes |
| 015 Financial transactions | e2e (T021, T025, T030, T035); Home no-access state (T044) | No |
| 008 Activity approvals | e2e (T055–T056) | No |
| 001 App shell | End-to-end accessibility test (T080) | No |
| 013 BCPIR and SP&A | Spec **Draft**, gated on Q-001; no code | No |
| 019 Budget-increase approval | **The spec folder is empty**; there is no spec and no code | No |
| Docs (FA) | `docs/product/field-catalogue.md` says "13 marketing GL codes" while the seed has 5; it also states a Cluster → Region dependency the FA does not show | No |

**Missing from the status note:** the end-to-end tests (12 tasks across 001, 008, 014, 015, 016 and 017), the 013 and 019 specs, and two small items (016 T028, 015 T044). The e2e gaps matter before UAT: CLAUDE.md requires an end-to-end test for every P1 story.

### 8.1 Build health, checked against the status note

| Status-note claim | Measured on 27 Oct 2026 | Verdict |
|---|---|---|
| Backend compiles clean | `go build ./...` succeeds | Confirmed |
| 36 of 36 backend test packages pass | `go test -short ./...`: 36 ok, 0 failed (22 packages have no tests) | Confirmed |
| 1,501 Go tests | 1,501 `func Test…` | Confirmed |
| Zero TODO/FIXME in Go | 0 | Confirmed |
| ~78,000 lines of Go | ~74,300 lines of non-test, non-generated Go; ~143,500 including tests and sqlc output | Depends on how you count |
| 665 frontend source files, 351 test files | 728 non-test `.ts`/`.tsx` files (excluding generated API types); 351 test files | Test files confirmed; source count depends on how you count |
| 41 browser tests | 40 `test(...)` cases in 15 Playwright files | Off by one |
| 176 API endpoints | 176 paths, 234 operations | Confirmed (as paths) |
| Every sidebar section has a real page | `COMING_SOON_SECTIONS` is empty and no Settings card is marked "coming soon" (the mechanism is still in the code) | Confirmed |
| GSAP transactions, rates, Entra ID sign-in and the nightly approvals job are simulated by design | Matches ADR-0005, ADR-0009, D-004, D-242, D-259, D-308 | Confirmed; the client-target gaps for these are in §5 E and §9 |

The short test run skips the scope-isolation tests that need PostgreSQL (`-short`). The full `go test ./...` run with `TEST_DATABASE_URL` was not part of this check.

---

## 9. Scope decisions needed before PR/PO work starts

1. **Bring purchase requests into scope.** This needs a constitution amendment (Current Phase & Scope), CLAUDE.md non-negotiable 13, feature-map rule 6, and replacing D-243. The BRD marks BUS.FIN.020 as Future/Later. The FA shows the MAPP side of a PR needs no GSAP.
2. **E-mail.** Keep it excluded and route PRs in-app with a downloadable PR form, or allow outbound e-mail through an adapter (constitution change). Hive9 e-mails the form to a mailbox (FA D3 1.5).
3. **Outbound file to GSAP.** Only the **SWBS request** (FSSI 1523); Hive9 sends no PR data by file (FA D4). Needs a new ADR, since ADR-0009 covers inbound simulated feeds only, and reversing D-255. **On hold:** there is no GSAP access to send to. Separately, storing a code that **Finance enters** (L4a) can be decided now.
4. **BRD "Later" items moving forward?** Invoice approval (BUS.FIN.050, ⏸), documents and PO PDFs (BUS.FIN.060), budget hard stop (BUS.BUD.030, which conflicts with D-015 and D-235).
5. **Finance Budgets module.** The hard stop is a **Finance Budget row** setting in Hive9 (FA C2), so it can only come with this module. Accrual calculation stays out; Hive9's outbound Accrual Requests are also ⏸. PR reservation itself needs neither.
6. **The actuals model.** Keep Paid = actuals (D-243, D-248) on the simulated feed, or move to CO postings and split links (client 012). **On hold:** this depends on the real GSAP actual-cost extract. Rates (annual and monthly) can change now (L7).
7. **Which decision log is authoritative.** The client specs' D-numbers clash with ours.
8. **GSAP access itself.** Who provides a GSAP test system or file drop, the integration technical document, the layouts (FSSI 1523, PGSI 941, actual-cost extract) and sample files, and when? Integrations are frozen until about March 2027 (FA D1). Every on-hold item in §6.1 waits on this. Until then the SWBS gate on PRs needs a choice (§6.1).
9. **(FA) HX numbering and legacy keys.** Open Hive9-era PRs and POs carry HX keys built from Hive9 line-item IDs. Decide how MAPP numbers line items and matches legacy keys (L18) **before** any data is loaded.
10. **(FA) Cut-over.** Decide what happens at go-live to Hive9 records in flight: Submitted activities, open PRs, POs, and the status mapping (L19). Hive9's own advice: leave nothing in Submitted when workflows switch (FA C6).

---

## 10. Recommended roadmap

| Phase | Content | Depends on |
|---|---|---|
| **0. Finish for UAT** (the status note's 1–2 weeks) | Every open task in §8, including the missing e2e tests · decide whether 012 is built or taken out of the contract · Q1 quick fixes (GL codes and finance lists where the FA gives them) · show the HX key · write or remove the empty 019 spec | — |
| **1. Planning rule alignment** | **L3 re-approval rule (High: it blocks the as-is flow)** · L2 per-workflow approval settings · **L17 permission split** · L11 owners and team flags · L12 delete protection · L14 master data with codes and Shell's real lists · L18 HX numbering decision · build 012 bulk and import | Client answers (§11 B, C, D) |
| **2. Engine and finance foundations** | L1 approve any record type (with record-owner and record-field approvers) · L7 exchange rates in MAPP · L9 Latest Estimate · L15 audit fields · L4a SWBS code entered by Finance · L19 cut-over design | Phase 1; rate decision |
| **3. PR/PO in MAPP** (after the scope decision) | L5 Purchase Request (MAPP side) · MOA matrix · PR master lists · PR list page · automatic in-app routing and PR form · SWBS gate handled per §6.1 | §9 decisions 1–2; L1, L4a |
| **4. BC/PIR, SP&A, budgets** | L10 BC and PIR records, SP&A and BCPIR views, PPT export · L16 notifications module · L8 hard stop and release (if the Finance Budgets module is brought in) | Q-001; L1; §9 decision 5 |
| **⏸ On hold: GSAP** | L4b SWBS round trip · L6 GSAP matching and richer inbound fields · PR and PO numbers back in MAPP · invoice link and invoice approval · SE/GR · daily actuals · PR top-up in GSAP · rejected-SWBS and unmatched-HX handling | (a) Shell's technical document and samples → file-import layouts; (b) live connection, not before ~March 2027 (§9 decision 8) |
| **Later** | Documents, Work board (026), templates, snapshots, calendar feed, multi-year linking, Media Code and Measure lineage fields | Client priorities |

---

## 11. Open questions for the client

Answered or narrowed by the FA:
- **SWBS:** one code per line item, requested when the activity is approved and "Create SWBS code" is ticked (default on).
- **Latest Estimate:** LE1–LE4 plus locked copies, single amounts on the activity.
- **"Released":** the GSAP release.
- **PRs are not sent to GSAP:** they are e-mailed and retyped.

**A. Purchase requests and GSAP**
1. Is a PR raised in MAPP and handed to the requisitioner (as in Hive9: approved form e-mailed, then retyped into GSAP), or raised in GSAP only? Can one PR span several line items, and if so, which HX key goes into the GSAP tracking number?
2. **PR types:** do FI Payment and Manual PR change MAPP's behaviour, or only the requisitioner's?
3. Does a Declined PR keep its reservation until deleted? What is "cancel", given there is no Cancelled status? Is the reservation checked against the linked finance budget row as well as the line item?
4. Without e-mail, is automatic in-app routing plus a downloadable PR form acceptable for the requisitioner (TA/Manila) and AP?
5. What does the PR setting "lines on approved PR = 1" mean?
6. **Self-approval of PRs:** the workshop says marketers self-approve the PR form, but MAPP forbids self-approval by default (D-302). Must PRs allow it?
7. Who owns the **MOA matrix**, and can MAPP hold it? Do thresholds vary by country or class of business?
8. Are vendor, OA, e-mail group, recipient and contract-holder lists maintained by a Shell admin in MAPP, or taken from GSAP?
9. Should MAPP guide the "3 ways to buy" choice (GCC check) before a PR?
10. Does "budget release" in the deck mean periodic release of finance budget, or freeing a PR's reservation?
11. Are Accrual Requests raised in MAPP and sent to GSAP (as in Hive9), or only read from GSAP?

**B. GSAP access, the freeze and cut-over**
12. **GSAP access:** when, and through whom, will we get the integration technical document, sample files and a test system or file drop? Will Nucleus / S/4HANA change FSSI 1523 or PGSI 941?
13. **SWBS during the freeze:** will Finance create codes manually and give them to us (one at a time or as a file)? Does the code encode the company code? Is an SWBS code required to **submit** a PR, or only for the requisitioner (§6.1 options a–d)? What happens if GSAP rejects a request or misses 48 hours (for when access exists)?
14. During the freeze, may requisitioners type GSAP PR and PO numbers into MAPP by hand? Is a manual "work delivered" confirmation an acceptable stand-in for invoice approval?
15. **HX numbering:** keep Hive9's line-item IDs, start MAPP above Hive9's highest ID, or map legacy keys, so that open Hive9-era POs and invoices still match?
16. **Cut-over:** what happens to Hive9 PRs that are Submitted, In-Review or Approved, to open approval rounds, and to their GSAP POs? How do GroupSubmitted, GroupApproved, In-Progress, Complete, Accepted and Committed map onto MAPP?
17. Can we see the definitions of the **SWBS Not Received, Pending PR and PR Track Invoices** dashboards?

**C. Planning rules**
18. **Exchange rates:** who maintains them, and are "use monthly rates" for actuals and for PRs switched on per currency in production? Should a rate change alter earlier years' USD values?
19. **Re-approval:** when a line item only splits the unassigned balance of an approved activity, should that restart approval? Planned cost only, switched per model?
20. **Hard stop:** confirm it sits on the Finance Budget row. Block or warn? Who approves a budget increase or release? What are the containment rules (e.g. initiative budget within campaign)?
21. **Latest Estimate:** who locks a round, and when? In USD or the plan currency?
22. **BC/PIR:** the deck shows records with approval and attachments that Hive9 does not have. Are they required, and what goes into the PIR PPT export?
23. **ZBB cost hierarchy:** live Hive9 has 10 BU-prefixed values; the deck has 7 other labels. Which is right? Single or multiple choice?
24. **Multi-year and dates:** cross-year activities are confirmed. Must initiatives and activities stay inside their parent's dates, given that a live initiative doesn't?
25. **Value lists:** please export every attribute list with codes, including Cost Centre (1,217), Company Code (61) and Country/OU (125). Which is authoritative, live Hive9 or the deck? Is there a real Spend Type → GL Code and Cluster → Region mapping? Should finance and initiative attributes be hidden on Mobility plans?
26. **Attribute shapes:** single choice, multiple choice or % split for Country/OU, Channel, Global Campaign, Growth Ambition, Brand? FIT4 values per level and per BU; do FIT4 ID, Innovation and "Actual Financials Rating" exist?
27. May drafts save with required fields empty (MAPP), where Hive9 refuses to save? Should the finance-budget link be checked at save or submit?
28. Are Date Sets, Media Codes, Related Activities, Related Plans and Custom Goals used in production? Is allocation by quarter needed? Should the unassigned balance follow the plan's default allocation?
29. Is MAPP's confirm-first behaviour on an activity type change acceptable, in place of Hive9's silent planned-cost reset?
30. **Plan publishing:** is MAPP's rule acceptable that a plan cannot be published until its campaigns and initiatives are complete? Keep or remove Unpublish?
31. **Views and editing:** are inline editing and drag-to-reschedule must-haves if every edit shows "this sends N activities back for approval"?
32. **Deletes and retention:** refuse deletes with transactions or PRs? What are Shell's retention periods for the bin and the audit log?

**D. Approvals, roles and administration**
33. Are the campaign "Statuses & Approvals" and initiative "SMP Initiative – Footer" group workflows used in production? Who approves them, and does GroupApproved gate anything? Should a campaign submit **approve** never-submitted children, bypassing their own workflows?
34. Which workflows use **Record Owner** or **Custom Attribute** approvers? Should a level's default be "All must sign off", with A/B/C labels?
35. **Self-approval and on-behalf:** set per workflow with "allowed" as the default? May an Administrator decide on behalf of someone else? Is dated delegation (≤ 90 days) acceptable in place of an on/off toggle?
36. **Roles:** which of the 19 production roles do we keep? Must campaign and initiative rights differ (Marketer, VP/GM, Mkg. Admin support)? Can Shell walk us through how each role is used day to day?
37. Should the plan audit log and the recycle bin stay **admin-only** as in Hive9?
38. Are record **comments** and a Commenter role needed in this phase? Is admin impersonation with user consent needed (after SSO)?
39. Confirm the **Shell-side administrator** persona (Business Champion?) and whether it also maintains organisation setup and models. Please send the full vendor-maintained configuration: GL codes, spend types, dropdowns, 167 activity types, Tactic Categories, and the organisation switches marked "Unknown" (time-out, unallocated amounts, default team flags, calendar export, use planned cost as actual).
40. Which Hive9 bulk actions are actually used: Change Status (which statuses by hand?), Package/Unpackage, Link To?
41. **Work board (026):** has Shell approved it for this phase?

---

## Appendix: evidence

**Client material:**
- `context/PRISM SP&A Demo v1.0 1 (1).pptx`: slides 4–7, 9–11, 13, 14, 16, 18.
- `context/Slide 7 - SP&A Campaign workflow (HD).png`.
- `context/Hive9_Functional_Analysis 2.docx`: parts A–D, F–H and Appendices 1–3, 8 screenshots, plus the attribute value lists in E3.0.2. Hive9 10.9.0 sandbox, User & Admin Guide, GSAP–Hive9 workshop of 10 Sep 2026.
- `context/specs/*`: 16 specs in scope.

**MAPP sources:**
- `specs/001–029` and `docs/sdd/feature-map.md`, `decision-log.md`.
- `docs/product/domain-model.md`, `field-catalogue.md`.
- `backend/internal/{plan,campaign,activity,lineitem,approval,financials,budget,settings,people,access,history,allwork,rollup}`.
- `backend/api/openapi.yaml` (176 paths, 234 operations) and `backend/seed/settings.go`.
- The internal build-status note (§8.1), checked on 27 October 2026 with `go build ./...`, `go test -short ./...`, counts over the source tree, `contract_routes_test.go` and the open tasks in `specs/*/tasks.md`.
- `frontend/src/features/*`.

**Spot-checked in the code:**
- ZBB seed values (`seed/settings.go:153–154`).
- Seeded GL codes 7250010, 7250040, 7250090 are not in the FA's list of Shell's 13 (`seed/settings.go:180–186`).
- Plan Unpublish (`plan/status.go:61`).
- Re-submit only from Created or Declined (`approval/rounds_submission.go:80`).
- Line-item numbers start at 1001 (`platform/ids/ids.go:67`).
- Campaign and initiative share `campaigns.edit` and `campaigns.delete` (`platform/authz/catalogue.go`).
- Activity log and recycle bin are open to every signed-in user (`access/routes_views.go:36–38`).
