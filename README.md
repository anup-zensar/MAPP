# MAPP

**Marketing planning for Shell — simple, clear, step by step.**

MAPP is Shell's marketing planning application. It replaces Hive9, starting with the **Plan
module**: marketers build a **plan**, break it into **campaigns**, **initiatives** and **activities**,
plan every cost as **line items**, and send activities for **approval** — always seeing the status of
everything and what is still missing.

It is built with **spec-driven development** (GitHub Spec Kit): every feature is first described in a
spec, agreed, planned, broken into tasks, and only then built and tested.

The requirements and features were collected from **the available documentation** — Shell's Hive9 Plan
Module document, Shell's Business Requirements, the GSAP workshop and the Hive9 user guide — and from
**the Hive9 application itself**, studied in Shell's Hive9 sandbox (see
[docs/reference/](docs/reference/README.md)). Hive9's screens are not copied: MAPP has its own simple
design, in Shell's enterprise design language.

| | |
|---|---|
| API | Go 1.27, standard library only at run time (`backend/`) |
| Web app | React 19 + TypeScript (`frontend/`) |
| Data | PostgreSQL 18 on your own computer: a native install, or the optional `compose.yaml` |
| Sign-in | Demo sign-in now (password `mapp`), Shell single sign-on later |
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

All demo people share the password **`mapp`**. Roles come from the demo data (feature 014) and can be
changed in **Settings › Roles and access** (Administrators only).

| Person | E-mail | Role (scope) | Good for showing |
|---|---|---|---|
| Alex Morgan | alex.morgan@shell.com | Administrator (all markets) | Settings, Roles and access, approval level 2, deciding on behalf, GSAP imports |
| Carlos Mendes | carlos.mendes@shell.com | Global planner (all markets) | Global budgets and allocations, Mobility approvals (level 1) |
| Daniel Okoro | daniel.okoro@shell.com | Planner and Approver (own teams: Lubricants Europe) | Approving Lubricants activities |
| Priya Raman | priya.raman@shell.com | Planner (own teams: Lubricants Europe) | Building the Lubricants plan, submitting for approval |
| Mei Lin | mei.lin@shell.com | Planner (own teams: Mobility China) | Building the Mobility plan |
| Sara Novak | sara.novak@shell.com | Viewer (own teams: Lubricants Europe, Mobility China) | View only: sees plans and financial data, changes nothing |

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
| Node.js | 22.22 or later (24 LTS recommended) | `node --version` |
| pnpm | the version in `frontend/package.json` (`packageManager`), through Corepack | `pnpm --version` |
| PostgreSQL | 18, native install (or Docker with `docker compose up -d postgres`) | `pg_isready -h 127.0.0.1` |

If `node` is not found after installing, open a new terminal; for Node.js unpacked from a zip file, add
its folder to `PATH`.

pnpm comes through Corepack, which ships with Node.js 24: run `corepack enable` once (on Windows, in a
terminal opened as administrator). Without administrator rights, install pnpm for your user instead with
`Invoke-WebRequest https://get.pnpm.io/install.ps1 -UseBasicParsing | Invoke-Expression` (PowerShell) or
`brew install pnpm` (macOS); either way pnpm switches itself to the version `package.json` names.

## Run MAPP

Use two terminals: one for the API, one for the web app.

**0. PostgreSQL** — on `127.0.0.1:5432`, with a role `mapp` that may create databases. With Docker,
`docker compose up -d postgres` sets that up; with a native install, create the role once
(`createuser --createdb mapp`). The API and the seed read the database from `DATABASE_URL`, by default
`postgres://mapp@127.0.0.1:5432/mapp?sslmode=disable`.

**1. The API** — <http://127.0.0.1:8080>

```powershell
cd backend
go run ./cmd/seed          # creates and migrates the database, then loads the demo data (people, roles, plans, approvals, GSAP data)
go run ./cmd/api           # logs {"msg":"MAPP API listening","url":"http://127.0.0.1:8080",…}
```

**Stop the API (Ctrl+C) before you run the seed again.** To start over from the demo data:
`go run ./cmd/seed -reset` (this deletes the current data). After pulling new migrations, run
`go run ./cmd/seed` once: it applies them, and the API refuses to start on a database behind them.

**2. The web app** — <http://127.0.0.1:5173>

```powershell
cd frontend
pnpm install               # first time only
pnpm run dev
```

If PowerShell says "running scripts is disabled on this system", type `pnpm.cmd` instead of `pnpm`
(for example `pnpm.cmd run dev`); it does the same and needs no change to the computer's settings.

Open <http://127.0.0.1:5173> and click a demo person, or type a demo e-mail with the password
**`mapp`**.

**Port 8080 already in use?** The API then stops with "Port 8080 is already in use — stop the other
program or set APP_PORT." Either stop the other program, or use another port in both terminals:

```powershell
$env:APP_PORT = '18081'                               # API terminal, before go run ./cmd/seed and ./cmd/api
$env:MAPP_API_URL = 'http://127.0.0.1:18081'     # web-app terminal, before ppnpm run dev
```

## Run MAPP in containers (optional)

For reviewers who don't develop MAPP, such as business analysts: install Podman Desktop (or Docker), get
the repository with Git, then from its folder:

```powershell
podman compose up -d --build   # first time and after every git pull; the first build takes a few minutes
```

Open <http://127.0.0.1:3000>. The demo data is loaded on the first start and kept after that;
`podman compose down -v` deletes everything, and the next start loads the demo data again. `docker compose`
works the same way. See ADR-0012.

## Test MAPP

```powershell
cd backend
gofmt -l .                 # prints nothing when the formatting is right
go vet ./...
go test -short ./...       # unit and contract tests; no database needed
go test ./...              # also the scope-isolation tests, on the PostgreSQL server of TEST_DATABASE_URL

cd ../frontend
pnpm run check             # API types up to date, type check, lint, format, unit and component tests
pnpm run test:e2e          # browser tests; they start their own API (port 18080) and web app (port 14173)
```

The end-to-end tests use their own database, `mapp_e2e` (`E2E_DATABASE_URL` moves it), which the seed
empties and reloads on every run; your own database is never touched. Set `TEST_DATABASE_URL`, for example
to `postgres://mapp@127.0.0.1:5432/postgres?sslmode=disable`, before `go test ./...`.

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
