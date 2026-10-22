# Repository Layout

The folders NorthStar uses. Feature 001 (`specs/001-app-shell-and-sign-in/plan.md`) created the
structure; the number after a package or folder is the feature that creates it.

```
NorthStar/
├── README.md                    what NorthStar is and how to run it
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
├── backend/                     Go API (module "northstar")
│   ├── api/openapi.yaml         the API contract (source of truth) + embed.go             (001)
│   ├── cmd/api/                 the API program                                           (001)
│   ├── cmd/seed/                loads the demo data into an empty data folder             (001)
│   ├── seed/                    the demo dataset; every feature adds its records          (001+)
│   ├── internal/
│   │   ├── app/                 configuration and wiring: the only package that knows every other   (001)
│   │   ├── platform/            small shared helpers                                                 (001)
│   │   │   ├── actor/           who is acting (id, name) and the context helpers
│   │   │   ├── clock/           "now", replaceable in tests
│   │   │   ├── errs/            the one error format and its codes
│   │   │   ├── web/             JSON in/out, errors to HTTP, middleware
│   │   │   ├── store/           the JSON file store and the Transactor interface
│   │   │   ├── ids/             readable IDs (P-1001 …)
│   │   │   ├── record/          version and created/updated bookkeeping
│   │   │   └── apitest/         contract-test helper (tests only)
│   │   ├── auth/                demo sign-in, sessions, the sign-in middleware              (001)
│   │   ├── people/              people, "me", preferences; teams and profiles               (001, 003)
│   │   ├── history/             activity log and recycle bin core; screens' needs           (001, 011)
│   │   ├── samples/             developer sample records (local/development only)          (001)
│   │   ├── settings/            models, activity types, attributes, currencies, budget items (002)
│   │   ├── plans/               plans                                                        (004)
│   │   ├── campaigns/           campaigns and initiatives                                    (005)
│   │   ├── activities/          activities                                                   (006)
│   │   ├── lineitems/           line items and spreads                                       (007)
│   │   ├── approvals/           approval workflows, approvals, notifications                 (008)
│   │   └── views/               cross-plan queries for All work                              (010)
│   │       each business package:  model.go · repository.go · store_json.go · service.go ·
│   │                               service_test.go · handler.go · handler_test.go
│   └── data/                    local JSON store — git-ignored
└── frontend/                    React + TypeScript web app
    ├── src/
    │   ├── app/                 routes, layout (sidebar, account menu, bell, theme), session          (001)
    │   ├── components/          shared building blocks: Button, PageHeader, StatusChip, …             (001)
    │   ├── features/<area>/     pages and components of one feature (sign-in, home, plans, …)
    │   ├── api/                 generated types (schema.d.ts) + small typed client                    (001)
    │   ├── lib/                 dates, money, page title, small helpers                               (001)
    │   ├── styles/              tokens.css (design tokens) and global.css                             (001)
    │   └── test/                test helpers (render, mock API)                                       (001)
    └── e2e/                     browser end-to-end tests (Playwright)                                 (001+)
```

Rules:

- A feature touches only its own package/folder plus the shared files it is allowed to extend (OpenAPI
  file, `internal/app/app.go`, `seed/seed.go`, `src/app/router.tsx`, `src/app/navigation.ts`). The plan of
  each feature lists those shared files.
- Package names match business areas, not technical layers (`plans`, not `controllers`).
- Tests sit next to the code they test. Handler (contract) tests are external test packages
  (`package plans_test`) because the contract-test helper builds the whole application.
- Business packages never import each other's internals in a circle: shared behaviour goes through
  `platform` packages or the `history.Kind` registry, and only `internal/app` wires everything together.
