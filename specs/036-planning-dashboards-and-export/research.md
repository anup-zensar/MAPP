# Research: Planning Dashboards and Export to PowerPoint and PDF

Each section states the decision, why, and what was rejected.

## R1 — Where the figures come from (reuse, not redefine)

**Decision**: the dashboard reads All work's year data through the existing `allwork.Source` (implemented by
`internal/app/allWorkSource`): plans in the actor's scope (`PlanStore.ListInScope`), with every campaign,
initiative, activity and line item, their USD budgets and planned costs (016 conversion), line-item spreads in
USD (007), 015's GSAP facts per line item, completeness (`Missing`), plan type and teams. Sums follow the
domain model: plan budget (§4.1), planned cost = Σ activities (§4.2), committed and paid = Σ line items' facts
(015). The monthly planned cost uses All work's Budget & cost rule, exported from `allwork` as
`PlannedByActivity(nodes, year)` (line-item spreads, the positive unassigned balance spread evenly, No dates and
Other years apart — 010 FR-023), so the dashboard's months equal the Budget & cost view.

**Rejected**: dedicated SQL aggregates (would redefine planned cost, conversion and spreads in SQL, and drift
from the record pages); calling the plan, campaign and activity services per plan (N+1).

## R2 — Actuals by month

**Decision**: a new `financials.Service.PaidByMonth(ctx)` returns, per linked line item, the payments' USD
amounts by posting month (`YYYY-MM`), converted with `usdOf` at the current rate and skipping currencies no
longer in Settings (D-257), exactly as `FactsByLineItem` sums Paid. The dashboard keeps the line items of the
plans whose financial data the person may see; payments posted outside the chosen year count in Actual and
appear as "Paid outside <year>".

**Rejected**: invoices as actuals (Shell: actuals are payments released); monthly exchange rates (Q-023).

## R3 — Approvals panel

**Decision**: a new `approval.Service.RoundSummaries(ctx)` returns every round's subject, outcome, submitted and
ended times, auto-approval and — for an open round — since when its current level waits (the existing
`waitingSince`). The dashboard keeps the rounds of the activities and purchase requests of the plans in view.
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
`PlanStore.GetInScope`. The permission's label changes to "Export from All work, plans and dashboards" in the
catalogue and the web app's two copies.

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

The dashboard reads in one read transaction: the year (All work's Year — its statements do not grow with
records, 028), the person's choices, the transactions (one statement), the rounds and actions (two), the
purchase requests (one scoped statement plus their lines), the global budgets and allocations (one per budget,
as the Global budgets page). No new store method is added, so no new scope-isolation test is due; the scoped
reads it relies on keep their PostgreSQL tests (`plan` store `ListInScope`/`GetInScope`, purchase requests
`OnLineItems`). Contract tests on the in-memory demo data prove the dashboard counts nothing outside the
person's scope (SC-002).

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
