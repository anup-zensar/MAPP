# Plan module build guide (features 002–012)

The one page build agents read before they start. It comes from the consolidation of the 002–012 designs on
2026-10-19 (decisions D-189–D-207). Precedence: this guide and the *Consolidation* section at the end of each
`specs/0NN-*/plan.md` win over the rest of that plan, its data model and its tasks (D-200). The contract is
`backend/api/openapi.yaml` (all 002–012 operations merged, D-192); the decisions are in
[decision-log.md](decision-log.md).

## 1. Build order

| Step | Features | Notes |
|---|---|---|
| 0 | Foundation | Backend: `platform/money` (002's design), `platform/date` (005's design, D-198), the ID prefixes below, `model.go` + `repository.go` + `store_json.go` for every stored record type, the demo seed (order in §6). Web: every route with a stub page, every embedded section as a stub in its owner's folder, the shared components of §4 |
| 1 | 002 + 003 | Settings; people, teams, profile |
| 2 | 004 | Plans |
| 3 | 005 | Campaigns and initiatives |
| 4 | 006 | Activities |
| 5 | 007 | Line items |
| 6 | 008 | Approvals |
| 7 | 009, 010, 011 | Home, All work, Activity log and recycle bin (011's history query first: 009's Recent changes uses it) |
| 8 | 012 | Bulk actions and import (after 010's and 004–006's web parts) |

Per feature, one **backend** agent and one **frontend** agent work in parallel from the contract (D-200).

## 2. Features at a glance

| Feature | Go packages (`backend/internal/…`) | Web folders (`frontend/src/…`) | Stored tables · ID prefixes | Owns in the contract |
|---|---|---|---|---|
| 002 Settings | `platform/money`, `settings`; changes `history` (`SnapshotsInBin`), `platform/errs` (`in_use`, `cannot_restore`) | `features/settings/` (+ `shared/`, `sections.ts`), `api/settingsQueries.ts`, `lib/decimal.ts`, `lib/waterfall.ts`, `lib/text.ts` | `models` MOD, `activity_types` AT, `shell_attributes` SA, `shell_attribute_options` SO, `currencies` (ISO code), `finance_budgets` FB, `finance_budget_items` FBI | Money, MoneyInput, Percent, ExchangeRate, SettingRef, Usage, Lock, StatusUpdate, ChangeImpact, ImpactWarning, AttributeValue(s), SplitShare, AttributeForm…, MissingField (`reason` optional), Model, ActivityType, Currency, FinanceBudget(Item) |
| 003 People and teams | `people` (extended: teams, memberships, profile, preferences, usage hooks); `errs` (`not_allowed`) | `features/people/` (people, teams, My profile), `lib/useFormatters.ts`, `lib/format.ts` (date formats) | `teams` T, `team_memberships` TM; `people`, `preferences` (001) | PersonRef, TeamRef, TeamColour, Person…, Team…, DateFormat; extends CurrentUser, Preferences (optional fields, D-191) |
| 004 Plans | `plans`; `errs` (`not_ready` + `Missing`) | `features/plans/`, `lib/recordLinks.ts` | `plans` P | Plan…, PlanStatus, GoalStage, SpreadMethod, ApprovalProgress, Completeness, AttentionItem, ReadinessGroup, ModelSummary, FinanceBudgetRef, OptionRef, PlanId; Error.missing |
| 005 Campaigns and initiatives | `campaigns`, `rollup` (pure), `platform/date` | `features/campaigns/` (+ `extensions.ts`) | `campaigns` C, `initiatives` I | PlanRef, CampaignRef, BudgetAndCost, ApprovalSummary, RecordCompleteness, Campaign…, Initiative…, Destination… |
| 006 Activities | `activities` | `features/activities/` | `activities` A; seeds `approval_workflows` WF (008's shape) | Activity…, ActivityStatus, ActivityDisplayStatus, ActivityForm…, ActivityId |
| 007 Line items | `lineitems`; `history` (`Kind.AfterRestore`) | `features/line-items/` | `line_items` L | LineItem…, Spread…, SwbsStatus, FinanceBudgetItemRef, LineItemId, RestartApproval (query) |
| 008 Approvals | `approvals` | `features/approvals/`, `features/approval-workflows/`, `features/notifications/` | `approval_workflows` WF, `approval_rounds` AR, `approval_actions` AH, `notifications` NT | ApprovalWorkflow…, ActivityApproval, ApprovalRow…, ApprovalCounts, Notification…, WorkflowId, NotificationId |
| 009 Home | `home` (read-only) | `features/home/` | — (preference `homeScope`) | HomeSummary…, HomeAttentionItem/Reason/Page, Waiting…, MyActivity…, HomeScope |
| 010 All work | `allwork`, `platform/sheet` | `features/all-work/`, `lib/download.ts` | `saved_views` SV, `all_work_preferences` | AllWork…, SavedView… and the AllWork* parameters |
| 011 Activity log and recycle bin | `history` (query, bin, purge, preview) | `features/activity-log/`, `features/recycle-bin/` | `activity_log`, `recycle_bin` (001); preference `historyIncludeInside` | ActivityLog…, RecycleBinListItem, RestorePreview…, PurgeResult, EmptyRecycleBin…; extends 001 schemas (optional fields, D-191) |
| 012 Bulk actions and import | `bulk`, `imports`; `history` (`WithNote`), `platform/web` (`ReadJSONUpTo`), `platform/sheet` (readers) | `features/bulk/`, `features/imports/` | `bulk_actions` BLK, `imports` IMP | Bulk…, Import…, BulkActionId, ImportId |

Error codes added once in the merged contract and in `frontend/src/api/errors.ts`: `in_use`, `cannot_restore` (002),
`not_allowed` (003), `not_ready` + `missing` (004), `parent_deleted_permanently` (011). Go constants are added by the owner
named in the table (the first feature that needs one adds it if the owner is not built yet, with exactly that name).

## 3. Cross-feature hooks (backend)

Rule (D-196): the calling package declares a small **consumer interface**; `backend/internal/app` holds the adapter that
implements it over the providing package, and `app.New` wires it. No package imports a later feature's package.

| Hook | Declared by | Implemented / registered by |
|---|---|---|
| `history.Kind` registration (parents first) | 001 | people: person, team, teamMembership (003) · settings kinds (002) · plan (004) · campaign, initiative (005) · activity (006) · lineItem (007) · approvalWorkflow (008) |
| `settings.ReferenceSource` + `AddReferenceSource` ("Used by", in-use, impact) | 002 | 003 people · 004 plans (with name) · 005 campaigns, initiatives · 006 activities · 007 line items (cost in own currency) |
| `people.UsePlans` / `AddPersonUsage` / `UseCurrencies` | 003 | 004 plans (`PlanUse` with `Name`) · 005, 006, 008 owners/approvers · 002 currencies |
| `plans.Settings`, `plans.Directory` | 004 | adapters over 002 and 003 |
| `plans.Level` + `RegisterLevel` (figures, completeness, copy with `CopyJob`, D-195) | 004 | campaign, initiative (005) · activity (006) · lineItem (007), top-down |
| `campaigns.Activities` (`Summaries`, `CopyAll(…, yearShift)`) | 005 (`NoActivities{}` default) | 006 |
| `activities.LineItems` (`Facts`, `CopyAll(…, yearShift)`, `DatesChanged`) + `UseLineItems` | 006 | 007 |
| `activities.Workflows`, `activities.ApprovalHooks` (`ApprovalRestarted`, `ActivityRestored`) | 006 | 008 (adapters `app/approvals_adapter.go`) |
| `(*activities.Service).RestartApproval(ctx, a, id, reasons)` | 006 exports | called by 007 (line-item changes, `AfterRestore`) |
| `history.Kind.AfterRestore` | 007 adds | 007 (restart on restore); 011's dry-run preview relies on it |
| `lineitems.Activities`, `approvals.Activities` | 007, 008 | adapters over 004/006 |
| `home.Sources` | 009 | adapters over 003–008 and 011's `QueryLog` |
| `allwork` sources | 010 | adapters over 004–007 |
| `bulk.Planning`, `imports.Planning`; `history.WithNote` | 012 | adapters over 005–008 (pass the version read in the same transaction) |

Names listed under *Expects* in 009–012 are the consumer side: their adapters are written against the real exported API
of the earlier features when they are built.

## 4. Shared web parts — one owner each (D-197)

| Owner | Components and helpers (`src/components/<Name>/`, exported from `index.ts`, test + gallery entry) |
|---|---|
| 001 | PageHeader, Card, SummaryCard, StatusChip, CompletenessChip, SidePanel, ConfirmDialog, ActionsMenu, FormErrorSummary, TextField, Button/LinkButton, InfoTip, Tooltip, EmptyState, ErrorState, Skeleton, Avatar, Toast/useToast; `useUndoableDelete`, `useUnsavedChanges`, `usePageTitle`, `conflictMessage`, `lib/format.ts` |
| 002 | Breadcrumb, SearchField, SegmentedFilter, MoneyField, DecimalField, CheckboxField, TextAreaField (`rows?`), ChoiceField, SplitEditor, LockedNote, ShellAttributeFields (+ ShellAttributeValues, ClearValuesDialog, rules.ts); `TextField.suggestions`; `api/settingsQueries.ts` |
| 003 | Combobox (base of ChoiceField's search mode and the pickers), PersonPicker, TeamPicker, PersonLabel, TeamPill/TeamPills (`TeamRef.colour`, tokens `--team-<colour>`), FormPanel; `ActionsMenu.hint`; `useFormatters()`, `parseDate` |
| 004 | TreeTable (+ filterTree), ProgressBar, LevelTag, FormDialog, FilterChips, PillTabs, FormSection, FormFooter; `SummaryCard.detail`; `CompletenessChip.itemsBelow` (count or list, D-199); `lib/recordLinks.ts` `recordHref` (D-194) |
| 005 | DateRangeField, `formatDateRange` |
| 008 | CountBadge, `formatRelativeTime`, `formatWaitingTime` |
| 009 | SummaryCard link variant (`icon`, `to`, `linkLabel`) |
| 010 | `lib/download.ts` |
| 011 | HistorySection (`features/activity-log/`) |
| 012 | SelectionBar; selection props on TreeTable and AllWorkGrid |

Not built (draft duplicates): FilterPills, PillFilter, SegmentedControl → SegmentedFilter · PageHeader breadcrumbs →
Breadcrumb · SelectField, MultiChoiceField, YesNoField, NumberField → ChoiceField, CheckboxField, DecimalField · MoneyInput →
MoneyField · AttributeFields, AttributeValueList/View → ShellAttributeFields, ShellAttributeValues · lib/teamColor.ts → `TeamRef.colour`.

## 5. Web routes and embedded sections

| Owner | Routes |
|---|---|
| 002 | `/settings`, `/settings/models`, `/settings/activity-types`, `/settings/shell-attributes` (+ `/new`, `/:attributeId`, `/:attributeId/edit`), `/settings/currencies`, `/settings/finance-budget-items` |
| 003 | `/settings/people/:personId?`, `/settings/teams/:teamId?`, `/profile` |
| 004 | `/plans`, `/plans/new`, `/plans/:planId`, `/plans/:planId/edit` |
| 005 | child routes of `/plans/:planId`: `campaigns/new`, `campaigns/:campaignId`, `…/edit`, `campaigns/:campaignId/initiatives/new`, `initiatives/:initiativeId`, `…/edit` |
| 006 | `/plans/:planId/initiatives/:initiativeId/activities/new`, `/plans/:planId/activities/:activityId`, `…/edit`; short `/activities/:activityId` (redirect, D-194) |
| 007 | `/activities/:activityId/line-items/new`, `/line-items/:lineItemId`, `/line-items/:lineItemId/edit` |
| 008 | `/approvals` (`?tab=`, `?activity=`), `/settings/approval-workflows`, `/settings/approval-workflows/:workflowId` |
| 009 · 010 · 011 · 012 | `/` (Home) · `/all-work` · `/activity-log`, `/recycle-bin` · `/plans/:planId/import`, `/plans/:planId/imports/:importId` |

Sections leave `COMING_SOON_SECTIONS` in `app/navigation.ts` when built: Settings (002), Plans (004), Approvals (008),
All work (010), Activity log and Recycle bin (011); My profile (003) is in the account menu.

| Host page (owner) | Embedded part (owner, its folder) |
|---|---|
| PlanPage (004) | campaign/initiative rows, toolbar action and totals footer via `usePlanTreeNodes` and `features/campaigns/extensions.ts` (005); HistorySection (011); Import activities + PlanImportsCard, tree selection (012) |
| Initiative rows and panel (005) | activity rows and **Add activity** in `features/campaigns/extensions.ts` (006) |
| ActivityPage (006) | LineItemsSection `#line-items` and SpreadEffectNote in the edit form (007); ActivityApprovalSection `#approval` and the Submit handler (008); HistorySection (011) |
| SettingsPage (002, `sections.ts`) | People, Teams cards (003); Approval workflows card (008) |
| NotificationBell, Sidebar (001) | notifications list and badges (008) |
| AllWorkGrid (010) | selection and SelectionBar (012) |

## 6. Demo data (D-202)

`seed.Load`, in this order: people (001) → Settings (002, returns `seed.SettingsSeed`, IDs by name) → teams T-1001…T-1003 and
memberships (003) → plans P-1001…P-1003 (004) → campaigns, initiatives (005) → activities A-1001…A-1011 (006) → line items
L-1001…L-1008 (007) → workflows WF-1001, WF-1002 (seeded by 006 in 008's shape), rounds and notifications (008) → demo
history (011). Steps look up earlier data by name and write **no** activity-log entries except 011's.

## 7. Conventions for build agents

- **Contract is fixed** (D-192). Build against `backend/api/openapi.yaml`; do not edit it — report problems. Remove your
  operations from `notYetServed` in `backend/internal/app/app_test.go` when you serve them. Schemas 001 serves that later
  features extend keep 001's required list; always send the new fields (D-191).
- **Stay in your lane.** Change only your packages and folders, plus the shared files assigned to you above. Shared files
  everyone touches (`internal/app/app.go`, `seed/seed.go`, `app/router.tsx`, `app/navigation.ts`, `components/index.ts`,
  `styles/tokens.css`, `platform/errs/errs.go`): add your lines, never rewrite others'.
- **Every record type** (as `internal/samples`): `record.Meta` (version, created/updated by and at), readable ID from
  `ids.Generator` with its prefix, update checks the version (`409 version_conflict` naming who and when) and calls `Touch`,
  one activity-log entry per change via `history.Log` in the same `Write`, delete only with `history.MoveToBin` and the web
  app's `useUndoableDelete` (exception: finance budgets, D-047). All field problems at once (`errs.Problems`).
- **Shapes.** Lists `{items, total}` (+ summaries a spec asks for). Errors as 001. Money `{"amount": "1234.50", "currency":
  "USD"}`; requests use `MoneyInput` or decimal strings with at most two decimals (D-190); never floats. Dates `YYYY-MM-DD`
  via `platform/date`. Planning records PATCH the whole form + `version`; Settings, people, teams, workflows PATCH only the
  changed fields (D-203). Re-approval: `restartApproval` (D-189). Shell attribute values in `attributes` (D-048).
- **Derived values are computed on read, never stored** (D-041): USD amounts, display-currency conversions (server side),
  roll-ups, completeness, goals.
- **Settings reads** only through 002's readers and attribute functions (002 plan, *How later features read Settings*).
- **Tests first**, names carry the FR (`TestX_Y_FR012`); handler tests with `apitest` validate every request and answer
  against the contract; web tests with `mockApi`; Playwright + axe for P1 stories.
- **Checks before you finish.** Backend: `gofmt -l .` prints nothing, `go vet ./...`, `go test ./...`. Frontend:
  `npm run check` (api:check, typecheck, lint, format, tests).
- **Rules.** 127.0.0.1 only; never log personal data; no secrets; no roles yet (D-004); vocabulary Plan → Campaign →
  Initiative → Activity → Line item; plain-language messages exactly as the plan's message tables.
- **Toolchain.** Git Bash on Windows: `export PATH="$HOME/tools/node:/c/Program Files/Go/bin:$PATH"`; write files with the
  editor tools (heredocs with apostrophes break); keep LF line endings.
