# Data Model: Alignment Guardrails (019)

No application data changes. This feature adds six kinds of repository records, kept as committed
files so every change to them is reviewed in a pull request.

## Baseline entry — `quality/baseline.json`

A legacy file that exceeded a limit when this feature shipped (FR-008).

| Field | Type | Rule |
|---|---|---|
| `path` | string | Repository-relative, forward slashes; unique per `metric` |
| `metric` | `"lines"` or `"comments"` | Which limit the file exceeds |
| `count` | integer | Non-blank lines, or comment lines, when recorded |

- A file passes its metric while its count ≤ `count`; it fails when the count rises.
- `-tighten` lowers `count` to the current value and removes entries now within the limit; it never
  raises a count or adds an entry.
- `-seed` creates the file once, from the current tree; it refuses when the file exists.
- An entry whose file no longer exists is reported as removable and does not fail the check.
- The array is sorted by `path`, then `metric`.

## Size exception — `quality/exceptions.json`

A file allowed into the "Needs justification" band (301–400 non-blank lines) (FR-007).

| Field | Type | Rule |
|---|---|---|
| `path` | string | Repository-relative; unique |
| `lines` | integer | 301–400; the file passes while its count ≤ `lines` |
| `reason` | string | Non-empty; why the file cannot be split |

- An exception never allows 401 or more lines.
- An exception for a file at 300 lines or fewer is reported as removable.

## Quality gate — `.github/workflows/ci.yml`

| Field | Value |
|---|---|
| Name | One of the 17 job names in [contracts/ci-checks.md](contracts/ci-checks.md) |
| Command | The command the job runs (same as the local command) |
| Branches | Pull requests into, and pushes to, `main`, `release`, `develop` |
| Blocking | Yes, through the ruleset's required status checks |

## Branch rule — `.github/rulesets/protected-branches.json`

| Field | Value |
|---|---|
| Branches | `main`, `release`, `develop` |
| Bypass | None (administrators included) |
| Rules | No deletion; no force push; pull request with 1 approval, stale approvals dismissed, last push approved by someone else; the 17 checks required |
| Allowed sources (checked by `branch-name`) | into `develop`: `feature/NNN-…`, `docs/…`, `fix/…`, module branches; into `release`: `develop`, `fix/…`; into `main`: `release` |

## JavaScript budget — `frontend/bundle-budget.json`

| Field | Type | Rule |
|---|---|---|
| `initialRouteGzipKiB` | integer | The gzipped KiB the scripts and module preloads referenced by `dist/index.html` may reach; `npm run build` fails above it. Starts at 400 (today 376) and is lowered, never raised without a recorded reason, as the web app is split per route |

## Release age — `frontend/.npmrc`

| Setting | Value | Rule |
|---|---|---|
| `min-release-age` | `1` (day) | npm installs only versions published at least 24 hours ago; the `dependency-age` check applies the same rule to pull requests, for npm and Go ([contracts/releaseage.md](contracts/releaseage.md)) |
