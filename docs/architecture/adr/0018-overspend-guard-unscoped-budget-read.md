# ADR-0018: The overspend guard reads every budget without the actor's scope

**Status:** Proposed (awaiting the product owner's approval) · **Date:** 2026-10-29

## Context
Constitution VII and B5 require every read of plan-owned data to be filtered by the signed-in actor's scope,
and Governance → Exceptions requires an approved ADR for a deviation from a MUST. Feature 037 (overspend
control) refuses a save that takes a budget set to Refuse over its amount. The rule runs in one guarded
transactor (`overspend.Guard`, research R3) around about twenty write paths: it reads every budget with its
planned cost before and after the save and refuses when one got worse. A global budget's planned cost is the
sum of every plan it allocates to, across markets (FR-008), and a save cannot say beforehand which records it
will reach (moves, copies, restores and Undo create or bring back records).

## Decision
- `overspend.Budgets` (`ListOverspendBudgets`, `ListOverspendGlobalBudgets`) reads every plan, campaign,
  initiative and global budget without the actor's scope. Only `overspend.Guard` calls it, inside a save's
  write transaction, and only to apply the refusal rule (FR-009).
- It reads only what the rule needs: kind, ID, name, overspend mode, amount and planned cost, and the IDs of
  the plans a global budget allocates to.
- Nothing it reads reaches the caller except, in the 409 `over_budget` refusal, the name, amount, planned cost
  and overage of the budgets the actor's own save took over (FR-010). It never feeds a list, a view, a
  filter or a count; every other read of these records stays scoped and keeps its isolation tests.
- A new caller of `overspend.Budgets` needs its own decision.

## Consequences
- Refuse holds for every market's saves, including those that take a global budget over (SC-001).
- A planner whose save would overspend a global budget learns its name, amount and total planned cost; the
  plan page already shows that global budget and its allocation (014 FR-031), so this was accepted in the
  037 threat model.
- There is no scope-isolation test for this read; the API tests in `backend/internal/overspend` prove the
  refusal names only the budgets the save reaches.
- Recorded as D-355 and in the Complexity Tracking of `specs/037-overspend-control/plan.md`.
