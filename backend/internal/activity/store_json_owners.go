package activity

import "context"

func (r JSONStore) CountOwnedBy(ctx context.Context, personID string) (int, error) {
	list, err := r.List(ctx)
	n := 0
	for _, a := range list {
		if a.OwnerID == personID {
			n++
		}
	}
	return n, err
}
