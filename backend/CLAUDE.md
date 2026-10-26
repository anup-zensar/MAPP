# Backend (Go)

The constitution, `.specify/memory/constitution.md` at the repository root, governs all work here;
its *Backend (Go) Standards* section (B1–B9) applies to every file in this directory. The guide
below explains those rules with examples. Follow it when writing or reviewing Go code.

@docs/GO_STANDARDS.md

## Before you finish a change

Run from `backend/` and fix every finding:

```sh
golangci-lint fmt ./...
golangci-lint run ./...
sqlc generate && sqlc vet        # when SQL or migrations changed
go test -race ./...              # scope-isolation tests need PostgreSQL at TEST_DATABASE_URL; -short skips them
govulncheck ./...
```

Until a package is aligned to constitution 2.x (Governance → Alignment) it keeps today's stack, and
`go vet ./...` and `go test ./...` are its checks; don't migrate it as a side effect of another
change.

- Never edit generated code in `internal/platform/database/sqlcdb`; change the SQL and regenerate.
- Never edit a merged migration; add a new one.
- Keep every file within the comment budget: at most 5% of its non-blank lines (Principle VIII).
- Suppress a linter only with `//nolint:<linter> // <reason>`, and only for a value in the
  constitution's "Needs justification" band. Never suppress `cyclop`.
- Report fan-out and expression-term exceptions in the pull request description.
