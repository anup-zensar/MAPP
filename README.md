# NorthStar

**Marketing planning for Shell — simple, clear, step by step.**

NorthStar replaces the Plan module of Hive9. Marketers build a **plan**, break it into **campaigns**,
**initiatives** and **activities**, plan every cost as **line items**, and send activities for
**approval** — always seeing the status of everything and what is still missing.

It is built with **spec-driven development** (GitHub Spec Kit): every feature is first described in a
spec, agreed, planned, broken into tasks, and only then built and tested.

| | |
|---|---|
| Web app | React + TypeScript (`frontend/`) |
| API | Go (`backend/`) |
| Data | Local JSON file store now, PostgreSQL later |
| Sign-in | Demo sign-in now (password `northstar`), Shell single sign-on later |
| Design | ZenReach design language (`docs/product/design-system.md`) |
| Phase | **Plan module** — see `docs/sdd/feature-map.md` |

## Where to start

1. **What and why:** [docs/product/vision-and-scope.md](docs/product/vision-and-scope.md)
2. **The rules:** [.specify/memory/constitution.md](.specify/memory/constitution.md)
3. **The features and their order:** [docs/sdd/feature-map.md](docs/sdd/feature-map.md)
4. **How we work:** [docs/sdd/sdd-process.md](docs/sdd/sdd-process.md)
5. **The business model:** [docs/product/domain-model.md](docs/product/domain-model.md) and
   [docs/product/field-catalogue.md](docs/product/field-catalogue.md)
6. **How screens look:** [docs/product/design-system.md](docs/product/design-system.md)
7. **Architecture:** [docs/architecture/overview.md](docs/architecture/overview.md)

## Repository layout

```
.specify/        Spec Kit: constitution, templates, scripts
.claude/skills/  Spec Kit commands for Claude Code (/speckit-specify, /speckit-plan, …)
docs/            product, architecture, SDD process, reference material
specs/           one folder per feature (spec → plan → tasks)
frontend/        React + TypeScript web app   (created by feature 001)
backend/         Go API                       (created by feature 001)
```

## Running NorthStar

Not yet — the code is created by feature **001-app-shell-and-sign-in**. Its `quickstart.md` will contain
the exact commands (prerequisites: Go 1.22+, Node.js 20+; everything runs on `127.0.0.1`, no cloud
services).
