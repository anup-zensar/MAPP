# Feature Specification: Home Dashboard

**Feature Branch**: `009-home-dashboard`

**Created**: 2026-10-18

**Status**: Draft

**Input**: User description: "Simple, modern home: summary cards, my plans, my activities, waiting for my approval, needs attention"

**References**: Domain model [§2.1–2.3 statuses, §2.6 completeness, §4.1 budget flows down, §4.2 cost flows up, §4.3 unassigned balance, §6 team, §7 activity log, §8 later visibility](../../docs/product/domain-model.md) · Design system [§2 layout and summary cards, §4 status and money, §5 components, §6.1 empty states, §10 usability targets](../../docs/product/design-system.md) · Glossary [display currency, budget used](../../docs/product/glossary.md) · Demo data [demo-users-and-data.md](../../docs/product/demo-users-and-data.md) · Feature 001 FR-018 (getting started), feature 008 (approvals) · Decisions D-004, D-005, D-010, D-015, D-016, D-020 · Open question Q-006

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See my work at a glance (Priority: P1)

Priya signs in and lands on Home, which follows the design-system Home layout: a blue "HOME" eyebrow,
"Welcome back, Priya." with her first name in grey, a one-line subtitle and a currency picker on the
right. Below, four summary cards show her teams' plans, the activities she owns, what waits for her
approval, and planned cost against plan budget. Each card opens the matching list.

**Why this priority**: Home is the first screen after sign-in; it must answer "how are my plans doing
and what is waiting for me?" without any navigation.

**Independent Test**: Sign in as Priya Raman with the seed data, read the four cards, compare each
figure with the Plans list, the plan page and the Approvals page, and follow each card's link.

**Acceptance Scenarios**:

1. **Given** Priya signs in, **When** Home opens, **Then** it shows the "HOME" eyebrow, "Welcome back,
   Priya.", the subtitle, the currency picker set to her display currency, and the four summary cards.
2. **Given** the seed data, **Then** **My teams' plans** shows 1 ("1 published"), **Activities I own**
   shows her activities with "n declined · n submitted", and **Planned cost vs plan budget** shows the
   FY2026 Lubricants Europe planned cost "of $1.20M plan budget" with the same budget used % as the plan
   page.
3. **Given** a card, **When** Priya chooses it, **Then** the matching list opens with the same filter
   (e.g. **Waiting for my approval** opens Approvals on the **Waiting for me** tab).
4. **Given** the currency picker, **When** Priya chooses EUR, **Then** every amount on Home is shown in
   EUR at once, labelled, and the choice becomes her display currency everywhere (feature 003).

---

### User Story 2 - See how my teams' plans are doing (Priority: P1)

Under the cards, a **Plans at a glance** card shows which team is working on which campaign: it lists
each plan with its status, its teams as coloured pills, the number of campaigns and activities, an
approval-progress bar, budget used and planned cost. A **My teams / Everything** switch widens the view
to every plan.

**Why this priority**: Team leaders and approvers watch plan progress; this replaces opening each plan.

**Independent Test**: Sign in as Sara Novak (two teams), check both FY2026 plans appear once each with
figures equal to their plan pages, switch to **Everything** and see FY2027 Global Brand appear.

**Acceptance Scenarios**:

1. **Given** Sara Novak, **Then** Plans at a glance lists FY2026 Lubricants Europe and FY2026 Mobility
   China, each once, with status chip, team pills, campaign and activity counts, "n of m approved" with
   a bar, budget used with a bar and planned cost.
2. **Given** a plan whose campaigns hold more planned cost than its budget, **Then** its budget-used bar
   is red and the percentage is above 100 %.
3. **Given** a plan with no budget or no activities, **Then** budget used shows "—" ("No budget") and
   approval progress shows "No activities".
4. **Given** the switch, **When** Sara chooses **Everything**, **Then** the plans card, the planned-cost
   card, Plans at a glance and Recent changes cover all plans, and her choice is kept on her next visit.

---

### User Story 3 - Know what needs me (Priority: P1)

Two cards tell the user what to do next. **Needs your attention** lists the records they own that have
a problem — missing fields, a declined activity, an over-allocated budget, line items exceeding the
planned cost — each with the reason and a link to fix it. **Waiting for my approval** lists the
activities waiting for their decision with quick links to review them.

**Why this priority**: "Status and next step are always visible" (design system §1) — Home is where a
user starts their day.

**Independent Test**: Sign in as Priya and Daniel with the seed data; Priya sees her declined activity
and the activity with a non-zero unassigned balance; Daniel sees the activity waiting for him and opens
it for review in one click.

**Acceptance Scenarios**:

1. **Given** Priya owns a Declined activity, **Then** Needs your attention shows it first with "Declined
   by Daniel Okoro on 12 Mar 2026: '<comment>'" and a link to its Approval section.
2. **Given** an activity Priya owns whose line items exceed its planned cost, **Then** it shows "Line items
   exceed the planned cost by $1,200.00" with a link to its line items.
3. **Given** Alex Morgan owns the FY2027 Global Brand campaign whose initiative budgets exceed its
   budget, **Then** he sees that campaign with "Over-allocated by $X" and a link to its budget.
4. **Given** a record with several problems, **Then** it appears once with all its reasons.
5. **Given** Daniel signs in, **Then** Waiting for my approval lists the submitted activities waiting
   for him with "Level 1 of 1" and how long they have waited; **Review** opens the Approvals page with
   that activity's preview open.
6. **Given** nothing needs attention or waits for approval, **Then** each card says so in one friendly
   sentence ("Nothing needs your attention right now.").

---

### User Story 4 - Follow my activities and recent changes (Priority: P2)

**My activities** groups the activities the user owns by status, so they see what is being planned,
waiting, approved, running or finished. **Recent changes** shows the latest activity-log entries for
the plans in scope: who changed what, and when.

**Why this priority**: Useful daily context, but the user can work without it.

**Independent Test**: Sign in as Priya, check that each seeded activity she owns appears under its
status, edit one activity's planned cost, return Home and see the change at the top of Recent changes.

**Acceptance Scenarios**:

1. **Given** Priya's seeded activities, **Then** My activities shows groups Declined, Created,
   Submitted, Approved, In progress and Complete with counts; Complete is collapsed; empty groups are
   hidden.
2. **Given** a group with more than five activities, **Then** it shows five and "Show all n".
3. **Given** Priya changed an activity's planned cost, **When** she returns Home, **Then** Recent
   changes starts with "Priya Raman changed planned cost of activity 'Launch film' (A-1042)" and "just
   now", linking to the activity.

---

### User Story 5 - Get started as a new user (Priority: P1)

A brand-new user who belongs to no team, or whose teams have no plan yet, sees the **Getting started**
card from feature 001 at the top of Home, so they still know what to do first.

**Why this priority**: The dashboard replaces 001's getting-started Home; new users must not land on an
empty screen.

**Independent Test**: Add a new demo person with no team, sign in as them and follow the steps.

**Acceptance Scenarios**:

1. **Given** a user in no team, **Then** Home shows the Getting started card with the five steps and "You're
   not in a team yet. Plans belong to teams — add yourself to one on the Teams page.", and every other
   card shows its own empty state.
2. **Given** that user joins a team that has a plan, **When** they return Home, **Then** the Getting
   started card is no longer shown.

---

### Edge Cases

- A plan belongs to two of the user's teams: it is counted and listed once.
- The user owns an activity in a plan outside their teams: it still appears in the personal cards (My
  activities, Needs your attention), which always cover all plans.
- Some in-scope plans have no budget: the planned-cost card says "Includes 2 plans with no budget".
- The user's display currency has been set inactive in Settings: amounts fall back to USD and the
  picker says "EUR is no longer available — showing USD".
- A record in Recent changes was deleted: it shows "(in recycle bin)" without a link.
- An ongoing activity is never shown as Complete (domain model §2.3).
- More than 10 plans in scope: Plans at a glance shows 10 and "View all n plans".
- One card cannot load: it shows "This section couldn't load" with **Try again**; the others still show.

## Requirements *(mandatory)*

### Functional Requirements

**Page header and scope**

- **FR-001**: Home MUST replace feature 001's getting-started Home and use the page header pattern
  (design system §2) in MAPP's Home layout: blue eyebrow "HOME", title "Welcome back," with the
  user's first name as the grey highlight ("Welcome back, Priya."), subtitle "Your plans, your activities
  and what needs you today.", and a **Currency** picker on the right.
  *Corrected by [038](../038-qa-findings/spec.md) (QA L-26): the design system (§2) dropped the eyebrow — "Home is
  the only exception: its title greets" — and the subtitle shows only to people with a role; the code is right.
  038 FR-032 (QA M-5): "Activities I own" opens All work on the plan years of those activities.*
- **FR-002**: The currency picker MUST list active currencies (feature 002), start on the user's display
  currency (feature 003) and, when changed, save it as the user's display currency and redraw every
  amount on Home in that currency.
- **FR-003**: Home MUST offer a pill switch **My teams / Everything**, default **My teams**, remembered
  per user. **My teams** covers plans with at least one of the user's teams; **Everything** covers all
  plans. The switch MUST change the plans card, the planned-cost card, Plans at a glance and Recent
  changes; a one-line hint MUST say that the user's own work and approvals always cover all plans.
- **FR-004**: A plan in several of the user's teams MUST be counted once everywhere on Home.

**Summary cards**

- **FR-005**: Home MUST show four summary cards (design system v2) in Shell's design language (no tinted icon
  tile), a label with an ⓘ tip, a large value coloured only when the colour carries meaning:
  (1) **My teams' plans** ("All plans" under **Everything**): number of plans, with "n published · n
  draft"; (2) **Activities I own**: number, with "n declined · n submitted"; (3) **Waiting for my
  approval**: number (amber when above zero), with "oldest waiting n days"; (4) **Planned cost vs plan
  budget**: total planned cost of the plans in scope, "of <total> plan budget", and a budget-used bar
  with "n % used" (red above 100 %).
- **FR-006**: Budget used on card 4 MUST be the total planned cost divided by the total plan budget of
  the plans in scope (domain model §4.2); when some plans have no budget the card MUST say "Includes n
  plans with no budget"; with no budget at all it MUST show "—" and "No plan budgets set".
- **FR-007**: Each card MUST open the matching list with the same filter: (1) and (4) the Plans list with
  the same scope (feature 004); (2) All work filtered to activities owned by the user (feature 010),
  or the My activities card on Home until feature 010 exists; (3) the Approvals page, **Waiting for me**
  tab (feature 008).

**Plans at a glance**

- **FR-008**: A content card **Plans at a glance** (ⓘ "Every plan your teams work on, the teams on it,
  and live status rolled up from its activities."; subtitle "Live roll-up across your teams" or "across
  all plans") MUST list the plans in scope with: plan name (link), ID, year and AOP tag; status chip;
  teams as coloured pills with a dot, the user's own teams first; number of campaigns; number of
  activities; approval progress as a bar with "n of m approved" or "No activities"; budget used as a bar
  with a percentage (red above 100 %, "—" without a plan budget); planned cost.
- **FR-009**: A plan's approval progress MUST apply the rule of domain model §2.2 to all of the plan's
  activities (Approved, In progress and Complete count as approved).
- **FR-010**: Rows MUST be ordered by year, newest first, then by name; at most 10 rows are shown, with
  "View all n plans" opening the Plans list; choosing a row opens the plan page.

**Needs your attention**

- **FR-011**: A card **Needs your attention** MUST list the plans, campaigns, initiatives and activities
  the user owns that have at least one of these reasons: (a) the activity is Declined — "Declined by
  <name> on <date>: '<comment>'"; (b) negative unassigned planned cost — "Line items exceed the planned
  cost by X" (domain model §4.3); (c) a plan or campaign whose children's budgets exceed its budget —
  "Over-allocated by X" (domain model §4.1, D-015); (d) budget used above 100 % — "Planned cost is X
  over budget" (domain model §4.2);
  (e) missing fields — "n fields needed", plus "n items below need attention" where it applies (domain
  model §2.6).
- **FR-012**: Each item MUST show the level tag, name, ID, plan and all its reasons, and link to the
  place where it is fixed (field list, Approval section, line items or budget). Items MUST be ordered
  declined, then balance, then budget, then missing fields, most recently changed first within each;
  8 are shown, with "Show all n".

**Waiting for my approval**

- **FR-013**: A card **Waiting for my approval** MUST list the same activities as the **Waiting for
  me** tab of feature 008, oldest first, at most 5: name (link), ID, plan, owner, planned cost, "Level n
  of m" and time waiting; each with **Review**, which opens the Approvals page with that activity's
  preview open; and "View all n" opening that tab.

**My activities**

- **FR-014**: A card **My activities** MUST group the activities the user owns by displayed status
  (domain model §2.3) in the order Declined, Created, Submitted, Approved, In progress, Complete, each
  with a status chip and count, at most 5 rows and "Show all n"; Complete MUST start collapsed and empty
  groups MUST be hidden. Each row shows the name (link), ID, plan › campaign, dates, planned cost and
  the completeness chip.

**Recent changes**

- **FR-015**: A card **Recent changes** MUST show the 10 most recent activity-log entries (feature 001)
  about plans, campaigns, initiatives, activities and line items of the plans in scope: the actor's
  avatar and name, a plain sentence naming the action, the changed fields and the record ("changed
  planned cost of activity 'Launch film' (A-1042)"), and a relative time with the full date and time on
  hover. The record name MUST link to the record; deleted records MUST show "(in recycle bin)".
- **FR-016**: The card MUST offer **View activity log**, opening the activity log page (feature 011, or
  its "Coming soon" page until then).

**Figures and currency**

- **FR-017**: Every count, amount, percentage and status on Home MUST be calculated with the same rules
  as the plan, campaign, initiative, activity and approvals pages (domain model §2 and §4), so each
  figure equals the one those pages show for the same scope; totals MUST be added in USD and converted
  to the display currency once (D-010).
- **FR-018**: Amounts MUST follow design system §4: compact in cards with the full value on hover,
  always labelled with the currency, negatives in red with a minus sign, "—" when empty.
- **FR-019**: Home MUST show current figures each time it is opened or returned to; deleted records
  MUST never count (feature 001).

**New users, empty and loading states**

- **FR-020**: When the user belongs to no team, or none of their teams has a plan, Home MUST show the
  **Getting started** card of feature 001 (FR-018 there) above the summary cards; for a user in no
  team it MUST add "You're not in a team yet. Plans belong to teams — add yourself to one on the Teams
  page."
- **FR-021**: Every card MUST have its own empty state (one sentence and, where useful, an action), never
  a blank area; with **My teams** and no plans, Plans at a glance MUST offer **Switch to Everything**.
- **FR-022**: While loading, each card MUST show a loading placeholder; a card that fails MUST show "This
  section couldn't load" with **Try again** without hiding the others.

### Key Entities *(include if feature involves data)*

Home stores almost nothing of its own; it reads plans, campaigns, initiatives, activities, line items,
approval rounds and the activity log from features 001 and 004–008.

- **Home preferences**: the user's **My teams / Everything** choice. The display currency is the user's
  preference from feature 003.
- **Attention item** (derived, not stored): a record the user owns, its reasons and the link to fix it.
- **Team colour** (derived): a colour MAPP gives each team for its pills, the same everywhere.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In a test with at least 5 first-time users, at least 90 % answer from Home alone, within
  2 minutes in total and without help: how many activities wait for their approval, which of their
  activities was declined and why, which plan is over budget, and the planned cost of their teams' plans.
- **SC-002**: For every seeded user and both scopes, 100 % of figures on Home equal those on the Plans
  list, plan pages and Approvals page in the same display currency.
- **SC-003**: From Home, any item that needs attention or waits for approval is opened in 1 step, and a
  waiting activity is approved in at most 3 steps (design system §10).
- **SC-004**: Home shows all its cards within 2 seconds with the seed data, and within 3 seconds for a
  user whose scope holds 50 plans and 5,000 activities.
- **SC-005**: Changing the display currency updates 100 % of amounts on Home within 1 second, each
  labelled with its currency.
- **SC-006**: A brand-new user with no team sees the getting-started steps and no blank or broken card.
- **SC-007**: Home passes an automated WCAG 2.2 AA check, can be used with the keyboard alone, and every
  bar and coloured pill also shows its value or name as text.

## Assumptions

- Home is the same for every user (no manager, lead or member variants), because
  there are no roles in this phase (D-004). The **My teams / Everything** switch shows visibility
  without enforcing it, like the Plans list (domain model §8, Q-006).
- Teams have no colour of their own (domain model §6); MAPP gives each team a stable colour from a
  fixed palette so its pills look the same on every screen.
- A plan's approval progress uses the campaign and initiative rule of domain model §2.2.
- The Plans list (feature 004) and All work (feature 010) can be opened with a filter already applied;
  feature 008's **Waiting for me** definition is reused unchanged.
- Line-item problems appear through their activity ("n items below need attention"), not as separate
  items, even when a line item has another owner.
- Charts, saved dashboards, AI analysis and a floating assistant are out of scope.
- Since feature 014, who may see and change these records follows roles and team scopes (014 Table A and Table C; 014 FR-041).
