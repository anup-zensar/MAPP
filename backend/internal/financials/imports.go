package financials

import (
	"context"
	"encoding/base64"
	"errors"
	"slices"
	"strings"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
)

// Imports returns every import, newest first (FR-008).
func (s *Service) Imports(ctx context.Context, a actor.Actor) (ImportPage, error) {
	page := ImportPage{Items: []ImportView{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		if err := s.access.SeeFinance(ctx, a); err != nil {
			return err
		}
		imports, err := s.repo.ListImports(ctx)
		if err != nil {
			return err
		}
		for _, i := range slices.Backward(imports) {
			page.Items = append(page.Items, i.view())
		}
		return nil
	})
	if err != nil {
		return ImportPage{}, err
	}
	page.Total = len(page.Items)
	return page, nil
}

// errDiscard ends the Write of a check, so that nothing it did is saved (D-177).
var errDiscard = errors.New("financials: check only, discard the changes")

// Check says what importing the file would do, without saving anything (FR-004): it runs the import inside
// a Write and throws it away, so the check and the import cannot disagree (SC-003). Only people who may
// import may check (FR-028).
func (s *Service) Check(ctx context.Context, a actor.Actor, req ImportRequest) (ImportCheck, error) {
	var check ImportCheck
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		result, err := s.apply(ctx, a, req)
		if err != nil {
			return err
		}
		problems, notLinked := capped(result.problems), capped(result.notLinked)
		check = ImportCheck{
			FileName: result.imp.FileName, Counts: result.imp.counts(), Problems: importProblems(problems),
			ProblemsTotal: len(result.problems), NotLinked: notLinked, NotLinkedTotal: len(result.notLinked),
			UnusedColumns: result.unused,
		}
		return errDiscard
	})
	if !errors.Is(err, errDiscard) {
		return ImportCheck{}, err
	}
	return check, nil
}

// Import imports the file (FR-004 – FR-007): its valid rows are applied, refused rows skipped and listed,
// every not-linked transaction matched again; one import record and one activity-log entry are written
// with the counts (FR-027). A problem with the whole file saves nothing.
func (s *Service) Import(ctx context.Context, a actor.Actor, req ImportRequest) (ImportResult, error) {
	var result ImportResult
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		done, err := s.apply(ctx, a, req)
		if err != nil {
			return err
		}
		_, err = s.history.Log(ctx, a, history.LogInput{
			Kind: importKind, RecordID: done.imp.ID, RecordName: done.imp.FileName, Action: history.ActionCreated,
			Verb: importedVerb, Details: countsText(done.imp.counts()),
		})
		if err != nil {
			return err
		}
		result = ImportResult{
			Import: done.imp.view(), Problems: importProblems(capped(done.problems)), ProblemsTotal: len(done.problems),
			NotLinked: capped(done.notLinked), NotLinkedTotal: len(done.notLinked),
		}
		return nil
	})
	if err != nil {
		return ImportResult{}, err
	}
	return result, nil
}

// apply checks a's right to import, decodes the file and applies it with the file feed.
func (s *Service) apply(ctx context.Context, a actor.Actor, req ImportRequest) (applied, error) {
	if err := s.access.Import(ctx, a); err != nil {
		return applied{}, err
	}
	content, err := base64.StdEncoding.DecodeString(req.Content)
	if err != nil {
		return applied{}, contentProblem(msgFileUnreadable)
	}
	ap, err := s.applier(ctx, a, s.feed)
	if err != nil {
		return applied{}, err
	}
	return ap.apply(ctx, Delivery{FileName: strings.TrimSpace(req.FileName), Content: content})
}

// applier returns the applier of a delivery by a with feed.
func (s *Service) applier(ctx context.Context, a actor.Actor, feed Feed) (applier, error) {
	nodes, err := s.planning.Nodes(ctx)
	if err != nil {
		return applier{}, err
	}
	inBin, err := s.planning.LineItemsInBin(ctx)
	if err != nil {
		return applier{}, err
	}
	items := storedSet{}
	for _, n := range nodes {
		if n.Level == LevelLineItem {
			items[n.ID] = true
		}
	}
	for _, id := range inBin {
		items[id] = true
	}
	rates, err := s.rates.Rates(ctx)
	if err != nil {
		return applier{}, err
	}
	return applier{repo: s.repo, ids: s.ids, items: items, rates: rates, actor: a, now: s.now(), feed: feed}, nil
}

// DemoLoad is what LoadDemo matches and dates the demo data with: the line items the seed stored, the
// rates of its currencies and the seed time.
type DemoLoad struct {
	Items LineItemStates
	Rates money.Rates
	At    time.Time
}

// LoadDemo loads the demo data of FR-031 inside the caller's Write: the demo feed's 11 transactions
// through the same apply code as an import, as MAPP (actor.System) at the time demo.At, and one import
// record "Demo data (simulated GSAP)". It writes no activity-log entry (D-188). The seed calls it.
func LoadDemo(ctx context.Context, repo Store, generator *ids.Generator, demo DemoLoad) (Import, error) {
	ap := applier{
		repo: repo, ids: generator, items: demo.Items, rates: demo.Rates, actor: actor.System, now: demo.At,
		feed: DemoFeed{},
	}
	done, err := ap.apply(ctx, Delivery{})
	if err != nil {
		return Import{}, err
	}
	return done.imp, nil
}

// capped returns at most the first maxListed of list.
func capped[T any](list []T) []T {
	return list[:min(maxListed, len(list))]
}

// importProblems returns row problems as the API answers them.
func importProblems(problems []RowProblem) []ImportProblem {
	list := make([]ImportProblem, 0, len(problems))
	for _, p := range problems {
		list = append(list, ImportProblem{Row: p.Row, Column: p.Column, Message: p.Message})
	}
	return list
}
