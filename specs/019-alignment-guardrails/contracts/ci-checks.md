# Contract: CI checks

Four checks, each a job in `.github/workflows/ci.yml`. The ruleset requires them by name, so a name
changes only together with `.github/rulesets/protected-branches.json`. Every check runs on pull
requests into, and pushes to, `main`, `release` and `develop`; steps marked *PR* run on pull
requests only. Each gate is a named step, so a failed check's log shows which gate failed.

| Check | Step (gate) | Command (from the directory shown) |
|---|---|---|
| `backend` | Format (1) | `backend/`: `gofmt -s -l .` and `goimports -local mapp -l .` print nothing |
| | Vet (2) | `backend/`: `go vet ./...` |
| | Lint (2) | `backend/`: `golangci-lint run` (v2.15.0), new and changed lines only |
| | Test (5) | `backend/`: `go test -race -covermode=atomic -coverprofile=coverage.out ./...` → artifact `backend-test` (log, `coverage.out`) |
| | Vulnerabilities (6) | `backend/`: `govulncheck ./...` (v1.9.0) |
| `frontend` | Format (7) | `frontend/`: `npm run format:check` |
| | Type check (8) | `frontend/`: `npm run typecheck` |
| | Lint (9) | `frontend/`: `npm run lint` (`--max-warnings 0`) |
| | API types (10) | `frontend/`: `npm run api:check` |
| | Test (11) | `frontend/`: `npm run test:coverage` → artifact `frontend-test` (`coverage/lcov.info`, `test-results/junit.xml`) |
| | Build (12) | `frontend/`: `npm run build` |
| | Audit (17) | `frontend/`: `npm audit --omit=dev --audit-level=high` |
| `e2e` | End-to-end (13) | `frontend/`: `npm run test:e2e` (Chromium) → artifact `e2e` on failure (`test-results/`) |
| `repo-checks` | Branch name (workflow) *PR* | rules of FR-011 on `head_ref` / `base_ref` |
| | Dependency review (17) *PR* | `actions/dependency-review-action@v5`, fails on high or critical |

Gate 14 (comment budget and file length) is checked in review, not by a step. Artifacts are kept
for 30 days (FR-005). Gates 3 and 4 (sqlc, goose) join `backend` with the
PostgreSQL feature. Gate 12's JavaScript budget, gate 16 (static code scanning) and gates 15, 17 and
18 in part are deferred (ADR-0010).
