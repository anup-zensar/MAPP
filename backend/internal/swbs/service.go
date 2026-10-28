package swbs

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

const (
	maxNamed     = 200
	maxIDLength  = 40
	defaultLimit = 20
	maxLimit     = 100
)

type Service struct {
	tx      Transactor
	repo    Store
	ids     *ids.Generator
	history *history.Service
	now     clock.Clock
	items   LineItems
	gsap    GSAP
}

func NewService(tools history.Tools, repo Store, items LineItems, gsap GSAP) *Service {
	return &Service{tx: tools.Tx, repo: repo, ids: tools.IDs, history: tools.Log, now: tools.Now, items: items, gsap: gsap}
}

// Run sends the ready line items in a's finance.import scope, or only in.LineItemIDs, in one Write: the codes
// land on the line items, the run is kept and logged once, or nothing is saved (FR-006 – FR-010).
func (s *Service) Run(ctx context.Context, a actor.Actor, in RunInput) (RunView, error) {
	if err := authz.Require(ctx, authz.FinanceImport); err != nil {
		return RunView{}, err
	}
	if err := checkInput(in); err != nil {
		return RunView{}, err
	}
	var run Run
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		requests, err := s.ready(ctx, in.LineItemIDs)
		if err != nil {
			return err
		}
		d := delivery{repo: s.repo, ids: s.ids, gsap: s.gsap, items: s.items, actor: a, source: SourceSimulated, at: s.now()}
		if run, err = d.send(ctx, requests); err != nil {
			return err
		}
		_, err = s.history.Log(ctx, a, history.LogInput{
			Kind: runKind, RecordID: run.ID, RecordName: sourceSimulatedText, Action: history.ActionCreated, Verb: ranVerb,
			Details: detailsOf(run.Codes),
		})
		return err
	})
	if err != nil {
		return RunView{}, err
	}
	return run.view(), nil
}

// Runs returns the latest runs, newest first, and how many line items are ready to send now (FR-011).
func (s *Service) Runs(ctx context.Context, _ actor.Actor, limit int) (RunPage, error) {
	if err := authz.Require(ctx, authz.FinanceImport); err != nil {
		return RunPage{}, err
	}
	limit = cmp.Or(limit, defaultLimit)
	if limit < 1 || limit > maxLimit {
		return RunPage{}, errs.Validation(errs.FieldError{Field: "limit", Message: msgLimit})
	}
	page := RunPage{Items: []RunView{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		runs, total, err := s.repo.ListRuns(ctx, limit)
		if err != nil {
			return err
		}
		for _, r := range runs {
			page.Items = append(page.Items, r.view())
		}
		page.Total = total
		ready, err := s.ready(ctx, nil)
		page.ReadyToSend = len(ready)
		return err
	})
	if err != nil {
		return RunPage{}, err
	}
	return page, nil
}

func (s *Service) ready(ctx context.Context, named []string) ([]Request, error) {
	candidates, err := s.items.Candidates(ctx, authz.PermissionScopeOf(ctx, authz.FinanceImport), named)
	if err != nil {
		return nil, err
	}
	if err := checkNamed(named, candidates); err != nil {
		return nil, err
	}
	requests := []Request{}
	for _, c := range candidates {
		if c.Ready {
			requests = append(requests, c.Request)
		}
	}
	slices.SortFunc(requests, func(x, y Request) int {
		return cmp.Or(cmp.Compare(len(x.LineItemID), len(y.LineItemID)), strings.Compare(x.LineItemID, y.LineItemID))
	})
	return requests, nil
}

func checkInput(in RunInput) error {
	var problems errs.Problems
	switch {
	case in.LineItemIDs == nil:
	case len(in.LineItemIDs) == 0:
		problems.Add("lineItemIds", msgEmptyList)
	case len(in.LineItemIDs) > maxNamed:
		problems.Add("lineItemIds", msgTooMany)
	case slices.ContainsFunc(in.LineItemIDs, func(id string) bool { return id == "" || len(id) > maxIDLength }):
		problems.Add("lineItemIds", msgBadID)
	}
	return problems.Err()
}

// checkNamed answers not found for a named line item outside the candidates and refuses, naming them, those
// that are not ready (FR-008).
func checkNamed(named []string, candidates []Candidate) error {
	var notReady []string
	for _, id := range named {
		i := slices.IndexFunc(candidates, func(c Candidate) bool { return c.LineItemID == id })
		if i < 0 {
			return errs.NotFound(msgNotFound)
		}
		name := fmt.Sprintf(lineItemText, candidates[i].LineItemName, id)
		if !candidates[i].Ready && !slices.Contains(notReady, name) {
			notReady = append(notReady, name)
		}
	}
	if len(notReady) > 0 {
		return errs.NotAllowed(fmt.Sprintf(msgNotReady, strings.Join(notReady, ", ")))
	}
	return nil
}

type delivery struct {
	repo   Store
	ids    *ids.Generator
	gsap   GSAP
	items  Receiver
	actor  actor.Actor
	source Source
	at     time.Time
}

func (d delivery) send(ctx context.Context, requests []Request) (Run, error) {
	answers, err := d.gsap.RequestCodes(ctx, requests)
	if err != nil {
		return Run{}, err
	}
	codes, err := matched(requests, answers)
	if err != nil {
		return Run{}, err
	}
	if len(answers) > 0 {
		if receiveErr := d.items.Receive(ctx, d.actor, answers, d.at); receiveErr != nil {
			return Run{}, receiveErr
		}
	}
	id, err := d.ids.Next(ctx, ids.PrefixSwbsRun)
	if err != nil {
		return Run{}, err
	}
	run := Run{ID: id, Source: d.source, Codes: codes, Meta: record.NewMeta(d.actor, d.at)}
	return run, d.repo.AddRun(ctx, run)
}

func matched(requests []Request, answers []Answer) ([]Code, error) {
	if len(answers) != len(requests) {
		return nil, fmt.Errorf("swbs: GSAP answered %d of %d requests", len(answers), len(requests))
	}
	codes := make([]Code, 0, len(requests))
	for i, r := range requests {
		if answers[i].LineItemID != r.LineItemID || answers[i].SwbsCode == "" {
			return nil, fmt.Errorf("swbs: GSAP's answer %d is not for line item %s", i+1, r.LineItemID)
		}
		codes = append(codes, Code{
			LineItemID: r.LineItemID, LineItemName: r.LineItemName, CompanyCode: r.CompanyCode, SwbsCode: answers[i].SwbsCode,
		})
	}
	return codes, nil
}
