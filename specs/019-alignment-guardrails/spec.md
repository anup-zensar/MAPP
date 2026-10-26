# Feature Specification: Alignment Guardrails

**Feature Branch**: `feature/019-alignment-guardrails`

**Created**: 2026-10-26

**Status**: Draft

**Input**: User description: "Alignment guardrails (first feature of the constitution alignment roadmap). Add the automated checks and repository settings the constitution requires, so every later alignment feature is verified automatically, without changing application behaviour: a pipeline running the quality gates the current code can pass on every pull request; comment-budget and file-length checks that block new problems; the package scripts and settings the constitution names; feature branches named feature/NNN-short-name; branch protection on main, release and develop."

**References**: Constitution 3.0.0 — Development Workflow and Quality Gates (gates 1–14, branches,
no direct pushes), Principle V (quality metrics, file length), Principle VI (coverage), Principle
VIII (comment budget), Governance → Alignment and Exceptions · Constitution F1 (package scripts,
`browserslist`) · Feature 001 research (today's stack and commands)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Every pull request is checked before it can merge (Priority: P1)

As a developer I open a pull request into `develop`, and within minutes I see, gate by gate, whether
my change keeps the code formatted, type-safe, lint-clean, tested, buildable and free of known
vulnerabilities. A reviewer cannot merge it while any required gate is red.

Rushikesh opens a pull request that changes a Go service and a React screen. The pull request shows
one named check per gate. The backend lint check reports a finding on a line he changed and
nothing about older lines he did not touch; he fixes it, pushes, and every check turns green. The
test results and coverage reports of the run are attached to it for the reviewer.

**Why this priority**: every later alignment feature rewrites large parts of the code; without
automatic gates, regressions would only be found by hand.

**Independent Test**: open a pull request with one deliberate problem per gate (an unformatted
file, a type error, a failing test, a lint finding on a changed line) and check that exactly that
gate fails and names the problem; then open a clean pull request and check that all gates pass.

**Acceptance Scenarios**:

1. **Given** a pull request into `develop`, `release` or `main`, **When** it is opened or updated,
   **Then** every quality gate of FR-002 and FR-003 runs as its own named check and reports pass or
   fail.
2. **Given** a pull request with an unformatted Go file, **When** the checks run, **Then** the
   backend formatting check fails and names the file, and the other checks are unaffected.
3. **Given** a pull request that changes one line of a Go file that already had lint findings
   elsewhere, **When** the lint check runs, **Then** it reports only findings on new and changed
   lines.
4. **Given** a pull request whose frontend test fails, **When** the checks run, **Then** the
   frontend test check fails, and the run still publishes the test results and the coverage
   report.
5. **Given** today's `develop` with this feature merged, **When** the pipeline runs on it, **Then**
   every gate passes.

---

### User Story 2 - New code stays small and sparsely commented (Priority: P1)

As the product owner I want every new or changed file to respect the file-length limit and the
comment budget, so the codebase gets leaner with each change instead of drifting back.

A developer adds a 420-line component with a comment on every function. The pull request's size
check fails, naming the file, its 420 non-blank lines and the 400-line hard limit; the comment check
fails, naming the file, its comment lines and its budget. He splits the component in two and keeps
two comments that explain *why*; both checks pass. A colleague changes a legacy 900-line Go file
that predates the limits: the check passes as long as the file does not grow.

**Why this priority**: the product owner asked for both limits explicitly; without a check they
erode within weeks.

**Independent Test**: add files just under and over each limit, and change a recorded legacy file
up and down by one line, then compare the verdicts with FR-006 to FR-009, in the pipeline and with
the local command.

**Acceptance Scenarios**:

1. **Given** a new source file of 401 non-blank lines, **When** the size check runs, **Then** it
   fails and names the file, its count and the limit.
2. **Given** a new source file of 320 non-blank lines with an exception recorded for it with a
   reason, **When** the size check runs, **Then** it passes and lists the exception; **Given** the
   same file without a recorded exception, **Then** it fails.
3. **Given** a new 100-line file with 6 comment lines, **When** the comment check runs, **Then** it
   fails (budget 5); with 5 comment lines it passes; tool directives and blank lines are not
   counted.
4. **Given** a legacy file recorded in the baseline at 900 lines, **When** a change brings it to
   901 lines, **Then** the size check fails; at 899 lines it passes and reports that the baseline
   can be lowered.
5. **Given** a generated file (the generated API types, code marked "Code generated … DO NOT
   EDIT"), **When** the checks run, **Then** it is ignored.
6. **Given** a developer's machine, **When** the developer runs the one local command, **Then** it
   gives the same verdicts as the pipeline for the same files.

---

### User Story 3 - Nobody pushes straight to the shared branches (Priority: P2)

As the team I want `main`, `release` and `develop` to accept changes only through reviewed pull
requests from working branches, so every change is seen by a second person and passes the gates.

Anup tries to push a quick fix directly to `develop`; the push is refused. He pushes it to
`fix/date-label`, opens a pull request, a colleague approves it, the checks pass, and it merges.
A pull request into `develop` from a branch named `quickfix` is flagged by the branch-name check.

**Why this priority**: the rule is in the constitution and the product owner asked for it, but it
only matters once the gates of stories 1 and 2 exist.

**Independent Test**: try a direct push, a force push and a self-approved merge on each protected
branch (all refused), and a pull request from a badly named branch (flagged).

**Acceptance Scenarios**:

1. **Given** the branch rules applied, **When** anyone (administrators included) pushes
   directly to `main`, `release` or `develop`, **Then** the push is refused.
2. **Given** a pull request into a protected branch, **When** it has no approval from someone other
   than its author, or any required check is red, **Then** it cannot be merged.
3. **Given** an approved pull request, **When** its author pushes a new commit, **Then** the earlier
   approval no longer counts.
4. **Given** a pull request into `develop` from a branch not named `feature/NNN-short-name`,
   `docs/…`, `fix/…` or a module branch, **When** the checks run, **Then** the branch-name check
   fails and says which names are allowed.
5. **Given** a pull request into `main` from any branch other than `release`, or into `release`
   from any branch other than `develop` or `fix/…`, **Then** the branch-name check fails.

---

### User Story 4 - New features start on correctly named branches (Priority: P3)

As a developer I run the Spec Kit specify step and get a branch named `feature/NNN-short-name`,
matching the spec folder `specs/NNN-short-name`, without typing the name myself.

**Why this priority**: small, and already set up for this very feature; it is recorded so it stays
that way.

**Independent Test**: run the specify step for a dummy feature in a scratch clone and check the
branch name.

**Acceptance Scenarios**:

1. **Given** the Spec Kit configuration, **When** a new feature is specified, **Then** its branch
   is `feature/NNN-short-name` and its folder `specs/NNN-short-name`, with the same number.

### Edge Cases

- A pull request that changes only documentation still runs every check, so required checks never
  wait for a result that will not come.
- A file renamed in a pull request is judged as the file it now is; a recorded baseline entry
  follows the rename only through an edit to the baseline in the same pull request.
- A file that shrinks below the limit is no longer an exception; the check reports that its
  baseline entry or exception can be removed, and does not fail for it.
- Test files are held to the same limits; table-driven tests are split by behaviour.
- A check that cannot run (the runner fails, a download times out) fails the pull request rather
  than passing it silently.
- A known vulnerability found in a dependency that today's code uses fails the vulnerability gate;
  this feature then upgrades the dependency, or records why the finding is not reachable.
- A single developer cannot approve their own pull request; the team needs at least two people with
  write access for the protected flow to work.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A pipeline MUST run on every pull request into `main`, `release` or `develop`, and on
  every push to those branches, using only the repository's own hosted runners and no secrets.
- **FR-002**: The pipeline MUST run these backend gates as separate named checks: formatting
  (`gofmt` and `goimports`), `go vet`, `golangci-lint` with `backend/.golangci.yml` reporting only
  findings on new and changed lines compared with the pull request's base, all Go tests with the
  race detector and a coverage profile, and `govulncheck`.
- **FR-003**: The pipeline MUST run these frontend gates as separate named checks: formatting,
  type check, lint with warnings counted as failures, the generated-API-types drift check, unit and
  component tests with branch coverage reported as LCOV, the production build, and the end-to-end
  tests against the real API with the seed data.
- **FR-004**: Every gate of FR-002, FR-003, FR-006, FR-007, FR-011, FR-017, FR-018 and FR-020 MUST be
  able to block a merge:
  each reports its own pass or fail, and a gate that cannot run counts as failed.
- **FR-005**: Each run MUST publish its test results and coverage reports (Go coverage profile,
  frontend LCOV) so that a reviewer can open them from the pull request, kept for at least 30 days.
- **FR-006**: A comment check MUST fail when a hand-written source file (Go, TypeScript, TSX, CSS,
  SQL; tests included) has more comment lines than 5% of its non-blank lines, rounded up. Blank
  lines, tool directives (`//go:`, `//nolint`, `eslint-disable`, `eslint-enable`, `prettier-ignore`,
  `@ts-expect-error`, `/// <reference>`, goose and sqlc annotations) and generated files do not
  count. The failure names the file, its comment lines and its budget.
- **FR-007**: A size check MUST count each hand-written source file's non-blank lines and fail at
  401 or more, and at 301–400 unless an exception with a written reason is recorded for that file.
  The failure names the file, its count and the limit.
- **FR-008**: Files that exceed a limit when this feature ships MUST be recorded in a committed
  baseline with their current counts. A recorded file passes while its count does not rise, fails
  when it rises, and is reported as improvable when it falls. No file may be added to the baseline
  after this feature ships; entries leave it as their packages are aligned.
- **FR-009**: One local command MUST run the comment and size checks with the same rules, baseline
  and exceptions as the pipeline, and give the same verdict.
- **FR-010**: The web app's package settings MUST include the scripts and settings the constitution
  names that are missing today: a `test:coverage` script producing LCOV with branch coverage, and a
  `browserslist` entry naming the two most recent major versions of Chrome, Edge, Firefox and
  Safari, desktop and mobile.
- **FR-011**: A branch-name check MUST fail a pull request into `develop` from any branch other than
  `feature/NNN-short-name`, `docs/…`, `fix/…` or a module branch (today `plan-module`); into
  `release` from any branch other than `develop` or `fix/…`; into `main` from any branch other than
  `release`. The message lists the allowed names.
- **FR-012**: The Spec Kit configuration MUST create feature branches named
  `feature/NNN-short-name`, with the same number as the spec folder.
- **FR-013**: A `release` branch MUST exist, created from `main` if missing, so that it can be
  protected.
- **FR-014**: The branch rules (ruleset) for `main`, `release` and `develop` MUST be written down in
  the repository, together with a command a repository administrator runs to apply them: no direct
  pushes or force pushes for anyone (administrators included), no deletion, a pull request with at
  least one approval from someone other than the author, approvals dismissed by new commits, and
  every check of FR-004 required.
- **FR-015**: The developer documentation MUST say how to run every gate locally, how to record a
  size exception, how the baseline works, and how to apply the branch rules (ruleset).
- **FR-016**: This feature MUST NOT change application behaviour: no change to the API, the
  screens, the stored data or the seed data. A gate that today's code fails is fixed with the
  smallest change that keeps behaviour, or the gap is recorded with its reason.
- **FR-017**: The pipeline MUST run, as separate named checks, the security gates the repository can
  run without organisation-level setup: a review of the dependencies a pull request adds or
  changes, and the audit of the web app's production dependencies. Each fails on a high or
  critical finding.
- **FR-018**: The production build MUST fail when the JavaScript the first page loads, measured
  gzipped, exceeds the budget recorded in the repository. The budget starts at today's size plus a
  small margin and is lowered as the web app is split per route.
- **FR-019**: An architecture decision record MUST record the pipeline, the branch rules and the
  size and comment checks, and the deviations the product owner agreed on 2026-10-26: static code
  scanning (SAST, gate 16) waits for SonarQube — GitHub code scanning is not used; SonarQube,
  Apiiro and Mend wait for organisation setup; secret scanning is switched on by a repository
  administrator; the 80% new-code coverage threshold is enforced once SonarQube is available.
- **FR-020**: No package version younger than 24 hours MAY be installed or added (constitution
  VII). The web app's npm settings MUST refuse to install a version published less than one day
  ago, and a dependency-age check MUST fail a pull request that adds or changes an npm package or a
  Go module to a version published less than 24 hours before the check runs, naming the package,
  the version and its publication time.

### Key Entities

- **Quality gate**: a named check with the command it runs, the branches it guards, and whether a
  failure blocks merging.
- **Baseline entry**: a legacy file that exceeded a limit when the feature shipped — its path, the
  limit it exceeds (size or comments) and its recorded count.
- **Size exception**: a file allowed into the 301–400 band — its path, its count and the written
  reason.
- **Branch rule**: a protected branch with its allowed source branches and its rules in the
  ruleset.
- **JavaScript budget**: the gzipped size the first page's JavaScript may reach.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every pull request into `develop`, `release` or `main` shows a result for every gate
  within 20 minutes of its latest push.
- **SC-002**: The pipeline passes on `develop` once this feature is merged: zero failing gates on
  today's code.
- **SC-003**: After this feature ships, no new file over 400 non-blank lines or over its comment
  budget reaches `develop`, and no baseline entry grows.
- **SC-004**: A developer runs the size and comment checks locally in under 30 seconds, with the
  same verdict as the pipeline in 100% of cases.
- **SC-005**: Once an administrator applies the branch rules, 100% of direct pushes to `main`,
  `release` and `develop` are refused, and no pull request merges without an approval from someone
  other than its author.
- **SC-006**: No pull request merges with a high or critical finding from the dependency review or
  the production dependency audit, or with a first-page JavaScript size over its budget.
- **SC-007**: No package version published less than 24 hours earlier reaches `develop`.

## Assumptions

- The repository runs on GitHub with GitHub Actions minutes available; the pipeline uses only
  GitHub-hosted runners.
- The account used for this work has push rights but not administrator rights on the repository,
  so the branch rules are applied by a repository administrator using the documented command
  (FR-014).
- At least two people have write access, so a pull request can be approved by someone other than
  its author.
- Today's code is made to pass every gate first (research R1 found 1 Go, 2 frontend and 6
  end-to-end tests failing on `develop`, all fixed without behaviour change, FR-016).
- The 80% new-code coverage threshold (constitution, Principle VI) is enforced once SonarQube is
  set up (FR-019, ADR-0010); until then the coverage reports are published and reviewers check
  them.
- The storage still uses the JSON file store, so the pipeline needs no database yet; the
  PostgreSQL feature adds a PostgreSQL service to the pipeline.
- The repository is public, so the dependency review and secret scanning are available without
  organisation setup; SonarQube, Apiiro and Mend are not. The product owner chose not to use GitHub
  code scanning (FR-019).
- Out of scope: the stack moves (PostgreSQL, Gin, `/api/v1`); the Shell palette; pruning existing
  comments and splitting existing long files; restructuring the frontend and splitting it per
  route.
