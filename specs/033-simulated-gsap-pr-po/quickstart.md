# Quickstart: GSAP PR Number and Purchase Order from a Simulated GSAP (033)

Needs 030, 031 and 032 merged. Rules in [data-model.md](data-model.md) §3; operation in
[contracts/openapi.yaml](contracts/openapi.yaml).

## Prerequisites and checks

```sh
export PATH="$HOME/.local/npm/bin:$HOME/go/bin:$PATH"
export DATABASE_URL='postgres://mapp:mapp@127.0.0.1:5432/mapp_f033?sslmode=disable'   # never "mapp"
cd backend && sqlc generate && sqlc vet && go vet ./... && golangci-lint run --new ./...
TEST_DATABASE_URL='postgres://mapp:mapp@127.0.0.1:5432/postgres?sslmode=disable' go test -race ./...
govulncheck ./...
go run ./cmd/seed && APP_PORT=18133 go run ./cmd/api
```

## API walkthrough

Follow 032's quickstart steps 1–7 on port 18133 to get an approved, routed PR (here PR-1002 on a new line item
"Gift boxes", 10,000.00 USD). Then:

```sh
API=http://127.0.0.1:18133/api/v1
V=$(curl -s -b lea.santos.jar $API/purchase-requests/PR-1002 | jq .version)
curl -s -b lea.santos.jar $API/purchase-requests/PR-1002 | jq .access.gsapOrder      # available: true
curl -s -b lea.santos.jar -H 'Content-Type: application/json' -X POST \
  -d "{\"version\":$V}" $API/simulated-gsap/purchase-requests/PR-1002/purchase-order | jq '{gsapPrNumber, gsapPrDate, gsapSource}, .lines[0]'
```

Expected: `gsapPrNumber` "SIM-0010001001", `gsapSource` "simulatedGsap"; line `poNumber` "SIM-4500001001",
`poLine` "10", `poLineStatus` "Released", `poValue` 10000.00 USD, `gsapPending` false.

```sh
curl -s -b lea.santos.jar "$API/financials/records/<the line item id>" | jq .figures          # committed 10000.00
curl -s -b lea.santos.jar "$API/financials/imports" | jq '.items[0] | {source, feedName, fileName}'
curl -s -b lea.santos.jar "$API/purchase-requests?waitingForGsap=true" | jq .total            # PR-1002 gone
# second run, same version re-read → 409 not_allowed "The simulated GSAP already created the PO …"
curl -s -b priya.raman.jar -X POST -H 'Content-Type: application/json' -d "{\"version\":$V}" \
  $API/simulated-gsap/purchase-requests/PR-1002/purchase-order | jq .error.code                 # forbidden
```

Top-up: as Priya raise the line to 12,000.00 USD with `then: "submitChanges"`, approve as Daniel, run again as
Lea → same numbers, `poValue` 12000.00, Committed 12,000.00, the transaction updated (import counts "updated 1").

## Web and end-to-end

As Lea: Purchase requests › Waiting for GSAP › PR-1002 › Create GSAP PR and PO (simulated) › Create; the
GSAP section and the PO columns show "Simulated GSAP". Then the line item's Financials shows Committed.

```sh
cd frontend && pnpm run api:types && pnpm run typecheck && pnpm run lint && pnpm run test && pnpm run build
E2E_DATABASE_URL=postgres://mapp:mapp@127.0.0.1:5432/mapp_e2e_033?sslmode=disable \
  E2E_API_PORT=18133 E2E_WEB_PORT=14133 pnpm run test:e2e -- e2e/pr-to-po.spec.ts
```
