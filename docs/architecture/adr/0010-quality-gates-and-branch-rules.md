# ADR-0010: Quality gates and branch rules

**Status:** Accepted · **Date:** 2026-10-26

## Context
Constitution 3.x names quality gates that every pull request must pass and requires that nobody
pushes directly to `main`, `release` or `develop`. The repository had no pipeline and no branch
rules. The product owner asked for few CI jobs, no tooling beyond what the gates need, and agreed
on 2026-10-26 to the deviations listed below (constitution Governance → Exceptions). Feature 019
implements this decision.

## Decision
- **One GitHub Actions workflow** (`.github/workflows/ci.yml`) with three checks, each gate a named
  step: `backend` (format, vet, lint on new and changed lines, race-enabled tests with coverage,
  `govulncheck`), `frontend` (format, type check, lint with zero warnings, API-types drift, tests
  with coverage, build, `npm audit`), `e2e` (Playwright against the real API). Dependency
  vulnerabilities are covered by `npm audit` and `govulncheck`; branch names are a convention
  (Spec Kit creates `feature/NNN-short-name`), not a check.
- **Branch rules** as a repository ruleset (`.github/rulesets/protected-branches.json`) on `main`,
  `release` and `develop`: no bypass, no deletion, no force push, pull requests only with one
  approval from someone other than the last pusher, stale approvals dismissed, the three checks
  required. A repository administrator applies it.
- **Release age**: npm's built-in `min-release-age=1` with `engine-strict` and `engines.npm
  >=11.10.0` in `frontend/`.

## Deviations agreed by the product owner (2026-10-26)
| Constitution | Deviation | Until |
|---|---|---|
| Gate 16, static code scanning | Not run; GitHub code scanning is not used | SonarQube is set up |
| Gates 15, 17, 18 (Apiiro, Mend, SonarQube) | Not run | Organisation setup |
| Gate 15, secret scanning | Not under the repository's control here | An administrator switches it on |
| Gates 5 and 11, 80% new-code coverage | Reports published; no threshold | SonarQube is set up |
| Gate 12, first-page JavaScript budget | Not checked | The web app is split per route |
| Gate 14, comment budget and file length | Checked by the reviewer, not by a tool | — (permanent) |
| VII, release age of Go modules | Checked by the reviewer; Go has no built-in setting | — (permanent) |

## Consequences
- Every pull request shows three checks; a failed check's log names the gate that failed.
- Code written before 2.0.0 keeps its lint findings until its package is aligned; only new and
  changed lines are linted.
- Follow-ups: an approved planning board, software asset registration and an export classification
  (ECCN) for the repository; pinning actions by commit once code scanning exists.
