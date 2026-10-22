# Feature Specification: BCPIR and SP&A Plan Views

**Feature Branch**: `013-bcpir-and-spa`

**Created**: 2026-10-18

**Status**: Draft

**Input**: User description: "Shell's BCPIR and SP&A plan views (meaning to be clarified with Shell)" (P3, depends on 004–008).

**References**: Domain model [§4.1 budget flows down, §4.2 cost flows up, §4.3 line items, §4.4 spreads, §4.5 goals, §4.6 values from finance later](../../docs/product/domain-model.md) · Field catalogue [§1 #8 plan budget, #11 BCPIR plan; §2 #5 campaign budget; §4.1 #4 activity type, #8 planned cost; §4.2 #18 PIR; §4.5 goals; §5 #6–8 line-item cost, spread, spend type](../../docs/product/field-catalogue.md) · Design system [§2 summary cards, §4 money and progress bars, §6.6 read-only values](../../docs/product/design-system.md) · Glossary *BCPIR*, *SP&A*, *PIR* · Decisions **Q-001 (open)**, D-008, D-010, D-011, D-015 · Project plan roadmap ("Plan module – BCPIR & SP&A") · FA §B3 (plan form, "BCPIR Plan?"), §E3.1.2 (activity categories BCPIR_Lubes, PIR_Retail …), §F4.4 (Hive9 "SP&A Lubes" and "SP&A Mobility" dashboards)

> **Planning gate:** this baseline is written before Shell has answered **Q-001** (decision log). It
> MUST NOT go to `/speckit-plan` until Q-001 is answered and the two open questions in FR-002 and FR-003
> are resolved with `/speckit-clarify`. The stories below follow the default answers in Assumptions.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See where a plan's money is planned to go (SP&A view) (Priority: P1)

Sara, a finance advisor, opens "FY2026 Lubricants Europe" and chooses the **SP&A (spend)** view: summary
cards for budget, planned cost, budget used and left to plan; per campaign the same figures with a bar;
a quarter table showing when money is planned to go out; and planned cost by activity type and by spend
type. Nothing can be edited here; every figure says where it comes from.

**Why this priority**: Shell's roadmap names SP&A in the Plan module, and finance advisors need the spend
picture of a plan without building it in a spreadsheet.

**Independent Test**: Open the SP&A view of each seeded plan and check every figure by hand against its
campaigns, activities and line-item spreads.

**Acceptance Scenarios**:

1. **Given** a plan with two campaigns, **When** the SP&A view opens, **Then** each campaign shows budget,
   planned cost, budget used and left to plan, and the totals row equals the plan's planned cost (§4.2).
2. **Given** a campaign without a budget, **Then** budget, budget used and left to plan show "—" with the
   tip "No campaign budget set", never 0.
3. **Given** a campaign whose planned cost exceeds its budget, **Then** its bar is red above 100 % and
   left to plan reads "Over budget by $12,000.00" in red.
4. **Given** a 12,000 USD line item spread Evenly over January–June, **Then** Q1 and Q2 each include
   6,000.00 for its campaign.
5. **Given** an activity with planned cost 10,000 whose line items total 7,000, **Then** "Not yet spread"
   includes 3,000.00 and the campaign's quarters plus not yet spread equal its planned cost.
6. **Given** line items without a spend type, **Then** the spend-type table shows them as "Spend type not
   filled in", and the table's total equals the plan's planned cost.
7. **Given** the filter "Approved activities only", **Then** every planned-cost figure counts only
   Approved, In progress and Complete activities, budgets are unchanged, and the filter in use is shown.

---

### User Story 2 - Review the business case of a BCPIR plan (Priority: P2)

Daniel opens a plan marked **BCPIR plan** and chooses the **BCPIR (business case)** view: planned cost,
planned revenue, expected return and PIR counts, then per campaign and activity the planned cost, INQ,
MQL, revenue, expected return and PIR answer. A section shows where actual results will appear later.

**Why this priority**: The BCPIR flag exists on Shell's plan form; this gives it a visible purpose, but
its exact meaning is still open (Q-001).

**Independent Test**: Mark a seeded plan as BCPIR, open the view and check expected return and PIR counts
by hand.

**Acceptance Scenarios**:

1. **Given** a campaign whose activities have planned cost 100,000 and revenue goals 420,000, **Then** its
   expected return reads "4.20×".
2. **Given** an activity without an INQ goal, **Then** its revenue and expected return show "—", and its
   campaign says "1 activity has no revenue goal" and counts that activity's revenue as 0.
3. **Given** an activity whose PIR is not filled in, **Then** it shows "Not filled in" (word and colour)
   and the summary counts it separately from Yes and No.
4. **Given** a plan not marked BCPIR, **When** the user opens the BCPIR view, **Then** it explains "This
   plan isn't a BCPIR plan. Mark it as a BCPIR plan in the plan's details to see its business case." with
   a link to edit the plan.
5. **Given** the BCPIR view, **Then** actual cost, actual revenue, actual return and review result show
   "—" with "Actual return and post-investment review results come with Measure and Finance later".
6. **Given** a plan goal of revenue 2,500,000 and activity goals adding up to 1,900,000, **Then** the
   view shows "Plan goal 2,500,000 USD · activities add up to 1,900,000 USD (76 %)".

---

### User Story 3 - Export a view (Priority: P3)

Sara exports the SP&A view to share with her finance team; Daniel exports the business case.

**Why this priority**: Finance and leadership review plans outside MAPP; export removes retyping.

**Independent Test**: Export both views of a seeded plan and compare every cell with the screen.

**Acceptance Scenarios**:

1. **Given** either view, **When** the user chooses **Export**, **Then** a spreadsheet or CSV file holds
   the same tables and figures, plus the plan name and ID, the view, the filter, the display currency,
   who exported it and when.

---

### Edge Cases

- A plan with no campaigns: both views show "No campaigns yet — figures appear once activities have
  planned cost" with **Add campaign**.
- Line items exceed their activity's planned cost (negative unassigned balance, §4.3): "Not yet spread"
  is negative and red, and the totals still reconcile.
- A Custom spread that no longer adds up (§4.4): the difference falls into "Not yet spread", with a
  warning that links to the line item.
- Spread months outside the plan's year (a deposit, a campaign crossing the year end, R4): shown in
  "Before 2026" or "After 2026" columns, which appear only when not zero.
- An activity with planned cost 0, or a container with no planned cost: expected return shows "—" with
  "No planned cost".
- The PIR attribute is inactive in Settings: the PIR column shows "PIR is not in use" instead of values.
- A display currency other than USD: totals are converted from the USD total, so converted rows may
  differ from it by a cent; a footnote says so.

## Requirements *(mandatory)*

### Functional Requirements

**Where the views live**

- **FR-001**: The plan page (feature 004) MUST offer two read-only views: **SP&A (spend)** for every plan,
  and **BCPIR (business case)**, which shows figures only when the plan's **BCPIR plan** flag (field
  catalogue §1 #11, owned by feature 004) is Yes and otherwise explains how to set it, with a link.
- **FR-002**: BCPIR MUST remain a yes/no flag on the plan, and the BCPIR view MUST add no records of its
  own. [NEEDS CLARIFICATION: Is BCPIR a yes/no flag on a plan (Shell's form) or a business-case plan
  paired with an SP&A spend plan for every campaign (an alternative interpretation)? The answer decides
  whether this feature only adds views or adds new plan records with their own life cycle.]
- **FR-003**: Both views MUST be calculated from the plan's current data each time they open, using the
  same calculations as the plan page and All work (features 004–007, 010), so one plan never shows two
  different totals; deleted records MUST be left out. [NEEDS CLARIFICATION: Do the SP&A and BCPIR views
  need their own sign-off (Draft → Submitted → Published), separate from publishing the plan and
  approving activities? If yes, a signed-off view needs a frozen copy, which this requirement excludes.]
- **FR-004**: Every figure MUST have an ⓘ tip giving its source or formula; every "—" MUST say why; every
  campaign, initiative and activity name MUST link to its record. No value can be edited in these views.
- **FR-005**: Both views MUST offer the filter **All activities** (default) or **Approved activities
  only** (Approved, In progress, Complete); it applies to planned cost, spreads and goals, not budgets,
  and the filter in use MUST be shown.
- **FR-006**: Amounts MUST be shown in the user's display currency (feature 003), labelled, following
  design system §4 money rules; calculations use USD (D-010).

**SP&A view**

- **FR-007**: Summary cards MUST show plan budget, planned cost (§4.2), budget used (§4.2), and **left to
  plan** = budget − planned cost, shown red as "Over budget by X" when negative (D-015); a note MUST give
  the budget not yet given to campaigns (remaining plan budget, §4.1).
- **FR-008**: A campaign table MUST show, per campaign, budget, planned cost, budget used as a progress
  bar (red above 100 %) and left to plan, open to the same figures per initiative, with a totals row equal
  to the plan's planned cost. Its ⓘ MUST explain that left to plan differs from the remaining budget on
  the campaign form (§4.1).
- **FR-009**: A quarter table MUST split each campaign's planned cost into Q1–Q4 of the plan's year
  (calendar quarters, D-011) by adding up the monthly amounts of its line-item spreads (§4.4) in USD —
  each month converted at the line item's rate, any rounding cent going to its last month — with
  "Before <year>" and "After <year>" columns when not zero, and **Not yet spread** = planned cost − all
  spread amounts (covering unassigned cost, §4.3, and Custom spreads that do not add up). Every row MUST
  add up to its planned cost.
- **FR-010**: A table MUST show planned cost by activity type (number of activities, planned cost, share
  of the plan's planned cost), largest first.
- **FR-011**: A table MUST show line items' planned cost (USD) by spend type, with rows "Spend type not
  filled in" and "Not assigned to line items" (the activities' unassigned planned cost, §4.3), totalling
  the plan's planned cost.

**BCPIR view**

- **FR-012**: Summary cards MUST show planned cost, planned revenue (sum of activity revenue goals, §4.5),
  expected return and the count of activities whose PIR (field catalogue §4.2 #18) is Yes, No and not
  filled in; a line MUST compare the plan's goal with the sum of its activities' goals (§4.5).
- **FR-013**: A table MUST show each campaign, opening to its initiatives and activities, with planned
  cost, INQ, MQL, revenue (activity goals, or their sums for containers), expected return and PIR (the
  activity's value, or counts for containers); it MUST be sortable and filterable by PIR.
- **FR-014**: **Expected return** MUST be planned revenue ÷ planned cost at every level, shown with two
  decimals (rounded half-up) and "×"; activities without a revenue goal count as 0 revenue and their
  container states how many there are; with no planned cost it shows "—".
- **FR-015**: A collapsed section "From Measure and Finance (coming later)" MUST list actual cost, actual
  revenue, actual return and post-investment review result as "—" with "Actual return and
  post-investment review results come with Measure and Finance later" (pattern of §4.6).

**Export**

- **FR-016**: Each view MUST be exportable as a spreadsheet or CSV file with the same tables and figures
  as on screen and a header giving plan name and ID, view, filter, display currency, who exported it and
  when. Exporting MUST NOT change any record.

### Key Entities *(include if feature involves data)*

- **SP&A view of a plan**: calculated, not stored — budget, planned cost, budget used and left to plan
  per plan, campaign and initiative; planned cost per quarter, activity type and spend type.
- **BCPIR view of a plan**: calculated, not stored — planned cost, goals, expected return and PIR per
  plan, campaign, initiative and activity; placeholders for actual results.
- **BCPIR plan flag**: the existing yes/no on the plan (feature 004); no new stored record under the
  default answer to Q-001.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: From a plan page, a user sees each campaign's planned Q3 spend in at most 2 steps (design
  system §10).
- **SC-002**: For every seeded plan, 100 % of SP&A totals reconcile to the cent: campaigns, quarters (with
  before, after and not yet spread) and spend types each add up to the plan's planned cost.
- **SC-003**: For every seeded BCPIR plan, 100 % of expected-return figures match a hand calculation.
- **SC-004**: 100 % of figures carry a source or formula tip, and 100 % of "—" values say why.
- **SC-005**: 100 % of cells in an exported file match the figures on screen.
- **SC-006**: In a test with at least 5 first-time users, at least 90 % find, without help and within 2
  minutes, the campaign that is most over budget and the campaign with the highest expected return.

## Assumptions

- **Default for FR-002 (until Q-001 is answered):** BCPIR stays a yes/no on the plan, as on Shell's form
  and in the decision log's default. Paired SP&A/BCPIR plans per campaign (an alternative
  interpretation) are not built; if Shell chooses them, this spec is rewritten before planning.
- **Default for FR-003:** the views need no sign-off of their own; the plan's Publish (feature 004) and
  activity approvals (feature 008) are the agreement.
- Expanding "SP&A" as *Sales, Promotion & Advertising* (glossary) is an assumption still to be
  confirmed with Shell (Q-001). Hive9 has "SP&A Lubes" and "SP&A Mobility" Measure dashboards
  (FA §F4.4), which suggests a cross-plan spend report per business unit; this spec covers the per-plan
  view, and a cross-plan view belongs to All work (010) or Measure later.
- Hive9's activity categories BC_Lubes, BCPIR_Lubes, PIR_Retail, … (FA §E3.1.2) hint that Shell may tag
  activity types for business cases; this is not modelled and may change after Q-001.
- Planned revenue is the activity revenue goal from the plan's model (§4.5), in USD; expected return is a
  plan-time ratio, not ROI. Actual results need Finance and Measure (D-008).
- The plan page frame (feature 004) has room for extra views (tabs); this feature adds two. No
  permissions apply (D-004).
- Demo data: this feature marks "FY2026 Lubricants Europe" as a BCPIR plan and fills PIR on its
  activities, so both views show figures in a demo.
