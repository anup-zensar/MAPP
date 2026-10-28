package history

import "testing"

func TestWithNote_EndsTheDetailsOfEveryEntry_012FR023(t *testing.T) {
	t.Parallel()
	f := newBareFixture(t)
	f.register(t, f.standardKinds()...)
	f.addTree(t)
	tests := []struct {
		name, details, want string
		notes               []string
	}{
		{name: "no note", details: "with 3 line items", want: "with 3 line items"},
		{name: "a note after the details", details: "with 3 line items.", notes: []string{"Bulk delete of 2 records (BLK-1001)"},
			want: "with 3 line items. Bulk delete of 2 records (BLK-1001)"},
		{name: "a note alone", notes: []string{"Imported from 'plan.xlsx' (IMP-1001)"}, want: "Imported from 'plan.xlsx' (IMP-1001)"},
		{name: "the inner note wins", notes: []string{"Bulk delete of 2 records (BLK-1001)", "Undo of bulk delete (BLK-1001)"},
			want: "Undo of bulk delete (BLK-1001)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			for _, note := range tt.notes {
				ctx = WithNote(ctx, note)
			}
			got, err := f.service.Log(ctx, priya, LogInput{
				Kind: "grandchild", RecordID: "G-1", RecordName: "Grandchild one", Action: ActionChanged, Details: tt.details,
			})
			if err != nil {
				t.Fatalf("Log: %v", err)
			}
			if got.Details != tt.want {
				t.Errorf("details = %q, want %q", got.Details, tt.want)
			}
		})
	}
}
