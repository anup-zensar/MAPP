package spa

import (
	"fmt"
	"time"

	"mapp/internal/platform/money"
)

// span is the months from first to last (0–11) of the plan's year: one month or one quarter.
type span struct {
	key, label  string
	first, last int
}

func monthSpans(year int) []span {
	spans := make([]span, 0, 12)
	for i := range 12 {
		label := fmt.Sprintf("%s %d", time.Month(i + 1).String()[:3], year)
		spans = append(spans, span{key: monthKey(year, i), label: label, first: i, last: i})
	}
	return spans
}

func quarterSpans(year int) []span {
	spans := make([]span, 0, 4)
	for q := range 4 {
		spans = append(spans, span{key: fmt.Sprintf("%d-Q%d", year, q+1), label: fmt.Sprintf("Q%d %d", q+1, year),
			first: q * 3, last: q*3 + 2})
	}
	return spans
}

func (p span) planned(a *activity) money.Amount {
	return money.Sum(a.months[p.first : p.last+1]...)
}

func (p span) actual(a *activity) money.Amount {
	return money.Sum(a.actual[p.first : p.last+1]...)
}

// periodsOf is FR-017: planned and actual by quarter and by month, with what is not scheduled.
func periodsOf(acts []*activity, year int, sees bool, s shower) Periods {
	notScheduled := sumOf(acts, func(a *activity) money.Amount { return a.notScheduled })
	p := Periods{
		Quarters: periodList(acts, quarterSpans(year), sees, s), Months: periodList(acts, monthSpans(year), sees, s),
		NotScheduledTrace: s.trace(acts, func(a *activity) money.Amount { return a.notScheduled }),
	}
	if !notScheduled.IsZero() {
		m := s.money(notScheduled)
		p.NotScheduled = &m
	}
	return p
}

// periodList gives every period its bars on one scale: the largest planned or actual amount of the list.
func periodList(acts []*activity, spans []span, sees bool, s shower) []Period {
	planned, actual := make([]money.Amount, len(spans)), make([]money.Amount, len(spans))
	var largest money.Amount
	for i, p := range spans {
		planned[i], actual[i] = sumOf(acts, p.planned), sumOf(acts, p.actual)
		if !sees {
			actual[i] = money.Amount{}
		}
		largest = maxAmount(largest, maxAmount(planned[i], actual[i]))
	}
	list := make([]Period, 0, len(spans))
	for i, p := range spans {
		period := Period{
			Key: p.key, Label: p.label, Planned: s.money(planned[i]), Actual: s.financial(sees, actual[i]),
			PlannedBar: barOf(planned[i], largest), ActualBar: barOf(actual[i], largest),
			PlannedTrace: s.trace(acts, p.planned), ActualTrace: []Contribution{},
		}
		if sees {
			period.ActualTrace = s.trace(acts, p.actual)
		}
		list = append(list, period)
	}
	return list
}

func maxAmount(a, b money.Amount) money.Amount {
	if a.Cmp(b) >= 0 {
		return a
	}
	return b
}
