package campaign

import "context"

func (r JSONStore) CountCampaignsOwnedBy(ctx context.Context, personID string) (int, error) {
	list, err := r.ListCampaigns(ctx)
	n := 0
	for _, c := range list {
		if c.OwnerID != nil && *c.OwnerID == personID {
			n++
		}
	}
	return n, err
}

func (r JSONStore) CountInitiativesOwnedBy(ctx context.Context, personID string) (int, error) {
	list, err := r.ListInitiatives(ctx)
	n := 0
	for _, i := range list {
		if i.OwnerID != nil && *i.OwnerID == personID {
			n++
		}
	}
	return n, err
}
