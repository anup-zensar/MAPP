# Repository Layout

The folders MAPP uses. Feature 001 (`specs/001-app-shell-and-sign-in/plan.md`) created the
structure; the number after a package or folder is the feature that creates it.

```
MAPP/
├── README.md                    what MAPP is and how to run it
├── CLAUDE.md                    guidance for AI coding agents
├── .specify/                    GitHub Spec Kit (templates, scripts, constitution)
│   └── memory/constitution.md   the project's rules
├── .claude/skills/              Spec Kit commands for Claude Code (/speckit-specify, …)
├── docs/
│   ├── product/                 vision, domain model, field catalogue, design system, glossary, demo data
│   ├── architecture/            overview, this file, ADRs
│   ├── sdd/                     how we work, feature map, decision log
│   └── reference/               source material (read-only): Shell documents, Hive9 analysis, BRD
├── specs/
│   └── NNN-short-name/          one folder per feature
│       ├── spec.md              WHAT and WHY            (/speckit-specify)
│       ├── checklists/          quality checklists      (/speckit-specify, /speckit-checklist)
│       ├── plan.md              HOW                     (/speckit-plan)
│       ├── research.md          technical decisions     (/speckit-plan)
│       ├── data-model.md        records and fields      (/speckit-plan)
│       ├── contracts/           API changes             (/speckit-plan)
│       ├── quickstart.md        how to try the feature  (/speckit-plan)
│       └── tasks.md             ordered work items      (/speckit-tasks)
├── backend/                     Go API (module "mapp")
│   ├── api/openapi.yaml         the API contract (source of truth) + embed.go             (001)
│   ├── cmd/api/                 the API program                                           (001)
│   ├── cmd/seed/                creates and migrates the database, loads the demo data    (001, 023)
│   ├── seed/                    the demo dataset; every feature adds its records          (001+)
│   ├── internal/
│   │   ├── app/                 configuration and wiring: the only package that knows every other   (001)
│   │   ├── platform/            small shared helpers                                                 (001)
│   │   │   ├── actor/           who is acting (id, name) and the context helpers
│   │   │   ├── clock/           "now", replaceable in tests
│   │   │   ├── errs/            the one error format and its codes
│   │   │   ├── web/             JSON in/out, errors to HTTP, middleware
│   │   │   ├── database/        PostgreSQL: pool, migrations, queries, sqlcdb, dbtest     (023)
│   │   │   ├── memo/            values kept for the rest of a read-only transaction       (023)
│   │   │   ├── fake/            test fakes' transactions, tables and ID counters (tests only)  (029)
│   │   │   ├── ids/             readable IDs (P-1001 …)
│   │   │   ├── record/          version and created/updated bookkeeping
│   │   │   └── apitest/         contract-test helper (tests only)
│   │   ├── auth/                demo sign-in, sessions, the sign-in middleware              (001)
│   │   ├── people/              people, "me", preferences; teams and profiles               (001, 003)
│   │   ├── history/             activity log and recycle bin core; screens' needs           (001, 011)
│   │   ├── sample/              developer sample records (local/development only)          (001)
│   │   ├── settings/            models, activity types, attributes, currencies, budget items (002)
│   │   ├── plan/                plans                                                        (004)
│   │   ├── campaign/            campaigns and initiatives                                    (005)
│   │   ├── activity/            activities                                                   (006)
│   │   ├── lineitem/            line items and spreads                                       (007)
│   │   ├── approval/            approval workflows, approvals, notifications                 (008)
│   │   ├── stores/              every area's store and the Transactor; storestest: in memory  (023, 029)
│   │   └── views/               cross-plan queries for All work                              (010)
│   │       each business package:  model.go · store.go · store_postgres.go · service.go ·
│   │                               handler.go, <area>test/ (its fake), a file per use case,
│   │                               each within 300 non-blank lines, tests beside them      (025)
└── frontend/                    React + TypeScript web app
    ├── src/                     constitution F2 (feature 026)
    │   ├── main.tsx             mounts <App /> and nothing else
    │   ├── app/                 providers, routes (one module), lazy pages, layout, sidebar, bell
    │   ├── features/<area>/     one business area each: index.ts (its only entry), api.ts (query
    │   │                        hooks), schemas.ts (Zod), pages and components; plans holds
    │   │                        campaigns/ and phasing/, people holds access/
    │   ├── ui/                  design tokens and shared building blocks (Button, FormField, …)
    │   ├── api/                 the typed client; generated/ holds the generated contract types
    │   ├── session/             the signed-in person, route guards, permissions
    │   ├── format/              Intl dates, times and money; exact decimals
    │   └── testing/             test set-up, the fake API on MSW, render helpers, fixtures/
    └── e2e/                     browser end-to-end tests (Playwright)                                 (001+)
```

Rules:

- A feature touches only its own package/folder plus the shared files it is allowed to extend (OpenAPI
  file, `internal/app/app.go`, `seed/seed.go`, `src/app/routes.tsx`, `src/app/pages.tsx`,
  `src/app/navigation.ts`). The plan of each feature lists those shared files.
- In the web app, dependencies point inward (`app` → `features` → `ui`, `api`, `session`, `format`),
  a feature reaches another only through its `index.ts`, and there are no import cycles;
  `frontend/eslint.config.js` enforces all three (feature 026).
- Package names match business areas, not technical layers (`plans`, not `controllers`).
- Tests sit next to the code they test. Handler (contract) tests are external test packages
  (`package plans_test`) because the contract-test helper builds the whole application.
- Business packages never import each other's internals in a circle: shared behaviour goes through
  `platform` packages or the `history.Kind` registry, and only `internal/app` wires everything together.
