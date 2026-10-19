package allwork

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
)

// Sorting by a column (FR-015, data model §7): the children of every parent — and the plans — are ordered by
// the column, so the hierarchy stays intact. Records without a value come last in both directions, ties keep
// the plan tree's order, statuses sort in their lifecycle order, and an activity's unassigned row stays
// after its line items.

// sortKey is the value a record sorts by.
type sortKey struct {
	missing bool   // no value: last in both directions
	number  int64  // money in cents, numbers, per cents, dates, moments and lifecycle positions
	text    string // texts, ignoring letter case
	isText  bool
}

// sortTree orders the plans and the children of every record of t by the column c in direction (FR-015).
func sortTree(t *tree, c Column, direction string, cur Currency, year int) {
	keys := map[*tnode]sortKey{}
	key := func(n *tnode) sortKey {
		k, known := keys[n]
		if !known {
			k = sortKeyOf(c, n, cur, year)
			keys[n] = k
		}
		return k
	}
	order := func(x, y *tnode) int {
		kx, ky := key(x), key(y)
		switch {
		case kx.missing || ky.missing:
			return compareMissing(kx.missing, ky.missing)
		case direction == SortDescending:
			return compareKeys(ky, kx)
		}
		return compareKeys(kx, ky)
	}
	slices.SortStableFunc(t.plans, order)
	for _, n := range t.byID {
		slices.SortStableFunc(n.children, order)
	}
}

// compareMissing puts records without a value last.
func compareMissing(x, y bool) int {
	switch {
	case x == y:
		return 0
	case x:
		return 1
	}
	return -1
}

// compareKeys compares two values, ascending.
func compareKeys(x, y sortKey) int {
	if x.isText || y.isText {
		return strings.Compare(x.text, y.text)
	}
	return cmp.Compare(x.number, y.number)
}

// sortKeyOf returns the value the record n sorts by in the column c: its status's place in the lifecycle,
// or the value of its cell.
func sortKeyOf(c Column, n *tnode, cur Currency, year int) sortKey {
	switch c.Key {
	case "status":
		if n.n.Level == LevelLineItem {
			return sortKey{number: int64(n.n.Missing)}
		}
		return lifecycle(statusFamily(n.n.Level), n.n.Status)
	case "timing":
		return lifecycle(timingStatuses, n.n.Timing)
	case "completeness":
		return sortKey{number: int64(n.n.Missing)}
	case "lineItem.swbsStatus":
		return lifecycle(swbsStatuses, n.n.Swbs)
	}
	if !slices.Contains(c.Levels, n.n.Level) {
		return sortKey{missing: true}
	}
	cell := cellOf(c, cellRow{level: n.n.Level, t: n, year: year}, cur)
	switch cell.Kind {
	case KindMoney:
		return sortKey{number: cell.Money.Amount.Cents()}
	case KindNumber:
		return sortKey{number: *cell.Number}
	case KindPercent:
		percent, _ := strconv.ParseInt(cell.Percent, 10, 64)
		return sortKey{number: percent}
	case KindDate:
		return sortKey{number: cell.Date.Time().Unix()}
	case KindDateTime:
		return sortKey{number: cell.DateTime.UnixNano()}
	case KindText, KindStatus:
		if c.Kind == KindDate { // "Ongoing": no end date
			return sortKey{missing: true}
		}
		return sortKey{text: strings.ToLower(cell.Text), isText: true}
	}
	return sortKey{missing: true}
}

// lifecycle returns the place of value in its family's lifecycle (data model §7); no value comes last.
func lifecycle(family map[string]chip, value string) sortKey {
	c, known := family[value]
	if !known {
		return sortKey{missing: true}
	}
	return sortKey{number: int64(c.order)}
}
