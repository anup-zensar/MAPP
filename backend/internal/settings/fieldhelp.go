package settings

import (
	"context"
	"strings"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/record"
)

// Field help (016 FR-045 – FR-048, data-model §2.5, D-271): the catalogue of fieldhelp_catalogue.go gives
// every built-in field and calculated value its key, label and default text; only texts an administrator
// edited are stored. Reset to default deletes the stored text.

// FieldHelpEntry is one entry as the API answers it (contract FieldHelpEntry).
type FieldHelpEntry struct {
	Key         string       `json:"key"`
	Level       HelpLevel    `json:"level"`
	Label       string       `json:"label"`
	Text        string       `json:"text"` // the text shown; "" hides the ⓘ
	DefaultText string       `json:"defaultText"`
	IsDefault   bool         `json:"isDefault"`
	Version     int          `json:"version"`   // 0 while the default is used
	UpdatedAt   *time.Time   `json:"updatedAt"` // null while the default is used
	UpdatedBy   *actor.Actor `json:"updatedBy"`
}

// FieldHelpPage is every entry in catalogue order (contract FieldHelpPage).
type FieldHelpPage struct {
	Items                      []FieldHelpEntry `json:"items"`
	Total                      int              `json:"total"`
	ShellAttributesWithoutHelp int              `json:"shellAttributesWithoutHelp"` // active ones with empty help (FR-048)
}

// FieldHelpUpdate is the body of PATCH /api/v1/field-help/{helpKey} (contract FieldHelpUpdate): a new text, or
// null to reset to the default, from the version the person started at (0 for a default).
type FieldHelpUpdate struct {
	Text    *string `json:"text"`
	Version *int    `json:"version"`
}

// helpGoneMessage answers a key the catalogue does not have.
const helpGoneMessage = "This help entry doesn't exist."

// ListFieldHelp returns every catalogue entry with its text, edited or default, and how many active Shell
// attributes have no help (FR-045, FR-048). Every signed-in person reads it: the forms show the texts.
func (s *Service) ListFieldHelp(ctx context.Context, a actor.Actor) (FieldHelpPage, error) {
	page := FieldHelpPage{Items: make([]FieldHelpEntry, 0, len(helpCatalogue))}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		edited, err := s.repo.ListFieldHelp(ctx)
		if err != nil {
			return err
		}
		byKey := make(map[string]FieldHelpText, len(edited))
		for _, h := range edited {
			byKey[h.Key] = h
		}
		for _, d := range helpCatalogue {
			stored, found := byKey[d.Key]
			page.Items = append(page.Items, helpEntry(d, stored, found))
		}
		page.Total = len(page.Items)
		attributes, err := s.repo.ListAttributes(ctx)
		if err != nil {
			return err
		}
		for _, attr := range attributes {
			if attr.Active && strings.TrimSpace(attr.HelpText) == "" {
				page.ShellAttributesWithoutHelp++
			}
		}
		return nil
	})
	return page, err
}

// helpEntry returns the catalogue entry d with its stored text, when found.
func helpEntry(d helpDefinition, stored FieldHelpText, found bool) FieldHelpEntry {
	entry := FieldHelpEntry{Key: d.Key, Level: d.Level, Label: d.Label, Text: d.Default, DefaultText: d.Default, IsDefault: true}
	if found {
		updatedAt, updatedBy := stored.UpdatedAt, stored.UpdatedBy
		entry.Text, entry.IsDefault, entry.Version = stored.Text, false, stored.Version
		entry.UpdatedAt, entry.UpdatedBy = &updatedAt, &updatedBy
	}
	return entry
}

// UpdateFieldHelp edits the help text of key, or resets it to the default when the text is null (FR-046):
// 0–300 characters after trimming, line breaks made spaces. The change is logged with the old and new
// text; a save from an out-of-date version is refused with who changed it and when. A key outside the
// catalogue answers not found.
func (s *Service) UpdateFieldHelp(ctx context.Context, a actor.Actor, key string, update FieldHelpUpdate) (FieldHelpEntry, error) {
	var entry FieldHelpEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		d, known := helpDefinitionOf(key)
		if !known {
			return errs.NotFound(helpGoneMessage)
		}
		var problems errs.Problems
		if update.Version == nil || *update.Version < 0 {
			problems.Add("version", reloadMessage)
		}
		var text string
		if update.Text != nil {
			text = plainHelpText(*update.Text)
			checkLength(&problems, "text", text, maxHelpText)
		}
		if err := problems.Err(); err != nil {
			return err
		}
		stored, found, err := s.repo.GetFieldHelp(ctx, key)
		if err != nil {
			return err
		}
		if err := s.checkHelpVersion(ctx, a, key, stored, found, *update.Version); err != nil {
			return err
		}
		before := d.Default
		if found {
			before = stored.Text
		}
		after := d.Default
		switch {
		case update.Text == nil && found: // Reset to default
			if err := s.repo.DeleteFieldHelp(ctx, key); err != nil {
				return err
			}
			found = false
		case update.Text != nil && (found || text != d.Default):
			if found {
				stored.Text = text
				stored.Touch(a, s.now())
			} else {
				stored, found = FieldHelpText{Key: key, Text: text, Meta: record.NewMeta(a, s.now())}, true
			}
			if err := s.repo.PutFieldHelp(ctx, stored); err != nil {
				return err
			}
			after = text
		}
		if before != after {
			if _, err := s.history.Log(ctx, a, history.LogInput{
				Kind: fieldHelpKind, RecordID: key, RecordName: fieldHelpName(key), Action: history.ActionChanged,
				Changes: []history.Change{{Field: "text", Old: jsonOf(before), New: jsonOf(after)}},
			}); err != nil {
				return err
			}
		}
		entry = helpEntry(d, stored, found)
		return nil
	})
	return entry, err
}

// checkHelpVersion refuses a save of key from an out-of-date version (FR-046): the stored text's version, or
// 0 while the default is used. A save from a version above 0 of a text that was reset meanwhile names who
// reset it.
func (s *Service) checkHelpVersion(ctx context.Context, a actor.Actor, key string, stored FieldHelpText, found bool, version int) error {
	if found {
		if version == 0 {
			return errs.VersionConflict(fieldHelpNoun, stored.UpdatedBy.Name, stored.UpdatedAt)
		}
		return stored.CheckVersion(version, fieldHelpNoun)
	}
	if version == 0 {
		return nil
	}
	entries, _, err := s.history.ListLog(ctx, a, history.LogFilter{RecordType: fieldHelpKind, RecordID: key, Limit: 1})
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil // never edited: nothing newer to lose
	}
	return errs.VersionConflict(fieldHelpNoun, entries[0].Actor.Name, entries[0].At)
}

// plainHelpText returns a help text as stored: line breaks made spaces, trimmed.
func plainHelpText(text string) string {
	text = strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(text)
	return strings.TrimSpace(text)
}
