package querycount_test

import (
	"testing"

	"mapp/internal/platform/querycount"
)

func TestWith_OneCountPerRequest_028_FR009(t *testing.T) {
	t.Parallel()
	request := querycount.With(t.Context())
	querycount.Add(request)
	inner := querycount.With(request)
	querycount.Add(inner)

	if got := querycount.Of(request); got != 2 {
		t.Errorf("Of(request) = %d after one statement outside and one inside a nested With, want 2", got)
	}
}

func TestAdd_WithoutCounterCountsNothing_028_FR009(t *testing.T) {
	t.Parallel()
	querycount.Add(t.Context())

	if got := querycount.Of(t.Context()); got != 0 {
		t.Errorf("Of() without a counter = %d, want 0", got)
	}
}
