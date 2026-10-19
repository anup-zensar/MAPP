package money

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// ErrInvalidRate means a text is not an exchange rate: not a decimal number, more than six decimals, or
// not above 0.
var ErrInvalidRate = errors.New("money: not an exchange rate above 0 with at most six decimals")

// microsPerUnit is the number of millionths in one unit.
const microsPerUnit = 1_000_000

// maxRateMicros is the largest rate: 1,000,000 units per 1 USD (D-053).
const maxRateMicros int64 = 1_000_000 * microsPerUnit

// Rate is an exchange rate: units of a currency per 1 USD (D-010), exact to six decimals (millionths
// inside). EUR at 0.92 means 1 USD = 0.92 EUR. In JSON it is a string without trailing zeros, "0.92". The
// zero value is not a rate: every rate comes from ParseRate, One or a stored record.
type Rate struct {
	micros int64
}

// One is the rate 1, the rate of USD.
var One = Rate{micros: microsPerUnit}

// ParseRate reads a rate: digits with at most six decimals, above 0 ("0.92", "150", "1.086957").
// Anything else, 0 included, is ErrInvalidRate; above 1,000,000 is ErrOutOfRange.
func ParseRate(s string) (Rate, error) {
	if strings.HasPrefix(s, "-") {
		return Rate{}, ErrInvalidRate
	}
	micros, err := parseFixed(s, 6, 7)
	switch {
	case errors.Is(err, ErrOutOfRange):
		return Rate{}, ErrOutOfRange
	case err != nil:
		return Rate{}, ErrInvalidRate
	case micros <= 0:
		return Rate{}, ErrInvalidRate
	case micros > maxRateMicros:
		return Rate{}, ErrOutOfRange
	}
	return Rate{micros: micros}, nil
}

// RateFromMicros returns the rate of m millionths: RateFromMicros(920000) is 0.92.
func RateFromMicros(m int64) Rate {
	return Rate{micros: m}
}

// Micros returns the rate in millionths: 0.92 is 920000.
func (r Rate) Micros() int64 {
	return r.micros
}

// String writes the rate without trailing zeros: "0.92", "150", "1.086957".
func (r Rate) String() string {
	return strings.TrimSuffix(strings.TrimRight(fixedText(r.micros, 6, false), "0"), ".")
}

// Format writes the rate as people read it: as String, with thousands separators in the integer part
// ("0.92", "1,000,000"). The activity log uses it.
func (r Rate) Format() string {
	return strings.TrimSuffix(strings.TrimRight(fixedText(r.micros, 6, true), "0"), ".")
}

// Inverse returns 1 ÷ r rounded half-up to six decimals: USD per one unit of the currency. 0.92 → 1.086957,
// 150 → 0.006667.
func (r Rate) Inverse() Rate {
	return Rate{micros: roundDiv(big.NewInt(microsPerUnit*microsPerUnit), big.NewInt(r.micros))}
}

// IsOne reports whether the rate is 1 (the rate of USD).
func (r Rate) IsOne() bool {
	return r.micros == microsPerUnit
}

// MarshalJSON writes the rate as a JSON string without trailing zeros: "0.92".
func (r Rate) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

// UnmarshalJSON reads a rate from a JSON string with ParseRate. JSON null leaves the rate as it is.
func (r *Rate) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("money: a rate is a JSON string such as \"0.92\": %w", err)
	}
	parsed, err := ParseRate(text)
	if err != nil {
		return err
	}
	*r = parsed
	return nil
}

// ToUSD converts an amount in a currency to USD with the currency's rate r: local ÷ r, rounded half-up to
// the cent (domain model §4.3). 1,000.00 EUR at 0.92 → 1,086.96 USD. r must be a rate (above 0).
func ToUSD(local Amount, r Rate) Amount {
	num := new(big.Int).Mul(big.NewInt(local.cents), big.NewInt(microsPerUnit))
	return Amount{cents: roundDiv(num, big.NewInt(r.micros))}
}

// FromUSD converts an amount in USD to a currency with its rate r: usd × r, rounded half-up to the cent
// (display currencies, feature 003). 1,086.96 USD at 0.92 → 1,000.00 EUR.
func FromUSD(usd Amount, r Rate) Amount {
	num := new(big.Int).Mul(big.NewInt(usd.cents), big.NewInt(r.micros))
	return Amount{cents: roundDiv(num, big.NewInt(microsPerUnit))}
}

// ConvertSpread converts month amounts in a currency to USD so that they add up to the USD value of their
// sum (D-024): each part with ToUSD, except the last, which gets ToUSD(sum) − the sum of the others.
// 10,000.00 EUR as 3,333.33 + 3,333.33 + 3,333.34 at 0.92 → 3,623.18 + 3,623.18 + 3,623.21 = 10,869.57 USD.
// An empty list gives an empty list.
func ConvertSpread(parts []Amount, r Rate) []Amount {
	converted := make([]Amount, len(parts))
	if len(parts) == 0 {
		return converted
	}
	var others Amount
	for i, part := range parts[:len(parts)-1] {
		converted[i] = ToUSD(part, r)
		others = others.Add(converted[i])
	}
	converted[len(parts)-1] = ToUSD(Sum(parts...), r).Sub(others)
	return converted
}

// Rates is the current rate of every currency, active or not, by ISO code (D-027: one current rate per
// currency). Feature 002's settings.(*Service).Rates gives it.
type Rates map[string]Rate

// ToUSD converts m to USD with its currency's rate. USD is returned unchanged, also when the map has no
// rate for it; another currency without a rate is ErrUnknownCurrency, and a rate that is not above 0
// ErrInvalidRate.
func (rs Rates) ToUSD(m Money) (Amount, error) {
	if m.Currency == BaseCurrency {
		return m.Amount, nil
	}
	r, err := rs.rate(m.Currency)
	if err != nil {
		return Amount{}, err
	}
	return ToUSD(m.Amount, r), nil
}

// FromUSD converts an amount in USD to the currency code (display currencies). USD is returned unchanged;
// a currency without a rate is ErrUnknownCurrency.
func (rs Rates) FromUSD(usd Amount, code string) (Money, error) {
	if code == BaseCurrency {
		return USD(usd), nil
	}
	r, err := rs.rate(code)
	if err != nil {
		return Money{}, err
	}
	return Money{Amount: FromUSD(usd, r), Currency: code}, nil
}

// rate returns the rate of code: ErrUnknownCurrency when there is none, ErrInvalidRate when it is not
// above 0.
func (rs Rates) rate(code string) (Rate, error) {
	r, found := rs[code]
	if !found {
		return Rate{}, fmt.Errorf("%w: %q", ErrUnknownCurrency, code)
	}
	if r.micros <= 0 {
		return Rate{}, fmt.Errorf("%w: the rate of %q", ErrInvalidRate, code)
	}
	return r, nil
}
