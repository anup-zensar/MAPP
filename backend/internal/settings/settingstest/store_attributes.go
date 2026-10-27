package settingstest

import (
	"context"

	"mapp/internal/settings"
)

func (s Store) GetAttribute(_ context.Context, id string) (settings.Attribute, bool, error) {
	return s.attributes.Get(id)
}

func (s Store) ListAttributes(context.Context) ([]settings.Attribute, error) {
	return s.attributes.List()
}

func (s Store) PutAttribute(ctx context.Context, a settings.Attribute) error {
	a.Condition = orEmpty(a.Condition)
	return s.attributes.Put(ctx, a)
}

func (s Store) DeleteAttribute(ctx context.Context, id string) error {
	return s.attributes.Delete(ctx, id)
}

func (s Store) GetOption(_ context.Context, id string) (settings.Option, bool, error) {
	return s.options.Get(id)
}

func (s Store) ListOptions(context.Context) ([]settings.Option, error) {
	return s.options.List()
}

func (s Store) OptionsOf(_ context.Context, attributeID string) ([]settings.Option, error) {
	return s.options.Where(func(o settings.Option) bool { return o.AttributeID == attributeID })
}

func (s Store) PutOption(ctx context.Context, o settings.Option) error {
	o.ParentOptionIDs = orEmpty(o.ParentOptionIDs)
	return s.options.Put(ctx, o)
}

func (s Store) AddOptions(ctx context.Context, options []settings.Option) error {
	for _, o := range options {
		if err := s.PutOption(ctx, o); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) DeleteOptions(ctx context.Context, ids []string) error {
	return s.options.Delete(ctx, ids...)
}

func (s Store) DeleteOption(ctx context.Context, id string) error {
	return s.options.Delete(ctx, id)
}
