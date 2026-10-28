# Quickstart: The Approval Engine Approves Any Record Type

Run locally on feature 031's own database and ports (integration brief §7).

```sh
export PATH="$HOME/.local/npm/bin:$HOME/go/bin:$PATH"
cd backend
sqlc generate && sqlc vet
DATABASE_URL='postgres://mapp:mapp@127.0.0.1:5432/mapp_f031?sslmode=disable' go run ./cmd/seed
DATABASE_URL='postgres://mapp:mapp@127.0.0.1:5432/mapp_f031?sslmode=disable' APP_PORT=18131 go run ./cmd/api
```

## 1. Re-submit a Submitted activity (US1)

1. Sign in as Priya Raman. Open "Trade magazine ads" (Submitted, round 2 waiting for Daniel Okoro).
2. In the Approval section choose **Re-submit for approval**: the dialog says "Round 2, waiting at Level 1,
   ends and round 3 starts at the first level that needs a decision."
3. Confirm: round 2 shows **Replaced by a new submission**; round 3 waits for Daniel Okoro.

API: `GET /api/v1/activities/A-1004/approval/submission` → `kind: "resubmit"`, `replaces.roundNumber: 2`;
`POST /api/v1/activities/A-1004/approval/submit {"version": <activityVersion>}` → `rounds[1].outcome:
"superseded"`, `rounds[0].outcome: "open"`.

## 2. Workflow object types (US2)

1. Sign in as Alex Morgan; Settings › Approval workflows: the **Approves** column shows "Activities" for
   every seeded workflow (feature 032 seeds the purchase-request ones).
2. Create "Purchase request approval" with **Approves: Purchase requests** (see §3), then filter
   **Approves › Purchase requests**: only that workflow is listed, and the address keeps `?approves=`.
3. Edit "Lubricants EU approval": **Approves** is fixed ("It approves activities and is in use…").
4. Open any activity's form: the Approval workflow field offers no purchase-request workflow.

API: `GET /api/v1/approval-workflows` → `items[].objectType`; `GET /api/v1/approval-rule-options` →
`objectTypes`, `recordFields` (`moaApprover`).

## 3. New approver sources (US3)

Create a purchase-request workflow with level 1 **Person named on the record › MoA approver**, then try the
same on an activity workflow: refused with "Level 1: activities have no field that names an approver.
Choose another option."

## 4. Purchase requests (US4, US5)

Proven by `go test ./internal/approval/ -run 'Subject|Purchase'` with a stand-in purchase request until
feature 032 connects the port (`UsePurchaseRequests`). The Approvals page shows a **Record type** column
("Activity" on every seeded row).

## Checks

```sh
cd backend && go vet ./... && golangci-lint run --new ./... && \
  TEST_DATABASE_URL='postgres://mapp:mapp@127.0.0.1:5432/postgres?sslmode=disable' go test -race ./... && govulncheck ./...
cd frontend && pnpm run api:check && pnpm run typecheck && pnpm run lint && pnpm run format:check && pnpm run test && pnpm run build
E2E_DATABASE_URL=postgres://mapp:mapp@127.0.0.1:5432/mapp_e2e_031?sslmode=disable E2E_API_PORT=18131 E2E_WEB_PORT=14131 \
  pnpm run test:e2e -- e2e/approval-any-record.spec.ts
```
