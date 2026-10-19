# NorthStar

**Marketing planning for Shell — simple, clear, step by step.**

NorthStar is Shell's marketing planning application. It replaces Hive9, starting with the **Plan
module**: marketers build a **plan**, break it into **campaigns**, **initiatives** and **activities**,
plan every cost as **line items**, and send activities for **approval** — always seeing the status of
everything and what is still missing.

It is built with **spec-driven development** (GitHub Spec Kit): every feature is first described in a
spec, agreed, planned, broken into tasks, and only then built and tested.

| | |
|---|---|
| API | Go 1.27, standard library only at run time (`backend/`) |
| Web app | React 19 + TypeScript (`frontend/`) |
| Data | Local JSON file store now, PostgreSQL later |
| Sign-in | Demo sign-in now (password `northstar`), Shell single sign-on later |
| Design | ZenReach design language (`docs/product/design-system.md`) |
| Phase | **Plan module** — see `docs/sdd/feature-map.md` |

Everything runs on your own computer and listens on `127.0.0.1` only — no cloud services.

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
go run ./cmd/seed          # once: loads the six demo people into backend/data
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
   so far implements [specs/001-app-shell-and-sign-in/](specs/001-app-shell-and-sign-in/); its
   [quickstart.md](specs/001-app-shell-and-sign-in/quickstart.md) walks through every screen
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
