# Feature Specification: Alignment Guardrails

**Feature Branch**: `feature/019-alignment-guardrails`

**Created**: 2026-10-26

**Status**: Draft

**Input**: User description: "Alignment guardrails (first feature of the constitution alignment roadmap). Add the automated checks and repository settings the constitution requires, so every later alignment feature is verified automatically, without changing application behaviour: a pipeline running the quality gates the current code can pass on every pull request; the package scripts and settings the constitution names; feature branches named feature/NNN-short-name; branch protection on main, release and develop."

**References**: Constitution 3.1.0 — Development Workflow and Quality Gates (gates 1–14, branches,
no direct pushes), Principle VI (coverage), Principle VII (release age), Governance → Alignment and
Exceptions · Constitution F1 (package scripts,
`browserslist`) · Feature 001 research (today's stack and commands)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Every pull request is checked before it can merge (Priority: P1)

As a developer I open a pull request into `develop`, and within minutes I see, gate by gate, whether
my change keeps the code formatted, type-safe, lint-clean, tested, buildable and free of known
vulnerabilities. A reviewer cannot merge it while any required gate is red.

Rushikesh opens a pull request that changes a Go service and a React screen. The pull request shows
four checks — backend, frontend, end-to-end and repository — and each check's log lists its gates
as named steps. The backend lint step reports a finding on a line he changed and nothing about
older lines he did not touch; he fixes it, pushes, and every check turns green. The
test results and coverage reports of the run are attached to it for the reviewer.

**Why this priority**: every later alignment feature rewrites large parts of the code; without
automatic gates, regressions would only be found by hand.

**Independent Test**: open a pull request with one deliberate problem per gate (an unformatted
file, a type error, a failing test, a lint finding on a changed line) and check that exactly that
gate fails and names the problem; then open a clean pull request and check that all gates pass.

**Acceptance Scenarios**:

1. **Given** a pull request into `develop`, `release` or `main`, **When** it is opened or updated,
   **Then** every quality gate of FR-002 and FR-003 runs as its own named step in one of the four
   checks, and each check reports pass or fail.
2. **Given** a pull request with an unformatted Go file, **When** the checks run, **Then** the
   backend check fails at its formatting step, which names the file; the other checks are
   unaffected.
3. **Given** a pull request that changes one line of a Go file that already had lint findings
   elsewhere, **When** the lint check runs, **Then** it reports only findings on new and changed
   lines.
4. **Given** a pull request whose frontend test fails, **When** the checks run, **Then** the
   frontend test check fails, and the run still publishes the test results and the coverage
   report.
5. **Given** today's `develop` with this feature merged, **When** the pipeline runs on it, **Then**
   every gate passes.

---

### User Story 3 - Nobody pushes straight to the shared branches (Priority: P2)

As the team I want `main`, `release` and `develop` to accept changes only through reviewed pull
requests from working branches, so every change is seen by a second person and passes the gates.

Anup tries to push a quick fix directly to `develop`; the push is refused. He pushes it to
`fix/date-label`, opens a pull request, a colleague approves it, the checks pass, and it merges.
A pull request into `develop` from a branch named `quickfix` is flagged by the branch-name check.

**Why this priority**: the rule is in the constitution and the product owner asked for it, but it
only matters once the gates of story 1 exist.

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

- A pull request that changes only documentation still runs all four checks, so required checks
  never wait for a result that will not come.
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
- **FR-002**: The pipeline MUST run these backend gates as named steps of one `backend` check: formatting
  (`gofmt` and `goimports`), `go vet`, `golangci-lint` with `backend/.golangci.yml` reporting only
  findings on new and changed lines compared with the pull request's base, all Go tests with the
  race detector and a coverage profile, and `govulncheck`.
- **FR-003**: The pipeline MUST run these frontend gates as named steps of one `frontend` check:
  formatting, type check, lint with warnings counted as failures, the generated-API-types drift
  check, unit and component tests with branch coverage reported as LCOV, and the production build;
  and the end-to-end tests against the real API with the seed data as a separate `e2e` check.
- **FR-004**: The pipeline MUST have exactly four checks — `backend`, `frontend`, `e2e` and
  `repo-checks` (the branch name of FR-011 and the dependency review of FR-017) — each able to
  block a merge. A failing step fails its check, the log names the gate, and a step that cannot run counts
  as failed.
- **FR-005**: Each run MUST publish its test results and coverage reports (Go coverage profile,
  frontend LCOV) so that a reviewer can open them from the pull request, kept for at least 30 days.
- **FR-006 – FR-009**: *Withdrawn on 2026-10-26.* The comment budget (constitution VIII) and the
  file-length limit (constitution V) stay constitutional rules that the reviewer checks (quality
  gate 14); no automated checker, baseline or local command is built (product owner).
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
- **FR-015**: The developer documentation MUST say how to run every gate locally, that the
  reviewer checks the comment budget and the file-length limit, and how to apply the branch rules
  (ruleset).
- **FR-016**: This feature MUST NOT change application behaviour: no change to the API, the
  screens, the stored data or the seed data. A gate that today's code fails is fixed with the
  smallest change that keeps behaviour, or the gap is recorded with its reason.
- **FR-017**: The pipeline MUST run the security gates the repository can run without
  organisation-level setup: a review of the dependencies a pull request adds or changes (a step of
  `repo-checks`), and the audit of the web app's production dependencies (a step of `frontend`).
  Each fails on a high or critical finding.
- **FR-018**: *Withdrawn on 2026-10-26.* The first-page JavaScript budget is deferred (FR-019).
- **FR-019**: An architecture decision record MUST record the pipeline and the branch rules, and
  the deviations the product owner agreed on 2026-10-26: static code
  scanning (SAST, gate 16) waits for SonarQube — GitHub code scanning is not used; SonarQube,
  Apiiro and Mend wait for organisation setup; secret scanning is switched on by a repository
  administrator; the 80% new-code coverage threshold is enforced once SonarQube is available; the
  first-page JavaScript budget (gate 12) waits until the web app is split per route.
- **FR-020**: No package version younger than 24 hours MAY be installed or added (constitution
  VII). The web app's npm settings MUST use npm's built-in `min-release-age` to refuse versions
  published less than one day ago, and MUST require an npm version that honours it. Go has no such
  setting: the developer documentation MUST tell reviewers to check the publication date of every
  Go module a pull request adds or upgrades.

### Key Entities

- **Quality gate**: a named step with the command it runs, inside one of the four checks that
  guard the branches and block merging.
- **Branch rule**: a protected branch with its allowed source branches and its rules in the
  ruleset.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every pull request into `develop`, `release` or `main` shows a result for every gate
  within 20 minutes of its latest push.
- **SC-002**: The pipeline passes on `develop` once this feature is merged: zero failing gates on
  today's code.
- **SC-003, SC-004**: *Withdrawn with FR-006 – FR-009.*
- **SC-005**: Once an administrator applies the branch rules, 100% of direct pushes to `main`,
  `release` and `develop` are refused, and no pull request merges without an approval from someone
  other than its author.
- **SC-006**: No pull request merges with a high or critical finding from the dependency review or
  the production dependency audit.
- **SC-007**: `npm install` in the web app never installs a package version published less than 24
  hours earlier.

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
