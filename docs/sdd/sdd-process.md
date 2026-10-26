# How We Work — Spec-Driven Development in MAPP

A step-by-step guide for anyone (developer, analyst or AI agent) working on MAPP with GitHub Spec
Kit. The rules behind it are in `.specify/memory/constitution.md`.

## 1. The idea in one paragraph

We first write down **what** a feature must do and **why** (`spec.md`), agree it with the product owner,
then decide **how** to build it (`plan.md`), break it into **tasks** (`tasks.md`), check everything is
consistent, and only then write code and tests. The spec is the source of truth: if the behaviour must
change, the spec changes first.

## 2. Where things are

| You need | Look in |
|---|---|
| The rules | `.specify/memory/constitution.md` |
| The list of features and their order | `docs/sdd/feature-map.md` |
| Decisions and open questions | `docs/sdd/decision-log.md` |
| What the records, statuses and money rules are | `docs/product/domain-model.md` |
| Every field per level | `docs/product/field-catalogue.md` |
| How screens look and behave | `docs/product/design-system.md` |
| Words | `docs/product/glossary.md` |
| Architecture and folders | `docs/architecture/` |
| Source material (read-only) | `docs/reference/` |
| A feature | `specs/NNN-short-name/` |

## 3. The flow for one feature

The baseline `spec.md` of every feature in this phase already exists on `main` (decision D-021). To
build a feature:

| Step | What you do | Command (Claude Code) | Result |
|---|---|---|---|
| 0 | Create the feature branch with the same name as the folder | `git switch -c 004-plans` | Branch `004-plans` |
| 1 | Review the spec with the product owner; update it if needed | `/speckit-specify` (only for new features) | `spec.md` agreed |
| 2 | Resolve open questions | `/speckit-clarify` | Answers in the spec's *Clarifications* section |
| 3 | Design the solution | `/speckit-plan` | `plan.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md` |
| 4 | Optional quality check of requirements | `/speckit-checklist` | `checklists/*.md` |
| 5 | Break into tasks (tests first) | `/speckit-tasks` | `tasks.md` |
| 6 | Check spec, plan and tasks agree | `/speckit-analyze` | Report with no CRITICAL issues |
| 7 | Build it | `/speckit-implement` | Code and tests; tasks ticked |
| 8 | Check nothing was missed | `/speckit-converge` | Remaining work appended to `tasks.md` or "clean" |
| 9 | Demo with seed data, update the feature map status, merge to `main` | — | Feature **Done** |

Tip: when running a command for an existing spec on its branch, Spec Kit finds the folder from
`.specify/feature.json` or the branch name. Set it with
`{"feature_directory": "specs/004-plans"}` in `.specify/feature.json` if needed.

## 4. Writing a good spec (checklist)

- User stories are in priority order (P1 first) and each can be built and demonstrated on its own.
- Every story has acceptance scenarios in **Given / When / Then** form.
- Every user-managed record has create, view, edit and delete scenarios (the standard CRUD pattern).
- Every record shows its status and completeness; every locked state says why.
- Requirements are numbered `FR-001…`, testable, and use the field catalogue and domain model by
  reference (e.g. "as defined in domain model §4.3").
- Success criteria are measurable and technology-free; at least one is a usability criterion from the
  design system §10.
- At most three `[NEEDS CLARIFICATION]` markers; everything else is a documented assumption.
- No technology words (React, Go, JSON, endpoint, table) in `spec.md`.

## 5. Writing a good plan

- Run the Constitution Check (the plan template has the section).
- Reuse shared building blocks from feature 001 — new shared components need a reason.
- Add the feature's endpoints to `backend/api/openapi.yaml`; list the shared files the feature changes.
- List every business rule and the unit test that proves it.

## 6. Branches, commits and reviews

- Branches (constitution 3.x): `main` (production), `release`, `develop` (integration) and the
  module branch `plan-module`. A feature branch is `feature/NNN-short-name`, created by Spec Kit
  from `develop` or the module branch; work without a spec uses `docs/…` or `fix/…`. Nobody pushes
  directly to `main`, `release` or `develop`: changes arrive by pull request, approved by someone
  other than the author, with the three checks green ([quality gates](quality-gates.md)).
- Commit messages: Conventional Commits with the feature number and requirement IDs, e.g.
  `feat(007): even spread puts remainder in last month (FR-009)`.
- A pull request (or local review) checks: tests green, behaviour matches the spec, screens match the
  design system, docs and feature map updated.

## 7. Definition of Ready and Done

- **Ready** (before `/speckit-plan`): spec agreed, quality checklist passing, no open question that
  changes scope, dependencies Done or planned.
- **Done:** all tasks complete, all tests green, OpenAPI and docs updated, screens checked against the
  design system, demonstrated with the seed data, feature map status set to Done.
