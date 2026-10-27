# Reference Material (read-only)

Inputs used to write MAPP's specs. They are **sources, not requirements** — the requirements are the
specs under `specs/` and the product docs under `docs/product/`. Do not edit these files except to
refresh them from a newer version of the source.

MAPP's requirements and features were collected from **the available documentation** listed here and
from **the Hive9 application itself** — Shell's live Hive9 sandbox, analysed in the functional analysis
(D-318). MAPP's look and layout are its own design system (`docs/product/design-system.md`).

| File | Abbrev. | Source | Date | Use |
|---|---|---|---|---|
| [hive9-plan-module-as-is.md](hive9-plan-module-as-is.md) | PM | Shell, *Hive9 Plan Module* (as-is plan creation, 6 steps, field tables, 11 screenshots) | 28 Sep 2026 | **Highest precedence** for Plan fields and steps |
| [hive9-functional-analysis.md](hive9-functional-analysis.md) | FA | Zensar analysis of the Shell Hive9 sandbox, vendor guide and GSAP workshop — Part A (foundations) and Part B (Plan module) are the relevant parts | 24 Sep 2026 | Behaviour, statuses, rules |
| [brd-requirements.md](brd-requirements.md) | BRD | Shell, *Business Requirements* (36 rows: MVP, future-state, RFP, needs clarification) | 20 Oct 2026 | Requirements checked in the gap analysis and the final validation report |
| [gsap-workshop-summary.md](gsap-workshop-summary.md) | WS | Shell/Zensar workshop "Understanding GSAP–Hive9 integrations" | 10 Sep 2026 | Line-item finance fields, SWBS, future GSAP flow |
| [hive9-replacement-project-plan.md](hive9-replacement-project-plan.md) | PP | Zensar project plan: personas (19 roles), in-scope features, roadmap | Oct 2026 | Scope |
| [prism-findings-summary.md](prism-findings-summary.md) | PRISM | Sidekick for Shell, *PRISM Phase 1 Findings and Future-State Requirements* | May 2026 | Priorities: foundations before features |
| [shell-target-architecture.md](shell-target-architecture.md) | ARCH | *Application and Azure Deployment Architecture* deck (v3, 6 slides) | Oct 2026 | Target platform (later) |

**Precedence when sources disagree on behaviour (D-007):** PM > FA (Live) > WS > FA (Guide).
**The design system decides layout, look and interaction (D-005).**

## Sources kept outside the repository

- `Hive9 - User & Admin Guide - EN.pdf` (Uptempo vendor guide) — the "Guide" evidence in FA.
- `Workshop_ Understanding GSAP-Hive9 Integrations.docx` — full transcript (summarised here).
- `Application_and_Azure_Deployment_Architecture_Enterprise_Updated_v3.pdf` — summarised here
  (supersedes `Hive9_Final_Architecture.pptx`).
- `Business Requirements.pdf` — transcribed in `brd-requirements.md`.
