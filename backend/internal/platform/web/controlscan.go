package web

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	controlMessage = "Remove the control characters from this text."
	maxScanDepth   = 10000
)

type scanFrame struct {
	object    bool
	expectKey bool
	key       string
	index     int
}

type controlScan struct {
	stack []scanFrame
}

// controlField returns the dotted path of the first string value holding a control character, an unpaired
// surrogate escape or invalid UTF-8. The decoder turns the last two into U+FFFD, so only the raw bytes show them.
func controlField(body []byte) (string, bool) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	var scan controlScan
	for {
		start := decoder.InputOffset()
		token, err := decoder.Token()
		if err != nil || len(scan.stack) > maxScanDepth {
			return "", false
		}
		if scan.refuses(token, body[start:decoder.InputOffset()]) {
			return scan.path(), true
		}
	}
}

func (s *controlScan) refuses(token json.Token, raw []byte) bool {
	switch token {
	case json.Delim('{'), json.Delim('['):
		s.stack = append(s.stack, scanFrame{object: token == json.Delim('{'), expectKey: true})
		return false
	case json.Delim('}'), json.Delim(']'):
		s.stack = s.stack[:len(s.stack)-1]
		s.advance()
		return false
	}
	text, isString := token.(string)
	if isString && s.atKey() {
		top := &s.stack[len(s.stack)-1]
		top.key, top.expectKey = text, false
		return false
	}
	if isString && (hasControl(text) || unreadableText(raw)) {
		return true
	}
	s.advance()
	return false
}

func (s *controlScan) atKey() bool {
	return len(s.stack) > 0 && s.stack[len(s.stack)-1].object && s.stack[len(s.stack)-1].expectKey
}

func (s *controlScan) advance() {
	if len(s.stack) == 0 {
		return
	}
	top := &s.stack[len(s.stack)-1]
	if top.object {
		top.expectKey = true
	} else {
		top.index++
	}
}

func (s *controlScan) path() string {
	parts := make([]string, 0, len(s.stack))
	for _, frame := range s.stack {
		if frame.object {
			parts = append(parts, frame.key)
		} else {
			parts = append(parts, strconv.Itoa(frame.index))
		}
	}
	return strings.Join(parts, ".")
}

// hasControl allows tab, line feed and carriage return: multi-line texts need them, names refuse them.
func hasControl(text string) bool {
	for _, r := range text {
		if (r < 0x20 && r != '\t' && r != '\n' && r != '\r') || r == 0x7f {
			return true
		}
	}
	return false
}

func unreadableText(raw []byte) bool {
	return !utf8.Valid(raw) || hasUnpairedSurrogate(raw)
}

func hasUnpairedSurrogate(raw []byte) bool {
	for i := 0; i < len(raw); i++ {
		if raw[i] != '\\' {
			continue
		}
		unit := unicodeEscape(raw, i)
		i++
		if unit < 0xd800 || unit > 0xdfff {
			continue
		}
		if low := unicodeEscape(raw, i+5); unit >= 0xdc00 || low < 0xdc00 || low > 0xdfff {
			return true
		}
		i += 10
	}
	return false
}

func unicodeEscape(raw []byte, at int) int {
	if at+6 > len(raw) || raw[at] != '\\' || raw[at+1] != 'u' {
		return -1
	}
	unit, err := strconv.ParseUint(string(raw[at+2:at+6]), 16, 16)
	if err != nil {
		return -1
	}
	return int(unit)
}
