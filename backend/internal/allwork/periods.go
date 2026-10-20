package allwork

import (
	"fmt"
	"strconv"

	"northstar/internal/platform/money"
)

// Budget & cost by period (FR-022 – FR-024, data model §8). A line item's period amounts are its spread as
// feature 007 works it out, in USD; a positive unassigned balance is spread evenly over its activity's months
// (all in the start month of an ongoing activity), a negative one is not spread; activities add up their line
// items and unassigned row, containers the matching activities below them. Amounts are summed in USD —
// quarters too — and converted once to the display currency.

// The period keys besides the months and quarters.
const (
	periodOther   = "other"   // Custom amounts in months outside the plan's year
	periodNoDates = "noDates" // the amounts of an activity without dates
)

// monthLabels are the short month names of the period headings.
var monthLabels = [12]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

// buckets are amounts in USD by month of the year ("2026-03"), other or noDates. A bucket that exists holds
// an amount, 0.00 included; a missing one is "—".
type buckets map[string]money.Amount

// add adds a to the bucket key.
func (b buckets) add(key string, a money.Amount) {
	b[key] = b[key].Add(a)
}

// addAll adds every bucket of o.
func (b buckets) addAll(o buckets) {
	for key, a := range o {
		b.add(key, a)
	}
}

// periods are the period amounts of one view: per line item, activity and container, per activity's
// unassigned row, and the footer.
type periods struct {
	from, to   int                // the range of years (016 FR-035)
	of         map[*tnode]buckets // line items, activities and containers
	unassigned map[*tnode]buckets // the unassigned row of each activity
	footer     buckets            // the matching activities
}

// periodsOf works out the period amounts of the shown records of t for year (FR-023, FR-024).
func periodsOf(t *tree, from, to int) periods {
	if to < from {
		to = from
	}
	p := periods{from: from, to: to, of: map[*tnode]buckets{}, unassigned: map[*tnode]buckets{}, footer: buckets{}}
	for _, plan := range t.plans {
		p.addNode(plan)
	}
	return p
}

// addNode works out the buckets of n and everything below it, and adds the matching activities to the
// footer.
func (p periods) addNode(n *tnode) buckets {
	own := buckets{}
	p.of[n] = own
	if n.n.Level == LevelActivity {
		for _, li := range n.children {
			items := p.lineItemBuckets(li.n)
			p.of[li] = items
			own.addAll(items)
		}
		unassigned := p.unassignedBuckets(n)
		p.unassigned[n] = unassigned
		own.addAll(unassigned)
		if n.shown && n.match {
			p.footer.addAll(own)
		}
		return own
	}
	for _, c := range n.children {
		below := p.addNode(c)
		if c.n.Level != LevelActivity || (c.shown && c.match) {
			own.addAll(below)
		}
	}
	return own
}

// lineItemBuckets returns the period amounts of a line item: its spread by month, months outside the year
// under Other years, and what cannot be spread while its activity has no dates under No dates (FR-023).
func (p periods) lineItemBuckets(li *Node) buckets {
	b := buckets{}
	for _, m := range li.MonthsUSD {
		b.add(p.bucketOf(m.Month), m.USD)
	}
	if !li.UnspreadUSD.IsZero() {
		b.add(periodNoDates, li.UnspreadUSD)
	}
	return b
}

// unassignedBuckets returns the period amounts of an activity's unassigned planned cost (FR-023, domain
// model §4.4): a positive balance split evenly over the activity's months — rounded down to the cent, the
// remainder in the last month; all in the start month of an ongoing activity, under No dates without dates
// — and nothing for a negative one.
func (p periods) unassignedBuckets(a *tnode) buckets {
	b := buckets{}
	if a.balance == nil || a.balance.Sign() <= 0 {
		return b
	}
	if len(a.n.Months) == 0 {
		b.add(periodNoDates, *a.balance)
		return b
	}
	for i, part := range a.balance.SplitEvenly(len(a.n.Months)) {
		b.add(p.bucketOf(a.n.Months[i]), part)
	}
	return b
}

// bucketOf returns the bucket of a month ("2026-03"): itself in the range's years, Other years otherwise.
func (p periods) bucketOf(month string) string {
	if len(month) != 7 {
		return periodOther
	}
	year, err := strconv.Atoi(month[:4])
	if err != nil || year < p.from || year > p.to {
		return periodOther
	}
	return month
}

// columns returns the period columns of the view (FR-022) — the months of the year or its quarters, then
// Other years and No dates when some shown record holds an amount in them — and the buckets each column
// adds up: a month its own, a quarter its three months.
func (p periods) columns(period string) ([]Period, [][]string) {
	columns, buckets := []Period{}, [][]string{}
	several := p.to > p.from // labels name their year only over several years ("Mar 2026", "Q1 2026")
	for year := p.from; year <= p.to; year++ {
		month := func(m int) string { return fmt.Sprintf("%04d-%02d", year, m) }
		withYear := func(label string) string {
			if several {
				return fmt.Sprintf("%s %d", label, year)
			}
			return label
		}
		switch period {
		case PeriodYear: // 016 FR-035
			keys := make([]string, 0, 12)
			for m := 1; m <= 12; m++ {
				keys = append(keys, month(m))
			}
			columns = append(columns, Period{Key: strconv.Itoa(year), Label: strconv.Itoa(year)})
			buckets = append(buckets, keys)
		case PeriodQuarter:
			for q := 1; q <= 4; q++ {
				columns = append(columns, Period{Key: fmt.Sprintf("Q%d-%d", q, year), Label: withYear(fmt.Sprintf("Q%d", q))})
				buckets = append(buckets, []string{month(3*q - 2), month(3*q - 1), month(3 * q)})
			}
		default:
			for m, label := range monthLabels {
				columns = append(columns, Period{Key: month(m + 1), Label: withYear(label)})
				buckets = append(buckets, []string{month(m + 1)})
			}
		}
	}
	for _, extra := range []Period{{Key: periodOther, Label: "Other years"}, {Key: periodNoDates, Label: "No dates"}} {
		if _, used := p.footer[extra.Key]; used {
			columns = append(columns, extra)
			buckets = append(buckets, []string{extra.Key})
		}
	}
	return columns, buckets
}

// amounts returns the amounts of b in the columns whose buckets are given, each converted once to the
// display currency (a quarter is the sum of its months in USD), nil where b holds nothing; and b's total,
// nil when it holds nothing.
func (p periods) amounts(b buckets, columns [][]string, cur Currency) ([]*money.Money, *money.Money) {
	amounts := make([]*money.Money, len(columns))
	for i, keys := range columns {
		var sum *money.Amount
		for _, key := range keys {
			if a, found := b[key]; found {
				sum = addAmount(sum, &a)
			}
		}
		amounts[i] = cur.optional(sum)
	}
	var total *money.Amount
	for _, a := range b {
		total = addAmount(total, &a)
	}
	return amounts, cur.optional(total)
}

// phased returns the phased budget of the container t in each period column (016 FR-038), converted once to
// the display currency — nil in periods outside its plan's year and for a container that is not phased — and
// the indexes of the periods where the planned cost planned is above it.
func (p periods) phased(t *tnode, planned buckets, columns [][]string, cur Currency) ([]*money.Money, []int) {
	amounts := make([]*money.Money, len(columns))
	over := []int{}
	if len(t.n.PhasedBudgetUSD) == 0 {
		return amounts, over
	}
	phased := buckets{}
	for _, m := range t.n.PhasedBudgetUSD {
		phased.add(p.bucketOf(m.Month), m.USD)
	}
	for i, keys := range columns {
		var budget, cost *money.Amount
		for _, key := range keys {
			if a, found := phased[key]; found {
				budget = addAmount(budget, &a)
			}
			if a, found := planned[key]; found {
				cost = addAmount(cost, &a)
			}
		}
		amounts[i] = cur.optional(budget)
		if budget != nil && cost != nil && cost.Cmp(*budget) > 0 {
			over = append(over, i)
		}
	}
	return amounts, over
}
