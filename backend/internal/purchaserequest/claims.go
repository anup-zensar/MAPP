package purchaserequest

import (
	"slices"

	"mapp/internal/platform/money"
)

// ClaimsOf is what the purchase requests prs claim on the line items lineItemIDs, a change in review included.
func ClaimsOf(prs []PurchaseRequest, lineItemIDs []string) []Claim {
	claims := []Claim{}
	for _, pr := range prs {
		for _, l := range pr.Lines {
			if slices.Contains(lineItemIDs, l.LineItemID) {
				claims = append(claims, Claim{LineID: l.ID, PurchaseRequestID: pr.ID, PurchaseRequest: pr.Name,
					LineItemID: l.LineItemID, Status: pr.Status, Amount: l.money()})
			}
		}
	}
	return WithChangesInReview(claims, prs, lineItemIDs)
}

// WithChangesInReview adds to claims what the approved versions of the purchase requests whose change is in
// review still reserve on lineItemIDs: the approved amount of each line, kept until the change is approved.
func WithChangesInReview(claims []Claim, prs []PurchaseRequest, lineItemIDs []string) []Claim {
	for _, pr := range prs {
		if !pr.ChangePending || pr.Approved == nil {
			continue
		}
		for _, v := range pr.Approved.Lines {
			if slices.Contains(lineItemIDs, v.LineItemID) {
				claims = withApprovedLine(claims, pr, v)
			}
		}
	}
	return claims
}

func withApprovedLine(claims []Claim, pr PurchaseRequest, v VersionLine) []Claim {
	approved := money.Money{Amount: v.Amount, Currency: v.Currency}
	i := slices.IndexFunc(claims, func(c Claim) bool { return c.LineID == v.ID && c.LineItemID == v.LineItemID })
	if i >= 0 {
		claims[i].Approved = &approved
		return claims
	}
	return append(claims, Claim{LineID: v.ID, PurchaseRequestID: pr.ID, PurchaseRequest: pr.Name, LineItemID: v.LineItemID,
		Status: pr.Status, Amount: approved, Approved: &approved, RemovedByChange: true})
}
