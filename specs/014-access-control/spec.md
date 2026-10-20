# Feature Specification: Access Control and Global Budgets

**Feature Branch**: `014-access-control`

**Created**: 2026-10-20

**Status**: Draft (defaults proposed for Shell to confirm, see *Assumptions*)

**Input**: Product owner decision of 2026-10-20: build every MVP gap of Shell's BRD now, with documented,
reversible defaults. This feature closes **BUS.ADM.020** (team- and role-based access control, configurable
through access rules, view-only versus edit) and **BUS.BUD.040** (global and local budget allocation with the
right visibility for global and local budget owners). It also provides what **BUS.WFL.010** needs from roles
(role-based approvals, segregation of duties), the access scoping of **BUS.PRF.030** ("Global can view all data
for all markets vs local market can view all data for their market"), the self-service role administration of
**BUS.ADM.010**, and the "who may see financial data" rule that **BUS.FIN.080** will use.

**References**: BRD [brd-requirements.md](../../docs/reference/brd-requirements.md) · Gap analysis
[§3.1, §3.3, §3.10, §5, §6 Q-010, Q-011, Q-006](../../docs/reports/brd-gap-analysis.md) · Domain model
[§1, §2, §4.1, §6, §7, §8](../../docs/product/domain-model.md) · Demo data
[demo-users-and-data.md](../../docs/product/demo-users-and-data.md) · Hive9 roles (functional analysis §F4) ·
Decisions D-004, D-015, D-020, D-035, D-057, D-172, D-205, D-206 · ADR-0005 · Features 002–012 (their deferred
role rules: 002 FR-008, 003 FR-026–FR-027, 004 FR-004 and FR-032, 005 FR-031, 006 and 007 Assumptions,
008 FR-017 and FR-021, 009 Assumptions, 011 Assumptions) · Sibling features written at the same time: 015
financial transactions (FR-028), 016 planning and currency extensions, 017 approval rules (FR-017, FR-018,
FR-021, FR-025)

## Words used in this feature

| Word | Meaning |
|---|---|
| **Permission** | One thing a person may do, such as "Edit plan details" or "Approve or decline when named as an approver". The list of permissions is fixed by NorthStar (FR-001, Table A). |
| **Role** | A named set of permissions, such as *Planner*. Administrators create and change roles in Settings, without a release. |
| **Scope** | Where a role applies: **All markets**, **My teams** (the teams the person belongs to at that moment) or **Chosen teams**. |
| **Market** | A team. Plans belong to teams (004), so "a market's plans" are the plans of its team. |
| **Global budget** | An amount of money for one year, owned by a global budget owner, shared out to market plans. |
| **Allocation** | The part of a global budget given to one plan. It becomes that plan's budget. |
| **Global / local budget owner** | A person with a *Global planner* role for All markets, and a plan owner whose roles cover only their own teams. |

## User Scenarios & Testing *(mandatory)*

The scenarios use the demo data (FR-040): **Alex Morgan** Administrator (All markets) · **Carlos Mendes** Global
planner (All markets) · **Daniel Okoro** Planner and Approver (My teams: Lubricants Europe) · **Priya Raman**
Planner (My teams: Lubricants Europe) · **Mei Lin** Planner (My teams: Mobility China) · **Sara Novak** Viewer (My
teams: Lubricants Europe, Mobility China). Plans: *FY2026 Lubricants Europe* (team Lubricants Europe),
*FY2026 Mobility China* (Mobility China), *FY2027 Global Brand* (Global Marketing Ops).

### User Story 1 - Local people see their own markets, global people see every market (Priority: P1)

A local planner opens NorthStar and sees only the plans of their own teams, with every count, total, list and
export limited to them. A global planner sees every market. Nobody can open a plan outside their access by
typing its address.

**Why this priority**: It is Shell's core access rule (BUS.ADM.020, BUS.BUD.040, BUS.PRF.030) and every other
story depends on it.

**Independent Test**: Sign in as Priya, Mei, Sara and Carlos in turn and compare the Plans list, Home, All work,
Approvals, Activity log and Recycle bin with the seed; open a plan of another market by its address.

**Acceptance Scenarios**:

1. **Given** the seed, **When** Priya opens Plans, **Then** she sees only *FY2026 Lubricants Europe*, the count
   reads "1 plan", and the summary cards add up that plan only.
2. **Given** the seed, **When** Carlos opens Plans, **Then** he sees all three plans.
3. **Given** the seed, **When** Mei types the address of *FY2026 Lubricants Europe* or one of its activities,
   **Then** NorthStar shows "You don't have access to this plan. Ask an administrator if you need it." with a
   link to Home, and shows nothing of the record (not even its name).
4. **Given** the seed, **When** Priya opens Home, All work (any view, including Budget & cost and Calendar), and
   exports All work, **Then** every row, count and total covers *FY2026 Lubricants Europe* only, and the export
   contains no other plan.
5. **Given** the seed, **When** Mei opens the Activity log and the Recycle bin, **Then** she sees entries about
   *FY2026 Mobility China* and its contents only, and no entry about Settings, people, teams or roles.
6. **Given** Sara (Viewer, two teams), **When** she opens Plans, **Then** she sees the Lubricants and Mobility
   plans and not *FY2027 Global Brand*.

---

### User Story 2 - View-only versus edit, explained on screen and enforced by the server (Priority: P1)

What a person may not do is not offered, and when an action is shown but not possible for this record, it says
why. If someone tries anyway (an old page, a direct API call), the server refuses with a plain message and
changes nothing.

**Why this priority**: The client's comment on BUS.ADM.020: "flexibility to control who can edit vs who can only
view". Without server enforcement there is no access control.

**Independent Test**: Sign in as Sara and open a plan, a campaign, an activity and a line item; then send the
edit, delete and submit requests directly to the API as Sara.

**Acceptance Scenarios**:

1. **Given** Sara, **When** she opens *FY2026 Lubricants Europe*, **Then** the page shows every value, no Edit,
   Delete, Copy, Publish, Add campaign or row actions, and one note: "View only — your role (Viewer) lets you
   see this plan but not change it. Ask an administrator (Alex Morgan) if you need to change it."
2. **Given** Sara, **When** she sends `PATCH /api/activities/{id}` for a Lubricants activity, **Then** the API
   answers 403 with code `forbidden` and "Changing activities and line items isn't part of your role
   (Viewer). Ask an administrator if you need it.", and the activity is unchanged.
3. **Given** Priya (Planner: no plan delete), **When** she opens the actions of her plan, **Then** **Delete
   plan** is shown disabled with "Deleting plans isn't part of your role (Planner). Ask an administrator if you
   need it.", while Edit, Copy and Publish work.
4. **Given** Priya, **When** she creates a plan, **Then** the Teams field offers only Lubricants Europe, and
   saving a plan with only Mobility China is refused: "You can only choose your own teams: Lubricants Europe."
5. **Given** Priya, **When** she opens Settings, **Then** she can read every list, sees "View only — only
   administrators change Settings. Ask Alex Morgan if something needs to change." and has no New, Edit, Delete
   or status actions; Settings › Roles and access is not listed.
6. **Given** Priya moves an activity, **When** the destination list opens, **Then** only initiatives of plans
   where she may change activities are listed.

---

### User Story 3 - An administrator gives people roles and scopes (Priority: P1)

In **Settings › Roles and access › People**, an administrator sees everyone with their roles, scopes and teams,
and changes a person's roles in a side panel. The change applies on that person's next action, without signing
in again.

**Why this priority**: Access must be configurable by Shell, not by a developer (BUS.ADM.020 "configurable
through access rules rather than code", BUS.ADM.010).

**Independent Test**: As Alex, give Sara the Planner role for Mobility China only; as Sara, edit a Mobility
activity and fail to edit a Lubricants one.

**Acceptance Scenarios**:

1. **Given** Alex, **When** he opens Settings › Roles and access, **Then** the People tab lists all six demo
   people with their roles and scopes ("Planner · My teams", "Global planner · All markets"), with search and a
   role filter that includes "No role".
2. **Given** Alex edits Sara's roles, **When** he ticks Planner with scope **Chosen teams: Mobility China** and
   saves, **Then** Sara's row reads "Viewer · My teams, Planner · Mobility China" and the summary of what she can
   do has the line "In Mobility China: create and copy plans, edit plan details, publish and unpublish plans, …".
3. **Given** that change, **When** Sara (already signed in) edits a Mobility activity, **Then** it saves; **When**
   she opens a Lubricants activity, **Then** it is view only.
4. **Given** Alex is the only administrator, **When** anyone removes his Administrator role or deactivates him,
   **Then** it is refused: "Alex Morgan is the only administrator. Give someone else the Administrator role
   first."
5. **Given** Daniel is the level-1 approver of "Lubricants EU approval" with 2 activities waiting, **When** Alex
   removes Daniel's Approver role, **Then** before saving he sees "Daniel Okoro is an approver in 1 approval
   workflow and has 2 activities waiting. Without a role that approves, an administrator will have to decide
   on their behalf." with **Keep editing** and **Save changes**.
6. **Given** a new person is added in Settings › People, **When** they first sign in, **Then** they hold the
   role set in the Rules tab ("Role for new people", default Viewer · My teams).

---

### User Story 4 - An administrator changes what a role may do (Priority: P2)

In **Settings › Roles and access › Roles**, an administrator opens a role and ticks or unticks permissions in a
grouped list, creates a new role (empty or as a copy), renames it or deletes it when nobody holds it.

**Why this priority**: Shell's role catalogue is not confirmed yet (Q-010); the defaults must be changeable by
Shell without code.

**Independent Test**: As Alex, untick "Publish and unpublish plans" on Planner; as Priya, try to publish.

**Acceptance Scenarios**:

1. **Given** Alex opens the Planner role, **When** he chooses **Edit**, unticks "Publish and unpublish plans" and
   saves, **Then** Priya's **Publish** is disabled with its reason and the API refuses publishing for her.
2. **Given** Alex chooses **New role** as a copy of Viewer named "Finance reviewer", **When** he saves it,
   **Then** it is listed with "0 people" and can be given to people at once.
3. **Given** the Administrator role, **When** Alex opens it, **Then** every permission is ticked, nothing can be
   changed, and a note says "The Administrator role always has every permission and can't be changed."
4. **Given** the Viewer role is held by Sara, **When** Alex deletes it, **Then** it is refused: "'Viewer' is given
   to 1 person. Take it away from them first."
5. **Given** an unused role "Finance reviewer", **When** Alex deletes it, **Then** it goes to the recycle bin with
   Undo.

---

### User Story 5 - Approvals follow roles and segregation of duties (Priority: P2)

Only the named approvers of the current level decide, unless someone is allowed to decide on their behalf; nobody
approves an activity they own or submitted. Feature 017 (approval rules) adds delegates, backup approvers and role
approvers on top of these rules; this story is what 014 provides on its own.

**Why this priority**: BUS.WFL.010 asks for role-based approvals and segregation of duties; 008 left "anyone may
approve" until roles existed (D-020).

**Independent Test**: Submit a Lubricants activity owned by Daniel; Daniel cannot approve it; Alex approves on
his behalf; Priya cannot approve at all.

**Acceptance Scenarios**:

1. **Given** a submitted Lubricants activity owned by Priya at level 1 (Daniel), **When** Priya opens it, **Then**
   Approve and Decline are not offered and the API refuses them: "Only the approvers of level 1 (Daniel Okoro)
   can approve or decline this activity."
2. **Given** a submitted Lubricants activity owned by Daniel at level 1 (Daniel), **When** Daniel opens it,
   **Then** Approve and Decline are disabled with "You own this activity, so someone else must approve it."
3. **Given** the same activity, **When** Alex (Administrator) approves, **Then** the confirmation says it is
   recorded on behalf of Daniel Okoro and the history shows "Alex Morgan on behalf of Daniel Okoro".
4. **Given** the Rules tab, **When** Alex turns off "Nobody approves an activity they own or submitted",
   **Then** Daniel can approve his own activity and the change is in the activity log.
5. **Given** Daniel submitted an activity owned by Priya that waits at level 1 (Daniel), **When** Daniel opens it,
   **Then** Approve and Decline are disabled with "You submitted this activity, so someone else must approve it."
6. **Given** Alex edits "Lubricants EU approval", **When** he adds Sara (Viewer) as an approver, **Then** the field
   says "Sara Novak can't approve activities. Give them a role that approves first." and nothing is saved.
7. **Given** Mei (Planner) on a Mobility activity in Created, **When** she submits it, **Then** it is submitted;
   **Given** Sara, **Then** Submit is not offered.

---

### User Story 6 - Global budget allocated to market plans (Priority: P2)

A global planner creates a global budget for a year and allocates it to market plans. Each allocation becomes the
plan's budget, which local planners see but cannot change. The global view shows, per market, allocated,
planned and remaining amounts; local owners see only their own allocation.

**Why this priority**: BUS.BUD.040 ("Global/Local budget allocation with appropriate visibility for global and
local budget owners respectively").

**Independent Test**: As Carlos, raise the Lubricants allocation; as Priya, see the new plan budget, read-only,
and fail to change it; as Mei, fail to open the global budget.

**Acceptance Scenarios**:

1. **Given** the seed, **When** Carlos opens **Global budgets** › *FY2026 Global marketing budget*, **Then** he
   sees Global budget 2,500,000.00 USD, Allocated 2,000,000.00, Unallocated 500,000.00, and one row per plan
   (*FY2026 Lubricants Europe* 1,200,000.00 and *FY2026 Mobility China* 800,000.00) with teams, owner, status,
   planned cost, left to plan and a budget-used bar.
2. **Given** that page, **When** Carlos changes the Lubricants allocation to 1,300,000.00, **Then** the plan's
   budget becomes 1,300,000.00 at once, and the activity log shows the change on the global budget and on the
   plan ("Plan budget 1,200,000.00 USD → 1,300,000.00 USD — allocated from 'FY2026 Global marketing budget'").
3. **Given** Priya opens her plan, **When** she edits it, **Then** Plan budget is read-only with "Allocated from
   'FY2026 Global marketing budget' — ask Carlos Mendes to change it." and a request that changes it is refused
   with the same sentence.
4. **Given** the seed, **When** Mei looks for Global budgets, **Then** it is not in her sidebar and its address
   shows the no-access page; her plan's budget card shows "Allocated from 'FY2026 Global marketing budget'".
5. **Given** allocations add up to more than the global budget, **When** Carlos saves, **Then** it saves and
   Unallocated shows in red "Over-allocated by 100,000.00 USD" (never blocks, as D-015).
6. **Given** Carlos removes the Mobility allocation, **When** Mei edits her plan, **Then** Plan budget is editable
   again and keeps 800,000.00.

---

### User Story 7 - Every access change is in the audit trail (Priority: P3)

Every change to roles, to a person's roles, to the access rules, to global budgets and to allocations is in the
activity log with who, when and old → new. The Roles and access page shows the latest access changes.

**Why this priority**: BUS.WFL.010 asks for an audit trail and compliance controls; BUS.ADM.010 asks for
permission and audit management by Shell.

**Independent Test**: Make one change of each kind as Alex and find each one in the activity log and in "Recent
access changes".

**Acceptance Scenarios**:

1. **Given** Alex gives Sara the Planner role, **When** he opens the activity log, **Then** the entry reads "Alex
   Morgan changed person 'Sara Novak'" with "Roles: Viewer · My teams → Viewer · My teams, Planner · Mobility
   China".
2. **Given** Alex unticks a permission of Planner, **Then** the entry on role 'Planner' lists "Permissions:
   removed Publish and unpublish plans".
3. **Given** the Roles and access page, **Then** "Recent access changes" lists the latest 20 access changes,
   newest first, each linking to its log entry.
4. **Given** Priya, **When** she opens the activity log, **Then** she does not see these entries.

---

### Edge Cases

- **No role**: a person without any role can sign in; Home says "You don't have access to any plans yet. Ask
  an administrator (Alex Morgan) to give you a role." Navigation shows Home, My profile and Settings (read only).
- **No team**: a Planner with scope My teams and no team cannot create plans: **New plan** is disabled with
  "Plans belong to teams, and you are not in a team yet. Ask an administrator to add you to one."
- **Plan without teams** (a draft): it is visible to people with All markets scope and to the person who created
  it, who may change it as their roles allow until teams are chosen.
- **Keeping access**: a person whose plan scope is not All markets must keep at least one of their teams on a plan
  they edit: "Keep at least one of your teams (Lubricants Europe) on the plan, or you won't be able to see it."
  Teams outside their scope already on the plan stay as they are.
- **Owners and named approvers**: a person always sees a plan they own, an activity they own and an activity they
  are named to approve in any round (that activity, its line items and its approval only, read-only unless
  their roles allow more). Breadcrumb parents they cannot open are shown as plain text.
- **Role changed while a page is open**: the next action is refused with the server's message; the web app
  reloads the person's access and shows the page as it now applies.
- **Removing your own Administrator role** while another administrator exists: allowed after "You will no
  longer be able to change Settings, people or access." confirmation.
- **Chosen teams scope with a team that is deleted**: the team leaves the scope; if no team is left, the role
  gives nothing and the People tab shows "No teams left in this scope".
- **Inactive team**: membership of an inactive team still counts for My teams, so older plans stay reachable.
- **Owner or submitter is the only approver of a level** (segregation of duties on): Submit shows the advice
  "You are the only approver of level 1. An administrator will have to approve on your behalf."; the submission
  is not blocked.
- **Approver loses the approving permission** with activities waiting: the chain shows "(can't approve — no
  approving role)" next to the name; someone allowed to decide on behalf can act.
- **Notifications and links** to records the person can no longer see open the no-access page.
- **Recycle bin**: an entry is listed only when the person can see its record; restoring needs the delete
  permission of that level in the record's plan (or, for Settings, people, roles and global budgets, the
  permission that manages them); permanent delete and emptying the bin need "Delete permanently".
- **Activity log of deleted plans**: entries about a plan in the recycle bin follow that plan's teams; entries
  about a permanently deleted plan are shown only to people with All markets.
- **Global budget of another year**: only plans of the budget's year can be allocated: "Choose a plan of 2026."
- **Plan already allocated** from another global budget: "'FY2026 Lubricants Europe' already has an allocation
  from 'FY2026 Global marketing budget'."
- **Copying an allocated plan** copies its budget as a typed amount; the copy has no allocation.
- **Deleting an allocated plan** takes its allocation to the recycle bin with it; restoring the plan brings the
  allocation back, unless the global budget no longer exists ("The allocation was not restored because its
  global budget was deleted.").
- **Global budget with allocations** cannot be deleted: "'FY2026 Global marketing budget' has allocations to 2
  plans. Remove them first."
- **Global budget page for someone with a team scope**: rows only for plans they can see, plus one row "Other
  markets (n plans)" with their total.

## Requirements *(mandatory)*

### Functional Requirements

**Permissions, roles and scopes**

- **FR-001** (BUS.ADM.020): NorthStar MUST decide what each person may see and do from **permissions**. The
  permissions are the fixed list of Table A; nothing else grants or refuses an action.
- **FR-002** (BUS.ADM.020): A **role** MUST be a named set of permissions. A person MAY hold several roles; they
  may do what any of their roles allows (the union).
- **FR-003** (BUS.ADM.020): NorthStar MUST be delivered with five roles and the permissions of Table B:
  **Administrator, Global planner, Planner, Approver, Viewer**. Administrators may change all of them except
  Administrator (FR-018).
- **FR-004** (BUS.ADM.020, BUS.BUD.040): Every role a person holds MUST have a **scope**: **All markets**, **My
  teams** (every team the person belongs to at the moment of the action, active or inactive) or **Chosen teams**
  (a list of teams). Scoped permissions (Table A) apply only to plans in scope; administration permissions
  always apply everywhere. The Administrator role is always All markets.
- **FR-005** (BUS.ADM.020, BUS.BUD.040): A plan MUST be in scope of a role when the scope is All markets, when one
  of the plan's teams is in the scope, or when the plan has no team and the person created it. A market is a
  team.

**Visibility (who sees what)**

- **FR-006** (BUS.ADM.020, BUS.BUD.040, BUS.PRF.030): A person MUST see a plan and everything in it only when a
  role gives them "View plans" for it or they own the plan. They MUST also see an activity they own or are named
  to approve in any approval round (that activity, its line items and its approval).
- **FR-007** (BUS.ADM.020, BUS.PRF.030): Every list, count, total and export of planning data MUST include only
  what the person can see: Plans list and its cards; plan, campaign and initiative pickers and move or copy
  destinations; every Home card; All work (rows, totals, Budget & cost, Calendar, record search, export);
  Approvals (All open, counts); the Activity log and every record's History; the Recycle bin; global budget rows
  (FR-033). The "My teams" switches of 004, 009 and 010 keep working as a narrowing inside what the person can
  see.
- **FR-008** (BUS.ADM.020): Opening a record the person cannot see MUST show "You don't have access to this
  <record type>. Ask an administrator if you need it." and nothing of the record. Records that do not exist keep
  their "not found" page.

**View versus edit**

- **FR-009** (BUS.ADM.020): Every action MUST need its permission of Table A for the record's plan; Table C of the
  plan lists, for every API operation of features 001–014, the rule that applies.
- **FR-010** (BUS.ADM.020): The server MUST refuse an action the person may not take with **403 forbidden** and a
  plain-language message (*User-visible messages* in the plan), and change nothing. An operation without a
  declared rule MUST be refused (fail closed), so a new operation cannot be forgotten.
- **FR-011** (BUS.ADM.020, constitution III): On screen, when the person may **change nothing** in the page's area
  (a plan and its contents, Settings, a global budget), every change action MUST be hidden and one **View only**
  note MUST say which roles they have and which administrators to ask. When the person may change some things
  but not this one, the action MUST stay visible, disabled, with its reason (design system §6.6). The same
  applies to actions blocked for one record (segregation of duties, an allocated budget).
- **FR-012** (BUS.ADM.020): Sidebar entries and Settings cards of pages the person cannot use MUST NOT be shown:
  Global budgets without "See global budgets"; Settings › Roles and access without "Give roles and change
  roles and access rules". Their addresses show the no-access page of FR-008.
- **FR-013** (BUS.ADM.020): When a person whose plan permission is not All markets creates or edits a plan, the
  Teams field MUST offer only teams in that scope, MUST keep teams outside it that the plan already has, and
  MUST keep at least one team in scope (Edge cases).
- **FR-014** (BUS.ADM.020): Move and copy destinations MUST list only places where the person may change records
  of that level; the server MUST check both the source and the destination.
- **FR-015** (BUS.ADM.020, BUS.FIN.080): Financial data MUST need "See financial data": today, the "From finance"
  sections of activities and line items; with feature 015 (financial transactions), every Financials card,
  column, filter and page. Linking transactions to line items MUST need "Link financial transactions" and
  importing GSAP extracts "Import GSAP extracts" (015 FR-028). Budgets, planned costs and line-item coding are
  planning data and follow "View plans".
- **FR-016** (BUS.ADM.020): Exporting All work MUST need "Export from All work".

**Administration of roles and access (Settings › Roles and access)**

- **FR-017** (BUS.ADM.020, BUS.ADM.010): **People** tab: every person (active first, then inactive) with job title,
  teams, roles and scopes; search by name or e-mail; filter by role including "No role". **Edit roles** opens a
  side panel: one checkbox per role and, for each ticked role, its scope (All markets, My teams, Chosen teams with
  a team picker), and a plain-language summary of what the person will be able to do (generated from Table A).
- **FR-018** (BUS.ADM.020, BUS.ADM.010): **Roles** tab: every role with description and number of people. A role's
  page shows its permissions grouped as in Table A; **Edit** changes name (unique, ≤ 60 characters),
  description (≤ 300), permissions (at least one) and default scope; **New role** starts empty or as a copy of a
  role; **Delete** is allowed only when nobody holds the role and sends it to the recycle bin with Undo. The
  Administrator role always has every permission and cannot be changed or deleted.
- **FR-019** (BUS.ADM.020, BUS.WFL.010): **Rules** tab: "Nobody approves an activity they own or submitted" (on by
  default) and "Role for new people" (a role, or none; default Viewer · My teams).
- **FR-020** (BUS.ADM.020): A change of roles, permissions or rules MUST apply to the person's next request,
  without signing in again; the web app MUST reload the person's access when the window regains focus and after
  every 403.
- **FR-021** (BUS.ADM.020): At least one active person MUST hold the Administrator role. Removing it from the last
  one, or deactivating the last one (003), MUST be refused with "<name> is the only administrator. Give someone
  else the Administrator role first."
- **FR-022** (BUS.ADM.020): Before a person's roles are saved, NorthStar MUST warn when (a) they are an approver
  in active approval workflows and will no longer hold "Approve or decline when named as an approver", with the
  number of workflows and of activities waiting for them; (b) the administrator removes their own Administrator
  role.
- **FR-023** (BUS.ADM.020): A person added in Settings › People MUST receive the Rules' "Role for new people" with
  scope My teams, in the same save (logged).
- **FR-024** (BUS.ADM.020): **My profile** MUST show a read-only **Your access** card: each role with its scope, a
  plain-language summary of what it allows, and the administrators to ask.

**Approvals (role-based, segregation of duties)**

- **FR-025** (BUS.WFL.010, BUS.ADM.020): Approve and Decline MUST be allowed only to (a) a named approver of the
  open round's current level who holds "Approve or decline when named as an approver" (with feature 017, also a
  current delegate or the backup approver of such a level, 017 FR-017 b–c, under the same permission), or (b) a
  person who holds "Approve or decline on behalf of the named approvers" for the plan (017's "administrator"),
  whose decision is recorded "on behalf of" the level's approvers. This replaces 008 FR-017 (D-020). Approve and
  Decline are offered only to people who may decide; someone who could decide but is blocked by FR-026 sees them
  disabled with the reason; others do not see them.
- **FR-026** (BUS.WFL.010): While "Nobody approves an activity they own or submitted" is on (default; 017 FR-018),
  nobody MAY approve or decline an activity they own or whose open round they submitted, in any capacity. When the
  owner or submitter is the only named approver of a level, Submit MUST show the advice of the Edge cases and
  still submit.
- **FR-027** (BUS.WFL.010, BUS.ADM.020): Submit and re-submit MUST need "Submit activities for approval and recall
  them" for the plan. Recall MUST be allowed to the activity's owner and to the person who submitted the open
  round, each holding that permission for the plan, and to holders of "Approve or decline on behalf of the named
  approvers" (017 FR-021). This replaces 008 FR-021's "anyone may recall".
- **FR-028** (BUS.WFL.010): A person MUST be added as an approver of a workflow level only if they hold "Approve or
  decline when named as an approver"; approvers who no longer hold it MUST be flagged on the workflow page and in
  the approval chain.

**Global and local budgets**

- **FR-029** (BUS.BUD.040): A **global budget** MUST have a name (1–120 characters, unique ignoring case), a year
  (current year −1 to +5, fixed once saved), an amount in USD (0 or more, two decimals), an owner (a person who
  holds "Create global budgets and allocate them to plans") and an optional description (≤ 2,000). It is created,
  changed and deleted with "Create global budgets and allocate them to plans" and seen with "See global budgets".
- **FR-030** (BUS.BUD.040): A global budget MUST be **allocated** to plans of its year that the person may manage
  (scope), one allocation per plan across all global budgets, with an amount of 0 or more in the plan's currency
  (USD unless the plan is planned in another currency, 016; D-303). Creating or changing an allocation MUST set
  the plan's budget to the allocated amount in the same save. The global budget's own figures are in USD.
- **FR-031** (BUS.BUD.040): The budget of an allocated plan MUST be read-only for everyone on the plan form, with
  "Allocated from '<global budget>' — ask <owner> to change it."; the server MUST refuse a different budget with
  409 `not_allowed` and the same sentence. Budgets of campaigns and initiatives stay editable as today (domain
  model §4.1).
- **FR-032** (BUS.BUD.040): The **global budget page** MUST show Global budget, Allocated, Unallocated (red
  "Over-allocated by X" when negative; never blocks, as D-015) and Planned cost of the allocated plans; one row
  per allocated plan with teams, owner, status, allocated, planned cost, left to plan and budget used; and
  "Plans of <year> without an allocation" with **Allocate**.
- **FR-033** (BUS.BUD.040): A global budget's rows MUST be limited to plans the viewer can see; allocations to
  other plans MUST be added up in one row "Other markets (n plans)". Local planners MUST see their plan's
  allocation on the plan page (budget card) and never other markets' allocations.
- **FR-034** (BUS.BUD.040): **Remove allocation** MUST leave the plan's budget at its last amount and make it
  editable again.
- **FR-035** (BUS.BUD.040): A global budget with allocations MUST NOT be deleted (409 `in_use`); one without goes
  to the recycle bin with Undo. Copying a plan MUST NOT copy its allocation. Deleting a plan MUST take its
  allocation to the recycle bin with it; restoring brings it back unless its global budget is gone.
- **FR-036** (BUS.BUD.040): **Global budgets** MUST be a sidebar entry (after Plans) for people with "See global
  budgets", listing the global budgets by year with their four totals and a **New global budget** button for
  people who may create them.

**Audit trail**

- **FR-037** (BUS.ADM.020, BUS.WFL.010, BUS.ADM.010): Every create, change (each field old → new, permissions as
  "added …" / "removed …"), delete and restore of a role; every change of a person's roles and scopes (on the
  person, "Roles: old → new"); every change of the access rules; every create, change, delete and restore of a
  global budget; and every allocation added, changed or removed (on the global budget and on the plan) MUST be
  written to the activity log with who and when, in the same save.
- **FR-038** (BUS.ADM.020): Settings › Roles and access MUST show **Recent access changes** (the latest 20 entries
  about roles, people's roles and access rules, newest first), each linking to the activity log. These entries,
  like every entry about Settings, people and teams, MUST be visible only to people with "See the whole activity
  log".
- **FR-039** (BUS.ADM.020): A refused request MUST NOT change data or write to the activity log; the server's
  technical log MUST record the operation, the person's ID and the code `forbidden`, without names or e-mails
  (constitution VI).

**Demo data and existing features**

- **FR-040** (BUS.ADM.020, BUS.BUD.040): The demo data MUST include the five roles of Table B, the role
  assignments of the scenarios above, the rules' defaults, and the global budget *FY2026 Global marketing
  budget* (2026, 2,500,000.00 USD, owner Carlos Mendes) allocated 1,200,000.00 USD to *FY2026 Lubricants Europe*
  and 800,000.00 USD to *FY2026 Mobility China*.
- **FR-041** (BUS.ADM.020): This feature replaces the deferred role rules of 002 FR-008, 003 FR-026–FR-027,
  004 FR-004 and FR-032, 005 FR-031, 006 and 007 (Assumptions), 008 FR-017 and FR-021, 009 (Assumptions) and
  011 (Assumptions). Each of those specs gains one sentence pointing here.

#### Table A — Permissions (fixed by NorthStar)

"Scoped" permissions apply to plans in the role's scope (FR-004); the others apply everywhere. The **refusal noun**
starts the refusal message ("<Noun> isn't part of your role …").

| Group | Key | Label (role page) | Scoped | Refusal noun |
|---|---|---|---|---|
| Plans and planning | `plans.view` | View plans and everything in them | Yes | Viewing this plan |
| | `plans.create` | Create and copy plans | Yes | Creating plans |
| | `plans.edit` | Edit plan details | Yes | Editing plans |
| | `plans.publish` | Publish and unpublish plans | Yes | Publishing plans |
| | `plans.delete` | Delete plans | Yes | Deleting plans |
| | `campaigns.edit` | Add, edit, move and copy campaigns and initiatives | Yes | Changing campaigns and initiatives |
| | `campaigns.delete` | Delete campaigns and initiatives | Yes | Deleting campaigns and initiatives |
| | `activities.edit` | Add, edit, move and copy activities and line items | Yes | Changing activities and line items |
| | `activities.delete` | Delete activities and line items | Yes | Deleting activities and line items |
| | `activities.submit` | Submit activities for approval and recall them | Yes | Submitting activities |
| | `data.export` | Export from All work | Yes | Exporting |
| Financial data | `finance.view` | See financial data | Yes | Seeing financial data |
| | `finance.link` | Link financial transactions to line items | Yes | Linking financial transactions |
| | `finance.import` | Import GSAP extracts | No | Importing GSAP extracts |
| Approvals | `approvals.decide` | Approve or decline when named as an approver | No | Approving activities |
| | `approvals.decideOnBehalf` | Approve or decline on behalf of the named approvers | Yes | Deciding on behalf of other approvers |
| Global budgets | `globalBudgets.view` | See global budgets and their allocations | Yes | Seeing global budgets |
| | `globalBudgets.manage` | Create global budgets and allocate them to plans | Yes | Managing global budgets |
| Administration | `settings.manage` | Change Settings lists (models, activity types, Shell attributes, finance budget items, field help) | No | Changing Settings |
| | `exchangeRates.manage` | Add currencies, change exchange rates and import rate files | No | Changing currencies and exchange rates |
| | `workflows.manage` | Create and change approval workflows | No | Changing approval workflows |
| | `people.manage` | Add and change people and teams | No | Changing people and teams |
| | `access.manage` | Give roles and change roles and access rules | No | Changing roles and access |
| | `activityLog.viewAll` | See the whole activity log, including Settings, people and access changes | No | Seeing the whole activity log |
| | `recycleBin.purge` | Delete permanently and empty the recycle bin | No | Deleting permanently |

#### Table B — Default roles (seeded; changeable except Administrator)

✓ = included. Default scope is what the People tab proposes when the role is ticked.

| Permission | Administrator | Global planner | Planner | Approver | Viewer |
|---|---|---|---|---|---|
| View plans | ✓ | ✓ | ✓ | ✓ | ✓ |
| Create and copy plans | ✓ | ✓ | ✓ | | |
| Edit plan details | ✓ | ✓ | ✓ | | |
| Publish and unpublish plans | ✓ | ✓ | ✓ | | |
| Delete plans | ✓ | ✓ | | | |
| Campaigns and initiatives: add, edit, move, copy · delete | ✓ · ✓ | ✓ · ✓ | ✓ · ✓ | | |
| Activities and line items: add, edit, move, copy · delete | ✓ · ✓ | ✓ · ✓ | ✓ · ✓ | | |
| Submit activities and recall them | ✓ | ✓ | ✓ | | |
| Export from All work | ✓ | ✓ | ✓ | ✓ | ✓ |
| See financial data | ✓ | ✓ | ✓ | ✓ | ✓ |
| Link financial transactions | ✓ | ✓ | ✓ | | |
| Import GSAP extracts | ✓ | | | | |
| Approve or decline when named | ✓ | ✓ | | ✓ | |
| Decide on behalf of named approvers | ✓ | | | | |
| See global budgets · manage them | ✓ · ✓ | ✓ · ✓ | | | |
| Every administration permission | ✓ | | | | |
| **Default scope** | All markets (fixed) | All markets | My teams | My teams | My teams |

### Key Entities *(include if feature involves data)*

- **Role**: name, description, permissions, default scope, built-in kind (one of the five, or custom).
- **Role assignment**: a person, a role and a scope (All markets, My teams, or a list of teams); one per person
  and role.
- **Access rules**: one record: "Nobody approves an activity they own or submitted" and "Role for new people".
- **Global budget**: name, year, amount (USD), owner, description; its allocations.
- **Budget allocation**: a global budget, a plan of the same year (at most one allocation per plan) and an amount;
  it sets the plan's budget.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100 % of the API's operations have a declared access rule (checked by an automated test that fails
  for any operation without one), and every operation that changes or reads planning data is shown by an
  automated test to refuse a Viewer and a person outside the plan's scope.
- **SC-002**: On the plan, campaign, initiative, activity, line-item, Settings and global-budget pages, a Viewer
  sees no enabled action that the API would refuse (automated browser check).
- **SC-003**: An administrator gives a person a role in at most 3 steps from Settings, in under 1 minute, and the
  person's next action follows it without signing in again.
- **SC-004**: For a local planner, every count and total on Plans, Home and All work equals the sum over the plans
  of their teams only (automated test on the seed).
- **SC-005**: A global planner sees allocated, planned and remaining amounts for every market of a year on one
  page, and an allocation change shows in the plan's budget on the next page load.
- **SC-006**: 100 % of access changes and allocation changes appear in the activity log with who, when and old →
  new.
- **SC-007**: Every new screen passes an automated WCAG 2.1 AA check and works with the keyboard alone.

## Assumptions

Defaults chosen on the product owner's instruction of 2026-10-20, all reversible by configuration or by a small
change; each is listed for Shell to confirm (Q-010, Q-011, Q-006, Q-014 of the gap analysis).

- **Roles**: five roles (Table B) instead of Hive9's twenty (functional analysis §F4); Shell can add roles itself.
- **Market = team**: access scope is by team. Scoping by Business unit, Cluster or Country/OU values is not offered;
  it can be added as a fourth scope type later.
- **Visibility**: a plan is visible only inside the roles' scope (answers Q-006: "yes, hide plans from people
  outside the plan's teams"), plus owners and named approvers (Edge cases).
- **Refusals**: 403 for a record outside the person's access (with a message that names nothing of the record);
  404 only for records that do not exist.
- **Financial data**: every default role may see it; the permission exists so Shell can restrict it.
- **Global budgets**: a global planner shares a global budget top-down; the allocation becomes the plan budget and
  is read-only for local owners (Q-011's first option). "Request and approve" allocations, budgets per business
  unit or category are not part of this feature. With plan currencies (016), an allocation is entered in the
  plan's currency and the global budget's figures convert it to USD on read (consolidation D-303).
- **Segregation of duties**: owner and submitter, on by default, as 017 FR-018. 014 keeps it as a switch in the
  Rules tab so Shell can relax it without code; 017 follows the switch (consolidation D-302).
- **Approval delegation, auto-approval, escalation and role approvers** are specified by feature 017 (approval
  rules); it reuses FR-025–FR-027 and resolves role approvers through 014's role holders for the plan's scope.
  017's example role "Global budget owner" is 014's **Global planner**.
- **Financial transactions** are feature 015; it uses "See financial data", "Link financial transactions" and
  "Import GSAP extracts" (FR-015). Field help and rate files (feature 016) are covered by "Change Settings lists"
  and "Add currencies, change exchange rates and import rate files"; 016's pages that add up several plans follow
  FR-007.
- **Demo sign-in stays** (D-004's sign-in part); roles are assigned to demo people. With Microsoft Entra ID later,
  the person's identity comes from single sign-on and role assignments may come from Entra app roles or groups;
  the permission model does not change.
- **Refused attempts** are not shown to administrators in this phase (technical log only).
- Out of scope: field-level permissions (other than the allocated plan budget and financial data), time-limited
  roles, per-record sharing, approval of role changes.
