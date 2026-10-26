# BRD gap-closure plan: closing Shell's MVP requirements with features 014–017

| | |
|---|---|
| **Date** | 2026-10-20 |
| **Readers** | Product owner, the Shell client team (§3), build leads and agents (§2, §4) |
| **Inputs** | [BRD transcription](../reference/brd-requirements.md) · [gap analysis](brd-gap-analysis.md) · specs [014](../../specs/014-access-control/spec.md), [015](../../specs/015-financial-transactions/spec.md), [016](../../specs/016-planning-and-currency-extensions/spec.md), [017](../../specs/017-approval-rules/spec.md) · [decision log](../sdd/decision-log.md) D-223–D-309 · [UI/UX implementation plan](ui-ux-implementation-plan.md) |
| **Product owner decision** | 2026-10-20: build every MVP gap now, with documented, reversible defaults for Shell to confirm. GSAP feeds and single sign-on stay simulated (an import endpoint plus demo data behind an adapter; demo sign-in) |
| **State** | Specified and consolidated. The contract is merged (`backend/api/openapi.yaml`, D-298); nothing of 014–017 is built yet |

## 1. Summary

- **Every MVP requirement now has a design that closes it.** Two were already met (BUS.WFL.030, BUS.PLN.050).
  The other nine are closed by four new features, plus the 011 merge for the audit trail:

  | Feature | Closes | New API operations | Tasks |
  |---|---|---|---|
  | 014 Access control and global budgets | BUS.ADM.020, BUS.BUD.040; the role part of BUS.WFL.010 | 22 | 64 |
  | 015 Financial transactions | BUS.FIN.080 | 10 | 50 |
  | 016 Planning and currency extensions | BUS.BUD.050, BUS.PLN.020, BUS.PLN.040, BUS.BUD.020, BUS.UX.020 | 21 | 95 |
  | 017 Approval rules | BUS.WFL.010 | 7 | 50 |

- **The four designs agree with each other and with 001–011.** The consolidation fixed where they did not
  (D-298–D-309): one refusal code (403 `forbidden`), one permission list (014 Table A) used by 015–017, one
  segregation-of-duties switch, one currency rule across global budgets, plan currencies, transactions and approval
  conditions, one order of sections on the shared pages, one demo-data order and one build order.
- **The contract is merged and checked.** 60 new operations; every operation except the four public ones answers
  403; `go vet` and `go test ./...` pass. Five type errors in four web files are the expected effect of new enum
  values and filters; they are fixed first, in wave 0 (§4.1, step 0.3).
- **Shell has 58 defaults to confirm (§3).** The most important: the five roles, hiding plans outside a person's
  markets, the meaning of global and local budgets, the daily-rate policy, the meaning of "split", and the approval
  rules. Shell still owes "the note on financial transactions" (BUS.FIN.080).
- **Build in four waves (§4):** preparation; foundations (the UI foundation beside the backend foundations of
  014, 016 and 015); backend stories plus frontend lanes per area; integration and validation.

## 2. MVP requirements and what closes them

"Before" is the gap analysis's status. Task IDs are those of each feature's `tasks.md`.

| BRD | Before | Closed by | Tasks | After the build |
|---|---|---|---|---|
| **BUS.ADM.020** Role- and team-based access; view-only vs edit | Not met | **014** FR-001–FR-028, FR-037–FR-041: 25 fixed permissions, five roles that Shell edits in Settings, scopes by team (a market is a team), every operation with a declared rule (fail closed), `access` objects so screens hide or explain, View only note, Settings › Roles and access, Your access on My profile. **015** FR-028 and **016** FR-049 use 014's permissions (D-237, D-301) | 014 T002–T032, T038–T056, T060–T062 · 015 T010, T043, T044 · route rows of 015–017 in each serving task (D-301) | Met: each person sees and changes only what their roles allow, per market; administrators change roles without code |
| **BUS.BUD.020** Budget categories; client-updatable attributes; flexible framework | Partly met | **016** FR-039–FR-044: Shell-attribute filters (with "Not filled in"), Breakdown by one or two attributes with totals, "Planned cost by" card on the plan page; hierarchies through dependent attributes | 016 T070–T077 | Met for reporting by category; setting budgets per category is not offered (default 54, Q-017) |
| **BUS.BUD.040** Global/local allocation with visibility for each owner | Not met | **014** FR-004–FR-007, FR-029–FR-036, FR-040: global budgets, top-down allocations that set the plan budget (read-only locally), Other markets row, plan allocation note; currency rule D-303 | 014 T033–T037, T057–T059, T061 (journey 5) | Met |
| **BUS.BUD.050** Currency conversion, rate management, GSAP daily rates; plan in local currency and USD | Partly met | **016** FR-001–FR-019: dated rates with history, simulated GSAP daily-rate import behind `settings.RateFeed`, plan currency for budgets, phasing and planned costs with the display currency beside, rates shown in tips and exports. **015** FR-010 converts transactions (current rate, D-303) | 016 T004–T043 | Met; the live GSAP rate feed replaces the file import later |
| **BUS.FIN.080** POs, invoices, payments, accruals prominent in the main views | Not met | **015** FR-001–FR-031: read-only GSAP transactions from a simulated extract behind `financials.Feed`, matching to line items, figures at every level, open Financials sections on the activity, line item, plan, campaign and initiative, All work columns and filter, Home card, Financials page; who may see, link and import from **014** FR-015 | 015 T001–T050 | Met with defaults; placements may change once Shell sends the note on financial transactions (Q-015) |
| **BUS.PLN.020** Multi-year, annual, quarterly, monthly; strategic, operational, cross-market | Partly met | **016** FR-021, FR-024, FR-032–FR-038: plan type (Operational, Strategic, Cross-market), year ranges on Plans and up to 5 years in All work (Yearly and Quarterly), budget by month and quarter | 016 T044–T053, T061–T069 | Met as views across single-year plans (default 53, Q-016) |
| **BUS.PLN.040** Split the budget by month; split an activity from its original budget | Partly met | **016** FR-020–FR-031: budget phasing for plans, campaigns and initiatives (Not phased, Evenly, Custom), phased vs planned card and comparison, Split activity | 016 T044–T060 | Met (split as defined in default 51, Q-013) |
| **BUS.PLN.050** Re-use prior-year planning content | Met (small gaps) | Nothing new needed for "met". 014 FR-035 (a copy never takes an allocation), 015 FR-030 (copies never copy transactions) and 016 FR-025 (copies keep currency, type and phasing; a next-year copy shifts the phasing) keep copying correct | — | Met. The two small improvements of the gap analysis (cross-year single copies, a "structure only" copy) are **not** in 014–017; they are a small follow-up (§5) |
| **BUS.UX.020** Contextual help maintained by administrators | Partly met | **016** FR-045–FR-048: Settings › Field help for every built-in field and calculated value, ⓘ tips on every form, detail and summary card, "No help yet" on Shell attributes | 016 T078–T088 | Met |
| **BUS.WFL.010** Multi-level, parallel, conditional approvals, delegation, notifications, escalation, roles, SoD, audit, monitoring | Partly met | **017** FR-001–FR-037: approver sources (named, line manager, role), any or all, "only needed when", "approve automatically when", backup approvers, delegation, reminders and escalation, Monitoring, new notifications and history. **014** FR-025–FR-028: who may decide, segregation of duties, workflow approvers must be able to approve. **011** (merge in progress) gives the audit-trail pages | 017 T001–T050 · 014 T031–T032, T056 | Met |
| **BUS.WFL.030** Highlight missing mandatory data and halt submission | Met | Already met by 008 FR-010 and 004 FR-024 (D-297); 017 FR-012 and FR-036 add the problems of approver sources | 017 T010, T013 | Met; Shell confirms the field markings (Q-005, default 58) |

Cross-cutting pieces that every row above relies on: the merged contract (D-298, D-299, D-300), the Table C
route rules (014 plan and its Consolidation section, D-301), the money rule (D-303), the log visibility rule
(D-238, D-304), the screen placement (D-305) and the demo data order (D-306).

## 3. Defaults for Shell to confirm

Each default is applied now and can be changed by configuration or a small change. The decision that records it
is in brackets; the question of the gap analysis it answers is named where there is one.

**Roles and access (014; Q-010, Q-006)**

1. Five roles — Administrator, Global planner, Planner, Approver, Viewer — with the permissions of 014 Table B.
   Shell can change them, or add roles, in Settings at any time (D-223, D-224).
2. A market is a team. A role applies to All markets, My teams or Chosen teams; there is no scoping by Business
   unit, Cluster or Country/OU values yet (D-225).
3. Plans are hidden from people outside their scope; owners and named approvers still see their own records
   (D-226, answers Q-006).
4. Planners cannot delete plans; Global planners and Administrators can (014 Table B).
5. Settings are readable by everyone and changed only by Administrators; currencies, exchange rates and rate files
   are a separate permission, also Administrator-only by default (D-240).
6. Every default role may see financial data; linking transactions is for Administrator, Global planner and
   Planner; importing GSAP extracts is for Administrators only (D-237).
7. Someone without access to a record is told "You don't have access to this …" and nothing of the record is
   shown (D-227).
8. Refused attempts are recorded only in the technical log, not shown to administrators (D-241).
9. New people get the Viewer role for their own teams (D-234).
10. Roles are assigned in MAPP while demo sign-in stays; with Microsoft Entra ID they may come from Entra app
    roles or groups later (D-223).

**Global and local budgets (014; Q-011)**

11. Global budgets are shared top-down: a global planner allocates to market plans; the allocation becomes the
    plan's budget and is read-only for the local owner; a plan has at most one allocation (D-235).
12. Over-allocation shows "Over-allocated by X" and never blocks (D-235).
13. A global budget is in USD; each allocation is in its plan's currency and counts in USD at the current rate
    (D-303).
14. Local owners never see other markets' allocations; the global budget page adds them up as "Other markets"
    (D-236).

**Approvals (014, 017; Q-014, Q-002)**

15. Segregation of duties: nobody approves an activity they own or submitted, in any capacity. On by default; an
    Administrator can turn it off (D-231, D-280, D-302).
16. Only Administrators decide on behalf of named approvers (recorded "on behalf of"); recall is for the owner,
    the submitter or an Administrator (D-231, D-281).
17. Approval rules look at the planned cost in USD, the plan's business units and the activity type, combined with
    AND, and are checked once, at submission (D-276, D-277).
18. The demo thresholds are examples: approved automatically below 10,000.00 USD; an extra level from
    100,000.00 USD. Shell sets its own on each workflow (D-295).
19. Only planned cost, dates and line-item amounts or currencies restart an approval; a change of activity type
    does not (D-276).
20. "Manager" is a line manager kept in MAPP (Settings › People), to come from Entra ID later (D-282).
21. "All must approve" means every approver of the level; one decline declines the activity (D-278).
22. A level that names nobody (no line manager, nobody with the role) stops the submission; a level whose people
    are all left out (inactive, owner, submitter) goes to its backup approver or waits for an Administrator
    (D-283).
23. Delegation: one delegate at a time who can approve, for at most 90 days, no chains; the person delegating keeps
    their rights; people who manage approval workflows manage everyone's delegations (D-284).
24. A reminder after 3 days and an escalation after 5 days waiting (calendar days), once each per level;
    escalation goes to the level's backup approver, who can then decide the whole level (D-285).
25. Until MAPP is hosted, reminders and escalations are sent when someone who manages workflows presses **Run
    reminders and escalations**; then a nightly job does it (D-286).
26. "Overdue" means waited at least the escalation days, or the reminder days when there is no escalation (D-287).
27. Notifications stay in MAPP (the bell); e-mail and Teams come with BUS.WFL.040 (017 Assumptions).
28. Delegation dates and "today" are UTC calendar dates (D-284).

**Financial transactions (015; Q-015)**

29. Please share "the note on financial transactions" that BUS.FIN.080 refers to. Until then: four types
    (purchase-order lines, invoices with credit notes, payments, accruals), the fields of 015 FR-001 and the
    placements of D-250 and D-305 (D-242, D-243).
30. Transactions are read-only and come only from GSAP — simulated now by a CSV file import and demo data; nothing
    is sent to GSAP (D-242).
31. Purchase requests are not shown (D-243).
32. "Paid" means Shell's actuals: payments released (D-243).
33. "Accrued" is GSAP's accruals of the latest accrual month in the data; MAPP calculates no accruals
    (D-244).
34. Committed (purchase orders) is the figure compared with planned cost ("% of planned cost committed"); "Not yet
    committed" = planned cost − committed; committed above planned cost is shown in red and never blocks (015
    FR-011, FR-013).
35. Matching: the HX key is "HX" + the MAPP line-item number; invoices, payments and accruals link through
    their PO number, then by SWBS code; unmatched rows are listed as "Not linked" with the reason (D-245).
36. Links made from GSAP data cannot be changed in MAPP; unmatched transactions can be linked by hand by
    people allowed to link (D-253).
37. Transactions are converted to USD at the current rate, like every other amount, not at their posting-date rate
    (D-248, D-303).
38. The simulated GSAP extract is a UTF-8 CSV of at most 700 KB and 5,000 rows with fixed columns; refused rows are
    skipped and the rest imported; importing a row again updates it and never deletes (D-246, D-247).
39. The Financials page's Year filter is the linked plan's year; not-linked transactions use their posting year
    (D-256).
40. All work shows Committed and Invoiced by default (Paid, Accrued and Not yet committed are optional columns);
    the plan tree gets no financial columns (D-250).
41. Deleting a line item with transactions stays allowed, with a warning; its transactions leave the totals until
    it is restored; copies never copy transactions (D-249).

**Currencies and exchange rates (016; Q-012)**

42. Every conversion uses the latest daily rate on or before today; there is no separate annual planning rate.
    Approved activities keep their plan-currency amounts; their USD values follow the rates (D-258).
43. The simulated GSAP daily-rate extract is a CSV with `rate_type`, `from_currency`, `to_currency`, `valid_from`
    and `rate`; only rate type M quoted from USD is used; one rate per currency per day; a different rate for a day
    already present replaces it (D-259).
44. Days without a rate (weekends, holidays) use the latest earlier rate (D-258).
45. One plan currency per plan, USD by default, chosen when the plan is created and locked once amounts are
    planned in it or the plan has an allocation; line items may use any currency (D-260, D-264).
46. Inside a plan, amounts show in the plan currency with the reader's display currency beside; pages covering
    several plans use the display currency (D-261).
47. Goals (deal size, revenue) stay in USD; an activity type's default planned cost stays in USD and is converted
    into the plan currency (D-260).
48. Campaigns cannot be moved or copied between plans with different plan currencies (D-264).

**Planning (016; Q-013, Q-016, Q-017)**

49. Budget phasing is monthly over the plan's year (Not phased, Evenly, Custom; a quarter typed is spread evenly
    over its months); a custom phasing that does not add up stops publishing (D-265).
50. Planned cost compared with the phased budget uses the line items' spreads and spreads a positive unassigned
    balance evenly, as All work's Budget & cost does (D-266).
51. "Split an activity from its original budget" means dividing one activity into two within the same plan (the
    new one may sit under another initiative); whole line items move; goals stay on the original; there is no
    undo and the activity log keeps the record (D-267, answers Q-013).
52. Plan type is a fixed list (Operational, Strategic, Cross-market) beside the Strategic/AOP flag; a strategic
    plan is still one year and does not link annual plans (D-268).
53. Multi-year planning means views across plans: a year range on Plans and up to 5 years in All work; plans stay
    single-year (D-269, answers Q-016).
54. Budgets are reported by category (attribute filters, Breakdown, Planned cost by), not set per category;
    multiple-choice values count in each group and % splits divide the amount (D-270, answers Q-017 and Q-009).
55. The planning level names stay fixed; Shell's hierarchies are modelled with dependent Shell attributes and
    analysed with "Break down by … then by …" (016 Assumptions).

**Field help (016)**

56. Field help is plain text of up to 300 characters; MAPP writes the defaults; an empty text hides the tip;
    Shell attributes keep their help on each attribute's page (D-271).

**General**

57. GSAP (transactions and daily rates), single sign-on and the nightly approvals job stay simulated until Shell's
    systems are available. Each sits behind an adapter, so the real feed replaces the simulation without changing
    screens or rules (D-242, D-259, D-286, D-308).
58. Still open from before: which built-in fields are "needed to publish/submit" (Q-005); MAPP keeps the
    markings of the field catalogue until Shell confirms.

## 4. Build order

The order follows D-307: access rules block every other lane, the UI foundation comes first on the web side,
rates come before figures in local currencies, and shared Go packages change in a fixed order.

### 4.1 Waves

| Wave | Work | Done when |
|---|---|---|
| **0 · Preparation** (≈ 2 days) | **0.1** Merge 011 (branch `lane-011b`, which adds D-222). Expected conflicts are trivial: keep D-222 before D-223 in the decision log; keep both edits of `notYetServed`; keep 011's History tag text. **0.2** Constitution 1.1.0, ADR-0008, ADR-0009 and CLAUDE.md (D-308); fold in the UI plan's Principle III amendment (its decision 10). **0.3** Make `npm run check` green after the contract merge with five small edits named in the Consolidation sections: `api/errors.ts` (`forbidden`, 014), `all-work/viewState.ts` (`FILTER_LABELS` for `attributes`, `planTypes`, `planCurrencies`, 016), `approvals/messages.ts` (`historyActionText` and `chainStateText` for `autoApproved`, `skipped`, `reminded`, `escalated`, 017), `notifications/NotificationList.tsx` (`KIND_ICONS` for `autoApproved`, `reminder`, `escalated`, 017). **0.4** Send §3 to Shell | 011 on `plan-module`; amendment merged; `npm run check` and `go test ./...` green |
| **1 · Foundations** (≈ 2–3 weeks, 4 lanes) | **UI**: UI plan phase 1 (harness, global fixes, tokens and dark mode, scales and lint guard, shared components, app shell). **B1** (merges first): 014 T002–T011 and **T062** (the existing journeys sign in as the persona whose seeded role allows each step). **B2**: 016 T004–T013 (money conversion, phasing, dated-rate tables and migration, stored fields, seed). **B3**: 015 T002–T011 (the `financials` package; its `Access` adapter over `authz` once B1 is merged) | B1 merged with the e2e suite green; B2 and B3 rebased on B1; UI phase 1 accepted |
| **2 · Stories** (≈ 4–5 weeks) | Backend lanes in package order (§4.2) and frontend lanes per area (§4.3). Each frontend lane applies UI plan phase 2 to its pages and the 014–017 web tasks of the same pages in one pass | Every story's tests green; each served operation removed from `notYetServed` and given its route rule |
| **3 · Integration and validation** (≈ 1–2 weeks) | End-to-end journeys: 014 T061, 015 T021, T025, T030, T035, 016 T089, 017 T047–T049, and the existing journeys. Documents: 014 T063 (all but the constitution, done in 0.2), 015 T048, 016 T090–T093, 017's documentation (domain model §2.3, §6, §8; demo users and data). Checks: 014 T064, 015 T049–T050, 016 T094–T095, 017 T046, T050. The final validation report | `notYetServed` lists only 012's operations; every check green; the final validation report written |

### 4.2 Backend lanes (wave 2)

| Step | Lane | Work | Tasks | Starts after |
|---|---|---|---|---|
| 2a | B1 | 014 US1–US2: visibility filters and `access` objects in plans, campaigns, activities, line items, approvals, Home, All work, history | 014 T012–T025 | wave 1 |
| 2a | B2 | 016 lane R (rates and rate import) and lane H (field help), package `settings` | 016 T018–T025, T078–T081 | wave 1 |
| 2a | B3 | 015 US1–US4, US6, US7 and the access seam | 015 T015–T016, T022, T026, T031–T033, T039, T041, T043, T046, T047 | wave 1 |
| 2b | B1 | 014 US3–US7: roles administration, rules, approvals, global budgets, audit | 014 T026–T038 | 2a B1 |
| 2b | B2 | 016 lane C (plan currency), then lane P (phasing, split) | 016 T029–T038, T044–T049, T054–T057 | 2a B1 |
| 2b | B4 | 016 lane A (year range, attribute filters, Breakdown) | 016 T061–T064, T070–T073 | 2a B1 |
| 2c | B5 | 017 backend | 017 T001, T004–T030 | 014 T031–T032 merged |
| 2c | B3 | 015 US5: financial columns and filter in All work | 015 T036–T037 | 2b B4 |

### 4.3 Frontend lanes (wave 2, after UI phase 1)

| Lane | Area (UI plan step) | 014–017 web tasks |
|---|---|---|
| F0 (first, ≈ 3 days) | Shared parts every lane uses | 014 T039–T042 (`useAccess`, `ViewOnlyNote`, `NoAccess`, `RequirePermission`, navigation, 403 handling); 016 T014–T017 (`RateTip`, `AmountPair`, `YearRangeField`, `useFieldHelp`); 015 T012–T014 (`FinancialFigures`, `FinancialsCard`); 017 T031 |
| F1 | Settings, people, teams, profile (2.7) | 014 T048, T049, T052–T055, T060; 016 T026–T028, T082, T083; 017 T032–T034, T036 |
| F2 | Plans, plan page and form, global budgets; campaign and initiative panels (2.2, 2.3) | 014 T043 (plan pages), T044, T045, T057–T059; 016 T039–T041, T050–T053, T065, T076, T084, T085, T088; 015 T023, T024 |
| F3 | Activities and line items (2.4) | 014 T043 (activity and line-item pages), T046, T047; 016 T042, T043, T058–T060, T086, T087; 015 T017–T020, T045 |
| F4 | Approvals (2.5) | 014 T056; 017 T035, T037–T045 |
| F5 | Home and All work (2.1, 2.6) | 014 T043 (Home no-role state), T050; 016 T066–T069, T074, T075, T077; 015 T038, T040 |
| F6 | Financials page (new) | 015 T027–T029, T034, T042, T044 |
| F7 | Activity log and Recycle bin (2.8) | 014 T043 (log pages), T051 |

Each frontend lane uses `mockApi` until its backend lane is merged, then runs against the real API.

### 4.4 Order rules for shared code (D-307)

- `internal/allwork` and `internal/app/allwork_source.go`: 014 (visibility filter) → 016 lane A → 015 US5.
- `plans`, `campaigns`, `activities`, `lineitems`: 014 → 016 lanes C and P.
- `approvals`: 014 US5 → 017. `people`: 014 → 017.
- Shared files everyone touches (`app/app.go`, `seed/*.go`, `access/routes.go`, `router.tsx`, `navigation.ts`,
  `components/index.ts`): add lines, never rewrite others'. A lane that merges second rebases and keeps both.

### 4.5 Definition of done for every lane

The build guide's checks (`gofmt -l .`, `go vet ./...`, `go test ./...`, `npm run check`), plus: every operation
the lane serves is removed from `notYetServed` and has its route rule (014's `app/access_routes_test.go` passes);
answers validate against the merged contract without contract edits (D-192); the lane's tasks are ticked; the UI
plan's definition of done (§0) holds for the pages the lane touched.

## 5. Not in this round, and open risks

| Item | Effect | Handling |
|---|---|---|
| BUS.PLN.050's small gaps (cross-year single copies, a "structure only" plan copy) are not specified | The requirement stays "Met"; the client's AI.010 example (a skeleton for next year) is not shown yet | A small follow-up change to 004–006 (gap analysis §3.8, size S) after wave 2 |
| The note on financial transactions (Q-015) is missing | 015's placements and fields may change | Defaults 29–41; placements are shared components, cheap to move |
| `npm run check` is red until step 0.3 | Any lane branching before it starts red | Step 0.3 is the first commit of the build |
| The 011 merge and this consolidation both append to the decision log and to `notYetServed` | Merge conflicts | Trivial; resolution in step 0.1 |
| 014 touches every package and every existing test (roles via `apitest.GrantRole`) | Schedule risk for the whole build | B1 builds the foundation alone and merges first with T062; other backend lanes rebase on it |
| Area lanes mix tasks of four features with UI phase 2 | Tasks could be missed or done twice | §4.3 assigns every web task to one lane; each feature's `tasks.md` stays the checklist |
| 016 changes All work period keys ("Q1" → "Q1-2026") and adds `yearFrom`/`yearTo` | Existing All work web code and saved views | 016 lane A migrates the view state (016 data-model §5.2); `year` stays in the contract (D-298) |
| One full `go test ./...` run saw a failure in `internal/home` that passed when re-run alone and in the next full run | A timing-sensitive test under load | Watch it in CI; not caused by the contract merge |

## 6. Sources

- Requirements: [../reference/brd-requirements.md](../reference/brd-requirements.md); gap analysis:
  [brd-gap-analysis.md](brd-gap-analysis.md) §1, §3, §5–§7.
- Specs, plans, data models, contracts and tasks: `specs/014-access-control/`, `specs/015-financial-transactions/`,
  `specs/016-planning-and-currency-extensions/`, `specs/017-approval-rules/` — each plan ends with a binding
  *Consolidation (2026-10-20)* section.
- Decisions D-223–D-309 and questions Q-010–Q-017: [../sdd/decision-log.md](../sdd/decision-log.md); feature map:
  [../sdd/feature-map.md](../sdd/feature-map.md).
- Contract: `backend/api/openapi.yaml` (version 0.3.0); not-yet-served list: `backend/internal/app/app_test.go`.
- UI work: [ui-ux-implementation-plan.md](ui-ux-implementation-plan.md).
