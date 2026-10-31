# Feature Specification: Search the whole hierarchy

**Feature Branch**: `feature/039-search-whole-hierarchy`

**Created**: 2026-10-31

**Status**: Built

**Input**: User description: "In search functionality, I should be able to type and search for all the items matching
in the entire hierarchy — so if I search anything in the search box, I should get matching results for that string
till line items, not just till what I can view."

Amends [004 FR-013, FR-014](../004-plans/spec.md) (the plan tree's search) and [004 FR-003](../004-plans/spec.md)
(the Plans list's search). All work's search already matches line items (010 FR-006) and is unchanged. Decision
D-366.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - The plan tree's search finds line items (Priority: P1)

A marketer on a plan page types a cost's name, such as "Stand build", or its ID, such as "L-1006", in "Search name
or ID in this plan". The tree shows only down to activities, so today nothing is found. Now the tree shows the path
to that line item — campaign, initiative, activity — opened, with the line item as a row under its activity, and
the row opens the line item.

**Why this priority**: Line items are the level people plan their costs on; a search that stops at the
activities hides exactly the records people look for most.

**Independent Test**: On the demo plan *FY2026 Lubricants Europe*, search "Venue hire": the tree shows the
campaign, initiative and activity above it, opened, and a "Line item" row "Venue hire" with its ID; Enter on it
opens the line item page.

**Acceptance Scenarios**:

1. **Given** a plan whose activity holds the line item "Venue hire", **When** the user types "venue" in the
   plan tree's search, **Then** the tree shows the plan, campaign, initiative and activity above it, opened, and a
   row "Venue hire" tagged LINE ITEM with its ID, whose name leads to the line item page.
2. **Given** the same plan, **When** the user types the line item's ID, **Then** the same row is shown.
3. **Given** a search that matches an activity and one of its line items, **When** the results show, **Then**
   the activity row shows and the matching line item row is under it; line items that do not match are not shown.
4. **Given** the search is cleared, **When** the tree shows again, **Then** it shows down to activities as
   before, without line item rows.
5. **Given** a text that matches nothing at any level, **When** the search has answered, **Then** the tree says
   "Nothing in this plan matches '<text>'".

---

### User Story 2 - The Plans list's search finds plans by what is inside them (Priority: P2)

On the Plans page a marketer types the name or ID of a campaign, initiative, activity or line item. The list shows
the plans that hold a match, and each card that matched only through its contents says where: "Found in line item
'Venue hire' (L-…)", with "and 2 more" when more records match.

**Why this priority**: People remember the campaign or cost, not the plan it sits in; finding the plan from any
record saves opening plans one by one.

**Independent Test**: On the Plans page search "Venue hire": *FY2026 Lubricants Europe* is listed with "Found in
line item 'Venue hire'"; clear the search and every plan shows again.

**Acceptance Scenarios**:

1. **Given** a plan whose name does not contain "venue" but which holds the line item "Venue hire", **When** the
   user searches "venue", **Then** the plan is listed and its card says "Found in line item 'Venue hire'
   (L-…)".
2. **Given** a plan holding three matching records, **When** the user searches, **Then** its card names the
   first, top-down (campaign, initiative, activity, line item, then by name), and adds "and 2 more".
3. **Given** a plan whose own name or ID matches, **When** the user searches, **Then** it is listed as before,
   without a "Found in" line.
4. **Given** a matching record in a plan the person may not see, or in the recycle bin, **When** the user
   searches, **Then** that plan is not listed because of it.
5. **Given** the status, year, plan type or BCPIR filters, **When** the user searches, **Then** they still apply
   to every plan listed.

### Edge Cases

- A text of spaces only is no search: every plan and the whole tree show.
- Letter case never matters; the text matches anywhere in a name or ID ("stand" finds "Stand build").
- A search over 200 characters is refused with the field's message, as today.
- While the line items are being searched, the rows of the levels above already filter; the line item rows
  join them when the answer arrives, and their path opens.
- When the line items cannot be searched (the API fails), the other rows still filter and the tree says
  "Line items couldn't be searched. Try again." under the search.
- A plan the person may not see answers 404 to the line-item search, like the plan page.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The plan tree's search MUST match line items too, by name or ID, ignoring letter case, anywhere in
  the text — the rule of every other row (004 FR-014).
- **FR-002**: A matching line item MUST be shown as a row under its activity, tagged LINE ITEM, with its name and
  ID, its name leading to the line item page; the path above it MUST open.
- **FR-003**: Line item rows MUST be shown only while a search is typed; without one the tree is unchanged.
- **FR-004**: The line items MUST be searched by the API 300 ms after the last key; the other rows MUST keep
  filtering at once. A failed search MUST leave the other rows filtered and say "Line items couldn't be
  searched. Try again.".
- **FR-005**: The API MUST answer `GET /api/v1/plans/{planId}/search?q=<text>` with the campaigns, initiatives,
  activities and line items of the plan whose name or ID contains the text, each with its level, ID, name and
  parent's ID, top-down then by name; only for a plan the person may see (404 otherwise); `q` of 1–200
  characters after trimming.
- **FR-006**: The Plans list's search MUST list a plan when its own name or ID matches, or when any campaign,
  initiative, activity or line item in it matches by name or ID.
- **FR-007**: A plan listed only because of its contents MUST carry the first match and the number of matches;
  its card MUST say "Found in <level> '<name>' (<ID>)" and, with more, "and <n> more".
- **FR-008**: Only records of plans the person may see MUST match; records in the recycle bin never match.
- **FR-009**: The other filters of the Plans list MUST still apply, and the summary cards are unchanged.

### Key Entities

- **Search match**: a record below a plan that contains the text: level, ID, name, the ID of its parent, and the
  plan's ID.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In a plan with 500 activities and 2,000 line items, the matching line items show within 1 second of
  the last key.
- **SC-002**: With 200 plans, the Plans list's search still answers within 1 second (004 SC-009), the records of
  every plan read in four statements whatever their number.
- **SC-003**: A line item found by its name or ID in the plan tree is opened in two actions: type, then Enter or
  click on its row.

## Assumptions

- "The search box" is the plan tree's "Search name or ID in this plan" and the Plans list's "Search name or ID";
  All work already searches line items (010 FR-006).
- Matching stays on name and ID, as these two searches do today; descriptions and other fields are All work's
  search (010 FR-006).
- The line item row shows only its level, name and ID; its figures stay on the line item page and in All work.
