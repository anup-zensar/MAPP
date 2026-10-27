package activity

import (
	"context"
	"slices"

	"mapp/internal/platform/authz"
)

func (r JSONStore) WorkflowIDs(ctx context.Context) (map[string]string, error) {
	all, err := r.List(ctx)
	fields := make(map[string]string, len(all))
	for _, a := range all {
		fields[a.ID] = a.ApprovalWorkflowID
	}
	return fields, err
}

func (r JSONStore) ListInScopeOf(ctx context.Context, scope authz.Scope, ids []string) ([]Activity, error) {
	seen, err := r.ListInScope(ctx, scope)
	return among(seen, ids), err
}

func (r JSONStore) ListByIDs(ctx context.Context, ids []string) ([]Activity, error) {
	all, err := r.List(ctx)
	return among(all, ids), err
}

func among(list []Activity, ids []string) []Activity {
	kept := []Activity{}
	for _, a := range list {
		if slices.Contains(ids, a.ID) {
			kept = append(kept, a)
		}
	}
	return kept
}
