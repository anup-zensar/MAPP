# Feature Specification: People and Teams

**Feature Branch**: `003-people-and-teams`

**Created**: 2026-10-18

**Status**: Draft

**Input**: User description: "People directory, teams and memberships, person picker, My profile"

**References**: Domain model [§4 display currency, §5 rule R1, §6 supporting records, §8 visibility later](../../docs/product/domain-model.md) · Field catalogue [§1 #4–#5 plan teams and owner](../../docs/product/field-catalogue.md) · Design system [§5 person picker and avatar, §6 CRUD pattern, §7 forms, §8 date format, §10 usability targets](../../docs/product/design-system.md) · Demo data [demo users and teams](../../docs/product/demo-users-and-data.md) · Features 001 (sign-in, account menu), 002 (Settings page, currencies) · Decisions D-004, D-010, Q-006 · Hive9 functional analysis E6, F5, F6

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Manage the people directory (Priority: P1)

The administrator opens **Settings › People**: everyone with avatar, name, e-mail, job title, teams and status
(**Active** / **Inactive**), a search box "Search name, e-mail or job title" and **New person**. A person has a
name, an e-mail (their sign-in name), a job title and teams. A new person can sign in straight away with the
demo password (decision D-004). People are never deleted, because the history names them; they are
**deactivated** instead.

**Why this priority**: Owners, approvers and the actor of every change are people.

**Independent Test**: Create a person and sign in as them; from another account deactivate them, check they
cannot sign in and their records show "(inactive)"; reactivate them.

**Acceptance Scenarios**:

1. **Given** the People list, **When** the user chooses **New person**, enters "Jane Doe",
   "jane.doe@shell.com", job title "Marketer", team "Lubricants Europe" and saves, **Then** she is listed
   as Active, the toast "Person 'Jane Doe' created — View" appears, and she can sign in with the demo password.
2. **Given** an empty name and an e-mail without "@", **When** the user saves, **Then** "Enter a name" and "Enter
   a valid e-mail address" are shown together; an e-mail already used, in any letter case, by an inactive person
   shows "jane.doe@shell.com already belongs to Jane Doe (inactive) — reactivate her instead."
3. **Given** a person, **When** the user selects them, **Then** the details show avatar, name, e-mail, job title,
   status, teams (as links), what they own or approve ("Owner of 2 plans and 5 activities · approver in 1
   approval workflow") and who created and last changed the record.
4. **Given** a person, **When** the user edits the job title and e-mail and saves, **Then** every screen naming
   them shows the change, their next sign-in uses the new e-mail, and the log shows old and new values.
5. **Given** Priya Raman owns records, **When** the user chooses **Deactivate**, **Then** a dialog says "Deactivate
   Priya Raman? She won't be able to sign in or be chosen in people lists. She owns 2 plans and 5 activities and
   approves in 1 workflow; these keep her name, shown as 'Priya Raman (inactive)'." with **Cancel** focused and
   **Deactivate person**; afterwards she is listed as Inactive and her open sessions end at her next action.
6. **Given** the signed-in user's own record, **Then** **Deactivate** stays visible and explains "You can't
   deactivate yourself — ask someone else to do it."
7. **Given** an inactive person, **When** the user chooses **Reactivate**, **Then** they can sign in again, are
   offered in people lists again and still belong to their teams.
8. **Given** any person, **When** the user opens **⋯ Actions**, **Then** there is no Delete; **Deactivate** sits in
   its place with the tip "People are never deleted, so the history keeps their name."

---

### User Story 2 - Manage teams and memberships (Priority: P1)

**Settings › Teams** lists each team with its description, member count and avatars, the number of plans using
it and its status. Plans are assigned to teams and a plan's owner must belong to one of them (R1).
Membership can be changed from the team and from the person.

**Why this priority**: Plans cannot be published without teams and an owner from those teams.

**Independent Test**: Create a team with two members, add a third from the person's form, remove a member who
owns a plan, try to delete a team used by a plan, and delete an unused team and undo.

**Acceptance Scenarios**:

1. **Given** the Teams list, **When** the user chooses **New team**, enters "Lubricants Asia", a description and
   two members picked with the person picker, and saves, **Then** it is listed as Active with 2 members and the
   toast "Team 'Lubricants Asia' created — View" appears.
2. **Given** a person's form, **When** the user adds the team "Lubricants Asia" and saves, **Then** the team's
   member list shows that person at once — and removing a member on the team shows on the person.
3. **Given** a team, **When** the user selects it, **Then** the details show its description, members (avatar,
   name, job title; inactive members labelled "(inactive)") and "Used by 1 plan" with a link.
4. **Given** a team, **When** the user renames it to a name another team has (any letter case), **Then** "A team
   called 'Mobility China' already exists." is shown; a unique new name appears on every plan using the team.
5. **Given** Priya Raman owns "FY2026 Lubricants Europe" through the team "Lubricants Europe", **When** the user
   removes her from that team, **Then** before saving MAPP says "Priya Raman owns 1 plan that uses this
   team: FY2026 Lubricants Europe. She will no longer belong to its teams, so its owner will need attention."
   and lets the user save or keep editing.
6. **Given** a team used by plans (including plans in the recycle bin), **When** the user chooses **Delete**,
   **Then** the dialog lists those plans and offers **Set team inactive**; an inactive team is not offered for new
   plans, and plans using it show "Lubricants Europe (inactive)".
7. **Given** a team no plan uses, **When** the user confirms **Delete team**, **Then** the team and its memberships
   move to the recycle bin, and **Undo** brings both back.

---

### User Story 3 - Pick a person anywhere (Priority: P1)

Every field that holds a person — owners, approvers, team members — uses one **person picker** (design system
§5). The user types part of a name, e-mail or job title; each result shows the avatar, the full name and the job
title, never cut off. The signed-in user comes first, marked "(you)". For a plan's owner the picker works in the
mode **Only members of the selected teams** (R1).

**Why this priority**: Choosing people is part of almost every create form in features 004–008.

**Independent Test**: In a team's member field search by job title and by e-mail; in the team-restricted mode
choose a team and check that only its active members are offered.

**Acceptance Scenarios**:

1. **Given** the demo people, **When** the user types "finance", **Then** Sara Novak (Local finance advisor) is
   offered, matched on her job title; typing "okoro@" offers Daniel Okoro, matched on his e-mail.
2. **Given** a person with a long name and job title, **Then** both are shown in full, wrapping onto a second line;
   **Given** 300 people, **Then** the list shows "300 people", narrows as the user types, and every match can be
   reached — none is silently left out.
3. **Given** Priya Raman is inactive, **Then** no picker offers her; a field already holding her shows "Priya
   Raman (inactive)" with the tip "Inactive people can't be chosen again. Keep her or choose someone else."
4. **Given** the team-restricted mode with no team chosen, **Then** the picker is disabled with "Choose teams
   first — the owner must belong to one of the plan's teams."
5. **Given** the team-restricted mode with "Lubricants Europe", **Then** only its active members are offered, each
   with the team that qualifies them; with no match it says "No members of the selected teams match. Add people
   to these teams in Settings › Teams."
6. **Given** the team-restricted mode and a current value who is not a member of the chosen teams, **Then** the
   value is kept and flagged "Not a member of the plan's teams".
7. **Given** a multiple-person field, **Then** chosen people show as chips with avatar and name, and the whole
   picker works with the keyboard (arrows, Enter, Backspace to remove, Escape to close).

---

### User Story 4 - My profile (Priority: P2)

From the account menu (feature 001), **My profile** shows the user's avatar, name, job title, e-mail, two
preferences — **display currency** and **date format** — and **My teams** as a read-only list.

**Why this priority**: Users need to see amounts in their currency and dates in their format; the demo works
with the defaults.

**Independent Test**: As Priya choose EUR and DD/MM/YYYY, check amounts and dates on another page, then sign
in from another browser and see the same preferences.

**Acceptance Scenarios**:

1. **Given** a signed-in user, **When** they open My profile, **Then** they see name, job title, e-mail with a lock
   and "Your e-mail is your sign-in name; it is changed in Settings › People.", display currency USD, date format
   "12 Mar 2026" and My teams with a link to Settings › Teams.
2. **Given** My profile, **When** the user chooses **Edit**, changes their name and saves, **Then** the avatar
   initials and the sidebar name update at once and the change is logged; an empty name shows "Enter a name".
3. **Given** display currency EUR, **Then** every amount on screen is converted with the current rate (feature
   002) and labelled "EUR"; stored values and amounts typed in other currencies do not change.
4. **Given** date format "12/03/2026 (DD/MM/YYYY)", **Then** every date shown and typed uses that format.
5. **Given** a user in no team, **Then** My teams says "You're not in any team yet, so you can't own a plan.
   Teams are managed in Settings › Teams."

---

### Edge Cases

- Two people share a name: the picker and lists show job title and e-mail to tell them apart.
- E-mail matching ignores letter case and surrounding spaces; e-mails are stored in lower case.
- A person deactivated while signed in is sent to the sign-in page at their next action (feature 001).
- A person's display currency is later set inactive (feature 002): they keep it; nobody else can choose it.
- A team with 250 members or a person in 85 teams (Hive9's scale): member and team lists are searchable.
- A one-word name such as "Cher" gets the initials "CH".

## Requirements *(mandatory)*

### Functional Requirements

**People**

- **FR-001**: Settings MUST have a **People** section following the standard CRUD pattern (design system §6)
  with avatar, name, e-mail, job title, teams and status (**Active** green / **Inactive** grey chips), search on
  name, e-mail and job title, and an Active / Inactive / All filter.
- **FR-002**: A person MUST have a name (≤ 120 characters) and an e-mail (valid format, ≤ 254 characters, unique
  across all people including inactive ones, ignoring letter case and surrounding spaces), and MAY have a job
  title (≤ 120) and teams. A duplicate e-mail message MUST name the person who has it.
- **FR-003**: Every active person MUST be able to sign in through the demo sign-in of feature 001.
- **FR-004**: A person's details MUST show their teams and counts of what they own or approve, per record type,
  as features 004–008 deliver those records.
- **FR-005**: **Deactivate** MUST first state what the person owns or approves. An inactive person MUST NOT be able
  to sign in (open sessions end at their next action) and MUST NOT be offered in any picker; every record that
  names them keeps them, shown as "<name> (inactive)"; their memberships are kept.
- **FR-006**: Users MUST NOT be able to deactivate themselves; the action MUST stay visible and say why.
- **FR-007**: **Reactivate** MUST restore sign-in and picker presence with memberships intact.
- **FR-008**: People MUST never be deleted; **Deactivate** takes Delete's place with the reason.
- **FR-009**: Avatars MUST show initials (first letters of the first two words, or the first two letters of a
  one-word name) on the brand gradient (design system §5).

**Teams**

- **FR-010**: Settings MUST have a **Teams** section following the standard CRUD pattern with name, description,
  member count and avatars, number of plans using it, and status.
- **FR-011**: A team MUST have a name (≤ 80 characters, unique ignoring letter case and surrounding spaces) and
  MAY have a description (≤ 300) and any number of active people as members.
- **FR-012**: Membership MUST be editable from the team and from the person, and both MUST show the same
  membership immediately.
- **FR-013**: Removing a member who owns plans that use the team MUST first list those plans and say their owner
  will need attention (R1); the change is allowed.
- **FR-014**: A team used by any plan, including plans in the recycle bin, MUST NOT be deleted: the dialog lists
  the plans and offers **Set team inactive**. Inactive teams MUST NOT be offered for new plan teams; plans keep
  them, labelled "(inactive)", and their members still count for R1. **Reactivate** reverses this.
- **FR-015**: An unused team MUST be deletable to the recycle bin together with its memberships; **Undo**
  restores both (feature 001).

**Person picker**

- **FR-016**: Every person field in MAPP MUST use one person picker, in single or multiple mode.
- **FR-017**: The picker MUST match any part of the name, e-mail or job title, ignoring letter case; show avatar,
  full name and job title (or e-mail when there is no title) without truncating; show the number of matches and
  let every match be reached; and list the signed-in user first, marked "(you)", when they qualify.
- **FR-018**: The picker MUST offer only active people; an inactive person already in a field MUST be shown with
  "(inactive)" and an explanation, and kept unless the user replaces them.
- **FR-019**: In **Only members of the selected teams** mode the picker MUST offer only active members of the
  given teams, showing the qualifying team; MUST be disabled with a hint while no team is chosen; and MUST flag a
  current value who is not a member. What the owning form does with that flag is defined by its feature (004).
- **FR-020**: When nothing matches, the picker MUST say so and suggest the next step; it MUST work with the
  keyboard alone and announce the number of matches to screen readers.

**My profile**

- **FR-021**: **My profile** in the account menu MUST replace feature 001's placeholder and show avatar, name, job
  title, e-mail (read-only, with the reason), display currency, date format and My teams (read-only, with a link
  to Settings › Teams), using the details-then-**Edit** pattern.
- **FR-022**: Name and job title MUST follow FR-002's rules; a change MUST appear everywhere the person is named.
- **FR-023**: The display currency MUST be chosen from the active currencies of feature 002 (default USD); every
  amount on screen MUST then be converted with the current rate and labelled with its currency (domain model §4,
  decision D-010); stored values and money inputs keep their own currency.
- **FR-024**: The date format MUST be one of DD MMM YYYY (default, "12 Mar 2026"), DD/MM/YYYY, MM/DD/YYYY or
  YYYY-MM-DD, each shown with an example, and MUST be used for every date shown and typed (replacing feature
  001's default, FR-021).
- **FR-025**: Preferences MUST belong to the person and apply in any browser they sign in from.

**Visibility, permissions, log and seed**

- **FR-026**: The signed-in person's teams MUST be available to the "My teams" filters of other features (004,
  010). No plan or record is hidden from anyone in this phase (Q-006, domain model §8); Shell's rule that plans
  are visible only to members of their teams is enforced when roles arrive.
- **FR-027**: Every signed-in user MUST be able to manage people and teams (decision D-004). Shell's rules — only
  administrators manage people and teams, and per-member "can edit team content" and "can approve" flags — are
  enforced when roles arrive.
- **FR-028**: Every create, change, membership change, deactivation, reactivation, delete and restore of a person
  or team MUST be written to the activity log (feature 001).
- **FR-029**: The demo data MUST include the teams and memberships of demo-users-and-data.md (Lubricants Europe,
  Mobility China, Global Marketing Ops).

### Key Entities *(include if feature involves data)*

- **Person**: extends feature 001's demo person — name, e-mail, job title, active, teams, preferences.
- **Team**: name, description, active.
- **Team membership**: one person in one team; a person may be in many teams.
- **Person preferences**: display currency and date format (here), sidebar and light/dark mode (feature 001).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: At least 90 % of first-time users create, edit, deactivate and reactivate a person, and create a
  team with members, without help, each within 2 minutes (design system §10).
- **SC-002**: A user finds and chooses any person in a picker in at most 2 steps (type, select); with 1,000
  people, matches appear while typing in under 1 second.
- **SC-003**: 100 % of pickers never offer an inactive person, and 100 % of fields holding one show "(inactive)".
- **SC-004**: 100 % of sign-in attempts by inactive people are refused; 100 % succeed again after reactivation.
- **SC-005**: 100 % of team-restricted pickers offer only active members of the selected teams.
- **SC-006**: After a display-currency or date-format change, 100 % of amounts and dates on the next page use it,
  and every amount is labelled with its currency.
- **SC-007**: 100 % of form problems are shown in a single save attempt.
- **SC-008**: Every screen of this feature passes an automated WCAG 2.2 AA check and works with the keyboard alone.

## Assumptions

- People and Teams are sections of the Settings page of feature 002; if 003 is built first, Settings shows only
  these two sections.
- A person has one name field (Hive9 uses first and last name); the e-mail is the sign-in name.
- Display currencies come from feature 002; until it exists, only USD is offered.
- Importing people (Shell has about 250 Hive9 users) is not in this phase; the demo needs only a few people.
- Personal data is limited to name, e-mail and job title (constitution VI).
- English is the only language in this phase.
