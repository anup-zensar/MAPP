package record_test

import (
	"testing"

	"mapp/internal/platform/record"
)

const singleLineMessage = "Names can't contain tabs, line breaks or text-direction characters."

func TestCheckSingleLine_038_FR055_L11(t *testing.T) {
	t.Parallel()
	refused := map[string]string{
		"tab": "Helix\tlaunch", "line feed": "Helix\nlaunch", "carriage return": "Helix\rlaunch",
		"left-to-right embedding": "Helix‪launch", "right-to-left override": "Helix‮launch",
		"left-to-right isolate": "⁦Helix", "pop directional isolate": "Helix⁩",
		"left-to-right mark": "Helix‎", "right-to-left mark": "‏Helix",
		"line separator": "Helix launch", "paragraph separator": "Helix launch",
	}
	for name, text := range refused {
		if got := record.CheckSingleLine(text); got != singleLineMessage {
			t.Errorf("%s: CheckSingleLine(%q) = %q, want the single-line message", name, text, got)
		}
	}
	for _, text := range []string{"Helix launch", "Café & crème — 2026", "Rocket 🚀 launch", "مرحبا", "日本の計画", ""} {
		if got := record.CheckSingleLine(text); got != "" {
			t.Errorf("CheckSingleLine(%q) = %q, want no problem", text, got)
		}
	}
}
