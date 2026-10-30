package campaign

import (
	"bytes"
	"encoding/json"

	"mapp/internal/platform/date"
)

func (in *CampaignInput) UnmarshalJSON(data []byte) error {
	type plain CampaignInput
	return decodeWithTextDates(data, (*plain)(in), &in.StartDate, &in.EndDate)
}

func (in *CampaignUpdate) UnmarshalJSON(data []byte) error {
	type plain CampaignUpdate
	return decodeWithTextDates(data, (*plain)(in), &in.StartDate, &in.EndDate)
}

func (in *InitiativeInput) UnmarshalJSON(data []byte) error {
	type plain InitiativeInput
	return decodeWithTextDates(data, (*plain)(in), &in.StartDate, &in.EndDate)
}

func (in *InitiativeUpdate) UnmarshalJSON(data []byte) error {
	type plain InitiativeUpdate
	return decodeWithTextDates(data, (*plain)(in), &in.StartDate, &in.EndDate)
}

// decodeWithTextDates reads the dates as text, so an impossible date is a field problem rather than an
// unreadable request: it becomes the zero Date, which checkForm reports.
func decodeWithTextDates(data []byte, body any, start, end **date.Date) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	startText, endText := fields["startDate"], fields["endDate"]
	delete(fields, "startDate")
	delete(fields, "endDate")
	rest, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(rest))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(body); err != nil {
		return err
	}
	if *start, err = dateFromText(startText); err != nil {
		return err
	}
	*end, err = dateFromText(endText)
	return err
}

func dateFromText(raw json.RawMessage) (*date.Date, error) {
	var text *string
	if raw != nil {
		if err := json.Unmarshal(raw, &text); err != nil {
			return nil, err
		}
	}
	if text == nil || *text == "" {
		return nil, nil
	}
	return parsedOrZero(*text), nil
}

func parsedOrZero(text string) *date.Date {
	parsed, err := date.Parse(text)
	if err != nil {
		return &date.Date{}
	}
	return &parsed
}
