package approval

// where keeps the records keep accepts, in order; never nil.
func where[T any](records []T, keep func(T) bool) []T {
	kept := []T{}
	for _, r := range records {
		if keep(r) {
			kept = append(kept, r)
		}
	}
	return kept
}
