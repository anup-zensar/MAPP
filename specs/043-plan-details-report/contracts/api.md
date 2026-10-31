# API Contract: Plan Details Report (043)

For the **wave A contract** (lane A writes it into `backend/api/openapi.yaml`, tag **Reports**). It follows
039's convention: `GET /api/v1/reports/<path>` returns the report's widgets, and the report's
`measure.Report.Sheets` feeds 039's shared widget export. Names marked *(039)* are 039's and are used as they
are; the integrator aligns them with 039's final contract. Errors use the one envelope.

## Operations

| Operation | Query | Answer | Access rule (`access/routes_measure.go`, 039) |
|---|---|---|---|
| `GET /api/v1/reports/plan-details` (`getPlanDetailsReport`) | 039's filter, period and currency parameters *(039)* + `level` (`plan`, `campaign`, `initiative`, `activity`; default `activity`) | 200 `PlanDetailsReport`; 400 `validation_failed` (field `level`, or a malformed 039 parameter) | `need(authz.ReportsView)` |
| `GET /api/v1/reports/plan-details/export` (`exportPlanDetailsWidget`) — served by 039's shared export through `measure.Report.Sheets` | the report's query + `widget` (a `PlanDetailsWidget.id`) + `format` (`csv`, `xlsx`) *(039)* | 200 file; 400 `validation_failed` (unknown `widget` at that level, or `format`); 404 never — an empty widget gives a file with the header and total row | `need(authz.DataExport)` |

The report counts only the plans in the person's scope (no `inPlan`: no record is named in the address). Planned
cost needs no `finance.view` (D-237), so the report has no financial gate. An out-of-scope plan named in 039's
`plans` filter is treated as 039 treats it *(039)*.

## Schemas

```yaml
PlanDetailsReport:
  required: [level, currency, state, cards, builtIn, attributes, attributesSetUp]
  properties:
    level:      { enum: [plan, campaign, initiative, activity] }
    currency:   { type: string }                 # the display currency, ISO code (D-303)
    state:      { $ref: ReportState }            # (039) data | noPlans | noData | filteredOut — the tab as a whole
    cards:      { $ref: PlanDetailsCards }
    builtIn:    { type: array, items: { $ref: PlanDetailsWidget } }     # FR-006, FR-007; [] unless state = data
    attributes: { type: array, items: { $ref: PlanDetailsAttributeRow } } # FR-008, Settings' display order
    attributesSetUp: { type: boolean }           # false: "No <level> attributes are set up in Settings"

PlanDetailsCards:
  required: [records, plannedCost, allFilledIn, allFilledInPercent]
  properties:
    records:            { type: integer }        # plans, campaigns, initiatives or activities matching
    plannedCost:        { $ref: Money }          # their planned cost (domain model §4.2), display currency
    allFilledIn:        { type: integer }        # records with Missing = 0 (domain model §2.6)
    allFilledInPercent: { type: integer, nullable: true }  # whole per cent, half up; null when records = 0

PlanDetailsAttributeRow:
  required: [attributeId, label, type, widgets]
  properties:
    attributeId: { type: string }                # identity, never the label (client 019 FR-030)
    label:       { type: string }
    type:        { enum: [singleChoice, multipleChoice, multipleChoiceSplit, yesNo, text, longText, number] }
    widgets:     { type: array, items: { $ref: PlanDetailsWidget } }  # 3 for choice and yes/no, 1 (filledIn) otherwise

PlanDetailsWidget:
  required: [id, title, measure, values, total]
  properties:
    id:       { type: string }    # stable: "activityType.count", "activityType.plannedCost", "status.count",
                                  # "timing.count", "startPeriod.count", "plannedCostEntered", "goalEntered",
                                  # "attr.<attributeId>.count" | ".plannedCost" | ".filledIn"
    title:    { type: string }    # "Activities by Company code", "Planned cost by Company code", "Company code filled in"
    measure:  { enum: [count, plannedCost, filledIn] }
    values:   { type: array, items: { $ref: PlanDetailsValue } }  # every value, largest first; then notFilledIn,
                                  # doesNotApply (filledIn: filled, notFilledIn, doesNotApply); start periods in time order
    other:    { $ref: PlanDetailsValue, nullable: true }  # the values after the ten largest, summed, for the chart (FR-017)
    total:    { $ref: PlanDetailsValue }   # each record once (FR-011)
    filledInPercent: { type: integer, nullable: true }    # filledIn only: filled ÷ (filled + notFilledIn), half up
    countsInEach:    { type: boolean }     # true: "<Records> with several values count in each group; the total counts them once"
    filterKey:       { type: string, nullable: true }     # All work filter the values open (FR-014): "attr.<id>",
                                  # "activityType", "status"; null for start period, planned cost entered, goal entered

PlanDetailsValue:
  required: [key, label, count]
  properties:
    key:         { type: string }  # option id, "yes", "no", activity type id, status, "2026-Q2" / "2026-04",
                                   # or notFilledIn | doesNotApply | noDates | outsideYear | filled | other | total
    label:       { type: string }  # "(inactive)" appended by the server for an inactive option
    inactive:    { type: boolean }
    count:       { type: integer }
    plannedCost: { $ref: Money, nullable: true }  # set when measure = plannedCost
```

## Export sheets (`measure.Report.Sheets`)

One sheet per widget, columns as the table view: **Value**, then **Activities** (or Plans, Campaigns,
Initiatives) for `count` and `filledIn`, **Planned cost (<currency>)** for `plannedCost`, **Share** where the
table shows it; every value, Not filled in, Does not apply and the total row (FR-017; client 019 FR-014). 039's
export adds the notes (filters, currency, rate, "as of") and the file name *(039)*.

## Go interfaces (`internal/measure/plandetails/ports.go`)

```go
type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
}
// Planning is All work's reads (010), adapted in app/measure_plandetails.go as 036's dashboardPlanning:
// the plans of a year in the person's scope with everything in them (research R1).
type Planning interface {
	Year(ctx context.Context, year int, today date.Date, columns []string) (allwork.YearData, error)
	Attributes(ctx context.Context) ([]allwork.Attribute, error)
	Choices(ctx context.Context, personID string) (allwork.Choices, error)
}
```

The display currency and its current rate come resolved from 039's Currency parameter *(039)*; amounts are
summed in USD and converted once (D-303), as All work does.

No exported function is added outside lane D's paths. Proposed to the integrator (research R2):
`allwork.AttributeParts`, which would replace 043's local copy of 016's attribution rule.
