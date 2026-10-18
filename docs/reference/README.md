# Reference Material (read-only)

Inputs used to write NorthStar's specs. They are **sources, not requirements** — the requirements are the
specs under `specs/` and the product docs under `docs/product/`. Do not edit these files except to
refresh them from a newer version of the source.

| File | Abbrev. | Source | Date | Use |
|---|---|---|---|---|
| [hive9-plan-module-as-is.md](hive9-plan-module-as-is.md) | PM | Shell, *Hive9 Plan Module* (as-is plan creation, 6 steps, field tables, 11 screenshots) | 28 Sep 2026 | **Highest precedence** for Plan fields and steps |
| [hive9-functional-analysis.md](hive9-functional-analysis.md) | FA | Zensar analysis of the Shell Hive9 sandbox, vendor guide and GSAP workshop — Part A (foundations) and Part B (Plan module) are the relevant parts | 24 Sep 2026 | Behaviour, statuses, rules |
| [gsap-workshop-summary.md](gsap-workshop-summary.md) | WS | Shell/Zensar workshop "Understanding GSAP–Hive9 integrations" | 10 Sep 2026 | Line-item finance fields, SWBS, future GSAP flow |
| [hive9-replacement-project-plan.md](hive9-replacement-project-plan.md) | PP | Zensar project plan: personas (19 roles), in-scope features, roadmap | Oct 2026 | Scope |
| [prism-findings-summary.md](prism-findings-summary.md) | PRISM | Sidekick for Shell, *PRISM Phase 1 Findings and Future-State Requirements* | May 2026 | Priorities: foundations before features |
| [shell-target-architecture.md](shell-target-architecture.md) | ARCH | *Hive9 Final Architecture* deck (4 slides) | Oct 2026 | Target platform (later) |
| [zenreach-feature-specification.md](zenreach-feature-specification.md) | ZR | ZenReach complete feature specification (reference implementation of a Hive9 alternative) | 17 Oct 2026 | **Design and interaction reference**; lowest precedence for behaviour |
| [zenreach-screenshots/](zenreach-screenshots/) | ZR-UI | Selected ZenReach screenshots (planning, home, budget, builder, admin, procurement, mobile, hero) | Oct 2026 | Visual reference for the design system |

**Precedence when sources disagree on behaviour (D-007):** PM > FA (Live) > WS > FA (Guide) > ZR.
**ZenReach decides layout, look and interaction (D-005).**

Note: the ZenReach repository copy available to the team (commit `bda4e42`) is older than the ZenReach
specification — the specification's five-level planning model, grid, Inspection Window and line-item
dialog are not in that code. Use the **specification**, not the old code, as the reference.

## Sources kept outside the repository

- `Hive9 - User & Admin Guide - EN.pdf` (Uptempo vendor guide) — the "Guide" evidence in FA.
- `Workshop_ Understanding GSAP-Hive9 Integrations.docx` — full transcript (summarised here).
- `Hive9_Final_Architecture.pptx` — summarised here.
- `ZenReach Solution Proposal for Shell_2.pdf` and the ZenReach decks — commercial proposals, not inputs.
