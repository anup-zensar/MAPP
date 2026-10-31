# Quickstart: Plan Details Report (043)

How to check the report end to end once 039's hub and 043 are on the branch. Rules and tests are in
[plan.md](plan.md); the operations in [contracts/api.md](contracts/api.md).

## Prerequisites

- `measure-module` with 039 merged; this branch rebased on it; `sqlc generate` run (build guide §4.6)
- Demo data with the Measure story (#DEMO): `cd backend && go run ./cmd/seed`

## Run

```powershell
cd backend;  go run ./cmd/api
cd frontend; pnpm run dev
```

## Check

| Step | Expected |
|---|---|
| Sign in as a planner with one team; open **Reports › Plan details** | Activity tab: cards, built-in section, one row per activity attribute in Settings' order |
| Compare **Planned cost by Country / OU** with All work › Breakdown by Country / OU, same filters | Equal value for value; Not filled in + Does not apply = Breakdown's Not filled in |
| Look at a Lubricants-only attribute (Product launch) | Mobility activities under Does not apply, outside the filled-in per cent |
| Select **Not filled in** on Activities by Company code | All work opens with "Company code: Not filled in" |
| Switch to Campaign, press Back | Back returns to Activity; the address holds `level` |
| Choose a team with no plans this year | "The filters exclude everything" with a reset |
| Sign in as a Viewer without "Export data"; then as a person without "See reports" | No export action; Plan details not offered |
| Export Planned cost by Country / OU as CSV and Excel | Every value, Not filled in, Does not apply, total; equals the table view |

## Automated

```powershell
cd backend;  go test -race -short ./internal/measure/plandetails/... ./internal/app/...
             go test -run '^$' -bench BenchmarkPlanDetails_SC001 -benchtime 5x ./internal/measure/plandetails/
cd frontend; pnpm run check; pnpm run test:e2e -- measure-plan-details
scripts/check-changed.ps1
```
