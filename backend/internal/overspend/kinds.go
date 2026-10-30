package overspend

import (
	"context"
	"encoding/json"
	"errors"

	"mapp/internal/history"
)

const (
	labelDefaultMode  = "New budgets start with"
	labelApprovalMode = "Approval over planned cost"
	labelOverspend    = "Overspend"
	fieldOverspend    = "overspend"
)

var errConfigurationKept = errors.New("overspend: the Overspend control setting is never removed")

// Kinds returns the history kind of Settings › Overspend control, which is only ever logged.
func (s *Service) Kinds() []history.Kind {
	return []history.Kind{{Name: configurationKind, Singular: "overspend control setting", Plural: "overspend control settings",
		Snapshot: s.snapshot, Remove: func(context.Context, string) error { return errConfigurationKept },
		Put: s.put, Describe: describe}}
}

func (s *Service) snapshot(ctx context.Context, _ string) (history.Snapshot, bool, error) {
	c, err := s.repo.Configuration(ctx)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	data, err := json.Marshal(c)
	return history.Snapshot{ID: configurationID, Name: settingsName, Data: data}, true, err
}

func (s *Service) put(ctx context.Context, snap history.Snapshot) error {
	var c Configuration
	if err := json.Unmarshal(snap.Data, &c); err != nil {
		return err
	}
	return s.repo.PutConfiguration(ctx, c)
}

func describe(_ context.Context, field string, value any) (string, string, error) {
	text, _ := value.(string)
	switch field {
	case fieldDefaultMode:
		return labelDefaultMode, ModeWord(Mode(text)), nil
	case fieldApprovalMode:
		return labelApprovalMode, ModeWord(Mode(text)), nil
	}
	return "", "", nil
}

// ModeWord is a mode as people read it, in the activity log of every budget.
func ModeWord(m Mode) string {
	if m.Normalised() == Refuse {
		return "Refuse"
	}
	return "Warn"
}

// Described gives the kinds of budgets the words of their overspend setting in the activity log (037 FR-005).
func Described(kinds []history.Kind) []history.Kind {
	described := make([]history.Kind, 0, len(kinds))
	for _, k := range kinds {
		describeRest := k.Describe
		k.Describe = func(ctx context.Context, field string, value any) (string, string, error) {
			if field == fieldOverspend {
				text, _ := value.(string)
				return labelOverspend, ModeWord(Mode(text)), nil
			}
			if describeRest == nil {
				return "", "", nil
			}
			return describeRest(ctx, field, value)
		}
		described = append(described, k)
	}
	return described
}
