# Demo Users and Seed Data

MAPP (shown as SP&A One) ships with a **fictitious** dataset so every feature can be demonstrated and tested. No real
Shell person, figure or code is used; option lists are realistic samples until Shell supplies the real
lists from a production data extract. The seed is loaded by an explicit command (defined in feature
001's plan) and never in a production environment.

Features add their own seed records; this page is the shared story they extend.

## Demo users (demo sign-in)

All demo users share the password **`mapp`**, shown on the sign-in page (decision D-004).

| Name | E-mail | Job title | Teams | Typical use |
|---|---|---|---|---|
| Alex Morgan | alex.morgan@shell.com | Marketing administrator | Global Marketing Ops | Settings, approval level 2 |
| Priya Raman | priya.raman@shell.com | Marketer | Lubricants Europe | Builds the Lubricants plan |
| Daniel Okoro | daniel.okoro@shell.com | Marketing team leader | Lubricants Europe | Approves Lubricants activities |
| Mei Lin | mei.lin@shell.com | Marketer | Mobility China | Builds the Mobility plan |
| Carlos Mendes | carlos.mendes@shell.com | Business champion | Mobility China, Global Marketing Ops | Approves Mobility activities (level 1) |
| Sara Novak | sara.novak@shell.com | Local finance advisor | Lubricants Europe, Mobility China | Reviews finance coding |
| Lea Santos | lea.santos@shell.com | Requisitioner | Global Marketing Ops | Role *Requisitioner* (All markets): the Purchase requests page, PR forms and the simulated GSAP (feature 032) |

**Roles** (feature 014; Settings › Roles and access): Alex Morgan — Administrator (All markets) · Carlos Mendes —
Global planner (All markets) · Daniel Okoro — Planner and Approver (My teams) · Priya Raman and Mei Lin — Planner
(My teams) · Sara Novak — Viewer (My teams) · Lea Santos — Requisitioner (All markets). New people get the Viewer
role, and nobody approves an activity they own or submitted. Each persona therefore sees a different part of the
data: Priya only Lubricants Europe, Mei only Mobility China, Sara both read-only, Carlos and Alex everything.

## Settings seed

| Record | Seed values |
|---|---|
| Teams | Lubricants Europe · Mobility China · Global Marketing Ops |
| Models | **Lubricants 1.0** — INQ→MQL 2 %, MQL→deal 5 %, deal size 10,000 USD · **Mobility 1.0** — 3 %, 4 %, 5,000 USD |
| Activity types | Agency fees – Creative · Event · Digital media · Content production · Sponsorship · MOBILITY – Long-term contract (Mobility only) |
| Currencies (units per 1 USD) | USD 1 · EUR 0.92 · GBP 0.79 · CNY 7.10 · INR 83.50 |
| Finance budget items (2026) | *2026 Lubricants Budget:* Europe – Digital · Europe – Events · Europe – Brand · *2026 Mobility Budget:* China – Retail · China – Fleet · Delhi |
| Line managers | Priya Raman and Sara Novak → Daniel Okoro · Mei Lin → Carlos Mendes · Daniel Okoro and Carlos Mendes → Alex Morgan · Alex Morgan has none |
| Approval workflows | **Lubricants EU approval** (WF-1001) — level 1: Daniel Okoro, backup Alex Morgan; remind after 1 day, escalate after 2 (short, so a demo shows both on the first run) · **China BC approval workflow** (WF-1002) — level 1: Carlos Mendes, backup Alex Morgan; level 2: Alex Morgan, backup Carlos Mendes; remind after 3 days, escalate after 5 · **Standard marketing approval** (WF-1003, used by no seeded activity) — level 1: the owner's line manager, approved automatically below 10,000 USD, backup Alex Morgan; level 2: Alex Morgan, only needed from 100,000 USD, backup Carlos Mendes; level 3 "Mobility second check": Daniel Okoro and Alex Morgan, all must approve, only for the Mobility business unit, backup Carlos Mendes; remind after 3 days, escalate after 5 |
| Approval delegations | **DLG-1001** — Daniel Okoro → Carlos Mendes from today + 7 days to today + 11 days, note "Annual leave" (planned, so it changes nothing in today's lists) |
| Shell attributes | The full set in the field catalogue, with 3–8 sample options each (e.g. Cluster: Lubricants – Europe, Retail – China, Fleet – China; Region per cluster; Company code: GB01, NL02, CN04), except Shell's own lists for the zero-based cost hierarchy (10), spend type (25), GL code (13), spend category (19) and contracted spend (5), e.g. spend type "ATL - Digital" with GL code "7250080 - Advertising - Digital" (D-325) |

## Planning seed (the demo story)

| Plan | State | Contents |
|---|---|---|
| **FY2026 Lubricants Europe** (Priya Raman, Lubricants Europe, Lubricants 1.0, budget 1,200,000 USD) | Published | 2 campaigns ("Helix launch", "Rimula fleet"), 4 initiatives, 7 activities in every status (Created, Submitted, Declined, Approved/In progress, Complete), line items in USD, EUR and GBP, one activity with a non-zero unassigned balance |
| **FY2026 Mobility China** (Mei Lin, Mobility China, Mobility 1.0, budget 800,000 USD) | Draft | 1 campaign, 2 initiatives, 3 activities — deliberately incomplete (missing cluster/region, missing GL code, Country/OU split at 90 %) to demonstrate completeness |
| **FY2027 Global Brand** (Alex Morgan, Global Marketing Ops, Lubricants 1.0) | Draft, Strategic / AOP | 1 campaign with an over-allocated initiative budget (negative remaining budget) |

Global budgets (feature 014): **FY2026 Global marketing budget** (2026, 2,500,000.00 USD, owner Carlos Mendes)
allocates 1,200,000.00 to FY2026 Lubricants Europe and 800,000.00 to FY2026 Mobility China, equal to their
budgets, so those plan budgets are read-only for their owners.

The recycle bin contains one deleted initiative (with its activity) and the activity log contains about
two weeks of back-dated history, so those screens are not empty in a demo.

Approval rules (feature 017) in the demo: the open round of **Trade magazine ads** (WF-1001) has waited two
days, so it is overdue on the Approvals monitoring and the first **Run reminders and escalations** escalates
it to Alex Morgan; **Fleet card launch** (WF-1002) waits at level 2. Submitting a Lubricants activity below
10,000 USD with WF-1003 shows an activity approved automatically.

Purchase requests (features 032 and 033) in the demo: Purchase Support is on and lines may join an approved
purchase request with modification approval. Two workflows approve them — **Purchase request approval (MoA)**
(WF-1004) and **Purchase request change approval (MoA)** (WF-1005), one level each that goes to the purchase
request's MoA approver, backup Alex Morgan, reminder after 3 days and escalation after 5. **PR-1001** "PR for
Agency fee (Ireland)" waits as Created on the UK plan's line item L-1010 for 23,000.00 EUR, delivery
30 Apr 2027, owner Priya Raman, MoA approver Daniel Okoro, vendor Brightside Media BV, Email group
G_GB_RETAIL. The Email group options include **DE_LUBES**, the recipient the walkthroughs route to.
