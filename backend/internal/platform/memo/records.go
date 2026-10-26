package memo

import (
	"cmp"
	"context"
	"slices"
	"strings"
)

// Records keeps the rows of one planning table by id and by parent for the rest of a transaction. Read reads
// the rows with the ids, or those of the parents: one of the two lists is given, never empty.
type Records[R any] struct {
	Area   string
	ID     func(R) string
	Parent func(R) string
	Read   func(ctx context.Context, ids, parentIDs []string) ([]R, error)
}

// ByID returns the rows with ids that exist, in table order: ids by length, then by text.
func (r Records[R]) ByID(ctx context.Context, ids []string) ([]R, error) {
	kept, err := Many(ctx, r.Area+":id:", ids, func(ctx context.Context, missing []string) (map[string]*R, error) {
		rows, err := r.read(ctx, missing, nil)
		found := make(map[string]*R, len(rows))
		for i := range rows {
			found[r.ID(rows[i])] = &rows[i]
		}
		return found, err
	})
	if err != nil {
		return nil, err
	}
	rows := make([]R, 0, len(kept))
	for _, row := range kept {
		if row != nil {
			rows = append(rows, *row)
		}
	}
	return r.inOrder(rows), nil
}

func (r Records[R]) ByParent(ctx context.Context, parentIDs []string) ([]R, error) {
	kept, err := Many(ctx, r.Area+":of:", parentIDs, func(ctx context.Context, missing []string) (map[string][]R, error) {
		rows, err := r.read(ctx, nil, missing)
		r.Keep(ctx, rows)
		found := make(map[string][]R, len(missing))
		for _, row := range rows {
			found[r.Parent(row)] = append(found[r.Parent(row)], row)
		}
		return found, err
	})
	if err != nil {
		return nil, err
	}
	var rows []R
	for _, children := range kept {
		rows = append(rows, children...)
	}
	return r.inOrder(rows), nil
}

// Keep keeps rows read by another query, such as a list in a person's scope, by id.
func (r Records[R]) Keep(ctx context.Context, rows []R) {
	for i := range rows {
		Keep(ctx, r.Area+":id:"+r.ID(rows[i]), &rows[i])
	}
}

func (r Records[R]) read(ctx context.Context, ids, parentIDs []string) ([]R, error) {
	if len(ids) == 0 && len(parentIDs) == 0 {
		return []R{}, nil
	}
	return r.Read(ctx, ids, parentIDs)
}

func (r Records[R]) inOrder(rows []R) []R {
	slices.SortFunc(rows, func(x, y R) int {
		a, b := r.ID(x), r.ID(y)
		return cmp.Or(cmp.Compare(len(a), len(b)), strings.Compare(a, b))
	})
	if rows == nil {
		return []R{}
	}
	return rows
}
