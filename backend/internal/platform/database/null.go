package database

// NullIfEmpty stores a text the domain leaves "" as NULL, so that a foreign key on its column holds.
func NullIfEmpty(text string) *string {
	if text == "" {
		return nil
	}
	return &text
}

func EmptyIfNull(text *string) string {
	if text == nil {
		return ""
	}
	return *text
}
