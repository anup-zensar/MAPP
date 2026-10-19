// Package money is NorthStar's exact money: amounts, exchange rates and percentages as exact decimals,
// never binary floating point (constitution VI; decisions D-010, D-024, D-027, D-052). Feature 002 owns it;
// every feature uses it.
//
//   - Amount: an exact amount in hundredths ("1234.50"). Every currency, JPY included, has two decimals.
//   - Money: an Amount with its ISO currency code, {"amount": "1234.50", "currency": "USD"} in JSON.
//   - Input: an amount as a request sends it, parsed with ParseInput so problems become field messages.
//   - Rate: units of a currency per 1 USD, exact to six decimals ("0.92").
//   - Rates: the one current rate of every currency, for conversions.
//   - Percent: a percentage with two decimals ("2.00"): model rates and % split shares.
//
// Rounding is half-up — for negative values half away from zero — and every product or quotient that could
// overflow is worked out with math/big. The package has no storage and no messages for people: callers turn
// its errors into their own field messages.
package money

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

// The errors of the package. Callers compare them with errors.Is and show their own messages.
var (
	// ErrInvalidAmount means a text is not an amount: not a decimal number, or more than two decimals.
	ErrInvalidAmount = errors.New("money: not an amount with at most two decimals")
	// ErrTooManyDecimals is the ErrInvalidAmount of a number with more than two decimals, so that a caller
	// can say "Use at most 2 decimals." instead of "Enter an amount".
	ErrTooManyDecimals = fmt.Errorf("%w: more than two decimals", ErrInvalidAmount)
	// ErrOutOfRange means a value is beyond what the package keeps exactly: an amount beyond
	// 999,999,999,999,999.99 either way, or a rate above 1,000,000.
	ErrOutOfRange = errors.New("money: value out of range")
	// ErrInvalidCurrency means a currency code is not three capital letters A–Z.
	ErrInvalidCurrency = errors.New("money: not a currency code of three letters A-Z")
	// ErrUnknownCurrency means Rates has no rate for a currency.
	ErrUnknownCurrency = errors.New("money: no rate for the currency")
)

// BaseCurrency is NorthStar's base currency: every rate is in units per 1 USD, and stored budgets and
// planned costs are in USD (D-010).
const BaseCurrency = "USD"

// maxCents is the largest amount kept: 999,999,999,999,999.99, fifteen integer digits and two decimals.
const maxCents int64 = 99_999_999_999_999_999

// amountText is the form of an amount: an optional minus, digits, and optionally a point and digits. The
// number of digits is checked afterwards, so that the error says what is wrong.
var amountText = regexp.MustCompile(`^(-?)([0-9]+)(?:\.([0-9]+))?$`)

// currencyCode is the form of an ISO currency code.
var currencyCode = regexp.MustCompile(`^[A-Z]{3}$`)

// Amount is an exact amount of money in hundredths (cents). The zero value is 0.00. In JSON it is a string
// with two decimals, "1234.50".
type Amount struct {
	cents int64
}

// ParseAmount reads an amount: an optional minus sign, at most fifteen integer digits, and at most two
// decimals ("1234", "1234.5", "-1234.50"). No spaces, signs other than a leading minus, or thousands
// separators. More than two decimals is ErrTooManyDecimals (an ErrInvalidAmount); more than fifteen integer
// digits (leading zeros not counted) is ErrOutOfRange.
func ParseAmount(s string) (Amount, error) {
	cents, err := parseFixed(s, 2, 15)
	if err != nil {
		return Amount{}, err
	}
	return Amount{cents: cents}, nil
}

// parseFixed reads a decimal text with at most decimals decimals and at most intDigits integer digits
// (leading zeros not counted), and returns it in units of 10^-decimals.
func parseFixed(s string, decimals, intDigits int) (int64, error) {
	parts := amountText.FindStringSubmatch(s)
	if parts == nil {
		return 0, ErrInvalidAmount
	}
	negative, whole, fraction := parts[1] == "-", strings.TrimLeft(parts[2], "0"), parts[3]
	if len(fraction) > decimals {
		return 0, ErrTooManyDecimals
	}
	if len(whole) > intDigits {
		return 0, ErrOutOfRange
	}
	digits := strings.TrimLeft(whole+fraction+strings.Repeat("0", decimals-len(fraction)), "0")
	if digits == "" {
		return 0, nil // zero, also "-0.00"
	}
	value, err := strconv.ParseInt(digits, 10, 64) // at most intDigits + decimals digits: always fits
	if err != nil {
		return 0, ErrOutOfRange
	}
	if negative {
		value = -value
	}
	return value, nil
}

// FromCents returns the amount of c hundredths: FromCents(123450) is 1,234.50.
func FromCents(c int64) Amount {
	return Amount{cents: c}
}

// Cents returns the amount in hundredths: 1,234.50 is 123450.
func (a Amount) Cents() int64 {
	return a.cents
}

// String writes the amount with two decimals and no thousands separators: "1234.50", "-0.05".
func (a Amount) String() string {
	return fixedText(a.cents, 2, false)
}

// Format writes the amount as people read it, with thousands separators: "1,234.50", "-1,000,000.00". The
// activity log and messages use it.
func (a Amount) Format() string {
	return fixedText(a.cents, 2, true)
}

// fixedText writes value (in units of 10^-decimals) with exactly decimals decimals, grouping the integer
// digits in threes when grouped.
func fixedText(value int64, decimals int, grouped bool) string {
	sign := ""
	magnitude := new(big.Int).SetInt64(value)
	if value < 0 {
		sign = "-"
		magnitude.Neg(magnitude)
	}
	digits := magnitude.String()
	if len(digits) <= decimals {
		digits = strings.Repeat("0", decimals-len(digits)+1) + digits
	}
	whole, fraction := digits[:len(digits)-decimals], digits[len(digits)-decimals:]
	if grouped {
		whole = groupThousands(whole)
	}
	if decimals == 0 {
		return sign + whole
	}
	return sign + whole + "." + fraction
}

// groupThousands puts a comma between every three digits from the right: "1234567" → "1,234,567".
func groupThousands(digits string) string {
	var b strings.Builder
	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Add returns a + b.
func (a Amount) Add(b Amount) Amount { return Amount{cents: a.cents + b.cents} }

// Sub returns a − b.
func (a Amount) Sub(b Amount) Amount { return Amount{cents: a.cents - b.cents} }

// Neg returns −a.
func (a Amount) Neg() Amount { return Amount{cents: -a.cents} }

// Cmp compares a and b: −1 when a < b, 0 when equal, +1 when a > b.
func (a Amount) Cmp(b Amount) int {
	switch {
	case a.cents < b.cents:
		return -1
	case a.cents > b.cents:
		return 1
	}
	return 0
}

// Sign returns −1, 0 or +1 for a negative, zero or positive amount.
func (a Amount) Sign() int { return a.Cmp(Amount{}) }

// IsZero reports whether the amount is 0.00.
func (a Amount) IsZero() bool { return a.cents == 0 }

// MulInt returns a × n exactly, or ErrOutOfRange when the product is beyond 999,999,999,999,999.99 either
// way (revenue = deals × average deal size).
func (a Amount) MulInt(n int64) (Amount, error) {
	product := new(big.Int).Mul(big.NewInt(a.cents), big.NewInt(n))
	if product.CmpAbs(big.NewInt(maxCents)) > 0 {
		return Amount{}, ErrOutOfRange
	}
	return Amount{cents: product.Int64()}, nil
}

// SplitEvenly splits a into n parts (n ≥ 1, a ≥ 0): each part rounded down to the cent, the remainder in
// the last part, so the parts always add up to a. 10,000.00 in 3 → 3,333.33, 3,333.33, 3,333.34; 0.02 in 3
// → 0.00, 0.00, 0.02 (domain model §4.4). n < 1 gives an empty list.
func (a Amount) SplitEvenly(n int) []Amount {
	if n < 1 {
		return []Amount{}
	}
	part := a.cents / int64(n)
	parts := make([]Amount, n)
	for i := range parts {
		parts[i] = Amount{cents: part}
	}
	parts[n-1] = Amount{cents: a.cents - part*int64(n-1)}
	return parts
}

// Sum returns the sum of the amounts; the sum of nothing is 0.00.
func Sum(as ...Amount) Amount {
	var total Amount
	for _, a := range as {
		total = total.Add(a)
	}
	return total
}

// MarshalJSON writes the amount as a JSON string with two decimals: "1234.50".
func (a Amount) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

// UnmarshalJSON reads a JSON string with at most two decimals ("1234", "1234.5", "1234.50"). JSON null
// leaves the amount as it is (a *Amount field becomes nil before this is called).
func (a *Amount) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("money: an amount is a JSON string such as \"1234.50\": %w", err)
	}
	parsed, err := ParseAmount(text)
	if err != nil {
		return err
	}
	*a = parsed
	return nil
}

// Money is an amount with its currency: the API shape of every amount in NorthStar,
// {"amount": "1234.50", "currency": "USD"}.
type Money struct {
	Amount   Amount `json:"amount"`
	Currency string `json:"currency"` // ISO code, "USD"
}

// USD returns a as an amount in US dollars.
func USD(a Amount) Money {
	return Money{Amount: a, Currency: BaseCurrency}
}

// Format writes the money as people read it: "1,234.50 EUR".
func (m Money) Format() string {
	return m.Amount.Format() + " " + m.Currency
}

// Input is an amount as a request body sends it (API schema MoneyInput): both fields as text, so that a
// wrong amount becomes a field message through ParseInput instead of an unreadable request.
type Input struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

// ParseInput reads a request's amount: the amount with ParseAmount (ErrInvalidAmount, ErrTooManyDecimals,
// ErrOutOfRange), then the currency code, which must be three capital letters (ErrInvalidCurrency). Whether
// the currency exists or is allowed is the caller's rule.
func ParseInput(in Input) (Money, error) {
	amount, err := ParseAmount(in.Amount)
	if err != nil {
		return Money{}, err
	}
	if !currencyCode.MatchString(in.Currency) {
		return Money{}, ErrInvalidCurrency
	}
	return Money{Amount: amount, Currency: in.Currency}, nil
}

// roundDiv returns num ÷ den rounded half-up, half away from zero for negative quotients: 5 ÷ 2 → 3,
// −5 ÷ 2 → −3, 4 ÷ 3 → 1. den must not be zero. The one rounding rule of the package.
func roundDiv(num, den *big.Int) int64 {
	quotient, remainder := new(big.Int).QuoRem(num, den, new(big.Int)) // truncated toward zero
	twice := new(big.Int).Mul(new(big.Int).Abs(remainder), big.NewInt(2))
	if twice.Cmp(new(big.Int).Abs(den)) >= 0 {
		if num.Sign()*den.Sign() < 0 {
			quotient.Sub(quotient, big.NewInt(1))
		} else {
			quotient.Add(quotient, big.NewInt(1))
		}
	}
	return quotient.Int64()
}
