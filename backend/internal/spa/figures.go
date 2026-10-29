package spa

import (
	"fmt"
	"math/big"

	"mapp/internal/platform/money"
)

type shower struct {
	display money.Display
}

func (s shower) money(usd money.Amount) money.Money {
	return s.display.Money(money.USD(usd))
}

func (s shower) financial(sees bool, usd money.Amount) *money.Money {
	if !sees {
		return nil
	}
	m := s.money(usd)
	return &m
}

// nonZero shows usd, or gives nil when it is zero.
func (s shower) nonZero(usd money.Amount) *money.Money {
	return s.financial(!usd.IsZero(), usd)
}

// trace lists the activities whose value is not zero, in table order (FR-007).
func (s shower) trace(acts []*activity, value func(*activity) money.Amount) []Contribution {
	list := []Contribution{}
	for _, a := range acts {
		if v := value(a); !v.IsZero() {
			list = append(list, Contribution{ActivityID: a.rec.ID, Name: a.rec.Name, Amount: s.money(v)})
		}
	}
	return list
}

func sumOf(acts []*activity, value func(*activity) money.Amount) money.Amount {
	var total money.Amount
	for _, a := range acts {
		total = total.Add(value(a))
	}
	return total
}

// percentOf is part × 100 ÷ whole with the given decimals, rounded half away from zero ("37.5", "-62.5");
// nil when whole is 0.00.
func percentOf(part, whole money.Amount, decimals int) *string {
	if whole.IsZero() {
		return nil
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	num := new(big.Int).Mul(big.NewInt(part.Cents()), new(big.Int).Mul(big.NewInt(100), scale))
	units := roundHalfAway(num, big.NewInt(whole.Cents()))
	sign := ""
	if units.Sign() < 0 {
		sign = "-"
		units.Neg(units)
	}
	integer, fraction := new(big.Int).QuoRem(units, scale, new(big.Int))
	return text(fmt.Sprintf("%s%s.%0*d", sign, integer, decimals, fraction.Int64()))
}

func barOf(amount, largest money.Amount) int {
	if largest.Sign() <= 0 || amount.Sign() <= 0 {
		return 0
	}
	bar := roundHalfAway(new(big.Int).Mul(big.NewInt(amount.Cents()), big.NewInt(100)), big.NewInt(largest.Cents()))
	return int(min(bar.Int64(), 100))
}

func roundHalfAway(num, den *big.Int) *big.Int {
	negative := num.Sign()*den.Sign() < 0
	n, d := new(big.Int).Abs(num), new(big.Int).Abs(den)
	q, r := new(big.Int).QuoRem(n, d, new(big.Int))
	if new(big.Int).Mul(r, big.NewInt(2)).Cmp(d) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	if negative {
		q.Neg(q)
	}
	return q
}

func text(s string) *string {
	return &s
}
