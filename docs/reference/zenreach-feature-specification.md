# ZenReach — Complete Feature Specification

A file-by-file functional and technical specification of the ZenReach marketing-performance platform and its companion Dashboard Studio. It is written so a new team can rebuild every feature from scratch with the same behaviour, rules, numbers and screens.

| | |
|---|---|
| **Product** | ZenReach — "Marketing performance, beautifully clear." A self-serve alternative to Hive9 (Uptempo). |
| **Companion app** | Dashboard Studio — a standalone no-code dashboard builder for uploaded CSV/Excel data (`dashboard-studio/`). |
| **Source baseline** | ZenReach repository, branch `develop` at commit `bda4e42`, plus the uncommitted working tree as of 17 Oct 2026, which adds the five-level Hive9 planning model: models, line items, approvals, Grid, Budget & Cost and the Inspection Window. |
| **Coverage** | Every source file in `server/`, `client/` and `dashboard-studio/`, plus all repository documentation, sample data files and persisted data shapes. |
| **Size of the system** | 141 HTTP routes · 138 client API methods · 16 JSON data stores · 15 routed pages (Planning alone has 5 views) · 9 attribution models · 13 integration connectors · 5 planning levels. |
| **Document date** | 17 October 2026 |

## 1. About this specification

### 1.1 Purpose

This document describes ZenReach exactly as it is built, so that a team can build a functionally equivalent product from an empty repository. It covers four layers for every feature:

- **What the user sees and does:** screens, controls, labels, states, permissions and messages.
- **What the system stores:** every entity and field, with types, defaults, limits and normalisation.
- **What the system computes:** every business rule, validation, formula and constant, with exact error messages.
- **How the parts talk:** every HTTP endpoint, with its request, response, role gate, side effects and errors.

It is a description of current behaviour, not a wish list. Where the current behaviour looks accidental, inconsistent or risky, it is marked **Quirk**, **Gap** or **Bug** and collected in a "decide deliberately" section at the end of each chapter. Chapter 17 rolls these up into a decision log for the rebuild.

### 1.2 How the document is organised

| Chapters | Content |
|---|---|
| 1–4 | Orientation: this guide, the product, the architecture, and the conventions every module shares (storage, IDs, API envelope, authentication, roles, audit, notifications, browser storage). |
| 5 | The complete HTTP API reference (all 141 routes) and the client API wrapper. |
| 6 | Identity, accounts, organisation chart, notifications, audit log and recycle bin. |
| 7–8 | Planning, the core module: the five-level hierarchy, dual SP&A/BCPIR campaign plans, approvals, line items, board, imports, calendars and the AI plan loop. Chapter 7 is the domain model and backend; chapter 8 is every planning screen. |
| 9 | Finance: top-down budgets, purchase requests and approval chains, and multi-currency. |
| 10 | Performance and intelligence: the demo data spine, Google Analytics 4, goals, insights, AI analysis, the optimizer, attribution, and the "Ask ZenReach" assistant. |
| 11 | The low-code dashboard Builder and the saved-dashboards viewer. |
| 12 | The application shell and design system: routes, navigation, theming, tokens, shared components and the glossary. |
| 13–15 | Non-functional requirements; the demo story, roadmap and Hive9 parity; and data reference material (target relational schema, import formats, persisted file shapes). |
| 16 | The companion product, Dashboard Studio. |
| 17 | The rebuild plan: suggested build order, acceptance checklist and the consolidated decision log. |
| Appendix A | An inventory of every source file with its responsibility. |

### 1.3 Conventions used in this document

- **Paths** are relative to the repository root. For example, `server/services/plans.js`, and `dashboard-studio/src/…` for the companion app.
- **Code identifiers, field names, endpoints and literal values** are in `monospace`. Quoted strings in "double quotes" are exact UI copy or exact error messages and should be reproduced verbatim.
- **Roles.** Two independent axes control access:
  - **workspace roles** `Owner`, `Admin`, `Editor`, `Viewer` (and the legacy `Member`, which ranks as Editor);
  - **org roles** `admin`, `manager`, `lead`, `member`.
  - "Editor+" means Editor or higher; "Admin+" means Admin or Owner.
- **Money** is stored in US dollars everywhere and converted for display (chapter 9).
- **Dates** are ISO `YYYY-MM-DD` strings; timestamps are ISO-8601 UTC strings.
- **Secrets** are referred to by environment-variable name only. No credential, token, password, hash, personal email address or account ID appears in this document.

### 1.4 What "rebuild" means here

The rebuild team may change the technology: for example, a relational database in place of the JSON file store (chapter 15 includes the target schema), or TypeScript in place of JavaScript. The externally observable behaviour described here is the contract: screens, rules, numbers, permissions and messages. Two kinds of data deserve care:

- **Deterministic demo data**, such as seeded campaigns, synthetic journeys and derived plan figures. Reproduce the algorithms and constants exactly, so that the reference outputs in chapter 10 and the demo script in chapter 14 still hold.
- **Live integrations** (Google Analytics 4, Google Calendar, FX rates, OpenAI/Azure/Anthropic, Trello and Jira). Implement them against the same external APIs, with the same fallbacks.

## 2. Product overview

What ZenReach is, who it is for, what it does module by module, and which capabilities are real versus simulated. Sources: `README.md`, `ZenReach-Product-Overview.md`, `ZenReach-Project-Context.md`, `docs/*`.

### 2.1 Product vision and positioning

- **Name and tagline:** ZenReach, "Marketing performance, beautifully clear." It was renamed OpenHive → Zenvue → ZenReach, briefly → ShellReach, then back to ZenReach.
  - The workspace folder was historically called `Hive9`.
  - Package names are `zenreach`, `zenreach-client` and `zenreach-server`.
  - Three runtime keys deliberately kept the old spelling: the localStorage key `zenreach.snapshots.v1`, the Google Calendar tag `extendedProperties.private.zenreachTacticId`, and the demo domain `zenreach.demo`.
- **One-line pitch:** "a lightweight, open alternative to Hive9". Root `package.json` describes it as: *"dual-plan (SP&A + BCPIR/ROI) marketing planning with budgeting, procurement, approvals, dashboards and live Google Analytics."*
- **The three leadership questions it answers:**
  1. Where is our money going?
  2. Is it making us money?
  3. Will we hit our targets?
- **Positioning against Hive9 (Uptempo):**
  - Hive9 became part of Uptempo in 2021, alongside Allocadia (budgeting) and BrandMaker (work management). Today "Hive9" mostly means the performance-measurement and attribution layer.
  - ZenReach keeps Hive9's best idea: one view of plan → budget → goals → results, with campaign ROI and cross-team dashboards.
  - It attacks Hive9's weaknesses:
    - too many clicks and complexity;
    - rigid hierarchy naming;
    - slow bulk entry;
    - roughly 24-hour batch syncs;
    - painful imports;
    - a ~$25k+/yr sales-led, opaque price;
    - vendor-led implementation;
    - a fragmented brand after the merger;
    - no media-mix modelling or experimentation.
  - The pitch is "free, runs locally in minutes, self-serve."
- **Pricing (spec doc):** not finalized. The one firm commitment is "meaningfully below Hive9 at every tier," transparent and self-serve. The illustrative tiers are:
  - **Free/Trial:** core planning and analytics, limited history.
  - **Team:** SMB.
  - **Business:** adds attribution, closed-loop AI and more integrations.
  - **Enterprise:** SSO, custom connectors, SLA, data residency.
- **"Beyond Hive9" differentiators (README):**
  1. Budget Optimizer, a what-if tool with an AI-recommended mix.
  2. AI Analysis on the dashboard: 4 ranked, quantified insights with actions.
  3. The "Ask ZenReach" chat on every page, with a "take me there" button.
  4. A low-code dashboard Builder.
  5. AI that builds a dashboard for you.
  6. A dual-plan (SP&A + BCPIR) Planning module.
  7. Collaboration features: an app-wide people picker with notifications, approvals locked to the assigned approver, context-aware Budget and Procurement, and a Recycle bin.
  8. The spec doc adds a closed-loop AI planner, called the "signature differentiator."
- **Client context:** the newer dual-plan model copies **Shell's** Hive9 usage.
  - **SP&A** (Sales, Promotion & Advertising) is the *spend* plan.
  - **BCPIR** (Business Case & Post-Investment Review) is the *ROI* plan.
  - The repo also holds untracked Shell and Zensar artifacts: a Shell Hive9 HLD SpecKit PDF, GSAP↔Hive9 integration tech specs (Projects & SWBS, Actuals, PurchaseRequisition), discovery and delivery-plan decks, a Gantt, a Sep–Dec 2026 delivery plan xlsx, and the "Hive9 User & Admin Guide" PDF.
  - The memory notes say "No AI is used in the Planning module (deterministic by design)."
- **Disclaimer used in the docs:** not affiliated with Uptempo or Apple; all screenshot figures are synthetic.

### 2.2 Target users and personas

**Stated audience (Product Overview):**
- marketing leaders who want a one-glance "how are we doing";
- marketing ops teams tired of stitching spreadsheets together;
- finance partners who want the same numbers;
- anyone who found Hive9 too heavy, slow or expensive.

**Spec-coding doc:** PMs and founders, designers and domain experts can drive specs; engineers own correctness.

**Permission roles** (who may edit):
- Owner: the first account created.
- Admin.
- Editor.
- Viewer: browse only.
- The legacy role `Member` ranks as Editor.

**Org roles** (a separate axis that drives dashboards and scope): admin → manager → lead → team → member.
- Managers and leads can customize the board.
- Members use the standard board.
- Home is role-aware: managers, leads and members get a TeamDashboard built around a "which team → which campaign" matrix; admins and others get the saved-dashboards viewer.

**Demo story cast** (Nova script):
- Abhishek Biyani, VP of Marketing, the protagonist. The demo org models him as a "lead" persona. He was renamed from "Priya Nair", and the script still uses "she."
- Marcus Lee, Demand Gen and ABM (the manager persona).
- Sam Okafor, Brand and Content.
- Dana Alvarez, Lifecycle and Partnerships.

**Seeded demo org:**
- 1 admin persona, 2 managers → 4 leads → 6 teams → 18 members, all login-able at `@zenreach.demo` with one shared demo password (a `DEMO_PASSWORD` constant; not reproduced).
- A persona switcher, "Explore as a demo persona", sits in the login modal.
- The real accounts are 2 Owner/Admin users.

**Seeded teams (6), name → focus:**

| Team | Focus |
|---|---|
| Paid Acquisition Squad | Paid search & social |
| Events & Webinars Crew | Field & digital events |
| Lifecycle & Email Team | Nurture & retention |
| Content Engine Team | Content & SEO |
| Named-Account Team | ABM plays |
| Partner Co-Marketing Team | Partnerships |

**Owner names in the sample CSVs:** Aisha Khan, Dana Alvarez, Elena Rossi, Jordan Kim, Lucia Mendes, Marcus Lee, Priya Nair, Raj Patel, Sam Okafor, Tom Becker, plus Nadia Haddad and Owen Wright in the 1500-row file only.

**Job titles in the user directory:** ABM Lead, Campaign Manager, Content Lead, Content Strategist, Demand Gen Lead, Growth Marketer, Lifecycle Lead, Marketing Analyst, Marketing Specialist, Program Coordinator, VP Brand & Content, VP Demand Generation, Workspace Admin.

### 2.3 Information architecture and navigation (as documented)

**Sections (Product Overview):** Overview, Planning, Budget, Procurement, Analytics, Attribution, Campaigns, Goals, Optimizer ("Budget optimizer"), Builder, Integrations. Later additions: Dashboards (viewer), Admin, Recycle bin (`/trash`), Audit log (`/audit`).

**Routes:**
`/`, `/campaigns`, `/budget`, `/goals`, `/optimizer`, `/analytics`, `/builder` (accepts `?dash=<id>` or `?new`), `/dashboards`, `/planning` (accepts `?plan=spa:<campaignId>` or `bcpir:<id>`), `/procurement`, `/attribution`, `/integrations`, `/admin` (Owner/Admin only), `/trash`, `/audit`. Budget and Procurement accept `?planId=` and `?campaignId=` for scoped views.

**How the navigation evolved:**
1. A top nav with 9 links, full width at `xl` and above, otherwise a hamburger.
2. PRIMARY links plus a "More" dropdown.
3. Final state: a collapsible left sidebar for signed-in users.
   - The sidebar is `bg-graybg` with `border-r`, 244px wide, collapsing to a 72px icon rail with tooltips.
   - Collapse state is stored in `localStorage.zr_sidebar_collapsed`.
   - On mobile it becomes an off-canvas drawer with scrim, Esc to close and scroll lock.
   - MAIN group: Home, Planning, Builder, Dashboards, Budget, Analytics.
   - More group: Campaigns, Goals, Budget optimizer, Procurement, Attribution, Integrations, Recycle bin, Audit log.
   - Admin appears for Owner/Admin.
   - The footer holds account, notification bell and theme toggle.

**Auth-aware home:**
- Logged out: a static marketing landing (Hero, features grid, 3 screenshots from `client/public/marketing/{ai-analysis,dashboard,planning}.png`, sign-up CTA). Nothing is fetched. Every other path redirects to `/`.
- Logged in: Home is the saved-dashboards viewer or a role-aware TeamDashboard.
- The old KPI/AI "OverviewDashboard" still exists in code but is not routed.
- Any component can open the login modal by firing a global `zr:open-auth` CustomEvent.

**Shared UI elements:**
- The floating "Ask ZenReach" assistant appears only when signed in.
- The notification bell appears only when signed in.
- A currency control sits in each `PageHeader` (top-right); on Home it sits above the KPIs.
- A light/dark toggle.
- InfoTip glossary tooltips on finance and marketing terms (`client/src/lib/glossary.js`).

### 2.4 Stated feature list, by module

#### 2.4.1 Overview / AI analysis
- KPI tiles: revenue, spend, ROI, leads. They count up on load and show up/down arrows.
- Revenue-vs-spend trend (30 days), revenue-by-channel donut, planned-vs-actual chart, daily leads.
- **AI Analysis** (`GET /api/insights/ai`):
  - `computeSignals()` builds ROI spread, week-over-week momentum, budget variance, goal pacing, reallocation upside and concentration risk.
  - The LLM turns these into **exactly 4** ranked insights, each with `{title, detail, action, impact, severity}`, using forced JSON.
  - Results are cached for 10 minutes; fallbacks are cached for only 30 seconds.
  - Transient 429/5xx errors are retried.
  - If AI is unavailable it falls back to deterministic `computedInsights()`.
- A legacy rule-based `GET /api/insights` also exists.

#### 2.4.2 Campaigns
- A table of channel, region, status, budget used, leads, pipeline and ROI.
- Filters: Active, Planned, Completed.
- ROI shows green at 2× or better.

#### 2.4.3 Budget
- **"Plan budgets" (top-down) tab:**
  - A pool for the plan, allocations by campaign × quarter, and a committed figure rolled up from tactic budgets and line items.
  - Shows plan-vs-committed, uncommitted and utilization, with a quarter drill-down.
  - Snapshots freeze the numbers for later comparison.
  - Multi-currency.
- **"By channel & region" tab:** planned vs actual from mock data, with a utilization bar.
- A scope selector (Portfolio, Plan, Campaign) filters every KPI, table and chart.
- **Self-heal:** if no allocation references a current campaign, the allocations are rebuilt.
- Story numbers quoted in the docs: $900K top-down, $735K committed, $165K uncommitted, 82% utilization. The channel/region tab is a separate dataset totalling $1.07M at 57% utilization.

#### 2.4.4 Procurement
- Purchase requests (PRs) carry vendor line items (description, qty, unitPrice, amount) and a running total.
- PRs can link to a tactic or campaign.
- **Approval chain by total:**
  - L1 Marketing Ops at ≥ $0;
  - L2 Finance at ≥ $25k;
  - L3 Executive/CMO at ≥ $100k.
  - Steps can be serial or parallel.
- **Lifecycle:** draft → submitted → approved or rejected → po → invoiced → received, with a history log.
- An approvals inbox and a PR drawer.
- **Only the assigned approver may approve or reject.** The server enforces this by `approverId`, then by name. With no accounts it falls back to persona approvers.
- The signed-in user is the actor, taken from `req.user`; client-supplied names are ignored.
- An SP&A cost line can "Raise PR."

#### 2.4.5 Goals
- Target vs actual vs a quarter-end forecast, labelled On track, At risk or Off track.
- A forecast marker sits on each bar.

#### 2.4.6 Budget Optimizer
- Per-channel sliders, with efficiency shown as revenue per $1.
- Projected revenue updates live.
- "Apply AI recommendation" applies a better mix at the same spend; the story quotes a 6–7% lift.
- Endpoints: `GET /api/optimizer`, `POST /api/optimizer/project`.

#### 2.4.7 Analytics (Google Analytics 4)
- A banner shows where data comes from: green means connected live (with the property shown), blue means demo.
- "Active users right now" from GA4 Realtime, refreshing about every 15 seconds.
- 28-day sessions, users, conversions and average engagement.
- A sessions/users trend, traffic by channel and top pages.
- Geography: sessions by country (flag emoji built from ISO alpha-2 `countryId`, top 10) and top regions (top 8). No map library was used, on purpose.
- A processing-delay banner explains that GA standard reports lag by hours, up to 24–48h.
- Falls back to demo data if live calls fail.

#### 2.4.8 Attribution
- Waterfall funnel from impressions to closed-won.
- **9 models:** first, last, last-non-direct, linear, time-decay, U-shaped/position, W-shaped, full-path, custom (data-driven).
- A reallocation ranking and a predictive yearly forecast (on-track / at-risk / off-track).
- All data comes from deterministic synthetic journeys.

#### 2.4.9 Planning
- **Hierarchy:** Plan → Campaign → Program → Tactic, with tactic line items. The newer (uncommitted) client vocabulary in `planLevels.js` calls it a "five-level hierarchy: Plan → Campaign → Program → Tactic → Line item."
- **Tactic detail:** line items, generated media codes, plan-vs-actual performance, custom attributes, and an activity/comment feed with @mentions.
- **Dual-plan register:** the default "Plans" view.
  - Every campaign has `spa:<id>` and `bcpir:<id>` plans, derived from its tactics.
  - Each plan has KPIs, quarter and channel charts, and editable cost lines and business cases.
  - Approval workflow: Draft → Submitted → Published.
  - Reconciles live with Budget and Procurement.
  - Full CRUD: New plan, + Add campaign, edit, delete.
- **Views (the docs disagree; see §15.2):** Plans, Board (Kanban), Calendar, Grid, Budget & Cost, Roadmap, AI Copilot.
- **Board:**
  - Default columns: Backlog → Planned → In progress → Launched. Drag-drop saves immediately.
  - Columns stay on a single non-wrapping row that scrolls sideways with edge fades.
  - Leads and managers can customize: column CRUD (name, hint, colour, WIP limit), templates (Default, Kanban+ with In review and Blocked, Campaign ops), and custom fields (text, number, date, select).
  - Swimlanes by owner, campaign, priority or type.
  - Saved views in `localStorage.zr_planning_views`.
  - Bulk move and delete.
- **Calendar:**
  - Month grid that opens on the current month, with a Today button and marker.
  - Drag to reschedule.
  - Export to `.ics` and PDF (via print CSS).
  - A "Connect calendar" dialog with the public feed URL, a `webcal://` link, and a note that providers can't reach localhost.
- **Grid:** inline edit, bulk status, CSV export, Delete selected, Delete all.
- **Budget & Cost:** a time-phased matrix with spread mechanisms C (Custom), F (First period), L (Last period), E (Evenly distributed), by month or quarter.
- **Import from Jira/Trello:** see §7.8 and §8.12.
- **Closed-loop AI plan engine:**
  1. Generate a plan from a goal (leads) plus budget plus quarter. Allocations are weighted by channel lead-efficiency^1.25, clamped to 8–40%.
  2. Preview, which persists nothing.
  3. Commit, which creates a Campaign, a Program and tactics with `loopId` and `targets`.
  4. Status reports pace-to-goal health: planned / on-track / at-risk / behind / overdue.
  5. Recommendations: reallocate, expedite, flag overspend.
  6. Apply moves budget or raises priority.
  - AI narration has a 9-second timeout and a deterministic fallback.
  - The README now says the engine is "available via the planning-loop API."

#### 2.4.10 Builder and Dashboards
- **Widget types:** KPI, gauge, line, area, bar, donut, funnel, table, each with an SVG preview thumbnail in the picker.
- Point-and-click binding to any catalog source.
- Drag to reorder; resize S/M/L (`w` = 1, 2 or 3).
- Save multiple dashboards. Names must be unique, ignoring case and surrounding whitespace; the server returns 400 on a duplicate and suggests "base 2".
- **Target thresholds** `{value, direction: above|below}` on KPI, gauge, bar, line and area. They draw a dashed target line and green/red zones.
- **Flexible "custom axes" sources** let the user pick an X dimension and Y measures:
  - built-in campaigns by channel, region or status;
  - the uploaded dataset (`cmpFlex`) by channel, type, segment, audience or location.
- **Snapshots:** PNG via html-to-image, stored in `localStorage` under `zenreach.snapshots.v1`. The cap is 6 per dashboard; the oldest is evicted when quota runs out.
- `/dashboards` is a read-only viewer with New and Edit buttons that hand off to the Builder.
- An external campaign dataset can be uploaded (CSV or Excel); it is pre-aggregated into Builder sources.

#### 2.4.11 Ask ZenReach (AI assistant)
- A floating chat on every page.
- Answers are grounded in a compact data context.
- **"Take me there" button:** chosen by keyword-scoring the user's *question*, not the reply. It is shown only on a clear match.
- A Prompts tab of ready-made templates.
- An iMessage-style typing indicator with an enforced, randomized 1.0–2.0 second minimum delay.
- **Guided dashboard builder:** name → focus → widgets (recommended, compact or detailed) → create.
  - It is data-aware: with a dataset loaded it recommends a campaign-performance layout; otherwise a marketing overview.
  - Users can type requests like "add a gauge of budget utilization with target 90."
  - Every widget is assembled from the real catalog, so results are always valid.
- `POST /api/assistant` takes `{message, history, flow}` and returns `{reply, flow, options?, link?, created?}`.

#### 2.4.12 Integrations
- A connector catalog of 13 cards, each with `{id, name, category, description, data: real|sample, sync: inbound|outbound|bidirectional, method: API|File|iCal feed, route, cta}`.
- CSV/Excel tactic import, with a template download and an "Undo import" banner.
- **Google Calendar push sync:**
  - Uses the GA service account (no OAuth).
  - Upserts each dated tactic as an all-day event with a stable SHA-1 event ID, so re-syncs are idempotent.
  - Supports connect, Sync again and disconnect.
  - The connection is persisted in `settings.json`.

#### 2.4.13 Platform-wide features
- Accounts: sign up, log in, profile, logout.
- Admin panel with Org chart, Teams, People and Demo access tabs.
- Notifications: approvals awaiting me, due/overdue tactics I own, @mentions, and pushed assignment alerts. Each user's feed is persisted.
- People picker: Teams/Outlook-style search by name, email or title, backed by `GET /api/org/people?q=`.
- Recycle bin and audit log (see chapter 6).
- Multi-currency: USD, EUR, GBP, CAD, AUD, INR, JPY, SGD. Live ECB rates come via frankfurter.app; everything is stored in USD.
- Light/dark mode.

### 2.5 Real versus simulated (as stated)

| Capability | Status |
|---|---|
| GA4 Data API (standard and Realtime, plus geography) | **Real**, via service account. Demo data when not configured or on failure. |
| Google Calendar push sync | **Real**, one-way, via service account. The user must share the calendar with it (desktop web only). |
| `.ics` calendar feed | **Real**, public and read-only. Live subscription needs a public URL. |
| FX rates | **Real** (ECB via frankfurter.app), cached hourly, static-table fallback offline. |
| AI (chat, analysis, plan narration) | **Real** when keys are set: OpenAI GPT-5 → Azure gpt-4o → Claude, with a built-in deterministic analyst always available. |
| Trello / Jira import | **Real**: server-side REST, one-way and read-only. Credentials are never stored. |
| CSV/Excel import (tactics, dataset) | **Real** (SheetJS converts xlsx/xls/ods to CSV). |
| Accounts, roles, approvals, notifications, recycle bin, audit | **Real** logic, persisted to JSON. |
| Salesforce, SAP, Google Ads, QuickBooks, NetSuite, Coupa, BigQuery, Snowflake, Asana, Jira catalog card, Outlook | **Simulated**: cards labelled "sample", each linking to the page its data would feed. |
| Campaign metrics, timeseries, goals, optimizer, insights | **Synthetic**: deterministic, from the `mockData.js` campaigns "spine". |
| Attribution journeys | **Synthetic**: deterministic in memory, with channel-aligned spend. |
| SP&A / BCPIR plan figures | **Derived** deterministically from tactics (seeded by tactic ID). |
| Finance actuals import | **Roadmap**: "committed-vs-planned" only. |
| SSO/SAML, two-way calendar, OAuth connectors | **Roadmap**. |

## 3. System architecture and technology

### 3.1 Runtime topology

ZenReach has two processes and no database.

| Process | Technology | Port (dev) | Responsibility |
|---|---|---|---|
| **Web client** | React 18 single-page app, built with Vite 5 | 5173 | Every screen. All data comes from `/api/*`. During development Vite proxies `/api` to the API server; in production the SPA must be served from the same origin as `/api`, or behind a proxy that forwards it. |
| **API server** | Node.js (ES modules) + Express 4 | 4000 (`PORT`) | Every business rule, all computation, all external calls, and persistence. It serves only `/api/*` and has no static hosting or SPA fallback. |

Persistence is a directory of JSON files (`server/data/`, or `ZR_DATA_DIR`). Each file is read and rewritten whole on every change; section 4.1 has the details. External services are called only from the server, so no key ever reaches the browser.

**Design principle.** The browser sends small configurations and receives small JSON payloads. Heavy work happens on the server: aggregating uploaded datasets, deriving plan finance, computing attribution and running AI prompts.

### 3.2 Technology stack

| Layer | Choice (version as installed) | Notes |
|---|---|---|
| UI framework | React 18.3, react-router-dom 6.30 | Routes in `client/src/App.jsx` (chapter 12). |
| Component library | MUI 5.18 (+ Emotion 11) | Theme built per light/dark mode in `client/src/theme.js`. |
| Styling | Tailwind CSS 3.4, PostCSS, Autoprefixer | Tailwind *preflight is disabled* so it doesn't fight MUI. `darkMode: 'class'`. CSS variables carry the design tokens. |
| Charts | Apache ECharts 6.1 (tree-shaken) | One `<EChart>` wrapper, with `zr-light` and `zr-dark` themes registered in `client/src/lib/chart.js`. |
| Animation | GSAP 3 | Hero, count-up numbers, staggered reveals. Scroll reveals use IntersectionObserver and respect `prefers-reduced-motion`. |
| Image capture | html-to-image 1.11 | PNG snapshots of dashboards. |
| HTTP server | Express 4.19, cors, morgan, dotenv | JSON body limit 60 MB, so spreadsheets can be uploaded as base64. |
| Spreadsheets | SheetJS `xlsx` 0.18.5 | XLSX/XLS/ODS → CSV for imports. |
| Analytics | `@google-analytics/data` 4.7 | GA4 Data API and Realtime API through a service account. |
| AI | `openai` 6.48, `@anthropic-ai/sdk` 0.68 | Provider cascade: OpenAI (default `gpt-5`) → Azure OpenAI → Anthropic Claude → built-in deterministic analyst. |
| Tooling | `concurrently` | `npm run dev` starts both processes. |

Root scripts: `install:all`, `dev`, `dev:server`, `dev:client`, `build` (client) and `start` (server). The server's `dev` script runs `node --watch index.js`; `start` runs plain `node index.js`, so a server restarted with `start` needs another restart to pick up code changes.

### 3.3 Server module map

All business logic lives in `server/services/`, one module per domain. `server/index.js` wires them to HTTP and applies authentication, role gates, audit logging and notifications.

| Domain | Modules | Persists to |
|---|---|---|
| Persistence | `store.js` | the data directory |
| Identity and organisation | `auth.js`, `org.js` | `users.json`, `teams.json` |
| Collaboration | `notifications.js`, `audit.js`, `recycleBin.js` | `notifications.json`, `audit.json`, `recycleBin.json` |
| Planning hierarchy | `plans.js` (plan, campaign, program), `planning.js` (tactics + line items), `models.js` | `plans.json`, `tactics.json`, `models.json` |
| Dual campaign plans | `campaignPlans.js` (SP&A spend plan + BCPIR return plan, derived from tactics) | stored on the campaign node |
| Work board | `boardConfig.js` (columns, custom fields, templates) | `boardConfig.json` |
| Imports | `importer.js` (CSV tactics), `spreadsheet.js` (XLSX → CSV), `boardImport.js` (Trello/Jira), `boardConnections.js` | `boardConnections.json` |
| Calendars | `calendar.js` (`.ics` feed), `googleCalendar.js` (push sync) | `settings.json` |
| AI planning | `planLoop.js` (closed-loop plan generate → commit → monitor → recommend → apply) | `planLoops.json` |
| Finance | `budgets.js` (pools, allocations, snapshots), `procurement.js` (purchase requests, approval chain), `fx.js` (live exchange rates, in memory) | `budgets.json`, `purchaseRequests.json` |
| Performance data | `mockData.js` (deterministic demo "spine" of 8 campaigns), `campaignDataset.js` (uploaded dataset → aggregated summary) | `campaignDataset.json` |
| Analytics and intelligence | `googleAnalytics.js`, `goals.js`, `insights.js`, `analysis.js`, `optimizer.js`, `attribution.js` | none, computed per request (with caches) |
| Assistant | `assistant.js` (chat analyst), `assistantDashboard.js` (guided dashboard wizard), `assistantWidgets.js` (natural language → widget) | dashboards via `builder.js` |
| Dashboard builder | `builder.js` (catalog, data bundle, dashboard CRUD) | `dashboards.json` |

### 3.4 Client structure

| Folder | Contents |
|---|---|
| `client/src/pages/` | One component per route: Dashboard (Home), Planning, Budget, Procurement, Analytics, Attribution, Campaigns, Goals, Optimizer, Builder, Dashboards, Integrations, Admin, RecycleBin, AuditLog. |
| `client/src/components/` | Shared UI (Layout, PageHeader, Panel, KpiCard, EChart, InfoTip, StateWrap, PeoplePicker, NotificationBell, AiAssistant, WidgetRenderer, ScopePicker, SnapshotPanel and others), plus `auth/`, `planning/` and `team/` sub-folders. |
| `client/src/context/` | `AuthContext` (session and user), `CurrencyContext` (global currency and live rates), `ThemeModeContext` (light/dark). |
| `client/src/lib/` | `format.js` (money/number formatting in the active currency), `chart.js` (ECharts setup), `glossary.js` (term definitions), `exporters.js` (CSV/ICS/print), `files.js` (upload reader), `snapshot.js` and `snapshotStore.js` (PNG snapshots). |
| `client/src/api.js` | The single fetch wrapper. One method per endpoint (138 methods); it attaches the bearer token. |
| `client/src/hooks/useApi.js` | A minimal `{ data, loading, error }` data-fetching hook. |

### 3.5 Two hierarchies and one data spine

Three structures hold the data together. Most cross-module behaviour follows from how they connect.

1. **The organisation chart:** users and teams (`users.json`, `teams.json`). Admin → managers → leads → teams → members. It decides whose Home dashboard shows what and how far the audit log reaches.
2. **The plan tree:** Plan → Campaign → Program → Tactic → Line item (`plans.json`, `tactics.json`). It is the source of truth for planned spend.
   - Budget "committed" figures, SP&A/BCPIR plan figures, procurement reconciliation, the board, the calendar and team work roll-ups are all derived from it.
   - Org and plan meet through `team.campaignIds`.
3. **The performance spine:** eight deterministic demo campaigns in `mockData.js`. Overview KPIs, budget by channel/region, revenue by channel, goals, insights, the optimizer, the AI context and the Builder's campaign sources are all derived from them.
   - An uploaded campaign dataset adds further Builder sources.
   - Google Analytics 4 provides web-traffic data separately.

### 3.6 External integrations

| Integration | Direction | Mechanism | Configured by | Fallback |
|---|---|---|---|---|
| Google Analytics 4 | inbound | Data API + Realtime API through a service account | `GA_PROPERTY_ID` + `GOOGLE_APPLICATION_CREDENTIALS` or `GOOGLE_CREDENTIALS_JSON` | Deterministic demo data, labelled as such |
| Google Calendar | outbound push | Calendar REST API through the same service account; the user shares their calendar with it | same key + a calendar ID entered in the UI | n/a (the feature reports why it can't connect) |
| Calendar feed (`.ics`) | outbound | Public, unauthenticated iCalendar feed | none | n/a |
| Exchange rates | inbound | frankfurter.app (ECB reference rates), cached hourly in memory | none | A static rate table |
| OpenAI / Azure OpenAI / Anthropic | outbound | Chat completions | `OPENAI_API_KEY` (+`OPENAI_MODEL`); `AZURE_OPENAI_API_KEY`, `AZURE_OPENAI_ENDPOINT`, `AZURE_OPENAI_DEPLOYMENT_NAME` (+`AZURE_OPENAI_API_VERSION`); `ANTHROPIC_API_KEY` (+`ANTHROPIC_MODEL`) | A deterministic rule-based analyst and computed insights |
| Trello / Jira | inbound, one-off import | REST APIs called by the server with credentials the user types in; credentials are never stored | entered per import | n/a |
| Other catalog connectors (Salesforce, SAP, Google Ads, QuickBooks, NetSuite, Coupa, Outlook, BigQuery, Snowflake, Asana) | — | Catalog cards labelled as sample data, each linking to the page its data would feed | none | Sample data |

### 3.7 Request lifecycle

1. The client calls `api.someMethod()`. The wrapper sends `Authorization: Bearer <token>` (token from `localStorage.zr_token`) and parses `{ ok, data | error }`.
2. Express runs `cors()` → `express.json({ limit: '60mb' })` → `morgan('dev')` → the global auth wall. The wall returns 401 unless the route is on the public allow-list or the token resolves to a user.
3. Route-level gates run: `requireRole('Editor' | 'Admin')` or `requirePlanner`. Domain approval guards run inside handlers.
4. The service validates the input (throwing a `ValidationError`, which becomes HTTP 400 with the message), reads and writes its JSON files, and returns plain objects.
5. The handler writes an audit event, pushes notifications to affected users, and responds `{ ok: true, data }`.

### 3.8 Deployment

- **Current:** two Node processes on one machine (`npm run dev`), or `npm run build` + `npm start` with a reverse proxy. Machine-local JSON data. A single workspace.
- **Documented production target (Azure):**
  - Static Web Apps for the SPA, behind an Application Gateway with WAF.
  - A VM Scale Set (2–3 × D2s v5) running the API under Docker/PM2/Nginx.
  - Azure Database for PostgreSQL Flexible Server (zone-redundant).
  - Blob Storage for uploads, snapshots and exports; optional Redis.
  - Key Vault with Managed Identity; GitHub Actions + Azure Container Registry; Application Insights.
  - Bicep or Terraform, with dev → staging → prod environments and blue-green deploys.
- **Migration path:** relational schema → one-time importer from `server/data/*.json` → feature flag → parity check → cut-over. Chapter 15 includes the target schema.

## 4. Cross-cutting conventions and the permission model

This chapter collects the rules every module shares. Later chapters assume them and add only module-specific detail.

### 4.1 Storage: the JSON file store

- **Location:** `server/data/`, or the directory in `ZR_DATA_DIR`. The directory is created on demand.
- **`readJson(file, fallback)`** is synchronous. It returns `fallback` if the file is missing, empty, whitespace-only or fails to parse.
  - **Gap:** a corrupt file is silently treated as empty, and the next write overwrites it.
- **`writeJson(file, data)`** writes the whole file synchronously as pretty JSON (2-space indent). It is not atomic and takes no lock. Every mutation is a whole-file read-modify-write, so concurrent writers race.
  - This is why bulk operations, such as deleting many tactics, are implemented as **one** read-modify-write rather than many single writes.
- **The sixteen stores:**

| File | Owner module | Holds |
|---|---|---|
| `users.json` | auth | users (with salted scrypt hashes) and the `sessions` map |
| `teams.json` | org | teams and the `orgSeeded` flag |
| `notifications.json` | notifications | per-user pushed items and read state |
| `audit.json` | audit | up to 5,000 audit events |
| `recycleBin.json` | recycleBin | soft-deleted entity snapshots |
| `plans.json` | plans | plan / campaign / program nodes (flat, parent-linked) |
| `tactics.json` | planning | tactics, with embedded line items, media codes, comments, approval history |
| `models.json` | models | planning models (waterfall rates, deal size, money rules) |
| `boardConfig.json` | boardConfig | board columns and custom fields |
| `boardConnections.json` | boardConnections | imported Trello/Jira boards |
| `planLoops.json` | planLoop | committed AI plans |
| `budgets.json` | budgets | pools, allocations, snapshots |
| `purchaseRequests.json` | procurement | purchase requests with lines, approvals, history |
| `dashboards.json` | builder | saved Builder dashboards |
| `campaignDataset.json` | campaignDataset | the aggregated uploaded campaign dataset |
| `settings.json` | googleCalendar | the Google Calendar connection |

- **Seeding and self-healing.** Stores seed themselves the first time they are read. Several also repair themselves:
  - plan nodes and tactics written before the five-level model are upgraded in place on first read (chapter 7);
  - budget allocations that reference no current campaign are rebuilt (chapter 9);
  - team–campaign links are re-mapped by campaign name (chapter 6).
- **At startup** the server seeds the demo organisation, backfills a curated audit history (only when the log is empty and accounts exist), and adds a second demo plan ("EMEA Field Marketing") if it is missing. Section 5.1 gives the exact boot order.

### 4.2 Identifiers and timestamps

- `uid(prefix)` = `` `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}` ``. Examples: `plan-…`, `campaign-…`, `program-…`, `tac-…`, `li-…`, `pr-…`, `tok-…`, `trash-…`.
  - Purchase-order and invoice numbers reuse the timestamp segment (chapter 9).
- `nowIso()` = `new Date().toISOString()`. Every record carries `createdAt` and, where it can change, `updatedAt`.
- **Rebuild note:** session tokens come from this non-cryptographic generator. A rebuild should use a CSPRNG for tokens.

### 4.3 API envelope and errors

- **Success:** HTTP 200 with `{ ok: true, data }`. The one exception is the iCalendar feed, which returns `text/calendar`.
- **Handled errors:** `{ ok: false, error: "<human-readable message>" }` with a status:
  - **400** validation (business rules throw a `ValidationError` carrying status 400);
  - **401** not signed in or session expired;
  - **403** role or approver gate;
  - **404** unknown id;
  - **500** unexpected.
- Error messages are written for end users and shown verbatim in toasts and inline alerts. Reproduce them exactly.
- **Gaps:**
  - malformed JSON bodies and uncaught exceptions return Express's default HTML error page;
  - there is no rate limiting, helmet or compression.

### 4.4 Authentication and sessions

- **Sign-up:** name, email, password (minimum 6 characters), optional company. The first account becomes Owner; later ones become Editor. Because the demo organisation is seeded at boot, a fresh install already has users.
- **Login** returns `{ token, user }`. An unknown email and a wrong password both give "Incorrect email or password."
- **Passwords** use a per-user 16-byte random salt and `scrypt` (64-byte key, Node defaults N=16384, r=8, p=1), compared with a timing-safe comparison. Hashes never leave the server.
- **Sessions** are opaque bearer tokens mapped to user ids in `users.json`.
  - They never expire. Logout removes only the presented token.
  - The client stores the token in `localStorage.zr_token` and sends `Authorization: Bearer <token>`.
- **The global auth wall:** every `/api/*` request needs a valid session, except these public routes:
  - `POST /api/auth/login`, `POST /api/auth/signup`, `GET /api/auth/me`, `POST /api/auth/logout`;
  - `GET /api/org/personas` (the demo persona switcher on the login screen);
  - `GET /api/currency`, `GET /api/health`;
  - `GET /api/planning/calendar.ics` (calendar clients can't send tokens).
  - Anything else answers 401 "Please sign in to continue."
- **Signed-out client:** only the marketing landing page renders at `/`; every other path redirects to `/`.
- **Demo personas:** the login modal offers "Explore as a demo persona". It lists one admin, managers, leads and members of the seeded org, who all share one demo password (a server constant).

### 4.5 Roles and the permission matrix

Access is decided on two independent axes.

| Axis | Values | Decides |
|---|---|---|
| **Workspace role** (`user.role`) | `Owner` (rank 4) · `Admin` (3) · `Editor` (2; legacy `Member` = 2) · `Viewer` (1) | Whether a user may write, and which writes. |
| **Org role** (`user.orgRole`) | `admin` · `manager` · `lead` · `member` · none | Where the user sits in the org chart: what Home shows, how far the audit log reaches, board customisation, and tactic approval rights. |

**Server gates:**

| Gate | Rule | Failure |
|---|---|---|
| Signed in | Valid bearer token | 401 "Please sign in to continue." |
| `requireRole('Editor')` | rank ≥ 2 | 403 "This action requires the Editor role or higher." |
| `requireRole('Admin')` | rank ≥ 3 | 403 "This action requires the Admin role or higher." |
| `requirePlanner` (board customisation) | org role `lead` or `manager`, **or** workspace rank ≥ Admin. Org role `admin` alone does not qualify. | 403 "Board customization is available to team leads and managers." |
| Plan / PR approver lock | When an approver was named at submit time, only that user may approve or reject. Matched by user id, falling back to name. | 403 "Only {name} can approve or reject this plan." |
| Tactic approval (no named approver) | workspace rank ≥ Admin, or org role `lead`, `manager` or `admin` | 403 "You don’t have approval rights for tactics — ask a team lead or manager." |

**What each kind of user can do:**

| Capability | Viewer | Editor | Admin / Owner | Notes |
|---|---|---|---|---|
| Read every module | ✓ | ✓ | ✓ | Audit log rows are scoped by org role (below). |
| Create and edit plans, campaigns, programs, tactics, line items | — | ✓ | ✓ | Required fields block *submitting*, never saving. |
| Delete a plan, campaign or program (cascades to children) | — | — | ✓ | Goes to the recycle bin. |
| Delete tactics and line items | — | ✓ | ✓ | Goes to the recycle bin. |
| Submit a tactic or campaign plan for approval | — | ✓ | ✓ | |
| Approve / decline a tactic | — | if lead/manager, or the named approver | ✓ | A comment is required to decline. |
| Approve / reject a campaign plan or purchase request | — | the named approver | the named approver | With no named approver, any Editor+ may act. |
| Create, submit and fulfil purchase requests | — | ✓ | ✓ | |
| Edit budgets (pools, allocations, snapshots) | — | ✓ | ✓ | |
| Customise the board (columns, fields, templates) | — | lead/manager only | ✓ | |
| Import CSV/Excel, Jira, Trello; sync Google Calendar | — | ✓ | ✓ | |
| Restore or purge from the recycle bin | — | ✓ | ✓ | |
| Edit planning models | — | — | ✓ | |
| Manage users' roles, org placement and teams (Admin page) | — | — | ✓ | The last Owner cannot be demoted. |
| Create, edit and delete Builder dashboards | ✓ (gap) | ✓ | ✓ | No server role gate; the client hides the controls from Viewers. |
| Use the optimizer, the assistant, notifications, own profile | ✓ | ✓ | ✓ | |

**Client gating** mirrors the server:
- `canEdit = !user || user.role !== 'Viewer'` hides write controls;
- the Admin navigation item and page need Owner or Admin;
- org managers, leads and members get a team-oriented Home (chapter 6).
- The client does not refresh the user object after login, so role changes apply in the UI only after a reload.

**Audit visibility by org role:**
- `admin` (or workspace Admin/Owner) sees everything;
- `manager` sees their direct reports and those reports' teams;
- `lead` sees their teams;
- everyone else sees only their own actions.
- Section 6.6 has the exact rules.

### 4.6 Audit logging

- Every successful mutation calls `audit(req, action, opts)` after the change is made. The action identifier, such as `tactic.approve`, `plan.delete`, `pr.submit` or `lineitem.update`, maps through a registry to a category, verb, noun and severity.
- Each event snapshots the actor (id, name, roles, initials, avatar gradient), the entity label (cut to 160 characters), a generated summary, and sanitised metadata.
- The log keeps the newest 5,000 events.
- A logging failure never breaks the user's action.
- Sections 5.4 and 6.6 list the full action catalogue.

### 4.7 Notifications

- **Two sources merge into each user's bell:**
  - **Derived** items computed on read: approvals waiting on the user, tactics they own that are due or overdue, and @mentions in comments.
  - **Pushed** items stored per recipient (capped at 50): assignment and approval alerts sent when someone makes the user an owner, approver or requester, or decides on their submission.
- A user is never notified about their own action.
- Opening the bell marks everything seen. Section 6.5 has the read/unread rules.

### 4.8 Soft delete: the recycle bin

- Deletes of plans, campaigns, programs (each with their whole subtree and tactics), tactics, line items, dashboards and purchase requests do not destroy data. A snapshot goes to the recycle bin and the live record is removed.
- Restore re-inserts the snapshot with its original ids.
- **Hive9 rules:**
  - deleting a parent deletes all its children;
  - a child can only be restored while its parent exists;
  - restored tactics come back in **Created** status and must be re-submitted;
  - restored campaign plans come back as Draft.
- Section 6.7 covers purge and empty.

### 4.9 Browser storage and global events

| Key / event | Owner | Value / purpose |
|---|---|---|
| `zr_token` | api.js / AuthContext | Session bearer token |
| `zr_theme` | ThemeModeContext | `light` or `dark`. The first visit follows the OS setting. |
| `zr_currency` | CurrencyContext | Active display currency code |
| `zr_sidebar_collapsed` | Layout | `1` or `0` |
| `zr_planning_view` | Planning | Last Planning tab: `plans`, `grid`, `board`, `calendar` or `budget` |
| `zr_planning_plans` | Planning | Selected plan chip (`all` or a plan id) |
| `zr_planning_groupby` | Planning | Board swimlane grouping |
| `zr_planning_views` | Planning | Saved views (JSON array) |
| `zr_import_jira_prefs` | ImportBoardDialog | `{ baseUrl, email }` for Jira. The API token is never stored. |
| `zenreach.snapshots.v1` | snapshotStore | Dashboard PNG snapshots, up to 6 per dashboard |
| `ga_mid` | tracking-test.html | GA4 measurement ID for the traffic generator |
| `zr:open-auth` (window event) | any component → AccountControls | Opens the login/sign-up modal (`detail.tab`). |
| `zr:auth-modal` (window event) | AccountControls → Hero | Broadcast when the modal opens or closes, so the hero pauses its animation. |

### 4.10 Environment variables

| Variable | Purpose | Default |
|---|---|---|
| `PORT` | API port | `4000` |
| `ZR_DATA_DIR` | Data directory (isolated test or demo instances) | `server/data` |
| `GA_PROPERTY_ID` | GA4 property | unset, so demo analytics |
| `GOOGLE_APPLICATION_CREDENTIALS` / `GOOGLE_CREDENTIALS_JSON` | Service-account key (file path or inline JSON) for GA4 and Google Calendar | unset |
| `OPENAI_API_KEY`, `OPENAI_MODEL` | Assistant, first provider | model `gpt-5` |
| `AZURE_OPENAI_API_KEY`, `AZURE_OPENAI_ENDPOINT`, `AZURE_OPENAI_DEPLOYMENT_NAME`, `AZURE_OPENAI_API_VERSION` | Assistant (second provider) and AI analysis (first provider) | version `2024-08-01-preview` |
| `ANTHROPIC_API_KEY`, `ANTHROPIC_MODEL` | Claude fallback | model `claude-opus-4-8` |

### 4.11 Formatting and currency

- Every amount is stored in USD.
- The client keeps one active currency (USD, EUR, GBP, CAD, AUD, INR, JPY or SGD), using live ECB rates. Every formatter converts at render time:
  - compact form, e.g. `$1.2M`;
  - full form, e.g. `$1,234,567`.
- Changing the currency re-renders the whole app without remounting pages.
- Chapter 9 has the exact formatting rules.

## 5. HTTP API reference

**Source files:** `server/index.js`, `server/package.json`, `server/.env.example`, `package.json`, `client/vite.config.js`, `client/src/api.js`, `client/src/hooks/useApi.js`.

**Route count:** 141 Express routes in `server/index.js`, all under `/api/`, including 3 back-compat `/api/planning/initiatives` aliases. The client `api` object has 138 methods.

### 5.1 Server bootstrap

#### 5.1.1 Packages and scripts
- **Root `package.json`** (`zenreach`, private)
  - devDependency: `concurrently ^8.2.2`.
  - Scripts:
    - `install:all` = `npm --prefix server install && npm --prefix client install`
    - `dev:server` = `npm --prefix server run dev`
    - `dev:client` = `npm --prefix client run dev`
    - `dev` = `concurrently -n server,client -c blue,magenta "npm run dev:server" "npm run dev:client"`
    - `build` = `npm --prefix client run build`
    - `start` = `npm --prefix server start`
- **`server/package.json`** (`zenreach-server`)
  - `"type": "module"` (ESM), main `index.js`.
  - Scripts: `start` = `node index.js`; `dev` = `node --watch index.js`.
  - Dependencies:
    - `@anthropic-ai/sdk ^0.68.0`
    - `@google-analytics/data ^4.7.0`
    - `cors ^2.8.5`
    - `dotenv ^16.4.5`
    - `express ^4.19.2` (Express 4: async rejections are not caught automatically)
    - `morgan ^1.10.0`
    - `openai ^6.48.0`
    - `xlsx ^0.18.5`
- **Client dev server (`client/vite.config.js`)**
  - `@vitejs/plugin-react`, `server.port: 5173`.
  - `proxy: { '/api': 'http://localhost:4000' }`: a simple string proxy with no changeOrigin or rewrite.
  - The client always calls the relative path `/api...`.

#### 5.1.2 Environment variables
Names only. No secret values are copied here.

| Var | Used in | Meaning | Default / behaviour |
|---|---|---|---|
| `PORT` | index.js | API listen port | `process.env.PORT \|\| 4000` |
| `GA_PROPERTY_ID` | googleAnalytics.js | GA4 numeric property id | GA counts as "configured" only when this is set AND (`GOOGLE_APPLICATION_CREDENTIALS` or `GOOGLE_CREDENTIALS_JSON`) is set. Otherwise GA endpoints return demo data. |
| `GOOGLE_APPLICATION_CREDENTIALS` | googleAnalytics.js, googleCalendar.js | Path to a service-account JSON key | A relative path resolves from the project root, not cwd. The .env.example placeholder is `your-service-account-key.json`. The same key is used for Google Calendar sync. |
| `GOOGLE_CREDENTIALS_JSON` | googleAnalytics.js, googleCalendar.js | Inline service-account JSON (alternative to a key file) | Takes precedence for the GA client when set. Not in .env.example. |
| `OPENAI_API_KEY` | assistant.js | OpenAI key (the assistant's first choice) | Empty means skipped |
| `OPENAI_MODEL` | assistant.js | OpenAI model | `'gpt-5'` |
| `AZURE_OPENAI_API_KEY`, `AZURE_OPENAI_ENDPOINT`, `AZURE_OPENAI_DEPLOYMENT_NAME` | assistant.js, analysis.js | Azure OpenAI (all three required) | Not in .env.example, but present in the real `server/.env` |
| `AZURE_OPENAI_API_VERSION` | assistant.js, analysis.js | Azure API version | `'2024-08-01-preview'` |
| `ANTHROPIC_API_KEY` | assistant.js, analysis.js | Claude fallback | Empty means skipped |
| `ANTHROPIC_MODEL` | assistant.js, analysis.js | Claude model override | `'claude-opus-4-8'` |
| `ZR_DATA_DIR` | store.js | Directory for the JSON file store | `server/data` (`path.join(__dirname,'..','data')`). It exists so a second instance can run on isolated data. |

- `.env` is loaded by `import 'dotenv/config'` on line 1, from `process.cwd()`. That is `server/` when the server is run through the npm scripts.
- **Provider order for `POST /api/assistant`:** OpenAI → Azure → Anthropic → built-in rule-based analyst. It cascades on failure.
- **Provider order for `GET /api/insights/ai`:** Azure → Anthropic → computed. It has no OpenAI branch.

#### 5.1.3 Startup sequence (exact order)
1. `dotenv/config` loads `.env`.
2. ESM imports evaluate. Side effect: `fx.js` runs `getRates().catch(() => {})` at module load, which warms the FX cache from frankfurter.app with a 6 s timeout. Other services seed lazily the first time they are read:
   - `listNodes()` seeds `plans.json` and normalises legacy nodes that lack `dateMode`/`planBudget`.
   - `listTactics()` migrates the legacy `initiatives.json` into `tactics.json` or seeds it, then upgrades tactics that lack `approvalStatus`/`goals`.
3. `seedOrg()` runs bare, with no try/catch, so a throw here crashes boot. It is idempotent: it skips when `db.meta.orgSeeded && db.teams.length`. It seeds the demo org (2 managers → 4 leads → 6 teams → 18 members) on the demo domain using `DEMO_PASSWORD`, and promotes any existing workspace Owner to org admin.
4. `try { seedAudit(); } catch (err) { console.warn('seedAudit skipped:', err.message); }`. This runs only when the audit log is empty and accounts exist, and backdates a curated history over about 2 weeks.
5. `const app = express()`, then middleware (1.4), then all routes.
6. After all routes: `try { ensureMultiPlanDemo(); } catch (err) { console.warn('ensureMultiPlanDemo skipped:', err.message); }`. This is additive and idempotent. It creates the plan "EMEA Field Marketing" (FY2026), campaign "EMEA Demand Gen", program "Field & Events" and 4 tactics, but only if that plan name doesn't exist.
7. `app.listen(PORT, ...)` logs `\n  ZenReach API running on http://localhost:${PORT}  (${mode})\n`. `mode` is `'LIVE Google Analytics'` when `analyticsStatus().configured` is true, otherwise `'DEMO mode'`.

#### 5.1.4 Middleware stack (in order)
1. `app.use(cors())`: default cors, so any origin with `Access-Control-Allow-Origin: *`. No credentials config.
2. `app.use(express.json({ limit: '60mb' }))`. The comment says the large limit is so CSV and spreadsheet datasets upload in one request. Malformed JSON goes to Express's default error handler, which returns a 400 HTML page rather than the JSON envelope.
3. `app.use(morgan('dev'))`: coloured `METHOD url status time ms - length` lines. It is registered after the JSON parser, so requests that fail to parse are not logged by morgan.
4. The global auth wall (§5.2.3).
5. Routes.

Not present:
- No `express.static` and no SPA fallback. The server serves only `/api/*`, and other paths get Express's default 404. Production needs separate static hosting, or same-origin proxying of `/api`.
- No custom error-handling middleware. A thrown exception in a route without try/catch gets Express's default HTML 500, which includes a stack trace when `NODE_ENV` isn't `production`.
- No rate limiting, no helmet, no compression.

#### 5.1.5 Persistence
Everything is synchronous JSON files via `store.js`:
- `readJson(file, fallback)` returns the fallback on any error or empty file.
- `writeJson` writes pretty JSON with 2-space indentation.
- `uid(prefix)` = `${prefix}-${Date.now().toString(36)}-${random5}`.
- `nowIso()`.

Files in `server/data/`: `audit.json`, `boardConfig.json`, `boardConnections.json`, `budgets.json`, `campaignDataset.json`, `dashboards.json`, `models.json`, `notifications.json`, `planLoops.json`, `plans.json`, `purchaseRequests.json`, `recycleBin.json`, `settings.json`, `tactics.json`, `teams.json`, `users.json`.

### 5.2 Middleware and helpers (exact behaviour)

#### 5.2.1 Envelope
- `const ok = (res, data) => res.json({ ok: true, data })`. Every success is HTTP 200 with `{ ok: true, data }`. If `data` is `undefined`, the key is dropped and the body is `{ ok: true }`.
- The only non-JSON success is `GET /api/planning/calendar.ics`.
- Every handled error is `{ ok: false, error: '<message>' }` with a status code.

#### 5.2.2 `bearer(req)`
Returns `h.slice(7)` when the header starts with the literal, case-sensitive `'Bearer '`; otherwise `''`. Tokens are opaque `tok-...` ids, stored as `sessions[token] = userId` in `users.json`. They never expire and are removed only by logout.

#### 5.2.3 Global auth wall and `PUBLIC_API`
```js
const PUBLIC_API = new Set([
  'POST /api/auth/login',
  'POST /api/auth/signup',
  'GET /api/auth/me',
  'POST /api/auth/logout',
  'GET /api/org/personas', // demo persona switcher on the login screen
  'GET /api/currency',     // public ECB FX rates (loaded before sign-in, not private)
  'GET /api/health',
]);
```
The wall runs in this order:
1. If `!req.path.startsWith('/api/')`, call `next()`.
2. If `PUBLIC_API.has(\`${req.method} ${req.path}\`)`, call `next()`.
3. Hard-coded exception: if `GET` and `req.path === '/api/planning/calendar.ics'`, call `next()`. This feed is public because calendar clients can't send tokens.
4. Otherwise, if `!me(bearer(req))`, respond **401** `{ ok:false, error: 'Please sign in to continue.' }`.

The match is an exact, case-sensitive string. Express routing is case-insensitive and tolerant of trailing slashes, so `GET /api/health/` routes to the handler but gets a 401 from the wall.

**Consequence:** every non-public route needs a valid session. There is no anonymous access to data any more.

#### 5.2.4 `requireRole(min)`
Role ranks come from `auth.js`: `ROLE_RANK = { Viewer: 1, Editor: 2, Member: 2, Admin: 3, Owner: 4 }`. `roleRank(role)` returns `ROLE_RANK[role] || 0`. Assignable roles are `ROLES = ['Owner','Admin','Editor','Viewer']`. Legacy `Member` ranks the same as Editor.

The middleware does this:
1. If `userCount() === 0 || !token`, set `req.user = null` and call `next()`. This is the "demo / anonymous" branch. **Behind the global wall it is effectively dead code**: any request reaching a `requireRole` route already carries a valid token, and a valid token implies `userCount() > 0`. It is kept for consistency with the old open-demo behaviour.
2. `user = me(token)`. If null, return **401** `'Your session has expired — please sign in again.'`.
3. If `roleRank(user.role) < roleRank(min)`, return **403** `` `This action requires the ${min} role or higher.` ``. In practice the messages are "This action requires the Editor role or higher." and "This action requires the Admin role or higher."
4. Otherwise set `req.user = user` (the public user shape, see §5.3.22) and call `next()`.

Routes without `requireRole` do NOT set `req.user`. Their `audit()` call resolves the actor from the token itself.

#### 5.2.5 `requirePlanner`
- Uses the same anonymous/demo branch as `requireRole`, and returns the same 401 message for an invalid token.
- `isPlanner = ['lead','manager'].includes(user.orgRole) || roleRank(user.role) >= roleRank('Admin')`.
- If not a planner, it returns **403** `'Board customization is available to team leads and managers.'`.
- An orgRole of `'admin'` alone does NOT qualify. The user needs permission role Admin or Owner.

#### 5.2.6 Effective permission matrix

| Caller | Can call |
|---|---|
| Anonymous | Only the 7 `PUBLIC_API` routes and the `.ics` feed |
| Signed-in **Viewer** | All reads, plus every write that has no role gate (listed below) |
| **Editor/Member** | Additionally, every `requireRole('Editor')` route |
| **Admin/Owner** | Additionally, `requireRole('Admin')` routes: model create/update, delete plan nodes, `/api/auth/users*`, org team writes, `/api/org/users*` |
| **Planner** | Board-config writes need a team lead or manager (by orgRole), or Admin+ |

Writes with no role gate, open to any signed-in user including a Viewer:
- `POST/PUT/DELETE /api/builder/dashboards*`
- `POST /api/optimizer/project`
- `POST /api/assistant`
- `POST /api/notifications/read`
- `PUT /api/auth/profile`
- `POST /api/auth/logout`

#### 5.2.7 `notifyAssignment({ actor, toUserId, type = 'assignment', title, text, route = '/' })`
- Returns early if `!toUserId || !title`.
- Never notifies the actor themselves: it returns if `actor && actor.id === toUserId`.
- Calls `pushNotification({ toUserId, type, title, text, route, from: actor?.name || '' })` inside a try/catch. On failure it logs `console.warn('notifyAssignment failed:', err.message)`.
- `pushNotification` stores the item under the recipient's user id as `{ id: 'push-<b36>-<rand5>', type, title, text, at, route, from, authored: true }`. It prepends the item and caps the list at 50.
- Types in use: `'assignment'` and `'approval'`.

#### 5.2.8 `audit(req, action, opts = {})`
- Actor: `opts.actor !== undefined ? opts.actor : (req?.user || me(bearer(req)) || null)`. An explicit `actor: null` stays null.
- Calls `recordAudit({ ...opts, actor, action })` inside a try/catch, warning `'audit failed:'` on failure. A logging failure never breaks the action.
- `opts` keys: `label`, `entity`, `entityId`, `summary`, `meta`.
- `record()` maps the action through the `ACTIONS` registry to `{category, verb, noun, severity}`. Unknown actions fall back to `DEFAULT_ACTION = { category: 'other', verb: 'updated', noun: 'item', severity: 'info' }`.
- `label` is truncated to 160 characters.
- `summary` defaults to `` `${verb}${noun ? ` ${noun}` : ''}${label ? ` “${label}”` : ''}` ``.
- `meta` is sanitised: nested objects and nulls are dropped, the `at` key is dropped, and values longer than 200 characters are cut with `…`.
- `entity` defaults to `spec.noun`.
- Each event snapshots the actor: `actorId`, `actorName` (`'A teammate'` when there is no actor), `actorRole`, `actorOrgRole`, `actorInitials`, `actorGradient`.
- The log is capped at `MAX_EVENTS = 5000`, keeping the newest.

#### 5.2.9 Approval guards
- **`approvalGuard(approval, user)`**, used for campaign plans and delegated to by the tactic guard:
  - If there is no `approverId` and no `approver`, it returns `{ ok: true }`.
  - It passes when `apId ? apId === user?.id : apName === user?.name`. It prefers the id and falls back to the name.
  - Otherwise it returns `{ ok:false, error: \`Only ${apName || 'the assigned approver'} can approve or reject this plan.\` }`.
- **`tacticApprovalGuard(t, user)`**:
  - If `!user`, it returns `{ok:true}` (demo mode).
  - If the tactic's `approval` has an `approverId` or `approver`, it delegates to `approvalGuard`. The error text then still says "…this plan."
  - Otherwise `canApprove = roleRank(user.role) >= roleRank('Admin') || ['lead','manager','admin'].includes(user.orgRole)`.
  - Failure message: `'You don’t have approval rights for tactics — ask a team lead or manager.'` (with a curly ’).
- **`decideRoute(decision)`**: the factory behind the tactic approve and decline routes (§5.3.9).

#### 5.2.10 Other helpers
- **`filterFromQuery(req)`** returns `{ planId, campaignId, programId, planIds }`. Each is `req.query.X || undefined`. `planIds` is a comma-separated multi-plan overlay, split server-side; whitespace is trimmed and empty entries are ignored.
- **`fail(res, err)`**: `status = err?.status || 500`. It logs with `console.error(err)` when `status >= 500` and responds `{ ok:false, error: err?.message || 'Something went wrong.' }`. `ValidationError` (from plans.js, `this.status = 400`) therefore becomes 400, and any other Error becomes 500.
- **`teamsById()`**: `Object.fromEntries(listTeams().map(t => [t.id, { memberIds: t.memberIds || [], leadId: t.leadId || null }]))`. It is passed to `validateNode` to enforce "plan owner must be on one of the plan's teams".
- **`resolveProgramId(body)`**:
  - Returns `body.programId` if set; null if there is no `campaignId`.
  - Returns null if the campaign is missing or isn't level `campaign`.
  - Otherwise returns the first program whose `parentId` is the campaign. If none exists, it creates `{ level:'program', parentId, name: \`${campaign.name} — General\`, owner, ownerId }` and returns its id.
  - When it returns null, the tactic validation then fails with "A tactic must sit inside a program…".
- **`prErr(res, err)`** returns 400 `{ ok:false, error: err.message }` for procurement.
- **`authError(res, err)`** returns 400 `{ ok:false, error: err.message }` for auth and org.
- **Unused imports in index.js:** `ask`, `deleteDashboard`, `deleteTactic`, `deleteTactics`, `deleteRequest`, `getUserById`.

### 5.3 Endpoint catalog (141 routes)

Legend for the Auth column:
- **PUBLIC**: in the allow-list.
- **SIGNED-IN**: the global wall only; any role, including Viewer.
- **Editor+**: `requireRole('Editor')`.
- **Admin+**: `requireRole('Admin')`.
- **Planner**: `requirePlanner`.

All routes except PUBLIC ones and the .ics feed can also return 401 `'Please sign in to continue.'`. Editor+ and Admin+ routes can also return 403 `This action requires the <Role> role or higher.`. "Uncaught → HTML 500" means the handler has no try/catch.

**Routes per domain:**

| Domain | Routes |
|---|---|
| Health | 1 |
| Executive and analytics | 13 |
| Assistant | 2 |
| Builder | 7 |
| Plans hierarchy and models | 12 |
| Board config | 4 |
| Campaign dual-plans | 5 |
| Board, summary and ICS | 3 |
| Tactics | 16 |
| Initiatives aliases | 3 |
| Currency | 1 |
| Budgets | 8 |
| Procurement | 10 |
| Attribution | 6 |
| Notifications | 2 |
| Audit | 1 |
| Trash | 4 |
| Google Calendar | 4 |
| Dataset | 2 |
| Imports | 8 |
| PlanLoop | 6 |
| Auth | 7 |
| Org | 16 |
| **Total** | **141** |

#### 5.3.1 Health (1)
| Method | Path | Auth | Params | Response `data` | Side effects | Errors |
|---|---|---|---|---|---|---|
| GET | /api/health | PUBLIC | none | `{ status: 'up', time: <ISO> }` | none | none |

#### 5.3.2 Executive dashboard and analytics (13)
All are SIGNED-IN, with no audit and no notifications.

| Method | Path | Params | Response `data` | Errors |
|---|---|---|---|---|
| GET | /api/overview | none | `overview()` (mockData KPI block) | uncaught → 500 |
| GET | /api/campaigns | none | `campaigns`: a static mock array (not the planning hierarchy) | none |
| GET | /api/budget | none | `budget()` (mock) | none |
| GET | /api/channels/revenue | none | `revenueByChannel()` | none |
| GET | /api/performance/timeseries | `?days`: `Math.min(90, Math.max(7, Number(days) \|\| 30))` | `timeseries(days)` | none |
| GET | /api/integrations | none | `integrationsCatalog`. The item with id `google-analytics` gets `status: ga.configured ? 'connected' : 'demo'`; every other item is forced to `status: 'connected'`. | none |
| GET | /api/analytics/google | none | `getAnalytics()`: GA4 Data API report over `28daysAgo`→`today`, or demo data when not configured | 500 `{error: err.message}` |
| GET | /api/analytics/realtime | none | `getRealtime()` | 500 err.message |
| GET | /api/goals | none | `goals()` | none |
| GET | /api/insights | none | `insights()` (rule-based) | none |
| GET | /api/insights/ai | none | `{ source:'ai'\|'computed', provider:'Azure OpenAI · gpt-4o'\|'Claude'\|'ZenReach analytics engine', insights:[{title,detail,action,impact,severity}] }`. Cached in memory for 10 min when AI, 30 s when computed. | 500 err.message |
| GET | /api/optimizer | none | `recommend()` | none |
| POST | /api/optimizer/project | body `{ allocations: [] }` (a non-array becomes `[]`) | `project(allocations)` | none (no role gate) |

#### 5.3.3 Assistant (2)
| Method | Path | Auth | Params | Response | Errors |
|---|---|---|---|---|---|
| GET | /api/assistant/status | SIGNED-IN | none | `{ ai: bool, provider }`. `provider` is `` `OpenAI · ${OPENAI_MODEL}` ``, `'Azure OpenAI · gpt-4o'`, `'Claude'`, or `'built-in analyst'` (with `ai:false`). | none |
| POST | /api/assistant | SIGNED-IN (no role) | body `{ message, history, flow }`. `message` is `String(...).slice(0, 2000)`; `history` must be an array (only the last 6 entries are used); `flow` must be an object, else null. | `assistantTurn()`. If `flow.active`: `{source:'builder', suggestions, ...continueBuild}`. On build intent: `{suggestions, ...startBuild()}`. Otherwise `{ reply, source:'ai'\|'local', suggestions, note?, link?: {label, route} }`. | 400 `'message is required'` when the trimmed message is empty; 500 err.message |

#### 5.3.4 Dashboard builder (7)
| Method | Path | Auth | Params | Response | Side effects | Errors |
|---|---|---|---|---|---|---|
| GET | /api/builder/catalog | SIGNED-IN | none | `builderCatalog()` | none | none |
| GET | /api/builder/data | SIGNED-IN | none | `dataBundle()` | none | none |
| GET | /api/builder/dashboards | SIGNED-IN | none | dashboards array | none | none |
| GET | /api/builder/dashboards/:id | SIGNED-IN | none | dashboard | none | 404 `'Dashboard not found'` |
| POST | /api/builder/dashboards | SIGNED-IN (**no role gate**) | body `{ name, widgets }`. The name is trimmed, defaults to `'Untitled dashboard'`, max 80 characters. | `{id:'dash-…', name, createdAt, widgets}` | audit `dashboard.create` {label: name, entityId} | 400 `` `A dashboard named “${clean}” already exists. Please choose a different name.` `` |
| PUT | /api/builder/dashboards/:id | SIGNED-IN (no role gate) | body `{ name?, widgets? }` | updated dashboard (with `updatedAt`) | audit `dashboard.update` | 404 `'Dashboard not found'`; 400 `'Dashboard name cannot be empty.'` / duplicate-name message |
| DELETE | /api/builder/dashboards/:id | SIGNED-IN (no role gate) | none | `{ trash }`: the trash entry, or `null` when the id is unknown (no 404) | **Recycle bin** type `dashboard` (`by` = `me(bearer).name`); audit `dashboard.delete` {label: prev?.name}, written even when the id is unknown | none |

#### 5.3.5 Planning hierarchy and models (12)

**Constants:**
- `LEVELS`: `[{id:'plan',name:'Plan',parentLevel:null,hint:'Annual / strategic container'},{id:'campaign',name:'Campaign',parentLevel:'plan',hint:'A grouping of related programs'},{id:'program',name:'Program',parentLevel:'campaign',hint:'A workstream that holds tactics'}]`.
- Model `STAGES`: inquiries/Inquiries, mqls/MQLs, tqls/TQLs, opportunities/Opportunities, won/Closed won.
- `MECHANISMS = ['C','F','L','E']`.
- `SEGMENTS = ['month','quarter']`.
- `APPROVAL_STATUSES = ['Created','Submitted','Declined','Approved']`.

| Method | Path | Auth | Params | Response | Side effects | Errors |
|---|---|---|---|---|---|---|
| GET | /api/plans/levels | SIGNED-IN | none | `LEVELS` | none | none |
| GET | /api/plans/tree | SIGNED-IN | none | `planTree(tacticsByProgram())`: a nested tree with decorated tactics under each program | none | none |
| GET | /api/plans/nodes | SIGNED-IN | none | flat node list | none | none |
| GET | /api/plans/meta | SIGNED-IN | none | `{ levels: [...LEVELS, {id:'tactic',name:'Tactic',parentLevel:'program'}, {id:'lineItem',name:'Line item',parentLevel:'tactic'}], models, stages, mechanisms, segments, approvalStatuses, teams: [{id,name,color,memberIds,leadId}], currentYear }` | none | none |
| GET | /api/plans/models | SIGNED-IN | none | models list | none | none |
| POST | /api/plans/models | **Admin+** | model body (name required) | model | audit `model.create` {label, entity:'model', entityId} | 400 `'A model needs a name.'`, `` `A model called “${name}” already exists.` ``. Errors without a status are forced to 400. |
| PUT | /api/plans/models/:id | **Admin+** | patch | model | audit `model.update` | 404 `'Model not found'`; 400 duplicate name |
| GET | /api/plans/nodes/:id | SIGNED-IN | none | node | none | 404 `'Node not found'` |
| POST | /api/plans/nodes | **Editor+** | body: `level`, `parentId`, `name`, `owner`, `ownerId`, `fiscalYear`, `currency`, `startDate`, `endDate`, `dateMode`, `planBudget`, `modelId`, `teams[]`, `description`, … | the created node (`comments` forced to `[]`) | audit `plan.create` {label: name, entity: node.level \|\| 'plan node'}; **notify** `body.ownerId` (§5.5) | 400 validation errors (listed below). A non-validation throw becomes 500. `createNode` is outside the try block, so its errors are uncaught. |
| PUT | /api/plans/nodes/:id | **Editor+** | patch. `comments` is stripped from the body. | updated node | audit `plan.update`; **notify** the new owner only when `body.ownerId` is set and differs from `prev.ownerId` | 404 `'Node not found'`; 400 validation |
| POST | /api/plans/nodes/:id/comments | **Editor+** | body `{ text, author? }` | updated node. The comment is appended as `{author: req.user?.name \|\| body.author \|\| 'You', text, at: ISO}`. | audit `plan.comment` | 404 `'Node not found'`; 400 `'Write a comment first.'` |
| DELETE | /api/plans/nodes/:id | **Admin+** | none | `{ removedIds, removedTactics }` | **Recycle bin** type `node`, cascading: the whole subtree plus every tactic under it, snapshotted together. Audit `plan.delete` {meta: {removed: count, tactics: count}}. | 404 `'Node not found'` |

**`validateNode` messages** (`word` is plan/campaign/program):
- `'A record cannot change level.'`
- `` `Give the ${word} a name.` ``
- `` `A ${word} must sit inside a ${expectedParent} — create the ${expectedParent} first.` ``
- `` `A ${word} can only be created under a ${expectedParent}, not a ${parent.level}.` ``
- `` `A plan called “${name}” already exists — plan names must be unique.` ``
- `` `“${parent?.name || 'This parent'}” already has a ${word} called “${name}”.` ``
- `'Choose the fiscal year this plan covers.'`
- `` `A plan's fiscal year must be between ${now} and ${now + 10}.` ``
- `'The start date must be on or before the end date.'`
- `` `Enter both a start and an end date, or let the ${word} calculate its dates from its tactics.` ``
- `` `Program dates must sit inside the campaign's dates (${start} → ${end}).` ``
- `` `Program “${name}” runs ${s} → ${e}, outside these campaign dates.` ``
- `` `A program's budget can't exceed its campaign's (${n}).` ``
- `` `Program “${name}” already has a bigger budget (${n}) than this.` ``
- `'One of the selected teams no longer exists.'`
- `'The plan owner must belong to one of the plan’s teams.'`

#### 5.3.6 Planning board configuration (4)
No audit on any of these.

| Method | Path | Auth | Params | Response | Errors (400) |
|---|---|---|---|---|---|
| GET | /api/planning/board-config | SIGNED-IN | none | `{ columns, fields, templates: [{id,name}] }` | none |
| PUT | /api/planning/board-config/columns | **Planner** | body `{ columns: [...] }`. Missing means `[]`. Each column is `{id,name,hint,color,wip}`; duplicate ids get a `-2` suffix. | saved columns | `'A board needs at least one column.'`, `'A board can have at most 12 columns.'` |
| PUT | /api/planning/board-config/fields | **Planner** | body `{ fields: [{id?,label,type,options?}] }`. `type` is one of `text`/`number`/`select`/`date`; select options max 20. | saved fields (entries without a label are dropped) | `'Invalid fields.'`, `'At most 20 custom fields.'` |
| POST | /api/planning/board-config/template | **Planner** | body `{ template }`, one of `'default'`, `'kanban-plus'`, `'campaign-ops'` | saved columns | `'Unknown board template.'` |

#### 5.3.7 Campaign dual-plans: SP&A (spend) and BCPIR (returns) (5)
Plan ids have the form `spa:<campaignId>` or `bcpir:<campaignId>`.

| Method | Path | Auth | Params | Response | Side effects | Errors |
|---|---|---|---|---|---|---|
| GET | /api/planning/plans | SIGNED-IN | none | `{ currency:'USD', totals:{plannedCost, committed, actualSpend, plannedReturn, actualReturn, roi, roas}, groups:[{planId, planName, fiscalYear, owner, ownerId, currency, startDate, endDate, campaigns[]}] (+ an 'Unassigned' group for orphans), campaigns: rows }` | none | none |
| GET | /api/planning/plans/:id | SIGNED-IN | `:id` = `spa:<cid>` or `bcpir:<cid>` | detail: `header {id, type:'SP&A'\|'BCPIR', kind, name, …}` and more | none | 404 `'Plan not found'` |
| POST | /api/planning/plans/:campaignId/submit | **Editor+** | body `{ approver, approverId, by? }`. `by` = `req.user?.name \|\| body.by`. | updated campaign node: `status:'Submitted'`, `approval:{submittedBy, submittedAt, approver, approverId, approvedBy:'', approvedAt:'', note:''}` | audit `plan.submit` {label: name, meta:{approver}}; **notify** `body.approverId` (type approval, route `/planning?plan=spa:<cid>`) | 404 `'Campaign not found'`. There is no state check, so a plan can be re-submitted at any time. |
| POST | /api/planning/plans/:campaignId/approve | **Editor+** | none | node with `status:'Published'`, `approval.approvedBy/approvedAt` | audit `plan.approve`. No notification. | 404 `'Campaign not found'` (missing or not a campaign); 403 approvalGuard message. There is no status check. |
| POST | /api/planning/plans/:campaignId/reject | **Editor+** | body `{ note }` | node with `status:'Draft'`, `approval.rejectedBy/rejectedAt/note` | audit `plan.reject` {meta:{note}}. No notification. | 404 / 403 as for approve |

#### 5.3.8 Board, summary and iCalendar (3)
| Method | Path | Auth | Params | Response |
|---|---|---|---|---|
| GET | /api/planning/board | SIGNED-IN | `?planId&campaignId&programId&planIds` | `{ columns: [{...columnConfig, initiatives: [decoratedTactic]}], meta: { types, priorities, owners:[unique owner names], columns } }`. A tactic whose status matches no column is appended to the first column. |
| GET | /api/planning/summary | SIGNED-IN | same filters | `{ total, inFlight, launched, plannedBudget, avgProgress, quarters: [sorted], p0 }` |
| GET | /api/planning/calendar.ics | **PUBLIC (hard-coded wall exception)** | same filters | `text/calendar` body from `tacticsToIcs(board.columns.flatMap(c => c.initiatives))`, calendar name `'ZenReach Marketing Calendar'`. Headers: `Content-Type: text/calendar; charset=utf-8`, `Content-Disposition: inline; filename="zenreach-calendar.ics"`, `Cache-Control: public, max-age=3600`. The client builds this URL in `CalendarView.jsx`, plus a `webcal://` variant. |

#### 5.3.9 Tactics (16)

**`decorateTactic(t)`** adds:
- `lifecycleStatus` (an Approved tactic shows In-Progress or Complete by date).
- `money: { plannedCost: budget, lineItems: Σ line-item amount, balance: budget − line items, actual: Σ line-item actual if > 0, else spendActual }`.

| Method | Path | Auth | Params | Response | Side effects | Errors |
|---|---|---|---|---|---|---|
| GET | /api/tactics/:id | SIGNED-IN | none | `{ ...decorated, missingForSubmit: string[] }`. Possible entries: `'name'`, `'owner'`, `'program'`, `'start date'`, `'end date'`, `'planned cost'` (the last only if the model has `requirePlannedCost`). | none | 404 `'Tactic not found'` |
| POST | /api/tactics | **Editor+** | tactic body: `title`, `programId` (or only `campaignId`, resolved via `resolveProgramId`), `type`, `status`, `priority`, `owner`, `ownerId`, `channel`, `budget`, `startDate`, `endDate`/`dueDate`, `evergreen`, `quarter`, `goal`, … | decorated tactic | audit `tactic.create` {label: title}; **notify** `body.ownerId` | 400 validateTactic (below) |
| PUT | /api/tactics/:id | **Editor+** | patch | decorated tactic (`updateTactic(..., {by: user name})`) | audit `tactic.update`; notify the new owner only if `ownerId` changed | 404 `'Tactic not found'`; 400 validation |
| POST | /api/tactics/:id/submit | **Editor+** | body `{ approver, approverId }` | decorated tactic, `approvalStatus:'Submitted'` and an `approval` history entry `{action:'submitted', comment: 'Sent to <approver>'}` | audit `tactic.submit` {meta:{approver}}; **notify** `body.approverId` (approval) | 400 `` `This tactic is already ${status.toLowerCase()}.` `` (only Created or Declined can be submitted); 400 `` `Fill in the required fields before submitting: ${missing.join(', ')}.` ``; 404 `'Tactic not found'` |
| POST | /api/tactics/:id/approve | **Editor+** and tacticApprovalGuard | body `{ comment? }` | decorated tactic, `approvalStatus:'Approved'` | audit `tactic.approve` {meta:{comment}}; **notify** `prev.ownerId` 'Tactic approved' | 404; 403 guard message; 400 `'Only submitted tactics can be approved or declined.'` |
| POST | /api/tactics/:id/decline | same | body `{ comment }` (required) | `approvalStatus:'Declined'` | audit `tactic.decline`; notify the owner 'Tactic declined' | as for approve, plus 400 `'Add a comment explaining why the tactic is declined.'` (and an internal `'Unknown decision.'`) |
| POST | /api/tactics/:id/line-items | **Editor+** | line-item body (`name`, `amount`, `actual`, …); the server assigns the `li-` id | `{ tactic: decorated, lineItem }` | audit `lineitem.create` {entity:'line item', meta:{tactic: title}} | 404 `'Tactic not found'`; 400 `` `“${t.title}” already has a line item called “${name}”.` ``; 400 `` `Line items (${sum}) can't add up to more than the tactic's planned cost (${budget}) — the “${model}” model blocks a negative balance.` `` |
| PUT | /api/tactics/:id/line-items/:liId | **Editor+** | patch | `{ tactic, lineItem }` | audit `lineitem.update` | 404 `'Tactic not found'` / `'Line item not found'`; 400 as above |
| DELETE | /api/tactics/:id/line-items/:liId | **Editor+** | none | `{ tactic: decorated }` | **Recycle bin** type `lineItem`; audit `lineitem.delete` | 404 `'Tactic not found'` / `'Line item not found'` |
| POST | /api/planning/tactics/bulk-move | **Editor+** | body `{ ids, programId }` | `{ moved, programId }` | audit `tactic.bulk_move` {label: "N tactic(s)", meta:{programId}} | 400 `'Pick the program to move the tactics into.'`, `'Select at least one tactic to move.'`, `` `“${program}” already has a tactic called “${title}”.` `` |
| POST | /api/planning/tactics/bulk-copy | **Editor+** | body `{ ids, programId }` | `{ copied, ids }`. Copies start fresh: `Created`, no approval, comments, source, loopId, mediaCodes or PR links, and line-item actuals set to 0. Titles that clash become `(copy)` / `(copy N)`. | audit `tactic.bulk_copy` | 400 `'Pick the program to copy the tactics into.'`, `'Select at least one tactic to copy.'` |
| POST | /api/planning/tactics/bulk-submit | **Editor+** | body `{ ids, approver, approverId }` | `{ submitted: <count>, skipped: [{id, title, reason}] }`. Per-id errors are collected; the request never fails. | Only if `submitted > 0`: audit `tactic.bulk_submit` {meta:{skipped: n}} and a single **notify** to `approverId` | none (always 200) |
| DELETE | /api/tactics/:id | **Editor+** | none | `{ trash }` (null if unknown; no 404) | **Recycle bin** type `tactic`; audit `tactic.delete`, always written | none |
| POST | /api/planning/tactics/bulk-delete | **Editor+** | body `{ ids: [...] }` or `{ all: true }` | `result` = `{ deleted, remaining }`, falling back to `{deleted:0}` | **Recycle bin** type `tactics` (sublabel `'Cleared all tactics'` or `'Bulk delete'`); audit `tactic.bulk_delete` {label: "N tactic(s)", meta:{count}} | none |
| POST | /api/tactics/:id/comments | **Editor+** | body `{ author, text }`. **The author comes from the client body, not the server identity.** | the raw tactic (not decorated). The comment is `{id:'cm-…', author ≤80 chars (default 'You'), text ≤1000 chars, at}`. | audit `tactic.comment` | 404 `'Tactic not found'` |
| POST | /api/tactics/:id/media-codes | **Editor+** | body `{ channel, label }` | the raw tactic with `mediaCodes` extended by `{id:'mc-…', code:'ZR-<ABBR≤4>-<SUFFIX>', channel ≤40, label ≤80}` | **no audit** | 404 `'Tactic not found'` |

**`validateTactic` messages:**
- `'A tactic must sit inside a program — pick (or create) the program first.'`
- `` `A tactic can only be created under a program, not a ${level}.` ``
- `'Give the tactic a name.'`
- `` `“${program.name}” already has a tactic called “${title}”.` ``
- `` `The “${model.name}” model doesn't allow “${type}” tactics.` ``
- `'An evergreen tactic cannot be made non-evergreen. Close it and create a new tactic with real dates.'`
- `'The start date must be on or before the end date.'`
- `` `End dates can't be more than 10 years past the plan year (${fy + 10}).` ``
- `'Planned cost cannot be negative.'`

#### 5.3.10 Back-compat aliases: `/api/planning/initiatives` (3)
These have **no audit, no notifications and no `decorate`**.

| Method | Path | Auth | Behaviour | Errors |
|---|---|---|---|---|
| POST | /api/planning/initiatives | Editor+ | Same as POST /api/tactics (resolveProgramId, validate, create). Returns the raw tactic. | 400 or 500 via `fail` |
| PUT | /api/planning/initiatives/:id | Editor+ | validate, then `updateTactic(...,{by})`. Returns the raw tactic. | 404 **`'Initiative not found'`**; validation 400 |
| DELETE | /api/planning/initiatives/:id | Editor+ | `trashTactic` (recycle bin). Returns `{trash}`. | none |

The client's "back-compat" methods (`createInitiative`, `updateInitiative`, `deleteInitiative`) actually call `/tactics`, not these routes. Nothing in the client calls `/planning/initiatives`.

#### 5.3.11 Currency (1)
| Method | Path | Auth | Params | Response | Errors |
|---|---|---|---|---|---|
| GET | /api/currency | **PUBLIC** | `?refresh=1` forces a fresh pull | `{ base:'USD', currencies:['USD','EUR','GBP','CAD','AUD','INR','JPY','SGD'], rates:{CODE: USD per 1 unit}, updatedAt, source }`. `source` is either `'frankfurter.app · ECB <date>'` or `'fallback (static)'`. | On a throw it still returns HTTP 200 `{ ok:true, data:{ base:'USD', currencies, rates:{}, source:'unavailable' } }` |

FX cache details:
- The cache refreshes at most hourly unless forced (`MAX_AGE_MS = 1h`).
- Live rates come from `https://api.frankfurter.app/latest?from=USD&to=…` with a 6 s timeout and are inverted to "USD per unit".
- The static fallback is USD 1, EUR 1.08, GBP 1.27, CAD 0.73, AUD 0.66, INR 0.012, JPY 0.0064, SGD 0.74.

#### 5.3.12 Budgeting (8)
| Method | Path | Auth | Params | Response | Side effects | Errors |
|---|---|---|---|---|---|---|
| GET | /api/budgets/currencies | SIGNED-IN | none | `CURRENCIES` (8 codes) | none | none |
| GET | /api/budgets/rollup | SIGNED-IN | `?currency&planId&campaignId` | `budgetRollup(currency, {planId, campaignId})` with `{currency, totals, rows, …}` | none | none |
| GET | /api/budgets/pools | SIGNED-IN | none | pools[] | none | none |
| POST | /api/budgets/pools | **Editor+** | body `{ id?, name, total, currency, planId }`. Upsert: with an `id` it updates name, total and currency. | pool `{id:'pool-…', name ≤120 chars, planId, currency (3 chars, upper), total (rounded, ≥0), createdAt, updatedAt}`, or **`null` (HTTP 200) for an unknown id** | audit `budget.pool` | uncaught → 500 |
| POST | /api/budgets/allocations | **Editor+** | body `{ poolId, nodeId, period, amount }`. Upserts on (poolId, nodeId, period). | `{id:'alloc-…', poolId, nodeId, period, amount}` | audit `budget.allocation` {label: `out.name` (always undefined), else `body.poolName`, else `body.poolId`} | 400 `'Unknown campaign/program.'` |
| GET | /api/budgets/snapshots | SIGNED-IN | none | `[{id, label, at, totals, currency}]` (rows omitted) | none | none |
| POST | /api/budgets/snapshots | **Editor+** | body `{ label, currency }` | `{id:'snap-…', label (default 'Snapshot N', ≤80), at, currency, totals, rows:[{name, planned, committed}]}` | audit `budget.snapshot_create` | uncaught → 500 |
| DELETE | /api/budgets/snapshots/:id | **Editor+** | none | `{ id }` (no 404) | audit `budget.snapshot_delete`, written **before** deletion; hard delete, no recycle bin | none |

#### 5.3.13 Procurement: purchase requests (10)

- PR statuses (`PR_STATUS`): `draft`, `submitted`, `approved`, `rejected`, `po`, `invoiced`, `received`.
- Fulfilment path: `approved → po → invoiced → received`.

| Method | Path | Auth | Params | Response | Side effects | Errors |
|---|---|---|---|---|---|---|
| GET | /api/procurement/requests | SIGNED-IN | `?planId&campaignId` | PRs[] | none | none |
| GET | /api/procurement/summary | SIGNED-IN | `?planId&campaignId` | `{ total, byStatus:{<status>:count}, totalValue, pendingApproval, approvedValue, chain: APPROVAL_CHAIN, scope }` | none | none |
| GET | /api/procurement/pending | SIGNED-IN | `?approver=<name>` (matches by name) | submitted PRs that have a pending step at the active level (for that approver) | none | none |
| GET | /api/procurement/requests/:id | SIGNED-IN | none | PR | none | 404 `'Request not found'` |
| POST | /api/procurement/requests | **Editor+** | body: `title`, `requester`, `requesterId`, `approver`, `approverId`, `vendor`, `currency`, `tacticId`, `campaignId`, `customAttrs`, `lines:[{description, qty, unitPrice, amount?}]` (max 40 lines) | PR `{id:'pr-…', status:'draft', total, approvals:[], poNumber:'', invoiceNumber:'', history:[{action:'created'}], …}`. An unknown tacticId or campaignId becomes null. | audit `pr.create` {meta:{total}}; **notify** `body.requesterId` | uncaught → 500 |
| PUT | /api/procurement/requests/:id | **Editor+** | patch (same fields) | PR | audit `pr.update` | 404 `'Request not found'`; 400 `'Only draft requests can be edited.'` |
| POST | /api/procurement/requests/:id/submit | **Editor+** | body `{ by? }`. The server calls `submitRequest(id, body.by \|\| actor, actor, req.user.id)`, where `actor = req.user.name \|\| body.by`. **The client's `approverName` field is ignored.** The chain uses the PR's stored `approver`/`approverId` and falls back to the submitter. | PR with `status:'submitted'` and the approvals chain built | audit `pr.submit` {meta:{approver: pr.approver}}; **notify** `pr.approverId` (approval) | 404; 400 `'Request already submitted.'`, `'Add at least one line item first.'` |
| POST | /api/procurement/requests/:id/approve | **Editor+** | body `{ stepId?, approver?, decision: 'approve'\|'reject', comment? }`. **`by` is forced to the authenticated name, and `actorId` to `req.user.id`.** | PR, recomputed (all steps approved means `approved`; any rejected step means `rejected`). The step records `actedBy`. | audit `pr.reject` if `body.decision === 'reject'`, else `pr.approve`. No notification. | 404; 400 `'Request is not awaiting approval.'`, `'No matching pending approval step.'`, `'An earlier approval level is still pending.'`, `` `Only ${step.approver || 'the assigned approver'} can approve or reject this request.` `` |
| POST | /api/procurement/requests/:id/advance | **Editor+** | body `{ poNumber?, invoiceNumber? }` | PR. The default PO number is `PO-<ID SEGMENT>`, the default invoice number `INV-<ID SEGMENT>`. | audit `pr.advance` {meta:{stage}} | 404; 400 `'Nothing to advance from the current status.'` |
| DELETE | /api/procurement/requests/:id | **Editor+** | none | `{ trash }` (null if unknown) | **Recycle bin** type `request`; audit `pr.delete` | none |

#### 5.3.14 Attribution and predictive analytics (6)
All are SIGNED-IN, read-only and unaudited.

`ATTRIBUTION_MODELS` ids: `first`, `last`, `last_non_direct`, `linear`, `time_decay`, `position`, `w_shaped`, `full_path`, `custom`. Each entry is `{id, name, desc}`.

| Method | Path | Params | Response |
|---|---|---|---|
| GET | /api/attribution/models | none | `ATTRIBUTION_MODELS` |
| GET | /api/attribution | `?model` | `attribute(model)` |
| GET | /api/attribution/compare | none | `compareModels()` |
| GET | /api/attribution/funnel | none | `funnel()` |
| GET | /api/attribution/reallocation | `?model` | `reallocation(model)` |
| GET | /api/attribution/forecast | none | `forecast()` |

#### 5.3.15 Notifications (2)
| Method | Path | Auth | Response | Side effects |
|---|---|---|---|---|
| GET | /api/notifications | SIGNED-IN. The user comes from the token only; the client's `?user=` is ignored. | `{ count: <unread>, total, items: [...] }`. Pushed items are merged with derived items (pending PR approvals, tactic deadlines and so on), sorted newest first, max 40. | **This GET writes:** it persists a snapshot (pushed, items, seenIds, total) to `notifications.json` |
| POST | /api/notifications/read | SIGNED-IN (no role) | `{ count: 0, total, items }` | marks every current item id as seen |

#### 5.3.16 Audit log (1)
| Method | Path | Auth | Query | Response |
|---|---|---|---|---|
| GET | /api/audit | SIGNED-IN; the scope comes from the token | `category`, `severity`, `actorId`, `q` (case-insensitive substring over `actorName summary label`), `limit` (default 200, clamped 1..500) | `{ scope:{key:'all'\|'org'\|'team'\|'self', label, description, canSeeAll}, stats:{total, today, people, critical (warning+critical)}, categories:[{key,label,count}], actors:[{id,name,initials,avatarGradient,count}], events:[newest first] }` |

**Scope rules:**
- `all`: Admin+ by permission role, or orgRole `admin`.
- `org`: a manager sees self, direct reports, and the members of teams led by their leads.
- `team`: a lead sees self and the members of teams they lead.
- `self`: everyone else.

Scope labels: 'Workspace-wide', 'Your organization', 'Your teams', 'Your activity'.

**Event shape:** `{id:'evt-…', at, action, category, severity, actorId, actorName, actorRole, actorOrgRole, actorInitials, actorGradient, entity, entityId, label, summary, meta}`.

#### 5.3.17 Recycle bin (4)
All four are **Editor+**.

| Method | Path | Response | Side effects | Errors |
|---|---|---|---|---|
| GET | /api/trash | `[{id:'trash-…', deletedAt, type, label, sublabel, count, by}]`, newest first, payload stripped | none | none |
| POST | /api/trash/:id/restore | `{ ok:true, type, label }`, inside the normal envelope | audit `recycle.restore` {label} | 400 `'This item is no longer in the recycle bin.'`; `` `Its parent no longer exists — restore the parent ${'campaign'\|'plan'} first.` ``; `'Its tactic no longer exists — restore the tactic first.'`; `` `Cannot restore unknown item type "${type}".` `` |
| DELETE | /api/trash/:id | `{ purged: 0\|1 }` | audit `recycle.purge`, written **before** the purge | none |
| POST | /api/trash/empty | `{ purged: n }` | audit `recycle.empty` | none |

Types: `node`, `lineItem`, `tactic`, `tactics`, `dashboard`, `request`.

**Restore rules:**
- Restored tactics come back as `approvalStatus:'Created'` with a history entry `{action:'restored', comment:'Restored from the recycle bin — re-submit for approval.'}`.
- Restored campaigns that have a status come back as `status:'Draft'` with `approval:null`.
- A dashboard name clash gets the suffix ` (restored)`.
- A line-item name clash gets ` (2)`, ` (3)` and so on.
- Re-inserts are idempotent by id.

#### 5.3.18 Google Calendar sync (4)
These use the GA service-account key and are one-way pushes.

| Method | Path | Auth | Params | Response | Errors |
|---|---|---|---|---|---|
| GET | /api/integrations/google-calendar/status | SIGNED-IN | none | `{ keyConfigured, serviceAccount, connected, calendarId, connectedAt, planIds, hint }`. `hint` is either `Share your Google Calendar with <sa email> ("Make changes to events"), then sync.` or `'No Google service-account key configured (GOOGLE_APPLICATION_CREDENTIALS).'` | 500 err.message |
| POST | …/sync | **Editor+** | body `{ calendarId (trimmed), planIds? }` | `{ calendarId, calendarName, scheduled, synced, failed, errors (≤5), connected:true, connectedAt, planIds }`. The connection is persisted. | 400 `'A valid Google Calendar ID (usually your Google account email) is required.'`, `'No Google service-account key configured (set GOOGLE_APPLICATION_CREDENTIALS).'`, `'Could not obtain a Google access token from the service account.'`, or a friendly Google API error |
| POST | …/clear | **Editor+** | body `{ calendarId, planIds? }` | `{ calendarId, deleted, note? }` | 400 `'A valid Google Calendar ID is required.'` |
| POST | …/disconnect | **Editor+** | none | `{ disconnected:true, calendarId, cleared }`. It clears events on a best-effort basis and forgets the connection. | 400 |

None of these are audited.

#### 5.3.19 External dataset, the Builder data source (2)
| Method | Path | Auth | Params | Response | Side effects | Errors (400) |
|---|---|---|---|---|---|---|
| GET | /api/dataset/campaign/meta | SIGNED-IN | none | `{loaded:false}` or `{loaded:true, ...meta}` | none | none |
| POST | /api/dataset/campaign/import | **Editor+** | body `{ csv }` or `{ fileBase64, name }`, normalised by `toCsvText` (below). `name` is also the source name, default `'uploaded'`. | `{ loaded:true, ...meta }` | Replaces `campaignDataset.json`. Audit `import.dataset` {label: name \|\| 'campaign dataset', meta:{rows}}. | `'No file content provided.'`, `'The workbook has no readable sheet.'`, `'CSV needs a header row and at least one data row.'`, `'This does not look like a campaign dataset (no recognizable columns).'`, `'No data rows found.'` |

`toCsvText`:
- A non-empty `csv` wins.
- Otherwise `fileBase64` is decoded. A `.csv`/`.txt`/`.tsv` name is read as UTF-8.
- Anything else is parsed by SheetJS: first sheet only, converted to CSV with `dateNF:'yyyy-mm-dd'` and `blankrows:false`.

#### 5.3.20 Imports: CSV, Trello and Jira (8)
| Method | Path | Auth | Params | Response | Side effects | Errors (400) |
|---|---|---|---|---|---|---|
| GET | /api/import/sample | SIGNED-IN | none | `{ csv: SAMPLE_CSV }` | none | none |
| POST | /api/import/tactics | **Editor+** | `{csv}` or `{fileBase64, name}` | `{ imported, ids, errors[] }`. Soft errors come back with HTTP 200: `'CSV needs a header row and at least one data row.'`, `'CSV must include a "title" (or "name") column.'`, `` `Row N: missing title — skipped.` ``, `` `Row N: <msg>` ``. | audit `import.tactics` {label: "<n> tactics", meta:{imported}} | toCsvText errors |
| POST | /api/import/trello/boards | **Editor+** | `{ key, token }` | Trello boards list | **no audit** | `'Trello needs both an API key and a token.'`; network errors (below) |
| POST | /api/import/trello | **Editor+** | `{ key, token, boardId }` | `{ source:'trello', board: <name string>, imported, ids, errors, connectionId }` | audit `import.trello`. Its label reads `out.board.name`, but `board` is a string, so the label always falls back to **'Trello board'** (a bug). A connection is recorded. | `'Pick a Trello board to import.'` |
| POST | /api/import/jira/projects | **Editor+** | `{ baseUrl, email, apiToken }` | projects list | none | `'Jira needs a site URL, your email, and an API token.'` |
| POST | /api/import/jira | **Editor+** | `{ baseUrl, email, apiToken, projectKey?, jql? }` | `{ source:'jira', project: <key or 'JQL'>, imported, ids, errors, connectionId }` | audit `import.jira`. The label falls back to **'Jira project'** for the same reason. | `'Pick a Jira project (or provide a JQL query).'` |
| GET | /api/import/connections | SIGNED-IN | none | `[{ id, provider, name, key, count, tacticIds, importedAt, liveCount }]` | none | 400 err.message |
| DELETE | /api/import/connections/:id | **Editor+** | none | `{ removed:1, deleted, connection }` or `{ removed:0, deleted:0 }` | **Hard-deletes** every tactic the connection created, bypassing the recycle bin. **No audit.** | 400 |

Network and remote error formats for Trello and Jira:
- `` `Couldn't reach ${label}. Check the URL / your network.` ``
- `` `${label} ${status}${hint} ${text…180}` ``, where the hint is `' — check your credentials/permissions.'` on 401 or 403.
- `` `${label} returned a non-JSON response.` ``

Credentials are sent with each request and are not stored by these routes.

#### 5.3.21 Closed-loop AI plan (6)
| Method | Path | Auth | Params | Response | Side effects | Errors |
|---|---|---|---|---|---|---|
| POST | /api/planloop/generate | **Editor+** | body `{ name='AI marketing plan', goalLeads=3000, budget=400000, quarter='Q4 2026', maxChannels=5 }` | `{ name, quarter, goalLeads, budget, channels, tactics, projection:{leads, revenue, attainmentPct, gapLeads, onTrack, blendedCpl}, rationale }`. `rationale` is replaced by an AI narration (via `assistant.ask`, 9 s timeout, ≤420 characters) when one is available. | not persisted, no audit | 400 err.message |
| POST | /api/planloop/commit | **Editor+** | body `{ name, quarter, goalLeads, budget, tactics[] }` | `{ loopId, campaignId, programId, tacticIds }` | Creates the campaign, program and tactics, and saves the loop. Audit `ai.plan_commit` {label: `out.name` (undefined), else `body.name`, else 'AI plan'; entityId: loopId}. | 400 `'No tactics to commit.'` |
| GET | /api/planloop/list | SIGNED-IN | none | loops with `liveTactics > 0`, newest first | none | none |
| GET | /api/planloop/status | SIGNED-IN | `?loopId` (defaults to the latest) | `loopStatus(loopId)`: pace to goal | none | none |
| GET | /api/planloop/recommendations | SIGNED-IN | `?loopId` | `{ loopId, recommendations }` | none | none |
| POST | /api/planloop/apply | **Editor+** | body `{ loopId, apply }`. `apply` is `{kind:'reallocate', fromId, toId, amount}` or `{kind:'raise-priority', id}` (P2→P1, anything else→P0). | `{ok:true, moved}` or `{ok:true, priority}`. **Logical failures return HTTP 200** with `data = {ok:false, error}`, where error is `'No action.'`, `'Tactic not found.'`, `'Nothing to move.'` or `'Unknown action.'`. | **no audit** | 400 only on a throw |

#### 5.3.22 Accounts and auth (7)

**Public user shape** (`publicUser`): `{ id, name, email, role, company, avatarGradient, initials, createdAt, orgRole|null, managerId|null, teamId|null, title }`. The hash and salt are never sent.

**Password storage:** scrypt with a 64-byte key and a 16-byte hex salt, compared with a timing-safe check.

| Method | Path | Auth | Params | Response | Side effects | Errors (400 unless noted) |
|---|---|---|---|---|---|---|
| POST | /api/auth/signup | PUBLIC (registration stays open even after users exist) | `{ name, email, password, company }` | `{ token, user }`. The **first account becomes Owner**; later accounts become Editor. The email is lower-cased and trimmed. | creates a session. **No audit.** | `'Please enter your name.'`, `'Please enter a valid email address.'`, `'Password must be at least 6 characters.'`, `'An account with that email already exists.'` |
| POST | /api/auth/login | PUBLIC | `{ email, password }` | `{ token, user }` | new session; audit `account.login` (actor = `out.user`) | `'Incorrect email or password.'` (the same message for an unknown user and a wrong password) |
| GET | /api/auth/me | PUBLIC | bearer | the public user, or **`null`** (still `ok:true`) | none | none |
| POST | /api/auth/logout | PUBLIC | bearer | `{ ok: true }` (nested inside the envelope) | deletes the session; audit `account.logout`, with the actor resolved before deletion (null and `'A teammate'` without a token) | none |
| PUT | /api/auth/profile | SIGNED-IN | `{ name?, company?, password?, currentPassword? }` | the public user | audit `account.profile` (actor = the updated user) | `'You are not signed in.'`, `'Account not found.'`, `'Name cannot be empty.'`, `'New password must be at least 6 characters.'`, `'Your current password is incorrect.'` |
| GET | /api/auth/users | **Admin+** | none | public users[] | none | none |
| PUT | /api/auth/users/:id/role | **Admin+** | `{ role }`, one of Owner, Admin, Editor or Viewer | the public user | audit `user.role` {meta:{role}} (severity critical) | `'Unknown role.'`, `'Account not found.'`, `'You cannot remove the last Owner.'`. Nothing stops an Admin from granting Owner. |

#### 5.3.23 Organization (16)

**Team card shape:** `{ id:'team-…', name ≤80, focus ≤120, color, leadId, memberIds, campaignIds, createdAt, updatedAt, lead: publicUser|null, members: publicUser[], memberCount, work }`.

All org write errors go through `authError`, so they return **400**. That includes "Team not found." on writes.

| Method | Path | Auth | Params | Response | Side effects |
|---|---|---|---|---|---|
| GET | /api/org/tree | SIGNED-IN | none | `{ admins:[{...user, managers:[{...user, leads:[{...user, teams:[teamCard]}]}]}], orphanManagers, counts:{managers, leads, teams, members} }` | none |
| GET | /api/org/me | SIGNED-IN | none | `{ orgRole, title, manager: publicUser\|null, team:{id,name,color}\|null }` | none |
| GET | /api/org/dashboard | SIGNED-IN | none | role-aware payload `{ role:'manager'\|'lead'\|…, me, … }` (`{role:null}` without a user) | none |
| GET | /api/org/personas | **PUBLIC** | none | `{ personas:[{name, email, orgRole, title, team, avatarGradient, initials}], password: DEMO_PASSWORD }`. The personas are demo-domain users ordered admin→manager→lead→member, then by name. `DEMO_PASSWORD` is a hard-coded demo constant in org.js (`'zenreach'`). It is deliberately exposed for the login-screen persona switcher and should not exist in production. | none |
| GET | /api/org/campaigns | SIGNED-IN | none | `listCampaigns()` | none |
| GET | /api/org/teams | SIGNED-IN | none | teamCard[] | none |
| GET | /api/org/teams/:id | SIGNED-IN | none | teamCard | 404 `'Team not found.'` |
| POST | /api/org/teams | **Admin+** | `{ name (default 'New team'), focus, color (default cycles TEAM_COLORS), leadId, memberIds, campaignIds }` | teamCard. Each member's `user.teamId` is synced and their orgRole defaults to `'member'`. | audit `team.create`; **notify** every member ('You were added to a team') and the lead ('You’re now a team lead') |
| PUT | /api/org/teams/:id | **Admin+** | partial `{name, focus, color, leadId, memberIds, campaignIds}` | teamCard | audit `team.update`; notify only newly added members, and the lead only if changed. Error `'Team not found.'` (400). |
| DELETE | /api/org/teams/:id | **Admin+** | none | `{ id, remaining }`. Clears the members' `teamId`. Hard delete. | audit `team.delete`. Error `'Team not found.'` (400). |
| PUT | /api/org/teams/:id/lead | **Admin+** | `{ leadId }` (falsy means null) | teamCard | **notify** the new lead if it differs from the previous lead. **No audit.** |
| PUT | /api/org/teams/:id/members | **Admin+** | `{ memberIds }` (replaces the list) | teamCard | **notify** newly added ids. **No audit.** |
| PUT | /api/org/teams/:id/campaigns | **Admin+** | `{ campaignIds }` | teamCard | **No audit.** |
| GET | /api/org/people | SIGNED-IN (the people picker) | `?q` (substring over name, email and title), `?limit` (default 50, clamped 1..200) | public users, sorted admin, manager, lead, member, then by name | none |
| GET | /api/org/users | **Admin+** | none | public users[] | none |
| PUT | /api/org/users/:id | **Admin+** | `{ orgRole ('admin'\|'manager'\|'lead'\|'member'\|null), managerId, teamId, title (≤80) }` | public user. Setting `teamId` also adds the user to that team's `memberIds`. | audit `user.org` {meta:{orgRole, title}}. Errors: `'Account not found.'`, `'Unknown org role.'`. |

#### 5.3.24 Server routes with no client caller
- `GET /api/health`
- `GET /api/plans/models` (models come through `/plans/meta`)
- `POST /api/plans/models` (there is no `createModel` in api.js)
- `GET /api/planning/calendar.ics` (only built as a URL string in `CalendarView.jsx`)
- All 3 `/api/planning/initiatives*` routes

### 5.4 Audit action catalogue

There are **53 distinct actions in index.js**: 49 literal strings plus the ternaries `tactic.approve`/`tactic.decline` and `pr.approve`/`pr.reject`.

**Registered in `audit.js` ACTIONS**, shown as category/severity:

| Group | Actions |
|---|---|
| planning | `plan.create`/`plan.update` info, `plan.delete` warning |
| tactics | `tactic.create`/`tactic.update` info, `tactic.delete` warning, `tactic.bulk_delete` warning, `tactic.comment` info |
| approvals | `plan.submit` notice, `plan.approve` notice, `plan.reject` warning, `pr.submit` notice, `pr.approve` notice, `pr.reject` warning |
| budget | `budget.pool`, `budget.allocation`, `budget.snapshot_create` (all info), `budget.snapshot_delete` warning |
| dashboards | `dashboard.create`/`dashboard.update` info, `dashboard.delete` warning |
| procurement | `pr.create`, `pr.update`, `pr.advance` (all info), `pr.delete` warning |
| org | `team.create`/`team.update` info, `team.delete` warning, `user.role` **critical**, `user.org` info |
| imports | `import.tactics`, `import.trello`, `import.jira`, `import.dataset` (all info) |
| recycle | `recycle.restore` notice, `recycle.purge` critical, `recycle.empty` critical |
| account | `account.login`, `account.logout`, `account.profile` (all info) |
| ai | `ai.plan_commit` notice |

**Used in index.js but NOT registered** (12). These fall into category `other`, severity `info`, with the summary "updated item “<label>”":
- `model.create`, `model.update`
- `plan.comment`
- `tactic.submit`, `tactic.approve`, `tactic.decline`
- `tactic.bulk_move`, `tactic.bulk_copy`, `tactic.bulk_submit`
- `lineitem.create`, `lineitem.update`, `lineitem.delete`

A rebuild should add these to the registry.

**Mutations with no audit at all:**
- signup
- board-config columns, fields and template
- media codes
- the 3 initiatives aliases
- Trello board and Jira project listing
- connection delete
- every Google Calendar operation
- planloop generate and apply
- team lead, members and campaigns setters
- notifications read
- the optimizer project call

Category labels, in filter-bar order: Planning, Tactics, Approvals, Budget, Procurement, Dashboards, Organization, Imports, Recycle bin, Account, AI plans, Other.

### 5.5 Notification catalogue

`actorName` = `req.user?.name || 'Someone'`. Self-notifications are always suppressed.

| Trigger | Recipient | type | title | text | route |
|---|---|---|---|---|---|
| POST /plans/nodes | body.ownerId | assignment | `You’re the owner of a ${level \|\| 'plan'}` | `${actorName} made you owner of “${name}”.` | /planning |
| PUT /plans/nodes/:id (owner changed) | body.ownerId | assignment | same | same | /planning |
| POST /planning/plans/:cid/submit | body.approverId | approval | `Plan approval needed` | `${actorName} asked you to approve the plan for “${r.name}”.` | `/planning?plan=spa:${cid}` |
| POST /tactics; PUT /tactics/:id (owner changed) | ownerId | assignment | `A tactic was assigned to you` | `${actorName} made you owner of “${title}”.` | /planning |
| POST /tactics/:id/submit | body.approverId | approval | `Tactic approval needed` | `${actorName} asked you to approve “${title}”.` | /planning |
| approve / decline | prev.ownerId | approval | `Tactic approved` / `Tactic declined` | `${actorName} approved\|declined “${title}”` followed by `: ${comment}` if there is a comment, otherwise `.` | /planning |
| bulk-submit (submitted > 0) | body.approverId | approval | `Tactic approvals needed` | `${actorName} sent you ${n} tactic${n===1?'':'s'} to approve.` | /planning |
| POST /procurement/requests | body.requesterId | assignment | `A purchase request names you as requester` | `${actorName} created “${title}” with you as the requester.` | /procurement |
| PR submit | pr.approverId | approval | `Approval needed` | `${actorName} submitted “${title}” for your approval.` | /procurement |
| team create/update/members | each newly added member | assignment | `You were added to a team` | `${actorName} added you to “${team}”.` | /admin |
| team create/update/lead (lead changed) | lead | assignment | `You’re now a team lead` | `${actorName} made you lead of “${team}”.` | /admin |

No notifications are sent for plan approve or reject, or for PR approve, reject or advance.

### 5.6 `client/src/api.js`

#### 5.6.1 Token handling
- A module-level `let authToken = localStorage.getItem('zr_token') || ''`. The **storage key is `zr_token`**.
- `setAuthToken(token)` sets the in-memory value and either `localStorage.setItem('zr_token', token)` or `removeItem('zr_token')` when the token is falsy.
- `authHeaders()` returns `{ Authorization: \`Bearer ${authToken}\` }` only when a token exists.
- AuthContext uses it like this:
  - On boot, if a stored token exists, it calls `api.me()`. A null result or an error clears the token.
  - `signup` and `login` call `setAuthToken(token)` with the response token.
  - `logout` calls `api.logout()` on a best-effort basis, then always clears the token.

#### 5.6.2 `get(path)`
1. `fetch(\`/api${path}\`, { headers: authHeaders() })`.
2. If `!res.ok`, it throws `` new Error(`Request failed: ${path} (${res.status})`) ``. **The server's error message is discarded for every non-2xx GET.** For example, a 404 for a missing tactic surfaces as "Request failed: /tactics/x (404)".
3. Otherwise it parses the JSON. If `!json.ok`, it throws `Error(json.error || 'API error')`.
4. It returns `json.data`. A non-JSON 2xx body would throw a SyntaxError.

#### 5.6.3 `send(method, path, body)`
1. It always sends `Content-Type: application/json` plus the auth header.
2. The body is `JSON.stringify(body)`, or none when `body === undefined`. That is the case for `del()` and `logout()`.
3. It parses JSON whatever the status. A parse failure becomes `{ ok:false, error: \`Request failed (${res.status})\` }`, which is what an HTML 500 produces.
4. If `!json.ok`, it throws `Error(json.error || \`Request failed: ${path} (${res.status})\`)`. **Server messages do surface for writes.**
5. It does not check `res.ok` itself. It returns `json.data`.
6. Wrappers: `post = send('POST')`, `put = send('PUT')`, `del = send('DELETE')` with no body.

#### 5.6.4 `qs(obj)`
- Drops entries whose value is `null`, `undefined` or `''`. `0` and `false` are kept.
- Joins the rest as `?k=v&…`, applying `encodeURIComponent` to values only, not keys.
- Returns `''` when nothing remains.
- An array value is serialised as a comma string, which suits the server's comma-split `planIds`.

#### 5.6.5 Method → endpoint map (138 methods)
In the "Call" column, P = POST, U = PUT, D = DELETE, G = GET.

| Method | Call |
|---|---|
| overview | G /overview |
| campaigns | G /campaigns |
| budget | G /budget |
| channelsRevenue | G /channels/revenue |
| timeseries(days=30) | G `/performance/timeseries?days=${days}` (not encoded) |
| integrations | G /integrations |
| googleAnalytics | G /analytics/google |
| realtime | G /analytics/realtime |
| goals | G /goals |
| insights | G /insights |
| aiInsights | G /insights/ai |
| optimizer | G /optimizer |
| optimizeProject(allocations) | P /optimizer/project {allocations} |
| assistant(message, history, flow) | P /assistant {message, history, flow} |
| assistantStatus | G /assistant/status |
| signup(p) | P /auth/signup |
| login(p) | P /auth/login |
| me | G /auth/me |
| logout | P /auth/logout (no body) |
| updateProfile(p) | U /auth/profile |
| builderCatalog | G /builder/catalog |
| builderData | G /builder/data |
| dashboards | G /builder/dashboards |
| dashboard(id) | G /builder/dashboards/:id |
| createDashboard(p) | P /builder/dashboards |
| updateDashboard(id,p) | U /builder/dashboards/:id |
| deleteDashboard(id) | D /builder/dashboards/:id |
| planLevels | G /plans/levels |
| planTree | G /plans/tree |
| planNodes | G /plans/nodes |
| planNode(id) | G /plans/nodes/:id |
| createNode(p) | P /plans/nodes |
| updateNode(id,p) | U /plans/nodes/:id |
| deleteNode(id) | D /plans/nodes/:id |
| planMeta | G /plans/meta |
| nodeComment(id,text) | P /plans/nodes/:id/comments {text} |
| updateModel(id,p) | U /plans/models/:id |
| planningBoard(filter) | G /planning/board + qs |
| planningSummary(filter) | G /planning/summary + qs |
| campaignPlans | G /planning/plans |
| campaignPlan(id) | G `/planning/plans/${encodeURIComponent(id)}` |
| planSubmit(cid, approver, approverId) | P /planning/plans/:cid/submit {approver, approverId} |
| planApprove(cid) | P …/approve {} |
| planReject(cid, note) | P …/reject {note} |
| tactic(id) | G /tactics/:id |
| createTactic(p) | P /tactics |
| updateTactic(id,p) | U /tactics/:id |
| deleteTactic(id) | D /tactics/:id |
| deleteTactics(ids) | P /planning/tactics/bulk-delete {ids} |
| deleteAllTactics | P /planning/tactics/bulk-delete {all:true} |
| moveTactics(ids, programId) | P /planning/tactics/bulk-move |
| copyTactics(ids, programId) | P /planning/tactics/bulk-copy |
| submitTactics(ids, approver, approverId) | P /planning/tactics/bulk-submit |
| submitTactic(id, approver, approverId) | P /tactics/:id/submit |
| approveTactic(id, comment) | P /tactics/:id/approve {comment} |
| declineTactic(id, comment) | P /tactics/:id/decline {comment} |
| addLineItem(tid,p) | P /tactics/:tid/line-items |
| updateLineItem(tid,id,p) | U /tactics/:tid/line-items/:id |
| deleteLineItem(tid,id) | D /tactics/:tid/line-items/:id |
| boardConfig | G /planning/board-config |
| setBoardColumns(columns) | U …/columns {columns} |
| setBoardFields(fields) | U …/fields {fields} |
| applyBoardTemplate(template) | P …/template {template} |
| addComment(id,p) | P /tactics/:id/comments |
| addMediaCode(id,p) | P /tactics/:id/media-codes |
| createInitiative(p) | P **/tactics** (alias) |
| updateInitiative(id,p) | U **/tactics/:id** (alias) |
| deleteInitiative(id) | D **/tactics/:id** (alias) |
| teamUsers | G /auth/users |
| setUserRole(id, role) | U /auth/users/:id/role {role} |
| orgTree | G /org/tree |
| orgMe | G /org/me |
| orgDashboard | G /org/dashboard |
| orgPersonas | G /org/personas |
| orgCampaigns | G /org/campaigns |
| orgTeams | G /org/teams |
| orgTeam(id) | G /org/teams/:id |
| orgCreateTeam(p) | P /org/teams |
| orgUpdateTeam(id,p) | U /org/teams/:id |
| orgDeleteTeam(id) | D /org/teams/:id |
| orgSetTeamLead(id, leadId) | U /org/teams/:id/lead {leadId} |
| orgSetTeamMembers(id, memberIds) | U …/members {memberIds} |
| orgSetTeamCampaigns(id, campaignIds) | U …/campaigns {campaignIds} |
| orgUsers | G /org/users |
| orgPeople(q) | G /org/people + qs({q}) (`limit` is never sent, so the server default of 50 applies) |
| orgSetUser(id,p) | U /org/users/:id |
| currencyInfo(refresh) | G `/currency` plus `?refresh=1` when refresh is truthy |
| budgetCurrencies | G /budgets/currencies |
| budgetRollup(currency, filter) | G /budgets/rollup + qs({currency, ...filter}) |
| budgetPools | G /budgets/pools |
| saveBudgetPool(p) | P /budgets/pools |
| setAllocation(p) | P /budgets/allocations |
| budgetSnapshots | G /budgets/snapshots |
| createSnapshot(label, currency) | P /budgets/snapshots {label, currency} |
| deleteSnapshot(id) | D /budgets/snapshots/:id |
| prList(filter) | G /procurement/requests + qs |
| prSummary(filter) | G /procurement/summary + qs |
| prPending(approver) | G /procurement/pending, plus `?approver=` encoded |
| prGet(id) | G /procurement/requests/:id |
| prCreate(p) | P /procurement/requests |
| prUpdate(id,p) | U /procurement/requests/:id |
| prSubmit(id, by, approverName) | P …/:id/submit {by, approverName} (**the server ignores approverName**) |
| prApprove(id,p) | P …/:id/approve |
| prAdvance(id,p) | P …/:id/advance |
| prDelete(id) | D /procurement/requests/:id |
| notifications(user) | G /notifications, plus `?user=` (**ignored by the server**) |
| markNotificationsRead | P /notifications/read {} |
| audit(filters) | G /audit + qs |
| trash | G /trash |
| restoreTrash(id) | P /trash/:id/restore {} |
| purgeTrash(id) | D /trash/:id |
| emptyTrash | P /trash/empty {} |
| importSample | G /import/sample |
| importTactics(p) | P /import/tactics (`{csv}` or `{fileBase64, name}`) |
| trelloBoards(creds) | P /import/trello/boards |
| importTrello(p) | P /import/trello |
| jiraProjects(creds) | P /import/jira/projects |
| importJira(p) | P /import/jira |
| boardConnections | G /import/connections |
| disconnectBoard(id) | D /import/connections/:id |
| planLoopGenerate(p) | P /planloop/generate |
| planLoopCommit(p) | P /planloop/commit |
| planLoops | G /planloop/list |
| planLoopStatus(loopId) | G /planloop/status, plus `?loopId=` encoded |
| planLoopRecommendations(loopId) | G /planloop/recommendations, plus `?loopId=` |
| planLoopApply(loopId, apply) | P /planloop/apply {loopId, apply} |
| campaignDatasetMeta | G /dataset/campaign/meta |
| importCampaignDataset(p) | P /dataset/campaign/import |
| googleCalendarStatus | G /integrations/google-calendar/status |
| googleCalendarSync(calendarId, planIds) | P …/sync |
| googleCalendarClear(calendarId, planIds) | P …/clear |
| googleCalendarDisconnect | P …/disconnect {} |
| attributionModels | G /attribution/models |
| attribution(model) | G /attribution, plus `?model=${model}` (not encoded) |
| attributionCompare | G /attribution/compare |
| attributionFunnel | G /attribution/funnel |
| attributionReallocation(model) | G /attribution/reallocation, plus `?model=` |
| attributionForecast | G /attribution/forecast |

- Path params are interpolated without encoding everywhere except `campaignPlan`.
- No call sites were found in `client/src` for:
  - `attributionCompare`, `budgetCurrencies`, `budgetSnapshots`, `createSnapshot`, `deleteSnapshot`
  - `dashboard`, `createInitiative`, `updateInitiative`, `deleteInitiative`, `googleCalendarClear`
  - `orgMe`, `orgSetTeamCampaigns`, `orgSetTeamLead`, `orgSetTeamMembers`, `orgTeam`
  - `planLevels`, `planNode`, `planNodes`
  - `prPending`, `prUpdate`, `saveBudgetPool`, `setAllocation`

### 5.7 `useApi(fn, deps = [])`

- Initial state: `data = null`, `loading = true`, `error = null`.
- On mount and on every `deps` change (the exhaustive-deps lint rule is disabled):
  1. It sets `loading = true` and `error = null`. **It does not reset `data`**, so stale data stays visible while reloading.
  2. It runs `Promise.resolve(fn())`. A sync value or a promise both work, but a synchronous throw inside `fn` escapes the effect.
  3. `.then(d => setData(d))`, `.catch(e => setError(e))`, `.finally(() => setLoading(false))`.
- Each of those state updates is guarded by an `alive` flag. Cleanup sets `alive = false`, so responses that arrive after unmount or a deps change are ignored, and an outdated request never overwrites newer state.
- It returns only `{ data, loading, error }`. There is no refetch or mutate function: callers re-run it by changing `deps`.
- `error` is the `Error` thrown by `get`/`send`, carrying the messages from §5.6.

### 5.8 Quirks to decide on in a rebuild

1. `requireRole`/`requirePlanner` contain an anonymous/demo branch that the global wall makes unreachable.
2. Dashboards CRUD and `/optimizer/project` have no role gate, so a Viewer can write.
3. A non-2xx GET hides the server's error message from the client, while writes surface it.
4. Several DELETE routes return 200 with `{trash:null}` instead of a 404 and still write an audit row: dashboards, tactics, PRs, snapshots and trash purge.
5. Tactic comments trust a client-supplied `author`; plan-node comments use the server identity.
6. `prSubmit`'s `approverName` and `notifications`' `?user=` parameters are dead.
7. `planloop/apply` reports logical failures as HTTP 200 with `data.ok:false`.
8. `import.trello`/`import.jira` audit labels never resolve (the board and project are strings), and `budget.allocation`/`ai.plan_commit` read `out.name`, which doesn't exist.
9. Twelve audit actions are unregistered (§5.4).
10. Disconnecting a board hard-deletes its tactics, bypassing the recycle bin.
11. Plan submit, approve and reject have no status-transition guards. Tactic and PR approvals do.
12. Signup stays open after the workspace exists, and an Admin can grant Owner.
13. `GET /api/notifications` writes state.
14. `seedOrg()` at boot isn't wrapped in try/catch.
15. There is no static serving and no JSON error handler: uncaught errors and malformed JSON return HTML.

## 6. Identity, organisation, notifications, audit log and recycle bin

**Source files:** `server/services/store.js`, `server/services/auth.js`, `server/services/org.js`, `server/services/notifications.js`, `server/services/audit.js`, `server/services/recycleBin.js`, `client/src/context/AuthContext.jsx`, `client/src/components/auth/*`, `client/src/components/PeoplePicker.jsx`, `client/src/components/NotificationBell.jsx`, `client/src/components/team/TeamDashboard.jsx`, `client/src/pages/Admin.jsx`, `client/src/pages/AuditLog.jsx`, `client/src/pages/RecycleBin.jsx`.

Items marked **Gap/bug** describe current behaviour that the rebuild should decide on deliberately.

### 6.1 Persistence layer (`store.js`)

- **Data directory:** `process.env.ZR_DATA_DIR` if set, otherwise `server/data/`. The env var lets a second instance (tests, demos) use an isolated copy. The directory is created if missing (`mkdirSync recursive`).
- **`readJson(file, fallback)`** is fully synchronous. It returns `fallback` when:
  - the file is missing,
  - the file is empty or whitespace only,
  - `JSON.parse` throws, or
  - any other error occurs.
  - Gap: a corrupt file is silently treated as empty, and the next write overwrites it, so data is lost.
- **`writeJson(file, data)`** uses `fs.writeFileSync` with `JSON.stringify(data, null, 2)` (2-space pretty print). It does not write atomically (no temp file and rename) and takes no lock. Every service does a whole-file read-modify-write.
- **`uid(prefix='id')`** returns `` `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2,7)}` ``, i.e. a base36 timestamp plus 5 base36 random characters.
  - Prefixes in this area: `user`, `tok`, `team`, `evt`, `trash`.
- **`nowIso()`** returns `new Date().toISOString()`.

Files owned by this area:

| File | Shape |
|---|---|
| `users.json` | `{ users: User[], sessions: { [token]: userId } }` |
| `teams.json` | `{ meta: { orgSeeded: boolean }, teams: Team[] }` |
| `notifications.json` | `{ [userId]: { pushed: Item[], items: Item[], seenIds: string[], total: number, updatedAt } }` |
| `audit.json` | `{ events: Event[] }` |
| `recycleBin.json` | `{ items: TrashEntry[] }` |

### 6.2 Identity: users and sessions (`auth.js`, file `users.json`)

#### 6.2.1 User record (stored)

| Field | Type | Default / normalisation |
|---|---|---|
| `id` | string | `uid('user')` |
| `name` | string | trimmed; required |
| `email` | string | `String(email).trim().toLowerCase()`; unique |
| `company` | string | trimmed, `''` |
| `role` | `'Owner'│'Admin'│'Editor'│'Viewer'` (legacy `'Member'`) | see below |
| `salt` | hex string | 16 random bytes → 32 hex characters |
| `passwordHash` | hex string | scrypt, 64-byte key → 128 hex characters |
| `avatarGradient` | CSS string | `AVATAR_GRADIENTS[users.length % 6]`, using the count before the push |
| `initials` | string | see algorithm below |
| `createdAt` | ISO string | now |
| `orgRole` | `'admin'│'manager'│'lead'│'member'│null` | null |
| `managerId` | userId or null | null |
| `teamId` | teamId or null | null |
| `title` | string | `''`, sliced to 80 characters |

**`AVATAR_GRADIENTS`** (6 entries, in order). Each is `linear-gradient(135deg, A, B)`:
1. `#2a78d6` → `#9f4bd6`
2. `#1baf7a` → `#2a78d6`
3. `#eb6834` → `#e34980`
4. `#9f4bd6` → `#e34980`
5. `#eda100` → `#eb6834`
6. `#0071e3` → `#1baf7a`

**Public user shape (`publicUser`).** This is all that ever leaves the server; it never includes `salt` or `passwordHash`:
- `{ id, name, email, role, company||'', avatarGradient, initials, createdAt, orgRole||null, managerId||null, teamId||null, title||'' }`

**Initials algorithm:**
- Source is `(name || email || '').trim()`. If empty → `'?'`.
- Split on whitespace. With 2 or more parts → first character of each of the first two parts, uppercased.
- Otherwise → the first 2 characters, uppercased.

#### 6.2.2 Password hashing

- `crypto.scryptSync(String(password), saltHex, 64).toString('hex')`.
- Uses Node's default scrypt parameters (N=16384, r=8, p=1).
- The salt is `crypto.randomBytes(16).toString('hex')` and is generated per user. A new salt is generated on every password change.
- Verification hashes the attempt and compares with `crypto.timingSafeEqual`. A length mismatch returns false.

#### 6.2.3 Sessions and tokens

- A token is `uid('tok')`, e.g. `tok-<ts36>-<5 chars>`.
  - Gap: despite the "opaque random" comment, it uses `Math.random`, not a CSPRNG.
- Sessions are stored as `sessions[token] = userId`.
- **No expiry, no rotation, no cap.** Sessions accumulate forever; the live data has 79 sessions for 27 users.
- Logout deletes only the presented token.
- A password change does not revoke other sessions.
- A role change takes effect on the next request, because `me()` re-reads the user every time.
- Transport: `Authorization: Bearer <token>`.
- The client keeps the token in `localStorage['zr_token']`.

#### 6.2.4 Operations and exact errors

All of these surface as HTTP 400 `{ok:false,error}` through `authError`.

**`signup({name,email,password,company})`**
- Errors, checked in this order:
  - `'Please enter your name.'`
  - `'Please enter a valid email address.'` (regex `/^[^@\s]+@[^@\s]+\.[^@\s]+$/`)
  - `'Password must be at least 6 characters.'`
  - `'An account with that email already exists.'`
- Role: `'Owner'` if `users.length === 0`, otherwise `'Editor'`.
  - Edge: the demo org is seeded at server start, so on a fresh install users already exist and **no signup ever becomes Owner**. The seeded Admin persona is then the top account.
- Mints a session and returns `{token, user}`.
- Records no audit event.

**`login({email,password})`**
- Normalises the email.
- Returns the same error, `'Incorrect email or password.'`, for an unknown email and for a wrong password, so it does not reveal which emails exist.
- Mints a new token and returns `{token,user}`.
- Records the audit event `account.login`.

**`me(token)`** returns the public user or null. A missing token, unknown token, or deleted user all give null.

**`logout(token)`** always returns `{ok:true}`. It records the audit event `account.logout`; if the token is invalid, the actor becomes "A teammate".

**`updateProfile(token, patch)`**
- Errors:
  - `'You are not signed in.'`
  - `'Account not found.'`
- Name, if present: trimmed. Empty → `'Name cannot be empty.'`. Initials are recomputed.
- `company`, if present: trimmed.
- If `patch.password` is truthy:
  - It must be at least 6 characters, else `'New password must be at least 6 characters.'`.
  - `patch.currentPassword` must verify, else `'Your current password is incorrect.'`.
  - A new salt and hash are written.
- Saves once at the end, so an error leaves nothing persisted.
- Email cannot be changed.
- Records the audit event `account.profile`.

**`listUsers()`** returns all users as public shapes, in file order.

**`setUserRole(id, role)`**
- Errors:
  - `'Unknown role.'` if the role is not one of the four in `ROLES`. Legacy `'Member'` cannot be set.
  - `'Account not found.'`
  - `'You cannot remove the last Owner.'`
- No other protection:
  - an Admin may promote anyone, including to Owner;
  - an Admin may demote an Owner when there is more than one;
  - the server allows changing your own role (only the UI disables that).

**`getUserById(id)`** returns the public user or null.

**`patchUserOrg(id, patch)`**
- `'Account not found.'`
- `orgRole`: must be in `ORG_ROLES` or null, else `'Unknown org role.'`.
- `managerId` / `teamId`: falsy → null.
- `title`: `String(...).slice(0,80)`.
- Only the keys provided are changed.

**`createManagedUser({...})`**
- Used for seeding and provisioning. It does not mint a session.
- Same validations as signup, except the name error is `'Please enter a name.'`.
- `role`: must be in `ROLES`, else `'Editor'`.
- `orgRole`: must be valid, else null.

**`userCount()`** returns `users.length`.

### 6.3 Role model and authorization

Access is controlled on two independent axes.

#### 6.3.1 Workspace permission role (gates edit versus view)

- `ROLES = ['Owner','Admin','Editor','Viewer']`
- `ROLE_RANK = { Viewer:1, Editor:2, Member:2 (legacy alias of Editor), Admin:3, Owner:4 }`. An unknown role ranks 0.

#### 6.3.2 Org role (where a person sits in the org chart)

- `ORG_ROLES = ['admin','manager','lead','member']`, or null.

#### 6.3.3 Server middleware (`index.js`)

**Global auth wall.** Every `/api/*` request needs a valid token, otherwise 401 `'Please sign in to continue.'`. The exceptions are the `PUBLIC_API` set:
- `POST /api/auth/login`
- `POST /api/auth/signup`
- `GET /api/auth/me`
- `POST /api/auth/logout`
- `GET /api/org/personas`
- `GET /api/currency`
- `GET /api/health`
- plus `GET /api/planning/calendar.ics`

**`requireRole(min)`**
- If `userCount()===0` or there is no token → `req.user=null` and the request passes. This is the legacy "open demo" branch; the global wall now makes it effectively unreachable.
- Invalid token → 401 `'Your session has expired — please sign in again.'`
- Rank too low → 403 `` `This action requires the ${min} role or higher.` ``
- Otherwise sets `req.user`.

**`requirePlanner`** (board configuration):
- Allowed: org role lead or manager, or workspace rank Admin or higher.
- Otherwise 403 `'Board customization is available to team leads and managers.'`

**Approval guards:**
- `approvalGuard`: if an approver is assigned, only that person may act — matched by `approverId`, otherwise by name. Error: `` `Only ${name||'the assigned approver'} can approve or reject this plan.` ``
- `tacticApprovalGuard`: with no assigned approver, anyone with Admin+ rank or org role lead/manager/admin may act. Error: `'You don’t have approval rights for tactics — ask a team lead or manager.'`

**Gates for this area:**

| Endpoint | Gate |
|---|---|
| `GET /api/auth/users`, `PUT /api/auth/users/:id/role` | Admin |
| `GET /api/org/users`, `PUT /api/org/users/:id` | Admin |
| `POST/PUT/DELETE /api/org/teams…` and `/lead`, `/members`, `/campaigns` | Admin |
| `GET /api/org/tree│teams│teams/:id│campaigns│dashboard│me│people` | any signed-in user |
| `GET /api/notifications`, `POST /api/notifications/read` | any signed-in user |
| `GET /api/audit` | any signed-in user (scoped inside the service) |
| `/api/trash*` | Editor |
| `DELETE /api/plans/nodes/:id` | Admin |
| Tactic, line-item and PR deletes | Editor |
| Builder dashboard create/update/delete | **no role gate**, only the wall. Gap: Viewers can delete dashboards via the API |

#### 6.3.4 Client gating

- `canEdit = !user || user.role !== 'Viewer'`. Used in Planning, PlansView, Procurement, Builder, Dashboards and Integrations.
- The "Admin" sidebar link and the Admin page require workspace role Owner or Admin.
  - Org role `admin` alone does **not** unlock the Admin page.
  - It **does** unlock the workspace-wide audit view.
- Users with org role manager, lead or member get `TeamDashboard` as home (`/`). Everyone else, including org admins and users with no placement, gets the saved-dashboards viewer.
- The client never refreshes `user` after the initial `/auth/me` or a login. Role or org changes made by an admin only show after a reload or re-login.

### 6.4 Organisation (`org.js`, file `teams.json`)

#### 6.4.1 Team record

| Field | Rules |
|---|---|
| `id` | `uid('team')` |
| `name` | slice 80; default `'New team'` |
| `focus` | slice 120 |
| `color` | default `TEAM_COLORS[teams.length % 6]` |
| `leadId` | userId or null; no validation that it is a lead or even exists |
| `memberIds` | deduped array |
| `campaignIds` | deduped array of plan nodes with `level === 'campaign'` |
| `createdAt`, `updatedAt` | ISO |

`TEAM_COLORS = ['#2a78d6','#9f4bd6','#1baf7a','#eb6834','#e34980','#0071e3']`. The client duplicates the same list.

The file's `meta` is `{ orgSeeded: boolean }`.

#### 6.4.2 Hierarchy semantics

```
admin user (orgRole admin)
  ← manager.managerId
manager
  ← lead.managerId
lead
  ← team.leadId
team
  → team.memberIds
member  (also user.teamId; single-team intent)
```

#### 6.4.3 Self-heal (`healCampaignLinks`, runs on every `load()` when teams exist)

- Skipped if there are no campaign nodes.
- For each team: if **none** of its `campaignIds` resolve to a live campaign node, and the team name is in `TEAM_CAMPAIGNS`, remap by campaign name and save.
- This keeps team-to-campaign rollups working after the plan hierarchy is re-seeded with new ids.
- `TEAM_CAMPAIGNS` mapping:
  - Paid Acquisition Squad → Nova GA Launch, Enterprise ABM
  - Events & Webinars Crew → Nova GA Launch, Brand & Community
  - Lifecycle & Email Team → Nova GA Launch
  - Content Engine Team → Brand & Community
  - Named-Account Team → Enterprise ABM
  - Partner Co-Marketing Team → Brand & Community, Enterprise ABM
- Edge: if an admin clears all campaigns from a seeded team, the next load restores them (an empty list counts as "none resolve").
- A malformed `teams.json` loads as `{meta:{orgSeeded:false},teams:[]}`.

#### 6.4.4 Rollups

**`teamWork(team)`** looks at tactics whose `campaignId` is in the team's set and returns:
- `campaigns`: `[{id,name}]`, only those that still resolve
- `tacticCount`
- `inFlight`: status `in-progress`
- `launched`
- `plannedBudget`: sum of `t.budget`
- `avgProgress`: rounded mean of `t.progress`, or 0
- `statusBreakdown`: seeded with `{backlog,planned,'in-progress',launched}`; any other status is added dynamically

**`campaignMatrix(teams)`** returns one row per campaign touched by the given teams, in order of first appearance:
- `{ campaignId, campaignName, teams:[{id,name,color}], tacticCount, inFlight, launched, plannedBudget, avgProgress }`
- The metrics cover the whole campaign (all of its tactics), not just the share of one team.

**`teamCard(team)`** returns `{...team, lead: publicUser|null, members: publicUser[] (missing users skipped), memberCount, work: teamWork}`.

#### 6.4.5 Read models

**`orgTree()`** returns:
- `admins`: users with org role admin, each with `managers` = managers whose `managerId === admin.id`. Each manager has `leads` (orgRole lead, `managerId === m.id`); each lead has `teams` (team cards with `leadId`).
- `orphanManagers`: managers with no `managerId`, or whose `managerId` is not an admin.
- `counts`: `{managers, leads, teams, members}`, where `members` counts users with org role member.
- Leads without a manager and teams without a lead do not appear in the tree.

**`dashboardFor(user)`**, by org role:
- No user → `{role:null}`.
- **manager:**
  - leads = org role lead with `managerId === me`
  - teams = teams led by those leads
  - returns `{role, me, stats:{leads, teams, members: distinct memberIds, campaigns: matrix.length}, leads: lead + teams[], teams, matrix}`
- **lead:**
  - teams = teams with `leadId === me`
  - returns `{role, me, manager, stats:{teams, members, campaigns}, teams, matrix}`
- **member:**
  - team = the first team whose `memberIds` include me, otherwise the team at `user.teamId`
  - none found → `{role:'member', me, team:null}`
  - otherwise `{role, me, team: card, lead, matrix: [team]}`
- **Anything else** (admin, or no org role) → `{role:'admin', me, counts, teams: all cards, matrix: all}`.
  - Gap: a user with no org role receives the org-wide payload if they call the API.

**`orgContextFor(user)`** returns `{orgRole, title, manager, team:{id,name,color}|null}`. It is served at `GET /api/org/me`, which the UI never calls.

**`listCampaigns()`** returns `[{id, name, owner}]` for campaign-level nodes.

#### 6.4.6 Team CRUD (the routes are Admin-gated)

**`createTeam`**
- Builds the record, saves, then runs `syncMemberTeamLinks`: for each member whose `teamId !== team.id`, it calls `patchUserOrg(teamId = team.id, orgRole = existing || 'member')`.

**`updateTeam(id, patch)`**
- `'Team not found.'`
- Patches `name/focus/color/leadId/memberIds/campaignIds`, bumps `updatedAt`, and runs sync if members changed.
- Edge: members **removed** from a team keep a `teamId` pointing at that team. Only a team delete clears it.

**`deleteTeam(id)`**
- `'Team not found.'`
- Clears `teamId` for members whose `teamId === id`, then removes the team.
- Returns `{id, remaining}`.
- This is a **hard delete**; teams never go to the recycle bin.

**`setTeamLead`, `setTeamMembers`, `setTeamCampaigns`** are wrappers around `updateTeam`.

**`setUserOrg(id, patch)`**
- Calls `patchUserOrg`. If `patch.teamId` is set, it also pushes the user into that team's `memberIds`.
- Gap: it never removes the user from their previous team's `memberIds`, and clearing `teamId` does not remove membership. A member view can then resolve to the old team, because `memberIds` are checked before `teamId`.

#### 6.4.7 People search (`searchPeople(q, limit=50)`, route `GET /api/org/people?q&limit`)

- Matching: case-insensitive substring on `"${name} ${email} ${title}"`, using the trimmed, lowercased query.
- Ranking: org role `admin` 0, `manager` 1, `lead` 2, `member` 3, none 4. Ties are broken by `name.localeCompare`.
- Limit: clamped to [1, 200]. The route passes `Number(limit) || 50`.
- Gap: the client picker loads with an empty query and therefore only ever gets the **first 50** people.

#### 6.4.8 Demo personas (`demoPersonas()`, public route `GET /api/org/personas`)

- Returns users whose email ends with `@zenreach.demo`, mapped to `{name, email, orgRole, title, team, avatarGradient, initials}`.
- `team` = the team they lead, otherwise the team they are a member of, otherwise `''`.
- Sorted admin 0, manager 1, lead 2, member 3, unknown 9, then by name.
- The response is `{ personas, password: DEMO_PASSWORD }`.
- `DEMO_PASSWORD` is a hard-coded shared constant in `org.js`. The **public, unauthenticated** endpoint returns it, and the login modal and Admin "Demo access" tab display it. That means anyone can sign in as the seeded **Admin**-role persona.
  - Rebuild: put this behind a demo-mode flag and source the value from environment or config.

#### 6.4.9 Seed (`seedOrg()`, called at server import; idempotent)

1. **Skip condition:** skips if `meta.orgSeeded && teams.length`. Every person is created via `ensurePerson`, which skips an email that already exists.
2. **Emails:** `slugEmail(name)` lowercases, replaces each run of non-letters with `.`, trims leading and trailing dots, and appends `@zenreach.demo`. All demo accounts get `company:'ZenReach'` and the shared demo password.
3. **Owner promotion:** if a workspace Owner exists, set their org role to `admin` and their title to the existing title or `'Workspace Owner'`.
4. **Demo admin:** "Ada Admin", `admin@zenreach.demo`, role **Admin**, org role admin, title "Workspace Admin".
5. **Managers** (role Editor, `managerId` = Ada):
   - Marcus Lee — "VP, Demand Generation"
   - Sam Okafor — "VP, Brand & Content"
6. **Leads** (role Editor):
   - Abhishek Biyani — "Demand Gen Lead", reports to Marcus. In code the variable is `priya`, a renamed persona.
   - Dana Alvarez — "Lifecycle Lead", reports to Marcus
   - Jordan Kim — "Content Lead", reports to Sam
   - Alex Rivera — "ABM Lead", reports to Sam
7. **Teams** (blueprint index i; color `TEAM_COLORS[i]`; campaigns from `TEAM_CAMPAIGNS`):

   | i | Team | Focus | Lead | Members |
   |---|---|---|---|---|
   | 0 | Paid Acquisition Squad | Paid search & social | Abhishek | Ravi Menon, Chloe Baker, Diego Santos |
   | 1 | Events & Webinars Crew | Field & digital events | Abhishek | Hana Suzuki, Liam Walsh, Noor Haddad |
   | 2 | Lifecycle & Email Team | Nurture & retention | Dana | Emma Novak, Tariq Aziz, Sofia Rossi |
   | 3 | Content Engine Team | Content & SEO | Jordan | Owen Clarke, Mia Fontaine, Kabir Shah |
   | 4 | Named-Account Team | ABM plays | Alex | Grace Park, Mateo Duarte, Ivy Chen |
   | 5 | Partner Co-Marketing Team | Partnerships | Jordan | Sara Kovac, Ben Adeyemi, Lena Fischer |

8. **Members:** 18 in total, role Editor, org role member. Title is `MEMBER_TITLES[(i+j)%6]`, where j is the member's position in the team:
   - `['Campaign Manager','Marketing Specialist','Growth Marketer','Content Strategist','Marketing Analyst','Program Coordinator']`
   - A team that already exists (matched by name) is not recreated. Members' `teamId` and org role (member) are always re-pointed.
9. **Finish:** sets `meta.orgSeeded = true` and saves.

The blueprint is 1 demo admin, 2 managers, 4 leads, 6 teams and 18 members.

### 6.5 Notifications (`notifications.js`, file `notifications.json`)

The feed is strictly per user: no user means `{count:0,total:0,items:[]}`. The user is resolved from the bearer token. The client also sends `?user=<name>`, which the server **ignores**.

#### 6.5.1 Item shape

```
{ id, type, title, text, at (ISO), route, [from], [authored:true] }
```
`type` is one of `approval | due | overdue | mention | assignment`.

#### 6.5.2 A. Derived items (recomputed on every request, matched by display **name**)

**1. Approvals waiting on me**
- For each purchase request with `status==='submitted'`:
  - take its pending approvals;
  - the active level is the minimum `level` among them;
  - the request is mine if there is a pending step at the active level with `approver === user.name`.
- Item:
  - id `approval-${pr.id}`, type `approval`
  - title `'Approval needed'`
  - text `` `${pr.title} awaits your sign-off (${step.name})` ``
  - at `pr.updatedAt`, route `/procurement`

**2. Due or overdue tactics I own**
- Tactics where `status !== 'launched'` and `owner === user.name`.
- `due = endDate || dueDate`; tactics with neither are skipped.
- `days = Math.round((new Date(`${due}T00:00:00`) - now) / 86400000)`. This uses server local time.
- Included if `days <= 14`. There is no lower bound, so arbitrarily old overdue items still appear.
- Overdue when `days < 0`.
- Item:
  - id `due-${t.id}`, type `overdue` or `due`
  - title `'Overdue'` or `'Due soon'`
  - text `` `${title} was due ${|days|}d ago` `` or `` `${title} is due in ${days}d` ``
  - at `` `${due}T09:00:00.000Z` ``, route `/planning`

**3. Mentions**
- First name = the first whitespace token of `user.name`.
- Regex: `new RegExp('@'+firstName+'\\b','i')`. The name is not regex-escaped.
- Scans only **tactic** comments (`t.comments[]`, each `{id: uid('cm'), author, text, at}`). Plan-node comments are not scanned.
- Item:
  - id `mention-${c.id}`, type `mention`
  - title `'You were mentioned'`
  - text `` `${c.author} on “${t.title}”: ${c.text.slice(0,80)}` ``
  - at `c.at`, route `/planning`
- Matches everyone who shares that first name, including self-mentions.

The derived items are sorted by `at` descending (string comparison) and cut to **30**.

#### 6.5.3 B. Pushed ("authored") items

**`pushNotification({toUserId, type='assignment', title, text='', route='/', from=''})`**
- Returns null unless both `toUserId` and `title` are set.
- id: `push-${Date.now().toString(36)}-${rand5}`; `at` = now; `authored: true`.
- Prepended to `entry.pushed` and capped at **50** per recipient.
- Keyed by the recipient's **user id**, never their name.

**`notifyAssignment` (index.js)** wraps it:
- no-op when there is no `toUserId`;
- no-op when `actor.id === toUserId` (you are never notified about your own action);
- errors are swallowed.

**Pushed triggers.** In the text column, `X` is the actor's name, or `'Someone'` if there is none.

| Event (route) | Recipient | type | Title | Text | Route |
|---|---|---|---|---|---|
| Create plan node (`POST /api/plans/nodes`) with `ownerId` | owner | assignment | `You’re the owner of a ${level││'plan'}` | `X made you owner of “name”.` | /planning |
| Update node when `ownerId` changed | new owner | assignment | same | same | /planning |
| Submit campaign plan (`/planning/plans/:cid/submit`) | `approverId` | approval | `Plan approval needed` | `X asked you to approve the plan for “name”.` | `/planning?plan=spa:<cid>` |
| Create tactic with `ownerId` / update when `ownerId` changed | owner | assignment | `A tactic was assigned to you` | `X made you owner of “title”.` | /planning |
| Submit tactic | `approverId` | approval | `Tactic approval needed` | `X asked you to approve “title”.` | /planning |
| Approve or decline tactic | previous `ownerId` | approval | `Tactic approved` / `Tactic declined` | `X approved│declined “title”: <comment>` or ending in `.` | /planning |
| Bulk submit (at least 1 submitted) | `approverId` | approval | `Tactic approvals needed` | `X sent you N tactic(s) to approve.` | /planning |
| Create PR with `requesterId` | requester | assignment | `A purchase request names you as requester` | `X created “title” with you as the requester.` | /procurement |
| Submit PR | `pr.approverId` | approval | `Approval needed` | `X submitted “title” for your approval.` | /procurement |
| Team create | each member; lead | assignment | `You were added to a team` / `You’re now a team lead` | `X added you to “team”.` / `X made you lead of “team”.` | /admin |
| Team update, `PUT …/lead`, `PUT …/members` | newly added members only; lead only if changed | assignment | same | same | /admin |

These events do **not** notify anyone: PR approve or reject, plan approve or reject, comments, deletes, role changes.

A submitted PR produces **two** items for its approver: the pushed "submitted … for your approval" and the derived "awaits your sign-off".

#### 6.5.4 Merge, read state and persistence

- **Merging:** `mergedItems = [...pushed, ...derived(30)]`, sorted by `at` descending, cut to **40**.
- **`notificationsFor(user)`:**
  - `seenIds = prev.seenIds` filtered to ids that are still present (self-pruning);
  - `count` = items not in `seenIds`;
  - then **writes** `{pushed, items, seenIds, total, updatedAt}` on every GET. The client polls every 30 s, so this is a disk write per poll per client.
  - returns `{count, total, items}`.
- **`markRead(user)`** sets `seenIds` to all current item ids and returns `{count:0,total,items}`.
- **Read model:** the "unread" state applies to the whole feed, not to individual items. A brand-new id makes the badge reappear. Nothing is dismissed or deleted; pushed items only fall out after 50, or when they drop below the merged top 40.
- **Migration:** older entries have the shape `{count, items, updatedAt}` and no `seenIds`/`pushed`. They are read as all unread, then overwritten with the new shape on the next fetch.

#### 6.5.5 Bell UI (`NotificationBell.jsx`)

- **Placement:** rendered only when signed in. It appears in the sidebar footer next to `AccountControls` and the theme toggle, and in the mobile top bar.
- **Fetching:** on mount and whenever `user` changes, then `setInterval` every **30000 ms**. Fetch errors are ignored.
- **Button:** `IconButton` (aria-label "Notifications") with `Badge` showing the unread count. Badge color is error, `max={9}` (shows "9+"), font 9 px, 15 px high.
- **Opening:** opens a Popover anchored bottom-right. If `count > 0`, it optimistically sets count to 0, then `POST /notifications/read` and replaces the data with the response.
- **Popover:**
  - width 340, max 92vw
  - header "Notifications"
  - list scrolls at max height 380 px
  - each item: icon by type, bold title (12.5 px), text (12 px)
  - clicking an item closes the popover and navigates to `route || '/'`
- **Empty state:** "You're all caught up."
- There are no timestamps, no per-item unread styling, no "mark all read", and no dismiss.
- **Icons:**

  | type | Icon | Color |
  |---|---|---|
  | approval | CheckCircleRounded | `#0071e3` |
  | due | ScheduleRounded | `#c77700` |
  | overdue | WarningAmberRounded | `STATUS.critical #d03b3b` |
  | mention | AlternateEmailRounded | `#9f4bd6` |
  | assignment (also the fallback for unknown types) | PersonAddAlt1Rounded | `#1aa179` |

### 6.6 Audit log (`audit.js`, file `audit.json`)

#### 6.6.1 Event record

| Field | Rules |
|---|---|
| `id` | `uid('evt')` |
| `at` | `meta.at` (used by the seed to backdate) or now |
| `action`, `category`, `severity` | from the action registry |
| `actorId` | `''` if anonymous |
| `actorName` | default `'A teammate'` |
| `actorRole`, `actorOrgRole`, `actorInitials`, `actorGradient` | snapshot of the actor at write time, so the log stays truthful after a rename or deletion |
| `entity` | `opts.entity`, otherwise the spec noun |
| `entityId` | String |
| `label` | slice 160 |
| `summary` | `opts.summary`, otherwise `${verb}${noun?' '+noun:''}${label?' “'+label+'”':''}` (the actor name is shown separately) |
| `meta` | sanitised: drops the `at` key, null values and objects; values are stringified and anything over 200 characters becomes `slice(0,200)+'…'` |

- **Retention:** `MAX_EVENTS = 5000`. After a push the log keeps the last 5000; the oldest are dropped.
- **Malformed file:** treated as `{events:[]}`.
- **Write path:** `audit(req, action, opts)` in index.js runs after the mutation succeeds. The actor is `opts.actor` if given, otherwise `req.user`, otherwise `me(token)`, otherwise null. Errors are swallowed.

#### 6.6.2 Action catalog (`ACTIONS`): action → category / verb / noun / severity

- **planning:**
  - `plan.create` — created / plan node / info
  - `plan.update` — updated / plan node / info
  - `plan.delete` — deleted / plan node / warning
- **tactics:**
  - `tactic.create` — created / tactic / info
  - `tactic.update` — updated / tactic / info
  - `tactic.delete` — deleted / tactic / warning
  - `tactic.bulk_delete` — bulk-deleted / tactics / warning
  - `tactic.comment` — commented on / tactic / info
- **approvals:**
  - `plan.submit` — submitted for approval / plan / notice
  - `plan.approve` — approved the plan for / campaign / notice
  - `plan.reject` — rejected the plan for / campaign / warning
  - `pr.submit` — submitted for approval / purchase request / notice
  - `pr.approve` — approved / purchase request / notice
  - `pr.reject` — rejected / purchase request / warning
- **budget:**
  - `budget.pool` — saved / budget pool / info
  - `budget.allocation` — changed an allocation in / budget pool / info
  - `budget.snapshot_create` — captured / budget snapshot / info
  - `budget.snapshot_delete` — deleted / budget snapshot / warning
- **dashboards:**
  - `dashboard.create` — created / dashboard / info
  - `dashboard.update` — updated / dashboard / info
  - `dashboard.delete` — deleted / dashboard / warning
- **procurement:**
  - `pr.create` — created / purchase request / info
  - `pr.update` — updated / purchase request / info
  - `pr.advance` — advanced fulfillment of / purchase request / info
  - `pr.delete` — deleted / purchase request / warning
- **org:**
  - `team.create` — created / team / info
  - `team.update` — updated / team / info
  - `team.delete` — deleted / team / warning
  - `user.role` — changed the permission role of / member / **critical**
  - `user.org` — updated the org placement of / member / info
- **imports:**
  - `import.tactics` — imported / tactics from a CSV / info
  - `import.trello` — imported / a Trello board / info
  - `import.jira` — imported / a Jira project / info
  - `import.dataset` — imported / a campaign dataset / info
- **recycle:**
  - `recycle.restore` — restored / a deleted item / notice
  - `recycle.purge` — permanently deleted / an item / critical
  - `recycle.empty` — emptied / the recycle bin / critical
- **account:**
  - `account.login` — signed in / (no noun) / info
  - `account.logout` — signed out / (no noun) / info
  - `account.profile` — updated / their profile / info
- **ai:**
  - `ai.plan_commit` — committed / an AI-generated plan / notice

`DEFAULT_ACTION = {category:'other', verb:'updated', noun:'item', severity:'info'}`.

Gap: these actions are emitted by routes but **not registered**, so they fall back to "other / updated item “X”":
- `model.create`, `model.update`
- `plan.comment`
- `tactic.submit`, `tactic.approve`, `tactic.decline`
- `tactic.bulk_move`, `tactic.bulk_copy`, `tactic.bulk_submit`
- `lineitem.create`, `lineitem.update`, `lineitem.delete` (the last is logged as "info" even though it is a delete)

The rebuild should register them.

`CATEGORY_LABELS` (this order is the filter order):
- planning: Planning
- tactics: Tactics
- approvals: Approvals
- budget: Budget
- procurement: Procurement
- dashboards: Dashboards
- org: Organization
- imports: Imports
- recycle: Recycle bin
- account: Account
- ai: AI plans
- other: Other

**Labels and meta passed by routes:**
- `plan.delete`: meta `{removed, tactics}`
- `plan.reject`: meta `{note}`
- `pr.create`: meta `{total}`
- `pr.submit`: meta `{approver}`
- `pr.advance`: meta `{stage}`
- `user.role`: label = target's name, meta `{role}`
- `user.org`: meta `{orgRole, title}`
- `recycle.restore`: label = restored item's label
- `recycle.purge`: only `entityId`, and it is recorded before the purge
- `recycle.empty`: no label
- `budget.snapshot_delete`: no label
- `account.login`: actor = the user just logged in
- signup records no event

#### 6.6.3 Visibility scoping (`visibleActorIds`)

- **`canSeeAll(user)`** is true when there is no user (anonymous; unreachable behind the wall), when `roleRank(role) >= rank('Admin')` (workspace Owner or Admin), or when `orgRole === 'admin'`. These viewers see every event, including those with an empty `actorId`.
- **manager:** self, plus **every user whose `managerId === me`** (any org role), plus all members of teams whose lead is one of those direct reports with org role lead.
- **lead:** self, plus members of every team they lead.
- **anyone else** (member, Viewer, no org role): self only.

Scope descriptor (`key` / `label` / `description`):
- `all` / "Workspace-wide" / "Every action taken across the entire workspace."
- `org` / "Your organization" / "Activity from you, your team leads, and their team members."
- `team` / "Your teams" / "Activity from you and the members of the teams you lead."
- `self` / "Your activity" / "Your own actions across the workspace."

#### 6.6.4 Read (`logsFor(user, {category, severity, actorId, q, limit})`, route `GET /api/audit`)

1. `visible`: the in-scope events, newest first (the array is iterated from the end).
2. `categories`: built from `CATEGORY_LABELS` order as `{key, label, count}`, keeping only `count > 0` within `visible`.
3. `actors`: built from `visible` events with an `actorId`, as `{id, name, initials, avatarGradient, count}`, sorted by name. Out-of-scope people are never shown as options.
4. Filters, applied in order:
   - category equals
   - severity equals
   - `actorId` equals
   - `q`: lowercase substring of `"${actorName} ${summary} ${label}"`
5. `limit`: clamp [1, 500], default 200.
6. Response:
   - `scope: {...descriptor, canSeeAll}`
   - `stats: { total: visible.length, today: count where server-local date is today, people: actors.length, critical: count with severity warning OR critical }`
   - `categories`, `actors`
   - `events: filtered.slice(0, limit)`
   - The stats always describe the whole scoped set, not the filtered or limited one.

#### 6.6.5 Seed (`seedAudit()`, at startup, wrapped in try/catch)

- Runs only if the log is empty and at least one user exists.
- Actors:
  - `admin` = the first user with org role admin (in file order), otherwise the first user;
  - managers, leads and members are taken by org role;
  - `pick(arr, i)` cycles through the array and falls back to `admin`.
- Labels: `campName(i)` / `tacName(i)` cycle through live campaigns and tactics, with fallbacks `'Nova GA Launch'` and `'Paid search — brand'`.
- Timestamps: `daysAgo(d, h)` = today minus d days, at hh:15 local time.
- 24 scripted events, as `[actor, action, label, daysAgo, hour]`:
  1. admin, `account.login`, —, 13, 8
  2. admin, `team.create`, "Paid Acquisition Squad", 13, 9
  3. manager0, `plan.create`, camp0, 12, 9
  4. lead0, `tactic.create`, tac0, 12, 11
  5. member0, `tactic.update`, tac1, 12, 14
  6. lead1, `budget.pool`, "camp1 — Working budget", 11, 10
  7. member1, `tactic.create`, tac2, 11, 13
  8. lead0, `plan.submit`, camp0, 10, 9
  9. manager0, `plan.approve`, camp0, 10, 15
  10. member2, `pr.create`, "Q3 paid media — Google Ads", 9, 11
  11. member2, `pr.submit`, same, 9, 12
  12. lead1, `pr.approve`, same, 8, 10
  13. member3, `dashboard.create`, "Demand-gen performance", 7, 14
  14. lead0, `import.tactics`, —, 6, 9
  15. manager1, `budget.snapshot_create`, "End-of-month baseline", 5, 16
  16. member4, `tactic.comment`, tac3, 5, 11
  17. lead1, `plan.reject`, camp2, 4, 15
  18. admin, `user.role`, member5's name, 3, 10
  19. member0, `tactic.delete`, tac4, 2, 13
  20. lead0, `recycle.restore`, tac4, 2, 14
  21. manager0, `dashboard.update`, "Pipeline & ROI", 1, 9
  22. member6, `tactic.update`, tac5, 1, 12
  23. admin, `account.login`, —, 0, 8
  24. lead0, `pr.advance`, "Event sponsorship — SaaStr", 0, 11
- Each event carries `meta: {at, seed:'true'}`. `at` is stripped from the stored meta, so `seed:'true'` remains.

### 6.7 Recycle bin (`recycleBin.js`, file `recycleBin.json`)

#### 6.7.1 Approach

Snapshot-then-hard-delete. At delete time the entity (or the whole subtree) is copied into the bin, then removed from its live store as normal. Restore re-inserts the snapshot verbatim with its original ids, so no read path ever filters "deleted" flags.

#### 6.7.2 Trash entry

```
{ id: uid('trash'), deletedAt: ISO, type, label, sublabel, count, by: <actor display NAME or ''>, payload }
```
- New entries are prepended (`unshift`), so the bin is newest first.
- No cap, no retention or auto-expiry, no per-user scoping. Any Editor+ sees, restores and purges everything.
- The list endpoint strips `payload`.
- A malformed file, or an `items` value that is not an array, self-heals to `[]`.

#### 6.7.3 Trash wrappers by type

**`node`: plan, campaign or program (`DELETE /api/plans/nodes/:id`, Admin)**
- Computes the descendant closure from `parentId`.
- Collects every tactic whose `programId`, `campaignId` or `planId` is in that set. Line items are included because they live inside the tactic.
- Calls `deleteNode(id)` (removes the subtree), then `deleteTactics({ids})`.
- Entry fields:
  - `label`: root name, or `'Plan node'`
  - `sublabel`: `[LEVEL_WORD[level]||'Node', (nodes>1 ? 'N items' : ''), (tactics ? 'M tactic(s)' : '')].join(' · ')`
    - `LEVEL_WORD = {plan:'Plan', campaign:'Campaign', program:'Program'}`
    - N includes the root
  - `count`: nodes + tactics
  - `payload`: `{nodes, tactics}`
- Returns `{removedIds, trash, removedTactics}`.
- Not found → `{removedIds:[], trash:null}`.
- **Legacy payload** `{nodes, tacticLineage:[{id,planId,campaignId,programId}]}` comes from an older version, which re-homed tactics instead of deleting them.

**`lineItem` (`DELETE /api/tactics/:id/line-items/:liId`, Editor)**
- `removeLineItem` updates the tactic without that item.
- Entry:
  - `label`: the line item's name
  - `sublabel`: `` `Line item · ${tactic.title||'tactic'}` ``
  - `count`: 1
  - `payload`: `{tacticId, lineItem}`
- Returns `{trash, tactic}`.
- The route returns 404 for a missing tactic or line item.

**`tactic` (`DELETE /api/tactics/:id`, plus the legacy `DELETE /api/planning/initiatives/:id`)**
- Entry: `label` = title, `sublabel` = `'Tactic'`, `payload` = `{tactic}`.
- Not found → `{trash:null}`.

**`tactics` (`POST /api/planning/tactics/bulk-delete {ids}|{all:true}`)**
- Entry:
  - `label`: `` `${n} tactic(s)` ``
  - `sublabel`: `'Cleared all tactics'` or `'Bulk delete'`
  - `payload`: `{tactics}`
- No targets → `{trash:null, result:{deleted:0, remaining}}`.

**`dashboard` (`DELETE /api/builder/dashboards/:id`; no role gate)**
- `label` = `d.name`, `sublabel` = `'Dashboard'`, `payload` = `{dashboard}`.

**`request` (`DELETE /api/procurement/requests/:id`, Editor)**
- `label` = `pr.title`, `sublabel` = `'Purchase request'`, `payload` = `{request}`.

All delete routes also write the matching audit event, even when the target was not found (the label is then undefined).

#### 6.7.4 Restore (`POST /api/trash/:id/restore`)

On failure the route returns 400 with `error`. On success it returns `{ok:true, type, label}`, removes the entry and records the audit event `recycle.restore`.

**Unknown trash id** → `'This item is no longer in the recycle bin.'`

**`node`**
- Finds the snapshot root: the node with no `parentId`, or whose parent is not in the snapshot.
- If `root.parentId` is not live → `` `Its parent no longer exists — restore the parent ${root.level==='program' ? 'campaign' : 'plan'} first.` ``
- `insertNodes`: skips ids that already exist. Campaign nodes with a truthy `status` come back with `status:'Draft', approval:null`.
- Tactics:
  - With a `tactics` payload: `insertTactics(tactics.map(asRestored))`.
  - With a legacy `tacticLineage` payload: for each tactic that still exists, `updateTactic` back to its original plan, campaign and program.

**`tactic` / `tactics`**
- `insertTactics(map(asRestored))`, which skips existing ids.
- There is **no parent-existence check**, so a tactic can come back pointing at a deleted program or campaign.

**`asRestored(t)`**
- Sets `approvalStatus:'Created'`.
- Appends `{at: now, by:'', action:'restored', comment:'Restored from the recycle bin — re-submit for approval.'}` to `approval.history` (an array is created if missing).

**`lineItem`**
- Calls `restoreLineItem(tacticId, li)`.
- Missing tactic → throws `'Its tactic no longer exists — restore the tactic first.'`, returned as the error.
- Same line-item id already present → no-op.
- Name clash (`sameName` = trim + lowercase equality) → renamed `"name (2)"`, `"(3)"`, …

**`dashboard`**
- Calls `insertDashboard`. An existing id is skipped.
- A name clash with a live dashboard appends `" (restored)"`, repeatedly until unique.

**`request`**
- Calls `insertRequest` verbatim (status unchanged). An existing id is skipped.

**Other type** → `` `Cannot restore unknown item type "${type}".` ``

Any thrown error is caught and returned as `{ok:false, error: err.message}`, and the entry is kept.

Edge: an "already exists" skip still counts as success, so the entry is removed.

#### 6.7.5 Purge and empty

- **`purge(id)`** (`DELETE /api/trash/:id`): filters the entry out and returns `{purged: 0|1}`. A missing id is not an error.
- **`emptyTrash()`** (`POST /api/trash/empty`): writes `{items:[]}` and returns `{purged:n}`.
- Both write critical audit events.

### 6.8 Client UI

#### 6.8.1 App shell (`App.jsx`, `Layout.jsx`)

**Session restore (`AuthProvider`):**
- On mount, reads `zr_token`.
- No token → `ready=true`.
- Token present → `api.me()`:
  - a user comes back → `setUser`;
  - null or an error → clear the token;
  - either way, `ready=true` at the end.
- Context value: `{user, ready, signup, login, logout, updateProfile}`.
  - `logout` is best effort: it clears locally even if the API call fails.
  - `useAuth` outside the provider throws `'useAuth must be used within an AuthProvider'`.

**Client API helper (`api.js`):**
- `setAuthToken` stores the token in `localStorage['zr_token']`, or removes it.
- GET errors throw `` `Request failed: ${path} (${status})` ``, so the **server message is lost**. Example: a Viewer opening the Recycle bin sees "Could not load data: Request failed: /trash (403). Is the API running on port 4000?".
- POST, PUT and DELETE surface `json.error`.
- There is no global 401 handler.

**Routing:**
- While `!ready`: a blank full-screen `bg-graybg`.
- Logged out: only `/` (marketing hero plus a features page whose CTAs dispatch `window` events `zr:open-auth {tab:'signup'|'login'}`). Every other path → `Navigate('/')`.
- Logged in: all routes, including `/admin`, `/trash` and `/audit`.

**Sidebar:**
- MAIN links, with "Admin" appended (AdminPanelSettings icon) only for workspace Owner/Admin.
- MORE section includes "Audit log" (`/audit`, History icon) and "Recycle bin" (`/trash`, RestoreFromTrash icon). Both are visible to all signed-in users.
- Footer: `AccountControls iconOnly={collapsed}`, NotificationBell and ThemeToggle.
- Logged-out top bar: ThemeToggle and `AccountControls`.

#### 6.8.2 `Avatar.jsx`

- A round span (aria-hidden) with semibold white text.
- Props: `size` default 32, `fontSize` default 12.5.
- Background is `user.avatarGradient`, or the default `linear-gradient(135deg,#2a78d6,#9f4bd6)`.
- Shows `initials` or `'?'`.

#### 6.8.3 `AuthModal.jsx`

**Props:** `{open, initialTab='login', onClose, gate=false}`.
- `gate` disables backdrop and Escape close and hides the X. It is currently never used.

**Dialog:** maxWidth xs, fullWidth, radius 3.5. On every open it resets the tab, form, error, password visibility, busy state and persona expander.

**Header** (gradient `#2a78d6→#9f4bd6`):
- logo svg and "ZenReach"
- H2: "Welcome back" (login) or "Create your account" (signup)
- sub: "Sign in to pick up where you left off." or "Start seeing your marketing, beautifully clear."

**Tabs:** "Log in" / "Sign up". Switching clears the error.

**Form:**
- Error `Alert`.
- Signup only: "Full name" (required, autofocus) and "Company (optional)".
- "Email" (type email, required, autofocus on login).
- "Password" (required, visibility toggle). Helper text "At least 6 characters." on signup, a blank line on login.
- Submit, pill shaped: "Please wait…" while busy, otherwise "Create account" / "Log in".
- Divider, then "Don't have an account? **Sign up**" or "Already have an account? **Log in**".
- Success → `onClose()` and `navigate('/')`.
- Error fallback: "Something went wrong. Please try again."

**Persona switcher** (login tab only, shown when personas exist):
- Personas are fetched once, lazily, on first open via `GET /org/personas`. On failure: `{personas:[], password:''}`.
- A full-width bordered button, "Explore as a demo persona", with a chevron that rotates.
- A collapsible list (max height 220 px, scrolls). Each row: Avatar 28, name, `` `${title} · ${team}` ``, and an uppercase org-role pill.
- Footer: "All demo accounts use the password `<code>`".
- Clicking a row logs in with the shared password and navigates to `/`. Error fallback: "Could not sign in as that persona."

#### 6.8.4 `AccountControls.jsx`

**Props:** `{compact=false, iconOnly=false}`.

**Before ready:** a blank 32 px span, so there is no flash of "Log in".

**Signed in:**
- A pill button (aria-label "Account menu") with Avatar and the first name. The name is shown at xl+ only; always when compact; never when `iconOnly`.
- Menu, anchored bottom-right, min width 220:
  - header with Avatar 38, name and email
  - divider
  - "Your profile" (Person icon) → ProfileDialog
  - "Log out" (Logout icon) → logout, then Snackbar "You have been signed out." (top-center, offset 70/78 px, success, filled, 3000 ms)

**Signed out:** "Log in" (text) and "Sign up" (contained) pill buttons, full width when compact.

**Window events:**
- Listens for `zr:open-auth`, only in the non-compact instance, to avoid opening twice.
- Dispatches `zr:auth-modal {open}` on every change. The Hero uses it to pause its animation.

#### 6.8.5 `ProfileDialog.jsx`

**Dialog:** maxWidth xs, title "Your profile" with an X. On open it seeds name and company from `user` and resets everything else.

**Header:** Avatar 52, name, email, and a pill with the **workspace** role.

**Form:**
- "Full name" (required)
- "Company"
- "Email" (disabled)
- Divider, then a "Change password" link that reveals "Current password" (required) and "New password" (required, helper "At least 6 characters.").
- Alerts: error, or success "Profile updated.".

**Actions:**
- "Cancel"
- "Save changes" ("Saving…" while busy). It sends `{name, company}` plus `{currentPassword, password}` when the password section is open.
- On success: collapse the password section, clear both password fields, keep the dialog open.
- Error fallback: "Could not save your profile."
- Edge: the reset effect depends on `[open, user]`. Updating the context user re-runs it and may clear the success message straight away.

**TeamSection** (only for workspace Owner/Admin):
- Divider and "Team & roles".
- Loads `GET /auth/users`; shows "Loading members…" until then.
- One row per user: Avatar 30, name, email, and a role Select (Owner/Admin/Editor/Viewer). The Select is **disabled for yourself**.
- Changing a role → `PUT /auth/users/:id/role`, then a refetch. Errors appear in an Alert.
- Footnote: "Viewers can browse but not edit. Editors can create and change work. Admins manage the team. Owners have full control."

#### 6.8.6 `PeoplePicker.jsx`: the app-wide Teams-style picker (MUI Autocomplete)

**Props:**
- `value`, `onChange`
- `multiple=false`
- `label='Person'`
- `placeholder='Search people…'` (shown only when `multiple` or nothing is selected)
- `size='small'`, `disabled`, `fullWidth=true`, `helperText`

**Value contract:**
- Single: `value` is an id, a name string or a person; `onChange(person|null)`.
- Multiple: an array of ids or names; `onChange(person[])`.
- Parents store both the name and the id, e.g. `owner` + `ownerId`, `approver` + `approverId`, `requester` + `requesterId`. The id targets pushed notifications; the name remains for display and back-compat.

**Data:**
- A module-level `CACHE` is filled once per page load from `GET /org/people` (no query, so it is capped at 50 by the server).
- It is never invalidated; new users appear only after a reload.
- A spinner shows in the end adornment while loading.

**Resolving a stored value:**
- falsy → null
- an object → used as is
- `'unassigned'` (case-insensitive) → null
- otherwise: match by id, then by case-insensitive name, otherwise a placeholder `{id:'', name, __unknown:true}`. The placeholder is added to the options so legacy free-text names stay visible.
- Equality: by id when both have ids, otherwise by name.

**Filtering and rendering:**
- Client-side substring filter on name, email and title. Options keep the server ranking order.
- `openOnFocus`.
- No-match text: "No matching people".
- Option: Avatar 28, name, and `title || email`.
- Chips: Avatar 18 and name.
- Single selection: Avatar 22 in the start adornment.

**Used for:** owner fields (tactic, node, plan, line item), approver fields (plan submit, tactic submit, bulk submit — "Approver (optional)" with helper "Leave empty to let any team lead or manager approve."), PR requester and approver ("Who signs this off?"), and Admin team lead ("Search for a team lead…") and members ("Add people…").

#### 6.8.7 `TeamDashboard.jsx` (home for org roles manager, lead and member)

**Loading:** fetches `GET /org/dashboard` whenever `user.id` changes. `StateWrap` shows a spinner (min height 240) or an error: "Could not load data: {msg}. Is the API running on port 4000?".

**PageHeader:**
- eyebrow from `ROLE_TITLE[data.role]`:
  - manager: "Manager dashboard"
  - lead: "Team-lead dashboard"
  - member: "My team"
  - admin: "Organization overview"
  - otherwise "Dashboard"
- title "Welcome back, " with highlight `` `${firstName||'there'}.` ``
- subtitle "Your teams, their people, and which campaigns they're driving — live."

**Manager view:**
- 4 StatCards (2 columns, 4 at lg+):
  - "Leads reporting to you" (`#9f4bd6`)
  - "Teams" (`#0071e3`)
  - "People" (`#1baf7a`)
  - "Campaigns in play" (`#eb6834`)
- Panel "Which team is working on which campaign".
  - InfoTip: "Every campaign your teams touch, the teams on it, and live status rolled up from their tactics."
  - Subtitle: "Live rollup across all your teams"
  - Body: CampaignMatrix
- Section "Your leads & their teams": for each lead, Avatar 28, name and `` `${title||'Lead'} · N team(s)` ``, followed by a TeamCard grid (1, 2 or 3 columns).

**Lead view:**
- StatCards: "Your teams", "People", "Campaigns", and a "Reports to" card showing the manager's name or "—".
- Panel "Your teams & campaigns".
  - InfoTip: "Campaigns your teams are driving, with live status from their tactics."
  - Subtitle: "Live rollup"
  - Body: CampaignMatrix
- TeamCards.

**Member view:**
- No team → "You're not assigned to a team yet. An admin can add you to one from the Admin panel."
- Otherwise:
  - Hero card: color dot, team name (20 px bold), focus, lead Avatar with "Lead" and the lead's name.
  - "Teammates": pills with Avatar 26, name and `title||'Member'`. The list includes the viewer.
  - Panel "What your team is working on".
    - InfoTip: "Campaigns your team is driving, with live status from the underlying tactics."
    - Subtitle: "Live rollup"

**Admin view:**
- StatCards: Managers, Leads, Teams, Members.
- Panel "Which team is working on which campaign", subtitle "Live rollup across the whole organization".
- TeamCards.
- This view is effectively unreachable, because `Dashboard.jsx` only renders TeamDashboard for manager, lead or member.

**CampaignMatrix:**
- Empty → "No campaigns assigned yet. An admin can assign campaigns to teams from the Admin panel."
- Otherwise a table (min width 640) with columns:
  - Campaign
  - Team(s) on it
  - Tactics — InfoTip "Number of individual pieces of work (tactics) running under this campaign."
  - In flight — InfoTip "Tactics currently being worked on (status 'in progress')."
  - Budget — InfoTip "Total planned budget committed across this campaign's tactics."
  - Progress — InfoTip "Average completion across the campaign's tactics."
- Cell rendering:
  - team pills use the team color text on a `color+'1c'` background with a dot
  - in-flight is blue when greater than 0
  - budget uses `fmtMoney` (currency-aware)
  - progress is a 7 px gradient bar (`#2a78d6→#9f4bd6`) plus "N%"

**TeamCard:**
- color dot, name, focus
- "N members" pill
- "Led by **name**", or "No lead assigned"
- campaign tags
- footer: "N tactics", "N in flight" (blue), "N launched" (`#0ca30c`), and the planned budget on the right

#### 6.8.8 `Admin.jsx` (`/admin`)

**Gate:** `canManage` = workspace role Owner or Admin. Otherwise:
- header: eyebrow "Admin", title "Admin" with highlight "panel.", subtitle "Manage the organization, teams and people."
- Panel "Admins only": "You need the **Admin** or **Owner** role to manage the organization. Ask a workspace owner to grant access, or open your profile to see your current role."

**Admin header:** eyebrow "Admin", title "Run your" with highlight "organization.", subtitle "Build the manager → lead → team → member hierarchy, and assign teams to campaigns."

**Tabs:** a pill ToggleButtonGroup — "Org chart" (default), "Teams", "People", "Demo access".

**Loading:**
- Initial: `Promise.all` of tree, teams, users, campaigns and personas. Any single failure puts the whole page in the error state.
- `refresh()` after a mutation refetches tree, teams, users (an error there → null) and campaigns, with no loading indicator.
- Toasts: Snackbar bottom-center, 3000 ms, filled Alert.

**Org chart tab:**
- 4 count cards: Managers, Leads, Teams, Members.
- For each admin: PersonRow (Avatar 30, name, RoleTag, sub-label "Workspace admin"), then a left-bordered list of ManagerBranch panels for managers whose `managerId` is that admin.
- ManagerBranch: title is Avatar 24 + name + "MANAGER" tag; subtitle is the title. For each lead, a bordered box with PersonRow (sub = title) and a grid of team tiles (color dot, name, "N members", member avatars of 22 px).
- Empty texts: "No teams yet." and "No leads yet.".
- Orphan managers are rendered after the admins.
- RoleTag colors: admin `#e34980`, manager `#9f4bd6`, lead `#0071e3`, member `#1baf7a`. Uppercase, 10 px, on a `1c`-alpha background.

**Teams tab:**
- "N teams" and a "New team" button (Add icon).
- Card grid (1 column, 2 at lg):
  - dot, name, focus
  - Edit and Delete icon buttons
  - "Led by **name**" or "No lead assigned"
  - "MEMBERS (N)" with chips (Avatar 20 + first name)
  - "CAMPAIGNS:" chips, or "none"
  - footer: "N tactics", "N in flight", budget
- **Editor dialog** (maxWidth sm, title "New team" or "Edit team"):
  - Team name
  - Focus
  - Lead (single PeoplePicker)
  - Color (Select over the 6 swatches, showing the hex value)
  - Members (multi PeoplePicker)
  - Campaigns (multi-select with checkboxes; the value is the names joined)
  - "Cancel" and "Create"/"Save" (disabled while the name is blank)
  - The default color for a new team is `TEAM_COLORS[teams.length % 6]`. Editing preloads only campaigns that still resolve.
  - Toasts: "Team created." / "Team updated." / the error message.
- **Delete dialog:**
  - title `Delete “name”?`
  - body "This removes the team and detaches its N member(s) (their accounts stay). This cannot be undone."
  - "Cancel" / "Delete" (error color)
  - toast "Team deleted."

**People tab:**
- Search field "Search people…" (260 px). It filters on name + email, case-insensitive, client-side. Next to it, "N people".
- Table (min width 820) with columns:
  - **Person:** Avatar 28, name, email.
  - **Org role:** Select with *None* plus Admin/Manager/Lead/Member. Sends `{orgRole}`; toast "Org role updated.".
  - **Reports to / Team:**
    - for leads, a manager Select (*No manager* plus all org-role managers); toast "Manager updated.";
    - for members, a team Select (*No team* plus teams); toast "Team updated.";
    - otherwise "—".
  - **Title:** inline TextField (placeholder "Job title"). Saves on blur when changed; Enter blurs. Toast "Title updated.".
- The workspace permission role is not edited here; that lives in ProfileDialog → Team & roles. A manager's own manager (the admin link) cannot be edited in the UI.

**Demo access tab:**
- Panel "Demo personas", subtitle "Sign in as any seeded persona to demo the role-aware experience".
- Password box: "Shared password for all demo accounts:", the code, and a copy icon button (clipboard, toast "Password copied.").
- Groups headed "MANAGERS", "LEADS", "MEMBERS". The admin persona is grouped but never rendered.
- Each card: Avatar 34, name, `title · team`, and an outlined "Sign in" button (Login icon; shows "…" while busy).
- Signing in logs in as that persona and navigates to `/`. The previous token is not revoked on the server.

#### 6.8.9 `AuditLog.jsx` (`/audit`)

**PageHeader:**
- eyebrow "Audit log", title "Every change," with highlight "on the record."
- subtitle "A tamper-evident trail of who did what, and when — across planning, budgets, procurement, approvals and admin. You see what your role allows."
- `currency={false}`
- Note: the copy claims "tamper-evident", but nothing implements it — no hash chaining.

**Loading:** one `GET /audit` on mount with no filters, so the server's 200-event default applies. StateWrap handles loading and error.

**Scope banner:**
- Icon by scope key: all → Public, org/team → Groups, self → Person, fallback Shield.
- Heading: "Workspace-wide visibility" when `canSeeAll`, otherwise `` `Scoped view · ${label}` ``.
- Body: the scope description.

**Stat cards** (2 columns, 4 at sm):
- "Total events" (History icon, `SERIES[0]` `#2a78d6`)
- "Today" (Shield icon, `SERIES[1]` `#1baf7a`)
- "People active" (Groups icon, `SERIES[4]` `#4a3aa7`)
- "Deletions & critical" (RestartAlt icon, `#d03b3b`)

**Filters** (client-side only, over the 200 events returned):
- Search "Search actions or people…", with a search icon.
- "Category" select: "All categories", then `` `${label} (${count})` ``.
- "Severity" select: "All severity", then Routine (info, `#8a8a8f`), Workflow (notice, `#2a78d6`), Deletion (warning, `#ec835a`), Critical (critical, `#d03b3b`).
- "Person" select, shown only when there is more than one actor: "Everyone", then `` `${name} (${count})` ``.
- "Clear" button (RestartAlt icon) while any filter is active.

**Empty state:** History icon (40 px).
- With filters: "No events match these filters" / "Try clearing a filter to widen the view."
- Without: "No activity yet" / "As people create, edit, approve and delete things, every action will appear here."

**List:**
- Grouped by day: "Today", "Yesterday", otherwise `toLocaleDateString({weekday:'long', month:'short', day:'numeric'})`.
- Each group header: the label, a hairline, and "N event(s)".
- Each row:
  - a severity dot with a tooltip giving the severity label
  - Avatar 30 built from the actor snapshot
  - **actorName** followed by the summary
  - a category pill (label from `data.categories`, falling back to the key)
  - optional italic `“meta.note”` (set only by plan reject)
  - time via `toLocaleTimeString({hour:'numeric', minute:'2-digit'})`
- Footer "Showing the most recent 200 events." when `filtered.length >= 200`.
- No pagination, no export, no detail drawer.

#### 6.8.10 `RecycleBin.jsx` (`/trash`)

**PageHeader:**
- eyebrow "Recycle bin", title "Recover" with highlight "what was deleted."
- subtitle "Deleted plans, campaigns, programs, tactics, line items, dashboards and purchase requests land here — restore them in one click (restored tactics come back as Created and need re-approval), or clear them for good."

**Top bar:**
- Text: "The recycle bin is empty." or "**N** item(s) in the recycle bin".
- When N > 0, an outlined error button "Empty recycle bin" (DeleteSweep icon).

**Empty state:** RestoreFromTrash icon, "Nothing to recover", "When you delete a plan, campaign, tactic, dashboard or purchase request, it lands here so you can bring it back."

**Rows** (newest first, as stored; no search, filter, sort or pagination):
- Icon tile by type (`TYPE_META`):
  - node: AccountTree, `SERIES[0]`, "Plan hierarchy"
  - tactic / tactics: ViewKanban, `SERIES[4]`, "Tactic" / "Tactics"
  - lineItem: ReceiptLong, `SERIES[1]`, "Line item"
  - dashboard: SpaceDashboard, `SERIES[2]` `#e87ba4`
  - request: ReceiptLong, `SERIES[3]` `#eda100`, "Purchase request"
  - fallback: RestoreFromTrash, muted, "Item"
- Label (bold), then `` `${sublabel||noun} · deleted ${timeAgo}${by ? ' by '+by : ''}` ``.
- "Restore" button (contained) and a purge icon button (outlined error, DeleteForever, tooltip "Delete permanently").
- Single purge has **no confirmation**.
- While a row is busy, both of its buttons are disabled.

**`timeAgo`:**
- under 60 s → "just now"
- under 60 min → "N min ago"
- under 24 h → "N hr ago"
- under 30 days → "N day(s) ago"
- otherwise `toLocaleDateString()`
- invalid → `''`

**Confirm dialog** for emptying: "Empty the recycle bin?" / "This permanently deletes all N item(s) in the recycle bin. This cannot be undone." / "Cancel" / "Empty recycle bin" (contained error).

**Toasts** (bottom-center, 3000 ms): `Restored “label”.`, `Permanently deleted “label”.`, "Recycle bin emptied.", or the server error, e.g. the parent-missing message.

### 6.9 Endpoint summary for this area

**Auth:**
- `POST /api/auth/signup`: public → `{token, user}`
- `POST /api/auth/login`: public → `{token, user}`
- `GET /api/auth/me`: public → user or null
- `POST /api/auth/logout`: public
- `PUT /api/auth/profile`: wall
- `GET /api/auth/users`: Admin
- `PUT /api/auth/users/:id/role {role}`: Admin

**Org:**
- `GET /api/org/tree│me│dashboard│campaigns│teams│teams/:id` (404 `'Team not found.'`)
- `GET /api/org/people?q&limit`
- `GET /api/org/personas`: public
- `POST /api/org/teams`: Admin
- `PUT│DELETE /api/org/teams/:id`: Admin
- `PUT /api/org/teams/:id/lead {leadId}`, `PUT /api/org/teams/:id/members {memberIds}`, `PUT /api/org/teams/:id/campaigns {campaignIds}`: Admin
- `GET /api/org/users`: Admin
- `PUT /api/org/users/:id {orgRole, managerId, teamId, title}`: Admin

**Notifications:** `GET /api/notifications`, `POST /api/notifications/read`.

**Audit:** `GET /api/audit?category&severity&actorId&q&limit`.

**Trash (Editor):** `GET /api/trash`, `POST /api/trash/:id/restore`, `DELETE /api/trash/:id`, `POST /api/trash/empty`.

**Response envelope:** `{ok:true, data}`, or `{ok:false, error}`.
- Auth and org errors → 400.
- Wall → 401.
- Role → 403.

**Client helpers that exist but the UI never calls:** `orgMe`, `orgTeam`, `orgSetTeamLead`, `orgSetTeamMembers`, `orgSetTeamCampaigns`.

### 6.10 Gaps and edge cases to decide on in the rebuild

1. Session tokens use `Math.random`, never expire, have no cap, and are not revoked on a password change.
2. The demo password is exposed by a public endpoint and the UI. The seeded Admin persona can therefore be accessed by anyone.
3. On a fresh install, seeding runs before any signup, so no one becomes Owner.
4. GET errors drop the server's message. There is no global 401 handling, and the client user is never refreshed after a role or org change.
5. Notifications match on display **names** (approver, owner, first-name mentions; the mention regex is not escaped). Overdue items have no age limit. Every poll writes to disk.
6. Twelve emitted audit actions are unregistered and fall back to "other / updated item". "Tamper-evident" is only copy. The audit page searches only the latest 200 events client-side.
7. `setUserOrg` does not remove a user from their previous team. Removing a member from a team leaves a stale `teamId`. Healing re-adds campaigns that an admin deliberately cleared from a seeded team.
8. The recycle bin has no retention, no per-user scope, and no confirmation for a single purge. Tactic restores skip the parent check. An "already exists" skip still deletes the entry. Teams are hard-deleted and never reach the bin.
9. Builder dashboard routes have no role gate on the server.
10. `dashboardFor` returns the org-wide payload to any user without an org role. Org read endpoints are open to every signed-in user.
11. The People picker gets at most 50 people and caches them for the whole page session.

## 7. Planning — domain model and backend

**Source files:** `server/services/plans.js`, `planning.js`, `models.js`, `campaignPlans.js`, `boardConfig.js`, `boardImport.js`, `boardConnections.js`, `importer.js`, `spreadsheet.js`, `calendar.js`, `googleCalendar.js`, `planLoop.js`, `docs/Trello-Jira-Import.md`.

The hierarchy is Hive9's five-level chain, created strictly in order: **Plan → Campaign → Program → Tactic → Line item**. Places where the code disagrees with its own comments or docs are flagged **QUIRK**.

### 7.0 Cross-cutting conventions

#### 7.0.1 Storage (`store.js`)
- All persistence is synchronous JSON files in `DATA_DIR` = `process.env.ZR_DATA_DIR` or `server/data/`. The directory is created if missing.
- `readJson(file, fallback)` returns `fallback` if the file is missing, empty or whitespace, or if anything throws (including a JSON parse error).
- `writeJson(file, data)` writes `JSON.stringify(data, null, 2)` and returns `data`. It is not atomic and has no locking. Every mutation is a whole-file read-modify-write.
- `uid(prefix='id')` returns `` `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2,7)}` ``, e.g. `tac-mfx3k2a1-9qz0c`.
- `nowIso()` returns `new Date().toISOString()`.

#### 7.0.2 Persistence files used by planning

| File | Owner module | Shape |
|---|---|---|
| `plans.json` | plans.js | flat array of container nodes (plan/campaign/program) |
| `tactics.json` | planning.js | array of tactics (line items embedded) |
| `initiatives.json` | planning.js (legacy, read-only migration source) | array of old board items |
| `models.json` | models.js | array of planning models |
| `boardConfig.json` | boardConfig.js | `{ columns: [...], fields: [...] }` |
| `boardConnections.json` | boardConnections.js | array of Trello/Jira connection records |
| `planLoops.json` | planLoop.js | array of AI plan-loop records |
| `settings.json` | googleCalendar.js (key `googleCalendar`) | shared settings object |

#### 7.0.3 HTTP envelope, errors, auth
- Success: `{ ok: true, data }`. Error: `{ ok: false, error: <message> }`.
- `ValidationError` (from plans.js) carries `status = 400`. The route helper `fail(res, err)` uses `err.status || 500` and logs anything ≥ 500. Message fallback: `'Something went wrong.'`
- **Global auth wall:** every `/api/*` route returns 401 `'Please sign in to continue.'` unless the bearer token resolves to a user. Exceptions: `POST /api/auth/login`, `POST /api/auth/signup`, `GET /api/auth/me`, `POST /api/auth/logout`, `GET /api/org/personas`, `GET /api/currency`, `GET /api/health`, and **`GET /api/planning/calendar.ics`** (intentionally public).
- `requireRole(min)`: passes when there are zero users or no token (sets `req.user = null`). An invalid token gets 401 `'Your session has expired — please sign in again.'`. Too low a rank gets 403 `` `This action requires the ${min} role or higher.` ``.
- `requirePlanner` (board customization): the user must have `orgRole` in `['lead','manager']` or role rank ≥ Admin. Otherwise 403 `'Board customization is available to team leads and managers.'`
- `approvalGuard(approval, user)` (campaign plans, and tactics that have a named approver):
  - No `approverId` and no `approver` name: allowed.
  - Otherwise, if `approverId` is set it must equal `user.id`; else `approver` must equal `user.name`.
  - Error (403): `` `Only ${apName || 'the assigned approver'} can approve or reject this plan.` `` (the word "plan" is used even for tactics).
- `tacticApprovalGuard(t, user)`:
  - If there is no user: allowed.
  - If the tactic has an approver: use `approvalGuard`.
  - Otherwise the user must have role ≥ Admin or `orgRole` in `['lead','manager','admin']`. Else 403 `'You don’t have approval rights for tactics — ask a team lead or manager.'`
- **Side effects on routes.** Audit records and notifications are fire-and-forget; failures are swallowed with `console.warn`. `notifyAssignment` never notifies the actor themselves and targets by `toUserId` only.

### 7.1 The hierarchy

```
Plan → Campaign → Program → Tactic → Line item
(one FY)  (theme)   (workstream)  (the work)  (where money lives)
```

- Containers (plan, campaign, program) live in one flat, parent-linked list in `plans.json`. Tactics are separate (`tactics.json`) and reference `planId`, `campaignId`, `programId`. Line items are embedded in `tactic.lineItems`.
- `LEVELS` (exported):
  ```
  [{id:'plan', name:'Plan', parentLevel:null, hint:'Annual / strategic container'},
   {id:'campaign', name:'Campaign', parentLevel:'plan', hint:'A grouping of related programs'},
   {id:'program', name:'Program', parentLevel:'campaign', hint:'A workstream that holds tactics'}]
  ```
- `LEAF_LEVEL = 'tactic'`.
- `levelMeta(id)` returns the matching level or null.
- `childLevel(id)`: `null` → `'plan'`; `plan` → `'campaign'`; `campaign` → `'program'`; `program` → `'tactic'`; unknown → `null`.
- `GET /api/plans/meta` also adds `{id:'tactic', name:'Tactic', parentLevel:'program'}` and `{id:'lineItem', name:'Line item', parentLevel:'tactic'}`.

### 7.2 Container nodes (plans.js)

#### 7.2.1 Fields and normalization: `clean(input, existing)`
`merged = {...existing, ...input}`; the output is built from `merged`.

| Field | Type | Rule / default |
|---|---|---|
| `id` | string | `uid(input.level || 'node')` on create. **QUIRK:** uses the raw input level, so an invalid level `'foo'` gets an id `foo-…` but is stored as `program`. Ids are preserved on update. |
| `createdAt` / `updatedAt` | ISO | Set on create; `updatedAt` is refreshed on every update. |
| `level` | `'plan' \| 'campaign' \| 'program'` | Anything else becomes **`'program'`**. |
| `parentId` | string \| null | Always `null` for a plan; otherwise `merged.parentId || null`. |
| `name` | string ≤140 | Trimmed; default `'Untitled'`. |
| `owner` | string ≤80 | Default `'Unassigned'` (not trimmed). |
| `ownerId` | string ≤60 | Stable user id from the people picker. If `merged.ownerId == null` it is `existing.ownerId || ''`. |
| `description` | string ≤800 | Default `''`. |
| `startDate` / `endDate` | `'YYYY-MM-DD'` | `String(v||'').slice(0,10)`. |
| `dateMode` | `'auto' \| 'manual'` | Plans are always `'manual'`. Otherwise the explicit value if valid; else `'manual'` if any date is present, else `'auto'`. |
| `fiscalYear` | string (4 digits) | Plans only: the first `\d{4}` in the value (`"FY2026"` → `"2026"`). `''` for campaigns and programs. |
| `currency` | string ≤3 | Uppercased; default `'USD'`. Present on every level. |
| `planBudget` | integer ≥0 | `max(0, round(Number||0))`. The Hive9 marketing "Plan Budget". |
| `status` | `'Draft' \| 'Submitted' \| 'Published' \| ''` | Kept if valid, else `existing.status || ''`. Used by campaigns (plan approval). |
| `approval` | object \| null | Any object is accepted verbatim, else `existing.approval || null`. |
| `attrs` | object | Object else `{}`. |
| `comments` | array | `commentsOf`: keeps the **last 200** (`slice(-200)`). Each is `{id: c.id||uid('cm'), author ≤80 default 'Someone', text ≤1000, at: c.at||now}`. |

Plan-only fields (added only when `level==='plan'`):

| Field | Rule |
|---|---|
| `modelId` | ≤60; default `defaultModelId()` (= first model, `'model-standard'`). |
| `teams` | Array of team ids, each `String().slice(0,60)`, de-duplicated, max 20; else `[]`. |
| `isStrategic` | Boolean. |
| `goals` | `goalsOf`: keys `inquiries, mqls, tqls, opportunities, won, revenue`. Values are `Math.round(Number)` and kept only if finite and > 0. Others are dropped, so the result may be `{}`. |
| `defaultMechanism` | One of `C/F/L/E`; default `'E'`. Stored only; the client uses it as the default line-item spread. |

- `updateNode(id, patch)` stores `{...list[i], ...clean(patch, list[i]), id, createdAt, updatedAt: now}`. Unknown legacy fields on the old record survive because of the leading spread.
- `createNode(input)` does **no validation**. Validation happens only in routes.

#### 7.2.2 `validateNode(input, existing=null, opts={})`
This is the exact order. It throws `ValidationError` (400) on the first failure and returns the cleaned record on success.

- `next = clean(input, existing||{})`.
- `touched(...keys)` is `!existing || keys.some(k => k in input)` (own property). So on create every rule runs.
- `word` = `'plan' | 'campaign' | 'program'`.

1. **No level change:** `existing && input.level && input.level !== existing.level` → `'A record cannot change level.'`
2. **Name required:** `String(input.name ?? existing?.name ?? '').trim()` empty → `` `Give the ${word} a name.` ``
3. **Parent ordering:** `expectedParent = {plan:null, campaign:'plan', program:'campaign'}`. If an expected parent exists and `touched('parentId','level')`:
   - parent missing → `` `A ${word} must sit inside a ${expectedParent} — create the ${expectedParent} first.` ``
   - wrong parent level → `` `A ${word} can only be created under a ${expectedParent}, not a ${parent.level}.` ``
4. **Name uniqueness**, when `touched('name','parentId')`. Names are compared case-insensitively after trimming. A clash is another node (different id) with the same level and same name, where either the level is plan, or the parentId is the same.
   - Plan → `` `A plan called “${next.name}” already exists — plan names must be unique.` `` (plan names are unique system-wide)
   - Child → `` `“${parent?.name || 'This parent'}” already has a ${word} called “${next.name}”.` `` (unique under the parent)
5. **Fiscal-year window** (plan and `touched('fiscalYear')`): `fy = Number(next.fiscalYear)`, `now = new Date().getFullYear()`.
   - `!fy` → `'Choose the fiscal year this plan covers.'`
   - `fy < now || fy > now+10` → `` `A plan's fiscal year must be between ${now} and ${now + 10}.` ``
   - **QUIRK:** once the calendar year passes the plan's FY, any PUT that includes `fiscalYear` (even unchanged) fails.
6. **Dates**, when `touched('startDate','endDate','dateMode','parentId')`:
   - `start && end && start > end` → `'The start date must be on or before the end date.'`
   - Non-plan, `manual`, and a date missing → `` `Enter both a start and an end date, or let the ${word} calculate its dates from its tactics.` ``
   - **Program containment:** applies when the program is manual, its parent campaign is manual, and the campaign has both dates. If `start < parent.start` or `end > parent.end` → `` `Program dates must sit inside the campaign's dates (${parent.startDate} → ${parent.endDate}).` ``
   - **Campaign narrowing:** applies on update when the campaign is manual with both dates. The first child program (level program, manual, both dates) that falls outside the new window → `` `Program “${outside.name}” runs ${outside.startDate} → ${outside.endDate}, outside these campaign dates.` ``
   - Not checked: plan dates against the FY, campaign dates against the plan, tactic dates against the program or campaign.
7. **Budget containment**, when `touched('planBudget','parentId')`:
   - Program: if parent `planBudget > 0` and `next.planBudget > parent.planBudget` → `` `A program's budget can't exceed its campaign's (${Number(parent.planBudget).toLocaleString('en-US')}).` ``
   - Campaign (update only, `next.planBudget > 0`): any child program with a bigger planBudget → `` `Program “${over.name}” already has a bigger budget (${Number(over.planBudget).toLocaleString('en-US')}) than this.` ``
   - A campaign exceeding its plan is not blocked; `tree()` flags it instead. The sum of children is never checked.
8. **Team membership.** Runs for a plan when `opts.teamsById` is given, `teams.length > 0`, `ownerId` is set, and `touched('teams','ownerId','owner')`:
   - Any team id not in `teamsById` → `'One of the selected teams no longer exists.'`
   - The owner is neither `leadId` nor in `memberIds` of any selected team → `'The plan owner must belong to one of the plan’s teams.'`
   - Routes build `teamsById` from `listTeams()` as `{id: {memberIds, leadId}}`.

#### 7.2.3 Other node functions
- `listNodes()` seeds if the file is not an array (see §7.6), then runs the migration (§7.6.3).
- `getNode(id)`.
- `defaultProgramId()` returns the first program in file order, or null. This is the import/orphan target.
- `lineageOf(nodeId)` walks up `parentId`, setting `out[`${level}Id`]` at each step. It returns `{planId, campaignId, programId}`; a missing node gives all nulls.
- `deleteNode(id)` hard-deletes the node and all descendant containers (fixed-point closure over `parentId`) and returns `{removedIds}`.
- `insertNodes(nodes)` re-inserts verbatim for recycle-bin restore. Existing ids are skipped. Returns `{added}`.
- **Delete route** `DELETE /api/plans/nodes/:id` (Admin) calls `recycleBin.trashNode`:
  - It computes the descendant set, collects tactics whose program, campaign or plan id is in that set, then calls `deleteNode` and `deleteTactics({ids})`.
  - It writes one recycle record `{type:'node', label: root name, sublabel: "<Level> · N items · M tactics", payload:{nodes, tactics}}`.
  - It responds with `{removedIds, removedTactics}`.
  - `planning.orphanTactics` (re-home to the default program) exists but is not used by routes.

#### 7.2.4 `tree(tacticsByProgram)`: roll-ups and auto dates
Input is a map `programId → decorated tactics` (from `planning.tacticsByProgram()`, which skips tactics without `programId`).

Helpers:
- `num(v)` = finite `Number(v)` else 0.
- `minDate` / `maxDate` compare ISO strings and ignore empty strings.

`tacticMoney(t)`:
```
lineItems = Σ li.amount
liActual  = Σ li.actual
{ plannedCost: num(t.budget),
  lineItems,
  actual: liActual > 0 ? liActual : num(t.spendActual),
  balance: num(t.budget) − lineItems }      // "Sys_Gen_Balance"
```

`build(node)`:
- **Program:**
  - `node.tactics = map[id] || []`.
  - Money is the sum of `tacticMoney`; `tacticCount += 1`; `lineItemCount += lineItems.length`.
  - `derivedStart = min(t.startDate || t.endDate || t.dueDate || '')`.
  - `derivedEnd = max(t.evergreen ? '' : (t.endDate || t.dueDate || t.startDate || ''))`.
- **Plan or campaign:**
  - `node.children` = nodes with `parentId === id`, built recursively.
  - Money and counts are the sum of the children's `rollup`.
  - `derivedStart` = min of the children's `effectiveStart`; `derivedEnd` = max of their `effectiveEnd`.
- **Effective dates:**
  - Plan: `startDate || FY start` and `endDate || FY end`. `fiscalWindow(plan)` = `{start:'YYYY-01-01', end:'YYYY-12-31'}` from the first 4-digit run of `fiscalYear`, else empty strings.
  - Manual node: the typed dates.
  - Auto node: the derived dates ("shrink dates to children").
- **`node.rollup`:**
  ```
  { money: {plannedCost, lineItems, actual, balance},
    tacticCount, lineItemCount,
    planBudget: num(n.planBudget),
    childBudget: Σ children.planBudget,          // 0 for programs (no children array)
    unallocatedBudget: planBudget − childBudget }
  ```
- **Roots** are nodes with `parentId === null` or a dangling parent, filtered to `level === 'plan'`. So a campaign with a missing parent is not in the tree, although the campaign-plan register shows it.
- **Over-parent flag:** walking down, each child gets `c.rollup.overParent = parent.planBudget > 0 && c.planBudget > parent.planBudget`. It compares individual values, not the sum. Plans themselves never get the flag.

`GET /api/plans/tree` → `tree(tacticsByProgram())`.

### 7.3 Planning models (models.js)

The "rulebook a plan is measured by". Every plan has a `modelId`.

- `STAGES` (waterfall order): `inquiries` (Inquiries), `mqls` (MQLs), `tqls` (TQLs), `opportunities` (Opportunities), `won` (Closed won).
- `rate[stage]` is the share converting **into** that stage from the stage above; inquiries has no rate.
- `TACTIC_TYPES = ['Campaign','Launch','Content','Event','Research','Ops']` (same list as `planning.TYPES`).

**Default model** (seeded when `models.json` is missing or an empty array):
```
id: 'model-standard', name: 'ZenReach Standard B2B',
description: 'Default waterfall: inquiries → MQL → TQL → opportunity → closed won.',
rates: { mqls: 0.25, tqls: 0.4, opportunities: 0.5, won: 0.3 },
velocityDays: { mqls: 7, tqls: 14, opportunities: 21, won: 45 },   // informational only
averageDealSize: 25000,
tacticTypes: all 6,
requirePlannedCost: true,
blockNegativeBalance: false,
goalHorizonMonths: 3,                                              // informational only
createdAt, updatedAt
```

**Model fields and their effects:**
- `requirePlannedCost`:
  - Makes "planned cost" a submit requirement.
  - Changing the budget of a Submitted or Approved tactic resets it to Created.
- `blockNegativeBalance`: line items may not sum above the tactic's planned cost.
- `tacticTypes`: the types a plan's tactics may use.
- `velocityDays` and `goalHorizonMonths` are stored but no server logic reads them.

**Normalization `clean(input, existing = defaultModel())`.** `m = {...existing, ...input}`:
- `rates[s]` = `clampRate(m.rates?.[s], existing.rates?.[s] ?? 0)`, i.e. a finite number clamped to [0,1], else the existing value.
- `velocityDays[s]` = `max(0, round(Number||0))`. **QUIRK:** no per-stage fallback, so a partial `velocityDays` object zeroes the missing stages.
- `name` ≤80 (default `'Untitled model'`); `description` ≤300.
- `averageDealSize` = `max(0, round)`.
- `tacticTypes` is filtered to `TACTIC_TYPES`; an empty result means all 6.
- `requirePlannedCost` and `blockNegativeBalance` are Booleans; `goalHorizonMonths` is clamped to 0–24 and rounded.
- New models inherit default values for any omitted field, because `existing` defaults to `defaultModel()`.

**CRUD:**
- `listModels()`.
- `getModel(id)` returns the match **or `list[0]`**, so it never returns null.
- `defaultModelId()` is the first model's id.
- `createModel(input)`:
  - Trimmed name required: `'A model needs a name.'`
  - Unique case-insensitively: `` `A model called “${name}” already exists.` ``
  - id `uid('model')`.
- `updateModel(id, patch)`: null if not found. If `patch.name` is given, it is checked for uniqueness against other models with the same message.
- Routes: `GET /api/plans/models`; `POST /api/plans/models` (Admin, errors → 400); `PUT /api/plans/models/:id` (Admin, 404 `'Model not found'`). Audits `model.create` / `model.update`.

**Waterfall `waterfall(model, inquiries)`:**
```
out.inquiries = max(0, round(inquiries))
prev = out.inquiries
for stage in [mqls, tqls, opportunities, won]:
    prev = round(prev × (rates[stage] ?? 0)); out[stage] = prev
out.revenue = round(out.won × averageDealSize)
```
Example with the default model: 1000 → 250 MQL → 100 TQL → 50 opp → 15 won → **$375,000**. Rounding happens at every stage.

### 7.4 Tactics (planning.js)

#### 7.4.1 Constants
- `TYPES = ['Campaign','Launch','Content','Event','Research','Ops']`; `PRIORITIES = ['P0','P1','P2']`.
- `APPROVAL_STATUSES = ['Created','Submitted','Declined','Approved']`.
- `MECHANISMS = ['C','F','L','E']`; `SEGMENTS = ['month','quarter']`.
- `MONTH_KEY = /^\d{4}-(0[1-9]|1[0-2])$/`; `QUARTER_KEY = /^\d{4}-Q[1-4]$/`.
- `COLUMNS` (legacy 4 columns with hints) is exported but unused; the board uses boardConfig.
- `statusOrDefault(status)` returns the status if it is a configured board column id, else the first column id (or `'backlog'`).

#### 7.4.2 Tactic record: `normalize(input, existing={}, forceId, opts={})`
- `merged = {...existing, ...input}`.
- `trusted = opts.trusted`. Only internal callers set it; it lets approval fields through.
- `evergreen = existing.evergreen || merged.evergreen`, which makes it one-way.

| Field | Type / limit | Rule |
|---|---|---|
| `id` | string | `forceId || existing.id || uid('tac')` |
| `title` | ≤140 | Trimmed; default `'Untitled tactic'` |
| `externalName` | ≤140 | Name used when pushed to a connected system; default `''` |
| `type` | enum TYPES | Default `'Campaign'` |
| `owner` | ≤80 | Default `'Unassigned'` |
| `ownerId` | ≤60 | Same rule as nodes (`== null` → `existing.ownerId || ''`) |
| `quarter` | ≤20 | `quarterOfDate(startDate)` (`"Qn YYYY"` from the start month) when dated; else `merged.quarter || 'Q3 2026'`. A dated tactic's typed quarter is overwritten. |
| `priority` | enum | Default `'P1'` |
| `status` | board column id | `statusOrDefault` |
| `progress` | int 0–100 | Rounded and clamped |
| `budget` | int ≥0 | **Planned cost** (Hive9: tactics have planned cost, never a budget) |
| `channel` | ≤60 | |
| `goal` | ≤200 | Free-text goal |
| `impact` / `effort` | int 1–5 | Default 3 |
| `tags` | ≤8 strings, each ≤24 | Not de-duplicated |
| `dueDate` | date | `evergreen ? '' : (merged.dueDate || merged.endDate).slice(0,10)` |
| `startDate` | date | slice(0,10) |
| `endDate` | date | `evergreen ? '' : (merged.endDate || merged.dueDate).slice(0,10)` |
| `evergreen` | bool | One-way |
| `description` | ≤800 | |
| `planId` / `campaignId` / `programId` | string\|null | Re-derived by `finalize` |
| `approvalStatus` | enum | trusted ? `merged.approvalStatus` : `existing.approvalStatus`; invalid → `'Created'`. API input can never set it. |
| `approval` | object | `normalizeApproval(trusted ? merged.approval : existing.approval)` |
| `goals` | object | `normalizeGoals`, then `computeGoals` |
| `lineItems` | ≤100 | `normalizeLineItems(merged.lineItems, ctx)` |
| `mediaCodes` | ≤50 | `{id||uid('mc'), code ≤40, channel ≤40, label ≤80}` |
| `performance` | object | `{plan:{leads,pipeline,revenue}, actual:{leads,pipeline,revenue}}`; each value is a finite number else 0 (not rounded) |
| `spendCommitted` / `spendActual` | int ≥0 \| null | `null` if null, undefined or `''`; else `max(0, round)`. SP&A overrides. |
| `customAttrs` | object | Values of board custom fields keyed by field id. Not validated against the field definitions. |
| `comments` | ≤200 | **First** 200 (`slice(0,200)`; nodes keep the last 200). `{id, author ≤80 'Someone', text ≤1000, at}` |
| `source` | object\|null | `normalizeSource`: needs `provider` (≤20); `{provider, name ≤140, connectionId ≤60 \| null}` |
| `loopId` | ≤60 \| null | AI plan-loop link |
| `targets` | `{leads, revenue}` \| null | Integers ≥0; null when not an object |
| `createdAt` | ISO | `existing.createdAt || merged.createdAt || now` |
| `updatedAt` | ISO | Now, unless `opts.keepUpdatedAt && existing.updatedAt` |

- **Approval record** (`normalizeApproval`). Default when missing: `{approver:'', approverId:'', submittedBy:'', submittedAt:null, history:[]}`.
  - Fields: `approver` ≤80, `approverId` ≤60, `submittedBy` ≤80, `submittedAt`.
  - `history`: the last 100 entries, each `{at, by ≤80, action ≤40, comment ≤1000}`.
  - Actions used: `submitted`, `approved`, `declined`, `reset`, `restored` (the last comes from the recycle bin).
- **Goals.**
  - `normalizeGoals(g)` = `{override: Boolean(g.override), inquiries, mqls, tqls, opportunities, won, revenue}`, each `max(0, round(num))`.
  - `computeGoals(t)`: if `override`, keep the typed numbers. Otherwise use `{...waterfall(getModel(plan?.modelId), goals.inquiries), override:false}`, where the plan comes from `getNode(t.planId)`.
- **Lineage and finalize.**
  - `applyLineage(t)`: if `t.programId`, then `lin = lineageOf(programId)` and the plan, campaign and program ids are overwritten from `lin`.
  - **QUIRK:** if `programId` points to a campaign, `programId` becomes null. If it points to a missing node, all three become null.
  - `finalize(t) = applyLineage + computeGoals`. It runs on every create and update.
- **End-date QUIRK:** `endDate` prefers `merged.endDate` and `dueDate` prefers `merged.dueDate`. A PUT carrying only one of them leaves the other stale. Every reader uses `endDate || dueDate`, so `endDate` effectively wins. A rebuild should use a single end field.

#### 7.4.3 Lifecycle (`lifecycleOf(t, on = today UTC 'YYYY-MM-DD')`)
- Stored statuses: Created → Submitted → Approved | Declined.
- Displayed status:
  - If `approvalStatus !== 'Approved'`, show the stored status.
  - Otherwise `end = evergreen ? '' : (endDate || dueDate)`:
    - `end && end < on` → `'Complete'`
    - else `startDate && startDate <= on` → `'In-Progress'`
    - else `'Approved'`
- `defaultApprovalFor(status)`: `launched` or `in-progress` → `'Approved'`, else `'Created'`. Used by seeds and migrations only.

#### 7.4.4 `decorate(t)` (response shape; derived fields are ignored on write)
```
{...t, lifecycleStatus: lifecycleOf(t),
 money: { plannedCost: budget, lineItems: Σamount, balance: budget − Σamount,
          actual: Σli.actual > 0 ? Σli.actual : (spendActual || 0) } }
```
`GET /api/tactics/:id` also adds `missingForSubmit: [...]`.

#### 7.4.5 `validateTactic(input, existing=null)` (routes only; internal creators skip it)
`touched` has the same semantics as for nodes. `programId` is `input.programId` if present, else `existing.programId`; `program = getNode(programId)`.

1. If `touched('programId')`:
   - no program → `'A tactic must sit inside a program — pick (or create) the program first.'`
   - not a program → `` `A tactic can only be created under a program, not a ${program.level}.` ``
2. `title` (input if present, else existing) trimmed and empty → `'Give the tactic a name.'` (always checked).
3. **Unique within the program** (`touched('title','programId')`, case-insensitive trimmed) → `` `“${program.name}” already has a tactic called “${title}”.` ``
4. **Type allowed by the model.** Runs when `touched('type','programId')`, a program exists, and `input.type !== undefined`. The model is that of the plan in `lineageOf(program)`. Not allowed → `` `The “${model.name}” model doesn't allow “${input.type}” tactics.` ``
   - Omitting `type` on create skips the check, and the type then defaults to `Campaign` even if the model forbids it.
   - Moving a tactic to another plan does not re-check its type.
5. **Evergreen is one-way:** `existing.evergreen && 'evergreen' in input && !input.evergreen` → `'An evergreen tactic cannot be made non-evergreen. Close it and create a new tactic with real dates.'`
6. **Dates**, when `touched('startDate','endDate','dueDate')`. Here `evergreen = existing?.evergreen || input.evergreen`, start is input or existing, and end is `''` if evergreen else (input.endDate, else input.dueDate, else existing.endDate).
   - `start > end` → `'The start date must be on or before the end date.'`
   - `fy` = the plan's 4-digit FY, or the current year if none. `end.year > fy+10` → `` `End dates can't be more than 10 years past the plan year (${fy + 10}).` ``
7. `touched('budget') && Number(input.budget) < 0` → `'Planned cost cannot be negative.'`

#### 7.4.6 Submit requirements (`missingForSubmit(t)`)
Each failed check adds a label, in this order:
- `name`: title is blank.
- `owner`: empty or `'Unassigned'`.
- `program`: no programId.
- `start date`.
- `end date`: unless evergreen.
- `planned cost`: if the plan model has `requirePlannedCost` and `budget ≤ 0`.

These never block creation.

#### 7.4.7 Approval transitions
- **`submitTactic(id, {by, approver, approverId})`**:
  - Not found → null (404 `'Tactic not found'`).
  - Status not in {Created, Declined} → `` `This tactic is already ${status.toLowerCase()}.` `` ("already submitted" / "already approved").
  - Missing fields → `` `Fill in the required fields before submitting: ${missing.join(', ')}.` ``
  - Sets `approval = {...prev, approver, approverId, submittedBy: by, submittedAt: now, history: [...h, {at, by, action:'submitted', comment: approver ? `Sent to ${approver}` : ''}]}` and `approvalStatus = 'Submitted'` via a trusted update.
  - The route passes `by = req.user?.name || 'You'` and notifies `approverId` (type `approval`, title `'Tactic approval needed'`).
- **`decideTactic(id, {by, decision, comment})`**:
  - Status not Submitted → `'Only submitted tactics can be approved or declined.'`
  - Decision not approve/decline → `'Unknown decision.'`
  - Decline with a blank comment → `'Add a comment explaining why the tactic is declined.'`
  - Appends history `approved` / `declined` with the comment and sets status Approved / Declined (trusted).
  - Routes: `POST /api/tactics/:id/approve` and `/decline`, guarded by `tacticApprovalGuard`. They notify the tactic **owner** (`prev.ownerId`) with title `'Tactic approved'` or `'Tactic declined'`.
- **Auto reset on cost change** in `updateTactic(id, patch, opts)` (untrusted path only):
  - Applies when `next.budget !== prev.budget`, prev status is Submitted or Approved, and the plan model has `requirePlannedCost`.
  - Then `approvalStatus = 'Created'` and history gains `{action:'reset', by: opts.by, comment: `Planned cost changed (${prev} → ${next}) — re-submit for approval.`}`.
- **Bulk submit** (`POST /api/planning/tactics/bulk-submit {ids, approver, approverId}`):
  - Runs submit per id and collects `skipped: [{id, title, reason}]`.
  - Responds `{submitted: count, skipped}` and sends one notification if anything was submitted.

#### 7.4.8 Line items (fifth level)
**Mechanisms:**
- `C` (custom): the user types each period; total = their sum.
- `F` (first): everything goes in the first period.
- `L` (last): everything goes in the last period.
- `E` (even): split equally.
- Evergreen tactics: L and E behave like F.

**`periodsFor(tactic, segment='month')`:**
```
start = (startDate || endDate || dueDate).slice(0,7); if !MONTH_KEY → []
endRaw = evergreen ? '' : (endDate || dueDate || startDate).slice(0,7)
end = (MONTH_KEY(endRaw) && endRaw >= start) ? endRaw : start
iterate months start..end inclusive (guard 132 iterations ≈ 11 years)
segment 'quarter' → unique 'YYYY-Qn' (n = floor((m−1)/3)+1) preserving order
```
- Example: 2026-08-15 → 2026-11-10 gives months `2026-08…2026-11`, quarters `2026-Q3, 2026-Q4`.
- **`fallbackPeriod(tactic, segment)`** (no dates):
  - If `quarter` matches `/^Q([1-4])\s?(\d{4})$/i`: `'YYYY-Qn'`, or the first month of that quarter, `'YYYY-' + pad((n−1)*3+1)`.
  - Otherwise the current UTC month or quarter.

**`spread(amount, mechanism, periods)`:**
```
total = max(0, round(amount)); if no periods → {}
F → {first: total};  L → {last: total}
else (E): each = floor(total / n); every period = each; last += total − each·n
```
Example: 10,000 over Jul/Aug/Sep → 3333 / 3333 / 3334.

**`normalizeLineItem(li, tacticCtx, existing)`.** `m = {...existing, ...li}`; `ctx = {title, startDate, endDate, evergreen, quarter: raw merged.quarter, owner: raw merged.owner}`.
- `segment` is month or quarter (default month); `mechanism` is C/F/L/E (default `'E'`). The stored mechanism stays as requested even when evergreen forces F.
- `amount = max(0, round(num(m.amount)))`.
- **Mechanism C:**
  - Keep allocation entries whose key matches the segment regex and whose `round(v) > 0`.
  - If `sum > 0 || !amount`, then `amount = sum`.
  - Otherwise (amount given but no valid allocations), `allocations = {[firstPeriod || fallbackPeriod]: amount}`.
- **Other mechanisms:** `periods = periodsFor || [fallbackPeriod]`, `mech = evergreen && mechanism !== 'F' ? 'F' : mechanism`, `allocations = spread(amount, mech, periods)`.
  - Allocations are **re-spread on every tactic save**, so changing tactic dates re-spreads F/L/E items. Custom allocations are kept as typed, even outside the new window.
- Other fields:
  - `id` (`uid('li')`); `name` ≤120, default `` `${tactic.title||'Tactic'}_Line Item` `` with fallback `'Line item'`.
  - `category` ≤40 (default `'Media'`); `vendor` ≤80; `owner` ≤80 (default the tactic owner); `ownerId` ≤60; `description` ≤400.
  - `currency` ≤3, uppercased, default USD.
  - Legacy `period`: for month segments, the first sorted allocation key. For quarter segments, `YYYY-` plus the first month of the first quarter (`2026-Q3` → `2026-07`).
  - `actual` = `max(0, round)`.
  - `budgetPoolId` ≤60 \| null (link to a Budget-module pool; unlinked spend lands in the pool's unallocated balance). `prId` \| null (purchase request).
  - `createdAt` keeps `m.createdAt || now`.
  - `updatedAt` is preserved when `li === existing`, which is always true inside `normalizeLineItems`. It is refreshed only when the patched item carries `updatedAt: undefined`, which `updateLineItem` sets.
- `normalizeLineItems(arr, ctx)` keeps the first 100.

**`checkLineItems(t, items, changedId)`** (runs on the normalized draft):
- Duplicate name (case-insensitive trimmed) against another item of the same tactic → `` `“${t.title}” already has a line item called “${changed.name}”.` ``
- If the model has `blockNegativeBalance` and `Σ amount > budget` → `` `Line items (${sum.toLocaleString('en-US')}) can't add up to more than the tactic's planned cost (${budget.toLocaleString('en-US')}) — the “${model.name}” model blocks a negative balance.` ``
- **QUIRK:** lowering the tactic's budget below its line-item total is never blocked. The check runs only on line-item operations.

**CRUD:**
- `addLineItem(tacticId, input)`: new `uid('li')`, input `createdAt` stripped, check, then `updateTactic` (untrusted). Returns `{tactic, lineItem}`.
- `updateLineItem(tacticId, liId, patch)`: merges `{...x, ...patch, id, createdAt, updatedAt: undefined}`, then checks. If the item is missing it returns `{tactic, lineItem: null}` (404 `'Line item not found'`).
- `removeLineItem` is used by `recycleBin.trashLineItem`. The route is `DELETE /api/tactics/:id/line-items/:liId`, with 404s for a missing tactic or item.
- `restoreLineItem(tacticId, li)`: throws `'Its tactic no longer exists — restore the tactic first.'` if the tactic is gone; is idempotent by id; renames clashes `"name (2)"`, `"(3)"`, and so on.

#### 7.4.9 Media codes, comments
- `generateMediaCode(id, {channel, label})`:
  - `abbr` = `(channel || t.channel || 'GEN')` with non-alphanumerics stripped, first 4 characters, uppercased, or `'GEN'`.
  - `code` = `` `ZR-${abbr}-${uid('').split('-')[1]}` `` uppercased. That segment is the base-36 timestamp, e.g. `ZR-PAID-MFX3K2A1`; the `Math.random` fallback is unreachable.
  - Appends `{id: uid('mc'), code, channel ≤40, label ≤80}`. Route: `POST /api/tactics/:id/media-codes` (no audit).
- `addComment(id, {author, text})`: author default `'You'` (≤80), text ≤1000, `at` now. Route `POST /api/tactics/:id/comments`.
- Node comments: `POST /api/plans/nodes/:id/comments`. A blank text gives 400 `'Write a comment first.'`; author is `req.user.name || body.author || 'You'`. `PUT /api/plans/nodes/:id` strips `comments` from the body.

#### 7.4.10 Bulk ops (Hive9 "Honeycomb")
- **`moveTactics(ids, programId)`**:
  - Target must be a program → `'Pick the program to move the tactics into.'`
  - Nothing selected → `'Select at least one tactic to move.'`
  - A moving tactic's title clashes with a non-moving tactic in the target → `` `“${program.name}” already has a tactic called “${clash.title}”.` `` Clashes among the moving tactics themselves are not checked.
  - Trusted update of `programId` (no approval reset; lineage re-derived). Returns `{moved, programId}`.
- **`copyTactics(ids, programId)`**:
  - Errors `'Pick the program to copy the tactics into.'` / `'Select at least one tactic to copy.'`
  - Titles are made unique in the target: `title`, then `title (copy)`, `title (copy 2)`, `title (copy 3)`, and so on.
  - Copies are reset: `approvalStatus 'Created'`, `approval null`, `comments []`, `source null`, `loopId null`, `mediaCodes []`, fresh `createdAt`. Line items get new ids with `prId null` and `actual 0`.
  - Kept: budget, performance, spend overrides, targets, goals, customAttrs, dates.
  - Created trusted. Returns `{copied, ids}`.
- **Delete:**
  - `deleteTactic(id)` and `deleteTactics({ids}|{all:true})` are a single read-modify-write and return `{deleted, remaining}`.
  - Routes go through the recycle bin: `DELETE /api/tactics/:id` → `trashTactic`; `POST /api/planning/tactics/bulk-delete {ids|all}` → `trashTactics`, responding with `result`.
  - `insertTactics` restores verbatim, skipping existing ids. The recycle bin restores tactics with `approvalStatus 'Created'` plus a `restored` history entry.

#### 7.4.11 Create/update routes
- `resolveProgramId(body)`:
  - `programId` given → use it.
  - Else `campaignId` given and it is a campaign → its first program. If it has none, create `` `${campaign.name} — General` `` (owner and ownerId copied, no validation).
  - Else null.
- `POST /api/tactics` (Editor): resolve → `validateTactic(body, null)` → `createTactic(body)` (untrusted, so always `Created`). Audits `tactic.create`, notifies `ownerId`, returns the decorated tactic.
- `PUT /api/tactics/:id` (Editor): `validateTactic(body, prev)` → `updateTactic(id, body, {by})`. Notifies only if `ownerId` changed.
- Legacy aliases: `POST /api/planning/initiatives`, `PUT /api/planning/initiatives/:id` (404 `'Initiative not found'`), `DELETE /api/planning/initiatives/:id`. Same logic, no audit or notify, and responses are not decorated.

### 7.5 Board and summary (planning.js + boardConfig.js)

#### 7.5.1 Board config (`boardConfig.json`)
- `load()`: if the file is missing or has no non-empty `columns` array, seed and write `{columns: DEFAULT_COLUMNS, fields: []}`. Missing `fields` are treated as `[]`.
- `DEFAULT_COLUMNS` (`wip` 0 = no limit):

| id | name | hint | color |
|---|---|---|---|
| backlog | Backlog | Ideas & requests not yet committed | #86868b |
| planned | Planned | Scoped and scheduled for a quarter | #0071e3 |
| in-progress | In progress | Actively being executed | #c77700 |
| launched | Launched | Live / delivered | #1baf7a |

- **Templates.** All keep the `in-progress` / `launched` ids that dashboards roll up on.
  - `default` "Default — 4 stages": `DEFAULT_COLUMNS`.
  - `kanban-plus` "Kanban+ — adds In review & Blocked": backlog, planned, in-progress, `in-review` (In review, 'Awaiting review / approval', #9f4bd6), `blocked` (Blocked, 'Stuck — needs unblocking', #d03b3b), launched.
  - `campaign-ops` "Campaign ops — brief → live":

    | id | name | hint | color |
    |---|---|---|---|
    | backlog | Brief | Requested / being briefed | #86868b |
    | planned | Building | In production | #0071e3 |
    | in-review | Review | QA / approval | #9f4bd6 |
    | in-progress | Scheduled | Ready & scheduled | #c77700 |
    | launched | Live | Launched | #1baf7a |
- `slug(s, sep)`: lowercase, non-`[a-z0-9]` runs become `sep`, leading and trailing `sep` trimmed.
- **`normColumn(c)`:**
  - `name` ≤40, default `'Column'`.
  - `id` = `(trimmed c.id || slug(name,'-') || uid('col'))`, ≤40.
  - `hint` ≤80; `color` must match `/^#[0-9a-fA-F]{6}$/`, else `#86868b`.
  - `wip` = `max(0, round)`.
- **`setColumns(cols)`:**
  - Must be a non-empty array: `'A board needs at least one column.'`
  - At most 12: `'A board can have at most 12 columns.'`
  - Duplicate ids get `-2` appended repeatedly until unique (`x`, `x-2`, `x-2-2`).
  - Replaces the columns and returns them. Tactic statuses are **not migrated**; see orphan handling below.
- `applyTemplate(id)`: unknown → `'Unknown board template.'`; otherwise `setColumns(template.columns)`.
- **`setFields(fields)`:**
  - Must be an array: `'Invalid fields.'`; at most 20: `'At most 20 custom fields.'`
  - `label` is trimmed, ≤40.
  - `id` = `(trimmed f.id || slug(label,'_') || uid('fld'))`, ≤40, de-duplicated by appending `_2` repeatedly.
  - `type` is one of `text|number|select|date` (default text).
  - `options` (select only): strings ≤40, non-empty, max 20; `[]` for other types.
  - Fields with an empty label are dropped after de-duplication.
- `getConfig()` returns `{columns, fields, templates: [{id, name}]}`.
- WIP limits are display-only; the client highlights `count > wip`. The server never enforces them.
- Routes:
  - `GET /api/planning/board-config` (any signed-in user).
  - `PUT /api/planning/board-config/columns {columns}`, `PUT /api/planning/board-config/fields {fields}`, `POST /api/planning/board-config/template {template}`. All use `requirePlanner`; errors → 400; no audit.

#### 7.5.2 `board(filter)` (`GET /api/planning/board?planId&campaignId&programId&planIds`)
- Applies the filters `programId`, `campaignId`, `planId` (exact), then `planIds`, which may be an array or comma-separated string, trimmed with empties dropped; an empty set means no filter.
- Decorates every tactic.
- `columns = config.columns.map(c => ({...c, initiatives: tactics with status === c.id}))`.
- **Orphans** (status not a configured column) are appended to the **first** column so they are never lost. The next save persists a valid status through `statusOrDefault`.
- Returns `{columns, meta: {types: TYPES, priorities: PRIORITIES, owners: unique owners of the filtered items, columns: config columns}}`.

#### 7.5.3 `summary(filter)` (`GET /api/planning/summary`, same filters)
```
total, inFlight = #status 'in-progress', launched = #status 'launched',
plannedBudget = Σ budget, avgProgress = round(mean progress) (0 if none),
quarters = unique quarter strings sorted lexicographically, p0 = #priority 'P0'
```
**QUIRK:** the lexicographic sort puts "Q1 2027" before "Q3 2026".

### 7.6 Seed data and one-time migrations

#### 7.6.1 Container seed (when `plans.json` is not an array)
Seed nodes have `description:''`, no dates, `attrs:{}`, and the same timestamp, unless overridden below.

- **Plan:** "FY2026 Marketing Plan".
  - Owner Abhishek Biyani, `fiscalYear '2026'`, `currency 'USD'`, dates 2026-01-01 → 2026-12-31.
  - Description: "Company-wide marketing plan of record — anchored on the Nova GA launch (Q3 2026)."
- **Campaigns** (all under the plan):
  - **Nova GA Launch** (Marcus Lee): "The flagship Q3 2026 go-to-market motion for Nova — paid, lifecycle and events."
  - **Enterprise ABM** (Marcus Lee): "Named-account motion targeting the top 150 enterprise accounts for Nova."
  - **Brand & Community** (Sam Okafor): "Organic growth, thought leadership, developer community and partner co-marketing for Nova."
- **Programs:**
  - Nova GA Launch: Paid Acquisition (Marcus Lee), Lifecycle & Email (Dana Alvarez), Events & Webinars (Abhishek Biyani).
  - Enterprise ABM: Named-Account Plays (Marcus Lee).
  - Brand & Community: Content Engine (Sam Okafor), Partner Co-Marketing (Dana Alvarez).
- File order is plan, 3 campaigns, 6 programs, so `defaultProgramId()` is **Paid Acquisition**.
- The seed lacks `dateMode` and `planBudget`, so the migration (§7.6.3) normalizes it immediately. Result:
  - Plan: `dateMode 'manual'`, `modelId 'model-standard'`, `teams []`, `isStrategic false`, `goals {}`, `defaultMechanism 'E'`.
  - Campaigns and programs: `dateMode 'auto'`.
  - All nodes: `planBudget 0`, `status ''`, `approval null`, `comments []`, `ownerId ''`.

#### 7.6.2 Tactic seed: the Nova launch board
Applies when `tactics.json` is not an array and there is no legacy file.
- Each tactic is attached by program name, gets lineage, and is created with `approvalStatus = defaultApprovalFor(status)` (trusted).
- Default `quarter 'Q3 2026'`, overridden by the start date when dated.
- Dates are given as `startDate` and `dueDate`; `endDate` mirrors `dueDate`.
- Other defaults: `goals` all 0 (waterfall of 0 inquiries); no line items; `impact`/`effort` 3.

| # | Title | Program | Status | Approval | Owner | Channel | Type | Pri | Budget | Prog% | Start → Due | Quarter (stored) | Goal |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | Nova launch site & pricing pages | Content Engine | launched | Approved | Sam Okafor | Organic / SEO | Content | P0 | 14,000 | 100 | 2026-05-12 → 06-20 | Q2 2026 | GA site + pricing live |
| 2 | Nova GA announcement & press release | Content Engine | launched | Approved | Sam Okafor | Content Syndication | Launch | P0 | 9,000 | 100 | 06-22 → 07-08 | Q2 2026 | Nova GA announced |
| 3 | Brand + category paid search | Paid Acquisition | in-progress | Approved | Marcus Lee | Paid Search | Campaign | P0 | 120,000 | 60 | 07-01 → 09-30 | Q3 2026 | 2,400 launch leads |
| 4 | Paid social retargeting flight | Paid Acquisition | in-progress | Approved | Marcus Lee | Paid Social | Campaign | P1 | 60,000 | 45 | 07-10 → 09-30 | Q3 2026 | Re-engage site visitors |
| 5 | 'Meet Nova' launch webinar | Events & Webinars | in-progress | Approved | Abhishek Biyani | Events & Webinars | Event | P0 | 45,000 | 70 | 07-05 → 08-14 | Q3 2026 | 1,500 registrations |
| 6 | Trial → paid lifecycle nurture | Lifecycle & Email | in-progress | Approved | Dana Alvarez | Email | Campaign | P1 | 15,000 | 40 | 07-01 → 09-15 | Q3 2026 | Lift trial → paid conversion |
| 7 | Top-150 named-account ABM plays | Named-Account Plays | planned | Created | Marcus Lee | Paid Social | Campaign | P0 | 90,000 | 0 | 08-01 → 09-30 | Q3 2026 | Open enterprise pipeline |
| 8 | Developer community launch | Content Engine | planned | Created | Sam Okafor | Organic / SEO | Content | P2 | 10,000 | 0 | 08-10 → 09-20 | Q3 2026 | Seed developer adoption |
| 9 | Partner co-marketing kits | Partner Co-Marketing | planned | Created | Dana Alvarez | Content Syndication | Content | P2 | 18,000 | 0 | 08-20 → 09-30 | Q3 2026 | Enable 5 launch partners |
| 10 | Analyst briefings (Nova GA) | Content Engine | backlog | Created | Sam Okafor | Content Syndication | Research | P1 | 20,000 | 0 | 10-01 → 10-31 | Q4 2026 | Secure analyst coverage |
| 11 | Customer story: beta → GA | Content Engine | backlog | Created | Sam Okafor | Content Syndication | Content | P2 | 6,000 | 0 | (undated) | Q4 2026 (typed) | Proof from a beta customer |
| 12 | Post-launch retargeting expansion | Paid Acquisition | backlog | Created | Marcus Lee | Paid Social | Campaign | P2 | 30,000 | 0 | 10-01 → 12-15 | Q4 2026 | Scale winning audiences |

The source typed a quarter of `'Q4 2026'` on #10–12; dated rows recompute it from the start date anyway.

**Roll-ups this produces** (planned cost / auto dates):

| Program | Planned cost | Auto dates | Tactics |
|---|---|---|---|
| Paid Acquisition | 210,000 | 07-01 → 12-15 | 3 |
| Lifecycle & Email | 15,000 | 07-01 → 09-15 | 1 |
| Events & Webinars | 45,000 | 07-05 → 08-14 | 1 |
| Named-Account Plays | 90,000 | 08-01 → 09-30 | 1 |
| Content Engine | 59,000 | 05-12 → 10-31 | 5 (the undated tactic is ignored for dates) |
| Partner Co-Marketing | 18,000 | 08-20 → 09-30 | 1 |

- Campaigns: Nova GA Launch 270,000 (07-01 → 12-15); Enterprise ABM 90,000 (08-01 → 09-30); Brand & Community 77,000 (05-12 → 10-31).
- Plan: **437,000** across 12 tactics.
- As of today (2026-10-17), the approved seed tactics display lifecycle `Complete`.

#### 7.6.3 Migrations and idempotent seeds
1. **Node upgrade** (every `listNodes`, effectively once): if any node lacks the `dateMode` or `planBudget` keys, every node becomes `{id, createdAt, updatedAt, ...clean(n, n)}` and is written back. Unknown extra fields are dropped.
2. **Legacy board migration** (first `listTactics` when `tactics.json` is missing): if `initiatives.json` is an array, each item keeps its id.
   - `programId = guessProgramId(item) || defaultProgramId()`, plus lineage and `approvalStatus = defaultApprovalFor(status)`; normalized trusted.
   - `guessProgramId` takes the first matching rule, then the first program whose lowercased name contains the needle:
     - channel contains `email` or type `ops` → `lifecycle`
     - channel contains `event` or type `event` → `event`
     - channel contains `organic` or `seo`, or type `content` → `content`
     - tags include `abm` → `named-account`
     - channel contains `paid`, `search` or `social` → `paid`
     - else null.
   - Type comparisons are lowercased.
3. **Tactic upgrade** (every `listTactics`): if any tactic lacks `approvalStatus` or `goals`, all are re-normalized trusted with `keepUpdatedAt`. A missing approval is defaulted from the board status.
4. **`ensureMultiPlanDemo()`** runs on every server start (end of `index.js`, errors warned). It is additive and idempotent by plan name.
   - If no plan is named **"EMEA Field Marketing"**, it creates:
     - plan (owner Noor Haddad, `fiscalYear 'FY2026'` → `'2026'`, USD, description "Regional field-marketing plan for EMEA — events, demand-gen and content.")
     - campaign **EMEA Demand Gen** (Noor Haddad)
     - program **Field & Events** (Mateo Duarte)
   - It then creates 4 tactics through the untrusted `createTactic`, so all are `Created` even when in-progress:

| Title | Type | Status | Pri | Owner | Channel | Goal | Budget | Prog | Dates | Quarter stored |
|---|---|---|---|---|---|---|---|---|---|---|
| EMEA Launch Webinar | Event | in-progress | P1 | Noor Haddad | Webinar | 600 registrations | 18,000 | 45 | 2026-08-20 → 08-20 | Q3 2026 |
| DACH Paid Search | Campaign | planned | P1 | Lena Fischer | Paid Search | 1,200 leads | 60,000 | 10 | 09-01 → 11-30 | Q3 2026 (typed Q4, overridden) |
| France Field Event — Paris | Event | planned | P0 | Mateo Duarte | Event | 80 enterprise meetings | 45,000 | 5 | 10-15 → 10-16 | Q4 2026 |
| EMEA Case Study Series | Content | backlog | P2 | Sofia Rossi | Content | 4 published stories | 12,000 | 0 | 09-10 → 10-10 | Q3 2026 (typed Q4) |

   - Returns `{created, planId, tactics: 4}`.
5. **Models:** `models.json` is seeded with the default model if missing or empty.
6. **Board config:** seeded with the default columns if missing.

### 7.7 Campaign dual-plan model (campaignPlans.js)

Each campaign is presented as a pair of plans, both **derived** deterministically from its tactics (`t.campaignId === campaign.id`), with no AI:
- **SP&A** (Sales, Promotion & Advertising): the spend plan (planned, committed, actual).
- **BCPIR** (Business Case & Post-Investment Review): the returns plan (planned vs realized revenue, pipeline, leads, ROI, ROAS).

#### 7.7.1 Deterministic jitter
```
hash(str): FNV-1a 32-bit — x = 2166136261; for each char: x ^= code; x = Math.imul(x, 16777619); return x >>> 0
rand01(id, salt) = (hash(`${id}:${salt}`) % 100000) / 100000          // [0,1)
```
Salts: `'roas'` (jr), `'pace'` (jp), `'leads'` (jl). Values are stable per tactic id. Seeded ids are random, so the figures differ per install but never change between reloads.

#### 7.7.2 Constants
- `ROAS_BY_TYPE = {Campaign: 3.4, Launch: 2.8, Content: 5.2, Event: 2.4, Research: 1.7, Ops: 1.4}`; any other type uses 3.
- `cplFor(channel, type)`: `c = `${channel} ${type}`.toLowerCase()`. The first match wins:
  - contains `event` or `webinar` → 220
  - `paid search` or `search` → 65
  - `paid social` or `social` → 48
  - `email` or `lifecycle` → 22
  - `organic`, `seo` or `content` → 34
  - else 60
- `COMMIT_FACTOR = {launched: 1, 'in-progress': 0.72, planned: 0.18, backlog: 0.03}`. Any other or custom column uses **0.1**.
- `r2 = Math.round`.

#### 7.7.3 `deriveTacticFinance(t)`
```
budget   = max(0, t.budget);  progress = clamp(t.progress/100, 0, 1)
plannedSpend = budget
actualSpend  = t.spendActual != null ? t.spendActual : round(budget × progress × (0.9 + 0.2·jp))
committed    = t.spendCommitted != null ? t.spendCommitted : max(actualSpend, round(budget × COMMIT_FACTOR[status]))

realPlanRev  = perf.plan.revenue || goals.revenue || 0
realActRev   = perf.actual.revenue || 0
roasTarget   = (ROAS_BY_TYPE[type] || 3) × (0.82 + 0.42·jr)
plannedRevenue = realPlanRev > 0 ? realPlanRev : round(budget × roasTarget)
realized     = min(1.18, progress × (0.86 + 0.34·jp))
actualRevenue  = realActRev > 0 ? realActRev : round(plannedRevenue × realized)

plannedPipeline = perf.plan.pipeline > 0 ? that : round(plannedRevenue × 3.1)
actualPipeline  = perf.actual.pipeline > 0 ? that : round(actualRevenue × 2.9)

cpl = cplFor(channel, type)
plannedLeads = (perf.plan.leads || goals.inquiries) > 0 ? that : round(budget / cpl)
actualLeads  = perf.actual.leads > 0 ? that : round(plannedLeads × min(1.15, progress × (0.9 + 0.2·jl)))

roi  = actualSpend > 0 ? (actualRevenue − actualSpend)/actualSpend : 0
roas = actualSpend > 0 ? actualRevenue/actualSpend : 0
```
The model-driven tactic goals (`goals.revenue`, `goals.inquiries`) feed the business case when no explicit performance plan exists.

#### 7.7.4 Register: `listCampaignPlans()` (`GET /api/planning/plans`)
Per campaign (`plansForCampaign`):
- **Sums:** `plannedCost`, `committed`, `actualSpend`, `plannedReturn`, `actualReturn`.
- `variancePct = plannedCost>0 ? (actualSpend − plannedCost)/plannedCost×100 : 0`; `roi` and `roas` as above.
- `status = campaign.status || statusOf(items)`, where `statusOf` = `'Published'` if any tactic is launched or in-progress, else `'Created'`. Possible values are therefore Draft, Submitted, Published or Created.
- `fyLabel(raw)` strips a leading `FY` and returns `FY<n>`, or `''`.
- Dates: `plan.startDate || campaign.startDate || '2026-01-01'`; end likewise with `'2026-12-31'`.
- `displayId(campaignId, type)`: `base = 8000 + hash(campaignId) % 900`. SP&A = base (8000–8899); BCPIR = base + 150.
- **Row:**
  ```
  {campaignId, campaignName, owner, ownerId, planId, planName, fiscalYear, tacticCount, approval, startDate, endDate,
   spa:   {id:'spa:<cid>', type:'SP&A', name:'<FY> · <Campaign> — SP&A', displayId, status, startDate, endDate, plannedCost, committed, actualSpend, variancePct, tacticCount},
   bcpir: {id:'bcpir:<cid>', type:'BCPIR', name:'… — BCPIR', displayId, status, dates, investment: actualSpend, plannedReturn, actualReturn, roi, roas, tacticCount}}
  ```
  The `"<FY> · "` prefix is omitted when there is no FY.
- **Totals** over all rows: `plannedCost`, `committed`, `actualSpend`, `plannedReturn`, `actualReturn`, plus `roi` and `roas`.
- **Groups:** one per plan node, including plans with no campaigns: `{planId, planName, fiscalYear, owner, ownerId, currency, startDate, endDate, campaigns}`. Campaigns whose parent plan is missing go in `{planId:null, planName:'Unassigned', fiscalYear:'', campaigns}`.
- Returns `{currency:'USD', totals, groups, campaigns: rows}`.

#### 7.7.5 Detail: `getCampaignPlan('spa:<cid>' | 'bcpir:<cid>')` (`GET /api/planning/plans/:id`)
The id is split on `:`. An invalid type or missing campaign returns null (404 `'Plan not found'`).

**Header:**
```
{id, type, kind:'Spend plan'|'Post-investment review', name, displayId, status, campaignId, campaignName, planName, fiscalYear,
 startDate: plan.startDate || '2026-01-01', endDate: plan.endDate || '2026-12-31', currency:'USD',
 counterpartId (the other plan), tacticCount, approval, planNodeId, defaultProgramId (the campaign's first program; new rows attach there)}
```
**QUIRK:** header dates ignore campaign dates, unlike the register.

Quarter buckets: `quarterKey(q)` gives `{key:'YYYY-Qn', label:'Qn YYYY'}`, or `{key:'zzzz', label: q || 'Unscheduled'}`. Buckets are sorted by key.

**SP&A response:**
- `kpis`:
  ```
  {plannedCost, committed, actual,
   remaining = max(0, planned − actual),
   variancePct = (actual − planned)/planned×100,
   utilizationPct = actual/planned×100}
  ```
- `byQuarter [{quarter, planned, actual}]`.
- `byCategory`: key is `channel || type || 'Other'`, summed planned spend, sorted descending → `[{name, value}]`.
- `lineItems` (one per tactic): `{id, tactic, owner, channel (channel||type), quarter, status, planned, committed, actual, variancePct}`, sorted by planned descending.
- `procurement` = `rollupForCampaign(campaignId, tacticIds)`. It takes PRs on the campaign or its tactics and returns `{count, committed (approved|po|invoiced|received), invoiced (invoiced|received), pending (submitted), draft, requests:[{id, title, vendor, status, total, tacticId}]}`.
- `budget` = Budget-module row for the campaign: `{allocated: row.planned, committed, variance, utilizationPct}`, or null if missing or on error.

**BCPIR response:**
- `kpis`:
  ```
  {investment (Σ actualSpend), plannedReturn, actualReturn, roi, roas,
   plannedPipeline, actualPipeline, plannedLeads, actualLeads,
   attainmentPct = actualReturn/plannedReturn×100}
  ```
- `byQuarter` (planned vs actual revenue).
- `byChannel [{name, planned, actual}]`, sorted by actual descending.
- `businessCase`: `[{id, tactic, owner, channel, targetLeads, actualLeads, targetPipeline, actualPipeline, targetRevenue, actualRevenue, spend, roi, roas}]`, sorted by actualRevenue descending.

#### 7.7.6 Campaign plan approval (Draft → Submitted → Published)
- `submitPlan(cid, {by, approver, approverId})` (campaign only, else null/404):
  - `status 'Submitted'`.
  - `approval = {submittedBy: by||'You', submittedAt: now, approver||'', approverId||'', approvedBy:'', approvedAt:'', note:''}`. The previous record is **replaced**, so earlier rejection fields are lost.
- `approvePlan(cid, {by})`: `status 'Published'`, `approval = {...prev, approvedBy: by||'You', approvedAt: now, note:''}`.
- `rejectPlan(cid, {by, note})`: `status 'Draft'`, `approval = {...prev, approvedBy:'', approvedAt:'', rejectedBy: by||'You', rejectedAt: now, note: note||''}`.
- **No state-machine checks in the service.** For example, a Draft plan can be "approved" directly. Routes only apply `approvalGuard`.
- **QUIRK:** `PUT /api/plans/nodes/:id` also accepts `status` and `approval` through `clean`, so a client could bypass the workflow. Tactics are write-protected; campaign status is not.
- Routes (`POST /api/planning/plans/:campaignId/{submit,approve,reject}`, Editor):
  - 404 `'Campaign not found'`; 403 from the guard on approve/reject.
  - Audits `plan.submit` / `plan.approve` / `plan.reject`.
  - Submit notifies `approverId`: title `'Plan approval needed'`, route `/planning?plan=spa:<cid>`.

### 7.8 Imports

#### 7.8.1 File import: CSV and XLSX (`importer.js` + `spreadsheet.js`)
**`toCsvText({csv, fileBase64, name})`:**
- A non-blank `csv` string is used as-is.
- Otherwise `fileBase64` is decoded. If `name` ends in `.csv`, `.txt` or `.tsv`, it is decoded as UTF-8 text.
  - **QUIRK:** TSV is not converted, and the parser only splits on commas.
- Otherwise it is read with SheetJS: `XLSX.read(buf, {type:'buffer', cellDates:true})`, first sheet → `sheet_to_csv(ws, {dateNF:'yyyy-mm-dd', blankrows:false})`. An empty workbook gives `'The workbook has no readable sheet.'`
- Nothing provided → `'No file content provided.'`

**`parseCsv(text)`:**
- CRLF and CR are normalized to LF.
- Quoted fields: `"` toggles quoting and `""` inside quotes is a literal quote. Commas and newlines inside quotes are preserved.
- A trailing field or row is flushed.
- Rows where every cell trims to empty are dropped.

**Header mapping** (trimmed, lowercased, exact alias):

| Field | Aliases |
|---|---|
| title | title, name, tactic |
| type | type |
| owner | owner |
| quarter | quarter, qtr |
| status | status |
| priority | priority |
| budget | budget, cost, amount |
| channel | channel |
| goal | goal, objective |

Other columns are ignored. Dates, descriptions and programs **cannot** be imported via CSV.

**`importTacticsCsv(text)`:**
- Fewer than 2 rows → `{imported:0, errors:['CSV needs a header row and at least one data row.']}`.
- No title column → `{imported:0, errors:['CSV must include a "title" (or "name") column.']}`.
- For each data row (`r` = index; the reported row number is `r+1`, matching spreadsheet numbering):
  - Cells are trimmed.
  - No title → error `` `Row ${r+1}: missing title — skipped.` ``
  - Otherwise `createTactic` with:
    - `type`: an exact, case-sensitive TYPES member, else Campaign.
    - `owner`: default 'Unassigned'; `quarter`: default 'Q3 2026'.
    - `status`: `STATUS_MAP[lowercased]`, else `'backlog'`. `STATUS_MAP = {backlog, planned, 'in-progress', 'in progress'→in-progress, launched, done→launched, complete→launched}`.
    - `priority`: an exact `P0|P1|P2`, else **`'P2'`** (the general tactic default is P1).
    - `budget = Number(str.replace(/[^0-9.]/g,''))||0`, so `"$45,000"` becomes 45000.
    - `channel`, `goal`, `programId = defaultProgramId()`.
  - A throw gives `` `Row ${r+1}: ${message}` ``.
- No `validateTactic`, so there is no uniqueness check. Approval is `Created`.
- Returns `{imported, ids, errors}`. `ids` powers "Undo import".
- `SAMPLE_CSV`:
  ```
  title,type,owner,quarter,status,priority,budget,channel,goal
  Spring Field Event,Event,Abhishek Biyani,Q2 2026,planned,P1,45000,Events & Webinars,300 registrations
  SEO Content Sprint,Content,Sam Okafor,Q3 2026,in-progress,P2,20000,Organic / SEO,+15% organic sessions
  ```
- Routes:
  - `GET /api/import/sample` → `{csv}`.
  - `POST /api/import/tactics {csv | fileBase64, name}` (Editor; JSON body limit 60 MB). Errors → 400. Audits `import.tactics`.

#### 7.8.2 Trello and Jira import (`boardImport.js`)
Calls are server-side `fetch`, so there are no CORS issues. **Credentials are never stored.** The client remembers only the Jira site URL and email.

**Column resolution:**
- `norm(s)`: lowercase, non-alphanumeric runs become a space, trimmed.
- `bucketOf(name)` tests `` ` ${norm} ` `` with substring regexes in this order:
  1. `/(done|complete|completed|closed|closs|launch|launched|shipped|resolved|live|released)/` → `launched`
  2. `/(progress|doing|active|develop|review|testing|qa|building|execut)/` → `in-progress`
  3. `/(planned|scheduled|ready|selected|committed|sprint|upcoming|next|to do|todo)/` → `planned`
  4. `/(backlog|idea|ideas|triage|new|open|requested|inbox|proposed)/` → `backlog`
  5. otherwise null
- These are substring matches, not word matches; for example "Ready for QA" goes to in-progress.
- `resolveColumn(sourceName)`, against the configured columns:
  1. No columns → `'backlog'`.
  2. Exact match, `norm(col.name) === norm(source)` or `col.id === norm(source)`.
  3. Bucket the source, then use the column whose id equals the bucket, else the first column whose name buckets the same.
  4. The first column.
- `PRIORITY_MAP = {highest:'P0', high:'P0', medium:'P1', low:'P2', lowest:'P2'}`, else P1.
- `quarterOf(date)` gives `Q{floor(month/3)+1} {year}` (local time), or undefined if invalid, in which case the tactic default 'Q3 2026' applies.
- `dateOnly` = `slice(0,10)`.

**`httpJson(url, opts, label)` errors:**
- Network failure → `` `Couldn't reach ${label}. Check the URL / your network.` ``
- Non-OK response → `` `${label} ${status}${hint} ${body.slice(0,180)}` ``, trimmed. `hint = ' — check your credentials/permissions.'` for 401/403.
- Unparseable body → `` `${label} returned a non-JSON response.` ``

**Trello** (base `https://api.trello.com/1`, auth query `key=…&token=…`):
- Missing creds → `'Trello needs both an API key and a token.'`
- `listTrelloBoards({key, token})`: `GET /members/me/boards?fields=name,closed` → non-closed boards as `{id, name}`.
- `importTrello({key, token, boardId})`:
  - No board → `'Pick a Trello board to import.'`
  - Three parallel calls:
    - `GET /boards/{id}?fields=name`
    - `GET /boards/{id}/lists?fields=name`
    - `GET /boards/{id}/cards?fields=name,desc,due,idList,labels&members=true&member_fields=fullName`
  - `connectionId = uid('conn')` is generated up front. `programId = defaultProgramId()`.
  - Each card becomes `createTactic` (untrusted, no validateTactic):
    - `title`: `name||'Untitled card'`; `description`: `desc`.
    - `status = resolveColumn(listName)`.
    - `owner` = first member's `fullName`, else Unassigned.
    - `tags` = label `name || color`, max 8.
    - `dueDate = endDate = dateOnly(due)`; `quarter = quarterOf(due)`.
    - `source {provider:'trello', name: boardName, connectionId}`.
  - Per-card errors: `` `Card “${name||'?'}”: ${msg}` ``.
  - The connection is recorded only if at least one tactic was created.
  - Returns `{source:'trello', board: boardName, imported, ids, errors, connectionId|null}`.

**Jira Cloud** (Basic auth `base64(email:apiToken)`, `Accept: application/json`, base URL with trailing slashes stripped):
- Missing creds → `'Jira needs a site URL, your email, and an API token.'`
- `listJiraProjects`: `GET /rest/api/3/project/search?maxResults=100` → `(data.values || data)` as `{key, name}`.
- `importJira({baseUrl, email, apiToken, projectKey, jql})`:
  - Neither given → `'Pick a Jira project (or provide a JQL query).'`
  - Query = trimmed JQL, or `` `project = "${projectKey}" ORDER BY status` ``. JQL overrides the project.
  - `jiraSearch` tries `/rest/api/3/search/jql`, then legacy `/rest/api/3/search`, falling through only when the error message matches `/ 404| 410/`.
    - Params: `jql`, `maxResults=100`, `fields=summary,status,assignee,priority,duedate,labels,description`.
    - No pagination, so at most 100 issues.
  - Each issue becomes a tactic:
    - `title`: `summary || key || 'Untitled issue'`.
    - `description` = `adfText(description).slice(0,800)`. ADF is flattened: a string is returned as-is; else `node.text`; else the children are concatenated, followed by `\n` for paragraph and heading nodes that have children.
    - `status = resolveColumn(status.name || status.statusCategory.name)`. **QUIRK:** the doc says status *category* drives the mapping, but the code prefers the status *name*.
    - `owner`: `assignee.displayName`; `priority` via `PRIORITY_MAP`.
    - `tags`: `[key, ...labels]`, max 8.
    - `due`, `quarter`, `programId`.
    - `source {provider:'jira', name: projectKey ? 'Jira · KEY' : 'Jira · JQL query', connectionId}`.
  - Per-issue errors: `` `Issue ${key||'?'}: ${msg}` ``.
  - Returns `{source:'jira', project: projectKey||'JQL', imported, ids, errors, connectionId|null}`.
- Re-importing never de-duplicates.
- Routes (Editor; errors → 400): `POST /api/import/trello/boards`, `/api/import/trello`, `/api/import/jira/projects`, `/api/import/jira`.
  - **QUIRK:** the audit label reads `out.board.name` / `out.project.name`, which are strings, so labels fall back to 'Trello board' / 'Jira project'.

#### 7.8.3 Board connections (`boardConnections.json`)
- **Record:**
  ```
  {id (from importer), provider ≤20, name ≤140, key (Trello boardId / Jira projectKey) ≤60 | null,
   count: tacticIds.length, tacticIds, importedAt}
  ```
- `listConnections()` adds `liveCount` = the number of tactics where `source.connectionId === conn.id` **or** the id is in `tacticIds`. Route: `GET /api/import/connections`.
- **Disconnect** `removeConnection(id)` (`DELETE /api/import/connections/:id`, Editor):
  - Target set = recorded ids ∪ tactics stamped with this `connectionId`. They are hard-deleted via `deleteTactics({ids})`, bypassing the recycle bin, so it is irreversible.
  - The record is then removed.
  - Returns `{removed:1, deleted, connection}`, or `{removed:0, deleted:0}` if unknown.
  - Never calls Trello or Jira. No audit.

### 7.9 Calendars

#### 7.9.1 ICS feed (`calendar.js`) at `GET /api/planning/calendar.ics`
- **Public, no auth.** Accepts the `planId`, `campaignId`, `programId` and `planIds` filters.
- Items = `board(filter).columns.flatMap(c => c.initiatives)`.
- Headers: `Content-Type: text/calendar; charset=utf-8`, `Content-Disposition: inline; filename="zenreach-calendar.ics"`, `Cache-Control: public, max-age=3600`.
- **`tacticsToIcs(tactics, calName='ZenReach Marketing Calendar')`.** Lines are joined with CRLF and end with a trailing CRLF.
  ```
  BEGIN:VCALENDAR / VERSION:2.0 / PRODID:-//ZenReach//Marketing Planning//EN / CALSCALE:GREGORIAN / METHOD:PUBLISH
  X-WR-CALNAME:<esc(calName)> / X-WR-CALDESC:Scheduled marketing tactics from ZenReach
  X-PUBLISHED-TTL:PT6H / REFRESH-INTERVAL;VALUE=DURATION:PT6H
  per tactic:
    start = icsDate(startDate||endDate||dueDate)   // strip '-', first 8 chars; skip tactic unless exactly 8 chars
    end   = addDay(icsDate(endDate||dueDate||startDate))   // all-day DTEND exclusive: +1 day via Date.UTC
    BEGIN:VEVENT
    UID:<tacticId>@zenreach
    DTSTAMP:<now as YYYYMMDDTHHMMSSZ>
    LAST-MODIFIED:<stamp(updatedAt)>   // invalid → now
    SUMMARY:<esc(title)>               (folded)
    DTSTART;VALUE=DATE:<start>
    DTEND;VALUE=DATE:<end>
    DESCRIPTION:<esc(lines)>           (folded)
    CATEGORIES:<esc(type||'Marketing')>
    END:VEVENT
  END:VCALENDAR
  ```
- DESCRIPTION is made from the present fields, joined with `\n`: `Type: …`, `Owner: …`, `Channel: …`, `Goal: …`, `Status: …` (falsy ones omitted).
- `esc` turns `\` into `\\`, `;` into `\;`, `,` into `\,`, and newlines into `\n`.
- `fold` cuts at 75 **characters**, then continuation lines of `' '` + 74 characters joined with CRLF. RFC 5545 counts octets, so non-ASCII text can overflow.
- Evergreen tactics (no end) become a single-day event.

#### 7.9.2 Google Calendar push (`googleCalendar.js`)
One-way push from ZenReach to Google, with **no OAuth**. It reuses the GA4 **service account**: the user shares their calendar with the service-account email and grants "Make changes to events".

- **Environment variables:** `GOOGLE_APPLICATION_CREDENTIALS`, or `GOOGLE_CREDENTIALS_JSON`. Both are treated as **file paths** (not inline JSON). A relative path is resolved against the project root, two directories above `services/`.
- **Client:** `google-auth-library` `JWT({keyFile, scopes:['https://www.googleapis.com/auth/calendar']})`, cached as a module singleton.
  - No key → `'No Google service-account key configured (set GOOGLE_APPLICATION_CREDENTIALS).'`
  - No token → `'Could not obtain a Google access token from the service account.'`
- API base `https://www.googleapis.com/calendar/v3`, sending a bearer token and a JSON body.
- **Connection persistence:** `settings.json` → `googleCalendar: {calendarId, connectedAt, planIds}`.
  - `setConnection` unions the new `planIds` with the previous ones. Empty input keeps the previous list.
  - `connectedAt` is kept if the calendar id is unchanged.
- **`calendarStatus()`** (`GET /api/integrations/google-calendar/status`):
  - Returns `{keyConfigured: Boolean(env path), serviceAccount: client_email read from the key file (or null), connected, calendarId, connectedAt, planIds, hint}`.
  - `hint` is `` `Share your Google Calendar with ${email} ("Make changes to events"), then sync.` `` or `'No Google service-account key configured (GOOGLE_APPLICATION_CREDENTIALS).'`
- **Plan colors:** `GCAL_COLORS = ['1','2','3','5','6','7','9','10','11']`. Plan index `i` in node order gets `colors[i % 9]`.
- **`friendlyError(status, json)`.** `msg = error.message || error_description || 'HTTP <status>'`:
  - 404 → `` `Calendar not found or not shared with the service account. ${msg}` ``
  - 403 with `/not been used|disabled|SERVICE_DISABLED/i` → `` `Google Calendar API is not enabled for this project. Enable it in Google Cloud Console, then retry. (${msg})` ``
  - Other 403 → `` `Access denied — share your calendar with the service account and grant "Make changes to events". (${msg})` ``
  - Otherwise `msg`.
- **Event mapping `tacticToEvent(t, planInfo)`:**
  - `start = (startDate||endDate||dueDate).slice(0,10)`, which must match `YYYY-MM-DD`; otherwise the tactic is skipped.
  - Inclusive end = `endDate||dueDate||startDate`, else start.
  - Event fields:
    ```
    id: sha1(tacticId) hex (40 chars — valid Google base32hex id)
    summary: planInfo ? `[${planName}] ${title||'Marketing tactic'}` : (title||'Marketing tactic')
    description: lines Plan:, Type:, Owner:, Channel:, Goal:, Status:, '', 'Synced from ZenReach'
    start:{date}, end:{date: inclusiveEnd + 1 day}
    source:{title:'ZenReach', url:'http://localhost:5173/planning'}
    extendedProperties.private:{zenreachTacticId, zenreachPlanId}
    colorId (per plan)
    ```
  - **QUIRK:** empty-string fields survive the description filter, which removes only undefined, null and false, so blank lines can appear.
- **Idempotency:** the event id is deterministic per tactic. Upsert is `POST /calendars/{cal}/events`, and on **409** `PUT /calendars/{cal}/events/{id}`. Re-syncing updates events rather than duplicating them. Events for tactics deleted since a sync are never removed.
- **`syncToCalendar(calendarId, planIds)`** (`POST …/google-calendar/sync {calendarId, planIds}`, Editor, errors → 400):
  1. The calendar id must contain `@`, else `'A valid Google Calendar ID (usually your Google account email) is required.'`
  2. Get a token, then probe `GET /calendars/{id}`; a failure throws `friendlyError`.
  3. Filter tactics by the plan set (null = all), map them to events and upsert sequentially.
  4. Record the connection with `syncedPlanIds` = the given set, or every distinct tactic `planId`.
  5. Return `{calendarId, calendarName: probe.summary||id, scheduled: #events, synced, failed, errors (first 5, "<summary>: <msg>"), connected:true, connectedAt, planIds}`.
- **`clearFromCalendar(calendarId, planIds)`** (`POST …/clear`):
  - Same `@` check, with the message `'A valid Google Calendar ID is required.'`
  - Probe failure → `{calendarId, deleted:0, note}`.
  - Otherwise `DELETE` each current tactic's event id. OK or 410 counts as deleted; 404 is ignored.
  - Returns `{calendarId, deleted}`.
- **`disconnect()`** (`POST …/disconnect`): best-effort `clearFromCalendar(conn.calendarId)` for all plans, then deletes `settings.googleCalendar`. Returns `{disconnected:true, calendarId, cleared}`.

### 7.10 Closed-loop AI plan (planLoop.js)

The loop is generate → commit → monitor → recommend → apply. The maths is deterministic; only the narration uses AI. Records live in `planLoops.json`.

#### 7.10.1 Helpers
- `clamp(v, lo, hi)`; `num` returns a finite number or 0.
- `money(n)`: `|n| ≥ 1e6` → `x.xM`; `≥ 1e3` → `round(n/1e3)K`; else rounded. The prefix is `$`, or `-$` for negatives.
- `quarterRange('Qn YYYY')` gives the UTC first and last day of the quarter (`Date.UTC(y, startMonth+3, 0)`). An unparseable value uses the current UTC quarter.

#### 7.10.2 Channel intelligence
Built from the **static `mockData.campaigns`** (8 Nova campaigns), aggregated by channel:
```
spend = Σ spent, leads = Σ leads, revenue = Σ revenue
leadsPerDollar = leads/spend, revPerDollar = revenue/spend, cpl = spend/leads,
efficiency = +(revenue/spend).toFixed(3)
```

| Channel | Spend | Leads | Revenue | Leads per $ | Revenue per $ | CPL |
|---|---|---|---|---|---|---|
| Organic / SEO | 41,000 | 1,350 | 300,000 | 0.03293 | 7.317 | 30.37 |
| Events & Webinars | 142,000 | 1,850 | 560,000 | 0.01303 | 3.944 | 76.76 |
| Paid Search | 488,000 | 4,550 | 1,560,000 | 0.00932 | 3.197 | 107.25 |
| Paid Social | 78,000 | 720 | 360,000 | 0.00923 | 4.615 | 108.33 |
| Content Syndication | 372,000 | 1,230 | 1,975,000 | 0.00331 | 5.309 | 302.44 |

The table is in leads-per-dollar order, the order generation ranks channels in. Email has no campaign rows, so it is absent.

#### 7.10.3 Generate (preview, not persisted)
`generatePlan({name='AI marketing plan', goalLeads=3000, budget=400000, quarter='Q4 2026', maxChannels=5})`:
```
goalLeads = max(1, round); budget = max(1000, round)
intel = channels with leadsPerDollar > 0, sorted desc by leadsPerDollar
chosen = intel.slice(0, clamp(maxChannels, 2, intel.length))
weights_i = (leadsPerDollar_i × 1000)^1.25
share_i  = clamp(w_i/Σw, 0.08, 0.40);  then renormalize shares to sum 1   (so final can exceed 40% / fall below 8%)
amount_i = round(budget × share_i / 1000) × 1000
leads_i  = round(amount_i × leadsPerDollar_i); revenue_i = round(amount_i × revPerDollar_i)
start_i  = quarterStart + 10·i days (UTC); end = quarterEnd
tactic_i = {title:`${channel} — ${name}`, channel, type:'Campaign',
            priority: i==0 ? 'P0' : i<3 ? 'P1' : 'P2', quarter, budget: amount_i,
            startDate, endDate, goal:`${leads_i.toLocaleString()} leads`,
            cpl: round(cpl), efficiency, targets:{leads, revenue}}
projection = {leads Σ, revenue Σ, attainmentPct = round(leads/goalLeads×100),
              gapLeads = goalLeads − leads, onTrack = attainmentPct ≥ 100,
              blendedCpl = leads ? round(budget/leads) : 0}     // uses requested budget, not Σ amounts
rationale = "Concentrating $400K on the 5 most lead-efficient channels (A, B, C) to pursue 3,000 leads in Q4 2026. Projected N leads (P% of goal) at a blended $X cost per lead — grounded in your real channel performance."
returns {name, quarter, goalLeads, budget, channels: intel, tactics, projection, rationale}
```
- **QUIRK:** the code comments say "^1.4" and "8%–45%", but the code uses exponent 1.25 and a 0.08–0.40 clamp.
- **Worked default:**
  - Weights 78.9, 24.8, 16.3, 16.1, 4.46; final shares 45.1%, 19.9%, 13.1%, 12.9%, 9.0%.
  - Amounts: Organic $180K (5,927 leads), Events $79K (1,029), Paid Search $52K (485), Paid Social $52K (480), Content Syndication $36K (119). Σ $399K.
  - Projection: 8,040 leads, 268% attainment, blended CPL $50.
  - Starts 10-01, 10-11, 10-21, 10-31, 11-10; all end 2026-12-31.
- **`narrate(plan)`:**
  - Lazily imports `assistant.ask`.
  - Prompt: "In two concise sentences (no markdown, no headers), explain the strategy behind this marketing plan and how confident we should be. Goal: … Budget is concentrated on … top 3 '<channel> (~N leads at $CPL CPL)' … Projected …"
  - Raced against a 9 s timeout. The reply is trimmed and sliced to 420 characters; any failure gives null.
  - The route `POST /api/planloop/generate` (Editor) replaces `rationale` when narration succeeds.

#### 7.10.4 Commit
`commitPlan({name, quarter, goalLeads, budget, tactics})` (`POST /api/planloop/commit`, Editor, audit `ai.plan_commit`):
- Empty tactics → `'No tactics to commit.'` (400).
- `planId` = the **first plan node** (null if none).
- Campaign name = `name.slice(0,76)`, made unique among campaigns of that plan (case-insensitive) with ` (2)`, ` (3)`, …
- Creates:
  - campaign: owner `'AI Copilot'`, description `` `Closed-loop plan · goal ${goalLeads} leads on ${money(budget)} (${quarter})` ``
  - program `` `${name.slice(0,60)} — execution` `` (owner AI Copilot)
  - These skip validation.
- `loopId = uid('loop')`.
- Each tactic is created via untrusted `createTactic`, so approval is `Created`:
  - `title` ≤140 (default `` `${channel} tactic` ``), `type || 'Campaign'`, `owner || 'AI Copilot'`, `quarter`, `priority || 'P1'`, **`status 'planned'`**.
  - `budget`, `channel`, `goal || `${leads} leads``.
  - `startDate`, `endDate`, and `dueDate = endDate`.
  - `programId`, `loopId`, `targets {leads, revenue}`.
  - `performance {plan: {leads, revenue}, actual: {}}`.
  - `source {provider:'ai-plan', name}` (no connectionId).
- Loop record: `{id, name, quarter, goalLeads, budget, planId, campaignId, programId, tacticIds, createdAt}`.
- Returns `{loopId, campaignId, programId, tacticIds}`.

#### 7.10.5 List
`listLoops()` (`GET /api/planloop/list`): annotates `liveTactics` (tactics with that `loopId`), drops loops with 0 live tactics, and sorts by `createdAt` descending.

#### 7.10.6 Monitor
`loopStatus(loopId)` (`GET /api/planloop/status?loopId`):
- The loop is the one with that id, or the newest live loop. Returns null if there is no loop or no tactics.
- Per tactic, with `today = new Date()` and dates parsed as `YYYY-MM-DDT00:00:00Z`:
  ```
  expectedPct = start&&end&&end>start ? clamp((today−start)/(end−start)×100, 0, 100)
              : (end && today > end) ? 100 : 0
  progress = clamp(round(progress), 0, 100)
  targetLeads = targets.leads || performance.plan.leads;  actualLeads = performance.actual.leads
  projectedLeads = expectedPct ≤ 0 ? targetLeads
                 : actualLeads > 0 ? round(actualLeads / (expectedPct/100))
                 : round(targetLeads × clamp(progress/expectedPct, 0, 1.2))
  projectedLeads = clamp(projectedLeads, 0, round(targetLeads × 1.5))
  committed = Σ lineItems.amount || round(budget × progress/100)
  overdue = end && today > end && progress < 100
  health = expectedPct ≤ 0 → 'planned'; overdue → 'overdue';
           progress ≥ expected−5 → 'on-track'; ≥ expected−20 → 'at-risk'; else 'behind'
  row = {id, title, channel, priority, budget, committed, progress, expectedPct (rounded), targetLeads, projectedLeads, health, overdue, endDate}
  ```
- Rollup:
  ```
  {goalLeads, targetLeads Σ, projectedLeads Σ,
   attainmentPct = round(projected/goalLeads×100),
   status = ≥95 'on-track' | ≥80 'at-risk' | else 'off-track',
   plannedBudget Σ, committed Σ, budgetPacePct = round(committed/plannedBudget×100),
   atRiskCount = #behind|overdue, tacticCount}
  ```

#### 7.10.7 Recommend
`recommendations(loopId)` (`GET /api/planloop/recommendations`). With no status it returns `{loopId||null, recommendations:[]}`.
- **Reallocate** (at most one):
  - Considers rows with `budget > 0`, with `eff = targetLeads/budget`, sorted descending; `best` is first and `worst` is last.
  - Fires if `best.eff > worst.eff × 1.25`, `worst.budget > 5000`, and they are different rows.
  - `amount = max(1000, round(min(worst.budget×0.3, best.budget×0.5)/1000)×1000)`.
  - `liftLeads = round(amount × (best.eff − worst.eff))`.
  - Output:
    ```
    {id:`realloc-${worst}-${best}`, type:'reallocate',
     title:`Shift ${money(amount)} from ${worst.channel} → ${best.channel}`,
     detail:`${best.channel} converts budget to leads ~${(best.eff/(worst.eff||1)).toFixed(1)}× more efficiently than ${worst.channel} in this plan.`,
     impact:`+${lift} projected leads`,
     apply:{kind:'reallocate', fromId, toId, amount}}
    ```
- **Expedite** (each `behind` or `overdue` row):
  ```
  {id:`expedite-${id}`, type:'expedite',
   title:`Expedite “${title}”`,
   detail:`${progress}% complete vs ${expectedPct}% expected by today${overdue?' · past due':''}.`,
   impact:`Protects ${targetLeads} target leads`,
   apply: priority==='P0' ? null : {kind:'raise-priority', id}}
  ```
- **Overspend** (`committed > budget × 1.05` and `budget > 0`):
  ```
  {id:`overspend-${id}`, type:'overspend',
   title:`Review spend on “${title}”`,
   detail:`Committed ${money(committed)} exceeds the ${money(budget)} plan.`,
   impact:'Budget risk', apply:null}
  ```

#### 7.10.8 Apply
`applyRecommendation(loopId, apply)` (`POST /api/planloop/apply {loopId, apply}`). `loopId` is unused.
- `reallocate`:
  - `amt = clamp(round(amount), 0, from.budget)`.
  - `fromEff = from.targets.leads/(from.budget||1)`; `toEff` likewise.
  - `from` gets `budget −= amt`, `targets.leads = max(0, round(leads − amt×fromEff))`, revenue unchanged.
  - `to` gets `budget += amt`, `targets.leads = round(leads + amt×toEff)`.
  - Returns `{ok:true, moved: amt}`.
  - These go through the untrusted `updateTactic`, so a Submitted or Approved tactic **resets to Created** under `requirePlannedCost`.
- `raise-priority`: P2 → P1, anything else → P0. Returns `{ok:true, priority}`.
- Errors are returned **inside** a 200 `ok:true` envelope as `{ok:false, error}`: `'No action.'`, `'Tactic not found.'`, `'Nothing to move.'`, `'Unknown action.'`

### 7.11 Route index (planning area)

| Method & path | Role | Notes |
|---|---|---|
| GET `/api/plans/levels` | signed in | LEVELS |
| GET `/api/plans/tree` | signed in | tree with roll-ups |
| GET `/api/plans/nodes`, `/api/plans/nodes/:id` | signed in | 404 `'Node not found'` |
| GET `/api/plans/meta` | signed in | levels + tactic/lineItem, models, stages, mechanisms, segments, approvalStatuses, teams `{id,name,color,memberIds,leadId}`, currentYear |
| GET/POST/PUT `/api/plans/models[/:id]` | read / Admin | |
| POST `/api/plans/nodes` | Editor | validateNode → createNode(comments []) ; audit `plan.create`; notify owner "You’re the owner of a <level>" |
| PUT `/api/plans/nodes/:id` | Editor | strips comments; notify if ownerId changed; audit `plan.update` |
| POST `/api/plans/nodes/:id/comments` | Editor | audit `plan.comment` |
| DELETE `/api/plans/nodes/:id` | **Admin** | recycle bin cascade; audit `plan.delete` |
| GET/PUT/POST `/api/planning/board-config…` | read / planner | columns, fields, template |
| GET `/api/planning/plans`, `/api/planning/plans/:id` | signed in | SP&A/BCPIR register and detail |
| POST `/api/planning/plans/:cid/{submit,approve,reject}` | Editor + guard | |
| GET `/api/planning/board`, `/api/planning/summary` | signed in | filters planId/campaignId/programId/planIds |
| GET `/api/planning/calendar.ics` | **public** | |
| GET `/api/tactics/:id` | signed in | decorated + missingForSubmit |
| POST `/api/tactics`, PUT `/api/tactics/:id` | Editor | |
| POST `/api/tactics/:id/{submit,approve,decline}` | Editor + guard | |
| POST/PUT/DELETE `/api/tactics/:id/line-items[/:liId]` | Editor | audits `lineitem.*` |
| POST `/api/planning/tactics/{bulk-move,bulk-copy,bulk-submit,bulk-delete}` | Editor | |
| DELETE `/api/tactics/:id` | Editor | recycle bin |
| POST `/api/tactics/:id/{comments,media-codes}` | Editor | |
| `/api/planning/initiatives…` | Editor | legacy aliases |
| GET `/api/import/sample`; POST `/api/import/tactics` | Editor | |
| POST `/api/import/{trello/boards,trello,jira/projects,jira}`; GET/DELETE `/api/import/connections[/:id]` | Editor | |
| `/api/integrations/google-calendar/{status,sync,clear,disconnect}` | status: signed in; others Editor | |
| `/api/planloop/{generate,commit,list,status,recommendations,apply}` | generate/commit/apply Editor | |

**Audit actions emitted:** `plan.create`, `plan.update`, `plan.comment`, `plan.delete`, `plan.submit`, `plan.approve`, `plan.reject`, `model.create`, `model.update`, `tactic.create`, `tactic.update`, `tactic.submit`, `tactic.approve`, `tactic.decline`, `tactic.bulk_move`, `tactic.bulk_copy`, `tactic.bulk_submit`, `tactic.delete`, `tactic.bulk_delete`, `tactic.comment`, `lineitem.create`, `lineitem.update`, `lineitem.delete`, `import.tactics`, `import.trello`, `import.jira`, `ai.plan_commit`.

Not audited: media codes, board-config changes, connection disconnect, plan-loop apply, Google Calendar actions, legacy initiative routes.

### 7.12 Quirks a rebuild team should decide on deliberately

1. Comment caps differ: nodes keep the last 200, tactics the first 200.
2. Campaign `status` and `approval` can be set through a plain node PUT. Only tactics protect their approval fields.
3. Campaign plan approve and reject have no state-machine check, and submit wipes earlier rejection details.
4. `endDate` and `dueDate` can diverge on partial updates.
5. The fiscal-year window check makes any PUT that includes `fiscalYear` fail once the calendar year passes the plan's FY.
6. Tactic type is not checked when `type` is omitted on create, or when a tactic moves between plans.
7. `blockNegativeBalance` is enforced only on line-item operations, not when the tactic budget is lowered.
8. CSV, Trello, Jira and AI-commit imports bypass `validateTactic`, so titles inside a program can repeat.
9. Board disconnect hard-deletes and bypasses the recycle bin.
10. Google Calendar sync never removes events for deleted tactics, and `GOOGLE_CREDENTIALS_JSON` is treated as a file path.
11. ICS line folding counts characters, not octets.
12. planLoop's comments disagree with its code (^1.4 vs 1.25; 45% vs 40% cap), and renormalization lets the top share exceed the cap. The channel intel uses static mock campaigns.
13. The summary sorts quarters lexicographically.
14. A plan-loop apply error comes back as HTTP 200 with `data.ok:false`.
15. The audit label for Trello and Jira imports never shows the real board or project name.
16. `applyLineage` silently clears all hierarchy ids if `programId` is dangling.
17. Campaign-plan figures are seeded by random tactic ids, so they differ per install while staying stable per id.

## 8. Planning — user interface

**Source files:** `client/src/pages/Planning.jsx`, `client/src/components/planning/*`, `client/src/components/GoogleCalendarSync.jsx`, `client/src/lib/exporters.js`, `client/src/lib/files.js`.

**Route:** `/planning`.

**Not currently mounted:** `PlanCopilot.jsx` (the former "AI Copilot" tab) and `PlanGrid.jsx` (the former flat grid, now replaced by the five-level `HierarchyGrid`). Both are still specified below so the team can decide whether to bring them back.

### 8.0 Shared conventions used throughout Planning

#### 8.0.1 Roles and permission flags
- `user.role` is one of `Owner | Admin | Editor | Viewer` (workspace role).
- `user.orgRole` is one of `admin | manager | lead | member` (org-chart role).
- When there is no user (anonymous or demo bootstrap), most checks default to permissive.

| Flag | Where | Rule | What it gates |
|---|---|---|---|
| `canEdit` | Planning.jsx, PlansView | `!user \|\| user.role !== 'Viewer'` | All create, edit and delete buttons, drag-and-drop, the Honeycomb button, Import, New tactic, the plan register's actions |
| `canCustomize` | Planning.jsx | `user && (orgRole ∈ {lead, manager} \|\| role ∈ {Owner, Admin})` | Board "Group by", the saved "Views" menu, "Board settings", card checkboxes and the board bulk bar. Note that `orgRole: 'admin'` does NOT qualify unless the user is also a workspace Owner or Admin. The server re-checks with `requirePlanner`. |
| `canAdmin` | Planning.jsx | `!user \|\| role ∈ {Owner, Admin}` | Deleting a plan, campaign or program (with everything under it) from the Grid and the Inspector. The server enforces this too. |
| `isAdmin` | NodeInspector | `!user \|\| role ∈ {Admin, Owner}` | The "Model settings" button on a plan |
| `readOnly` | TacticDrawer | `user && role === 'Viewer'` | Every editing control in the drawer |
| `canApprove` (tactic) | TacticDrawer | `!user` OR, if a named approver exists, match on `approverId === user.id`, falling back to `approver === user.name`. With no named approver: `role ∈ {Admin, Owner}` or `orgRole ∈ {lead, manager, admin}`. | Approve / Decline buttons |
| `canApprove` (campaign plan) | PlansView CampaignBlock | If an approver is assigned: match on id, else on name. With no approver assigned, anyone with `canEdit`. | Approve / Reject buttons in the register |

#### 8.0.2 Toasts
- Material UI `Snackbar` with `autoHideDuration=3000`, anchored bottom-centre, containing a filled `Alert` with `borderRadius: 3`.
- Toast state is `{ severity: 'success'|'error'|'warning', msg }`.
- Planning.jsx has one Snackbar and PlansView has its own. CampaignPlanDetail sends its toasts through PlansView's `onToast`.

#### 8.0.3 Loading and error (`StateWrap`)
- **Loading:** a centred `CircularProgress size=28` with `minHeight 240`.
- **Error:** an outlined error Alert reading: "Could not load data: {error.message}. Is the API running on port 4000?"
- **Otherwise:** the children render.

#### 8.0.4 Formatting (`lib/format.js`)
- **`fmtMoney(usd)`:** converts to the active currency (`ACTIVE.rate`) and compacts. ≥1M → `$1.23M`; ≥1K → `$12.3K`; otherwise rounded (e.g. `$950`). Negatives are prefixed with `-`.
- **`fmtMoneyFull`:** `Intl.NumberFormat` in currency style with 0 decimals.
- **`fmtNum`:** compact K/M formatting.
- **`fmtNumFull`:** en-US grouping.
- **`fmtPct(n)`:** `n.toFixed(1)%`.
- **`activeCurrency()`:** returns `{code, rate, symbol}`.
- **`CURRENCY_SYMBOLS`:** USD $, EUR €, GBP £, CAD C$, AUD A$, INR ₹, JPY ¥, SGD S$.
- PageHeader shows the app's `CurrencyControl` on the right by default (`currency=true`).

#### 8.0.5 Palette (`theme.js`)
- **`SERIES`:** `#2a78d6` blue, `#1baf7a` aqua, `#e87ba4` magenta, `#eda100` yellow, `#4a3aa7` violet, `#eb6834` orange, `#008300` green, `#e34948` red.
- **`STATUS`:** good `#0ca30c`, warning `#fab219`, serious `#ec835a`, critical `#d03b3b`.
- **Tactic type colours** (board and calendar): Campaign = SERIES[0], Launch = SERIES[2], Content = SERIES[1], Event = SERIES[4], Research = SERIES[5], Ops = SERIES[6].
- **Priority colours:** P0 = `#d03b3b`, P1 = `#c77700`, P2 = `#0071e3`.
- **SP&A colour** = SERIES[0] (blue). **BCPIR colour** = SERIES[4] (violet). **"Planned" bars** = `#9aa0aa`.

#### 8.0.6 `planLevels.js`: shared vocabulary and client-side maths
- **`LEVEL_LABEL`:** plan Plan, campaign Campaign, program Program, tactic Tactic, lineItem "Line item".
- **`CHILD_LEVEL`:** plan→campaign→program→tactic→lineItem.
- **`LEVEL_COLOR`:** plan SERIES[0], campaign SERIES[4], program SERIES[1], tactic SERIES[2], lineItem `#86868b`.
- **`LIFECYCLE_COLOR`:** Created `#86868b`, Submitted `#c77700`, Declined critical, Approved `#0071e3`, In-Progress good, Complete `#5b6b7a`.
- **`MECHANISM_LABEL`:** C Custom, F "First period", L "Last period", E "Evenly distributed".
- **`SEGMENT_LABEL`:** month Monthly, quarter Quarterly.
- **`periodLabel(key, short)`:** `"2026-03"` → "Mar 2026" (short form "Mar"); `"2026-Q2"` → "Q2 2026" (short form "Q2"); anything else is returned as-is, and empty becomes "—".
- **`quarterOfMonth("2026-05")`** → `"2026-Q2"`.
- **`periodsFor(tactic, segment)`:**
  - Start month comes from `startDate || endDate || dueDate`.
  - End month comes from `endDate || dueDate || startDate`. It is ignored when the tactic is evergreen, and when it is earlier than the start month the end is set to the start.
  - Lists every month in between, capped at 132 iterations.
  - Quarter segment de-duplicates the months into quarters.
  - Mirrors the server's `planning.js periodsFor`.
- **`spread(amount, mech, periods)`:**
  - The total is `max(0, round(amount))`.
  - F puts the whole total in the first period; L puts it in the last.
  - Any other mechanism splits it evenly: `floor(total/n)` in each period, with the remainder added to the last period.
- **`yearPeriods(year, segment)`:** the 12 months or 4 quarters of a year.
- **`allocationsIn(li, segment)`:**
  - Re-keys a line item's allocations into the requested segment.
  - Months roll up into quarters.
  - A quarter splits evenly into its 3 months using `spread(...,'E')`.
- **`findNode(tree, id)`:** recursive search.
- **`lineagePath(tree, {planId, campaignId, programId})`:** builds `"Plan › Campaign › Program"`.
- **`programOptions(tree)`:**
  - Returns a flat list of `{value: programId, label, planId}`.
  - The label is `"Campaign › Program"`, prefixed with `"Plan › "` when there is more than one plan.
  - Used by every program picker.

#### 8.0.7 Glossary tooltips (`InfoTip`, shown as a muted (i) icon)
The texts used in Planning, verbatim:
- `tactic`: "The smallest unit of work you actually execute — a specific campaign, launch, event or piece of content, sitting at the bottom of the plan hierarchy."
- `inFlight`: "Tactics currently being worked on (status "in progress") — what's live right now."
- `launched`: "Tactics that have shipped or gone live — completed and out in the world."
- `priorityP0`: "How important a tactic is: P0 = must-do / top priority, P1 = important, P2 = nice-to-have."
- `plannedBudget`: "The total budget set aside across all the tactics shown."
- `progress`: "How far along a tactic is, from 0% (not started) to 100% (done)."
- `hierarchy`: "How marketing work is organized — five levels, created in order: Plan → Campaign → Program → Tactic → Line item. The plan is one fiscal year; tactics are the things you do; line items are where the money is planned."
- `plannedCost`: "What a tactic is expected to cost. Tactics have planned cost, never a budget — the line items underneath carry the detail."
- `evergreenTactic`: "An ongoing tactic with no end date. Once saved as evergreen it can't be undone — close it and create a new tactic with real dates instead."
- `sysGenBalance`: "System Generated Balance — the tactic's planned cost minus the sum of its line items. If you use line items you want this to be zero."
- `planBudgetField`: "A simple marketing-only budget on a plan, campaign or program. Children's budgets are deducted from the parent's; what is left is unallocated."
- `dateModeAuto`: "Auto: the dates are calculated from the tactics underneath. Manual: you type them in."
- `strategicPlan`: "A high-level Strategic / AOP (annual operating plan). Shown in grey and can be filtered separately from working plans."
- `unallocatedBudget`: "This level's plan budget minus the budgets handed down to its children."
- `approvalLifecycle`: "Created → Submitted → Approved or Declined. An approved tactic reads In-Progress once its start date arrives and Complete once its end date passes."
- `spreadMechanism`: "How a line item's total is spread over time: C = Custom (you type each period), F = First period, L = Last period, E = Evenly across the tactic's duration."
- `planLevel`: "One fiscal year of work aimed at a goal. Holds the budget, the goals, the teams and the model. Plan names are unique across the whole system."
- `campaignLevel`: "A theme or overarching message inside a plan — usually runs most of the year."
- `programLevel`: "A group of similar activities inside a campaign (e.g. Social Media, Demand Creation). Its dates must sit inside the campaign's and its budget can't exceed the campaign's."
- `planModel`: "The rulebook a plan is measured by: waterfall conversion rates, average deal size, which tactic types are allowed, and line-item money rules."
- `waterfallGoals`: "Enter the inquiries a tactic should generate; the plan's model cascades them through its conversion rates (MQL → TQL → opportunity → won) and prices won deals at its average deal size."
- `tacticLineItem`: "A cost line under a tactic — the only place spend is planned and tracked. Its total is spread across months or quarters and can be linked to a budget."

### 8.1 Planning page shell (`pages/Planning.jsx`)

#### 8.1.1 Page header (PageHeader)
- eyebrow: "Planning & work management"
- title: "Run marketing"
- highlight (secondary colour): "like a practice."
- subtitle: "Every campaign runs paired SP&A (spend) and BCPIR (ROI) plans. Manage them in the register, run the work on the board, and track delivery on the calendar."
- The currency control sits on the right.

The body is `max-w-content px-5 pb-24`, wrapped in `StateWrap(loading, error)`.

#### 8.1.2 Data loading
**On mount, and every time `filter` changes**, the page sets `loading=true` and calls these in parallel:
1. `api.planTree()` → `GET /api/plans/tree`
2. `api.planningBoard(filter)` → `GET /api/planning/board?{filter}`
3. `api.planningSummary(filter)` → `GET /api/planning/summary?{filter}`
4. `api.boardConfig()` → `GET /api/planning/board-config`
5. `api.planMeta()` → `GET /api/plans/meta`. Failure is tolerated and becomes `null`.

**Behaviour:**
- Any failure in calls 1–4 sets `error`, so StateWrap shows the error alert.
- The view tabs live inside StateWrap, so every filter change replaces the whole body (tabs included) with a spinner for a moment.
- **`refresh()`** re-fetches tree, board and summary with no loading flag.
- **`refreshMeta()`** re-fetches `planMeta` and silently ignores failure.
- **`refreshConfig()`** re-fetches `boardConfig` and silently ignores failure.
- All requests carry `Authorization: Bearer <zr_token>`. Responses are `{ok, data, error}`; `ok:false` throws `Error(error)`.
- The query string is built by `qs()`, which drops null and empty values.

**`filter` (memoised):**
- If a hierarchy node is selected: `{ [`${level}Id`]: id }`, i.e. `planId`, `campaignId` or `programId`. The node takes precedence.
- Otherwise, if a single plan chip is active: `{ planId }`.
- Otherwise `{}`.

#### 8.1.3 View tabs (`ViewTabs`)
- A Material UI `ToggleButtonGroup`, small and exclusive, with pill buttons (`borderRadius 999, px 2`) that wrap.
- It sits in a fixed spot (`mb-5`) for every view.
- Order and labels: **Plans** (`plans`), **Grid** (`grid`), **Board** (`board`), **Calendar** (`calendar`), **Budget & Cost** (`budget`). Clicking the active tab does nothing (`v && setView(v)`).
- Persisted to `localStorage.zr_planning_view`. On load, an invalid or missing value falls back to `'plans'`.

**What each view shows:**

| View | Summary strip | Plan selector chips | Hierarchy sidebar | "Showing:" toolbar row | Content |
|---|---|---|---|---|---|
| plans | no | no | no | no | `<PlansView/>`, self-contained, fetches its own data |
| grid | yes | yes (if >1 plan) | no | no | `<HierarchyGrid tree={visibleTree}/>` |
| board | yes | yes | yes | yes | Kanban (`BoardColumns`) |
| calendar | yes | yes | yes | yes | `<CalendarView/>` |
| budget | yes | yes | no | no | `<BudgetCostView key={activePlanId\|\|'all'} tree={visibleTree}/>`. The key remounts it on plan switch. |

- Constants: `EDITABLE_VIEWS = ['board','calendar']` decides where "New tactic" appears. `FULL_WIDTH_VIEWS = ['plans']`. `HIERARCHY_VIEWS = ['grid','budget']` (these views skip the sidebar).
- The non-Plans views render only once both `board` and `summary` are loaded.

#### 8.1.4 Summary strip (non-Plans views)
- A grid of six cards: 2 columns, then 3 at `sm`, then 6 at `lg`.
- Each `Stat` card has a label (12.5px, ink2, with InfoTip) and a value (24px, extra-bold, tabular, coloured by its accent).
- Data comes from `GET /api/planning/summary`.

| Label | Field | Accent | Tip |
|---|---|---|---|
| Tactics | `total` | ink | G.tactic |
| In flight | `inFlight` | SERIES[0] | G.inFlight |
| Launched | `launched` | STATUS.good | G.launched |
| P0 priorities | `p0` | STATUS.critical | G.priorityP0 |
| Planned budget | `fmtMoney(plannedBudget)` | ink | G.plannedBudget |
| Avg. progress | `${avgProgress}%` | SERIES[4] | G.progress |

The summary follows `filter`, so a hierarchy node picked on the Board still scopes the summary after switching to Grid or Budget, even though those views show the whole plan.

#### 8.1.5 Plan selector (`PlanSelector`), modelled on YouTube's topic chips
- **When shown:** only when `tree.length > 1`, and only in non-Plans views.
- **Selection:** single-select, like radio buttons.
  - "All plans" pill (blue when active, ghost otherwise), then one pill per plan.
  - Each plan pill has a 10px colour dot, the plan name, and `title="Show {name}"`.
- **Plan colours:** `planColor[planId] = SERIES[index % 8]`, by order in the tree.
- **Scrolling:**
  - The row scrolls horizontally with no visible scrollbar.
  - Circular chevron buttons (32px, white with shadow) appear only when content overflows on that side (`scrollLeft > 2`, or right edge more than 2px from the end).
  - They scroll by ±260px smoothly and recompute on scroll and on window resize.
- **State:** `selectedPlanIds` holds `'all'` or a single plan id (the plural name is a misnomer).
  - Read synchronously from `localStorage.zr_planning_plans` so there is no flicker on reload.
  - Written back on every change.
  - Once the tree loads, a stored id that no longer exists resets to `'all'`.
- **Effects of choosing a chip:**
  - Clears any hierarchy-node drill-down.
  - `activePlanId` becomes that id (or `null` for All).
  - `visibleTree` is that one plan (or every plan).
  - `colorByPlan = !activePlanId && tree.length > 1`. In "All" mode with several plans, board cards get a 3px left border in their plan's colour, and calendar chips use the plan colour instead of the type colour.

#### 8.1.6 Hierarchy sidebar (Board and Calendar only)
- **Layout:** two columns on `lg`, `minmax(0,300px)` for the sidebar and `minmax(0,1fr)` for the main area. Stacked below that.
- **Header:** "HIERARCHY" (12px uppercase) with an InfoTip (G.hierarchy). If `canEdit`, a small "+" IconButton (`title="Add plan"`) opens the Inspector in add-plan mode.
- **Body:** `HierarchyTree` (§8.8) showing `visibleTree`, with `selectedId = selectedNode?.id`.
- **Selecting a node:** finds it in the full tree and sets `selectedNode = {id, level}`. Selecting null ("All work") clears it.

#### 8.1.7 Main-column toolbar (Board and Calendar)
- **Left:** "Showing: **{selectedLabel}**". The label is "All work" when nothing is selected, otherwise the node's name, falling back to "Selection".
- **Right**, in this order, each control only where its condition holds:
  1. **Group by** select. Board only, `canCustomize`. minWidth 132, pill shape. Options: No grouping (`none`), Owner, Campaign, Priority, Type. Persisted to `localStorage.zr_planning_groupby`, default `none`.
  2. **Views** button. `canCustomize`, Board and Calendar. Outlined, bookmark icon, label `Views` or `Views · N`. Opens the saved-views menu (§8.1.8).
  3. **Board settings** button. Board only, `canCustomize`. Tune icon. Opens BoardSettingsDialog (§8.1.11).
  4. **Import from Jira / Trello**. Board only, `canEdit`. Cloud-download icon. Opens ImportBoardDialog (§8.12).
  5. **New tactic**. Contained, "+" icon. `canEdit`, Board or Calendar. Calls `openAdd()`.

#### 8.1.8 Saved views
**Storage:** a per-browser array in `localStorage.zr_planning_views`. Each entry is `{id: Date.now(), name (≤40 chars, trimmed), view, groupBy, nodeId, nodeLevel}`.

**The menu** (right-aligned, minWidth 260):
- "Save current view…" with a bookmark-add icon.
- A divider.
- When there are no views, a disabled "No saved views yet".
- Otherwise each saved view on a row:
  - A blue check if the view is active, or a 16px spacer.
  - The name (13.5px).
  - A sub-label: `"{View label}"`, plus `" · by {groupBy}"` when grouping is set.
  - A trash IconButton that deletes it (the click does not bubble to the row).

**Behaviour:**
- **Save:**
  - Uses `window.prompt('Name this view:')`; a blank name cancels.
  - Names are unique without regard to case, so saving an existing name replaces that view.
  - Toast: `View "X" saved.`
- **Apply:**
  - Sets `groupBy`, then `view` if it is set.
  - Restores `selectedNode` only if the node still exists in the tree; otherwise clears it.
  - Toast: `View "X" applied.`
- **Delete:** toast "View deleted."
- **Active** means view, groupBy and nodeId all match the current state.
- Saved views do **not** capture the plan-chip selection.

#### 8.1.9 Board (Kanban)
**Data:** `GET /planning/board` returns:
- `columns: [{id, name, color, wip, hint, initiatives: [tactic…]}]`
- `meta: {types[], priorities[]}`

`allInitiatives` is every column's initiatives flattened together.

**Swimlanes.** When `groupBy ≠ none`:
- Tactics are bucketed by: owner (or "Unassigned"), priority, type, or campaign name (looked up from the tree, else "No campaign").
- Lanes are sorted alphabetically.
- Each lane header shows the label (13px semibold), a count pill, and a hairline rule.
- Under each lane is a full `BoardColumns` row.

**`BoardColumns`:**
- **Bucketing:** tactics are bucketed by `status`. A status that matches no column falls into the first column.
- **Layout:**
  - One row that never wraps: `grid-template-columns: repeat(n, minmax(210px, 1fr))`.
  - Wrapped in a horizontal-scroll `.board-scroll` container with an edge fade.
- **Column panel:** rounded, `var(--gray-bg)` background.
- **Column header:**
  - A colour dot (`col.color` or `#86868b`) and the column name.
  - A count pill showing `N` or `N / WIP`. It turns red (`rgba(208,59,59,.15)` background, `#d03b3b` text) when `wip > 0 && count > wip`, with `title="WIP limit X"`.
  - A "+" IconButton (`canEdit`) that calls `openAdd(col.id)`: a new tactic pre-set to that column's status.
- **Empty column:** centred text showing `col.hint` or "—".

**Card (`InitiativeCard`):**
- **Card states:**
  - The whole card is clickable and opens the TacticDrawer.
  - Selected cards get a 2px blue ring.
  - While dragging, the card is at 40% opacity.
  - In plan-overlay mode, the card gets a left border `3px solid planColor`.
- **Content, top to bottom:**
  1. **Top row:**
     - A checkbox (only when `canCustomize`; the click doesn't open the drawer). It toggles board selection.
     - A type chip (uppercase, tinted with the type colour).
     - On the right, a priority chip (tinted with the priority colour).
     - If `canEdit`, hover-revealed Edit (pencil) and Delete (trash) icons. Delete calls `remove(id)`, which is `DELETE /api/tactics/:id`, **with no confirmation**, then toasts "Tactic removed." and refreshes.
  2. Title (14px semibold).
  3. Goal, if set (12px).
  4. Up to two custom-field chips, `"Label: value"`, for fields that have a value.
  5. Progress bar (6px high, gradient `#2a78d6 → #9f4bd6`).
  6. Owner on the left, quarter on the right (11.5px).
  7. Channel on the left; `fmtMoney(budget)` on the right when budget > 0.

**Drag and drop** (native HTML5):
- Cards are draggable only when `canEdit`. `onDragStart` sets `draggedId`.
- A column handles `dragOver` with `preventDefault`. The hover highlight key is `lane::colId`, shown as `rgba(0,113,227,0.07)`.
- **On drop:**
  - If the tactic is already in that status, nothing happens.
  - Otherwise there is an optimistic move: the card is removed from all columns and placed at the **top** of the target column with the new status.
  - Then `PUT /api/tactics/:id {status: colId}` is sent, followed by re-fetching only the summary.
  - On error: an error toast and a full `refresh()`.
- Dropping into a different swimlane changes only the status, never the owner, priority or other grouping attribute.

**Board bulk-action bar** (Board, `canCustomize`, at least one selected):
- A rounded bar showing "{N} selected".
- A status select listing the column names. It displays `boardBulkStatus || columnDefs[0].id`.
- **"Move to column":**
  - Calls `bulkUpdate(ids, {status})`, which sends one `PUT /tactics/:id` per id in parallel.
  - Then refreshes and toasts "Updated {N} tactics."
  - **Bug:** `boardBulkStatus` starts as `''` and `applyBoardBulkStatus` returns early when it is empty, so the button does nothing until the user actually changes the select.
- **"Delete"** (red outlined) → `askBulkDelete(ids, clearBoardSel)` → bulk-delete confirmation (§8.1.13).
- **"Clear"**.

#### 8.1.10 Tactic editor dialog (create and edit)
**Opened from:**
- Board "New tactic" and column "+"
- Calendar "New tactic"
- The card pencil
- The drawer's "Edit all fields"
- The Grid's "+" on a program row
- The Tree's "+" on a program

**`openAdd(status='backlog', programId=null)` picks the program in this order:**
1. The explicit `programId` argument.
2. Else the selected node, if it is a program.
3. Else the first program belonging to the active plan (or the first program overall).

**Draft defaults** (`emptyInitiative`):

```
title '', type 'Campaign', owner '', ownerId '', quarter 'Q3 2026', priority 'P1', status 'backlog',
progress 0, budget 0, channel '', goal '', impact 3, effort 3, dueDate '', startDate '', endDate '',
description '', tags [], programId null, customAttrs {}, externalName '', evergreen false
```

**`openEdit(it)`** merges the tactic over these defaults.

**Dialog:** `maxWidth sm`. Title "New tactic" or "Edit tactic". The body is a 2-column grid:
- **Title.** Full width.
- **Type.** Select from `board.meta.types`.
- **Priority.** Select from `board.meta.priorities`.
- **Owner.** PeoplePicker; stores both `owner` (name) and `ownerId`.
- **Quarter.** Free text.
- **Status.** Select of the board columns (`{value: col.id, label: col.name}`).
- **Program.** Required select; shown in error-red when empty.
  - Options come from `programOptions(tree)`.
  - If there are none, a disabled "*Create a program first*".
- **External name.** helperText "Name used in connected systems (optional)".
- **Channel.**
- **Planned cost ($).** Number field, with an InfoTip (G.plannedCost) as end adornment. Stored as `budget`.
- **Progress (%).** Number, min 0, max 100.
- **Impact.** Select 1–5.
- **Effort.** Select 1–5.
- **Start date.** Date input.
- **End / due date.** Date input.
  - Writes **both** `endDate` and `dueDate`.
  - Disabled and blank when evergreen, with helperText "Evergreen — no end date".
- **Evergreen checkbox.** Full width; label "Ongoing / evergreen tactic — can't be undone once saved" with an InfoTip (G.evergreenTactic). It is disabled when editing a tactic that is already evergreen.
- **Goal.** Full width.
- **Tags (comma-separated).** Full width. On save, split on commas, trimmed, empties dropped.
- **Description.** Multiline, at least 2 rows.
- **"CUSTOM FIELDS"** section, if `config.fields.length`:
  - A `select` field renders a Select with options "—" plus `f.options`.
  - Other types render a TextField of type number, date (with shrink label) or text.
  - Values are stored in `customAttrs[f.id]`.

**Actions:**
- Cancel.
- "Add tactic" or "Save". Disabled unless `title.trim()` and `programId` are both set.

**Save:**
- Edit sends `PUT /api/tactics/:id`; create sends `POST /api/tactics`.
- Then `refresh()`, close, and toast "Tactic updated." or "Tactic added."
- On error, the server message appears as an error toast and the dialog stays open.

#### 8.1.11 Board settings dialog (leads, managers, Owner and Admin)
**Dialog:** `maxWidth sm`, title "Board settings". A pill toggle switches between tabs **"Columns & WIP"** and **"Custom fields"**. Each time it opens, the tab resets to Columns and local copies of `config.columns` and `config.fields` are made.

**Columns tab:**
- **Templates row:**
  - Reads "Start from a template:" followed by one outlined button per `config.templates[]` (`{id, name}`).
  - Clicking one applies it **immediately**: `POST /api/planning/board-config/template {template: id}`.
  - The dialog closes; refresh and refreshConfig run; toast "Board template applied."
- **Column rows**, each in a bordered box:
  - Up and down arrows to reorder (disabled at the ends).
  - A colour select, 64px wide, rendering a swatch. Palette: `#86868b #0071e3 #c77700 #1baf7a #9f4bd6 #d03b3b #e34980 #eda100`.
  - A **Name** text field.
  - A **WIP** number field (min 0, clamped to ≥0).
  - A delete button, disabled when only one column is left.
- **"+ Add column"** adds `{id:'', name:'New stage', hint:'', color:'#86868b', wip:0}`.
- Helper text: "WIP = work-in-progress limit (0 = none). A column turns red when it exceeds its limit."

**Custom fields tab:**
- **Field rows:**
  - **Field label**.
  - A **Type** select: Text, Number, Date, Select.
  - A delete button.
  - For Select fields, an extra full-width "Options (comma-separated)" input, parsed into an array as the user types.
- **Empty state:** "No custom fields yet. Add fields like "Region", "Est. reach", or a "Stage gate" dropdown."
- **"+ Add field"** adds `{id:'', label:'', type:'text', options:[]}`.

**Actions:**
- Cancel.
- The primary button depends on the active tab, and **only that tab's edits are saved**:
  - **"Save columns"** → `PUT /api/planning/board-config/columns {columns}`, then refresh, refreshConfig, toast "Board columns updated."
  - **"Save fields"** → `PUT /api/planning/board-config/fields {fields}`, then refreshConfig, toast "Custom fields updated."
- Server errors appear as error toasts. The dialog closes immediately in either case.

#### 8.1.12 Node (plan, campaign, program) CRUD wiring
- `openAddTop()` opens the Inspector as `{mode:'add', level:'plan', parentId:null}`.
- `openAddChild(parent, level)`: when `level === 'tactic'` it opens the tactic editor for that program; otherwise the Inspector in add mode for that level and parent.
- `openEditNode(node)` opens the Inspector in edit mode with the live tree node.

**`onNodeSaved(saved, isAdd, extra)`:**
- When `extra.modelChanged`: refresh and refreshMeta, toast "Model updated — goals recalculate on each tactic's next save." The Inspector stays open.
- Otherwise: close the Inspector, refresh, and toast either `{Level} "{name}" created.` or `{Level} saved.`

**Cascading delete confirmation dialog:**
- `maxWidth xs`. Title: `Delete {level} "{name}"?`
- Body: "Deleting a {level} deletes everything under it — {N} tactic(s) and {M} line item(s). It all goes to the recycle bin together and can be restored (tactics come back as Created)." The count clause is shown only when `rollup` is present.
- Buttons: Cancel, and a red contained Delete that reads "Deleting…" while busy. Closing is blocked while deleting.
- The delete calls `DELETE /api/plans/nodes/:id`, which returns `{removedIds, removedTactics}`.
  - Closes both the confirm dialog and the Inspector.
  - Clears `selectedNode` if it was deleted.
  - Refreshes, then toasts `{Level} deleted[ with N tactic(s)] — restore it from the recycle bin.`

#### 8.1.13 Bulk tactic delete confirmation
`confirmDelete` is `{kind: 'selected'|'all', ids, count, clearSel}`.

**Selected:**
- Title: `Delete {N} tactic(s)?`
- Body: "This removes the {N} selected tactic(s) (with line items). It goes / They go to the recycle bin and can be restored."
- Calls `POST /api/planning/tactics/bulk-delete {ids}`.

**All** (via `askDeleteAll`, which is now unused):
- Title: "Delete all tactics?"
- Body: "This removes all {N} tactics (with their line items). They go to the recycle bin and can be restored."
- Calls `POST /api/planning/tactics/bulk-delete {all: true}`.

**Buttons:** Cancel, and red "Delete" (or "Delete all"), which reads "Deleting…" while busy.

**On success:** calls `clearSel`, closes the drawer and dialog, refreshes, and toasts `Deleted {res.deleted ?? count} tactic(s).`

#### 8.1.14 Import result toasts (from ImportBoardDialog's `onImported`)
After `refresh()`:
- **Disconnected:** `Disconnected {board} — removed {N} tactic(s).`
- **Undo:** "Import undone — those tactics were removed."
- **Imported:** `Imported {N} card(s) from Jira|Trello.`

#### 8.1.15 Page-level mounts
These are always mounted:
- `TacticDrawer`: `tacticId=drawerId`, full tree, `config.fields`. `onChanged` calls refresh; `onEditFull` closes the drawer and opens the editor; `onDelete` calls `remove`.
- `BoardSettingsDialog`, `NodeInspector`, the two confirmation dialogs, `ImportBoardDialog`, and the Snackbar.

#### 8.1.16 Leftovers from the removed PlanGrid
- `gridUpdate(id, patch)` (PUT then refresh) and `askDeleteAll()` are defined but no longer used.
- `reschedule(id, patch)` is used by the Calendar: it optimistically patches the tactic inside `board`, then sends `PUT /tactics/:id`. On error, a toast and a refresh.

### 8.2 Plans view: dual-plan register (`PlansView.jsx`)

**Purpose:** a Hive9-style register. Every campaign carries two paired plans, **SP&A** (Sales, Promotion & Advertising: spend) and **BCPIR** (Business Case & Post-Investment Review: returns/ROI). Plans and campaigns are created, edited, deleted and approved here.

**Data:** `api.campaignPlans()` → `GET /api/planning/plans`, called once on mount and after every mutation. It does not use Planning's tree, filter or plan chips. Loading and error are handled by its own StateWrap.

**Deep link:**
- Initial state comes from `new URLSearchParams(location.search).get('plan')`, e.g. `/planning?plan=spa:<campaignId>` or `?plan=bcpir:<campaignId>`.
- `setOpenPlanId(id)` writes or removes `?plan=` with `history.replaceState`, so no history entry is added.
- When `openPlanId` is set, PlansView renders only `<CampaignPlanDetail>` (§8.3).
- Procurement's PR drawer links here ("View linked SP&A plan →" → `/planning?plan=spa:${pr.campaignId}`).
- **Caveat:** the param is read only when PlansView mounts, which happens only when the remembered tab is `plans`. If the user last used Board, the deep link does nothing until they click the Plans tab. A rebuild should force `view='plans'` whenever `?plan=` is present.

**Layout, top to bottom:**
1. **Portfolio totals.** Five `Total` cards (22px values) in a grid: 2 columns, then 3 at sm, then 5 at lg.

   | Label | Value | Accent | Tooltip |
   |---|---|---|---|
   | Planned cost | `fmtMoney(totals.plannedCost)` | SPA blue | "Total planned SP&A spend across every campaign." |
   | Actual spend | `fmtMoney(totals.actualSpend)` | ink | "Spend recognised to date across all SP&A plans." |
   | Planned return | `fmtMoney(totals.plannedReturn)` | BCPIR violet | "Revenue the BCPIR business cases committed to." |
   | Actual return | `fmtMoney(totals.actualReturn)` | ink | "Revenue realised to date across all BCPIR plans." |
   | Portfolio ROI | `+X%` (`roi*100`, 0 dp, sign shown) | good if roi ≥ 1, else warning | "Blended return on investment across the portfolio." |

2. **Intro row.**
   - Left: "Every campaign runs two paired plans — **SP&A** to track spend and **BCPIR** to track ROI (return on investment)." SP&A and BCPIR are in their plan colours. An InfoTip: "Shell's Hive9 model: SP&A (Sales, Promotion & Advertising) plans track the money going out; BCPIR (Business Case & Post-Investment Review) plans track the revenue, pipeline and ROI earned on that spend."
   - Right: a currency badge `"{symbol} {code}"` from `activeCurrency()`, and a **New plan** button (contained, "+", `canEdit`).

3. **Register table.** `min-w 900`, rounded border, horizontal scroll. Columns: Name, Plan type, ID (right), Start, End, Status, Planned cost (right), ROI (right), and a last column headed "Actions" when `canEdit`, otherwise blank.

**Rows per plan group (`GroupRows`):**

- **Plan group header row** (tinted):
  - A collapse button with a chevron, the plan name (bold), the fiscal year (11px), and a chip "{N} campaign(s)".
  - Collapsed groups are held in a Set in component state, not persisted.
  - If `canEdit` and the group is a real plan (`planId` truthy; a group with no planId has no actions):
    - "+ Add campaign" (outlined) opens the campaign dialog with that plan pre-selected.
    - Edit IconButton (tooltip "Edit plan").
    - Delete IconButton (tooltip "Delete plan").
  - Column spans: name cell spans 6 when editable (else 9), actions cell spans 3.
- **Empty plan:** "No campaigns in this plan yet." plus, for editors, a link "Add the first campaign →".

**Per campaign (`CampaignBlock`):**

1. **Campaign header row:**
   - Campaign name (13.5px semibold).
   - A status pill coloured from `c.spa.status` (via `statusStyle`):
     - Published: green tint, green text.
     - Submitted: `STATUS.warning` tint, `#b0790a` text.
     - Anything else (Draft or Created): grey `#8a8a8f`.
   - "{N} tactic(s)".
   - An italic approval note:
     - Submitted: "· Awaiting approval · {approver}".
     - Published and `approvedBy` set: "· Approved by {name}".
   - **Actions** (`canEdit`):
     - Status Draft, Created or empty → **"Submit for approval"** (outlined, send icon). Opens the submit dialog, pre-filled with any existing approver.
     - Status Submitted and the user can approve → **Approve** (green contained, check icon) and **Reject** (red outlined).
     - Status Submitted and the user cannot approve → a grey pill "Awaiting {approver || 'approver'}".
     - Always: Edit (pencil, tooltip **"Edit plan"**, though it edits the campaign) and Delete (tooltip "Delete plan").
   - **Approve** calls `POST /api/planning/plans/:campaignId/approve {}` and toasts "{campaign} approved & published."
   - **Reject** calls `POST /api/planning/plans/:campaignId/reject {note: undefined}`. No note is collected. Toast "{campaign} returned to draft."

2. **Two plan rows (`PlanRow`)**, SP&A first, then BCPIR:
   - Each row has a 3px left border in the plan-type colour, and clicking anywhere on it opens the plan.
   - **Name:** "↳ **{campaignName}** — {plan.type}".
   - **Plan type:** a type chip (`plan.type`, tinted) plus a small label, "SPEND" for SP&A or "ROI" for BCPIR.
   - **ID:** `plan.displayId`.
   - **Start and End:** raw `startDate` and `endDate` strings.
   - **Status:** a status pill.
   - **Planned cost:** SP&A rows show `fmtMoney(c.spa.plannedCost)`; BCPIR rows show "—".
   - **ROI:** BCPIR rows show `+X%`, green if roi ≥ 1 else warning; SP&A rows show "—".
   - **Last cell:** an "Open plan" icon (OpenInFull), shown for everyone, including Viewers.

**Empty register:** "No plans yet. Click "New plan" to create one, then add campaigns to it."

**Dialogs:**

1. **Plan dialog** (create or edit). Title "New plan" or "Edit plan".
   - **Plan name.** Autofocus, placeholder "e.g. FY2026 Marketing Plan".
   - **Owner.** PeoplePicker.
   - **Fiscal year.** Text, placeholder "2026". When editing, the "FY" prefix is stripped.
   - **Currency.** Select: USD, EUR, GBP, CAD, AUD, INR, JPY, SGD.
   - **Start date.** Default 2026-01-01.
   - **End date.** Default 2026-12-31.
   - Help text: "A plan is a top-level container (e.g. an annual marketing plan). Add campaigns to it — each campaign gets its paired SP&A and BCPIR plans."
   - **Buttons:** Cancel, and "Create plan" or "Save". Disabled while busy or when the name is empty. Closing is blocked while busy.
   - **API:** create `POST /api/plans/nodes {level:'plan', parentId:null, name, owner, ownerId, fiscalYear, currency, startDate, endDate, description}`; edit `PUT /api/plans/nodes/:id {…same}`.
   - **Toasts:** "Plan created — add campaigns to it next." or "Plan updated."
   - **Quirk:** edit sets `description: ''`, which wipes any existing description.

2. **Campaign dialog.** Title "New campaign" or "Edit campaign".
   - **Campaign name.** Autofocus.
   - **Plan.** Select of the register groups; disabled when editing.
   - **Owner.** PeoplePicker.
   - **Status.** Select: Draft, Submitted, Published. A "Created" value is normalised to Draft. **Editors can set Published directly, bypassing approval.**
   - **Start date and End date.** Defaults 2026-01-01 and 2026-12-31.
   - Note: "This creates the campaign and both its paired plans — SP&A (spend) and BCPIR (ROI)."
   - **Buttons:** Cancel, and **"Create plan"** (the label even in add mode) or "Save". Disabled unless both name and plan are set.
   - **API:** `POST /api/plans/nodes {level:'campaign', parentId: planId, name, owner, ownerId, startDate, endDate, status, description}`, or `PUT /api/plans/nodes/:id`. The description is always `''`.
   - **Toasts:** "Campaign created — its SP&A and BCPIR plans are ready." or "Campaign updated."

3. **Delete confirmation.** Title `Delete "{name}"?`
   - Plan text: "This permanently removes the plan and every campaign inside it — including all their SP&A and BCPIR plans and tactics. This cannot be undone."
   - Campaign text: "This permanently removes the campaign, both its SP&A and BCPIR plans, and all its tactics. This cannot be undone."
   - This wording contradicts the server's soft delete to the recycle bin and the Planning page's own copy; the rebuild should reconcile them.
   - Calls `DELETE /api/plans/nodes/:id`. Toast "Plan deleted." or "Campaign deleted."
   - The buttons are shown to every editor, but the server allows only admins, so a non-admin sees an error toast.

4. **Submit for approval.** Title `Submit "{name}" for approval`.
   - Text: "Send both the SP&A and BCPIR plans for sign-off. Pick the approver — they'll get a notification and be shown as the pending approver until the plan is published."
   - A PeoplePicker "Approver" with placeholder "Search for an approver…". The approver is optional.
   - Buttons: Cancel, Submit.
   - Calls `POST /api/planning/plans/:campaignId/submit {approver, approverId}`.
   - Toast: "Submitted for approval to {name}." or "Submitted for approval."

### 8.3 Campaign plan detail: SP&A and BCPIR (`CampaignPlanDetail.jsx`)

**Props:** `planId` (`"spa:<campaignId>"` or `"bcpir:<campaignId>"`), `onBack`, `onSwitch`, `canEdit`, `onToast`.

**Data:** `GET /api/planning/plans/{encodeURIComponent(planId)}`, re-fetched when `planId` or `reloadKey` changes. `reload()` increments `reloadKey`.

**Header row** (rendered outside StateWrap, so it stays visible while the body loads):
- **Left:**
  - A back IconButton (`title="Back to plans"`) that clears the `?plan=` param.
  - The campaign name as an h3 (17px bold) plus a `StatusPill` (Published green, Submitted amber, otherwise grey).
  - A sub-line: "{planName} · {fiscalYear} · Plan ID {displayId} · {startDate} → {endDate}".
- **Right:** a pill toggle with **"SP&A · Spend"** and **"BCPIR · ROI"**. Switching calls `onSwitch('spa:'|'bcpir:' + header.campaignId)`, which also updates the URL.

**Connection bar:**
- Labelled "CONNECTED", followed by pill links (react-router `Link`) with tooltips.
- **SP&A links:**
  - Budget → `/budget?campaignId=X` ("This campaign's budget, scoped")
  - Purchase requests → `/procurement?campaignId=X` ("This campaign's purchase requests & approvals")
  - Board → `/planning` ("See the tactics on the board")
- **BCPIR links:**
  - Analytics → `/analytics` ("Measured performance behind these returns")
  - Attribution → `/attribution` ("How revenue is attributed across channels")
  - Budget → `/budget?campaignId=X` ("The spend behind these returns")
- **Quirk:** "Board" stays on the same route, so it does not close the detail or switch the tab.

#### 8.3.1 SP&A detail (`SpaDetail`)
**Data:** `{kpis, byQuarter, byCategory, lineItems, procurement, budget, header}`.

1. **KPI row** (5 `Kpi` cards, 23px values):
   - **Planned cost.** Tip: "Total planned SP&A spend for the campaign — the sum of tactic budgets."
   - **Committed.** Accent SERIES[3]. Tip: "Spend already committed via purchase orders / bookings."
   - **Actual spend.** Accent SPA blue. Tip: "Spend recognised to date, paced with delivery."
   - **Utilisation.** `fmtPct(utilizationPct)`; red if > 100, else green. Sub-line: "{remaining} remaining". Tip: "Actual spend as a share of planned cost."
   - **Variance.** `+x.x%`; red if > 0, else green. Tip: "Actual vs planned spend. Negative = under budget."

2. **Charts** (ECharts `EChart`, height 280, laid out 1.6fr | 1fr):
   - **"Spend by quarter — planned vs actual".** Tip: "When the SP&A budget is planned to go out vs what has been spent so far." Grouped bars: Planned (`#9aa0aa`) and Actual (blue), `barMaxWidth 22`. Tooltip `trigger: axis` with full-money values; y-axis in compact money; legend top-right.
   - **"Spend by category".** Tip: "Planned SP&A spend split across channels / cost categories." Donut with radius 52–74%, centre (50%, 44%), no labels, scrolling legend at the bottom, slice colours from SERIES.

3. **Two panels side by side:**
   - **Budget reconciliation** (if `budget`).
     - InfoTip: "Top-down budget allocated to this campaign in the Budget module vs the bottom-up cost planned in this SP&A plan. Keeps planning and funding in sync."
     - Link "Open Budget →" to `/budget?campaignId=`.
     - Four `Mini` tiles, each with its value pinned to the bottom so all values line up:
       - Allocated (top-down), SERIES[4]
       - Planned (this plan), SPA blue
       - Committed (roll-up), SERIES[3]
       - Vs budget: `(planned − allocated) / allocated × 100`, signed; red if planned > allocated, else green.
   - **Procurement & approvals · {count}** (if `procurement`).
     - InfoTip: "Purchase requests raised against this plan, and where they are in the approval chain. Committed = approved & beyond; invoiced = money actually billed."
     - Link "Open Procurement →" to `/procurement?campaignId=`.
     - Mini tiles: Committed (approved) SERIES[0]; Invoiced SERIES[1]; Pending approval `#b0790a`; Draft `#8a8a8f`.
     - Table (`min-w 560`): Purchase request, Vendor (or "—"), Approval status chip, Amount (`r.total`).
     - Status chip map: draft "Draft" grey; submitted "Pending approval" amber; approved "Approved" blue; rejected "Rejected" red; po "PO raised" violet; invoiced "Invoiced" aqua; received "Received" green. Unknown statuses use their raw value in grey.
     - Empty: "No purchase requests yet. Use "Raise PR" on a cost line below to commit spend through the approval chain."

4. **Cost line items card.**
   - Toolbar: "Cost line items · {N}", an "Export CSV" text button, and "+ Add cost line" (`canEdit`).
   - Export file `spa-cost-lines.csv`, columns: Tactic, Channel, Quarter, Planned, Committed, Actual.
   - Table (`min-w 760`) columns: Tactic, Channel, Quarter, Planned, Committed, Actual (bold), Var. (`+x.x%`, red if > 0, else green), and Actions (`canEdit`).
   - Row actions appear on hover:
     - **Raise purchase request** (receipt icon).
     - Edit.
     - Delete, which opens the confirmation.
   - Empty: "No cost lines yet. Add one to build the SP&A plan."

**Raise PR:**

```
POST /api/procurement/requests {
  title: `${r.tactic} — ${header.campaignName}`,
  vendor: r.channel || '',
  tacticId: r.id,
  campaignId: header.campaignId,
  lines: [{ description: `${r.tactic} (planned)`, qty: 1, unitPrice: r.planned, amount: r.planned }]
}
```

Then the toast "Draft purchase request created — opening Procurement to submit it." and, 700 ms later, `navigate('/procurement')`.

#### 8.3.2 BCPIR detail (`BcpirDetail`)
**Data:** `{kpis, byQuarter, byChannel, businessCase}`.

`roiAccent` is good when roi ≥ 1, warning when ≥ 0, and critical otherwise.

1. **KPI row (5):**
   - **Investment.** Tip: "SP&A spend under review — the cost this business case is measured against."
   - **Planned return.** SERIES[3]. Tip: "Revenue the business case committed to."
   - **Actual return.** BCPIR violet. Sub-line: "{attainmentPct}% of plan". Tip: "Revenue realised to date."
   - **ROI.** `+X%`. Tip: "Return on investment = (return − spend) ÷ spend."
   - **ROAS.** `x.xx×`. Tip: "Return on ad spend = return ÷ spend."

2. **Charts** (50 | 50):
   - **"Return by quarter — planned vs actual".** Tip: "The business case's expected revenue vs what has been realised each quarter." Bars: "Planned return" (grey) and "Actual return" (violet).
   - **"Return by channel".** Tip: "Planned vs realised revenue for each channel in the campaign." Horizontal bars (category y-axis, order reversed), Planned (grey) and Actual (violet), `barMaxWidth 12`.

3. **Second KPI row (4):**
   - Planned pipeline. Tip: "Sales pipeline the plan expects to generate."
   - Actual pipeline (SERIES[1]).
   - Planned leads (`fmtNum`). Tip: "Target lead volume for the campaign."
   - Actual leads (SERIES[1]).

4. **Business case by tactic · {N}.**
   - Export CSV `bcpir-business-case.csv`, columns: Tactic, Channel, Spend, Target revenue, Actual revenue, Target leads, Actual leads, ROAS (2 dp).
   - "+ Add entry" (`canEdit`).
   - Table (`min-w 860`) columns: Tactic, Channel, Spend, Target rev., Actual rev. (bold), Leads (a/t) shown as "actual / target", ROAS (`x.xx×`, green if ≥ 1, else warning), and Actions (Edit, Delete; no PR action).
   - Empty: "No business-case entries yet. Add one to build the BCPIR plan."

#### 8.3.3 Row dialog (each line is a real tactic)
**Title:** "Add" or "Edit", followed by "cost line" (SP&A) or "business-case entry" (BCPIR).

**Fields:**
- **Tactic / line name.** Autofocus. Required: the save button stays disabled when it is empty.
- **Channel.**
- **SP&A only:**
  - **Quarter.** Placeholder "Q3 2026"; the add default is "Q3 2026".
  - **Planned cost ($).**
  - **Actual spend ($).** Helper: "Leave blank to auto-track by progress".
- **BCPIR only:**
  - **Spend / investment ($)**, **Target revenue ($)**, **Actual revenue ($)**, **Target leads**, **Actual leads**.

**Buttons:** Cancel, and Add or Save.

**Save:**
- Add sends `POST /api/tactics {type:'Campaign', status:'in-progress', campaignId: header.campaignId, planId: header.planNodeId, programId: header.defaultProgramId, …payload}`.
- Edit sends `PUT /api/tactics/:id {…payload}`.
- **SP&A payload:** `{title, channel, quarter, budget: Number, spendActual: '' → null | Number}`.
- **BCPIR payload:** `{title, channel, budget, performance: {plan: {revenue, leads}, actual: {revenue, leads}}}`. This overwrites the whole performance object, dropping pipeline values.
- Toasts: "Line added." or "Line updated."

**Remove confirmation:**
- Title `Remove "{name}"?`
- Body: "This removes the tactic from both the SP&A and BCPIR plans (and the board). This cannot be undone."
- Buttons: Cancel, and Remove ("Removing…").
- Calls `DELETE /api/tactics/:id`. Toast "Line removed."

### 8.4 Grid view: the five-level Hive9 plan grid (`HierarchyGrid.jsx`)

**Props:** `tree` (= visibleTree), `canEdit`, `canAdmin`, `onOpenNode`, `onAddNode(parent|null, level)`, `onOpenTactic`, `onAddTactic(programId)`, `onDeleteNode`, `onDeleteTactics(ids, clear)`, `onChanged` (= refresh), `onToast`.

**Fields the tree nodes are expected to carry:**
- **Plan, campaign or program node:** `id, level, name, owner, ownerId, parentId, children[], planBudget, fiscalYear, isStrategic, status (campaign), effectiveStart, effectiveEnd, dateMode ('auto'|'manual'), startDate, endDate, currency, modelId, teams[], goals{}, defaultMechanism, description, comments[]`.
- **Roll-up on each node:** `rollup: {tacticCount, lineItemCount, money: {plannedCost, lineItems, balance, actual}, unallocatedBudget, overParent}`.
- **Programs** also carry `tactics[]`.
- **Tactic:** `id, title, externalName, owner, startDate, endDate, dueDate, evergreen, lifecycleStatus, approvalStatus, budget, money{plannedCost, lineItems, balance, actual}, lineItems[], planId, campaignId, programId, goals, …`.
- **Line item:** `id, name, vendor, owner, amount, actual, mechanism, segment, allocations{period: amount}, budgetPoolId`.

**Expand state:**
- Starts with every plan, campaign and program id expanded. Tactics start collapsed, so line items are shown on demand.
- Containers that appear later (newly created) are auto-expanded, tracked through a `seen` ref.
- Expanded state is not persisted.

#### 8.4.1 Control panel
**Left side:**
- A search field (280px, pill-shaped, search icon) with placeholder **"Search name, external name or ID"**.
- "Expand all" (unfold-more icon, with tooltip): every container and every tactic.
- "Collapse all" (unfold-less icon): an empty set, so only plan rows remain visible.
- A **"System rows"** switch (default ON) with an InfoTip (G.sysGenBalance).

**Right side:**
- **Honeycomb toggle** (`canEdit`). Hexagon icon, tooltip "Honeycomb: select tactics for bulk copy, move, submit or delete". The label is "Select" (outlined) or "Done selecting" (contained). Toggling clears the selection.
- **Export CSV.**
- **New plan** (contained, `canEdit`), which opens the Inspector in add-plan mode.

**Search semantics:**
- Matching is case-insensitive substring.
- Nodes match on `name + id`. Tactics match on `title + externalName + id`. Line items match on `name + vendor`, and a line item is also shown when its parent tactic matches.
- A match keeps its ancestors visible and auto-expands the path to it.
- A matched tactic with line items is shown expanded.
- No matches: "Nothing matches that search."

#### 8.4.2 Honeycomb bulk bar (when honeycomb is on)
- Hexagon icon, "{N} tactic(s) selected", and "· tick tactics below".
- Buttons, all disabled when nothing is selected: **Copy to…**, **Move to…**, **Submit for approval**, **Delete** (red, which calls `onDeleteTactics(selectedIds, clearSel)` and so the page's bulk-delete confirmation).
- **Clear**, shown when there is a selection.
- Checkboxes appear only on tactic rows. A selected row is tinted `rgba(0,113,227,0.06)`.

**Bulk dialog.** Title "Move | Copy | Submit {N} tactic(s)".
- **Copy / Move:**
  - A **Target program** select (`programOptions`).
  - Copy note: "Copies keep their line items (without actuals or purchase requests) and start in Created status."
  - Move info Alert: "Moved tactics take on the target's plan and campaign."
  - Confirm is disabled until a program is chosen.
- **Submit:**
  - A PeoplePicker "Approver (optional)" with helper "Leave empty to let any team lead or manager approve."
  - Note: "Tactics missing required fields (owner, dates, planned cost) are skipped and listed."
- **Buttons:** Cancel, Confirm ("Working…").
- **API calls:**
  - Move: `POST /api/planning/tactics/bulk-move {ids, programId}` returns `{moved}`. Toast "Moved N tactic(s)."
  - Copy: `POST /api/planning/tactics/bulk-copy {ids, programId}` returns `{copied}`. Toast "Copied N tactic(s) (they start as Created)."
  - Submit: `POST /api/planning/tactics/bulk-submit {ids, approver, approverId}` returns `{submitted, skipped: [{title, reason}]}`. Toast "Submitted N." and, if any were skipped, with severity **warning**: " Skipped M: Title (reason); Title2 (reason2)".
- **After each:** close the dialog, clear the selection, then `onChanged()`.

#### 8.4.3 Table
**Size and columns:**
- `min-w 1040`, rounded, scrolls horizontally.
- Columns: **Name** (InfoTip hierarchy), **Owner**, **Dates**, **Status**, **Plan budget** (InfoTip planBudgetField, right-aligned), **Planned cost** (InfoTip, right), **Line items** (right), **Balance** (InfoTip sysGenBalance, right), **Actual** (right), and an actions column (104px).

**Name cell:**
- Indented 16px per depth.
- A chevron toggle if the row has children; otherwise a 21px spacer.
- A 4×16px colour bar in the level colour.
- Row actions sit at 60% opacity and become 100% on row hover. Each action is an icon with an arrow tooltip.

**Node row (plan, campaign, program):**
- Plan rows are tinted. Strategic (AOP) plans have muted text and a 40%-opacity colour bar.
- Name cell contents: a LevelTag chip (e.g. "PLAN"), the name as a link-button that opens the Inspector (bold for plans), and the tactic count (11px).
- **Owner:** truncated at 140px.
- **Dates:** `effectiveStart → effectiveEnd`.
  - Format "Jul 1 → Dec 31 ’26". The year is appended to the end date, and to the start date when the years differ or the end is missing.
  - The `title` attribute shows the full ISO dates.
  - Non-plan nodes in `dateMode === 'auto'` show an "AUTO" badge (tooltip G.dateModeAuto).
  - "—" when there are no dates.
- **Status chips:**
  - Plans: `FY{fiscalYear}` (`#5b6b7a`).
  - Strategic plans: "AOP" (grey, tooltip strategicPlan).
  - Campaigns: their status (Draft `#86868b`, Submitted `#c77700`, Published green; tooltip "SP&A / BCPIR plan status").
  - `rollup.overParent`: "Over parent" (red, tooltip "This budget is larger than its parent's").
- **Plan budget:** `fmtMoney(planBudget)` or "—". Plans and campaigns with a budget > 0 also show "{unallocated} unalloc." underneath.
- **Money cells:** Planned cost (bold), Line items, Balance (red when < 0), Actual. Null values show a muted "—".
- **Actions:**
  - Eye "Open {level}" → Inspector.
  - "+" "Add {child level}" (`canEdit`). A program opens the tactic editor; other levels open the Inspector in add mode for the child level.
  - Trash "Delete {level} and everything under it" (`canAdmin`) → the page's cascading delete confirmation.

**Tactic row:**
- A checkbox when honeycomb is on, a LevelTag "TACTIC", the title as a link that opens the TacticDrawer, and the line-item count.
  - The title tooltip shows "{title} · external: {externalName}" when there is an external name.
- **Dates:** start → end. Evergreen tactics show an ∞ icon (`titleAccess` "Evergreen").
- **Status:** a lifecycle chip showing `lifecycleStatus || approvalStatus || 'Created'` in LIFECYCLE_COLOR (tooltip approvalLifecycle).
- **Plan budget:** "—" (tooltip "Tactics have planned cost, not a budget").
- **Money:** from `t.money`. If missing, it falls back to `{plannedCost: budget, lineItems: 0, balance: budget, actual: 0}`.
- **Actions:** Open tactic (eye); Add line item ("+", `canEdit`), which opens LineItemDialog; Delete tactic (`canEdit`), which calls `onDeleteTactics([id])` and so the confirmation.

**Line item row** (12.5px):
- LevelTag "LINE ITEM". The name is a link that opens the edit dialog (only for editors). "· vendor" is shown muted.
- **Owner:** owner or "—".
- **Dates:** the first to last allocation period labels, e.g. "Mar 2026 → Dec 2026".
- **Status:** a chip "{mechanism} · Monthly|Quarterly" (tooltip `"{MECHANISM_LABEL} — {spreadMechanism tip}"`). A muted "unlinked" appears when there is no `budgetPoolId`.
- **Planned cost** = `amount`. **Actual** shown. Other money cells are blank.
- **Actions** (`canEdit`): Edit, which opens LineItemDialog; Delete, which opens the line-item confirmation:
  - Title "Delete line item?"
  - Body: `"{name}" ({amount}) goes to the recycle bin — you can restore it onto "{tactic}" later.`
  - Calls `DELETE /api/tactics/:tacticId/line-items/:id`. Toast `Line item "X" moved to the recycle bin.` Then `onChanged`.

**System row** (when the switch is on, the tactic has line items, and the tactic is expanded):
- Italic. Name "Sys_Gen_Balance" with an InfoTip. Owner "System".
- The Planned cost column shows the tactic's `money.balance`: red if < 0, muted otherwise.

**Footer** (when the tree is not empty):
- "Total · {N} tactic(s)", counting every tactic in the visible tree.
- Sums of plan budget (plans only), planned cost, line items, balance (red if < 0) and actual.

**Empty:** "No plans yet — create one to start."

**Caption under the table:** "Levels are created in order — use + on a row to add the next level down. Plans → campaigns → programs → tactics → line items."

**LineItemDialog saved toast:** "Line item updated." or "Line item added."

#### 8.4.4 Grid CSV export (`zenreach-plan-grid.csv`)
- Exports the whole visible tree, ignoring search and expand state.
- Columns: Level, Path ("A › B › C"), Name, Owner, Start, End, Status, Plan budget, Planned cost, Line items, Sys_Gen_Balance, Actual.
- **Plans, campaigns and programs:** Start and End are `effectiveStart` and `effectiveEnd`. Status is `FY{fiscalYear}` for plans, otherwise `node.status`.
- **Tactics:** End is "Evergreen" for evergreen tactics. Status is `lifecycleStatus`. Plan budget is blank.
- **Line items:** Start and End are the first and last allocation keys. Status is "{mechanism} · {segment}". Planned cost is `amount`.

### 8.5 Budget & Cost view (`BudgetCostView.jsx`)

**Purpose:** a read-only grid that spreads the planned cost of every level over the periods of one fiscal year.

The header tooltip reads: "Planned cost spread across the months (or quarters) it will be spent in, built from each tactic's line items. Any planned cost not yet itemized (the Sys_Gen_Balance) is spread evenly over the tactic's dates."

**Maths:**
- **Per tactic (always computed in months):**
  - Start with the sum of `allocationsIn(lineItem,'month')` over its line items.
  - Then, if `balance = budget − Σ lineItem.amount > 0`, spread the balance evenly over `periodsFor(t,'month')`.
  - If the tactic has no dates, the balance lands on the first month of the quarter parsed from `t.quarter` (format "Q3 2026"). In quarter mode it lands on that quarter.
- **Containers:** the sum of their children.
- **Quarter mode:** months roll up into quarters at display time.

**State:**
- `segment`: `month` by default.
- `expanded`: plans and campaigns start open, programs closed.
- `year`: defaults to the first plan's `fiscalYear`, else the current year.
- The available years are the union of the plans' fiscal years and every year that appears in the phased data. When nothing is found, the current year.
- The component is keyed by the active plan, so switching plan chips resets this state.

**Toolbar:**
- **Left:** "Planned cost, time-phased (i) · **{fmtMoney(grand)}** in FY{year}", plus "against **{budgetTotal}** plan budget" when the plans have budgets.
- **Right:**
  - An FY select (`FY2026`…, pill-shaped, 32px high).
  - A Monthly / Quarterly toggle.
  - **Export CSV**, file `zenreach-budget-cost-{year}.csv`. Columns: Level, Name (indented two spaces per depth), Plan budget, one column per period label ("Jan 2026"… or "Q1 2026"…), and "Total {year}". It includes every row regardless of expansion.

**Table:**
- `min-w 1280` in month mode, 820 in quarter mode.
- **Name column:** sticky on the left. It has an expand chevron for anything with children: plans, campaigns, programs (which hold tactics) and tactics (which hold line items). Then a colour bar and the name, with tooltip "{Level}: {name}". Line items are shown in ink2.
- **Plan budget:** containers only; "—" otherwise.
- **Period columns:** short labels "Jan" or "Q1". A cell shows `fmtMoney`, or a muted "·" when it is zero.
- **FY total column:** bold.
- Plan rows are tinted and semibold.
- **Footer:** "Total", the budget total, column totals, and the grand total.
- **Empty:** "No cost data."
- **No editing** anywhere in this view.

### 8.6 Calendar view (`CalendarView.jsx`)

**Scope:** a **month view only**. There is no quarter, week or Gantt view and no multi-day bars. Each tactic appears once, on `startDate || endDate || dueDate`.

**Props:**
- `tactics` (= allInitiatives, scoped by the plan and node filter)
- `canEdit`, `onReschedule`, `onOpen`
- `planColor`, `colorByPlan`
- `plans` (= visibleTree)
- `planIds` (`[activePlanId]` or `[]`)

**Toolbar** (hidden when printing):
- **Left:**
  - A "‹" chevron, the label "{Month} {Year}" (160px, 15px semibold), and a "›" chevron.
  - **Today** resets to the current month.
  - The view always opens on the current month.
- **Right:**
  - **Connect calendar** (outlined, RSS icon) opens the SubscribeDialog.
  - **Export .ics** (event icon) calls `exportTacticsIcs(tactics)`.
  - **Print / PDF** calls `window.print()`.
  - None of these is permission-gated. Viewers can use them too.

**Plan legend:** shown when `colorByPlan && plans.length > 1`, as a colour dot plus name for each plan (hidden in print).

**Grid** (`.calendar-print`, rounded border):
- A weekday header Mon–Sun on a grey background, then 6 weeks × 7 days, starting on Monday.
- **Day cells:** at least 92px tall.
  - Days outside the month: grey background, 55% opacity.
  - Today: the date in a blue `#0071e3` circle.
  - While a drag hovers: background `rgba(0,113,227,.08)`.
- **Chips:**
  - Truncated title in white 10.5px text.
  - Background is the plan colour in overlay mode, otherwise the type colour (`#888` fallback).
  - Tooltip "{title} · {owner}". Clicking opens the TacticDrawer.
  - Draggable when `canEdit`.

**Unscheduled section** (when some tactics have no dates; hidden in print):
- "Unscheduled ({N}) — drag onto a day", with draggable chips.

**Drag to reschedule:**
- The day accepts `dragOver` only when `canEdit`.
- On drop:
  - `newStart` = the ISO date of that day.
  - If the tactic has both a start and an end (`endDate || dueDate`), the duration in days is kept: `endDate = dueDate = newStart + duration`.
  - Otherwise `endDate = dueDate = newStart`.
- Then `onReschedule(id, {startDate, endDate, dueDate})`, an optimistic update plus `PUT /api/tactics/:id`. On error, a toast and a refresh.
- **Note:** dropping an evergreen tactic sets an end date on it.

**Date helpers:** `parse` treats `yyyy-mm-dd` as a local date to avoid time-zone shifts; `iso` formats a local date.

#### 8.6.1 Subscribe dialog: "Connect Google Calendar / Outlook" (`SubscribeDialog`)
- Close X in the top right.
- **"Syncing: {planLabel}".** `planLabel` is the visible plans' names joined with commas (in "All" mode, every plan), or "all plans" when there are none.
- **`<GoogleCalendarSync compact planIds planLabel/>`** (§8.7).
- **"Or subscribe to a live .ics feed"**, with explanation: "Subscribe your calendar to this live feed of scheduled tactics — it refreshes automatically (about every few hours)."
- **Feed URL:** a read-only monospace field containing `${origin}/api/planning/calendar.ics` plus `?planIds=<id>` when a single plan is active. A copy icon's tooltip switches to "Copied!" for 1.5 s.
- **Localhost warning** (amber box), shown when the origin contains localhost or 127.0.0.1: "You're on localhost. Google and Outlook fetch feeds from their own servers, which can't reach your machine — so live subscription needs ZenReach on a public URL. For now, use Export .ics and import the file (one-time), which works locally."
- **Instructions:**
  - **Google Calendar:** "Other calendars → + → From URL → paste the link → Add calendar. (Read-only; Google refreshes it periodically.)"
  - **Outlook:** "Add calendar → Subscribe from web → paste the link → name it → Import. (Read-only; auto-refreshes.)"
  - **One-time import (works on localhost):** "Use Export .ics above, then in Google: Settings → Import & export → Import; in Outlook: Add calendar → Upload from file."
- **Actions:**
  - "Copy webcal:// link" (the http(s) scheme replaced with `webcal://`); shows "Copied webcal://" for 1.5 s.
  - "Done".

**Server:**
- `GET /api/planning/calendar.ics` is **exempt from authentication** (`server/index.js:196`), so the feed is public.
- It accepts `planId`, `campaignId`, `programId` and `planIds` (comma-separated).
- Response headers: `text/calendar`, `inline; filename="zenreach-calendar.ics"`, `Cache-Control: public, max-age=3600`.

### 8.7 Google Calendar push (`GoogleCalendarSync.jsx`)

This component is reused on the Integrations page (full mode) and inside the Calendar's subscribe dialog (`compact`).

**Props:** `compact`, `onChange(status)`, `planIds`, `planLabel`.

**Data:** on mount, `GET /api/integrations/google-calendar/status` returns `{serviceAccount, connected, calendarId, connectedAt}`. Failure is swallowed.

**Header:**
- **Compact:** "Google Calendar" plus either a green "✓ Connected" badge or a blue "LIVE · works on localhost" badge.
- **Full:** a gradient icon tile, "Push to your Google Calendar", and "Your scheduled tactics appear in your Google Calendar — and on your phone — in seconds."

**Not connected:**
- One-time setup text: "One-time setup on a computer: Google Calendar → your calendar → Settings & sharing → Share with specific people → add **{service-account email}** with "Make changes to events.""
- If a service-account email is known, a read-only monospace field shows it, with a copy icon whose tooltip is "Copy service-account email" (or "Copied!").
- An email input with placeholder "your-google-account@gmail.com", and a **Connect** button that is disabled unless the input contains "@" (shows "Connecting…" while busy).

**Connected:**
- A green box: "Connected to {calendarId}" and "Since {connectedAt localeString}".
- Buttons: **Sync again** (re-syncs to `status.calendarId`; shows "Working…" while busy) and **Disconnect** (red).

**Sync:**
- `POST /api/integrations/google-calendar/sync {calendarId, planIds | undefined}` returns `{synced, scheduled, calendarName}`.
- Success message, in green: "Synced {synced}/{scheduled} scheduled tactics[ from {planLabel}] to "{calendarName}". Check your phone's Google Calendar."
- Afterwards, status is reloaded and `onChange` is called.

**Disconnect:**
- `POST /api/integrations/google-calendar/disconnect {}`. The email is cleared.
- Message: "Disconnected. ZenReach events were removed from that calendar."

**Errors:** shown as red text (`#c0392b`) containing the error message.

`api.googleCalendarClear` exists but this component does not use it.

### 8.8 Hierarchy tree sidebar (`HierarchyTree.jsx`)

- A card. At the top is an **"All work"** button (folder icon), bold and highlighted when nothing is selected. Clicking it calls `onSelect(null)`.
- **Nodes (plan → campaign → program):**
  - Indented 10px per depth.
  - Icons: plan = Folder, campaign = Campaign (megaphone), program = AccountTree. The icon is blue when the node is selected, muted otherwise.
  - Plans and campaigns have an expand chevron; programs get a 22px spacer instead.
  - Default open state is `depth < 2`, so plans and campaigns start open. This state is local and not persisted.
  - The name is truncated, with an arrow tooltip showing the full name (200 ms delay).
  - A count pill shows the number of tactics underneath, recursing through `node.tactics`.
  - A selected row is tinted and its name is bold.
- **Hover actions** (`canEdit`; hidden until hover so names get the full width):
  - **Edit** (pencil) → the Inspector.
  - **"Add {child}"** ("+"): plans add a campaign, campaigns add a program, programs add a tactic (which opens the tactic editor with that program).
- Selecting a node changes the page `filter`, which re-fetches the board and summary.

### 8.9 Inspection window for plan, campaign and program (`NodeInspector.jsx`)

**Container:** a right-hand Drawer, 560px wide (full width on xs). Closing is blocked while busy.

**Target:** `target = {mode: 'add'|'edit', level, parentId, node?}`.
- `live` is the fresh tree node (with roll-ups).
- `parent` is the parent from `target.parentId` (in add mode) or `live.parentId`.

**Header:**
- A level chip tinted in the level colour, with an InfoTip (planLevel, campaignLevel or programLevel).
- The title: "New {level}" or the node name (17px bold).
- When there is a parent: `Inside {parent level} "{parent name}"`.

**Tabs** (edit mode only): **Setup** and **Review** (shows "Review · N" when there are comments).

**Error:** an error Alert at the top of the body, showing the server message.

**Draft defaults** (`draftFrom`):
- `owner`: the node's owner, except that "Unassigned" becomes empty.
- `fiscalYear`: the node's value, else `meta.currentYear`, else the current year.
- `currency`: USD.
- `modelId`: the node's model, else the first model.
- `teams`: [].
- `isStrategic`: false.
- `goals`: {}.
- `defaultMechanism`: 'E'.
- `dateMode`: 'auto'.
- `startDate`, `endDate`, `planBudget`: from the node, or ''.

The draft resets each time the drawer opens, or when the node id, mode or parentId changes.

#### 8.9.1 Setup tab
A two-column grid.
- **Name** (required).
  - Helper for plans: "Must be unique across the whole system."
  - Helper otherwise: "Must be unique within this {parent level}."
  - Autofocus in add mode.
- **Owner.** PeoplePicker, full width.
- **Plan only:**
  - **Fiscal year.** Select showing `FY{y}` for `currentYear … currentYear+10`; the existing value is prepended if it falls outside that range.
  - **Currency.** Select over `CURRENCY_SYMBOLS` keys.
  - **Model.** Select of `meta.models`, plus an InfoTip (planModel). Admins also get a **"Model settings"** button (tune icon), which opens the ModelDialog.
  - **Teams.** Multi-select with a checkbox per item. Selections render as chips, with "Removed team" as the fallback name. Helper: "Teams on a plan can see it and everything under it. The owner must belong to one of them."
  - **"Set plan as Strategic / AOP"** checkbox, with an InfoTip.
  - **GOALS** (tip "The plan's targets. Tactics underneath roll their waterfall goals up against these."):
    - A 3-column grid of number fields: Inquiries, MQLs, TQLs, Deals won, Revenue ($).
    - In edit mode, a line underneath: "Tactic goals so far: {Σ inquiries} inquiries · {Σ revenue} revenue", summed over the tactics' `goals`.
  - **Default line-item spread.** Select with options "E · Evenly distributed", "F · First period", "L · Last period", "C · Custom".
- **Campaign and program:**
  - **DATES** (InfoTip dateModeAuto), with a toggle between **"Auto from tactics"** and **"Manual"**.
  - **Manual:** Start date and End date fields. For a program whose parent campaign has manual dates: "Must sit inside the campaign's dates: {start} → {end}."
  - **Auto:** a grey box reading "Calculated from tactics: **{effectiveStart} → {effectiveEnd | 'ongoing'}**", or "Calculated from its tactics once they have dates."
- **Plan budget.** Number field, min 0, with an InfoTip.
  - Helper when the parent has a budget: "{Parent level} budget {$} · {$ remaining} left", plus " (can't be exceeded)" for programs.
  - The remaining amount is the parent's budget minus the sum of the siblings' budgets.
- **Unallocated** (edit mode only; disabled field): `fmtMoneyFull(rollup.unallocatedBudget)` with an InfoTip. Program helper: "Programs hand nothing down — tactics have planned cost instead."
- **Description.** Multiline.
- **Roll-up stats** (edit mode, 3×2 tiles): Tactics, Line items, Planned cost (tip), Line-item total, Sys_Gen_Balance (tip; red if < 0), Actual.
- **Warning Alert** when `rollup.overParent`: "This budget is larger than its parent's budget."

#### 8.9.2 Review tab
- For a campaign that has a status: "SP&A / BCPIR plan status: **{status}** — manage approval from the Plans tab."
- The comments list: author (bold), `toLocaleString` time, and text (whitespace kept). Empty: "No comments yet."
- **Comment input** (`canEdit`): placeholder "Comment for colleagues…"; Enter or **Post** sends it.
  - Calls `POST /api/plans/nodes/:id/comments {text}` and uses the returned `comments`.

#### 8.9.3 Footer (Setup tab, `canEdit` only)
**Visibility:** Viewers see read-only disabled fields and no footer.

**Buttons:**
- **Delete.** Red, trash icon. Edit mode only, and only when `canDelete` (= canAdmin). Opens the page's cascading confirmation.
- **Cancel.**
- **Save** or **"Create {level}"**. Shows "Saving…" while busy and is disabled when the name is empty.

**Save payload:**
- **Plan:** `{name, owner, ownerId, description, planBudget: Number, fiscalYear, currency, modelId, teams, isStrategic, goals, defaultMechanism, startDate: `${FY}-01-01`, endDate: `${FY}-12-31`}`.
- **Campaign or program:** `{name, owner, ownerId, description, planBudget, dateMode, startDate/endDate (only in manual mode, else '')}`.

**Endpoints:**
- Create: `POST /api/plans/nodes {level, parentId, …}`.
- Update: `PUT /api/plans/nodes/:id`.
- Then `onSaved(saved, isAdd)`.

#### 8.9.4 Model settings dialog (admin)
**Fields:**
- **Name.**
- **Average deal size ($).**
- **CONVERSION RATE INTO EACH STAGE.** One field per stage after the first (`meta.stages.slice(1)`), labelled "{prev stage} → {stage} (%)". The value is stored as a fraction and displayed as a percentage rounded to 0.1 (min 0, max 100, step 0.5).
- **Switches:**
  - "Require planned cost (changing it triggers re-approval)".
  - "Block negative balance (line items can't exceed planned cost)".

**Buttons:** Cancel, and "Save model" ("Saving…").

**Save:**
- `PUT /api/plans/models/:id {name, averageDealSize, rates, requirePlannedCost, blockNegativeBalance}`.
- Then `onSaved(null, false, {modelChanged: true})`, which triggers the page's model toast. Errors appear in an Alert inside the dialog.

**`GET /api/plans/meta`** returns `{levels, models: [{id, name, averageDealSize, rates{stageId: fraction}, requirePlannedCost, blockNegativeBalance}], stages: [{id, name}], teams: [{id, name}], currentYear}`.

### 8.10 Tactic drawer: the tactic's inspection window (`TacticDrawer.jsx`)

**Container:** a right-hand Drawer, 520px wide. It opens when `tacticId` is set and is mounted at page level, so the Board, Calendar and Grid can all open it. The Plans view cannot.

**Data:** `GET /api/tactics/:id` returns the decorated tactic plus `missingForSubmit[]`.
- **Loading:** a centred pulsing sparkle icon.
- **Load error:** the message text, centred.

**Saving:** every mutation goes through `act(fn)`:
1. Set busy.
2. Call the API.
3. Re-fetch the tactic.
4. Call `onChanged(fresh)`, which runs the page `refresh()`.

Errors appear in a dismissible error Alert at the top of the body.

**Header:**
- The lineage "Plan › Campaign › Program" (11.5px, truncated).
- The title (17px bold).
- "External name: **X**", if set.
- **Chip row:**
  - The lifecycle chip (`lifecycleStatus || approvalStatus || 'Created'`).
  - An "∞ Evergreen" chip (tooltip evergreenTactic), if evergreen.
  - The priority chip and the type chip.
  - A **status select** with Backlog, Planned, In progress, Launched. It is hard-coded and ignores custom board columns. Changing it sends `PUT {status}`; it is disabled when read-only or busy.

**Sections, top to bottom** (each with an uppercase 12px title and optional InfoTip / trailing value):

1. **Approval** (tip approvalLifecycle).
   - **`approvalStatus` is Created or Declined:**
     - If fields are missing: "Fill in before submitting: **{list}**. (Required fields never block saving.)"
     - Otherwise: "Ready to submit for approval.", or for Declined: "Declined — address the comment below and re-submit."
     - Editors get a PeoplePicker "Approver (optional)" and **Submit for approval**, which is disabled while busy or while fields are missing. It calls `POST /api/tactics/:id/submit {approver, approverId}`.
   - **Submitted:**
     - "Waiting on **{approver || 'a team lead or manager'}** · submitted by {submittedBy}."
     - If the user can approve and is not read-only: a comment field "Comment (required to decline)", **Approve** (`POST /api/tactics/:id/approve {comment}`), and **Decline** (red outlined; disabled without a comment; `POST /api/tactics/:id/decline {comment}`). The comment clears on success.
     - Otherwise: "Only {approver || 'a team lead, manager or admin'} can approve this."
   - **Approved:** "Approved[ · now {lifecycle}]. Changing the planned cost sends it back for approval."
   - **History:** the last 6 entries, newest first, with a left border. Each shows "**{action}** by {by} · {datetime}", and the comment in quotes when there is one.

2. **Details.** A 2-column meta grid:
   - Owner, Quarter, Channel (or "—"), Planned cost (`fmtMoneyFull(budget)` or "—"), Start, End / due ("Ongoing (evergreen)" when evergreen), and each custom field (`customAttrs[f.id]`, or "—").
   - Then "Goal: …", the description, and a Progress bar with a "{progress}%" label.

3. **Goals** (tip waterfallGoals; the trailing value is total revenue).
   - Number fields Inquiries, MQLs, TQLs, Opportunities, Closed won and Revenue.
   - Only **Inquiries** is editable unless `goals.override` is set.
   - Changes save on blur with `PUT {goals}`.
   - The checkbox "Override — type the downstream numbers yourself instead of using the model" saves immediately.

4. **Line items** (tip tacticLineItem; trailing value: the line total).
   - Three money boxes: Planned cost, Line items, and Sys_Gen_Balance (= budget − Σ amount; red if < 0).
   - **List rows:**
     - Name, amount, and Edit / Delete icons for editors.
     - **Delete is immediate, with no confirmation:** `DELETE /api/tactics/:id/line-items/:liId`.
     - Underneath: "{mech} · Monthly", the period range, "· vendor", "· actual $X", and "· not linked to a budget".
   - Empty: "No line items yet — add one to plan this tactic's spend month by month."
   - **"+ Add line item"** opens LineItemDialog. Its `onSaved` re-fetches the tactic and notifies the page.

5. **Media codes** (tip: "A unique tracking tag (like a UTM) attached to a campaign link so you can tell exactly which tactic drove a visit or conversion.").
   - Rows show the code in monospace, the channel, and a copy icon (`navigator.clipboard`).
   - Empty: "No media codes generated."
   - Editors get an input with placeholder "Channel (default: {channel || 'GEN'})" and a **Generate** button, which calls `POST /api/tactics/:id/media-codes {channel: input || t.channel}`.

6. **Performance — plan vs. actual** (tip: "The results you expected (plan) next to what actually happened (actual) — leads, pipeline and revenue — so you can see if the tactic delivered.").
   - Three boxes (leads, pipeline, revenue), each with Plan and Actual number fields.
   - Typing updates local state; blur saves `PUT {performance}`.
   - Footer: "Actual pipeline X vs plan Y".

7. **Activity.**
   - Comments with author, time and text. Empty: "No activity yet."
   - Input "Add a comment…": Enter or **Post** calls `POST /api/tactics/:id/comments {author: user.name || 'You', text}`.

**Footer** (editors only):
- **Delete** (red) calls the page's `remove(id)`: `DELETE /api/tactics/:id` **with no confirmation**, then the drawer closes and the toast reads "Tactic removed."
- **"Edit all fields"** (outlined) closes the drawer and opens the tactic editor dialog.

### 8.11 Line item dialog (`LineItemDialog.jsx`), the fifth level

**Opened from:** the Grid (tactic "+" and line-item edit) and the TacticDrawer.

**Title:** "New line item" or "Edit line item", with the sub-line `Under tactic "{title}"`.

**Defaults (`blank`):**

```
name `${tactic.title}_Line Item`, category 'Media', vendor '', owner (tactic owner unless 'Unassigned'),
ownerId, currency 'USD', amount '', segment 'month', mechanism 'E', allocations {}, actual '',
budgetPoolId '', description ''
```

**On open:**
- The error and the extra-years counters are reset.
- The form loads from the existing line item, or from the defaults.
- `GET /api/budgets/pools` fills the pool list. On failure the list is empty.

**Fields:**
- **Name.** Autofocus.
- **Category.** Free text.
- **Vendor.**
- **Owner.** PeoplePicker.
- **Currency.** Select of the 8 codes.
- **SPREAD** (InfoTip spreadMechanism):
  - A Monthly / Quarterly toggle. Changing it clears `allocations`.
  - Mechanism buttons **C F L E** (bold), with tooltips Custom, First period, Last period, Evenly distributed, followed by the mechanism's full label as text.
- **Total planned cost.** A number field when the mechanism is not C. For C it is a disabled field showing the sum of the periods, with helper "Sum of the periods below".
- **Actual spend.** Number, min 0.
- **Evergreen notice.** Shown when the tactic is evergreen and the mechanism is E or L: "This tactic is evergreen (no end date), so {mech} puts the total in the first period. Switch to Custom if that's wrong." The preview then uses F.
- **Custom (C) grid:**
  - Buttons **"− Year"** and **"+ Year"** extend the grid by a whole year before or after the tactic's span.
  - Centre text: "Costs can be planned before or after the tactic's dates".
  - A grid of number inputs, 3 or 4 per row, scrolling past 220px height, one per period.
  - Periods offered: the tactic's span (or the current year if the tactic has no dates), plus the extra years, plus any periods that already have allocations.
  - Inputs outside the tactic's span get a chip-coloured background.
- **Non-custom preview:** read-only chips "Mar 2026 · $X" from `spread()`, or the text "Give the tactic start and end dates to preview the spread — until then the total lands in the tactic's quarter."
- **Link to budget.** Select with "*Not linked — lands in the budget's unallocated balance*" plus the pools, shown as "{name} · {currency}".
- **Description.** Multiline.
- **Summary bar:** "Tactic planned cost **$X**" and "Sys_Gen_Balance after save (i) **$Y**". Y = tactic budget − the other line items − this total, and is red if negative.

**Buttons:** Cancel, and **"Save line item"** ("Saving…"), disabled when the name is empty. Closing is blocked while busy.

**Payload:**

```
{name, category, vendor, owner, ownerId, currency, segment, mechanism,
 amount: (C ? customTotal : Number(amount)),
 allocations: (C ? {period: round(n)} for n > 0 within the offered periods : {}),
 actual: Number, budgetPoolId: id | null, description}
```

**Endpoints:**
- Create: `POST /api/tactics/:tacticId/line-items`.
- Edit: `PUT /api/tactics/:tacticId/line-items/:id`.
- Then `onSaved(res.tactic)` and close.
- Server errors (for example "Block negative balance") appear in an error Alert inside the dialog.

### 8.12 Jira / Trello import (`ImportBoardDialog.jsx`)

**Dialog:** title "Import a board", close X, `maxWidth sm`.

**On open:**
- error, result, boards, projects and confirmId are reset.
- `GET /api/import/connections` returns `[{id, provider: 'jira'|'trello', name, liveCount, importedAt}]`.

#### 8.12.1 Form state

1. **Connected boards** (when there are any):
   - Section title "CONNECTED BOARDS".
   - Each board has a provider tile: "J" on Jira blue `#0052cc`, or "T" on Trello blue `#0079bf`.
   - Name, then "{liveCount} tactic(s) on the board · imported {date}".
   - **Disconnect** (red outlined, link-off icon) switches to an inline confirmation: **"Remove {liveCount}"** (red contained; "Removing…") and Cancel.
   - While confirming, a warning shows: "Disconnecting permanently removes that board's tactics from your board. This can't be undone."
   - Confirming calls `DELETE /api/import/connections/:id`, which returns `{deleted}`. The list reloads, and the page receives `onImported({imported: 0, disconnected: true, deleted, board: name})`.

2. **Source toggle:** **Trello** (default) or **Jira**. Switching clears the error.

3. **Trello:**
   - Help: "Get your key & token at trello.com/app-key (see the setup guide for the exact steps)." The link opens in a new tab.
   - Fields: **API key** and **Token**. These are plain text fields, not password fields.
   - **"Load boards"** (or "Reload boards"), with a spinner while busy. It is disabled until both fields are filled.
     - Calls `POST /api/import/trello/boards {key, token}`, which returns `[{id, name}]`.
     - The first board is auto-selected.
     - An empty result shows "No boards found for this token."
   - **Board** select.

4. **Jira:**
   - Help: "Create an API token at id.atlassian.com (see the setup guide)."
   - **Site URL.** Placeholder "https://your-domain.atlassian.net".
   - **Account email.**
   - **API token.** Plain text.
   - **Load projects.**
     - First saves `{baseUrl, email}` (never the token) to `localStorage.zr_import_jira_prefs`, which pre-fills these fields next time.
     - Then calls `POST /api/import/jira/projects {baseUrl, email, apiToken}`, which returns `[{key, name}]`.
     - An empty result shows "No projects found for this account."
   - **Project** select, shown as "{name} ({key})".
   - **"Advanced: JQL (optional — overrides project)"**, placeholder `e.g. project = ABC AND sprint in openSprints()`.

5. **Footer of the form:**
   - The error Alert, if any.
   - Footnote: "Cards/issues become tactics; their columns/statuses map to your board columns. Credentials are used only for this request and are never stored."
   - Buttons: Cancel, and **Import** ("Importing…" with a spinner).
   - Import is disabled for Trello until key, token and board are all set, and for Jira until site URL, email and token are set plus either a project or JQL.

**Import:**
- **Trello:** `POST /api/import/trello {key, token, boardId}`.
- **Jira:** `POST /api/import/jira {baseUrl, email, apiToken, projectKey (omitted when JQL is given), jql}`.
- Both return `{imported, source, board|project, errors[], ids[]}`.
- Then `onImported(res)` and the connections reload. If `imported` is 0, the error shows `errors[0]` or "Nothing was imported."

#### 8.12.2 Success state
- A green check icon, "Imported {N} card(s)", and "from Jira ({project}) / Trello ({board}) · {k} skipped — they're on your board now."
- Footer:
  - **Undo import** (red outlined, undo icon). It calls `POST /api/planning/tactics/bulk-delete {ids: result.ids}`, then `onImported({undo: true})`, then closes.
  - **Done.**

### 8.13 Plan Copilot: closed-loop AI plan (`PlanCopilot.jsx`). Currently not mounted anywhere.

It used to be the "AI Copilot" tab. **Props:** `canEdit`, `onCommitted` (was `refresh`), `onToast`.

**Form defaults:**
- name "FY plan — pipeline push"
- goalLeads 5000
- budget 400000
- quarter "Q4 2026"
- Quarter options: Q1 2026, Q2 2026, Q3 2026, Q4 2026, Q1 2027, Q2 2027.

**On mount:** `loadLoops()`.
1. `GET /api/planloop/list` returns `[{id, name, quarter}]`.
2. It picks the requested loop, else the current one, else the first.
3. Then `loadStatus(id)`, which calls in parallel `GET /api/planloop/status?loopId=` and `GET /api/planloop/recommendations?loopId=`.
4. Errors are swallowed silently.

**Layout:**

1. **Intro tile.**
   - A gradient sparkle icon and **"Closed-loop planning"**.
   - Text: "Set a goal and budget — the copilot drafts a plan from your real channel performance, puts it on the board, then tracks pace-to-goal and recommends corrections as work happens."

2. **"1 · Generate a plan"** (`canEdit`).
   - **Plan name** (full width).
   - **Goal (leads):** number, min 1.
   - **Budget ($):** number, min 1000, step 10000.
   - **Quarter:** select.
   - **Generate plan** button (sparkle icon; shows "Thinking…" with a spinner).
   - Generate sends `POST /api/planloop/generate {name (or 'AI marketing plan' when blank), goalLeads, budget, quarter}`.
   - It returns the preview: `{goalLeads, rationale, projection: {onTrack, attainmentPct, leads}, tactics: [{channel, budget, targets: {leads}, cpl, startDate, endDate}]}`.

3. **"2 · Proposed plan"** (when a preview exists).
   - **Badge:** "Meets goal" (green) or "Below goal" (amber), followed by "· {attainmentPct}%".
   - **Rationale** paragraph with a violet sparkle icon.
   - **Half-dial gauge** (ECharts): max = max(120, attainment rounded up to the next 20), coloured green or amber.
   - Caption: "{projected} projected / {goal} goal".
   - **Table:**
     - **Channel** (with a SERIES colour dot)
     - **Budget**
     - **Target leads**
     - **CPL**
     - **Window** (`MM-DD → MM-DD`)
   - **Buttons** (`canEdit`):
     - **Add plan to board** (bolt icon; "Adding…"). Calls `POST /api/planloop/commit {preview}`, which returns `{loopId, tacticIds}`. Toast "Plan added — {N} tactics on the board.", then the preview clears, `onCommitted()` runs, and `loadLoops(loopId)` reloads.
     - **Discard** clears the preview.
     - **Regenerate** generates again.

4. **"3 · Live plan — pace to goal"** (when at least one loop exists).
   - **Header controls:**
     - A loop select showing "{name} · {quarter}" (pill-shaped, minimum 200px).
     - **Refresh**.
   - **Loading:** a spinner.
   - **When `status.rollup` exists:**
     - **Four tiles:**
       - **Attainment gauge:** coloured green for on-track, `#c77700` for at-risk, red otherwise; caption "attainment of goal".
       - **Projected leads:** sub-line "goal {goalLeads}".
       - **Budget committed:** sub-line "of {plannedBudget} · {budgetPacePct}%".
       - **At risk:** the count, red when above zero; sub-line "of {tacticCount} tactics".
     - **"Recommended corrections":**
       - When there are none: "Nothing to correct — the plan is pacing to goal. 🎯"
       - **Each recommendation card shows:**
         - an icon: trending-up for `reallocate`, bolt otherwise; red tint for `overspend`
         - the title and detail text
         - a green **impact** chip
         - **Apply** (`canEdit` and the recommendation has an `apply` action; shows "…" while running)
       - **Apply** calls `POST /api/planloop/apply {loopId, apply}`, which returns `{ok, moved, priority, error}`.
       - **On success:**
         - The toast reads "Moved {fmtMoney(moved)} — plan rebalanced." for a reallocation, otherwise "Priority raised to {priority}."
         - Then `onCommitted()` and `loadStatus()`.
       - When `ok` is false, the error toast shows `error`, or "Could not apply." if there is none.
     - **"Tactics" table:**
       - **Tactic:** the title, plus the channel in small text.
       - **Pace:** a 96px bar filled to `progress%` in the health colour, with a vertical marker at `expectedPct` (tooltip "expected X%") and the label "p% / e%".
       - **Target**
       - **Projected:** bold.
       - **Budget**
       - **Health chip:** planned "Planned" `#8a8a8e`; on-track "On track" green; at-risk "At risk" `#c77700`; behind "Behind" red; overdue "Overdue" `#b3261e`.
     - Footnote: "The vertical mark on each pace bar is where the tactic **should** be today; the fill is actual progress. Update a tactic's progress on the board and the projection re-learns."
   - **When there is no roll-up:** "No live plan yet — generate one above and add it to the board."

**Gauge settings** (`gaugeOption`): a half-circle 180→0°, centre (50%, 76%), radius 116%, progress width 14 with rounded caps, a 14%-alpha track, no pointer, ticks or labels, and a detail label of "{n}%" at 26px weight 800.

### 8.14 Legacy flat grid (`PlanGrid.jsx`). Currently not mounted.

**Purpose:** this was the original "Grid" tab, a spreadsheet of tactics.

**Props:** `tactics, canEdit, columns, onUpdate, onBulk, onBulkDelete, onDeleteAll, onOpen`.

**Status options:** the board columns, falling back to Backlog / Planned / In progress / Launched.

**Toolbar:**
- **With a selection** (editors): "{N} selected", a status select, **"Apply to selected"** (calls `onBulk(ids, {status})`), and **"Delete selected"** (calls `onBulkDelete(ids, clear)`).
- **Otherwise:** "{N} tactics".
- **Right side:**
  - **Delete all**, for editors when tactics exist.
  - **Export CSV** to `zenreach-plan.csv` with columns Title, Type, Owner, Quarter, Status, Priority, Budget, Progress, Channel.

**Table** (`min-w 820`):
- **Select-all checkbox column**, editors only.
- **Tactic:** a link that opens the drawer.
- **Type:** inline select (Campaign, Launch, Content, Event, Research, Ops).
- **Owner:** inline text.
- **Quarter:** inline text.
- **Status:** inline select.
- **Priority:** inline select (P0, P1, P2).
- **Budget:** inline number field, displayed with `fmtMoney`.
- **Progress:** read-only.

**Inline editing (`InlineText`):**
- Clicking turns the cell into a borderless (standard-variant) text field.
- Enter commits by blurring; Escape cancels.
- Blur saves only when the value changed.
- Selects save on change.

**Empty:** "No tactics."

### 8.15 Client exporters and file helpers

#### 8.15.1 `lib/exporters.js`
- **`download(filename, text, mime)`:** creates a Blob, an object URL and a temporary `<a download>`, clicks it, then revokes the URL.
- **`exportTacticsIcs(tactics, calName='ZenReach Marketing Calendar')`:**
  - Writes a VCALENDAR with `VERSION 2.0`, `PRODID:-//ZenReach//Planning//EN`, `CALSCALE:GREGORIAN` and `X-WR-CALNAME`.
  - One all-day VEVENT per tactic that has a valid date:
    - `UID:{id}@zenreach`
    - `SUMMARY:{title}`
    - `DTSTART;VALUE=DATE:` from `startDate || endDate || dueDate`
    - `DTEND;VALUE=DATE:` from `endDate || dueDate || startDate` **plus one day** (iCal's end date is exclusive)
    - `DESCRIPTION:{type} · {owner} · {channel} · {goal}`
  - Escapes `, ; \` and newlines.
  - Lines are joined with CRLF.
  - Downloads as `zenreach-calendar.ics` (`text/calendar`).
- **`exportCsv(filename, rows, columns)`:**
  - Each column is `{label, key}` or `{label, get(row)}`.
  - Values are quoted when they contain `"`, a comma or a newline, with embedded quotes doubled.
  - Lines are joined with LF. MIME type `text/csv`.
- **`printElement()`:** `window.print()`. The calendar toolbar and the unscheduled section are hidden through `print:hidden`, and the grid is marked `.calendar-print`.

#### 8.15.2 `lib/files.js` (not used by Planning)
- **`readUpload(file)`:** returns a Promise that resolves to `{csv, name}` for `.csv`, `.txt` and `.tsv` files (read as text), or to `{fileBase64, name}` for anything else (read with `readAsDataURL`, keeping the part after the comma). A read error rejects with "Could not read the file."
- **`UPLOAD_ACCEPT = '.csv,.tsv,.txt,.xlsx,.xls,.ods,text/csv'`.**
- Used by `Integrations.jsx` and `Builder.jsx`.

### 8.16 Index of API calls made by the Planning frontend

| Action | Client fn | HTTP |
|---|---|---|
| Hierarchy tree | `planTree` | GET `/api/plans/tree` |
| Board columns + tactics | `planningBoard(filter)` | GET `/api/planning/board?planId\|campaignId\|programId` |
| Summary stats | `planningSummary(filter)` | GET `/api/planning/summary?…` |
| Board config | `boardConfig` | GET `/api/planning/board-config` |
| Planning meta (models, stages, teams) | `planMeta` | GET `/api/plans/meta` |
| Save columns | `setBoardColumns` | PUT `/api/planning/board-config/columns {columns}` |
| Save custom fields | `setBoardFields` | PUT `/api/planning/board-config/fields {fields}` |
| Apply template | `applyBoardTemplate` | POST `/api/planning/board-config/template {template}` |
| Create node | `createNode` | POST `/api/plans/nodes {level, parentId, …}` |
| Update node | `updateNode` | PUT `/api/plans/nodes/:id` |
| Delete node (cascading, soft) | `deleteNode` | DELETE `/api/plans/nodes/:id` → `{removedIds, removedTactics}` |
| Node comment | `nodeComment` | POST `/api/plans/nodes/:id/comments {text}` |
| Update model | `updateModel` | PUT `/api/plans/models/:id` |
| Plan register | `campaignPlans` | GET `/api/planning/plans` |
| Plan detail | `campaignPlan` | GET `/api/planning/plans/:spa\|bcpir:<campaignId>` |
| Submit campaign plans | `planSubmit` | POST `/api/planning/plans/:campaignId/submit {approver, approverId}` |
| Approve | `planApprove` | POST `/api/planning/plans/:campaignId/approve {}` |
| Reject | `planReject` | POST `/api/planning/plans/:campaignId/reject {note}` |
| Get tactic | `tactic` | GET `/api/tactics/:id` |
| Create tactic | `createTactic` | POST `/api/tactics` |
| Update tactic (status, dates, goals, performance…) | `updateTactic` | PUT `/api/tactics/:id` |
| Delete tactic | `deleteTactic` | DELETE `/api/tactics/:id` |
| Bulk delete / delete all | `deleteTactics` / `deleteAllTactics` | POST `/api/planning/tactics/bulk-delete {ids}` / `{all: true}` |
| Bulk move / copy / submit | `moveTactics` / `copyTactics` / `submitTactics` | POST `/api/planning/tactics/bulk-move\|bulk-copy\|bulk-submit` |
| Submit / approve / decline a tactic | `submitTactic` / `approveTactic` / `declineTactic` | POST `/api/tactics/:id/submit\|approve\|decline` |
| Line items | `addLineItem` / `updateLineItem` / `deleteLineItem` | POST / PUT / DELETE `/api/tactics/:id/line-items[/:liId]` |
| Tactic comment | `addComment` | POST `/api/tactics/:id/comments {author, text}` |
| Media code | `addMediaCode` | POST `/api/tactics/:id/media-codes {channel}` |
| Budget pools | `budgetPools` | GET `/api/budgets/pools` |
| Raise PR | `prCreate` | POST `/api/procurement/requests` |
| Trello boards / import | `trelloBoards` / `importTrello` | POST `/api/import/trello/boards`, POST `/api/import/trello` |
| Jira projects / import | `jiraProjects` / `importJira` | POST `/api/import/jira/projects`, POST `/api/import/jira` |
| Connections / disconnect | `boardConnections` / `disconnectBoard` | GET `/api/import/connections`, DELETE `/api/import/connections/:id` |
| Google Calendar | `googleCalendarStatus` / `googleCalendarSync` / `googleCalendarDisconnect` | GET `/api/integrations/google-calendar/status`; POST `…/sync {calendarId, planIds}`; POST `…/disconnect` |
| .ics feed (public) | URL only | GET `/api/planning/calendar.ics?planIds=` |
| Plan loop (Copilot) | `planLoopGenerate` / `planLoopCommit` / `planLoops` / `planLoopStatus` / `planLoopRecommendations` / `planLoopApply` | POST `/api/planloop/generate`, POST `/commit`, GET `/list`, GET `/status?loopId`, GET `/recommendations?loopId`, POST `/apply {loopId, apply}` |
| People directory (PeoplePicker) | `orgPeople('')` | Loaded once, cached at module level |

### 8.17 Persistence: localStorage keys and URL parameters

| Key / param | Written by | Value | Effect |
|---|---|---|---|
| `zr_planning_view` | Planning | `plans\|grid\|board\|calendar\|budget` | Remembers the last tab. Invalid → `plans`. |
| `zr_planning_plans` | Planning | `'all'` or one plan id | Remembers the plan chip. Read synchronously; resets to `all` if the plan is gone. |
| `zr_planning_groupby` | Planning | `none\|owner\|campaign\|priority\|type` | Board swimlanes |
| `zr_planning_views` | Planning | JSON `[{id, name, view, groupBy, nodeId, nodeLevel}]` | Saved views (per browser) |
| `zr_import_jira_prefs` | ImportBoardDialog | JSON `{baseUrl, email}` | Pre-fills the Jira form. The token is never saved. |
| `zr_token` | api.js | Bearer token | Authentication for every call |
| `?plan=spa:<campaignId>` / `?plan=bcpir:<campaignId>` | PlansView (`history.replaceState`) | Deep link to the plan detail | Read only when PlansView mounts |
| `?campaignId=` | Outgoing links from the detail page | — | Scoped links to `/budget` and `/procurement` |

**Not persisted:** Grid and Budget expand state, the Grid search box, honeycomb selection, register group collapse, the calendar month, the Budget segment and year, and the Tree's open/closed state.

### 8.18 Quirks and decisions for the rebuild

1. **The `?plan=` deep link** only works when the remembered tab is Plans. Force the Plans view when the param is present.
2. **The detail page's "Board" link** points at `/planning`. It neither closes the detail nor switches to the Board tab.
3. **The board bulk "Move to column"** does nothing until the status select is changed, because the initial state is `''`.
4. **Delete wording is inconsistent.** The PlansView dialogs and the CampaignPlanDetail "Remove" dialog say "permanently / cannot be undone". The Planning page dialogs and the server say the item goes to the recycle bin.
5. **Plan and campaign edits from the register send `description: ''`**, wiping any description set in the Inspector.
6. **The campaign dialog's Status select** lets any editor set Submitted or Published directly, skipping the approval flow. Its add button is labelled "Create plan", and the campaign-row edit tooltip says "Edit plan".
7. **Register delete buttons are shown to every editor**, but the server requires an admin, so non-admins get an error toast. The Grid and Inspector gate this correctly with `canAdmin`.
8. **Some deletes skip confirmation:** the board card trash, the TacticDrawer Delete button and the drawer's line-item trash. The Grid confirms both tactic and line-item deletes.
9. **TacticDrawer's status select** is hard-coded to the four default statuses, so a custom board column shows a blank value.
10. **Editing a BCPIR row** overwrites `performance` with revenue and leads only, dropping pipeline values.
11. **Saved views** don't store the plan chip, and can only be saved from the Board or Calendar tabs.
12. **A node selected on Board or Calendar** stays active on Grid and Budget, where the sidebar is hidden. The summary strip is then scoped to that node while the grid shows the whole plan.
13. **Every filter change shows a full-body spinner**, tabs included. The Plans tab also triggers the board, tree and summary fetches even though it doesn't use them.
14. **The calendar is month-only** and places each tactic on its start day only. Dropping an evergreen tactic gives it an end date.
15. **The `.ics` feed requires no sign-in**, and its URL is shareable.
16. **The Trello and Jira token fields** are plain text inputs.
17. **Board settings saves only the active tab**, and the dialog closes even if the save fails (the failure appears as a toast).
18. **Dragging a card into another swimlane** changes only its status.
19. **`canCustomize` excludes `orgRole: 'admin'`** unless the user is also a workspace Owner or Admin.
20. **Dead code:** `gridUpdate`, `askDeleteAll` and the "all" branch of the bulk-delete confirmation in Planning.jsx. `PlanGrid.jsx` and `PlanCopilot.jsx` are not mounted.
21. **The plan selector is hidden** when there is one plan or none. The SP&A and BCPIR colours are fixed (SERIES[0] and SERIES[4]); plan colours depend on tree order, so they shift when plans are added or removed.
22. **The register Reject action** sends no note, although the API accepts `note`.

## 9. Finance — budgets, procurement and multi-currency

**Source files:** `server/services/budgets.js`, `server/services/procurement.js`, `server/services/fx.js`, `client/src/pages/Budget.jsx`, `client/src/pages/Procurement.jsx`, `client/src/components/ScopePicker.jsx`, `client/src/components/SnapshotPanel.jsx`, `client/src/lib/snapshotStore.js`, `client/src/lib/snapshot.js`, `client/src/context/CurrencyContext.jsx`, `client/src/components/CurrencyControl.jsx`, `client/src/lib/format.js`.

### 9.0 Shared infrastructure the finance modules rely on

**Persistence (`server/services/store.js`)**
- Storage is a flat JSON file store in `server/data/`. The `ZR_DATA_DIR` env var overrides the directory. The directory is created on demand.
- `readJson(file, fallback)` is synchronous. It returns `fallback` if the file is missing, empty or fails to parse.
- `writeJson(file, data)` writes pretty JSON (2-space indent) synchronously.
- `uid(prefix)` returns `` `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2,7)}` ``, e.g. `pr-mtha524e-p5zas`. The middle segment is the base-36 timestamp, and PO/invoice numbers are derived from it.
- `nowIso()` returns `new Date().toISOString()`.
- Finance files:
  - `budgets.json`: `{ pools[], allocations[], snapshots[] }`
  - `purchaseRequests.json`: `{ requests[] }`
  - FX rates are held **in memory only**. They are never persisted.

**API envelope**
- Success is always `{ ok: true, data }`.
- Errors are `{ ok: false, error }`, with HTTP 400 for validation, 404 for not found, 401 or 403 for auth.
- In the client wrapper (`api.js`), `get` throws `Request failed: <path> (<status>)` on a non-2xx response, or `json.error` when `ok` is false. `send` (POST/PUT/DELETE) throws `json.error`.
- `qs(obj)` drops null and empty values and URI-encodes the rest.

**Auth and permissions**
- A global auth wall returns 401 `Please sign in to continue.` on every `/api/*` route for callers without a valid bearer token. Public exceptions:
  - `POST /api/auth/login`
  - `POST /api/auth/signup`
  - `GET /api/auth/me`
  - `POST /api/auth/logout`
  - `GET /api/org/personas`
  - **`GET /api/currency`** (FX is public so rates load before sign-in)
  - `GET /api/health`
  - `GET /api/planning/calendar.ics`
- `requireRole('Editor')` gates every finance write:
  - Role rank order is Owner > Admin > Editor > Viewer.
  - If there are 0 users, or no token, the request passes with `req.user = null` (demo bootstrap). In practice the auth wall already requires a token.
  - An invalid token gets 401 `Your session has expired — please sign in again.`
  - A rank below Editor gets 403 `This action requires the Editor role or higher.`
  - On success `req.user` is set.
- Client mirror: `canEdit = !user || !['Viewer'].includes(user.role)`.

**Routing and navigation**
- `/budget` is in the MAIN sidebar group, label "Budget", `PaidRounded` icon.
- `/procurement` is in the "More" group, label "Procurement", `ReceiptLongRounded` icon.
- Both routes exist only when signed in. Logged-out users are redirected to `/`.

### 9.1 FX service (server): `server/services/fx.js`

#### 9.1.1 Constants
- `CURRENCIES = ['USD','EUR','GBP','CAD','AUD','INR','JPY','SGD']`. These 8 are the supported currencies.
- Rate convention: **"USD per 1 unit of the currency"**. USD = 1.
- `FALLBACK` static table (also duplicated client-side):
  `{ USD: 1, EUR: 1.08, GBP: 1.27, CAD: 0.73, AUD: 0.66, INR: 0.012, JPY: 0.0064, SGD: 0.74 }`
- `MAX_AGE_MS = 60 * 60 * 1000`. The cache refreshes at most hourly unless forced.

#### 9.1.2 Module state (in-memory cache)
- `rates` starts as a copy of FALLBACK.
- `updatedAt` starts as `null`.
- `source` starts as `'fallback (static)'`.
- `inflight` holds the single shared fetch promise (de-duplication).

#### 9.1.3 `fetchLive()`
- `GET https://api.frankfurter.app/latest?from=USD&to=EUR,GBP,CAD,AUD,INR,JPY,SGD`, with `AbortSignal.timeout(6000)` (6 s timeout). No API key; these are ECB reference rates.
- A non-OK response throws `FX provider returned <status>`.
- Frankfurter returns "units per USD" (e.g. EUR 0.87). Each one is stored as `1 / unitsPerUsd`, only when the value is > 0. `USD` is set to 1.
- Any currency missing from the response is filled from FALLBACK.
- Sets `rates`, `updatedAt = now ISO`, `source = "frankfurter.app · ECB <json.date>"` (trimmed).

#### 9.1.4 `getRates({ force=false })` (async)
- Data is stale when `!updatedAt` or older than 1 h.
- If forced or stale, it starts or joins the single `inflight` fetch.
  - A failure is swallowed (`.catch(() => rates)`), so the last-known or fallback rates are kept.
  - `inflight` is cleared in `finally`.
- Returns `{ base: 'USD', currencies: CURRENCIES, rates, updatedAt, source }`.
- Quirk: if the network never succeeds, `updatedAt` stays `null`. Every call then re-attempts the fetch, which can mean up to a 6 s wait per call.

#### 9.1.5 Other exports
- `ratesSync()` returns the current `rates` object.
- `convert(amount, from='USD', to='USD')` (synchronous, uses the current cache):
  `usd = (Number(amount)||0) * (rates[from] || 1); return usd / (rates[to] || 1)`.
  Unknown codes are treated as rate 1. The result is **not rounded**.
- On module load: `getRates().catch(()=>{})` warms the cache, best effort.

#### 9.1.6 Route
`GET /api/currency[?refresh=1]` (public):
- Returns `getRates({ force: refresh==='1' })`.
- On an exception it still returns 200: `{ base:'USD', currencies, rates:{}, source:'unavailable' }`.

### 9.2 Budgets service: `server/services/budgets.js`

#### 9.2.1 Data model (`budgets.json`)

**Pool** (top-down money assigned to a plan)

| field | type / rule |
|---|---|
| `id` | `uid('pool')` |
| `name` | string, max 120 chars; default on create `'New budget pool'` |
| `planId` | plan node id or `null`; set on create only, never updated |
| `currency` | 3-letter, upper-cased (`slice(0,3).toUpperCase()`); default `'USD'` |
| `total` | integer ≥ 0 (`Math.max(0, Math.round(Number(x)))`); default 0 on create |
| `createdAt`, `updatedAt` | ISO strings |

**Allocation** (a pool's money for one node, for one period)

| field | type / rule |
|---|---|
| `id` | `uid('alloc')` |
| `poolId` | pool id; **not validated** |
| `nodeId` | plan-hierarchy node id; validated via `getNode` (any level), but only campaign-level allocations are ever read by the rollup |
| `period` | free string, e.g. `'Q1 2026'` |
| `amount` | integer ≥ 0, rounded |

Allocations are unique per `(poolId, nodeId, period)`. `setAllocation` upserts on that key.

**Snapshot (server-side JSON; exists but no UI uses it)**

| field | type / rule |
|---|---|
| `id` | `uid('snap')` |
| `label` | max 80 chars; default `` `Snapshot ${db.snapshots.length + 1}` `` |
| `at` | ISO |
| `currency` | rollup currency |
| `totals` | `{ planned, committed, variance, utilizationPct }` |
| `rows` | `[{ name, planned, committed }]`, a portfolio (unscoped) rollup |

#### 9.2.2 Constants
- `POOL_TOTAL = 900000`
- `QUARTERS = ['Q1 2026','Q2 2026','Q3 2026','Q4 2026']`
- `SPLIT = { 'Nova GA Launch': 0.5, 'Enterprise ABM': 0.3, 'Brand & Community': 0.2 }`, keyed by campaign **name**. Any other campaign falls back to a share of `1 / campaigns.length`.
- `quarterOf(tactic) = tactic.quarter || 'Unscheduled'`.

#### 9.2.3 `buildAllocations(pool, campaigns)`
For each campaign:
- `campTotal = pool.total * share`.
- For each of the 4 QUARTERS, push `{ id, poolId: pool.id, nodeId: c.id, period: q, amount: Math.round(campTotal / 4) }`.

Shares are not normalised. If an extra campaign exists at seed time, total allocations exceed the pool (see seed note below).

#### 9.2.4 Seed (when `budgets.json` is missing or has no `pools` array)
- Takes the first `level==='plan'` node and all `level==='campaign'` nodes.
- One pool:
  - `name 'FY2026 Marketing Budget'`
  - `planId` = that plan
  - `currency` = plan.currency or `'USD'`
  - `total 900000`
- Allocations from `buildAllocations` (against **all** campaigns). Empty `snapshots[]`.

With the base plan seed (plan "FY2026 Marketing Plan" with 3 campaigns), you get **12 allocations**:

| campaign | share | annual | per quarter (Q1–Q4 2026) |
|---|---|---|---|
| Nova GA Launch | 0.5 | 450,000 | 112,500 |
| Enterprise ABM | 0.3 | 270,000 | 67,500 |
| Brand & Community | 0.2 | 180,000 | 45,000 |

**Seed-order caveat:**
- `ensureMultiPlanDemo()` runs at server start and creates a second plan, "EMEA Field Marketing" (fiscalYear FY2026, USD), with campaign "EMEA Demand Gen" and 4 tactics: 18k (Q3), 60k, 45k, 12k (Q4).
- The budgets file is seeded lazily on first use. On a **fresh** data directory, EMEA therefore exists at seed time and gets a share of 1/4 = 0.25 → 225,000 (56,250 per quarter). That makes total allocations 1,125,000, more than the 900,000 pool.
- In the current persisted data, budgets were seeded before EMEA existed. EMEA (and a user-made campaign "Adi") have **no** allocations: planned 0.

#### 9.2.5 `load()`: self-healing of stale allocations
1. Read the DB. Seed and write if it is missing or invalid. Default `snapshots` to `[]`.
2. Collect the current campaigns. If any exist, and **no** allocation's `nodeId` matches any current campaign id (e.g. the hierarchy was re-seeded with new ids), rebuild:
   - `db.allocations = buildAllocations(db.pools[0], campaigns)`
   - write the DB.
3. This is an all-or-nothing check. Partial drift (e.g. a new campaign added) is not healed, and new campaigns stay at planned 0. The heal uses `pools[0]` and does not update `pool.planId`. If pools were ever empty it would throw.

#### 9.2.6 Rollup algorithm: `rollup(currency, filter = {})`
1. `cur = (currency || plan.currency || 'USD').toUpperCase()`, where plan is the first plan node.
2. **Scoping:**
   - Start with all campaign nodes.
   - If `filter.campaignId`: keep only that campaign. `scope = { campaignId, name: campaign.name || 'Campaign' }`.
   - Else if `filter.planId`: keep campaigns with `parentId === planId` (direct children). `scope = { planId, name: plan.name || 'Plan' }`.
   - `campaignId` wins if both are given.
   - With no filter, `scope = null`, which is the portfolio view.
3. `quarters` = the sorted unique set of all allocation periods plus all tactics' `quarterOf()`.
   - This list is **unscoped**: columns come from all data, not just the filtered campaigns.
   - Sorting is lexicographic (`Q1 2026 < … < Q4 2026 < Unscheduled`, and `Q1 2027` would sort before `Q2 2026`).
4. Per campaign row:
   - **Planned:** allocations with `nodeId === c.id`. The pool currency is taken from the pool of the **first** allocation (default USD). Each amount is `convert(a.amount, poolCur, cur)` and accumulated into `byQuarterPlanned[a.period]` and `planned`.
   - **Committed:** for each tactic with `t.campaignId === c.id`, with `q = t.quarter || 'Unscheduled'`:
     - For each `lineItem`: `v = convert(li.amount, li.currency || 'USD', cur)`, added to `byQuarterCommitted[q]` and `committed`. Also `itemized += Number(li.amount)`. This is raw and unconverted, a quirk.
     - `remaining = max(0, tactic.budget − itemized)`, converted from **USD**, then added to the same quarter and to `committed`.
     - Net effect (all-USD case): a tactic contributes `max(budget, Σ line items)`.
     - The line items' own monthly/quarterly `allocations` are ignored. The whole tactic lands in the tactic's quarter.
     - The line item's `budgetPoolId` is **not** used by the rollup.
   - Row output:
     ```
     { campaignId, name,
       planned: round(planned), committed: round(committed),
       variance: round(planned − committed),
       utilizationPct: planned ? +((committed/planned)*100).toFixed(1) : 0,
       byQuarterPlanned, byQuarterCommitted }   // per-quarter maps are unrounded
     ```
5. `totals` = sums of the rounded row planned and committed, then:
   - `variance = planned − committed`
   - `utilizationPct = planned ? +((committed/planned)*100).toFixed(1) : 0`
6. Returns `{ currency: cur, quarters, rows, totals, pools: db.pools, scope }`.

Rollup is synchronous and uses whatever FX cache the server currently holds.

**Reference numbers (USD):**

| campaign | planned | committed |
|---|---|---|
| Nova GA Launch | 450,000 | 270,000 (paid search 120k, retargeting 60k, webinar 45k, nurture 15k in Q3; post-launch retargeting 30k in Q4) |
| Enterprise ABM | 270,000 | 90,000 (ABM plays, Q3) |
| Brand & Community | 180,000 | 77,000 (Q3: 14k + 9k + 10k + 18k = 51k; Q4: analyst 20k + customer story 6k = 26k) |

- Pristine-seed totals: planned **900,000**, committed **437,000**, uncommitted **463,000**, utilization **48.6%**.
- Current persisted data adds a 1,200 tactic ("TalentLMS") to Nova and EMEA's 135,000. That gives committed 573,200 and utilization 63.7%. The EMEA row shows planned 0, committed 135,000, utilization 0.0% (planned 0 forces 0%).

#### 9.2.7 Pools and allocations CRUD
- `listPools()` returns `db.pools`.
- `upsertPool(input)`:
  - **With `id`:** find the pool (return `null` if not found). Update:
    - `name` (`input.name ?? p.name`, max 120)
    - `total` (`max(0, round(Number(input.total ?? p.total)))`)
    - `currency` (3 chars, upper-cased)
    - `updatedAt`
    `planId` is not updatable.
  - **Without `id`:** create a pool with the defaults above.
- `setAllocation({ poolId, nodeId, period, amount })`:
  - Throws `Unknown campaign/program.` if `getNode(nodeId)` is falsy.
  - Upserts by `(poolId, nodeId, period)` with `amount = max(0, round(Number(amount)||0))`.

#### 9.2.8 Server snapshots
- `listSnapshots()` returns `[{ id, label, at, totals, currency }]` (no rows).
- `createSnapshot(label, currency)` takes an **unscoped** `rollup(currency)`, pushes the snapshot and saves.
- `getSnapshot(id)` returns the snapshot or `null`. There is no route for it.
- `deleteSnapshot(id)` filters it out and returns `{ id }`.

#### 9.2.9 Routes
| Method / path | Auth | Behaviour | Audit action |
|---|---|---|---|
| `GET /api/budgets/currencies` | signed-in | `CURRENCIES` array | — |
| `GET /api/budgets/rollup?currency=&planId=&campaignId=` | signed-in | `rollup(...)` | — |
| `GET /api/budgets/pools` | signed-in | pools | — |
| `POST /api/budgets/pools` | Editor+ | `upsertPool(body)` | `budget.pool` (label = pool name, entityId) |
| `POST /api/budgets/allocations` | Editor+ | `setAllocation(body)`; 400 with message on throw | `budget.allocation` (label = `out?.name` (always undefined), then `body.poolName`, then `body.poolId`) |
| `GET /api/budgets/snapshots` | signed-in | list | — |
| `POST /api/budgets/snapshots` `{label, currency}` | Editor+ | create | `budget.snapshot_create` (label or `'Snapshot'`) |
| `DELETE /api/budgets/snapshots/:id` | Editor+ | delete | `budget.snapshot_delete` (severity warning) |

Audit metadata: category `budget`. Verbs: "saved budget pool", "changed an allocation in budget pool", "captured budget snapshot", "deleted budget snapshot".

**Client API wrappers:** `budgetCurrencies`, `budgetRollup(currency, filter)`, `budgetPools`, `saveBudgetPool`, `setAllocation`, `budgetSnapshots`, `createSnapshot(label, currency)`, `deleteSnapshot(id)`.
- Only `budgetRollup` (Budget page) and `budgetPools` (the Planning LineItemDialog "Link to budget" select) are used by any UI.
- **No UI exists for creating or editing pools, allocations or server snapshots.**

### 9.3 Procurement service: `server/services/procurement.js`

#### 9.3.1 Constants
```
APPROVAL_CHAIN = [
  { level: 1, name: 'Marketing Ops',   threshold: 0,      mode: 'serial',   approvers: ['Dana Alvarez'] },
  { level: 2, name: 'Finance',         threshold: 25000,  mode: 'serial',   approvers: ['Marcus Lee'] },
  { level: 3, name: 'Executive (CMO)', threshold: 100000, mode: 'parallel', approvers: ['Abhishek Biyani', 'Sam Okafor'] },
]
PR_STATUS = ['draft','submitted','approved','rejected','po','invoiced','received']
```
- A level applies when `total >= threshold`. The comment says "in USD", but the PR's raw `total` is compared with **no FX conversion**.
- L1 always applies, because the threshold is 0.

#### 9.3.2 Data model (`purchaseRequests.json → { requests: [] }`)

**Purchase request**

| field | rule / default (createRequest) |
|---|---|
| `id` | `uid('pr')` |
| `title` | max 140; default `'New purchase request'` |
| `requester` | max 80; default `'You'` |
| `requesterId` | max 60; default `''` (people-picker user id) |
| `approver` | max 80; default `''`. The chosen sign-off person; drives the chain on submit |
| `approverId` | max 60; default `''` |
| `vendor` | max 100; default `''` |
| `currency` | 3 chars upper-cased; default `'USD'`. **Informational only:** totals are never converted |
| `tacticId` | kept only if `getTactic(id)` exists, else `null` |
| `campaignId` | kept only if `getNode(id)` exists, else `null` |
| `customAttrs` | object; default `{}` (no UI) |
| `lines` | normalized line array (below) |
| `total` | `lineTotal(lines)` |
| `status` | `'draft'` |
| `approvals` | `[]` until submit |
| `poNumber`, `invoiceNumber` | `''` |
| `createdAt`, `updatedAt` | ISO |
| `history` | `[{ at, by: input.requester \|\| 'You', action: 'created' }]` |

**Line** (from `normalizeLines`, capped at **40** lines):
- `{ id: l.id || uid('ln'), description: String(l.description || 'Line').slice(0,160), qty, unitPrice, amount }`
- `qty = Number(l.qty) || 1`. A qty of 0 or blank becomes 1.
- `unitPrice = Number(l.unitPrice) || 0`.
- `amount = l.amount != null ? (Number(l.amount) || 0) : qty * unitPrice`.

**`lineTotal(lines)`** = Σ `(Number(l.amount) || Number(l.qty)*Number(l.unitPrice) || 0)`.

**Approval step:**
- `{ id: uid('ap'), level, name (level name), mode ('serial'|'parallel'), approver (name), approverId ('' for the default chain), status: 'pending'|'approved'|'rejected', at: null|ISO, actedBy? }`
- `actedBy` is added when acted on.

**History entry:** `{ at: ISO, by: string, action: string }`. Actions written:
- `'created'`
- `'submitted'`
- `'approved (L<n> <Level name>)'` or `'rejected (L<n> <Level name>)'`
- `'moved to po'`, `'moved to invoiced'`, `'moved to received'`

#### 9.3.3 Approval-chain algorithms
- **`buildApprovals(total, approverName?, approverId?)`**: for each chain level where `total >= threshold`:
  - If `approverName` is given, emit **one** step for that level, assigned to `{name: approverName, id: approverId||''}`. The same person owns every applicable level, and parallel L3 collapses to a single step.
  - Otherwise emit one step per default approver of the level (ids blank). L3 therefore yields 2 steps, Abhishek Biyani and Sam Okafor.
- **`activeLevel(approvals)`** = the minimum `level` among `pending` steps, or `null`.
- **Serial vs parallel in practice:**
  - Levels are strictly sequential: only steps at the active level can be acted on.
  - Within a level, **all** steps must be approved before the next level activates, in any order.
  - `mode` is effectively a display label. A serial level only ever has one approver, and a parallel level requires all of its approvers.
- **`recompute(pr)`**:
  - If the status is `draft` or `rejected`, do nothing.
  - If any step is `rejected`, set status `rejected`. This is immediate, even if other steps are still pending; they stay `pending`.
  - Else if there is ≥1 step and every step is `approved`, and the status is `submitted`, set status `approved`.

#### 9.3.4 Status state machine
```
draft ──submit──► submitted ──(every step approved)──► approved ──Issue PO──► po ──Mark invoiced──► invoiced ──Mark received──► received
                     └──(any step rejected)──► rejected   (terminal: cannot be edited or resubmitted, only deleted)
```
- `NEXT = { approved: 'po', po: 'invoiced', invoiced: 'received' }`.

#### 9.3.5 Seed (when the file is missing or has no `requests` array)
All three seed PRs share these properties:
- `currency 'USD'`, `customAttrs {}`, empty PO and invoice numbers.
- History is only `[{at, by:'system', action:'created'}]`. Seeds don't add a "submitted" entry.
- **No** `requesterId`, `approver` or `approverId` fields.
- Approvals come from `buildApprovals(total)`, the default chain, for any non-draft status.
- Campaigns are resolved by exact name. Tactics are resolved by case-insensitive substring of the title.

| # | title | requester | vendor | status | campaign | tactic (substring → title) | lines | total | approvals |
|---|---|---|---|---|---|---|---|---|---|
| r1 | `Nova ABM ad flight (LinkedIn) — Q3` | Marcus Lee | LinkedIn | submitted | Enterprise ABM | `named-account abm` → "Top-150 named-account ABM plays" | "Sponsored content — Nova ABM (Q3)" 1×60,000; "Conversation ads — top-150 accounts" 1×18,000 | 78,000 | L1 Marketing Ops / Dana Alvarez (pending); L2 Finance / Marcus Lee (pending). Active level = 1 |
| r2 | `Nova launch webinar platform (ON24)` | Abhishek Biyani | ON24 | approved | Nova GA Launch | `launch webinar` → "'Meet Nova' launch webinar" | "Platform license for 'Meet Nova' series (annual)" 1×22,000 | 22,000 | L1 Dana Alvarez, approved with `at` = seed stamp |
| r3 | `Nova launch content retainer` | Sam Okafor | Column Five | draft | Brand & Community | `partner co-marketing` → "Partner co-marketing kits" | "Launch design + content retainer (3 mo)" 3×12,000 | 36,000 | none |

Seed summary values:
- total 3, pendingApproval 1, approvedValue 22,000, totalValue 136,000.
- byStatus: draft 1, submitted 1, approved 1, everything else 0.

Seeded personas with matching names exist in the org seed: Marcus Lee (VP Demand Gen, manager), Sam Okafor (VP Brand & Content, manager), Abhishek Biyani (Demand Gen Lead), Dana Alvarez (Lifecycle Lead). All are Editors. Demo password `zenreach`.

#### 9.3.6 Queries
- **Scope filter `filterByScope(reqs, {planId, campaignId})`**:
  - With no filter, return all requests.
  - Build `tacticId → campaignId` from all tactics.
  - **campaignId:** keep PRs where `r.campaignId === campaignId`, or where `r.tacticId` maps to that campaign.
  - **planId:** the set of campaigns whose `parentId === planId`. Keep PRs whose `campaignId` is in the set, or whose tactic's campaign is in the set.
- **`scopeInfo`** = `{campaignId, name}` (fallback name `'Campaign'`), or `{planId, name}` (fallback `'Plan'`), or `null`.
- **`listRequests(filter)`** returns the filtered list in storage order: insertion order, with restores appended at the end.
- **`getRequest(id)`** returns the request or `null`.
- **`pendingFor(approverName?)`** returns `submitted` PRs that have a pending step at the active level whose `approver === approverName` (any approver if no name is given). Served by `GET /api/procurement/pending?approver=`. **Unused by the UI.**
- **`summary(filter)`** returns:
  - `total` (count)
  - `byStatus` (count per PR_STATUS)
  - `totalValue` (Σ total)
  - `pendingApproval` (count of submitted)
  - `approvedValue` (Σ total where status ∈ {approved, po, invoiced, received})
  - `chain: APPROVAL_CHAIN`
  - `scope: scopeInfo(filter)`
  Sums are raw: currencies are mixed with no conversion.
- **`rollupForCampaign(campaignId, tacticIds=[])`** (used by the SP&A plan detail):
  - Rows are PRs with `campaignId === campaignId`, or whose `tacticId` is in `tacticIds`.
  - `committed` = Σ total where status ∈ {approved, po, invoiced, received}.
  - `invoiced` = Σ total where status ∈ {invoiced, received}.
  - `pending` = Σ total where status is submitted.
  - `draft` = Σ total where status is draft.
  - Rejected PRs are counted in `count` but in no sum.
  - Returns `{ count, committed, invoiced, pending, draft, requests: [{id,title,vendor,status,total,tacticId}] }`.

#### 9.3.7 Mutations
- **`createRequest(input)`**: see the field rules in 3.2. Pushes and saves.
- **`updateRequest(id, patch)`**:
  - Returns `null` if not found.
  - Throws `Only draft requests can be edited.` unless the status is `draft`.
  - Patches any provided field with the same sanitisation. `lines` recomputes `total`. `customAttrs` must be an object. Sets `updatedAt`.
  - **No UI calls this.** `api.prUpdate` exists but is unused.
- **`submitRequest(id, by='You', approverName, approverId)`**:
  - `null` if not found.
  - Not draft: throws `Request already submitted.`
  - No lines: throws `Add at least one line item first.`
  - Builds `approvals = buildApprovals(total, pr.approver || approverName, pr.approverId || approverId)`.
  - Sets status `submitted`, pushes history `{by, action:'submitted'}`, runs `recompute`, saves.
  - Note: `updatedAt` is **not** bumped on submit.
- **`actOnApproval(id, { stepId, approver, decision, by, actorId })`**:
  - Not found: `null`.
  - Status not `submitted`: throws `Request is not awaiting approval.`
  - Target step: the one matching `stepId`, else the first pending step at the active level (and matching `approver` if given).
  - No step found: `No matching pending approval step.`
  - Step not at the active level: `An earlier approval level is still pending.`
  - **Approver lock:** `signedIn = Boolean(by || actorId)`. `isAssigned` = the step has an `approverId`, matched against `actorId`, or the step has no id, matched by `step.approver === by` (name). If signed in and not assigned, it throws `` `Only ${step.approver || 'the assigned approver'} can approve or reject this request.` ``. Anonymous calls (no by, no actorId) are permissive.
  - Then:
    - `step.status = decision === 'reject' ? 'rejected' : 'approved'`
    - `step.at = now`
    - `step.actedBy = by || step.approver`
    - history `{by: by||step.approver, action: '<status> (L<level> <name>)'}`
    - `recompute`, `updatedAt`, save.
- **`advanceFulfillment(id, { by, poNumber, invoiceNumber })`**:
  - `next = NEXT[status]`. With none, it throws `Nothing to advance from the current status.`
  - Moving to `po` sets `poNumber = poNumber || 'PO-' + id.split('-')[1].toUpperCase()`. The fallback segment is `'0000'`. Example: `PO-MTHA524E`.
  - Moving to `invoiced` sets `invoiceNumber = invoiceNumber || 'INV-' + <same segment>`.
  - History `{by: by||'You', action: 'moved to <next>'}`. Sets `updatedAt`.
  - There is no gate on who advances, beyond Editor+.
- **`deleteRequest(id)`** hard-removes the request and returns `{id}`. The route wraps it in the recycle bin.
- **`insertRequest(pr)`** is the recycle-bin restore. It is idempotent: `{added:0}` if the id already exists or is missing, else it pushes and returns `{added:1}`.

#### 9.3.8 Routes (all reads need sign-in; all writes need Editor+)
| Method / path | Behaviour | Side effects |
|---|---|---|
| `GET /api/procurement/requests?planId=&campaignId=` | `listRequests` | — |
| `GET /api/procurement/summary?planId=&campaignId=` | `summary` | — |
| `GET /api/procurement/pending?approver=` | `pendingFor` | — |
| `GET /api/procurement/requests/:id` | 404 `Request not found` | — |
| `POST /api/procurement/requests` | `createRequest(body)` | audit `pr.create` (meta `{total}`). **Notify** `body.requesterId` (unless it is the actor): title "A purchase request names you as requester", text `${actor.name||'Someone'} created “${title}” with you as the requester.`, route `/procurement` |
| `PUT /api/procurement/requests/:id` | `updateRequest`; 404 or 400 | audit `pr.update` |
| `POST /api/procurement/requests/:id/submit` | `actor = req.user?.name \|\| body.by`; calls `submitRequest(id, body.by \|\| actor, actor, req.user?.id)`. If the PR has no chosen approver, **the submitter becomes the approver of every applicable level** (self-approval). Client-sent `approverName` is ignored | audit `pr.submit` (meta `{approver}`). **Notify** `pr.approverId` (unless self): type `approval`, title "Approval needed", text `${actor} submitted “${title}” for your approval.`, route `/procurement` |
| `POST /api/procurement/requests/:id/approve` `{stepId, decision:'approve'\|'reject'}` | `by` is forced to `req.user.name` and `actorId = req.user.id` (client `by` is a fallback only) | audit `pr.approve` / `pr.reject` |
| `POST /api/procurement/requests/:id/advance` `{poNumber?, invoiceNumber?}` | `by` = user name | audit `pr.advance` (meta `{stage: pr.status}`) |
| `DELETE /api/procurement/requests/:id` | **Soft delete**: `trashRequest(id, user.name)` removes the PR and records a recycle-bin item `{ id: uid('trash'), deletedAt, type:'request', label: pr.title, sublabel:'Purchase request', count:1, by, payload:{request: pr} }`. Restore from `/trash` calls `insertRequest` | audit `pr.delete` (severity warning) |

- Errors from these routes use `prErr`, which returns 400 `{ok:false, error: err.message}`.
- Audit categories and severities:
  - `pr.create` / `pr.update` / `pr.advance` / `pr.delete`: category `procurement`.
  - `pr.submit` / `pr.approve`: category `approvals`, severity `notice`.
  - `pr.reject`: category `approvals`, severity `warning`.

**Computed notification feed** (`notifications.js`):
- For every `submitted` PR with a pending step at the active level where `a.approver === user.name` (matched by **name**, not id), emit `{ id: 'approval-<prId>', type:'approval', title:'Approval needed', text: '<title> awaits your sign-off (<level name>)', at: pr.updatedAt, route:'/procurement' }`.

### 9.4 Integration with Planning (SP&A plan) — `campaignPlans.js` + `CampaignPlanDetail.jsx`

- `budgetForCampaign(campaignId)`:
  - Finds the row for the campaign in an **unscoped `budgetRollup('USD')`**.
  - Returns `{ allocated: row.planned, committed: row.committed, variance, utilizationPct }`.
  - Returns `null` if the campaign is missing or on error.
- `getCampaignPlan('spa:<campaignId>')` includes:
  - `procurement: rollupForCampaign(campaignId, <ids of the campaign's tactics>)`
  - `budget: budgetForCampaign(campaignId)`
  - BCPIR plans carry neither.
- **SP&A detail UI:**
  - **BudgetPanel** "Budget reconciliation", with an InfoTip. Link "Open Budget →" goes to `/budget?campaignId=<id>`. Four mini cards:
    - Allocated (top-down): SERIES[4]
    - Planned (this plan): `kpis.plannedCost`
    - Committed (roll-up): SERIES[3]
    - Vs budget: `±fmtPct((planned−allocated)/allocated·100)`. Red when planned > allocated, else green.
  - **ProcurementPanel** "Procurement & approvals · {count}", with an InfoTip. Link "Open Procurement →" goes to `/procurement?campaignId=<id>`.
    - Mini cards:
      - Committed (approved): SERIES[0]
      - Invoiced: SERIES[1]
      - Pending approval: #b0790a
      - Draft: #8a8a8f
    - Table columns: Purchase request / Vendor / Approval status (pill) / Amount.
    - Pill labels: Draft, Pending approval, Approved, Rejected, PO raised, Invoiced, Received.
    - Empty-state text: "No purchase requests yet. Use “Raise PR” on a cost line below to commit spend through the approval chain."
  - Money in these panels uses `fmtMoney` (USD converted to the active currency).
  - **"Raise PR"** row action (receipt icon, tooltip "Raise purchase request", only when canEdit):
    - Calls `prCreate({ title: '<tactic> — <campaignName>', vendor: row.channel||'', tacticId: row.id, campaignId, lines: [{ description: '<tactic> (planned)', qty:1, unitPrice: row.planned, amount: row.planned }] })`.
    - Toasts "Draft purchase request created — opening Procurement to submit it." and navigates to `/procurement` after 700 ms. Note this goes to the unscoped page.
- **Planning line items** carry `budgetPoolId` (and `prId`). LineItemDialog has a "Link to budget" select:
  - Empty option: "Not linked — lands in the budget’s unallocated balance".
  - Other options: `<pool name> · <currency>`.
  - The tactic drawer and grid show "· not linked to a budget" or "unlinked".
  - The budget rollup ignores this link.

### 9.5 Client multi-currency system

#### 9.5.1 `client/src/lib/format.js`
- `CURRENCY_SYMBOLS = { USD:'$', EUR:'€', GBP:'£', CAD:'C$', AUD:'A$', INR:'₹', JPY:'¥', SGD:'S$' }`.
- Module-level `ACTIVE = { code:'USD', rate:1, symbol:'$' }`, where `rate` is USD per 1 unit.
- `setActiveCurrency({code='USD', rate=1, symbol})` → `ACTIVE = { code, rate: rate||1, symbol: symbol || CURRENCY_SYMBOLS[code] || '$' }`.
- `activeCurrency()` returns ACTIVE.
- `toActive(usd) = (Number(usd)||0) / (ACTIVE.rate||1)`. **All stored money is assumed to be USD.**
- **`fmtMoney(usd)`** (compact):
  - Convert, then `neg = n<0 ? '-' : ''`, `a = |n|`.
  - `a ≥ 1,000,000` → `${neg}${sym}${(a/1e6).toFixed(2)}M`
  - `a ≥ 1,000` → `${neg}${sym}${(a/1e3).toFixed(1)}K`
  - otherwise → `${neg}${sym}${Math.round(a)}`
  - The sign goes before the symbol, e.g. `-$1.5K`.
  - There is no B suffix (2.5 billion renders as `$2500.00M`).
- **`fmtMoneyFull(usd)`** (full):
  - `new Intl.NumberFormat(undefined /* browser locale */, { style:'currency', currency: ACTIVE.code, maximumFractionDigits: 0 }).format(n)`, e.g. `$78,000` or `€72,222`.
  - On an exception (e.g. an older engine's min/max fraction conflict), it falls back to `${symbol}${Math.round(n).toLocaleString()}`.
- `fmtNum(n)`: the same compact K/M rules with no symbol and no conversion.
- `fmtNumFull(n)`: `Intl.NumberFormat('en-US')` of the rounded value.
- `fmtPct(n)`: `${Number(n).toFixed(1)}%`.
- Also exported, but not finance-specific:
  - `parseHuman(input)`: strips `$ , whitespace %` and accepts a `k/m/b/t` suffix, e.g. "2.5M" → 2,500,000. Returns null if unparseable.
  - `suggestThresholds(value)`: nice round numbers bracketing the value. Uses multipliers `[0.5,0.75,1,1.25,1.5,2,2.5,3,4,5,7.5,10]` × 10^floor(log10), taking 2 below and 3 above, max 5.
  - `fmtDuration(seconds)`: `"45s"`, `"2m 30s"`, `"3m"`.

#### 9.5.2 `CurrencyContext.jsx` (CurrencyProvider wraps the app)
- `FALLBACK_RATES` is identical to the server's.
- State:
  - `rates` (fallback)
  - `currencies` (fallback keys)
  - `currency` = `localStorage['zr_currency'] || 'USD'`
  - `updatedAt` (null)
  - `busy` (false)
- **During every render** it calls `setActiveCurrency({ code: currency, rate: rates[currency]||1, symbol })`. This is synchronous, so the first paint is already in the right currency.
- On mount: `GET /api/currency` (no refresh). If `rates` is non-empty, set rates; if `currencies` is non-empty, set currencies; set `updatedAt` if present. Errors are ignored.
- **`setCurrency(code)`**:
  1. No-op if the code is empty or unchanged.
  2. `busy = true`.
  3. Fetch `GET /api/currency?refresh=1` (forces a live FX pull). If its rates are non-empty, use them and update `rates` and `updatedAt`; otherwise keep the last-known rates.
  4. Persist `localStorage.zr_currency = code`.
  5. `setActiveCurrency` with the fresh rate, then `setCurrencyState(code)`, then `busy = false`.
- Context value: `{ currency, setCurrency, currencies, rates, updatedAt, busy, symbol: CURRENCY_SYMBOLS[currency] || '$' }`.
- `useCurrency()` throws if used outside the provider.
- **Global re-render:**
  - `App.jsx` calls `useCurrency()`, so a change re-renders the whole routed subtree **without remounting** it. Tabs, expanded rows and scroll position are preserved.
  - The routed page is keyed only by pathname.
  - Every `fmtMoney` / `fmtMoneyFull` value re-formats in place.
  - `AnimatedNumber` includes `currency` in its effect dependencies and repaints immediately.
  - Server-converted views (the Budget rollup) refetch because `currency` is an effect dependency.

#### 9.5.3 `CurrencyControl.jsx` (the "top bar" switch)
- Rendered by `PageHeader` in the right-side actions area on every page that uses it (`currency` prop defaults to true). It is also on the Home dashboard.
- Layout: label "Currency" (12px, ink2; hidden when `compact`), then a 13px spinner while busy, then an MUI `Select`:
  - size small, height 32, font 13, radius 2
  - disabled while busy
  - `renderValue` = `"<symbol> <CODE>"` (tabular numerals), e.g. `€ EUR`
  - menu items: symbol in a 24px-wide muted column, then the code
  - options: `currencies` from the server (8 codes)

### 9.6 Budget page (`/budget`, `client/src/pages/Budget.jsx`)

#### 9.6.1 Frame
- **PageHeader:**
  - eyebrow "Budget" (blue, uppercase 12px)
  - title "Spend with" with highlight "conviction." (highlight in ink2, semibold)
  - subtitle "Top-down budgets by campaign and quarter vs. committed spend — multi-currency, with snapshots."
  - CurrencyControl at top right
- Content container: `max-w-content px-5 pb-24`.
- **Tab switch:** MUI `ToggleButtonGroup`, exclusive, small, pill buttons (radius 999, px 2, no text-transform), margin-bottom 4.
  - "Plan budgets" + InfoTip `planTab` (value `plan`, the **default**)
  - "By channel & region" + InfoTip `channelTab` (value `channel`)
  - Clicking the active tab does nothing (`v && setTab(v)`).
  - Tab state is local and not in the URL.

**InfoTip texts (TIPS), verbatim:**
- planTab: "Top-down budgeting: money handed down from leadership as a target, then spent against real plans. This is the finance view of the marketing plan."
- channelTab: "A delivery view of spend grouped by marketing channel and by region — what was planned vs. what has actually been spent to date."
- topDown: "The total money leadership sets aside for marketing this year, handed down as a target before teams decide exactly how to use it. It is the ceiling everything else is measured against."
- committed: "The part of the budget already assigned to specific campaigns and tactics (whether or not it has been paid yet). It shows how much of the plan is already spoken for."
- uncommitted: "Budget still free to allocate = Top-down budget − Committed. Positive means you have room left to invest; negative means you have over-committed."
- utilization: "Committed ÷ Top-down budget, shown as a percentage. It tells you how much of the plan is used up — near 100% means the budget is fully allocated."
- campaignRow: "Each bar shows how much of a campaign’s allocated budget is already committed. Click a row to see the split by quarter."
- planned: "The budget allocated to this campaign (or quarter) from the top-down pool — the amount you intend to spend."
- committedRow: "What this campaign’s tactics actually intend to spend — the money already committed against the plan."
- quarter: "The three-month period the money is phased into (Q1–Q4), so you can see spending planned across the year."
- totalPlanned: "The sum of the budgets planned across every channel (or region) for the period."
- actualSpend: "Money actually spent so far this period — the real invoices and costs to date."
- remaining: "Total planned budget − actual spend to date. What is still left to spend."
- overallUtil: "Actual spend ÷ total planned budget, as a percentage — how much of the plan you have used so far."
- actual: "Money actually spent, versus the amount that was planned."

InfoTip component: an `InfoOutlined` icon (default 14px, muted, blue on hover, cursor help, focusable, aria-label = text) with an MUI dark tooltip (12px, max width 260, arrow). Touch shows it immediately and keeps it for 4 s.

#### 9.6.2 Plan budgets tab (`PlanBudgets`)
- **Inputs:** `currency` and `symbol` from `useCurrency()`; `planId` and `campaignId` from the URL search params.
- **Fetch:**
  - `GET /api/budgets/rollup?currency=<active>&planId=&campaignId=` on mount and whenever currency, planId or campaignId changes. Uses an `alive` guard.
  - `loading` is set true on each refetch. Note that `error` is never cleared on a later success.
  - The whole tab, **including the ScopePicker and SnapshotPanel**, sits inside `StateWrap`. Each scope or currency change shows a centered 28px spinner (minHeight 240), then remounts the content.
  - Error text: "Could not load data: <message>. Is the API running on port 4000?" (outlined error Alert).
- **Local `money(n)`:** the same compact K/M rules as `fmtMoney`, but **no conversion**, because figures arrive already converted by the server. Uses the context `symbol`. Format: `${neg}${symbol}${body}`.
- **Scope row** (flex, wrap, space-between, mb-6):
  - `<ScopePicker/>` (section 8)
  - Right side: "All figures in **USD** — change from the top bar." (13px; the code is bold)
- **Scope sentence** (12.5px, ink2):
  - If `roll.scope`: "Showing budget for **<name>** only — campaign scope." (or "plan scope.")
  - Else: "Showing the **portfolio roll-up** across every campaign. Pick a plan or campaign above to focus."
- **Captured region** (`ref=budgetRef`; this is what PNG snapshots capture) contains:
  - **KPI cards:** a `ScrollReveal` grid, 2 columns (4 at `lg`), gap 5. Each card is `card card-hover p-6`, with a label (13px secondary) + InfoTip and a value (30px, weight 700, letter-spacing −0.03em, coloured with the accent).
    1. **Top-down budget**: `money(totals.planned)`, accent SERIES[4] `#4a3aa7` (violet)
    2. **Committed**: `money(totals.committed)`, accent SERIES[0] `#2a78d6` (blue)
    3. **Uncommitted**: `money(totals.variance)`, accent STATUS.good `#0ca30c` if ≥ 0, else STATUS.critical `#d03b3b`
    4. **Utilization**: `fmtPct(totals.utilizationPct)`, accent SERIES[2] `#e87ba4` (magenta)
  - **"Budget by campaign" Panel** (title + InfoTip `campaignRow`; subtitle "Top-down plan vs. committed — click a row to drill into quarters"). Rows are separated by hairline dividers. Each row is a full-width button:
    - Chevron: right when collapsed, down when expanded (18px, muted).
    - Campaign name: `w-44`, medium weight.
    - MUI `LinearProgress` (determinate): value `min(100, utilizationPct)`, height 10, pill, track `var(--chip)`, bar gradient `linear-gradient(90deg,#2a78d6,#9f4bd6)`.
    - `"<money(committed)> / <money(planned)>"` (`w-28`, right-aligned, 13px, ink2).
    - `fmtPct(utilizationPct)` (`w-14`, semibold, **red `#d03b3b` when > 100**, else ink).
    - Click toggles the drill-down. Only **one** row is expanded at a time (`expanded = campaignId | null`).
  - **Drill-down** (indented `pl-9`, horizontal scroll, `min-w-[520px]`, 12.5px):
    - Header row: "Quarter" + InfoTip, then one right-aligned column per entry in `roll.quarters` (global list).
    - Row "Planned" + InfoTip: `money(byQuarterPlanned[q])`, or `—` when 0 or absent.
    - Row "Committed" + InfoTip: `money(byQuarterCommitted[q])`, or `—`.
    - Tabular numerals.
- **SnapshotPanel** sits outside the captured region: `dashboardId="budget"`, `targetRef=budgetRef`, `noun="budget view"` (section 9).
  - Snapshots are browser-local PNGs, shared across all scopes and currencies under the single key "budget".
  - The server snapshot API is not used here.
- **Permissions:** the page is read-only. There are no edit controls and no gating.

#### 9.6.3 By channel & region tab (`ChannelBudgets`)
- **Data:** `GET /api/budget` → `mockData.budget()` returns `{ byChannel: [{category, planned, actual}], byRegion: [...] }`, in USD. It is **not scoped** and has no ScopePicker. Conversion is client-side via the formatters.
- **Mock derivation:**
  - Channels: Paid Search, Paid Social, Organic / SEO, Email, Events & Webinars, Content Syndication.
  - Regions: North America, EMEA, APAC, LATAM.
  - Each value is the sum of mock campaigns' `budget` and `spent` for that category.
  - Current values:
    - Paid Search 660,000 / 488,000
    - Paid Social 90,000 / 78,000
    - Organic 70,000 / 41,000
    - Events & Webinars 180,000 / 142,000
    - Content Syndication 550,000 / 372,000
    - North America 820,000 / 631,000
    - EMEA 490,000 / 314,000
    - APAC 240,000 / 176,000
  - Empty categories (Email, LATAM) get a seeded-PRNG random value:
    - Email planned is `money(20000,80000)` (rounded to 100), actual `round(planned×0.7)`.
    - LATAM planned is `money(30000,90000)`, actual `round(planned×0.65)`.
    - The PRNG is module-level (mulberry32 seed 42), so these two values **change on each request**.
- **Totals (client):** planned = Σ byChannel.planned; actual = Σ byChannel.actual; remaining = planned − actual; util = actual/planned·100.
- **KPI cards:** a 1/3-column grid, each `card card-hover p-7`, value 34px bold.
  1. "Total planned budget": `fmtMoneyFull`, SERIES[4], tip totalPlanned
  2. "Actual spend to date": `fmtMoneyFull`, SERIES[0], tip actualSpend
  3. "Remaining": `fmtMoneyFull`, **always** STATUS.good, tip remaining
- **Panel "Overall budget utilization"** + InfoTip overallUtil; subtitle "Actual spend against total plan". Contains a LinearProgress (height 14, same gradient, `min(100, util)`) and `fmtPct(util)` (22px, weight 800, min width 76).
- **Two chart panels** (`lg`: 2 columns): "Budget by channel" and "Budget by region", each with subtitle "Planned vs. actual".
  - Header legend: violet dot "Planned" + InfoTip, blue dot "Actual" + InfoTip.
  - **ECharts horizontal grouped bar**, height 300:
    - `grid {left 6, right 16, top 8, bottom 4, containLabel}`
    - tooltip: `trigger 'axis'`, axisPointer `shadow`, formatter `seriesTooltip(fmtMoneyFull)`. It shows the category header, then marker, series name and bold value per series.
    - xAxis: `value`, labels `fmtMoney` (compact)
    - yAxis: `category`, **inverse** (first item at top), the category names, font 12
    - series: `Planned` (bar, SERIES[4]) and `Actual` (bar, SERIES[0]), `borderRadius [0,6,6,0]`, `barMaxWidth 16`
  - The app's ECharts wrapper applies `zr-light` / `zr-dark` themes.

### 9.7 Procurement page (`/procurement`, `client/src/pages/Procurement.jsx`)

#### 9.7.1 State and data
- `canEdit = !user || role !== 'Viewer'`.
- `tab`: `'all'` (default) or `'mine'`.
- `toast`, `openId` (drawer), `createOpen`, `draft`.
- Scope comes from the URL: `planId` and `campaignId`.
- **Initial load** (re-run when `planId` or `campaignId` changes), in parallel:
  - `GET /procurement/requests?scope`
  - `GET /procurement/summary?scope`
  - `GET /planning/board`, which is **unscoped**. Its tactics (`columns.flatMap(c => c.initiatives)`) feed the "Link to tactic" select.
- Everything is wrapped in StateWrap (spinner or error).
- **`refresh()`** re-fetches the list and summary only.

#### 9.7.2 Layout
- **PageHeader:**
  - eyebrow "Procurement"
  - title "Purchase requests," with highlight "approved with control."
  - subtitle "Vendor purchase requests, multi-level approvals, then PO → invoice → receipt."
  - CurrencyControl
- **Scope row:** `<ScopePicker/>` on the left. On the right (12.5px):
  - scoped: "Requests for **<scope name>** only"
  - unscoped: "All purchase requests across the portfolio"
- **KPI strip:** 2 columns (4 at `sm`). Each `Kpi` is `card p-4`, with a label (12.5px ink2) + optional InfoTip (13px) and a value (24px, extra-bold, tabular, letter-spacing −0.02em, accent colour).
  1. "Requests": `summary.total`, ink, tip G.purchaseRequest ("A formal request to spend money with a vendor, routed for approval before any commitment is made.")
  2. "Awaiting approval": `summary.pendingApproval`, `#c77700`, tip G.approvalChain ("The ordered list of people who must sign off before a purchase can proceed — bigger amounts may need more approvers.")
  3. "Approved value": `fmtMoney(summary.approvedValue)`, STATUS.good `#0ca30c`, tip G.approvedValue ("The total money across requests that have cleared approval — spend that’s been officially green-lit.")
  4. "Total value": `fmtMoney(summary.totalValue)`, ink, no tip
  - KPIs reflect the scope but **not** the tab.
  - `summary.byStatus` and `summary.chain` are returned but never displayed.
- **Toolbar row:**
  - ToggleButtonGroup (pill): "All requests" (`all`) and "Approvals inbox" (`mine`).
  - Right side, only if canEdit: contained pill button "New request" with a + icon, opening the create dialog.
- **Approvals inbox filter** (client-side): status `submitted`, AND there are pending steps, AND some pending step at the active level (min pending level) is assigned to me. "Assigned to me" means `a.approverId ? a.approverId === user.id : a.approver === user.name`.
- **Requests table:** rounded-2xl hairline border, horizontal scroll, `min-w-[720px]`, 13px.
  - Header row: gray background, 11px uppercase.
  - Columns:
    1. **Request**: title, medium weight
    2. **Vendor** + InfoTip G.vendor ("The outside company or supplier you’re paying (e.g. an ad platform, agency or software provider).") — vendor or `—`
    3. **Requester**
    4. **Amount** (right-aligned): `fmtMoneyFull(total)`, semibold, tabular
    5. **Status**: StatusChip
  - The whole row is clickable and opens the drawer. Hover tint: black/2% (light), white/3% (dark).
  - No sorting, search, pagination or column filters. Order is the server order.
  - Empty row (colSpan 5, centered): "Nothing awaiting your approval." (inbox) or "No purchase requests yet." (all).
- **StatusChip** (rounded-full, px-2, py-0.5, 11px semibold; background = colour + `1c` alpha, text = colour):

| status | label | colour |
|---|---|---|
| draft | Draft | `#86868b` |
| submitted | Awaiting approval | `#c77700` |
| approved | Approved | `#0071e3` |
| rejected | Rejected | `#d03b3b` |
| po | PO issued | `#1baf7a` |
| invoiced | Invoiced | `#1baf7a` |
| received | Received | `#0ca30c` |

Unknown statuses fall back to the raw label in `#888`.

#### 9.7.3 Create dialog ("New purchase request")
- MUI Dialog, `maxWidth sm`, fullWidth. Title is bold. Content has dividers and a 2-column grid with gap 3.
- **Initial draft:**
  `{ title:'', vendor:'', requester: user?.name || 'You', requesterId: user?.id || '', approver:'', approverId:'', currency:'USD', tacticId:'', lines:[{description:'', qty:1, unitPrice:0}] }`
- **Fields:**
  1. **Title**: TextField, full width (spans both columns)
  2. **Vendor**: TextField
  3. **Currency**: Select with the hardcoded `USD, EUR, GBP, CAD, AUD, INR, JPY, SGD`
  4. **Requester**: PeoplePicker, pre-filled with me. `onChange(p)` sets `requester = p?.name||''` and `requesterId = p?.id||''`.
  5. **Approver**: PeoplePicker, placeholder "Who signs this off?". Sets `approver` and `approverId`.
  6. **Link to tactic**: Select, full width. Options: "*None*" (value `''`), then every tactic title from the board.
  7. **Line items** section (label "LINE ITEMS", 12px uppercase). Each line is a 12-column grid row:
     - Description (span 6, placeholder "Description")
     - Qty (number, span 2)
     - "Unit $" (number, span 3)
     - Delete icon button (span 1; removes the line, and can remove all lines)
     - "+ Add line" text button appends an empty line.
     - Right-aligned "Total: <fmtMoneyFull(draftTotal)>", where `draftTotal = Σ (Number(qty)||0) × (Number(unitPrice)||0)`.
- **PeoplePicker:** an Autocomplete over the cached `/api/org/people` directory, showing avatar chips. The value may be an id or a name; unknown names stay visible as free text.
- **Actions:** "Cancel", and the contained "Create draft" button, which is **disabled while the title is blank (trimmed)**.
- **Save logic:**
  - Drop lines whose description is blank.
  - For each remaining line, coerce `qty = Number(qty)||1`, `unitPrice = Number(unitPrice)||0`, `amount = qty×unitPrice`.
  - Call `prCreate({...draft, tacticId: draft.tacticId || null, lines})`.
  - Close the dialog, refresh, and toast success "Purchase request created (draft)."; on error, toast the error message.
  - Not set by the dialog: no `campaignId` (scope comes via the tactic), no prefill from the current scope, no customAttrs.
  - Quirk: the displayed total counts a blank qty as 0, while the saved line counts it as 1, and the displayed total includes lines that are dropped for a blank description.

#### 9.7.4 Detail drawer (`PrDrawer`)
- Right-anchored MUI Drawer: width 500px (`sm+`), 100% on `xs`. Open when `openId` is set.
- On `id` change: `GET /procurement/requests/:id`. Errors are swallowed, so the drawer stays on a centered "Loading…".
- **Header** (border-bottom):
  - Title (17px bold).
  - StatusChip + "<vendor> · <requester>" (12.5px).
  - If the PR has a `campaignId`: a pill link "View linked SP&A plan →" to `/planning?plan=spa:<campaignId>`, which also closes the drawer.
  - Close (×) icon.
- **Body** (scrolls):
  - **"LINE ITEMS · <CURRENCY>"** + InfoTip G.lineItems ("The individual things being bought within a request, each with a quantity and unit price that add up to the total.").
    - One bordered row per line: description (truncated), `"<qty> × <fmtMoney(unitPrice)>"`, `fmtMoneyFull(amount)` (semibold).
    - Then a bold "Total" row with `fmtMoneyFull(total)`.
    - Amounts are treated as USD and converted to the active currency, whatever the PR's currency.
  - **"APPROVAL CHAIN"** (only when `approvals.length > 0`) + InfoTip. One bordered row per step:
    - A blue circular badge `L<level>` (`rgba(0,113,227,.1)` background, `#0071e3` text).
    - Line 1: `"<level name> · <assigned approver>"`, plus `" · you"` when assigned to me and still pending.
    - Line 2 (11px): `parallel` or `serial`, plus `" · <status>"` when not pending.
    - Approved: green check icon. Rejected: red block icon.
    - **Approve / Reject buttons** appear only when `canEdit && pr.status==='submitted' && step pending && step.level === activeLevel && assigned to me` (by id, else by name):
      - "Approve" (contained, success) calls `prApprove(id, {stepId, decision:'approve', by: user.name})` and toasts "Approved."
      - "Reject" (outlined, error) does the same with `decision:'reject'` and toasts "Rejected."
  - **"HISTORY"**: one line per entry, the action on the left and `new Date(at).toLocaleString()` on the right. The `by` field is not shown.
  - **PO / invoice chips** (if set): `PO <poNumber>`, `Invoice <invoiceNumber>`. The label reads e.g. "PO PO-MTHA524E".
- **Footer** (only if canEdit; divider above):
  - Left: red text button "Delete" (trash icon). There is **no confirmation**. It calls `prDelete`, ignores errors, closes the drawer and refreshes. No toast. Allowed in any status. The PR goes to the recycle bin.
  - Right:
    - draft: "Submit for approval" calls `prSubmit(id, pr.requester, user.name)` and toasts "Submitted for approval."
    - approved: **"Issue PO"**; po: **"Mark invoiced"**; invoiced: **"Mark received"**. Each calls `prAdvance(id, {})` (server auto-numbers) and toasts "Advanced."
    - submitted, rejected and received show no lifecycle button.
- **Action wrapper:** `act(fn, okMsg)` runs fn (the action, then a drawer reload), then `refresh()` of the list and summary, then a success toast. On an error it toasts the server message, for example:
  - `Request already submitted.`
  - `Add at least one line item first.`
  - `Request is not awaiting approval.`
  - `No matching pending approval step.`
  - `An earlier approval level is still pending.`
  - `Only <name> can approve or reject this request.`
  - `Nothing to advance from the current status.`
  - 403 `This action requires the Editor role or higher.`
- **Snackbar:** 3000 ms auto-hide, bottom-centre, filled Alert (success or error), rounded, closable.

#### 9.7.5 Worked approval examples
- **$78,000 PR with no approver chosen, submitted by user U:** the server sets the approver to U's name and id. Steps are L1 (U) and L2 (U), both serial. U approves L1, then L2 becomes active; U approves L2, and the PR becomes approved.
- **Seeded r1 (default chain, no ids):** only a signed-in user named "Dana Alvarez" can act on L1, then only "Marcus Lee" on L2.
- **$150,000 PR with chosen approver X:** three single steps, L1, L2 and L3, all assigned to X. The L3 step is labelled "parallel" but has one approver. X must click Approve three times in sequence.
- **Default chain at ≥ $100k** (only reachable via the anonymous path or seeds): L3 has two steps, Abhishek Biyani and Sam Okafor. Both must approve; either one rejecting rejects the PR.

### 9.8 ScopePicker (`client/src/components/ScopePicker.jsx`)

Shared by the Budget (plan tab) and Procurement pages. **The URL is the single source of scope**: `?planId=` or `?campaignId=`.

- **Data:** `useApi(() => api.campaignPlans())`, i.e. `GET /api/planning/plans` (fetched once per mount).
  - It uses `data.groups`.
  - Each group has `{ planId, planName, fiscalYear: 'FY2026'-style label (normalised: strip/add 'FY'), campaigns: [{ campaignId, campaignName, ... }] }`.
  - Every plan node is a group, even one with no campaigns.
  - Campaigns whose parent plan is missing go into a trailing group `{ planId: null, planName: 'Unassigned' }`.
- **Value encoding:**
  - `''` = Portfolio
  - `plan:<id>`
  - `campaign:<id>`
  - When reading the URL, campaignId takes precedence over planId.
- **Label map:**
  - `'' → 'Portfolio — all campaigns'`
  - plan → `"<planName> · <fiscalYear>"` (or just the name if there is no FY)
  - campaign → the campaign name
- **Out-of-range guard:** until the options load, or if the URL id is unknown, the Select shows Portfolio. The URL itself is not changed.
- **Options, in order:**
  1. "Portfolio — all campaigns"
  2. For each group:
     - If it has a `planId`: a bold, selectable plan item. Otherwise a non-selectable `ListSubheader` (12px) with the group name ("Unassigned").
     - Then that group's campaigns, indented (`pl: 4`, 13px).
- **onChange:**
  - Copy the current search params and delete `planId` and `campaignId`.
  - Set `planId` for a plan option or `campaignId` for a campaign option. Portfolio sets neither.
  - `setSearchParams(p)` pushes a history entry and preserves any other params.
  - Selecting a campaign sets only `campaignId`, not the parent planId.
- **Look:**
  - Inline label "VIEWING" (11px uppercase semibold; the `label` prop defaults to 'Viewing').
  - Pill Select: radius 999, min width 240, 13px weight 600, background `var(--surface)`, `var(--card-border)` outline with `var(--blue)` on hover, `UnfoldMoreRounded` icon (18px muted), menu max height 420.
  - `displayEmpty`, and `renderValue` uses the label map.
- **Deep links into these pages:**
  - `/budget?campaignId=<id>` (SP&A "Open Budget →")
  - `/procurement?campaignId=<id>` (SP&A "Open Procurement →")
  - `?planId=<id>` is supported equally on both pages.
- Both pages refetch when the scope changes. On Budget the currency is also sent; on Procurement only the scope.

### 9.9 Snapshots: PNG capture and browser store

#### 9.9.1 Capture (`client/src/lib/snapshot.js`)
- `captureNode(node, mode='light')` uses `html-to-image`'s `toPng(node, { pixelRatio: 1.5, backgroundColor: mode==='dark' ? '#0b0b0d' : '#ffffff', cacheBust: true, filter: el => !(el.dataset && el.dataset.noSnapshot !== undefined) })`.
- Elements with a `data-no-snapshot` attribute are excluded.
- ECharts canvases are same-origin, so they capture correctly.
- Throws `Nothing to capture.` if there is no node.

#### 9.9.2 Store (`client/src/lib/snapshotStore.js`), localStorage only, no backend
- Key `'zenreach.snapshots.v1'` holds a JSON map `{ [dashboardId]: Snapshot[] }`.
- `MAX_PER_DASHBOARD = 6`.
- Snapshot shape: `{ id: 'snap-' + Date.now().toString(36) + '-' + 4 random base-36 chars, dashboardId, label, image (PNG data URL), createdAt: epoch ms }`.
- `listSnapshots(id)`: newest first, sorted by `createdAt` descending. Returns `[]` with no id.
- `saveSnapshot(id, image, label)`:
  - Throws `Save the dashboard first.` with no id.
  - Prepends the new snapshot and keeps the first 6, so the oldest per dashboard is dropped.
  - If `setItem` throws (quota): repeatedly evict the globally oldest snapshot (across all dashboards, never the new one), removing empty keys, and retry. At most 200 iterations.
  - If it still does not fit, it throws `This snapshot is too large for local storage — try a narrower dashboard or delete older snapshots.`
  - Default label: `Snapshot · <toLocaleString()>`.
- `deleteSnapshot(id, snapId)` removes the snapshot and drops an empty key.
- `deleteSnapshotsFor(id)` is called when a Builder dashboard is deleted.
- Reads and writes are wrapped in try/catch; a failed read returns `{}`.

#### 9.9.3 `SnapshotPanel` UI (`dashboardId`, `targetRef`, `noun='dashboard'`)
- Card `mt-8 p-5`, marked `data-no-snapshot`.
- **Header:** camera icon + "Snapshots", with " · N" when N > 0.
- **Button "Take snapshot"** (contained pill, camera icon):
  - While busy: a 14px spinner and the text "Capturing…".
  - Disabled when busy or when there is no `dashboardId`.
- **take():**
  - No `dashboardId`: info toast "Save the dashboard first, then take a snapshot."
  - No target: no-op.
  - Otherwise capture with the current theme mode, save with label `Snapshot · <toLocaleString()>`, refresh, and toast success "Snapshot saved to this browser."
  - On failure: error toast "Snapshot failed: <message>".
- **Body states:**
  - No id: "Save this <noun> first — then you can capture PNG snapshots to revisit later." On Budget this never shows, because the id is the constant "budget".
  - Empty: "No snapshots yet. Capture the current <noun> to save a PNG (kept in this browser)."
  - Otherwise a wrapping gallery of 200px-wide tiles:
    - The image (aspect 16/10, object-cover anchored to the top, lazy-loaded, gray background) opens the viewer.
    - The caption shows `new Date(createdAt).toLocaleString()` and a delete icon with tooltip "Delete snapshot". Delete is immediate, with no confirmation, and it closes the viewer if it was showing that snapshot.
- **Viewer dialog** (`maxWidth lg`, fullWidth):
  - Header with the label, "Open in new tab" (`window.open()` then `document.write('<img src=dataUrl style="max-width:100%">')`) and a close button.
  - Image `max-h-[78vh]`, object-contain.
- **Snackbar:** 3000 ms, bottom-centre, filled.

### 9.10 Behaviours and quirks a rebuild should keep, or fix deliberately

1. Money is assumed to be **USD everywhere on the client**. The PR `currency` and pool `currency` fields are not used by the client formatters. Only the server Budget rollup converts (pool currency, line-item currency → requested currency). Procurement thresholds, summaries and `rollupForCampaign` sum raw totals across currencies.
2. The Budget rollup's `quarters` columns are global, not scoped. Allocations only count for **campaign-level** nodes. Line-item periods and `budgetPoolId` are ignored by the rollup.
3. Self-heal only fires when **zero** allocations match the current campaigns. New campaigns get planned 0 and utilization 0% even if they have committed spend.
4. The SPLIT fallback share (`1/n`) can over-allocate the pool when campaign names beyond the 3 seeded names exist at seed time.
5. When no approver is picked, the submitter becomes the approver of every level (self-approval). A chosen approver owns every level as one step per level, and parallel L3 collapses to a single person. The default multi-approver chain only applies to seeds or anonymous calls.
6. The approver lock matches by stable id when the step carries one, otherwise by display name. The client mirrors the same rule for showing buttons and for the inbox. The notification feed matches by name only.
7. Rejected is terminal. There is no edit UI for drafts (PUT exists but is unused), no confirmation on delete, and no "who" in the history display.
8. PO and invoice numbers are derived from the PR id's timestamp segment, so the PO and invoice share the same suffix.
9. Unused APIs: pools/allocations/snapshot CRUD (server snapshots are JSON rollups, separate from the PNG snapshots), `/api/procurement/pending`, `/api/budgets/currencies`, and `PUT /api/procurement/requests/:id`.
10. The FX cache is in memory: it lasts 1 h, is forced on every currency switch, uses a 6 s timeout, and falls back to the static table. The client keeps its own fallback copy and persists the choice in `localStorage.zr_currency`.

## 10. Performance data, analytics, attribution, goals, optimizer and AI

**Source files:** `server/services/mockData.js`, `googleAnalytics.js`, `goals.js`, `insights.js`, `analysis.js`, `optimizer.js`, `attribution.js`, `assistant.js`, `assistantDashboard.js`, `assistantWidgets.js`, `campaignDataset.js`, `client/src/pages/Dashboard.jsx`, `Analytics.jsx`, `Attribution.jsx`, `Campaigns.jsx`, `Goals.jsx`, `Optimizer.jsx`, `Integrations.jsx`, `client/src/components/AiAssistant.jsx`, `Hero.jsx`, `KpiCard.jsx`, `AnimatedNumber.jsx`, `client/public/tracking-test.html`.

Reference numbers in this chapter were produced by running the pure service modules against the demo data. Use them as acceptance-test fixtures.

### 10.0 Stack and plumbing for this area

- **Server:** Node ESM + Express 4 (`server/index.js`), port `PORT` (default 4000). Every response is wrapped as `{ ok: true, data }`; errors are `{ ok:false, error }`. JSON body limit is 60 MB, so whole spreadsheet uploads arrive as base64. Dependencies: `@google-analytics/data` ^4.7, `openai` ^6.48, `@anthropic-ai/sdk` ^0.68, `xlsx` ^0.18.5, `dotenv` (loaded via `import 'dotenv/config'` in index.js only).
- **Auth wall:** every `/api/*` route needs a valid bearer token. The only exceptions are login, signup, `GET /auth/me`, logout, `GET /org/personas`, `GET /currency`, `GET /health` and the public `.ics` feed. So all analytics endpoints require sign-in. `requireRole('Editor')` also guards the dataset import (anonymous callers pass when there are no users or no token).
- **Client:** React 18, MUI 5, Tailwind, ECharts 6 (tree-shaken: Line, Bar, Pie, Gauge, Funnel plus Grid, Tooltip, Legend, MarkLine, MarkArea, Title, LegacyGridContainLabel, Canvas), GSAP 3.
  - `api.js` is a thin fetch wrapper (`/api` prefix via the Vite proxy) that attaches `Authorization: Bearer <localStorage zr_token>`.
  - `useApi(fn)` returns `{data, loading, error}`.
- **Routes:** when signed in: `/` (Dashboard), `/campaigns`, `/goals`, `/optimizer`, `/analytics`, `/integrations`, `/attribution`, plus others. When signed out, only `/` exists and everything else redirects there. `AiAssistant` mounts only when `user` is set. It sits outside the keyed page `div`, so chat history survives navigation.
- **Sidebar nav:**
  - MAIN: Home `/`, Planning, Builder, Dashboards, Budget, Analytics `/analytics`.
  - MORE: Campaigns `/campaigns`, Goals `/goals`, "Budget optimizer" `/optimizer`, Procurement, Attribution `/attribution`, Integrations `/integrations`, Audit log, Recycle bin.
  - Admin is separate.

#### 10.0.1 Endpoints in this area (server/index.js)

| Method / path | Handler | Notes |
|---|---|---|
| GET /api/overview | `mockData.overview()` | |
| GET /api/campaigns | `mockData.campaigns` (array) | |
| GET /api/budget | `mockData.budget()` | `{byChannel, byRegion}` |
| GET /api/channels/revenue | `revenueByChannel()` | |
| GET /api/performance/timeseries?days= | `timeseries(days)` | days clamped to 7–90, default 30 |
| GET /api/integrations | catalog + status | GA status is `connected` if configured, else `demo`; every other entry is `connected` |
| GET /api/analytics/google | `getAnalytics()` | 500 on throw (in practice it never throws; it falls back to demo) |
| GET /api/analytics/realtime | `getRealtime()` | |
| GET /api/goals | `goals()` | |
| GET /api/insights | rule-based `insights()` | not used by any page; feeds assistant context |
| GET /api/insights/ai | `aiAnalysis()` | cached |
| GET /api/optimizer | `recommend()` | |
| POST /api/optimizer/project | `project(body.allocations)` | not used by client (client computes locally) |
| GET /api/assistant/status | `assistantStatus()` | |
| POST /api/assistant | `assistantTurn({message, history, flow})` | message sliced to 2000 chars; 400 if empty |
| GET /api/attribution/models | `ATTRIBUTION_MODELS` | |
| GET /api/attribution?model= | `attribute(model)` | invalid/missing model → linear |
| GET /api/attribution/compare | `compareModels()` | exposed in api.js as `attributionCompare`, unused by UI |
| GET /api/attribution/funnel | `funnel()` | |
| GET /api/attribution/reallocation?model= | `reallocation(model)` | default `position` (but `attribute` maps an unknown model to linear) |
| GET /api/attribution/forecast | `forecast()` | |
| GET /api/dataset/campaign/meta | `getMeta()` | |
| POST /api/dataset/campaign/import | `toCsvText(body)` → `ingestCsv(csv, body.name)` | Editor role; audit `import.dataset` with `meta.rows`; 400 on error |
| GET /api/builder/catalog, /api/builder/data | builder catalog / data bundle | include dataset sources when loaded |
| GET /api/import/sample, POST /api/import/tactics | tactics CSV template/import | used by the Integrations page |

### 10.1 Demo dataset: `server/services/mockData.js`

#### 10.1.1 Seeded PRNG
- `mulberry32(seed)`: standard implementation (`seed|=0; seed=(seed+0x6d2b79f5)|0; t=Math.imul(seed^(seed>>>15),1|seed); t=(t+Math.imul(t^(t>>>7),61|t))^t; return ((t^(t>>>14))>>>0)/4294967296`).
- The module-level `rand = mulberry32(42)`.
- `between(min,max) = Math.round(min + rand()*(max-min))`; `money(min,max) = Math.round(between(min,max)/100)*100`.
- **Important quirk:** `rand` is shared and consumed on every call, so the values come out in a fixed sequence from process start but change with each request. It affects:
  - budget fallbacks for channels/regions with no campaigns (Email, LATAM);
  - `gaDemo().totals.avgEngagementSec` (95–190);
  - `gaDemo().topPages[*].views`.
- Example: Email planned was 56,100 on the first call and 36,400 on the second. Everything else is pure, deterministic arithmetic.

#### 10.1.2 Constants
- `CHANNELS = ['Paid Search','Paid Social','Organic / SEO','Email','Events & Webinars','Content Syndication']`
- `REGIONS = ['North America','EMEA','APAC','LATAM']`

#### 10.1.3 Campaigns (the "Nova GA launch, Q3 2026" story)

Fields: `id` (`cmp-${100+i}`), name, channel, region, status, budget, spent, leads, pipeline, revenue, `roi = +(revenue/spent).toFixed(2)`, `startDate: '2026-07-01'`.

| id | name | channel | region | status | budget | spent | leads | pipeline | revenue | roi | used% |
|---|---|---|---|---|---|---|---|---|---|---|---|
| cmp-100 | Nova GA Launch | Paid Search | North America | Active | 420000 | 312000 | 2450 | 2180000 | 1090000 | 3.49 | 74 |
| cmp-101 | Nova Enterprise ABM | Content Syndication | EMEA | Active | 360000 | 268000 | 640 | 3050000 | 1525000 | 5.69 | 74 |
| cmp-102 | Nova Launch Webinar Series | Events & Webinars | North America | Active | 180000 | 142000 | 1850 | 1180000 | 560000 | 3.94 | 79 |
| cmp-103 | Nova Demand Gen | Paid Search | APAC | Active | 240000 | 176000 | 2100 | 980000 | 470000 | 2.67 | 73 |
| cmp-104 | Nova Retargeting | Paid Social | North America | Active | 90000 | 78000 | 720 | 640000 | 360000 | 4.62 | 87 |
| cmp-105 | Nova Developer Community | Organic / SEO | North America | Active | 70000 | 41000 | 1350 | 520000 | 300000 | 7.32 | 59 |
| cmp-106 | Nova Partner Co-Marketing | Content Syndication | EMEA | Planned | 130000 | 46000 | 380 | 410000 | 150000 | 3.26 | 35 |
| cmp-107 | Nova Analyst Relations | Content Syndication | North America | Completed | 60000 | 58000 | 210 | 720000 | 300000 | 5.17 | 97 |

#### 10.1.4 `overview()`
- Sums: totalBudget 1,550,000; totalSpent 1,121,000; totalPipeline 9,680,000; totalRevenue 4,755,000; totalLeads 9,700.
- `budgetUtilization = +((spent/budget)*100).toFixed(1)` → 72.3.
- `blendedRoi = +(revenue/spent).toFixed(2)` → 4.24.
- `activeCampaigns` = count with status Active → 6.

#### 10.1.5 `budget()`
- `byChannel`: for each of the 6 CHANNELS, `planned` = Σ budget of that channel's campaigns, falling back to `money(20000,80000)` when 0. `actual` = Σ spent, falling back to `round(planned*0.7)`. Shape: `{category, planned, actual}`.
  - Values: Paid Search 660,000/488,000; Paid Social 90,000/78,000; Organic 70,000/41,000; Email random/0.7×; Events 180,000/142,000; Content Syndication 550,000/372,000.
- `byRegion`: same rule over REGIONS, with fallback `money(30000,90000)` and `actual = round(planned*0.65)`.
  - Values: NA 820,000/631,000; EMEA 490,000/314,000; APAC 240,000/176,000; LATAM random.

#### 10.1.6 `revenueByChannel()`
- A Map in insertion order of channel → Σ revenue.
- Result: Paid Search 1,560,000; Content Syndication 1,975,000; Events & Webinars 560,000; Paid Social 360,000; Organic / SEO 300,000. Email does not appear.

#### 10.1.7 `timeseries(days=30)`: deterministic launch ramp
For `d = days-1 … 0`, with `idx = days-1-d` (0 = oldest) and `t = days>1 ? idx/(days-1) : 1`:
- `spend = round(3800 + 2800t + sin(idx/3)*320)`
- `leads = round(46 + 74t + sin(idx/2)*7)`
- `mult = 2.4 + 2.1t + (t>0.66 ? (t−0.66)*4.2 : 0)`; `revenue = round(spend*mult)`
- `sessions = round(2600 + 4200t + sin(idx/2)*480)`
- `date = dayLabel(d)`: `new Date('2026-07-15T00:00:00Z')` minus d UTC days, as `YYYY-MM-DD`. The anchor date is fixed, not "today".

30-day series: first row 2026-06-16 `{spend 3800, leads 46, revenue 9120, sessions 2600}`; last row 2026-07-15 `{spend 6523, leads 127, revenue 38668, sessions 7249}`. Last 7 days revenue = 239,675 vs the prior 7 days = 154,577 (+55.1%); leads +21.1%.

#### 10.1.8 `gaDemo()`: demo GA4 payload
- `timeseries`: `timeseries(28)` mapped to `{date, sessions, users: round(sessions*0.72), conversions: round(leads*0.6)}`. 28 rows, 2026-06-18 → 2026-07-15.
- `totals`: sessions Σ = 132,174; users `round(total*0.72)` = 95,165; conversions Σ = 1,399; `avgEngagementSec = between(95,190)` (varies per call).
- `topChannels`: `[Organic Search .34, Direct .21, Paid Search .18, Referral .12, Social .10, Email .05]` → `{channel, sessions: round(total*share), sharePct: share*100 (1dp)}`.
- `topPages`: `/` between(9000,15000); `/pricing` 3000–6000; `/product` 2500–5000; `/blog/marketing-roi` 1500–3500; `/demo` 1000–2500 (random per call).
- `topCountries` (country, ISO alpha-2, share): United States US .38; India IN .16; United Kingdom GB .11; Germany DE .08; Canada CA .07; Australia AU .05; France FR .04; Brazil BR .035; Netherlands NL .03; Singapore SG .025. Each maps to `{country, countryId, sessions: round(total*share), users: round(total*share*0.72), sharePct}`.
- `topRegions` (region, country, share): California/US .14; Texas/US .09; New York/US .08; Maharashtra/India .07; England/UK .06; Ontario/Canada .05; Bavaria/Germany .04; New South Wales/Australia .03. Each maps to `{region, country, sessions, users}`.
- Also returns `live:false`, `propertyId:'demo'`.

#### 10.1.9 `integrationsCatalog` (13 entries)

Every entry has: `id, name, category, description, data ('real'|'sample'), sync ('inbound'|'outbound'|'bidirectional'), method ('API'|'iCal feed'), route, cta`.

| id | name | category | data | sync | method | route | cta | description |
|---|---|---|---|---|---|---|---|---|
| google-analytics | Google Analytics 4 | Web Analytics | real | inbound | API | /analytics | View analytics | Sessions, users, conversions and channel data from your GA4 property. |
| salesforce | Salesforce | CRM | sample | bidirectional | API | /attribution | View attribution | Opportunities, campaign responses and closed-won revenue powering your pipeline and attribution. |
| sap | SAP | ERP | sample | bidirectional | API | /budget | View plan vs actuals | Actuals, purchase orders and cost centres from your ERP — powering plan-vs-actual budgets and procurement. |
| google-ads | Google Ads | Advertising | sample | inbound | API | /optimizer | Optimize ad spend | Spend, impressions and clicks by campaign and channel. |
| quickbooks | QuickBooks | Finance | sample | inbound | API | /budget | View plan vs actuals | Actual expenses for plan-vs-actual budget tracking. |
| netsuite | NetSuite | Finance & Purchasing | sample | bidirectional | API | /procurement | View procurement | Purchase orders, invoices and receipts synced to purchase requests. |
| coupa | Coupa | Finance & Purchasing | sample | bidirectional | API | /procurement | View purchase requests | Procurement approvals and spend management for purchase requests. |
| google-calendar | Google Calendar | Calendars | sample | outbound | iCal feed | /planning | Connect calendar | Subscribe to the marketing calendar via its live .ics feed (Planning → Calendar → Connect calendar). |
| outlook | Outlook Calendar | Calendars | sample | outbound | iCal feed | /planning | Connect calendar | Subscribe to scheduled tactics via the live .ics feed, or import a one-time snapshot. |
| bigquery | Google BigQuery | Data Warehouse | sample | inbound | API | /attribution | View analytics | Query performance and cost data from your SQL warehouse. |
| snowflake | Snowflake | Data Warehouse | sample | inbound | API | /attribution | View analytics | Blend warehouse data with plans, costs and performance. |
| asana | Asana | Project & Content | sample | bidirectional | API | /planning | Open planning | Link tactics to production tasks and content projects. |
| jira | Jira | Project & Content | sample | bidirectional | API | /planning | Open planning | Track content and campaign work items against tactics. |

Only GA is genuinely live. The others are honest labels over the sample data (the Google Calendar card gets real status from a separate endpoint; see §10.11.7).

### 10.2 Uploaded campaign dataset: `server/services/campaignDataset.js`

#### 10.2.1 Accepted formats and transport
- Client `readUpload(file)` (`lib/files.js`):
  - `.csv/.txt/.tsv` → read as text → `{csv, name}`.
  - Anything else (`.xlsx/.xls/.ods`) → `readAsDataURL` → `{fileBase64, name}`.
  - File input `accept=".csv,.tsv,.txt,.xlsx,.xls,.ods,text/csv"`.
- Server `toCsvText({csv,fileBase64,name})`:
  - Non-empty `csv` is used as-is.
  - Base64 with a .csv/.txt/.tsv name is decoded as utf8.
  - Otherwise SheetJS reads it (`XLSX.read(buf,{type:'buffer',cellDates:true})`), takes the first sheet only and runs `sheet_to_csv(ws,{dateNF:'yyyy-mm-dd', blankrows:false})`. Error if there is no sheet: "The workbook has no readable sheet."; no content: "No file content provided."
- `parseCsv`: RFC-ish, comma-only (so a .tsv will not parse into columns). Handles quoted fields, escaped `""` and CR/CRLF, and drops rows whose cells are all blank.

#### 10.2.2 Column mapping
Header matching is case-insensitive, exact after trim. The first matching header wins.

| key | aliases |
|---|---|
| company | `company` |
| type | `campaign_type`, `type` |
| audience | `target_audience`, `audience` |
| channel | `channel_used`, `channel` |
| conversion | `conversion_rate`, `conversion` |
| acquisition | `acquisition_cost`, `acquisition`, `cost` |
| roi | `roi` |
| location | `location` |
| date | `date` |
| clicks | `clicks` |
| impressions | `impressions` |
| engagement | `engagement_score`, `engagement` |
| segment | `customer_segment`, `segment` |

- Ignored columns: Campaign_ID, Duration, Language.
- Expected source format (as in `marketing_campaign_dataset.xlsx` / `zenreach-campaign-500.xlsx`): `Campaign_ID, Company, Campaign_Type, Target_Audience, Duration, Channel_Used, Conversion_Rate (fraction e.g. 0.04), Acquisition_Cost ("$16,174.00"), ROI, Location, [Language], Date, Clicks, Impressions, Engagement_Score, Customer_Segment`.

#### 10.2.3 Validation and errors
- Fewer than 2 rows → "CSV needs a header row and at least one data row."
- None of channel, type or roi found → "This does not look like a campaign dataset (no recognizable columns)."
- No records → "No data rows found."
- The check is permissive: a tactics file with a `type` or `channel` column passes. The currently persisted `server/data/campaignDataset.json` is exactly that case: 1,500 rows from `zenreach-tactics-1500.xlsx`, all metrics 0, dateRange null.

#### 10.2.4 Per-row normalisation
- `num(v)` strips `$ , % whitespace` and calls `Number`; non-finite → 0.
- Text dimensions are trimmed, with a default of `'—'`.
- Month: regex `/(\d{4})[-/](\d{1,2})/` → `YYYY-MM`; otherwise `new Date(raw)` → local `YYYY-MM`; otherwise `''`.

#### 10.2.5 Aggregation (only the summary is persisted; the raw rows never are)
- **kpis:**
  - `campaigns = n`
  - `avgConversionRate = +(Σconv/n*100).toFixed(2)` (assumes fractional rates; a "4%" value becomes 400)
  - `avgRoi = +(Σroi/n).toFixed(2)`
  - `totalClicks`, `totalImpressions`
  - `totalAcquisitionCost = round(Σ)`
  - `avgEngagement` (2dp)
  - `ctr = Σclicks/Σimpressions*100` (2dp, 0 if no impressions)
- **`groupBy(records,key)`:** per group `{key, count, clicks Σ, impressions Σ, acquisitionCost round Σ, conversionRate avg*100 2dp, roi avg 2dp, engagement avg 2dp, ctr 2dp}`. Then the generic `key` is renamed to the dimension name.
- **Summary object:**
  - `meta {rows, source: name, updatedAt ISO, dateRange {start,end} | null, dimensions {channels, types, segments distinct counts}}`
  - `kpis`
  - `byChannel`, `byType`, `bySegment`, `byAudience`: all groups, sorted by count desc
  - `byLocation`: top 12 by count
  - `monthly`: sorted months, `{date: 'YYYY-MM', clicks Σ, impressions Σ, conversionRate avg*100, roi avg, acquisitionCost Σ}`
  - `topCompanies`: top 15 by clicks
- **Persistence:** `writeJson('campaignDataset.json')` plus an in-memory `CACHE`. Replacing overwrites. There is no delete or unload endpoint.
- **Return values:** `ingestCsv` returns `{loaded:true, ...meta}`. `getMeta()` returns `{loaded:true,...meta}` or `{loaded:false}`.

#### 10.2.6 What the dataset changes elsewhere
It reaches only the Builder and the assistant's dashboard wizard. Overview, Campaigns, Goals, Insights, AI analysis, Optimizer and Attribution all still read the static mock campaigns.

- **`builderBundle()`** adds data keys `cmpKpis, cmpChannel, cmpType, cmpSegment, cmpAudience, cmpLocation, cmpMonthly, cmpCompanies`.
- **`builderSources()`** adds catalog sources, all with `group: 'Campaign dataset'`:
  - `cmpFlex`: "Campaign dataset — custom axes (pick X & Y)". Kinds series and breakdown, `flexible:true`, default x `channel`. `dims` = Channel(cmpChannel) / Campaign type(cmpType) / Customer segment(cmpSegment) / Target audience(cmpAudience) / Location(cmpLocation).
  - `METRIC_FIELDS`, shared by cmpFlex and the five dimension sources:
    - count "Campaigns" (number)
    - roi "Avg ROI" (x)
    - conversionRate "Avg conversion %" (percent)
    - clicks (number)
    - impressions (number)
    - acquisitionCost "Acquisition cost" (money)
    - engagement "Avg engagement" (number)
    - ctr "CTR %" (percent)
  - `cmpKpis` "Campaign dataset — KPIs" (single):
    - campaigns "Campaigns"
    - avgRoi "Average ROI" (x)
    - avgConversionRate "Avg conversion %"
    - totalClicks "Total clicks"
    - totalImpressions "Total impressions"
    - totalAcquisitionCost "Total acquisition cost" (money)
    - avgEngagement "Avg engagement score"
    - ctr "Overall CTR %"
  - `cmpChannel`, `cmpType`, `cmpSegment`, `cmpAudience`, `cmpLocation`: named "Campaign dataset — by channel / by campaign type / by customer segment / by target audience / by location", kinds series and breakdown, x = the dimension, METRIC_FIELDS.
  - `cmpMonthly` "Campaign dataset — monthly trend" (series, x `date`): clicks, impressions, conversionRate "Conversion %", roi "ROI", acquisitionCost.
  - `cmpCompanies` "Campaign dataset — top companies" (table). Columns: company "Company", count "Campaigns", roi "Avg ROI", conversionRate "Conv %", clicks, impressions.
- **`catalog().datasetSources`** lists those ids.
- **Assistant wizard:** offers "My uploaded campaign dataset  (recommended)" as a focus, recommends it by default and has a dataset blueprint (§10.10).
- **Builder UI** (Builder.jsx, for context):
  - Toolbar line: "Dataset: **N rows** from {source} · {start} → {end}" or "No external dataset loaded yet."
  - Button "Upload dataset" / "Replace dataset" / "Loading…" (disabled when the user cannot edit).
  - After upload it refetches catalog, bundle and meta, then toasts "Dataset loaded: N rows. New sources are in "Add widget"."
  - Helper copy: "Add a widget and pick a **"Campaign dataset — …"** source to chart it (by channel, type, segment, audience, location, monthly trend, or top companies)."

### 10.3 Google Analytics 4: `server/services/googleAnalytics.js`

#### 10.3.1 Configuration and environment variables
- `GA_PROPERTY_ID`: numeric GA4 property id.
- Credentials, either of:
  - `GOOGLE_CREDENTIALS_JSON`: inline service-account JSON; takes precedence.
  - `GOOGLE_APPLICATION_CREDENTIALS`: key-file path. A relative path resolves against `PROJECT_ROOT = path.resolve(server/services, '../..')`, the repo root.
- `isConfigured()` = property id plus one of the credentials. `analyticsStatus()` returns `{configured}`. The startup log reads "LIVE Google Analytics" or "DEMO mode".
- `.env.example` setup steps:
  1. Create a GCP service account.
  2. Enable the "Google Analytics Data API".
  3. Put the JSON key in the project root.
  4. Add the service-account email as a Viewer in GA4 Property Access Management.
- **Client:** `BetaAnalyticsDataClient` from `@google-analytics/data`, dynamically imported and memoised in `clientPromise` (one client per process). Constructed as `new BetaAnalyticsDataClient({credentials})` or `({keyFilename})`.

#### 10.3.2 `getAnalytics()`
Not configured → `gaDemo()`. Otherwise it runs six `runReport` calls in parallel on `properties/${GA_PROPERTY_ID}`, all with `dateRanges [{startDate:'28daysAgo', endDate:'today'}]`:

1. Timeseries: dimension `date`; metrics `sessions, totalUsers, conversions`; order by date dimension.
2. Totals: no dimension; metrics `sessions, totalUsers, conversions, averageSessionDuration`.
3. Channels: dimension `sessionDefaultChannelGroup`; metric `sessions`; order by sessions desc; limit 6.
4. Pages: dimension `pagePath`; metric `screenPageViews`; desc; limit 5.
5. Countries: dimensions `country, countryId` (ISO alpha-2); metrics `sessions, totalUsers`; desc; limit 10.
6. Regions: dimensions `region, country`; metrics `sessions, totalUsers`; desc; limit 8.

Mapping:
- `timeseries[] = {date: YYYYMMDD → YYYY-MM-DD, sessions, users, conversions}`; `totalSessions` = Σ timeseries sessions.
- `topChannels {channel, sessions, sharePct = sessions/totalSessions*100 1dp}`.
- `topPages {page, views}`.
- `topCountries`: filters out empty and `(not set)` → `{country, countryId, sessions, users, sharePct}`.
- `topRegions`: same filter → `{region, country, sessions, users}`.
- `totals`: `sessions = totals-report sessions || totalSessions`, users, conversions, `avgEngagementSec = round(averageSessionDuration)`.
- Result: `{live:true, propertyId, totals, timeseries, topChannels, topPages, topCountries, topRegions}`.

Guards:
- Zero sessions or empty timeseries → return `gaDemo()` with `error: 'Live GA4 returned no data for the period yet — showing demo figures so the page stays populated.'`.
- Any exception → `gaDemo()` with `error: 'Live GA query failed, showing demo data: <msg>'`.
- No server-side caching: every page load runs all six GA queries.

#### 10.3.3 `getRealtime()`
- Not configured → `{live:false, activeUsers: 4 + floor(random*14) (4–17), views: 0}`.
- Configured → `runRealtimeReport` with metrics `activeUsers, screenPageViews`.
  - activeUsers === 0 → `{live:false, activeUsers 4–17 random, views 60 + floor(random*260)}`.
  - Otherwise `{live:true, activeUsers, views}`.
  - Error → `{live:false, activeUsers:0, views:0, error}`.
- The client polls every 15 s.

#### 10.3.4 Traffic generator: `client/public/tracking-test.html`
An identical copy also exists at the repo root.
- Standalone page "ZenReach — GA4 Traffic Generator", served at `/tracking-test.html`. It sends real hits to a GA4 property so the live connection can be demonstrated.
- **Card 1 "1 · Connect your property":** input placeholder `G-XXXXXXXXXX`, "Connect" button, status dot plus "Not connected" / "Connected to {id} — sending live".
  - Validation `/^G-[A-Z0-9]{6,}$/i`; failure alerts "Please paste a valid Measurement ID that looks like G-XXXXXXXXXX".
  - Connect injects `https://www.googletagmanager.com/gtag/js?id=…`, then calls `gtag('js', new Date())` and `gtag('config', id, {send_page_view:false, debug_mode:true})`.
  - It stores the id in `localStorage.ga_mid` and sends an initial page_view for `/`.
- **Card 2 "2 · Generate traffic":** a 2-column grid of buttons, disabled until connected.
  - 5 pages: `/` Home, `/pricing` Pricing, `/product` Product, `/blog/marketing-roi` "Blog — Marketing ROI", `/demo` "Book a demo". Label "Visit X", sub-label "page_view · path". Each sends `gtag('event','page_view',{page_title, page_path, page_location:'https://zenreach.demo'+path})`.
  - 4 events: view_pricing "Viewed pricing", start_demo "Started demo", sign_up "Signed up", generate_lead "Generated a lead". Each sends `gtag('event',name,{engagement_time_msec: 1500–5499 random})`.
  - Button "⚡ Auto-generate 25 hits": 25 hits, one every 350 ms, 60% page_view and 40% event, chosen at random; ends with "✔ Burst complete — 25 hits sent."
- **Card 3 "3 · Live send log":** monospace dark log, newest first, `[time] msg`; initial text "Waiting to connect…".
- Lead copy explains: watch GA4 → Reports → Realtime; ZenReach dashboards update a few hours later. A hint suggests multiple tabs or browsers to create sessions.
- Prefill order: `?id=` → `localStorage.ga_mid` → a hard-coded default measurement ID (`DEFAULT_ID`).

### 10.4 Goals: `server/services/goals.js`

- `ELAPSED = 0.62` (Q3 2026, "~day 57 of 92").
- `status(forecastPct)`: ≥100 → `on-track`; ≥85 → `at-risk`; else `off-track`.
- Definitions, using `overview()`:
  - Attributed revenue: unit `$`, actual totalRevenue, targetFactor 0.98, project true.
  - Pipeline generated: `$`, totalPipeline, 1.06, project.
  - Total leads: `#`, totalLeads, 0.95, project.
  - Marketing ROI: `x`, blendedRoi, `absoluteTarget 2.5`, project false.
- Formulas:
  - `target = absoluteTarget ?? round((actual/ELAPSED)*targetFactor)`
  - `forecast = project ? round(actual/ELAPSED) : actual` (straight-line run-rate to quarter end)
  - `attainmentPct = +(actual/target*100).toFixed(1)`
  - `forecastPct = +(forecast/target*100).toFixed(1)`
  - `status = status(forecastPct)`
  - Because forecast = actual/E and target = actual/E × factor, `forecastPct = 100/targetFactor` for projected goals. So revenue is always 102.0% (on-track), pipeline 94.3% (at-risk) and leads 105.3% (on-track).
- Output `{period:'Q3 2026', elapsedPct: 62, daysRemaining: 35 (hard-coded), goals:[{metric, unit, actual, target, forecast, attainmentPct, forecastPct, status}]}`.
- Reference values:
  - Revenue: actual 4,755,000; target 7,515,968; forecast 7,669,355; attainment 63.3%.
  - Pipeline: 9,680,000; target 16,549,677; forecast 15,612,903; attainment 58.5%.
  - Leads: 9,700; target 14,863; forecast 15,645; attainment 65.3%.
  - ROI: 4.24 vs 2.5 → 169.6% / 169.6%, on-track.

### 10.5 Rule-based insights: `server/services/insights.js`

Output items are `{id, severity, title, detail, metric}`; `pct(n)` renders as `+x.x%` / `-x.x%`. The rules run in this order:

1. **best-roi** (always): top campaign by ROI.
   - severity `good`; title "`{name}` is your top performer"
   - detail "It's returning {roi 2dp}x on spend via {channel}. Consider shifting more budget here."
   - metric "{roi}x ROI"
2. **low-roi**: the lowest-ROI campaign excluding status Planned, emitted only if roi < 2.
   - `serious`; "`{name}` is underperforming"
   - "Only {roi}x return on {channel}. Review targeting or pause to protect budget."
   - metric "{roi}x ROI"
3. **budget-pacing**: the Active campaign with the highest spent/budget, only if > 0.88.
   - `warning`; "`{name}` has nearly exhausted its budget"
   - "{used}% of budget spent with the quarter still running. Top up now to avoid a mid-flight stall."
   - metric "{used}% spent"
4. **overspend**: the channel in `budget().byChannel` with the largest `actual − planned`, only if > 0.
   - `warning`; "`{channel}` is over its planned budget"
   - "Actual spend exceeds plan by ${over.toLocaleString()}. Reforecast or reallocate before month-end."
   - metric "+${round(over/1000)}K"
   - An `eff` ratio array is computed but unused.
5. **momentum** (always): from `timeseries(30)`, last 7 days vs prior 7 days revenue; `delta = (last7−prev7)/prev7*100`.
   - severity `good` if ≥0, else `serious`; "Revenue is accelerating/slowing week-over-week"
   - "Last 7 days generated ${round(last7/1000)}K vs ${round(prev7/1000)}K the week before ({pct})."
   - metric pct
6. **blended-roi** (always): severity `good` if blendedRoi ≥2, else `warning`.
   - "Blended marketing ROI is {x}x"
   - "Every $1 of marketing spend is generating ${x} of attributed revenue across all channels."

With the demo data only rules 1, 5 and 6 fire: "Nova Developer Community is your top performer" (7.32x), "Revenue is accelerating week-over-week" (+55.1%, $240K vs $155K), and "Blended marketing ROI is 4.24x". The worst ROI is 2.67, which is not below 2. No active campaign is over 88% spent (Analyst Relations is at 97% but Completed). No channel is overspent.

### 10.6 AI analysis: `server/services/analysis.js` (`GET /api/insights/ai`)

#### 10.6.1 `computeSignals()`: the JSON handed to the LLM
- `overview`: full `overview()`.
- `channelRoi`: per campaign channel `{channel, spend Σspent, revenue Σ, roi 2dp}`, sorted by roi desc. Demo: Organic 7.32, Content Synd 5.31, Paid Social 4.62, Events 3.94, Paid Search 3.20.
- `blendedRoi`.
- `momentum`: `{revenuePct, leadsPct}` (1dp % change, last 7 vs prior 7 of `timeseries(30)`), plus `last7Revenue` and `prev7Revenue`. Demo: 55.1 / 21.1 / 239,675 / 154,577.
- `budgetVariance`: from `budget().byChannel`, `{channel, planned, actual, diff = actual−planned, pct = (actual/planned−1)*100 1dp}`, sorted by diff desc.
- `goals`: `{metric, status, attainmentPct, forecastPct}`.
- `reallocation`: `{liftPct: recommend().liftPct, bestChannel, bestEfficiency}` taken from the highest-efficiency channel. Demo: 6.7, Organic / SEO, 7.317.
- `concentration`:
  - `topChannel` and `topChannelSharePct`, from revenueByChannel sorted. Demo: Content Syndication 41.5.
  - `topCampaign` (by revenue) and `topCampaignSharePct`. Demo: Nova Enterprise ABM 32.1.

#### 10.6.2 `computedInsights()`: deterministic fallback
Up to 5 items, each `{severity, title, detail, action, impact}`, in this order:

1. good, "Rebalancing budget could add {liftPct}% revenue"
   - detail "At the same total spend, tilting toward {bestChannel} (best efficiency, {bestEfficiency}x) is projected to lift attributed revenue."
   - action "Open the Optimizer and apply the recommended mix."
   - impact "+{liftPct}% revenue"
2. good/serious, "Revenue up|down {abs}% week-over-week"
   - detail "Last 7 days generated {$full} vs {$full} the prior week."
   - action "Double down on the channels driving the lift." or "Investigate which channels slipped this week."
   - impact "{±x}% WoW"
3. warning, "{worst} lags at {worstRoi}x vs {bestRoi}x"
   - detail "{best} is {(best/worst).toFixed(1)}× more efficient. {worst} sits below your {blended}x blended average."
   - action "Rework or trim {worst}; move that budget to {best}."
   - impact "{ratio}× ROI gap"
4. Only if any goal is not on-track: severity critical if any is off-track, else warning; "{n} goal(s) off pace this quarter"
   - detail "{metric} forecasts {forecastPct}% of target; …."
   - action "Shift focus to the metrics trailing target on the Goals page."
   - impact = metric names joined
5. Only if topCampaignSharePct > 22: warning, "{x}% of revenue rides on one campaign"
   - detail "{campaign} drives {x}% of revenue — risky if it dips."
   - action "Scale a second proven campaign to diversify."
   - impact "{x}% concentration"
6. Only if budgetVariance[0].diff > 0: warning, "{channel} is {pct}% over plan"
   - detail "Actual spend on {channel} exceeds plan by {$}."
   - action "Reforecast or cap {channel} spend on the Budget page."
   - impact "+{$} spend"

`.slice(0,5)`. Demo output, 5 items:
- "Rebalancing budget could add 6.7% revenue"
- "Revenue up 55.1% week-over-week"
- "Paid Search lags at 3.2x vs 7.32x" (2.3× gap)
- "1 goal off pace this quarter" (Pipeline generated 94.3%)
- "32.1% of revenue rides on one campaign"

#### 10.6.3 LLM call
- **Providers, in order:** Azure OpenAI if `AZURE_OPENAI_API_KEY`, `AZURE_OPENAI_ENDPOINT` and `AZURE_OPENAI_DEPLOYMENT_NAME` are all set; otherwise Anthropic if `ANTHROPIC_API_KEY` is set.
  - Note: analysis does not use `OPENAI_API_KEY`.
  - If Azure is configured but fails, Anthropic is not tried; it falls to the computed engine.
- **System prompt (verbatim):**

```
You are a senior marketing analyst briefing a CMO. Turn marketing data into sharp, non-obvious, decision-ready insights — never generic restatements of the numbers.
Return STRICT JSON only: {"insights":[{"title","detail","action","impact","severity"}]}.
Rules:
- Exactly 4 insights, ranked most important first.
- "title": <= 9 words, states the finding WITH a number.
- "detail": <= 28 words on why it matters to the business.
- "action": one concrete next step; reference a ZenReach page (Optimizer, Budget, Campaigns, Goals, Analytics) when relevant.
- "impact": a short quantified phrase (e.g. "+7.4% revenue", "$46K at risk", "3.2x ROI gap").
- "severity": one of good | warning | serious | critical.
- Use ONLY numbers present in the DATA. Never invent metrics. Prefer connecting two signals into one insight (e.g. momentum + concentration).
```

- **User message:** `DATA (JSON):\n` + `JSON.stringify(computeSignals())`.
- **Azure request:**
  - `POST {endpoint without trailing /}/openai/deployments/{DEPLOYMENT}/chat/completions?api-version={AZURE_OPENAI_API_VERSION || '2024-08-01-preview'}`
  - Header `api-key`.
  - Body `{messages:[system,user], max_tokens:900, temperature:0.5, response_format:{type:'json_object'}}`.
  - Up to 3 attempts, sleeping 600 ms and then 1200 ms before the retries. It retries only on 429/500/502/503/504; any other status breaks out immediately.
- **Anthropic request:** `new Anthropic()` (env key), `model: ANTHROPIC_MODEL || 'claude-opus-4-8'`, `max_tokens:1024`, system = SYSTEM + "\nRespond with JSON only, no prose, no code fences.", and the concatenated text blocks are taken as the reply.
- **`normalize(raw)`:**
  - Strips leading ```` ```json ```` and a trailing ```` ``` ````, then parses JSON. Accepts either an array or `{insights:[]}`.
  - Filters to items with a title and keeps at most 5.
  - Coerces every field to a string; severity must be good/warning/serious/critical, defaulting to `warning`. Parse failure → null.
- **Result:**
  - AI: `{source:'ai', provider: 'Azure OpenAI · gpt-4o' (a fixed label, whatever the deployment) | 'Claude', insights}`.
  - Otherwise: `{source:'computed', provider:'ZenReach analytics engine', insights: computedInsights()}`.
- **Cache:** module-level `{until, payload}`. TTL is 10 min for an AI result and 30 s for the computed fallback, so a transient failure retries soon.

### 10.7 Optimizer: `server/services/optimizer.js`

- **`channelModel()`:** per campaign channel (Map, insertion order: Paid Search, Content Syndication, Events & Webinars, Paid Social, Organic / SEO), `spend = Σspent` and `efficiency = +(Σrevenue/Σspent).toFixed(3)`.
  - Demo: PS 488,000 @3.197; CS 372,000 @5.309; E&W 142,000 @3.944; PSoc 78,000 @4.615; Org 41,000 @7.317.
- **`project(allocations)`:** for each `{channel, amount}`:
  - `amount = max(0, round(amount))`
  - `projectedRevenue = round(max(0,amount) × eff(channel) || 1)`; unknown channels get efficiency 1.
  - Returns `{totalSpend, projectedRevenue, breakdown[]}`. Linear model, no diminishing returns.
- **`recommend()`:**
  1. `total` = Σ current spend (1,121,000).
  2. Weights `w = efficiency^1.6`.
  3. `ideal = w/Σw × total`.
  4. Clamp to `[0.5×current, 1.8×current]`.
  5. Renormalise so Σ equals total exactly: `round(amount/allocTotal × total)`. Because renormalisation runs after clamping, final amounts can exceed the 1.8× band.
  6. Baseline = `project(current)`; optimized = `project(alloc)`.
  7. `liftPct = +((opt−base)/base*100).toFixed(1)`.
  - Returns `{channels: channelModel(), baseline, recommended:{...optimized, allocations}, liftPct}`.
  - Demo:
    - Baseline projected 4,755,099.
    - Recommended allocations: PS 318,433; CS 322,545; E&W 200,479; PSoc 183,230; Org 96,313. These come from pre-normalisation values of 244,000 (floor), 247.1K, 153.6K, 140.4K (cap) and 73.8K (cap), scaled ×1.305.
    - Projected 5,071,438; lift **6.7%**.

### 10.8 Attribution: `server/services/attribution.js`

#### 10.8.1 Data
- `CHANNELS = ['Paid Search','Paid Social','Organic / SEO','Email','Events & Webinars','Referral','Direct']` (seven; note Content Syndication is absent and Referral/Direct are added).
- `CHANNEL_SPEND` (USD): Paid Search 488,000; Paid Social 346,000; Organic / SEO 41,000; Email 15,000; Events & Webinars 142,000; Referral 20,000; Direct 0.
  - The comments say Paid Social = "Enterprise ABM + Retargeting", but ABM is Content Syndication in mockData; the narrative is inconsistent.
- **Journeys:** built once and memoised in `JOURNEYS`, using `mulberry32(1337)`. 600 journeys, each:
  - `len = 1 + floor(rand*5)` (1–5 touches).
  - Each touch `j` is `{channel: CHANNELS[floor(rand*7)], daysAgo: round((len−j)*(2 + rand*6))}`; earlier touches are older.
  - `converted = rand < 0.42`.
  - `revenue = converted ? round((1500 + rand*12000)/100)*100 : 0`.
  - Totals: 266 conversions; 1,954,600 revenue.

#### 10.8.2 Nine models
Each model returns per-touch weights; `n` = number of touches.

| id | name | desc (UI copy) | weights |
|---|---|---|---|
| first | First touch | 100% credit to the first interaction | [1,0,…] |
| last | Last touch | 100% credit to the final interaction | […,0,1] |
| last_non_direct | Last non-direct | Last touch, skipping Direct | walk back from the last touch while channel is Direct (stop at index 0); 1 at that index |
| linear | Linear | Equal credit to every touch | 1/n each |
| time_decay | Time decay | More credit to recent touches (7-day half-life) | raw `2^(−daysAgo/7)`, normalised |
| position | Position-based (U) | 40% first, 40% last, 20% middle | n=1 → [1]; n=2 → [.5,.5]; else first and last 0.4, each middle 0.2/(n−2) |
| w_shaped | W-shaped | 30% first, 30% middle marker, 30% last, 10% rest | n≤2 → equal; marker = floor(n/2); first, last and marker get 0.3; the rest get 0.1/(n−3) (0 if no rest). **Bug:** at n=3 the weights sum to 0.9, so W-shaped credits only 1,916,870 of 1,954,600 |
| full_path | Full-path | Four key stages weighted, remainder shared | n≤4 → equal; keys = [0, floor(n/3), floor(2n/3), n−1] get 0.225 each; the rest share 0.1 |
| custom | Custom (data-driven) | Recency-weighted blend tuned to this data | linear ramp: raw i+1, normalised |
| (default) | | | equal |

#### 10.8.3 Functions
- **`attribute(modelId='linear')`:**
  - An unknown model falls back to `linear`.
  - Over converted journeys only, credits each touch `w[i]` conversions and `w[i]×revenue`.
  - Rows `{channel, conversions 1dp, revenue round, sharePct = credited/totalRevenue*100 1dp}`, sorted by revenue desc.
  - Returns `{model, models: ATTRIBUTION_MODELS, rows, totals:{conversions, revenue}}`.
  - Demo, Position-based: Email 314,193 (16.1%), Organic 305,153, Paid Search 294,647, Direct 290,070, Referral 260,477, Paid Social 247,277, Events 242,783.
- **`compareModels()`:** `{channels: CHANNELS, models:[{model, name, credited: revenue per channel in CHANNELS order}]}`.
- **`funnel()`:** fixed stage volumes, each with `conversionPct = vol/prevVol*100` (1dp; first stage 100) and `dropPct = (prev−vol)/prev*100` (1dp; first stage 0).

  | stage | volume | step % |
  |---|---|---|
  | Impressions | 4,200,000 | 100 |
  | Visits | 168,000 | 4.0 |
  | Leads | 24,500 | 14.6 |
  | MQLs | 9,800 | 40.0 |
  | SQLs | 3,430 | 35.0 |
  | Opportunities | 1,200 | 35.0 |
  | Closed won | 384 | 32.0 |

- **`reallocation(modelId='position')`:**
  - Rows `{channel, spend: CHANNEL_SPEND, revenue: credited, roi: spend>0 ? +(rev/spend).toFixed(2) : null}`, sorted by roi desc with nulls last (treated as −1).
  - `paid` = rows with non-null roi; best = paid[0], worst = last.
  - Recommendation: `{from: worst, to: best, amount: round(CHANNEL_SPEND[worst]×0.2), rationale: "{best} returns {roi}× vs {worst}'s {roi}× under {model} attribution."}`, or null if best and worst are the same channel. The rationale uses the model id, e.g. "position".
  - Demo (position): Email 20.95×, Referral 13.02×, Organic 7.44×, Events 1.71×, Paid Social 0.71×, Paid Search 0.60×, Direct —. Move $97,600 from Paid Search → Email. Every model recommends Paid Search → Email.
- **`forecast()`:**
  - Uses `attribute('linear').totals.revenue` as one quarter of revenue (1,954,600).
  - `growth = 0.08` QoQ; `quarters q1..q4 = round(Q × 1.08^(q−1))` → 1,954,600; 2,110,968; 2,279,845; 2,462,233.
  - `projectedAnnual` = Σ = 8,807,646.
  - `target = round(projected × 1.1)` = 9,688,411.
  - `attainmentPct = projected/target*100` → always 90.9 (= 1/1.1).
  - Status: ≥100 on-track, ≥90 at-risk, else off-track → always **at-risk**.
  - Returns `{quarterRevenue, growthPct: 8, quarters:[{quarter:'Q1',revenue}…], projectedAnnual, target, attainmentPct, status}`.

### 10.9 "Ask ZenReach" chat analyst: `server/services/assistant.js`

#### 10.9.1 Providers and environment variables
The providers cascade on failure: OpenAI → Azure → Anthropic → local rules.
- **OpenAI (preferred):** `OPENAI_API_KEY`, `OPENAI_MODEL` (default `'gpt-5'`). Uses `client.chat.completions.create({model, max_completion_tokens:1024, messages:[system, ...convo, user]})` with no temperature (GPT-5 accepts only the default). An empty reply throws.
- **Azure OpenAI:** `AZURE_OPENAI_API_KEY`, `AZURE_OPENAI_ENDPOINT`, `AZURE_OPENAI_DEPLOYMENT_NAME`, `AZURE_OPENAI_API_VERSION` (default `2024-08-01-preview`). Same URL pattern as §10.6.3; body `{messages, max_tokens:800, temperature:0.4}`; no retries; error text "Azure OpenAI {status}: {body≤200}".
- **Anthropic:** `ANTHROPIC_API_KEY`, `ANTHROPIC_MODEL` (default `'claude-opus-4-8'`); `max_tokens 1024`; text blocks joined with "\n".
- `.env.example` documents OpenAI and Anthropic but not the Azure variables; the live `server/.env` defines OpenAI and Azure.
- **`assistantStatus()`:**
  - `{ai:true, provider:'OpenAI · {model}'}`, else
  - `{ai:true, provider:'Azure OpenAI · gpt-4o'}`, else
  - `{ai:true, provider:'Claude'}`, else
  - `{ai:false, provider:'built-in analyst'}`.

#### 10.9.2 Prompt
- **SYSTEM (verbatim):** "You are ZenReach's marketing analyst assistant. You help marketing leaders understand their performance data and make decisions. Answer ONLY from the DATA CONTEXT provided in each request — never invent numbers. Be concise and specific: lead with the answer, cite the exact figures, and give one clear recommendation when relevant. Use a confident, friendly, executive tone. Keep answers under ~120 words unless asked for detail. Format money and percentages cleanly. If the data doesn't cover the question, say so and suggest what to look at instead."
- **History:** the last 6 messages, role-normalised to assistant/user, content stringified.
- **User content:** `DATA CONTEXT:\n{buildContext()}\n\nQUESTION: {message}`. `buildContext()` has 5 blocks joined by blank lines:
  1. `Overview: attributed revenue $X, spend $X, blended ROI Nx, pipeline $X, N leads, N active campaigns, budget N% utilized.`
  2. `Top campaigns by ROI:` 4 bullets `Name (channel, region): spend $, pipeline $, ROI Nx, Status`.
  3. `Revenue by channel:` bullets sorted desc.
  4. `Goals (Q3 2026):` bullets `metric: actual N[x] vs target N — status`.
  5. `Auto-insights:` bullets of the `insights()` titles.

  Money is formatted `$` + `round().toLocaleString()`.

#### 10.9.3 Local rule-based answers (`localAnswer`)
Lower-cased regexes are tested in this order; the first match wins:

1. `/best|top|winner|highest roi|performing/` → "Your top performer is **{name}** on {channel}, returning **{roi}x** ({pipeline} pipeline from {spent} spend). It's a strong candidate for more budget."
2. `/worst|underperform|pause|lowest|cut/` → lowest ROI excluding Planned: "**{name}** ({channel}) is lagging at **{roi}x** ROI. I'd review its targeting or reallocate that budget to higher-return channels." Demo answer: Nova Demand Gen 2.67x.
3. `/budget|reallocat|move|optimi|where should/` → "Reallocating toward higher-efficiency channels could lift attributed revenue by about **{liftPct}%** at the same total spend. **{best}** has the best efficiency ({eff}x) — shift budget there. Open the Optimizer to model it live."
4. `/goal|target|on track|forecast|quarter/`:
   - All on track: "You're on track across all {n} goals for {period}, with {days} days left. Blended ROI is already at {x}x."
   - Otherwise: "For {period}, {k} of {n} goals are on track. Watch: {metric} ({status}, forecast {pct}% of target); …."
5. `/channel|revenue by|which channel/` → "**{top}** is your biggest revenue channel at **{$}**. See the full split on the Overview dashboard."
6. `/summ|cmo|report|overview|how are we/` → "This quarter: **{rev}** attributed revenue on **{spend}** spend — a blended **{x}x ROI** — with {pipe} in pipeline and {leads} leads. {n} campaigns are live and budget is {util}% utilized."
   - Quirk: the starter "Summarize this quarter for my CMO." matches rule 4 first (`quarter`) and returns the goals answer.
7. `/roi|return/` → "Blended marketing ROI is **{x}x** — {rev} attributed revenue on {spend} spend."
8. `/hello|hi|hey|help|what can you/` → "Hi! I'm your ZenReach analyst. Ask me about campaign performance, budget optimization, revenue by channel, or whether you're on track to hit your goals."
   - Note `hi` is an unanchored substring test, so words containing "hi" (e.g. "which", "this") can reach it if no earlier rule matched.
9. Default → "Here's the headline: **{rev}** attributed revenue at **{x}x** ROI this quarter. Ask me about your best campaigns, where to move budget, revenue by channel, or goal progress."

#### 10.9.4 Navigation button (`pickLink`)
Computed from the user's question for analyst turns, whether AI or local. Score = number of keyword substrings found. A strictly higher score wins, so ties go to the earlier entry. No match → no button.

| order | route | label | keywords |
|---|---|---|---|
| 1 | /optimizer | Open the Optimizer | optimi, reallocat, move budget, shift budget, what-if, what if, rebalanc, efficien, best mix, allocat |
| 2 | /goals | View Goals | goal, target, on track, off track, forecast, quarter, attainment, pacing, make the number |
| 3 | /budget | View Budget | budget, spend, planned, actual, overspend, over budget, utiliz, pace |
| 4 | /campaigns | View Campaigns | campaign, pipeline, performer, underperform, best roi, top roi, which campaign |
| 5 | /analytics | View Analytics | analytic, traffic, session, visitor, active user, engagement, realtime, real-time, page view, website, conversion |
| 6 | / | Open the dashboard | overview, summar, dashboard, headline, kpi, how are we, how is everything, revenue by channel, which channel, cmo, report |

Examples:
- "Which campaign has the best ROI?" → View Campaigns
- "Where should I move budget?" → Open the Optimizer (move budget 1, allocat 0, budget 1 → tie, optimizer wins by order)
- "Are we on track…goals?" → View Goals
- "Summarize this quarter for my CMO." → Open the dashboard

#### 10.9.5 `ask()` and `assistantTurn()`
- **`ask(message, history)`** returns `{reply, source:'ai'|'local', suggestions: SUGGESTIONS, note?: lastProviderError, link?}`.
  - `SUGGESTIONS = ['Which campaign has the best ROI?','Where should I move budget?','Are we on track to hit our goals?','Summarize this quarter for my CMO.']` is returned but not rendered by the client.
- **`assistantTurn({message, history, flow})`:**
  1. If `flow?.active` → `{source:'builder', suggestions, ...continueBuild(msg, flow)}`.
  2. Else if `isBuildIntent(msg)` → `{suggestions, ...startBuild()}`.
  3. Else `ask()`.
- The server is stateless: the client echoes `flow` back on every turn.

### 10.10 Dashboard-building flow: `assistantDashboard.js` + `assistantWidgets.js`

#### 10.10.1 Intent and command detection
- **`isBuildIntent`:** the text must contain `dashboard` or `dash board`, AND match `\b(build|create|make|set ?up|design|generate|new|assemble|put together|can you|could you|help me)\b`, or `/dashboard (for me|please|for us)/`.
- **`isCancel`:** `\b(cancel|never ?mind|nevermind|forget it|abort|quit|exit)\b` (case-insensitive). Checked first at every step. Reply: "No problem — I cancelled the dashboard. Ask me anything else about your marketing." with `flow {active:false}`.
- **`isDone`:** the trimmed message, with trailing `.`/`!` removed, matches either
  - `^(done|finish(ed)?|that'?s it|looks good|go ahead|ship it|proceed|let'?s go|all done|good to go)$`, or
  - `^(ok(ay)?[, ]*)?(yes[, ]*)?(build|create|make|save|generate)( it| them| the dashboard| this| now)?$`.

  Both are anchored, so "create a bar chart" is not treated as done.
- **`datasetLoaded()`:** `getMeta()?.loaded`.

#### 10.10.2 Steps
The `flow` object is `{active, step, draft:{name, focus, widgets[], suggestions[]}}`. Option chips are `{label, value}`; the client displays the label and sends the value.

**1. `startBuild()`**
- Reads the existing dashboard names. Reply: "Love it — I'll build a dashboard for you.{existingNote} First, what should we call it?"
- existingNote: " You already have N dashboard(s) (first 4 names joined, … if more than 4). Names must be unique, so I'll steer clear of those."
- Options: `suggestName('Campaign Performance')`, `suggestName('Marketing Overview')`, Cancel. `suggestName` returns the base name, or "base 2", "base 3"… when taken (case-insensitive, trimmed).
- `flow {active:true, step:'name', draft:{}}`.

**2. step `name`**
- `cleanName` strips "please / let's call/name it", "the name is / name: / title: / it's" and surrounding quotes, then trims. Fewer than 2 characters → null → "My Dashboard". At most 80 characters.
- If the name exists: "A dashboard named "X" already exists — names have to be unique. I can use **"X 2"** instead, or type another name." Options `Use "X 2"` (value is the suggestion) and Cancel; the step stays `name`.
- Otherwise: "Great — "{name}". What should it focus on? I'd recommend **your uploaded campaign dataset** | **a marketing overview**."
- Focus options:
  - "My uploaded campaign dataset  (recommended)" (value `dataset`, only when a dataset is loaded)
  - "Marketing overview" (`marketing`)
  - "Budget & spend" (`budget`)
  - "Goals & pipeline" (`goals`)
  - Cancel

**3. step `focus`**
- `matchFocus` regexes:
  - `/dataset|upload|campaign data|my data|the data/` → dataset if loaded, else marketing
  - `/budget|spend|cost/` → budget
  - `/goal|pipeline|target|attainment/` → goals
  - `/marketing|overview|revenue|roi|channel/` → marketing
  - Otherwise default to dataset if loaded, else marketing.
- Seeds `draft.widgets = blueprint(focus,'recommended')`. Reply: "Here's a {FOCUS_LABEL} starter set. Keep it, swap widgets, add your own, or take AI suggestions — build exactly what you want." followed by the widget list (`showWidgets`).
- FOCUS_LABEL: campaign-dataset / marketing overview / budget & spend / goals & pipeline.
- DEFAULT_NAME: Campaign Performance / Marketing Overview / Budget & Spend / Goals & Pipeline.

**4. step `widgets`**
- `showWidgets` reply: "{lead}\n\n**{name}** — N widget(s):\n1. Title…\n\nType a chart to add — e.g. **add a KPI of blended ROI**, **bar of planned vs actual by channel**, or **revenue by channel** — or **remove 2**, or pick an option below. Say **build it** when you're ready."
- Options: `Build it ✓ (N)` → `build` (only if N>0), `✨ Suggest charts` → `suggest`, `Recommended`, `Compact`, `Detailed`, `Clear all` → `clear` (only if N>0), `Cancel`.
- Commands, checked in this order:
  1. `pickall` / `pick:<i>` (with draft.suggestions): add the picked suggestion(s), de-duplicated by `widgetKey = type|source|field|fields-joined`, and remove them from the suggestion list.
     - If suggestions remain → `showSuggestions` with lead "Added N. Add more, or **build it**." or "That one is already in — pick another.".
     - Otherwise → `showWidgets` with "Added N chart(s). Anything else?" or "Those are all added already.".
  2. `\b(suggest|suggestions?|recommend|ideas?|options)\b` → `suggestWidgets()` minus existing.
     - None left: "I don't have new suggestions beyond what's here — add your own, e.g. "gauge of budget utilization" or "revenue by channel"."
     - Otherwise `showSuggestions(lead 'Here are charts I recommend. Tap to add:')`: a numbered list plus "Current dashboard has N widget(s)."; options `＋ {title}` → `pick:i`, `Add all` → `pickall` (only if more than 1), Build it (if N>0), Cancel.
  3. `^(recommended|use recommended|default|reset)$` → recommended blueprint, "Reset to the recommended set."
     - `^(compact|smaller|minimal|simple)$` → "Switched to a compact layout."
     - `^(detailed|more detail|comprehensive|everything)$` → "Switched to a detailed layout."
  4. `^(clear|clear all|start empty|empty|remove all|delete all|start over)$` → [] with "Cleared — now add exactly the charts you want."
  5. `\b(?:remove|delete|drop|take out)\b\s*#?(\d+)` → remove the item by 1-based index: "Removed "{title}"." or "There's no widget #N. Pick 1–M."
  6. `isDone`:
     - With 0 widgets: "Add at least one chart first — try "add a KPI of blended ROI" or tap **Suggest charts**."
     - Otherwise: name = draft.name || DEFAULT_NAME; if taken, `suggestName`. Then `createDashboard({name, widgets})`, persisted to `dashboards.json` and given an id `dash-<ts36>-<rand5>`.
     - Reply: "✅ Done! I built **"{name}"** with N widgets:\n1. …\n\nIt's saved in the Builder — open it to explore, recolor, or keep editing."
     - Returns `created {id,name}`, `link {label:'Open my dashboard', route:'/builder?dash={id}'}` and `flow {active:false}`. A creation error gives "I hit a snag creating the dashboard: {msg}. Want to try again?"
  7. Anything else → `parseWidgetRequest(raw)`:
     - error → "I couldn't add that: {error}"
     - duplicate → ""{title}" is already on the dashboard."
     - otherwise push → "Added **{title}**."
- An unknown or stale step restarts with `startBuild()`.

#### 10.10.3 Blueprints
Helpers:
- `K(src,field,title)` → kpi with w 1
- `S(type,src,fields,title,w)` → series chart
- `D` → donut with w 1
- `T` → table with w 3

**dataset** (compact = first 5)
- base:
  - KPIs on cmpKpis: campaigns "Campaigns", avgRoi "Average ROI", avgConversionRate "Avg conversion %", ctr "Overall CTR %"
  - line cmpMonthly [clicks, impressions] "Monthly clicks & impressions" (w2)
  - donut cmpChannel count "Campaigns by channel"
  - bar cmpType [roi, conversionRate] "ROI & conversion by campaign type" (w3)
- extra: donut cmpSegment count "Campaigns by segment"; bar cmpChannel [clicks, impressions] "Clicks & impressions by channel" (w2); table cmpCompanies "Top companies"

**budget** (compact 3)
- base: KPIs on overview totalSpent "Marketing spend", budgetUtilization "Budget utilization", totalPipeline "Pipeline"; bar budgetChannel [planned, actual] "Planned vs. actual by channel" (w3)
- extra: bar budgetRegion [planned, actual] "Planned vs. actual by region" (w3); table campaigns "Campaigns"

**goals** (compact 4)
- base: KPIs totalPipeline "Pipeline", totalLeads "Total leads", blendedRoi "Blended ROI"; table goals "Goals & attainment"; area timeseries [leads] "Leads trend" (w2)
- extra: donut channels revenue "Revenue by channel"

**marketing** (default; compact 4)
- base: KPIs totalRevenue "Attributed revenue", blendedRoi "Blended ROI", totalLeads "Total leads"; area timeseries [revenue, spend] "Revenue vs. spend" (w2); donut channels revenue "Revenue by channel"
- extra: bar budgetChannel [planned, actual] "Planned vs. actual by channel" (w3); table campaigns "Campaigns"

Variants: `recommended` = base; `compact` = base.slice(0, compactCount); `detailed` = base + extra. Each widget is stripped to `{type, source, title, w, field?, fields?}`.

#### 10.10.4 `suggestWidgets()`
Filtered so every field exists in the catalog:
- kpi overview totalRevenue "Attributed revenue"
- kpi blendedRoi "Blended ROI"
- gauge overview budgetUtilization "Budget utilization" with `threshold {value:90, direction:'below'}`
- area timeseries [revenue, spend] "Revenue vs. spend" (w2)
- donut channels revenue "Revenue by channel"
- bar budgetChannel [planned, actual] "Planned vs. actual by channel" (w3)
- funnel channels revenue "Revenue funnel by channel" (w2)

#### 10.10.5 `parseWidgetRequest(message)`: natural language → valid catalog widget
- `norm` = lower-case, non-alphanumerics → spaces, trimmed. `mentions(hay, phrase)` = whole-word containment, phrase length ≥3.
- **Type words** (first match wins):

  | type | matched words |
  |---|---|
  | gauge | gauge(s), dial(s), speedometer |
  | funnel | funnel(s) |
  | donut | donut, doughnut, pie, share, breakdown, split by, split |
  | area | area |
  | line | line, trend, over time, time series, timeseries |
  | bar | bar, column, compare, comparison |
  | table | table, grid, rows, records, list of |
  | kpi | kpi(s), tile(s), metric, headline, scorecard, single number, big number |

- **Widths:** kpi 1, gauge 1, donut 1, funnel 2, line 2, area 2, bar 2, table 3.
- **Field hits:** every catalog source field whose name or id is mentioned, sorted by match length desc (stable, so catalog order breaks ties). **Source hits:** sources whose name or `x` dimension is mentioned.
- **Threshold:** `\b(?:target|goal|threshold)\s*(?:of|is|at|=|:)?\s*(-?\d[\d.,]*)\s*(k|m|b|t)?\b` → value × suffix. `direction = 'below'` if the text contains below / under / less than / at most / no more than / keep under / <= / ≤, else 'above'.
- **Inferring a type when none is named:**
  - "over time"/"trend" with a series field → line
  - breakdown field plus by/share/split → donut
  - single-kind field → kpi
  - series field → bar
  - a source hit → table, donut, bar or kpi depending on the source's kinds
  - otherwise error: "I couldn't find that in your data. Try a metric like {first 4 single metrics}, or say e.g. "revenue by channel"."
- **By kind** (from the catalog widget type):
  - **single** (kpi/gauge): source = the first single field hit's source, else a single-kind source hit, else `overview`. Field = the hit, else the source's first field. title = field name; threshold = target.
  - **breakdown** (donut/funnel): first breakdown field hit, else breakdown source hit, else the first breakdown source. title = source name.
  - **series** (line/area/bar): source = the first series field hit's source. Fields = all hit fields of that source, else its first field. title = source name. Threshold only if there is one field.
  - **table**: a table-kind source hit, else the first table source; title = source name.
- **Quirk:** `campFlex` ("Campaigns — custom axes (pick X & Y)") is the first catalog source and owns Revenue / Pipeline / Spend / Budget / Leads with kinds series and breakdown. So "revenue over time", "donut of revenue by channel", "funnel of revenue by channel" and "leads trend" all resolve to campFlex (default X = channel) with that long title, not to `timeseries`/`channels`. Words like "foo bar" match `bar` and fall back to campFlex revenue.
- **Verified examples:**
  - "add a KPI of blended ROI" → kpi overview blendedRoi
  - "gauge of budget utilization with target 90%" → gauge budgetUtilization threshold 90 above
  - "pipeline" → kpi totalPipeline
  - "campaigns table" → table campaigns
  - "bar of planned vs actual by channel" → bar budgetChannel [planned, actual] "Budget by channel"
  - "clicks by channel" (dataset loaded) → donut cmpFlex clicks

#### 10.10.6 Builder catalog the wizard depends on (builder.js)
- **Widget types:**
  - kpi "KPI tile" (single), gauge "Gauge" (single)
  - line "Line chart", area "Area chart", bar "Bar chart" (all series)
  - donut "Donut chart", funnel "Funnel" (both breakdown)
  - table "Table" (table)
- **Static sources:**
  - campFlex (dims channel/region/status → campByChannel/Region/Status; measures revenue, pipeline, spent "Spend", budget, leads)
  - overview: totalRevenue "Attributed revenue", totalSpent "Marketing spend", totalPipeline "Pipeline", totalLeads "Total leads", blendedRoi "Blended ROI", budgetUtilization "Budget utilization", activeCampaigns "Active campaigns"
  - timeseries "Daily performance" (x date; revenue, spend, leads, sessions)
  - channels "Revenue by channel" (breakdown + series; revenue)
  - budgetChannel / budgetRegion (x category; planned, actual)
  - campaigns table (name, channel, status, spent, leads, pipeline, roi)
  - goals table (metric "Goal", actual, target, attainmentPct, status)
  - dataset sources (§10.2.6) when loaded
- **Data bundle keys:** `overview, timeseries(30), channels, budgetChannel, budgetRegion, campaigns, goals, campByChannel, campByRegion, campByStatus` plus the dataset keys.
- Dashboard names are unique (case-insensitive) and at most 80 characters. Widgets are normalised to `{id, type, title, source, field, fields, x, threshold, w∈{1,2,3}}`. A seed dashboard "Executive summary" is created when the store is empty.

### 10.11 UI

#### 10.11.1 Shared building blocks
- **Palette** `SERIES`: `#2a78d6` blue, `#1baf7a` aqua, `#e87ba4` magenta, `#eda100` yellow, `#4a3aa7` violet, `#eb6834` orange, `#008300` green, `#e34948` red.
- **`STATUS`:** good `#0ca30c`, warning `#fab219`, serious `#ec835a`, critical `#d03b3b`. Pages use `#c77700` for "warning / at-risk" text.
- **ECharts themes** `zr-light`/`zr-dark`:
  - Colours from SERIES, transparent background, system font.
  - Axis label `#6e6e73` (light) / `#9a9aa0` (dark), size 11, no axis line or tick; value-axis split lines `rgba(0,0,0,.07)` / `rgba(255,255,255,.08)`.
  - Tooltip: white 97% / `rgba(28,28,30,.96)`, 14 px radius, blur, shadow.
- **`EChart`:** re-inits on a theme flip, uses a ResizeObserver, merges option updates, takes a `height` prop.
- **Chart helpers:**
  - `areaGradient(color, top=.28, bottom=0)`: vertical gradient fill.
  - `seriesTooltip(fmt)`: bold header (axis label) plus one row per series: marker, name, bold formatted value.
- **Formatters** (USD base, converted to the active currency):
  - `fmtMoney`: ≥1M `$x.xxM`, ≥1K `$x.xK`, else `$n`.
  - `fmtMoneyFull`: `Intl` currency, 0 decimals.
  - `fmtNum`: ≥1M `x.xxM`, ≥1K `x.xK`.
  - `fmtNumFull`: en-US grouping.
  - `fmtPct`: `x.x%`.
  - `fmtDuration`: under 60 → `Ns`; otherwise `Xm Ys` or `Xm`.
- **Currency:** the `CurrencyControl` select (label "Currency", options like "$ USD") sits in every `PageHeader`. Codes are USD, EUR, GBP, CAD, AUD, INR, JPY, SGD. Live FX rates come from `/api/currency`; the choice persists in `localStorage zr_currency`.
- **`PageHeader`:** blue uppercase 12 px eyebrow; h1 22/26 px bold `{title}` + `{highlight}` in ink-2 semibold; 13.5 px subtitle; right side shows actions plus the currency picker.
- **`SectionHeading`** (marketing-style, Home only): blue eyebrow, 32/44 px headline with the highlight in `.grad-text` (gradient `#2a78d6 → #9f4bd6 → #eb6834 → #eda100`), 18/20 px subtitle; center or left aligned.
- **`Panel`:** card with a header row (15 px semibold title, 12.5 px subtitle, optional right `action`), hairline divider, body padding 20 px.
- **`StateWrap`:** spinner (28 px) centred in `minHeight` (default 240); error Alert "Could not load data: {msg}. Is the API running on port 4000?".
- **`InfoTip`:** ⓘ `InfoOutlined` icon (size 14 default), MUI Tooltip with arrow, dark bubble, max width 260, opens immediately on touch and stays 4 s; hover turns it blue.
- **`ScrollReveal`:** IntersectionObserver fade-up (y capped at 8 px, 0.4 s, stagger ≤0.04 s); honours reduced motion.
- **Card classes:**
  - `.card`: surface, radius token, border, shadow.
  - `.card-hover`: lifts 3 px, blue-tinted border and a cursor-following radial "spotlight" driven by `--mx/--my`, which a delegated pointer listener in App.jsx sets.
- **Buttons:** `.btn-pill` (10 px radius, 8×16 padding, 13.5 px); `.btn-blue` (blue gradient + glow); `.btn-ghost` (4.5% black).

#### 10.11.2 `KpiCard`
- Props: `label, value, format, delta, deltaGood=true, icon, accent='#0071e3', size 'md'|'lg', tip`.
- Layout: a header row with a 13 px secondary label, optional InfoTip, and an icon chip on the right (background `${accent}14`; lg = 36 px circle, md = 28 px rounded square).
- Value: `AnimatedNumber`, 38 px (lg) / 27 px (md), bold.
- Optional delta row: an up or down arrow and `|delta|%`, green `#0a8f0a` if good else `#c0392b`, followed by muted "vs last period". "Good" means up when `deltaGood`, else down.
- lg adds `card-hover` and p-6 (md is p-4).

#### 10.11.3 `AnimatedNumber`
- GSAP tweens `{n: prev} → value` over 1.2 s with `power2.out`, writing `format(n)` to `textContent` each frame.
- Uses `useLayoutEffect` and paints `format(prev)` immediately, so there is no flash. It re-runs when the currency changes. Initial render is `format(0)`.

#### 10.11.4 `Hero` (signed-out landing only)
- Black section, `min-height: min(88vh,760px)`.
- Background:
  - `.aurora` layer at `inset:-15%`, opacity .8: four radial gradients (blue .55, purple .5, orange .45, aqua .4) with `blur(20px)`.
  - A vignette `radial-gradient(120% 80% at 50% 0%, transparent 40%, rgba(0,0,0,.6))`.
- Content, centred: eyebrow "ZenReach" (white/70); h1 46/84 px "Marketing performance," newline "**beautifully clear.**" (`.grad-text`).
- Paragraph (19/23 px, white/70): "Every plan, every dollar, every result — in one breathtakingly simple view. Campaigns, budgets and goals. Live Google Analytics. No spreadsheets."
- "Scroll to explore": uppercase, 0.2em tracking, white/40, pulsing.
- GSAP:
  - Entrance timeline: `[data-hero]` from y 34 / opacity 0, 1 s, stagger .12; then `[data-hero-cta]` from y 16, 0.7 s, stagger .08, overlapping by 0.4 s.
  - Endless aurora drift: scale 1.15, rotate 8°, 14 s, yoyo, sine.inOut.
  - Mouse parallax via `gsap.quickTo` on xPercent/yPercent = (pointer/viewport − .5)×24, 1.2 s power2.out.
  - Both continuous animations pause while the `zr:auth-modal` event says the modal is open.
  - Reduced motion: no continuous animation and zero-duration entrance.

#### 10.11.5 `Dashboard.jsx` (route `/`)
Routing logic:
- `!ready` → blank `min-h-[50vh]`.
- Signed in with `orgRole ∈ {manager, lead, member}` → `<TeamDashboard/>`.
- Other signed-in users (admins/owners, no org placement) → `<Dashboards/>`, the saved-dashboards viewer.
- Signed out → `<Hero/>` + `<OverviewMarketing/>`.

`OverviewDashboard` (the fixed KPI and AI overview) is still in the file but is not rendered anywhere. It is the only consumer of `/api/insights/ai`, so a rebuild must decide where AI analysis lives. Its spec, for reuse:

1. **KPI section:** SectionHeading eyebrow "At a glance", title "The whole picture." highlight "One screen.", subtitle "Revenue, spend, return and pipeline — updated continuously and rendered with clarity."
   - Then a right-aligned CurrencyControl, then a 1/2/4-column grid of lg KpiCards. Deltas are hard-coded:
     - "Attributed revenue" (fmtMoney, +12.4%, TrendingUp icon, SERIES[0])
     - "Marketing spend" (+4.1%, `deltaGood=false`, so it shows red, Paid icon, SERIES[3])
     - "Blended ROI" (`n.toFixed(2)+'x'`, +8.6%, Savings icon, SERIES[1])
     - "Total leads" (fmtNum, +6.2%, Groups icon, SERIES[4])
2. **AI analysis:** eyebrow "AI analysis", "Not just numbers —" highlight "what to do about them.", subtitle "ZenReach reads every signal in your data and returns ranked, quantified recommendations — each with the action to take and its expected impact."
   - A pill "✦ Analyzed by {provider}" (AutoAwesome icon in `#9f4bd6`).
   - A 2-column grid of cards, one per insight:
     - severity-tinted 32 px circle icon (good `#0ca30c`, warning `#c77700`, serious `#ec835a`, critical `#d03b3b`) beside a 16.5 px bold title;
     - an impact pill tinted with the severity colour;
     - 14 px detail text;
     - a divider, then a purple "→" with the action in 13.5 px semibold.
3. **Performance** (grey band): eyebrow "Performance", "Watch return" highlight "outpace spend." (left-aligned).
   - Panel "Revenue vs. spend" / "Last 30 days — both in USD" (2 columns wide), 320 px. Two smooth area lines, Revenue in SERIES[0] (area .28) and Spend in SERIES[3] (area .24), width 2.5, no symbols.
     - Legend circles at top; x axis = dates shown as `MM-DD` (slice 5), boundaryGap false; y axis fmtMoney; axis tooltip `seriesTooltip(fmtMoneyFull)`.
   - Panel "Revenue by channel" / "Attributed revenue by source": a 220 px donut (radius 58–86%, no labels), item tooltip "{name}<br/><b>{full $}</b> · {percent}%".
     - Below it, a legend list: coloured dot, name, `fmtMoney` value.
4. **Discipline:** "Plan and actuals," highlight "reconciled."
   - Panel "Planned vs. actual spend" / "By channel — USD": 300 px grouped bars, Planned SERIES[4] and Actual SERIES[0], rounded tops, max bar width 26, x labels rotated 24°, shadow axis tooltip.
   - Panel "Daily leads" / "Last 30 days": a 300 px smooth line in SERIES[2], tooltip "{date}<br/><b>{n}</b> leads".
     - Below it, a tinted strip "Budget utilization … {fmtPct}".
5. **CTA band:** "Ready when you are." / "Rebalance spend in the optimizer, or connect your live Google Analytics in a minute." Buttons "Try the optimizer" (→ /optimizer, blue) and "Connect a source ›" (→ /integrations, ghost).

`OverviewMarketing` (signed out):
- Features section: eyebrow "Why ZenReach", "Everything marketing needs," highlight "beautifully simple.", subtitle "A lighter, faster, self-serve alternative to heavyweight marketing-performance tools — with an AI analyst built in."
- Six feature cards in a 3-column grid, each with a 44 px gradient icon tile (`#2a78d6 → #9f4bd6`):
  - AI analyst: "Ranked, quantified recommendations — and a chat that can build a dashboard for you."
  - Attribution & forecasting: "Nine multi-touch models, a waterfall funnel and a predictive yearly forecast."
  - Planning: "A true Plan → Campaign → Program → Tactic hierarchy, seen five ways."
  - Budget optimizer: "Rebalance spend with what-if sliders and an AI-recommended mix."
  - No-code Builder: "Assemble your own dashboards from KPIs, charts and tables — no code."
  - Calendar sync: "Push scheduled campaigns to your Google Calendar — on your phone in seconds."
- "A quick look": "See it" highlight "in action.". Three screenshots from `/marketing/ai-analysis.png`, `/marketing/planning.png`, `/marketing/dashboard.png`, shown at 16:10 with object-top, with captions "AI reads your data and tells you what to do next." / "Run marketing like a practice — plan to tactic." / "Build any dashboard, or just ask the AI to."
- Sign-up CTA: "Create your free account." / "Sign up to see your live dashboard, AI analysis, planning and more — set up in minutes." Buttons "Get started free" and "Log in ›" dispatch `CustomEvent('zr:open-auth',{detail:{tab:'signup'|'login'}})`.

#### 10.11.6 Analytics page (`/analytics`)
- **Header:** eyebrow "Google Analytics", "Your traffic," highlight "in living color.", subtitle "Sessions, users and conversions from GA4 — demo out of the box, live in a minute."
- **Data:** `api.googleAnalytics()` once. `api.realtime()` on mount and every 15 s via setInterval, cleared on unmount; errors are swallowed.
- **Status alert** (outlined, radius 4):
  - live → success with a check icon, "Connected live to GA4 property **{propertyId}**."
  - otherwise info: "Showing **demo** Google Analytics data. Add `GA_PROPERTY_ID` and service-account credentials in `server/.env` to connect your real GA4 property." followed by ` ({error})` when present.
- **Realtime tile** (white card, row layout on sm+):
  - A pinging green dot (`#0a8f0a`), "Active users right now ⓘ(activeUsersNow)", and sub-line "Live from GA4 Realtime[ · demo] · updates every 15s".
  - On the right, a 40 px extra-bold number (`rt.activeUsers` or "—") and "{views} views".
- **Empty-live alert:** shown if `data.live && sessions===0`. Text: "Your live connection is active and Realtime is receiving traffic. GA4 is still **processing** today's hits — the 28-day totals and charts below fill in a few hours after data is collected (Google's standard-report delay), so they may read 0 for now." It is effectively unreachable because the server substitutes demo data in that case.
- **KPI row** (md KpiCards, no delta):
  - "Sessions (28d)" ⓘsessions (fmtNum, Visibility icon, SERIES[0])
  - "Users" ⓘusers (SERIES[1])
  - "Conversions" ⓘconversions (SERIES[2])
  - "Avg. engagement" ⓘengagement (fmtDuration, Timer icon, SERIES[4])
- **Row 2:**
  - Panel "Sessions & users" / "Last 28 days" (2 columns wide), 320 px. Smooth lines: Sessions SERIES[0] with area gradient, Users SERIES[1] without area. Legend; x axis MM-DD; y axis fmtNum; tooltip `seriesTooltip(fmtNumFull)`.
  - Panel "Traffic by channel ⓘ" / "Session share": 320 px horizontal bars (y category inverse = channel names, x value fmtNum), each bar coloured SERIES[i], radius right 6, max width 22. Tooltip "{channel}<br/><b>{n}</b> sessions".
- **Row 3** (only if topCountries is non-empty):
  - Panel "Sessions by country ⓘ" / "Top countries by sessions · last 28 days" (2 columns wide). The tip reads "Where your visitors are, from GA4's geography report (derived from IP). The same breakdown GA shows on its 'Users by country' map."
    - 340 px horizontal bars; y labels are "{flag emoji}  {country}", where the flag is built from regional-indicator code points `0x1F1E6 + letter`, with 🌐 as the fallback.
    - Tooltip "{flag} <b>{country}</b><br/>{sessions} sessions · {users} users · {share}%".
  - Panel "Top regions ⓘ" / "By sessions". Tip: "The regions / states within countries sending the most sessions."
    - A list of rows: region name with sessions on the right; below, the country and a 96 px mini bar with width `max(6%, sessions/max*100)` coloured SERIES[i].
    - Empty state: "No region data yet."
- **Panel "Top pages ⓘ"** / "Most-viewed pages in the period": a ranked list with a 28 px numbered blue circle, the path, and "{n} views".

#### 10.11.7 Integrations page (`/integrations`)
- **Header:** eyebrow "Integrations", "Everything," highlight "already flowing in.", subtitle "Google Analytics is live; the rest come pre-connected with sample data. Import your own via CSV or Excel."
- **Panel 1 "Import from a file"** / "Bring an existing plan in from a spreadsheet — CSV or Excel (.xlsx / .xls)". This is the tactics importer, not the campaign dataset.
  - Contained button "Upload file" / "Importing…" (hidden file input with the shared accept list).
  - "Download CSV template": GETs `/api/import/sample` and downloads `zenreach-tactics-template.csv`.
  - `canEdit = !user || role !== 'Viewer'`; when false, shows "Sign in as an Editor to import."
  - On success: toast "Imported N tactics. (k skipped)", plus a status strip "**N** tactics imported from {file} · k skipped" with buttons "Undo import" (outlined red; calls bulk-delete of the created ids, toast "Removed N imported tactics.") and "Manage in Planning".
  - Tip line: "Tip: you can also select and delete tactics anytime in Planning → Grid (Delete selected / Delete all)."
  - Server side: headers title/name/tactic (required), type, owner, quarter/qtr, status, priority, budget/cost/amount, channel, goal/objective.
- **Panel 2 "Google Calendar"** / "Push your scheduled tactics into your Google Calendar — they appear on your phone in seconds (one-way, live)." contains `<GoogleCalendarSync onChange>` (out of scope here). Its status comes from `/api/integrations/google-calendar/status`.
- **Card grid** (1/2/3 columns), one card per catalog entry:
  - Top row: 48 px icon tile and a status chip. GA uses the Analytics icon on the brand gradient; Salesforce Cloud SERIES[0]; SAP AccountBalance SERIES[5]; Google Ads AdsClick SERIES[1]; QuickBooks ReceiptLong SERIES[4]; all others Cloud SERIES[0], tinted.
  - Chip: `connected` → "Connected" green with a check icon; `demo` → "Demo data" `#c77700`. Google Calendar instead shows "Connected" (green, check) or "Not connected" (`#86868b`).
  - Name (20 px). Meta line: category · data label · sync label · method badge.
    - Data label: real+connected "Live data" (green), real+demo "Demo data", otherwise "Sample data" (muted); for GCal "Your calendar" / "Not connected".
    - Sync label: Two-way sync (SyncAlt icon) / Inbound (South) / Outbound (North), with ⓘsyncDirection.
    - Method badge with ⓘconnectMethod.
  - Description (15 px).
  - Full-width button "{cta} ›":
    - blue for GA and an unconnected GCal, ghost otherwise;
    - navigates to `route`;
    - for GCal it reads "Manage calendar ›" / "Connect ›" and smooth-scrolls to the Calendar panel.
- **Snackbar toast:** bottom center, 4 s, filled Alert.

#### 10.11.8 Campaigns page (`/campaigns`)
- **Header:** eyebrow "Campaigns", "Every campaign," highlight "in focus.", subtitle "Plan, spend, pipeline and return — tracked per campaign, filtered in a tap."
- **Four plain stat cards** (2/4 columns, card-hover, 13 px label, 26/30 px value):
  - "Total campaigns" (8)
  - "Active now" (6)
  - "Average ROI ⓘroi": simple mean of campaign ROIs, 2dp + x → "4.52x"
  - "Total pipeline ⓘpipeline": fmtMoney → "$9.68M"
- **Panel "All campaigns"** / "Plan → spend → results" (flush body). Action is an exclusive pill ToggleButtonGroup: All / Active / Planned / Completed (selected = `#0071e3` with white text).
- **Table columns:**
  - Campaign (bold)
  - Channel and Region (ink-2)
  - Status: Chip tinted Active `#0ca30c`, Planned `#0071e3`, Completed `#86868b`
  - "Budget used ⓘutilization": a 6 px LinearProgress with `pct = min(100, round(spent/budget*100))`, red (`STATUS.critical`) when >90 else blue, plus "{pct}%"
  - Leads ⓘ (right-aligned, fmtNumFull)
  - Pipeline ⓘ (fmtMoneyFull)
  - ROI ⓘ (bold `{roi.toFixed(2)}x`, green `#0a8f0a` if ≥2)
  - Rows highlight on hover. There is no sorting, search or pagination.

#### 10.11.9 Goals page (`/goals`)
- **Header:** eyebrow "Goals & forecast", "Are we going to" highlight "make the number?", subtitle "Targets, attainment and quarter-end forecast."
- **Centred meta row:** "Period" Q3 2026 · "Elapsed ⓘ" 62% · "Days remaining" 35 (22 px bold values).
- **One Panel per goal** in a 2-column grid:
  - Metric name (16 px) and "Target ⓘ {fmtVal(target)}".
  - Status pill on the right: On track (CheckCircle, green), At risk (Warning, `#c77700`) or Off track (Error, red).
  - A 34 px extra-bold actual value and "{attainmentPct}% of target so far ⓘattainment".
  - A 12 px progress track: fill width `min(100, attainment)%` with gradient `#2a78d6 → #9f4bd6`, plus a 2 px vertical forecast marker at `left: min(min(120, forecastPct)%, 100%)` in the status colour (title "Forecast").
  - Footer: "Now: {att}%" and, in the status colour, "Forecast to close: {forecastPct}% of target ⓘgoalForecast".
- **`fmtVal`** hard-codes `$` and does not convert currency:
  - `$`: ≥1M `$x.xxM`, else `$xK`
  - `x`: 2dp + x
  - `#`: ≥1000 `x.xK`

#### 10.11.10 Optimizer page (`/optimizer`)
- **Header:** eyebrow "Budget optimizer", "Spend less." highlight "Earn more.", subtitle "Model budget shifts across channels, then apply the AI-recommended mix."
- **State:** `alloc` = {channel: spend}, initialised from `data.channels`. All maths is client-side:
  - `spend = Σalloc`
  - `projected = Σ alloc×efficiency`
  - `baseline = data.baseline.projectedRevenue`
  - `lift = (projected−baseline)/baseline×100`
  - Total budget is not conserved: sliders can raise or lower the total, and lift reflects that.
- **Left column (3 of 5)**, Panel "Allocate your budget" / "Drag to rebalance — projections update live":
  - Per channel: a coloured dot, the name, a pill "{eff 2dp}x efficiency ⓘefficiency", and the value in fmtMoney.
  - MUI Slider: min 0, max `round(spend×2)`, step 1000, colour SERIES[i], 18 px thumb, rail opacity .2.
  - Buttons: "✦ Apply AI recommendation" (blue; sets alloc to `recommended.allocations`) and "↺ Reset to current" (ghost).
- **Right column (2 of 5):**
  - Dark card (`linear-gradient(160deg,#0b1020,#141a2e)`): "Projected attributed revenue ⓘprojectedRevenue", a 44 px white fmtMoney value.
    - Lift pill: green (`rgba(12,163,12,.2)` / `#4ade80`) with ▲, or red (`#f87171`) with ▼; "{|lift| 1dp}% vs current plan ⓘlift".
    - Footer "Total budget {fmtMoneyFull} · baseline ⓘ {fmtMoney}".
  - Panel "Projected revenue by channel" / "At the current allocation": a 300 px horizontal bar chart of `round(alloc×eff)` per channel.
    - Bar width 18, radius right 6, SERIES colours.
    - x axis uses `compactMoney` (`$0 · $600K · $1.2M`) with splitNumber 4 and dashed split lines.
    - Item-trigger tooltip "{channel}<br/><b>{full $}</b> projected"; emphasis shadow.
- **Demo:** baseline $4.76M; after Apply, $5.07M with ▲ 6.7%.

#### 10.11.11 Attribution page (`/attribution`)
- **Header:** eyebrow "Attribution & forecasting", "Credit where" highlight "it's earned.", subtitle "Waterfall funnel, nine attribution models, reallocation ranking and a predictive forecast."
- **Loading:** models, funnel and forecast load on mount behind a StateWrap. Attribution and reallocation refetch whenever `model` changes (default `position`), with errors swallowed.
- **Info alert:** "Attribution runs over a sample set of cross-channel customer journeys, so it works with no external data. Connect a real touchpoint feed to make it live."
- **Panel "Multi-touch attribution ⓘmultiTouch"** / "Revenue credited to each channel":
  - Action: a small "Model" Select (min width 200) listing the 9 model names.
  - Body: the model `desc` (13 px), then a 300 px horizontal bar chart of credited revenue sorted desc (SERIES colours, bar max 24), tooltip "{channel}<br/><b>{$}</b> credited".
  - Footer "{conversions} conversions · {$ total} total credited revenue".
- **Two-column row:**
  - Panel "Waterfall funnel ⓘwaterfallFunnel" / "Volume and step conversion through the funnel": for each stage, a label row (stage name; volume plus "(x%)" for later stages), then a 24 px bar on a chip track.
    - Fill width `max(volume/impressions×100, 6)%`, min 40 px, coloured SERIES[i], white 10.5 px text inside: "100%" for the first stage, then the step `conversionPct%`.
  - Panel "Predictive yearly forecast ⓘforecast" / "Projected revenue vs. target":
    - "Projected annual" with a 32 px fmtMoney value ($8.81M).
    - Status pill "{status with hyphen → space} · {attainment}%" (e.g. "at risk · 90.9%"), tinted by status.
    - A 200 px smooth area line over Q1–Q4, tooltip "{Qn}<br/><b>{$}</b> revenue".
    - Footnote "Assumes 8% quarter-over-quarter growth · stretch target ⓘ {fmtMoney(target)}".
- **Panel "Reallocation ranking ⓘreallocation"** / "Channel efficiency (credited revenue ÷ spend)":
  - Recommendation box (grey): "Recommendation: move {$full} from **{from}** → **{to}**" plus the rationale.
  - Table (min width 520): Channel | Spend (`—` if 0) | "Credited revenue ⓘ" | "ROI ⓘ" (`{roi}×` or `—`). ROI colour: null → muted, ≥2 → green, ≥1 → ink, below 1 → red.

#### 10.11.12 AiAssistant ("Ask ZenReach")
- **Launcher:** fixed bottom-6/right-6, 56 px gradient circle (`#2a78d6 → #9f4bd6`, blue glow), tooltip "Ask ZenReach" placed left, aria "Open AI assistant". Icon is `ChatMark` (white speech bubble with a blue rising-trend line and dot), swapping to a Close icon when open.
- **Panel:** fixed bottom-24/right-6, width `min(92vw,392px)`, height `min(72vh,620px)`, rounded-3xl, big shadow. Opens with GSAP from y 24 / opacity 0 / scale .96 to rest, 0.4 s power3.out, origin bottom right. Dark-mode colours are hand-tuned (surface `#1c1c1e`, bubble `#2c2c2e`).
- **Header** (gradient): ChatMark avatar, "Ask ZenReach", and a sub-line "Powered by {provider}" when status says `ai`, else "Built-in analyst · demo". Status is fetched only when signed in. Close button.
- **Tabs:** Chat / Prompts (active tab blue with a 2 px underline).
- **Prompts tab:** intro "Tap a template to drop it into the chat — tweak it, then send." Tapping fills the input, switches to Chat and focuses the input after 60 ms. Groups (label → text):
  - Build a whole dashboard: "Build a dashboard from my data" → "Build me a dashboard"
  - Add a single chart (during a build):
    - KPI — one headline number → "Add a KPI of blended ROI"
    - Gauge — a value against a target → "Add a gauge of budget utilization with target 90"
    - Line — a measure over time → "Show revenue over time"
    - Bar — compare across a dimension → "Bar of planned vs actual by channel"
    - Donut — share of a whole → "Donut of revenue by channel"
    - Funnel — stages by a measure → "Funnel of revenue by channel"
    - Table — rows of records → "Table of campaigns"
  - Curate:
    - Ask for AI-suggested charts → "Suggest charts"
    - Remove a chart by number → "Remove 2"
    - Clear and start from scratch → "Clear all"
    - Finish and create it → "Build it"
  - Ask the analyst: "Which campaign has the best ROI?", "Where should I move budget?", "Are we on track to hit our goals?"
- **Chat tab:**
  - Welcome message (assistant): "Hi! I'm your ZenReach analyst. I can also **build a dashboard exactly the way you want** — say **"build a dashboard"**, then take my suggestions or describe the charts you want (e.g. **"add a gauge of budget utilization with target 90"**). Check the **Prompts** tab for ready-to-use templates."
  - Starter buttons appear only while there is just the welcome message: "Build me a dashboard", "Which campaign has the best ROI?", "Where should I move budget?".
  - Bubbles: max 85% width; user bubbles blue `#0071e3` aligned right; assistant bubbles white with a hairline border, aligned left.
  - Text rendering: `**bold**` → `<strong>`, `\n` → `<br>`; no other markdown.
  - An assistant message with `link` gets a pill button "{label} →" that navigates and closes the panel.
  - Option chips render only on the latest assistant message and only when not loading. Cancel chips are grey. Clicking a chip sends `send(label, value)`: the bubble shows the label and the API receives the value.
  - Typing indicator: three bouncing dots (`zr-typing` 1.25 s, staggered .16 s).
- **Send logic:**
  - `history` = all messages except WELCOME, as `{role, content}`.
  - Calls `api.assistant(payload, history, flow)`.
  - Enforces a minimum "typing" delay of 1.0–2.0 s (random per turn), applied to errors too.
  - Stores `flow` only when `res.flow.active`.
  - Network error: "Sorry — I could not reach the analyst service. Is the API running?", and the flow resets.
- **Input:** rounded TextField, placeholder "Ask about your marketing…"; Enter without Shift sends. Round blue send button, disabled while loading or when the input is empty.

### 10.12 Reference outputs from the demo data (for acceptance tests)

- Overview: revenue $4,755,000 · spend $1,121,000 · ROI 4.24x · pipeline $9,680,000 · leads 9,700 · 6 active · utilization 72.3%.
- Revenue by channel: Content Syndication 1.975M (41.5%) > Paid Search 1.56M > Events 560K > Paid Social 360K > Organic 300K.
- Goals: revenue on-track 102% · pipeline at-risk 94.3% · leads on-track 105.3% · ROI on-track 169.6%.
- Optimizer lift 6.7%. Attribution totals 266 conversions / $1,954,600. Position-model reallocation: $97,600 Paid Search → Email. Forecast $8,807,646 vs target $9,688,411 (90.9%, at-risk).
- GA demo: 132,174 sessions · 95,165 users · 1,399 conversions over 28 days.
- Local assistant reply to "Which campaign has the best ROI?": "Your top performer is **Nova Developer Community** on Organic / SEO, returning **7.32x** ($520,000 pipeline from $41,000 spend). It's a strong candidate for more budget." with the "View Campaigns" link.

### 10.13 Quirks and gaps to decide on in the rebuild

1. The shared PRNG makes Email/LATAM budget fallbacks, GA demo engagement and top-page views change on every request.
2. The fixed "today" (2026-07-15) and the hard-coded Q3 `ELAPSED 0.62` / `daysRemaining 35` never move.
3. Goals forecastPct is mathematically `100/targetFactor` for projected goals. The attribution forecast is always 90.9% / at-risk.
4. The W-shaped model under-credits 3-touch journeys (weights sum to 0.9).
5. Attribution's channel set and `CHANNEL_SPEND` don't match the campaign channels (no Content Syndication; Referral/Direct added; Paid Social 346K vs 78K in campaigns).
6. AI analysis uses only Azure → Anthropic, while chat uses OpenAI → Azure → Anthropic. The provider label "gpt-4o" is hard-coded. The analysis UI (OverviewDashboard) is currently unmounted.
7. `/api/insights`, `/api/optimizer/project` and `/api/attribution/compare` have no UI consumer.
8. The "Summarize…CMO" starter returns the goals answer in local mode. The NL widget parser resolves most revenue/leads requests to the `campFlex` source and titles them "Campaigns — custom axes (pick X & Y)".
9. Dataset validation is permissive (a tactics file was accepted, giving all-zero metrics). Conversion values are assumed fractional. TSV files aren't actually split on tabs. There is no endpoint to unload a dataset. The dataset only reaches the Builder and the assistant.
10. Goals `fmtVal` ignores the currency setting. Optimizer sliders don't hold the total constant. The Analytics empty-live alert is effectively unreachable.
11. There is no GA response caching; each page view runs six Data API reports, and realtime polls every 15 s.

## 11. Dashboard Builder and saved dashboards

**Source files:** `server/services/builder.js`, `client/src/pages/Builder.jsx`, `client/src/pages/Dashboards.jsx`, `client/src/components/WidgetRenderer.jsx`, `ChartThumb.jsx`, `EChart.jsx`, `client/src/lib/chart.js`.

Text in quotes is exact UI copy.

### 11.1 Architecture overview

- **Server service** `server/services/builder.js` has four jobs:
  1. `dataBundle()`: one payload containing every dataset a widget can bind to.
  2. `catalog()`: the palette of widget types, data sources and bindable fields.
  3. Dashboard CRUD, persisted to the JSON file store as `dashboards.json`.
  4. Helpers for unique names and for restore (`nameExists`, `suggestName`, `listDashboardNames`, `insertDashboard`).
- **Dataset service** `server/services/campaignDataset.js` handles uploaded external datasets. It adds extra bundle keys and catalog sources, but only once a dataset has been ingested.
- **Client pages**:
  - `client/src/pages/Builder.jsx` (route `/builder`) is the editor.
  - `client/src/pages/Dashboards.jsx` (route `/dashboards`) is the read-only viewer, plus snapshots.
- **Rendering**:
  - `client/src/components/WidgetRenderer.jsx` renders one widget from its config, its source definition and the shared data bundle.
  - `EChart.jsx` wraps ECharts.
  - `lib/chart.js` holds ECharts registration, themes and helpers.
  - `ChartThumb.jsx` draws the SVG previews in the widget picker.
- **Rendering is entirely client-side.** The client downloads the whole data bundle once and every widget reads from it. No widget makes its own query.

### 11.2 Persistence and data model

#### 11.2.1 Storage
- `server/services/store.js` is a synchronous JSON file store.
  - The data directory is `process.env.ZR_DATA_DIR` or `server/data/`. It is created on demand.
  - `readJson(file, fallback)` returns the fallback if the file is missing, empty or unparseable.
  - `writeJson(file, data)` pretty-prints with 2-space indentation.
  - `uid(prefix)` returns `${prefix}-${Date.now().toString(36)}-${random base36, 5 chars}`.
  - `nowIso()` returns an ISO timestamp.
- Dashboards live in `server/data/dashboards.json`, which holds an array of Dashboard objects.
- **Seed:** if the file is missing or not an array, `listDashboards()` writes the seed and returns it. The seed is one dashboard:
  - `id: 'dash-sample'`, `name: 'Executive summary'`, `createdAt: now`
  - Widgets:
    - `w1` kpi "Attributed revenue": source `overview`, field `totalRevenue`, w 1
    - `w2` kpi "Blended ROI": `overview.blendedRoi`, w 1
    - `w3` kpi "Total leads": `overview.totalLeads`, w 1
    - `w4` area "Revenue vs. spend": source `timeseries`, fields `['revenue','spend']`, w 2
    - `w5` donut "Revenue by channel": source `channels`, field `revenue`, w 1
    - `w6` bar "Planned vs. actual by channel": source `budgetChannel`, fields `['planned','actual']`, w 3

#### 11.2.2 Dashboard object
```
{ id: 'dash-<ts36>-<rand5>', name: string (trimmed, ≤80 chars, unique case-insensitively),
  createdAt: ISO, updatedAt?: ISO (set on every update), widgets: Widget[] }
```

#### 11.2.3 Widget object (after server normalisation)
```
{ id: string,                 // client makes 'w-<ts36>-<rand4>'; server falls back to uid('w')
  type: 'kpi'|'gauge'|'line'|'area'|'bar'|'donut'|'funnel'|'table',
  title: string ('' default),
  source: string (catalog source id),
  field?: string,             // single value (kpi/gauge) or breakdown measure (donut/funnel)
  fields?: string[],          // series measures (line/area/bar); undefined if not an array
  x?: string,                 // chosen X dimension id, only for "flexible" sources
  threshold?: { value: number, direction: 'above'|'below' },
  w: 1|2|3 }                  // width in a 3-column grid; anything else becomes 1
```
- `cleanThreshold(t)` returns undefined if `t`, `t.value` or `Number(t.value)` is not finite. Otherwise it returns `{value:Number, direction: t.direction==='below' ? 'below' : 'above'}`.
- `normalizeWidgets` turns a non-array into `[]`. It keeps only the keys listed above, so unknown keys are dropped.
- **Layout model:** there are no x/y coordinates. A dashboard is an **ordered list**. Array order is reading order, and each widget spans 1, 2 or 3 columns of a CSS grid that is 3 columns wide on large screens. Rows wrap naturally and heights come from the content. There is no free resize, only S/M/L widths.

#### 11.2.4 Name rules (server)
- `normName(s)` is `String(s||'').trim().toLowerCase()`.
- `nameExists(name, exceptId=null)` is true if any other dashboard has the same normalised name. An empty name never "exists".
- `suggestName(base)`:
  - The base is trimmed, defaults to "Dashboard", and is cut to 80 chars.
  - If the base is free, it is returned. Otherwise the first free of `"base 2"` … `"base 999"` is returned, each cut to 80.
  - The last resort is `"base <Date.now()>"`.
  - The AI assistant's dashboard builder uses this.
- `createDashboard({name, widgets})`:
  - The name is trimmed, defaults to `'Untitled dashboard'`, and is cut to 80.
  - A duplicate throws: `A dashboard named “<name>” already exists. Please choose a different name.` (curly quotes).
  - The new dashboard is pushed onto the end of the list.
- `updateDashboard(id, patch)`:
  - Returns null if the id is not found.
  - If `patch.name` is a string, it is trimmed and cut to 80. Empty throws `Dashboard name cannot be empty.` A duplicate (excluding itself) throws the same duplicate message.
  - If `patch.widgets !== undefined`, the widgets are replaced in normalised form.
  - The id is preserved and `updatedAt` is set to now.
- `deleteDashboard(id)` filters the dashboard out and returns `{id, remaining}`.
- `insertDashboard(dash)`, used for recycle-bin restore:
  - Skips if `dash` or `dash.id` is missing, or the id already exists. Returns `{added:0}`.
  - Otherwise, while a live dashboard has the **exact (case-sensitive)** same name, it appends `" (restored)"`. Then it pushes and returns `{added:1, name}`.
  - Inconsistency: restore compares names case-sensitively, but `nameExists` does not.

### 11.3 Endpoints

All routes sit under `/api`.
- **Response envelope:** success is `{ ok: true, data }`. Errors are `{ ok:false, error }` with an HTTP status.
- **Auth wall:** global middleware makes every `/api/*` route require a valid `Authorization: Bearer <token>`. The only exceptions are the `PUBLIC_API` set:
  - `POST /api/auth/login`, `POST /api/auth/signup`, `GET /api/auth/me`, `POST /api/auth/logout`
  - `GET /api/org/personas`, `GET /api/currency`, `GET /api/health`
  - `GET /api/planning/calendar.ics`
  - Anything else without a valid token gets 401 `Please sign in to continue.`
- JSON bodies are limited to 60mb so large CSVs can be uploaded.

| Method & path | Client call (`client/src/api.js`) | Behaviour |
|---|---|---|
| GET `/api/builder/catalog` | `api.builderCatalog()` | Returns `catalog()` |
| GET `/api/builder/data` | `api.builderData()` | Returns `dataBundle()` |
| GET `/api/builder/dashboards` | `api.dashboards()` | Returns the full list (seeds if needed) |
| GET `/api/builder/dashboards/:id` | `api.dashboard(id)` (defined, **unused** by the pages) | 404 `Dashboard not found` |
| POST `/api/builder/dashboards` | `api.createDashboard({name, widgets})` | 400 with the error message on name errors. Audit `dashboard.create` {label: name, entityId} |
| PUT `/api/builder/dashboards/:id` | `api.updateDashboard(id, {name, widgets})` | 404 if missing, 400 on name errors. Audit `dashboard.update` |
| DELETE `/api/builder/dashboards/:id` | `api.deleteDashboard(id)` | **Soft delete**: `trashDashboard(id, actorName)` records a recycle-bin item `{type:'dashboard', label:name, sublabel:'Dashboard', count:1, by, payload:{dashboard}}`, then deletes. Returns `{trash}` (`{trash:null}` if not found). Audit `dashboard.delete` |
| GET `/api/dataset/campaign/meta` | `api.campaignDatasetMeta()` | `{loaded:false}` or `{loaded:true, rows, source, updatedAt, dateRange:{start,end}\|null, dimensions:{channels,types,segments}}` |
| POST `/api/dataset/campaign/import` | `api.importCampaignDataset(payload)` | `requireRole('Editor')`. Body is `{csv, name}` or `{fileBase64, name}`. Runs `toCsvText`, then `ingestCsv(csv, name\|\|'uploaded')`. Audit `import.dataset` {label: name\|\|'campaign dataset', meta:{rows}}. Errors give 400 |

- The builder dashboard routes have **no role gate** beyond being signed in. A signed-in Viewer can save or delete through the API, and the client does not block them in the Builder either (see §11.7).
- `requireRole(min)`:
  - If no users exist or there is no token, the request passes as anonymous. In practice the auth wall already requires a token.
  - An invalid token gets 401 `Your session has expired — please sign in again.`
  - Too low a rank gets 403 `This action requires the <min> role or higher.`
- Restore from the recycle bin (`/trash`) calls `insertDashboard(payload.dashboard)`.

### 11.4 The catalog

`GET /api/builder/catalog` returns `{ datasetSources: string[], widgetTypes: [...], sources: [...] }`. `datasetSources` (ids of dataset-backed sources) is not used by the client.

#### 11.4.1 Widget types, in this order

| id | name | desc (button tooltip and caption) | kind | icon (unused by client) |
|---|---|---|---|---|
| kpi | KPI tile | One headline number | single | number |
| gauge | Gauge | One value against a target | single | gauge |
| line | Line chart | Trend over time | series | line |
| area | Area chart | Filled trend over time | series | area |
| bar | Bar chart | Compare categories | series | bar |
| donut | Donut chart | Share of a whole | breakdown | donut |
| funnel | Funnel | Stages narrowing by a measure | breakdown | funnel |
| table | Table | Rows of records | table | table |

**Kinds** decide two things: which sources are compatible (`source.kinds.includes(type.kind)`) and how fields are bound:
- `single`: one `field` from an object source.
- `series`: one or more `fields`, plotted over the source's X key.
- `breakdown`: one `field`, sliced by the X key.
- `table`: fixed `columns` with no field choice.

#### 11.4.2 Built-in sources, in order. Field format codes: money, number, x, percent

1. **`campFlex`**: "Campaigns — custom axes (pick X & Y)"
   - kinds `['series','breakdown']`, `flexible: true`, default `x: 'channel'`.
   - `dims`, each with `{id, name, bundleKey, field}`:
     - channel / "Channel" / `campByChannel` / `channel`
     - region / "Region" / `campByRegion` / `region`
     - status / "Status" / `campByStatus` / `status`
   - `fields` (measures): revenue "Revenue" money, pipeline "Pipeline" money, spent "Spend" money, budget "Budget" money, leads "Leads" number.
2. **`overview`**: "Overview KPIs", kinds `['single']`. Fields:
   - totalRevenue "Attributed revenue" money
   - totalSpent "Marketing spend" money
   - totalPipeline "Pipeline" money
   - totalLeads "Total leads" number
   - blendedRoi "Blended ROI" x
   - budgetUtilization "Budget utilization" percent
   - activeCampaigns "Active campaigns" number
3. **`timeseries`**: "Daily performance", kinds `['series']`, x `date`. Fields: revenue money, spend money, leads number, sessions number. Names: Revenue, Spend, Leads, Sessions.
4. **`channels`**: "Revenue by channel", kinds `['breakdown','series']`, x `channel`. Fields: revenue "Revenue" money.
5. **`budgetChannel`**: "Budget by channel", kinds `['series']`, x `category`. Fields: planned "Planned" money, actual "Actual" money.
6. **`budgetRegion`**: "Budget by region". Same as budgetChannel.
7. **`campaigns`**: "Campaigns", kinds `['table']`. Columns:
   - name "Campaign", channel "Channel", status "Status"
   - spent "Spent" money, leads "Leads" number, pipeline "Pipeline" money, roi "ROI" x
8. **`goals`**: "Goals", kinds `['table']`. Columns:
   - metric "Goal", actual "Actual", target "Target"
   - attainmentPct "Attainment %" percent, status "Status"
   - Actual and Target have no format, so they render as raw unformatted numbers.

#### 11.4.3 Dataset-backed sources (only after an upload; `group: 'Campaign dataset'`, which the client ignores)

`METRIC_FIELDS`, shared by the dimension sources:
- count "Campaigns" number
- roi "Avg ROI" x
- conversionRate "Avg conversion %" percent
- clicks "Clicks" number
- impressions "Impressions" number
- acquisitionCost "Acquisition cost" money
- engagement "Avg engagement" number
- ctr "CTR %" percent

The sources:
- **`cmpFlex`**: "Campaign dataset — custom axes (pick X & Y)"
  - kinds series and breakdown, flexible, default x `channel`, fields METRIC_FIELDS.
  - dims:
    - channel "Channel" → `cmpChannel`.`channel`
    - type "Campaign type" → `cmpType`.`type`
    - segment "Customer segment" → `cmpSegment`.`segment`
    - audience "Target audience" → `cmpAudience`.`audience`
    - location "Location" → `cmpLocation`.`location`
- **`cmpKpis`**: "Campaign dataset — KPIs", kind single. Fields:
  - campaigns "Campaigns" number
  - avgRoi "Average ROI" x
  - avgConversionRate "Avg conversion %" percent
  - totalClicks "Total clicks" number
  - totalImpressions "Total impressions" number
  - totalAcquisitionCost "Total acquisition cost" money
  - avgEngagement "Avg engagement score" number
  - ctr "Overall CTR %" percent
- **Dimension sources** (kinds series and breakdown, fields METRIC_FIELDS):
  - `cmpChannel` "Campaign dataset — by channel" (x `channel`)
  - `cmpType` "… — by campaign type" (x `type`)
  - `cmpSegment` "… — by customer segment" (x `segment`)
  - `cmpAudience` "… — by target audience" (x `audience`)
  - `cmpLocation` "… — by location" (x `location`)
- **`cmpMonthly`**: "Campaign dataset — monthly trend", kind series, x `date` (values are `YYYY-MM`). Fields: clicks "Clicks" number, impressions "Impressions" number, conversionRate "Conversion %" percent, roi "ROI" x, acquisitionCost "Acquisition cost" money.
- **`cmpCompanies`**: "Campaign dataset — top companies", kind table. Columns: company "Company", count "Campaigns" number, roi "Avg ROI" x, conversionRate "Conv %" percent, clicks "Clicks" number, impressions "Impressions" number.

#### 11.4.4 Compatibility, i.e. what the "Data source" dropdown lists for each widget kind
- **single** (KPI, Gauge): overview, plus cmpKpis.
- **series** (Line, Area, Bar): campFlex, timeseries, channels, budgetChannel, budgetRegion, plus cmpFlex, cmpChannel, cmpType, cmpSegment, cmpAudience, cmpLocation, cmpMonthly.
- **breakdown** (Donut, Funnel): campFlex, channels, plus cmpFlex and the five cmp dimension sources.
- **table**: campaigns, goals, plus cmpCompanies.

### 11.5 The data bundle (`GET /api/builder/data`)

Keys and their shapes. Widgets read `data[widget.source]`; flexible sources read `data[dim.bundleKey]`.

**Built-in keys:**
- `overview`: an object.
  - Fields: `{totalBudget, totalSpent, budgetUtilization, totalPipeline, totalRevenue, totalLeads, blendedRoi, activeCampaigns}`.
  - `budgetUtilization` is `(spent/budget*100)` to 1 decimal. `blendedRoi` is revenue/spent to 2 decimals.
  - Demo values: budget 1,550,000; spent 1,121,000; utilisation 72.3; pipeline 9,680,000; revenue 4,755,000; leads 9,700; ROI 4.24; active 6.
- `timeseries`: `timeseries(30)`, giving `[{date:'YYYY-MM-DD', spend, leads, revenue, sessions}]` for the 30 days ending 2026-07-15. It is a deterministic ramp.
- `channels`: `[{channel, revenue}]`, revenue summed per campaign channel.
- `budgetChannel`: `[{category: channel, planned, actual}]` over 6 channels: Paid Search, Paid Social, Organic / SEO, Email, Events & Webinars, Content Syndication.
- `budgetRegion`: `[{category: region, planned, actual}]` over North America, EMEA, APAC, LATAM.
- `campaigns`: the 8 "Nova" demo campaigns.
  - Fields: `{id:'cmp-10x', name, channel, region, status, budget, spent, leads, pipeline, revenue, roi, startDate}`.
  - Statuses: Active, Planned, Completed.
- `goals`: `goals().goals`, giving `[{metric, unit, actual, target, forecast, attainmentPct, forecastPct, status:'on-track'|'at-risk'|'off-track'}]`.
- `campByChannel`, `campByRegion`, `campByStatus`: `campaignsBy(dim)`.
  - Groups the campaigns by the dim (a missing value becomes '—').
  - Sums revenue, pipeline, spent, budget and leads.
  - Sorted by revenue, descending.
  - Row shape is `{[dim]: key, revenue, pipeline, spent, budget, leads}`.

**After a dataset upload,** `...builderBundle()` adds:
- `cmpKpis` (object), `cmpChannel`, `cmpType`, `cmpSegment`, `cmpAudience`, `cmpLocation`, `cmpMonthly`, `cmpCompanies`.
- Before any upload it adds nothing.

Neither `campFlex` nor `cmpFlex` is a bundle key; flexible sources are resolved through their dims.

### 11.6 External dataset ingestion (a Builder data source)

**Client side (`lib/files.js`):**
- `UPLOAD_ACCEPT = '.csv,.tsv,.txt,.xlsx,.xls,.ods,text/csv'`.
- `readUpload(file)`:
  - `.csv`, `.txt` and `.tsv` are read with `readAsText` and give `{csv, name}`.
  - Anything else is read with `readAsDataURL`; the base64 after the comma gives `{fileBase64, name}`.
  - A read error rejects with `Could not read the file.`

**Server side, conversion (`spreadsheet.js` `toCsvText`):**
- Non-blank `csv` is returned as-is.
- Otherwise `fileBase64` is decoded to a Buffer. Names ending .csv, .txt or .tsv are decoded as utf8 text.
- Anything else goes through `XLSX.read(buf,{type:'buffer', cellDates:true})`. The first sheet is converted with `sheet_to_csv({dateNF:'yyyy-mm-dd', blankrows:false})`. If there is no sheet: `The workbook has no readable sheet.`
- Nothing usable: `No file content provided.`

**Server side, aggregation (`campaignDataset.aggregate`):**
- Parses with `importer.parseCsv`.
  - Fewer than 2 rows: `CSV needs a header row and at least one data row.`
- Header aliases are matched case-insensitively against trimmed headers:
  - company: company
  - type: campaign_type, type
  - audience: target_audience, audience
  - channel: channel_used, channel
  - conversion: conversion_rate, conversion
  - acquisition: acquisition_cost, acquisition, cost
  - roi: roi
  - location: location
  - date: date
  - clicks: clicks
  - impressions: impressions
  - engagement: engagement_score, engagement
  - segment: customer_segment, segment
  - If channel, type and roi are all missing: `This does not look like a campaign dataset (no recognizable columns).`
- Building each record:
  - Text dimensions are trimmed and default to '—'.
  - Numbers strip `$`, `,`, `%` and whitespace; non-finite becomes 0.
  - `month` comes from a `YYYY-M(M)` or `YYYY/M` regex, or else from `new Date()`, giving `YYYY-MM`.
  - No records: `No data rows found.`
- KPIs:
  - `campaigns` = n.
  - `avgConversionRate` = mean(conversion)×100 to 2 decimals. The input is assumed to be a fraction.
  - `avgRoi` = mean to 2 decimals.
  - `totalClicks`, `totalImpressions`.
  - `totalAcquisitionCost` = rounded sum.
  - `avgEngagement` = mean to 2 decimals.
  - `ctr` = clicks/impressions×100 to 2 decimals.
- `groupBy(dim)` produces `{key, count, clicks, impressions, acquisitionCost(rounded), conversionRate (mean×100, 2dp), roi (mean, 2dp), engagement (mean, 2dp), ctr}`. Then `key` is renamed to the dim name. Rates are averaged; volumes are summed.
  - `byChannel`, `byType`, `bySegment` and `byAudience` are sorted by count, descending.
  - `byLocation` is the top 12 by count.
  - `topCompanies` is the top 15 by clicks.
  - `monthly` is a sorted list of distinct months: `{date: 'YYYY-MM', clicks, impressions, conversionRate, roi, acquisitionCost}`.
- `meta`:
  - `{rows, source: name, updatedAt, dateRange: {start,end} | null}`.
  - `dimensions`: distinct counts of channels, types and segments.

**Persistence:**
- `ingestCsv` writes the summary (not the raw rows) to `campaignDataset.json`, keeps a module-level `CACHE`, and returns `{loaded:true, ...meta}`.
- A new upload **replaces** the previous dataset. Only one is ever stored.
- `getSummary()` reads the cache first, then the file.

### 11.7 Builder page (`/builder`)

#### 11.7.1 Header
`PageHeader`, which includes the currency picker at the top right:
- eyebrow "Dashboard builder"
- title "Build any dashboard —"
- highlight "no code required."
- subtitle "Drag in charts and tables, bind them to your data, arrange and save."

Content sits in `mx-auto max-w-content px-5 pb-24`, wrapped in `StateWrap`.

#### 11.7.2 Permissions
- `canEdit = !user || user.role !== 'Viewer'`.
- In the Builder, this **only** disables the dataset upload button. Saving, editing and deleting are not role-gated in the UI.

#### 11.7.3 Load sequence on mount
- Runs `Promise.all([builderCatalog, builderData, dashboards, campaignDatasetMeta])`.
- Any failure shows the StateWrap error.
- **Deep link:** `?dash=<id>` opens that dashboard for editing (used by the Dashboards "Edit" button and the AI assistant's "Open my dashboard" link).
- Otherwise, including `?new=1` (whose param is ignored, it is simply the default), the page starts on a **blank unsaved canvas**:
  - `activeId=''`.
  - `name` = the first free of "New dashboard", "New dashboard 2", … (case-insensitive check against existing names).
  - `widgets=[]`.
- The query string is read only on mount. The URL is not updated afterwards.

#### 11.7.4 Toolbar
A card with `card mb-8 flex flex-wrap items-center gap-3 p-4`, containing in order:

1. **Dashboard picker.** MUI Select, small, minWidth 190.
   - Label "Dashboard", always shrunk and notched.
   - `displayEmpty`. The render value is the active dashboard's name, or "New dashboard (unsaved)" when there is no activeId.
   - The first menu item is a disabled italic "New dashboard (unsaved)", followed by one item per dashboard (in server order).
   - Picking one calls `loadDashboard(d)`: it copies the widgets and clears dirty. Unsaved changes are **silently discarded** (no confirm).
2. **Name field.** TextField, small, label "Name", minWidth 220, flex `1 1 220px`.
   - Typing sets dirty.
   - `error` is shown when the name duplicates another dashboard: trimmed, case-insensitive, excluding the active id. This mirrors the server check.
3. **Button group.** Every button uses `btnSx = {textTransform:'none', borderRadius:999, boxShadow:'none'}`.
   - "Add widget": contained, AddRounded icon. Opens the editor in add mode.
   - Save button: contained primary, SaveRounded icon.
     - Label is "Save" when there is no activeId, "Save changes" when dirty, otherwise "Saved".
     - Disabled when the name is a duplicate, the name is blank, or (not dirty and activeId is set). A blank canvas with a free name can therefore always be saved, even with zero widgets.
   - "New": outlined, DashboardCustomizeRounded icon. Resets to a blank unsaved canvas with a fresh free name. Toast "Blank canvas — add widgets, then Save." Nothing is persisted until Save.
   - "View dashboards": text variant, VisibilityRounded icon. Navigates to `/dashboards`.
   - Delete: IconButton, DeleteOutlineRounded, wrapped in a tooltip "Delete this dashboard". Disabled without an activeId.
4. **Duplicate-name warning.** Shown only when the name is a duplicate, as a full-width `basis-full` line, 12.5px, weight 500, colour `#c0392b`: `A dashboard named “<name>” already exists — pick a different name.`
5. **Dataset row.** Full width, top hairline border, `pt-3`.
   - Left side: DatasetRounded icon (17px) and 13px ink2 text.
     - When loaded: `Dataset: <b>N rows</b> from <source>` (N via toLocaleString), followed by ` · <start> → <end>` if there is a date range.
     - When not loaded: "No external dataset loaded yet."
   - Then a spacer.
   - Upload button: small, outlined, UploadFileRounded icon, a `<label>` wrapping a hidden file input with `accept=UPLOAD_ACCEPT`.
     - Label is "Loading…" while uploading, "Replace dataset" if loaded, otherwise "Upload dataset".
     - Disabled when `!canEdit` or while uploading.
   - When loaded, helper text at 12px ink2: `Add a widget and pick a <b>“Campaign dataset — …”</b> source to chart it (by channel, type, segment, audience, location, monthly trend, or top companies).`

#### 11.7.5 Save behaviour
- `activeId` set: PUT `{name, widgets}`. Otherwise: POST `{name, widgets}`.
- Then re-fetches the list, sets activeId to the saved id, and clears dirty.
- Toast "Dashboard saved." on success; on error, the server's message (e.g. the duplicate-name message).

#### 11.7.6 Upload behaviour
1. Read the file, then POST the import.
2. Re-fetch catalog, data bundle and dataset meta in parallel. New sources appear immediately, with no reload.
3. Success toast: `Dataset loaded: <rows> rows. New sources are in "Add widget".`
4. On error, a toast with the error message.
- The input value is reset so the same file can be uploaded again.

#### 11.7.7 Delete behaviour
- No confirmation dialog.
- Calls `api.deleteDashboard(activeId)`, which is a server soft delete into the recycle bin, restorable at `/trash`.
- Then `deleteSnapshotsFor(activeId)` removes this dashboard's local PNG snapshots.
- Then the list is re-fetched. If any dashboard remains, the **first one is loaded into the editor**. Otherwise the canvas is cleared with an empty name.
- Toast "Dashboard deleted.". API errors here are not caught.

#### 11.7.8 Canvas
- **Empty state** (no widgets): a full-width button.
  - Classes: `rounded-[22px] border-2 border-dashed border-black/10 py-24`, hover `border-blue/40 bg-black/[0.015]`, marked `data-no-snapshot`.
  - Contents: a 56×56 `rounded-2xl` tile with gradient `linear-gradient(135deg,#2a78d6,#9f4bd6)` and a white AddRounded icon; "Your canvas is empty" (19px semibold ink); "Click to add your first widget." (14px ink2).
  - Clicking opens the editor.
- **Grid:** `grid grid-cols-1 gap-6 lg:grid-cols-3`. Each widget gets `lg:col-span-{w}`, so it is single-column below the `lg` breakpoint (1024px).
- **Widget card:** `card group relative flex flex-col p-5`, `draggable`.
  - Header, left side: a DragIndicatorRounded icon (18px, `#c7c7cc`, cursor grab); the title (15px semibold ink, truncated, "Untitled" if empty); and a subline (12px ink2, truncated) `<Type name> · <Source name or raw source id>`.
  - Header, right side: controls marked `data-no-snapshot`, **hidden (opacity 0) until the card is hovered** (`group-hover:opacity-100`, with an opacity transition).
    - Three width buttons S/M/L, 24×24px, `rounded-md`, 11px bold, tooltip `Width S|M|L`.
    - Active width: background `#0071e3`, white text. Inactive: `var(--chip)` background, `var(--ink-2)` text. Clicking sets `w` and dirty.
    - Edit IconButton (EditRounded, 16px) opens the editor prefilled.
    - Delete IconButton (DeleteOutlineRounded, 16px) removes the widget immediately and sets dirty.
  - Body: `min-h-[140px] flex-1`, containing `<WidgetRenderer widget source data/>`.
- **Drag to reorder** uses native HTML5 drag and drop on the whole card:
  - `onDragStart` sets dragIndex. The dragged card renders at `opacity-40`.
  - `onDragOver` calls `preventDefault`.
  - `onDrop(targetIndex)` splices the dragged item out and inserts it at the target index, then sets dirty.
  - `onDragEnd` clears the drag state. Dropping on itself does nothing.
  - There is no drop indicator and no touch support.
- **Toasts:** MUI Snackbar at bottom-center with autoHide 3000ms, containing a filled `Alert` with `sx borderRadius:3` (36px, because theme.shape is 12) and the given severity.

### 11.8 Widget editor dialog

- MUI Dialog, `maxWidth="sm" fullWidth`. Title is "Edit widget" or "Add a widget" (fontWeight 700). Content uses `dividers`.
- **Draft state:** `{id:null, type:'kpi', source:'', field:'', fields:[], x:'', title:'', w:1, threshold:null}`, plus a separate `threshText` string.
- Opening for edit copies the widget's values. `threshText` is `fmtNum(threshold.value)` or `''`.

#### 11.8.1 Section "1. Choose a widget"
- Labels are 13px semibold ink2.
- Eight type buttons in a 3-column grid, `rounded-xl border p-2.5`, tooltip = type desc.
- Selected button: border `#0071e3`, background `rgba(0,113,227,0.12)`. Unselected: border `var(--hairline)`, background `var(--surface)`.
- Each button shows `ChartThumb` (active state), the name (13px semibold) and the desc (11px ink2).
- Changing type resets source, field, fields, x and threshold (and threshText). It keeps title and width.

#### 11.8.2 Section "2. Pick a data source"
- Full-width Select labelled "Data source", listing the sources compatible with the type's kind. If none: a `MenuItem value=""` reading "No compatible source".
- On change:
  - `single` and `breakdown` kinds: `field` = the first field id.
  - `series` kind: `fields` = `[first field id]`.
  - Flexible source: `x` = the first dim id.
  - Title = `defaultTitle(...)`, **only if the title is currently empty**.
- `defaultTitle`:
  - Flexible source: `"<measure name> by <dim name>"`, e.g. "Revenue by Channel".
  - kpi or donut: the field's name.
  - Otherwise: the source name.

#### 11.8.3 "Choose what to show" (shown once a source is picked)
- **Flexible source:** a Select "X-axis · dimension" listing `dims`. Changing it **recomputes and overwrites** the title.
- **table kind:** an info box (`rounded-xl bg-black/[0.03] px-3 py-2`, 13px): `Shows columns: <comma-separated column names>`.
- **single or breakdown kind:** a Select labelled "Metric", or "Y-axis · measure" for a flexible source, listing `fields`.
- **series kind:** a row of checkboxes, one per field, labels 13.5px. A flexible source gets a "Y-axis · measure(s)" caption.

#### 11.8.4 Target threshold
Shown only if a source is picked and the type is one of `kpi, gauge, bar, line, area`.
- **Checkbox label:** "Target threshold — highlight when the value is on / off target". For bar, line and area it reads "values are" instead of "the value is".
- **Turning it on:** `value` = `suggestions[min(2, n-1)]` (with 5 suggestions that is the first "nice" value at or above the reference), or the reference value, or 0. Direction is `'above'`.
- **Reference value:**
  - `single` kind: `data[source][field]`.
  - Array sources: the **median** of `field` (or `fields[0]` for series).
  - Flexible sources have no bundle key, so no reference is found: there are no suggestions and the default is 0.
  - The format comes from the field's format, defaulting to `number`.
- **Panel when on** (`rounded-xl border border-hairline p-3`):
  - "Good when the value is", with two pill toggles: `≥ target` (above) and `≤ target` (below). Active pill: `#0071e3` background, white text. Inactive: `var(--chip)` background, ink-2 text.
  - A "Target" TextField (width 150, placeholder "e.g. 40 or 150K"). Typing updates `threshText`. When `parseHuman(text)` parses, it sets the value. Beside it: `= <b>formatted value</b>`, using the field's format via `fmtBy`.
  - "Suggestions:" chips, up to 5, formatted. Clicking one sets the value and `threshText = fmtNum(v)`. The selected chip is blue.
  - Help text (11px): `Type a number (K/M/B suffixes work) or pick a suggestion. The <noun> turn(s) green when on target, red when off.`
    - noun: kpi "tile", gauge "dial", bar "bars", line and area "points & zone".
    - "turns" (singular) is used for kpi and gauge.
- **`parseHuman`:** strips `$`, `,`, whitespace and `%`. Accepts `-?\d*\.?\d+` with an optional k, m, b or t suffix (case-insensitive; ×1e3, 1e6, 1e9, 1e12). Falls back to `Number()`. Returns null if unparseable.
- **`suggestThresholds(v)`:**
  - v = 0 or non-finite gives `[]`.
  - mag = 10^floor(log10|v|).
  - Candidates = mag × `[0.5,0.75,1,1.25,1.5,2,2.5,3,4,5,7.5,10]`, deduplicated with toPrecision(4) and sorted.
  - Result = the two largest candidates below |v| plus the first three at or above it, signed, sorted, at most 5.
  - Example: 4,755,000 gives [3M, 4M, 5M, 7.5M, 10M], and the default is 5M.

#### 11.8.5 Section "3. Title & size"
- A "Widget title" TextField (flex `1 1 220px`).
- S/M/L buttons, 36×40px, `rounded-lg`, 12px bold, same blue and chip styling as above.

#### 11.8.6 Actions
- "Cancel" (text button).
- Primary contained button "Update widget" or "Add to dashboard". Disabled unless `draftValid`:
  - A source is required.
  - A flexible source requires `x`.
  - `single` and `breakdown` kinds require `field`.
  - `series` requires at least one field.
  - `table` is valid once a source is picked.

#### 11.8.7 saveDraft
- Builds the widget:
  - `id`: the existing id, or `w-<ts36>-<rand4>`.
  - `title`: the draft title, or `defaultTitle` (computed without `x`).
  - `field`, or undefined.
  - `fields`, if non-empty.
  - `x`, only for flexible sources.
  - `threshold`, only for thresholdable types and only if set.
  - `w`.
- Replaces the widget in place (edit) or appends it (add), sets dirty and closes the dialog.
- Nothing reaches the server until the toolbar Save.

### 11.9 Dashboards page (`/dashboards`): the read-only viewer

#### 11.9.1 Header
- eyebrow "Dashboards"
- title "Your dashboards,"
- highlight "at a glance."
- subtitle "View any saved dashboard. Create and edit them in the Builder."

#### 11.9.2 Behaviour
- Loads catalog, data and the dashboard list. The **first dashboard** becomes active.
- `canEdit` is the same rule as the Builder (not Viewer).

#### 11.9.3 Toolbar (`mb-8 flex flex-wrap justify-between gap-3`)
- **Left:** one pill button per dashboard. Active: `btn-pill btn-blue`. Others: `btn-pill btn-ghost`. Clicking switches the active dashboard. With no dashboards: "No dashboards yet." (14px ink2).
- **Right (canEdit only):**
  - "Edit": outlined small, EditRounded icon, radius 999. Navigates to `/builder?dash=<activeId>`. Shown only when a dashboard is active.
  - "New dashboard": contained small, AddRounded icon. Navigates to `/builder?new=1`.

#### 11.9.4 Body states
1. **No dashboards:** a dashed empty-state button (same styling as the Builder's, plus `dark:border-white/10`). It shows a DashboardCustomizeRounded gradient tile, "No dashboards yet" and "Create your first one in the Builder.", and navigates to `/builder?new=1`.
2. **Active dashboard with widgets:**
   - A `ref=canvasRef` grid (`grid-cols-1 gap-6 lg:grid-cols-3`) of cards: `card flex flex-col p-5 lg:col-span-{w}`, each with a title, a `Type · Source` subline and a `WidgetRenderer`. There are no edit controls and no drag.
   - Below the grid: `<SnapshotPanel dashboardId={activeId} targetRef={canvasRef}/>`.
3. **Active dashboard with no widgets:** a card (`grid place-items-center p-16`) reading "This dashboard has no widgets yet.", plus (canEdit) a contained button "Add widgets in the Builder" that goes to `/builder?dash=<id>`.

### 11.10 Snapshots (dashboard PNG thumbnails) and widget-type thumbnails

#### 11.10.1 Snapshots
Implemented in `SnapshotPanel.jsx`, `lib/snapshot.js` and `lib/snapshotStore.js`. They are browser-only and never reach the server.

**Capture:**
- `captureNode(node, mode)` uses `html-to-image` `toPng` with `pixelRatio 1.5`.
- Background is `#0b0b0d` in dark mode and `#ffffff` in light mode.
- `cacheBust: true`. A filter excludes any element with a `data-no-snapshot` attribute.
- ECharts canvases are captured fine because they are same-origin.

**Store:**
- localStorage key `zenreach.snapshots.v1` holds a map `{dashboardId: Snapshot[]}`.
- Snapshot shape: `{id:'snap-<ts36>-<rand4>', dashboardId, label:'Snapshot · <toLocaleString>', image: dataURL, createdAt: ms}`.
- At most **6 per dashboard**; the newest is prepended and the list truncated.
- If localStorage throws on quota, the oldest snapshot across all dashboards (never the new one) is evicted repeatedly, up to 200 times. If it still fails: `This snapshot is too large for local storage — try a narrower dashboard or delete older snapshots.`
- With no dashboard id: `Save the dashboard first.`
- `deleteSnapshot(dashboardId, id)` and `deleteSnapshotsFor(dashboardId)`. The latter runs when a dashboard is deleted in the Builder.

**Panel UI** (`card mt-8 p-5`, itself marked `data-no-snapshot`):
- Header: a PhotoCameraRounded icon (18px), then "Snapshots" plus " · N" when N > 0.
- Button "Take snapshot": contained small, radius 999. While busy it shows a spinner and "Capturing…". Disabled while busy or without a dashboard id.
- Empty texts:
  - No id: `Save this dashboard first — then you can capture PNG snapshots to revisit later.`
  - None yet: `No snapshots yet. Capture the current dashboard to save a PNG (kept in this browser).`
  - The `noun` prop defaults to "dashboard".
- **Gallery:** wrapping 200px-wide tiles (`rounded-xl border-hairline`).
  - Each tile has an image at `aspect-[16/10] object-cover object-top` (background `var(--gray-bg)`, lazy-loaded), plus a footer with the timestamp (11px) and a delete IconButton with tooltip "Delete snapshot".
  - Clicking the image opens a Dialog (`maxWidth lg fullWidth`). It has a header with the label, an "Open in new tab" button (`window.open()` and `document.write('<img …>')`), a close button, and the image at `max-h-[78vh] object-contain`.
- Toasts:
  - Success: "Snapshot saved to this browser."
  - Error: "Snapshot failed: <msg>"
  - Info: "Save the dashboard first, then take a snapshot."

#### 11.10.2 Widget-type thumbnails (`ChartThumb.jsx`)
- Inline SVG, `viewBox 0 0 48 30`, width 100%, height 24px, `aria-hidden`.
- Colour is `#0071e3` when active, otherwise `var(--ink-2)`.
- Shared axis path: `M6 3 V26 H44`, stroke opacity 0.35, width 1.2.

| Type | Glyph |
|---|---|
| kpi | A short 30%-opacity bar (8,6, 20×4, rx2) over a solid block (8,13, 32×9, rx2) |
| gauge | A 25%-opacity semicircle arc `M8 24 A16 16 0 0 1 40 24` (width 4) with a solid partial arc to (22,8.6), a needle `M24 24 L30 14` and a hub circle r2 |
| line | Axis, polyline 8,20 → 16,12 → 24,16 → 32,7 → 40,11 (width 2), with dots r1.6 |
| area | Axis, the same path filled at 22% opacity, plus the line (width 1.8) |
| bar | Axis, 5 bars at x 10, 17, 24, 31, 38 with tops 14, 8, 17, 5, 11, width 5, opacity 0.55 + i×0.09 |
| donut | A 30%-opacity ring (r11, width 6) with a solid arc segment |
| funnel | Three trapezoids at opacity 1, 0.65 and 0.4 |
| table | An outlined rect with a header line and two column and row lines |
| default | A 30%-opacity rect |

### 11.11 Widget rendering (`WidgetRenderer.jsx`)

**Data resolution:**
- `dataset = data[widget.source]` and `xKey = source.x`.
- For a flexible source: `dim = dims.find(id === widget.x) || dims[0]`, `dataset = data[dim.bundleKey]`, `xKey = dim.field`.
- No source, or dataset undefined: an empty message "No data source selected." (centered, min-h 120px, 13px ink2).

**Formatting (`fmtBy(format, v)`):**
- undefined, null or NaN: returns `String(v ?? '—')`. A non-numeric string comes back as itself.
- `money`: `fmtMoney`. Converts to the active currency and abbreviates: `$1.23M`, `$12.3K`, `$123`, with the sign before the symbol.
- `moneyFull`: `Intl` currency format with 0 decimals.
- `percent`: `n.toFixed(1)%`.
- `x`: `n.toFixed(2)x`.
- `number`: `fmtNum` (`1.23M`, `12.3K`, rounded integer; no B suffix).
- Anything else: `String(v)`.

**Threshold colours:** GOOD `#12a150`, BAD `#e0403f`.
- `validThreshold(w)` returns `{value:Number, direction||'above'}` or null.
- `onTarget(v, th)`: `below` means v ≤ target; otherwise v ≥ target.
- `niceCeil(n)` rounds up to 1, 2, 5 or 10 × 10^k, and returns 1 for n ≤ 0.

#### 11.11.1 KPI
- A flex column centered vertically.
- Label: the field name (13px, text.secondary, weight 500).
- Value: `fmtBy` (44px, weight 800, letter-spacing -0.03em, line-height 1). The colour is GOOD or BAD when a threshold is set, otherwise text.primary.
- With a threshold, a pill badge (`rounded-full px-2.5 py-1`, 12px semibold). Colour is GOOD or BAD; the background is the same hex with `22` alpha. Text is `▲ On target · ≥ X` or `▼ Off target · ≤ X`. The ≥/≤ symbol follows the direction; the arrow follows met or not met.

#### 11.11.2 Gauge
- ECharts `gauge`, height 168, `notMerge`.
- Value is `Number(field)` or 0.
- `max`: 100 if the format is percent, otherwise `niceCeil(max(value, target??0, 1) × 1.2)`.
- Colour: GOOD or BAD with a threshold, otherwise `SERIES[0]` `#2a78d6`.
- Options:
  - `center ['50%','74%']`, `radius '116%'`, start angle 180, end angle 0 (a semicircle), min 0, splitNumber 4.
  - `progress {show, width 16, roundCap}`, no pointer.
  - Track: `axisLine` width 16 in `rgba(color,0.14)`.
  - Ticks, split lines, labels, anchor and title are all hidden.
  - `detail`: valueAnimation, `offsetCenter [0,'-6%']`, formatter `fmtBy`, the colour, 30px, weight 800.
- With a threshold, below the dial: the caption `target ≥|≤ X` (11px ink2, -mt-3), then a pill `✓ On target` or `✕ Off target`.

#### 11.11.3 Donut
- Needs a non-empty array, otherwise "No rows to chart.".
- Name key is xKey. Value key is `widget.field` or the first field.
- ECharts pie: `radius ['52%','82%']`, center, `avoidLabelOverlap:false`, transparent 2px borders, no labels or label lines, height 180.
- Tooltip (item): `name<br/><b>value</b> · percent%`.
- Below the chart, a legend list of the **first 6** slices: a 10px dot in `SERIES[i]`, the name in ink2, and the value right-aligned, semibold, tabular-nums, at 12.5px.

#### 11.11.4 Funnel
- Same data rules as donut. Each item is coloured `SERIES[i]` (assigned **before** sorting), then sorted by value, descending.
- ECharts funnel: top and bottom 10, left and right 6%, `minSize '14%'`, gap 2, `sort 'descending'`.
- Labels sit inside, white, 12px, formatter `{b}`. Tooltip: `name<br/><b>value</b>`. Height 240.

#### 11.11.5 Table
- Needs a non-empty array, otherwise "No rows.".
- Columns come from `source.columns`. Shows the **first 12 rows**.
- Container is `max-h-[280px] overflow-auto`. The table is `w-full border-collapse` at 13px.
- Header row: left-aligned, ink2, sticky top, background `bg-white/95` (dark `#1c1c1e/95`) with `backdrop-blur`, `px-2 py-2`, weight medium.
- Rows have a top border `black/[0.06]`. Cells are `whitespace-nowrap px-2 py-2 tabular-nums`, using `fmtBy` when the column has a format, otherwise `String(value ?? '—')`.

#### 11.11.6 Line, area and bar (series)
- Needs a non-empty array.
- `fields` = `widget.fields`, or `[first field]`.
- `isDate` = `xKey === 'date'`.
- Y tick formatter, chosen by the first field's format: money gives `fmtMoney`; percent gives `` `${v}%` ``; anything else gives `fmtNum`.
- Categories are `dataset.map(r => r[xKey])`.
- `multi` = more than one field.
- **Threshold handling** (only when exactly one field):
  - `vmax` = `niceCeil(max(values, target) × 1.08)`, with the y axis forced to `min 0, max vmax`.
  - `markLine` (silent, no symbol): a dashed line in `#111` at opacity 0.6, width 1, at `yAxis = target`. Label `Target ≥|≤ X`, position `insideEndTop`, 10px, `#111`. The hard-coded `#111` is poor in dark mode.
  - `markArea`: a good zone in `rgba(GOOD,0.08)` and a bad zone in `rgba(BAD,0.08)`. For 'above', good runs from target to vmax and bad from 0 to target; 'below' flips them.
  - Both are attached to the first series only.
- **Bar series:**
  - `barMaxWidth 30`, `barGap '10%'`, `itemStyle {color: SERIES[i], borderRadius [6,6,0,0]}`.
  - With a threshold, each bar is coloured GOOD or BAD individually.
- **Line and area series:**
  - `type 'line'`, `smooth`, `showSymbol:false`, line width 2.5 in `SERIES[i]`.
  - Area adds `areaStyle: areaGradient(color)`, which fades from 0.28 to 0 alpha.
  - A threshold does not recolour the line, despite the "points & zone" wording in the editor.
- **Option:**
  - `grid {left 6, right 12, top multi?34:12, bottom isDate?4:18, containLabel:true}`.
  - Legend only when multi: top 0, circle icon 9×9, text 12px.
  - `tooltip {trigger:'axis', axisPointer: bar?'shadow':'line', formatter: seriesTooltip(per-series formatter map by series display name)}`.
  - `xAxis`: category, `boundaryGap` only for bar. Labels: date values show `slice(5)` (`MM-DD`, or `MM` for monthly `YYYY-MM`). Rotation is 0 for dates, otherwise 24° when there are more than 6 categories. `hideOverlap`, 11px.
  - `yAxis`: value, with the tick formatter, 11px.
  - Height 260, `notMerge`.

### 11.12 ECharts foundation (`lib/chart.js`) and `EChart.jsx`

**Registration:** tree-shaken ECharts 6 core with:
- Charts: LineChart, BarChart, PieChart, GaugeChart, FunnelChart.
- Components: Grid, Tooltip, Legend, MarkLine, MarkArea, Title.
- Features: `LegacyGridContainLabel` (needed because ECharts 6 moved `grid.containLabel` behind a feature flag).
- Renderer: CanvasRenderer.

Exports: `echarts`, and `PALETTE = SERIES`.

**Themes** `zr-light` and `zr-dark`, from `makeTheme(dark)`:
- `color: SERIES`, transparent background.
- Font: `-apple-system, BlinkMacSystemFont, "Segoe UI", Inter, sans-serif`.
- Title colour: ink `#1d1d1f` light, `#f5f5f7` dark.
- Axes:
  - No axis line and no ticks.
  - Labels `#6e6e73` light, `#9a9aa0` dark, 11px.
  - Split lines `rgba(0,0,0,0.07)` light, `rgba(255,255,255,0.08)` dark.
  - The category axis hides split lines; the value axis shows them.
- Legend text: `#3a3a3c` light, `#c9c9ce` dark.
- Tooltip:
  - Background `rgba(255,255,255,0.97)` light, `rgba(28,28,30,0.96)` dark.
  - Border `rgba(0,0,0,0.08)` light, `rgba(255,255,255,0.12)` dark, width 1.
  - Text in the ink colour at 12px.
  - `extraCssText: border-radius:14px; box-shadow:0 10px 30px -8px rgba(0,0,0,0.22); backdrop-filter:blur(12px); padding:8px 12px;`

**Helpers:**
- `rgba(hex, a=1)`: supports 3-digit and 6-digit hex.
- `areaGradient(color, top=0.28, bottom=0)`: a vertical linear gradient.
- `seriesTooltip(fmts)`, where `fmts` is a function or a map of series name to formatter. Produces HTML:
  - A header `<div style="font-weight:600;margin-bottom:4px">axisValueLabel||name</div>`.
  - One row per series: `display:flex;align-items:center;gap:8px`, then the marker, a `flex:1` label (the series name, or the item name if the series is named 'series'), then the value in `<b style="margin-left:12px">`.

**`EChart({option, height=300, className, style, notMerge=false})`:**
- Renders a div at `width:100%`, `height` px.
- Initialises an ECharts instance with the theme matching the current mode (`zr-dark` or `zr-light`) and the canvas renderer.
- **Re-creates the instance when the theme flips**, because the theme is applied at init.
- A ResizeObserver calls `chart.resize()`.
- When `option` changes it calls `setOption(option, notMerge)`.
- Disposes on unmount.
- Options are rebuilt on every render, including currency changes, so `setOption` runs on every re-render. Currency changes re-format chart values this way.

### 11.13 AI-assistant hook into the Builder

- `server/services/assistantDashboard.js` drives a chat wizard ("build a dashboard").
- It uses `nameExists`, `suggestName` and `listDashboardNames`, and calls `createDashboard({name, widgets})` directly.
- It returns `link: {label:'Open my dashboard', route:'/builder?dash=<id>'}`, which the floating assistant renders as a button that navigates there.
- Default names come from `suggestName('Campaign Performance')` and `suggestName('Marketing Overview')`.

### 11.14 Quirks and gaps to decide on in the rebuild

- No unsaved-changes guard on dashboard switch, New, navigation or reload.
- No confirmation on dashboard delete. After a delete, the first remaining dashboard is loaded into the editor rather than a blank canvas.
- The `?new=1` parameter is ignored. A blank canvas is already the default.
- Viewer role restrictions: in the Builder only the upload button is gated; the builder APIs have no role gate.
- Threshold suggestions don't work for flexible sources, which have no bundle key.
- Changing the X dimension overwrites a custom title.
- Goals table Actual and Target columns are unformatted.
- `datasetSources`, `widgetTypes[].icon`, the `group` on dataset sources and `api.dashboard(id)` are unused.
- The Builder still carries `data-no-snapshot` markers, but SnapshotPanel appears only on Dashboards.
- `hover:border-blue/40` uses Tailwind's opacity modifier on a `var(--blue)` colour, which Tailwind 3 can't apply. It is likely a no-op.
- Widget heights are not configurable. There is no free-form resize or 2-D placement.
- Only one external dataset at a time; a new upload replaces it.

## 12. Application shell and design system

**Source files:** `client/index.html`, `client/src/main.jsx`, `client/src/App.jsx`, `client/src/components/Layout.jsx`, `client/src/theme.js`, `client/src/index.css`, `client/tailwind.config.js`, `client/postcss.config.js`, `client/package.json`, `client/src/context/ThemeModeContext.jsx`, `client/src/components/ThemeToggle.jsx`, `client/src/lib/glossary.js`, `PageHeader`, `Panel`, `StateWrap`, `ScrollReveal`, `Reveal`, `InfoTip`, `BrandLogo`, `ChatMark`, `SectionHeading`, `ScrollToTop`.

### 12.1 Bootstrapping

**`client/index.html`:**
- `lang="en"`, charset UTF-8, `viewport width=device-width, initial-scale=1.0`.
- `<meta name="theme-color" content="#ffffff">`.
- Favicon `/favicon.svg`: a 32×32 rounded tile (rx 8) with a gradient from `#2a78d6` to `#9f4bd6` and the white BrandLogo glyph.
- Preconnects to fonts.googleapis.com and fonts.gstatic.com (crossorigin). Loads **Inter** at weights 400, 500, 600, 700, 800 and 900 with `display=swap`.
- Title: "ZenReach — Marketing performance, beautifully clear."
- `<div id="root">` and `<script type="module" src="/src/main.jsx">`.

**`main.jsx`:**
- Sets `history.scrollRestoration='manual'` and calls `window.scrollTo(0,0)`.
- Renders inside `React.StrictMode`, with providers in this order (outer to inner): `ThemeModeProvider`, `AuthProvider`, `CurrencyProvider`, `BrowserRouter`, `App`.
- Imports `index.css`.

**Vite** (`vite.config.js`):
- `@vitejs/plugin-react`, dev port 5173.
- Proxy `/api` → `http://localhost:4000`. The Express server listens on `PORT` or 4000.

**Root `package.json` scripts:**
- `install:all`
- `dev`: runs server and client together with `concurrently`, labelled "server,client", coloured blue and magenta.
- `build`: builds the client.
- `start`: starts the server.

### 12.2 Route table (`App.jsx`)

`App` subscribes to `useCurrency()`, so a currency switch re-renders the whole tree without remounting it. Page state is kept.

- **Before auth is ready** (stored token being validated): `<div className="min-h-screen bg-graybg"/>`, with no layout, to avoid a logged-out flash.
- **Signed in:** everything renders inside `<Layout>`. Order: `<ScrollToTop/>`, then `<div key={location.pathname} className="zr-page-enter">` containing the `<Routes>`, then `<AiAssistant/>`. Only the routed page is keyed, so the assistant keeps its chat history.

| Path | Page component |
|---|---|
| `/` | `Dashboard` (Home) |
| `/campaigns` | `Campaigns` |
| `/budget` | `Budget` |
| `/goals` | `Goals` |
| `/optimizer` | `Optimizer` |
| `/analytics` | `Analytics` |
| `/integrations` | `Integrations` |
| `/dashboards` | `Dashboards` |
| `/builder` | `Builder` (query `?dash=<id>`; `?new=1` is tolerated) |
| `/planning` | `Planning` |
| `/procurement` | `Procurement` |
| `/attribution` | `Attribution` |
| `/admin` | `Admin`. No route guard; the page itself shows a "You need the Admin or Owner role…" message when `user.role` is not Owner or Admin |
| `/trash` | `RecycleBin` |
| `/audit` | `AuditLog` |

- The signed-in tree has **no catch-all**. An unknown path renders an empty main area inside the layout.
- **Signed out:** only `/` renders `Dashboard`, which acts as the marketing landing page. `*` redirects with `<Navigate to="/" replace/>`. AiAssistant is not rendered.

**App-level effects:**
- **Cursor spotlight:** a single passive `pointermove` listener on `document`, throttled with rAF.
  - It finds `e.target.closest('.card-hover')` and sets `--mx` and `--my` (px relative to the card) on that card.
  - It is skipped entirely when `prefers-reduced-motion: reduce` is set.

**`ScrollToTop`:** on every `pathname` change it calls `window.scrollTo({top:0,left:0,behavior:'instant'})`. 'instant' overrides the global smooth scrolling. It renders null.

### 12.3 Layout and navigation (`components/Layout.jsx`)

#### 12.3.1 Navigation data

**MAIN**, in order:
1. `/` "Home" (HomeRounded)
2. `/planning` "Planning" (AccountTreeRounded)
3. `/builder` "Builder" (DashboardCustomizeRounded)
4. `/dashboards` "Dashboards" (SpaceDashboardRounded)
5. `/budget` "Budget" (PaidRounded)
6. `/analytics` "Analytics" (InsightsRounded)
7. **Admin only** (`user.role` in Owner, Admin): `/admin` "Admin" (AdminPanelSettingsRounded), appended to the end of MAIN.

**MORE**, under a "More" label, in order:
1. `/campaigns` "Campaigns" (CampaignRounded)
2. `/goals` "Goals" (FlagRounded)
3. `/optimizer` "Budget optimizer" (TuneRounded)
4. `/procurement` "Procurement" (ReceiptLongRounded)
5. `/attribution` "Attribution" (AltRouteRounded)
6. `/integrations` "Integrations" (ExtensionRounded)
7. `/audit` "Audit log" (HistoryRounded)
8. `/trash` "Recycle bin" (RestoreFromTrashRounded)

All icons are `@mui/icons-material` Rounded variants. Nothing is hidden in dropdowns.

#### 12.3.2 Logged-out layout
- `flex min-h-screen flex-col`.
- Header: `frost sticky top-0 z-50 border-b border-black/[0.07] dark:border-white/[0.08]`, inner `mx-auto h-14 max-w-content px-5 flex justify-between`. BrandMark on the left; on the right, `ThemeToggle` and `AccountControls` ("Log in" text button and "Sign up" contained button, both with border radius 100).
- `<main class="flex-1">`, then the footer.

#### 12.3.3 Signed-in layout

**Sidebar (`<aside>`):**
- Classes: `fixed inset-y-0 left-0 z-50 flex w-[244px] flex-col border-r border-hairline bg-graybg transition-[width,transform] duration-200 ease-out`.
- On desktop (`lg`, 1024px and up): always `lg:translate-x-0`, width `lg:w-[244px]` expanded or `lg:w-[72px]` collapsed.
- On mobile: an off-canvas drawer, always 244px wide. Hidden with `-translate-x-full`; open with `translate-x-0 shadow-2xl`.

**Sidebar top row** (h-14, px-4):
- `BrandMark`:
  - A NavLink to `/` with aria-label "ZenReach home".
  - Contains `BrandLogo size 28 radius 8` and "ZenReach" (16px semibold, tracking-tight, ink).
  - The label is hidden on `lg` when collapsed, and the row centers.
- A close IconButton (CloseRounded, aria "Close menu"), hidden on `lg`.

**Nav** (`flex-1 space-y-0.5 overflow-y-auto px-3 py-2`): the MAIN links, the "More" divider, then the MORE links.
- The "More" label is `px-3 pb-1 pt-4 text-[10.5px] font-semibold uppercase tracking-wider text-muted`. When collapsed it is hidden on lg and replaced by a hairline `mx-3 my-2 border-t`.

**`SideLink`:**
- A NavLink with `end` set only for `/`. Clicking closes the mobile drawer.
- Inner span: `relative flex items-center gap-3 rounded-xl py-2 text-[13.5px] transition-all duration-200 ease-out`, `px-3`. Collapsed adds `lg:justify-center lg:px-0`.
- **Active:**
  - `font-semibold text-ink`.
  - Background `linear-gradient(90deg, color-mix(in srgb, var(--blue) 15%, transparent), color-mix(in srgb, var(--blue) 4%, transparent))`.
  - A 3px × 16px left indicator bar (`rounded-r-full`, `var(--blue)`, vertically centered), shown only when expanded.
  - Icon coloured `var(--blue)`.
- **Inactive:** `font-normal text-ink2`. Hover: `translate-x-[2px]`, `bg-black/[0.05]` (dark `white/[0.06]`), `text-ink`.
- Icon 19px, flexShrink 0.
- The label is hidden on lg when collapsed. Collapsed links get an MUI Tooltip with the label (placement right, enterDelay 200).

**Sidebar footer** (`border-t border-hairline p-3`):
- **Collapse toggle:** desktop only (`hidden lg:flex`), `rounded-xl px-3 py-2 text-[12.5px] text-ink2`, hover background.
  - Expanded: ChevronLeftRounded (20px) plus the text "Collapse".
  - Collapsed: ChevronRightRounded only.
  - aria-label is "Collapse sidebar" or "Expand sidebar".
  - State persists in `localStorage.zr_sidebar_collapsed`, '1' or '0'.
- A row with `AccountControls iconOnly={collapsed}`, then `NotificationBell` and `ThemeToggle`. When collapsed on lg the row stacks vertically (`lg:flex-col lg:gap-2`).

**Mobile drawer behaviour:**
- Opened by the hamburger.
- Closes on route change, Escape, a click on the scrim, the close button or a link click.
- While open, body scrolling is locked (`body.style.overflow='hidden'`, restored on close).
- Scrim: `fixed inset-0 z-40 bg-black/30 lg:hidden`.

**Content column:**
- `flex min-h-screen flex-col transition-[padding] duration-200 ease-out`, with `lg:pl-[244px]` or `lg:pl-[72px]`.
- **Mobile top bar:** `frost sticky top-0 z-30 flex h-14 items-center gap-2 border-b border-hairline px-4 lg:hidden`.
  - Hamburger (MenuRounded, aria "Open menu"), then BrandMark.
  - On the right: NotificationBell and ThemeToggle. The account controls live only in the drawer footer.
- `<main class="flex-1">`, then the footer.

**Footer** (both layouts):
- `mt-8 border-t border-hairline bg-graybg`, inner `max-w-content px-5 py-4`, `text-[11.5px] text-muted`. Stacked on mobile; a row on `sm` and up with justify-between.
- Text: `© <current year> ZenReach — synthetic demo data. Not affiliated with Uptempo.` and `Marketing performance, minimal & clear.`

#### 12.3.4 Header and chrome controls

**Currency control:**
- Not in the sidebar. It sits in every **PageHeader**'s right-hand action area (`currency` prop, default true). The Home page also imports it directly.
- Layout: an optional "Currency" label (12px medium ink2, omitted when `compact`), a small spinner (13px, muted) while busy, then an MUI Select (small, height 32, font 13, border radius 2 = 24px).
- Selected value renders as `<symbol> <code>` in tabular-nums. Menu items show a 24px symbol column, then the code.
- Currencies come from `/api/currency`. Fallback list: USD, EUR, GBP, CAD, AUD, INR, JPY, SGD.
- Fallback rates (USD per unit): USD 1, EUR 1.08, GBP 1.27, CAD 0.73, AUD 0.66, INR 0.012, JPY 0.0064, SGD 0.74.
- Symbols: `$ € £ C$ A$ ₹ ¥ S$`.
- **On switch:** fetches fresh rates (`/api/currency?refresh=1`), stores `localStorage.zr_currency`, and sets the module-level active currency in `format.js`. Every `fmtMoney` and `fmtMoneyFull` call then converts USD to the active currency, and the whole app re-renders in place.

**Notification bell:**
- Polls `GET /api/notifications?user=<name>` on mount and every 30s.
- An IconButton (aria "Notifications") with an MUI Badge: `color="error"`, `max 9`, badge font 9, height 15, minWidth 15. Icon NotificationsNoneRounded, 21px.
- **Opening it** counts as seen: the count is zeroed optimistically and `POST /api/notifications/read` is sent.
- Popover: anchored bottom-right, width 340 (max 92vw), radius 1.5 (18px), `mt 1`.
  - Header "Notifications" (14px semibold), then a Divider.
  - List `max-h-[380px] overflow-auto`. Each item is a full-width button (`px-4 py-2.5`, hover `black/[0.03]`, dark `white/[0.05]`) with an icon, a title (12.5px semibold) and text (12px ink2). Clicking closes the popover and navigates to `n.route || '/'`.
  - Empty: "You're all caught up."
- Type icons and colours:

| Type | Icon | Colour |
|---|---|---|
| approval | CheckCircleRounded | `#0071e3` |
| due | ScheduleRounded | `#c77700` |
| overdue | WarningAmberRounded | `STATUS.critical` `#d03b3b` |
| mention | AlternateEmailRounded | `#9f4bd6` |
| assignment (also the fallback) | PersonAddAlt1Rounded | `#1aa179` |

**Account controls (`auth/AccountControls.jsx`):**
- **Signed in:**
  - A rounded-full button with a gradient-initials `Avatar` (32px; background `user.avatarGradient` or the brand gradient; initials or '?').
  - The first name (13px medium, max 120px, truncated) shows only on `xl` and up, unless `iconOnly`. `compact` always shows it.
  - Menu: anchored bottom-right, radius 1.25 (15px), minWidth 220.
    - Header: avatar 38px, name (13.5px semibold) and email (12px ink2), then a Divider.
    - "Your profile" (PersonRounded) opens `ProfileDialog`.
    - "Log out" (LogoutRounded) logs out, then shows a toast "You have been signed out." at top-center, offset `top {xs:70, sm:78}`, as a filled success Alert.
- **Signed out:** "Log in" and "Sign up" buttons. Both open `AuthModal` on the corresponding tab.
- **Global window events:**
  - `zr:open-auth` (detail `{tab:'signup'|'login'}`) opens the modal. Only the non-compact instance listens.
  - `zr:auth-modal` (detail `{open}`) is broadcast whenever the modal opens or closes.
- Before auth is ready it renders an 8×8 placeholder span.

**Theme toggle:**
- A small IconButton in text.primary colour, aria "Toggle light or dark mode".
- Tooltip "Switch to light mode" or "Switch to dark mode".
- Shows LightModeRounded in dark mode and DarkModeRounded in light mode.

**Floating AI assistant (`AiAssistant.jsx`, signed-in only):**
- **Launcher:**
  - Classes `fixed bottom-6 right-6 z-[60] h-14 w-14 rounded-full`, white, `hover:scale-105`.
  - Background `linear-gradient(135deg,#2a78d6,#9f4bd6)`, box-shadow `0 12px 32px -8px rgba(42,120,214,0.6)`.
  - Tooltip "Ask ZenReach" (placement left). aria-label "Open AI assistant".
  - Shows `ChatMark size 28` when closed and CloseRounded when open.
- **Panel:**
  - Classes `fixed bottom-24 right-6 z-[60] w-[min(92vw,392px)]`, height `min(72vh,620px)`, `rounded-3xl`.
  - Background `#1c1c1e` dark, `#fff` light.
  - Border `rgba(255,255,255,0.1)` dark, `rgba(0,0,0,0.08)` light.
  - Shadow `0 30px 80px -20px rgba(0,0,0,0.55)`.
  - Entrance animation (GSAP): from y 24, opacity 0, scale 0.96 (origin bottom right) to rest, 0.4s, `power3.out`.
- **Header:** brand gradient. A 36px white/15 circle containing ChatMark 24, then "Ask ZenReach" (15px semibold) and a subline (11px, white/80): `Powered by <provider>` when the AI is live (from `/api/assistant/status`), otherwise "Built-in analyst · demo". Then a close button.
- **Tabs:** "Chat" and "Prompts". Active tab: text and 2px bottom border in `#0071e3`, or `#59a8ff` in dark mode. Inactive text `#6e6e73`, or `#8a8a8e` in dark mode.
- **Chat:**
  - Message area background `#f5f5f7` light, `#161618` dark.
  - User bubbles: `#0071e3` (dark `#2a91ff`), white text, bottom-right radius 6.
  - Assistant bubbles: `#fff` (dark `#2c2c2e`), with a hairline border and bottom-left radius 6.
  - A mini-markdown renderer turns `**bold**` into bold and keeps line breaks.
  - An optional link button (pill, light-blue tint) navigates and closes the panel. Option chips appear only on the latest message; the `cancel` chip is grey.
  - Starter buttons appear while only the welcome message is shown: "Build me a dashboard", "Which campaign has the best ROI?", "Where should I move budget?".
  - Typing indicator: three `.zr-typing-dot`s. A minimum 1–2s random delay is applied before each reply, errors included.
  - Error reply: "Sorry — I could not reach the analyst service. Is the API running?"
  - Input: a pill TextField with placeholder "Ask about your marketing…". Enter sends; Shift+Enter does not. The round send button is blue.
  - Requests go to `POST /api/assistant {message, history, flow}`.
- **Prompts tab:** the intro "Tap a template to drop it into the chat — tweak it, then send.", then grouped templates. Tapping one fills the input and switches to Chat.
  - **Build a whole dashboard:**
    - "Build a dashboard from my data" → `Build me a dashboard`
  - **Add a single chart (during a build):**
    - "KPI — one headline number" → `Add a KPI of blended ROI`
    - "Gauge — a value against a target" → `Add a gauge of budget utilization with target 90`
    - "Line — a measure over time" → `Show revenue over time`
    - "Bar — compare across a dimension" → `Bar of planned vs actual by channel`
    - "Donut — share of a whole" → `Donut of revenue by channel`
    - "Funnel — stages by a measure" → `Funnel of revenue by channel`
    - "Table — rows of records" → `Table of campaigns`
  - **Curate:**
    - "Ask for AI-suggested charts" → `Suggest charts`
    - "Remove a chart by number" → `Remove 2`
    - "Clear and start from scratch" → `Clear all`
    - "Finish and create it" → `Build it`
  - **Ask the analyst:**
    - "Best-performing campaign" → `Which campaign has the best ROI?`
    - "Where to move budget" → `Where should I move budget?`
    - "Goal progress" → `Are we on track to hit our goals?`
- **Welcome message:** `Hi! I'm your ZenReach analyst. I can also **build a dashboard exactly the way you want** — say **"build a dashboard"**, then take my suggestions or describe the charts you want (e.g. **"add a gauge of budget utilization with target 90"**). Check the **Prompts** tab for ready-to-use templates.`

### 12.4 Theme mode (`context/ThemeModeContext.jsx`)

- **Initial mode:** `localStorage.zr_theme` if it is 'light' or 'dark'; otherwise the OS preference `(prefers-color-scheme: dark)`.
- **On every change:**
  - Toggles `html.dark`.
  - Sets `data-theme=<mode>` and `style.colorScheme=<mode>` on `<html>`.
  - Persists `zr_theme`.
- Provides `{mode, setMode, toggle}` and wraps children in MUI `ThemeProvider theme={makeTheme(mode)}` plus `CssBaseline`.
- Tailwind's `darkMode: 'class'` keys off `html.dark`. CSS variables flip under `html.dark`.

### 12.5 Design tokens

#### 12.5.1 CSS variables (`index.css`)

| Variable | Light (`:root`) | Dark (`html.dark`) |
|---|---|---|
| `--white` (page bg) | `#ffffff` | `#0b0b0d` |
| `--surface` (cards) | `#ffffff` | `#1c1c1e` |
| `--gray-bg` | `#f5f5f7` | `#161618` |
| `--ink` | `#1d1d1f` | `#f5f5f7` |
| `--ink-2` | `#6e6e73` | `#a1a1a6` |
| `--muted` | `#86868b` | `#8a8a8e` |
| `--hairline` | `#d2d2d7` | `#38383a` |
| `--blue` | `#0071e3` | `#2a91ff` |
| `--blue-link` | `#0066cc` | `#59a8ff` |
| `--card-border` | `rgba(0,0,0,0.07)` | `rgba(255,255,255,0.1)` |
| `--card-shadow` | `0 1px 2px rgba(0,0,0,0.04), 0 1px 3px rgba(0,0,0,0.05)` | `0 1px 2px rgba(0,0,0,0.4)` |
| `--frost-bg` | `rgba(255,255,255,0.8)` | `rgba(22,22,24,0.72)` |
| `--frost-bg-solid` | `rgba(255,255,255,0.96)` | `rgba(18,18,20,0.96)` |
| `--chip` | `rgba(0,0,0,0.05)` | `rgba(255,255,255,0.09)` |
| `--scrollbar-thumb` | `#c7c7cc` | `#48484a` |
| `--scrollbar-border` | `#fff` | `#1c1c1e` |
| `--ease-out` | `cubic-bezier(0.22, 1, 0.36, 1)` | same |
| `--ease-spring` | `cubic-bezier(0.34, 1.4, 0.64, 1)` | same |
| `--dur` | `0.24s` | same |
| `--radius-card` | `16px` | same |
| `--shadow-sm` | `0 1px 2px rgba(0,0,0,0.04), 0 1px 3px rgba(0,0,0,0.05)` | `0 1px 2px rgba(0,0,0,0.5)` |
| `--shadow-md` | `0 10px 24px -10px rgba(0,0,0,0.16), 0 3px 8px -4px rgba(0,0,0,0.09)` | `0 12px 28px -10px rgba(0,0,0,0.62), 0 4px 10px -6px rgba(0,0,0,0.5)` |
| `--shadow-lg` | `0 28px 60px -18px rgba(0,0,0,0.26), 0 10px 24px -10px rgba(0,0,0,0.14)` | `0 30px 66px -18px rgba(0,0,0,0.8), 0 12px 26px -10px rgba(0,0,0,0.6)` |
| `--spot` (spotlight tint) | `rgba(0,113,227,0.10)` | `rgba(90,168,255,0.14)` |
| `--accent-2` | `#9f4bd6` | `#c07bef` |
| `color-scheme` | light | dark |

#### 12.5.2 Tailwind (`tailwind.config.js`)
- `content: ['./index.html','./src/**/*.{js,jsx}']`, `darkMode:'class'`.
- **`corePlugins.preflight: false`**, so Tailwind's reset doesn't fight MUI.
- Extended colours map to the CSS variables: `ink`, `ink2`, `muted`, `hairline`, `graybg`, `surface`, `blue`.
- `fontFamily.sans`: `-apple-system, BlinkMacSystemFont, 'SF Pro Display', Inter, system-ui, sans-serif`.
- `maxWidth.content: '1120px'`. This is the standard page width (`max-w-content`, plus `px-5` gutters).
- No plugins.
- `postcss.config.js`: tailwindcss and autoprefixer.

#### 12.5.3 Base layer and global CSS
- Under `@layer base`: `a {color: inherit; text-decoration: none}`, `button {font-family: inherit}`, `* {box-sizing: border-box}`. These are needed because preflight is off.
- `html`: antialiased font smoothing (webkit and moz) and `scroll-behavior: smooth`.
- `body`:
  - `margin 0`, background `var(--white)`, colour `var(--ink)`.
  - Font: `-apple-system, BlinkMacSystemFont, 'SF Pro Display', 'SF Pro Text', 'Inter', system-ui, 'Segoe UI', sans-serif`.
  - **letter-spacing -0.01em**.
  - Transitions background-color and color over 0.3s ease.
- `#root { min-height: 100% }`.

#### 12.5.4 Typography
- **Font:** San Francisco on Apple devices, Inter (Google Fonts) elsewhere.
- **CSS scale classes:**
  - `.display`: weight 700, letter-spacing -0.025em, line-height 1.05.
  - `.headline`: weight 700, -0.02em, line-height 1.08.
  - `.eyebrow`: weight 600, letter-spacing 0.01em.
- **`.grad-text`:** `linear-gradient(90deg, #2a78d6 0%, #9f4bd6 35%, #eb6834 70%, #eda100 100%)` clipped to the text, with transparent colour.
- **MUI typography:**
  - fontFamily `-apple-system, BlinkMacSystemFont, "SF Pro Display", Inter, system-ui, sans-serif`.
  - h1 700 / -0.025em; h2 700 / -0.022em; h3 700 / -0.02em; h4 700 / -0.02em; h5 600 / -0.015em; h6 600 / -0.01em.
  - button: `textTransform none`, weight 500.
- **Recurring sizes in components:**
  - Page title 22px, 26px on `sm`.
  - Eyebrow 12px uppercase.
  - Body copy 13–14px. Captions 11–12.5px.
  - KPI values 44px at weight 800.
  - Section heading 32px, 44px on `sm`.
  - Values use `tabular-nums` throughout.

#### 12.5.5 Radii
- Cards: 16px (`--radius-card`, MuiCard 16).
- MUI `shape.borderRadius`: 12.
- Paper rounded: 14. Dialog: 20. Menu and Popover: 14. MenuItem: 9.
- Button, OutlinedInput and ToggleButton: 10. Chip: 8. Tooltip: 9.
- `.btn-pill`: 10px. Builder and Dashboards buttons use 999 (pill).
- Empty-state boxes: 22px. Assistant panel: `rounded-3xl` (24px).
- ECharts tooltip: 14px.
- **MUI `sx` numeric borderRadius multiplies by 12.** So `borderRadius:3` is 36px, `1` is 12px, `1.5` is 18px and `1.25` is 15px.

#### 12.5.6 Shadows
- CSS: `--shadow-sm`, `--shadow-md` and `--shadow-lg` (table above).
- **MUI elevation ramp** from `makeShadows(dark)`: 25 entries, index 0 'none'. For i = 1–24:
  - `y1 = round(i·0.6)+1`, `b1 = round(i·1.7)+3`, `y2 = round(i·1.5)+2`, `b2 = round(i·3.4)+8`.
  - Light: `a1 = min(0.16, 0.05+i·0.005)`, `a2 = min(0.2, 0.08+i·0.006)`.
  - Dark: `a1 = min(0.6, 0.3+i·0.005)`, `a2 = min(0.7, 0.42+i·0.006)`.
  - Each entry is `0 y1px b1px -round(b1/2)px rgba(0,0,0,a1), 0 y2px b2px -round(b2/2)px rgba(0,0,0,a2)`.
  - Examples:
    - Light level 1: `0 2px 5px -3px rgba(0,0,0,0.055), 0 4px 11px -6px rgba(0,0,0,0.086)`.
    - Light level 8: `0 6px 17px -9px rgba(0,0,0,0.090), 0 14px 35px -18px rgba(0,0,0,0.128)`.
    - Light level 24: `0 15px 44px -22px rgba(0,0,0,0.160), 0 38px 90px -45px rgba(0,0,0,0.200)`.
    - Dark level 1: `… rgba(0,0,0,0.305) …, … rgba(0,0,0,0.426)`.
- **Menu shadow:**
  - Light: `0 18px 44px -16px rgba(15,23,42,0.24), 0 6px 14px -8px rgba(15,23,42,0.12)`.
  - Dark: `0 18px 44px -16px rgba(0,0,0,0.7), 0 6px 14px -8px rgba(0,0,0,0.5)`.
- **Dialog shadow:**
  - Light: `0 44px 100px -28px rgba(15,23,42,0.34), 0 14px 30px -12px rgba(15,23,42,0.16)`.
  - Dark: `0 44px 100px -28px rgba(0,0,0,0.85), 0 14px 30px -12px rgba(0,0,0,0.6)`.

#### 12.5.7 Component and utility classes (`index.css`)
- **`.card`:** `var(--surface)` background, `var(--radius-card)` radius, `1px solid var(--card-border)` border, `var(--shadow-sm)` shadow. Transitions transform, border-color, box-shadow (`var(--dur) var(--ease-out)`) and background-color (0.25s).
- **`.card-hover`:** `position: relative`, plus a `::before` spotlight layer.
  - The layer is `radial-gradient(260px circle at var(--mx,50%) var(--my,0%), var(--spot), transparent 72%)`, opacity 0, pointer-events none, z-index 0.
  - Children get `position: relative; z-index: 1`.
  - Hover: `translateY(-3px)`, border `color-mix(in srgb, var(--blue) 22%, var(--card-border))`, shadow `--shadow-md`, glow opacity 1.
  - Active: `translateY(-1px)`.
- **`.frost`:** `var(--frost-bg)` with `backdrop-filter: saturate(180%) blur(20px)` (plus the -webkit- version). Falls back to `--frost-bg-solid` when backdrop-filter is unsupported.
- **`.frost-dark`:** `rgba(22,22,24,0.6)` with the same blur.
- **`.btn-pill`:** radius 10px, padding 8px 16px, weight 500, 13.5px, no border, pointer. Transitions background, border-color, shadow and transform (`0.12s var(--ease-spring)`). Active: `scale(0.96)`.
- **`.btn-blue`:** background `linear-gradient(180deg, #1a86ff 0%, var(--blue) 100%)`, white text, shadow `0 1px 2px rgba(0,0,0,0.14), 0 6px 16px -6px rgba(0,113,227,0.5)`. Hover: gradient `#2a91ff → #0077ed`, shadow `0 2px 4px rgba(0,0,0,0.16), 0 10px 22px -6px rgba(0,113,227,0.55)`, `translateY(-1px)`.
- **`.btn-ghost`:** `rgba(0,0,0,0.045)` background, ink text, `inset 0 0 0 1px rgba(0,0,0,0.02)`. Hover: `rgba(0,0,0,0.08)` and -1px. Dark: `rgba(255,255,255,0.08)`, hover `0.14`.
- **`.link-arrow`:** `var(--blue-link)`, weight 400, underline on hover.
- **`.zr-typing-dot`:** 7px circle, `currentColor`, animation `zr-typing 1.25s infinite ease-in-out`. The 2nd and 3rd dots are delayed 0.16s and 0.32s. Keyframes: at 0, 70 and 100% `translateY(0)` opacity 0.32; at 35% `translateY(-4px)` opacity 1. Reduced motion uses an opacity-only pulse (0.35 → 1 → 0.35).
- **Scrollbars (webkit):** 10px. Thumb `var(--scrollbar-thumb)`, radius 8, `3px solid var(--scrollbar-border)`. Transparent track.
- **`.board-scroll`:** horizontal kanban scroller with CSS scroll-shadows (local and scroll background layers; 24px covers, 28px shadows at 0.08 alpha, 0.5 in dark). Thin scrollbar, smooth scrolling, 8px webkit scrollbar height.
- **`.no-scrollbar`:** hides scrollbars in Firefox and WebKit.
- **`.h9-grid`:** dense table styling. Even rows `rgba(0,0,0,0.015)` (dark `rgba(255,255,255,0.022)`). Row hover `rgba(0,113,227,0.06)`.
- **`.aurora`:** hero background of four radial gradients, then `blur(20px)`:
  - blue `rgba(42,120,214,0.55)` at 20%/20%
  - violet `rgba(159,75,214,0.5)` at 80%/25%
  - orange `rgba(235,104,52,0.45)` at 75%/80%
  - aqua `rgba(27,175,122,0.4)` at 25%/85%
- **Focus:** `:where(a, button, input, select, textarea, summary, [role=button], [tabindex]):focus-visible` gets `outline: 2px solid var(--blue); outline-offset: 2px; border-radius: 8px`. Mouse focus on a, button and [role=button] has no outline.
- **`.zr-page-enter`:** keyframes from opacity 0 / `translateY(6px)` to the resting state, 0.28s `cubic-bezier(0.22,1,0.36,1)`.
- **`.glass`:** `color-mix(in srgb, var(--surface) 72%, transparent)` with `saturate(180%) blur(16px)` and a card-border.
- **`.hover-lift`:** hover `translateY(-2px)` with `--shadow-md`.
- **`.shine`:** an animated conic border sheen.
  - Uses `@property --shine-angle` (angle type, initial value 0deg).
  - `::after` sits at inset -1px with padding 1px and `conic-gradient(from var(--shine-angle), transparent 0deg, color-mix(var(--blue) 60%) 90deg, var(--accent-2) 140deg, transparent 200deg)`.
  - It is masked to the border (xor/exclude), opacity 0.7, with `zr-shine 6s linear infinite`.
- **Reduced motion** (`prefers-reduced-motion: reduce`) disables:
  - smooth scrolling and the page-enter animation;
  - the press scale and hover transforms on the pill, blue and ghost buttons;
  - the card-hover transform and the spotlight layer (`display: none`);
  - the sheen animation.
- **Print:** everything is hidden except `.calendar-print`, which is positioned absolutely at the top-left at full width.

#### 12.5.8 Chart palette and status colours (`theme.js`)
- **`SERIES`**, a categorical palette shared by both modes, in order:
  1. `#2a78d6` blue
  2. `#1baf7a` aqua
  3. `#e87ba4` magenta
  4. `#eda100` yellow
  5. `#4a3aa7` violet
  6. `#eb6834` orange
  7. `#008300` green
  8. `#e34948` red
- **`STATUS`:** good `#0ca30c`, warning `#fab219`, serious `#ec835a`, critical `#d03b3b`.
- **Builder threshold colours** (separate from STATUS): GOOD `#12a150`, BAD `#e0403f`.
- **Brand gradient:** `linear-gradient(135deg,#2a78d6,#9f4bd6)`. Used by the logo, the favicon, avatars, the assistant launcher and header, and the empty-state tiles.
- **Builder accent blue:** `#0071e3`. Used for selected states, and is the same as the light `--blue`.

### 12.6 MUI theme (`theme.js` `makeTheme(mode)`)

**Palettes:**

| | Light | Dark |
|---|---|---|
| background.default | `#ffffff` | `#0b0b0d` |
| background.paper | `#ffffff` | `#1c1c1e` |
| primary.main | `#0071e3` | `#2a91ff` |
| secondary.main | `#9f4bd6` | `#c07bef` |
| success.main | `#0ca30c` | `#30d158` |
| warning.main | `#fab219` | `#ffd426` |
| error.main | `#d03b3b` | `#ff5a5a` |
| text.primary | `#1d1d1f` | `#f5f5f7` |
| text.secondary | `#6e6e73` | `#a1a1a6` |
| divider | `rgba(0,0,0,0.08)` | `rgba(255,255,255,0.12)` |

**Theme-level settings:**
- Shared: `shape.borderRadius 12` and the typography described in B5.
- `shadows: makeShadows(isDark)`.
- `transitions.easing.easeOut = 'cubic-bezier(0.22, 1, 0.36, 1)'`.
- Overlay colour (backdrop): `rgba(15,23,42,0.30)` light, `rgba(0,0,0,0.62)` dark.

**Component overrides:**
- **MuiPaper:** root `backgroundImage:'none'`; rounded `borderRadius 14`.
- **MuiCard:** `borderRadius 16`.
- **MuiDialog** paper: `borderRadius 20`, `boxShadow` = the dialog shadow, `backgroundImage none`.
- **MuiBackdrop:** root `backgroundColor` = the overlay, with `backdropFilter: blur(3px)` (plus the -webkit- version). The invisible variant is transparent with no filter.
- **MuiMenu:** paper `borderRadius 14`, `marginTop 6`, `boxShadow` = the menu shadow, `border 1px solid var(--card-border)`. List `padding 6`.
- **MuiMenuItem:** `borderRadius 9`, `marginBottom 1`, `minHeight 38`.
- **MuiPopover** paper: `borderRadius 14`, the menu shadow.
- **MuiButton:**
  - `defaultProps.disableElevation: true`.
  - root: `borderRadius 10`, transition `background .18s ease, box-shadow .24s cubic-bezier(0.22,1,0.36,1), transform .12s cubic-bezier(0.34,1.4,0.64,1)`.
  - contained: `boxShadow 0 1px 2px rgba(0,0,0,0.12), 0 6px 16px -8px rgba(0,113,227,0.45)`. Hover: `0 2px 4px rgba(0,0,0,0.14), 0 10px 22px -8px rgba(0,113,227,0.5)` with `translateY(-1px)`. Active: `translateY(0)`.
  - Many call sites override this with `boxShadow:'none'`.
- **MuiToggleButton:** `borderRadius 10`.
- **MuiChip:** `borderRadius 8`, `fontWeight 500`.
- **MuiOutlinedInput:** `borderRadius 10`.
- **MuiTooltip:** `fontSize 12`, `borderRadius 9`, `padding '6px 10px'`, `backdropFilter blur(6px)`, `boxShadow` = the menu shadow. Background `rgba(29,29,31,0.92)` light, `rgba(60,60,63,0.96)` dark.

The default export is `makeTheme('light')`, kept for legacy imports.

### 12.7 Dependencies

**Client** (`client/package.json`: name `zenreach-client`, version 1.0.0, private, `"type": "module"`). Scripts: `dev: vite`, `build: vite build`, `preview: vite preview`.

| Package | Declared | Installed |
|---|---|---|
| @emotion/react | ^11.11.4 | 11.14.0 |
| @emotion/styled | ^11.11.5 | 11.14.1 |
| @mui/icons-material | ^5.15.20 | 5.18.0 |
| @mui/material | ^5.15.20 | 5.18.0 |
| echarts | ^6.1.0 | 6.1.0 |
| gsap | ^3.12.5 | 3.15.0 |
| html-to-image | ^1.11.13 | 1.11.13 |
| react | ^18.3.1 | 18.3.1 |
| react-dom | ^18.3.1 | 18.3.1 |
| react-router-dom | ^6.24.0 | 6.30.4 |
| dev: @vitejs/plugin-react | ^4.3.1 | 4.7.0 |
| dev: autoprefixer | ^10.4.19 | 10.5.4 |
| dev: postcss | ^8.4.38 | 8.5.19 |
| dev: tailwindcss | ^3.4.4 | 3.4.19 |
| dev: vite | ^5.3.1 | 5.4.21 |

**Server, for context:**
- express ^4.19.2, cors ^2.8.5, morgan ^1.10.0, dotenv ^16.4.5
- xlsx ^0.18.5 (spreadsheet upload conversion)
- @anthropic-ai/sdk ^0.68.0, openai ^6.48.0
- @google-analytics/data ^4.7.0

**Root:** devDependency concurrently ^8.2.2.

### 12.8 Glossary (`client/src/lib/glossary.js`)

- Exported as `GLOSSARY`, a plain object. Pages import it as `GLOSSARY as G` and pass `G.<key>` to `<InfoTip text=…/>`.
- The file says definitions should stay at one or two sentences, because they render in hover tooltips.
- **Duplicate key:** `campaignLevel` is defined twice. The later definition wins at runtime (shown below; the overridden first value is also listed).
- These are the exact strings in the current working tree. Planning entries added since the last commit are included.

**Money / return**
- `roi`: Return on investment = revenue ÷ spend. "3x" means every $1 spent brought back $3. Higher is better; below 1x means the activity lost money.
- `blendedRoi`: ROI across all channels combined (total revenue ÷ total spend), rather than for a single campaign — the headline efficiency of your whole marketing mix.
- `pipeline`: The total value of sales opportunities created by marketing that could still close — future revenue in progress, not money in the bank yet.
- `attributedRevenue`: Revenue credited to marketing — the closed sales that marketing activity helped generate, as opposed to total company revenue.
- `creditedRevenue`: The share of revenue an attribution model assigns to a channel for its part in the customer journey.
- `marketingSpend`: The money actually spent running marketing (media, tools, events, agencies) over the period.
- `projectedRevenue`: An estimate of the revenue a given budget allocation would generate, based on each channel’s past efficiency.
- `baseline`: The starting point you’re comparing against — here, the revenue expected from your current, unchanged budget.
- `lift`: The percentage improvement over the baseline. A positive lift means the new plan is projected to beat the current one.

**Leads / conversion**
- `leads`: People who showed interest (e.g. filled a form or requested a demo) — potential customers marketing captured.
- `mql`: A lead that fits your ideal customer and is engaged enough for sales to look at — a "Marketing Qualified Lead".
- `conversions`: The count of visitors who completed a goal action (sign-up, purchase, demo request) rather than just browsing.
- `conversionRate`: The share of people who took the desired action out of everyone who had the chance — e.g. 3 of 100 visitors = 3%.

**Web analytics (GA4)**
- `sessions`: A single visit to your site — one person browsing counts as one session (it ends after ~30 minutes of inactivity).
- `users`: The number of distinct people who visited, however many times each — so users is usually lower than sessions.
- `engagement`: Average time visitors actively spent per session. Longer engagement usually signals more relevant traffic.
- `activeUsersNow`: People on your site in the last few minutes — a live count from GA4 Realtime, with no processing delay.
- `trafficByChannel`: Where your visits came from (paid search, social, organic, email, etc.), so you can see which sources drive traffic.
- `topPages`: Your most-viewed pages in the period — useful for spotting what content attracts the most attention.

**Budget / finance**
- `topDownBudget`: The total money leadership sets aside for marketing this period, handed down as a target before teams decide exactly how to use it. It’s the ceiling everything is measured against.
- `committed`: The part of the budget already assigned to specific campaigns and tactics (whether or not it’s been paid yet) — how much of the plan is spoken for.
- `uncommitted`: Budget still free to allocate = budget − committed. Positive means room left to invest; negative means over-committed.
- `utilization`: How much of the budget is used up, as a percentage. Near 100% means fully allocated/spent.
- `planned`: The amount you intended to spend — the budget set aside, before comparing it to what actually happened.
- `actual`: The money actually spent to date — compared against what was planned.
- `remaining`: Planned budget minus what’s been spent so far — what’s still left to spend.
- `plannedBudget`: The total budget set aside across all the tactics shown.

**Planning / work management**
- `hierarchy`: How marketing work is organized — five levels, created in order: Plan → Campaign → Program → Tactic → Line item. The plan is one fiscal year; tactics are the things you do; line items are where the money is planned.
- `planLevel`: One fiscal year of work aimed at a goal. Holds the budget, the goals, the teams and the model. Plan names are unique across the whole system.
- `campaignLevel` (first definition, overridden): A theme or overarching message inside a plan — usually runs most of the year.
- `programLevel`: A group of similar activities inside a campaign (e.g. Social Media, Demand Creation). Its dates must sit inside the campaign’s and its budget can’t exceed the campaign’s.
- `tacticLineItem`: A cost line under a tactic — the only place spend is planned and tracked. Its total is spread across months or quarters and can be linked to a budget.
- `plannedCost`: What a tactic is expected to cost. Tactics have planned cost, never a budget — the line items underneath carry the detail.
- `planBudgetField`: A simple marketing-only budget on a plan, campaign or program. Children’s budgets are deducted from the parent’s; what is left is unallocated.
- `unallocatedBudget`: This level’s plan budget minus the budgets handed down to its children.
- `sysGenBalance`: System Generated Balance — the tactic’s planned cost minus the sum of its line items. If you use line items you want this to be zero.
- `spreadMechanism`: How a line item’s total is spread over time: C = Custom (you type each period), F = First period, L = Last period, E = Evenly across the tactic’s duration.
- `approvalLifecycle`: Created → Submitted → Approved or Declined. An approved tactic reads In-Progress once its start date arrives and Complete once its end date passes.
- `evergreenTactic`: An ongoing tactic with no end date. Once saved as evergreen it can’t be undone — close it and create a new tactic with real dates instead.
- `waterfallGoals`: Enter the inquiries a tactic should generate; the plan’s model cascades them through its conversion rates (MQL → TQL → opportunity → won) and prices won deals at its average deal size.
- `planModel`: The rulebook a plan is measured by: waterfall conversion rates, average deal size, which tactic types are allowed, and line-item money rules.
- `strategicPlan`: A high-level Strategic / AOP (annual operating plan). Shown in grey and can be filtered separately from working plans.
- `dateModeAuto`: Auto: the dates are calculated from the tactics underneath. Manual: you type them in.
- `tactic`: The smallest unit of work you actually execute — a specific campaign, launch, event or piece of content, sitting at the bottom of the plan hierarchy.
- `campaignLevel` (**effective value**): A group of related programs working toward one goal (e.g. "Demand Generation") — one level up from the day-to-day tactics.
- `program`: A workstream inside a campaign that holds tactics (e.g. "Paid Acquisition", "Events & Webinars").
- `inFlight`: Tactics currently being worked on (status "in progress") — what’s live right now.
- `launched`: Tactics that have shipped or gone live — completed and out in the world.
- `priorityP0`: How important a tactic is: P0 = must-do / top priority, P1 = important, P2 = nice-to-have.
- `impact`: How much value a tactic is expected to create, rated 1–5 (5 = highest). Used to prioritize what to work on.
- `effort`: How much work a tactic takes, rated 1–5 (5 = most). High-impact, low-effort work is usually done first.
- `progress`: How far along a tactic is, from 0% (not started) to 100% (done).
- `roadmap`: A quarter-by-quarter view of when work is planned across the year.

**Attribution**
- `attribution`: Deciding which marketing touchpoints get credit for a sale when a customer interacted with several before buying.
- `multiTouch`: Attribution that splits credit across every touchpoint in the journey, instead of giving it all to the first or last one.
- `attributionModel`: A rule for sharing credit — e.g. first-touch, last-touch, linear (even split), or time-decay (recent touches get more).
- `waterfallFunnel`: The stages a prospect passes through (visitor → lead → opportunity → customer) and how many drop off at each step.
- `reallocation`: A suggestion to move budget from lower-return channels to higher-return ones to improve overall results.
- `forecast`: A projection of where a metric will land by period-end if current trends continue.
- `stretchTarget`: An ambitious goal set above the expected result to push for more — you’re "on track" even if you land a bit under it.

**Goals**
- `target`: The number you’re aiming to hit for this metric by the end of the period.
- `attainment`: How much of the target you’ve achieved so far, as a percentage.
- `elapsed`: How much of the time period has already passed — useful context for whether attainment is on pace.
- `goalForecast`: Where this metric is projected to finish the period, so you can tell early if it’s on-track, at-risk or off-track.

**Procurement**
- `purchaseRequest`: A formal request to spend money with a vendor, routed for approval before any commitment is made.
- `vendor`: The outside company or supplier you’re paying (e.g. an ad platform, agency or software provider).
- `approvalChain`: The ordered list of people who must sign off before a purchase can proceed — bigger amounts may need more approvers.
- `lineItems`: The individual things being bought within a request, each with a quantity and unit price that add up to the total.
- `purchaseOrder`: A PO — the official document issued to a vendor to confirm an approved order and its agreed price.
- `invoice`: The bill the vendor sends for goods or services delivered, which you then pay.
- `approvedValue`: The total money across requests that have cleared approval — spend that’s been officially green-lit.

**Optimizer**
- `efficiency`: Revenue generated per $1 of spend on a channel. A 3x efficiency means $1 in returns about $3 — used to decide where extra budget goes furthest.

**Integrations**
- `syncDirection`: Which way data flows with a connected app: inbound (into ZenReach), outbound (out to the app), or bidirectional (both ways).
- `connectMethod`: How the connection works: API (a live automatic link) or File (data brought in via upload/export).

**Usage notes:**
- Keys referenced by pages include roi, pipeline, sysGenBalance (used 6 times), plannedCost, planBudgetField, approvalChain, utilization, sessions, users, leads and engagement, among others.
- Never referenced via `G.`: attributedRevenue, marketingSpend, mql, conversionRate, topDownBudget, committed, uncommitted, planned, actual, remaining, impact, effort, roadmap, attribution, attributionModel, purchaseOrder, invoice, blendedRoi.

### 12.9 Shared components

**`PageHeader({eyebrow, title, highlight, subtitle, actions, currency=true})`**
- A compact, left-aligned workbench header. Container: `mx-auto max-w-content px-5 pt-8 pb-5`. Row: `flex flex-wrap items-end justify-between gap-3`.
- Left side:
  - Optional eyebrow: 12px semibold uppercase `tracking-wide text-blue`, mb-1.
  - `h1`: 22px (26px on `sm`) bold, leading-tight, `tracking-[-0.02em]`, ink. It renders `{title} {highlight}`, where the highlight is a `text-ink2 font-semibold` span.
  - Optional subtitle: mt-1.5, `max-w-3xl`, 13.5px, leading-snug, ink2.
- Right side (when actions or currency are present): `flex flex-none items-center gap-2`, with `{actions}` and then `<CurrencyControl/>`. `currency={false}` hides the picker.

**`Panel({title, subtitle, action, children, className, bodyClassName, hover=false})`**
- `<section class="card [card-hover] {className}">`.
- If a title or action is given, a header row: `flex flex-wrap items-start justify-between gap-3 border-b border-hairline px-5 pt-3.5 pb-3`.
  - Title: MUI Typography 600, 15px, -0.01em, line-height 1.2.
  - Subtitle: text.secondary, 12.5px, mt 0.2.
  - `action` sits on the right.
- Body: `px-5 pb-5`, plus `pt-4` if there is a title, otherwise `pt-5`, plus `bodyClassName`.

**`StateWrap({loading, error, children, minHeight=240})`**
- Loading: a centered MUI `CircularProgress size 28` in a grid box of `minHeight`.
- Error: `Alert severity="error" variant="outlined"` with radius 3 (36px): `Could not load data: {error.message}. Is the API running on port 4000?`
- Otherwise it renders the children.

**`ScrollReveal({children, className, y=40, stagger=0.1, duration=0.9, once=true, as='div'})`**
- A deterministic scroll reveal built on IntersectionObserver and CSS transitions. It deliberately avoids GSAP ScrollTrigger, which froze under StrictMode.
- **Clamps:** y ≤ 8px, duration ≤ 0.4s, stagger ≤ 0.04s (0 if falsy).
- **Targets:** the element's direct children when there is a stagger, otherwise the element itself.
- **Initial state:** opacity 0, `translateY(y)`, transition `opacity/transform {duration}s cubic-bezier(0.22,1,0.36,1) {i·stagger}s`, will-change.
- **Reveal:** opacity 1, transform none.
- **Observer:** threshold 0.01, rootMargin `0px 0px -8% 0px`. With `once` it disconnects after revealing; otherwise it re-hides on exit.
- Also reveals on the next animation frame if the element's top is already within 92% of the viewport height.
- Reduced motion: no animation.
- Cleanup removes the inline styles, so nothing is left stranded.
- Used by most analytics-style pages.

**`Reveal({children, className, stagger=0.08, y=18, delay=0})`**
- GSAP `gsap.from(children, {y, opacity:0, duration:0.6, ease:'power3.out', stagger, delay})` inside a `gsap.context`, reverted on cleanup.
- Currently **unused** anywhere.

**`InfoTip({text, size=14, color='var(--muted)'})`**
- An `InfoOutlined` icon at `size` px, `cursor: help`, vertical-align middle, ml 0.4, colour transition 0.15s. Hover turns it `var(--blue)`. `tabIndex 0`. `aria-label` is the text when the text is a string, otherwise "More info".
- Wrapped in an MUI Tooltip: `arrow`, `enterTouchDelay 0`, `leaveTouchDelay 4000`.
- Tooltip styling: 12px, line-height 1.45, maxWidth 260, px 1.25, py 0.9, background `rgba(29,29,31,0.94)`, radius 1 (12px). The arrow uses the same colour.

**`BrandLogo({size=28, radius=8})`**
- A `size`×`size` span with the given radius, the brand gradient `linear-gradient(135deg,#2a78d6,#9f4bd6)`, white glyph, aria-hidden.
- The SVG is `round(size·0.64)` px with viewBox 24. It contains:
  - a reach ring: circle (18,7) r 4.4, stroke 1.3, opacity 0.4;
  - a rising trend line: `M4 17.5 L9.5 12 L12.8 15 L16.4 8.6`, stroke 2.3, round caps and joins;
  - a reach node: circle (18,7) r 2.4, filled.
- Used in the sidebar and top-bar BrandMark.

**`ChatMark({size=26, accent='#2a78d6'})`**
- The assistant mark. SVG, viewBox 24, aria-hidden.
- A speech bubble with a bottom-left tail, filled `currentColor` (white in the launcher). Path: `M6.5 3.5h11A3 3 0 0 1 20.5 6.5v6A3 3 0 0 1 17.5 15.5H11l-3.7 3.1a.7.7 0 0 1-1.15-.54V15.5H6.5A3 3 0 0 1 3.5 12.5v-6A3 3 0 0 1 6.5 3.5Z`.
- Inside it, a trend line `M7.3 12 L10 9.4 L11.7 11 L14.7 7.9` in the accent colour (stroke 1.7) and a dot (15.4,7.4) r 1.5.

**`SectionHeading({eyebrow, title, highlight, subtitle, align='center'})`**
- An Apple-style marketing heading, used only on Home.
- Wrapper: `max-w-3xl`, with `mx-auto text-center` (or `text-left`).
- Eyebrow: `.eyebrow mb-2`, 14px semibold, colour `#0071e3`.
- `h2.headline`: 32px (44px on `sm`). The `highlight` gets `.grad-text`.
- Subtitle: mt-4, 18px (20px on `sm`), leading-relaxed, ink2.

**`ScrollToTop`:** see §12.2.

**`ThemeToggle`, `CurrencyControl`, `NotificationBell`, `AccountControls`, `Avatar`, `AiAssistant`:** see §12.3.

### 12.10 Client-side persistence keys and global events

**localStorage:**
- `zr_token`: auth bearer token.
- `zr_theme`: 'light' or 'dark'.
- `zr_currency`: currency code.
- `zr_sidebar_collapsed`: '1' or '0'.
- `zenreach.snapshots.v1`: dashboard PNG snapshots.

**Window events:**
- `zr:open-auth` (detail `{tab}`): opens the auth modal.
- `zr:auth-modal` (detail `{open}`): broadcast when the modal opens or closes. The Hero pauses its aurora animation while the modal is open.

**CSS custom properties set from JS:** `--mx` and `--my` on `.card-hover` elements (the cursor spotlight).

### 12.11 Shell quirks worth deciding on

- `/admin` has no route-level guard. Only the nav link is hidden, and the page shows its own no-access message.
- The signed-in route table has no 404 or catch-all.
- The account controls appear in the mobile drawer footer only; the mobile top bar shows only the bell and theme toggle.
- The duplicate `campaignLevel` glossary key means the newer "theme or overarching message" definition is silently overridden.
- The `#111` target-line colour in charts and the hard-coded light hexes (`#0071e3`, `#c0392b`, `#c7c7cc`) in the Builder do not adapt to dark mode.
- Viewer read-only rules are enforced inconsistently between client and server.

## 13. Non-functional requirements

Stated and observed requirements for storage, performance, security, accessibility, responsiveness, theming, resilience and testing.

### 13.1 Stated non-functional requirements

**Storage and persistence**
- No database by design: a synchronous JSON file-store (`store.js`).
  - Reads degrade to a fallback on any error.
  - Writes are pretty-printed whole files.
  - IDs are `prefix-<base36 time>-<5 random>`.
  - `ZR_DATA_DIR` can isolate an instance's data.
- `server/data/` is gitignored machine-local state.
- The parity plan flags it as not concurrent-safe. N parallel writes race, which is why bulk delete does one read-modify-write ("safe for thousands").
- Re-seeding means deleting only `plans`, `tactics`, `budgets` and `purchaseRequests`. Never delete `users`, `teams`, `dashboards` or `campaignDataset`.

**Performance**
- AI analysis cache: 10 minutes (fallback 30 seconds). Retries on 429/5xx.
- FX cache: hourly; `?refresh=1` forces a refresh.
- GA Realtime refreshes about every 15 seconds.
- Plan narration times out after 9 seconds.
- The dataset is always pre-aggregated and never served raw; the baked-in file has 200k rows.
- `express.json` limit: 60 MB.
- Client bundle: about 1.48 MB, with ECharts tree-shaken.
- The assistant deliberately waits 1–2 seconds minimum before replying.

**Security**
- Passwords are salted and scrypt-hashed, and never returned to clients.
- Sessions are opaque bearer tokens.
- RBAC (role-based access control) is enforced server-side.
- Approvals are locked to the assigned approver (403 or a thrown error otherwise).
- Notifications are resolved from the caller's own token, not a query parameter (a spoofing fix).
- All external calls are server-side. Trello and Jira credentials only pass through, never stored; only the Jira site and email are remembered in the browser.
- Secrets live in `.env`, which is gitignored along with key files.
- **Target:** WAF, private endpoints, Key Vault, TLS everywhere, SSO on the roadmap.

**Accessibility**
- Colorblind-safe categorical palette.
- `prefers-reduced-motion` respected (scroll reveals, typing dots, page transitions).
- A `:focus-visible` brand ring on interactive elements.
- Esc and tap-outside close menus and drawers.
- InfoTips work on hover, focus and touch.
- **A full WCAG 2.1 AA audit is roadmap only.**

**Responsiveness**
- "Fully responsive" across phone, tablet and laptop, with a 390×844 mobile capture.
- Mobile drawer navigation.
- The board scrolls horizontally rather than wrapping.

**Theming**
- Light (Apple-inspired) and dark, following the OS on first visit, then remembered in `zr_theme`.
- Driven by CSS variables (`--ink`, `--ink-2`, `--muted`, `--hairline`, `--surface`, `--gray-bg`, `--blue`, …).
- Tokens: ink `#1d1d1f`, secondary `#6e6e73`, muted `#86868b`, section gray `#f5f5f7`, Apple blue `#0071e3`.
- Brand gradient: `linear-gradient(135deg,#2a78d6,#9f4bd6)`.
- Font stack: SF / -apple-system, falling back to Inter.
- Cards: radius 12, faint shadow. Grids use `.h9-grid` zebra stripes with hover.

**Resilience**
- GA falls back to demo data.
- The AI provider cascade; deterministic engines everywhere.
- Audit logging is wrapped in try/catch so it never breaks a request.

**Testing**
- "No automated tests in the repo." Verification was node smoke tests, a clean `vite build`, and headless Chrome or Brave screenshots via puppeteer-core.
- Brave renders ECharts horizontal bars fine; the old headless-Chrome quirk was Recharts-only.

**localStorage keys**
- `zr_token`, `zr_theme`, `zr_sidebar_collapsed`, `zr_currency`, `zr_planning_view`, `zr_planning_views`, `zenreach.snapshots.v1`.

**Hive9 NFRs, for contrast**
- Multi-tenant SaaS, roughly 99.9% SLA.
- About a 24-hour Salesforce sync.
- Steep learning curve.
- $25k+/yr.

## 14. Demo story, roadmap and Hive9 parity

The scripted "Nova launch" demonstration the product is built around, the documented roadmap, and how ZenReach maps to Hive9 capabilities.

### 14.1 The Nova launch demo story

- **Arc:** Plan → Fund → Run → See → Prove → Optimize → Build → Ask.
- **Length:** about 13 minutes of core content, about 14:30 total.
- **Premise:** a company launching "Nova" (GA in Q3 2026); the VP has one quarter to make it land.

**Spine data: 8 campaigns from `mockData.js`**

| Campaign | Channel | Region | Status | Budget | Spent | Leads | Pipeline | Revenue |
|---|---|---|---|---|---|---|---|---|
| Nova GA Launch | Paid Search | NA | Active | 420K | 312K | 2,450 | 2.18M | 1.09M |
| Nova Enterprise ABM | Content Syndication | EMEA | Active | 360K | 268K | 640 | 3.05M | 1.525M |
| Nova Launch Webinar Series | Events & Webinars | NA | Active | 180K | 142K | 1,850 | 1.18M | 560K |
| Nova Demand Gen | Paid Search | APAC | Active | 240K | 176K | 2,100 | 980K | 470K |
| Nova Retargeting | Paid Social | NA | Active | 90K | 78K | 720 | 640K | 360K |
| Nova Developer Community | Organic / SEO | NA | Active | 70K | 41K | 1,350 | 520K | 300K |
| Nova Partner Co-Marketing | Content Syndication | EMEA | Planned | 130K | 46K | 380 | 410K | 150K |
| Nova Analyst Relations | Content Syndication | NA | Completed | 60K | 58K | 210 | 720K | 300K |

**Headline numbers (Appendix A)**
- Revenue $4.75M; blended ROI 4.24×; pipeline $9.68M; 9,700 leads.
- Budget utilization 72%; 6 active campaigns.
- Top ROI: Developer Community at 7.3×. Biggest pipeline: Enterprise ABM at about $3.05M.
- Goals: revenue, ROI and leads on track; **pipeline at-risk**.
- Optimizer lift about 6–7%.
- 14 Kanban items.

**Plan tree:** FY2026 Marketing Plan → Nova GA Launch, Enterprise ABM, Brand & Community → programs (Paid Acquisition, Events & Webinars, Content Engine, …).

**Tactics named in the script:**
- Landing pages and the GA announcement: launched.
- "Top-150 named-account plays" and "Brand + category paid search": in progress, owned by Marcus.
- "'Meet Nova' launch webinar": 70%, owned by Abhishek.

**Seeded purchase requests:** Nova ABM LinkedIn ad flight (about $78K), launch webinar platform, and launch content. Seeded totals are 78,000, 22,000 and 36,000.

**Timed acts**

| Time | Act | What happens |
|---|---|---|
| 0:00 | Cold open | 5–10 tools per launch |
| 0:45 | Setup | The VP's problem |
| 1:30 | Planning | Hierarchy, board, drag a card, Jira/Trello import with undo, calendar subscribe |
| 3:30 | Budget and Procurement | Top-down pool, planned vs actual, PRs with approval chains ($78K vs $6K routing) |
| 5:00 | Campaigns | ABM for scale vs Community for efficiency |
| 6:15 | Home and Goals | Headline KPIs, auto insights, the at-risk pipeline flag |
| 8:00 | Analytics and Attribution | GA live; switch First touch → Position-based; funnel; reallocation |
| 9:45 | Optimizer | Sliders and recommendation, 6–7% lift |
| 10:45 | Builder | Blank canvas, chart previews, gauge with target turning green/red, custom axes, snapshot |
| 12:15 | Ask ZenReach | "Which campaign has the best ROI?", then "build me a dashboard" by conversation, the Prompts tab |
| 13:45 | Close | Summary |

- **5-minute version:** Planning → Home → Goals → Attribution → AI builds a dashboard.
- **Presenter tips:**
  - Name the job before showing the page.
  - Use the ROI contrast.
  - Pause on the at-risk goal.
  - Save the AI for last.
  - Technical rooms: attribution and custom axes. Executive rooms: Home, Goals, Optimizer.

**`docs/demo-scripts/` (newer, prose talk-tracks)**
- **01 Product Overview:**
  - Covers the three questions, the Overview with AI analysis, the hierarchy with SP&A (spend) and BCPIR (return) plans moving Draft → Submitted → Published, and the register, board and calendar views.
  - Mentions the people picker with notifications, top-down budget with a scope selector, and Procurement with assigned-approver enforcement.
  - Mentions the **audit log and recycle bin**, then Analytics, Attribution, Goals, Optimizer and the AI assistant.
- **02 Analytics:** the green/blue connection banner, the realtime counter, 4 KPIs with info icons, the 28-day trend, channels, countries and regions, top pages.
- **03 Integrations:**
  - Template download and CSV/Excel import showing imported and skipped counts, with Undo and "Manage in Planning". Import is Editor and above only.
  - Google Calendar push plus the Outlook feed.
  - Connector cards showing type, live vs sample, direction and method.
  - States plainly: GA, file import, Google Calendar and Jira/Trello import are working today; the rest are sample data.

### 14.2 Roadmap and future scope (`ZenReach-Future-Scope.md`)

The doc tags items green (near-term), yellow (mid-term) and blue (long-term).

- **§0 Foundations (near):**
  - Postgres with migrations.
  - Multi-tenancy.
  - JWT with refresh tokens, email verification, password reset, optional 2FA.
  - A background job queue.
- **§1 Teams and collaboration (near):**
  - Teams with per-team roles; team-scoped planning.
  - Assignment.
  - Presence and live updates with edit locking.
  - A richer activity feed, email/Slack notifications and an "assigned to me" inbox.
  - Shared vs private dashboards.
- **§2 Customizable boards: shipped.** Remaining polish: drag-to-reorder columns, per-user WIP limits, bulk actions in the calendar.
- **§3 Real integrations (mid):**
  - OAuth two-way connectors: Google, Meta and LinkedIn Ads, HubSpot, Salesforce, QuickBooks/NetSuite, Snowflake/BigQuery, Jira/Asana.
  - Scheduled ingestion.
  - Two-way Google Calendar with per-user OAuth.
  - Real UTM/pixel touchpoints for attribution.
- **§4 Smarter AI (mid):**
  - An agentic assistant whose tool calls (create tactic, move budget, reschedule) need confirmation.
  - Proactive anomaly alerts.
  - Scheduled AI digests.
  - Markov and Shapley attribution.
  - AI dashboard builder v2 that can edit existing dashboards.
- **§5 Reporting (mid):** scheduled PDF/CSV/email/Slack reports, public links and embeds, live-data slide decks, a Slack/Teams app.
- **§6 Finance depth (mid):** budget versioning and approval, multi-year and scenario planning, actuals reconciliation, vendor management, locked historical FX.
- **§7 Goals and optimization (long):** custom metrics and formulas, pacing alerts, Optimizer v2 with constraints, multiple objectives and write-back to budget.
- **§8 Analytics (long):** GA funnels, cohorts and custom dimensions; multi-property rollups; blending web, product and CRM data.
- **§9 Mobile and accessibility (long):** a PWA or native app with push, a WCAG 2.1 AA audit, i18n.
- **§10 Enterprise (long):** guided onboarding and templates, SSO/SAML, SCIM, granular permissions, a full audit log (since partly shipped), data residency, SOC 2.
- **Suggested first slice:** foundations → teams and boards → 1–2 real connectors (Google Ads, HubSpot) → agentic assistant plus scheduled summaries.
- **Open items from Project Context §16:**
  - Realtime users tile on the Overview.
  - Deep-linking the assistant to a specific chart.
  - AI takeaways on more pages.
  - Real connector scaffolds.
  - An editable (non-image) deck.

### 14.3 Hive9 parity: the plan and remaining gaps

**Original gap analysis (parity plan, 2026-07-17):**
- The app was a "beautiful performance dashboard with a flat board."
- Missing: hierarchy, line-item budgets, procurement, multiple views and attribution.
- Roles existed but were not enforced.

**Six phases, all shipped 2026-07-17:**
1. Hierarchy and tactic detail (line items, media codes, comments), migrating `initiatives.json` to `tactics.json`.
2. Calendar, Grid and Budget & Cost views; remembered view; saved filters; a heads-up chart strip.
3. Top-down pools, period allocations, multi-currency, snapshots, drill-down.
4. PRs with vendor lines, threshold-based serial/parallel chains, PO/invoice/receipt, approvals inbox.
5. Waterfall funnel, 9 models, reallocation, predictive forecast.
6. 13-connector catalog with sync labels, a working CSV import, role enforcement, notifications.

**Decisions taken:** all 6 phases; non-GA connectors as labelled simulations; roles enforced; `.ics` plus PDF instead of two-way calendar sync.

**Hive9 capabilities and how they map:**
- **Marketing calendar:** colour-coded by type, drag to reschedule, annual-to-weekly frames, PDF export, push to Outlook/Google. ZenReach has a month grid, `.ics`, PDF and Google push. Weekly and annual frames are not documented.
- **Five hierarchy levels:** ZenReach has four plus line items. Hive9 users say naming can't be changed.
- **Custom attributes, pivot, segmentation:** ZenReach has custom fields, group-by and saved views.
- **Role- and team-based security:** ZenReach has RBAC, org-scoped dashboards and audit; SSO is roadmap.
- **API and file connectors, bidirectional:** only GA, Calendar, FX and Jira/Trello are real, and those are one-way.
- **Budgets linked via line items, drill-down to PR, snapshots:** present.
- **PRs from budget lines, bulk edit, custom attributes:** partial. PRs have `customAttrs`.
- **Tactic IDs and media codes for martech ingestion:** a media-code generator exists; no ingestion.
- **Closed-loop reporting and predictive modelling:** synthetic.

**Gaps ZenReach still admits (spec doc "honest read"):**
- Breadth of enterprise connectors: warehouse, finance actuals, native CRM/MAP.
- Enterprise SSO and governance at scale.
- Actuals import.
- Real cross-channel touchpoints for attribution.
- Concurrency and multi-tenancy (JSON store, a single workspace).

**Hive9 gaps ZenReach targets (deep research):** usability, rigid naming, slow bulk entry, roughly 24h sync lag, painful import, partner-dependent connectors, price and opacity, heavy implementation, a fragmented brand, attribution that doesn't prove causation, little incrementality or MMM work.

**Product-management framing (`Product-Management-Module.md`):** Hive9/Uptempo runs marketing "the way Jira/Aha! manage a product."
- The original v1 "initiative" fields: `title`, `description`, `type`, `owner`, `quarter`, `priority`, `status`, `progress`, `budget`, `channel`, `goal`, `impact`, `effort`, `tags`, `dueDate`.
  - Types: Campaign, Launch, Content, Event, Research, Ops.
  - Priorities: P0, P1, P2.
  - Progress 0–100%; impact and effort 1–5.
- v1 had Board and Roadmap views and a summary strip (total, in-flight, launched, P0 count, planned budget, average progress).
- v1 endpoints: `/api/planning/board|summary|initiatives[/:id]`.

## 15. Data reference — documented architecture, conflicts, target schema, import formats and data files

Reference material for data migration and for reconciling the repository's documents with the code.

### 15.1 Architecture as documented

**Stack (current, per `package.json`):**
- **Client:** React 18.3, Vite 5, MUI 5 (with Emotion), Tailwind 3 (preflight disabled to coexist with MUI; `darkMode: 'class'`), GSAP 3, Apache ECharts 6.1 (tree-shaken, behind an `<EChart>` wrapper, with themes `zr-light` and `zr-dark`), react-router 6, html-to-image.
- **Server:** Express 4 (ES modules), dotenv, cors, morgan, `@google-analytics/data`, `openai`, `@anthropic-ai/sdk`, `xlsx`. It calls the Google Calendar REST API directly with fetch using `google-auth-library`; `googleapis` is not installed.
- **Root:** `concurrently`. Scripts: `install:all`, `dev`, `dev:server`, `dev:client`, `build`, `start`.
- **Ports:** API on 4000 (`node --watch` in dev); Vite on 5173, proxying `/api`.

**Containers (`docs/ZenReach-Architecture.md`):**
- The browser SPA (pages, Auth/Currency/ThemeMode contexts, a `useApi` hook, an `api.js` fetch wrapper that sends a Bearer token) talks to `/api/*`.
- `index.js` handles routes plus gating with `requireRole(min)` and `requirePlanner`.
- Services are grouped as:
  - **core:** mockData, overview, goals, insights, optimizer;
  - **planning & org:** plans, planning, boardConfig, org, budgets, procurement, planLoop (plus campaignPlans and models in the current code);
  - **builder & import:** builder, campaignDataset, importer, boardImport, boardConnections, spreadsheet;
  - **AI:** assistant, assistantDashboard, assistantWidgets, analysis;
  - **integrations:** googleAnalytics, googleCalendar, calendar, fx, attribution, notifications;
  - **auth & persistence:** auth, store (plus audit and recycleBin in the current code).
- Persistence is `readJson` / `writeJson` against `server/data/*.json`.
- **Principle:** the browser sends only configs and receives small JSON. All compute and all external calls happen on the server, so keys never reach the browser.

**Auth lifecycle:**
1. Login returns `{token, user}`.
2. The token is stored in `localStorage.zr_token` and sent as a Bearer header on every request.
3. The server gates each request; failures return 401 or 403.
- **Public allow-list:**
  - `POST auth/login`, `POST auth/signup`, `GET auth/me`, `POST auth/logout`;
  - `GET org/personas`, `GET currency`, `GET health`;
  - `GET planning/calendar.ics`.
- Everything else returns 401 to anonymous callers.

**The data spine:**
- `mockData.campaigns[]` holds 8 Nova campaigns. Overview, budget-by-channel, revenueByChannel, optimizer, goals, insights, the AI context and the Builder's campaign pivots are all derived from it.
- `timeseries()` is a deterministic "launch ramp."
- `mockData` is a module constant, so code changes need a server restart. File-backed stores are read on every request, so file edits show up live.

**Two hierarchies:**
- The org chart lives in `teams.json` and `users.json`.
- The plan tree lives in `plans.json` and `tactics.json`.
- They meet through `team.campaignIds`. Rollups come from tactics.
- **Self-heal pattern:** `org.healCampaignLinks()` re-maps team campaigns by name using `TEAM_CAMPAIGNS`, and `budgets.js` rebuilds stale allocations.

**Assistant flow:** `assistantTurn()` continues an active build flow, starts one on a build intent, or otherwise runs `ask()`. `ask()` builds context (overview, campaigns, goals, insights) and walks the provider cascade.

**Production target (spec doc), on Azure:**
- Static Web Apps (CDN, TLS) for the SPA.
- Application Gateway with WAF in front of a VM Scale Set of 2–3 Standard D2s v5 Ubuntu 22.04 VMs across availability zones, running Docker, PM2 and Nginx.
- **Azure Database for PostgreSQL Flexible Server**, General Purpose D2ds v5, zone-redundant HA, point-in-time restore.
- Blob Storage for uploads, snapshots and exports.
- Optional Redis.
- Key Vault with Managed Identity.
- GitHub Actions with Azure Container Registry.
- Application Insights and Log Analytics.
- Bicep or Terraform; environments dev → staging → prod with approval gates; blue-green deploys.
- **Migration path:** a schema mirroring the entities → a one-time importer from `server/data/*.json` → a feature flag → verify parity → cut over. `store.js` isolates persistence.

### 15.2 Where the docs conflict with each other or with the repo

1. **Planning views.**
   - Product Overview and Architecture §9 list five: Board, Calendar, Grid, Budget & Cost, Roadmap.
   - Project Context lists six, adding AI Copilot.
   - The spec doc says "Five views incl. AI Copilot."
   - The README says three focused views: Plans, Board, Calendar.
   - The current code (`Planning.jsx`) has `VIEWS = plans, grid, board, calendar, budget`, labelled Plans / Grid / Board / Calendar / Budget & Cost, with Plans as the default. There is no Roadmap or Copilot toggle, and editable views are only board and calendar.
2. **Hierarchy depth.**
   - The docs say four levels.
   - `planLevels.js` (uncommitted) says five, counting Line item.
   - The parity plan cites Hive9's "five planning hierarchy levels."
   - The ER file uses Shell/Hive9 names: plans → campaigns → initiatives → activities.
3. **AI provider.**
   - Product Overview and Project Context §3 and §10 say Azure gpt-4o is primary.
   - The README and Architecture doc say OpenAI GPT-5 → Azure → Claude → built-in. The README's own structure comment still says `analysis.js` uses Azure/Claude.
   - `.env.example` lists only OpenAI and Anthropic, says Claude is "used only if no OPENAI_API_KEY", and gives the default Claude model as `claude-opus-4-8`.
4. **Charts.** Project Context §3 and the parity plan say Recharts. The code moved to ECharts, and `recharts` was uninstalled.
5. **Attribution.** Project Context §4 and §7 say it was removed and moved to the roadmap. It was restored in Phase 5, and `Attribution.jsx` exists.
6. **Connectors.**
   - Product Overview and Project Context §7 say Salesforce, **HubSpot**, Google Ads and QuickBooks are pre-connected.
   - The current catalog has **no HubSpot**. Its 13 entries are: GA4, Salesforce, SAP, Google Ads, QuickBooks, NetSuite, Coupa, Google Calendar, Outlook Calendar, BigQuery, Snowflake, Asana, Jira.
   - The Google Calendar and Jira cards are labelled `sample` even though calendar push sync and Jira import are real.
   - The spec doc calls Salesforce, SAP, HubSpot and Marketo "connector-ready."
   - Product Overview says "Every connector is active."
7. **Login.**
   - The README and Product Overview say everything runs in demo mode with no account and sign-in is optional; after the first account, edits need an Editor.
   - An intermediate session fix made anonymous requests count as an open demo user. The doc flags this as security-relevant.
   - The final Project Context state is a login wall: anonymous calls get 401 except the allow-list, and only the landing page renders.
8. **Home page.**
   - Product Overview and the Nova script ("Go to Home… KPI tiles… insights cards") describe the KPI and AI overview.
   - In code, Home is the dashboards viewer or TeamDashboard, and the old overview is unrouted. Screenshots 02, 03 and 04 were repurposed to other surfaces.
9. **Currency switch location.** One memory note says top bar; the later docs say `PageHeader` (`CurrencyControl.jsx`, which exists).
10. **Page and store inventories.**
    - Architecture lists 13 pages and 13 stores.
    - The repo has 15 pages (adding `AuditLog.jsx` and `RecycleBin.jsx`) and 16 data files (adding `audit.json`, `recycleBin.json`, `models.json`). The dual-plan routes are also missing from the Architecture doc.
11. **Missing files referenced by the docs.**
    - `docs/Nova-Demo-Script.md` (the quick script).
    - `Connect-Google-Analytics.md` (removed).
    - `scripts/capture-screenshots.mjs` (there is no `scripts/` folder).
    - `server/data/initiatives.json` (the legacy migration source).
    - `attribution.json` and `fx.json` (proposed in the parity plan, never created).
    - `docs/Product-Management-Module.md` (the file is actually at the repo root).
12. **Seed drift between docs and live data.**
    - The script says 14 Kanban items (Backlog 3, Planned 4, In progress 5, Launched 2). `tactics.json` has 17 (4/5/4/4) in Q3–Q4 2026.
    - The docs say $735K committed. Tactic budgets now sum to $573.2K.
    - The docs say `campaignDataset.json` holds the 500-row `zenreach-campaign-500.xlsx`. It actually holds an aggregate of **`zenreach-tactics-1500.xlsx`**, the wrong template. As a result, segment, audience, location and company are all "—", `monthly` is empty and `dateRange` is null. This is the clobbering the docs warn about.
13. **Template persona.** The root `zenreach-tactics-template.csv` uses "Priya Nair"; the API's served `SAMPLE_CSV` uses "Abhishek Biyani."
14. **ER file.** `hive9-schema-erdiagram.md` is raw Mermaid with no code fence, so it won't render as a diagram in Markdown.
15. **Old instructions.** README quick-start says `cd Hive9`, an old folder name.

### 15.3 Target relational schema (`hive9-schema-erdiagram.md`, 50 entities)

The file uses a Shell/Hive9-style schema. Every tenant-scoped table carries `tenant_id` → `tenants`. Columns are abbreviated to keys plus a few fields, and all IDs are UUIDs.

**Identity and tenancy (10)**
- `tenants(id, name, slug, status)`
- `users(id, tenant_id, email, first_name, last_name, avatar_file_id → file_uploads)`
- `roles(id, tenant_id, name)`
- `permissions(id, key, module_key)`
- `role_permissions(role_id, permission_id)`
- `user_roles(user_id, role_id, assigned_at, assigned_by → users)`
- `teams(id, tenant_id, name, status)`
- `team_memberships(team_id, user_id, can_edit_team_content, can_approve)`
- `user_preferences(user_id PK, preferred_currency_code, default_home_route, theme_key)`, 1:1 with users
- `delegated_access(id, tenant_id, user_id, delegate_user_id, access_mode, starts_at, ends_at, status)`

**Planning (8)**
- `plans(id, tenant_id, status_id → plan_statuses, owner_user_id, fiscal_year, start_date, end_date)`
- `campaigns(id, tenant_id, plan_id, status_id, owner_user_id)`
- `initiatives(id, tenant_id, campaign_id, status_id, owner_user_id)`
- `activities(id, tenant_id, initiative_id, status_id, activity_type_id, owner_user_id)`
- `plan_statuses(id, tenant_id, entity_type, name)`
- `activity_types(id, tenant_id, name)`
- `hierarchy_closure(tenant_id, ancestor_type, ancestor_id, descendant_type, descendant_id, depth)`
- `planning_snapshots(id, tenant_id, root_entity_type, root_entity_id, snapshot_type, version_no)`
- Mapping: Shell's initiatives ≈ ZenReach programs; activities ≈ tactics.

**Custom attributes and views (5)**
- `attribute_definitions(id, tenant_id, entity_type, key, label, data_type)`
- `attribute_options(id, attribute_definition_id, value_key, value_label)`
- `entity_attribute_values(id, tenant_id, entity_type, entity_id, attribute_definition_id, option_id)`
- `saved_views(id, tenant_id, module_key, entity_scope, owner_user_id)`
- `saved_view_filters(id, saved_view_id, filter_key, operator)`

**Finance (9)**
- `fiscal_periods(id, tenant_id, fiscal_year, month_no)`
- `currencies(code PK, name, symbol)`
- `exchange_rates(id, tenant_id, fiscal_year, currency_code)`
- `budgets(id, tenant_id, owner_user_id, currency_code, fiscal_year, status)`
- `budget_lines(id, tenant_id, budget_id, parent_budget_line_id, entity_type, entity_id, name)`, self-referencing tree
- `vendors(id, tenant_id, vendor_code, name, status)`
- `transactions(id, tenant_id, external_transaction_id, transaction_date, vendor_id, object_currency_code, posted_currency_code, import_batch_id)`
- `transaction_lines(id, transaction_id, line_no, name, amount)`
- `transaction_attributions(id, tenant_id, transaction_id, transaction_line_id, target_entity_type, target_entity_id, attributed_amount, attributed_by)`

**Procurement and accruals (5)**
- `purchase_requests(id, tenant_id, owner_user_id, created_by, approval_workflow_id → workflow_definitions, name, level_type, status)`
- `purchase_request_lines(id, purchase_request_id, vendor_id, budget_line_id, linked_entity_type, linked_entity_id, line_no, amount)`
- `accrual_types(id, tenant_id, name, is_active)`
- `accrual_requests(id, tenant_id, owner_user_id, accrual_type_id, approval_workflow_id, name, level_type, status)`
- `accrual_request_lines(id, accrual_request_id, budget_line_id, line_item_entity_id, planned_amount_in_period, actual_amount_in_period, line_amount)`

**Workflow and approvals (5)**
- `workflow_definitions(id, tenant_id, name, entity_type, status, trigger_mode)`
- `workflow_steps(id, workflow_definition_id, step_order, name, approver_type, approver_role_id, approver_user_id)`
- `approval_requests(id, tenant_id, workflow_definition_id, requester_user_id, entity_type, entity_id, current_step_order, approval_status)`
- `approval_assignments(id, approval_request_id, workflow_step_id, approver_user_id, assignment_status)`
- `approval_decisions(id, approval_request_id, approval_assignment_id, decided_by, decision)`

**Reporting (3)**
- `report_definitions(id, tenant_id, key, name, module_key, dataset_key)`
- `report_widgets(id, report_definition_id, widget_type, title, sort_order)`
- `report_shares(id, tenant_id, report_definition_id, shared_by, shared_to_user_id, shared_to_team_id, access_level)`

**Platform (5)**
- `notifications(id, tenant_id, user_id, type, title, is_read)`
- `alerts(id, tenant_id, user_id, severity, title, status)`
- `audit_events(id, tenant_id, actor_user_id, entity_type, entity_id, action, module_key)`
- `file_uploads(id, tenant_id, uploaded_by, storage_key, original_filename, mime_type, size_bytes)`
- `import_batches(id, tenant_id, created_by, module_key, status, filename)`

**Relationships**
- Most are 1:N. `users` to `user_preferences` is 1:1. `budget_lines` references itself.
- Accruals have no ZenReach equivalent today. Neither do transactions, delegated access, workflow definitions as data, or reports as data.

**The current JSON ER (Architecture §3), for comparison**
- USER manages USER via `managerId`; USER leads TEAM via `leadId`.
- TEAM to USER via `memberIds`, and TEAM to CAMPAIGN via `campaignIds`, both N:M.
- PLAN → CAMPAIGN → PROGRAM → TACTIC.
- BOARD_CONNECTION → TACTIC via `source.connectionId`.
- DASHBOARD → WIDGET.
- BUDGET_POOL → ALLOCATION (campaign × quarter).
- PURCHASE_REQUEST optionally funds a TACTIC; PURCHASE_REQUEST → APPROVAL.

### 15.4 Import file formats

#### 15.4.1 Tactics CSV or Excel

Endpoints: `POST /api/import/tactics`; template from `GET /api/import/sample`. Requires Editor or above.

**Exact header** (template, both sample CSVs and `zenreach-tactics-1500.xlsx`, sheet "Tactics"):
```
title,type,owner,quarter,status,priority,budget,channel,goal
```

**Header aliases** (case-insensitive):

| Field | Accepted headers |
|---|---|
| title | `title`, `name`, `tactic` (required) |
| type | `type` |
| owner | `owner` |
| quarter | `quarter`, `qtr` |
| status | `status` |
| priority | `priority` |
| budget | `budget`, `cost`, `amount` |
| channel | `channel` |
| goal | `goal`, `objective` |

Unknown columns are ignored.

**Value handling and defaults:**
- `type` must be one of Campaign, Launch, Content, Event, Research, Ops; otherwise **Campaign**.
- `owner` defaults to "Unassigned".
- `quarter` defaults to "Q3 2026".
- `status` maps backlog, planned, in-progress, "in progress", launched, done→launched and complete→launched; anything else becomes **backlog**.
- `priority` must be P0, P1 or P2; otherwise **P2**.
- `budget` is stripped of non-numeric characters; otherwise 0.
- `channel` and `goal` are free text.
- Every row lands in the default program.
- Rows without a title are skipped with "Row N: missing title — skipped."
- The response is `{imported, ids, errors}`. The `ids` drive "Undo import."

**Parser:** a custom CSV parser that handles quoted fields, escaped quotes and CRLF, and drops blank rows. Excel files (.xlsx, .xls, .ods) arrive base64-encoded and are converted to CSV with SheetJS.

**Sample files:**
- `zenreach-tactics-template.csv`: 2 rows.
- `zenreach-tactics-500.csv`: 500 rows.
- `zenreach-tactics-1500.csv` and `.xlsx`: 1,500 rows each.

**Value domains in the sample files:**
- **Statuses:** lower-case (`backlog`, `planned`, `in-progress`, `launched`).
- **Channels:** Content Syndication, Direct, Email, Events & Webinars, Organic / SEO, Paid Search, Paid Social, Referral.
- **Quarters:** Q1 2026 to Q4 2027.
- **Budgets:** 5,000 to 300,000.
- **Goals:** free text, e.g. "300 registrations", "+15% organic sessions", "ROI 3.6x", "Open $4.7M pipeline".
- **Titles:** repeat with numeric suffixes or em-dash variants (e.g. "Field Event — NYC", "Product Launch — Halo").

First 5 data rows of the 500-row file (no PII):
- Data Hygiene Sweep 4, Ops, Sam Okafor, Q1 2026, in-progress, P2, 19500, Paid Social, "Reduce dupes 8%"
- Attribution Model Review 12, Research, Dana Alvarez, Q1 2027, planned, P1, 5500, Paid Search, "Pick default model"
- Case Study Program 2, Content, Raj Patel, Q1 2026, backlog, P0, 35000, Paid Social, "+8% organic sessions"
- Attribution Model Review 12, Research, Dana Alvarez, Q1 2027, in-progress, P2, 20500, Referral, "Ship findings deck"
- Feature GA — Orbit, Launch, Raj Patel, Q3 2026, in-progress, P1, 104000, Referral, "Open $4.7M pipeline"

First 5 data rows of the 1500-row file:
- ABM Direct Play, Campaign, Owen Wright, Q1 2027, backlog, P0, 94000, Direct, "ROI 3.6x"
- Email Demand Gen Push, Campaign, Tom Becker, Q1 2026, backlog, P1, 26000, Email, "3000 MQLs"
- Field Event — NYC, Event, Sam Okafor, Q2 2026, backlog, P0, 173500, Paid Search, "500 attendees"
- Product Launch — Halo, Launch, Jordan Kim, Q4 2027, planned, P2, 71000, Email, "+20% adoption"
- Thought-Leadership Series 13, Content, Sam Okafor, Q2 2026, in-progress, P0, 45000, Paid Social, "1500 backlinks"

#### 15.4.2 Campaign dataset for Builder sources

Endpoints: `POST /api/dataset/campaign/import`, `GET /api/dataset/campaign/meta`.

**Documented columns:**
```
Campaign_ID, Company, Campaign_Type, Target_Audience, Duration, Channel_Used, Conversion_Rate, Acquisition_Cost, ROI, Location, Date, Clicks, Impressions, Engagement_Score, Customer_Segment
```
The test files add a `Language` column.

**Aliases** (case-insensitive):

| Field | Accepted headers |
|---|---|
| company | `company` |
| type | `campaign_type`, `type` |
| audience | `target_audience`, `audience` |
| channel | `channel_used`, `channel` |
| conversion | `conversion_rate`, `conversion` |
| acquisition | `acquisition_cost`, `acquisition`, `cost` |
| roi | `roi` |
| location | `location` |
| date | `date` |
| clicks | `clicks` |
| impressions | `impressions` |
| engagement | `engagement_score`, `engagement` |
| segment | `customer_segment`, `segment` |

- Numbers are stripped of `$`, `,`, `%` and spaces.
- Dates are normalized to YYYY-MM for the monthly series.
- Rows are pre-aggregated: top 12 locations, top 15 companies by clicks.

**Sample files:**
- `zenreach-campaign-500.xlsx` and `zenreach-campaign-1500.xlsx`: sheet "Campaigns", 16 columns ending `…,Engagement_Score,Customer_Segment,Date`, with Language after Location. Example values: Duration "45 days", Channel_Used "Facebook" or "Google Ads", Conversion_Rate 0.138, Date "2026-09-21".
- `marketing_campaign_dataset.xlsx`: about 200,005 data rows, 15 columns, no Language. Date is stored as an Excel serial number; `Duration` is like "30 days".

**Builder source IDs:** `cmpKpis`, `cmpChannel`, `cmpType`, `cmpSegment`, `cmpAudience`, `cmpLocation`, `cmpMonthly`, `cmpCompanies`, and `cmpFlex` (custom axes).

#### 15.4.3 Trello and Jira (API, not files)

**Trello**
- Needs an API key (Power-Up admin) and a token (`scope=read`, `expiration=never`). The board is picked in the app.
- Field mapping:
  - card name → title;
  - description → description;
  - list name → column;
  - first member → owner;
  - labels → tags;
  - due → due date and quarter.

**Jira Cloud**
- Needs the site URL, email and API token (Basic auth). The project is picked in the app, or a JQL query overrides it.
- Field mapping:
  - summary → title;
  - description, flattened to plain text → description;
  - status category → column (To Do → Planned, In Progress → In progress, Done → Launched; names containing "backlog" → Backlog; otherwise the closest column, else the first);
  - assignee → owner;
  - priority: Highest/High → P0, Medium → P1, Low/Lowest → P2;
  - issue key plus labels → tags;
  - due → due date and quarter.

**Import behaviour**
- Re-importing does not de-duplicate.
- Undo deletes the whole batch.
- Each import records a connection; disconnect deletes the union of `tacticIds` and tactics whose `source.connectionId` matches. It is Editor-only, permanent, and never calls the provider.
- **Troubleshooting:** 401/403 means a wrong token or no access; "Couldn't reach" means a bad site URL; everything landing in one column means names didn't match.

**Endpoints:** `POST /api/import/{trello,jira}/{boards,projects}`, `POST /api/import/{trello,jira}`, `GET /api/import/connections`, `DELETE /api/import/connections/:id`.

#### 15.4.4 `.ics` feed
`GET /api/planning/calendar.ics[?planId|campaignId|programId]` is public. It serves all-day events with CRLF line endings and refresh hints.

### 15.5 Persisted data files (`server/data/`, gitignored; shapes and counts only)

| File | Top-level shape | Count / notes |
|---|---|---|
| `users.json` | `{ users: [...], sessions: { <token>: <userId> } }` | 27 users, 79 sessions |
| `teams.json` | `{ meta: { orgSeeded: bool }, teams: [...] }` | 6 teams |
| `plans.json` | array of hierarchy nodes | 14 nodes |
| `tactics.json` | array of tactics | 17 tactics |
| `models.json` | array of planning models | 1 model |
| `boardConfig.json` | `{ columns: [...], fields: [] }` | 4 columns, 0 custom fields |
| `boardConnections.json` | array | empty |
| `planLoops.json` | array | empty |
| `budgets.json` | `{ pools, allocations, snapshots }` | 1 pool, 12 allocations, 0 snapshots |
| `purchaseRequests.json` | `{ requests: [...] }` | 3 PRs |
| `dashboards.json` | array of dashboards | 4 dashboards |
| `campaignDataset.json` | object of aggregates | built from 1,500 rows of the wrong file |
| `notifications.json` | object keyed by user ID | 12 users |
| `recycleBin.json` | `{ items: [...] }` | 5 items |
| `audit.json` | `{ events: [...] }` | 50 events |
| `settings.json` | `{}` | Google Calendar not currently connected |

**`users.json`**
- **User fields:** `id`, `name`, `email`, `company`, `role`, `salt`, `passwordHash`, `avatarGradient`, `initials`, `createdAt`, `orgRole`, `title`, `teamId` (26 of 27), `managerId` (25 of 27). Hashes and emails are not copied here.
- **Roles:** Owner 1, Admin 1, Editor 25.
- **Org roles:** admin 3, manager 2, lead 4, member 18.
- **Email domains:** 25 on the demo domain, 2 real.
- **Sessions:** a map from opaque token to user ID, with no expiry fields, so it keeps growing.

**`teams.json`** team fields: `id`, `name`, `focus`, `color`, `leadId`, `memberIds[]` (3–4 each), `campaignIds[]` (1–2 each, all 9 valid), `createdAt`, `updatedAt`.

**`plans.json`**
- **Counts:** 2 `plan`, 5 `campaign`, 7 `program`.
- **Common fields:** `id` (e.g. `plan-…`, `campaign-…`), `createdAt`, `updatedAt`, `level`, `parentId` (null for plans), `name`, `owner`, `ownerId`, `description`, `startDate`, `endDate`, `dateMode` (auto 11, manual 3), `fiscalYear`, `currency` (all USD), `planBudget` (all 0 currently), `status` ('' 11, 'Published' 3), `approval`, `attrs{}`, `comments[]`.
- **`approval` keys:** `submittedBy`, `submittedAt`, `approver`, `approverId`, `approvedBy`, `approvedAt`, `rejectedBy`, `rejectedAt`, `note`.
- **Plan-level-only fields:** `modelId` ("model-standard"), `teams[]`, `isStrategic`, `goals{}`, `defaultMechanism` ("E").
- All parent links are valid.

**`tactics.json`** (all 17 have every field)
- **Identity and text:** `id`, `title`, `externalName`, `type`, `description`, `goal`, `tags[]`.
- **Ownership:** `owner`, `ownerId`.
- **Scheduling:** `quarter`, `dueDate`, `startDate`, `endDate`, `evergreen`.
- **Prioritization and progress:** `priority`, `status` (a board column ID), `progress`, `impact`, `effort`.
- **Money:** `budget`, `channel`, `spendCommitted` (null), `spendActual` (null or number).
- **Hierarchy:** `planId`, `campaignId`, `programId` (all valid).
- **Approval:** `approvalStatus` (Created 9, Approved 8) and `approval{approver, approverId, submittedBy, submittedAt, history[]}`.
- **Waterfall goals:** `goals{inquiries, mqls, tqls, opportunities, won, revenue, override}`.
- **Detail:** `lineItems[]`, `mediaCodes[]`, `performance{plan{leads, pipeline, revenue}, actual{leads, pipeline, revenue}}`, `customAttrs{}`, `comments[]`.
- **Provenance:** `source` (null or `{provider, name, connectionId}`), `loopId` (null), `targets` (null or `{leads, revenue}`).
- **Timestamps:** `createdAt`, `updatedAt`.
- **Value domains:**
  - types: Campaign 7, Content 5, Event 3, Launch 1, Research 1;
  - statuses: planned 5, launched 4, in-progress 4, backlog 4;
  - priorities: P0 6, P1 6, P2 5;
  - channels mix the canonical list with free-text ones (Webinar, Event, Content, Social Media);
  - quarters: Q3 2026 and Q4 2026.
- Line items, media codes and comments are currently empty everywhere.
- The client also uses a lifecycle palette: Created, Submitted, Declined, Approved, In-Progress, Complete.

**`models.json`**
- One model with fields `id`, `name`, `description`, `rates{mqls, tqls, opportunities, won}` (conversion into each stage, 0–1), `velocityDays{…}`, `averageDealSize`, `tacticTypes[6]`, `requirePlannedCost`, `blockNegativeBalance`, `goalHorizonMonths`, `createdAt`, `updatedAt`.
- Code defaults:
  - ID "model-standard", name "ZenReach Standard B2B";
  - rates .25 / .4 / .5 / .3;
  - velocity 7 / 14 / 21 / 45 days;
  - average deal size 25,000;
  - require planned cost: true; block negative balance: false;
  - goal horizon 3 months.
- Waterfall stages: inquiries → MQLs → TQLs → opportunities → closed won.

**`boardConfig.json`**
- Column fields: `id`, `name`, `hint`, `color`, `wip`. The four columns are `backlog`, `planned`, `in-progress`, `launched`.
- Custom field definitions, when present: `{id, label, type: text|number|date|select, options}`.

**`boardConnections.json`** record shape: `{id, provider (≤20 chars), name (≤140), key (≤60 or null), count, tacticIds[], importedAt}`. The list endpoint adds a `liveCount`.

**`planLoops.json`** record shape: `{id, name, quarter, goalLeads, budget, planId, campaignId, programId, tacticIds[], createdAt}`. The list endpoint adds `liveTactics` and drops loops with no live tactics.

**`budgets.json`**
- Pool: `{id, name, planId, currency (USD), total (900,000), createdAt, updatedAt}`.
- Allocations: `{id, poolId, nodeId, period, amount}` for 3 campaigns × Q1–Q4 2026. They sum to 900,000 and all reference valid campaigns. Only 3 of the 5 campaigns have allocations.
- Snapshot shape (from code): `{id, label (≤80), at, currency, totals, rows[{name, planned, committed}]}`.

**`purchaseRequests.json`**
- PR fields: `id`, `title`, `requester`, `vendor`, `currency`, `tacticId`, `campaignId`, `customAttrs`, `poNumber`, `invoiceNumber`, `createdAt`, `updatedAt`, `history[{at, by, action}]`, `status`, `lines[{id, description, qty, unitPrice, amount}]`, `total`, `approvals[{id, level, name, mode (serial|parallel), approver, status (pending|approved|rejected), at}]`.
- Newer code also adds `requesterId` and `approverId` on PRs and steps.
- **Current data:** statuses draft, submitted, approved; totals 78K, 22K, 36K; approval steps named Marketing Ops and Finance.

**`dashboards.json`**
- Dashboard fields: `id`, `name`, `createdAt`, `updatedAt`, `widgets[]`.
- Widget fields: `id`, `type`, `title`, `source`, `field` or `fields[]`, `w` (1, 2 or 3), optional `threshold{value, direction}` and optional `x` (dimension).
- **Current data:**
  - 4 dashboards with 7, 6, 1 and 7 widgets.
  - Widget types: kpi 11, bar 4, donut 2, table 2, area 1, line 1.
  - Sources: overview, cmpKpis, cmpChannel, budgetChannel, campaigns, cmpFlex, budgetRegion, cmpMonthly, cmpType.
  - Documented seeds include "Executive summary" and "Campaign dataset overview."

**`campaignDataset.json`**
- **Top-level keys:**
  - `meta{rows, source (filename), updatedAt, dateRange ({start, end} or null), dimensions{channels, types, segments}}`;
  - `kpis{campaigns, avgConversionRate, avgRoi, totalClicks, totalImpressions, totalAcquisitionCost, avgEngagement, ctr}`;
  - `byChannel`, `byType`, `bySegment`, `byAudience`, `byLocation`, `topCompanies`, each an array of `{<dim>, count, clicks, impressions, acquisitionCost, conversionRate, roi, engagement, ctr}`;
  - `monthly[]`.
- **Current data:** 1,500 rows from the wrong file. It has 8 channels and 6 types; the other dimensions are "—"; `monthly` is empty.

**`notifications.json`**
- Keyed by user ID. There are two generations of feed shape:
  - old: `{count, items[], updatedAt}`;
  - new: `{pushed[], items[], seenIds[], total, updatedAt}`.
- Item shape: `{id, type, title, text, at, route, from?, authored?}`.
- Types: approval, overdue, due, assignment, mention.
- Routes: `/planning`, `/procurement`, `/planning?plan=spa:<campaignId>`.
- ID prefixes: `push-…`, `due:…`, `approval:…`.
- Pushed alerts are stored per recipient by user ID, never by name.

**`recycleBin.json`**
- Item shape: `{id ('trash-…'), deletedAt, type (node|dashboard|tactic|request), label, sublabel, count, by, payload}`.
- Payload by type:
  - dashboard: `{dashboard{…}}`;
  - node: `{nodes[...full node records], tacticLineage[{id, planId, campaignId, programId}]}`.
- **Current data:** 3 node items and 2 dashboard items.
- Teams, budget snapshots and board connections are still hard-deleted.

**`audit.json`**
- Event shape: `{id ('evt-…'), at, action ('<category>.<verb>'), category, severity, actorId, actorName, actorRole, actorOrgRole, actorInitials, actorGradient, entity (a noun phrase), entityId, label, summary, meta}`.
- **Current data:** 50 events, 24 of them seeded with `meta.seed`.
- **Actions seen:**
  - account.login / logout;
  - team.create;
  - plan.create / submit / approve / reject;
  - tactic.create / update / delete / comment;
  - budget.pool / snapshot_create;
  - pr.create / submit / approve / advance;
  - dashboard.create / update / delete;
  - import.tactics;
  - user.role;
  - recycle.restore.
- **Categories:** account, org, planning, tactics, budget, approvals, procurement, dashboards, imports, recycle.
- **Severities:**
  - info: routine actions;
  - notice: workflow steps (submit, approve, restore);
  - warning: deletes;
  - critical: purge, role change, empty bin.
- The store is capped at 5,000 events.
- **Visibility:**
  - Owner, Admin and org admin see everything.
  - A manager sees themselves, their leads and those leads' teams.
  - A lead sees themselves and their teams.
  - Everyone else sees only their own events.

**`settings.json`**: `{}` now. When connected it holds `{googleCalendar: {calendarId, connectedAt}}`.

**Referenced but no longer present:** `initiatives.json` (the legacy v1 store, migrated into `tactics.json`).

### 15.6 `screenshots/` folder (26 PNGs)

`01-hero.png`, `02-home-full.png`, `02-kpis.png`, `03-ai-analysis.png`, `04-performance.png`, `05-campaigns.png`, `06-budget.png`, `07-goals.png`, `09-optimizer.png`, `10-analytics.png`, `11-integrations.png`, `12-assistant.png`, `13-mobile-goals.png`, `14-builder.png`, `15-planning.png`, `16-procurement.png`, `17-attribution.png`, `18-ai-dashboard-ask.png`, `19-ai-dashboard-focus.png`, `20-ai-dashboard-recommend.png`, `21-ai-dashboard-built.png`, `22-ai-dashboard-result.png`, `23-google-calendar-integrations.png`, `24-google-calendar-full.png`, `25-google-calendar-dialog.png`, `26-admin.png`.

- There is no `08`.
- Product Overview embeds 01, 02-kpis, 03, 04, 05, 06, 07, 09, 10, 11, 12, 13, 14, 15, 20, 21, 22 and 23.
- 02, 03 and 04 were repurposed to the team dashboard, the assistant recommendation and the Dashboards viewer.
- The captures were taken at 1440×900 @2x in light mode; mobile at 390×844.
- `client/public/marketing/` holds 3 landing images: `ai-analysis.png`, `dashboard.png`, `planning.png`.
- `Media/` (gitignored) holds 47 Hive9 reference screenshots and 4 recordings.

### 15.7 Other repo artifacts and lessons worth carrying into the rebuild

**Other artifacts**
- `tracking-test.html` (tracked; also at `client/public/`) is a GA4 traffic generator. It must be opened over http, and it can "Auto-generate 25 hits."
- The pitch deck (`ZenReach-Pitch-Deck.pptx`, 19 slides, every slide an image) and the solution and roadmap decks are artifacts, not app inputs.

**Engineering lessons recorded in the docs**
- Disable Tailwind preflight and add an anchor reset.
- Avoid GSAP ScrollTrigger under React StrictMode; use an IntersectionObserver reveal instead.
- Use `history.scrollRestoration='manual'` plus a ScrollToTop component.
- Pin header grid cells to explicit columns.
- Don't remount pages on currency change; re-render instead.
- Use stable, unique dashboard names.
- Self-heal any ID-linked seeds (budgets, team campaigns) after a re-seed.
- Stamp import provenance on tactics.
- Snapshot entities into the recycle bin rather than using a `deletedAt` flag.
- Store an actor snapshot on each audit event.
- Key notifications by user ID, not name.
- Enforce the assigned approver on the server as well as hiding buttons in the UI.

## 16. Companion product — Dashboard Studio

**Location:** `dashboard-studio/` (a separate app with its own client and server; git-ignored in the ZenReach repository). Section 16.29 lists known bugs, gaps and documentation inconsistencies.

### 16.1 Purpose and product summary

Dashboard Studio is a standalone, no-code dashboard builder that was spun out of the larger ZenReach app. A user:

1. uploads CSV, TSV/TXT, XLSX, XLS or ODS files;
2. the server parses each file into its own named **dataset** and infers each column's type (`number`, `date` or `category`) plus a display format;
3. the user builds **dashboards**. Each dashboard is bound to exactly **one dataset** and holds **widgets** (charts, KPI tiles, tables). A widget is made by mapping columns onto chart "slots", and the editor only offers columns that fit each slot;
4. the user can recolor series from a fixed 14-color palette (no two series in a widget may share a color), resize widgets (S/M/L), drag to reorder, pin to a top row, hide from viewers, sort, keep only the top N, add reference lines, and add green/red target thresholds;
5. the user views dashboards read-only and saves **PNG snapshots** on the server;
6. a built-in **assistant** (a chat bubble in the bottom-right) builds dashboards through a guided wizard and turns plain-English chart requests into widgets. It also answers questions about the data. If `OPENAI_API_KEY` is set, open-ended answers come from OpenAI (default model `gpt-5`).

**Core rule: no dummy data.** Nothing renders until a dataset exists. If a widget's dataset is deleted, the widget stops rendering and nothing is substituted.

**Core architecture rule.** All parsing and aggregation run on the server. The browser never downloads raw rows. It sends a small widget config and gets back a small, chart-ready payload.

### 16.2 Architecture and tech stack

```
Browser (React 18 SPA, Vite :5174) ──/api proxy──► Express API (:4100)
  pages: Landing · Builder · Viewer                services: parse · infer · chartTypes · aggregate
  contexts: Datasets, ThemeMode                    filters (dormant) · join · suggest · jobs
  hooks: useDashboards, useWidgetQuery             snapshots · store · assistant · assistantDashboard
  ECharts v6 (modular, canvas)                     · assistantWidgets
                                                   disk: server/data/** (JSON + PNG)
                                                   optional: OpenAI chat.completions
```

#### 16.2.1 Client (`/package.json`, `"type": "module"`, name `dashboard-studio` v1.0.0)
- **Dependencies:**
  - react / react-dom ^18.3.1
  - react-router-dom ^6.24.0
  - @mui/material and @mui/icons-material ^5.15.20
  - @emotion/react ^11.11.4, @emotion/styled ^11.11.5
  - **echarts ^6.1.0** (6.1.0 is installed)
  - **html-to-image ^1.11.13**
- **Dev dependencies:** vite ^5.3.1, @vitejs/plugin-react ^4.3.1, tailwindcss ^3.4.4, postcss ^8.4.38, autoprefixer ^10.4.19, concurrently ^10.0.3.
- **Scripts:**
  - `dev`: `concurrently -k -n api,web -c blue,magenta "npm run server" "npm run web"`. This runs the API and the web app together; `-k` kills both if either exits.
  - `web`: `vite`
  - `build`: `vite build`
  - `preview`: `vite preview`
  - `server`: `npm --prefix server run dev`
  - `server:install`: `npm --prefix server install`
- **`vite.config.js`:** react plugin, `server.port: 5174`, `proxy: { '/api': 'http://localhost:4100' }`.
- **`tailwind.config.js`:**
  - `content: ['./index.html','./src/**/*.{js,jsx}']`, `darkMode: 'class'`, and `corePlugins.preflight: false` (so Tailwind's reset does not fight MUI's baseline).
  - Extended colors map to CSS variables: `ink`→`var(--ink)`, `ink2`→`var(--ink-2)`, `muted`, `hairline`, `graybg`→`var(--gray-bg)`, `surface`, `blue`.
  - `fontFamily.sans`: `-apple-system, BlinkMacSystemFont, 'SF Pro Display', Inter, system-ui, sans-serif`.
  - `maxWidth.content: '1180px'`.
- **`postcss.config.js`:** tailwindcss + autoprefixer.
- **`index.html`:**
  - `<meta name="theme-color" content="#ffffff">`; the app changes it at runtime.
  - favicon `/favicon.svg`: a 32×32 rounded square with a gradient from #2a78d6 to #9f4bd6 and three white rising bars.
  - Preconnects to Google Fonts and loads Inter at weights 400–900.
  - Title: `Dashboard Studio — build dashboards from your own data.`
  - Mounts at `#root` with `/src/main.jsx`.

#### 16.2.2 Server (`/server/package.json`, `"type": "module"`, name `dashboard-studio-server`)
- **Dependencies:** express ^4.19.2, cors ^2.8.5, morgan ^1.10.0, multer ^1.4.5-lts.1, **openai ^6.48.0**, **xlsx ^0.18.5** (SheetJS).
- **Scripts:**
  - `start`: `node --env-file-if-exists=.env index.js`
  - `dev`: `node --env-file-if-exists=.env --watch index.js`
  - The `--env-file-if-exists` flag needs a recent Node (22.9 or later); v26.4.0 is installed on this machine.
- **Environment variables** (`server/.env.example`):
  - `PORT`: default 4100.
  - `OPENAI_API_KEY`: optional. Turns on OpenAI answers. Without it, the wizard and local Q&A still work fully.
  - `OPENAI_MODEL`: default `gpt-5`.
  - `.env` is git-ignored; `.env.example` is committed.
- **`.gitignore`:** node_modules, dist, .DS_Store, \*.local, `.env`, `.env.*` (but not `.env.example`), `server/.env*`, `server/node_modules`, `server/data`.

#### 16.2.3 Running it
```
npm install && npm run server:install
npm run dev    # API :4100 + client http://localhost:5174
```
For production the README says to run `npm run build` (client into `dist/`) and then `npm --prefix server start`. The Express server does **not** serve `dist/`, so production needs a separate static host that proxies `/api`.

### 16.3 Data model

#### 16.3.1 Dataset metadata (stored in `data/datasets/index.json`; also what `GET /api/datasets` returns)
```js
{ id: uuid, name: string, fileName: string, sheet: string, sheetNames: string[],
  columns: Column[], rows: number /* count only */, createdAt: epochMs }
```
#### 16.3.2 Column
```js
{ id: 'snake_case_id', name: 'Original Header', type: 'number'|'date'|'category',
  format: 'money'|'percent'|'number'|'date'|'text', distinct: number /* distinct non-null values within a 2000-row sample */,
  sample: [first 3 sampled values] }
```
#### 16.3.3 Rows
Stored in `data/datasets/<id>.rows.json`. Each row is an object keyed by column id. Values are a number, a string, an ISO `YYYY-MM-DD` date string, or `null`.

#### 16.3.4 Dashboard (`data/dashboards.json` holds an array)
```js
{ id: uuid, name: string (≤80, unique case-insensitively), widgets: Widget[], filters: Filter[] /* always [] from the client */,
  createdAt: epochMs, updatedAt?: epochMs }
```
#### 16.3.5 Widget (the complete field set)

| field | type | meaning |
|---|---|---|
| `id` | string | Unique within the dashboard. Client ids look like `w-<base36 time>-<4 random>`; server-healed ids look like `w-<8 hex>`. |
| `datasetId` | uuid | The bound dataset (always the dashboard's dataset). |
| `type` | chart id | One of the 17 types in §16.10. |
| `title` | string | Display title. |
| `x` | colId | Dimension, category or ordered X; numeric X for scatter/bubble. |
| `y` | colId or colId[] | Measure(s). An array for bar/hbar/line/area/radar; a single id for pie/donut/treemap/funnel/heatmap/scatter/bubble. |
| `group` | colId | Split-by, color-by, or box group. For heatmap it is the **Y category**. |
| `size` | colId | Bubble size. |
| `field` | colId | Single measure for kpi/gauge/histogram/box. |
| `cols` | colId[] | Table columns. |
| `agg` | `sum\|avg\|count\|countDistinct\|min\|max\|median` | Default `sum`. |
| `bins` | int 3–40 | Histogram bin count (default 12). |
| `stacked` | bool | Stack bar/hbar/area series. |
| `sort` | `desc\|asc\|none` | Default `desc`. |
| `limit` | number or `''` | Top N. `''` means all. |
| `refLine` | `'none'\|'avg'\|'median'\|number` | Reference line. |
| `threshold` | `{ value:number, direction:'above'\|'below' }` or `null` | Green/red target. |
| `colors` | `{ [colorKey]: paletteColorId }` | Per-series or per-slice color. |
| `w` | 1\|2\|3 | Grid width: S = 1/3, M = 2/3, L = full. |
| `pinned` | bool | Shown in the "Pinned" top row. |
| `disabled` | bool | Hidden in the Viewer and dimmed in the Builder. |

#### 16.3.6 Snapshot (`data/snapshots/index.json` holds an array)
`{ id: uuid, dashboardId, label (≤120 chars, default 'Snapshot'), file: '<id>.png', createdAt }`

#### 16.3.7 Upload job (in memory only)
`{ id: uuid, fileName, status: 'parsing'|'ready'|'error', progress: 0–100, phase: string, datasetId: uuid|null, error: string|null, createdAt }`

#### 16.3.8 Filter (dormant)
`{ col, op: 'in'|'between'|'dateBetween', value }`

### 16.4 Server persistence (`server/services/store.js`, `snapshots.js`)

- **Data root:** `server/data/` (resolved from the service file location as `path.join(__dirname,'..','data')`). The folders `data/datasets/` and `data/snapshots/` are created at module load with `mkdirSync({recursive:true})`.
- **Files:**
  - `data/datasets/index.json`: array of dataset metadata, columns included, rows excluded.
  - `data/datasets/<id>.rows.json`: the row array (the large part).
  - `data/dashboards.json`: array of dashboards.
  - `data/snapshots/index.json`: snapshot records.
  - `data/snapshots/<id>.png`: snapshot images.
- **I/O:** all reads and writes are synchronous `fs.readFileSync` / `writeFileSync` of compact JSON (no pretty-printing). `readJson(file, fallback)` returns the fallback on any read or parse error.
- **Row cache:** a module-level `Map(id → rows[])`. `addDataset` fills it. `getDatasetRows` loads rows from disk lazily and caches them. Nothing is evicted except on delete.
- **Store functions:**
  - `listDatasets()`: the index, sorted by `createdAt` descending (newest first).
  - `getDatasetMeta(id)`.
  - `getDataset(id)` → `{name, columns, rows}`. It has no `id` field, and returns `null` if either the metadata or the rows are missing.
  - `addDataset({name,fileName,columns,rows,sheet,sheetNames})`: new uuid; writes the rows file, caches it, appends to the index, and returns the metadata.
  - `deleteDataset(id)`: drops the cache entry, unlinks the rows file, removes the index entry. **No cascade** to dashboards.
- **Dashboard rules:**
  - `nameTaken(name, exceptId)` compares trimmed, lowercased names.
  - `normalizeWidgets(widgets)` guarantees a unique id per widget. A missing or duplicate id is replaced with `w-${randomUUID().slice(0,8)}`. Non-arrays become `[]`.
  - `createDashboard({name, widgets=[], filters=[]})`: trimmed name, default `'Untitled dashboard'`, cut to 80 chars. Throws `A dashboard named “X” already exists.` if the name is taken.
  - `updateDashboard(id, patch)`:
    - `patch.name` (string) is trimmed and cut to 80 chars. It throws `Dashboard name cannot be empty.` or the duplicate-name error.
    - `patch.widgets` is normalized. `patch.filters` must be an array, otherwise `[]`.
    - `updatedAt` is always set. Returns `null` if the id is not found.
  - `deleteDashboard(id)`.
- **Snapshots:**
  - `saveSnapshot(dashboardId, dataUrl, label)` checks the data URL against `/^data:image\/png;base64,(.+)$/`, otherwise it throws `Snapshot must be a base64 PNG data URL.` It writes `<uuid>.png` and appends the record. It does not check that the dashboard exists.
  - `listSnapshots(dashboardId)`: newest first.
  - `deleteSnapshot(id)`: unlinks the file and drops the record.
  - `deleteSnapshotsFor(dashboardId)` runs when a dashboard is deleted.

### 16.5 API reference (`server/index.js`)

**Middleware, in order:**
1. `cors()`
2. `morgan('dev')`
3. `express.json({ limit: '30mb' })` (snapshots arrive as base64)
4. `express.static(SNAP_DIR)` mounted at **`/api/snapshots/files`**, before the API routes
5. multer `{ dest: os.tmpdir(), limits: { fileSize: 500*1024*1024 } }`

**Envelope.** Success is `200 { ok: true, data }`. Failure is `{ ok: false, error: message }` with the status code noted below (default 400).

| Method & path | Request | Response `data` | Errors |
|---|---|---|---|
| `POST /api/upload` | multipart: field **`file`** (one file per request), optional text field `name` | `{ jobId }`, returned **immediately** | 400 `No file uploaded.`; 413 `File is too large (max 500 MB).` from the central handler |
| `GET /api/jobs/:id` | – | Job object (§16.3.7) | 404 `Job not found.` |
| `GET /api/datasets` | – | `DatasetMeta[]`, newest first | – |
| `GET /api/datasets/:id` | – | `DatasetMeta` | 404 `Dataset not found.` |
| `DELETE /api/datasets/:id` | – | `{ id }` (always ok, even if the id is unknown) | – |
| `POST /api/datasets/join` | JSON `{ leftId, rightId, leftKey, rightKey, type: 'inner'\|'left', name? }` | New `DatasetMeta` plus `{ truncated, matchedLeft, leftRows }` | 400 with the join error messages (§16.14) |
| `POST /api/datasets/:id/query` | JSON `{ widget, filters?: Filter[] }` | Computed payload (§16.12). If the widget is invalid: `{ kind:'empty', reason, colorKeys:[] }` with **200 ok** | 404 dataset not found; 500 on an aggregation exception |
| `GET /api/datasets/:id/suggest[?col=<colId>]` | – | `{ suggestions: [{score,label,title,reason,widget}] }` | 404 |
| `GET /api/datasets/:id/column?col=<colId>` (dormant) | – | `{id,name,type}` plus `range:{min,max}` (number), `dateRange:{start,end}` (date) or `values:[{value,count}]` (category, top 300 by frequency) | 404 dataset or `Column not found.` |
| `GET /api/dashboards` | – | `Dashboard[]` in file order | – |
| `POST /api/dashboards` | `{ name, widgets?, filters? }` | the created `Dashboard` | 400 duplicate name |
| `PUT /api/dashboards/:id` | partial `{ name?, widgets?, filters? }` | the updated `Dashboard` | 404 `Dashboard not found.`; 400 empty or duplicate name |
| `DELETE /api/dashboards/:id` | – | `{ id }`; also deletes all its snapshots | – |
| `GET /api/dashboards/:id/snapshots` | – | `Snapshot[]`, newest first | – |
| `POST /api/dashboards/:id/snapshots` | `{ image: 'data:image/png;base64,…', label? }` | `Snapshot` | 400 bad data URL |
| `DELETE /api/snapshots/:id` | – | `{ id }` | – |
| `GET /api/snapshots/files/<file>.png` | static | PNG bytes | static 404 |
| `GET /api/assistant/status` | – | `{ ai: true, provider: <model> }` or `{ ai: false, provider: 'built-in assistant' }` | – |
| `POST /api/assistant` | `{ message, history?: [{role,content}], flow?: Flow\|null, datasetId? }` | `{ reply, source:'builder'\|'ai'\|'local', suggestions:string[], flow?, options?, created?, link?, note? }` | 500 |
| `GET /api/health` | – | `{ up: true }` | – |
| any other `/api/*` | – | – | 404 `{ ok:false, error:'Unknown endpoint <METHOD> <url>' }` |

**Central error handler.** It always responds with JSON. Status is `err.status`, or 413 for multer's `LIMIT_FILE_SIZE`, otherwise 500. It logs `API error: <msg>` and only writes a response if headers have not been sent yet.

### 16.6 Upload → async job pipeline

#### 16.6.1 Server
1. Multer stores the file in the OS temp dir.
2. The dataset name is `(req.body.name || originalname || 'dataset')` with its last extension removed (`/\.[^.]+$/`).
3. `createJob(originalname)` creates `{status:'parsing', progress:0, phase:'Queued'}`, and the response returns `{jobId}`.
4. In `setImmediate`:
   - `parseFileAtPath(tmpPath, originalname, onProgress)` runs, where `onProgress(phase, pct)` calls `updateJob`.
   - `addDataset(...)` saves the result.
   - The job is updated to `{status:'ready', progress:100, phase:'Ready', datasetId}`.
   - On an exception the job becomes `{status:'error', phase:'Failed', error: e.message}`.
   - The temp file is always unlinked.
5. **Progress phases**, in order:

   | phase | progress |
   |---|---|
   | Queued | 0 |
   | Reading file | 8 |
   | Extracting rows | 30 |
   | Normalizing values | 55; for files over 20,000 rows it repeats every 20,000 rows as 55 + round(r/total × 25), up to about 80 |
   | Inferring column types | 88 |
   | Finalizing | 98 |
   | Ready | 100, or Failed on error |

6. **Job reaper:** every 60 s (an `unref`'d interval) it deletes jobs whose status is not `parsing` and that are older than 10 minutes.

#### 16.6.2 Client (`DatasetsContext.uploadFiles`)
1. Files are uploaded **one request per file**, one after another. Each becomes its own job and its own dataset.
2. `pollJob` adds an `uploads` entry `{jobId, fileName, status:'parsing', progress:0, phase:'Queued'}`, then polls `GET /api/jobs/:id` **every 500 ms** and merges the job fields into the entry.
3. When the status leaves `parsing`, polling stops.
   - **On `ready`:** `refresh()` reloads the dataset list, `activeId` is set to the new `datasetId`, and the tile is removed **1200 ms** later.
   - **On `error`:** the tile stays and shows the error.
   - **On a polling network error:** the entry becomes `{status:'error', error}`.
4. If the upload request itself fails, an error tile is added with id `err-<fileName>-<n>`. If the error was "offline", `apiOffline` is set.

### 16.7 Parsing (`server/services/parse.js`)

- **Read:** `XLSX.readFile(path, { cellDates: true, dense: true })`. One SheetJS path covers CSV, TSV, TXT, XLSX, XLS and ODS. **Only the first sheet** is used, and `sheetNames` and `sheet` are recorded.
- **Errors:**
  - `The file has no readable sheet.`
  - `The sheet is empty.` (no rows at all)
  - `The sheet has a header but no data rows.`
- **Rows:** `sheet_to_json(ws, { header: 1, defval: null, blankrows: false, raw: true })` gives an array of arrays. Row 0 is the header.
- **Header names:** a blank or null header becomes `Column <i+1>`; others are trimmed strings.
- **Column ids (`toId`):**
  1. lowercase;
  2. replace each run of non-`[a-z0-9]` characters with `_`;
  3. trim leading and trailing `_`;
  4. if empty, use `col_<i+1>`;
  5. de-duplicate with suffixes `_2`, `_3`, …
- **Rows that are kept:** a row is skipped if every cell is null or blank. Each kept row becomes `{ [colId]: coerce(cell) }`.
- **`coerce`:** null/undefined → `null`; numbers stay as they are; a `Date` becomes `toISOString().slice(0,10)`; strings are trimmed, and an empty string becomes `null`.
- **Return value:** `{ sheetNames, sheet, columns: inferColumns(...), rows }`.

### 16.8 Type inference (`server/services/infer.js`)

- **Sample:** the first **2000 rows**.
- **`toNumber(v)`:**
  - null, undefined or `''` → null;
  - a finite number stays; a non-finite one → null;
  - strings have `$`, `,`, `%` and whitespace removed; `''` or `'-'` → null; otherwise `Number(s)` if finite.
- **`looksLikeDate(v)`** is true for:
  - a `Date` instance;
  - `^\d{4}[-/]\d{1,2}([-/]\d{1,2})?$` (YYYY-MM or YYYY-MM-DD, with `-` or `/`);
  - `^\d{1,2}[-/]\d{1,2}[-/]\d{2,4}$`;
  - `^[A-Za-z]{3,}\s+\d{1,2},?\s*\d{2,4}$` (for example "March 5, 2024").
- **Classification**, over non-null and non-empty values:
  - each value counts as dateish if it looks like a date; otherwise it counts as numeric if `toNumber` is not null;
  - **dateish ≥ 70% → `date`**; **numeric ≥ 80% → `number`**; otherwise **`category`**;
  - a column with no values at all → `category`.
- **Format** (tested against the header **name**):
  - `date` → `'date'`; `category` → `'text'`;
  - number: if the name matches `/(rate|percent|pct|%|ratio|share|margin)/i` → `'percent'` (checked first); else if it matches `/(cost|price|revenue|spend|amount|sales|budget|value|\$|usd|eur|gbp|income|profit|expense)/i` → `'money'`; else `'number'`.
- **`distinct`** is the number of distinct non-null, non-empty values **in the sample**. **`sample`** is the first 3 sampled values.
- `DATE_RE` is defined but has no effect.

### 16.9 Column-role rules shared by the server and client

- **`ROLE_TYPES`:**
  - `dimension`: `[category, date]`
  - `temporal`: `[date, category]`
  - `measure`: `[number]`
  - `anyDimension`: `[category, date, number]`
- **Categorical numbers.** The regex `CATEGORICAL_NUM_NAME` is:
  `/(^|[_\s])(id|ids|code|codes|key|keys|no|nos|num|nums|number|numbers|type|types|status|flag|flags|category|categories|group|groups|segment|segments|region|regions|zone|zones|dept|department|year|years|month|months|quarter|quarters|week|weeks|day|days)$/i`
  It is tested against both `col.id` and `col.name`.
  - `isCategoricalNumberName(col)` (server only, exported): the column is a number and the name rule matches.
  - `isCategoricalNumber(col)`: the column is a number and either the name rule matches **or** `0 < distinct ≤ 30`.
- **`colFitsSlot(col, slot)`:** true if the slot's types include the column type, **or** the slot role is `dimension`/`temporal` and `isCategoricalNumber(col)`. This lets numeric codes, low-cardinality numbers and years fill X, group and slice slots.
- **`isIdentifier(col, rowCount)`** (`suggest.js`) decides which numbers are not measures for suggestions, auto-dashboards and the plain-English parser. It is true if the name rule matches, **or** the column is a number and `distinct ≥ max(50, rowCount×0.95)`.
- **`smartDefaultAgg(col)`:** returns `avg` if the format is `percent` or `"<id> <name>"` lowercased matches `/(rate|ratio|roi|score|avg|average|mean|percent|pct|margin|index)/`. Otherwise `sum`. `defaultAgg(type)` always returns `'sum'`.
- **`AGGREGATIONS`**, with labels: `sum` Sum, `avg` Average, `count` Count (rows), `countDistinct` Count (distinct), `min` Min, `max` Max, `median` Median.

### 16.10 Chart-type catalog

The registry lives in `server/services/chartTypes.js`, and `src/lib/chartTypes.js` is effectively identical. Each slot is `{key,label,role,types,multi,required(default true),aggregated,hint}`.

| id | name | group | desc | slots (key: label, role, flags, hint) | flags |
|---|---|---|---|---|---|
| `kpi` | KPI tile | Core | One headline number | `field`: "Metric", measure, aggregated. "A numeric column to summarize into a single figure." | aggregates |
| `gauge` | Gauge | Core | One value against a target, as a dial | `field`: "Metric", measure, aggregated. "A numeric column summarized to a single value shown on the dial." | aggregates |
| `bar` | Bar chart | Core | Compare a measure across categories | `x`: "X · category", dimension ("The category or date to compare across."); `y`: "Y · measure(s)", measure, **multi**, aggregated ("Numeric column(s) aggregated for each X."); `group`: "Split by (optional)", dimension, optional ("Optionally split each bar into a series per category.") | aggregates, supportsStack |
| `hbar` | Horizontal bar | Core | Bars laid on their side — good for long labels | `x`: "Category", dimension ("The category to compare (drawn down the Y axis)."); `y`: "Measure(s)", multi, aggregated; `group`: "Split by (optional)", optional | aggregates, supportsStack |
| `line` | Line chart | Core | Trend of a measure over an ordered axis | `x`: "X · time / order", **temporal** ("A date or ordered category for the trend."); `y`: "Y · measure(s)", multi, aggregated; `group`: optional split | aggregates (no stack) |
| `area` | Area chart | Core | Filled trend over an ordered axis | `x` temporal; `y` multi, aggregated; `group` optional | aggregates, supportsStack |
| `pie` | Pie chart | Core | Share of a whole | `x`: "Slices · category", dimension ("The category that forms the slices."); `y`: "Value · measure", single, aggregated ("Numeric column summed per slice.") | aggregates, colorByCategory |
| `donut` | Donut chart | Core | Share of a whole (with a hole) | `x`: "Slices · category"; `y`: "Value · measure", aggregated | aggregates, colorByCategory |
| `table` | Table | Core | Raw rows of records | `cols`: "Columns", anyDimension, multi ("Pick the columns to show.") | – |
| `histogram` | Histogram | Distribution | How one numeric column is distributed | `field`: "Numeric column", measure ("The column whose distribution to bin.") | bins |
| `box` | Box plot | Distribution | Min / quartiles / max, optionally per group | `field`: "Numeric column" ("The column to summarize."); `group`: "Group by (optional)", dimension, optional ("One box per category.") | colorByCategory |
| `scatter` | Scatter plot | Correlation | Two numeric columns against each other | `x`: "X · numeric", measure; `y`: "Y · numeric", measure ("A DIFFERENT numeric column — related to X — for the vertical axis."); `group`: "Color by (optional)" | **raw**, colorByCategory |
| `bubble` | Bubble chart | Correlation | Scatter with a third measure as size | `x`, `y` ("A different numeric column, related to X."), `size`: "Size · numeric" ("A numeric column that sets each bubble’s size."), `group`: "Color by (optional)" | raw, colorByCategory |
| `radar` | Radar chart | Extras | A measure across a few categories, as spokes | `x`: "Spokes · category" ("A low-cardinality category (a handful of values)."); `y`: "Measure(s)", multi, aggregated | aggregates |
| `treemap` | Treemap | Extras | Nested rectangles sized by a measure | `x`: "Tiles · category"; `y`: "Size · measure", aggregated | aggregates, colorByCategory |
| `funnel` | Funnel | Extras | Stages narrowing by a measure | `x`: "Stages · category" ("The stages, largest to smallest."); `y`: "Value · measure", aggregated ("Numeric value aggregated per stage.") | aggregates, colorByCategory |
| `heatmap` | Heatmap | Extras | A measure across two categories, as a color grid | `x`: "X · category" ("Columns of the grid."); `group`: "Y · category", **required** ("Rows of the grid — a second category."); `y`: "Value · measure", aggregated ("Numeric value aggregated per cell.") | aggregates |

- **Groups**, in order: Core, Distribution, Correlation, Extras.
- **Types that support a threshold:** `['kpi','gauge','bar','hbar','line','area','heatmap']`.
- **Editor-only lists:**
  - `SORTABLE = ['bar','hbar','pie','donut','treemap','radar','line','area']`
  - `REFLINEABLE = ['bar','hbar','line','area']`

### 16.11 Validation and "relatability"

- **`validateWidget(widget, columns)` → `{valid, reason}`.** For each slot:
  - A value is present if, for a multi slot, it is a non-empty array, or for a single slot, it is truthy.
  - Missing and required: ``Choose ${label minus ' (optional)'}.`` (for example "Choose X · category.").
  - Each id must exist (``Column "<id>" is not in this dataset.``) and must fit the slot via `colFitsSlot` (``"<name>" isn't a valid <label>.``).
  - Raw types (scatter/bubble) with `x === y` fail with `Pick two different numeric columns for X and Y.`
  - An unknown type fails with `Unknown chart type.`
- **`compatibleColumns(typeId, slotKey, columns, exclude=[])`:** the columns that fit the slot, minus excluded ids. The editor passes `exclude` only for raw charts: the values currently chosen in the other `x`/`y`/`size` slots, so a column is never plotted against itself.
- **`relatabilityHint(typeId, slotKey)`:** the slot's `hint`, or a generated "Choose a numeric / date or ordered / category or date column (one or more)."
- `/query` validates on the server and returns a `kind:'empty'` payload with the reason when invalid. The editor validates on the client to enable Save and show the preview message.

### 16.12 Aggregation engine (`server/services/aggregate.js`)

**Entry point.** `computeWidgetData(widget, {columns, rows})` returns a payload tagged with `kind`, always including **`colorKeys`**: the exact keys that need a palette color. A missing chart definition or dataset returns `{kind:'empty', colorKeys:[]}`. `agg` defaults to `'sum'`.

**`/query` order of operations:** validate, then `applyFilters(rows, filters)` (dormant, §16.13), then compute.

#### 16.12.1 Caps
| constant | value | applies to |
|---|---|---|
| `MAX_BARS` | 50 | bar/hbar/line/area category count |
| radar cap | 12 | radar spokes |
| `MAX_SLICES` | 12 | pie, donut, funnel |
| `MAX_TREEMAP` | 14 | treemap |
| `MAX_GROUP_SERIES` | 12 | split series, box groups, scatter groups |
| `MAX_POINTS` | 4000 | scatter/bubble sampling |
| `MAX_HEATMAP` | 20 | per heatmap axis |
| table rows | 200 | table |

#### 16.12.2 `reduceAgg(values, agg)`
| agg | result |
|---|---|
| `count` | `values.length`: every row, **nulls included** |
| `countDistinct` | size of the set of raw values excluding null and `''` |
| `avg` | mean of the numeric values (via `toNumber`), rounded to 3 decimals |
| `min` / `max` | min / max of the numeric values, not rounded |
| `median` | middle value; for an even count, the mean of the two middle values rounded to 3 decimals |
| `sum` (default) | sum of the numeric values, rounded to 3 decimals |

Any other aggregation with no numeric values returns **0**. Quantiles use linear interpolation: `pos = (n-1)·q`.

#### 16.12.3 Grouping and sorting helpers
- **`groupByKey(rows, key)`:** a Map from the value as a string to its rows. null, undefined and `''` all become `'—'`.
- **`topKeys(rows, key, cap)`:** keys ranked by row count, descending, capped.
- **`sortKeys(keys, {date, valueOf, dir})`:**
  - if X is a **date column**, keys are always sorted ascending chronologically (`new Date(a) - new Date(b)`) and `dir` is ignored;
  - `dir:'none'` keeps insertion order (the order values first appear);
  - otherwise keys are sorted by `valueOf` descending, and reversed for `asc`.

#### 16.12.4 Output by kind
1. **KPI** (`type:'kpi'`): `{kind:'kpi', value: reduceAgg(all rows[field]), label: widget.title || colName(field), format: colFmt(field), colorKeys:[]}`.
2. **Gauge**: the same with `kind:'gauge'`.
3. **Table**:
   - columns are `widget.cols`, or every column if `cols` is empty, given as `{id,name,format}`;
   - `rows: rows.slice(0,200)`, which contains **every field** of each row, not just the selected ones;
   - `total: rows.length`, `colorKeys: []`.
4. **Histogram** (`kind:'bins'`):
   - uses the numeric values of `field`; returns empty if there are none;
   - `bins = clamp(widget.bins||12, 3, 40)`, `width = (max-min)/bins || 1`;
   - index = `floor((n-min)/width)`, clamped to `[0, bins-1]`, so the last bin includes the maximum;
   - `data: [{label:'<fmtEdge(x0)>–<fmtEdge(x1)>', count, x0, x1}]`, `field: colName`, `colorKeys:['count']`;
   - `fmtEdge`: if |n| ≥ 1000, `${round(n/100)/10}k`; an integer → the string; otherwise `toFixed(1)` if |n| < 10, else `toFixed(0)`.
5. **Box** (`kind:'box'`):
   - groups come from `group`, or there is a single group named after the column;
   - each group gives `{name, count, min, q1, median, q3, max}` from its sorted numeric values; empty groups are skipped;
   - capped to the **first 12 groups in first-seen order** (not by frequency);
   - `format`, `colorKeys` = the group names.
6. **Scatter/bubble** (`kind:'points'`, raw, no aggregation):
   - points are `{x, y, size?, group?}`, skipping rows where x or y is not numeric;
   - `size` falls back to `?? 0`; `group` is `'—'` for null or empty;
   - if there are more than 4000 points, sample evenly: `pts[floor(i·n/4000)]`, and set `truncated: true`;
   - `groups` = the first 12 distinct group values;
   - also returns `hasSize`, `xName`, `yName`, `sizeName`, `xFormat`, `yFormat`;
   - `colorKeys` = `groups`, or `['points']` if there is no group.
7. **Pie/donut/treemap/funnel** (`kind:'parts'`):
   - `parts` = for each x value, `{name, value: reduceAgg(group[y])}`;
   - sorted by value: ascending if `sort==='asc'`, **otherwise descending, including `'none'`**;
   - `capBase` = 14 for treemap, else 12; `cap = limit>0 ? min(limit, capBase×3) : capBase`;
   - if the direction is not `asc` and there are more parts than the cap: keep the top `cap-1` and add `{name:'Other', value: round(sum of the rest)}`;
   - if the direction is `asc` and there are more parts than the cap: just cut to `cap`;
   - `format` = the y column's format, `colorKeys` = the part names.
8. **Heatmap** (`kind:'heatmap'`):
   - `xs` = top 20 x values by frequency, `ys` = top 20 `group` values;
   - cells are bucketed with the key `` `${xv} ${yv}` `` and later split on a space (see the bug in §16.29);
   - each cell's value is `reduceAgg(cell[y])`;
   - returns `{xs, ys, cells:[{x,y,value}], min, max (0 if empty), format, colorKeys:[]}`.
9. **Series** (bar/hbar/line/area/radar, `kind:'series'`):
   - `yIds` = the de-duplicated y array; returns empty if `x` is missing, or there are no y ids and `agg ≠ count`;
   - `cap` = 12 for radar, else 50; `limit = limit>0 ? min(limit, cap) : cap`; `dir = sort || 'desc'`;
   - **Split mode** (a `group` is set and there are ≤ 1 measures):
     - `groupVals` = the top 12 group values by frequency;
     - X keys are sorted by **row count** per x, using the requested direction (dates always ascending), then cut to the limit;
     - each data row is `{x, [groupVal]: reduceAgg(subgroup[yId] ?? (count?1:0))}`;
     - `series` = `[{key: gv, name: gv, format: y's format}]`.
   - **Measure mode:**
     - X keys are sorted by the primary measure's aggregated value, then cut to the limit;
     - each data row is `{x, [yId]: reduceAgg(...)}`; if `agg==='count'` and there are no y ids, `count = rows`;
     - `series` = `[{key: yId, name: colName, format}]`, or `[{key:'count', name:'Count', format:'number'}]`.
     - With **several measures, `group` is ignored**.
   - Returns `{kind:'series', chart: type, data, xKey:'x', xIsDate, series, xFormat:'text', stacked: !!widget.stacked, colorKeys: series keys}`.

#### 16.12.5 Client query cache (`useWidgetQuery`)
- **Key:** a module-level `Map`, keyed by `JSON.stringify([datasetId, type, x, y, group, size, field, cols, agg, bins, stacked, sort, limit, filters])`. **Title, colors, threshold and refLine are left out of the key** because they only affect display.
- **De-duplication:** an in-flight Map so identical queries are sent only once.
- **Debounce:** 200 ms by default; 300 ms in the editor preview.
- **Returns** `{computed, loading, error}`.
- `invalidateDataset(id)` exists but nothing calls it.

### 16.13 Filters (dormant; `server/services/filters.js`)

- `applyFilters(rows, filters)` keeps only filters that have both `col` and `op`, and requires **every** filter to match (AND).
- **Operators:**
  - `in`: the value array of allowed strings. An empty or missing array lets everything through. A null or empty cell is compared as `'—'`.
  - `between`: `[lo|null, hi|null]`, inclusive, via `toNumber`. A non-numeric cell fails.
  - `dateBetween`: `[startISO|null, endISO|null]`, inclusive, via `Date.parse`. An unparseable cell fails.
  - An unknown op lets the row through.
- **Helpers:**
  - `distinctValues(rows, col, cap=300)` → `[{value,count}]`, by frequency descending;
  - `numericRange` → `{min,max}` or null;
  - `dateRange` → `{start,end}` (ISO dates) or null.
- **Status.** The filter-bar UI and click-to-cross-filter were built and then removed. The `/query` `filters` parameter and `/column` still exist on the server. The client always sends `filters: []`, and dashboards persist `filters: []`. `EChart` still accepts an unused `onClick` prop left over from cross-filtering.

### 16.14 Joins (`server/services/join.js`)

`joinDatasets({leftId, rightId, leftKey, rightKey, type='inner', name})`:

1. **Checks**, each with its error message:
   - `Left dataset not found.` / `Right dataset not found.`
   - `Pick two different datasets to join.`
   - `Left key column not found.` / `Right key column not found.`
2. **Right-hand columns:** every right column **except the right key** is brought in. If its id collides with a left column id:
   - the id becomes `${id}__${slug(R.name)}`, where `slug` lowercases, replaces each run of non-alphanumerics with `_`, trims `_`, takes the first 12 characters, and falls back to `'r'`;
   - the name becomes `${name} (${R.name})`.
3. **Index:** the right rows are indexed by `String(row[rightKey] ?? '')`.
4. **Row loop:** for each left row, look up `String(lr[leftKey] ?? '')`.
   - If it matches, emit one row **per match** (one-to-many): a copy of the left row plus the mapped right fields. `matchedLeft` counts the left rows that matched.
   - If there is no match and `type==='left'`, emit the left row with null right fields. Any type other than `'left'` behaves as inner.
   - Stop at **300,000 rows** and set `truncated:true`.
5. **No rows:** throw `The join produced no rows — the key columns don’t share any values.`
6. **Columns** are re-inferred with `inferColumns` over the joined rows, in left-then-right order.
7. **Save:** `addDataset({ name: name.trim() || `${L.name} + ${R.name}`, fileName: `join:${L.name}+${R.name}`, sheet:'joined', sheetNames:['joined'] })`.
8. **Return** the metadata plus `{truncated, matchedLeft, leftRows}`. The extra fields are not saved.

The result is an ordinary dataset, so query, suggest and the assistant all work on it.

### 16.15 Chart suggestions (`server/services/suggest.js`, `GET /api/datasets/:id/suggest`)

#### 16.15.1 Column buckets
- `dates`: date columns.
- `allNums`: number columns.
- `catNums`: numbers where `isIdentifier` is true.
- `cats`: category columns plus `catNums`.
- `measures`: numbers that are not identifiers. `M` = measures, or `allNums` if there are no measures.
- `lowCats`: cats with `2 ≤ distinct ≤ 20`, sorted by distinct ascending. `bestCat` = the first one, i.e. the lowest cardinality.
- `countY` = `measures[0] || allNums[0]`. It fills the value slot for "count of records" widgets; its values are ignored.

Each candidate is `{score, label, title, reason, widget}`. The widget has no title; the client adds one.

#### 16.15.2 Focus mode (`?col=`; the client never uses it)
- **Date column:** 100 Line "`M0` over `c`".
- **Category column with distinct ≥ 2:**
  - 100 Bar "`M0` by `c`";
  - if distinct ≤ 12: 96 Donut "Share by `c`" with agg count;
  - if there is a second low-cardinality category: 90 Heatmap "`M0`: `c` × `c2`".
- **Number column:**
  - 100 KPI "Average/Total `c`" with the smart default aggregation;
  - 94 Histogram "Distribution of `c`" (12 bins);
  - 92 Bar by `bestCat`;
  - 90 Line over `dates[0]`;
  - 88 Scatter "`c` vs `o`".

#### 16.15.3 Dataset-wide candidates
- **96** Line "`m0` over time" if there are dates and measures. Otherwise **94** Line "Records over time" with agg count, if there are dates and `countY`.
- **92** Bar "`m0` by `bestCat`". Otherwise **90** Bar "Records by `bestCat`" with count.
- **82** Donut "Share by `c`", where `c` is the first low-cardinality category with ≤ 8 values, or `bestCat`; agg count.
- **70** Scatter "`m0` vs `m1`" if there are ≥ 2 measures.
- **68, 67, 66** KPI "Total/Average `m`" for the first 3 measures, then **58** Histogram of `m0`.
- If there are no measures:
  - **66, 65** KPI "Unique `code`" with countDistinct for the first 2 `catNums`;
  - **60** KPI "Total records" with count.

#### 16.15.4 Output
In focus mode, keep only candidates that use the column (in `x`, `field`, `group` or `y`). Then:
1. keep only candidates that pass `validateWidget`;
2. sort by score, descending;
3. de-duplicate on `type|x|JSON(y)|field|group`;
4. keep **6** (or **4** in focus mode).

Reasons are plain-English sentences, for example: “region” groups the rows and “revenue” is a measure — compare them with a bar chart.

### 16.16 AI assistant (server)

#### 16.16.1 Turn routing (`assistant.js → assistantTurn({message, history=[], flow=null, datasetId=null})`)
1. If `flow.active`, return `{source:'builder', suggestions, ...continueBuild(msg, flow)}`.
2. Else if `isBuildIntent(msg)`, return `{suggestions, ...startBuild()}`.
3. Else, if a key is set, try `askOpenAI`; on failure, record `note = err.message` in the response.
4. If there is no reply yet, use `localAnswer` with `source:'local'`.

The response always includes `suggestions: ['Build me a dashboard','What can you chart from my data?','What dashboards do I have?']`. The client ignores them and shows its own starter prompts.

**Status endpoint:** `{ai:true, provider: OPENAI_MODEL||'gpt-5'}` or `{ai:false, provider:'built-in assistant'}`.

#### 16.16.2 OpenAI (`askOpenAI`)
- The `openai` SDK is loaded with a dynamic `import('openai')`. `new OpenAI()` reads `OPENAI_API_KEY` from the environment.
- **Call:** `client.chat.completions.create({ model: MODEL(), max_completion_tokens: 1024, messages })`. No temperature is passed, because GPT-5 only allows the default.
- **Messages:**
  - the system prompt;
  - the **last 6** history items, mapped to role `assistant`/`user`;
  - a user message built as `"CONTEXT:\n<buildContext()>" + optional "\n\nCOMPUTED: <metric sentence>" + "\n\nQUESTION: <message>"`.
- **Reply:** `choices[0].message.content.trim()`.
- **System prompt (verbatim):** "You are the assistant inside "Dashboard Studio", a no-code tool where people upload CSV/Excel files and build charts. You know the user's datasets and the dashboards/visualizations they've built (given as CONTEXT). Answer ONLY from that context and the numbers provided — never invent column names or values. Be concise and concrete. When the user wants a new chart/dashboard, tell them to say "build a dashboard" so the guided builder runs. Keep answers under ~110 words."
- **`buildContext()`:**
  - for each dataset: `Dataset "<name>" — N rows.` followed by lines `numeric: …`, `categories: …`, `dates: …`;
  - then `DASHBOARDS:`, with one line per dashboard: `Dashboard "<name>" (N widgets on “<dataset>”): <describeWidget…; …>`;
  - the dataset is taken from the first widget.
- **`describeWidget`:**
  - kpi → "KPI of X (agg)"
  - table → "a table of records"
  - histogram / box → "histogram of X" / "box plot of X"
  - pie / donut / treemap → "<type> of Y by X"
  - scatter / bubble → "<type> of X vs Y"
  - heatmap → "heatmap of Y across X × G"
  - anything else → "<type> chart of Y1 & Y2 by X"

#### 16.16.3 Local responder (`localAnswer`, no key needed)
**Matching.** `norm` lowercases and turns `_` and whitespace into a single space. `mentions(h, p)` requires `p.length ≥ 3` and a whole-word regex match `\bphrase\b`. `findColumn` matches a column's normalized name or id as a whole word; the longest match wins, with +100 for numeric columns when `preferNumeric`. `findAnyColumn` tries the active dataset first, then the rest.

**Aggregation words.** `average/avg→avg`, `sum/total→sum`, `maximum/highest→max`, `minimum/lowest→min`, `median`, `count/'how many'→count`. "mean" is deliberately left out so that "what does X mean" is not treated as an average.

**Decision order:**
0. No datasets: "You haven't uploaded any data yet. Open the Builder and drop in a CSV or Excel file…"
1. `/\b(join|combine|merge|relate|link)\b/` → `joinHelp`. It says how to use Builder → Join. If a column was mentioned, it lists the datasets that contain it (same id or same normalized name); if only one dataset has it, it says so.
2. A column is mentioned, the message reads like a definition question (`what is|what does|what's|meaning|mean|means|explain|describe|define|about|stands for|which|what`), and there is no aggregation word → `describeColumn`, built from `columnProfile` over **all rows**:
   - number: distinct count, min–max, average, plus "It looks like an identifier…" if the id matches `/(^|_)(id|code|no|number|key|uuid)s?$/i`;
   - date: span start → end and distinct count;
   - category: distinct count and the top 4 sample values.
3. `computeMetric`: the first aggregation word found; then the best column (numeric preferred unless counting) across datasets, active first. The answer is "The <average|total|max|…> of **Col** is **value** (from “Dataset”, N rows)."
   - `fmtVal` rounds to 2 decimals and adds `$` for money or `%` for percent.
   - Count here excludes nulls (unlike the engine's count).
4. A dashboard name is mentioned → "**Name** shows N visualizations: …".
5. `dashboards?|visuali\w*|widgets?` together with `what|which|list|have|my|how many|all` → a list of dashboards with their widget counts.
6. Column-type questions about the active dataset (or the first one):
   - `numeric|numbers?|measures?|metrics?` → the numeric columns;
   - `categor\w*|dimensions?` → the category columns;
   - `dates?|time|trends?` → the date columns;
   - `datasets?|columns?|files?|loaded|uploaded` → all datasets with row and column counts.
7. Any column mentioned → `describeColumn`.
8. A greeting (`hi|hello|hey|help|what can you|who are you`) → a capabilities line.
9. Fallback: "I couldn't map that to your data. I can: define a column…, compute a metric…, explain joins, or describe a dashboard. For open-ended questions, set an OPENAI_API_KEY…"

#### 16.16.4 Dashboard wizard (`assistantDashboard.js`)
**Flow state** round-trips through the client: `flow = { active, step: 'dataset'|'name'|'widgets', draft: { datasetId, name, widgets[], suggestions? } }`. A response can include:
- `options: [{label, value}]`: tap chips; tapping sends `value` while showing `label`;
- `created: {id, name}`;
- `link: {label:'Open my dashboard', route:'/builder?dash=<id>'}`.

**Intent detection:**
- `isBuildIntent`: the text matches `dashboard|dash board|charts?|report` **and** either `\b(build|create|make|set ?up|design|generate|new|assemble|put together|can you|could you|help me|start)\b` or `dashboard (for me|please|for us)`.
- `isCancel`: `\b(cancel|never ?mind|nevermind|forget it|abort|quit|exit)\b`. It is checked first at every step. The reply is "No problem — cancelled…" and the flow becomes inactive.
- `isDone` must match the **whole** message, after trimming trailing `.!`:
  - `done|finish(ed)?|that's it|looks good|go ahead|ship it|proceed|let's go|i'm done|finalize|all done|good to go`, or
  - `(ok(ay)?)?(yes)?(build|create|make|save|generate)( it| them| the dashboard| this| now)?`.

**`startBuild()`:**
- With no datasets, it asks the user to upload a file first and the flow stays inactive.
- With one dataset, it goes straight to `step:'name'`: "Great — I'll build a dashboard from **D** (N rows). What should we call it?" The options are the suggested name `suggestName('<D> dashboard')` and Cancel.
- With several datasets, `step:'dataset'`: "Love it — which dataset should the dashboard use?" There is one option per dataset (`"<name>  ·  N rows"`, value = id) plus Cancel.

**`suggestName(base)`** cuts to 80 chars, then if the name is taken tries `base 2` … `base 998`, and finally `base <timestamp>`.

**Dataset step.** Match by id, then by exact name ignoring case, and fall back to the newest dataset. Reply "Using **D**. What should we call the dashboard?" with a name suggestion.

**Name step.**
- `cleanName` strips `please`, `let's`, `call it` / `name it`, `the name is`, `name:`, `title:`, `it's` and surrounding quotes. Anything under 2 chars → null → `suggestName('My dashboard')`. The result is cut to 80 chars.
- If the name is taken, the reply is `A dashboard named “X” already exists — names must be unique. I can use **“X 2”**…`, with the option `Use “X 2”`.
- Otherwise `draft.widgets = autoWidgets(meta,'recommended')` and the widget list is shown.

**`autoWidgets(meta, variant)`**, using measures, identifier numbers, dates and dims (categories plus identifier numbers):
- **KPIs**, all `pinned:true`, `w:1`:
  - if there are measures: the first 4 (or 3 for `compact`), each with the smart aggregation and the title `"<Total|Average|…> <name>"`;
  - otherwise: `Unique <code>` with countDistinct for up to kpiN−1 identifier numbers, plus `Total records` with count.
- **Line** (not in compact), `w:2`: if there are dates and measures, x = the first date, y = the first 2 measures, title "`m0` over time". Otherwise "Records over time" with count.
- **Bar**, `w` = 1 if there is a date column, else 2: "`m0` by `dim0`", or "Records by `dim0`" with count.
- **Donut** (not in compact), `w:1`: "Records by `dim1||dim0`" with count.
- **Detailed only:** a scatter of `m0` vs `m1` (w 2); a histogram of `m0` with 12 bins (w 1); a table of the first 6 columns titled "Records" (w 3).
- Finally it keeps only widgets that validate, and de-duplicates on `type|x|JSON(y)|field|group|cols`.

**Default widths** (`WIDGET_WIDTH` / `WIDTHS`): kpi, gauge, pie, donut, histogram, box and radar are 1; bar, hbar, line, area, scatter, bubble, treemap and funnel are 2; heatmap and table are 3.

**`showWidgets`** replies with `<lead>\n\n**Name** — N widgets on **D**:\n1. Title\n2. …\n\nType a chart to add — e.g. **add a bar chart of <num> by <cat>** — or **remove 2**, or pick an option below. Say **build it** when you're happy.` Its options are:
- `Build it ✓ (N)` (value `done`, only if there are widgets)
- `✨ Suggest charts` (`suggest`)
- `Use recommended` (`recommended`)
- `Compact` (`compact`)
- `Detailed` (`detailed`)
- `Clear all` (`clear`, only if there are widgets)
- `Cancel` (`cancel`)

**Widgets step**, checked in this order (the message is lowercased and trimmed):
1. **`pickall` or `pick:<i>`**, using the stored `draft.suggestions`: add any suggestions not already present (by widget key), and remove the added ones from the suggestions. If suggestions remain, show them again ("Added N. Add more, or **build it**." or "That one is already in — pick another."). Otherwise show the widget list.
2. **Suggest** (`\b(suggest|suggestions?|recommend|ideas?|what (can|do) you|options)\b`):
   - run `suggestWidgets(dataset)` and map each result to a widget with `datasetId`, the default width and `title`;
   - keep only valid ones that are not already present, de-duplicate, and keep at most 6;
   - if none are left: "I don't have new suggestions… add your own with a request like “pie of sales by region”".
   - `showSuggestions` lists them with options `＋ <title>` (value `pick:<i>`), `Add all` (`pickall`, only if more than 1), `Build it ✓ (N)` and `Cancel`.
3. **Presets:**
   - `^(recommended|use recommended|default|reset)$`, or `recommended` in a message under 24 chars → the recommended set;
   - `compact`: `^(compact|make it compact|smaller|minimal)$` or a short message under 20 chars containing it;
   - `detailed`: `^(detailed|detail|more detail|comprehensive|everything)$` or a short message under 20 chars containing it.
   - Each replaces the widgets and clears the suggestions.
4. **Clear:** `^(clear|clear all|start empty|empty|remove all|delete all|start over)$`.
5. **Remove:** `\b(remove|delete|drop|take out)\b\s*#?(\d+)` removes by 1-based index ("Removed “T”."). An invalid index replies "There's no widget #N…".
6. **Done:**
   - with zero widgets: "Add at least one chart first…";
   - otherwise call `createDashboard({name, widgets})`, re-suggesting the name if it is taken. Reply `✅ Done! I built **“Name”** with N widgets:\n…\n\nOpen it to explore, recolor, or keep editing.`, include `created` and `link`, and set the flow inactive. A create error replies "I hit a snag creating it: …".
7. **Anything else** goes to `parseWidgetRequest(raw, meta)`. On error: `I couldn't build that: <error>`. If the widget is a duplicate: `“T” is already on the dashboard.` Otherwise it is added ("Added **T**.").

If the dataset was deleted mid-flow, the flow resets. An unknown step calls `startBuild()`.

#### 16.16.5 Plain-English widget parser (`assistantWidgets.js → parseWidgetRequest(message, meta)`)
- **`norm`:** lowercase, and replace every run of non-alphanumeric characters with a single space.
- **Chart type** comes from the first matching regex, in this order:
  1. hbar: `horizontal bars?|h ?bars?|sideways bars?`
  2. bar: `bar charts?|bars?|column charts?|columns?`
  3. area
  4. line: `line charts?|line graphs?|lines?|trend lines?`
  5. donut: `donut|doughnut|rings?`
  6. pie
  7. kpi: `kpis?|kpi tiles?|metric tiles?|headline numbers?|single numbers?|big numbers?|scorecards?|stat tiles?`
  8. gauge: `gauges?|dials?|speedometers?`
  9. funnel
  10. table: `tables?|grids?|raw rows?|record lists?|data table`
  11. histogram
  12. box: `box plots?|boxplots?|box and whisker`
  13. bubble
  14. scatter: `scatter ?plots?|scatters?`
  15. radar: `radar|spider charts?`
  16. treemap: `tree ?maps?`
  17. heatmap: `heat ?maps?`
- **Aggregation** comes from the first matching word group, in priority order:
  1. avg: `average|avg|mean`
  2. countDistinct: `distinct|unique|number of distinct`
  3. median
  4. min: `min|minimum|lowest|smallest`
  5. max: `max|maximum|highest|largest|peak`
  6. sum: `sum|total|combined|overall`
  7. count: `count|how many|record count|frequency`
- **`matchColumns(text, cols)`:** a space-padded whole-phrase match of each column's normalized name or id (at least 2 chars), longest match per column. A match fully contained inside a longer match is dropped (so "price" loses to "unit price"). Results are ordered by position in the text.
- **Column buckets:** `isMeasure` = a number and not an identifier; `isDimNum` = a number and an identifier; cats = categories plus dim-numbers; `countY` = the first measure, or the first number.
- **Errors and helpers:**
  - If no columns are mentioned **and** no type is named: `I couldn't find those columns in “D”. Available — numeric: …; categories: …; dates: …`.
  - `byPart` is the text after `by|per|across|grouped by|split by|for each|broken down by`. `byDim` is the first non-measure column mentioned in it.
  - `wantsVs`: `vs|versus|against|correlat|relationship|relate|compared? to`.
  - `wantsSplit`: `split|grouped|stacked|by series|broken down|segmented`.
- **Type inference when no type is named:**
  1. wantsVs and ≥ 2 measures mentioned → scatter;
  2. `distribution|spread|histogram` and any numeric mentioned → histogram;
  3. a date mentioned, or "over time", and a measure or countY exists → line;
  4. a category mentioned, or a byDim → bar;
  5. exactly one measure mentioned → kpi;
  6. otherwise → table.
- **Slot filling by type:**
  - **kpi/gauge:**
    - a measure was mentioned → `field` = it, `agg` = the stated aggregation or the smart default, title `"<AGG_LABEL> <name>"`;
    - only a code column was mentioned → that code with countDistinct, or the stated aggregation unless it is sum;
    - nothing mentioned → the first measure;
    - otherwise → countY with count, titled "Total records";
    - error: `A KPI/gauge needs a numeric column, and “D” has none.`;
    - **kpi gets `pinned:true`**; gauge does not.
  - **bar/hbar/line/area/radar:**
    - X: for line/area, the mentioned date, else the first date, else the category; for the others, category first, then date;
    - Y: all mentioned measures, with the title `"<y0 name>[ & more] <by|over> <x name>"`; or countY with count, titled `Records by|over X`;
    - `group` is set only if wantsSplit and there is a second category;
    - errors name what is available.
  - **pie/donut/treemap/funnel:** X = byDim, else the first category mentioned, else the first category. Y = the first measure ("Y by X"), or countY with count ("Records by X").
  - **scatter:** 2 measures (mentioned, or else the first two in the dataset); colored by the first category mentioned; title "A vs B".
  - **bubble:** 3 measures; title "A vs B (size: C)".
  - **histogram:** field = the first measure, else any numeric, with 12 bins; title "Distribution of F".
  - **box:** field plus an optional group (byDim, or another category); title "F by G" or "Spread of F".
  - **heatmap:** two categories; value = a measure, or count; title "M: X × G" or "Records: X × G".
  - **table:** the mentioned columns, or the first 6 columns, max 8; titled "Records".
- The result always gets `datasetId`, `type` and the default width. It is then checked with `validateWidget`; on failure, `{error: reason}`.

### 16.17 Client application shell

- **`main.jsx`:** `StrictMode` → `BrowserRouter` → `ThemeModeProvider` → `DatasetsProvider` → `App`.
- **Routes (`App.jsx`, inside `Layout`):** `/` Landing, `/dashboards` Viewer, `/builder` Builder, `*` → `Navigate('/')`.
- **`Layout`:**
  - **Offline banner:** shown when `apiOffline`, with background `#d03b3b`, white 13 px text and a CloudOff icon: "API server not reachable. Start it in a terminal: `npm run server` — then it reconnects automatically."
  - **Sticky header:** frosted, h-14, max-width 1180, z-40. It contains:
    - the logo, a 28 px rounded square with a gradient from #2a78d6 to #9f4bd6 and three white bars, plus "Dashboard Studio", linking to `/`;
    - nav pills Dashboards and Builder, where the active pill is `bg-blue text-white` and hover is `bg-black/5`, or `white/10` in dark mode;
    - a theme toggle IconButton with tooltip "Light mode" or "Dark mode".
  - `<main key={pathname} className="ds-page-enter">` remounts the page on each path change and runs a 0.28 s fade-up.
  - `<AiAssistant/>` is mounted outside `main`, so the chat survives route changes.
- **`DatasetsContext`:**
  - **State:** `datasets` (metadata only), `ready`, `apiOffline`, `activeId`, `uploads`.
  - **`refresh()`:** calls `GET /datasets`, clears the offline flag, and keeps `activeId` if it still exists, otherwise switches to the newest dataset.
  - When `apiOffline`, it retries `refresh` **every 4 s**.
  - **Exposes:** `uploadFiles`, `removeDataset(id)` (delete, then refresh), `refresh`, `dismissUpload` (never used in the UI), `activeDataset`, `getDataset(id)`, `hasData`.
- **`useDashboards()`** is a per-component hook, not a context:
  - `list`, `ready`, `refresh`;
  - `create(name, widgets, filters)`, which defaults the name to 'Untitled dashboard';
  - `update(id, patch)`, `remove(id)` (each refreshes afterwards);
  - `nameTaken(name, exceptId)`;
  - `suggest(base='New dashboard')`, which tries `base`, `base 2` … `base 998`, then `base <ts>`.
- **`lib/api.js`:**
  - `ApiError {offline, status}`.
  - Every response is read as text first:
    - an empty body with status ≥ 500 → offline error "Cannot reach the API server. Start it with “npm run server” (it listens on port 4100)." (this is what the Vite proxy returns when the server is down);
    - an empty body otherwise → `Empty response (<status>) from <path>.`;
    - non-JSON → "The API server returned a non-JSON response (<status>). Is it running correctly?";
    - `!ok` → the server's error text.
  - A failed `fetch` (network error) → the offline error.
  - JSON requests send `Content-Type: application/json`; form uploads send none.
  - **Methods:** `health`, `uploadFile(file,name)`, `job`, `datasets`, `dataset`, `deleteDataset`, `joinDatasets`, `query(datasetId, widget, filters=[])`, `columnInfo`, `suggestWidgets(datasetId, col?)`, `dashboards`, `createDashboard`, `updateDashboard`, `deleteDashboard`, `snapshots`, `saveSnapshot(dashboardId, image, label)`, `deleteSnapshot`, `snapshotUrl(file)` → `/api/snapshots/files/<file>`, `assistantStatus`, `assistant(message, history, flow, datasetId)`.

### 16.18 Rendering

#### 16.18.1 `QueryChart({datasetId, widget, debounce})`
| condition | shows |
|---|---|
| no `datasetId` | "Bind this widget to a dataset." |
| error | "Could not load: <message>" |
| loading, nothing cached | skeleton "Aggregating…" |
| `kind:'empty'` | the reason, or "Finish configuring this widget." |
| otherwise | `ChartRenderer` |

`ChartSkeleton`: six pulsing hairline-colored bars and a message; respects reduced motion.

#### 16.18.2 `ChartRenderer({widget, computed})`
- **Colors:** `colorOf(key) = pick(assignDistinct(colorKeys, widget.colors)[key], mode)`.
- **Theme tokens:** `tokensFor(mode)`.
- **Empty kind:** a centered message.
- **KPI (HTML):**
  - the title (`widget.title || c.label`) in 13 px ink2, then the value via `fmtBy(format)` at **42 px**, extrabold, tight tracking;
  - with a valid threshold:
    - met when `value ≥ target` for `above`, or `≤` for `below`;
    - color is green `#0ca30c` (dark `#30d158`) or red `#d03b3b` (dark `#ff5a5a`), applied to the value;
    - a pill reads `▲ On target · ≥ X` or `▼ Off target · ≤ X`, with the color as text on the color plus `1f` (about 12% alpha) background.
- **Table (HTML):**
  - max height 300 px, scrollable, sticky header on `bg-surface`, 13 px text, hairline row borders, tabular numbers, cells formatted with `fmtBy(col.format)`;
  - footer "Showing 200 of N rows." when truncated.
- **Every other kind** goes through `buildOption` into `<EChart option height/>`. If no option comes back: "Unsupported chart."

#### 16.18.3 `EChart` wrapper
- `echarts.init(el, null, {renderer:'canvas'})` once, a `click` handler that forwards to `onClick` through a ref, and a `ResizeObserver` that calls `resize()`. It disposes the chart on unmount.
- On every option change: `setOption(option, true)` (notMerge).
- Renders `<div style={{width:'100%', height}}>`.
- **Registered in the modular build (`lib/echartsCore.js`):** Bar, Line, Pie, Scatter, Radar, Treemap, Boxplot and Heatmap charts; Grid, Tooltip, Legend, VisualMap, DataZoom, Title and Graphic components; LabelLayout; CanvasRenderer. This list is missing some items (see §16.29).

#### 16.18.4 `buildOption(widget, computed, {mode, colorOf, tokens})` → `{option, height}`; the default height is 260
- **Shared base:**
  - `animationDuration: 320`; text in the system font, colored `tokens.ink2`;
  - `grid {left:52, right:18, top:30, bottom:40, containLabel:true}`;
  - toolbox (see below).
- **Tooltip box:** `surface` background, `cardBorder` border, padding [8,12], ink 12 px text, radius 12 px, shadow `0 4px 18px -6px rgba(0,0,0,.2)`.
- **Category axis:** hairline axis line, no ticks, ink2 11 px labels with `hideOverlap`, an optional rotation (and `interval: 0` when rotated).
- **Value axis:** split lines in hairline at 50% opacity, no axis line or ticks, labels via `tickFormatter(format)`.
- **Legend:** shown at the bottom only for multiple series; circle icons 9×9, 11 px ink2 text.
- **Toolbox:** `right:6, top:-3, itemSize:12, itemGap:7`, icon border `muted` (`ink` on hover). Features:
  - `saveAsImage` (title "Save PNG", name "chart", pixelRatio 2);
  - `dataView` (read-only, title "Data", lang `['Data','Close','']`);
  - optional `dataZoom` (titles Zoom / Reset zoom);
  - `restore` ("Reset").
- **Gauge:**
  - `max` = 100 for percent formats; otherwise `niceCeil(max(val, target, 1)×1.2)`, where `niceCeil` rounds up to 1/2/5/10 × 10^k;
  - the arc runs from 210° to −30°, radius 90%, center (50%, 60%);
  - axis line width 14 with color zones:
    - with a target: `frac = clamp(target/max, .02, .98)`; `above` → `[[frac,bad],[1,good]]`; `below` → `[[frac,good],[1,bad]]`;
    - without a target: a single blue, `#256abf` (dark `#3987e5`);
  - pointer width 5, length 60%, ink; anchor size 10; no ticks or split lines; labels 9 px at distance 14 (money is shown as a plain number);
  - the value detail sits at offset (0, 36%) in 26 px weight 800, colored green or red if there is a target, otherwise ink.
- **Funnel:**
  - item tooltip `marker name<br/><b>value</b> (pct%)`; scrolling bottom legend;
  - `top:12, bottom:34, left/right:'8%', minSize:'14%', gap:2, sort:'descending'`;
  - labels inside, white 11 px weight 600; surface border width 2; each item colored by its name.
- **Series (bar/hbar/line/area):**
  - **Radar** has its own builder: indicators = x values; one radar data entry per series, with line width 2 and area opacity 0.15 in the series color; legend if more than 1 series; hairline split lines at 0.6 opacity; no split area.
  - **Reference line**, on the primary series, on the value axis (`xAxis` for hbar):
    - `avg` → ECharts `{type:'average', name:'Avg'}`;
    - `median` → the median of the primary values, computed on the client;
    - a number → `{[axis]: n, name:'Target'}`;
    - styled dashed ink2 width 1.5, label `"<name>: <formatted value>"`, silent.
  - **Threshold**, applied only when there is **exactly one series**:
    - it **replaces** the reference line with a dashed `ink` line (width 1.6) labelled `Target ≥ X` / `Target ≤ X` in white on ink, padding [3,6], radius 4, 10 px, position `insideEndTop`;
    - each bar or point is colored green or red by `onTarget`;
    - for line/area, a `markArea` adds two bands: target→max and min→target, the good one tinted green and the other red (alpha 0.10 in light, 0.14/0.16 in dark).
  - **Bars:** `barMaxWidth: 38`; radius `[4,4,0,0]` (hbar `[0,4,4,0]`); stack `'st'` when `stacked`.
  - **Lines:** `smooth: 0.35`, line width 2.5; symbols shown (size 7) only when a threshold applies; area fill opacity 0.22; **stacking only for area**.
  - **X-axis label rotation** (non-hbar): 28° if there are more than 6 categories **or** X is not a date (so category X axes are always rotated); otherwise 0.
  - **hbar:** the value axis is horizontal and the category axis is vertical with `inverse:true`; grid left is 110; height is `max(260, n×26+60)`.
  - Grid bottom is 48 when there are multiple series.
  - Tooltip: axis trigger, pointer `shadow` for bars and `line` for lines, `valueFormatter` with the primary format.
  - Line/area also get the toolbox zoom feature and `dataZoom: [{type:'inside', filterMode:'none'}]`.
- **Pie/donut:**
  - radius `['48%','74%']` for donut, `['0%','74%']` for pie; center (50%, 44%);
  - no labels or label lines; surface border width 2; scrolling bottom legend; tooltip with value and percent.
- **Treemap:** no roam, no node click, no breadcrumb; width and height 100%, top/bottom 4; white 11 px weight-600 labels; border surface 2, gap 2.
- **Scatter/bubble:**
  - one series per group, named by group or by `yName`;
  - points are `[x,y]` or `[x,y,size]`; bubble size scales linearly from 8 to 38 px between the min and max size; otherwise a fixed 9;
  - opacity 0.7; `large:true, largeThreshold:2000, progressive:4000`;
  - value axes named by column (`nameLocation:'middle'`, gap 26 on x and 40 on y, muted 11 px) with `scale:true`;
  - tooltip lists X, Y and size; legend if more than 1 group.
- **Histogram:**
  - category axis of bin labels rotated 30°; a single bar series with `colorOf('count')`, radius [3,3,0,0], `barCategoryGap '6%'`;
  - tooltip `"<field>: <bin><br/><b>n</b> rows"`.
- **Box:**
  - one boxplot series; each box is filled with its color at 0.28 alpha, with a border in the same color at width 1.5;
  - x labels rotate 20° if there are more than 5 groups;
  - tooltip lists max, Q3, median, Q1 and min.
- **Heatmap:**
  - grid `left 90, bottom 60`; cell labels only when `xs × ys ≤ 120` (money formatted as a number); cell border surface 2, ink on hover;
  - x labels rotate 30° if there are more than 6 columns;
  - **without a threshold:** a continuous visualMap (min/max, calculable, horizontal, centered at the bottom, itemWidth 12, itemHeight 120) over a blue ramp:
    - light: `#eef5fd #cde2fb #86b6ef #3987e5 #256abf #0d366b`
    - dark: `#12233b #184f95 #256abf #3987e5 #6da7ec #b7d3f6`
  - **with a threshold:** a hidden piecewise visualMap on dimension 2, green or red at alpha (.32/.30 in light, .55/.60 in dark): `above` → `gte` good, `lt` bad; `below` → `lte` good, `gt` bad.

### 16.19 Palette and color allocation (`lib/palette.js`, `ColorSwatchPicker.jsx`)

**The 14 colors** (id, name, light, dark):

| id | name | light | dark |
|---|---|---|---|
| blue | Blue | #2a78d6 | #3987e5 |
| orange | Orange | #eb6834 | #d95926 |
| aqua | Aqua | #1baf7a | #199e70 |
| yellow | Yellow | #eda100 | #c98500 |
| magenta | Magenta | #e87ba4 | #d55181 |
| green | Green | #008300 | #17a217 |
| violet | Violet | #4a3aa7 | #9085e9 |
| red | Red | #e34948 | #e66767 |
| teal | Teal | #0d9488 | #14b8a6 |
| rose | Rose | #db2777 | #ec4899 |
| lime | Lime | #65a30d | #84cc16 |
| indigo | Indigo | #4f46e5 | #6366f1 |
| brown | Brown | #b45309 | #d97706 |
| slate | Slate | #64748b | #94a3b8 |

Slots 1–8 are the CVD-validated categorical order used for automatic assignment; 9–14 extend the set for hand-picking.

- **`pick(id, mode)`** returns the light or dark hex, falling back to blue.
- **`nextFreeColor(used)`** returns the first unused id, or null.
- **`assignDistinct(keys, existing)`:**
  1. keep every existing assignment that is valid and not yet taken (first come, first served);
  2. fill the rest with the next free color in palette order;
  3. with more than 14 keys, reuse colors deterministically as `PALETTE_IDS[index % 14]`.

  Both the editor and the renderer use this function, so they always agree.
- **`colorKeys`** come from the server payload:
  - series → series keys (measure ids, or group values);
  - parts / box / points → names or groups (or `['points']`);
  - histogram → `['count']`;
  - kpi, gauge, table, heatmap → none, so these types have no color controls.
- **`ColorSwatchPicker({value, used, onChange})`:** 14 circular 28 px swatches.
  - A swatch taken by another key is **disabled**: opacity 0.28, cursor not-allowed, title "<Name> — used by another point".
  - The selected swatch has a 2 px ink outline with 2 px offset, scale 1.08 and a white ✓ with a text shadow.
  - Others have a 1 px `--card-border` outline.

### 16.20 Formatting (`lib/format.js`)

- **`fmtNum`:**
  - |n| ≥ 1e9 → `x.xxB`; ≥ 1e6 → `M`; **≥ 10,000** → `K` (two decimals max);
  - otherwise an integer with en-US separators (**0 decimals**);
  - non-finite → `String(v ?? '—')`.
- **`fmtNumFull`** uses up to 2 decimals. **`fmtMoney`** is `$` + `fmtNum`. **`fmtMoneyFull`** is `$` + 2 decimals.
- **`fmtPct`** is up to 2 decimals + `%`. The value is assumed to be a percentage already; it is not multiplied by 100.
- **`fmtBy(format, v)`:** empty or null → `'—'`; money, moneyFull, percent and number use the functions above; date, text and default → `String(v)`.
- **`tickFormatter`:** money → `fmtMoney`; percent → `${v}%`; number → `fmtNum`; else `String`.
- **`parseHuman(input)`:** removes `$ , % whitespace`, then matches `^(-?\d*\.?\d+)\s*([kmbt])?$` with case-insensitive multipliers 1e3, 1e6, 1e9, 1e12; otherwise falls back to `Number`; returns null if it cannot parse.
- **`suggestThresholds(v)`:**
  - `mag = 10^floor(log10|v|)`; the multipliers `[0.5,0.75,1,1.25,1.5,2,2.5,3,4,5,7.5,10]` are applied and rounded to 4 significant digits;
  - take the 2 candidates below |v| and the 3 at or above it, de-duplicate, restore the sign, sort, and keep at most 5;
  - 0 or NaN → `[]`.

### 16.21 Pages

#### 16.21.1 Landing (`/`)
- **Hero:** an "aurora" background (four blurred radial gradients in blue, purple, orange and green at 60% opacity) and 96–128 px vertical padding.
  - eyebrow "No-code dashboard studio" (blue, uppercase, 14 px);
  - H1, 44 / 64 px: "Turn a spreadsheet into **a dashboard.**", with the last phrase in the gradient text (#2a78d6 → #9f4bd6 → #eb6834 → #eda100);
  - subtitle (18 px): "Upload your CSV or Excel file and build beautiful, fully-customizable charts — choose the axes, pick the colors, arrange the layout. All in the browser."
  - **CTAs:** a blue pill reading "Open the builder" if there is data or "Upload data & build" otherwise, going to `/builder`; a ghost pill "View dashboards ›" going to `/dashboards`.
- **Feature grid** (1/2/3 columns): each card has a 44 px gradient icon tile, a title and a description.
  - "Your data, only" — Drop in a CSV or Excel file. Nothing is charted until you do — no dummy data, ever.
  - "Every chart" — Bar, line, area, pie, donut, scatter, bubble, histogram, box, radar, treemap and heatmap.
  - "Any axis" — Pick what goes on X and Y. The builder only offers columns that relate to your choice.
  - "Your colors" — A curated 14-color palette. Every series a distinct color — no two points ever clash.
  - "No code" — Drag, resize and arrange widgets. Save as many dashboards as you like.
  - "Live" — Charts recompute the instant your data or mapping changes. Everything runs in your browser.

#### 16.21.2 `PageHeader`
Bottom hairline border, max-width 1180, padding 40–48 px: eyebrow (13 px uppercase blue), H1 (30 / 38 px) with an optional gradient `highlight`, subtitle (15 px ink2, max-w-2xl), and optional children.

#### 16.21.3 Builder (`/builder`)
**Header.** Eyebrow "Builder"; title "Build a dashboard —" with the highlight "no code required."; subtitle "Upload data, add charts, map them to your columns, pick colors, arrange and save."

**URL parameters, read once when the dashboards list is ready:**
- `?new=1` → a blank dashboard;
- otherwise `?dash=<id>` loads that dashboard, or the **first** dashboard, or a blank one;
- `?add=1` also opens the widget editor;
- `?join=1` also opens the join dialog.

**State.** `dashId`, `name`, `widgets`, `dirty`, `dragId`, editor open/editing, toast, join open, and **`boardDatasetId`** (the dashboard's single dataset).
- `loadDashboard(d)` fills any missing widget ids with `uid()` and sets the dataset from the first widget (or the active or first dataset).
- `startBlank()` uses the name `suggest('New dashboard')`.
- An effect keeps an **empty** board in sync with `activeId`, so after an upload the board switches to the new dataset.

**No data yet.** A card "Start by uploading your data" with the full drop zone.

**Dataset bar** (a card):
- the label "Dataset";
- a select (min width 230) showing `"<name> · N rows"`, placeholder "Select a dataset";
- the caption "Every widget on this dashboard uses this dataset.";
- a **Join** outlined pill with the merge icon, disabled when there are fewer than 2 datasets (tooltip "Upload 2+ datasets to join" or "Join two datasets on a shared key");
- a delete-dataset icon (tooltip "Delete this dataset");
- the compact upload button **"Add dataset"**.
- **Switching dataset** when widgets exist asks `window.confirm`: "This dashboard is built on “A”. Switch to “B”? Its N widget(s) will be removed — a dashboard can only visualize one dataset." On confirm it clears the widgets, marks the board dirty, and sets both the board dataset and the active dataset.
- **Deleting a dataset** asks "Delete dataset “X”?", adding "This dashboard is built on it — its widgets will be cleared." when relevant. It clears the widgets if they used it, deletes the dataset, re-points the board to another dataset, and toasts "Dataset deleted."

**Toolbar** (a card):
- a **Dashboard** select (min width 200, notched label, placeholder "New (unsaved)", option "No dashboards yet" when the list is empty). Choosing one loads it;
- a **Name** text field. A duplicate name (case-insensitive against other dashboards) turns it red and shows "A dashboard named “X” already exists — pick another name." in `#c0392b`;
- buttons:
  - **Add widget** (contained, opens the editor);
  - **Add card** (outlined; opens the editor pre-filled with `{type:'kpi', w:1, pinned:true}`);
  - **Save** (contained). It reads "Save" when new, "Save changes" when dirty, and "Saved" otherwise. It is disabled when the name is a duplicate, the name is empty, or the board is saved and clean. It calls `update` or `create` and toasts "Dashboard saved." or the error;
  - **New** (outlined; blank board, toast "Blank dashboard — add widgets and save.");
  - **View** (text; goes to `/dashboards`);
  - a **delete-dashboard** icon, disabled for unsaved boards. It deletes **without confirmation**, loads the next dashboard or a blank one, and toasts "Dashboard deleted."

**Canvas.**
- **Empty:** a large dashed button: gradient + icon, "Your canvas is empty", "Add a chart, or a card for the top row."
- **Otherwise:**
  - a **Pinned** section with a header "★ Pinned" (star #eda100, 12 px uppercase muted) and its own grid;
  - then the main grid. Both grids are `grid-cols-1 lg:grid-cols-3 gap-6 items-start`, so each card keeps its own height.
- **Widget card:**
  - a `.card` with 20 px padding, `lg:col-span-{w}`, HTML5-draggable; opacity 40% while dragging, 55% when disabled;
  - a header with:
    - the drag handle icon;
    - the title ("Untitled" if empty) plus a **"Hidden"** chip when disabled;
    - a subtitle "<chart type name> · <dataset name or 'dataset removed'>";
  - controls on the right:
    - Edit and Delete icons, visible on hover;
    - the **eye toggle**: hover-only when visible; always shown in orange `#c77700` when hidden; tooltips "Hide from viewers" / "Show when viewing";
    - the **star pin**: hover-only unless pinned; filled `#eda100` when pinned; tooltips "Pin to top" / "Unpin from top";
    - an **always-visible S/M/L segmented control** (24 px buttons on a chip background; active is blue/white; titles "Width: Small (1/3)", "Medium (2/3)", "Large (full)");
  - the body is a `QueryChart` with a 140 px minimum height.
- **Drag to reorder:** dropping card A on card B moves A to B's index in the single widgets array. It does not change `pinned`. Any reorder marks the board dirty.

**Toasts.**
- Saving from the editor: "Widget updated.", "Card added to the top." (pinned) or "Widget added to the bottom."
- Quick-adding suggestions: each gets a `uid()`, the board's dataset, `w` = given value or (1 for KPI, otherwise 2), `pinned:false`, `disabled:false`. Toast "Added <Chart name>." or "Added N widgets." The editor stays open.
- After a join: refresh datasets, clear the widgets, point the board at the new dataset, mark it dirty, toast `Joined dataset “X” created — N rows[ (capped at 300k)].`

#### 16.21.4 Viewer (`/dashboards`)
- **Header:** eyebrow "Dashboards"; title "Your dashboards," with the highlight "at a glance."; subtitle "View any saved dashboard. Build and edit them in the Builder."
- **Top bar:**
  - one pill button per dashboard (the active one blue, the others ghost), or the text "No dashboards yet." if none;
  - on the right, **Edit** (outlined, goes to `/builder?dash=<id>`) and **New dashboard** (contained, goes to `/builder?new=1`).
- **Initial selection:** the `?dash=` parameter, otherwise the first dashboard.
- **No dashboards:** a large dashed button "No dashboards yet / Create your first one in the Builder." that goes to `/builder?new=1`.
- **Rendering:** widgets with `disabled` set are filtered out. Pinned widgets get their own row with the "★ Pinned" header, then the main grid, using the same spans as the Builder.
  - Read-only cards show the title and the "type · dataset" subtitle.
  - The two grids sit inside `captureRef`, the area captured for snapshots.
- **All widgets hidden:** a card reading "All widgets on this dashboard are hidden."
- **A dashboard with no widgets:** a card "This dashboard has no widgets yet." with an **Add widgets** button.
- **`<SnapshotPanel dashboardId targetRef={captureRef}/>`** renders below the grids.

### 16.22 Widget editor in detail (`WidgetEditor.jsx`)

**Props:** `open, initial, datasets, defaultDatasetId, lockedDatasetId, existingWidgets, onAdd, onSave, onClose`. The Builder always passes the board's dataset as both `defaultDatasetId` and `lockedDatasetId`.

**Draft defaults (`blank`):**
```js
{ id:null, datasetId, type:'bar', title:'', w:2, agg:'sum', bins:12, stacked:false, pinned:false,
  disabled:false, sort:'desc', limit:'', refLine:'none', threshold:null, colors:{} }
```

**Opening the editor:** `draft = initial ? {...blank(ds), ...initial, datasetId: ds, colors: copy} : blank(ds)`. "Advanced" starts collapsed.
- `aggTouched` starts true **only for saved widgets** (those with `initial.id`); new widgets get the smart default.
- `threshText` = `fmtNum(threshold.value)` if a threshold exists.

**Layout.** An MUI Dialog, `maxWidth md`, fullWidth, title "Edit widget" or "Add a widget". The content is two columns on md and up.

#### 16.22.1 Left column: configuration
1. **"1 · Dataset".**
   - Locked: a read-only box with the dataset name and "N rows · M columns" (or "Dataset unavailable").
   - Unlocked: a select (not used in practice).
   - Caption: "All widgets on this dashboard use the same dataset."
2. **"Suggested charts" panel.** Shown when there is a dataset and at least one visible suggestion. It is fetched from `GET /suggest` whenever the editor opens.
   - Styling: purple border `rgba(159,75,214,.35)`, background `.06`, sparkle icon `#9f4bd6`, header "Suggested charts · add as many as you like".
   - **"Add all (N)"** appears when there is more than one suggestion.
   - Each row shows `"<label> · <title>"` (12.5 px, semibold, truncated), the reason (11 px muted), and two buttons:
     - **Edit** loads the suggestion into the draft: type, x, y, group, field, agg, bins and title; clears size and cols; resets sort, limit, refLine, stacked and colors; sets `aggTouched`.
     - **Add** (contained pill) calls `onAdd([{...widget, title}])`.
   - **De-duplication:** suggestions whose signature `type|x|JSON(y)|field|group` matches an existing widget are hidden.
3. **"2 · Choose a chart".**
   - For each group, a small uppercase heading and a 3-column grid of tiles. Each tile has a `ChartThumb` SVG (26 px tall) and the type name; its tooltip is the type's description.
   - The selected tile has a `--blue` border and `rgba(0,113,227,.1)` background.
   - **Changing type resets the mapping:** it keeps id, datasetId, title, w, bins, pinned and disabled; sets `agg` to 'sum', `sort` to 'desc', `limit` to '', `refLine` to 'none', `threshold` to null, `colors` to {}, `stacked` to false; clears the threshold text; and resets `aggTouched`.
4. **"3 · Map your data".**
   - Without a dataset: a warning "Pick a dataset first."
   - One **`SlotControl`** per slot:
     - the slot label (12.5 px);
     - **multi slots** use toggle chips, filled primary when selected and outlined otherwise, or "No compatible columns." if there are none;
     - **single slots** use a select with `displayEmpty`: placeholder "— select —" (required) or "— none —" (optional), a "none" item for optional slots, items "<name> · <type>", and a disabled "No compatible columns" item when empty;
     - the hint below (11.5 px muted).
   - Options come from `compatibleColumns`, excluding the other raw-measure slots for scatter/bubble.
   - **Any slot change clears `colors`.**
   - **Automatic aggregation:** while `!aggTouched` and the type aggregates, `agg` = `smartDefaultAgg(first measure: field or y[0] or y)`.
5. **Advanced disclosure** (only for types that aggregate): "▾ Advanced · summarized by **<agg name lowercased>**". The expanded section holds:
   - **Summarize by** select (7 options; changing it sets `aggTouched`), with help text "How rows are combined for each category. Auto-chosen from the column (rates average, amounts sum); change only if needed."
   - **Sort** (High → Low `desc` / Low → High `asc` / As-is `none`) plus a **Top N** number field (width 96, placeholder "All"; empty → `''`, otherwise at least 1). Only for sortable types.
   - **Reference line** (None / Average / Median / Custom value…). "Custom value…" stores `refLine = 0`. A **Target value** number field (width 160) appears for numeric values. Only for types that take reference lines.
6. **Bins** (histogram): "Bins: **N**" with a slider from 3 to 40.
7. **Target threshold** (thresholdable types):
   - Checkbox text: "Target threshold — highlight when the value is / values are on / off target".
   - **Turning it on:**
     - `threshRef` is the KPI/gauge value, the median of the first series, or the median of heatmap cells, all from the live preview;
     - `threshOptions = suggestThresholds(threshRef)`;
     - the initial value is `options[min(2, len−1)]`, otherwise `threshRef`, otherwise 0; the direction is `above`.
   - **Panel contents:**
     - "Good when the value is" with a toggle **≥ target** / **≤ target**;
     - a **Target** text field (width 150, placeholder "e.g. 150B"), parsed live with `parseHuman` (an unparseable value is ignored), followed by "= **<fmtBy(format, value)>**";
     - **Suggestions:** chips, with the selected one blue/white;
     - help text "Type a number with K/M/B (e.g. 200M, 1.5B) or pick a suggestion. The <tile|dial|bars|points & band|cells> turn(s) green when on target, red when off."
   - The format comes from `computed.format`, else the first series format, else 'number'.
8. **Stack checkbox.** "Stack the series", shown for stackable types when `y` is an array and there is more than one measure or a group.
9. **"4 · Colors · each unique"**, shown only when the preview returns `colorKeys`.
   - An **Auto** button (restart icon) resets `colors` to `{}`.
   - Each key gets a label and a `ColorSwatchPicker`, where `used` holds the colors assigned to the other keys.
10. **"5 · Title & size".** A "Widget title" field and S/M/L buttons (36×40, blue/white when active).

#### 16.22.2 Right column: live preview
- "Live preview": a `.card` with a 300 px minimum height and the title (`draft.title` or the chart type name).
- If the draft is not valid: a centered message ("Pick a dataset to begin.", or `validation.reason`, or "Configure the widget to preview it.").
- While loading with no cached data: the skeleton "Aggregating…".
- Otherwise: `ChartRenderer` with `{...draft, id: draft.id || 'preview'}`.
- The preview query uses `useWidgetQuery(validDraft ? datasetId : null, draft, {debounce: 300})`. **That same response supplies the `colorKeys` for the color pickers.**

#### 16.22.3 Actions
**Cancel**, and the primary button "Update widget" or "Add to dashboard", which is disabled unless the draft validates. It calls `onSave(draft)`.

### 16.23 Dataset upload and loader

- **`DatasetUpload({compact, onQueued})`** accepts `.csv,.tsv,.txt,.xlsx,.xls,.ods` and multiple files.
  - **Full mode:** a 22 px-radius dashed drop zone (64 px vertical padding).
    - It is focusable (`role=button`) and opens the file picker on click, Enter or Space.
    - While a file is dragged over it, the border turns `--blue` with a `rgba(0,113,227,.06)` background.
    - Contents: a 56 px gradient upload icon, "Drop CSV or Excel files here", and "or click to browse · multiple files supported · .csv, .xlsx, .xls, .ods".
    - Active uploads show as a 2-column grid of loaders below.
  - **Compact mode:** an outlined pill "Add dataset" with an upload icon (a hidden file input inside a label), with loaders stacked below.
  - Every upload that is not `ready` stays listed, so error tiles persist.
- **`ProcessingLoader({fileName, phase, progress, error})`**, a card with 32 px padding:
  - **7 bars** 10 px wide and 64 px tall with a vertical gradient (#2a78d6 → #9f4bd6). Each scales on Y from 0.3 to 1, in a 1.1 s cycle staggered by 0.11 s per bar. Reduced motion freezes them at 0.7.
  - **Normal state:** "Processing <file>", "<phase or 'Working…'> · NN%", an MUI LinearProgress (determinate if progress > 0, otherwise indeterminate; height 6, pill shape), and "Large files are parsed on the server — this can take a moment."
  - **Error state:** "Couldn’t process <file>" in red `#d03b3b`, plus the error message.

### 16.24 Join dialog (`JoinDialog.jsx`)

- An MUI Dialog (`maxWidth sm`) titled "Join datasets", with the intro: "Combine two datasets on a shared key column (e.g. **category_id**) into one dataset you can chart."
- **On open:** left = `datasets[0]`, right = `datasets[1]` (the two newest); keys, type and name are reset; `type` = 'inner'.
- **Two columns:** "Left dataset" and "Right dataset". Each has a dataset select (excluding whichever dataset the other side has chosen; items show "name · N rows") and, once chosen, a key select (items "name · type").
- **Key auto-match** whenever either dataset changes: the first left column whose **id** also exists on the right, otherwise the first one whose **name** matches ignoring case. If nothing matches, both keys are cleared. The name defaults to `"<L> + <R>"`, cut to 60 chars.
- Once both keys are set, a chip shows "⇄ Matching **L** = **R**".
- **"Join type"** toggle: **Inner (matches only)** / **Left (keep all left)**. Next to it, a "New dataset name" field.
- An error Alert shows server errors. An info Alert "Upload at least two datasets to join." appears when there are fewer than 2.
- **Actions:** Cancel, and **"Join & use"** ("Joining…" while busy). It is enabled only when both datasets are chosen and different and both keys are set.
- On success it calls `onJoined(meta)` (see Builder).

### 16.25 Snapshot panel and export

- **`SnapshotPanel({dashboardId, targetRef})`** is a `.card` with 32 px top margin, marked `data-no-snapshot`.
  - **Header:** a camera icon, "Snapshots · N", and a **Take snapshot** contained pill showing a spinner and "Capturing…" while busy. It is disabled without a dashboard id; with no id it would toast "Save the dashboard first."
  - **Capture:** `captureNode(targetRef.current, mode)` → `api.saveSnapshot(id, dataUrl, 'Snapshot · <toLocaleString()>')`, then refresh, then toast "Snapshot saved." On failure: "Snapshot failed: …".
  - **Gallery:**
    - empty: "No snapshots yet. Capture the current dashboard to save a PNG you can revisit.";
    - otherwise a wrapping row of 200 px tiles: the image at 16:10 aspect, cover-fit and top-aligned, lazy-loaded, on a gray background; the timestamp; and a delete icon.
  - Clicking a tile opens a large Dialog (`maxWidth lg`) with the label, an **Open in new tab** icon (`window.open(url)`), a close button, and the image (max height 78vh, contain).
  - Deleting a snapshot closes the viewer if it was showing that one.
- **`lib/snapshot.js captureNode(node, mode)`** uses html-to-image `toPng` with:
  - `pixelRatio: 2`;
  - `backgroundColor` `#0b0b0d` (dark) or `#ffffff` (light);
  - `cacheBust: true`;
  - a `filter` that skips any element carrying a `data-no-snapshot` attribute.

  ECharts canvases are captured as they are.
- **Per-chart export** comes from the ECharts toolbox: Save PNG, Data view and Reset, plus zoom on line/area. **This currently does not render; see §16.29.**

### 16.26 Assistant UI (`AiAssistant.jsx`)

- **Launcher:** a fixed button at `bottom-6 right-6`, z-60, 56 px circle with a gradient (#2a78d6 → #9f4bd6) and a blue glow shadow; hovering scales it to 105%. It shows the `ChatMark` speech-bubble icon (with three dots in the accent color) or a close icon when open. Tooltip "Dashboard assistant", `aria-label` "Open dashboard assistant".
- **Panel:**
  - fixed at `bottom-24 right-6`, width `min(92vw, 392px)`, height `min(72vh, 620px)`, `rounded-3xl`; background `#1c1c1e` (dark) or `#fff`; shadow `0 30px 80px -20px rgba(0,0,0,.55)`; entry animation `ds-chat-in` (0.32 s).
  - **Header:** gradient background, a 36 px white/15 circle with the ChatMark, "Dashboard assistant", and the subtitle "Powered by <provider>" when AI is on (from `/assistant/status`) or "Built-in · guided builder" otherwise. A close button.
  - **Tabs:** Chat / Prompts. The active tab is `#0071e3` (dark `#59a8ff`) with a 2 px underline.
- **Prompts tab:**
  - Intro: "Tap a template to drop it into the chat — filled in with columns from **<active dataset>**." (or "upload data to auto-fill the columns.") followed by "Edit the columns, then send."
  - Templates are filled from the active dataset's columns: N/N2 = the first and second numbers, C/C2 = the first and second categories, D = the first date. Missing columns become placeholders `[measure]`, `[measure #2]`, `[category]`, `[category #2]`, `[date]`.
    - **Build a whole dashboard:** "Build a dashboard from my data" → "Build me a dashboard".
    - **Add a single chart (during a build):**
      - Bar → `Add a bar chart of N by C`
      - KPI → `Add a KPI of total N`
      - Line → `Show N over D as a line chart`
      - Pie → `Add a pie chart of N by C`
      - Scatter → `Add a scatter plot of N vs N2`
      - Histogram → `Add a histogram of N`
      - Stacked bar → `Add a bar chart of N by C split by C2`
      - Heatmap → `Add a heatmap of N across C and C2`
      - Table → `Add a table of C, N, D`
    - **Curate:** `Suggest charts`, `Remove 2`, `Clear all`, `Build it`.
  - Tapping a template puts its text in the input, switches to Chat, and focuses the input after 60 ms.
- **Chat tab:**
  - **Welcome message (fixed):** "Hi! I can **build a dashboard exactly the way you want**. Say **"build a dashboard"** — then take my AI-suggested charts, or just describe the ones you want (e.g. **"add a bar chart of revenue by region"**). I also **define columns**, **compute metrics** ("average amount"), and **explain joins**. Peek at the **Prompts** tab for ready-to-use templates."
  - **Starters**, shown while there is at most one message: "Build me a dashboard", "What dashboards do I have?", "Which columns are numeric?".
  - **Bubbles:**
    - user: right-aligned, `#0071e3` (dark `#2a91ff`), white text, bottom-right radius 6;
    - assistant: left-aligned, white (dark `#2c2c2e`) with a hairline border, bottom-left radius 6;
    - max width 85%, 14 px text;
    - rendered with a minimal markdown pass: `**bold**` becomes `<strong>` and newlines become `<br>`.
  - **Link button** (when the reply has `link`): a tinted blue pill "<label> →". It navigates to the route and closes the panel.
  - **Option chips:** only on the **last** assistant message and only when not loading. Cancel is gray; the others are tinted blue. **A tap shows `label` as the user bubble but sends `value` as the message.**
  - While loading: a bubble with a spinner.
  - **Input:** a rounded TextField with placeholder "Ask about your data, or say “build a dashboard”…". Enter (without Shift) sends. The blue round send button is disabled while loading or when the input is empty.
- **`send(text, value)`:**
  - `history` = every message except the welcome one, as `{role, content}`. The server keeps only the last 6.
  - Calls `POST /assistant {message: value ?? text, history, flow, datasetId: activeId}`.
  - Stores `flow` if it is active, otherwise null.
  - If `res.created`, it calls the datasets context `refresh()`.
  - Appends `{role:'assistant', content: reply, link, options}`.
  - On an error it clears the flow and shows "Sorry — <message>".
  - **Chat history is not persisted** across reloads.

### 16.27 Theming

- **`ThemeModeContext`:**
  - `mode` starts from `localStorage['ds.mode']` (`'light'|'dark'`), otherwise from `prefers-color-scheme`.
  - An effect toggles the `dark` class on `<html>`, saves `ds.mode`, and sets `<meta name="theme-color">` to `#0b0b0d` or `#ffffff`.
  - Provides MUI `ThemeProvider(makeTheme(mode))` with `CssBaseline`, and `toggle()`.
- **MUI theme (`theme.js`):**
  - border radius 10; the system font stack;
  - weights and letter-spacing: h1 700 / −0.025em, h2 700 / −0.022em, h5 600 / −0.015em, h6 600 / −0.01em; buttons not uppercased, weight 500;
  - `MuiPaper` has no background image; `MuiTooltip` is 12 px with radius 8 and background `rgba(29,29,31,.92)` (dark `rgba(60,60,63,.96)`);
  - **light palette:** background `#fff`, primary `#0071e3`, secondary `#9f4bd6`, success `#0ca30c`, warning `#fab219`, error `#d03b3b`, text `#1d1d1f` / `#6e6e73`, divider `rgba(0,0,0,.08)`;
  - **dark palette:** background `#0b0b0d`, paper `#1c1c1e`, primary `#2a91ff`, secondary `#c07bef`, success `#30d158`, warning `#ffd426`, error `#ff5a5a`, text `#f5f5f7` / `#a1a1a6`, divider `rgba(255,255,255,.12)`.
- **Chart tokens (`tokensFor`).** ECharts draws on a canvas and cannot read CSS variables, so these values are fixed in code:
  - light: `ink #1d1d1f, ink2 #6e6e73, muted #86868b, hairline #d2d2d7, surface #ffffff, cardBorder rgba(0,0,0,.08)`;
  - dark: `ink #f5f5f7, ink2 #a1a1a6, muted #8a8a8e, hairline #38383a, surface #1c1c1e, cardBorder rgba(255,255,255,.12)`.
- **CSS variables (`index.css`).** In `:root`, with the `html.dark` value after the slash:
  - `--white` #fff / #0b0b0d (the body background)
  - `--surface` #fff / #1c1c1e
  - `--gray-bg` #f5f5f7 / #161618
  - `--ink` #1d1d1f / #f5f5f7
  - `--ink-2` #6e6e73 / #a1a1a6
  - `--muted` #86868b / #8a8a8e
  - `--hairline` #d2d2d7 / #38383a
  - `--blue` #0071e3 / #2a91ff
  - `--blue-link` #0066cc / #59a8ff
  - `--card-border` rgba(0,0,0,.08) / rgba(255,255,255,.1)
  - `--card-shadow`
  - `--frost-bg` (and `--frost-bg-solid`)
  - `--chip` rgba(0,0,0,.05) / rgba(255,255,255,.09)
  - scrollbar colors
  - `color-scheme`
- **Utility classes:**
  - `.display`, `.headline`, `.eyebrow` (type styles);
  - `.grad-text` (four-stop gradient text);
  - `.card` (radius 14, 1 px card-border, shadow) and `.card-hover`;
  - `.frost` (saturate 180% + blur 20 px, with a solid fallback);
  - `.btn-pill` (radius 999, padding 8×18, 13.5 px, presses to 0.97) with `.btn-blue` and `.btn-ghost`;
  - custom WebKit scrollbars;
  - `.aurora`;
  - a focus-visible outline of 2 px `--blue` with 2 px offset;
  - `ds-page-enter` (6 px fade-up, 0.28 s);
  - reduced-motion overrides.
- **Body:** letter-spacing −0.01em, antialiased, smooth scrolling, and a 0.3 s background/color transition.

### 16.28 Persistence summary

| where | key / file | contents |
|---|---|---|
| Browser `localStorage` | **`ds.mode`** | `'light'` or `'dark'`. This is the **only** client-side storage. The chat, uploads and editor state live in memory. |
| Server memory | job registry | upload jobs, reaped 10 min after they finish |
| Server memory | rows cache | dataset rows by id |
| Disk | `server/data/datasets/index.json` | dataset metadata array, with columns |
| Disk | `server/data/datasets/<uuid>.rows.json` | rows array |
| Disk | `server/data/dashboards.json` | dashboards array, with widgets and `filters: []` |
| Disk | `server/data/snapshots/index.json` and `<uuid>.png` | snapshot records and images |
| Temp | `os.tmpdir()` | multer upload temp file, unlinked after parsing |

### 16.29 Known gaps, bugs and doc inconsistencies (decide before rebuilding)

1. **Missing ECharts registrations.** `src/lib/echartsCore.js` does not register `GaugeChart`, `FunnelChart`, `ToolboxComponent`, `MarkLineComponent` or `MarkAreaComponent`. In ECharts 6's modular build that means:
   - **gauge and funnel widgets draw nothing**, only a dev-console error;
   - the **toolbox** (PNG, data view, zoom, reset) is not shown;
   - **reference lines, threshold target lines and line/area threshold bands** are not drawn.

   Per-bar and per-point threshold colors, the heatmap's piecewise visualMap and inside zoom do work. A rebuild should register all five.
2. **Progress is effectively invisible.** Parsing runs synchronously (`XLSX.readFile`, the loops, and the synchronous JSON write) inside one `setImmediate`. That blocks the event loop, so `GET /jobs/:id` polls cannot be answered mid-parse, and the client sees roughly "Queued 0%" then "Ready". Real progress needs a worker thread or streaming parse.
3. **Heatmap cell keys break on spaces.** Cells are keyed by `` `${xv} ${yv}` `` and later `split(' ')`. Any category value containing a space lands in the wrong cell. Use a separator that cannot appear in values, or nested maps.
4. **`distinct` only counts the 2000-row sample.** So the "near-unique" rule in `isIdentifier` (`distinct ≥ 0.95×rows`) cannot fire for datasets above about 2,100 rows; only the name rule works there. The `≤ 30 distinct` rule for categorical numbers is also based on the sample.
5. **Pie/donut/treemap/funnel ignore `sort:'none'`.** It behaves like `desc`. The "Other" bucket appears for any direction except `asc`.
6. **Split-mode series sort by row count, not by the measure.** With a `group`, X keys are ranked by how many rows each has. With more than one measure, `group` is silently ignored.
7. **Count is inconsistent.** In the engine, `count` counts every row including nulls. The assistant's metric count excludes nulls.
8. **Table payloads are larger than needed.** They return all fields of the first 200 rows, not just the selected columns.
9. **Box groups are not ranked.** They are capped to the first 12 in first-seen order. Scatter keeps points from groups beyond 12 in `data`, but they are not drawn.
10. **A custom reference line of `0` draws nothing** (`!refLine` is true for 0).
11. **Joins treat null keys as matching.** Null or empty keys are compared as `''`, so null-keyed rows on both sides join to each other.
12. **No cascade on dataset delete.** Dashboards keep their widgets. Those widgets then show "Could not load: Dataset not found.", and the card subtitle shows "dataset removed".
13. **Deleting a dashboard in the Builder has no confirmation.** Deleting a dataset or switching datasets does ask.
14. **Upload error tiles never go away.** `dismissUpload` exists, but no UI calls it.
15. **The Builder reads URL parameters only on first mount** (its effect depends only on `ready`). The assistant's "Open my dashboard" link (`/builder?dash=…`) clicked while already on `/builder` does not load the new dashboard, because the `main` key changes only with the pathname. `useDashboards` is per-component state, and after an assistant build only the datasets context is refreshed.
16. **"What can you chart from my data?" starts the build wizard.** It is one of the server's suggestions, and `isBuildIntent` sees both "chart" and "can you".
17. **Dead code paths.**
    - The client never sends `?col=` to `/suggest`, never calls `/column`, never sends filters, and never calls `invalidateDataset`.
    - `EChart`'s `onClick` prop is unused.
    - `dashboard.filters` is always `[]`.
18. **Production hosting.** The server does not serve `dist/`, so a separate static host plus an `/api` proxy is needed.
19. **Documentation drift.**
    - The README describes a filter bar and cross-filtering that were removed. It lists 14 chart types without gauge and funnel, and its source map puts `WidgetEditor.jsx` and `ColorSwatchPicker.jsx` under `lib/` (they are in `components/`).
    - `PROJECT_CONTEXT.md` says "16 chart types" but lists 17; the registry has **17**. Its item 12 still refers to `ANTHROPIC_API_KEY`; the code uses OpenAI.
    - The ER diagram in `ARCHITECTURE.md` lists 15 types.
    - The Landing "Live" card says "Everything runs in your browser", but parsing and aggregation run on the server.
    - The Landing "Every chart" card omits kpi, hbar, table, gauge and funnel.

## 17. Rebuild plan and acceptance checklist

### 17.1 Suggested build order

Each phase leaves a working, demonstrable product. Later phases depend on the data the earlier ones create.

| Phase | Build | Depends on | Done when |
|---|---|---|---|
| 1. Foundations | Persistence layer (or database), ids, API envelope, error model, auth (sign-up, login, sessions), workspace roles, global auth wall, role gates, app shell (routes, sidebar, theme, currency control) | — | A user can sign up, log in, see the empty shell in light and dark mode, and a Viewer is blocked from writes with the exact 403 message. |
| 2. Organisation and collaboration | Org roles, teams, people search and picker, demo personas, notifications (derived + pushed), audit log with role scoping, recycle bin framework, Admin page, profile | 1 | An admin can place people in the org chart, assignment notifications arrive, and every action shows in the audit log for the right viewers. |
| 3. Planning core | Five-level hierarchy with every validation rule, models, tactics, line items with C/F/L/E spread, approvals lifecycle, roll-ups and auto dates, Grid, Inspection Window, tactic drawer, Budget & Cost view | 1–2 | The scenarios in chapter 7 pass: duplicate names rejected, program budget ≤ campaign, evergreen one-way, Sys_Gen_Balance correct, cascade delete and restore as Created. |
| 4. Planning views and imports | Board (columns, WIP, custom fields, templates, swimlanes, saved views, bulk), calendar (drag, `.ics`, print), CSV/Excel import with undo, Jira/Trello import and disconnect, Google Calendar push | 3 | Drag-and-drop persists, imports create tactics under programs and can be undone, and the feed subscribes in a calendar client. |
| 5. Dual campaign plans | SP&A and BCPIR derivation, the register, plan detail, plan approval workflow, the "Raise PR" hand-off | 3 | Reference figures match chapter 7's formulas for the seeded tactics. |
| 6. Finance | FX service, budgets (pools, allocations, committed roll-up, scope picker, snapshots), procurement (PR lines, threshold approval chain, approver lock, PO → invoice → receipt) | 3, 5 | The seeded Budget page totals and procurement routing ($78K vs $6K examples) match chapter 9. |
| 7. Performance and intelligence | Deterministic demo spine, Home/Overview, Campaigns, Goals, insights, AI analysis with provider cascade and fallback, optimizer, attribution (9 models, funnel, forecast), GA4 with demo fallback and realtime, Integrations catalog | 1 | Chapter 10's reference outputs reproduce exactly from the demo data. |
| 8. Builder and assistant | Widget catalog, data bundle, Builder editor (bind, thresholds, custom axes, resize, reorder), dashboards viewer, snapshots, uploaded dataset sources, "Ask ZenReach" chat, guided dashboard wizard, natural-language widgets | 7 | The assistant builds a valid dashboard from conversation, and every widget type renders with thresholds. |
| 9. Companion app | Dashboard Studio (chapter 16) | none (separate product) | Upload → infer → build → view → snapshot works on a 100k-row file. |
| 10. Hardening | Chapter 13 non-functional requirements, decision-log fixes below, automated tests | all | Security and resilience checks pass. |

### 17.2 Acceptance checklist (cross-module)

- [ ] Every error message listed in chapters 5–11 is reproduced verbatim at the right status code.
- [ ] Every role gate in the section 4.5 permission matrix holds on the server, not only in the UI.
- [ ] Money stored in USD and displayed in the active currency everywhere, including charts, exports and tooltips.
- [ ] Every mutation writes exactly one audit event with the catalogued action id, and the audit view respects org-role scoping.
- [ ] Every delete of a covered entity goes to the recycle bin and restores with original ids; parent-before-child restore enforced.
- [ ] Planning roll-ups (planned cost, line items, Sys_Gen_Balance, actuals, unallocated budget, auto dates) agree in the Grid, Budget & Cost, tactic drawer and Inspection Window.
- [ ] Budget "committed" and SP&A plan figures reconcile with tactics and line items; procurement committed and invoiced figures reconcile with PR statuses.
- [ ] Deterministic demo data reproduces chapter 10's reference outputs and the chapter 14 demo story numbers.
- [ ] Each external integration degrades to its documented fallback when unconfigured or failing, and labels the data as demo or sample.
- [ ] Light and dark themes, reduced motion, keyboard focus rings, and responsive layouts at 390 px width all work.

### 17.3 Consolidated decision log

The chapters below each end with a list of current behaviours the rebuild should keep or fix *deliberately*. Review them together before implementation starts.

| Area | Where | Highlights |
|---|---|---|
| API surface | §5.8 | Default HTML error pages; dead anonymous-demo branches; unused imports; case-sensitive public allow-list. |
| Identity and org | §6.10 | Non-expiring, non-cryptographic session tokens; no signup becomes Owner after seeding; Viewers can mutate dashboards; no live user refresh. |
| Planning backend | §7.12 | Quirks in validation scope, approval resets, migration rules, import edge cases and derived-finance heuristics. |
| Planning UI | §8.18 | Orphaned components (Plan Copilot, legacy grid), prompt-based saved views, permission display mismatches. |
| Finance | §9.10 | Approval-chain edge cases, FX caching, scope behaviours, snapshot storage limits. |
| Performance and AI | §10.13 | Provider-order inconsistency between chat and analysis, keyword-matching quirks, demo-data constants. |
| Builder | §11.14 | No role gate on dashboard writes, client-only snapshots, data bundle size. |
| App shell | §12.11 | No catch-all route, the old Overview page still in code but not routed. |
| Documentation | §15 (docs conflicts) | Planning view lists, hierarchy depth, AI provider and connector lists disagree across documents. |
| Dashboard Studio | §16.29 | Dormant filters, URL-parameter reload bug, assistant intent overlaps. |

**Recommended baseline decisions for a production rebuild:**
1. Use a relational database with transactions (chapter 15 schema) in place of whole-file JSON writes.
2. Use cryptographically random session tokens with expiry and revocation on password change.
3. Return JSON error envelopes for all failures, including malformed bodies, and add rate limiting and security headers.
4. Role-gate every write on the server, including dashboards.
5. Make tenant boundaries explicit if more than one workspace is planned.
6. Keep every deterministic algorithm and constant as specified, so demo data and tests stay reproducible.

## Appendix A. Source file inventory

Every source file reviewed for this specification, with its line count and responsibility. Documentation, sample data and generated files are covered in chapter 15.

### A.1 ZenReach (`server/`, `client/`)

| File | Lines | Responsibility |
|---|---|---|
| `client/index.html` | 21 | HTML shell; mounts the React app (`#root`), sets title/meta/favicon. |
| `client/postcss.config.js` | 7 | PostCSS pipeline: Tailwind + Autoprefixer. |
| `client/public/tracking-test.html` | 215 | Standalone GA4 traffic generator used to demonstrate the live Analytics connection. |
| `client/src/App.jsx` | 98 | Route table, auth-aware layout switch, currency re-render subscription, cursor-spotlight effect. |
| `client/src/api.js` | 221 | Tiny fetch wrapper. All calls go through Vite's /api proxy to the Express server. |
| `client/src/components/AiAssistant.jsx` | 341 | Ready-to-run prompt templates surfaced in the "Prompts" tab. Tapping one drops it into the chat input to tweak and send. Tuned to ZenReach's real marketing data catalog. |
| `client/src/components/AnimatedNumber.jsx` | 32 | Counts up from 0 to `value` using GSAP, formatting each frame. Reads the active currency so that when it changes the number is re-formatted in place (the app no… |
| `client/src/components/BrandLogo.jsx` | 34 | ZenReach brand mark: a rising trend line that lands on a "reach" node with a soft pulse ring — growth that reaches its audience. Rendered white on the brand gradient… |
| `client/src/components/ChartThumb.jsx` | 71 | Tiny inline SVG preview for each builder widget type, so the picker shows WHAT you'll get before you pick it. Pure SVG (no assets) using an accent color, so it tracks… |
| `client/src/components/ChatMark.jsx` | 24 | "Ask ZenReach" assistant mark: a white speech bubble carrying the brand's rising trend line (in brand blue). Reads as an analyst-in-a-chat — deliberately unlike the… |
| `client/src/components/CurrencyControl.jsx` | 39 | In-page currency picker. One choice re-renders every KPI, chart and table on the page in the selected currency, using real-time FX rates fetched on switch. Rendered… |
| `client/src/components/EChart.jsx` | 41 | Thin, self-contained React wrapper around Apache ECharts. - Registers the app's light/dark theme (baked at init) and re-creates the instance when the theme flips, so… |
| `client/src/components/GoogleCalendarSync.jsx` | 127 | Push ZenReach's scheduled tactics INTO a user's Google Calendar via the service account (immediate, works on localhost). The connection persists (server-side), so… |
| `client/src/components/Hero.jsx` | 103 | Cinematic dark hero, Apple product-page style: aurora gradient, oversized headline that eases up on load, and a parallax drift on the glow. |
| `client/src/components/InfoTip.jsx` | 45 | A tiny "?"-style info icon that reveals a plain-English definition on hover. Used beside finance/marketing terms so non-experts understand what each number means and… |
| `client/src/components/KpiCard.jsx` | 45 | size: 'md' (default, compact — used across the app) or 'lg' (roomier — Home page). `tip` (optional) shows an info tooltip beside the label explaining the metric. |
| `client/src/components/Layout.jsx` | 216 | The everyday workspaces sit at the top of the sidebar; the rest live under a "More" label. A vertical rail has room for everything, so nothing hides in a dropdown. |
| `client/src/components/NotificationBell.jsx` | 86 | Opening the bell counts as "seen": clear the unread badge (optimistically, then confirm with the server so it stays cleared until something new arrives). |
| `client/src/components/PageHeader.jsx` | 31 | Compact, workbench-style page header. Left-aligned title + optional one-line subtitle, with a right-side actions area that always includes the page currency picker… |
| `client/src/components/Panel.jsx` | 26 | Apple-style light content card. |
| `client/src/components/PeoplePicker.jsx` | 144 | App-wide, Teams/Outlook-style people picker. Type to search colleagues by name, email or title; pick one person or several. Each option and chip shows the person's… |
| `client/src/components/Reveal.jsx` | 32 | Staggered entrance animation for a group of children (GSAP). Wrap a grid/row; direct children animate up + fade in on mount. |
| `client/src/components/ScopePicker.jsx` | 101 | Shared plan / campaign scope selector for the Budget & Procurement pages. These pages are reachable on their own from the nav, so they must never show unexplained… |
| `client/src/components/ScrollReveal.jsx` | 91 | Deterministic scroll reveal using IntersectionObserver + CSS transitions. (GSAP ScrollTrigger + React StrictMode could freeze tweens mid-flight, leaving elements… |
| `client/src/components/ScrollToTop.jsx` | 13 | Reset scroll to the top on every route change (and mount/refresh). |
| `client/src/components/SectionHeading.jsx` | 18 | Centered Apple-style section heading used between content blocks. |
| `client/src/components/SnapshotPanel.jsx` | 114 | Capture + gallery of PNG snapshots for one dashboard, stored in the browser (localStorage). `targetRef` points at the DOM region to capture (the widgets grid). |
| `client/src/components/StateWrap.jsx` | 21 | Renders loading / error states consistently; otherwise renders children. |
| `client/src/components/ThemeToggle.jsx` | 26 | Light/dark toggle button bound to ThemeModeContext. |
| `client/src/components/WidgetRenderer.jsx` | 340 | Renders one Builder widget (KPI, gauge, line, area, bar, donut, funnel, table) with thresholds. |
| `client/src/components/auth/AccountControls.jsx` | 150 | Renders the right-hand account area of the nav: - signed out → "Log in" + "Sign up" - signed in → gradient avatar → menu (Profile, Log out) `compact` drops the text… |
| `client/src/components/auth/AuthModal.jsx` | 282 | Login / sign-up modal with demo persona switcher. |
| `client/src/components/auth/Avatar.jsx` | 18 | Gradient initials avatar — the user's assigned gradient with their initials. |
| `client/src/components/auth/ProfileDialog.jsx` | 234 | Profile editing (name, company, password) and team role management for Owners/Admins. |
| `client/src/components/planning/BudgetCostView.jsx` | 198 | Budget & Cost view: planned cost time-phased by month/quarter across the hierarchy. |
| `client/src/components/planning/CalendarView.jsx` | 269 | Marketing calendar month grid with drag-to-reschedule, .ics/print export and connect dialog. |
| `client/src/components/planning/CampaignPlanDetail.jsx` | 586 | Detail for one campaign plan. `planId` is "spa:<campaignId>" or "bcpir:<campaignId>". |
| `client/src/components/planning/HierarchyGrid.jsx` | 570 | Five-level Plan Grid: roll-ups, search, Sys_Gen_Balance rows, honeycomb bulk copy/move/submit/delete, CSV export. |
| `client/src/components/planning/HierarchyTree.jsx` | 99 | The Plan → Campaign → Program tree. Selecting a node filters the board/summary to it; selecting nothing shows the whole plan of record. |
| `client/src/components/planning/ImportBoardDialog.jsx` | 240 | Remember non-secret Jira connection fields for convenience; never persist tokens/keys. |
| `client/src/components/planning/LineItemDialog.jsx` | 219 | Create / edit one line item — the fifth planning level, where a tactic's money is planned month by month (or quarter by quarter) and tracked against actuals. |
| `client/src/components/planning/NodeInspector.jsx` | 409 | Inspection Window for plans, campaigns and programs (Setup / Review tabs, model settings dialog). |
| `client/src/components/planning/PlanCopilot.jsx` | 315 | Closed-loop AI plan UI (generate → commit → monitor → recommend → apply). Not currently mounted. |
| `client/src/components/planning/PlanGrid.jsx` | 153 | Editable, spreadsheet-like grid of tactics with inline edit, row select + bulk status change, and CSV export — Hive9's "Plan Grid" view. Status options come from the… |
| `client/src/components/planning/PlansView.jsx` | 460 | Dual-plan (SP&A / BCPIR) register with plan/campaign CRUD and approval workflow. |
| `client/src/components/planning/TacticDrawer.jsx` | 432 | Tactic inspection drawer: approval tracker, goals waterfall, line items, media codes, performance, activity. |
| `client/src/components/planning/planLevels.js` | 119 | Shared vocabulary for the five-level planning hierarchy: Plan → Campaign → Program → Tactic → Line item Labels, colours, the approval lifecycle, and a client-side… |
| `client/src/components/team/TeamDashboard.jsx` | 294 | Role-aware Home for org managers, leads and members (team × campaign matrix). |
| `client/src/context/AuthContext.jsx` | 75 | On first load, if a token is stored, validate it and restore the session. |
| `client/src/context/CurrencyContext.jsx` | 69 | Global currency: one choice drives every KPI, chart and table across the whole app. Values are stored in USD; the money formatters (lib/format.js) convert to the… |
| `client/src/context/ThemeModeContext.jsx` | 47 | Fall back to the OS preference the first time. |
| `client/src/hooks/useApi.js` | 25 | Minimal data-fetching hook with loading/error state. |
| `client/src/index.css` | 407 | Design tokens (CSS variables, light/dark), utility classes, grid, cards, animations. |
| `client/src/lib/chart.js` | 105 | Apache ECharts foundation for ZenReach. We register a tree-shaken ECharts core (only the chart types + components we use) and two themes — light and dark — so every… |
| `client/src/lib/exporters.js` | 68 | Client-side exporters: iCalendar (.ics) and CSV downloads, plus a print helper. These are genuinely functional and portable (no server, no OAuth) — the honest… |
| `client/src/lib/files.js` | 27 | Read an uploaded file into a payload for the import endpoints: • CSV / text → { csv, name } • xlsx / xls / ods / other spreadsheets → { fileBase64, name } Uses… |
| `client/src/lib/format.js` | 85 | ---- Global currency ------------------------------------------------------ All monetary values in the app are stored in USD (the base). The active currency is held… |
| `client/src/lib/glossary.js` | 102 | Central glossary of marketing / finance / budget terms used across ZenReach. One source of truth so a term means the same thing on every page. Each entry is… |
| `client/src/lib/snapshot.js` | 18 | Capture a DOM node (a rendered dashboard region) to a PNG data URL. ECharts renders to a same-origin <canvas>, which html-to-image captures via toDataURL (not… |
| `client/src/lib/snapshotStore.js` | 67 | Local (browser) store for dashboard snapshots — PNG data URLs kept in localStorage, keyed by dashboard id. ZenReach persists dashboards server-side, but snapshots are… |
| `client/src/main.jsx` | 29 | Stop the browser from restoring the old scroll position on refresh, and start at top. |
| `client/src/pages/Admin.jsx` | 442 | Admin console: org chart, teams, people, demo access. |
| `client/src/pages/Analytics.jsx` | 249 | GA4 analytics: connection banner, realtime users, KPIs, trends, channels, geography, top pages. |
| `client/src/pages/Attribution.jsx` | 203 | Attribution page: funnel, model switcher (9 models), reallocation ranking, forecast. |
| `client/src/pages/AuditLog.jsx` | 231 | Role-scoped audit log with filters and severity markers. |
| `client/src/pages/Budget.jsx` | 277 | Budget page: top-down pools/allocations vs committed, by-channel tab, scope picker, snapshots. |
| `client/src/pages/Builder.jsx` | 732 | Low-code dashboard builder (widget picker, bindings, thresholds, layout, save/rename/delete, dataset upload). |
| `client/src/pages/Campaigns.jsx` | 167 | Campaign performance table with status filters. |
| `client/src/pages/Dashboard.jsx` | 373 | Home: marketing landing (signed out); dashboards viewer or TeamDashboard (signed in); legacy overview. |
| `client/src/pages/Dashboards.jsx` | 165 | Read-only viewer for saved dashboards. Building/editing lives in the Builder — the "New" and "Edit" actions here hand off to it (separation of view vs. build). |
| `client/src/pages/Goals.jsx` | 113 | Goals: target vs actual vs quarter-end forecast. |
| `client/src/pages/Integrations.jsx` | 259 | Integrations: file import with undo, Google Calendar push, connector catalog. |
| `client/src/pages/Optimizer.jsx` | 192 | Compact axis money: $0 · $600K · $1.2M — keeps ticks short so they never overlap or clip. |
| `client/src/pages/Planning.jsx` | 1215 | Planning page shell: view tabs, summary strip, plan selector, hierarchy sidebar, board, tactic editor, dialogs. |
| `client/src/pages/Procurement.jsx` | 382 | Purchase requests: list, approvals inbox, PR drawer, create/submit/approve/fulfil. |
| `client/src/pages/RecycleBin.jsx` | 157 | Recycle bin: restore, permanent delete, empty. |
| `client/src/theme.js` | 154 | Validated categorical palette (from the data-viz reference set). Colours are chosen to stay legible on both light and dark surfaces, so they're shared across modes. |
| `client/tailwind.config.js` | 37 | Tailwind config: content paths, class-based dark mode, token colours, preflight disabled. |
| `client/vite.config.js` | 13 | Vite config: React plugin, port 5173, `/api` proxy to :4000. |
| `server/index.js` | 1189 | Express app: middleware, auth wall, role gates, all 141 routes, audit + notification hooks, boot seeding. |
| `server/services/analysis.js` | 267 | Meaningful analysis layer. 1) computeSignals(): turns raw data into analytical signals (trends, ROI spread, variance, goal gaps, reallocation upside, concentration… |
| `server/services/assistant.js` | 261 | "Ask ZenReach" — the AI marketing analyst. If OPENAI_API_KEY is set, answers are generated by OpenAI's GPT-5 (grounded in the live marketing data below); Azure OpenAI… |
| `server/services/assistantDashboard.js` | 340 | Conversational dashboard builder for the "Ask ZenReach" assistant. When a user asks the assistant to build a dashboard, we run a short, deterministic wizard: name ->… |
| `server/services/assistantWidgets.js` | 141 | Natural-language widget builder for "Ask ZenReach". Turns a free-text request like "add a KPI of blended ROI", "bar of planned vs actual by channel", "revenue over… |
| `server/services/attribution.js` | 215 | Advanced performance measurement — waterfall funnel + multi-touch attribution (up to 9 models) + reallocation ranking + a predictive yearly forecast. Attribution runs… |
| `server/services/audit.js` | 333 | Audit log — an append-only trail of every meaningful action taken in the app (who did what, to what, and when), surfaced ROLE-AWARELY. Two axes decide what a viewer… |
| `server/services/auth.js` | 276 | Lightweight account system for ZenReach. No database — users and sessions live in a JSON file on disk (server/data/users.json) via the same tiny file-store the… |
| `server/services/boardConfig.js` | 132 | Customizable planning board configuration — columns (stages), WIP limits, custom fields, and board templates. A workspace-level config (one board), persisted to… |
| `server/services/boardConnections.js` | 64 | Persisted connections to external Kanban boards (Jira / Trello). When a board is imported (boardImport.js), we record ONE connection here: which provider, a friendly… |
| `server/services/boardImport.js` | 208 | Import a Kanban board from Trello or Jira into ZenReach's Planning module. Both providers are called SERVER-SIDE (from here), so the browser never talks to Trello/… |
| `server/services/budgets.js` | 239 | Budgeting & financial management — Hive9's top-down budgets tied to the plan hierarchy. Model: • Pools — top-down money assigned to a Plan (currency + total). •… |
| `server/services/builder.js` | 303 | Low-code / no-code dashboard builder. The builder lets a user assemble their own dashboard from widgets (KPI tiles, line/area/bar/donut charts, tables) bound to any… |
| `server/services/calendar.js` | 68 | iCalendar (.ics) feed of the marketing calendar — the real, credential-free way to "connect" ZenReach to Google Calendar / Outlook. Served read-only and… |
| `server/services/campaignDataset.js` | 284 | External campaign dataset — ingest a (large) CSV of campaign records, aggregate it into a compact summary, and expose that summary as bindable data sources in the… |
| `server/services/campaignPlans.js` | 390 | Campaign dual-plan model — Shell / Hive9 parity. In Hive9, Shell runs every campaign as a PAIR of plans: • SP&A (Sales, Promotion & Advertising) — the SPEND plan.… |
| `server/services/fx.js` | 61 | Real-time foreign-exchange rates. Pulls live rates from frankfurter.app (European Central Bank reference rates — free, no API key). Rates are normalized to "USD per 1… |
| `server/services/goals.js` | 52 | Goals, revenue modelling & performance-vs-expectation. This is one of Hive9's core capabilities ("align plans to goals through revenue modelling and custom… |
| `server/services/googleAnalytics.js` | 230 | Google Analytics 4 connector. If GA_PROPERTY_ID + service-account credentials are configured, this queries the real GA4 Data API. Otherwise it transparently returns… |
| `server/services/googleCalendar.js` | 264 | Push ZenReach's marketing tactics (events/campaigns) INTO a user's Google Calendar so they show up on their phone. Uses the SAME service account already configured… |
| `server/services/importer.js` | 90 | File-based import — the genuinely-functional side of "integrations" for a demo with no external credentials. Parses a CSV of tactics and creates them. |
| `server/services/insights.js` | 91 | Proactive AI-style insights & anomaly detection — a differentiator vs Hive9, which surfaces data but does little automated interpretation. These are computed… |
| `server/services/mockData.js` | 220 | Deterministic-ish mock data generator for ZenReach demo mode. Produces realistic marketing-performance figures so the whole app works out-of-the-box without any… |
| `server/services/models.js` | 129 | Planning models — Hive9's "rulebook a plan is measured by". Every plan points at one model. The model decides how success is measured (the waterfall: conversion rates… |
| `server/services/notifications.js` | 156 | In-app notifications — strictly per-user. Nothing is shown to a logged-out visitor; a signed-in user sees ONLY the items that concern them: • purchase requests… |
| `server/services/optimizer.js` | 72 | Budget optimizer / scenario planner — a "what-if" tool that recommends reallocating spend toward higher-return channels. Hive9 has no media-mix / optimization layer,… |
| `server/services/org.js` | 460 | Organization hierarchy — Admin → Manager → Lead → Team → Member. This is a SEPARATE axis from the permission role in auth.js (Owner/Admin/Editor/ Viewer, which gates… |
| `server/services/planLoop.js` | 314 | Closed-loop, AI-driven marketing plan — ZenReach's signature differentiator. The loop: GENERATE a plan from a goal + budget (grounded in the app's REAL channel… |
| `server/services/planning.js` | 820 | Tactics — the fundamental executable unit of the marketing plan (Hive9's "tactic"). A tactic is the leaf of the Plan → Campaign → Program → Tactic hierarchy… |
| `server/services/plans.js` | 435 | Marketing planning hierarchy — the structural backbone Hive9 is built on. Hive9 hangs everything off a five-level chain, created strictly in order: Plan → Campaign →… |
| `server/services/procurement.js` | 364 | Purchase requests & approval workflow — Hive9's procurement layer. A purchase request (PR) has vendor line items, is tied to a tactic/campaign, and moves through a… |
| `server/services/recycleBin.js` | 207 | Recycle bin — a soft-delete safety net across the app. Rather than filtering "deleted" flags through every read path, we snapshot an entity (and, for a… |
| `server/services/spreadsheet.js` | 23 | Normalizes an uploaded file to CSV text so the existing CSV aggregators/importers work unchanged. Accepts either raw CSV text or a base64-encoded spreadsheet (.xlsx /… |
| `server/services/store.js` | 43 | Tiny JSON file-store used by the dashboard builder and the planning board. Keeps user-created content (custom dashboards, initiatives) on disk so it survives restarts… |

### A.2 Dashboard Studio (`dashboard-studio/`)

| File | Lines | Responsibility |
|---|---|---|
| `dashboard-studio/index.html` | 21 | HTML shell. |
| `dashboard-studio/postcss.config.js` | 7 | PostCSS: Tailwind + Autoprefixer. |
| `dashboard-studio/server/index.js` | 171 | Dashboard Studio backend. Responsibilities the client no longer does: • parse large CSV/XLSX files (async job + progress) • infer column types • aggregate a widget… |
| `dashboard-studio/server/services/aggregate.js` | 277 | The aggregation engine: (widget config + raw rows) → chart-ready data. One entry point, computeWidgetData(widget, dataset), returns a normalized shape tagged by… |
| `dashboard-studio/server/services/assistant.js` | 254 | Dashboard Studio assistant. Its main job is BUILDING dashboards from the user's uploaded data (a deterministic wizard — see assistantDashboard.js). It is also… |
| `dashboard-studio/server/services/assistantDashboard.js` | 324 | Conversational dashboard builder for the assistant. When the user asks to build a dashboard, we run a short, deterministic wizard: dataset -> name -> WIDGETS… |
| `dashboard-studio/server/services/assistantWidgets.js` | 228 | Natural-language widget builder for the assistant. Turns a free-text request like "a bar chart of revenue by region", "KPI of total amount", "revenue over time", or… |
| `dashboard-studio/server/services/chartTypes.js` | 279 | Chart-type registry + axis "relatability" rules. Each chart type declares the encoding SLOTS it needs. A slot names the column role it accepts (dimension =… |
| `dashboard-studio/server/services/filters.js` | 68 | Dashboard filters — applied server-side to the raw rows BEFORE aggregation, so every widget on a dashboard reflects the same filter set (and huge datasets stay on the… |
| `dashboard-studio/server/services/infer.js` | 61 | Column type + format inference (server-side). Every column is classified as 'number' / 'date' / 'category', with a display `format` hint. |
| `dashboard-studio/server/services/jobs.js` | 25 | In-memory registry of upload/parse jobs. The client polls GET /api/jobs/:id and renders a loader from { status, progress, phase }. |
| `dashboard-studio/server/services/join.js` | 70 | Join two datasets on a shared key column into a NEW materialized dataset. Once joined, the result is an ordinary dataset — every existing feature (query, suggest,… |
| `dashboard-studio/server/services/parse.js` | 60 | Server-side file parsing. Reads a CSV / XLSX / XLS / ODS from a path (uploaded via multer to a temp file), normalizes it into { columns, rows }, and reports coarse… |
| `dashboard-studio/server/services/snapshots.js` | 47 | Dashboard snapshots — a PNG image of a rendered dashboard, stored on disk so it can be revisited later (a lightweight "point in time" record). Files live under… |
| `dashboard-studio/server/services/store.js` | 130 | Persistence. Datasets are kept in server memory for fast aggregation and snapshotted to disk so they survive a restart: data/datasets/index.json — array of dataset… |
| `dashboard-studio/server/services/suggest.js` | 88 | Chart recommender — suggests the best chart(s) for a dataset's columns, each with a plain-English reason. Column types (number / date / category) + cardinality drive… |
| `dashboard-studio/src/App.jsx` | 19 | Routes: landing, builder, viewer. |
| `dashboard-studio/src/components/AiAssistant.jsx` | 254 | Prompt templates surfaced in the "Prompts" tab. Each is filled in with the active dataset's real column names when available, so a tap drops a ready-to-run request… |
| `dashboard-studio/src/components/ChartRenderer.jsx` | 88 | Renders a widget from a SERVER-COMPUTED payload. KPI and Table are HTML; every actual chart is drawn by ECharts via buildOption(). Colors resolve idempotently from… |
| `dashboard-studio/src/components/ChartSkeleton.jsx` | 20 | Small shimmer shown in a widget card while its query is in flight. |
| `dashboard-studio/src/components/ChartThumb.jsx` | 130 | Tiny inline SVG preview for each chart type, so the picker shows WHAT you'll get before you build it. Pure SVG (no assets) using currentColor + a faint fill, so it… |
| `dashboard-studio/src/components/ChatMark.jsx` | 15 | Small chat/spark mark used in the assistant launcher + header. |
| `dashboard-studio/src/components/ColorSwatchPicker.jsx` | 40 | A palette picker that ENFORCES idempotency: colors already taken by sibling data points (passed in `used`) are disabled, so no two points in a widget can share a… |
| `dashboard-studio/src/components/DatasetUpload.jsx` | 71 | Drag-and-drop / click upload. Multiple files can be dropped at once — each is parsed on the server as its own job and becomes its own named dataset. While a job runs,… |
| `dashboard-studio/src/components/EChart.jsx` | 31 | Thin, imperative React wrapper around an ECharts instance. Recreates the option on change (notMerge, so removed series/axes don't linger), resizes with its container,… |
| `dashboard-studio/src/components/JoinDialog.jsx` | 122 | Join two datasets on a shared key into a new dataset the dashboard can visualize. |
| `dashboard-studio/src/components/Layout.jsx` | 60 | App chrome and navigation. |
| `dashboard-studio/src/components/PageHeader.jsx` | 15 | Page title header. |
| `dashboard-studio/src/components/ProcessingLoader.jsx` | 48 | A distinctive "assembling bars" loader shown while the server parses a dataset. Bars rise and fall in a gradient wave — evoking a chart building itself — with the… |
| `dashboard-studio/src/components/QueryChart.jsx` | 20 | Fetches a widget's server-computed data and renders it. |
| `dashboard-studio/src/components/SnapshotPanel.jsx` | 95 | Capture + gallery of PNG snapshots for one dashboard. `targetRef` points at the DOM region to capture (the rendered dashboard grids). |
| `dashboard-studio/src/components/WidgetEditor.jsx` | 416 | Median of numeric values — seeds threshold suggestions for charts (bar/line/heatmap). |
| `dashboard-studio/src/context/DatasetsContext.jsx` | 97 | Holds the list of server-side datasets (metadata + columns only — never rows) and drives file uploads through their async parse jobs so the UI can show a loader. |
| `dashboard-studio/src/context/ThemeModeContext.jsx` | 33 | Light/dark mode state. |
| `dashboard-studio/src/hooks/useDashboards.js` | 44 | Server-backed dashboards CRUD. Names are unique (enforced by the server too). |
| `dashboard-studio/src/hooks/useWidgetQuery.js` | 48 | A small module-level cache so identical widget queries (across cards / re-renders) hit the server once. Keyed by everything that affects the COMPUTED result — the… |
| `dashboard-studio/src/index.css` | 154 | Apple-inspired light system (default) |
| `dashboard-studio/src/lib/api.js` | 88 | Thin fetch wrapper over the backend. Everything heavy (parse/aggregate) is server-side; the client only sends configs and receives small payloads. Robustness: the API… |
| `dashboard-studio/src/lib/chartOption.js` | 427 | Pure mapper: a server-computed payload → an ECharts option object. One function, buildOption(widget, computed, { mode, colorOf, tokens }), returns { option, height }.… |
| `dashboard-studio/src/lib/chartTypes.js` | 274 | Chart-type registry + axis "relatability" rules. Each chart type declares the encoding SLOTS it needs. A slot names the column role it accepts (dimension =… |
| `dashboard-studio/src/lib/echartsCore.js` | 23 | Modular ECharts build — register only the chart types and components the studio uses, so the bundle stays lean (canvas renderer only). Every chart in the app goes… |
| `dashboard-studio/src/lib/format.js` | 91 | Value formatting helpers. `format` is a per-field hint inferred at parse time or set by the user: 'number' / 'money' / 'percent' / 'text' / 'date'. |
| `dashboard-studio/src/lib/palette.js` | 80 | The fixed color palette for the whole studio. These are the ONLY colors a user may assign to data points/series. Slots 1–8 are the CVD-validated categorical order… |
| `dashboard-studio/src/lib/snapshot.js` | 16 | Capture a DOM node (a rendered dashboard region) to a PNG data URL. ECharts canvases are captured as-is. Background is set so dark mode doesn't come out transparent. |
| `dashboard-studio/src/main.jsx` | 20 | React bootstrap. |
| `dashboard-studio/src/pages/Builder.jsx` | 327 | Dashboard builder: dataset binding, widget grid, editor, pin/hide, snapshots. |
| `dashboard-studio/src/pages/Landing.jsx` | 63 | Landing page with upload entry point. |
| `dashboard-studio/src/pages/Viewer.jsx` | 104 | Read-only viewer for saved dashboards. Building/editing happens in the Builder. |
| `dashboard-studio/src/theme.js` | 76 | Chart theme tokens per mode. ECharts renders to canvas, which cannot read CSS variables — and React runs child effects before the ancestor that toggles the `.dark`… |
| `dashboard-studio/tailwind.config.js` | 26 | @type {import('tailwindcss').Config} |
| `dashboard-studio/vite.config.js` | 13 | The client is a thin front-end: parsing + aggregation happen on the backend, reached through the /api proxy. |
