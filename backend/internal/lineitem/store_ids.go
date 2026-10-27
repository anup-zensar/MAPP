package lineitem

import "context"

func (s *PostgresStore) ListByIDs(ctx context.Context, ids []string) ([]LineItem, error) {
	return lineItemsFromRows(s.records().ByID(ctx, ids))
}
