package purchaserequest

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"mapp/internal/history"
)

var errConfigurationKept = errors.New("purchaserequest: the Purchase Request Configuration is never removed")

// Kinds returns the history kinds of purchase requests (research R18): the purchase request with its lines in
// the recycle bin, and the configuration, which is only ever logged.
func (s *Service) Kinds() []history.Kind {
	return []history.Kind{
		{Name: recordKind, Singular: recordNoun, Plural: pluralNoun, Snapshot: s.snapshot, Remove: s.repo.Delete,
			Put: s.putHeader, Describe: s.describe, AfterRestore: s.afterRestore},
		{Name: lineKind, Singular: "PR line", Plural: "PR lines", Parent: recordKind, Snapshot: s.lineSnapshot,
			ChildIDs: s.lineIDs, Remove: s.repo.DeleteLine, Put: s.putLine, PrepareRestore: s.prepareLineRestore},
		{Name: configurationKind, Singular: "Purchase request configuration", Plural: "Purchase request configurations",
			Snapshot: s.configurationSnapshot, Remove: func(context.Context, string) error { return errConfigurationKept },
			Put: s.putConfiguration},
	}
}

func (s *Service) snapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	pr, found, err := s.repo.Get(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, found, err
	}
	pr.Lines = nil
	data, err := encode(pr)
	return history.Snapshot{ID: pr.ID, Name: pr.Name, Data: data}, true, err
}

func (s *Service) putHeader(ctx context.Context, snap history.Snapshot) error {
	var pr PurchaseRequest
	if err := json.Unmarshal(snap.Data, &pr); err != nil {
		return err
	}
	pr.Lines = nil
	return s.repo.Put(ctx, pr)
}

func (s *Service) lineSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	l, found, err := s.repo.GetLine(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, found, err
	}
	data, err := encode(l)
	return history.Snapshot{ID: l.ID, Name: HXKey(l.LineItemID) + " · " + l.LineItemID, ParentID: l.PurchaseRequestID,
		Data: data}, true, err
}

func (s *Service) lineIDs(ctx context.Context, parentID string) ([]string, error) {
	pr, found, err := s.repo.Get(ctx, parentID)
	if err != nil || !found {
		return nil, err
	}
	ids := make([]string, 0, len(pr.Lines))
	for _, l := range pr.Lines {
		ids = append(ids, l.ID)
	}
	return ids, nil
}

func (s *Service) putLine(ctx context.Context, snap history.Snapshot) error {
	l, err := decodeLine(snap.Data)
	if err != nil {
		return err
	}
	return s.repo.PutLine(ctx, l)
}

func (s *Service) configurationSnapshot(ctx context.Context, _ string) (history.Snapshot, bool, error) {
	c, err := s.repo.Configuration(ctx)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	data, err := encode(c)
	return history.Snapshot{ID: configurationID, Name: settingsName, Data: data}, true, err
}

func (s *Service) putConfiguration(ctx context.Context, snap history.Snapshot) error {
	var c Configuration
	if err := json.Unmarshal(snap.Data, &c); err != nil {
		return err
	}
	return s.repo.PutConfiguration(ctx, c)
}

// describe gives the log the labels and texts of a purchase request's fields: people and workflows by name, Shell
// attributes as Settings describes them.
func (s *Service) describe(ctx context.Context, field string, value any) (string, string, error) {
	text, _ := value.(string)
	switch {
	case strings.HasPrefix(field, "attributes."):
		return s.deps.Attributes.Describe(ctx, field, value)
	case field == fieldOwner || field == fieldMoA:
		found, err := s.deps.People.People(ctx, []string{text})
		label := map[string]string{fieldOwner: "Owner", fieldMoA: labelMoA}[field]
		return label, found[text].Name, err
	case field == fieldWorkflow || field == fieldModificationWorkflow:
		w, _, err := s.deps.Workflows.Workflow(ctx, text)
		label := map[string]string{fieldWorkflow: "Approval workflow",
			fieldModificationWorkflow: "Approval workflow for modification"}[field]
		return label, w.Name, err
	}
	return "", "", nil
}

func decodeLine(data json.RawMessage) (Line, error) {
	var l Line
	err := json.Unmarshal(data, &l)
	return l, err
}

func encode(v any) (json.RawMessage, error) {
	return json.Marshal(v)
}
