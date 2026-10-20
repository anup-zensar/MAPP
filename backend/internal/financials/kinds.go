package financials

import (
	"context"
	"encoding/json"

	"northstar/internal/history"
)

// The history kinds of feature 015 (data model §7, D-254). Both are top level, so no recycle-bin sweep ever
// reaches a transaction or an import: nothing moves them to the bin, but history requires every kind to
// read, remove and put back its records.
const (
	importKind      = "financialImport"
	transactionKind = "financialTransaction"
)

// Kinds returns the history kinds of GSAP imports and transactions, for internal/app to register. Their
// order is free: neither has a parent.
func (s *Service) Kinds() []history.Kind {
	return []history.Kind{
		{
			Name: importKind, Singular: importKindSingular, Plural: importKindPlural,
			Snapshot: s.importSnapshot, Remove: s.repo.DeleteImport, Put: s.putImport,
		},
		{
			Name: transactionKind, Singular: transactionKindSingular, Plural: transactionKindPlural,
			Snapshot: s.transactionSnapshot, Remove: s.repo.DeleteTransaction, Put: s.putTransaction,
			Describe: s.describeTransaction,
		},
	}
}

// importSnapshot reads the import id as history keeps it: its file name (the feed's name for the demo
// data) as its name.
func (s *Service) importSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	i, found, err := s.repo.GetImport(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	data, err := json.Marshal(i)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	name := i.FileName
	if name == "" {
		name = i.FeedName
	}
	return history.Snapshot{ID: i.ID, Name: name, Data: data}, true, nil
}

// transactionSnapshot reads the transaction id as history keeps it: its readable name ("PO 4500001004 /
// 10") as its name.
func (s *Service) transactionSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	t, found, err := s.repo.GetTransaction(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	data, err := json.Marshal(t)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	return history.Snapshot{ID: t.ID, Name: t.Name(), Data: data}, true, nil
}

// putImport stores the import of a snapshot again, exactly as the snapshot holds it.
func (s *Service) putImport(ctx context.Context, snapshot history.Snapshot) error {
	var i Import
	if err := json.Unmarshal(snapshot.Data, &i); err != nil {
		return err
	}
	return s.repo.PutImport(ctx, i)
}

// putTransaction stores the transaction of a snapshot again, exactly as the snapshot holds it.
func (s *Service) putTransaction(ctx context.Context, snapshot history.Snapshot) error {
	var t Transaction
	if err := json.Unmarshal(snapshot.Data, &t); err != nil {
		return err
	}
	return s.repo.PutTransaction(ctx, t)
}

// describeTransaction gives the log's words for a link change (data model §7): the field lineItemId is
// "Line item", a line item reads "<name> (L-1007)" — its ID alone when it is no longer live — and none
// reads "Not linked".
func (s *Service) describeTransaction(ctx context.Context, field string, value any) (string, string, error) {
	if field != "lineItemId" {
		return "", "", nil
	}
	id, _ := value.(string)
	if id == "" {
		return lineItemLabel, notLinkedText, nil
	}
	nodes, err := s.planning.Nodes(ctx)
	if err != nil {
		return "", "", err
	}
	for _, n := range nodes {
		if n.ID == id && n.Level == LevelLineItem {
			return lineItemLabel, n.Name + " (" + id + ")", nil
		}
	}
	return lineItemLabel, id, nil
}
