package performance

import (
	"context"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
)

type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}

type SourceView struct {
	Source
	LastLoadedAt time.Time
	DataThrough  time.Time
	FactCount    int
}

type Supplier struct {
	MappingID  string
	SourceID   string
	SourceName string
	Meaning    Meaning
	Stage      string
}

type FactView struct {
	Fact
	SourceName string
}

type ReviewRowView struct {
	ReviewRow
	SourceName string
}

// Sources and mappings are organisation-wide configuration, so they carry no scope.
type Sources interface {
	ListSources(ctx context.Context, id string) ([]SourceView, error)
	PutSource(ctx context.Context, s Source) error
	DeleteSource(ctx context.Context, id string) (bool, error)
	ListMappings(ctx context.Context, sourceIDs []string) ([]Mapping, error)
	PutMapping(ctx context.Context, m Mapping) error
	DeleteMapping(ctx context.Context, id string) error
	ActiveSuppliers(ctx context.Context, modelID string) ([]Supplier, error)
}

// Matching resolves a whole batch's distinct keys in one query per key type, never one per row (research
// R2, plan R16). Each answer maps a key to every activity it reached: more than one means ambiguous, and
// the matcher refuses to guess (FR-019).
type Matching interface {
	MatchByActivityID(ctx context.Context, scope authz.Scope, keys []string) (map[string][]string, error)
	MatchByMediaCode(ctx context.Context, scope authz.Scope, codes []int64) (map[string][]string, error)
	MatchByIntegrationID(ctx context.Context, scope authz.Scope, sourceID string, keys []string) (map[string][]string, error)
	MatchByExternalName(ctx context.Context, scope authz.Scope, keys []string) (map[string][]string, error)
}

type Facts interface {
	ListFactsForActivity(ctx context.Context, scope authz.Scope, activityID string, countedOnly bool) ([]FactView, error)
	PutFact(ctx context.Context, f Fact) error
	CurrentFactByReference(ctx context.Context, sourceID, reference string) (Fact, bool, error)
	ReleaseHeld(ctx context.Context, activityID string, by actor.Actor, at time.Time) (int, error)
}

type Imports interface {
	ListImports(ctx context.Context, id, sourceID string, limit int) ([]Import, error)
	PutImport(ctx context.Context, i Import) error
}

type Review interface {
	ListReviewRows(ctx context.Context, status, sourceID string, limit int) ([]ReviewRowView, error)
	PutReviewRow(ctx context.Context, r ReviewRow) error
}

type Codes interface {
	ListMediaCodes(ctx context.Context, scope authz.Scope, activityID string) ([]MediaCode, error)
	TakeNextCode(ctx context.Context, by actor.Actor, at time.Time) (int64, error)
}

type Store interface {
	Sources
	Matching
	Facts
	Imports
	Review
	Codes
}
