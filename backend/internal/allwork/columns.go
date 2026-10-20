package allwork

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"northstar/internal/platform/date"
	"northstar/internal/platform/money"
	"northstar/internal/rollup"
)

// The columns of the grid (data model §6; FR-018 – FR-020): the catalogue, the default columns and the value
// of every cell. Name is always shown and is not a column. A column has values on its own levels only: on
// other levels its cell is blank (null), and an empty value on its own level is "—" (KindEmpty).

// The column groups of the chooser (FR-018).
const (
	GroupStandard   = "standard"
	GroupPlan       = "plan"
	GroupCampaign   = "campaign"
	GroupInitiative = "initiative"
	GroupActivity   = "activity"
	GroupLineItem   = "lineItem"
)

// The level sets of the columns.
var (
	recordLevels    = []Level{LevelPlan, LevelCampaign, LevelInitiative, LevelActivity, LevelLineItem}
	datedLevels     = []Level{LevelPlan, LevelCampaign, LevelInitiative, LevelActivity}
	containerLevels = []Level{LevelPlan, LevelCampaign, LevelInitiative}
	rolledLevels    = []Level{LevelPlan, LevelCampaign, LevelInitiative, LevelActivity}
	costLevels      = []Level{LevelPlan, LevelCampaign, LevelInitiative, LevelActivity, LevelLineItem, LevelUnassigned}
	balanceLevels   = []Level{LevelPlan, LevelCampaign, LevelInitiative, LevelActivity, LevelUnassigned}
)

// standardColumns is the Standard group, in the order of FR-018 (with the Activities column of D-119).
var standardColumns = []Column{
	standard("id", "ID", KindText, recordLevels),
	standard("level", "Level", KindText, recordLevels),
	standard("owner", "Owner", KindText, recordLevels),
	standard("startDate", "Start date", KindDate, datedLevels),
	standard("endDate", "End date", KindDate, datedLevels),
	standard("status", "Status", KindStatus, recordLevels),
	standard("timing", "Timing", KindStatus, []Level{LevelCampaign, LevelInitiative}),
	standard("completeness", "Completeness", KindStatus, recordLevels),
	standard("planBudget", "Plan budget", KindMoney, containerLevels),
	standard("remainingBudget", "Remaining budget", KindMoney, []Level{LevelPlan, LevelCampaign}),
	standard("budgetUsed", "Budget used", KindPercent, containerLevels),
	standard("plannedCost", "Planned cost", KindMoney, costLevels),
	standard("lineItems", "Line items", KindMoney, rolledLevels),
	standard("unassigned", "Unassigned planned cost", KindMoney, balanceLevels),
	standard("activities", "Activities", KindNumber, containerLevels),
	standard("inq", "INQ", KindNumber, rolledLevels),
	standard("mql", "MQL", KindNumber, rolledLevels),
	standard("revenue", "Revenue", KindMoney, rolledLevels),
	standard("createdAt", "Created", KindDateTime, recordLevels),
	standard("updatedAt", "Last changed", KindDateTime, recordLevels),
	// 016 FR-037: the plan's year, type and currency, and the amounts in the plan currency.
	standard("year", "Year", KindNumber, recordLevels),
	standard("planType", "Plan type", KindText, recordLevels),
	standard("planCurrency", "Plan currency", KindText, recordLevels),
	standard("planBudgetPlan", "Plan budget (plan currency)", KindMoney, containerLevels),
	standard("plannedCostPlan", "Planned cost (plan currency)", KindMoney, []Level{LevelActivity, LevelLineItem}),
}

// planTypeWords are the plan types as people read them (016 FR-032).
var planTypeWords = map[string]string{"operational": "Operational", "strategic": "Strategic", "crossMarket": "Cross-market"}

// DefaultColumns are the columns of the standard view, in order (FR-019).
var DefaultColumns = []string{"id", "status", "owner", "startDate", "endDate", "planBudget", "plannedCost", "lineItems", "unassigned"}

// rolledColumns are the roll-up columns: on a container they add up the matching activities shown below it
// and carry the unfiltered value when filters hide some (FR-013).
var rolledColumns = []string{"budgetUsed", "plannedCost", "lineItems", "unassigned", "activities", "inq", "mql", "revenue"}

// standard returns a column of the Standard group.
func standard(key, label, kind string, levels []Level) Column {
	return Column{Key: key, Label: label, Group: GroupStandard, Kind: kind, Levels: levels, Sortable: true}
}

// levelColumns are the fields of each level's group (data model §6.2, field catalogue §1–§5), before the
// group's Shell attributes. Fields that Finance or GSAP fill later are not offered (FR-018).
var levelColumns = []Column{
	ofLevel(LevelPlan, "plan.year", "Year", KindNumber),
	ofLevel(LevelPlan, "plan.model", "Model", KindText),
	ofLevel(LevelPlan, "plan.teams", "Teams", KindText),
	ofLevel(LevelPlan, "plan.strategic", "Strategic / AOP plan", KindText),
	ofLevel(LevelPlan, "plan.description", "Description", KindText),
	ofLevel(LevelPlan, "plan.defaultFinanceBudget", "Default finance budget", KindText),
	ofLevel(LevelPlan, "plan.defaultSpread", "Default line-item spread", KindText),
	ofLevel(LevelPlan, "plan.bcpir", "BCPIR plan", KindText),
	ofLevel(LevelPlan, "plan.goalStage", "Goal stage", KindText),
	ofLevel(LevelPlan, "plan.goalAmount", "Goal amount", KindNumber),
	ofLevel(LevelPlan, "plan.goalMql", "MQL goal", KindNumber),
	ofLevel(LevelPlan, "plan.goalRevenue", "Revenue goal", KindMoney),
	ofLevel(LevelCampaign, "campaign.description", "Description", KindText),
	ofLevel(LevelInitiative, "initiative.description", "Description", KindText),
	ofLevel(LevelActivity, "activity.externalName", "External name", KindText),
	ofLevel(LevelActivity, "activity.type", "Activity type", KindText),
	ofLevel(LevelActivity, "activity.ongoing", "Ongoing", KindText),
	ofLevel(LevelActivity, "activity.approvalWorkflow", "Approval workflow", KindText),
	ofLevel(LevelActivity, "activity.description", "Description", KindText),
	ofLevel(LevelActivity, "activity.setLaterStages", "Set later stages myself", KindText),
	ofLevel(LevelLineItem, "lineItem.description", "Description", KindText),
	ofLevel(LevelLineItem, "lineItem.localPlannedCost", "Planned cost in local currency", KindMoney),
	ofLevel(LevelLineItem, "lineItem.currency", "Currency", KindText),
	ofLevel(LevelLineItem, "lineItem.spread", "Spread over time", KindText),
	ofLevel(LevelLineItem, "lineItem.contractedAmount", "Contracted amount", KindMoney),
	ofLevel(LevelLineItem, "lineItem.requestSwbs", "Request SWBS code from GSAP", KindText),
	ofLevel(LevelLineItem, "lineItem.swbsStatus", "SWBS request", KindStatus),
	ofLevel(LevelLineItem, "lineItem.financeBudgetYear", "Finance budget year", KindNumber),
	ofLevel(LevelLineItem, "lineItem.financeBudgetItem", "Finance budget item", KindText),
}

// ofLevel returns a column of a level's group, with values on that level only. The groups are named after
// their levels.
func ofLevel(level Level, key, label, kind string) Column {
	return Column{Key: key, Label: label, Group: string(level), Kind: kind, Levels: []Level{level}, Sortable: true}
}

// attributeColumn returns the column "attr.<id>" of a Shell attribute, in its level's group.
func attributeColumn(a Attribute) Column {
	kind := KindText
	if a.Number {
		kind = KindNumber
	}
	c := ofLevel(a.Level, attributeKey(a.ID), a.Label, kind)
	c.Inactive = !a.Active
	return c
}

// attributeKey returns the column key of the Shell attribute id.
func attributeKey(id string) string {
	return "attr." + id
}

// catalogue returns every column the grid offers (FR-018): the Standard group, then each level's group —
// its fields, then its Shell attributes in Settings' order, active and inactive.
func catalogue(attributes []Attribute) []Column {
	columns := slices.Clone(standardColumns)
	for _, level := range []Level{LevelPlan, LevelCampaign, LevelInitiative, LevelActivity, LevelLineItem} {
		for _, c := range levelColumns {
			if c.Group == string(level) {
				columns = append(columns, c)
			}
		}
		for _, a := range attributes {
			if a.Level == level {
				columns = append(columns, attributeColumn(a))
			}
		}
	}
	return columns
}

// offered returns the catalogue as the column chooser offers it: without the inactive attributes, except
// those that named lists — the person's remembered columns and saved views — still name (data model §6.2).
func offered(columns []Column, named ...[]string) []Column {
	kept := []Column{}
	for _, c := range columns {
		if !c.Inactive || slices.ContainsFunc(named, func(keys []string) bool { return slices.Contains(keys, c.Key) }) {
			kept = append(kept, c)
		}
	}
	return kept
}

// catalogueOf returns the columns by key.
func catalogueOf(columns []Column) map[string]Column {
	byKey := make(map[string]Column, len(columns))
	for _, c := range columns {
		byKey[c.Key] = c
	}
	return byKey
}

// resolveColumns returns the columns the query asks for that exist, in its order (the default columns when
// it names none), and the left-out keys of unknown columns and of an unknown sort column (FR-031). It also
// returns the sort to apply: nil when the query has none or its column is unknown.
func resolveColumns(q Query, catalogue []Column) ([]Column, *Sort, []LeftOut) {
	byKey := catalogueOf(catalogue)
	keys := q.Columns
	if keys == nil {
		keys = DefaultColumns
	}
	columns := []Column{}
	leftOut := []LeftOut{}
	for _, key := range keys {
		if c, known := byKey[key]; known {
			columns = append(columns, c)
		} else {
			leftOut = append(leftOut, LeftOut{Filter: "column", ID: key})
		}
	}
	sort := q.Sort
	if sort != nil {
		if _, known := byKey[sort.Column]; !known {
			leftOut = append(leftOut, LeftOut{Filter: "sort", ID: sort.Column})
			sort = nil
		}
	}
	return columns, sort, leftOut
}

// cellRow is what the cells of one row are worked out from.
type cellRow struct {
	level Level
	t     *tnode // the record; for an unassigned row, its activity
	year  int
}

// cellOf returns the cell of column c in row r (FR-011, FR-017, FR-020): nil (blank) on a level the
// column does not cover; on a container, a roll-up column carries its unfiltered value when filters hide
// records below (FR-013).
func cellOf(c Column, r cellRow, cur Currency) *Cell {
	if !slices.Contains(c.Levels, r.level) {
		return nil
	}
	if r.level == LevelUnassigned { // its amount in Planned cost and Unassigned planned cost
		return moneyCell(cur, r.t.balance)
	}
	if r.t.isContainer() && slices.Contains(rolledColumns, c.Key) {
		cell := rolledCell(c.Key, r.t, r.t.fig, cur)
		if r.t.filtered() {
			cell.Unfiltered = rolledCell(c.Key, r.t, r.t.all, cur)
		}
		return cell
	}
	return recordCell(c.Key, r, cur)
}

// recordCell returns the cell of a column that is not a container's roll-up.
func recordCell(key string, r cellRow, cur Currency) *Cell {
	n := r.t.n
	switch key {
	case "id":
		return textCell(n.ID)
	case "level":
		return textCell(levelWords[n.Level])
	case "owner":
		if n.Owner == nil {
			return emptyCell()
		}
		return &Cell{Kind: KindText, Text: n.Owner.Name, Inactive: n.Owner.Inactive}
	case "startDate":
		if n.Level == LevelPlan {
			return dateCell(date.YearStart(r.year))
		}
		return optionalDateCell(n.StartDate)
	case "endDate":
		switch {
		case n.Level == LevelPlan:
			return dateCell(date.YearEnd(r.year))
		case n.Ongoing:
			return textCell(ongoingWord)
		}
		return optionalDateCell(n.EndDate)
	case "status":
		return statusCell(statusOf(n))
	case "timing":
		return statusCell(timingOf(n))
	case "completeness":
		return statusCell(completenessOf(n.Missing))
	case "planBudget":
		return moneyCell(cur, n.Budget)
	case "remainingBudget":
		return remainingCell(r.t, cur)
	case "plannedCost":
		return moneyCell(cur, n.PlannedCost)
	case "createdAt":
		return dateTimeCell(n.CreatedAt)
	case "updatedAt":
		return dateTimeCell(n.UpdatedAt)
	case "year": // 016 FR-037
		year := int64(r.year)
		return numberCell(&year)
	case "planType":
		return textCell(planTypeWords[planOf(r.t).n.PlanType]) // of the record's plan
	case "planCurrency":
		return textCell(planOf(r.t).n.PlanCurrency)
	case "planBudgetPlan":
		return planMoneyCell(n.BudgetPlan)
	case "plannedCostPlan":
		return planMoneyCell(n.PlannedCostPlan)
	}
	if slices.Contains(rolledColumns, key) { // an activity's own values
		return rolledCell(key, r.t, r.t.fig, cur)
	}
	if key == "lineItem.swbsStatus" {
		return statusCell(swbsOf(n))
	}
	return fieldCell(n.Fields[key], cur)
}

// fieldCell returns the cell of a level-specific value (FR-020): "—" when empty; a choice by its label,
// several joined with ", ", a % split with the shares ("Germany 60 %, France 40 %"); inactive values
// labelled; Yes or No; an amount in USD in the display currency, a local amount as it is.
func fieldCell(v Value, cur Currency) *Cell {
	switch {
	case v.USD != nil:
		return moneyCell(cur, v.USD)
	case v.Money != nil:
		m := *v.Money
		cell := &Cell{Kind: KindMoney, Money: &m}
		if m.Amount.Sign() < 0 {
			cell.Tone = ToneProblem
		}
		return cell
	case v.Number != nil:
		return numberCell(v.Number)
	case v.Yes != nil:
		return textCell(yesNo(*v.Yes))
	case len(v.Choices) == 1 && v.Choices[0].Share == nil:
		return &Cell{Kind: KindText, Text: v.Choices[0].Label, Inactive: v.Choices[0].Inactive}
	case len(v.Choices) > 0:
		parts := make([]string, 0, len(v.Choices))
		for _, c := range v.Choices {
			part := c.Label
			if c.Inactive {
				part += " (inactive)"
			}
			if c.Share != nil {
				part += " " + shareText(*c.Share) + " %"
			}
			parts = append(parts, part)
		}
		return textCell(strings.Join(parts, ", "))
	}
	cell := textCell(v.Text)
	cell.Inactive = v.Inactive && v.Text != ""
	return cell
}

// yesNo is a yes/no value as people read it.
func yesNo(yes bool) string {
	if yes {
		return "Yes"
	}
	return "No"
}

// shareText writes a share without trailing zeros: 60.00 → "60", 33.50 → "33.5".
func shareText(p money.Percent) string {
	return strings.TrimSuffix(strings.TrimRight(p.String(), "0"), ".")
}

// rolledCell returns the value of a roll-up column from figures f of the record t.
func rolledCell(key string, t *tnode, f figures, cur Currency) *Cell {
	switch key {
	case "budgetUsed":
		return budgetUsedCell(f.planned, t.n.Budget)
	case "plannedCost":
		return moneyCell(cur, f.planned)
	case "lineItems":
		return moneyCell(cur, f.lineItems)
	case "unassigned":
		return moneyCell(cur, f.unassigned)
	case "activities":
		count := int64(f.activities)
		return numberCell(&count)
	case "inq":
		return numberCell(f.inq)
	case "mql":
		return numberCell(f.mql)
	case "revenue":
		return moneyCell(cur, f.revenue)
	}
	return emptyCell()
}

// ongoingWord is the End date of an ongoing activity.
const ongoingWord = "Ongoing"

// remainingCell is a plan's or campaign's own budget minus the budgets of its children; red when negative
// (D-117). "—" without a budget.
func remainingCell(t *tnode, cur Currency) *Cell {
	if t.n.Budget == nil {
		return emptyCell()
	}
	children := make([]*money.Money, 0, len(t.children))
	for _, c := range t.children {
		if c.n.Budget != nil {
			b := money.USD(*c.n.Budget)
			children = append(children, &b)
		}
	}
	budget := money.USD(*t.n.Budget)
	remaining := rollup.Remaining(&budget, rollup.Total(children))
	return moneyCell(cur, &remaining.Amount)
}

// budgetUsedCell is planned cost ÷ own budget as a whole per cent, half up (005's rule); red above 100.
// "—" without a budget, with a budget of 0.00 or without a planned cost.
func budgetUsedCell(planned, budget *money.Amount) *Cell {
	if planned == nil || budget == nil {
		return emptyCell()
	}
	p, b := money.USD(*planned), money.USD(*budget)
	percent := rollup.BudgetUsedPercent(&p, &b)
	if percent == nil {
		return emptyCell()
	}
	cell := &Cell{Kind: KindPercent, Percent: strconv.Itoa(*percent)}
	if *percent > 100 {
		cell.Tone = ToneProblem
	}
	return cell
}

// emptyCell is "—": the column's own level without a value.
func emptyCell() *Cell {
	return &Cell{Kind: KindEmpty}
}

// textCell is a text, "—" when empty.
func textCell(text string) *Cell {
	if text == "" {
		return emptyCell()
	}
	return &Cell{Kind: KindText, Text: text}
}

// moneyCell is an amount in USD shown in the display currency; red when negative. "—" for no amount.
func moneyCell(cur Currency, usd *money.Amount) *Cell {
	if usd == nil {
		return emptyCell()
	}
	m := cur.money(*usd)
	cell := &Cell{Kind: KindMoney, Money: &m}
	if m.Amount.Sign() < 0 {
		cell.Tone = ToneProblem
	}
	return cell
}

// planMoneyCell is an amount in its plan's currency, as it is (016 FR-037); red when negative, "—" for none.
func planMoneyCell(m *money.Money) *Cell {
	if m == nil {
		return emptyCell()
	}
	shown := *m
	cell := &Cell{Kind: KindMoney, Money: &shown}
	if shown.Amount.Sign() < 0 {
		cell.Tone = ToneProblem
	}
	return cell
}

// numberCell is a whole number, "—" for none.
func numberCell(n *int64) *Cell {
	if n == nil {
		return emptyCell()
	}
	v := *n
	return &Cell{Kind: KindNumber, Number: &v}
}

// dateCell is a date.
func dateCell(d date.Date) *Cell {
	return &Cell{Kind: KindDate, Date: d}
}

// optionalDateCell is a date, "—" for none.
func optionalDateCell(d *date.Date) *Cell {
	if d == nil {
		return emptyCell()
	}
	return dateCell(*d)
}

// dateTimeCell is a moment in UTC, "—" for none.
func dateTimeCell(t time.Time) *Cell {
	if t.IsZero() {
		return emptyCell()
	}
	utc := t.UTC()
	return &Cell{Kind: KindDateTime, DateTime: &utc}
}

// statusCell is a status chip, "—" for none.
func statusCell(s *Status) *Cell {
	if s == nil {
		return emptyCell()
	}
	return &Cell{Kind: KindStatus, Text: s.Label, Tone: s.Tone}
}
