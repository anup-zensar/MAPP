# Quickstart: SWBS Codes from a Simulated GSAP

Ports and database of this feature's own run: API 18130, web 14130, database `mapp_f030`.

## Start

```sh
export PATH="$HOME/.local/npm/bin:$HOME/go/bin:$PATH"
cd backend
createdb -h 127.0.0.1 -U mapp mapp_f030            # once
export DATABASE_URL='postgres://mapp:mapp@127.0.0.1:5432/mapp_f030?sslmode=disable'
go run ./cmd/seed
APP_PORT=18130 APP_ENV=local go run ./cmd/api
```

## Try it through the API

```sh
# sign in as Alex Morgan (Administrator: holds Import GSAP extracts)
curl -s -c /tmp/f030.jar -H 'Content-Type: application/json' \
  -d '{"email":"alex.morgan@shell.com","password":"mapp"}' http://127.0.0.1:18130/api/v1/auth/sign-in
# runs and the ready count: SWR-1001 (demo data, Venue hire SIM-DE03-0001), readyToSend 1
curl -s -b /tmp/f030.jar http://127.0.0.1:18130/api/v1/simulated-gsap/swbs-runs
# send everything ready: Stand build receives SIM-GB01-0001
curl -s -b /tmp/f030.jar -H 'Content-Type: application/json' -d '{}' \
  http://127.0.0.1:18130/api/v1/simulated-gsap/swbs-runs
# the line item now shows its code, dates and HX key
curl -s -b /tmp/f030.jar http://127.0.0.1:18130/api/v1/line-items/L-1006
```

## Try it in the browser

1. Sign in as Daniel Okoro, open Approvals and approve "Trade magazine ads": Media booking (L-1005) becomes
   Ready to send to GSAP.
2. Sign in as Alex Morgan, open Financials: the Simulated GSAP panel says 2 line items are ready. Send,
   confirm, and read the codes `SIM-GB01-0001` and `SIM-NL02-0001`.
3. Open Media booking: SWBS code received, the code, requested and received times, "Simulated GSAP", HX key
   HX1005 with Copy. Edit it: "Request SWBS code from GSAP" is locked on, with the reason.
4. Open Venue hire: `SIM-DE03-0001` from the demo data's run.

## Checks

```sh
cd backend && golangci-lint run --new ./... && sqlc generate && sqlc vet && go vet ./... \
  && TEST_DATABASE_URL='postgres://mapp:mapp@127.0.0.1:5432/postgres?sslmode=disable' go test -race ./... \
  && govulncheck ./...
cd ../frontend && pnpm run api:check && pnpm run typecheck && pnpm run lint && pnpm run format:check \
  && pnpm run test && pnpm run build
E2E_DATABASE_URL=postgres://mapp:mapp@127.0.0.1:5432/mapp_e2e_030?sslmode=disable E2E_API_PORT=18130 \
  E2E_WEB_PORT=14130 pnpm run test:e2e -- e2e/simulated-gsap-swbs.spec.ts
```
