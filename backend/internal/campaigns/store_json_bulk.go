package campaigns

import "context"

func (r JSONStore) AddCampaigns(ctx context.Context, list []Campaign) error {
	for _, c := range list {
		if err := r.PutCampaign(ctx, c); err != nil {
			return err
		}
	}
	return nil
}

func (r JSONStore) AddInitiatives(ctx context.Context, list []Initiative) error {
	for _, i := range list {
		if err := r.PutInitiative(ctx, i); err != nil {
			return err
		}
	}
	return nil
}

func (r JSONStore) DeleteCampaigns(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := r.DeleteCampaign(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (r JSONStore) DeleteInitiatives(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := r.DeleteInitiative(ctx, id); err != nil {
			return err
		}
	}
	return nil
}
