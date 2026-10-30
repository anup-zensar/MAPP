package performance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/database/sqlcdb"
)

const maxRows = 500

// rowLimit bounds a list, which the standards require of every list endpoint, and makes the narrowing to
// int32 safe by construction rather than by assertion.
func rowLimit(limit int) int32 {
	if limit < 1 || limit > maxRows {
		return maxRows
	}
	return int32(limit)
}

func (s *PostgresStore) ListImports(ctx context.Context, id, sourceID string, limit int) ([]Import, error) {
	rows, err := s.db.Queries(ctx).ListPerformanceImports(ctx, sqlcdb.ListPerformanceImportsParams{
		ID: optional(id), SourceID: optional(sourceID), MaxImports: rowLimit(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("select performance imports: %w", err)
	}
	imports := make([]Import, 0, len(rows))
	for _, r := range rows {
		p := r.PerformanceImport
		item := Import{
			ID: p.ID, SourceID: p.SourceID, FileName: p.FileName, Origin: Origin(p.Origin),
			Status: ImportStatus(p.Status),
			Counts: Counts{Read: p.RowsRead, Matched: p.RowsMatched, Held: p.RowsHeld,
				Unattributed: p.RowsUnattributed, Rejected: p.RowsRejected, Unmatched: p.RowsUnmatched},
			Meta: metaOf(p.Version, metaRow{p.CreatedAt, p.CreatedByID, p.CreatedByName,
				p.UpdatedAt, p.UpdatedByID, p.UpdatedByName}),
		}
		if p.DataThrough != nil {
			item.DataThrough = p.DataThrough
		}
		imports = append(imports, item)
	}
	return imports, nil
}

func (s *PostgresStore) PutImport(ctx context.Context, i Import) error {
	err := s.db.Queries(ctx).PutPerformanceImport(ctx, sqlcdb.PutPerformanceImportParams{
		ID: i.ID, SourceID: i.SourceID, FileName: i.FileName, Origin: string(i.Origin),
		Status: string(i.Status), DataThrough: i.DataThrough,
		RowsRead: i.Counts.Read, RowsMatched: i.Counts.Matched, RowsHeld: i.Counts.Held,
		RowsUnattributed: i.Counts.Unattributed, RowsRejected: i.Counts.Rejected,
		RowsUnmatched: i.Counts.Unmatched,
		Version:       i.Version, CreatedAt: i.CreatedAt, CreatedByID: i.CreatedBy.ID,
		CreatedByName: i.CreatedBy.Name, UpdatedAt: i.UpdatedAt, UpdatedByID: i.UpdatedBy.ID,
		UpdatedByName: i.UpdatedBy.Name,
	})
	if err != nil {
		return fmt.Errorf("save performance import %s: %w", i.ID, err)
	}
	return nil
}

func (s *PostgresStore) ListReviewRows(ctx context.Context, status, sourceID string, limit int) ([]ReviewRowView, error) {
	rows, err := s.db.Queries(ctx).ListPerformanceReviewRows(ctx, sqlcdb.ListPerformanceReviewRowsParams{
		Status: optional(status), SourceID: optional(sourceID), MaxRows: rowLimit(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("select rows to review: %w", err)
	}
	reviewed := make([]ReviewRowView, 0, len(rows))
	for _, r := range rows {
		p := r.PerformanceReviewRow
		row := ReviewRow{
			ID: p.ID, ImportID: p.ImportID, SourceID: p.SourceID, Raw: p.Raw, Reason: p.Reason,
			Status: ReviewStatus(p.Status), Comment: p.Comment,
			Meta: metaOf(p.Version, metaRow{p.CreatedAt, p.CreatedByID, p.CreatedByName,
				p.UpdatedAt, p.UpdatedByID, p.UpdatedByName}),
		}
		if len(p.Candidates) > 0 {
			if err := json.Unmarshal(p.Candidates, &row.Candidates); err != nil {
				return nil, fmt.Errorf("read the candidates of review row %s: %w", p.ID, err)
			}
		}
		reviewed = append(reviewed, ReviewRowView{ReviewRow: row, SourceName: r.SourceName})
	}
	return reviewed, nil
}

func (s *PostgresStore) PutReviewRow(ctx context.Context, r ReviewRow) error {
	var candidates []byte
	if len(r.Candidates) > 0 {
		encoded, err := json.Marshal(r.Candidates)
		if err != nil {
			return fmt.Errorf("write the candidates of review row %s: %w", r.ID, err)
		}
		candidates = encoded
	}
	err := s.db.Queries(ctx).PutPerformanceReviewRow(ctx, sqlcdb.PutPerformanceReviewRowParams{
		ID: r.ID, ImportID: r.ImportID, SourceID: r.SourceID, Raw: r.Raw, Reason: r.Reason,
		Candidates: candidates, Status: string(r.Status), Comment: r.Comment,
		Version: r.Version, CreatedAt: r.CreatedAt, CreatedByID: r.CreatedBy.ID,
		CreatedByName: r.CreatedBy.Name, UpdatedAt: r.UpdatedAt, UpdatedByID: r.UpdatedBy.ID,
		UpdatedByName: r.UpdatedBy.Name,
	})
	if err != nil {
		return fmt.Errorf("save review row %s: %w", r.ID, err)
	}
	return nil
}

func (s *PostgresStore) ListMediaCodes(ctx context.Context, scope authz.Scope, activityID string) ([]MediaCode, error) {
	a := argsOf(scope)
	rows, err := s.db.Queries(ctx).ListMediaCodesForActivity(ctx, sqlcdb.ListMediaCodesForActivityParams{
		ActivityID: activityID, SeesEverything: a.seesEverything, ViewerID: a.viewerID,
		ViewsPlans: a.viewsPlans, ViewerTeamIds: a.teamIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("select the media codes of activity %s: %w", activityID, err)
	}
	codes := make([]MediaCode, 0, len(rows))
	for _, r := range rows {
		m := r.MediaCode
		code := MediaCode{
			ID: m.ID, Code: m.Code, ActivityID: m.ActivityID, LandingPage: m.LandingPage,
			Status: Status(m.Status),
			Meta: metaOf(m.Version, metaRow{m.CreatedAt, m.CreatedByID, m.CreatedByName,
				m.UpdatedAt, m.UpdatedByID, m.UpdatedByName}),
		}
		if len(m.Attributes) > 0 {
			if err := json.Unmarshal(m.Attributes, &code.Attributes); err != nil {
				return nil, fmt.Errorf("read the attributes of media code %s: %w", m.ID, err)
			}
		}
		codes = append(codes, code)
	}
	return codes, nil
}

// TakeNextCode raises the stored next number and returns the one it replaced, in a single statement, so two
// codes can never take the same number (FR-042).
func (s *PostgresStore) TakeNextCode(ctx context.Context, by actor.Actor, at time.Time) (int64, error) {
	code, err := s.db.Queries(ctx).TakeNextMediaCode(ctx, sqlcdb.TakeNextMediaCodeParams{
		UpdatedAt: at, UpdatedByID: by.ID, UpdatedByName: by.Name,
	})
	if err != nil {
		return 0, fmt.Errorf("take the next media code: %w", err)
	}
	return code, nil
}
