package performance

import "mapp/internal/platform/record"

// Alias is a superseded external name: the current one stays feature 006's to write (046-C14).
type Alias struct {
	ID         string `json:"id"`
	ActivityID string `json:"activityId"`
	Name       string `json:"name"`
	record.Meta
}

type IntegrationID struct {
	ID         string `json:"id"`
	ActivityID string `json:"activityId"`
	SourceID   string `json:"sourceId"`
	Value      string `json:"value"`
	record.Meta
}

type MediaCode struct {
	ID          string            `json:"id"`
	Code        int64             `json:"code"`
	ActivityID  string            `json:"activityId"`
	Attributes  map[string]string `json:"attributes"`
	LandingPage string            `json:"landingPage"`
	Status      Status            `json:"status"`
	record.Meta
}

// firstMediaCode starts above Hive9's numbers so a new code cannot collide (FR-042).
const firstMediaCode = 9000001

type CodeAttribute struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
}

// Configuration lives here rather than on the activity type, which feature 002 owns (046-C12).
type Configuration struct {
	ActivityTypeIDs []string        `json:"activityTypeIds"`
	CodeAttributes  []CodeAttribute `json:"codeAttributes"`
	MediaCodeNext   int64           `json:"mediaCodeNext"`
	record.Meta
}

func (c Configuration) RequiresCodes(activityTypeID string) bool {
	for _, id := range c.ActivityTypeIDs {
		if id == activityTypeID {
			return true
		}
	}
	return false
}

func (c Configuration) MissingAttributes(given map[string]string) []string {
	var missing []string
	for _, a := range c.CodeAttributes {
		if !a.Required {
			continue
		}
		if value, ok := given[a.Name]; !ok || value == "" {
			missing = append(missing, a.Name)
		}
	}
	return missing
}
