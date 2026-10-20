package plans

import (
	"context"

	"northstar/internal/people"
	"northstar/internal/platform/actor"
)

// Overview is a plan as feature 009's Home reads it (Go only): the stored plan with what the Plans list and
// the plan page show of it, every amount in US dollars, so that Home can add plans up before it converts
// the totals once (D-010).
type Overview struct {
	Plan         Plan             // as stored
	Teams        []people.TeamRef // by name
	Figures      Figures          // in USD
	Completeness Completeness
	Contents     Contents
}

// Overviews returns every plan with its figures in US dollars, its completeness and its contents, worked
// out exactly as the Plans list and the plan page work them out (FR-012, FR-017): for feature 009's Home,
// through an adapter in internal/app (D-196). Each level below is asked once for all the plans (SC-009).
// It joins the caller's transaction.
func (s *Service) Overviews(ctx context.Context, a actor.Actor) ([]Overview, error) {
	var list []Overview
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		all, err := s.repo.List(ctx)
		if err != nil {
			return err
		}
		planIDs := make([]string, 0, len(all))
		for _, p := range all {
			planIDs = append(planIDs, p.ID)
		}
		facts, err := s.facts(ctx, planIDs)
		if err != nil {
			return err
		}
		l, err := s.lookupsFor(ctx, all)
		if err != nil {
			return err
		}
		rates, err := s.settings.Rates(ctx)
		if err != nil {
			return err
		}
		list = make([]Overview, 0, len(all))
		for _, p := range all {
			f := *facts[p.ID]
			sh, err := s.shownOf(ctx, p, f, l, usdConverter.reading(p.Currency, rates)) // 016 FR-014: each plan in USD
			if err != nil {
				return err
			}
			list = append(list, Overview{
				Plan: p, Teams: sh.teams, Figures: sh.figures, Completeness: sh.completeness, Contents: contentsOf(f),
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return list, nil
}
