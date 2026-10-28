package purchaserequest

import (
	"context"
	"fmt"
	"slices"

	"mapp/internal/platform/errs"
	"mapp/internal/settings"
)

const (
	groupPurchaseRequest = "purchaseRequest"
	groupLine            = "line"
	groupActivity        = "activity"
	groupLineItem        = "lineItem"
	activityApproved     = "approved"
)

var swbsStatusWords = map[string]string{ //nolint:gochecknoglobals // a fixed lookup of the 030 status labels
	"notRequested": "Request SWBS code is off", "waitingForApproval": "Waiting for approval",
	"readyToSend": "Ready to send to GSAP", "received": "Received",
}

// submissionKind is the round a submission starts now: a change once the purchase request was approved (R5).
func submissionKind(pr PurchaseRequest) RoundKind {
	if pr.Approved != nil && (pr.Status == StatusApproved || pr.ChangePending) {
		return KindModification
	}
	return KindPurchaseRequest
}

// gates are FR-021: everything that stops a submission of kind, every problem at once.
func (s *Service) gates(ctx context.Context, pr PurchaseRequest, facts map[string]LineItemFacts, kind RoundKind) ([]Problem, error) {
	problems := []Problem{}
	if len(pr.Lines) == 0 {
		problems = append(problems, Problem{Field: fieldLines, Group: groupPurchaseRequest, RecordID: pr.ID, Message: msgAddLine})
	}
	problems = append(problems, activityGates(pr, facts)...)
	problems = append(problems, overspentActivityGates(pr, facts)...)
	problems = append(problems, companyCodeGates(pr, facts)...)
	workflow, err := s.workflowGate(ctx, pr, kind)
	if err != nil {
		return nil, err
	}
	problems = append(problems, workflow...)
	if pr.MoAApproverID == "" {
		problems = append(problems, Problem{Field: fieldMoA, Group: groupPurchaseRequest, RecordID: pr.ID, Message: msgChooseMoA})
	}
	missing, err := s.attributeGates(ctx, pr, facts)
	return append(problems, missing...), err
}

func activityGates(pr PurchaseRequest, facts map[string]LineItemFacts) []Problem {
	problems := []Problem{}
	var named []string
	for _, l := range pr.Lines {
		f := facts[l.LineItemID]
		if f.ActivityStatus != activityApproved && !slices.Contains(named, f.ActivityID) {
			named = append(named, f.ActivityID)
			problems = append(problems, Problem{Group: groupActivity, RecordID: f.ActivityID,
				Message: fmt.Sprintf(msgActivityNotApproved, f.ActivityName)})
		}
		if f.SwbsCode == "" {
			problems = append(problems, Problem{Group: groupLineItem, RecordID: l.LineItemID,
				Message: fmt.Sprintf(msgNoSwbs, l.LineItemID, f.Name, swbsStatusWords[f.SwbsStatus])})
		}
	}
	return problems
}

// overspentActivityGates stop a submission while an activity's line items cost more than its planned cost, whatever
// its budgets' setting (037 FR-019, D-349), naming each activity once.
func overspentActivityGates(pr PurchaseRequest, facts map[string]LineItemFacts) []Problem {
	problems := []Problem{}
	var named []string
	for _, l := range pr.Lines {
		f := facts[l.LineItemID]
		if f.ActivityOverBy != nil && !slices.Contains(named, f.ActivityID) {
			named = append(named, f.ActivityID)
			problems = append(problems, Problem{Group: groupActivity, RecordID: f.ActivityID,
				Message: fmt.Sprintf(msgLineItemsOverCost, f.ActivityName, f.ActivityOverBy.Format())})
		}
	}
	return problems
}

func companyCodeGates(pr PurchaseRequest, facts map[string]LineItemFacts) []Problem {
	problems := []Problem{}
	var codes, unnamed []string
	for _, l := range pr.Lines {
		f := facts[l.LineItemID]
		switch {
		case f.CompanyCode == "" && !slices.Contains(unnamed, f.ActivityID):
			unnamed = append(unnamed, f.ActivityID)
			problems = append(problems, Problem{Group: groupActivity, RecordID: f.ActivityID,
				Message: fmt.Sprintf(msgNoCompanyCode, f.ActivityName)})
		case f.CompanyCode != "" && !slices.Contains(codes, f.CompanyCode):
			codes = append(codes, f.CompanyCode)
		}
	}
	if len(codes) > 1 {
		problems = append(problems, Problem{Field: fieldLines, Group: groupPurchaseRequest, RecordID: pr.ID,
			Message: fmt.Sprintf(msgTwoCompanyCodes, codes[0], codes[1])})
	}
	return problems
}

func (s *Service) workflowGate(ctx context.Context, pr PurchaseRequest, kind RoundKind) ([]Problem, error) {
	field, id := fieldWorkflow, pr.WorkflowID
	if kind == KindModification {
		field, id = fieldModificationWorkflow, pr.ModificationWorkflowID
	}
	w, found, err := s.deps.Workflows.Workflow(ctx, id)
	if err != nil || (found && w.Active && w.ObjectType == string(kind)) {
		return nil, err
	}
	return []Problem{{Field: field, Group: groupPurchaseRequest, RecordID: pr.ID,
		Message: fmt.Sprintf(msgChooseWorkflow, kindWords(kind))}}, nil
}

// attributeGates are the empty needed-to-submit fields of the header and of each line.
func (s *Service) attributeGates(ctx context.Context, pr PurchaseRequest, facts map[string]LineItemFacts) ([]Problem, error) {
	header, err := s.deps.Attributes.Missing(ctx, settings.LevelPurchaseRequest, pr.Attributes, businessUnitOf(pr, facts))
	if err != nil {
		return nil, err
	}
	problems := make([]Problem, 0, len(header))
	for _, m := range header {
		problems = append(problems, Problem{Field: m.Field, Group: groupPurchaseRequest, RecordID: pr.ID,
			Message: fmt.Sprintf(msgNeededToSubmit, m.Label)})
	}
	for i, l := range pr.Lines {
		missing, err := s.lineMissing(ctx, l, facts[l.LineItemID].BusinessUnit)
		if err != nil {
			return nil, err
		}
		for _, m := range missing {
			problems = append(problems, Problem{Field: fmt.Sprintf("lines[%d].%s", i, m.Field), Group: groupLine,
				RecordID: l.ID, Message: fmt.Sprintf(msgNeededToSubmit, m.Label)})
		}
	}
	return problems, nil
}

// headerMissing is the completeness chip's list for the header: attributes and the MoA approver.
func (s *Service) headerMissing(ctx context.Context, pr PurchaseRequest, facts map[string]LineItemFacts) ([]settings.MissingField, error) {
	missing, err := s.deps.Attributes.Missing(ctx, settings.LevelPurchaseRequest, pr.Attributes, businessUnitOf(pr, facts))
	if err != nil {
		return nil, err
	}
	if pr.MoAApproverID == "" {
		missing = append([]settings.MissingField{{Field: fieldMoA, Label: labelMoA}}, missing...)
	}
	return nonNilMissing(missing), nil
}

func (s *Service) lineMissing(ctx context.Context, l Line, businessUnit []string) ([]settings.MissingField, error) {
	missing, err := s.deps.Attributes.Missing(ctx, settings.LevelPurchaseRequestLine, l.Attributes, businessUnit)
	if err != nil {
		return nil, err
	}
	if l.OverallLimit == nil {
		missing = append([]settings.MissingField{{Field: "overallLimit", Label: "Overall limit"}}, missing...)
	}
	return nonNilMissing(missing), nil
}

func businessUnitOf(pr PurchaseRequest, facts map[string]LineItemFacts) []string {
	for _, l := range pr.Lines {
		if f, found := facts[l.LineItemID]; found {
			return f.BusinessUnit
		}
	}
	return nil
}

func nonNilMissing(list []settings.MissingField) []settings.MissingField {
	if list == nil {
		return []settings.MissingField{}
	}
	return list
}

// notReady is the 409 answer of a refused submission: the problems grouped by record (data-model §3.4).
func notReady(pr PurchaseRequest, problems []Problem, facts map[string]LineItemFacts) error {
	var groups []errs.ReadinessGroup
	for _, p := range problems {
		ref := recordOf(pr, p, facts)
		i := slices.IndexFunc(groups, func(g errs.ReadinessGroup) bool { return g.Record == ref })
		if i < 0 {
			groups = append(groups, errs.ReadinessGroup{Record: ref, Fields: []errs.MissingField{}})
			i = len(groups) - 1
		}
		groups[i].Fields = append(groups[i].Fields, errs.MissingField{Field: p.Field, Label: p.Message, Reason: p.Message})
	}
	return errs.NotReady(msgNotReady, groups)
}

func recordOf(pr PurchaseRequest, p Problem, facts map[string]LineItemFacts) errs.RecordRef {
	switch p.Group {
	case groupActivity:
		for _, f := range facts {
			if f.ActivityID == p.RecordID {
				return errs.RecordRef{Type: groupActivity, ID: p.RecordID, Name: f.ActivityName}
			}
		}
	case groupLineItem:
		return errs.RecordRef{Type: groupLineItem, ID: p.RecordID, Name: facts[p.RecordID].Name}
	case groupLine:
		return errs.RecordRef{Type: lineKind, ID: p.RecordID, Name: p.RecordID}
	}
	return errs.RecordRef{Type: recordKind, ID: pr.ID, Name: pr.Name}
}
