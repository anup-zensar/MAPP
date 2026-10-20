# Implementation Plan: Approval Rules

**Branch**: `017-approval-rules` | **Date**: 2026-10-20 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/017-approval-rules/spec.md` (amends 008, 014 and 003)

**Design artifacts**: [data-model.md](data-model.md) · [contracts/openapi.yaml](contracts/openapi.yaml)
(fragment: 7 new operations; extended schemas answered by 008's and 003's operations) ·
[tasks.md](tasks.md). Conventions are 001's and 008's ([008 plan](../008-activity-approvals/plan.md)); the
build guide [docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md) §7 applies; access rules are
014's ([014 plan](../014-access-control/plan.md): `platform/authz`, `internal/access`, Table C). This plan
only adds what is new or changed.

## Summary

Feature 017 extends the built approvals of feature 008 (package `internal/approvals`, web folders
`features/approvals`, `features/approval-workflows`, `features/notifications`) on top of feature 014's
access control, along the client's three parts:

- **Approval workflow** — each level gets an approver source (named people, the owner's line manager, or a
  014 role that approves), a mode (any one / all must approve), an "only needed when" condition, an
  "approve automatically when" condition and a backup approver; each workflow gets reminder and
  escalation days. A pure rule engine (`rules.go`) evaluates the levels once at submission and the
  resolver (`resolve.go`) finds the approvers, leaving out inactive people, people who cannot approve, and
  (014's access rule) the owner and the submitter.
- **Delegation and control** — a new record, approval delegations (`delegations.go`), and one authority
  function (`authority.go`) holding 014's approve/decline check order, extended with delegates and the
  backup approver after escalation, and with the submitter in segregation of duties.
- **Tracking and notification** — a job endpoint that sends reminders and escalations (`escalations.go`),
  a Monitoring tab computed on read (`monitoring.go`), new notification kinds, and every event in the
  approval history and the activity log (011 shows them).

People (003) gain a **line manager** field. **Feature 014 must be merged first**: 017 reads the signed-in
person's permissions from 014's `authz` grants and other people's permissions and role holders through a
consumer port, `approvals.Access`, implemented over 014's `internal/access` hooks.

## Technical Context

**Language/Version**: Go 1.27; TypeScript 5.9 strict + React 19 — as 001

**Primary Dependencies**: none new. `northstar/internal/platform/authz` (014). Icons from the existing
`lucide-react`.

**Storage**: JSON file store. One new table `approval_delegations`; new fields in `approval_workflows`,
`approval_rounds`, `approval_actions`, `notifications` (kinds) and `people` — all read with defaults
when missing (data-model §10), so no migration.

**Testing**: Go `testing` (table-driven rule tests, service tests with `fakeActivities`, a fake `Access`
and `authz` grants in the context; handler tests with `apitest` on the seeded app); Vitest + Testing
Library (`mockApi`); Playwright + axe in `frontend/e2e/approval-rules.spec.ts`.

**Target Platform / Project Type / Constraints**: as 008 — 127.0.0.1 only, no third runtime part, fixed
clock in tests (2026-03-12 09:00 UTC), WCAG 2.2 AA.

**Performance Goals**: every approvals call under 100 ms with demo data; the job over the demo data
under 200 ms.

**Scale/Scope**: 3 demo workflows, 1 delegation; 7 new API operations, and 008's and 003's operations
answering extended schemas (table *API operations*); 1 new tab, 1 new side panel, extended workflow
editor, chain, history, dialogs and people form.

## Constitution Check

| Principle | How this plan complies | Result |
|---|---|---|
| I. Spec first | Every element traces to a 017 FR (traceability table); every 008, 014 and 003 FR it changes is named in the spec's amendment index; defaults are Proposed decisions below | Pass |
| II. Shell business, plain language | Plan → … → Activity vocabulary; "level", "round", "delegate", "backup approver" as defined in the spec; all texts in *User-visible messages* | Pass |
| III. Simple, friendly experience | Rules are chosen with plain choices ("Who approves", "Only needed when", "Approve automatically when"); the submit dialog says what will happen to every level; nothing is disabled without the reason; all errors at once | Pass |
| IV. Simple, layered architecture | Same package and layers as 008; all rules in the service; the job is an HTTP endpoint, not a worker or scheduler; 014 reached through `platform/authz` and a consumer port with an adapter in `internal/app` (D-196) | Pass |
| V. Tested business rules | Table-driven tests for conditions, resolution and authority; one test per FR; handler test per operation; e2e for P1 stories | Pass |
| VI. Safe by default | Server-side authority and segregation of duties (014's checks, extended); every event logged with the actor (NorthStar for automatic ones). Relies on 014's amendment of Principle VI ("no roles yet") | Pass (with 014) |
| VII. Keep it small | No new dependency; delegation is one record; monitoring is computed on read; no scheduler; no new 014 permission | Pass |

## Decisions

### Decisions (numbered by the consolidation step: D-276–D-297 in docs/sdd/decision-log.md)

| Tag | Proposed decision | Reason |
|---|---|---|
| 017-P1 | **D-276**: Rules are evaluated once, at submission, on the activity's planned cost (USD), activity type and the plan's business units, and the outcome, reasons, resolved approvers and those facts are frozen in the round. Only the changes of domain model §2.5 restart approval; a change of activity type or of the plan's business unit does not re-evaluate an open or approved round. | Predictable and auditable; reuses D-013. Reversible: add those fields to §2.5 |
| 017-P2 | **D-277**: A condition's parts are combined with AND: planned cost at least / below (USD), business unit is one of (any of the plan's business units), activity type is one of. An activity without planned cost matches no cost part. | Covers the client's examples with one simple form |
| 017-P3 | **D-278** (replaces D-205): A level has one approver source — named people, the owner's line manager, or a 014 role that includes `approvals.decide`, resolved to the active holders whose scope covers the plan — and a mode, any one (default) or all must approve. Line-manager levels have one approver. | Client comment 1 (auto/manager); BRD parallel and role-based approvals |
| 017-P4 | **D-279**: When no level needs a decision, the activity goes from Created or Declined straight to Approved at submission. The log gets two entries in the same save (the submitter's "submitted", NorthStar's "approved automatically"); the history gets one `autoApproved` or `skipped` entry per level by NorthStar; the owner is notified only when someone else submitted. Domain model §2.3 gains this transition. | The approval is visible as a rule decision, not as the submitter's |
| 017-P5 | **D-280** (extends 014 D-A9): 014's access rule "Nobody approves an activity they own or submitted" (on by default) bars the owner **and the round's submitter** in every capacity (approver, delegate, backup approver, on behalf). While it is on, both are left out when approvers are resolved. | Maker-checker control; 014's *Access rules* entity already names both |
| 017-P6 | **D-281** (extends 014 D-A9): Who may decide, in order: waiting resolved approver holding `approvals.decide`; current delegate of one, holding it; the backup approver once the level is escalated, holding it; someone holding `approvals.decideOnBehalf` for the plan (recorded on behalf). 014's refusals (403 messages 5 and 6) stay; 017 adds "You already approved level <n>…" and the submitter message. Submit and recall stay 014's (`activities.submit`). | One check order, owned by 014, extended |
| 017-P7 | **D-282**: "Manager" is a `managerId` field on the person, kept in NorthStar and changed with 014's `people.manage` in Settings › People; no self, no loop. Microsoft Entra ID's manager attribute replaces it with real sign-in. | SSO is simulated (demo sign-in) |
| 017-P8 | **D-283**: A level whose source names nobody (no line manager; unknown role; nobody holds the role for the plan) blocks the submission with a problem. A level whose candidates are all left out (inactive, cannot approve, owner, submitter) takes its usable backup approver, or else waits with **no approver** for someone deciding on behalf, with 014's advice; the submission is not blocked. | "Ask an administrator to fix the workflow" vs 014's "an administrator approves on your behalf" |
| 017-P9 | **D-284**: Approval delegation = delegating person, one delegate who holds `approvals.decide`, inclusive date range in UTC calendar dates, at most 90 days, no overlap per person, no chains; the delegating person keeps every right. Delegations are cancelled, never deleted, and kept for the audit trail. People holding `workflows.manage` manage anyone's. | Delegation of authority with a clear audit trail; no new permission |
| 017-P10 | **D-285**: Reminder and escalation days are set per workflow (new workflows 3 and 5; off when empty), counted in calendar days of 24 hours since the level was reached, copied into the round. At most one reminder and one escalation per level; escalation gives the level's backup approver authority over the whole level; when both are due in one run only the escalation is sent. | Simplest rule that answers "escalations" |
| 017-P11 | **D-286**: Reminders and escalations run when `POST /api/approvals/escalations/run` is called — from **Run reminders and escalations** on Approvals › Monitoring, by people holding `workflows.manage`. Its events are by NorthStar, with "run by <person>" in the log details. No scheduler now (constitution IV); when NorthStar is hosted, a scheduled Azure Container Apps job calls the endpoint (ADR then). | Simulated integration with a seam for the real schedule |
| 017-P12 | **D-287**: A level is overdue when it has waited at least the round's escalation days, or without escalation its reminder days; otherwise never. | One definition for rows, cards and lists |
| 017-P13 | **D-288**: Refusals because of who the person is use 014's 403 `forbidden`; refusals because of a record's state (an ended or cancelled delegation) use 409 `not_allowed`. | Same split as 014 (403 for permissions, 409 for record rules such as allocation locks) |
| 017-P14 | **D-289** (extends D-167): every approval event is logged on the activity with action `statusChanged` and its own verb — "approved automatically", "sent a reminder about", "escalated" (by NorthStar); decisions keep 008's verbs with the capacity in the details. Delegations are history kind `approvalDelegation` (created, changed, verb "cancelled"). | 011 shows and filters them with no change |
| 017-P15 | **D-290**: `approvals` reads the signed-in person's permissions and 014's access rule from `authz.From(ctx)`, and other people's from `approvals.Access` (`CanApprove`, `ApprovingRoles`, `RoleHolders`), implemented in `internal/app/approvals_access.go` over 014's `internal/access` hooks. Without grants in the context (internal calls) everything is allowed and the rule is on. | D-196 seam rule; 014 D-A6 |
| 017-P16 | **D-291**: Approval delegations get readable IDs `DLG-`. | 001 R4 |
| 017-P17 | **D-292**: A delegation itself sends no notification; banners on the Approvals page tell the delegate and the delegating person. `Notification.activity` stays required. | Keeps the notification contract unchanged |
| 017-P18 | **D-293**: Monitoring is computed on read over the rows of **All open** (014-filtered); "Average time to decision" leaves out rounds approved at submission. | Demo-sized data; always consistent with the tabs |
| 017-P19 | **D-294**: Data stored before 017 reads with the defaults of data-model §10; no migration. | Existing rounds keep behaving exactly as before |
| 017-P20 | **D-295**: The seed keeps WF-1001/WF-1002's levels (the core-flow journeys stay valid), adds timings and backups, a third workflow WF-1003 "Standard marketing approval" showing every rule, line managers and one planned delegation (data-model §13). Every named, backup and delegate person in the seed holds `approvals.decide` (014 seed). | Demonstrable without breaking seeded stories |
| 017-P21 | **D-296**: The 7 new operations get these 014 Table C rules: rule options, monitoring — Signed in (monitoring filtered); delegations — Custom (own: signed in; others' and `scope=all`: `workflows.manage`); run escalations — Need `workflows.manage`. No new 014 permission. | 014 FR-010 fails closed on undeclared operations |
| 017-P22 | **D-297**: BUS.WFL.030 is met by 008 FR-010 and 004 FR-024; 017 only adds problems. Q-005 (field markings) stays open. | Gap analysis §3.11 |

### Decisions and questions this reopens

- **D-205 / Q-002** (ordered levels, any one approver, no thresholds) → replaced by 017-P2, 017-P3 with
  Q-014's defaults. **D-020** was already replaced by 014 D-A9; 017-P5/P6 extend it. **D-162** extended by
  017-P8 (inactive approvers are left out at submission; those who become inactive later stay).
- Documentation the consolidation step updates (not this feature's agents): domain model §2.3 (Created /
  Declined → Approved at submission), §6 (approval workflow), §8; feature map (017 row);
  demo-users-and-data (line managers, WF-1003, delegation); 014 FR-019 rule label ("Nobody approves an
  activity they own or submitted", as 014's entity already reads).

### Defaults for Shell to confirm (Q-014)

1. Conditions use planned cost in USD, the plan's business units and the activity type, combined with AND.
2. Demo thresholds only: automatic below $10,000.00; extra level from $100,000.00.
3. Rules are evaluated once at submission; only planned cost, dates and line-item amounts/currencies restart approval.
4. "Manager" is a line manager kept in NorthStar (from Entra ID later).
5. "All must approve" means every resolved approver; one decline declines the activity.
6. Segregation of duties (014's rule, on by default) bars the owner and the submitter, in every capacity.
7. People allowed to decide on behalf (Administrators) can decide any level, recorded "on behalf of".
8. A workflow level that names nobody (no line manager, nobody with the role) blocks submission; a level whose people are all left out does not.
9. Delegation: one delegate at a time who can approve, at most 90 days, no chains, the delegating person keeps their rights.
10. Reminder after 3 days, escalation after 5 days, calendar days, once each per level.
11. Escalation goes to the level's backup approver, who can then decide the whole level.
12. The reminder/escalation job is started by someone who can change approval workflows until hosting provides a nightly schedule.
13. Notifications are in-app only; e-mail and Teams come with BUS.WFL.040.
14. Overdue = waited at least the escalation days (or the reminder days without escalation).
15. People who can change approval workflows also manage other people's delegations.
16. Dates of delegations and "today" are UTC calendar dates.

## Project Structure

### Documentation (this feature)

```text
specs/017-approval-rules/
├── spec.md · plan.md · data-model.md · tasks.md
└── contracts/openapi.yaml    # fragment merged into backend/api/openapi.yaml
```

### Source code

```text
backend/
├── api/openapi.yaml                         # + fragment (7 new ops, schema extensions)       (shared)
├── internal/approvals/
│   ├── model.go           # + Condition, Level/RoundLevel/Round/Action fields, Delegation, API types (changed)
│   ├── ports.go           # ActivityFacts + Plan (authz.PlanFacts), ActivityTypeID, BusinessUnits; Access; RuleChoices (changed)
│   ├── repository.go · store_json.go   # + approval_delegations table                       (changed)
│   ├── rules.go           # NEW pure: match, conditionText, levelsText, evaluate
│   ├── resolve.go         # NEW resolve approvers, excluded, backup, noApprover, problems
│   ├── authority.go       # NEW 014's check order + 017 extensions; Waiting-for-me membership; NamedIn
│   ├── delegations.go     # NEW delegations list/create/update/cancel, current delegations by day, kind
│   ├── escalations.go     # NEW RunEscalations
│   ├── monitoring.go      # NEW Monitoring
│   ├── options.go         # NEW RuleOptions
│   ├── workflows.go · rounds.go · view.go · inbox.go · notifications.go · kinds.go · messages.go · handler.go (changed)
│   └── *_test.go          # rules_test, resolve_test, authority_test, delegations_test, escalations_test,
│                          # monitoring_test, + additions to existing tests; handler_rules_test.go
├── internal/access/routes.go · hooks.go     # Table C rules for the 7 operations; RoleHolders hook if missing (014, additive)
├── internal/people/       # Person.ManagerID, validation, views, usage role "manager"          (003, changed)
├── internal/app/
│   ├── approvals_adapter.go        # ActivityFacts additions; ruleChoices over settings       (changed)
│   ├── approvals_access.go         # NEW: approvals.Access over internal/access
│   └── app.go                      # wiring UseAccess, UseRuleChoices                         (shared)
├── internal/platform/ids/ids.go    # PrefixApprovalDelegation = "DLG"                         (shared)
└── seed/approvals.go · people step   # data-model §13                                         (shared)

frontend/src/
├── api/schema.d.ts                                  # regenerated
├── features/approval-workflows/
│   ├── LevelsEditor.tsx (+css, test)                # approver source, mode, conditions, backup (changed)
│   ├── RuleConditionFields.tsx (+css, test)         # NEW
│   ├── WorkflowFormPanel.tsx (+test)                # reminders and escalation section     (changed)
│   ├── WorkflowDetailsPanel.tsx · ApprovalWorkflowsPage.tsx · levelDrafts.ts · workflowTexts.ts · workflowsApi.ts (changed)
├── features/approvals/
│   ├── MonitoringView.tsx (+css, test) · AgeingBars.tsx (+css, test)   # NEW
│   ├── DelegationPanel.tsx (+css, test) · DelegationForm.tsx (+test) · delegationsApi.ts   # NEW
│   ├── DelegationBanner.tsx (+test) · RunEscalationsButton.tsx (+test)                       # NEW
│   ├── ApprovalsPage.tsx · ApprovalsTable.tsx · ApprovalChain.tsx · ApprovalHistory.tsx ·
│   │   ApprovalSection.tsx · ApprovalPreviewPanel.tsx · DecisionDialog.tsx · SubmitDialog.tsx ·
│   │   DecisionButton.tsx · approvalsApi.ts · approvalTypes.ts · messages.ts (changed)
├── features/notifications/NotificationList.tsx (+test)   # icons and texts of new kinds      (changed)
├── features/people/…                                     # Line manager field and details lines (003, changed)
frontend/e2e/approval-rules.spec.ts                      # NEW
```

## Backend design

### Ports (`ports.go`)

```go
// ActivityFacts gains (filled by internal/app/approvals_adapter.go):
Plan           authz.PlanFacts // the activity's plan: ID, TeamIDs, OwnerID, CreatedByID (014 scope checks)
ActivityTypeID string          // "" when empty; TypeName already exists
BusinessUnits  []Choice        // the plan's Business unit values: settings.BusinessUnit(plan.Attributes) with option labels

type Choice struct{ ID, Label string; Active bool }

// Access is what approvals needs to know about OTHER people from feature 014 (the signed-in person's own
// permissions come from authz.From(ctx)). Implemented in internal/app/approvals_access.go.
type Access interface {
    CanApprove(ctx context.Context, personIDs []string) (map[string]bool, error) // hold approvals.decide (014 hook CanApprove)
    ApprovingRoles(ctx context.Context) ([]Choice, error)                         // 014 roles that include approvals.decide
    RoleHolders(ctx context.Context, roleID string, plan authz.PlanFacts) ([]string, error) // active holders whose scope of roleID covers the plan
}

// RuleChoices lists what conditions may name (implemented over feature 002's readers).
type RuleChoices interface {
    ActivityTypes(ctx context.Context) ([]Choice, error) // settings.ActivityTypes
    BusinessUnits(ctx context.Context) ([]Choice, error) // options of the attribute with Role businessUnit
}
```

`(*Service).UseAccess(Access)` and `(*Service).UseRuleChoices(RuleChoices)` set them; `NewService` keeps its
signature (defaults: an `Access` that allows everyone and has no roles, and empty choices), so 008's tests
keep compiling. `app.New` always wires the real ones. People's line manager is read from
`people.Person.ManagerID` through the existing `people.Repository`. 014's `authz` helpers used:
`authz.From`, `Grants.Has(authz.ApprovalsDecide)`, `Grants.InScope(authz.ApprovalsDecideOnBehalf, plan)`,
`Grants.Has(authz.WorkflowsManage)`, `Grants.Rules.NoSelfApproval` (D-302), `authz.Require` (use the names 014
delivered; T001).

### Rule engine and resolution (`rules.go`, `resolve.go`)

- `func (c Condition) Matches(f RuleFacts) bool`; `func conditionText(c Condition, names choiceNames) string`;
  `func levelsText(levels []Level, d directory, names choiceNames) string` — data-model §2.
- `func evaluate(levels []Level, f RuleFacts) []levelPlan` → per level `resolution` and `ruleText`
  (data-model §3 steps 1–3). Pure; table-tested (SC-001).
- `func (s *Service) resolve(ctx, plan []levelPlan, facts ActivityFacts, submitter actor.Actor, rule bool,
  d directory) ([]RoundLevel, []SubmitProblem, error)` — steps 3–6, `noApprover`, the problems table and the
  advice (uses `Access.RoleHolders`, `Access.ApprovingRoles` for names, `Access.CanApprove`,
  `d[owner].ManagerID`). Source texts: "line manager of <owner>", "role <name>".
- `submitCheck` (rounds.go) runs both after 008's checks; resolution problems follow the workflow problem.

### Authority (`authority.go`)

```go
type Capacity string // "approver" | "delegate" | "backup" | "onBehalf"
type Authority struct {
    Allowed    bool
    Capacity   Capacity
    For        *actor.Actor  // delegate: the approver acted for
    DelegateOf []actor.Actor // every waiting approver who delegated to the person today
    OnBehalfOf []actor.Actor // onBehalf: the waiting approvers
    Refusal    *errs.Error   // 403 forbidden with the data-model §6 message; nil when allowed
    Shown      string        // the reason for actions.approve/decline ("" hides the buttons, 014)
}
func (s *Service) authority(ctx, a actor.Actor, facts ActivityFacts, round Round, entries []Action,
    delegations []Delegation, today date.Date) (Authority, error)
func (s *Service) NamedIn(ctx, personID, activityID string) (bool, error) // 014 visibility hook, extended (FR-021)
```

Data-model §6 is the specification. If 014's build already put its check order into `decide`, move it into
`authority` unchanged and add 017's steps 3b, 3c, 3e and the submitter in step 4. `NamedIn` answers true for a
person named in any round of the activity (014), a current delegate of a waiting approver of its open round,
or the backup approver of its escalated current level; 014's `activities.Approvers.NamedIn` adapter calls it.
`currentDelegations(ctx, today) ([]Delegation, error)` reads the table once per operation.

### Service operations (new and changed)

Every write stays one `s.tx.Write` with its history entries, log entries and notifications (008).

| Operation | Rules (FR) | Errors |
|---|---|---|
| `CreateWorkflow`, `UpdateWorkflow` (changed) | 014's route rule (`workflows.manage`) and its approver check (014 FR-028) stay; add data-model §1–§2 validation all at once (FR-008, FR-025): backup approver holds `approvals.decide`, role includes it; normalise: clear `approverIds` unless `people`, `roleId` unless `role`, `mode = any` for `manager`; `Describe` gives the levels text (FR-009, FR-035) | `validation_failed`, `version_conflict`, gone |
| `ListWorkflows`, `GetWorkflow` (changed) | levels views with the new fields and texts; timings; warnings (data-model §2) | — |
| `RuleOptions(ctx, a)` (new) | `{activityTypes, businessUnits, roles}` from `RuleChoices` and `Access.ApprovingRoles` | — |
| `Submission` (changed) | + `outcomes` (one per level: number, label, outcome, approvers, source, mode, ruleText, excluded, backupUsed, noApprover, advice) and `approvedAutomatically`; resolution problems in `problems` (FR-012, FR-013) | gone |
| `Submit` (changed) | Re-run check + evaluate + resolve in the Write; problems → `validation_failed` (FR-012). Round with `facts`, timings, levels (FR-010). Entries: `submitted`/`resubmitted` at the first `decide` level (level 0, no approvers, when none), then one `autoApproved`/`skipped` per such level by NorthStar with `rule`. **A `decide` level exists**: `startedAt` = now on it, `currentLevel` = it, `SetStatus(submitted)`, log `submitted` (details below), notify its approvers and their current delegates (FR-032). **None**: round `approved`, `autoApproved` true, `endedAt` now, `SetStatus(approved)`, log the submitter's `submitted` (Created/Declined → Submitted) and NorthStar's `approved automatically` (Submitted → Approved), notify the owner `autoApproved` when owner ≠ submitter (FR-014, 017-P4) | 008's |
| `Approve`, `Decline` (changed) | `authority` (data-model §6); refused → its 403. Entry with `capacity`, `asDelegateOf`, `onBehalfOf` (on behalf only). Approve: level done per data-model §4 → next `decide` level (`startedAt` now, notify its approvers + delegates, owner `levelApproved`) or Approved (owner `approved`); not done → round `Touch` only, log verb `approved level <n> of` with "<a> of <b> approved", no notification (FR-015, FR-032). Decline: as 008, any capacity | `forbidden` + 008's |
| `Approval` (changed) | Viewer gains `canDecide`, `capacity`, `asDelegateOf`, `cannotDecideReason` (= `Authority.Shown`), `isSubmitter`; `onBehalfOf` only for capacity `onBehalf`. `actions.approve` / `actions.decline` = `{available: Submitted ∧ canDecide, reason: Shown}`. Chain levels with the new fields (FR-033); history with new fields (FR-034); round `facts` | gone |
| `Inbox` (changed) | `waiting` = Submitted with `authority` capacity approver, delegate or backup (FR-031); rows gain `asDelegateOf`, `escalatedTo`, `overdue`, `waitingDays`, `noApprover`, `round.mode`, `round.approvedCount`, `round.requiredCount` | 008's |
| `Summary` (changed) | `waitingForMe` = the `waiting` tab count (SC-007 holds) | 008's |
| `Monitoring(ctx, a, monthStart, monthEnd)` (new) | Data-model §11 over the `open` tab's rows (014-filtered); `overdue` rows oldest waiting first; `canRunEscalations` = grants hold `workflows.manage` (FR-029, FR-030) | `validation_failed` (bounds, as `Summary`) |
| `RunEscalations(ctx, a)` (new) | Route rule Need `workflows.manage`. In one Write, open rounds of stored activities by activity ID: current `decide` level; waited = ⌊(now − waiting since)/24 h⌋. **Escalation due** (`escalationDays` set, waited ≥, `escalatedAt` null): `backup` usable (active, `CanApprove`, not owner/submitter while the rule is on, not a waiting approver) → `escalatedAt` = now (and `remindedAt` if null), entry `escalated` (NorthStar, `escalatedTo`, `waitedDays`), log, notify (data-model §8); else add to `notEscalated`. **Else reminder due** (`reminderDays` set, waited ≥, `remindedAt` null) → `remindedAt` = now, entry `reminded`, log, notify. Returns `{reminded, escalated, notEscalated[], events[]}` (FR-026 – FR-028, SC-005) | `forbidden` |
| `ListDelegations(ctx, a, scope)` (new) | `mine` (default): person or delegate is `a`; `all`: `workflows.manage`. Order: current, planned (by start), then ended and cancelled (latest end first). With `state`, `canChange`, page `canManageOthers` (FR-024) | `validation_failed` (scope), `forbidden` |
| `CreateDelegation`, `UpdateDelegation`, `CancelDelegation` (new) | Data-model §7 rules all at once; ID `DLG`; log `created` / `changed` / `statusChanged` verb `cancelled` with record name "<person> → <delegate>" (FR-022, FR-024, FR-035). Update takes only changed fields + `version` (D-203) | `validation_failed`, `forbidden`, `not_allowed`, `version_conflict`, gone ("This delegation no longer exists.") |

### Access rules for the new operations (added to 014's Table C and `internal/access/routes.go`)

| Operation | Rule |
|---|---|
| GET /api/approval-rule-options | Signed in |
| GET /api/approval-delegations | Custom: `scope=mine` signed in; `scope=all` needs `workflows.manage` (service) |
| POST /api/approval-delegations | Custom: for oneself signed in; `personId` of someone else needs `workflows.manage` (service) |
| PATCH /api/approval-delegations/{delegationId} · POST …/{delegationId}/cancel | Custom: the delegating person, or `workflows.manage` (service) |
| GET /api/approvals/monitoring | Signed in → filtered (the rows of All open) |
| POST /api/approvals/escalations/run | Need `workflows.manage` |

### Activity-log entries (FR-035, extends 008's table)

All on kind `activity`, action `statusChanged`, in the same Write. Capacity suffix on decisions:
"; as delegate of <name>" · "; as backup approver" · "; on behalf of <names>".

| Event | Actor | Verb | Status change | Details |
|---|---|---|---|---|
| Submit, a level waits | submitter | `submitted` / `re-submitted` | → Submitted | "Round <r> · <workflow> · <level notes> · waiting for level <n> (<approvers>, or "no approver")"; level notes "level <k> approved automatically (<reason>)", "level <k> not needed (<reason>)" joined "; " |
| Submit, no level waits | submitter, then NorthStar | `submitted`, then `approved automatically` | → Submitted, then Submitted → Approved | second: "Round <r> · <level notes>" or "Round <r> · no level applies" |
| Approval that does not finish an "all" level | decider | `approved level <n> of` | none | "Round <r> · level <n> of <m> · <a> of <b> approved · still waiting for <names>" + capacity |
| Other decisions | as 008 | as 008 | as 008 | 008's + capacity suffix |
| Reminder | NorthStar | `sent a reminder about` | none | "Round <r> · level <n> of <m> · waited <d> days · to <names> · run by <person>" |
| Escalation | NorthStar | `escalated` | none | "Round <r> · level <n> of <m> · waited <d> days · escalated to <backup> · run by <person>" |

Delegations: kind `approvalDelegation` (`Kinds()` adds it; never moved to the recycle bin, so its
`PrepareRestore` is not used). Workflow and person changes: their kinds' `changed` entries (FR-035).

## API operations

All require a session; 014 adds 403 `Forbidden` to every operation. Schemas: [contracts/openapi.yaml](contracts/openapi.yaml).

| # | Method and path | Request | Response | Errors | New / changed |
|---|---|---|---|---|---|
| 1 | `GET /api/approval-rule-options` | — | 200 `ApprovalRuleOptions` | — | new |
| 2 | `GET /api/approval-delegations?scope=mine\|all` | — | 200 `ApprovalDelegationPage` | 400, 403 | new |
| 3 | `POST /api/approval-delegations` | `ApprovalDelegationCreate` | 201 `ApprovalDelegation` | 400, 403 | new |
| 4 | `PATCH /api/approval-delegations/{delegationId}` | `ApprovalDelegationUpdate` | 200 `ApprovalDelegation` | 400, 403, 404, 409 | new |
| 5 | `POST /api/approval-delegations/{delegationId}/cancel` | `ApprovalDelegationCancel` | 200 `ApprovalDelegation` | 400, 403, 404, 409 | new |
| 6 | `GET /api/approvals/monitoring?monthStart&monthEnd` | — | 200 `ApprovalMonitoring` | 400 | new |
| 7 | `POST /api/approvals/escalations/run` | — | 200 `EscalationRunResult` | 403 | new |
| 8 | `GET/POST /api/approval-workflows`, `GET/PATCH /api/approval-workflows/{workflowId}` | level and timing fields | same schemas, extended | 400 (new field problems) | changed |
| 9 | `GET /api/activities/{activityId}/approval`, `GET /api/approvals/activities/{activityId}` | — | `ActivityApproval` extended | — | changed |
| 10 | `GET /api/activities/{activityId}/approval/submission` | — | `ApprovalSubmission` + `outcomes`, `approvedAutomatically` | — | changed |
| 11 | `POST …/approval/submit` | unchanged | may answer an Approved activity | — | changed |
| 12 | `POST …/approval/approve`, `…/decline` | unchanged | unchanged | 403 (data-model §6) | changed |
| 13 | `GET /api/approvals?tab=` | — | `ApprovalRow` extended | — | changed |
| 14 | `GET /api/notifications` | — | `Notification.kind` + 3 values | — | changed |
| 15 | `GET/POST /api/people`, `GET/PATCH /api/people/{personId}`, `POST /api/people/{personId}/impact` | `managerId` | `Person.manager`, `reportCount`; `UsageCount.role` + `manager` | 400 | changed (003) |

`GET /api/approvals/summary` is unchanged in shape (its count follows FR-031). Handlers follow 008's
(`web.ReadJSON` → `actor.From` → service → `web.WriteJSON`). Remove each served operation from
`notYetServed` in `backend/internal/app/app_test.go`.

## Demo data (seed)

Data-model §13: the people step sets `managerId` after all people exist; `seed/approvals.go` adds the
timings and backups to WF-1001/WF-1002 (stored by 006's seed step), creates WF-1003 through
`approvals.NewJSONRepository()`, fills the new round fields of the seeded rounds (`startedAt`, `backup`,
`facts`, timings, `resolution: decide`, `mode: any`) and writes DLG-1001 after 014's roles; 011's
demo-history step adds the delegation's `created` entry. The seed summary line reads ", 3 approval
workflows, 1 approval delegation".

## Frontend design

### Routes

| Path | Change |
|---|---|
| `/approvals?tab=monitoring` | New fourth pill tab **Monitoring** (`MonitoringView`) |
| `/approvals?panel=delegations` | Opens `DelegationPanel` (header secondary action **Delegate my approvals**) |
| `/settings/approval-workflows(/:id)` | Extended form and details (014 already hides changes without `workflows.manage`) |

### Query keys (`approvalsApi.ts`, `delegationsApi.ts`, `workflowsApi.ts`)

`['approval-monitoring', monthStart]` (`refetchInterval: 30_000`) · `['approval-delegations', scope]` ·
`['approval-rule-options']` (staleTime 5 min). Every new mutation calls 008's `invalidateAllButSession`.
Permission-dependent controls read the server's flags (`canRunEscalations`, `canManageOthers`,
`canChange`) or 014's `useAccess().has('workflows.manage')`.

### Settings › Approval workflows (US1, US2; FR-001 – FR-009, FR-025)

- `LevelsEditor` per level block, in this order: Level name · **Who approves** (radio group: "Named
  people" · "The owner's line manager" · "People with a role") · Approvers (`PersonPicker` multiple) or Role
  (`ChoiceField` of `roles`, the roles that approve) · **When there are several approvers** (radio: "Any
  one approves" · "All must approve"; not shown for the line manager) · **Only needed when** (checkbox
  revealing `RuleConditionFields`) · **Approve automatically when** (checkbox revealing
  `RuleConditionFields`) · **Backup approver** (`PersonPicker` single, hint "Takes over when an approval
  is escalated, or when nobody else can approve.").
- `RuleConditionFields` (props `value`, `onChange`, `options`, `idPrefix`, `errors`): "Planned cost at
  least (USD)" and "Planned cost below (USD)" (`MoneyField`, USD only), "Business unit is one of" and
  "Activity type is one of" (`ChoiceField` multiple; inactive values shown "(inactive)").
- `WorkflowFormPanel` gains a `FormSection` **Reminders and escalation**: `CheckboxField` "Send a
  reminder" + days input ("after <n> days waiting"), `CheckboxField` "Escalate to the backup approver" +
  days input; new workflows start checked with 3 and 5. The form's level hint changes to the text in
  *User-visible messages*. Server problems map to fields by key (`levels.<n>.appliesWhen` → that
  level's condition group, etc.) and to `FormErrorSummary` (FR-008).
- `WorkflowDetailsPanel`: each level shows source, mode, conditions in words, backup; a line "Reminder
  after 3 days · escalation after 5 days" or "No reminders or escalation".

### Activity page and preview (US1–US3, US7; FR-013, FR-017 – FR-020, FR-033, FR-034)

- `SubmitDialog` renders `outcomes`: per level a row with the badge, label, and either approvers +
  source, "Not needed — <reason>" or "Approved automatically — <reason>"; the left-out notes, "The backup
  approver <name> approves this level." or the no-approver advice; when `approvedAutomatically`, the
  sentence "This activity will be approved automatically." The confirm label stays; the toast after an
  automatic approval is "Activity '<name>' approved automatically".
- `ApprovalChain` new states (chip colours: Not needed grey, Approved automatically green): progress
  "<a> of <b> approved" on "all" levels with each approver's state; "Escalated to <name> on <date>";
  "Reminder sent on <date>"; source text; left-out people with 014's "(can't approve — no approving role)"
  for `cannotApprove`; "No approver" warning with the advice.
- `ApprovalHistory`: new action labels and capacity sentences; per round "Rules applied to: planned cost
  <amount> · <type> · <business units>".
- Decide buttons (`DecisionButton`, header and preview) follow `actions.approve` / `actions.decline` as
  014 defines: shown when available; a non-empty reason is shown as text; an empty reason hides them.
- `DecisionDialog` sentences by `viewer.capacity`: delegate "You approve as delegate of <names>." ·
  backup "You approve as backup approver of level <n>." · onBehalf: 014's on-behalf sentence. 008's "You
  own this activity." note is removed (014 bars owners). On "all" levels, approve shows "<a> of <b>
  already approved." and "Level <n> is complete after this approval." or "Still waiting for <names>
  after this approval."

### Approvals page (US4 – US6; FR-024, FR-030, FR-031)

- `PageHeader` secondary action **Delegate my approvals** → `DelegationPanel`.
- `DelegationBanner` (info tone, above the cards) from `GET /api/approval-delegations?scope=mine`, one
  line per current delegation.
- Tabs: Waiting for me · All open · Decided by me · **Monitoring**. Search and filters apply to the first
  three only; Monitoring hides the toolbar.
- `ApprovalsTable` adds chips in the Activity cell: "Overdue" (red), "Escalated to <name>" (amber), "as
  delegate of <names>" (blue), "No approver" (red); the Level cell shows "<a> of <b> approved" for "all"
  levels.
- `MonitoringView`: four `SummaryCard`s with tips; card **How long approvals have waited** with
  `AgeingBars` — a list (`<ul>`), each item the label, a bar whose width is count / max, and the count as
  text (accessible without colour); card **Waiting by level** (table: Workflow, Level, Waiting, Overdue,
  Oldest waiting); card **Overdue** (`ApprovalsTable` with the overdue rows, row click opens 008's
  preview); `RunEscalationsButton` in the tab's toolbar when `canRunEscalations`: `ConfirmDialog` "Run
  reminders and escalations now?" → POST → toast with the result sentence; when `notEscalated` is not
  empty a second toast lists them.
- `DelegationPanel` (`SidePanel` "Approval delegation"): "Everyone" switch when `canManageOthers`
  (`scope=all`); groups Current, Planned, Ended and cancelled (last 10, then "Show all"); each item
  "<person> → <delegate> · <dateRange> · <state chip>" + note + **Edit** and **Cancel delegation** when
  `canChange`; **Add delegation** opens `DelegationForm` (Delegating person — only when `canManageOthers`,
  default me; Delegate `PersonPicker` (active people); From – To `DateRangeField`; Note `TextAreaField`
  500). Validation shows every API problem (FR-022). Cancel uses `ConfirmDialog`.

### Notifications, people

- `NotificationList` icons: `autoApproved` Zap, `reminder` Clock, `escalated` CircleArrowUp.
- People form (003): **Line manager** `PersonPicker` single (hint "Approves this person's activities when a
  workflow level uses the line manager."); person details lines "Line manager: <name>" / "No line
  manager" and "Line manager of <n> people"; the Deactivate dialog adds the usage line.

### States

| Screen | Loading | Empty | Error | Validation | Conflict |
|---|---|---|---|---|---|
| Monitoring | `Skeleton` cards and bars | "Nothing is waiting for approval." (all zero) · overdue card "Nothing is overdue." | `ErrorState` + Try again | — | — |
| Delegation panel | `Skeleton` | "You haven't delegated your approvals. Add a delegation when you will be away." | `ErrorState` | `FormErrorSummary` + field messages | `conflictMessage(error, 'delegation')` |
| Run button | button busy "Running…" | — | toast with the API message | — | — |

## User-visible messages

Every text appears here once (008's and 014's tables still apply; these are added or replace 008's rows).

| Where | Text |
|---|---|
| Level form | Who approves · Named people · The owner's line manager · People with a role · Role · When there are several approvers · Any one approves · All must approve · Only needed when · Approve automatically when · Planned cost at least (USD) · Planned cost below (USD) · Business unit is one of · Activity type is one of · Backup approver · Takes over when an approval is escalated, or when nobody else can approve. |
| Form hint (replaces 008's) | Levels are checked in order. A level can be needed only sometimes, approved automatically, or need every approver. Nobody approves an activity they own or submitted. |
| Reminders section | Reminders and escalation · Send a reminder · after <n> days waiting · Escalate to the backup approver · after <n> days waiting · Days are calendar days. |
| Workflow details | Reminder after <n> days · escalation after <n> days · No reminders or escalation · Backup approver: <name> · Line manager · Role: <name> · All must approve |
| Validation (API) | data-model §1, §2, §7, §9 messages |
| Submit dialog | Level <n> · <approvers> (line manager of <owner>) / (role <role>) · Not needed — only when <condition> · Approved automatically — <condition> · <name> is left out: owns this activity. / …: is submitting it. / …: is inactive. / …: can't approve — no approving role. · The backup approver <name> approves this level. · (no-approver advice, data-model §3) · This activity will be approved automatically. |
| Resolution problems (API) | data-model §3 table |
| Toasts | Activity '<name>' approved automatically · Delegation to <delegate> saved · Delegation to <delegate> cancelled · <r> reminders sent, <e> escalations (1: "1 reminder sent", "1 escalation"; 0: "no reminders", "no escalations") · Not escalated: <list of "'<activity>' (<ID>), level <n>: no backup approver can take it."> |
| Chain | Not needed — <reason> · Approved automatically — <reason> · <a> of <b> approved · waiting for <names> · Escalated to <name> on <date> · Reminder sent on <date> · No approver · Line manager of <owner> · Role: <name> · <name> was left out: <reason> |
| History actions | Approved automatically · Not needed · Reminder sent · Escalated to <name> · <actor> as delegate of <name> · <actor> as backup approver · <actor> on behalf of <names> · waited <d> days · Rules applied to: planned cost <amount> · <type> · <business units> · No level applies — approved automatically |
| Refusals (API 403) | data-model §6 (014 messages 5 and 6; "You submitted this activity, so someone else must approve it."; "You already approved level <n>. Waiting for <names>.") |
| Decision dialog | You approve as delegate of <names>. · You approve as backup approver of level <n>. · <a> of <b> already approved. · Level <n> is complete after this approval. · Still waiting for <names> after this approval. |
| Approvals page | Delegate my approvals · Monitoring · You are approving for <name> until <date>. · <name> approves for you until <date>. · Overdue · Escalated to <name> · as delegate of <names> · No approver · <a> of <b> approved |
| Monitoring | Overdue · Approvals that have waited at least the workflow's escalation days (or reminder days without escalation). — Escalated · Open approvals whose current level was escalated to a backup approver. — Average time to decision · From submission to final approval or decline, for activities decided this calendar month. Automatic approvals are not counted. — Approved automatically · Activities approved by a rule when they were submitted, this calendar month. · <n> days (one decimal) / — · How long approvals have waited · Under 1 day · 1–2 days · 3–5 days · 6–10 days · More than 10 days · Waiting by level · Workflow · Level · Waiting · Overdue · Oldest waiting · Nothing is waiting for approval. · Nothing is overdue. · Run reminders and escalations · Run reminders and escalations now? · Approvers get a reminder once a level has waited the reminder days, and the backup approver takes over once it has waited the escalation days. Each happens once per level. · Run now · Cancel · Running… |
| Delegation panel | Approval delegation · Everyone · Current · Planned · Ended and cancelled · Show all · Add delegation · Edit delegation · Delegating person · Delegate · From · To · Note (optional) · Save · Cancel · Cancel delegation · Cancel the delegation to <delegate>? · <delegate> can no longer approve for <person> from now on. Decisions already made stay recorded. · Keep delegation · You haven't delegated your approvals. Add a delegation when you will be away. · Planned · Current · Ended · Cancelled |
| Delegation gone (API) | This delegation no longer exists. |
| Notifications (title · text) | Approval needed · <008 text> — you approve as delegate of <name> — Activity approved automatically · '<activity>' (<ID>) was approved automatically: <reason> — Reminder — approval waiting <d> days · '<activity>' (<ID>) has waited <d> days for you at level <n>[ — as delegate of <name>] — Approval escalated to you · '<activity>' (<ID>) waited <d> days at level <n>. You can now approve it as backup approver. — Approval escalated · '<activity>' (<ID>) waited <d> days at level <n> and was escalated to <backup>. |
| People (003) | Line manager · Approves this person's activities when a workflow level uses the line manager. · Line manager: <name> · No line manager · Line manager of 1 person / Line manager of <n> people |

Day counts read "1 day" / "<n> days" everywhere.

## Tests

### Go (package `approvals`, fixed clock, fakes, `authz` grants in the context)

| Rule | Test |
|---|---|
| Condition parts, AND, no planned cost, business unit any-of, type | `rules_test.go` `TestConditionMatches_Table_FR005` |
| Condition and levels texts | `TestConditionText_FR009`, `TestLevelsText_FR009` |
| Evaluate: skipped / autoApproved / decide in order, both conditions on one level | `TestEvaluate_Table_FR003_FR004_SC001` |
| Resolve: named, manager, role, excluded inactive/cannotApprove/owner/submitter (rule on and off), backup used, no approver + advice, the three problems | `resolve_test.go` `TestResolve_Sources_FR001_FR011`, `TestResolve_NoApprover_017P8`, `TestResolve_Problems_FR012` |
| Workflow validation, all at once, normalisation; backup approver and timings; backup and role must approve | `workflows_test.go` `TestCreateWorkflow_RuleProblemsAtOnce_FR008`, `TestWorkflow_BackupAndTimings_FR006_FR007`, `TestUpdateWorkflow_Normalises_FR001_FR002`, `TestWorkflow_BackupAndRoleMustApprove_FR025` |
| Submit: round facts, outcomes, entries by NorthStar, first `decide` level waits, notifications to delegates | `rounds_test.go` `TestSubmit_PartlyAutomatic_FR010_FR014` |
| Submit: everything automatic → Approved, two log entries, owner notified only when someone else submitted | `TestSubmit_ApprovedAtSubmission_FR014_017P4` |
| Submission preview outcomes and problems change nothing | `TestSubmission_OutcomesAndProblems_FR012_FR013` |
| Authority: every step of data-model §6, table-driven (approver, delegate, backup, on behalf, already approved, other; owner and submitter with the rule on and off; delegate who owns; approver without `approvals.decide`) | `authority_test.go` `TestAuthority_Table_FR017_FR018_FR020_SC002` |
| Visibility hook for delegates and escalated backups | `TestNamedIn_DelegatesAndBackup_FR021` |
| "All" levels: counting, delegate counts for the delegator, backup/on-behalf complete, decline ends | `TestDecide_AllMustApprove_FR015` |
| Capacity recorded; on-behalf list only for on-behalf | `TestDecide_RecordsCapacity_FR019` |
| Delegation rules, overlap, states, changes, cancel, others with `workflows.manage` | `delegations_test.go` `TestCreateDelegation_Rules_FR022`, `TestDelegation_States_FR024`, `TestUpdateCancelDelegation_FR024`, `TestDelegations_ForOthersNeedWorkflowsManage_FR022` |
| Delegate sees, decides, is notified; no chains; inactive or non-approving delegate has no effect | `TestDelegation_Effect_FR023` |
| Reminders once, escalation once, escalation replaces a due reminder, unusable backup reported, second run sends nothing | `escalations_test.go` `TestRunEscalations_FR026_FR027_FR028_SC005` |
| Waiting days, overdue, buckets, by level, average decision time, automatic count | `monitoring_test.go` `TestMonitoring_FR029_FR030_SC006` |
| Waiting-for-me membership and row flags; summary equals tab | `inbox_test.go` `TestInbox_DelegateAndBackup_FR031`, `TestInbox_RowFlags_FR031` |
| Log entries and history entries of every new event | `TestApprovalRuleEvents_Logged_FR034_FR035_SC004` |
| Legacy records read with defaults and behave as 008 | `store_json_test.go` `TestLegacyRecords_ReadWithDefaults_017P19` |
| People: manager rules (self, loop, inactive), usage "manager", log | `people/…_test.go` `TestManager_Rules_FR016`, `TestUsage_Manager_FR016` |
| Route rules of the 7 operations | `internal/access` route-rule test (014's "every operation has a rule") passes with the new rows |
| Seed | `seed/approvals_test.go` `TestSeed_ApprovalRules_FR037` (WF-1003 shape, timings, backups, DLG-1001 planned, managers; 008's seeded rounds unchanged) |

### Handler and contract tests (`approvals/handler_rules_test.go`, seeded app)

One test per new operation and one per changed behaviour, every request and response validated against
the contract: `TestRuleOptions_FR001`, `TestDelegations_CRUD_FR022_FR024`, `TestDelegationsAll_NeedsWorkflowsManage_FR024`,
`TestMonitoring_Seeded_TradeMagazineOverdue_FR030`, `TestRunEscalations_SeededEscalatesToAlex_FR027`,
`TestRunEscalations_WithoutPermission403_FR028`, `TestApprove_BySubmitter403_FR018`,
`TestApprove_AlreadyApprovedAllLevel403_FR020`, `TestSubmit_WF1003Automatic_FR014`,
`TestPeople_ManagerField_FR016`. 001's `TestRoutesMatchContract` covers the new routes.

### Component tests (Vitest)

| File | Covers |
|---|---|
| `LevelsEditor.test.tsx`, `RuleConditionFields.test.tsx`, `WorkflowFormPanel.test.tsx` | Source switch shows the right fields; roles list; conditions; reminders section defaults 3/5; every API problem at its field (FR-001 – FR-008, FR-025) |
| `WorkflowDetailsPanel.test.tsx` | Rule texts and timings (FR-009) |
| `SubmitDialog.test.tsx` | Outcomes, left-out notes, backup line, no-approver advice, automatic sentence and toast (FR-011, FR-013, FR-014) |
| `ApprovalChain.test.tsx`, `ApprovalHistory.test.tsx` | New states, progress, escalation, capacities, rules facts (FR-033, FR-034) |
| `ApprovalSection.test.tsx`, `DecisionDialog.test.tsx` | Buttons per `actions` (shown, reason, hidden); capacity sentences; "all" lines (FR-017 – FR-020) |
| `ApprovalsTable.test.tsx`, `ApprovalsPage.test.tsx` | Row chips; Monitoring tab; banners; Delegate my approvals opens the panel (FR-024, FR-030, FR-031) |
| `MonitoringView.test.tsx`, `AgeingBars.test.tsx`, `RunEscalationsButton.test.tsx` | Cards, bars with text values, by-level table, overdue list, run flow and result toasts, hidden without `canRunEscalations` (FR-028, FR-030) |
| `DelegationPanel.test.tsx`, `DelegationForm.test.tsx`, `DelegationBanner.test.tsx` | Groups, create/edit/cancel, delegating-person field only when allowed, errors, banners (FR-022 – FR-024) |
| `NotificationList.test.tsx` | New kinds' icons and texts (FR-032) |
| People form and details tests (003) | Line manager field and lines (FR-016) |

### End-to-end (`frontend/e2e/approval-rules.spec.ts`, real API, fresh seed, workers 1)

1. **US1** — Alex opens Settings › People › Priya Raman: "Line manager: Daniel Okoro" (seeded) and
   WF-1003's details: three levels in words. Priya creates two Lubricants activities with WF-1003 (planning
   helpers): $8,000.00 → the dialog says it will be approved automatically → Approved; $30,000.00 → waits
   for Daniel Okoro (line manager of Priya Raman).
2. **US2** — Mei submits a $50,000.00 Mobility activity with WF-1003: Carlos (line manager) approves level
   1; level 2 shows "Not needed"; at level 3 Daniel approves ("1 of 2 approved"), Alex approves → Approved.
3. **US3** — Daniel creates and submits his own Lubricants activity with WF-1001: the dialog says he is
   left out and Alex Morgan (backup) approves this level; Alex approves; history shows the backup line.
4. **US4** — Daniel delegates to Carlos from today for 3 days; Carlos's banner and Waiting for me show
   Trade magazine ads "as delegate of Daniel Okoro"; Carlos approves; history shows the capacity.
5. **US5 + US6** (own `test.describe` with a fresh seed) — Alex opens Monitoring: Trade magazine ads
   overdue; **Run reminders and escalations** → "no reminders, 1 escalation"; the row shows "Escalated to
   Alex Morgan"; Alex approves as backup approver.
6. **Accessibility** — axe on the workflow form with conditions and errors, Monitoring, the delegation panel
   and form; keyboard only: add a delegation.

## Requirement traceability

| FR | Implemented in | Tested by |
|---|---|---|
| FR-001, FR-002 | `model.go`, `workflows.go`, `resolve.go`, `LevelsEditor` | resolve/workflow tests, LevelsEditor test, e2e 1–2 |
| FR-003 – FR-005 | `rules.go`, `RuleConditionFields` | rules tests, component tests, e2e 1–2 |
| FR-006, FR-007, FR-025 | `workflows.go`, `WorkflowFormPanel` | workflow tests, form test |
| FR-008, FR-009 | `workflows.go`, `kinds.go`, details panel | workflow tests, details test |
| FR-010, FR-011 | `rounds.go` `Submit`, `resolve.go` | rounds/resolve tests, e2e 3 |
| FR-012, FR-036 | `submitCheck` | `TestResolve_Problems_FR012`, `TestSubmission_OutcomesAndProblems_FR012_FR013` |
| FR-013, FR-014 | `Submission`, `Submit`, `SubmitDialog` | rounds tests, dialog test, e2e 1 |
| FR-015 | `decide` | `TestDecide_AllMustApprove_FR015`, e2e 2 |
| FR-016 | `people` package, people form | people tests, e2e 1 |
| FR-017 – FR-020 | `authority.go`, `decide`, `view.go`, dialog/section | authority tests, handler 403 tests, e2e 3–4 |
| FR-021 | `NamedIn` | `TestNamedIn_DelegatesAndBackup_FR021` |
| FR-022 – FR-024 | `delegations.go`, `DelegationPanel`, `DelegationBanner` | delegation tests, e2e 4 |
| FR-026 – FR-028 | `escalations.go`, `RunEscalationsButton`, route rule | escalation tests, handler tests, e2e 5 |
| FR-029 – FR-031 | `monitoring.go`, `inbox.go`, `MonitoringView`, `ApprovalsTable` | monitoring/inbox tests, e2e 5 |
| FR-032 | `notifications.go` recipients, `NotificationList` | notification cases in rounds/escalation tests, list test |
| FR-033, FR-034 | `view.go`, `ApprovalChain`, `ApprovalHistory` | view tests, component tests |
| FR-035 | log entries in every write; delegation kind | `TestApprovalRuleEvents_Logged_FR034_FR035_SC004` |
| FR-037 | seed | `TestSeed_ApprovalRules_FR037` |

## Dependencies and reconciliation

| From | What | Rule for the build |
|---|---|---|
| 014 Access control (**prerequisite**) | `platform/authz` (grants, `PlanFacts`, permissions `approvals.decide`, `approvals.decideOnBehalf`, `workflows.manage`, `Rules.NoSelfApproval` (D-302), `Require`, `Forbidden`); `internal/access` hooks (`CanApprove`, roles with permissions, assignments with scopes); Table C and `routes.go`; the approve/decline check order and messages; `ApprovalPerson.canApprove`; `activities.Approvers.NamedIn` | Use the names 014 delivered (T001). `approvals_access.go` implements `approvals.Access`; if 014 has no `RoleHolders`, add it to `internal/access/hooks.go` (additive: active holders of the role whose scope covers the plan, as `Grants.InScope` decides). Add the six rows above to `routes.go` |
| 008 | Everything in `internal/approvals` and the three web folders | Extend in place; keep 008's and 014's tests green except where 017 changes behaviour (submitter barred, delegates, backup): update them and rename with the 017 FR |
| 011 Activity log | Activity log page, History section, record-type filter | Nothing to build: new verbs and the `approvalDelegation` kind appear automatically |
| 003 People | `Person`, validation, details, usage, people form | 017 adds `managerId` in the `people` package (FR-016) |
| 002 Settings | `ActivityTypes`, business unit attribute and options, `MoneyField`, `ChoiceField`, `CheckboxField` | `ruleChoices` adapter in `approvals_adapter.go` |
| 006 | Activity facts (type id, plan) | Adapter fills the new `ActivityFacts` fields |
| 009 Home | "Waiting for my approval" reads the waiting tab | No change; delegate and backup rows appear (FR-031) |

### Shared files this feature changes

`backend/api/openapi.yaml`, `backend/internal/app/{app.go,approvals_adapter.go,app_test.go}` (+ new
`approvals_access.go`), `backend/internal/access/{routes.go,hooks.go}` (additive), `backend/internal/platform/ids/ids.go`,
`backend/seed/*`, `backend/internal/people/*`, `frontend/src/api/schema.d.ts`, `frontend/src/features/people/*`,
008's three web folders.

## Complexity Tracking

| Choice | Why needed | Simpler alternative rejected because |
|---|---|---|
| `approvals.Access` port for other people's permissions | Resolution, backups and delegates need permissions of people who are not signed in; 014 owns them | Importing `internal/access` from `approvals` would break D-196 and tie the tests to 014's storage |
| Rule outcomes frozen in the round | Audit: what the rule decided and why, even after workflow edits | Re-evaluating on read would change history when a workflow changes |
| Job endpoint instead of a scheduler | Escalations need "time passing"; a worker is a third runtime part (constitution IV) | Checking on every read would write during reads and notify at random moments |

## Consolidation (2026-10-20, binding)

The consolidation of 014–017 settled the points below. They win where this plan, its data model, its tasks or its
fragment say otherwise (as D-200). Build order across the four features:
[docs/reports/brd-gap-closure-plan.md](../../docs/reports/brd-gap-closure-plan.md).

- **Decisions.** 017-P1–017-P22 are D-276–D-297 in [decision-log.md](../../docs/sdd/decision-log.md); the
  consolidation decisions are D-298–D-309.
- **Contract (D-298, D-300).** The fragment is merged into `backend/api/openapi.yaml`; T002 and T003 are done (7
  operations in `notYetServed`). The `x-merge: extend` properties are in the 008 and 003 schemas, but **optional**:
  their "required" entries were not added, so 008's answers keep validating until 017 is built. The API always
  sends them; the web app reads an absent value with the defaults of data-model §10. The enum values were added.
  `ApprovalWorkflowLevel.role` is 014's `RoleRef` (not `SettingRef`). `ApprovalActionAvailability` is now an alias
  of 014's `ActionAvailability` (same shape).
- **First web task (typecheck).** Since the merge, `npm run typecheck` fails in
  `frontend/src/features/approvals/messages.ts` (`historyActionText` lacks `autoApproved`, `skipped`, `reminded`,
  `escalated`; `chainStateText`'s switch lacks `skipped`, `autoApproved`) and in
  `frontend/src/features/notifications/NotificationList.tsx` (`KIND_ICONS` lacks `autoApproved`, `reminder`,
  `escalated`). The first 017 web task adds them with the texts and icons of *User-visible messages* and
  *Notifications, people*.
- **Segregation of duties (D-302).** 014's Rules switch decides (on by default): on — owner and submitter barred
  in every capacity and left out at resolution; off — neither. The field is `Grants.Rules.NoSelfApproval`
  (contract `noSelfApproval`); `OwnersMayNotApprove` is not a name.
- **Hooks (D-290).** 014 exports `ApprovingRoles` and `RoleHolders` from `internal/access/hooks.go`; if 014 was
  merged without `ApprovingRoles`, 017 adds it there (additive).
- **Money (D-303).** "Planned cost (USD)" in conditions and in `ApprovalRuleFacts` is the activity's planned cost
  converted from its plan currency (016) to USD at the current rate at submission, then frozen in the round.
  Until 016 lane C is built every plan is USD and no conversion happens.
- **Access (D-296, D-301).** The rules of the 7 operations are in the Table C additions of the 014 plan's
  Consolidation section. Managing a person's line manager is 003's `PATCH /api/people/{personId}`
  (`people.manage`).
- **Log visibility (D-304).** `approvalDelegation` entries need `activityLog.viewAll`; the people involved see their
  delegations in the delegation panel. Approval events on activities follow the activity's visibility.
- **Demo data (D-306).** Line managers are set at the end of the people step; WF-1003, timings, backups and
  DLG-1001 in the approvals step, after 014's roles (Carlos Mendes holds `approvals.decide` through Global planner).
- **Build order (D-307).** 017 starts when 014's US5 (approvals) is merged; it changes `internal/approvals` and
  `people` after 014. T001's check ("014 is merged") means 014's Phase 2 and US5.
- **Documents.** The updates the plan leaves to the consolidation step (domain model §2.3, §6, §8;
  demo-users-and-data) are done by 017's documentation task with the build; the feature map row is done.
