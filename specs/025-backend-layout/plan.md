# Implementation Plan: Backend Layout

**Branch**: `feature/025-backend-layout` | **Date**: 2026-10-27 | **Spec**: [spec.md](spec.md)

## Summary

A refactoring of `backend/` with behaviour, SQL and the API contract unchanged: every Go file within 300
non-blank lines (FR-001), a small consumer-defined `Store` per service and constructors within four
parameters (FR-002), then, as far as it goes cleanly, fakes instead of the JSON stand-in, deleting the JSON
store and singular package names (FR-005, FR-006).

## Technical Context

Go (pinned in `go.mod`), Gin, pgx/v5, sqlc, goose; tests with `testing`, `httptest`, `go-cmp`,
`libopenapi-validator`; PostgreSQL 18 for the scope-isolation and end-to-end tests. No new dependency.

## Constitution Check

- Principle V (file length, metrics): the goal of the feature. Moved code counts as new for CI's lint, so
  its old findings are fixed here: `cyclop` 15+, 7+ parameters and `funlen` above 60 are never suppressed.
- Principle VI: no test is added or removed; each keeps its name, its `t.Parallel` and its requirement ID.
- Principle VIII: feature 025 is exempt from the whole-file comment rule; new comment lines keep the 5%
  budget and say why.
- Governance → Alignment: the packages keep their stack; only their structure changes.

## The interface convention

The service declares the `Store` it needs in `store.go`, with only the methods its service code calls
(GO_STANDARDS §3). The concrete stores (`*PostgresStore`, and the JSON stand-in `JSONStore` while it
exists) keep every method and satisfy it implicitly. Every other consumer declares its own interface:

- `internal/stores/<area>.go` declares `<X>Store` — the area's `Store` plus the methods only the wiring,
  the seed and other areas' adapters call. `stores.Set` holds these, so `internal/app` and `seed` compile
  against the stores, not against a service's needs.
- An area that reads another area's records declares a small interface of its own (`auth.People`,
  `approvals.People`, `activities.approvalWorkflows`) instead of naming the other area's `Store`.

Worked example (plans):

```go
// internal/plans/store.go — what the plans service calls, nothing more.
type Store interface {
	Get(ctx context.Context, id string) (Plan, bool, error)
	ListInScope(ctx context.Context, scope authz.Scope) ([]Plan, error)
	GetInScope(ctx context.Context, scope authz.Scope, id string) (Plan, bool, error)
	PlansOfYear(ctx context.Context, year int) ([]Plan, error)
	ListByIDs(ctx context.Context, ids []string) ([]Plan, error)
	Put(ctx context.Context, p Plan) error
	Delete(ctx context.Context, id string) error
}

// internal/stores/plans.go — the plans store as the wiring and the seed use it.
type PlanStore interface {
	plans.Store
	List(ctx context.Context) ([]plans.Plan, error) // the seed and the All work adapter read every plan
}

// internal/app/areas.go
w.plans = plans.NewService(w.tools, w.set.Plans, w.settings, w.people)
```

Constructors: `history.Tools` bundles the transaction, the ID generator, the activity log and the clock
that every service changes records with, so `NewService(tools history.Tools, store Store, …)` stays within
four parameters; an area without them (auth) groups its options in a struct.

## The split rules

1. One primary concept per file, named after it: a use case (`copy.go`, `rounds_decide.go`), a group of
   domain types (`model_rounds.go`), the handlers of one section (`handler_rates.go`), the tests of one
   behaviour (`status_test.go`), a fixture (`fixture_test.go`, `fakes_test.go`).
2. Move code verbatim: no renames, no logic change; `git mv` keeps the history of the largest part.
3. Prefer moving clean code and leaving the functions with old lint findings where they are; fix what
   still surfaces (constants, shadowing, helpers), never with a suppression beyond "needs justification".
4. Aim for about 280 non-blank lines, so a small later change doesn't cross 300.

## Helper areas

The lead keeps `internal/app`, `internal/stores`, `internal/platform/*`, the cross-area adapters and the
merges. Helpers work in their own worktree and branch `025-<area>` from the lead's branch, at most about
six at a time:

| Helper | Packages |
|---|---|
| settings-services, settings-rules, settings-handler | `settings` (by file group) |
| approvals-rounds, approvals-workflows | `approvals` (by file group) |
| allwork-views, allwork-home | `allwork`, `home` |
| plans | `plans`, `rollup` |
| campaigns | `campaigns` |
| activities, lineitems | `activities`, `lineitems` |
| people | `people` |
| access | `access`, `auth`, `samples` |
| history | `history` |
| finance | `financials`, `budgets` |
| seed-store | `seed`, `internal/platform/store` |

## Verification

Gates (FR-007): `gofmt -s`, `goimports -local mapp`, `go vet ./...`, golangci-lint on new lines,
`sqlc generate` (no diff) and `sqlc vet`, `go test -race ./...` with `TEST_DATABASE_URL`,
`go test -short ./...`, govulncheck, the e2e suite on PostgreSQL and `pnpm run api:check`.
