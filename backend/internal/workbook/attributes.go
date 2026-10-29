package workbook

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"mapp/internal/platform/money"
)

// The Shell attribute types, as Settings stores them.
const (
	typeSingle   = "singleChoice"
	typeMultiple = "multipleChoice"
	typeSplit    = "multipleChoiceSplit"
	typeYesNo    = "yesNo"
	typeNumber   = "number"
	listSep      = "; "
)

var sharePattern = regexp.MustCompile(`^(.*?)\s+(\d+(?:\.\d+)?)\s*%?$`)

type splitShare struct {
	OptionID string        `json:"optionId"`
	Share    money.Percent `json:"share"`
}

func formatAttribute(a Attribute, raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	switch a.Type {
	case typeSingle:
		var id string
		_ = json.Unmarshal(raw, &id) // a stored value of another shape reads as empty
		return labelOf(a.Options, id)
	case typeMultiple:
		var ids []string
		_ = json.Unmarshal(raw, &ids)
		labels := make([]string, 0, len(ids))
		for _, id := range ids {
			labels = append(labels, labelOf(a.Options, id))
		}
		return strings.Join(labels, listSep)
	case typeSplit:
		return formatSplit(a, raw)
	case typeYesNo:
		var b bool
		_ = json.Unmarshal(raw, &b)
		return yesNo(b)
	}
	var plain any
	_ = json.Unmarshal(raw, &plain)
	if n, isNumber := plain.(float64); isNumber {
		return strconv.FormatFloat(n, 'f', -1, 64)
	}
	text, _ := plain.(string)
	return text
}

func formatSplit(a Attribute, raw json.RawMessage) string {
	var shares []splitShare
	_ = json.Unmarshal(raw, &shares)
	parts := make([]string, 0, len(shares))
	for _, s := range shares {
		share := strings.TrimSuffix(strings.TrimRight(s.Share.String(), "0"), ".")
		parts = append(parts, labelOf(a.Options, s.OptionID)+" "+share)
	}
	return strings.Join(parts, listSep)
}

// parseAttribute reads a cell into the attribute's stored value; an empty cell clears it (nil).
func parseAttribute(a Attribute, text string) (json.RawMessage, string) {
	if text == "" {
		return nil, ""
	}
	var value any
	problem := ""
	switch a.Type {
	case typeSingle:
		var id string
		problem = parseOption(text, a.Options, &id, choiceCheck(a))
		value = id
	case typeMultiple:
		value, problem = parseChoices(a, text)
	case typeSplit:
		value, problem = parseSplit(a, text)
	case typeYesNo:
		var b bool
		problem = parseYesNo(text, &b)
		value = b
	case typeNumber:
		n, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return nil, msgWholeNumber
		}
		value = n
	default:
		value = text
	}
	if problem != "" {
		return nil, problem
	}
	raw, _ := json.Marshal(value) // strings, numbers, booleans and lists of them always encode
	return raw, ""
}

func parseChoices(a Attribute, text string) ([]string, string) {
	ids := []string{}
	for _, part := range strings.Split(text, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var id string
		if problem := parseOption(part, a.Options, &id, choiceCheck(a)); problem != "" {
			return nil, problem
		}
		ids = append(ids, id)
	}
	return ids, ""
}

// parseSplit reads "China 60; Hong Kong 40", or "China; Hong Kong" split evenly (R10).
func parseSplit(a Attribute, text string) ([]splitShare, string) {
	shares := []splitShare{}
	withShares := 0
	for _, part := range strings.Split(text, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		share, problem := parseShare(a, part)
		if problem != "" {
			return nil, problem
		}
		if !share.Share.IsZero() {
			withShares++
		}
		shares = append(shares, share)
	}
	switch withShares {
	case 0:
		for i, p := range money.PercentSplitEvenly(len(shares)) {
			shares[i].Share = p
		}
	case len(shares):
	default:
		return nil, msgSharesAll
	}
	return shares, ""
}

func parseShare(a Attribute, part string) (splitShare, string) {
	var id string
	if parseOption(part, a.Options, &id, choiceCheck(a)) == "" {
		return splitShare{OptionID: id}, ""
	}
	m := sharePattern.FindStringSubmatch(part)
	if m == nil {
		return splitShare{}, parseOption(part, a.Options, &id, choiceCheck(a))
	}
	if problem := parseOption(strings.TrimSpace(m[1]), a.Options, &id, choiceCheck(a)); problem != "" {
		return splitShare{}, problem
	}
	share, err := money.ParsePercent(m[2])
	if err != nil || share.Cmp(money.PercentFromHundredths(10000)) > 0 || share.IsZero() {
		return splitShare{}, msgShares
	}
	return splitShare{OptionID: id, Share: share}, ""
}

func choiceCheck(a Attribute) optionCheck {
	return optionCheck{heading: a.Label, unknown: fmt.Sprintf(msgNotAnOption, a.Label)}
}
