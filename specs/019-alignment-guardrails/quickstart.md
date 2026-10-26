# Quickstart: validating Alignment Guardrails (019)

## Prerequisites

Go 1.27 (1.27.3 is fetched automatically through `toolchain`), Node 24, `npm ci` in `frontend/`,
`npx playwright install chromium`, and the GitHub CLI for the pipeline scenarios.

## 1. Local gates (same commands as the pipeline)

```sh
cd backend
gofmt -s -l . && go run golang.org/x/tools/cmd/goimports@v0.52.0 -local mapp -l .   # prints nothing
go vet ./...
golangci-lint run --new-from-merge-base=origin/develop ./...                        # no new findings
go test -race ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...                               # no vulnerabilities
cd ../frontend
npm run format:check && npm run typecheck && npm run lint && npm run api:check
npm run test:coverage          # writes coverage/lcov.info and test-results/junit.xml
npm run build                  # includes the JavaScript budget check
npm audit --omit=dev --audit-level=high
npm run test:e2e
cd .. && go -C backend run ./cmd/sourcecheck   # exit 0
go -C backend run ./cmd/releaseage -base origin/develop   # exit 0: no version younger than 24 h
```

Expected: every command succeeds on the feature branch (SC-002).

## 2. Size and comment checks (US2)

| Do | Expect |
|---|---|
| Add a new `.go` file with 401 non-blank lines | `sourcecheck` exit 1, names the file, "limit 400" |
| Give a 320-line file an entry in `quality/exceptions.json` with a reason | passes; without it, fails |
| Add a 100-line file with 6 comment lines | fails, "budget 5"; with 5 it passes; `//nolint:…` lines do not count |
| Add one line to a file recorded in the baseline | fails; remove two lines → passes and says it can be tightened |
| Run `-tighten` after shrinking a baseline file | its count drops in `quality/baseline.json`; no count rises |

## 2b. Release age and JavaScript budget

| Do | Expect |
|---|---|
| `npm install` a package version published an hour ago | npm refuses (`min-release-age=1`) |
| Change `package-lock.json` or `go.sum` to a version younger than 24 h and run `releaseage` | exit 1, names the package, version and publication time |
| Lower `initialRouteGzipKiB` below today's size and run `npm run build` | the build fails and prints the size and the budget |

## 3. Pipeline (US1)

Open a draft pull request into `develop` with one deliberate problem per gate (an unformatted Go
file, a type error, a failing test, a lint finding on a changed line, a 401-line file), one at a
time. Expect exactly the matching check to fail and name the problem; push the fix and expect all
17 checks green, with the `backend-test` and `frontend-test` artifacts attached.

## 4. Branch rules (US3) — after an administrator applies the ruleset

```sh
gh api --method POST repos/anup-zensar/MAPP/rulesets --input .github/rulesets/protected-branches.json
```

| Do | Expect |
|---|---|
| `git push origin HEAD:develop` | refused |
| `git push --force origin …:main` | refused |
| Merge a pull request without an approval from someone else | blocked |
| Open a pull request into `develop` from a branch named `quickfix` | `branch-name` fails and lists the allowed names |

## 5. Feature branches (US4)

Run the Spec Kit specify step for a scratch feature in a throwaway clone: the branch is
`feature/NNN-short-name` and the folder `specs/NNN-short-name`.
