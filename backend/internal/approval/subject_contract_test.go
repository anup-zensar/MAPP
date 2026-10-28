package approval

import (
	"encoding/json"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi-validator/schema_validation"

	"mapp/api"
)

func TestSectionMatchesContract(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-023 a purchase request's section is an ApprovalSection of the contract", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		prs := f.usePurchases(t)
		f.putKindWorkflow(t, prTwoLevelFlow, KindPurchaseRequest, named(daniel), Level{ApproverType: SourceRecordField,
			RecordField: RecordFieldMoAApprover, Mode: ModeAny})
		f.addPurchaseRequest(t, prs, purchaseRequest{id: purchaseRequestA, owner: priya.ID, moa: priya.ID,
			workflow: prTwoLevelFlow, amount: "12000.00"})
		f.submitPR(t, KindPurchaseRequest)
		sec, err := f.decideSubject(t, daniel, ActionApproved, "Fine.")
		if err != nil {
			t.Fatalf("approve level 1: %v", err)
		}
		data, err := json.Marshal(sec)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		document, err := libopenapi.NewDocument(api.OpenAPI)
		if err != nil {
			t.Fatalf("read the contract: %v", err)
		}
		model, err := document.BuildV3Model()
		if err != nil {
			t.Fatalf("build the contract: %v", err)
		}
		schema := model.Model.Components.Schemas.GetOrZero("ApprovalSection").Schema()
		if valid, problems := schema_validation.NewSchemaValidator().ValidateSchemaBytesWithVersion(schema, data, 3.1); !valid {
			for _, p := range problems {
				t.Errorf("%s: %v", p.Message, p.SchemaValidationErrors)
			}
		}
	})
}
