package record_test

import (
	"testing"

	"mapp/internal/platform/record"
)

func TestCheckSingleLine_038_FR055_L11(t *testing.T) {
	t.Parallel()
	const refused = "Names can't contain tabs, line breaks or text-direction characters."
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"ordinary text", "Helix Ultra launch – Q3 (UK & Ireland)", ""},
		{"emoji", "Summer launch 🚀", ""},
		{"empty", "", ""},
		{"tab", "Helix\tUltra", refused},
		{"line feed", "Helix\nUltra", refused},
		{"carriage return", "Helix\rUltra", refused},
		{"left-to-right embedding U+202A", "Helix\u202aUltra", refused},
		{"right-to-left override U+202E", "Helix\u202eUltra", refused},
		{"left-to-right isolate U+2066", "Helix\u2066Ultra", refused},
		{"pop directional isolate U+2069", "Helix\u2069Ultra", refused},
		{"left-to-right mark U+200E", "Helix\u200eUltra", refused},
		{"right-to-left mark U+200F", "Helix\u200fUltra", refused},
		{"zero-width joiner in an emoji U+200D", "👩‍💻 team", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := record.CheckSingleLine(tt.in); got != tt.want {
				t.Errorf("CheckSingleLine(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
