package settings

import (
	"testing"

	"mapp/internal/platform/errs"
)

func TestAttributeLevels(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		level   string
		typ     string
		marking string
		want    *errs.Error
	}{
		{name: "032 FR-006 a purchase-request attribute may be needed to submit", level: "purchaseRequest", typ: "text",
			marking: "neededToSubmit"},
		{name: "032 FR-006 a PR-line attribute may be optional", level: "purchaseRequestLine", typ: "longText",
			marking: "optional"},
		{name: "032 FR-006 a purchase-request attribute is never needed to publish", level: "purchaseRequest", typ: "text",
			marking: "neededToPublish", want: problems("marking", "At purchase request level the marking is Optional or Needed to submit.")},
		{name: "032 FR-006 a PR-line attribute has no % split", level: "purchaseRequestLine", typ: "multipleChoiceSplit",
			marking: "optional", want: problems("type", splitLevelMessage)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newTestService(t)
			input := ShellAttributeInput{Label: "Vendor", Level: tt.level, Type: tt.typ, Marking: tt.marking}
			if tt.typ == string(TypeMultipleChoiceSplit) {
				input.Options = []NewOption{{Label: "One"}}
			}

			created, err := f.service.CreateAttribute(t.Context(), alex, input)

			if tt.want != nil {
				assertError(t, err, tt.want)
				return
			}
			if err != nil || created.Level != Level(tt.level) || created.Marking != Marking(tt.marking) {
				t.Errorf("CreateAttribute = %+v, %v", created, err)
			}
		})
	}
}

func TestRoleLocks(t *testing.T) {
	t.Parallel()
	t.Run("032 FR-006 an attribute with a role keeps level and type and cannot be deleted", func(t *testing.T) {
		t.Parallel()
		f := newTestService(t)
		tf := f.addAttributes(t, testAttribute{level: LevelPurchaseRequest, label: "Vendor", typ: TypeText, role: RoleVendor},
			testAttribute{level: LevelPurchaseRequest, label: "Remarks", typ: TypeText})
		vendor, remarks := tf.attribute(LevelPurchaseRequest, "Vendor"), tf.attribute(LevelPurchaseRequest, "Remarks")

		_, retyped := f.service.UpdateAttribute(t.Context(), alex, vendor.ID, ShellAttributeUpdate{Type: text("longText"), Version: version(1)})
		_, deleted := f.service.DeleteAttribute(t.Context(), alex, vendor.ID)
		_, plainDeleted := f.service.DeleteAttribute(t.Context(), alex, remarks.ID)

		assertError(t, retyped, problems("type", roleLock))
		assertError(t, deleted, errs.InUse(roleLock))
		if plainDeleted != nil {
			t.Errorf("an attribute without a role is deleted, got %v", plainDeleted)
		}
	})
}
