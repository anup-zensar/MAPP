# Quality gates

Every pull request into `develop`, `release` or `main` runs three checks
(`.github/workflows/ci.yml`, [ADR-0010](../architecture/adr/0010-quality-gates-and-branch-rules.md)).
Run the same commands before you push.

| Check | Run locally |
|---|---|
| `backend` | in `backend/`: `gofmt -s -l .` and `go run golang.org/x/tools/cmd/goimports@v0.52.0 -local mapp -l .` (print nothing), `go vet ./...`, `golangci-lint run --new-from-merge-base=origin/develop ./...`, `go test -race ./...`, `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...` |
| `frontend` | in `frontend/`: `npm run format:check`, `npm run typecheck`, `npm run lint`, `npm run api:check`, `npm run test:coverage`, `npm run build`, `npm audit --omit=dev --audit-level=high` |
| `e2e` | in `frontend/`: `npm run test:e2e` |

## Branches

Nobody pushes directly to `main`, `release` or `develop`. Open a pull request from:

- into `develop`: `feature/NNN-short-name` (Spec Kit creates it), `docs/…`, `fix/…`, or the module
  branch `plan-module`;
- into `release`: `develop` or `fix/…`;
- into `main`: `release`.

A pull request needs one approval from someone other than its last pusher, and the three checks
green.

## What the reviewer checks

- **Comment budget** (constitution VIII): at most 5% of a file's non-blank lines hold a comment,
  rounded up; tool directives do not count.
- **File length** (constitution V): at most 300 non-blank lines; 301–400 needs a reason in the pull
  request; 401 or more is never merged.
- **Release age of Go modules** (constitution VII): a module version added or upgraded was published
  more than 24 hours ago — see `https://proxy.golang.org/<module>/@v/<version>.info`. npm enforces
  this itself (`frontend/.npmrc`, npm 11.10 or newer).

## For a repository administrator

Apply the branch rules once, and switch on secret scanning (Settings → Code security):

```sh
gh api --method POST repos/anup-zensar/MAPP/rulesets --input .github/rulesets/protected-branches.json
```
