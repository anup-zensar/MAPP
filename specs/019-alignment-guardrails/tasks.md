---

description: "Tasks for 019 Alignment Guardrails"
---

# Tasks: Alignment Guardrails

**Input**: Design documents from `specs/019-alignment-guardrails/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md),
[data-model.md](data-model.md), [contracts/](contracts/), [quickstart.md](quickstart.md)

**Tests**: this feature is configuration and documentation; no code needs unit tests. Today's
failing tests were fixed (T003); the pipeline is validated by the quickstart scenarios and its first
run on the pull request. User story 2 (an automated size and comment checker) was withdrawn by the
product owner on 2026-10-26: the two limits stay review checks (quality gate 14).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: the user story the task belongs to (US1, US3, US4)

---

## Phase 1: Setup (make today's code pass the gates)

- [X] T001 Pin the toolchain and fix the vulnerable dependency in `backend/go.mod` and `backend/go.sum`: `toolchain go1.27.3`, `golang.org/x/net` v0.60.0; `govulncheck` reports no vulnerabilities (R3)
- [X] T002 [P] Fix the one `gofmt -s` finding in `backend/internal/approvals/rounds_test.go:37` (R1)
- [X] T003 [P] Make today's tests pass without changing behaviour (R1): race-detector build tag for the 2 s timing in `backend/internal/history/query_test.go`; canned answers in `frontend/src/features/campaigns/CampaignPanel.test.tsx` and `frontend/src/features/line-items/LineItemPage.test.tsx`; `reducedMotion: 'reduce'` in `frontend/playwright.config.ts`; web-first checkbox step in `frontend/e2e/activity-log.spec.ts`

---

## Phase 2: Foundational (npm settings and scripts the pipeline calls)

- [X] T004 Release age with npm's built-in setting (FR-020, R15): `frontend/.npmrc` (`min-release-age=1`, `engine-strict=true`) and `engines.npm >=11.10.0` in `frontend/package.json`
- [X] T005 Dev dependency `@vitest/coverage-v8` 5.0.5 in `frontend/package.json` and `frontend/package-lock.json`
- [X] T006 Coverage settings in `frontend/vite.config.ts` and the `test:coverage` script (LCOV, JUnit)
- [X] T007 [P] `lint` with `--max-warnings 0` and `browserslist` in `frontend/package.json`

---

## Phase 3: User Story 1 — Every pull request is checked before it can merge (P1) 🎯 MVP

- [X] T008 [US1] `.github/workflows/ci.yml`: triggers, `contents: read`, concurrency (R2)
- [X] T009 [US1] The `backend` job: Format, Vet, Lint (new and changed lines), Test (race, coverage), Vulnerabilities; artifact `backend-test`
- [X] T010 [US1] The `frontend` job: Format, Type check, Lint, API types, Test (coverage), Build, Audit; artifact `frontend-test`
- [X] T011 [US1] The `e2e` job: Go, Node, Chromium, `npm run test:e2e`; artifact `e2e` on failure
- [X] T012 [US1] `actionlint` reports no findings; names match [contracts/ci-checks.md](contracts/ci-checks.md)

---

## Phase 4: User Story 3 — Nobody pushes straight to the shared branches (P2)

- [X] T013 [US3] The `repo-checks` job in `.github/workflows/ci.yml` with two named steps, both only on pull requests: `Branch name` (FR-011: into `develop` only `feature/NNN-*`, `docs/*`, `fix/*` or `plan-module`; into `release` only `develop` or `fix/*`; into `main` only `release`; otherwise print the allowed names and fail) and `Dependency review` (`actions/dependency-review-action@v5`, `fail-on-severity: high`, FR-017)
- [X] T014 [P] [US3] `.github/rulesets/protected-branches.json`: `main`, `release`, `develop`; no bypass; `deletion`, `non_fast_forward`, `pull_request` (1 approval, stale approvals dismissed, last push approved by someone else), `required_status_checks` `backend`, `frontend`, `e2e`, `repo-checks` (FR-014, R8)
- [ ] T015 [US3] Create `release` from `main`: `git push origin origin/main:refs/heads/release` (FR-013)

---

## Phase 5: User Story 4 — New features start on correctly named branches (P3)

- [X] T016 [US4] `branch_prefix: "feature"` in `.specify/extensions/git/git-config.yml` (FR-012)

---

## Phase 6: Polish

- [X] T017 [P] `docs/architecture/adr/0010-quality-gates-and-branch-rules.md` and its row in `docs/architecture/adr/README.md` (FR-019, R14): the pipeline, the branch rules, the release age; the agreed deviations (static code scanning until SonarQube, GitHub code scanning not used; SonarQube, Apiiro, Mend; secret scanning by an administrator; the 80% coverage threshold; the JavaScript budget; Go release age by review; the comment budget and file length by review)
- [X] T018 [P] `docs/sdd/quality-gates.md` (FR-015): the four checks and how to run each gate locally; the review checks (comment budget, file length, Go release age); how an administrator applies the ruleset and switches on secret scanning
- [X] T019 [P] `docs/sdd/sdd-process.md` branch section and `docs/sdd/feature-map.md` 019 row (with the alignment roadmap 020–027 as Planned)
- [X] T020 [P] `CLAUDE.md`: the npm release-age rule
- [ ] T021 Run every command of [quickstart.md](quickstart.md) §1; all pass (SC-002)
- [ ] T022 Push the branch, open a pull request into `develop` asking an administrator to apply the ruleset and switch on secret scanning; the four checks pass (SC-001, SC-002, SC-006); leave the pull request open for review
- [ ] T023 After the administrator applies the ruleset: run [quickstart.md](quickstart.md) §4 and record the result in the pull request (SC-005) — waits for the administrator

---

## Dependencies & Execution Order

- Phases 1–3 and 5 are done. T013, T014 and T017–T020 touch different files and run in parallel;
  T015 needs nothing else. T021 follows them all, T022 follows T021, T023 waits for an
  administrator.
