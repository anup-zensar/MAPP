package activities

import "context"

func (r JSONStore) AddMany(ctx context.Context, list []Activity) error {
	for _, a := range list {
		if err := r.Put(ctx, a); err != nil {
			return err
		}
	}
	return nil
}

func (r JSONStore) DeleteMany(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := r.Delete(ctx, id); err != nil {
			return err
		}
	}
	return nil
}
