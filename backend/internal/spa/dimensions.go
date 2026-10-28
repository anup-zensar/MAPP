package spa

import (
	"cmp"
	"math/big"
	"slices"
	"strings"

	"mapp/internal/platform/money"
)

const (
	keyNotFilledIn   = "notFilledIn"
	keyNotItemised   = "notYetItemised"
	labelNotFilledIn = "Not filled in"
	labelNotItemised = "Not yet itemised"
	valueKeyPrefix   = "value:"
	sharesDecimals   = 2
	percentsDecimals = 1
	msgChooseListed  = "Choose one of the dimensions listed."
	fieldDimension   = "dimension"
)

// piece is a part of an activity's planned cost with the values it is grouped by.
type piece struct {
	activity *activity
	amount   money.Amount
	values   []Value
	special  string // keyNotItemised for the planned cost not on a line item
}

// group is one row of the breakdown while it is worked out.
type group struct {
	key, label string
	amount     money.Amount
	parts      map[*activity]money.Amount
	special    bool
}

// breakdownOf is FR-018: the scope's planned cost by the values of dim (D-336).
func breakdownOf(acts []*activity, dim Dimension, s shower) DimensionBreakdown {
	groups, countsInEach := map[string]*group{}, false
	for _, p := range piecesOf(acts, dim) {
		for _, part := range splitPiece(p) {
			g := groups[part.key]
			if g == nil {
				g = &group{key: part.key, label: part.label, parts: map[*activity]money.Amount{}, special: part.special}
				groups[part.key] = g
			}
			g.amount = g.amount.Add(part.amount)
			g.parts[p.activity] = g.parts[p.activity].Add(part.amount)
		}
		countsInEach = countsInEach || severalWithoutShares(p.values)
	}
	total := sumOf(acts, plannedOf)
	return DimensionBreakdown{Key: dim.Key, Label: dim.Label, Rows: rowsOf(groups, acts, total, s), Total: s.money(total),
		CountsInEach: countsInEach}
}

func piecesOf(acts []*activity, dim Dimension) []piece {
	var list []piece
	for _, a := range acts {
		if dim.Level != LevelLineItem {
			list = append(list, piece{activity: a, amount: a.planned, values: a.rec.Values[dim.Key]})
			continue
		}
		rest := a.planned
		for _, li := range a.lineItems {
			list = append(list, piece{activity: a, amount: li.planned, values: li.rec.Values[dim.Key]})
			rest = rest.Sub(li.planned)
		}
		list = append(list, piece{activity: a, amount: rest, special: keyNotItemised})
	}
	return list
}

type valuePart struct {
	key, label string
	amount     money.Amount
	special    bool
}

// splitPiece gives each value its part: a % split by its share (the last takes the rest), several values
// without shares the whole amount each, as All work's Breakdown (D-270).
func splitPiece(p piece) []valuePart {
	switch {
	case p.special != "":
		return []valuePart{{key: p.special, label: labelNotItemised, amount: p.amount, special: true}}
	case len(p.values) == 0:
		return []valuePart{{key: keyNotFilledIn, label: labelNotFilledIn, amount: p.amount, special: true}}
	}
	parts := make([]valuePart, 0, len(p.values))
	given := money.Amount{}
	for i, v := range p.values {
		amount := p.amount
		if v.Share != nil && i < len(p.values)-1 {
			amount = shareOfAmount(p.amount, *v.Share)
			given = given.Add(amount)
		} else if v.Share != nil {
			amount = p.amount.Sub(given)
		}
		parts = append(parts, valuePart{key: valueKeyPrefix + v.Label, label: v.Label, amount: amount})
	}
	return parts
}

func severalWithoutShares(values []Value) bool {
	return len(values) > 1 && values[0].Share == nil
}

func shareOfAmount(a money.Amount, p money.Percent) money.Amount {
	num := new(big.Int).Mul(big.NewInt(a.Cents()), big.NewInt(p.Hundredths()))
	return money.FromCents(roundHalfAway(num, big.NewInt(10000)).Int64())
}

// rowsOf leaves out rows of 0.00 and orders the rest largest first, Not filled in and Not yet itemised last.
func rowsOf(groups map[string]*group, acts []*activity, total money.Amount, s shower) []DimensionRow {
	list := make([]*group, 0, len(groups))
	var largest money.Amount
	for _, g := range groups {
		if !g.amount.IsZero() {
			list = append(list, g)
			largest = maxAmount(largest, g.amount)
		}
	}
	slices.SortFunc(list, func(x, y *group) int {
		return cmp.Or(compareSpecial(x, y), y.amount.Cmp(x.amount), strings.Compare(x.label, y.label))
	})
	rows := make([]DimensionRow, 0, len(list))
	for _, g := range list {
		share := percentOf(g.amount, total, sharesDecimals)
		if share == nil {
			share = text("0.00")
		}
		rows = append(rows, DimensionRow{Key: g.key, Label: g.label, Amount: s.money(g.amount), Share: *share,
			Bar: barOf(g.amount, largest), Trace: s.trace(acts, func(a *activity) money.Amount { return g.parts[a] })})
	}
	return rows
}

func compareSpecial(x, y *group) int {
	switch {
	case x.special == y.special:
		return 0
	case x.special:
		return 1
	}
	return -1
}
