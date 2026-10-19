package money

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// ErrInvalidPercent means a text is not a percentage: not digits, more than three integer digits, or more
// than two decimals.
var ErrInvalidPercent = errors.New("money: not a percentage with at most three digits and two decimals")

// Percent is a percentage with two decimals (hundredths of a percent inside): the conversion rates of a
// model and the shares of a % split. In JSON it is a string with two decimals, "2.00". The zero value is
// 0.00 %.
type Percent struct {
	hundredths int64
}

// Hundred is 100.00 %.
var Hundred = Percent{hundredths: 10000}

// ParsePercent reads a percentage: at most three integer digits and at most two decimals, never negative
// ("2", "2.5", "33.33", "100.00"). Anything else is ErrInvalidPercent. Ranges such as "above 0 and at most
// 100" are the caller's rule.
func ParsePercent(s string) (Percent, error) {
	if strings.HasPrefix(s, "-") {
		return Percent{}, ErrInvalidPercent
	}
	hundredths, err := parseFixed(s, 2, 3)
	if err != nil {
		return Percent{}, ErrInvalidPercent
	}
	return Percent{hundredths: hundredths}, nil
}

// PercentFromHundredths returns the percentage of h hundredths of a percent: 200 is 2.00 %.
func PercentFromHundredths(h int64) Percent {
	return Percent{hundredths: h}
}

// Hundredths returns the percentage in hundredths of a percent: 2.00 % is 200.
func (p Percent) Hundredths() int64 {
	return p.hundredths
}

// String writes the percentage with two decimals: "2.00".
func (p Percent) String() string {
	return fixedText(p.hundredths, 2, false)
}

// Format writes the percentage as people read it: "2.00 %". The activity log uses it.
func (p Percent) Format() string {
	return p.String() + " %"
}

// Cmp compares p and q: −1, 0 or +1.
func (p Percent) Cmp(q Percent) int {
	switch {
	case p.hundredths < q.hundredths:
		return -1
	case p.hundredths > q.hundredths:
		return 1
	}
	return 0
}

// IsZero reports whether the percentage is 0.00 %.
func (p Percent) IsZero() bool {
	return p.hundredths == 0
}

// OfWhole returns p % of n, rounded half-up to a whole number (domain model §4.5): 2.00 % of 250,000 →
// 5,000; 2.00 % of 25 = 0.5 → 1.
func (p Percent) OfWhole(n int64) int64 {
	num := new(big.Int).Mul(big.NewInt(n), big.NewInt(p.hundredths))
	return roundDiv(num, big.NewInt(10000))
}

// PercentSplitEvenly splits 100.00 % into n shares: each rounded down to the hundredth, the remainder in
// the last share (R10). 3 → 33.33, 33.33, 33.34. n < 1 gives an empty list.
func PercentSplitEvenly(n int) []Percent {
	if n < 1 {
		return []Percent{}
	}
	share := Hundred.hundredths / int64(n)
	shares := make([]Percent, n)
	for i := range shares {
		shares[i] = Percent{hundredths: share}
	}
	shares[n-1] = Percent{hundredths: Hundred.hundredths - share*int64(n-1)}
	return shares
}

// SumPercent returns the sum of the percentages; the sum of nothing is 0.00 %.
func SumPercent(ps ...Percent) Percent {
	var total Percent
	for _, p := range ps {
		total.hundredths += p.hundredths
	}
	return total
}

// MarshalJSON writes the percentage as a JSON string with two decimals: "2.00".
func (p Percent) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

// UnmarshalJSON reads a percentage from a JSON string with ParsePercent. JSON null leaves it as it is.
func (p *Percent) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("money: a percentage is a JSON string such as \"2.00\": %w", err)
	}
	parsed, err := ParsePercent(text)
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}
