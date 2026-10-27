package access

import "mapp/internal/platform/authz"

func settingsRoutes() map[string]Rule {
	return map[string]Rule{
		// 002 Settings: readable by everyone (forms need the lists, D-240), changed with settings.manage;
		// currencies with exchangeRates.manage.
		"GET /api/v1/settings/overview":                                    signedIn(),
		"GET /api/v1/models":                                               signedIn(),
		"POST /api/v1/models":                                              need(authz.SettingsManage),
		"GET /api/v1/models/{modelId}":                                     signedIn(),
		"PATCH /api/v1/models/{modelId}":                                   need(authz.SettingsManage),
		"DELETE /api/v1/models/{modelId}":                                  need(authz.SettingsManage),
		"POST /api/v1/models/{modelId}/impact":                             need(authz.SettingsManage),
		"POST /api/v1/models/{modelId}/status":                             need(authz.SettingsManage),
		"GET /api/v1/activity-types":                                       signedIn(),
		"POST /api/v1/activity-types":                                      need(authz.SettingsManage),
		"GET /api/v1/activity-types/{activityTypeId}":                      signedIn(),
		"PATCH /api/v1/activity-types/{activityTypeId}":                    need(authz.SettingsManage),
		"DELETE /api/v1/activity-types/{activityTypeId}":                   need(authz.SettingsManage),
		"POST /api/v1/activity-types/{activityTypeId}/impact":              need(authz.SettingsManage),
		"POST /api/v1/activity-types/{activityTypeId}/status":              need(authz.SettingsManage),
		"GET /api/v1/shell-attributes":                                     signedIn(),
		"POST /api/v1/shell-attributes":                                    need(authz.SettingsManage),
		"GET /api/v1/shell-attributes/{attributeId}":                       signedIn(),
		"PATCH /api/v1/shell-attributes/{attributeId}":                     need(authz.SettingsManage),
		"DELETE /api/v1/shell-attributes/{attributeId}":                    need(authz.SettingsManage),
		"POST /api/v1/shell-attributes/{attributeId}/impact":               need(authz.SettingsManage),
		"POST /api/v1/shell-attributes/{attributeId}/status":               need(authz.SettingsManage),
		"POST /api/v1/shell-attributes/{attributeId}/move":                 need(authz.SettingsManage),
		"GET /api/v1/shell-attributes/{attributeId}/options":               signedIn(),
		"POST /api/v1/shell-attributes/{attributeId}/options":              need(authz.SettingsManage),
		"POST /api/v1/shell-attributes/{attributeId}/options/import-check": need(authz.SettingsManage),
		"POST /api/v1/shell-attributes/{attributeId}/options/import":       need(authz.SettingsManage),
		"GET /api/v1/shell-attribute-options/{optionId}":                   signedIn(),
		"PATCH /api/v1/shell-attribute-options/{optionId}":                 need(authz.SettingsManage),
		"DELETE /api/v1/shell-attribute-options/{optionId}":                need(authz.SettingsManage),
		"POST /api/v1/shell-attribute-options/{optionId}/status":           need(authz.SettingsManage),
		"POST /api/v1/shell-attribute-options/{optionId}/move":             need(authz.SettingsManage),
		"GET /api/v1/shell-attribute-forms/{level}":                        signedIn(),
		"POST /api/v1/shell-attribute-forms/{level}/check":                 signedIn(),
		"GET /api/v1/currencies":                                           signedIn(),
		"POST /api/v1/currencies":                                          need(authz.ExchangeRatesManage),
		"GET /api/v1/currencies/{code}":                                    signedIn(),
		"PATCH /api/v1/currencies/{code}":                                  need(authz.ExchangeRatesManage),
		"DELETE /api/v1/currencies/{code}":                                 need(authz.ExchangeRatesManage),
		"POST /api/v1/currencies/{code}/impact":                            need(authz.ExchangeRatesManage),
		"POST /api/v1/currencies/{code}/status":                            need(authz.ExchangeRatesManage),
		"GET /api/v1/finance-budgets":                                      signedIn(),
		"PATCH /api/v1/finance-budgets/{budgetId}":                         need(authz.SettingsManage),
		"POST /api/v1/finance-budget-items":                                need(authz.SettingsManage),
		"GET /api/v1/finance-budget-items/{itemId}":                        signedIn(),
		"PATCH /api/v1/finance-budget-items/{itemId}":                      need(authz.SettingsManage),
		"DELETE /api/v1/finance-budget-items/{itemId}":                     need(authz.SettingsManage),
		"POST /api/v1/finance-budget-items/{itemId}/status":                need(authz.SettingsManage),
	}
}

func extensionRoutes() map[string]Rule {
	return map[string]Rule{
		// 016 Planning and currency extensions (D-301)
		"GET /api/v1/currencies/{code}/rates":                   signedIn(),
		"POST /api/v1/currencies/{code}/rates":                  need(authz.ExchangeRatesManage),
		"POST /api/v1/currencies/{code}/rates/impact":           need(authz.ExchangeRatesManage),
		"PATCH /api/v1/exchange-rates/{rateId}":                 need(authz.ExchangeRatesManage),
		"DELETE /api/v1/exchange-rates/{rateId}":                need(authz.ExchangeRatesManage),
		"POST /api/v1/exchange-rates/{rateId}/impact":           need(authz.ExchangeRatesManage),
		"GET /api/v1/exchange-rates/current":                    signedIn(),
		"POST /api/v1/exchange-rates/import-check":              need(authz.ExchangeRatesManage),
		"POST /api/v1/exchange-rates/import":                    need(authz.ExchangeRatesManage),
		"GET /api/v1/exchange-rates/imports":                    signedIn(),
		"GET /api/v1/exchange-rates/import-sample":              need(authz.ExchangeRatesManage),
		"GET /api/v1/field-help":                                signedIn(),
		"PATCH /api/v1/field-help/{helpKey}":                    need(authz.SettingsManage),
		"PUT /api/v1/plans/{planId}/budget-phasing":             inPlan(authz.PlansEdit),
		"PUT /api/v1/campaigns/{campaignId}/budget-phasing":     inPlan(authz.CampaignsEdit),
		"PUT /api/v1/initiatives/{initiativeId}/budget-phasing": inPlan(authz.CampaignsEdit),
		"GET /api/v1/plans/{planId}/budget-by-month":            view(),
		"GET /api/v1/activities/{activityId}/split-options":     inPlan(authz.ActivitiesEdit),
		"POST /api/v1/activities/{activityId}/split-preview":    inPlan(authz.ActivitiesEdit),
		"POST /api/v1/activities/{activityId}/split":            inPlan(authz.ActivitiesEdit),
		"GET /api/v1/all-work/breakdown":                        signedIn(), // filtered; with plan: View of that plan
	}
}
