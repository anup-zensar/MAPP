# Feature Specification: Overspend Control — Warn or Refuse per Budget

**Feature Branch**: `037-overspend-control`

**Created**: 2026-10-28

**Status**: Draft

**Input**: RFP §8 and BRD BUS.BUD.030: "Tool to allow budget owners to restrict or block overspending against
approved budgets, and to configure whether this is a hard stop or a warning." Brought into scope by D-333
(constitution 4.6.0): each budget owner chooses whether going over a budget warns — the default and today's
behaviour (D-015, D-235) — or is refused, enforced by the API. Also closes a reviewer's blocking finding on
feature 032: an activity approved at 80,000.00 can have its line items re-costed to 200,000.00 without
re-approval (034 FR-001) and then carry 200,000.00 of purchase requests.

**Amends feature 032** ([Purchase requests](../032-purchase-requests/spec.md)): **FR-021** gains a gate (FR-019
below). **Amends features 004, 005 and 014** ([Plans](../004-plans/spec.md),
[Campaigns and initiatives](../005-campaigns-and-initiatives/spec.md),
[Access control and global budgets](../014-access-control/spec.md)): the budget forms gain the overspend
setting, and a budget set to Refuse no longer only warns when planned cost goes over it (D-015 and D-235 keep
applying to every budget set to Warn, and to over-allocation). The table at the end lists every amendment.

**References**: RFP §8 · BRD BUS.BUD.030 · Finance discovery v8 p. 15 ("Hard limits convert a warning into a
block", a budget-level default) and p. 31 rule 21 ("Plan warns, Finance (hard limit) blocks") · Hive9
functional analysis §C2 ("Is Hard Limit" per finance budget) · Client spec 011 FR-024, FR-025 and the edge case
"Hard Limit switched on while rows are already negative" · Client spec 005 scenarios B1 and 3/4 (refusal
message naming the remaining budget and the excess) · Client spec 004 FR-049 (over-allocation is allowed) ·
Decisions D-015, D-016, D-235, D-303, D-322, D-333 · Domain model §4.1–§4.3.

Today a budget in MAPP is advice: when the planned cost below a plan, campaign, initiative or global budget
goes over it, the figures turn red and nothing stops the save. Shell's budget owners want to choose, budget by
budget, whether that is acceptable. A budget owner who sets **Refuse** knows that nobody can plan more than
the budget holds — whichever way the cost arrives: a new or re-costed activity, a move, a copy, a split, a
restore or an allocation. A budget left on **Warn** behaves exactly as today.

Demo people (014 seed): Priya Raman owns the plan "FY2026 Lubricants Europe" (budget 1,200,000.00 USD) and
its campaign "Helix launch" (700,000.00 USD); Alex Morgan is the administrator and manages global budgets;
Daniel Okoro approves Lubricants activities.

## Clarifications

### Session 2026-10-28

No product owner was available in the loop; each point is resolved from the sources (RFP > client specs >
Finance discovery > functional analysis; decision log) and recorded as a decision for the product owner to
confirm (D-346 – D-349, Q-024, Q-025).

- Q: What does "going over a budget" mean — the planned cost below it, or the budgets handed down to its
  children? → A: The **planned cost** below it (domain model §4.2): the sum of the activities' planned cost of
  the plan, campaign or initiative, and for a global budget the planned cost of every plan it allocates to,
  in USD at the current rates. Children's budgets exceeding the parent (over-allocation) keep warning only
  (D-015, D-235; client 004 FR-049 requires over-allocation to be allowed). Recorded as D-346 and Q-024.
- Q: Which budgets carry the setting? → A: Every budget MAPP has: plan, campaign, initiative and global
  budget. An allocated plan's budget is set by its allocation, so its setting is chosen with the allocation by
  the person who manages it (D-346).
- Q: May a budget be set to Refuse while its planned cost is already over it? → A: Yes (client 011 edge case):
  the form says it is already over; nothing is removed; any further increase is refused until the planned
  cost is back within the budget, and lowering the budget further is refused (D-346).
- Q: What is the organisation default — inherited by every budget, or the setting new budgets start with?
  → A: The setting new budgets start with. Changing it never changes an existing budget, so nothing changes
  until an owner (or the organisation, for budgets created afterwards) opts in. Copies keep their source's
  setting (D-347; Q-025 asks whether Shell wants inheritance instead).
- Q: Who may change the setting? → A: The budget's owner — the owner of the plan, campaign or initiative —
  or a person who manages global budgets for that plan; for a global budget and an allocation, whoever may
  change them. No new permission (D-348).
- Q: Do line-item changes count? → A: No. A line item never changes the planned cost of its activity, so it
  never changes the planned cost under a budget (domain model §4.3). The loophole the reviewer found — line
  items costing more than the approved activity — is closed where the money is committed: a purchase request
  cannot be submitted while its activities' line items exceed their planned cost (D-349).
- Q: Are underspend warnings part of this feature (BRD comment on BUS.BUD.030)? → A: No; out of scope by the
  instruction that brought this feature in.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A budget owner refuses overspending on a budget (Priority: P1)

Priya owns the plan "FY2026 Lubricants Europe". She opens the plan's form and, under the budget, chooses
**When planned cost goes over this budget: Refuse**. Later a colleague raises an activity's planned cost so
that the plan's planned cost would pass 1,200,000.00 USD: the activity form says before saving that the save
will be refused, and the save is refused with a message naming the plan budget, its amount and the overage.
Nothing is saved.

**Why this priority**: It is the RFP requirement itself (§8, BUS.BUD.030), and the budget forms and the
activity form are where planners and owners work every day.

**Independent Test**: On the demo data, set Refuse on "FY2026 Lubricants Europe"; raise an activity's planned
cost by more than the plan's left to plan: the form warns, the save is refused naming the budget, its amount
and the overage; raise it by less: it saves. Set Warn again: the larger raise saves and the red figures show.

**Acceptance Scenarios**:

1. **Given** Priya owns the plan, **When** she opens its edit form, **Then** below Plan budget she sees "When
   planned cost goes over this budget" with **Warn** ("Save and show the overage") chosen and **Refuse**
   ("Refuse any save that takes the planned cost over the budget"); **When** she chooses Refuse and saves,
   **Then** the plan shows "Refuses overspending" next to its budget and the activity log has one change
   "Overspend: Warn → Refuse" by Priya.
2. **Given** the plan refuses overspending with 1,200,000.00 USD budget and planned cost P, **When** anyone
   saves an activity whose new planned cost would take the plan's planned cost to more than the budget,
   **Then** nothing is saved and the message reads "Not saved — the planned cost of plan 'FY2026 Lubricants
   Europe' would be <new total>, over its budget of 1,200,000.00 USD by <overage>. Its owner refuses
   overspending."
3. **Given** the same plan, **When** the activity form's planned cost is typed above what is left to plan,
   **Then** before saving the field says "This is <overage> over the budget of plan 'FY2026 Lubricants
   Europe', which refuses overspending" and Save is refused on the form with that message.
4. **Given** a campaign or an initiative with a budget and an owner, **When** its owner chooses Refuse on its
   form, **Then** saves that would take the campaign's or initiative's planned cost over its budget are
   refused in the same way, naming the campaign or initiative.
5. **Given** a budget set to **Warn** (every budget until someone chooses Refuse), **When** a save takes the
   planned cost over it, **Then** it is saved and the existing figures show the overage in red (unchanged
   behaviour of features 004, 005 and 014).
6. **Given** a planner who is not the owner and does not manage global budgets, **When** they open the form,
   **Then** the choice is shown read-only with "Only the budget's owner or someone who manages budgets can
   change this."; a request that changes it anyway is refused and changes nothing.
7. **Given** a budget set to Refuse, **When** its owner lowers the budget below its planned cost, **Then** the
   form says before saving "The planned cost is <P>; this budget refuses overspending, so it can't be lower
   than that." and the save is refused; lowering it to the planned cost or above saves.

---

### User Story 2 - Refuse holds whichever way cost arrives (Priority: P2)

Cost reaches a budget not only through the activity form: activities, initiatives and campaigns are moved
and copied, activities are split, records come back from the recycle bin and plans are copied. Every one of
these is refused when it would take a Refuse budget's planned cost over the budget, with the same message.

**Why this priority**: A control with a side door is no control; but these are less frequent actions than
editing an activity.

**Independent Test**: With "Helix launch" set to Refuse and little left to plan, move, copy, restore and
split cost into it; each is refused naming "Helix launch"; the same actions into a campaign set to Warn
succeed.

**Acceptance Scenarios**:

1. **Given** a campaign set to Refuse, **When** an activity or initiative is moved or copied into it, a deleted
   one is restored or undone into it, or the campaign is moved or copied into a plan set to Refuse, **Then**
   each is refused when it would take any Refuse budget it reaches over its budget, and succeeds otherwise.
2. **Given** a plan set to Refuse, **When** an activity is created with a planned cost, or a plan is copied
   with its contents, **Then** the same rule applies to every Refuse budget the new cost reaches.
3. **Given** a budget already over and set to Refuse, **When** a save removes or lowers cost below it (delete,
   move away, lower a planned cost), **Then** it is saved even though the budget stays over.
4. **Given** any save refused for overspending, **Then** nothing of it is saved — no record, no activity-log
   entry, no recycle-bin change.
5. **Given** line items of an activity under a Refuse budget, **When** they are added, re-costed or spread,
   **Then** they are never refused for overspending: they do not change the planned cost under a budget
   (the purchase-request gate of User Story 5 applies instead).

---

### User Story 3 - A global budget owner refuses overspending on a global budget and its allocations (Priority: P2)

Alex manages the global budget "FY2026 Global marketing budget". On its form he chooses Refuse: the plans it
allocates to may not plan more, all together, than the global budget. When he allocates to a plan, the
Allocate dialog lets him choose Warn or Refuse for that plan's budget, which is then read-only on the plan.

**Why this priority**: Global budgets are the top of Shell's budget flow (014); they matter to fewer people
than plan and campaign budgets.

**Independent Test**: Set Refuse on a global budget whose allocated plans plan close to its amount; raise an
activity in an allocated plan past it (refused naming the global budget); allocate to a plan whose planned
cost does not fit (refused); set Refuse in the Allocate dialog and confirm the plan's form shows it locked.

**Acceptance Scenarios**:

1. **Given** Alex may edit a global budget, **When** he chooses Refuse on its form and saves, **Then** any save
   that would take the planned cost of all its allocated plans (in USD at the current rates) over its amount
   is refused naming the global budget, its amount and the overage; allocating to a plan whose planned cost
   no longer fits is refused the same way; lowering the amount below that planned cost is refused.
2. **Given** the Allocate or Edit allocation dialog, **Then** it shows "When planned cost goes over this
   budget" for the plan's budget (the plan's current choice pre-selected); **When** saved, **Then** the plan's
   budget follows it and the plan form shows the choice read-only with "Allocated from '<global budget>' — ask
   <owner> to change it."
3. **Given** an allocation set to Refuse, **When** the allocation's amount is lowered below the plan's planned
   cost, **Then** the dialog says so before saving and the save is refused.
4. **Given** the exchange rates change (feature 016), **Then** nothing already saved is refused; the next save
   that adds cost under the global budget is checked at the new rates.

---

### User Story 4 - Shell chooses the setting new budgets start with (Priority: P2)

Alex opens **Settings › Overspend control** and chooses whether new budgets start on Warn (as delivered) or
Refuse. Budgets that exist keep their own setting; the page says so before saving.

**Why this priority**: Lets Shell make Refuse the norm without asking every owner, while nothing changes for
existing budgets.

**Independent Test**: As Alex, set the default to Refuse; create a plan and a campaign with budgets: their
forms start on Refuse; existing budgets are unchanged; the change is in the activity log. As Priya, the page
is read-only with the reason.

**Acceptance Scenarios**:

1. **Given** Alex, who may change Settings, **When** he opens Settings › Overspend control, **Then** he sees
   "New budgets start with" Warn or Refuse, the note "Existing budgets keep their own setting.", who changed it
   last and when; **When** he chooses Refuse and saves, **Then** it is saved and logged.
2. **Given** the default is Refuse, **When** anyone creates a plan, campaign, initiative or global budget,
   **Then** its form starts on Refuse (and may be changed by whoever may change it); a copy keeps its
   source's setting.
3. **Given** a person who may not change Settings, **When** they open the page, **Then** the setting is shown
   read-only with the reason; a request that changes it anyway is refused.

---

### User Story 5 - A purchase request cannot commit more than its activities plan (Priority: P1)

Priya's activity was approved at 12,000.00 USD; since feature 034 she may still add and re-cost its line items
without re-approval, so they now total 20,000.00 USD. She raises a purchase request on one of them and submits
it: the submission is refused, listed with the other submit problems: "Activity '<name>': line items exceed
its planned cost by 8,000.00 USD — lower them, or raise the activity's planned cost and get it approved
again." This holds whatever the budget's Warn or Refuse setting.

**Why this priority**: It closes a blocking review finding on feature 032: without it an approval of 80,000.00
can carry 200,000.00 of purchase requests, and the purchase request is what commits Shell's money.

**Independent Test**: Approve an activity, add line items costing more than its planned cost, raise a
purchase request on one and send it to GSAP for its SWBS code; Submit is refused naming the activity and the
overage; lower the line items: Submit goes through.

**Acceptance Scenarios**:

1. **Given** a purchase request whose lines draw on line items of an activity whose line items total more than
   its planned cost (negative unassigned planned cost), **When** anyone chooses Submit, Re-submit, Save and
   re-submit or Submit changes, **Then** it is refused and the problem names the activity and the overage,
   together with every other submit problem.
2. **Given** the line items are brought back within the activity's planned cost (or the activity's planned
   cost is raised and approved again), **When** the purchase request is submitted, **Then** this gate no
   longer stops it.
3. **Given** the activity's line items exceed its planned cost, **Then** raising, editing and saving the
   purchase request as Created is still allowed (only submission is gated), as are line-item changes (034
   FR-001).

---

### Edge Cases

- **Budget with no amount**: a record without a budget has nothing to go over; its setting is kept and applies
  once a budget is typed. Typing a budget below the planned cost on a record set to Refuse is refused like
  lowering it.
- **Budget of 0.00**: with Refuse, any planned cost above 0.00 under it is refused.
- **Exactly at the budget**: planned cost equal to the budget is within it; one cent more is over.
- **Already over when Refuse is chosen** (client 011 edge case): allowed; the form says "The planned cost is
  already over this budget by <X>. Anything that adds cost will be refused until it is back within the
  budget."; saves that lower the cost succeed.
- **Several budgets at once**: one save can reach the initiative, campaign, plan and global budget above an
  activity; the refusal lists every Refuse budget it would take over.
- **Warn budgets** never refuse anything and keep today's red figures and over-allocation warnings.
- **Over-allocation** (children's budgets above the parent's): still allowed and shown as today, whatever the
  setting (D-015, D-235).
- **Concurrent saves**: two saves that each fit but together do not — the second is checked against the first
  and refused (saves are applied one at a time).
- **Exchange-rate changes** and imports of GSAP data never refuse; they may leave a global budget over.
- **Restore from the recycle bin** of a record that brings cost back is refused like any other save; the
  record stays in the bin and can be restored once there is room.
- **Copies** keep their source's setting; a copy that would itself be over a Refuse budget is refused.
- **An allocated plan's setting** is changed only through its allocation; removing the allocation keeps the
  plan's setting, which the plan's owner may then change.
- **Purchase requests already approved** before this gate are not touched; the gate applies to the next
  submission (including Submit changes).
- **Underspend** is not flagged (out of scope).

## Requirements *(mandatory)*

### Functional Requirements

**The setting**

- **FR-001**: Every plan, campaign, initiative and global budget MUST carry the setting **When planned cost
  goes over this budget**: **Warn** or **Refuse**. Warn MUST behave exactly as before this feature (D-015,
  D-235).
- **FR-002**: The setting MUST be shown on the plan, campaign, initiative and global budget forms next to the
  budget, and as "Refuses overspending" next to the budget where the record is shown, with each option's
  meaning in plain words.
- **FR-003**: The setting MAY be changed only by the record's owner or a person who manages global budgets
  for the record's plan; for a global budget and for an allocation, by whoever may change them. Everyone else
  MUST see it read-only with the reason, and a change they send MUST be refused with nothing saved.
- **FR-004**: An allocated plan's setting MUST be chosen in the Allocate and Edit allocation dialogs (the
  plan's current choice pre-selected), and shown read-only on the plan's form with the existing reason of an
  allocated budget.
- **FR-005**: Every change of the setting MUST be written to the activity log of its record with who, when,
  the old and the new value, in the same save.

**The organisation default**

- **FR-006**: Settings MUST offer **Overspend control** with "New budgets start with": Warn (as delivered) or
  Refuse, changed only by people who may change Settings, versioned, and logged.
- **FR-007**: A new plan, campaign, initiative or global budget MUST start with the organisation default unless
  its form says otherwise; a copy MUST keep its source's setting; changing the default MUST NOT change any
  existing budget, and the page MUST say so before saving.

**Enforcement**

- **FR-008**: The planned cost under a budget is: for a plan, campaign or initiative, the sum of the planned
  cost of the activities below it, in the plan currency (domain model §4.2); for a global budget, the sum of
  the planned cost of every plan it allocates to, each converted to USD at the current rate (D-303).
- **FR-009**: A save MUST be refused when, after it, the planned cost under a budget set to Refuse would be
  more than the budget and the save raised that planned cost, lowered that budget, gave the record a budget it
  did not have, or created the budget (a copy). The check MUST cover every save that changes planned cost or a
  budget: creating, editing, moving, copying and splitting activities; moving and copying initiatives and
  campaigns; copying plans; restoring and undoing from the recycle bin; editing plan, campaign, initiative and
  global budgets; and creating, changing and removing allocations.
- **FR-010**: A refused save MUST change nothing and MUST answer with a message that names every budget it
  would take over — its level, its name, its amount, the planned cost it would reach and the overage, in the
  budget's currency — and says that its owner refuses overspending.
- **FR-011**: A save that keeps or lowers the planned cost under a Refuse budget, or keeps or raises its
  amount, MUST NOT be refused for that budget, even when the budget is over (switching to Refuse while over,
  deleting or moving cost away).
- **FR-012**: Line-item changes MUST NOT be refused for overspending (they do not change the planned cost
  under a budget); changes of exchange rates and imports MUST NOT be refused.
- **FR-013**: Over-allocation (the children's budgets above the parent's) MUST keep warning without refusing,
  whatever the setting.

**Shown before saving**

- **FR-014**: The activity form MUST show, for every budget above the activity set to Refuse, how much is left
  to plan, and MUST say before saving when the typed planned cost would go over one of them, naming the
  budget and the overage, refusing the save on the form with that message.
- **FR-015**: The plan, campaign, initiative and global budget forms and the allocation dialogs MUST say before
  saving when, with Refuse chosen, the typed budget is below the current planned cost and would be lowered
  (or set) below it, naming the planned cost; and when Refuse is chosen while the planned cost is already over
  the budget, that it is already over and further increases will be refused.
- **FR-016**: Every screen that saves something refused for overspending MUST show the API's message where the
  person is looking, keep what they typed, and let them correct it.

**Purchase requests (amends 032 FR-021, D-349)**

- **FR-017**: The planned cost and the line items of an activity are those of domain model §4.3: line items
  exceed the activity's planned cost when its unassigned planned cost is negative.
- **FR-018**: Raising, editing and saving a purchase request MUST stay allowed while its activities' line
  items exceed their planned cost; only submission is gated.
- **FR-019**: Submit, Re-submit, Save and re-submit and Submit changes of a purchase request MUST be refused
  while any line's activity has line items exceeding its planned cost, whatever the setting of its budgets,
  naming each such activity once with the overage in the plan currency, reported with every other problem of
  032 FR-021.

**Records and traceability**

- **FR-020**: The domain model §4.1, the decision log (D-346 – D-349, Q-024, Q-025), spec 032 FR-021 and the
  feature map MUST state these rules.

### Key Entities *(include if feature involves data)*

- **Budget setting (Overspend)**: Warn or Refuse, held by each plan, campaign, initiative and global budget; an
  allocated plan's is chosen with its allocation.
- **Overspend control (Settings)**: the organisation's setting new budgets start with; versioned and logged.
- **Overage**: for one budget, its amount, the planned cost a save would reach and the difference; part of a
  refusal, never stored.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: With a budget set to Refuse, 100 % of saves that would take its planned cost over it are refused,
  across every kind of change listed in FR-009, and 0 % of saves that keep or lower it are refused.
- **SC-002**: With every budget on Warn, every journey that passed before this feature still passes unchanged.
- **SC-003**: A budget owner switches a budget to Refuse in at most 3 steps from the record (open the form,
  choose Refuse, save) — design system §10.
- **SC-004**: Every refusal names the budget, its amount and the overage, to the cent, matching a hand
  calculation.
- **SC-005**: 0 purchase requests can be submitted while their activities' line items exceed the planned cost.

## Assumptions

- Hive9 and Shell's finance discovery describe hard limits on finance budget rows (Finance module, out of
  scope); MAPP applies the same "warning or block" choice to the planning budgets it has.
- "Budget owner" is the record's owner field, or a person who manages global budgets (they set budgets
  top-down today).
- Refusal is checked when saving, against what is saved at that moment; nothing already saved is undone.
- The activity form's warning repeats the server's check for faster feedback; the server decides.
- Underspend warnings (BRD comment) are out of scope.

## Amendment index

| Feature | Requirement | Change |
|---|---|---|
| 004 | FR (plan budget) | Plan form gains the overspend setting; a Refuse plan budget refuses (037 FR-001 – FR-011) |
| 005 | FR (campaign and initiative budgets) | Campaign and initiative forms gain the setting; Refuse budgets refuse (037 FR-001 – FR-011) |
| 014 | Global budgets and allocations (FR-029 – FR-036) | Global budget form and allocation dialogs gain the setting; D-235's "never blocks" applies to Warn only (037 FR-004, FR-009) |
| 032 | FR-021 | New gate: line items exceeding the activity's planned cost stop submission (037 FR-019) |
| Docs | Domain model §4.1, decision log | Updated (037 FR-020) |
