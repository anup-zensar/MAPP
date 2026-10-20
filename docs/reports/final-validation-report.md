# Final validation report: NorthStar against Shell's MVP requirements

| | |
|---|---|
| **Deliverable** | Final validation report confirming alignment with client requirements and MVP scope |
| **For** | The Shell client team and the product owner |
| **Date** | 21 October 2026 |
| **Application** | NorthStar Plan module, branch `plan-module` after the final merge (commit `a1ae8fe`): features 001–011 and 014–017 integrated (012 bulk actions and import is deferred, 013 BCPIR and SP&A waits on Shell's question Q-001) |
| **Requirements** | Shell's Business Requirements document, 36 rows, of which 11 are **MVP Requirement** ([transcription](../reference/brd-requirements.md)) |
| **Starting point** | [BRD gap analysis](brd-gap-analysis.md): of the 11 MVP rows, 2 met, 6 partly met, 3 not met |
| **How the gaps were closed** | [BRD gap-closure plan](brd-gap-closure-plan.md), features [014](../../specs/014-access-control/spec.md), [015](../../specs/015-financial-transactions/spec.md), [016](../../specs/016-planning-and-currency-extensions/spec.md), [017](../../specs/017-approval-rules/spec.md) and the [011 merge](../../specs/011-activity-log-and-recycle-bin/spec.md) |
| **UI work** | [UX/UI recommendations](ux-design-recommendations.md), [UI/UX implementation plan](ui-ux-implementation-plan.md), [design system v2](../product/design-system.md) |

**Status key.** **Met**: the requirement works from start to finish. **Met, finish open**: the requirement
works, and a named screen detail or test is still open (section 7). **Partly met**: a named part of the
requirement cannot be used yet. **Not met**: nothing usable exists. **Not applicable**: not a product
requirement (moved to the technical track, or a vendor question).

**What "simulated" means here.** Shell's systems are not connected yet. GSAP transactions and GSAP daily
exchange rates come in through a file import and demo data, single sign-on is a demo sign-in, and the nightly
approvals job is a button. Each sits behind an adapter, so the real feed replaces the simulation without
changing screens or rules.

---

## 1. Executive summary

- **All 11 MVP requirements now have working software behind them.** Before this round, 3 were not met and
  6 were partly met. Now:

  | Status | Count | Requirements |
  |---|---|---|
  | Met | 5 | BUS.PLN.020 multi-year and plan types, BUS.PLN.040 monthly budgets and split, BUS.PLN.050 re-use of last year, BUS.WFL.030 missing mandatory data, BUS.BUD.050 currency |
  | Met, finish open | 4 | BUS.ADM.020 roles and access, BUS.BUD.040 global and local budgets, BUS.FIN.080 financial transactions, BUS.WFL.010 approvals |
  | Partly met | 2 | BUS.BUD.020 budget categories (the screens for filtering and breaking down by category are not built; the API is), BUS.UX.020 field help (administrators edit the help for every field, but most forms do not show those tips yet) |
  | Not met | 0 | |

- **Four new features close the gaps:** 014 access control and global budgets, 015 financial transactions,
  016 planning and currency extensions, 017 approval rules. Feature **011** (activity log, record history and
  recycle bin) is complete and merged. It provides the audit trail that the approvals requirement asks for.
- **The four "finish open" items are small.** Some screens still show buttons that a person's role does not
  allow; the API refuses them with a clear message. Approvals still take people's roles from a stand-in that
  mirrors the demo roles, so a role changed in Settings does not yet change who may approve. A line manager cannot yet be set
  on the People screen. Section 7 lists each item
  with its task.
- **NorthStar now follows Shell's design language**: Shell yellow for the one primary action, a light-grey
  page with white cards, Jost headings and Arial text, WCAG 2.2 AA, red only for danger. The foundation and
  the page-by-page pass of all eight areas are done (section 4).
- **Shell has 58 defaults to confirm (section 8).** Every one is applied now and can be changed by
  configuration or a small change. The most important: the five roles, hiding plans outside a person's
  markets, the meaning of global and local budgets, the daily-rate policy, the meaning of "split", and the
  approval rules. Shell still owes **"the note on financial transactions"** that BUS.FIN.080 refers to.
- **Test results** are in section 6: Go tests, web unit and component tests, and two full end-to-end runs on
  fresh demo data.

---

## 2. The 11 MVP requirements

Defaults are numbered as in section 8. FR numbers are the functional requirements of each feature's spec.

### 2.1 BUS.ADM.020 — Role- and team-based access control; edit vs view only

> *Team-based and role-based access control (RBAC) for plans, POs, invoices and related financial data,
> configurable through access rules rather than code. Client: flexibility to control who can edit vs who can
> only view.*

**Status: Met, finish open.** Before: Not met (every signed-in user could see and change everything).

**How it is met (014 FR-001–FR-028, FR-037–FR-041)**
- 25 fixed permissions and five roles (Administrator, Global planner, Planner, Approver, Viewer). Shell edits
  the roles, adds roles and assigns them in **Settings › Roles and access** (People, Roles and Rules tabs,
  Recent access changes). No code change is needed.
- A role applies to **All markets**, **My teams** or **Chosen teams** (a market is a team). A person sees
  only the plans in their scope, and everything under them: campaigns, activities, line items, approvals,
  All work, Home, the activity log, the recycle bin and financial data.
- Every API operation has a declared access rule and refuses by default ("fail closed"). A refusal answers
  "You don't have access to this …" and shows nothing of the record.
- Each answer carries what the reader may do with the record, so screens can hide or explain actions. My
  profile shows **Your access**. A person without any role sees whom to ask on Home.
- Financial data (015 FR-028) and exchange rates and field help (016 FR-049) use the same permissions.
  Approvals (014 FR-025–FR-028) apply the same five roles, read for now from a stand-in (see below).

**What is still open**
- Several pages still show actions that the reader's role does not allow, such as Edit or Allocate. Pressing
  one gives the API's refusal message instead of a hidden button or a "View only" note. The shared parts
  (`useAccess`, `ViewOnlyNote`, `NoAccess`, `RequirePermission`, the filtered sidebar) are built; applying
  them on every page is open (014 T043–T051, 015 T044).
- Approvals read people's roles from a stand-in (`backend/internal/app/approvals_access.go`) that gives the
  demo people their seeded roles. The demo behaves correctly, but a role given or removed in Settings › Roles
  and access does not yet change who may approve or decide on behalf (017 T001, T008).
- End to end: `brd-gaps.spec.ts` covers a role change and a viewer's read-only plan (014 T061).

**Evidence**
- Go: `TestAccessMatrix` checks that every served operation refuses each demo persona whose rule excludes
  them, with the exact message (014 SC-001). `TestAnswers_CarryWhatTheReaderMayDo_FR011` checks what each
  answer says the reader may do. A route-table test checks that every operation of the contract has a rule.
  There are 81 test functions in the access, authorisation and access-enforcement tests.
- Web: component tests, with an axe check (WCAG 2.2 AA), of the People, Roles and Rules tabs, the role page
  and the role form.
- End to end: the existing 29 journeys sign in as the persona whose seeded role allows each step (014 T062).
  All 29 passed on the 014 branch.

**Defaults for Shell to confirm:** 1–10.

### 2.2 BUS.BUD.020 — Budget categories; client-updatable attributes; flexible framework

> *Support various budget categories, e.g. region, brand, spend type, business unit, channel. Client:
> flexibility to update attributes; the tool should accommodate the business's framework or hierarchy.*

**Status: Partly met.** Before: Partly met.

**How it is met**
- The categories are Shell attributes that administrators add and change themselves, at every planning
  level, with dependent lists for hierarchies (002, unchanged).
- The API now filters All work by any Shell attribute (with "Not filled in"), answers a **Breakdown** by one
  or two attributes with totals, and exports it with the year range and the rates used (016 FR-039–FR-043).

**What is still open:** the screens for these: **More filters** by attribute in All work, the **Breakdown**
tab, and the **Planned cost by** card on the plan page (016 T074–T077). Until they are built, a planner cannot
analyse budgets by category on screen. Exporting All work and pivoting in Excel is the workaround.

**Evidence:** Go handler and service tests named after 016 FR-039–FR-043, on the 016 backend.

**Defaults for Shell to confirm:** 54 (budgets are reported by category, not set per category), 55.

### 2.3 BUS.BUD.040 — Global and local budget allocation, with visibility for each owner

> *Ability to Global/Local budget allocation with appropriate visibility for global and local budget owners
> respectively.*

**Status: Met, finish open.** Before: Not met.

**How it is met (014 FR-004–FR-007, FR-029–FR-036, FR-040)**
- **Global budgets** in the sidebar after Plans: the list for a year, and each global budget's page with
  Global budget, Allocated, Unallocated and Planned cost. Over-allocation shows "Over-allocated by X" in red
  and never blocks.
- A global planner **allocates** to market plans. The allocation becomes the plan's budget, is read-only for
  the local owner, and the plan shows "Allocated from …".
- Local owners never see other markets' allocations. The global budget page adds them up as one
  **Other markets** row.
- A global budget is in USD; each allocation is in its plan's currency.

**What is still open:** the global budget pages show every action to everyone; the API refuses those a
person may not use (same item as 2.1).

**Evidence:** Go: 33 test functions in the global budgets package (validation, figures, Other markets,
allocations, recycle bin and restore) and handler tests on the demo data. Web: component tests of the list,
page, form and allocation dialog. Checked at 1440 × 900 and 390 × 844, light and dark, axe with 0 violations.

**Defaults for Shell to confirm:** 11–14.

### 2.4 BUS.BUD.050 — Currency conversion, exchange-rate management, GSAP daily rates

> *Support currency conversion in planning, analysis and reporting, exchange-rate management, and whether it
> can consume daily conversion rates from GSAP. Client: ability to plan in local currency and USD.*

**Status: Met.** Before: Partly met (USD-only budgets and costs, one rate per currency with no
history).

**How it is met (016 FR-001–FR-019)**
- **Dated exchange rates with history** in Settings › Currencies: add, edit and delete a rate, with a
  question that names the amounts affected before saving.
- **Import GSAP rates**: upload the GSAP daily-rate file, review the rows grouped by outcome, import, and see
  recent imports. This is simulated now; the live GSAP feed replaces the file later.
- **Plan currency**: each plan has a currency (USD by default). Budgets, phasing and planned costs are
  entered in it. The reader's display currency is shown beside it, and the rate used and its date are shown
  on the summary cards. A line item can be in any currency and shows both rates.
- Financial transactions convert to USD at the current rate (015 FR-010).

**What was open:** the **Import GSAP rates** page was not in the app's routes. It is routed now
(`/settings/currencies/import`, 016 T028), found and fixed in the end-to-end run (section 6).

**Evidence:** Go: tests of the money conversion, the dated rates, the rate import and the plan-currency rules
(016 backend, 21 operations served). Web: component tests of the rate history, the rate dialog, the GSAP rate
import and recent imports, each with an axe check.

**Defaults for Shell to confirm:** 42–48.

### 2.5 BUS.FIN.080 — Financial transactions prominent in the main working views

> *Are financial transactions (POs, invoices, payments, accruals) surfaced prominently in the user's main
> working views rather than buried several levels deep. Client: simple to access and intuitive.*

**Status: Met, finish open.** Before: Not met (a collapsed "From finance (coming later)" section).

**How it is met (015 FR-001–FR-031)**
- Read-only GSAP transactions (purchase-order lines, invoices with credit notes, payments, accruals) are
  matched to line items. They come from a simulated GSAP extract and demo data.
- **Financials** sections, open by default, on the activity page (right under the summary), the line item
  page, the plan page and the campaign and initiative panels: Committed against planned cost with a meter,
  Invoiced, Paid and Accrued. Committed above planned cost is said in words and red, never by colour alone.
- **All work**: Committed and Invoiced columns and a Financials filter. **Home**: a Financials card under the
  summary cards.
- A **Financials** page in the sidebar after All work: tabs per type and **Not linked**, search and filters,
  the details panel, **Import GSAP extract** with a check before saving and an import history, and a manual
  **Link to a line item** for unmatched transactions.
- The line item and activity delete dialogs say how many GSAP transactions stay linked. The activity log
  links to each transaction.

**What is still open**
- Shell's "note on financial transactions" (default 29). The placements above are shared components and
  cheap to move.
- The Financials page shows every action to everyone; the API refuses import and linking to people without
  the permission (same item as 2.1). The Home Financials card has no "no access" text yet (015 T044).
- The campaign, initiative and plan delete dialogs do not yet say how many transactions stay linked (015 T045).
- End to end: `brd-gaps.spec.ts` covers an activity's financials and the Financials page; the import and
  linking journeys (015 T025, T030, T035) are not written.

**Evidence:** Go: 62 test functions in the financials package, including the extract's column rules
(FR-004, FR-005), read-only, empty and delete-dialog counts, and SC-004 stress tests for import and list.
Web: component tests of the Financials page, details panel, import and link dialogs, the Home card and the
record sections; screens checked at both widths, light and dark, axe with 0 violations.

**Defaults for Shell to confirm:** 29–41.

### 2.6 BUS.PLN.020 — Multi-year, annual, quarterly and monthly; strategic, operational, cross-market

**Status: Met** (as views across single-year plans, default 53). Before: Partly met.

**How it is met (016 FR-021, FR-024, FR-032–FR-038)**
- **Plan type** (Operational, Strategic, Cross-market) on every plan, beside the Strategic/AOP flag. It is
  shown in the header and can be filtered on.
- **Years range** on the Plans list, and up to 5 years in All work. A range opens Budget & cost on
  **Yearly**, with Quarterly and Monthly a click away. The calendar runs across the years of the range.
- Budget by month and by quarter (see 2.7).

**Evidence:** Go: `TestQuery_CalendarOverRange_FR036`, `TestUpdatePlan_PlanTypeLogged_FR032_FR049` and the
year-range and plan-type filter tests. Web: Plans filter and All work filter tests.

**Defaults for Shell to confirm:** 52, 53.

### 2.7 BUS.PLN.040 — Split the budget by month; split an activity from its original budget

> *Client: phasing and ongoing updating of budget.*

**Status: Met.** Before: Partly met (costs were phased by month, budgets were one annual amount, no split).

**How it is met (016 FR-020–FR-031)**
- **Budget by month** for plans, campaigns and initiatives: Not phased, Evenly or Custom, entered by month
  or quarter, with Split evenly and Copy from planned cost. The card compares the phased budget with the
  planned cost month by month and shows "Over by X" in red. A custom phasing that does not add up stops
  publishing, and the problem links straight to the card.
- **Compare with phased budget** in All work's Budget & cost view.
- **Split activity** on the activity page and on its row: a new activity in the same plan (it can sit under
  another initiative), whole line items moved, a preview of both activities, and "Split from / Split into"
  links. The activity log keeps the record.

**Evidence:** Go: `TestSplitOptions_Contract_FR026`, `TestSplitPreview_MatchesSplit_FR027`,
`TestSplit_SumsUnchanged_SC005` and the phasing rules tests. Web: phasing arithmetic tests (whole cents) and
the split dialog tests.

**Defaults for Shell to confirm:** 49–51.

### 2.8 BUS.PLN.050 — Re-use prior-year planning content

**Status: Met.** Unchanged from the gap analysis: a whole plan copies into next year with every campaign,
initiative, activity and line item, dates moved one year and statuses reset. The new features keep copies
correct: a copy never takes a global allocation (014 FR-035), never copies transactions (015 FR-030), and
keeps the plan currency, type and phasing, shifting the phasing to the new year (016 FR-025).

**Not included:** copying a single campaign or activity into another year, and a "structure only" copy. Both
are small follow-ups (section 7).

**Evidence:** the plan-copy journey in the end-to-end suite (`core-flow.spec.ts`); 016 copy tests.

### 2.9 BUS.UX.020 — Contextual help maintained by Shell administrators

**Status: Partly met.** Before: Partly met.

**How it is met (016 FR-045–FR-048)**
- **Settings › Field help** lists every built-in field and calculated value, by level, with search and
  Default / Edited chips. Administrators edit the text (up to 300 characters) or reset it to the default.
- Shell attributes keep their own help, edited on each attribute's page. The attribute list shows a
  **No help yet** chip and filter.

**What is still open:** the tips on the plan, campaign, initiative, activity and line item forms, record pages
and summary cards do not yet read the texts edited in Field help (016 T084–T088). The ready-made tip
(`FieldHelpTip`) exists; placing it on those screens is the remaining work. Until then, the existing fixed
tips show.

**Evidence:** Go: field help catalogue, edit and reset tests with version checks. Web: Field help page tests
(list, search, level tabs, edit, 300-character limit, empty text, conflict, reset, axe).

**Defaults for Shell to confirm:** 56.

### 2.10 BUS.WFL.010 — Multi-level, parallel and conditional approvals, delegation, notifications, escalation, roles, segregation of duties, audit, monitoring

> *Client: multi-layered approval with auto approval / manager approval; simplify approval workflow,
> delegation and control, tracking and notification.*

**Status: Met, finish open.** Before: Partly met (ordered levels only).

**How it is met (017 FR-001–FR-037; 014 FR-025–FR-028; 011)**
- Each level of an approval workflow says **who approves**: named people, the owner's **line manager**, or
  people with a **role** that approves. **Any one** or **all** must approve (parallel approval).
- **Only needed when** and **Approve automatically when** conditions on planned cost (USD), business unit and
  activity type (conditional and automatic approval). A **backup approver** per level.
- **Submit** shows what each level will do, including "This activity will be approved automatically."
- **Delegation**: "Delegate my approvals" with a banner saying who approves for whom today.
- **Reminders and escalation** after set days, with **Run reminders and escalations** until NorthStar is
  hosted (then a nightly job).
- **Monitoring** tab: overdue, escalated, average time to decision, automatic approvals, waiting by level,
  and how long approvals have waited.
- **Segregation of duties**: nobody approves what they own or submitted (on by default). Only people whose
  role allows it decide (roles read from the demo roles for now; see "still open").
- **Notifications** in the bell for automatic approvals, reminders and escalations.
- **Audit trail**: the approval history on each activity, and feature 011's activity log and record History.

**What is still open**
- The line manager field on the People screen (017 T036). Line managers come from the demo data until then.
- Approvals take people's roles from a stand-in that mirrors the seeded roles, not yet from Settings › Roles
  and access (017 T001, T008; same item as 2.1).
- End to end: `brd-gaps.spec.ts` covers automatic approval and a delegate's approval; conditional and
  all-must-approve levels have Go tests only (017 T048).

**Evidence:** Go: 136 test functions in the approvals package (rule engine, resolution, authority,
delegations, escalations, monitoring, audit), for example `TestAuthority_Table_FR017_FR018_FR020_SC002`,
`TestSubmit_WF1003Automatic_FR014` and `TestApprove_BySubmitter403_FR018`. Web: component tests of the level
editor, submit dialog, chain, history, delegation panel and monitoring view. End to end: the existing
approval journeys and 011's log journeys.

**Defaults for Shell to confirm:** 15–28.

### 2.11 BUS.WFL.030 — Highlight missing mandatory data and halt the submission

**Status: Met.** Unchanged: a submission or publish stops while mandatory data is missing, and every missing
field is listed at once with a link to it (008 FR-010, 004 FR-024). 017 adds approver problems to the same
list (FR-012, FR-036), and 016 adds a custom budget phasing that does not add up.

**Evidence:** Go: `TestSubmit_Problems400_FR010`, `TestGetSubmission_IncompleteMobilityListsAll_FR010`. End to
end: the submission journey in `core-flow.spec.ts`.

**Defaults for Shell to confirm:** 58 (which built-in fields are needed to publish or submit).

---

## 3. Future-state, RFP and needs-clarification rows

These 25 rows are not MVP. "Before" is the gap analysis; "Now" is after this round.

| Requirement | Disposition | Before | Now | What changed, what is left |
|---|---|---|---|---|
| BUS.AI.010 AI capabilities | Future | Not met | Not met (by design) | No AI in scope. The open REST API and the next-year plan copy are the base |
| BUS.BUD.030 Overspend hard stop or warning; underspend | Future | Partly met | Partly met | Overspending and over-allocation warn in red and never block. No underspend warning, no setting |
| BUS.FIN.020 Full PR/PO automation | Future | Not met | Not met | Technical track, after the GSAP freeze |
| BUS.FIN.040 Monthly accruals per PO | Future | Not met | Not met | GSAP's accruals are now **shown** (015); NorthStar does not calculate them |
| BUS.FIN.050 Review and approve supplier invoices | Future | Not met | Not met | Invoices are now visible, read-only (015); no approve or decline |
| BUS.FIN.060 PO/invoice documents linked to the PR | Future | Not met | Not met | No file attachments |
| BUS.PRF.020 PIR linked to business case / SP&A | Future | Not met | Not met | Feature 013, after Shell answers Q-001 |
| BUS.PRF.030 Central view, scoped by access | Future | Partly met | Partly met (better) | Home and All work are now **scoped by access** (014): a market sees its market, a global role sees all. No KPIs or PIRs yet |
| BUS.RPT.010 Standard dashboards | Future | Partly met | Partly met | New: the Home Financials card, the approvals Monitoring tab and the global budget figures. No executive, regional or KPI dashboards |
| BUS.RPT.050 Calendar views by campaign, owner, period, country | Future | Partly met | Partly met | The calendar now runs over a range of years. Filtering by country needs the attribute-filter screen (2.2) |
| BUS.UX.010 Web/mobile, accessibility, languages, Excel, bulk, dashboards, notifications, search | Future | Partly met | Partly met (better) | Now WCAG 2.2 AA and phone layouts on every area. No offline use, English only, no bulk (012) |
| BUS.WFL.040 Notify owners of changes; customisable | Future | Not met | Not met | Notifications cover approvals, reminders and escalations only |
| BUS.WS.01 Export to ppt/pdf/word/excel; share | Future | Partly met | Partly met | All work exports .xlsx and .csv (the Breakdown export is in the API). No PDF, PowerPoint or Word; no sharing |
| BUS.WS.02 Import a plan skeleton / upload an updated plan | Future | Not met | Not met | Import is feature 012, deferred |
| BUS.ADM.010 Shell configures without the vendor | RFP | Partly met | Met for the Plan module | Self-service now covers roles and permissions (014), approval rules (017), exchange rates and field help (016), and the activity log (011). Releases stay with the delivery team |
| BUS.ADM.050 Migrate Hive9 data | RFP | Not met | Not met | Needs a migration feature |
| BUS.FIN.010 GSAP integration | RFP | Not met (technical) | Not met (technical) | Transactions and daily rates are simulated behind adapters, ready for the real GSAP feeds |
| BUS.INT.010 Standard integrations | RFP | Not applicable | Not applicable | Technical track. Open REST API with an OpenAPI contract |
| BUS.WS.03 Multiple time zones; offices | RFP | Not applicable | Not applicable | Vendor answer. Times show in each person's time zone, stored in UTC |
| BUS.WS.04 Keep 5 years of data | RFP | Partly met | Partly met | Nothing is purged; the activity log is append-only and now has its own page. No Hive9 history (ADM.050) |
| BUS.BUD.010 Ownership, approvals, transfers, locking, freeze, multi-currency, carry-forward | Clarification (duplicate) | Partly met | Partly met (better) | New: plan currencies, global allocations, and an allocated budget locked for the local owner. No transfers, freeze or carry-forward |
| BUS.PLN.010 Global template, standard hierarchy and attributes | Clarification (in MVP) | Partly met | Partly met (better) | Global budgets and roles give a global layer over market plans. No "global template" record |
| BUS.PLN.030 Rolling forecasts, phasing, allocation, scenarios | Clarification (duplicate) | Partly met | Partly met (better) | Budget phasing and allocation are now met. No forecasts or scenarios |
| BUS.PLN.060 In-cell formulas | Clarification (not required) | Not applicable | Not applicable | The client says it is not required |
| BUS.RPT.060 Filters and saved filter sets | Clarification (duplicate) | Met (mostly) | Met (mostly) | New filters: years range, plan type, plan currency, Financials. Attribute filters await their screen (2.2) |

---

## 4. UI/UX revamp

### 4.1 What changed for users

Following the product owner's decisions of 20 October 2026 (D-310–D-314), NorthStar now looks like a Shell
enterprise tool rather than the earlier Apple-style look:

- **Colour:** Shell yellow `#FFC800` with dark text for the one primary action per page or dialog; a
  light-grey page with white cards; one meaning per status colour; red only for danger and errors.
- **Type:** Jost 600/700 for headings and Arial for text (stand-ins for Shell's own font), seven text sizes,
  nothing below 12 px. No Shell logo until Shell consents.
- **One way to build each part:** one page header (breadcrumb, title, meta row, actions with a More menu),
  one panel footer, one table standard (grey header band, 40 px rows, numbers right-aligned under their
  headings, scroll inside the card with an edge shadow), one set-inactive flow, one money format per screen.
- **App shell:** a white top bar with theme switch, bell and account menu; a white sidebar with a yellow bar
  on the current page; a drawer below 960 px.
- **Accessibility:** WCAG 2.2 AA as the target, with axe checks on every area, keyboard-scrollable tables
  and 24 px minimum targets.
- **Phones:** tables become cards or scroll inside their card, so pages do not scroll sideways (checked by
  hand in each area; see 4.2, step 1.1).

### 4.2 Against the implementation plan

| Step of the plan | State | Notes |
|---|---|---|
| 1.1 Test harness (visual baseline suite, layout helpers) | **Not built** | Each area was checked by hand instead: screenshots at 1440 × 900 and 390 × 844, light and dark, with axe (WCAG 2.2 AA). The automated layout helpers (no sideways scroll, one primary, numbers under headers) are open |
| 1.2 Global fixes (the layout bugs that broke key screens) | Done | Letter-by-letter wrapping and pages wider than a phone fixed (X-01, X-02) |
| 1.3 Tokens, dark mode and fonts | Done | Full token set in light and dark, contrast-checked; Jost self-hosted |
| 1.4 Scales and lint guard | **Partly done** | The scales are applied (153 style files moved onto the tokens and the 4 px grid). The stylelint guard that keeps them is not set up |
| 1.5 Shared components | Done | Button, fields, Tabs, Card, PageHeader, panels and dialogs, chips, TreeTable, EmptyState, Toast, InfoTip and others on version 2 |
| 1.6 App shell | Done | Top bar, sidebar, drawer, page width and gutters |
| 1.7 Documents and decisions | Done | Design system v2, decisions D-310–D-314, constitution 1.1.0 |
| 2.1 Home | Done | Dashboard template, equal card pairs, Financials card, the no-role state |
| 2.2 Plans, plan page, plan form | Done | Record template with tabs, plan tree fits 1,440 px, toolbar in two rows on phones |
| 2.3 Campaign and initiative panels | Done | Plan and campaign named as values, timing as a word |
| 2.4 Activities and line items | Done | Activity tabs (Overview, Details, Approval, History), form section cards with an "On this page" rail and needed-field counts, one primary by role |
| 2.5 Approvals | Done | Table standard, cards on phones, Decline then Approve, Monitoring that fits phones |
| 2.6 All work | Done | One toolbar order, table standard, scroll cue, filters in a drawer on phones |
| 2.7 Settings, people, teams, profile | Done | One list toolbar, one set-inactive flow, one panel footer |
| 2.8 Activity log and Recycle bin | Done | List template, restyled History, sign-in card, one layout for Not found and Coming soon |

### 4.3 The 99 audit items

The plan places each of the UI audit's 99 IDs (97 entries plus M-01 and D-03) under one step. Every step
above except 1.1 and the guard of 1.4 is done. About 45 IDs are named explicitly in the change records, among
them X-01–X-04, X-08, X-09, X-13, X-14, X-16, H-01–H-03, H-06, P-06, PL-01, C-02, C-03, AC-01–AC-04, AC-06,
AF-01, AF-02, LI-01–LI-03, AP-01, AP-02, AP-04, AW-01, AW-03–AW-06, ST-04–ST-06, PE-01, PR-02, PH-03, PH-04,
S-01, M-02 and D-01. The others are within the scope of the area lanes.

**Confirmed in the final visual pass: _x of 99_ (to be filled).** Known leftovers: AX-01 and X-10 rely on the
missing lint guard, so the scales are applied but not enforced. One plan tree (FY2026 Mobility China) is
11 px wider than its card at 1,440 px because of an activity row's planned-cost note.

**Functional review findings (25):** the plan addressed 18. The change records confirm FR·P-02, FR·P-04 (in
part), FR·P-10, FR·A-01, FR·A-02, FR·S-01, FR·S-02, FR·X-02, FR·X-03, FR·X-07 and FR·X-08. FR·X-04 (the log and
bin had no demo content) is closed by 011's seed.

---

## 5. Work item 011 — activity log, record history and recycle bin

**Status: Complete and merged** into `plan-module` on 20 October 2026.

- **All 52 tasks are done.** The success criteria SC-003, SC-004 and SC-007 are recorded in the 011
  checklist.
- **What users get:**
  - the **Activity log** page: who changed what and when, with search and filters;
  - a **History** section on every plan, campaign, initiative, activity and line item, opened from View history;
  - the **Recycle bin** with restore (the record returns with its original IDs) and permanent delete;
  - **Recent changes** on Home;
  - demo history (two weeks of log entries and a deleted initiative in the bin).
- **End to end:** a journey follows one activity through create, change, delete from its page, the bin,
  restore, its History and the log narrowed by search and action. All 29 journeys passed twice on fresh demo
  data.
- **Performance fix found by the end-to-end run:** each table is now read once per request. The Home summary
  answers in 0.04 s, and the full end-to-end run takes about 5 minutes instead of timing out.
- **Since the merge:**
  - 014 limits the log, History and bin to the entries a person may see, and checks restore rights on every
    entry that comes back;
  - the pages are restyled to design system v2 (area 2.8);
  - the log links global budgets and GSAP transactions to their pages.
- **What it gives the BRD:** the audit trail of BUS.WFL.010, and the recycle bin that delete messages
  promised (FR·P-02, FR·X-03).

---

## 6. Test results

Run on the merged `plan-module` (`e0b5f45` plus the end-to-end commit), with fresh demo data, on `127.0.0.1`
only, on 21 Oct 2026.

| Check | Command | Result | Run on |
|---|---|---|---|
| Go formatting | `gofmt -l .` (prints nothing) | Pass: no file listed | 21 Oct 2026 |
| Go static checks | `go vet ./...` | Pass: no findings | 21 Oct 2026 |
| Go tests | `go test ./...` (number of packages and tests) | Pass: 33 packages (32 with tests, `api` has none), 1,572 tests and 479 subtests, 0 failures | 21 Oct 2026 |
| Web checks | `npm run check`: API types, type check, lint, format | Pass: API types match `openapi.yaml`, `tsc -b`, `eslint .` and `prettier --check .` clean | 21 Oct 2026 |
| Web unit and component tests | Vitest, part of `npm run check` (number of files and tests) | Pass: 288 files, 2,274 tests | 21 Oct 2026 |
| End to end, run 1 | Playwright, full suite (number of journeys) | Pass: 36 of 36 journeys (29 existing + 7 in `brd-gaps.spec.ts`), 6.6 min | 21 Oct 2026 |
| End to end, run 2 | Playwright, full suite on fresh demo data | Pass: 36 of 36 journeys on a second fresh seed, 5.9 min (run at the same time as run 1, with its own API, web app, ports and data folder) | 21 Oct 2026 |
| Accessibility | axe WCAG 2.2 AA inside the end-to-end journeys | Pass: 26 axe checks in 7 spec files, 0 violations (after the All work fix below) | 21 Oct 2026 |

**What the end-to-end run found and fixed**
- The **Import GSAP rates** page is now routed: `/settings/currencies/import` opens it from the Currencies
  page (016 T028). BUS.BUD.050 has no open item left from this.
- The All work grid's record names were smaller than the 24 px target size that axe checks (WCAG 2.2
  SC 2.5.8); they are now 24 px high.
- 11 of the 29 existing journeys failed on the first run because they asserted wording or layout from
  before the Shell redesign, 016's plan currency or 014's roles: the plan header's facts row and More
  menu, the breadcrumb of a new activity, "Search plans", "My profile", the "1,204 of 1,204 entries" and
  "1 of 1 item" counts, the "Years" filter, the one completeness chip, amounts in the plan's currency with
  their display-currency value, the tree's "67% used" meter, the Home summary card values, and the sidebar
  for a person whose role hides Global budgets (that journey now signs in as Alex Morgan). Each now checks
  the same behaviour in today's wording; no check was dropped.
- New `frontend/e2e/brd-gaps.spec.ts`, 7 journeys of the BRD gaps: an administrator gives a person a
  role and it takes effect (014), a viewer sees a plan read-only (014), a planner sees an activity's GSAP
  financials and the Financials page (015), a small activity is approved automatically (017), a delegate
  approves for the approver who is away (017), a planner splits an activity (016), and a global planner
  allocates part of the global budget to a plan (016).

**Recorded at the final merge (`a1ae8fe`, before the full runs above)**
- `go build ./...`, `go vet ./...` and the web type check (`tsc -b`) pass.
- The `seed`, `cmd/seed` and `ids` Go tests pass, and the 27 approval-workflow and approval-section web tests
  whose merge conflicts were resolved pass.
- Three web tests fail, in `src/app/RequireSignIn.test.tsx` and `src/app/SessionProvider.test.tsx`; they failed
  before the last router fix as well. Go tests that name the demo's deleted initiative "Trade shows" by its
  old IDs (I-1009, A-1011, L-1009) may need updating, because 016's records moved it to I-1011, A-1013 and
  L-1011. The full suites had not been run at that point.

**Evidence recorded on the branches before the merge**
- 011: 29 end-to-end journeys passed twice on fresh demo data.
- 014: the full end-to-end suite passed (29 journeys) with roles applied.
- 016: `gofmt`, `go vet` and `go test ./...` passed on the backend branch. The timing tests of 009 and 010
  can miss their limit on a heavily loaded machine.
- 015 and 017: backend merged into `plan-module` with their tests (62 and 136 test functions).
- Web areas: component tests with mocked API answers; screens checked at 1440 × 900 and 390 × 844, light and
  dark, axe with 0 violations (Home, All work, Settings, Roles and access, Financials, Global budgets,
  Approvals).

---

## 7. Open items and recommendations

### 7.1 Open items

| # | Item | Requirement | Task(s) | Size |
|---|---|---|---|---|
| 1 | Screens for budget categories: attribute filters, Breakdown tab, Planned cost by card | BUS.BUD.020 | 016 T074–T077 | M |
| 2 | Field-help tips on the plan, campaign, initiative, activity and line item screens | BUS.UX.020 | 016 T084–T088 | S–M |
| 3 | Hide or explain actions the person's role does not allow, on every page ("View only" note); the Home Financials card's no-access text | BUS.ADM.020, BUD.040, FIN.080 | 014 T043–T051, 015 T044 | M |
| 4 | Approvals read people's roles from 014 instead of the stand-in that mirrors the demo roles | BUS.ADM.020, WFL.010 | 017 T001, T008 | S–M |
| 5 | ~~Route the Import GSAP rates page (`/settings/currencies/import`)~~ Done in the end-to-end run | BUS.BUD.050 | 016 T028 | S |
| 6 | Line manager field on the People screen | BUS.WFL.010 | 017 T036 | S |
| 7 | Linked-transactions sentence in the campaign, initiative and plan delete dialogs | BUS.FIN.080 | 015 T045 | S |
| 8 | End-to-end journeys of 014–017: 7 written (`brd-gaps.spec.ts`); GSAP import and linking, conditional levels open | all four | 015 T025/T030/T035, 017 T048 | S |
| 9 | Product documents: domain model, field catalogue, glossary, design system §5, build guide (the `tasks.md` files and the feature map are up to date) | — | 014 T063, 015 T048, 016 T090–T093 | S |
| 10 | UI test harness and the stylelint guard | UI plan | UI plan 1.1, 1.4 | S–M |
| 11 | Re-use small follow-ups: copy a single campaign or activity into another year; "structure only" plan copy | BUS.PLN.050, AI.010 example | gap analysis §3.8 | S |
| 12 | Replace the simulations when Shell's systems are available: GSAP transactions, GSAP daily rates, Microsoft Entra ID sign-in, the nightly approvals job | BUD.050, FIN.080, ADM.020, WFL.010 | adapters in place | M each |

### 7.2 Needed from Shell

- Confirm or change the 58 defaults in section 8.
- Send **"the note on financial transactions"** (BUS.FIN.080).
- Confirm which built-in fields are needed to publish or submit (Q-005).
- Say whether Shell has an internal design system for staff tools that NorthStar must use, and whether the
  Shell typeface and pecten may be used.

### 7.3 Recommendations

1. **Send section 8 to Shell now.** The answers change configuration and wording, not the build.
2. **Run a short finishing sprint (about 1–2 weeks) before user acceptance testing** for items 1–8. That
   makes all 11 MVP rows Met with end-to-end proof.
3. **Demonstrate with the demo personas.** Each has a different role and market, so access, global and local
   budgets, and approvals can be shown from each side.
4. **Plan the GSAP and Entra ID connections** with Shell's technical track (BUS.FIN.010). Only the adapters
   change.

---

## 8. Defaults for Shell to confirm

Each default is applied now and can be changed by configuration or a small change. The decision that records
it is in brackets ([decision log](../sdd/decision-log.md)).

**Roles and access (014)**
1. Five roles — Administrator, Global planner, Planner, Approver, Viewer — with the permissions of 014
   Table B. Shell can change them or add roles in Settings (D-223, D-224).
2. A market is a team. A role applies to All markets, My teams or Chosen teams. There is no scoping by
   business unit, cluster or country values yet (D-225).
3. Plans are hidden from people outside their scope. Owners and named approvers still see their own records
   (D-226).
4. Planners cannot delete plans; Global planners and Administrators can (014 Table B).
5. Everyone can read Settings; only Administrators change them. Currencies, exchange rates and rate files are
   a separate permission, also Administrator-only by default (D-240).
6. Every default role may see financial data. Linking transactions is for Administrator, Global planner and
   Planner; importing GSAP extracts is for Administrators only (D-237).
7. Someone without access to a record is told "You don't have access to this …" and sees nothing of it
   (D-227).
8. Refused attempts are recorded only in the technical log, not shown to administrators (D-241).
9. New people get the Viewer role for their own teams (D-234).
10. Roles are assigned in NorthStar while demo sign-in stays. With Microsoft Entra ID they may come from
    Entra app roles or groups later (D-223).

**Global and local budgets (014)**
11. Global budgets are shared top-down. A global planner allocates to market plans; the allocation becomes
    the plan's budget and is read-only for the local owner. A plan has at most one allocation (D-235).
12. Over-allocation shows "Over-allocated by X" and never blocks (D-235).
13. A global budget is in USD. Each allocation is in its plan's currency and counts in USD at the current
    rate (D-303).
14. Local owners never see other markets' allocations; the global budget page adds them up as "Other
    markets" (D-236).

**Approvals (014, 017)**
15. Segregation of duties: nobody approves an activity they own or submitted, in any capacity. On by
    default; an Administrator can turn it off (D-231, D-280, D-302).
16. Only Administrators decide on behalf of named approvers (recorded "on behalf of"). Recall is for the
    owner, the submitter or an Administrator (D-231, D-281).
17. Approval rules look at the planned cost in USD, the plan's business units and the activity type,
    combined with AND, and are checked once, at submission (D-276, D-277).
18. The demo thresholds are examples: approved automatically below 10,000.00 USD; an extra level from
    100,000.00 USD. Shell sets its own on each workflow (D-295).
19. Only planned cost, dates and line-item amounts or currencies restart an approval; a change of activity
    type does not (D-276).
20. "Manager" is a line manager kept in NorthStar, to come from Entra ID later (D-282).
21. "All must approve" means every approver of the level; one decline declines the activity (D-278).
22. A level that names nobody stops the submission. A level whose people are all left out (inactive, owner,
    submitter) goes to its backup approver or waits for an Administrator (D-283).
23. Delegation: one delegate at a time who can approve, for at most 90 days, no chains. The person
    delegating keeps their rights. People who manage approval workflows manage everyone's delegations
    (D-284).
24. A reminder after 3 days and an escalation after 5 days waiting (calendar days), once each per level.
    Escalation goes to the level's backup approver, who can then decide the whole level (D-285).
25. Until NorthStar is hosted, reminders and escalations are sent when someone who manages workflows presses
    **Run reminders and escalations**; then a nightly job does it (D-286).
26. "Overdue" means waited at least the escalation days, or the reminder days when there is no escalation
    (D-287).
27. Notifications stay in NorthStar (the bell). E-mail and Teams come with BUS.WFL.040.
28. Delegation dates and "today" are UTC calendar dates (D-284).

**Financial transactions (015)**
29. Please share "the note on financial transactions". Until then: four types (purchase-order lines,
    invoices with credit notes, payments, accruals), the fields of 015 FR-001 and the placements in 2.5
    (D-242, D-243, D-250, D-305).
30. Transactions are read-only and come only from GSAP, simulated now by a CSV import and demo data. Nothing
    is sent to GSAP (D-242).
31. Purchase requests are not shown (D-243).
32. "Paid" means Shell's actuals: payments released (D-243).
33. "Accrued" is GSAP's accruals of the latest accrual month in the data. NorthStar calculates no accruals
    (D-244).
34. Committed (purchase orders) is compared with planned cost ("% of planned cost committed"). "Not yet
    committed" = planned cost − committed. Committed above planned cost shows in red and never blocks.
35. Matching: the HX key is "HX" + the NorthStar line-item number. Invoices, payments and accruals link
    through their PO number, then by SWBS code. Unmatched rows are listed as "Not linked" with the reason
    (D-245).
36. Links made from GSAP data cannot be changed in NorthStar. Unmatched transactions can be linked by hand
    by people allowed to link (D-253).
37. Transactions convert to USD at the current rate, like every other amount, not at their posting-date
    rate (D-248, D-303).
38. The simulated GSAP extract is a UTF-8 CSV of at most 700 KB and 5,000 rows with fixed columns. Refused
    rows are skipped and the rest imported. Importing a row again updates it and never deletes (D-246,
    D-247).
39. The Financials page's Year filter is the linked plan's year; not-linked transactions use their posting
    year (D-256).
40. All work shows Committed and Invoiced by default (Paid, Accrued and Not yet committed are optional
    columns). The plan tree has no financial columns (D-250).
41. Deleting a line item with transactions stays allowed, with a warning. Its transactions leave the totals
    until it is restored. Copies never copy transactions (D-249).

**Currencies and exchange rates (016)**
42. Every conversion uses the latest daily rate on or before today; there is no separate annual planning
    rate. Approved activities keep their plan-currency amounts; their USD values follow the rates (D-258).
43. The simulated GSAP daily-rate file is a CSV with `rate_type`, `from_currency`, `to_currency`,
    `valid_from` and `rate`. Only rate type M quoted from USD is used, one rate per currency per day; a
    different rate for a day already present replaces it (D-259).
44. Days without a rate (weekends, holidays) use the latest earlier rate (D-258).
45. One plan currency per plan, USD by default, chosen when the plan is created and locked once amounts are
    planned in it or the plan has an allocation. Line items may use any currency (D-260, D-264).
46. Inside a plan, amounts show in the plan currency with the reader's display currency beside. Pages
    covering several plans use the display currency (D-261).
47. Goals (deal size, revenue) stay in USD. An activity type's default planned cost stays in USD and is
    converted into the plan currency (D-260).
48. Campaigns cannot be moved or copied between plans with different plan currencies (D-264).

**Planning (016)**
49. Budget phasing is monthly over the plan's year (Not phased, Evenly, Custom; a quarter typed is spread
    evenly over its months). A custom phasing that does not add up stops publishing (D-265).
50. Planned cost compared with the phased budget uses the line items' spreads and spreads a positive
    unassigned balance evenly, as All work's Budget & cost does (D-266).
51. "Split an activity from its original budget" means dividing one activity into two within the same plan
    (the new one may sit under another initiative). Whole line items move, goals stay on the original,
    there is no undo, and the activity log keeps the record (D-267).
52. Plan type is a fixed list (Operational, Strategic, Cross-market) beside the Strategic/AOP flag. A
    strategic plan is still one year and does not link annual plans (D-268).
53. Multi-year planning means views across plans: a year range on Plans and up to 5 years in All work.
    Plans stay single-year (D-269).
54. Budgets are reported by category (attribute filters, Breakdown, Planned cost by), not set per category.
    Multiple-choice values count in each group and percentage splits divide the amount (D-270).
55. The planning level names stay fixed. Shell's hierarchies are modelled with dependent Shell attributes
    and analysed with "Break down by … then by …".

**Field help (016)**
56. Field help is plain text of up to 300 characters. NorthStar writes the defaults; an empty text hides
    the tip; Shell attributes keep their help on each attribute's page (D-271).

**General**
57. GSAP (transactions and daily rates), single sign-on and the nightly approvals job stay simulated until
    Shell's systems are available. Each sits behind an adapter (D-242, D-259, D-286, D-308).
58. Which built-in fields are needed to publish or submit (Q-005). NorthStar keeps the markings of the
    field catalogue until Shell confirms.

---

## Sources

- Requirements: [BRD transcription](../reference/brd-requirements.md). Starting point:
  [gap analysis](brd-gap-analysis.md). Closure design and defaults: [gap-closure plan](brd-gap-closure-plan.md).
- Specs, plans and tasks: `specs/011-activity-log-and-recycle-bin/`, `specs/014-access-control/`,
  `specs/015-financial-transactions/`, `specs/016-planning-and-currency-extensions/`,
  `specs/017-approval-rules/`.
- UI: [UX/UI recommendations](ux-design-recommendations.md), [UI/UX implementation plan](ui-ux-implementation-plan.md),
  [UI audit](ui-audit.md), [functional review](functional-review.md), [design system v2](../product/design-system.md).
- What was built: the change history of `plan-module` and of the lane branches `lane-b014c`, `lane-b014g`,
  `lane-b015`, `lane-b016`, `lane-b017`, `lane-f1`–`lane-f6`, `lane-011b` and `lane-ui`.
