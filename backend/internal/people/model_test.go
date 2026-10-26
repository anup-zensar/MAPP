package people

import "testing"

// The sign-in page and the sidebar show a person's first name and initials, worked out from the full name
// (data model §1).
func TestPerson_FirstNameAndInitials_FR001(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		firstName string
		initials  string
	}{
		{"Priya Raman", "Priya", "PR"},
		{"Mary Ann Smith", "Mary", "MA"}, // the first letters of the first two words
		{"Cher", "Cher", "CH"},           // a one-word name gives its first two letters
		{"  mei   lin ", "mei", "ML"},    // extra spaces do not count; initials are upper case
		{"Émile Zola", "Émile", "ÉZ"},    // letters, not bytes
		{"X", "X", "X"},
	}
	for _, tt := range tests {
		p := Person{Name: tt.name}

		if got := p.FirstName(); got != tt.firstName {
			t.Errorf("FirstName() of %q = %q, want %q", tt.name, got, tt.firstName)
		}
		if got := p.Initials(); got != tt.initials {
			t.Errorf("Initials() of %q = %q, want %q", tt.name, got, tt.initials)
		}
	}
}
