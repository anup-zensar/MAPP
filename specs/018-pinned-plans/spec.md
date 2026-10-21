# Feature Specification: Pinned Plans

**Feature Branch**: `018-pinned-plans`

**Created**: 2026-10-21

**Status**: Built

**Input**: User description: "A person can pin plans; pinned plans sit at the top of the Plans list with a star; saved per person on the server."

**References**: 004 spec (Plans list FR-001 – FR-004) · 001 research R16 and **D-038** (display preferences: per person, saved on the server, not logged, no version) · 016 FR-033 (the list's filters and layouts this feature must not disturb) · Design system [§5 buttons (icon-only buttons take their name from `aria-label` and carry a tooltip with the same words), §3.1 yellow is a fill](../../docs/product/design-system.md) · Decision **D-317**

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Keep the plans I work in daily at the top (Priority: P1)

As a planner I pin the plans I work in daily, so they sit at the top of my Plans list.

Priya works in two of the forty plans her filters show. She presses the star on "FY2026 Lubricants
Europe"; the card moves to the top of the list and its star fills yellow. The choice is hers alone
and follows her to every browser she signs in from, in both the cards and the table layout.
Pressing the filled star puts the plan back among the others.

**Why this priority**: The Plans list grows with every year and team; the people who live in it
asked for their own plans first without changing anyone else's list.

**Independent Test**: Pin and unpin seeded plans in both layouts, reload and sign in from a second
browser, and check the order, the star and that nothing else about the list changed.

**Acceptance Scenarios**:

1. **Given** the Plans list in cards with "FY2025 Lubricants Europe" shown last, **When** Priya
   presses the star named "Pin plan 'FY2025 Lubricants Europe'", **Then** the plan moves to the top
   of the list at once, its star is filled and now named "Unpin plan 'FY2025 Lubricants Europe'",
   and the choice is saved for Priya on the server (D-038: no log entry, no version change).
2. **Given** a pinned plan, **When** Priya presses its filled star, **Then** the plan returns to
   the place the list's own sort gives it and the star shows as an outline again.
3. **Given** Priya pinned two plans, **When** she opens the list in the table layout (or signs in
   from another browser), **Then** the same two plans sit in the first rows, each marked with the
   filled star, and within the pinned rows and the rows below the list keeps its current sort.
4. **Given** Priya pinned a plan that someone later deleted (or that her filters or visibility no
   longer include), **When** she opens the list, **Then** the list simply shows without it — no
   error, no empty row — and the stored id stays, harmless, until she unpins a shown plan again.

### Edge Cases

- Pinning the 101st plan is refused: at most 100 plans can be pinned (FR-006); the web app says so
  and changes nothing.
- A request that repeats an id pins the plan once: duplicates are dropped, keeping the order sent
  (FR-005).
- An id that is not a plan id (`^P-\d+$`) is refused next to the field; an id of a plan that does
  not exist is allowed — the stored id is harmless (FR-005, FR-007).
- A person who pins nothing sees the list exactly as before this feature, empty states included.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A person can pin and unpin any plan from the Plans list with one star control per
  plan, in both the cards and the table layout.
- **FR-002**: Pinned plans are listed first in both layouts; within the pinned group and within the
  rest, the list keeps its current sort unchanged.
- **FR-003**: The star marks a pinned plan — filled while pinned, an outline while not — and stays
  visible on pinned cards and rows. Its accessible name names the plan: "Pin plan '&lt;name&gt;'" /
  "Unpin plan '&lt;name&gt;'", with a tooltip saying the same words (icon-only button, design
  system §5).
- **FR-004**: Pinned plans are saved per person on the server as the display preference
  `pinnedPlanIds`, like the theme and the collapsed sidebar (D-038: not a business record, not
  written to the activity log, no version — and no version change of any plan). The API always
  answers the list, never null, in the order pinned (insertion order, D-317).
- **FR-005**: A pinned-plans update accepts only ids of the plan form `^P-\d+$`; duplicates are
  dropped, keeping the order they were sent; ids of unknown plans are accepted (FR-007).
- **FR-006**: At most **100** plans can be pinned; a longer list is refused next to the field
  (D-317).
- **FR-007**: A pinned plan that is later deleted, or that the person's filters or visibility do
  not include, simply doesn't show; the stored id stays and is harmless.

### Key Entities *(include if feature involves data)*

- **Preferences** (001/003, table `preferences`): gains `pinnedPlanIds`, the pinned plans' ids in
  the order pinned. No new table, no new record type.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A person who pinned plans finds them in the first positions of the Plans list in
  either layout, from any browser they sign in on, with no other change to the list's order,
  filters, counts or empty states.
- **SC-002**: Pinning or unpinning is one press on the star and shows at once; the save happens in
  the background (research R16), and a failed save says so with Try again.
- **SC-003**: Pinning changes no plan: no activity-log entry and no version change anywhere
  (D-038).
- **SC-004**: Every star is reachable by keyboard and named for the plan it pins, so a screen
  reader hears which plan the star belongs to and whether it pins or unpins.

## Assumptions

- Pin time does not order the list: the stored insertion order only says *what* is pinned; the
  list's own sort orders the pinned group (D-317).
- The cap of 100 needs no paging or management screen; unpinning frees room.
- Other lists (Home, All work) do not read `pinnedPlanIds` in this feature.
