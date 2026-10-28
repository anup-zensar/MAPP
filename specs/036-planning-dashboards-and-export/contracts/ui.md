# UI Contract: Planning Dashboards and Export

## Routes

| Path | Page | Title |
|---|---|---|
| `/dashboards` | redirects to `/dashboards/executive` (keeps the query) | — |
| `/dashboards/executive?year&planType&currency` | `DashboardsPage kind="executive"` | Executive dashboard |
| `/dashboards/regional?year&planType&currency&team&plan` | `DashboardsPage kind="regional"` | Regional dashboard |

The sidebar item **Dashboards** (`LayoutDashboard` icon, permission `plans.view`) sits after All work.

## Feature `src/features/dashboards`

| Module | Role |
|---|---|
| `index.ts` | exports `DashboardsPage`, `DashboardsRedirect` |
| `api.ts` | `useDashboard(kind, filters)` (key `['dashboards', kind, query]`), `downloadDashboard(kind, filters, format)` |
| `schemas.ts` | Zod schema of the address: `year` (int 2000–2100), `planType`, `currency` (`/^[A-Z]{3}$/`), `team`, `plan`; invalid values drop to the default |
| `dashboard-links.ts` | address ↔ filters ↔ API query; the Regional link of a market row |
| `dashboard-text.ts` | words: figure labels, status words, empty and finance notes |
| `DashboardsPage.tsx` | page header (title, tabs, Export menu), filters, states (loading, error, forbidden, empty), panels |
| `DashboardFilters.tsx` | Year, Market (Regional), Plan type, Currency selects |
| `FigureCards.tsx` | the summary figures |
| `MarketsTable.tsx`, `PlansTable.tsx`, `CampaignTable.tsx` | tables; plan rows choose the plan (Regional) and link to the plan |
| `MonthlyChart.tsx` | SVG grouped bars with a Chart / Table switch |
| `ApprovalsPanel.tsx`, `PurchaseRequestsPanel.tsx`, `CompletenessPanel.tsx`, `GlobalBudgetsPanel.tsx` | panels |
| `ExportMenu.tsx` | Export › PowerPoint / PDF, busy state, error toast |

## Plan page

`src/features/plans/usePlanExport.tsx` gives the plan page's "More" menu two items, **Export to PowerPoint** and
**Export to PDF**, shown to holders of `data.export`; while a file is made the items are disabled with "Preparing
the file…", and a failure shows the API's message in a toast.

## States

| State | What the page shows |
|---|---|
| Loading | Skeleton cards and panels in reserved space (`aria-busy`) |
| Error | `ErrorState` with Retry and the request ID |
| Forbidden | `NoAccess` (403 from the route rule) |
| Empty (no plans) | `EmptyState` "No plans to show for 2027" with "Change the year or the filters" |
| Data | every panel; panels with nothing to count say so ("No approvals yet", "Purchase requests are switched off") |
