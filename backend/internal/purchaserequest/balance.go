package purchaserequest

import (
	"fmt"

	"mapp/internal/platform/money"
)

// balance is what a line item has had requested, in its own currency at today's rates.
type balance struct {
	rates  money.Rates
	claims []Claim
}

func (b balance) requested(li LineItemFacts, except string) (money.Amount, error) {
	var total money.Amount
	for _, c := range b.claims {
		if c.LineItemID != li.ID || (except != "" && c.PurchaseRequestID == except) {
			continue
		}
		reserved, err := b.reserved(c, li.Currency)
		if err != nil {
			return money.Amount{}, err
		}
		total = total.Add(reserved)
	}
	return total, nil
}

// reserved is the larger of a claim's amount and its approved amount, both at today's rate.
func (b balance) reserved(c Claim, currency string) (money.Amount, error) {
	converted, err := b.rates.Convert(c.Amount, currency)
	if err != nil || c.Approved == nil {
		return converted.Amount, err
	}
	approved, err := b.rates.Convert(*c.Approved, currency)
	if err != nil {
		return money.Amount{}, err
	}
	if approved.Amount.Cmp(converted.Amount) > 0 {
		return approved.Amount, nil
	}
	return converted.Amount, nil
}

// available is the planned cost of li minus what other purchase requests than except request; negative when
// today's rates moved claims in other currencies above it.
func (b balance) available(li LineItemFacts, except string) (money.Amount, error) {
	requested, err := b.requested(li, except)
	if err != nil {
		return money.Amount{}, err
	}
	return li.PlannedCost.Sub(requested), nil
}

func (b balance) claimProblem(li LineItemFacts, amount money.Money, except string) (string, error) {
	left, err := b.available(li, except)
	if err != nil {
		return "", err
	}
	converted, err := b.rates.Convert(amount, li.Currency)
	if err != nil {
		return "", err
	}
	if converted.Amount.Cmp(left) <= 0 {
		return "", nil
	}
	message := fmt.Sprintf(msgOnlyAvailable, max0(left).Format(), li.Currency, li.ID)
	if amount.Currency != li.Currency {
		message += fmt.Sprintf(msgOtherCurrency, amount.Amount.Format(), amount.Currency, converted.Amount.Format(), li.Currency)
	}
	return message, nil
}

func max0(a money.Amount) money.Amount {
	if a.Sign() < 0 {
		return money.Amount{}
	}
	return a
}
