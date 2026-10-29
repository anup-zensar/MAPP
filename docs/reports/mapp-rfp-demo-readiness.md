# MAPP RFP Demo Readiness

28 October 2026

Up to MVP go-live (RFP Phases 1 and 2, 17 rows), **about 60% is done**. We can take it to **about 65% without Shell**; **the last 35% needs Shell**. The product build itself is **86% done** and can reach **100% without Shell**.

| Area | Rows | Done today | Without Shell | Needs Shell |
| --- | --- | --- | --- | --- |
| Phase 1: foundation and design | 6 | 58% | 58% | 42% |
| Phase 2: product build | 7 | 86% | 100% | — |
| Phase 2: configuration, migration, UAT, go-live | 4 | 13% | 13% | 87% |
| **Total** | **17** | **59%** | **65%** | **35%** |

## What is there: ready to demo

- **Planning:** plans, campaigns, initiatives, activities and line items; yearly, quarterly and monthly views; copy last year's plan
- **Budgets:** monthly phasing, global budgets split to markets, exchange rates in local currency and USD
- **Approvals:** multi-level, auto-approval under a threshold, delegation, missing data blocks submission
- **Access:** market vs global, edit vs view vs approve
- **Settings:** attributes, currencies, roles, teams and workflows, run by Shell admins
- **Purchase request to PO,** on a simulated GSAP

## What we can do without Shell

- SP&A view and BCPIR marker, as defined in Shell's spec 006
- Excel import and bulk edit of plans
- Executive and regional dashboards; export to PowerPoint and PDF
- Overspend control (hard stop or warning), with a default rule Shell can change
- Finish the open screens: role-aware buttons, attribute filters, field help, approvers' roles from Settings
- The 12 missing end-to-end tests

## What we cannot do without Shell

| Item | Needs from Shell |
| --- | --- |
| **Requirements and operating model sign-off** | Answers to 41 open questions; sign-off on 58 defaults |
| **Mobility and Lubricants configuration** | Real markets, users, roles and option lists |
| **Hive9 data migration** | Hive9 data dump, mapping sign-off, cut-over rules |
| **Deployment and single sign-on** | Azure@Shell environments; Entra ID app registration |
| **UAT and go-live** | Business SMEs and UAT time |
| **GSAP integration** (after Nucleus, ~Mar 2027) | Interface specs and sample files |

Scoring: covered = 1, mostly = 0.75, partly = 0.5, not = 0 per RFP row; vendor contracting left out. An estimate, not a measurement.
