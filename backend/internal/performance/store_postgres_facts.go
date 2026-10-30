package performance

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/money"
)

func factOf(f sqlcdb.PerformanceFact) Fact {
	fact := Fact{
		ID: f.ID, SourceID: f.SourceID, ImportID: f.ImportID, SourceReference: f.SourceReference,
		ActivityID: text(f.ActivityID), MatchedOn: MatchedOn(f.MatchedOn), FactDate: f.FactDate,
		Stage: f.Stage, WebMetric: WebMetric(f.WebMetric), Count: f.CountValue,
		Held: f.Held, ReplacedByID: text(f.ReplacedByID),
		Meta: metaOf(f.Version, metaRow{f.CreatedAt, f.CreatedByID, f.CreatedByName,
			f.UpdatedAt, f.UpdatedByID, f.UpdatedByName}),
	}
	if f.AmountMinor != nil && f.Currency != nil {
		fact.Amount = &money.Money{Amount: money.FromCents(*f.AmountMinor), Currency: *f.Currency}
	}
	return fact
}

func factParams(f Fact) sqlcdb.PutPerformanceFactParams {
	p := sqlcdb.PutPerformanceFactParams{
		ID: f.ID, SourceID: f.SourceID, ImportID: f.ImportID, SourceReference: f.SourceReference,
		ActivityID: optional(f.ActivityID), MatchedOn: string(f.MatchedOn), FactDate: f.FactDate,
		Stage: f.Stage, WebMetric: string(f.WebMetric), CountValue: f.Count,
		Held: f.Held, ReplacedByID: optional(f.ReplacedByID),
		Version: f.Version, CreatedAt: f.CreatedAt, CreatedByID: f.CreatedBy.ID,
		CreatedByName: f.CreatedBy.Name, UpdatedAt: f.UpdatedAt, UpdatedByID: f.UpdatedBy.ID,
		UpdatedByName: f.UpdatedBy.Name,
	}
	if f.Amount != nil {
		cents := f.Amount.Amount.Cents()
		currency := f.Amount.Currency
		p.AmountMinor, p.Currency = &cents, &currency
	}
	return p
}

func (s *PostgresStore) ListFactsForActivity(ctx context.Context, scope authz.Scope, activityID string, countedOnly bool) ([]FactView, error) {
	a := argsOf(scope)
	rows, err := s.db.Queries(ctx).ListFactsForActivity(ctx, sqlcdb.ListFactsForActivityParams{
		ActivityID: optional(activityID), CountedOnly: countedOnly, SeesEverything: a.seesEverything,
		ViewerID: a.viewerID, ViewsPlans: a.viewsPlans, ViewerTeamIds: a.teamIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("select the facts of activity %s: %w", activityID, err)
	}
	facts := make([]FactView, 0, len(rows))
	for _, r := range rows {
		facts = append(facts, FactView{Fact: factOf(r.PerformanceFact), SourceName: r.SourceName})
	}
	return facts, nil
}

func (s *PostgresStore) PutFact(ctx context.Context, f Fact) error {
	if err := s.db.Queries(ctx).PutPerformanceFact(ctx, factParams(f)); err != nil {
		return fmt.Errorf("save performance fact %s: %w", f.ID, err)
	}
	return nil
}

func (s *PostgresStore) CurrentFactByReference(ctx context.Context, sourceID, reference string) (Fact, bool, error) {
	row, err := s.db.Queries(ctx).CurrentFactByReference(ctx, sqlcdb.CurrentFactByReferenceParams{
		SourceID: sourceID, SourceReference: reference,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Fact{}, false, nil
	}
	if err != nil {
		return Fact{}, false, fmt.Errorf("select the current fact of %s/%s: %w", sourceID, reference, err)
	}
	return factOf(row.PerformanceFact), true, nil
}

// ReleaseHeld clears the hold on an activity's facts in one statement, inside the caller's transaction, so
// approval and release commit together (research R4).
func (s *PostgresStore) ReleaseHeld(ctx context.Context, activityID string, by actor.Actor, at time.Time) (int, error) {
	released, err := s.db.Queries(ctx).ReleaseHeldFactsForActivity(ctx, sqlcdb.ReleaseHeldFactsForActivityParams{
		ActivityID: optional(activityID), UpdatedAt: at, UpdatedByID: by.ID, UpdatedByName: by.Name,
	})
	if err != nil {
		return 0, fmt.Errorf("release the held facts of activity %s: %w", activityID, err)
	}
	return int(released), nil
}
