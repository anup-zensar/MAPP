# Data Model: Alignment Guardrails (019)

No application data changes. This feature adds three kinds of repository records, kept as committed
files so every change to them is reviewed in a pull request.

## Quality gate — `.github/workflows/ci.yml`

| Field | Value |
|---|---|
| Name | One of the four checks (`backend`, `frontend`, `e2e`, `repo-checks`) and its named step in [contracts/ci-checks.md](contracts/ci-checks.md) |
| Command | The command the job runs (same as the local command) |
| Branches | Pull requests into, and pushes to, `main`, `release`, `develop` |
| Blocking | Yes, through the ruleset's required status checks |

## Branch rule — `.github/rulesets/protected-branches.json`

| Field | Value |
|---|---|
| Branches | `main`, `release`, `develop` |
| Bypass | None (administrators included) |
| Rules | No deletion; no force push; pull request with 1 approval, stale approvals dismissed, last push approved by someone else; the four checks required |
| Allowed sources (checked by `branch-name`) | into `develop`: `feature/NNN-…`, `docs/…`, `fix/…`, module branches; into `release`: `develop`, `fix/…`; into `main`: `release` |

## Release age — `frontend/.npmrc`

| Setting | Value | Rule |
|---|---|---|
| `min-release-age` | `1` (day) | npm installs only versions published at least 24 hours ago when it chooses a version |
| `engine-strict` | `true` | npm refuses to run when `engines.npm` (`>=11.10.0`, the first npm with `min-release-age`) does not match |

Go modules have no such setting; reviewers check the publication date of every module a pull
request adds or upgrades.
