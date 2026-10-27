package app

import (
	"context"
	"fmt"
	"slices"

	"mapp/internal/allwork"
	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// Years returns the years that have plans (004) the person may see (014 FR-007).
func (s allWorkSource) Years(ctx context.Context) ([]int, error) {
	all, err := s.plansRepo.ListInScope(ctx, authz.ScopeOf(ctx))
	if err != nil {
		return nil, err
	}
	years := []int{}
	for _, p := range all {
		if !slices.Contains(years, p.Year) {
			years = append(years, p.Year)
		}
	}
	return years, nil
}

// Choices returns every team with the person's own marked (003), the options of the plan-level Business
// unit attribute and every activity type (002), and every person (003), active and inactive.
func (s allWorkSource) Choices(ctx context.Context, personID string) (allwork.Choices, error) {
	c := allwork.Choices{ActivityTypes: []allwork.Option{}, People: []allwork.Option{}}
	var err error
	if c.Teams, err = s.teamChoices(ctx, personID); err != nil {
		return allwork.Choices{}, err
	}
	if c.BusinessUnits, err = s.businessUnitChoices(ctx); err != nil {
		return allwork.Choices{}, err
	}
	types, err := s.settings.ActivityTypes(ctx)
	if err != nil {
		return allwork.Choices{}, err
	}
	for _, t := range types {
		c.ActivityTypes = append(c.ActivityTypes, allwork.Option{ID: t.ID, Label: t.Name, Inactive: !t.Active})
	}
	everyone, err := s.peopleRepo.List(ctx)
	if err != nil {
		return allwork.Choices{}, err
	}
	for _, p := range everyone {
		c.People = append(c.People, allwork.Option{ID: p.ID, Label: p.Name, Inactive: !p.Active})
	}
	c.SeesFinance = s.seesFinance(ctx, personID) // feature 015 (financials.go)
	if c.SeesFinance {
		if c.FinancePlans, err = s.financeAccess.VisiblePlanIDs(ctx, actor.Actor{ID: personID}); err != nil {
			return allwork.Choices{}, err
		}
	}
	return c, nil
}

func (s allWorkSource) teamChoices(ctx context.Context, personID string) ([]allwork.Option, error) {
	mine, err := s.people.TeamIDsOf(ctx, personID)
	if err != nil {
		return nil, err
	}
	teams, err := s.peopleRepo.ListTeams(ctx)
	if err != nil {
		return nil, err
	}
	list := []allwork.Option{}
	for _, t := range teams {
		list = append(list, allwork.Option{ID: t.ID, Label: t.Name, Inactive: !t.Active, Mine: slices.Contains(mine, t.ID)})
	}
	return list, nil
}

func (s allWorkSource) businessUnitChoices(ctx context.Context) ([]allwork.Option, error) {
	attributes, err := s.settings.Attributes(ctx, settings.LevelPlan)
	if err != nil {
		return nil, err
	}
	list := []allwork.Option{}
	for _, a := range attributes {
		if a.Role != settings.RoleBusinessUnit {
			continue
		}
		options, err := s.settings.Options(ctx, a.ID)
		if err != nil {
			return nil, err
		}
		for _, o := range options {
			list = append(list, allwork.Option{ID: o.ID, Label: o.Label, Inactive: !o.Active})
		}
	}
	return list, nil
}

// Attributes returns every Shell attribute of Settings (002), active and inactive, level by level top-down
// and in display order within a level.
func (s allWorkSource) Attributes(ctx context.Context) ([]allwork.Attribute, error) {
	list := []allwork.Attribute{}
	for _, level := range settings.Levels {
		attributes, err := s.settings.Attributes(ctx, level)
		if err != nil {
			return nil, err
		}
		for _, a := range attributes {
			attribute := allwork.Attribute{
				ID: a.ID, Label: a.Label, Level: allwork.Level(level), Number: a.Type == settings.TypeNumber, Active: a.Active,
				Type: string(a.Type), Condition: slices.Clone(a.Condition), Options: []allwork.Option{},
			}
			if a.IsChoice() { // 016: the options of the attribute filters and the Breakdown, in Settings' order
				options, err := s.settings.Options(ctx, a.ID)
				if err != nil {
					return nil, err
				}
				for _, o := range options {
					attribute.Options = append(attribute.Options, allwork.Option{ID: o.ID, Label: o.Label, Inactive: !o.Active})
				}
			}
			list = append(list, attribute)
		}
	}
	return list, nil
}

// RatesUsed states the current rate of every currency other than USD, with its date and source (016 FR-009):
// "1 USD = 0.79 GBP · rate of 19 Oct 2026 · GSAP import".
func (s allWorkSource) RatesUsed(ctx context.Context) ([]string, error) {
	infos, err := s.settings.CurrentRates(ctx)
	if err != nil {
		return nil, err
	}
	lines := []string{}
	for _, info := range infos {
		if info.Currency == money.BaseCurrency {
			continue
		}
		line := fmt.Sprintf("1 USD = %s %s", info.Rate.String(), info.Currency)
		if info.RateDate != nil {
			line += " · rate of " + date.Format(*info.RateDate)
		}
		if info.Source != nil {
			line += " · " + rateSourceWords()[*info.Source]
		}
		lines = append(lines, line)
	}
	return lines, nil
}

// rateSourceWords returns the sources of a rate as people read them.
func rateSourceWords() map[settings.RateSource]string {
	return map[settings.RateSource]string{settings.RateManual: "Entered by hand", settings.RateGSAPImport: "GSAP import"}
}

// Currency returns the person's display currency and its current rate (003's DisplayRateOf); an unusable
// rate shows US dollars.
func (s allWorkSource) Currency(ctx context.Context, personID string) (allwork.Currency, error) {
	d, err := s.people.DisplayRateOf(ctx, personID)
	if err != nil {
		return allwork.Currency{}, err
	}
	return displayCurrency(d), nil
}

func displayCurrency(d people.DisplayRate) allwork.Currency {
	rate, err := money.ParseRate(d.Rate)
	if err != nil || d.Code == "" {
		return allwork.Currency{Code: money.BaseCurrency, Rate: money.One}
	}
	return allwork.Currency{Code: d.Code, Rate: rate}
}
