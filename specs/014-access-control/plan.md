# Implementation Plan: Access Control and Global Budgets

**Branch**: `014-access-control` | **Date**: 2026-10-20 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/014-access-control/spec.md` (BRD BUS.ADM.020, BUS.BUD.040; hooks for
BUS.WFL.010, BUS.PRF.030, BUS.ADM.010, BUS.FIN.080)

**Design artifacts**: [data-model.md](data-model.md) · [contracts/openapi.yaml](contracts/openapi.yaml) (a fragment
merged into `backend/api/openapi.yaml`) · [tasks.md](tasks.md). Conventions are feature 001's and the build guide's
([plan-module-build.md](../../docs/sdd/plan-module-build.md) §7); this plan states only what is new. Decisions are in
*Proposed decisions* (no research.md: no new dependency).

## Summary

- **One policy package, used everywhere.** `backend/internal/platform/authz` holds the permission catalogue,
  the per-request **Grants** of the signed-in person and the check functions every service calls. The new
  business package `internal/access` stores roles, role assignments and access rules, builds the Grants after
  sign-in in one **middleware** (wrapped into the `signedIn` wrap that every package's `Routes` already
  receives), and enforces a **central route table** that declares a rule for every API operation (fail
  closed). Scoped checks ("in this plan's teams") run in the service entry point each handler calls.
- **Every list is filtered by visibility** (plans, Home, All work, approvals, activity log, recycle bin, global
  budgets), and every record answer carries an `access` object of `{available, reason}` per action, so the web
  app never works out a permission itself.
- **Settings › Roles and access** (People, Roles, Rules tabs, Recent access changes) and **My profile › Your
  access**. Approvals gain the named-approver rule and segregation of duties.
- **Global budgets**: new package `internal/budgets` (global budgets and allocations), an allocation sets the
  plan's budget through the plans service (one write path), and a **Global budgets** section for global planners.
- **Demo data**: five roles, six people's roles, the rules, one global budget with two allocations.

## Technical Context

**Language/Version**: Go 1.27 (API); TypeScript 5.9 strict + React 19.3 (web app) — unchanged

**Primary Dependencies**: none new

**Storage**: JSON file store; new tables `roles`, `role_assignments`, `access_rules`, `global_budgets`,
`budget_allocations`; new ID counters `RL`, `RA`, `GB`, `BA`

**Testing**: Go `testing` + `apitest` (contract-validated); Vitest + Testing Library + `mockApi`; Playwright + axe

**Project Type**: web application (React SPA + Go REST API)

**Performance Goals**: Grants are built once per request from four small tables (≤ 1 ms on the demo data, well
under 10 ms at Hive9's scale of ≈ 250 people, 20 roles); list filtering is in memory over plans already read

**Constraints**: 127.0.0.1 only; no personal data in technical logs (the forbidden log line carries the person's
ID only); WCAG 2.2 AA; 390 px without horizontal scroll; the UI follows the design system and the UI foundation
revamp (white, modern, obvious CRUD)

**Scale/Scope**: 22 new API operations; every one of the ≈ 190 existing operations gets a declared rule and a 403
answer in the contract; 6 new web routes; changes in every package of 002–011

## Constitution Check

| Principle | How this plan complies | Result |
|---|---|---|
| I. Spec first | Every element traces to an FR of spec.md, and every FR to a BRD ID (*Requirement traceability*). No `[NEEDS CLARIFICATION]`: open client questions use documented defaults (spec *Assumptions*, *Defaults for the client* below) | Pass |
| II. Shell business, plain language | Shell's words (plan, market, team, global budget, allocation); no Hive9 role names on screen; roles and permissions are **data** edited in Settings, not code; the permission list itself is product behaviour and lives in code like every other action | Pass |
| III. Simple, friendly experience | Hide-or-explain rule (FR-011): nothing disabled without its reason; one View only note instead of many greyed controls; role editing in one side panel with a plain summary | Pass |
| IV. Simple, layered architecture | No new runtime part. Two new business packages (`access`, `budgets`) with handler → service → repository; one platform package (`authz`) with no storage. Every rule on the server; the web app shows the server's `access` decisions | Pass |
| V. Tested business rules | Unit tests for every Grants rule and message; a route-table completeness test; an access-matrix test calling every scoped operation as refused personas; component tests; Playwright journeys for US1–US3 (P1) and US5–US6 | Pass |
| VI. Safe by default | **Conflicts with the current text** ("There are no roles or permissions yet — every signed-in user can do everything") and with *Current Phase & Scope* ("Out of scope: … role-based access control"). The product owner's decision of 2026-10-20 brings roles into scope; this needs constitution **1.1.0** (D-239 (D-A17)). Everything else of VI holds: actor on every call, every change logged, demo sign-in only locally | Pass after amendment |
| VII. Keep it small | No dependency; one catalogue of 25 permissions instead of Hive9's per-level matrix of 20 roles; the route table is one Go map | Pass |

## Proposed decisions

Numbered at consolidation (D-223–D-241 in docs/sdd/decision-log.md; the tags D-A1–D-A19 are kept). "Replaces" names the decision this one supersedes.

| # | Proposed decision | Why | FR |
|---|---|---|---|
| D-223 (D-A1) | **Permissions are fixed in code (Table A, 25 keys); roles are data.** A role is a named set of permissions edited in Settings; a person may hold several roles (union). Replaces D-004's "no roles" part (demo sign-in stays) | BRD: "configurable through access rules rather than code"; the list of possible actions is product behaviour | FR-001–FR-003 |
| D-224 (D-A2) | **Five default roles** — Administrator, Global planner, Planner, Approver, Viewer — with Table B's permissions and default scopes; Administrator is locked to every permission and All markets | Gap analysis §3.1 (≈ six roles from Hive9's twenty); product owner's list | FR-003, FR-018 |
| D-225 (D-A3) | **Scope per role assignment: All markets, My teams (live membership, active or inactive teams), Chosen teams. A market is a team.** A team-less plan is in scope for its creator | Plans already belong to teams (004); BRD "global vs local market" | FR-004, FR-005 |
| D-226 (D-A4) | **Visibility = "View plans" in scope, or plan owner; activities also for their owner and named approvers.** Answers Q-006 ("yes, hide"); replaces D-035's "My teams never stops anyone opening a plan" (the switch stays as a narrowing) and the "not enforced" part of D-172 | Shell's rule (domain model §8) | FR-006–FR-008 |
| D-227 (D-A5) | **New error code `forbidden` (HTTP 403)**, used for every refusal by permission or scope; records outside a person's access answer 403 with a message that names nothing of the record; 404 stays for records that do not exist. `not_allowed` (409) stays for rule refusals (last administrator, allocated budget) | One code the web app recognises; friendly "ask an administrator" message | FR-008, FR-010, FR-021, FR-031 |
| D-228 (D-A6) | **Enforcement in one place per kind of rule, fail closed**: (1) the access middleware builds Grants and applies the central route table (`internal/access/routes.go`) to every request; an operation without a rule answers 500 in tests and 403 in production; (2) scoped rules are checked by the service method the handler calls, never by methods other packages call through adapters; (3) lists are filtered in the service or adapter that reads the plans; (4) recycle-bin and log visibility through one `history.Access` adapter. A context without Grants (seed, internal jobs, service unit tests) is unrestricted; every HTTP route except the public ones passes the middleware, which a test proves | One policy package used by every service; a forgotten operation cannot slip through | FR-009, FR-010 |
| D-229 (D-A7) | **The server decides, the web app shows**: answers carry `access` objects of `{available, reason}` per action (same shape as 008's `ApprovalActionAvailability`); `GET /api/me` carries `access` (roles, effective permissions and scopes, administrators) used only for navigation, page-level gates and the View only note | Constitution IV (server is the source of truth); no duplicated permission logic in TypeScript | FR-011, FR-012, FR-024 |
| D-230 (D-A8) | **Hide-or-explain rule**: when the person can change nothing in the page's area, change actions are hidden and one View only note explains; otherwise unavailable actions stay visible, disabled, with the server's reason | Constitution III ("nothing greyed out without saying why") without a page full of disabled buttons | FR-011 |
| D-231 (D-A9) | **Approvals**: deciding needs `approvals.decide` and being a named approver of the current level (with 017: or its delegate or backup approver), or `approvals.decideOnBehalf` for the plan (017's "administrator", recorded "on behalf of"); segregation of duties: nobody decides on an activity they own or whose open round they submitted, while the access rule `noSelfApproval` is on (default on; 017 follows the switch, D-302); submit and re-submit need `activities.submit`; recall: owner or submitter with `activities.submit`, or `approvals.decideOnBehalf` (017 FR-021). Replaces D-020 and 008 FR-017/FR-021's "anyone" | BRD BUS.WFL.010 (role-based approvals, SoD); one rule set shared with 017 | FR-025–FR-027 |
| D-232 (D-A10) | **Workflow approvers must hold `approvals.decide`** when added; existing approvers who lose it are flagged (`ApprovalPerson.canApprove`) | Prevents approvals no one can make | FR-028 |
| D-233 (D-A11) | **A person's roles are set as a whole** (`PUT /api/people/{personId}/roles`) with the **person's version**, logged on the person as field "Roles" (like memberships, D-057); assignment rows are links, deleted outright | One save = one meaning; the person's History shows access changes | FR-017, FR-037 |
| D-234 (D-A12) | **Last administrator guard** (409 `not_allowed`) on role changes and on deactivating a person; **new people get "Role for new people"** (default Viewer · My teams) in the same save | No lock-out; safe default for new people | FR-021, FR-023 |
| D-235 (D-A13) | **Global budgets are top-down**: an allocation sets the plan budget through `plans.Service.SetBudgetFromAllocation` (the plans service owns plan writes) and the plan budget is then read-only for everyone (409 `not_allowed`); over-allocation warns and never blocks (as D-015); allocations are history children of their plan; a global budget with allocations cannot be deleted. Answers Q-011 with its first option | BRD BUS.BUD.040; one write path for the plan budget keeps every roll-up of 004–010 unchanged | FR-029–FR-035 |
| D-236 (D-A14) | **Global budget rows follow plan visibility**; allocations to plans the viewer cannot see are summed in one "Other markets (n plans)" row | Local owners must not see other markets' detail (BUS.BUD.040) | FR-033 |
| D-237 (D-A15) | **Financial data has its own three permissions**: `finance.view` (the "From finance" sections now; feature 015's Financials cards, columns, filters and pages), `finance.link` (link and unlink transactions, scoped) and `finance.import` (import GSAP extracts, everywhere) — the "one access check" 015 FR-028 expects; budgets and planned costs are planning data | BRD "RBAC for plans, POs, invoices and related financial data" | FR-015 |
| D-238 (D-A16) | **Activity log and recycle bin visibility**: entries about planning records follow the plan's visibility (plan from the entry's path, else from the bin snapshot, else All markets only); entries about Settings, people, teams, workflows, roles and access rules need `activityLog.viewAll`; restoring needs the level's delete permission in the plan (or the managing permission of an administration record); permanent delete and Empty need `recycleBin.purge`. Replaces 011's "every user can restore, delete permanently and empty" | Audit data is itself access-controlled | FR-007, FR-038 |
| D-239 (D-A17) | **Constitution 1.1.0** (MINOR): Principle VI and *Current Phase & Scope* say roles and permissions are in scope (demo sign-in stays; Entra ID later); Technology Choices "Sign-in: Demo sign-in with seeded users and roles". **ADR-0008 "Role-based access with team scopes"** supersedes ADR-0005's "every signed-in user may do everything" | Product owner decision 2026-10-20 | all |
| D-240 (D-A18) | **Settings stay readable by everyone** (forms need the lists); changing them needs the administration permissions; currencies and rates need `exchangeRates.manage` (002 FR-008's "permitted finance roles") | Planners must see what Settings offer | FR-009, FR-011 |
| D-241 (D-A19) | **Refused requests are not in the activity log**; the technical log records `forbidden`, the operation pattern and the person's ID | The log records changes; constitution VI (no names in technical logs) | FR-039 |

## Authorization model and where it is enforced

```
request ─► web.Middleware ─► ServeMux ─► signedIn wrap = auth.RequireSignIn ∘ access.Authorize ─► handler ─► service
                                              │                   │                                       │
                                        actor in ctx      Grants in ctx (authz.With)              authz.RequireIn(ctx, p, planFacts)
                                                          route rule by r.Pattern:                 authz.RequireView(ctx, …)
                                                            Need(p) → Has(p) or 403                 lists: authz.CanView filter
                                                            InPlan(p) → Has(p) or 403 (pre-check)   answers: authz.Available(…)
                                                            View / SignedIn / Custom → pass
```

| Layer | Who | What it checks |
|---|---|---|
| Middleware `access.Authorize` | `internal/access/middleware.go` | Builds `authz.Grants` (data-model §2.1) for the actor, puts it in the context, looks up `routeRules[r.Pattern]`; `Need(p)`: `Has(p)` else 403; `InPlan(p)`: `Has(p)` in any scope else 403 (the service checks the plan's scope); no rule: 403 + error log (a test makes this impossible) |
| Service entry points | each package's service method that a handler calls | `authz.RequireView` before reading a record; `authz.RequireIn(ctx, p, facts)` before changing it, for every record touched (source and destination of moves and copies) |
| List filters | plans, campaigns (destinations), activities (destinations), approvals (inbox), home sources, all-work source, budgets, history adapter | keep only records `authz.CanView…` allows; totals computed after filtering |
| Answers | plans, campaigns, activities, line items, approvals, history, budgets | `access` objects from `authz.Available(ctx, p, facts)` plus record rules (allocation lock, SoD) |
| History hooks | `app/history_access.go` implementing `history.Access` | log entry and bin entry visibility, restore and purge guards (D-A16) |
| Web app | `useAccess()`, `RequirePermission`, `access` objects | navigation, page gates, View only note, disabled actions with reasons — never the only line of defence |

### `platform/authz` (new, imports only `platform/actor` and `platform/errs`)

```go
type Permission string // the 25 keys of Table A: PlansView = "plans.view", … RecycleBinPurge = "recycleBin.purge"
type PermissionInfo struct{ Key Permission; Group, Label, Noun string; Scoped bool }
func Catalogue() []PermissionInfo                 // Table A order; also served by GET /api/access/permissions
func Lookup(key string) (PermissionInfo, bool)

type ScopeType string                             // "all" | "myTeams" | "teams"
type GrantedRole struct{ ID, Name, BuiltIn string; Scope ScopeType; TeamIDs []string; ScopeText string }
type Rules struct{ NoSelfApproval bool }
type Grants struct{ PersonID string; Roles []GrantedRole; Administrators []actor.Actor; Rules Rules /* + unexported perms */ }
func NewGrants(personID string, roles []GrantedRole, myTeamIDs []string, permsByRole map[string][]Permission,
	admins []actor.Actor, rules Rules) Grants
func (g Grants) Has(p Permission) bool             // in any scope
func (g Grants) InScope(p Permission, plan PlanFacts) bool
func (g Grants) CanViewPlan(plan PlanFacts) bool   // data-model §2.2
func (g Grants) Teams(p Permission) (all bool, teamIDs []string)

type PlanFacts struct{ ID string; TeamIDs []string; OwnerID, CreatedByID string }

func With(ctx context.Context, g Grants) context.Context
func From(ctx context.Context) (Grants, bool)      // false: internal call → unrestricted (D-A6)
func Require(ctx context.Context, p Permission) error
func RequireIn(ctx context.Context, p Permission, plan PlanFacts) error
func RequireView(ctx context.Context, plan PlanFacts, record string) error // record: "plan", "campaign", …
func CanView(ctx context.Context, plan PlanFacts) bool
type Availability struct{ Available bool `json:"available"`; Reason string `json:"reason"` }
func Available(ctx context.Context, p Permission, plan PlanFacts) Availability
func Unavailable(reason string) Availability
func Refusal(ctx context.Context, p Permission, plan *PlanFacts) string // the message (User-visible messages 1–3)
```

`errs` gains `CodeForbidden = "forbidden"` and `Forbidden(message string) *Error` (403). Team names for
messages: `NewGrants` receives them through `GrantedRole.ScopeText`; `Refusal` lists the team names of the
permission's scope, sorted.

### `internal/access` (new business package)

| File | Content |
|---|---|
| `model.go` | `Role`, `Assignment`, `AccessRules` (data-model §1.1–§1.3); API types `RoleView`, `RolePage`, `PersonRoles`, `PeopleAccessRow`, `MyAccess` |
| `repository.go`, `store_json.go` | tables `roles`, `role_assignments`, `access_rules` |
| `service.go` | `NewService(db, repo, ids, history, dir Directory, now)`; `GrantsFor(ctx, personID) (authz.Grants, error)`; `Summary(ctx, personID) (authz.MyAccess, error)` (roles, merged permissions, administrators, rule) |
| `roles.go` | `ListRoles`, `GetRole`, `CreateRole`, `UpdateRole`, `DeleteRole` (bin), name rule, Administrator locks |
| `assignments.go` | `ListPeopleAccess`, `PersonRoles`, `SetPersonRoles` (whole set, person version via `Directory.TouchPerson`), `CheckRolesImpact`, last-administrator guard |
| `rules.go` | `GetRules`, `UpdateRules` |
| `changes.go` | `RecentChanges(ctx)`: latest 20 log entries of kinds `role`, `accessRules`, and `person` entries whose changes include field `roles` (reads `history.ListLog` per kind, limit 200, merges by `seq`) |
| `hooks.go` | for other packages through app adapters: `AssignNewPerson(ctx, a, personID)`, `CheckDeactivate(ctx, personID)`, `TeamDeleted(ctx, a, teamID)`, `CanApprove(ctx, personIDs) (map[string]bool, error)`, `HoldersOf(ctx, p authz.Permission, all bool) ([]string, error)`, `RoleHolders(ctx, roleID string, plan authz.PlanFacts) ([]string, error)` (active people holding the role with a scope covering the plan, sorted by name — 017's Role approver source) |
| `middleware.go` | `Authorize(next http.HandlerFunc) http.HandlerFunc` (above) |
| `routes.go` | `type Rule struct{ Kind RuleKind; Permission authz.Permission }`; `var routeRules = map[string]Rule{…}` — exactly Table C; `func RuleFor(pattern string) (Rule, bool)`; `func RulePatterns() []string` |
| `kinds.go` | history kinds `role` (top level) and `accessRules` (top level, never deleted) |
| `handler.go` | the 13 access operations |
| `messages.go` | the texts of *User-visible messages* |

`Directory` (consumer interface, adapter `app/access_adapters.go` over `people`): `Person(ctx, id)`,
`ActivePeople(ctx)`, `TeamIDsOf(ctx, personID)`, `TeamNames(ctx, ids)`, `TouchPerson(ctx, a, personID, version)
(newVersion int, err error)`.

Wiring in `app.New`, right after the people service:

```go
accessService := access.NewService(db, access.NewJSONRepository(), generator, historyService, accessDirectory{people: peopleService}, now)
signedIn := func(next http.HandlerFunc) http.HandlerFunc { return authService.RequireSignIn(accessService.Authorize(next)) }
```

### `internal/budgets` (new business package)

| File | Content |
|---|---|
| `model.go` | `GlobalBudget`, `Allocation` (data-model §1.4–§1.5); API types `GlobalBudgetView`, `GlobalBudgetSummary`, `AllocationRow`, `OtherMarketsRow`, `AllocationOption` |
| `repository.go`, `store_json.go` | tables `global_budgets`, `budget_allocations` |
| `service.go` | `List(year)`, `Get`, `Create`, `Update`, `Delete` (in_use / bin), `AllocationOptions`, `Allocate`, `UpdateAllocation`, `RemoveAllocation`, `AllocationOf(ctx, planID)` (for the plans hook) |
| `figures.go` | totals: allocated = Σ allocations; unallocated = amount − allocated; planned = Σ planned cost of allocated plans; rows' left to plan and budget used (domain model §4.2 formulas) |
| `kinds.go` | `globalBudget` (top level), `budgetAllocation` (parent `plan`, `PrepareRestore` of data-model §1.5) |
| `handler.go` | the 9 budget operations |

Consumer interfaces (adapters in `app/budgets_adapter.go`): `Plans { Facts(ctx, id) (PlanInfo, bool, error);
OfYear(ctx, year) ([]PlanInfo, error); SetBudgetFromAllocation(ctx, a, planID string, amount money.Amount, version
int, comment string) error; PlannedCost(ctx, planIDs) (map[string]money.Amount, error) }` over `plans.Service` and
`campaigns.PlanRollups`; `People { Ref(ctx, id) (people.PersonRef, error) }`; `Managers { CanManageAll(ctx,
personID) (bool, error) }` over `access.HoldersOf`.

### Changes to existing packages (backend)

| Package | Change | FR |
|---|---|---|
| `platform/errs` | `CodeForbidden`, `Forbidden()` | FR-010 |
| `platform/ids` | prefixes `RL`, `RA`, `GB`, `BA` | — |
| `platform/apitest` | `(*Server).GrantRole(t, email, builtIn string, scope string, teamNames ...string)` writing an assignment directly in the store (test helper, D-A6 test strategy) | — |
| `people` | `CurrentUser.Access *authz.MyAccess` filled through hook `UseAccessSummary(func(ctx, personID) (authz.MyAccess, error))` (type `MyAccess` lives in `authz` to avoid an import cycle); `Create` calls hook `NewPerson(ctx, a, personID)` in the same Write (FR-023); status change to inactive calls hook `CanDeactivate(ctx, personID) error` (FR-021); team delete calls hook `TeamDeleted(ctx, a, teamID)`; `kinds.go` `Describe`: field `roles` → "Roles" | FR-020–FR-024 |
| `plans` | `AccessFacts(Plan) authz.PlanFacts`; `List` filters by `CanViewPlan`; `Get`, `CopyOptions`, `ChangeEffects`: `RequireView`; `Create`: `RequireIn(plans.create)` against the chosen teams + team-scope rule FR-013 (field errors on `teamIds`); `Update`: `RequireIn(plans.edit)` + FR-013 + allocation lock (409 `not_allowed` when `budget` differs from an allocated plan's budget); `Delete`: `plans.delete`; `Publish`/`Unpublish`: `plans.publish`; `Copy`: view source + `plans.create` in scope of the copy's teams; `Defaults`: `Has(plans.create)`; answers `Plan.access`, `PlanSummary.access` (`PlanAccess`), `Plan.allocation` via hook `UseAllocations(Allocations)` (default none); new `SetBudgetFromAllocation` (writes budget, `Touch`, logs `changed` with comment) | FR-005–FR-009, FR-013, FR-031 |
| `campaigns` | view check on `ListPlanCampaigns`, `GetCampaign`, `GetInitiative`; `campaigns.edit` on create, update, move (source and destination plan), copy (destination); `campaigns.delete` on delete; destination lists filtered to plans where the person has `campaigns.edit`; answers `PlanCampaigns.access` (`PlanContentAccess`), `Campaign.access`, `Initiative.access` (`ContainerAccess`) | FR-006, FR-009, FR-014 |
| `activities` | view (plan, owner or named approver via hook `activities.Approvers.NamedIn(ctx, activityID)`) on `Get`, `ListForPlan`; `activities.edit` on `NewForm`, `EditForm`, `Create`, `Update`, `Move` (both ends), `Copy` (destination); `activities.delete` on `Delete`; destinations filtered; answers `Activity.access` (`ActivityAccess`, incl. `seeFinance`), `ActivitySummaryPage.access` | FR-006, FR-009, FR-014, FR-015 |
| `lineitems` | view of the activity on `Section`, `Get`; `activities.edit` on `FormOptions`, `Create`, `Update`, `Duplicate`; `activities.delete` on `Delete`; answers `LineItemSection.access`, `LineItemDetails.access` (`LineItemAccess`) | FR-009, FR-015 |
| `approvals` | `GetActivityApproval`, `Preview`: activity view; `Submission`, `Submit`: `activities.submit` in the plan; `Recall`: owner or the open round's submitter with `activities.submit` in the plan, or `InScope(approvals.decideOnBehalf, plan)`; `Approve`/`Decline`: FR-025/FR-026 (order of checks below); `Inbox`: All open filtered by activity view; `Summary` counts after filtering; workflow create/update: approvers must hold `approvals.decide` via hook `UseApprovers(Approvers)` (field error `levels.<n>.approverIds`); `ApprovalPerson.canApprove`; ready-to-submit advice when the owner or the submitter is a level's only approver (SoD on); `actions.approve`/`actions.decline`: available per the check order below; when not available, `reason` is message 6a or 6b for someone who could otherwise decide but owns or submitted the activity, and empty for everyone else (008's rule: an empty reason is not shown, so the buttons are hidden); `actions.submit`/`resubmit`/`recall` carry message 2 or 3 when `activities.submit` is missing | FR-025–FR-028 |
| `home` | every source (`app/home_sources.go`) filters plans and activities by `authz.CanView…` before counting; recent changes through the history access adapter; no-role answer: empty lists (the web app shows the no-role state from `me.access`) | FR-007 |
| `allwork` | `app/allwork_source.go` filters plans by `CanViewPlan` before building rows, totals, calendar, options, record location and export; `exportAllWork` route rule `InPlan(data.export)` | FR-007, FR-016 |
| `history` | `UseAccess(Access)` with `CanSeeEntry(ctx, Entry) bool`, `CanSeeBinEntry(ctx, BinEntry) bool`, `CanRestore(ctx, BinEntry) error`; `ListLog`, the 011 summary and record History filter with it; `ListBin` filters and answers `RecycleBinListItem.access` (`restore`, `deletePermanently`); `Undo`, `Restore`, restore preview call `CanRestore`; default (unset) allows all | FR-007, FR-038 |
| `settings` | none in the service (route rules `Need(settings.manage)` / `Need(exchangeRates.manage)`) | FR-009 |
| `app` | wiring above; adapters `access_adapters.go`, `budgets_adapter.go`, `history_access.go`; `app_test.go` route list includes the new operations | — |
| `seed` | `access.go` (`addAccess` after `addTeams`), `budgets.go` (`addGlobalBudgets` after `addPlans`); no log entries (D-202) | FR-040 |

**Approve/decline check order** (first failure wins): (1) activity not visible → 403 message 4; (2) no open round,
or the level already decided → existing 008 answers; (3) the actor *could decide*: a named approver of the current
level who `Has(approvals.decide)` (decides as approver), else `InScope(approvals.decideOnBehalf, plan)` (decides on
behalf); neither → 403 message 5; (4) `noSelfApproval` on and the actor is the activity's owner (message 6a) or
submitted the open round (message 6b) → 403; (5) decide. The `actions.approve`/`actions.decline` availability runs
the same steps: message 6a/6b when step 4 fails, an empty reason when step 3 fails. Feature 017 extends step 3 with
delegates and backup approvers and keeps steps 1, 2, 4 and 5.

## Table C — every API operation and its rule

Rule kinds: **Public** (no sign-in) · **Signed in** (any signed-in person; answer filtered as noted) · **Need p**
(middleware: holds p in any scope) · **InPlan p** (middleware pre-check `Has(p)`; the service checks p in the
record's plan) · **View** (the service checks that the person can see the record) · **Custom** (described).
"→ filtered" means the answer contains only what the person can see (FR-007).

**001 shell, sign-in, profile, developer pages**

| Operation | Rule |
|---|---|
| GET /api/health · GET /api/auth/demo-people · POST /api/auth/sign-in · POST /api/auth/sign-out | Public |
| GET /api/me | Signed in (answers `access`) |
| PATCH /api/me · PATCH /api/me/preferences · GET /api/me/display-currencies | Signed in (own record) |
| GET, POST /api/sample-folders · PATCH, DELETE /api/sample-folders/{folderId} · POST /api/sample-folders/{folderId}/notes · DELETE /api/sample-notes/{noteId} | Signed in (developer pages, demo mode only) |

**002 Settings**

| Operation | Rule |
|---|---|
| GET /api/settings/overview; GET of models, activity types, shell attributes (+ options), shell attribute options, currencies, finance budgets, finance budget items (lists and items); GET /api/shell-attribute-forms/{level}; POST /api/shell-attribute-forms/{level}/check | Signed in |
| POST /api/models · PATCH, DELETE /api/models/{modelId} · POST …/impact · POST …/status | Need settings.manage |
| POST /api/activity-types · PATCH, DELETE /api/activity-types/{activityTypeId} · POST …/impact · POST …/status | Need settings.manage |
| POST /api/shell-attributes · PATCH, DELETE /api/shell-attributes/{attributeId} · POST …/impact, …/status, …/move · POST …/options · POST …/options/import-check · POST …/options/import | Need settings.manage |
| PATCH, DELETE /api/shell-attribute-options/{optionId} · POST …/status · POST …/move | Need settings.manage |
| PATCH /api/finance-budgets/{budgetId} · POST /api/finance-budget-items · PATCH, DELETE /api/finance-budget-items/{itemId} · POST …/status | Need settings.manage |
| POST /api/currencies · PATCH, DELETE /api/currencies/{code} · POST …/impact · POST …/status | Need exchangeRates.manage |

**003 People and teams**

| Operation | Rule |
|---|---|
| GET /api/people · GET /api/people/{personId} · GET /api/teams · GET /api/teams/{teamId} | Signed in (pickers need everyone) |
| POST /api/people · PATCH /api/people/{personId} · POST …/impact · POST …/status | Need people.manage (+ last-administrator guard on status, FR-021) |
| POST /api/teams · PATCH, DELETE /api/teams/{teamId} · POST …/impact · POST …/status | Need people.manage |

**004 Plans**

| Operation | Rule |
|---|---|
| GET /api/plan-defaults | Need plans.create |
| GET /api/plans | Signed in → filtered (cards and "n of m" count over visible plans) |
| POST /api/plans | InPlan plans.create (scope checked against the chosen teams, FR-013) |
| GET /api/plans/{planId} | View |
| PATCH /api/plans/{planId} | InPlan plans.edit (+ FR-013, FR-031) |
| DELETE /api/plans/{planId} | InPlan plans.delete |
| POST /api/plans/{planId}/publish · …/unpublish | InPlan plans.publish |
| GET /api/plans/{planId}/copy-options | View + Need plans.create |
| POST /api/plans/{planId}/copy | View (source) + InPlan plans.create (for the copy's teams) |
| POST /api/plans/{planId}/change-effects | InPlan plans.edit |
| GET /api/plans/{planId}/campaigns | View (answers `access`) |

**005 Campaigns and initiatives**

| Operation | Rule |
|---|---|
| POST /api/plans/{planId}/campaigns · POST /api/campaigns/{campaignId}/initiatives | InPlan campaigns.edit |
| GET /api/campaigns/{campaignId} · GET /api/initiatives/{initiativeId} | View |
| PATCH /api/campaigns/{campaignId} · PATCH /api/initiatives/{initiativeId} | InPlan campaigns.edit |
| DELETE /api/campaigns/{campaignId} · DELETE /api/initiatives/{initiativeId} | InPlan campaigns.delete |
| GET /api/campaigns/{campaignId}/destinations · GET /api/initiatives/{initiativeId}/destinations | View (source) → filtered to destinations with campaigns.edit |
| POST /api/campaigns/{campaignId}/move · POST /api/initiatives/{initiativeId}/move | InPlan campaigns.edit (source and destination) |
| POST /api/campaigns/{campaignId}/copy · POST /api/initiatives/{initiativeId}/copy | View (source) + InPlan campaigns.edit (destination) |

**006 Activities**

| Operation | Rule |
|---|---|
| GET /api/initiatives/{initiativeId}/activity-form · POST /api/initiatives/{initiativeId}/activities | InPlan activities.edit |
| GET /api/plans/{planId}/activities | View (answers `access`) |
| GET /api/activities/{activityId} | View (plan, owner or named approver) |
| PATCH /api/activities/{activityId} · GET /api/activities/{activityId}/form | InPlan activities.edit |
| DELETE /api/activities/{activityId} | InPlan activities.delete |
| GET /api/activities/{activityId}/destinations | View → filtered to destinations with activities.edit |
| POST /api/activities/{activityId}/move | InPlan activities.edit (source and destination) |
| POST /api/activities/{activityId}/copy | View (source) + InPlan activities.edit (destination) |

**007 Line items**

| Operation | Rule |
|---|---|
| GET /api/activities/{activityId}/line-items · GET /api/line-items/{lineItemId} | View (of the activity) |
| POST /api/activities/{activityId}/line-items · GET …/line-items/form-options · PATCH /api/line-items/{lineItemId} · POST …/duplicate | InPlan activities.edit |
| DELETE /api/line-items/{lineItemId} | InPlan activities.delete |

**008 Approvals and notifications**

| Operation | Rule |
|---|---|
| GET /api/approval-workflows · GET /api/approval-workflows/{workflowId} | Signed in |
| POST /api/approval-workflows · PATCH, DELETE /api/approval-workflows/{workflowId} · POST …/deactivate · POST …/activate | Need workflows.manage (+ FR-028 on create/update) |
| GET /api/activities/{activityId}/approval · GET /api/approvals/activities/{activityId} | View |
| GET /api/activities/{activityId}/approval/submission · POST …/approval/submit | InPlan activities.submit |
| POST /api/activities/{activityId}/approval/recall | Custom (FR-027: owner or submitter with activities.submit, or approvals.decideOnBehalf) |
| POST /api/activities/{activityId}/approval/approve · …/decline | Custom (check order above) |
| GET /api/approvals · GET /api/approvals/summary | Signed in → filtered (Waiting for me: named approver; All open: visible activities; Decided by me: own decisions) |
| GET /api/notifications · POST /api/notifications/{notificationId}/read · POST /api/notifications/read-all | Signed in (own) |

**009 Home**

| Operation | Rule |
|---|---|
| GET /api/home/summary · /plans · /attention · /waiting · /my-activities · /recent-changes | Signed in → filtered |

**010 All work**

| Operation | Rule |
|---|---|
| GET /api/all-work · GET /api/all-work/options · GET /api/all-work/records/{recordId} | Signed in → filtered (a record the person cannot see answers 403 no access) |
| GET /api/all-work/export | InPlan data.export → filtered |
| GET, PUT /api/all-work/preferences · GET, POST /api/saved-views · PATCH, DELETE /api/saved-views/{viewId} | Signed in (own) |

**011 Activity log and recycle bin**

| Operation | Rule |
|---|---|
| GET /api/activity-log · GET /api/activity-log/summary | Signed in → filtered (D-A16) |
| GET /api/recycle-bin | Signed in → filtered, rows answer `access` |
| POST /api/recycle-bin/{entryId}/undo · POST …/restore · GET …/restore-preview | Custom (`history.Access.CanRestore`, D-A16) |
| DELETE /api/recycle-bin/{entryId} · POST /api/recycle-bin/empty | Need recycleBin.purge |

**012 Bulk actions and import (deferred; rules declared now so the route table is complete when it is built)**

| Operation | Rule |
|---|---|
| GET /api/bulk-actions/fields | Signed in |
| POST /api/bulk-actions/targets | Signed in → filtered |
| POST /api/bulk-actions/preview · POST /api/bulk-actions | Custom: each target checked like its single operation; refused targets are outcomes "You can't change this record." and are not changed |
| POST /api/bulk-actions/{bulkActionId}/undo | Custom: each record through `CanRestore` |
| GET /api/plans/{planId}/imports · GET /api/imports/{importId} | View |
| POST /api/plans/{planId}/imports/check · POST /api/plans/{planId}/imports · GET /api/plans/{planId}/import-template | InPlan activities.edit (+ campaigns.edit when the file creates campaigns or initiatives) |
| POST /api/imports/{importId}/undo | InPlan activities.delete |

**014 Access control and global budgets (new)**

| Operation | Rule |
|---|---|
| GET /api/access/permissions · GET /api/access/people · GET /api/access/changes | Need access.manage |
| GET, POST /api/roles · GET, PATCH, DELETE /api/roles/{roleId} | Need access.manage |
| GET, PUT /api/people/{personId}/roles · POST /api/people/{personId}/roles/impact | Need access.manage |
| GET, PATCH /api/access-rules | Need access.manage |
| GET /api/global-budgets · GET /api/global-budgets/{globalBudgetId} | Need globalBudgets.view → rows filtered (FR-033) |
| POST /api/global-budgets · PATCH, DELETE /api/global-budgets/{globalBudgetId} | Need globalBudgets.manage; the service also requires it for All markets (a global budget spans markets, FR-029) |
| GET /api/global-budgets/{globalBudgetId}/allocation-options | Need globalBudgets.manage → plans of the year in scope without an allocation |
| POST /api/global-budgets/{globalBudgetId}/allocations · PATCH, DELETE /api/budget-allocations/{allocationId} | InPlan globalBudgets.manage (the allocation's plan) |

## API operations (new)

| Operation | Purpose | Answer |
|---|---|---|
| `GET /api/access/permissions` (listPermissions) | Table A for the role editor | `PermissionCatalogue` |
| `GET /api/access/people` (listPeopleAccess) | People tab: everyone with roles and scopes | `PeopleAccessPage` |
| `GET /api/access/changes` (listAccessChanges) | Recent access changes (latest 20) | `ActivityLogPage` |
| `GET /api/roles` (listRoles) · `POST` (createRole, body `RoleCreate`; "Copy of …" is the web form pre-filled from another role) | Roles tab | `RolePage` · 201 `Role` |
| `GET /api/roles/{roleId}` · `PATCH` (`RoleUpdate`, changed fields + `version`) · `DELETE` | Role page | `Role` · `Role` · `DeleteResult` |
| `GET /api/people/{personId}/roles` · `PUT` (`PersonRolesUpdate`: complete list + person `version`) | Person roles panel | `PersonRoles` |
| `POST /api/people/{personId}/roles/impact` (`PersonRolesUpdate`) | Warnings before saving (FR-022) | `ChangeImpact` |
| `GET /api/access-rules` · `PATCH` (`AccessRulesUpdate`) | Rules tab | `AccessRules` |
| `GET /api/global-budgets?year=` · `POST` (`GlobalBudgetCreate`) | Global budgets list | `GlobalBudgetPage` · 201 `GlobalBudget` |
| `GET /api/global-budgets/{globalBudgetId}` · `PATCH` (`GlobalBudgetUpdate`) · `DELETE` | Global budget page | `GlobalBudget` · `GlobalBudget` · `DeleteResult` |
| `GET /api/global-budgets/{globalBudgetId}/allocation-options` | Allocate dialog | `AllocationOptionPage` |
| `POST /api/global-budgets/{globalBudgetId}/allocations` (`BudgetAllocationCreate`) | Allocate | 201 `GlobalBudget` |
| `PATCH /api/budget-allocations/{allocationId}` (`BudgetAllocationUpdate`: `amount`, `version`, `planVersion`) · `DELETE` | Change or remove | `GlobalBudget` |

`planVersion` on allocation writes is the plan's version the global planner saw; a stale one answers 409
`version_conflict` naming who changed the plan (the plan's budget would otherwise overwrite a newer plan). The
contract fragment has every schema and field.

## Frontend design

### Session access and shared parts (owner 014)

| Part | Where | What |
|---|---|---|
| `useAccess()` | `src/lib/access.ts` | from `useSession().user.access`: `has(p)`, `scopeOf(p)` (`{all, teams}`), `roles`, `administrators`, `adminNames()` ("Alex Morgan" / "Alex Morgan or Nina Berg"), `rulesNoSelfApproval` |
| `permissionLabel`, `summariseAccess(permissions)` | `src/lib/access.ts` | the plain-language summary of FR-017/FR-024, the only implementation (display text, not a rule). Input: merged permissions `{key, all, teams}` (`MyAccess.permissions`, `PersonRoles.permissions`). Rule: group the scoped permissions by where they apply ("all markets", or the team names joined with ", " and " and "), one line per group in Table A order: "In <where>: <label>, <label> and <label>." with each label's first letter lower-cased; then one line "Everywhere: …" for the permissions that are not scoped; no permission → "No access yet." Example (Sara after US3 scenario 2): "In Lubricants Europe and Mobility China: view plans and everything in them, see financial data and export from All work." / "In Mobility China: create and copy plans, edit plan details, publish and unpublish plans, …". The roles panel previews it before saving by merging the ticked roles' permissions (from `GET /api/roles`) with the chosen scopes (the person's teams for My teams); after saving, the server's `permissions` replace the preview |
| `ViewOnlyNote` | `src/components/ViewOnlyNote/` | the note of FR-011 (info tone, lock icon, one sentence + admin names); props `area: 'plan' \| 'settings' \| 'globalBudget' \| 'record'` |
| `NoAccess` | `src/components/NoAccess/` | content of the no-access page (FR-008): title "No access", the server's sentence, **Go to Home** |
| `RequirePermission` | `src/app/RequirePermission.tsx` | route guard: renders `NoAccess` unless `has(p)` |
| `availabilityProps(a)` | `src/lib/access.ts` | maps `{available, reason}` to `{disabled, hint}` for `ActionsMenu` items and `Button` (`aria-describedby` to the reason) |
| 403 handling | `src/api/errors.ts`, `src/app/SessionProvider.tsx` | `forbidden` added to the error codes; the QueryClient's mutation and query error handlers: on 403 → `session.refetch()`, invalidate the current page's queries, show the server message (toast for actions, `NoAccess` for page loads) |
| Navigation | `src/app/navigation.ts`, `Sidebar.tsx`, `NavDrawer.tsx` | `NavItem.permission?: Permission`; new `GLOBAL_BUDGETS` item (icon `Landmark`, after Plans, permission `globalBudgets.view`); items without permission are not rendered |
| Settings cards | `src/features/settings/sections.ts` | `SettingsSection.permission?`; adds `ACCESS_SETTINGS_SECTIONS` from `features/access/settingsSections.ts` ("Roles and access" — "Who can see and change what, by role and market.", icon `ShieldCheck`, count "<n> roles") |

### Routes (new)

| Route | Page | Guard |
|---|---|---|
| `/settings/access` (`?tab=people\|roles\|rules`, `?person=U-…` opens the roles panel) | `AccessPage` | `access.manage` |
| `/settings/access/roles/new` (`?copyOf=RL-…`), `/settings/access/roles/:roleId`, `/settings/access/roles/:roleId/edit` | `RolePage`, `RoleForm` | `access.manage` |
| `/budgets`, `/budgets/:globalBudgetId` | `GlobalBudgetsPage`, `GlobalBudgetPage` | `globalBudgets.view` |

### Pages

- **AccessPage** (`features/access/`): `PageHeader` "Roles and access"; `PillTabs` People · Roles · Rules; a
  **Recent access changes** card (latest 20, `RecentAccessChanges.tsx`, each line "Alex Morgan changed roles of
  Sara Novak · 10:42" linking to `/activity-log?record=<id>`).
  - *People tab*: `SearchField` (name or e-mail), `SegmentedFilter`-style role filter (All, each role, No role);
    table: person (`PersonLabel`), teams (`TeamPills`), roles as chips "Planner · My teams"; flag "No teams left in
    this scope"; row action **Edit roles** → `PersonRolesPanel` (`FormPanel`): one `CheckboxField` per role (with
    description), and for each ticked role a scope radio group (All markets / My teams / Chosen teams +
    `TeamPicker`); live summary (`summariseAccess`); **Save** calls `…/roles/impact` first when the change removes
    `approvals.decide` or the editor's own Administrator role, showing warnings with **Keep editing** / **Save
    changes**; then `PUT`.
  - *Roles tab*: list of roles (name, description, "n people", tag "Built in"); **New role** (menu: Empty, Copy of
    …). `RolePage`: details with permissions grouped by Table A group (ticks), **Edit**, **Delete** (menu, with
    `in_use` reason from the server), Administrator shows `LockedNote` and no actions. `RoleForm`: name,
    description, default scope, permission checkboxes by group, unsaved-changes guard.
  - *Rules tab*: a card with the toggle "Nobody approves an activity they own or submitted" and the "Role for new people"
    `ChoiceField` (roles except Administrator, plus "No role"); **Edit** opens a `FormPanel`; saved with version.
- **My profile** (`features/people/ProfilePage.tsx`): adds `YourAccessCard` from `features/access/` — roles with
  scope chips, the summary, "To change your access, ask Alex Morgan."
- **GlobalBudgetsPage** (`features/global-budgets/`): `PageHeader` "Global budgets" with **New global budget**
  (shown with `globalBudgets.manage`); year filter (default current year); one card per global budget: name,
  owner, four `SummaryCard` figures (Global budget, Allocated, Unallocated in red when negative with
  "Over-allocated by X", Planned cost), link to the page; `EmptyState` "No global budgets for 2026 yet."
- **GlobalBudgetPage**: header with name, year, owner, **Edit** and **Delete** (from the answer's `access`);
  four summary cards; table of allocated plans (plan link, `TeamPills`, owner, `StatusChip`, Allocated, Planned
  cost, Left to plan, `ProgressBar` budget used, row menu **Change allocation**, **Remove allocation**); "Other
  markets (n plans)" row when present; section "Plans of 2026 without an allocation" with **Allocate**;
  `AllocationDialog` (`FormDialog`: plan `ChoiceField` from allocation-options, `MoneyField` USD).
  `GlobalBudgetForm` in a `FormPanel` (name, year — locked after save, amount, owner `PersonPicker` limited to
  people who can manage all markets, description).
- **Plan page and form** (004): the budget summary card shows `PlanAllocationNote` ("Allocated from 'FY2026
  Global marketing budget' — Carlos Mendes", link when `allocation.canOpen`); `PlanForm` budget field read-only
  with `LockedNote` when `allocation` is set; Teams `TeamPicker` options limited to `useAccess().scopeOf
  ('plans.create' | 'plans.edit')` teams unless All, keeping the plan's other teams as fixed pills.

### Changes to existing web features

| Feature | Change | FR |
|---|---|---|
| 002 Settings pages | When `!has('settings.manage')` (currencies: `exchangeRates.manage`): `ViewOnlyNote area="settings"`, hide New, Edit, Delete, status, move and import actions | FR-011 |
| 003 People, Teams | Same with `people.manage`; person panel shows a "Roles" line and **Change roles** link (`/settings/access?person=…`) for `access.manage`; the last-administrator refusal is shown as returned | FR-011, FR-021 |
| 004 Plans | List: **New plan** from `has('plans.create')`, disabled with the no-team reason when the scope has no team; `PlanCard` menu from `PlanSummary.access`; plan header actions and tree toolbar from `Plan.access` / `PlanCampaigns.access`; `ViewOnlyNote` when nothing is available | FR-011, FR-013 |
| 005 Campaigns and initiatives | Row menus and **Add** buttons from `PlanCampaigns.access`; record pages from `Campaign.access` / `Initiative.access`; destination dialogs show the server's filtered lists | FR-011, FR-014 |
| 006 Activities | Activity rows from `ActivitySummaryPage.access`; activity page actions from `Activity.access`; "From finance (coming later)" section rendered only when `access.seeFinance.available` | FR-011, FR-015 |
| 007 Line items | Section **Add line item**, row menus from `LineItemSection.access`; page actions from `LineItemDetails.access`; `FromFinanceSection` only with `seeFinance` | FR-011, FR-015 |
| 008 Approvals | Approval section already follows `actions.*`; reasons now come from the server; chain marks `canApprove: false` with "(can't approve — no approving role)"; submit panel shows the SoD advice line; workflow editor shows the `approverIds` field error | FR-025–FR-028 |
| 009 Home | No-role state (FR edge case) when `me.access.roles` is empty; otherwise unchanged | FR-007 |
| 010 All work | **Export** disabled with its reason without `data.export` | FR-016 |
| 011 Activity log, Recycle bin | Bin rows' **Restore** and **Delete permanently** from `RecycleBinListItem.access`; **Empty recycle bin** only with `recycleBin.purge`; record-type filter gains Role, Access rules, Global budget, Budget allocation | FR-007, FR-038 |

## User-visible messages

`<Noun>` is the refusal noun of Table A; `<roles>` is the role names joined with ", " ("your role (Viewer)" for
one, "your roles (Planner, Approver)" for several); `<teams>` the team names joined with ", ".

| # | Where | Text |
|---|---|---|
| 1 | 403, person holds no role | "<Noun> needs a role, and you don't have one yet. Ask an administrator." |
| 2 | 403, no role grants it | "<Noun> isn't part of your role (<roles>). Ask an administrator if you need it." |
| 3 | 403, granted only for other teams | "<Noun> is allowed for you only in plans of <teams>. Ask an administrator if you need it here." |
| 4 | 403, record outside access (any record type) | "You don't have access to this <record type>. Ask an administrator if you need it." |
| 5 | 403 approve/decline, not an approver | "Only the approvers of level <n> (<names>) can approve or decline this activity." |
| 6a | 403 approve/decline, SoD (owner) | "You own this activity, so someone else must approve it." |
| 6b | 403 approve/decline, SoD (submitter) | "You submitted this activity, so someone else must approve it." |
| 7 | Submit advice (SoD; the owner or the person submitting is a level's only approver) | "You are the only approver of level <n>. An administrator will have to approve on your behalf." |
| 8 | Workflow field `levels.<n>.approverIds` | "<Name> can't approve activities. Give them a role that approves first." |
| 9 | Approval chain flag | "(can't approve — no approving role)" |
| 10 | 409 not_allowed, last administrator | "<Name> is the only administrator. Give someone else the Administrator role first." |
| 11 | Lock on Administrator role | "The Administrator role always has every permission and can't be changed." |
| 12 | Field `roles.<i>.scope` | "The Administrator role always applies to all markets." |
| 13 | 409 in_use, role delete | "'<Role>' is given to <n> person." / "… to <n> people." + " Take it away from them first." |
| 14 | Field `name` (role) | "A role called '<name>' already exists." |
| 15 | Field `permissions` | "Choose at least one permission." |
| 16 | Field `roles.<i>.teamIds` | "Choose at least one team." |
| 17 | Field `newPersonRoleId` | "New people can't be made administrators automatically." |
| 18 | Impact `approverCannotApprove` | "<Name> is an approver in <n> approval workflow[s] and has <m> activit[y/ies] waiting. Without a role that approves, an administrator will have to decide on their behalf." (singular or plural by count; "has no activities waiting" when m = 0) |
| 19 | Impact `removesOwnAdministration` | "You will no longer be able to change Settings, people or access." |
| 20 | Plan field `teamIds`, out of scope | "You can only choose your own teams: <teams>." |
| 21 | Plan field `teamIds`, none left | "Keep at least one of your teams (<teams>) on the plan, or you won't be able to see it." |
| 22 | New plan hint, no team | "Plans belong to teams, and you are not in a team yet. Ask an administrator to add you to one." |
| 23 | Plan budget lock and 409 not_allowed | "Allocated from '<global budget>' — ask <owner> to change it." |
| 24 | Global budget field `name` | "A global budget called '<name>' already exists." |
| 25 | Global budget field `year` (lock) | "The year can't change once the global budget is saved." |
| 26 | Field `amount` | "Enter an amount of 0 or more." |
| 27 | Field `ownerId` | "Choose a person who can manage global budgets for all markets." |
| 28 | Allocation field `planId`, year | "Choose a plan of <year>." |
| 29 | Allocation field `planId`, taken | "'<plan>' already has an allocation from '<global budget>'." |
| 30 | 409 in_use, global budget delete | "'<name>' has an allocation to 1 plan. Remove it first." / "'<name>' has allocations to <n> plans. Remove them first." |
| 31 | Restore note | "The allocation was not restored because its global budget was deleted." |
| 32 | Unallocated negative | "Over-allocated by <amount>" |
| 33 | View only, plan area | "View only — <your role (Viewer) lets / your roles (A, B) let> you see this plan but not change it. Ask an administrator (<names>) if you need to change it." |
| 34 | View only, Settings | "View only — only administrators change Settings. Ask <names> if something needs to change." |
| 35 | View only, global budget | "View only — you can see this global budget but not change it." |
| 36 | Home, no role | "You don't have access to any plans yet. Ask an administrator (<names>) to give you a role." |
| 37 | People tab flag | "No teams left in this scope" |
| 38 | Toasts | "Roles of <name> saved" · "Role '<name>' created" · "Role '<name>' saved" · "Access rules saved" · "Global budget '<name>' saved" · "Allocation to '<plan>' saved" · "Allocation to '<plan>' removed" |

## Demo data (`backend/seed/access.go`, `backend/seed/budgets.go`)

- After `addTeams`: roles RL-1001 Administrator, RL-1002 Global planner, RL-1003 Planner, RL-1004 Approver,
  RL-1005 Viewer (Table B, descriptions: "Changes everything, including Settings and access." · "Plans for every
  market and shares global budgets." · "Plans for their own markets." · "Approves activities they are named on."
  · "Sees plans of their markets without changing them."); assignments: Alex → Administrator (all); Carlos →
  Global planner (all); Daniel → Planner (myTeams), Approver (myTeams); Priya → Planner (myTeams); Mei → Planner
  (myTeams); Sara → Viewer (myTeams); access rules: owners may not approve = true, new person role = Viewer.
- After `addPlans`: GB-1001 "FY2026 Global marketing budget", 2026, 2,500,000.00 USD, owner Carlos Mendes,
  description "Shared out to market plans by Global Marketing Ops."; BA-1001 → FY2026 Lubricants Europe
  1,200,000.00; BA-1002 → FY2026 Mobility China 800,000.00 (equal to their seeded plan budgets, so plans are not
  changed).
- No activity-log entries (D-202). `docs/product/demo-users-and-data.md` gains a "Roles" column (consolidation).

## Tests

### Go

| Package · file | What (test names carry the FR, e.g. `TestGrants_UnionOfRoles_FR002`) |
|---|---|
| `platform/authz` `grants_test.go` | union of roles; scope merge (all wins; myTeams = live membership; teams list); `InScope` (team overlap, team-less plan for its creator only, no overlap); `CanViewPlan` (owner); Administrator = every permission incl. a fake new key; `Refusal` messages 1–4 exactly |
| `access` `roles_test.go`, `assignments_test.go`, `rules_test.go` | role CRUD rules and messages 11–15; Administrator locks; delete in_use and bin + Undo; set roles as a whole with person version and `version_conflict`; log entries of data-model §3 exactly; last administrator (assignment and deactivation hook); new person default role; team deleted → removed from scopes; impact warnings 18–19; recent changes merge |
| `access` `middleware_test.go` | grants in context; `Need` 403 with message 2; `InPlan` pre-check; missing rule → refused; forbidden technical log line has the ID and no name |
| `access` `handler_test.go` | every access operation, contract-validated (`apitest`) |
| `budgets` `service_test.go`, `handler_test.go` | FR-029–FR-035: validation messages 24–30; allocation sets plan budget and logs on both; plan version conflict; remove keeps budget; over-allocation figures; rows filtered + Other markets row; delete in_use / bin; plan delete takes allocation, restore brings it back or drops it with note 31 |
| `app` `access_routes_test.go` | every pattern in `App.Routes` has a rule in `access.RulePatterns()` and every rule pattern is served (SC-001) |
| `app` `access_matrix_test.go` | one row per non-public operation of Table C with a request built from the seed: as **Mei** on Lubricants records → 403 (View and InPlan rows) and as **Sara** → 403 for every change operation (InPlan, Need rows); as **Priya** for Need rows of administration → 403; each answer validated against the contract (SC-001) |
| `plans`, `campaigns`, `activities`, `lineitems` | `access` objects in answers per persona; FR-013 team rules (messages 20–21); allocation lock (message 23); destinations filtered |
| `approvals` | check order (US5 scenarios 1–5, messages 5, 6a, 6b); SoD rule off; on-behalf by administrator; recall by owner, submitter, administrator and refused for others; workflow approver rule (message 8); `canApprove`; submit advice 7; inbox filtered |
| `home`, `allwork` (via `app` adapters), `history` | US1 scenarios 1, 4, 5: counts and totals for Priya equal the Lubricants plan only; export contains only visible plans; log and bin filtered; restore guard |

**Existing tests (D-A6 strategy).** Run the whole suite after the middleware lands. A failing existing test that
is not about access gets the role it needs with `server.GrantRole(t, email, "administrator", "all")` (or a narrower
role) at its start; its assertions do not change. Tests that sign in as a person to see another market's plan get
`GrantRole(t, email, "globalPlanner", "all")`. Never weaken a 403 assertion of this feature to make an older test
pass.

### Web (Vitest + Testing Library, `mockApi`)

`useAccess`, `summariseAccess` (sentences per group and scope), `availabilityProps`; `ViewOnlyNote`, `NoAccess`,
`RequirePermission`; Sidebar and Settings cards hidden by permission; `AccessPage` tabs, `PersonRolesPanel`
(scope choice, impact warnings, save), `RolePage`/`RoleForm` (locks, grouped permissions), Rules tab;
`YourAccessCard`; `GlobalBudgetsPage`, `GlobalBudgetPage` (figures, red over-allocation, Other markets row,
allocate, change, remove); plan form budget lock and team limits; every changed feature's page renders hidden
or disabled actions from `access` objects with their reasons; 403 handling refetches the session.

### End-to-end (Playwright + axe; new file `frontend/e2e/access-control.spec.ts`)

1. US1: Priya, Mei, Sara, Carlos see the expected plans; Mei opens a Lubricants address → no-access page.
2. US2: Sara on a plan and an activity → View only note, no change actions; Priya's Delete plan disabled with its
   reason; Priya's team choice limited.
3. US3: Alex gives Sara Planner for Mobility China; Sara (other browser context, already signed in) edits a
   Mobility activity; last-administrator refusal.
4. US5: Daniel cannot approve his own activity; Alex approves on behalf.
5. US6: Carlos changes an allocation; Priya sees the read-only budget; Mei has no Global budgets.
Each page passes axe (SC-007).

**Existing journeys** (`core-flow.spec.ts`, `module-rest.spec.ts`, `smoke.spec.ts`): sign in as the persona whose
seeded roles allow the step — Settings, people, teams and workflows as Alex; Lubricants planning as Priya;
Mobility planning as Mei; Lubricants approvals as Daniel; Mobility approvals as Carlos; steps that move or copy
across markets as Carlos. Expected texts that name the actor change with it.

## Sibling features 015–017 (written at the same time)

| Feature | What it takes from 014 | What 014 takes from it |
|---|---|---|
| 015 Financial transactions | `finance.view` (cards, columns, filters, pages; "You don't have access to financial data — ask an administrator" in place of each), `finance.link` (scoped to the line item's plan), `finance.import` (Need); its lists filtered by plan visibility (FR-007). Its operations get rows in `access/routes.go` when it is built | Nothing; the 3 permissions are in Table A from the start |
| 016 Planning and currency extensions | Field help is `settings.manage`; rate entry and GSAP rate-file import are `exchangeRates.manage`; its multi-plan pages (016 FR-014) are filtered by FR-007; its operations get rows in `routes.go` | Nothing |
| 017 Approval rules | Who may decide (FR-025 steps 1–5, extended by delegates and backup approvers at step 3), `noSelfApproval` (FR-026), recall (FR-027), `approvals.decideOnBehalf` as "administrator", `workflows.manage` for workflow changes (017 FR-025), `access.RoleHolders` for Role approvers, Rules-tab switch | Its owner+submitter SoD scope (adopted here); 017 follows the switch (D-302) |

Every route a sibling adds needs its row in `routeRules`; `app/access_routes_test.go` fails until it has one.

## Requirement traceability

| FR | BRD | Backend | Web | Tests |
|---|---|---|---|---|
| FR-001–FR-003 | ADM.020 | `authz` catalogue, `access` roles, seed | Roles tab | authz, access, matrix |
| FR-004–FR-005 | ADM.020, BUD.040 | `authz.Grants`, `InScope` | People tab scopes | authz |
| FR-006–FR-008 | ADM.020, BUD.040, PRF.030 | services' view checks, list filters, history adapter | `NoAccess`, 403 handling | matrix, home/allwork/history, e2e 1 |
| FR-009–FR-010 | ADM.020 | route table, middleware, `RequireIn` in services | — | routes test, matrix |
| FR-011–FR-012 | ADM.020 | `access` objects in answers | `ViewOnlyNote`, `availabilityProps`, nav and cards | web tests, e2e 2 |
| FR-013–FR-014 | ADM.020 | plans team rules; destinations | plan form; destination dialogs | plans, campaigns, activities, e2e 2 |
| FR-015–FR-016 | ADM.020, FIN.080 | `seeFinance`; export rule | finance sections; Export | activities, lineitems, allwork |
| FR-017–FR-020 | ADM.020, ADM.010 | `access` service and handler | AccessPage | access, web, e2e 3 |
| FR-021–FR-024 | ADM.020 | guards, impact, people hooks, `MyAccess` | panel warnings, Your access | access, people, web |
| FR-025–FR-028 | WFL.010, ADM.020 | approvals | approval section, chain, workflow editor | approvals, e2e 4 |
| FR-029–FR-036 | BUD.040 | `budgets`, plans hook and `SetBudgetFromAllocation` | global budgets pages, plan budget lock | budgets, plans, e2e 5 |
| FR-037–FR-039 | ADM.020, WFL.010, ADM.010 | log entries, recent changes, technical log | Recent access changes | access, budgets, middleware |
| FR-040 | ADM.020, BUD.040 | seed | — | seed test |
| FR-041 | ADM.020 | all of the above | all of the above | all |

## Defaults for the client to confirm

1. Five roles and their permissions (Table B); Shell may change them in Settings at any time.
2. A market is a team; scopes are All markets, My teams or Chosen teams (no scoping by Business unit or
   Country/OU values).
3. Plans are hidden from people outside their scope (answers Q-006); owners and named approvers still see their
   own records.
4. Planners cannot delete plans (Global planners and Administrators can).
5. Settings are readable by everyone and changed only by Administrators; exchange rates are a separate permission.
6. Every default role may see financial data.
7. Segregation of duties = nobody approves an activity they own or submitted (on by default; an administrator can
   turn it off in the Rules tab).
8. New people get the Viewer role for their teams.
9. Global budgets are shared top-down; the allocation becomes the local plan's budget and is read-only for the
   local owner; over-allocation warns and never blocks; global budgets in USD, each allocation in its plan's
   currency (D-303).
10. Local owners never see other markets' allocations; global budget pages show them as "Other markets".
11. Refused attempts are recorded only in the technical log, not shown to administrators.
12. Role assignments are made in MAPP; with Entra ID later they may come from Entra app roles or groups.

## Complexity Tracking

| Item | Why needed | Simpler alternative rejected because |
|---|---|---|
| Constitution amendment (VI, Current Phase) | Roles are now in scope by product-owner decision | Building roles under a constitution that forbids them would break Principle I's governance |
| Central route table plus service checks | Fail-closed guarantee for ≈ 210 operations; scoped checks need the record | Only service checks: an operation can be forgotten; only middleware: cannot know a record's plan |
| `access` objects on many answers | Server decides every per-record action (constitution IV) | Computing permissions in TypeScript would duplicate the scope rules and drift |
| `budgets` package (new business area) | BUS.BUD.040 allocation across plans | Putting global budgets in `plans` mixes two areas and records (constitution IV: one package per area) |

## Consolidation (2026-10-20, binding)

The consolidation of 014–017 settled the points below. They win where this plan, its data model, its tasks or its
fragment say otherwise (as D-200). The gap-closure plan, [docs/reports/brd-gap-closure-plan.md](../../docs/reports/brd-gap-closure-plan.md),
gives the build order across the four features.

- **Decisions.** D-A1–D-A19 are D-223–D-241 in [decision-log.md](../../docs/sdd/decision-log.md) (the tags are
  kept above); the consolidation decisions are D-298–D-309.
- **Contract (D-298, D-299, D-300).** The fragment is merged into `backend/api/openapi.yaml`; T001 is done. The 22
  operations are in `notYetServed`. The properties 014 adds to existing schemas (`CurrentUser.access`, `Plan.access`,
  `Plan.allocation`, `PlanSummary.access`, `PlanCampaigns.access`, `ActivitySummaryPage.access`, `Campaign.access`,
  `Initiative.access`, `Activity.access`, `LineItemSection.access`, `LineItemDetails.access`,
  `ApprovalPerson.canApprove`, `RecycleBinListItem.access`) are optional in the contract; the API always sends them,
  and the web app treats an absent `access` as "available" (as before 014). `Error.code` has `forbidden`; every
  operation except the four public ones lists `403 Forbidden`. `ActionAvailability` is the contract's one
  availability shape; 008's `ApprovalActionAvailability` is an alias of it.
- **First web task (typecheck).** Since the merge, `npm run typecheck` fails in `frontend/src/api/errors.ts`
  (`KIND_BY_CODE` lacks `forbidden`). T039 adds
  `forbidden: 'forbidden'` with a new `ApiErrorKind`, before anything else.
- **Segregation of duties (D-302).** The Rules switch stays, on by default; 017 follows it. The names are
  `noSelfApproval` (contract) and `Rules.NoSelfApproval` (`authz`).
- **Money (D-303).** An allocation is entered and kept in the allocated plan's currency (USD for every plan until
  016 lane C is built): `BudgetAllocationCreate.amount` and `BudgetAllocationUpdate.amount` in another currency are
  a field problem on `amount`, "Enter the amount in <code>, the plan's currency." `PlanAllocation.amount` equals the
  plan budget. `GlobalBudgetFigures` and `OtherMarketsRow` are USD: allocations and planned costs converted once each
  at the current rate, on read. `BudgetAllocationRow` gains `amountUsd` (the allocation in USD); its `amount`,
  `plannedCost` and `leftToPlan` are in the plan's currency. 016 locks the plan currency while a plan has an
  allocation (`Plan.currencyLock`).
- **Hooks for 017 (D-290).** `internal/access/hooks.go` also exports `ApprovingRoles(ctx) ([]RoleView, error)`: the
  roles that include `approvals.decide`, sorted by name. `RoleHolders` is as above.
- **Financial sections (D-305).** Feature 015 removes the "From finance (coming later)" sections; `seeFinance`
  then decides whether 015's Financials card (activity page) and sections (line item) are shown. Until 015 is
  built, it decides the "From finance" sections as written here.
- **Log visibility of the new kinds (D-304).** `role` and `accessRules` need `activityLog.viewAll`; `globalBudget`
  needs `globalBudgets.view` for All markets or `activityLog.viewAll`; `budgetAllocation` follows its plan. The
  same adapter (`app/history_access.go`) also answers for 015's, 016's and 017's kinds as D-304 says.
- **Navigation (D-305).** Sidebar: Home · Plans · Global budgets · Approvals · All work · Financials (015). Settings
  cards: the existing ones, then Roles and access, then Field help (016).
- **Demo data (D-306).** `addAccess` runs after `addTeams`; `addGlobalBudgets` after `addPlans` (016's P-1004 is a
  2027 plan and gets no allocation).
- **Constitution and ADRs (D-308).** T063's constitution wording and ADR-0008 are done once for 014, 015 and 016,
  as the **first** task of the build (before code): constitution 1.1.0 (Principles II and VI, Technology Choices,
  Current Phase & Scope), ADR-0008 "Role-based access with team scopes", ADR-0009 "Simulated GSAP feeds behind
  adapters", and CLAUDE.md non-negotiables 6 and 11.
- **Build order (D-307).** Phase 2 (backend foundation: `authz`, `errs.Forbidden`, the access middleware and route
  table, `apitest.GrantRole`, the seeded roles) is merged first, **together with T062** (the existing journeys sign
  in as the persona whose role allows each step), so the end-to-end suite stays green when the middleware lands.
  015's and 016's backend foundations may run beside it and rebase on it; every other lane of 014–017 starts
  after it. The web foundation (T039–T042) is the first web task of wave 2, on the UI foundation's components.
  Shared code is changed in this order: `internal/allwork` and `app/allwork_source.go` 014 → 016 lane A → 015;
  `plans` 014 → 016; `approvals` 014 (US5) → 017; `people` 014 → 017.

### Table C additions (D-301): the operations of 015, 016 and 017

Each feature adds its rows to `internal/access/routes.go` in the change that serves the operation;
`app/access_routes_test.go` fails until it does.

| Operation | Rule |
|---|---|
| GET /api/financials/records/{recordId} | View (the record) + InPlan finance.view (the record's plan) |
| GET /api/financials/transactions | Need finance.view → filtered: transactions linked under plans the person can see and holds finance.view in; not-linked ones only with finance.import |
| GET /api/financials/transactions/{transactionId} | Need finance.view; a linked transaction needs View and finance.view in its plan, a not-linked one needs finance.import (else 403, message 4) |
| POST /api/financials/transactions/{transactionId}/link | InPlan finance.link (the target line item's plan, which the person must see) |
| POST /api/financials/transactions/{transactionId}/unlink | InPlan finance.link (the linked line item's plan) |
| GET /api/financials/overview | Need finance.view → filtered |
| GET /api/financials/imports | Need finance.view |
| POST /api/financials/imports/check · POST /api/financials/imports · GET /api/financials/imports/template | Need finance.import |
| GET /api/currencies/{code}/rates · GET /api/exchange-rates/current · GET /api/exchange-rates/imports | Signed in |
| POST /api/currencies/{code}/rates · POST /api/currencies/{code}/rates/impact · PATCH, DELETE /api/exchange-rates/{rateId} · POST /api/exchange-rates/{rateId}/impact · POST /api/exchange-rates/import-check · POST /api/exchange-rates/import · GET /api/exchange-rates/import-sample | Need exchangeRates.manage |
| GET /api/field-help | Signed in |
| PATCH /api/field-help/{helpKey} | Need settings.manage |
| PUT /api/plans/{planId}/budget-phasing | InPlan plans.edit |
| PUT /api/campaigns/{campaignId}/budget-phasing · PUT /api/initiatives/{initiativeId}/budget-phasing | InPlan campaigns.edit |
| GET /api/plans/{planId}/budget-by-month | View |
| GET /api/activities/{activityId}/split-options · POST …/split-preview · POST …/split | InPlan activities.edit |
| GET /api/all-work/breakdown | Signed in → filtered (with `plan`: View of that plan) |
| GET /api/approval-rule-options | Signed in |
| GET /api/approval-delegations | Custom: `scope=mine` signed in; `scope=all` needs workflows.manage (service) |
| POST /api/approval-delegations | Custom: for oneself signed in; for someone else workflows.manage (service) |
| PATCH /api/approval-delegations/{delegationId} · POST …/{delegationId}/cancel | Custom: the delegating person, or workflows.manage (service) |
| GET /api/approvals/monitoring | Signed in → filtered (the rows of All open) |
| POST /api/approvals/escalations/run | Need workflows.manage |

All work with 015: the Financials column group, the `finance` filter and their export columns are offered when the
person holds `finance.view` in any scope; rows of plans where they do not hold it show "—" in those columns and do
not match the `finance` filter. API refusal texts are 014's messages 1–4; 015's sentence "You don't have access to
financial data — ask an administrator." is the web app's in-place note where a Financials card or column would be.
