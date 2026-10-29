# Quickstart: Bulk Actions and Excel Import of Plans (012)

## Checks

```sh
cd backend
golangci-lint fmt ./... && golangci-lint run ./...
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate && go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 vet
TEST_DATABASE_URL='postgres://mapp:mapp@127.0.0.1:5432/postgres?sslmode=disable' go test -race ./...
go test -run XXX -fuzz FuzzReadXLSX -fuzztime 30s ./internal/platform/sheet
cd ../frontend && pnpm run check && pnpm run test:e2e -- bulk workbook
```

## API walkthrough (signed in as Priya Raman)

```text
POST /api/v1/bulk-actions/preview {"action":"changeOwner","recordIds":["A-1001","A-1002"],"ownerId":"U-1003"}
  → 200 {"preview":true,"id":null,"counts":{"done":2,…},"note":"Bulk change of 2 activities"}
POST /api/v1/bulk-actions        (same body) → 201 {"id":"BLK-1001",…}
GET  /api/v1/activities/A-1001/history → the change, details "… Bulk change of 2 activities (BLK-1001)"
POST /api/v1/bulk-actions {"action":"delete","recordIds":["A-1001"]} → 201 {"recycleBinEntries":[…],"undoAvailableUntil":…}
POST /api/v1/bulk-actions/BLK-1002/undo → 200 {"restored":[…]}

GET  /api/v1/plans/P-1001/import-template → .xlsx (sheets Campaigns … Instructions)
GET  /api/v1/plans/P-1001/export          → the same layout with the plan's rows
POST /api/v1/plans/P-1001/imports/check {"fileName":"plan.xlsx","content":"<base64>"} → 200 ImportCheck
POST /api/v1/plans/P-1001/imports       (same body) → 201 ImportDetail {"import":{"id":"IMP-1001",…}}
POST /api/v1/imports/IMP-1001/undo → 200 ImportDetail (status undone)
```

## Web

All work → tick rows → the bar → **Change owner** … ; plan page ⋯ → **Import from Excel** → the three steps →
**Import** → results → **Undo import**.
