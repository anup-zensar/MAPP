# Data Model: Approval Rules

**Feature**: `017-approval-rules` · **Plan**: [plan.md](plan.md) · **Contract**: [contracts/openapi.yaml](contracts/openapi.yaml)

Changes to the records of [feature 008's data model](../008-activity-approvals/data-model.md) and one
new record. Conventions are 001's and 008's: `record.Meta`, readable IDs, `ActorRef`, lists never null,
money `{amount, currency}` with decimal-string amounts, dates `YYYY-MM-DD`, timestamps UTC. Everything
008's data model says still holds unless this page changes it.

**Stored data from before this feature** reads with the defaults of §10, so no migration is needed and
every existing workflow and round behaves exactly as before.

---

## 1. Approval workflow — table `approval_workflows` (changed)

New fields (008 §1 fields unchanged):

| Field | JSON | Type | Rules (checked together with 008's, FR-008) |
|---|---|---|---|
| Remind after | `reminderDays` | integer or null | null = no reminders. 1–60 ("Remind after: enter a whole number of days from 1 to 60."). The web form starts new workflows at 3 |
| Escalate after | `escalationDays` | integer or null | null = no escalation. 1–90 ("Escalate after: enter a whole number of days from 1 to 90."); more than `reminderDays` when both are set ("Escalate after must be more days than Remind after."). The web form starts new workflows at 5 |

Field keys of problems: `reminderDays`, `escalationDays`.

## 2. Approval level — inside the workflow (changed)

| Field | JSON | Type | Rules |
|---|---|---|---|
| Name | `name` | text | Unchanged (0–60) |
| Approver source | `approverType` | enum | `people` · `manager` · `role`. Missing or unknown → "Level <n>: choose who approves." |
| Approvers | `approverIds` | list of person ids | `people`: unchanged 008 rules (at least one, known, no duplicate, added ones active) plus 014 FR-028 (added ones hold `approvals.decide`: "<Name> can't approve activities. Give them a role that approves first."). `manager`, `role`: must be empty — the server clears it |
| Role | `roleId` | string | `role` only: required ("Level <n>: choose a role."); a role of 014 ("Level <n>: choose a role from the list."); the role includes `approvals.decide` ("Level <n>: people with the role '<role>' can't approve activities. Choose a role that approves."). Other sources: the server clears it |
| Mode | `mode` | enum | `any` (default) · `all`. Unknown → "Level <n>: choose 'Any one approves' or 'All must approve'." `manager` levels are always stored `any` |
| Only needed when | `appliesWhen` | **Condition** or null | null = always needed. An empty condition → "Level <n>: 'Only needed when' needs at least one condition." |
| Approve automatically when | `autoApproveWhen` | **Condition** or null | null = never automatic. Empty → "Level <n>: 'Approve automatically when' needs at least one condition." |
| Backup approver | `backupApproverId` | person id or "" | Optional; required on **every** level when the workflow's `escalationDays` is set ("Level <n>: choose a backup approver, or turn escalation off."). Known person ("Level <n>: choose the backup approver from the list."); when added or changed by this save, active ("Level <n>: <name> is inactive — choose an active person.") and holding `approvals.decide` ("Level <n>: <name> can't approve activities. Give them a role that approves first."); not one of the level's `approverIds` ("Level <n>: <name> is already an approver of this level — choose someone else.") |

**Condition** (a value; no id):

| Field | JSON | Type | Rules (all messages prefixed "Level <n>: ") |
|---|---|---|---|
| Planned cost at least | `plannedCostAtLeast` | decimal string (USD) or null | ≥ 0, at most two decimals ("enter amounts of 0.00 or more with at most two decimals.") |
| Planned cost below | `plannedCostBelow` | decimal string (USD) or null | Same; when both are set, at-least < below ("the 'at least' amount must be lower than the 'below' amount.") |
| Business unit is one of | `businessUnitIds` | list of option ids | Options of the plan-level Business unit attribute (002, `Attribute.Role = businessUnit`), no duplicates ("choose business units from the list."). Inactive options may stay |
| Activity type is one of | `activityTypeIds` | list of activity type ids | Activity types of 002, no duplicates ("choose activity types from the list."). Inactive types may stay |

At least one part must be set. Field keys: `levels.<n>.approverType`, `levels.<n>.approverIds`,
`levels.<n>.roleId`, `levels.<n>.mode`, `levels.<n>.appliesWhen`, `levels.<n>.autoApproveWhen`,
`levels.<n>.backupApproverId` (n from 1, D-215). One message per condition field even when several of
its parts are wrong (the first in the order of the table).

**Condition matching** (pure function, FR-005): against the **rule facts** of §3 — every set part must
match: `plannedCostAtLeast` ⇒ planned cost ≥ amount; `plannedCostBelow` ⇒ planned cost < amount (an
activity without planned cost matches neither); `businessUnitIds` ⇒ at least one of the plan's business
units is listed; `activityTypeIds` ⇒ the activity's type is listed.

**Condition text** (English, used in rule texts, levels text and history): the set parts in the table's
order joined with " and ": "planned cost is at least $100,000.00" · "planned cost is below $10,000.00" ·
"business unit is Mobility" / "business unit is Mobility or Lubricants" (labels as now, ", " between all
but the last two) · "activity type is Event" / "… is Event or Sponsorship". Amounts are written as in
008's change texts ("$10,000.00").

**Levels text** (008 §1 derived, changed): per level "<label> · <source>[ · all must approve][ ·
automatic when <condition text>][ · only when <condition text>]", levels joined " → ". Source: the
approvers' names (", ", "(inactive)" as today) · "Line manager" · "Role: <role name>". Example:
"Level 1 · Line manager · automatic when planned cost is below $10,000.00 → Level 2 · Alex Morgan · only
when planned cost is at least $100,000.00".

**Warnings** (008 §1 derived, extended): "Level <n> has no active approver" (named people, unchanged);
"Level <n>: the backup approver <name> is inactive"; "Level <n>: the backup approver <name> can't approve
activities"; "Level <n>: role '<name>' no longer exists".

**History kind `approvalWorkflow`** `Describe` (extended): `levels` text = the levels text above;
`reminderDays` → "Remind after", "<n> days" / "Off"; `escalationDays` → "Escalate after", likewise.

## 3. Rule evaluation and approver resolution (at Submit and Re-submit)

**Rule facts** (taken once from the activity, stored in the round as `facts`): planned cost (USD, may
be null), activity type (id, name), the plan's business units (option ids and labels).

For each workflow level, in order (FR-010, FR-011):

1. `appliesWhen` set and not matching → outcome **`skipped`**, reason "only when <condition text>".
2. Else `autoApproveWhen` set and matching → outcome **`autoApproved`**, reason "<condition text>".
3. Else outcome **`decide`**. Candidates, in order, without duplicates:
   - `people`: `approverIds`;
   - `manager`: the owner's `managerId` (none → no candidate);
   - `role`: `Access.RoleHolders(roleId, planId)`, sorted by name.
4. Leave out, recording each in `excluded` with its reason, in this order of reasons: inactive people
   (`inactive`); people without `approvals.decide` (`cannotApprove`, 014 `CanApprove`); and, while 014's
   access rule "Nobody approves an activity they own or submitted" is on, the activity's owner (`owner`)
   and the person submitting (`submitter`).
5. Nobody left → the level's backup approver, when it is **usable** (active, holds `approvals.decide`,
   and not owner or submitter while the rule is on), becomes the only approver (`backupUsed = true`).
   Otherwise the level keeps **no approver** (`approvers` empty, `noApprover = true`): it waits for
   someone allowed to decide on behalf (014), and the confirmation shows the advice below. This does not
   block the submission.
6. `backup` (for escalation) = the backup approver when usable and not among the resolved approvers;
   otherwise null.

**Resolution problems** — only when the source names nobody at all (008 FR-010 (f), field
`approvalWorkflowId`, group `activity`, after 008's workflow problem; one per level):

| Case | Message |
|---|---|
| `manager`, the owner has no line manager | Level <n>: <owner name> has no line manager. Ask an administrator to set one in Settings › People. |
| `role`, role unknown | Level <n>: the role of this level no longer exists. Ask an administrator to change the approval workflow. |
| `role`, no active holder whose scope covers the plan | Level <n>: nobody holds the role '<role name>' for this plan. |

**No-approver advice** (confirmation and chain): when the submitter was the only candidate left out as
owner or submitter: 014's "You are the only approver of level <n>. An administrator will have to approve
on your behalf."; otherwise "Nobody else can approve level <n>. Someone allowed to decide on behalf, such as
an administrator, will have to approve it."

Activity status after Submit: **Submitted**, waiting at the first `decide` level; when there is none,
**Approved** at once (round outcome `approved`, `endedAt` = submit time, `currentLevel` = the last level).

## 4. Approval round — table `approval_rounds` (changed)

New round fields:

| Field | JSON | Type | Rules |
|---|---|---|---|
| Rule facts | `facts` | object | `{plannedCost: Money or null, activityType: {id, name} or null, businessUnits: [{id, label}]}` at submission |
| Remind after / escalate after | `reminderDays`, `escalationDays` | integer or null | Copied from the workflow at submission (008 FR-006: later edits do not apply) |
| Approved at submission | `autoApproved` | yes/no | true when no level needed a decision |

New **RoundLevel** fields (008 §2 `number`, `name`, `approvers` unchanged; `approvers` now holds the
resolved deciders and is empty for `skipped` and `autoApproved` levels):

| Field | JSON | Type | Rules |
|---|---|---|---|
| Approver source | `approverType` | enum | Copy; `people` for old rounds |
| Source text | `source` | text | "" for named people; "line manager of <owner name>"; "role <role name>" |
| Role | `roleId`, `roleName` | text | Copy (role levels) |
| Mode | `mode` | enum | `any` / `all` |
| Outcome at submission | `resolution` | enum | `decide` · `skipped` · `autoApproved` |
| Reason | `ruleText` | text | §3 reason; "" for `decide` |
| Left out | `excluded` | list of `{person: ActorRef, reason: inactive / cannotApprove / owner / submitter}` | §3 step 4 |
| Backup used | `backupUsed` | yes/no | §3 step 5 |
| No approver | `noApprover` | yes/no | §3 step 5: nobody resolved and no usable backup |
| Backup | `backup` | ActorRef or null | §3 step 6 |
| Waiting since | `startedAt` | timestamp or null | Set when the round reaches the level; null before and for skipped/automatic levels |
| Reminded at | `remindedAt` | timestamp or null | Set by the reminder (or by an escalation that replaces it, FR-027) |
| Escalated at | `escalatedAt` | timestamp or null | Set by the escalation |

`currentLevel`: the first `decide` level not yet approved; moving on skips every level whose
`resolution` is not `decide` (FR-015).

**Level done** (FR-015): `mode = any` → the first approval; `mode = all` → every resolved approver has an
approval counted for them (their own, or their delegate's); in both modes an approval with capacity
`backup` or `onBehalf` completes it. A level with `noApprover` is done by an `onBehalf` (or, after
escalation, `backup`) approval.

**No approver shown** (API `noApprover` on chain levels and rows, derived): the stored `noApprover`, or
every waiting approver of the current level is now inactive or without `approvals.decide`; `advice` is
§3's no-approver advice in both cases.

## 5. Approval history entry — table `approval_actions` (changed)

New `action` values: **`autoApproved`** and **`skipped`** (one per such level, at submission, actor
NorthStar `{id: "system", name: "NorthStar"}`), **`reminded`** and **`escalated`** (actor NorthStar).

New fields:

| Field | JSON | Type | Rules |
|---|---|---|---|
| Capacity | `capacity` | enum or "" | Decisions only: `approver` · `delegate` · `backup` · `onBehalf`; "" for other actions and for entries written before this feature |
| As delegate of | `asDelegateOf` | ActorRef or null | `delegate` decisions: the approver they acted for |
| Rule | `rule` | text | `autoApproved`, `skipped`: the reason of §3 |
| Escalated to | `escalatedTo` | ActorRef or null | `escalated` only |
| Waited days | `waitedDays` | integer | `reminded`, `escalated`: whole days waited; 0 otherwise |

`onBehalfOf` is set only for capacity `onBehalf` (the approvers still waiting; empty on a `noApprover`
level). History texts are built by the web from these fields (plan *User-visible messages*).

**Counted approvals of an "all" level**: an `approved` entry of the round and level counts for X when
(`capacity` ∈ {`approver`, ""} and `actor` = X) or (`capacity` = `delegate` and `asDelegateOf` = X).

## 6. Who may decide (FR-017, FR-018, FR-020) — 014's check order, extended

Approve and decline run 014's check order (014 plan, "Approve/decline check order"); 017 extends steps 3
and 4. First failure wins:

1. Activity not visible to the person → 014's 403 message 4. (Visibility includes FR-021: current
   delegates of a waiting approver and the backup approver of an escalated level.)
2. No open round, or the level already decided → 008's answers (FR-018 of 008, unchanged).
3. **Could decide** — *waiting* = the current level's resolved approvers with no approval counted for
   them (§5); the first of:
   - a. person ∈ *waiting* and holds `approvals.decide` → **approver**;
   - b. a **current** delegation (§7) from someone in *waiting* (level order) to the person, who holds
     `approvals.decide` → **delegate** of that approver (the viewer's `asDelegateOf` lists every such
     approver);
   - c. level escalated, person = round level `backup`, holds `approvals.decide` → **backup**;
   - d. person holds `approvals.decideOnBehalf` for the plan → **onBehalf**, on behalf of *waiting*;
   - e. person is a resolved approver with an approval counted → 403 `forbidden` "You already approved
     level <n>. Waiting for <names of waiting>.";
   - f. otherwise → 014's 403 message 5 "Only the approvers of level <n> (<names>) can approve or decline
     this activity." (`<names>`: the level's resolved approvers; "nobody" when `noApprover`).
4. **Segregation of duties** (014 rule on): person = activity owner → 014's 403 message 6 "You own this
   activity, so someone else must approve it."; person = round submitter → 403 "You submitted this
   activity, so someone else must approve it."
5. Decide.

`actions.approve` / `actions.decline` availability runs the same steps (014): reason = the step 3e or step
4 message; an empty reason (buttons hidden) when step 3f fails. **Waiting for me** = steps 3a–3c passing
and step 4 passing (never 3d). Submit and recall permissions are 014's (FR-027, `activities.submit`).

Permissions of the signed-in person come from 014's `authz` grants in the request context; permissions
of other people (resolution, backup, delegate) from `Access.CanApprove` (plan *Ports*). Internal calls
without grants (seed, tests without grants) behave as "everything allowed, rule on".

## 7. Approval delegation — table `approval_delegations` (new), kind `approvalDelegation`

| Field | JSON | Type | Mark | Rules (all at once) |
|---|---|---|---|---|
| ID | `id` | string | calc | `DLG-1001`, … (counter `DLG`) |
| Delegating person | `personId` | person id | R | Defaults to the signed-in person. Another person only with `workflows.manage` (014's 403 refusal). Known ("Choose a person from the list."), active ("<name> is inactive — choose an active person.") |
| Delegate | `delegateId` | person id | R | "Choose who approves for you." · known · active · holds `approvals.decide` ("<name> can't approve activities. Choose someone with a role that approves.") · not the delegating person ("You can't delegate to yourself." / for someone else: "<name> can't delegate to themselves.") |
| From | `startDate` | date | R | "Enter the first day." Today or later on create and while planned ("The first day must be today or later."); cannot change once current ("The delegation has started, so its first day can't change.") |
| To | `endDate` | date | R | "Enter the last day." ≥ `startDate` ("The last day must be on or after the first day."); ≥ today ("The last day must be today or later."); at most 90 days from `startDate` inclusive ("A delegation can last at most 90 days.") |
| Note | `note` | text | — | Trimmed, 0–500 ("Use 500 characters or fewer.") |
| Cancelled | `cancelledAt`, `cancelledBy` | timestamp / ActorRef or null | calc | Set by **Cancel delegation** |
| *bookkeeping* | | `record.Meta` | calc | Version checked on update and cancel (D-030) |

**No overlap**: two delegations of the same person that are not cancelled must not share a day
("You already delegate to <delegate> on some of these days. Change that delegation instead." /
when acting for someone else: "<person> already delegates to <delegate> on some of these days. Change
that delegation instead."). Field `startDate`.

**State** (derived; "today" = the clock's date in UTC): `cancelled` (cancelledAt set) · `ended`
(endDate < today) · `current` (startDate ≤ today ≤ endDate) · `planned` (startDate > today).

**Changes**: only the delegating person or someone holding `workflows.manage` (else 403 `forbidden` "Only
<person> or someone who can change approval workflows can change this delegation."); only `planned` and
`current` ones (else 409 `not_allowed` "Ended and cancelled delegations can't be changed. They are kept for
the audit trail."). Delegations are never deleted (Proposed 017-P9). `scope=all` lists need
`workflows.manage` (014's 403 refusal).

**Effect** (FR-023): a delegation counts only on days it is `current`, and only while the delegate is active
and holds `approvals.decide`.

**Record name** (log, kind): "<person name> → <delegate name>". `Describe`: `delegateId` → "Delegate",
`startDate` → "From", `endDate` → "To", `note` → "Note", `cancelledAt` → "Status" ("Cancelled").
Log entries: `created`; `changed` (history.Diff); `statusChanged` verb `cancelled`.

**Life cycle**: (none) ─create→ planned/current ─(date passes)→ current/ended; planned/current ─Cancel→
cancelled.

## 8. Notification — table `notifications` (changed)

New `kind` values: `autoApproved`, `reminder`, `escalated`. Notifications of NorthStar events have actor
NorthStar. Recipients (one per person, the actor removed, duplicates removed; 008 §4 rows unchanged):

| Event | Recipients and kind |
|---|---|
| A level is reached (submit, or the level before approved) | Its resolved approvers → `approvalNeeded` (as today) **and** the current delegates of each → `approvalNeeded`, text + " — you approve as delegate of <name>" |
| Approved at submission | Owner, when not the submitter → `autoApproved` |
| Approval on an "all" level that does not complete it | nobody |
| Reminder | Approvers still waiting and their current delegates → `reminder` |
| Escalation | Backup approver → `escalated` ("Approval escalated to you"); approvers still waiting and the owner → `escalated` ("Approval escalated") |

Titles and texts: plan *User-visible messages*.

## 9. Person — table `people` (changed, feature 003)

| Field | JSON | Type | Rules |
|---|---|---|---|
| Line manager | `managerId` | person id or "" | Optional. Known person ("Choose the line manager from the list."); when added or changed, active ("<name> is inactive — choose an active person."); not the person ("A person can't be their own line manager."); no loop through the chain of managers ("<manager> reports to <person>, so <person> can't report to <manager>." — the first person found in the chain is named) |

Derived: `manager` (PersonRef or null), `reportCount` (active and inactive people whose `managerId` is
this person). 003's usage count gains role `manager` ("Line manager of 2 people") for details and the
Deactivate dialog. Log: 003's `changed` entry with "Line manager", old → new names. Changing it needs
014's `people.manage` (Table C rule of the people operations, unchanged).

## 10. Reading data stored before this feature

| Stored value missing | Read as |
|---|---|
| Level `approverType`, `mode`, `appliesWhen`, `autoApproveWhen`, `backupApproverId`, `roleId` | `people`, `any`, null, null, "", "" |
| Workflow `reminderDays`, `escalationDays` | null, null |
| RoundLevel `approverType`, `mode`, `resolution`, `ruleText`, `source`, `excluded`, `backupUsed`, `noApprover`, `backup`, `startedAt`, `remindedAt`, `escalatedAt` | `people`, `any`, `decide`, "", "", [], false, false, null, null (waiting since then uses 008's rule: submission for level 1, else the previous level's approval), null, null |
| Round `facts`, `reminderDays`, `escalationDays`, `autoApproved` | null, null, null, false |
| Action `capacity`, `asDelegateOf`, `rule`, `escalatedTo`, `waitedDays` | "", null, "", null, 0 |
| Person `managerId` | "" |

## 11. Monitoring values (derived, FR-029, FR-030)

Over the **open rounds of stored activities** that the viewer may see (the rows of **All open**):

| Value | Rule |
|---|---|
| Waiting days | ⌊(now − waiting since) / 24 h⌋ of the current level |
| Overdue | waiting days ≥ round `escalationDays`; without it ≥ `reminderDays`; without both never |
| Escalated (card) | open rounds whose current level has `escalatedAt` |
| Ageing buckets | `under1` (0), `days1to2` (1–2), `days3to5` (3–5), `days6to10` (6–10), `over10` (> 10) — labels "Under 1 day", "1–2 days", "3–5 days", "6–10 days", "More than 10 days" |
| Waiting by level | groups by (round workflow id, workflow name at submission, current level number, level label): `waiting`, `overdue`, `oldestWaitingSince`; ordered by workflow name, then level |
| Average time to decision | over rounds of stored activities with outcome `approved` or `declined`, `endedAt` in [monthStart, monthEnd) and `autoApproved` false: mean of (endedAt − submittedAt) in days, rounded half up to one decimal ("2.4"); null when there are none |
| Decided this month | the count of those rounds |
| Approved automatically this month | rounds with `autoApproved` true and `endedAt` in the month |

## 12. Readable-ID counters added

| Prefix | Record |
|---|---|
| `DLG` | Approval delegation |

## 13. Demo data (seed, FR-037)

Times as 008 (offsets from the seed's `now`). Seeded statuses, rounds and notifications of 008 stay.

| Record | Seed |
|---|---|
| Line managers | Priya Raman → Daniel Okoro · Sara Novak → Daniel Okoro · Mei Lin → Carlos Mendes · Daniel Okoro → Alex Morgan · Carlos Mendes → Alex Morgan · Alex Morgan → none |
| `WF-1001` Lubricants EU approval | Unchanged levels (people, any); `reminderDays` 1, `escalationDays` 2 (short, so a demo shows both on the first run); level 1 backup Alex Morgan |
| `WF-1002` China BC approval workflow | Unchanged levels; `reminderDays` 3, `escalationDays` 5; level 1 backup Alex Morgan, level 2 backup Carlos Mendes |
| `WF-1003` **Standard marketing approval** (new, active, used by no seeded activity) | Description "Approved automatically below 10,000 USD, otherwise by the owner's line manager. The marketing administrator joins from 100,000 USD; Mobility activities also need a second check by two people." Level 1 "Line manager": `manager`, auto when below 10000.00, backup Alex Morgan. Level 2 "Marketing administrator": people [Alex Morgan], only when at least 100000.00, backup Carlos Mendes. Level 3 "Mobility second check": people [Daniel Okoro, Alex Morgan], `all`, only when business unit is Mobility, backup Carlos Mendes. Remind after 3, escalate after 5. (Every named and backup approver holds `approvals.decide` in 014's seed: Alex Administrator, Carlos Global planner, Daniel Approver.) |
| Seeded open rounds | Trade magazine ads round 2 (WF-1001): level 1 `startedAt` = its submission (now − 2 d), backup Alex Morgan, timings 1/2 → overdue, escalation due on the first run. Fleet card launch (WF-1002): level 1 `startedAt` = now − 4 d, level 2 `startedAt` = now − 2 h, backups as the workflow, timings 3/5. All seeded round levels `resolution` `decide`, `mode` `any`; `facts` from the activity |
| `DLG-1001` | Daniel Okoro → Carlos Mendes, from today + 7 d to today + 11 d, note "Annual leave" (planned: no effect on today's lists) |
| Activity log | 011's demo-history step adds `created` for DLG-1001 by Daniel Okoro at now − 1 d |
