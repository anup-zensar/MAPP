# Data Model: PostgreSQL Storage (023)

The relational schema for every record type the JSON store holds today: its 39 tables (named in
`internal/*/store_json*.go`) become 52 tables, because the lists a record owns become child tables.
Fields and rules come from each
area's stored struct and its services; [research](research.md) R3, R4, R9 and R10 explain the
conventions. Migration numbers give the order: a table comes after every table it references.

## 1. Conventions

| Go value | Column(s) | Constraint |
|---|---|---|
| Readable id `P-1001` | `id text PRIMARY KEY` | `CHECK (id ~ '^P-[0-9]+$')` with the area's prefix (`ids.Prefix…`) |
| `record.Meta` (marked **M**) | `version integer`, `created_at timestamptz`, `created_by_id text`, `created_by_name text`, `updated_at timestamptz`, `updated_by_id text`, `updated_by_name text` | all `NOT NULL`; `CHECK (version >= 1)`; actor ids are not foreign keys (`system` is no person) |
| `actor.Actor` field `x` | `x_id text`, `x_name text` | both `NULL` together when the Go field is a pointer |
| `money.Amount` `x` | `x_minor bigint` (cents) | `CHECK (x_minor >= 0)` where the rule says "0 or more" |
| `money.Money` `x` | `x_minor bigint`, `x_currency text` | currency `REFERENCES currencies (code)`; both `NULL` together when optional |
| `money.Rate` / `money.Percent` | `numeric(13,6)` / `numeric(5,2)` | ranges as in the rules |
| `date.Date` / `*date.Date` | `date NOT NULL` / `date NULL` | |
| month `"2026-03"` | `date` (first of the month) | `CHECK (extract(day FROM x) = 1)` |
| `time.Time` | `timestamptz` | returned in UTC (R3) |
| enumeration | `text` | `CHECK (x IN (…))`, `''` included where Go allows it |
| optional reference `""` / `nil` | `x_id text NULL` | foreign key, unless listed as a soft reference (§4) |
| text where `""` means none | `text NOT NULL DEFAULT ''` | |
| list the record owns | child table, `position integer` when ordered | `ON DELETE CASCADE`; the parent's `Put` replaces the rows |

Every foreign key is `DEFERRABLE INITIALLY DEFERRED` (R4). "Unique ignoring case" is a unique index on
`lower(col COLLATE "C")`, never stricter than the service rule (R4). Lists are read
`ORDER BY length(id), id COLLATE "C"`, the JSON store's order.

## 2. Tables by migration

### 00001 — samples (built: pilot)

```text
sample_folders  id SF-n · name text NOT NULL CHECK (name <> '') · M
                UNIQUE lower(name COLLATE "C")
sample_notes    id SN-n · folder_id → sample_folders · title text NOT NULL CHECK (title <> '') · M
                INDEX (folder_id)
```

### 00002 — platform: readable ids, activity log, recycle bin

```text
counters        prefix text PK CHECK (prefix ~ '^[A-Z]+$') · last integer NOT NULL CHECK (last >= 1000)
activity_log    id LOG-n · seq bigint NOT NULL UNIQUE · at timestamptz · actor_id, actor_name ·
                action text CHECK IN (created, changed, statusChanged, moved, copied, deleted, restored,
                  deletedPermanently, emptiedRecycleBin) ·
                verb, record_type, record_type_label, record_id, record_name text NOT NULL ·
                path jsonb (array) · changes jsonb (array) · items jsonb (array) ·
                comment, details text NOT NULL DEFAULT ''
                INDEX (record_type, record_id) · INDEX (actor_id) · INDEX (at) · GIN (path)
                append-only: a trigger refuses UPDATE and DELETE (011 FR-023)
recycle_bin     id BIN-n · record_type, record_type_label, record_id, record_name text NOT NULL ·
                path jsonb (array) · parent jsonb NULL (object) · deleted_at timestamptz ·
                deleted_by_id, deleted_by_name · contents jsonb (array) · snapshots jsonb (array) ·
                resolved_how text NULL · resolved_by_id, resolved_by_name NULL · resolved_at NULL
                CHECK resolution columns all NULL or all set · INDEX (record_type, record_id)
```

`Increment` is one statement: `INSERT … VALUES ($1, 1001) ON CONFLICT (prefix) DO UPDATE SET last =
counters.last + 1 RETURNING last`. `Entries` reads `ORDER BY seq`.

### 00003 — people, teams, sessions

```text
people             id U-n · name text NOT NULL CHECK (name <> '') · email text NOT NULL ·
                   job_title text NOT NULL DEFAULT '' · active boolean ·
                   manager_id → people NULL CHECK (manager_id <> id) · M
                   UNIQUE lower(email COLLATE "C")
teams              id T-n · name text NOT NULL · description text NOT NULL DEFAULT '' ·
                   colour text CHECK IN (blue, violet, teal, magenta, orange, green, slate, gold) · active · M
                   UNIQUE lower(name COLLATE "C")
team_memberships   id TM-n · team_id → teams · person_id → people · M · UNIQUE (team_id, person_id)
preferences        person_id PK → people · theme CHECK IN (system, light, dark) · sidebar_collapsed boolean ·
                   display_currency text NOT NULL DEFAULT '' · date_format CHECK IN ('', DD MMM YYYY,
                   DD/MM/YYYY, MM/DD/YYYY, YYYY-MM-DD) · home_scope CHECK IN ('', myTeams, everything) ·
                   history_include_inside boolean
preference_pinned_plans  person_id → preferences CASCADE · position integer · plan_id text (soft)
                   PK (person_id, plan_id) · UNIQUE (person_id, position)
sessions           id text PK CHECK (id ~ '^[0-9a-f]{64}$') (SHA-256 of the token) · person_id → people ·
                   created_at · last_used_at
```

`display_currency` stays a plain code because `''` means USD and currencies come in 00004.

### 00004 — settings

```text
currencies             code text PK CHECK (code ~ '^[A-Z]{3}$') · name text NOT NULL ·
                       symbol text NOT NULL DEFAULT '' · active boolean · M
                       UNIQUE lower(name COLLATE "C") · CHECK (code <> 'USD' OR active)
rate_imports           id RI-n · file_name text NOT NULL · feed text CHECK (feed = 'gsapDailyRatesFile') ·
                       first_date, last_date date NULL · added, replaced, unchanged, not_used, problems
                       integer CHECK (>= 0) · M
exchange_rates         id XR-n · currency → currencies CHECK (currency <> 'USD') · date date ·
                       rate numeric(13,6) CHECK (rate > 0 AND rate <= 1000000) ·
                       source CHECK IN (manual, gsapImport) · import_id → rate_imports NULL · M
                       UNIQUE (currency, date) · CHECK ((source = 'manual') = (import_id IS NULL))
field_help             key text PK · text text NOT NULL DEFAULT '' · M
activity_types         id AT-n · name text NOT NULL · default_planned_cost_minor bigint NULL CHECK (>= 0)
                       (USD) · active · M · UNIQUE lower(name COLLATE "C")
models                 id MOD-n · name text NOT NULL · inq_to_mql_rate, mql_to_deal_rate numeric(5,2)
                       CHECK (> 0 AND <= 100) · average_deal_size_minor bigint CHECK (> 0 AND
                       <= 10000000000) (USD) · active · M · UNIQUE lower(name COLLATE "C")
model_activity_types   model_id → models CASCADE · position · activity_type_id → activity_types
                       PK (model_id, activity_type_id) · UNIQUE (model_id, position)
shell_attributes       id SA-n · label text NOT NULL · help_text text NOT NULL DEFAULT '' ·
                       level CHECK IN (plan, campaign, initiative, activity, lineItem) ·
                       type CHECK IN (singleChoice, multipleChoice, multipleChoiceSplit, yesNo, text,
                         longText, number) ·
                       marking CHECK IN (optional, neededToPublish, neededToSubmit) · section text NOT NULL ·
                       position integer CHECK (>= 1) · parent_attribute_id → shell_attributes NULL ·
                       role CHECK IN ('', businessUnit) · active · M
                       UNIQUE (level, lower(label COLLATE "C")) · UNIQUE (role) WHERE role <> '' ·
                       CHECK (type <> 'multipleChoiceSplit' OR level IN ('activity', 'lineItem'))
shell_attribute_options  id SO-n · attribute_id → shell_attributes · label text NOT NULL ·
                       position integer CHECK (>= 1) · active · M
                       UNIQUE (attribute_id, lower(label COLLATE "C"))
shell_attribute_conditions  attribute_id → shell_attributes CASCADE · option_id → shell_attribute_options
                       PK (attribute_id, option_id)
shell_option_parents   option_id → shell_attribute_options CASCADE · parent_option_id →
                       shell_attribute_options · PK (option_id, parent_option_id)
finance_budgets        id FB-n · year integer · name text NOT NULL · M · UNIQUE (year, lower(name COLLATE "C"))
finance_budget_items   id FBI-n · budget_id → finance_budgets · name text NOT NULL ·
                       code text NOT NULL DEFAULT '' · active · M
                       UNIQUE (budget_id, lower(name COLLATE "C"))
```

The legacy `Currency.Rate`, `RateChangedAt` and `RateChangedBy` fields have no columns: `MigrateRates`
already moves them into `exchange_rates`, and a fresh seed has none. "A code is unique within the
budget's year" spans two tables and stays a service rule (safe under serializable writes, R2).

### 00005 — access

```text
roles                   id RL-n · name text NOT NULL · description text NOT NULL DEFAULT '' ·
                        built_in CHECK IN ('', administrator, globalPlanner, planner, approver, viewer) ·
                        default_scope CHECK IN (all, myTeams) · M
                        UNIQUE lower(name COLLATE "C") · UNIQUE (built_in) WHERE built_in <> ''
role_permissions        role_id → roles CASCADE · permission text CHECK IN (the 25 permissions of
                        authz/catalogue.go) · PK (role_id, permission)
role_assignments        id RA-n · person_id → people · role_id → roles ·
                        scope_type CHECK IN (all, myTeams, teams) · M · UNIQUE (person_id, role_id)
role_assignment_teams   assignment_id → role_assignments CASCADE · team_id → teams ·
                        PK (assignment_id, team_id)
access_rules            id text PK CHECK (id = 'access-rules') · no_self_approval boolean ·
                        new_person_role_id → roles NULL · M
```

Permissions are listed in catalogue order by the store, so `role_permissions` needs no position.

### 00006 — plans (plan-owned: the scope starts here)

```text
plans                id P-n · name text NOT NULL · year integer · model_id → models ·
                     owner_id → people NULL · aop boolean · description text NOT NULL DEFAULT '' ·
                     budget_minor bigint NULL · default_finance_budget_id → finance_budgets NULL ·
                     default_spread CHECK IN (evenly, firstMonth, lastMonth, custom) · bcpir boolean ·
                     goal_stage CHECK IN (inq, mql, revenue) · goal_amount text NOT NULL DEFAULT '' ·
                     attributes jsonb CHECK (object) · status CHECK IN (draft, published) ·
                     currency → currencies · plan_type CHECK IN (operational, strategic, crossMarket) ·
                     phasing_method CHECK IN (none, evenly, custom) · M
                     UNIQUE lower(name COLLATE "C")
plan_teams           plan_id → plans CASCADE · team_id → teams · PK (plan_id, team_id) ·
                     INDEX (team_id, plan_id)
plan_phasing_months  plan_id → plans CASCADE · month date · amount_minor bigint · PK (plan_id, month)
```

`plan_teams` is the scope: a plan is in an actor's scope when one of its teams is in the scope's team
list (`EXISTS (SELECT 1 FROM plan_teams WHERE plan_id = plans.id AND team_id = ANY($team_ids))`).
`budget_minor` is a bare amount in the plan's `currency` (as `Plan.Budget`). Legacy `''` currency, plan
type and phasing method read as USD, operational and none: the seed writes the full values.

### 00007 — campaigns and initiatives

```text
campaigns                  id C-n · plan_id → plans · name text NOT NULL · description text NOT NULL
                           DEFAULT '' · owner_id → people NULL · start_date, end_date date NULL ·
                           budget_minor, budget_currency NULL · attributes jsonb CHECK (object) ·
                           phasing_method · M · UNIQUE (plan_id, lower(name COLLATE "C")) ·
                           CHECK ((start_date IS NULL) = (end_date IS NULL)) · CHECK (start_date <= end_date)
campaign_phasing_months    campaign_id → campaigns CASCADE · month · amount_minor · PK (campaign_id, month)
initiatives                id I-n · campaign_id → campaigns · same columns as campaigns ·
                           UNIQUE (campaign_id, lower(name COLLATE "C"))
initiative_phasing_months  initiative_id → initiatives CASCADE · month · amount_minor ·
                           PK (initiative_id, month)
```

Scope: a campaign through its plan; an initiative through campaign → plan (joins, no copied plan id).

### 00008 — approval workflows

```text
approval_workflows                 id WF-n · name text NOT NULL · description text NOT NULL DEFAULT '' ·
                                   active · reminder_days integer NULL CHECK (1..60) ·
                                   escalation_days integer NULL CHECK (1..90) · M
                                   UNIQUE lower(name COLLATE "C") ·
                                   CHECK (escalation_days > reminder_days) when both are set
approval_workflow_levels           workflow_id → approval_workflows CASCADE · number integer CHECK (>= 1) ·
                                   name text NOT NULL DEFAULT '' · approver_type CHECK IN (people,
                                   manager, role) · role_id → roles NULL · mode CHECK IN (any, all) ·
                                   applies_when, auto_approve_when jsonb NULL (object) ·
                                   backup_approver_id → people NULL · PK (workflow_id, number) ·
                                   CHECK ((approver_type = 'role') = (role_id IS NOT NULL))
approval_workflow_level_approvers  (workflow_id, level_number) → approval_workflow_levels CASCADE ·
                                   position · person_id → people · PK (workflow_id, level_number, person_id)
```

### 00009 — activities and line items

```text
activities                id A-n · initiative_id → initiatives · name text NOT NULL ·
                          external_name text NOT NULL DEFAULT '' · activity_type_id → activity_types ·
                          owner_id → people NULL · start_date, end_date date NULL · ongoing boolean ·
                          planned_cost_minor, planned_cost_currency NULL · approval_workflow_id →
                          approval_workflows NULL · description · attributes jsonb CHECK (object) ·
                          inq bigint NULL CHECK (>= 0) · set_later_stages_myself boolean ·
                          mql bigint NULL CHECK (>= 0) · revenue_minor bigint NULL (USD) ·
                          status CHECK IN (created, submitted, declined, approved) ·
                          split_from_id text NULL (soft) · M
                          UNIQUE (initiative_id, lower(name COLLATE "C")) ·
                          CHECK (set_later_stages_myself OR (mql IS NULL AND revenue_minor IS NULL))
line_items                id L-n · activity_id → activities · name text NOT NULL · owner_id → people NULL ·
                          description · currency → currencies · planned_cost_minor bigint NOT NULL
                          CHECK (>= 0) · spread_method CHECK IN (evenly, firstMonth, lastMonth, custom) ·
                          attributes jsonb CHECK (object) · contracted_amount_minor bigint NULL ·
                          request_swbs boolean · finance_budget_year integer NULL ·
                          finance_budget_item_id → finance_budget_items NULL · M
                          UNIQUE (activity_id, lower(name COLLATE "C"))
line_item_spread_months   line_item_id → line_items CASCADE · month · amount_minor CHECK (>= 0) ·
                          PK (line_item_id, month)
```

Scope: activities through initiative → campaign → plan; line items one join further.

### 00010 — approval rounds, actions, notifications, delegations

```text
approval_rounds       id AR-n · activity_id text (soft) · number integer CHECK (>= 1) ·
                      workflow_id text (soft) · workflow_name text · levels jsonb (array, frozen copy) ·
                      current_level integer · outcome CHECK IN (open, approved, declined, recalled,
                      restarted, cancelled) · submitted_at · submitted_by_id, submitted_by_name ·
                      comment text NOT NULL DEFAULT '' · ended_at timestamptz NULL ·
                      facts jsonb NULL (object) · reminder_days, escalation_days integer NULL ·
                      auto_approved boolean · M
                      UNIQUE (activity_id, number) · UNIQUE (activity_id) WHERE outcome = 'open'
approval_actions      id AH-n · round_id → approval_rounds · round_number · activity_id text (soft) ·
                      action CHECK IN (submitted, resubmitted, approved, declined, recalled, restarted,
                      cancelled, autoApproved, skipped, reminded, escalated) · level integer CHECK (>= 0) ·
                      level_name · approvers jsonb (array of actors) · actor_id, actor_name ·
                      on_behalf_of jsonb (array) · at · comment · changes jsonb (array) ·
                      capacity CHECK IN ('', approver, delegate, backup, onBehalf) ·
                      as_delegate_of_id, as_delegate_of_name NULL · rule text ·
                      escalated_to_id, escalated_to_name NULL · waited_days integer CHECK (>= 0)
                      write-once: a trigger refuses UPDATE and DELETE
notifications         id NT-n · recipient_id → people · kind CHECK IN (approvalNeeded, levelApproved,
                      approved, declined, recalled, restarted, autoApproved, reminder, escalated) ·
                      title, text · activity_id text (soft) · activity_name · actor_id, actor_name · at ·
                      read boolean · read_at NULL · CHECK (read = (read_at IS NOT NULL)) ·
                      INDEX (recipient_id, at DESC)
approval_delegations  id DLG-n · person_id → people · delegate_id → people · start_date, end_date date ·
                      note text NOT NULL DEFAULT '' · cancelled_at NULL · cancelled_by_id,
                      cancelled_by_name NULL · M
                      CHECK (end_date BETWEEN start_date AND start_date + 90) ·
                      EXCLUDE USING gist (person_id WITH =, daterange(start_date, end_date, '[]') WITH &&)
                      WHERE (cancelled_at IS NULL)   -- extension btree_gist
```

### 00011 — global budgets

```text
global_budgets      id GB-n · name text NOT NULL · year integer · amount_minor bigint CHECK (>= 0) (USD) ·
                    owner_id → people · description text NOT NULL DEFAULT '' · M
                    UNIQUE lower(name COLLATE "C")
budget_allocations  id BA-n · global_budget_id → global_budgets · plan_id → plans UNIQUE ·
                    amount_minor bigint CHECK (>= 0) · amount_currency → currencies · M
```

An allocation goes to the recycle bin with its plan (it is the plan's child kind), so its foreign key holds.

### 00012 — financial transactions

```text
financial_imports       id FIM-n · source CHECK IN (file, demo) · feed_name text · file_name text NOT NULL
                        DEFAULT '' · rows, new, updated, unchanged, refused, not_linked, relinked integer
                        CHECK (>= 0) · M
financial_transactions  id FT-n · type CHECK IN (purchaseOrder, invoice, payment, accrual) ·
                        document_number text · line_number text CHECK (line_number ~ '^[0-9]{0,6}$') ·
                        po_number, po_line_number text NOT NULL DEFAULT '' · posting_date date ·
                        accrual_month date NULL · amount_minor bigint (negative for credits) ·
                        currency → currencies · vendor_code, vendor_name, description, swbs_code,
                        tracking_key, sent_line_item_id text NOT NULL DEFAULT '' ·
                        line_item_id text NULL (soft) · matched_by CHECK IN ('', lineItemId, trackingKey,
                        poNumber, swbsCode, manual) · not_linked_reason text NOT NULL DEFAULT '' ·
                        manual_link_by_id, manual_link_by_name, manual_link_at NULL ·
                        import_id → financial_imports · M
                        UNIQUE (type, document_number, line_number) ·
                        CHECK ((type = 'accrual') = (accrual_month IS NOT NULL)) ·
                        CHECK ((matched_by = 'manual') = (manual_link_at IS NOT NULL))
```

Scope: through the linked line item when it is stored (line item → activity → … → `plan_teams`).
Unlinked transactions and those of a deleted line item keep the visibility rule of today's financials
service, which the financials task moves into its queries unchanged.

### 00013 — All work

```text
all_work_preferences  person_id PK → people · columns jsonb (array of column keys) ·
                      sort_column text NULL · sort_direction CHECK IN (asc, desc) NULL · updated_at
saved_views           id SV-n · person_id → people · name text NOT NULL · is_default boolean ·
                      state jsonb CHECK (object) · created_at · updated_at
                      UNIQUE (person_id, lower(name COLLATE "C")) · UNIQUE (person_id) WHERE is_default
```

## 3. Ownership and scope

| Owner | Tables | Filter in every query |
|---|---|---|
| Plan (team scope) | plans, plan_teams, plan_phasing_months, campaigns, initiatives, their months, activities, line_items, line_item_spread_months, budget_allocations, approval_rounds and approval_actions (through their activity), financial_transactions (through their line item) | join to `plan_teams` and `team_id = ANY($scope_team_ids)` (B5); isolation test per read (FR-009) |
| Person | preferences, preference_pinned_plans, sessions, notifications, approval_delegations, all_work_preferences, saved_views | `person_id` / `recipient_id` = the actor (from the session, never the request) |
| Organisation | people, teams, settings tables, roles and assignments, workflows, global budgets, financial imports, counters, activity log, recycle bin | none in SQL; the service checks the permission (and filters the log and bin by plan scope as today) |

## 4. Soft references (no foreign key)

| Column | Why |
|---|---|
| every `*_by_id`, `actor_id`, actors inside `jsonb` | history; `system` is not a person |
| `preference_pinned_plans.plan_id` | ids of deleted plans are kept |
| `activities.split_from_id` | the source activity may be deleted |
| `approval_rounds.activity_id`, `approval_rounds.workflow_id`, `approval_actions.activity_id`, `notifications.activity_id` | rounds, actions and notifications outlive an activity in the recycle bin; the workflow is a snapshot |
| `financial_transactions.line_item_id` | may point at a line item in the recycle bin (015 `lineItemDeleted`) |
| ids inside `attributes`, `path`, `snapshots`, `levels`, `facts`, `state` | values of a document (R10); the services validate them |

The area task checks each foreign key above §4 against the area's delete rules with a store test (a
delete that the service allows must not be refused by the database); a reference whose target can leave
first moves to this table.
