package access

import "mapp/internal/performance"

// Feature 046. The permissions are declared in the performance package rather than authz's Table A, which
// features 003 and 014 assert the exact shape of.
func performanceRoutes() map[string]Rule {
	return map[string]Rule{
		"GET /api/v1/performance/sources":                        signedIn(),
		"POST /api/v1/performance/sources":                       need(performance.PermissionImport),
		"GET /api/v1/performance/sources/{sourceId}":             signedIn(),
		"PATCH /api/v1/performance/sources/{sourceId}":           need(performance.PermissionImport),
		"DELETE /api/v1/performance/sources/{sourceId}":          need(performance.PermissionImport),
		"POST /api/v1/performance/sources/{sourceId}/deactivate": need(performance.PermissionImport),
		"POST /api/v1/performance/sources/{sourceId}/reactivate": need(performance.PermissionImport),
		"GET /api/v1/performance/sources/{sourceId}/mappings":    signedIn(),
		"POST /api/v1/performance/sources/{sourceId}/mappings":   need(performance.PermissionImport),
		"GET /api/v1/performance/sources/{sourceId}/template":    signedIn(),
		"PATCH /api/v1/performance/mappings/{mappingId}":         need(performance.PermissionImport),
		"DELETE /api/v1/performance/mappings/{mappingId}":        need(performance.PermissionImport),
	}
}
