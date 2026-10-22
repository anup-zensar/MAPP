# NorthStar

**Marketing planning for Shell — simple, clear, step by step.**

NorthStar is Shell's marketing planning application. It replaces Hive9, starting with the **Plan
module**: marketers build a **plan**, break it into **campaigns**, **initiatives** and **activities**,
plan every cost as **line items**, and send activities for **approval** — always seeing the status of
everything and what is still missing.

It is built with **spec-driven development** (GitHub Spec Kit): every feature is first described in a
spec, agreed, planned, broken into tasks, and only then built and tested.

The requirements and features were collected from **the available documentation** — Shell's Hive9 Plan
Module document, Shell's Business Requirements, the GSAP workshop and the Hive9 user guide — and from
**the Hive9 application itself**, studied in Shell's Hive9 sandbox (see
[docs/reference/](docs/reference/README.md)). Hive9's screens are not copied: NorthStar has its own simple
design, in Shell's enterprise design language.

| | |
|---|---|
| API | Go 1.27, standard library only at run time (`backend/`) |
| Web app | React 19 + TypeScript (`frontend/`) |
| Data | Local JSON file store now, PostgreSQL later |
| Sign-in | Demo sign-in now (password `northstar`), Shell single sign-on later |
| Design | Shell's enterprise design language, design system v2 (`docs/product/design-system.md`) |
| Phase | **Plan module** — see `docs/sdd/feature-map.md` |

Everything runs on your own computer and listens on `127.0.0.1` only — no cloud services.

## What the Plan module does

| Area | Feature | What a person can do |
|---|---|---|
| Sign-in, Home | 001, 009 | Demo sign-in; Home with summary cards, plans at a glance, needs attention, waiting for my approval, a Financials card and recent changes |
| Plans and planning records | 004–007 | Plans, campaigns, initiatives, activities and line items: create, edit, copy (also into next year), move, delete with Undo; status and completeness everywhere |
| Approvals | 008, 017 | Approval workflows with levels of named people, the owner's line manager or a role; any or all must approve; conditional and automatic levels; backup approvers; delegation; reminders and escalation (**Run reminders and escalations**); a Monitoring tab |
| All work | 010 | One grid over all plans with roll-ups, filters (years range, plan type, plan currency, Financials), calendar, Budget & cost (Monthly, Quarterly, Yearly, compare with phased budget), saved views, export |
| Activity log, Recycle bin | 011 | Who changed what; a History section on every record; restore or delete permanently |
| Roles and access, Global budgets | 014 | Five roles with team scopes, view-only vs edit enforced by the API, segregation of duties; global budgets allocated to market plans |
| Financials | 015 | GSAP purchase orders, invoices, payments and accruals (read-only, from a **simulated** GSAP extract) on the activity, line item, plan, campaign, initiative, All work, Home and a Financials page; import and manual linking |
| Planning and currency | 016 | Dated exchange rates and a **simulated** GSAP rate file import, plan currency, budget by month, split activity, plan type, multi-year views, Settings › Field help |
| Settings, People | 002, 003, 014, 016 | Models, activity types, Shell attributes, currencies and rates, finance budget items, people and teams, approval workflows, roles and access, field help |

What is still open (screens for attribute filters and Breakdown, field-help tips on record screens, hiding
actions a role does not allow, the end-to-end journeys of 014–017) is listed in the
[final validation report](docs/reports/final-validation-report.md) §7 and at the top of each feature's `tasks.md`.

## Demo people and their roles

All demo people share the password **`northstar`**. Roles come from the demo data (feature 014) and can be
changed in **Settings › Roles and access** (Administrators only).

| Person | E-mail | Role (scope) | Good for showing |
|---|---|---|---|
| Alex Morgan | alex.morgan@northstar.demo | Administrator (all markets) | Settings, Roles and access, approval level 2, deciding on behalf, GSAP imports |
| Carlos Mendes | carlos.mendes@northstar.demo | Global planner (all markets) | Global budgets and allocations, Mobility approvals (level 1) |
| Daniel Okoro | daniel.okoro@northstar.demo | Planner and Approver (own teams: Lubricants Europe) | Approving Lubricants activities |
| Priya Raman | priya.raman@northstar.demo | Planner (own teams: Lubricants Europe) | Building the Lubricants plan, submitting for approval |
| Mei Lin | mei.lin@northstar.demo | Planner (own teams: Mobility China) | Building the Mobility plan |
| Sara Novak | sara.novak@northstar.demo | Viewer (own teams: Lubricants Europe, Mobility China) | View only: sees plans and financial data, changes nothing |

| Role | May |
|---|---|
| Administrator | Everything, including Settings, roles and access, deleting plans, GSAP imports and deciding approvals on behalf of others |
| Global planner | Plan in every market, publish and delete plans, link GSAP transactions, decide approvals, manage global budgets |
| Planner | Plan in their own markets (create, edit, publish plans; edit and delete campaigns and activities; submit), link GSAP transactions |
| Approver | View plans of their markets, export, see financial data, decide approvals they are named on |
| Viewer | View plans of their markets, export, see financial data |

Nobody approves an activity they own or submitted (segregation of duties, on by default). New people get the
Viewer role for their own teams. Details: [014 spec](specs/014-access-control/spec.md) Table B and
[demo users and data](docs/product/demo-users-and-data.md).

## Prerequisites

| Tool | Version | Check |
|---|---|---|
| Go | 1.27 | `go version` |
| Node.js | 22.22 or later, with npm (24 LTS recommended) | `node --version` |

If `node` is not found after installing, open a new terminal; for Node.js unpacked from a zip file, add
its folder to `PATH`.

## Run NorthStar

Use two terminals: one for the API, one for the web app.

**1. The API** — <http://127.0.0.1:8080>

```powershell
cd backend
go run ./cmd/seed          # once: loads the demo data (people, roles, plans, approvals, GSAP data) into backend/data
go run ./cmd/api           # prints "NorthStar API listening on http://127.0.0.1:8080 …"
```

The API reads its data only when it starts, so **stop the API (Ctrl+C) before you run the seed again**.
To start over from the demo data: `go run ./cmd/seed -reset` (this deletes the current data).

**2. The web app** — <http://127.0.0.1:5173>

```powershell
cd frontend
npm install                # first time only
npm run dev
```

If PowerShell says "running scripts is disabled on this system", type `npm.cmd` instead of `npm`
(for example `npm.cmd run dev`); it does the same and needs no change to the computer's settings.

Open <http://127.0.0.1:5173> and click a demo person, or type a demo e-mail with the password
**`northstar`**.

**Port 8080 already in use?** The API then stops with "Port 8080 is already in use — stop the other
program or set APP_PORT." Either stop the other program, or use another port in both terminals:

```powershell
$env:APP_PORT = '18081'                               # API terminal, before go run ./cmd/seed and ./cmd/api
$env:NORTHSTAR_API_URL = 'http://127.0.0.1:18081'     # web-app terminal, before npm run dev
```

## Test NorthStar

```powershell
cd backend
gofmt -l .                 # prints nothing when the formatting is right
go vet ./...
go test ./...              # unit and contract tests

cd ../frontend
npm run check              # API types up to date, type check, lint, format, unit and component tests
npm run test:e2e           # browser tests; they start their own API (port 18080) and web app (port 14173)
```

The end-to-end tests use their own temporary data and never touch `backend/data`.

## Developer pages

Two pages that are not in the navigation:

- <http://127.0.0.1:5173/dev/sample-records> — try create, edit, delete, Undo, restore and the activity
  log on sample records.
- <http://127.0.0.1:5173/dev/components> — every shared building block in all its variants (switch to
  dark mode to check both themes).

Like demo sign-in, they exist only when `APP_ENV` is `local` (the default) or `development`.

## Where to read more

1. **What and why:** [docs/product/vision-and-scope.md](docs/product/vision-and-scope.md)
2. **The rules:** [.specify/memory/constitution.md](.specify/memory/constitution.md)
3. **The features and their order:** [docs/sdd/feature-map.md](docs/sdd/feature-map.md)
4. **The features themselves:** [specs/](specs/) — one folder per feature (spec → plan → tasks). The code
   implements 001–011 and 014–017 (012 bulk actions and import is deferred; 013 waits on Shell); the
   [final validation report](docs/reports/final-validation-report.md) checks it against Shell's MVP
   requirements
5. **How we work:** [docs/sdd/sdd-process.md](docs/sdd/sdd-process.md)
6. **The business model:** [docs/product/domain-model.md](docs/product/domain-model.md) and
   [docs/product/field-catalogue.md](docs/product/field-catalogue.md)
7. **How screens look:** [docs/product/design-system.md](docs/product/design-system.md)
8. **Architecture:** [docs/architecture/overview.md](docs/architecture/overview.md)
9. **Working in the code:** [backend/README.md](backend/README.md) and [frontend/README.md](frontend/README.md);
   guidance for AI coding agents in [CLAUDE.md](CLAUDE.md)

## Repository layout

```
.specify/        Spec Kit: constitution, templates, scripts
.claude/skills/  Spec Kit commands for Claude Code (/speckit-specify, /speckit-plan, …)
docs/            product, architecture, SDD process, reference material
specs/           one folder per feature (spec → plan → tasks)
backend/         Go API
frontend/        React + TypeScript web app
CLAUDE.md        guidance for AI coding agents
```

Every folder in detail: [docs/architecture/repository-layout.md](docs/architecture/repository-layout.md).
