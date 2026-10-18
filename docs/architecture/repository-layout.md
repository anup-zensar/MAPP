# Repository Layout

The folders NorthStar uses. Folders marked *(feature 001)* are created when feature 001 is implemented;
until then `frontend/` and `backend/` contain only a README.

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
│   └── reference/               source material (read-only): Shell documents, Hive9 analysis, ZenReach spec
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
├── backend/                     Go API
│   ├── cmd/api/                 main program                              (feature 001)
│   ├── api/openapi.yaml         the API contract (source of truth)        (feature 001)
│   ├── internal/
│   │   ├── platform/            shared helpers: http/json, errors, ids, money, clock, store   (001)
│   │   ├── auth/                demo sign-in and the actor                                   (001)
│   │   ├── settings/            models, activity types, attributes, currencies, budget items (003)
│   │   ├── people/              people and teams                                             (004)
│   │   ├── plans/               plans                                                        (005)
│   │   ├── campaigns/           campaigns and initiatives                                    (006)
│   │   ├── activities/          activities                                                   (007)
│   │   ├── lineitems/           line items and spreads                                       (008)
│   │   ├── approvals/           approval workflows, approvals, notifications                 (009)
│   │   ├── views/               cross-plan queries for All work                              (010)
│   │   └── history/             activity log and recycle bin                                 (011)
│   │       each package:  model.go · service.go · service_test.go · repository.go ·
│   │                      store_json.go · handler.go · handler_test.go
│   ├── seed/                    demo data loader                                             (001+)
│   └── data/                    local JSON store — git-ignored
└── frontend/                    React + TypeScript web app
    ├── src/
    │   ├── app/                 router, layout (sidebar, header), providers                  (001)
    │   ├── components/          shared UI: Button, StatusChip, PageHeader, SummaryCard, …    (001)
    │   ├── features/<area>/     pages and components of one feature (plans, activities, …)
    │   ├── api/                 generated types + small fetch client                         (001)
    │   ├── lib/                 money, dates, formatting                                     (001)
    │   └── styles/tokens.css    design tokens from docs/product/design-system.md             (001)
    └── e2e/                     browser end-to-end tests                                     (001+)
```

Rules:

- A feature touches only its own package/folder plus the shared files it is allowed to extend (OpenAPI
  file, router, navigation, seed). The plan of each feature lists those shared files.
- Package names match business areas, not technical layers (`plans`, not `controllers`).
- Tests sit next to the code they test.
