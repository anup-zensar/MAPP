# Demo Users and Seed Data

NorthStar ships with a **fictitious** dataset so every feature can be demonstrated and tested. No real
Shell person, figure or code is used; option lists are realistic samples until Shell supplies the real
lists from a production data extract. The seed is loaded by an explicit command (defined in feature
001's plan) and never in a production environment.

Features add their own seed records; this page is the shared story they extend.

## Demo users (demo sign-in)

All demo users share the password **`northstar`**, shown on the sign-in page (decision D-004).

| Name | E-mail | Job title | Teams | Typical use |
|---|---|---|---|---|
| Alex Morgan | alex.morgan@northstar.demo | Marketing administrator | Global Marketing Ops | Settings, approval level 2 |
| Priya Raman | priya.raman@northstar.demo | Marketer | Lubricants Europe | Builds the Lubricants plan |
| Daniel Okoro | daniel.okoro@northstar.demo | Marketing team leader | Lubricants Europe | Approves Lubricants activities |
| Mei Lin | mei.lin@northstar.demo | Marketer | Mobility China | Builds the Mobility plan |
| Carlos Mendes | carlos.mendes@northstar.demo | Business champion | Mobility China, Global Marketing Ops | Approves Mobility activities (level 1) |
| Sara Novak | sara.novak@northstar.demo | Local finance advisor | Lubricants Europe, Mobility China | Reviews finance coding |

## Settings seed

| Record | Seed values |
|---|---|
| Teams | Lubricants Europe · Mobility China · Global Marketing Ops |
| Models | **Lubricants 1.0** — INQ→MQL 2 %, MQL→deal 5 %, deal size 10,000 USD · **Mobility 1.0** — 3 %, 4 %, 5,000 USD |
| Activity types | Agency fees – Creative · Event · Digital media · Content production · Sponsorship · MOBILITY – Long-term contract (Mobility only) |
| Currencies (units per 1 USD) | USD 1 · EUR 0.92 · GBP 0.79 · CNY 7.10 · INR 83.50 |
| Finance budget items (2026) | *2026 Lubricants Budget:* Europe – Digital · Europe – Events · Europe – Brand · *2026 Mobility Budget:* China – Retail · China – Fleet · Delhi |
| Approval workflows | **Lubricants EU approval** — level 1: Daniel Okoro · **China BC approval workflow** — level 1: Carlos Mendes; level 2: Alex Morgan |
| Shell attributes | The full set in the field catalogue with 3–8 sample options each (e.g. Cluster: Lubricants – Europe, Retail – China, Fleet – China; Region per cluster; Company code: GB01, NL02, CN04; Spend type → GL code pairs such as "Digital – Platform development → 7250080 – Advertising digital") |

## Planning seed (the demo story)

| Plan | State | Contents |
|---|---|---|
| **FY2026 Lubricants Europe** (Priya Raman, Lubricants Europe, Lubricants 1.0, budget 1,200,000 USD) | Published | 2 campaigns ("Helix launch", "Rimula fleet"), 4 initiatives, 7 activities in every status (Created, Submitted, Declined, Approved/In progress, Complete), line items in USD, EUR and GBP, one activity with a non-zero unassigned balance |
| **FY2026 Mobility China** (Mei Lin, Mobility China, Mobility 1.0, budget 800,000 USD) | Draft | 1 campaign, 2 initiatives, 3 activities — deliberately incomplete (missing cluster/region, missing GL code, Country/OU split at 90 %) to demonstrate completeness |
| **FY2027 Global Brand** (Alex Morgan, Global Marketing Ops, Lubricants 1.0) | Draft, Strategic / AOP | 1 campaign with an over-allocated initiative budget (negative remaining budget) |

The recycle bin contains one deleted initiative (with its activity) and the activity log contains about
two weeks of back-dated history, so those screens are not empty in a demo.
