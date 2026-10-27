package settings

func filter[T any](list []T, keep func(T) bool) []T {
	kept := []T{}
	for _, element := range list {
		if keep(element) {
			kept = append(kept, element)
		}
	}
	return kept
}
