package deck

import (
	"strings"
	"unicode"
)

const (
	ellipsis      = "…"
	ellipsisWidth = 1000
	defaultWidth  = 556
	firstMeasured = ' '
	lastMeasured  = '~'
)

// The widths of ' ' to '~' in thousandths of the font size, from the Helvetica metrics (Arial is metric-compatible).
var helvetica = [...]int{ //nolint:gochecknoglobals // font metrics: read only
	278, 278, 355, 556, 556, 889, 667, 191, 333, 333, 389, 584, 278, 333, 278, 278,
	556, 556, 556, 556, 556, 556, 556, 556, 556, 556, 278, 278, 584, 584, 584, 556,
	1015, 667, 667, 722, 722, 667, 611, 778, 722, 278, 500, 667, 556, 833, 722, 778,
	667, 778, 722, 667, 611, 722, 667, 944, 667, 667, 611, 278, 278, 278, 469, 556,
	333, 556, 556, 500, 556, 556, 278, 556, 556, 222, 222, 500, 222, 833, 556, 556,
	556, 556, 333, 500, 278, 556, 500, 722, 500, 500, 500, 334, 260, 334, 584,
}

var helveticaBold = [...]int{ //nolint:gochecknoglobals // font metrics: read only
	278, 333, 474, 556, 556, 889, 722, 238, 333, 333, 389, 584, 278, 333, 278, 278,
	556, 556, 556, 556, 556, 556, 556, 556, 556, 556, 333, 333, 584, 584, 584, 611,
	975, 722, 722, 722, 722, 667, 611, 778, 722, 278, 556, 722, 611, 833, 722, 778,
	667, 778, 722, 667, 611, 722, 667, 944, 667, 667, 611, 333, 278, 333, 584, 556,
	333, 556, 611, 556, 611, 556, 333, 611, 611, 278, 278, 556, 278, 889, 611, 611,
	611, 611, 389, 556, 333, 611, 556, 778, 556, 556, 500, 389, 280, 389, 584,
}

// windows1252 maps the characters of Windows-1252 between 0x80 and 0x9F; Latin-1 maps to itself above them.
var windows1252 = map[rune]byte{ //nolint:gochecknoglobals // a code page: read only
	'€': 0x80, '‚': 0x82, 'ƒ': 0x83, '„': 0x84, '…': 0x85, '†': 0x86, '‡': 0x87, 'ˆ': 0x88, '‰': 0x89,
	'Š': 0x8A, '‹': 0x8B, 'Œ': 0x8C, 'Ž': 0x8E, '‘': 0x91, '’': 0x92, '“': 0x93, '”': 0x94, '•': 0x95,
	'–': 0x96, '—': 0x97, '˜': 0x98, '™': 0x99, 'š': 0x9A, '›': 0x9B, 'œ': 0x9C, 'ž': 0x9E, 'Ÿ': 0x9F,
}

func runeWidth(r rune, bold bool) int {
	if r >= firstMeasured && r <= lastMeasured {
		if bold {
			return helveticaBold[r-firstMeasured]
		}
		return helvetica[r-firstMeasured]
	}
	switch r {
	case '…', '—', '‰':
		return ellipsisWidth
	case '•':
		return 350
	}
	return defaultWidth
}

// textWidth is the width of s in points at size, rounded up.
func textWidth(s string, size int, bold bool) int {
	units := 0
	for _, r := range s {
		units += runeWidth(r, bold)
	}
	return (units*size + 999) / 1000
}

func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
}

// fit returns s, shortened with "…" when it is wider than width points.
func fit(s string, width, size int, bold bool) string {
	cleaned := clean(s)
	if textWidth(cleaned, size, bold) <= width {
		return cleaned
	}
	runes := []rune(cleaned)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		candidate := strings.TrimRight(string(runes), " ") + ellipsis
		if textWidth(candidate, size, bold) <= width {
			return candidate
		}
	}
	return ""
}

// winANSI encodes s for the PDF's standard fonts; a character Windows-1252 does not have becomes "?".
func winANSI(s string) []byte {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		switch {
		case r < 0x80 || (r >= 0xA0 && r <= 0xFF):
			out = append(out, byte(r)) //nolint:gosec // G115: r is at most 0xFF here
		default:
			b, found := windows1252[r]
			if !found {
				b = '?'
			}
			out = append(out, b)
		}
	}
	return out
}
