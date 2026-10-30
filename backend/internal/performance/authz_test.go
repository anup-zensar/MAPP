package performance_test

import (
	"context"
	"errors"
	"testing"

	"mapp/internal/performance"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
)

// grantsWithout gives Alex a role holding every permission named, and nothing else, so an operation this
// feature guards can be shown to fail closed.
func grantsWithout(held ...authz.Permission) authz.Grants {
	role := authz.GrantedRole{ID: "RL-1", Name: "Planner", Scope: authz.ScopeAll}
	return authz.NewGrants(alex.ID, []authz.GrantedRole{role}, nil,
		map[string][]authz.Permission{"RL-1": held}, nil, authz.Rules{})
}

func TestOperations_FailClosedWithoutTheirPermission_FR043(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		held []authz.Permission
		call func(ctx context.Context, f *fixture) error
	}{
		{
			name: "FR-043 creating a data source needs performance.import",
			held: []authz.Permission{authz.PlansView},
			call: func(ctx context.Context, f *fixture) error {
				_, err := f.service.CreateSource(ctx, alex, performance.SourceInput{
					Name: "Eloqua", OwnerID: alex.ID, Category: performance.CategoryCRM,
					DeliveryRoute: performance.RouteUpload, Cadence: performance.CadenceMonthly,
				})
				return err
			},
		},
		{
			name: "FR-043 deleting a data source needs performance.import",
			held: []authz.Permission{authz.PlansView},
			call: func(ctx context.Context, f *fixture) error {
				return f.service.DeleteSource(ctx, alex, "PS-1")
			},
		},
		{
			name: "FR-043 deactivating a data source needs performance.import",
			held: []authz.Permission{authz.PlansView},
			call: func(ctx context.Context, f *fixture) error {
				return f.service.SetSourceStatus(ctx, alex, "PS-1", performance.StatusInactive)
			},
		},
		{
			name: "FR-044 listing data sources needs plans.view",
			held: []authz.Permission{performance.PermissionImport},
			call: func(ctx context.Context, f *fixture) error {
				_, err := f.service.Sources(ctx, alex)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			ctx := authz.With(t.Context(), grantsWithout(tt.held...))

			err := tt.call(ctx, f)

			var refusal *errs.Error
			if !errors.As(err, &refusal) || refusal.Code != errs.CodeForbidden {
				t.Fatalf("the operation answered %v, want %s: it must fail closed", err, errs.CodeForbidden)
			}
		})
	}
}
