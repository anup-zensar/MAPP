package activities

import (
	"bytes"
	"encoding/json"

	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// Small helpers for Shell attribute values as activities stores them (JSON values keyed by attribute id).

// splitTotal adds up the shares of a split value; an unreadable value has no shares.
func splitTotal(raw json.RawMessage) money.Percent {
	var shares []settings.SplitShare
	if json.Unmarshal(raw, &shares) != nil {
		return money.Percent{}
	}
	var total money.Percent
	for _, s := range shares {
		total = money.SumPercent(total, s.Share)
	}
	return total
}

// isEmptyValue reports whether an attribute value is no value: missing, null, "" or [].
func isEmptyValue(raw json.RawMessage) bool {
	switch string(bytes.TrimSpace(raw)) {
	case "", "null", `""`, "[]":
		return true
	}
	return false
}

// sameJSON reports whether two values are the same JSON, ignoring spaces.
func sameJSON(a, b json.RawMessage) bool {
	var x, y bytes.Buffer
	if json.Compact(&x, a) != nil || json.Compact(&y, b) != nil {
		return bytes.Equal(a, b)
	}
	return bytes.Equal(x.Bytes(), y.Bytes())
}

// mustJSON encodes values that always encode: strings, numbers, bools and lists of them.
func mustJSON(v any) json.RawMessage {
	encoded, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return encoded
}
