# Quickstart: Purchase Requests on MAPP's Side (032)

Validation guide; contracts in [contracts/openapi.yaml](contracts/openapi.yaml), rules in
[data-model.md](data-model.md) §3. Needs features 030 and 031 merged.

## Prerequisites

```sh
export PATH="$HOME/.local/npm/bin:$HOME/go/bin:$PATH"
# PostgreSQL 18 on 127.0.0.1:5432, role mapp/mapp. Never use the database "mapp" (the user's demo data).
export DATABASE_URL='postgres://mapp:mapp@127.0.0.1:5432/mapp_f032?sslmode=disable'
```

## Backend checks

```sh
cd backend
sqlc generate && sqlc vet
gofmt -l ./internal/purchaserequest ./internal/app
golangci-lint run --new ./...
go vet ./...
TEST_DATABASE_URL='postgres://mapp:mapp@127.0.0.1:5432/postgres?sslmode=disable' go test -race ./...
govulncheck ./...
```

Expected: all pass; `internal/app`'s route-table and contract tests list no operation without a rule and no
unserved operation outside `notYetServed`.

## Run the API with the demo data (ports of this feature)

```sh
cd backend
createdb -h 127.0.0.1 -U mapp mapp_f032 2>/dev/null || true
go run ./cmd/seed
APP_PORT=18132 go run ./cmd/api
```

## API walkthrough (curl; cookies in a jar per person)

```sh
API=http://127.0.0.1:18132/api/v1
signin() { curl -s -c "$1.jar" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$1@shell.com\",\"password\":\"mapp\"}" $API/auth/sign-in >/dev/null; }
signin priya.raman; signin daniel.okoro; signin lea.santos; signin alex.morgan
```

1. **Defaults** — `curl -b priya.raman.jar "$API/purchase-requests/new-defaults?lineItemId=L-1006"` →
   name "PR for Stand build", line 9200.00 EUR, overallLimit 9200.00, hxKey "HX1006".
2. **Create** — POST `$API/purchase-requests` with the defaults, the two seeded PR workflows, a delivery date,
   attributes Vendor, Short text, PR type, Material or service, Invoice approver name, Email group DE_LUBES,
   `moaApproverId: "U-1003"` (Daniel), `then: "save"` → 201, status `created`.
   `curl -b priya.raman.jar $API/line-items/L-1006/purchase-requests` → requested 9200.00 EUR, available 0.00.
3. **Gate** — `curl -b priya.raman.jar $API/purchase-requests/PR-1002/approval/submission` → `ready:false`,
   a problem "Line item L-1006 'Stand build' has no SWBS code yet (Ready to send to GSAP)."
4. **SWBS code (030)** — `curl -b alex.morgan.jar -X POST $API/simulated-gsap/swbs-runs -d '{}'` → L-1006
   receives its code; the submission check is now `ready:true`.
5. **Submit** — POST `$API/purchase-requests/PR-1002/approval/submit` `{"version":<v>}` → status `submitted`.
6. **Approve** — Daniel: `curl -b daniel.okoro.jar $API/purchase-requests/PR-1002/approval` (read `roundId`,
   level), POST `…/approval/approve` `{"roundId":"AR-…","level":1}` → status `approved`; GET the PR → one
   routing `original` to `["DE_LUBES"]`, the line's `sentOn` today, `waitingForGsap:true`.
7. **Requisitioner** — Lea: `curl -b lea.santos.jar "$API/purchase-requests?waitingForGsap=true"` lists PR-1002;
   `curl -b lea.santos.jar -o form.xlsx "$API/purchase-requests/PR-1002/routings/PRR-1001/form?format=xlsx"`;
   `…?format=csv` shows the row with HX1006. Lea's PUT of the PR answers 403.
8. **Protection** — Priya: `curl -b priya.raman.jar -X DELETE $API/line-items/L-1006` → 409 `in_use` naming
   PR-1002; DELETE of PR-1002 → 409 "Approved purchase requests cannot be deleted — close them instead."
9. **Close** — GET `…/close-preview` (released 9200.00 EUR, no invoices), POST `…/close` `{"version":<v>}` →
   `closed`; L-1006 available 9200.00 EUR again; a routing `close`.
10. **Switched off** — Alex: PUT `$API/purchase-request-configuration` `{"version":1,"purchaseSupport":false,
    "linesOnApprovedPurchaseRequests":"withModificationApproval"}`; Priya's create → 409 with "Purchase requests
    are switched off for your organisation."; switch it back on.

## Web walkthrough

```sh
cd frontend && pnpm install --frozen-lockfile --offline && pnpm run dev   # proxy to the API above
```

As Priya: Plans › FY2026 Lubricants Europe › Billboard winter wave › Stand build › Purchase requests › New
purchase request; save; Submit (refused, then accepted after the SWBS run); as Daniel: Approvals › the PR ›
Approve; as Lea: Purchase requests › Waiting for GSAP › open › Download PR form. Check every screen at 320 and
1280 px, light and dark.

## Frontend checks and end-to-end

```sh
cd frontend
pnpm run api:types && pnpm run typecheck && pnpm run lint && pnpm run format:check && pnpm run test && pnpm run build
E2E_DATABASE_URL=postgres://mapp:mapp@127.0.0.1:5432/mapp_e2e_032?sslmode=disable \
  E2E_API_PORT=18132 E2E_WEB_PORT=14132 pnpm run test:e2e -- e2e/purchase-requests.spec.ts
# frontend lane: E2E_API_PORT=18142 E2E_WEB_PORT=14142, database mapp_e2e_032w
```

Expected: the journeys of US1–US3 pass with no axe violations.
