package purchaserequest

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

func canSee(ctx context.Context, f LineItemFacts) bool {
	return authz.CanViewActivity(ctx, f.Access)
}

// lines checks the lines of a save; a line with an id changes that stored line.
func (c *checker) lines(ctx context.Context, inputs []LineInput) ([]Line, error) {
	switch {
	case len(inputs) == 0:
		c.problems.Add(fieldLines, msgAddLine)
	case len(inputs) > maxLines:
		c.problems.Add(fieldLines, msgTooManyLines)
		return nil, nil
	}
	lines := make([]Line, 0, len(inputs))
	seen := map[string]bool{}
	for i, in := range inputs {
		field := fmt.Sprintf("lines[%d].", i)
		l := c.stored(in.ID, field)
		if l.ID != "" && l.LineItemID != in.LineItemID {
			c.problems.Add(field+"lineItemId", msgLineItemFixed)
		}
		l.Position, l.LineItemID = i+1, in.LineItemID
		c.lineItem(ctx, field, l, seen)
		if err := c.values(ctx, field, in, &l); err != nil {
			return nil, err
		}
		if err := c.claim(field, l); err != nil {
			return nil, err
		}
		lines = append(lines, l)
	}
	c.companyCodes(lines)
	return lines, nil
}

// stored returns the stored line id of the purchase request, or a new line when id is "".
func (c *checker) stored(id, field string) Line {
	if id == "" {
		return Line{}
	}
	for _, l := range c.before.Lines {
		if l.ID == id {
			return l
		}
	}
	c.problems.Add(field+"id", msgUnknownLine)
	return Line{}
}

func (c *checker) lineItem(ctx context.Context, field string, l Line, seen map[string]bool) {
	f, found := c.facts[l.LineItemID]
	switch {
	case l.LineItemID == "":
		c.problems.Add(field+"lineItemId", msgLineItemNeeded)
	case !found || !canSee(ctx, f):
		c.problems.Add(field+"lineItemId", msgLineItemGone)
	case seen[l.LineItemID]:
		c.problems.Add(field+"lineItemId", fmt.Sprintf(msgOnceOnPR, l.LineItemID))
	case f.Plan.Access.ID != c.plan.Access.ID:
		c.problems.Add(field+"lineItemId", fmt.Sprintf(msgOtherPlan, l.LineItemID))
	}
	seen[l.LineItemID] = true
}

// values reads the line's currency, amounts, delivery date, description and Shell attributes.
func (c *checker) values(ctx context.Context, field string, in LineInput, l *Line) error {
	if err := c.currency(ctx, field, in.Currency, l); err != nil {
		return err
	}
	amount, err := money.ParseAmount(strings.TrimSpace(in.Amount))
	if err != nil || amount.Sign() <= 0 {
		c.problems.Add(field+"amount", msgEnterAmount)
	}
	l.Amount = amount
	if l.DeliveryDate, err = date.Parse(strings.TrimSpace(in.DeliveryDate)); err != nil {
		c.problems.Add(field+"deliveryDate", msgEnterDate)
	}
	l.OverallLimit = c.overallLimit(field, in.OverallLimit)
	l.Description = strings.TrimSpace(in.Description)
	c.tooLong(field+"description", l.Description)
	result, err := c.s.deps.Attributes.Check(ctx, settings.LevelPurchaseRequestLine, settings.AttributeInput{
		Values: in.Attributes, Previous: l.Attributes, BusinessUnit: c.plan.businessUnit(c.facts),
	})
	if err != nil {
		return err
	}
	for _, p := range result.Problems {
		c.problems.Add(field+p.Field, p.Message)
	}
	l.Attributes = result.Values
	return nil
}

// currency accepts an active currency, or the inactive one the line already has.
func (c *checker) currency(ctx context.Context, field, code string, l *Line) error {
	cur, found, err := c.s.deps.Currencies.Currency(ctx, code)
	if err != nil {
		return err
	}
	if !found || (!cur.Active && code != l.Currency) {
		c.problems.Add(field+"currency", msgChooseCurrency)
	}
	l.Currency = code
	return nil
}

func (c *checker) overallLimit(field string, text *string) *money.Amount {
	if text == nil || strings.TrimSpace(*text) == "" {
		return nil
	}
	limit, err := money.ParseAmount(strings.TrimSpace(*text))
	if err != nil || limit.Sign() < 0 {
		c.problems.Add(field+"overallLimit", msgEnterLimit)
		return nil
	}
	return &limit
}

// savedCodes are the filled company codes of the lines already saved.
func (c *checker) savedCodes(lines []Line) []string {
	var codes []string
	for _, l := range lines {
		code := c.facts[l.LineItemID].CompanyCode
		if l.ID != "" && code != "" && !slices.Contains(codes, code) {
			codes = append(codes, code)
		}
	}
	return codes
}

// claim checks the line's amount fits what its line item has left, its stored lines left out.
func (c *checker) claim(field string, l Line) error {
	f, found := c.facts[l.LineItemID]
	if !found || l.Amount.Sign() <= 0 || l.Currency == "" {
		return nil
	}
	problem, err := c.balance.claimProblem(f, l.money(), c.before.ID)
	if err != nil || problem == "" {
		return err
	}
	c.problems.Add(field+"amount", problem)
	return nil
}

// companyCodes refuses a line added to lines of another filled company code; lines already saved keep
// theirs, the submission gate checks them all (edge case "company code changes").
func (c *checker) companyCodes(lines []Line) {
	codes := c.savedCodes(lines)
	for i, l := range lines {
		code := c.facts[l.LineItemID].CompanyCode
		if l.ID != "" || code == "" {
			continue
		}
		if len(codes) > 0 && !slices.Contains(codes, code) {
			c.problems.Add(fmt.Sprintf("lines[%d].lineItemId", i), fmt.Sprintf(msgTwoCompanyCodes, codes[0], code))
			continue
		}
		codes = append(codes, code)
	}
}
