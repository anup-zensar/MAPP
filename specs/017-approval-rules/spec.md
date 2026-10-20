# Feature Specification: Approval Rules

**Feature Branch**: `017-approval-rules`

**Created**: 2026-10-20

**Status**: Draft

**Input**: Shell BRD **BUS.WFL.010** (MVP) with the client's comments, and **BUS.WFL.030** (MVP, confirm it
is met). Product-owner decision of 2026-10-20: build every MVP gap now with sensible, documented,
reversible defaults, each listed for Shell to confirm.

**Builds on feature 014** ([Access control](../014-access-control/spec.md)), which must be merged first: its
permissions (`approvals.decide` "Approve or decline when named as an approver", `approvals.decideOnBehalf`,
`activities.submit`, `workflows.manage`), its roles and scopes, its access rule "Nobody approves an activity
they own or submitted" (on by default), its 403 `forbidden` refusals and its approve/decline check order
(014 FR-025 – FR-028). This feature extends them; it does not redefine them.

**Amends feature 008** ([spec](../008-activity-approvals/spec.md)): FR-002, FR-003, **FR-004 (replaced)**,
FR-005, FR-010, FR-011, FR-014, FR-015, FR-016, FR-018, FR-024, FR-025, FR-026, FR-027, FR-029, FR-030,
FR-032, FR-033, FR-035, FR-037; Edge case 2; Assumption 1. (008 FR-017 and FR-021 were replaced by 014
FR-025 and FR-027; 017 extends those.) **Amends feature 014**: FR-006 (delegates and backup approvers see
what they may decide), FR-025 (who may decide: + delegates, backup approvers), FR-026 (the access rule also
covers the round's submitter, as 014's *Access rules* entity already says), FR-028 (also backup approvers
and role levels), plan Table C (7 new operations). **Amends feature 003**: FR-002, FR-004, FR-005 (line
manager). The table at the end lists every amendment.

**References**: BRD [BUS.WFL.010, BUS.WFL.030](../../docs/reference/brd-requirements.md) · Gap analysis
[§3.10, §3.11, Q-014](../../docs/reports/brd-gap-analysis.md) · Domain model §2.3, §2.5, §6, §7, §8 ·
Decisions D-013, D-020, D-155 – D-169, D-205 · 014 D-231 (D-A9), D-232 (D-A10) · Open questions Q-002, Q-014 ·
Feature 011 (Activity log page and record History).

The client asks to keep approvals simple in three parts. This spec follows them:

- **Part A — Approval workflow**: who approves and when (automatic or manager approval, conditional
  levels, "all must approve" levels, role-based approvers).
- **Part B — Delegation and control**: delegation of authority, segregation of duties, who may decide.
- **Part C — Tracking and notification**: reminders, escalation, a monitoring dashboard, the audit trail.

Demo people and roles (014 seed): Alex Morgan Administrator · Carlos Mendes Global planner · Daniel Okoro
Planner and Approver (Lubricants Europe) · Priya Raman and Mei Lin Planners · Sara Novak Viewer. People who
can approve: Alex, Carlos, Daniel.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Small activities approve themselves, the rest go to the owner's manager (Priority: P1)

Alex edits the workflow "Standard marketing approval": level 1 is approved by **the activity owner's line
manager**, and is **approved automatically when the planned cost is below $10,000.00**. Priya submits an
$8,000.00 activity: it is Approved at once. She submits a $30,000.00 one: it waits for Daniel Okoro, her
line manager.

**Why this priority**: The client's first comment: "auto approval / manager approval — so approval
flexibility". It removes approval work for small spend while keeping control of the rest.

**Independent Test**: With the seeded "Standard marketing approval", submit an $8,000.00 and a $30,000.00
Lubricants activity as Priya Raman; the first is Approved, the second waits for Daniel Okoro.

**Acceptance Scenarios**:

1. **Given** level 1 "Line manager" with "Approve automatically when planned cost is below $10,000.00",
   **When** Priya submits an activity of $8,000.00, **Then** the confirmation shows "Level 1 · Line
   manager — Approved automatically: planned cost is below $10,000.00" and "This activity will be
   approved automatically."; confirming makes it **Approved**; its history shows "Submitted" by Priya
   Raman and "Approved automatically" by NorthStar with the rule; Priya receives no notification.
2. **Given** the same workflow, **When** Priya submits an activity of $30,000.00, **Then** the
   confirmation shows "Level 1 · Daniel Okoro (line manager of Priya Raman)", the activity becomes
   Submitted waiting for Daniel, and Daniel is notified.
3. **Given** the owner has no line manager, **When** she submits, **Then** nothing changes and the
   problem list (with every other problem) says "Level 1: Priya Raman has no line manager. Ask an
   administrator to set one in Settings › People."
4. **Given** a level whose approvers are "people with the role Approver" (014), **When** a Lubricants
   activity is submitted, **Then** the level's approvers are the active people who hold Approver with a
   scope covering the plan (Daniel Okoro); for a plan nobody's Approver scope covers, the problem reads
   "Level 2: nobody holds the role 'Approver' for this plan."
5. **Given** Settings › People, **When** an administrator sets Priya's **Line manager** to Daniel
   Okoro, **Then** Priya's details show "Line manager: Daniel Okoro" and Daniel's show "Line manager of
   2 people"; choosing Priya as her own manager, or Priya as Daniel's manager while Daniel is hers, is
   refused with a message.

---

### User Story 2 - Levels that apply only sometimes, and levels everyone must approve (Priority: P1)

The same workflow has level 2 "Marketing administrator" (Alex Morgan), **only needed when the planned cost
is at least $100,000.00**, and level 3 "Mobility second check" with Daniel Okoro and Alex Morgan, **all
must approve**, only needed when the plan's business unit is Mobility.

**Why this priority**: Multi-level, conditional and parallel approvals are named in BUS.WFL.010; they
let one workflow serve small and large spend.

**Independent Test**: Submit a $120,000.00 Lubricants activity and a $50,000.00 Mobility activity with
"Standard marketing approval"; the first needs levels 1 and 2, the second levels 1 and 3, and level 3
completes only after both Daniel and Alex approve.

**Acceptance Scenarios**:

1. **Given** a $30,000.00 Lubricants activity, **When** it is submitted, **Then** the confirmation and
   the chain show level 2 "Not needed — only when planned cost is at least $100,000.00" and level 3
   "Not needed — only when business unit is Mobility"; once level 1 is approved, the activity is
   Approved.
2. **Given** a $120,000.00 Lubricants activity, **When** Daniel approves level 1, **Then** it waits at
   level 2 for Alex Morgan, and Alex's approval makes it Approved.
3. **Given** Mei's $50,000.00 Mobility activity, **When** Carlos (her line manager) approves level 1,
   **Then** it waits at level 3 for Daniel Okoro and Alex Morgan, shown "0 of 2 approved"; **When**
   Daniel approves, **Then** it stays Submitted, the chain shows "1 of 2 approved — waiting for Alex
   Morgan", and Daniel's **Waiting for me** no longer lists it; **When** Alex approves, **Then** the
   activity is Approved.
4. **Given** level 3 waiting, **When** either declines with a comment, **Then** the activity is Declined
   at once.
5. **Given** the workflow form, **When** Alex saves a level with "Only needed when" and no condition, an
   "at least" amount of $50,000.00 with a "below" amount of $20,000.00, and "Escalate" on without a
   backup approver, **Then** all three problems are shown at once, each next to its field.

---

### User Story 3 - Only the right people decide (Priority: P1)

014 lets only the named approvers decide (or someone allowed to decide on behalf), and stops owners
deciding on their own activity. This feature keeps those rules for the new approver sources and adds the
submitter to segregation of duties, the backup approver, and delegates (US4).

**Why this priority**: BUS.WFL.010 asks for role-based approvals, segregation of duties and compliance
controls.

**Independent Test**: As Daniel, submit an activity he owns in "Lubricants EU approval" (level 1: Daniel,
backup Alex Morgan); Daniel is left out and the round waits for Alex.

**Acceptance Scenarios**:

1. **Given** Daniel owns an activity in a workflow whose level-1 approver is Daniel, with Alex Morgan as
   backup approver, **When** Daniel submits it, **Then** the confirmation says "Daniel Okoro is left out:
   owns this activity." and "The backup approver Alex Morgan approves this level.", and the round waits
   for Alex.
2. **Given** the same without a backup approver, **When** Daniel submits it, **Then** it is submitted with
   014's advice "You are the only approver of level 1. An administrator will have to approve on your
   behalf.", the level shows "No approver", and only someone allowed to decide on behalf can decide.
3. **Given** Daniel submitted Priya's activity and is her line manager, **Then** Daniel is left out of
   level 1 ("is submitting it"), and a decision he sends anyway is refused: "You submitted this activity,
   so someone else must approve it."
4. **Given** a resolved approver who has lost every role that approves, **Then** the chain flags "(can't
   approve — no approving role)" (014) and they cannot decide; a delegate, the backup approver after
   escalation, or someone allowed to decide on behalf can.
5. **Given** the workflow form, **When** Alex picks Sara Novak (Viewer) as a backup approver, or a role
   that does not approve, **Then** the field says "Level 1: Sara Novak can't approve activities. Give them
   a role that approves first." or "Level 2: people with the role 'Viewer' can't approve activities.
   Choose a role that approves."

---

### User Story 4 - Delegate my approvals while I'm away (Priority: P1)

Daniel is on leave from 1 to 10 November. On the Approvals page he chooses **Delegate my approvals**,
picks Carlos Mendes and the dates. During those days Carlos sees Daniel's waiting activities, is told about
new ones, and his decisions are recorded "Carlos Mendes as delegate of Daniel Okoro".

**Why this priority**: Delegation of authority is named in BUS.WFL.010 and in the client's second
comment ("approval delegations/control").

**Independent Test**: As Daniel, delegate to Carlos from today for three days; as Carlos, approve the
seeded activity waiting for Daniel and see "as delegate of Daniel Okoro" in its history.

**Acceptance Scenarios**:

1. **Given** Daniel on Approvals, **When** he chooses **Delegate my approvals**, picks Carlos Mendes,
   01 Nov 2026 – 10 Nov 2026 and the note "Annual leave", and saves, **Then** the panel lists it as
   **Planned** (or **Current** when it starts today), a message confirms it, and the activity log reads
   "Daniel Okoro created approval delegation 'Daniel Okoro → Carlos Mendes'".
2. **Given** the delegation is current, **When** Carlos opens Approvals, **Then** a banner says "You are
   approving for Daniel Okoro until 10 Nov 2026.", **Waiting for me** lists the activities waiting for
   Daniel marked "as delegate of Daniel Okoro", and his sidebar badge counts them; Daniel keeps them and
   sees the banner "Carlos Mendes approves for you until 10 Nov 2026."
3. **Given** Carlos approves one, **Then** the history shows "Carlos Mendes as delegate of Daniel Okoro"
   and the activity leaves both people's **Waiting for me**.
4. **Given** Daniel already delegates to Carlos on those days, **When** he adds a second delegation that
   overlaps, **Then** it is refused: "You already delegate to Carlos Mendes on some of these days. Change
   that delegation instead."
5. **Given** Daniel picks Sara Novak (Viewer) as delegate, **Then** it is refused: "Sara Novak can't
   approve activities. Choose someone with a role that approves."
6. **Given** the delegate owns or submitted one of the waiting activities, **Then** it is not in their
   **Waiting for me** and they cannot decide it (segregation of duties applies to delegates).
7. **Given** a current delegation, **When** Daniel chooses **Cancel delegation** and confirms, **Then**
   Carlos loses the rights at once, the delegation shows **Cancelled**, and decisions he already made
   stay recorded.

---

### User Story 5 - Reminders and escalation when an approval waits too long (Priority: P2)

Each workflow can send a reminder after N days of waiting and escalate to the level's backup approver
after M days. Someone who can change approval workflows chooses **Run reminders and escalations** (it
stands in for the nightly job that will run it once NorthStar is hosted).

**Why this priority**: BUS.WFL.010 asks for automated notifications and escalations; approvals already
work without them.

**Independent Test**: With the seed (Lubricants EU approval: remind after 1 day, escalate after 2 days),
run the job as Alex Morgan: "Trade magazine ads" is escalated to Alex, who then approves it as backup
approver.

**Acceptance Scenarios**:

1. **Given** a level waiting 3 days in a workflow that reminds after 3 days, **When** the job runs,
   **Then** the level's waiting approvers and their current delegates get "Reminder — approval waiting
   3 days", the history shows "Reminder sent" by NorthStar, and the result says "1 reminder sent, no
   escalations".
2. **Given** the job ran, **When** it runs again the same day, **Then** nothing new is sent.
3. **Given** a level waiting 5 days in a workflow that escalates after 5 days, **When** the job runs,
   **Then** the backup approver gets "Approval escalated to you", the waiting approvers and the owner
   are told, the row shows "Escalated to Alex Morgan", and Alex's approval completes the level ("Alex
   Morgan as backup approver").
4. **Given** escalation is due but the backup approver is inactive, cannot approve, owns or submitted the
   activity, or is already an approver of the level, **Then** nothing is escalated and the result lists
   "'Trade magazine ads' (A-1004), level 1: no backup approver can take it."
5. **Given** a person without "Create and change approval workflows", **Then** **Run reminders and
   escalations** is not shown and the API refuses the request (403).

---

### User Story 6 - See where approvals are stuck (Priority: P2)

The Approvals page gains a **Monitoring** tab: summary cards, how long open approvals have waited, open
approvals by workflow level, and the overdue list.

**Why this priority**: BUS.WFL.010 asks for workflow monitoring and process dashboards; the client asks
for "approval tracking".

**Independent Test**: Open Approvals › Monitoring as Alex with the seed; the ageing bars add up to the
**All open** count and the overdue list holds "Trade magazine ads".

**Acceptance Scenarios**:

1. **Given** the seed, **When** anyone opens **Monitoring**, **Then** four cards show **Overdue**,
   **Escalated**, **Average time to decision** (this month) and **Approved automatically** (this month),
   each with an ⓘ tip, counting only activities the person can see (014).
2. **Given** open approvals, **Then** **How long approvals have waited** shows five bars — Under 1 day,
   1–2 days, 3–5 days, 6–10 days, More than 10 days — each with its count as text, adding up to **All
   open**.
3. **Given** open approvals, **Then** **Waiting by level** lists each workflow and level with the number
   waiting, the number overdue and the oldest waiting time.
4. **Given** overdue approvals, **Then** **Overdue** lists them oldest first with the approvals-page row
   columns plus the "Overdue" flag and any escalation; selecting a row opens the preview.

---

### User Story 7 - Every approval event is on the record (Priority: P2)

Every rule outcome, decision, reminder, escalation, delegation change and workflow change is in the
activity's approval history and in the Activity log (feature 011), with who, when and why.

**Why this priority**: BUS.WFL.010 asks for an audit trail and compliance controls.

**Independent Test**: After stories 1–5, open the activity log filtered on one activity and on the record
type "Approval delegation"; every event is there with its person (or NorthStar) and its rule or reason.

**Acceptance Scenarios**:

1. **Given** any approval event of this feature, **Then** the activity's **Approval history** and the
   Activity log both show it with the person (NorthStar for automatic events), the time and the rule,
   capacity or reason ("as delegate of Daniel Okoro", "planned cost is below $10,000.00", "waited 5
   days").
2. **Given** a round, **Then** its history heading shows the facts the rules used: "Rules applied to:
   planned cost $30,000.00 · Event · Lubricants".
3. **Given** delegations are created, changed or cancelled, **Then** the Activity log lists them under the
   record type "approval delegation".

---

### Edge Cases

- **Workflow edited while rounds are open**: open rounds keep their levels, resolved approvers, rule
  outcomes, timings and backups (008 FR-006 holds).
- **Values change after submission**: the changes of domain model §2.5 restart approval (D-013) and the
  next submission evaluates the rules again. Other changes (activity type, the plan's business unit) do
  not restart approval and do not re-evaluate an open or approved round (Proposed 017-P1).
- **Line manager, roles or role holders change after submission**: the round keeps the people it resolved;
  whether each can still decide is checked when they decide (014 FR-025).
- **Approver deactivated during a round**: listed "(inactive)"; their delegates, the backup after
  escalation, or someone allowed to decide on behalf can decide. The row shows "No approver" when nobody
  waiting can act.
- **The same person on two levels** decides each level separately; **a person who is both an approver and
  a delegate of another approver** on an "all must approve" level approves for their own place first; the
  activity stays in their **Waiting for me** "as delegate of …" until they approve again.
- **Every level is "Not needed" or approved automatically**: the activity is Approved at submission.
  With no level needed at all, the history reads "No level applies — approved automatically".
- **The access rule is turned off** (014 Rules tab): owners and submitters are no longer left out or
  refused; everything else stays.
- **Delegate deactivated or loses every approving role**: the delegation has no effect meanwhile.
- **Activity deleted while waiting**: unchanged (008 FR-023); reminders and escalations skip activities
  that are not stored.

## Requirements *(mandatory)*

BRD trace: every FR names its BRD capability. "W10" = BUS.WFL.010, "W30" = BUS.WFL.030.

### Functional Requirements

**Part A — Approval workflow: who approves and when**

- **FR-001** *(W10 multi-level, role-based approvals; amends 008 FR-002)*: Each level of an approval
  workflow MUST have one approver source: **Named people** (one or more people, as today), **Line
  manager** (the activity owner's line manager, FR-016) or **Role** (the active people who hold a 014 role
  that approves, with a scope that covers the activity's plan).
- **FR-002** *(W10 parallel approvals; amends 008 FR-002, replaces FR-004)*: A level with named people
  or a role MUST have a mode: **Any one approves** (default) or **All must approve**. A line-manager
  level has one approver and no mode.
- **FR-003** *(W10 conditional approvals; replaces 008 FR-004)*: A level MAY have an **Only needed
  when** condition. A level without one is always needed; a level whose condition does not match is
  **Not needed** for that submission and is passed over.
- **FR-004** *(W10 client comment 1: auto vs manager approval)*: A level MAY have an **Approve
  automatically when** condition. When a needed level's condition matches, NorthStar approves the
  level itself, with the rule as its reason; otherwise the level's approvers decide.
- **FR-005** *(W10 conditional approvals)*: A condition MUST hold one or more of: planned cost (USD)
  **at least** an amount; planned cost **below** an amount; the plan's **business unit is one of**
  chosen values; the **activity type is one of** chosen types. It matches when every part it holds
  matches. Amounts are 0.00 or more with at most two decimals, and "at least" must be lower than
  "below" when both are given.
- **FR-006** *(W10 escalations)*: A level MAY have a **backup approver**: one active person who can
  approve (014 `approvals.decide`) and is not one of the level's named approvers. It is required on every
  level when the workflow escalates (FR-007).
- **FR-007** *(W10 escalations, automated notifications)*: A workflow MAY **remind after** N days (1–60)
  and **escalate after** M days (1–90, more than N when both are set). New workflows start with 3 and 5
  days; either can be turned off.
- **FR-008** *(amends 008 FR-003)*: Saving a workflow MUST report every problem at once, including the
  new ones of data-model §1–§2 (no approver source, no role or a role that does not approve, empty
  condition, wrong amounts, unknown business unit or activity type, missing backup approver or one who
  cannot approve, wrong day counts).
- **FR-009** *(amends 008 FR-005)*: The workflow list and details MUST summarise each level's rule in
  words, e.g. "Level 1 · Line manager · automatic when planned cost is below $10,000.00 → Level 2 · Alex
  Morgan · only when planned cost is at least $100,000.00" (data-model §2), and show the reminder and
  escalation timings.
- **FR-010** *(W10 compliance controls; amends 008 FR-011)*: On submission NorthStar MUST evaluate the
  rules once, level by level, on the activity's values at that moment (planned cost, activity type, the
  plan's business units), and keep in the round each level's outcome — **Needs approval**, **Not
  needed** or **Approved automatically** — with its reason, its resolved approvers and the values used.
  Open rounds are never re-evaluated; a re-submission evaluates afresh.
- **FR-011** *(W10 role-based approvals, segregation of duties)*: For each level that needs approval,
  NorthStar MUST resolve the approvers (named people; the owner's line manager; or the role's holders
  for the plan) and leave out, with the reason, inactive people, people who cannot approve, and — while
  014's access rule is on — the activity's owner and the person submitting. When nobody is left, the
  level's backup approver (when it can act) becomes its approver; otherwise the level waits with **no
  approver**, for someone allowed to decide on behalf, and the confirmation says so (014's advice).
- **FR-012** *(W30; W10 compliance controls; amends 008 FR-010)*: The ready-to-submit check MUST add a
  problem for each level that needs approval whose source names nobody at all — the owner has no line
  manager, or the role is unknown or nobody holds it for the plan — listed with every other problem at
  once; nothing changes until all are fixed.
- **FR-013** *(amends 008 FR-011)*: The submit confirmation MUST show each level with its outcome — the
  approvers and how they were found ("line manager of Priya Raman", "role Approver"), "Not needed —
  <reason>" or "Approved automatically — <reason>" — the people left out and why, the backup approver
  used, the "no approver" advice, and, when no level needs a decision, "This activity will be approved
  automatically."
- **FR-014** *(W10 auto approval)*: Confirming MUST start the round as today, record the levels that are
  not needed or approved automatically, and wait at the first level that needs approval. When no level
  needs approval, the activity MUST become **Approved** at once and the round end as approved.
- **FR-015** *(W10 parallel approvals; amends 008 FR-015 and FR-016)*: A level that needs approval is
  approved by the first approval when "Any one approves", and when every resolved approver has approved
  when "All must approve" (a delegate's approval counts for the person they act for). An approval by
  the backup approver after escalation, or on behalf (014), approves the whole level. Any decline by
  someone who may decide (FR-017) declines the activity. After a level, the round moves to the next
  level that needs approval; after the last, the activity is Approved.
- **FR-016** *(W10 client comment 1: manager approval; amends 003 FR-002, FR-004, FR-005)*: A person
  MAY have a **Line manager** (an active person, not themselves, without a loop of managers), set in
  Settings › People. Person details MUST show the line manager and "Line manager of <n> people";
  **Deactivate** MUST also state "Line manager of <n> people". (Later it comes from Microsoft Entra ID.)

**Part B — Delegation and control**

- **FR-017** *(W10 role-based approvals, delegation; extends 014 FR-025, amends 008 FR-014)*: Only these
  people MAY approve or decline the current level, in this order: (a) a resolved approver who has not yet
  approved it and holds `approvals.decide`; (b) a current delegate (FR-023) of such an approver who holds
  `approvals.decide`; (c) the level's backup approver once the level is escalated, holding
  `approvals.decide`; (d) someone holding `approvals.decideOnBehalf` for the plan, recorded as on behalf
  of the approvers still waiting. Everyone else is refused with 014's 403 messages; an approver who has
  already approved an "all must approve" level is told "You already approved level <n>. Waiting for
  <names>."
- **FR-018** *(W10 segregation of duties; extends 014 FR-026)*: While 014's access rule is on, nobody MAY
  approve or decline an activity they own or whose round they submitted — in any capacity, delegates,
  backup approvers and people deciding on behalf included. The server MUST enforce it ("You own this
  activity, so someone else must approve it." / "You submitted this activity, so someone else must
  approve it.").
- **FR-019** *(W10 audit trail)*: Every decision MUST record the capacity it was made in: approver,
  delegate ("<person> as delegate of <approver>"), backup approver ("<person> as backup approver") or on
  behalf ("<person> on behalf of <approvers>").
- **FR-020** *(amends 008 FR-018)*: Only the first decision of each approver counts. A request for a
  level or round that moved on keeps 008's messages.
- **FR-021** *(W10 delegation; extends 014 FR-006)*: A current delegate of a waiting approver, and the
  backup approver of an escalated level, MUST be able to see the activity they may decide (its page, line
  items and approval), as 014 lets named approvers.
- **FR-022** *(W10 delegation of authority)*: A person MUST be able to delegate their approvals to one
  other active person who can approve (`approvals.decide`) for an inclusive date range (to-date today or
  later, at most 90 days long), with an optional note (500 characters). A person has at most one
  delegation on any day; overlapping ones are refused. People holding `workflows.manage` MAY create,
  change and cancel delegations for anyone.
- **FR-023** *(W10 delegation of authority)*: On each day of a current delegation, the delegate MUST be
  able to decide wherever the delegating person can as an approver (FR-017 a), MUST see those activities
  in **Waiting for me** marked "as delegate of <name>" (also counted in the sidebar badge and Home's
  "Waiting for my approval"), and MUST receive their approval-needed and reminder notifications. The
  delegating person keeps every right. Delegations do not chain.
- **FR-024** *(W10 delegations/control)*: **Delegate my approvals** on the Approvals page MUST list the
  person's delegations (Current, Planned, Ended, Cancelled), allow changing a planned or current one
  (a current one keeps its start date) and **Cancel delegation** with confirmation; ended and cancelled
  ones are read-only and kept. Banners on the Approvals page MUST tell a delegate and a delegating
  person about a current delegation.
- **FR-025** *(W10 role-based approvals; extends 014 FR-028)*: A workflow MUST NOT be saved with a backup
  approver who cannot approve, or a role level whose role does not include `approvals.decide`
  (data-model §2 messages). Resolved approvers who no longer hold it are left out at submission (FR-011)
  and flagged in open rounds (014).

**Part C — Tracking and notification**

- **FR-026** *(W10 automated notifications)*: Running reminders and escalations MUST, for each open
  round whose current level has waited at least the reminder days and has no reminder yet, notify the
  approvers still waiting and their current delegates, once per level.
- **FR-027** *(W10 escalations)*: For each current level that has waited at least the escalation days
  and is not escalated yet, the run MUST give the level's backup approver the right to decide it (when
  active, holding `approvals.decide`, not barred by FR-018, and not already an approver of the level) and
  notify the backup approver, the approvers still waiting and the owner. A due reminder is then not sent
  separately. When no backup approver can take it, nothing changes and the run reports it.
- **FR-028** *(W10 escalations, monitoring)*: People holding `workflows.manage` MUST be able to choose
  **Run reminders and escalations** on Approvals › Monitoring; it MUST be safe to run any number of times,
  answer how many reminders and escalations were sent and which levels could not be escalated, and record
  its events as made by NorthStar.
- **FR-029** *(W10 monitoring)*: A level's waiting time MUST be counted from when the round reached it
  (submission for the first level that needs approval, otherwise the approval of the level before), in
  whole days of 24 hours. A level is **overdue** when it has waited at least the round's escalation
  days, or, without escalation, its reminder days; without both it is never overdue.
- **FR-030** *(W10 workflow monitoring, process dashboards; amends 008 FR-029)*: The Approvals page MUST
  offer a fourth tab, **Monitoring**, with the cards, ageing bars, "Waiting by level" table and overdue
  list of User Story 6 and data-model §11, counting the same activities as **All open** (014-filtered).
- **FR-031** *(amends 008 FR-030, FR-027)*: Approval rows MUST also show, when true: "Overdue",
  "Escalated to <name>", "as delegate of <names>", "No approver" and, on "all must approve" levels,
  "<a> of <b> approved". **Waiting for me** and its counts MUST hold the activities the person can decide
  as approver, delegate or backup approver (not on behalf).
- **FR-032** *(W10 automated notifications; amends 008 FR-033, FR-035)*: Notifications MUST also be sent
  as in data-model §8: approval-needed to current delegates; "Approved automatically" to the owner when
  someone else submitted; reminders; escalations. Nobody is notified about their own action
  (008 FR-036 holds); a partial approval of an "all must approve" level notifies nobody.
- **FR-033** *(amends 008 FR-024, FR-032)*: The approval chain MUST show the new level states — Not
  needed, Approved automatically, "<a> of <b> approved", Escalated to <name>, Reminder sent, No approver —
  with each level's approver source and the people left out and why.
- **FR-034** *(W10 audit trail; amends 008 FR-025)*: The approval history MUST also show "Approved
  automatically", "Not needed", "Reminder sent" and "Escalated to <name>" entries (by NorthStar, with
  their rule or waiting days), each decision's capacity (FR-019), and per round the values the rules used.
- **FR-035** *(W10 audit trail, compliance controls; amends 008 FR-026)*: Every event of this feature —
  rule outcomes that approve an activity, decisions in every capacity, reminders, escalations,
  delegation created, changed and cancelled, workflow rule changes and line-manager changes — MUST be
  written to the activity log with the person (NorthStar for automatic events) and be readable on
  feature 011's Activity log page and record History.
- **FR-036** *(W30)*: Submitting MUST still be halted with every missing mandatory field and every other
  problem listed at once (008 FR-010); this feature only adds problems (FR-012) and removes none.

**Demo data**

- **FR-037** *(amends 008 FR-037)*: The demo data MUST add line managers to the demo people, reminder,
  escalation and backup settings to the two seeded workflows, a third workflow "Standard marketing
  approval" showing every rule type, and one planned delegation, as in data-model §13; seeded statuses,
  rounds and notifications stay as they are.

### Key Entities *(include if feature involves data)*

- **Approval level** (changed): approver source, named approvers or role, mode, "only needed when"
  condition, "approve automatically when" condition, backup approver.
- **Approval workflow** (changed): reminder days, escalation days.
- **Approval round** (changed): per level its outcome, reason, resolved approvers, people left out,
  backup, when it began waiting, reminder and escalation times; the values the rules used; the timings.
- **Approval history entry** (changed): new actions (approved automatically, not needed, reminder sent,
  escalated) and the decision's capacity.
- **Approval delegation** (new): delegating person, delegate, from and to dates, note, cancelled.
- **Person** (changed): line manager.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For every combination in the rule test matrix (approver source × mode × "only needed
  when" × "approve automatically when" × planned cost, business unit and type values), 100 % of
  submissions produce the documented level outcomes and approvers.
- **SC-002**: 0 decisions are accepted from a person barred by FR-017 or FR-018 (server tests on every
  capacity, including delegates, backup approvers and people deciding on behalf).
- **SC-003**: An administrator who has never seen the screen sets up "line manager, automatic below
  $10,000.00" in under 3 minutes; a person sets up a delegation in at most 4 steps.
- **SC-004**: 100 % of approval events of this feature appear in the approval history and the activity
  log with the correct person or NorthStar, capacity and reason.
- **SC-005**: Running reminders and escalations twice in a row sends nothing the second time.
- **SC-006**: Monitoring's ageing bars add up to the **All open** count and its overdue list equals the
  rows flagged overdue.
- **SC-007**: Every new or changed screen passes the automated WCAG 2.2 AA check and works with the
  keyboard alone.

## Assumptions

Defaults chosen under the product-owner decision of 2026-10-20 (Proposed decisions in plan.md); each is
listed for Shell to confirm (Q-014, Q-002):

- Auto-approval and conditions use the activity's **planned cost in USD**, the **plan's business
  units** and the **activity type**; the $10,000.00 and $100,000.00 thresholds are demo values only.
- "Manager" is a **line manager field kept in NorthStar** (Settings › People), to be filled from
  Microsoft Entra ID when real sign-in arrives (SSO is simulated by demo sign-in).
- Rules are evaluated **once, at submission**; only the changes of domain model §2.5 restart approval.
- Segregation of duties is 014's access rule, **on by default**, and covers the owner **and the
  submitter**; people allowed to decide on behalf (Administrators) can unblock a level with no approver,
  and that is recorded.
- A level whose source names nobody (no line manager, a role nobody holds) **blocks** the submission; a
  level whose named people are all left out does **not** (the backup approver or someone deciding on
  behalf takes it).
- Delegation: one delegate at a time who can approve, at most 90 days, no chains, the delegating person
  keeps their rights, days counted as calendar dates in UTC.
- Reminders and escalation in **calendar days**, at most one of each per level; escalation goes to the
  level's backup approver, who can then decide the whole level. The job runs when someone who can change
  approval workflows starts it; a nightly schedule comes with hosting.
- Notifications stay **in-app**; e-mail and Teams come later with BUS.WFL.040.
- **BUS.WFL.030** is already met (gap analysis §3.11) and needs only Shell's confirmation of the field
  markings (Q-005); this feature keeps the check and adds to it.
- Out of scope: approval of campaigns and initiatives (D-012), e-mail, bulk approval, approvals of
  purchase requests or invoices (Finance).

## Amendment index

| Feature · FR | Change | By |
|---|---|---|
| 008 FR-002 | Level gains approver source, mode, conditions, backup; workflow gains timings | FR-001 – FR-007 |
| 008 FR-003 | More validation problems, all at once | FR-008 |
| 008 FR-004 | **Replaced**: levels may be conditional, automatic or "all must approve" | FR-002 – FR-005 |
| 008 FR-005 | Levels text and details describe rules and timings | FR-009 |
| 008 FR-010 | Check adds "source names nobody" problems | FR-012 |
| 008 FR-011 | Rules evaluated and approvers resolved at submission; confirmation shows outcomes | FR-010, FR-011, FR-013, FR-014 |
| 008 FR-014 | Approve / Decline only for people allowed to decide | FR-017 |
| 008 FR-015, FR-016 | Level completion per mode; passed-over levels; decline on "all" levels | FR-015 |
| 008 FR-018 | First decision per approver | FR-020 |
| 008 FR-024, FR-032 | New chain states | FR-033 |
| 008 FR-025 | New history entries and capacities | FR-034 |
| 008 FR-026 | More logged events | FR-035 |
| 008 FR-027, FR-030 | Waiting for me includes delegate and backup; row flags | FR-031 |
| 008 FR-029 | Fourth tab, Monitoring | FR-030 |
| 008 FR-033, FR-035 | New notification kinds and recipients | FR-032 |
| 008 FR-037 | Demo data additions | FR-037 |
| 008 Edge case 2 (inactive approver: anyone acts) | Delegates, backup after escalation or on behalf act | FR-017, Edge cases |
| 008 Assumption 1 | Rules, delegates and thresholds are now in scope | whole spec |
| 014 FR-006 | Delegates and escalated backup approvers see what they may decide | FR-021 |
| 014 FR-025 | Who may decide: + current delegates, + backup approver after escalation | FR-017 |
| 014 FR-026 | The access rule also covers the round's submitter (as 014's *Access rules* entity states) and applies to every capacity | FR-018 |
| 014 FR-028 | Also backup approvers and role levels | FR-025 |
| 014 plan Table C | Rules for the 7 new operations | plan *Access rules for the new operations* |
| 003 FR-002, FR-004, FR-005 | Line manager field, details, deactivate statement | FR-016 |
