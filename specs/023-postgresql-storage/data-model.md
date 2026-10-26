# Data Model: PostgreSQL Storage (023)

The relational schema for every record type the JSON store holds today: its 39 tables (named in
`internal/*/store_json*.go`) become 50 tables in 16 migrations, because the lists a record owns become
child tables. The migrations in `backend/internal/platform/database/migrations` are the source of truth;
this page is their map. [Research](research.md) R2–R4, R9 and R10 explain the conventions.

## 1. Conventions

| Go value | Column(s) | Constraint |
|---|---|---|
| Readable id `P-1001` | `id text PRIMARY KEY` | `CHECK (id ~ '^P-[0-9]+$')` with the area's prefix |
| `record.Meta` (marked **M**) | `version integer`, `created_at timestamptz`, `created_by_id text`, `created_by_name text`, `updated_at timestamptz`, `updated_by_id text`, `updated_by_name text` | all `NOT NULL`; `CHECK (version >= 1)`; actor ids are not foreign keys (`system` is no person) |
| `actor.Actor` field `x` | `x_id text`, `x_name text` | both `NULL` together when the Go field is a pointer |
| `money.Amount` / `money.Money` | `x_minor bigint` / plus `x_currency text` | `>= 0` where the rule says so; the pair `NULL` together when optional |
| `money.Rate` / `money.Percent` | `x_micros bigint` / `x_hundredths bigint` | ranges as in the rules (R3) |
| `date.Date` / `*date.Date` | `date NOT NULL` / `date` | |
| month `"2026-03"` | `text` | `CHECK (month ~ '^[0-9]{4}-(0[1-9]|1[0-2])$')` |
| `time.Time` / `*time.Time` | `timestamptz NOT NULL` / `timestamptz` | returned in UTC (R3) |
| enumeration | `text` | `CHECK (x IN (…))`, `''` included where Go allows it |
| optional reference `""` / `nil` | `text` (`NULL` for none) | foreign key, unless a soft reference (§4) |
| list of references a record owns | child table (`position` when ordered) | `ON DELETE CASCADE`; the parent's `Put` replaces the rows |
| list of plain values | `text[]` | `NULL` for a `nil` list, `'{}'` for an empty one |
| document (R10) | `json` | `CHECK (json_typeof(x) IN (…))` |

Every foreign key is `DEFERRABLE INITIALLY DEFERRED`, checked at commit (R4). "Unique ignoring case" is a
unique index on `lower(col COLLATE "C")`, never stricter than the service rule (R4); uniqueness that a
service may break for a moment inside one Write is a deferrable constraint. Lists are read
`ORDER BY length(id), id COLLATE "C"`, the JSON store's order.

## 2. Tables by migration

| Migration | Tables (key constraints) |
|---|---|
| 00001 samples | `sample_folders` (name unique ignoring case), `sample_notes` (→ folder) |
| 00002 ids counters | `counters` (prefix, last ≥ 1001; raised by one `INSERT … ON CONFLICT … RETURNING`) |
| 00003 history | `activity_log` (seq unique, read in seq order; `path`, `changes`, `items` json), `recycle_bin` (`path`, `parent`, `contents`, `snapshots` json; the resolution's four columns all set or all empty) |
| 00004 people | `people` (e-mail unique ignoring case; manager → people, not themselves), `preferences` (enumerations; `pinned_plan_ids text[]` in pinned order), `teams` (name unique; palette colour), `team_memberships` (one per team and person) |
| 00005 sessions | `sessions` (id = 64 hex digits of the token's SHA-256; → people) |
| 00006 settings | `currencies` (ISO code key; name unique; USD never inactive; legacy rate columns), `rate_imports`, `exchange_rates` (→ currency, never USD; one per currency and day; → import), `field_help`, `activity_types` (name unique; default cost in USD), `models` (name unique; percentages and deal size ranges), `model_activity_types` (ordered), `shell_attributes` (label unique per level; one Business unit; % split only at activity and line-item level; → parent attribute), `shell_attribute_options` (label unique per attribute), `shell_attribute_conditions` and `shell_option_parents` (ordered option references), `finance_budgets` (name unique per year), `finance_budget_items` (name unique per budget) |
| 00007 access | `roles` (`permissions text[]` limited to Table A, at least one; one role per built-in key; name unique), `role_assignments` (one per person and role), `role_assignment_teams` (ordered), `access_rules` (the single `access-rules` row) |
| 00008 approval workflows | `approval_workflows` (name unique; escalation after the reminder), `approval_workflow_levels` (ordered; conditions json; backup → people), `approval_workflow_level_approvers` (ordered, → people) |
| 00009 All work | `all_work_preferences` (`columns text[]`; sort as two columns set together), `saved_views` (state json; name unique per person; one default per person) |
| 00010 plans | `plans` (name unique; → model, owner, default finance budget, currency; attributes json), `plan_teams` (ordered; the scope), `plan_phasing_months`; function `plan_in_scope()` |
| 00011 campaigns | `campaigns` (→ plan; name unique per plan; dates both or neither and in order; budget with currency), `campaign_phasing_months`, `initiatives` (→ campaign; same rules), `initiative_phasing_months` |
| 00012 activities | `activities` (→ initiative, activity type, owner; name unique per initiative; MQL or revenue only with Set later stages myself), `line_items` (→ activity, currency, finance budget item; name unique per activity), `line_item_spread_months` (amount ≥ 0); function `activity_in_scope()` |
| 00013 approvals | `approval_rounds` (levels and facts json; one number per activity and at most one open round, both at commit), `approval_actions` (→ round), `notifications` (→ recipient; read exactly when `read_at` is set), `approval_delegations` (≤ 90 days; no overlap with another active delegation of the person: a `btree_gist` exclusion at commit); `activity_in_scope()` gains the named approvers |
| 00014 global budgets | `global_budgets` (name unique; owner → people), `budget_allocations` (one per plan at commit; → plan, currency) |
| 00015 financials | `financial_imports`, `financial_transactions` (one per GSAP identity at commit; an accrual has its month; a manual match has who and when; → import); function `plan_in_permission_scope()` |
| 00016 write-once | triggers that refuse `UPDATE` and `DELETE` on `activity_log`, `approval_actions` and `rate_imports`, the audit trails written once (011 FR-023, 016 data-model §2.3); their stores only insert. A reset of the demo data uses `TRUNCATE`, which row triggers do not see |

## 3. Ownership and scope

| Owner | Tables | How queries filter |
|---|---|---|
| Plan (team scope) | plans and their teams and months, campaigns, initiatives, activities, line items and their months, budget allocations, approval rounds and history (through their activity), financial transactions (through their line item) | the `…InScope` reads of each store call `plan_in_scope()` (authz.CanView), `activity_in_scope()` (CanViewActivity: plan scope, activity owner, named approvers) or, for transactions, also `plan_in_permission_scope()` with finance.view; each has an isolation test (FR-009) |
| Person | preferences, sessions, notifications, delegations, All-work preferences, saved views | by the person's id in SQL (`NotificationsOf`, `ListViews`, …) |
| Organisation | people, teams, settings, roles and assignments, workflows, global budgets, imports, counters, activity log, recycle bin | none in SQL; the service checks the permission |

The interfaces' unscoped reads stay for internal work that must see every record (usage counts, restores,
the seed); the services switch their person-facing reads to the scoped ones when they move to PostgreSQL
(tasks T025a).

## 4. Soft references (no foreign key)

| Column | Why |
|---|---|
| every `*_by_id`, `actor_id`, actors inside `json` | history; `system` is not a person |
| `preferences.pinned_plan_ids`, `activities.split_from_id` | ids of deleted plans and activities are kept |
| `approval_workflow_levels.role_id` | access refuses to delete only a role people hold, not one a workflow names |
| `activities.approval_workflow_id` | a workflow counts only stored activities as users, so an activity in the recycle bin may name a deleted workflow and must stay restorable |
| `approval_rounds.activity_id`, `approval_rounds.workflow_id`, `approval_actions.activity_id`, `notifications.activity_id` | they outlive an activity in the recycle bin; the workflow is a snapshot |
| `budget_allocations.global_budget_id` | a plan in the recycle bin keeps its allocation; the global budget, then without stored allocations, may be deleted |
| `financial_transactions.line_item_id`, `financial_transactions.currency` | as GSAP sent them: the line item may be in the recycle bin, and a currency used only by transactions may be deleted |
| ids inside `attributes`, `path`, `snapshots`, `levels`, `facts`, `state` | values of a document (R10); the services validate them |

Each foreign key was checked against the area's delete and restore rules; the rows above are the ones
whose target can leave first.
