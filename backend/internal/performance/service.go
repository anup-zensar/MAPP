package performance

import (
	"context"
	"fmt"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

const maxNameLength = 120

type Service struct {
	tx      Transactor
	repo    Store
	ids     *ids.Generator
	history *history.Service
	now     clock.Clock
}

func NewService(tools history.Tools, repo Store) *Service {
	return &Service{tx: tools.Tx, repo: repo, ids: tools.IDs, history: tools.Log, now: tools.Now}
}

type SourceInput struct {
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	OwnerID       string        `json:"ownerId"`
	Category      Category      `json:"category"`
	DeliveryRoute DeliveryRoute `json:"deliveryRoute"`
	Cadence       Cadence       `json:"cadence"`
	IntegrationID string        `json:"integrationId"`
}

func (s *Service) Sources(ctx context.Context, _ actor.Actor) ([]SourceView, error) {
	if err := authz.Require(ctx, authz.PlansView); err != nil {
		return nil, err
	}
	var sources []SourceView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		found, readErr := s.repo.ListSources(ctx, "")
		sources = found
		return readErr
	})
	if err != nil {
		return nil, fmt.Errorf("list performance sources: %w", err)
	}
	return sources, nil
}

func (s *Service) CreateSource(ctx context.Context, a actor.Actor, in SourceInput) (Source, error) {
	if err := authz.Require(ctx, PermissionImport); err != nil {
		return Source{}, err
	}
	if err := checkSource(in); err != nil {
		return Source{}, err
	}
	var created Source
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		id, idErr := s.ids.Next(ctx, ids.PrefixPerformanceSource)
		if idErr != nil {
			return idErr
		}
		created = Source{
			ID: id, Name: strings.TrimSpace(in.Name), Description: strings.TrimSpace(in.Description),
			OwnerID: in.OwnerID, Category: in.Category, Status: StatusActive,
			DeliveryRoute: in.DeliveryRoute, Cadence: in.Cadence,
			IntegrationID: strings.TrimSpace(in.IntegrationID),
			Meta:          record.NewMeta(a, s.now()),
		}
		if putErr := s.repo.PutSource(ctx, created); putErr != nil {
			return putErr
		}
		return s.log(ctx, a, created, logEntry{action: history.ActionCreated})
	})
	if err != nil {
		return Source{}, fmt.Errorf("create performance source: %w", err)
	}
	return created, nil
}

// DeleteSource refuses while the source holds facts and says Deactivate instead, because deleting it would
// silently restate every figure its data fed (FR-002).
func (s *Service) DeleteSource(ctx context.Context, a actor.Actor, id string) error {
	if err := authz.Require(ctx, PermissionImport); err != nil {
		return err
	}
	return s.tx.Write(ctx, func(ctx context.Context) error {
		found, err := s.repo.ListSources(ctx, id)
		if err != nil {
			return err
		}
		if len(found) == 0 {
			return errs.NotFound(msgSourceNotFound)
		}
		if found[0].FactCount > 0 {
			return errs.InUse(fmt.Sprintf(msgHoldsFacts, found[0].Name))
		}
		// MoveToBin writes the ActionDeleted entry itself, so this must not log again (one entry per change).
		_, err = s.history.MoveToBin(ctx, a, sourceKind, id)
		return err
	})
}

func (s *Service) SetSourceStatus(ctx context.Context, a actor.Actor, id string, status Status) error {
	if err := authz.Require(ctx, PermissionImport); err != nil {
		return err
	}
	return s.tx.Write(ctx, func(ctx context.Context) error {
		found, err := s.repo.ListSources(ctx, id)
		if err != nil {
			return err
		}
		if len(found) == 0 {
			return errs.NotFound(msgSourceNotFound)
		}
		next := found[0].Source
		next.Status = status
		next.Version++
		next.UpdatedAt, next.UpdatedBy = s.now(), a
		if putErr := s.repo.PutSource(ctx, next); putErr != nil {
			return putErr
		}
		verb := deactivatedVerb
		if status == StatusActive {
			verb = reactivatedVerb
		}
		return s.log(ctx, a, next, logEntry{action: history.ActionStatusChanged, verb: verb})
	})
}

// CheckSupplier refuses a second active source for a model's stage or revenue, naming the one that already
// supplies it. The rule spans two tables, so it cannot be a constraint (ADR-0021).
func (s *Service) CheckSupplier(ctx context.Context, modelID string, meaning Meaning, stage string) error {
	suppliers, err := s.repo.ActiveSuppliers(ctx, modelID)
	if err != nil {
		return err
	}
	for _, supplier := range suppliers {
		if supplier.Meaning != meaning {
			continue
		}
		if meaning == MeaningRevenue {
			return errs.NotAllowed(fmt.Sprintf(msgRevenueSupplied, supplier.SourceName))
		}
		if supplier.Stage == stage {
			return errs.NotAllowed(fmt.Sprintf(msgStageSupplied, supplier.SourceName, stage))
		}
	}
	return nil
}

func checkSource(in SourceInput) error {
	var problems []errs.FieldError
	if name := strings.TrimSpace(in.Name); name == "" || len(name) > maxNameLength {
		problems = append(problems, errs.FieldError{Field: "name", Message: msgNameLength})
	}
	if in.OwnerID == "" {
		problems = append(problems, errs.FieldError{Field: "ownerId", Message: msgOwnerNeeded})
	}
	if len(problems) > 0 {
		return errs.Validation(problems...)
	}
	return nil
}

type logEntry struct {
	action history.Action
	verb   string
}

func (s *Service) log(ctx context.Context, a actor.Actor, src Source, e logEntry) error {
	_, err := s.history.Log(ctx, a, history.LogInput{
		Kind: sourceKind, RecordID: src.ID, RecordName: src.Name, Action: e.action, Verb: e.verb,
	})
	return err
}
