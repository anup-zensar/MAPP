# Implementation Plan: Pinned Plans

**Branch**: `018-pinned-plans` | **Date**: 2026-10-21 | **Spec**: [spec.md](spec.md)

## Summary

One new display preference, `pinnedPlanIds`, carried end to end exactly like the theme (D-038,
D-317): stored with the person's preferences, changed through `PATCH /api/me/preferences`, shown
at once and saved in the background by the session's `updatePreferences`. The Plans list sorts
pinned plans first in both layouts and puts a named star on every card and row.

## Touch points

**Backend — `backend/internal/people`**

- `model.go`: `Preferences.PinnedPlanIDs []string` (stored, `pinnedPlanIds`, omitted when empty so
  old records stay untouched), `PreferencesView.PinnedPlanIDs` (always sent, never null),
  `PreferencesUpdate.PinnedPlanIDs *[]string`; `defaultPreferences`/`withDefaults` keep it never
  nil; an `equal` method because a struct with a slice no longer compares with `==`.
- `service.go` `UpdatePreferences`: when sent, dedupe with `uniqueIDs` (order kept), then validate:
  every id `^P-\d+$`, at most 100 (`maxPinnedPlans`); unknown ids allowed. Not logged (D-038).
- `messages.go`: the two refusal texts on field `pinnedPlanIds`.

**Contract — `backend/api/openapi.yaml`**

- `Preferences` gains `pinnedPlanIds` (array of string, **required**: always sent, insertion
  order); `PreferencesUpdate` gains it optional (`maxItems: 100`, items `^P-\d+$`). Regenerate
  `frontend/src/api/schema.d.ts` with `npm run api:types`; every test that asserts an exact
  `/api/me` or preferences body gains `"pinnedPlanIds":[]` (people and auth handler tests,
  `testUser()`).

**Frontend — `frontend/src/features/plans`**

- `PlansPage`: `pinnedFirst(items, pinnedIds)` before rendering cards and table (stable within
  each group); `togglePin` sends the full new list with the session's `updatePreferences` (query
  cache first, PATCH in the background — the ThemeToggle pattern) and refuses the 101st pin with a
  toast.
- New `PinButton` (lucide `Star`, tertiary sm icon button): outline ink, filled `--brand-yellow`
  while pinned; `aria-label` and tooltip "Pin plan '&lt;name&gt;'" / "Unpin plan '&lt;name&gt;'".
  Used by `PlanCard` (next to the ⋯ menu) and by a new first `Pinned` column in `PlanTable`.

## Tests

- Go service (`service_test.go`): `TestUpdatePreferences_PinnedPlans_RoundTrip_FR004` (default
  empty never nil, insertion order, unknown id kept, per person, unpin all, not logged),
  `…_DuplicatesDropped_FR005`, `…_InvalidIDRefused_FR005`, `…_CapAt100_FR006` (101 refused, 100
  saved, duplicates deduped before the cap). Store round trip extended in `store_json_test.go`.
- Go handler (`handler_test.go`): `TestPatchPreferences_PinnedPlans_FR004_FR005` (exact body,
  dedupe, refusal); every existing exact `/api/me`/preferences body extended.
- Web (`PlansPage.test.tsx`): pin reorders to the top and PATCHes the full list; unpin returns the
  plan and PATCHes; pinned-first in the table, sort kept within groups; a deleted pinned plan
  simply doesn't show; tooltip says the button's words. `PlanCard.test.tsx` / `PlanTable.test.tsx`:
  the star's name and filled state.
