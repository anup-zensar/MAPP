package lineitem

import "context"

func (r JSONStore) CountOwnedBy(ctx context.Context, personID string) (int, error) {
	list, err := r.List(ctx)
	n := 0
	for _, li := range list {
		if li.OwnerID == personID {
			n++
		}
	}
	return n, err
}
