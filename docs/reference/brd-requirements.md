# Shell Business Requirements (BRD) — transcription

A faithful transcription of the client's latest Business Requirements document, so that specs, decisions
and reports can refer to each requirement by ID. The wording of the requirements and of the client's
comments is kept exactly as written in the source, including its spelling.

| | |
|---|---|
| **Source** | `Business Requirements.pdf` (2 pages), received from Shell; read on 2026-10-20 |
| **Rows** | 36: 11 MVP Requirement · 14 Future-State/Later · 6 RFP/Selection Requirement · 5 Needs Clarification |
| **Requirement source** | "Business" on every row |
| **Rows without an ID** | 4 rows have no requirement ID or area in the source. They were added in a workshop (the comment says "Workshop-added requirement captured from Miro sticky note"). This document numbers them **BUS.WS.01–BUS.WS.04** in the order they appear, and marks them *workshop-added*. These IDs are MAPP's, not Shell's. |
| **Order** | As in the source: MVP rows first, then Future-State/Later, RFP/Selection and Needs Clarification |
| **Gap analysis** | [../reports/brd-gap-analysis.md](../reports/brd-gap-analysis.md) compares every row with MAPP |

Line breaks inside a client comment are shown as `<br>`. "—" means the source cell is empty.

## MVP Requirement (11)

| ID | Area | Requirement | RFP disposition | Client comment |
|---|---|---|---|---|
| BUS.ADM.020 | Administration & Access | Does the application support team-based and role-based access control (RBAC) for plans, POs, invoices and related financial data, configurable through access rules rather than code. | MVP Requirement | Also, flexibility to control who can edit vs who can only view. |
| BUS.BUD.020 | Budget Management | Support various budget categories, e.g. region, brand, spend type, business unit, channel. | MVP Requirement | 1) flexibility to update attributes by client.<br>2) Tool should be flexible to accommodate the framework or hierarchy basis the need of the business. |
| BUS.BUD.040 | Budget Management | Ability to Global/Local budget allocation with appropriate visibility for global and local budget owners respectively. | MVP Requirement | — |
| BUS.BUD.050 | Budget Management | Support currency conversion in planning, analysis and reporting, exchange-rate management, and whether it can consume daily conversion rates from GSAP. | MVP Requirement | Ability to plan in local currency and USD |
| BUS.FIN.080 | Financial Integration (PR/PO) | Are financial transactions (POs, invoices, payments, accruals) surfaced prominently in the user's main working views rather than buried several levels deep in the navigation. | MVP Requirement | UI interface to be simple to access and intuitive for users to navigate. Example is mentioned in the note on financial transactions. |
| BUS.PLN.020 | Planning Framework | Support multi-year, annual, quarterly and monthly planning; strategic, operational and cross-market planning; and future or custom development options. | MVP Requirement | — |
| BUS.PLN.040 | Planning Framework | Ability to users to split their budget by month and to split an activity from its original budget. | MVP Requirement | Phasing and ongoing updating of budget |
| BUS.PLN.050 | Planning Framework | Describe how your solution enables re-use of prior-year planning content when starting a new planning cycle. | MVP Requirement | Flexibility in the tool to reuse the historical or prior year plans/inputs to save time from recreating from scratch |
| BUS.UX.020 | User Experience | Ability to display contextual help ('info' icons or equivalent) explaining what data is expected in a particular attribute or field, maintainable by Shell administrators. | MVP Requirement | — |
| BUS.WFL.010 | Workflow & Approvals | Support multi-level approvals, parallel approvals, conditional approvals, delegation of authority, automated notifications, escalations, role-based approvals, segregation of duties, compliance controls, audit trail, workflow monitoring and process dashboards. | MVP Requirement | 1) Support multi layered approval approach that is auto approval/manager approval - So approval flexibility.<br>2) To be simplified between approval workflow + approval delegations/control + Approval tracking/notification |
| BUS.WFL.030 | Workflow & Approvals | Does the application have functionality to highlight missing mandatory data and halt a submission until the required data is completed. | MVP Requirement | — |

## Future-State/Later (14)

| ID | Area | Requirement | RFP disposition | Client comment |
|---|---|---|---|---|
| BUS.AI.010 | AI & Advanced Analytics | Support current and planned AI capabilities across forecasting, scenarios, analysis, reporting, workflow, optimisation and assistants; clearly distinguish generally available, controlled-release and roadmap features. | Future-State/Later | To be simplified - Product should be future ready and flexible to accommodate shell owned AI capabilities vs available outside. For example: Automation of plans copying from previous year to create a skeleton for next year (broad example). |
| BUS.BUD.030 | Budget Management | Does the application allow budget owners to restrict or block overspending against approved budgets, and to configure whether this is a hard stop or a warning. | Future-State/Later | include underspend. Not a blocker but a warning |
| BUS.FIN.020 | Financial Integration (PR/PO) | Does the application allow for full automation of the PR/PO process, from requisition through approval to PO creation and closure. | Future-State/Later | 1) Move to tech not business.<br>2) Also, no vendor can understand the PR/PO process so difficult to answer whether it can be done or not.<br>Source retained as Business pending final source/category decision. |
| BUS.FIN.040 | Financial Integration (PR/PO) | Does the application calculate accrual amounts on a monthly basis for each PO using committed amounts and invoices received. | Future-State/Later | — |
| BUS.FIN.050 | Financial Integration (PR/PO) | Does the application allow marketers to review supplier invoices received against their POs and approve or decline them. | Future-State/Later | — |
| BUS.FIN.060 | Financial Integration (PR/PO) | Ability to map a PO PDF to the originating PR so that the marketer has direct visibility of the document. | Future-State/Later | 1) Ability to get copies of invoices and POs to be used instead of text in the box.<br>2) The ability to get copies of invoices and purchase orders.<br>3) Keep the new wording in the comment |
| BUS.PRF.020 | Performance & PIR | Ability to attach a Post Investment Review (PIR) to a Business Case / SP&A Activity so that the investment and its outcome stay linked. | Future-State/Later | Ability to track and connect all phases of planning to performance. For example, Business case to Plans to PIR |
| BUS.PRF.030 | Performance & PIR | Ability to view all Business Cases and Post Investment Reviews from one central location, including for users who do not own them. | Future-State/Later | 1) Broader reference to the data management. Does platform have capability to do business case and performance evaluation.<br>2) To be rephased as - Ability to view all the content (KPIs, plans, budget, PIRs etc) from a central dashboard with controls kept at user profile or access level i.e. Global can view all data for all markets vs local market can view all data for their market. |
| BUS.RPT.010 | Reporting & Dashboards | Standard reporting capabilities: executive dashboards, regional and operating-unit dashboards, campaign, portfolio and budget dashboards, original business case vs forecast financial reporting, performance reporting, benchmarking and KPI dashboards. | Future-State/Later | — |
| BUS.RPT.050 | Reporting & Dashboards | Ability to provide multiple calendar views, e.g. by marketing campaign, budget owner, time period or country. | Future-State/Later | — |
| BUS.UX.010 | User Experience | Web and mobile interfaces (including sync after offline planning), accessibility compliance, multi-language support, Excel integration, bulk upload and editing, personal dashboards, notifications and search functionality. | Future-State/Later | — |
| BUS.WFL.040 | Workflow & Approvals | Does the application allow notifications to be sent to budget owners when changes are made to plans, budgets or activities they own. | Future-State/Later | Can I customise notifications and select how to rec notifications |
| BUS.WS.01 *(workshop-added)* | — | Ability to export our plans into any format (ppt, pdf, word, excel etc.) or share via a click with MS outlook or teams or other platform | Future-State/Later | Workshop-added requirement captured from Miro sticky note. |
| BUS.WS.02 *(workshop-added)* | — | Flexibility to import the skeleton of plan and upload the updated plan using excels. | Future-State/Later | Workshop-added requirement captured from Miro sticky note. |

## RFP/Selection Requirement (6)

| ID | Area | Requirement | RFP disposition | Client comment |
|---|---|---|---|---|
| BUS.ADM.010 | Administration & Access | Support administrative work such as configuration, role/permission/audit management, workflow configuration and release management, and state which of these Shell can perform without vendor involvement. | RFP/Selection Requirement | More control at Shell side for small configurations without a significant reliance on vendor i.e. DIY capabilities |
| BUS.ADM.050 | Administration & Access | Integrate Shell's current and historical Hive9 data so it can be used seamlessly for planning in the future platform. | RFP/Selection Requirement | — |
| BUS.FIN.010 | Financial Integration (PR/PO) | Describe GSAP integration of purchase requisitions, purchase orders, master data, goods receipts, invoices, commitments, accruals, actual spend, cost centres, internal orders, WBS elements, event integrations, monitoring, reconciliation, error handling and retry mechanisms. | RFP/Selection Requirement | Move to technical not business.<br>Source retained as Business pending final source/category decision. |
| BUS.INT.010 | Integration & Interfaces | Describe standard integrations, e.g. Power BI, SAP, Azure, Salesforce, Adobe, Microsoft Teams, SharePoint, Outlook. State which are productised connectors and which require a custom build. | RFP/Selection Requirement | Move to tech not business.<br>Source retained as Business pending final source/category decision. |
| BUS.WS.03 *(workshop-added)* | — | TO BE DECIDED - Ability to support multiple time zone and having offices available is a preference | RFP/Selection Requirement | Workshop-added requirement captured from Miro sticky note. Decision pending. |
| BUS.WS.04 *(workshop-added)* | — | Vendor alignment & tool ability to share and maintain the data of previous 5 years for users to refer back in case they need it. | RFP/Selection Requirement | Workshop-added requirement captured from Miro sticky note. |

## Needs Clarification (5)

| ID | Area | Requirement | RFP disposition | Client comment |
|---|---|---|---|---|
| BUS.BUD.010 | Budget Management | Support budget and activity ownership, approvals, transfers, locking, freeze, multi-currency budgeting and budget carry-forward. | Needs Clarification | Duplicate |
| BUS.PLN.010 | Planning Framework | Ability to create the framework for budget planning, tracking and analysis (global template, standard hierarchy, standard attributes) that regional and market teams then plan within. | Needs Clarification | Captured in MVP in Green |
| BUS.PLN.030 | Planning Framework | Support rolling forecasts, planned-spend phasing, budget allocation and reallocation, scenario planning and future or custom development options. | Needs Clarification | Duplicate |
| BUS.PLN.060 | Planning Framework | Ability to in-cell / in-field formulas where applicable, so planners can derive values without exporting to spreadsheets. | Needs Clarification | hive 9 specific and not required in the new version of the application. |
| BUS.RPT.060 | Reporting & Dashboards | Does the application have intuitive filter capability across list views, dashboards and reports, including saved filter sets. | Needs Clarification | Duplicate |

## Notes on the source

- **BUS.FIN.080** refers to "the note on financial transactions". That note is not part of this PDF;
  MAPP needs a copy before the requirement can be specified in detail.
- "Captured in MVP in Green" (BUS.PLN.010) refers to the green MVP rows of the source (the first 11
  rows above).
- Three rows (BUS.FIN.020, BUS.FIN.010, BUS.INT.010) are marked by the client to move to the technical
  requirements, with "Source retained as Business pending final source/category decision".
  BUS.WS.03 is marked "TO BE DECIDED" and "Decision pending".
