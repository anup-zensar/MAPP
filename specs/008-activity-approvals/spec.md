# Feature Specification: Activity Approvals

**Feature Branch**: `008-activity-approvals`

**Created**: 2026-10-18

**Status**: Draft

**Input**: User description: "Approval workflows (Settings), submit / approve / decline / re-submit, approvals inbox, notifications"

**References**: Domain model [§2.3 activity statuses, §2.4 line item, §2.5 re-approval, §3 required fields, §4.3 unassigned balance, §4.4 spreads, §5 R10, R13, R14, §6 supporting records, §7 notification, §8 later roles](../../docs/product/domain-model.md) · Field catalogue [§4 activity, §5 line item](../../docs/product/field-catalogue.md) · Design system [§2 layout, §4 status colours, §5 components, §6 CRUD pattern and locked states, §10 usability targets](../../docs/product/design-system.md) · Demo data [demo-users-and-data.md](../../docs/product/demo-users-and-data.md) · Decisions D-004, D-009, D-012, D-013, D-016, D-020 · Open question Q-002 · Hive9 functional analysis §B9, §C6, §E6.1.2

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Submit an activity for approval (Priority: P1)

Priya chooses **Submit for approval** on the activity "Launch film". If anything is missing, MAPP
lists every problem at once with links to fix them; otherwise a short confirmation shows the approval
workflow and who approves each level, and the activity becomes **Submitted**, waiting for level 1.

**Why this priority**: The approved activity is the gate for SWBS requests and purchasing; Hive9 showed
one error per save.

**Independent Test**: Submit an incomplete seeded Mobility China activity, see every problem at once, fix
them, submit again and confirm it is Submitted and waiting for Carlos Mendes.

**Acceptance Scenarios**:

1. **Given** a Created activity with nothing missing, **When** the user chooses **Submit for approval**
   and confirms, **Then** it becomes Submitted, round 1 waits at level 1, and the approval history and
   the activity log show "Submitted" with the user, date and time.
2. **Given** an activity with problems, **When** the user chooses **Submit for approval**, **Then**
   nothing changes and one list shows every problem, e.g. "Owner — needed to submit", "Line item
   'Agency fee' (L-1003): GL code — needed to submit", "Country / OU split totals 90.00 % — it must
   total 100.00 %", "Line items exceed the activity's planned cost by $1,200.00", each with a link.
3. **Given** the confirmation, **Then** it names the workflow, lists each level with its approvers
   ("Level 1 · Carlos Mendes", "Level 2 · Alex Morgan") and accepts an optional comment.

---

### User Story 2 - Approve or decline, level by level (Priority: P1)

The activity's **Approval** section shows the chain of levels, approvers and states. Approving a level
moves the activity on; approving the last level makes it **Approved**. Declining (comment required)
makes it **Declined**; re-submitting starts a new round at level 1. Anyone can act (D-020); a decision
by someone who is not an approver of the level is recorded "on behalf of" the designated approver(s).

**Why this priority**: Without decisions, submitting has no purpose.

**Independent Test**: Approve the seeded two-level Mobility activity at level 2 as Alex Morgan; it is
Approved and the history shows both levels' decisions.

**Acceptance Scenarios**:

1. **Given** an activity waiting at level 1 of 2, **When** a level-1 approver approves, **Then** it stays
   Submitted, the chain shows "Approved by Carlos Mendes on 12 Mar 2026" and level 2 is waiting.
2. **Given** an activity at its last level with two approvers, **When** either approves, **Then** it
   becomes Approved, shown as In progress from its start date and Complete after its end date.
3. **Given** the decline dialog, **When** the user confirms without a comment, **Then** nothing changes
   and it says "Add a comment explaining why the activity is declined."; with a comment the activity
   becomes Declined and the comment heads its Approval section.
4. **Given** a fixed Declined activity, **When** the user chooses **Re-submit for approval**, **Then** the
   same check runs and round 2 starts at level 1 with the workflow's levels as they are now.
5. **Given** Alex Morgan, not an approver of level 1, **When** he approves, **Then** the dialog says "You
   are not an approver of level 1. Your decision will be recorded as made on behalf of Daniel Okoro."
   and the history shows "Alex Morgan on behalf of Daniel Okoro" with round, level, date, time, comment.

---

### User Story 3 - Work through my approvals (Priority: P1)

The **Approvals** page shows summary cards and three tabs — **Waiting for me**, **All open**, **Decided
by me**. A row opens a preview with the activity's key facts, line items and approval chain, so the
approver decides without leaving the page.

**Why this priority**: Approvers handle many activities; Hive9 needed about eight steps per approval.

**Independent Test**: As Daniel Okoro, approve one waiting activity and decline another from the
preview, then find both under **Decided by me**.

**Acceptance Scenarios**:

1. **Given** Daniel is signed in, **When** he opens Approvals, **Then** **Waiting for me** lists the
   Submitted activities whose current level names him, oldest first, and the sidebar shows that count.
2. **Given** a row, **When** he selects it, chooses **Approve** and confirms, **Then** the row leaves the
   tab, a message names the outcome, and the preview moves to the next waiting activity.
3. **Given** **All open** or **Decided by me**, **Then** they list every Submitted activity with the level
   it waits at, or every activity he decided with his latest decision, its date and the current status.
4. **Given** search and filters, **When** he types "A-1042" or filters by plan, workflow, owner or
   submitted date, **Then** only matching rows remain with filter chips and a count ("3 of 12").

---

### User Story 4 - Set up approval workflows (Priority: P1)

Alex, the marketing administrator, maintains **Settings › Approval workflows** with the standard CRUD
pattern: name, description and ordered levels with one or more approvers each; details show where a
workflow is used; a workflow in use is set inactive instead of deleted.

**Why this priority**: Every activity needs a workflow before it can be submitted.

**Independent Test**: Create a two-level workflow, submit an activity with it, edit the workflow; the
open round keeps its levels while the next submission uses the new ones.

**Acceptance Scenarios**:

1. **Given** the seed data, **Then** the list shows "Lubricants EU approval" and "China BC approval
   workflow" with their levels ("Level 1 · Carlos Mendes → Level 2 · Alex Morgan"), status and "Used by
   N activities"; the details list those activities with links.
2. **Given** **New approval workflow**, **When** Alex saves with no name and a level without approvers,
   **Then** both problems are shown at once; once fixed, the workflow is created and a message names it.
3. **Given** a workflow with open rounds, **When** Alex edits it, **Then** the panel says "Changes apply
   to activities submitted from now on. 2 activities waiting for approval keep the levels they were
   submitted with." and, after saving, those rounds are unchanged.
4. **Given** **Set inactive**, **When** Alex confirms, **Then** the workflow is no longer offered on
   activities, activities using it show "(inactive)", open rounds continue, and **Set active** undoes it.

---

### User Story 5 - Keep approval meaningful when things change (Priority: P1)

A change listed in domain model §2.5 on a Submitted or Approved activity ends its approval and returns
it to **Created** once the user confirms. The owner can also **recall** a submitted activity.

**Why this priority**: An approval must always match the figures that were approved.

**Independent Test**: Change the planned cost of a Submitted and an Approved seed activity and recall a
third; all three are Created, the history explains why, and the right people are told.

**Acceptance Scenarios**:

1. **Given** a Submitted activity, **When** the user changes its planned cost and confirms "This activity
   will need approval again", **Then** the round ends as "Approval restarted", the activity is Created,
   the history shows "Planned cost changed from $50,000.00 to $65,000.00", and the waiting approvers and
   the owner are notified (except the person who made the change).
2. **Given** an Approved activity shown as In progress, **When** a line item's currency is changed and
   confirmed, **Then** it returns to Created and its line items show "Waiting for activity approval".
   *Replaced by feature 034 (D-322): a line item's currency change keeps the activity Approved.*
3. **Given** a Submitted activity, **When** the owner chooses **Recall submission**, optionally gives a
   reason and confirms, **Then** it becomes Created, the round ends as "Recalled" with the reason, and it
   leaves the approvers' **Waiting for me** tab.

---

### User Story 6 - Be told when something needs me (Priority: P2)

The bell from feature 001 now works: approvers learn that an activity waits for them and owners learn
what happened to theirs, with an unread count, **Mark all as read** and a link to the activity.

**Why this priority**: The approvals page already shows everything; notifications make it faster.

**Independent Test**: Submit as Priya, decline as Daniel, check both bells; nobody is told about their
own action.

**Acceptance Scenarios**:

1. **Given** Priya submits, **Then** within one minute Daniel's bell shows 1 unread, "Approval needed —
   Priya Raman submitted 'Launch film' (A-1042)"; choosing it opens the Approval section and marks it read.
2. **Given** Daniel declines, **Then** Priya is notified with the start of his comment and Daniel
   receives nothing; when level 1 of 2 is approved, the owner is told and level-2 approvers receive
   "Approval needed".
3. **Given** unread notifications, **When** the user chooses **Mark all as read**, **Then** the count
   disappears and the items stay listed; with none, the panel says "You're all caught up."

---

### Edge Cases

- The owner is an approver of the current level: they may approve (D-020); the dialog notes "You own
  this activity". One person on two levels decides each level separately.
- A designated approver is inactive: the chain shows "(inactive)", anyone can act on their behalf, and
  the workflow's details warn "Level 2 has no active approver".
- The start date passes while Submitted: no automatic approval; the approvals list shows "Start date
  passed" in red.
- A submitted activity is deleted: its round ends as "Cancelled — activity deleted"; a notification about
  it says the activity is in the recycle bin. Copies carry no approval history (R13).
- A positive unassigned planned cost, or no line items, does not block submitting (domain model §4.3);
  the confirmation shows the unassigned amount as a note.

## Requirements *(mandatory)*

### Functional Requirements

**Approval workflows (Settings)**

- **FR-001**: MAPP MUST provide **Settings › Approval workflows** following the standard CRUD
  pattern (design system §6): list with search by name, Active/Inactive pill filters and a result count;
  create and edit in a side panel; a details view; **Set inactive / Set active**.
- **FR-002**: A workflow MUST have a name (required, at most 120 characters, unique ignoring letter case
  and surrounding spaces), an optional description, an active flag and one or more ordered levels
  (domain model §6). Each level MUST have at least one approver (person picker), MAY have a name
  (default "Level n") and MUST NOT list a person twice. Levels can be added, removed and reordered.
- **FR-003**: Saving a workflow MUST report every problem at once (missing or duplicate name, no level,
  a level without approvers, a person twice in a level, an inactive person).
- **FR-004**: Every level MUST apply to every submission, in order; any one approver of a level
  approves it. Resolved: no cost thresholds — every level applies to every
  submission, in order, and any one designated approver of a level approves it (D-205).
- **FR-005**: The list and details MUST show each workflow's levels with their approvers, its status and
  "Used by N activities (n waiting for approval)"; the details MUST list those activities with links.
- **FR-006**: A change to a workflow MUST apply only to later submissions; an open round MUST keep the
  levels and approvers it was submitted with, and the edit panel MUST say so when open rounds exist.
- **FR-007**: Setting a workflow inactive MUST be confirmed with its effects; an inactive workflow MUST
  no longer be offered in the activity's Approval workflow field (field catalogue §4.1 #9), MUST stay on
  activities using it labelled "(inactive)", and MUST let open rounds finish.
- **FR-008**: A workflow never used by any activity MAY be deleted with the standard delete pattern
  (design system §6.5); for any other, Delete MUST stay visible and explain "Used by N activities — set
  it inactive instead."

**Submitting**

- **FR-009**: A Created activity MUST offer **Submit for approval**, and a Declined one **Re-submit for
  approval**, on its page.
- **FR-010**: Submitting MUST first run the **ready-to-submit check** and, if anything fails, change
  nothing and list every problem at once, each linking to its field or line item: (a) every empty
  "needed to submit" field of the activity and of each of its line items (domain model §3, field
  catalogue §4–5); (b) every % split not totalling 100.00 % (R10); (c) every custom spread not adding up
  to its line item's planned cost (domain model §4.4); (d) a negative unassigned planned cost (domain
  model §4.3, D-016); (e) an inactive approval workflow.
- **FR-011**: When the check passes, a confirmation MUST show the workflow, each level with its
  approvers, any positive unassigned planned cost as a note, and an optional comment. Confirming MUST set
  the activity to **Submitted** and start a round at level 1 holding a copy of the workflow's current
  levels and approvers, with the submitter, time and comment.
- **FR-012**: Submitting MUST be possible whatever the plan's status. Resolved: the plan does not need to be
  Published; activities of Draft plans can be submitted (D-206).
- **FR-013**: While a round is open, the activity's Approval workflow field MUST be locked with the
  reason "Recall the activity to choose another workflow."

**Deciding, level by level**

- **FR-014**: While an activity is Submitted, its page and the approvals-page preview MUST offer
  **Approve** (optional comment) and **Decline** for the current level of the open round.
- **FR-015**: Approving a level other than the last MUST move the round to the next level; approving the
  last level MUST set the activity to **Approved**.
- **FR-016**: Decline MUST require a comment that is not blank (comments hold at most 2,000 characters)
  and MUST end the round and set the activity to **Declined** at any level; the comment MUST be kept as
  a comment on the activity (domain model §7) and shown at the top of its Approval section.
- **FR-017**: Any signed-in user MUST be able to approve or decline (D-020). When the user is not a
  designated approver of the current level, the confirmation MUST say the decision will be recorded as
  made on behalf of the level's approvers (named), and the history MUST show "<actor> on behalf of
  <approver names>".
- **FR-018**: Only the first decision on a level MUST count; if the level was decided or the round ended
  after the user opened the activity, the action MUST change nothing and say what happened ("Level 1 was
  already approved by Daniel Okoro at 10:42.").
- **FR-019**: Approved activities MUST be shown as **In progress** and **Complete** according to their
  dates (domain model §2.3); these displayed statuses never notify anyone.

**Re-submit, recall and re-approval**

- **FR-020**: Re-submitting MUST run the check of FR-010 and start a new round (next round number) at
  level 1 with the workflow's levels and approvers as they are at that moment.
- **FR-021**: A Submitted activity MUST offer **Recall submission** with an optional reason; confirming
  MUST end the round as "Recalled" and set the activity to **Created**. Approved activities MUST NOT offer
  it and MUST explain that the changes of domain model §2.5 need approval again. Anyone may recall in
  this phase (only the owner once roles arrive). *Amended by 034 FR-005: the explanation names only the
  activity's planned cost and dates.*
- **FR-022**: When a change listed in domain model §2.5 is saved on a Submitted or Approved activity
  (after the confirmation provided by features 006 and 007), MAPP MUST end any open round as
  "Approval restarted", set the activity to **Created** and record the changed values (old → new) in the
  approval history. Other edits MUST NOT affect approval (D-013).
- **FR-023**: Deleting a Submitted activity MUST end its round as "Cancelled — activity deleted" and
  remove it from the approvals page; restored activities come back Created with their history (R14).

**Approval section, history and log**

- **FR-024**: The activity page MUST have an **Approval** section with the status chip, the workflow and
  the chain of the current or last round — one row per level with a badge (L1, L2…), its designated
  approvers ("you" for the signed-in user) and its state (Approved by … on …, Declined by … on …,
  Waiting, Not reached yet) — followed by the available actions. A Submitted activity MUST show the
  banner "Submitted for approval — changing planned cost or dates will restart approval." (design
  system §6.6).
- **FR-025**: The Approval section MUST show the history of every round, newest first: round number,
  level (number and name), designated approver(s) at the time, who acted (with "on behalf of"), action
  (Submitted, Re-submitted, Approved, Declined, Recalled, Approval restarted, Cancelled), date and time,
  and comment or reason.
- **FR-026**: Every approval action and every change to a workflow MUST be written to the activity log
  with the actor (feature 001).

**Approvals page**

- **FR-027**: The sidebar's **Approvals** item MUST open the approvals page (page header pattern, design
  system §2) and show a count badge equal to the **Waiting for me** count when above zero.
- **FR-028**: The page MUST show four summary cards — **Waiting for me**, **All open**, **Approved this
  month**, **Declined this month** (rounds ended by a final approval or a decline in the current
  calendar month) — which tabs and filters do not change.
- **FR-029**: The page MUST offer pill tabs **Waiting for me** (Submitted activities whose current level
  lists the signed-in user), **All open** (every Submitted activity) and **Decided by me** (activities on
  which the signed-in user approved or declined a level in any round).
- **FR-030**: Each row MUST show the activity name and ID, plan › campaign, owner, planned cost in the
  display currency, workflow, "Level n of m · <approver names>", submitted date and time waiting;
  **Decided by me** adds the user's latest decision, its date and the current status. Default order:
  oldest submission first (**Decided by me**: latest decision first).
- **FR-031**: The page MUST offer search by activity name or ID and filters for plan, workflow, owner and
  submitted date range, with removable chips, a result count and the empty states of design system
  §6.1 ("Nothing is waiting for you. You're all caught up." on an empty **Waiting for me**).
- **FR-032**: Selecting a row MUST open a preview side panel with the activity's status, completeness,
  plan › campaign › initiative, owner, dates, planned cost, type and description; its line items (name,
  planned cost in local currency and USD, spend type, GL code, finance budget item); the unassigned
  planned cost; the approval chain and latest history; and **Approve**, **Decline** and **Open
  activity**. After a decision the panel MUST move to the next row and a message MUST name the outcome.

**Notifications**

- **FR-033**: The bell (feature 001) MUST list the signed-in user's 50 most recent in-app notifications,
  newest first, each with an icon, title, one line naming the activity and the actor, a relative time
  (full date and time on hover) and an unread marker; a badge MUST show the unread count ("9+" above 9).
- **FR-034**: Choosing a notification MUST mark it read and open the activity's Approval section;
  **Mark all as read** MUST mark all read; with none, the panel MUST say "You're all caught up."
- **FR-035**: MAPP MUST notify the designated approvers of a level when a round reaches it
  ("Approval needed"); the activity's owner on every approval, on a decline (with the start of the
  comment), on a recall and on an approval restart; and the approvers of the waiting level when a round
  is recalled or restarted (for a restarted Approved activity, the people who approved it).
- **FR-036**: A person MUST never be notified about their own action, and new notifications MUST appear
  within one minute without reloading the page.

**Demo data**

- **FR-037**: The demo data MUST include the two workflows of
  [demo-users-and-data.md](../../docs/product/demo-users-and-data.md); rounds, history and notifications
  consistent with every seeded activity's status (including a Declined activity with Daniel Okoro's
  comment and a Submitted one waiting for him); and one complete FY2026 Mobility China activity waiting
  at level 2 for Alex Morgan after Carlos Mendes approved level 1.

### Key Entities *(include if feature involves data)*

- **Approval workflow**: name, description, active flag, ordered levels; created and last changed by.
- **Approval level**: position, optional name, one or more approvers (people).
- **Approval round**: one attempt to get an activity approved — round number, a copy of the workflow's
  levels and approvers at submission, current level, outcome (open, Approved, Declined, Recalled,
  Approval restarted, Cancelled), submitter, start and end time.
- **Approval history entry**: round, level, designated approver(s), actor, "on behalf of", action, time,
  comment or reason, changed values for a restart.
- **Notification** (domain model §7): recipient, kind, title, text, activity, actor, time, read flag.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A ready activity is submitted in at most 2 steps (**Submit for approval**, confirm); for
  an incomplete one, 100 % of its problems are shown in the first attempt (design system §10).
- **SC-002**: An approver approves a waiting activity from the approvals page in at most 3 steps
  (select the row, **Approve**, confirm), and declines in at most 4 (adding the comment).
- **SC-003**: In a test with at least 5 first-time users, at least 90 % submit an activity, and approve
  and decline one as an approver, without help, each within 2 minutes; at least 90 % of first-time
  administrators create a two-level workflow within 2 minutes.
- **SC-004**: 100 % of approval actions appear in the approval history and the activity log with the
  correct round, level, designated approvers, actor and "on behalf of".
- **SC-005**: 100 % of the changes listed in domain model §2.5 return a Submitted or Approved activity to
  Created; no other edit does; editing or inactivating a workflow changes no open round.
- **SC-006**: 100 % of notifications reach exactly the intended people within one minute, and nobody is
  ever notified about their own action.
- **SC-007**: The approvals page's card counts always equal the rows of the matching unfiltered tab, and
  the sidebar badge equals the **Waiting for me** count.
- **SC-008**: Every screen of this feature passes an automated WCAG 2.2 AA check and can be used with
  the keyboard alone.

## Assumptions

- Until Q-002 is answered, a workflow is ordered levels where any one approver approves a level; Hive9's
  approver types, delegates and amount thresholds (FA §C6, §E6.1.2) are not part of this phase.
- No roles (D-004, D-020): anyone may approve, decline or recall; once roles arrive, deciding is limited
  to the level's approvers and recall to the owner (domain model §8).
- Feature 006 owns the activity page, its Approval workflow field and the confirmation of re-approval
  changes; feature 007 owns line items and their SWBS status (domain model §2.4). This feature owns the
  ready-to-submit check (reusing their completeness rules), the Approval section and approval actions.
- "This month" is the current calendar month in the viewer's local time. The owner, not the submitter,
  is notified when they differ. Deleting a never-used workflow uses the recycle bin (feature 001).
- Out of scope: approving many activities at once, purchase-request and accrual approvals (Finance),
  campaign and initiative approvals (D-012), e-mail notifications.
