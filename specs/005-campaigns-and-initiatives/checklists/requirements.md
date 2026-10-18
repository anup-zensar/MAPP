# Specification Quality Checklist: Campaigns and Initiatives

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-10-18
**Feature**: [spec.md](../spec.md)

## Content Quality

- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

## Requirement Completeness

- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [x] Success criteria are technology-agnostic (no implementation details)
- [x] All acceptance scenarios are defined
- [x] Edge cases are identified
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

## Feature Readiness

- [x] All functional requirements have clear acceptance criteria
- [x] User scenarios cover primary flows
- [x] Feature meets measurable outcomes defined in Success Criteria
- [x] No implementation details leak into specification

## Notes

- Validated 2026-10-18 (baseline, decision D-021). 31 FRs, 8 SCs, 6 user stories (4 × P1, 2 × P2);
  no [NEEDS CLARIFICATION] markers — open points are documented assumptions.
- Create (stories 1–2), view and edit (story 3), delete (story 5), move and copy (story 6) follow design
  system §6 with side panels (§6.2). SC-001 uses the design system §10 targets (campaign ≤ 5 steps,
  initiative ≤ 3 steps).
- Fields and rules are referenced (field catalogue §2–3; domain model §2.2, §2.6, §4.1–4.2, §4.5, R2–R4,
  R8, R9, R13, R14), not redefined. Details added here: Timing and remaining budget when dates or the
  parent budget are missing, "0 of n approved" wording, a cluster with no regions never blocks publishing,
  date narrowing is checked against children, R4 also warns for dates before 1 Jan.
- Move rule: a destination whose dates do not contain an initiative's dates is shown but not selectable
  (no automatic date changes, no re-approvals). Campaigns move only within the same year.
- Boundaries: the plan tree frame, summary cards and publish check belong to 004; activity rows,
  **Add activity** and activity-date rules to 006; bulk move/copy to 012; history and recycle-bin screens
  to 011.
- Length is 352 lines, slightly above the ~340 guide, because two record types share every requirement.
- Items marked incomplete require spec updates before `/speckit-clarify` or `/speckit-plan`.
