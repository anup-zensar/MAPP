package web

import (
	"encoding/json"
	"strings"
	"testing"
)

// FuzzControlScan checks the scan against the decoder: every decoded control character is found, and
// anything found is a control character or an unpaired surrogate, which the decoder shows as U+FFFD.
func FuzzControlScan(f *testing.F) {
	for _, seed := range []string{
		`{"name":"Ideas"}`, `{"a":{"b":["x","y\u0000"]}}`, `{"name":"\ud800"}`, `{"name":"🚀"}`,
		`{"name":"\\ud800"}`, `["\udc00",1,null,true]`, `"\u007f"`, `{"a":[[{"b":"\t\n\r"}]]}`, `{"a":"\ud800A"}`,
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, body []byte) {
		_, found := controlField(body)
		var value any
		if !json.Valid(body) || json.Unmarshal(body, &value) != nil {
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
	})
}

func decodedStrings(value any) (control, replaced bool) {
	switch v := value.(type) {
	case string:
		return hasControl(v), strings.ContainsRune(v, '�')
	case []any:
		for _, item := range v {
			c, r := decodedStrings(item)
			control, replaced = control || c, replaced || r
		}
	case map[string]any:
		for _, item := range v {
			c, r := decodedStrings(item)
			control, replaced = control || c, replaced || r
		}
	}
	return control, replaced
}
