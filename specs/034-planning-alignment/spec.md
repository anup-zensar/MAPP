# Feature Specification: Planning Rules Aligned with Shell's Process

**Feature Branch**: `034-planning-alignment`

**Created**: 2026-10-27

**Status**: Draft

**Input**: Product owner instruction of 2026-10-27: existing behaviour that is "partially wrong" is fixed so
that the planning steps before a purchase request follow Shell's process — (1) line-item changes keep the
activity's approval (D-322); (2) Shell's own configuration lists replace invented options: GL code (13),
spend type, spend category and contracted spend on line items, and the zero-based cost hierarchy on
activities (D-325); (3) a plan's year is the current year to ten years ahead, no past year, and existing
plans keep their year (D-326).

**Amends feature 007** ([Line items](../007-line-items/spec.md)): User story 4 and its scenarios 3 and 5,
**FR-031 (replaced)**, FR-033 (second sentence replaced). **Amends feature 008**
([Activity approvals](../008-activity-approvals/spec.md)): User story 5 scenario 2 (replaced), FR-021 (the
explanation shown for Approved activities). **Amends feature 004** ([Plans](../004-plans/spec.md)): FR-006
(year choices), FR-028 (the last permitted plan year). **Amends feature 002**
([Settings](../002-settings-and-shell-attributes/spec.md)): the demo option lists of five Shell attributes
(data model §13). The table at the end lists every amendment.

**References**: Client specs 005 FR-002 (line items in any status do not change the activity's status), 009
FR-050 (only the activity's planned cost re-approves), 004 FR-009 (year from the current year and the
following ten) and FR-036 (Shell activity attributes) · Hive9 functional analysis §A4, §B3, §B6, §B7, §D3 and
§E3.0.2 (the value lists observed live) · GSAP workshop summary · Decisions D-013, D-016, D-322, D-325, D-326 ·
Domain model §2.5, §4.3, R11 · Field catalogue §1 #2, §4.2 #21, §5 #8–#11.

In Shell's process a marketer gets the activity approved first, then adds and adjusts its line items, asks
GSAP for each line item's SWBS code and raises purchase requests against them (FA §A4, §D3). MAPP today
sends an approved activity back to Created whenever a line item is added, re-costed or removed, which would
undo the approval every time Shell's process moves forward. This feature keeps approval tied to what was
approved — the activity's own planned cost and dates — and puts Shell's real finance coding lists and plan
years in place.

Demo people (014 seed): Priya Raman and Mei Lin are planners, Daniel Okoro approves Lubricants Europe, Alex
Morgan is the administrator. Demo activities: "Social always-on" (A-1002) is Approved with the line items
"Venue hire" (15,800.00 GBP) and "Speaker fees" (5,000.00 USD); "Trade magazine ads" (A-1004) is Submitted
and waits for Daniel.

## Clarifications

### Session 2026-10-27

No product owner was available in the loop; each point is resolved from the sources (client spec > Hive9
functional analysis > workshop > guide; decision log) and recorded as an assumption for the product owner
to confirm.

- Q: Shell has 57 spend types but the functional analysis names only 25 — should MAPP seed the 25 or make up
  the rest? → A: Seed the 25 named ones, in the analysis's order; the others are added in Settings when Shell
  supplies them (client 002: "seeded with the values recorded in FA §E3.0.2 … and completed by import").
- Q: Should GL code keep depending on spend type when no source gives Shell's spend type → GL code mapping?
  → A: No. GL code is seeded without a parent attribute, so every code is offered with every spend type; the
  workshop's "one spend type can map to several GL codes" stays possible through Settings once Shell supplies
  the mapping (FR-011, domain model R11).
- Q: Should the option labels keep MAPP's typographic dash ("–") or Shell's text? → A: Shell's text, character
  for character as observed live (hyphen "-"), because the values travel to GSAP with the SWBS request and
  purchase requests (FR-006 – FR-010).
- Q: May a plan of a past year still be copied within its own year? → A: Yes, as today: the permitted years
  bind a new plan and a next-year copy only (FR-016); "existing plans keep their year" (D-326).
- Q: Is the zero-based cost hierarchy shown only on Lubricants plans (client 002 lists it in the Lubricants
  set)? → A: No, on every activity as today: its live values include RETAIL ones, which Mobility plans use
  (FR-010).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Work on the line items of an approved activity without losing its approval (Priority: P1)

Priya's activity "Social always-on" is approved for 80,000.00 USD. She adds a line item "Agency fee", changes
the venue hire from GBP to EUR and removes the speaker fees. Each change saves straight away; the activity
stays Approved and its approval history is untouched. Changing the activity's own planned cost to
90,000.00 USD still asks first and sends it back for approval.

**Why this priority**: Shell adds line items after the activity is approved, then requests the SWBS code and
raises the purchase request (FA §A4, §D3; client 005 FR-002). As long as every line-item change restarts
approval, that process cannot be followed in MAPP.

**Independent Test**: On the demo data, add, duplicate, re-cost and delete line items of the Approved
activity "Social always-on" and the Submitted "Trade magazine ads"; neither changes status, no confirmation
appears, and Daniel still finds "Trade magazine ads" waiting for him. Then change the planned cost of
"Social always-on": the confirmation appears and the activity returns to Created.

**Acceptance Scenarios**:

1. **Given** the Approved activity "Social always-on", **When** Priya adds the line item "Agency fee" of
   2,000.00 USD and saves, **Then** it is saved without any confirmation, the activity is still Approved, its
   approval history has no new entry, and the activity log has one "created" entry for the line item and no
   status entry for the activity.
2. **Given** the Submitted activity "Trade magazine ads", **When** Priya changes the planned cost or the
   currency of one of its line items and saves, **Then** it is saved without any confirmation, the activity
   stays Submitted, its open round continues at the same level and Daniel still finds it in **Waiting for
   me**.
3. **Given** the Approved activity "Social always-on", **When** Priya duplicates "Venue hire", **Then** the
   copy "Venue hire (copy)" is created and opens without any confirmation, and the activity stays Approved.
4. **Given** the Approved activity "Social always-on", **When** Priya deletes "Speaker fees", **Then** the
   delete dialog names the line item, the effect on the unassigned planned cost and the recycle bin, and does
   not say the activity will need approval again; after the delete the activity is still Approved, and
   **Undo** — or a later restore from the recycle bin — brings the line item back without changing the
   activity's status.
5. **Given** the Approved activity "Social always-on", **When** Priya changes the activity's own planned cost,
   start date, end date or ongoing flag, **Then** "This activity will need approval again" still lists the
   change, and on confirmation the activity returns to Created and its round records the change (unchanged
   behaviour of features 006 and 008).
6. **Given** the Approved activity "Social always-on" (80,000.00 USD planned), **When** Priya adds a line item
   that makes the line items total more than the planned cost, **Then** it is saved, the unassigned planned
   cost shows negative in red with its warning, and the activity stays Approved; **When** the activity is
   later back in Created (after a planned-cost change) or Declined, **Then** submitting or re-submitting it is
   refused until the unassigned planned cost is no longer negative (D-016).
7. **Given** an Approved activity, **When** anyone opens its approval section, **Then** the explanation of
   why it can't be recalled names only the activity's planned cost and dates: "Approved activities can't be
   recalled. Changing its planned cost or dates sends it back for approval."

---

### User Story 2 - Code line items and activities with Shell's own lists (Priority: P1)

Priya codes a line item with Shell's spend type "BTL - Event" and GL code "7250150 - Sponsorship programmes",
marks it "Yes - ST" contracted spend and spend category "Tradeshow". On the activity she picks the
zero-based cost hierarchy "LUBES - Growth drivers". These are the values Shell uses in Hive9 today and that
GSAP receives with the SWBS request.

**Why this priority**: The finance coding travels with the SWBS request and the purchase request (FA §B7,
§D3); invented options make every demo of that process wrong (D-325).

**Independent Test**: On fresh demo data, open the line-item form of any activity and count the options of
GL code (13), spend type (25), spend category (19) and contracted spend (5), and the zero-based cost hierarchy
on the activity form (10); compare every label with the FA; save a line item with Shell values; check that
every demo record still shows the completeness it showed before.

**Acceptance Scenarios**:

1. **Given** fresh demo data, **When** Alex opens Settings › Shell attributes › GL code, **Then** it lists
   Shell's 13 marketing GL codes in this order: 6140900 - PMTDR Credit · 7250050 - Advertising - Media
   Planning/Auditing · 7250060 - Advertising - Corp/Agency/Production · 7250070 - Advertising - Media
   Buying · 7250080 - Advertising - Digital · 7250100 - Public Relations · 7250150 - Sponsorship programmes ·
   7250190 - Loyalty Programmes - Materials · 7250200 - Loyalty Programmes - Services · 7250250 - Market
   Research · 7250300 - Promotions & POP - Incentives/Reimbursement · 7250310 - Promotions & POP -
   Agency/Print/Logistics · 7250350 - Brand Development & Coordination; and GL code depends on no other
   attribute.
2. **Given** a line-item form, **When** Priya chooses any spend type, **Then** GL code still offers all 13
   codes, and changing the spend type never clears a chosen GL code.
3. **Given** a line-item form, **Then** Spend type offers the 25 spend types of FR-007 in that order, Spend
   category the 19 values of FR-008 and Contracted spend the 5 values of FR-009.
4. **Given** an activity form of a Lubricants or a Mobility plan, **Then** Zero-based cost hierarchy offers
   the 10 values of FR-010, several may be chosen, and it is optional.
5. **Given** fresh demo data, **When** anyone opens a demo activity or line item, **Then** every coded value
   is one of Shell's values, and each record shows the same completeness as before this feature (for
   example "Station signage" of "Grand opening Shanghai" still lacks its GL code).
6. **Given** the seeded lists, **When** Alex adds, renames, deactivates or reorders an option, or links GL
   codes to spend types, **Then** Settings behaves as feature 002 defines (unchanged).

---

### User Story 3 - Plan for this year and the next ten, never for a past year (Priority: P2)

Mei creates a plan in October 2026. The Year list offers 2026 to 2036, with 2026 chosen; 2025 is not offered.
In 2027 she can still open and edit her FY2026 plan, whose year stays 2026.

**Why this priority**: Shell plans several years ahead and never starts a new plan in a closed year (client
004 FR-009; FA §B3). It is a small rule change with no effect on existing plans.

**Independent Test**: With the date set to 27 October 2026, open **New plan** and read the Year list; send a
plan of 2025 and of 2037 (refused); with the date set to 2027, edit a FY2026 plan (saved, year unchanged).

**Acceptance Scenarios**:

1. **Given** today is 27 October 2026, **When** Mei chooses **New plan**, **Then** Year offers 2026, 2027, …,
   2036 (11 years), defaults to 2026 and does not offer 2025.
2. **Given** today is 27 October 2026, **When** a plan of 2025 or of 2037 is saved, **Then** nothing is saved
   and Year says "Choose a year from 2026 to 2036." together with every other problem of the form.
3. **Given** the plan "FY2026 Mobility China" and today is 3 February 2027, **When** Mei edits and saves it,
   **Then** it is saved, its year stays 2026 and is shown locked as before, and every other action on it
   (publish, add records, copy within its year) works as before.
4. **Given** today is 27 October 2026, **When** Mei copies a plan of 2036, **Then** "Next year (2037)" is shown
   unavailable with "Plans can only be created up to 2036."; **When** she copies a plan of 2035, **Then**
   "Next year (2036)" is available.

---

### Edge Cases

- **Approval state when line items change:** a line-item change on a Created, Declined, Submitted or Approved
  (shown as In progress or Complete) activity never changes the activity's status, never ends or restarts a
  round, and never notifies anyone about approval.
- **Line items and the approved amount:** an approved activity may end up with line items that total more
  or less than its approved planned cost. The negative unassigned planned cost stays visible (red, with its
  warning) and blocks the next submission, but does not undo the approval (D-016, D-322).
- **Undo after a delete:** Undo restores the line item exactly; there is no longer an approval to give back,
  because the delete ended none.
- **Splitting an activity** (feature 016) changes the activity's own planned cost, so it keeps asking for
  confirmation and restarting approval as today.
- **Older web pages still open** that send the old confirmation with a line-item change are served normally;
  the confirmation is ignored.
- **Stored values outside the new lists:** a database seeded before this feature keeps its old options
  until it is seeded again; the new lists apply to every freshly seeded demo database. Options an
  administrator added stay as they are.
- **Spend types not yet recorded:** Shell has 57 spend types; the functional analysis records 25 of them by
  name. The other 32 are added through Settings once Shell supplies them (D-325).
- **A plan in the last permitted year** (current + 10) can be copied only within its year.
- **New year's day:** on 1 January the list moves on one year by itself; plans of the year just ended keep
  working.
- **Global budgets and finance budget items** keep their own year rules (current year − 1 to + 5, features
  014 and 002); a plan of a later year simply has no finance budget of its year to choose yet.

## Requirements *(mandatory)*

### Functional Requirements

**Line-item changes keep the activity's approval (D-322)**

- **FR-001**: Adding, duplicating or deleting a line item, and changing any of its fields — its planned cost
  and currency included — MUST NOT change the status of its activity, MUST NOT end, restart or otherwise
  change the activity's approval rounds, and MUST NOT ask the person to confirm that the activity will need
  approval again. This applies whatever the activity's status. *(Replaces 007 FR-031.)*
- **FR-002**: Bringing a deleted line item back — by **Undo** or from the recycle bin — MUST NOT change the
  status of its activity. *(Replaces the second sentence of 007 FR-033.)*
- **FR-003**: Only a change to the activity's own planned cost, start date, end date or ongoing flag (and a
  split, which changes its planned cost) MAY send a Submitted or Approved activity back to Created, after the
  confirmation "This activity will need approval again", as features 006, 008 and 016 define (domain model
  §2.5; unchanged).
- **FR-004**: A negative unassigned planned cost MUST still be shown as a warning wherever the balance is
  shown, and MUST still block submitting and re-submitting the activity (D-016); it MUST NOT change the
  status of an activity that is already Submitted or Approved.
- **FR-005**: Every message, dialog and help text MUST stop saying that line-item changes send an activity
  back for approval: the line-item delete dialog drops "This activity will need approval again.", and the
  explanation for an Approved activity reads "Approved activities can't be recalled. Changing its planned
  cost or dates sends it back for approval." *(Amends 008 FR-021.)*

**Shell's configuration lists (D-325)**

- **FR-006**: The demo configuration MUST give the line-item attribute **GL code** exactly Shell's 13
  marketing GL codes, labelled "<code> - <name>" as in FA §E3.0.2, in the order of acceptance scenario 2.1.
- **FR-007**: **Spend type** MUST offer the 25 spend types the FA records by name, in this order: ATL - Agency
  Fees - Creative · ATL - Agency Fees - PR · ATL - Digital · ATL - Event · ATL - Paid Media Campaign
  Activation · ATL - Paid Media Strategy and Planning · ATL - Public Relations · ATL - Sponsorships · Audits
  e.g. mystery shopper · BTL - Agency Fees · BTL - Branded Shop · BTL - Capability building Training · BTL -
  CRM Campaign · BTL - Customer Meetings/Conference · BTL - Distributor Marketing Budget · BTL - Event · BTL -
  Incentive Scheme · BTL - Loyalty Operations Costs · BTL - Market research/Insights · BTL - Merchandise ·
  BTL - Pipeline · BTL - Promotions · BTL - Sponsorship · Contractors or Field Support · CRM - Agency Fee.
- **FR-008**: **Spend category** MUST offer, in this order: Not Applicable · ABM (B2B Media) · B2B Customer
  Specific · Brand · Canada Customer Specific · Channel - DIFM · Channel - DIY · Channel - Indirect · Consumer
  Media · Content, Creative & Assets · Influencer Marketing · Innovation · Insights · MarTech · Motorsports ·
  Program · Promotions · Sustainability · Tradeshow (19).
- **FR-009**: **Contracted spend** MUST offer, in this order: Not Applicable · No · Yes - Both · Yes - LT ·
  Yes - ST (5).
- **FR-010**: The activity attribute **Zero-based cost hierarchy** MUST offer, in this order: LUBES - Growth
  drivers · LUBES - Legal or regulatory requirements · LUBES - Long term contractual obligations · LUBES -
  Other · LUBES - Protecting the core · LUBES - Strategic Priorities · LUBES - Tactical initiatives · RETAIL -
  Long-Term Contracts · RETAIL - Strategic Priorities · RETAIL - Tactical Investments (10). It stays an
  optional multiple choice shown on every activity.
- **FR-011**: GL code MUST NOT depend on spend type in the demo configuration: every GL code is offered
  whatever spend type is chosen, and choosing a spend type never clears a GL code. Linking options of
  dependent attributes in Settings stays available (feature 002). *(Replaces the seeded Spend type → GL code
  links; domain model R11.)*
- **FR-012**: The labels, markings (Spend type and GL code needed to submit; the other three optional),
  sections and business-unit conditions of these five attributes MUST stay as they are; their help texts
  MUST describe Shell's lists and MUST NOT say that GL codes are chosen within a spend type.
- **FR-013**: Every demo record that holds one of these attributes MUST hold a value from the new lists, and
  every demo record MUST show the same completeness, status and figures as before this feature, so that every
  demonstrated journey keeps working.

**Plan year (D-326)**

- **FR-014**: The Year of a new plan MUST be chosen from the current year to the current year + 10, with the
  current year as default; past years MUST NOT be offered, and a year outside that range MUST be refused with
  "Choose a year from <current> to <current + 10>." together with every other problem of the form.
  *(Amends 004 FR-006.)*
- **FR-015**: A saved plan MUST keep its year, also once that year is in the past: viewing, editing,
  publishing, adding and changing its records, copying it within its year and deleting it MUST work as
  before.
- **FR-016**: A next-year copy MUST be available only when the source year + 1 is at most the current year +
  10, and otherwise shown unavailable with "Plans can only be created up to <current + 10>."
  *(Amends 004 FR-028.)*

**Records and traceability**

- **FR-017**: The domain model §2.5 and rule R11, and the field catalogue §1 #2, §4.2 #21 and §5 #8–#11, MUST
  state these rules and lists, so that the product documents and MAPP agree.

### Key Entities *(include if feature involves data)*

- **Activity**: keeps its status through line-item changes; only its own planned cost, dates and ongoing
  flag restart its approval (domain model §2.5).
- **Line item**: its changes are logged on the line item only; it carries Shell's spend type, GL code, spend
  category and contracted spend values.
- **Shell attribute option lists** (Settings, feature 002): GL code, Spend type, Spend category, Contracted
  spend (line item) and Zero-based cost hierarchy (activity), seeded with Shell's values.
- **Plan**: its year is chosen from the current year to ten years ahead when it is created, and never changes
  afterwards.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On a Submitted or Approved activity, 100 % of line-item adds, duplicates, edits, deletes and
  restores leave the activity's status and approval rounds unchanged, and none shows a re-approval
  confirmation.
- **SC-002**: A planner adds a line item to an approved activity in one save attempt, with no step beyond
  those of the line-item form (design system §10: an activity with one line item in ≤ 15 steps).
- **SC-003**: The five lists match the functional analysis character for character: 13 GL codes, 25 spend
  types, 19 spend categories, 5 contracted-spend values, 10 zero-based cost hierarchy values.
- **SC-004**: 100 % of demo records keep the completeness, status and figures they had before the change,
  and every end-to-end demo journey that passed before still passes.
- **SC-005**: The new-plan Year list shows exactly 11 years, none of them in the past, and 100 % of plans of a
  past year can still be edited and saved.

## Assumptions

- The 32 spend types Shell has but the functional analysis does not name are not invented; they are added in
  Settings when Shell supplies them (D-325; client 002 "seeded … and completed by import").
- No source documents Shell's spend type → GL code mapping (the workshop only says one spend type can map to
  several GL codes); seeding GL codes without a parent is closer to Shell's data than an invented mapping.
- The zero-based cost hierarchy stays on every activity rather than only on Lubricants plans (client 002
  lists it in the Lubricants set), because its live values include the RETAIL ones used by Mobility.
- A copy within its own year stays possible for a plan of any year, as today; only a next-year copy is
  bound by the permitted years.
- Global budget and finance budget item years keep their own rules (current − 1 to + 5).
- Line-item changes are still logged on the line item with who, what and when (feature 011); D-013's
  "line-item coding without re-approval" now covers every line-item change.

## Amendment index

| Feature | Requirement | Change |
|---|---|---|
| 007 | US4 (text), scenarios 3 and 5 | Line-item changes no longer ask for confirmation or send the activity back for approval |
| 007 | FR-031 | Replaced by 034 FR-001 |
| 007 | FR-033 | Second sentence replaced by 034 FR-002 (Undo part unchanged) |
| 008 | US5 scenario 2 | Replaced: a line-item currency change keeps the activity Approved |
| 008 | FR-021 | The explanation for Approved activities names only planned cost and dates (034 FR-005) |
| 004 | FR-006 | Year choices: current year to + 10 (034 FR-014) |
| 004 | FR-028 | Last permitted plan year: current + 10 (034 FR-016) |
| 002 | Data model §13 | Demo options of GL code, Spend type, Spend category, Contracted spend and Zero-based cost hierarchy; GL code without parent (034 FR-006 – FR-011) |
| Docs | Domain model §2.5, R11; field catalogue §1 #2, §4.2 #21, §5 #8–#11 | Updated (034 FR-017) |
