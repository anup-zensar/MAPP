# Architecture Decision Records

Short records of decisions that are significant or hard to reverse. Format: Context · Decision ·
Consequences. Add a new ADR (next number) instead of editing an accepted one; mark the old one
"Superseded by ADR-NNNN".

| ADR | Title | Status |
|---|---|---|
| [0001](0001-spec-driven-development-with-spec-kit.md) | Spec-driven development with GitHub Spec Kit | Accepted |
| [0002](0002-react-typescript-web-app.md) | React + TypeScript web app | Accepted |
| [0003](0003-go-api-layered-monolith.md) | Go API as one layered program | Accepted; HTTP server superseded by ADR-0011 |
| [0004](0004-json-file-store-now-postgresql-later.md) | JSON file store now, PostgreSQL later | Superseded by ADR-0013 |
| [0005](0005-demo-sign-in-rbac-later.md) | Demo sign-in now, Entra ID and roles later | Superseded in part by ADR-0008 (demo sign-in stays) |
| [0006](0006-design-language.md) | MAPP's own design language | Accepted; visual part replaced by D-310–D-314 |
| [0007](0007-activity-log-and-recycle-bin-core.md) | Activity log and recycle bin core | Accepted |
| [0008](0008-role-based-access-with-team-scopes.md) | Role-based access with team scopes | Accepted |
| [0009](0009-simulated-gsap-feeds-behind-adapters.md) | Simulated GSAP feeds behind adapters | Accepted |
| [0010](0010-quality-gates-and-branch-rules.md) | Quality gates and branch rules | Accepted |
| [0011](0011-gin-transport-and-opentelemetry.md) | Gin transport and OpenTelemetry | Accepted |
| [0012](0012-local-container-images.md) | Local container images for reviewers | Accepted |
| [0013](0013-postgresql-storage.md) | PostgreSQL storage | Accepted |
| [0014](0014-transaction-memo-and-batched-reads.md) | Transaction memo and batched reads | Accepted |
| [0015](0015-pnpm-package-manager.md) | pnpm as the web app's package manager | Accepted |
| [0016](0016-simulated-gsap-round-trip.md) | A simulated GSAP for SWBS codes, GSAP PRs and purchase orders | Accepted |
| [0017](0017-dashboards-charts-and-office-exports.md) | Dashboard charts as plain SVG; PowerPoint and PDF files made by the API | Accepted |
| [0018](0018-web-content-security-policy.md) | The web app's content security policy, with inline style elements allowed | Accepted |
| [0019](0019-overspend-guard-unscoped-budget-read.md) | The overspend guard reads every budget without the actor's scope | Proposed |
| [0020](0020-simulated-performance-feed.md) | Performance data arrives only by upload or a simulated feed behind an adapter | Proposed |
| [0021](0021-one-active-source-rule-in-the-service.md) | One active source per model stage is a service rule, not a database constraint | Proposed |
