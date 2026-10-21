---

description: "Task list for feature 018 — Pinned Plans"
---

# Tasks: Pinned Plans

**Input**: [plan.md](plan.md), [spec.md](spec.md); features 001, 003 and 004 built.

**Tests**: required — constitution V. Test names carry the FR they prove.

## Format: `[ID] Description`

- [ ] T001 Backend data: `PinnedPlanIDs` on `Preferences` (stored, omitted when empty), `PreferencesView`
      (always sent, never nil) and `PreferencesUpdate` in `backend/internal/people/model.go`; defaults and
      `withDefaults` keep it never nil; `equal` replaces the struct compare in `UpdatePreferences`.
- [ ] T002 Backend rules in `backend/internal/people/service.go`: dedupe keeping order (`uniqueIDs`), every
      id `^P-\d+$`, at most 100, unknown ids allowed, not logged (D-038); texts in `messages.go`.
- [ ] T003 Backend service tests named after FR-004 – FR-006 (round trip, refusals, dedupe, cap) in
      `service_test.go`; store round trip with pinned ids in `store_json_test.go`.
- [ ] T004 Contract: `pinnedPlanIds` on `Preferences` (required, insertion order) and
      `PreferencesUpdate` (optional, `maxItems` 100, items `^P-\d+$`) in `backend/api/openapi.yaml`.
- [ ] T005 Extend every test that asserts an exact `/api/me` or preferences body:
      `backend/internal/people/handler_test.go` (incl. `defaultPreferencesBody`),
      `backend/internal/auth/handler_test.go`; new `TestPatchPreferences_PinnedPlans_FR004_FR005`.
- [ ] T006 Regenerate `frontend/src/api/schema.d.ts` (`npm run api:types`); `testUser()` and the other
      typed preferences literals gain `pinnedPlanIds: []`.
- [ ] T007 `PinButton` in `frontend/src/features/plans/`: lucide `Star` icon button, filled
      `--brand-yellow` when pinned, outline ink when not; `aria-label` + tooltip
      "Pin plan '&lt;name&gt;'" / "Unpin plan '&lt;name&gt;'" (FR-003).
- [ ] T008 Star on every `PlanCard` (next to ⋯) and a first `Pinned` column in `PlanTable` (FR-001);
      existing component tests extended for the new column and props.
- [ ] T009 `PlansPage`: pinned plans first in both layouts, each group keeping the current sort
      (FR-002); `togglePin` PATCHes the full new `pinnedPlanIds` through the session's
      `updatePreferences` (cache + background save), cap message at 100 (FR-006).
- [ ] T010 Web tests in `PlansPage.test.tsx` (pin reorders to top, unpin returns, PATCH body, names,
      table order, deleted pinned plan doesn't show — FR-001, FR-002, FR-004, FR-007) and the star
      tests in `PlanCard.test.tsx` / `PlanTable.test.tsx` (FR-003).
- [ ] T011 All checks green (gofmt, vet, go test ./...; typecheck, lint, prettier, vitest,
      api:check); feature map row 018 and decision **D-317** recorded.
