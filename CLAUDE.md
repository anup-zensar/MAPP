# CLAUDE.md — guidance for AI agents working on NorthStar

Read this with `.specify/memory/constitution.md` (the rules) before doing anything.

## Project

NorthStar is Shell's marketing planning application replacing the Hive9 Plan module. This phase covers
only the **Plan module**: demo sign-in, application shell, home dashboard, settings, people and teams,
plans, campaigns, initiatives, activities, line items, activity approvals, planning views, activity log
and recycle bin. Spec-driven development with GitHub Spec Kit.

## Non-negotiables

1. **Spec first.** Only build behaviour described in `specs/NNN-short-name/spec.md`. Use the Spec Kit
   skills: `/speckit-specify`, `/speckit-clarify`, `/speckit-plan`, `/speckit-checklist`,
   `/speckit-tasks`, `/speckit-analyze`, `/speckit-implement`, `/speckit-converge`.
2. **Keep it simple and readable** (constitution VII): standard library first, few well-known libraries,
   the same folder pattern everywhere, comments that explain why. A beginner must be able to follow it.
3. **Two parts only:** React + TypeScript in `frontend/`, Go in `backend/`. Go packages per business area,
   each handler → service → repository. All business rules in services.
4. **Storage via repository interfaces**; the JSON file store now, PostgreSQL later. Keep records
   relational in shape.
5. **Contract first:** endpoints are described in `backend/api/openapi.yaml`; the web app's API types
   are generated from it.
6. **Demo sign-in, no roles** (D-004): every signed-in user can do everything; still pass the actor to
   every service call and write the activity log.
7. **Tests:** unit tests for every rule and calculation, a test for every acceptance scenario, an
   end-to-end test for every P1 story, contract tests for every endpoint. Deterministic (fixed clock and
   seed).
8. **Money** = exact decimal + currency (base USD). Dates are calendar dates; timestamps in UTC.
9. **Shell vocabulary** on screen: Plan → Campaign → Initiative → Activity → Line item. Never Program or
   Tactic; never Hive9 interface terms (see `docs/product/design-system.md` §8).
10. **ZenReach design, beginner-friendly:** follow `docs/product/design-system.md` — the standard CRUD
    pattern, status and completeness chips on every record, all validation errors at once, delete to the
    recycle bin with Undo, nothing greyed out without a reason.
11. **Out of scope:** Finance, Measure, GSAP integration, e-mail, real sign-in, AI. Do not build them.
12. **Local only:** servers listen on `127.0.0.1`; no cloud services; no secrets in the repository.

## Key documents

| Need | File |
|---|---|
| Features, order, scope boundaries | `docs/sdd/feature-map.md` |
| Decisions and open questions | `docs/sdd/decision-log.md` |
| Process, branches, Definition of Done | `docs/sdd/sdd-process.md` |
| Records, statuses, money rules | `docs/product/domain-model.md` |
| Fields per level | `docs/product/field-catalogue.md` |
| Screens and wording | `docs/product/design-system.md` |
| Terms | `docs/product/glossary.md` |
| Demo users and seed story | `docs/product/demo-users-and-data.md` |
| Architecture, folders, ADRs | `docs/architecture/` |
| Source material (read-only) | `docs/reference/` (precedence in its README) |

## Conventions

- Branch per feature: `NNN-short-name`, the same as its folder under `specs/`.
- Conventional Commits with the feature number and requirement IDs, e.g. `feat(004): copy plan (FR-012)`.
- Test names mention the requirement ID they prove.
- Shell environment: Windows; Spec Kit scripts are PowerShell (`.specify/scripts/powershell/`).

## Active technologies

To be filled in by feature 001's `/speckit-plan` (libraries and versions for the web app and the API).
