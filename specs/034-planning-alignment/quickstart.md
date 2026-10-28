# Quickstart: Planning Rules Aligned with Shell's Process

How to prove feature 034 on a developer's machine. Contract: [contracts/openapi.yaml](contracts/openapi.yaml);
rules: [data-model.md](data-model.md).

## Prerequisites

- PostgreSQL 18 on 127.0.0.1:5432 (role `mapp`, password `mapp`, CREATEDB).
- `export PATH="$HOME/.local/npm/bin:$HOME/go/bin:$PATH"`
- Own databases: `mapp_f034` (app), `mapp_e2e_034` (end-to-end). Ports 18134 (API) and 14134 (web).

## Automated checks

```sh
cd backend
golangci-lint run --new ./...
sqlc generate && sqlc vet                       # no SQL change: no diff expected
go vet ./...
TEST_DATABASE_URL='postgres://mapp:mapp@127.0.0.1:5432/postgres?sslmode=disable' go test -race ./...
govulncheck ./...

cd ../frontend
pnpm run api:check && pnpm run typecheck && pnpm run lint && pnpm run format:check && pnpm run test
pnpm run build
E2E_DATABASE_URL=postgres://mapp:mapp@127.0.0.1:5432/mapp_e2e_034?sslmode=disable \
  E2E_API_PORT=18134 E2E_WEB_PORT=14134 pnpm run test:e2e -- e2e/planning-alignment.spec.ts
```

## Run the app on the demo data

```sh
cd backend
DATABASE_URL='postgres://mapp:mapp@127.0.0.1:5432/mapp_f034?sslmode=disable' go run ./cmd/seed -reset
DATABASE_URL='postgres://mapp:mapp@127.0.0.1:5432/mapp_f034?sslmode=disable' APP_PORT=18134 go run ./cmd/api
```

(`APP_PORT` and `DATABASE_URL` are read by `internal/app/config.go`; demo sign-in is on in local mode.)

## Scenarios to check through the API

Sign in as Priya Raman (`priya.raman@shell.com`, password `mapp`) and keep the session cookie.

1. **Line item on an approved activity (US1, FR-001).** `GET /api/v1/activities/A-1002` → status
   `approved`. `POST /api/v1/activities/A-1002/line-items` with name, `"currency":"USD"`,
   `"plannedCost":"2000.00"` and an even spread → 201 without any confirmation; `GET
   /api/v1/activities/A-1002` → still `approved`; `GET /api/v1/activities/A-1002/approval` → no new history
   entry. `LineItemActivity.restartsApproval` in the answer is `false`.
2. **Delete and Undo (FR-001, FR-002).** `DELETE /api/v1/line-items/<new id>` → 200 with the bin entry;
   activity still `approved`; `POST /api/v1/recycle-bin/<entry>/undo` → line item back, activity still
   `approved`.
3. **The activity's own planned cost still restarts (FR-003).** `PATCH /api/v1/activities/A-1002` with a new
   planned cost and `restartApproval:false` → 400 `restartApproval`; with `true` → `created`.
4. **Shell's lists (US2).** `GET /api/v1/shell-attribute-forms/lineItem` → GL code with 13 options from
   "6140900 - PMTDR Credit", no parent; Spend type 25; Spend category 19; Contracted spend 5. `GET
   /api/v1/shell-attribute-forms/activity` → Zero-based cost hierarchy with 10 options.
5. **Plan years (US3).** `GET /api/v1/plan-defaults` → `years` = current … current + 10. `POST /api/v1/plans`
   with last year → 400 "Choose a year from <current> to <current + 10>."

## In the browser

Open http://127.0.0.1:14134 (web dev server proxying to the API on 18134), sign in as Priya, open "Social
always-on": **Add line item** saves without the "This activity will need approval again" dialog and the
activity stays Approved; **More › Delete** on a line item does not mention approval; the line-item form's
GL code offers Shell's 13 codes whatever the spend type; **New plan** offers 11 years from the current one.
