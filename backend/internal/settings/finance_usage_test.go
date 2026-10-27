package settings

import (
	"reflect"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
)

// An item counts the line items linked to it, those in the recycle bin too; its budget shows the plans
// using it as their default finance budget (US5-4, FR-003).
func TestItemUsage_LineItemsAndPlans_FR003(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	item := f.createItem(t, itemInput(2026, "2026 Lubricants Budget", "Europe – Digital", ""))
	plan := planRef("P-1001", "Lubricants FY2026", "MOD-1001", false, nil)
	plan.FinanceBudgetID = item.BudgetID
	f.refs = []Reference{plan}
	for i, id := range []string{"L-1001", "L-1002", "L-1003", "L-1004", "L-1005", "L-1006"} {
		ref := lineItemRef(id, "P-1001", "A-1001", "USD", "100.00", i == 5)
		ref.FinanceBudgetItemID = item.ID
		f.refs = append(f.refs, ref)
	}

	view, err := f.service.GetFinanceBudgetItem(t.Context(), alex, item.ID)
	page, _ := f.service.ListFinanceBudgets(t.Context(), alex, nil)

	wantItem := Usage{InUse: true, Summary: "Linked to 6 line items (1 in the recycle bin)",
		Parts:   []UsagePart{{Kind: "lineItem", Text: "Linked to 6 line items (1 in the recycle bin)", Count: 6, InRecycleBin: 1}},
		Records: []history.RecordRef{}}
	if err != nil || !reflect.DeepEqual(view.Usage, wantItem) {
		t.Errorf("item usage = %+v, %v\nwant %+v", view.Usage, err, wantItem)
	}
	wantBudget := Usage{InUse: true, Summary: "Default finance budget of 1 plan",
		Parts:   []UsagePart{{Kind: "plan", Text: "Default finance budget of 1 plan", Count: 1}},
		Records: []history.RecordRef{{Type: "plan", ID: "P-1001", Name: "Lubricants FY2026"}}}
	if !reflect.DeepEqual(page.Items[0].Usage, wantBudget) {
		t.Errorf("budget usage = %+v\nwant %+v", page.Items[0].Usage, wantBudget)
	}
}

// An item in use — linked to line items, also in the recycle bin — cannot be deleted: the answer says
// where it is used and to set it inactive instead (FR-004, SC-004).
func TestDeleteItem_InUseRefused_FR004(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	item := f.createItem(t, itemInput(2026, "2026 Lubricants Budget", "Europe – Digital", ""))
	ref := lineItemRef("L-1001", "P-1001", "A-1001", "USD", "100.00", true)
	ref.FinanceBudgetItemID = item.ID
	f.refs = []Reference{ref}

	_, used := f.service.DeleteFinanceBudgetItem(t.Context(), alex, item.ID)
	_, gone := f.service.DeleteFinanceBudgetItem(t.Context(), alex, "FBI-9999")

	assertError(t, used, errs.InUse("'Europe – Digital' is in use (linked to 1 line item (1 in the recycle bin)) — set it inactive instead."))
	assertError(t, gone, errs.NotFound("This finance budget item no longer exists — someone may have deleted it."))
}
