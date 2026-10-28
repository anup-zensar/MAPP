package access

import "mapp/internal/platform/authz"

func overspendRoutes() map[string]Rule {
	return map[string]Rule{
		"GET /api/v1/overspend-configuration": signedIn(),
		"PUT /api/v1/overspend-configuration": need(authz.SettingsManage),
	}
}
