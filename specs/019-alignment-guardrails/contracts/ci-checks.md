# Contract: CI checks

The ruleset requires these checks by name, so a name changes only together with
`.github/rulesets/protected-branches.json`. Every check runs on pull requests into, and pushes to,
`main`, `release` and `develop`; `branch-name` runs on pull requests only (a skipped check counts as
passed on pushes).

| Check | Constitution gate | Command (from the directory shown) | Publishes |
|---|---|---|---|
| `backend-format` | 1 | `backend/`: `gofmt -s -l .` and `goimports -local mapp -l .` print nothing | — |
| `backend-vet` | 2 | `backend/`: `go vet ./...` | — |
| `backend-lint` | 2 | `backend/`: `golangci-lint run` (v2.15.0), new and changed lines only | — |
| `backend-test` | 5 | `backend/`: `go test -race -covermode=atomic -coverprofile=coverage.out ./...` | `backend-test` artifact: test log, `coverage.out` |
| `backend-vulnerabilities` | 6 | `backend/`: `govulncheck ./...` (v1.9.0) | — |
| `frontend-format` | 7 | `frontend/`: `npm run format:check` | — |
| `frontend-typecheck` | 8 | `frontend/`: `npm run typecheck` | — |
| `frontend-lint` | 9 | `frontend/`: `npm run lint` (`--max-warnings 0`) | — |
| `frontend-api-types` | 10 | `frontend/`: `npm run api:check` | — |
| `frontend-test` | 11 | `frontend/`: `npm run test:coverage` | `frontend-test` artifact: `coverage/lcov.info`, `test-results/junit.xml` |
| `frontend-build` | 12 | `frontend/`: `npm run build` | — |
| `e2e` | 13 | `frontend/`: `npm run test:e2e` (Chromium) | `e2e` artifact on failure: `test-results/` (traces) |
| `source-limits` | 14 | repository root: `go -C backend run ./cmd/sourcecheck` | — |
| `branch-name` | Workflow → no direct pushes | rules of FR-011 on `head_ref` / `base_ref` | — |

Artifacts are kept for 30 days (FR-005). Gates 3 and 4 (sqlc, goose) join `backend-test`'s job
list with the PostgreSQL feature; gates 15–18 need organisation setup (research R11).
