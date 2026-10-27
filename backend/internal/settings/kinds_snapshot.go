package settings

import (
	"context"
	"encoding/json"

	"mapp/internal/history"
)

// snapshot returns record as history keeps it: its id, name, parent and stored data.
func snapshot(record any, id, name, parentID string) (history.Snapshot, bool, error) {
	data, err := json.Marshal(record)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	return history.Snapshot{ID: id, Name: name, ParentID: parentID, Data: data}, true, nil
}

// putRecord returns the Put of a kind: it stores the record of a snapshot exactly as the snapshot holds it.
func putRecord[T any](put func(context.Context, T) error) func(context.Context, history.Snapshot) error {
	return func(ctx context.Context, s history.Snapshot) error {
		var record T
		if err := json.Unmarshal(s.Data, &record); err != nil {
			return err
		}
		return put(ctx, record)
	}
}

func (s *Service) modelSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	m, found, err := s.repo.GetModel(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	return snapshot(m, m.ID, m.Name, "")
}

func (s *Service) activityTypeSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	t, found, err := s.repo.GetActivityType(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	return snapshot(t, t.ID, t.Name, "")
}

func (s *Service) attributeSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	a, found, err := s.repo.GetAttribute(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	return snapshot(a, a.ID, a.Label, "")
}

func (s *Service) optionSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	o, found, err := s.repo.GetOption(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	return snapshot(o, o.ID, o.Label, o.AttributeID)
}

// The snapshot name of a currency is its code (data model §11).
func (s *Service) currencySnapshot(ctx context.Context, code string) (history.Snapshot, bool, error) {
	c, found, err := s.repo.GetCurrency(ctx, code)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	return snapshot(c, c.Code, c.Code, "")
}

func (s *Service) budgetSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	b, found, err := s.repo.GetFinanceBudget(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	return snapshot(b, b.ID, b.Name, "")
}

func (s *Service) itemSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	i, found, err := s.repo.GetFinanceBudgetItem(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	return snapshot(i, i.ID, i.Name, i.BudgetID)
}

// optionIDs lists the ids of the options of the attribute attributeID: deleting an attribute takes them
// with it.
func (s *Service) optionIDs(ctx context.Context, attributeID string) ([]string, error) {
	options, err := s.repo.OptionsOf(ctx, attributeID)
	if err != nil {
		return nil, err
	}
	list := make([]string, 0, len(options))
	for _, o := range options {
		list = append(list, o.ID)
	}
	return list, nil
}

// itemIDs lists the ids of the items of the finance budget budgetID.
func (s *Service) itemIDs(ctx context.Context, budgetID string) ([]string, error) {
	items, err := s.repo.ItemsOf(ctx, budgetID)
	if err != nil {
		return nil, err
	}
	list := make([]string, 0, len(items))
	for _, i := range items {
		list = append(list, i.ID)
	}
	return list, nil
}
