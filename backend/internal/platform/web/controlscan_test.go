package web

import (
	"encoding/json"
	"strings"
	"testing"
)

// unicodeEscapes writes ~u as \u, so escaped surrogate pairs stay escapes in the source.
var unicodeEscapes = strings.NewReplacer("~u", `\u`)

// FuzzControlScan checks the scan against the decoder: every decoded control character is found, and
// anything found is a control character or an unpaired surrogate, which the decoder shows as U+FFFD.
func FuzzControlScan(f *testing.F) {
	for _, seed := range []string{
		`{"name":"Ideas"}`, `{"a":{"b":["x","y\u0000"]}}`, `{"name":"\ud800"}`, `{"name":"🚀"}`,
		`{"name":"\\ud800"}`, `["\udc00",1,null,true]`, `"\u007f"`, `{"a":[[{"b":"\t\n\r"}]]}`, `{"a":"\ud800A"}`,
		`{"name":"~udc00~ud800"}`, `{"name":"~ud83d~ude00"}`, `{"name":"~uDBFF~uDFFF"}`,
	} {
		f.Add([]byte(unicodeEscapes.Replace(seed)))
	}
	f.Fuzz(checkScanAgainstDecoder)
}

func checkScanAgainstDecoder(t *testing.T, body []byte) {
	_, found := controlField(body)
	var value any
	if json.Unmarshal(body, &value) != nil {
		return
	}
	control, replaced := decodedStrings(value)
	if control && !found {
		t.Errorf("%q: a decoded control character was not found", body)
	}
	duplicateKeysMayHideIt := strings.Contains(string(body), `\u`) || strings.ContainsRune(string(body), 0x7f)
	if found && !control && !replaced && !duplicateKeysMayHideIt {
		t.Errorf("%q: found a problem the decoded value does not show", body)
	}
}

func TestControlField_UnpairedSurrogates_038_FR055_L11(t *testing.T) {
	t.Parallel()
	tests := []struct {
		body, field string
		found       bool
	}{
		{`{"name":"~ud800"}`, "name", true},
		{`{"name":"~uDFFF"}`, "name", true},
		{`{"name":"~udc00~ud800"}`, "name", true},
		{`{"name":"~ud800~ud800~udc00"}`, "name", true},
		{`{"a":[{"name":"x~uD83D"}]}`, "a.0.name", true},
		{`{"name":"~ud83d~ude00"}`, "", false},
		{`{"name":"~uD83D~uDE00 😀"}`, "", false},
		{`{"name":"😀"}`, "", false},
	}
	for _, tt := range tests {
		body := unicodeEscapes.Replace(tt.body)
		field, found := controlField([]byte(body))
		if field != tt.field || found != tt.found {
			t.Errorf("%s: controlField = %q %v, want %q %v", body, field, found, tt.field, tt.found)
		}
	}
}

func decodedStrings(value any) (control, replaced bool) {
	var items []any
	switch v := value.(type) {
	case string:
		return hasControl(v), strings.ContainsRune(v, '�')
	case []any:
		items = v
	case map[string]any:
		for _, item := range v {
			items = append(items, item)
		}
	}
	for _, item := range items {
		c, r := decodedStrings(item)
		control, replaced = control || c, replaced || r
	}
	return control, replaced
}
