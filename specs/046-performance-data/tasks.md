# Tasks: Performance Data from a Simulated Feed

**Input**: Design documents from `specs/046-performance-data/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md),
[data-model.md](./data-model.md), [contracts/api.md](./contracts/api.md)

**Tests**: required, not optional. Constitution VI is non-negotiable and the build guide §8 says tests first.
Every test name mentions the requirement ID it proves.

**Organization**: one phase per user story, in the spec's priority order. Implement **one story per prompt**
(build guide §8): "implement US1 of 046 only, tests first, then stop".

## Format: `[ID] [P?] [Story] Description`

- **[P]**: parallelisable — different file, no dependency on an unfinished task
- **[Story]**: US1–US6, on user-story tasks only

---

## Phase 1: Setup

**Purpose**: the package skeleton and its reserved numbers, before any rule exists.

- [X] T001 Create `backend/internal/performance/` with `model.go`, `store.go`, `service.go`, `handler.go`, `ports.go` as empty-but-compiling files following the area pattern of `backend/internal/bulk/`
- [X] T002 [P] Create `backend/internal/performance/performancetest/store.go` as the area's fake, in the shape of `backend/internal/bulk/bulktest/`
- [ ] T003 [P] Create `backend/internal/performance/simulatedfeed/` with its port defined in the parent's `ports.go` (research R1)
- [ ] T004 [P] Create `frontend/src/features/performance/` with `api.ts`, `schemas.ts` and an `index.ts` exporting nothing yet

**Checkpoint**: the package compiles and is importable; no behaviour yet.

---

## Phase 2: Foundational (blocking)

**⚠️ CRITICAL**: no user story may begin until this phase is complete.

- [X] T005 Write migration `backend/internal/platform/database/migrations/00030_performance_sources.sql` for `performance_sources` and `performance_source_mappings`, with the partial unique index enforcing "at most one **active** source per (model, stage) and per (model, revenue)" (FR-003)
- [X] T006 Write migration `.../00031_performance_facts.sql` for `performance_imports`, `performance_facts` and `performance_review_rows`, with the CHECK `rows_read = matched + held + unattributed + rejected + unmatched` (FR-022), the unique index on (`source_id`, `source_reference`) among rows not yet replaced (research R3), the index on `held` (research R4), and the CHECK requiring `comment` when `status = 'dismissed'` (FR-024)
- [X] T007 Write migration `.../00032_media_codes.sql` for `media_codes` (`code` bigint unique, never reused), `activity_external_names` (unique across names **and** aliases, one `is_current` per activity) and `activity_integration_ids` (unique together on `activity_id`, `source_id`)
- [X] T008 Write `backend/internal/platform/database/queries/performance.sql` with the six queries named in plan.md *Queries*, each carrying its scope filter; run `sqlc generate && sqlc vet`
- [X] T009 [P] Write `backend/internal/performance/model.go`: the eight entities of [data-model.md](./data-model.md), each embedding `record.Meta`, with money as integer minor units plus currency (FR-014), never a float
- [X] T010 [P] Write `backend/internal/performance/store.go` declaring the small `Store` the service calls and its `Transactor`, then `store_postgres.go` implementing it
- [X] T011 Register the `history.Kind` values for data source, fact, import, review row and media code so every change writes one activity-log entry inside the same `Write` transaction
- [ ] T012 [P] Write `backend/internal/app/performance.go` wiring the service, stores and handler — **deferred to US1**: `app.go` holds the call and no lane edits it, so the function would be unused and fail the linter (integrator item 4). The store is already registered in `stores.Set`
- [X] T013 [P] Write `backend/internal/performance/performancetest/store.go` fully, and add the new `Store` methods to the in-memory set in `backend/internal/stores/`
- [X] T014 Write the scope-isolation test file `backend/internal/performance/store_postgres_scope_test.go` covering every scoped read in T008 on a fresh PostgreSQL database per test (FR-044): a record outside scope answers as missing, never as a refusal
- [X] T015 Write `backend/internal/performance/authz_test.go`: every operation refuses without its permission and **fails closed** — `performance.import` for loading, configuration and review actions, `performance.link` for external names, integration IDs and media codes (FR-043). A record outside the actor's scope answers as **missing**, while `forbidden` refuses only an action on a record the actor can see (constitution non-negotiable 7)
- [ ] T016 Write `backend/internal/performance/records_test.go`: one activity-log entry per change on each of the eight record types, a stale update refused with `version_conflict`, and a delete routed to the recycle bin through `history.MoveToBin` (FR-045, CLAUDE.md "Every new record type"). **Partly done** — the activity log and the recycle-bin delete are covered for the data source, and the four activity children are registered as bin kinds. Still missing: the log on the other record types, which arrive with the stories that create them, and the `version_conflict` check

**Checkpoint**: schema, stores and fakes exist; the six stories can now proceed in parallel.

---

## Phase 3: User Story 1 — Data source and mappings (P1) 🎯 MVP

**Goal**: configure where a feed comes from and what its columns mean, with a downloadable template.

**Independent Test**: create a source with mappings, download its template, edit a mapping, delete the source
while it holds no facts. No upload needed.

### Tests (write first, confirm they fail)

- [ ] T017 [P] [US1] Write `backend/internal/performance/source_test.go`: create, find, view, edit (FR-001); one active source per (model, stage) refused naming the conflicting source (FR-003); delete refused while facts exist and Deactivate offered (FR-002); deactivate keeps facts, refuses uploads and updates *Is integrated* at once (FR-004); reactivate · proves SC-005
- [ ] T018 [P] [US1] Write `backend/internal/performance/freshness_test.go`: *Current · Stale · Failed · Inactive* derived from the last import, data-through date and cadence (FR-005), on a fixed clock
- [ ] T019 [P] [US1] Write `backend/internal/performance/mapping_test.go`: each `meaning` value accepted; `web_metric` restricted to **sessions, visits, engagement touches** (046-C11); a mapping change affects future loads only (FR-007) (FR-006)
- [ ] T020 [P] [US1] Write `backend/internal/performance/handler_source_test.go` as a contract test (external `package performance_test`, through `internal/platform/apitest`) for every source and mapping operation in [contracts/api.md](./contracts/api.md), including the **409** on delete-with-facts and on the duplicate stage supplier
- [ ] T021 [P] [US1] Write `frontend/src/features/performance/sources/SourceList.test.tsx` and `SourceForm.test.tsx`: the states data, empty, no access, and all validation errors shown at once

### Implementation

- [ ] T022 [US1] Implement the data-source rules in `backend/internal/performance/source.go`: create, edit with version check (`version_conflict` on a stale update), delete-vs-deactivate, reactivate
- [ ] T023 [US1] Implement derived freshness in `backend/internal/performance/freshness.go` — computed, never stored (data-model.md)
- [ ] T024 [US1] Implement mapping rules in `backend/internal/performance/mapping.go`, including the one-active-source-per-stage refusal naming the conflicting source (FR-006, FR-003)
- [ ] T025 [US1] Implement the blank file template in `backend/internal/performance/template.go`, carrying exactly the columns the mappings define (FR-008)
- [ ] T026 [US1] Implement the source and mapping handlers in `backend/internal/performance/handler.go`; add each operation's rule to the access route table and report it in the pull request (the shared `routes_measure.go` is not this lane's to edit)
- [ ] T027 [US1] Remove this story's operations from `notYetServed` in `backend/internal/app/app_test.go` (this feature's group only)
- [ ] T028 [P] [US1] Build `frontend/src/features/performance/sources/` — list, form and the Settings › Performance data route — with React Hook Form + Zod and MSW-typed answers; run `pnpm run api:types` first
- [ ] T029 [US1] Run the checks of [quality-gates.md](../../docs/sdd/quality-gates.md) on everything changed

**Checkpoint**: sources and mappings are configurable end to end.

---

## Phase 4: User Story 2 — Upload and match (P1)

> **On size**: this phase exceeds the build guide's 10–20 tasks per story. It is kept as one story because spec.md defines exactly six, and splitting it would invent a seventh that no requirement, acceptance scenario or issue refers to. It carries validation, the personal-data gate, set-based matching, holding, reconciliation, idempotency, undo, money rules and the adapter — the feature's whole spine.

**Goal**: load a file, validate it, match every row, and reconcile the five outcomes.

**Independent Test**: upload a file exercising all five outcomes; confirm the counts add up; re-upload it
unchanged; undo it.

### Tests (write first, confirm they fail)

- [ ] T030 [P] [US2] Write `backend/internal/performance/validate_test.go`: all faults in one row reported together — empty required column, invalid or future date, negative or fractional count, invalid amount or unconfigured currency, unknown stage, missing or repeated reference (FR-011)
- [ ] T031 [P] [US2] Write `backend/internal/performance/personaldata_test.go`: a file with a personal-data column is refused **in full** and **no row is stored** (FR-012) · proves SC-007
- [ ] T032 [P] [US2] Write `backend/internal/performance/matcher_test.go` table-driven over the four keys in their strict order — activity ID → media code → integration ID → external name (FR-018) — plus ambiguous, unknown, and **keys disagree** naming both candidates and never guessing (FR-019) · proves SC-006
- [ ] T033 [P] [US2] Write `backend/internal/performance/held_test.go`: a fact on an activity that is Created, Submitted or Declined is held and counts nowhere; Approved, In progress and Complete do not hold (FR-020, FR-028) · proves SC-002
- [ ] T034 [P] [US2] Write `backend/internal/performance/binned_test.go`: a fact matched to a **binned** activity is held until restore **and** approval; a fact matched to a **permanently deleted** activity goes to the review queue (FR-021)
- [ ] T035 [P] [US2] Write `backend/internal/performance/summary_test.go` as a property test: for any generated file, matched + held + unattributed + rejected + unmatched **equals** rows read (FR-022) · proves SC-001
- [ ] T036 [P] [US2] Write `backend/internal/performance/idempotent_test.go`: an identical file changes nothing; a repeated (source, source reference) replaces and audits both versions (FR-013) · proves SC-004
- [ ] T037 [P] [US2] Write `backend/internal/performance/undo_test.go`: undo restores replaced versions, leaves a later upload's facts alone, and leaves the import listed as *Undone* (FR-017)
- [ ] T038 [P] [US2] Write `backend/internal/performance/money_test.go`: revenue stored exactly in its currency; without a rate it is marked and excluded with its count (FR-014); a fact counts in the fiscal year of its own date across a multi-year activity (FR-015) · proves SC-010
- [ ] T039 [P] [US2] Write `backend/internal/performance/handler_import_test.go` as a contract test for the upload, simulated-pull, list, view and undo operations, including the **422** whole-file refusal and FR-016's "an import MUST NOT be editable, and the reason MUST be stated"
- [ ] T040 [P] [US2] Write `backend/internal/performance/simulatedfeed/feed_test.go`: rows from the adapter take the identical validation and matching path as an uploaded file (046-C13, ADR-0020), on a fixed seed (FR-010)
- [ ] T041 [P] [US2] Write `frontend/src/features/performance/imports/ImportPage.test.tsx`: the states data, empty, no access, upload in progress, refused file, and the summary showing all five counts

### Implementation

- [ ] T042 [US2] Implement row validation in `backend/internal/performance/validate.go`, collecting **every** fault per row rather than returning on the first
- [ ] T043 [US2] Implement the whole-file personal-data gate in `backend/internal/performance/personaldata.go`, run before any row is read
- [ ] T044 [US2] Implement **set-based** matching in `backend/internal/performance/matcher.go`: collect distinct keys by type, resolve each type in one query, build the key → activity map, then decide rows as pure functions over it. A per-row lookup is the failure this rule exists to prevent (research R2, plan R16)
- [ ] T045 [US2] Implement holding and release in `backend/internal/performance/held.go`, with the release port the approvals area calls (research R4) declared in `ports.go`
- [ ] T046 [US2] Implement the binned and permanently-deleted paths in `backend/internal/performance/held.go`, reusing the holding rule rather than branching it (FR-021)
- [ ] T047 [US2] Implement the import run and its five-count summary in `backend/internal/performance/importer.go`, computing the counts in SQL (FR-009)
- [ ] T048 [US2] Implement undo in `backend/internal/performance/undo.go`, walking `replaced_by_id` back
- [ ] T049 [US2] Implement the `simulatedfeed` adapter behind its port, labelling everything it produces as simulated (ADR-0020)
- [ ] T050 [US2] Implement the import handlers; add each operation's route rule and report it in the pull request
- [ ] T051 [US2] Remove this story's operations from `notYetServed` in `backend/internal/app/app_test.go`
- [ ] T052 [P] [US2] Build `frontend/src/features/performance/imports/` — upload, simulated pull, import list and the summary with its five counts and the undo confirmation stating them (FR-017)
- [ ] T053 [US2] Record the benchmark in [plan.md](./plan.md): a 50,000-row file against 5,000 in-scope activities, confirming a **constant** query count (plan R16) · proves SC-009
- [ ] T054 [US2] Run the quality-gate checks on everything changed

**Checkpoint**: data loads, matches and reconciles; undo works.

---

## Phase 5: User Story 3 — Performance on an approved activity (P1)

**Goal**: show the activity's figures, gated until it is approved.

**Independent Test**: open a Submitted activity (gate message, no figures), approve it, confirm its held facts
appear with goal, actual and conversion per stage.

### Tests (write first, confirm they fail)

- [ ] T055 [P] [US3] Write `backend/internal/performance/activityview_test.go`: a non-approved activity returns the stated reason and **no figures** (FR-027); an approved one returns per-stage goal, actual, projected and conversion, and revenue naming its model (FR-029)
- [ ] T056 [P] [US3] Write `backend/internal/performance/facts_list_test.go`: the fact list carries date, source, stage or metric, value, currency, **the key matched on** and the import (FR-030), searchable and exportable
- [ ] T057 [P] [US3] Write `backend/internal/performance/handler_activity_test.go` as a contract test for the four activity performance operations
- [ ] T058 [P] [US3] Write `frontend/src/features/performance/activity/PerformancePanel.test.tsx` with **one test per state** — not approved, no source, no goal, no data yet, stale, failed (FR-032) — asserting no sample figure appears in any of them · proves SC-008
- [ ] T059 [P] [US3] Write `frontend/e2e/performance-data.spec.ts` covering US3's journey with axe: Submitted activity shows the gate, approval releases its facts

### Implementation

- [ ] T060 [US3] Implement the activity performance read in `backend/internal/performance/activityview.go`, one query for the panel rather than one per stage
- [ ] T061 [US3] Implement the facts list, search and export in `backend/internal/performance/facts.go`
- [ ] T062 [US3] Implement the six empty-state reasons as explicit values the API returns, so the web app never infers them (constitution IV)
- [ ] T063 [US3] Implement the activity performance handlers; add their route rules and report them
- [ ] T064 [US3] Remove this story's operations from `notYetServed`
- [ ] T065 [P] [US3] Build `frontend/src/features/performance/activity/PerformancePanel.tsx` and mount it in the Performance slot of `frontend/src/features/activities/ActivityPage.tsx` — **the one file outside this feature's folders it owns** (build guide §4.1)
- [ ] T066 [US3] Run the quality-gate checks on everything changed

**Checkpoint**: the P1 journey is visible to activity owners.

---

## Phase 6: User Story 4 — External names and integration IDs (P1)

**Goal**: create the two matching keys that let outside systems address an activity.

**Independent Test**: approve an activity and confirm its generated external name; rename by hand; confirm the
previous name still matches as an alias.

### Tests (write first, confirm they fail)

- [ ] T067 [P] [US4] Write `backend/internal/performance/externalname_test.go`: a row carrying an activity's **current** external name matches it and records `externalName` as the key used (FR-033)
- [ ] T068 [P] [US4] Write `backend/internal/performance/alias_test.go`: a rename keeps the superseded name as an alias, and a row carrying that **old** name still matches the activity (FR-034)
- [ ] T069 [P] [US4] Write `backend/internal/performance/alias_uniqueness_test.go`: an alias repeating another activity's external name or alias is **refused with the reason**, so a key never resolves to two activities (FR-035); and this feature never writes `activities.external_name`, which feature 006 owns (FR-036, 046-C14)
- [ ] T070 [P] [US4] Write `backend/internal/performance/integrationid_test.go`: at most one per source, unique within the source, removal confirmed with its effect on matching, every change audited (FR-037)
- [ ] T071 [P] [US4] Write `backend/internal/performance/copy_test.go`: a copied activity carries **no** external name, alias, integration ID, media code or fact (FR-038)
- [ ] T072 [P] [US4] Write `backend/internal/performance/handler_keys_test.go` as a contract test for the five external-name and integration-ID operations, including both **409** cases
- [ ] T073 [P] [US4] Write `frontend/src/features/performance/activity/DataSourcesArea.test.tsx`: external name and aliases, media-code count, both flags, per-source integration ID, fact count and last fact date, plus the no-source notice (FR-031)

### Implementation

- [ ] T074 [US4] Implement alias capture and resolution in `backend/internal/performance/externalname.go`, reading `activities.external_name` and never writing it (046-C14)
- [ ] T075 [US4] Implement the alias uniqueness check across current external names and existing aliases
- [ ] T076 [US4] Implement integration IDs in `backend/internal/performance/integrationid.go`
- [ ] T077 [US4] Extend the matcher to resolve integration ID and external name **including aliases**, keeping the strict order of T044
- [ ] T078 [US4] Implement the handlers; add their route rules and report them. Report **two** changes outside this feature's folders: the activity-copy exclusion (FR-038), and the request to feature 006's lane for generation on approval (FR-036, 046-C14)
- [ ] T079 [US4] Remove this story's operations from `notYetServed`
- [ ] T080 [P] [US4] Build the Data sources area in `frontend/src/features/performance/activity/`
- [ ] T081 [US4] Run the quality-gate checks on everything changed

**Checkpoint**: all four matching keys exist; matching is complete.

---

## Phase 7: User Story 5 — Review and fix rows (P2)

**Goal**: turn silent data loss into visible, fixable work.

**Independent Test**: upload a file with an unknown key and a rejected row; find both; correct one; dismiss the
other with a comment and reopen it.

### Tests (write first, confirm they fail)

- [ ] T082 [P] [US5] Write `backend/internal/performance/review_test.go`: every rejected and unmatched row appears with its reason; **no row can be created by hand** (FR-023); find by key, source, reason, upload, date and status · proves SC-003
- [ ] T083 [P] [US5] Write `backend/internal/performance/correct_test.go`: a corrected row re-checks and becomes *Resolved* on a match; dismissal **requires** a comment; reopening restores it, with both actions in the history (FR-024)
- [ ] T084 [P] [US5] Write `backend/internal/performance/recheck_test.go`: open rows and held facts re-check on demand and automatically at each load from the same source (FR-025), in a **constant** query count regardless of queue size
- [ ] T085 [P] [US5] Write `backend/internal/performance/precedence_test.go`: a later load with the same source and reference replaces a corrected row — **the source wins** (FR-026)
- [ ] T086 [P] [US5] Write `backend/internal/performance/handler_review_test.go` as a contract test for the six review operations, including the **422** on dismissal without a comment
- [ ] T087 [P] [US5] Write `frontend/src/features/performance/review/ReviewQueue.test.tsx`: the states data, empty, no access, filtered to nothing, and the required-comment dialog

### Implementation

- [ ] T088 [US5] Implement the review queue reads and filters in `backend/internal/performance/review.go`
- [ ] T089 [US5] Implement correct, dismiss (comment required) and reopen, each writing its history entry
- [ ] T090 [US5] Implement on-demand and automatic re-check in `backend/internal/performance/recheck.go`, reusing the set-based matcher of T044 rather than looping rows
- [ ] T091 [US5] Implement source-wins precedence over a correction
- [ ] T092 [US5] Implement the review handlers; add their route rules and report them
- [ ] T093 [US5] Remove this story's operations from `notYetServed`
- [ ] T094 [P] [US5] Build `frontend/src/features/performance/review/` — the queue, its filters, the correction form and the dismissal dialog
- [ ] T095 [US5] Run the quality-gate checks on everything changed

**Checkpoint**: nothing is lost silently.

---

## Phase 8: User Story 6 — Media codes (P2)

**Goal**: unique campaign-link codes that make web facts attributable.

**Independent Test**: on an activity whose type requires codes, generate one, copy its tagged link, then try
to delete it after a fact has used it.

### Tests (write first, confirm they fail)

- [ ] T096 [P] [US6] Write `backend/internal/performance/mediacode_test.go`: codes are system-generated, unique and **never reused**, from a configurable starting number (FR-042); generation succeeds at **any** activity status
- [ ] T097 [P] [US6] Write `backend/internal/performance/mediacode_rules_test.go`: the area appears only where the activity type requires codes and a code never attaches to a line item or any other level (FR-040); every missing required attribute is reported at once (FR-041); delete allowed only while unused, otherwise deactivate
- [ ] T098 [P] [US6] Write `backend/internal/performance/configuration_test.go`: the activity types requiring codes, the attribute catalogue and the start number live in **this feature's** Settings › Performance data configuration, not on the activity type record (046-C12) (FR-039)
- [ ] T099 [P] [US6] Write `backend/internal/performance/handler_mediacode_test.go` as a contract test for the five media-code operations and the two configuration operations, including the **409** on delete-when-used
- [ ] T100 [P] [US6] Write `frontend/src/features/performance/mediacodes/MediaCodes.test.tsx`: hidden for a type that does not require codes; all missing attributes shown at once; copy code and copy tagged link

### Implementation

- [ ] T101 [US6] Implement the code generator in `backend/internal/performance/mediacode.go`, never reusing a number
- [ ] T102 [US6] Implement the configuration in `backend/internal/performance/configuration.go`
- [ ] T103 [US6] Implement generate, edit (attributes and landing page only), delete-while-unused and deactivate
- [ ] T104 [US6] Extend the matcher to resolve media codes, keeping the strict order of T044
- [ ] T105 [US6] Implement the media-code and configuration handlers; add their route rules and report them
- [ ] T106 [US6] Remove this story's operations from `notYetServed`
- [ ] T107 [P] [US6] Build `frontend/src/features/performance/mediacodes/` and the configuration screen in Settings › Performance data
- [ ] T108 [US6] Run the quality-gate checks on everything changed

**Checkpoint**: every story is independently functional.

---

## Phase 9: Polish and cross-cutting

- [ ] T109 Extend `frontend/e2e/performance-data.spec.ts` to cover **every P1 story** (US1, US2, US3, US4) with axe included
- [ ] T110 Seed the wave B demo facts in `backend/seed/measure.go` — **blocked**: that file is lane D's to create and 039 wires its call into `seed.go`. Raise in the pull request if it still does not exist (plan *Risks*)
- [ ] T111 [P] Confirm every file changed is within 300 non-blank lines and every file's comments are within 5% of its non-blank lines (constitution V, VIII)
- [ ] T112 [P] Confirm every test name carries the requirement ID it proves, and that no test restates the code, tests generated code or exists only for coverage (constitution VI)
- [ ] T113 Check every FR of [spec.md](./spec.md) against plan.md: each has a passing test named with it; tick the tasks above
- [ ] T114 Run `/speckit-analyze` and fix every CRITICAL finding **before** this feature is considered done
- [ ] T115 Run `/speckit-converge`; append anything remaining to this file or record "clean"
- [ ] T116 Open the pull request into `measure-module` with: the feature-map status change, the three integrator items from [contracts/api.md](./contracts/api.md), the ADR-0019 → ADR-0020 renumber, and the `seed/measure.go` dependency

---

## Dependencies and execution order

### Phase dependencies

- **Setup (1)** → **Foundational (2)** blocks everything else.
- **US1, US2, US3, US5, US6** can then proceed in parallel if staffed. This lane is one developer, so the
  order below is the sequential one.
- **Polish (9)** last.

### Story dependencies — the one that is not independent

- **US1** is independent and is the MVP.
- **US2** needs US1's sources and mappings. It can match on **activity ID alone** and still be fully testable;
  the other three keys arrive with US4 (T077) and US6 (T104), which extend the matcher without changing its
  order rule.
- **US3** needs US2's facts to show anything, but its gate and its six empty states are testable without them.
- **US4** and **US6** extend US2's matcher; neither blocks the other.
- **US5** needs US2's rejected and unmatched rows.

### Suggested sequence for one developer

`US1 → US2 → US3 → US4 → US5 → US6`, which is the spec's priority order and delivers all four P1 stories
before either P2 story.

### Parallel opportunities

- T002, T003, T004 together in Setup.
- T009, T010, T012, T013 together in Foundational, after the migrations.
- Within every story, all test tasks marked [P] are different files and can be written together.
- Every story's frontend task ([P]) can run beside its backend handler work from the fixed contract, as the
  build guide §8 allows ("backend and web of a story may run as two prompts in two worktrees").

## Implementation strategy

**MVP** is Phase 1 + Phase 2 + **US1**: Shell can configure its feeds and review them before a single row is
loaded. Stop there and validate before starting US2.

Then deliver incrementally, one story per prompt, running the quality-gate checks after each. A story is done
when its tests, its end-to-end test (P1) and the check script are green and the spec's scenarios pass on the
demo data.

## Notes

- `scripts/check-changed.ps1` does not exist yet — it belongs to lane A's open kick-off issue #31. Until it
  does, run the checks of [quality-gates.md](../../docs/sdd/quality-gates.md) by hand.
- Run `sqlc generate` after every rebase; run `pnpm run api:types` after the integrator applies the contract.
- Touch only the paths issue #39 lists under **Owns**. Report everything else in the pull request.
