# Data Model: Access Control and Global Budgets (014)

Conventions are feature 001's (record.Meta with version and created/updated by and at, readable IDs from
`ids.Generator`, append-only activity log, recycle bin through `history.MoveToBin`) and the build guide's
(`docs/sdd/plan-module-build.md` §7). Stored records are relational: ids and foreign keys, no nested ownership.
Money is `money.Amount` (exact decimal) in USD (D-010).

## 1. Stored tables

### 1.1 `roles` — Role (`RL-1001`, …)

| Field | Type | Rules |
|---|---|---|
| `id` | string | `RL-` + number, never reused |
| `name` | string | Required, 1–60 characters after trimming, unique among roles ignoring letter case and surrounding spaces. Message: "A role called '<name>' already exists." |
| `description` | string | 0–300 characters |
| `builtIn` | string | `administrator`, `globalPlanner`, `planner`, `approver`, `viewer` for the seeded roles; `""` for roles made in Settings. Never changes. Lets the seed and tests find a role whatever its name |
| `permissions` | []string | Keys of Table A (spec), at least one ("Choose at least one permission."), no duplicates, stored in Table A order. Unknown keys are refused ("'<key>' is not a permission.") |
| `defaultScope` | string | `all` or `myTeams`: what the People tab proposes when the role is ticked |
| `record.Meta` | | version, createdAt/By, updatedAt/By |

- `builtIn = administrator`: `permissions` is always every key of Table A and `defaultScope` is `all`. Name,
  description and permissions cannot change (`Lock` reasons in the plan); it cannot be deleted. When NorthStar
  adds a permission later, the administrator role has it without a data change (`authz` treats this role as
  "everything").
- Delete: only when no `role_assignments` row references it (409 `in_use`), and never the Administrator role.
  Deleted roles go to the recycle bin (history kind `role`, top level, no children). Restore follows 001's name
  rule ("<name> (restored)" when the name is taken).

### 1.2 `role_assignments` — RoleAssignment (`RA-1001`, …)

| Field | Type | Rules |
|---|---|---|
| `id` | string | `RA-` + number |
| `personId` | string | An existing person (`U-…`); active or not |
| `roleId` | string | An existing role |
| `scopeType` | string | `all`, `myTeams` or `teams`. Administrator role: always `all` ("The Administrator role always applies to all markets.") |
| `teamIds` | []string | Only with `teams`: 1 or more existing teams, no duplicates, sorted by team name; `[]` otherwise. Message: "Choose at least one team." |
| `record.Meta` | | |

- Unique `(personId, roleId)`.
- The set of a person's assignments is changed **as a whole** through `PUT /api/people/{personId}/roles`, with the
  **person's** version (as memberships, D-057): the person's version goes up by one and one `changed` entry
  with field `roles` is logged on the person (§3). Assignment rows are created, updated or removed to match;
  removed rows are deleted outright (not to the bin: they are links, and the log keeps the old value).
- When a team is deleted (003), its id is removed from every `teamIds` in the same save; an assignment left with
  `teams` and no team stays and grants nothing (People tab flag "No teams left in this scope"). Restoring the team
  does not put it back.
- When a role is restored from the bin, its assignments are not restored (there were none: FR-018).
- Last administrator (FR-021): after any change of assignments or any deactivation of a person (003), at least one
  **active** person must hold an assignment of the `administrator` role, else 409 `not_allowed`.

### 1.3 `access_rules` — AccessRules (one record, id `access-rules`)

| Field | Type | Default | Rules |
|---|---|---|---|
| `id` | string | `access-rules` | Fixed |
| `noSelfApproval` | bool | `true` | FR-026 |
| `newPersonRoleId` | string | the Viewer role's id | `""` = no role; otherwise an existing role that is not Administrator ("New people can't be made administrators automatically.") |
| `record.Meta` | | | Created by the seed; when missing (empty store) the defaults above apply and the first PATCH creates it |

- Never deleted. Changes are logged on history kind `accessRules` (top level, label "access rules", name
  "Access rules").
- If the role named by `newPersonRoleId` is deleted, the value becomes `""` in the same save (logged on
  `accessRules`).

### 1.4 `global_budgets` — GlobalBudget (`GB-1001`, …)

| Field | Type | Rules |
|---|---|---|
| `id` | string | `GB-` + number |
| `name` | string | Required, 1–120 characters, unique among global budgets ignoring case: "A global budget called '<name>' already exists." |
| `year` | int | Required, current year −1 to current year +5 (same rule as plans, `plans/validate.go`); cannot change after the first save ("The year can't change once the global budget is saved.") |
| `amount` | money.Amount (USD) | Required, 0 or more, at most two decimals: "Enter an amount of 0 or more." |
| `ownerId` | string | Required: an active person who holds `globalBudgets.manage` for All markets: "Choose a person who can manage global budgets for all markets." |
| `description` | string | 0–2,000 characters |
| `record.Meta` | | |

- Delete: 409 `in_use` while any `budget_allocations` row references it; otherwise to the recycle bin (history
  kind `globalBudget`, top level).

### 1.5 `budget_allocations` — BudgetAllocation (`BA-1001`, …)

| Field | Type | Rules |
|---|---|---|
| `id` | string | `BA-` + number |
| `globalBudgetId` | string | An existing global budget |
| `planId` | string | An existing plan of the same year ("Choose a plan of <year>."); unique across all allocations ("'<plan>' already has an allocation from '<global budget>'.") |
| `amount` | money.Money in the plan's currency (D-303; USD until 016 is built) | Required, 0 or more, two decimals; another currency: "Enter the amount in <code>, the plan's currency." |
| `record.Meta` | | |

- History kind `budgetAllocation`, **parent kind `plan`** (so a plan's delete, Undo and restore carry it):
  - its `PrepareRestore` drops the allocation, with the note "The allocation was not restored because its global
    budget was deleted.", when the global budget no longer exists or the plan meanwhile got another allocation;
  - **Remove allocation** (`DELETE /api/budget-allocations/{id}`) deletes the row outright, not to the bin: the
    plan keeps its budget and the log keeps the value (FR-034).
- Writing an allocation (create or amount change) sets `plans.budget` of its plan to the allocation's amount in the
  same `Write`, calls `Touch` on the plan and logs the plan change (§3).

### 1.6 Readable-ID prefixes added to `platform/ids`

`PrefixRole = "RL"`, `PrefixRoleAssignment = "RA"`, `PrefixGlobalBudget = "GB"`, `PrefixBudgetAllocation = "BA"`.

## 2. Derived, never stored

### 2.1 Grants (one per request, `platform/authz.Grants`)

Built by the access middleware after sign-in, from the person's assignments, their roles, their team memberships
(all teams they belong to, active or not) and the access rules:

```
Grants {
  PersonID       string
  Roles          []GrantedRole        // name, builtIn, scope as shown ("All markets" | "My teams" | team names)
  Administrators []actor.Actor        // active holders of the Administrator role, by name: "ask …"
  Rules          { NoSelfApproval bool }
  perms          map[Permission]Scope // the union of every role's permissions
}
Scope { All bool; TeamIDs set }       // merged: All if any granting role is All markets; else the union of teams
```

- A role with scope `myTeams` contributes the person's team ids; `teams` contributes its `teamIds`; `all` sets All.
- The Administrator role contributes every permission with All.
- A permission held by no role is absent (`Has` = false).

### 2.2 PlanFacts (what a decision needs to know about a plan)

`authz.PlanFacts { ID, TeamIDs []string, OwnerID, CreatedByID string }`, built from a stored plan (or from a plan
snapshot in the recycle bin) by `plans.AccessFacts(Plan)`.

**In scope** (FR-005): `scope.All` or `TeamIDs ∩ scope.TeamIDs ≠ ∅` or (`len(TeamIDs) == 0` and `CreatedByID ==
PersonID` and the person holds the permission in some scope).

**Can view a plan** (FR-006): in scope for `plans.view`, or `OwnerID == PersonID`.

**Can view an activity**: can view its plan, or the activity's owner is the person, or the person is a named
approver of any level of any of its approval rounds (008's round snapshot of levels).

### 2.3 Availability (answers' `access` objects)

`{available bool, reason string}` — `reason` is "" when available, else the exact refusal message of the plan's
*User-visible messages*. Same shape as 008's `ApprovalActionAvailability`.

## 3. Activity-log entries written by this feature

| Change | Kind · record | Action / verb | Changes (field · label · text) |
|---|---|---|---|
| Role created | `role` · role | created | — (details: "with 9 permissions") |
| Role changed | `role` · role | changed | `name` · Name; `description` · Description; `defaultScope` · Default scope (All markets / My teams); `permissions` · Permissions — OldText/NewText list labels; the entry's `details` reads "Added: …; removed: …" |
| Role deleted / restored | `role` | deleted / restored | 001's rules |
| Person's roles changed | `person` (003's kind) · person | changed | `roles` · Roles — texts like "Planner · My teams, Approver · Mobility China" (roles sorted by name; "No role" when empty) |
| New person given the default role | `person` | changed | same field, logged right after 003's `created` entry |
| Access rules changed | `accessRules` · "Access rules" | changed | `noSelfApproval` · Nobody approves an activity they own or submitted (On/Off); `newPersonRoleId` · Role for new people (role name / "No role") |
| Global budget created / changed / deleted / restored | `globalBudget` | created / changed / deleted / restored | `name`, `year`, `amount` (money text "2,500,000.00 USD"), `ownerId` (person name), `description` |
| Allocation added / changed / removed | `globalBudget` · global budget | changed | `allocations` · Allocation to '<plan name>' — "Not allocated" → "1,200,000.00 USD" |
| Plan budget set by an allocation | `plan` (004's kind) · plan | changed | `budget` · Plan budget; `comment` "Allocated from '<global budget>'" (or "Allocation from '<global budget>' removed" on removal, with no field change) |

`people/kinds.go` `Describe` gains the label "Roles" for field `roles`; `plans` needs no change (its `budget`
label exists).

## 4. Changes to answers of existing features (contract fragment, "Changes to other features")

| Schema (owner) | Added | Filled by |
|---|---|---|
| `CurrentUser` (001/003) | `access: MyAccess` | access service in `people.CurrentUser` through a hook (`people.UseAccessSummary`) |
| `Plan`, `PlanSummary` (004) | `access: PlanAccess`; `Plan.allocation: PlanAllocation \| null` | plans service (allocation through hook `plans.UseAllocations`, implemented over `budgets`) |
| `PlanCampaigns` (005) | `access: PlanContentAccess` | campaigns service |
| `Campaign`, `Initiative` (005) | `access: ContainerAccess` | campaigns service |
| `ActivitySummaryPage` (006) | `access: PlanContentAccess` | activities service |
| `Activity` (006) | `access: ActivityAccess` | activities service |
| `LineItemSection`, `LineItemDetails` (007) | `access: LineItemAccess` | line items service |
| `ApprovalPerson` (008) | `canApprove: boolean` | approvals service (hook `approvals.UseApprovers`) |
| `ActivityApproval.actions` (008) | — (existing `available`/`reason` now also reflect FR-025–FR-027) | approvals service |
| `RecycleBinListItem` (011) | `access: BinEntryAccess` | history service (hook `history.UseAccess`) |
| `ImpactWarning.kind` (002) | `approverCannotApprove`, `removesOwnAdministration` | access service |
| `Error.code` (001) | `forbidden` | `errs.Forbidden` |

All added properties are optional in the schema and always sent by this feature (D-191).
