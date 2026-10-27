package settings

import "context"

func (r JSONStore) AddOptions(ctx context.Context, options []Option) error {
	for _, o := range options {
		if err := r.PutOption(ctx, o); err != nil {
			return err
		}
	}
	return nil
}

func (r JSONStore) DeleteOptions(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := r.DeleteOption(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (r JSONStore) PutRates(ctx context.Context, rates []ExchangeRate) error {
	for _, rate := range rates {
		if err := r.PutRate(ctx, rate); err != nil {
			return err
		}
	}
	return nil
}
