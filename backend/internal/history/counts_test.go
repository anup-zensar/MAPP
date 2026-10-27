package history

import "testing"

// A copy's log entry says what came with the copied record in the words of a delete: kinds in the order
// they were registered, the singular for one record, kinds without records left out.
func TestCountsText_CopyDetails(t *testing.T) {
	t.Parallel()
	f := newFixture(t) // kinds parent, child and grandchild

	tests := []struct {
		counts map[string]int
		want   string
	}{
		{map[string]int{"grandchild": 3, "child": 2}, "with 2 children and 3 grandchildren"},
		{map[string]int{"child": 1}, "with 1 child"},
		{map[string]int{"child": 0, "unknown": 4}, ""},
		{nil, ""},
	}
	for _, tt := range tests {
		if got := f.service.CountsText(tt.counts); got != tt.want {
			t.Errorf("CountsText(%v) = %q, want %q", tt.counts, got, tt.want)
		}
	}
}
