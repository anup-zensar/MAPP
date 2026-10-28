package app

import (
	"context"
	"encoding/json"
	"slices"

	"mapp/internal/financials"
	"mapp/internal/people"
	"mapp/internal/platform/money"
	"mapp/internal/purchaserequest"
	"mapp/internal/settings"
	"mapp/internal/stores"
)

// prPeople implements purchaserequest.People over the people store and service.
type prPeople struct {
	repo    stores.PeopleStore
	service *people.Service
}

func (x prPeople) People(ctx context.Context, ids []string) (map[string]purchaserequest.Person, error) {
	found := map[string]purchaserequest.Person{}
	for _, id := range ids {
		if _, read := found[id]; read || id == "" {
			continue
		}
		p, ok, err := x.repo.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		if ok {
			found[id] = purchaserequest.Person{PersonRef: p.Ref(), Email: p.Email}
		}
	}
	return found, nil
}

func (x prPeople) DisplayRateOf(ctx context.Context, personID string) (people.DisplayRate, error) {
	return x.service.DisplayRateOf(ctx, personID)
}

// prAttributes implements purchaserequest.Attributes over Settings: checks, needed fields and the attributes
// MAPP reads by their role (032 data-model §4).
type prAttributes struct {
	settings *settings.Service
	repo     stores.SettingsStore
}

func (x prAttributes) Check(ctx context.Context, level settings.Level, in settings.AttributeInput) (settings.AttributeResult, error) {
	return x.settings.CheckAttributes(ctx, level, in)
}

func (x prAttributes) Missing(ctx context.Context, level settings.Level, values settings.AttributeValues, businessUnit []string) ([]settings.MissingField, error) {
	return x.settings.MissingAttributes(ctx, level, values, businessUnit)
}

func (x prAttributes) Describe(ctx context.Context, field string, value any) (string, string, error) {
	return x.settings.DescribeAttribute(ctx, field, value)
}

func (x prAttributes) Roles(ctx context.Context, values settings.AttributeValues, roles []string) (map[string][]string, error) {
	c, err := codingOf(ctx, x.repo)
	if err != nil {
		return nil, err
	}
	found := map[string][]string{}
	for _, role := range roles {
		if texts := c.texts(role, values); len(texts) > 0 {
			found[role] = texts
		}
	}
	return found, nil
}

func (x prAttributes) AttributeID(ctx context.Context, role string) (string, error) {
	c, err := codingOf(ctx, x.repo)
	return c.byRole[role].ID, err
}

// decodedValues reads an attribute value as texts: a text or option id, or a list of option ids.
func decodedValues(raw json.RawMessage) []string {
	var one string
	if json.Unmarshal(raw, &one) == nil {
		return []string{one}
	}
	var many []string
	if json.Unmarshal(raw, &many) == nil {
		return many
	}
	return nil
}

// prWorkflows implements purchaserequest.Workflows over the approval workflows (031 object types).
type prWorkflows struct {
	repo stores.ApprovalStore
}

func (x prWorkflows) Workflow(ctx context.Context, id string) (purchaserequest.WorkflowFacts, bool, error) {
	if id == "" {
		return purchaserequest.WorkflowFacts{}, false, nil
	}
	w, found, err := x.repo.GetWorkflow(ctx, id)
	if err != nil || !found {
		return purchaserequest.WorkflowFacts{}, false, err
	}
	return purchaserequest.WorkflowFacts{ID: w.ID, Name: w.Name, ObjectType: string(w.Kind()), Active: w.Active}, true, nil
}

// prInvoices implements purchaserequest.Invoices over the GSAP transactions (015): invoices and credit notes that
// refer to a PO line, in USD at today's rates.
type prInvoices struct {
	repo  stores.FinancialsStore
	rates func(ctx context.Context) (money.Rates, error)
}

func (x prInvoices) InvoicedOn(ctx context.Context, lines []purchaserequest.POLineRef) (map[purchaserequest.POLineRef]money.Amount, error) {
	invoiced := map[purchaserequest.POLineRef]money.Amount{}
	if len(lines) == 0 {
		return invoiced, nil
	}
	all, err := x.repo.ListTransactions(ctx)
	if err != nil {
		return nil, err
	}
	rates, err := x.rates(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range all {
		ref := purchaserequest.POLineRef{PONumber: t.PONumber, Line: t.POLineNumber}
		if t.Type != financials.TypeInvoice || !slices.Contains(lines, ref) {
			continue
		}
		usd, err := rates.ToUSD(money.Money{Amount: t.Amount, Currency: t.Currency})
		if err != nil {
			return nil, err
		}
		invoiced[ref] = invoiced[ref].Add(usd)
	}
	return invoiced, nil
}
