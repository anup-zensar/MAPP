package financials

import (
	"context"
	"testing"
)

// The two history kinds are top level, named for people, read and put back their records, and describe a
// link change (FR-027, data model §7, D-254).
func TestKinds_TopLevelAndDescribed_FR027(t *testing.T) {
	f := newFixture(t)
	kinds := f.service.Kinds()
	if len(kinds) != 2 {
		t.Fatalf("%d kinds, want 2", len(kinds))
	}
	byName := map[string]int{}
	for i, k := range kinds {
		byName[k.Name] = i
		if k.Parent != "" || k.Snapshot == nil || k.Remove == nil || k.Put == nil {
			t.Errorf("kind %s: parent %q, hooks missing", k.Name, k.Parent)
		}
	}
	imports, transactions := kinds[byName["financialImport"]], kinds[byName["financialTransaction"]]
	if imports.Singular != "GSAP extract" || imports.Plural != "GSAP extracts" ||
		transactions.Singular != "GSAP transaction" || transactions.Plural != "GSAP transactions" {
		t.Errorf("words = %+v / %+v", imports, transactions)
	}

	f.write(t, func(ctx context.Context) error {
		s, found, err := transactions.Snapshot(ctx, "FT-1004")
		if err != nil || !found || s.Name != "PO 4500001004 / 10" {
			t.Errorf("snapshot = %+v, %v, %v", s, found, err)
		}
		if err := transactions.Remove(ctx, "FT-1004"); err != nil {
			return err
		}
		if err := transactions.Put(ctx, s); err != nil {
			return err
		}
		demo, found, err := imports.Snapshot(ctx, "FIM-1001")
		if err != nil || !found || demo.Name != "Demo data (simulated GSAP)" {
			t.Errorf("import snapshot = %+v, %v, %v", demo, found, err)
		}
		if _, found, _ := imports.Snapshot(ctx, "FIM-9999"); found {
			t.Errorf("FIM-9999 found")
		}
		return imports.Put(ctx, demo)
	})
	if got := f.stored(t, "FT-1004"); got.Version != 1 || got.NotLinkedReason == "" {
		t.Errorf("put back = %+v", got)
	}

	cases := []struct {
		value any
		text  string
	}{
		{"L-1007", "Station signage (L-1007)"},
		{nil, "Not linked"},
		{"L-9999", "L-9999"}, // no longer live
	}
	for _, c := range cases {
		label, text, err := transactions.Describe(t.Context(), "lineItemId", c.value)
		if err != nil || label != "Line item" || text != c.text {
			t.Errorf("Describe(%v) = %q, %q, %v; want %q", c.value, label, text, err, c.text)
		}
	}
	if label, text, _ := transactions.Describe(t.Context(), "amount", "1"); label != "" || text != "" {
		t.Errorf("other fields use the defaults: %q %q", label, text)
	}
}
