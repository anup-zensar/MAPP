package performance

import "mapp/internal/platform/authz"

// Declared here, not in authz's Table A: a row there breaks features 003 and 014 (contracts/api.md item 1).
const (
	PermissionImport authz.Permission = "performance.import"
	PermissionLink   authz.Permission = "performance.link"
)
