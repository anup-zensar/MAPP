# Data Model: People and Teams

**Feature**: `003-people-and-teams` · **Plan**: [plan.md](plan.md) · **Contract**: [contracts/openapi.yaml](contracts/openapi.yaml)

The records this feature stores or extends, their fields, rules and life cycles. Conventions (timestamps,
text lengths, `ActorRef`, `RecordRef`, bookkeeping `record.Meta`, relational shape, one JSON table per
record type) are those of [feature 001's data model](../001-app-shell-and-sign-in/data-model.md); they are
not repeated here. Field labels are the on-screen labels; JSON names are used in files and in the API.

Marking legend (domain model §3): **R** Required — blocks saving · — optional · calc calculated, read-only.
People and teams have no "needed to publish/submit" fields: they are Settings records (domain model §6).

---

## 1. Person — table `people` (extends feature 001 §1)

The stored record is **unchanged** (`people.Person`); 001's seed files stay valid. Feature 003 adds the
screens that create and change people and enforces the rules 001 wrote on the type.

| # | Label | JSON | Type | Mark | Validation (field error message) |
|---|---|---|---|---|---|
| 1 | — | `id` | string | calc | `U-1001`, … from the readable-ID counter `U`; never changes, never reused |
| 2 | Name | `name` | text | R | Trimmed. Empty → "Enter a name." · more than 120 characters → "Use 120 characters or fewer." Not unique (two people may share a name) |
| 3 | E-mail | `email` | text | R | Trimmed and stored in lower case. Empty → "Enter an e-mail address." · not `local@domain.tld` (exactly one `@`, no spaces, a `.` inside the domain, not at its ends) → "Enter a valid e-mail address." · more than 254 characters → "Use 254 characters or fewer." · used by another person, active or not, ignoring letter case and surrounding spaces → "<e-mail> already belongs to <name>." or, when that person is inactive, "<e-mail> already belongs to <name> (inactive) — reactivate them instead." |
| 4 | Job title | `jobTitle` | text | — | Trimmed. More than 120 characters → "Use 120 characters or fewer." Empty is allowed |
| 5 | Status | `active` | yes/no | calc | `true` on create; changed only by **Deactivate** / **Reactivate** |
| 6 | Teams | (table `team_memberships`) | reference (Teams, many) | — | See §3. A team the person is not yet in must be active ("<team> is inactive, so people can't be added to it. Reactivate the team first."); an id that is not a team → "Choose teams from the list." |
| — | *bookkeeping* | `version`, `createdAt`, `createdBy`, `updatedAt`, `updatedBy` | `record.Meta` | calc | Every update and status action sends `version` (missing or < 1 → field `version`: "Reload the page and try again."; out of date → `409 version_conflict`) |

Derived (API only, never stored): `firstName` (first word of the name), `initials` (FR-009: first letters
of the first two words; a one-word name gives its first two letters; upper case — "Cher" → "CH").

**Status and transitions** (FR-005 – FR-008):

```
            create                 Deactivate (not yourself)
  (none) ───────────► Active ─────────────────────────────► Inactive
                         ▲                                     │
                         └────────────── Reactivate ───────────┘
  There is no delete (FR-008, D-028).
```

| Status | Chip | Sign-in | Offered in pickers | Records that name them |
|---|---|---|---|---|
| Active | **Active** (positive, green) | yes (demo password) | yes | name shown as is |
| Inactive | **Inactive** (neutral, grey) | refused; an open session ends at its next request (001 `auth.Authenticate`) | no | kept, shown as "<name> (inactive)" |

- Both actions are `POST /api/people/{personId}/status` with `{active, version}` (feature 002's
  `StatusUpdate`, the status action of every Settings record).
- Deactivate refuses the signed-in person themselves (`409 not_allowed`, FR-006).
- Deactivate and Reactivate keep every team membership (FR-005, FR-007).
- Deactivating an inactive person, or reactivating an active one, changes nothing (no version, no log).
- The e-mail is the sign-in name: after a change the next sign-in uses the new e-mail; open sessions go on.

**Usage** (FR-004, API only): what the person owns or approves, per record type, counted from records not in
the recycle bin, provided by later features through hooks (plan *Constructor, hooks and wiring*):

| Role | Record type | Source | Example label |
|---|---|---|---|
| `owner` | `plan` | feature 004 `PlanSource` (plans whose owner is the person) | "2 plans" |
| `owner` | `campaign`, `initiative`, `activity` | features 005, 006 `PersonUsage` | "5 activities" |
| `approver` | `approvalWorkflow` | feature 008 `PersonUsage` | "1 approval workflow" |

Only counts of at least 1 are returned; `label` is singular when the count is 1.

## 2. Team — table `teams` (new)

A group of people that plans are assigned to (R1: a plan's owner must belong to one of the plan's teams).

| # | Label | JSON | Type | Mark | Validation (field error message) |
|---|---|---|---|---|---|
| 1 | — | `id` | string | calc | `T-1001`, … from the readable-ID counter `T` (proposed prefix); never changes, never reused |
| 2 | Name | `name` | text | R | Trimmed. Empty → "Enter a name." · more than 80 characters → "Use 80 characters or fewer." · another team (active, inactive — not in the recycle bin) has it, ignoring letter case and surrounding spaces → "A team called '<other team's stored name>' already exists." |
| 3 | Description | `description` | long text | — | Trimmed. More than 300 characters → "Use 300 characters or fewer." |
| 4 | Colour | `colour` | single choice | — | One of `blue`, `violet`, `teal`, `magenta`, `orange`, `green`, `slate`, `gold` (labels Blue … Gold), otherwise "Choose one of the colours." On create, when not sent: the palette colour used by the fewest teams (not in the recycle bin), ties in palette order (decision D-025) |
| 5 | Status | `active` | yes/no | calc | `true` on create; changed only by **Set team inactive** / **Reactivate** |
| 6 | Members | (table `team_memberships`) | people | — | A person who is not yet a member must be active ("<name> is inactive and can't be added. Reactivate them first."); an id that is not a person → "Choose people from the list." Existing inactive members may stay |
| — | *bookkeeping* | `record.Meta` | | calc | as for people |

**Status and transitions** (FR-014, FR-015):

```
            create             Set team inactive              Delete (no plan uses it, incl. the bin)
  (none) ──────────► Active ─────────────────────► Inactive ──────────────────────────────┐
                       ▲  │                           │                                    ▼
                       │  └──────────── Delete (unused) ─────────────────────────► in the recycle bin
                       └──────────── Reactivate ──────┘                          (with its memberships)
                                                                  Undo (≤ 2 min, exact) / Restore ──► back
```

| Status | Chip | Offered for new plan teams (004) | Plans that use it | Members count for R1 |
|---|---|---|---|---|
| Active | **Active** (positive) | yes | name shown as is | yes |
| Inactive | **Inactive** (neutral) | no | kept, shown as "<name> (inactive)" | yes |

- Set team inactive and Reactivate are `POST /api/teams/{teamId}/status` with `{active, version}`.
- **Delete** is refused with `409 in_use` while any plan uses the team, including plans in the recycle bin
  (FR-014); the web app offers **Set team inactive** instead.
- **Delete** of an unused team moves the team and all its memberships to one recycle-bin entry
  (`history.MoveToBin`; details "with 2 team memberships"); **Undo** puts both back exactly (FR-015).
- **Undo** is refused with `409 name_taken` when another team has meanwhile taken the name; **Restore**
  then renames it to "<name> (restored)", "<name> (restored 2)", … (001 `PrepareRestore` pattern, 011 FR-022).
- Set team inactive on an inactive team, or Reactivate on an active one, changes nothing.

**Usage** (FR-010, FR-014, API only): `plans` — every plan that uses the team, from feature 004's
`PlanSource`, including plans in the recycle bin (`inRecycleBin: true`), sorted by name. Empty until 004.

## 3. Team membership — table `team_memberships` (new)

One person in one team. A person may be in many teams; a team may have many members (edge cases: 250
members, 85 teams).

| Field | JSON | Type | Rules |
|---|---|---|---|
| ID | `id` | string | `TM-1001`, … from the counter `TM` (proposed prefix); internal, never shown |
| Team | `teamId` | string | An existing team; never changes |
| Person | `personId` | string | An existing person; never changes |
| — | `record.Meta` | | Version is always 1: a membership is created and removed, never changed |

- **Unique** pair (`teamId`, `personId`): the service never stores a second membership for the same pair
  (duplicate ids in a request are ignored).
- **Edited from both sides** (FR-012): the team's **Members** and the person's **Teams** are the same rows,
  so both show a change at once.
- **A membership change is a change of both records** (proposed decision): every person and every team
  whose membership set changes gets `Touch` (new version) and one `changed` log entry (§6). An edit form
  opened before the change is then refused with `version_conflict` instead of silently undoing it.
- Deactivating a person or a team keeps its memberships (FR-005, FR-014).
- Deleting a team removes its memberships with it (children in the recycle-bin entry); people are never
  deleted, so a membership never outlives its person.
- **Owner impact** (FR-013, R1): a plan **needs attention** when its owner loses a membership in one of the
  plan's teams and, after the change, belongs to none of the plan's teams (inactive teams count). Only
  stored plans of 004's `PlanSource` are considered (not those in the recycle bin). `POST
  /api/teams/{teamId}/impact` and `POST /api/people/{personId}/impact` take the same body as the update,
  save nothing, and answer feature 002's `ChangeImpact`: one warning of kind `ownerNeedsAttention` per
  affected owner, with the sentence of plan *User-visible messages* and `count` = their plans. The update
  itself is always allowed (FR-013 "the change is allowed").

## 4. Preferences — table `preferences` (extends feature 001 §2)

Still one record per person who changed a preference, written only when a value changes; not a business
record: no version, not logged (decision D-038, extended to the two new preferences).

| Field | JSON | Type | Default | Validation |
|---|---|---|---|---|
| Person | `personId` | string | — | key |
| Theme | `theme` | enum | `system` | 001 |
| Sidebar collapsed | `sidebarCollapsed` | yes/no | `false` | 001 |
| Display currency | `displayCurrency` | ISO code | `USD` | Must be a currency of feature 002 that is **active**, or the person's current value (a display currency set inactive later is kept — spec edge case); otherwise "Choose one of the active currencies." |
| Date format | `dateFormat` | enum | `DD MMM YYYY` | `DD MMM YYYY` · `DD/MM/YYYY` · `MM/DD/YYYY` · `YYYY-MM-DD`; otherwise "Choose one of the date formats." |

- Records saved by feature 001 have no `displayCurrency` / `dateFormat`: an empty stored value means the
  default, so no migration is needed.
- Until feature 002 is wired, the only currency is USD (`people.OnlyUSD`, spec Assumptions).

| Date format | Example (12 March 2026) | Shown in choices as |
|---|---|---|
| `DD MMM YYYY` | 12 Mar 2026 | 12 Mar 2026 (DD MMM YYYY) |
| `DD/MM/YYYY` | 12/03/2026 | 12/03/2026 (DD/MM/YYYY) |
| `MM/DD/YYYY` | 03/12/2026 | 03/12/2026 (MM/DD/YYYY) |
| `YYYY-MM-DD` | 2026-03-12 | 2026-03-12 (YYYY-MM-DD) |

Typed dates are read in the chosen format (1- or 2-digit day and month, month names in any letter case for
`DD MMM YYYY`); an impossible date (31/02/2026) is not a date.

**Display rate** (Go only, `people.DisplayRateOf(ctx, personID)`): `{Code, Rate}` — the person's display
currency and its current rate in units per 1 USD as decimal text, from feature 002 (`"1"` for USD). The APIs
of 004–012 convert their USD amounts with it (feature 002's `money.FromUSD`) and label them; the web app
never converts. When the stored code is no longer known to the currency source, it falls back to
`{"USD", "1"}`.

## 5. API views (not stored)

| View | Fields | Notes |
|---|---|---|
| `PersonRef` | `id`, `name`, `initials`, `jobTitle`, `active` | **The person reference every feature returns** for owners, approvers and members (004–008); names are read at answer time, so a rename shows everywhere |
| `TeamRef` | `id`, `name`, `colour`, `active` | The team reference every feature returns (plan teams, My teams) |
| `Person` | `id`, `name`, `email`, `jobTitle`, `initials`, `active`, `teams` (TeamRef[], by name), bookkeeping | List item and picker data |
| `PersonDetails` | `Person` + `usage` (UsageCount[]) | Details, create and update answers |
| `UsageCount` | `role` (`owner` · `approver`), `recordType`, `label`, `count` ≥ 1 | Like 001's `ContentCount` |
| `Team` | `id`, `name`, `description`, `colour`, `active`, `members` (PersonRef[], by name), `plans` (TeamPlanUse[], by name), bookkeeping | List item, details, create and update answers |
| `TeamPlanUse` | `id`, `name`, `inRecycleBin` | From 004's `PlanSource` |
| `CurrentUser` | 001 fields + `version`, `teams` (TeamRef[], by name — FR-026 "My teams") | `GET /api/me`, `PATCH /api/me`, sign-in |
| `Preferences` | `theme`, `sidebarCollapsed`, `displayCurrency`, `dateFormat` | |
| `DisplayCurrencyOption` | `code`, `name`, `active` | Active currencies, plus the person's current one when inactive, sorted by code |
| `ChangeImpact` (feature 002's) | `warnings: [{kind: "ownerNeedsAttention", message, count}]` | Answer of the two `…/impact` checks |

Sorting: people by name ignoring letter case, then e-mail; teams by name ignoring letter case; members and
teams inside a view the same way. Every list in a view is `[]`, never `null`.

## 6. History kinds and activity-log entries (FR-028)

Kinds registered with `history` (in `internal/app`, before the samples' kinds):

| Kind `Name` | Singular / Plural | Parent | Snapshot name | `Describe` | `PrepareRestore` |
|---|---|---|---|---|---|
| `person` | person / people | — | name | `email` → "E-mail"; `jobTitle` → "Job title"; `teamIds` → "Teams", team names by name joined ", " ("No team" when empty); `active` → "Status", "Active" / "Inactive" | — (never deleted; its `Remove` returns an error) |
| `team` | team / teams | — | name | `memberIds` → "Members", people's names by name joined ", " ("No members" when empty); `colour` → "Colour", palette label; `active` → "Status" | Undo: refuse when the name is taken (`name_taken`); Restore: " (restored)" suffix + note |
| `teamMembership` | team membership / team memberships | `team` | the person's name | — | — |

The log compares **log views** with `history.Diff` (bookkeeping is never part of them):
`person` → `{name, email, jobTitle, active, teamIds}`; `team` → `{name, description, colour, active,
memberIds}` (id lists sorted). Names in texts are those at the time of the change.

| Person's action | Entries written (one `Write` with the change) |
|---|---|
| Create a person with teams A, B | 1 × person `created`; 1 × team `changed` for A and for B (Members old → new) |
| Edit a person's name / e-mail / job title | 1 × person `changed` with each changed field |
| Edit a person's teams (+A, −B) | 1 × person `changed` (Teams); 1 × team `changed` for A and for B (Members) |
| Deactivate / Reactivate a person | 1 × person `statusChanged`, verb `deactivated` / `reactivated`, change Status Active → Inactive |
| Edit My profile (name, job title) | 1 × person `changed` |
| Change display currency or date format | none (D-038) |
| Create a team with members P, Q | 1 × team `created`; 1 × person `changed` for P and for Q (Teams) |
| Edit a team (name, members +P −Q) | 1 × team `changed` (Name, Members); 1 × person `changed` for P and for Q |
| Set team inactive / Reactivate | 1 × team `statusChanged`, verb `deactivated` / `reactivated` |
| Delete an unused team with 2 members | 1 × team `deleted`, details "with 2 team memberships" (history) |
| Undo / Restore it | 1 × team `restored` (history; a rename on Restore is in the details) |
| Save without a change; a refused save | none |

## 7. Seed (FR-029, docs/product/demo-users-and-data.md)

Created by `actor.System` at the seed time, after the six people (U-1001 … U-1006), through the counters:

| Team | ID | Colour | Description | Members (membership IDs) |
|---|---|---|---|---|
| Lubricants Europe | T-1001 | blue | Marketing for lubricants across Europe. | Priya Raman (TM-1001), Daniel Okoro (TM-1002), Sara Novak (TM-1003) |
| Mobility China | T-1002 | violet | Mobility and retail marketing in China. | Mei Lin (TM-1004), Carlos Mendes (TM-1005), Sara Novak (TM-1006) |
| Global Marketing Ops | T-1003 | teal | Marketing operations and Settings for every business. | Alex Morgan (TM-1007), Carlos Mendes (TM-1008) |

No preferences records are seeded (everyone starts with USD and DD MMM YYYY). The seed writes no log
entries for these records (001 seed convention). Next IDs after the seed: `U-1007`, `T-1004`, `TM-1009`.
