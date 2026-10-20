package money

import (
	"fmt"
	"math/big"
)

// Convert converts the amount a between two currencies given their rates in units per 1 USD (016 D-262):
// a × to ÷ from, rounded half up to the cent once. The same rate on both sides gives a unchanged, so an
// amount in the plan currency never moves. Convert(a, r, One) is ToUSD(a, r) and Convert(a, One, r) is
// FromUSD(a, r). 23,000.00 EUR (0.92) into GBP (0.79) → 19,750.00. Both rates must be above 0.
func Convert(a Amount, from, to Rate) Amount {
	if from == to {
		return a
	}
	num := new(big.Int).Mul(big.NewInt(a.cents), big.NewInt(to.micros))
	return Amount{cents: roundDiv(num, big.NewInt(from.micros))}
}

// ConvertParts converts parts — month amounts of one total — so that they add up to the conversion of
// their sum (D-024): every part but the last with Convert, the last gets Convert(sum) − the others. It
// generalises ConvertSpread: ConvertParts(parts, r, One) == ConvertSpread(parts, r). An empty list gives an
// empty list.
func ConvertParts(parts []Amount, from, to Rate) []Amount {
	converted := make([]Amount, len(parts))
	if len(parts) == 0 {
		return converted
	}
	var others Amount
	for i, part := range parts[:len(parts)-1] {
		converted[i] = Convert(part, from, to)
		others = others.Add(converted[i])
	}
	converted[len(parts)-1] = Convert(Sum(parts...), from, to).Sub(others)
	return converted
}

// Convert converts m into the currency code with the rates rs (USD needs no rate). An amount already in
// code is returned unchanged; a currency without a rate is ErrUnknownCurrency.
func (rs Rates) Convert(m Money, code string) (Money, error) {
	if m.Currency == code {
		return m, nil
	}
	from, err := rs.rateOrOne(m.Currency)
	if err != nil {
		return Money{}, err
	}
	to, err := rs.rateOrOne(code)
	if err != nil {
		return Money{}, err
	}
	return Money{Amount: Convert(m.Amount, from, to), Currency: code}, nil
}

// Of returns the rate of code: 1 for USD (also when the map has none), otherwise as rate does.
func (rs Rates) Of(code string) (Rate, error) {
	return rs.rateOrOne(code)
}

// rateOrOne returns the rate of code, One for USD.
func (rs Rates) rateOrOne(code string) (Rate, error) {
	if code == BaseCurrency {
		return One, nil
	}
	r, err := rs.rate(code)
	if err != nil {
		return Rate{}, fmt.Errorf("convert: %w", err)
	}
	return r, nil
}
