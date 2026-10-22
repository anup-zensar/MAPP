# PRISM Phase 1 Findings (summary)

*Shell Global — PRISM: Phase 1 Findings and Future-State Requirements*, prepared by Sidekick, May 2026.
Evidence: global user survey (120), focus groups (31), senior leadership interviews (7), process mapping,
ideal-state and gap workshops, 9 market interviews.

## Main findings

- The planning environment is **governed but not decision-capable**: approvals, structure and cross-market
  visibility work; plan → actual → ROI continuity, trusted real-time data and in-flight decisions do not.
- **Hive9 is used to submit plans, not to create them** — planning happens in Excel and local tools.
- The **Hive9–GSAP handoff** is the main break: manual handoffs, duplicate entry, approval delays, little
  PR/PO visibility. **PR/PO scored lowest (about 2.5–2.9)** in the survey.
- Reconciliation with GSAP is a permanent manual activity; outputs are not trusted without checking.
- Lubricants and Mobility & Convenience show different symptoms but the same structural problems.

## Design principles for the future state

Standardise the core (definitions, taxonomy, required fields, source-of-truth rules) · establish data
trust first · focus on core capabilities · integrate, do not replicate (GSAP stays the system of record) ·
streamline governance (keep control, remove unnecessary approvals) · use AI as an enabler only after the
data foundation is trusted.

## Prioritised requirements (MoSCoW)

- **Must:** seamless integration (R01), single data flow across the lifecycle (R02), end-to-end financial
  continuity (R04), trusted plan-to-actual visibility (R06), standardised data model and taxonomy (R03),
  data validation and reconciliation controls (R08), consistent FX treatment (R05), resilience and
  supportability (R21), a clear GSAP/planning boundary (R07), security and compliance (R22).
- **Should:** planning capture and structured workflows, plan-to-actual and PR/PO visibility, cross-market
  comparability, PIR/ROI support.
- **Could / not now:** UX polish, advanced analytics or AI, vendor differentiators, extended
  configurability.

## What it means for MAPP's Plan module

- Make planning *in* the tool easier than in Excel (friendly CRUD, drafts, completeness, no
  one-error-per-save).
- Standardise data at entry: configured attribute lists, required finance coding on line items, one FX
  rule (base USD), readable IDs that will key the future GSAP link.
- Keep the GSAP boundary clean: MAPP records the SWBS request; it never creates financial truth.
