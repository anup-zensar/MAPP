package approval

import (
	"context"

	"mapp/internal/platform/actor"
)

// RuleOptions returns what a level's rule may name (017 FR-001, FR-005): every activity type and business
// unit, active or not, as Settings sorts them, and feature 014's roles that include approvals.decide, by
// name. Every signed-in person may read them (D-296), so a is not used.
func (s *Service) RuleOptions(ctx context.Context, a actor.Actor) (RuleOptions, error) {
	options := RuleOptions{ActivityTypes: []SettingRef{}, BusinessUnits: []OptionRef{}, Roles: []RoleRef{}}
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
