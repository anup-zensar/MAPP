# Measure module — create the GitHub issues (hand-off)

Everything needed to create the Measure module's GitHub issues on **`anup-zensar/MAPP`**: 8 labels, 2 milestones, 18 work issues and 4 lane trackers. Every title, label, milestone and body is below, ready to copy. The issue texts come from `docs/sdd/measure-issues.md` on the `measure-module` branch; the build guide is `docs/sdd/measure-module-build.md`.

Two ways to do it: **A** hand the prompt below to Claude Code on a machine where the GitHub CLI (`gh`) is installed and signed in; **B** create them by hand in the GitHub web UI (about 30 minutes).

## A — Prompt for Claude Code

Save this file on the machine, open Claude Code in any folder, and paste:

````text
Create the Measure module's GitHub issues on anup-zensar/MAPP with the gh CLI, exactly as described in the file
measure-issues-setup.md (I will give you its path). Copy every title, label list, milestone and body byte for byte.

1. Run `gh auth status`. If gh is missing or not signed in with write access to anup-zensar/MAPP, stop and tell me.
2. Labels: for each row of the "Labels" table run
   gh label create "<name>" --repo anup-zensar/MAPP --color <colour> --description "<description>" --force
3. Milestones: read `gh api "repos/anup-zensar/MAPP/milestones?state=all"`; for each row of the "Milestones" table that
   does not exist yet run  gh api repos/anup-zensar/MAPP/milestones -f title="<title>" -f description="<description>"
4. Read the existing issues once:
   gh issue list --repo anup-zensar/MAPP --label measure --state all --limit 200 --json number,title
   and skip any issue below whose title already exists (use its number instead).
5. Work issues, in the order listed under "Work issues": write the Body block (the text inside the four-backtick
   fence, without the fence lines) to a temporary UTF-8 file with LF line endings, then
   gh issue create --repo anup-zensar/MAPP --title "<title>" --label "<labels>" --milestone "<milestone>" --body-file <file>
   Record each issue's number against its key (M0, 039, 040, … DEMO).
6. Lane trackers, last: in each tracker Body replace every placeholder #ISSUE-<key> (for example #ISSUE-041)
   with # and that key's issue number, then
   gh issue create --repo anup-zensar/MAPP --title "<title>" --label "<labels>" --body-file <file>   (no milestone)
7. Print a table: number, title, lane. Do not edit, commit or push anything in any repository.

````

## B — By hand in the GitHub web UI

1. **Labels:** open https://github.com/anup-zensar/MAPP/labels and use **New label** for each row of the Labels table (colour without `#`).
2. **Milestones:** open https://github.com/anup-zensar/MAPP/milestones/new for each row of the Milestones table; no due date.
3. **Work issues:** for each one in order, open https://github.com/anup-zensar/MAPP/issues/new, paste the title, paste the body (the text inside the four-backtick fence), pick its labels and milestone in the right-hand panel, **Create**, and note its number next to its key.
4. **Lane trackers:** create them last; in each body replace every `#ISSUE-<key>` with the real number (`#ISSUE-041` → `#57`). Labels as given, no milestone.
5. Each developer assigns themselves to their lane's tracker and to the issues it lists.

## Labels

| Name | Colour | Description |
|---|---|---|
| `measure` | `0E7C86` | Measure module (features 039-050) |
| `wave-a` | `C5DEF5` | Measure wave A: reports from data MAPP holds |
| `wave-b` | `BFD4F2` | Measure wave B: performance reporting |
| `lane-a` | `FFC600` | Measure lane A: foundation, plan budget, attribution |
| `lane-b` | `F9A825` | Measure lane B: performance data, projections, revenue |
| `lane-c` | `A2EEEF` | Measure lane C: approvals, purchase requests, summary |
| `lane-d` | `D4C5F9` | Measure lane D: plan details, GSAP link, user access |
| `tracking` | `EDEDED` | A lane's tracking issue: its work in order |

## Milestones

| Title | Description |
|---|---|
| Measure wave A | Reports from data MAPP holds: 039-045, integration, QA pass, demo data. |
| Measure wave B | Performance reporting: 046-050, integration, QA pass. |

## Work issues (create in this order)

### 1. `M0` — Measure · Kick-off: check script and Claude Code hook

- **Title:** `Measure · Kick-off: check script and Claude Code hook`
- **Labels:** `measure,wave-a,lane-a`
- **Milestone:** Measure wave A
- **Lane:** A · 8 h

**Body:**

````markdown
**Lane A · 8 h · wave A** — Measure module. Read [measure-module-build.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/measure-module-build.md) first.

Done on 2026-10-30: constitution 4.8.0, decisions D-366–D-370 and Q-028–Q-031, feature map, the build guide,
`hive9-measure-summary.md`, `shell-measure-requirements.md`, `measure-metrics.md`, CI on `measure-module`.
Remaining: `scripts/check-changed.ps1` and its Claude Code hook in `.claude/settings.json` (format, lint, comment
budget, file length, tests of changed packages and folders).

- [ ] `scripts/check-changed.ps1` runs in under two minutes on a one-feature change
- [ ] Hook runs it after each implementation step and prints only failures
````

### 2. `039` — 039 · Measure foundation

- **Title:** `039 · Measure foundation`
- **Labels:** `measure,wave-a,lane-a`
- **Milestone:** Measure wave A
- **Lane:** A · 28 h

**Body:**

````markdown
**Lane A · 28 h · wave A** — Measure module. Read [measure-module-build.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/measure-module-build.md) first.

**Goal.** The Reports area and everything every report shares: report frame, filter panel (year, plans, Shell
attributes per level with *Not filled in*, owner, activity type, status, team), periods (months, quarters),
display currency, saved report views (per person, a named address, star a default), table view and CSV/Excel
export of every widget, the `reports.view` permission, the route rules and the wiring hub with one stub per report.

**Owns.** `backend/internal/measure`, `access/routes_measure.go`, `app/measure.go` + `app/measure_<report>.go`
stubs, migration 00028 (`report_views`), `frontend/src/features/measure/` root and `shared/`, every
`/reports/…` route in `navigation.ts`, `pages.tsx`, `router.tsx`, `features/measure/index.ts` with the stubs,
`e2e/measure.spec.ts`. Writes the **wave A contract** into `openapi.yaml` from every lane's `contracts/api.md`.

**References.** FA §E2.4, §E3.0, §E3.9.0; client 016 US2, US6–US8 (FR of the shared frame); D-367, D-368;
036 `dashboard` package and `features/dashboards` as the model.

**Stories.** US1 open the Reports area and a report with filters in the address (P1) · US2 save, star and delete
a view (P1) · US3 table view and export of a widget (P1) · US4 empty states that say why (P2).

**Order.** Backend types, registry, rules, hub and migration first (hour 16, pushed so lanes can import), then
the web frame and the stubs (hour 28).

### Checklist

- [ ] Branch `feature/039-measure-foundation` from `measure-module`
- [ ] `/speckit-specify` citing the client FRs and FA §E; `/speckit-clarify` answers recorded as `039-Cn`
- [ ] `/speckit-plan` (short), `contracts/api.md` handed to the integrator for the wave contract
- [ ] `/speckit-tasks`, `/speckit-analyze` with no CRITICAL issue
- [ ] `/speckit-implement` one user story at a time; `scripts/check-changed.ps1` green after each
- [ ] Scope-isolation test for every scoped store read; contract test for every operation; one test per widget state
- [ ] End-to-end test for every P1 story in `frontend/e2e/` (axe included)
- [ ] Benchmark at the 028 volumes recorded in `plan.md`
- [ ] `/speckit-converge` clean; feature-map status change written in the pull request
- [ ] Pull request into `measure-module`, three checks green, one approval

### Start prompt for Claude Code

```text
I'm on lane A of the Measure module, building feature 039 Measure foundation.
1. Read docs/sdd/measure-module-build.md, then the "## #039" section of docs/sdd/measure-issues.md (the text of
   this GitHub issue), then the rows of docs/product/measure-metrics.md and the client lines in
   docs/reference/shell-measure-requirements.md it cites.
2. git switch measure-module && git pull. Run /speckit-specify with
   SPECIFY_FEATURE_DIRECTORY=specs/039-measure-foundation and GIT_BRANCH_NAME=feature/039-measure-foundation,
   using the issue's Goal, Stories and References as the feature description.
3. Read-only report: no create/edit/delete scenarios; write the state scenarios instead
   (data, empty, no access, no financial access, filters exclude everything).
4. Stop after the spec and its quality checklist so I can review. Record clarifications as 039-Cn in the
   spec, never in docs/sdd/decision-log.md. Touch only the paths this issue lists under "Owns".
```
````

### 3. `040` — 040 · Plan budget report

- **Title:** `040 · Plan budget report`
- **Labels:** `measure,wave-a,lane-a`
- **Milestone:** Measure wave A
- **Lane:** A · 16 h

**Body:**

````markdown
**Lane A · 16 h · wave A** — Measure module. Read [measure-module-build.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/measure-module-build.md) first.

**Goal.** Budget, planned cost, committed and actual by period split at today, summary cards, and the **ranked
list** (show plan, campaign, initiative or activity; group by none, plan, owner or a Shell attribute; sort by any
figure; order; page size; open a row to show its level below with the row on top; a link to the record). The
ranked list is exported for 049.

**Owns.** `measure/planbudget`, `app/measure_planbudget.go`, `queries/measure_planbudget.sql`,
`features/measure/plan-budget/` (exports `RankedList`), `e2e/measure-plan-budget.spec.ts`.

**References.** FA §E3.4, §E3.2.5 (drill); client 017 US1, US3, FR-001–FR-021 (money series, cards, list);
[measure-metrics.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/product/measure-metrics.md) §1; 036 figures (D-342), 010 Budget & cost (D-266).

**Stories.** US1 spend pacing by period with the today split (P1) · US2 ranked list with drill and record link
(P1) · US3 group by and sort (P2) · US4 export (P2).

### Checklist

- [ ] Branch `feature/040-plan-budget-report` from `measure-module`
- [ ] `/speckit-specify` citing the client FRs and FA §E; `/speckit-clarify` answers recorded as `040-Cn`
- [ ] `/speckit-plan` (short), `contracts/api.md` handed to the integrator for the wave contract
- [ ] `/speckit-tasks`, `/speckit-analyze` with no CRITICAL issue
- [ ] `/speckit-implement` one user story at a time; `scripts/check-changed.ps1` green after each
- [ ] Scope-isolation test for every scoped store read; contract test for every operation; one test per widget state
- [ ] End-to-end test for every P1 story in `frontend/e2e/` (axe included)
- [ ] Benchmark at the 028 volumes recorded in `plan.md`
- [ ] `/speckit-converge` clean; feature-map status change written in the pull request
- [ ] Pull request into `measure-module`, three checks green, one approval

### Start prompt for Claude Code

```text
I'm on lane A of the Measure module, building feature 040 Plan budget report.
1. Read docs/sdd/measure-module-build.md, then the "## #040" section of docs/sdd/measure-issues.md (the text of
   this GitHub issue), then the rows of docs/product/measure-metrics.md and the client lines in
   docs/reference/shell-measure-requirements.md it cites.
2. git switch measure-module && git pull. Run /speckit-specify with
   SPECIFY_FEATURE_DIRECTORY=specs/040-plan-budget-report and GIT_BRANCH_NAME=feature/040-plan-budget-report,
   using the issue's Goal, Stories and References as the feature description.
3. Read-only report: no create/edit/delete scenarios; write the state scenarios instead
   (data, empty, no access, no financial access, filters exclude everything).
4. Stop after the spec and its quality checklist so I can review. Record clarifications as 040-Cn in the
   spec, never in docs/sdd/decision-log.md. Touch only the paths this issue lists under "Owns".
```
````

### 4. `041` — 041 · Approvals report

- **Title:** `041 · Approvals report`
- **Labels:** `measure,wave-a,lane-c`
- **Milestone:** Measure wave A
- **Lane:** C · 10 h

**Body:**

````markdown
**Lane C · 10 h · wave A** — Measure module. Read [measure-module-build.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/measure-module-build.md) first.

**Goal.** Status mix of approval rounds by record type (activities, purchase requests, modified purchase
requests), average days to a decision, and an approvals grid (record, approver, level, submitted, decided,
outcome, amount, comments) with approver, status and record-type filters, search, paging and export.

**Owns.** `measure/approvalreport`, `app/measure_approvalreport.go`, `queries/measure_approvals.sql`,
`features/measure/approvals/`, `e2e/measure-approvals.spec.ts`.

**References.** FA §E3.6; client 018 US1, US4, FR-001–FR-012 (widgets, grid columns, filters); 017's
average-days rule (D-342); 031 record types.

**Stories.** US1 workload and cycle time (P1) · US2 find and export approval records (P2).

### Checklist

- [ ] Branch `feature/041-approvals-report` from `measure-module`
- [ ] `/speckit-specify` citing the client FRs and FA §E; `/speckit-clarify` answers recorded as `041-Cn`
- [ ] `/speckit-plan` (short), `contracts/api.md` handed to the integrator for the wave contract
- [ ] `/speckit-tasks`, `/speckit-analyze` with no CRITICAL issue
- [ ] `/speckit-implement` one user story at a time; `scripts/check-changed.ps1` green after each
- [ ] Scope-isolation test for every scoped store read; contract test for every operation; one test per widget state
- [ ] End-to-end test for every P1 story in `frontend/e2e/` (axe included)
- [ ] Benchmark at the 028 volumes recorded in `plan.md`
- [ ] `/speckit-converge` clean; feature-map status change written in the pull request
- [ ] Pull request into `measure-module`, three checks green, one approval

### Start prompt for Claude Code

```text
I'm on lane C of the Measure module, building feature 041 Approvals report.
1. Read docs/sdd/measure-module-build.md, then the "## #041" section of docs/sdd/measure-issues.md (the text of
   this GitHub issue), then the rows of docs/product/measure-metrics.md and the client lines in
   docs/reference/shell-measure-requirements.md it cites.
2. git switch measure-module && git pull. Run /speckit-specify with
   SPECIFY_FEATURE_DIRECTORY=specs/041-approvals-report and GIT_BRANCH_NAME=feature/041-approvals-report,
   using the issue's Goal, Stories and References as the feature description.
3. Read-only report: no create/edit/delete scenarios; write the state scenarios instead
   (data, empty, no access, no financial access, filters exclude everything).
4. Stop after the spec and its quality checklist so I can review. Record clarifications as 041-Cn in the
   spec, never in docs/sdd/decision-log.md. Touch only the paths this issue lists under "Owns".
```
````

### 5. `042` — 042 · Purchase request report

- **Title:** `042 · Purchase request report`
- **Labels:** `measure,wave-a,lane-c`
- **Milestone:** Measure wave A
- **Lane:** C · 14 h

**Body:**

````markdown
**Lane C · 14 h · wave A** — Measure module. Read [measure-module-build.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/measure-module-build.md) first.

**Goal.** Counts and amounts by status, commitment by vendor and by owner per delivery month (Approved and Closed
lines by delivery date), invoiced per line, and a grid vendor › purchase request › line › line item with
column choice, paging and export. Vendor delivery reliability is **not** built (Q-030).

**Owns.** `measure/purchasereport`, `app/measure_purchasereport.go`, `queries/measure_purchases.sql`,
`features/measure/purchase-requests/`, `e2e/measure-purchase-requests.spec.ts`.

**References.** FA §E3.7; client 018 US2, US5, FR-013–FR-030; 032 statuses and amounts (D-335), 033 GSAP PR
and PO, 015 invoices.

**Stories.** US1 forward commitment by vendor and owner (P1) · US2 the hierarchical grid (P2) · US3 export (P2).

### Checklist

- [ ] Branch `feature/042-purchase-request-report` from `measure-module`
- [ ] `/speckit-specify` citing the client FRs and FA §E; `/speckit-clarify` answers recorded as `042-Cn`
- [ ] `/speckit-plan` (short), `contracts/api.md` handed to the integrator for the wave contract
- [ ] `/speckit-tasks`, `/speckit-analyze` with no CRITICAL issue
- [ ] `/speckit-implement` one user story at a time; `scripts/check-changed.ps1` green after each
- [ ] Scope-isolation test for every scoped store read; contract test for every operation; one test per widget state
- [ ] End-to-end test for every P1 story in `frontend/e2e/` (axe included)
- [ ] Benchmark at the 028 volumes recorded in `plan.md`
- [ ] `/speckit-converge` clean; feature-map status change written in the pull request
- [ ] Pull request into `measure-module`, three checks green, one approval

### Start prompt for Claude Code

```text
I'm on lane C of the Measure module, building feature 042 Purchase request report.
1. Read docs/sdd/measure-module-build.md, then the "## #042" section of docs/sdd/measure-issues.md (the text of
   this GitHub issue), then the rows of docs/product/measure-metrics.md and the client lines in
   docs/reference/shell-measure-requirements.md it cites.
2. git switch measure-module && git pull. Run /speckit-specify with
   SPECIFY_FEATURE_DIRECTORY=specs/042-purchase-request-report and GIT_BRANCH_NAME=feature/042-purchase-request-report,
   using the issue's Goal, Stories and References as the feature description.
3. Read-only report: no create/edit/delete scenarios; write the state scenarios instead
   (data, empty, no access, no financial access, filters exclude everything).
4. Stop after the spec and its quality checklist so I can review. Record clarifications as 042-Cn in the
   spec, never in docs/sdd/decision-log.md. Touch only the paths this issue lists under "Owns".
```
````

### 6. `043` — 043 · Plan details report

- **Title:** `043 · Plan details report`
- **Labels:** `measure,wave-a,lane-d`
- **Milestone:** Measure wave A
- **Lane:** D · 10 h

**Body:**

````markdown
**Lane D · 10 h · wave A** — Measure module. Read [measure-module-build.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/measure-module-build.md) first.

**Goal.** Generated from Settings: for each active Shell attribute of each level, a chart of records per value, a
chart of planned cost per value and a count of records with the attribute *Not filled in*; plus Planned cost
entered, Goal entered and, once 046 exists, Is integrated. Level tabs Plan · Campaign · Initiative · Activity.

**Owns.** `measure/plandetails`, `app/measure_plandetails.go`, `queries/measure_plandetails.sql`,
`features/measure/plan-details/`, `e2e/measure-plan-details.spec.ts`.

**References.** FA §E3.9.1–E3.9.3; client 019 US2, FR-010–FR-020; 016 Breakdown rules (D-270, D-336); 036
completeness panel; domain model §2.6.

**Stories.** US1 attribute spread and completeness at the activity level (P1) · US2 the other levels (P2) · US3
export (P2).

### Checklist

- [ ] Branch `feature/043-plan-details-report` from `measure-module`
- [ ] `/speckit-specify` citing the client FRs and FA §E; `/speckit-clarify` answers recorded as `043-Cn`
- [ ] `/speckit-plan` (short), `contracts/api.md` handed to the integrator for the wave contract
- [ ] `/speckit-tasks`, `/speckit-analyze` with no CRITICAL issue
- [ ] `/speckit-implement` one user story at a time; `scripts/check-changed.ps1` green after each
- [ ] Scope-isolation test for every scoped store read; contract test for every operation; one test per widget state
- [ ] End-to-end test for every P1 story in `frontend/e2e/` (axe included)
- [ ] Benchmark at the 028 volumes recorded in `plan.md`
- [ ] `/speckit-converge` clean; feature-map status change written in the pull request
- [ ] Pull request into `measure-module`, three checks green, one approval

### Start prompt for Claude Code

```text
I'm on lane D of the Measure module, building feature 043 Plan details report.
1. Read docs/sdd/measure-module-build.md, then the "## #043" section of docs/sdd/measure-issues.md (the text of
   this GitHub issue), then the rows of docs/product/measure-metrics.md and the client lines in
   docs/reference/shell-measure-requirements.md it cites.
2. git switch measure-module && git pull. Run /speckit-specify with
   SPECIFY_FEATURE_DIRECTORY=specs/043-plan-details-report and GIT_BRANCH_NAME=feature/043-plan-details-report,
   using the issue's Goal, Stories and References as the feature description.
3. Read-only report: no create/edit/delete scenarios; write the state scenarios instead
   (data, empty, no access, no financial access, filters exclude everything).
4. Stop after the spec and its quality checklist so I can review. Record clarifications as 043-Cn in the
   spec, never in docs/sdd/decision-log.md. Touch only the paths this issue lists under "Owns".
```
````

### 7. `044` — 044 · GSAP link reports

- **Title:** `044 · GSAP link reports`
- **Labels:** `measure,wave-a,lane-d`
- **Milestone:** Measure wave A
- **Lane:** D · 10 h

**Body:**

````markdown
**Lane D · 10 h · wave A** — Measure module. Read [measure-module-build.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/measure-module-build.md) first.

**Goal.** Three tables in one report page with tabs: SWBS requested and not received (line item, plan, requested
on, days waiting), purchase requests waiting for GSAP (approved and routed, no GSAP PR number), and the purchase
orders per line item (the PO ledger: line item, purchase request, GSAP PR number, PO number and date, PO amount,
invoiced, paid). Each with paging and export; financial columns need `finance.view`.

**Owns.** `measure/gsaplink`, `app/measure_gsaplink.go`, `queries/measure_gsaplink.sql`,
`features/measure/gsap-link/`, `e2e/measure-gsap-link.spec.ts`.

**References.** FA §E3.10.3 (PO report columns), §F4.4 (SWBS Not Received, Pending PR); client 019 US3,
FR-021–FR-030; 030 SWBS statuses, 032 routing, 033 GSAP PR and PO.

**Stories.** US1 SWBS not received (P1) · US2 waiting for GSAP (P2) · US3 purchase orders per line item (P2).

### Checklist

- [ ] Branch `feature/044-gsap-link-reports` from `measure-module`
- [ ] `/speckit-specify` citing the client FRs and FA §E; `/speckit-clarify` answers recorded as `044-Cn`
- [ ] `/speckit-plan` (short), `contracts/api.md` handed to the integrator for the wave contract
- [ ] `/speckit-tasks`, `/speckit-analyze` with no CRITICAL issue
- [ ] `/speckit-implement` one user story at a time; `scripts/check-changed.ps1` green after each
- [ ] Scope-isolation test for every scoped store read; contract test for every operation; one test per widget state
- [ ] End-to-end test for every P1 story in `frontend/e2e/` (axe included)
- [ ] Benchmark at the 028 volumes recorded in `plan.md`
- [ ] `/speckit-converge` clean; feature-map status change written in the pull request
- [ ] Pull request into `measure-module`, three checks green, one approval

### Start prompt for Claude Code

```text
I'm on lane D of the Measure module, building feature 044 GSAP link reports.
1. Read docs/sdd/measure-module-build.md, then the "## #044" section of docs/sdd/measure-issues.md (the text of
   this GitHub issue), then the rows of docs/product/measure-metrics.md and the client lines in
   docs/reference/shell-measure-requirements.md it cites.
2. git switch measure-module && git pull. Run /speckit-specify with
   SPECIFY_FEATURE_DIRECTORY=specs/044-gsap-link-reports and GIT_BRANCH_NAME=feature/044-gsap-link-reports,
   using the issue's Goal, Stories and References as the feature description.
3. Read-only report: no create/edit/delete scenarios; write the state scenarios instead
   (data, empty, no access, no financial access, filters exclude everything).
4. Stop after the spec and its quality checklist so I can review. Record clarifications as 044-Cn in the
   spec, never in docs/sdd/decision-log.md. Touch only the paths this issue lists under "Owns".
```
````

### 8. `045` — 045 · User access report

- **Title:** `045 · User access report`
- **Labels:** `measure,wave-a,lane-d`
- **Milestone:** Measure wave A
- **Lane:** D · 6 h

**Body:**

````markdown
**Lane D · 6 h · wave A** — Measure module. Read [measure-module-build.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/measure-module-build.md) first.

**Goal.** People with role, teams, job title and last sign-in; search, sort, export; opens with `access.manage`.
Demo sign-in records the time of a successful sign-in (migration 00029, `people.last_signed_in_at`, UTC).

**Owns.** `measure/useraccess`, `app/measure_useraccess.go`, the sign-in change in `auth`, migration 00029,
`features/measure/user-access/`, `e2e/measure-user-access.spec.ts`.

**References.** FA §E3.9.5; client 019 US7, FR-040–FR-044; 003 people and teams, 014 roles.

**Stories.** US1 the report with last sign-in (P1) · US2 export (P2).

### Checklist

- [ ] Branch `feature/045-user-access-report` from `measure-module`
- [ ] `/speckit-specify` citing the client FRs and FA §E; `/speckit-clarify` answers recorded as `045-Cn`
- [ ] `/speckit-plan` (short), `contracts/api.md` handed to the integrator for the wave contract
- [ ] `/speckit-tasks`, `/speckit-analyze` with no CRITICAL issue
- [ ] `/speckit-implement` one user story at a time; `scripts/check-changed.ps1` green after each
- [ ] Scope-isolation test for every scoped store read; contract test for every operation; one test per widget state
- [ ] End-to-end test for every P1 story in `frontend/e2e/` (axe included)
- [ ] Benchmark at the 028 volumes recorded in `plan.md`
- [ ] `/speckit-converge` clean; feature-map status change written in the pull request
- [ ] Pull request into `measure-module`, three checks green, one approval

### Start prompt for Claude Code

```text
I'm on lane D of the Measure module, building feature 045 User access report.
1. Read docs/sdd/measure-module-build.md, then the "## #045" section of docs/sdd/measure-issues.md (the text of
   this GitHub issue), then the rows of docs/product/measure-metrics.md and the client lines in
   docs/reference/shell-measure-requirements.md it cites.
2. git switch measure-module && git pull. Run /speckit-specify with
   SPECIFY_FEATURE_DIRECTORY=specs/045-user-access-report and GIT_BRANCH_NAME=feature/045-user-access-report,
   using the issue's Goal, Stories and References as the feature description.
3. Read-only report: no create/edit/delete scenarios; write the state scenarios instead
   (data, empty, no access, no financial access, filters exclude everything).
4. Stop after the spec and its quality checklist so I can review. Record clarifications as 045-Cn in the
   spec, never in docs/sdd/decision-log.md. Touch only the paths this issue lists under "Owns".
```
````

### 9. `046` — 046 · Performance data from a simulated feed

- **Title:** `046 · Performance data from a simulated feed`
- **Labels:** `measure,wave-b,lane-b`
- **Milestone:** Measure wave B
- **Lane:** B · 30 h

**Body:**

````markdown
**Lane B · 30 h · wave B** — Measure module. Read [measure-module-build.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/measure-module-build.md) first.

**Goal.** The record types behind every performance figure, built like the simulated GSAP (ADR-0016, D-370):
data sources with column mappings (Settings › Performance data), performance facts (inquiries, MQLs, deals,
revenue, web metrics) loaded by file upload or a simulated pull, matched to activities on activity ID, media
code, integration ID or external name, rows to review and fix, media codes generated per activity, the external
name set when an activity is approved, and *Is integrated* on the activity. A Performance section on the
activity page shows the activity's facts once it is approved. Permissions `performance.import` and
`performance.link` (reserved by D-367).

**Owns.** `backend/internal/performance`, the `simulatedfeed` adapter, `app/performance.go`, migrations
00030–00032, `features/performance/`, the Performance section slot in `features/activities/ActivityPage.tsx`,
`e2e/performance-data.spec.ts`, wave B demo facts in `seed/measure.go`. Writes ADR-0019 (simulated performance
feed) with its plan and, with 047, the **wave B contract**.

**References.** FA §E4.1, §E7.2–E7.4, §G7; client 020 US1–US6, FR-001–FR-040; 015 import, matching and review
pattern; 030 external keys.

**Stories.** US1 data source and mappings (P1) · US2 upload and match (P1) · US3 performance on an approved
activity (P1) · US4 external names and integration IDs (P1) · US5 review unmatched rows (P2) · US6 media codes
(P2).

### Checklist

- [ ] Branch `feature/046-performance-data` from `measure-module`
- [ ] `/speckit-specify` citing the client FRs and FA §E; `/speckit-clarify` answers recorded as `046-Cn`
- [ ] `/speckit-plan` (short), `contracts/api.md` handed to the integrator for the wave contract
- [ ] `/speckit-tasks`, `/speckit-analyze` with no CRITICAL issue
- [ ] `/speckit-implement` one user story at a time; `scripts/check-changed.ps1` green after each
- [ ] Scope-isolation test for every scoped store read; contract test for every operation; one test per widget state
- [ ] End-to-end test for every P1 story in `frontend/e2e/` (axe included)
- [ ] Benchmark at the 028 volumes recorded in `plan.md`
- [ ] `/speckit-converge` clean; feature-map status change written in the pull request
- [ ] Pull request into `measure-module`, three checks green, one approval

### Start prompt for Claude Code

```text
I'm on lane B of the Measure module, building feature 046 Performance data from a simulated feed.
1. Read docs/sdd/measure-module-build.md, then the "## #046" section of docs/sdd/measure-issues.md (the text of
   this GitHub issue), then the rows of docs/product/measure-metrics.md and the client lines in
   docs/reference/shell-measure-requirements.md it cites.
2. git switch measure-module && git pull. Run /speckit-specify with
   SPECIFY_FEATURE_DIRECTORY=specs/046-performance-data and GIT_BRANCH_NAME=feature/046-performance-data,
   using the issue's Goal, Stories and References as the feature description.
3. Read-only report: no create/edit/delete scenarios; write the state scenarios instead
   (data, empty, no access, no financial access, filters exclude everything).
4. Stop after the spec and its quality checklist so I can review. Record clarifications as 046-Cn in the
   spec, never in docs/sdd/decision-log.md. Touch only the paths this issue lists under "Owns".
```
````

### 10. `047` — 047 · Projections

- **Title:** `047 · Projections`
- **Labels:** `measure,wave-b,lane-b`
- **Milestone:** Measure wave B
- **Lane:** B · 14 h

**Body:**

````markdown
**Lane B · 14 h · wave B** — Measure module. Read [measure-module-build.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/measure-module-build.md) first.

**Goal.** Models gain velocity (months between stages) and the allowed goal stages; Settings gains activity
categories (groups of activity types used by the reports); a pure projection calculation gives projected INQ,
MQL and revenue per period from each activity's goal, forward from its start date to the plan year's end.

**Owns.** `measure/projection`, the model and category changes in `settings` and `features/settings/models/`,
`features/settings/activity-categories/`, migration 00033, `e2e/projections.spec.ts`.

**References.** FA §E3.2.6, §E7.5.1; client 020 US7, FR-041–FR-050; domain model §4.5.

**Stories.** US1 velocity and allowed stages on a model (P1) · US2 projected figures per period (P1) · US3
activity categories (P2).

### Checklist

- [ ] Branch `feature/047-projections` from `measure-module`
- [ ] `/speckit-specify` citing the client FRs and FA §E; `/speckit-clarify` answers recorded as `047-Cn`
- [ ] `/speckit-plan` (short), `contracts/api.md` handed to the integrator for the wave contract
- [ ] `/speckit-tasks`, `/speckit-analyze` with no CRITICAL issue
- [ ] `/speckit-implement` one user story at a time; `scripts/check-changed.ps1` green after each
- [ ] Scope-isolation test for every scoped store read; contract test for every operation; one test per widget state
- [ ] End-to-end test for every P1 story in `frontend/e2e/` (axe included)
- [ ] Benchmark at the 028 volumes recorded in `plan.md`
- [ ] `/speckit-converge` clean; feature-map status change written in the pull request
- [ ] Pull request into `measure-module`, three checks green, one approval

### Start prompt for Claude Code

```text
I'm on lane B of the Measure module, building feature 047 Projections.
1. Read docs/sdd/measure-module-build.md, then the "## #047" section of docs/sdd/measure-issues.md (the text of
   this GitHub issue), then the rows of docs/product/measure-metrics.md and the client lines in
   docs/reference/shell-measure-requirements.md it cites.
2. git switch measure-module && git pull. Run /speckit-specify with
   SPECIFY_FEATURE_DIRECTORY=specs/047-projections and GIT_BRANCH_NAME=feature/047-projections,
   using the issue's Goal, Stories and References as the feature description.
3. Read-only report: no create/edit/delete scenarios; write the state scenarios instead
   (data, empty, no access, no financial access, filters exclude everything).
4. Stop after the spec and its quality checklist so I can review. Record clarifications as 047-Cn in the
   spec, never in docs/sdd/decision-log.md. Touch only the paths this issue lists under "Owns".
```
````

### 11. `048` — 048 · Summary report

- **Title:** `048 · Summary report`
- **Labels:** `measure,wave-b,lane-c`
- **Milestone:** Measure wave B
- **Lane:** C · 14 h

**Body:**

````markdown
**Lane C · 14 h · wave B** — Measure module. Read [measure-module-build.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/measure-module-build.md) first.

**Goal.** Projected against goal with the variance word, the trend split at today, four league tables (top by
revenue, performance, cost, ROI for a chosen Shell attribute, each refreshing alone), stage volumes and
conversion rates against goal, and links into the Revenue and Funnel reports.

**Owns.** `measure/summary`, `app/measure_summary.go`, `features/measure/summary/`, `e2e/measure-summary.spec.ts`.

**References.** FA §E3.1; client 016 US1, US4, FR-020–FR-045; [measure-metrics.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/product/measure-metrics.md) §2.

**Stories.** US1 are we on track (P1) · US2 league tables (P2) · US3 funnel health and links (P2).

### Checklist

- [ ] Branch `feature/048-summary-report` from `measure-module`
- [ ] `/speckit-specify` citing the client FRs and FA §E; `/speckit-clarify` answers recorded as `048-Cn`
- [ ] `/speckit-plan` (short), `contracts/api.md` handed to the integrator for the wave contract
- [ ] `/speckit-tasks`, `/speckit-analyze` with no CRITICAL issue
- [ ] `/speckit-implement` one user story at a time; `scripts/check-changed.ps1` green after each
- [ ] Scope-isolation test for every scoped store read; contract test for every operation; one test per widget state
- [ ] End-to-end test for every P1 story in `frontend/e2e/` (axe included)
- [ ] Benchmark at the 028 volumes recorded in `plan.md`
- [ ] `/speckit-converge` clean; feature-map status change written in the pull request
- [ ] Pull request into `measure-module`, three checks green, one approval

### Start prompt for Claude Code

```text
I'm on lane C of the Measure module, building feature 048 Summary report.
1. Read docs/sdd/measure-module-build.md, then the "## #048" section of docs/sdd/measure-issues.md (the text of
   this GitHub issue), then the rows of docs/product/measure-metrics.md and the client lines in
   docs/reference/shell-measure-requirements.md it cites.
2. git switch measure-module && git pull. Run /speckit-specify with
   SPECIFY_FEATURE_DIRECTORY=specs/048-summary-report and GIT_BRANCH_NAME=feature/048-summary-report,
   using the issue's Goal, Stories and References as the feature description.
3. Read-only report: no create/edit/delete scenarios; write the state scenarios instead
   (data, empty, no access, no financial access, filters exclude everything).
4. Stop after the spec and its quality checklist so I can review. Record clarifications as 048-Cn in the
   spec, never in docs/sdd/decision-log.md. Touch only the paths this issue lists under "Owns".
```
````

### 12. `049` — 049 · Revenue and funnel-stage reports

- **Title:** `049 · Revenue and funnel-stage reports`
- **Labels:** `measure,wave-b,lane-b`
- **Milestone:** Measure wave B
- **Lane:** B · 14 h

**Body:**

````markdown
**Lane B · 14 h · wave B** — Measure module. Read [measure-module-build.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/measure-module-build.md) first.

**Goal.** Two reports on 040's template: summary cards per stage and activity category (plan goal, goal
coverage, activity goals, ROI), actual, projected and goal by period split at today with the data table, and the
ranked list sorted by actual revenue, projected revenue, ROI or a stage; the Funnel report chooses the stage.

**Owns.** `measure/revenue`, `app/measure_revenue.go`, `features/measure/revenue/`, `features/measure/funnel/`,
`e2e/measure-revenue.spec.ts`.

**References.** FA §E3.2, §E3.3; client 016 US3, US5, US9, FR-046–FR-084; 040's ranked list.

**Stories.** US1 rank and drill revenue (P1) · US2 a funnel stage (P2) · US3 summary cards per category (P3).

### Checklist

- [ ] Branch `feature/049-revenue-and-funnel-reports` from `measure-module`
- [ ] `/speckit-specify` citing the client FRs and FA §E; `/speckit-clarify` answers recorded as `049-Cn`
- [ ] `/speckit-plan` (short), `contracts/api.md` handed to the integrator for the wave contract
- [ ] `/speckit-tasks`, `/speckit-analyze` with no CRITICAL issue
- [ ] `/speckit-implement` one user story at a time; `scripts/check-changed.ps1` green after each
- [ ] Scope-isolation test for every scoped store read; contract test for every operation; one test per widget state
- [ ] End-to-end test for every P1 story in `frontend/e2e/` (axe included)
- [ ] Benchmark at the 028 volumes recorded in `plan.md`
- [ ] `/speckit-converge` clean; feature-map status change written in the pull request
- [ ] Pull request into `measure-module`, three checks green, one approval

### Start prompt for Claude Code

```text
I'm on lane B of the Measure module, building feature 049 Revenue and funnel-stage reports.
1. Read docs/sdd/measure-module-build.md, then the "## #049" section of docs/sdd/measure-issues.md (the text of
   this GitHub issue), then the rows of docs/product/measure-metrics.md and the client lines in
   docs/reference/shell-measure-requirements.md it cites.
2. git switch measure-module && git pull. Run /speckit-specify with
   SPECIFY_FEATURE_DIRECTORY=specs/049-revenue-and-funnel-reports and GIT_BRANCH_NAME=feature/049-revenue-and-funnel-reports,
   using the issue's Goal, Stories and References as the feature description.
3. Read-only report: no create/edit/delete scenarios; write the state scenarios instead
   (data, empty, no access, no financial access, filters exclude everything).
4. Stop after the spec and its quality checklist so I can review. Record clarifications as 049-Cn in the
   spec, never in docs/sdd/decision-log.md. Touch only the paths this issue lists under "Owns".
```
````

### 13. `050` — 050 · Attribution and ROI

- **Title:** `050 · Attribution and ROI`
- **Labels:** `measure,wave-b,lane-a`
- **Milestone:** Measure wave B
- **Lane:** A · 16 h

**Body:**

````markdown
**Lane A · 16 h · wave B** — Measure module. Read [measure-module-build.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/measure-module-build.md) first.

**Goal.** Settings › Attribution chooses the model that credits revenue facts to activities (first touch, last
touch, equally distributed, linear time decay, U-shaped — the five that need no weights); attributed revenue and
ROI appear on 048 and 049 with the model and the cost basis named next to every figure; *Use planned cost* is a
labelled choice, never silent.

**Owns.** `measure/attribution`, `app/measure_attribution.go`, migration 00034, `features/settings/attribution/`,
ROI parts in `features/measure/shared/`, `e2e/attribution.spec.ts`.

**References.** FA §E7.5; client 020 US8, FR-051–FR-065; measure-metrics §2 (ROI, confirm the definition in the spec).

**Stories.** US1 choose the model and see credited revenue (P1) · US2 ROI with its basis named (P1) · US3 change
the model and see every figure follow (P2).

### Checklist

- [ ] Branch `feature/050-attribution-and-roi` from `measure-module`
- [ ] `/speckit-specify` citing the client FRs and FA §E; `/speckit-clarify` answers recorded as `050-Cn`
- [ ] `/speckit-plan` (short), `contracts/api.md` handed to the integrator for the wave contract
- [ ] `/speckit-tasks`, `/speckit-analyze` with no CRITICAL issue
- [ ] `/speckit-implement` one user story at a time; `scripts/check-changed.ps1` green after each
- [ ] Scope-isolation test for every scoped store read; contract test for every operation; one test per widget state
- [ ] End-to-end test for every P1 story in `frontend/e2e/` (axe included)
- [ ] Benchmark at the 028 volumes recorded in `plan.md`
- [ ] `/speckit-converge` clean; feature-map status change written in the pull request
- [ ] Pull request into `measure-module`, three checks green, one approval

### Start prompt for Claude Code

```text
I'm on lane A of the Measure module, building feature 050 Attribution and ROI.
1. Read docs/sdd/measure-module-build.md, then the "## #050" section of docs/sdd/measure-issues.md (the text of
   this GitHub issue), then the rows of docs/product/measure-metrics.md and the client lines in
   docs/reference/shell-measure-requirements.md it cites.
2. git switch measure-module && git pull. Run /speckit-specify with
   SPECIFY_FEATURE_DIRECTORY=specs/050-attribution-and-roi and GIT_BRANCH_NAME=feature/050-attribution-and-roi,
   using the issue's Goal, Stories and References as the feature description.
3. Read-only report: no create/edit/delete scenarios; write the state scenarios instead
   (data, empty, no access, no financial access, filters exclude everything).
4. Stop after the spec and its quality checklist so I can review. Record clarifications as 050-Cn in the
   spec, never in docs/sdd/decision-log.md. Touch only the paths this issue lists under "Owns".
```
````

### 14. `MA` — Measure · Wave A integration

- **Title:** `Measure · Wave A integration`
- **Labels:** `measure,wave-a,lane-a`
- **Milestone:** Measure wave A
- **Lane:** A · 6 h

**Body:**

````markdown
**Lane A · 6 h · wave A** — Measure module. Read [measure-module-build.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/measure-module-build.md) first.

- [ ] Merge 040–045 into `measure-module`; apply reported contract fixes; `pnpm run api:types`
- [ ] Move the lanes' `0NN-Cn` clarifications into the decision log (D-371 onwards); apply feature-map statuses
- [ ] Full check on `measure-module`: backend, frontend and e2e green; pull request `measure-module` → `develop`
````

### 15. `MB` — Measure · Wave B integration

- **Title:** `Measure · Wave B integration`
- **Labels:** `measure,wave-b,lane-b`
- **Milestone:** Measure wave B
- **Lane:** B · 6 h

**Body:**

````markdown
**Lane B · 6 h · wave B** — Measure module. Read [measure-module-build.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/measure-module-build.md) first.

- [ ] Wave B contract written before 048–050 start; merge 046–050; decisions and statuses moved as in #MA
- [ ] Pull request `measure-module` → `develop`
````

### 16. `QA-A` — Measure · Wave A QA pass

- **Title:** `Measure · Wave A QA pass`
- **Labels:** `measure,wave-a,lane-c`
- **Milestone:** Measure wave A
- **Lane:** C · 8 h

**Body:**

````markdown
**Lane C · 8 h · wave A** — Measure module. Read [measure-module-build.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/measure-module-build.md) first.

- [ ] Browser QA of every Reports screen on the demo data under four roles (every scope, no financial access)
- [ ] Findings as issues on the owning feature; the critical and high ones fixed before the wave merges
````

### 17. `QA-B` — Measure · Wave B QA pass

- **Title:** `Measure · Wave B QA pass`
- **Labels:** `measure,wave-b,lane-d`
- **Milestone:** Measure wave B
- **Lane:** D · 8 h

**Body:**

````markdown
**Lane D · 8 h · wave B** — Measure module. Read [measure-module-build.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/measure-module-build.md) first.

- [ ] As the wave A QA pass, for the Summary, Revenue, Funnel, Performance data and Attribution screens
````

### 18. `DEMO` — Measure · Demo data and end-to-end fixtures

- **Title:** `Measure · Demo data and end-to-end fixtures`
- **Labels:** `measure,wave-a,lane-d`
- **Milestone:** Measure wave A
- **Lane:** D · 8 h

**Body:**

````markdown
**Lane D · 8 h · wave A** — Measure module. Read [measure-module-build.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/measure-module-build.md) first.

- [ ] `seed/measure.go` behind the call 039 creates: goals on the demo activities, dates and spreads that fill
      every period, approvals and purchase requests in every status, a routed request without a GSAP number, a
      line item waiting for an SWBS code
- [ ] `docs/product/demo-users-and-data.md` gains the Measure story; e2e fixtures in `e2e/measure.ts`
````

## Lane trackers (create last)

Replace every `#ISSUE-<key>` with the issue number you noted. Labels as given; no milestone.

### Lane A — Measure lane A · Foundation, plan budget, attribution (66 h)

- **Title:** `Measure lane A · Foundation, plan budget, attribution (66 h)`
- **Labels:** `measure,tracking,lane-a`
- **Milestone:** none

**Body:**

````markdown
**Lane A — Foundation, plan budget, attribution · 66 h.** Pick this lane, assign yourself to this issue and to the issues below.

Read [measure-module-build.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/measure-module-build.md) first — §2 lists your features, §4 the files you own and the files you never edit, §8 the cycle.

### Your work, in order

- [ ] #ISSUE-M0 Measure · Kick-off: check script and Claude Code hook (8 h)
- [ ] #ISSUE-039 039 · Measure foundation (28 h)
- [ ] #ISSUE-040 040 · Plan budget report (16 h)
- [ ] #ISSUE-050 050 · Attribution and ROI (16 h)
- [ ] #ISSUE-MA Measure · Wave A integration (6 h)

### Timing (hours from the start of wave A)

1. **Hour 0–4:** specify 039 (`/speckit-specify`, then `/speckit-clarify`).
2. **By hour 8:** collect `contracts/api.md` from lanes C and D (041–045) and your own 040, and write the **wave A contract** into `backend/api/openapi.yaml` on the 039 branch.
3. **By hour 16:** 039 backend first — `measure` package (filter model, periods, registry), `access/routes_measure.go` with every wave A rule, `app/measure.go` with one stub function per report, migration 00028 — merged into `measure-module` so the other lanes can import it.
4. **By hour 28:** 039's web frame, `features/measure/shared/` and the `features/measure/index.ts` stubs.
5. Then 040 (it exports `RankedList` for lane B's 049), then wave A integration, then 050 once 048 and 049 exist.

### The cycle, one prompt per step

1. **Start** — paste the *Start prompt* at the bottom of the feature's issue.
2. `/speckit-clarify` — at most five questions; answers as `0NN-Cn` in the spec's Clarifications.
3. `/speckit-plan` — short plan: rules and the tests that prove them, queries and their scope filter, permissions, benchmark; `contracts/api.md` for the wave contract; `data-model.md` only for a new table.
4. `/speckit-tasks`, then `/speckit-analyze` — fix every CRITICAL finding before implementing.
5. `/speckit-implement` **one user story per prompt**, e.g. "implement US1 of {feature} only, tests first, then stop".
6. After each story: the checks of [quality-gates.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/quality-gates.md) on what you changed (`scripts/check-changed.ps1` once it exists).
7. End-to-end test for every P1 story, `/speckit-converge`, then the pull request into `measure-module`.

### Rules that keep four lanes conflict-free

- Change only the paths your feature issue lists under **Owns**.
- Never edit `openapi.yaml`, `routes_measure.go`, `app/measure.go`, `app.go`, `navigation.ts`, `pages.tsx`, `router.tsx`, `features/measure/index.ts`, `seed/seed.go`, `feature-map.md` or `decision-log.md`: report what you need in your pull request.
- Rebase on `measure-module` before every pull request; run `sqlc generate` and `pnpm run api:types` after rebasing.
````

### Lane B — Measure lane B · Performance data, projections, revenue (64 h)

- **Title:** `Measure lane B · Performance data, projections, revenue (64 h)`
- **Labels:** `measure,tracking,lane-b`
- **Milestone:** none

**Body:**

````markdown
**Lane B — Performance data, projections, revenue · 64 h.** Pick this lane, assign yourself to this issue and to the issues below.

Read [measure-module-build.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/measure-module-build.md) first — §2 lists your features, §4 the files you own and the files you never edit, §8 the cycle.

### Your work, in order

- [ ] #ISSUE-046 046 · Performance data from a simulated feed (30 h)
- [ ] #ISSUE-047 047 · Projections (14 h)
- [ ] #ISSUE-049 049 · Revenue and funnel-stage reports (14 h)
- [ ] #ISSUE-MB Measure · Wave B integration (6 h)

### Timing (hours from the start of wave A)

1. **Hour 0:** start 046 at once — it needs nothing from 039. Its plan writes ADR-0019 (simulated performance feed).
2. Seed the wave B performance facts in `seed/measure.go`'s wave B part early, so lanes A and C can build on them.
3. Then 047; with it, write the **wave B contract** into `openapi.yaml` before 048–050 start.
4. Then 049 on lane A's `RankedList` (040), then wave B integration.

### The cycle, one prompt per step

1. **Start** — paste the *Start prompt* at the bottom of the feature's issue.
2. `/speckit-clarify` — at most five questions; answers as `0NN-Cn` in the spec's Clarifications.
3. `/speckit-plan` — short plan: rules and the tests that prove them, queries and their scope filter, permissions, benchmark; `contracts/api.md` for the wave contract; `data-model.md` only for a new table.
4. `/speckit-tasks`, then `/speckit-analyze` — fix every CRITICAL finding before implementing.
5. `/speckit-implement` **one user story per prompt**, e.g. "implement US1 of {feature} only, tests first, then stop".
6. After each story: the checks of [quality-gates.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/quality-gates.md) on what you changed (`scripts/check-changed.ps1` once it exists).
7. End-to-end test for every P1 story, `/speckit-converge`, then the pull request into `measure-module`.

### Rules that keep four lanes conflict-free

- Change only the paths your feature issue lists under **Owns**.
- Never edit `openapi.yaml`, `routes_measure.go`, `app/measure.go`, `app.go`, `navigation.ts`, `pages.tsx`, `router.tsx`, `features/measure/index.ts`, `seed/seed.go`, `feature-map.md` or `decision-log.md`: report what you need in your pull request.
- Rebase on `measure-module` before every pull request; run `sqlc generate` and `pnpm run api:types` after rebasing.
````

### Lane C — Measure lane C · Approvals, purchase requests, summary (46 h)

- **Title:** `Measure lane C · Approvals, purchase requests, summary (46 h)`
- **Labels:** `measure,tracking,lane-c`
- **Milestone:** none

**Body:**

````markdown
**Lane C — Approvals, purchase requests, summary · 46 h.** Pick this lane, assign yourself to this issue and to the issues below.

Read [measure-module-build.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/measure-module-build.md) first — §2 lists your features, §4 the files you own and the files you never edit, §8 the cycle.

### Your work, in order

- [ ] #ISSUE-041 041 · Approvals report (10 h)
- [ ] #ISSUE-042 042 · Purchase request report (14 h)
- [ ] #ISSUE-048 048 · Summary report (14 h)
- [ ] #ISSUE-QA-A Measure · Wave A QA pass (8 h)

### Timing (hours from the start of wave A)

1. **By hour 8:** specify, clarify and plan 041 and 042; push the branches so lane A can read each `contracts/api.md` for the wave A contract.
2. **From hour 16:** backends against the contract and 039's `measure` types; **from hour 28:** the web parts inside 039's report frame.
3. Then the wave A QA pass, then 048 once the wave B contract is written.

### The cycle, one prompt per step

1. **Start** — paste the *Start prompt* at the bottom of the feature's issue.
2. `/speckit-clarify` — at most five questions; answers as `0NN-Cn` in the spec's Clarifications.
3. `/speckit-plan` — short plan: rules and the tests that prove them, queries and their scope filter, permissions, benchmark; `contracts/api.md` for the wave contract; `data-model.md` only for a new table.
4. `/speckit-tasks`, then `/speckit-analyze` — fix every CRITICAL finding before implementing.
5. `/speckit-implement` **one user story per prompt**, e.g. "implement US1 of {feature} only, tests first, then stop".
6. After each story: the checks of [quality-gates.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/quality-gates.md) on what you changed (`scripts/check-changed.ps1` once it exists).
7. End-to-end test for every P1 story, `/speckit-converge`, then the pull request into `measure-module`.

### Rules that keep four lanes conflict-free

- Change only the paths your feature issue lists under **Owns**.
- Never edit `openapi.yaml`, `routes_measure.go`, `app/measure.go`, `app.go`, `navigation.ts`, `pages.tsx`, `router.tsx`, `features/measure/index.ts`, `seed/seed.go`, `feature-map.md` or `decision-log.md`: report what you need in your pull request.
- Rebase on `measure-module` before every pull request; run `sqlc generate` and `pnpm run api:types` after rebasing.
````

### Lane D — Measure lane D · Plan details, GSAP link, user access (42 h)

- **Title:** `Measure lane D · Plan details, GSAP link, user access (42 h)`
- **Labels:** `measure,tracking,lane-d`
- **Milestone:** none

**Body:**

````markdown
**Lane D — Plan details, GSAP link, user access · 42 h.** Pick this lane, assign yourself to this issue and to the issues below.

Read [measure-module-build.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/measure-module-build.md) first — §2 lists your features, §4 the files you own and the files you never edit, §8 the cycle.

### Your work, in order

- [ ] #ISSUE-043 043 · Plan details report (10 h)
- [ ] #ISSUE-044 044 · GSAP link reports (10 h)
- [ ] #ISSUE-045 045 · User access report (6 h)
- [ ] #ISSUE-QA-B Measure · Wave B QA pass (8 h)
- [ ] #ISSUE-DEMO Measure · Demo data and end-to-end fixtures (8 h)

### Timing (hours from the start of wave A)

1. **By hour 8:** specify, clarify and plan 043, 044 and 045; push the branches so lane A can read each `contracts/api.md` for the wave A contract.
2. **From hour 16:** backends against the contract and 039's `measure` types; **from hour 28:** the web parts.
3. The demo data (`seed/measure.go` behind the call 039 creates) alongside, then the wave B QA pass at the end.

### The cycle, one prompt per step

1. **Start** — paste the *Start prompt* at the bottom of the feature's issue.
2. `/speckit-clarify` — at most five questions; answers as `0NN-Cn` in the spec's Clarifications.
3. `/speckit-plan` — short plan: rules and the tests that prove them, queries and their scope filter, permissions, benchmark; `contracts/api.md` for the wave contract; `data-model.md` only for a new table.
4. `/speckit-tasks`, then `/speckit-analyze` — fix every CRITICAL finding before implementing.
5. `/speckit-implement` **one user story per prompt**, e.g. "implement US1 of {feature} only, tests first, then stop".
6. After each story: the checks of [quality-gates.md](https://github.com/anup-zensar/MAPP/blob/measure-module/docs/sdd/quality-gates.md) on what you changed (`scripts/check-changed.ps1` once it exists).
7. End-to-end test for every P1 story, `/speckit-converge`, then the pull request into `measure-module`.

### Rules that keep four lanes conflict-free

- Change only the paths your feature issue lists under **Owns**.
- Never edit `openapi.yaml`, `routes_measure.go`, `app/measure.go`, `app.go`, `navigation.ts`, `pages.tsx`, `router.tsx`, `features/measure/index.ts`, `seed/seed.go`, `feature-map.md` or `decision-log.md`: report what you need in your pull request.
- Rebase on `measure-module` before every pull request; run `sqlc generate` and `pnpm run api:types` after rebasing.
````
