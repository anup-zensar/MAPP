# Contract: `releaseage`

## Command

```sh
go -C backend run ./cmd/releaseage -base REV [-root DIR] [-min-age 24h]
```

| Flag | Default | Meaning |
|---|---|---|
| `-base` | required | The Git revision to compare with (the pull request's base, or the previous commit of a push) |
| `-root` | the Git repository's top level | The repository to check |
| `-min-age` | `24h` | The youngest a version may be |

## What is checked

- **npm**: `frontend/package-lock.json` (lockfile version 3). Each entry under `packages` whose key
  contains `node_modules/` and that has a `version` (links excluded) is a name and version; the
  name is the part after the last `node_modules/`. A pair present now and absent at `-base` is
  checked.
- **Go**: `backend/go.sum`. Each line `module version hash` gives a module and version (`/go.mod`
  stripped). A pair present now and absent at `-base` is checked.
- A file missing at `-base` counts as empty.

## Publication time

| Ecosystem | Request | Field |
|---|---|---|
| npm | `GET https://registry.npmjs.org/<name>` (`/` of a scoped name sent as `%2F`) | `time[<version>]` |
| Go | `GET https://proxy.golang.org/<escaped module>/@v/<escaped version>.info` (each upper-case letter `X` sent as `!x`) | `Time` |

Every request has a 15-second timeout. A failed request or a missing time is a usage error.

## Output and exit codes

```text
npm @vitest/coverage-v8@5.0.6: published 2026-10-26T01:00:00Z, 7h ago; minimum 24h
2 versions checked; 1 too young
```

| Code | Meaning |
|---|---|
| 0 | Every added version is at least `-min-age` old (or none were added) |
| 1 | At least one version is younger |
| 2 | Usage error, Git error, unreadable lockfile, or a registry that does not answer |
