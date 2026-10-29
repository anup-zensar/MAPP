# Quickstart: Planning Dashboards and Export

1. Start PostgreSQL, load the demo data and run the API and the web app:

   ```sh
   cd backend && go run ./cmd/seed -reset && go run ./cmd/api
   cd frontend && pnpm run dev
   ```

2. Sign in as **Carlos Mendes** (Global planner, All markets). Open **Dashboards**: the Executive dashboard for
   the current year shows FY2026 Lubricants Europe and FY2026 Mobility China, the By market table both teams,
   the monthly chart (switch to **Table**), the campaign lists and the approvals, purchase requests, plan details
   and global budgets panels. Change **Currency** to EUR: every amount follows.
3. Select the **Lubricants Europe** row: the Regional dashboard opens for that market. Choose the plan in its
   table: its campaigns appear.
4. Choose **Export › PowerPoint** and **Export › PDF**: both files download and open.
5. Sign in as **Priya Raman** (Planner, Lubricants Europe). The Executive dashboard shows only Lubricants
   Europe's plans. Open **FY2026 Lubricants Europe** › **More** › **Export to PowerPoint**.
6. Sign in as **Lea Santos** (no financial data): Committed and Actual show "—" with "You can't see financial
   data".

Checks: `go test ./internal/dashboard/... ./internal/platform/deck/...`; `pnpm vitest run src/features/dashboards`;
`pnpm run test:e2e -- dashboards`.
