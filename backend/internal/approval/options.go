package approval

import (
	"context"

	"mapp/internal/platform/actor"
)

// RuleOptions returns what a level's rule may name; every signed-in
// person may read them (D-296).
func (s *Service) RuleOptions(ctx context.Context, _ actor.Actor) (RuleOptions, error) {
	options := RuleOptions{ActivityTypes: []SettingRef{}, BusinessUnits: []OptionRef{}, Roles: []RoleRef{}}
	options.ObjectTypes, options.RecordFields = kindOptions()
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		types, err := s.choices.ActivityTypes(ctx)
		if err != nil {
			return err
		}
		for _, c := range types {
			options.ActivityTypes = append(options.ActivityTypes, SettingRef{ID: c.ID, Name: c.Label, Active: c.Active})
		}
		units, err := s.choices.BusinessUnits(ctx)
		if err != nil {
			return err
		}
		for _, c := range units {
			options.BusinessUnits = append(options.BusinessUnits, OptionRef{ID: c.ID, Label: c.Label, Active: c.Active})
		}
		roles, err := s.access.Roles(ctx)
		if err != nil {
			return err
		}
		for _, r := range roles {
			if r.Approves {
				options.Roles = append(options.Roles, r.RoleRef)
			}
		}
		return nil
	})
	if err != nil {
		return RuleOptions{}, err
	}
	return options, nil
}

// kindOptions are what a workflow can approve and the record fields a level may name, per object type.
func kindOptions() ([]KindOption, []RecordFieldOption) {
	kinds, fields := []KindOption{}, []RecordFieldOption{}
	for _, kind := range allKinds() {
		kinds = append(kinds, KindOption{ID: kind, Label: kind.Label()})
		for _, f := range kind.recordFields() {
			fields = append(fields, RecordFieldOption{ObjectType: kind, ID: f.ID, Label: f.Label})
		}
	}
	return kinds, fields
}
