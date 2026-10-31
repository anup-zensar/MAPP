package performance

import (
	"context"
	"fmt"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

type MappingInput struct {
	ColumnName string    `json:"columnName"`
	Meaning    Meaning   `json:"meaning"`
	Stage      string    `json:"stage"`
	WebMetric  WebMetric `json:"webMetric"`
	ModelID    string    `json:"modelId"`
	Version    int       `json:"version"`
}

func (s *Service) Mappings(ctx context.Context, _ actor.Actor, sourceID string) ([]Mapping, error) {
	if err := authz.Require(ctx, authz.PlansView); err != nil {
		return nil, err
	}
	var found []Mapping
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var readErr error
		found, readErr = s.repo.ListMappings(ctx, []string{sourceID})
		return readErr
	})
	if err != nil {
		return nil, fmt.Errorf("list the mappings of source %s: %w", sourceID, err)
	}
	return found, nil
}

func (s *Service) CreateMapping(ctx context.Context, a actor.Actor, sourceID string, in MappingInput) (Mapping, error) {
	if err := authz.Require(ctx, PermissionImport); err != nil {
		return Mapping{}, err
	}
	if err := checkMapping(in); err != nil {
		return Mapping{}, err
	}
	var created Mapping
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		if err := s.mappableSource(ctx, sourceID, in); err != nil {
			return err
		}
		id, idErr := s.ids.Next(ctx, ids.PrefixPerformanceMapping)
		if idErr != nil {
			return idErr
		}
		created = Mapping{
			ID: id, SourceID: sourceID, ColumnName: strings.TrimSpace(in.ColumnName),
			Meaning: in.Meaning, Stage: in.Stage, WebMetric: in.WebMetric, ModelID: in.ModelID,
			Meta: record.NewMeta(a, s.now()),
		}
		return s.repo.PutMapping(ctx, created)
	})
	if err != nil {
		return Mapping{}, fmt.Errorf("create a mapping on source %s: %w", sourceID, err)
	}
	return created, nil
}

// mappableSource refuses a mapping on a source that is gone or deactivated, and a second active supplier
// for the model's stage or revenue (FR-003, FR-004).
func (s *Service) mappableSource(ctx context.Context, sourceID string, in MappingInput) error {
	sources, err := s.repo.ListSources(ctx, sourceID)
	if err != nil {
		return err
	}
	if len(sources) == 0 {
		return errs.NotFound(msgSourceNotFound)
	}
	if sources[0].Status == StatusInactive {
		return errs.NotAllowed(fmt.Sprintf(msgSourceInactive, sources[0].Name))
	}
	if in.ModelID == "" {
		return nil
	}
	return s.CheckSupplier(ctx, in.ModelID, in.Meaning, in.Stage)
}

func (s *Service) DeleteMapping(ctx context.Context, _ actor.Actor, id string) error {
	if err := authz.Require(ctx, PermissionImport); err != nil {
		return err
	}
	return s.tx.Write(ctx, func(ctx context.Context) error {
		return s.repo.DeleteMapping(ctx, id)
	})
}

// checkMapping reports every fault at once, as every form in MAPP does (FR-001).
func checkMapping(in MappingInput) error {
	problems := namingFaults(in)
	problems = append(problems, consistencyFaults(in)...)
	if len(problems) > 0 {
		return errs.Validation(problems...)
	}
	return nil
}

func namingFaults(in MappingInput) []errs.FieldError {
	var problems []errs.FieldError
	if name := strings.TrimSpace(in.ColumnName); name == "" || len(name) > maxNameLength {
		problems = append(problems, errs.FieldError{Field: fieldColumnName, Message: msgColumnLength})
	}
	if !knownMeaning(in.Meaning) {
		problems = append(problems, errs.FieldError{Field: fieldMeaning, Message: msgMeaningUnknown})
	}
	if in.WebMetric != "" && !knownMetric(in.WebMetric) {
		problems = append(problems, errs.FieldError{Field: fieldWebMetric, Message: msgMetricUnknown})
	}
	return problems
}

// consistencyFaults checks that a column names exactly what its meaning requires and nothing it does not.
func consistencyFaults(in MappingInput) []errs.FieldError {
	var problems []errs.FieldError
	if (in.Meaning == MeaningStage) != (in.Stage != "") {
		problems = append(problems, errs.FieldError{Field: fieldStage, Message: msgStageNeeded})
	}
	if (in.Meaning == MeaningWebMetric) != (in.WebMetric != "") {
		problems = append(problems, errs.FieldError{Field: fieldWebMetric, Message: msgMetricNeeded})
	}
	if suppliesModel(in.Meaning) != (in.ModelID != "") {
		problems = append(problems, errs.FieldError{Field: fieldModelID, Message: msgModelNeeded})
	}
	return problems
}

func suppliesModel(m Meaning) bool {
	return m == MeaningStage || m == MeaningRevenue
}

func knownMeaning(m Meaning) bool {
	switch m {
	case MeaningKey, MeaningReference, MeaningFactDate, MeaningStage,
		MeaningRevenue, MeaningCurrency, MeaningWebMetric, MeaningJourney:
		return true
	}
	return false
}

func knownMetric(w WebMetric) bool {
	return w == MetricSessions || w == MetricVisits || w == MetricEngagementTouches
}
