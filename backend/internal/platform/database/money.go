package database

import "mapp/internal/platform/money"

// MoneyColumns is an optional amount with its currency as two columns that are both NULL when it is nil.
func MoneyColumns(m *money.Money) (minor *int64, currency *string) {
	if m == nil {
		return nil, nil
	}
	cents := m.Amount.Cents()
	return &cents, &m.Currency
}

func MoneyOf(minor *int64, currency *string) *money.Money {
	if minor == nil || currency == nil {
		return nil
	}
	return &money.Money{Amount: money.FromCents(*minor), Currency: *currency}
}
