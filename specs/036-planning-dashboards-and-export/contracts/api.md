# API Contract: Planning Dashboards and Export

The schemas are in `backend/api/openapi.yaml` (tag **Dashboards**): `PlanningDashboard`, `DashboardFilters`,
`DashboardChoices`, `DashboardFigures`, `DashboardFinance`, `DashboardMarketRow`, `DashboardPlanRow`,
`DashboardCampaignRow`, `DashboardPlanCampaigns`, `DashboardMonths`, `DashboardMonth`, `DashboardApprovals`,
`DashboardPurchaseRequests`, `DashboardCompleteness`, `DashboardGlobalBudget`. Errors use the one envelope.

| Operation | Query | Answer | Access rule (`access/routes_dashboards.go`) |
|---|---|---|---|
| `GET /api/v1/dashboards/executive` (`getExecutiveDashboard`) | `year`, `planType`, `currency` | 200 `PlanningDashboard`; 400 `validation_failed` (fields `year`, `planType`, `currency`) for a malformed value only — an unknown currency falls back to the default and `filters` says which (038 FR-012) | `need(plans.view)` |
| `GET /api/v1/dashboards/regional` (`getRegionalDashboard`) | `year`, `planType`, `currency`, `teamId`, `planId` | 200 `PlanningDashboard`; 400 (also a malformed `teamId`); an unknown or out-of-scope team falls back to the default market (038 FR-012) | `need(plans.view)` |
| `GET /api/v1/dashboards/executive/export` (`exportExecutiveDashboard`) | `format` (`pptx`, `pdf`) + the view's query | 200 file; 400 | `need(data.export)` |
| `GET /api/v1/dashboards/regional/export` (`exportRegionalDashboard`) | `format` + the view's query | 200 file; 400 | `need(data.export)` |
| `GET /api/v1/plans/{planId}/export` (`exportPlan`) | `format`, `currency` | 200 file; 400; 403 (no `data.export` in the plan); 404 (missing or not visible) | `inPlan(data.export)` |

Files: `application/vnd.openxmlformats-officedocument.presentationml.presentation` or `application/pdf`, with
`Content-Disposition: attachment; filename="…"; filename*=UTF-8''…` (All work's `ContentDisposition`). Names:

- `SP&A One – <plan name> – 28 Oct 2026.pptx`
- `SP&A One – Executive dashboard – 2026 – 28 Oct 2026.pdf`
- `SP&A One – Regional dashboard – Lubricants Europe – 2026 – 28 Oct 2026.pptx`

## Go interfaces (consumers' ports, `internal/dashboard/ports.go`)

```go
type Planning interface { // allwork.Source's reads (010)
	Years(ctx context.Context) ([]int, error)
	Year(ctx context.Context, year int, today date.Date, columns []string) (allwork.YearData, error)
	Choices(ctx context.Context, personID string) (allwork.Choices, error)
}
type Plans interface { // 004, scoped
	YearOf(ctx context.Context, planID string) (int, bool, error)
}
type Spending interface { // 015: only the line items asked for (D-356, D-358)
	SpendByMonth(ctx context.Context, lineItemIDs []string) (map[string]map[string]financials.Facts, error)
}
type Rounds interface { // 017: only the records asked for (D-356)
	RoundSummaries(ctx context.Context, subjectIDs []string) ([]approval.RoundSummary, error)
}
type Purchases interface { // 032, scoped
	Summaries(ctx context.Context, lineItemIDs []string) (purchaserequest.Summaries, error)
}
type GlobalBudgets interface { // 014, needs globalBudgets.view (D-343, D-356)
	YearFigures(ctx context.Context, year int) ([]budget.Summary, error)
}
type Directory interface { // 002, 003
	Currencies(ctx context.Context, personID string) (Currencies, error)
	TeamRefs(ctx context.Context, teamIDs []string) (map[string]people.TeamRef, error)
}
```

New exported functions in other areas:

- `allwork.PlannedByActivity(nodes []allwork.Node, year int) map[string]allwork.Months` — each activity's
  planned cost by month of `year`, with `NoDates` and `OtherYears`, in USD (010 FR-023).
- `financials.(*Service).SpendByMonth(ctx, lineItemIDs)` — the Committed and Paid of those line items, USD by
  posting month (store read `SpendOnLineItems`).
- `approval.(*Service).RoundSummaries(ctx, subjectIDs)` (store reads `RoundsOfSubjects`, `ActionsOfSubjects`)
  and `approval.AverageDecisionDays(decided, seconds int64) *string`.
- `purchaserequest.(*Service).Summaries(ctx, lineItemIDs)` — scoped purchase requests with status, USD
  amount, waiting for GSAP, and whether Purchase Support is on.
- `budget.(*Service).YearFigures(ctx, year)` — the year's global budgets with the Global budgets page's figures,
  in three statements (store read `AllocationsOfBudgets`).
