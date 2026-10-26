# Implementation Plan: Alignment Guardrails

**Branch**: `feature/019-alignment-guardrails` | **Date**: 2026-10-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/019-alignment-guardrails/spec.md`

## Summary

A GitHub Actions workflow with four checks — `backend`, `frontend`, `e2e` and `repo-checks` — runs
every quality gate the constitution names as a named step, on pull requests and pushes to `main`,
`release` and `develop`. GitHub's dependency review and `npm audit` cover the dependency gates. npm's built-in
`min-release-age=1` refuses package versions younger than a day. Two dependency fixes make the
vulnerability gate pass on today's code. A committed ruleset, applied by a repository administrator,
makes the three branches pull-request only, and ADR-0010 records the pipeline and the agreed
deviations. No application behaviour changes ([research](research.md) R1–R15).

## Technical Context

**Language/Version**: Go 1.27 (`toolchain go1.27.3`), TypeScript 5.9, Node 24 (npm ≥ 11.10); GitHub
Actions YAML

**Primary Dependencies**: GitHub Actions (`checkout`, `setup-go`, `setup-node`, `upload-artifact`
v7; `golangci-lint-action` v9; `dependency-review-action` v5); golangci-lint v2.15.0, `goimports`
(x/tools v0.52.0), `govulncheck` v1.9.0 run as pinned tools; `@vitest/coverage-v8` 5.0.5 (new dev
dependency, R9).

**Storage**: Files in the repository: `frontend/.npmrc`, `.github/rulesets/protected-branches.json`. The application's storage is
untouched.

**Testing**: no new code needs unit tests (configuration only); the pipeline is validated by the
scenarios in [quickstart.md](quickstart.md) and by its own first run on the pull request.

**Target Platform**: GitHub-hosted `ubuntu-latest` runners; developers' macOS, Windows and Linux
machines for the local commands.

**Project Type**: Web application (Go API + React web app); this feature adds tooling only.

**Performance Goals**: every check reports within 20 minutes of a push (SC-001).

**Constraints**: no secrets; `contents: read` token; four checks, no more (product owner); no
application behaviour change (FR-016).

**Scale/Scope**: 4 checks with 15 gate steps over the whole repository.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle / standard | Check | Result |
|---|---|---|
| I Spec-first | Spec with FR-001–FR-020 (FR-006–FR-009 and FR-018 withdrawn) and acceptance scenarios; branch `feature/019-…` | Pass |
| III Documented design | Plan, research, data model, contracts, quickstart; threat and risk model below; ADR-0010 | Pass |
| IV Two parts | CI is not part of the application | Pass |
| V Simplicity | One workflow with four jobs; no new tool to maintain (product owner) | Pass |
| VI Meaningful tests | No test is written for configuration; today's failing tests were fixed, not skipped | Pass |
| VII Security | No secrets; least-privilege token; fork pull requests run with a read-only token; release age through npm's `min-release-age`, reviewers for Go modules | Pass |
| VIII Clean construction | The comment budget and file length stay review checks (gate 14) | Pass |
| B1 / F1 stack | No new Go module dependency; one dev dependency justified in R9; scripts `test:coverage` and `lint --max-warnings 0` per F1 | Pass |
| Gates 1–14 | Gates 1, 2, 5–13 run as steps, gate 2's lint on changed lines (Alignment); gate 14 is checked in review; gate 12's JavaScript budget deferred until per-route splitting — recorded in ADR-0010 | Pass |
| Gates 15–18 | Dependency review and `npm audit` run now; static code scanning (gate 16, until SonarQube), SonarQube, Apiiro, Mend and switching on secret scanning are deviations agreed by the product owner on 2026-10-26, recorded in ADR-0010; `gosec` in golangci-lint covers common Go security mistakes meanwhile | Pass |
| Gates 5 and 11, coverage ≥ 80% | Reports published now; the threshold waits for SonarQube — deviation recorded in ADR-0010 | Pass |
| Workflow | Feature branch from `develop`, pull request left open for review | Pass |
| Governance → Alignment | Legacy lint findings reported on changed lines only; no package migrated | Pass |

**Post-design re-check (after Phase 1)**: unchanged — Pass. No Complexity Tracking entries.

### Threat model (STRIDE) for the new pipeline

| Threat | Where | Response |
|---|---|---|
| Spoofing: a fork's pull request pretends to be trusted | `pull_request` from forks (the repository is public) | `pull_request` (not `pull_request_target`) runs fork code with a read-only token and no secrets |
| Tampering: a moved action tag injects code | third-party actions | Only official `actions/*`, `golangci/golangci-lint-action` and `actions/dependency-review-action`, pinned by major version; pinning by commit is a follow-up (ADR-0010) |
| Tampering: a compromised package version is installed as soon as it is published | npm and Go dependencies | npm refuses versions younger than a day (`min-release-age=1`, npm ≥ 11.10 required); reviewers check new Go module versions |
| Repudiation: a direct push bypasses review | `main`, `release`, `develop` | Ruleset: pull request required, no bypass actors, last push must be approved by someone else |
| Information disclosure: secrets in logs | all jobs | The workflow uses no secrets; logs contain only build and test output of public code |
| Denial of service: a hung job blocks merges | e2e, tests | `timeout-minutes` on every job; `concurrency` cancels superseded runs |
| Elevation of privilege: a job writes to the repository | all jobs | `permissions: contents: read` at workflow level; `backend` adds `pull-requests: read` for the lint step — nothing writes |

### Risk model

| Risk | Response | Owner |
|---|---|---|
| A flaky end-to-end test blocks a merge | Tests run with `retries: 0` and pass 36/36; a flaky test is fixed, never retried away | Team |
| The ruleset is not applied (no administrator rights here) | Documented command; the pull request description asks the administrator to run it | Repository administrator |
| A young Go module version slips through review | Accepted by the product owner: no built-in check exists for Go; recorded in ADR-0010 | Product owner |
| Only one developer available, so no second reviewer | Accepted: the rule is constitutional; merges wait for a reviewer | Product owner |

## Project Structure

### Documentation (this feature)

```text
specs/019-alignment-guardrails/
├── plan.md              # this file
├── research.md          # decisions R1–R15
├── data-model.md        # gate, branch rule, release age
├── quickstart.md        # validation scenarios
├── contracts/
│   └── ci-checks.md     # the four checks and their steps
└── tasks.md             # /speckit-tasks
```

### Source Code (repository root)

```text
.github/
├── workflows/ci.yml                     # the pipeline: four jobs (R2)
└── rulesets/protected-branches.json     # branch rules for main, release, develop (R8)
backend/
└── go.mod / go.sum                      # toolchain go1.27.3; golang.org/x/net v0.60.0 (R3)
backend/internal/approvals/rounds_test.go  # one gofmt -s fix (R1)
frontend/
├── .npmrc                               # min-release-age=1, engine-strict (R15)
├── package.json                         # test:coverage, lint --max-warnings 0, browserslist, engines.npm
├── package-lock.json
└── vite.config.ts                       # coverage settings for test:coverage
docs/architecture/adr/
├── 0010-quality-gates-and-branch-rules.md   # pipeline, branch rules, checks, deviations (R14)
└── README.md                            # index row
docs/sdd/
├── quality-gates.md                     # how to run each gate locally, review checks, ruleset
└── feature-map.md                       # 019 row and the alignment roadmap
.specify/extensions/git/git-config.yml   # branch_prefix: feature (already on this branch)
```

**Structure Decision**: configuration and documentation only. No application package changes
except the one-line formatting fix and the test fixes of R1.

## Complexity Tracking

No violations to justify.
