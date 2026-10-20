package money

// Display converts amounts in US dollars into a person's display currency (feature 003, D-061): usd × rate,
// rounded half-up to the cent. Features work every sum out in USD first and convert each amount they
// answer once, so a total can differ by a cent from the sum of its converted parts.
type Display struct {
	currency string
	rate     Rate
}

// DisplayUSD shows amounts as they are, in US dollars.
var DisplayUSD = Display{currency: BaseCurrency, rate: One}

// DisplayIn returns the Display of the currency code with its rate in units per 1 USD as decimal text
// ("0.92"). An empty code, USD or a rate that is not above 0 shows US dollars.
func DisplayIn(code, rate string) Display {
	r, err := ParseRate(rate)
	if err != nil || code == "" || code == BaseCurrency {
		return DisplayUSD
	}
	return Display{currency: code, rate: r}
}

// Currency is the ISO code amounts are shown in.
func (d Display) Currency() string {
	if d.currency == "" {
		return BaseCurrency
	}
	return d.currency
}

// Money converts usd, an amount in US dollars.
func (d Display) Money(usd Money) Money {
	if d.currency == "" || d.currency == BaseCurrency {
		return usd
	}
	return Money{Amount: FromUSD(usd.Amount, d.rate), Currency: d.currency}
}

// Optional converts usd, or gives nil when there is no amount.
func (d Display) Optional(usd *Money) *Money {
	if usd == nil {
		return nil
	}
	converted := d.Money(*usd)
	return &converted
}
