package seed

import (
	"context"
	"fmt"
	"slices"

	"mapp/internal/approval"
	"mapp/internal/lineitem"
	"mapp/internal/platform/date"
	"mapp/internal/platform/ids"
	"mapp/internal/purchaserequest"
	"mapp/internal/settings"
)

const (
	prWorkflowName           = "Purchase request approval (MoA)"
	prChangeWorkflowName     = "Purchase request change approval (MoA)"
	seedPurchaseRequestOn    = "Agency fee (Ireland)"
	seedPurchaseRequestOwner = "Priya Raman"
	seedVendor               = "Brightside Media BV"
)

// addPurchaseRequests adds 032's demo data after the financial transactions (D-306): the Purchase Request
// Configuration, the two workflows whose approver is the MoA approver, and PR-1001, Created, on the UK plan's Agency fee (Ireland).
func (d *demo) addPurchaseRequests(ctx context.Context) error {
	repo := d.stores.PurchaseRequests
	if err := repo.PutConfiguration(ctx, purchaserequest.Configuration{PurchaseSupport: true,
		LinesOnApproved: purchaserequest.WithModificationApproval, Meta: d.meta()}); err != nil {
		return err
	}
	approvalID, err := d.addPRWorkflow(ctx, prWorkflowName, approval.KindPurchaseRequest,
		"The MoA approver named on the purchase request approves it; Alex Morgan is the backup.")
	if err != nil {
		return err
	}
	changeID, err := d.addPRWorkflow(ctx, prChangeWorkflowName, approval.KindModifiedPurchaseRequest,
		"The MoA approver approves changes of an approved purchase request; Alex Morgan is the backup.")
	if err != nil {
		return err
	}
	pr, err := d.seedPurchaseRequest(ctx, approvalID, changeID)
	if err != nil {
		return err
	}
	return repo.Put(ctx, pr)
}

func (d *demo) addPRWorkflow(ctx context.Context, name string, kind approval.SubjectKind, description string) (string, error) {
	alex, err := find(d.people, "person", "Alex Morgan")
	if err != nil {
		return "", err
	}
	id, err := d.next(ctx, ids.PrefixApprovalWorkflow)
	if err != nil {
		return "", err
	}
	w := approval.Workflow{ID: id, Name: name, Description: description, Active: true, ObjectType: kind, Meta: d.meta(),
		Levels: []approval.Level{{Name: "MoA approver", ApproverType: approval.SourceRecordField, ApproverIDs: []string{},
			RecordField: approval.RecordFieldMoAApprover, Mode: approval.ModeAny, BackupApproverID: alex}},
		ReminderDays: days(3), EscalationDays: days(5)}
	if err = d.stores.Approvals.PutWorkflow(ctx, w); err != nil {
		return "", err
	}
	d.workflows[name] = id
	return id, nil
}

// seedPurchaseRequest is PR-1001 "PR for Agency fee (Ireland)" (research R15): 23,000.00 EUR, vendor Brightside Media BV.
func (d *demo) seedPurchaseRequest(ctx context.Context, approvalID, changeID string) (purchaserequest.PurchaseRequest, error) {
	items, err := d.stores.LineItems.List(ctx)
	if err != nil {
		return purchaserequest.PurchaseRequest{}, err
	}
	i := slices.IndexFunc(items, func(li lineitem.LineItem) bool { return li.Name == seedPurchaseRequestOn })
	if i < 0 {
		return purchaserequest.PurchaseRequest{}, fmt.Errorf("seed: there is no line item called '%s'", seedPurchaseRequestOn)
	}
	var r resolver
	owner, moa := r.id(d.people, "person", seedPurchaseRequestOwner), r.id(d.people, "person", "Daniel Okoro")
	planID := r.id(d.plans, "plan", ukPlan)
	attributes := d.prAttributes(&r)
	if r.err != nil {
		return purchaserequest.PurchaseRequest{}, r.err
	}
	prID, err := d.next(ctx, ids.PrefixPurchaseRequest)
	if err != nil {
		return purchaserequest.PurchaseRequest{}, err
	}
	lineID, err := d.next(ctx, ids.PrefixPurchaseRequestLine)
	if err != nil {
		return purchaserequest.PurchaseRequest{}, err
	}
	li := items[i]
	limit := li.PlannedCost
	return purchaserequest.PurchaseRequest{
		ID: prID, PlanID: planID, Name: "PR for " + li.Name, OwnerID: owner, MoAApproverID: moa, WorkflowID: approvalID,
		ModificationWorkflowID: changeID, Status: purchaserequest.StatusCreated, Attributes: attributes,
		Vendor: seedVendor, Meta: d.meta(),
		Lines: []purchaserequest.Line{{ID: lineID, PurchaseRequestID: prID, Position: 1, LineItemID: li.ID,
			Currency: li.Currency, Amount: li.PlannedCost, DeliveryDate: date.New(2027, 4, 30), OverallLimit: &limit,
			Attributes: settings.AttributeValues{}, Meta: d.meta()}},
	}, nil
}

func (d *demo) prAttributes(r *resolver) settings.AttributeValues {
	attribute := func(label string) string {
		return r.id(d.settings.Attributes, "Shell attribute", attributeKey(settings.LevelPurchaseRequest, label))
	}
	option := func(label, opt string) string {
		return r.id(d.settings.Options, "option", optionKey(settings.LevelPurchaseRequest, label, opt))
	}
	return settings.AttributeValues{
		attribute("Material or service"): value([]string{option("Material or service", "Service")}),
		attribute("PR type"):             value([]string{option("PR type", "Standard PR/PO")}),
		attribute("Short text"):          value("Ireland agency fee, 2027 launch"),
		attribute("Vendor"):              value(seedVendor),
		attribute("Vendor code"):         value("V-100104"),
		attribute("Email group"):         value(option("Email group", "G_GB_RETAIL")),
	}
}
