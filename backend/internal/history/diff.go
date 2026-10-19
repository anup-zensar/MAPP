package history

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// notFilledIn is how the activity log shows an empty value (data model §5).
const notFilledIn = "Not filled in"

// bookkeeping lists the fields of record.Meta. They change with every save, so they are never listed as
// changed fields (data model "Bookkeeping").
var bookkeeping = []string{"version", "createdAt", "createdBy", "updatedAt", "updatedBy"}

// Diff compares two versions of a record and returns every top-level field that changed, sorted by field
// name (FR-022, research R8). Both versions are encoded as JSON objects and compared field by field by
// their compact JSON, so a nested object or list counts as one value. The bookkeeping fields of record.Meta
// are left out, and null, a missing field, "", [] and {} all count as the same empty value. Each change has
// its Field, Old and New; Log fills in the texts. The list is empty, never nil, when nothing changed.
func Diff(before, after any) ([]Change, error) {
	oldFields, err := topLevelFields(before)
	if err != nil {
		return nil, err
	}
	newFields, err := topLevelFields(after)
	if err != nil {
		return nil, err
	}

	// The names of the fields of either version.
	names := map[string]bool{}
	for name := range oldFields {
		names[name] = true
	}
	for name := range newFields {
		names[name] = true
	}

	changes := []Change{}
	for _, name := range slices.Sorted(maps.Keys(names)) {
		oldValue, newValue := oldFields[name], newFields[name]
		unchanged := bytes.Equal(oldValue, newValue) || (isEmpty(oldValue) && isEmpty(newValue))
		if unchanged || slices.Contains(bookkeeping, name) {
			continue
		}
		changes = append(changes, Change{Field: name, Old: orNull(oldValue), New: orNull(newValue)})
	}
	return changes, nil
}

// topLevelFields encodes record as a JSON object and returns its fields, each as compact JSON (that is how
// encoding/json writes them).
func topLevelFields(record any) (map[string]json.RawMessage, error) {
	encoded, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("history: diff: %w", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, fmt.Errorf("history: diff compares records, which are JSON objects: %w", err)
	}
	return fields, nil
}

// isEmpty reports whether a stored value counts as empty: missing, null, "", [] or {}.
func isEmpty(value json.RawMessage) bool {
	switch string(value) {
	case "", "null", `""`, "[]", "{}":
		return true
	}
	return false
}

// orNull returns value, or JSON null when the value is missing, so that a change always has both values.
func orNull(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage("null")
	}
	return value
}

// decode turns a stored JSON value into a Go value for Describe and the default texts: nil, bool,
// json.Number, string, []any or map[string]any. A missing value is nil.
func decode(value json.RawMessage) (any, error) {
	if len(value) == 0 {
		return nil, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.UseNumber() // numbers keep the digits they were written with: 7.50 stays 7.50
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, fmt.Errorf("history: read a changed value: %w", err)
	}
	return decoded, nil
}

// fieldLabel writes a JSON field name in words, for the activity log of a kind that gives no label of its
// own: "plannedCost" → "Planned cost" (data model §5). A run of capitals counts as one word: "imageURL" →
// "Image url".
func fieldLabel(field string) string {
	if field == "" {
		return ""
	}
	var words strings.Builder
	previous := ' '
	for _, r := range field {
		if unicode.IsUpper(r) && !unicode.IsUpper(previous) && words.Len() > 0 {
			words.WriteByte(' ') // a capital starts a new word
		}
		words.WriteRune(unicode.ToLower(r))
		previous = r
	}
	label := words.String()
	first, size := utf8.DecodeRuneInString(label)
	return string(unicode.ToUpper(first)) + label[size:]
}

// defaultText writes a stored value as people read it, for a kind that does not describe it (data model
// §5): text as it is, true and false as "Yes" and "No", numbers as written, lists joined with ", ", and
// "Not filled in" for every empty value. An object has no default wording, so it is shown as its JSON;
// kinds describe their objects, such as money, with Describe.
func defaultText(value any) string {
	switch v := value.(type) {
	case nil:
		return notFilledIn
	case bool:
		if v {
			return "Yes"
		}
		return "No"
	case json.Number:
		return v.String()
	case string:
		if v == "" {
			return notFilledIn
		}
		return v
	case []any:
		if len(v) == 0 {
			return notFilledIn
		}
		texts := make([]string, 0, len(v))
		for _, item := range v {
			texts = append(texts, defaultText(item))
		}
		return strings.Join(texts, ", ")
	default: // map[string]any, the only other type decode returns
		object, _ := json.Marshal(v) // cannot fail: it was decoded from JSON
		if string(object) == "{}" {
			return notFilledIn
		}
		return string(object)
	}
}
