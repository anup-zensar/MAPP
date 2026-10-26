package apitest

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
	"github.com/pb33f/libopenapi-validator/cache"
	"github.com/pb33f/libopenapi-validator/config"
	openapierrors "github.com/pb33f/libopenapi-validator/errors"

	"mapp/api"
)

func buildContract() (validator.Validator, error) {
	document, err := libopenapi.NewDocument(api.OpenAPI)
	if err != nil {
		return nil, fmt.Errorf("reading the contract: %w", err)
	}
	schemas := &lazySchemas{SchemaCache: cache.NewDefaultCache()}
	built, problems := validator.NewValidator(document, config.WithFormatAssertions(),
		config.WithoutSecurityValidation(), config.WithSchemaCache(schemas))
	if len(problems) > 0 {
		return nil, fmt.Errorf("building the contract validator: %w", errors.Join(problems...))
	}
	schemas.built.Store(true)
	return &serialValidator{Validator: built}, nil
}

// serialValidator checks one request or answer at a time: compiling a schema renders the shared document model.
type serialValidator struct {
	validator.Validator
	mu sync.Mutex
}

//nolint:revive // var-naming: the name of the libopenapi-validator method it replaces
func (v *serialValidator) ValidateHttpRequest(r *http.Request) (bool, []*openapierrors.ValidationError) {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.ValidateHttpRequestSync(r)
}

//nolint:revive // var-naming: the name of the libopenapi-validator method it replaces
func (v *serialValidator) ValidateHttpResponse(r *http.Request, answer *http.Response) (bool, []*openapierrors.ValidationError) {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.Validator.ValidateHttpResponse(r, answer)
}

// lazySchemas makes the validator compile a schema when a request first needs it, not all up front (15 to 30 s
// under the race detector, in every test binary): it reports every schema as held until the validator is built.
type lazySchemas struct {
	cache.SchemaCache
	built atomic.Bool
}

func (s *lazySchemas) Load(key uint64) (*cache.SchemaCacheEntry, bool) {
	if !s.built.Load() {
		return nil, true
	}
	return s.SchemaCache.Load(key)
}
