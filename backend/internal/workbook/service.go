package workbook

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/sheet"
)

const maxListed = 100

// errDiscard ends the Write of a check, so that nothing it did is saved (D-126).
var errDiscard = errors.New("workbook: check only, discard the changes")

type Service struct {
	tx       Transactor
	repo     Store
	ids      *ids.Generator
	history  *history.Service
	now      clock.Clock
	planning Planning
}

type File struct {
	Name string
	Data []byte
}

func NewService(tools history.Tools, repo Store, planning Planning) *Service {
	return &Service{tx: tools.Tx, repo: repo, ids: tools.IDs, history: tools.Log, now: tools.Now, planning: planning}
}

// Template is the plan workbook without rows; Export holds every record of the plan (FR-027).
func (s *Service) Template(ctx context.Context, a actor.Actor, planID string) (File, error) {
	return s.file(ctx, a, planID, false)
}

func (s *Service) Export(ctx context.Context, a actor.Actor, planID string) (File, error) {
	return s.file(ctx, a, planID, true)
}

func (s *Service) file(ctx context.Context, a actor.Actor, planID string, withRecords bool) (File, error) {
	var plan Plan
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		plan, err = s.planning.Plan(ctx, a, planID)
		return err
	})
	if err != nil {
		return File{}, err
	}
	var buf bytes.Buffer
	if err := sheet.WriteXLSX(&buf, buildWorkbook(plan, withRecords)); err != nil {
		return File{}, err
	}
	name := fmt.Sprintf(templateFileName, plan.Name)
	if withRecords {
		name = fmt.Sprintf(exportFileName, plan.Name)
	}
	return File{Name: name, Data: buf.Bytes()}, nil
}

// Check runs the import and discards it: every problem of every row and what Import would do (FR-031).
// The file is read before the Write, which holds back other people's saves (038 FR-009).
func (s *Service) Check(ctx context.Context, a actor.Actor, planID string, up Upload) (Check, error) {
	tables, err := tablesOf(up)
	if err != nil {
		return Check{}, err
	}
	var result Check
	err = s.tx.Write(ctx, func(ctx context.Context) error {
		ap, err := s.apply(ctx, a, planID, up.FileName, tables)
		if err != nil {
			return err
		}
		result = ap.result
		return errDiscard
	})
	if err != nil && !errors.Is(err, errDiscard) {
		return Check{}, err
	}
	return result, nil
}

// Import applies the ready rows of the check in one Write and keeps the import for its plan (FR-034, FR-035).
func (s *Service) Import(ctx context.Context, a actor.Actor, planID string, up Upload) (Detail, error) {
	tables, err := tablesOf(up)
	if err != nil {
		return Detail{}, err
	}
	var imp Import
	err = s.tx.Write(ctx, func(ctx context.Context) error {
		id, err := s.ids.Next(ctx, ids.PrefixPlanImport)
		if err != nil {
			return err
		}
		ap, err := s.apply(history.WithNote(ctx, fmt.Sprintf(noteImported, up.FileName, id)), a, planID, up.FileName, tables)
		if err != nil {
			return err
		}
		imp = Import{ID: id, PlanID: planID, FileName: up.FileName, ImportedAt: s.now(), ImportedBy: a,
			RowsRead: ap.result.Rows, RowsLeftOut: ap.result.WithProblems, Records: ap.records, Problems: ap.result.Problems}
		return s.repo.AddImport(ctx, imp)
	})
	if err != nil {
		return Detail{}, err
	}
	return s.Get(ctx, a, imp.ID)
}

func (s *Service) apply(ctx context.Context, a actor.Actor, planID, fileName string, tables []sheet.Table) (*applier, error) {
	plan, err := s.planning.Plan(ctx, a, planID)
	if err != nil {
		return nil, err
	}
	sheets, ignored, err := levelSheets(tables, plan.Choices)
	if err != nil {
		return nil, err
	}
	writer, err := s.planning.Prepare(ctx, a, plan)
	if err != nil {
		return nil, err
	}
	ap := newApplier(a, writer, plan)
	ap.result.FileName, ap.result.IgnoredColumns = fileName, ignored
	return ap, ap.run(ctx, sheets)
}

// List returns the plan's imports, newest first (FR-035).
func (s *Service) List(ctx context.Context, _ actor.Actor, planID string) (Page, error) {
	page := Page{Items: []Summary{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		imports, err := s.repo.ListImports(ctx, authz.ScopeOf(ctx), planID, maxListed)
		for _, imp := range imports {
			page.Items = append(page.Items, summaryOf(imp))
		}
		page.Total = len(page.Items)
		return err
	})
	if err != nil {
		return Page{}, err
	}
	return page, nil
}

func summaryOf(imp Import) Summary {
	sum := Summary{ID: imp.ID, PlanID: imp.PlanID, FileName: imp.FileName, Status: StatusDone, ImportedAt: imp.ImportedAt,
		ImportedBy: imp.ImportedBy, RowsRead: imp.RowsRead, RowsLeftOut: imp.RowsLeftOut, UndoneAt: imp.UndoneAt, UndoneBy: imp.UndoneBy}
	if imp.UndoneAt != nil {
		sum.Status = StatusUndone
	}
	for _, r := range imp.Records {
		totals := &sum.Updated
		if r.Action == ActionCreated {
			totals = &sum.Created
		}
		totals.add(r.Level, 1)
	}
	return sum
}

func (t *LevelTotals) add(l Level, n int) {
	switch l {
	case LevelCampaign:
		t.Campaigns += n
	case LevelInitiative:
		t.Initiatives += n
	case LevelActivity:
		t.Activities += n
	case LevelLineItem:
		t.LineItems += n
	}
}
