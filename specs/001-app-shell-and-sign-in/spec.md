# Feature Specification: Application Shell and Demo Sign-in

**Feature Branch**: `001-app-shell-and-sign-in`

**Created**: 2026-10-18

**Status**: Draft

**Input**: User description: "Keep one single login page in the start same as ZenReach, then a dashboard same as ZenReach. Authentication and RBAC will be done later or be dummy now. The application must be user-friendly and easily understandable for beginner users."

**References**: Design system [§2 layout, §3 tokens, §5 components, §6 CRUD pattern](../../docs/product/design-system.md) · Domain model [§7 cross-cutting records](../../docs/product/domain-model.md) · Demo users [demo-users-and-data.md](../../docs/product/demo-users-and-data.md) · Decisions D-004, D-005, D-019 · ZenReach spec §6.8.3 (auth modal), §12.2–12.3 (shell, sidebar)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Sign in with a demo account (Priority: P1)

A Shell user opens NorthStar and lands on a single, friendly sign-in page in ZenReach style: the
NorthStar logo on the brand gradient, "Welcome back", e-mail and password fields, and a section
"Explore as a demo person" listing the seeded demo users with their job titles. The page states clearly
that this is a demo sign-in and shows the shared demo password. The user either types a demo e-mail and
the password, or clicks a demo person, and arrives on the Home page.

**Why this priority**: Nothing else can be used without getting in, and every change must be recorded
against a person (the actor).

**Independent Test**: Start NorthStar with the seed data, sign in as each demo person in turn, and confirm
the Home page greets that person by name.

**Acceptance Scenarios**:

1. **Given** the user is not signed in, **When** they open any NorthStar address, **Then** they see the
   sign-in page, and after signing in they are taken to the address they originally asked for (or Home).
2. **Given** the sign-in page, **When** the user enters a demo e-mail (any letter case, surrounding spaces
   ignored) and the demo password and chooses **Sign in**, **Then** they arrive on Home and the sidebar
   shows their avatar and first name.
3. **Given** the sign-in page, **When** the user clicks a person in "Explore as a demo person", **Then**
   they are signed in as that person without typing anything.
4. **Given** the sign-in page, **When** the e-mail is unknown or the password is wrong, **Then** the page
   shows "Incorrect e-mail or password." without saying which one was wrong, and keeps the typed e-mail.
5. **Given** the sign-in page, **When** the user presses **Sign in** with an empty field, **Then** every
   empty field is marked at once ("Enter your e-mail", "Enter the password").
6. **Given** a signed-in user, **When** they reload the page or open NorthStar in a new tab of the same
   browser, **Then** they are still signed in.

---

### User Story 2 - Find my way around the application (Priority: P1)

After signing in the user sees the ZenReach-style layout: a light-grey left sidebar with the NorthStar
logo, the main sections (**Home, Plans, Approvals, All work**), a **MORE** group (**Settings, Recycle
bin, Activity log**), and at the bottom a collapse toggle, their account button, a notification bell and
a light/dark switch. Every page uses the same page header (blue eyebrow, title, one-line description,
primary action on the right). Sections that later features deliver show a friendly "Coming soon" page
that explains what will be there.

**Why this priority**: A predictable frame is what makes every later screen easy to understand.

**Independent Test**: Sign in, visit every navigation item, collapse and expand the sidebar, switch to
dark mode and back, and resize to a phone width.

**Acceptance Scenarios**:

1. **Given** a signed-in user, **When** they choose a navigation item, **Then** that page opens, the item
   is highlighted in the sidebar, and the browser tab title names the page ("Plans · NorthStar").
2. **Given** a section not yet built, **When** the user opens it, **Then** a "Coming soon" page explains
   in one sentence what the section will do, without errors.
3. **Given** the sidebar, **When** the user collapses it, **Then** it shrinks to icons with tooltips, and
   it stays collapsed on their next visit.
4. **Given** a screen narrower than a tablet, **When** the user opens the menu button, **Then** the sidebar
   slides in as a drawer that closes on Escape, on a click outside it, or after choosing an item.
5. **Given** any page, **When** the user switches between light and dark mode, **Then** the whole
   application changes at once and keeps the choice on their next visit; on the first visit it follows
   the computer's setting.
6. **Given** an address that does not exist, **When** the user opens it, **Then** a "Page not found" page
   offers a link back to Home.
7. **Given** any page in this phase, **Then** a thin banner says "Demo environment — sign-in and data are
   for demonstration only".

---

### User Story 3 - See a getting-started Home page (Priority: P1)

Until the full dashboard (feature 009) is built, Home greets the user ("Welcome back, Priya.") and shows
a short **Getting started** card with the steps of the Plan module in order — *Set up Settings · Create a
plan · Add campaigns and initiatives · Add activities and line items · Submit for approval* — each linking
to its page (or to "Coming soon").

**Why this priority**: The sign-in must land somewhere useful, and the steps teach beginners the
order of work from the first minute.

**Independent Test**: Sign in as any demo person and follow each step link.

**Acceptance Scenarios**:

1. **Given** a signed-in user, **When** Home opens, **Then** it shows "Welcome back, <first name>." and
   the five getting-started steps in order, each with a one-line explanation and a link.
2. **Given** feature 009 is delivered later, **Then** this Home content is replaced by the dashboard
   without changing the sign-in flow.

---

### User Story 4 - Manage my session (Priority: P2)

The account button opens a small menu with the user's avatar, name, e-mail and job title, **My profile**
(a placeholder until feature 003) and **Sign out**. Signing out returns to the sign-in page with the
message "You have been signed out."

**Why this priority**: Needed to switch between demo people during demonstrations and testing.

**Independent Test**: Sign in as one person, sign out, sign in as another and confirm the name changes.

**Acceptance Scenarios**:

1. **Given** a signed-in user, **When** they choose **Sign out**, **Then** they return to the sign-in page
   with "You have been signed out." and pressing the browser's Back button does not show the previous
   page's data.
2. **Given** a user whose session is no longer valid, **When** they perform any action, **Then** they are
   sent to the sign-in page with "Please sign in again" and, after signing in, return to where they were.

---

### User Story 5 - Every change is recorded and every delete can be undone (Priority: P1)

Behind every screen of every later feature, NorthStar records **who** did **what** to **which record**
and **when** (the activity log), and a delete never destroys data immediately: the deleted record and
everything under it is kept in the recycle bin and can be brought back. This feature provides that
common behaviour and a shared **Undo** for deletes; feature 011 adds the screens to browse the log and
the recycle bin.

**Why this priority**: Every later feature relies on it; adding it later would mean revisiting them all.

**Independent Test**: Using a simple demonstration record type provided for testing, create, edit and
delete a record; confirm the log holds three entries with the actor and the changed fields, and that
Undo after the delete brings the record back unchanged.

**Acceptance Scenarios**:

1. **Given** any record is created, changed, deleted, restored, moved, copied or changes status, **Then**
   one activity-log entry is written with the actor, the time, the record type, its ID and name, the
   action and, for changes, each changed field with its old and new value.
2. **Given** a record with children is deleted, **Then** the record and all its children are kept
   together in the recycle bin with who deleted them and when, and they disappear from normal views and
   totals.
3. **Given** a delete has just happened, **When** the user chooses **Undo** in the confirmation message
   within its display time, **Then** the record and its children come back exactly as they were, with the
   same IDs, and the restore is logged.
4. **Given** the parent of a deleted record no longer exists, **When** a restore is attempted, **Then** it
   is refused with "Restore '<parent name>' first" and nothing changes.

---

### Edge Cases

- The same demo person signs in in two browsers: both sessions work independently.
- The seed data has not been loaded: the sign-in page says "No demo users found — load the demo data
  first" instead of failing.
- NorthStar's server is not reachable: pages show "NorthStar can't reach its server right now. Check that it is
  running and try again." with a **Try again** button, never a technical error.
- A very long name or job title is shortened with an ellipsis in the sidebar and shown in full in a
  tooltip.
- Two activity-log writes for the same record at the same moment are both kept, in order.

## Requirements *(mandatory)*

### Functional Requirements

**Sign-in and session**

- **FR-001**: NorthStar MUST show a single sign-in page in the ZenReach style (design system §2 and
  ZenReach spec §6.8.3): logo on the brand gradient, "Welcome back", e-mail, password with a show/hide
  control, **Sign in**, and "Explore as a demo person" listing every active demo person with avatar,
  name and job title.
- **FR-002**: The sign-in page MUST state "Demo sign-in — Shell single sign-on comes later" and show the
  shared demo password (decision D-004).
- **FR-003**: NorthStar MUST sign a user in when the e-mail matches an active demo person (ignoring letter
  case and surrounding spaces) and the password equals the demo password, or when a demo person is chosen
  from the list.
- **FR-004**: A failed sign-in MUST show "Incorrect e-mail or password." for both unknown e-mails and wrong
  passwords; empty fields MUST all be reported in one attempt.
- **FR-005**: Every page except the sign-in and "not found" pages MUST require a signed-in user; an
  unsigned visitor MUST be sent to the sign-in page and returned to the requested page afterwards.
- **FR-006**: A session MUST survive page reloads and new tabs in the same browser, MUST end on **Sign
  out**, and MUST expire after 12 hours without activity (then the user is asked to sign in again and
  returned to where they were).
- **FR-007**: Every action a signed-in user performs MUST be attributed to that person (the actor), and
  the actor MUST be available to every business operation.
- **FR-008**: In this phase NorthStar MUST NOT restrict any signed-in user's actions by role or team
  (decision D-004); no permission messages appear anywhere.
- **FR-009**: Demo sign-in MUST be available only when NorthStar runs in a local or development
  environment.

**Layout and navigation**

- **FR-010**: Every signed-in page MUST use the layout of design system §2: left sidebar (logo; Home,
  Plans, Approvals, All work; MORE: Settings, Recycle bin, Activity log; footer with collapse toggle,
  account button, notification bell, light/dark switch), page header pattern, content area of at most
  1120 px.
- **FR-011**: The sidebar MUST highlight the current section, collapse to an icon rail with tooltips,
  remember its collapsed state per user, and become a slide-in drawer on narrow screens.
- **FR-012**: The account button MUST open a menu with avatar, name, e-mail, job title, **My profile** and
  **Sign out**.
- **FR-013**: The notification bell MUST be present and show "You're all caught up." until feature 008
  delivers notifications.
- **FR-014**: Light and dark modes MUST be available, follow the computer's setting on first use, and be
  remembered per user.
- **FR-015**: Sections delivered by later features MUST show a "Coming soon" page with one sentence on
  what they will do; unknown addresses MUST show "Page not found" with a link to Home.
- **FR-016**: Every page MUST show the demo-environment banner during this phase.
- **FR-017**: Each page MUST set the browser tab title to "<Page> · NorthStar".

**Home (getting started)**

- **FR-018**: Home MUST greet the user by first name and show the five getting-started steps of the Plan
  module, in order, each with a one-line explanation and a link to its page.

**Shared building blocks for later features**

- **FR-019**: NorthStar MUST provide the shared visual building blocks listed in design system §5 (page
  header, summary card, content card, buttons, status chip, completeness chip, info tip, person avatar,
  side panel, confirmation dialog, toast with Undo, empty state, loading placeholder, form error summary),
  styled with the tokens of design system §3, so later features reuse them instead of creating their own.
- **FR-020**: NorthStar MUST present every error from its server in plain language: field problems next
  to the fields and in a summary at the top (all at once), and other failures as a friendly message with
  **Try again**; technical details MUST never be shown to users.
- **FR-021**: Dates MUST be shown as `DD MMM YYYY` (e.g. 12 Mar 2026) and money with its currency, using
  the formats in design system §4, until a user's own preference exists (feature 003).

**Activity log and recycle bin core**

- **FR-022**: NorthStar MUST write one activity-log entry for every create, change, status change, move,
  copy, delete and restore of any business record (planning records and Settings records), containing
  the actor, the time (UTC), the record type, its readable ID (planning records) or name (Settings
  records), the action and, for changes, each changed field with old and new values.
- **FR-023**: Activity-log entries MUST be read-only and kept for at least the duration of the phase.
- **FR-024**: Deleting a business record MUST move it, together with every record under it, into the
  recycle bin with who deleted it and when; deleted records MUST disappear from all views, searches and
  totals.
- **FR-025**: A deleted record MUST be restorable with its original readable IDs, provided its parent
  exists; otherwise the restore MUST be refused naming the parent to restore first (the recycle bin page
  of feature 011 additionally offers to restore both together). Status resets on later restores follow
  domain model R14.
- **FR-026**: After any delete, the confirmation message MUST offer **Undo** for at least 10 seconds;
  Undo MUST restore exactly what was deleted, statuses included (domain model R14).
- **FR-027**: NorthStar MUST give every planning record (plan, campaign, initiative, activity, line item)
  a permanent readable ID with a prefix per level (P-, C-, I-, A-, L-, decision D-019); IDs MUST never be
  reused, even after deletion. Settings records are identified by their type and name.

**Editing safely**

- **FR-029**: When two people edit the same record, a save based on an out-of-date copy MUST be refused
  with a message naming who changed the record and when ("Not saved — Daniel Okoro changed this activity
  at 14:32"), keeping the user's typed values on screen so they can review and try again; nothing is
  silently overwritten. Every feature relies on this shared behaviour.

**Demo data**

- **FR-028**: NorthStar MUST provide a way to load the demo people of
  [demo-users-and-data.md](../../docs/product/demo-users-and-data.md) into an empty data store, and the
  sign-in page MUST explain what to do when no demo people exist.

### Key Entities *(include if feature involves data)*

- **Demo person**: someone who can sign in — name, e-mail, job title, active flag. (Teams and profile
  settings arrive in feature 003.)
- **Session**: a signed-in person in one browser, with when it started and when it was last used.
- **Activity-log entry**: actor, time, record type, record ID and name, action, changed fields (old → new).
- **Recycle-bin entry**: the deleted record with everything that was under it, who deleted it, when, and
  the parent it belonged to.
- **User preferences**: sidebar collapsed, light/dark mode (extended in feature 003).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A first-time user signs in as a demo person in at most 2 steps (one click on a person, or
  e-mail + password + Sign in) and reaches Home in under 10 seconds.
- **SC-002**: In a test with at least 5 first-time users, every user finds any navigation item asked for
  within 2 attempts and can say what each sidebar section is for.
- **SC-003**: 100 % of navigation items open a page (real or "Coming soon") without an error.
- **SC-004**: 100 % of business changes made through any feature produce exactly one activity-log entry
  naming the correct person.
- **SC-005**: 100 % of deletes can be undone within 10 seconds with nothing lost.
- **SC-006**: Every screen of this feature passes an automated WCAG 2.1 AA check with no failures and can
  be used with the keyboard alone.
- **SC-007**: The layout is usable without horizontal scrolling at 1280 × 720 and on a 390 px-wide phone.

## Assumptions

- Demo sign-in is temporary (ADR-0005); Microsoft Entra ID replaces only the sign-in page and the way the
  actor is identified.
- The demo password is shown openly because all data is fictitious; NorthStar is not exposed outside the
  developer's computer in this phase.
- The 12-hour inactivity expiry is a reasonable default for a demo environment.
- The Approvals, All work, Settings, Recycle bin and Activity log pages are delivered by features 008,
  010, 002/003, 011 and 011; until then they show "Coming soon".
- English is the only language in this phase.
