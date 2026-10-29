# Research: Planning Dashboards and Export to PowerPoint and PDF

Each section states the decision, why, and what was rejected.

## R1 — Where the figures come from (reuse, not redefine)

**Decision**: the dashboard reads All work's year data through the existing `allwork.Source` (implemented by
`internal/app/allWorkSource`): plans in the actor's scope (`PlanStore.ListInScope`), with every campaign,
initiative, activity and line item, their USD budgets and planned costs (016 conversion), line-item spreads in
USD (007), 015's GSAP facts per line item, completeness (`Missing`), plan type and teams. Sums follow the
domain model: plan budget (§4.1), planned cost = Σ activities (§4.2), committed = Σ line items' facts
(015), actual = Σ line items' payments posted in the plan's year (R2, D-357). The monthly planned cost uses All work's Budget & cost rule, exported from `allwork` as
`PlannedByActivity(nodes, year)` (line-item spreads, the positive unassigned balance spread evenly, No dates and
Other years apart — 010 FR-023), so the dashboard's months equal the Budget & cost view.

**Rejected**: dedicated SQL aggregates (would redefine planned cost, conversion and spreads in SQL, and drift
from the record pages); calling the plan, campaign and activity services per plan (N+1).

## R2 — Actuals by month

**Decision**: a new `financials.Service.PaidByMonth(ctx, lineItemIDs)` returns, for the line items of the plans
whose financial data the person may see, the payments' USD amounts by posting month (`YYYY-MM`), converted
with `usdOf` at the current rate and skipping currencies no longer in Settings (D-257), exactly as
`FactsByLineItem` sums Paid. It reads them in one statement, `financials.Store.PaymentsOnLineItems` (R9).
A plan's Actual — in the totals, the rows, the campaigns and a plan export's activities — is its payments
posted in the plan's year, as 013's Actual spend (013 FR-010, D-335), so one plan never shows two actuals
(D-357, superseding D-342 there); payments posted outside the year count nowhere and appear under the monthly
chart as "Paid outside the year (not counted)". Committed stays 015's Committed.

**Rejected**: invoices as actuals (Shell: actuals are payments released); monthly exchange rates (Q-023).

## R3 — Approvals panel

**Decision**: a new `approval.Service.RoundSummaries(ctx, subjectIDs)` returns, for the activities and purchase
requests of the plans in view, each round's subject, outcome, submitted and ended times, auto-approval and —
for an open round — since when its current level waits (the existing `waitingSince`). It reads the rounds and
their history in one statement each (`RoundsOfSubjects`, `ActionsOfSubjects`, R9).
The average days to decide reuses the monitoring rule, extracted as `approval.AverageDecisionDays(decided,
seconds)` (half-up to one decimal); monitoring calls it too, so both screens agree.

## R4 — Purchase requests panel

**Decision**: a new `purchaserequest.Service.Summaries(ctx, lineItemIDs)` reads the purchase requests on the
plans' line items through the existing scoped `Store.OnLineItems(viewScope(ctx), …)`, with each request's
amount in USD as the Purchase requests page counts it (`reader.usd`), its status and whether it waits for GSAP
(`waiting`), and whether Purchase Support is on. The dashboard groups by status.

## R5 — Access (D-343)

`GET /api/v1/dashboards/executive|regional` need `plans.view` (any scope); the service counts only plans in
`authz.ScopeOf(ctx)`. Finance figures use `Choices.SeesFinance` and `Choices.FinancePlans` as All work does
(015 FR-028, D-301). Global budgets: `authz.From(ctx).Has(GlobalBudgetsView)`. Exports: the dashboards need
`data.export`; the plan export is `inPlan(data.export)` on `{planId}`, so the access middleware answers 404 for
a plan the person cannot see and 403 when they cannot export in it, and the service reads the plan with
`PlanStore.GetInScope`. The permission keeps its label "Export from All work" for now: the catalogue's two
copies in the web app are over the comment budget, so widening the label waits for the next change of the
access screens (recorded in D-343).

## R6 — Where files are made (ADR-0017)

**Decision**: in the API, with Go's standard library. A .pptx is a zip of XML parts (ECMA-376); a PDF 1.4 with
the 14 standard fonts needs no embedded font. Both are written from one laid-out page model, so they hold the
same pages.

**Rejected**: in the browser with `pptxgenjs` and `jspdf` — two dependencies (~400 kB and ~350 kB minified)
for every user of the route, figures assembled on the client (constitution IV), scope enforced twice;
a headless browser or LibreOffice on the server — a third runtime part (constitution IV needs an ADR for one)
and not available locally on every laptop; Go modules (`unidoc`, `gofpdf`) — licence (AGPL/commercial) or
archived. The 2026 release ages of the candidates were checked but are moot.

## R7 — File layout

16:9 pages, 960 × 540 pt (PowerPoint's widescreen 13.333 × 7.5 in; the PDF's MediaBox is the same), 40 pt
margins. Each page: a title (24 pt bold), a subtitle (12 pt, Text Subtle), optional figure cards (up to 6
across), then either a table, a chart with its table beside it, or both, and notes; a footer "SP&A One — demo
data · Page n of N". Tables continue on new pages titled "<title> (continued)", header repeated. Text is fitted
to its cell with "…", measured with the Helvetica metrics (Arial is metric-compatible), so both writers draw
the same text. In the .pptx tables are real PowerPoint tables (editable); cards, bars, legends and labels are
shapes and text boxes. Colours: Text Strong `#292929`, Text Subtle `#757575`, borders `#E0E0E0`, header fill
`#F5F5F5`, chart tokens of §4.7 (R8), over-budget text Secondary Red `#DD1D21` on white (4.92:1).

PDF text uses WinAnsiEncoding: runes of Windows-1252 are mapped (hand-written table, 27 entries above 0x7F
plus Latin-1), others become "?". The .pptx keeps every character (UTF-8 XML).

## R8 — Charts on screen (D-345)

**Decision**: the web app draws bars as SVG from the API's figures: monthly planned vs actual (grouped bars)
and nothing else; each chart has a Chart / Table switch, a legend with words, `role="img"` with a text summary,
and the Table view is a real `<table>`. Four tokens in `tokens.css` and design system §4.7: `--chart-budget`
`#5B6B7A` (dark `#8696A6`), `--chart-planned` `#336094` (`#6F9BD1`), `--chart-committed` `#008192`
(`#3FB6C5`), `--chart-actual` `#008557` (`#3DBB8C`) — all ≥ 3:1 on white and on the dark surface, none near
Primary Yellow or Secondary Red. Bar heights are proportions of the largest amount, worked out from the API's
decimal strings with `BigInt` cents (no float for money; geometry only).

**Rejected**: Recharts, Chart.js, ECharts — bundle cost for one bar chart; F1 needs an ADR for a chart
library, and a table equivalent is required anyway.

## R9 — Statements per request and scope tests

The dashboard reads in one read-only snapshot, and no read loads data outside the scope to filter it in Go
(constitution B5, D-356): the year (All work's Year over `PlanStore.ListInScope` — its statements do not grow
with records, 028), the person's choices, then for the records of the plans in view only:

| Read | Statement | Limited to |
|---|---|---|
| Payments | `financials.Store.PaymentsOnLineItems` — `type = 'payment' AND line_item_id = ANY(…)` | the line items of the plans whose financial data the person may see |
| Approval rounds and history | `approval.Store.RoundsOfSubjects`, `ActionsOfSubjects` — `subject_id = ANY(…)` | the plans' activities and the purchase requests below |
| Purchase requests | 032's `OnLineItems(viewScope, …)` plus their lines | the plans' line items and 032's view scope |
| Global budgets | `ListBudgets`, then `budget.Store.AllocationsOfBudgets` — `global_budget_id = ANY(…)` | the year's budgets; no plan scope by 014's rule (exception, plan Complexity Tracking) |

The record IDs come from the scoped plan read in the same snapshot, so each statement returns nothing outside
the scope, whatever the data; one statement per read, never one per budget or per record (the Global budgets
page's `List` reads one per budget and is not used). The new scoped reads have PostgreSQL scope-isolation
tests (`TestPostgresStorePaymentsOnLineItemsScopeIsolation_036_FR002`,
`TestPostgresStoreRoundsOfSubjectsScopeIsolation_036_FR002`), as the plan and purchase-request reads already
do; contract tests on the in-memory demo data prove the dashboard counts nothing outside the person's scope
(SC-002). A scope function in each statement (`activity_in_scope`, `plan_in_permission_scope`) was rejected:
the IDs already carry the scope, and the in-memory fakes would have to resolve every record's plan.

**SC-003** is measured by `BenchmarkDashboards` in `internal/platform/apibench` on 028's volume (Priya's team:
50 plans of 2026, 5,000 activities, 20,000 line items; 5,000 GSAP rows):

```sh
cd backend
export TEST_DATABASE_URL='postgres://mapp@127.0.0.1:5432/postgres?sslmode=disable'
go test -run '^$' -bench 'BenchmarkDashboards/demo/' -benchtime 20x ./internal/platform/apibench/
go test -run '^$' -bench 'BenchmarkDashboards/volume/' -benchtime 5x -timeout 3h ./internal/platform/apibench/
```

**Measured on the demo data** (2026-10-29, Apple M5, PostgreSQL 18 local, 20 runs each): Executive dashboard
17 ms, 47 statements; Executive of one market (Priya) 16 ms, 45; Regional 15 ms, 45; Executive export .pptx
21 ms, 47; plan export .pptx 18 ms and .pdf 18 ms, 51 each. **The volume half is still to run**: its template
database was being built by another benchmark on the shared server and did not finish within the ten minutes
given to this change, so SC-003's 3 s and 5 s at 028's volume are not yet measured; run the second command
above and record the result here (task T034).

## R10 — Threat and risk model

| Threat (STRIDE) | Response |
|---|---|
| Information disclosure: a market person sees another market's figures through totals | Every figure is computed from plans in `ScopeOf(ctx)`; contract tests per persona; e2e checks Priya's dashboard |
| Information disclosure: finance figures without `finance.view` | Finance sums only over `FinancePlans`; "—" otherwise; contract test with Lea Santos (no `finance.view`) |
| Information disclosure: probing plan IDs through the export | Middleware answers 404 for invisible plans (`inPlan` target), 403 only for visible ones |
| Tampering: scope or team in the query | The scope comes from the actor only; `teamId` only narrows within the scope |
| Denial of service: very large exports | One year, one plan or one dashboard per request; read-only; request time bounded by the server's timeouts; tables paginate |
| Injection into files: formula or markup in names | Office XML is escaped by `encoding/xml`; PDF strings escape `(`, `)` and `\`; no formulas exist in .pptx text; control characters are dropped |
| Repudiation | Read-only; the request log records who exported what (operation, actor) |
| Elevation | Route rules fail closed; the dashboard calls services with the actor's grants in `ctx` |

**Risks**: (1) PowerPoint is stricter than other readers — mitigated by writing the minimal part set
PowerPoint's own blank deck has, validated by `python-pptx` and macOS Quick Look during development and by a
structural test in CI; accepted by the team. (2) PDF standard fonts cannot show non-Latin text — accepted
(the .pptx keeps it; noted on screen in the spec's edge cases) until Shell supplies an embeddable font.
