# Implementation Plan: Test Hygiene

**Branch**: `feature/027-test-hygiene` | **Date**: 2026-10-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/027-test-hygiene/spec.md`

## Summary

Review every test against Principle VI, F10 and GO_STANDARDS §12; remove or merge what proves
nothing, recording each removal's rule (research R6), and make the suites fast and deterministic.
Measurements first (research R1–R4): the backend's slowest cost was the contract validator of
`internal/platform/apitest`, which compiled every schema of `openapi.yaml` up front in each of 18
test binaries (13 s idle, 20–30 s in a busy run, under the race detector); then seeding the demo
data for every test server (0.19 s each). Phase 1 fixes both in `apitest`, runs the phase-1
backend packages' tests in parallel, removes their restating tests, reviews all 361 web-app test
files and the end-to-end tests, and replaces the three fixed waits. Phase 2, after feature 028,
applies the same review to the business packages, guided by the timing report (R4, R7).

## Technical Context

**Language/Version**: Go (pinned in `go.mod`), TypeScript 5.9 strict, React 19

**Primary Dependencies**: unchanged; `libopenapi-validator` 0.16.1 is configured differently
(schema cache filled on first use, synchronous request checks behind a mutex)

**Storage**: none changed; the JSON test stand-in is 028's and is not touched

**Testing**: Go `testing` with `-race`; Vitest 5 + Testing Library + MSW; Playwright

**Target Platform**: developer laptops and the GitHub Actions runners (4 vCPU)

**Project Type**: web application; this feature changes tests, `internal/platform/apitest` and
the 027 documents

**Performance Goals**: the sum of the backend package times at least 25% lower (SC-002); no fixed
waits (SC-003)

**Constraints**: phase 1 stays out of the packages feature 028 changes (FR-008); no dependency
added (FR-010); files ≤ 300 non-blank lines; new lines within the comment budget

**Scale/Scope**: 244 Go test files (phase 1: 50 in 14 packages), 361 web-app test files (46,731
lines), 19 end-to-end files (40 tests)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle / standard | Check | Result |
|---|---|---|
| I Spec-first | spec with stories, FR-001–FR-011, branch `feature/027-test-hygiene` | Pass |
| III Documented design | plan and research; no data model, contract or flow change | Pass |
| V Simplicity | no new abstraction beyond two small types in `apitest`; split files stay ≤ 300 lines | Pass |
| VI Meaningful tests | the feature applies VI; required tests kept, each removal recorded with its rule (R6) | Pass |
| VII Security | no dependency, no runtime change | Pass |
| VIII Clean construction | new lines within the budget; 027 exempt from the whole-file rule (4.2.0) | Pass |
| B2 Package state | `apitest` keeps one write-once demo-data cache per test binary; see Complexity Tracking | Justified |
| B9 Go testing | `t.Parallel` in phase-1 packages; no `time.Sleep` | Pass |
| F10 Frontend testing | no fixed waits; MSW and fake timers as before | Pass |
| Gates 1–14 | run for the changed packages and the whole web app before the pull request | Pass |

**Post-design re-check**: unchanged — Pass with the one justified entry below.

### Threat model (STRIDE)

No runtime code changes; `apitest` is compiled only into test binaries. No new trust boundary.

## Project Structure

### Documentation (this feature)

```text
specs/027-test-hygiene/
├── spec.md
├── plan.md
├── research.md
├── tasks.md
└── checklists/requirements.md
```

### Source Code (repository root)

```text
backend/internal/platform/apitest/   apitest.go, contract.go (lazy schemas, serial checks), demo_data.go
backend/internal/{auth,samples,app}/  tests: parallel, duplicates removed, files split ≤ 300 lines
backend/internal/platform/{errs,actor,record,ids,clock}/  tests: restating tests removed, parallel
frontend/src/**/*.test.ts(x)          duplicates and restating tests removed; fixed waits replaced
```

**Structure Decision**: no new folders; split test files keep the package's naming
(`service_test.go`, `session_test.go`, `fixture_test.go`, …).

## Complexity Tracking

| Violation | Why needed | Simpler alternative rejected because |
|---|---|---|
| Package-level demo-data cache in `apitest/demo_data.go` (`//nolint:gochecknoglobals`; B2 forbids package-level mutable state) | A read-only fixture shared by the tests of one binary must live at package level; `_test.go` files may hold such fixtures (the lint configuration exempts them), but `apitest` is imported by other packages' tests, so it cannot be a `_test.go` file. Written once under `sync.Once`, never changed. Saves 0.19 s per test server (R2). | Seeding every server costs 7 s per heavy package (R2); a cache on disk across processes would need invalidation and clean-up |
| `//nolint:revive` on two methods of `serialValidator` | They implement libopenapi-validator's interface, whose names (`ValidateHttpRequest`) break the initialism rule | The interface is the library's |
