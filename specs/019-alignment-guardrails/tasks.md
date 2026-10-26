---

description: "Tasks for 019 Alignment Guardrails"
---

# Tasks: Alignment Guardrails

**Input**: Design documents from `specs/019-alignment-guardrails/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md),
[data-model.md](data-model.md), [contracts/](contracts/), [quickstart.md](quickstart.md)

**Tests**: the constitution (Principle VI) requires tests for every rule. The `sourcecheck` rules
(counting, limits, baseline, exceptions) get unit tests on real files; the pipeline, the npm
settings and the branch rules are validated by the quickstart scenarios, not by tests of YAML.

**Organization**: tasks are grouped by user story; each story can be shipped and checked on its own.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: the user story the task belongs to (US1–US4)

---

## Phase 1: Setup (make today's code pass the gates)

**Purpose**: the smallest behaviour-preserving fixes so every gate is green on today's code
(FR-016, research R1, R3).

- [X] T001 Pin the toolchain and fix the vulnerable dependency in `backend/go.mod` and `backend/go.sum`: add `toolchain go1.27.3`, run `go get golang.org/x/net@v0.60.0` and `go mod tidy`; then `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...` must exit 0 (R3)
- [X] T002 [P] Fix the one `gofmt -s` finding (redundant `PersonView` type in the composite literal) in `backend/internal/approvals/rounds_test.go:37`; `gofmt -s -l .` must print nothing (R1)
- [X] T003 [P] Make today's tests pass without changing behaviour (R1): race-detector build tag for the 2 s timing of `TestQueryLog_50000Entries_SC006` in `backend/internal/history/query_test.go`, `race_on_test.go`, `race_off_test.go`; canned answers in `frontend/src/features/campaigns/CampaignPanel.test.tsx` and `frontend/src/features/line-items/LineItemPage.test.tsx`; `reducedMotion: 'reduce'` in `frontend/playwright.config.ts`; web-first checkbox step in `frontend/e2e/activity-log.spec.ts`

**Checkpoint**: `go test -race ./...`, `npm test`, `npm run test:e2e` (36/36) and `govulncheck` pass.

---

## Phase 2: Foundational (npm settings and scripts the pipeline calls)

**Purpose**: the frontend settings and scripts that constitution F1 and VII name and the pipeline
runs (FR-010, FR-020; R9, R10, R15).

- [X] T004 Release age with npm's built-in setting (FR-020, R15): create `frontend/.npmrc` with `min-release-age=1` and `engine-strict=true`, and add `"npm": ">=11.10.0"` to `engines` in `frontend/package.json` (the first npm with `min-release-age`); `npm ci` still installs
- [X] T005 Add the dev dependency `@vitest/coverage-v8` at exactly `5.0.5` (published 2026-10-15; matches `vitest` 5.0.5) to `frontend/package.json` and `frontend/package-lock.json` with `npm install --save-dev --save-exact @vitest/coverage-v8@5.0.5` (after T004, so the age rule applies)
- [X] T006 Configure coverage in `frontend/vite.config.ts` (`test.coverage`: provider `v8`, reporters `text-summary` and `lcov`, `reportsDirectory: 'coverage'`, include `src/**/*.{ts,tsx}`, exclude `src/**/*.test.{ts,tsx}`, `src/test/**`, `src/api/schema.d.ts`) and add the script `"test:coverage": "vitest run --coverage --reporter=default --reporter=junit --outputFile.junit=test-results/junit.xml"` to `frontend/package.json`; `npm run test:coverage` writes `frontend/coverage/lcov.info` and `frontend/test-results/junit.xml` (both already git-ignored)
- [X] T007 [P] In `frontend/package.json`: make `lint` `eslint . --max-warnings 0`, and add `browserslist`: `last 2 Chrome versions`, `last 2 ChromeAndroid versions`, `last 2 Edge versions`, `last 2 Firefox versions`, `last 2 FirefoxAndroid versions`, `last 2 Safari versions`, `last 2 iOS versions` (R10); `npm run check` still passes

**Checkpoint**: every frontend command in [contracts/ci-checks.md](contracts/ci-checks.md) runs locally.

---

## Phase 3: User Story 1 — Every pull request is checked before it can merge (P1) 🎯 MVP

**Goal**: four blocking checks with one named step per gate on pull requests and pushes to `main`,
`release`, `develop` (FR-001–FR-005, FR-017).

**Independent test**: [quickstart.md](quickstart.md) §3 — one deliberate problem per gate fails
the check holding it at the named step; a clean pull request is green, with artifacts attached.

- [X] T008 [US1] Create `.github/workflows/ci.yml`: `name: CI`; `on.pull_request.branches` and `on.push.branches` = `main`, `release`, `develop`; `permissions: contents: read`; `concurrency: { group: ci-${{ github.ref }}, cancel-in-progress: true }`; four jobs on `ubuntu-latest`, each with `timeout-minutes` and `name` equal to its id (R2)
- [X] T009 [US1] The `backend` job in `.github/workflows/ci.yml` (job permission `pull-requests: read` added): `actions/checkout@v7`, `actions/setup-go@v7` (`go-version-file: backend/go.mod`, `cache-dependency-path: backend/go.sum`), then named steps in `backend/`: `Format` (`gofmt -s -l .` and `go run golang.org/x/tools/cmd/goimports@v0.52.0 -local mapp -l .` must print nothing), `Vet` (`go vet ./...`), `Lint` (`golangci/golangci-lint-action@v9`, `version: v2.15.0`, `working-directory: backend`, `only-new-issues: true`), `Test` (`set -o pipefail; go test -race -covermode=atomic -coverprofile=coverage.out ./... | tee go-test.txt`), `Vulnerabilities` (`go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...`); upload `go-test.txt` and `coverage.out` as artifact `backend-test` with `if: always()` and `retention-days: 30`
- [X] T010 [US1] The `frontend` job in `.github/workflows/ci.yml`: `actions/checkout@v7`, `actions/setup-node@v7` (`node-version: 24`, `cache: npm`, `cache-dependency-path: frontend/package-lock.json`), `npm ci`, then named steps in `frontend/`: `Format` (`npm run format:check`), `Type check` (`npm run typecheck`), `Lint` (`npm run lint`), `API types` (`npm run api:check`), `Test` (`npm run test:coverage`), `Build` (`npm run build`), `Audit` (`npm audit --omit=dev --audit-level=high`); upload `frontend/coverage/lcov.info` and `frontend/test-results/junit.xml` as artifact `frontend-test`, `if: always()`, `retention-days: 30`
- [X] T011 [US1] The `e2e` job in `.github/workflows/ci.yml`: checkout, setup-go (as T009) and setup-node (as T010), `npm ci`, `npx playwright install --with-deps chromium`, `npm run test:e2e` in `frontend/`; upload `frontend/test-results/` as artifact `e2e` with `if: failure()` and `retention-days: 30`; `timeout-minutes: 30`
- [X] T012 [US1] Lint the workflow with `go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/ci.yml` (no findings) and confirm the job and step names match [contracts/ci-checks.md](contracts/ci-checks.md)

**Checkpoint**: the workflow is valid and named as the contract says.

---

## Phase 4: User Story 2 — New code stays small and sparsely commented (P1)

**Goal**: the size and comment checks of FR-006–FR-009, with the legacy baseline, in the pipeline
and on a laptop.

**Independent test**: [quickstart.md](quickstart.md) §2.

### Tests for User Story 2 (rules, Principle VI)

- [ ] T013 [P] [US2] Comment-counting tests in `backend/internal/sourcecheck/comments_test.go` (table-driven, files written to `t.TempDir()`): Go line, block (counted per line it spans) and trailing comments; `//` inside a string or raw string not counted; directives not counted (`//go:build`, `//go:embed`, `//nolint:gosec // reason`); TypeScript/TSX `//`, `/* */`, `{/* */}`, `//` inside `'…'`, `"…"`, `` `…` `` and a regular-expression literal not counted, `// eslint-disable-next-line …`, `// eslint-enable`, `// prettier-ignore`, `// @ts-expect-error …`, `/// <reference …>` not counted; CSS `/* */`; SQL `--`, with `-- +goose Up` and `-- name: GetX :one` not counted (FR-006)
- [ ] T014 [P] [US2] Limit, exception and generated-file tests in `backend/internal/sourcecheck/limits_test.go`: 300 non-blank lines pass, 301 fail without an exception, 301–400 pass with an exception whose `lines` ≥ the count and fail above it, 401 always fail; an exception for a file at 300 lines or fewer is reported as removable; blank lines not counted; comment budget = ceil(5% × non-blank lines) (100 lines → 5 passes, 6 fails; 20 lines → 1); files with a `// Code generated … DO NOT EDIT.` header, the openapi-typescript header, or under `frontend/src/api/schema.d.ts` or `backend/internal/platform/database/sqlcdb/` are skipped (FR-006, FR-007)
- [ ] T015 [P] [US2] Baseline tests in `backend/internal/sourcecheck/baseline_test.go`: a recorded file passes at its `count`, fails at `count + 1`, passes at `count − 1` and is listed as improvable; an entry whose file is gone (deleted or renamed) is listed as removable and does not fail, and the renamed file is judged as a new file; `-seed` writes every file over a limit and refuses when the baseline exists; `-tighten` lowers counts and drops entries now within the limit and never raises a count or adds an entry; JSON is sorted by `path` then `metric`, two-space indented, final newline (FR-008)

### Implementation for User Story 2

- [ ] T016 [US2] `backend/internal/sourcecheck/files.go`: list `git ls-files --cached --others --exclude-standard -z` under the root, keep `.go .ts .tsx .css .sql`, skip generated files (header within the first 5 lines, or configured path) (R6)
- [ ] T017 [P] [US2] `backend/internal/sourcecheck/comments.go`: comment lines of a Go file via `go/scanner` (mode `ScanComments`), each line counted once, directives excluded (R6)
- [ ] T018 [P] [US2] `backend/internal/sourcecheck/lexer.go`: comment lines of TypeScript, TSX, CSS and SQL files — skips `'…'`, `"…"`, `` `…` `` and regular-expression literals (a `/` after an operator, `(`, `,`, `=`, `:`, `[`, `!`, `&`, `|`, `?`, `{`, `}`, `;`, `return` or line start), knows `//`, `/* */`, SQL `--`; directives excluded (R6)
- [ ] T019 [US2] `backend/internal/sourcecheck/limits.go`: per file, non-blank lines and comment lines → problems: size > 400; 301–400 without an exception covering the count; comments > ceil(5% × non-blank lines) — the baseline overrides both while the count ≤ the recorded `count`
- [ ] T020 [US2] `backend/internal/sourcecheck/baseline.go`: read and write `quality/baseline.json` (`path` repository-relative with forward slashes, unique per `metric`; `metric` `"lines"` or `"comments"`; `count` integer) and `quality/exceptions.json` (`path` unique; `lines` 301–400; `reason` non-empty); compare, seed (refuse if the file exists), tighten (never raise, never add)
- [ ] T021 [US2] `backend/internal/sourcecheck/report.go` and `Run`: one line per problem and per improvable or removable entry, then `N problems in M files; baseline: B entries (T can be tightened)`; exit 0 / 1 / 2 as in [contracts/sourcecheck.md](contracts/sourcecheck.md)
- [ ] T022 [US2] `backend/cmd/sourcecheck/main.go`: flags `-root` (default: `git rev-parse --show-toplevel`), `-tighten`, `-seed`; calls `sourcecheck.Run` and exits with its code
- [ ] T023 [US2] Seed the baseline with `go -C backend run ./cmd/sourcecheck -seed`, create `quality/exceptions.json` as `[]`, check that every baseline entry is a file that existed on `develop` at `049ebf4`, then `go -C backend run ./cmd/sourcecheck` exits 0
- [ ] T024 [US2] The `repo-checks` job in `.github/workflows/ci.yml`: checkout, setup-go (as T009), then the named step `Size and comments` (`go -C backend run ./cmd/sourcecheck` from the repository root)
- [ ] T025 [US2] Hold the new package to the full constitution: `golangci-lint run ./internal/sourcecheck/... ./cmd/sourcecheck/...` (all lines, not only new) has no findings; every new file ≤ 300 non-blank lines and within its comment budget; `go test -race ./internal/sourcecheck/...` passes; `go -C backend run ./cmd/sourcecheck` finishes in under 30 seconds (SC-004)

**Checkpoint**: quickstart §2 behaves as written, locally and in the `repo-checks` check.

---

## Phase 5: User Story 3 — Nobody pushes straight to the shared branches (P2)

**Goal**: the branch-name check, the dependency review, the `release` branch and the written-down
branch rules (FR-011, FR-013, FR-014, FR-017).

**Independent test**: [quickstart.md](quickstart.md) §4.

- [ ] T026 [US3] Add two named steps to `repo-checks` in `.github/workflows/ci.yml`, both `if: github.event_name == 'pull_request'`: `Branch name` — with `MODULE_BRANCHES: plan-module`, allow into `develop` only `feature/[0-9][0-9][0-9]-*`, `docs/*`, `fix/*` or a module branch; into `release` only `develop` or `fix/*`; into `main` only `release`; otherwise print the allowed names and exit 1 (FR-011) — and `Dependency review` (`actions/dependency-review-action@v5`, `fail-on-severity: high`) (FR-017)
- [ ] T027 [US3] Write `.github/rulesets/protected-branches.json`: `name` "Protected branches", `target` branch, `enforcement` active, `bypass_actors` `[]`, `conditions.ref_name.include` `refs/heads/main`, `refs/heads/release`, `refs/heads/develop`, `exclude` `[]`; rules `deletion`, `non_fast_forward`, `pull_request` (`required_approving_review_count` 1, `dismiss_stale_reviews_on_push` true, `require_last_push_approval` true, `require_code_owner_review` false, `required_review_thread_resolution` false), `required_status_checks` (`strict_required_status_checks_policy` false) listing `backend`, `frontend`, `e2e`, `repo-checks` (FR-014, R8)
- [ ] T028 [US3] Create `release` from `main`: `git push origin origin/main:refs/heads/release` (FR-013)

**Checkpoint**: the ruleset lists exactly the workflow's four job names; `release` exists.

---

## Phase 6: User Story 4 — New features start on correctly named branches (P3)

- [X] T029 [US4] Set `branch_prefix: "feature"` in `.specify/extensions/git/git-config.yml`, so Spec Kit creates `feature/NNN-short-name` (FR-012; this branch was created that way)

---

## Phase 7: Polish & Cross-Cutting Concerns

- [ ] T030 [P] Write `docs/architecture/adr/0010-quality-gates-and-branch-rules.md` (Context · Decision · Consequences) and add its row to `docs/architecture/adr/README.md` (FR-019, R14): the pipeline and its four checks, the branch rules, the size and comment checks with the baseline, the release age through npm's setting; the deviations the product owner agreed on 2026-10-26 — static code scanning (gate 16) until SonarQube, GitHub code scanning not used; SonarQube, Apiiro and Mend until organisation setup; secret scanning until an administrator switches it on; the 80% new-code coverage threshold until SonarQube; the first-page JavaScript budget (gate 12) until per-route splitting; Go modules' release age checked by reviewers; follow-ups: approved board, software asset registration, ECCN, pinning actions by commit
- [ ] T031 [P] Write `docs/sdd/quality-gates.md` (FR-015): the four checks, their steps and how to run each locally; how to record a size exception (`quality/exceptions.json`, with a reason); how the baseline works (`-tighten`, no new entries); the release-age rule (npm setting; how a reviewer checks a Go module's date); how an administrator applies the ruleset (`gh api --method POST repos/anup-zensar/MAPP/rulesets --input .github/rulesets/protected-branches.json`) and switches on secret scanning
- [ ] T032 [P] Update `docs/sdd/sdd-process.md` §0 and §6 to the branch model of constitution 3.x (`feature/NNN-short-name` from `develop` or the module branch, no direct pushes to `main`, `release`, `develop`, pull requests reviewed by someone other than the author) and link `docs/sdd/quality-gates.md`
- [ ] T033 [P] Update `docs/sdd/feature-map.md`: a 019 row and the alignment roadmap 020–027 as Planned (Shell palette, comment budget, API v1, PostgreSQL storage, Gin, backend layout, web-app structure, test hygiene)
- [ ] T034 [P] Add `go -C backend run ./cmd/sourcecheck`, `npm run test:coverage` and the release-age rule to `CLAUDE.md` and `backend/CLAUDE.md`
- [ ] T035 Run every command of [quickstart.md](quickstart.md) §1 on the branch; all pass (SC-002)
- [ ] T036 Push `feature/019-alignment-guardrails`, open a pull request into `develop` that asks a repository administrator to apply the ruleset and switch on secret scanning; confirm the four checks report within 20 minutes and pass (SC-001, SC-002, SC-006); leave the pull request open for review
- [ ] T037 After the administrator applies the ruleset: run [quickstart.md](quickstart.md) §4 and record the result in the pull request (SC-005) — waits for the administrator

---

## Dependencies & Execution Order

### Phase dependencies

- **Setup (Phase 1)** first: the gates must be green on today's code.
- **Foundational (Phase 2)** before US1: the pipeline calls `test:coverage` and the stricter `lint`.
  T004 before T005, so the age rule applies to the new dependency.
- **US1 (Phase 3)** and **US2 (Phase 4)** after Phase 2; T024 adds the fourth job to the workflow
  from T008.
- **US3 (Phase 5)**: T026 extends `repo-checks` (after T024); T027 lists the four checks.
- **US4** is done.
- **Polish** last; T036 needs everything else; T037 waits for an administrator.

### Within User Story 2

T013–T015 (tests) before T016–T022; T017 and T018 in parallel; T019 needs T017–T018; T020 before
T021; T022 after T021; T023 after T022; T024 and T025 last.

### Parallel opportunities

- T002 ‖ T003; T007 ‖ T006 once T005 is done.
- T013 ‖ T014 ‖ T015; T017 ‖ T018.
- T030 ‖ T031 ‖ T032 ‖ T033 ‖ T034.

## Parallel Example: User Story 2

```text
Task: "Comment-counting tests in backend/internal/sourcecheck/comments_test.go"
Task: "Limit, exception and generated-file tests in backend/internal/sourcecheck/limits_test.go"
Task: "Baseline tests in backend/internal/sourcecheck/baseline_test.go"
```

## Implementation Strategy

### MVP first

Phases 1–3: today's code is green and every pull request is checked by three of the four checks.
That alone protects the alignment features that follow.

### Incremental delivery

1. Setup + Foundational → gates green locally; npm's release age on.
2. US1 → the pipeline (MVP).
3. US2 → size and comment checks with the baseline.
4. US3 → branch names, dependency review, `release`, the ruleset for an administrator.
5. Polish → ADR-0010, documentation, full validation, pull request left open for review.

## Notes

- Every new file follows the constitution in full: ≤ 300 non-blank lines, comments ≤ 5%, the
  function metrics, no test for the sake of a test.
- Commit after each phase, authored by Rushikesh Jawale, Conventional Commits with `019`.
