# Research: Alignment Guardrails (019)

Findings come from running every gate on `develop` at `049ebf4` on 2026-10-26 (Go 1.27.1, Node
24.18, npm 11.16) and from the GitHub API for the repository and the actions used.

## R1 — Today's code against each gate

| Gate | Result on `develop` | Consequence |
|---|---|---|
| `gofmt -l` | clean | — |
| `gofmt -s` (golangci-lint's formatter) | 1 file: `internal/approvals/rounds_test.go:37` (a redundant composite-literal type) | Fix the one line (FR-016) |
| `goimports -l -local mapp` | clean | — |
| `go vet ./...` | clean | — |
| `golangci-lint run` (config verified with v2.15.0) | 488+ findings in legacy code (linters capped at 50 each) | Lint gate reports **new and changed lines only** (R4) |
| `govulncheck ./...` | exit 3: 10 standard-library vulnerabilities reached (fixed in Go 1.27.2/1.27.3); `golang.org/x/net` v0.50.0 has findings not reached (fixed in v0.60.0) | Pin `toolchain go1.27.3`; upgrade `golang.org/x/net` to v0.60.0 (R3) |
| `npm run format:check`, `typecheck`, `api:check` | pass | — |
| `eslint . --max-warnings 0` | pass | `lint` script gains `--max-warnings 0` |
| `go test -race ./...` | 1 failure: `TestQueryLog_50000Entries_SC006` asserts a wall-clock 2 s; it takes 1.94 s without the race detector and 2.15 s with it, on its own | Measure the 2 s only without the race detector (build tag); the test still checks every query answers. Gap recorded: a wall-clock assertion is not deterministic (Principle VI) — it becomes a benchmark when the history package is aligned |
| `npm test` (2,281 tests) | 2 failures, every run: `CampaignPanel` and `LineItemPage` tests make a real request (Budget by month; GSAP financials of the duplicated line item) with no canned answer — they pass only where the request happens after the test ends | Add the two canned answers with the existing helpers (`budgetByMonth()`, `financialsNotServed()`) |
| `npm run test:e2e` (36 tests) | 6 failures: 5 axe `color-contrast` findings on `ConfirmDialog` while it fades in (opacity < 1); 1 `locator.check()` on a preference-backed checkbox that turns checked a tick after the click | Playwright runs with `reducedMotion: 'reduce'` (the app already drops animations then, `global.css`), and the checkbox step uses `click()` + `toBeChecked()` (web-first). After the fixes: 36/36 |

## R2 — Pipeline: one GitHub Actions workflow, one job per gate

- **Decision**: `.github/workflows/ci.yml`, triggered by `pull_request` and `push` on `main`,
  `release` and `develop`; one job per gate, each job's `name` equal to its id, so check names are
  stable for the branch rules: `backend-format`, `backend-vet`, `backend-lint`, `backend-test`,
  `backend-vulnerabilities`, `frontend-format`, `frontend-typecheck`, `frontend-lint`,
  `frontend-api-types`, `frontend-test`, `frontend-build`, `e2e`, `source-limits`, `branch-name`.
  `permissions: contents: read` (plus `pull-requests: read` for the lint job); `concurrency`
  cancels superseded runs of the same ref.
- **Rationale**: FR-004 asks for one named, blocking check per gate; jobs are the unit GitHub
  reports and requires. Fourteen parallel jobs finish well inside SC-001's 20 minutes.
- **Alternatives**: one job with many steps (a single check, so a reviewer cannot see which gate
  failed and the branch rules cannot require gates one by one); path filters (required checks
  that never run would block merges — spec edge case).

**Pinned versions** (latest releases on 2026-10-26): `actions/checkout@v7`, `actions/setup-go@v7`
(reads `go.mod`, including `toolchain`), `actions/setup-node@v7` (`node-version: 24`, npm cache),
`actions/upload-artifact@v7` (`retention-days: 30`), `golangci/golangci-lint-action@v9` with
golangci-lint `v2.15.0`, `goimports` from `golang.org/x/tools` v0.52.0, `govulncheck` v1.9.0.

## R3 — Go toolchain and the vulnerability gate

- **Decision**: add `toolchain go1.27.3` to `backend/go.mod` (constitution B1: pinned by `go` and
  `toolchain`); `go get golang.org/x/net@v0.60.0` and `go mod tidy`. `GOTOOLCHAIN=auto` (Go's
  default) downloads 1.27.3 for anyone on an older 1.27.
- **Rationale**: smallest change that makes `govulncheck` pass without touching application code
  (FR-016).
- **Alternatives**: excluding findings (hides reachable standard-library vulnerabilities, which
  the constitution forbids).

## R4 — Lint only new and changed lines

- **Decision**: `golangci-lint-action` with `only-new-issues: true` (it diffs against the pull
  request's base, or the previous commit on a push). Locally:
  `golangci-lint run --new-from-merge-base=origin/develop ./...`.
- **Rationale**: the constitution's Alignment rule: findings on lines a change did not touch apply
  once a package is aligned.

## R5 — Formatting gate

- **Decision**: `gofmt -s -l .` and `goimports -local mapp -l .` must both print nothing; the job
  prints the files and fails otherwise. Developers fix with `golangci-lint fmt ./...`, which uses
  the same two formatters from `.golangci.yml`.
- **Rationale**: exactly quality gate 1, with no extra install beyond `go run` of a pinned
  `goimports`.

## R6 — Size and comment checker (`sourcecheck`)

- **Decision**: a small Go command, standard library only, in the backend module:
  `backend/internal/sourcecheck` (rules, counting, baseline) and `backend/cmd/sourcecheck`
  (flags, output). Run from anywhere with `go -C backend run ./cmd/sourcecheck`; it checks the
  whole repository.
  - **Files**: tracked and new, not ignored files from `git ls-files --cached --others
    --exclude-standard`, with extensions `.go .ts .tsx .css .sql`.
  - **Generated files** are skipped: a Go `// Code generated … DO NOT EDIT.` header, the
    openapi-typescript header ("auto-generated by openapi-typescript"), and paths listed in the
    configuration (`frontend/src/api/schema.d.ts`, `backend/internal/platform/database/sqlcdb/`).
  - **Comments**: Go via `go/scanner` (exact); TypeScript, TSX, CSS and SQL via a small lexer that
    skips string, template and regular-expression literals and knows `//`, `/* */`, `{/* */}` and
    SQL `--`. A line counts once if it holds any comment. Directives are not counted (FR-006 list).
  - **Limits**: comments ≤ ceil(5% × non-blank lines); size ≤ 300, 301–400 only with an exception,
    401+ never.
  - **Baseline** `quality/baseline.json` and **exceptions** `quality/exceptions.json` (sorted by
    path, so diffs are readable). `-seed` writes the baseline once and refuses if it exists;
    `-tighten` lowers recorded counts and drops entries now within limits, never raising one.
  - **Output**: one line per problem (`path: 420 non-blank lines; limit 400`), a summary, files
    that can be tightened; exit 1 on any problem, 2 on a usage or I/O error.
- **Rationale**: one implementation for the pipeline and the laptop gives the same verdict
  (FR-009, SC-004); `go/scanner` avoids miscounting `//` inside strings and URLs; Go needs no new
  dependency.
- **Alternatives**: a shell script with `grep` (miscounts comment markers in strings); a Node
  script with the TypeScript compiler (exact for TS but a second implementation for Go); an
  off-the-shelf tool such as `cloc` (counts comments but has no budget, baseline or exceptions,
  and is a new dependency).

## R7 — Branch names

- **Decision**: a `branch-name` job, run on pull requests only, compares `github.head_ref` with
  the rules of FR-011 for `github.base_ref`. Module branches are listed in the workflow
  (`plan-module`).
- **Rationale**: a few lines of shell, no dependency; the job's message lists the allowed names.

## R8 — Branch rules (protection)

- **Decision**: one repository ruleset, committed as `.github/rulesets/protected-branches.json`,
  targeting `main`, `release` and `develop`, with no bypass actors (administrators included):
  `deletion`, `non_fast_forward`, `pull_request` (1 approval, `dismiss_stale_reviews_on_push`,
  `require_last_push_approval`), and `required_status_checks` listing the 14 checks of R2. An
  administrator applies it with `gh api --method POST repos/anup-zensar/MAPP/rulesets --input
  .github/rulesets/protected-branches.json`.
- **Rationale**: the repository is public, so rulesets are available; the account used here has
  push but not administrator rights (`gh api …/MAPP` → `admin: false`), so the settings are
  written down and applied by an administrator (FR-014). Rulesets express "no direct pushes, for
  everyone" in one place for all three branches.
- **Alternatives**: classic branch protection, one call per branch (three copies of the same
  settings).

## R9 — Frontend coverage and test results

- **Decision**: add `@vitest/coverage-v8` 5.0.5 (matches `vitest` 5.0.5; MIT; maintained by the
  Vitest team; no install scripts) as a dev dependency. `test:coverage` runs `vitest run
  --coverage` with the `lcov` and `text-summary` reporters over `src/**/*.{ts,tsx}` (tests and the
  generated `schema.d.ts` excluded) and writes a JUnit report (`test-results/junit.xml`) for the
  pipeline. V8 coverage includes branch coverage.
- **Rationale**: constitution F1 names `test:coverage`, F10 asks for LCOV and a machine-readable
  report.

## R10 — `release` branch, browsers and Node

- **Decision**: create `release` from `origin/main` (push rights suffice). Add `browserslist`:
  `last 2 Chrome versions`, `last 2 ChromeAndroid versions`, `last 2 Edge versions`, `last 2
  Firefox versions`, `last 2 FirefoxAndroid versions`, `last 2 Safari versions`, `last 2 iOS
  versions`. CI uses Node 24, the version in use today, which satisfies `engines.node`
  (`>=22.22`).
- **Rationale**: constitution F1 and the branch model; nothing in the build reads `browserslist`
  yet, so behaviour does not change (FR-016).

## R11 — What the pipeline cannot do yet

- **New-code coverage ≥ 80%** needs SonarQube (or a diff-coverage service); until then the
  reports are published and reviewers check them (spec, Assumptions).
- **Gates 15–18** (secret scanning, SAST, SCA, SonarQube) need organisation-level setup; recorded
  as follow-ups.
- **PostgreSQL** is not needed until the storage feature; that feature adds a `postgres` service
  to `backend-test` and `e2e`.
