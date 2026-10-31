package plan

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

const (
	maxSearchLength   = 200
	msgSearchEmpty    = "Type a name or ID to search for."
	msgSearchTooLong  = "Use 200 characters or fewer."
	LevelCampaign     = "campaign"
	LevelInitiative   = "initiative"
	LevelActivity     = "activity"
	LevelLineItem     = "lineItem"
	searchQueryField  = "q"
	unknownLevelOrder = 4
)

// SearchMatch is a record below a plan whose name or ID holds the searched text; ParentID is the plan's for a
// campaign, the campaign's for an initiative, the initiative's for an activity and the activity's for a line item.
type SearchMatch struct {
	Level    string `json:"level"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parentId"`
	PlanID   string `json:"-"`
}

type PlanSearch struct {
	Items []SearchMatch `json:"items"`
}

// FoundIn tells why the Plans list shows a plan whose own name and ID do not hold the text.
type FoundIn struct {
	Match SearchMatch `json:"match"`
	More  int         `json:"more"`
}

// Searcher finds, inside the caller's transaction, the records below planIDs whose name or ID holds lowerText.
type Searcher interface {
	Matching(ctx context.Context, planIDs []string, lowerText string) ([]SearchMatch, error)
}

func (s *Service) UseSearcher(c Searcher) {
	s.searcher = c
}

// Search lists the records below the plan planID whose name or ID holds q; the route rule has checked the scope.
func (s *Service) Search(ctx context.Context, a actor.Actor, planID, q string) (PlanSearch, error) {
	text := strings.TrimSpace(q)
	if err := searchProblems(text).Err(); err != nil {
		return PlanSearch{}, err
	}
	result := PlanSearch{Items: []SearchMatch{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		if _, err := s.stored(ctx, planID); err != nil {
			return err
		}
		var err error
		result.Items, err = s.matching(ctx, []string{planID}, text)
		return err
	})
	if err != nil {
		return PlanSearch{}, fmt.Errorf("search plan %s for %s: %w", planID, a.ID, err)
	}
	return result, nil
}

func searchProblems(text string) errs.Problems {
	var problems errs.Problems
	switch {
	case text == "":
		problems.Add(searchQueryField, msgSearchEmpty)
	case utf8.RuneCountInString(text) > maxSearchLength:
		problems.Add(searchQueryField, msgSearchTooLong)
	}
	return problems
}

func (s *Service) matching(ctx context.Context, planIDs []string, text string) ([]SearchMatch, error) {
	if s.searcher == nil || len(planIDs) == 0 || text == "" {
		return []SearchMatch{}, nil
	}
	found, err := s.searcher.Matching(ctx, planIDs, strings.ToLower(text))
	if err != nil {
		return nil, err
	}
	slices.SortFunc(found, func(x, y SearchMatch) int {
		return cmp.Or(cmp.Compare(levelOrder(x.Level), levelOrder(y.Level)),
			strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name)), strings.Compare(x.ID, y.ID))
	})
	return found, nil
}

func levelOrder(level string) int {
	switch level {
	case LevelCampaign:
		return 0
	case LevelInitiative:
		return 1
	case LevelActivity:
		return 2
	case LevelLineItem:
		return 3
	}
	return unknownLevelOrder
}

// foundIn gives each plan of planIDs with a match its first match and how many more.
func (s *Service) foundIn(ctx context.Context, planIDs []string, text string) (map[string]*FoundIn, error) {
	matches, err := s.matching(ctx, planIDs, text)
	if err != nil {
		return nil, err
	}
	found := map[string]*FoundIn{}
	for _, m := range matches {
		if f, seen := found[m.PlanID]; seen {
			f.More++
			continue
		}
		found[m.PlanID] = &FoundIn{Match: m}
	}
	return found, nil
}
