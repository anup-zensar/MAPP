package approval

// SubjectKind is the type of record an approval round or workflow is about.
type SubjectKind string

const (
	KindActivity                SubjectKind = "activity"
	KindPurchaseRequest         SubjectKind = "purchaseRequest"
	KindModifiedPurchaseRequest SubjectKind = "modifiedPurchaseRequest"
)

// kindTexts: record is the history kind and reference type; both purchase-request kinds are about the PR.
type kindTexts struct {
	label, word, plural, records, record string
	fields                               []RecordFieldRef
}

func allKinds() []SubjectKind {
	return []SubjectKind{KindActivity, KindPurchaseRequest, KindModifiedPurchaseRequest}
}

func (k SubjectKind) texts() kindTexts {
	const purchaseRequest, purchaseRequests, activities = "purchase request", "purchase requests", "activities"
	moa := []RecordFieldRef{{ID: RecordFieldMoAApprover, Label: "MoA approver"}}
	switch k {
	case KindPurchaseRequest:
		return kindTexts{label: "Purchase request", word: purchaseRequest, plural: purchaseRequests,
			records: purchaseRequests, record: string(KindPurchaseRequest), fields: moa}
	case KindModifiedPurchaseRequest:
		return kindTexts{label: "Modified purchase request", word: purchaseRequest,
			plural: "modified purchase requests", records: purchaseRequests, record: string(KindPurchaseRequest), fields: moa}
	case KindActivity:
	}
	return kindTexts{label: "Activity", word: "activity", plural: activities, records: activities, record: activityKind}
}

func (k SubjectKind) IsValid() bool {
	return k == KindActivity || k == KindPurchaseRequest || k == KindModifiedPurchaseRequest
}

func (k SubjectKind) Label() string { return k.texts().label }

func (k SubjectKind) word() string { return k.texts().word }

func (k SubjectKind) plural() string { return k.texts().plural }

func (k SubjectKind) records() string { return k.texts().records }

func (k SubjectKind) recordType() string { return k.texts().record }

// barsSelfApproval: 014's rule "Nobody approves an activity they own or submitted" is for activities (D-324).
func (k SubjectKind) barsSelfApproval() bool { return k == KindActivity }

func (k SubjectKind) isPurchaseRequest() bool {
	return k == KindPurchaseRequest || k == KindModifiedPurchaseRequest
}

func (k SubjectKind) recordField(id string) (RecordFieldRef, bool) {
	for _, f := range k.texts().fields {
		if f.ID == id {
			return f, true
		}
	}
	return RecordFieldRef{}, false
}

func (k SubjectKind) recordFields() []RecordFieldRef { return k.texts().fields }
