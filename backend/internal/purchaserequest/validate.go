package purchaserequest

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// checker checks one save of a purchase request (data-model §3.2), collecting every problem at once.
type checker struct {
	s        *Service
	before   PurchaseRequest // the stored purchase request; its ID is "" on create
	facts    map[string]LineItemFacts
	balance  balance
	plan     PlanFacts
	problems errs.Problems
}

// checked is a save checked: the purchase request it turns into, the facts of its line items, its plan and
// every problem found; the caller adds its own before answering them all at once.
type checked struct {
	pr       PurchaseRequest
	facts    map[string]LineItemFacts
	balance  balance
	plan     PlanFacts
	problems errs.Problems
}

func (s *Service) check(ctx context.Context, in Input, before PurchaseRequest) (checked, error) {
	c, err := s.checkerFor(ctx, in, before)
	if err != nil {
		return checked{}, err
	}
	pr := before
	if err = c.header(ctx, in, &pr); err != nil {
		return checked{}, err
	}
	if pr.Lines, err = c.lines(ctx, in.Lines); err != nil {
		return checked{}, err
	}
	pr.PlanID = c.plan.Access.ID
	return checked{pr: pr, facts: c.facts, balance: c.balance, plan: c.plan, problems: c.problems}, nil
}

func (s *Service) checkerFor(ctx context.Context, in Input, before PurchaseRequest) (*checker, error) {
	ids := make([]string, 0, len(in.Lines)+len(before.Lines))
	for _, l := range in.Lines {
		ids = append(ids, l.LineItemID)
	}
	ids = append(ids, before.lineItemIDs()...)
	facts, err := s.deps.LineItems.Facts(ctx, ids)
	if err != nil {
		return nil, err
	}
	rates, err := s.deps.Currencies.Rates(ctx)
	if err != nil {
		return nil, err
	}
	claims, err := s.repo.Claims(ctx, ids)
	if err != nil {
		return nil, err
	}
	c := &checker{s: s, before: before, facts: facts, balance: balance{rates: rates, claims: claims}}
	c.plan = c.planOf(ctx, in.Lines)
	return c, nil
}

// planOf is the purchase request's plan: the stored one, or the plan of the first line item the person sees.
func (c *checker) planOf(ctx context.Context, lines []LineInput) PlanFacts {
	for _, l := range c.before.Lines {
		if f, found := c.facts[l.LineItemID]; found {
			return f.Plan
		}
	}
	for _, l := range lines {
		if f, found := c.facts[l.LineItemID]; found && canSee(ctx, f) {
			return f.Plan
		}
	}
	return PlanFacts{}
}

func (c *checker) header(ctx context.Context, in Input, pr *PurchaseRequest) error {
	pr.Name, pr.Description = strings.TrimSpace(in.Name), strings.TrimSpace(in.Description)
	switch {
	case pr.Name == "":
		c.problems.Add("name", msgEnterName)
	case utf8.RuneCountInString(pr.Name) > maxNameLength:
		c.problems.Add("name", msgNameTooLong)
	case record.CheckSingleLine(pr.Name) != "":
		c.problems.Add("name", record.CheckSingleLine(pr.Name))
	}
	c.tooLong("description", pr.Description)
	c.tooLong("comment", in.Comment)
	pr.OwnerID, pr.MoAApproverID = strings.TrimSpace(in.OwnerID), strings.TrimSpace(in.moaApprover())
	if err := c.people(ctx, pr); err != nil {
		return err
	}
	pr.WorkflowID, pr.ModificationWorkflowID = in.ApprovalWorkflowID, in.ModificationWorkflowID
	if err := c.workflow(ctx, pr.WorkflowID, KindPurchaseRequest); err != nil {
		return err
	}
	if err := c.workflow(ctx, pr.ModificationWorkflowID, KindModification); err != nil {
		return err
	}
	return c.attributes(ctx, in.Attributes, pr)
}

func (c *checker) tooLong(field, text string) {
	if utf8.RuneCountInString(text) > maxTextLength {
		c.problems.Add(field, msgTooLong2000)
	}
}

// people checks the owner and the MoA approver: active people, or the ones the purchase request already has.
func (c *checker) people(ctx context.Context, pr *PurchaseRequest) error {
	found, err := c.s.deps.People.People(ctx, []string{pr.OwnerID, pr.MoAApproverID})
	if err != nil {
		return err
	}
	if p, ok := found[pr.OwnerID]; !ok || (!p.Active && pr.OwnerID != c.before.OwnerID) {
		c.problems.Add(fieldOwner, msgChoosePerson)
	}
	if pr.MoAApproverID == "" {
		return nil
	}
	if p, ok := found[pr.MoAApproverID]; !ok || (!p.Active && pr.MoAApproverID != c.before.MoAApproverID) {
		c.problems.Add(fieldMoA, msgChoosePerson)
	}
	return nil
}

// workflow checks that id is an approval workflow of kind (031 object types), active unless the purchase
// request already has it: a workflow made inactive stops new submissions, not saves.
func (c *checker) workflow(ctx context.Context, id string, kind RoundKind) error {
	field, before := fieldWorkflow, c.before.WorkflowID
	if kind == KindModification {
		field, before = fieldModificationWorkflow, c.before.ModificationWorkflowID
	}
	w, found, err := c.s.deps.Workflows.Workflow(ctx, id)
	if err != nil {
		return err
	}
	if !found || w.ObjectType != string(kind) || (!w.Active && id != before) {
		c.problems.Add(field, fmt.Sprintf(msgChooseWorkflow, kindWords(kind)))
	}
	return nil
}

func kindWords(kind RoundKind) string {
	if kind == KindModification {
		return "changes of purchase requests"
	}
	return pluralNoun
}

// attributes checks the purchase request's Shell attributes, the vendor e-mail address, and copies
// the vendor's name onto the record for the list (research R2).
func (c *checker) attributes(ctx context.Context, values settings.AttributeValues, pr *PurchaseRequest) error {
	result, err := c.s.deps.Attributes.Check(ctx, settings.LevelPurchaseRequest, settings.AttributeInput{
		Values: values, Previous: c.before.Attributes, BusinessUnit: c.plan.businessUnit(c.facts),
	})
	if err != nil {
		return err
	}
	c.problems = append(c.problems, result.Problems...)
	pr.Attributes = result.Values
	roles, err := c.s.deps.Attributes.Roles(ctx, pr.Attributes, []string{settings.RoleVendor, settings.RoleVendorEmail})
	if err != nil {
		return err
	}
	pr.Vendor = first(roles[settings.RoleVendor])
	email := first(roles[settings.RoleVendorEmail])
	if email == "" || validEmail(email) {
		return nil
	}
	id, err := c.s.deps.Attributes.AttributeID(ctx, settings.RoleVendorEmail)
	c.problems.Add("attributes."+id, msgVendorEmail)
	return err
}

func validEmail(s string) bool {
	addr, err := mail.ParseAddress(s)
	return err == nil && addr.Address == s
}

func first(list []string) string {
	if len(list) == 0 {
		return ""
	}
	return list[0]
}

// businessUnit returns the Business unit of the plan of the line items, for attribute conditions.
func (p PlanFacts) businessUnit(facts map[string]LineItemFacts) []string {
	for _, f := range facts {
		if f.Plan.Access.ID == p.Access.ID {
			return f.BusinessUnit
		}
	}
	return nil
}
