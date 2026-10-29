package bulk

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

const maxTargets = 100

// Fields lists what Change field offers for a level (FR-009).
func (s *Service) Fields(ctx context.Context, _ actor.Actor, level Level) (FieldList, error) {
	if level != LevelCampaign && level != LevelInitiative && level != LevelActivity {
		return FieldList{}, errs.Validation(errs.FieldError{Field: "level", Message: msgChooseOne})
	}
	items, err := s.planning.Fields(ctx, level)
	if err != nil {
		return FieldList{}, err
	}
	if items == nil {
		items = []Field{}
	}
	return FieldList{Level: level, Items: items, Total: len(items), NotOffered: notOfferedText}, nil
}

// Targets lists where the selected records may be copied or moved: the destinations the single action
// offers the first of them, when they share a plan (a year for campaigns), without the selected records,
// searched by name, ID or path (FR-016).
func (s *Service) Targets(ctx context.Context, a actor.Actor, q TargetQuery) (TargetPage, error) {
	recordIDs := uniqueIDs(q.RecordIDs)
	if err := checkTargetQuery(q, recordIDs); err != nil {
		return TargetPage{}, err
	}
	page := TargetPage{Items: []Target{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		records, err := s.found(ctx, a, recordIDs)
		if err != nil || len(records) == 0 {
			return err
		}
		if note := placeNote(q.Action, records); note != "" {
			page.Note = &note
			return nil
		}
		targets, err := s.planning.Destinations(ctx, a, q.Action, records[0])
		page.Items, page.Total = matching(targets, recordIDs, q.Search)
		return err
	})
	if err != nil {
		return TargetPage{}, err
	}
	if page.Total == 0 && page.Note == nil {
		note := fmt.Sprintf(msgNoTarget, nouns[q.Action])
		page.Note = &note
	}
	return page, nil
}

func checkTargetQuery(q TargetQuery, recordIDs []string) error {
	var p errs.Problems
	if q.Action != ActionCopy && q.Action != ActionMove {
		p.Add("action", msgChooseAction)
	}
	switch {
	case len(recordIDs) == 0:
		p.Add(fieldRecordIDs, msgSelectOne)
	case len(recordIDs) > maxRecords:
		p.Add(fieldRecordIDs, msgTooMany)
	case slices.ContainsFunc(recordIDs, func(id string) bool { _, ok := levelOf(id); return !ok }):
		p.Add(fieldRecordIDs, msgNotSelectable)
	case !isOneLevel(countByLevel(recordIDs)):
		p.Add(fieldRecordIDs, fmt.Sprintf(msgOneLevel, actionWord(q.Action)))
	}
	return p.Err()
}

// found reads the selected records the person sees, leaving out the others.
func (s *Service) found(ctx context.Context, a actor.Actor, recordIDs []string) ([]Record, error) {
	records := []Record{}
	for _, id := range recordIDs {
		rec, err := s.planning.Find(ctx, a, id)
		if apiErr, ok := errors.AsType[*errs.Error](err); ok && apiErr.Code == errs.CodeNotFound {
			continue
		}
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, nil
}

// placeNote says why records of several plans (campaigns: several years) have no common target (D-033).
func placeNote(action Kind, records []Record) string {
	first := records[0]
	if first.Level == LevelCampaign {
		if slices.ContainsFunc(records, func(r Record) bool { return r.Year != first.Year }) {
			return msgNoTargetYear
		}
		return ""
	}
	if slices.ContainsFunc(records, func(r Record) bool { return r.PlanID != first.PlanID }) {
		return fmt.Sprintf(msgNoTargetPlan, actionWord(action))
	}
	return ""
}

func matching(targets []Target, selected []string, search string) ([]Target, int) {
	words := strings.ToLower(strings.TrimSpace(search))
	kept := []Target{}
	for _, t := range targets {
		if slices.Contains(selected, t.ID) {
			continue
		}
		text := strings.ToLower(t.Name + " " + t.ID + " " + t.Path)
		if words == "" || strings.Contains(text, words) {
			kept = append(kept, t)
		}
	}
	slices.SortStableFunc(kept, func(x, y Target) int {
		return cmp.Or(strings.Compare(strings.ToLower(x.Path), strings.ToLower(y.Path)), strings.Compare(x.ID, y.ID))
	})
	return kept[:min(len(kept), maxTargets)], len(kept)
}
