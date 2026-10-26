# Implementation Plan: Alignment Guardrails

**Branch**: `feature/019-alignment-guardrails` | **Date**: 2026-10-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/019-alignment-guardrails/spec.md`

## Summary

A GitHub Actions workflow runs every quality gate the constitution names, one named job per gate,
on pull requests and pushes to `main`, `release` and `develop`. A new Go command, `sourcecheck`,
enforces the file-length limit and the comment budget, with a committed baseline that freezes
today's oversized files so they can only shrink. Two dependency fixes make the vulnerability gate
pass on today's code. A committed ruleset, applied by a repository administrator, makes the three
branches pull-request only. No application behaviour changes ([research](research.md) R1–R11).

## Technical Context

**Language/Version**: Go 1.27 (`toolchain go1.27.3`), TypeScript 5.9, Node 24; GitHub Actions YAML

**Primary Dependencies**: GitHub Actions (`checkout`, `setup-go`, `setup-node`,
`upload-artifact` v7; `golangci-lint-action` v9); golangci-lint v2.15.0, `goimports`
(x/tools v0.52.0), `govulncheck` v1.9.0 run as pinned tools; `@vitest/coverage-v8` 5.0.5 (new dev
dependency, R9). `sourcecheck` uses the Go standard library only.

**Storage**: Files in the repository: `quality/baseline.json`, `quality/exceptions.json`,
`.github/rulesets/protected-branches.json`. The application's storage is untouched.

**Testing**: Go `testing` + `go-cmp` for `sourcecheck` (table-driven, real files in
`t.TempDir()`); the pipeline itself is validated by the scenarios in [quickstart.md](quickstart.md).

**Target Platform**: GitHub-hosted `ubuntu-latest` runners; developers' macOS, Windows and Linux
machines for the local commands.

**Project Type**: Web application (Go API + React web app); this feature adds tooling only.

**Performance Goals**: every check reports within 20 minutes of a push (SC-001);
`sourcecheck` finishes in under 30 seconds on a laptop (SC-004).

**Constraints**: no secrets; `contents: read` token; no application behaviour change (FR-016);
new code within the constitution's limits (300 lines per file, 5% comments, function metrics).

**Scale/Scope**: about 1,350 source files checked (523 Go, 815 TypeScript, 207 CSS); 14 checks.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle / standard | Check | Result |
|---|---|---|
| I Spec-first | Spec with FR-001–FR-016 and acceptance scenarios exists; branch `feature/019-…` | Pass |
| III Documented design | Plan, research, data model, contracts, quickstart; threat and risk model below | Pass |
| IV Two parts | `sourcecheck` is a developer tool, not a runtime part; CI is not part of the application | Pass |
| V Simplicity, metrics | `sourcecheck` split into files under 300 lines, functions within the metrics; one workflow file | Pass |
| VI Meaningful tests | Tests prove the counting rules, directives, generated-file skipping, baseline ratchet and exceptions; no tests of flags parsing or of the standard library | Pass |
| VII Security | No secrets; least-privilege token; fork pull requests run with a read-only token | Pass |
| VIII Clean construction | New files within the comment budget and the size limit; `sourcecheck` checks itself | Pass |
| B1 / F1 stack | No new Go module dependency; one dev dependency justified in R9; scripts `test:coverage` and `lint --max-warnings 0` per F1 | Pass |
| B2 layout | `cmd/sourcecheck/main.go` only parses flags and calls `run`; logic in `internal/sourcecheck` | Pass |
| Workflow | Feature branch from `develop`, pull request left open for review | Pass |
| Governance → Alignment | Legacy findings handled by the baseline and by lint on changed lines only; no package migrated | Pass |

**Post-design re-check (after Phase 1)**: unchanged — Pass. No Complexity Tracking entries.

### Threat model (STRIDE) for the new pipeline

| Threat | Where | Response |
|---|---|---|
| Spoofing: a fork's pull request pretends to be trusted | `pull_request` from forks (the repository is public) | `pull_request` (not `pull_request_target`) runs fork code with a read-only token and no secrets |
| Tampering: a moved action tag injects code | third-party actions | Only official `actions/*` and `golangci/golangci-lint-action`, pinned by major version; pinning by commit SHA is a follow-up once SonarQube/GHAS are set up (the product owner accepts or rejects this risk when reviewing the pull request) |
| Repudiation: a direct push bypasses review | `main`, `release`, `develop` | Ruleset: pull request required, no bypass actors, last push must be approved by someone else |
| Information disclosure: secrets in logs | all jobs | The workflow uses no secrets; logs contain only build and test output of public code |
| Denial of service: a hung job blocks merges | e2e, tests | `timeout-minutes` on every job; `concurrency` cancels superseded runs |
| Elevation of privilege: a job writes to the repository | all jobs | `permissions: contents: read` at workflow level; the lint job adds `pull-requests: read` only |

### Risk model

| Risk | Response | Owner |
|---|---|---|
| A flaky end-to-end test blocks a merge | Tests run with `retries: 0` today and passed 36/36; a flaky test is fixed, never retried away | Team |
| The baseline hides how much legacy debt remains | `sourcecheck -tighten` and the summary line show the count; alignment features empty it | Product owner |
| The ruleset is not applied (no administrator rights here) | Documented command; the pull request description asks the administrator to run it | Repository administrator |
| Only one developer available, so no second reviewer | Accepted: the rule is constitutional; merges wait for a reviewer | Product owner |

## Project Structure

### Documentation (this feature)

```text
specs/019-alignment-guardrails/
├── plan.md              # this file
├── research.md          # decisions R1–R11
├── data-model.md        # baseline, exception, gate, branch rule
├── quickstart.md        # validation scenarios
├── contracts/
│   ├── ci-checks.md     # the 14 checks, their commands and names
│   └── sourcecheck.md   # command, flags, output, exit codes, file formats
└── tasks.md             # /speckit-tasks
```

### Source Code (repository root)

```text
.github/
├── workflows/ci.yml                     # the pipeline: one job per gate (R2)
└── rulesets/protected-branches.json     # branch rules for main, release, develop (R8)
quality/
├── baseline.json                        # legacy files over a limit, with their counts (R6)
└── exceptions.json                      # files allowed into 301–400 lines, with reasons
backend/
├── go.mod / go.sum                      # toolchain go1.27.3; golang.org/x/net v0.60.0 (R3)
├── cmd/sourcecheck/main.go              # flags → sourcecheck.Run
└── internal/sourcecheck/
    ├── files.go                         # which files are checked; generated-file detection
    ├── comments.go                      # comment lines for Go (go/scanner)
    ├── lexer.go                         # comment lines for TypeScript, TSX, CSS, SQL
    ├── limits.go                        # budgets, verdicts
    ├── baseline.go                      # baseline and exceptions: read, compare, seed, tighten
    ├── report.go                        # output lines and summary
    └── *_test.go                        # rule tests on real files in t.TempDir()
backend/internal/approvals/rounds_test.go  # one gofmt -s fix (R1)
frontend/
├── package.json                         # test:coverage, lint --max-warnings 0, browserslist
├── package-lock.json
└── vite.config.ts                       # coverage settings for test:coverage
docs/sdd/
├── quality-gates.md                     # how to run each gate locally, exceptions, baseline, ruleset
└── feature-map.md                       # 019 row and the alignment roadmap
.specify/extensions/git/git-config.yml   # branch_prefix: feature (already on this branch)
```

**Structure Decision**: tooling only. The checker lives in the backend Go module because it
needs `go/scanner` and the module already has the toolchain every developer runs; its package is
new, so it follows the constitution in full. No application package changes except the one-line
formatting fix.

## Complexity Tracking

No violations to justify.
