package financials

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// SourceSimulatedGSAP is GSAP data the simulated GSAP produced for a purchase request (033, ADR-0016).
const SourceSimulatedGSAP Source = "simulatedGsap"

type SimulatedDelivery struct {
	FeedName string
	FileName string
	Rows     []Fields
}

// RecordSimulatedGSAP books the delivery through the import's rules inside the caller's Write: one import record
// and one log entry. A row the rules refuse fails the whole delivery, so nothing of it is kept.
func (s *Service) RecordSimulatedGSAP(ctx context.Context, a actor.Actor, d SimulatedDelivery) (Import, error) {
	var imp Import
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		if err := s.access.Import(ctx, a); err != nil {
			return err
		}
		ap, err := s.applier(ctx, a, rowsFeed{name: d.FeedName, rows: d.Rows})
		if err != nil {
			return err
		}
		done, err := ap.apply(ctx, Delivery{FileName: d.FileName})
		if err != nil {
			return err
		}
		if len(done.problems) > 0 {
			return errs.NotAllowed(done.problems[0].Message)
		}
		_, err = s.history.Log(ctx, a, history.LogInput{
			Kind: importKind, RecordID: done.imp.ID, RecordName: done.imp.FileName, Action: history.ActionCreated,
			Verb: importedVerb, Details: countsText(done.imp.counts()),
		})
		imp = done.imp
		return err
	})
	return imp, err
}

type rowsFeed struct {
	name string
	rows []Fields
}

func (f rowsFeed) Name() string { return f.name }

func (rowsFeed) Source() Source { return SourceSimulatedGSAP }

func (f rowsFeed) Read(context.Context, Delivery) ([]Incoming, []RowProblem, []string, error) {
	rows := make([]Incoming, 0, len(f.rows))
	for i, r := range f.rows {
		rows = append(rows, Incoming{Row: i + 2, Fields: r})
	}
	return rows, []RowProblem{}, []string{}, nil
}
