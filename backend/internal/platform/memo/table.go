package memo

import "context"

// Table is every row of a small table, such as a Settings list, as one transaction keeps it.
type Table[R any] struct {
	Rows  []R
	index map[string]int
}

func (t Table[R]) Find(id string) (R, bool) {
	i, found := t.index[id]
	if !found {
		var none R
		return none, false
	}
	return t.Rows[i], true
}

// GetTable reads the rows once per context of With, as Get does, and indexes them by idOf.
func GetTable[R any](ctx context.Context, name string, read func(ctx context.Context) ([]R, error),
	idOf func(R) string) (Table[R], error) {
	return Get(ctx, name, func(ctx context.Context) (Table[R], error) {
		rows, err := read(ctx)
		if err != nil {
			return Table[R]{}, err
		}
		index := make(map[string]int, len(rows))
		for i, row := range rows {
			index[idOf(row)] = i
		}
		return Table[R]{Rows: rows, index: index}, nil
	})
}
