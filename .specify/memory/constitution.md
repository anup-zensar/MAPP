<!--
Sync Impact Report
- Version change: (template) → 1.0.0
- Principles defined: I Spec first · II Shell business, plain language · III Simple, friendly experience ·
  IV Simple layered architecture · V Tested business rules · VI Safe by default (even with demo sign-in) ·
  VII Keep it small
- Added sections: Current Phase & Scope, Technology Choices, Development Workflow, Governance
- Templates reviewed: plan-template.md (Constitution Check reads this file — OK), spec-template.md (OK),
  tasks-template.md (tests are required by Principle V — OK), checklist-template.md (OK)
- Follow-up TODOs: none
-->

# NorthStar Constitution

NorthStar is Shell's marketing planning application that replaces Hive9. This constitution sets the few
rules everyone — people and AI agents — follows when specifying, designing, building and testing it.
When another document disagrees with this one, this one wins.

## Core Principles

### I. Spec First (NON-NEGOTIABLE)

- No code is written for behaviour that is not described in a feature specification under
  `specs/NNN-short-name/spec.md`.
- Every feature follows the Spec Kit flow: **specify → clarify → plan → tasks → analyze → implement**.
  `clarify` may be skipped only when the spec has no `[NEEDS CLARIFICATION]` markers.
- A spec says **what** the user can do and **why**, in business language. **How** (technology, code,
  storage) belongs in `plan.md`.
- Every functional requirement (`FR-###`) is covered by at least one task and one automated test.
- A change in behaviour starts with a change in the spec, in the same pull request as the code.

Rationale: the spec is the agreement between Shell and the team; code that is not in a spec is
unrequested work.

### II. Shell Business, Plain Language

- NorthStar uses Shell's planning vocabulary on every screen: **Plan → Campaign → Initiative → Activity →
  Line item**. Hive9 product words (Program, Tactic, Inspection Window, Honeycomb, HUD) are never shown
  to users. `docs/product/glossary.md` is the authoritative list of terms.
- **Shell's process is the functional reference; ZenReach is the design reference.** When sources
  disagree on behaviour, follow the precedence in `docs/reference/README.md` (Shell's own documents
  first). Every simplification of Hive9 behaviour is recorded in `docs/sdd/decision-log.md`.
- The data each level holds, its statuses and its calculations are defined once in
  `docs/product/domain-model.md` and `docs/product/field-catalogue.md`; specs reference them instead of
  redefining them.
- Business configuration is data, not code: models, activity types, custom attributes and their option
  lists, currencies and exchange rates, and finance budget items are managed on Settings screens, never
  hard-coded.
- **NorthStar owns plans; GSAP owns the money.** NorthStar never creates SWBS codes, purchase orders or
  invoices. GSAP integration is out of scope for the current phase.

Rationale: users already know Shell's words and process; they must not have to learn Hive9's screens.

### III. Simple, Friendly Experience

- A first-time user must understand every screen without training. The visual language follows
  `docs/product/design-system.md` (ZenReach style: clean white layout, left sidebar, compact page
  header, summary cards, rounded cards, one blue primary action, plain words).
- **Obvious CRUD.** Every record is created, viewed, edited and deleted with the same pattern: a clearly
  named create button, a searchable list, a details view with an **Edit** action, and a delete that says
  what will be removed and sends it to the recycle bin.
- **Status is always visible.** Every plan, campaign, initiative, activity and line item shows its status
  as a word plus a colour, and a completeness indicator ("3 required fields missing") that links to the
  missing fields.
- Validation shows **all** problems at once, next to the fields, in plain language. Nothing is greyed
  out without saying why and what to do instead.
- Unsaved changes are protected; destructive actions are confirmed; deletes can be undone.
- Screens meet WCAG 2.1 AA basics: keyboard use, visible focus, labelled controls, sufficient contrast,
  colour never the only signal.

Rationale: Hive9 is described by Shell users as too complex; NorthStar's value is that anyone can plan
in it.

### IV. Simple, Layered Architecture

- Exactly two runtime parts: a **React + TypeScript web app** (`frontend/`) and a **Go API**
  (`backend/`). Adding a third part (worker, queue, cache, separate service) requires an ADR.
- The Go API is one program with one package per business area (plans, activities, line items,
  settings, people, approvals, audit). Each package follows the same three layers:
  **handler (HTTP) → service (business rules) → repository (storage)**.
- **The server is the source of truth.** Every business rule, validation and calculation (roll-ups,
  remaining budget, Sys_Gen_Balance, spreads, currency conversion, statuses) runs in the Go API. The web
  app may repeat a check only to give faster feedback.
- The API is **REST + JSON**, described in an **OpenAPI** file in the repository. The web app's API types
  are generated from it, so the two sides cannot drift apart.
- All storage goes through repository interfaces. **Now:** a local JSON file store. **Later:**
  PostgreSQL, by adding a new repository implementation — no change to services or the web app. The
  data model is therefore kept relational in shape (ids, foreign keys, no nested ownership tricks).

Rationale: a small, predictable structure is easy for new developers and AI agents to follow, and keeps
the move to PostgreSQL and Azure cheap.

### V. Tested Business Rules (NON-NEGOTIABLE)

- Every business rule and calculation has unit tests written with or before the code, including edge and
  negative cases (money, roll-ups, spreads, budgets, status transitions, required fields).
- Every acceptance scenario in a spec becomes at least one automated test; every P1 user story has an
  end-to-end browser test.
- Every API endpoint is checked against the OpenAPI file by a contract test.
- Tests are deterministic: fixed dates and seed data, no dependence on run order or the real clock.

Rationale: planning money must be trustworthy; calculations cannot be checked by eye.

### VI. Safe by Default (Even With Demo Sign-in)

- Sign-in is a **demo sign-in** for now (decision D-004): a user picks or types a seeded demo account.
  There are no roles or permissions yet — every signed-in user can do everything.
- The seam for real security exists from day one: every request carries the signed-in user (the
  "actor"); every service operation receives it; every change is written to the activity log with who,
  what and when. Demo sign-in is enabled only in local and development environments.
- Input is validated by the API; no secrets, passwords or keys are committed; personal data is kept to
  name, e-mail and job title and never written to technical logs.
- Money is an exact decimal with a currency code — never a binary floating-point number. Dates are
  calendar dates; timestamps are stored in UTC.
- Deletes of business records go to the recycle bin and can be restored (parents first).

Rationale: adding real sign-in, roles and a database later must not require rewriting features.

### VII. Keep It Small and Readable

- Build the simplest thing that satisfies the approved spec (YAGNI). No speculative options, layers or
  abstractions.
- Prefer the standard library and well-known, widely used libraries. Every new dependency is justified in
  the feature's `research.md`.
- Code is written for a beginner to read: clear names, small functions, comments that explain *why*, the
  same folder layout in every package and feature.
- Significant or hard-to-reverse decisions are recorded as ADRs in `docs/architecture/adr/`.

Rationale: the user asked for an application that is industry-standard but not complex.

## Current Phase & Scope

- **Scope:** the **Plan module** and what it needs to work — demo sign-in, application shell, home
  dashboard, settings lists and custom attributes, people and teams, plans, campaigns, initiatives,
  activities, line items, activity approvals, planning views, activity log and recycle bin. The feature
  list and build order are in `docs/sdd/feature-map.md`.
- **Out of scope for now:** Finance (finance budgets, transactions, purchase requests, accruals),
  Measure reporting, GSAP integration, real sign-in and role-based access control, e-mail, AI features.
  Specs may mention these only as future integration points.
- **Running locally:** everything runs on a developer laptop with no cloud services: the Go API with the
  local JSON file store, and the web app's development server. Servers listen on `127.0.0.1` only.
- **Demo data:** a clearly labelled, fictitious seed dataset is loaded by an explicit command; it never
  contains real Shell people or figures.

## Technology Choices

| Area | Choice now | Later (when Azure is available) |
|---|---|---|
| Web app | React + TypeScript (strict), built with Vite | Same, served as static files |
| API | Go (current stable), standard-library HTTP server, REST + JSON, OpenAPI contract | Same |
| Storage | Local JSON file store behind repository interfaces | PostgreSQL (Azure Database for PostgreSQL) |
| Sign-in | Demo sign-in with seeded users, no roles | Microsoft Entra ID single sign-on and role-based access |
| Hosting | Local laptop | Azure Container Apps behind Azure Front Door (see `docs/reference/README.md`) |
| Source control | Git, one branch per feature | GitHub with pull requests and CI |

Detailed library choices are made in each feature's `plan.md` and recorded in `research.md`; the
shared ones are fixed by feature 001.

## Development Workflow

| Step | Spec Kit command | Output | Done when |
|---|---|---|---|
| 1. Specify | `/speckit-specify` | `spec.md`, `checklists/requirements.md` | Product owner agrees the stories, rules and acceptance scenarios |
| 2. Clarify | `/speckit-clarify` | Answers recorded in `spec.md` | No open `[NEEDS CLARIFICATION]` that affects scope |
| 3. Plan | `/speckit-plan` | `plan.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md` | Constitution Check passes |
| 4. Tasks | `/speckit-tasks` | `tasks.md` | Every FR has a task and a test |
| 5. Analyze | `/speckit-analyze` | Consistency report | No CRITICAL findings |
| 6. Implement | `/speckit-implement` | Code and tests | All tasks done, tests green, demo accepted |

- One feature per branch, named exactly like its folder: `NNN-short-name`. `main` only receives finished,
  reviewed features.
- Commits follow Conventional Commits and mention the feature number, e.g. `feat(005): copy plan (FR-012)`.
- **Definition of Done:** tasks complete, all tests green, OpenAPI and docs updated, screens checked
  against `docs/product/design-system.md`, feature demonstrated with the seed data.

## Governance

- This constitution overrides every other practice. `/speckit-plan` runs a Constitution Check against
  it; any exception is written in the plan's Complexity Tracking table with a reason.
- Changes are made by pull request that updates this file and its Sync Impact Report, and are agreed by
  the product owner and the lead developer.
- Versioning: MAJOR when a principle is removed or redefined, MINOR when one is added or materially
  expanded, PATCH for wording.
- Runtime guidance for AI agents is in `CLAUDE.md`.

**Version**: 1.0.0 | **Ratified**: 2026-10-18 | **Last Amended**: 2026-10-18
