# Shell Target Architecture (summary)

Summary of *Application and Azure Deployment Architecture* (v3, 6 slides, Oct 2026). This is the
**production target** for later; MAPP's current phase runs locally (see the constitution's Technology
Choices). Supersedes the 4-slide *Hive9 Final Architecture* deck this page used to summarise; the two
slides added in v3 are the delivery ones, in §Delivery below.

## Components

| Layer | Component | Purpose |
|---|---|---|
| Edge | Microsoft Entra ID + SAM | Shell single sign-on with MFA (auth code + PKCE via MSAL); roles are provisioned through SAM (Shell Access Manager), which owns access governance |
| Edge | Azure Front Door Premium + WAF | The only public entry; one domain; `/` → web app, `/api` → API Management |
| Presentation | React + TypeScript app | All screens; static files served by nginx; Shell Design System (RCF); WCAG |
| Gateway | Azure API Management (internal) | Validates the JWT, rate limits, routes to the Go API, publishes the OpenAPI contract |
| Application | Go API | Packages per domain (plan, budget, workflow, reconcile, report, admin); RBAC middleware (role + team); stateless |
| Application | Go GSAP worker | Daily Container Apps Job: writes FSSI 1523 (SWBS requests) to `/outbound`, reads PGSI 941 (PR/PO/invoice) from `/inbound`, matches on HX + line-item ID, idempotent reruns, rejected rows to a finance exception list |
| Hosting | Azure Container Apps | Web, API and worker containers in a private network |
| Data | Azure Database for PostgreSQL Flexible Server | Plans, budgets, workflow, audit; read replica for dashboards |
| Data | Azure Blob Storage with SFTP | GSAP file drop (`/outbound`, `/inbound`, `/archive`), attachments, exports |
| External | SAP Cloud Integration → SAP GSAP | Moves files between Blob and GSAP; GSAP owns SWBS, PR, PO, invoices |
| Hosting | One resource group per environment | `rg-mapp-dev`, `rg-mapp-test`, `rg-mapp-prod`, each in a Shell landing-zone spoke VNet |
| Controls | VNet + private endpoints, Key Vault + managed identity, Dynatrace | Only Front Door is public; no secrets in code; Dynatrace OneAgent for APM, dashboards, alerting and SLOs |
| Delivery (SEMS) | GitHub + Actions (OIDC) aligned to Shell TCM, ACR, SonarQube, Apiiro, GHAS, Mend, Wiz, Invicti, Azure Boards, ServiceNow | Quality and security gates; a change ticket for every production release |

## Principle

"The application owns plans and workflow · GSAP owns financial truth · the application never calls GSAP
directly — files only, via SAP Cloud Integration." About 2,000 users.

## Line-item GSAP status (future)

Approved → Sent to GSAP → Confirmed by GSAP (24–48 h). Rejected rows go to a finance exception list.

## Delivery (slides 5 and 6, added in v3)

Eight steps from Plan to Release: Azure Boards item (ECCN and asset record) → GitHub PR review → build
and test → code scans → package → deploy test → ServiceNow approval → release the same image to prod.

Three gates must pass:

| Gate | Must be true |
|---|---|
| Architecture | Front Door, APIM and SAM onboarding agreed; GSAP interface ownership confirmed; data flows and retention documented |
| Security & compliance | No secrets in source; SAST, SCA, DAST and CSPM with no high or critical; unit decision coverage ≥ 80% on changed code |
| Operational readiness | Dynatrace SLOs, dashboards and alerts live; backup/restore and DR tested; runbooks and support handover; load test at expected peak |

Three pipelines, one image built once in Dev and promoted unchanged by commit SHA:

- **Dev** (`rg-mapp-dev`), on pull request and merge to `main`: lint and unit tests → code scans → build
  image → push to ACR → deploy → smoke test.
- **Test** (`rg-mapp-test`), manual or nightly: promote the same SHA (no rebuild) → migrations and
  deploy → Selenium E2E and contract tests → DAST and CSPM → GSAP round-trip via CPI SFTP and load test.
- **Release** (`rg-mapp-prod`), on a release tag: ServiceNow change → CAB and environment approval →
  migrations and blue-green revision switch → smoke test and Dynatrace SLOs → rollback ready.

Each pipeline has its own GitHub environment, OIDC federated identity and resource group; no long-lived
secrets. After release: critical vulnerabilities fixed ≤ 30 days, high ≤ 60 days, secrets found after
deployment ≤ 30 days. Zensar delivers under the same SEMS compliance points.

**Not built yet.** MAPP's CI today is one GitHub Actions workflow (`.github/workflows/ci.yml`) that
lints, tests and scans for vulnerabilities. There is no image build, registry push, deployment,
environment or IaC in the repository, and none of the SEMS-named scanning tools are wired up.
