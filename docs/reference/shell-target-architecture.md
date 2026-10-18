# Shell Target Architecture (summary)

Summary of *Hive9 Final Architecture* (4 slides, Oct 2026). This is the **production target** for later;
NorthStar's current phase runs locally (see the constitution's Technology Choices).

## Components

| Layer | Component | Purpose |
|---|---|---|
| Edge | Microsoft Entra ID | Shell single sign-on with MFA; the token carries app roles from Shell groups (auth code + PKCE via MSAL) |
| Edge | Azure Front Door Premium + WAF | The only public entry; one domain; `/` → web app, `/api` → API Management |
| Presentation | React + TypeScript app | All screens; static files served by nginx; Shell Design System (RCF); WCAG |
| Gateway | Azure API Management (internal) | Validates the JWT, rate limits, routes to the Go API, publishes the OpenAPI contract |
| Application | Go API | Packages per domain (plan, budget, workflow, reconcile, report, admin); RBAC middleware (role + team); stateless |
| Application | Go GSAP worker | Daily Container Apps Job: writes FSSI 1523 (SWBS requests) to `/outbound`, reads PGSI 941 (PR/PO/invoice) from `/inbound`, matches on HX + line-item ID, idempotent reruns, rejected rows to a finance exception list |
| Hosting | Azure Container Apps | Web, API and worker containers in a private network |
| Data | Azure Database for PostgreSQL Flexible Server | Plans, budgets, workflow, audit; read replica for dashboards |
| Data | Azure Blob Storage with SFTP | GSAP file drop (`/outbound`, `/inbound`, `/archive`), attachments, exports |
| External | SAP Cloud Integration → SAP GSAP | Moves files between Blob and GSAP; GSAP owns SWBS, PR, PO, invoices |
| Controls | VNet + private endpoints, Key Vault + managed identity, Azure Monitor | Only Front Door is public; no secrets in code; logs, traces, alerts |
| Delivery (SEMS) | GitHub + Actions (OIDC), ACR, SonarQube, GHAS, Mend, Wiz, Invicti, Azure Boards, ServiceNow | Quality and security gates; a change ticket for every production release |

## Principle

"The application owns plans and workflow · GSAP owns financial truth · the application never calls GSAP
directly — files only, via SAP Cloud Integration." About 2,000 users.

## Line-item GSAP status (future)

Approved → Sent to GSAP → Confirmed by GSAP (24–48 h). Rejected rows go to a finance exception list.
