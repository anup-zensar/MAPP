package performance

import "mapp/internal/platform/authz"

// PermissionImport and PermissionLink are reserved by D-367. They are declared here rather than added to
// authz's Table A, which is a shared contract: a row there changes every role's permission count and the
// group sizes that features 003 and 014 assert. The integrator registers them (contracts/api.md item 1),
// and until then no role holds either, so every guarded operation fails closed.
const (
	PermissionImport authz.Permission = "performance.import"
	PermissionLink   authz.Permission = "performance.link"
)
