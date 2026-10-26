package settings_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/dbtest"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

var (
	pgMeta  = record.NewMeta(actor.System, time.Date(2026, time.March, 12, 9, 0, 0, 0, time.UTC))
	pgExact = cmpopts.EquateComparable(money.Amount{}, money.Percent{}, money.Rate{}, date.Date{})
)

func pgStore(t *testing.T) (*database.DB, *settings.PostgresStore) {
	t.Helper()
	db := dbtest.NewDB(t)
	return db, settings.NewPostgresStore(db)
}

func pgWrite(t *testing.T, db *database.DB, fn func(ctx context.Context) error) {
	t.Helper()
	if err := db.Write(t.Context(), fn); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
}

func pgActivityType(id, name string, cost *money.Money) settings.ActivityType {
	return settings.ActivityType{ID: id, Name: name, DefaultPlannedCost: cost, Active: true, Meta: pgMeta}
}

func pgModel(id, name string, activityTypeIDs ...string) settings.Model {
	return settings.Model{ID: id, Name: name, InqToMqlRate: money.PercentFromHundredths(200),
		MqlToDealRate: money.PercentFromHundredths(1000), AverageDealSize: money.USD(money.FromCents(500000)),
		ActivityTypeIDs: activityTypeIDs, Active: true, Meta: pgMeta}
}

func pgAttribute(id, label, parentID string, condition ...string) settings.Attribute {
	a := settings.Attribute{ID: id, Label: label, Level: settings.LevelCampaign, Type: settings.TypeSingleChoice,
		Marking: settings.MarkingOptional, Section: settings.DefaultSection, Position: 1, ParentAttributeID: parentID,
		Condition: append([]string{}, condition...), Active: true, Meta: pgMeta}
	if parentID == "" {
		a.Level, a.Role = settings.LevelPlan, settings.RoleBusinessUnit
	}
	return a
}

func pgOption(id, attributeID, label string, parentIDs ...string) settings.Option {
	return settings.Option{ID: id, AttributeID: attributeID, Label: label, Position: 1,
		ParentOptionIDs: append([]string{}, parentIDs...), Active: true, Meta: pgMeta}
}

func TestPostgresStoreModelsAndActivityTypesFR008(t *testing.T) {
	t.Parallel()
	db, s := pgStore(t)
	cost := money.USD(money.FromCents(150000))
	types := []settings.ActivityType{pgActivityType("AT-1001", "Webinar", &cost), pgActivityType("AT-1002", "Event", nil)}
	model := pgModel("MOD-1001", "Lead generation", "AT-1002", "AT-1001")
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.PutActivityType(ctx, types[0]), s.PutActivityType(ctx, types[1]), s.PutModel(ctx, model))
	})
	model.ActivityTypeIDs = []string{"AT-1001"}
	pgWrite(t, db, func(ctx context.Context) error { return s.PutModel(ctx, model) })

	gotTypes, typesErr := s.ListActivityTypes(t.Context())
	gotModel, found, modelErr := s.GetModel(t.Context(), "MOD-1001")

	if err := errors.Join(typesErr, modelErr); err != nil || !found {
		t.Fatalf("ListActivityTypes(), GetModel() = found %v, error %v", found, err)
	}
	if diff := cmp.Diff(types, gotTypes, pgExact); diff != "" {
		t.Errorf("ListActivityTypes() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(model, gotModel, pgExact); diff != "" {
		t.Errorf("GetModel() after replacing its activity types mismatch (-want +got):\n%s", diff)
	}
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.DeleteModel(ctx, "MOD-1001"), s.DeleteActivityType(ctx, "AT-1001"))
	})
	if models, err := s.ListModels(t.Context()); err != nil || len(models) != 0 {
		t.Errorf("ListModels() after the delete = %+v, error %v; want none", models, err)
	}
}

func TestPostgresStoreAttributesAndOptionsFR008(t *testing.T) {
	t.Parallel()
	db, s := pgStore(t)
	businessUnit := pgAttribute("SA-1001", "Business unit", "")
	channel := pgAttribute("SA-1002", "Channel", "SA-1001", "SO-1002", "SO-1001")
	options := []settings.Option{pgOption("SO-1001", "SA-1001", "Lubricants"), pgOption("SO-1002", "SA-1001", "Mobility"),
		pgOption("SO-1003", "SA-1002", "Social", "SO-1001")}
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.PutAttribute(ctx, channel), s.PutAttribute(ctx, businessUnit),
			s.PutOption(ctx, options[2]), s.PutOption(ctx, options[0]), s.PutOption(ctx, options[1]))
	})

	attributes, attributesErr := s.ListAttributes(t.Context())
	ofBusinessUnit, optionsErr := s.OptionsOf(t.Context(), "SA-1001")
	social, found, optionErr := s.GetOption(t.Context(), "SO-1003")

	if err := errors.Join(attributesErr, optionsErr, optionErr); err != nil || !found {
		t.Fatalf("ListAttributes(), OptionsOf(), GetOption() = found %v, error %v", found, err)
	}
	if diff := cmp.Diff([]settings.Attribute{businessUnit, channel}, attributes, pgExact); diff != "" {
		t.Errorf("ListAttributes() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(options[:2], ofBusinessUnit, pgExact); diff != "" {
		t.Errorf("OptionsOf() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(options[2], social, pgExact); diff != "" {
		t.Errorf("GetOption() mismatch (-want +got):\n%s", diff)
	}
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.DeleteOption(ctx, "SO-1003"), s.DeleteAttribute(ctx, "SA-1002"))
	})
	if left, err := s.ListOptions(t.Context()); err != nil || !cmp.Equal(options[:2], left, pgExact) {
		t.Errorf("ListOptions() after deleting the dependent attribute = %+v, error %v; want %+v", left, err, options[:2])
	}
}

func TestPostgresStoreRefusesBrokenSettingsFR004(t *testing.T) {
	t.Parallel()
	splitAtPlanLevel := pgAttribute("SA-1003", "Split", "SA-1001")
	splitAtPlanLevel.Level, splitAtPlanLevel.Type = settings.LevelPlan, settings.TypeMultipleChoiceSplit
	tests := []struct {
		name  string
		write func(ctx context.Context, s *settings.PostgresStore) error
	}{
		{name: "a model name that differs only in letter case", write: func(ctx context.Context, s *settings.PostgresStore) error {
			return s.PutModel(ctx, pgModel("MOD-1002", "LEAD GENERATION", "AT-1001"))
		}},
		{name: "an activity type deleted while a model allows it", write: func(ctx context.Context, s *settings.PostgresStore) error {
			return s.DeleteActivityType(ctx, "AT-1001")
		}},
		{name: "a second Business unit attribute", write: func(ctx context.Context, s *settings.PostgresStore) error {
			return s.PutAttribute(ctx, pgAttribute("SA-1003", "Division", ""))
		}},
		{name: "a condition on an option that does not exist", write: func(ctx context.Context, s *settings.PostgresStore) error {
			return s.PutAttribute(ctx, pgAttribute("SA-1003", "Channel", "SA-1001", "SO-9999"))
		}},
		{name: "a % split attribute at plan level", write: func(ctx context.Context, s *settings.PostgresStore) error {
			return s.PutAttribute(ctx, splitAtPlanLevel)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db, s := pgStore(t)
			pgWrite(t, db, func(ctx context.Context) error {
				return errors.Join(s.PutActivityType(ctx, pgActivityType("AT-1001", "Webinar", nil)),
					s.PutModel(ctx, pgModel("MOD-1001", "Lead generation", "AT-1001")),
					s.PutAttribute(ctx, pgAttribute("SA-1001", "Business unit", "")),
					s.PutOption(ctx, pgOption("SO-1001", "SA-1001", "Lubricants")))
			})

			err := db.Write(t.Context(), func(ctx context.Context) error { return tt.write(ctx, s) })

			if err == nil {
				t.Errorf("Write() error = nil, want the database to refuse the change")
			}
		})
	}
}
